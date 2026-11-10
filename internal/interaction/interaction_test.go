package interaction

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

// TestReplayReplacesApprovalSetKeepsQuestions 核对：审批重放整体替换审批集合；
// 提问集合由清单装载单独替换，重放不碰它。
func TestReplayReplacesApprovalSetKeepsQuestions(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	arbiter.Apply(approvalFact("a1"))
	seedQuestionList(t, arbiter, 10, questionItem("q1", "c1", 1))
	if got := arbiter.Counts(); got.Approvals != 1 || got.Questions != 1 {
		t.Fatalf("初始条数不对: %+v", got)
	}

	rejections := arbiter.Replay(Snapshot{Pending: []Fact{approvalFact("a2")}})
	if len(rejections) != 0 {
		t.Fatalf("重放不应产生丢弃: %v", rejections)
	}

	items := arbiter.Items()
	if len(items) != 2 || items[0].RequestID != "a2" || items[1].RequestID != "q1" {
		t.Fatalf("重放后审批应替换、提问应保留: %+v", items)
	}
	if got := arbiter.Counts(); got.Approvals != 1 || got.Questions != 1 {
		t.Fatalf("重放后条数不对: %+v", got)
	}
	if _, ok := arbiter.Item("a1"); ok {
		t.Fatal("旧审批没有被替换掉")
	}
	if current, ok := arbiter.Current(); !ok || current.RequestID != "a2" {
		t.Fatalf("队首应为重放的第一条审批: %+v ok=%v", current, ok)
	}

	// 提问事实不能从审批重放通道装回来。
	rejections = arbiter.Replay(Snapshot{Pending: []Fact{
		approvalFact("a2"),
		{Kind: PendingQuestion, RequestID: "q9", SessionID: testSession},
	}})
	if len(rejections) != 1 || rejections[0].Reason != RejectionQuestionSource {
		t.Fatalf("重放里的提问应被拒: %+v", rejections)
	}
	if questions := arbiter.Questions(); len(questions) != 1 || questions[0].RequestID != "q1" {
		t.Fatalf("提问集合不应被重放改动: %+v", questions)
	}
}

// TestQueueAdvancesOneAtATime 核对：一次只推一条，只有队首可应答，结算后才推进。
func TestQueueAdvancesOneAtATime(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	seedQuestionList(t, arbiter, 10,
		questionItem("q1", "c1", 1),
		questionItem("q2", "c2", 2),
	)
	ctx := context.Background()

	if _, err := arbiter.Respond(ctx, "q2", answerResponse("好")); !errors.Is(err, ErrNotCurrent) {
		t.Fatalf("第二条提问越前应答应被拒: %v", err)
	}
	outcome, err := arbiter.Respond(ctx, "q1", answerResponse("好"))
	if err != nil {
		t.Fatalf("应答队首提问: %v", err)
	}
	if outcome.Item.State != StateAnswered || outcome.Item.Decision != DecisionAnswer {
		t.Fatalf("提问应答后状态不对: %+v", outcome.Item)
	}
	// 应答写入成功后仍停在队首，等工具结果结算。
	if current, ok := arbiter.Current(); !ok || current.RequestID != "q1" {
		t.Fatalf("结算前不应推进: %+v ok=%v", current, ok)
	}
	if _, err := arbiter.Respond(ctx, "q1", answerResponse("再答")); !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("同一提问重复应答应被拒: %v", err)
	}

	if !arbiter.SettleQuestion("q1", "c1", 20) {
		t.Fatal("按工具结果结算队首应成功")
	}
	if current, ok := arbiter.Current(); !ok || current.RequestID != "q2" {
		t.Fatalf("结算后应推进到第二条: %+v ok=%v", current, ok)
	}
	if _, err := arbiter.Respond(ctx, "q2", answerResponse("好")); err != nil {
		t.Fatalf("应答第二条提问: %v", err)
	}
	if !arbiter.SettleQuestion("q2", "c2", 21) {
		t.Fatal("结算第二条应成功")
	}
	if _, ok := arbiter.Current(); ok {
		t.Fatal("全部结算后不应还有队首")
	}
	if _, err := arbiter.Respond(ctx, "q1", answerResponse("好")); !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("结算后的提问不应再被应答: %v", err)
	}
	if got := writer.count(); got != 2 {
		t.Fatalf("两次提问应答应只产生两条写意图，实际 %d", got)
	}
}

