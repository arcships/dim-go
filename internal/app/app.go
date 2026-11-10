package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
	"github.com/arcships/dimagent-mygo/internal/attachments"
	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/extensions"
	"github.com/arcships/dimagent-mygo/internal/interaction"
	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/settings"
	"github.com/arcships/dimagent-mygo/internal/terminal"
	"github.com/arcships/dimagent-mygo/internal/workspace"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// Phase 是启动阶段，对应架构文档「启动与退出」。阶段的推进由宿主外壳（D10）驱动，
// 本包只提供每个阶段的执行体。
type Phase string

const (
	// PhaseInitial 表示进程刚开始，还没有装载本地状态。
	PhaseInitial Phase = "initial"
	// PhaseLocalState 表示本地状态（归档隐藏列表、附件暂存区、工作区）已装载。
	PhaseLocalState Phase = "local_state_loaded"
	// PhaseConnected 表示已握手，控制连接可用。
	PhaseConnected Phase = "connected"
	// PhaseReady 表示允许写，可以登记意图。
	PhaseReady Phase = "ready"
	// PhaseStopping 表示正在按阶段收尾。
	PhaseStopping Phase = "stopping"
	// PhaseClosed 表示已结束。
	PhaseClosed Phase = "closed"
)

// 错误值。调用方按 errors.Is 判断。
var (
	// ErrNotConnected 表示控制连接不可用：意图不会发出，按连接缺失处理。
	ErrNotConnected = errors.New("未连接服务端")
	// ErrNotReady 表示启动阶段尚未完成，拒绝新意图。
	ErrNotReady = errors.New("启动尚未完成，拒绝新意图")
	// ErrExiting 表示正在退出，拒绝新意图。
	ErrExiting = errors.New("正在退出，拒绝新意图")
	// ErrClosed 表示集成层已关闭。
	ErrClosed = errors.New("集成层已关闭")
	// ErrSessionNotOpen 表示该会话还没有打开。
	ErrSessionNotOpen = errors.New("会话尚未打开")
	// ErrInvalidConfig 表示配置不完整。
	ErrInvalidConfig = errors.New("集成层配置不合法")
)

// Config 是集成层配置。
type Config struct {
	// DataDir 是客户端数据目录：归档隐藏列表、附件暂存区与 App Server 端点都放这里。
	DataDir string
	// ServerPath 是 App Server 可执行文件；端点已存在时不会拉起它。
	ServerPath string
	// ServerArgs 是追加给服务端的参数（例如 dimcode-server 需要 app-server 子命令）。
	ServerArgs []string
	// MockModel 为真时用 mock 模型拉起服务端，只用于验收。
	MockModel bool
	// ModelConfig 是显式的模型配置路径。
	ModelConfig string
	// ClientName 是 hello 里的客户端名，只用于诊断。
	ClientName string
	// WorkspaceRoot 是工作区根；空表示不提供工作区能力。
	WorkspaceRoot string
	// InitialSessionID 是订阅阶段要打开的会话；空时由订阅阶段读目录取第一条。
	InitialSessionID string
	// Version 是客户端版本，只用于进程级诊断（必须形如 x.y.z）。
	Version string
	// HistoryLimit 是首屏历史与向前翻页的单页条数；0 用默认值。
	HistoryLimit int
	// RequestTimeout 是控制连接单请求超时；0 用连接领域默认值。
	RequestTimeout time.Duration
	// SendTimeout 是写意图单次发送上限；超时按结果未知处理。
	SendTimeout time.Duration
	// Verify 是结果未知时的核对策略；零值用默认值。
	Verify write.VerifyPolicy
	// CloseTimeout 是没有宿主外壳时收尾每一步的上限；0 用 10 秒。
	CloseTimeout time.Duration
	// AttachmentLimits 是附件暂存上限；零值用默认值。
	AttachmentLimits attachments.Limits
	// Logf 接收本包的自由文本诊断；nil 时丢弃。它不写入进程级诊断记录。
	Logf func(format string, args ...any)
}

