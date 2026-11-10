// Package write 负责出站写意图的在途登记、一次性发送、结果核对与状态观察。
//
// 范围内：登记带幂等键的写意图；同一意图只发一次请求；明确成功或失败后终结；
// 超时与断线转入未知；未知按注入的最小读取接口核对，命中且归属一致才算成功；
// 核对确认未发生时允许重提；核对次数用尽进入「需人工确认」终态；失败与未知时
// 保留待发内容；成功时清空内容与匹配的草稿；提供在途表、单条结果与状态变化通知。
//
// 不包含：界面、连接实现、协议 wire 定义、投影内部、写入内容的业务校验、任何
// 持久化。幂等键只在内存中存在；进程重启后本包不补发任何旧意图。
package write

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
)

// Kind 是写意图的类型。实现方按类型决定用哪个连接方法发送；本包不解释参数。
type Kind string

const (
	// KindSubmitInput 是向会话提交一次输入。
	KindSubmitInput Kind = "submit_input"
	// KindStop 是停止会话或分支上的运行。
	KindStop Kind = "stop"
	// KindApprovalResponse 是对一次审批的同意或拒绝应答。
	KindApprovalResponse Kind = "approval_response"
	// KindLongTermAllow 是把「长期允许」写成服务端事实。
	KindLongTermAllow Kind = "long_term_allow"
	// KindQuestionAnswer 是对一次提问的应答。
	KindQuestionAnswer Kind = "question_answer"
	// KindManagementWrite 是管理面的设置写入。
	KindManagementWrite Kind = "management_write"
	// KindAccountAction 是账号动作（登录、登出、重置等）。
	KindAccountAction Kind = "account_action"
	// KindAttachmentWrite 是附件资源写入。
	KindAttachmentWrite Kind = "attachment_resource_write"
	// KindSessionLifecycle 是会话与分支的新建、改名、删除等生命周期动作。
	// 具体动作由参数里的 op 区分（create / rename / delete / branch.create /
	// branch.end / fork，兼容目录领域现有的 catalog.create / catalog.rename /
	// catalog.delete）；新建会话时还没有会话标识，因此这种类型不强制要求
	// SessionID。
	KindSessionLifecycle Kind = "session_lifecycle"
	// KindShellRun 是会话内命令执行与停止（`!` 命令由服务端执行）。
	// 参数用 op 区分 run / stop：run 带命令，stop 带运行标识。
	KindShellRun Kind = "shell_run"
	// KindModalityWrite 是模态任务提交与取消、实时传输的打开与关闭。
	// 参数用 op 区分 task.submit / task.cancel / realtime.open / realtime.close。
	KindModalityWrite Kind = "modality_write"
)

func (k Kind) valid() bool {
	switch k {
	case KindSubmitInput, KindStop, KindApprovalResponse, KindLongTermAllow,
		KindQuestionAnswer, KindManagementWrite, KindAccountAction, KindAttachmentWrite,
		KindSessionLifecycle, KindShellRun, KindModalityWrite:
		return true
	}
	return false
}

// sessionScoped 表示该类型必须指定目标会话。会话生命周期不在这里判断：
// 新建时尚无会话标识，改名与删除由 normalizeIntent 按 op 单独要求。
func (k Kind) sessionScoped() bool {
	switch k {
	case KindSubmitInput, KindStop, KindApprovalResponse, KindLongTermAllow,
		KindQuestionAnswer, KindAttachmentWrite, KindShellRun, KindModalityWrite:
		return true
	}
	return false
}

// Delivery 是提交输入的投递方式。运行中显式选插话，否则排队。
type Delivery string

const (
	// DeliveryQueue 排队到下一轮；这是默认值（R1 拍板）。
	DeliveryQueue Delivery = "queue"
	// DeliveryInterject 插话进当前运行。
	DeliveryInterject Delivery = "interject"
)

// Status 是写意图的对外状态。它与 A.md「在途写」状态机的状态一一对应。
type Status string

const (
	// StatusRegistered 已登记，尚未发出。
	StatusRegistered Status = "registered"
	// StatusInFlight 已发出，等待明确回执。
	StatusInFlight Status = "in_flight"
	// StatusUnknown 结果未知，正在核对。
	StatusUnknown Status = "unknown"
	// StatusSucceeded 明确成功（含核对命中且归属一致）。
	StatusSucceeded Status = "succeeded"
	// StatusFailed 明确失败。待发内容与草稿保留。
	StatusFailed Status = "failed"
	// StatusNotOccurred 核对确认服务端没有这次写入。允许用新的幂等键重提。
	StatusNotOccurred Status = "not_occurred"
	// StatusNeedsManualConfirmation 核对无法确认且次数用尽；终态，不自动重发。
	StatusNeedsManualConfirmation Status = "needs_manual_confirmation"
)

