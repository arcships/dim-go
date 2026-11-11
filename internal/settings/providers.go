package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// ProviderSummary 是 provider 清单里的一项概要。
type ProviderSummary struct {
	ID                   string
	Source               string
	DisplayName          string
	DriverKind           string
	AuthType             string
	CredentialAuthority  string
	BaseURL              string
	ProxyConfigured      bool
	TimeoutMS            *int64
	HeaderNames          []string
	CredentialConfigured bool
	RemoteModelList      bool
	Enabled              bool
	Version              int
}

// ProviderList 是一次 provider 清单读取；Errors 保留服务端逐项错误。
type ProviderList struct {
	Providers []ProviderSummary
	Errors    []ProviderRecordError
}

// ProviderRecordError 是一个 provider 记录的读取错误。
type ProviderRecordError struct {
	ProviderID string
	Message    string
}

// ProviderAccount 是 provider 的账号视图；只保留登录方式与状态。
type ProviderAccount struct {
	Namespace    string
	LoginMethods []string
	State        string
}

// ProviderConnectionInfo 是 provider 连接的安全视图；不含凭据。
type ProviderConnectionInfo struct {
	DriverKind           string
	AuthType             string
	BaseURL              string
	CredentialConfigured bool
	HeaderNames          []string
	ProxyURL             string
	ProxyAuthConfigured  bool
	TimeoutMS            *int64
}

// ProviderModelCapabilities 是模型可编辑能力的视图。
type ProviderModelCapabilities struct {
	ContextWindow   *int64
	MaxOutputTokens *int64
	ToolCalling     *bool
	Vision          *bool
	Modalities      []string
	Reasoning       *ProviderModelReasoning
}

// ProviderModelReasoning 是模型推理能力。
type ProviderModelReasoning struct {
	Supported     bool     `json:"supported"`
	EffortOptions []string `json:"effortOptions,omitempty"`
}

// ProviderModelPricing 是模型目录里的计价字段。数值单位由产品目录约定为
// 每百万 token；wire 没有 unit 字段，客户端不得只凭这三个字段推断最终费用。
type ProviderModelPricing struct {
	InputTokenPrice  *float64
	OutputTokenPrice *float64
	Currency         string
}

// ProviderModelInfo 是 provider 管理视图里的模型。
type ProviderModelInfo struct {
	ID                string
	DisplayName       string
	Status            string
	Source            string
	Enabled           bool
	Endpoint          string
	MissingLimits     []string
	UnavailableReason string
	Actions           []string
	Capabilities      ProviderModelCapabilities
	// Pricing 只在服务端给出计价字段时存在。
	Pricing *ProviderModelPricing
}

// ModelPreferences 是模型偏好。
type ModelPreferences struct {
	ReasoningEffort string
	ServiceTier     string
}

// ProviderDetail 是 provider.management 的安全视图；不含凭据值。
type ProviderDetail struct {
	ID                  string
	DisplayName         string
	Source              string
	CredentialAuthority string
	// Version 为空表示服务端没有给出可写版本；这类 provider 不可更新。
	Version     *int
	Enabled     *bool
	Account     *ProviderAccount
	Actions     []string
	Connection  *ProviderConnectionInfo
	Models      []ProviderModelInfo
	Preferences map[string]ModelPreferences
}

// ProviderAuth 是新建/探测时的认证描述。
type ProviderAuth struct {
	// Type 取 bearer / api_key / custom_header / anonymous / aws_bedrock。
	Type       string `json:"type"`
	HeaderName string `json:"headerName,omitempty"`
	Prefix     string `json:"prefix,omitempty"`
}

// ProviderCredential 是新建 provider 时显式给出的凭据。它只用于本次写入，
// 不缓存、不回显、不写日志。
type ProviderCredential struct {
	// Action 取 set / set_ref。
	Action    string
	Value     string
	Reference string
}

// ProviderConnectionInput 是新建/探测 provider 的连接输入。
type ProviderConnectionInput struct {
	DriverKind string
	Auth       ProviderAuth
	BaseURL    string
	Headers    map[string]string
	// Credential 只在调用方显式给出时下发；nil 表示不下发。
	Credential *ProviderCredential
}

// ProviderModelInput 是新建 provider 时显式给出的模型条目。
type ProviderModelInput struct {
	ID              string
	DisplayName     string
	Status          string
	ContextWindow   *int64
	MaxOutputTokens *int64
	ToolCalling     *bool
	Vision          *bool
	Modalities      []string
	Reasoning       *ProviderModelReasoning
	Pricing         *ProviderModelPricing
	GuidePrompt     string
}

// ProviderCreateRequest 是一次 provider 新建。
type ProviderCreateRequest struct {
	ID          string
	DisplayName string
	Connection  ProviderConnectionInput
	Models      []ProviderModelInput
	Enabled     bool
}

// ProviderHeaderUpdate 是一次请求头变更。
type ProviderHeaderUpdate struct {
	// Action 取 set / remove。
	Action string
	Name   string
	Value  string
}

// ProviderCredentialUpdate 是一次显式凭据变更。Value/Reference 只用于本次写入。
type ProviderCredentialUpdate struct {
	// Action 取 keep / clear / set / set_ref。
	Action    string
	Value     string
	Reference string
}

// ProviderPatch 是一次 provider 更新；nil 字段表示不下发。
type ProviderPatch struct {
	DisplayName *string
	BaseURL     *string
	Enabled     *bool
	Credential  *ProviderCredentialUpdate
	Headers     []ProviderHeaderUpdate
}

// ProviderUpdateRequest 是一次带版本前置条件的 provider 更新。
type ProviderUpdateRequest struct {
	ProviderID      string
	ExpectedVersion int
	Patch           ProviderPatch
}

// ProviderProbeRequest 是一次显式探测：New 与 Existing 只能给一个。
type ProviderProbeRequest struct {
	ProviderID string
	Connection *ProviderConnectionInput
	Patch      *ProviderPatch
}

// ProviderProbeResult 是显式探测返回的模型；它不写配置。
type ProviderProbeResult struct {
	Models []ProviderModelInput
}

// ProviderModelEdit 是模型可编辑字段；nil 表示不下发。
type ProviderModelEdit struct {
	DisplayName     *string
	ContextWindow   *int64
	MaxOutputTokens *int64
	ToolCalling     *bool
	Vision          *bool
	// ModalityCapabilities 为 nil 表示不下发；指向空切片表示显式置空。
	ModalityCapabilities *[]string
	Reasoning            *ProviderModelReasoning
}

