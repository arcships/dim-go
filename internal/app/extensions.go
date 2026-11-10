package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/extensions"
)

// 扩展与后台领域在服务端上的真实方法名（alpha.21 实测，见 D1 §3.7）。
//
// 与第一版的差别：工具服务器叫 mcp.*、计划任务叫 schedule.*、唤醒是会话方法
// session.wakeup.*、技能与钩子的读取是 skills.catalog / hooks.catalog。
// workflow.* 在 alpha.21 不注册，按能力缺失处理，不发请求。
const (
	methodPluginList          = "plugin.list"
	methodPluginInstall       = "plugin.install"
	methodPluginEnable        = "plugin.enable"
	methodPluginDisable       = "plugin.disable"
	methodPluginUpdate        = "plugin.update"
	methodPluginRemove        = "plugin.remove"
	methodSkillsCatalog       = "skills.catalog"
	methodSkillsInstall       = "skills.install"
	methodSkillsSetEnabled    = "skills.set_enabled"
	methodHooksCatalog        = "hooks.catalog"
	methodHooksSave           = "hooks.save"
	methodMCPCatalog          = "mcp.catalog"
	methodMCPSave             = "mcp.save"
	methodMCPRemove           = "mcp.remove"
	methodMCPSetEnabled       = "mcp.set_enabled"
	methodMCPTest             = "mcp.test"
	methodMCPImportPreview    = "mcp.import.preview"
	methodMCPImportApply      = "mcp.import.apply"
	methodScheduleList        = "schedule.list"
	methodSessionWakeupList   = "session.wakeup.list"
	methodSessionWakeupCancel = "session.wakeup.cancel"
)

// extensionReaders 是扩展与后台领域的七个读取方。
//
// 七个接口都落在同一个类型上：它们共用一条控制连接，wire 形状只在集成层出现一次。
// 只有读取，不探测、不安装、不改配置。会话方法（唤醒）走会话自己的流。
type extensionReaders struct {
	app *App
}

// ReadPlugins 实现 extensions.PluginReader（`plugin.list` 返回数组）。
func (r *extensionReaders) ReadPlugins(ctx context.Context) ([]extensions.Plugin, error) {
	client, err := r.app.requireControl("读取插件清单")
	if err != nil {
		return nil, err
	}
	var entries []struct {
		Name    string   `json:"name"`
		ID      string   `json:"id"`
		Use     *bool    `json:"use"`
		Off     *bool    `json:"off"`
		Locked  *bool    `json:"locked"`
		Status  string   `json:"status"`
		Version string   `json:"version"`
		Error   string   `json:"error"`
		Refused string   `json:"refused"`
		Missing []string `json:"missing"`
	}
	if err := client.CallInto(ctx, methodPluginList, nil, &entries); err != nil {
		return nil, err
	}
	plugins := make([]extensions.Plugin, 0, len(entries))
	for _, entry := range entries {
		id := firstNonEmpty(entry.Name, entry.ID)
		if id == "" {
			continue
		}
		enabled := true
		if entry.Off != nil {
			enabled = !*entry.Off
		}
		locked := false
		if entry.Locked != nil {
			locked = *entry.Locked
		}
		use := ""
		if entry.Use != nil && *entry.Use {
			use = "enabled"
		}
		plugins = append(plugins, extensions.Plugin{
			ID:        id,
			Name:      firstNonEmpty(entry.Name, id),
			Version:   entry.Version,
			Enabled:   enabled,
			Installed: true,
			// 服务端清单里没有「可选但未安装」的条目，这里恒为假。
			Optional: false,
			Use:      use,
			Locked:   locked,
			Status:   entry.Status,
			Error:    firstNonEmpty(entry.Error, entry.Refused),
		})
	}
	return plugins, nil
}

// ReadSkills 实现 extensions.SkillReader（`skills.catalog`）。
func (r *extensionReaders) ReadSkills(ctx context.Context) ([]extensions.Skill, error) {
	client, err := r.app.requireControl("读取技能清单")
	if err != nil {
		return nil, err
	}
	raw, err := client.Call(ctx, methodSkillsCatalog, nil)
	if err != nil {
		return nil, err
	}
	catalog, err := protocol.ParseSkillsCatalogResult(raw)
	if err != nil {
		return nil, fmt.Errorf("技能目录无法解析: %w", err)
	}
	skills := make([]extensions.Skill, 0, len(catalog.Entries))
	for _, entry := range catalog.Entries {
		skills = append(skills, extensions.Skill{
			ID:          entry.ID,
			Name:        firstNonEmpty(entry.Name, entry.ID),
			Description: entry.Description,
			Source:      sourceFromWire(entry.Source),
			Installed:   true,
			Enabled:     entry.Enabled,
		})
	}
	return skills, nil
}