// terminal 表示该状态不再变化。
func (s Status) terminal() bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusNotOccurred, StatusNeedsManualConfirmation:
		return true
	}
	return false
}

// unsettled 表示该状态仍占着在途表。
func (s Status) unsettled() bool {
	return s == StatusRegistered || s == StatusInFlight || s == StatusUnknown
}

// Intent 是调用方登记的一次写意图。除 Key 与 Kind 外，其余字段按类型使用。
type Intent struct {
	// Key 是幂等键，必填。同一 Key 重复登记返回同一意图，不重复发送。
	// 它只在内存中存在，不落盘。
	Key string
	// Kind 是意图类型，必填。
	Kind Kind
	// SessionID 是目标会话；提交、停止、审批、提问与附件写入必填。会话生命周期
	// 按 op 决定：改名与删除必填，新建可以为空（服务端在回执里给新会话标识）。
	SessionID string
	// BranchID 是目标分支；空表示由发送方按主线处理。
	BranchID string
	// Delivery 是投递方式；空按排队处理（R1）。
	Delivery Delivery
	// InputID 是核对用的输入标识。提交类用于服务端去重；为空时由本包生成并随
	// Request 下发。其他类型可填服务端请求标识（审批、提问、停止目标等）；为空
	// 时核对只能依赖发送方或检查器自行解释，通常最终进入需人工确认。会话生命
	// 周期的新建没有会话标识，建议在这里填一个客户端生成的新建标识，供核对使用。
	InputID string
	// DraftRevision 是登记时草稿的修订号。成功时只有草稿修订号仍相同才清空，
	// 避免清掉用户后来输入的新草稿；0 表示登记时自动取当前草稿修订号。
	DraftRevision uint64
	// Params 是不透明的请求参数。本包只校验它是合法 JSON，不解释内容。
	Params json.RawMessage
}

// Request 是交给发送方的已登记请求。发送方负责把 Kind 映射为连接领域的具体
// wire 方法，并把会话、分支、投递方式与输入标识合入参数。
type Request struct {
	IntentID  string
	Kind      Kind
	SessionID string
	BranchID  string
	Delivery  Delivery
	InputID   string
	Params    json.RawMessage
}

// Reply 是一次明确回执。
type Reply struct {
	// InputID 是服务端给出的输入标识；为空时沿用 Request.InputID。
	InputID string
	// Result 是回执原文，供调用方使用。
	Result json.RawMessage
}

// Sender 发送一次写请求。实现方只发一次，不得自行重试；返回明确回执或错误。
//
// 错误分类沿用连接领域：超时与连接中断（可能在发出之后发生）会被本包按
// 「结果未知」处理；能力缺失、参数错误与连接缺失（请求未发出）按明确失败处理。
type Sender interface {
	Send(ctx context.Context, req Request) (Reply, error)
}

// InputQuery 是一次核对查询：按输入标识查归属。
type InputQuery struct {
	IntentID  string
	Kind      Kind
	InputID   string
	SessionID string
	BranchID  string
}

// InputState 是一次核对读取的结果。
type InputState struct {
	// Confirmed 表示这次读取可信；false 表示核对无法确认（读取失败、能力缺失
	// 或读取到的不是确定状态）。
	Confirmed bool
	// Present 表示服务端历史已含该输入。
	Present bool
	// SessionID 与 BranchID 是该输入的实际归属；Present 为真时有意义。
	SessionID string
	BranchID  string
	// Running 是目标会话/分支当前的运行状态，作为核对上下文。
	Running bool
}

// Inspector 是核对用的最小读取接口。它只读，不写、不重发。
type Inspector interface {
	LookupInput(ctx context.Context, query InputQuery) (InputState, error)
}

// VerifyPolicy 是未知结果的核对策略。
type VerifyPolicy struct {
	// MaxAttempts 是最多核对次数（至少 1）；用尽进入需人工确认。
	MaxAttempts int
	// Interval 是两次核对之间的固定间隔。
	Interval time.Duration
	// Timeout 是单次核对读取的超时。
	Timeout time.Duration
}

// Config 是写登记配置。Sender 与 Inspector 必填。
type Config struct {
	Sender    Sender
	Inspector Inspector
	// SendTimeout 是单次发送的上限；超时进入未知。
	SendTimeout time.Duration
	Verify      VerifyPolicy
	// Now 与 NewID 可在测试中替换；nil 时使用系统实现。
	Now   func() time.Time
	NewID func() (string, error)
}

