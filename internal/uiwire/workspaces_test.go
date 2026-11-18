package uiwire

import (
	"context"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
)

// TestSourceWorkspacesFromSettings 证明已授权项目根进了 Source：连接时读一次
// settings.projects，界面按服务端给的目录建工作区清单（不改形状、不拼路径）。
func TestSourceWorkspacesFromSettings(t *testing.T) {
	script := e2eScript()
	script.Features = append(script.Features, "settings.projects")
	script.Replies = append(script.Replies, fakeappserver.Reply{
		Method: "settings.projects",
		Result: map[string]any{"projects": []any{
			map[string]any{"root": 7, "cwd": "/work/demo-one"},
			map[string]any{"root": 9, "cwd": "/work/demo-two"},
		}},
	})
	server := startE2EServer(t, script)
	base := newE2EApp(t, server)
	w := New(base)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	got := w.Workspaces()
	if len(got) != 2 {
		t.Fatalf("工作区 %d 个，想要 2 个：%+v", len(got), got)
	}
	if got[0].Root != 7 || got[0].Path != "/work/demo-one" {
		t.Fatalf("首个工作区 = %+v，想要 root 7 / /work/demo-one", got[0])
	}
	if got[1].Root != 9 || got[1].Path != "/work/demo-two" {
		t.Fatalf("第二个工作区 = %+v，想要 root 9 / /work/demo-two", got[1])
	}
}
