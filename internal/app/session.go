package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/interaction"
	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/terminal"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// SessionView 是一个打开会话的只读视图，供界面使用。它不含可写引用，也不含包内临时键。
type SessionView struct {
	SessionID    string
	BranchID     string
	BranchStatus projection.BranchStatus
	LoadPhase    projection.LoadPhase
	Rows         []projection.Row
	Turns        []projection.Turn
	Pending      projection.PendingItems
	Goal         *projection.Goal
	Branches     []projection.BranchInfo
	Selection    projection.Selection
	Anchor       int64
	HasOlder     bool
	Unlanded     int
	// Running 是打开会话时服务端给出的运行状态；当前只在 attach 回执里读取。
	Running bool
	// PendingCounts 是审批与提问的待决条数。
	PendingCounts interaction.Counts
	// Current 是队列里的第一条待决项；HasCurrent 为假表示没有待决。
	Current    interaction.Item
	HasCurrent bool
	// Compaction 是最近一次上下文压缩的摘要；Compactions 是压缩次数。
	Compaction  *projection.Compaction
	Compactions int64
}

// OpenSession 打开一个会话：建立唯一订阅、折叠首屏历史、创建交互裁决。
// 同一会话重复打开返回同一份视图，不新增连接（多窗口共用一份订阅）。
//
// 订阅阶段已经打开过的会话直接返回现有视图。
func (a *App) OpenSession(ctx context.Context, sessionID string) (SessionView, error) {
	runtime, err := a.subscribeSession(ctx, sessionID)
	if err != nil {
		return SessionView{}, err
	}
	if !runtime.historyLoaded {
		a.loadFirstHistory(ctx, runtime)
	}
	return a.viewOf(runtime), nil
}

// subscribeSession 建立会话订阅，但不折叠首屏历史（首轮历史阶段单独做）。
func (a *App) subscribeSession(ctx context.Context, sessionID string) (*sessionRuntime, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("%w: 缺少会话标识", ErrInvalidConfig)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil, ErrClosed
	}
	if existing, ok := a.sessions[sessionID]; ok {
		a.mu.Unlock()
		return existing, nil
	}
	manager, writes := a.manager, a.writes
	a.mu.Unlock()
	if manager == nil || writes == nil {
		return nil, ErrNotConnected
	}

	folder, err := projection.New(projection.Config{SessionID: sessionID})
	if err != nil {
		return nil, fmt.Errorf("创建会话投影: %w", err)
	}
	arbiter, err := interaction.New(interaction.Config{
		SessionID: sessionID,
		Writer:    writes,
		// 命令规则来源：permission.rules / permission.set_rules，只读缓存由裁决持有。
		Rules: &ruleSource{app: a},
	})
	if err != nil {
		return nil, fmt.Errorf("创建交互裁决: %w", err)
	}
	listener := projection.Listen(folder, func(rejection projection.Rejection) {
		a.record("丢弃会话事实: 分支=%s 序号=%d 类型=%s/%s 原因=%s %s",
			rejection.BranchID, rejection.Sequence, rejection.Type, rejection.Name,
			rejection.Reason, rejection.Detail)
	})
	listener.OnFatal = func(err error) {
		a.record("会话流进入终态: 会话=%s 错误=%v", sessionID, err)
	}
	// 审批与标题走同一条流：先交给它们各自的归属，再折叠。
	innerEvent := listener.OnEvent
	listener.OnEvent = func(event session.Event) {
		a.routeSessionEvent(sessionID, arbiter, event)
		if innerEvent != nil {
			innerEvent(event)
		}
	}
	// 断线先把界面标记为 detached；重连重读后再重新 ui.attach。
	innerResync := listener.OnResync
	var runtimeRef *sessionRuntime
	listener.OnResync = func(resync session.Resync) {
		if innerResync != nil {
			innerResync(resync)
		}
		if runtimeRef != nil {
			a.markUIDetached(runtimeRef)
		}
	}
	var landed atomic.Int64
	innerHistory := listener.OnHistory
	if innerHistory != nil {
		listener.OnHistory = func(replay session.HistoryReplay) {
			innerHistory(replay)
			landed.Store(replay.History.Through)
			if runtimeRef != nil {
				go a.reattachUIAfterResync(runtimeRef)
			}
		}
	}

	snapshot, lease, err := manager.Acquire(ctx, sessionID, listener)
	if err != nil {
		return nil, fmt.Errorf("打开会话订阅: %w", err)
	}
	runtime := &sessionRuntime{
		id:       sessionID,
		folder:   folder,
		arbiter:  arbiter,
		lease:    lease,
		attached: snapshot.Attach,
		snapshot: snapshot,
	}
	runtimeRef = runtime
	runtime.landed.Store(landed.Load())
	if snapshot.History.Through > runtime.landed.Load() {
		runtime.landed.Store(snapshot.History.Through)
	}

	a.mu.Lock()
	// 并发打开同一会话：后到者释放自己的租约，交回先到者的订阅。
	if existing, ok := a.sessions[sessionID]; ok {
		a.mu.Unlock()
		lease.Release()
		return existing, nil
	}
	a.sessions[sessionID] = runtime
	a.mu.Unlock()

	a.record("已建立会话订阅: %s 头部=%d", sessionID, snapshot.Head)
	return runtime, nil
}

