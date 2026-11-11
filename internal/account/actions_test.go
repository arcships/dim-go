package account

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// fixedOutcome 让写登记替身对每次登记返回同一个终态。
func fixedOutcome(status write.Status) func(write.Intent, string) (write.Status, error) {
	return func(write.Intent, string) (write.Status, error) { return status, nil }
}

// signedInAccount 造一个已登录、带用量与 provider 声明的账号，便于核对动作参数。
func signedInAccount(t *testing.T, writer Writer) *Account {
	t.Helper()
	account := newTestAccount(t, Config{
		Status: &stubStatusReader{fact: signedInFact()},
		Usage: &stubUsageReader{usage: Usage{
			Lines:             []UsageLine{{Key: "tokens", Used: 5, Limit: 50}},
			SubscriptionID:    1234,
			HasSubscriptionID: true,
		}},
		Providers: browserProviderSource(),
		Actions:   writer,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取已登录状态: %v", err)
	}
	if _, err := account.RefreshUsage(context.Background()); err != nil {
		t.Fatalf("读取用量: %v", err)
	}
	return account
}

func decodeParams(t *testing.T, intent write.Intent) map[string]any {
	t.Helper()
	var params map[string]any
	if err := json.Unmarshal(intent.Params, &params); err != nil {
		t.Fatalf("写意图参数无法解析: %v", err)
	}
	return params
}

// TestLogoutSendsOnceAndClearsOnSuccess 确认登出经写登记只发一次，带 provider，
// 成功后清空本地账号状态；同一会话再次登出不会重复登记。
func TestLogoutSendsOnceAndClearsOnSuccess(t *testing.T) {
	writer := &stubWriter{outcome: fixedOutcome(write.StatusSucceeded)}
	account := signedInAccount(t, writer)

	view, err := account.Logout(context.Background(), LogoutRequest{})
	if err != nil {
		t.Fatalf("登出: %v", err)
	}
	if view.Status != write.StatusSucceeded {
		t.Fatalf("登出结果不符：%+v", view)
	}
	intents := writer.registered()
	if len(intents) != 1 {
		t.Fatalf("登出应只登记一次：%d", len(intents))
	}
	if intents[0].Kind != write.KindAccountAction || intents[0].Key != "account:logout:1" {
		t.Fatalf("登出意图不符：%+v", intents[0])
	}
	params := decodeParams(t, intents[0])
	if params["op"] != actionLogout || params["providerId"] != "provider-browser" {
		t.Fatalf("登出参数不符：%+v", params)
	}

	snapshot := account.Snapshot()
	if snapshot.Status != StatusUnknown || snapshot.Identity != (Identity{}) {
		t.Fatalf("登出成功后本地状态应清空：%+v", snapshot)
	}
	if snapshot.Usage.Ready || len(snapshot.Usage.Lines) != 0 {
		t.Fatalf("登出后不应保留用量：%+v", snapshot.Usage)
	}

	if _, err := account.Logout(context.Background(), LogoutRequest{}); err != nil {
		t.Fatalf("再次登出: %v", err)
	}
	if count := writer.intentCount(); count != 1 {
		t.Fatalf("已成功的登出被重复登记：%d", count)
	}
}

// TestLogoutFailureKeepsState 确认明确失败的登出不清空本地状态，并允许显式重提。
func TestLogoutFailureKeepsState(t *testing.T) {
	writer := &stubWriter{outcome: fixedOutcome(write.StatusFailed)}
	account := signedInAccount(t, writer)

	view, err := account.Logout(context.Background(), LogoutRequest{})
	if err != nil {
		t.Fatalf("登出: %v", err)
	}
	if view.Status != write.StatusFailed {
		t.Fatalf("登出结果不符：%+v", view)
	}
	if account.Status() != StatusSignedIn || account.Snapshot().Identity.AccountID != "acct-1" {
		t.Fatalf("失败时不应清空本地状态：%+v", account.Snapshot())
	}
	if usage := account.Usage(); !usage.Ready {
		t.Fatalf("失败时不应清空用量：%+v", usage)
	}

	if _, err := account.Logout(context.Background(), LogoutRequest{}); err != nil {
		t.Fatalf("重试登出: %v", err)
	}
	mustIntentKeys(t, writer, "account:logout:1", "account:logout:2")
}

// TestLogoutUnknownIsNotResent 确认结果未知或需人工确认的登出不重复提交，也不
// 提前清空本地状态。
func TestLogoutUnknownIsNotResent(t *testing.T) {
	for _, status := range []write.Status{write.StatusUnknown, write.StatusNeedsManualConfirmation} {
		t.Run(string(status), func(t *testing.T) {
			writer := &stubWriter{outcome: fixedOutcome(status)}
			account := signedInAccount(t, writer)

			first, err := account.Logout(context.Background(), LogoutRequest{})
			if err != nil {
				t.Fatalf("登出: %v", err)
			}
			if first.Status != status {
				t.Fatalf("登出结果不符：%+v", first)
			}
			if account.Status() != StatusSignedIn {
				t.Fatalf("结果未定时不应清空本地状态：%q", account.Status())
			}

			second, err := account.Logout(context.Background(), LogoutRequest{})
			if err != nil {
				t.Fatalf("再次登出: %v", err)
			}
			if second.ID != first.ID {
				t.Fatalf("结果未定时应等待同一条意图：got %q，want %q", second.ID, first.ID)
			}
			if count := writer.intentCount(); count != 1 {
				t.Fatalf("结果未定的登出被重复提交：%d", count)
			}
		})
	}
}

// TestLogoutAgainAfterRelogin 确认重新登录后允许新的登出，不会被上一次成功挡住。
func TestLogoutAgainAfterRelogin(t *testing.T) {
	status := &stubStatusReader{fact: signedInFact()}
	writer := &stubWriter{outcome: fixedOutcome(write.StatusSucceeded)}
	account := newTestAccount(t, Config{
		Status:    status,
		Providers: browserProviderSource(),
		Actions:   writer,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取已登录状态: %v", err)
	}
	if _, err := account.Logout(context.Background(), LogoutRequest{}); err != nil {
		t.Fatalf("第一次登出: %v", err)
	}

	// 服务端事实变回未登录再变回已登录，代表新会话。
	status.set(Fact{Status: StatusSignedOut}, nil)
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取未登录状态: %v", err)
	}
	status.set(signedInFact(), nil)
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("重新读取已登录状态: %v", err)
	}
	if _, err := account.Logout(context.Background(), LogoutRequest{}); err != nil {
		t.Fatalf("新会话登出: %v", err)
	}
	mustIntentKeys(t, writer, "account:logout:1", "account:logout:2")
}

