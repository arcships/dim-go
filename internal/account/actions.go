package account

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// 账号动作的操作名。它们放在写意图的透明参数里，由集成层的发送方映射到 wire。
const (
	actionLogout = "logout"
	actionReset  = "reset"
)

const (
	logoutKeyPrefix = "account:logout:"
	resetKeyPrefix  = "account:reset:"
)

// LogoutRequest 是一次登出请求。ProviderID 为空时，只有在服务端只声明一个
// provider 时自动选择；多个时必须指定。
type LogoutRequest struct {
	ProviderID string
}

// ResetRequest 是一次额度重置请求。字段与原客户端一致：重置类型、订阅标识与
// 稳定幂等键。SubscriptionID 为 0 时使用最近一次用量里读到的订阅标识。
type ResetRequest struct {
	ProviderID     string
	Type           ResetType
	SubscriptionID int64
}

// Logout 通过写登记发出一次登出意图，并等待它收敛。
//
// 只有写登记明确成功才清空本地账号状态（状态回到未知，身份、用量与重置键清空）；
// 失败、未知或需人工确认时本地状态不变。已经发出过且未明确失败的登出不会重复
// 登记；同一账号会话里再次登出只等待同一条意图。重新登录后允许新的登出。
func (a *Account) Logout(ctx context.Context, request LogoutRequest) (write.IntentView, error) {
	if a.cfg.Actions == nil {
		return write.IntentView{}, ErrActionsUnavailable
	}
	providerID, err := a.resolveActionProvider(ctx, request.ProviderID)
	if err != nil {
		return write.IntentView{}, err
	}
	a.logoutMu.Lock()
	defer a.logoutMu.Unlock()
	return a.runLogout(ctx, providerID)
}

// Reset 通过写登记发出一次重置意图，并等待它收敛。
//
// 与原客户端一致：只允许已登录时重置；必须知道重置类型与订阅标识；同一订阅的
// 同一类型在结果未定时复用同一个幂等键，显式重试不会重复扣减；成功或明确失败后
// 换新的键。本包不自动重发，重启后也不会补发。
func (a *Account) Reset(ctx context.Context, request ResetRequest) (write.IntentView, error) {
	if a.cfg.Actions == nil {
		return write.IntentView{}, ErrActionsUnavailable
	}
	if a.Status() != StatusSignedIn {
		return write.IntentView{}, ErrResetRequiresAccount
	}
	if !request.Type.valid() {
		return write.IntentView{}, fmt.Errorf("%w: %q", ErrResetInvalidType, request.Type)
	}
	providerID, err := a.resolveActionProvider(ctx, request.ProviderID)
	if err != nil {
		return write.IntentView{}, err
	}
	subscriptionID := request.SubscriptionID
	if subscriptionID == 0 {
		usage := a.Usage()
		if !usage.HasSubscriptionID && a.cfg.Usage != nil {
			// 与原客户端一致：没有缓存的订阅标识时先读一次用量。
			refreshed, readErr := a.RefreshUsage(ctx)
			if readErr != nil {
				return write.IntentView{}, fmt.Errorf("%w: %v", ErrResetMissingSubscription, readErr)
			}
			usage = refreshed
		}
		if usage.HasSubscriptionID {
			subscriptionID = usage.SubscriptionID
		}
	}
	if subscriptionID == 0 {
		return write.IntentView{}, ErrResetMissingSubscription
	}

	a.resetMu.Lock()
	defer a.resetMu.Unlock()

	keyScope := fmt.Sprintf("%d:%s", subscriptionID, request.Type)
	idempotencyKey, err := a.resetIdempotencyKey(keyScope)
	if err != nil {
		return write.IntentView{}, err
	}

	a.mu.Lock()
	a.resetAttempt++
	attempt := a.resetAttempt
	a.mu.Unlock()

	params := map[string]any{
		"op":             actionReset,
		"resetType":      string(request.Type),
		"subscriptionId": subscriptionID,
		"idempotencyKey": idempotencyKey,
	}
	if providerID != "" {
		params["providerId"] = providerID
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return write.IntentView{}, err
	}

	view, err := a.cfg.Actions.Register(write.Intent{
		Key:    resetKeyPrefix + string(request.Type) + ":" + strconv.Itoa(attempt),
		Kind:   write.KindAccountAction,
		Params: encoded,
	})
	if err != nil {
		return view, err
	}
	view, err = a.cfg.Actions.Wait(ctx, view.ID)
	// 明确结果之后换新键；未知与需人工确认保留键，显式重试复用同一幂等键。
	switch view.Status {
	case write.StatusSucceeded, write.StatusFailed, write.StatusNotOccurred:
		a.forgetResetKey(keyScope)
	}
	return view, err
}

// runLogout 收敛一条登出意图。providerID 为空表示本端没有 provider 声明。
func (a *Account) runLogout(ctx context.Context, providerID string) (write.IntentView, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	a.mu.Lock()
	previous := a.logoutView
	a.mu.Unlock()

	// 已经发出的登出：只在明确失败或确认未发生时允许重提；其余情况等同一条意图。
	if previous != nil && !actionRetryable(previous.Status) {
		view, err := a.cfg.Actions.Wait(ctx, previous.ID)
		a.storeLogout(view)
		if err == nil && view.Status == write.StatusSucceeded {
			a.clearConfirmedLogout()
		}
		return view, err
	}

	a.mu.Lock()
	a.logoutAttempt++
	attempt := a.logoutAttempt
	a.mu.Unlock()

	params := map[string]any{"op": actionLogout}
	if providerID != "" {
		params["providerId"] = providerID
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return write.IntentView{}, err
	}
	view, err := a.cfg.Actions.Register(write.Intent{
		Key:    logoutKeyPrefix + strconv.Itoa(attempt),
		Kind:   write.KindAccountAction,
		Params: encoded,
	})
	if err != nil {
		return view, err
	}
	view, err = a.cfg.Actions.Wait(ctx, view.ID)
	a.storeLogout(view)
	if err == nil && view.Status == write.StatusSucceeded {
		a.clearConfirmedLogout()
	}
	return view, err
}

// resolveActionProvider 解析账号动作要带的 provider。本端没有 provider 声明时
// 不构造一个，交给集成层按自己的口径处理。
func (a *Account) resolveActionProvider(ctx context.Context, requested string) (string, error) {
	if a.cfg.Providers == nil {
		return requested, nil
	}
	providers, err := a.Providers(ctx)
	if err != nil {
		return "", err
	}
	provider, err := selectProvider(providers, requested)
	if err != nil {
		return "", err
	}
	return provider.ID, nil
}

func (a *Account) resetIdempotencyKey(scope string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.resetKeys == nil {
		a.resetKeys = make(map[string]string)
	}
	key := a.resetKeys[scope]
	if key == "" {
		generated, err := randomActionKey()
		if err != nil {
			return "", err
		}
		key = generated
		a.resetKeys[scope] = key
	}
	return key, nil
}

func (a *Account) forgetResetKey(scope string) {
	a.mu.Lock()
	delete(a.resetKeys, scope)
	a.mu.Unlock()
}

func randomActionKey() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("生成账号动作幂等键: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

// storeLogout 保存最近一次登出视图，供同一会话内判断是否还能重提。
func (a *Account) storeLogout(view write.IntentView) {
	if view.ID == "" {
		return
	}
	a.mu.Lock()
	copied := view
	a.logoutView = &copied
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
