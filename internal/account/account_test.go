package account

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestNewStartsUnknownWithoutDefaults 确认新实例既不是已登录也不是未登录：服务端
// 还没确认过状态，本包不给默认值。
func TestNewStartsUnknownWithoutDefaults(t *testing.T) {
	account := newTestAccount(t, Config{})
	snapshot := account.Snapshot()

	if snapshot.Status != StatusUnknown {
		t.Fatalf("初始状态应是未知：%q", snapshot.Status)
	}
	if snapshot.Identity != (Identity{}) {
		t.Fatalf("未读取时不应有身份字段：%+v", snapshot.Identity)
	}
	if snapshot.Usage.Ready {
		t.Fatal("未读取时用量不应是可用状态")
	}
	if !snapshot.ReadAt.IsZero() || snapshot.ReadError != "" {
		t.Fatalf("未读取时不应有读取时间或错误：%+v", snapshot)
	}
	if snapshot.Reason == "" {
		t.Fatal("未知状态应说明原因")
	}

	models := account.Models()
	if len(models.Models) != 0 || models.Ready || models.Reason == "" {
		t.Fatalf("未读取模型清单时应是空清单并给原因：%+v", models)
	}
}

// TestRefreshSignedOutClearsIdentityAndUsage 确认未登录时身份与用量被清空，
// 状态只由读取结果决定。
func TestRefreshSignedOutClearsIdentityAndUsage(t *testing.T) {
	status := &stubStatusReader{fact: signedInFact()}
	usage := &stubUsageReader{usage: Usage{
		Lines:     []UsageLine{{Key: "tokens", Used: 10, Limit: 100, Unit: "k"}},
		Aggregate: AggregateUsage{Available: true, Used: 10, Limit: 100, Unit: "k"},
	}}
	account := newTestAccount(t, Config{Status: status, Usage: usage})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取已登录状态: %v", err)
	}
	if _, err := account.RefreshUsage(context.Background()); err != nil {
		t.Fatalf("读取用量: %v", err)
	}

	status.set(Fact{Status: StatusSignedOut}, nil)
	snapshot, err := account.Refresh(context.Background())
	if err != nil {
		t.Fatalf("读取未登录状态: %v", err)
	}
	if snapshot.Status != StatusSignedOut {
		t.Fatalf("状态不符：%q", snapshot.Status)
	}
	if snapshot.Identity != (Identity{}) {
		t.Fatalf("未登录时不应保留身份字段：%+v", snapshot.Identity)
	}
	if snapshot.Usage.Ready || len(snapshot.Usage.Lines) != 0 {
		t.Fatalf("未登录时不应保留上一账号的用量：%+v", snapshot.Usage)
	}
	if snapshot.ReadAt.IsZero() {
		t.Fatal("读取成功后应有读取时间")
	}
	if snapshot.Reason != "" || snapshot.ReadError != "" {
		t.Fatalf("成功读取后不应有原因或错误：%+v", snapshot)
	}
}

// TestRefreshSignedInAndExpired 确认已登录与过期两种状态都带回服务端给出的身份字段。
func TestRefreshSignedInAndExpired(t *testing.T) {
	status := &stubStatusReader{fact: signedInFact()}
	account := newTestAccount(t, Config{Status: status})

	snapshot, err := account.Refresh(context.Background())
	if err != nil {
		t.Fatalf("读取已登录状态: %v", err)
	}
	if snapshot.Status != StatusSignedIn || snapshot.Identity.Email != "user@example.com" {
		t.Fatalf("已登录状态不符：%+v", snapshot)
	}

	expired := signedInFact()
	expired.Status = StatusExpired
	status.set(expired, nil)
	snapshot, err = account.Refresh(context.Background())
	if err != nil {
		t.Fatalf("读取过期状态: %v", err)
	}
	if snapshot.Status != StatusExpired || snapshot.Identity.AccountID != "acct-1" {
		t.Fatalf("过期状态不符：%+v", snapshot)
	}
}

