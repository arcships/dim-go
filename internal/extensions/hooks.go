package extensions

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// Hook 是一条钩子定义。来源决定可写范围：只有用户来源可写；项目来源与插件来源
// 由服务端或插件自身维护，本端只读。
type Hook struct {
	ID      string
	Name    string
	Event   string
	Command string
	Source  Source
	Enabled bool
	// Matcher/Type/Timeout/OnFailure/Async/State 是 alpha.21 hooks.catalog 的
	// 扁平字段；写入时只使用 Matcher、Type、Command 与可选的超时/失败策略。
	Matcher   string
	Type      string
	Timeout   int
	OnFailure string
	Async     bool
	State     string
}

// Writable 表示该钩子是否允许由用户写入。来源未知按只读处理。
func (h Hook) Writable() bool { return h.Source == SourceUser }

// HookReader 读取钩子清单。只读。
type HookReader interface {
	ReadHooks(ctx context.Context) ([]Hook, error)
}

// HookSourceState 是 hooks.catalog 的一个来源状态。
type HookSourceState struct {
	Source  Source
	Path    string
	Enabled bool
	State   string
}

// ProjectHookCatalog 是按项目目录读取的钩子目录；ProjectsRevision 是项目
// 启停写入的版本前置条件。
type ProjectHookCatalog struct {
	Hooks            []Hook
	Sources          []HookSourceState
	ProjectsRevision string
	ReadAt           time.Time
}

// ProjectHookReader 按项目目录读取钩子目录。只读。
type ProjectHookReader interface {
	ReadProjectHooks(ctx context.Context, cwd string) (ProjectHookCatalog, error)
}

// HooksProjectEnabledPayload 是 hooks.set_enabled 意图的载荷。
type HooksProjectEnabledPayload struct {
	Cwd              string `json:"cwd"`
	Enabled          bool   `json:"enabled"`
	ExpectedRevision string `json:"expectedRevision"`
}

// HookWritePayload 是 hooks.save 意图的载荷。ExpectedRevision 是目录版本前置
// 条件；Document 是按 alpha.21 实测形状生成的 hooks.save 文档。
type HookWritePayload struct {
	Hook             Hook            `json:"hook"`
	ExpectedRevision string          `json:"expectedRevision,omitempty"`
	Document         json.RawMessage `json:"document,omitempty"`
}

// HookDocument 把一条用户钩子转成 alpha.21 `hooks.save` 接受的文档形状：
// `{"hooks": {<event>: [{"matcher": <string|null>, "hooks": [{...}]}]}}`。
func HookDocument(hook Hook) (json.RawMessage, error) {
	if hook.Event == "" {
		return nil, invalidRequest("钩子事件不能为空")
	}
	hookType := hook.Type
	if hookType == "" {
		hookType = "command"
	}
	inner := map[string]any{"type": hookType, "command": hook.Command}
	if hook.Timeout > 0 {
		inner["timeout"] = hook.Timeout
	}
	if hook.OnFailure != "" {
		inner["onFailure"] = hook.OnFailure
	}
	if hook.Async {
		inner["async"] = true
	}
	entry := map[string]any{"matcher": nil, "hooks": []any{inner}}
	if hook.Matcher != "" {
		entry["matcher"] = hook.Matcher
	}
	document := map[string]any{
		"hooks": map[string]any{hook.Event: []any{entry}},
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("编码钩子文档: %w", err)
	}
	return raw, nil
}

// ListHooks 读取钩子清单并更新镜像读数。
func (s *Service) ListHooks(ctx context.Context) (Snapshot[Hook], error) {
	if s.cfg.Hooks == nil {
		return missingReader(s, "hooks.catalog", &s.hooks)
	}
	return readInto(s, ctx, "hooks.catalog", &s.hooks, s.cfg.Hooks.ReadHooks)
}

// HookSnapshot 返回最近一次钩子读数，不发起请求。
func (s *Service) HookSnapshot() Snapshot[Hook] {
	return snapshotOf(s, &s.hooks)
}

// WriteHook 写入一条用户来源的钩子。非用户来源在发出任何写请求前拒绝，
// 返回 ErrSourceNotWritable。调用方没有持有目录版本时可用这个入口；服务端可能
// 以版本冲突拒绝，冲突会体现为写意图明确失败。
func (s *Service) WriteHook(ctx context.Context, hook Hook) (write.IntentView, error) {
	return s.WriteHookWithRevision(ctx, hook, "")
}

// WriteHookWithRevision 带 hooks.catalog 的 userRevision 写入用户来源钩子；
// 版本已变化时服务端返回 hooks_conflict，写意图明确失败且不覆盖当前文档。
func (s *Service) WriteHookWithRevision(ctx context.Context, hook Hook, expectedRevision string) (write.IntentView, error) {
	if hook.Source != SourceUser {
		return write.IntentView{}, ErrSourceNotWritable
	}
	if err := requireText("钩子事件", hook.Event); err != nil {
		return write.IntentView{}, err
	}
	document, err := HookDocument(hook)
	if err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpHookWrite, HookWritePayload{
		Hook:             hook,
		ExpectedRevision: expectedRevision,
		Document:         document,
	}, false)
}

// ReadProjectHooks 按项目目录读取钩子目录与来源状态；它不发写请求。
func (s *Service) ReadProjectHooks(ctx context.Context, cwd string) (ProjectHookCatalog, error) {
	if err := requireText("项目目录", cwd); err != nil {
		return ProjectHookCatalog{}, err
	}
	if err := s.requireFeature("hooks.catalog"); err != nil {
		return ProjectHookCatalog{}, err
	}
	if s.cfg.ProjectHooks == nil {
		return ProjectHookCatalog{}, s.wrapReadError("hooks.catalog", ErrCapabilityMissing)
	}
	s.readMu.Lock()
	defer s.readMu.Unlock()
	if s.isClosed() {
		return ProjectHookCatalog{}, ErrClosed
	}
	catalog, err := s.cfg.ProjectHooks.ReadProjectHooks(ctx, cwd)
	if err != nil {
		return ProjectHookCatalog{}, s.wrapReadError("hooks.catalog", err)
	}
	catalog.Hooks = append([]Hook(nil), catalog.Hooks...)
	catalog.Sources = append([]HookSourceState(nil), catalog.Sources...)
	if catalog.ReadAt.IsZero() {
		catalog.ReadAt = s.cfg.Now()
	}
	return catalog, nil
}

// SetProjectHooksEnabled 写入项目作用域的钩子总开关；版本冲突由服务端返回，
// 写意图明确失败且不覆盖当前状态。
func (s *Service) SetProjectHooksEnabled(
	ctx context.Context,
	cwd string,
	enabled bool,
	expectedRevision string,
) (write.IntentView, error) {
	if err := requireText("项目目录", cwd); err != nil {
		return write.IntentView{}, err
	}
	if err := s.requireFeature("hooks.set_enabled"); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpHookSetEnabled, HooksProjectEnabledPayload{
		Cwd:              cwd,
		Enabled:          enabled,
		ExpectedRevision: expectedRevision,
	}, true)
}
