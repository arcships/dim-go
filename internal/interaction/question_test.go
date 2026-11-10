package interaction

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// TestQuestionSeedPresentsInListOrder 核对：清单注入后按清单顺序呈现，不分页也不重排。
func TestQuestionSeedPresentsInListOrder(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	seedQuestionList(t, arbiter, 10,
		questionItem("q2", "c2", 2),
		questionItem("q1", "c1", 1),
	)

	seed, ok := arbiter.QuestionSeedState()
	if !ok || seed.Through != 10 || seed.BranchID != "main" || seed.Source != QuestionSourceList {
		t.Fatalf("清单来源状态不对: %+v ok=%v", seed, ok)
	}
	items := arbiter.Items()
	if len(items) != 2 || items[0].RequestID != "q2" || items[1].RequestID != "q1" {
		t.Fatalf("清单顺序被改动: %+v", items)
	}
	current, ok := arbiter.Current()
	if !ok || current.RequestID != "q2" || current.Sequence != 2 {
		t.Fatalf("队首应是清单第一条: %+v ok=%v", current, ok)
	}
	if current.Source != QuestionSourceList || len(current.CallIDs) != 1 || current.CallIDs[0] != "c2" {
		t.Fatalf("队首来源或调用清单不对: %+v", current)
	}
	if got := arbiter.Counts(); got.Questions != 2 {
		t.Fatalf("提问条数不对: %+v", got)
	}
}

// TestQuestionSeedGroupsBatchCalls 核对：同一请求的多个 ask_user 调用合成一批，
// 逐个结算，全部结算后才移除。
func TestQuestionSeedGroupsBatchCalls(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	seedQuestionList(t, arbiter, 10,
		questionItem("q1", "c1", 1),
		questionItem("q1", "c2", 2),
	)
	items := arbiter.Items()
	if len(items) != 1 {
		t.Fatalf("同一请求应合成一批: %+v", items)
	}
	if len(items[0].CallIDs) != 2 || len(items[0].Questions) != 2 {
		t.Fatalf("批次应带两个调用与两个问题: %+v", items[0])
	}
	if !arbiter.SettleQuestion("q1", "c1", 20) {
		t.Fatal("结算第一个调用")
	}
	remaining, ok := arbiter.Current()
	if !ok || len(remaining.CallIDs) != 1 || remaining.CallIDs[0] != "c2" {
		t.Fatalf("批次不应在部分结算时移除: %+v ok=%v", remaining, ok)
	}
	if !arbiter.SettleQuestion("q1", "c2", 21) {
		t.Fatal("结算第二个调用")
	}
	if _, ok := arbiter.Current(); ok {
		t.Fatal("全部调用结算后应移除整批")
	}
}

// TestQuestionSettlementByToolResultAdvances 核对：工具结果结算后移除并推进。
func TestQuestionSettlementByToolResultAdvances(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	seedQuestionList(t, arbiter, 10,
		questionItem("q1", "c1", 1),
		questionItem("q2", "c2", 2),
	)
	if !arbiter.SettleQuestion("q1", "c1", 20) {
		t.Fatal("结算第一条失败")
	}
	if current, ok := arbiter.Current(); !ok || current.RequestID != "q2" {
		t.Fatalf("结算后应推进: %+v ok=%v", current, ok)
	}
	if arbiter.SettleQuestion("q1", "c1", 21) {
		t.Fatal("重复结算应按无操作处理")
	}
	if got := arbiter.Counts(); got.Questions != 1 {
		t.Fatalf("重复结算不应改变条数: %+v", got)
	}
}

