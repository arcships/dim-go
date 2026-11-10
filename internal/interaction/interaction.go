// Package interaction 是交互裁决领域：维护审批与提问的待决集合，按服务端事实
// 推进队首，并通过写登记发出应答、长期允许与成员停止意图。
//
// 范围内：审批与提问分开维护的待决集合；按到达顺序排成一条队列，一次只推进一条；
// 审批按通知事实与重放驱动，提问按 `session.question.pending` 清单（或清单失败后的
// 历史推导）与截止点之后的 `ask_user` 事实驱动，按工具结果结算；断线清空待决、以
// 重连后的事实为准；应答、取消、长期允许；长期允许先写规则后发应答；重复应答与
// 已结算项的拒绝；权限档位的保存与透传；成员停止。
//
// 不包含：界面、连接实现、协议 wire 定义、会话折叠内部、目录、账号、管理面。
// 本包只消费归一化事实（Fact、QuestionSeed、QuestionItem）与写登记（Writer），
// 不自行读取服务端存储，也不生成服务端标识。
package interaction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// PendingKind 是待决项的类型。审批与提问分开维护，但共用一条推进队列。
type PendingKind string

const (
	// PendingApproval 是一次审批请求。
	PendingApproval PendingKind = "approval"
	// PendingQuestion 是一次提问请求。
	PendingQuestion PendingKind = "question"
)

// Decision 是用户对一条待决项作出的决定。
type Decision string

const (
	// DecisionApprove 是同意一次审批。
	DecisionApprove Decision = "approve"
	// DecisionReject 是拒绝一次审批或提问。
	DecisionReject Decision = "reject"
	// DecisionAllowLongTerm 是把这次审批写入长期规则后再同意。
	DecisionAllowLongTerm Decision = "allow_long_term"
	// DecisionCancel 是取消一次提问（不改变服务端状态的自愿放弃）。
	DecisionCancel Decision = "cancel"
	// DecisionAnswer 是回答一次提问，需要带应答原文。
	DecisionAnswer Decision = "answer"
)

// Response 是用户对一条待决项作出的决定。DecisionAnswer 时必须带合法 JSON 的
// Answer；其它决定忽略 Answer。
type Response struct {
	Decision Decision
	Answer   json.RawMessage
}

// Question 是一条提问里的一个问题。候选项与是否多选由服务端事实给出，本包不解释。
type Question struct {
	ID       string
	Text     string
	Options  []string
	Multiple bool
}

// Fact 是一条归一化后的审批事实。wire 形状由连接领域负责，调用方在交界处把
// 审批请求与取消/结算通知映射成这个结构。提问不走这里：见 QuestionSeed 与
// QuestionItem。
type Fact struct {
	Kind      PendingKind
	RequestID string
	SessionID string
	BranchID  string
	TurnID    string
	// Action 是审批的动作描述；提问为空。
	Action string
	// AllowLongTerm 表示服务端声明这次审批可以长期允许。
	AllowLongTerm bool
	// LongTermParams 是长期规则的写入参数原文。本包不解释内容，只在选择长期
	// 允许时原样交给写登记；没有参数即无法写入长期规则。
	LongTermParams json.RawMessage
}

// Snapshot 是一次审批重放快照。Pending 的顺序就是服务端审批的到达顺序，
// 整体替换本地审批集合：不累加、不与旧集合去重合并；提问集合不受影响。
type Snapshot struct {
	Pending []Fact
}

// Rejection 是一条被丢弃事实的原因；怎么记录由调用方决定。
type Rejection struct {
	Kind      PendingKind
	RequestID string
	Reason    string
}

// 丢弃原因。
const (
	// RejectionInvalidFact 表示事实缺类型、缺请求标识、缺会话标识，或参数不是合法 JSON。
	RejectionInvalidFact = "invalid_fact"
	// RejectionSessionMismatch 表示事实属于别的会话。
	RejectionSessionMismatch = "session_mismatch"
	// RejectionDuplicateRequest 表示请求标识已在待决集合里（或快照里重复）。
	RejectionDuplicateRequest = "duplicate_request"
	// RejectionQuestionSource 表示用审批事实通道投递了提问；提问改由清单与工具事实驱动。
	RejectionQuestionSource = "question_source"
	// RejectionBelowCutoff 表示事实序号不晚于清单截止点，已由清单反映，丢弃。
	RejectionBelowCutoff = "below_cutoff"
	// RejectionAlreadySettled 表示该调用已被工具结果结算，旧事实不再装回来。
	RejectionAlreadySettled = "already_settled"
)

