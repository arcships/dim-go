package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// wireHarness 是「连接 + 写登记」集成形状的测试适配器：设置、模型、存储、
// 运行组件与模态能力都走真实 NDJSON；wire 形状只出现在这里，本包不复制它们。
type wireHarness struct {
	client   *protocol.Client
	features Features

	mu                sync.Mutex
	selectionOverride *Selection
}

// setSelectionOverride 模拟服务端在选择写入成功后的新事实；假服务端的脚本
// 回执是固定的，不能在两次读取之间变化。
func (h *wireHarness) setSelectionOverride(selection *Selection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if selection == nil {
		h.selectionOverride = nil
		return
	}
	copied := *selection
	h.selectionOverride = &copied
}

func (h *wireHarness) currentSelectionOverride() *Selection {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.selectionOverride == nil {
		return nil
	}
	copied := *h.selectionOverride
	return &copied
}

// HasFeature 把假服务端 hello 回执里的能力声明转给本包。
func (h *wireHarness) HasFeature(name string) bool {
	return h.features != nil && h.features.HasFeature(name)
}

func scopeWireValue(scope Scope) any {
	if scope.Kind == ScopeProject {
		return map[string]any{"project": scope.Project}
	}
	return "user"
}

func (h *wireHarness) ReadSettingsSources(ctx context.Context) (Sources, error) {
	var reply struct {
		Dialect string `json:"dialect"`
		Sources []struct {
			Source    string          `json:"source"`
			Schema    json.RawMessage `json:"schema"`
			Scopes    []string        `json:"scopes"`
			UserOnly  []string        `json:"userOnly"`
			WriteOnly bool            `json:"writeOnly"`
		} `json:"sources"`
	}
	if err := h.client.CallInto(ctx, "settings.sources", nil, &reply); err != nil {
		return Sources{}, err
	}
	sources := Sources{Dialect: reply.Dialect, Sources: make([]Source, 0, len(reply.Sources))}
	for _, source := range reply.Sources {
		scopes := make([]ScopeKind, 0, len(source.Scopes))
		for _, scope := range source.Scopes {
			scopes = append(scopes, ScopeKind(scope))
		}
		sources.Sources = append(sources.Sources, Source{
			Name:      source.Source,
			Scopes:    scopes,
			UserOnly:  source.UserOnly,
			WriteOnly: source.WriteOnly,
			Schema:    source.Schema,
		})
	}
	return sources, nil
}

func (h *wireHarness) ReadSettingsProjects(ctx context.Context) ([]Project, error) {
	var reply struct {
		Projects []struct {
			Root int    `json:"root"`
			CWD  string `json:"cwd"`
		} `json:"projects"`
	}
	if err := h.client.CallInto(ctx, "settings.projects", nil, &reply); err != nil {
		return nil, err
	}
	projects := make([]Project, 0, len(reply.Projects))
	for _, project := range reply.Projects {
		projects = append(projects, Project{Root: project.Root, CWD: project.CWD})
	}
	return projects, nil
}

func (h *wireHarness) ReadSettings(ctx context.Context, source string, scope Scope) (Document, error) {
	var reply struct {
		Revision string          `json:"revision"`
		Document json.RawMessage `json:"document"`
	}
	if err := h.client.CallInto(ctx, "settings.read", map[string]any{
		"source": source,
		"scope":  scopeWireValue(scope),
	}, &reply); err != nil {
		return Document{}, err
	}
	return Document{Source: source, Scope: scope, Revision: reply.Revision, Document: reply.Document}, nil
}

func (h *wireHarness) ReadModelCatalog(ctx context.Context) (ModelCatalog, error) {
	var reply struct {
		Providers []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Enabled     *bool  `json:"enabled"`
			Models      []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
			} `json:"models"`
			EnabledModelIDs []string `json:"enabledModelIds"`
		} `json:"providers"`
		Selection *struct {
			ProviderID      string          `json:"providerId"`
			ModelID         string          `json:"modelId"`
			Version         json.RawMessage `json:"version"`
			ReasoningEffort string          `json:"reasoningEffort"`
			ServiceTier     string          `json:"serviceTier"`
		} `json:"selection"`
	}
	if err := h.client.CallInto(ctx, "models.providers.list", nil, &reply); err != nil {
		return ModelCatalog{}, err
	}
	catalog := ModelCatalog{Providers: make([]Provider, 0, len(reply.Providers))}
	for _, provider := range reply.Providers {
		if provider.Enabled == nil {
			return ModelCatalog{}, errors.New("provider 缺少 enabled")
		}
		name := provider.Name
		if name == "" {
			name = provider.DisplayName
		}
		models := make([]ProviderModel, 0, len(provider.Models))
		for _, model := range provider.Models {
			modelName := model.Name
			if modelName == "" {
				modelName = model.DisplayName
			}
			models = append(models, ProviderModel{ID: model.ID, Name: modelName})
		}
		catalog.Providers = append(catalog.Providers, Provider{
			ID:              provider.ID,
			Name:            name,
			Enabled:         *provider.Enabled,
			Models:          models,
			EnabledModelIDs: provider.EnabledModelIDs,
		})
	}
	if reply.Selection != nil {
		catalog.Selection = &Selection{
			ProviderID:      reply.Selection.ProviderID,
			ModelID:         reply.Selection.ModelID,
			Revision:        wireRevision(reply.Selection.Version),
			ReasoningEffort: reply.Selection.ReasoningEffort,
			ServiceTier:     reply.Selection.ServiceTier,
		}
	}
	if override := h.currentSelectionOverride(); override != nil {
		catalog.Selection = override
	}
	return catalog, nil
}

// wireRevision 把服务端的选择版本（字符串或数字）映射成领域里的不透明版本。
func wireRevision(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	return string(raw)
}

