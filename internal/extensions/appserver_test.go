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
	toolSources     map[string]string
	sessionCall     func(context.Context, string, any) (json.RawMessage, error)
	// overlayWrites 为真时，读取会在服务端回执上叠加本测试记录的写入结果；
	// 真实 daemon 证据应关闭，直接读取服务端事实。
	overlayWrites bool
	addedCount    int
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
		toolSources:     make(map[string]string),
		overlayWrites:   true,
	}
}

// 以下读取实现是集成层在真实 wire 与本领域类型之间的映射。方法名与主要回执
// 形状按 alpha.21 实测；尚未冻结的条目只取本测试需要的字段。

type wirePluginCandidate struct {
	ID    string   `json:"id"`
	Kinds []string `json:"kinds"`
}

// wirePlugin 是 `plugin.list` 的一项；alpha.21 的 list 回执是数组。
type wirePlugin struct {
	Name       string                `json:"name"`
	Use        *string               `json:"use"`
	Off        bool                  `json:"off"`
	Locked     bool                  `json:"locked"`
	Candidates []wirePluginCandidate `json:"candidates"`
	Status     string                `json:"status"`
	Error      string                `json:"error"`
}

func (h *appServerHarness) ReadPlugins(ctx context.Context) ([]Plugin, error) {
	var entries []wirePlugin
	if err := h.client.CallInto(ctx, "plugin.list", nil, &entries); err != nil {
		return nil, err
	}
	plugins := make([]Plugin, 0, len(entries))
	for _, plugin := range entries {
		use := ""
		if plugin.Use != nil {
			use = *plugin.Use
		}
		plugins = append(plugins, Plugin{
			ID:        plugin.Name,
			Name:      plugin.Name,
			Source:    sourceFromPluginCandidates(plugin.Candidates),
			Installed: true,
			Enabled:   !plugin.Off,
			Use:       use,
			Locked:    plugin.Locked,
			Status:    plugin.Status,
			Error:     plugin.Error,
		})
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.overlayWrites {
		for index := range plugins {
			if enabled, ok := h.pluginEnabled[plugins[index].ID]; ok {
				plugins[index].Enabled = enabled
			}
			if h.pluginUpdated[plugins[index].ID] {
				plugins[index].UpdateAvailable = false
			}
		}
		for _, installed := range h.pluginInstalls {
			if !hasPluginByID(plugins, installed.ID) {
				plugins = append(plugins, installed)
			}
		}
	}
	return plugins, nil
}

func sourceFromPluginCandidates(candidates []wirePluginCandidate) Source {
	for _, candidate := range candidates {
		for _, kind := range candidate.Kinds {
			switch kind {
			case "git":
				return SourceGit
			case "path", "local":
				return SourceLocal
			case "builtin", "bundled", "code":
				return SourceBundled
			}
		}
	}
	return SourceUnknown
}

func hasPluginByID(plugins []Plugin, id string) bool {
	for _, plugin := range plugins {
		if plugin.ID == id {
			return true
		}
	}
	return false
}

// wireSkill 是 `skills.catalog` 的一项。
type wireSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Enabled     bool   `json:"enabled"`
}

func (h *appServerHarness) ReadSkills(ctx context.Context) ([]Skill, error) {
	var result struct {
		Version uint64      `json:"version"`
		Entries []wireSkill `json:"entries"`
		Issues  []any       `json:"issues"`
	}
	if err := h.client.CallInto(ctx, "skills.catalog", nil, &result); err != nil {
		return nil, err
	}
	skills := make([]Skill, 0, len(result.Entries))
	for _, entry := range result.Entries {
		skills = append(skills, Skill{
			ID:          entry.ID,
			Name:        entry.Name,
			Description: entry.Description,
			Source:      sourceFromWire(entry.Source),
			Installed:   true,
			Enabled:     entry.Enabled,
		})
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.overlayWrites {
		for index := range skills {
			if enabled, ok := h.skillEnabled[skills[index].ID]; ok {
				skills[index].Enabled = enabled
			}
		}
		for _, installed := range h.skillInstalls {
			if !hasSkillByID(skills, installed.ID) {
				skills = append(skills, installed)
			}
		}
	}
	return skills, nil
}

func hasSkillByID(skills []Skill, id string) bool {
	for _, skill := range skills {
		if skill.ID == id {
			return true
		}
	}
	return false
}

// wireHook 是 `hooks.catalog` 里 hooks 数组的一项。真实条目形状尚未冻结；测试
// 适配器只接受带 id 的扁平对象，非空且无法识别时明确报错。
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
		UserRevision     string            `json:"userRevision"`
		UserDocument     json.RawMessage   `json:"userDocument"`
		ProjectsRevision string            `json:"projectsRevision"`
		DisableAllHooks  bool              `json:"disableAllHooks"`
		Sources          []json.RawMessage `json:"sources"`
		Hooks            []json.RawMessage `json:"hooks"`
	}
	if err := h.client.CallInto(ctx, "hooks.catalog", nil, &result); err != nil {
		return nil, err
	}
	hooks := make([]Hook, 0, len(result.Hooks))
	for index, raw := range result.Hooks {
		var hook wireHook
		if err := json.Unmarshal(raw, &hook); err != nil || hook.ID == "" {
			return nil, fmt.Errorf("hooks.catalog hooks[%d] 的条目形状尚未冻结: %w", index, err)
		}
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
	if h.overlayWrites {
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
	}
	return hooks, nil
}

