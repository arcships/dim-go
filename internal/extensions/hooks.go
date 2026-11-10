package extensions

import (
	"context"

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
}

// Writable 表示该钩子是否允许由用户写入。来源未知按只读处理。
func (h Hook) Writable() bool { return h.Source == SourceUser }

// HookReader 读取钩子清单。只读。
type HookReader interface {
	ReadHooks(ctx context.Context) ([]Hook, error)
}

// HookWritePayload 是 hooks.save 意图的载荷。ExpectedRevision 是目录版本前置
// 条件；空表示由集成层按最近一次读取补齐。
type HookWritePayload struct {
	Hook             Hook   `json:"hook"`
	ExpectedRevision string `json:"expectedRevision,omitempty"`
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
// 返回 ErrSourceNotWritable。
func (s *Service) WriteHook(ctx context.Context, hook Hook) (write.IntentView, error) {
	if hook.Source != SourceUser {
		return write.IntentView{}, ErrSourceNotWritable
	}
	if err := requireText("钩子事件", hook.Event); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpHookWrite, HookWritePayload{Hook: hook}, false)
}