func (h *wireHarness) ReadStorage(ctx context.Context) (StorageReading, error) {
	var reply struct {
		DatabaseBytes    *int64 `json:"databaseBytes"`
		WALBytes         *int64 `json:"walBytes"`
		ReclaimableBytes *int64 `json:"reclaimableBytes"`
	}
	if err := h.client.CallInto(ctx, "storage.report", nil, &reply); err != nil {
		return StorageReading{}, err
	}
	return StorageReading{
		DatabaseBytes:    reply.DatabaseBytes,
		WALBytes:         reply.WALBytes,
		ReclaimableBytes: reply.ReclaimableBytes,
	}, nil
}

func (h *wireHarness) ReadRuntimes(ctx context.Context) ([]RuntimeEntry, error) {
	var reply []struct {
		Name    string `json:"name"`
		Source  string `json:"source"`
		Auto    *bool  `json:"auto"`
		Version string `json:"version"`
		Path    string `json:"path"`
		Status  string `json:"status"`
		Reason  string `json:"reason"`
	}
	if err := h.client.CallInto(ctx, "runtime.list", nil, &reply); err != nil {
		return nil, err
	}
	entries := make([]RuntimeEntry, 0, len(reply))
	for _, entry := range reply {
		if entry.Auto == nil {
			return nil, errors.New("运行组件缺少 auto")
		}
		entries = append(entries, RuntimeEntry{
			Name:    entry.Name,
			Source:  RuntimeSource(entry.Source),
			Auto:    *entry.Auto,
			Version: entry.Version,
			Path:    entry.Path,
			Status:  RuntimeStatus(entry.Status),
			Reason:  entry.Reason,
		})
	}
	return entries, nil
}

type wireReason struct {
	Dimension string `json:"dimension"`
	Detail    string `json:"detail"`
}

type wireModalityOption struct {
	ProviderID string      `json:"providerId"`
	ModelID    string      `json:"modelId"`
	Available  bool        `json:"available"`
	Reason     *wireReason `json:"reason"`
}

type wireModalityOperation struct {
	CapabilityID string `json:"capabilityId"`
	Category     string `json:"category"`
	Operation    string `json:"operation"`
	Key          string `json:"key"`
	Available    bool   `json:"available"`
	Reasons      []wireReason
	Default      *wireModalityOption `json:"default"`
	Effective    *struct {
		ProviderID string `json:"providerId"`
		ModelID    string `json:"modelId"`
		Source     string `json:"source"`
	} `json:"effective"`
	Options []wireModalityOption `json:"options"`
}

func (h *wireHarness) ReadModalityCapabilities(ctx context.Context) (ModalityCapabilities, error) {
	var reply struct {
		Supported         bool  `json:"supported"`
		CanConfigure      bool  `json:"canConfigure"`
		TaskSupported     *bool `json:"taskSupported"`
		RealtimeSupported *bool `json:"realtimeSupported"`
		Revision          *string
		Categories        []struct {
			Category   string                  `json:"category"`
			Available  bool                    `json:"available"`
			Reasons    []wireReason            `json:"reasons"`
			Operations []wireModalityOperation `json:"operations"`
		} `json:"categories"`
		UnavailableReason string `json:"unavailableReason"`
	}
	if err := h.client.CallInto(ctx, "modality.capabilities", nil, &reply); err != nil {
		return ModalityCapabilities{}, err
	}
	caps := ModalityCapabilities{
		Supported:         reply.Supported,
		CanConfigure:      reply.CanConfigure,
		TaskSupported:     reply.TaskSupported,
		RealtimeSupported: reply.RealtimeSupported,
		Revision:          reply.Revision,
		UnavailableReason: reply.UnavailableReason,
	}
	for _, category := range reply.Categories {
		mapped := ModalityCategory{
			Category:   category.Category,
			Available:  category.Available,
			Reasons:    mapReasons(category.Reasons),
			Operations: make([]ModalityOperation, 0, len(category.Operations)),
		}
		for _, operation := range category.Operations {
			item := ModalityOperation{
				CapabilityID: operation.CapabilityID,
				Category:     operation.Category,
				Operation:    operation.Operation,
				Key:          operation.Key,
				Available:    operation.Available,
				Reasons:      mapReasons(operation.Reasons),
				Options:      make([]ModalityOption, 0, len(operation.Options)),
			}
			if operation.Default != nil {
				option := mapOption(*operation.Default)
				item.Default = &option
			}
			if operation.Effective != nil {
				item.Effective = &ModalityModel{
					ProviderID: operation.Effective.ProviderID,
					ModelID:    operation.Effective.ModelID,
					Source:     operation.Effective.Source,
				}
			}
			for _, option := range operation.Options {
				item.Options = append(item.Options, mapOption(option))
			}
			mapped.Operations = append(mapped.Operations, item)
		}
		caps.Categories = append(caps.Categories, mapped)
	}
	return caps, nil
}

func mapReasons(reasons []wireReason) []ModalityReason {
	mapped := make([]ModalityReason, 0, len(reasons))
	for _, reason := range reasons {
		mapped = append(mapped, ModalityReason{Dimension: reason.Dimension, Detail: reason.Detail})
	}
	return mapped
}

func mapOption(option wireModalityOption) ModalityOption {
	mapped := ModalityOption{
		ProviderID: option.ProviderID,
		ModelID:    option.ModelID,
		Available:  option.Available,
	}
	if option.Reason != nil {
		mapped.Reason = &ModalityReason{Dimension: option.Reason.Dimension, Detail: option.Reason.Detail}
	}
	return mapped
}