func (c Config) withDefaults() Config {
	if c.SendTimeout <= 0 {
		c.SendTimeout = 30 * time.Second
	}
	if c.Verify.MaxAttempts <= 0 {
		c.Verify.MaxAttempts = 3
	}
	if c.Verify.Interval <= 0 {
		c.Verify.Interval = time.Second
	}
	if c.Verify.Timeout <= 0 {
		c.Verify.Timeout = 5 * time.Second
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.NewID == nil {
		c.NewID = randomID
	}
	return c
}

// IntentView 是一条意图的只读视图；不含内部状态机字段。
type IntentView struct {
	ID        string
	Key       string
	Kind      Kind
	SessionID string
	BranchID  string
	Delivery  Delivery
	InputID   string
	Status    Status
	Reason    string
	// Content 是待发内容；成功前保留，成功后清空。
	Content json.RawMessage
	// Result 是成功回执原文；核对命中成功时没有回执，保持为空。
	Result    json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Change 是一次状态变化通知。
type Change struct {
	IntentID string
	Key      string
	Kind     Kind
	Previous Status
	Status   Status
	Reason   string
}

// Draft 是一个会话上的本地草稿（输入文本与待发附件引用）。
type Draft struct {
	SessionID string
	Text      string
	// AttachmentIDs 是待发附件的暂存标识；本包不解释、不释放它们。
	AttachmentIDs []string
	// Revision 由 SetDraft 递增；用于提交成功时判断草稿是否被改过。
	Revision uint64
}

// 错误值；调用方按 errors.Is 判断，不匹配错误文本。
var (
	// ErrClosed 表示写登记已关闭，不再接受登记或草稿更新。
	ErrClosed = errors.New("写登记已关闭")
	// ErrUnknownIntent 表示没有这条意图。
	ErrUnknownIntent = errors.New("未知写意图")
	// ErrInvalidIntent 表示意图字段不合法。
	ErrInvalidIntent = errors.New("写意图不合法")
	// ErrKeyConflict 表示同一幂等键被用于不同的意图。
	ErrKeyConflict = errors.New("幂等键已被其他意图占用")
)

// Manager 持有进程内的写意图表与草稿表。所有导出方法可并发调用。
type Manager struct {
	cfg    Config
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// registerMu 串行化登记与关闭，保证「登记通知先于关闭通知」。
	registerMu sync.Mutex

	mu        sync.Mutex
	closed    bool
	byID      map[string]*intentState
	byKey     map[string]string
	order     []string
	drafts    map[string]Draft
	draftSeq  map[string]uint64
	listeners map[int]func(Change)
	nextLsn   int

	closedCh  chan struct{}
	closeOnce sync.Once
}

type intentState struct {
	view IntentView
	req  Request
	// draftRevision 是登记时草稿的修订号；成功时据此判断是否清空草稿。
	draftRevision uint64
	done          chan struct{}
}

// New 创建写登记。启动时校验状态迁移表；表不合法即失败。
func New(cfg Config) (*Manager, error) {
	if cfg.Sender == nil {
		return nil, errors.New("写登记缺少发送方")
	}
	if cfg.Inspector == nil {
		return nil, errors.New("写登记缺少核对读取接口")
	}
	if err := validateTransitionTable(); err != nil {
		return nil, err
	}
	cfg = cfg.withDefaults()
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		cfg:       cfg,
		ctx:       ctx,
		cancel:    cancel,
		byID:      make(map[string]*intentState),
		byKey:     make(map[string]string),
		drafts:    make(map[string]Draft),
		draftSeq:  make(map[string]uint64),
		listeners: make(map[int]func(Change)),
		closedCh:  make(chan struct{}),
	}, nil
}

// Register 登记一条写意图；登记完成后异步发送一次。
//
// 同一 Key 的重复登记返回同一意图，不重复发送。登记先于发送：调用返回时意图
// 已在表中，发送在之后的协程里进行。
func (m *Manager) Register(intent Intent) (IntentView, error) {
	normalized, err := normalizeIntent(intent)
	if err != nil {
		return IntentView{}, err
	}

	// 关闭与登记互斥：登记后的通知一定先于关闭通知发出。
	m.registerMu.Lock()
	defer m.registerMu.Unlock()

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return IntentView{}, ErrClosed
	}
	// 提交成功后只清空登记时的那一版草稿；未显式给出修订号时取当前值。
	if normalized.Kind == KindSubmitInput && normalized.DraftRevision == 0 {
		if draft, ok := m.drafts[normalized.SessionID]; ok {
			normalized.DraftRevision = draft.Revision
		}
	}
	if id, ok := m.byKey[normalized.Key]; ok {
		existing := m.byID[id]
		if !sameIntent(existing.req, normalized) {
			m.mu.Unlock()
			return IntentView{}, ErrKeyConflict
		}
		view := existing.snapshot()
		m.mu.Unlock()
		return view, nil
	}
	if normalized.Kind == KindSubmitInput && normalized.InputID == "" {
		id, err := m.cfg.NewID()
		if err != nil {
			m.mu.Unlock()
			return IntentView{}, fmt.Errorf("生成输入标识: %w", err)
		}
		normalized.InputID = id
	}
	id, err := m.cfg.NewID()
	if err != nil {
		m.mu.Unlock()
		return IntentView{}, fmt.Errorf("生成写意图标识: %w", err)
	}
	if _, duplicate := m.byID[id]; duplicate {
		m.mu.Unlock()
		return IntentView{}, errors.New("生成的写意图标识重复")
	}

	now := m.cfg.Now()
	state := &intentState{
		view: IntentView{
			ID:        id,
			Key:       normalized.Key,
			Kind:      normalized.Kind,
			SessionID: normalized.SessionID,
			BranchID:  normalized.BranchID,
			Delivery:  normalized.Delivery,
			InputID:   normalized.InputID,
			Status:    StatusRegistered,
			Content:   cloneBytes(normalized.Params),
			CreatedAt: now,
			UpdatedAt: now,
		},
		req: Request{
			IntentID:  id,
			Kind:      normalized.Kind,
			SessionID: normalized.SessionID,
			BranchID:  normalized.BranchID,
			Delivery:  normalized.Delivery,
			InputID:   normalized.InputID,
			Params:    cloneBytes(normalized.Params),
		},
		draftRevision: normalized.DraftRevision,
		done:          make(chan struct{}),
	}
	m.byID[id] = state
	m.byKey[normalized.Key] = id
	m.order = append(m.order, id)
	view := state.snapshot()
	// 先占好协程名额，保证 Close 一定能等到这次发送结束。
	m.wg.Add(1)
	m.mu.Unlock()

	m.notify(Change{
		IntentID: id,
		Key:      normalized.Key,
		Kind:     normalized.Kind,
		Status:   StatusRegistered,
	})
	go m.dispatch(id)
	return view, nil
}

