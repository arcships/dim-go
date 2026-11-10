package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/extensions"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/settings"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestManagementReadsUseRealMethods 核对管理面读取用 alpha.21 的真实方法名，
// 并且不再调用旧猜测的名字。
func TestManagementReadsUseRealMethods(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "plugin.list", Result: []any{
				map[string]any{"name": "demo", "use": true, "off": false},
			}},
			{Method: "skills.catalog", Result: map[string]any{
				"version": 1,
				"entries": []any{map[string]any{
					"id": "skill-1", "source": "user", "name": "提交信息",
					"filePath": "/skills/skill-1", "enabled": true,
				}},
				"issues": []any{},
			}},
			{Method: "hooks.catalog", Result: map[string]any{
				"userRevision": "r1", "userDocument": map[string]any{},
				"projectsRevision": "r1", "disableAllHooks": false,
				"sources": []any{}, "hooks": []any{map[string]any{
					"id": "h1", "name": "格式化", "event": "pre", "command": "gofmt", "source": "user", "enabled": true,
				}},
			}},
			{Method: "mcp.catalog", Result: map[string]any{
				"userRevision": "r1", "userApplication": map[string]any{},
				"pluginApplication": map[string]any{}, "servers": []any{map[string]any{
					"id": "srv-1", "source": "user", "enabled": true, "transport": "stdio",
				}}, "invalid": []any{},
			}},
			{Method: "schedule.list", Result: map[string]any{"jobs": []any{map[string]any{
				"id": "job-1", "name": "夜跑", "schedule": map[string]any{"every_minutes": 60},
				"prompt": "p", "cwd": "/w", "mode": "agent", "policy_cap": map[string]any{}, "enabled": true,
			}}}},
			{Method: "session.wakeup.list", Result: map[string]any{"wakeups": []any{map[string]any{
				"wakeupId": "w-1", "at": "2026-11-11T10:00:00Z", "prompt": "提醒", "state": "scheduled",
			}}}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	service := result.base.Extensions()

	plugins, err := service.ListPlugins(ctx)
	if err != nil || len(plugins.Items) != 1 || plugins.Items[0].ID != "demo" || !plugins.Items[0].Enabled {
		t.Fatalf("插件清单映射不对: %v %+v", err, plugins.Items)
	}
	skills, err := service.ListSkills(ctx)
	if err != nil || len(skills.Items) != 1 || skills.Items[0].Name != "提交信息" {
		t.Fatalf("技能清单映射不对: %v %+v", err, skills.Items)
	}
	hooks, err := service.ListHooks(ctx)
	if err != nil || len(hooks.Items) != 1 || hooks.Items[0].Command != "gofmt" {
		t.Fatalf("钩子清单映射不对: %v %+v", err, hooks.Items)
	}
	servers, err := service.ListToolServers(ctx)
	if err != nil || len(servers.Items) != 1 || servers.Items[0].ID != "srv-1" {
		t.Fatalf("MCP 清单映射不对: %v %+v", err, servers.Items)
	}
	tasks, err := service.ListScheduledTasks(ctx)
	if err != nil || len(tasks.Items) != 1 || tasks.Items[0].Name != "夜跑" ||
		!strings.Contains(tasks.Items[0].Schedule, "every_minutes") {
		t.Fatalf("计划任务映射不对: %v %+v", err, tasks.Items)
	}
	wakeups, err := service.ListWakeups(ctx, extensions.WakeupScope{SessionID: testSession})
	if err != nil || len(wakeups.Items) != 1 || wakeups.Items[0].ID != "w-1" ||
		wakeups.Items[0].Status != extensions.WakeupScheduled {
		t.Fatalf("唤醒清单映射不对: %v %+v", err, wakeups.Items)
	}

	// 旧猜测的方法名一个都不该被调用。
	for _, method := range []string{
		"toolServer.list", "scheduler.list", "wakeup.list", "skill.list", "hook.list",
	} {
		if got := len(server.RequestsFor(method)); got != 0 {
			t.Fatalf("不应再调用 %s（%d 次）", method, got)
		}
	}
}