// TestResetReusesIdempotencyKeyAfterUnknown 确认未知结果后的显式重试复用同一
// 幂等键，但每次显式调用仍是一次登记；本包不会自动重发。
func TestResetReusesIdempotencyKeyAfterUnknown(t *testing.T) {
	writer := &stubWriter{outcome: fixedOutcome(write.StatusUnknown)}
	account := signedInAccount(t, writer)

	first, err := account.Reset(context.Background(), ResetRequest{Type: ResetTypeWindow})
	if err != nil {
		t.Fatalf("重置: %v", err)
	}
	if first.Status != write.StatusUnknown {
		t.Fatalf("重置结果不符：%+v", first)
	}
	second, err := account.Reset(context.Background(), ResetRequest{Type: ResetTypeWindow})
	if err != nil {
		t.Fatalf("再次重置: %v", err)
	}
	if second.Status != write.StatusUnknown {
		t.Fatalf("重置结果不符：%+v", second)
	}

	intents := writer.registered()
	if len(intents) != 2 {
		t.Fatalf("显式重试应登记新意图：%d", len(intents))
	}
	firstParams := decodeParams(t, intents[0])
	secondParams := decodeParams(t, intents[1])
	if firstParams["idempotencyKey"] == "" || firstParams["idempotencyKey"] != secondParams["idempotencyKey"] {
		t.Fatalf("未知结果后的重试应复用幂等键：%v vs %v", firstParams, secondParams)
	}
	for _, params := range []map[string]any{firstParams, secondParams} {
		if params["op"] != actionReset || params["resetType"] != string(ResetTypeWindow) ||
			params["subscriptionId"] != float64(1234) || params["providerId"] != "provider-browser" {
			t.Fatalf("重置参数不符：%+v", params)
		}
	}
}

