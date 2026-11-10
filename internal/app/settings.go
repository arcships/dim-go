package app

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/settings"
)

// featureSource 把当前控制连接的 hello 能力声明交给设置领域。
//
// 它每次调用都取当前连接：重连或服务实例被替换后，能力清单跟着换，不会拿着旧
// 连接的能力清单做判断。
type featureSource struct {
	app *App
}

// HasFeature 实现 settings.Features。
func (s *featureSource) HasFeature(name string) bool {
	conn := s.app.connection()
	if conn == nil {
		return false
	}
	return conn.HasFeature(name)
}

// settingsSourcesReader 读取设置来源清单。
type settingsSourcesReader struct {
	app *App
}

func (r *settingsSourcesReader) ReadSettingsSources(ctx context.Context) (settings.Sources, error) {
	client, err := r.app.requireControl("读取设置来源")
	if err != nil {
		return settings.Sources{}, err
	}
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
	if err := client.CallInto(ctx, "settings.sources", nil, &reply); err != nil {
		return settings.Sources{}, err
	}
	sources := settings.Sources{Dialect: reply.Dialect, Sources: make([]settings.Source, 0, len(reply.Sources))}
	for _, source := range reply.Sources {
		scopes := make([]settings.ScopeKind, 0, len(source.Scopes))
		for _, scope := range source.Scopes {
			scopes = append(scopes, settings.ScopeKind(scope))
		}
		sources.Sources = append(sources.Sources, settings.Source{
			Name:      source.Source,
			Scopes:    scopes,
			UserOnly:  source.UserOnly,
			WriteOnly: source.WriteOnly,
			Schema:    source.Schema,
		})
	}
	return sources, nil
}

// settingsProjectsReader 读取已授权项目根。
type settingsProjectsReader struct {
	app *App
}

func (r *settingsProjectsReader) ReadSettingsProjects(ctx context.Context) ([]settings.Project, error) {
	client, err := r.app.requireControl("读取项目根清单")
	if err != nil {
		return nil, err
	}
	var reply struct {
		Projects []struct {
			Root int    `json:"root"`
			CWD  string `json:"cwd"`
		} `json:"projects"`
	}
	if err := client.CallInto(ctx, "settings.projects", nil, &reply); err != nil {
		return nil, err
	}
	projects := make([]settings.Project, 0, len(reply.Projects))
	for _, project := range reply.Projects {
		projects = append(projects, settings.Project{Root: project.Root, CWD: project.CWD})
	}
	return projects, nil
}

// settingsDocumentReader 按来源与作用域读取设置文档。
type settingsDocumentReader struct {
	app *App
}

func (r *settingsDocumentReader) ReadSettings(
	ctx context.Context,
	source string,
	scope settings.Scope,
) (settings.Document, error) {
	client, err := r.app.requireControl("读取设置文档")
	if err != nil {
		return settings.Document{}, err
	}
	var reply struct {
		Revision string          `json:"revision"`
		Document json.RawMessage `json:"document"`
	}
	if err := client.CallInto(ctx, "settings.read", map[string]any{
		"source": source,
		"scope":  scopeWireValue(scope),
	}, &reply); err != nil {
		return settings.Document{}, err
	}
	return settings.Document{
		Source:   source,
		Scope:    scope,
		Revision: reply.Revision,
		Document: reply.Document,
	}, nil
}

// settingsModelReader 读取 provider 与模型目录。
type settingsModelReader struct {
	app *App
}

