package uiwire

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	appui "github.com/arcships/dimagent-mygo/internal/ui"
)

// intentHarness 是一套"界面 + 适配器 + 假服务端"：点界面上的元素，断言服务端
// 收到了对应请求。
type intentHarness struct {
	t      *testing.T
	server *fakeappserver.Server
	wire   *App
	app    *appui.App
	tester *gui.Tester
}

// newIntentHarness 启动假服务端、连接集成层，并用真实 Source/Intents 建界面。
func newIntentHarness(t *testing.T, script fakeappserver.Script) *intentHarness {
	t.Helper()
	server := startE2EServer(t, script)
	base := newE2EApp(t, server)
	wire := New(base)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := wire.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	view := appui.NewApp(appui.Options{
		State:   appui.StateDefault,
		Source:  wire,
		Intents: wire,
	})
	h := &intentHarness{t: t, server: server, wire: wire, app: view}
	h.tester = gui.NewTester(view.View, appui.WindowWidth, appui.WindowHeight)
	h.tester.Frame()
	return h
}

// frame 重绘一帧：后台意图交回的成功回调在渲染线程上应用。
func (h *intentHarness) frame() { h.tester.Frame() }

// waitFor 反复渲染直到元素出现；不依赖固定帧数，超时报出等了多少帧。
func (h *intentHarness) waitFor(label string, timeout time.Duration) {
	h.t.Helper()
	deadline := time.Now().Add(timeout)
	frames := 0
	for time.Now().Before(deadline) {
		h.tester.Frame()
		frames++
		if _, ok := h.tester.Find(label); ok {
			return
		}
		time.Sleep(time.Millisecond)
	}
	h.t.Fatalf("等待 %q 超时：%d 帧后仍未出现", label, frames)
}

// click 点击界面上显示指定文字或读屏名的元素。
func (h *intentHarness) click(label string) {
	h.t.Helper()
	if err := h.tester.Click(label); err != nil {
		h.t.Fatalf("点击 %q 失败：%v", label, err)
	}
	h.frame()
}

// typeInto 点中输入框并键入文本。
func (h *intentHarness) typeInto(label, text string) {
	h.t.Helper()
	h.click(label)
	h.tester.Type(text)
	h.frame()
}

// replaceInto 点中输入框，全选后键入文本（用于已带默认值的字段）。
func (h *intentHarness) replaceInto(label, text string) {
	h.t.Helper()
	h.click(label)
	h.tester.Key(gui.Cmd, gui.KeyA)
	h.tester.Type(text)
	h.frame()
}

// waitRequest 等到服务端收到指定方法的请求，并返回第一条请求的参数。
func (h *intentHarness) waitRequest(method string) map[string]any {
	h.t.Helper()
	waitFor(h.t, "服务端收到 "+method, func() bool {
		return len(h.server.RequestsFor(method)) > 0
	})
	requests := h.server.RequestsFor(method)
	return requestParamsMap(h.t, requests[0])
}

// waitCount 等到服务端收到至少 want 次 method；返回等待时长。
func (h *intentHarness) waitCount(method string, want int, timeout time.Duration, round int) time.Duration {
	h.t.Helper()
	start := time.Now()
	deadline := start.Add(timeout)
	for time.Now().Before(deadline) {
		h.tester.Frame()
		if len(h.server.RequestsFor(method)) >= want {
			return time.Since(start)
		}
		time.Sleep(time.Millisecond)
	}
	h.t.Fatalf("第 %d 轮等待 %s 到 %d 次超时：已等 %v（当前 %d 次）",
		round, method, want, time.Since(start), len(h.server.RequestsFor(method)))
	return 0
}

