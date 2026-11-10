package app

import (
	"context"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
)

// TestAccountProviderSource 核对 `account.providers` 的映射。
func TestAccountProviderSource(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "account.providers", Result: []any{
				map[string]any{
					"providerId": "dimcode-api-oauth", "displayName": "Dim OAuth",
					"loginMethods":     []any{"browser", "device"},
					"defaultDirectory": "https://dimagent.cn",
				},
				map[string]any{"providerId": "other"},
			}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	source := &accountProviderSource{wire: result.base.accountWire}
	providers, err := source.ReadProviders(context.Background())
	if err != nil {
		t.Fatalf("读取账号提供者: %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("提供者数量不对: %+v", providers)
	}
	first := providers[0]
	if first.ID != "dimcode-api-oauth" || first.DisplayName != "Dim OAuth" ||
		first.Directory != "https://dimagent.cn" || len(first.LoginMethods) != 2 ||
		first.LoginMethods[0] != account.LoginMethodBrowser {
		t.Fatalf("提供者映射不对: %+v", first)
	}
}

// TestAccountProviderAuthorization 核对按 provider 的浏览器登录发起与完成参数。
func TestAccountProviderAuthorization(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "account.login.browser.begin", Result: map[string]any{
				"attemptId": "att-1", "authorizationUrl": "https://dimagent.cn/auth?x=1",
			}},
			{Method: "account.login.browser.complete", Result: map[string]any{}},
			{Method: "account.login.cancel", Result: map[string]any{}},
			{Method: "account.providers", Result: []any{map[string]any{
				"providerId": "dimcode-api-oauth", "loginMethods": []any{"browser"},
			}}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	source := &accountAuthSource{wire: result.base.accountWire}
	ctx := context.Background()

	request, err := source.BeginProviderAuthorization(ctx, account.ProviderAuthorizationBegin{
		ProviderID:  "dimcode-api-oauth",
		Directory:   "https://dimagent.cn",
		RedirectURI: "http://127.0.0.1:4321/auth/callback",
	})
	if err != nil {
		t.Fatalf("发起浏览器登录: %v", err)
	}
	if request.Attempt != "att-1" || request.URL != "https://dimagent.cn/auth?x=1" {
		t.Fatalf("发起回执映射不对: %+v", request)
	}
	beginParams := requestParams(t, server.RequestsFor("account.login.browser.begin")[0])
	if beginParams["providerId"] != "dimcode-api-oauth" ||
		beginParams["directory"] != "https://dimagent.cn" ||
		beginParams["redirectUri"] != "http://127.0.0.1:4321/auth/callback" {
		t.Fatalf("发起参数不对: %v", beginParams)
	}

	if err := source.CompleteProviderAuthorization(ctx, "dimcode-api-oauth", "att-1", account.AuthorizationResult{
		CallbackURL: "http://127.0.0.1:4321/auth/callback?code=x",
	}); err != nil {
		t.Fatalf("完成交换: %v", err)
	}
	completeParams := requestParams(t, server.RequestsFor("account.login.browser.complete")[0])
	if completeParams["providerId"] != "dimcode-api-oauth" || completeParams["attemptId"] != "att-1" ||
		!strings.Contains(completeParams["callbackUrl"].(string), "code=x") {
		t.Fatalf("完成参数不对: %v", completeParams)
	}

	if err := source.CancelAuthorization(ctx, "att-2"); err != nil {
		t.Fatalf("取消尝试: %v", err)
	}
	cancelParams := requestParams(t, server.RequestsFor("account.login.cancel")[0])
	if cancelParams["attemptId"] != "att-2" || cancelParams["providerId"] == "" {
		t.Fatalf("取消参数不对: %v", cancelParams)
	}
}

// TestAccountDeviceAuthorization 核对设备码发起与轮询的映射。
func TestAccountDeviceAuthorization(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "account.login.begin", Result: map[string]any{
				"attemptId": "dev-1", "verificationUri": "https://dimagent.cn/device",
				"verificationUriComplete": "https://dimagent.cn/device?code=ABCD",
				"userCode":                "ABCD", "expiresAt": int64(1900000000), "interval": int64(5),
			}},
			{Method: "account.login.status", Result: map[string]any{"status": "pending"}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	source := &accountDeviceSource{wire: result.base.accountWire}
	ctx := context.Background()

	device, err := source.BeginDeviceAuthorization(ctx, "dimcode-api-oauth", "https://dimagent.cn")
	if err != nil {
		t.Fatalf("发起设备码登录: %v", err)
	}
	if device.Attempt != "dev-1" || device.UserCode != "ABCD" ||
		device.VerificationURI != "https://dimagent.cn/device" || device.ExpiresAt.IsZero() {
		t.Fatalf("设备码映射不对: %+v", device)
	}
	beginParams := requestParams(t, server.RequestsFor("account.login.begin")[0])
	if beginParams["providerId"] != "dimcode-api-oauth" || beginParams["directory"] != "https://dimagent.cn" {
		t.Fatalf("设备码发起参数不对: %v", beginParams)
	}
	if _, ok := beginParams["redirectUri"]; ok {
		t.Fatalf("设备码发起不应带 redirectUri: %v", beginParams)
	}

	state, err := source.PollDeviceAuthorization(ctx, "dev-1")
	if err != nil {
		t.Fatalf("轮询设备码: %v", err)
	}
	if state.Status != account.DeviceLoginPending {
		t.Fatalf("设备码状态映射不对: %+v", state)
	}
	statusParams := requestParams(t, server.RequestsFor("account.login.status")[0])
	if statusParams["providerId"] != "dimcode-api-oauth" || statusParams["attemptId"] != "dev-1" {
		t.Fatalf("轮询参数不对: %v", statusParams)
	}

	// 未知状态按事实不合法拒绝，不改写成 pending。
	if _, err := deviceLoginStatus("weird"); err == nil {
		t.Fatal("未知设备码状态应报错")
	}
}