func (r *settingsModelReader) ReadModelCatalog(ctx context.Context) (settings.ModelCatalog, error) {
	client, err := r.app.requireControl("读取模型目录")
	if err != nil {
		return settings.ModelCatalog{}, err
	}
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
	if err := client.CallInto(ctx, "models.providers.list", nil, &reply); err != nil {
		return settings.ModelCatalog{}, err
	}
	catalog := settings.ModelCatalog{Providers: make([]settings.Provider, 0, len(reply.Providers))}
	for _, provider := range reply.Providers {
		if provider.Enabled == nil {
			// 缺字段不补假值：按事实不合法报出来。
			return settings.ModelCatalog{}, errors.New("provider 缺少 enabled")
		}
		name := provider.Name
		if name == "" {
			name = provider.DisplayName
		}
		models := make([]settings.ProviderModel, 0, len(provider.Models))
		for _, model := range provider.Models {
			modelName := model.Name
			if modelName == "" {
				modelName = model.DisplayName
			}
			models = append(models, settings.ProviderModel{ID: model.ID, Name: modelName})
		}
		catalog.Providers = append(catalog.Providers, settings.Provider{
			ID:              provider.ID,
			Name:            name,
			Enabled:         *provider.Enabled,
			Models:          models,
			EnabledModelIDs: provider.EnabledModelIDs,
		})
	}
	if reply.Selection != nil {
		catalog.Selection = &settings.Selection{
			ProviderID:      reply.Selection.ProviderID,
			ModelID:         reply.Selection.ModelID,
			Revision:        wireRevision(reply.Selection.Version),
			ReasoningEffort: reply.Selection.ReasoningEffort,
			ServiceTier:     reply.Selection.ServiceTier,
		}
	}
	return catalog, nil
}

// settingsStorageReader 只读存储占用读数，不做任何清理。
type settingsStorageReader struct {
	app *App
}

func (r *settingsStorageReader) ReadStorage(ctx context.Context) (settings.StorageReading, error) {
	client, err := r.app.requireControl("读取存储占用")
	if err != nil {
		return settings.StorageReading{}, err
	}
	var reply struct {
		DatabaseBytes    *int64 `json:"databaseBytes"`
		WALBytes         *int64 `json:"walBytes"`
		ReclaimableBytes *int64 `json:"reclaimableBytes"`
	}
	if err := client.CallInto(ctx, "storage.report", nil, &reply); err != nil {
		return settings.StorageReading{}, err
	}
	return settings.StorageReading{
		DatabaseBytes:    reply.DatabaseBytes,
		WALBytes:         reply.WALBytes,
		ReclaimableBytes: reply.ReclaimableBytes,
	}, nil
}

// settingsRuntimeReader 读取运行组件清单。
//
// alpha.21 不再注册 runtime.*（D1 实测全部 unknown_method）：按能力缺失处理，
// 不发请求、不发明方法；能力恢复后由设置领域重新读取即可。
type settingsRuntimeReader struct {
	app *App
}

func (r *settingsRuntimeReader) ReadRuntimes(ctx context.Context) ([]settings.RuntimeEntry, error) {
	return nil, runtimeCapabilityMissing()
}

// runtimeCapabilityMissing 是运行组件面在 alpha.21 的能力缺失错误。
func runtimeCapabilityMissing() error {
	return &settings.CapabilityError{
		Capability: settings.FeatureRuntimeList,
		Reason:     "服务端未注册 runtime.*（alpha.21 实测 unknown_method）",
	}
}

// settingsModalityReader 读取模态能力状态。
type settingsModalityReader struct {
	app *App
}

type modalityWireReason struct {
	Dimension string `json:"dimension"`
	Detail    string `json:"detail"`
}

type modalityWireOption struct {
	ProviderID string              `json:"providerId"`
	ModelID    string              `json:"modelId"`
	Available  bool                `json:"available"`
	Reason     *modalityWireReason `json:"reason"`
}

type modalityWireOperation struct {
	CapabilityID string               `json:"capabilityId"`
	Category     string               `json:"category"`
	Operation    string               `json:"operation"`
	Key          string               `json:"key"`
	Available    bool                 `json:"available"`
	Reasons      []modalityWireReason `json:"reasons"`
	Default      *modalityWireOption  `json:"default"`
	Effective    *struct {
		ProviderID string `json:"providerId"`
		ModelID    string `json:"modelId"`
		Source     string `json:"source"`
	} `json:"effective"`
	Options []modalityWireOption `json:"options"`
}