// sessionRuntime 是一个打开会话的装配结果。字段只在 App.mu 下整体替换。
type sessionRuntime struct {
	id       string
	folder   *projection.Folder
	arbiter  *interaction.Arbiter
	lease    *session.Lease
	attached protocol.SessionAttachResult
	snapshot *session.Snapshot
	// historyLoaded 表示首屏历史已经折叠过。
	historyLoaded bool
	// landed 是最近一次历史读取覆盖到的头部序号，用作重读尾部的 after 游标。
	// 它只是读取位置，不参与事实判断；分页锚点仍由投影持有。
	landed atomic.Int64
	// methods 是会话贡献方法清单的缓存；一次读取服务本会话生命周期。
	methodsOnce sync.Once
	methods     protocol.SessionMethodsResult
	methodsErr  error
}

// App 是集成层的对外类型。所有导出方法可并发调用。
//
// 本包不决定"什么时候允许写"：启动阶段、退出阶段与写闸门由宿主外壳（D10）驱动，
// 这里只提供每个阶段的执行体，并通过 SetWriteGate 接受外壳的闸门。
type App struct {
	cfg Config

	// 本地状态：随进程存在，丢了不影响服务端事实。
	archive   *catalog.Archive
	staging   *attachments.Store
	workspace *workspace.Workspace
	terminals *terminal.Registry

	// 连接与订阅。
	supervisor *daemon.Supervisor
	conn       *daemon.Connection
	manager    *session.Manager

	// 各领域实例。
	writes     *write.Manager
	catalog    *catalog.Catalog
	account    *account.Account
	settings   *settings.Manager
	extensions *extensions.Service

	mu     sync.Mutex
	phase  Phase
	closed bool
	// 各启动阶段是否已完成；阶段只能按序完成，重复调用是空操作。
	localLoaded   bool
	connected     bool
	subscribed    bool
	historyLoaded bool
	writesAllowed bool
	// stopping 由退出第一阶段置位；置位后一切新意图被拒。
	stopping bool
	// startupSession 是订阅阶段打开的会话；首轮历史阶段用它。
	startupSession string
	// writeGate 是宿主外壳的写闸门（host.Shell.Writable）；nil 表示没有外壳。
	writeGate func() error

	sessions  map[string]*sessionRuntime
	notes     []string
	noticeOff func()

	// 退出阶段的交接：StopIntents 取出引用，EndInteraction 与 CloseConnection 消费。
	pendingSessions   []*sessionRuntime
	pendingManager    *session.Manager
	pendingWrites     *write.Manager
	pendingExtensions *extensions.Service
}

// New 完成配置校验并创建集成层；它不装载本地状态、也不连服务端。
// 本地状态在 LoadLocalState 阶段装载，连接在 Connect 阶段建立。
func New(cfg Config) (*App, error) {
	if cfg.DataDir == "" {
		return nil, fmt.Errorf("%w: 缺少客户端数据目录", ErrInvalidConfig)
	}
	if cfg.HistoryLimit == 0 {
		cfg.HistoryLimit = 50
	}
	if cfg.SendTimeout == 0 {
		cfg.SendTimeout = 30 * time.Second
	}
	if cfg.CloseTimeout == 0 {
		cfg.CloseTimeout = 10 * time.Second
	}
	if cfg.Verify.MaxAttempts == 0 {
		cfg.Verify = write.VerifyPolicy{MaxAttempts: 3, Interval: time.Second, Timeout: 5 * time.Second}
	}
	if cfg.AttachmentLimits == (attachments.Limits{}) {
		cfg.AttachmentLimits = attachments.DefaultLimits()
	}

	return &App{
		cfg:       cfg,
		phase:     PhaseInitial,
		terminals: terminal.NewRegistry(),
		sessions:  make(map[string]*sessionRuntime),
	}, nil
}

// SetWriteGate 设置宿主外壳的写闸门。闸门返回错误时，本包登记意图的入口一律拒绝。
func (a *App) SetWriteGate(gate func() error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.writeGate = gate
}

