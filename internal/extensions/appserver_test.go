package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// noVerifyInspector 表示本测试不进入未知核对；能力缺失与明确回执都不需要它。
type noVerifyInspector struct{}

func (noVerifyInspector) LookupInput(context.Context, write.InputQuery) (write.InputState, error) {
	return write.InputState{}, nil
}

// appServerHarness 是「连接 + 写登记」的集成形状：管理面读取与写入都走真实
// NDJSON，wire 方法名与回执形状只出现在这里，本包不复制它们。
//
// 写成功后，harness 在自己的测试状态上模拟服务端事实变化，让后续读取能看到
// 安装、启停、删除与取消的结果；这些状态只活在本测试进程里。
type appServerHarness struct {
	client *protocol.Client

	mu              sync.Mutex
	pluginInstalls  []Plugin
	pluginEnabled   map[string]bool
	pluginUpdated   map[string]bool
	skillInstalls   []Skill
	skillEnabled    map[string]bool
	hookWrites      map[string]Hook
	hookOrder       []string
	toolAdded       []ToolServer
	toolUpdated     map[string]ToolServerSpec
	toolRemoved     map[string]bool
	wakeupCancelled map[string]bool
	addedCount      int
}

func newAppServerHarness(client *protocol.Client) *appServerHarness {
	return &appServerHarness{
		client:          client,
		pluginEnabled:   make(map[string]bool),
		pluginUpdated:   make(map[string]bool),
		skillEnabled:    make(map[string]bool),
		hookWrites:      make(map[string]Hook),
		toolUpdated:     make(map[string]ToolServerSpec),
		toolRemoved:     make(map[string]bool),
		wakeupCancelled: make(map[string]bool),
	}
}

// 以下六个读取实现是集成层在 wire 与本领域类型之间的映射。

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