// Result 返回一条意图的当前视图。
func (m *Manager) Result(intentID string) (IntentView, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.byID[intentID]
	if !ok {
		return IntentView{}, false
	}
	return state.snapshot(), true
}

// InFlight 返回仍在途（已登记、在途、未知）的意图视图，按登记顺序。
func (m *Manager) InFlight() []IntentView {
	m.mu.Lock()
	defer m.mu.Unlock()
	views := make([]IntentView, 0, len(m.order))
	for _, id := range m.order {
		state := m.byID[id]
		if state.view.Status.unsettled() {
			views = append(views, state.snapshot())
		}
	}
	return views
}

// Table 返回本进程登记过的全部意图视图，按登记顺序。终态也保留，供人工核对。
func (m *Manager) Table() []IntentView {
	m.mu.Lock()
	defer m.mu.Unlock()
	views := make([]IntentView, 0, len(m.order))
	for _, id := range m.order {
		views = append(views, m.byID[id].snapshot())
	}
	return views
}

// Wait 等待一条意图进入终态。ctx 结束或写登记关闭时返回当前视图与原因。
func (m *Manager) Wait(ctx context.Context, intentID string) (IntentView, error) {
	m.mu.Lock()
	state, ok := m.byID[intentID]
	if !ok {
		m.mu.Unlock()
		return IntentView{}, ErrUnknownIntent
	}
	view := state.snapshot()
	done := state.done
	closed := m.closedCh
	m.mu.Unlock()
	if view.Status.terminal() {
		return view, nil
	}

	select {
	case <-done:
		if current, ok := m.Result(intentID); ok {
			return current, nil
		}
		return IntentView{}, ErrUnknownIntent
	case <-closed:
		if current, ok := m.Result(intentID); ok {
			if current.Status.terminal() {
				return current, nil
			}
			return current, ErrClosed
		}
		return IntentView{}, ErrClosed
	case <-ctx.Done():
		if current, ok := m.Result(intentID); ok {
			return current, ctx.Err()
		}
		return IntentView{}, ctx.Err()
	}
}

// Subscribe 订阅状态变化；返回取消函数。回调在发送与核对协程里同步执行，
// 不要阻塞，也不要回调本包做重活。
func (m *Manager) Subscribe(fn func(Change)) (cancel func()) {
	if fn == nil {
		return func() {}
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return func() {}
	}
	id := m.nextLsn
	m.nextLsn++
	m.listeners[id] = fn
	m.mu.Unlock()
	return func() {
		m.mu.Lock()
		delete(m.listeners, id)
		m.mu.Unlock()
	}
}

