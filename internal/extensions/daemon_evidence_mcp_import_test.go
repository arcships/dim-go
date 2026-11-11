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
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
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

	// 1. 已知来源：id/client/scope/path/state/entries，不读取条目内容。
	var sourcesReply struct {
		Sources []struct {
			ID      string `json:"id"`
			Client  string `json:"client"`
			Scope   string `json:"scope"`
			Path    string `json:"path"`
			State   string `json:"state"`
			Entries *int   `json:"entries"`
			Error   string `json:"error"`
		} `json:"sources"`
	}
	if err := conn.Client.CallInto(ctx, "mcp.import.sources", map[string]any{"cwd": workDir}, &sourcesReply); err != nil {
		t.Fatalf("mcp.import.sources: %v", err)
	}
	chosen := ""
	for _, source := range sourcesReply.Sources {
		entries := -1
		if source.Entries != nil {
			entries = *source.Entries
		}
		t.Logf("import.source id=%s client=%s scope=%s state=%s entries=%d path=%s",
			source.ID, source.Client, source.Scope, source.State, entries, sanitizeHomePath(source.Path))
		if chosen == "" && source.State == "ready" && source.Entries != nil && *source.Entries > 0 {
			chosen = source.ID
		}
	}
	if chosen == "" {
		t.Skip("没有可导入的 ready 来源：真实导入证据 blocked（缺真实配置样本）")
	}

	// 2. 预览：target source 固定 user，from 为来源 id。
	var preview struct {
		Revision       string `json:"revision"`
		SourceRevision string `json:"sourceRevision"`
		Store          *struct {
			ID     string `json:"id"`
			Client string `json:"client"`
			Scope  string `json:"scope"`
			Path   string `json:"path"`
		} `json:"store"`
		Entries []struct {
			ID        string `json:"id"`
			Transport string `json:"transport"`
			Status    string `json:"status"`
			Changes   []struct {
				Field string `json:"field"`
				Kind  string `json:"kind"`
			} `json:"changes"`
			Unsupported []struct {
				Field  string `json:"field"`
				Effect string `json:"effect"`
			} `json:"unsupported"`
			Omitted int    `json:"omitted"`
			Error   string `json:"error"`
		} `json:"entries"`
		UnknownTopLevel []string `json:"unknownTopLevel"`
	}
	if err := conn.Client.CallInto(ctx, "mcp.import.preview", map[string]any{
		"cwd":    workDir,
		"source": "user",
		"from":   chosen,
	}, &preview); err != nil {
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
		if importID == "" && (entry.Status == "changed" || entry.Status == "identical") {
			importID = entry.ID
		}
	}
	if importID == "" {
		t.Skip("预览没有可导入条目：真实导入证据 blocked（缺可用样本）")
	}

	// 3. 只应用这一条；resolutions 是逐条动作，不是一次性改全量。
	applyParams := map[string]any{
		"cwd":                    workDir,
		"source":                 "user",
		"from":                   chosen,
		"expectedVersion":        preview.Revision,
		"expectedSourceRevision": preview.SourceRevision,
		"resolutions": []any{map[string]any{
			"id": importID, "action": "import",
		}},
	}
	var applied struct {
		Revision string `json:"revision"`
	}
	if err := conn.Client.CallInto(ctx, "mcp.import.apply", applyParams, &applied); err != nil {
		t.Fatalf("mcp.import.apply: %v", err)
	}
	t.Logf("apply revision=%s imported=%s", applied.Revision, importID)

	var catalog struct {
		UserRevision string `json:"userRevision"`
		Servers      []struct {
			ID        string `json:"id"`
			Source    string `json:"source"`
			Enabled   bool   `json:"enabled"`
			Transport string `json:"transport"`
		} `json:"servers"`
	}
	if err := conn.Client.CallInto(ctx, "mcp.catalog", nil, &catalog); err != nil {
		t.Fatalf("apply 后 mcp.catalog: %v", err)
	}
	found := false
	for _, server := range catalog.Servers {
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
	_, staleErr := conn.Client.CallWithTimeout(ctx, "mcp.import.apply", applyParams, 10*time.Second)
	var rpcErr *protocol.RPCError
	conflict := errors.As(staleErr, &rpcErr) && rpcErr.AppError() == "mcp_settings_conflict"
	t.Logf("apply stale err=%v conflict=%v", staleErr, conflict)
	if !conflict {
		t.Fatalf("旧目标版本没有返回 mcp_settings_conflict: %v", staleErr)
	}

	// 5. 清理：删除导入条目，来源文件保持不变。
	var after struct {
		UserRevision string `json:"userRevision"`
	}
	if err := conn.Client.CallInto(ctx, "mcp.catalog", nil, &after); err != nil {
		t.Fatalf("清理前 mcp.catalog: %v", err)
	}
	rawRemove, err := conn.Client.CallWithTimeout(ctx, "mcp.remove", map[string]any{
		"id": importID, "source": "user", "expectedVersion": after.UserRevision,
	}, 10*time.Second)
	if err != nil {
		t.Fatalf("清理 mcp.remove: %v", err)
	}
	t.Logf("cleanup mcp.remove shape=%s", shapeOf(rawRemove))
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
	home, err := os.UserHomeDir()
	if err == nil && home != "" && len(path) > len(home) && path[:len(home)] == home {
		return "$HOME" + path[len(home):]
	}
	return path
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
