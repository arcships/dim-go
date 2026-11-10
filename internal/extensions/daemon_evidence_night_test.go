package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestRealDaemonNightAuthorization 覆盖夜间授权的三件真实能力：
//   - hooks.set_enabled 的项目作用域启停、读回与版本冲突；
//   - skills.refresh 的版本变化与 skills.reload_catalog 的能力缺失；
//   - mcp.auth.* 的方法接入与无认证服务器时的服务端错误。
//
// 日志只留状态、标识、字段名与布尔结果。
func TestRealDaemonNightAuthorization(t *testing.T) {
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("未设置 DIM_APP_SERVER_BIN：真实夜间授权证据 blocked（缺 App Server 可执行文件）")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dataDir, err := os.MkdirTemp("/tmp", "d11b-night-")
	if err != nil {
		t.Fatalf("创建临时数据目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	workDir := filepath.Join(dataDir, "project")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建项目目录: %v", err)
	}

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
		t.Logf("hello %s=%v", method, conn.Hello.HasFeature(method))
	}

	harness := newAppServerHarness(conn.Client)
	harness.overlayWrites = false
	writes := newWriteManager(t, harness)
	service := newService(t, Config{
		Hooks:        harness,
		ProjectHooks: harness,
		Skills:       harness,
		ToolServers:  harness,
		Features:     conn,
		Writer:       writes,
		Logf: func(format string, args ...any) {
			t.Logf(format, args...)
		},
	})

	// 1. hooks.set_enabled：项目作用域启停、读回与冲突。
	project, err := service.ReadProjectHooks(ctx, workDir)
	if err != nil {
		t.Fatalf("读取项目钩子: %v", err)
	}
	initialRevision := project.ProjectsRevision
	initialEnabled := projectStateEnabled(project, "project")
	t.Logf("项目钩子初始: projectsRevision=%s enabled=%v sources=%d",
		initialRevision, initialEnabled, len(project.Sources))

	enabledView, err := service.SetProjectHooksEnabled(ctx, workDir, true, initialRevision)
	if err != nil || enabledView.Status != write.StatusSucceeded {
		t.Fatalf("启用项目钩子: %v %+v", err, enabledView)
	}
	project, err = service.ReadProjectHooks(ctx, workDir)
	if err != nil {
		t.Fatalf("启用后读取项目钩子: %v", err)
	}
	enabledRevision := project.ProjectsRevision
	if !projectStateEnabled(project, "project") {
		t.Fatalf("启用后项目来源仍为关闭: %+v", project.Sources)
	}
	t.Logf("项目钩子启用后: projectsRevision=%s enabled=true", enabledRevision)

	disabledView, err := service.SetProjectHooksEnabled(ctx, workDir, false, enabledRevision)
	if err != nil || disabledView.Status != write.StatusSucceeded {
		t.Fatalf("停用项目钩子: %v %+v", err, disabledView)
	}
	project, err = service.ReadProjectHooks(ctx, workDir)
	if err != nil {
		t.Fatalf("停用后读取项目钩子: %v", err)
	}
	if projectStateEnabled(project, "project") {
		t.Fatalf("停用后项目来源仍为开启: %+v", project.Sources)
	}
	t.Logf("项目钩子停用后: projectsRevision=%s enabled=false", project.ProjectsRevision)

	staleView, err := service.SetProjectHooksEnabled(ctx, workDir, true, initialRevision)
	if err != nil {
		t.Fatalf("旧版本启停返回等待错误: %v", err)
	}
	if staleView.Status != write.StatusFailed || !strings.Contains(staleView.Reason, "hooks_conflict") {
		t.Fatalf("旧版本启停没有按冲突处理: %+v", staleView)
	}
	project, err = service.ReadProjectHooks(ctx, workDir)
	if err != nil {
		t.Fatalf("冲突后读取项目钩子: %v", err)
	}
	if projectStateEnabled(project, "project") {
		t.Fatalf("冲突覆盖了项目钩子状态: %+v", project.Sources)
	}
	t.Logf("项目钩子旧版本: status=failed conflict=true 当前enabled=false")

	// 2. skills.refresh：刷新后版本变化；skills.reload_catalog 能力缺失零请求。
	before, err := service.ListSkills(ctx)
	if err != nil {
		t.Fatalf("刷新前读取技能: %v", err)
	}
	versionBefore := service.SkillCatalogVersion()
	t.Logf("技能刷新前: version=%d entries=%d", versionBefore, len(before.Items))

	refreshed, err := service.RefreshSkills(ctx)
	if err != nil || refreshed.Status != write.StatusSucceeded {
		t.Fatalf("skills.refresh: %v %+v", err, refreshed)
	}
	var refreshResult struct {
		Version uint64       `json:"version"`
		Issues  []SkillIssue `json:"issues"`
	}
	if err := json.Unmarshal(refreshed.Result, &refreshResult); err != nil {
		t.Fatalf("解析 skills.refresh 回执: %v", err)
	}
	after, err := service.ListSkills(ctx)
	if err != nil {
		t.Fatalf("刷新后读取技能: %v", err)
	}
	versionAfter := service.SkillCatalogVersion()
	if versionAfter <= versionBefore {
		t.Fatalf("刷新后版本没有变化: %d -> %d", versionBefore, versionAfter)
	}
	t.Logf("skills.refresh: replyVersion=%d issues=%d; catalog version=%d -> %d entries=%d",
		refreshResult.Version, len(refreshResult.Issues), versionBefore, versionAfter, len(after.Items))

	tableBefore := len(writes.Table())
	if _, err := service.ReloadSkillCatalog(ctx); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("skills.reload_catalog 没有按能力缺失: %v", err)
	}
	if afterTable := len(writes.Table()); afterTable != tableBefore {
		t.Fatalf("skills.reload_catalog 缺能力时登记了写意图: %d -> %d", tableBefore, afterTable)
	}
	t.Logf("skills.reload_catalog: capability_missing=true zero_request=true")

	// 3. mcp.auth.*：方法接入；stdio 服务器不支持 OAuth，成功认证需要真实认证服务器。
	var mcpCatalog struct {
		UserRevision string `json:"userRevision"`
	}
	if err := conn.Client.CallInto(ctx, "mcp.catalog", nil, &mcpCatalog); err != nil {
		t.Fatalf("读取 mcp.catalog: %v", err)
	}
	scriptPath := filepath.Join(dataDir, "mcp_server.py")
	if err := os.WriteFile(scriptPath, []byte(minimalMCPServerScript), 0o700); err != nil {
		t.Fatalf("写入 MCP 夹具: %v", err)
	}
	saved, err := service.AddToolServer(ctx, ToolServerSpec{
		Name:            "d11b-auth",
		Transport:       ToolTransportStdio,
		Command:         "python3",
		Args:            []string{scriptPath},
		Source:          SourceUser,
		ExpectedVersion: mcpCatalog.UserRevision,
	})
	if err != nil || saved.Status != write.StatusSucceeded {
		t.Fatalf("保存 MCP 服务器: %v %+v", err, saved)
	}
	target := MCPTarget{ID: "d11b-auth", Source: SourceUser}

	started, err := service.StartMCPAuth(ctx, target, "")
	if err != nil {
		t.Fatalf("mcp.auth.start 返回等待错误: %v", err)
	}
	if started.Status != write.StatusFailed || !strings.Contains(started.Reason, "mcp_auth_unsupported") {
		t.Fatalf("stdio 服务器认证开始没有返回不支持: %+v", started)
	}
	t.Logf("mcp.auth.start: status=failed reason=mcp_auth_unsupported（无 OAuth 认证服务器）")

	attemptID := "00000000-0000-4000-8000-000000000000"
	completed, err := service.CompleteMCPAuth(ctx, attemptID, "http://localhost:1455/callback")
	if err != nil {
		t.Fatalf("mcp.auth.complete 返回等待错误: %v", err)
	}
	t.Logf("mcp.auth.complete: status=%s reason=%s", completed.Status, shortReason(completed.Reason))
	canceled, err := service.CancelMCPAuth(ctx, attemptID)
	if err != nil {
		t.Fatalf("mcp.auth.cancel 返回等待错误: %v", err)
	}
	t.Logf("mcp.auth.cancel: status=%s reason=%s", canceled.Status, shortReason(canceled.Reason))
	loggedOut, err := service.LogoutMCPAuth(ctx, target)
	if err != nil {
		t.Fatalf("mcp.auth.logout 返回等待错误: %v", err)
	}
	if loggedOut.Status != write.StatusFailed || !strings.Contains(loggedOut.Reason, "mcp_auth_unsupported") {
		t.Fatalf("stdio 服务器登出没有返回不支持: %+v", loggedOut)
	}
	t.Logf("mcp.auth.logout: status=failed reason=mcp_auth_unsupported")

	// 清理：删除测试服务器。
	var afterMCP struct {
		UserRevision string `json:"userRevision"`
	}
	if err := conn.Client.CallInto(ctx, "mcp.catalog", nil, &afterMCP); err != nil {
		t.Fatalf("清理前读取 mcp.catalog: %v", err)
	}
	if _, err := service.RemoveToolServerWithVersion(ctx, "d11b-auth", afterMCP.UserRevision); err != nil {
		t.Fatalf("清理 MCP 服务器: %v", err)
	}
}

func projectStateEnabled(catalog ProjectHookCatalog, source string) bool {
	for _, state := range catalog.Sources {
		if string(state.Source) == source {
			return state.Enabled
		}
	}
	return false
}

func shortReason(reason string) string {
	if len(reason) > 120 {
		return reason[:120] + "..."
	}
	if reason == "" {
		return "<empty>"
	}
	return reason
}