// SetDraft 保存一个会话的草稿并递增修订号。文本与附件都为空时等价于清空。
func (m *Manager) SetDraft(draft Draft) error {
	if draft.SessionID == "" {
		return fmt.Errorf("%w: 草稿缺少会话标识", ErrInvalidIntent)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosed
	}
	// 修订号按会话单调递增，不复用：清空草稿后重新输入也会得到新修订号，
	// 这样早先那次提交的成功不会误清后来的草稿。
	m.draftSeq[draft.SessionID]++
	if draft.Text == "" && len(draft.AttachmentIDs) == 0 {
		delete(m.drafts, draft.SessionID)
		return nil
	}
	draft.Revision = m.draftSeq[draft.SessionID]
	draft.AttachmentIDs = append([]string(nil), draft.AttachmentIDs...)
	m.drafts[draft.SessionID] = draft
	return nil
}

// Draft 返回一个会话的草稿副本。
func (m *Manager) Draft(sessionID string) (Draft, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	draft, ok := m.drafts[sessionID]
	if !ok {
		return Draft{}, false
	}
	draft.AttachmentIDs = append([]string(nil), draft.AttachmentIDs...)
	return draft, true
}

// ClearDraft 清空一个会话的草稿。
func (m *Manager) ClearDraft(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.drafts, sessionID)
}

// Close 停止发送与核对。已发出的未结项意图按「需人工确认」留在表里；还没发出
// 的登记按「未发生」处理。两种情况都不补发。
func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		m.registerMu.Lock()
		m.mu.Lock()
		m.closed = true
		m.mu.Unlock()
		m.registerMu.Unlock()

		m.cancel()
		m.wg.Wait()

		m.mu.Lock()
		now := m.cfg.Now()
		changes := make([]Change, 0, len(m.order))
		for _, id := range m.order {
			state := m.byID[id]
			if state.view.Status.terminal() {
				continue
			}
			next, ok := transition(state.view.Status, eventClose)
			if !ok {
				continue
			}
			previous := state.view.Status
			state.view.Status = next
			state.view.Reason = "写登记关闭时结果未结项，需人工确认"
			if previous == StatusRegistered {
				state.view.Reason = "写登记关闭时请求尚未发出，视为未发生"
			}
			state.view.UpdatedAt = now
			close(state.done)
			changes = append(changes, Change{
				IntentID: state.view.ID,
				Key:      state.view.Key,
				Kind:     state.view.Kind,
				Previous: previous,
				Status:   next,
				Reason:   state.view.Reason,
			})
		}
		m.mu.Unlock()

		for _, change := range changes {
			m.notify(change)
		}

		m.mu.Lock()
		m.listeners = nil
		m.mu.Unlock()
		close(m.closedCh)
	})
}

// dispatch 发送一次并收敛结果。它是每条意图唯一的发送入口。
func (m *Manager) dispatch(intentID string) {
	defer m.wg.Done()

	req, ok := m.beginSend(intentID)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(m.ctx, m.cfg.SendTimeout)
	defer cancel()

	reply, err := m.cfg.Sender.Send(ctx, req)
	if err != nil {
		if m.stopped() {
			return
		}
		switch failure.KindOf(err) {
		case failure.KindTimeout, failure.KindRetryable:
			m.markUnknown(intentID, "发送结果未知: "+err.Error())
			m.verify(intentID)
		default:
			// 能力缺失、参数错误与连接缺失（请求未发出）都是明确失败。
			m.transition(intentID, eventFail, "发送明确失败: "+err.Error())
		}
		return
	}
	m.markSucceeded(intentID, reply.InputID, reply.Result, "")
}

// beginSend 把意图从已登记推到在途，并返回发送快照。
func (m *Manager) beginSend(intentID string) (Request, bool) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return Request{}, false
	}
	state, ok := m.byID[intentID]
	if !ok || state.view.Status != StatusRegistered {
		m.mu.Unlock()
		return Request{}, false
	}
	next, ok := transition(state.view.Status, eventSend)
	if !ok {
		m.mu.Unlock()
		return Request{}, false
	}
	previous := state.view.Status
	state.view.Status = next
	state.view.UpdatedAt = m.cfg.Now()
	req := state.req
	req.Params = cloneBytes(state.req.Params)
	change := Change{
		IntentID: intentID,
		Key:      state.view.Key,
		Kind:     state.view.Kind,
		Previous: previous,
		Status:   next,
	}
	m.mu.Unlock()
	m.notify(change)
	return req, true
}

