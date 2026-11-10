package settings

import (
	"context"
	"fmt"
)

// ModalityReason 是服务端给出的不可用原因。Detail 由服务端提供，客户端不解析。
type ModalityReason struct {
	Dimension string
	Detail    string
}

// ModalityModel 是一个模型引用。
type ModalityModel struct {
	ProviderID string
	ModelID    string
	// Source 是有效模型来源：default 或 automatic；空表示服务端没有给。
	Source string
}

// ModalityOption 是一个可用或不可用的模型选项。
type ModalityOption struct {
	ProviderID string
	ModelID    string
	Available  bool
	// Reason 只在不可用时给出。
	Reason *ModalityReason
}

// ModalityOperation 是一个模态操作的能力状态。
type ModalityOperation struct {
	CapabilityID string
	Category     string
	Operation    string
	Key          string
	Available    bool
	Reasons      []ModalityReason
	// Default 为空表示该操作使用自动解析。
	Default *ModalityOption
	// Effective 为空表示服务端没有给出有效模型。
	Effective *ModalityModel
	Options   []ModalityOption
}

// ModalityCategory 是一类模态能力。
type ModalityCategory struct {
	Category   string
	Available  bool
	Reasons    []ModalityReason
	Operations []ModalityOperation
}

// ModalityCapabilities 是服务端模态能力的只读状态。
//
// TaskSupported、RealtimeSupported 与 Revision 用指针保留「服务端没有给」与
// 「给了空值/假值」的区别，不拿零值冒充事实。
type ModalityCapabilities struct {
	Supported         bool
	CanConfigure      bool
	TaskSupported     *bool
	RealtimeSupported *bool
	Revision          *string
	Categories        []ModalityCategory
	// UnavailableReason 是服务端给出的整体不可用原因；空表示没有给。
	UnavailableReason string
}

// ModalityReader 读取模态能力状态。实现方只读，不提交任务、不打开实时传输。
type ModalityReader interface {
	ReadModalityCapabilities(ctx context.Context) (ModalityCapabilities, error)
}

// RefreshModality 读取模态能力状态并替换缓存。
//
// 服务端声明 supported=false 时返回快照与原因，不当作错误；服务端没有该方法
// 时返回能力缺失错误。事实不合法时整份不采用。
func (m *Manager) RefreshModality(ctx context.Context) (ModalityCapabilities, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := m.requireFeature(FeatureModalityCapabilities); err != nil {
		return ModalityCapabilities{}, err
	}
	if m.cfg.Modality == nil {
		return ModalityCapabilities{}, missing(FeatureModalityCapabilities, "未提供模态能力读取接口")
	}
	caps, err := m.cfg.Modality.ReadModalityCapabilities(ctx)
	if err != nil {
		return ModalityCapabilities{}, readFailure(FeatureModalityCapabilities, ErrModalityRead, err)
	}
	if err := validateModality(caps); err != nil {
		return ModalityCapabilities{}, err
	}
	m.mu.Lock()
	m.modality = cloneModality(caps)
	m.modalityOK = true
	m.mu.Unlock()
	return cloneModality(caps), nil
}

// Modality 返回最近一次确认的模态能力状态；第二个返回值表示是否读过。
func (m *Manager) Modality() (ModalityCapabilities, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.modalityOK {
		return ModalityCapabilities{}, false
	}
	return cloneModality(m.modality), true
}

