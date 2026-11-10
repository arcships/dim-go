package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/extensions"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/host"
	"github.com/arcships/dimagent-mygo/internal/write"
)

const testSession = "s1"

// testVersion 是能通过外壳版本形状校验的客户端版本。
const testVersion = "0.1.0"

// testEvents 是一段最小对话：一轮用户输入、一轮助手文本。
//
// 事件词表来自会话投影：input/message、turn/start、step/text、turn/end。
func testEvents() []protocol.RawEvent {
	return []protocol.RawEvent{
		rawEvent("e1", 1, "input", "message", `{"mode":"queue","user":{"content":[{"type":"text","text":"帮我看下构建"}]}}`),
		rawEvent("e2", 2, "turn", "start", `{"turn_id":"t1"}`),
		rawEvent("e3", 3, "step", "text", `{"turn_id":"t1","request_id":"r1","delta":"我先看文件"}`),
		rawEvent("e4", 4, "step", "text", `{"turn_id":"t1","request_id":"r1","delta":"，再跑构建"}`),
		rawEvent("e5", 5, "turn", "end", `{"turn_id":"t1","status":"completed"}`),
	}
}

func rawEvent(id string, sequence int64, typ, name, data string) protocol.RawEvent {
	return protocol.RawEvent{
		ID:       id,
		Sequence: sequence,
		CreateAt: 1700000000000 + sequence,
		Type:     typ,
		Name:     name,
		Data:     json.RawMessage(data),
	}
}

