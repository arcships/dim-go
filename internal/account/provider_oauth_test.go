package account

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// providerOAuthStub 记录调用；每个方法返回脚本化的值。
type providerOAuthStub struct {
	calls      []string
	status     ProviderOAuthStatus
	browser    ProviderOAuthStart
	device     ProviderOAuthStart
	progress   ProviderOAuthProgress
	browserErr error
}

func (s *providerOAuthStub) ReadProviderOAuthStatus(
	_ context.Context, namespace, providerID string,
) (ProviderOAuthStatus, error) {
	s.calls = append(s.calls, "status "+namespace+" "+providerID)
	return s.status, nil
}

func (s *providerOAuthStub) BeginProviderBrowser(
	_ context.Context, namespace, providerID, redirectURI string,
) (ProviderOAuthStart, error) {
	s.calls = append(s.calls, "browser "+namespace+" "+providerID+" "+redirectURI)
	return s.browser, s.browserErr
}

func (s *providerOAuthStub) BeginProviderDevice(
	_ context.Context, namespace, providerID string,
) (ProviderOAuthStart, error) {
	s.calls = append(s.calls, "device "+namespace+" "+providerID)
	return s.device, nil
}

func (s *providerOAuthStub) CompleteProviderBrowser(
	_ context.Context, namespace, providerID, attempt, callbackURL string,
) error {
	s.calls = append(s.calls, "complete "+namespace+" "+providerID+" "+attempt+" "+callbackURL)
	return nil
}

func (s *providerOAuthStub) PollProviderOAuth(
	_ context.Context, namespace, providerID, attempt string,
) (ProviderOAuthProgress, error) {
	s.calls = append(s.calls, "poll "+namespace+" "+providerID+" "+attempt)
	return s.progress, nil
}

func (s *providerOAuthStub) CancelProviderOAuth(
	_ context.Context, namespace, providerID, attempt string,
) error {
	s.calls = append(s.calls, "cancel "+namespace+" "+providerID+" "+attempt)
	return nil
}

func (s *providerOAuthStub) LogoutProviderOAuth(
	_ context.Context, namespace, providerID string,
) error {
	s.calls = append(s.calls, "logout "+namespace+" "+providerID)
	return nil
}

func contractProviderOAuth(t *testing.T, stub *providerOAuthStub) *ProviderOAuth {
	t.Helper()
	oauth := NewProviderOAuth(stub)
	if oauth == nil {
		t.Fatal("有来源时应建出入口")
	}
	return oauth
}

// TestProviderOAuthRejectsUnknownTarget 断言命名空间与定位参数在发请求前校验：
// 托管帐号要 providerId，原生 OAuth 不认未知命名空间。
func TestProviderOAuthRejectsUnknownTarget(t *testing.T) {
	stub := &providerOAuthStub{}
	oauth := contractProviderOAuth(t, stub)

	if _, err := oauth.ReadStatus(context.Background(), "account", ""); !errors.Is(err, ErrProviderOAuthRequest) {
		t.Fatalf("托管帐号缺少 providerId 没有拒绝：%v", err)
	}
	if _, err := oauth.BeginBrowser(context.Background(), "account.other", "p", ""); !errors.Is(err, ErrProviderOAuthRequest) {
		t.Fatalf("未知命名空间没有拒绝：%v", err)
	}
	stub.device = ProviderOAuthStart{Attempt: "a1", VerificationURI: "https://v", UserCode: "CD-1"}
	if _, err := oauth.BeginDevice(context.Background(), "account.grok", ""); err != nil {
		t.Fatalf("原生 OAuth 不该要求 providerId：%v", err)
	}
	if len(stub.calls) != 1 || !strings.HasPrefix(stub.calls[0], "device account.grok ") {
		t.Fatalf("拒绝的请求也发了：%v", stub.calls)
	}
}

