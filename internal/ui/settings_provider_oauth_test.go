package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「模型供应」详情里按提供商登录的回归：四种形态的状态、入口与写通道。
// 需求 ID 写在各个用例注释里。

// oauthIntents 记录按提供商登录的入口调用；begin / poll 的返回值可脚本化。
type oauthIntents struct {
	Intents
	calls chan string

	beginErr    error
	completeErr error
	pollStage   string
	pollReason  string
	logoutErr   error
}

func newOAuthIntents() *oauthIntents {
	return &oauthIntents{calls: make(chan string, 16), pollStage: "completed"}
}

func (o *oauthIntents) expect(t *testing.T, want string) {
	t.Helper()
	select {
	case got := <-o.calls:
		if got != want {
			t.Fatalf("调用了 %q，想要 %q", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("没有等到调用 %q", want)
	}
}

func (o *oauthIntents) BeginProviderOAuth(
	_ context.Context, providerID, namespace, method, redirectURI string,
) (ProviderOAuthStartSnapshot, error) {
	o.calls <- "Begin " + providerID + " " + namespace + " " + method + " " + redirectURI
	if o.beginErr != nil {
		return ProviderOAuthStartSnapshot{}, o.beginErr
	}
	if method == "device" {
		return ProviderOAuthStartSnapshot{
			Attempt: "att-device", VerificationURI: "https://verify.test/device",
			UserCode: "ABCD-1234", Interval: 1,
		}, nil
	}
	return ProviderOAuthStartSnapshot{
		Attempt: "att-browser", AuthorizationURL: "https://auth.test/start",
		RedirectURI: redirectURI,
	}, nil
}

func (o *oauthIntents) CompleteProviderOAuth(
	_ context.Context, providerID, namespace, attempt, callbackURL string,
) error {
	o.calls <- "Complete " + providerID + " " + namespace + " " + attempt + " " + callbackURL
	return o.completeErr
}

func (o *oauthIntents) PollProviderOAuth(
	_ context.Context, providerID, namespace, attempt string,
) (ProviderOAuthProgressSnapshot, error) {
	o.calls <- "Poll " + providerID + " " + namespace + " " + attempt
	return ProviderOAuthProgressSnapshot{Stage: o.pollStage, Reason: o.pollReason}, nil
}

func (o *oauthIntents) CancelProviderOAuth(
	_ context.Context, providerID, namespace, attempt string,
) error {
	o.calls <- "Cancel " + providerID + " " + namespace + " " + attempt
	return nil
}

func (o *oauthIntents) LogoutProviderOAuth(_ context.Context, providerID, namespace string) error {
	o.calls <- "Logout " + providerID + " " + namespace
	return o.logoutErr
}

func (o *oauthIntents) RefreshProviders(context.Context) error {
	o.calls <- "RefreshProviders"
	return nil
}

func (o *oauthIntents) OpenProviderOAuthURL(_ context.Context, url string) error {
	o.calls <- "OpenURL " + url
	return nil
}

// oauthSource 造一个带账号形态的 provider 读数。
func oauthSource(providerID, name, namespace, authority string, authenticated bool,
	methods ...string,
) *fakeSource {
	src := wiredSource()
	src.settings.ProviderRecordsKnown = true
	src.settings.ProviderRecords = []ProviderRecordSnapshot{{
		ID: providerID, Name: name, BaseURL: "https://api.test/v1",
		DriverKind: "openai", AuthType: "bearer", Enabled: true, Version: 1,
		RemoteModelList: true, Authority: authority,
		Account: &ProviderAccountSnapshot{
			Namespace: namespace, LoginMethods: methods, Authenticated: authenticated,
			AccountLabel: "ada@example.test", AccountPreview: "acct_1",
		},
	}}
	return src
}

// TestProviderOAuthCodexSignedOut 断言 Codex 未登录分支：说明、登录入口、没有
// 通用「编辑」入口；点登录按 account.codex / browser 发起（S-configuration-29/30）。
func TestProviderOAuthCodexSignedOut(t *testing.T) {
	resetSettingsScreenState()
	oauth := newOAuthIntents()
	app := NewApp(Options{State: StateDefault,
		Source: oauthSource("openai-codex", "OpenAI Codex", configNamespaceCodex,
			"openai_codex", false, configLoginBrowser), Intents: oauth})
	tt := openSettingsSection(t, app, secProviders)

	for _, want := range []string{configCopyCodexSignedOut, configCopyCodexSignIn} {
		if !tt.HasText(want) {
			t.Fatalf("Codex 未登录缺少 %q：%v", want, tt.Texts())
		}
	}
	if _, ok := tt.Find("编辑"); ok {
		t.Fatalf("账号形态的 provider 不该出现通用编辑入口：%v", tt.Texts())
	}
	if err := tt.Click(configCopyCodexSignIn); err != nil {
		t.Fatalf("点不到登录：%v", err)
	}
	oauth.expect(t, "Begin openai-codex account.codex browser ")
	tt.Frame()
	oauth.expect(t, "OpenURL https://auth.test/start")
	// 等待浏览器回调：说明、取消、打开 / 复制与回调输入都在。
	for _, want := range []string{configCopyCodexWaiting, configCopyCancel, configCopyOpenURL,
		configCopyCopy, configCopyCallbackURL, configCopyConfirm} {
		if !tt.HasText(want) {
			t.Fatalf("等待回调分支缺少 %q：%v", want, tt.Texts())
		}
	}
}

// TestProviderOAuthCodexComplete 断言粘贴回调地址后完成交换，成功后回到未登录
// 说明并写回执（S-configuration-31/36）。
func TestProviderOAuthCodexComplete(t *testing.T) {
	resetSettingsScreenState()
	oauth := newOAuthIntents()
	app := NewApp(Options{State: StateDefault,
		Source: oauthSource("openai-codex", "OpenAI Codex", configNamespaceCodex,
			"openai_codex", false, configLoginBrowser), Intents: oauth})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click(configCopyCodexSignIn); err != nil {
		t.Fatalf("点不到登录：%v", err)
	}
	oauth.expect(t, "Begin openai-codex account.codex browser ")
	tt.Frame()
	oauth.expect(t, "OpenURL https://auth.test/start")
	configOAuthFlow.callbackURL = "http://127.0.0.1:1455/auth/callback?code=abc"
	tt.Frame()
	if err := tt.Click(configCopyConfirm); err != nil {
		t.Fatalf("点不到确认：%v", err)
	}
	oauth.expect(t, "Complete openai-codex account.codex att-browser "+
		"http://127.0.0.1:1455/auth/callback?code=abc")
	tt.Frame()
	if configOAuthFlow.attempt != "" {
		t.Fatalf("完成后仍在等待：%+v", configOAuthFlow)
	}
	if !tt.HasText("登录成功") {
		t.Fatalf("完成登录没有回执：%v", tt.Texts())
	}
}

// TestProviderOAuthCodexFailed 断言登录失败的原因就地显示（S-configuration-36）。
func TestProviderOAuthCodexFailed(t *testing.T) {
	resetSettingsScreenState()
	oauth := newOAuthIntents()
	oauth.beginErr = errors.New("授权服务端拒绝了这次登录")
	app := NewApp(Options{State: StateDefault,
		Source: oauthSource("openai-codex", "OpenAI Codex", configNamespaceCodex,
			"openai_codex", false, configLoginBrowser), Intents: oauth})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click(configCopyCodexSignIn); err != nil {
		t.Fatalf("点不到登录：%v", err)
	}
	oauth.expect(t, "Begin openai-codex account.codex browser ")
	tt.Frame()
	tt.Frame()
	if !tt.HasText("授权服务端拒绝了这次登录") {
		t.Fatalf("登录失败没有写原因：%v", tt.Texts())
	}
}

// TestProviderOAuthGrokDevice 断言 Grok 设备码分支：等待授权、设备码、验证地址、
// 取消；轮询结果由状态处理器结算（S-configuration-37/38/40）。
func TestProviderOAuthGrokDevice(t *testing.T) {
	resetSettingsScreenState()
	oauth := newOAuthIntents()
	app := NewApp(Options{State: StateDefault,
		Source: oauthSource("grok-build", "Grok Build", configNamespaceGrok,
			"grok", false, configLoginDevice), Intents: oauth})
	tt := openSettingsSection(t, app, secProviders)

	if !tt.HasText(configCopyGrokSignedOut) || !tt.HasText(configCopyGrokSignIn) {
		t.Fatalf("Grok 未登录说明或入口缺失：%v", tt.Texts())
	}
	if err := tt.Click(configCopyGrokSignIn); err != nil {
		t.Fatalf("点不到登录：%v", err)
	}
	oauth.expect(t, "Begin grok-build account.grok device ")
	tt.Frame()
	for _, want := range []string{configCopyGrokWaiting, configCopyGrokDeviceCode,
		"ABCD-1234", configCopyGrokOpenVerify, configCopyCopy, configCopyCancel} {
		if !tt.HasText(want) {
			t.Fatalf("等待设备码分支缺少 %q：%v", want, tt.Texts())
		}
	}
	// 轮询结算：pending 再排一次；completed 结束等待。
	app.configOAuthPollResult("grok-build", "grok-build", configNamespaceGrok,
		"att-device", ProviderOAuthProgressSnapshot{Stage: "completed"}, time.Second)
	if configOAuthFlow.attempt != "" {
		t.Fatalf("completed 后仍在等待：%+v", configOAuthFlow)
	}
}

// TestProviderOAuthGrokCancel 断言取消登录按尝试标识发一次取消（S-configuration-32）。
func TestProviderOAuthGrokCancel(t *testing.T) {
	resetSettingsScreenState()
	oauth := newOAuthIntents()
	app := NewApp(Options{State: StateDefault,
		Source: oauthSource("grok-build", "Grok Build", configNamespaceGrok,
			"grok", false, configLoginDevice), Intents: oauth})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click(configCopyGrokSignIn); err != nil {
		t.Fatalf("点不到登录：%v", err)
	}
	oauth.expect(t, "Begin grok-build account.grok device ")
	tt.Frame()
	if err := tt.Click(configCopyCancel); err != nil {
		t.Fatalf("点不到取消：%v", err)
	}
	oauth.expect(t, "Cancel grok-build account.grok att-device")
	tt.Frame()
	if configOAuthFlow.attempt != "" {
		t.Fatalf("取消后仍在等待：%+v", configOAuthFlow)
	}
}

// TestProviderOAuthAuthenticated 断言已连接卡：账号标识、说明、刷新与退出登录；
// 点退出登录按命名空间发一次登出（S-configuration-33/34）。
func TestProviderOAuthAuthenticated(t *testing.T) {
	resetSettingsScreenState()
	oauth := newOAuthIntents()
	app := NewApp(Options{State: StateDefault,
		Source: oauthSource("openai-codex", "OpenAI Codex", configNamespaceCodex,
			"openai_codex", true, configLoginBrowser), Intents: oauth})
	tt := openSettingsSection(t, app, secProviders)

	for _, want := range []string{"ada@example.test", "账户：acct_1",
		configCopyCodexSignedIn, configCopyRefresh, configCopySignOut} {
		if !tt.HasText(want) {
			t.Fatalf("已连接卡缺少 %q：%v", want, tt.Texts())
		}
	}
	if err := tt.Click(configCopySignOut); err != nil {
		t.Fatalf("点不到退出登录：%v", err)
	}
	oauth.expect(t, "Logout openai-codex account.codex")
	tt.Frame()
	if !tt.HasText("已退出登录") {
		t.Fatalf("退出登录没有回执：%v", tt.Texts())
	}
}

// TestProviderOAuthManagedMethods 断言托管帐号按服务端声明摆入口：两种都声明时
// 两个按钮都在，回调地址输入也在；点浏览器登录带回调地址发起
// （S-configuration-41/42）。
func TestProviderOAuthManagedMethods(t *testing.T) {
	resetSettingsScreenState()
	oauth := newOAuthIntents()
	app := NewApp(Options{State: StateDefault,
		Source: oauthSource("other-oauth", "Other OAuth", configNamespaceAccount,
			"account", false, configLoginBrowser, configLoginDevice), Intents: oauth})
	tt := openSettingsSection(t, app, secProviders)

	for _, want := range []string{
		"登录 Other OAuth 以加载其模型。", configCopyManagedBrowserLogin,
		configCopyManagedDeviceLogin, configCopyManagedRedirect,
	} {
		if !tt.HasText(want) {
			t.Fatalf("托管帐号未登录缺少 %q：%v", want, tt.Texts())
		}
	}
	if err := tt.Click(configCopyManagedBrowserLogin); err != nil {
		t.Fatalf("点不到浏览器登录：%v", err)
	}
	oauth.expect(t, "Begin other-oauth account browser http://127.0.0.1:53682/oauth/callback")
}

// TestProviderOAuthModelsEmptyCopy 断言账号形态未登录时的模型空态是「登录成功后
// 将自动拉取可用模型。」，不是通用的「点击刷新」（S-configuration-25/41）。
func TestProviderOAuthModelsEmptyCopy(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault,
		Source: oauthSource("openai-codex", "OpenAI Codex", configNamespaceCodex,
			"openai_codex", false, configLoginBrowser)})
	tt := openSettingsSection(t, app, secProviders)

	if !tt.HasText(configCopyAutoModels) {
		t.Fatalf("未登录的模型空态文案不对：%v", tt.Texts())
	}
	if strings.Contains(joinedText(tt), "点击「刷新」获取") {
		t.Fatalf("未登录时不该写「点击刷新获取」")
	}
}