// wireToolServer 是 `mcp.catalog` 的一项。真实回执只有 id/source/enabled/
// transport/auth；其余字段只由测试脚本给出，用于保持摘要完整。
type wireToolServer struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Source    string   `json:"source"`
	Transport string   `json:"transport"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	URL       string   `json:"url"`
	Enabled   bool     `json:"enabled"`
	Connected bool     `json:"connected"`
	EnvKeys   []string `json:"envKeys"`
	ToolCount int      `json:"toolCount"`
	Auth      *struct {
		Status        string   `json:"status"`
		MissingScopes []string `json:"missingScopes"`
	} `json:"auth"`
}

func (h *appServerHarness) ReadToolServers(ctx context.Context) ([]ToolServer, error) {
	var result struct {
		UserRevision    string            `json:"userRevision"`
		ProjectRevision *string           `json:"projectRevision"`
		Servers         []wireToolServer  `json:"servers"`
		Invalid         []json.RawMessage `json:"invalid"`
	}
	if err := h.client.CallInto(ctx, "mcp.catalog", nil, &result); err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.toolSources == nil {
		h.toolSources = make(map[string]string)
	}
	servers := make([]ToolServer, 0, len(result.Servers)+len(h.toolAdded))
	for _, server := range result.Servers {
		if h.overlayWrites && h.toolRemoved[server.ID] {
			continue
		}
		name := server.Name
		if name == "" {
			name = server.ID
		}
		mapped := ToolServer{
			ID:        server.ID,
			Name:      name,
			Source:    sourceFromWire(server.Source),
			Transport: ToolTransport(server.Transport),
			Command:   server.Command,
			Args:      append([]string(nil), server.Args...),
			URL:       server.URL,
			Enabled:   server.Enabled,
			Connected: server.Connected,
			EnvKeys:   append([]string(nil), server.EnvKeys...),
			ToolCount: server.ToolCount,
		}
		if server.Auth != nil {
			mapped.AuthStatus = server.Auth.Status
			mapped.AuthMissingScopes = append([]string(nil), server.Auth.MissingScopes...)
		}
		if h.overlayWrites {
			if spec, ok := h.toolUpdated[server.ID]; ok {
				mapped.Name = spec.Name
				mapped.Transport = spec.Transport
				mapped.Command = spec.Command
				mapped.Args = append([]string(nil), spec.Args...)
				mapped.URL = spec.URL
				mapped.EnvKeys = sortedEnvKeys(spec.Env)
				if spec.Source != "" {
					mapped.Source = spec.Source
				}
			}
		}
		h.toolSources[server.ID] = server.Source
		servers = append(servers, mapped)
	}
	if h.overlayWrites {
		servers = append(servers, h.toolAdded...)
	}
	return servers, nil
}

func (h *appServerHarness) sourceForTool(serverID string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if source := h.toolSources[serverID]; source != "" {
		return source
	}
	return string(SourceUser)
}

func (h *appServerHarness) ProbeToolServer(ctx context.Context, serverID string) (ToolServerProbe, error) {
	var result struct {
		ServerID  string           `json:"serverId"`
		Reachable bool             `json:"reachable"`
		Tools     []ToolDescriptor `json:"tools"`
		Reason    string           `json:"reason"`
	}
	params := map[string]any{"id": serverID, "source": h.sourceForTool(serverID)}
	if err := h.client.CallInto(ctx, "mcp.test", params, &result); err != nil {
		return ToolServerProbe{}, err
	}
	if result.ServerID == "" {
		result.ServerID = serverID
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
	if err := h.client.CallInto(ctx, "mcp.import.preview", map[string]any{"source": source}, &result); err != nil {
		return ImportPreview{}, err
	}
	preview := ImportPreview{Source: result.Source, Candidates: make([]ImportCandidate, 0, len(result.Candidates))}
	for _, candidate := range result.Candidates {
		name := candidate.Server.Name
		if name == "" {
			name = candidate.Server.ID
		}
		preview.Candidates = append(preview.Candidates, ImportCandidate{
			Key: candidate.Key,
			Server: ToolServerSpec{
				Name:      name,
				Transport: ToolTransport(candidate.Server.Transport),
				Command:   candidate.Server.Command,
				Args:      append([]string(nil), candidate.Server.Args...),
				URL:       candidate.Server.URL,
				Source:    sourceFromWire(candidate.Server.Source),
			},
			Existing: candidate.Existing,
			Warnings: append([]string(nil), candidate.Warnings...),
		})
	}
	return preview, nil
}

// wireScheduledTask 是 `schedule.list` 的一项；wire 是 snake_case。
type wireScheduledTask struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Schedule   json.RawMessage `json:"schedule"`
	Prompt     string          `json:"prompt"`
	Cwd        string          `json:"cwd"`
	Mode       string          `json:"mode"`
	Enabled    bool            `json:"enabled"`
	Revision   int64           `json:"revision"`
	NextWindow json.RawMessage `json:"next_window"`
	LastRun    json.RawMessage `json:"last_run"`
}

func (h *appServerHarness) ReadScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	var result struct {
		Jobs []wireScheduledTask `json:"jobs"`
	}
	if err := h.client.CallInto(ctx, "schedule.list", nil, &result); err != nil {
		return nil, err
	}
	tasks := make([]ScheduledTask, 0, len(result.Jobs))
	for _, job := range result.Jobs {
		tasks = append(tasks, ScheduledTask{
			ID:       job.ID,
			Name:     job.Name,
			Schedule: string(job.Schedule),
			Enabled:  job.Enabled,
			Revision: job.Revision,
			Prompt:   job.Prompt,
			Cwd:      job.Cwd,
			Mode:     job.Mode,
		})
	}
	return tasks, nil
}

// wireWakeup 是 `session.wakeup.list` 的一项（camelCase，时间 RFC 3339）。
type wireWakeup struct {
	WakeupID string `json:"wakeupId"`
	At       string `json:"at"`
	Prompt   string `json:"prompt"`
	State    string `json:"state"`
}

func (h *appServerHarness) ReadWakeups(ctx context.Context, scope WakeupScope) ([]Wakeup, error) {
	params := map[string]any{"sessionId": scope.SessionID}
	if scope.BranchID != "" {
		params["branchId"] = scope.BranchID
	}
	var result struct {
		Wakeups []wireWakeup `json:"wakeups"`
	}
	raw, err := h.call(ctx, "session.wakeup.list", params)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	wakeups := make([]Wakeup, 0, len(result.Wakeups))
	for _, wakeup := range result.Wakeups {
		if h.overlayWrites && h.wakeupCancelled[wakeup.WakeupID] {
			continue
		}
		triggerAt := time.Time{}
		if wakeup.At != "" {
			parsed, err := time.Parse(time.RFC3339, wakeup.At)
			if err != nil {
				return nil, fmt.Errorf("session.wakeup.list 的 at 无法解析: %w", err)
			}
			triggerAt = parsed
		}
		wakeups = append(wakeups, Wakeup{
			ID:        wakeup.WakeupID,
			SessionID: scope.SessionID,
			BranchID:  scope.BranchID,
			Prompt:    wakeup.Prompt,
			TriggerAt: triggerAt,
			Status:    WakeupStatus(wakeup.State),
		})
	}
	return wakeups, nil
}

// call 按方法路由：唤醒方法属于会话流，其余管理方法走控制连接。
func (h *appServerHarness) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if h.sessionCall != nil && strings.HasPrefix(method, "session.wakeup.") {
		return h.sessionCall(ctx, method, params)
	}
	return h.client.Call(ctx, method, params)
}

// Send 是写登记的发送方：按实测方法名映射到 wire，只调用一次。
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
			"source": installSourceParams(payload.Source),
		})
		if err != nil {
			return write.Reply{}, err
		}
		pluginID := installRefID(payload.Source)
		h.mu.Lock()
		h.pluginInstalls = append(h.pluginInstalls, Plugin{
			ID:        pluginID,
			Name:      pluginID,
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
		method := "plugin.disable"
		if payload.Enabled {
			method = "plugin.enable"
		}
		raw, err := h.client.Call(ctx, method, map[string]any{"name": payload.PluginID})
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
		raw, err := h.client.Call(ctx, "plugin.update", map[string]any{"id": payload.PluginID})
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
		raw, err := h.client.Call(ctx, "skills.install", map[string]any{
			"source": installSourceParams(payload.Source),
			"skills": []any{},
		})
		if err != nil {
			return write.Reply{}, err
		}
		skillID := installRefID(payload.Source)
		h.mu.Lock()
		h.skillInstalls = append(h.skillInstalls, Skill{
			ID:        skillID,
			Name:      skillID,
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
		raw, err := h.client.Call(ctx, "skills.set_enabled", map[string]any{
			"id":      payload.SkillID,
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
		hook := map[string]any{
			"id":      payload.Hook.ID,
			"name":    payload.Hook.Name,
			"event":   payload.Hook.Event,
			"command": payload.Hook.Command,
			"source":  string(payload.Hook.Source),
			"enabled": payload.Hook.Enabled,
		}
		raw, err := h.client.Call(ctx, "hooks.save", map[string]any{
			"expectedRevision": payload.ExpectedRevision,
			"document":         map[string]any{"hooks": []any{hook}},
		})
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
		if err := json.Unmarshal(managed.Payload, &payload); err == nil && payload.Server.Name != "" {
			if payload.ImportSource != "" {
				raw, err := h.client.Call(ctx, "mcp.import.apply", map[string]any{
					"source": payload.ImportSource,
					"ids":    []string{payload.CandidateKey},
				})
				if err != nil {
					return write.Reply{}, err
				}
				return write.Reply{Result: raw}, nil
			}
			raw, err := h.client.Call(ctx, "mcp.save", mcpSaveParams(payload.Server.Name, payload.Server))
			if err != nil {
				return write.Reply{}, err
			}
			h.mu.Lock()
			h.addedCount++
			h.toolAdded = append(h.toolAdded, ToolServer{
				ID:        fmt.Sprintf("srv-added-%d", h.addedCount),
				Name:      payload.Server.Name,
				Source:    payload.Server.Source,
				Transport: payload.Server.Transport,
				Command:   payload.Server.Command,
				Args:      append([]string(nil), payload.Server.Args...),
				URL:       payload.Server.URL,
				EnvKeys:   sortedEnvKeys(payload.Server.Env),
			})
			h.mu.Unlock()
			return write.Reply{Result: raw}, nil
		}
		var update ToolServerUpdatePayload
		if err := json.Unmarshal(managed.Payload, &update); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "mcp.save", mcpSaveParams(update.ServerID, update.Server))
		if err != nil {
			return write.Reply{}, err
		}
		h.mu.Lock()
		h.toolUpdated[update.ServerID] = update.Server
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpToolServerRemove:
		var payload ToolServerRemovePayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.client.Call(ctx, "mcp.remove", map[string]any{
			"id":     payload.ServerID,
			"source": string(SourceUser),
		})
		if err != nil {
			return write.Reply{}, err
		}
		h.mu.Lock()
		h.toolRemoved[payload.ServerID] = true
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpWakeupCancel:
		var payload WakeupCancelPayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			return write.Reply{}, err
		}
		raw, err := h.call(ctx, "session.wakeup.cancel", map[string]any{
			"sessionId": payload.SessionID,
			"wakeupId":  payload.WakeupID,
		})
		if err != nil {
			return write.Reply{}, err
		}
		h.mu.Lock()
		h.wakeupCancelled[payload.WakeupID] = true
		h.mu.Unlock()
		return write.Reply{Result: raw}, nil
	case OpWorkflowStart, OpWorkflowStop:
		// alpha.21 不注册 workflow.*；测试适配器不发明服务端方法。
		return write.Reply{}, fmt.Errorf("workflow 在 alpha.21 没有 wire 方法")
	default:
		return write.Reply{}, fmt.Errorf("未支持的管理写入 %q", managed.Op)
	}
}

func installSourceParams(source InstallSource) map[string]any {
	switch source.Kind {
	case SourceGit:
		return map[string]any{"git": source.Ref}
	default:
		return map[string]any{"path": source.Ref}
	}
}

func installRefID(source InstallSource) string {
	ref := strings.TrimSuffix(strings.TrimRight(source.Ref, "/"), ".git")
	if index := strings.LastIndexAny(ref, "/\\"); index >= 0 && index+1 < len(ref) {
		return ref[index+1:]
	}
	if ref == "" {
		return "installed"
	}
	return ref
}

// mcpSaveParams 把本领域的服务器摘要映射成 mcp.save 参数；配置文档尚未冻结，
// 测试适配器按命令/地址构造一份最小文档。
func mcpSaveParams(id string, spec ToolServerSpec) map[string]any {
	source := spec.Source
	if source == "" {
		source = SourceUser
	}
	configuration := map[string]any{
		"name":      spec.Name,
		"transport": string(spec.Transport),
		"command":   spec.Command,
		"url":       spec.URL,
	}
	if len(spec.Args) > 0 {
		configuration["args"] = append([]string(nil), spec.Args...)
	}
	if len(spec.Env) > 0 {
		env := make(map[string]string, len(spec.Env))
		for key, value := range spec.Env {
			env[key] = value
		}
		configuration["env"] = env
	}
	return map[string]any{
		"id":              id,
		"source":          string(source),
		"expectedVersion": spec.ExpectedVersion,
		"configuration":   configuration,
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
		case "management.progress":
			var params struct {
				Method string `json:"method"`
			}
			if err := json.Unmarshal(message.Params, &params); err != nil {
				return
			}
			if !strings.HasPrefix(params.Method, "schedule.") {
				return
			}
			_ = service.Notify(Notice{Topic: TopicSchedule})
		case "session.notice":
			var params struct {
				SessionID string `json:"sessionId"`
				Source    string `json:"source"`
				Kind      string `json:"kind"`
				Data      struct {
					BranchID string `json:"branchId"`
				} `json:"data"`
			}
			if err := json.Unmarshal(message.Params, &params); err != nil {
				return
			}
			if params.Source != "Scheduler" || params.Kind != "wakeup" {
				return
			}
			_ = service.Notify(Notice{Topic: TopicWakeups, SessionID: params.SessionID, BranchID: params.Data.BranchID})
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
			{Method: "plugin.list", Result: []any{
				map[string]any{
					"name": "installed", "use": "builtin", "off": false, "locked": false,
					"candidates": []any{map[string]any{"id": "builtin", "kinds": []string{"builtin"}}},
					"status":     "running",
				},
				map[string]any{
					"name": "optional", "use": nil, "off": true, "locked": false,
					"candidates": []any{map[string]any{"id": "builtin", "kinds": []string{"builtin"}}},
				},
			}},
			{Method: "plugin.install", Result: map[string]any{"status": "installed"}},
			{Method: "plugin.enable", Result: map[string]any{"status": "running"}},
			{Method: "plugin.disable", Result: map[string]any{}},
			{Method: "plugin.update", Result: map[string]any{"updated": true}},
			{Method: "skills.catalog", Result: map[string]any{"version": uint64(1), "entries": []any{
				map[string]any{"id": "skill-1", "name": "提交信息", "source": "bundled", "enabled": false},
			}, "issues": []any{}}},
			{Method: "skills.install", Result: map[string]any{"version": uint64(2), "issues": []any{}}},
			{Method: "skills.set_enabled", Result: map[string]any{"version": uint64(3), "issues": []any{}}},
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
	if plugins.Items[0].ID != "installed" || !plugins.Items[0].Enabled || plugins.Items[0].Status != "running" {
		t.Fatalf("插件条目映射不符: %+v", plugins.Items[0])
	}
	if plugins.Items[1].ID != "optional" || plugins.Items[1].Enabled || plugins.Items[1].Locked {
		t.Fatalf("停用插件映射不符: %+v", plugins.Items[1])
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
	if source["path"] != "/tmp/plugins/demo" {
		t.Fatalf("安装来源没有按 plugin.install 的 path 形状透传: %+v", installParams[0])
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
	if params := methodParams(t, server, "plugin.enable"); len(params) != 1 || params[0]["name"] != "demo" {
		t.Fatalf("启用参数不符: %+v", params)
	}
	if _, err := service.SetPluginEnabled(ctx, "demo", false); err != nil {
		t.Fatalf("停用插件: %v", err)
	}
	if params := methodParams(t, server, "plugin.disable"); len(params) != 1 || params[0]["name"] != "demo" {
		t.Fatalf("停用参数不符: %+v", params)
	}

	updated, err := service.UpdatePlugin(ctx, "demo")
	if err != nil || updated.Status != write.StatusSucceeded {
		t.Fatalf("更新插件: %v %+v", err, updated)
	}
	if params := methodParams(t, server, "plugin.update"); len(params) != 1 || params[0]["id"] != "demo" {
		t.Fatalf("更新参数不符: %+v", params)
	}

	skills, err := service.ListSkills(ctx)
	if err != nil || len(skills.Items) != 1 {
		t.Fatalf("读取技能: %v %+v", err, skills)
	}
	if _, err := service.SetSkillEnabled(ctx, "skill-1", true); err != nil {
		t.Fatalf("启用技能: %v", err)
	}
	if params := methodParams(t, server, "skills.set_enabled"); len(params) != 1 ||
		params[0]["id"] != "skill-1" || params[0]["enabled"] != true {
		t.Fatalf("技能启用参数不符: %+v", params)
	}
	if _, err := service.InstallSkill(ctx, InstallSource{Kind: SourceGit, Ref: "https://example.com/skill.git"}); err != nil {
		t.Fatalf("安装技能: %v", err)
	}
	if params := methodParams(t, server, "skills.install"); len(params) != 1 {
		t.Fatalf("技能安装请求次数不符: %d", len(params))
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
	if !hasSkill(skills.Items, "skill") {
		t.Fatalf("重启后安装的技能没有出现: %+v", skills.Items)
	}
}

func TestAppServerHookSourceScope(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "hooks.catalog", Result: map[string]any{
				"userRevision": "u1", "projectsRevision": "p1",
				"userDocument":    map[string]any{"hooks": map[string]any{}, "disableAllHooks": false},
				"disableAllHooks": false,
				"sources":         []any{map[string]any{"source": "user", "enabled": true, "state": "ok"}},
				"hooks": []any{
					map[string]any{"id": "user-1", "name": "用户钩子", "event": "pre", "command": "echo user", "source": "user", "enabled": true},
					map[string]any{"id": "project-1", "name": "项目钩子", "event": "pre", "command": "echo project", "source": "project", "enabled": true},
					map[string]any{"id": "plugin-1", "name": "插件钩子", "event": "post", "command": "echo plugin", "source": "plugin", "enabled": true},
				},
			}},
			{Method: "hooks.save", Result: map[string]any{"revision": "u2"}},
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
	if requests := server.RequestsFor("hooks.save"); len(requests) != 0 {
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
	requests := server.RequestsFor("hooks.save")
	if len(requests) != 1 {
		t.Fatalf("用户来源钩子请求次数不符: %d", len(requests))
	}
	var params struct {
		ExpectedRevision string `json:"expectedRevision"`
		Document         struct {
			Hooks []struct {
				Source string `json:"source"`
				Event  string `json:"event"`
			} `json:"hooks"`
		} `json:"document"`
	}
	if err := json.Unmarshal(requests[0].Params, &params); err != nil {
		t.Fatalf("解析钩子写入参数: %v", err)
	}
	if len(params.Document.Hooks) != 1 || params.Document.Hooks[0].Source != "user" || params.Document.Hooks[0].Event != "pre" {
		t.Fatalf("钩子写入参数不符: %+v", params)
	}
}

func TestAppServerToolServersAndImport(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "mcp.catalog", Result: map[string]any{
				"userRevision": "u1", "projectRevision": nil,
				"servers": []any{
					map[string]any{
						"id": "srv-1", "name": "现有服务器", "source": "user", "transport": "stdio",
						"command": "srv-bin", "args": []string{"--flag"}, "enabled": true,
						"auth": map[string]any{"status": "ready", "missingScopes": []string{}},
					},
				},
				"invalid": []any{},
			}},
			{Method: "mcp.test", Result: map[string]any{
				"serverId": "srv-1", "reachable": true,
				"tools": []any{
					map[string]any{"name": "search", "description": "搜索"},
					map[string]any{"name": "fetch"},
				},
			}},
			{Method: "mcp.import.preview", Result: map[string]any{"source": "config-a", "candidates": []any{
				map[string]any{"key": "alpha", "server": map[string]any{"name": "alpha", "transport": "stdio", "command": "alpha-bin"}},
				map[string]any{"key": "beta", "server": map[string]any{"name": "beta", "transport": "http", "url": "https://beta.example"}},
				map[string]any{"key": "gamma", "server": map[string]any{"name": "gamma", "transport": "sse", "url": "https://gamma.example"}},
			}}},
			{Method: "mcp.save", Result: map[string]any{"serverId": "srv-new"}},
			{Method: "mcp.remove", Result: map[string]any{}},
			{Method: "mcp.import.apply", Result: map[string]any{}},
		},
	})
	harness := newAppServerHarness(client)
	service := newService(t, Config{ToolServers: harness, Writer: newWriteManager(t, harness)})
	ctx := context.Background()

	servers, err := service.ListToolServers(ctx)
	if err != nil || len(servers.Items) != 1 {
		t.Fatalf("读取工具服务器: %v %+v", err, servers)
	}
	if servers.Items[0].ID != "srv-1" || servers.Items[0].Source != SourceUser ||
		servers.Items[0].AuthStatus != "ready" {
		t.Fatalf("MCP 条目映射不符: %+v", servers.Items[0])
	}
	if requests := server.RequestsFor("mcp.test"); len(requests) != 0 {
		t.Fatalf("读取清单触发了探测: %d", len(requests))
	}

	added, err := service.AddToolServer(ctx, ToolServerSpec{
		Name: "新服务器", Transport: ToolTransportStdio, Command: "new-bin",
		Args: []string{"--serve"}, Env: map[string]string{"TOKEN": "secret"}, Source: SourceUser,
	})
	if err != nil || added.Status != write.StatusSucceeded {
		t.Fatalf("新增工具服务器: %v %+v", err, added)
	}
	saveParams := methodParams(t, server, "mcp.save")
	if len(saveParams) != 1 || saveParams[0]["id"] != "新服务器" || saveParams[0]["source"] != "user" {
		t.Fatalf("新增没有按 mcp.save 透传: %+v", saveParams)
	}
	if requests := server.RequestsFor("mcp.test"); len(requests) != 0 {
		t.Fatalf("新增触发了探测: %d", len(requests))
	}

	probe, err := service.ProbeToolServer(ctx, "srv-1")
	if err != nil || !probe.Reachable || len(probe.Tools) != 2 {
		t.Fatalf("探测工具服务器: %v %+v", err, probe)
	}
	probeParams := methodParams(t, server, "mcp.test")
	if len(probeParams) != 1 || probeParams[0]["id"] != "srv-1" || probeParams[0]["source"] != "user" {
		t.Fatalf("探测没有按 mcp.test 透传: %+v", probeParams)
	}
	if requests := server.RequestsFor("mcp.save"); len(requests) != 1 {
		t.Fatalf("探测写出了配置: %d", len(requests))
	}
	if snapshot, ok := service.ProbeSnapshot("srv-1"); !ok || len(snapshot.Tools) != 2 {
		t.Fatalf("探测预览没有保留: %+v", snapshot)
	}

	if _, err := service.UpdateToolServer(ctx, "srv-1", ToolServerSpec{
		Name: "改名服务器", Transport: ToolTransportHTTP, URL: "https://srv.example", Source: SourceUser,
	}); err != nil {
		t.Fatalf("修改工具服务器: %v", err)
	}
	if requests := server.RequestsFor("mcp.save"); len(requests) != 2 {
		t.Fatalf("修改请求次数不符: %d", len(requests))
	}

	preview, err := service.PreviewToolServerImport(ctx, "config-a")
	if err != nil || len(preview.Candidates) != 3 {
		t.Fatalf("导入预览: %v %+v", err, preview)
	}
	if requests := server.RequestsFor("mcp.save"); len(requests) != 2 {
		t.Fatalf("预览写出了配置: %d", len(requests))
	}

	if _, err := service.ApplyToolServerImport(ctx, "config-a", []string{"missing"}); !errors.Is(err, ErrUnknownCandidate) {
		t.Fatalf("未知候选没有拒绝: %v", err)
	}
	result, err := service.ApplyToolServerImport(ctx, "config-a", []string{"gamma", "beta"})
	if err != nil || result.Written != 2 {
		t.Fatalf("按勾选写入: %v %+v", err, result)
	}
	applyParams := methodParams(t, server, "mcp.import.apply")
	if len(applyParams) != 2 {
		t.Fatalf("导入写入条数不等于勾选条数: %d", len(applyParams))
	}
	keys := make([]string, 0, len(applyParams))
	for _, params := range applyParams {
		ids, _ := params["ids"].([]any)
		if params["source"] != "config-a" || len(ids) != 1 {
			t.Fatalf("导入写入缺少来源或候选: %+v", params)
		}
		keys = append(keys, ids[0].(string))
	}
	if strings.Join(keys, ",") != "beta,gamma" {
		t.Fatalf("导入只应写入勾选条目: %v", keys)
	}
	if requests := server.RequestsFor("mcp.save"); len(requests) != 2 {
		t.Fatalf("导入绕过了 mcp.import.apply: %d", len(requests))
	}

	if _, err := service.RemoveToolServer(ctx, "srv-1"); err != nil {
		t.Fatalf("删除工具服务器: %v", err)
	}
	if requests := server.RequestsFor("mcp.remove"); len(requests) != 1 {
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
		Faults: []fakeappserver.Fault{{Method: "schedule.list", Occurrence: 1, UnknownMethod: true}},
		Replies: []fakeappserver.Reply{
			{Method: "schedule.list", Result: map[string]any{"jobs": []any{
				map[string]any{
					"id": "task-1", "name": "夜间构建", "schedule": map[string]any{"every_minutes": 60},
					"prompt": "构建", "cwd": "/work", "mode": "create", "enabled": true, "revision": int64(1),
				},
				map[string]any{
					"id": "task-2", "name": "每周清理", "schedule": map[string]any{"cron": "0 3 * * 0"},
					"prompt": "清理", "cwd": "/work", "mode": "create", "enabled": false, "revision": int64(1),
				},
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
	if requests := server.RequestsFor("schedule.list"); len(requests) != 2 {
		t.Fatalf("能力恢复请求次数不符: %d", len(requests))
	}

	arrived := attachNotices(t, client, service)
	if err := server.Notify("management.progress", map[string]any{
		"id": 1, "method": "schedule.watch", "data": map[string]any{"kind": "changed"},
	}); err != nil {
		t.Fatalf("推送计划任务通知: %v", err)
	}
	<-arrived
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("等待计划任务跟随: %v", err)
	}
	if requests := server.RequestsFor("schedule.list"); len(requests) != 3 {
		t.Fatalf("通知后重读次数不符: %d", len(requests))
	}
	time.Sleep(100 * time.Millisecond)
	if requests := server.RequestsFor("schedule.list"); len(requests) != 3 {
		t.Fatalf("计划任务出现定时轮询: %d", len(requests))
	}
	if stats := service.FollowStats(); stats.Reads[TopicSchedule] != 1 {
		t.Fatalf("计划任务跟随统计不符: %+v", stats)
	}
}

func TestAppServerWakeupFollowAndCancel(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "session.wakeup.list", Result: map[string]any{"wakeups": []any{
				map[string]any{
					"wakeupId": "wake-1", "at": "2026-11-11T10:00:00Z", "prompt": "继续", "state": "scheduled",
				},
			}}},
			{Method: "session.wakeup.cancel", Result: map[string]any{}},
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

	view, err := service.CancelWakeup(ctx, WakeupRef{SessionID: "s-1", BranchID: "main", WakeupID: "wake-1"})
	if err != nil || view.Status != write.StatusSucceeded {
		t.Fatalf("取消唤醒: %v %+v", err, view)
	}
	if requests := server.RequestsFor("session.wakeup.cancel"); len(requests) != 1 {
		t.Fatalf("取消请求次数不符: %d", len(requests))
	}
	if requests := server.RequestsFor("session.wakeup.list"); len(requests) != 1 {
		t.Fatalf("取消本身不应重读: %d", len(requests))
	}

	if err := server.Notify("session.notice", map[string]any{
		"sessionId": "s-1", "source": "Scheduler", "kind": "wakeup",
		"data": map[string]any{"branchId": "main", "wakeupId": "wake-1", "state": "cancelled"},
	}); err != nil {
		t.Fatalf("推送唤醒通知: %v", err)
	}
	<-arrived
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("等待唤醒跟随: %v", err)
	}
	if requests := server.RequestsFor("session.wakeup.list"); len(requests) != 2 {
		t.Fatalf("通知后重读次数不符: %d", len(requests))
	}
	if snapshot := service.WakeupSnapshot(scope); len(snapshot.Items) != 0 {
		t.Fatalf("取消后唤醒记录仍在: %+v", snapshot)
	}
	time.Sleep(100 * time.Millisecond)
	if requests := server.RequestsFor("session.wakeup.list"); len(requests) != 2 {
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
	if requests := server.RequestsFor("session.wakeup.list"); len(requests) != 3 {
		t.Fatalf("再次读取请求次数不符: %d", len(requests))
	}
}

// TestAlpha21WorkflowCapabilityMissingMakesNoRequest 核对 alpha.21 未注册 workflow.*
// 时的边界：类型保留，读取与启动/停止都报能力缺失，且不发出任何 wire 请求。
func TestAlpha21WorkflowCapabilityMissingMakesNoRequest(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{})
	harness := newAppServerHarness(client)
	manager := newWriteManager(t, harness)
	service := newService(t, Config{Writer: manager})
	ctx := context.Background()

	if _, err := service.ListWorkflows(ctx); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("浏览工作流没有报能力缺失: %v", err)
	}
	if _, err := service.ReadWorkflow(ctx, "wf-1"); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("读取工作流定义没有报能力缺失: %v", err)
	}
	if _, err := service.ListWorkflowRuns(ctx, "wf-1"); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("读取运行状态没有报能力缺失: %v", err)
	}
	if _, err := service.StartWorkflow(ctx, "wf-1", nil); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("启动工作流没有报能力缺失: %v", err)
	}
	if _, err := service.StopWorkflow(ctx, "run-1"); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("停止工作流没有报能力缺失: %v", err)
	}

	for _, method := range server.Methods() {
		if strings.HasPrefix(method, "workflow.") {
			t.Fatalf("workflow 能力缺失时仍发出了请求: %s", method)
		}
	}
	if table := manager.Table(); len(table) != 0 {
		t.Fatalf("workflow 能力缺失时登记了写意图: %+v", table)
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
