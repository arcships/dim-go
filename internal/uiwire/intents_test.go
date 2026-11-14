package uiwire

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"

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

// waitRequest 等到服务端收到指定方法的请求，并返回第一条请求的参数。
func (h *intentHarness) waitRequest(method string) map[string]any {
	h.t.Helper()
	waitFor(h.t, "服务端收到 "+method, func() bool {
		return len(h.server.RequestsFor(method)) > 0
	})
	requests := h.server.RequestsFor(method)
	return requestParamsMap(h.t, requests[0])
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
				"providerId": "acme-account", "displayName": "Acme", "loginMethods": []any{"browser"},
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
	h.frame()
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
	h.frame()
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
	h.frame()
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
		return h.wire.SessionID == "s2"
	})
	requests := h.server.RequestsFor("session.attach")
	t.Logf("点了「s2」→ 服务端收到 session.attach（%d 次），当前会话=%s",
		len(requests), h.wire.SessionID)
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
	h.frame()
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
	h.frame()
	h.click("确认删除")
	params := h.waitRequest("session.delete")
	t.Logf("点了「删除 · 确认」→ 服务端收到 session.delete（%d 次），sessionId=%v",
		len(h.server.RequestsFor("session.delete")), params["sessionId"])
	if params["sessionId"] != e2eSession {
		t.Fatalf("删除参数不对：%v", params)
	}
}

// TestIntentSelectModel 设置 · 模型与供应商 → 选 provider → provider.select。
func TestIntentSelectModel(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("模型与供应商")
	h.click("切换到 Beta")
	params := h.waitRequest("provider.select")
	t.Logf("点了「供应商 · Beta」→ 服务端收到 provider.select（%d 次），providerId=%v modelId=%v",
		len(h.server.RequestsFor("provider.select")), params["providerId"], params["modelId"])
	if params["providerId"] != "beta" || params["modelId"] != "model-b" {
		t.Fatalf("模型选择参数不对：%v", params)
	}
}

// TestIntentChangeRule 设置 · 命令规则 → 改一条决定 → permission.set_rules。
func TestIntentChangeRule(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("命令规则")
	h.click("拒绝：go test ./...")
	params := h.waitRequest("permission.set_rules")
	t.Logf("点了「允许→拒绝」→ 服务端收到 permission.set_rules（%d 次），rules=%v",
		len(h.server.RequestsFor("permission.set_rules")), params["commandRules"])
	rules, _ := params["commandRules"].([]any)
	if len(rules) == 0 {
		t.Fatalf("规则写入为空：%v", params)
	}
	first, _ := rules[0].(map[string]any)
	if first["decision"] != "deny" {
		t.Fatalf("规则决定不对：%v", first)
	}
}