// ProviderModelEditRequest 是一次模型配置编辑。
type ProviderModelEditRequest struct {
	ProviderID      string
	ModelID         string
	ExpectedVersion int
	Edit            ProviderModelEdit
}

// PreferenceUpdate 是一次偏好字段变更：Action 取 set / clear。
type PreferenceUpdate struct {
	Action string
	Value  string
}

// ModelPreferencePatch 是一次模型偏好写入；nil 字段表示不下发。
type ModelPreferencePatch struct {
	ReasoningEffort *PreferenceUpdate
	ServiceTier     *PreferenceUpdate
}

// ModelPreferenceRequest 是一次带版本前置条件的模型偏好写入。
type ModelPreferenceRequest struct {
	ProviderID      string
	ModelID         string
	ExpectedVersion int
	Patch           ModelPreferencePatch
}

// providerCreateParams 是交给写登记的透明参数形状。
type providerCreateParams struct {
	Op          string                  `json:"op"`
	ProviderID  string                  `json:"providerId"`
	DisplayName string                  `json:"displayName"`
	Connection  providerConnectionParam `json:"connection"`
	Models      []providerModelParam    `json:"models"`
	Enabled     bool                    `json:"enabled"`
}

type providerConnectionParam struct {
	DriverKind string            `json:"driverKind"`
	Auth       ProviderAuth      `json:"auth"`
	BaseURL    string            `json:"baseUrl"`
	Headers    map[string]string `json:"headers,omitempty"`
	Credential *credentialParam  `json:"credential,omitempty"`
}

type credentialParam struct {
	Action    string `json:"action"`
	Value     string `json:"value,omitempty"`
	Reference string `json:"reference,omitempty"`
}

type providerModelParam struct {
	ModelID          string                    `json:"modelId"`
	DisplayName      string                    `json:"displayName"`
	Status           string                    `json:"status"`
	Capabilities     providerCapabilitiesParam `json:"capabilities"`
	Modalities       []string                  `json:"modalityCapabilities,omitempty"`
	Reasoning        *ProviderModelReasoning   `json:"reasoning,omitempty"`
	Pricing          *ProviderModelPricing     `json:"pricing,omitempty"`
	ModelGuidePrompt string                    `json:"modelGuidePrompt,omitempty"`
}

type providerCapabilitiesParam struct {
	ContextWindow   *int64 `json:"contextWindow,omitempty"`
	MaxOutputTokens *int64 `json:"maxOutputTokens,omitempty"`
	ToolCalling     *bool  `json:"toolCalling,omitempty"`
	Vision          *bool  `json:"vision,omitempty"`
}

type providerUpdateParams struct {
	Op              string             `json:"op"`
	ProviderID      string             `json:"providerId"`
	ExpectedVersion int                `json:"expectedVersion"`
	Patch           providerPatchParam `json:"patch"`
}

type providerPatchParam struct {
	DisplayName *string               `json:"displayName,omitempty"`
	BaseURL     *string               `json:"baseUrl,omitempty"`
	Enabled     *bool                 `json:"enabled,omitempty"`
	Credential  *credentialParam      `json:"credential,omitempty"`
	Headers     []providerHeaderParam `json:"headers,omitempty"`
}

type providerHeaderParam struct {
	Action string `json:"action"`
	Name   string `json:"name"`
	Value  string `json:"value,omitempty"`
}

type providerProbeParams struct {
	ProviderID string                   `json:"providerId"`
	Connection *providerConnectionParam `json:"connection,omitempty"`
	Patch      *providerPatchParam      `json:"connectionPatch,omitempty"`
}

type providerModelsEnabledParams struct {
	Op              string   `json:"op"`
	ProviderID      string   `json:"providerId"`
	ExpectedVersion int      `json:"expectedVersion"`
	ModelIDs        []string `json:"modelIds"`
}

type providerModelEditParams struct {
	Op              string                 `json:"op"`
	ProviderID      string                 `json:"providerId"`
	ExpectedVersion int                    `json:"expectedVersion"`
	ModelID         string                 `json:"modelId"`
	Edit            providerModelEditParam `json:"edit"`
}

type providerModelEditParam struct {
	DisplayName          *string                 `json:"displayName,omitempty"`
	ContextWindow        *int64                  `json:"contextWindow,omitempty"`
	MaxOutputTokens      *int64                  `json:"maxOutputTokens,omitempty"`
	ToolCalling          *bool                   `json:"toolCalling,omitempty"`
	Vision               *bool                   `json:"vision,omitempty"`
	ModalityCapabilities *[]string               `json:"modalityCapabilities,omitempty"`
	Reasoning            *ProviderModelReasoning `json:"reasoning,omitempty"`
}

type providerModelResetParams struct {
	Op              string `json:"op"`
	ProviderID      string `json:"providerId"`
	ExpectedVersion int    `json:"expectedVersion"`
	ModelID         string `json:"modelId"`
}

type providerPreferencesParams struct {
	Op              string                  `json:"op"`
	ProviderID      string                  `json:"providerId"`
	ExpectedVersion int                     `json:"expectedVersion"`
	ModelID         string                  `json:"modelId"`
	Patch           providerPreferenceParam `json:"patch"`
}

type providerPreferenceParam struct {
	ReasoningEffort *preferenceUpdateParam `json:"reasoningEffort,omitempty"`
	ServiceTier     *preferenceUpdateParam `json:"serviceTier,omitempty"`
}

type preferenceUpdateParam struct {
	Action string `json:"action"`
	Value  string `json:"value,omitempty"`
}

// ProviderAdmin 是 provider 清单、详情与显式探测的读取面。
type ProviderAdmin interface {
	ReadProviders(ctx context.Context) (ProviderList, error)
	ReadProviderDetail(ctx context.Context, providerID string) (ProviderDetail, error)
	ProbeProvider(ctx context.Context, request ProviderProbeRequest) (ProviderProbeResult, error)
}

// RefreshProviders 读取 provider 概要清单并替换缓存。
func (m *Manager) RefreshProviders(ctx context.Context) (ProviderList, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := m.requireFeature(FeatureProviderList); err != nil {
		return ProviderList{}, err
	}
	if m.cfg.Providers == nil {
		return ProviderList{}, missing(FeatureProviderList, "未提供 provider 读取接口")
	}
	list, err := m.cfg.Providers.ReadProviders(ctx)
	if err != nil {
		return ProviderList{}, readFailure(FeatureProviderList, ErrProviderReadFailed, err)
	}
	if err := validateProviderList(list); err != nil {
		return ProviderList{}, err
	}
	m.mu.Lock()
	m.providers = cloneProviderList(list)
	m.providersOK = true
	m.mu.Unlock()
	return cloneProviderList(list), nil
}

