package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
)

// TestRealDaemonMCPImport 调查真实 mcp.import.sources/preview/apply：
//   - 列出服务端扫描的已知客户端配置来源与状态；
//   - 对一个 ready 来源做预览，记录字段名/类型与状态；
//   - 应用一条无害条目，验证只写勾选条目；
//   - 用旧目标版本再次应用，确认 mcp_settings_conflict；
//   - 删除导入条目并清理。
//
// 来源文件只读；写入只落临时数据目录。日志不打印任何配置值。
func TestRealDaemonMCPImport(t *testing.T) {
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("未设置 DIM_APP_SERVER_BIN：真实 MCP 导入证据 blocked（缺 App Server 可执行文件）")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dataDir, err := os.MkdirTemp("/tmp", "d11b-import-")
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

	harness := newAppServerHarness(conn.Client)
	harness.overlayWrites = false
	writes := newWriteManager(t, harness)
	service := newService(t, Config{
		ToolServers: harness,
		MCPImports:  harness,
		Features:    conn,
		Writer:      writes,
		Logf: func(format string, args ...any) {
			t.Logf(format, args...)
		},
	})

	// 1. 已知来源：id/client/scope/path/state/entries，不读取条目内容。
	sources, err := service.ListMCPImportSources(ctx, workDir)
	if err != nil {
		t.Fatalf("mcp.import.sources: %v", err)
	}
	chosen := ""
	for _, source := range sources {
		entries := -1
		if source.Entries != nil {
			entries = *source.Entries
		}
		t.Logf("import.source id=%s client=%s scope=%s state=%s entries=%d path=%s",
			source.ID, source.Client, source.Scope, source.State, entries, sanitizeHomePath(source.Path))
		if chosen == "" && source.State == MCPImportSourceReady && source.Entries != nil && *source.Entries > 0 {
			chosen = source.ID
		}
	}
	if chosen == "" {
		t.Skip("没有可导入的 ready 来源：真实导入证据 blocked（缺真实配置样本）")
	}

	// 2. 预览：target source 固定 user，from 为来源 id。
	preview, err := service.PreviewMCPImport(ctx, MCPImportPreviewRequest{
		Cwd:    workDir,
		Source: SourceUser,
		From:   chosen,
	})
	if err != nil {
		t.Fatalf("mcp.import.preview: %v", err)
	}
	storeID := ""
	if preview.Store != nil {
		storeID = preview.Store.ID
	}
	t.Logf("preview revision=%s sourceRevision=%s store=%s entries=%d unknownTopLevel=%v",
		preview.Revision, shortDigest(preview.SourceRevision), storeID, len(preview.Entries), preview.UnknownTopLevel)

	importID := ""
	for _, entry := range preview.Entries {
		fields := make([]string, 0, len(entry.Changes))
		for _, change := range entry.Changes {
			fields = append(fields, change.Field+":"+change.Kind)
		}
		t.Logf("preview.entry id=%s status=%s transport=%s changes=%v unsupported=%d omitted=%d",
			entry.ID, entry.Status, entry.Transport, fields, len(entry.Unsupported), entry.Omitted)
		if importID == "" && (entry.Status == MCPImportChanged || entry.Status == MCPImportIdentical) {
			importID = entry.ID
		}
	}
	if importID == "" {
		t.Skip("预览没有可导入条目：真实导入证据 blocked（缺可用样本）")
	}

	// 3. 只应用这一条；resolutions 是逐条动作。
	applyRequest := MCPImportApplyRequest{
		Cwd:                    workDir,
		Source:                 SourceUser,
		From:                   chosen,
		ExpectedVersion:        preview.Revision,
		ExpectedSourceRevision: preview.SourceRevision,
		Resolutions:            []MCPImportResolution{{ID: importID, Action: MCPImportActionImport}},
	}
	applied, err := service.ApplyMCPImport(ctx, applyRequest)
	if err != nil {
		t.Fatalf("mcp.import.apply: %v", err)
	}
	t.Logf("apply revision=%s imported=%s", applied.Revision, importID)

	servers, err := service.ListToolServers(ctx)
	if err != nil {
		t.Fatalf("apply 后 mcp.catalog: %v", err)
	}
	found := false
	for _, server := range servers.Items {
		t.Logf("catalog.server id=%s source=%s enabled=%v transport=%s",
			server.ID, server.Source, server.Enabled, server.Transport)
		if server.ID == importID {
			found = true
		}
	}
	if !found {
		t.Fatalf("导入后目录没有目标条目 %s", importID)
	}

	// 4. 用旧目标版本再次应用：必须 mcp_settings_conflict，且不重复写入。
	_, staleErr := service.ApplyMCPImport(ctx, applyRequest)
	conflict := errors.Is(staleErr, ErrImportConflict)
	t.Logf("apply stale err=%v conflict=%v", staleErr, conflict)
	if !conflict {
		t.Fatalf("旧目标版本没有返回 ErrImportConflict: %v", staleErr)
	}

	// 5. 清理：删除导入条目，来源文件保持不变。
	removed, err := service.RemoveToolServerWithVersion(ctx, importID, applied.Revision)
	if err != nil {
		t.Fatalf("清理 mcp.remove: %v", err)
	}
	t.Logf("cleanup mcp.remove shape=%s", shapeOf(removed.Result))

}

func shortDigest(value string) string {
	if value == "" {
		return "<empty>"
	}
	if len(value) > 12 {
		value = value[:12]
	}
	return "sha256:" + value + "..."
}

func sanitizeHomePath(path string) string {
	if path == "" {
		return "<none>"
	}
	return "<client-config>"
}

func shapeOf(raw json.RawMessage) string {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "<invalid>"
	}
	return shapeValueOf(value, 0)
}

func shapeValueOf(value any, depth int) string {
	if depth > 3 {
		return "..."
	}
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, key+":"+shapeValueOf(typed[key], depth+1))
		}
		return "{" + joinStrings(parts) + "}"
	case []any:
		if len(typed) == 0 {
			return "[]"
		}
		return "[" + shapeValueOf(typed[0], depth+1) + "]"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "bool"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func joinStrings(parts []string) string {
	result := ""
	for index, part := range parts {
		if index > 0 {
			result += ", "
		}
		result += part
	}
	return result
}
