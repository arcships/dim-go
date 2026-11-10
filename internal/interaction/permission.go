package interaction

import (
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

// ErrPermissionTierEmpty 表示档位取值为空。
var ErrPermissionTierEmpty = errors.New("权限档位取值为空")

// ErrPermissionUnset 表示本会话还没有选择权限档位；提交必须显式要求一个档位。
var ErrPermissionUnset = errors.New("本会话尚未选择权限档位")

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

// WithPermission 校验一次提交意图是否可以带上本会话的权限档位。
//
// 协议事实（alpha.21）：`session.submit` 没有 permission 字段，权限由会话分支的
// policy 携带，切换走 `session.permission.update`。因此本方法不向提交参数塞字段，
// 只要求本会话已经明确选择过档位：
//   - 没有取值时拒绝提交（ErrPermissionUnset），不猜默认值；
//   - 只接受提交意图（ErrNotSubmitIntent），且会话必须是本裁决器的会话
//     （ErrPermissionSession）。
//
// 提交路径必须先调用本方法；服务端按该会话已有的 policy 执行。
func (a *Arbiter) WithPermission(intent write.Intent) (write.Intent, error) {
	if intent.Kind != write.KindSubmitInput {
		return intent, fmt.Errorf("%w: %s", ErrNotSubmitIntent, intent.Kind)
	}
	if intent.SessionID != a.sessionID {
		return intent, fmt.Errorf("%w: 意图会话 %q，本会话 %q",
			ErrPermissionSession, intent.SessionID, a.sessionID)
	}
	a.mu.Lock()
	ok := a.permissionSet
	a.mu.Unlock()
	if !ok {
		return intent, fmt.Errorf("%w: %s", ErrPermissionUnset, a.sessionID)
	}
	return intent, nil
}