// LoadLocalState 装载本地状态：归档隐藏列表、附件暂存区、工作区根。
// 它对应启动阶段 local_state；重复调用是空操作。
func (a *App) LoadLocalState(ctx context.Context) error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return ErrClosed
	}
	if a.localLoaded {
		a.mu.Unlock()
		return nil
	}
	a.mu.Unlock()

	if err := os.MkdirAll(a.cfg.DataDir, 0o700); err != nil {
		return fmt.Errorf("创建客户端数据目录: %w", err)
	}
	// 归档列表损坏或不可读时按空处理：它只影响列表整洁，不影响服务端事实。
	archive, err := catalog.OpenArchive(filepath.Join(a.cfg.DataDir, "catalog-archive.json"))
	if err != nil {
		a.record("装载归档隐藏列表失败，按空列表处理: %v", err)
	}
	store, err := attachments.Open(a.cfg.DataDir, a.cfg.AttachmentLimits)
	if err != nil {
		return fmt.Errorf("打开附件暂存区: %w", err)
	}
	if report, err := store.Cleanup(); err != nil {
		a.record("启动清理附件暂存区失败: %v", err)
	} else if removed := len(report.RemovedInvalid) + len(report.RemovedUnreferenced) + len(report.RemovedOrphans); removed > 0 {
		a.record("启动清理附件暂存区: 删除 %d 项，释放 %d 字节", removed, report.BytesFreed)
	}
	var ws *workspace.Workspace
	if a.cfg.WorkspaceRoot != "" {
		opened, err := workspace.NewWorkspace(a.cfg.WorkspaceRoot)
		if err != nil {
			return fmt.Errorf("打开工作区: %w", err)
		}
		ws = opened
	}

	a.mu.Lock()
	a.archive = archive
	a.staging = store
	a.workspace = ws
	a.localLoaded = true
	a.phase = PhaseLocalState
	a.mu.Unlock()
	a.record("已装载本地状态: 工作区=%t", ws != nil)
	return nil
}

// Connect 连服务端并装配各领域。它对应启动阶段 connect；重复调用是空操作。
func (a *App) Connect(ctx context.Context) error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return ErrClosed
	}
	if a.connected {
		a.mu.Unlock()
		return nil
	}
	if !a.localLoaded {
		a.mu.Unlock()
		return fmt.Errorf("%w: 本地状态尚未装载", ErrNotReady)
	}
	a.mu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}
	supervisor, err := daemon.New(daemon.Config{
		DataDir:        a.cfg.DataDir,
		ServerPath:     a.cfg.ServerPath,
		ServerArgs:     a.cfg.ServerArgs,
		MockModel:      a.cfg.MockModel,
		ModelConfig:    a.cfg.ModelConfig,
		ClientName:     a.cfg.ClientName,
		RequestTimeout: a.cfg.RequestTimeout,
	})
	if err != nil {
		return fmt.Errorf("解析服务端点: %w", err)
	}
	conn, err := supervisor.Connect(ctx)
	if err != nil {
		supervisor.Close(nil)
		return fmt.Errorf("连接服务端: %w", err)
	}

	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		supervisor.Close(nil)
		return fmt.Errorf("创建会话流管理器: %w", err)
	}
	writes, err := write.New(write.Config{
		Sender:      &appSender{app: a},
		Inspector:   &appInspector{app: a},
		SendTimeout: a.cfg.SendTimeout,
		Verify:      a.cfg.Verify,
		// 服务端要求提交去重标识是 UUID，不能沿用写登记的十六进制默认值。
		NewID: uuidV4,
	})
	if err != nil {
		manager.Close()
		supervisor.Close(nil)
		return fmt.Errorf("创建写登记: %w", err)
	}
	catalogDomain, err := catalog.New(catalog.Config{
		Reader:   &catalogReader{app: a},
		Writes:   writes,
		Archive:  a.archive,
		Releaser: &sessionReleaser{app: a},
		History:  &turnHistorySource{app: a},
	})
	if err != nil {
		writes.Close()
		manager.Close()
		supervisor.Close(nil)
		return fmt.Errorf("创建会话目录: %w", err)
	}
	accountDomain, err := account.New(account.Config{
		Status:  &accountStatusReader{app: a},
		Usage:   &accountUsageReader{app: a},
		Models:  &accountModelReader{app: a},
		Auth:    &accountAuthSource{app: a},
		Actions: writes,
	})
	if err != nil {
		writes.Close()
		manager.Close()
		supervisor.Close(nil)
		return fmt.Errorf("创建账号领域: %w", err)
	}
	settingsDomain, err := settings.New(settings.Config{
		Features: &featureSource{app: a},
		Sources:  &settingsSourcesReader{app: a},
		Projects: &settingsProjectsReader{app: a},
		Settings: &settingsDocumentReader{app: a},
		Models:   &settingsModelReader{app: a},
		Account:  accountDomain,
		Storage:  &settingsStorageReader{app: a},
		Runtimes: &settingsRuntimeReader{app: a},
		Modality: &settingsModalityReader{app: a},
		Writer:   writes,
	})
	if err != nil {
		writes.Close()
		manager.Close()
		supervisor.Close(nil)
		return fmt.Errorf("创建设置领域: %w", err)
	}
	extensionReaders := &extensionReaders{app: a}
	extensionsDomain, err := extensions.New(extensions.Config{
		Plugins:     extensionReaders,
		Skills:      extensionReaders,
		Hooks:       extensionReaders,
		ToolServers: extensionReaders,
		Scheduler:   extensionReaders,
		Workflows:   extensionReaders,
		Wakeups:     extensionReaders,
		Writer:      writes,
		Logf:        a.record,
	})
	if err != nil {
		writes.Close()
		manager.Close()
		supervisor.Close(nil)
		return fmt.Errorf("创建扩展与后台领域: %w", err)
	}

	a.mu.Lock()
	a.supervisor = supervisor
	a.conn = conn
	a.manager = manager
	a.writes = writes
	a.catalog = catalogDomain
	a.account = accountDomain
	a.settings = settingsDomain
	a.extensions = extensionsDomain
	a.connected = true
	a.phase = PhaseConnected
	a.mu.Unlock()

	a.attachControlNotices(conn)
	a.record("已连接服务端: 实例=%s 版本=%s 协议=%d 自启=%t",
		conn.Hello.InstanceID, conn.Hello.ServerVersion, conn.Hello.ProtocolVersion, conn.Owned)
	return nil
}