// Send 是写登记的发送方：管理写入按透明参数里的操作名映射到服务端方法，只发一次。
func (h *wireHarness) Send(ctx context.Context, req write.Request) (write.Reply, error) {
	if req.Kind != write.KindManagementWrite {
		return write.Reply{}, errors.New("未支持的写意图: " + string(req.Kind))
	}
	var payload struct {
		Op               string          `json:"op"`
		Name             string          `json:"name"`
		ProviderID       string          `json:"providerId"`
		ModelID          string          `json:"modelId"`
		Source           string          `json:"source"`
		Scope            Scope           `json:"scope"`
		ExpectedRevision string          `json:"expectedRevision"`
		Document         json.RawMessage `json:"document"`
	}
	if err := json.Unmarshal(req.Params, &payload); err != nil {
		return write.Reply{}, err
	}
	switch payload.Op {
	case "settings.update":
		raw, err := h.client.Call(ctx, "settings.update", map[string]any{
			"source":           payload.Source,
			"scope":            scopeWireValue(payload.Scope),
			"expectedRevision": payload.ExpectedRevision,
			"document":         payload.Document,
		})
		if err != nil {
			return write.Reply{}, err
		}
		return write.Reply{Result: raw}, nil
	case "provider.select":
		raw, err := h.client.Call(ctx, "provider.select", map[string]any{
			"providerId":      payload.ProviderID,
			"modelId":         payload.ModelID,
			"expectedVersion": payload.ExpectedRevision,
		})
		if err != nil {
			return write.Reply{}, err
		}
		h.setSelectionOverride(&Selection{
			ProviderID: payload.ProviderID,
			ModelID:    payload.ModelID,
			Revision:   "sel-2",
		})
		return write.Reply{Result: raw}, nil
	case "runtime.install":
		raw, err := h.client.Call(ctx, "runtime.install", map[string]any{"name": payload.Name})
		if err != nil {
			return write.Reply{}, err
		}
		return write.Reply{Result: raw}, nil
	default:
		return write.Reply{}, errors.New("未支持的管理写入操作: " + payload.Op)
	}
}

// accountWire 只实现账号领域需要的状态与模型读取。
type accountWire struct {
	client *protocol.Client
}

func (h *accountWire) ReadAccount(ctx context.Context) (account.Fact, error) {
	var reply struct {
		Status   string `json:"status"`
		Identity struct {
			AccountID   string `json:"accountId"`
			Email       string `json:"email"`
			DisplayName string `json:"displayName"`
			Plan        string `json:"plan"`
		} `json:"identity"`
	}
	if err := h.client.CallInto(ctx, "account.status", nil, &reply); err != nil {
		return account.Fact{}, err
	}
	return account.Fact{
		Status: account.Status(reply.Status),
		Identity: account.Identity{
			AccountID:   reply.Identity.AccountID,
			Email:       reply.Identity.Email,
			DisplayName: reply.Identity.DisplayName,
			Plan:        reply.Identity.Plan,
		},
	}, nil
}

func (h *accountWire) ReadModels(ctx context.Context) ([]account.Model, error) {
	var reply struct {
		Models []struct {
			ID              string `json:"id"`
			DisplayName     string `json:"displayName"`
			RequiresAccount bool   `json:"requiresAccount"`
			Unavailable     bool   `json:"unavailable"`
		} `json:"models"`
	}
	if err := h.client.CallInto(ctx, "model.list", nil, &reply); err != nil {
		return nil, err
	}
	models := make([]account.Model, 0, len(reply.Models))
	for _, model := range reply.Models {
		models = append(models, account.Model{
			ID:              model.ID,
			DisplayName:     model.DisplayName,
			RequiresAccount: model.RequiresAccount,
			Unavailable:     model.Unavailable,
		})
	}
	return models, nil
}

type wireInspector struct{}

func (wireInspector) LookupInput(context.Context, write.InputQuery) (write.InputState, error) {
	return write.InputState{}, nil
}

func allWireFeatures() []string {
	return []string{
		FeatureSettingsSources,
		FeatureSettingsProjects,
		FeatureSettingsRead,
		FeatureSettingsUpdate,
		FeatureModelProviders,
		FeatureModelSelection,
		FeatureProviderList,
		FeatureProviderManagement,
		FeatureProviderCreate,
		FeatureProviderUpdate,
		FeatureProviderProbe,
		FeatureProviderModelsSetEnabled,
		FeatureProviderModelsEdit,
		FeatureProviderModelsReset,
		FeatureProviderPreferencesUpdate,
		FeatureCollaborationPolicy,
		FeatureCollaborationSetPolicy,
		FeatureAgentList,
		FeatureSessionAgentGet,
		FeatureSessionAgentUpdate,
		FeatureStorageReport,
		FeatureRuntimeList,
		FeatureRuntimeInstall,
		FeatureModalityCapabilities,
	}
}

// startSettingsServer 启动假服务端，并建好连接与写登记。
func startSettingsServer(t *testing.T, script fakeappserver.Script) (*fakeappserver.Server, *wireHarness, *write.Manager) {
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
	connection, err := supervisor.Connect(context.Background())
	if err != nil {
		supervisor.Close(nil)
		_ = server.Close()
		t.Fatalf("连接假服务端: %v", err)
	}
	harness := &wireHarness{client: connection.Client, features: connection}
	manager, err := write.New(write.Config{
		Sender:      harness,
		Inspector:   wireInspector{},
		SendTimeout: 2 * time.Second,
		Verify: write.VerifyPolicy{
			MaxAttempts: 1,
			Interval:    time.Millisecond,
			Timeout:     500 * time.Millisecond,
		},
	})
	if err != nil {
		supervisor.Close(nil)
		_ = server.Close()
		t.Fatalf("创建写登记: %v", err)
	}
	t.Cleanup(func() {
		manager.Close()
		supervisor.Close(nil)
		_ = server.Close()
	})
	return server, harness, manager
}

func settingsSourcesReply() map[string]any {
	return map[string]any{
		"dialect": "json",
		"sources": []any{
			map[string]any{
				"source":    "execution",
				"schema":    map[string]any{"type": "object"},
				"scopes":    []any{"user", "project"},
				"userOnly":  []any{"font"},
				"writeOnly": false,
			},
			map[string]any{
				"source": "plugins",
				"scopes": []any{"user"},
			},
		},
	}
}