func (h *appServerHarness) ReadPlugins(ctx context.Context) ([]Plugin, error) {
	var result struct {
		Plugins []wirePlugin `json:"plugins"`
	}
	if err := h.client.CallInto(ctx, "plugin.list", nil, &result); err != nil {
		return nil, err
	}
	plugins := make([]Plugin, 0, len(result.Plugins))
	for _, plugin := range result.Plugins {
		plugins = append(plugins, Plugin{
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
	h.mu.Lock()
	defer h.mu.Unlock()
	for index := range plugins {
		if enabled, ok := h.pluginEnabled[plugins[index].ID]; ok {
			plugins[index].Enabled = enabled
		}
		if h.pluginUpdated[plugins[index].ID] {
			plugins[index].UpdateAvailable = false
		}
	}
	for _, installed := range h.pluginInstalls {
		found := false
		for _, plugin := range plugins {
			if plugin.ID == installed.ID {
				found = true
				break
			}
		}
		if !found {
			plugins = append(plugins, installed)
		}
	}
	return plugins, nil
}

type wireSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Source      string `json:"source"`
	Installed   bool   `json:"installed"`
	Enabled     bool   `json:"enabled"`
}

func (h *appServerHarness) ReadSkills(ctx context.Context) ([]Skill, error) {
	var result struct {
		Skills []wireSkill `json:"skills"`
	}
	if err := h.client.CallInto(ctx, "skill.list", nil, &result); err != nil {
		return nil, err
	}
	skills := make([]Skill, 0, len(result.Skills))
	for _, skill := range result.Skills {
		skills = append(skills, Skill{
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			Version:     skill.Version,
			Source:      sourceFromWire(skill.Source),
			Installed:   skill.Installed,
			Enabled:     skill.Enabled,
		})
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for index := range skills {
		if enabled, ok := h.skillEnabled[skills[index].ID]; ok {
			skills[index].Enabled = enabled
		}
	}
	for _, installed := range h.skillInstalls {
		found := false
		for _, skill := range skills {
			if skill.ID == installed.ID {
				found = true
				break
			}
		}
		if !found {
			skills = append(skills, installed)
		}
	}
	return skills, nil
}

type wireHook struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Event   string `json:"event"`
	Command string `json:"command"`
	Source  string `json:"source"`
	Enabled bool   `json:"enabled"`
}

func (h *appServerHarness) ReadHooks(ctx context.Context) ([]Hook, error) {
	var result struct {
		Hooks []wireHook `json:"hooks"`
	}
	if err := h.client.CallInto(ctx, "hook.list", nil, &result); err != nil {
		return nil, err
	}
	hooks := make([]Hook, 0, len(result.Hooks))
	for _, hook := range result.Hooks {
		hooks = append(hooks, Hook{
			ID:      hook.ID,
			Name:    hook.Name,
			Event:   hook.Event,
			Command: hook.Command,
			Source:  sourceFromWire(hook.Source),
			Enabled: hook.Enabled,
		})
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, id := range h.hookOrder {
		written := h.hookWrites[id]
		replaced := false
		for index := range hooks {
			if hooks[index].ID == id {
				hooks[index] = written
				replaced = true
				break
			}
		}
		if !replaced {
			hooks = append(hooks, written)
		}
	}
	return hooks, nil
}

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

func (h *appServerHarness) ReadToolServers(ctx context.Context) ([]ToolServer, error) {
	var result struct {
		Servers []wireToolServer `json:"servers"`
	}
	if err := h.client.CallInto(ctx, "toolServer.list", nil, &result); err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	servers := make([]ToolServer, 0, len(result.Servers)+len(h.toolAdded))
	for _, server := range result.Servers {
		if h.toolRemoved[server.ID] {
			continue
		}
		mapped := ToolServer{
			ID:        server.ID,
			Name:      server.Name,
			Transport: ToolTransport(server.Transport),
			Command:   server.Command,
			Args:      append([]string(nil), server.Args...),
			URL:       server.URL,
			Enabled:   server.Enabled,
			Connected: server.Connected,
			EnvKeys:   append([]string(nil), server.EnvKeys...),
			ToolCount: server.ToolCount,
		}
		if spec, ok := h.toolUpdated[server.ID]; ok {
			mapped.Name = spec.Name
			mapped.Transport = spec.Transport
			mapped.Command = spec.Command
			mapped.Args = append([]string(nil), spec.Args...)
			mapped.URL = spec.URL
			mapped.EnvKeys = sortedEnvKeys(spec.Env)
		}
		servers = append(servers, mapped)
	}
	servers = append(servers, h.toolAdded...)
	return servers, nil
}

func (h *appServerHarness) ProbeToolServer(ctx context.Context, serverID string) (ToolServerProbe, error) {
	var result struct {
		ServerID  string           `json:"serverId"`
		Reachable bool             `json:"reachable"`
		Tools     []ToolDescriptor `json:"tools"`
		Reason    string           `json:"reason"`
	}
	if err := h.client.CallInto(ctx, "toolServer.probe", map[string]any{"serverId": serverID}, &result); err != nil {
		return ToolServerProbe{}, err
	}
	return ToolServerProbe{
		ServerID:  result.ServerID,
		Reachable: result.Reachable,
		Tools:     append([]ToolDescriptor(nil), result.Tools...),
		Reason:    result.Reason,
	}, nil
}

func (h *appServerHarness) PreviewToolServerImport(ctx context.Context, source string) (ImportPreview, error) {
	var result struct {
		Source     string `json:"source"`
		Candidates []struct {
			Key      string         `json:"key"`
			Server   wireToolServer `json:"server"`
			Existing bool           `json:"existing"`
			Warnings []string       `json:"warnings"`
		} `json:"candidates"`
	}
	if err := h.client.CallInto(ctx, "toolServer.import.preview", map[string]any{"source": source}, &result); err != nil {
		return ImportPreview{}, err
	}
	preview := ImportPreview{Source: result.Source, Candidates: make([]ImportCandidate, 0, len(result.Candidates))}
	for _, candidate := range result.Candidates {
		preview.Candidates = append(preview.Candidates, ImportCandidate{
			Key: candidate.Key,
			Server: ToolServerSpec{
				Name:      candidate.Server.Name,
				Transport: ToolTransport(candidate.Server.Transport),
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

func (h *appServerHarness) ReadScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	var result struct {
		Tasks []wireScheduledTask `json:"tasks"`
	}
	if err := h.client.CallInto(ctx, "scheduler.list", nil, &result); err != nil {
		return nil, err
	}
	tasks := make([]ScheduledTask, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		tasks = append(tasks, ScheduledTask{
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

func (h *appServerHarness) ReadWorkflows(ctx context.Context) ([]WorkflowDefinition, error) {
	var result struct {
		Workflows []wireWorkflow `json:"workflows"`
	}
	if err := h.client.CallInto(ctx, "workflow.list", nil, &result); err != nil {
		return nil, err
	}
	workflows := make([]WorkflowDefinition, 0, len(result.Workflows))
	for _, workflow := range result.Workflows {
		workflows = append(workflows, mapWireWorkflow(workflow))
	}
	return workflows, nil
}

func (h *appServerHarness) ReadWorkflow(ctx context.Context, workflowID string) (WorkflowDefinition, error) {
	var result wireWorkflow
	if err := h.client.CallInto(ctx, "workflow.get", map[string]any{"workflowId": workflowID}, &result); err != nil {
		return WorkflowDefinition{}, err
	}
	return mapWireWorkflow(result), nil
}

func (h *appServerHarness) ReadWorkflowRuns(ctx context.Context, workflowID string) ([]WorkflowRun, error) {
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
	if err := h.client.CallInto(ctx, "workflow.runs", map[string]any{"workflowId": workflowID}, &result); err != nil {
		return nil, err
	}
	runs := make([]WorkflowRun, 0, len(result.Runs))
	for _, run := range result.Runs {
		runs = append(runs, WorkflowRun{
			ID:         run.ID,
			WorkflowID: run.WorkflowID,
			Status:     WorkflowStatus(run.Status),
			StartedAt:  fromMillis(run.StartedAt),
			EndedAt:    fromMillis(run.EndedAt),
			Error:      run.Error,
		})
	}
	return runs, nil
}

type wireWakeup struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	Title     string `json:"title"`
	Prompt    string `json:"prompt"`
	TriggerAt int64  `json:"triggerAt"`
	Status    string `json:"status"`
}

func (h *appServerHarness) ReadWakeups(ctx context.Context, scope WakeupScope) ([]Wakeup, error) {
	params := map[string]any{"sessionId": scope.SessionID}
	if scope.BranchID != "" {
		params["branchId"] = scope.BranchID
	}
	var result struct {
		Wakeups []wireWakeup `json:"wakeups"`
	}
	if err := h.client.CallInto(ctx, "wakeup.list", params, &result); err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	wakeups := make([]Wakeup, 0, len(result.Wakeups))
	for _, wakeup := range result.Wakeups {
		if h.wakeupCancelled[wakeup.ID] {
			continue
		}
		wakeups = append(wakeups, Wakeup{
			ID:        wakeup.ID,
			SessionID: wakeup.SessionID,
			BranchID:  wakeup.BranchID,
			Title:     wakeup.Title,
			Prompt:    wakeup.Prompt,
			TriggerAt: fromMillis(wakeup.TriggerAt),
			Status:    WakeupStatus(wakeup.Status),
		})
	}
	return wakeups, nil
}

// Send 是写登记的发送方：按管理写入操作名映射到 wire 方法，只调用一次。
func (h *appServerHarness) Send(ctx context.Context, req write.Request) (write.Reply, error) {
	if req.Kind != write.KindManagementWrite {
		return write.Reply{}, fmt.Errorf("未支持的写意图 %s", req.Kind)
	}
	managed, err := DecodeManagementWrite(req.Params)
	if err != nil {
		return write.Reply{}, err
	}
	switch managed.Op {
	case OpPluginInstall:
		var payload PluginInstallPayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "plugin.install", map[string]any{
			"source": map[string]any{"kind": string(payload.Source.Kind), "ref": payload.Source.Ref},
		})
		if err != nil {
			return write.Reply{}, err
		}
		var result struct {
			PluginID string `json:"pluginId"`
		}
		_ = json.Unmarshal(raw, &result)
		if result.PluginID == "" {
			result.PluginID = payload.Source.Ref
		}
		h.mu.Lock()
		h.pluginInstalls = append(h.pluginInstalls, Plugin{
			ID:        result.PluginID,
			Name:      result.PluginID,
			Source:    payload.Source.Kind,
			SourceRef: payload.Source.Ref,
			Installed: true,
		})
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpPluginSetEnabled:
		var payload PluginEnabledPayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "plugin.set_enabled", map[string]any{
			"pluginId": payload.PluginID,
			"enabled":  payload.Enabled,
		})
		if err != nil {
			return write.Reply{}, err
		}
		h.mu.Lock()
		h.pluginEnabled[payload.PluginID] = payload.Enabled
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpPluginUpdate:
		var payload PluginUpdatePayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "plugin.update", map[string]any{"pluginId": payload.PluginID})
		if err != nil {
			return write.Reply{}, err
		}
		h.mu.Lock()
		h.pluginUpdated[payload.PluginID] = true
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpSkillInstall:
		var payload SkillInstallPayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "skill.install", map[string]any{
			"source": map[string]any{"kind": string(payload.Source.Kind), "ref": payload.Source.Ref},
		})
		if err != nil {
			return write.Reply{}, err
		}
		var result struct {
			SkillID string `json:"skillId"`
		}
		_ = json.Unmarshal(raw, &result)
		if result.SkillID == "" {
			result.SkillID = payload.Source.Ref
		}
		h.mu.Lock()
		h.skillInstalls = append(h.skillInstalls, Skill{
			ID:        result.SkillID,
			Name:      result.SkillID,
			Source:    payload.Source.Kind,
			Installed: true,
		})
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpSkillSetEnabled:
		var payload SkillEnabledPayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "skill.set_enabled", map[string]any{
			"skillId": payload.SkillID,
			"enabled": payload.Enabled,
		})
		if err != nil {
			return write.Reply{}, err
		}
		h.mu.Lock()
		h.skillEnabled[payload.SkillID] = payload.Enabled
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpHookWrite:
		var payload HookWritePayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "hook.write", map[string]any{"hook": map[string]any{
			"id":      payload.Hook.ID,
			"name":    payload.Hook.Name,
			"event":   payload.Hook.Event,
			"command": payload.Hook.Command,
			"source":  string(payload.Hook.Source),
			"enabled": payload.Hook.Enabled,
		}})
		if err != nil {
			return write.Reply{}, err
		}
		h.mu.Lock()
		if _, exists := h.hookWrites[payload.Hook.ID]; !exists && payload.Hook.ID != "" {
			h.hookOrder = append(h.hookOrder, payload.Hook.ID)
		}
		h.hookWrites[payload.Hook.ID] = payload.Hook
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpToolServerAdd:
		var payload ToolServerAddPayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		params := toolServerParams(payload.Server)
		if payload.ImportSource != "" {
			params["importSource"] = payload.ImportSource
			params["candidateKey"] = payload.CandidateKey
		}
		raw, err := h.client.Call(ctx, "toolServer.add", params)
		if err != nil {
			return write.Reply{}, err
		}
		h.mu.Lock()
		h.addedCount++
		h.toolAdded = append(h.toolAdded, ToolServer{
			ID:        fmt.Sprintf("srv-added-%d", h.addedCount),
			Name:      payload.Server.Name,
			Transport: payload.Server.Transport,
			Command:   payload.Server.Command,
			Args:      append([]string(nil), payload.Server.Args...),
			URL:       payload.Server.URL,
			EnvKeys:   sortedEnvKeys(payload.Server.Env),
		})
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpToolServerUpdate:
		var payload ToolServerUpdatePayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		params := toolServerParams(payload.Server)
		params["serverId"] = payload.ServerID
		raw, err := h.client.Call(ctx, "toolServer.update", params)
		if err != nil {
			return write.Reply{}, err
		}
		h.mu.Lock()
		h.toolUpdated[payload.ServerID] = payload.Server
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpToolServerRemove:
		var payload ToolServerRemovePayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "toolServer.remove", map[string]any{"serverId": payload.ServerID})
		if err != nil {
			return write.Reply{}, err
		}
		h.mu.Lock()
		h.toolRemoved[payload.ServerID] = true
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpWorkflowStart:
		var payload WorkflowStartPayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "workflow.start", map[string]any{
			"workflowId": payload.WorkflowID,
			"inputs":     payload.Inputs,
		})
		if err != nil {
			return write.Reply{}, err
		}
		return write.Reply{Result: raw}, nil
	case OpWorkflowStop:
		var payload WorkflowStopPayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "workflow.stop", map[string]any{"runId": payload.RunID})
		if err != nil {
			return write.Reply{}, err
		}
		return write.Reply{Result: raw}, nil
	case OpWakeupCancel:
		var payload WakeupCancelPayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "wakeup.cancel", map[string]any{"wakeupId": payload.WakeupID})
		if err != nil {
			return write.Reply{}, err
		}
		h.mu.Lock()
		h.wakeupCancelled[payload.WakeupID] = true
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	default:
		return write.Reply{}, fmt.Errorf("未支持的管理写入 %q", managed.Op)
	}
}

