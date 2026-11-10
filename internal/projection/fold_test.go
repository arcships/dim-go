package projection

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// TestHistoryAndLiveConverge 验证同一 fixture 走历史与逐条实时得到同一份内容，
// 并且实时那侧在被历史覆盖后身份一致、没有第二条。
func TestHistoryAndLiveConverge(t *testing.T) {
	fixture := conversationFixture()

	historyFolder := newFolder(t)
	mustHistory(t, historyFolder, mainBranch, page(18, fixture...))
	historyView := historyFolder.View()

	liveFolder := newFolder(t)
	for _, event := range fixture {
		mustNotice(t, liveFolder, mainBranch, event)
	}
	liveView := liveFolder.View()

	if !reflect.DeepEqual(fingerprint(historyView), fingerprint(liveView)) {
		t.Fatalf("历史与实时的行内容不一致\n历史: %+v\n实时: %+v", fingerprint(historyView), fingerprint(liveView))
	}
	wantKinds := []RowKind{
		RowUserMessage, RowThinking, RowAssistantText, RowTool,
		RowAssistantText, RowGoalChange, RowTool, RowSystem, RowUserMessage,
	}
	if len(historyView.Rows) != len(wantKinds) {
		t.Fatalf("行数应为 %d，实际 %d: %+v", len(wantKinds), len(historyView.Rows), fingerprint(historyView))
	}
	for index, kind := range wantKinds {
		if historyView.Rows[index].Kind != kind {
			t.Fatalf("第 %d 行类型应为 %s，实际 %s", index, kind, historyView.Rows[index].Kind)
		}
	}
	if historyView.Rows[0].Text != "帮我看下构建" || historyView.Rows[0].TurnIndex != 1 {
		t.Fatalf("用户行不对: %+v", historyView.Rows[0])
	}
	if historyView.Rows[8].TurnIndex != 0 {
		t.Fatalf("轮次结束后到达的排队输入不该归属上一轮: %+v", historyView.Rows[8])
	}
	if historyView.Pending.Total() != 4 {
		t.Fatalf("四类未完成事项应各一条: %+v", historyView.Pending)
	}
	if historyView.Goal == nil || historyView.Goal.Objective != "修好构建" || historyView.Goal.Status != GoalActive {
		t.Fatalf("目标折叠不对: %+v", historyView.Goal)
	}

	if !reflect.DeepEqual(historyView.Turns, liveView.Turns) {
		t.Fatalf("轮索引不一致\n历史: %+v\n实时: %+v", historyView.Turns, liveView.Turns)
	}
	if !reflect.DeepEqual(historyView.Goal, liveView.Goal) {
		t.Fatalf("目标不一致\n历史: %+v\n实时: %+v", historyView.Goal, liveView.Goal)
	}
	if !reflect.DeepEqual(historyView.Pending, liveView.Pending) {
		t.Fatalf("未完成事项不一致\n历史: %+v\n实时: %+v", historyView.Pending, liveView.Pending)
	}

	// 历史那侧全部落库；实时那侧还没有被历史覆盖，因此都是尾部行。
	for _, row := range historyView.Rows {
		if row.Pending || row.ID == "" {
			t.Fatalf("历史折叠的行没有落库: %+v", row)
		}
	}
	if liveView.Unlanded != len(liveView.Rows) {
		t.Fatalf("实时折叠应全部算作尾部行: 尾部 %d 行 %d", liveView.Unlanded, len(liveView.Rows))
	}

	// attach 重叠：实时之后再来一页同日历史，身份补上且不重复。
	mustHistory(t, liveFolder, mainBranch, page(18, fixture...))
	if !reflect.DeepEqual(liveFolder.View(), historyView) {
		t.Fatalf("历史覆盖后与纯历史结果不一致\n覆盖后: %+v\n历史: %+v", liveFolder.View(), historyView)
	}
}

