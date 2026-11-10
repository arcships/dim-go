package write

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
)

var errLookup = errors.New("核对读取失败")

// fakeSender 记录每次发送，并按 handle 回执。
type fakeSender struct {
	mu     sync.Mutex
	calls  []Request
	handle func(context.Context, Request) (Reply, error)
}

func (s *fakeSender) Send(ctx context.Context, req Request) (Reply, error) {
	s.mu.Lock()
	s.calls = append(s.calls, cloneRequest(req))
	handle := s.handle
	s.mu.Unlock()
	if handle == nil {
		return Reply{}, nil
	}
	return handle(ctx, req)
}

func (s *fakeSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func (s *fakeSender) requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Request, 0, len(s.calls))
	for _, call := range s.calls {
		out = append(out, cloneRequest(call))
	}
	return out
}

func cloneRequest(req Request) Request {
	req.Params = append(json.RawMessage(nil), req.Params...)
	return req
}

// fakeInspector 记录每次核对，并按 handle 返回。
type fakeInspector struct {
	mu     sync.Mutex
	calls  []InputQuery
	handle func(InputQuery, int) (InputState, error)
}

func (s *fakeInspector) LookupInput(ctx context.Context, query InputQuery) (InputState, error) {
	s.mu.Lock()
	index := len(s.calls)
	s.calls = append(s.calls, query)
	handle := s.handle
	s.mu.Unlock()
	if handle == nil {
		return InputState{}, nil
	}
	return handle(query, index)
}

func (s *fakeInspector) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func (s *fakeInspector) queries() []InputQuery {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]InputQuery(nil), s.calls...)
}

func newTestManager(t *testing.T, sender Sender, inspector Inspector) *Manager {
	t.Helper()
	manager, err := New(Config{
		Sender:      sender,
		Inspector:   inspector,
		SendTimeout: 200 * time.Millisecond,
		Verify: VerifyPolicy{
			MaxAttempts: 3,
			Interval:    time.Millisecond,
			Timeout:     time.Second,
		},
	})
	if err != nil {
		t.Fatalf("创建写登记: %v", err)
	}
	t.Cleanup(manager.Close)
	return manager
}

func waitStatus(t *testing.T, manager *Manager, intentID string, want Status) IntentView {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	view, err := manager.Wait(ctx, intentID)
	if err != nil {
		t.Fatalf("等待意图 %s: %v", intentID, err)
	}
	if view.Status != want {
		t.Fatalf("状态=%s 期望 %s（原因 %q）", view.Status, want, view.Reason)
	}
	return view
}

func mustRegister(t *testing.T, manager *Manager, intent Intent) IntentView {
	t.Helper()
	view, err := manager.Register(intent)
	if err != nil {
		t.Fatalf("登记意图: %v", err)
	}
	return view
}

func submitIntent(key string) Intent {
	return Intent{
		Key:       key,
		Kind:      KindSubmitInput,
		SessionID: "s1",
		BranchID:  "main",
		Params:    json.RawMessage(`{"text":"你好"}`),
	}
}

func TestRegisterSendsOnceAndDuplicateReturnsSameIntent(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{InputID: "server-input-1", Result: json.RawMessage(`{"ok":true}`)}, nil
	}}
	inspector := &fakeInspector{}
	manager := newTestManager(t, sender, inspector)

	first := mustRegister(t, manager, submitIntent("key-1"))
	if first.Status != StatusRegistered {
		t.Fatalf("登记后状态=%s 期望 registered", first.Status)
	}
	if first.InputID == "" {
		t.Fatal("提交类意图应生成输入标识")
	}
	view := waitStatus(t, manager, first.ID, StatusSucceeded)
	if view.InputID != "server-input-1" {
		t.Fatalf("成功后输入标识=%q 期望服务端回执值", view.InputID)
	}
	if len(view.Content) != 0 {
		t.Fatalf("成功后仍保留待发内容: %s", view.Content)
	}
	if string(view.Result) != `{"ok":true}` {
		t.Fatalf("成功回执=%s 期望原样保留", view.Result)
	}

	second, err := manager.Register(submitIntent("key-1"))
	if err != nil {
		t.Fatalf("重复登记: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("重复登记产生新意图 %s，期望 %s", second.ID, first.ID)
	}
	if second.Status != StatusSucceeded {
		t.Fatalf("重复登记返回状态=%s，期望 succeeded", second.Status)
	}
	if sender.count() != 1 {
		t.Fatalf("发送次数=%d 期望 1", sender.count())
	}
	if inspector.count() != 0 {
		t.Fatalf("明确成功不应触发核对，核对次数=%d", inspector.count())
	}
}

func TestExplicitFailurePreservesContentAndDraft(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, failure.New(failure.KindNonRetryable, "session.submit", errors.New("参数错误"))
	}}
	inspector := &fakeInspector{}
	manager := newTestManager(t, sender, inspector)

	if err := manager.SetDraft(Draft{SessionID: "s1", Text: "草稿"}); err != nil {
		t.Fatalf("保存草稿: %v", err)
	}
	first := mustRegister(t, manager, submitIntent("key-fail"))
	view := waitStatus(t, manager, first.ID, StatusFailed)
	if len(view.Content) == 0 {
		t.Fatal("失败后待发内容被清空")
	}
	if _, ok := manager.Draft("s1"); !ok {
		t.Fatal("失败后草稿被清空")
	}
	if sender.count() != 1 {
		t.Fatalf("失败后发送次数=%d 期望 1", sender.count())
	}
	if inspector.count() != 0 {
		t.Fatalf("明确失败不应触发核对，核对次数=%d", inspector.count())
	}
}

