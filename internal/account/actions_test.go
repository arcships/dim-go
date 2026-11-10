package account

import (
	"context"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// fixedOutcome 让写登记替身对每次登记返回同一个终态。
func fixedOutcome(status write.Status) func(write.Intent, string) (write.Status, error) {
	return func(write.Intent, string) (write.Status, error) { return status, nil }
}

// signedInAccount 造一个已登录、带用量数据的账号，便于核对登出后清空。
func signedInAccount(t *testing.T, writer Writer) *Account {
	t.Helper()
	account := newTestAccount(t, Config{
		Status:  &stubStatusReader{fact: signedInFact()},
		Usage:   &stubUsageReader{usage: Usage{Lines: []UsageLine{{Key: "tokens", Used: 5, Limit: 50}}}},
		Actions: writer,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取已登录状态: %v", err)
	}
	if _, err := account.RefreshUsage(context.Background()); err != nil {
		t.Fatalf("读取用量: %v", err)
	}
	return account
}

// TestLogoutSendsOnceAndClearsOnSuccess 确认登出经写登记只发一次，成功后清空本地
// 账号状态；再次登出不会重复登记。
func TestLogoutSendsOnceAndClearsOnSuccess(t *testing.T) {
	writer := &stubWriter{outcome: fixedOutcome(write.StatusSucceeded)}
	account := signedInAccount(t, writer)

	view, err := account.Logout(context.Background())
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
	if params := intentParams(t, intents[0]); params["op"] != actionLogout {
		t.Fatalf("登出参数不符：%+v", params)
	}

	snapshot := account.Snapshot()
	if snapshot.Status != StatusUnknown {
		t.Fatalf("登出成功后本地状态应清空为未知：%q", snapshot.Status)
	}
	if snapshot.Identity != (Identity{}) {
		t.Fatalf("登出后不应保留身份字段：%+v", snapshot.Identity)
	}
	if snapshot.Usage.Ready || len(snapshot.Usage.Lines) != 0 {
		t.Fatalf("登出后不应保留用量：%+v", snapshot.Usage)
	}
	if snapshot.Reason == "" {
		t.Fatal("清空后应说明原因")
	}

	// 再次登出：同一条意图已成功，不重复登记。
	if _, err := account.Logout(context.Background()); err != nil {
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

	view, err := account.Logout(context.Background())
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

	if _, err := account.Logout(context.Background()); err != nil {
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

			first, err := account.Logout(context.Background())
			if err != nil {
				t.Fatalf("登出: %v", err)
			}
			if first.Status != status {
				t.Fatalf("登出结果不符：%+v", first)
			}
			if account.Status() != StatusSignedIn {
				t.Fatalf("结果未定时不应清空本地状态：%q", account.Status())
			}

			second, err := account.Logout(context.Background())
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

// TestResetUnknownIsNotResent 确认重置结果未定时不重复提交：第二次显式重置只等待
// 同一条意图。
func TestResetUnknownIsNotResent(t *testing.T) {
	writer := &stubWriter{outcome: fixedOutcome(write.StatusUnknown)}
	account := newTestAccount(t, Config{Actions: writer})

	first, err := account.Reset(context.Background())
	if err != nil {
		t.Fatalf("重置: %v", err)
	}
	if first.Status != write.StatusUnknown {
		t.Fatalf("重置结果不符：%+v", first)
	}
	second, err := account.Reset(context.Background())
	if err != nil {
		t.Fatalf("再次重置: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("结果未定时应等待同一条意图：got %q，want %q", second.ID, first.ID)
	}
	intents := writer.registered()
	if len(intents) != 1 {
		t.Fatalf("结果未定的重置被重复提交：%d", len(intents))
	}
	if intents[0].Kind != write.KindAccountAction || intents[0].Key != "account:reset:1" {
		t.Fatalf("重置意图不符：%+v", intents[0])
	}
	if params := intentParams(t, intents[0]); params["op"] != actionReset {
		t.Fatalf("重置参数不符：%+v", params)
	}
}

// TestResetRetriesOnlyAfterExplicitFailure 确认重置只在明确失败或确认未发生后允许
// 用新的幂等键重提。
func TestResetRetriesOnlyAfterExplicitFailure(t *testing.T) {
	writer := &stubWriter{outcome: fixedOutcome(write.StatusFailed)}
	account := newTestAccount(t, Config{Actions: writer})

	if _, err := account.Reset(context.Background()); err != nil {
		t.Fatalf("重置: %v", err)
	}
	writer.setOutcome(fixedOutcome(write.StatusSucceeded))
	if _, err := account.Reset(context.Background()); err != nil {
		t.Fatalf("重试重置: %v", err)
	}
	mustIntentKeys(t, writer, "account:reset:1", "account:reset:2")
}

// TestActionsRequireWriter 确认没有写登记时登出与重置都明确不可用。
func TestActionsRequireWriter(t *testing.T) {
	account := newTestAccount(t, Config{})
	if _, err := account.Logout(context.Background()); !errors.Is(err, ErrActionsUnavailable) {
		t.Fatalf("登出应返回账号动作不可用：%v", err)
	}
	if _, err := account.Reset(context.Background()); !errors.Is(err, ErrActionsUnavailable) {
		t.Fatalf("重置应返回账号动作不可用：%v", err)
	}
}