// TestAppServerSettingsProjectScopeAndRevision 用假服务端覆盖范围 1：来源清单、
// 项目作用域读取、写入一次与版本冲突（本地缓存发现冲突时不发请求）。
func TestAppServerSettingsProjectScopeAndRevision(t *testing.T) {
	server, harness, writer := startSettingsServer(t, fakeappserver.Script{
		Features: allWireFeatures(),
		Replies: []fakeappserver.Reply{
			{Method: "settings.sources", Result: settingsSourcesReply()},
			{Method: "settings.projects", Result: map[string]any{
				"projects": []any{
					map[string]any{"root": 7, "cwd": "/workspace/project"},
				},
			}},
			{Method: "settings.read", Result: map[string]any{
				"revision": "r1",
				"document": map[string]any{"sandbox": true},
			}},
			{Method: "settings.update", Result: map[string]any{
				"revision": "r2",
				"document": map[string]any{"sandbox": false},
			}},
		},
	})

	manager := mustNewManager(t, Config{
		Features: connectedFeatures(server, harness),
		Sources:  harness,
		Projects: harness,
		Settings: harness,
		Writer:   writer,
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	if _, err := manager.RefreshProjects(context.Background()); err != nil {
		t.Fatalf("读取项目根: %v", err)
	}
	project, err := manager.ResolveProject(context.Background(), "/workspace/project")
	if err != nil {
		t.Fatalf("解析项目: %v", err)
	}
	if project.Root != 7 {
		t.Fatalf("项目根不符：%+v", project)
	}

	userDocument, err := manager.Read(context.Background(), "execution", UserScope())
	if err != nil {
		t.Fatalf("读取用户设置: %v", err)
	}
	if userDocument.Revision != "r1" {
		t.Fatalf("用户设置版本不符：%+v", userDocument)
	}
	projectDocument, err := manager.Read(context.Background(), "execution", ProjectScope(project.Root))
	if err != nil {
		t.Fatalf("读取项目设置: %v", err)
	}
	if projectDocument.Scope.Project != 7 {
		t.Fatalf("项目设置作用域不符：%+v", projectDocument)
	}
	readRequests := server.RequestsFor("settings.read")
	if len(readRequests) != 2 {
		t.Fatalf("设置读取次数不符：%d", len(readRequests))
	}
	var projectParams map[string]any
	if err := json.Unmarshal(readRequests[1].Params, &projectParams); err != nil {
		t.Fatalf("项目读取参数无法解析: %v", err)
	}
	projectScope, _ := projectParams["scope"].(map[string]any)
	if projectScope["project"] != float64(7) {
		t.Fatalf("项目作用域参数不符：%+v", projectParams)
	}

	updated, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            UserScope(),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{"sandbox":false}`),
	})
	if err != nil {
		t.Fatalf("写入设置: %v", err)
	}
	if updated.Revision != "r2" {
		t.Fatalf("写入后版本不符：%+v", updated)
	}
	if count := len(server.RequestsFor("settings.update")); count != 1 {
		t.Fatalf("写入请求次数不符：%d", count)
	}
	updateRequest := server.RequestsFor("settings.update")[0]
	var params map[string]any
	if err := json.Unmarshal(updateRequest.Params, &params); err != nil {
		t.Fatalf("写入参数无法解析: %v", err)
	}
	if params["source"] != "execution" || params["expectedRevision"] != "r1" || params["scope"] != "user" {
		t.Fatalf("写入参数不符：%+v", params)
	}
	if document, _ := params["document"].(map[string]any); document["sandbox"] != false {
		t.Fatalf("写入文档不符：%+v", params["document"])
	}

	// 再写一次旧版本：本地缓存已是 r2，冲突在发出请求前就被拒绝。
	if _, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            UserScope(),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{"sandbox":true}`),
	}); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("旧版本写入应冲突：%v", err)
	}
	if count := len(server.RequestsFor("settings.update")); count != 1 {
		t.Fatalf("冲突不应再发写入请求：%d", count)
	}

	// 项目作用域写只允许用户作用域的键：客户端先行拒绝，不发请求。
	if _, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            ProjectScope(7),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{"font":"mono"}`),
	}); !errors.Is(err, ErrUserOnlySetting) {
		t.Fatalf("项目作用域写只允许用户作用域的键应拒绝：%v", err)
	}
	if count := len(server.RequestsFor("settings.update")); count != 1 {
		t.Fatalf("前置条件拒绝不应发请求：%d", count)
	}

	table := writer.Table()
	if len(table) != 1 || table[0].Kind != write.KindManagementWrite {
		t.Fatalf("设置写入应经管理写入类型：%+v", table)
	}
}

// TestAppServerSettingsServerConflictNotOverwritten 用假服务端覆盖服务端版本
// 冲突：写入被拒绝后只读一次当前文档判断冲突，不重发、不覆盖。
func TestAppServerSettingsServerConflictNotOverwritten(t *testing.T) {
	server, harness, writer := startSettingsServer(t, fakeappserver.Script{
		Features: allWireFeatures(),
		Replies: []fakeappserver.Reply{
			{Method: "settings.sources", Result: map[string]any{
				"dialect": "json",
				"sources": []any{
					map[string]any{"source": "execution", "scopes": []any{"user"}},
				},
			}},
			{Method: "settings.read", Result: map[string]any{
				"revision": "r2",
				"document": map[string]any{"sandbox": true},
			}},
			{Method: "settings.update", Error: fakeappserver.AppError(-32000, "revision_conflict", "设置版本已变")},
		},
	})

	manager := mustNewManager(t, Config{
		Features: connectedFeatures(server, harness),
		Sources:  harness,
		Settings: harness,
		Writer:   writer,
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}

	if _, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            UserScope(),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{"sandbox":false}`),
	}); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("服务端版本冲突应返回冲突：%v", err)
	}
	if count := len(server.RequestsFor("settings.update")); count != 1 {
		t.Fatalf("冲突不应重发写入：%d", count)
	}
	if count := len(server.RequestsFor("settings.read")); count != 1 {
		t.Fatalf("冲突判定只应读一次当前文档：%d", count)
	}
	cached, ok := manager.Cached("execution", UserScope())
	if !ok || cached.Revision != "r2" || !strings.Contains(string(cached.Document), `"sandbox":true`) {
		t.Fatalf("冲突后应采用服务端当前文档，而不是写入内容：%+v", cached)
	}
}

