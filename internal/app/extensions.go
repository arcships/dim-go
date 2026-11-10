package app

import (
	"context"
	"time"

	"github.com/arcships/dimagent-mygo/internal/extensions"
)

// extensionReaders 是扩展与后台领域的七个读取方。
//
// 七个接口都落在同一个类型上：它们共用一条控制连接，wire 形状只在集成层出现一次。
// 只有读取，不探测、不安装、不改配置；探测与导入预览由用户显式触发时才调用。
type extensionReaders struct {
	app *App
}

// wirePlugin 是插件清单里的条目。
type wirePlugin struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Source          string `json:"source"`
	SourceRef       string `json:"sourceRef"`
	Installed       bool   `json:"installed"`
	Enabled         bool   `json:"enabled"`
	Optional        bool   `json:"optional"`
	UpdateAvailable bool   `json:"updateAvailable"`
	LatestVersion   string `json:"latestVersion"`
	Description     string `json:"description"`
}

// ReadPlugins 实现 extensions.PluginReader。
func (r *extensionReaders) ReadPlugins(ctx context.Context) ([]extensions.Plugin, error) {
	client, err := r.app.requireControl("读取插件清单")
	if err != nil {
		return nil, err
	}
	var result struct {
		Plugins []wirePlugin `json:"plugins"`
	}
	if err := client.CallInto(ctx, "plugin.list", nil, &result); err != nil {
		return nil, err
	}
	plugins := make([]extensions.Plugin, 0, len(result.Plugins))
	for _, plugin := range result.Plugins {
		plugins = append(plugins, extensions.Plugin{
			ID:              plugin.ID,
			Name:            plugin.Name,
			Version:         plugin.Version,
			Source:          sourceFromWire(plugin.Source),
			SourceRef:       plugin.SourceRef,
			Installed:       plugin.Installed,
			Enabled:         plugin.Enabled,
			Optional:        plugin.Optional,
			UpdateAvailable: plugin.UpdateAvailable,
			LatestVersion:   plugin.LatestVersion,
			Description:     plugin.Description,
		})
	}
	return plugins, nil
}

// wireSkill 是技能清单里的条目。
type wireSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Source      string `json:"source"`
	Installed   bool   `json:"installed"`
	Enabled     bool   `json:"enabled"`
}

// ReadSkills 实现 extensions.SkillReader。
func (r *extensionReaders) ReadSkills(ctx context.Context) ([]extensions.Skill, error) {
	client, err := r.app.requireControl("读取技能清单")
	if err != nil {
		return nil, err
	}
	var result struct {
		Skills []wireSkill `json:"skills"`
	}
	if err := client.CallInto(ctx, "skill.list", nil, &result); err != nil {
		return nil, err
	}
	skills := make([]extensions.Skill, 0, len(result.Skills))
	for _, skill := range result.Skills {
		skills = append(skills, extensions.Skill{
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			Version:     skill.Version,
			Source:      sourceFromWire(skill.Source),
			Installed:   skill.Installed,
			Enabled:     skill.Enabled,
		})
	}
	return skills, nil
}

// wireHook 是钩子清单里的条目。
type wireHook struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Event   string `json:"event"`
	Command string `json:"command"`
	Source  string `json:"source"`
	Enabled bool   `json:"enabled"`
}

// ReadHooks 实现 extensions.HookReader。
func (r *extensionReaders) ReadHooks(ctx context.Context) ([]extensions.Hook, error) {
	client, err := r.app.requireControl("读取钩子清单")
	if err != nil {
		return nil, err
	}
	var result struct {
		Hooks []wireHook `json:"hooks"`
	}
	if err := client.CallInto(ctx, "hook.list", nil, &result); err != nil {
		return nil, err
	}
	hooks := make([]extensions.Hook, 0, len(result.Hooks))
	for _, hook := range result.Hooks {
		hooks = append(hooks, extensions.Hook{
			ID:      hook.ID,
			Name:    hook.Name,
			Event:   hook.Event,
			Command: hook.Command,
			Source:  sourceFromWire(hook.Source),
			Enabled: hook.Enabled,
		})
	}
	return hooks, nil
}

// wireToolServer 是工具服务器清单里的条目。
type wireToolServer struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	URL       string   `json:"url"`
	Enabled   bool     `json:"enabled"`
	Connected bool     `json:"connected"`
	EnvKeys   []string `json:"envKeys"`
	ToolCount int      `json:"toolCount"`
}

