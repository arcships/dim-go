// Package extensions 是扩展与后台领域（D11b）：插件、技能、钩子、外部工具服务器、
// 计划任务、工作流与会话级唤醒的管理面读取与写入。
//
// 边界：
//   - 管理面只持有服务端读取确认过的镜像读数；每个读数只有本包一个驱动者，事实只
//     来自读取与服务端变更通知。缺能力时报能力缺失，不用空列表或默认值冒充。
//   - 插件、技能、钩子、工具服务器、工作流的变更与唤醒取消都登记成写意图，经写
//     登记（D3）发送一次并收敛；本包不自行发送、不自行重试，也不做乐观更新。
//   - 计划任务与唤醒按服务端事件重读一次；本包没有任何定时器，不做全量轮询。
//   - 钩子只有用户来源可写；项目来源与插件来源只读，写入口在发出任何请求前拒绝。
//   - 工具服务器探测与配置导入只产出预览；导入只在调用方勾选后逐条写入。
//
// 不包含：界面、连接实现、协议 wire 定义、会话折叠、设置与模型、账号。
package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// Source 是扩展能力的来源。取值与服务端声明一致；未知来源按只读处理。
type Source string

const (
	// SourceLocal 是本地路径来源。
	SourceLocal Source = "local"
	// SourceGit 是 Git 来源。
	SourceGit Source = "git"
	// SourceBundled 是随包可选插件或内置技能来源。
	SourceBundled Source = "bundled"
	// SourceUser 是用户来源；只有这个来源的钩子可写。
	SourceUser Source = "user"
	// SourceProject 是项目来源；钩子只读。
	SourceProject Source = "project"
	// SourcePlugin 是插件来源；钩子只读。
	SourcePlugin Source = "plugin"
	// SourceUnknown 表示服务端给出的来源无法识别；按只读处理。
	SourceUnknown Source = "unknown"
)

// installSourceKinds 是安装来源允许的取值。
func (s Source) installSourceKind() bool {
	switch s {
	case SourceLocal, SourceGit, SourceBundled:
		return true
	default:
		return false
	}
}

// 领域错误。调用方按 errors.Is 判断，不匹配错误文本。
var (
	// ErrCapabilityMissing 表示服务端或本端缺少该管理面能力。
	ErrCapabilityMissing = errors.New("管理面能力缺失")
	// ErrWriterUnavailable 表示没有可用的写登记。
	ErrWriterUnavailable = errors.New("写登记不可用")
	// ErrInvalidRequest 表示请求结构不合法，请求没有发出。
	ErrInvalidRequest = errors.New("管理面请求不合法")
	// ErrSourceNotWritable 表示该来源不允许写入；请求没有发出。
	ErrSourceNotWritable = errors.New("该来源不允许写入")
	// ErrPreviewRequired 表示应用导入前必须先预览该来源。
	ErrPreviewRequired = errors.New("应用导入前必须先预览")
	// ErrUnknownCandidate 表示勾选的导入候选不在最近一次预览里。
	ErrUnknownCandidate = errors.New("勾选的导入候选不存在")
	// ErrEmptySelection 表示没有勾选任何导入候选。
	ErrEmptySelection = errors.New("没有勾选任何导入候选")
	// ErrWriteNotSucceeded 表示写意图进入终态但没有成功。
	ErrWriteNotSucceeded = errors.New("管理面写入未成功")
	// ErrClosed 表示领域实例已关闭。
	ErrClosed = errors.New("扩展与后台领域已关闭")
)

// CapabilityError 是一条带能力名的能力缺失错误。它同时满足
// errors.Is(err, ErrCapabilityMissing) 与按能力名的匹配。
type CapabilityError struct {
	// Feature 是缺失的能力名，例如 "plugin.list"。
	Feature string
	// Err 是底层错误；本端没有读取方时为空。
	Err error
}

func (e *CapabilityError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Feature, ErrCapabilityMissing)
	}
	return fmt.Sprintf("%s: %s: %v", e.Feature, ErrCapabilityMissing, e.Err)
}

func (e *CapabilityError) Unwrap() error { return e.Err }

func (e *CapabilityError) Is(target error) bool {
	if target == ErrCapabilityMissing {
		return true
	}
	other, ok := target.(*CapabilityError)
	return ok && other.Feature == e.Feature
}

