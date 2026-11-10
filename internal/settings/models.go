package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// ProviderModel 是 provider 目录里的一个模型。
type ProviderModel struct {
	// ID 是模型标识，必填。
	ID string
	// Name 是展示名；服务端没给时为空。
	Name string
}

// Provider 是服务端声明的一个 provider。
type Provider struct {
	// ID 是 provider 标识，必填。
	ID string
	// Name 是展示名；服务端没给时为空。
	Name string
	// Enabled 是服务端声明的 provider 启用状态。
	Enabled bool
	// Models 是该 provider 声明的全部模型。
	Models []ProviderModel
	// EnabledModelIDs 是该 provider 允许选用的模型标识。
	EnabledModelIDs []string
}

// Selection 是当前选用的模型。字段全部来自服务端。
type Selection struct {
	ProviderID string
	ModelID    string
	// Revision 是服务端给出的选择版本；空表示服务端没有给，写入时不带前置条件。
	Revision string
	// ReasoningEffort 与 ServiceTier 是当前选择附带的可选偏好；空表示服务端没有给。
	ReasoningEffort string
	ServiceTier     string
}

// ModelCatalog 是一次 provider 目录读取的结果。
type ModelCatalog struct {
	Providers []Provider
	// Selection 为空表示服务端没有给出当前选择。
	Selection *Selection
}

// ModelCatalogReader 读取 provider 与模型目录。
type ModelCatalogReader interface {
	ReadModelCatalog(ctx context.Context) (ModelCatalog, error)
}

// AccountAvailability 是账号领域提供的可用模型视图；*account.Account 直接满足它。
// 模型可用性只按这份清单过滤，本包不自行推断账号状态。
type AccountAvailability interface {
	Models() account.ModelList
}

// ModelView 是一个模型在当前账号状态下的可选性与原因。
type ModelView struct {
	ID   string
	Name string
	// Enabled 表示模型在 provider 的启用模型清单里。
	Enabled bool
	// Selectable 表示当前账号状态下可以选用。
	Selectable bool
	// Reason 说明不可选的原因；可选时为空。
	Reason string
	// Selected 表示它是当前选择。
	Selected bool
}

// ProviderView 是一个 provider 及其模型的可选性。
type ProviderView struct {
	ID   string
	Name string
	// Enabled 是服务端声明的 provider 启用状态。
	Enabled bool
	Models  []ModelView
}

// SelectionView 是当前选择的可用性。Known 为假表示选择指向目录里不存在的
// provider 或模型；这时不改变选择，只说明无法确认。
type SelectionView struct {
	ProviderID      string
	ModelID         string
	Revision        string
	ReasoningEffort string
	ServiceTier     string
	Known           bool
	Selectable      bool
	Reason          string
}

// ModelSnapshot 是按账号状态过滤后的 provider 与模型视图。
type ModelSnapshot struct {
	Providers []ProviderView
	// Selection 为空表示服务端没有给出当前选择。
	Selection *SelectionView
	// AvailabilityReady 表示账号可用模型清单来自一次成功读取。
	AvailabilityReady bool
	// Reason 说明可用性不全或不可选的原因；空表示没有额外说明。
	Reason string
	// Catalog 是供调用方核对原始事实的目录副本。
	Catalog ModelCatalog
}

// RefreshModels 读取 provider 目录并按当前账号可用模型清单过滤。
//
// 账号可用模型视图缺失时返回能力缺失；账号清单尚未读取或为空时返回带原因的
// 快照（所有模型不可选），不返回可选的零值。目录事实不合法时整份不采用。
func (m *Manager) RefreshModels(ctx context.Context) (ModelSnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := m.requireFeature(FeatureModelProviders); err != nil {
		return ModelSnapshot{}, err
	}
	if m.cfg.Models == nil {
		return ModelSnapshot{}, missing(FeatureModelProviders, "未提供模型目录读取接口")
	}
	if m.cfg.Account == nil {
		return ModelSnapshot{}, missing(capabilityAccountModels, "未提供账号可用模型视图")
	}
	catalog, err := m.cfg.Models.ReadModelCatalog(ctx)
	if err != nil {
		return ModelSnapshot{}, readFailure(FeatureModelProviders, ErrModelsRead, err)
	}
	if err := validateCatalog(catalog); err != nil {
		return ModelSnapshot{}, err
	}
	snapshot := buildModelSnapshot(catalog, m.cfg.Account.Models())
	m.mu.Lock()
	m.models = cloneModelSnapshot(snapshot)
	m.modelsOK = true
	m.mu.Unlock()
	return cloneModelSnapshot(snapshot), nil
}

