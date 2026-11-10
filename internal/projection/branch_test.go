package projection

import (
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// branchFixture 在主线开一个分支，并在分支里放两条事实。
func branchFixture() []protocol.RawEvent {
	return []protocol.RawEvent{
		ev("x1", 1, "branch", "create", `{"branch":"side","at":{"sequence":1},"prompt":"从第一轮开分支"}`),
		ev("x2", 2, "input", "message", `{"user":{"content":[{"type":"text","text":"分支里的话"}]}}`),
		ev("x3", 3, "turn", "start", `{"turn_id":"t9"}`),
		ev("x4", 4, "step", "text", `{"turn_id":"t9","request_id":"r9","delta":"分支回复"}`),
	}
}

// TestFactsAreBucketedByBranch 验证同一会话的分支事实各归各的桶。
func TestFactsAreBucketedByBranch(t *testing.T) {
	folder := newFolder(t)
	fixture := branchFixture()

	mustNotice(t, folder, mainBranch, fixture[0])
	for _, event := range fixture[1:] {
		mustNotice(t, folder, "side", event)
	}

	mainView := folder.View()
	if len(mainView.Rows) != 0 {
		t.Fatalf("主线不该看到分支的行: %+v", fingerprint(mainView))
	}
	if mainView.BranchStatus != BranchOpen {
		t.Fatalf("主线状态不对: %s", mainView.BranchStatus)
	}

	sideView, ok := folder.BranchView("side")
	if !ok {
		t.Fatalf("分支视图缺失")
	}
	if len(sideView.Rows) != 2 {
		t.Fatalf("分支行数应为 2，实际 %d: %+v", len(sideView.Rows), fingerprint(sideView))
	}
	if sideView.Rows[0].Text != "分支里的话" || sideView.Rows[1].Text != "分支回复" {
		t.Fatalf("分支行内容不对: %+v", sideView.Rows)
	}
	if sideView.Rows[1].TurnIndex != 1 {
		t.Fatalf("分支里的轮次应从 1 开始: %+v", sideView.Rows[1])
	}

	branches := folder.Branches()
	if len(branches) != 2 {
		t.Fatalf("分支目录应有主线与分支，实际 %+v", branches)
	}
	if branches[1].BranchID != "side" || branches[1].ParentID != mainBranch || branches[1].BranchPoint != 1 {
		t.Fatalf("分支目录项不对: %+v", branches[1])
	}
}

// TestEndedBranchDropsFactsAndKeepsHead 验证分支结束后不再接收新事实，头部不再变。
func TestEndedBranchDropsFactsAndKeepsHead(t *testing.T) {
	folder := newFolder(t)
	fixture := branchFixture()
	mustNotice(t, folder, mainBranch, fixture[0])
	for _, event := range fixture[1:] {
		mustNotice(t, folder, "side", event)
	}
	mustNotice(t, folder, "side", ev("x5", 5, "branch", "end", `{"reason":"done"}`))

	if err := folder.SelectBranch("side"); err != nil {
		t.Fatalf("选择分支失败: %v", err)
	}
	view, _ := folder.BranchView("side")
	if view.BranchStatus != BranchEnded {
		t.Fatalf("分支应为已结束: %s", view.BranchStatus)
	}
	head := view.Head

	rejections := folder.ApplyNotice("session.event",
		noticeJSON(t, "side", ev("x6", 6, "step", "text", `{"turn_id":"t9","request_id":"r9","delta":"结束后的"}`)))
	if len(rejections) != 1 || rejections[0].Reason != ReasonBranchEnded {
		t.Fatalf("结束后的事实应被丢弃并说明原因: %+v", rejections)
	}
	view, _ = folder.BranchView("side")
	if view.Head != head {
		t.Fatalf("结束后头部不该变: %d -> %d", head, view.Head)
	}
	for _, row := range view.Rows {
		if row.Text == "结束后的" {
			t.Fatalf("结束后的事实不该产生行")
		}
	}
	if folder.Selection() != SelectionBranch {
		t.Fatalf("分支只读状态不该变: %s", folder.Selection())
	}
	if diagnostics := folder.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("表内迁移不该记诊断: %+v", diagnostics)
	}

	// 已结束的分支仍然可以只读查看，再切回主线。
	if err := folder.SelectBranch(mainBranch); err != nil {
		t.Fatalf("切回主线失败: %v", err)
	}
	if folder.Selection() != SelectionMain {
		t.Fatalf("应回到主线: %s", folder.Selection())
	}
}

// TestSelectBranchRejectsUnknown 验证选择未知分支被拒绝并记诊断。
func TestSelectBranchRejectsUnknown(t *testing.T) {
	folder := newFolder(t)
	if err := folder.SelectBranch("nope"); err == nil {
		t.Fatalf("未知分支应被拒绝")
	}
	if folder.Selected() != mainBranch {
		t.Fatalf("拒绝后仍应停在主线")
	}
	diagnostics := folder.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Reason != ReasonUnknownBranch {
		t.Fatalf("应留下一条未知分支诊断: %+v", diagnostics)
	}
}

// TestSwitchBranchFromBranch 验证从一个分支切到另一个分支走两步合法迁移。
func TestSwitchBranchFromBranch(t *testing.T) {
	folder := newFolder(t)
	mustNotice(t, folder, mainBranch, branchFixture()[0])
	mustNotice(t, folder, mainBranch, ev("y1", 2, "branch", "create", `{"branch":"other","at":{"sequence":1}}`))

	if err := folder.SelectBranch("side"); err != nil {
		t.Fatalf("选择分支失败: %v", err)
	}
	if err := folder.SelectBranch("other"); err != nil {
		t.Fatalf("切换分支失败: %v", err)
	}
	if folder.Selected() != "other" || folder.Selection() != SelectionBranch {
		t.Fatalf("切换后应停在分支: %s %s", folder.Selected(), folder.Selection())
	}
	if diagnostics := folder.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("两步迁移都在表内，不该记诊断: %+v", diagnostics)
	}
}