// startFakeServer 起一个假服务端并返回它；App 的数据目录取假服务端的目录，
// 这样端点、单实例端点与本地状态都在同一处。
func startFakeServer(t *testing.T, script fakeappserver.Script) *fakeappserver.Server {
	t.Helper()
	server, err := fakeappserver.Start(script)
	if err != nil {
		t.Fatalf("启动假服务端: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server
}

// assembly 是一次完整装配：集成层 + 宿主外壳。
type assembly struct {
	base   *App
	shell  *host.Shell
	server *fakeappserver.Server
	diag   *bytes.Buffer
}

// assemblyOptions 控制一次装配的参数。
type assemblyOptions struct {
	// session 是订阅阶段要打开的会话；空表示由订阅阶段读目录决定。
	session string
	// diag 非空时收集进程级诊断。
	diag *bytes.Buffer
}

// startAssembly 装配并走完六阶段启动；失败即测试失败。
func startAssembly(t *testing.T, server *fakeappserver.Server, options assemblyOptions) *assembly {
	t.Helper()
	result := buildAssembly(t, server, options)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	outcome, err := result.shell.Start(ctx)
	if err != nil {
		shutdownAndWait(t, result.shell)
		t.Fatalf("启动失败（阶段 %s）: %v", result.shell.State().Stage, err)
	}
	if outcome != host.StartPrimary {
		t.Fatalf("首次启动应为主实例，实际 %s", outcome)
	}
	if state := result.shell.State(); state.Stage != host.StageReady {
		t.Fatalf("启动阶段应为 ready，实际 %s（%v）", state.Stage, state.Failed)
	}
	t.Cleanup(func() { shutdownAndWait(t, result.shell) })
	return result
}

// buildAssembly 只装配不启动。
func buildAssembly(t *testing.T, server *fakeappserver.Server, options assemblyOptions) *assembly {
	t.Helper()
	base, err := New(Config{
		DataDir:          server.DataDir(),
		ClientName:       "dimagent-mygo-test",
		InitialSessionID: options.session,
		Version:          testVersion,
		SendTimeout:      2 * time.Second,
		Verify:           write.VerifyPolicy{MaxAttempts: 1, Interval: time.Millisecond, Timeout: time.Second},
		CloseTimeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	recorder := host.NewRecorder(io.Discard)
	if options.diag != nil {
		recorder = host.NewRecorder(options.diag)
	}
	shell, err := NewHost(base, HostOptions{
		DataDir: server.DataDir(),
		Args:    []string{"selfcheck"},
		Version: testVersion,
		Diag:    recorder,
	})
	if err != nil {
		t.Fatalf("接到宿主外壳: %v", err)
	}
	return &assembly{base: base, shell: shell, server: server, diag: options.diag}
}

// shutdownAndWait 走完退出五阶段。
func shutdownAndWait(t *testing.T, shell *host.Shell) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := shell.Shutdown(ctx); err != nil {
		t.Logf("退出未全部完成: %v", err)
	}
}

// TestHostWiringStartupAndShutdown 核对接线：六阶段全部完成、订阅阶段打开了会话、
// 写闸门放行、退出五阶段按序走完。
func TestHostWiringStartupAndShutdown(t *testing.T) {
	var diag bytes.Buffer
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              testSession,
			LatestTurnStart: 1,
			Events:          testEvents(),
		}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession, diag: &diag})

	if phase := result.base.Phase(); phase != PhaseReady {
		t.Fatalf("启动完成后阶段应为 ready，实际 %s", phase)
	}
	if got := result.base.StartupSession(); got != testSession {
		t.Fatalf("订阅阶段应打开 %s，实际 %q", testSession, got)
	}
	if err := result.base.Writable(); err != nil {
		t.Fatalf("就绪后写闸门应放行: %v", err)
	}
	if err := result.shell.Writable(); err != nil {
		t.Fatalf("外壳写闸门应放行: %v", err)
	}
	view, ok := result.base.View(testSession)
	if !ok {
		t.Fatal("订阅阶段的会话不在打开列表里")
	}
	if view.LoadPhase != "ready" || len(view.Rows) != 2 {
		t.Fatalf("首轮历史没有折叠进来: load=%s rows=%d", view.LoadPhase, len(view.Rows))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	report, err := result.shell.Shutdown(ctx)
	if err != nil {
		t.Fatalf("退出五阶段失败: %v", err)
	}
	want := host.ShutdownPhases()
	if len(report.Results) != len(want) {
		t.Fatalf("退出应走 %d 段，实际 %d", len(want), len(report.Results))
	}
	for index, phase := range report.Results {
		if phase.Phase != want[index] {
			t.Fatalf("第 %d 段应为 %s，实际 %s", index+1, want[index], phase.Phase)
		}
	}
	if !report.Complete() {
		t.Fatalf("退出应全部完成，实际超时 %v 失败 %v", report.TimedOut(), report.Failed())
	}
	if state := result.shell.State(); !state.Exited {
		t.Fatalf("退出后状态应为已退出: %+v", state)
	}
	deadline := time.Now().Add(3 * time.Second)
	for server.Connections() != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("退出后服务端仍有 %d 条连接", server.Connections())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(diag.String(), `"event":"startup.ready"`) {
		t.Fatalf("诊断里缺少 startup.ready: %s", diag.String())
	}
	if !strings.Contains(diag.String(), `"event":"shutdown.done"`) {
		t.Fatalf("诊断里缺少 shutdown.done: %s", diag.String())
	}
}

// TestAssemblyOpenSessionAndSubmit 走一次完整装配里的读写：历史行 → 提交 → 回执。
func TestAssemblyOpenSessionAndSubmit(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              testSession,
			LatestTurnStart: 1,
			Events:          testEvents(),
		}},
		Replies: []fakeappserver.Reply{
			{Method: "session.submit", Result: map[string]any{"inputId": "srv-input-1"}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	base := result.base
	ctx := context.Background()

	view, err := base.OpenSession(ctx, testSession)
	if err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	if len(view.Rows) != 2 {
		t.Fatalf("行数应为 2，实际 %d: %+v", len(view.Rows), view.Rows)
	}
	if view.Rows[0].ID != "e1" || view.Rows[0].Kind != "user_message" || view.Rows[0].Text != "帮我看下构建" {
		t.Fatalf("用户行不对: %+v", view.Rows[0])
	}
	if view.Rows[1].ID != "e3" || view.Rows[1].Kind != "assistant_text" || view.Rows[1].Text != "我先看文件，再跑构建" {
		t.Fatalf("助手行不对: %+v", view.Rows[1])
	}

	intent, err := base.Submit(ctx, SubmitRequest{SessionID: testSession, Text: "再跑一次"})
	if err != nil {
		t.Fatalf("登记提交意图: %v", err)
	}
	if intent.InputID == "" {
		t.Fatal("提交意图没有输入标识")
	}
	settled, err := base.Wait(ctx, intent.ID)
	if err != nil {
		t.Fatalf("等待提交收敛: %v", err)
	}
	if settled.Status != write.StatusSucceeded {
		t.Fatalf("提交应成功，实际 %s（%s）", settled.Status, settled.Reason)
	}
	if settled.InputID != "srv-input-1" {
		t.Fatalf("回执里的输入标识应取服务端值，实际 %q", settled.InputID)
	}

	requests := server.RequestsFor("session.submit")
	if len(requests) != 1 {
		t.Fatalf("一次提交只应发一次请求，实际 %d", len(requests))
	}
	params := requestParams(t, requests[0])
	if params["sessionId"] != testSession || params["branchId"] != mainBranchID {
		t.Fatalf("提交没有带会话或主线: %v", params)
	}
	if params["mode"] != "queue" || params["text"] != "再跑一次" {
		t.Fatalf("提交参数不对: %v", params)
	}
	if params["submissionId"] != intent.InputID {
		t.Fatalf("提交去重标识应为意图的输入标识: %v", params)
	}
	if len(intent.InputID) != 36 || intent.InputID[8] != '-' {
		t.Fatalf("提交去重标识应为 UUID，实际 %q", intent.InputID)
	}
}

// TestLiveEventLandsIntoRows 核对实时通知走同一份折叠，重读后尾部行换成服务端标识。
func TestLiveEventLandsIntoRows(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              testSession,
			LatestTurnStart: 1,
			Events:          testEvents(),
		}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()

	live := rawEvent("e6", 6, "input", "message", `{"user":{"content":[{"type":"text","text":"实时输入"}]}}`)
	if err := server.PushEvent(testSession, live); err != nil {
		t.Fatalf("推送实时事实: %v", err)
	}
	waitRows(t, result.base, testSession, 3)
	view, _ := result.base.View(testSession)
	last := view.Rows[len(view.Rows)-1]
	if last.ID != "" || !last.Pending {
		t.Fatalf("实时事实应先是未落库行: %+v", last)
	}
	if last.Text != "实时输入" {
		t.Fatalf("实时行内容不对: %+v", last)
	}

	updated, err := result.base.ReloadLatest(ctx, testSession)
	if err != nil {
		t.Fatalf("重读历史: %v", err)
	}
	if len(updated.Rows) != 3 {
		t.Fatalf("重读后行数应为 3，实际 %d", len(updated.Rows))
	}
	landed := updated.Rows[2]
	if landed.ID != "e6" || landed.Pending || landed.Text != "实时输入" {
		t.Fatalf("重读后应原地换成服务端标识: %+v", landed)
	}
}

