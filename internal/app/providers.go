package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/settings"
)

// provider 与协作/agent 面在服务端上的方法名（alpha.21 实测，见 D11a §6.3）。
const (
	methodProviderList           = "provider.list"
	methodProviderManagement     = "provider.management"
	methodProviderProbe          = "provider.probe"
	methodCollaborationPolicy    = "collaboration.model_policy"
	methodCollaborationSetPolicy = "collaboration.set_model_policy"
	methodAgentList              = "agent.list"
	methodSessionAgentGet        = "session.agent.get"
	methodSessionAgentUpdate     = "session.agent.update"
)

// providerAdmin 实现 settings.ProviderAdmin：清单、详情与只读探测。
//
// 只读；探测不改配置。凭据值从不出现在回执映射里（只保留「是否已配置」）。
type providerAdmin struct {
	app *App
}

func (p *providerAdmin) ReadProviders(ctx context.Context) (settings.ProviderList, error) {
	client, err := p.app.requireControl("读取 provider 清单")
	if err != nil {
		return settings.ProviderList{}, err
	}
	raw, err := client.Call(ctx, methodProviderList, map[string]any{})
	if err != nil {
		return settings.ProviderList{}, err
	}
	var reply struct {
		Providers []struct {
			ProviderID  string `json:"providerId"`
			Source      string `json:"source"`
			DisplayName string `json:"displayName"`
			DriverKind  string `json:"driverKind"`
			Auth        struct {
				Type string `json:"type"`
			} `json:"auth"`
			CredentialAuthority  string   `json:"credentialAuthority"`
			BaseURL              string   `json:"baseUrl"`
			ProxyConfigured      bool     `json:"proxyConfigured"`
			TimeoutMS            *int64   `json:"timeoutMs"`
			HeaderNames          []string `json:"headerNames"`
			CredentialConfigured bool     `json:"credentialConfigured"`
			RemoteModelList      bool     `json:"remoteModelList"`
			Enabled              bool     `json:"enabled"`
			Version              int      `json:"version"`
		} `json:"providers"`
		Errors []struct {
			ProviderID string `json:"providerId"`
			Message    string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return settings.ProviderList{}, fmt.Errorf("provider.list 回执无法解析: %w", err)
	}
	list := settings.ProviderList{Providers: make([]settings.ProviderSummary, 0, len(reply.Providers))}
	for _, item := range reply.Providers {
		list.Providers = append(list.Providers, settings.ProviderSummary{
			ID:                   item.ProviderID,
			Source:               item.Source,
			DisplayName:          item.DisplayName,
			DriverKind:           item.DriverKind,
			AuthType:             item.Auth.Type,
			CredentialAuthority:  item.CredentialAuthority,
			BaseURL:              item.BaseURL,
			ProxyConfigured:      item.ProxyConfigured,
			TimeoutMS:            item.TimeoutMS,
			HeaderNames:          append([]string(nil), item.HeaderNames...),
			CredentialConfigured: item.CredentialConfigured,
			RemoteModelList:      item.RemoteModelList,
			Enabled:              item.Enabled,
			Version:              item.Version,
		})
	}
	for _, item := range reply.Errors {
		list.Errors = append(list.Errors, settings.ProviderRecordError{
			ProviderID: item.ProviderID,
			Message:    item.Message,
		})
	}
	return list, nil
}

func (p *providerAdmin) ReadProviderDetail(
	ctx context.Context,
	providerID string,
) (settings.ProviderDetail, error) {
	client, err := p.app.requireControl("读取 provider 详情")
	if err != nil {
		return settings.ProviderDetail{}, err
	}
	raw, err := client.Call(ctx, methodProviderManagement, map[string]any{"providerId": providerID})
	if err != nil {
		return settings.ProviderDetail{}, err
	}
	detail, err := providerDetailFromWire(raw, providerID)
	if err != nil {
		return settings.ProviderDetail{}, err
	}
	return detail, nil
}

// ProbeProvider 调 `provider.probe`：只读探测，不写配置。
func (p *providerAdmin) ProbeProvider(
	ctx context.Context,
	request settings.ProviderProbeRequest,
) (settings.ProviderProbeResult, error) {
	client, err := p.app.requireControl("探测 provider")
	if err != nil {
		return settings.ProviderProbeResult{}, err
	}
	params := map[string]any{"providerId": request.ProviderID}
	if request.Connection != nil {
		params["connection"] = providerConnectionWire(*request.Connection)
	}
	if request.Patch != nil {
		params["connectionPatch"] = providerPatchWire(*request.Patch)
	}
	raw, err := client.Call(ctx, methodProviderProbe, params)
	if err != nil {
		return settings.ProviderProbeResult{}, err
	}
	var reply struct {
		Models []map[string]json.RawMessage `json:"models"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return settings.ProviderProbeResult{}, fmt.Errorf("provider.probe 回执无法解析: %w", err)
	}
	result := settings.ProviderProbeResult{Models: make([]settings.ProviderModelInput, 0, len(reply.Models))}
	for _, item := range reply.Models {
		result.Models = append(result.Models, providerModelInputFromWire(item))
	}
	return result, nil
}

// providerDetailFromWire 把 `provider.management` 回执映射成详情视图。
func providerDetailFromWire(raw json.RawMessage, providerID string) (settings.ProviderDetail, error) {
	var reply struct {
		Providers []struct {
			ProviderID          string          `json:"providerId"`
			Source              string          `json:"source"`
			DisplayName         string          `json:"displayName"`
			CredentialAuthority string          `json:"credentialAuthority"`
			Version             int             `json:"version"`
			Enabled             *bool           `json:"enabled"`
			Actions             []string        `json:"actions"`
			Connection          json.RawMessage `json:"connection"`
			Models              json.RawMessage `json:"models"`
			Preferences         map[string]struct {
				ReasoningEffort json.RawMessage `json:"reasoningEffort"`
				ServiceTier     json.RawMessage `json:"serviceTier"`
			} `json:"preferences"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return settings.ProviderDetail{}, fmt.Errorf("provider.management 回执无法解析: %w", err)
	}
	if len(reply.Providers) == 0 {
		return settings.ProviderDetail{}, fmt.Errorf("provider.management 没有返回 provider")
	}
	// 回执可能包含多条；按请求标识挑选，找不到时用第一条。
	entry := reply.Providers[0]
	if providerID != "" {
		for _, candidate := range reply.Providers {
			if candidate.ProviderID == providerID {
				entry = candidate
				break
			}
		}
	}
	version := entry.Version
	detail := settings.ProviderDetail{
		ID:                  entry.ProviderID,
		DisplayName:         entry.DisplayName,
		Source:              entry.Source,
		CredentialAuthority: entry.CredentialAuthority,
		Version:             &version,
		Enabled:             entry.Enabled,
		Actions:             append([]string(nil), entry.Actions...),
	}
	if len(entry.Connection) > 0 {
		var connection struct {
			DriverKind string `json:"driverKind"`
			Auth       struct {
				Type string `json:"type"`
			} `json:"auth"`
			BaseURL              string   `json:"baseUrl"`
			CredentialConfigured bool     `json:"credentialConfigured"`
			HeaderNames          []string `json:"headerNames"`
			ProxyURL             string   `json:"proxyUrl"`
			ProxyAuthConfigured  bool     `json:"proxyAuthConfigured"`
			TimeoutMS            *int64   `json:"timeoutMs"`
		}
		if err := json.Unmarshal(entry.Connection, &connection); err == nil {
			detail.Connection = &settings.ProviderConnectionInfo{
				DriverKind:           connection.DriverKind,
				AuthType:             connection.Auth.Type,
				BaseURL:              connection.BaseURL,
				CredentialConfigured: connection.CredentialConfigured,
				HeaderNames:          append([]string(nil), connection.HeaderNames...),
				ProxyURL:             connection.ProxyURL,
				ProxyAuthConfigured:  connection.ProxyAuthConfigured,
				TimeoutMS:            connection.TimeoutMS,
			}
		}
	}
	for _, model := range parseProviderModels(entry.Models) {
		detail.Models = append(detail.Models, settings.ProviderModelInfo{
			ID: model.Model,
			// 真实 provider.management 的模型条目不一定带显示名；为空时按标识
			// 兜底展示（只是显示值，不回写服务端）。
			DisplayName:       firstNonEmpty(model.DisplayName, model.Model),
			Source:            model.Source,
			Enabled:           model.Enabled,
			Endpoint:          model.Endpoint,
			MissingLimits:     append([]string(nil), model.MissingLimits...),
			UnavailableReason: model.UnavailableReason,
			Actions:           append([]string(nil), model.Actions...),
			// 真实回执没有模型状态字段；设置领域要求 active/beta/deprecated
			// 三值之一，缺省按 active 展示（不写回服务端）。
			Status: firstNonEmpty(model.Status, "active"),
		})
	}
	if len(entry.Preferences) > 0 {
		detail.Preferences = make(map[string]settings.ModelPreferences, len(entry.Preferences))
		for modelID, preference := range entry.Preferences {
			detail.Preferences[modelID] = settings.ModelPreferences{
				ReasoningEffort: preferenceString(preference.ReasoningEffort),
				ServiceTier:     preferenceString(preference.ServiceTier),
			}
		}
	}
	return detail, nil
}

// providerModelEntry 是管理回执里的一个模型条目（数组或对象两种写法都归一）。
type providerModelEntry struct {
	Model             string   `json:"model"`
	Source            string   `json:"source"`
	Enabled           bool     `json:"enabled"`
	Endpoint          string   `json:"endpoint"`
	DisplayName       string   `json:"displayName"`
	Status            string   `json:"status"`
	MissingLimits     []string `json:"missingLimits"`
	UnavailableReason string   `json:"unavailableReason"`
	Actions           []string `json:"actions"`
}

// parseProviderModels 兼容 `models` 的两种形状：数组，或按模型标识索引的对象。
func parseProviderModels(raw json.RawMessage) []providerModelEntry {
	if len(raw) == 0 {
		return nil
	}
	var list []providerModelEntry
	if err := json.Unmarshal(raw, &list); err == nil {
		return list
	}
	var keyed map[string]providerModelEntry
	if err := json.Unmarshal(raw, &keyed); err != nil {
		return nil
	}
	keys := make([]string, 0, len(keyed))
	for key := range keyed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	list = make([]providerModelEntry, 0, len(keys))
	for _, key := range keys {
		entry := keyed[key]
		if entry.Model == "" {
			entry.Model = key
		}
		list = append(list, entry)
	}
	return list
}

// preferenceString 读取偏好取值：服务端可能给字符串或 {value} 对象。
func preferenceString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var object struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &object); err == nil {
		return object.Value
	}
	return ""
}

