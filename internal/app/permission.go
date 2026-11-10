package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/interaction"
)

// 权限档位在服务端上的真实方法（alpha.21 实测，见 D1 §3.9）。
const (
	methodPermissionGet    = "session.permission.get"
	methodPermissionUpdate = "session.permission.update"
)

// ErrPermissionConflict 表示权限档位写入的版本前置条件已过期；不覆盖服务端。
var ErrPermissionConflict = errors.New("权限档位版本冲突")

// refreshPermission 打开会话时读一次当前权限策略，把 preset 装进交互裁决。
//
// 读取失败只记诊断：权限读取不属于打开会话的前置条件，缺能力时提交仍按原样
// 进行（不补默认档位）。成功后同时记下版本号，供下一次 update 作前置条件。
func (a *App) refreshPermission(ctx context.Context, runtime *sessionRuntime) {
	branchID := runtime.folder.Selected()
	if branchID == "" {
		branchID = mainBranchID
	}
	raw, err := runtime.lease.Subscription().Call(ctx, methodPermissionGet, protocol.PermissionGetParams{
		SessionID: runtime.id,
		BranchID:  branchID,
	})
	if err != nil {
		a.record("读取权限档位失败: %v", err)
		return
	}
	result, err := protocol.ParsePermissionResult(raw)
	if err != nil {
		a.record("权限回执无法解析: %v", err)
		return
	}
	runtime.permissionVersion.Store(result.Version)
	policy := result.Policy
	runtime.permissionPolicy.Store(&policy)
	a.setPermissionFromPolicy(runtime, result.Policy)
}

// setPermissionFromPolicy 把服务端策略的 preset 原样装进交互裁决。
func (a *App) setPermissionFromPolicy(runtime *sessionRuntime, policy protocol.PermissionPolicy) {
	if policy.Preset == "" {
		return
	}
	if err := runtime.arbiter.SetPermissionTier(interaction.PermissionTier(policy.Preset)); err != nil {
		a.record("权限档位无法装入裁决: %v", err)
		return
	}
	a.record("权限档位=%s: 会话=%s", policy.Preset, runtime.id)
}

// SetPermissionTier 切换权限档位：经 `session.permission.update` 写入服务端，
// 成功后以回执里的策略为准；时间线上的 `permission/changed` 也会更新同一状态。
//
// 写闸门未放行时直接拒绝；服务端没有该方法时按能力缺失处理，不发请求。
func (a *App) SetPermissionTier(ctx context.Context, sessionID string, tier interaction.PermissionTier) error {
	if err := a.Writable(); err != nil {
		return err
	}
	runtime, err := a.openRuntime(sessionID, methodPermissionUpdate)
	if err != nil {
		return err
	}
	branchID := runtime.folder.Selected()
	if branchID == "" {
		branchID = mainBranchID
	}
	raw, err := runtime.lease.Subscription().Call(ctx, methodPermissionUpdate, protocol.PermissionUpdateParams{
		SessionID:       runtime.id,
		BranchID:        branchID,
		ExpectedVersion: runtime.permissionVersion.Load(),
		// 保留已读到的工具列表，只换 preset；服务端要求策略三件套齐全，不接受 null。
		Policy: permissionPolicyFor(runtime, tier),
	})
	if err != nil {
		if appErrorOf(err) == "permission_conflict" {
			return fmt.Errorf("%w: %s", ErrPermissionConflict, methodPermissionUpdate)
		}
		if failure.Is(err, failure.KindNonRetryable) {
			return fmt.Errorf("%w: %s: %v", ErrCapabilityMissing, methodPermissionUpdate, err)
		}
		return err
	}
	result, err := protocol.ParsePermissionResult(raw)
	if err != nil {
		return fmt.Errorf("权限回执无法解析: %w", err)
	}
	runtime.permissionVersion.Store(result.Version)
	policy := result.Policy
	runtime.permissionPolicy.Store(&policy)
	a.setPermissionFromPolicy(runtime, result.Policy)
	return nil
}

// permissionPolicyFor 用最近读到的策略换 preset，保留工具列表。
func permissionPolicyFor(runtime *sessionRuntime, tier interaction.PermissionTier) protocol.PermissionPolicy {
	policy := protocol.PermissionPolicy{
		Preset:          string(tier),
		AllowedTools:    []string{},
		DisallowedTools: []string{},
	}
	current := runtime.permissionPolicy.Load()
	if current == nil {
		return policy
	}
	policy.AllowedTools = append([]string{}, current.AllowedTools...)
	policy.DisallowedTools = append([]string{}, current.DisallowedTools...)
	return policy
}

// PermissionTier 返回本会话当前的权限档位；没有读取或没有值时为 false。
func (a *App) PermissionTier(sessionID string) (interaction.PermissionTier, bool) {
	runtime, ok := a.session(sessionID)
	if !ok {
		return "", false
	}
	return runtime.arbiter.PermissionTier()
}

// routePermissionChanged 把时间线事实 `permission/changed` 驱动到交互裁决。
func (a *App) routePermissionChanged(arbiter *interaction.Arbiter, raw []byte) {
	changed, err := protocol.ParsePermissionChanged(raw)
	if err != nil {
		a.record("权限变更事实无法解析: %v", err)
		return
	}
	if changed.Policy.Preset == "" {
		return
	}
	if err := arbiter.SetPermissionTier(interaction.PermissionTier(changed.Policy.Preset)); err != nil {
		a.record("权限档位无法装入裁决: %v", err)
	}
}