// 确认 tester 的取消点击走的是可读名，不是位置。
var _ = gui.RoleTextField

// TestProviderOAuthDevicePolls 断言设备码登录的轮询由渲染线程按间隔发起：pending
// 推后下一次，completed 结束等待并写回执（S-configuration-32）。
func TestProviderOAuthDevicePolls(t *testing.T) {
	resetSettingsScreenState()
	oauth := newOAuthIntents()
	oauth.pollStage = "pending"
	app := NewApp(Options{State: StateDefault,
		Source: oauthSource("grok-build", "Grok Build", configNamespaceGrok,
			"grok", false, configLoginDevice), Intents: oauth})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click(configCopyGrokSignIn); err != nil {
		t.Fatalf("点不到登录：%v", err)
	}
	oauth.expect(t, "Begin grok-build account.grok device ")
	tt.Frame()
	// 间隔没到：这一帧不查。
	tt.Frame()
	select {
	case got := <-oauth.calls:
		t.Fatalf("间隔没到就查了：%q", got)
	default:
	}
	configOAuthFlow.nextPoll = time.Now().Add(-time.Second)
	tt.Frame()
	oauth.expect(t, "Poll grok-build account.grok att-device")
	tt.Frame()
	tt.Frame()
	if configOAuthFlow.polling {
		t.Fatal("pending 后仍标记查询在路上")
	}
	if configOAuthFlow.nextPoll.Before(time.Now()) {
		t.Fatal("pending 没有推后下一次查询")
	}
	oauth.pollStage = "completed"
	configOAuthFlow.nextPoll = time.Now().Add(-time.Second)
	tt.Frame()
	oauth.expect(t, "Poll grok-build account.grok att-device")
	tt.Frame()
	tt.Frame()
	if configOAuthFlow.attempt != "" {
		t.Fatalf("completed 后仍在等待：%+v", configOAuthFlow)
	}
	if !tt.HasText("登录成功") {
		t.Fatalf("完成后没有回执：%v", tt.Texts())
	}
}