func TestTimeoutGoesUnknownAndVerificationHitSucceeds(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, context.DeadlineExceeded
	}}
	inspector := &fakeInspector{handle: func(query InputQuery, _ int) (InputState, error) {
		return InputState{
			Confirmed: true,
			Present:   true,
			SessionID: query.SessionID,
			BranchID:  query.BranchID,
		}, nil
	}}
	manager := newTestManager(t, sender, inspector)

	changes := subscribeChanges(manager)
	first := mustRegister(t, manager, submitIntent("key-timeout"))
	view := waitStatus(t, manager, first.ID, StatusSucceeded)
	if len(view.Content) != 0 {
		t.Fatal("核对命中成功后待发内容被保留")
	}
	if sender.count() != 1 {
		t.Fatalf("核对后发送次数=%d 期望 1（不自动重发）", sender.count())
	}
	if inspector.count() == 0 {
		t.Fatal("未知状态没有触发核对")
	}
	queries := inspector.queries()
	if queries[0].InputID != first.InputID {
		t.Fatalf("核对输入标识=%q 期望 %q", queries[0].InputID, first.InputID)
	}
	if !hasChange(changes, StatusUnknown) {
		t.Fatal("状态变化通知里没有 unknown")
	}
	if !hasChange(changes, StatusSucceeded) {
		t.Fatal("状态变化通知里没有 succeeded")
	}
}

func TestVerificationConfirmsNotOccurredAllowsResubmit(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, context.DeadlineExceeded
	}}
	inspector := &fakeInspector{handle: func(InputQuery, int) (InputState, error) {
		return InputState{Confirmed: true, Present: false}, nil
	}}
	manager := newTestManager(t, sender, inspector)

	first := mustRegister(t, manager, submitIntent("key-miss"))
	view := waitStatus(t, manager, first.ID, StatusNotOccurred)
	if len(view.Content) == 0 {
		t.Fatal("确认未发生后待发内容被清空")
	}
	if sender.count() != 1 {
		t.Fatalf("核对阶段发送次数=%d 期望 1", sender.count())
	}

	// 同一幂等键返回原意图，不重发；重提必须用新的幂等键。
	again, err := manager.Register(submitIntent("key-miss"))
	if err != nil {
		t.Fatalf("重复登记: %v", err)
	}
	if again.ID != first.ID || sender.count() != 1 {
		t.Fatalf("同一键重复登记引发重发：id=%s calls=%d", again.ID, sender.count())
	}

	sender.mu.Lock()
	sender.handle = func(context.Context, Request) (Reply, error) {
		return Reply{InputID: "server-input-2"}, nil
	}
	sender.mu.Unlock()
	retried := mustRegister(t, manager, submitIntent("key-miss-retry"))
	if retried.ID == first.ID {
		t.Fatal("新键登记复用了旧意图")
	}
	waitStatus(t, manager, retried.ID, StatusSucceeded)
	if sender.count() != 2 {
		t.Fatalf("重提后发送次数=%d 期望 2", sender.count())
	}
}

func TestVerificationUnconfirmedReachesManualAfterLimit(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, context.DeadlineExceeded
	}}
	inspector := &fakeInspector{handle: func(InputQuery, int) (InputState, error) {
		return InputState{Confirmed: false}, nil
	}}
	manager := newTestManager(t, sender, inspector)
	if err := manager.SetDraft(Draft{SessionID: "s1", Text: "草稿"}); err != nil {
		t.Fatalf("保存草稿: %v", err)
	}

	first := mustRegister(t, manager, submitIntent("key-unknown"))
	view := waitStatus(t, manager, first.ID, StatusNeedsManualConfirmation)
	if len(view.Content) == 0 {
		t.Fatal("需人工确认时待发内容被清空")
	}
	if _, ok := manager.Draft("s1"); !ok {
		t.Fatal("需人工确认时草稿被清空")
	}
	if sender.count() != 1 {
		t.Fatalf("发送次数=%d 期望 1", sender.count())
	}
	if inspector.count() != 3 {
		t.Fatalf("核对次数=%d 期望上限 3", inspector.count())
	}
}

func TestOwnershipMismatchReachesManual(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, context.DeadlineExceeded
	}}
	inspector := &fakeInspector{handle: func(InputQuery, int) (InputState, error) {
		return InputState{Confirmed: true, Present: true, SessionID: "s2", BranchID: "main"}, nil
	}}
	manager := newTestManager(t, sender, inspector)

	first := mustRegister(t, manager, submitIntent("key-other"))
	waitStatus(t, manager, first.ID, StatusNeedsManualConfirmation)
	if inspector.count() != 3 {
		t.Fatalf("归属不一致仍应继续核对到上限，核对次数=%d", inspector.count())
	}
}

// gateSender 把每次发送卡在 release 上，便于在途时改草稿或关闭。
type gateSender struct {
	started chan Request
	release chan Reply
	mu      sync.Mutex
	calls   int
}

