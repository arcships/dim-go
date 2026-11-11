package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/extensions"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestAccountEmailAndDeviceCancel 核对邮箱透传与设备码取消。
func TestAccountEmailAndDeviceCancel(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "account.login.browser.begin", Result: map[string]any{
				"attemptId": "att-1", "authorizationUrl": "https://dimagent.cn/auth",
			}},
			{Method: "account.login.begin", Result: map[string]any{
				"attemptId": "dev-1", "verificationUri": "https://dimagent.cn/device", "userCode": "ABCD",
			}},
			{Method: "account.login.cancel", Result: map[string]any{}},
			{Method: "account.providers", Result: []any{map[string]any{
				"providerId": "dimcode-api-oauth", "loginMethods": []any{"browser", "device"},
			}}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	auth := &accountAuthSource{wire: result.base.accountWire}

	if _, err := auth.BeginProviderAuthorization(ctx, account.ProviderAuthorizationBegin{
		ProviderID: "dimcode-api-oauth", Directory: "https://dimagent.cn",
		RedirectURI: "http://127.0.0.1:4321/auth/callback", Email: "user@example.invalid",
	}); err != nil {
		t.Fatalf("发起浏览器登录: %v", err)
	}
	beginParams := requestParams(t, server.RequestsFor("account.login.browser.begin")[0])
	if beginParams["email"] != "user@example.invalid" {
		t.Fatalf("浏览器登录应透传 email: %v", beginParams)
	}

	device := &accountDeviceSource{wire: result.base.accountWire}
	if _, err := device.BeginDeviceAuthorizationWithEmail(ctx, "dimcode-api-oauth", "https://dimagent.cn", "user@example.invalid"); err != nil {
		t.Fatalf("发起设备码登录: %v", err)
	}
	deviceParams := requestParams(t, server.RequestsFor("account.login.begin")[0])
	if deviceParams["email"] != "user@example.invalid" {
		t.Fatalf("设备码登录应透传 email: %v", deviceParams)
	}
	if err := device.CancelAuthorization(ctx, "dev-1"); err != nil {
		t.Fatalf("取消设备码尝试: %v", err)
	}
	cancelParams := requestParams(t, server.RequestsFor("account.login.cancel")[0])
	if cancelParams["attemptId"] != "dev-1" || cancelParams["providerId"] == "" {
		t.Fatalf("设备码取消参数不对: %v", cancelParams)
	}
}

// TestLogoutClearsProviderBinding 核对登出成功后清掉 provider 解析缓存。
func TestLogoutClearsProviderBinding(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "account.providers", Result: []any{map[string]any{
				"providerId": "dimcode-api-oauth", "loginMethods": []any{"browser"},
			}}},
			{Method: "account.login.browser.begin", Result: map[string]any{
				"attemptId": "att-1", "authorizationUrl": "https://dimagent.cn/auth",
			}},
			{Method: "account.logout", Result: map[string]any{}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	// 先解析一次 provider，让缓存有值。
	if _, err := (&accountAuthSource{wire: result.base.accountWire}).BeginProviderAuthorization(ctx,
		account.ProviderAuthorizationBegin{ProviderID: "dimcode-api-oauth"}); err != nil {
		t.Fatalf("发起登录: %v", err)
	}
	result.base.accountWire.mu.Lock()
	cached := result.base.accountWire.provider
	result.base.accountWire.mu.Unlock()
	if cached == nil {
		t.Fatal("provider 解析结果应缓存")
	}

	registered, err := result.base.Writes().Register(write.Intent{
		Key: "logout", Kind: write.KindAccountAction,
		Params: json.RawMessage(`{"op":"logout","providerId":"dimcode-api-oauth"}`),
	})
	if err != nil {
		t.Fatalf("登记登出: %v", err)
	}
	settled, err := result.base.Writes().Wait(ctx, registered.ID)
	if err != nil || settled.Status != write.StatusSucceeded {
		t.Fatalf("登出应成功: %v %+v", err, settled)
	}
	result.base.accountWire.mu.Lock()
	after := result.base.accountWire.provider
	result.base.accountWire.mu.Unlock()
	if after != nil {
		t.Fatalf("登出后应清掉 provider 绑定缓存: %+v", after)
	}
}

// TestMCPImportApplyMapsResolutions 核对新版 mcp.import.apply 形状与冲突映射。
func TestMCPImportApplyMapsResolutions(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"mcp.import.apply"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "mcp.import.apply", Result: map[string]any{"revision": "r2"}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	applied, err := result.base.Extensions().ApplyMCPImport(ctx, extensions.MCPImportApplyRequest{
		Cwd:                    "/w",
		Source:                 extensions.SourceUser,
		From:                   "claude-code:user",
		ExpectedVersion:        "r1",
		ExpectedSourceRevision: "s1",
		Resolutions: []extensions.MCPImportResolution{
			{ID: "srv-1", Action: extensions.MCPImportActionImport},
			{ID: "srv-2", Action: extensions.MCPImportActionRename, NewID: "srv-2b"},
		},
	})
	if err != nil {
		t.Fatalf("应用导入: %v", err)
	}
	if applied.Revision != "r2" {
		t.Fatalf("导入回执 revision 不对: %+v", applied)
	}
	if got := len(server.RequestsFor("mcp.import.apply")); got != 1 {
		t.Fatalf("导入应只发一次，实际 %d", got)
	}
	params := requestParams(t, server.RequestsFor("mcp.import.apply")[0])
	if params["source"] != "user" || params["expectedVersion"] != "r1" ||
		params["expectedSourceRevision"] != "s1" || params["from"] != "claude-code:user" {
		t.Fatalf("导入参数不对: %v", params)
	}
	resolutions, _ := params["resolutions"].([]any)
	if len(resolutions) != 2 {
		t.Fatalf("resolutions 数量不对: %v", params)
	}
	first, _ := resolutions[0].(map[string]any)
	second, _ := resolutions[1].(map[string]any)
	if first["id"] != "srv-1" || first["action"] != "import" {
		t.Fatalf("第一条 resolution 不对: %v", first)
	}
	if second["id"] != "srv-2" || second["newId"] != "srv-2b" {
		t.Fatalf("第二条 resolution 不对: %v", second)
	}

	// 冲突：appError=mcp_settings_conflict 归到扩展领域的 ErrImportConflict。
	server2 := startFakeServer(t, fakeappserver.Script{
		Features: []string{"mcp.import.apply"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "mcp.import.apply", Error: fakeappserver.AppError(-32000, "mcp_settings_conflict", "settings changed")},
		},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	if _, err := result2.base.Extensions().ApplyMCPImport(ctx, extensions.MCPImportApplyRequest{
		Source:                 extensions.SourceUser,
		From:                   "claude-code:user",
		ExpectedVersion:        "r1",
		ExpectedSourceRevision: "s1",
		Resolutions:            []extensions.MCPImportResolution{{ID: "srv-1", Action: extensions.MCPImportActionImport}},
	}); !errors.Is(err, extensions.ErrImportConflict) {
		t.Fatalf("冲突应映射成 ErrImportConflict: %v", err)
	}
}