// loadFirstHistory 折叠首屏历史，并按截止点读四类未完成事项。
// 装载状态机走"打开请求 → 首轮历史到达"。
func (a *App) loadFirstHistory(ctx context.Context, runtime *sessionRuntime) {
	if runtime.historyLoaded {
		return
	}
	if transition := runtime.folder.AdvanceLoad(projection.LoadOpenRequested); !transition.OK {
		a.record("会话装载迁移被丢弃: %s", transition.Reason)
	}
	if runtime.snapshot != nil {
		runtime.folder.ApplySnapshot(runtime.snapshot)
	}
	if transition := runtime.folder.AdvanceLoad(projection.LoadFirstTurnArrived); !transition.OK {
		a.record("会话装载迁移被丢弃: %s", transition.Reason)
	}
	runtime.historyLoaded = true
	// 清单按 attach 之后、首轮历史之后的截止点读：之后的变化由事件结算。
	a.seedPending(ctx, runtime)
	// 权限档位来自服务端策略；读不到只记诊断，不补默认档位。
	a.refreshPermission(ctx, runtime)
	// 上下文占用只能按需读取（协议 v3 不推 context/budget）；缺方法时零请求。
	a.readContextUsageQuiet(ctx, runtime)
}

// firstCatalogSession 读目录第一页，返回第一条会话标识；读不到时返回错误与空标识。
func (a *App) firstCatalogSession(ctx context.Context) (string, error) {
	catalogDomain := a.Catalog()
	if catalogDomain == nil {
		return "", ErrNotConnected
	}
	if _, err := catalogDomain.Refresh(ctx, catalog.Query{}); err != nil {
		return "", err
	}
	entries := catalogDomain.Snapshot().Entries
	if len(entries) == 0 {
		return "", nil
	}
	return entries[0].ID, nil
}

// CloseSession 关闭一个会话：释放订阅（最后一个持有者才 detach）并结束它的终端。
func (a *App) CloseSession(sessionID string) error {
	a.mu.Lock()
	runtime, ok := a.sessions[sessionID]
	if ok {
		delete(a.sessions, sessionID)
	}
	a.mu.Unlock()
	if a.terminals != nil {
		_ = a.terminals.Forget(sessionID)
	}
	if !ok {
		return nil
	}
	// 会话关闭时清理本地界面注册；服务端会随会话移除注册，不发额外请求。
	runtime.ui.clear()
	runtime.lease.Release()
	a.record("已关闭会话: %s", sessionID)
	return nil
}

// Sessions 返回当前打开的会话标识，按打开顺序不保证。
func (a *App) Sessions() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := make([]string, 0, len(a.sessions))
	for id := range a.sessions {
		ids = append(ids, id)
	}
	return ids
}

// View 返回一个已打开会话的只读视图。
func (a *App) View(sessionID string) (SessionView, bool) {
	runtime, ok := a.session(sessionID)
	if !ok {
		return SessionView{}, false
	}
	return a.viewOf(runtime), true
}

// LoadOlder 向前翻一页历史：用投影给出的游标读一页，再交回同一个折叠入口。
func (a *App) LoadOlder(ctx context.Context, sessionID string) (SessionView, error) {
	runtime, ok := a.session(sessionID)
	if !ok {
		return SessionView{}, ErrSessionNotOpen
	}
	branchID := runtime.folder.Selected()
	params, ok := runtime.folder.OlderPageRequest(branchID, a.cfg.HistoryLimit)
	if !ok {
		return a.viewOf(runtime), nil
	}
	runtime.folder.AdvanceLoad(projection.LoadPageRequested)
	page, err := runtime.lease.Subscription().History(ctx, params)
	if err != nil {
		return a.viewOf(runtime), fmt.Errorf("读取更早的轮次: %w", err)
	}
	runtime.folder.ApplyOlderPage(branchID, page)
	runtime.folder.AdvanceLoad(projection.LoadPageArrived)
	return a.viewOf(runtime), nil
}