// TestProviderOAuthValidatesStart 断言发起回执必须带尝试标识与地址，设备码还要
// 验证地址与用户码；缺项时不把半截结果交给界面。
func TestProviderOAuthValidatesStart(t *testing.T) {
	stub := &providerOAuthStub{browser: ProviderOAuthStart{}}
	oauth := contractProviderOAuth(t, stub)
	if _, err := oauth.BeginBrowser(context.Background(), "account.codex", "", ""); !errors.Is(err, ErrProviderOAuthRequest) {
		t.Fatalf("缺尝试标识没有拒绝：%v", err)
	}
	stub.device = ProviderOAuthStart{Attempt: "a1", VerificationURI: "https://v", UserCode: ""}
	if _, err := oauth.BeginDevice(context.Background(), "account.grok", ""); !errors.Is(err, ErrProviderOAuthRequest) {
		t.Fatalf("缺用户码没有拒绝：%v", err)
	}
	stub.device = ProviderOAuthStart{Attempt: "a1", VerificationURI: "https://v", UserCode: "CD-1"}
	start, err := oauth.BeginDevice(context.Background(), "account.grok", "")
	if err != nil {
		t.Fatalf("完整设备码被拒绝：%v", err)
	}
	if start.Interval != DefaultDevicePollInterval {
		t.Fatalf("默认轮询间隔 = %v，想要 %v", start.Interval, DefaultDevicePollInterval)
	}
}

// TestProviderOAuthForwards 断言各动作按命名空间与参数转发，取消没有尝试时什么
// 也不发。
func TestProviderOAuthForwards(t *testing.T) {
	stub := &providerOAuthStub{
		browser:  ProviderOAuthStart{Attempt: "a1", AuthorizationURL: "https://auth"},
		progress: ProviderOAuthProgress{Stage: ProviderOAuthPending},
	}
	oauth := contractProviderOAuth(t, stub)
	ctx := context.Background()

	if _, err := oauth.BeginBrowser(ctx, "account", "p1", " http://127.0.0.1:1/cb "); err != nil {
		t.Fatalf("发起浏览器登录：%v", err)
	}
	if err := oauth.CompleteBrowser(ctx, "account", "p1", "a1", " http://127.0.0.1:1/cb?code=x "); err != nil {
		t.Fatalf("完成浏览器登录：%v", err)
	}
	if err := oauth.CompleteBrowser(ctx, "account", "p1", "a1", "  "); !errors.Is(err, ErrProviderOAuthRequest) {
		t.Fatalf("空回调没有拒绝：%v", err)
	}
	if progress, err := oauth.Poll(ctx, "account.codex", "", "a1"); err != nil ||
		progress.Stage != ProviderOAuthPending {
		t.Fatalf("轮询：%+v %v", progress, err)
	}
	if err := oauth.Cancel(ctx, "account.codex", "", ""); err != nil {
		t.Fatalf("没有尝试时取消应什么也不做：%v", err)
	}
	if err := oauth.Logout(ctx, "account", "p1"); err != nil {
		t.Fatalf("登出：%v", err)
	}
	want := []string{
		"browser account p1 http://127.0.0.1:1/cb",
		"complete account p1 a1 http://127.0.0.1:1/cb?code=x",
		"poll account.codex  a1",
		"logout account p1",
	}
	if strings.Join(stub.calls, "|") != strings.Join(want, "|") {
		t.Fatalf("调用 = %v，想要 %v", stub.calls, want)
	}
}

// TestProviderOAuthRejectsUnknownStage 断言认不出的尝试状态不被当成完成。
func TestProviderOAuthRejectsUnknownStage(t *testing.T) {
	stub := &providerOAuthStub{progress: ProviderOAuthProgress{Stage: "weird"}}
	oauth := contractProviderOAuth(t, stub)
	if _, err := oauth.Poll(context.Background(), "account.grok", "", "a1"); !errors.Is(err, ErrProviderOAuthRequest) {
		t.Fatalf("未知状态没有拒绝：%v", err)
	}
	_ = time.Now
}