// Snapshot 是一项管理面读数的只读视图。
//
// Items 是最近一次成功读取确认的事实；读取失败时保留上一次确认值，不先清空。
// ReadError 与 CapabilityMissing 描述最近一次读取的结果，供界面区分「空」与「读不到」。
type Snapshot[T any] struct {
	Items             []T
	ReadAt            time.Time
	ReadError         string
	CapabilityMissing bool
}

// Writer 是写登记的窄接口；*write.Manager 直接满足它。
type Writer interface {
	Register(intent write.Intent) (write.IntentView, error)
	Wait(ctx context.Context, intentID string) (write.IntentView, error)
}

var _ Writer = (*write.Manager)(nil)

// Config 是扩展与后台领域的配置。读取方为 nil 表示本端没有该能力；写登记为 nil
// 时所有写入口返回 ErrWriterUnavailable。
type Config struct {
	Plugins     PluginReader
	Skills      SkillReader
	Hooks       HookReader
	ToolServers ToolServerReader
	Scheduler   SchedulerReader
	Workflows   WorkflowReader
	Wakeups     WakeupReader
	Writer      Writer
	// Now 可在测试中替换；nil 时使用系统时间。
	Now func() time.Time
	// Logf 记录能力缺失与事件跟随重读；不得写入凭据、参数或环境变量内容。
	Logf func(format string, args ...any)
}

func (c Config) withDefaults() Config {
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.Logf == nil {
		c.Logf = func(string, ...any) {}
	}
	return c
}

// Service 持有管理面读数与写意图键。所有导出方法可并发调用。
type Service struct {
	cfg Config

	// readMu 串行化全部管理面读取，保证旧读数不会覆盖后完成的读数。
	readMu sync.Mutex

	mu            sync.Mutex
	closed        bool
	plugins       Snapshot[Plugin]
	skills        Snapshot[Skill]
	hooks         Snapshot[Hook]
	toolServers   Snapshot[ToolServer]
	probes        map[string]ToolServerProbe
	imports       map[string]ImportPreview
	scheduled     Snapshot[ScheduledTask]
	workflows     Snapshot[WorkflowDefinition]
	workflowRuns  map[string]*Snapshot[WorkflowRun]
	workflowWatch map[string]struct{}
	wakeups       map[string]*Snapshot[Wakeup]
	wakeupWatch   map[string]WakeupScope
	followStats   FollowStats

	// writeMu 保护同一操作的重试序号与最近状态，防止重复写入。
	writeMu sync.Mutex
	writes  map[string]writeAttempt

	follower *follower
}

type writeAttempt struct {
	key     string
	attempt int
	status  write.Status
	seen    bool
}

// New 创建扩展与后台领域实例，并启动唯一的事件跟随协程。调用方负责 Close。
func New(cfg Config) (*Service, error) {
	cfg = cfg.withDefaults()
	service := &Service{
		cfg:           cfg,
		probes:        make(map[string]ToolServerProbe),
		imports:       make(map[string]ImportPreview),
		workflowRuns:  make(map[string]*Snapshot[WorkflowRun]),
		workflowWatch: make(map[string]struct{}),
		wakeups:       make(map[string]*Snapshot[Wakeup]),
		wakeupWatch:   make(map[string]WakeupScope),
		writes:        make(map[string]writeAttempt),
	}
	service.follower = newFollower(service.handleNotice)
	return service, nil
}

// Close 停止事件跟随。关闭后读取与写入都返回 ErrClosed。
func (s *Service) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()
	s.follower.close()
}

func (s *Service) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// wrapReadError 把能力缺失归一成 CapabilityError；其他错误原样返回。
//
// 识别两类能力缺失：本端没有读取方（ErrCapabilityMissing）与服务端稳定的
// appError（unknown_method 等）。不把参数错误、超时或连接失败冒充成能力缺失。
func (s *Service) wrapReadError(feature string, err error) error {
	if err == nil {
		return nil
	}
	var capability *CapabilityError
	if errors.As(err, &capability) {
		if capability.Feature == feature {
			return err
		}
		return &CapabilityError{Feature: feature, Err: err}
	}
	if errors.Is(err, ErrCapabilityMissing) {
		s.cfg.Logf("extensions: 能力缺失 feature=%s reason=no_reader", feature)
		return &CapabilityError{Feature: feature, Err: err}
	}
	var carrier interface{ AppError() string }
	if errors.As(err, &carrier) && isCapabilityAppError(carrier.AppError()) {
		s.cfg.Logf("extensions: 能力缺失 feature=%s appError=%s", feature, carrier.AppError())
		return &CapabilityError{Feature: feature, Err: err}
	}
	return err
}

