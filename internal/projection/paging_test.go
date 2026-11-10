package projection

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// turnFixture 造若干轮对话，每轮 7 条事实：输入、消费、开轮、模型开始、文本、
// 模型结束、收轮。
func turnFixture(turns int) []protocol.RawEvent {
	var events []protocol.RawEvent
	for turn := 1; turn <= turns; turn++ {
		base := int64((turn - 1) * 7)
		events = append(events,
			ev(fmt.Sprintf("i%d", turn), base+1, "input", "message",
				fmt.Sprintf(`{"mode":"queue","user":{"content":[{"type":"text","text":"第 %d 句"}]}}`, turn)),
			ev(fmt.Sprintf("c%d", turn), base+2, "input", "consumed", fmt.Sprintf(`{"input_id":"i%d"}`, turn)),
			ev(fmt.Sprintf("t%d", turn), base+3, "turn", "start", fmt.Sprintf(`{"turn_id":"t%d"}`, turn)),
			ev(fmt.Sprintf("m%d", turn), base+4, "step", "model-start",
				fmt.Sprintf(`{"turn_id":"t%d","request_id":"r%d"}`, turn, turn)),
			ev(fmt.Sprintf("x%d", turn), base+5, "step", "text",
				fmt.Sprintf(`{"turn_id":"t%d","request_id":"r%d","delta":"第 %d 答"}`, turn, turn, turn)),
			ev(fmt.Sprintf("o%d", turn), base+6, "step", "model-output-end",
				fmt.Sprintf(`{"turn_id":"t%d","request_id":"r%d","status":"completed"}`, turn, turn)),
			ev(fmt.Sprintf("e%d", turn), base+7, "turn", "end", fmt.Sprintf(`{"turn_id":"t%d","status":"completed"}`, turn)),
		)
	}
	return events
}

// turnPage 取某一轮的事实做成一页历史。through 是会话头部序号，done 表示没有更早的轮次。
func turnPage(events []protocol.RawEvent, turn, totalTurns int, through int64, done bool) protocol.SessionHistoryResult {
	start := (turn - 1) * 7
	page := protocol.SessionHistoryResult{
		Events:  append([]protocol.RawEvent(nil), events[start:start+7]...),
		Through: through,
		Done:    done,
	}
	turnStart := events[start].Sequence
	page.TurnStart = &turnStart
	page.NextAfter = through
	return page
}

// TestBackwardPagingCoversAllTurnsWithoutDuplicates 连续向前翻三页：行与"一次读全"
// 完全一致，没有重复也没有缺口，最后锚点停在最早一轮、hasOlder 为假。
func TestBackwardPagingCoversAllTurnsWithoutDuplicates(t *testing.T) {
	events := turnFixture(4)
	const through = int64(28)

	reference := newFolder(t)
	mustHistory(t, reference, mainBranch, page(through, events...))
	referenceView := reference.View()

	folder := newFolder(t)
	// 最近一轮打开。
	if rejections := folder.ApplyFirstPage(mainBranch, turnPage(events, 4, 4, through, false)); len(rejections) > 0 {
		t.Fatalf("首屏折叠出现丢弃: %+v", rejections[0])
	}
	if window := folder.Window(mainBranch); window.Anchor != 22 || !window.HasOlder {
		t.Fatalf("首屏后分页位置不对: %+v", window)
	}
	if got := len(folder.View().Rows); got != 2 {
		t.Fatalf("首屏应只有最近一轮的两行，实际 %d", got)
	}

	for turn := 3; turn >= 1; turn-- {
		params, ok := folder.OlderPageRequest(mainBranch, 50)
		if !ok {
			t.Fatalf("第 %d 轮前应还能向前翻页", turn)
		}
		// 游标是"已装载的最早一轮"的起点：第 3 页之前已装载第 4 轮，因此取第 4 轮的起点。
		wantBefore := events[turn*7].Sequence
		if params.Before == nil || *params.Before != wantBefore {
			t.Fatalf("第 %d 轮的游标应为 %d，实际 %+v", turn, wantBefore, params)
		}
		if params.Limit == nil || *params.Limit != 50 || params.BranchID != mainBranch {
			t.Fatalf("翻页参数不完整: %+v", params)
		}
		result := turnPage(events, turn, 4, through, turn == 1)
		if rejections := folder.ApplyOlderPage(mainBranch, result); len(rejections) > 0 {
			t.Fatalf("第 %d 轮翻页出现丢弃: %+v", turn, rejections[0])
		}
	}

	if _, ok := folder.OlderPageRequest(mainBranch, 50); ok {
		t.Fatalf("最早一轮之后不该再有更早的页")
	}
	window := folder.Window(mainBranch)
	if window.Anchor != 1 || window.HasOlder {
		t.Fatalf("翻到最早一轮后分页位置不对: %+v", window)
	}

	got := folder.View()
	if !reflect.DeepEqual(rowIDs(got), rowIDs(referenceView)) {
		t.Fatalf("向前翻页的行标识与一次读全不一致\n翻页: %v\n读全: %v", rowIDs(got), rowIDs(referenceView))
	}
	if !reflect.DeepEqual(fingerprint(got), fingerprint(referenceView)) {
		t.Fatalf("向前翻页的行内容与一次读全不一致\n翻页: %+v\n读全: %+v", fingerprint(got), fingerprint(referenceView))
	}
	if len(got.Rows) != len(referenceView.Rows) {
		t.Fatalf("行数不一致: %d vs %d", len(got.Rows), len(referenceView.Rows))
	}
	if !reflect.DeepEqual(got.Turns, referenceView.Turns) {
		t.Fatalf("轮索引不一致\n翻页: %+v\n读全: %+v", got.Turns, referenceView.Turns)
	}
	if got.Unlanded != 0 {
		t.Fatalf("历史页覆盖的行不该留作尾部: %d", got.Unlanded)
	}
}