func newGateSender() *gateSender {
	return &gateSender{
		started: make(chan Request, 8),
		release: make(chan Reply, 8),
	}
}

func (s *gateSender) Send(ctx context.Context, req Request) (Reply, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	s.started <- cloneRequest(req)
	select {
	case reply := <-s.release:
		return reply, nil
	case <-ctx.Done():
		return Reply{}, ctx.Err()
	}
}

func (s *gateSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestSubmitSuccessClearsOnlyMatchingDraft(t *testing.T) {
	t.Run("未改动时清空", func(t *testing.T) {
		sender := newGateSender()
		manager := newTestManager(t, sender, &fakeInspector{})
		if err := manager.SetDraft(Draft{SessionID: "s1", Text: "旧草稿"}); err != nil {
			t.Fatal(err)
		}
		first := mustRegister(t, manager, submitIntent("key-draft-clear"))
		<-sender.started
		sender.release <- Reply{InputID: "server-input"}
		waitStatus(t, manager, first.ID, StatusSucceeded)
		if _, ok := manager.Draft("s1"); ok {
			t.Fatal("提交成功后匹配的草稿没有被清空")
		}
	})

	t.Run("用户已改草稿时保留新草稿", func(t *testing.T) {
		sender := newGateSender()
		manager := newTestManager(t, sender, &fakeInspector{})
		if err := manager.SetDraft(Draft{SessionID: "s1", Text: "旧草稿"}); err != nil {
			t.Fatal(err)
		}
		first := mustRegister(t, manager, submitIntent("key-draft-keep"))
		<-sender.started
		if err := manager.SetDraft(Draft{SessionID: "s1", Text: "新草稿"}); err != nil {
			t.Fatal(err)
		}
		sender.release <- Reply{InputID: "server-input"}
		waitStatus(t, manager, first.ID, StatusSucceeded)
		draft, ok := manager.Draft("s1")
		if !ok || draft.Text != "新草稿" {
			t.Fatalf("新草稿被旧提交抹掉: %+v ok=%v", draft, ok)
		}
	})

	t.Run("清空后重新输入的草稿不被旧提交抹掉", func(t *testing.T) {
		sender := newGateSender()
		manager := newTestManager(t, sender, &fakeInspector{})
		if err := manager.SetDraft(Draft{SessionID: "s1", Text: "旧草稿"}); err != nil {
			t.Fatal(err)
		}
		first := mustRegister(t, manager, submitIntent("key-draft-reuse"))
		<-sender.started
		manager.ClearDraft("s1")
		if err := manager.SetDraft(Draft{SessionID: "s1", Text: "新草稿"}); err != nil {
			t.Fatal(err)
		}
		sender.release <- Reply{InputID: "server-input"}
		waitStatus(t, manager, first.ID, StatusSucceeded)
		draft, ok := manager.Draft("s1")
		if !ok || draft.Text != "新草稿" {
			t.Fatalf("清空后重输的草稿被旧提交抹掉: %+v ok=%v", draft, ok)
		}
	})
}

func TestInFlightTableAndChangeNotifications(t *testing.T) {
	sender := newGateSender()
	manager := newTestManager(t, sender, &fakeInspector{})
	changes := subscribeChanges(manager)

	first := mustRegister(t, manager, submitIntent("key-table"))
	<-sender.started

	inflight := manager.InFlight()
	if len(inflight) != 1 || inflight[0].ID != first.ID || inflight[0].Status != StatusInFlight {
		t.Fatalf("在途表=%+v 期望一条 in_flight", inflight)
	}
	if table := manager.Table(); len(table) != 1 {
		t.Fatalf("全表=%d 期望 1", len(table))
	}

	sender.release <- Reply{InputID: "server-input"}
	waitStatus(t, manager, first.ID, StatusSucceeded)
	if got := manager.InFlight(); len(got) != 0 {
		t.Fatalf("成功后仍在途: %+v", got)
	}
	if table := manager.Table(); len(table) != 1 || table[0].Status != StatusSucceeded {
		t.Fatalf("全表=%+v 期望保留成功记录", table)
	}
	got := waitForChanges(t, changes, 3)
	want := []Status{StatusRegistered, StatusInFlight, StatusSucceeded}
	for i := range want {
		if got[i].Status != want[i] {
			t.Fatalf("状态变化=%v 期望 %v", changeList(got), want)
		}
	}
}

func TestAllKindsAreSentOnce(t *testing.T) {
	cases := []struct {
		kind    Kind
		session string
		params  json.RawMessage
	}{
		{KindSubmitInput, "s1", nil},
		{KindStop, "s1", nil},
		{KindApprovalResponse, "s1", nil},
		{KindLongTermAllow, "s1", nil},
		{KindQuestionAnswer, "s1", nil},
		{KindManagementWrite, "", nil},
		{KindAccountAction, "", nil},
		{KindAttachmentWrite, "s1", nil},
		{KindSessionLifecycle, "", json.RawMessage(`{"op":"create","cwd":"/tmp"}`)},
		{KindShellRun, "s1", json.RawMessage(`{"op":"run","runId":"run-1","command":"pwd"}`)},
		{KindModalityWrite, "s1", json.RawMessage(`{"op":"task.submit"}`)},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
				return Reply{InputID: "server-input"}, nil
			}}
			manager := newTestManager(t, sender, &fakeInspector{})
			params := tc.params
			if params == nil {
				params = json.RawMessage(`{"value":1}`)
			}
			intent := Intent{
				Key:       "key-" + string(tc.kind),
				Kind:      tc.kind,
				SessionID: tc.session,
				Params:    params,
			}
			view := mustRegister(t, manager, intent)
			waitStatus(t, manager, view.ID, StatusSucceeded)
			if sender.count() != 1 {
				t.Fatalf("发送次数=%d 期望 1", sender.count())
			}
			call := sender.requests()[0]
			if call.Kind != tc.kind || call.SessionID != tc.session {
				t.Fatalf("发送请求=%+v", call)
			}
			if call.Delivery != DeliveryQueue {
				t.Fatalf("默认投递方式=%q 期望 queue", call.Delivery)
			}
			if tc.kind == KindSubmitInput && call.InputID == "" {
				t.Fatal("提交类意图没有生成输入标识")
			}
		})
	}
}

