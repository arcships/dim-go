package uiwire

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
)

// 会话与目录的固定事实（形状照抄 internal/app/app_test.go 的装配脚本）。
const e2eSession = "s1"

// e2eEvents 是首屏历史的两行：一条用户、一条助手。
func e2eEvents() []protocol.RawEvent {
	return []protocol.RawEvent{
		e2eRawEvent("e1", 1, "input", "message", `{"mode":"queue","user":{"content":[{"type":"text","text":"帮我看下构建"}]}}`),
		e2eRawEvent("e2", 2, "turn", "start", `{"turn_id":"t1"}`),
		e2eRawEvent("e3", 3, "step", "text", `{"turn_id":"t1","request_id":"r1","delta":"我先看文件"}`),
		e2eRawEvent("e4", 4, "step", "text", `{"turn_id":"t1","request_id":"r1","delta":"，再跑构建"}`),
		e2eRawEvent("e5", 5, "turn", "end", `{"turn_id":"t1","status":"completed"}`),
	}
}

func e2eRawEvent(id string, sequence int64, typ, name, data string) protocol.RawEvent {
	return protocol.RawEvent{
		ID:       id,
		Sequence: sequence,
		CreateAt: 1700000000000 + sequence,
		Type:     typ,
		Name:     name,
		Data:     json.RawMessage(data),
	}
}

// e2eScript 是五步端到端的服务端脚本：目录、权限档位、提交、停止与审批应答。
func e2eScript() fakeappserver.Script {
	return fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              e2eSession,
			LatestTurnStart: 1,
			Events:          e2eEvents(),
		}},
		Replies: []fakeappserver.Reply{
			{Method: "session.catalog", Result: map[string]any{"sessions": []any{map[string]any{
				"sessionId": e2eSession, "cwd": "/work", "createdAt": 1, "updatedAt": 2,
				"running": false, "title": "演示会话",
			}}, "nextCursor": nil}},
			{Method: "session.permission.get", Result: map[string]any{
				"version": 1, "policy": map[string]any{"preset": "full-access"},
			}},
			{Method: "session.submit", Result: map[string]any{"inputId": "srv-input-1"}},
			{Method: "session.stop", Result: map[string]any{}},
			{Method: "approval.respond", Result: map[string]any{"ok": true}},
			{Method: "question.respond", Result: map[string]any{"ok": true}},
		},
	}
}