// TestDuplicateSettlementDoesNotReanswer 核对：重复结算不会让条目重新出现或被再次应答。
func TestDuplicateSettlementDoesNotReanswer(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	seedQuestionList(t, arbiter, 10, questionItem("q1", "c1", 1))
	if _, err := arbiter.Respond(context.Background(), "q1", answerResponse("好")); err != nil {
		t.Fatalf("应答: %v", err)
	}
	if !arbiter.SettleQuestion("q1", "c1", 20) {
		t.Fatal("结算失败")
	}
	if arbiter.SettleQuestion("q1", "c1", 21) {
		t.Fatal("重复结算应按无操作处理")
	}
	if _, ok := arbiter.Current(); ok {
		t.Fatal("重复结算不应让条目重新出现")
	}
	if _, err := arbiter.Respond(context.Background(), "q1", answerResponse("再答")); !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("结算后不应重复应答: %v", err)
	}
	if writer.count() != 1 {
		t.Fatalf("不应重发: %d", writer.count())
	}
}

// TestQuestionListFailureFallsBackToFacts 核对：清单读取失败时，退回历史推导装载，
// 之后的工具事实仍按截止点生效。
func TestQuestionListFailureFallsBackToFacts(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)

	// 没有清单：先用时间线事实推导出第一条。
	if rejections := arbiter.ApplyAskUser(questionItem("q1", "c1", 5)); len(rejections) != 0 {
		t.Fatalf("推导事实被拒: %v", rejections)
	}
	current, ok := arbiter.Current()
	if !ok || current.RequestID != "q1" || current.Source != "" {
		t.Fatalf("无清单时应由事实推导: %+v ok=%v", current, ok)
	}

	// 清单读取失败：集成层注入历史推导结果，标记来源为 derived。
	seedDerivedQuestions(t, arbiter, 6,
		questionItem("q1", "c1", 5),
		questionItem("q2", "c2", 6),
	)
	seed, ok := arbiter.QuestionSeedState()
	if !ok || seed.Source != QuestionSourceDerived || seed.Through != 6 {
		t.Fatalf("推导来源状态不对: %+v ok=%v", seed, ok)
	}
	if items := arbiter.Items(); len(items) != 2 || items[0].Source != QuestionSourceDerived {
		t.Fatalf("推导装载结果不对: %+v", items)
	}

	// 截止点之后的工具事实继续生效。
	if rejections := arbiter.ApplyAskUser(questionItem("q3", "c3", 7)); len(rejections) != 0 {
		t.Fatalf("截止点后的调用被拒: %v", rejections)
	}
	if !arbiter.SettleQuestion("q1", "c1", 8) {
		t.Fatal("结算第一条失败")
	}
	if current, ok := arbiter.Current(); !ok || current.RequestID != "q2" {
		t.Fatalf("结算后应推进: %+v ok=%v", current, ok)
	}

	// 清单恢复后整体替换推导结果，来源回到 list。
	seedQuestionList(t, arbiter, 9, questionItem("q4", "c4", 8))
	if seed, _ := arbiter.QuestionSeedState(); seed.Source != QuestionSourceList {
		t.Fatalf("清单恢复后来源应为 list: %+v", seed)
	}
	if items := arbiter.Items(); len(items) != 1 || items[0].RequestID != "q4" {
		t.Fatalf("清单应整体替换推导结果: %+v", items)
	}
}

// TestQuestionBelowCutoffFactsAreIgnored 核对：截止点及以下的调用与结算不改动清单。
func TestQuestionBelowCutoffFactsAreIgnored(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	seedQuestionList(t, arbiter, 10, questionItem("q1", "c1", 5))

	rejections := arbiter.ApplyAskUser(questionItem("q2", "c2", 10))
	if len(rejections) != 1 || rejections[0].Reason != RejectionBelowCutoff {
		t.Fatalf("截止点上的调用应被丢弃: %+v", rejections)
	}
	if arbiter.SettleQuestion("q1", "c1", 5) {
		t.Fatal("截止点上的结算不该改动清单")
	}
	if got := arbiter.Counts(); got.Questions != 1 {
		t.Fatalf("清单不应被截止点以下事实改动: %+v", got)
	}
	if !arbiter.SettleQuestion("q1", "c1", 11) {
		t.Fatal("截止点后的结算应生效")
	}
}