// TestDuplicateAndOutOfOrderNotifications 验证重复与乱序都不产生重复行。
func TestDuplicateAndOutOfOrderNotifications(t *testing.T) {
	fixture := conversationFixture()

	reference := newFolder(t)
	mustHistory(t, reference, mainBranch, page(18, fixture...))
	referenceView := reference.View()

	// 倒序逐条到达，再原样重复一遍：每一条都被重复投递两次。
	shuffled := newFolder(t)
	for index := len(fixture) - 1; index >= 0; index-- {
		mustNotice(t, shuffled, mainBranch, fixture[index])
		mustNotice(t, shuffled, mainBranch, fixture[index])
	}
	if !reflect.DeepEqual(fingerprint(shuffled.View()), fingerprint(referenceView)) {
		t.Fatalf("乱序重复到达产生了不同的行\n乱序: %+v\n期望: %+v", fingerprint(shuffled.View()), fingerprint(referenceView))
	}
	if len(shuffled.View().Rows) != len(referenceView.Rows) {
		t.Fatalf("行数不一致: %d vs %d", len(shuffled.View().Rows), len(referenceView.Rows))
	}

	// 乱序那侧被历史覆盖后，身份也与纯历史一致。
	mustHistory(t, shuffled, mainBranch, page(18, fixture...))
	if !reflect.DeepEqual(shuffled.View(), referenceView) {
		t.Fatalf("乱序折叠落库后与历史不一致")
	}
}

// TestToolResultBeforeCallKeepsOneRow 验证结果先于调用到达时也只留一行工具行。
func TestToolResultBeforeCallKeepsOneRow(t *testing.T) {
	folder := newFolder(t)
	result := ev("r-1", 3, "step", "tool-completed", `{"turn_id":"t1","request_id":"r1","call_id":"c1","result":{"content":[{"type":"text","text":"done"}]}}`)
	call := ev("u-1", 4, "step", "tool-use", `{"turn_id":"t1","request_id":"r1","call":{"id":"c1","name":"run","input":{"cmd":"ls"}}}`)

	for _, event := range []protocol.RawEvent{result, call} {
		if rejections := folder.ApplyNotice("session.event", noticeJSON(t, mainBranch, event)); len(rejections) > 0 {
			t.Fatalf("折叠出现丢弃: %+v", rejections[0])
		}
	}
	view := folder.View()
	var tools []Row
	for _, row := range view.Rows {
		if row.Kind == RowTool {
			tools = append(tools, row)
		}
	}
	if len(tools) != 1 {
		t.Fatalf("工具行应有 1 条，实际 %d: %+v", len(tools), tools)
	}
	if tools[0].Tool.Result != "done" || tools[0].Status != StatusComplete {
		t.Fatalf("工具结果没有合并: %+v", tools[0].Tool)
	}
}

// TestRowIdentityStableAcrossReads 验证两次读取的行标识序列一致。
func TestRowIdentityStableAcrossReads(t *testing.T) {
	fixture := conversationFixture()

	first := newFolder(t)
	mustHistory(t, first, mainBranch, page(18, fixture...))
	firstView := first.View()
	if !reflect.DeepEqual(first.View(), firstView) {
		t.Fatalf("同一折叠器两次读取不一致")
	}

	second := newFolder(t)
	mustHistory(t, second, mainBranch, page(18, fixture...))
	secondView := second.View()
	if !reflect.DeepEqual(firstView, secondView) {
		t.Fatalf("同一 fixture 两次读取不一致")
	}
	ids := rowIDs(firstView)
	if len(ids) != len(secondView.Rows) {
		t.Fatalf("行数不一致")
	}
	for index, id := range ids {
		if id == "" {
			t.Fatalf("第 %d 行没有正式标识", index)
		}
		if id != secondView.Rows[index].ID {
			t.Fatalf("第 %d 行标识不一致: %q vs %q", index, id, secondView.Rows[index].ID)
		}
	}
}