func (r *settingsModalityReader) ReadModalityCapabilities(ctx context.Context) (settings.ModalityCapabilities, error) {
	client, err := r.app.requireControl("读取模态能力")
	if err != nil {
		return settings.ModalityCapabilities{}, err
	}
	var reply struct {
		Supported         bool  `json:"supported"`
		CanConfigure      bool  `json:"canConfigure"`
		TaskSupported     *bool `json:"taskSupported"`
		RealtimeSupported *bool `json:"realtimeSupported"`
		Revision          *string
		Categories        []struct {
			Category   string                  `json:"category"`
			Available  bool                    `json:"available"`
			Reasons    []modalityWireReason    `json:"reasons"`
			Operations []modalityWireOperation `json:"operations"`
		} `json:"categories"`
		UnavailableReason string `json:"unavailableReason"`
	}
	if err := client.CallInto(ctx, "modality.capabilities", nil, &reply); err != nil {
		return settings.ModalityCapabilities{}, err
	}
	caps := settings.ModalityCapabilities{
		Supported:         reply.Supported,
		CanConfigure:      reply.CanConfigure,
		TaskSupported:     reply.TaskSupported,
		RealtimeSupported: reply.RealtimeSupported,
		Revision:          reply.Revision,
		UnavailableReason: reply.UnavailableReason,
	}
	for _, category := range reply.Categories {
		mapped := settings.ModalityCategory{
			Category:   category.Category,
			Available:  category.Available,
			Reasons:    mapModalityReasons(category.Reasons),
			Operations: make([]settings.ModalityOperation, 0, len(category.Operations)),
		}
		for _, operation := range category.Operations {
			// 真实回执没有单独的能力标识：由类别与操作名派生一个稳定标识
			// （只用于本地视图，不写回服务端）。不用 key：真实回执里同一 key
			// 会出现在多个操作上，不能当主键。
			capabilityID := category.Category + "." + operation.Operation
			item := settings.ModalityOperation{
				CapabilityID: capabilityID,
				Category:     category.Category,
				Operation:    operation.Operation,
				Key:          operation.Key,
				Available:    operation.Available,
				Reasons:      mapModalityReasons(operation.Reasons),
				Options:      make([]settings.ModalityOption, 0, len(operation.Options)),
			}
			if operation.Default != nil {
				option := mapModalityOption(*operation.Default)
				item.Default = &option
			}
			if operation.Effective != nil {
				item.Effective = &settings.ModalityModel{
					ProviderID: operation.Effective.ProviderID,
					ModelID:    operation.Effective.ModelID,
					Source:     operation.Effective.Source,
				}
			}
			for _, option := range operation.Options {
				item.Options = append(item.Options, mapModalityOption(option))
			}
			mapped.Operations = append(mapped.Operations, item)
		}
		caps.Categories = append(caps.Categories, mapped)
	}
	return caps, nil
}

func mapModalityReasons(reasons []modalityWireReason) []settings.ModalityReason {
	mapped := make([]settings.ModalityReason, 0, len(reasons))
	for _, reason := range reasons {
		mapped = append(mapped, settings.ModalityReason{Dimension: reason.Dimension, Detail: reason.Detail})
	}
	return mapped
}

func mapModalityOption(option modalityWireOption) settings.ModalityOption {
	mapped := settings.ModalityOption{
		ProviderID: option.ProviderID,
		ModelID:    option.ModelID,
		Available:  option.Available,
	}
	if option.Reason != nil {
		mapped.Reason = &settings.ModalityReason{Dimension: option.Reason.Dimension, Detail: option.Reason.Detail}
	}
	return mapped
}

// wireRevision 把服务端的选择版本（字符串或数字）映射成不透明版本。
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

// scopeWireValue 把设置作用域映射成服务端取值：用户作用域是 "user"，
// 项目作用域是 {"project": 根编号}。
func scopeWireValue(scope settings.Scope) any {
	if scope.Kind == settings.ScopeProject {
		return map[string]any{"project": scope.Project}
	}
	return "user"
}

// 让编译期确认连接类型满足设置与账号需要的能力读取接口。
var _ settings.Features = (*daemon.Connection)(nil)