// waitRows 等到视图的行数达到期望值；超时即失败。
func waitRows(t *testing.T, base *App, sessionID string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		view, ok := base.View(sessionID)
		if ok && len(view.Rows) >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待行数 %d 超时，当前 %+v", want, view.Rows)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestSenderMapsIntentKinds 核对 Sender 的映射表：每种写意图落到哪个方法。
func TestSenderMapsIntentKinds(t *testing.T) {
	pluginParams, err := extensions.ManagementParams(
		extensions.OpPluginSetEnabled,
		extensions.PluginEnabledPayload{PluginID: "demo", Enabled: true},
	)
	if err != nil {
		t.Fatalf("编码管理写入参数: %v", err)
	}
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession}},
		Replies: []fakeappserver.Reply{
			{Method: "approval.respond", Result: map[string]any{"ok": true}},
			{Method: "approval.allow_always", Result: map[string]any{"ok": true}},
			{Method: "question.respond", Result: map[string]any{"ok": true}},
			{Method: "session.stop", Result: map[string]any{"stopped": true}},
			{Method: "plugin.set_enabled", Result: map[string]any{"pluginId": "demo"}},
			{Method: "settings.update", Result: map[string]any{"revision": "r2"}},
			{Method: "session.delete", Result: nil},
			{Method: "account.logout", Result: map[string]any{"ok": true}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	writes := result.base.Writes()

	cases := []struct {
		name   string
		intent write.Intent
		method string
		check  func(t *testing.T, params map[string]any)
	}{
		{
			name: "approval response",
			intent: write.Intent{
				Key: "k-approval", Kind: write.KindApprovalResponse,
				SessionID: testSession, BranchID: "main", InputID: "a1",
				Params: json.RawMessage(`{"decision":"approve"}`),
			},
			method: "approval.respond",
			check: func(t *testing.T, params map[string]any) {
				if params["requestId"] != "a1" || params["decision"] != "approve" || params["sessionId"] != testSession {
					t.Fatalf("审批应答参数不对: %v", params)
				}
			},
		},
		{
			name: "long term allow",
			intent: write.Intent{
				Key: "k-longterm", Kind: write.KindLongTermAllow,
				SessionID: testSession, InputID: "a1",
				Params: json.RawMessage(`{"tool":"Bash"}`),
			},
			method: "approval.allow_always",
			check: func(t *testing.T, params map[string]any) {
				if params["requestId"] != "a1" || params["tool"] != "Bash" {
					t.Fatalf("长期规则参数不对: %v", params)
				}
			},
		},
		{
			name: "question answer",
			intent: write.Intent{
				Key: "k-question", Kind: write.KindQuestionAnswer,
				SessionID: testSession, InputID: "q1",
				Params: json.RawMessage(`{"decision":"answer","answer":{"choice":["继续"]}}`),
			},
			method: "question.respond",
			check: func(t *testing.T, params map[string]any) {
				if params["requestId"] != "q1" {
					t.Fatalf("提问应答没有带请求标识: %v", params)
				}
			},
		},
		{
			name:   "stop",
			intent: write.Intent{Key: "k-stop", Kind: write.KindStop, SessionID: testSession},
			method: "session.stop",
			check: func(t *testing.T, params map[string]any) {
				if params["sessionId"] != testSession {
					t.Fatalf("停止没有带会话标识: %v", params)
				}
			},
		},
		{
			name: "management write",
			intent: write.Intent{
				Key: "k-ext", Kind: write.KindManagementWrite, Params: pluginParams,
			},
			method: "plugin.set_enabled",
			check: func(t *testing.T, params map[string]any) {
				if params["pluginId"] != "demo" || params["enabled"] != true {
					t.Fatalf("插件启停参数不对: %v", params)
				}
			},
		},
		{
			name: "settings update",
			intent: write.Intent{
				Key: "k-settings", Kind: write.KindManagementWrite,
				Params: json.RawMessage(`{"op":"settings.update","source":"execution","scope":{"kind":"user"},"expectedRevision":"r1","document":{"sandbox":false}}`),
			},
			method: "settings.update",
			check: func(t *testing.T, params map[string]any) {
				if params["source"] != "execution" || params["scope"] != "user" || params["expectedRevision"] != "r1" {
					t.Fatalf("设置写入参数不对: %v", params)
				}
			},
		},
		{
			name: "session lifecycle",
			intent: write.Intent{
				Key: "k-delete", Kind: write.KindSessionLifecycle,
				SessionID: testSession, Params: json.RawMessage(`{"op":"catalog.delete"}`),
			},
			method: "session.delete",
			check: func(t *testing.T, params map[string]any) {
				if params["sessionId"] != testSession {
					t.Fatalf("删除没有带会话标识: %v", params)
				}
			},
		},
		{
			name: "account action",
			intent: write.Intent{
				Key: "k-logout", Kind: write.KindAccountAction,
				Params: json.RawMessage(`{"op":"logout"}`),
			},
			method: "account.logout",
			check:  func(t *testing.T, params map[string]any) {},
		},
	}

	ctx := context.Background()
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			registered, err := writes.Register(testCase.intent)
			if err != nil {
				t.Fatalf("登记意图: %v", err)
			}
			settled, err := writes.Wait(ctx, registered.ID)
			if err != nil {
				t.Fatalf("等待收敛: %v", err)
			}
			if settled.Status != write.StatusSucceeded {
				t.Fatalf("应成功，实际 %s（%s）", settled.Status, settled.Reason)
			}
			requests := server.RequestsFor(testCase.method)
			if len(requests) != 1 {
				t.Fatalf("%s 请求数应为 1，实际 %d", testCase.method, len(requests))
			}
			testCase.check(t, requestParams(t, requests[0]))
		})
	}
}