// Item 是一条待决项的只读视图。字段是拷贝，调用方改不动内部状态。
type Item struct {
	Kind      PendingKind
	RequestID string
	SessionID string
	BranchID  string
	TurnID    string
	Action    string
	// AllowLongTerm 是服务端声明的可长期允许标记；是否真的能写还取决于是否
	// 带长期规则参数。
	AllowLongTerm bool
	// LongTermAvailable 表示本端可以写长期规则（服务端声明可长期允许，且事实
	// 带了规则参数）。
	LongTermAvailable bool
	// Questions 是提问批次里按调用顺序汇总的问题；审批为空。
	Questions []Question
	// CallIDs 是提问批次里尚未结算的 ask_user 调用；审批为空。
	CallIDs []string
	// Source 是提问集合的来源：清单或历史推导；审批为空。
	Source QuestionSource
	// Sequence 是提问批次的起始事实序号；审批为 0。
	Sequence int64
	// State 是这条待决项在推进状态机里的状态。
	State State
	// Decision 是已发出并被写登记受理的决定；未答时为空。
	Decision Decision
	// Position 是它在队列里的位置：0 是队首（当前一条）。
	Position int
}

// Counts 是待决集合的条数。已答但尚未结算的项仍算待决。
type Counts struct {
	Approvals int
	Questions int
}

// Writer 是写登记的窄接口；*write.Manager 直接满足它。应答、长期允许与成员
// 停止都必须经过它。
type Writer interface {
	Register(intent write.Intent) (write.IntentView, error)
	Wait(ctx context.Context, intentID string) (write.IntentView, error)
}

// Config 是交互裁决的配置。每个打开的会话一个裁决器，因此会话标识必填。
type Config struct {
	SessionID string
	Writer    Writer
}

// Arbiter 持有一个会话的审批集合、提问集合与推进队列。所有导出方法可并发调用。
type Arbiter struct {
	sessionID string
	writer    Writer

	mu sync.Mutex
	// queue 是一条统一队列：审批与提问按到达顺序排在一起，队首就是当前呈现项。
	queue []*entry
	byID  map[string]*entry
	// answered 记录已被写登记受理的决定，跨重放保留，用来拒绝重复应答。
	answered map[string]Decision
	// rules 记录长期规则已写入的请求标识，跨重放与断线保留：重连后只重试应答，
	// 不重写规则、也不允许改口。
	rules map[string]struct{}
	// inProgress 是正在写应答的请求标识，同一时刻只允许一次应答。
	inProgress map[string]struct{}
	// attempts 按（请求标识、步骤、决定）递增，给每次重试一个新的幂等键。
	attempts map[string]int
	// 提问来源：截止点、分支、来源与是否装载过。清单优先，读取失败才退回推导。
	questionFloor  int64
	questionBranch string
	questionSource QuestionSource
	questionSeeded bool
	// questionSettled 记录截止点以上已结算的调用（键为请求+调用），防止旧事实装回来。
	questionSettled map[string]int64
	// permissionTier 与 permissionSet 是本会话的权限档位；没有设置过就不猜默认值。
	permissionTier PermissionTier
	permissionSet  bool
}

// entry 内部保存一条待决项及其推进状态。
type entry struct {
	fact Fact
	// state 只由本包的迁移入口改。
	state State
	// decision 是已受理的决定。
	decision Decision
	// ruleWritten 表示长期规则已写入成功；重试时不再重写规则。
	ruleWritten bool
	// retryResponse 表示长期规则已写入，但上一次应答写入明确失败，允许只重试应答。
	retryResponse bool
	// source 是提问条目的来源；审批为空。
	source QuestionSource
	// calls 是提问批次里尚未结算的 ask_user 调用，按清单或到达顺序。
	calls     map[string]*questionCall
	callOrder []string
}

