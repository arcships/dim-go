package uiwire

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// rowsEvents 是一轮完整的事实：思考 → 正文 → 两次工具调用（一次读、一次写）→
// 收口。工作区的写声明用来核对改动块。
func rowsEvents() []protocol.RawEvent {
	return []protocol.RawEvent{
		e2eRawEvent("e1", 1, "input", "message",
			`{"mode":"queue","user":{"content":[{"type":"text","text":"接上工具行"}]}}`),
		e2eRawEvent("e2", 2, "turn", "start", `{"turn_id":"t1"}`),
		e2eRawEvent("e3", 3, "step", "thinking",
			`{"turn_id":"t1","request_id":"r1","delta":"先读时间线"}`),
		e2eRawEvent("e4", 4, "step", "text",
			`{"turn_id":"t1","request_id":"r1","delta":"我把工具行接上了。"}`),
		e2eRawEvent("e5", 5, "step", "tool-use",
			`{"turn_id":"t1","request_id":"r1","call":{"id":"c-read","name":"read_file","input":{"path":"internal/ui/timeline.go"}}}`),
		e2eRawEvent("e6", 6, "step", "tool-completed",
			`{"turn_id":"t1","request_id":"r1","call_id":"c-read","result":{"content":"读到了"}}`),
		e2eRawEvent("e7", 7, "step", "tool-use",
			`{"turn_id":"t1","request_id":"r1","call":{"id":"c-write","name":"write","input":{"path":"internal/ui/source.go"}}}`),
		e2eRawEvent("e8", 8, "step", "tool-completed",
			`{"turn_id":"t1","request_id":"r1","call_id":"c-write","result":{"structuredContent":{"path":"internal/ui/source.go"}}}`),
		e2eRawEvent("e9", 9, "turn", "end", `{"turn_id":"t1","status":"completed"}`),
		// 服务端结算这条输入：用户行的状态从 pending 变 done。
		e2eRawEvent("e10", 10, "input", "consumed", `{"input_id":"e1"}`),
	}
}

// rowsScript 是给行投射用的服务端脚本：一条会话 + 目录与权限档位的回执。
func rowsScript() fakeappserver.Script {
	return fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              e2eSession,
			LatestTurnStart: 1,
			Events:          rowsEvents(),
		}},
		Replies: []fakeappserver.Reply{
			{Method: "session.catalog", Result: map[string]any{"sessions": []any{map[string]any{
				"sessionId": e2eSession, "cwd": "/work", "createdAt": 1, "updatedAt": 2,
				"running": false, "title": "行投射会话",
			}}, "nextCursor": nil}},
			{Method: "session.permission.get", Result: map[string]any{
				"version": 1, "policy": map[string]any{"preset": "full-access"},
			}},
		},
	}
}