// TestAppServerModelsFilteredByAccount 用假服务端覆盖范围 2：provider 与模型
// 目录、当前选择，以及按账号状态过滤的可用性。
func TestAppServerModelsFilteredByAccount(t *testing.T) {
	server, harness, _ := startSettingsServer(t, fakeappserver.Script{
		Features: allWireFeatures(),
		Replies: []fakeappserver.Reply{
			{Method: "models.providers.list", Result: map[string]any{
				"providers": []any{
					map[string]any{
						"id":      "local",
						"name":    "本地",
						"enabled": true,
						"models": []any{
							map[string]any{"id": "m1", "displayName": "一号"},
							map[string]any{"id": "m2", "displayName": "二号"},
						},
						"enabledModelIds": []any{"m1", "m2"},
					},
					map[string]any{
						"id":      "cloud",
						"name":    "云端",
						"enabled": true,
						"models": []any{
							map[string]any{"id": "c1", "displayName": "云端一号"},
						},
						"enabledModelIds": []any{"c1"},
					},
				},
				"selection": map[string]any{
					"providerId":      "cloud",
					"modelId":         "c1",
					"reasoningEffort": "high",
				},
			}},
			{Method: "account.status", Result: map[string]any{
				"status": "signed_in",
				"identity": map[string]any{
					"accountId": "acct-wire",
					"email":     "wire@example.com",
				},
			}},
			{Method: "model.list", Result: map[string]any{
				"models": []any{
					map[string]any{"id": "m1"},
					map[string]any{"id": "c1"},
				},
			}},
		},
	})

	accountWire := &accountWire{client: harness.client}
	acct, err := account.New(account.Config{Status: accountWire, Models: accountWire})
	if err != nil {
		t.Fatalf("创建账号: %v", err)
	}
	if _, err := acct.Refresh(context.Background()); err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}
	if _, err := acct.RefreshModels(context.Background()); err != nil {
		t.Fatalf("读取账号模型: %v", err)
	}

	manager := mustNewManager(t, Config{
		Features: connectedFeatures(server, harness),
		Models:   harness,
		Account:  acct,
	})
	snapshot, err := manager.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型目录: %v", err)
	}
	selectable := map[string]bool{}
	for _, provider := range snapshot.Providers {
		for _, model := range provider.Models {
			selectable[provider.ID+"/"+model.ID] = model.Selectable
		}
	}
	if !selectable["local/m1"] || selectable["local/m2"] || !selectable["cloud/c1"] {
		t.Fatalf("按账号过滤的可用性不符：%+v", selectable)
	}
	if snapshot.Selection == nil || !snapshot.Selection.Known || !snapshot.Selection.Selectable {
		t.Fatalf("当前选择应可选：%+v", snapshot.Selection)
	}
	if snapshot.Selection.ReasoningEffort != "high" {
		t.Fatalf("当前选择偏好不符：%+v", snapshot.Selection)
	}
	if count := len(server.RequestsFor("models.providers.list")); count != 1 {
		t.Fatalf("模型目录读取次数不符：%d", count)
	}
}

// TestAppServerModelsSignedOut 用假服务端覆盖未登录时的可用性过滤：模型列表
// 仍可读，但没有任何模型可选，并带原因。
func accountModels(snapshot ModelSnapshot) []ModelView {
	var models []ModelView
	for _, provider := range snapshot.Providers {
		for _, model := range provider.Models {
			if model.Selectable {
				models = append(models, model)
			}
		}
	}
	return models
}

func TestAppServerModelsSignedOut(t *testing.T) {
	server, harness, _ := startSettingsServer(t, fakeappserver.Script{
		Features: allWireFeatures(),
		Replies: []fakeappserver.Reply{
			{Method: "models.providers.list", Result: map[string]any{
				"providers": []any{
					map[string]any{
						"id":      "cloud",
						"name":    "云端",
						"enabled": true,
						"models": []any{
							map[string]any{"id": "c1", "displayName": "云端一号"},
						},
						"enabledModelIds": []any{"c1"},
					},
				},
				"selection": nil,
			}},
			{Method: "account.status", Result: map[string]any{"status": "signed_out"}},
			{Method: "model.list", Result: map[string]any{
				"models": []any{map[string]any{"id": "c1"}},
			}},
		},
	})

	accountWire := &accountWire{client: harness.client}
	acct, err := account.New(account.Config{Status: accountWire, Models: accountWire})
	if err != nil {
		t.Fatalf("创建账号: %v", err)
	}
	if _, err := acct.Refresh(context.Background()); err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}
	if _, err := acct.RefreshModels(context.Background()); err != nil {
		t.Fatalf("读取账号模型: %v", err)
	}

	manager := mustNewManager(t, Config{
		Features: connectedFeatures(server, harness),
		Models:   harness,
		Account:  acct,
	})
	snapshot, err := manager.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型目录: %v", err)
	}
	if len(accountModels(snapshot)) != 0 {
		t.Fatalf("未登录时可用模型应为空：%+v", accountModels(snapshot))
	}
	if !strings.Contains(snapshot.Reason, "未登录") {
		t.Fatalf("未登录原因不符：%q", snapshot.Reason)
	}
	for _, provider := range snapshot.Providers {
		for _, model := range provider.Models {
			if model.Selectable {
				t.Fatalf("未登录时不能有可选模型：%s/%s", provider.ID, model.ID)
			}
			if model.Reason == "" {
				t.Fatalf("不可选应说明原因：%s/%s", provider.ID, model.ID)
			}
		}
	}
}