// TestResetNewKeyAfterDefinitiveFailure 确认明确失败后换新的幂等键；每次显式
// 调用都带完整字段。
func TestResetNewKeyAfterDefinitiveFailure(t *testing.T) {
	writer := &stubWriter{outcome: fixedOutcome(write.StatusFailed)}
	account := signedInAccount(t, writer)

	if _, err := account.Reset(context.Background(), ResetRequest{Type: ResetTypeMonthlyFull}); err != nil {
		t.Fatalf("重置: %v", err)
	}
	writer.setOutcome(fixedOutcome(write.StatusSucceeded))
	if _, err := account.Reset(context.Background(), ResetRequest{Type: ResetTypeMonthlyFull}); err != nil {
		t.Fatalf("重试重置: %v", err)
	}
	intents := writer.registered()
	if len(intents) != 2 {
		t.Fatalf("重置登记次数不符：%d", len(intents))
	}
	first := decodeParams(t, intents[0])
	second := decodeParams(t, intents[1])
	if first["idempotencyKey"] == second["idempotencyKey"] || first["idempotencyKey"] == "" {
		t.Fatalf("明确失败后应换新幂等键：%v vs %v", first, second)
	}
}

// TestResetRequiresSignedIn 确认未登录时拒绝重置。
func TestResetRequiresSignedIn(t *testing.T) {
	writer := &stubWriter{}
	account := newTestAccount(t, Config{
		Status:  &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Usage:   &stubUsageReader{},
		Actions: writer,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取未登录状态: %v", err)
	}
	if _, err := account.Reset(context.Background(), ResetRequest{Type: ResetTypeWindow}); !errors.Is(err, ErrResetRequiresAccount) {
		t.Fatalf("应拒绝未登录重置：%v", err)
	}
	if writer.intentCount() != 0 {
		t.Fatal("未登录时不应登记重置")
	}
}

// TestResetRejectsInvalidTypeAndMissingSubscription 确认结构前置条件都明确报错。
func TestResetRejectsInvalidTypeAndMissingSubscription(t *testing.T) {
	writer := &stubWriter{}
	account := newTestAccount(t, Config{
		Status:  &stubStatusReader{fact: signedInFact()},
		Usage:   &stubUsageReader{usage: Usage{Lines: []UsageLine{{Key: "tokens"}}}},
		Actions: writer,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取已登录状态: %v", err)
	}
	if _, err := account.Reset(context.Background(), ResetRequest{Type: ResetType("bad")}); !errors.Is(err, ErrResetInvalidType) {
		t.Fatalf("应拒绝非法重置类型：%v", err)
	}
	if _, err := account.Reset(context.Background(), ResetRequest{Type: ResetTypeWindow}); !errors.Is(err, ErrResetMissingSubscription) {
		t.Fatalf("缺少订阅标识时应拒绝：%v", err)
	}
	if writer.intentCount() != 0 {
		t.Fatal("前置条件不满足时不应登记重置")
	}
}

// TestResetExplicitSubscriptionOverridesUsage 确认显式订阅标识优先于最近用量。
func TestResetExplicitSubscriptionOverridesUsage(t *testing.T) {
	writer := &stubWriter{outcome: fixedOutcome(write.StatusSucceeded)}
	account := signedInAccount(t, writer)
	if _, err := account.Reset(context.Background(), ResetRequest{
		Type:           ResetTypeWindow,
		SubscriptionID: 9876,
	}); err != nil {
		t.Fatalf("重置: %v", err)
	}
	intent := writer.registered()[0]
	if params := decodeParams(t, intent); params["subscriptionId"] != float64(9876) || params["resetType"] != string(ResetTypeWindow) {
		t.Fatalf("显式订阅标识没有生效：%+v", params)
	}
}

// TestResetKeyRegeneratedAfterLogout 确认登出后重置幂等键重新生成，不会把旧账号
// 会话的键带到新会话。
func TestResetKeyRegeneratedAfterLogout(t *testing.T) {
	status := &stubStatusReader{fact: signedInFact()}
	writer := &stubWriter{outcome: fixedOutcome(write.StatusUnknown)}
	account := newTestAccount(t, Config{
		Status: status,
		Usage: &stubUsageReader{usage: Usage{
			SubscriptionID:    1234,
			HasSubscriptionID: true,
		}},
		Providers: browserProviderSource(),
		Actions:   writer,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取已登录状态: %v", err)
	}
	if _, err := account.RefreshUsage(context.Background()); err != nil {
		t.Fatalf("读取用量: %v", err)
	}
	if _, err := account.Reset(context.Background(), ResetRequest{Type: ResetTypeWindow}); err != nil {
		t.Fatalf("第一次重置: %v", err)
	}
	firstKey := decodeParams(t, writer.registered()[0])["idempotencyKey"]

	writer.setOutcome(fixedOutcome(write.StatusSucceeded))
	if _, err := account.Logout(context.Background(), LogoutRequest{}); err != nil {
		t.Fatalf("登出: %v", err)
	}
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("重新读取已登录状态: %v", err)
	}
	if _, err := account.RefreshUsage(context.Background()); err != nil {
		t.Fatalf("重新读取用量: %v", err)
	}

	writer.setOutcome(fixedOutcome(write.StatusUnknown))
	if _, err := account.Reset(context.Background(), ResetRequest{Type: ResetTypeWindow}); err != nil {
		t.Fatalf("新会话重置: %v", err)
	}
	intents := writer.registered()
	secondKey := decodeParams(t, intents[len(intents)-1])["idempotencyKey"]
	if firstKey == nil || secondKey == nil || firstKey == secondKey {
		t.Fatalf("登出后应重新生成幂等键：%v vs %v", firstKey, secondKey)
	}
}

// TestLogoutProviderSelection 确认多 provider 时登出必须指定，指定后带上它。
func TestLogoutProviderSelection(t *testing.T) {
	source := &stubProviderSource{providers: []Provider{
		{ID: "provider-a", LoginMethods: []LoginMethod{LoginMethodBrowser}},
		{ID: "provider-b", LoginMethods: []LoginMethod{LoginMethodBrowser}},
	}}
	writer := &stubWriter{outcome: fixedOutcome(write.StatusSucceeded)}
	account := newTestAccount(t, Config{
		Status:    &stubStatusReader{fact: signedInFact()},
		Providers: source,
		Actions:   writer,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取已登录状态: %v", err)
	}
	if _, err := account.Logout(context.Background(), LogoutRequest{}); !errors.Is(err, ErrProviderSelectionRequired) {
		t.Fatalf("多个 provider 未指定时应拒绝：%v", err)
	}
	if _, err := account.Logout(context.Background(), LogoutRequest{ProviderID: "provider-b"}); err != nil {
		t.Fatalf("指定 provider 登出: %v", err)
	}
	intent := writer.registered()[0]
	if params := decodeParams(t, intent); params["providerId"] != "provider-b" {
		t.Fatalf("登出没有带上指定 provider：%+v", params)
	}
}

// TestActionsRequireWriter 确认没有写登记时登出与重置都明确不可用。
func TestActionsRequireWriter(t *testing.T) {
	account := newTestAccount(t, Config{})
	if _, err := account.Logout(context.Background(), LogoutRequest{}); !errors.Is(err, ErrActionsUnavailable) {
		t.Fatalf("登出应返回账号动作不可用：%v", err)
	}
	if _, err := account.Reset(context.Background(), ResetRequest{Type: ResetTypeWindow}); !errors.Is(err, ErrActionsUnavailable) {
		t.Fatalf("重置应返回账号动作不可用：%v", err)
	}
}
