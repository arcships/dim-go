package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// CollaborationPolicyMode 是协作子代理的模型策略。
type CollaborationPolicyMode string

const (
	CollaborationPolicyAgent   CollaborationPolicyMode = "agent"
	CollaborationPolicyInherit CollaborationPolicyMode = "inherit"
	CollaborationPolicyFixed   CollaborationPolicyMode = "fixed"
)

// CollaborationModelPolicy 是协作模型策略；Fixed 时才有 Model。
type CollaborationModelPolicy struct {
	Mode            CollaborationPolicyMode
	ModelProviderID string
	ModelID         string
}

// CollaborationPolicySnapshot 是服务端确认的策略与版本。
type CollaborationPolicySnapshot struct {
	Revision string
	Policy   CollaborationModelPolicy
}

// CollaborationPolicyRequest 是一次带版本前置条件的策略写入。
type CollaborationPolicyRequest struct {
	ExpectedRevision string
	Policy           CollaborationModelPolicy
}

type collaborationPolicyParams struct {
	Op               string                  `json:"op"`
	ExpectedRevision string                  `json:"expectedRevision"`
	Policy           collaborationPolicyWire `json:"policy"`
}

type collaborationPolicyWire struct {
	Mode  CollaborationPolicyMode `json:"mode"`
	Model *collaborationModelWire `json:"model,omitempty"`
}

type collaborationModelWire struct {
	ProviderID string `json:"providerId"`
	ModelID    string `json:"modelId"`
}

// CollaborationReader 读取协作模型策略。
type CollaborationReader interface {
	ReadCollaborationPolicy(ctx context.Context) (CollaborationPolicySnapshot, error)
}

// RefreshCollaborationPolicy 读取协作模型策略并缓存。
func (m *Manager) RefreshCollaborationPolicy(ctx context.Context) (CollaborationPolicySnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := m.requireFeature(FeatureCollaborationPolicy); err != nil {
		return CollaborationPolicySnapshot{}, err
	}
	if m.cfg.Collaboration == nil {
		return CollaborationPolicySnapshot{}, missing(FeatureCollaborationPolicy, "未提供协作策略读取接口")
	}
	snapshot, err := m.cfg.Collaboration.ReadCollaborationPolicy(ctx)
	if err != nil {
		return CollaborationPolicySnapshot{}, readFailure(FeatureCollaborationPolicy, ErrCollaborationReadFailed, err)
	}
	if err := validateCollaborationSnapshot(snapshot); err != nil {
		return CollaborationPolicySnapshot{}, err
	}
	m.mu.Lock()
	m.collaboration = snapshot
	m.collaborationOK = true
	m.mu.Unlock()
	return snapshot, nil
}

// CollaborationPolicy 返回最近一次确认的协作策略。
func (m *Manager) CollaborationPolicy() (CollaborationPolicySnapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.collaborationOK {
		return CollaborationPolicySnapshot{}, false
	}
	return m.collaboration, true
}