// verify 在未知状态下按策略核对，最多 MaxAttempts 次。
func (m *Manager) verify(intentID string) {
	for attempt := 1; attempt <= m.cfg.Verify.MaxAttempts; attempt++ {
		if !m.waitInterval(m.cfg.Verify.Interval) {
			return
		}
		query, ok := m.queryFor(intentID)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(m.ctx, m.cfg.Verify.Timeout)
		state, err := m.cfg.Inspector.LookupInput(ctx, query)
		cancel()
		if m.stopped() {
			return
		}

		switch judge(query, state, err) {
		case verifyHit:
			m.markSucceeded(intentID, query.InputID, nil, "")
			return
		case verifyMiss:
			m.transition(intentID, eventVerifyMiss, "核对确认服务端没有这次写入")
			return
		default:
			if attempt >= m.cfg.Verify.MaxAttempts {
				m.transition(intentID, eventVerifyExhausted, "核对无法确认，需人工确认")
				return
			}
			// A.md 的「未知 → 未知」：状态不变，继续按上限核对。
			m.transition(intentID, eventVerifyUnconfirmed, "")
		}
	}
}

// queryFor 取核对所需的只读输入；意图已不在未知状态时返回 false。
func (m *Manager) queryFor(intentID string) (InputQuery, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.byID[intentID]
	if !ok || state.view.Status != StatusUnknown {
		return InputQuery{}, false
	}
	return InputQuery{
		IntentID:  intentID,
		Kind:      state.view.Kind,
		InputID:   state.view.InputID,
		SessionID: state.view.SessionID,
		BranchID:  state.view.BranchID,
	}, true
}

// markSucceeded 把意图推到成功；成功时清空待发内容与匹配的草稿。
func (m *Manager) markSucceeded(intentID, replyInputID string, result json.RawMessage, reason string) {
	m.mu.Lock()
	state, ok := m.byID[intentID]
	if !ok || state.view.Status.terminal() {
		m.mu.Unlock()
		return
	}
	// 在途用「明确成功」，未知用「核对命中」；两条迁移都到成功。
	event := eventSucceed
	if state.view.Status == StatusUnknown {
		event = eventVerifyHit
	}
	next, ok := transition(state.view.Status, event)
	if !ok {
		m.mu.Unlock()
		return
	}
	previous := state.view.Status
	if replyInputID != "" {
		state.view.InputID = replyInputID
	}
	state.view.Status = next
	state.view.Reason = reason
	state.view.Content = nil
	state.view.Result = cloneBytes(result)
	state.view.UpdatedAt = m.cfg.Now()
	close(state.done)

	// 只清空登记时的那一版草稿：用户后来输入的新草稿不能被旧提交抹掉。
	if state.view.Kind == KindSubmitInput {
		if draft, ok := m.drafts[state.view.SessionID]; ok && draft.Revision == state.draftRevision {
			delete(m.drafts, state.view.SessionID)
		}
	}
	change := Change{
		IntentID: intentID,
		Key:      state.view.Key,
		Kind:     state.view.Kind,
		Previous: previous,
		Status:   next,
		Reason:   reason,
	}
	m.mu.Unlock()
	m.notify(change)
}

// markUnknown 把在途意图推到未知。
func (m *Manager) markUnknown(intentID, reason string) {
	m.transition(intentID, eventUnknown, reason)
}

// transition 是除成功外的唯一状态迁移入口。
func (m *Manager) transition(intentID string, event transitionEvent, reason string) {
	m.mu.Lock()
	state, ok := m.byID[intentID]
	if !ok || state.view.Status.terminal() {
		m.mu.Unlock()
		return
	}
	next, ok := transition(state.view.Status, event)
	if !ok {
		// 非法迁移按 A.md 处理：丢弃事件、状态不变、不向用户报错。
		m.mu.Unlock()
		return
	}
	previous := state.view.Status
	if previous == next {
		// 「未知 → 未知」是自环：状态与原因都不变，不发通知。
		m.mu.Unlock()
		return
	}
	state.view.Status = next
	state.view.Reason = reason
	state.view.UpdatedAt = m.cfg.Now()
	if next.terminal() {
		close(state.done)
	}
	change := Change{
		IntentID: intentID,
		Key:      state.view.Key,
		Kind:     state.view.Kind,
		Previous: previous,
		Status:   next,
		Reason:   reason,
	}
	m.mu.Unlock()
	m.notify(change)
}

func (m *Manager) waitInterval(duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return !m.stopped()
	case <-m.ctx.Done():
		return false
	}
}

func (m *Manager) stopped() bool {
	return m.ctx.Err() != nil
}

func (m *Manager) notify(change Change) {
	m.mu.Lock()
	listeners := make([]func(Change), 0, len(m.listeners))
	for _, fn := range m.listeners {
		listeners = append(listeners, fn)
	}
	m.mu.Unlock()
	for _, fn := range listeners {
		// 监听器 panic 不能带走状态机；按诊断静默丢弃。
		func() {
			defer func() { _ = recover() }()
			fn(change)
		}()
	}
}

