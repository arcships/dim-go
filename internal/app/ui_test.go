package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
)

// TestUIAttachCloseAndStatus 核对界面注册、关闭与清单读取。
func TestUIAttachCloseAndStatus(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"ui.attach", "ui.close", "ui.status"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "ui.attach", Result: map[string]any{"uiId": "ui-gui-1", "generation": 3}},
			{Method: "ui.close", Result: map[string]any{}},
			{Method: "ui.status", Result: map[string]any{
				"instances": []any{map[string]any{
					"uiId": "ui-gui-1", "sessionId": testSession, "kind": "gui",
					"sinceMs": 1, "generation": 3, "state": "online",
				}},
				"webConnections": 2,
			}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()

	registration, err := result.base.AttachUI(ctx, testSession)
	if err != nil {
		t.Fatalf("注册界面: %v", err)
	}
	if registration.UIID != "ui-gui-1" || registration.Generation != 3 || registration.State != UIRegistrationOnline {
		t.Fatalf("注册回执映射不对: %+v", registration)
	}
	params := requestParams(t, server.RequestsFor("ui.attach")[0])
	if params["sessionId"] != testSession || params["kind"] != "gui" {
		t.Fatalf("注册参数不对: %v", params)
	}
	if _, ok := params["uiId"]; ok {
		t.Fatalf("普通路径不应带 uiId: %v", params)
	}
	stored, ok := result.base.UIRegistration(testSession)
	if !ok || stored.UIID != "ui-gui-1" || stored.Generation != 3 {
		t.Fatalf("注册应保存到会话: %+v ok=%t", stored, ok)
	}

	if err := result.base.CloseUI(ctx, testSession, "test"); err != nil {
		t.Fatalf("关闭界面: %v", err)
	}
	closeParams := requestParams(t, server.RequestsFor("ui.close")[0])
	if closeParams["uiId"] != "ui-gui-1" || closeParams["reason"] != "test" {
		t.Fatalf("关闭参数不对: %v", closeParams)
	}
	if _, ok := result.base.UIRegistration(testSession); ok {
		t.Fatal("关闭后应清理本地注册")
	}

	status, err := result.base.UIStatus(ctx)
	if err != nil {
		t.Fatalf("读取界面清单: %v", err)
	}
	if len(status.Instances) != 1 || status.Instances[0].UIID != "ui-gui-1" || status.WebConnections != 2 {
		t.Fatalf("界面清单映射不对: %+v", status)
	}
}

// TestUIShutdownIsAdvisory 核对 ui.shutdown 只记录、不强制关闭。
func TestUIShutdownIsAdvisory(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"ui.attach"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "ui.attach", Result: map[string]any{"uiId": "ui-gui-1", "generation": 1}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	if _, err := result.base.AttachUI(ctx, testSession); err != nil {
		t.Fatalf("注册界面: %v", err)
	}
	if err := server.Notify("ui.shutdown", map[string]any{
		"uiId": "ui-gui-1", "reason": "server closing",
	}); err != nil {
		t.Fatalf("推送 ui.shutdown: %v", err)
	}
	var events []UIShutdownEvent
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		events = result.base.DrainUIShutdowns()
		if len(events) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(events) != 1 || events[0].UIID != "ui-gui-1" || events[0].Reason != "server closing" {
		t.Fatalf("ui.shutdown 应被记录为建议性关闭: %+v", events)
	}
	registration, ok := result.base.UIRegistration(testSession)
	if !ok || registration.State != UIRegistrationShutdown {
		t.Fatalf("注册状态应标成 shutdown_requested: %+v ok=%t", registration, ok)
	}
	// 不强制关闭：没有发 ui.close，会话仍打开。
	if got := len(server.RequestsFor("ui.close")); got != 0 {
		t.Fatalf("建议性关闭不应自动发 ui.close（%d 次）", got)
	}
}

// TestUIErrorCodesAndReattach 核对失败码分类与 detach/reattach 后注册跟随。
func TestUIErrorCodesAndReattach(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"ui.attach"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "ui.attach", Error: fakeappserver.AppError(-32000, "ui_in_use", "in use")},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	if _, err := result.base.AttachUI(ctx, testSession); UIErrorCode(err) != "ui_in_use" {
		t.Fatalf("应分类为 ui_in_use: %v", err)
	}

	server2 := startFakeServer(t, fakeappserver.Script{
		Features: []string{"ui.attach"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "ui.attach", Error: fakeappserver.AppError(-32000, "session_not_attached", "not attached")},
		},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	if _, err := result2.base.AttachUI(ctx, testSession); UIErrorCode(err) != "session_not_attached" {
		t.Fatalf("应分类为 session_not_attached: %v", err)
	}

	// detach 后重新注册：注册跟随到新会话。
	server3 := startFakeServer(t, fakeappserver.Script{
		Features: []string{"ui.attach"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "ui.attach", Result: map[string]any{"uiId": "ui-gui-9", "generation": 1}},
		},
	})
	result3 := startAssembly(t, server3, assemblyOptions{session: testSession})
	if _, err := result3.base.AttachUI(ctx, testSession); err != nil {
		t.Fatalf("首次注册: %v", err)
	}
	runtime, ok := result3.base.session(testSession)
	if !ok {
		t.Fatal("会话应已打开")
	}
	result3.base.markUIDetached(runtime)
	registration, _ := result3.base.UIRegistration(testSession)
	if registration.State != UIRegistrationDetached {
		t.Fatalf("detach 后状态应为 detached: %+v", registration)
	}
	result3.base.reattachUIAfterResync(runtime)
	registration, _ = result3.base.UIRegistration(testSession)
	if registration.State != UIRegistrationOnline || registration.UIID != "ui-gui-9" {
		t.Fatalf("重新注册后应恢复 online: %+v", registration)
	}
}

// TestUIAttachMissingCapability 核对 hello 未声明时零请求。
func TestUIAttachMissingCapability(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	if _, err := result.base.AttachUI(context.Background(), testSession); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺能力应报能力缺失: %v", err)
	}
	if got := len(server.RequestsFor("ui.attach")); got != 0 {
		t.Fatalf("缺能力时不应请求（%d 次）", got)
	}
}