func toolServerParams(spec ToolServerSpec) map[string]any {
	params := map[string]any{
		"name":      spec.Name,
		"transport": string(spec.Transport),
		"command":   spec.Command,
		"url":       spec.URL,
	}
	if len(spec.Args) > 0 {
		params["args"] = append([]string(nil), spec.Args...)
	}
	if len(spec.Env) > 0 {
		env := make(map[string]string, len(spec.Env))
		for key, value := range spec.Env {
			env[key] = value
		}
		params["env"] = env
	}
	return params
}

func mapWireWorkflow(workflow wireWorkflow) WorkflowDefinition {
	definition := WorkflowDefinition{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Version:     workflow.Version,
		Source:      sourceFromWire(workflow.Source),
	}
	for _, step := range workflow.Steps {
		definition.Steps = append(definition.Steps, WorkflowStep{ID: step.ID, Name: step.Name, Kind: step.Kind})
	}
	return definition
}

func sourceFromWire(raw string) Source {
	switch Source(raw) {
	case SourceLocal, SourceGit, SourceBundled, SourceUser, SourceProject, SourcePlugin:
		return Source(raw)
	default:
		return SourceUnknown
	}
}

// sortedEnvKeys 返回稳定的环境变量键名，供测试适配器展示；值不参与读取。
func sortedEnvKeys(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func fromMillis(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.UnixMilli(value).UTC()
}

// startFakeServer 在临时数据目录里启动假服务端并连上控制连接。
func startFakeServer(t *testing.T, script fakeappserver.Script) (*fakeappserver.Server, *protocol.Client) {
	t.Helper()
	server, err := fakeappserver.Start(script)
	if err != nil {
		t.Fatalf("启动假服务端: %v", err)
	}
	supervisor, err := daemon.New(daemon.Config{
		DataDir:        server.DataDir(),
		RequestTimeout: 2 * time.Second,
	})
	if err != nil {
		_ = server.Close()
		t.Fatalf("创建连接监督者: %v", err)
	}
	conn, err := supervisor.Connect(context.Background())
	if err != nil {
		supervisor.Close(nil)
		_ = server.Close()
		t.Fatalf("连接假服务端: %v", err)
	}
	t.Cleanup(func() {
		supervisor.Close(nil)
		_ = server.Close()
	})
	return server, conn.Client
}

func newWriteManager(t *testing.T, harness *appServerHarness) *write.Manager {
	t.Helper()
	manager, err := write.New(write.Config{
		Sender:      harness,
		Inspector:   noVerifyInspector{},
		SendTimeout: 2 * time.Second,
		Verify:      write.VerifyPolicy{MaxAttempts: 1, Interval: time.Millisecond, Timeout: time.Second},
	})
	if err != nil {
		t.Fatalf("创建写登记: %v", err)
	}
	t.Cleanup(manager.Close)
	return manager
}

func newService(t *testing.T, cfg Config) *Service {
	t.Helper()
	service, err := New(cfg)
	if err != nil {
		t.Fatalf("创建扩展与后台领域: %v", err)
	}
	t.Cleanup(service.Close)
	return service
}

// attachNotices 把假服务端通知映射成本领域的变更通知，并返回一条已入队信号，
// 避免测试在通知到达之前就断言重读次数。
func attachNotices(t *testing.T, client *protocol.Client, service *Service) <-chan struct{} {
	t.Helper()
	arrived := make(chan struct{}, 8)
	client.OnNotification(func(message protocol.Message) {
		switch message.Method {
		case "scheduler.changed":
			_ = service.Notify(Notice{Topic: TopicScheduler})
		case "wakeup.changed":
			var params struct {
				SessionID string `json:"sessionId"`
				BranchID  string `json:"branchId"`
			}
			if err := json.Unmarshal(message.Params, &params); err != nil {
				return
			}
			_ = service.Notify(Notice{Topic: TopicWakeups, SessionID: params.SessionID, BranchID: params.BranchID})
		case "workflow.changed":
			var params struct {
				WorkflowID string `json:"workflowId"`
			}
			if err := json.Unmarshal(message.Params, &params); err != nil {
				return
			}
			_ = service.Notify(Notice{Topic: TopicWorkflows, WorkflowID: params.WorkflowID})
		}
		arrived <- struct{}{}
	})
	return arrived
}

func methodParams(t *testing.T, server *fakeappserver.Server, method string) []map[string]any {
	t.Helper()
	requests := server.RequestsFor(method)
	params := make([]map[string]any, 0, len(requests))
	for _, request := range requests {
		var decoded map[string]any
		if err := json.Unmarshal(request.Params, &decoded); err != nil {
			t.Fatalf("解析 %s 参数: %v", method, err)
		}
		params = append(params, decoded)
	}
	return params
}

func TestAppServerPluginsAndSkills(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "plugin.list", Result: map[string]any{"plugins": []any{
				map[string]any{"id": "installed", "name": "已装插件", "version": "1.0.0", "source": "local", "installed": true, "enabled": false},
				map[string]any{"id": "optional", "name": "可选插件", "version": "0.9.0", "source": "bundled", "installed": false, "optional": true},
			}}},
			{Method: "plugin.install", Result: map[string]any{"pluginId": "demo"}},
			{Method: "plugin.set_enabled", Result: map[string]any{"pluginId": "demo", "enabled": true}},
			{Method: "plugin.update", Result: map[string]any{"updated": true}},
			{Method: "skill.list", Result: map[string]any{"skills": []any{
				map[string]any{"id": "skill-1", "name": "提交信息", "source": "bundled", "installed": true, "enabled": false},
			}}},
			{Method: "skill.install", Result: map[string]any{"skillId": "skill-new"}},
			{Method: "skill.set_enabled", Result: map[string]any{"skillId": "skill-1", "enabled": true}},
		},
	})
	harness := newAppServerHarness(client)
	service := newService(t, Config{
		Plugins: harness,
		Skills:  harness,
		Writer:  newWriteManager(t, harness),
	})
	ctx := context.Background()

	plugins, err := service.ListPlugins(ctx)
	if err != nil {
		t.Fatalf("读取插件: %v", err)
	}
	if len(plugins.Items) != 2 {
		t.Fatalf("插件条数不符: %+v", plugins.Items)
	}
	if plugins.Items[1].Installed || !plugins.Items[1].Optional {
		t.Fatalf("可选插件没有显示为未安装: %+v", plugins.Items[1])
	}

	installed, err := service.InstallPlugin(ctx, InstallSource{Kind: SourceLocal, Ref: "/tmp/plugins/demo"})
	if err != nil || installed.Status != write.StatusSucceeded {
		t.Fatalf("安装插件: %v %+v", err, installed)
	}
	if installed.Kind != write.KindManagementWrite {
		t.Fatalf("安装插件没有走管理写入: %+v", installed)
	}
	installParams := methodParams(t, server, "plugin.install")
	if len(installParams) != 1 {
		t.Fatalf("安装请求次数不符: %d", len(installParams))
	}
	source, _ := installParams[0]["source"].(map[string]any)
	if source["kind"] != "local" || source["ref"] != "/tmp/plugins/demo" {
		t.Fatalf("安装来源没有透传: %+v", installParams[0])
	}

	plugins, err = service.ListPlugins(ctx)
	if err != nil {
		t.Fatalf("安装后读取插件: %v", err)
	}
	if !hasPlugin(plugins.Items, "demo") {
		t.Fatalf("安装后清单没有插件标识: %+v", plugins.Items)
	}

	enabled, err := service.SetPluginEnabled(ctx, "demo", true)
	if err != nil || enabled.Status != write.StatusSucceeded {
		t.Fatalf("启用插件: %v %+v", err, enabled)
	}
	if params := methodParams(t, server, "plugin.set_enabled"); len(params) != 1 ||
		params[0]["pluginId"] != "demo" || params[0]["enabled"] != true {
		t.Fatalf("启用参数不符: %+v", params)
	}

	updated, err := service.UpdatePlugin(ctx, "demo")
	if err != nil || updated.Status != write.StatusSucceeded {
		t.Fatalf("更新插件: %v %+v", err, updated)
	}
	if params := methodParams(t, server, "plugin.update"); len(params) != 1 || params[0]["pluginId"] != "demo" {
		t.Fatalf("更新参数不符: %+v", params)
	}

	skills, err := service.ListSkills(ctx)
	if err != nil || len(skills.Items) != 1 {
		t.Fatalf("读取技能: %v %+v", err, skills)
	}
	if _, err := service.SetSkillEnabled(ctx, "skill-1", true); err != nil {
		t.Fatalf("启用技能: %v", err)
	}
	if params := methodParams(t, server, "skill.set_enabled"); len(params) != 1 ||
		params[0]["skillId"] != "skill-1" || params[0]["enabled"] != true {
		t.Fatalf("技能启用参数不符: %+v", params)
	}
	if _, err := service.InstallSkill(ctx, InstallSource{Kind: SourceGit, Ref: "https://example.com/skill.git"}); err != nil {
		t.Fatalf("安装技能: %v", err)
	}

	// 模拟客户端重启：新实例不依赖任何本地记录，重新读取仍是服务端保存的状态。
	restarted := newService(t, Config{Skills: harness})
	skills, err = restarted.ListSkills(ctx)
	if err != nil {
		t.Fatalf("重启后读取技能: %v", err)
	}
	if !hasSkillEnabled(skills.Items, "skill-1") {
		t.Fatalf("重启后技能状态没有保持: %+v", skills.Items)
	}
	if !hasSkill(skills.Items, "skill-new") {
		t.Fatalf("重启后安装的技能没有出现: %+v", skills.Items)
	}
}