// ReadHooks 实现 extensions.HookReader（`hooks.catalog`）。
//
// 用户文档与项目文档由服务端原样给出；本函数只把 hooks[] 里的条目映射成只读视图，
// 认不出的字段留空，不猜结构。
func (r *extensionReaders) ReadHooks(ctx context.Context) ([]extensions.Hook, error) {
	client, err := r.app.requireControl("读取钩子清单")
	if err != nil {
		return nil, err
	}
	raw, err := client.Call(ctx, methodHooksCatalog, nil)
	if err != nil {
		return nil, err
	}
	catalog, err := protocol.ParseHooksCatalogResult(raw)
	if err != nil {
		return nil, fmt.Errorf("钩子目录无法解析: %w", err)
	}
	hooks := make([]extensions.Hook, 0, len(catalog.Hooks))
	for index, item := range catalog.Hooks {
		var entry struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Event   string `json:"event"`
			Command string `json:"command"`
			Source  string `json:"source"`
			Enabled *bool  `json:"enabled"`
		}
		if err := json.Unmarshal(item, &entry); err != nil {
			continue
		}
		id := firstNonEmpty(entry.ID, entry.Event+":"+entry.Command)
		if id == "" {
			id = fmt.Sprintf("hook-%d", index)
		}
		enabled := true
		if entry.Enabled != nil {
			enabled = *entry.Enabled
		}
		hooks = append(hooks, extensions.Hook{
			ID:      id,
			Name:    firstNonEmpty(entry.Name, id),
			Event:   entry.Event,
			Command: entry.Command,
			Source:  sourceFromWire(entry.Source),
			Enabled: enabled,
		})
	}
	return hooks, nil
}

// ReadToolServers 实现 extensions.ToolServerReader 的清单读取（`mcp.catalog`）。
//
// 清单只给 id/source/enabled/transport/auth；命令、参数与 URL 不在清单里，
// 因此这些字段留空，不推断。
func (r *extensionReaders) ReadToolServers(ctx context.Context) ([]extensions.ToolServer, error) {
	client, err := r.app.requireControl("读取工具服务器清单")
	if err != nil {
		return nil, err
	}
	raw, err := client.Call(ctx, methodMCPCatalog, nil)
	if err != nil {
		return nil, err
	}
	catalog, err := protocol.ParseMCPCatalogResult(raw)
	if err != nil {
		return nil, fmt.Errorf("MCP 目录无法解析: %w", err)
	}
	servers := make([]extensions.ToolServer, 0, len(catalog.Servers))
	for _, entry := range catalog.Servers {
		server := extensions.ToolServer{
			ID:        entry.ID,
			Name:      entry.ID,
			Source:    sourceFromWire(entry.Source),
			Transport: extensions.ToolTransport(entry.Transport),
			Enabled:   entry.Enabled,
		}
		if entry.Auth != nil {
			server.AuthStatus = entry.Auth.Status
			server.AuthMissingScopes = append([]string(nil), entry.Auth.MissingScopes...)
			server.Connected = entry.Auth.Status == "authenticated"
		}
		servers = append(servers, server)
	}
	return servers, nil
}