// TestWorkflowAndRuntimeMissingWithoutRequests 核对未注册的方法面按能力缺失处理，
// 且不发请求。
func TestWorkflowAndRuntimeMissingWithoutRequests(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()

	if _, err := result.base.Extensions().ListWorkflows(ctx); !errors.Is(err, extensions.ErrCapabilityMissing) {
		t.Fatalf("workflow 读取应报能力缺失: %v", err)
	}
	if _, err := result.base.Settings().RefreshRuntimes(ctx); !errors.Is(err, settings.ErrCapabilityMissing) {
		t.Fatalf("运行组件读取应报能力缺失: %v", err)
	}
	for _, method := range []string{"workflow.list", "runtime.list"} {
		if got := len(server.RequestsFor(method)); got != 0 {
			t.Fatalf("缺能力时不应请求 %s（%d 次）", method, got)
		}
	}

	// 写侧同样零请求。
	registered, err := result.base.Writes().Register(write.Intent{
		Key: "runtime-install", Kind: write.KindManagementWrite,
		Params: json.RawMessage(`{"op":"runtime.install","name":"uv"}`),
	})
	if err != nil {
		t.Fatalf("登记组件安装: %v", err)
	}
	settled, err := result.base.Writes().Wait(ctx, registered.ID)
	if err != nil {
		t.Fatalf("等待收敛: %v", err)
	}
	if settled.Status != write.StatusFailed || !strings.Contains(settled.Reason, "能力") {
		t.Fatalf("组件安装应明确失败并说明能力缺失: %+v", settled)
	}
	if got := len(server.RequestsFor("runtime.install")); got != 0 {
		t.Fatalf("缺能力时不应请求 runtime.install（%d 次）", got)
	}
}

// TestRenameGoesThroughSessionStream 核对改名走会话流，并用真实回执的修订号合并。
func TestRenameGoesThroughSessionStream(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.catalog", Result: map[string]any{"sessions": []any{map[string]any{
				"sessionId": testSession, "title": "旧名字", "titleRevision": 1,
			}}}},
			{Method: "session.rename", Result: map[string]any{
				"title": "新名字", "titleSource": "manual", "revision": 2,
			}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	if _, err := result.base.Catalog().Refresh(ctx, catalogQuery()); err != nil {
		t.Fatalf("读取目录: %v", err)
	}

	entry, err := result.base.Catalog().Rename(ctx, catalogRenameRequest(testSession, "新名字"))
	if err != nil {
		t.Fatalf("改名: %v", err)
	}
	if entry.Title != "新名字" || entry.TitleRevision != 2 {
		t.Fatalf("改名后标题不对: %+v", entry)
	}
	requests := server.RequestsFor("session.rename")
	if len(requests) != 1 {
		t.Fatalf("session.rename 应只发一次，实际 %d", len(requests))
	}
	params := requestParams(t, requests[0])
	if params["sessionId"] != testSession || params["title"] != "新名字" {
		t.Fatalf("改名参数不对: %v", params)
	}
}

// TestPermissionGetUpdateAndTimeline 核对权限档位：打开时读、切换写、时间线事实驱动。
func TestPermissionGetUpdateAndTimeline(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.permission.get", Result: map[string]any{
				"version": 1, "policy": map[string]any{"preset": "workspace-write", "allowedTools": []any{"exec"}, "disallowedTools": []any{}},
			}},
			{Method: "session.permission.update", Result: map[string]any{
				"version": 2, "policy": map[string]any{"preset": "full-access", "allowedTools": []any{}, "disallowedTools": []any{}},
			}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()

	// 打开会话时按服务端策略装入档位。
	deadline := time.Now().Add(2 * time.Second)
	for {
		if tier, ok := result.base.PermissionTier(testSession); ok && tier == "workspace-write" {
			break
		}
		if time.Now().After(deadline) {
			tier, ok := result.base.PermissionTier(testSession)
			t.Fatalf("打开会话后应从服务端读到档位，实际 %q ok=%t", tier, ok)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := len(server.RequestsFor("session.permission.get")); got != 1 {
		t.Fatalf("session.permission.get 应只读一次，实际 %d", got)
	}

	if err := result.base.SetPermissionTier(ctx, testSession, "full-access"); err != nil {
		t.Fatalf("切换档位: %v", err)
	}
	requests := server.RequestsFor("session.permission.update")
	if len(requests) != 1 {
		t.Fatalf("session.permission.update 应只发一次，实际 %d", len(requests))
	}
	params := requestParams(t, requests[0])
	if params["sessionId"] != testSession || params["branchId"] != mainBranchID {
		t.Fatalf("切换档位缺少会话或分支: %v", params)
	}
	if params["expectedVersion"] != float64(1) {
		t.Fatalf("切换档位应带上一版本号: %v", params)
	}
	policy, _ := params["policy"].(map[string]any)
	if policy["preset"] != "full-access" {
		t.Fatalf("切换档位的策略不对: %v", params)
	}
	allowed, _ := policy["allowedTools"].([]any)
	if len(allowed) != 1 || allowed[0] != "exec" {
		t.Fatalf("切换档位不应清空工具列表: %v", policy)
	}
	if tier, ok := result.base.PermissionTier(testSession); !ok || tier != "full-access" {
		t.Fatalf("切换后档位应取回执值，实际 %q ok=%t", tier, ok)
	}

	// 时间线事实 `permission/changed` 驱动同一份状态。
	changed := rawEvent("e60", 60, "permission", "changed", `{"policy":{"preset":"read-only"}}`)
	if err := server.PushEvent(testSession, changed); err != nil {
		t.Fatalf("推送权限变更: %v", err)
	}
	deadline = time.Now().Add(2 * time.Second)
	for {
		if tier, ok := result.base.PermissionTier(testSession); ok && tier == "read-only" {
			break
		}
		if time.Now().After(deadline) {
			tier, ok := result.base.PermissionTier(testSession)
			t.Fatalf("时间线权限变更没有驱动状态，实际 %q ok=%t", tier, ok)
		}
		time.Sleep(5 * time.Millisecond)
	}
	// 没有 permission.changed 通知：控制连接也不该收到它。
	if got := len(server.RequestsFor("session.permission.changed")); got != 0 {
		t.Fatalf("不应请求 permission.changed 通知（%d）", got)
	}
}

// TestPermissionUpdateConflictIsNotOverwritten 核对版本冲突按冲突处理，不覆盖服务端。
func TestPermissionUpdateConflictIsNotOverwritten(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.permission.get", Result: map[string]any{
				"version": 1, "policy": map[string]any{"preset": "workspace-write", "allowedTools": []any{"exec"}, "disallowedTools": []any{}},
			}},
			{Method: "session.permission.update", Error: fakeappserver.AppError(
				-32602, "permission_conflict", "permission policy changed",
			)},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if tier, ok := result.base.PermissionTier(testSession); ok && tier == "workspace-write" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("打开会话后没有读到初始档位")
		}
		time.Sleep(5 * time.Millisecond)
	}

	err := result.base.SetPermissionTier(ctx, testSession, "full-access")
	if !errors.Is(err, ErrPermissionConflict) {
		t.Fatalf("版本冲突应报冲突，实际 %v", err)
	}
	if tier, ok := result.base.PermissionTier(testSession); !ok || tier != "workspace-write" {
		t.Fatalf("冲突后不应覆盖本地档位，实际 %q ok=%t", tier, ok)
	}
	requests := server.RequestsFor("session.permission.update")
	if len(requests) != 1 {
		t.Fatalf("冲突时只应发一次更新，实际 %d", len(requests))
	}
	params := requestParams(t, requests[0])
	if params["expectedVersion"] != float64(1) {
		t.Fatalf("更新应带已知版本号: %v", params)
	}
}