func validateModality(caps ModalityCapabilities) error {
	operations := make(map[string]bool)
	categories := make(map[string]bool)
	for _, category := range caps.Categories {
		categoryName, ok := trimNonEmpty(category.Category)
		if !ok {
			return fmt.Errorf("%w: 模态类别缺少名称", ErrMalformedModality)
		}
		if categories[categoryName] {
			return fmt.Errorf("%w: 模态类别重复 %q", ErrMalformedModality, categoryName)
		}
		categories[categoryName] = true
		if err := validateReasons(category.Reasons); err != nil {
			return err
		}
		for _, operation := range category.Operations {
			capabilityID, ok := trimNonEmpty(operation.CapabilityID)
			if !ok {
				return fmt.Errorf("%w: 类别 %q 有操作缺少能力标识", ErrMalformedModality, categoryName)
			}
			if operations[capabilityID] {
				return fmt.Errorf("%w: 模态能力标识重复 %q", ErrMalformedModality, capabilityID)
			}
			operations[capabilityID] = true
			if _, ok := trimNonEmpty(operation.Operation); !ok {
				return fmt.Errorf("%w: 模态能力 %q 缺少操作名", ErrMalformedModality, capabilityID)
			}
			if err := validateReasons(operation.Reasons); err != nil {
				return err
			}
			if operation.Default != nil {
				if err := validateOption(*operation.Default); err != nil {
					return err
				}
			}
			if operation.Effective != nil {
				if err := validateModel(*operation.Effective); err != nil {
					return err
				}
				switch operation.Effective.Source {
				case "", "default", "automatic":
				default:
					return fmt.Errorf("%w: 模态能力 %q 的有效模型来源 %q 未知", ErrMalformedModality, capabilityID, operation.Effective.Source)
				}
			}
			for _, option := range operation.Options {
				if err := validateOption(option); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateOption(option ModalityOption) error {
	if err := validateModel(ModalityModel{ProviderID: option.ProviderID, ModelID: option.ModelID}); err != nil {
		return err
	}
	if option.Reason != nil {
		if _, ok := trimNonEmpty(option.Reason.Dimension); !ok {
			return fmt.Errorf("%w: 模态选项原因缺少维度", ErrMalformedModality)
		}
	}
	return nil
}

func validateModel(model ModalityModel) error {
	if _, ok := trimNonEmpty(model.ProviderID); !ok {
		return fmt.Errorf("%w: 模态模型缺少 provider 标识", ErrMalformedModality)
	}
	if _, ok := trimNonEmpty(model.ModelID); !ok {
		return fmt.Errorf("%w: 模态模型缺少模型标识", ErrMalformedModality)
	}
	return nil
}

func validateReasons(reasons []ModalityReason) error {
	for _, reason := range reasons {
		if _, ok := trimNonEmpty(reason.Dimension); !ok {
			return fmt.Errorf("%w: 模态原因缺少维度", ErrMalformedModality)
		}
	}
	return nil
}

func cloneModality(caps ModalityCapabilities) ModalityCapabilities {
	cloned := ModalityCapabilities{
		Supported:         caps.Supported,
		CanConfigure:      caps.CanConfigure,
		UnavailableReason: caps.UnavailableReason,
		Categories:        make([]ModalityCategory, 0, len(caps.Categories)),
	}
	if caps.TaskSupported != nil {
		value := *caps.TaskSupported
		cloned.TaskSupported = &value
	}
	if caps.RealtimeSupported != nil {
		value := *caps.RealtimeSupported
		cloned.RealtimeSupported = &value
	}
	if caps.Revision != nil {
		value := *caps.Revision
		cloned.Revision = &value
	}
	for _, category := range caps.Categories {
		copied := ModalityCategory{
			Category:   category.Category,
			Available:  category.Available,
			Reasons:    append([]ModalityReason(nil), category.Reasons...),
			Operations: make([]ModalityOperation, 0, len(category.Operations)),
		}
		for _, operation := range category.Operations {
			op := ModalityOperation{
				CapabilityID: operation.CapabilityID,
				Category:     operation.Category,
				Operation:    operation.Operation,
				Key:          operation.Key,
				Available:    operation.Available,
				Reasons:      append([]ModalityReason(nil), operation.Reasons...),
				Options:      cloneOptions(operation.Options),
			}
			if operation.Default != nil {
				option := cloneOption(*operation.Default)
				op.Default = &option
			}
			if operation.Effective != nil {
				model := *operation.Effective
				op.Effective = &model
			}
			copied.Operations = append(copied.Operations, op)
		}
		cloned.Categories = append(cloned.Categories, copied)
	}
	return cloned
}

func cloneOptions(options []ModalityOption) []ModalityOption {
	cloned := make([]ModalityOption, 0, len(options))
	for _, option := range options {
		cloned = append(cloned, cloneOption(option))
	}
	return cloned
}

func cloneOption(option ModalityOption) ModalityOption {
	cloned := option
	if option.Reason != nil {
		reason := *option.Reason
		cloned.Reason = &reason
	}
	return cloned
}