// Models 返回最近一次确认的模型视图；第二个返回值表示是否读过。
func (m *Manager) Models() (ModelSnapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.modelsOK {
		return ModelSnapshot{}, false
	}
	return cloneModelSnapshot(m.models), true
}

// buildModelSnapshot 把目录与账号可用模型清单合成为可选范围。匹配只按模型标识：
// 账号清单里出现该标识才算可选。匹配口径是接口约定，不在本地猜测别名。
func buildModelSnapshot(catalog ModelCatalog, availability account.ModelList) ModelSnapshot {
	available := make(map[string]bool, len(availability.Models))
	for _, model := range availability.Models {
		available[model.ID] = true
	}
	snapshot := ModelSnapshot{
		AvailabilityReady: availability.Ready,
		Reason:            availability.Reason,
		Catalog:           cloneCatalog(catalog),
	}
	if availability.Ready && len(availability.Models) == 0 && snapshot.Reason == "" {
		snapshot.Reason = "账号可用模型清单为空"
	}

	selectedProvider, selectedModel := "", ""
	if catalog.Selection != nil {
		selectedProvider = catalog.Selection.ProviderID
		selectedModel = catalog.Selection.ModelID
	}
	snapshot.Providers = make([]ProviderView, 0, len(catalog.Providers))
	selectedFound := false
	for _, provider := range catalog.Providers {
		enabledModels := make(map[string]bool, len(provider.EnabledModelIDs))
		for _, id := range provider.EnabledModelIDs {
			enabledModels[id] = true
		}
		view := ProviderView{
			ID:      provider.ID,
			Name:    provider.Name,
			Enabled: provider.Enabled,
			Models:  make([]ModelView, 0, len(provider.Models)),
		}
		for _, model := range provider.Models {
			isSelected := provider.ID == selectedProvider && model.ID == selectedModel
			selectable, reason := modelSelectability(provider, model, enabledModels, available, availability)
			view.Models = append(view.Models, ModelView{
				ID:         model.ID,
				Name:       model.Name,
				Enabled:    enabledModels[model.ID],
				Selectable: selectable,
				Reason:     reason,
				Selected:   isSelected,
			})
			if isSelected {
				selectedFound = true
			}
		}
		snapshot.Providers = append(snapshot.Providers, view)
	}

	if catalog.Selection != nil {
		selection := &SelectionView{
			ProviderID:      catalog.Selection.ProviderID,
			ModelID:         catalog.Selection.ModelID,
			Revision:        strings.TrimSpace(catalog.Selection.Revision),
			ReasoningEffort: catalog.Selection.ReasoningEffort,
			ServiceTier:     catalog.Selection.ServiceTier,
			Known:           selectedFound,
		}
		if !selectedFound {
			selection.Reason = "当前选择指向目录里不存在的模型"
		} else {
			selection.Selectable, selection.Reason = selectionSelectability(
				snapshot.Providers, catalog.Selection.ProviderID, catalog.Selection.ModelID,
			)
		}
		snapshot.Selection = selection
	}
	return snapshot
}

// modelSelectability 按 provider 启用状态、模型启用清单与账号可用清单判定。
func modelSelectability(
	provider Provider,
	model ProviderModel,
	enabledModels map[string]bool,
	available map[string]bool,
	availability account.ModelList,
) (bool, string) {
	switch {
	case !provider.Enabled:
		return false, "provider 已停用"
	case !enabledModels[model.ID]:
		return false, "模型未启用"
	case !availability.Ready:
		reason := availability.Reason
		if reason == "" {
			reason = "账号可用模型尚未读取"
		}
		return false, reason
	case !available[model.ID]:
		if availability.Reason != "" {
			return false, availability.Reason
		}
		return false, "当前账号状态不包含该模型"
	default:
		return true, ""
	}
}

// selectionSelectability 返回当前选择在合成视图里的可选性。
func selectionSelectability(providers []ProviderView, providerID, modelID string) (bool, string) {
	for _, provider := range providers {
		if provider.ID != providerID {
			continue
		}
		for _, model := range provider.Models {
			if model.ID != modelID {
				continue
			}
			if model.Selectable {
				return true, ""
			}
			if model.Reason != "" {
				return false, model.Reason
			}
			return false, "当前选择不可选"
		}
		return false, "当前选择指向 provider 目录里不存在的模型"
	}
	return false, "当前选择指向目录里不存在的 provider"
}

// SelectModelRequest 是一次模型选择写入。ExpectedRevision 是调用方从最近一次
// SelectionView.Revision 取得的版本前置条件；服务端没有给出选择版本时可以为空。
type SelectModelRequest struct {
	ProviderID       string
	ModelID          string
	ExpectedRevision string
}