// ProbeToolServer 实现显式探测（`mcp.test`）：只读，不改配置。
//
// 回执形状未在 D1 冻结；这里按 `{ok|reachable, tools[], reason|error}` 宽容解析，
// 认不出的字段留空。
func (r *extensionReaders) ProbeToolServer(
	ctx context.Context,
	serverID string,
) (extensions.ToolServerProbe, error) {
	client, err := r.app.requireControl("探测工具服务器")
	if err != nil {
		return extensions.ToolServerProbe{}, err
	}
	raw, err := client.Call(ctx, methodMCPTest, map[string]any{"id": serverID, "source": "user"})
	if err != nil {
		return extensions.ToolServerProbe{}, err
	}
	var reply struct {
		ID        string                      `json:"id"`
		OK        *bool                       `json:"ok"`
		Reachable *bool                       `json:"reachable"`
		Tools     []extensions.ToolDescriptor `json:"tools"`
		Reason    *string                     `json:"reason"`
		Error     *string                     `json:"error"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return extensions.ToolServerProbe{}, fmt.Errorf("探测回执无法解析: %w", err)
	}
	reachable := false
	if reply.Reachable != nil {
		reachable = *reply.Reachable
	} else if reply.OK != nil {
		reachable = *reply.OK
	}
	reason := ""
	if reply.Reason != nil {
		reason = *reply.Reason
	} else if reply.Error != nil {
		reason = *reply.Error
	}
	return extensions.ToolServerProbe{
		ServerID:  firstNonEmpty(reply.ID, serverID),
		Reachable: reachable,
		Tools:     append([]extensions.ToolDescriptor(nil), reply.Tools...),
		Reason:    reason,
	}, nil
}

// PreviewToolServerImport 实现导入预览（`mcp.import.preview`）：只读。
func (r *extensionReaders) PreviewToolServerImport(
	ctx context.Context,
	source string,
) (extensions.ImportPreview, error) {
	client, err := r.app.requireControl("预览工具服务器导入")
	if err != nil {
		return extensions.ImportPreview{}, err
	}
	raw, err := client.Call(ctx, methodMCPImportPreview, protocol.MCPImportPreviewParams{Source: source})
	if err != nil {
		return extensions.ImportPreview{}, err
	}
	var reply struct {
		Source     string `json:"source"`
		Candidates []struct {
			Key       string          `json:"key"`
			ID        string          `json:"id"`
			Server    json.RawMessage `json:"server"`
			Name      string          `json:"name"`
			Transport string          `json:"transport"`
			Command   string          `json:"command"`
			Args      []string        `json:"args"`
			URL       string          `json:"url"`
			Existing  *bool           `json:"existing"`
			Warnings  []string        `json:"warnings"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return extensions.ImportPreview{}, fmt.Errorf("导入预览回执无法解析: %w", err)
	}
	preview := extensions.ImportPreview{
		Source:     firstNonEmpty(reply.Source, source),
		Candidates: make([]extensions.ImportCandidate, 0, len(reply.Candidates)),
	}
	for _, candidate := range reply.Candidates {
		spec := extensions.ToolServerSpec{
			Name:      firstNonEmpty(candidate.Name, candidate.ID),
			Transport: extensions.ToolTransport(candidate.Transport),
			Command:   candidate.Command,
			Args:      append([]string(nil), candidate.Args...),
			URL:       candidate.URL,
		}
		if len(candidate.Server) > 0 {
			var nested struct {
				Name      string   `json:"name"`
				Transport string   `json:"transport"`
				Command   string   `json:"command"`
				Args      []string `json:"args"`
				URL       string   `json:"url"`
			}
			if err := json.Unmarshal(candidate.Server, &nested); err == nil {
				if nested.Name != "" {
					spec.Name = nested.Name
				}
				if nested.Transport != "" {
					spec.Transport = extensions.ToolTransport(nested.Transport)
				}
				if nested.Command != "" {
					spec.Command = nested.Command
				}
				if len(nested.Args) > 0 {
					spec.Args = append([]string(nil), nested.Args...)
				}
				if nested.URL != "" {
					spec.URL = nested.URL
				}
			}
		}
		existing := false
		if candidate.Existing != nil {
			existing = *candidate.Existing
		}
		preview.Candidates = append(preview.Candidates, extensions.ImportCandidate{
			Key:      firstNonEmpty(candidate.Key, candidate.ID, spec.Name),
			Server:   spec,
			Existing: existing,
			Warnings: append([]string(nil), candidate.Warnings...),
		})
	}
	return preview, nil
}

// ReadScheduledTasks 实现 extensions.SchedulerReader（`schedule.list`）。
func (r *extensionReaders) ReadScheduledTasks(ctx context.Context) ([]extensions.ScheduledTask, error) {
	client, err := r.app.requireControl("读取计划任务清单")
	if err != nil {
		return nil, err
	}
	raw, err := client.Call(ctx, methodScheduleList, nil)
	if err != nil {
		return nil, err
	}
	result, err := protocol.ParseScheduleListResult(raw)
	if err != nil {
		return nil, fmt.Errorf("计划任务清单无法解析: %w", err)
	}
	tasks := make([]extensions.ScheduledTask, 0, len(result.Jobs))
	for _, job := range result.Jobs {
		tasks = append(tasks, extensions.ScheduledTask{
			ID:       job.ID,
			Name:     job.Name,
			Schedule: compactJSON(job.Schedule),
			Enabled:  job.Enabled,
			// 运行状态与下次/上次运行时间不在本轮冻结的字段里，留空。
			Revision: job.Revision,
			Prompt:   job.Prompt,
			Cwd:      job.Cwd,
			Mode:     job.Mode,
		})
	}
	return tasks, nil
}

