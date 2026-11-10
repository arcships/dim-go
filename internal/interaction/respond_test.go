package interaction

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestLongTermWritesRuleBeforeResponse 核对：长期允许先写入长期规则，成功后才发应答。
func TestLongTermWritesRuleBeforeResponse(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	fact := longTermApprovalFact("a1")
	arbiter.Apply(fact)

	outcome, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionAllowLongTerm})
	if err != nil {
		t.Fatalf("长期允许: %v", err)
	}
	wantKinds := []write.Kind{write.KindLongTermAllow, write.KindApprovalResponse}
	if got := writer.kinds(); len(got) != 2 || got[0] != wantKinds[0] || got[1] != wantKinds[1] {
		t.Fatalf("写入顺序不对: %v", got)
	}

	rule := writer.intentsOf(write.KindLongTermAllow)[0]
	if rule.SessionID != testSession || rule.BranchID != "main" || rule.InputID != "a1" {
		t.Fatalf("长期规则意图的目标不对: %+v", rule)
	}
	if !bytes.Equal(rule.Params, fact.LongTermParams) {
		t.Fatalf("长期规则参数没有原样透传: %s", rule.Params)
	}

	response := writer.intentsOf(write.KindApprovalResponse)[0]
	params := decodeParams(t, response.Params)
	if params["decision"] != "allow_always" {
		t.Fatalf("应答决定不对: %v", params)
	}
	if response.InputID != "a1" {
		t.Fatalf("应答应以审批请求标识作为核对标识: %+v", response)
	}

	if outcome.LongTerm == nil || outcome.LongTerm.Status != write.StatusSucceeded {
		t.Fatalf("长期规则回执不对: %+v", outcome.LongTerm)
	}
	if outcome.Response == nil || outcome.Response.Status != write.StatusSucceeded {
		t.Fatalf("应答应答回执不对: %+v", outcome.Response)
	}
	if outcome.Item.State != StateAwaitingResponse || outcome.Item.Decision != DecisionAllowLongTerm {
		t.Fatalf("应答后状态不对: %+v", outcome.Item)
	}
	if !outcome.Item.LongTermAvailable {
		t.Fatalf("视图应标明可长期允许: %+v", outcome.Item)
	}
	if current, ok := arbiter.Current(); !ok || current.RequestID != "a1" {
		t.Fatalf("结算前应停在队首: %+v ok=%v", current, ok)
	}
}

// TestLongTermRuleUnavailable 核对：没有可写规则参数时不允许长期允许，也不写任何意图。
func TestLongTermRuleUnavailable(t *testing.T) {
	cases := []struct {
		name string
		fact Fact
	}{
		{
			name: "服务端未声明可长期允许",
			fact: func() Fact {
				f := approvalFact("a1")
				f.LongTermParams = json.RawMessage(`{"tool":"Bash"}`)
				return f
			}(),
		},
		{
			name: "没有长期规则参数",
			fact: func() Fact {
				f := approvalFact("a1")
				f.AllowLongTerm = true
				return f
			}(),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			arbiter, writer := newTestArbiter(t, testSession)
			arbiter.Apply(tc.fact)
			_, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionAllowLongTerm})
			if !errors.Is(err, ErrLongTermUnavailable) {
				t.Fatalf("应拒绝长期允许: %v", err)
			}
			if writer.count() != 0 {
				t.Fatalf("拒绝时不应写任何意图: %v", writer.kinds())
			}
			item, _ := arbiter.Item("a1")
			if item.State != StatePresented {
				t.Fatalf("待决项应留在待呈现: %+v", item)
			}
		})
	}
}