// TestMalformedFactRejectedWithoutStateChange 确认服务端事实结构不合法时不改状态：
// 未知状态、已登录缺账号标识都按错误处理，不降级成未登录。
func TestMalformedFactRejectedWithoutStateChange(t *testing.T) {
	status := &stubStatusReader{fact: signedInFact()}
	account := newTestAccount(t, Config{Status: status})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取已登录状态: %v", err)
	}

	cases := []struct {
		name string
		fact Fact
	}{
		{name: "未知状态", fact: Fact{Status: StatusUnknown}},
		{name: "空状态", fact: Fact{}},
		{name: "已登录缺账号标识", fact: Fact{Status: StatusSignedIn, Identity: Identity{Email: "a@b.c"}}},
		{name: "过期缺账号标识", fact: Fact{Status: StatusExpired}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status.set(testCase.fact, nil)
			snapshot, err := account.Refresh(context.Background())
			if !errors.Is(err, ErrMalformedFact) {
				t.Fatalf("应返回事实不合法错误：%v", err)
			}
			if snapshot.Status != StatusSignedIn {
				t.Fatalf("读取失败不应改变已确认状态：%q", snapshot.Status)
			}
			if snapshot.ReadError == "" {
				t.Fatal("读取失败应记录原因")
			}
		})
	}
}

// TestStatusReadFailureKeepsLastConfirmedFact 确认读取失败只记录错误，不清空已确认
// 的状态与读取时间。
func TestStatusReadFailureKeepsLastConfirmedFact(t *testing.T) {
	status := &stubStatusReader{fact: signedInFact()}
	account := newTestAccount(t, Config{Status: status})
	first, err := account.Refresh(context.Background())
	if err != nil {
		t.Fatalf("读取已登录状态: %v", err)
	}

	status.set(Fact{}, errors.New("连接中断"))
	snapshot, err := account.Refresh(context.Background())
	if !errors.Is(err, ErrStatusRead) {
		t.Fatalf("应返回状态读取错误：%v", err)
	}
	if snapshot.Status != StatusSignedIn || snapshot.Identity.AccountID != "acct-1" {
		t.Fatalf("读取失败不应清空已确认状态：%+v", snapshot)
	}
	if !snapshot.ReadAt.Equal(first.ReadAt) {
		t.Fatalf("读取失败不应推进读取时间：got %v，want %v", snapshot.ReadAt, first.ReadAt)
	}
	if snapshot.ReadError == "" {
		t.Fatal("读取失败应记录原因")
	}
}

// TestSnapshotIsDetached 确认调用方改不动账号内部的用量副本。
func TestSnapshotIsDetached(t *testing.T) {
	usage := &stubUsageReader{usage: Usage{Lines: []UsageLine{{Key: "tokens", Used: 1}}}}
	account := newTestAccount(t, Config{Usage: usage})
	if _, err := account.RefreshUsage(context.Background()); err != nil {
		t.Fatalf("读取用量: %v", err)
	}

	snapshot := account.Snapshot()
	snapshot.Usage.Lines[0].Used = 999
	if current := account.Usage(); current.Lines[0].Used != 1 {
		t.Fatalf("快照不是副本：%+v", current)
	}
}

// TestErrorCodeMapsToOriginalCodes 确认类型化错误能映射到原客户端的稳定错误码。
func TestErrorCodeMapsToOriginalCodes(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{ErrProviderUnknown, "account_invalid_input"},
		{ErrLoginMethodUnavailable, "account_invalid_input"},
		{ErrResetMissingSubscription, "account_invalid_input"},
		{ErrProvidersUnavailable, "account_provider_unavailable"},
		{ErrLoginCanceled, "login_cancelled"},
		{ErrLoginTimeout, "login_timeout"},
		{ErrLoginExpired, "login_timeout"},
		{ErrLoginCompletion, "login_failed"},
		{ErrDeviceLogin, "login_failed"},
		{ErrResetRequiresAccount, "account_required"},
		{ErrStatusRead, "account_status_failed"},
		{ErrUsageUnavailable, "account_usage_failed"},
		{ErrModelsRead, "model_setup_failed"},
	}
	for _, testCase := range cases {
		if got := ErrorCode(testCase.err); got != testCase.want {
			t.Fatalf("错误码不符：err=%v got=%q want=%q", testCase.err, got, testCase.want)
		}
	}
	if code := ErrorCode(errors.New("other")); code != "" {
		t.Fatalf("未知错误不应映射：%q", code)
	}
	if code := ErrorCode(nil); code != "" {
		t.Fatalf("nil 不应映射：%q", code)
	}
}

// TestReadTimeUsesConfiguredClock 确认读取时间来自可替换时钟，便于核对。
func TestReadTimeUsesConfiguredClock(t *testing.T) {
	fixed := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	status := &stubStatusReader{fact: signedInFact()}
	account := newTestAccount(t, Config{Status: status, Now: func() time.Time { return fixed }})
	snapshot, err := account.Refresh(context.Background())
	if err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}
	if !snapshot.ReadAt.Equal(fixed) {
		t.Fatalf("读取时间不符：%v", snapshot.ReadAt)
	}
}
