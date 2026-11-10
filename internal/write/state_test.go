package write

import (
	"reflect"
	"testing"
)

// TestTransitionTableMatchesDocument 逐条核对 A.md「在途写」迁移表。
func TestTransitionTableMatchesDocument(t *testing.T) {
	want := map[Status]map[transitionEvent]Status{
		StatusRegistered: {
			eventSend:  StatusInFlight,
			eventClose: StatusNotOccurred,
		},
		StatusInFlight: {
			eventSucceed: StatusSucceeded,
			eventFail:    StatusFailed,
			eventUnknown: StatusUnknown,
			eventClose:   StatusNeedsManualConfirmation,
		},
		StatusUnknown: {
			eventVerifyHit:         StatusSucceeded,
			eventVerifyMiss:        StatusNotOccurred,
			eventVerifyUnconfirmed: StatusUnknown,
			eventVerifyExhausted:   StatusNeedsManualConfirmation,
			eventClose:             StatusNeedsManualConfirmation,
		},
	}
	if !reflect.DeepEqual(transitionTable, want) {
		t.Fatalf("迁移表与 A.md 不一致:\n实际 %#v\n期望 %#v", transitionTable, want)
	}
}

func TestTransitionTableIsValid(t *testing.T) {
	if err := validateTransitionTable(); err != nil {
		t.Fatalf("迁移表不合法: %v", err)
	}
}

func TestBrokenTransitionTablesAreRejected(t *testing.T) {
	cases := []struct {
		name  string
		table map[Status]map[transitionEvent]Status
	}{
		{
			name: "未知目标状态",
			table: map[Status]map[transitionEvent]Status{
				StatusRegistered: {eventSend: "nowhere", eventClose: StatusNotOccurred},
				StatusInFlight:   {eventClose: StatusNeedsManualConfirmation},
				StatusUnknown:    {eventClose: StatusNeedsManualConfirmation},
			},
		},
		{
			name: "未知事件",
			table: map[Status]map[transitionEvent]Status{
				StatusRegistered: {"teleport": StatusInFlight, eventClose: StatusNotOccurred},
				StatusInFlight:   {eventClose: StatusNeedsManualConfirmation},
				StatusUnknown:    {eventClose: StatusNeedsManualConfirmation},
			},
		},
		{
			name: "终态仍有出边",
			table: map[Status]map[transitionEvent]Status{
				StatusRegistered: {eventSend: StatusInFlight, eventClose: StatusNotOccurred},
				StatusInFlight:   {eventClose: StatusNeedsManualConfirmation},
				StatusUnknown:    {eventClose: StatusNeedsManualConfirmation},
				StatusSucceeded:  {eventFail: StatusFailed},
			},
		},
		{
			name: "非终态缺少关闭迁移",
			table: map[Status]map[transitionEvent]Status{
				StatusRegistered: {eventSend: StatusInFlight},
				StatusInFlight:   {eventClose: StatusNeedsManualConfirmation},
				StatusUnknown:    {eventClose: StatusNeedsManualConfirmation},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateTransitionTableFor(tc.table); err == nil {
				t.Fatal("坏迁移表没有被拒绝")
			}
		})
	}
}

// TestJudgeOwnership 覆盖核对判定的三条收敛路径。
func TestJudgeOwnership(t *testing.T) {
	query := InputQuery{SessionID: "s1", BranchID: "main", InputID: "input-1"}
	cases := []struct {
		name  string
		state InputState
		err   error
		want  verdict
	}{
		{name: "读取失败", state: InputState{}, err: errLookup, want: verifyUnconfirmed},
		{name: "读取不可信", state: InputState{Confirmed: false, Present: true}, want: verifyUnconfirmed},
		{name: "命中且归属一致", state: InputState{Confirmed: true, Present: true, SessionID: "s1", BranchID: "main"}, want: verifyHit},
		{name: "命中但会话不一致", state: InputState{Confirmed: true, Present: true, SessionID: "s2", BranchID: "main"}, want: verifyUnconfirmed},
		{name: "命中但分支不一致", state: InputState{Confirmed: true, Present: true, SessionID: "s1", BranchID: "side"}, want: verifyUnconfirmed},
		{name: "命中但归属为空", state: InputState{Confirmed: true, Present: true}, want: verifyUnconfirmed},
		{name: "确认未发生", state: InputState{Confirmed: true, Present: false, SessionID: "s1", BranchID: "main"}, want: verifyMiss},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := judge(query, tc.state, tc.err); got != tc.want {
				t.Fatalf("judge=%v 期望 %v", got, tc.want)
			}
		})
	}

	t.Run("无会话的生命周期意图允许读到新会话", func(t *testing.T) {
		q := InputQuery{Kind: KindSessionLifecycle, InputID: "create-1"}
		state := InputState{Confirmed: true, Present: true, SessionID: "s-new"}
		if got := judge(q, state, nil); got != verifyHit {
			t.Fatalf("judge=%v 期望 verifyHit", got)
		}
	})

	t.Run("其他无会话意图仍不确认归属", func(t *testing.T) {
		q := InputQuery{Kind: KindAccountAction, InputID: "reset-1"}
		state := InputState{Confirmed: true, Present: true, SessionID: "s1"}
		if got := judge(q, state, nil); got != verifyUnconfirmed {
			t.Fatalf("judge=%v 期望 verifyUnconfirmed", got)
		}
	})
}