// TestLongTermRuleFailureSendsNoResponse 核对：长期规则写入没有成功时保留待决、不发应答。
func TestLongTermRuleFailureSendsNoResponse(t *testing.T) {
	cases := []struct {
		name    string
		outcome func(write.Intent) fakeOutcome
	}{
		{
			name: "明确失败",
			outcome: func(write.Intent) fakeOutcome {
				return fakeOutcome{status: write.StatusFailed, reason: "服务端拒绝"}
			},
		},
		{
			name: "需人工确认",
			outcome: func(write.Intent) fakeOutcome {
				return fakeOutcome{status: write.StatusNeedsManualConfirmation}
			},
		},
		{
			name: "登记就失败",
			outcome: func(write.Intent) fakeOutcome {
				return fakeOutcome{registerErr: errors.New("写登记已关闭")}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			arbiter, writer := newTestArbiter(t, testSession)
			writer.outcome = tc.outcome
			arbiter.Apply(longTermApprovalFact("a1"))

			outcome, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionAllowLongTerm})
			if !errors.Is(err, ErrLongTermWriteFailed) {
				t.Fatalf("应报长期规则写入失败: %v", err)
			}
			if outcome.Response != nil {
				t.Fatalf("写入失败不应有应答回执: %+v", outcome.Response)
			}
			if got := writer.intentsOf(write.KindApprovalResponse); len(got) != 0 {
				t.Fatalf("写入失败不得发出应答: %+v", got)
			}
			item, ok := arbiter.Current()
			if !ok || item.RequestID != "a1" {
				t.Fatalf("审批应仍在待决集合: %+v ok=%v", item, ok)
			}
			if item.State != StatePresented {
				t.Fatalf("审批应回到待呈现: %+v", item)
			}
		})
	}
}

// TestResponseFailureAllowsRetry 核对：普通应答明确失败时留在待呈现，可换新幂等键重试。
func TestResponseFailureAllowsRetry(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	calls := 0
	writer.outcome = func(intent write.Intent) fakeOutcome {
		if intent.Kind == write.KindApprovalResponse {
			calls++
			if calls == 1 {
				return fakeOutcome{status: write.StatusFailed, reason: "服务端拒绝"}
			}
		}
		return fakeOutcome{status: write.StatusSucceeded}
	}
	arbiter.Apply(approvalFact("a1"))

	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionApprove}); !errors.Is(err, ErrResponseWriteFailed) {
		t.Fatalf("首次应答应明确失败: %v", err)
	}
	item, _ := arbiter.Current()
	if item.State != StatePresented {
		t.Fatalf("失败后应回到待呈现: %+v", item)
	}
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionApprove}); err != nil {
		t.Fatalf("重试应成功: %v", err)
	}
	responses := writer.intentsOf(write.KindApprovalResponse)
	if len(responses) != 2 {
		t.Fatalf("应有两次应答写入: %d", len(responses))
	}
	if responses[0].Key == responses[1].Key {
		t.Fatal("重试应使用新的幂等键")
	}
	item, _ = arbiter.Current()
	if item.State != StateAwaitingResponse {
		t.Fatalf("重试成功后应进入待应答: %+v", item)
	}
}

// TestLongTermResponseFailureRetriesOnlyResponse 核对：规则已写入后应答失败，只重试应答。
func TestLongTermResponseFailureRetriesOnlyResponse(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	calls := 0
	writer.outcome = func(intent write.Intent) fakeOutcome {
		if intent.Kind == write.KindApprovalResponse {
			calls++
			if calls == 1 {
				return fakeOutcome{status: write.StatusFailed, reason: "服务端拒绝"}
			}
		}
		return fakeOutcome{status: write.StatusSucceeded}
	}
	arbiter.Apply(longTermApprovalFact("a1"))

	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionAllowLongTerm}); !errors.Is(err, ErrResponseWriteFailed) {
		t.Fatalf("应答应明确失败: %v", err)
	}
	item, _ := arbiter.Current()
	if item.State != StateAwaitingResponse {
		t.Fatalf("规则已写入，应停在待应答: %+v", item)
	}
	// 规则已写入后不能改口。
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionReject}); !errors.Is(err, ErrUnsupportedDecision) {
		t.Fatalf("不应允许改口: %v", err)
	}
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionAllowLongTerm}); err != nil {
		t.Fatalf("重试应答应成功: %v", err)
	}
	if got := len(writer.intentsOf(write.KindLongTermAllow)); got != 1 {
		t.Fatalf("重试不应重写长期规则: %d", got)
	}
	responses := writer.intentsOf(write.KindApprovalResponse)
	if len(responses) != 2 || responses[0].Key == responses[1].Key {
		t.Fatalf("应答应重试一次并使用新键: %+v", responses)
	}
}

