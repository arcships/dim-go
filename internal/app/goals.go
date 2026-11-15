package app

import (
	"context"
	"fmt"
	"strings"
)

// GoalMethod 是目标方法的动作名；服务端注册的是 `session.goal.<动作>`。
type GoalMethod string

const (
	GoalSet    GoalMethod = "set"
	GoalEdit   GoalMethod = "edit"
	GoalPause  GoalMethod = "pause"
	GoalResume GoalMethod = "resume"
	GoalClear  GoalMethod = "clear"
	GoalGet    GoalMethod = "get"
)

// 目标方法由会话贡献（插件注册），先按 hello 能力与会话方法表判定；缺方法时
// 报能力缺失、不发请求。

// ReadGoal 读取当前目标（session.goal.get）。
func (a *App) ReadGoal(ctx context.Context, sessionID string) ([]byte, error) {
	return a.callGoal(ctx, sessionID, GoalGet, "", nil)
}

// SetGoal 新建或覆盖目标（session.goal.set）。
func (a *App) SetGoal(ctx context.Context, sessionID, objective string) error {
	_, err := a.callGoal(ctx, sessionID, GoalSet, objective, nil)
	return err
}

// EditGoal 修改目标文本（session.goal.edit）。
func (a *App) EditGoal(ctx context.Context, sessionID, objective string) error {
	_, err := a.callGoal(ctx, sessionID, GoalEdit, objective, nil)
	return err
}

// PauseGoal 暂停目标（session.goal.pause）。
func (a *App) PauseGoal(ctx context.Context, sessionID string) error {
	_, err := a.callGoal(ctx, sessionID, GoalPause, "", nil)
	return err
}

// ResumeGoal 恢复目标（session.goal.resume）。
func (a *App) ResumeGoal(ctx context.Context, sessionID string) error {
	_, err := a.callGoal(ctx, sessionID, GoalResume, "", nil)
	return err
}

// ClearGoal 清除目标（session.goal.clear）。
func (a *App) ClearGoal(ctx context.Context, sessionID string) error {
	_, err := a.callGoal(ctx, sessionID, GoalClear, "", nil)
	return err
}

// callGoal 是目标方法的唯一调用点：拼参数、过能力门禁、发一次会话调用。
func (a *App) callGoal(
	ctx context.Context,
	sessionID string,
	method GoalMethod,
	objective string,
	extra map[string]any,
) ([]byte, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("%w: 缺少会话标识", ErrInvalidConfig)
	}
	name := "session.goal." + string(method)
	runtime, err := a.openRuntime(sessionID, name)
	if err != nil {
		return nil, err
	}
	if !a.sessionSupportsMethod(ctx, runtime, name) {
		return nil, capabilityMissing(name)
	}
	branchID := runtime.folder.Selected()
	if branchID == "" {
		branchID = mainBranchID
	}
	params := map[string]any{"sessionId": sessionID, "branchId": branchID}
	if method == GoalSet || method == GoalEdit {
		if strings.TrimSpace(objective) == "" {
			return nil, fmt.Errorf("%w: 目标内容为空", ErrInvalidConfig)
		}
		params["objective"] = objective
	}
	for key, value := range extra {
		params[key] = value
	}
	raw, err := runtime.lease.Subscription().Call(ctx, name, params)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// StopExecTask 停止一条后台任务（session.exec.stop，参数按 StopExec 的
// sessionId / branchId / taskId）。
func (a *App) StopExecTask(ctx context.Context, sessionID, taskID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("%w: 缺少会话标识", ErrInvalidConfig)
	}
	if strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("%w: 缺少任务标识", ErrInvalidConfig)
	}
	const name = "session.exec.stop"
	runtime, err := a.openRuntime(sessionID, name)
	if err != nil {
		return err
	}
	if !a.sessionSupportsMethod(ctx, runtime, name) {
		return capabilityMissing(name)
	}
	branchID := runtime.folder.Selected()
	if branchID == "" {
		branchID = mainBranchID
	}
	_, err = runtime.lease.Subscription().Call(ctx, name, map[string]any{
		"sessionId": sessionID,
		"branchId":  branchID,
		"taskId":    taskID,
	})
	return err
}
