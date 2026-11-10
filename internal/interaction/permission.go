package interaction

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// PermissionTier 是权限档位的取值。本包只保存并透传，不猜默认值：没有设置过就
// 没有值，不会补一个「全自动」或「工作区自动」。
type PermissionTier string

const (
	// PermissionReadOnly 是只读档：写入与启动进程都需要确认或拒绝。
	PermissionReadOnly PermissionTier = "read-only"
	// PermissionWorkspaceWrite 是工作区可写档：启动进程与工作区外访问需要确认。
	PermissionWorkspaceWrite PermissionTier = "workspace-write"
	// PermissionFullAccess 是全自动档：工作区内外的读写都不再逐条询问。
	PermissionFullAccess PermissionTier = "full-access"
)

// permissionParam 是提交意图里携带档位的字段名。取值原样透传给发送方。
const permissionParam = "permission"

// ErrPermissionTierEmpty 表示档位取值为空。
var ErrPermissionTierEmpty = errors.New("权限档位取值为空")

// ErrPermissionParams 表示提交意图的参数不是 JSON 对象，无法合入档位。
var ErrPermissionParams = errors.New("提交意图参数不是 JSON 对象")

// ErrNotSubmitIntent 表示档位只能透传给提交意图。
var ErrNotSubmitIntent = errors.New("权限档位只能透传给提交意图")

// ErrPermissionSession 表示提交意图不属于本裁决器的会话。
var ErrPermissionSession = errors.New("权限档位只作用于本会话")

// SetPermissionTier 保存本会话的权限档位。取值原样保存，只要求非空；档位是服务端
// 或用户给出的字符串，本包不猜默认值、也不把未知取值改写成本地常量。
//
// 档位归交互裁决，作用域按会话：每个打开的会话有自己的裁决器与自己的档位。
func (a *Arbiter) SetPermissionTier(tier PermissionTier) error {
	if tier == "" {
		return ErrPermissionTierEmpty
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.permissionTier = tier
	a.permissionSet = true
	return nil
}

// ClearPermissionTier 清空本会话的档位，回到「没有值」。
func (a *Arbiter) ClearPermissionTier() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.permissionTier = ""
	a.permissionSet = false
}

// PermissionTier 返回本会话的当前档位；没有取值时第二个返回值为 false。
func (a *Arbiter) PermissionTier() (PermissionTier, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.permissionTier, a.permissionSet
}

// WithPermission 把本会话的当前档位显式透传给一次提交意图。
//
// 约束：只接受提交意图，且意图的会话必须是本裁决器的会话；没有取值时意图原样
// 返回，不补默认值。已有参数必须是 JSON 对象；本包只增加一个字段，不改其它内容。
// 提交路径必须先调用本方法，服务端才能收到所选档位。
func (a *Arbiter) WithPermission(intent write.Intent) (write.Intent, error) {
	if intent.Kind != write.KindSubmitInput {
		return intent, fmt.Errorf("%w: %s", ErrNotSubmitIntent, intent.Kind)
	}
	if intent.SessionID != a.sessionID {
		return intent, fmt.Errorf("%w: 意图会话 %q，本会话 %q",
			ErrPermissionSession, intent.SessionID, a.sessionID)
	}

	a.mu.Lock()
	tier, ok := a.permissionTier, a.permissionSet
	a.mu.Unlock()
	if !ok {
		return intent, nil
	}

	params := make(map[string]json.RawMessage)
	if len(intent.Params) > 0 {
		if err := json.Unmarshal(intent.Params, &params); err != nil {
			return intent, fmt.Errorf("%w: %v", ErrPermissionParams, err)
		}
		if params == nil {
			// JSON null 解码后是 nil map；按空对象处理，避免写入时 panic。
			params = make(map[string]json.RawMessage)
		}
	}
	rawTier, err := json.Marshal(string(tier))
	if err != nil {
		return intent, fmt.Errorf("编码权限档位: %w", err)
	}
	params[permissionParam] = rawTier
	merged, err := json.Marshal(params)
	if err != nil {
		return intent, fmt.Errorf("编码提交意图参数: %w", err)
	}
	intent.Params = merged
	return intent, nil
}