// requestParamsMap 把请求参数解析成 map。
func requestParamsMap(t *testing.T, request fakeappserver.Request) map[string]any {
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

// baseScript 是交互测试共用的服务端脚本：一个会话、目录与各写入方法的回执。
func baseScript(cwd string, running bool, features ...string) fakeappserver.Script {
	features = append(features,
		"mcp.import.sources", "mcp.import.preview", "mcp.import.apply",
		"models.providers.list", "provider.select",
		"permission.rules", "permission.set_rules",
		"session.branch_create", "session.branch_end",
		"attachment.put",
		"modality.task.list", "modality.task.cancel",
		"session.exec.stop", "session.exec.pending",
		"session.goal.set", "session.goal.edit", "session.goal.pause",
		"session.goal.resume", "session.goal.clear", "session.goal.get",
		"provider.list", "provider.management", "provider.probe",
		"provider.create", "provider.update", "provider.models.set_enabled",
	)
	return fakeappserver.Script{
		Features: features,
		Sessions: []fakeappserver.Session{{
			ID:              e2eSession,
			Running:         running,
			LatestTurnStart: 1,
			Events:          e2eEvents(),
		}},
		Replies: []fakeappserver.Reply{
			{Method: "session.catalog", Result: map[string]any{"sessions": []any{map[string]any{
				"sessionId": e2eSession, "cwd": cwd, "createdAt": 1, "updatedAt": 2,
				"running": running, "title": "演示会话",
			}}, "nextCursor": nil}},
			{Method: "session.permission.get", Result: map[string]any{
				"version": 1, "policy": map[string]any{"preset": "full-access"},
			}},
			{Method: "session.submit", Result: map[string]any{"inputId": "srv-input-1"}},
			{Method: "session.stop", Result: map[string]any{}},
			{Method: "approval.respond", Result: map[string]any{"ok": true}},
			{Method: "question.respond", Result: map[string]any{"ok": true}},
			{Method: "session.create", Result: map[string]any{"sessionId": "s2"}},
			{Method: "session.rename", Result: map[string]any{
				"title": "改名后的会话", "titleSource": "user", "revision": 2,
			}},
			{Method: "session.delete", Result: nil},
			{Method: "plugin.list", Result: []any{map[string]any{
				"name": "demo-plugin", "version": "v1.0.0", "off": false,
			}}},
			{Method: "plugin.enable", Result: map[string]any{"status": "running"}},
			{Method: "plugin.disable", Result: map[string]any{"status": "stopped"}},
			{Method: "plugin.remove", Result: map[string]any{}},
			{Method: "hooks.catalog", Result: map[string]any{
				"userRevision": "hooks-rev-1", "projectsRevision": "hooks-rev-1",
				"userDocument": map[string]any{},
				"sources":      []any{},
				"hooks": []any{map[string]any{
					"id": "hook-1", "name": "保存时格式化", "event": "PreToolUse",
					"command": "gofmt -w .", "source": "user", "enabled": true,
				}},
			}},
			{Method: "hooks.save", Result: map[string]any{"revision": "hooks-rev-2"}},
			{Method: "schedule.list", Result: map[string]any{"jobs": []any{map[string]any{
				"id": "job-1", "name": "每晚检查", "schedule": map[string]any{"kind": "cron", "expr": "0 2 * * *"},
				"prompt": "检查依赖", "cwd": "/work", "mode": "queue",
				"policy_cap": map[string]any{}, "enabled": true, "max_run_minutes": 0,
				"revision": 1, "created_by": map[string]any{},
			}}}},
			{Method: "schedule.set_enabled", Result: map[string]any{"job": map[string]any{
				"id": "job-1", "name": "每晚检查", "schedule": map[string]any{"kind": "cron", "expr": "0 2 * * *"},
				"prompt": "检查依赖", "cwd": "/work", "mode": "queue",
				"policy_cap": map[string]any{}, "enabled": false, "max_run_minutes": 0,
				"revision": 2, "created_by": map[string]any{},
			}}},
			{Method: "schedule.delete", Result: map[string]any{"job_id": "job-1"}},
			{Method: "mcp.set_enabled", Result: map[string]any{}},
			{Method: "mcp.import.sources", Result: map[string]any{"sources": []any{map[string]any{
				"id": "src-1", "client": "Cursor", "scope": "user", "path": "/tmp/cursor.json",
				"state": "ready", "entries": 1,
			}}}},
			{Method: "mcp.import.preview", Result: map[string]any{
				"revision": "rev-2", "sourceRevision": "src-rev-1",
				"store": map[string]any{"id": "user", "client": "Cursor", "scope": "user", "path": "/tmp/cursor.json"},
				"entries": []any{map[string]any{
					"id": "mcp-imported", "transport": "stdio", "status": "changed",
					"changes": []any{}, "unsupported": []any{}, "omitted": 0, "error": "",
				}},
				"unknownTopLevel": []any{},
			}},
			{Method: "mcp.import.apply", Result: map[string]any{"revision": "rev-3"}},
			{Method: "account.providers", Result: []any{map[string]any{
				"providerId": "acme-account", "displayName": "Acme",
				"loginMethods": []any{"browser", "device"},
			}}},
			{Method: "account.status", Result: map[string]any{
				"credentialEpoch": "e1",
				"account": map[string]any{
					"issuer": "acme", "subject": "acct-1", "email": "ada@real.test", "name": "Ada", "plan": "Pro",
				},
			}},
			{Method: "provider.catalog", Result: map[string]any{"catalog": map[string]any{
				"models": []any{
					map[string]any{"id": "model-x", "name": "Model-X"},
					map[string]any{"id": "model-b", "name": "Model-B"},
				},
			}}},
			{Method: "models.providers.list", Result: map[string]any{
				"providers": []any{
					map[string]any{"id": "acme", "name": "Acme", "enabled": true,
						"models":          []any{map[string]any{"id": "model-x", "name": "Model-X"}},
						"enabledModelIds": []any{"model-x"}},
					map[string]any{"id": "beta", "name": "Beta", "enabled": true,
						"models":          []any{map[string]any{"id": "model-b", "name": "Model-B"}},
						"enabledModelIds": []any{"model-b"}},
				},
				"selection": map[string]any{"providerId": "acme", "modelId": "model-x", "version": "sel-1"},
			}},
			{Method: "provider.select", Result: map[string]any{}},
			{Method: "provider.list", Result: map[string]any{"providers": []any{map[string]any{
				"providerId": "acme", "displayName": "Acme", "driverKind": "openai",
				"auth": map[string]any{"type": "bearer"}, "baseUrl": "https://acme.test",
				"enabled": true, "version": 1, "credentialConfigured": true,
			}}, "errors": []any{}}},
			{Method: "provider.management", Result: map[string]any{"providers": []any{map[string]any{
				"providerId": "acme", "displayName": "Acme", "version": 1, "enabled": true,
				"connection": map[string]any{
					"driverKind": "openai", "auth": map[string]any{"type": "bearer"},
					"baseUrl": "https://acme.test", "credentialConfigured": true,
				},
				"models": []any{map[string]any{
					"model":   map[string]any{"modelId": "model-x", "displayName": "Model-X", "status": "active", "capabilities": map[string]any{}},
					"enabled": true,
				}},
			}}}},
			{Method: "provider.probe", Result: map[string]any{"models": []any{
				map[string]any{"modelId": "model-x", "displayName": "Model-X"},
				map[string]any{"modelId": "model-y", "displayName": "Model-Y"},
			}}},
			{Method: "provider.create", Result: map[string]any{}},
			{Method: "provider.update", Result: map[string]any{}},
			{Method: "provider.models.set_enabled", Result: map[string]any{}},
			{Method: "account.logout", Result: map[string]any{"ok": true}},
			{Method: "account.login.begin", Result: map[string]any{
				"attemptId": "att-1", "verificationUri": "https://example.test/device",
				"verificationUriComplete": "https://example.test/device?code=ABCD-EFGH",
				"userCode":                "ABCD-EFGH", "expiresAt": 1700000600, "interval": 1,
			}},
			{Method: "account.login.status", Result: map[string]any{"status": "pending"}},
			{Method: "permission.rules", Result: map[string]any{
				"revision": "rules-rev-1",
				"commandRules": []any{
					map[string]any{"command": "go test ./...", "match": "exact", "decision": "allow"},
					map[string]any{"command": "rm -rf", "match": "prefix", "decision": "deny"},
				},
			}},
			{Method: "permission.set_rules", Result: map[string]any{
				"revision": "rules-rev-2", "commandRules": []any{},
			}},
			{Method: "session.branch_create", Result: map[string]any{"branch": map[string]any{
				"branchId": "side", "parentBranchId": "main", "branchPoint": 5,
			}}},
			{Method: "session.branch_end", Result: map[string]any{}},
			{Method: "attachment.put", Result: map[string]any{
				"attachmentId": "att-1", "path": "/work/attachment.txt", "bytes": 3,
			}},
			{Method: "modality.task.list", Result: map[string]any{"tasks": []any{map[string]any{
				"version": 1, "task_id": "task-1", "client_task_id": nil, "revision": 1,
				"category": "image", "operation": "generate", "executor": map[string]any{},
				"params": map[string]any{}, "state": "running", "progress": map[string]any{},
				"result": nil, "error": nil, "created_at": "2026-11-15T00:00:00Z",
				"updated_at": "2026-11-15T00:00:01Z", "finished_at": nil,
			}}, "next": nil}},
			{Method: "session.exec.pending", Result: map[string]any{
				"branchId": "main", "through": 6, "next": nil,
				"tasks": []any{map[string]any{
					"eventId": "task-e1", "sequence": 5, "startedAt": 1, "taskId": "task-1",
					"turnId": "t1", "requestId": "r1", "callId": "c1", "description": "构建",
				}},
			}},
			{Method: "session.exec.stop", Result: map[string]any{}},
			{Method: "session.goal.get", Result: map[string]any{"objective": "甲目标", "status": "active"}},
			{Method: "session.goal.set", Result: map[string]any{"sequence": 21}},
			{Method: "session.goal.edit", Result: map[string]any{"sequence": 22}},
			{Method: "session.goal.pause", Result: map[string]any{"sequence": 23}},
			{Method: "session.goal.resume", Result: map[string]any{"sequence": 24}},
			{Method: "session.goal.clear", Result: map[string]any{"sequence": 25}},
			{Method: "modality.task.cancel", Result: map[string]any{"task": map[string]any{
				"version": 1, "task_id": "task-1", "client_task_id": nil, "revision": 2,
				"category": "image", "operation": "generate", "executor": map[string]any{},
				"params": map[string]any{}, "state": "cancelled", "progress": map[string]any{},
				"result": nil, "error": nil, "created_at": "2026-11-15T00:00:00Z",
				"updated_at": "2026-11-15T00:00:02Z", "finished_at": "2026-11-15T00:00:02Z",
			}}},
			{Method: "mcp.catalog", Result: map[string]any{
				"userRevision": "rev-1",
				"servers": []any{map[string]any{
					"id": "mcp-one", "source": "user", "enabled": true, "transport": "stdio",
				}},
				"invalid": []any{},
			}},
			{Method: "mcp.remove", Result: map[string]any{}},
			{Method: "mcp.test", Result: map[string]any{"ok": true, "tools": []any{}}},
			{Method: "shell.run", Result: map[string]any{"runId": "run-srv-1"}},
			{Method: "shell.stop", Result: map[string]any{}},
		},
	}
}

// TestIntentSubmit 点发送槽 → session.submit。
func TestIntentSubmit(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.typeInto("消息", "你好")
	h.click("发送")
	params := h.waitRequest("session.submit")
	t.Logf("点了「发送」→ 服务端收到 session.submit（%d 次），text=%v",
		len(h.server.RequestsFor("session.submit")), params["text"])
	if params["text"] != "你好" || params["sessionId"] != e2eSession {
		t.Fatalf("提交参数不对：%v", params)
	}
}

// TestConcurrentUIInteractions 是 UI 并发交互测试，不是压测：每轮等上一次提交
// 落地再发下一次，避免把写登记的串行背压误判成偶发失败。
func TestConcurrentUIInteractions(t *testing.T) {
	script := baseScript("/work", false)
	script.Sessions = append(script.Sessions, fakeappserver.Session{ID: "s2"})
	script.Replies[0] = fakeappserver.Reply{Method: "session.catalog", Result: map[string]any{
		"sessions": []any{
			map[string]any{"sessionId": "s1", "cwd": "/work", "title": "演示会话"},
			map[string]any{"sessionId": "s2", "cwd": "/work", "title": "第二个会话"},
		},
		"nextCursor": nil,
	}}
	h := newIntentHarness(t, script)

	const rounds = 50
	var total, slowest time.Duration
	slowRounds := 0
	for i := 0; i < rounds; i++ {
		wantSubmit := len(h.server.RequestsFor("session.submit")) + 1
		wantAttachment := len(h.server.RequestsFor("attachment.put")) + 1
		start := time.Now()

		h.click("附件")
		h.waitFor("附件名", 2*time.Second)
		h.replaceInto("附件名", "note.txt")
		h.replaceInto("附件内容", "hello")
		h.click("添加附件")
		h.waitFor("note.txt", 2*time.Second)
		h.click("消息")
		h.tester.Type("hi")
		h.frame()
		h.waitFor("发送", 2*time.Second)
		h.click("发送")
		_ = h.tester.Click("s2")
		h.tester.Frame()
		_ = h.tester.Click("s1")
		h.tester.Frame()

		h.waitCount("attachment.put", wantAttachment, 10*time.Second, i+1)
		elapsed := h.waitCount("session.submit", wantSubmit, 10*time.Second, i+1)
		total += time.Since(start)
		if elapsed > slowest {
			slowest = elapsed
		}
		if elapsed > 100*time.Millisecond {
			slowRounds++
			t.Logf("第 %d 轮提交较慢：%v", i+1, elapsed)
		}
	}
	t.Logf("%d 轮：总耗时 %v，最慢单轮提交 %v，超过 100ms 的轮次 %d；session.submit=%d attachment.put=%d",
		rounds, total, slowest, slowRounds,
		len(h.server.RequestsFor("session.submit")),
		len(h.server.RequestsFor("attachment.put")))
}

// TestIntentSubmitContent 文本附件与图片附件一起提交，断言 user.content 块。
func TestIntentSubmitContent(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("附件")
	h.waitFor("附件类型：图片", 2*time.Second)
	h.typeInto("附件名", "note.txt")
	h.typeInto("附件内容", "hello")
	h.click("添加附件")
	h.waitRequest("attachment.put")
	h.waitFor("note.txt", 2*time.Second)
	h.click("附件")
	h.waitFor("附件类型：图片", 2*time.Second)
	h.click("附件类型：图片")
	h.waitFor("图片 base64", 2*time.Second)
	h.replaceInto("附件名", "pic.png")
	h.typeInto("媒体类型", "image/png")
	h.typeInto("图片 base64", "aGk=")
	h.click("添加附件")
	h.waitFor("pic.png · 图片", 2*time.Second)
	h.typeInto("消息", "hi")
	h.click("发送")
	params := h.waitRequest("session.submit")
	user, _ := params["user"].(map[string]any)
	content, _ := user["content"].([]any)
	types := make([]string, 0, len(content))
	for _, raw := range content {
		block, _ := raw.(map[string]any)
		types = append(types, block["type"].(string))
	}
	t.Logf("点了「发送」（带附件）→ session.submit（%d 次），content 块=%v", len(h.server.RequestsFor("session.submit")), types)
	if len(types) != 3 || types[0] != "text" || types[1] != "text" || types[2] != "image" {
		t.Fatalf("内容块不对：%v（params=%v）", types, params)
	}
	image, _ := content[2].(map[string]any)["source"].(map[string]any)
	if image["type"] != "base64" || image["mediaType"] != "image/png" || image["data"] != "aGk=" {
		t.Fatalf("图片块不对：%v", image)
	}
}

// TestIntentSubmitLargeText 超过 4KB 的文本先 attachment.put，再以引用块提交。
func TestIntentSubmitLargeText(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.typeInto("消息", strings.Repeat("长文本", 2000))
	h.click("发送")
	h.waitRequest("attachment.put")
	params := h.waitRequest("session.submit")
	user, _ := params["user"].(map[string]any)
	content, _ := user["content"].([]any)
	block, _ := content[0].(map[string]any)
	text, _ := block["text"].(string)
	t.Logf("点了「发送」（长文本）→ attachment.put + session.submit，首块=%q", text)
	if !strings.Contains(text, "[Pasted text:") {
		t.Fatalf("长文本没有走引用块：%v", params)
	}
}

// TestIntentStop 点停止槽 → session.stop。
func TestIntentStop(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", true))
	h.click("停止")
	params := h.waitRequest("session.stop")
	t.Logf("点了「停止」→ 服务端收到 session.stop（%d 次），sessionId=%v",
		len(h.server.RequestsFor("session.stop")), params["sessionId"])
	if params["sessionId"] != e2eSession {
		t.Fatalf("停止参数不对：%v", params)
	}
}

// TestIntentApproval 点「允许一次」→ approval.respond。
func TestIntentApproval(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	if err := h.server.Notify("approval.request", e2eApprovalNotice()); err != nil {
		t.Fatalf("推送审批：%v", err)
	}
	waitFor(t, "审批出现", func() bool {
		_, ok := h.wire.Approval()
		return ok
	})
	h.waitFor("允许一次", 2*time.Second)
	h.click("允许一次")
	params := h.waitRequest("approval.respond")
	t.Logf("点了「允许一次」→ 服务端收到 approval.respond（%d 次），decision=%v",
		len(h.server.RequestsFor("approval.respond")), params["decision"])
	if params["decision"] != "allow" {
		t.Fatalf("审批应答参数不对：%v", params)
	}
}

// TestIntentQuestion 点提问卡的「回答」→ question.respond。
func TestIntentQuestion(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	questionEvent := e2eRawEvent("q1", 10, "step", "tool-use",
		`{"turn_id":"t1","request_id":"r1","call":{"id":"c1","name":"ask_user","input":{"questions":[{"id":"q1","question":"选哪个？","options":["A","B"]}]}}}`)
	if err := h.server.PushEvent(e2eSession, questionEvent); err != nil {
		t.Fatalf("推送提问：%v", err)
	}
	waitFor(t, "提问出现", func() bool {
		_, ok := h.wire.Question()
		return ok
	})
	h.waitFor("提交回答", 2*time.Second)
	h.typeInto("回答", "A")
	h.click("提交回答")
	params := h.waitRequest("question.respond")
	t.Logf("点了「回答」→ 服务端收到 question.respond（%d 次）",
		len(h.server.RequestsFor("question.respond")))
	if params["sessionId"] != e2eSession {
		t.Fatalf("提问应答参数不对：%v", params)
	}
}

// TestIntentQuestionSkip 点提问卡的「跳过」→ question.respond（decision=cancel）。
func TestIntentQuestionSkip(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	questionEvent := e2eRawEvent("q1", 10, "step", "tool-use",
		`{"turn_id":"t1","request_id":"r1","call":{"id":"c1","name":"ask_user","input":{"questions":[{"id":"q1","question":"选哪个？","options":["A","B"]}]}}}`)
	if err := h.server.PushEvent(e2eSession, questionEvent); err != nil {
		t.Fatalf("推送提问：%v", err)
	}
	waitFor(t, "提问出现", func() bool {
		_, ok := h.wire.Question()
		return ok
	})
	h.waitFor("跳过", 2*time.Second)
	h.click("跳过")
	params := h.waitRequest("question.respond")
	t.Logf("点了「跳过」→ 服务端收到 question.respond（%d 次），decision=%v",
		len(h.server.RequestsFor("question.respond")), params["decision"])
	if params["decision"] != "cancel" {
		t.Fatalf("跳过参数不对：%v", params)
	}
}

// TestIntentOpenSession 点左栏另一条会话 → session.attach。
func TestIntentOpenSession(t *testing.T) {
	script := baseScript("/work", false)
	script.Sessions = append(script.Sessions, fakeappserver.Session{ID: "s2"})
	script.Replies[0] = fakeappserver.Reply{Method: "session.catalog", Result: map[string]any{
		"sessions": []any{
			map[string]any{"sessionId": "s1", "cwd": "/work", "title": "演示会话"},
			map[string]any{"sessionId": "s2", "cwd": "/work", "title": "第二个会话"},
		},
		"nextCursor": nil,
	}}
	h := newIntentHarness(t, script)
	h.click("s2")
	waitFor(t, "会话切到 s2", func() bool {
		return h.wire.currentSession() == "s2"
	})
	requests := h.server.RequestsFor("session.attach")
	t.Logf("点了「s2」→ 服务端收到 session.attach（%d 次），当前会话=%s",
		len(requests), h.wire.currentSession())
}

// TestIntentCreateSession 点左栏「新建会话」→ session.create。
func TestIntentCreateSession(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("新建会话")
	params := h.waitRequest("session.create")
	t.Logf("点了「新建会话」→ 服务端收到 session.create（%d 次），cwd=%v",
		len(h.server.RequestsFor("session.create")), params["cwd"])
	if params["cwd"] != "/work" {
		t.Fatalf("新建会话参数不对：%v", params)
	}
}

// TestIntentRename 会话菜单 → 重命名 → 保存 → session.rename。
func TestIntentRename(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("会话操作：s1")
	if err := h.tester.ChooseMenuItem("重命名"); err != nil {
		t.Fatalf("选择重命名：%v", err)
	}
	h.waitFor("保存名称", 2*time.Second)
	h.click("保存名称")
	params := h.waitRequest("session.rename")
	t.Logf("点了「重命名 · 保存」→ 服务端收到 session.rename（%d 次），title=%v",
		len(h.server.RequestsFor("session.rename")), params["title"])
	if params["title"] != "s1" {
		t.Fatalf("改名参数不对：%v", params)
	}
}

// TestIntentArchive 会话菜单 → 归档：本地隐藏，不出现在列表里。
func TestIntentArchive(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("会话操作：s1")
	if err := h.tester.ChooseMenuItem("归档"); err != nil {
		t.Fatalf("选择归档：%v", err)
	}
	waitFor(t, "会话从列表消失", func() bool {
		return len(h.wire.SessionNames()) == 0
	})
	t.Logf("点了「归档」→ 目录列表 %d 个会话（本地隐藏，不发服务端请求）",
		len(h.wire.SessionNames()))
}

// TestIntentExport 会话菜单 → 导出：本地写出一份 Markdown。
func TestIntentExport(t *testing.T) {
	dir := t.TempDir()
	h := newIntentHarness(t, baseScript(dir, false))
	h.click("会话操作：s1")
	if err := h.tester.ChooseMenuItem("导出"); err != nil {
		t.Fatalf("选择导出：%v", err)
	}
	var file string
	waitFor(t, "导出文件出现", func() bool {
		matches, _ := filepath.Glob(filepath.Join(dir, "dimagent-export-*.md"))
		if len(matches) == 0 {
			return false
		}
		file = matches[0]
		return true
	})
	t.Logf("点了「导出」→ 本地写出 %s", filepath.Base(file))
}

// TestIntentDelete 会话菜单 → 删除 → 确认 → session.delete。
func TestIntentDelete(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("会话操作：s1")
	if err := h.tester.ChooseMenuItem("删除"); err != nil {
		t.Fatalf("选择删除：%v", err)
	}
	h.waitFor("确认删除", 2*time.Second)
	h.click("确认删除")
	params := h.waitRequest("session.delete")
	t.Logf("点了「删除 · 确认」→ 服务端收到 session.delete（%d 次），sessionId=%v",
		len(h.server.RequestsFor("session.delete")), params["sessionId"])
	if params["sessionId"] != e2eSession {
		t.Fatalf("删除参数不对：%v", params)
	}
}

// signedOutScript 是一份账号未登录的脚本，用于登录入口测试。
func signedOutScript() fakeappserver.Script {
	script := baseScript("/work", false)
	for index := range script.Replies {
		if script.Replies[index].Method == "account.status" {
			script.Replies[index].Result = map[string]any{"credentialEpoch": "e1", "account": nil}
		}
	}
	// 重建后的账户屏只有 browser 登录；服务端按登录方式分方法名（account.login.browser.begin）。
	script.Replies = append(script.Replies, fakeappserver.Reply{Method: "account.login.browser.begin",
		Result: map[string]any{
			"attemptId": "att-browser-1", "authorizationUrl": "https://example.test/authorize",
			"expiresAt": 1700000600,
		}})
	return script
}

// TestIntentAccountSignIn 设置 · 账户 → 登录 → account.login.browser.begin。
//
// 旧的「设备码登录」入口随旧分节删除；重建后的账户屏只留参考产品的「登录」
// （browser 方式），这里按新入口走，断言仍是登录请求的 provider。
func TestIntentAccountSignIn(t *testing.T) {
	h := newIntentHarness(t, signedOutScript())
	h.click("设置")
	h.click("登录")
	params := h.waitRequest("account.login.browser.begin")
	t.Logf("点了「登录」→ 服务端收到 account.login.browser.begin（%d 次），providerId=%v",
		len(h.server.RequestsFor("account.login.browser.begin")), params["providerId"])
	if params["providerId"] != "acme-account" {
		t.Fatalf("登录参数不对：%v", params)
	}
}

// TestIntentLogout 设置 · 账户 → 退出登录 → account.logout。
func TestIntentLogout(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("退出登录")
	h.click("退出登录")
	h.waitRequest("account.logout")
	t.Logf("点了「退出登录」→ 服务端收到 account.logout（%d 次）",
		len(h.server.RequestsFor("account.logout")))
}

// TestIntentCreateBranch 标题行分支菜单 → 从当前分支切出 → session.branch_create。
func TestIntentCreateBranch(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("切换分支：main")
	if err := h.tester.ChooseMenuItem("从当前分支切出"); err != nil {
		t.Fatalf("选择从当前分支切出：%v", err)
	}
	params := h.waitRequest("session.branch_create")
	t.Logf("点了「从当前分支切出」→ 服务端收到 session.branch_create（%d 次），parentBranchId=%v at=%v prompt=%v",
		len(h.server.RequestsFor("session.branch_create")), params["parentBranchId"], params["at"], params["prompt"])
	if params["parentBranchId"] != "main" {
		t.Fatalf("分支创建参数不对：%v", params)
	}
}

// TestIntentSelectAndEndBranch 分支事实到达后切换分支并结束。
func TestIntentSelectAndEndBranch(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	branchEvent := e2eRawEvent("b1", 20, "branch", "create",
		`{"branch":"side","at":{"sequence":5},"prompt":"从第一轮开分支"}`)
	if err := h.server.PushEvent(e2eSession, branchEvent); err != nil {
		t.Fatalf("推送分支事实：%v", err)
	}
	waitFor(t, "分支出现", func() bool {
		snap, ok := h.wire.Session()
		if !ok {
			return false
		}
		for _, branch := range snap.Branches {
			if branch.ID == "side" {
				return true
			}
		}
		return false
	})
	h.waitFor("切换分支：main", 2*time.Second)
	h.click("切换分支：main")
	if err := h.tester.ChooseMenuItem("side"); err != nil {
		t.Fatalf("选择 side：%v", err)
	}
	waitFor(t, "当前分支切到 side", func() bool {
		snap, _ := h.wire.Session()
		return snap.Branch == "side"
	})
	t.Logf("点了「分支 · side」→ 当前分支=%s（投影切换，不发请求）", "side")
	h.waitFor("切换分支：side", 2*time.Second)
	h.click("切换分支：side")
	if err := h.tester.ChooseMenuItem("结束分支"); err != nil {
		t.Fatalf("选择结束分支：%v", err)
	}
	h.waitFor("确认结束分支", 2*time.Second)
	h.click("确认结束分支")
	h.waitRequest("session.branch_end")
	t.Logf("点了「结束分支 · 确认」→ 服务端收到 session.branch_end（%d 次）",
		len(h.server.RequestsFor("session.branch_end")))
}

// TestIntentAttachment 输入卡附件编辑 → attachment.put。
func TestIntentAttachment(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("附件")
	h.typeInto("附件名", "note.txt")
	h.typeInto("附件内容", "hello")
	h.click("添加附件")
	params := h.waitRequest("attachment.put")
	t.Logf("点了「添加附件」→ 服务端收到 attachment.put（%d 次），name=%v text=%v",
		len(h.server.RequestsFor("attachment.put")), params["name"], params["text"])
	if params["text"] != "hello" {
		t.Fatalf("附件参数不对：%v", params)
	}
}

// goalEvent 造一条承载目标记录的输入事实。
func goalEvent(id string, sequence int64, status, op, objective string) protocol.RawEvent {
	body := "<dim-goal status=\"" + status + "\" judge=\"model\" op=\"" + op + "\">\n<objective>\n" + objective + "\n</objective>\n</dim-goal>"
	data, _ := json.Marshal(map[string]any{"blocks": []any{map[string]any{"key": "goal:state", "text": body}}})
	return protocol.RawEvent{ID: id, Sequence: sequence, CreateAt: 1700000000000 + sequence,
		Type: "input", Name: "message", Data: data}
}

// TestIntentSetGoal 目标行 → 设定 → 保存 → session.goal.set；打开编辑时会先 get。
func TestIntentSetGoal(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("修改目标")
	h.waitRequest("session.goal.get")
	h.typeInto("目标", "把发送路径改成排队")
	h.click("保存目标")
	params := h.waitRequest("session.goal.set")
	t.Logf("点了「保存目标」→ 服务端收到 session.goal.set（%d 次），objective=%v",
		len(h.server.RequestsFor("session.goal.set")), params["objective"])
	if params["objective"] != "把发送路径改成排队" {
		t.Fatalf("目标写入参数不对：%v", params)
	}
}

// TestIntentEditGoal 已有目标 → 修改 → 保存 → session.goal.edit。
func TestIntentEditGoal(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	if err := h.server.PushEvent(e2eSession, goalEvent("g1", 30, "active", "set", "甲目标")); err != nil {
		t.Fatalf("推送目标：%v", err)
	}
	waitFor(t, "目标出现", func() bool {
		snap, ok := h.wire.Session()
		return ok && snap.Goal == "甲目标"
	})
	h.waitFor("修改目标", 2*time.Second)
	h.click("修改目标")
	h.waitRequest("session.goal.get")
	h.typeInto("目标", "改")
	h.click("保存目标")
	params := h.waitRequest("session.goal.edit")
	t.Logf("点了「修改目标 · 保存」→ 服务端收到 session.goal.edit（%d 次），objective=%v",
		len(h.server.RequestsFor("session.goal.edit")), params["objective"])
}

// TestIntentGoalPauseResumeClear 目标暂停 / 恢复 / 两步清除。
func TestIntentGoalPauseResumeClear(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	if err := h.server.PushEvent(e2eSession, goalEvent("g1", 30, "active", "set", "甲目标")); err != nil {
		t.Fatalf("推送目标：%v", err)
	}
	waitFor(t, "目标出现", func() bool {
		snap, ok := h.wire.Session()
		return ok && snap.Goal == "甲目标"
	})
	h.waitFor("暂停目标", 2*time.Second)
	h.click("暂停目标")
	params := h.waitRequest("session.goal.pause")
	t.Logf("点了「暂停目标」→ 服务端收到 session.goal.pause（%d 次）", len(h.server.RequestsFor("session.goal.pause")))

	if err := h.server.PushEvent(e2eSession, goalEvent("g2", 31, "paused", "pause", "甲目标")); err != nil {
		t.Fatalf("推送暂停目标：%v", err)
	}
	waitFor(t, "目标暂停", func() bool {
		snap, ok := h.wire.Session()
		return ok && snap.GoalStatus == "paused"
	})
	h.waitFor("恢复目标", 2*time.Second)
	h.click("恢复目标")
	h.waitRequest("session.goal.resume")
	t.Logf("点了「恢复目标」→ 服务端收到 session.goal.resume（%d 次）", len(h.server.RequestsFor("session.goal.resume")))

	h.waitFor("清除目标", 2*time.Second)
	h.click("清除目标")
	h.waitFor("确认清除目标", 2*time.Second)
	h.click("确认清除目标")
	clear := h.waitRequest("session.goal.clear")
	t.Logf("点了「清除目标 · 确认」→ 服务端收到 session.goal.clear（%d 次），objective=%v",
		len(h.server.RequestsFor("session.goal.clear")), clear["objective"])
	_ = params
}

// TestIntentStopExecTask 右栏后台任务 → 停止 → session.exec.stop。
func TestIntentStopExecTask(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("打开右栏")
	h.click("后台")
	h.click("停止任务")
	params := h.waitRequest("session.exec.stop")
	t.Logf("点了「停止任务」→ 服务端收到 session.exec.stop（%d 次），taskId=%v",
		len(h.server.RequestsFor("session.exec.stop")), params["taskId"])
	if params["taskId"] != "task-1" {
		t.Fatalf("停止任务参数不对：%v", params)
	}
}

// TestIntentReconnect 断线后点「重新连接」恢复连接阶段。
func TestIntentReconnect(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.server.DisconnectAll()
	waitFor(t, "断线阶段", func() bool {
		return h.wire.Connection().Phase == "disconnected"
	})
	h.waitFor("重新连接", 2*time.Second)
	h.click("重新连接")
	waitFor(t, "重新连上", func() bool {
		return h.wire.Connection().Phase == "connected"
	})
	t.Logf("点了「重新连接」→ 连接阶段=%s", h.wire.Connection().Phase)
}

// TestIntentShellRun 输入 `!命令` → 运行 → shell.run。
func TestIntentShellRun(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false, "shell.run", "shell.stop"))
	h.typeInto("消息", "!echo hi")
	h.click("运行命令")
	params := h.waitRequest("shell.run")
	t.Logf("点了「运行命令」→ 服务端收到 shell.run（%d 次），command=%v",
		len(h.server.RequestsFor("shell.run")), params["command"])
	if params["command"] != "echo hi" {
		t.Fatalf("命令执行参数不对：%v", params)
	}
}

// TestIntentShellStop 运行后打开右栏终端 → 中止 → shell.stop。
func TestIntentShellStop(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false, "shell.run", "shell.stop"))
	h.typeInto("消息", "!echo hi")
	h.click("运行命令")
	h.waitRequest("shell.run")
	h.waitFor("消息", 2*time.Second)
	h.click("打开右栏")
	h.click("终端")
	h.waitFor("中止运行", 2*time.Second)
	h.click("中止运行")
	params := h.waitRequest("shell.stop")
	t.Logf("点了「终端 · 中止」→ 服务端收到 shell.stop（%d 次），runId=%v",
		len(h.server.RequestsFor("shell.stop")), params["runId"])
	if params["runId"] != "run-srv-1" {
		t.Fatalf("停止命令参数不对：%v", params)
	}
}