// ReloadLatest 从上次读到的位置往后读历史并折叠：把已经落库的尾部行原地换成
// 服务端标识。
//
// 用 after 游标而不是 latestTurn：旧服务端拒绝未知字段，after 是各版本都支持的
// 读取形状。它不改分页锚点，也不发起新的订阅；没读到覆盖它的历史之前，行保持
// 无标识。
func (a *App) ReloadLatest(ctx context.Context, sessionID string) (SessionView, error) {
	runtime, ok := a.session(sessionID)
	if !ok {
		return SessionView{}, ErrSessionNotOpen
	}
	branchID := runtime.folder.Selected()
	limit := a.cfg.HistoryLimit
	after := runtime.landed.Load()
	for page := 0; page < reloadPageBudget; page++ {
		params := protocol.SessionHistoryParams{BranchID: branchID, After: &after, Limit: &limit}
		result, err := runtime.lease.Subscription().History(ctx, params)
		if err != nil {
			return a.viewOf(runtime), fmt.Errorf("重读最近一轮: %w", err)
		}
		runtime.folder.ApplyHistory(branchID, result)
		runtime.landed.Store(result.Through)
		if result.Done || result.NextAfter <= after {
			break
		}
		after = result.NextAfter
	}
	return a.viewOf(runtime), nil
}

// SubmitRequest 是一次提交意图。文本之外的业务校验归调用方，本包只要求能发出。
type SubmitRequest struct {
	// Key 是幂等键；空时按会话、分支与一次随机标识生成，两次提交是两次独立意图。
	Key string
	// SessionID 是目标会话，必填。
	SessionID string
	// BranchID 是目标分支；空表示主线由发送方决定。
	BranchID string
	// Text 是提交的文本。
	Text string
	// Delivery 是投递方式；空按排队处理（R1）。
	Delivery write.Delivery
	// InputID 是提交去重标识；空时由写登记生成 UUID。
	InputID string
}

// Submit 登记一次提交意图并立即返回意图视图；结果未知的核对与不重发由写登记负责。
func (a *App) Submit(ctx context.Context, request SubmitRequest) (write.IntentView, error) {
	if strings.TrimSpace(request.SessionID) == "" {
		return write.IntentView{}, fmt.Errorf("%w: 缺少会话标识", ErrInvalidConfig)
	}
	if strings.TrimSpace(request.Text) == "" {
		return write.IntentView{}, fmt.Errorf("%w: 提交文本为空", write.ErrInvalidIntent)
	}
	if err := a.Writable(); err != nil {
		return write.IntentView{}, err
	}
	writes := a.Writes()
	if writes == nil {
		return write.IntentView{}, ErrNotConnected
	}
	key := request.Key
	if key == "" {
		id, err := uuidV4()
		if err != nil {
			return write.IntentView{}, fmt.Errorf("生成提交标识: %w", err)
		}
		key = "submit:" + request.SessionID + ":" + request.BranchID + ":" + id
	}
	params, err := json.Marshal(map[string]any{"text": request.Text})
	if err != nil {
		return write.IntentView{}, fmt.Errorf("编码提交参数: %w", err)
	}
	intent := write.Intent{
		Key:       key,
		Kind:      write.KindSubmitInput,
		SessionID: request.SessionID,
		BranchID:  request.BranchID,
		Delivery:  request.Delivery,
		InputID:   request.InputID,
		Params:    params,
	}
	// 权限档位归交互裁决：提交路径必须先透传当前档位，服务端才能收到所选档位。
	if runtime, ok := a.session(request.SessionID); ok {
		intent, err = runtime.arbiter.WithPermission(intent)
		if err != nil {
			return write.IntentView{}, fmt.Errorf("透传权限档位: %w", err)
		}
	}
	return writes.Register(intent)
}

// Respond 对当前队首的审批或提问作出决定；顺序与一次一条由交互裁决保证。
func (a *App) Respond(
	ctx context.Context,
	sessionID string,
	requestID string,
	response interaction.Response,
) (interaction.Outcome, error) {
	if err := a.Writable(); err != nil {
		return interaction.Outcome{}, err
	}
	runtime, ok := a.session(sessionID)
	if !ok {
		return interaction.Outcome{}, ErrSessionNotOpen
	}
	return runtime.arbiter.Respond(ctx, requestID, response)
}

// StopMember 停止某个成员的运行，不触碰主线。
func (a *App) StopMember(sessionID string, member interaction.MemberRef) (write.IntentView, error) {
	if err := a.Writable(); err != nil {
		return write.IntentView{}, err
	}
	runtime, ok := a.session(sessionID)
	if !ok {
		return write.IntentView{}, ErrSessionNotOpen
	}
	return runtime.arbiter.StopMember(member)
}

// Wait 等到一条写意图进入终态；ctx 结束时返回当前视图。
func (a *App) Wait(ctx context.Context, intentID string) (write.IntentView, error) {
	if err := a.Writable(); err != nil {
		return write.IntentView{}, err
	}
	writes := a.Writes()
	if writes == nil {
		return write.IntentView{}, ErrNotConnected
	}
	return writes.Wait(ctx, intentID)
}