// providerConnectionWire 把连接输入映射成 wire 形状（凭据只透传）。
func providerConnectionWire(input settings.ProviderConnectionInput) map[string]any {
	out := map[string]any{
		"driverKind": input.DriverKind,
		"auth":       input.Auth,
		"baseUrl":    input.BaseURL,
	}
	if len(input.Headers) > 0 {
		headers := make(map[string]string, len(input.Headers))
		keys := make([]string, 0, len(input.Headers))
		for key := range input.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			headers[key] = input.Headers[key]
		}
		out["headers"] = headers
	}
	if input.Credential != nil {
		out["credential"] = map[string]any{
			"action":    input.Credential.Action,
			"value":     input.Credential.Value,
			"reference": input.Credential.Reference,
		}
	}
	return out
}

// providerPatchWire 把更新补丁映射成 wire 形状：nil 字段不下发。
func providerPatchWire(patch settings.ProviderPatch) map[string]any {
	out := map[string]any{}
	if patch.DisplayName != nil {
		out["displayName"] = *patch.DisplayName
	}
	if patch.BaseURL != nil {
		out["baseUrl"] = *patch.BaseURL
	}
	if patch.Enabled != nil {
		out["enabled"] = *patch.Enabled
	}
	if patch.Credential != nil {
		out["credential"] = map[string]any{
			"action":    patch.Credential.Action,
			"value":     patch.Credential.Value,
			"reference": patch.Credential.Reference,
		}
	}
	if len(patch.Headers) > 0 {
		headers := make([]map[string]any, 0, len(patch.Headers))
		for _, header := range patch.Headers {
			headers = append(headers, map[string]any{
				"action": header.Action,
				"name":   header.Name,
				"value":  header.Value,
			})
		}
		out["headers"] = headers
	}
	return out
}

