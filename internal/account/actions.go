package account

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// 账号动作的操作名。它放在写意图的透明参数里，由集成层的发送方映射到 wire 方法。
const (
	// actionLogout 是登出。
	actionLogout = "logout"
	// actionReset 是用量重置。
	actionReset = "reset"
)

const (
	logoutKeyPrefix = "account:logout:"
	resetKeyPrefix  = "account:reset:"
)

// Logout 通过写登记发出一次登出意图，并等待它收敛。
//
// 只有写登记明确成功才清空本地账号状态（状态回到未知，身份与用量清空）；失败、
// 未知或需人工确认时本地状态不变。已经发出过且未明确失败的登出不会重复登记，
// 调用方最多等到同一条意图的终态。
func (a *Account) Logout(ctx context.Context) (write.IntentView, error) {
	if a.cfg.Actions == nil {
		return write.IntentView{}, ErrActionsUnavailable
	}
	a.logoutMu.Lock()
	defer a.logoutMu.Unlock()
	return a.runAction(ctx, actionLogout, logoutKeyPrefix, &a.logoutAttempt, &a.logoutView, true)
}

// Reset 通过写登记发出一次重置意图，并等待它收敛。结果未知时不重复提交；只有
// 明确失败或确认未发生，下一次显式重置才会登记新的意图。
func (a *Account) Reset(ctx context.Context) (write.IntentView, error) {
	if a.cfg.Actions == nil {
		return write.IntentView{}, ErrActionsUnavailable
	}
	a.resetMu.Lock()
	defer a.resetMu.Unlock()
	return a.runAction(ctx, actionReset, resetKeyPrefix, &a.resetAttempt, &a.resetView, false)
}

// runAction 收敛一条账号动作意图。attempt 与 stored 由调用方持有并在锁内访问。
func (a *Account) runAction(
	ctx context.Context,
	op string,
	keyPrefix string,
	attempt *int,
	stored **write.IntentView,
	clearOnSuccess bool,
) (write.IntentView, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	a.mu.Lock()
	previous := *stored
	a.mu.Unlock()

	// 已经发出的动作：只在明确失败或确认未发生时允许重提；其余情况等同一条意图。
	if previous != nil && !actionRetryable(previous.Status) {
		view, err := a.cfg.Actions.Wait(ctx, previous.ID)
		a.storeAction(stored, view)
		if err == nil && view.Status == write.StatusSucceeded && clearOnSuccess {
			a.clearConfirmedLogout()
		}
		return view, err
	}

	a.mu.Lock()
	*attempt = *attempt + 1
	attemptNumber := *attempt
	a.mu.Unlock()

	params, err := json.Marshal(map[string]string{"op": op})
	if err != nil {
		return write.IntentView{}, err
	}
	view, err := a.cfg.Actions.Register(write.Intent{
		Key:    keyPrefix + strconv.Itoa(attemptNumber),
		Kind:   write.KindAccountAction,
		Params: params,
	})
	if err != nil {
		return view, err
	}
	view, err = a.cfg.Actions.Wait(ctx, view.ID)
	a.storeAction(stored, view)
	if err == nil && view.Status == write.StatusSucceeded && clearOnSuccess {
		a.clearConfirmedLogout()
	}
	return view, err
}

// storeAction 保存最近一次动作视图，供下一次调用判断是否还能重提。
func (a *Account) storeAction(stored **write.IntentView, view write.IntentView) {
	if view.ID == "" {
		return
	}
	a.mu.Lock()
	copied := view
	*stored = &copied
	a.mu.Unlock()
}

// actionRetryable 判断一次账号动作能否重新登记：只有明确失败与确认未发生可以；
// 已成功、未知、需人工确认与尚未收敛都不重提。
func actionRetryable(status write.Status) bool {
	switch status {
	case write.StatusFailed, write.StatusNotOccurred:
		return true
	default:
		return false
	}
}