// StartTerminal 为会话启动本机 shell，工作目录取该会话的工作目录。
//
// 工作目录只来自目录里的服务端事实：目录没有读过或该条目没有工作目录时拒绝启动，
// 不猜测。
func (a *App) StartTerminal(sessionID string, options terminal.Options) (*terminal.Terminal, error) {
	catalogDomain := a.Catalog()
	if catalogDomain == nil {
		return nil, ErrNotConnected
	}
	entry, ok := catalogDomain.Entry(sessionID)
	if !ok {
		return nil, fmt.Errorf("%w: 目录里没有该会话，先刷新目录", ErrSessionNotOpen)
	}
	if entry.Workspace == "" {
		return nil, fmt.Errorf("%w: 该会话没有工作目录", ErrSessionNotOpen)
	}
	return a.terminals.Start(sessionID, entry.Workspace, options)
}

// session 返回一个打开会话的装配结果。
func (a *App) session(sessionID string) (*sessionRuntime, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	runtime, ok := a.sessions[sessionID]
	return runtime, ok
}

// viewOf 把折叠结果与待决集合拼成只读视图。
func (a *App) viewOf(runtime *sessionRuntime) SessionView {
	view := runtime.folder.View()
	counts := runtime.arbiter.Counts()
	current, hasCurrent := runtime.arbiter.Current()
	running := false
	if runtime.attached.Running != nil {
		running = *runtime.attached.Running
	}
	return SessionView{
		SessionID:     view.SessionID,
		BranchID:      view.BranchID,
		BranchStatus:  view.BranchStatus,
		LoadPhase:     view.LoadPhase,
		Rows:          view.Rows,
		Turns:         view.Turns,
		Pending:       view.Pending,
		Goal:          view.Goal,
		Branches:      runtime.folder.Branches(),
		Selection:     view.Selection,
		Anchor:        view.Anchor,
		HasOlder:      view.HasOlder,
		Unlanded:      view.Unlanded,
		Running:       running,
		PendingCounts: counts,
		Current:       current,
		HasCurrent:    hasCurrent,
		Compaction:    view.Compaction,
		Compactions:   view.Compactions,
	}
}

// reloadPageBudget 是一次重读最多读几页，避免服务端游标不前进时原地打转。
const reloadPageBudget = 5

// sessionMethods 读一次会话方法清单并缓存；失败缓存到本会话生命周期结束。
func (a *App) sessionMethods(ctx context.Context, runtime *sessionRuntime) (protocol.SessionMethodsResult, error) {
	runtime.methodsOnce.Do(func() {
		runtime.methods, runtime.methodsErr = runtime.lease.Subscription().Methods(ctx)
	})
	return runtime.methods, runtime.methodsErr
}

// sessionSupportsMethod 判断某个会话贡献方法是否存在。
//
// hello 的能力清单先判一次：服务端声明了就直接算有；没声明时读 session.methods，
// 两者都不知道才算缺失。缺失时不发请求，也不按版本猜。
func (a *App) sessionSupportsMethod(ctx context.Context, runtime *sessionRuntime, method string) bool {
	if conn := a.connection(); conn != nil && conn.HasFeature(method) {
		return true
	}
	methods, err := a.sessionMethods(ctx, runtime)
	if err != nil {
		a.record("读取会话方法清单失败，按能力缺失处理: %v", err)
		return false
	}
	return methods.Has(method)
}

// openRuntime 返回一个已打开会话；没有打开时按不可重试失败处理，不发请求。
func (a *App) openRuntime(sessionID, method string) (*sessionRuntime, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("%w: %s 缺少会话标识", write.ErrInvalidIntent, method)
	}
	runtime, ok := a.session(sessionID)
	if !ok {
		return nil, failure.New(failure.KindNonRetryable, method,
			fmt.Errorf("%w: %s 需要先打开会话", ErrSessionNotOpen, method))
	}
	return runtime, nil
}

// controlSupports 回答控制连接是否在 hello.features 里声明了某个方法。
//
// 控制面的方法没有会话方法表可查；能力缺失时发送方据此直接拒绝，不发请求。
func (a *App) controlSupports(method string) bool {
	conn := a.connection()
	return conn != nil && conn.HasFeature(method)
}

// readContextUsageQuiet 在打开会话后尽力读一次上下文占用；失败只记诊断。
func (a *App) readContextUsageQuiet(ctx context.Context, runtime *sessionRuntime) {
	if !a.sessionSupportsMethod(ctx, runtime, methodContextUsage) {
		return
	}
	if _, err := a.ReadContextUsage(ctx, runtime.id); err != nil {
		a.record("读取上下文占用失败: %v", err)
	}
}