// providerModelInputFromWire 映射探测或新建回执里的模型条目。
func providerModelInputFromWire(item map[string]json.RawMessage) settings.ProviderModelInput {
	modelID := firstNonEmpty(rawString(item, "modelId"), rawString(item, "model"), rawString(item, "id"))
	input := settings.ProviderModelInput{
		ID: modelID,
		// 探测回执同样可能没有显示名/状态：按标识与 active 兜底展示。
		DisplayName: firstNonEmpty(rawString(item, "displayName"), modelID),
		Status:      firstNonEmpty(rawString(item, "status"), "active"),
		GuidePrompt: rawString(item, "modelGuidePrompt"),
	}
	var capabilities struct {
		ContextWindow   *int64 `json:"contextWindow"`
		MaxOutputTokens *int64 `json:"maxOutputTokens"`
		ToolCalling     *bool  `json:"toolCalling"`
		Vision          *bool  `json:"vision"`
	}
	if raw, ok := item["capabilities"]; ok {
		_ = json.Unmarshal(raw, &capabilities)
		input.ContextWindow = capabilities.ContextWindow
		input.MaxOutputTokens = capabilities.MaxOutputTokens
		input.ToolCalling = capabilities.ToolCalling
		input.Vision = capabilities.Vision
	}
	return input
}

func rawString(values map[string]json.RawMessage, key string) string {
	raw, ok := values[key]
	if !ok {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return ""
	}
	return text
}

// collaborationReader 实现 settings.CollaborationReader。
type collaborationReader struct {
	app *App
}

