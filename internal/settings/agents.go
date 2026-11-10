package settings

import (
	"context"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// AgentProfile 是 agent 清单里的一个档案。真实 agent.list 不含版本字段；
// 本包按标识与名称做结构校验，不伪造版本。
type AgentProfile struct {
	ID          string
	Name        string
	Description string
	Sessions    int
	Default     bool
}

// SessionAgent 是会话分支当前采用的 agent 档案；AgentID 为空表示未采用。
type SessionAgent struct {
	SessionID string
	BranchID  string
	AgentID   string
}

// SessionAgentUpdateRequest 是一次会话 agent 档案更新。
//
// 真实 session.agent.update 没有服务端版本号，因此前置条件使用
// ExpectedAgentID：它必须等于写入前读到的当前 AgentID。空字符串表示期望当前
// 没有采用任何档案。它不是服务端 revision，但同样能阻止覆盖别人刚改过的选择。
type SessionAgentUpdateRequest struct {
	SessionID       string
	BranchID        string
	AgentID         string
	ExpectedAgentID string
}

// SessionAgentUpdateResult 是写入后按服务端读回的结果。
type SessionAgentUpdateResult struct {
	SessionID string
	BranchID  string
	AgentID   string
	Changed   bool
}

type sessionAgentUpdateParams struct {
	Op              string `json:"op"`
	SessionID       string `json:"sessionId"`
	BranchID        string `json:"branchId"`
	AgentID         string `json:"agentId"`
	ExpectedAgentID string `json:"expectedAgentId,omitempty"`
}

// AgentAdmin 是 agent 清单与会话当前 agent 的读取面。
type AgentAdmin interface {
	ListAgents(ctx context.Context) ([]AgentProfile, error)
	ReadSessionAgent(ctx context.Context, sessionID, branchID string) (SessionAgent, error)
	// SupportsSessionAgentGet 回答该会话是否声明 session.agent.get。
	SupportsSessionAgentGet(ctx context.Context, sessionID string) (bool, error)
	// SupportsSessionAgentUpdate 回答该会话是否声明 session.agent.update。
	SupportsSessionAgentUpdate(ctx context.Context, sessionID string) (bool, error)
}

// RefreshAgents 读取 agent 档案清单并缓存。
func (m *Manager) RefreshAgents(ctx context.Context) ([]AgentProfile, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// agent.list 是内建方法，真实 hello.features 不一定声明它；不按 hello
	// 预判，直接调用，unknown_method 由 readFailure 归为能力缺失。
	if m.cfg.Agents == nil {
		return nil, missing(FeatureAgentList, "未提供 agent 清单读取接口")
	}
	agents, err := m.cfg.Agents.ListAgents(ctx)
	if err != nil {
		return nil, readFailure(FeatureAgentList, ErrAgentReadFailed, err)
	}
	if err := validateAgentProfiles(agents); err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.agents = cloneAgentProfiles(agents)
	m.agentsOK = true
	m.mu.Unlock()
	return cloneAgentProfiles(agents), nil
}

// Agents 返回最近一次确认的 agent 档案清单。
func (m *Manager) Agents() ([]AgentProfile, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.agentsOK {
		return nil, false
	}
	return cloneAgentProfiles(m.agents), true
}

// ReadSessionAgent 读取会话分支当前采用的 agent。方法不存在时返回能力缺失。
func (m *Manager) ReadSessionAgent(ctx context.Context, sessionID, branchID string) (SessionAgent, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID, branchID, err := normalizeAgentScope(sessionID, branchID)
	if err != nil {
		return SessionAgent{}, err
	}
	if m.cfg.Agents == nil {
		return SessionAgent{}, missing(FeatureSessionAgentGet, "未提供会话 agent 读取接口")
	}
	supported, err := m.cfg.Agents.SupportsSessionAgentGet(ctx, sessionID)
	if err != nil {
		return SessionAgent{}, missing(FeatureSessionAgentGet, "读取会话方法表失败: "+err.Error())
	}
	if !supported {
		return SessionAgent{}, missing(FeatureSessionAgentGet, "会话未声明 session.agent.get")
	}
	return m.readSessionAgent(ctx, sessionID, branchID)
}

func (m *Manager) readSessionAgent(ctx context.Context, sessionID, branchID string) (SessionAgent, error) {
	if m.cfg.Agents == nil {
		return SessionAgent{}, missing(FeatureSessionAgentGet, "未提供会话 agent 读取接口")
	}
	agent, err := m.cfg.Agents.ReadSessionAgent(ctx, sessionID, branchID)
	if err != nil {
		return SessionAgent{}, readFailure(FeatureSessionAgentGet, ErrAgentReadFailed, err)
	}
	agent.SessionID = sessionID
	agent.BranchID = branchID
	return agent, nil
}

// UpdateSessionAgent 更新会话分支采用的 agent 档案。
//
// 真实方法没有服务端 revision；本包先读当前 AgentID，与 ExpectedAgentID 不符即
// 返回冲突且不发请求。写入经 KindManagementWrite 只登记一次，成功后重读确认。
// 缺 session.agent.update / session.agent.get 时返回能力缺失，零请求。
func (m *Manager) UpdateSessionAgent(ctx context.Context, request SessionAgentUpdateRequest) (SessionAgentUpdateResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID, branchID, err := normalizeAgentScope(request.SessionID, request.BranchID)
	if err != nil {
		return SessionAgentUpdateResult{}, err
	}
	agentID, ok := trimNonEmpty(request.AgentID)
	if !ok {
		return SessionAgentUpdateResult{}, fmt.Errorf("%w: 缺少 agent 标识", ErrInvalidAgentRequest)
	}
	expected, _ := trimNonEmpty(request.ExpectedAgentID)

	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	if m.cfg.Agents == nil {
		return SessionAgentUpdateResult{}, missing(FeatureSessionAgentUpdate, "未提供会话 agent 接口")
	}
	supportsGet, err := m.cfg.Agents.SupportsSessionAgentGet(ctx, sessionID)
	if err != nil {
		return SessionAgentUpdateResult{}, missing(FeatureSessionAgentGet, "读取会话方法表失败: "+err.Error())
	}
	if !supportsGet {
		return SessionAgentUpdateResult{}, missing(FeatureSessionAgentGet, "会话未声明 session.agent.get")
	}
	current, err := m.readSessionAgent(ctx, sessionID, branchID)
	if err != nil {
		return SessionAgentUpdateResult{}, err
	}
	if current.AgentID != expected {
		return SessionAgentUpdateResult{}, fmt.Errorf("%w: 期望当前 agent %q，服务端当前 %q", ErrAgentConflict, expected, current.AgentID)
	}
	supported, err := m.cfg.Agents.SupportsSessionAgentUpdate(ctx, sessionID)
	if err != nil {
		return SessionAgentUpdateResult{}, missing(FeatureSessionAgentUpdate, "读取会话方法表失败: "+err.Error())
	}
	if !supported {
		return SessionAgentUpdateResult{}, missing(FeatureSessionAgentUpdate, "会话未声明 session.agent.update")
	}
	if m.writer() == nil {
		return SessionAgentUpdateResult{}, fmt.Errorf("%w: %s", ErrWriteUnavailable, "未提供写登记")
	}
	view, err := m.registerWrite(ctx, sessionAgentUpdateParams{
		Op:              "session.agent.update",
		SessionID:       sessionID,
		BranchID:        branchID,
		AgentID:         agentID,
		ExpectedAgentID: expected,
	})
	if err != nil {
		return SessionAgentUpdateResult{}, err
	}
	if view.Status != write.StatusSucceeded {
		if after, readErr := m.readSessionAgent(ctx, sessionID, branchID); readErr == nil && after.AgentID != expected {
			return SessionAgentUpdateResult{}, fmt.Errorf("%w: 期望当前 agent %q，服务端当前 %q", ErrAgentConflict, expected, after.AgentID)
		}
		return SessionAgentUpdateResult{}, fmt.Errorf("%w: %s", ErrAgentWriteFailed, failureReason(view))
	}
	after, err := m.readSessionAgent(ctx, sessionID, branchID)
	if err != nil {
		return SessionAgentUpdateResult{}, fmt.Errorf("agent 档案已写入，但重读失败: %w", err)
	}
	if after.AgentID != agentID {
		return SessionAgentUpdateResult{}, fmt.Errorf("%w: 写入后服务端返回 agent %q，期望 %q", ErrAgentWriteFailed, after.AgentID, agentID)
	}
	return SessionAgentUpdateResult{
		SessionID: sessionID,
		BranchID:  branchID,
		AgentID:   after.AgentID,
		Changed:   expected != agentID,
	}, nil
}

func normalizeAgentScope(sessionID, branchID string) (string, string, error) {
	sessionID, ok := trimNonEmpty(sessionID)
	if !ok {
		return "", "", fmt.Errorf("%w: 缺少会话标识", ErrInvalidAgentRequest)
	}
	branchID, ok = trimNonEmpty(branchID)
	if !ok {
		return "", "", fmt.Errorf("%w: 缺少分支标识", ErrInvalidAgentRequest)
	}
	return sessionID, branchID, nil
}

func validateAgentProfiles(agents []AgentProfile) error {
	seen := make(map[string]bool, len(agents))
	for _, agent := range agents {
		id, ok := trimNonEmpty(agent.ID)
		if !ok {
			return fmt.Errorf("%w: agent 缺少标识", ErrMalformedAgent)
		}
		if seen[id] {
			return fmt.Errorf("%w: agent 标识重复 %q", ErrMalformedAgent, id)
		}
		seen[id] = true
		if _, ok := trimNonEmpty(agent.Name); !ok {
			return fmt.Errorf("%w: agent %q 缺少名称", ErrMalformedAgent, id)
		}
		if agent.Sessions < 0 {
			return fmt.Errorf("%w: agent %q 的会话数为负", ErrMalformedAgent, id)
		}
	}
	return nil
}

func cloneAgentProfiles(agents []AgentProfile) []AgentProfile {
	return append([]AgentProfile(nil), agents...)
}