func TestSessionLifecycleIntentIsSentOnce(t *testing.T) {
	cases := []struct {
		name       string
		session    string
		branchID   string
		wantBranch string
		params     json.RawMessage
	}{
		{name: "create", session: "", params: json.RawMessage(`{"op":"create","cwd":"/tmp"}`)},
		{name: "create-catalog-op", session: "", params: json.RawMessage(`{"op":"catalog.create","cwd":"/tmp"}`)},
		{name: "rename", session: "s1", params: json.RawMessage(`{"op":"rename","title":"新名字"}`)},
		{name: "delete", session: "s1", params: json.RawMessage(`{"op":"delete"}`)},
		{
			name:    "branch.create",
			session: "s1",
			params:  json.RawMessage(`{"op":"branch.create","parentBranchId":"main","at":3,"prompt":"换一条路"}`),
		},
		{
			name:       "branch.end",
			session:    "s1",
			branchID:   "side",
			wantBranch: "side",
			params:     json.RawMessage(`{"op":"branch.end"}`),
		},
		{
			name:       "branch.end-param-id",
			session:    "s1",
			wantBranch: "side",
			params:     json.RawMessage(`{"op":"branch.end","branchId":"side"}`),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
				return Reply{InputID: "server-" + tc.name}, nil
			}}
			manager := newTestManager(t, sender, &fakeInspector{})
			view := mustRegister(t, manager, Intent{
				Key:       "lifecycle-" + tc.name,
				Kind:      KindSessionLifecycle,
				SessionID: tc.session,
				BranchID:  tc.branchID,
				Params:    tc.params,
			})
			if view.BranchID != tc.wantBranch {
				t.Fatalf("登记后分支=%q 期望 %q", view.BranchID, tc.wantBranch)
			}
			waitStatus(t, manager, view.ID, StatusSucceeded)
			if sender.count() != 1 {
				t.Fatalf("发送次数=%d 期望 1", sender.count())
			}
			call := sender.requests()[0]
			if call.Kind != KindSessionLifecycle || call.SessionID != tc.session || call.BranchID != tc.wantBranch {
				t.Fatalf("发送请求=%+v", call)
			}
			if string(call.Params) != string(tc.params) {
				t.Fatalf("参数=%s 期望原样下发 %s", call.Params, tc.params)
			}
			if call.Delivery != DeliveryQueue {
				t.Fatalf("默认投递方式=%q 期望 queue", call.Delivery)
			}
		})
	}
}

func TestSessionLifecycleUnknownVerification(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, context.DeadlineExceeded
	}}
	inspector := &fakeInspector{handle: func(query InputQuery, _ int) (InputState, error) {
		if query.InputID != "create-1" {
			t.Errorf("核对输入标识=%q 期望 create-1", query.InputID)
		}
		if query.SessionID != "" {
			t.Errorf("新建会话核对不应带会话标识，实际 %q", query.SessionID)
		}
		// 新建成功后服务端会给出会话标识；意图登记时没有，因此核对允许读到它。
		return InputState{
			Confirmed: true,
			Present:   true,
			SessionID: "s-new",
			BranchID:  "main",
		}, nil
	}}
	manager := newTestManager(t, sender, inspector)
	view := mustRegister(t, manager, Intent{
		Key:     "lifecycle-create-unknown",
		Kind:    KindSessionLifecycle,
		InputID: "create-1",
		Params:  json.RawMessage(`{"op":"create","cwd":"/tmp"}`),
	})
	result := waitStatus(t, manager, view.ID, StatusSucceeded)
	if result.InputID != "create-1" {
		t.Fatalf("成功后输入标识=%q 期望 create-1", result.InputID)
	}
	if sender.count() != 1 {
		t.Fatalf("核对后发送次数=%d 期望 1（不自动重发）", sender.count())
	}
	if inspector.count() == 0 {
		t.Fatal("未知状态没有触发核对")
	}
}