// isCapabilityAppError 判断服务端错误是否明确表示该方法不存在。
func isCapabilityAppError(appError string) bool {
	switch appError {
	case "unknown_method", "method_not_found", "capability_missing", "not_supported":
		return true
	default:
		return false
	}
}

// readInto 执行一次管理面读取并更新目标读数。
//
// 成功时整体替换事实；失败时保留上一次确认事实，只更新错误与能力缺失标记。
// 全部读取共用 readMu，保证「旧状态不覆盖新状态」。
func readInto[T any](
	s *Service,
	ctx context.Context,
	feature string,
	target *Snapshot[T],
	read func(context.Context) ([]T, error),
) (Snapshot[T], error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.readMu.Lock()
	defer s.readMu.Unlock()
	if s.isClosed() {
		return Snapshot[T]{}, ErrClosed
	}
	items, err := read(ctx)
	now := s.cfg.Now()
	if err != nil {
		wrapped := s.wrapReadError(feature, err)
		s.mu.Lock()
		target.ReadError = wrapped.Error()
		target.CapabilityMissing = errors.Is(wrapped, ErrCapabilityMissing)
		snapshot := cloneSnapshot(*target)
		s.mu.Unlock()
		return snapshot, wrapped
	}
	s.mu.Lock()
	target.Items = append([]T(nil), items...)
	target.ReadAt = now
	target.ReadError = ""
	target.CapabilityMissing = false
	snapshot := cloneSnapshot(*target)
	s.mu.Unlock()
	return snapshot, nil
}

// missingReader 在配置里没有读取方时更新读数并返回能力缺失。
func missingReader[T any](s *Service, feature string, target *Snapshot[T]) (Snapshot[T], error) {
	err := s.wrapReadError(feature, ErrCapabilityMissing)
	s.mu.Lock()
	target.ReadError = err.Error()
	target.CapabilityMissing = true
	snapshot := cloneSnapshot(*target)
	s.mu.Unlock()
	return snapshot, err
}

func cloneSnapshot[T any](snapshot Snapshot[T]) Snapshot[T] {
	snapshot.Items = append([]T(nil), snapshot.Items...)
	return snapshot
}

func snapshotOf[T any](s *Service, target *Snapshot[T]) Snapshot[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneSnapshot(*target)
}

// WriteOp 是管理写入意图的操作名。它写在写登记参数里，由集成层的发送方映射到
// 具体 wire 方法；本包不定义 wire 方法名。
type WriteOp string

// WriteOp 是写登记参数的判别名，不是 wire 方法名；集成层按它映射到 alpha.21
// 实测方法。这样既保持写意图稳定，也避免在领域包里复制 wire 方法表。
//
// 与实测方法的对应见 D11b §4.3；其中插件启停由 payload.Enabled 决定落到
// plugin.enable 或 plugin.disable，MCP 导入落到 mcp.import.apply。
const (
	OpPluginInstall    WriteOp = "plugin.install"
	OpPluginSetEnabled WriteOp = "plugin.set_enabled"
	OpPluginUpdate     WriteOp = "plugin.update"
	OpSkillInstall     WriteOp = "skill.install"
	OpSkillSetEnabled  WriteOp = "skill.set_enabled"
	OpHookWrite        WriteOp = "hook.write"
	OpToolServerAdd    WriteOp = "tool_server.add"
	OpToolServerUpdate WriteOp = "tool_server.update"
	OpToolServerRemove WriteOp = "tool_server.remove"
	OpWorkflowStart    WriteOp = "workflow.start"
	OpWorkflowStop     WriteOp = "workflow.stop"
	OpWakeupCancel     WriteOp = "wakeup.cancel"

	// 以下只在解码时识别，随后归一到上面的判别名；集成层可以先按真实方法名
	// 传参，本包仍以稳定判别名交给发送方。
	opWireSkillsInstall    WriteOp = "skills.install"
	opWireSkillsSetEnabled WriteOp = "skills.set_enabled"
	opWireHooksSave        WriteOp = "hooks.save"
	opWireMCPSave          WriteOp = "mcp.save"
	opWireMCPRemove        WriteOp = "mcp.remove"
	opWireMCPImportApply   WriteOp = "mcp.import.apply"
	opWirePluginEnable     WriteOp = "plugin.enable"
	opWirePluginDisable    WriteOp = "plugin.disable"
	opWireWakeupCancel     WriteOp = "session.wakeup.cancel"
)