// Subscribe 建立订阅：打开订阅阶段指定的会话，没有指定时读目录取第一条。
// 它对应启动阶段 subscribe；重复调用是空操作。
//
// 没有可打开的会话不是失败：客户端可以在没有打开会话的情况下就绪，只记一条诊断。
func (a *App) Subscribe(ctx context.Context) error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return ErrClosed
	}
	if a.subscribed {
		a.mu.Unlock()
		return nil
	}
	if !a.connected {
		a.mu.Unlock()
		return fmt.Errorf("%w: 尚未连接服务端", ErrNotReady)
	}
	target := a.cfg.InitialSessionID
	a.mu.Unlock()

	if target == "" {
		first, err := a.firstCatalogSession(ctx)
		if err != nil {
			a.record("订阅阶段：没有指定会话，读取目录失败，本次不打开会话: %v", err)
			first = ""
		}
		target = first
	}
	if target != "" {
		if _, err := a.subscribeSession(ctx, target); err != nil {
			return err
		}
	} else {
		a.record("订阅阶段：没有可打开的会话")
	}

	a.mu.Lock()
	a.subscribed = true
	a.startupSession = target
	a.mu.Unlock()
	return nil
}

// FirstHistory 折叠订阅阶段的会话首屏历史。它对应启动阶段 first_history。
func (a *App) FirstHistory(ctx context.Context) error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return ErrClosed
	}
	if a.historyLoaded {
		a.mu.Unlock()
		return nil
	}
	target := a.startupSession
	a.mu.Unlock()

	if target == "" {
		a.mu.Lock()
		a.historyLoaded = true
		a.mu.Unlock()
		a.record("首轮历史阶段：没有打开的会话")
		return nil
	}
	runtime, ok := a.session(target)
	if !ok {
		return fmt.Errorf("%w: 订阅阶段没有建立 %s 的订阅", ErrNotReady, target)
	}
	a.loadFirstHistory(ctx, runtime)
	head := int64(0)
	events := 0
	if runtime.snapshot != nil {
		head = runtime.snapshot.Head
		events = len(runtime.snapshot.History.Events)
	}

	a.mu.Lock()
	a.historyLoaded = true
	a.mu.Unlock()
	a.record("首轮历史：会话=%s 头部=%d 首屏事实=%d", target, head, events)
	return nil
}

// AllowWrites 打开写入闸门。它对应启动阶段 allow_writes。
func (a *App) AllowWrites(ctx context.Context) error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return ErrClosed
	}
	if !a.historyLoaded {
		a.mu.Unlock()
		return fmt.Errorf("%w: 首轮历史尚未到达", ErrNotReady)
	}
	a.writesAllowed = true
	a.phase = PhaseReady
	a.mu.Unlock()
	a.record("允许写：启动阶段完成")
	return nil
}

