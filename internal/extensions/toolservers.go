package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// ToolTransport 是外部工具服务器的传输方式。
type ToolTransport string

const (
	ToolTransportStdio ToolTransport = "stdio"
	ToolTransportHTTP  ToolTransport = "http"
	ToolTransportSSE   ToolTransport = "sse"
)

// ToolServerSpec 是新建或修改工具服务器时提交的配置。
//
// Env 只用于写入，不会出现在清单读数里；本包也不记录、不打印它的内容。
type ToolServerSpec struct {
	Name      string            `json:"name"`
	Transport ToolTransport     `json:"transport"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	URL       string            `json:"url,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	// Source 是 mcp.save 的目标作用域（user/project/plugin）；空表示由集成层决定。
	Source Source `json:"source,omitempty"`
	// ExpectedVersion 是 mcp.save/mcp.remove 的版本前置条件；空表示不声明。
	ExpectedVersion string `json:"expectedVersion,omitempty"`
	// Configuration 是 mcp.save 的原始配置文档；非空时集成层按它写入，
	// Name/Transport/Command/URL 只作为领域侧摘要。
	Configuration json.RawMessage `json:"configuration,omitempty"`
}

func (spec ToolServerSpec) validate() error {
	if err := requireText("工具服务器名称", spec.Name); err != nil {
		return err
	}
	switch spec.Transport {
	case ToolTransportStdio:
		return requireText("stdio 工具服务器命令", spec.Command)
	case ToolTransportHTTP, ToolTransportSSE:
		return requireText("工具服务器地址", spec.URL)
	default:
		return invalidRequest("工具服务器传输方式 %q 不支持", spec.Transport)
	}
}

// ToolServer 是一个已登记的外部工具服务器。清单不返回环境变量内容，只给键名。
type ToolServer struct {
	ID        string
	Name      string
	Source    Source
	Transport ToolTransport
	Command   string
	Args      []string
	URL       string
	Enabled   bool
	Connected bool
	EnvKeys   []string
	ToolCount int
	// AuthStatus 与 AuthMissingScopes 是 mcp.catalog 给出的鉴权状态。
	AuthStatus        string
	AuthMissingScopes []string
}

// ToolServerConfiguration 生成 alpha.21 `mcp.save` 的配置文档。
//
// stdio 不能带 url，http/sse 不能带 command；空字段必须省略，否则服务端返回
// mcp_invalid_config。载荷里已有原始 Configuration 时原样返回。
func ToolServerConfiguration(spec ToolServerSpec) (json.RawMessage, error) {
	if len(spec.Configuration) > 0 {
		return append(json.RawMessage(nil), spec.Configuration...), nil
	}
	document := map[string]any{"transport": string(spec.Transport)}
	if spec.Command != "" {
		document["command"] = spec.Command
	}
	if spec.URL != "" {
		document["url"] = spec.URL
	}
	if len(spec.Args) > 0 {
		document["args"] = append([]string(nil), spec.Args...)
	}
	if len(spec.Env) > 0 {
		env := make(map[string]string, len(spec.Env))
		for key, value := range spec.Env {
			env[key] = value
		}
		document["env"] = env
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("编码工具服务器配置: %w", err)
	}
	return raw, nil
}

// MCPTarget 定位一个 MCP 条目；Source 为空时按 user 处理。
type MCPTarget struct {
	ID     string `json:"id"`
	Source Source `json:"source"`
	Cwd    string `json:"cwd,omitempty"`
}

func (t MCPTarget) normalize() (MCPTarget, error) {
	if err := requireText("MCP 条目标识", t.ID); err != nil {
		return MCPTarget{}, err
	}
	if t.Source == "" {
		t.Source = SourceUser
	}
	return t, nil
}