func TestSessionLifecycleBranchOpsUnknownAndFailure(t *testing.T) {
	cases := []struct {
		name     string
		branchID string
		params   json.RawMessage
	}{
		{
			name:   "branch.create",
			params: json.RawMessage(`{"op":"branch.create","parentBranchId":"main","at":3,"prompt":"换一条路"}`),
		},
		{
			name:     "branch.end",
			branchID: "side",
			params:   json.RawMessage(`{"op":"branch.end"}`),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/unknown", func(t *testing.T) {
			sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
				return Reply{}, context.DeadlineExceeded
			}}
			inspector := &fakeInspector{handle: func(query InputQuery, _ int) (InputState, error) {
				if query.Kind != KindSessionLifecycle || query.SessionID != "s1" {
					t.Errorf("核对查询=%+v 期望 session_lifecycle/s1", query)
				}
				return InputState{
					Confirmed: true,
					Present:   true,
					SessionID: "s1",
					BranchID:  query.BranchID,
				}, nil
			}}
			manager := newTestManager(t, sender, inspector)
			view := mustRegister(t, manager, Intent{
				Key:       "branch-" + tc.name + "-unknown",
				Kind:      KindSessionLifecycle,
				SessionID: "s1",
				BranchID:  tc.branchID,
				Params:    tc.params,
			})
			waitStatus(t, manager, view.ID, StatusSucceeded)
			if sender.count() != 1 {
				t.Fatalf("核对后发送次数=%d 期望 1（不自动重发）", sender.count())
			}
			if inspector.count() == 0 {
				t.Fatal("未知状态没有触发核对")
			}
		})

		t.Run(tc.name+"/failure", func(t *testing.T) {
			sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
				return Reply{}, failure.New(failure.KindNonRetryable, "branch", errors.New("参数错误"))
			}}
			inspector := &fakeInspector{}
			manager := newTestManager(t, sender, inspector)
			if err := manager.SetDraft(Draft{SessionID: "s1", Text: "草稿"}); err != nil {
				t.Fatalf("保存草稿: %v", err)
			}
			view := mustRegister(t, manager, Intent{
				Key:       "branch-" + tc.name + "-fail",
				Kind:      KindSessionLifecycle,
				SessionID: "s1",
				BranchID:  tc.branchID,
				Params:    tc.params,
			})
			result := waitStatus(t, manager, view.ID, StatusFailed)
			if len(result.Content) == 0 {
				t.Fatal("失败后待发内容被清空")
			}
			if _, ok := manager.Draft("s1"); !ok {
				t.Fatal("失败后草稿被清空")
			}
			if sender.count() != 1 || inspector.count() != 0 {
				t.Fatalf("发送=%d 核对=%d 期望 1/0", sender.count(), inspector.count())
			}
		})
	}
}

func TestSessionLifecycleFailurePreservesContentAndDraft(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, failure.New(failure.KindNonRetryable, "session.rename", errors.New("会话不存在"))
	}}
	inspector := &fakeInspector{}
	manager := newTestManager(t, sender, inspector)
	if err := manager.SetDraft(Draft{SessionID: "s1", Text: "草稿"}); err != nil {
		t.Fatalf("保存草稿: %v", err)
	}
	view := mustRegister(t, manager, Intent{
		Key:       "lifecycle-rename-fail",
		Kind:      KindSessionLifecycle,
		SessionID: "s1",
		Params:    json.RawMessage(`{"op":"rename","title":"新名字"}`),
	})
	result := waitStatus(t, manager, view.ID, StatusFailed)
	if len(result.Content) == 0 {
		t.Fatal("失败后待发内容被清空")
	}
	if _, ok := manager.Draft("s1"); !ok {
		t.Fatal("失败后草稿被清空")
	}
	if sender.count() != 1 {
		t.Fatalf("失败后发送次数=%d 期望 1", sender.count())
	}
	if inspector.count() != 0 {
		t.Fatalf("明确失败不应核对，核对次数=%d", inspector.count())
	}
}