// TestIntentClearRules 设置 · 命令规则 → 清空（两步）→ permission.set_rules。
func TestIntentClearRules(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("命令规则")
	h.click("清空命令规则")
	h.click("确认清空")
	params := h.waitRequest("permission.set_rules")
	t.Logf("点了「清空命令规则 · 确认」→ 服务端收到 permission.set_rules（%d 次），expectedRevision=%v rules=%v",
		len(h.server.RequestsFor("permission.set_rules")), params["expectedRevision"], params["commandRules"])
	if params["expectedRevision"] != "rules-rev-1" {
		t.Fatalf("规则写入版本不对：%v", params)
	}
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
	h.frame()
	h.click("切换分支：main")
	if err := h.tester.ChooseMenuItem("side"); err != nil {
		t.Fatalf("选择 side：%v", err)
	}
	waitFor(t, "当前分支切到 side", func() bool {
		snap, _ := h.wire.Session()
		return snap.Branch == "side"
	})
	t.Logf("点了「分支 · side」→ 当前分支=%s（投影切换，不发请求）", "side")
	h.frame()
	h.click("切换分支：side")
	if err := h.tester.ChooseMenuItem("结束分支"); err != nil {
		t.Fatalf("选择结束分支：%v", err)
	}
	h.frame()
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

// TestIntentReconnect 断线后点「重新连接」恢复连接阶段。
func TestIntentReconnect(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.server.DisconnectAll()
	waitFor(t, "断线阶段", func() bool {
		return h.wire.Connection().Phase == "disconnected"
	})
	h.frame()
	h.click("重新连接")
	waitFor(t, "重新连上", func() bool {
		return h.wire.Connection().Phase == "connected"
	})
	t.Logf("点了「重新连接」→ 连接阶段=%s", h.wire.Connection().Phase)
}

// TestIntentExtensionToggle 设置 · 扩展 → 插件开关 → plugin.disable。
func TestIntentExtensionToggle(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("扩展")
	h.click("插件")
	h.click("启用：demo-plugin")
	params := h.waitRequest("plugin.disable")
	t.Logf("点了「插件 · 启用开关」→ 服务端收到 plugin.disable（%d 次），name=%v",
		len(h.server.RequestsFor("plugin.disable")), params["name"])
	if params["name"] != "demo-plugin" {
		t.Fatalf("插件启停参数不对：%v", params)
	}
}

// TestIntentMCPTest 设置 · 扩展 · MCP → 测试 → mcp.test。
func TestIntentMCPTest(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("扩展")
	h.click("MCP")
	h.click("连接测试：mcp-one")
	params := h.waitRequest("mcp.test")
	t.Logf("点了「MCP · 测试」→ 服务端收到 mcp.test（%d 次），id=%v",
		len(h.server.RequestsFor("mcp.test")), params["id"])
	if params["id"] != "mcp-one" {
		t.Fatalf("MCP 测试参数不对：%v", params)
	}
}

// TestIntentMCPUninstall 设置 · 扩展 · MCP → 详情 → 卸载 → mcp.remove。
func TestIntentMCPUninstall(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("扩展")
	h.click("MCP")
	h.click("详情：mcp-one")
	h.frame()
	h.click("卸载扩展")
	params := h.waitRequest("mcp.remove")
	t.Logf("点了「MCP · 卸载」→ 服务端收到 mcp.remove（%d 次），id=%v",
		len(h.server.RequestsFor("mcp.remove")), params["id"])
	if params["id"] != "mcp-one" {
		t.Fatalf("MCP 删除参数不对：%v", params)
	}
}

// TestIntentScheduleToggle 设置 · 定时任务 → 开关 → schedule.set_enabled。
func TestIntentScheduleToggle(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("定时任务")
	h.click("启用任务：每晚检查")
	params := h.waitRequest("schedule.set_enabled")
	t.Logf("点了「定时任务 · 开关」→ 服务端收到 schedule.set_enabled（%d 次），job_id=%v enabled=%v",
		len(h.server.RequestsFor("schedule.set_enabled")), params["job_id"], params["enabled"])
	if params["job_id"] != "job-1" || params["enabled"] != false {
		t.Fatalf("定时任务启停参数不对：%v", params)
	}
}

// TestIntentScheduleDelete 设置 · 定时任务 → 删除 → 确认 → schedule.delete。
func TestIntentScheduleDelete(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("定时任务")
	h.click("删除任务：每晚检查")
	h.frame()
	h.click("确认删除任务")
	params := h.waitRequest("schedule.delete")
	t.Logf("点了「定时任务 · 删除」→ 服务端收到 schedule.delete（%d 次），job_id=%v",
		len(h.server.RequestsFor("schedule.delete")), params["job_id"])
	if params["job_id"] != "job-1" {
		t.Fatalf("定时任务删除参数不对：%v", params)
	}
}

// TestIntentPluginUninstall 设置 · 扩展 · 插件 → 详情 → 卸载 → plugin.remove。
func TestIntentPluginUninstall(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("扩展")
	h.click("插件")
	h.click("详情：demo-plugin")
	h.frame()
	h.click("卸载扩展")
	params := h.waitRequest("plugin.remove")
	t.Logf("点了「插件 · 卸载」→ 服务端收到 plugin.remove（%d 次），id=%v",
		len(h.server.RequestsFor("plugin.remove")), params["id"])
	if params["id"] != "demo-plugin" {
		t.Fatalf("插件卸载参数不对：%v", params)
	}
}

// TestIntentMCPToggle 设置 · 扩展 · MCP → 开关 → mcp.set_enabled。
func TestIntentMCPToggle(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("扩展")
	h.click("MCP")
	h.click("启用：mcp-one")
	params := h.waitRequest("mcp.set_enabled")
	t.Logf("点了「MCP · 启用开关」→ 服务端收到 mcp.set_enabled（%d 次），id=%v enabled=%v",
		len(h.server.RequestsFor("mcp.set_enabled")), params["id"], params["enabled"])
	if params["id"] != "mcp-one" || params["enabled"] != false {
		t.Fatalf("MCP 启停参数不对：%v", params)
	}
}

// TestIntentMCPImport 设置 · 扩展 · MCP → 导入来源：先预览再应用。
func TestIntentMCPImport(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("扩展")
	h.click("MCP")
	h.click("从 Cursor 导入")
	preview := h.waitRequest("mcp.import.preview")
	apply := h.waitRequest("mcp.import.apply")
	t.Logf("点了「从 Cursor 导入」→ mcp.import.preview（%d 次），from=%v",
		len(h.server.RequestsFor("mcp.import.preview")), preview["from"])
	t.Logf("  随后 mcp.import.apply（%d 次），from=%v expectedVersion=%v sourceRevision=%v",
		len(h.server.RequestsFor("mcp.import.apply")), apply["from"],
		apply["expectedVersion"], apply["expectedSourceRevision"])
	if preview["from"] != "src-1" || apply["from"] != "src-1" ||
		apply["expectedVersion"] != "rev-2" || apply["expectedSourceRevision"] != "src-rev-1" {
		t.Fatalf("MCP 导入参数不对：preview=%v apply=%v", preview, apply)
	}
}

// TestIntentHookToggle 设置 · 扩展 · 钩子 → 开关 → hooks.save。
func TestIntentHookToggle(t *testing.T) {
	h := newIntentHarness(t, baseScript("/work", false))
	h.click("设置")
	h.click("扩展")
	h.click("钩子")
	h.click("启用：保存时格式化")
	h.waitRequest("hooks.save")
	t.Logf("点了「钩子 · 开关」→ 服务端收到 hooks.save（%d 次）",
		len(h.server.RequestsFor("hooks.save")))
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
	h.frame() // 应用 RunShell 的成功回调，记下运行标识
	h.click("打开右栏")
	h.click("终端")
	h.click("中止运行")
	params := h.waitRequest("shell.stop")
	t.Logf("点了「终端 · 中止」→ 服务端收到 shell.stop（%d 次），runId=%v",
		len(h.server.RequestsFor("shell.stop")), params["runId"])
	if params["runId"] != "run-srv-1" {
		t.Fatalf("停止命令参数不对：%v", params)
	}
}