func TestAppServerHookSourceScope(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "hook.list", Result: map[string]any{"hooks": []any{
				map[string]any{"id": "user-1", "name": "用户钩子", "event": "pre", "command": "echo user", "source": "user", "enabled": true},
				map[string]any{"id": "project-1", "name": "项目钩子", "event": "pre", "command": "echo project", "source": "project", "enabled": true},
				map[string]any{"id": "plugin-1", "name": "插件钩子", "event": "post", "command": "echo plugin", "source": "plugin", "enabled": true},
			}}},
			{Method: "hook.write", Result: nil},
		},
	})
	harness := newAppServerHarness(client)
	manager := newWriteManager(t, harness)
	service := newService(t, Config{Hooks: harness, Writer: manager})

	hooks, err := service.ListHooks(context.Background())
	if err != nil || len(hooks.Items) != 3 {
		t.Fatalf("读取钩子: %v %+v", err, hooks)
	}
	writable := 0
	for _, hook := range hooks.Items {
		if hook.Writable() {
			writable++
			if hook.Source != SourceUser {
				t.Fatalf("非用户来源被标成可写: %+v", hook)
			}
		}
	}
	if writable != 1 {
		t.Fatalf("可写钩子数量不符: %d", writable)
	}

	for _, source := range []Source{SourceProject, SourcePlugin, SourceUnknown} {
		if _, err := service.WriteHook(context.Background(), Hook{Event: "pre", Command: "echo", Source: source}); !errors.Is(err, ErrSourceNotWritable) {
			t.Fatalf("来源 %s 没有拒绝写入: %v", source, err)
		}
	}
	if requests := server.RequestsFor("hook.write"); len(requests) != 0 {
		t.Fatalf("只读来源发出了写请求: %d", len(requests))
	}
	if table := manager.Table(); len(table) != 0 {
		t.Fatalf("只读来源登记了写意图: %+v", table)
	}

	view, err := service.WriteHook(context.Background(), Hook{
		ID: "user-2", Name: "新钩子", Event: "pre", Command: "echo new", Source: SourceUser, Enabled: true,
	})
	if err != nil || view.Status != write.StatusSucceeded {
		t.Fatalf("用户来源钩子写入: %v %+v", err, view)
	}
	requests := server.RequestsFor("hook.write")
	if len(requests) != 1 {
		t.Fatalf("用户来源钩子请求次数不符: %d", len(requests))
	}
	var params struct {
		Hook struct {
			Source string `json:"source"`
			Event  string `json:"event"`
		} `json:"hook"`
	}
	if err := json.Unmarshal(requests[0].Params, &params); err != nil {
		t.Fatalf("解析钩子写入参数: %v", err)
	}
	if params.Hook.Source != "user" || params.Hook.Event != "pre" {
		t.Fatalf("钩子写入参数不符: %+v", params)
	}
}