// selectionWriteParams 是交给写登记的透明参数形状。集成层按 Op 映射到服务端方法，
// 并把 ExpectedRevision 映射为服务端的选择版本字段。
type selectionWriteParams struct {
	Op               string `json:"op"`
	ProviderID       string `json:"providerId"`
	ModelID          string `json:"modelId"`
	ExpectedRevision string `json:"expectedRevision,omitempty"`
}

// SelectModel 写入当前模型选择。
//
// 过程固定为：可用性校验 → 本地版本前置条件 → 经写登记登记一次
// KindManagementWrite（op=provider.select）→ 等待终态 → 成功后重读模型目录并
// 返回服务端确认的选择。它不做乐观更新、不自行重试；未登录、没有可选模型或
// 目标模型不可选时返回 ErrSelectionUnavailable 并说明原因。
func (m *Manager) SelectModel(ctx context.Context, request SelectModelRequest) (ModelSnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	providerID, ok := trimNonEmpty(request.ProviderID)
	if !ok {
		return ModelSnapshot{}, fmt.Errorf("%w: provider 标识为空", ErrMalformedModels)
	}
	modelID, ok := trimNonEmpty(request.ModelID)
	if !ok {
		return ModelSnapshot{}, fmt.Errorf("%w: 模型标识为空", ErrMalformedModels)
	}

	// 管理写入串行：与设置写入、组件安装共用同一条写入通道。
	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	if err := m.requireFeature(FeatureModelSelection); err != nil {
		return ModelSnapshot{}, err
	}
	if m.cfg.Account == nil {
		return ModelSnapshot{}, missing(capabilityAccountModels, "未提供账号可用模型视图")
	}
	if m.writer() == nil {
		return ModelSnapshot{}, fmt.Errorf("%w: %s", ErrWriteUnavailable, "未提供写登记")
	}

	catalogSnapshot, err := m.currentModelSnapshot(ctx)
	if err != nil {
		return ModelSnapshot{}, err
	}
	// 可用性按此刻的账号状态重算，不依赖上一次刷新时留下的可选性。
	snapshot := buildModelSnapshot(catalogSnapshot.Catalog, m.cfg.Account.Models())
	model, found := modelInSnapshot(snapshot, providerID, modelID)
	if !found {
		return ModelSnapshot{}, &SelectionUnavailableError{
			ProviderID: providerID,
			ModelID:    modelID,
			Reason:     "模型目录里没有该 provider 或模型",
		}
	}
	if !model.Selectable {
		reason := model.Reason
		if reason == "" {
			reason = "当前账号状态下该模型不可选"
		}
		return ModelSnapshot{}, &SelectionUnavailableError{
			ProviderID: providerID,
			ModelID:    modelID,
			Reason:     reason,
		}
	}

	expected := strings.TrimSpace(request.ExpectedRevision)
	cachedRevision := ""
	if snapshot.Selection != nil {
		cachedRevision = strings.TrimSpace(snapshot.Selection.Revision)
	}
	switch {
	case expected == "" && cachedRevision != "":
		return ModelSnapshot{}, fmt.Errorf("%w: 模型选择已有版本 %q，写入需要版本前置条件", ErrRevisionMissing, cachedRevision)
	case expected != "" && cachedRevision != "" && expected != cachedRevision:
		return ModelSnapshot{}, fmt.Errorf("%w: 期望版本 %q，本地最近确认版本 %q", ErrRevisionConflict, expected, cachedRevision)
	}

	view, err := m.registerWrite(ctx, selectionWriteParams{
		Op:               "provider.select",
		ProviderID:       providerID,
		ModelID:          modelID,
		ExpectedRevision: expected,
	})
	if err != nil {
		return ModelSnapshot{}, err
	}
	if view.Status != write.StatusSucceeded {
		return ModelSnapshot{}, m.selectionFailure(ctx, providerID, modelID, expected, view)
	}

	refreshed, err := m.RefreshModels(ctx)
	if err != nil {
		return ModelSnapshot{}, fmt.Errorf("选择已登记成功，但重读模型目录失败: %w", err)
	}
	return refreshed, nil
}

// currentModelSnapshot 取已确认的模型快照；还没读过时先读一次。
func (m *Manager) currentModelSnapshot(ctx context.Context) (ModelSnapshot, error) {
	if snapshot, ok := m.Models(); ok {
		return snapshot, nil
	}
	return m.RefreshModels(ctx)
}