// 错误值；调用方按 errors.Is 判断，不匹配错误文本。
var (
	// ErrUnknownPending 表示待决集合里没有这条请求。
	ErrUnknownPending = errors.New("未知待决项")
	// ErrNotCurrent 表示这条请求不是当前队首；一次只推进一条。
	ErrNotCurrent = errors.New("待决项尚未轮到")
	// ErrAlreadyAnswered 表示这条请求已经应答或被服务端结算，不再接受应答。
	ErrAlreadyAnswered = errors.New("待决项已应答或已结算")
	// ErrResponseInProgress 表示同一待决项正在写应答。
	ErrResponseInProgress = errors.New("待决项正在应答")
	// ErrUnsupportedDecision 表示该类型不接受这个决定。
	ErrUnsupportedDecision = errors.New("不支持的决定")
	// ErrAnswerRequired 表示提问应答缺少合法的应答原文。
	ErrAnswerRequired = errors.New("提问应答缺少应答内容")
	// ErrLongTermUnavailable 表示该审批不能长期允许，或缺长期规则参数。
	ErrLongTermUnavailable = errors.New("该审批不能长期允许")
	// ErrLongTermWriteFailed 表示长期规则没有写入成功，因此未发出应答。
	ErrLongTermWriteFailed = errors.New("长期允许写入失败，未发出应答")
	// ErrResponseWriteFailed 表示应答没有发出（明确失败或未发生）。
	ErrResponseWriteFailed = errors.New("应答未发出")
	// ErrResponseUnsettled 表示应答写入尚未收敛（可能已到达），按已答处理、不重发。
	ErrResponseUnsettled = errors.New("应答结果未知，未重发")
	// ErrInvalidMember 表示成员停止缺少成员标识。
	ErrInvalidMember = errors.New("成员停止缺少成员标识")
)

// New 创建交互裁决。启动时校验两张推进状态机；表不合法即失败。
func New(cfg Config) (*Arbiter, error) {
	if cfg.SessionID == "" {
		return nil, errors.New("交互裁决缺少会话标识")
	}
	if cfg.Writer == nil {
		return nil, errors.New("交互裁决缺少写登记")
	}
	if err := validateTransitionTables(); err != nil {
		return nil, err
	}
	return &Arbiter{
		sessionID:       cfg.SessionID,
		writer:          cfg.Writer,
		byID:            make(map[string]*entry),
		answered:        make(map[string]Decision),
		rules:           make(map[string]struct{}),
		inProgress:      make(map[string]struct{}),
		attempts:        make(map[string]int),
		questionSettled: make(map[string]int64),
	}, nil
}

// SessionID 返回本裁决器所属的会话标识。
func (a *Arbiter) SessionID() string {
	return a.sessionID
}

// Apply 接收一条实时审批事实。审批走通知事实；提问不走这里，改用 SeedQuestions
// 装载清单、ApplyAskUser 接收 ask_user 事实、SettleQuestion 接收工具结果。
//
// 重复的请求标识按丢弃处理；已答过的请求会以「已答」状态重新出现，但不会再接受应答。
func (a *Arbiter) Apply(fact Fact) []Rejection {
	a.mu.Lock()
	defer a.mu.Unlock()

	if fact.Kind == PendingQuestion {
		return []Rejection{{
			Kind:      fact.Kind,
			RequestID: fact.RequestID,
			Reason:    RejectionQuestionSource,
		}}
	}
	normalized, reason := a.normalizeApproval(fact)
	if reason != "" {
		return []Rejection{{Kind: fact.Kind, RequestID: fact.RequestID, Reason: reason}}
	}
	if _, exists := a.byID[normalized.RequestID]; exists {
		return []Rejection{{
			Kind:      normalized.Kind,
			RequestID: normalized.RequestID,
			Reason:    RejectionDuplicateRequest,
		}}
	}
	a.appendLocked(normalized)
	return nil
}