func TestAppServerToolServersAndImport(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "toolServer.list", Result: map[string]any{"servers": []any{
				map[string]any{
					"id": "srv-1", "name": "现有服务器", "transport": "stdio", "command": "srv-bin",
					"args": []string{"--flag"}, "enabled": true, "connected": true,
					"envKeys": []string{"TOKEN"}, "toolCount": 1,
				},
			}}},
			{Method: "toolServer.probe", Result: map[string]any{
				"serverId": "srv-1", "reachable": true,
				"tools": []any{
					map[string]any{"name": "search", "description": "搜索"},
					map[string]any{"name": "fetch"},
				},
			}},
			{Method: "toolServer.import.preview", Result: map[string]any{"source": "config-a", "candidates": []any{
				map[string]any{"key": "alpha", "server": map[string]any{"name": "alpha", "transport": "stdio", "command": "alpha-bin"}},
				map[string]any{"key": "beta", "server": map[string]any{"name": "beta", "transport": "http", "url": "https://beta.example"}},
				map[string]any{"key": "gamma", "server": map[string]any{"name": "gamma", "transport": "sse", "url": "https://gamma.example"}},
			}}},
			{Method: "toolServer.add", Result: map[string]any{"serverId": "srv-new"}},
			{Method: "toolServer.update", Result: nil},
			{Method: "toolServer.remove", Result: nil},
		},
	})
	harness := newAppServerHarness(client)
	service := newService(t, Config{ToolServers: harness, Writer: newWriteManager(t, harness)})
	ctx := context.Background()

	servers, err := service.ListToolServers(ctx)
	if err != nil || len(servers.Items) != 1 {
		t.Fatalf("读取工具服务器: %v %+v", err, servers)
	}
	if servers.Items[0].EnvKeys[0] != "TOKEN" {
		t.Fatalf("环境变量键名不符: %+v", servers.Items[0])
	}
	if requests := server.RequestsFor("toolServer.probe"); len(requests) != 0 {
		t.Fatalf("读取清单触发了探测: %d", len(requests))
	}

	added, err := service.AddToolServer(ctx, ToolServerSpec{
		Name: "新服务器", Transport: ToolTransportStdio, Command: "new-bin",
		Args: []string{"--serve"}, Env: map[string]string{"TOKEN": "secret"},
	})
	if err != nil || added.Status != write.StatusSucceeded {
		t.Fatalf("新增工具服务器: %v %+v", err, added)
	}
	if requests := server.RequestsFor("toolServer.add"); len(requests) != 1 {
		t.Fatalf("新增请求次数不符: %d", len(requests))
	}
	if requests := server.RequestsFor("toolServer.probe"); len(requests) != 0 {
		t.Fatalf("新增触发了探测: %d", len(requests))
	}

	probe, err := service.ProbeToolServer(ctx, "srv-1")
	if err != nil || !probe.Reachable || len(probe.Tools) != 2 {
		t.Fatalf("探测工具服务器: %v %+v", err, probe)
	}
	if requests := server.RequestsFor("toolServer.probe"); len(requests) != 1 {
		t.Fatalf("探测请求次数不符: %d", len(requests))
	}
	if requests := server.RequestsFor("toolServer.add"); len(requests) != 1 {
		t.Fatalf("探测写出了配置: %d", len(requests))
	}
	if snapshot, ok := service.ProbeSnapshot("srv-1"); !ok || len(snapshot.Tools) != 2 {
		t.Fatalf("探测预览没有保留: %+v", snapshot)
	}

	if _, err := service.UpdateToolServer(ctx, "srv-1", ToolServerSpec{
		Name: "改名服务器", Transport: ToolTransportHTTP, URL: "https://srv.example",
	}); err != nil {
		t.Fatalf("修改工具服务器: %v", err)
	}
	if requests := server.RequestsFor("toolServer.update"); len(requests) != 1 {
		t.Fatalf("修改请求次数不符: %d", len(requests))
	}

	preview, err := service.PreviewToolServerImport(ctx, "config-a")
	if err != nil || len(preview.Candidates) != 3 {
		t.Fatalf("导入预览: %v %+v", err, preview)
	}
	if requests := server.RequestsFor("toolServer.add"); len(requests) != 1 {
		t.Fatalf("预览写出了配置: %d", len(requests))
	}

	if _, err := service.ApplyToolServerImport(ctx, "config-a", []string{"missing"}); !errors.Is(err, ErrUnknownCandidate) {
		t.Fatalf("未知候选没有拒绝: %v", err)
	}
	result, err := service.ApplyToolServerImport(ctx, "config-a", []string{"gamma", "beta"})
	if err != nil || result.Written != 2 {
		t.Fatalf("按勾选写入: %v %+v", err, result)
	}
	addParams := methodParams(t, server, "toolServer.add")
	if len(addParams) != 3 {
		t.Fatalf("导入写入条数不等于勾选条数: %d", len(addParams))
	}
	imported := addParams[1:]
	names := make([]string, 0, len(imported))
	for _, params := range imported {
		names = append(names, params["name"].(string))
		if params["importSource"] != "config-a" || params["candidateKey"] == "" {
			t.Fatalf("导入写入缺少来源或候选键: %+v", params)
		}
	}
	if strings.Join(names, ",") != "beta,gamma" {
		t.Fatalf("导入只应写入勾选条目: %v", names)
	}
	for _, params := range imported {
		if params["name"] == "alpha" {
			t.Fatalf("未勾选条目被写入: %+v", params)
		}
	}

	if _, err := service.RemoveToolServer(ctx, "srv-1"); err != nil {
		t.Fatalf("删除工具服务器: %v", err)
	}
	if requests := server.RequestsFor("toolServer.remove"); len(requests) != 1 {
		t.Fatalf("删除请求次数不符: %d", len(requests))
	}
	servers, err = service.ListToolServers(ctx)
	if err != nil {
		t.Fatalf("删除后读取: %v", err)
	}
	for _, item := range servers.Items {
		if item.ID == "srv-1" {
			t.Fatalf("删除后清单仍有该服务器: %+v", servers.Items)
		}
	}
}