// Providers 返回最近一次确认的 provider 清单。
func (m *Manager) Providers() (ProviderList, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.providersOK {
		return ProviderList{}, false
	}
	return cloneProviderList(m.providers), true
}

// ReadProvider 读取 provider 管理详情并缓存。
func (m *Manager) ReadProvider(ctx context.Context, providerID string) (ProviderDetail, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	providerID, err := normalizeProviderID(providerID)
	if err != nil {
		return ProviderDetail{}, err
	}
	if err := m.requireFeature(FeatureProviderManagement); err != nil {
		return ProviderDetail{}, err
	}
	return m.readProviderDetail(ctx, providerID)
}

// Provider 返回最近一次确认的 provider 详情。
func (m *Manager) Provider(providerID string) (ProviderDetail, bool) {
	providerID, ok := trimNonEmpty(providerID)
	if !ok {
		return ProviderDetail{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	detail, found := m.providerDetail[providerID]
	if !found {
		return ProviderDetail{}, false
	}
	return cloneProviderDetail(detail), true
}

func (m *Manager) readProviderDetail(ctx context.Context, providerID string) (ProviderDetail, error) {
	if m.cfg.Providers == nil {
		return ProviderDetail{}, missing(FeatureProviderManagement, "未提供 provider 读取接口")
	}
	detail, err := m.cfg.Providers.ReadProviderDetail(ctx, providerID)
	if err != nil {
		return ProviderDetail{}, readFailure(FeatureProviderManagement, ErrProviderReadFailed, err)
	}
	detail.ID = providerID
	if err := validateProviderDetail(detail); err != nil {
		return ProviderDetail{}, err
	}
	m.mu.Lock()
	m.providerDetail[providerID] = cloneProviderDetail(detail)
	m.mu.Unlock()
	return cloneProviderDetail(detail), nil
}

// ProbeProvider 显式探测 provider 的模型列表。探测只读，不写配置、不缓存凭据。
func (m *Manager) ProbeProvider(ctx context.Context, request ProviderProbeRequest) (ProviderProbeResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	providerID, err := normalizeProviderID(request.ProviderID)
	if err != nil {
		return ProviderProbeResult{}, err
	}
	if (request.Connection == nil) == (request.Patch == nil) {
		return ProviderProbeResult{}, fmt.Errorf("%w: 探测必须且只能给新连接或已有连接补丁", ErrInvalidProviderRequest)
	}
	if request.Connection != nil {
		if err := validateConnectionInput(*request.Connection); err != nil {
			return ProviderProbeResult{}, err
		}
	}
	if request.Patch != nil {
		if _, err := normalizeProviderPatch(*request.Patch); err != nil {
			return ProviderProbeResult{}, err
		}
	}
	if err := m.requireFeature(FeatureProviderProbe); err != nil {
		return ProviderProbeResult{}, err
	}
	if m.cfg.Providers == nil {
		return ProviderProbeResult{}, missing(FeatureProviderProbe, "未提供 provider 读取接口")
	}
	result, err := m.cfg.Providers.ProbeProvider(ctx, ProviderProbeRequest{
		ProviderID: providerID,
		Connection: request.Connection,
		Patch:      request.Patch,
	})
	if err != nil {
		return ProviderProbeResult{}, readFailure(FeatureProviderProbe, ErrProbeFailed, err)
	}
	for index := range result.Models {
		if err := validateModelInput(result.Models[index]); err != nil {
			return ProviderProbeResult{}, err
		}
	}
	return result, nil
}

// CreateProvider 新建 provider。成功后重读管理详情；不缓存凭据。
func (m *Manager) CreateProvider(ctx context.Context, request ProviderCreateRequest) (ProviderDetail, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	providerID, err := normalizeProviderID(request.ID)
	if err != nil {
		return ProviderDetail{}, err
	}
	if err := validateProviderCreateID(providerID); err != nil {
		return ProviderDetail{}, err
	}
	displayName, ok := trimNonEmpty(request.DisplayName)
	if !ok {
		return ProviderDetail{}, fmt.Errorf("%w: provider 缺少显示名", ErrInvalidProviderRequest)
	}
	if err := validateConnectionInput(request.Connection); err != nil {
		return ProviderDetail{}, err
	}
	models, err := normalizeModelInputs(request.Models)
	if err != nil {
		return ProviderDetail{}, err
	}
	if err := m.requireProviderWrite(FeatureProviderCreate); err != nil {
		return ProviderDetail{}, err
	}
	view, err := m.registerWrite(ctx, providerCreateParams{
		Op:          "provider.create",
		ProviderID:  providerID,
		DisplayName: displayName,
		Connection:  providerConnectionParamFrom(request.Connection),
		Models:      models,
		Enabled:     request.Enabled,
	})
	if err != nil {
		return ProviderDetail{}, err
	}
	if view.Status != write.StatusSucceeded {
		return ProviderDetail{}, fmt.Errorf("%w: %s", ErrProviderWriteFailed, failureReason(view))
	}
	return m.readProviderAfterWrite(ctx, providerID, "新建")
}

// UpdateProvider 更新 provider；本地缓存或服务端版本不符都返回冲突，不覆盖。
func (m *Manager) UpdateProvider(ctx context.Context, request ProviderUpdateRequest) (ProviderDetail, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	providerID, err := normalizeProviderID(request.ProviderID)
	if err != nil {
		return ProviderDetail{}, err
	}
	if request.ExpectedVersion <= 0 {
		return ProviderDetail{}, fmt.Errorf("%w: provider 版本前置条件不合法", ErrInvalidProviderRequest)
	}
	patch, err := normalizeProviderPatch(request.Patch)
	if err != nil {
		return ProviderDetail{}, err
	}
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	if err := m.requireProviderWrite(FeatureProviderUpdate); err != nil {
		return ProviderDetail{}, err
	}
	if cached, ok := m.Provider(providerID); ok && cached.Version != nil && *cached.Version != request.ExpectedVersion {
		return ProviderDetail{}, fmt.Errorf("%w: 期望版本 %d，本地最近确认版本 %d", ErrProviderConflict, request.ExpectedVersion, *cached.Version)
	}
	view, err := m.registerWrite(ctx, providerUpdateParams{
		Op:              "provider.update",
		ProviderID:      providerID,
		ExpectedVersion: request.ExpectedVersion,
		Patch:           patch,
	})
	if err != nil {
		return ProviderDetail{}, err
	}
	if view.Status != write.StatusSucceeded {
		return ProviderDetail{}, m.providerWriteFailure(ctx, providerID, request.ExpectedVersion, view, ErrProviderWriteFailed)
	}
	return m.readProviderAfterWrite(ctx, providerID, "更新")
}

// SetProviderEnabled 是 UpdateProvider 的启用状态便捷入口。
func (m *Manager) SetProviderEnabled(ctx context.Context, providerID string, expectedVersion int, enabled bool) (ProviderDetail, error) {
	return m.UpdateProvider(ctx, ProviderUpdateRequest{
		ProviderID:      providerID,
		ExpectedVersion: expectedVersion,
		Patch:           ProviderPatch{Enabled: &enabled},
	})
}

// SetModelsEnabled 写入 provider 的完整启用模型集合。它是显式全量集合，不做默认补齐。
func (m *Manager) SetModelsEnabled(ctx context.Context, providerID string, expectedVersion int, modelIDs []string) (ProviderDetail, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	providerID, err := normalizeProviderID(providerID)
	if err != nil {
		return ProviderDetail{}, err
	}
	if expectedVersion <= 0 {
		return ProviderDetail{}, fmt.Errorf("%w: provider 版本前置条件不合法", ErrInvalidProviderRequest)
	}
	normalized, err := normalizeModelIDs(modelIDs)
	if err != nil {
		return ProviderDetail{}, err
	}
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	if err := m.requireProviderWrite(FeatureProviderModelsSetEnabled); err != nil {
		return ProviderDetail{}, err
	}
	if cached, ok := m.Provider(providerID); ok && cached.Version != nil && *cached.Version != expectedVersion {
		return ProviderDetail{}, fmt.Errorf("%w: 期望版本 %d，本地最近确认版本 %d", ErrProviderConflict, expectedVersion, *cached.Version)
	}
	view, err := m.registerWrite(ctx, providerModelsEnabledParams{
		Op:              "provider.models.set_enabled",
		ProviderID:      providerID,
		ExpectedVersion: expectedVersion,
		ModelIDs:        normalized,
	})
	if err != nil {
		return ProviderDetail{}, err
	}
	if view.Status != write.StatusSucceeded {
		return ProviderDetail{}, m.providerWriteFailure(ctx, providerID, expectedVersion, view, ErrModelWriteFailed)
	}
	return m.readProviderAfterWrite(ctx, providerID, "设置启用模型")
}

// SetModelEnabled 读取当前启用集合后改为目标状态；版本不符时不发写请求。
func (m *Manager) SetModelEnabled(ctx context.Context, providerID, modelID string, expectedVersion int, enabled bool) (ProviderDetail, error) {
	providerID, err := normalizeProviderID(providerID)
	if err != nil {
		return ProviderDetail{}, err
	}
	modelID, ok := trimNonEmpty(modelID)
	if !ok {
		return ProviderDetail{}, fmt.Errorf("%w: 缺少模型标识", ErrInvalidProviderRequest)
	}
	detail, err := m.ReadProvider(ctx, providerID)
	if err != nil {
		return ProviderDetail{}, err
	}
	if detail.Version == nil {
		return ProviderDetail{}, fmt.Errorf("%w: provider %q 没有可写版本", ErrProviderNotEditable, providerID)
	}
	if *detail.Version != expectedVersion {
		return ProviderDetail{}, fmt.Errorf("%w: 期望版本 %d，本地最近确认版本 %d", ErrProviderConflict, expectedVersion, *detail.Version)
	}
	found := false
	enabledIDs := make([]string, 0, len(detail.Models))
	for _, model := range detail.Models {
		if model.ID == modelID {
			found = true
			if enabled {
				enabledIDs = append(enabledIDs, model.ID)
			}
			continue
		}
		if model.Enabled {
			enabledIDs = append(enabledIDs, model.ID)
		}
	}
	if !found {
		return ProviderDetail{}, fmt.Errorf("%w: provider %q 没有模型 %q", ErrInvalidProviderRequest, providerID, modelID)
	}
	if !enabled && !modelEnabled(detail.Models, modelID) {
		return detail, nil
	}
	if enabled && modelEnabled(detail.Models, modelID) {
		return detail, nil
	}
	return m.SetModelsEnabled(ctx, providerID, expectedVersion, enabledIDs)
}

// EditModel 编辑模型配置；版本冲突不覆盖。
func (m *Manager) EditModel(ctx context.Context, request ProviderModelEditRequest) (ProviderDetail, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	providerID, err := normalizeProviderID(request.ProviderID)
	if err != nil {
		return ProviderDetail{}, err
	}
	modelID, err := normalizeModelID(request.ModelID)
	if err != nil {
		return ProviderDetail{}, err
	}
	if request.ExpectedVersion <= 0 {
		return ProviderDetail{}, fmt.Errorf("%w: provider 版本前置条件不合法", ErrInvalidProviderRequest)
	}
	edit, err := normalizeModelEdit(request.Edit)
	if err != nil {
		return ProviderDetail{}, err
	}
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	if err := m.requireProviderWrite(FeatureProviderModelsEdit); err != nil {
		return ProviderDetail{}, err
	}
	if cached, ok := m.Provider(providerID); ok && cached.Version != nil && *cached.Version != request.ExpectedVersion {
		return ProviderDetail{}, fmt.Errorf("%w: 期望版本 %d，本地最近确认版本 %d", ErrProviderConflict, request.ExpectedVersion, *cached.Version)
	}
	view, err := m.registerWrite(ctx, providerModelEditParams{
		Op:              "provider.models.edit",
		ProviderID:      providerID,
		ExpectedVersion: request.ExpectedVersion,
		ModelID:         modelID,
		Edit:            edit,
	})
	if err != nil {
		return ProviderDetail{}, err
	}
	if view.Status != write.StatusSucceeded {
		return ProviderDetail{}, m.providerWriteFailure(ctx, providerID, request.ExpectedVersion, view, ErrModelWriteFailed)
	}
	return m.readProviderAfterWrite(ctx, providerID, "编辑模型")
}

// ResetModel 重置模型配置；版本冲突不覆盖。
func (m *Manager) ResetModel(ctx context.Context, providerID, modelID string, expectedVersion int) (ProviderDetail, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	providerID, err := normalizeProviderID(providerID)
	if err != nil {
		return ProviderDetail{}, err
	}
	modelID, err = normalizeModelID(modelID)
	if err != nil {
		return ProviderDetail{}, err
	}
	if expectedVersion <= 0 {
		return ProviderDetail{}, fmt.Errorf("%w: provider 版本前置条件不合法", ErrInvalidProviderRequest)
	}
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	if err := m.requireProviderWrite(FeatureProviderModelsReset); err != nil {
		return ProviderDetail{}, err
	}
	if cached, ok := m.Provider(providerID); ok && cached.Version != nil && *cached.Version != expectedVersion {
		return ProviderDetail{}, fmt.Errorf("%w: 期望版本 %d，本地最近确认版本 %d", ErrProviderConflict, expectedVersion, *cached.Version)
	}
	view, err := m.registerWrite(ctx, providerModelResetParams{
		Op:              "provider.models.reset",
		ProviderID:      providerID,
		ExpectedVersion: expectedVersion,
		ModelID:         modelID,
	})
	if err != nil {
		return ProviderDetail{}, err
	}
	if view.Status != write.StatusSucceeded {
		return ProviderDetail{}, m.providerWriteFailure(ctx, providerID, expectedVersion, view, ErrModelWriteFailed)
	}
	return m.readProviderAfterWrite(ctx, providerID, "重置模型")
}

// UpdateModelPreferences 写入模型偏好；版本冲突不覆盖。
func (m *Manager) UpdateModelPreferences(ctx context.Context, request ModelPreferenceRequest) (ProviderDetail, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	providerID, err := normalizeProviderID(request.ProviderID)
	if err != nil {
		return ProviderDetail{}, err
	}
	modelID, err := normalizeModelID(request.ModelID)
	if err != nil {
		return ProviderDetail{}, err
	}
	if request.ExpectedVersion <= 0 {
		return ProviderDetail{}, fmt.Errorf("%w: provider 版本前置条件不合法", ErrInvalidProviderRequest)
	}
	patch, err := normalizePreferencePatch(request.Patch)
	if err != nil {
		return ProviderDetail{}, err
	}
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	if err := m.requireProviderWrite(FeatureProviderPreferencesUpdate); err != nil {
		return ProviderDetail{}, err
	}
	if cached, ok := m.Provider(providerID); ok && cached.Version != nil && *cached.Version != request.ExpectedVersion {
		return ProviderDetail{}, fmt.Errorf("%w: 期望版本 %d，本地最近确认版本 %d", ErrProviderConflict, request.ExpectedVersion, *cached.Version)
	}
	view, err := m.registerWrite(ctx, providerPreferencesParams{
		Op:              "provider.preferences.update",
		ProviderID:      providerID,
		ExpectedVersion: request.ExpectedVersion,
		ModelID:         modelID,
		Patch:           patch,
	})
	if err != nil {
		return ProviderDetail{}, err
	}
	if view.Status != write.StatusSucceeded {
		return ProviderDetail{}, m.providerWriteFailure(ctx, providerID, request.ExpectedVersion, view, ErrModelWriteFailed)
	}
	return m.readProviderAfterWrite(ctx, providerID, "写入模型偏好")
}

// requireProviderWrite 要求 provider 管理详情与具体写方法都存在；缺一项即零请求。
func (m *Manager) requireProviderWrite(feature string) error {
	if err := m.requireFeature(FeatureProviderManagement); err != nil {
		return err
	}
	return m.requireFeature(feature)
}

// providerWriteFailure 区分版本冲突与普通失败。它只读一次当前详情，不重发。
func (m *Manager) providerWriteFailure(ctx context.Context, providerID string, expected int, view write.IntentView, base error) error {
	if current, err := m.readProviderDetail(ctx, providerID); err == nil {
		if current.Version != nil && *current.Version != expected {
			return fmt.Errorf("%w: 期望版本 %d，服务端当前版本 %d", ErrProviderConflict, expected, *current.Version)
		}
	}
	reason := failureReason(view)
	return fmt.Errorf("%w: %s", base, reason)
}

func (m *Manager) readProviderAfterWrite(ctx context.Context, providerID, action string) (ProviderDetail, error) {
	detail, err := m.readProviderDetail(ctx, providerID)
	if err != nil {
		return ProviderDetail{}, fmt.Errorf("%s已登记成功，但重读 provider 失败: %w", action, err)
	}
	return detail, nil
}

func normalizeProviderID(providerID string) (string, error) {
	trimmed, ok := trimNonEmpty(providerID)
	if !ok {
		return "", fmt.Errorf("%w: 缺少 provider 标识", ErrInvalidProviderRequest)
	}
	return trimmed, nil
}

// validateProviderCreateID 按真实服务端要求校验新建 provider 的标识：
// 规范的小写 UUID。已有 provider 的读取与更新不受此限制。
func validateProviderCreateID(providerID string) error {
	if strings.ToLower(providerID) != providerID {
		return fmt.Errorf("%w: 新建 provider 标识必须是小写 UUID", ErrInvalidProviderRequest)
	}
	parts := strings.Split(providerID, "-")
	if len(parts) != 5 {
		return fmt.Errorf("%w: 新建 provider 标识必须是小写 UUID", ErrInvalidProviderRequest)
	}
	for index, part := range parts {
		want := 12
		switch index {
		case 0:
			want = 8
		case 1, 2, 3:
			want = 4
		}
		if len(part) != want {
			return fmt.Errorf("%w: 新建 provider 标识必须是小写 UUID", ErrInvalidProviderRequest)
		}
		for _, char := range part {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
				return fmt.Errorf("%w: 新建 provider 标识必须是小写 UUID", ErrInvalidProviderRequest)
			}
		}
	}
	return nil
}

func normalizeModelID(modelID string) (string, error) {
	trimmed, ok := trimNonEmpty(modelID)
	if !ok {
		return "", fmt.Errorf("%w: 缺少模型标识", ErrInvalidProviderRequest)
	}
	return trimmed, nil
}

func validateConnectionInput(connection ProviderConnectionInput) error {
	if _, ok := trimNonEmpty(connection.DriverKind); !ok {
		return fmt.Errorf("%w: 连接缺少 driverKind", ErrInvalidProviderRequest)
	}
	if _, ok := trimNonEmpty(connection.BaseURL); !ok {
		return fmt.Errorf("%w: 连接缺少 baseUrl", ErrInvalidProviderRequest)
	}
	authType, ok := trimNonEmpty(connection.Auth.Type)
	if !ok {
		return fmt.Errorf("%w: 连接缺少认证类型", ErrInvalidProviderRequest)
	}
	switch authType {
	case "bearer", "anonymous", "aws_bedrock":
	case "api_key", "custom_header":
		if _, ok := trimNonEmpty(connection.Auth.HeaderName); !ok {
			return fmt.Errorf("%w: 认证类型 %q 缺少 headerName", ErrInvalidProviderRequest, authType)
		}
	default:
		return fmt.Errorf("%w: 未知认证类型 %q", ErrInvalidProviderRequest, authType)
	}
	for name, value := range connection.Headers {
		if _, ok := trimNonEmpty(name); !ok {
			return fmt.Errorf("%w: 请求头名称为空", ErrInvalidProviderRequest)
		}
		if value == "" {
			return fmt.Errorf("%w: 请求头 %q 的值为空", ErrInvalidProviderRequest, name)
		}
	}
	if connection.Credential != nil {
		switch connection.Credential.Action {
		case "set":
			if connection.Credential.Value == "" {
				return fmt.Errorf("%w: 凭据 set 缺少 value", ErrInvalidProviderRequest)
			}
		case "set_ref":
			if _, ok := trimNonEmpty(connection.Credential.Reference); !ok {
				return fmt.Errorf("%w: 凭据 set_ref 缺少 reference", ErrInvalidProviderRequest)
			}
		default:
			return fmt.Errorf("%w: 新建 provider 的凭据动作 %q 不合法", ErrInvalidProviderRequest, connection.Credential.Action)
		}
	}
	return nil
}

func providerConnectionParamFrom(connection ProviderConnectionInput) providerConnectionParam {
	param := providerConnectionParam{
		DriverKind: strings.TrimSpace(connection.DriverKind),
		Auth:       connection.Auth,
		BaseURL:    strings.TrimSpace(connection.BaseURL),
		Headers:    cloneStringMap(connection.Headers),
	}
	if connection.Credential != nil {
		param.Credential = &credentialParam{
			Action:    connection.Credential.Action,
			Value:     connection.Credential.Value,
			Reference: connection.Credential.Reference,
		}
	}
	return param
}

func normalizeProviderPatch(patch ProviderPatch) (providerPatchParam, error) {
	param := providerPatchParam{}
	empty := true
	if patch.DisplayName != nil {
		value, ok := trimNonEmpty(*patch.DisplayName)
		if !ok {
			return param, fmt.Errorf("%w: 显示名为空", ErrInvalidProviderRequest)
		}
		param.DisplayName = &value
		empty = false
	}
	if patch.BaseURL != nil {
		value, ok := trimNonEmpty(*patch.BaseURL)
		if !ok {
			return param, fmt.Errorf("%w: baseUrl 为空", ErrInvalidProviderRequest)
		}
		param.BaseURL = &value
		empty = false
	}
	if patch.Enabled != nil {
		value := *patch.Enabled
		param.Enabled = &value
		empty = false
	}
	if patch.Credential != nil {
		credential, err := normalizeCredentialUpdate(*patch.Credential)
		if err != nil {
			return param, err
		}
		param.Credential = &credential
		empty = false
	}
	if len(patch.Headers) > 0 {
		names := make(map[string]bool, len(patch.Headers))
		for _, header := range patch.Headers {
			name, ok := trimNonEmpty(header.Name)
			if !ok {
				return param, fmt.Errorf("%w: 请求头名称为空", ErrInvalidProviderRequest)
			}
			if names[name] {
				return param, fmt.Errorf("%w: 请求头 %q 重复", ErrInvalidProviderRequest, name)
			}
			names[name] = true
			item := providerHeaderParam{Action: header.Action, Name: name}
			switch header.Action {
			case "set":
				if header.Value == "" {
					return param, fmt.Errorf("%w: 请求头 %q 的值为空", ErrInvalidProviderRequest, name)
				}
				item.Value = header.Value
			case "remove":
				if header.Value != "" {
					return param, fmt.Errorf("%w: 请求头 remove 不应带值", ErrInvalidProviderRequest)
				}
			default:
				return param, fmt.Errorf("%w: 未知请求头动作 %q", ErrInvalidProviderRequest, header.Action)
			}
			param.Headers = append(param.Headers, item)
		}
		empty = false
	}
	if empty {
		return param, fmt.Errorf("%w: 更新补丁为空", ErrInvalidProviderRequest)
	}
	return param, nil
}

func normalizeCredentialUpdate(update ProviderCredentialUpdate) (credentialParam, error) {
	param := credentialParam{Action: update.Action}
	switch update.Action {
	case "keep", "clear":
		if update.Value != "" || update.Reference != "" {
			return param, fmt.Errorf("%w: 凭据动作 %q 不应带值", ErrInvalidProviderRequest, update.Action)
		}
	case "set":
		if update.Value == "" {
			return param, fmt.Errorf("%w: 凭据 set 缺少 value", ErrInvalidProviderRequest)
		}
		param.Value = update.Value
	case "set_ref":
		reference, ok := trimNonEmpty(update.Reference)
		if !ok {
			return param, fmt.Errorf("%w: 凭据 set_ref 缺少 reference", ErrInvalidProviderRequest)
		}
		param.Reference = reference
	default:
		return param, fmt.Errorf("%w: 未知凭据动作 %q", ErrInvalidProviderRequest, update.Action)
	}
	return param, nil
}

func normalizeModelInputs(inputs []ProviderModelInput) ([]providerModelParam, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	params := make([]providerModelParam, 0, len(inputs))
	seen := make(map[string]bool, len(inputs))
	for index, input := range inputs {
		modelID, ok := trimNonEmpty(input.ID)
		if !ok {
			return nil, fmt.Errorf("%w: 第 %d 个模型缺少标识", ErrInvalidProviderRequest, index)
		}
		if seen[modelID] {
			return nil, fmt.Errorf("%w: 模型标识重复 %q", ErrInvalidProviderRequest, modelID)
		}
		seen[modelID] = true
		param, err := normalizeModelInput(input)
		if err != nil {
			return nil, err
		}
		param.ModelID = modelID
		params = append(params, param)
	}
	return params, nil
}

func normalizeModelInput(input ProviderModelInput) (providerModelParam, error) {
	modelID, ok := trimNonEmpty(input.ID)
	if !ok {
		return providerModelParam{}, fmt.Errorf("%w: 模型缺少标识", ErrInvalidProviderRequest)
	}
	displayName, ok := trimNonEmpty(input.DisplayName)
	if !ok {
		return providerModelParam{}, fmt.Errorf("%w: 模型 %q 缺少显示名", ErrInvalidProviderRequest, modelID)
	}
	status, ok := trimNonEmpty(input.Status)
	if !ok {
		return providerModelParam{}, fmt.Errorf("%w: 模型 %q 缺少状态", ErrInvalidProviderRequest, modelID)
	}
	switch status {
	case "active", "beta", "deprecated":
	default:
		return providerModelParam{}, fmt.Errorf("%w: 模型 %q 的状态 %q 未知", ErrInvalidProviderRequest, modelID, status)
	}
	if input.ContextWindow != nil && *input.ContextWindow <= 0 {
		return providerModelParam{}, fmt.Errorf("%w: 模型 %q 的上下文窗口必须为正", ErrInvalidProviderRequest, modelID)
	}
	if input.MaxOutputTokens != nil && *input.MaxOutputTokens <= 0 {
		return providerModelParam{}, fmt.Errorf("%w: 模型 %q 的输出上限必须为正", ErrInvalidProviderRequest, modelID)
	}
	if err := validateModalities(input.Modalities); err != nil {
		return providerModelParam{}, err
	}
	if err := validateReasoning(input.Reasoning); err != nil {
		return providerModelParam{}, err
	}
	if err := validatePricing(input.Pricing); err != nil {
		return providerModelParam{}, err
	}
	return providerModelParam{
		ModelID:     modelID,
		DisplayName: displayName,
		Status:      status,
		Capabilities: providerCapabilitiesParam{
			ContextWindow:   input.ContextWindow,
			MaxOutputTokens: input.MaxOutputTokens,
			ToolCalling:     input.ToolCalling,
			Vision:          input.Vision,
		},
		Modalities:       append([]string(nil), input.Modalities...),
		Reasoning:        input.Reasoning,
		Pricing:          cloneProviderPricing(input.Pricing),
		ModelGuidePrompt: strings.TrimSpace(input.GuidePrompt),
	}, nil
}

func validateModelInput(input ProviderModelInput) error {
	_, err := normalizeModelInput(input)
	return err
}

func validateModalities(modalities []string) error {
	seen := make(map[string]bool, len(modalities))
	for _, modality := range modalities {
		value, ok := trimNonEmpty(modality)
		if !ok {
			return fmt.Errorf("%w: 模态标识为空", ErrInvalidProviderRequest)
		}
		if seen[value] {
			return fmt.Errorf("%w: 模态标识重复 %q", ErrInvalidProviderRequest, value)
		}
		seen[value] = true
	}
	return nil
}

func validatePricing(pricing *ProviderModelPricing) error {
	if pricing == nil {
		return nil
	}
	if pricing.InputTokenPrice != nil && *pricing.InputTokenPrice < 0 {
		return fmt.Errorf("%w: 输入单价为负", ErrInvalidProviderRequest)
	}
	if pricing.OutputTokenPrice != nil && *pricing.OutputTokenPrice < 0 {
		return fmt.Errorf("%w: 输出单价为负", ErrInvalidProviderRequest)
	}
	if pricing.Currency != "" {
		if _, ok := trimNonEmpty(pricing.Currency); !ok {
			return fmt.Errorf("%w: 计价货币为空", ErrInvalidProviderRequest)
		}
	}
	return nil
}

func validateReasoning(reasoning *ProviderModelReasoning) error {
	if reasoning == nil {
		return nil
	}
	seen := make(map[string]bool, len(reasoning.EffortOptions))
	for _, option := range reasoning.EffortOptions {
		value, ok := trimNonEmpty(option)
		if !ok {
			return fmt.Errorf("%w: 推理档位为空", ErrInvalidProviderRequest)
		}
		if seen[value] {
			return fmt.Errorf("%w: 推理档位重复 %q", ErrInvalidProviderRequest, value)
		}
		seen[value] = true
	}
	return nil
}

func normalizeModelEdit(edit ProviderModelEdit) (providerModelEditParam, error) {
	param := providerModelEditParam{}
	empty := true
	if edit.DisplayName != nil {
		value, ok := trimNonEmpty(*edit.DisplayName)
		if !ok {
			return param, fmt.Errorf("%w: 模型显示名为空", ErrInvalidProviderRequest)
		}
		param.DisplayName = &value
		empty = false
	}
	if edit.ContextWindow != nil {
		if *edit.ContextWindow <= 0 {
			return param, fmt.Errorf("%w: 上下文窗口必须为正", ErrInvalidProviderRequest)
		}
		param.ContextWindow = edit.ContextWindow
		empty = false
	}
	if edit.MaxOutputTokens != nil {
		if *edit.MaxOutputTokens <= 0 {
			return param, fmt.Errorf("%w: 输出上限必须为正", ErrInvalidProviderRequest)
		}
		param.MaxOutputTokens = edit.MaxOutputTokens
		empty = false
	}
	if edit.ToolCalling != nil {
		param.ToolCalling = edit.ToolCalling
		empty = false
	}
	if edit.Vision != nil {
		param.Vision = edit.Vision
		empty = false
	}
	if edit.ModalityCapabilities != nil {
		if err := validateModalities(*edit.ModalityCapabilities); err != nil {
			return param, err
		}
		param.ModalityCapabilities = edit.ModalityCapabilities
		empty = false
	}
	if edit.Reasoning != nil {
		if err := validateReasoning(edit.Reasoning); err != nil {
			return param, err
		}
		param.Reasoning = edit.Reasoning
		empty = false
	}
	if empty {
		return param, fmt.Errorf("%w: 模型编辑为空", ErrInvalidProviderRequest)
	}
	return param, nil
}

func normalizeModelIDs(modelIDs []string) ([]string, error) {
	normalized := make([]string, 0, len(modelIDs))
	seen := make(map[string]bool, len(modelIDs))
	for _, modelID := range modelIDs {
		value, ok := trimNonEmpty(modelID)
		if !ok {
			return nil, fmt.Errorf("%w: 启用模型标识为空", ErrInvalidProviderRequest)
		}
		if seen[value] {
			return nil, fmt.Errorf("%w: 启用模型标识重复 %q", ErrInvalidProviderRequest, value)
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	return normalized, nil
}

func normalizePreferencePatch(patch ModelPreferencePatch) (providerPreferenceParam, error) {
	param := providerPreferenceParam{}
	empty := true
	for _, update := range []struct {
		raw    *PreferenceUpdate
		target **preferenceUpdateParam
		label  string
	}{
		{patch.ReasoningEffort, &param.ReasoningEffort, "reasoningEffort"},
		{patch.ServiceTier, &param.ServiceTier, "serviceTier"},
	} {
		if update.raw == nil {
			continue
		}
		value, err := normalizePreferenceUpdate(*update.raw, update.label)
		if err != nil {
			return param, err
		}
		*update.target = &value
		empty = false
	}
	if empty {
		return param, fmt.Errorf("%w: 模型偏好补丁为空", ErrInvalidProviderRequest)
	}
	return param, nil
}

func normalizePreferenceUpdate(update PreferenceUpdate, label string) (preferenceUpdateParam, error) {
	param := preferenceUpdateParam{Action: update.Action}
	switch update.Action {
	case "set":
		value, ok := trimNonEmpty(update.Value)
		if !ok {
			return param, fmt.Errorf("%w: %s set 缺少值", ErrInvalidProviderRequest, label)
		}
		param.Value = value
	case "clear":
		if update.Value != "" {
			return param, fmt.Errorf("%w: %s clear 不应带值", ErrInvalidProviderRequest, label)
		}
	default:
		return param, fmt.Errorf("%w: %s 未知动作 %q", ErrInvalidProviderRequest, label, update.Action)
	}
	return param, nil
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneProviderList(list ProviderList) ProviderList {
	cloned := ProviderList{
		Providers: make([]ProviderSummary, 0, len(list.Providers)),
		Errors:    make([]ProviderRecordError, 0, len(list.Errors)),
	}
	for _, provider := range list.Providers {
		copied := provider
		copied.HeaderNames = append([]string(nil), provider.HeaderNames...)
		cloned.Providers = append(cloned.Providers, copied)
	}
	cloned.Errors = append(cloned.Errors, list.Errors...)
	return cloned
}

func cloneProviderDetail(detail ProviderDetail) ProviderDetail {
	cloned := detail
	cloned.Actions = append([]string(nil), detail.Actions...)
	if detail.Version != nil {
		value := *detail.Version
		cloned.Version = &value
	}
	if detail.Enabled != nil {
		value := *detail.Enabled
		cloned.Enabled = &value
	}
	if detail.Account != nil {
		account := *detail.Account
		account.LoginMethods = append([]string(nil), detail.Account.LoginMethods...)
		cloned.Account = &account
	}
	if detail.Connection != nil {
		connection := *detail.Connection
		connection.HeaderNames = append([]string(nil), detail.Connection.HeaderNames...)
		if detail.Connection.TimeoutMS != nil {
			value := *detail.Connection.TimeoutMS
			connection.TimeoutMS = &value
		}
		cloned.Connection = &connection
	}
	cloned.Models = make([]ProviderModelInfo, 0, len(detail.Models))
	for _, model := range detail.Models {
		copied := model
		copied.Actions = append([]string(nil), model.Actions...)
		copied.MissingLimits = append([]string(nil), model.MissingLimits...)
		copied.Capabilities = cloneProviderCapabilities(model.Capabilities)
		copied.Pricing = cloneProviderPricing(model.Pricing)
		cloned.Models = append(cloned.Models, copied)
	}
	if detail.Preferences != nil {
		cloned.Preferences = make(map[string]ModelPreferences, len(detail.Preferences))
		for key, value := range detail.Preferences {
			cloned.Preferences[key] = value
		}
	}
	return cloned
}

func cloneProviderPricing(pricing *ProviderModelPricing) *ProviderModelPricing {
	if pricing == nil {
		return nil
	}
	cloned := *pricing
	if pricing.InputTokenPrice != nil {
		value := *pricing.InputTokenPrice
		cloned.InputTokenPrice = &value
	}
	if pricing.OutputTokenPrice != nil {
		value := *pricing.OutputTokenPrice
		cloned.OutputTokenPrice = &value
	}
	return &cloned
}

func cloneProviderCapabilities(capabilities ProviderModelCapabilities) ProviderModelCapabilities {
	cloned := capabilities
	cloned.Modalities = append([]string(nil), capabilities.Modalities...)
	if capabilities.ContextWindow != nil {
		value := *capabilities.ContextWindow
		cloned.ContextWindow = &value
	}
	if capabilities.MaxOutputTokens != nil {
		value := *capabilities.MaxOutputTokens
		cloned.MaxOutputTokens = &value
	}
	if capabilities.ToolCalling != nil {
		value := *capabilities.ToolCalling
		cloned.ToolCalling = &value
	}
	if capabilities.Vision != nil {
		value := *capabilities.Vision
		cloned.Vision = &value
	}
	if capabilities.Reasoning != nil {
		reasoning := *capabilities.Reasoning
		reasoning.EffortOptions = append([]string(nil), capabilities.Reasoning.EffortOptions...)
		cloned.Reasoning = &reasoning
	}
	return cloned
}

func validateProviderList(list ProviderList) error {
	seen := make(map[string]bool, len(list.Providers))
	for _, provider := range list.Providers {
		id, ok := trimNonEmpty(provider.ID)
		if !ok {
			return fmt.Errorf("%w: provider 清单缺少标识", ErrMalformedProvider)
		}
		if seen[id] {
			return fmt.Errorf("%w: provider 标识重复 %q", ErrMalformedProvider, id)
		}
		seen[id] = true
		if provider.Version <= 0 {
			return fmt.Errorf("%w: provider %q 的版本不合法", ErrMalformedProvider, id)
		}
	}
	return nil
}

func validateProviderDetail(detail ProviderDetail) error {
	if _, ok := trimNonEmpty(detail.ID); !ok {
		return fmt.Errorf("%w: provider 详情缺少标识", ErrMalformedProvider)
	}
	if detail.Version != nil && *detail.Version <= 0 {
		return fmt.Errorf("%w: provider %q 的版本不合法", ErrMalformedProvider, detail.ID)
	}
	for _, model := range detail.Models {
		if _, ok := trimNonEmpty(model.ID); !ok {
			return fmt.Errorf("%w: provider %q 有模型缺少标识", ErrMalformedProvider, detail.ID)
		}
		if _, ok := trimNonEmpty(model.DisplayName); !ok {
			return fmt.Errorf("%w: provider %q 的模型 %q 缺少显示名", ErrMalformedProvider, detail.ID, model.ID)
		}
		switch model.Status {
		case "active", "beta", "deprecated":
		default:
			return fmt.Errorf("%w: provider %q 的模型 %q 状态 %q 未知", ErrMalformedProvider, detail.ID, model.ID, model.Status)
		}
	}
	return nil
}

func modelEnabled(models []ProviderModelInfo, modelID string) bool {
	for _, model := range models {
		if model.ID == modelID {
			return model.Enabled
		}
	}
	return false
}