// TestUnlandedIncrementLandsWithoutDuplication 验证未落库增量叠加后落库不重复。
func TestUnlandedIncrementLandsWithoutDuplication(t *testing.T) {
	fixture := conversationFixture()
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(12, fixture[:12]...))

	live := ev("e19", 19, "step", "text", `{"turn_id":"t2","request_id":"r2","delta":"新一段"}`)
	mustNotice(t, folder, mainBranch, live)

	view := folder.View()
	last := view.Rows[len(view.Rows)-1]
	if !last.Pending || last.ID != "" {
		t.Fatalf("未落库的行不应带正式标识: %+v", last)
	}
	if last.Text != "新一段" {
		t.Fatalf("未落库增量没有并入: %+v", last)
	}
	if view.Unlanded != 1 {
		t.Fatalf("尾部行数应为 1，实际 %d", view.Unlanded)
	}
	before := len(view.Rows)

	// 同一事实落库：原地换成正式标识，行数不变。
	mustHistory(t, folder, mainBranch, page(19, live))
	view = folder.View()
	if len(view.Rows) != before {
		t.Fatalf("落库后行数变了: %d -> %d", before, len(view.Rows))
	}
	last = view.Rows[len(view.Rows)-1]
	if last.Pending || last.ID != "e19" {
		t.Fatalf("落库后应换成服务端标识: %+v", last)
	}
	if last.Text != "新一段" {
		t.Fatalf("落库后文本被重复叠加: %q", last.Text)
	}
	if view.Unlanded != 0 {
		t.Fatalf("落库后不应再有尾部行: %d", view.Unlanded)
	}
}

// TestDisconnectClearsTail 验证断线即清空未落库尾部，已落库内容不受影响。
func TestDisconnectClearsTail(t *testing.T) {
	fixture := conversationFixture()
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(12, fixture[:12]...))
	if transition := folder.AdvanceLoad(LoadOpenRequested); !transition.OK {
		t.Fatalf("打开失败: %+v", transition)
	}
	if transition := folder.AdvanceLoad(LoadFirstTurnArrived); !transition.OK {
		t.Fatalf("首轮到达失败: %+v", transition)
	}

	mustNotice(t, folder, mainBranch, ev("e19", 19, "step", "text", `{"turn_id":"t2","request_id":"r2","delta":"新一段"}`))
	folder.ApplyNotice("session.model.progress", progressJSON(t, mainBranch, "t2", "r2", "预览", 0, "", 0))

	before := folder.View()
	if before.Unlanded == 0 {
		t.Fatalf("断线前应有未落库尾部")
	}

	transition := folder.AdvanceLoad(LoadConnectionLost)
	if !transition.OK || transition.To != string(PhaseLoadingFirst) {
		t.Fatalf("断线迁移不符合迁移表: %+v", transition)
	}
	after := folder.View()
	if after.Unlanded != 0 {
		t.Fatalf("断线后尾部应清空，实际 %d", after.Unlanded)
	}
	for _, row := range after.Rows {
		if row.Pending {
			t.Fatalf("断线后仍有尾部行: %+v", row)
		}
	}
	if len(after.Rows) >= len(before.Rows) {
		t.Fatalf("断线后行数没有减少: %d -> %d", len(before.Rows), len(after.Rows))
	}
	if !reflect.DeepEqual(fingerprint(after)[:len(after.Rows)], fingerprint(before)[:len(after.Rows)]) {
		t.Fatalf("断线不该改动已落库的行")
	}
}