// StopIntents 是退出第一阶段：从此拒绝一切新意图。它幂等。
func (a *App) StopIntents(ctx context.Context) error {
	a.mu.Lock()
	a.stopping = true
	if a.phase != PhaseClosed {
		a.phase = PhaseStopping
	}
	sessions := make([]*sessionRuntime, 0, len(a.sessions))
	for id, runtime := range a.sessions {
		sessions = append(sessions, runtime)
		delete(a.sessions, id)
	}
	a.pendingSessions = append(a.pendingSessions, sessions...)
	a.pendingManager = a.manager
	a.pendingWrites = a.writes
	a.pendingExtensions = a.extensions
	a.mu.Unlock()

	a.record("停止新意图")
	return nil
}

// EndInteraction 是退出第二阶段：释放订阅、结束在途写的登记、停事件跟随与终端。
func (a *App) EndInteraction(ctx context.Context) error {
	a.mu.Lock()
	sessions := a.pendingSessions
	a.pendingSessions = nil
	writes := a.pendingWrites
	a.pendingWrites = nil
	extensionsDomain := a.pendingExtensions
	a.pendingExtensions = nil
	a.mu.Unlock()

	for _, runtime := range sessions {
		runtime.lease.Release()
	}
	if writes != nil {
		writes.Close()
	}
	if extensionsDomain != nil {
		extensionsDomain.Close()
	}
	if a.terminals != nil {
		a.terminals.StopAll()
	}
	a.record("已结束交互：释放 %d 个会话订阅", len(sessions))
	return nil
}

// CloseConnection 是退出第三阶段：关闭会话流与控制连接。
func (a *App) CloseConnection(ctx context.Context) error {
	a.mu.Lock()
	manager := a.pendingManager
	a.pendingManager = nil
	supervisor := a.supervisor
	if a.noticeOff != nil {
		a.noticeOff()
		a.noticeOff = nil
	}
	a.conn = nil
	a.connected = false
	a.mu.Unlock()

	if manager != nil {
		manager.Close()
	}
	if supervisor != nil {
		supervisor.Close(nil)
	}
	a.record("已关闭连接")
	return nil
}

// ReclaimService 是退出第四阶段：只回收自己拉起的服务实例。
func (a *App) ReclaimService(ctx context.Context) error {
	a.mu.Lock()
	supervisor := a.supervisor
	a.mu.Unlock()
	if supervisor == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := supervisor.Shutdown(ctx)
	switch {
	case err != nil:
		a.record("回收自启服务失败: %v", err)
		return err
	case result.Stopped:
		a.record("已回收自启服务实例")
	default:
		a.record("未回收服务实例: %s", result.Reason)
	}
	return nil
}

// Close 是没有宿主外壳时的收尾顺序：停止新意图 → 结束交互 → 关连接 → 回收自启服务。
// 接上宿主外壳（NewHost）时应由外壳按阶段调用上面四个步骤，以便逐段超时与诊断。
func (a *App) Close(ctx context.Context) error {
	a.mu.Lock()
	alreadyClosed := a.closed
	a.closed = true
	a.mu.Unlock()
	if alreadyClosed {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := a.cfg.CloseTimeout
	steps := []struct {
		name string
		run  func(context.Context) error
	}{
		{"停止新意图", a.StopIntents},
		{"结束交互", a.EndInteraction},
		{"关闭连接", a.CloseConnection},
		{"回收自启服务", a.ReclaimService},
	}
	var errs []error
	for _, step := range steps {
		stepCtx, cancel := context.WithTimeout(ctx, timeout)
		err := step.run(stepCtx)
		cancel()
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", step.name, err))
		}
	}
	a.mu.Lock()
	a.phase = PhaseClosed
	a.mu.Unlock()
	return errors.Join(errs...)
}

// Writable 报告此刻能否登记新意图：未就绪、退出中或外壳闸门拒绝时返回明确错误。
func (a *App) Writable() error {
	a.mu.Lock()
	stopping := a.stopping
	allowed := a.writesAllowed
	gate := a.writeGate
	closed := a.closed
	a.mu.Unlock()
	switch {
	case closed:
		return ErrClosed
	case stopping:
		return ErrExiting
	case !allowed:
		return ErrNotReady
	case gate != nil:
		return gate()
	default:
		return nil
	}
}