// TestEndToEndFiveSteps 用假 App Server 走完整五步：
// 列会话 → 打开取历史 → 提交 → 流式事件增长 → 审批出现 / 应答 / 消失。
//
// 脚本的 wire 形状与 internal/app 的装配测试一致；协议路径与真实服务端相同。
func TestEndToEndFiveSteps(t *testing.T) {
	server := startE2EServer(t, e2eScript())
	base := newE2EApp(t, server)
	w := New(base)
	changes := 0
	w.OnChange = func() { changes++ }

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	if conn := w.Connection(); conn.Phase != "connected" {
		t.Fatalf("连接阶段 = %q（%s）", conn.Phase, conn.Reason)
	}

	// ① 目录里的会话来自服务端。
	names := w.SessionNames()
	if len(names) == 0 {
		t.Fatal("① 会话列表为空")
	}
	t.Logf("① 目录会话 %d 个，首项 %q", len(names), names[0])
	if names[0] != e2eSession {
		t.Fatalf("① 首项 = %q，想要 %q", names[0], e2eSession)
	}

	// ② 打开会话后首屏历史成为行。
	snap, ok := w.Session()
	if !ok {
		t.Fatal("② 没有会话快照")
	}
	if snap.ID != e2eSession {
		t.Fatalf("② 快照会话 = %q，想要 %q", snap.ID, e2eSession)
	}
	if len(snap.Rows) < 2 {
		t.Fatalf("② 历史行数 = %d，至少要 2: %+v", len(snap.Rows), snap.Rows)
	}
	if snap.Rows[0].Text != "帮我看下构建" || snap.Rows[len(snap.Rows)-1].Text != "我先看文件，再跑构建" {
		t.Fatalf("② 历史行内容不对: %+v", snap.Rows)
	}
	t.Logf("② 打开 %q，历史 %d 行", snap.ID, len(snap.Rows))

	// ③ 提交成功发到服务端。
	if err := w.Submit(ctx, "再跑一次"); err != nil {
		t.Fatalf("③ 提交失败：%v", err)
	}
	waitFor(t, "session.submit 到达服务端", func() bool {
		return len(server.RequestsFor("session.submit")) > 0
	})
	t.Logf("③ 提交已发出（session.submit %d 次）", len(server.RequestsFor("session.submit")))

	// ④ 服务端推一条实时事实后，时间线行数增长。
	before := len(snap.Rows)
	live := e2eRawEvent("e6", 6, "input", "message",
		`{"user":{"content":[{"type":"text","text":"实时输入"}]}}`)
	if err := server.PushEvent(e2eSession, live); err != nil {
		t.Fatalf("④ 推送实时事实：%v", err)
	}
	waitFor(t, "时间线增长", func() bool {
		cur, ok := w.Session()
		return ok && len(cur.Rows) > before
	})
	grown, _ := w.Session()
	if grown.Rows[len(grown.Rows)-1].Text != "实时输入" {
		t.Fatalf("④ 实时行内容不对: %+v", grown.Rows[len(grown.Rows)-1])
	}
	t.Logf("④ 时间线 %d → %d 行", before, len(grown.Rows))

	// ⑤ 审批请求出现：命令原文来自通知。
	if err := server.Notify("approval.request", e2eApprovalNotice()); err != nil {
		t.Fatalf("⑤ 推送审批请求：%v", err)
	}
	waitFor(t, "审批入队", func() bool {
		_, ok := w.Approval()
		return ok
	})
	approval, ok := w.Approval()
	if !ok {
		t.Fatal("⑤ 审批没有出现")
	}
	if approval.ID != "ap-1" {
		t.Fatalf("⑤ 审批标识 = %q，想要 ap-1", approval.ID)
	}
	// 审批卡上的动作描述来自刚到的那条通知（工具名 + reason），不是样例常量。
	if approval.Command != "exec：启动进程" {
		t.Fatalf("⑤ 审批动作描述 = %q，想要 %q", approval.Command, "exec：启动进程")
	}
	if strings.Contains(approval.Command, "rm -rf") {
		t.Fatalf("⑤ 审批里出现了样例命令: %q", approval.Command)
	}
	t.Logf("⑤ 审批出现：%q", approval.Command)

	if err := w.RespondApproval(ctx, "allow"); err != nil {
		t.Fatalf("⑤ 应答审批失败：%v", err)
	}
	waitFor(t, "approval.respond 到达服务端", func() bool {
		return len(server.RequestsFor("approval.respond")) > 0
	})
	// 服务端结算后通知客户端，待决项消失（真实服务端同样先回执再通知）。
	if err := server.Notify("approval.resolved", map[string]any{
		"sessionId": e2eSession, "branchId": "main", "requestId": "ap-1", "decision": "allow",
	}); err != nil {
		t.Fatalf("⑤ 推送审批结算：%v", err)
	}
	waitFor(t, "审批消失", func() bool {
		_, ok := w.Approval()
		return !ok
	})
	t.Log("⑤ 审批已应答并消失")

	if changes == 0 {
		t.Fatal("全程没有触发 OnChange")
	}
}

// TestDisconnectSurfacesPhase 在服务端断开全部连接后，Source.Connection 必须
// 转成 disconnected 并带上原因；视图据此禁用输入位。
func TestDisconnectSurfacesPhase(t *testing.T) {
	server := startE2EServer(t, e2eScript())
	base := newE2EApp(t, server)
	w := New(base)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	if server.DisconnectAll() == 0 {
		t.Fatal("没有断开任何连接")
	}
	waitFor(t, "连接阶段转为 disconnected", func() bool {
		return w.Connection().Phase == "disconnected"
	})
	conn := w.Connection()
	if conn.Reason == "" {
		t.Fatal("断线没有给出原因")
	}
	t.Logf("断线原因：%s", conn.Reason)
}

// e2eApprovalNotice 是审批请求通知（字段照抄 internal/app/wire_test.go）。
func e2eApprovalNotice() map[string]any {
	return map[string]any{
		"sessionId": e2eSession,
		"branchId":  "main",
		"requestId": "ap-1",
		"source": map[string]any{
			"kind": "tool", "turn_id": "t1", "request_id": "r1", "call_id": "c1",
		},
		"call":   map[string]any{"name": "exec", "input": map[string]any{"command": "printf DIMAGENT_APPROVAL_OK"}},
		"reason": "启动进程",
	}
}

func startE2EServer(t *testing.T, script fakeappserver.Script) *fakeappserver.Server {
	t.Helper()
	server, err := fakeappserver.Start(script)
	if err != nil {
		t.Fatalf("启动假服务端：%v", err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server
}

func newE2EApp(t *testing.T, server *fakeappserver.Server) *app.App {
	t.Helper()
	base, err := app.New(app.Config{
		DataDir:      server.DataDir(),
		ClientName:   "uiwire-e2e",
		SendTimeout:  2 * time.Second,
		CloseTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层：%v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = base.Close(ctx)
	})
	return base
}

// waitFor 轮询条件直到成立；超时即失败并带上最后一次的观察条件。
func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("等待 %s 超时", what)
}