func (s *intentState) snapshot() IntentView {
	view := s.view
	view.Content = cloneBytes(s.view.Content)
	view.Result = cloneBytes(s.view.Result)
	return view
}

// normalizeIntent 校验并补齐意图字段。只做结构校验，不校验写入内容。
func normalizeIntent(intent Intent) (Intent, error) {
	if intent.Key == "" {
		return Intent{}, fmt.Errorf("%w: 缺少幂等键", ErrInvalidIntent)
	}
	if !intent.Kind.valid() {
		return Intent{}, fmt.Errorf("%w: 未知类型 %q", ErrInvalidIntent, intent.Kind)
	}
	if intent.Kind.sessionScoped() && intent.SessionID == "" {
		return Intent{}, fmt.Errorf("%w: %s 缺少目标会话", ErrInvalidIntent, intent.Kind)
	}
	if intent.Delivery == "" {
		intent.Delivery = DeliveryQueue
	}
	if intent.Delivery != DeliveryQueue && intent.Delivery != DeliveryInterject {
		return Intent{}, fmt.Errorf("%w: 未知投递方式 %q", ErrInvalidIntent, intent.Delivery)
	}
	if len(intent.Params) > 0 {
		if !json.Valid(intent.Params) {
			return Intent{}, fmt.Errorf("%w: 参数不是合法 JSON", ErrInvalidIntent)
		}
		var compacted bytes.Buffer
		if err := json.Compact(&compacted, intent.Params); err != nil {
			return Intent{}, fmt.Errorf("%w: 参数无法解析", ErrInvalidIntent)
		}
		intent.Params = append(json.RawMessage(nil), compacted.Bytes()...)
	}
	// 各类型按 op 做结构校验；只判断动作是否认得、必需字段在不在，不校验内容语义。
	switch intent.Kind {
	case KindSessionLifecycle:
		// 新建时还没有会话标识；改名、删除、开分支与分叉必须有。
		op, err := sessionLifecycleOp(intent.Params)
		if err != nil {
			return Intent{}, err
		}
		switch op {
		case sessionOpCreate:
			// 新建会话：SessionID 可以为空；服务端在回执里给出新会话标识。
		case sessionOpRename, sessionOpDelete, sessionOpBranchCreate, sessionOpFork:
			// 改名、删除、开分支与分叉都发生在已有会话里，必须有目标会话。
			if intent.SessionID == "" {
				return Intent{}, fmt.Errorf("%w: 会话生命周期 %s 缺少目标会话", ErrInvalidIntent, op)
			}
		case sessionOpBranchEnd:
			if intent.SessionID == "" {
				return Intent{}, fmt.Errorf("%w: 会话生命周期 branch.end 缺少目标会话", ErrInvalidIntent)
			}
			branchID, err := branchEndID(intent)
			if err != nil {
				return Intent{}, err
			}
			// 分支标识统一放到 Intent.BranchID，核对归属时能据此比对。
			intent.BranchID = branchID
		default:
			return Intent{}, fmt.Errorf("%w: 未知会话生命周期动作 %q", ErrInvalidIntent, op)
		}
	case KindShellRun:
		if err := validateShellRunParams(intent.Params); err != nil {
			return Intent{}, err
		}
	case KindModalityWrite:
		if err := validateModalityWriteParams(intent.Params); err != nil {
			return Intent{}, err
		}
	}
	return intent, nil
}

// 会话生命周期动作；参数里的 op 归一到这些值。
const (
	sessionOpCreate       = "create"
	sessionOpRename       = "rename"
	sessionOpDelete       = "delete"
	sessionOpBranchCreate = "branch.create"
	sessionOpBranchEnd    = "branch.end"
	sessionOpFork         = "fork"
)

// sessionLifecycleOps 把参数里的 op 归一为动作。目录领域现有写法带 catalog
// 前缀；本版同时接受裸写法，切换类型时不必改参数。
var sessionLifecycleOps = map[string]string{
	"create":         sessionOpCreate,
	"rename":         sessionOpRename,
	"delete":         sessionOpDelete,
	"catalog.create": sessionOpCreate,
	"catalog.rename": sessionOpRename,
	"catalog.delete": sessionOpDelete,
	"branch.create":  sessionOpBranchCreate,
	"branch.end":     sessionOpBranchEnd,
	"fork":           sessionOpFork,
}

