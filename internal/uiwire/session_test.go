package uiwire

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// twoSessionScript 是两会话的目录脚本：s1 与 s2 各有自己的首屏历史，
// 用来核对两个窗口各自选中一个会话时读到的内容互不串。
func twoSessionScript() fakeappserver.Script {
	history := func(id, text string) []protocol.RawEvent {
		return []protocol.RawEvent{
			e2eRawEvent(id+"-1", 1, "input", "message",
				`{"mode":"queue","user":{"content":[{"type":"text","text":"`+text+`"}]}}`),
			e2eRawEvent(id+"-2", 2, "turn", "start", `{"turn_id":"t1"}`),
			e2eRawEvent(id+"-3", 3, "turn", "end", `{"turn_id":"t1","status":"completed"}`),
		}
	}
	return fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{
			{ID: "s1", Head: 3, LatestTurnStart: 1, Events: history("s1", "第一个窗口的会话")},
			{ID: "s2", Head: 3, LatestTurnStart: 1, Events: history("s2", "第二个窗口的会话")},
		},
		Replies: []fakeappserver.Reply{
			{Method: "session.catalog", Result: map[string]any{"sessions": []any{
				map[string]any{"sessionId": "s1", "cwd": "/work", "createdAt": 1, "updatedAt": 2,
					"running": false, "title": "发送路径"},
				map[string]any{"sessionId": "s2", "cwd": "/work", "createdAt": 1, "updatedAt": 1,
					"running": false, "title": "参考实现"},
			}, "nextCursor": nil}},
			{Method: "session.permission.get", Result: map[string]any{
				"version": 1, "policy": map[string]any{"preset": "workspace-write"},
			}},
		},
	}
}

// TestWindowPinsKeepSeparateSessions 核对多窗口的会话各自独立：两个窗口各自打开
// 一个会话后，各自读到的快照是自己的；App 自己的当前会话不被窗口切换改掉；
// 两个会话各一条订阅（共享同一条控制连接，不重复订阅）。
func TestWindowPinsKeepSeparateSessions(t *testing.T) {
	server := startE2EServer(t, twoSessionScript())
	base := newE2EApp(t, server)
	w := New(base)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	// 连接阶段 App 自己的当前会话是目录首项。
	appBefore := w.currentSession()
	if appBefore != "s1" {
		t.Fatalf("连接后的当前会话 = %q，想要 s1", appBefore)
	}

	first := w.Pin("")
	second := w.Pin("")
	if err := first.OpenSession(ctx, "s1"); err != nil {
		t.Fatalf("第一个窗口打开 s1：%v", err)
	}
	if err := second.OpenSession(ctx, "s2"); err != nil {
		t.Fatalf("第二个窗口打开 s2：%v", err)
	}

	one, ok := first.Session()
	if !ok || one.ID != "s1" {
		t.Fatalf("第一个窗口的会话 = %+v（ok=%t），想要 s1", one, ok)
	}
	two, ok := second.Session()
	if !ok || two.ID != "s2" {
		t.Fatalf("第二个窗口的会话 = %+v（ok=%t），想要 s2", two, ok)
	}
	if firstText(t, one.Rows) != "第一个窗口的会话" || firstText(t, two.Rows) != "第二个窗口的会话" {
		t.Fatalf("两个窗口的历史串了：%q / %q", firstText(t, one.Rows), firstText(t, two.Rows))
	}
	// App 自己的当前会话不因为某个窗口切换而改：主窗口的跟随语义还在。
	if got := w.currentSession(); got != appBefore {
		t.Fatalf("窗口切换改了 App 的当前会话：%q → %q", appBefore, got)
	}
	if got := w.Pin("").ID(); got != appBefore {
		t.Fatalf("没有固定会话的窗口应跟随 App 的当前会话，实际 %q", got)
	}
	// 每个会话一条订阅，没有重复订阅。
	for _, id := range []string{"s1", "s2"} {
		attaches := 0
		for _, request := range server.RequestsFor("session.attach") {
			var params struct {
				SessionID string `json:"sessionId"`
			}
			if err := json.Unmarshal(request.Params, &params); err != nil {
				t.Fatalf("解析 session.attach 参数：%v", err)
			}
			if params.SessionID == id {
				attaches++
			}
		}
		if attaches != 1 {
			t.Fatalf("%s 的订阅数 = %d，想要 1", id, attaches)
		}
	}
}

// firstText 取第一条时间线里的用户消息文本。
func firstText(t *testing.T, rows []ui.RowSnapshot) string {
	t.Helper()
	for _, row := range rows {
		if row.Text != "" {
			return row.Text
		}
	}
	t.Fatalf("时间线没有文本行：%+v", rows)
	return ""
}