// TestAppServerStorageReportIsReadOnly 用假服务端覆盖范围 3：只读存储读数，
// 且不发出任何删除或清理动作；缺能力时返回分类错误。
func TestAppServerStorageReportIsReadOnly(t *testing.T) {
	server, harness, _ := startSettingsServer(t, fakeappserver.Script{
		Features: allWireFeatures(),
		Replies: []fakeappserver.Reply{
			{Method: "storage.report", Result: map[string]any{
				"databaseBytes":    int64(4096),
				"walBytes":         int64(512),
				"reclaimableBytes": int64(128),
			}},
		},
	})

	manager := mustNewManager(t, Config{
		Features: connectedFeatures(server, harness),
		Storage:  harness,
	})
	report, err := manager.RefreshStorage(context.Background())
	if err != nil {
		t.Fatalf("读取存储: %v", err)
	}
	if report.DatabaseBytes != 4096 || report.WALBytes != 512 || report.ReclaimableBytes != 128 {
		t.Fatalf("存储读数不符：%+v", report)
	}
	if count := len(server.RequestsFor("storage.report")); count != 1 {
		t.Fatalf("存储读数请求次数不符：%d", count)
	}
	for _, method := range server.Methods() {
		lower := strings.ToLower(method)
		if strings.Contains(lower, "delete") || strings.Contains(lower, "cleanup") ||
			strings.Contains(lower, "vacuum") || strings.Contains(lower, "remove") {
			t.Fatalf("存储读数不应触发删除类动作：%q", method)
		}
	}

	// 缺能力：不调用存储读数，直接返回分类错误。
	noFeatureServer, noFeatureHarness, _ := startSettingsServer(t, fakeappserver.Script{
		Features: []string{},
	})
	noFeature := mustNewManager(t, Config{
		Features: connectedFeatures(noFeatureServer, noFeatureHarness),
		Storage:  noFeatureHarness,
	})
	if _, err := noFeature.RefreshStorage(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺存储读数能力应分类：%v", err)
	}
	if count := len(noFeatureServer.RequestsFor("storage.report")); count != 0 {
		t.Fatalf("缺能力时不应发出请求：%d", count)
	}
}

// TestAppServerRuntimeInstallRelay 用假服务端覆盖范围 4：运行组件状态读取与
// 安装转达；安装经写登记只发一次，成功后重读状态。
func TestAppServerRuntimeInstallRelay(t *testing.T) {
	server, harness, writer := startSettingsServer(t, fakeappserver.Script{
		Features: allWireFeatures(),
		Replies: []fakeappserver.Reply{
			{Method: "runtime.list", Result: []any{
				map[string]any{"name": "node", "source": "download", "auto": false, "status": "off"},
				map[string]any{
					"name": "uv", "source": "download", "auto": true,
					"version": "0.4.0", "path": "/rt/uv", "status": "ready",
				},
			}},
			{Method: "runtime.install", Result: map[string]any{
				"name": "uv", "source": "download", "auto": true,
				"version": "0.4.0", "status": "ready",
			}},
		},
	})

	manager := mustNewManager(t, Config{
		Features: connectedFeatures(server, harness),
		Runtimes: harness,
		Writer:   writer,
	})
	entries, err := manager.RefreshRuntimes(context.Background())
	if err != nil {
		t.Fatalf("读取运行组件: %v", err)
	}
	if len(entries) != 2 || entries[1].Status != RuntimeStatusReady {
		t.Fatalf("运行组件清单不符：%+v", entries)
	}

	entry, err := manager.InstallRuntime(context.Background(), "uv")
	if err != nil {
		t.Fatalf("安装运行组件: %v", err)
	}
	if entry.Status != RuntimeStatusReady || entry.Version != "0.4.0" {
		t.Fatalf("安装后状态不符：%+v", entry)
	}
	if count := len(server.RequestsFor("runtime.install")); count != 1 {
		t.Fatalf("安装请求次数不符：%d", count)
	}
	if count := len(server.RequestsFor("runtime.list")); count != 2 {
		t.Fatalf("安装成功后应重读一次清单：%d", count)
	}
	var params map[string]any
	if err := json.Unmarshal(server.RequestsFor("runtime.install")[0].Params, &params); err != nil {
		t.Fatalf("安装参数无法解析: %v", err)
	}
	if params["name"] != "uv" {
		t.Fatalf("安装参数不符：%+v", params)
	}
	table := writer.Table()
	if len(table) != 1 || table[0].Kind != write.KindManagementWrite || table[0].Status != write.StatusSucceeded {
		t.Fatalf("安装应经写登记：%+v", table)
	}
}

