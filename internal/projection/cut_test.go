package projection

import (
	"reflect"
	"testing"
)

// TestTurnCutPointsForEachTurn 验证两轮会话里每一轮的切点都是该轮结束事实的序号。
func TestTurnCutPointsForEachTurn(t *testing.T) {
	events := turnFixture(2)
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(14, events...))

	// 夹具里每轮 7 条事实，第 7 条就是该轮的 turn/end。
	if events[6].Type != "turn" || events[6].Name != "end" {
		t.Fatalf("夹具形状变了: %+v", events[6])
	}
	if events[13].Type != "turn" || events[13].Name != "end" {
		t.Fatalf("夹具形状变了: %+v", events[13])
	}

	first, ok := folder.TurnCut("s-1", mainBranch, 1)
	if !ok || first != events[6].Sequence {
		t.Fatalf("第 1 轮切点应为 %d，实际 %d（ok=%v）", events[6].Sequence, first, ok)
	}
	second, ok := folder.TurnCut("s-1", mainBranch, 2)
	if !ok || second != events[13].Sequence {
		t.Fatalf("第 2 轮切点应为 %d，实际 %d（ok=%v）", events[13].Sequence, second, ok)
	}
	if first == second {
		t.Fatalf("两轮的切点不该相同: %d", first)
	}

	// 视图里的轮次也带着同一个结束序号，切点查询没有另算一份。
	view := folder.View()
	for _, turn := range view.Turns {
		if turn.Index == 1 && turn.EndSequence != first {
			t.Fatalf("视图里的结束序号与切点不一致: %+v", turn)
		}
	}
}

// TestTurnCutOutsideLoadedRange 验证轮次不在已加载范围时返回未加载，且查询不触发读取、
// 不改视图。
func TestTurnCutOutsideLoadedRange(t *testing.T) {
	events := turnFixture(2)
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(7, events[:7]...))

	before := folder.View()
	beforeHead := folder.Head(mainBranch)
	point := folder.CutPoint(mainBranch, 2)
	if point.Status != CutNotLoaded {
		t.Fatalf("未加载的轮次应返回未加载，实际 %+v", point)
	}
	if point.Usable() {
		t.Fatalf("未加载的轮次不该给出切点")
	}
	if _, ok := folder.TurnCut("s-1", mainBranch, 2); ok {
		t.Fatalf("未加载的轮次不该给出切点（接口口径）")
	}
	if !reflect.DeepEqual(before, folder.View()) {
		t.Fatalf("切点查询不该改视图")
	}
	if folder.Head(mainBranch) != beforeHead {
		t.Fatalf("切点查询不该推进头部")
	}

	// 向前翻页补齐之后可以取到。
	mustHistory(t, folder, mainBranch, page(14, events...))
	if cut, ok := folder.TurnCut("s-1", mainBranch, 2); !ok || cut != events[13].Sequence {
		t.Fatalf("补齐后应能取到第 2 轮切点，实际 %d（ok=%v）", cut, ok)
	}
}

// TestTurnCutWithoutEndFact 验证当前轮没有结束事实时明确不可用，拿到结束事实后可用。
func TestTurnCutWithoutEndFact(t *testing.T) {
	events := turnFixture(1)
	folder := newFolder(t)
	// 只到 model-output-end：本轮已开、还没收轮。
	mustHistory(t, folder, mainBranch, page(6, events[:6]...))

	point := folder.CutPoint(mainBranch, 1)
	if point.Status != CutNotEnded || point.Usable() {
		t.Fatalf("没结束的一轮应返回不可用，实际 %+v", point)
	}
	if point.TurnID == "" {
		t.Fatalf("已知轮次标识时应一并给出: %+v", point)
	}
	if _, ok := folder.TurnCut("s-1", mainBranch, 1); ok {
		t.Fatalf("没结束的一轮不该给出切点（接口口径）")
	}

	mustNotice(t, folder, mainBranch, events[6]) // turn/end
	point = folder.CutPoint(mainBranch, 1)
	if point.Status != CutReady || point.Sequence != events[6].Sequence {
		t.Fatalf("收轮后应给出切点，实际 %+v", point)
	}
}

// TestTurnCutOnEndedBranchStaysFixed 验证已结束分支上仍能取到切点，且头部保持固定。
func TestTurnCutOnEndedBranchStaysFixed(t *testing.T) {
	folder := newFolder(t)
	const side = "side"
	mustNotice(t, folder, mainBranch, ev("bc1", 1, "branch", "create", `{"branch":"side","at":{"sequence":1}}`))
	events := turnFixture(1)
	for _, event := range events {
		mustNotice(t, folder, side, event)
	}

	cut, ok := folder.TurnCut("s-1", side, 1)
	if !ok || cut != events[6].Sequence {
		t.Fatalf("分支切点应为 %d，实际 %d（ok=%v）", events[6].Sequence, cut, ok)
	}

	mustNotice(t, folder, side, ev("be1", 8, "branch", "end", `{"reason":"done"}`))
	head := branchHead(folder, side)
	if head != 8 {
		t.Fatalf("结束后的头部应为 8，实际 %d", head)
	}

	if cut, ok = folder.TurnCut("s-1", side, 1); !ok || cut != events[6].Sequence {
		t.Fatalf("已结束分支上仍应取到固定切点，实际 %d（ok=%v）", cut, ok)
	}

	// 结束后的事实被丢弃，切点与头部都不变。
	rejections := folder.ApplyNotice("session.event",
		noticeJSON(t, side, ev("t9", 9, "turn", "start", `{"turn_id":"t9"}`)))
	if len(rejections) != 1 || rejections[0].Reason != ReasonBranchEnded {
		t.Fatalf("结束后的事实应被丢弃: %+v", rejections)
	}
	if branchHead(folder, side) != head {
		t.Fatalf("已结束分支的头部不该变: %d -> %d", head, branchHead(folder, side))
	}
	if cut, ok = folder.TurnCut("s-1", side, 1); !ok || cut != events[6].Sequence {
		t.Fatalf("切点应保持固定，实际 %d（ok=%v）", cut, ok)
	}
}

// TestTurnCutRejectsForeignSessionAndUnknownBranch 验证会话与分支不匹配时不可用。
func TestTurnCutRejectsForeignSessionAndUnknownBranch(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(14, turnFixture(2)...))

	if _, ok := folder.TurnCut("s-2", mainBranch, 1); ok {
		t.Fatalf("别的会话不该拿到切点")
	}
	if point := folder.CutPoint("nope", 1); point.Status != CutNoBranch {
		t.Fatalf("未知分支应返回无分支，实际 %+v", point)
	}
	if _, ok := folder.TurnCut("s-1", "nope", 1); ok {
		t.Fatalf("未知分支不该拿到切点")
	}
}

// branchHead 取分支头部序号。
func branchHead(folder *Folder, branchID string) int64 {
	for _, branch := range folder.Branches() {
		if branch.BranchID == branchID {
			return branch.Head
		}
	}
	return 0
}