// newRowsApp 建一个带工作区根的集成层：改动读取需要工作区才能规范化路径。
func newRowsApp(t *testing.T, server *fakeappserver.Server, root string) *app.App {
	t.Helper()
	base, err := app.New(app.Config{
		DataDir:       server.DataDir(),
		ClientName:    "uiwire-rows",
		WorkspaceRoot: root,
		SendTimeout:   2 * time.Second,
		CloseTimeout:  2 * time.Second,
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

// TestSessionRowsProjectFacts 用假服务端造一轮「思考 + 工具调用 + 文件改动」，
// 断言界面拿到的行种类与字段来自事实，而不是来自样例。
func TestSessionRowsProjectFacts(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "ui"), 0o755); err != nil {
		t.Fatalf("准备工作区目录：%v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "ui", "source.go"),
		[]byte("package ui\n"), 0o600); err != nil {
		t.Fatalf("准备工作区文件：%v", err)
	}
	server := startE2EServer(t, rowsScript())
	base := newRowsApp(t, server, root)
	w := New(base)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	// 改动块要等一次后台读取（声明来自历史里的工具结果），等到它出现为止。
	waitFor(t, "改动块到达时间线", func() bool {
		snap, ok := w.Session()
		if !ok {
			return false
		}
		for _, row := range snap.Rows {
			if row.Kind == ui.RowFileChanges {
				return true
			}
		}
		return false
	})

	snap, ok := w.Session()
	if !ok {
		t.Fatal("没有会话快照")
	}
	kinds := make([]string, 0, len(snap.Rows))
	for _, row := range snap.Rows {
		kinds = append(kinds, row.Kind)
	}
	want := []string{ui.RowUser, ui.RowThinking, ui.RowAssistant, ui.RowToolGroup, ui.RowFileChanges}
	if len(kinds) != len(want) {
		t.Fatalf("行种类 = %v，想要 %v（全部行：%+v）", kinds, want, snap.Rows)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("第 %d 行种类 = %q，想要 %q（全部行：%v）", i, kinds[i], want[i], kinds)
		}
	}

	// ① 用户行：正文与状态都来自输入事实。
	user := snap.Rows[0]
	if user.Text != "接上工具行" || user.Status != ui.RowStatusDone {
		t.Fatalf("用户行 = %+v", user)
	}
	// ② 思考行：正文是思考增量，时长来自这条事实的起止时间。
	thinking := snap.Rows[1]
	if thinking.Text != "先读时间线" || thinking.Duration <= 0 || thinking.Status != ui.RowStatusDone {
		t.Fatalf("思考行 = %+v", thinking)
	}
	// ③ 助手行：正文是文本增量。
	if assistant := snap.Rows[2]; assistant.Text != "我把工具行接上了。" {
		t.Fatalf("助手行 = %+v", assistant)
	}
	// ④ 工具组：两次调用合成一组，子行的动作与参数来自工具名与调用输入。
	group := snap.Rows[3]
	if len(group.Children) != 2 {
		t.Fatalf("工具组子行 = %+v", group.Children)
	}
	read, write := group.Children[0], group.Children[1]
	if read.Title != "读取" || read.Args != "internal/ui/timeline.go" || read.Text != "读到了" {
		t.Fatalf("读取行 = %+v", read)
	}
	if read.Status != ui.RowStatusDone {
		t.Fatalf("读取行状态 = %q，想要 done", read.Status)
	}
	if write.Title != "写入" || write.Args != "internal/ui/source.go" || write.Status != ui.RowStatusDone {
		t.Fatalf("写入行 = %+v", write)
	}
	// ⑤ 改动块：路径来自这一轮工具结果声明的改动；差异行数事实里没有，留 0。
	changes := snap.Rows[4]
	if len(changes.Children) != 1 {
		t.Fatalf("改动块子行 = %+v", changes.Children)
	}
	change := changes.Children[0]
	if change.Kind != ui.RowFileChange || change.Text != "internal/ui/source.go" {
		t.Fatalf("改动行 = %+v", change)
	}
	if change.DiffAdd != 0 || change.DiffRemove != 0 {
		t.Fatalf("改动行不该有差异计数（事实里没有）：%+v", change)
	}
}

// TestRowSnapshotFallback 认不出的行种类仍按正文画；没有正文就没有可画的。
func TestRowSnapshotFallback(t *testing.T) {
	row, ok := rowSnapshot(projection.Row{Kind: "some_future_kind", Text: "还有正文"})
	if !ok || row.Kind != ui.RowAssistant || row.Text != "还有正文" {
		t.Fatalf("认不出的行没有按正文回退：%+v（ok=%t）", row, ok)
	}
	if _, ok := rowSnapshot(projection.Row{Kind: projection.RowGoalChange}); ok {
		t.Fatal("没有正文的目标变更行不该占一行")
	}
}

// TestToolActionAndArgs 核对工具名到动作词的映射与参数摘要的取法。
func TestToolActionAndArgs(t *testing.T) {
	cases := []struct {
		name  string
		word  string
		input string
		args  string
	}{
		{"read_file", "读取", `{"path":"internal/ui/timeline.go"}`, "internal/ui/timeline.go"},
		{"bash", "执行", `{"command":"go test ./..."}`, "go test ./..."},
		{"mcp__fs__write_file", "写入", `{"file_path":"a.txt"}`, "a.txt"},
		{"grep", "搜索", `{"pattern":"liveTimeline"}`, "liveTimeline"},
		{"web_fetch", "抓取", `{"url":"https://example.com"}`, "https://example.com"},
		{"apply_patch", "改动", `{"patch":"*** Begin Patch"}`, ""},
		{"plot", "", `{"series":[1,2]}`, `{"series":[1,2]}`},
	}
	for _, tc := range cases {
		if got := toolAction(tc.name); got != tc.word {
			t.Fatalf("%s 的动作词 = %q，想要 %q", tc.name, got, tc.word)
		}
		args := toolArgs(json.RawMessage(tc.input))
		if tc.args == "" {
			if args == "" {
				t.Fatalf("%s 的参数摘要为空", tc.name)
			}
			continue
		}
		if args != tc.args {
			t.Fatalf("%s 的参数摘要 = %q，想要 %q", tc.name, args, tc.args)
		}
	}
}