// TestSettlementBeforeAnswerRemovesItem 核对：服务端先结算的审批直接移除。
func TestSettlementBeforeAnswerRemovesItem(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	arbiter.Apply(approvalFact("a1"))
	if !arbiter.Settle("a1") {
		t.Fatal("结算应成功")
	}
	if _, ok := arbiter.Current(); ok {
		t.Fatal("结算后不应还有队首")
	}
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionApprove}); !errors.Is(err, ErrUnknownPending) {
		t.Fatalf("已结算且未答过的审批应报未知: %v", err)
	}
	if arbiter.Settle("a1") {
		t.Fatal("重复结算应按无操作处理")
	}
	if writer.count() != 0 {
		t.Fatal("已结算的待决项不应产生应答")
	}
}

// TestDuplicateResponseWhileWritingIsRejected 核对：同一时刻同一待决项只允许一次应答。
func TestDuplicateResponseWhileWritingIsRejected(t *testing.T) {
	writer := newFakeWriter()
	writer.entered = make(chan struct{})
	writer.release = make(chan struct{})
	arbiter, err := New(Config{SessionID: testSession, Writer: writer})
	if err != nil {
		t.Fatalf("创建交互裁决: %v", err)
	}
	arbiter.Apply(approvalFact("a1"))

	type result struct {
		outcome Outcome
		err     error
	}
	done := make(chan result, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		outcome, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionApprove})
		done <- result{outcome: outcome, err: err}
	}()

	<-writer.entered
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionApprove}); !errors.Is(err, ErrResponseInProgress) {
		t.Fatalf("并发应答应被拒: %v", err)
	}
	close(writer.release)
	first := <-done
	wg.Wait()
	if first.err != nil {
		t.Fatalf("第一次应答应成功: %v", first.err)
	}
	if writer.count() != 1 {
		t.Fatalf("并发应答只应发一次，实际 %d", writer.count())
	}
}

// TestDisconnectClearsUntilReload 核对：断线清空待决；重连后审批由重放、提问由
// 清单重建。
func TestDisconnectClearsUntilReload(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	arbiter.Apply(approvalFact("a1"))
	seedQuestionList(t, arbiter, 10, questionItem("q1", "c1", 1))

	arbiter.OnDisconnect()
	if items := arbiter.Items(); len(items) != 0 {
		t.Fatalf("断线后应清空待决集合: %+v", items)
	}
	if _, ok := arbiter.Current(); ok {
		t.Fatal("断线后不应还有队首")
	}
	if got := arbiter.Counts(); got.Approvals != 0 || got.Questions != 0 {
		t.Fatalf("断线条数应归零: %+v", got)
	}
	if _, ok := arbiter.QuestionSeedState(); ok {
		t.Fatal("断线后提问截止点应作废")
	}

	arbiter.Replay(Snapshot{Pending: []Fact{approvalFact("a1")}})
	seedQuestionList(t, arbiter, 20, questionItem("q2", "c2", 10))
	if !arbiter.Settle("a1") {
		t.Fatal("结算审批")
	}
	if current, ok := arbiter.Current(); !ok || current.RequestID != "q2" {
		t.Fatalf("重连后应以服务端事实为准: %+v ok=%v", current, ok)
	}
}

// TestApplyRejectsBadFacts 核对：坏审批事实、重复审批与提问事实被丢弃并带原因。
func TestApplyRejectsBadFacts(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)

	foreign := approvalFact("a-foreign")
	foreign.SessionID = "s2"
	missingID := approvalFact("")
	missingSession := approvalFact("a-no-session")
	missingSession.SessionID = ""
	unknownKind := approvalFact("a-unknown")
	unknownKind.Kind = "elicitation"
	badParams := longTermApprovalFact("a-bad-params")
	badParams.LongTermParams = json.RawMessage(`{`)

	cases := []struct {
		name   string
		fact   Fact
		reason string
	}{
		{"别的会话", foreign, RejectionSessionMismatch},
		{"缺请求标识", missingID, RejectionInvalidFact},
		{"缺会话标识", missingSession, RejectionInvalidFact},
		{"未知类型", unknownKind, RejectionInvalidFact},
		{"坏长期参数", badParams, RejectionInvalidFact},
		{"提问走错通道", Fact{Kind: PendingQuestion, RequestID: "q1", SessionID: testSession}, RejectionQuestionSource},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rejections := arbiter.Apply(tc.fact)
			if len(rejections) != 1 || rejections[0].Reason != tc.reason {
				t.Fatalf("丢弃原因不对: %+v", rejections)
			}
		})
	}

	if items := arbiter.Items(); len(items) != 0 {
		t.Fatalf("坏事实不应进入待决集合: %+v", items)
	}

	arbiter.Apply(approvalFact("a1"))
	rejections := arbiter.Apply(approvalFact("a1"))
	if len(rejections) != 1 || rejections[0].Reason != RejectionDuplicateRequest {
		t.Fatalf("重复事实应被丢弃: %+v", rejections)
	}
	if got := len(arbiter.Items()); got != 1 {
		t.Fatalf("重复事实不应累加: %d", got)
	}
}