// TestAppServerModalityCapabilities 用假服务端覆盖范围 5：模态能力状态与不可用
// 原因；服务端声明不支持时是快照，不是客户端错误。
func TestAppServerModalityCapabilities(t *testing.T) {
	server, harness, _ := startSettingsServer(t, fakeappserver.Script{
		Features: allWireFeatures(),
		Replies: []fakeappserver.Reply{
			{Method: "modality.capabilities", Result: map[string]any{
				"supported":         true,
				"canConfigure":      true,
				"taskSupported":     true,
				"realtimeSupported": false,
				"revision":          "rev-1",
				"categories": []any{
					map[string]any{
						"category":  "image",
						"available": false,
						"reasons": []any{
							map[string]any{"dimension": "configuration", "detail": "no provider"},
						},
						"operations": []any{
							map[string]any{
								"capabilityId": "image.generate",
								"category":     "image",
								"operation":    "generate",
								"available":    false,
								"reasons": []any{
									map[string]any{"dimension": "runtime", "detail": "runtime missing"},
								},
								"default": nil,
								"options": []any{
									map[string]any{
										"providerId": "p1",
										"modelId":    "m1",
										"available":  false,
										"reason": map[string]any{
											"dimension": "runtime", "detail": "not installed",
										},
									},
								},
							},
						},
					},
				},
			}},
		},
	})

	manager := mustNewManager(t, Config{
		Features: connectedFeatures(server, harness),
		Modality: harness,
	})
	caps, err := manager.RefreshModality(context.Background())
	if err != nil {
		t.Fatalf("读取模态能力: %v", err)
	}
	if !caps.Supported || caps.TaskSupported == nil || !*caps.TaskSupported {
		t.Fatalf("模态能力状态不符：%+v", caps)
	}
	if caps.RealtimeSupported == nil || *caps.RealtimeSupported {
		t.Fatalf("实时传输支持位不符：%+v", caps.RealtimeSupported)
	}
	if caps.Revision == nil || *caps.Revision != "rev-1" {
		t.Fatalf("模态版本不符：%+v", caps.Revision)
	}
	if len(caps.Categories) != 1 || caps.Categories[0].Available {
		t.Fatalf("模态类别不符：%+v", caps.Categories)
	}
	operation := caps.Categories[0].Operations[0]
	if operation.CapabilityID != "image.generate" || operation.Available {
		t.Fatalf("模态操作不符：%+v", operation)
	}
	if len(operation.Reasons) != 1 || operation.Reasons[0].Detail != "runtime missing" {
		t.Fatalf("模态操作原因不符：%+v", operation.Reasons)
	}
	if operation.Default != nil {
		t.Fatalf("服务端给 null 的默认项不应变成占位值：%+v", operation.Default)
	}
	if len(operation.Options) != 1 || operation.Options[0].Reason == nil ||
		operation.Options[0].Reason.Detail != "not installed" {
		t.Fatalf("模态选项原因不符：%+v", operation.Options)
	}
	if count := len(server.RequestsFor("modality.capabilities")); count != 1 {
		t.Fatalf("模态能力读取次数不符：%d", count)
	}

	// 服务端声明不支持：返回快照与原因，不当成能力缺失错误。
	unsupportedServer, unsupportedHarness, _ := startSettingsServer(t, fakeappserver.Script{
		Features: allWireFeatures(),
		Replies: []fakeappserver.Reply{
			{Method: "modality.capabilities", Result: map[string]any{
				"supported":         false,
				"canConfigure":      false,
				"unavailableReason": "服务端未提供模态能力",
			}},
		},
	})
	unsupported := mustNewManager(t, Config{
		Features: connectedFeatures(unsupportedServer, unsupportedHarness),
		Modality: unsupportedHarness,
	})
	caps, err = unsupported.RefreshModality(context.Background())
	if err != nil {
		t.Fatalf("服务端声明不支持不是能力缺失错误：%v", err)
	}
	if caps.Supported || caps.UnavailableReason == "" {
		t.Fatalf("不支持状态应带原因：%+v", caps)
	}

	// 方法不声明：能力缺失错误，且不发出请求。
	missingServer, missingHarness, _ := startSettingsServer(t, fakeappserver.Script{Features: []string{}})
	missing := mustNewManager(t, Config{
		Features: connectedFeatures(missingServer, missingHarness),
		Modality: missingHarness,
	})
	if _, err := missing.RefreshModality(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺模态能力应分类：%v", err)
	}
	if count := len(missingServer.RequestsFor("modality.capabilities")); count != 0 {
		t.Fatalf("缺能力时不应发出请求：%d", count)
	}
}

// connectedFeatures 用假服务端的 hello 回执做能力声明检查。
func connectedFeatures(_ *fakeappserver.Server, harness *wireHarness) Features {
	return harness
}

func modelProvidersReply(version any, providerID, modelID string) map[string]any {
	return map[string]any{
		"providers": []any{
			map[string]any{
				"id":      "cloud",
				"name":    "云端",
				"enabled": true,
				"models": []any{
					map[string]any{"id": "c1", "displayName": "云端一号"},
				},
				"enabledModelIds": []any{"c1"},
			},
		},
		"selection": map[string]any{
			"providerId": providerID,
			"modelId":    modelID,
			"version":    version,
		},
	}
}

func signedInAccountReply() map[string]any {
	return map[string]any{
		"status": "signed_in",
		"identity": map[string]any{
			"accountId": "acct-wire",
			"email":     "wire@example.com",
		},
	}
}