// TestServerArgsForUnifiedBinary 核对统一二进制的自启参数补齐。
func TestServerArgsForUnifiedBinary(t *testing.T) {
	cases := []struct {
		path string
		args []string
		want []string
	}{
		{"/opt/bin/dimcode-server", nil, []string{"app-server"}},
		{"/opt/bin/dim-app-server", nil, nil},
		{"/opt/bin/dimcode-server", []string{"--x"}, []string{"--x"}},
	}
	for _, testCase := range cases {
		got := serverArgsFor(testCase.path, testCase.args)
		if len(got) != len(testCase.want) {
			t.Fatalf("%s %v -> %v，期望 %v", testCase.path, testCase.args, got, testCase.want)
		}
		for index := range got {
			if got[index] != testCase.want[index] {
				t.Fatalf("%s %v -> %v，期望 %v", testCase.path, testCase.args, got, testCase.want)
			}
		}
	}
}

// catalogQuery / catalogRenameRequest 是最小目录请求，避免测试里重复字面量。
func catalogQuery() catalog.Query { return catalog.Query{} }

func catalogRenameRequest(sessionID, title string) catalog.RenameRequest {
	return catalog.RenameRequest{SessionID: sessionID, Title: title}
}

// 保证引用编译期成立。
var _ = protocol.ProtocolVersion