// Replay 用快照整体替换审批集合；提问集合不受影响（由 SeedQuestions 整体替换）。
// 旧审批里不在快照中的项一律消失，快照里的项按给出顺序排在新提问之前；不做累加，
// 也不与旧集合去重合并。已答记录跨重放保留，因此快照里仍然待决、但本端已应答过
// 的审批会显示为「已答」，不再接受应答。
//
// 快照里的提问事实按 RejectionQuestionSource 丢弃：提问只有一个驱动者，不能从
// 审批重放通道装回来。
func (a *Arbiter) Replay(snapshot Snapshot) []Rejection {
	a.mu.Lock()
	defer a.mu.Unlock()

	queue := make([]*entry, 0, len(snapshot.Pending)+len(a.queue))
	byID := make(map[string]*entry, len(snapshot.Pending)+len(a.queue))
	var rejections []Rejection

	// 先保留现有提问条目，并把它们的请求标识视为已占用。
	var questions []*entry
	for _, old := range a.queue {
		if old.fact.Kind != PendingQuestion {
			continue
		}
		questions = append(questions, old)
		byID[old.fact.RequestID] = old
	}

	for _, fact := range snapshot.Pending {
		if fact.Kind == PendingQuestion {
			rejections = append(rejections, Rejection{
				Kind:      fact.Kind,
				RequestID: fact.RequestID,
				Reason:    RejectionQuestionSource,
			})
			continue
		}
		normalized, reason := a.normalizeApproval(fact)
		if reason != "" {
			rejections = append(rejections, Rejection{
				Kind:      fact.Kind,
				RequestID: fact.RequestID,
				Reason:    reason,
			})
			continue
		}
		if _, duplicate := byID[normalized.RequestID]; duplicate {
			rejections = append(rejections, Rejection{
				Kind:      normalized.Kind,
				RequestID: normalized.RequestID,
				Reason:    RejectionDuplicateRequest,
			})
			continue
		}
		entry := a.entryLocked(normalized)
		queue = append(queue, entry)
		byID[normalized.RequestID] = entry
	}
	queue = append(queue, questions...)

	a.queue = queue
	a.byID = byID
	return rejections
}

// OnDisconnect 在连接中断时清空本地待决集合；重新连上后审批由 Replay、提问由
// SeedQuestions 以服务端事实重建。已答记录、规则已写入记录与正在写应答的请求
// 标识保留，避免重连后重复应答；提问截止点与已结算调用随本次断线作废。
func (a *Arbiter) OnDisconnect() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.queue = nil
	a.byID = make(map[string]*entry)
	a.questionFloor = 0
	a.questionBranch = ""
	a.questionSource = ""
	a.questionSeeded = false
	a.questionSettled = make(map[string]int64)
}

// Settle 应用一条服务端审批结算事实（回执或取消）：从队列移除并推进下一条。
// 重复结算按无操作处理；提问不用这里，改由 SettleQuestion 按工具结果结算。
// 结算只来自服务端事实，本包不会在应答写入成功时自行移除。
func (a *Arbiter) Settle(requestID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	entry, ok := a.byID[requestID]
	if !ok || entry.fact.Kind != PendingApproval {
		return false
	}
	a.removeEntryLocked(entry)
	return true
}

// Current 返回队首待决项；空队列返回 false。一次只呈现这一条。
func (a *Arbiter) Current() (Item, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.queue) == 0 {
		return Item{}, false
	}
	return a.queue[0].item(0), true
}

// Item 返回一条待决项的当前视图；已结算或从未出现返回 false。
func (a *Arbiter) Item(requestID string) (Item, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, ok := a.byID[requestID]
	if !ok {
		return Item{}, false
	}
	return entry.item(a.positionLocked(entry)), true
}

// Approvals 返回待决审批的视图，按推进顺序。
func (a *Arbiter) Approvals() []Item {
	return a.filter(PendingApproval)
}

// Questions 返回待决提问的视图，按推进顺序。
func (a *Arbiter) Questions() []Item {
	return a.filter(PendingQuestion)
}

// filter 按类型取队列视图；调用方持锁由本方法负责。
func (a *Arbiter) filter(kind PendingKind) []Item {
	a.mu.Lock()
	defer a.mu.Unlock()
	items := make([]Item, 0, len(a.queue))
	for i, entry := range a.queue {
		if entry.fact.Kind == kind {
			items = append(items, entry.item(i))
		}
	}
	return items
}