// TestInspectorLooksUpInputThroughProjection 核对 Inspector 的三条结论。
func TestInspectorLooksUpInputThroughProjection(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              testSession,
			LatestTurnStart: 1,
			Events:          testEvents(),
		}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	inspector := &appInspector{app: result.base}

	hit, err := inspector.LookupInput(ctx, write.InputQuery{SessionID: testSession, InputID: "e1"})
	if err != nil {
		t.Fatalf("核对命中: %v", err)
	}
	if !hit.Confirmed || !hit.Present || hit.BranchID != "main" {
		t.Fatalf("已落库的输入应命中且归属主线: %+v", hit)
	}

	miss, err := inspector.LookupInput(ctx, write.InputQuery{SessionID: testSession, InputID: "never-sent"})
	if err != nil {
		t.Fatalf("核对未发生: %v", err)
	}
	if !miss.Confirmed || miss.Present {
		t.Fatalf("读完历史仍未出现应确认未发生: %+v", miss)
	}

	unknown, err := inspector.LookupInput(ctx, write.InputQuery{SessionID: "not-open", InputID: "x"})
	if err != nil {
		t.Fatalf("核对未打开会话: %v", err)
	}
	if unknown.Confirmed {
		t.Fatalf("会话没打开时应为无法确认: %+v", unknown)
	}
}