// TestAppServerSelectModelWritesAndRefreshes 用假服务端覆盖模型选择写入：经
// 管理写入只发一次，成功后重读并回填服务端确认的选择。
func TestAppServerSelectModelWritesAndRefreshes(t *testing.T) {
	server, harness, writer := startSettingsServer(t, fakeappserver.Script{
		Features: allWireFeatures(),
		Replies: []fakeappserver.Reply{
			{Method: "models.providers.list", Result: modelProvidersReply(41, "cloud", "c1")},
			{Method: "provider.select", Result: map[string]any{"ok": true}},
			{Method: "account.status", Result: signedInAccountReply()},
			{Method: "model.list", Result: map[string]any{
				"models": []any{map[string]any{"id": "c1"}},
			}},
		},
	})

	accountWire := &accountWire{client: harness.client}
	acct, err := account.New(account.Config{Status: accountWire, Models: accountWire})
	if err != nil {
		t.Fatalf("创建账号: %v", err)
	}
	if _, err := acct.Refresh(context.Background()); err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}
	if _, err := acct.RefreshModels(context.Background()); err != nil {
		t.Fatalf("读取账号模型: %v", err)
	}

	manager := mustNewManager(t, Config{
		Features: connectedFeatures(server, harness),
		Models:   harness,
		Account:  acct,
		Writer:   writer,
	})
	if _, err := manager.RefreshModels(context.Background()); err != nil {
		t.Fatalf("预读模型目录: %v", err)
	}

	snapshot, err := manager.SelectModel(context.Background(), SelectModelRequest{
		ProviderID:       "cloud",
		ModelID:          "c1",
		ExpectedRevision: "41",
	})
	if err != nil {
		t.Fatalf("写入模型选择: %v", err)
	}
	if snapshot.Selection == nil || snapshot.Selection.ProviderID != "cloud" ||
		snapshot.Selection.ModelID != "c1" || snapshot.Selection.Revision != "sel-2" {
		t.Fatalf("选择写入后应回填服务端事实：%+v", snapshot.Selection)
	}
	if count := len(server.RequestsFor("provider.select")); count != 1 {
		t.Fatalf("选择写入请求次数不符：%d", count)
	}
	if count := len(server.RequestsFor("models.providers.list")); count != 2 {
		t.Fatalf("成功写入后应重读一次目录：%d", count)
	}
	var params map[string]any
	if err := json.Unmarshal(server.RequestsFor("provider.select")[0].Params, &params); err != nil {
		t.Fatalf("选择写入参数无法解析: %v", err)
	}
	if params["providerId"] != "cloud" || params["modelId"] != "c1" || params["expectedVersion"] != "41" {
		t.Fatalf("选择写入参数不符：%+v", params)
	}
	table := writer.Table()
	if len(table) != 1 || table[0].Kind != write.KindManagementWrite || table[0].Status != write.StatusSucceeded {
		t.Fatalf("选择写入应经写登记：%+v", table)
	}
}

// TestAppServerSelectModelServerConflict 用假服务端覆盖选择写入的服务端版本冲突：
// 拒绝后只读一次目录判断，不重发。
func TestAppServerSelectModelServerConflict(t *testing.T) {
	server, harness, writer := startSettingsServer(t, fakeappserver.Script{
		Features: allWireFeatures(),
		Replies: []fakeappserver.Reply{
			{Method: "models.providers.list", Result: modelProvidersReply(41, "cloud", "c1")},
			{Method: "provider.select", Error: fakeappserver.AppError(-32000, "revision_conflict", "选择版本已变")},
			{Method: "account.status", Result: signedInAccountReply()},
			{Method: "model.list", Result: map[string]any{
				"models": []any{map[string]any{"id": "c1"}},
			}},
		},
	})

	accountWire := &accountWire{client: harness.client}
	acct, err := account.New(account.Config{Status: accountWire, Models: accountWire})
	if err != nil {
		t.Fatalf("创建账号: %v", err)
	}
	if _, err := acct.Refresh(context.Background()); err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}
	if _, err := acct.RefreshModels(context.Background()); err != nil {
		t.Fatalf("读取账号模型: %v", err)
	}

	manager := mustNewManager(t, Config{
		Features: connectedFeatures(server, harness),
		Models:   harness,
		Account:  acct,
		Writer:   writer,
	})
	if _, err := manager.RefreshModels(context.Background()); err != nil {
		t.Fatalf("预读模型目录: %v", err)
	}
	// 模拟另一个客户端在写入前改了选择。
	harness.setSelectionOverride(&Selection{ProviderID: "cloud", ModelID: "c1", Revision: "42"})

	if _, err := manager.SelectModel(context.Background(), SelectModelRequest{
		ProviderID:       "cloud",
		ModelID:          "c1",
		ExpectedRevision: "41",
	}); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("服务端版本冲突应返回冲突：%v", err)
	}
	if count := len(server.RequestsFor("provider.select")); count != 1 {
		t.Fatalf("冲突不应重发：%d", count)
	}
	if count := len(server.RequestsFor("models.providers.list")); count != 2 {
		t.Fatalf("冲突判定只应读一次当前目录：%d", count)
	}
	cached, _ := manager.Models()
	if cached.Selection == nil || cached.Selection.Revision != "42" {
		t.Fatalf("冲突后应采用服务端当前选择：%+v", cached.Selection)
	}
}

// TestAppServerSelectModelSignedOut 用假服务端覆盖未登录时不得写入选择。
func TestAppServerSelectModelSignedOut(t *testing.T) {
	server, harness, writer := startSettingsServer(t, fakeappserver.Script{
		Features: allWireFeatures(),
		Replies: []fakeappserver.Reply{
			{Method: "models.providers.list", Result: modelProvidersReply(41, "cloud", "c1")},
			{Method: "account.status", Result: map[string]any{"status": "signed_out"}},
			{Method: "model.list", Result: map[string]any{
				"models": []any{map[string]any{"id": "c1"}},
			}},
		},
	})

	accountWire := &accountWire{client: harness.client}
	acct, err := account.New(account.Config{Status: accountWire, Models: accountWire})
	if err != nil {
		t.Fatalf("创建账号: %v", err)
	}
	if _, err := acct.Refresh(context.Background()); err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}
	if _, err := acct.RefreshModels(context.Background()); err != nil {
		t.Fatalf("读取账号模型: %v", err)
	}

	manager := mustNewManager(t, Config{
		Features: connectedFeatures(server, harness),
		Models:   harness,
		Account:  acct,
		Writer:   writer,
	})
	if _, err := manager.RefreshModels(context.Background()); err != nil {
		t.Fatalf("预读模型目录: %v", err)
	}

	_, err = manager.SelectModel(context.Background(), SelectModelRequest{
		ProviderID:       "cloud",
		ModelID:          "c1",
		ExpectedRevision: "41",
	})
	if !errors.Is(err, ErrSelectionUnavailable) {
		t.Fatalf("未登录时不得写入选择：%v", err)
	}
	if !strings.Contains(err.Error(), "未登录") {
		t.Fatalf("未登录原因不符：%v", err)
	}
	if count := len(server.RequestsFor("provider.select")); count != 0 {
		t.Fatalf("未登录时不应发出写入：%d", count)
	}
}