// MCPAuthStartPayload 是 mcp.auth.start 意图的载荷。
type MCPAuthStartPayload struct {
	ID           string `json:"id"`
	Source       Source `json:"source"`
	Cwd          string `json:"cwd,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
}

// MCPAuthCompletePayload 是 mcp.auth.complete 意图的载荷。
type MCPAuthCompletePayload struct {
	AttemptID   string `json:"attemptId"`
	CallbackURL string `json:"callbackUrl"`
}

// MCPAuthCancelPayload 是 mcp.auth.cancel 意图的载荷。
type MCPAuthCancelPayload struct {
	AttemptID string `json:"attemptId"`
}

// MCPAuthLogoutPayload 是 mcp.auth.logout 意图的载荷。
type MCPAuthLogoutPayload struct {
	ID     string `json:"id"`
	Source Source `json:"source"`
	Cwd    string `json:"cwd,omitempty"`
}

// MCPAuthStarted 是 mcp.auth.start 的成功回执；AuthorizationURL 含一次性授权
// 参数，调用方不得写入日志。
type MCPAuthStarted struct {
	AttemptID        string `json:"attemptId"`
	AuthorizationURL string `json:"authorizationUrl"`
	RedirectURI      string `json:"redirectUri"`
	Resource         string `json:"resource"`
	Issuer           string `json:"issuer"`
	ExpiresAt        int64  `json:"expiresAt"`
}

// MCPAuthLogoutResult 是 mcp.auth.logout 的回执。
type MCPAuthLogoutResult struct {
	Revoked bool `json:"revoked"`
}

// StartMCPAuth 发起一次 MCP 认证。没有该能力时按 hello 门禁返回能力缺失且
// 零请求；真实认证服务器不可用时服务端返回 mcp_auth_* 错误。
func (s *Service) StartMCPAuth(ctx context.Context, target MCPTarget, clientSecret string) (write.IntentView, error) {
	target, err := target.normalize()
	if err != nil {
		return write.IntentView{}, err
	}
	if err := s.requireFeature("mcp.auth.start"); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpMCPAuthStart, MCPAuthStartPayload{
		ID:           target.ID,
		Source:       target.Source,
		Cwd:          target.Cwd,
		ClientSecret: clientSecret,
	}, true)
}

// CompleteMCPAuth 用回调 URL 完成一次认证。
func (s *Service) CompleteMCPAuth(ctx context.Context, attemptID, callbackURL string) (write.IntentView, error) {
	if err := requireText("MCP 认证尝试标识", attemptID); err != nil {
		return write.IntentView{}, err
	}
	if err := requireText("MCP 认证回调地址", callbackURL); err != nil {
		return write.IntentView{}, err
	}
	if err := s.requireFeature("mcp.auth.complete"); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpMCPAuthComplete, MCPAuthCompletePayload{
		AttemptID:   attemptID,
		CallbackURL: callbackURL,
	}, true)
}

// CancelMCPAuth 取消一次认证尝试。
func (s *Service) CancelMCPAuth(ctx context.Context, attemptID string) (write.IntentView, error) {
	if err := requireText("MCP 认证尝试标识", attemptID); err != nil {
		return write.IntentView{}, err
	}
	if err := s.requireFeature("mcp.auth.cancel"); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpMCPAuthCancel, MCPAuthCancelPayload{AttemptID: attemptID}, true)
}

// LogoutMCPAuth 清除一个 MCP 条目的凭据。
func (s *Service) LogoutMCPAuth(ctx context.Context, target MCPTarget) (write.IntentView, error) {
	target, err := target.normalize()
	if err != nil {
		return write.IntentView{}, err
	}
	if err := s.requireFeature("mcp.auth.logout"); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpMCPAuthLogout, MCPAuthLogoutPayload{
		ID:     target.ID,
		Source: target.Source,
		Cwd:    target.Cwd,
	}, true)
}

// ToolDescriptor 是探测得到的工具摘要。
type ToolDescriptor struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ToolServerProbe 是一次显式探测的结果。它只是预览，不修改服务器配置。
type ToolServerProbe struct {
	ServerID  string `json:"serverId"`
	Reachable bool   `json:"reachable"`
	// Status 与 LatencyMillis 是 alpha.21 mcp.test 回执里的状态与耗时。
	Status        string           `json:"status,omitempty"`
	LatencyMillis int64            `json:"latencyMs,omitempty"`
	Tools         []ToolDescriptor `json:"tools,omitempty"`
	Reason        string           `json:"reason,omitempty"`
	ProbedAt      time.Time        `json:"probedAt"`
}

// ImportCandidate 是配置导入预览里的一条候选。
type ImportCandidate struct {
	Key      string         `json:"key"`
	Server   ToolServerSpec `json:"server"`
	Existing bool           `json:"existing,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
}

