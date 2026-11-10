package extensions

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestRealDaemonHooksAndMCPWrites 在真实 alpha.21 上补写回证据：
//   - hooks.save 的文档形状、读回状态、旧版本冲突且不覆盖；
//   - mcp.save/mcp.configuration/mcp.test/mcp.remove 的真实回执形状。
//
// 所有状态都落在临时数据目录；MCP 服务器是无凭据的本地 Python stdio 夹具。
// 日志只输出状态、标识、字段名与布尔结果，不打印文档内容或环境变量值。
func TestRealDaemonHooksAndMCPWrites(t *testing.T) {
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("未设置 DIM_APP_SERVER_BIN：真实写回证据 blocked（缺 App Server 可执行文件）")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dataDir, err := os.MkdirTemp("/tmp", "d11b-write-")
	if err != nil {
		t.Fatalf("创建临时数据目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })

	supervisor, err := daemon.New(evidenceDaemonConfig(dataDir, serverPath))
	if err != nil {
		t.Fatalf("创建连接监督者: %v", err)
	}
	conn, err := supervisor.Connect(ctx)
	if err != nil {
		supervisor.Close(nil)
		t.Fatalf("连接服务端: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer shutdownCancel()
		if result, shutdownErr := supervisor.Shutdown(shutdownCtx); shutdownErr == nil {
			t.Logf("回收自启服务端: stopped=%v", result.Stopped)
		}
		supervisor.Close(nil)
	})
	t.Logf("服务端 %s（profile %s）", conn.Hello.ServerVersion, conn.Hello.DaemonProfile.Kind)
	for _, method := range []string{
		"hooks.set_enabled", "skills.refresh", "skills.reload_catalog",
		"mcp.auth.start", "mcp.auth.complete", "mcp.auth.cancel", "mcp.auth.logout",
	} {
		t.Logf("hello feature %s=%v", method, conn.Hello.HasFeature(method))
	}

	harness := newAppServerHarness(conn.Client)
	harness.overlayWrites = false
	writes := newWriteManager(t, harness)
	service := newService(t, Config{
		Hooks:       harness,
		ToolServers: harness,
		Writer:      writes,
		Logf: func(format string, args ...any) {
			t.Logf(format, args...)
		},
	})

	// 1. hooks.catalog：记录 revision、文档字段名与钩子条数。
	var hookCatalog struct {
		UserRevision string                     `json:"userRevision"`
		UserDocument map[string]json.RawMessage `json:"userDocument"`
		Hooks        []struct {
			Source    string          `json:"source"`
			Event     string          `json:"event"`
			Matcher   json.RawMessage `json:"matcher"`
			Command   string          `json:"command"`
			Timeout   int             `json:"timeout"`
			OnFailure string          `json:"onFailure"`
			Async     bool            `json:"async"`
			State     string          `json:"state"`
		} `json:"hooks"`
	}
	if err := conn.Client.CallInto(ctx, "hooks.catalog", nil, &hookCatalog); err != nil {
		t.Fatalf("读取 hooks.catalog: %v", err)
	}
	t.Logf("hooks.catalog: userRevision=%s userDocumentKeys=%v hooks=%d",
		hookCatalog.UserRevision, mapKeys(hookCatalog.UserDocument), len(hookCatalog.Hooks))
	initialRevision := hookCatalog.UserRevision

	// 2. 写一条最小、无副作用的用户钩子；读回状态。
	hook := Hook{
		Event:   "SessionStart",
		Command: "printf d11b",
		Source:  SourceUser,
		Type:    "command",
	}
	saved, err := service.WriteHookWithRevision(ctx, hook, initialRevision)
	if err != nil || saved.Status != write.StatusSucceeded {
		t.Fatalf("hooks.save: %v %+v", err, saved)
	}
	t.Logf("hooks.save 回执字段=%v revision=%s", rawKeys(saved.Result), extractString(saved.Result, "revision"))

	var afterSave struct {
		UserRevision string `json:"userRevision"`
		Hooks        []struct {
			Source  string `json:"source"`
			Event   string `json:"event"`
			Command string `json:"command"`
			State   string `json:"state"`
		} `json:"hooks"`
	}
	if err := conn.Client.CallInto(ctx, "hooks.catalog", nil, &afterSave); err != nil {
		t.Fatalf("写后读取 hooks.catalog: %v", err)
	}
	t.Logf("hooks.catalog 写后: userRevision=%s hooks=%d", afterSave.UserRevision, len(afterSave.Hooks))
	found := false
	for _, item := range afterSave.Hooks {
		if item.Source == "user" && item.Event == "SessionStart" && item.Command == "printf d11b" {
			found = true
			t.Logf("读回钩子: source=%s event=%s state=%s", item.Source, item.Event, item.State)
		}
	}
	if !found {
		t.Fatalf("hooks.catalog 没有读回刚写入的钩子: %+v", afterSave.Hooks)
	}

	// 领域读取也要把真实扁平条目映射出来。
	hookSnapshot, err := service.ListHooks(ctx)
	if err != nil {
		t.Fatalf("领域读取钩子: %v", err)
	}
	domainFound := false
	for _, item := range hookSnapshot.Items {
		if item.Source == SourceUser && item.Event == "SessionStart" &&
			item.Command == "printf d11b" && item.State == "ok" {
			domainFound = true
		}
	}
	if !domainFound {
		t.Fatalf("领域读取没有映射真实钩子: %+v", hookSnapshot.Items)
	}
	t.Logf("领域钩子读取: hooks=%d writable=%v", len(hookSnapshot.Items), hookSnapshot.Items[0].Writable())

	// 3. 用旧版本再写一次：必须冲突，且不覆盖已写入的钩子。
	stale, err := service.WriteHookWithRevision(ctx, Hook{
		Event:   "SessionStart",
		Command: "printf stale",
		Source:  SourceUser,
		Type:    "command",
	}, initialRevision)
	if err != nil {
		t.Fatalf("旧版本写入返回等待错误: %v", err)
	}
	if stale.Status != write.StatusFailed {
		t.Fatalf("旧版本写入没有明确失败: %+v", stale)
	}
	if !strings.Contains(stale.Reason, "hooks_conflict") {
		t.Fatalf("旧版本写入没有返回 hooks_conflict: %s", stale.Reason)
	}
	t.Logf("hooks.save 旧版本: status=%s conflict=true", stale.Status)

	var afterStale struct {
		UserRevision string `json:"userRevision"`
		Hooks        []struct {
			Event   string `json:"event"`
			Command string `json:"command"`
		} `json:"hooks"`
	}
	if err := conn.Client.CallInto(ctx, "hooks.catalog", nil, &afterStale); err != nil {
		t.Fatalf("冲突后读取 hooks.catalog: %v", err)
	}
	commands := make([]string, 0, len(afterStale.Hooks))
	for _, item := range afterStale.Hooks {
		commands = append(commands, item.Command)
	}
	t.Logf("冲突后 hooks.catalog: userRevision=%s commands=%v", afterStale.UserRevision, commands)
	if len(afterStale.Hooks) != 1 || afterStale.Hooks[0].Command != "printf d11b" {
		t.Fatalf("冲突覆盖了已写入的钩子: %+v", afterStale.Hooks)
	}

	// 4. MCP：保存一条无凭据的本地 stdio 服务器。
	scriptPath := filepath.Join(dataDir, "mcp_server.py")
	if err := os.WriteFile(scriptPath, []byte(minimalMCPServerScript), 0o700); err != nil {
		t.Fatalf("写入 MCP 夹具: %v", err)
	}
	var mcpCatalog struct {
		UserRevision string `json:"userRevision"`
		Servers      []struct {
			ID        string `json:"id"`
			Source    string `json:"source"`
			Enabled   bool   `json:"enabled"`
			Transport string `json:"transport"`
		} `json:"servers"`
	}
	if err := conn.Client.CallInto(ctx, "mcp.catalog", nil, &mcpCatalog); err != nil {
		t.Fatalf("读取 mcp.catalog: %v", err)
	}
	t.Logf("mcp.catalog: userRevision=%s servers=%d", mcpCatalog.UserRevision, len(mcpCatalog.Servers))
	mcpInitialRevision := mcpCatalog.UserRevision

	savedMCP, err := service.AddToolServer(ctx, ToolServerSpec{
		Name:            "d11b-mcp",
		Transport:       ToolTransportStdio,
		Command:         "python3",
		Args:            []string{scriptPath},
		Source:          SourceUser,
		ExpectedVersion: mcpInitialRevision,
	})
	if err != nil || savedMCP.Status != write.StatusSucceeded {
		t.Fatalf("mcp.save: %v %+v", err, savedMCP)
	}
	t.Logf("mcp.save 回执字段=%v", rawKeys(savedMCP.Result))

	servers, err := service.ListToolServers(ctx)
	if err != nil {
		t.Fatalf("写后读取 MCP 目录: %v", err)
	}
	var savedServer *ToolServer
	for index := range servers.Items {
		if servers.Items[index].ID == "d11b-mcp" {
			savedServer = &servers.Items[index]
		}
	}
	if savedServer == nil {
		t.Fatalf("mcp.catalog 没有读回刚保存的服务器: %+v", servers.Items)
	}
	t.Logf("读回 MCP: id=%s source=%s enabled=%v transport=%s",
		savedServer.ID, savedServer.Source, savedServer.Enabled, savedServer.Transport)

	var mcpAfterSave struct {
		UserRevision string `json:"userRevision"`
	}
	if err := conn.Client.CallInto(ctx, "mcp.catalog", nil, &mcpAfterSave); err != nil {
		t.Fatalf("写后读取 mcp.catalog: %v", err)
	}

	// 5. 读回配置：只记录字段名、类型与计数。
	var configuration struct {
		Revision           string `json:"revision"`
		Enabled            bool   `json:"enabled"`
		SecretsInUserScope bool   `json:"secretsInUserScope"`
		Configuration      struct {
			Type                string   `json:"type"`
			Command             string   `json:"command"`
			ArgsCount           int      `json:"argsCount"`
			Cwd                 *string  `json:"cwd"`
			EnvNames            []string `json:"envNames"`
			ReferencedVariables []string `json:"referencedVariables"`
			UnsetVariables      []string `json:"unsetVariables"`
		} `json:"configuration"`
	}
	if err := conn.Client.CallInto(ctx, "mcp.configuration", map[string]any{
		"id": "d11b-mcp", "source": "user",
	}, &configuration); err != nil {
		t.Fatalf("读取 mcp.configuration: %v", err)
	}
	t.Logf("mcp.configuration: revision=%s enabled=%v secretsInUserScope=%v configurationFields=%v type=%s argsCount=%d envNames=%d",
		configuration.Revision, configuration.Enabled, configuration.SecretsInUserScope,
		[]string{"type", "command", "argsCount", "cwd", "envNames", "referencedVariables", "unsetVariables"},
		configuration.Configuration.Type, configuration.Configuration.ArgsCount, len(configuration.Configuration.EnvNames))

	// 6. mcp.test：真实探测本地夹具，只记录状态、耗时与工具数/名。
	probe, err := service.ProbeToolServer(ctx, "d11b-mcp")
	if err != nil {
		t.Fatalf("mcp.test: %v", err)
	}
	toolNames := make([]string, 0, len(probe.Tools))
	for _, tool := range probe.Tools {
		toolNames = append(toolNames, tool.Name)
	}
	t.Logf("mcp.test: status=%s reachable=%v latencyMs=%d tools=%v",
		probe.Status, probe.Reachable, probe.LatencyMillis, toolNames)
	if !probe.Reachable || len(probe.Tools) == 0 {
		t.Fatalf("mcp.test 没有返回可用工具: %+v", probe)
	}

	// 7. mcp.remove：用写后版本清理；再读目录确认消失。
	removed, err := service.RemoveToolServerWithVersion(ctx, "d11b-mcp", mcpAfterSave.UserRevision)
	if err != nil || removed.Status != write.StatusSucceeded {
		t.Fatalf("mcp.remove: %v %+v", err, removed)
	}
	t.Logf("mcp.remove 回执字段=%v", rawKeys(removed.Result))

	var afterRemove struct {
		Servers []json.RawMessage `json:"servers"`
	}
	if err := conn.Client.CallInto(ctx, "mcp.catalog", nil, &afterRemove); err != nil {
		t.Fatalf("删除后读取 mcp.catalog: %v", err)
	}
	t.Logf("mcp.remove 后 servers=%d", len(afterRemove.Servers))
	if len(afterRemove.Servers) != 0 {
		t.Fatalf("删除后 MCP 目录仍有条目: %d", len(afterRemove.Servers))
	}
}

// minimalMCPServerScript 是一个无凭据、只响应 initialize 与 tools/list 的
// MCP stdio 夹具；它不访问网络、不读写外部文件。
const minimalMCPServerScript = `import sys, json
def send(obj):
    sys.stdout.write(json.dumps(obj) + "\n")
    sys.stdout.flush()
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    msg = json.loads(line)
    method = msg.get("method")
    if method == "initialize":
        send({"jsonrpc":"2.0","id":msg["id"],"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"d11b","version":"1"}}})
    elif method == "tools/list":
        send({"jsonrpc":"2.0","id":msg["id"],"result":{"tools":[{"name":"echo","description":"echo","inputSchema":{"type":"object","properties":{}}}]}})
    elif "id" in msg:
        send({"jsonrpc":"2.0","id":msg["id"],"result":{}})
`

func mapKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func rawKeys(raw json.RawMessage) []string {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil
	}
	return mapKeys(object)
}

func extractString(raw json.RawMessage, key string) string {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return ""
	}
	var value string
	_ = json.Unmarshal(object[key], &value)
	return value
}