func TestSessionLifecycleScope(t *testing.T) {
	manager := newTestManager(t, &fakeSender{}, &fakeInspector{})
	accepted := Intent{
		Key:    "lifecycle-create-ok",
		Kind:   KindSessionLifecycle,
		Params: json.RawMessage(`{"op":"create","cwd":"/tmp"}`),
	}
	if _, err := manager.Register(accepted); err != nil {
		t.Fatalf("新建会话不应要求会话标识: %v", err)
	}
	if _, err := manager.Register(Intent{
		Key:    "lifecycle-create-catalog-op",
		Kind:   KindSessionLifecycle,
		Params: json.RawMessage(`{"op":"catalog.create","cwd":"/tmp"}`),
	}); err != nil {
		t.Fatalf("目录领域的 catalog.create 写法应被接受: %v", err)
	}

	cases := []struct {
		name   string
		intent Intent
	}{
		{
			name: "改名缺少目标会话",
			intent: Intent{
				Key:    "lifecycle-rename-no-session",
				Kind:   KindSessionLifecycle,
				Params: json.RawMessage(`{"op":"rename","title":"新名字"}`),
			},
		},
		{
			name: "删除缺少目标会话",
			intent: Intent{
				Key:    "lifecycle-delete-no-session",
				Kind:   KindSessionLifecycle,
				Params: json.RawMessage(`{"op":"delete"}`),
			},
		},
		{
			name: "缺少 op",
			intent: Intent{
				Key:    "lifecycle-no-op",
				Kind:   KindSessionLifecycle,
				Params: json.RawMessage(`{"title":"新名字"}`),
			},
		},
		{
			name: "branch.create 缺少目标会话",
			intent: Intent{
				Key:    "lifecycle-branch-create-no-session",
				Kind:   KindSessionLifecycle,
				Params: json.RawMessage(`{"op":"branch.create","parentBranchId":"main","at":3,"prompt":"换一条路"}`),
			},
		},
		{
			name: "branch.end 缺少目标会话",
			intent: Intent{
				Key:      "lifecycle-branch-end-no-session",
				Kind:     KindSessionLifecycle,
				BranchID: "side",
				Params:   json.RawMessage(`{"op":"branch.end"}`),
			},
		},
		{
			name: "branch.end 缺少分支标识",
			intent: Intent{
				Key:       "lifecycle-branch-end-no-branch",
				Kind:      KindSessionLifecycle,
				SessionID: "s1",
				Params:    json.RawMessage(`{"op":"branch.end"}`),
			},
		},
		{
			name: "branch.end 分支标识不一致",
			intent: Intent{
				Key:       "lifecycle-branch-end-conflict",
				Kind:      KindSessionLifecycle,
				SessionID: "s1",
				BranchID:  "side",
				Params:    json.RawMessage(`{"op":"branch.end","branchId":"other"}`),
			},
		},
		{
			name: "catalog.rename 缺少目标会话",
			intent: Intent{
				Key:    "lifecycle-catalog-rename-no-session",
				Kind:   KindSessionLifecycle,
				Params: json.RawMessage(`{"op":"catalog.rename","title":"新名字"}`),
			},
		},
		{
			name: "catalog.delete 缺少目标会话",
			intent: Intent{
				Key:    "lifecycle-catalog-delete-no-session",
				Kind:   KindSessionLifecycle,
				Params: json.RawMessage(`{"op":"catalog.delete"}`),
			},
		},
		{
			name: "未知 op",
			intent: Intent{
				Key:       "lifecycle-unknown-op",
				Kind:      KindSessionLifecycle,
				SessionID: "s1",
				Params:    json.RawMessage(`{"op":"archive"}`),
			},
		},
		{
			name: "参数不是对象",
			intent: Intent{
				Key:    "lifecycle-array-params",
				Kind:   KindSessionLifecycle,
				Params: json.RawMessage(`[]`),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := manager.Register(tc.intent); !errors.Is(err, ErrInvalidIntent) {
				t.Fatalf("错误=%v 期望 ErrInvalidIntent", err)
			}
		})
	}
}

func TestShellRunIntentIsSentOnce(t *testing.T) {
	cases := []struct {
		name   string
		params json.RawMessage
	}{
		{name: "run", params: json.RawMessage(`{"op":"run","runId":"run-1","command":"pwd"}`)},
		{name: "stop", params: json.RawMessage(`{"op":"stop","runId":"run-1"}`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
				return Reply{InputID: "server-shell-" + tc.name}, nil
			}}
			manager := newTestManager(t, sender, &fakeInspector{})
			view := mustRegister(t, manager, Intent{
				Key:       "shell-" + tc.name,
				Kind:      KindShellRun,
				SessionID: "s1",
				InputID:   "run-1",
				Params:    tc.params,
			})
			waitStatus(t, manager, view.ID, StatusSucceeded)
			if sender.count() != 1 {
				t.Fatalf("发送次数=%d 期望 1", sender.count())
			}
			call := sender.requests()[0]
			if call.Kind != KindShellRun || call.SessionID != "s1" || call.InputID != "run-1" {
				t.Fatalf("发送请求=%+v", call)
			}
			if string(call.Params) != string(tc.params) {
				t.Fatalf("参数=%s 期望原样下发 %s", call.Params, tc.params)
			}
		})
	}
}

func TestShellRunUnknownVerification(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, context.DeadlineExceeded
	}}
	inspector := &fakeInspector{handle: func(query InputQuery, _ int) (InputState, error) {
		if query.Kind != KindShellRun || query.InputID != "run-1" || query.SessionID != "s1" {
			t.Errorf("核对查询=%+v 期望 shell_run/s1/run-1", query)
		}
		return InputState{Confirmed: true, Present: true, SessionID: "s1"}, nil
	}}
	manager := newTestManager(t, sender, inspector)
	view := mustRegister(t, manager, Intent{
		Key:       "shell-unknown",
		Kind:      KindShellRun,
		SessionID: "s1",
		InputID:   "run-1",
		Params:    json.RawMessage(`{"op":"run","runId":"run-1","command":"pwd"}`),
	})
	waitStatus(t, manager, view.ID, StatusSucceeded)
	if sender.count() != 1 {
		t.Fatalf("核对后发送次数=%d 期望 1（不自动重发）", sender.count())
	}
	if inspector.count() == 0 {
		t.Fatal("未知状态没有触发核对")
	}
}

func TestShellRunFailurePreservesContentAndDraft(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, failure.New(failure.KindNonRetryable, "shell.run", errors.New("能力缺失"))
	}}
	inspector := &fakeInspector{}
	manager := newTestManager(t, sender, inspector)
	if err := manager.SetDraft(Draft{SessionID: "s1", Text: "草稿"}); err != nil {
		t.Fatalf("保存草稿: %v", err)
	}
	view := mustRegister(t, manager, Intent{
		Key:       "shell-fail",
		Kind:      KindShellRun,
		SessionID: "s1",
		Params:    json.RawMessage(`{"op":"run","runId":"run-1","command":"pwd"}`),
	})
	result := waitStatus(t, manager, view.ID, StatusFailed)
	if len(result.Content) == 0 {
		t.Fatal("失败后待发内容被清空")
	}
	if _, ok := manager.Draft("s1"); !ok {
		t.Fatal("失败后草稿被清空")
	}
	if sender.count() != 1 || inspector.count() != 0 {
		t.Fatalf("发送=%d 核对=%d 期望 1/0", sender.count(), inspector.count())
	}
}

