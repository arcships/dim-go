// Command dimagent-fake-server 是界面证据用的假 App Server：不属于产品入口，
// 只用来给 cmd/dimagent-ui 的 -connect 模式提供与真实服务端同一条协议路径。
//
// 用法（两个终端）：
//
//	go run ./cmd/dimagent-fake-server -dir /tmp/dimagent-ui-fixture -scenario streaming
//	go run ./cmd/dimagent-ui -connect -data-dir /tmp/dimagent-ui-fixture
//
// 数据目录里的 socket 路径与真实客户端一致（<dir>/rust/daemon.sock）；
// 目录已存在端点时 daemon.Config 不会另起进程。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
)

const sessionID = "s1"

func main() {
	dir := flag.String("dir", "/tmp/dimagent-ui-fixture", "假服务端数据目录（socket 落在 <dir>/rust/daemon.sock）")
	scenario := flag.String("scenario", "streaming", "场景：streaming（推送流式文本）或 approval（推送审批请求）")
	delay := flag.Duration("delay", 0, "会话 attach 之后再等多久推送事实（0 = 立即）")
	flag.Parse()

	if err := os.MkdirAll(*dir, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "建数据目录：", err)
		os.Exit(1)
	}
	server, err := fakeappserver.StartIn(*dir, script(*scenario))
	if err != nil {
		fmt.Fprintln(os.Stderr, "启动假服务端：", err)
		os.Exit(1)
	}
	defer func() { _ = server.Close() }()
	fmt.Printf("data=%s\nendpoint=%s\n", server.DataDir(), server.Endpoint().Address)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go push(ctx, server, *scenario, *delay)

	<-ctx.Done()
	fmt.Println("bye")
}

// script 是两种场景共用的目录与能力脚本。
func script(scenario string) fakeappserver.Script {
	running := scenario == "streaming"
	return fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              sessionID,
			Running:         running,
			LatestTurnStart: 1,
			Events:          history("我先看一遍发送路径。排队不再覆盖队列，运行中的提交先入队，这一轮结束后自动发出。", true),
		}},
		Replies: []fakeappserver.Reply{
			{Method: "session.catalog", Result: map[string]any{"sessions": []any{map[string]any{
				"sessionId": sessionID, "cwd": "/work", "createdAt": 1, "updatedAt": 2,
				"running": running, "title": "发送路径排队",
			}}, "nextCursor": nil}},
			{Method: "session.permission.get", Result: map[string]any{
				"version": 1, "policy": map[string]any{"preset": "workspace-write"},
			}},
			{Method: "session.submit", Result: map[string]any{"inputId": "srv-input-1"}},
			{Method: "session.stop", Result: map[string]any{}},
			{Method: "approval.respond", Result: map[string]any{"ok": true}},
		},
	}
}

// history 是首屏历史：一条用户消息、一轮开始与一段助手文本。
func history(text string, open bool) []protocol.RawEvent {
	events := []protocol.RawEvent{
		raw("e1", 1, "input", "message", `{"mode":"queue","user":{"content":[{"type":"text","text":"把发送路径改成排队"}]}}`),
		raw("e2", 2, "turn", "start", `{"turn_id":"t1"}`),
		raw("e3", 3, "step", "text", `{"turn_id":"t1","request_id":"r1","delta":"`+text+`"}`),
	}
	if !open {
		events = append(events, raw("e4", 4, "turn", "end", `{"turn_id":"t1","status":"completed"}`))
	}
	return events
}

// push 在会话 attach 之后推送场景事实：截图取的是窗口第一帧，推送要赶在它之前。
func push(ctx context.Context, server *fakeappserver.Server, scenario string, delay time.Duration) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if len(server.AttachedSessions()) > 0 {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Millisecond):
		}
	}
	if delay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
	fmt.Printf("attached=%v\n", server.AttachedSessions())
	switch scenario {
	case "approval":
		if err := server.Notify("approval.request", map[string]any{
			"sessionId": sessionID,
			"branchId":  "main",
			"requestId": "ap-1",
			"source": map[string]any{
				"kind": "tool", "turn_id": "t1", "request_id": "r1", "call_id": "c1",
			},
			"call":   map[string]any{"name": "exec", "input": map[string]any{"command": "go test ./..."}},
			"reason": "运行测试",
		}); err != nil {
			fmt.Fprintln(os.Stderr, "推送审批请求：", err)
		}
	default:
		// streaming 场景把流式文本放在首屏历史里：窗口不可见时框架只提交首帧，
		// 推送的增量不会出现在截图里。实时增量由 internal/uiwire 的端到端测试覆盖。
	}
}

func raw(id string, sequence int64, typ, name, data string) protocol.RawEvent {
	return protocol.RawEvent{
		ID:       id,
		Sequence: sequence,
		CreateAt: 1700000000000 + sequence,
		Type:     typ,
		Name:     name,
		Data:     json.RawMessage(data),
	}
}
