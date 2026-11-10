package extensions

import (
	"context"
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
	Transport ToolTransport
	Command   string
	Args      []string
	URL       string
	Enabled   bool
	Connected bool
	EnvKeys   []string
	ToolCount int
}

// ToolDescriptor 是探测得到的工具摘要。
type ToolDescriptor struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ToolServerProbe 是一次显式探测的结果。它只是预览，不修改服务器配置。
type ToolServerProbe struct {
	ServerID  string           `json:"serverId"`
	Reachable bool             `json:"reachable"`
	Tools     []ToolDescriptor `json:"tools,omitempty"`
	Reason    string           `json:"reason,omitempty"`
	ProbedAt  time.Time        `json:"probedAt"`
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

// ToolServerAddPayload 是 tool_server.add 意图的载荷。ImportSource 与 CandidateKey
// 非空时表示这次新增来自一次导入预览。
type ToolServerAddPayload struct {
	Server       ToolServerSpec `json:"server"`
	ImportSource string         `json:"importSource,omitempty"`
	CandidateKey string         `json:"candidateKey,omitempty"`
}

// ToolServerUpdatePayload 是 tool_server.update 意图的载荷。
type ToolServerUpdatePayload struct {
	ServerID string         `json:"serverId"`
	Server   ToolServerSpec `json:"server"`
}

// ToolServerRemovePayload 是 tool_server.remove 意图的载荷。
type ToolServerRemovePayload struct {
	ServerID string `json:"serverId"`
}

// ImportResult 是一次按勾选写入的结果。Written 是明确成功的条数。
type ImportResult struct {
	Source   string             `json:"source"`
	Selected []string           `json:"selected"`
	Views    []write.IntentView `json:"views,omitempty"`
	Written  int                `json:"written"`
}

// ListToolServers 读取工具服务器清单并更新镜像读数。读取清单不探测。
func (s *Service) ListToolServers(ctx context.Context) (Snapshot[ToolServer], error) {
	if s.cfg.ToolServers == nil {
		return missingReader(s, "tool_server.list", &s.toolServers)
	}
	return readInto(s, ctx, "tool_server.list", &s.toolServers, s.cfg.ToolServers.ReadToolServers)
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
	return s.write(ctx, OpToolServerUpdate, ToolServerUpdatePayload{ServerID: serverID, Server: spec}, false)
}

// RemoveToolServer 删除一个外部工具服务器。写入经写登记。
func (s *Service) RemoveToolServer(ctx context.Context, serverID string) (write.IntentView, error) {
	if err := requireText("工具服务器标识", serverID); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpToolServerRemove, ToolServerRemovePayload{ServerID: serverID}, false)
}

// ProbeToolServer 显式探测一个工具服务器。探测只在调用时发生，清单读取与增删改
// 都不会触发它；结果只是预览，不写回配置。
func (s *Service) ProbeToolServer(ctx context.Context, serverID string) (ToolServerProbe, error) {
	if err := requireText("工具服务器标识", serverID); err != nil {
		return ToolServerProbe{}, err
	}
	if s.cfg.ToolServers == nil {
		return ToolServerProbe{}, s.wrapReadError("tool_server.probe", ErrCapabilityMissing)
	}
	s.readMu.Lock()
	defer s.readMu.Unlock()
	if s.isClosed() {
		return ToolServerProbe{}, ErrClosed
	}
	probe, err := s.cfg.ToolServers.ProbeToolServer(ctx, serverID)
	if err != nil {
		return ToolServerProbe{}, s.wrapReadError("tool_server.probe", err)
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

// PreviewToolServerImport 读取一份配置里的候选并返回预览。它不写任何配置；
// 预览结果按来源保存，供 ApplyToolServerImport 校验勾选。
func (s *Service) PreviewToolServerImport(ctx context.Context, source string) (ImportPreview, error) {
	if err := requireText("导入来源", source); err != nil {
		return ImportPreview{}, err
	}
	if s.cfg.ToolServers == nil {
		return ImportPreview{}, s.wrapReadError("tool_server.import_preview", ErrCapabilityMissing)
	}
	s.readMu.Lock()
	defer s.readMu.Unlock()
	if s.isClosed() {
		return ImportPreview{}, ErrClosed
	}
	preview, err := s.cfg.ToolServers.PreviewToolServerImport(ctx, source)
	if err != nil {
		return ImportPreview{}, s.wrapReadError("tool_server.import_preview", err)
	}
	preview.Source = source
	if preview.ReadAt.IsZero() {
		preview.ReadAt = s.cfg.Now()
	}
	if err := validateImportPreview(preview); err != nil {
		return ImportPreview{}, err
	}
	stored := ImportPreview{
		Source:     preview.Source,
		Candidates: make([]ImportCandidate, len(preview.Candidates)),
		ReadAt:     preview.ReadAt,
	}
	for index, candidate := range preview.Candidates {
		candidate.Server.Args = append([]string(nil), candidate.Server.Args...)
		candidate.Warnings = append([]string(nil), candidate.Warnings...)
		stored.Candidates[index] = candidate
	}
	s.mu.Lock()
	s.imports[source] = stored
	s.mu.Unlock()
	return cloneImportPreview(stored), nil
}

// ImportPreviewSnapshot 返回某个来源最近一次预览，不重读配置。
func (s *Service) ImportPreviewSnapshot(source string) (ImportPreview, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	preview, ok := s.imports[source]
	if !ok {
		return ImportPreview{}, false
	}
	return cloneImportPreview(preview), true
}

// ApplyToolServerImport 按勾选逐条写入导入候选。
//
// 先校验全部勾选键都在最近一次预览里、没有重复；任一不合法都不写入任何条目。
// 每条候选登记一条管理写入，发送顺序与预览顺序一致。任一条没有明确成功即停止，
// 返回已登记的结果与 ErrWriteNotSucceeded。
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

	s.mu.Lock()
	preview, ok := s.imports[source]
	s.mu.Unlock()
	if !ok {
		return ImportResult{}, ErrPreviewRequired
	}
	known := make(map[string]ImportCandidate, len(preview.Candidates))
	for _, candidate := range preview.Candidates {
		known[candidate.Key] = candidate
	}
	for key := range selected {
		if _, ok := known[key]; !ok {
			return ImportResult{}, fmt.Errorf("%w: %q", ErrUnknownCandidate, key)
		}
	}

	result := ImportResult{Source: source}
	for _, candidate := range preview.Candidates {
		if _, wanted := selected[candidate.Key]; !wanted {
			continue
		}
		result.Selected = append(result.Selected, candidate.Key)
		view, err := s.write(ctx, OpToolServerAdd, ToolServerAddPayload{
			Server:       candidate.Server,
			ImportSource: source,
			CandidateKey: candidate.Key,
		}, false)
		result.Views = append(result.Views, view)
		if err != nil {
			return result, err
		}
		if view.Status != write.StatusSucceeded {
			return result, ErrWriteNotSucceeded
		}
		result.Written++
	}
	return result, nil
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