// Items 返回整条队列的视图，按推进顺序；含已答但尚未结算的项。
func (a *Arbiter) Items() []Item {
	a.mu.Lock()
	defer a.mu.Unlock()
	items := make([]Item, 0, len(a.queue))
	for i, entry := range a.queue {
		items = append(items, entry.item(i))
	}
	return items
}

// Counts 返回审批与提问的待决条数。
func (a *Arbiter) Counts() Counts {
	a.mu.Lock()
	defer a.mu.Unlock()
	counts := Counts{}
	for _, entry := range a.queue {
		switch entry.fact.Kind {
		case PendingApproval:
			counts.Approvals++
		case PendingQuestion:
			counts.Questions++
		}
	}
	return counts
}

// appendLocked 把一条已校验的事实加入队尾；调用方持锁。
func (a *Arbiter) appendLocked(fact Fact) {
	entry := a.entryLocked(fact)
	a.queue = append(a.queue, entry)
	a.byID[fact.RequestID] = entry
}

// entryLocked 由事实建一条内部条目，已答过的项直接进入已答状态。
func (a *Arbiter) entryLocked(fact Fact) *entry {
	entry := &entry{fact: fact, state: StatePresented}
	if decision, ok := a.answered[fact.RequestID]; ok {
		entry.state = answeredState(fact.Kind)
		entry.decision = decision
		return entry
	}
	// 规则已写入但应答尚未受理：重连后停在待应答，只允许重试同一应答。
	if _, ok := a.rules[fact.RequestID]; ok && fact.Kind == PendingApproval {
		entry.state = StateAwaitingResponse
		entry.decision = DecisionAllowLongTerm
		entry.ruleWritten = true
		entry.retryResponse = true
	}
	return entry
}

func (a *Arbiter) positionLocked(target *entry) int {
	for i, entry := range a.queue {
		if entry == target {
			return i
		}
	}
	return -1
}

// normalizeApproval 校验审批事实并做防御性拷贝；只做结构校验，不解释业务内容。
func (a *Arbiter) normalizeApproval(fact Fact) (Fact, string) {
	if fact.Kind != PendingApproval {
		return Fact{}, RejectionInvalidFact
	}
	if fact.RequestID == "" {
		return Fact{}, RejectionInvalidFact
	}
	if fact.SessionID == "" {
		return Fact{}, RejectionInvalidFact
	}
	if fact.SessionID != a.sessionID {
		return Fact{}, RejectionSessionMismatch
	}
	if len(fact.LongTermParams) > 0 && !json.Valid(fact.LongTermParams) {
		return Fact{}, RejectionInvalidFact
	}
	fact.LongTermParams = cloneBytes(fact.LongTermParams)
	return fact, ""
}

// item 把内部条目转成只读视图；调用方持锁。
func (e *entry) item(position int) Item {
	item := Item{
		Kind:              e.fact.Kind,
		RequestID:         e.fact.RequestID,
		SessionID:         e.fact.SessionID,
		BranchID:          e.fact.BranchID,
		TurnID:            e.fact.TurnID,
		Action:            e.fact.Action,
		AllowLongTerm:     e.fact.AllowLongTerm,
		LongTermAvailable: e.fact.AllowLongTerm && len(e.fact.LongTermParams) > 0,
		Source:            e.source,
		State:             e.state,
		Decision:          e.decision,
		Position:          position,
	}
	if e.fact.Kind == PendingQuestion {
		item.CallIDs = append([]string(nil), e.callOrder...)
		for _, callID := range e.callOrder {
			call := e.calls[callID]
			if call == nil {
				continue
			}
			if item.Sequence == 0 || call.Sequence < item.Sequence {
				item.Sequence = call.Sequence
			}
			for _, question := range call.Questions {
				question.Options = append([]string(nil), question.Options...)
				item.Questions = append(item.Questions, question)
			}
		}
	}
	return item
}

// answeredState 返回已受理应答后条目所在的状态：审批停在「待应答」等结算，
// 提问进入「已答」。
func answeredState(kind PendingKind) State {
	if kind == PendingQuestion {
		return StateAnswered
	}
	return StateAwaitingResponse
}

func cloneBytes(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

// String 便于日志与测试输出。
func (r Rejection) String() string {
	return fmt.Sprintf("%s:%s:%s", r.Kind, r.RequestID, r.Reason)
}