// SetCollaborationPolicy 写入协作模型策略。
//
// 必须先读：本地缓存版本不符时不发请求；服务端拒绝后只读一次判断冲突。
// 成功后重读服务端策略，不做乐观更新；只登记一次，不重试。
func (m *Manager) SetCollaborationPolicy(ctx context.Context, request CollaborationPolicyRequest) (CollaborationPolicySnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	expected, ok := trimNonEmpty(request.ExpectedRevision)
	if !ok {
		return CollaborationPolicySnapshot{}, fmt.Errorf("%w: 缺少版本前置条件", ErrInvalidCollaborationRequest)
	}
	policy, err := normalizeCollaborationPolicy(request.Policy)
	if err != nil {
		return CollaborationPolicySnapshot{}, err
	}
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	if err := m.requireFeature(FeatureCollaborationSetPolicy); err != nil {
		return CollaborationPolicySnapshot{}, err
	}
	if err := m.requireFeature(FeatureCollaborationPolicy); err != nil {
		return CollaborationPolicySnapshot{}, err
	}
	if cached, ok := m.CollaborationPolicy(); ok && cached.Revision != expected {
		return CollaborationPolicySnapshot{}, fmt.Errorf("%w: 期望版本 %q，本地最近确认版本 %q", ErrCollaborationConflict, expected, cached.Revision)
	}
	if m.writer() == nil {
		return CollaborationPolicySnapshot{}, fmt.Errorf("%w: %s", ErrWriteUnavailable, "未提供写登记")
	}
	view, err := m.registerWrite(ctx, collaborationPolicyParams{
		Op:               "collaboration.set_model_policy",
		ExpectedRevision: expected,
		Policy:           policy,
	})
	if err != nil {
		return CollaborationPolicySnapshot{}, err
	}
	if view.Status != write.StatusSucceeded {
		if current, readErr := m.refreshCollaborationPolicy(ctx); readErr == nil && current.Revision != expected {
			return CollaborationPolicySnapshot{}, fmt.Errorf("%w: 期望版本 %q，服务端当前版本 %q", ErrCollaborationConflict, expected, current.Revision)
		}
		return CollaborationPolicySnapshot{}, fmt.Errorf("%w: %s", ErrCollaborationWriteFailed, failureReason(view))
	}
	refreshed, err := m.refreshCollaborationPolicy(ctx)
	if err != nil {
		return CollaborationPolicySnapshot{}, fmt.Errorf("协作策略已写入，但重读失败: %w", err)
	}
	return refreshed, nil
}

func (m *Manager) refreshCollaborationPolicy(ctx context.Context) (CollaborationPolicySnapshot, error) {
	if m.cfg.Collaboration == nil {
		return CollaborationPolicySnapshot{}, missing(FeatureCollaborationPolicy, "未提供协作策略读取接口")
	}
	snapshot, err := m.cfg.Collaboration.ReadCollaborationPolicy(ctx)
	if err != nil {
		return CollaborationPolicySnapshot{}, readFailure(FeatureCollaborationPolicy, ErrCollaborationReadFailed, err)
	}
	if err := validateCollaborationSnapshot(snapshot); err != nil {
		return CollaborationPolicySnapshot{}, err
	}
	m.mu.Lock()
	m.collaboration = snapshot
	m.collaborationOK = true
	m.mu.Unlock()
	return snapshot, nil
}

func normalizeCollaborationPolicy(policy CollaborationModelPolicy) (collaborationPolicyWire, error) {
	mode := CollaborationPolicyMode(strings.TrimSpace(string(policy.Mode)))
	wire := collaborationPolicyWire{Mode: mode}
	switch mode {
	case CollaborationPolicyAgent, CollaborationPolicyInherit:
		if policy.ModelProviderID != "" || policy.ModelID != "" {
			return wire, fmt.Errorf("%w: %s 策略不能带模型", ErrInvalidCollaborationRequest, mode)
		}
	case CollaborationPolicyFixed:
		provider, ok := trimNonEmpty(policy.ModelProviderID)
		if !ok {
			return wire, fmt.Errorf("%w: fixed 策略缺少 provider 标识", ErrInvalidCollaborationRequest)
		}
		model, ok := trimNonEmpty(policy.ModelID)
		if !ok {
			return wire, fmt.Errorf("%w: fixed 策略缺少模型标识", ErrInvalidCollaborationRequest)
		}
		wire.Model = &collaborationModelWire{ProviderID: provider, ModelID: model}
	default:
		return wire, fmt.Errorf("%w: 未知协作策略 %q", ErrInvalidCollaborationRequest, policy.Mode)
	}
	return wire, nil
}

func validateCollaborationSnapshot(snapshot CollaborationPolicySnapshot) error {
	if _, ok := trimNonEmpty(snapshot.Revision); !ok {
		return fmt.Errorf("%w: 缺少版本", ErrMalformedCollaboration)
	}
	if _, err := normalizeCollaborationPolicy(snapshot.Policy); err != nil {
		return fmt.Errorf("%w: %v", ErrMalformedCollaboration, err)
	}
	return nil
}