// ImportPreview 是一次导入预览。它不写任何配置。
type ImportPreview struct {
	Source     string            `json:"source"`
	Candidates []ImportCandidate `json:"candidates"`
	ReadAt     time.Time         `json:"readAt"`
}

// ToolServerReader 读取工具服务器清单、显式探测与导入预览。三者都不写配置。
type ToolServerReader interface {
	ReadToolServers(ctx context.Context) ([]ToolServer, error)
	ProbeToolServer(ctx context.Context, serverID string) (ToolServerProbe, error)
	PreviewToolServerImport(ctx context.Context, source string) (ImportPreview, error)
}

// MCPImportSourceState 是 mcp.import.sources 的一个来源状态。
type MCPImportSourceState string

const (
	MCPImportSourceReady      MCPImportSourceState = "ready"
	MCPImportSourceAbsent     MCPImportSourceState = "absent"
	MCPImportSourceUnreadable MCPImportSourceState = "unreadable"
	MCPImportSourceTooLarge   MCPImportSourceState = "too-large"
	MCPImportSourceMalformed  MCPImportSourceState = "malformed"
)

// MCPImportSource 是服务端扫描到的另一个客户端的配置来源。
type MCPImportSource struct {
	ID      string               `json:"id"`
	Client  string               `json:"client"`
	Scope   string               `json:"scope"`
	Path    string               `json:"path"`
	State   MCPImportSourceState `json:"state"`
	Entries *int                 `json:"entries,omitempty"`
	Error   string               `json:"error,omitempty"`
}

// MCPImportStatus 是预览里一条会被怎么处理。
type MCPImportStatus string

const (
	MCPImportIdentical MCPImportStatus = "identical"
	MCPImportChanged   MCPImportStatus = "changed"
	MCPImportConflict  MCPImportStatus = "conflict"
	MCPImportInvalid   MCPImportStatus = "invalid"
)

// MCPImportField 是写入会改变的字段名与值类型；不含值。
type MCPImportField struct {
	Field string `json:"field"`
	Kind  string `json:"kind"`
}

// MCPImportUnsupported 是解码器不执行的成员及其处理方式。
type MCPImportUnsupported struct {
	Field  string `json:"field"`
	Effect string `json:"effect"`
}