func TestModalityWriteIntentIsSentOnce(t *testing.T) {
	ops := []string{"task.submit", "task.cancel", "realtime.open", "realtime.close"}
	for _, op := range ops {
		t.Run(op, func(t *testing.T) {
			params := json.RawMessage(`{"op":"` + op + `"}`)
			sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
				return Reply{InputID: "server-modality"}, nil
			}}
			manager := newTestManager(t, sender, &fakeInspector{})
			view := mustRegister(t, manager, Intent{
				Key:       "modality-" + op,
				Kind:      KindModalityWrite,
				SessionID: "s1",
				Params:    params,
			})
			waitStatus(t, manager, view.ID, StatusSucceeded)
			if sender.count() != 1 {
				t.Fatalf("发送次数=%d 期望 1", sender.count())
			}
			call := sender.requests()[0]
			if call.Kind != KindModalityWrite || call.SessionID != "s1" {
				t.Fatalf("发送请求=%+v", call)
			}
			if string(call.Params) != string(params) {
				t.Fatalf("参数=%s 期望原样下发 %s", call.Params, params)
			}
		})
	}
}

func TestModalityWriteUnknownVerification(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, context.DeadlineExceeded
	}}
	inspector := &fakeInspector{handle: func(query InputQuery, _ int) (InputState, error) {
		if query.Kind != KindModalityWrite || query.InputID != "task-1" || query.SessionID != "s1" {
			t.Errorf("核对查询=%+v 期望 modality_write/s1/task-1", query)
		}
		return InputState{Confirmed: true, Present: true, SessionID: "s1"}, nil
	}}
	manager := newTestManager(t, sender, inspector)
	view := mustRegister(t, manager, Intent{
		Key:       "modality-unknown",
		Kind:      KindModalityWrite,
		SessionID: "s1",
		InputID:   "task-1",
		Params:    json.RawMessage(`{"op":"task.cancel","taskId":"task-1"}`),
	})
	waitStatus(t, manager, view.ID, StatusSucceeded)
	if sender.count() != 1 {
		t.Fatalf("核对后发送次数=%d 期望 1（不自动重发）", sender.count())
	}
	if inspector.count() == 0 {
		t.Fatal("未知状态没有触发核对")
	}
}

func TestModalityWriteFailurePreservesContentAndDraft(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, failure.New(failure.KindNonRetryable, "modality.task.submit", errors.New("能力缺失"))
	}}
	inspector := &fakeInspector{}
	manager := newTestManager(t, sender, inspector)
	if err := manager.SetDraft(Draft{SessionID: "s1", Text: "草稿"}); err != nil {
		t.Fatalf("保存草稿: %v", err)
	}
	view := mustRegister(t, manager, Intent{
		Key:       "modality-fail",
		Kind:      KindModalityWrite,
		SessionID: "s1",
		Params:    json.RawMessage(`{"op":"realtime.open"}`),
	})
	result := waitStatus(t, manager, view.ID, StatusFailed)
	if len(result.Content) == 0 {
		t.Fatal("失败后待发内容被清空")
	}
	if _, ok := manager.Draft("s1"); !ok {
		t.Fatal("失败后草稿被清空")
	}
	if sender.count() != 1 || inspector.count() != 0 {
		t.Fatalf("发送=%d 核对=%d 期望 1/0", sender.count(), inspector.count())
	}
}

func TestShellRunAndModalityScope(t *testing.T) {
	manager := newTestManager(t, &fakeSender{}, &fakeInspector{})
	cases := []struct {
		name   string
		intent Intent
	}{
		{
			name: "shell 缺少目标会话",
			intent: Intent{
				Key:    "shell-no-session",
				Kind:   KindShellRun,
				Params: json.RawMessage(`{"op":"run","runId":"run-1","command":"pwd"}`),
			},
		},
		{
			name: "shell 缺少 op",
			intent: Intent{
				Key:       "shell-no-op",
				Kind:      KindShellRun,
				SessionID: "s1",
				Params:    json.RawMessage(`{"command":"pwd"}`),
			},
		},
		{
			name: "shell 未知 op",
			intent: Intent{
				Key:       "shell-unknown-op",
				Kind:      KindShellRun,
				SessionID: "s1",
				Params:    json.RawMessage(`{"op":"kill","runId":"run-1"}`),
			},
		},
		{
			name: "shell run 缺少命令",
			intent: Intent{
				Key:       "shell-run-no-command",
				Kind:      KindShellRun,
				SessionID: "s1",
				Params:    json.RawMessage(`{"op":"run","runId":"run-1","command":"  "}`),
			},
		},
		{
			name: "shell stop 缺少运行标识",
			intent: Intent{
				Key:       "shell-stop-no-run",
				Kind:      KindShellRun,
				SessionID: "s1",
				Params:    json.RawMessage(`{"op":"stop"}`),
			},
		},
		{
			name: "modality 缺少目标会话",
			intent: Intent{
				Key:    "modality-no-session",
				Kind:   KindModalityWrite,
				Params: json.RawMessage(`{"op":"task.submit"}`),
			},
		},
		{
			name: "modality 缺少 op",
			intent: Intent{
				Key:       "modality-no-op",
				Kind:      KindModalityWrite,
				SessionID: "s1",
				Params:    json.RawMessage(`{}`),
			},
		},
		{
			name: "modality 未知 op",
			intent: Intent{
				Key:       "modality-unknown-op",
				Kind:      KindModalityWrite,
				SessionID: "s1",
				Params:    json.RawMessage(`{"op":"task.pause"}`),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := manager.Register(tc.intent); !errors.Is(err, ErrInvalidIntent) {
				t.Fatalf("错误=%v 期望 ErrInvalidIntent", err)
			}
		})
	}
}