// TestManagementWritesUseRealMethods 核对管理写入落到 alpha.21 的真实方法名。
func TestManagementWritesUseRealMethods(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.wakeup.cancel"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "skills.set_enabled", Result: map[string]any{"ok": true}},
			{Method: "hooks.save", Result: map[string]any{"userRevision": "r2"}},
			{Method: "mcp.catalog", Result: map[string]any{
				"userRevision": "r1", "userApplication": map[string]any{},
				"pluginApplication": map[string]any{}, "servers": []any{}, "invalid": []any{},
			}},
			{Method: "mcp.remove", Result: map[string]any{}},
			{Method: "mcp.import.apply", Result: map[string]any{"applied": 1}},
			{Method: "session.wakeup.cancel", Result: map[string]any{}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	writes := result.base.Writes()

	cases := []struct {
		name   string
		intent write.Intent
		method string
		check  func(t *testing.T, params map[string]any)
	}{
		{
			name: "skill set enabled",
			intent: write.Intent{
				Key: "skill-enabled", Kind: write.KindManagementWrite,
				Params: json.RawMessage(`{"op":"skill.set_enabled","payload":{"skillId":"skill-1","enabled":true}}`),
			},
			method: "skills.set_enabled",
			check: func(t *testing.T, params map[string]any) {
				if params["id"] != "skill-1" || params["enabled"] != true {
					t.Fatalf("技能启停参数不对: %v", params)
				}
			},
		},
		{
			name: "hook write",
			intent: write.Intent{
				Key: "hook-write", Kind: write.KindManagementWrite,
				Params: json.RawMessage(`{"op":"hook.write","payload":{"hook":{"id":"h1","event":"pre","command":"gofmt","source":"user","enabled":true},"expectedRevision":"r1"}}`),
			},
			method: "hooks.save",
			check: func(t *testing.T, params map[string]any) {
				// 真实文档是嵌套结构：hooks -> event -> [{matcher, hooks:[{type,command}]}]
				document, _ := params["document"].(map[string]any)
				hooks, _ := document["hooks"].(map[string]any)
				entries, _ := hooks["pre"].([]any)
				if len(entries) != 1 {
					t.Fatalf("钩子文档不对: %v", params)
				}
				entry, _ := entries[0].(map[string]any)
				commands, _ := entry["hooks"].([]any)
				if len(commands) != 1 {
					t.Fatalf("钩子条目缺少命令: %v", entry)
				}
				command, _ := commands[0].(map[string]any)
				if command["type"] != "command" || command["command"] != "gofmt" {
					t.Fatalf("钩子命令不对: %v", command)
				}
			},
		},
		{
			name: "mcp remove",
			intent: write.Intent{
				Key: "mcp-remove", Kind: write.KindManagementWrite,
				Params: json.RawMessage(`{"op":"tool_server.remove","payload":{"serverId":"srv-1"}}`),
			},
			method: "mcp.remove",
			check: func(t *testing.T, params map[string]any) {
				if params["id"] != "srv-1" || params["source"] != "user" {
					t.Fatalf("MCP 删除参数不对: %v", params)
				}
				if params["expectedVersion"] != "r1" {
					t.Fatalf("MCP 删除应带目录版本: %v", params)
				}
			},
		},
		{
			name: "mcp import apply",
			intent: write.Intent{
				Key: "mcp-import", Kind: write.KindManagementWrite,
				Params: json.RawMessage(`{"op":"tool_server.add","payload":{"server":{"name":"导入","transport":"stdio"},"importSource":"claude-code:user","candidateKey":"cand-1"}}`),
			},
			method: "mcp.import.apply",
			check: func(t *testing.T, params map[string]any) {
				if params["source"] != "claude-code:user" {
					t.Fatalf("MCP 导入来源不对: %v", params)
				}
				ids, _ := params["ids"].([]any)
				if len(ids) != 1 || ids[0] != "cand-1" {
					t.Fatalf("MCP 导入候选不对: %v", params)
				}
			},
		},
		{
			name: "wakeup cancel",
			intent: write.Intent{
				Key: "wakeup-cancel", Kind: write.KindManagementWrite,
				SessionID: testSession,
				Params:    json.RawMessage(`{"op":"wakeup.cancel","payload":{"wakeupId":"w-1"}}`),
			},
			method: "session.wakeup.cancel",
			check: func(t *testing.T, params map[string]any) {
				if params["sessionId"] != testSession || params["wakeupId"] != "w-1" {
					t.Fatalf("唤醒取消参数不对: %v", params)
				}
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			registered, err := writes.Register(testCase.intent)
			if err != nil {
				t.Fatalf("登记意图: %v", err)
			}
			settled, err := writes.Wait(ctx, registered.ID)
			if err != nil || settled.Status != write.StatusSucceeded {
				t.Fatalf("应成功: %v %+v", err, settled)
			}
			requests := server.RequestsFor(testCase.method)
			if len(requests) != 1 {
				t.Fatalf("%s 应只发一次，实际 %d", testCase.method, len(requests))
			}
			testCase.check(t, requestParams(t, requests[0]))
		})
	}

	// workflow.start 在 alpha.21 不注册：明确失败且零请求。
	registered, err := writes.Register(write.Intent{
		Key: "workflow-start", Kind: write.KindManagementWrite,
		Params: json.RawMessage(`{"op":"workflow.start","payload":{"workflowId":"w1"}}`),
	})
	if err != nil {
		t.Fatalf("登记工作流启动: %v", err)
	}
	settled, err := writes.Wait(ctx, registered.ID)
	if err != nil || settled.Status != write.StatusFailed || !strings.Contains(settled.Reason, "能力") {
		t.Fatalf("工作流启动应报能力缺失: %v %+v", err, settled)
	}
	if got := len(server.RequestsFor("workflow.start")); got != 0 {
		t.Fatalf("缺能力时不应请求 workflow.start（%d 次）", got)
	}
}