// selectionFailure 在写入没有成功时区分版本冲突与普通失败。它只读一次模型目录，
// 不发送任何写请求，也不重试。
func (m *Manager) selectionFailure(ctx context.Context, providerID, modelID, expected string, view write.IntentView) error {
	refreshed, err := m.RefreshModels(ctx)
	if err == nil {
		current := ""
		if refreshed.Selection != nil {
			current = strings.TrimSpace(refreshed.Selection.Revision)
		}
		if current != expected {
			return fmt.Errorf("%w: 期望版本 %q，服务端当前版本 %q", ErrRevisionConflict, expected, current)
		}
	}
	reason := view.Reason
	if reason == "" {
		reason = string(view.Status)
	}
	return fmt.Errorf("%w: %s", ErrSelectionFailed, reason)
}

// modelInSnapshot 在合成视图里查找 provider 与模型。
func modelInSnapshot(snapshot ModelSnapshot, providerID, modelID string) (ModelView, bool) {
	for _, provider := range snapshot.Providers {
		if provider.ID != providerID {
			continue
		}
		for _, model := range provider.Models {
			if model.ID == modelID {
				return model, true
			}
		}
	}
	return ModelView{}, false
}

// validateCatalog 校验目录结构。标识缺失或重复都无法稳定选择，按不合法拒绝，
// 不保留半份清单。
func validateCatalog(catalog ModelCatalog) error {
	providers := make(map[string]bool, len(catalog.Providers))
	for index, provider := range catalog.Providers {
		id, ok := trimNonEmpty(provider.ID)
		if !ok {
			return fmt.Errorf("%w: 第 %d 个 provider 缺少标识", ErrMalformedModels, index)
		}
		if providers[id] {
			return fmt.Errorf("%w: provider 标识重复 %q", ErrMalformedModels, id)
		}
		providers[id] = true
		models := make(map[string]bool, len(provider.Models))
		for modelIndex, model := range provider.Models {
			modelID, ok := trimNonEmpty(model.ID)
			if !ok {
				return fmt.Errorf("%w: provider %q 的第 %d 个模型缺少标识", ErrMalformedModels, id, modelIndex)
			}
			if models[modelID] {
				return fmt.Errorf("%w: provider %q 的模型标识重复 %q", ErrMalformedModels, id, modelID)
			}
			models[modelID] = true
		}
		enabled := make(map[string]bool, len(provider.EnabledModelIDs))
		for enabledIndex, modelID := range provider.EnabledModelIDs {
			trimmed, ok := trimNonEmpty(modelID)
			if !ok {
				return fmt.Errorf("%w: provider %q 的第 %d 个启用模型标识为空", ErrMalformedModels, id, enabledIndex)
			}
			if enabled[trimmed] {
				return fmt.Errorf("%w: provider %q 的启用模型标识重复 %q", ErrMalformedModels, id, trimmed)
			}
			if !models[trimmed] {
				return fmt.Errorf("%w: provider %q 的启用模型 %q 不在模型清单里", ErrMalformedModels, id, trimmed)
			}
			enabled[trimmed] = true
		}
	}
	if catalog.Selection != nil {
		if _, ok := trimNonEmpty(catalog.Selection.ProviderID); !ok {
			return fmt.Errorf("%w: 当前选择缺少 provider 标识", ErrMalformedModels)
		}
		if _, ok := trimNonEmpty(catalog.Selection.ModelID); !ok {
			return fmt.Errorf("%w: 当前选择缺少模型标识", ErrMalformedModels)
		}
	}
	return nil
}

func cloneCatalog(catalog ModelCatalog) ModelCatalog {
	cloned := ModelCatalog{Providers: make([]Provider, 0, len(catalog.Providers))}
	for _, provider := range catalog.Providers {
		copied := Provider{
			ID:              provider.ID,
			Name:            provider.Name,
			Enabled:         provider.Enabled,
			Models:          append([]ProviderModel(nil), provider.Models...),
			EnabledModelIDs: append([]string(nil), provider.EnabledModelIDs...),
		}
		cloned.Providers = append(cloned.Providers, copied)
	}
	if catalog.Selection != nil {
		selection := *catalog.Selection
		cloned.Selection = &selection
	}
	return cloned
}

func cloneModelSnapshot(snapshot ModelSnapshot) ModelSnapshot {
	cloned := ModelSnapshot{
		AvailabilityReady: snapshot.AvailabilityReady,
		Reason:            snapshot.Reason,
		Catalog:           cloneCatalog(snapshot.Catalog),
		Providers:         make([]ProviderView, 0, len(snapshot.Providers)),
	}
	for _, provider := range snapshot.Providers {
		copied := ProviderView{
			ID:      provider.ID,
			Name:    provider.Name,
			Enabled: provider.Enabled,
			Models:  append([]ModelView(nil), provider.Models...),
		}
		cloned.Providers = append(cloned.Providers, copied)
	}
	if snapshot.Selection != nil {
		selection := *snapshot.Selection
		cloned.Selection = &selection
	}
	return cloned
}
