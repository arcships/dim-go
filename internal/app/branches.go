package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/catalog"
)

// branchSource 把会话投影已经推出的分支事实交给目录领域，用于读取分支目录与
// 查询切点。它只做映射，不另建一套分支推导。
type branchSource struct {
	app *App
}

// ReadBranches 实现 catalog.BranchSource。
func (s *branchSource) ReadBranches(
	ctx context.Context,
	sessionID string,
) ([]catalog.Branch, bool, error) {
	runtime, ok := s.app.session(sessionID)
	if !ok {
		return nil, false, ErrSessionNotOpen
	}
	infos := runtime.folder.Branches()
	branches := make([]catalog.Branch, 0, len(infos))
	for _, info := range infos {
		branches = append(branches, catalog.Branch{
			ID:          info.BranchID,
			ParentID:    info.ParentID,
			CutSequence: info.BranchPoint,
			Status:      catalog.BranchStatus(info.Status),
			Head:        info.Head,
		})
	}
	return branches, true, nil
}

// TurnCut 实现 catalog.BranchSource：切点状态直接采用投影的公开类型。
func (s *branchSource) TurnCut(sessionID, branchID string, turnIndex int) catalog.CutPoint {
	runtime, ok := s.app.session(sessionID)
	if !ok {
		return catalog.CutPoint{Status: catalog.CutNoBranch}
	}
	return runtime.folder.CutPoint(branchID, turnIndex)
}

// Branches 读取会话的分支目录；目录没读过时先读一次。
func (a *App) Branches(ctx context.Context, sessionID string) (catalog.BranchDirectory, error) {
	domain := a.Catalog()
	if domain == nil {
		return catalog.BranchDirectory{}, ErrNotConnected
	}
	return domain.RefreshBranches(ctx, sessionID)
}

// SelectBranch 切换查看的分支并返回新的会话视图；未知分支由投影拒绝。
func (a *App) SelectBranch(ctx context.Context, sessionID, branchID string) (SessionView, error) {
	runtime, ok := a.session(sessionID)
	if !ok {
		return SessionView{}, ErrSessionNotOpen
	}
	if err := runtime.folder.SelectBranch(branchID); err != nil {
		return SessionView{}, err
	}
	return a.viewOf(runtime), nil
}

// CreateBranch 从当前分支的某一轮结束处切出新分支。
func (a *App) CreateBranch(
	ctx context.Context,
	sessionID, parentID string,
	turnIndex int,
	prompt string,
) (catalog.Branch, error) {
	domain := a.Catalog()
	if domain == nil {
		return catalog.Branch{}, ErrNotConnected
	}
	if strings.TrimSpace(prompt) == "" {
		return catalog.Branch{}, fmt.Errorf("%w: 创建分支缺少提示词", catalog.ErrInvalidRequest)
	}
	return domain.CreateBranch(ctx, catalog.BranchCreateRequest{
		SessionID:      sessionID,
		ParentBranchID: parentID,
		TurnIndex:      turnIndex,
		Prompt:         prompt,
	})
}

// EndBranch 结束一个非主线分支。
func (a *App) EndBranch(ctx context.Context, sessionID, branchID string) (catalog.Branch, error) {
	domain := a.Catalog()
	if domain == nil {
		return catalog.Branch{}, ErrNotConnected
	}
	return domain.EndBranch(ctx, catalog.BranchEndRequest{SessionID: sessionID, BranchID: branchID})
}