// attachControlNotices 在控制连接上登记通知路由。重连后重新登记。
func (a *App) attachControlNotices(conn *daemon.Connection) {
	if conn == nil || conn.Client == nil {
		return
	}
	cancel := conn.Client.OnNotification(a.handleControlNotice)
	a.mu.Lock()
	previous := a.noticeOff
	a.noticeOff = cancel
	a.mu.Unlock()
	if previous != nil {
		previous()
	}
}

// Reconnect 丢弃控制连接并重新握手；自己拉起的实例消失时会重新拉起。
func (a *App) Reconnect(ctx context.Context) error {
	a.mu.Lock()
	supervisor := a.supervisor
	closed := a.closed
	stopping := a.stopping
	a.mu.Unlock()
	switch {
	case closed:
		return ErrClosed
	case stopping:
		return ErrExiting
	case supervisor == nil:
		return ErrNotConnected
	}
	result, err := supervisor.Reconnect(ctx)
	if err != nil {
		return fmt.Errorf("重连服务端: %w", err)
	}
	a.mu.Lock()
	a.conn = result.Connection
	a.connected = true
	// 实例被替换时，旧订阅的序号不再可用；按事实重读交给上层重新打开会话。
	a.mu.Unlock()
	a.attachControlNotices(result.Connection)
	if result.Restarted {
		a.record("服务实例已替换，会话需重新打开后重读")
	} else {
		a.record("控制连接已恢复")
	}
	return nil
}

// control 返回当前控制连接；未连接时为 nil。
func (a *App) control() *protocol.Client {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conn == nil {
		return nil
	}
	return a.conn.Client
}

// connection 返回当前连接（含 hello 结果）；未连接时为 nil。
func (a *App) connection() *daemon.Connection {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.conn
}

// StartupSession 返回订阅阶段打开的会话标识；没有时为「空」。
func (a *App) StartupSession() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.startupSession
}

// Phase 返回当前启动阶段。
func (a *App) Phase() Phase {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.phase
}

// Connected 报告控制连接是否可用；它是宿主外壳的只读连接探针。
func (a *App) Connected() bool { return a.control() != nil }

// Catalog 返回会话目录领域实例；Connect 之前为 nil。
func (a *App) Catalog() *catalog.Catalog {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.catalog
}

// Account 返回账号领域实例；Connect 之前为 nil。
func (a *App) Account() *account.Account {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.account
}

// Settings 返回设置领域实例；Connect 之前为 nil。
func (a *App) Settings() *settings.Manager {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings
}

// Extensions 返回扩展与后台领域实例；Connect 之前为 nil。
func (a *App) Extensions() *extensions.Service {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.extensions
}

// Writes 返回写登记；Connect 之前为 nil。界面只读它，不直接发请求。
func (a *App) Writes() *write.Manager {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.writes
}

// Workspace 返回工作区只读能力；没有配置根目录时为 nil。
func (a *App) Workspace() *workspace.Workspace {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.workspace
}

// Attachments 返回附件暂存区；本地状态尚未装载时为 nil。
func (a *App) Attachments() *attachments.Store {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.staging
}

// Terminals 返回本机终端登记表。
func (a *App) Terminals() *terminal.Registry { return a.terminals }

// Diagnostics 返回这一路的诊断行（有上限，最早的先丢）。它是自由文本，只给人看，
// 不进入宿主外壳的白名单诊断记录。
func (a *App) Diagnostics() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.notes...)
}

func (a *App) record(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	if a.cfg.Logf != nil {
		a.cfg.Logf("%s", line)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	const maxNotes = 256
	a.notes = append(a.notes, line)
	if len(a.notes) > maxNotes {
		a.notes = append([]string(nil), a.notes[len(a.notes)-maxNotes:]...)
	}
}

// uuidV4 生成小写连字符形式的 UUID v4，用作提交去重标识与本地意图标识。
func uuidV4() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	text := hex.EncodeToString(buf[:])
	return text[0:8] + "-" + text[8:12] + "-" + text[12:16] + "-" + text[16:20] + "-" + text[20:32], nil
}