// ReadWorkflows / ReadWorkflow / ReadWorkflowRuns：workflow.* 在 alpha.21 不注册，
// 按能力缺失处理，不发请求。
func (r *extensionReaders) ReadWorkflows(ctx context.Context) ([]extensions.WorkflowDefinition, error) {
	return nil, workflowCapabilityMissing()
}

func (r *extensionReaders) ReadWorkflow(
	ctx context.Context,
	workflowID string,
) (extensions.WorkflowDefinition, error) {
	return extensions.WorkflowDefinition{}, workflowCapabilityMissing()
}

func (r *extensionReaders) ReadWorkflowRuns(
	ctx context.Context,
	workflowID string,
) ([]extensions.WorkflowRun, error) {
	return nil, workflowCapabilityMissing()
}

func workflowCapabilityMissing() error {
	return &extensions.CapabilityError{
		Feature: "workflow.list",
		Err:     fmt.Errorf("服务端未注册 workflow.*（alpha.21 实测 unknown_method）"),
	}
}

// ReadWakeups 实现某会话或分支的唤醒读取（`session.wakeup.list`，会话方法）。
func (r *extensionReaders) ReadWakeups(
	ctx context.Context,
	scope extensions.WakeupScope,
) ([]extensions.Wakeup, error) {
	runtime, err := r.app.openRuntime(scope.SessionID, methodSessionWakeupList)
	if err != nil {
		return nil, err
	}
	raw, err := runtime.lease.Subscription().Call(ctx, methodSessionWakeupList, map[string]any{
		"sessionId": scope.SessionID,
		"branchId":  scope.BranchID,
	})
	if err != nil {
		return nil, err
	}
	result, err := protocol.ParseSessionWakeupListResult(raw)
	if err != nil {
		return nil, fmt.Errorf("唤醒清单无法解析: %w", err)
	}
	wakeups := make([]extensions.Wakeup, 0, len(result.Wakeups))
	for _, wakeup := range result.Wakeups {
		wakeups = append(wakeups, extensions.Wakeup{
			ID:        wakeup.WakeupID,
			SessionID: scope.SessionID,
			BranchID:  scope.BranchID,
			Prompt:    wakeup.Prompt,
			TriggerAt: parseWireTime(wakeup.At),
			Status:    extensions.WakeupStatus(wakeup.State),
		})
	}
	return wakeups, nil
}

// sourceFromWire 把服务端给出的来源映射成领域取值；无法识别的按只读处理。
func sourceFromWire(raw string) extensions.Source {
	switch extensions.Source(raw) {
	case extensions.SourceLocal, extensions.SourceGit, extensions.SourceBundled,
		extensions.SourceUser, extensions.SourceProject, extensions.SourcePlugin:
		return extensions.Source(raw)
	default:
		return extensions.SourceUnknown
	}
}

// parseWireTime 解析服务端给的 RFC3339 时间；解析失败按未给出处理。
func parseWireTime(raw string) time.Time {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

// compactJSON 压缩一段 JSON 文本，失败时原样返回。
func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, raw); err != nil {
		return string(raw)
	}
	return buffer.String()
}

// sortedStrings 返回排序后的副本，供稳定输出。
func sortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return copied
}

// 编译期确认：本类型同时满足扩展与后台领域需要的七个读取接口。
var (
	_ extensions.PluginReader     = (*extensionReaders)(nil)
	_ extensions.SkillReader      = (*extensionReaders)(nil)
	_ extensions.HookReader       = (*extensionReaders)(nil)
	_ extensions.ToolServerReader = (*extensionReaders)(nil)
	_ extensions.SchedulerReader  = (*extensionReaders)(nil)
	_ extensions.WorkflowReader   = (*extensionReaders)(nil)
	_ extensions.WakeupReader     = (*extensionReaders)(nil)
)