// TestProgressPreviewIsReplacedByLandedRows 验证末快照按整行替换：预览只在落库前出现，
// 落库后整行消失，也不与落库文本重复。
func TestProgressPreviewIsReplacedByLandedRows(t *testing.T) {
	fixture := conversationFixture()
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(12, fixture[:12]...))

	folder.ApplyNotice("session.model.progress", progressJSON(t, mainBranch, "t2", "r2", "正在写", 0, "思考中", 0))
	view := folder.View()
	previewRows := view.Rows[len(view.Rows)-2:]
	if len(previewRows) != 2 || previewRows[0].Kind != RowAssistantText || previewRows[1].Kind != RowThinking {
		t.Fatalf("末快照没有摊成尾部行: %+v", previewRows)
	}
	for _, row := range previewRows {
		if !row.Pending || row.ID != "" {
			t.Fatalf("预览行不该带正式标识: %+v", row)
		}
	}
	if previewRows[0].Text != "正在写" || previewRows[1].Text != "思考中" {
		t.Fatalf("预览内容不对: %+v", previewRows)
	}

	// 同一条预览再来一次：按偏移只留尾部，整行覆盖而不是叠加。
	folder.ApplyNotice("session.model.progress", progressJSON(t, mainBranch, "t2", "r2", "正在写下去", 3, "", 0))
	view = folder.View()
	last := view.Rows[len(view.Rows)-1]
	if last.Text != "下去" {
		t.Fatalf("末快照没有按偏移裁剪: %q", last.Text)
	}

	// 事实到达后预览整行消失，只留事实那一行，不拼接也不重复。
	durable := ev("e20", 20, "step", "text", `{"turn_id":"t2","request_id":"r2","delta":"正在写下去"}`)
	mustNotice(t, folder, mainBranch, durable)
	folder.ApplyNotice("session.model.progress", progressJSON(t, mainBranch, "t2", "r2", "正在写下去", 5, "", 0))
	view = folder.View()
	last = view.Rows[len(view.Rows)-1]
	if last.Text != "正在写下去" || last.Sequence != 20 {
		t.Fatalf("预览与落库内容没有按整行替换: %+v", last)
	}
	if view.Unlanded != 1 {
		t.Fatalf("预览应已整行消失，只留未落库的那一行: %d", view.Unlanded)
	}

	// 历史覆盖后这一行落库并换成服务端标识。
	mustHistory(t, folder, mainBranch, page(20, durable))
	view = folder.View()
	last = view.Rows[len(view.Rows)-1]
	if last.Pending || last.ID != "e20" || last.Text != "正在写下去" {
		t.Fatalf("落库行不对: %+v", last)
	}
	if view.Unlanded != 0 {
		t.Fatalf("落库后仍有尾部行: %d", view.Unlanded)
	}
}

// TestIllegalFactsAreRejectedWithReason 验证非法事实被丢弃并带原因。
func TestIllegalFactsAreRejectedWithReason(t *testing.T) {
	folder := newFolder(t)

	missingID := protocol.RawEvent{Sequence: 1, CreateAt: fixtureBaseAt, Type: "input", Name: "message"}
	rejections := folder.ApplyNotice("session.event", noticeJSON(t, mainBranch, missingID))
	if len(rejections) != 1 || rejections[0].Reason != ReasonInvalidEvent {
		t.Fatalf("缺标识的事实应被拒绝: %+v", rejections)
	}

	otherSession, err := json.Marshal(protocol.SessionEventParams{
		SessionID: "s-2",
		BranchID:  mainBranch,
		Event:     ev("x1", 1, "input", "message", `{"user":{"content":[]}}`),
	})
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}
	rejections = folder.ApplyNotice("session.event", otherSession)
	if len(rejections) != 1 || rejections[0].Reason != ReasonSessionMismatch {
		t.Fatalf("别的会话的事实应被拒绝: %+v", rejections)
	}

	rejections = folder.ApplyNotice("session.event", json.RawMessage(`{`))
	if len(rejections) != 1 || rejections[0].Reason != ReasonMalformedNotice {
		t.Fatalf("坏通知应被拒绝: %+v", rejections)
	}
	if len(folder.View().Rows) != 0 {
		t.Fatalf("非法事实不该产生行")
	}
}
