package uiwire

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// 本文件是三条实测 bug 的回归断言：排队输入的正文/附件条数来自 wire 事实、右栏
// 文件页在没有预设工作区根时按会话 cwd 读、模型槽能读到会话自己的模型。三条都用
// 假 App Server 的 wire 形状与真实目录，不读样例。

// bugScript 造一个目录里没有、但 attach 回执带 cwd 与 model 的会话：真实客户端
// 打开会话时这两项只能来自 attach（或目录），目录没给时不能再退回样例。
func bugScript(cwd string) fakeappserver.Script {
	return fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              e2eSession,
			LatestTurnStart: 1,
			Events:          e2eEvents(),
			Cwd:             cwd,
			ModelProviderID: "dimoauth",
			ModelID:         "deepseek-v4.1-flash",
		}},
		Replies: []fakeappserver.Reply{
			// 目录里没有这个会话：会话的 cwd 不能从目录条目补。
			{Method: "session.catalog", Result: map[string]any{
				"sessions": []any{}, "nextCursor": nil,
			}},
			{Method: "session.permission.get", Result: map[string]any{
				"version": 1, "policy": map[string]any{"preset": "full-access"},
			}},
			{Method: "session.input.pending", Result: map[string]any{
				"branchId": "main", "through": 10, "next": nil,
				"inputs": []any{
					map[string]any{
						"inputId": "q1", "sequence": 8, "mode": "queue", "applied": false,
						"user": map[string]any{"content": []any{
							map[string]any{"type": "text", "text": "再补一条错误分支的测试"},
						}},
					},
					map[string]any{
						"inputId": "q2", "sequence": 9, "mode": "queue", "applied": false,
						"user": map[string]any{"content": []any{
							map[string]any{"type": "input_text", "text": "先跑一遍编译"},
						}},
					},
					map[string]any{
						"inputId": "q3", "sequence": 10, "mode": "queue", "applied": false,
						"user": nil,
						"attachments": []any{
							map[string]any{"attachmentId": "a1"},
							map[string]any{"attachmentId": "a2"},
						},
					},
					// 队列项形状：没有 user，正文在 message / preview 里，附件在
					// attachmentNames 里（参考产品队列项的同一组事实）。
					map[string]any{
						"inputId": "q4", "sequence": 11, "mode": "queue", "applied": false,
						"user": nil, "preview": "预览里的一句话",
						"attachmentNames": []any{"截图.png", "日志.txt"},
					},
					map[string]any{
						"inputId": "q5", "sequence": 12, "mode": "queue", "applied": false,
						"user": nil, "message": "完整正文在这里", "preview": "摘要",
					},
				},
			}},
		},
	}
}

// TestQueuedInputsMapTextAndAttachmentCount 断言排队输入的映射：正文来自文本块
// （text 与 input_text 都认），只剩附件的输入带附件条数、正文为空——界面据此写
// 「附件 N 个」，不再把每条输入都写成「附件」。
func TestQueuedInputsMapTextAndAttachmentCount(t *testing.T) {
	server := startE2EServer(t, bugScript(t.TempDir()))
	w := New(newE2EApp(t, server))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	pin := w.Pin("")
	if err := pin.OpenSession(ctx, e2eSession); err != nil {
		t.Fatalf("打开会话：%v", err)
	}
	snap, ok := pin.Session()
	if !ok {
		t.Fatal("会话已打开却没有快照")
	}
	if len(snap.Queued) < 5 {
		t.Fatalf("排队输入 %d 条，至少要 5：%+v", len(snap.Queued), snap.Queued)
	}
	byID := map[string]ui.QueuedMessageSnapshot{}
	for _, queued := range snap.Queued {
		byID[queued.ID] = queued
	}
	for _, id := range []string{"q1", "q2", "q3", "q4", "q5"} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("清单里没有 %s：%+v", id, snap.Queued)
		}
	}
	if got := byID["q1"].Text; got != "再补一条错误分支的测试" {
		t.Fatalf("q1 正文 = %q，想要「再补一条错误分支的测试」", got)
	}
	if got := byID["q2"].Text; got != "先跑一遍编译" {
		t.Fatalf("input_text 块的正文 = %q，想要「先跑一遍编译」", got)
	}
	if got := byID["q3"]; got.Text != "" || got.Attachments != 2 {
		t.Fatalf("只有附件的输入 = %+v，想要正文空、附件 2 个", got)
	}
	if got := byID["q4"]; got.Text != "预览里的一句话" || got.Attachments != 2 {
		t.Fatalf("队列项形状的输入 = %+v，想要 preview 正文与 2 个附件", got)
	}
	if got := byID["q5"].Text; got != "完整正文在这里" {
		t.Fatalf("有完整正文时不该退回摘要：%q", got)
	}
}

// TestSnapshotUsesSessionCwdAndModel 断言快照里的 cwd 与模型来自已打开会话的
// 事实：目录里没有这个会话时，cwd 仍是 attach 回执给的目录；provider 记录里没有
// 选择时，模型槽仍读到会话自己的模型，而不是「未选择模型」。
func TestSnapshotUsesSessionCwdAndModel(t *testing.T) {
	cwd := t.TempDir()
	server := startE2EServer(t, bugScript(cwd))
	w := New(newE2EApp(t, server))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	pin := w.Pin("")
	if err := pin.OpenSession(ctx, e2eSession); err != nil {
		t.Fatalf("打开会话：%v", err)
	}
	snap, ok := pin.Session()
	if !ok {
		t.Fatal("会话已打开却没有快照")
	}
	if snap.Cwd != cwd {
		t.Fatalf("快照 cwd = %q，想要 attach 回执里的 %q", snap.Cwd, cwd)
	}
	settings := pin.Settings()
	if settings.ModelID != "deepseek-v4.1-flash" {
		t.Fatalf("模型标识 = %q，想要 deepseek-v4.1-flash", settings.ModelID)
	}
	if settings.Model == "" || settings.Model == "未选择模型" {
		t.Fatalf("模型槽 = %q，想要会话自己的模型", settings.Model)
	}
	if settings.ProviderID != "dimoauth" {
		t.Fatalf("供应商 = %q，想要 dimoauth", settings.ProviderID)
	}
}

// TestFileTreeReadsSessionCwdWithoutWorkspaceRoot 断言没有预设工作区根时右栏仍能
// 读文件：真实客户端不配置工作区根，会话 cwd 就是读目录的界。
func TestFileTreeReadsSessionCwdWithoutWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatalf("建目录：%v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "app.go"), []byte("package app\n"), 0o600); err != nil {
		t.Fatalf("写文件：%v", err)
	}
	server := startE2EServer(t, bugScript(root))
	// newE2EApp 不传 WorkspaceRoot：与真实客户端启动参数一致。
	w := New(newE2EApp(t, server))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	request := ui.FileTreeRequest{Cwd: root, Limit: 200}
	waitFor(t, "没有预设工作区根时目录读回", func() bool {
		return w.FileTree(request).Phase == ui.PanelReadReady
	})
	snapshot := w.FileTree(request)
	if snapshot.Error != "" {
		t.Fatalf("没有预设工作区根时目录读取报错：%s", snapshot.Error)
	}
	if _, ok := panelEntry(snapshot, "internal"); !ok {
		t.Fatalf("目录里没有 internal：%+v", snapshot.Entries)
	}
}