// TestCatalogAssembles 核对目录读取方与订阅阶段的目录决定。
func TestCatalogAssembles(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession}},
		Replies: []fakeappserver.Reply{{
			Method: "session.catalog",
			Result: map[string]any{"sessions": []any{map[string]any{
				"sessionId": testSession,
				"cwd":       "/work/repo",
				"running":   false,
				"title":     "构建排查",
			}}},
		}},
	})
	// 启动时不指定会话：订阅阶段读目录取第一条。
	result := startAssembly(t, server, assemblyOptions{})
	if got := result.base.StartupSession(); got != testSession {
		t.Fatalf("订阅阶段应从目录取第一条，实际 %q", got)
	}

	load, err := result.base.Catalog().Refresh(context.Background(), catalog.Query{})
	if err != nil {
		t.Fatalf("读取目录: %v", err)
	}
	if load.Added != 1 {
		t.Fatalf("目录应加入 1 条，实际 %d", load.Added)
	}
	entry, ok := result.base.Catalog().Entry(testSession)
	if !ok {
		t.Fatal("目录里没有该会话")
	}
	if entry.Workspace != "/work/repo" || entry.Title != "构建排查" {
		t.Fatalf("条目映射不对: %+v", entry)
	}

	hit, err := (&appInspector{app: result.base}).LookupInput(
		context.Background(), write.InputQuery{InputID: testSession})
	if err != nil {
		t.Fatalf("核对新建: %v", err)
	}
	if !hit.Confirmed || !hit.Present {
		t.Fatalf("目录里已有的会话应命中: %+v", hit)
	}
}