// TestSettledCallIsNotReadded 核对：已结算的调用不会被旧事实装回来。
func TestSettledCallIsNotReadded(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	if rejections := arbiter.ApplyAskUser(questionItem("q1", "c1", 5)); len(rejections) != 0 {
		t.Fatalf("调用被拒: %v", rejections)
	}
	if !arbiter.SettleQuestion("q1", "c1", 10) {
		t.Fatal("结算失败")
	}
	rejections := arbiter.ApplyAskUser(questionItem("q1", "c1", 5))
	if len(rejections) != 1 || rejections[0].Reason != RejectionAlreadySettled {
		t.Fatalf("旧调用应被拒: %+v", rejections)
	}
	if _, ok := arbiter.Current(); ok {
		t.Fatal("旧调用不应重新出现")
	}
}

// TestQuestionSeedRejectsInvalidItems 核对：清单坏项被丢弃，合法项照常装载。
func TestQuestionSeedRejectsInvalidItems(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)

	missingCall := questionItem("q1", "", 1)
	wrongTool := questionItem("q1", "c1", 1)
	wrongTool.ToolName = "read_file"
	emptyInput := questionItem("q1", "c1", 1)
	emptyInput.Input = json.RawMessage(`{"questions":[]}`)
	valid := questionItem("q2", "c2", 2)

	rejections := arbiter.SeedQuestions(QuestionSeed{
		BranchID: "main",
		Through:  10,
		Source:   QuestionSourceList,
		Items:    []QuestionItem{missingCall, wrongTool, emptyInput, valid, valid},
	})
	if len(rejections) != 4 {
		t.Fatalf("坏项与重复项应被丢弃: %+v", rejections)
	}
	for _, rejection := range rejections {
		if rejection.Kind != PendingQuestion || rejection.Reason != RejectionInvalidFact && rejection.Reason != RejectionDuplicateRequest {
			t.Fatalf("丢弃原因不对: %+v", rejection)
		}
	}
	if items := arbiter.Items(); len(items) != 1 || items[0].RequestID != "q2" {
		t.Fatalf("合法项应照常装载: %+v", items)
	}
}

// TestSeedQuestionsRequiresSourceAndBranch 核对：装载缺来源或分支直接拒绝。
func TestSeedQuestionsRequiresSourceAndBranch(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	cases := []QuestionSeed{
		{Through: 1, Source: QuestionSourceList, Items: []QuestionItem{questionItem("q1", "c1", 1)}},
		{BranchID: "main", Through: 1, Items: []QuestionItem{questionItem("q1", "c1", 1)}},
		{BranchID: "main", Through: -1, Source: QuestionSourceList, Items: []QuestionItem{questionItem("q1", "c1", 1)}},
	}
	for _, seed := range cases {
		rejections := arbiter.SeedQuestions(seed)
		if len(rejections) != 1 || rejections[0].Reason != RejectionInvalidFact {
			t.Fatalf("坏装载应被拒: %+v", rejections)
		}
	}
	if _, ok := arbiter.QuestionSeedState(); ok {
		t.Fatal("坏装载不应记录来源")
	}
}

// TestApplyAskUserRejectsDuplicateCall 核对：同一次调用重复到达按重复丢弃。
func TestApplyAskUserRejectsDuplicateCall(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	if rejections := arbiter.ApplyAskUser(questionItem("q1", "c1", 5)); len(rejections) != 0 {
		t.Fatalf("调用被拒: %v", rejections)
	}
	rejections := arbiter.ApplyAskUser(questionItem("q1", "c1", 5))
	if len(rejections) != 1 || rejections[0].Reason != RejectionDuplicateRequest {
		t.Fatalf("重复调用应被拒: %+v", rejections)
	}
	if got := arbiter.Counts(); got.Questions != 1 {
		t.Fatalf("重复调用不应累加: %+v", got)
	}
}
