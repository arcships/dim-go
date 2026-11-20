package uiwire

import (
	"context"
	"errors"

	"github.com/arcships/dimagent-mygo/internal/settings"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

var (
	errSettingsNotReady = errors.New("设置领域未就绪")
	errNoSession        = errors.New("没有打开的会话")
)

// 本文件把 agent 档案清单与会话当前档案（protocol v3 的 `agent.list` /
// `session.agent.get`）读成界面要的快照。界面每帧只读快照，读取在这里按需触发：
// 连接成功后、打开会话后、切换分支后与切换档案成功后各读一次。
//
// 不在每帧读：两个方法都是请求/回执，而且 `session.agent.get` 走会话流。
// 失败不抛到界面：读失败的原因放在快照里，选择器据此写原因或隐藏。

// refreshAgentConfig 读 agent 清单与当前会话分支采用的档案。
func (w *App) refreshAgentConfig(ctx context.Context, manager *settings.Manager) {
	profiles, err := manager.RefreshAgents(ctx)
	w.agentMu.Lock()
	if err != nil {
		w.agentErr = err.Error()
		w.agentKnown = false
	} else {
		w.agentErr = ""
		w.agentKnown = true
		w.agentProfiles = w.agentProfiles[:0]
		for _, profile := range profiles {
			w.agentProfiles = append(w.agentProfiles, ui.AgentConfigProfile{
				ID: profile.ID, Name: firstNonEmpty(profile.Name, profile.ID),
				Default: profile.Default,
			})
		}
	}
	w.agentMu.Unlock()

	sessionID := w.currentSession()
	branchID := w.currentBranch()
	if sessionID == "" {
		w.agentMu.Lock()
		w.agentCurrent = ""
		w.agentCurrentScope = ""
		w.agentCurrentKnown = false
		w.agentCanSelect = false
		w.agentMu.Unlock()
		return
	}
	supported, _ := manager.SupportsSessionAgentUpdate(ctx, sessionID)
	current, readErr := manager.ReadSessionAgent(ctx, sessionID, branchID)
	w.agentMu.Lock()
	defer w.agentMu.Unlock()
	w.agentCanSelect = supported
	if readErr != nil {
		// 读不到当前档案时清掉旧值：不拿上一个会话的档案顶替。
		w.agentCurrent = ""
		w.agentCurrentScope = sessionID + "\x00" + branchID
		w.agentCurrentKnown = false
		return
	}
	w.agentCurrent = current.AgentID
	w.agentCurrentScope = sessionID + "\x00" + branchID
	w.agentCurrentKnown = true
}

// AgentConfig 返回状态条选择器要的读数。会话或分支换了但还没重读时，当前档案
// 标成未知，不沿用上一个会话的值。
func (w *App) AgentConfig() ui.AgentConfigSnapshot {
	w.agentMu.Lock()
	defer w.agentMu.Unlock()
	out := ui.AgentConfigSnapshot{
		Profiles:  append([]ui.AgentConfigProfile(nil), w.agentProfiles...),
		Known:     w.agentKnown,
		Error:     w.agentErr,
		Current:   w.agentCurrent,
		CanSelect: w.agentCanSelect,
	}
	scope := w.currentSession() + "\x00" + w.currentBranch()
	out.CurrentKnown = w.agentCurrentKnown && w.agentCurrentScope == scope
	if !out.CurrentKnown {
		out.Current = ""
	}
	return out
}

// SelectAgent 切换当前会话分支的 agent 档案：用读到的当前值做前置条件，成功后
// 用回执更新快照。
func (w *App) SelectAgent(ctx context.Context, agentID, expected string) error {
	manager := w.A.Settings()
	if manager == nil {
		return errSettingsNotReady
	}
	sessionID := w.currentSession()
	if sessionID == "" {
		return errNoSession
	}
	result, err := manager.UpdateSessionAgent(ctx, settings.SessionAgentUpdateRequest{
		SessionID:       sessionID,
		BranchID:        w.currentBranch(),
		AgentID:         agentID,
		ExpectedAgentID: expected,
	})
	if err != nil {
		return err
	}
	w.agentMu.Lock()
	w.agentCurrent = result.AgentID
	w.agentCurrentScope = sessionID + "\x00" + w.currentBranch()
	w.agentCurrentKnown = true
	w.agentMu.Unlock()
	w.changed()
	return nil
}