func TestAppServerSchedulerFollowing(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Faults: []fakeappserver.Fault{{Method: "scheduler.list", Occurrence: 1, UnknownMethod: true}},
		Replies: []fakeappserver.Reply{
			{Method: "scheduler.list", Result: map[string]any{"tasks": []any{
				map[string]any{"id": "task-1", "name": "夜间构建", "schedule": "0 2 * * *", "enabled": true},
				map[string]any{"id": "task-2", "name": "每周清理", "schedule": "0 3 * * 0", "enabled": false},
			}}},
		},
	})
	harness := newAppServerHarness(client)
	var logs []string
	service := newService(t, Config{
		Scheduler: harness,
		Logf: func(format string, args ...any) {
			logs = append(logs, fmt.Sprintf(format, args...))
		},
	})
	ctx := context.Background()

	snapshot, err := service.ListScheduledTasks(ctx)
	if !errors.Is(err, ErrCapabilityMissing) || !snapshot.CapabilityMissing {
		t.Fatalf("计划任务缺能力没有分类: %v %+v", err, snapshot)
	}
	if len(logs) == 0 {
		t.Fatal("计划任务缺能力没有记录日志")
	}

	snapshot, err = service.ListScheduledTasks(ctx)
	if err != nil || len(snapshot.Items) != 2 {
		t.Fatalf("能力恢复后读取任务: %v %+v", err, snapshot)
	}
	if requests := server.RequestsFor("scheduler.list"); len(requests) != 2 {
		t.Fatalf("能力恢复请求次数不符: %d", len(requests))
	}

	arrived := attachNotices(t, client, service)
	if err := server.Notify("scheduler.changed", map[string]any{"reason": "updated"}); err != nil {
		t.Fatalf("推送计划任务通知: %v", err)
	}
	<-arrived
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("等待计划任务跟随: %v", err)
	}
	if requests := server.RequestsFor("scheduler.list"); len(requests) != 3 {
		t.Fatalf("通知后重读次数不符: %d", len(requests))
	}
	time.Sleep(100 * time.Millisecond)
	if requests := server.RequestsFor("scheduler.list"); len(requests) != 3 {
		t.Fatalf("计划任务出现定时轮询: %d", len(requests))
	}
	if stats := service.FollowStats(); stats.Reads[TopicScheduler] != 1 {
		t.Fatalf("计划任务跟随统计不符: %+v", stats)
	}
}

