package account

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
)

// 真实账号夹具验证需要两个环境变量：
//
//	DIMAGENT_ACCOUNT_FIXTURE   已复制到临时目录的产品 home（含 auth.json）
//	DIMAGENT_APP_SERVER_BIN    真实 App Server 可执行文件
//
// 来源产品 home 只作为只读复制源，测试不读取也不写回；本包代码里没有任何来源
// 路径。未设置变量时整组测试跳过，普通 go test 不受影响。
const (
	fixtureProviderID = "dimcode-api-oauth"
	fixtureHomeEnv    = "DIMAGENT_ACCOUNT_FIXTURE"
	fixtureBinEnv     = "DIMAGENT_APP_SERVER_BIN"
)

func fixturePaths(t *testing.T) (string, string) {
	t.Helper()
	home := os.Getenv(fixtureHomeEnv)
	bin := os.Getenv(fixtureBinEnv)
	if home == "" || bin == "" {
		t.Skipf("未设置 %s / %s，跳过真实账号夹具验证", fixtureHomeEnv, fixtureBinEnv)
	}
	if _, err := os.Stat(filepath.Join(home, "auth.json")); err != nil {
		t.Fatalf("夹具缺少 auth.json（不读取内容）: %v", err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("App Server 可执行文件不可用: %v", err)
	}
	return home, bin
}

// openFixture 在夹具 home 上拉起真实 App Server 控制连接。测试结束后只回收自己
// 拉起的实例。
func openFixture(t *testing.T, home, bin string) *daemon.Connection {
	t.Helper()
	supervisor, err := daemon.New(daemon.Config{
		DataDir:        home,
		ServerPath:     bin,
		ServerArgs:     []string{"app-server"},
		RequestTimeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建连接监督者: %v", err)
	}
	connection, err := supervisor.Connect(context.Background())
	if err != nil {
		supervisor.Close(nil)
		t.Fatalf("连接真实 App Server: %v", err)
	}
	t.Cleanup(func() {
		if result, shutdownErr := supervisor.Shutdown(context.Background()); shutdownErr == nil {
			t.Logf("真实服务回收：stopped=%v reason=%s", result.Stopped, result.Reason)
		}
		supervisor.Close(nil)
	})
	return connection
}

// TestRealAccountFixtureReads 用真实 App Server 与临时夹具覆盖三条读取：账号状态、
// 逐项用量、模型清单，并核对 provider 声明。证据只输出条数与有无。
func TestRealAccountFixtureReads(t *testing.T) {
	home, bin := fixturePaths(t)
	connection := openFixture(t, home, bin)
	harness := &wireHarness{client: connection.Client}

	features := make(map[string]bool, len(connection.Hello.Features))
	for _, feature := range connection.Hello.Features {
		features[feature] = true
	}
	providers, err := harness.ReadProviders(context.Background())
	if err != nil {
		t.Fatalf("读取账号 provider: %v", err)
	}
	var picked Provider
	for _, provider := range providers {
		if provider.ID == fixtureProviderID {
			picked = provider
		}
	}
	if picked.ID == "" {
		t.Fatalf("真实服务端没有 %s provider", fixtureProviderID)
	}
	t.Logf("服务端版本=%s 能力：account.status=%v account.usage=%v provider.catalog=%v",
		connection.Hello.ServerVersion, features["account.status"], features["account.usage"], features["provider.catalog"])
	t.Logf("provider=%q loginMethods=%v directoryConfigured=%v", picked.ID, picked.LoginMethods, picked.Directory != "")

	account := newTestAccount(t, Config{Status: harness, Usage: harness, Models: harness, Providers: harness})

	snapshot, err := account.Refresh(context.Background())
	if err != nil {
		t.Fatalf("真实账号状态读取失败: %v", err)
	}
	if snapshot.Status != StatusSignedIn || snapshot.Identity.AccountID == "" {
		t.Fatalf("夹具账号状态不符：%+v", snapshot)
	}
	t.Logf("账号状态：status=%s identity=%s", snapshot.Status, identityShape(snapshot.Identity))

	usage, err := account.RefreshUsage(context.Background())
	if err != nil {
		t.Fatalf("真实用量读取失败: %v", err)
	}
	if !usage.Ready {
		t.Fatal("真实用量读取后应为可用")
	}
	t.Logf("用量：lines=%d aggregateAvailable=%v subscriptionIDSet=%v aggregateReason=%q observedAtSet=%v",
		len(usage.Lines), usage.Aggregate.Available, usage.HasSubscriptionID, usage.Aggregate.Reason, !usage.ObservedAt.IsZero())
	plan := usage.Plan
	credits := usage.Credits
	t.Logf("套餐形状：planSet=%v planNameSet=%v planStatusSet=%v priceAmountSet=%v currencySet=%v intervalSet=%v expiresAtSet=%v currentTermStartSet=%v",
		plan != nil,
		plan != nil && plan.Name != "", plan != nil && plan.Status != "",
		plan != nil && plan.PriceAmount != nil,
		plan != nil && plan.PriceCurrency != "", plan != nil && plan.PriceInterval != "",
		plan != nil && !plan.ExpiresAt.IsZero(), plan != nil && !plan.CurrentTermStart.IsZero())
	t.Logf("额度形状：creditsSet=%v usedSet=%v totalSet=%v remainingSet=%v unlimitedSet=%v resetsAtSet=%v",
		credits != nil,
		credits != nil && credits.UsedUnits != nil,
		credits != nil && credits.TotalUnits != nil,
		credits != nil && credits.RemainingUnits != nil,
		credits != nil && credits.Unlimited != nil,
		credits != nil && !credits.ResetsAt.IsZero())
	t.Logf("窗口形状：count=%d durationSet=%v usedSet=%v capSet=%v startedAtSet=%v expiresAtSet=%v",
		len(usage.Windows), hasWindowDuration(usage.Windows), hasWindowUsed(usage.Windows),
		hasWindowCap(usage.Windows), hasWindowStarted(usage.Windows), hasWindowExpires(usage.Windows))
	t.Logf("功能额度形状：count=%d keySet=%v usedSet=%v allowanceSet=%v remainingSet=%v unitSet=%v unlimitedSet=%v periodEndSet=%v",
		len(usage.Features), hasFeatureKey(usage.Features), hasFeatureUsed(usage.Features),
		hasFeatureAllowance(usage.Features), hasFeatureRemaining(usage.Features),
		hasFeatureUnit(usage.Features), hasFeatureUnlimited(usage.Features), hasFeaturePeriodEnd(usage.Features))
	t.Logf("链接形状：subscriptionURLSet=%v pricingURLSet=%v", usage.SubscriptionURL != "", usage.PricingURL != "")

	models, err := account.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("真实模型清单读取失败: %v", err)
	}
	if !models.Ready || len(models.Models) == 0 {
		t.Fatalf("已登录应有可用模型：%+v", models)
	}
	names := make([]string, 0, 3)
	for _, model := range models.Models {
		if len(names) >= 3 {
			break
		}
		names = append(names, model.ID)
	}
	t.Logf("模型清单：available=%d first=%v", len(models.Models), names)
}

// TestRealAccountFixtureBrowserLoginBeginReleases 用真实服务端验证按 provider 选择
// browser、授权地址、环回 /auth/callback，以及超时后按尝试标识取消。
func TestRealAccountFixtureBrowserLoginBeginReleases(t *testing.T) {
	home, bin := fixturePaths(t)
	connection := openFixture(t, home, bin)
	harness := &wireHarness{client: connection.Client, rawAuthorizationURL: true}
	browser := &stubBrowser{}
	account := newTestAccount(t, Config{
		Status:       harness,
		Providers:    harness,
		Auth:         harness,
		DeviceAuth:   harness,
		Browser:      browser,
		LoginTimeout: 2 * time.Second,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("真实账号状态读取失败: %v", err)
	}

	if _, err := account.Login(context.Background(), LoginRequest{ProviderID: fixtureProviderID}); !errors.Is(err, ErrLoginTimeout) {
		t.Fatalf("登录应以等待回调超时结束：%v", err)
	}
	if harness.cancels != 1 {
		t.Fatalf("超时后应取消一次服务端尝试：%d", harness.cancels)
	}
	opened := browser.openedURLs()
	if len(opened) != 1 {
		t.Fatalf("应打开一次授权地址：%d", len(opened))
	}
	authorizationURL, err := url.Parse(opened[0])
	if err != nil {
		t.Fatalf("授权地址无法解析: %v", err)
	}
	if authorizationURL.Hostname() != "dimagent.cn" {
		t.Fatalf("授权地址不是账号目录：%q", authorizationURL.Hostname())
	}
	redirectURI := authorizationURL.Query().Get("redirect_uri")
	parsedRedirect, err := url.Parse(redirectURI)
	if err != nil || parsedRedirect.Hostname() != "127.0.0.1" || parsedRedirect.Path != callbackPath {
		t.Fatalf("授权地址没有带环回回调：%q", redirectURI)
	}
	state := account.LoginState()
	if state.ProviderID != fixtureProviderID || state.Method != LoginMethodBrowser || state.Phase != LoginTimedOut {
		t.Fatalf("按 provider 分发结果不符：%+v", state)
	}
	if account.Status() != StatusSignedIn {
		t.Fatalf("登录超时不应改变已确认的账号状态：%q", account.Status())
	}
	t.Logf("浏览器登录发起：host=%s redirectLoopback=%v provider=%s method=%s cancelled=%d",
		authorizationURL.Hostname(), true, state.ProviderID, state.Method, harness.cancels)
}

// TestRealAccountFixtureDeviceLoginBeginReleases 用真实服务端验证按 provider 选择
// device、拿到验证地址与用户码，并在取消时交回服务端释放。不轮询、不完成交换。
func TestRealAccountFixtureDeviceLoginBeginReleases(t *testing.T) {
	home, bin := fixturePaths(t)
	connection := openFixture(t, home, bin)
	harness := &wireHarness{client: connection.Client}
	account := newTestAccount(t, Config{
		Status:       harness,
		Providers:    harness,
		Auth:         harness,
		DeviceAuth:   harness,
		Browser:      &stubBrowser{},
		LoginTimeout: 2 * time.Second,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("真实账号状态读取失败: %v", err)
	}

	start, err := account.BeginLogin(context.Background(), LoginRequest{
		ProviderID: fixtureProviderID,
		Method:     LoginMethodDevice,
	})
	if err != nil {
		t.Fatalf("发起设备码登录: %v", err)
	}
	if start.Attempt == "" || start.VerificationURI == "" || start.UserCode == "" {
		t.Fatalf("设备码登录信息不完整：%+v", start)
	}
	state := account.LoginState()
	if state.ProviderID != fixtureProviderID || state.Method != LoginMethodDevice || state.Phase != LoginWaiting {
		t.Fatalf("按 provider 分发结果不符：%+v", state)
	}

	account.CancelLogin()
	if harness.cancels != 1 {
		t.Fatalf("取消后应交回服务端释放一次：%d", harness.cancels)
	}
	if state := account.LoginState(); state.Phase != LoginCanceled {
		t.Fatalf("取消后的登录尝试状态不符：%+v", state)
	}
	t.Logf("设备码登录发起：provider=%s method=%s attemptIssued=%v verificationURISet=%v userCodeSet=%v cancelled=%d",
		state.ProviderID, state.Method, start.Attempt != "", start.VerificationURI != "", start.UserCode != "", harness.cancels)
}

func hasWindowDuration(windows []UsageWindow) bool {
	for _, window := range windows {
		if window.DurationHours > 0 {
			return true
		}
	}
	return false
}

func hasWindowUsed(windows []UsageWindow) bool {
	for _, window := range windows {
		if window.TokenUsed > 0 {
			return true
		}
	}
	return false
}

func hasWindowCap(windows []UsageWindow) bool {
	for _, window := range windows {
		if window.TokenCap > 0 {
			return true
		}
	}
	return false
}

func hasWindowStarted(windows []UsageWindow) bool {
	for _, window := range windows {
		if !window.StartedAt.IsZero() {
			return true
		}
	}
	return false
}

func hasWindowExpires(windows []UsageWindow) bool {
	for _, window := range windows {
		if !window.ExpiresAt.IsZero() {
			return true
		}
	}
	return false
}

func hasFeatureKey(features []UsageFeature) bool {
	for _, feature := range features {
		if feature.Key != "" {
			return true
		}
	}
	return false
}

func hasFeatureUsed(features []UsageFeature) bool {
	for _, feature := range features {
		if feature.Used != nil {
			return true
		}
	}
	return false
}

func hasFeatureAllowance(features []UsageFeature) bool {
	for _, feature := range features {
		if feature.Allowance != nil {
			return true
		}
	}
	return false
}

func hasFeatureRemaining(features []UsageFeature) bool {
	for _, feature := range features {
		if feature.Remaining != nil {
			return true
		}
	}
	return false
}

func hasFeatureUnit(features []UsageFeature) bool {
	for _, feature := range features {
		if feature.Unit != "" {
			return true
		}
	}
	return false
}

func hasFeatureUnlimited(features []UsageFeature) bool {
	for _, feature := range features {
		if feature.Unlimited != nil {
			return true
		}
	}
	return false
}

func hasFeaturePeriodEnd(features []UsageFeature) bool {
	for _, feature := range features {
		if !feature.PeriodEnd.IsZero() {
			return true
		}
	}
	return false
}

// identityShape 只输出身份字段的有无，不输出邮箱等个人信息。
func identityShape(identity Identity) string {
	return fmt.Sprintf("accountIDSet=%v emailSet=%v nameSet=%v planSet=%v",
		identity.AccountID != "", identity.Email != "", identity.DisplayName != "", identity.Plan != "")
}