func TestInvalidIntentsAreRejected(t *testing.T) {
	manager := newTestManager(t, &fakeSender{}, &fakeInspector{})
	cases := []struct {
		name   string
		intent Intent
	}{
		{name: "缺少幂等键", intent: Intent{Kind: KindStop, SessionID: "s1"}},
		{name: "未知类型", intent: Intent{Key: "k", Kind: "unknown", SessionID: "s1"}},
		{name: "缺少目标会话", intent: Intent{Key: "k", Kind: KindSubmitInput}},
		{name: "未知投递方式", intent: Intent{Key: "k", Kind: KindSubmitInput, SessionID: "s1", Delivery: "later"}},
		{name: "参数不是合法 JSON", intent: Intent{Key: "k", Kind: KindSubmitInput, SessionID: "s1", Params: json.RawMessage(`{`)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := manager.Register(tc.intent); !errors.Is(err, ErrInvalidIntent) {
				t.Fatalf("错误=%v 期望 ErrInvalidIntent", err)
			}
		})
	}
	if err := manager.SetDraft(Draft{Text: "没有会话"}); !errors.Is(err, ErrInvalidIntent) {
		t.Fatalf("无会话草稿错误=%v 期望 ErrInvalidIntent", err)
	}
}

func TestKeyConflictIsRejected(t *testing.T) {
	sender := &fakeSender{handle: func(context.Context, Request) (Reply, error) {
		return Reply{}, nil
	}}
	manager := newTestManager(t, sender, &fakeInspector{})
	first := mustRegister(t, manager, submitIntent("key-conflict"))
	waitStatus(t, manager, first.ID, StatusSucceeded)

	other := submitIntent("key-conflict")
	other.Params = json.RawMessage(`{"text":"另一句"}`)
	if _, err := manager.Register(other); !errors.Is(err, ErrKeyConflict) {
		t.Fatalf("冲突错误=%v 期望 ErrKeyConflict", err)
	}
	if sender.count() != 1 {
		t.Fatalf("冲突登记引发了发送，次数=%d", sender.count())
	}
}

func TestClosedManagerRejectsNewIntents(t *testing.T) {
	manager := newTestManager(t, &fakeSender{}, &fakeInspector{})
	manager.Close()
	if _, err := manager.Register(submitIntent("key-closed")); !errors.Is(err, ErrClosed) {
		t.Fatalf("关闭后登记错误=%v 期望 ErrClosed", err)
	}
	if err := manager.SetDraft(Draft{SessionID: "s1", Text: "草稿"}); !errors.Is(err, ErrClosed) {
		t.Fatalf("关闭后保存草稿错误=%v 期望 ErrClosed", err)
	}
}

func TestCloseMarksUnsettledIntentAsManual(t *testing.T) {
	sender := newGateSender()
	manager := newTestManager(t, sender, &fakeInspector{})
	first := mustRegister(t, manager, submitIntent("key-close"))
	<-sender.started

	manager.Close()
	view, ok := manager.Result(first.ID)
	if !ok {
		t.Fatal("关闭后意图从表中消失")
	}
	if view.Status != StatusNeedsManualConfirmation {
		t.Fatalf("关闭后在途意图状态=%s 期望 needs_manual_confirmation", view.Status)
	}
	if len(view.Content) == 0 {
		t.Fatal("关闭后待发内容被清空")
	}
	if sender.count() != 1 {
		t.Fatalf("关闭后发送次数=%d 期望 1", sender.count())
	}
}

func subscribeChanges(manager *Manager) chan Change {
	changes := make(chan Change, 16)
	manager.Subscribe(func(change Change) { changes <- change })
	return changes
}

func waitForChanges(t *testing.T, changes chan Change, count int) []Change {
	t.Helper()
	got := make([]Change, 0, count)
	deadline := time.After(2 * time.Second)
	for len(got) < count {
		select {
		case change := <-changes:
			got = append(got, change)
		case <-deadline:
			t.Fatalf("等待状态变化超时：已有 %v", got)
		}
	}
	return got
}

func hasChange(changes chan Change, want Status) bool {
	deadline := time.After(2 * time.Second)
	for {
		select {
		case change := <-changes:
			if change.Status == want {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

func changeList(changes []Change) []Status {
	statuses := make([]Status, 0, len(changes))
	for _, change := range changes {
		statuses = append(statuses, change.Status)
	}
	return statuses
}