func (r *collaborationReader) ReadCollaborationPolicy(
	ctx context.Context,
) (settings.CollaborationPolicySnapshot, error) {
	client, err := r.app.requireControl("读取协作模型策略")
	if err != nil {
		return settings.CollaborationPolicySnapshot{}, err
	}
	raw, err := client.Call(ctx, methodCollaborationPolicy, map[string]any{})
	if err != nil {
		return settings.CollaborationPolicySnapshot{}, err
	}
	return collaborationPolicyFromWire(raw)
}

func collaborationPolicyFromWire(raw json.RawMessage) (settings.CollaborationPolicySnapshot, error) {
	var reply struct {
		Revision string `json:"revision"`
		Policy   struct {
			Mode  string `json:"mode"`
			Model *struct {
				ProviderID string `json:"providerId"`
				ModelID    string `json:"modelId"`
			} `json:"model"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return settings.CollaborationPolicySnapshot{}, fmt.Errorf("collaboration.model_policy 回执无法解析: %w", err)
	}
	snapshot := settings.CollaborationPolicySnapshot{
		Revision: reply.Revision,
		Policy: settings.CollaborationModelPolicy{
			Mode: settings.CollaborationPolicyMode(reply.Policy.Mode),
		},
	}
	if reply.Policy.Model != nil {
		snapshot.Policy.ModelProviderID = reply.Policy.Model.ProviderID
		snapshot.Policy.ModelID = reply.Policy.Model.ModelID
	}
	return snapshot, nil
}

// agentAdmin 实现 settings.AgentAdmin：agent 清单与会话当前 agent。
type agentAdmin struct {
	app *App
}

func (a *agentAdmin) ListAgents(ctx context.Context) ([]settings.AgentProfile, error) {
	client, err := a.app.requireControl("读取 agent 清单")
	if err != nil {
		return nil, err
	}
	// agent.list 是内建方法，hello 不声明；直接调用，缺失时按调用结果分类。
	raw, err := client.Call(ctx, methodAgentList, map[string]any{})
	if err != nil {
		return nil, err
	}
	var entries []struct {
		AgentID     string `json:"agentId"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Sessions    int    `json:"sessions"`
		Default     bool   `json:"default"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("agent.list 回执无法解析: %w", err)
	}
	agents := make([]settings.AgentProfile, 0, len(entries))
	for _, entry := range entries {
		agents = append(agents, settings.AgentProfile{
			ID:          entry.AgentID,
			Name:        entry.Name,
			Description: entry.Description,
			Sessions:    entry.Sessions,
			Default:     entry.Default,
		})
	}
	return agents, nil
}

func (a *agentAdmin) ReadSessionAgent(
	ctx context.Context,
	sessionID string,
	branchID string,
) (settings.SessionAgent, error) {
	runtime, err := a.app.openRuntime(sessionID, methodSessionAgentGet)
	if err != nil {
		return settings.SessionAgent{}, err
	}
	raw, err := runtime.lease.Subscription().Call(ctx, methodSessionAgentGet, map[string]any{
		"sessionId": sessionID,
		"branchId":  branchOrMain(branchID),
	})
	if err != nil {
		return settings.SessionAgent{}, err
	}
	var reply struct {
		AgentID string `json:"agentId"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return settings.SessionAgent{}, fmt.Errorf("session.agent.get 回执无法解析: %w", err)
	}
	return settings.SessionAgent{
		SessionID: sessionID,
		BranchID:  branchOrMain(branchID),
		AgentID:   reply.AgentID,
	}, nil
}

func (a *agentAdmin) SupportsSessionAgentGet(ctx context.Context, sessionID string) (bool, error) {
	return a.supports(ctx, sessionID, methodSessionAgentGet)
}

func (a *agentAdmin) SupportsSessionAgentUpdate(ctx context.Context, sessionID string) (bool, error) {
	return a.supports(ctx, sessionID, methodSessionAgentUpdate)
}

func (a *agentAdmin) supports(ctx context.Context, sessionID, method string) (bool, error) {
	// 门禁：先 hello，再会话方法表；两边都确认不了按缺失。
	if conn := a.app.connection(); conn != nil && conn.HasFeature(method) {
		return true, nil
	}
	runtime, err := a.app.openRuntime(sessionID, method)
	if err != nil {
		return false, nil
	}
	methods, err := a.app.sessionMethods(ctx, runtime)
	if err != nil {
		a.app.record("读取会话方法清单失败，按缺失处理: %v", err)
		return false, nil
	}
	return methods.Has(method), nil
}

// branchOrMain 把空分支按主线处理，与服务端约定一致。
func branchOrMain(branchID string) string {
	if branchID == "" {
		return mainBranchID
	}
	return branchID
}

// 编译期确认。
var (
	_ settings.ProviderAdmin       = (*providerAdmin)(nil)
	_ settings.CollaborationReader = (*collaborationReader)(nil)
	_ settings.AgentAdmin          = (*agentAdmin)(nil)
	_                              = protocol.ProtocolVersion
)