// TestRuleWrittenStateSurvivesReplay 核对：长期规则已写入后断线重放，只允许重试应答。
func TestRuleWrittenStateSurvivesReplay(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	calls := 0
	writer.outcome = func(intent write.Intent) fakeOutcome {
		if intent.Kind == write.KindApprovalResponse {
			calls++
			if calls == 1 {
				return fakeOutcome{status: write.StatusFailed, reason: "服务端拒绝"}
			}
		}
		return fakeOutcome{status: write.StatusSucceeded}
	}
	fact := longTermApprovalFact("a1")
	arbiter.Apply(fact)
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionAllowLongTerm}); !errors.Is(err, ErrResponseWriteFailed) {
		t.Fatalf("应答应明确失败: %v", err)
	}

	arbiter.Replay(Snapshot{Pending: []Fact{fact}})
	item, ok := arbiter.Current()
	if !ok || item.State != StateAwaitingResponse || item.Decision != DecisionAllowLongTerm {
		t.Fatalf("重放后应停在待应答: %+v ok=%v", item, ok)
	}
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionReject}); !errors.Is(err, ErrUnsupportedDecision) {
		t.Fatalf("规则已写入后不应改口: %v", err)
	}
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionAllowLongTerm}); err != nil {
		t.Fatalf("重试应答应成功: %v", err)
	}
	if got := len(writer.intentsOf(write.KindLongTermAllow)); got != 1 {
		t.Fatalf("重放后不应重写规则: %d", got)
	}
}

// TestQuestionAnswerValidation 核对：提问应答需要合法内容，决定与类型匹配。
func TestQuestionAnswerValidation(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	seedQuestionList(t, arbiter, 10, questionItem("q1", "c1", 1))

	if _, err := arbiter.Respond(context.Background(), "q1", Response{Decision: DecisionAnswer}); !errors.Is(err, ErrAnswerRequired) {
		t.Fatalf("缺应答内容应被拒: %v", err)
	}
	if _, err := arbiter.Respond(context.Background(), "q1", Response{Decision: DecisionAnswer, Answer: json.RawMessage(`{`)}); !errors.Is(err, ErrAnswerRequired) {
		t.Fatalf("坏应答内容应被拒: %v", err)
	}
	if _, err := arbiter.Respond(context.Background(), "q1", Response{Decision: DecisionApprove}); !errors.Is(err, ErrUnsupportedDecision) {
		t.Fatalf("提问不接受同意: %v", err)
	}
	if writer.count() != 0 {
		t.Fatalf("校验失败不应写任何意图: %v", writer.kinds())
	}
	if item, _ := arbiter.Current(); item.State != StatePresented {
		t.Fatalf("校验失败后应仍是待答: %+v", item)
	}
}

// TestQuestionAnswerAndCancelParams 核对：提问应答与取消的参数形状。
func TestQuestionAnswerAndCancelParams(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	seedQuestionList(t, arbiter, 10, questionItem("q1", "c1", 1))
	answer := json.RawMessage(`{"choice":["继续"]}`)
	outcome, err := arbiter.Respond(context.Background(), "q1", Response{Decision: DecisionAnswer, Answer: answer})
	if err != nil {
		t.Fatalf("应答提问: %v", err)
	}
	if outcome.Item.State != StateAnswered {
		t.Fatalf("提问应答后应进入已答: %+v", outcome.Item)
	}
	sent := writer.intentsOf(write.KindQuestionAnswer)[0]
	params := decodeParams(t, sent.Params)
	if params["decision"] != "answer" {
		t.Fatalf("决定不对: %v", params)
	}
	raw, ok := params["answer"].(map[string]any)
	if !ok || len(raw) != 1 {
		t.Fatalf("应答原文没有透传: %v", params["answer"])
	}

	seedQuestionList(t, arbiter, 10, questionItem("q2", "c2", 2))
	arbiter.SettleQuestion("q1", "c1", 20)
	if _, err := arbiter.Respond(context.Background(), "q2", Response{Decision: DecisionCancel}); err != nil {
		t.Fatalf("取消提问: %v", err)
	}
	cancels := writer.intentsOf(write.KindQuestionAnswer)
	if len(cancels) != 2 {
		t.Fatalf("应有两条提问写入: %d", len(cancels))
	}
	cancelParams := decodeParams(t, cancels[1].Params)
	if cancelParams["decision"] != "cancel" {
		t.Fatalf("取消决定不对: %v", cancelParams)
	}
	if _, exists := cancelParams["answer"]; exists {
		t.Fatalf("取消不应带应答内容: %v", cancelParams)
	}
}