// ReadToolServers 实现 extensions.ToolServerReader。
func (r *extensionReaders) ReadToolServers(ctx context.Context) ([]extensions.ToolServer, error) {
	client, err := r.app.requireControl("读取工具服务器清单")
	if err != nil {
		return nil, err
	}
	var result struct {
		Servers []wireToolServer `json:"servers"`
	}
	if err := client.CallInto(ctx, "toolServer.list", nil, &result); err != nil {
		return nil, err
	}
	servers := make([]extensions.ToolServer, 0, len(result.Servers))
	for _, server := range result.Servers {
		servers = append(servers, extensions.ToolServer{
			ID:        server.ID,
			Name:      server.Name,
			Transport: extensions.ToolTransport(server.Transport),
			Command:   server.Command,
			Args:      append([]string(nil), server.Args...),
			URL:       server.URL,
			Enabled:   server.Enabled,
			Connected: server.Connected,
			EnvKeys:   append([]string(nil), server.EnvKeys...),
			ToolCount: server.ToolCount,
		})
	}
	return servers, nil
}

// ProbeToolServer 实现显式探测；只读，不改配置。
func (r *extensionReaders) ProbeToolServer(
	ctx context.Context,
	serverID string,
) (extensions.ToolServerProbe, error) {
	client, err := r.app.requireControl("探测工具服务器")
	if err != nil {
		return extensions.ToolServerProbe{}, err
	}
	var result struct {
		ServerID  string                      `json:"serverId"`
		Reachable bool                        `json:"reachable"`
		Tools     []extensions.ToolDescriptor `json:"tools"`
		Reason    string                      `json:"reason"`
	}
	if err := client.CallInto(ctx, "toolServer.probe", map[string]any{"serverId": serverID}, &result); err != nil {
		return extensions.ToolServerProbe{}, err
	}
	return extensions.ToolServerProbe{
		ServerID:  result.ServerID,
		Reachable: result.Reachable,
		Tools:     append([]extensions.ToolDescriptor(nil), result.Tools...),
		Reason:    result.Reason,
	}, nil
}

// PreviewToolServerImport 实现导入预览；只读，不写任何配置。
func (r *extensionReaders) PreviewToolServerImport(
	ctx context.Context,
	source string,
) (extensions.ImportPreview, error) {
	client, err := r.app.requireControl("预览工具服务器导入")
	if err != nil {
		return extensions.ImportPreview{}, err
	}
	var result struct {
		Source     string `json:"source"`
		Candidates []struct {
			Key      string         `json:"key"`
			Server   wireToolServer `json:"server"`
			Existing bool           `json:"existing"`
			Warnings []string       `json:"warnings"`
		} `json:"candidates"`
	}
	if err := client.CallInto(ctx, "toolServer.import.preview", map[string]any{"source": source}, &result); err != nil {
		return extensions.ImportPreview{}, err
	}
	preview := extensions.ImportPreview{
		Source:     result.Source,
		Candidates: make([]extensions.ImportCandidate, 0, len(result.Candidates)),
	}
	for _, candidate := range result.Candidates {
		preview.Candidates = append(preview.Candidates, extensions.ImportCandidate{
			Key: candidate.Key,
			Server: extensions.ToolServerSpec{
				Name:      candidate.Server.Name,
				Transport: extensions.ToolTransport(candidate.Server.Transport),
				Command:   candidate.Server.Command,
				Args:      append([]string(nil), candidate.Server.Args...),
				URL:       candidate.Server.URL,
			},
			Existing: candidate.Existing,
			Warnings: append([]string(nil), candidate.Warnings...),
		})
	}
	return preview, nil
}

// wireScheduledTask 是计划任务清单里的条目。
type wireScheduledTask struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Schedule   string `json:"schedule"`
	Enabled    bool   `json:"enabled"`
	Running    bool   `json:"running"`
	NextRunAt  int64  `json:"nextRunAt"`
	LastRunAt  int64  `json:"lastRunAt"`
	LastStatus string `json:"lastStatus"`
}

// ReadScheduledTasks 实现 extensions.SchedulerReader。
func (r *extensionReaders) ReadScheduledTasks(ctx context.Context) ([]extensions.ScheduledTask, error) {
	client, err := r.app.requireControl("读取计划任务清单")
	if err != nil {
		return nil, err
	}
	var result struct {
		Tasks []wireScheduledTask `json:"tasks"`
	}
	if err := client.CallInto(ctx, "scheduler.list", nil, &result); err != nil {
		return nil, err
	}
	tasks := make([]extensions.ScheduledTask, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		tasks = append(tasks, extensions.ScheduledTask{
			ID:         task.ID,
			Name:       task.Name,
			Schedule:   task.Schedule,
			Enabled:    task.Enabled,
			Running:    task.Running,
			NextRunAt:  fromMillis(task.NextRunAt),
			LastRunAt:  fromMillis(task.LastRunAt),
			LastStatus: task.LastStatus,
		})
	}
	return tasks, nil
}