// MCPImportEntry 是预览里的一个条目。
type MCPImportEntry struct {
	ID          string                 `json:"id"`
	Transport   string                 `json:"transport,omitempty"`
	Status      MCPImportStatus        `json:"status"`
	Changes     []MCPImportField       `json:"changes,omitempty"`
	Unsupported []MCPImportUnsupported `json:"unsupported,omitempty"`
	Omitted     int                    `json:"omitted,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// MCPImportStore 是预览读取的已知来源身份。
type MCPImportStore struct {
	ID     string `json:"id"`
	Client string `json:"client"`
	Scope  string `json:"scope"`
	Path   string `json:"path"`
}

// MCPImportPreview 是 mcp.import.preview 的归一化回执。
type MCPImportPreview struct {
	Revision        string
	SourceRevision  string
	Store           *MCPImportStore
	Entries         []MCPImportEntry
	UnknownTopLevel []string
	ReadAt          time.Time
}

// MCPImportPreviewRequest 是预览参数。Source 是写入目标作用域（user/project）；
// From 是 mcp.import.sources 的来源 id；Document 是客户端自带 JSON，二者只能给一个。
type MCPImportPreviewRequest struct {
	Cwd      string `json:"cwd,omitempty"`
	Source   Source `json:"source"`
	From     string `json:"from,omitempty"`
	Document string `json:"document,omitempty"`
}

func (r MCPImportPreviewRequest) validate() error {
	if r.Source != SourceUser && r.Source != SourceProject {
		return invalidRequest("MCP 导入目标作用域 %q 不支持", r.Source)
	}
	if (r.From == "") == (r.Document == "") {
		return invalidRequest("MCP 导入来源必须且只能给 from 或 document")
	}
	return nil
}

// MCPImportAction 是 resolutions 里一条动作。
type MCPImportAction string

const (
	MCPImportActionImport  MCPImportAction = "import"
	MCPImportActionSkip    MCPImportAction = "skip"
	MCPImportActionReplace MCPImportAction = "replace"
	MCPImportActionRename  MCPImportAction = "rename"
)

// MCPImportResolution 是要对一条预览条目执行的动作。
type MCPImportResolution struct {
	ID     string          `json:"id"`
	Action MCPImportAction `json:"action"`
	NewID  string          `json:"newId,omitempty"`
}

func (r MCPImportResolution) validate() error {
	if err := requireText("MCP 导入条目标识", r.ID); err != nil {
		return err
	}
	switch r.Action {
	case MCPImportActionImport, MCPImportActionSkip, MCPImportActionReplace:
		if r.NewID != "" {
			return invalidRequest("动作 %s 不接受 newId", r.Action)
		}
	case MCPImportActionRename:
		if err := requireText("MCP 导入新标识", r.NewID); err != nil {
			return err
		}
	default:
		return invalidRequest("MCP 导入动作 %q 不支持", r.Action)
	}
	return nil
}

// MCPImportApplyRequest 是 mcp.import.apply 的参数。
type MCPImportApplyRequest struct {
	Cwd                    string                `json:"cwd,omitempty"`
	Source                 Source                `json:"source"`
	From                   string                `json:"from,omitempty"`
	Document               string                `json:"document,omitempty"`
	ExpectedVersion        string                `json:"expectedVersion"`
	ExpectedSourceRevision string                `json:"expectedSourceRevision,omitempty"`
	Resolutions            []MCPImportResolution `json:"resolutions"`
}

func (r MCPImportApplyRequest) validate() error {
	if r.Source != SourceUser && r.Source != SourceProject {
		return invalidRequest("MCP 导入目标作用域 %q 不支持", r.Source)
	}
	if (r.From == "") == (r.Document == "") {
		return invalidRequest("MCP 导入来源必须且只能给 from 或 document")
	}
	if err := requireText("MCP 导入目标版本", r.ExpectedVersion); err != nil {
		return err
	}
	if r.From != "" && r.ExpectedSourceRevision == "" {
		return invalidRequest("从来源导入时必须给 expectedSourceRevision")
	}
	if r.From == "" && r.ExpectedSourceRevision != "" {
		return invalidRequest("expectedSourceRevision 只用于从来源导入")
	}
	if len(r.Resolutions) == 0 {
		return invalidRequest("MCP 导入 resolutions 不能为空")
	}
	for _, resolution := range r.Resolutions {
		if err := resolution.validate(); err != nil {
			return err
		}
	}
	return nil
}

// MCPImportApplyPayload 是 mcp.import.apply 写意图的载荷。
type MCPImportApplyPayload struct {
	Cwd                    string                `json:"cwd,omitempty"`
	Source                 Source                `json:"source"`
	From                   string                `json:"from,omitempty"`
	Document               string                `json:"document,omitempty"`
	ExpectedVersion        string                `json:"expectedVersion"`
	ExpectedSourceRevision string                `json:"expectedSourceRevision,omitempty"`
	Resolutions            []MCPImportResolution `json:"resolutions"`
}

// MCPImportApplyResult 是应用结果；Revision 是 apply 回执里的新目标版本。
type MCPImportApplyResult struct {
	Revision string
	Intent   write.IntentView
}

// MCPImportReader 读取已知配置来源与导入预览。只读；应用走写登记。
type MCPImportReader interface {
	ReadMCPImportSources(ctx context.Context, cwd string) ([]MCPImportSource, error)
	PreviewMCPImport(ctx context.Context, request MCPImportPreviewRequest) (MCPImportPreview, error)
}

// 导入失败分类。
var (
	// ErrImportConflict 表示目标或来源在预览之后已变化，未写入。
	ErrImportConflict = errors.New("MCP 导入目标已变化")
)

// ToolServerAddPayload 是 mcp.save 意图的载荷（新增）。ImportSource 与 CandidateKey
// 非空时表示这次新增来自一次导入预览。
type ToolServerAddPayload struct {
	Server       ToolServerSpec `json:"server"`
	ImportSource string         `json:"importSource,omitempty"`
	CandidateKey string         `json:"candidateKey,omitempty"`
}

// ToolServerUpdatePayload 是 mcp.save 意图的载荷（修改）。
type ToolServerUpdatePayload struct {
	ServerID string         `json:"serverId"`
	Server   ToolServerSpec `json:"server"`
}

// ToolServerRemovePayload 是 mcp.remove 意图的载荷。ExpectedVersion 是目录
// 版本前置条件；空表示由集成层按最近一次读取补齐。
type ToolServerRemovePayload struct {
	ServerID        string `json:"serverId"`
	ExpectedVersion string `json:"expectedVersion,omitempty"`
}

// ImportResult 是一次按勾选写入的结果。Written 是明确成功的条数。
type ImportResult struct {
	Source   string             `json:"source"`
	Selected []string           `json:"selected"`
	Views    []write.IntentView `json:"views,omitempty"`
	Written  int                `json:"written"`
	Revision string             `json:"revision,omitempty"`
}

// ListToolServers 读取工具服务器清单并更新镜像读数。读取清单不探测。
func (s *Service) ListToolServers(ctx context.Context) (Snapshot[ToolServer], error) {
	if s.cfg.ToolServers == nil {
		return missingReader(s, "mcp.catalog", &s.toolServers)
	}
	return readInto(s, ctx, "mcp.catalog", &s.toolServers, s.cfg.ToolServers.ReadToolServers)
}

// ToolServerSnapshot 返回最近一次工具服务器读数，不发起请求。
func (s *Service) ToolServerSnapshot() Snapshot[ToolServer] {
	return snapshotOf(s, &s.toolServers)
}

// AddToolServer 新增一个外部工具服务器。写入经写登记。
func (s *Service) AddToolServer(ctx context.Context, spec ToolServerSpec) (write.IntentView, error) {
	if err := spec.validate(); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpToolServerAdd, ToolServerAddPayload{Server: spec}, false)
}

// UpdateToolServer 修改一个外部工具服务器。写入经写登记。
func (s *Service) UpdateToolServer(ctx context.Context, serverID string, spec ToolServerSpec) (write.IntentView, error) {
	if err := requireText("工具服务器标识", serverID); err != nil {
		return write.IntentView{}, err
	}
	if err := spec.validate(); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpToolServerAdd, ToolServerUpdatePayload{ServerID: serverID, Server: spec}, false)
}

// RemoveToolServer 删除一个外部工具服务器。写入经写登记。
func (s *Service) RemoveToolServer(ctx context.Context, serverID string) (write.IntentView, error) {
	return s.RemoveToolServerWithVersion(ctx, serverID, "")
}

// RemoveToolServerWithVersion 带 mcp.catalog 的 userRevision 删除服务器；
// 版本已变化时服务端返回冲突，写意图明确失败且不清除当前配置。
func (s *Service) RemoveToolServerWithVersion(ctx context.Context, serverID, expectedVersion string) (write.IntentView, error) {
	if err := requireText("工具服务器标识", serverID); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpToolServerRemove, ToolServerRemovePayload{
		ServerID:        serverID,
		ExpectedVersion: expectedVersion,
	}, false)
}

// ProbeToolServer 显式探测一个工具服务器。探测只在调用时发生，清单读取与增删改
// 都不会触发它；结果只是预览，不写回配置。
func (s *Service) ProbeToolServer(ctx context.Context, serverID string) (ToolServerProbe, error) {
	if err := requireText("工具服务器标识", serverID); err != nil {
		return ToolServerProbe{}, err
	}
	if s.cfg.ToolServers == nil {
		return ToolServerProbe{}, s.wrapReadError("mcp.test", ErrCapabilityMissing)
	}
	s.readMu.Lock()
	defer s.readMu.Unlock()
	if s.isClosed() {
		return ToolServerProbe{}, ErrClosed
	}
	probe, err := s.cfg.ToolServers.ProbeToolServer(ctx, serverID)
	if err != nil {
		return ToolServerProbe{}, s.wrapReadError("mcp.test", err)
	}
	if probe.ProbedAt.IsZero() {
		probe.ProbedAt = s.cfg.Now()
	}
	probe.ServerID = serverID
	probe.Tools = append([]ToolDescriptor(nil), probe.Tools...)
	s.mu.Lock()
	s.probes[serverID] = probe
	s.mu.Unlock()
	return probe, nil
}

// ProbeSnapshot 返回某个工具服务器最近一次探测结果，不重新探测。
func (s *Service) ProbeSnapshot(serverID string) (ToolServerProbe, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	probe, ok := s.probes[serverID]
	if !ok {
		return ToolServerProbe{}, false
	}
	probe.Tools = append([]ToolDescriptor(nil), probe.Tools...)
	return probe, true
}

// ListMCPImportSources 读取服务端已知的客户端配置来源；只读，不返回值。
func (s *Service) ListMCPImportSources(ctx context.Context, cwd string) ([]MCPImportSource, error) {
	if err := s.requireFeature("mcp.import.sources"); err != nil {
		return nil, err
	}
	if s.cfg.MCPImports == nil {
		return nil, s.wrapReadError("mcp.import.sources", ErrCapabilityMissing)
	}
	s.readMu.Lock()
	defer s.readMu.Unlock()
	if s.isClosed() {
		return nil, ErrClosed
	}
	sources, err := s.cfg.MCPImports.ReadMCPImportSources(ctx, cwd)
	if err != nil {
		return nil, s.wrapReadError("mcp.import.sources", err)
	}
	return append([]MCPImportSource(nil), sources...), nil
}

// PreviewMCPImport 读取一次真实形状的导入预览并缓存，供 ApplyMCPImport 与旧包装
// 使用。它不写任何配置。
func (s *Service) PreviewMCPImport(ctx context.Context, request MCPImportPreviewRequest) (MCPImportPreview, error) {
	if err := request.validate(); err != nil {
		return MCPImportPreview{}, err
	}
	if err := s.requireFeature("mcp.import.preview"); err != nil {
		return MCPImportPreview{}, err
	}
	if s.cfg.MCPImports == nil {
		return MCPImportPreview{}, s.wrapReadError("mcp.import.preview", ErrCapabilityMissing)
	}
	s.readMu.Lock()
	defer s.readMu.Unlock()
	if s.isClosed() {
		return MCPImportPreview{}, ErrClosed
	}
	preview, err := s.cfg.MCPImports.PreviewMCPImport(ctx, request)
	if err != nil {
		return MCPImportPreview{}, s.wrapReadError("mcp.import.preview", err)
	}
	if preview.ReadAt.IsZero() {
		preview.ReadAt = s.cfg.Now()
	}
	preview = cloneMCPImportPreview(preview)
	s.mu.Lock()
	if s.mcpPreviews == nil {
		s.mcpPreviews = make(map[string]MCPImportPreview)
	}
	s.mcpPreviews[mcpImportKey(request)] = preview
	s.mu.Unlock()
	return cloneMCPImportPreview(preview), nil
}

// ApplyMCPImport 按 resolutions 逐条应用导入。写入经写登记只发一次；目标或来源
// 在预览后变化时返回 ErrImportConflict，不重写。
func (s *Service) ApplyMCPImport(ctx context.Context, request MCPImportApplyRequest) (MCPImportApplyResult, error) {
	if err := request.validate(); err != nil {
		return MCPImportApplyResult{}, err
	}
	if err := s.requireFeature("mcp.import.apply"); err != nil {
		return MCPImportApplyResult{}, err
	}
	view, err := s.write(ctx, OpMCPImportApply, MCPImportApplyPayload{
		Cwd:                    request.Cwd,
		Source:                 request.Source,
		From:                   request.From,
		Document:               request.Document,
		ExpectedVersion:        request.ExpectedVersion,
		ExpectedSourceRevision: request.ExpectedSourceRevision,
		Resolutions:            append([]MCPImportResolution(nil), request.Resolutions...),
	}, true)
	result := MCPImportApplyResult{Intent: view}
	if err != nil {
		return result, err
	}
	if view.Status == write.StatusFailed && strings.Contains(view.Reason, "mcp_settings_conflict") {
		return result, ErrImportConflict
	}
	if view.Status != write.StatusSucceeded {
		return result, ErrWriteNotSucceeded
	}
	if len(view.Result) == 0 {
		return result, fmt.Errorf("%w: MCP 导入回执为空", ErrInvalidRequest)
	}
	var reply struct {
		Revision string `json:"revision"`
	}
	if err := json.Unmarshal(view.Result, &reply); err != nil {
		return result, fmt.Errorf("解析 MCP 导入回执: %w", err)
	}
	if reply.Revision == "" {
		return result, fmt.Errorf("%w: MCP 导入回执缺少 revision", ErrInvalidRequest)
	}
	result.Revision = reply.Revision
	return result, nil
}

func mcpImportKey(request MCPImportPreviewRequest) string {
	raw, _ := json.Marshal(request)
	sum := sha256.Sum256(raw)
	return "extensions:mcpimport:" + hex.EncodeToString(sum[:8])
}

func cloneMCPImportPreview(preview MCPImportPreview) MCPImportPreview {
	cloned := preview
	if preview.Store != nil {
		store := *preview.Store
		cloned.Store = &store
	}
	cloned.Entries = make([]MCPImportEntry, len(preview.Entries))
	for index, entry := range preview.Entries {
		entry.Changes = append([]MCPImportField(nil), entry.Changes...)
		entry.Unsupported = append([]MCPImportUnsupported(nil), entry.Unsupported...)
		cloned.Entries[index] = entry
	}
	cloned.UnknownTopLevel = append([]string(nil), preview.UnknownTopLevel...)
	return cloned
}

// PreviewToolServerImport 是旧便捷方法：内部转成真实 mcp.import.preview 请求
// （目标作用域 user、from=source），并把真实预览转回扁平候选供旧调用方使用。
func (s *Service) PreviewToolServerImport(ctx context.Context, source string) (ImportPreview, error) {
	if err := requireText("导入来源", source); err != nil {
		return ImportPreview{}, err
	}
	if s.cfg.MCPImports == nil {
		return ImportPreview{}, s.wrapReadError("mcp.import.preview", ErrCapabilityMissing)
	}
	preview, err := s.PreviewMCPImport(ctx, MCPImportPreviewRequest{
		Source: SourceUser,
		From:   source,
	})
	if err != nil {
		return ImportPreview{}, err
	}
	legacy := legacyImportPreview(source, preview)
	s.mu.Lock()
	s.imports[source] = legacy
	s.mu.Unlock()
	return cloneImportPreview(legacy), nil
}

// ImportPreviewSnapshot 返回某个来源最近一次旧格式预览，不重读配置。
func (s *Service) ImportPreviewSnapshot(source string) (ImportPreview, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	preview, ok := s.imports[source]
	if !ok {
		return ImportPreview{}, false
	}
	return cloneImportPreview(preview), true
}

// ApplyToolServerImport 是旧便捷方法：内部转成真实 mcp.import.apply，
// resolutions 只包含勾选条目；不再逐条写 tool_server.add。
func (s *Service) ApplyToolServerImport(ctx context.Context, source string, keys []string) (ImportResult, error) {
	if err := requireText("导入来源", source); err != nil {
		return ImportResult{}, err
	}
	if len(keys) == 0 {
		return ImportResult{}, ErrEmptySelection
	}
	selected := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if err := requireText("导入候选键", key); err != nil {
			return ImportResult{}, err
		}
		if _, duplicate := selected[key]; duplicate {
			return ImportResult{}, invalidRequest("导入候选键重复: %q", key)
		}
		selected[key] = struct{}{}
	}

	request := MCPImportPreviewRequest{Source: SourceUser, From: source}
	s.mu.Lock()
	preview, ok := s.mcpPreviews[mcpImportKey(request)]
	s.mu.Unlock()
	if !ok {
		return ImportResult{}, ErrPreviewRequired
	}
	known := make(map[string]struct{}, len(preview.Entries))
	for _, entry := range preview.Entries {
		known[entry.ID] = struct{}{}
	}
	for key := range selected {
		if _, ok := known[key]; !ok {
			return ImportResult{}, fmt.Errorf("%w: %q", ErrUnknownCandidate, key)
		}
	}
	resolutions := make([]MCPImportResolution, 0, len(selected))
	result := ImportResult{Source: source, Selected: make([]string, 0, len(selected))}
	for _, entry := range preview.Entries {
		if _, wanted := selected[entry.ID]; !wanted {
			continue
		}
		result.Selected = append(result.Selected, entry.ID)
		resolutions = append(resolutions, MCPImportResolution{ID: entry.ID, Action: MCPImportActionImport})
	}

	applied, err := s.ApplyMCPImport(ctx, MCPImportApplyRequest{
		Source:                 SourceUser,
		From:                   source,
		ExpectedVersion:        preview.Revision,
		ExpectedSourceRevision: preview.SourceRevision,
		Resolutions:            resolutions,
	})
	result.Revision = applied.Revision
	if applied.Intent.ID != "" {
		result.Views = []write.IntentView{applied.Intent}
	}
	if err != nil {
		return result, err
	}
	result.Written = len(resolutions)
	return result, nil
}

func legacyImportPreview(source string, preview MCPImportPreview) ImportPreview {
	legacy := ImportPreview{
		Source:     source,
		Candidates: make([]ImportCandidate, 0, len(preview.Entries)),
		ReadAt:     preview.ReadAt,
	}
	for _, entry := range preview.Entries {
		candidate := ImportCandidate{
			Key:      entry.ID,
			Server:   ToolServerSpec{Name: entry.ID, Transport: ToolTransport(entry.Transport)},
			Existing: entry.Status == MCPImportIdentical,
		}
		for _, unsupported := range entry.Unsupported {
			candidate.Warnings = append(candidate.Warnings, unsupported.Field+":"+unsupported.Effect)
		}
		if entry.Error != "" {
			candidate.Warnings = append(candidate.Warnings, entry.Error)
		}
		legacy.Candidates = append(legacy.Candidates, candidate)
	}
	return legacy
}

func validateImportPreview(preview ImportPreview) error {
	seen := make(map[string]struct{}, len(preview.Candidates))
	for index, candidate := range preview.Candidates {
		if strings.TrimSpace(candidate.Key) == "" {
			return invalidRequest("导入预览第 %d 条候选缺少键", index+1)
		}
		if _, duplicate := seen[candidate.Key]; duplicate {
			return invalidRequest("导入预览候选键重复: %q", candidate.Key)
		}
		seen[candidate.Key] = struct{}{}
		if err := candidate.Server.validate(); err != nil {
			return fmt.Errorf("导入预览候选 %q: %w", candidate.Key, err)
		}
	}
	return nil
}

func cloneImportPreview(preview ImportPreview) ImportPreview {
	cloned := ImportPreview{Source: preview.Source, ReadAt: preview.ReadAt}
	cloned.Candidates = make([]ImportCandidate, len(preview.Candidates))
	for index, candidate := range preview.Candidates {
		candidate.Server.Args = append([]string(nil), candidate.Server.Args...)
		candidate.Warnings = append([]string(nil), candidate.Warnings...)
		cloned.Candidates[index] = candidate
	}
	return cloned
}