// TestReplayRejectsBadAndDuplicateFacts 核对：快照里的坏审批与重复标识被丢弃，
// 提问事实被拒绝，其余按顺序重放。
func TestReplayRejectsBadAndDuplicateFacts(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	foreign := approvalFact("a-foreign")
	foreign.SessionID = "s2"
	seedQuestionList(t, arbiter, 10, questionItem("q1", "c1", 1))

	rejections := arbiter.Replay(Snapshot{Pending: []Fact{
		approvalFact("a1"),
		foreign,
		approvalFact("a1"),
		{Kind: PendingQuestion, RequestID: "q9", SessionID: testSession},
	}})
	if len(rejections) != 3 {
		t.Fatalf("应有三条丢弃: %+v", rejections)
	}
	if rejections[0].Reason != RejectionSessionMismatch ||
		rejections[1].Reason != RejectionDuplicateRequest ||
		rejections[2].Reason != RejectionQuestionSource {
		t.Fatalf("丢弃原因不对: %+v", rejections)
	}
	items := arbiter.Items()
	if len(items) != 2 || items[0].RequestID != "a1" || items[1].RequestID != "q1" {
		t.Fatalf("快照顺序或内容不对: %+v", items)
	}
}

// TestAnsweredItemStaysAnsweredAcrossSeed 核对：已答但未结算的提问重新装载后不再应答。
func TestAnsweredItemStaysAnsweredAcrossSeed(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	seedQuestionList(t, arbiter, 10, questionItem("q1", "c1", 1))
	if _, err := arbiter.Respond(context.Background(), "q1", answerResponse("好")); err != nil {
		t.Fatalf("应答: %v", err)
	}

	seedQuestionList(t, arbiter, 10,
		questionItem("q1", "c1", 1),
		questionItem("q2", "c2", 2),
	)
	item, ok := arbiter.Item("q1")
	if !ok || item.State != StateAnswered || item.Decision != DecisionAnswer {
		t.Fatalf("重装后已答项状态不对: %+v ok=%v", item, ok)
	}
	if _, err := arbiter.Respond(context.Background(), "q1", answerResponse("再答")); !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("重装后不应重复应答: %v", err)
	}
	if _, err := arbiter.Respond(context.Background(), "q2", answerResponse("好")); !errors.Is(err, ErrNotCurrent) {
		t.Fatalf("q1 未结算，q2 不应被应答: %v", err)
	}
	arbiter.SettleQuestion("q1", "c1", 20)
	if current, ok := arbiter.Current(); !ok || current.RequestID != "q2" {
		t.Fatalf("结算后应推进: %+v ok=%v", current, ok)
	}
	if writer.count() != 1 {
		t.Fatalf("重装不应重发已答项: %d", writer.count())
	}
}

// TestViewsAreCopies 核对：解析结果与视图都不会被外部改动影响。
func TestViewsAreCopies(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	item := questionItem("q1", "c1", 1)
	input := append(json.RawMessage(nil), item.Input...)
	seedQuestionList(t, arbiter, 10, item)
	item.Input[0] = 'X'

	current, ok := arbiter.Current()
	if !ok {
		t.Fatal("应有队首")
	}
	if current.Questions[0].Options[0] != "继续" {
		t.Fatalf("清单输入被外部改动影响: %+v", current.Questions[0])
	}
	if string(input) == string(item.Input) {
		t.Fatal("测试没有改动原输入")
	}
	current.Questions[0].Options[0] = "视图被改了"
	current.CallIDs[0] = "被改了"
	again, _ := arbiter.Current()
	if again.Questions[0].Options[0] != "继续" {
		t.Fatalf("视图不是拷贝: %+v", again.Questions[0])
	}
	if again.CallIDs[0] != "c1" {
		t.Fatalf("调用清单不是拷贝: %v", again.CallIDs)
	}
}

// TestNewRequiresConfig 核对：缺会话或写登记即启动失败。
func TestNewRequiresConfig(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("缺配置应失败")
	}
	if _, err := New(Config{SessionID: testSession}); err == nil {
		t.Fatal("缺写登记应失败")
	}
	arbiter, err := New(Config{SessionID: testSession, Writer: newFakeWriter()})
	if err != nil {
		t.Fatalf("合法配置应成功: %v", err)
	}
	if arbiter.SessionID() != testSession {
		t.Fatalf("会话标识不对: %s", arbiter.SessionID())
	}
}