// TestDecisionMatrix 核对：各类型只接受自己的决定。
func TestDecisionMatrix(t *testing.T) {
	cases := []struct {
		name      string
		kind      PendingKind
		requestID string
		decision  Decision
	}{
		{"审批取消", PendingApproval, "a1", DecisionCancel},
		{"审批当提问应答", PendingApproval, "a1", DecisionAnswer},
		{"提问当审批同意", PendingQuestion, "q1", DecisionApprove},
		{"提问长期允许", PendingQuestion, "q1", DecisionAllowLongTerm},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			arbiter, writer := newTestArbiter(t, testSession)
			if tc.kind == PendingQuestion {
				seedQuestionList(t, arbiter, 10, questionItem(tc.requestID, "c1", 1))
			} else {
				arbiter.Apply(approvalFact(tc.requestID))
			}
			response := Response{Decision: tc.decision, Answer: json.RawMessage(`"好"`)}
			if _, err := arbiter.Respond(context.Background(), tc.requestID, response); !errors.Is(err, ErrUnsupportedDecision) {
				t.Fatalf("应拒绝该决定: %v", err)
			}
			if writer.count() != 0 {
				t.Fatalf("拒绝时不应写意图: %v", writer.kinds())
			}
		})
	}
}

// TestManualConfirmationCountsAsAnswered 核对：需人工确认按已发出处理，不重发。
func TestManualConfirmationCountsAsAnswered(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	writer.outcome = func(write.Intent) fakeOutcome {
		return fakeOutcome{status: write.StatusNeedsManualConfirmation, reason: "核对无法确认"}
	}
	arbiter.Apply(approvalFact("a1"))
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionReject}); err != nil {
		t.Fatalf("需人工确认不应报错: %v", err)
	}
	item, _ := arbiter.Current()
	if item.State != StateAwaitingResponse {
		t.Fatalf("应视为已发出: %+v", item)
	}
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionReject}); !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("不应重复应答: %v", err)
	}
	if writer.count() != 1 {
		t.Fatalf("不应重发: %d", writer.count())
	}
}

// TestUnsettledResponseIsAnsweredNotResent 核对：等待结束但结果未收敛时按已答处理、不重发。
func TestUnsettledResponseIsAnsweredNotResent(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	writer.outcome = func(write.Intent) fakeOutcome {
		return fakeOutcome{status: write.StatusUnknown, waitErr: context.DeadlineExceeded}
	}
	arbiter.Apply(approvalFact("a1"))
	_, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionApprove})
	if !errors.Is(err, ErrResponseUnsettled) {
		t.Fatalf("应报结果未知: %v", err)
	}
	item, _ := arbiter.Current()
	if item.State != StateAwaitingResponse {
		t.Fatalf("结果未知应按已发出处理: %+v", item)
	}
	if _, err := arbiter.Respond(context.Background(), "a1", Response{Decision: DecisionApprove}); !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("不重发: %v", err)
	}
	if writer.count() != 1 {
		t.Fatalf("不应重发: %d", writer.count())
	}
}

func decodeParams(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	params := make(map[string]any)
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatalf("参数不是 JSON 对象: %s (%v)", raw, err)
	}
	return params
}

// TestRespondUnknownRequest 核对：未知请求与空会话的拒绝。
func TestRespondUnknownRequest(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	_, err := arbiter.Respond(context.Background(), "missing", Response{Decision: DecisionApprove})
	if !errors.Is(err, ErrUnknownPending) {
		t.Fatalf("未知请求应报未知: %v", err)
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("错误应带请求标识: %v", err)
	}
}