// branchEndID 取结束分支的目标标识。Intent.BranchID 与参数 branchId 都可以给；
// 两处都给时必须一致，否则拒绝。
func branchEndID(intent Intent) (string, error) {
	var body struct {
		BranchID string `json:"branchId"`
	}
	if len(intent.Params) > 0 {
		if err := json.Unmarshal(intent.Params, &body); err != nil {
			return "", fmt.Errorf("%w: 会话生命周期 branch.end 参数无法解析", ErrInvalidIntent)
		}
	}
	if intent.BranchID != "" && body.BranchID != "" && intent.BranchID != body.BranchID {
		return "", fmt.Errorf("%w: 会话生命周期 branch.end 的分支标识不一致", ErrInvalidIntent)
	}
	branchID := intent.BranchID
	if branchID == "" {
		branchID = body.BranchID
	}
	if branchID == "" {
		return "", fmt.Errorf("%w: 会话生命周期 branch.end 缺少分支标识", ErrInvalidIntent)
	}
	return branchID, nil
}

// 会话内命令动作与模态写入动作。参数里的 op 取这些值。
const (
	shellOpRun              = "run"
	shellOpStop             = "stop"
	modalityOpTaskSubmit    = "task.submit"
	modalityOpTaskCancel    = "task.cancel"
	modalityOpRealtimeOpen  = "realtime.open"
	modalityOpRealtimeClose = "realtime.close"
)

var shellRunOps = map[string]bool{
	shellOpRun:  true,
	shellOpStop: true,
}

var modalityWriteOps = map[string]bool{
	modalityOpTaskSubmit:    true,
	modalityOpTaskCancel:    true,
	modalityOpRealtimeOpen:  true,
	modalityOpRealtimeClose: true,
}

// sessionLifecycleOp 从参数里取会话生命周期动作并归一。未知动作直接拒绝，不猜。
func sessionLifecycleOp(params json.RawMessage) (string, error) {
	op, err := requireOp(params, KindSessionLifecycle)
	if err != nil {
		return "", err
	}
	action, ok := sessionLifecycleOps[op]
	if !ok {
		return "", fmt.Errorf("%w: 未知会话生命周期动作 %q", ErrInvalidIntent, op)
	}
	return action, nil
}

// validateShellRunParams 校验会话内命令执行/停止的动作与必需字段。
// run 必须有命令；stop 必须有运行标识。运行标识也可放在 Intent.InputID 里，
// 供未知核对使用。
func validateShellRunParams(params json.RawMessage) error {
	op, err := requireOp(params, KindShellRun)
	if err != nil {
		return err
	}
	if !shellRunOps[op] {
		return fmt.Errorf("%w: 未知会话命令动作 %q", ErrInvalidIntent, op)
	}
	var body struct {
		RunID   string `json:"runId"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(params, &body); err != nil {
		return fmt.Errorf("%w: 会话命令参数无法解析", ErrInvalidIntent)
	}
	switch op {
	case shellOpRun:
		if strings.TrimSpace(body.Command) == "" {
			return fmt.Errorf("%w: 会话命令 run 缺少命令", ErrInvalidIntent)
		}
	case shellOpStop:
		if body.RunID == "" {
			return fmt.Errorf("%w: 会话命令 stop 缺少运行标识", ErrInvalidIntent)
		}
	}
	return nil
}

// validateModalityWriteParams 校验模态写入的动作。任务与连接的标识由调用方放在
// 参数里，并建议同时填进 Intent.InputID，供未知核对使用。
func validateModalityWriteParams(params json.RawMessage) error {
	op, err := requireOp(params, KindModalityWrite)
	if err != nil {
		return err
	}
	if !modalityWriteOps[op] {
		return fmt.Errorf("%w: 未知模态写入动作 %q", ErrInvalidIntent, op)
	}
	return nil
}

// requireOp 取出参数里的 op。参数必须是非空对象，op 不能为空。
func requireOp(params json.RawMessage, kind Kind) (string, error) {
	if len(params) == 0 {
		return "", fmt.Errorf("%w: %s 意图缺少参数", ErrInvalidIntent, kind)
	}
	var body struct {
		Op string `json:"op"`
	}
	if err := json.Unmarshal(params, &body); err != nil {
		return "", fmt.Errorf("%w: %s 参数无法解析", ErrInvalidIntent, kind)
	}
	if body.Op == "" {
		return "", fmt.Errorf("%w: %s 意图缺少 op", ErrInvalidIntent, kind)
	}
	return body.Op, nil
}

// sameIntent 判断重复登记是否仍是同一意图。调用方未给 InputID 时，以首次
// 生成的那一个为准，不把空值当作不同意图。
func sameIntent(existing Request, incoming Intent) bool {
	if existing.Kind != incoming.Kind ||
		existing.SessionID != incoming.SessionID ||
		existing.BranchID != incoming.BranchID ||
		existing.Delivery != incoming.Delivery {
		return false
	}
	if incoming.InputID != "" && existing.InputID != incoming.InputID {
		return false
	}
	return bytes.Equal(existing.Params, incoming.Params)
}

func cloneBytes(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func randomID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