// TestSecondLaunchForwardsAndSkipsShutdown 核对二次启动：参数转交、状态停在 idle、
// 不进入退出五阶段。
func TestSecondLaunchForwardsAndSkipsShutdown(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	primary := startAssembly(t, server, assemblyOptions{session: testSession})

	var secondDiag bytes.Buffer
	second := buildAssembly(t, server, assemblyOptions{session: testSession, diag: &secondDiag})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	outcome, err := second.shell.Start(ctx)
	if err != nil {
		t.Fatalf("二次启动应转交成功: %v", err)
	}
	if outcome != host.StartForwarded {
		t.Fatalf("二次启动应为 forwarded，实际 %s", outcome)
	}

	// 状态停在 idle：本进程没有推进任何阶段，也没有进入退出。
	state := second.shell.State()
	if state.Stage != host.StageIdle {
		t.Fatalf("二次启动的阶段应停在 idle，实际 %s", state.Stage)
	}
	if state.Exiting || state.Exited {
		t.Fatalf("二次启动不应进入退出: %+v", state)
	}
	if phase := second.base.Phase(); phase != PhaseInitial {
		t.Fatalf("二次启动不应装载任何本地状态，实际阶段 %s", phase)
	}
	if strings.Contains(secondDiag.String(), "shutdown.") {
		t.Fatalf("二次启动不应写退出诊断: %s", secondDiag.String())
	}
	if !strings.Contains(secondDiag.String(), `"event":"instance.forwarded"`) {
		t.Fatalf("诊断里缺少 instance.forwarded: %s", secondDiag.String())
	}

	// 主实例收到了这次转交的参数。
	select {
	case activation := <-primary.shell.Activations():
		if len(activation.Args) == 0 {
			t.Fatal("转交的参数为空")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("主实例没有收到转交的参数")
	}
}

// TestStartupStopsAtFailingStage 核对启动失败：停在失败阶段、如实报、不推进后续。
func TestStartupStopsAtFailingStage(t *testing.T) {
	// 没有服务端可执行文件、数据目录里也没有端点：连服务阶段必然失败。
	// 数据目录放在 /tmp 下：Unix 单实例端点有路径长度上限。
	dataDir, err := os.MkdirTemp("/tmp", "dimagent-app-stage-")
	if err != nil {
		t.Fatalf("创建临时数据目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	base, err := New(Config{
		DataDir:     dataDir,
		Version:     testVersion,
		SendTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	var diag bytes.Buffer
	shell, err := NewHost(base, HostOptions{
		DataDir: dataDir,
		Version: testVersion,
		Diag:    host.NewRecorder(&diag),
	})
	if err != nil {
		t.Fatalf("接到宿主外壳: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	outcome, err := shell.Start(ctx)
	if err == nil {
		t.Fatal("连不上服务端时启动应失败")
	}
	if outcome != host.StartPrimary {
		t.Fatalf("首次启动应为主实例，实际 %s", outcome)
	}
	state := shell.State()
	if state.Stage != host.StageConnect {
		t.Fatalf("启动应停在 connect 阶段，实际 %s", state.Stage)
	}
	if state.Failed == nil {
		t.Fatal("失败阶段应带上错误")
	}
	if phase := base.Phase(); phase != PhaseLocalState {
		t.Fatalf("连服务失败后应停在本地状态阶段，实际 %s", phase)
	}
	// 未就绪：写闸门与提交入口都拒绝，且不把"未连接"当成空数据。
	if err := shell.Writable(); err == nil {
		t.Fatal("未就绪时外壳写闸门应拒绝")
	}
	if _, err := base.Submit(ctx, SubmitRequest{SessionID: testSession, Text: "不该发出"}); !errors.Is(err, ErrNotReady) {
		t.Fatalf("未就绪时提交应被拒: %v", err)
	}
	if !strings.Contains(diag.String(), `"event":"startup.failed"`) ||
		!strings.Contains(diag.String(), `"stage":"connect"`) {
		t.Fatalf("诊断应记录停在 connect 阶段: %s", diag.String())
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	report, shutdownErr := shell.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		t.Fatalf("启动失败后仍应能按阶段退出: %v", shutdownErr)
	}
	if len(report.Results) != len(host.ShutdownPhases()) {
		t.Fatalf("退出应走 %d 段，实际 %d", len(host.ShutdownPhases()), len(report.Results))
	}
}

// TestShutdownTimeoutContinues 核对退出超时：一段超时被记录后继续走完后续阶段。
func TestShutdownTimeoutContinues(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	base, err := New(Config{
		DataDir:          server.DataDir(),
		InitialSessionID: testSession,
		Version:          testVersion,
		SendTimeout:      time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	timeouts := host.DefaultTimeouts()
	timeouts.EndInteraction = 10 * time.Millisecond
	var diag bytes.Buffer
	shell, err := host.New(host.Config{
		DataDir:   server.DataDir(),
		Version:   testVersion,
		Diag:      host.NewRecorder(&diag),
		Connected: base.Connected,
		Timeouts:  timeouts,
		Startup: host.StartupSteps{
			LocalState:   base.LoadLocalState,
			Connect:      base.Connect,
			Subscribe:    base.Subscribe,
			FirstHistory: base.FirstHistory,
			AllowWrites:  base.AllowWrites,
		},
		Shutdown: host.ShutdownSteps{
			StopIntents: base.StopIntents,
			// 故意不响应上下文：外壳必须按时进入下一段。
			EndInteraction: func(context.Context) error {
				time.Sleep(300 * time.Millisecond)
				return base.EndInteraction(context.Background())
			},
			CloseConnection: base.CloseConnection,
			ReclaimService:  base.ReclaimService,
		},
	})
	if err != nil {
		t.Fatalf("接到宿主外壳: %v", err)
	}
	base.SetWriteGate(shell.Writable)

	startCtx, startCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer startCancel()
	if _, err := shell.Start(startCtx); err != nil {
		t.Fatalf("启动: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	report, err := shell.Shutdown(shutdownCtx)
	if err == nil {
		t.Fatal("有阶段超时时应返回合并错误")
	}
	timedOut := report.TimedOut()
	if len(timedOut) != 1 || timedOut[0] != host.PhaseEndInteraction {
		t.Fatalf("超时阶段应为 end_interaction，实际 %v", timedOut)
	}
	if len(report.Results) != len(host.ShutdownPhases()) {
		t.Fatalf("超时后仍应走完五段，实际 %d", len(report.Results))
	}
	for _, phase := range report.Failed() {
		t.Fatalf("除超时外不应有失败阶段: %v", phase)
	}
	if state := shell.State(); !state.Exited {
		t.Fatalf("超时后仍应标记退出完成: %+v", state)
	}
	if !strings.Contains(diag.String(), `"event":"shutdown.timeout"`) {
		t.Fatalf("诊断里应记录超时: %s", diag.String())
	}
	// 后续阶段确实执行过：连接已关闭。
	if base.Connected() {
		t.Fatal("超时后仍应关闭连接")
	}
}

// TestDiagnosticsWhitelistOnly 核对日志白名单：凭据与会话正文进不来，字段集合固定。
func TestDiagnosticsWhitelistOnly(t *testing.T) {
	const credential = "sk-live-abcdef123456"
	const body = "我的私密正文：这是不该进日志的内容"
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              testSession,
			LatestTurnStart: 1,
			Events:          testEvents(),
		}},
		Replies: []fakeappserver.Reply{
			{Method: "session.submit", Result: map[string]any{"inputId": "srv-input-1"}},
		},
	})
	var diag bytes.Buffer
	result := startAssembly(t, server, assemblyOptions{session: testSession, diag: &diag})

	ctx := context.Background()
	if _, err := result.base.Submit(ctx, SubmitRequest{
		SessionID: testSession,
		Text:      body + " token=" + credential,
	}); err != nil {
		t.Fatalf("提交意图: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if _, err := result.shell.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("退出五阶段: %v", err)
	}

	output := diag.String()
	if strings.Contains(output, credential) || strings.Contains(output, "这是不该进日志的内容") {
		t.Fatalf("诊断里出现了凭据或正文: %s", output)
	}
	if !strings.Contains(output, `"event":"startup.stage"`) || !strings.Contains(output, `"event":"shutdown.phase"`) {
		t.Fatalf("诊断缺少阶段记录: %s", output)
	}
	allowed := map[string]bool{
		"time": true, "level": true, "event": true,
		"stage": true, "from_stage": true, "next_stage": true, "phase": true,
		"outcome": true, "classification": true,
		"connected": true, "exiting": true, "version": true,
		"elapsed_ms": true, "timeout_ms": true, "args": true, "dropped": true,
	}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("诊断行不是 JSON: %s (%v)", line, err)
		}
		for key := range record {
			if !allowed[key] {
				t.Fatalf("诊断行出现白名单之外的字段 %q: %s", key, line)
			}
		}
	}
}

// TestWriteGateBeforeReadyAndWhileExiting 核对写闸门：未就绪与退出中都拒绝新意图。
func TestWriteGateBeforeReadyAndWhileExiting(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	// 不接宿主外壳：这里验证的是 App 自己的阶段标记与写闸门。
	base, err := New(Config{
		DataDir:          server.DataDir(),
		InitialSessionID: testSession,
		Version:          testVersion,
		SendTimeout:      time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	ctx := context.Background()
	t.Cleanup(func() { _ = base.Close(context.Background()) })

	// 只走到连服务：允许写阶段还没有执行。
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	if err := base.Connect(ctx); err != nil {
		t.Fatalf("连接服务端: %v", err)
	}
	if err := base.Writable(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("未就绪时写闸门应报未就绪: %v", err)
	}
	if _, err := base.Submit(ctx, SubmitRequest{SessionID: testSession, Text: "不该发出"}); !errors.Is(err, ErrNotReady) {
		t.Fatalf("未就绪时提交应被拒: %v", err)
	}
	if requests := server.RequestsFor("session.submit"); len(requests) != 0 {
		t.Fatalf("被拒绝的提交不应发出请求: %d", len(requests))
	}

	// 补齐启动阶段后再退出：退出第一阶段之后的新意图一律被拒。
	for _, step := range []func(context.Context) error{
		base.Subscribe, base.FirstHistory, base.AllowWrites,
	} {
		if err := step(ctx); err != nil {
			t.Fatalf("补齐启动阶段: %v", err)
		}
	}
	if err := base.Writable(); err != nil {
		t.Fatalf("就绪后写闸门应放行: %v", err)
	}
	if err := base.StopIntents(ctx); err != nil {
		t.Fatalf("停止新意图: %v", err)
	}
	if err := base.Writable(); !errors.Is(err, ErrExiting) {
		t.Fatalf("退出中写闸门应报退出中: %v", err)
	}
	if _, err := base.Submit(ctx, SubmitRequest{SessionID: testSession, Text: "不该发出"}); !errors.Is(err, ErrExiting) {
		t.Fatalf("退出中提交应被拒: %v", err)
	}
	if requests := server.RequestsFor("session.submit"); len(requests) != 0 {
		t.Fatalf("退出中的提交不应发出请求: %d", len(requests))
	}
}

// TestReadPathsWorkAfterReady 核对读路径不受写闸门影响，且各领域都装配上了。
func TestReadPathsWorkAfterReady(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.catalog", Result: map[string]any{"sessions": []any{map[string]any{
				"sessionId": testSession,
			}}}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	if result.base.Catalog() == nil || result.base.Account() == nil ||
		result.base.Settings() == nil || result.base.Extensions() == nil {
		t.Fatal("目录、账号、设置或扩展领域没有装配")
	}
	if _, err := result.base.Catalog().Refresh(context.Background(), catalog.Query{}); err != nil {
		t.Fatalf("读取目录: %v", err)
	}
}

func requestParams(t *testing.T, request fakeappserver.Request) map[string]any {
	t.Helper()
	params := make(map[string]any)
	if len(request.Params) == 0 {
		return params
	}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		t.Fatalf("请求参数不是 JSON 对象: %s (%v)", request.Params, err)
	}
	return params
}