// TestOlderPageDropsEventsOutsideWindow 验证越界事实（序号不小于本次游标）被丢弃，
// 不进视图也不改锚点。
func TestOlderPageDropsEventsOutsideWindow(t *testing.T) {
	events := turnFixture(3)
	const through = int64(21)
	folder := newFolder(t)
	if rejections := folder.ApplyFirstPage(mainBranch, turnPage(events, 3, 3, through, false)); len(rejections) > 0 {
		t.Fatalf("首屏折叠出现丢弃: %+v", rejections[0])
	}
	before := folder.View()
	anchor := folder.Window(mainBranch).Anchor

	// 这一页把最近一轮的事实又带回来了：越界。
	bad := turnPage(events, 3, 3, through, false)
	bad.Events = append(bad.Events, events[14:21]...)
	rejections := folder.ApplyOlderPage(mainBranch, bad)
	if len(rejections) == 0 {
		t.Fatalf("越界事实应被丢弃")
	}
	for _, rejection := range rejections {
		if rejection.Reason != ReasonOutOfWindow {
			t.Fatalf("越界原因不对: %+v", rejection)
		}
	}
	after := folder.View()
	if !reflect.DeepEqual(fingerprint(before), fingerprint(after)) {
		t.Fatalf("越界事实不该进视图: %+v", fingerprint(after))
	}
	if folder.Window(mainBranch).Anchor != anchor {
		t.Fatalf("越界事实不该改锚点")
	}
}

// TestRepeatedOlderPageStopsWithoutDuplicates 验证重复页不产生重复行、不挪锚点，
// 并且停止继续翻页（记一条诊断）。
func TestRepeatedOlderPageStopsWithoutDuplicates(t *testing.T) {
	events := turnFixture(3)
	const through = int64(21)
	folder := newFolder(t)
	if rejections := folder.ApplyFirstPage(mainBranch, turnPage(events, 3, 3, through, false)); len(rejections) > 0 {
		t.Fatalf("首屏折叠出现丢弃: %+v", rejections[0])
	}

	older := turnPage(events, 2, 3, through, false)
	if rejections := folder.ApplyOlderPage(mainBranch, older); len(rejections) > 0 {
		t.Fatalf("第二页折叠出现丢弃: %+v", rejections[0])
	}
	rows := len(folder.View().Rows)
	anchor := folder.Window(mainBranch).Anchor
	if anchor != 8 {
		t.Fatalf("锚点应推进到第 2 轮: %d", anchor)
	}

	// 重复页里的事实都落在这次请求的窗口之外：按越界丢弃，但绝不产生第二行。
	rejections := folder.ApplyOlderPage(mainBranch, older)
	for _, rejection := range rejections {
		if rejection.Reason != ReasonOutOfWindow {
			t.Fatalf("重复页的原因应越界: %+v", rejection)
		}
	}
	if got := len(folder.View().Rows); got != rows {
		t.Fatalf("重复页产生了重复行: %d -> %d", rows, got)
	}
	window := folder.Window(mainBranch)
	if window.Anchor != anchor {
		t.Fatalf("重复页挪动了锚点: %d -> %d", anchor, window.Anchor)
	}
	if window.HasOlder {
		t.Fatalf("重复页之后应停止向前翻页")
	}
	diagnostics := folder.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Machine != machinePaging {
		t.Fatalf("应记一条翻页诊断: %+v", diagnostics)
	}
}

