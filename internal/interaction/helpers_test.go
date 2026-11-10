package interaction

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// testSession 是测试里的会话标识。
const testSession = "s1"

// fakeOutcome 是一条写意图在假写登记里的结局。
type fakeOutcome struct {
	status      write.Status
	reason      string
	registerErr error
	waitErr     error
}

// fakeWriter 是写登记的测试替身：记录每次登记，并按脚本给出终态。
type fakeWriter struct {
	mu      sync.Mutex
	intents []write.Intent
	views   map[string]write.IntentView
	results map[string]fakeOutcome
	next    int
	// outcome 按意图决定结局；nil 时一律成功。
	outcome func(write.Intent) fakeOutcome
	// entered 与 release 非 nil 时，Register 在登记前通知并等待，用来制造并发应答。
	entered chan struct{}
	release chan struct{}
}

func newFakeWriter() *fakeWriter {
	return &fakeWriter{
		views:   make(map[string]write.IntentView),
		results: make(map[string]fakeOutcome),
	}
}

func (f *fakeWriter) Register(intent write.Intent) (write.IntentView, error) {
	if f.entered != nil {
		f.entered <- struct{}{}
		<-f.release
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.next++
	id := fmt.Sprintf("intent-%d", f.next)
	f.intents = append(f.intents, intent)
	out := fakeOutcome{status: write.StatusSucceeded}
	if f.outcome != nil {
		out = f.outcome(intent)
	}
	if out.registerErr != nil {
		return write.IntentView{}, out.registerErr
	}
	view := write.IntentView{
		ID:        id,
		Key:       intent.Key,
		Kind:      intent.Kind,
		SessionID: intent.SessionID,
		BranchID:  intent.BranchID,
		InputID:   intent.InputID,
		Status:    write.StatusRegistered,
	}
	f.views[id] = view
	f.results[id] = out
	return view, nil
}

func (f *fakeWriter) Wait(ctx context.Context, intentID string) (write.IntentView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out, ok := f.results[intentID]
	if !ok {
		return write.IntentView{}, fmt.Errorf("假写登记没有意图 %s", intentID)
	}
	view := f.views[intentID]
	view.Status = out.status
	view.Reason = out.reason
	f.views[intentID] = view
	return view, out.waitErr
}

func (f *fakeWriter) kinds() []write.Kind {
	f.mu.Lock()
	defer f.mu.Unlock()
	kinds := make([]write.Kind, 0, len(f.intents))
	for _, intent := range f.intents {
		kinds = append(kinds, intent.Kind)
	}
	return kinds
}

func (f *fakeWriter) intentsOf(kind write.Kind) []write.Intent {
	f.mu.Lock()
	defer f.mu.Unlock()
	intents := make([]write.Intent, 0, len(f.intents))
	for _, intent := range f.intents {
		if intent.Kind == kind {
			intents = append(intents, intent)
		}
	}
	return intents
}

func (f *fakeWriter) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.intents)
}

func newTestArbiter(t *testing.T, sessionID string) (*Arbiter, *fakeWriter) {
	t.Helper()
	writer := newFakeWriter()
	arbiter, err := New(Config{SessionID: sessionID, Writer: writer})
	if err != nil {
		t.Fatalf("创建交互裁决: %v", err)
	}
	return arbiter, writer
}

func approvalFact(id string) Fact {
	return Fact{
		Kind:      PendingApproval,
		RequestID: id,
		SessionID: testSession,
		BranchID:  "main",
		Action:    "运行命令",
	}
}

func longTermApprovalFact(id string) Fact {
	fact := approvalFact(id)
	fact.AllowLongTerm = true
	fact.LongTermParams = json.RawMessage(`{"tool":"Bash","scope":"workspace"}`)
	return fact
}

// questionItem 造一条 ask_user 调用；问题内容从 Input 解析，调用方不再直接给 Question。
func questionItem(requestID, callID string, sequence int64) QuestionItem {
	input := fmt.Sprintf(
		`{"questions":[{"id":"%s-1","question":"继续吗","options":["继续","停止"]}]}`,
		requestID,
	)
	return QuestionItem{
		RequestID: requestID,
		CallID:    callID,
		EventID:   "e-" + callID,
		Sequence:  sequence,
		ToolName:  "ask_user",
		Input:     json.RawMessage(input),
	}
}

// seedQuestionList 按清单装载提问，断言没有丢弃。
func seedQuestionList(t *testing.T, arbiter *Arbiter, through int64, items ...QuestionItem) {
	t.Helper()
	rejections := arbiter.SeedQuestions(QuestionSeed{
		BranchID: "main",
		Through:  through,
		Source:   QuestionSourceList,
		Items:    items,
	})
	if len(rejections) != 0 {
		t.Fatalf("装载提问清单有丢弃: %v", rejections)
	}
}

// seedDerivedQuestions 按历史推导装载提问，断言没有丢弃。
func seedDerivedQuestions(t *testing.T, arbiter *Arbiter, through int64, items ...QuestionItem) {
	t.Helper()
	rejections := arbiter.SeedQuestions(QuestionSeed{
		BranchID: "main",
		Through:  through,
		Source:   QuestionSourceDerived,
		Items:    items,
	})
	if len(rejections) != 0 {
		t.Fatalf("装载推导提问有丢弃: %v", rejections)
	}
}

func answerResponse(text string) Response {
	raw, _ := json.Marshal(text)
	return Response{Decision: DecisionAnswer, Answer: raw}
}