// wireWorkflow 是工作流定义。
type wireWorkflow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Source      string `json:"source"`
	Steps       []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"steps"`
}

// ReadWorkflows 实现 extensions.WorkflowReader。
func (r *extensionReaders) ReadWorkflows(ctx context.Context) ([]extensions.WorkflowDefinition, error) {
	client, err := r.app.requireControl("读取工作流清单")
	if err != nil {
		return nil, err
	}
	var result struct {
		Workflows []wireWorkflow `json:"workflows"`
	}
	if err := client.CallInto(ctx, "workflow.list", nil, &result); err != nil {
		return nil, err
	}
	workflows := make([]extensions.WorkflowDefinition, 0, len(result.Workflows))
	for _, workflow := range result.Workflows {
		workflows = append(workflows, mapWireWorkflow(workflow))
	}
	return workflows, nil
}

// ReadWorkflow 实现单份工作流定义的读取。
func (r *extensionReaders) ReadWorkflow(
	ctx context.Context,
	workflowID string,
) (extensions.WorkflowDefinition, error) {
	client, err := r.app.requireControl("读取工作流定义")
	if err != nil {
		return extensions.WorkflowDefinition{}, err
	}
	var result wireWorkflow
	if err := client.CallInto(ctx, "workflow.get", map[string]any{"workflowId": workflowID}, &result); err != nil {
		return extensions.WorkflowDefinition{}, err
	}
	return mapWireWorkflow(result), nil
}

// ReadWorkflowRuns 实现工作流运行读数。
func (r *extensionReaders) ReadWorkflowRuns(
	ctx context.Context,
	workflowID string,
) ([]extensions.WorkflowRun, error) {
	client, err := r.app.requireControl("读取工作流运行")
	if err != nil {
		return nil, err
	}
	var result struct {
		Runs []struct {
			ID         string `json:"id"`
			WorkflowID string `json:"workflowId"`
			Status     string `json:"status"`
			StartedAt  int64  `json:"startedAt"`
			EndedAt    int64  `json:"endedAt"`
			Error      string `json:"error"`
		} `json:"runs"`
	}
	if err := client.CallInto(ctx, "workflow.runs", map[string]any{"workflowId": workflowID}, &result); err != nil {
		return nil, err
	}
	runs := make([]extensions.WorkflowRun, 0, len(result.Runs))
	for _, run := range result.Runs {
		runs = append(runs, extensions.WorkflowRun{
			ID:         run.ID,
			WorkflowID: run.WorkflowID,
			Status:     extensions.WorkflowStatus(run.Status),
			StartedAt:  fromMillis(run.StartedAt),
			EndedAt:    fromMillis(run.EndedAt),
			Error:      run.Error,
		})
	}
	return runs, nil
}

// wireWakeup 是唤醒记录。
type wireWakeup struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	Title     string `json:"title"`
	Prompt    string `json:"prompt"`
	TriggerAt int64  `json:"triggerAt"`
	Status    string `json:"status"`
}

// ReadWakeups 实现某个会话或分支的唤醒记录读取。
func (r *extensionReaders) ReadWakeups(
	ctx context.Context,
	scope extensions.WakeupScope,
) ([]extensions.Wakeup, error) {
	client, err := r.app.requireControl("读取唤醒记录")
	if err != nil {
		return nil, err
	}
	params := map[string]any{"sessionId": scope.SessionID}
	if scope.BranchID != "" {
		params["branchId"] = scope.BranchID
	}
	var result struct {
		Wakeups []wireWakeup `json:"wakeups"`
	}
	if err := client.CallInto(ctx, "wakeup.list", params, &result); err != nil {
		return nil, err
	}
	wakeups := make([]extensions.Wakeup, 0, len(result.Wakeups))
	for _, wakeup := range result.Wakeups {
		wakeups = append(wakeups, extensions.Wakeup{
			ID:        wakeup.ID,
			SessionID: wakeup.SessionID,
			BranchID:  wakeup.BranchID,
			Title:     wakeup.Title,
			Prompt:    wakeup.Prompt,
			TriggerAt: fromMillis(wakeup.TriggerAt),
			Status:    extensions.WakeupStatus(wakeup.Status),
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

func mapWireWorkflow(workflow wireWorkflow) extensions.WorkflowDefinition {
	definition := extensions.WorkflowDefinition{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Version:     workflow.Version,
		Source:      sourceFromWire(workflow.Source),
	}
	for _, step := range workflow.Steps {
		definition.Steps = append(definition.Steps, extensions.WorkflowStep{
			ID:   step.ID,
			Name: step.Name,
			Kind: step.Kind,
		})
	}
	return definition
}

// fromMillis 把服务端的毫秒时间戳映射成 UTC 时间；0 表示服务端没有给出。
func fromMillis(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.UnixMilli(value).UTC()
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