// TestFirstPageWithoutOlderTurns 验证首屏就说完时不再有向前翻页。
func TestFirstPageWithoutOlderTurns(t *testing.T) {
	events := turnFixture(1)
	folder := newFolder(t)
	if rejections := folder.ApplyFirstPage(mainBranch, turnPage(events, 1, 1, 7, true)); len(rejections) > 0 {
		t.Fatalf("首屏折叠出现丢弃: %+v", rejections[0])
	}
	if folder.HasOlder(mainBranch) {
		t.Fatalf("只有一轮时不该还有更早的页")
	}
	if _, ok := folder.OlderPageRequest(mainBranch, 50); ok {
		t.Fatalf("没有更早的页时不该给出请求")
	}
}

// TestAnchorDoesNotMoveWithLiveFacts 验证锚点只由历史页推进，实时事实不动它。
func TestAnchorDoesNotMoveWithLiveFacts(t *testing.T) {
	events := turnFixture(2)
	const through = int64(14)
	folder := newFolder(t)
	if rejections := folder.ApplyFirstPage(mainBranch, turnPage(events, 2, 2, through, false)); len(rejections) > 0 {
		t.Fatalf("首屏折叠出现丢弃: %+v", rejections[0])
	}
	before := folder.Window(mainBranch)

	mustNotice(t, folder, mainBranch, ev("live1", 15, "input", "message",
		`{"mode":"queue","user":{"content":[{"type":"text","text":"实时一句"}]}}`))
	mustNotice(t, folder, mainBranch, ev("live2", 16, "turn", "start", `{"turn_id":"t3"}`))

	after := folder.Window(mainBranch)
	if after != before {
		t.Fatalf("实时事实不该改分页位置: %+v -> %+v", before, after)
	}
	if !after.HasOlder {
		t.Fatalf("实时事实不该改变 hasOlder")
	}
}

// TestDisconnectResetsWindow 验证断线重开回到"未装载"，重设锚点重读。
func TestDisconnectResetsWindow(t *testing.T) {
	events := turnFixture(2)
	folder := newFolder(t)
	if rejections := folder.ApplyFirstPage(mainBranch, turnPage(events, 2, 2, 14, false)); len(rejections) > 0 {
		t.Fatalf("首屏折叠出现丢弃: %+v", rejections[0])
	}
	if transition := folder.AdvanceLoad(LoadOpenRequested); !transition.OK {
		t.Fatalf("打开失败: %+v", transition)
	}
	if transition := folder.AdvanceLoad(LoadFirstTurnArrived); !transition.OK {
		t.Fatalf("首轮到达失败: %+v", transition)
	}
	if !folder.HasOlder(mainBranch) {
		t.Fatalf("首屏之后应还有更早的轮次")
	}

	if transition := folder.AdvanceLoad(LoadConnectionLost); !transition.OK {
		t.Fatalf("断线迁移失败: %+v", transition)
	}
	window := folder.Window(mainBranch)
	if window.Anchor != 0 || window.HasOlder || window.Cursor != 0 {
		t.Fatalf("断线后应重设分页位置: %+v", window)
	}
	if _, ok := folder.OlderPageRequest(mainBranch, 50); ok {
		t.Fatalf("重设后不该直接给出翻页请求")
	}
}