func (op WriteOp) valid() bool {
	switch op {
	case OpPluginInstall, OpPluginSetEnabled, OpPluginUpdate,
		OpSkillInstall, OpSkillSetEnabled, OpHookWrite,
		OpToolServerAdd, OpToolServerUpdate, OpToolServerRemove,
		OpWorkflowStart, OpWorkflowStop, OpWakeupCancel,
		opWireSkillsInstall, opWireSkillsSetEnabled, opWireHooksSave,
		opWireMCPSave, opWireMCPRemove, opWireMCPImportApply,
		opWirePluginEnable, opWirePluginDisable, opWireWakeupCancel:
		return true
	default:
		return false
	}
}

// ManagementWrite 是写登记参数里的形状：操作名加该操作自己的载荷。
// 集成层的发送方用 DecodeManagementWrite 解析。
type ManagementWrite struct {
	Op      WriteOp         `json:"op"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ManagementParams 把操作名与载荷编码成写意图参数。
func ManagementParams(op WriteOp, payload any) (json.RawMessage, error) {
	if !op.valid() {
		return nil, fmt.Errorf("%w: 未知管理写入操作 %q", ErrInvalidRequest, op)
	}
	managed := ManagementWrite{Op: op}
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("编码管理写入载荷: %w", err)
		}
		managed.Payload = raw
	}
	raw, err := json.Marshal(managed)
	if err != nil {
		return nil, fmt.Errorf("编码管理写入参数: %w", err)
	}
	return raw, nil
}

// DecodeManagementWrite 解析写登记参数，供集成层按操作名分发。
//
// 真实方法名（skills.*、hooks.save、mcp.*、session.wakeup.*）会被归一到本包
// 稳定的判别名，载荷同时改写为发送方认识的结构；旧判别名原样通过。
func DecodeManagementWrite(params json.RawMessage) (ManagementWrite, error) {
	var managed ManagementWrite
	if err := json.Unmarshal(params, &managed); err != nil {
		return ManagementWrite{}, fmt.Errorf("%w: 管理写入参数无法解析: %v", ErrInvalidRequest, err)
	}
	if !managed.Op.valid() {
		return ManagementWrite{}, fmt.Errorf("%w: 未知管理写入操作 %q", ErrInvalidRequest, managed.Op)
	}
	return normalizeManagementWrite(managed)
}

// normalizeManagementWrite 把真实方法名与载荷归一成发送方认识的判别名。
func normalizeManagementWrite(managed ManagementWrite) (ManagementWrite, error) {
	switch managed.Op {
	case opWireSkillsInstall:
		managed.Op = OpSkillInstall
	case opWireSkillsSetEnabled:
		managed.Op = OpSkillSetEnabled
	case opWireHooksSave:
		managed.Op = OpHookWrite
	case opWireMCPSave:
		managed.Op = OpToolServerAdd
	case opWireMCPRemove:
		managed.Op = OpToolServerRemove
	case opWireWakeupCancel:
		managed.Op = OpWakeupCancel
	case opWirePluginEnable, opWirePluginDisable:
		var payload struct {
			PluginID string `json:"pluginId"`
			Name     string `json:"name"`
			Enabled  *bool  `json:"enabled"`
		}
		if len(managed.Payload) > 0 {
			if err := json.Unmarshal(managed.Payload, &payload); err != nil {
				return ManagementWrite{}, fmt.Errorf("%w: 插件启停载荷无法解析: %v", ErrInvalidRequest, err)
			}
		}
		pluginID := payload.PluginID
		if pluginID == "" {
			pluginID = payload.Name
		}
		if pluginID == "" {
			return ManagementWrite{}, invalidRequest("插件启停缺少插件标识")
		}
		enabled := managed.Op == opWirePluginEnable
		raw, err := json.Marshal(PluginEnabledPayload{PluginID: pluginID, Enabled: enabled})
		if err != nil {
			return ManagementWrite{}, fmt.Errorf("编码插件启停载荷: %w", err)
		}
		managed.Op = OpPluginSetEnabled
		managed.Payload = raw
	case opWireMCPImportApply:
		var payload struct {
			Source string   `json:"source"`
			IDs    []string `json:"ids"`
		}
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return ManagementWrite{}, fmt.Errorf("%w: MCP 导入载荷无法解析: %v", ErrInvalidRequest, err)
		}
		if payload.Source == "" || len(payload.IDs) != 1 || payload.IDs[0] == "" {
			return ManagementWrite{}, invalidRequest("MCP 导入需要来源与单个候选标识")
		}
		raw, err := json.Marshal(ToolServerAddPayload{
			ImportSource: payload.Source,
			CandidateKey: payload.IDs[0],
		})
		if err != nil {
			return ManagementWrite{}, fmt.Errorf("编码 MCP 导入载荷: %w", err)
		}
		managed.Op = OpToolServerAdd
		managed.Payload = raw
	}
	return managed, nil
}

// managementKey 用操作名与载荷摘要生成幂等键。相同操作与相同内容重复登记不会
// 产生第二次请求。
func managementKey(op WriteOp, payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("编码管理写入摘要: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "extensions:" + string(op) + ":" + hex.EncodeToString(sum[:8]), nil
}

// nextWriteKey 给出本次要用的幂等键。
//
// 同一操作与内容只有一条意图：未结项或已成功时沿用原键。上一次明确失败或确认
// 未发生时换一个键，允许用户重试；命令型操作在成功后再次调用也换键，允许重复
// 启动或重新检查。
func (s *Service) nextWriteKey(base string, repeatOnSuccess bool) string {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	state := s.writes[base]
	switch {
	case !state.seen:
		state.key = base
	case state.status == write.StatusFailed || state.status == write.StatusNotOccurred:
		state.attempt++
		state.key = fmt.Sprintf("%s#retry-%d", base, state.attempt)
	case state.status == write.StatusSucceeded && repeatOnSuccess:
		state.attempt++
		state.key = fmt.Sprintf("%s#again-%d", base, state.attempt)
	}
	state.seen = true
	s.writes[base] = state
	return state.key
}

// finishWrite 记录一条写意图的终态，供下一次决定是否沿用键。
func (s *Service) finishWrite(base string, view write.IntentView, err error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	state := s.writes[base]
	if view.ID == "" {
		// 登记本身失败，没有产生意图；下次仍用同一个基础键。
		delete(s.writes, base)
		return
	}
	state.key = view.Key
	state.status = view.Status
	state.seen = true
	s.writes[base] = state
}

// write 经写登记发出一次管理写入并等待终态。本包不自行重试。
func (s *Service) write(
	ctx context.Context,
	op WriteOp,
	payload any,
	repeatOnSuccess bool,
) (write.IntentView, error) {
	if s.cfg.Writer == nil {
		return write.IntentView{}, ErrWriterUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return write.IntentView{}, err
	}
	if s.isClosed() {
		return write.IntentView{}, ErrClosed
	}
	base, err := managementKey(op, payload)
	if err != nil {
		return write.IntentView{}, err
	}
	params, err := ManagementParams(op, payload)
	if err != nil {
		return write.IntentView{}, err
	}
	key := s.nextWriteKey(base, repeatOnSuccess)
	view, err := s.cfg.Writer.Register(write.Intent{
		Key:    key,
		Kind:   write.KindManagementWrite,
		Params: params,
	})
	if err != nil {
		s.finishWrite(base, write.IntentView{}, err)
		return view, err
	}
	view, err = s.cfg.Writer.Wait(ctx, view.ID)
	s.finishWrite(base, view, err)
	return view, err
}

// invalidRequest 构造结构不合法错误。
func invalidRequest(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidRequest, fmt.Sprintf(format, args...))
}

// requireText 校验必填文本，并拒绝 NUL。
func requireText(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return invalidRequest("%s 不能为空", field)
	}
	if strings.ContainsRune(value, 0) {
		return invalidRequest("%s 含非法空字符", field)
	}
	return nil
}