func TestAppServerWakeupFollowAndCancel(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "wakeup.list", Result: map[string]any{"wakeups": []any{
				map[string]any{
					"id": "wake-1", "sessionId": "s-1", "branchId": "main",
					"title": "提醒", "prompt": "继续", "triggerAt": int64(1700000000000), "status": "scheduled",
				},
			}}},
			{Method: "wakeup.cancel", Result: nil},
		},
	})
	harness := newAppServerHarness(client)
	var logs []string
	service := newService(t, Config{
		Wakeups: harness,
		Writer:  newWriteManager(t, harness),
		Logf: func(format string, args ...any) {
			logs = append(logs, fmt.Sprintf(format, args...))
		},
	})
	ctx := context.Background()
	scope := WakeupScope{SessionID: "s-1", BranchID: "main"}

	snapshot, err := service.ListWakeups(ctx, scope)
	if err != nil || len(snapshot.Items) != 1 {
		t.Fatalf("读取唤醒: %v %+v", err, snapshot)
	}
	arrived := attachNotices(t, client, service)

	view, err := service.CancelWakeup(ctx, "wake-1")
	if err != nil || view.Status != write.StatusSucceeded {
		t.Fatalf("取消唤醒: %v %+v", err, view)
	}
	if requests := server.RequestsFor("wakeup.cancel"); len(requests) != 1 {
		t.Fatalf("取消请求次数不符: %d", len(requests))
	}
	if requests := server.RequestsFor("wakeup.list"); len(requests) != 1 {
		t.Fatalf("取消本身不应重读: %d", len(requests))
	}

	if err := server.Notify("wakeup.changed", map[string]any{"sessionId": "s-1", "branchId": "main", "wakeupId": "wake-1"}); err != nil {
		t.Fatalf("推送唤醒通知: %v", err)
	}
	<-arrived
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("等待唤醒跟随: %v", err)
	}
	if requests := server.RequestsFor("wakeup.list"); len(requests) != 2 {
		t.Fatalf("通知后重读次数不符: %d", len(requests))
	}
	if snapshot := service.WakeupSnapshot(scope); len(snapshot.Items) != 0 {
		t.Fatalf("取消后唤醒记录仍在: %+v", snapshot)
	}
	time.Sleep(100 * time.Millisecond)
	if requests := server.RequestsFor("wakeup.list"); len(requests) != 2 {
		t.Fatalf("唤醒跟随出现定时轮询: %d", len(requests))
	}
	stats := service.FollowStats()
	if stats.WakeupReads[scopeKey(scope)] != 1 || stats.Reads[TopicWakeups] != 1 {
		t.Fatalf("唤醒跟随统计不符: %+v", stats)
	}
	notified := 0
	for _, line := range logs {
		if strings.Contains(line, "topic=wakeups") {
			notified++
		}
	}
	if notified != 1 {
		t.Fatalf("唤醒通知重读日志次数不符: %d %v", notified, logs)
	}

	snapshot, err = service.ListWakeups(ctx, scope)
	if err != nil || len(snapshot.Items) != 0 {
		t.Fatalf("取消后再次读取仍有记录: %v %+v", err, snapshot)
	}
	if requests := server.RequestsFor("wakeup.list"); len(requests) != 3 {
		t.Fatalf("再次读取请求次数不符: %d", len(requests))
	}
}

func TestAppServerWorkflowRuns(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "workflow.list", Result: map[string]any{"workflows": []any{
				map[string]any{"id": "wf-1", "name": "发布", "version": "1", "source": "project", "steps": []any{
					map[string]any{"id": "step-1", "name": "构建", "kind": "shell"},
				}},
			}}},
			{Method: "workflow.get", Result: map[string]any{
				"id": "wf-1", "name": "发布", "version": "1", "source": "project", "steps": []any{
					map[string]any{"id": "step-1", "name": "构建", "kind": "shell"},
					map[string]any{"id": "step-2", "name": "测试", "kind": "shell"},
				},
			}},
			{Method: "workflow.runs", Result: map[string]any{"runs": []any{
				map[string]any{"id": "run-1", "workflowId": "wf-1", "status": "running", "startedAt": int64(1700000000000)},
			}}},
			{Method: "workflow.start", Result: map[string]any{"runId": "run-2"}},
			{Method: "workflow.stop", Result: nil},
		},
	})
	harness := newAppServerHarness(client)
	service := newService(t, Config{Workflows: harness, Writer: newWriteManager(t, harness)})
	ctx := context.Background()

	workflows, err := service.ListWorkflows(ctx)
	if err != nil || len(workflows.Items) != 1 {
		t.Fatalf("浏览工作流: %v %+v", err, workflows)
	}
	definition, err := service.ReadWorkflow(ctx, "wf-1")
	if err != nil || len(definition.Steps) != 2 {
		t.Fatalf("读取工作流定义: %v %+v", err, definition)
	}
	runs, err := service.ListWorkflowRuns(ctx, "wf-1")
	if err != nil || len(runs.Items) != 1 || runs.Items[0].Status != WorkflowRunning {
		t.Fatalf("读取运行状态: %v %+v", err, runs)
	}

	started, err := service.StartWorkflow(ctx, "wf-1", map[string]string{"target": "prod"})
	if err != nil || started.Status != write.StatusSucceeded {
		t.Fatalf("启动工作流: %v %+v", err, started)
	}
	if params := methodParams(t, server, "workflow.start"); len(params) != 1 ||
		params[0]["workflowId"] != "wf-1" {
		t.Fatalf("启动参数不符: %+v", params)
	}
	stopped, err := service.StopWorkflow(ctx, "run-2")
	if err != nil || stopped.Status != write.StatusSucceeded {
		t.Fatalf("停止工作流: %v %+v", err, stopped)
	}
	if params := methodParams(t, server, "workflow.stop"); len(params) != 1 || params[0]["runId"] != "run-2" {
		t.Fatalf("停止参数不符: %+v", params)
	}

	arrived := attachNotices(t, client, service)
	if err := server.Notify("workflow.changed", map[string]any{"workflowId": "wf-1"}); err != nil {
		t.Fatalf("推送工作流通知: %v", err)
	}
	<-arrived
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("等待工作流跟随: %v", err)
	}
	if requests := server.RequestsFor("workflow.runs"); len(requests) != 2 {
		t.Fatalf("工作流通知重读次数不符: %d", len(requests))
	}
	time.Sleep(100 * time.Millisecond)
	if requests := server.RequestsFor("workflow.runs"); len(requests) != 2 {
		t.Fatalf("工作流出现定时轮询: %d", len(requests))
	}
}

func TestAppServerMissingCapabilityOnWire(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{})
	harness := newAppServerHarness(client)
	manager := newWriteManager(t, harness)
	service := newService(t, Config{Plugins: harness, Writer: manager})
	ctx := context.Background()

	snapshot, err := service.ListPlugins(ctx)
	if !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("wire 能力缺失没有分类: %v", err)
	}
	if !snapshot.CapabilityMissing {
		t.Fatalf("wire 能力缺失没有标记读数: %+v", snapshot)
	}
	if kind := failure.KindOf(err); kind != failure.KindNonRetryable {
		t.Fatalf("能力缺失的连接分类不符: %s", kind)
	}
	if requests := server.RequestsFor("plugin.list"); len(requests) != 1 {
		t.Fatalf("能力缺失读取次数不符: %d", len(requests))
	}

	view, err := service.InstallPlugin(ctx, InstallSource{Kind: SourceLocal, Ref: "/tmp/plugin"})
	if err != nil {
		t.Fatalf("写入返回等待错误: %v", err)
	}
	if view.Status != write.StatusFailed {
		t.Fatalf("能力缺失写入没有明确失败: %+v", view)
	}
	if view.Reason == "" {
		t.Fatalf("能力缺失写入没有原因: %+v", view)
	}
	if requests := server.RequestsFor("plugin.install"); len(requests) != 1 {
		t.Fatalf("能力缺失写入没有只发一次: %d", len(requests))
	}
}

func hasPlugin(plugins []Plugin, id string) bool {
	for _, plugin := range plugins {
		if plugin.ID == id && plugin.Installed {
			return true
		}
	}
	return false
}

func hasSkill(skills []Skill, id string) bool {
	for _, skill := range skills {
		if skill.ID == id {
			return true
		}
	}
	return false
}

func hasSkillEnabled(skills []Skill, id string) bool {
	for _, skill := range skills {
		if skill.ID == id && skill.Enabled {
			return true
		}
	}
	return false
}
