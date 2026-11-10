package interaction

import (
	"reflect"
	"strings"
	"testing"
)

// TestTransitionTablesMatchDocument 逐条核对 A.md 的两张推进表（含服务端提前
// 结算的补充出边）。
func TestTransitionTablesMatchDocument(t *testing.T) {
	wantApproval := map[State]map[stateEvent]State{
		StateNone: {
			eventIncoming: StatePresented,
		},
		StatePresented: {
			eventDecide:        StateAwaitingResponse,
			eventAllowLongTerm: StateAwaitingLongTerm,
			eventSettled:       StateNone,
		},
		StateAwaitingLongTerm: {
			eventLongTermWritten: StateAwaitingResponse,
			eventLongTermFailed:  StatePresented,
			eventSettled:         StateNone,
		},
		StateAwaitingResponse: {
			eventSettled: StateNone,
		},
	}
	if !reflect.DeepEqual(approvalTransitions, wantApproval) {
		t.Fatalf("审批推进表与文档不一致:\n实际 %#v\n期望 %#v", approvalTransitions, wantApproval)
	}

	wantQuestion := map[State]map[stateEvent]State{
		StateNone: {
			eventIncoming: StatePresented,
		},
		StatePresented: {
			eventDecide:  StateAnswered,
			eventSettled: StateNone,
		},
		StateAnswered: {
			eventSettled: StateNone,
		},
	}
	if !reflect.DeepEqual(questionTransitions, wantQuestion) {
		t.Fatalf("提问推进表与文档不一致:\n实际 %#v\n期望 %#v", questionTransitions, wantQuestion)
	}
}

func TestTransitionTablesAreValid(t *testing.T) {
	if err := validateTransitionTables(); err != nil {
		t.Fatalf("推进表不合法: %v", err)
	}
}

// TestBrokenTransitionTablesAreRejected 核对：坏表在启动时被拒绝并点名状态机。
func TestBrokenTransitionTablesAreRejected(t *testing.T) {
	cases := []struct {
		name  string
		table map[State]map[stateEvent]State
	}{
		{
			name: "未知目标状态",
			table: map[State]map[stateEvent]State{
				StateNone:      {eventIncoming: StatePresented},
				StatePresented: {eventDecide: "nowhere", eventSettled: StateNone},
			},
		},
		{
			name: "未知事件",
			table: map[State]map[stateEvent]State{
				StateNone:      {eventIncoming: StatePresented},
				StatePresented: {"teleport": StateAnswered, eventSettled: StateNone},
			},
		},
		{
			name: "缺少结算出边",
			table: map[State]map[stateEvent]State{
				StateNone:      {eventIncoming: StatePresented},
				StatePresented: {eventDecide: StateAnswered},
				StateAnswered:  {eventSettled: StateNone},
			},
		},
		{
			name: "无待决不是入口",
			table: map[State]map[stateEvent]State{
				StatePresented: {eventIncoming: StateAnswered, eventSettled: StateNone},
				StateAnswered:  {eventSettled: StateNone},
			},
		},
		{
			name: "无待决出现别的出边",
			table: map[State]map[stateEvent]State{
				StateNone:      {eventIncoming: StatePresented, eventDecide: StateAnswered},
				StatePresented: {eventSettled: StateNone},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTransitionTableFor(machineApproval, tc.table)
			if err == nil {
				t.Fatal("坏表应被拒绝")
			}
			if !strings.Contains(err.Error(), string(machineApproval)) {
				t.Fatalf("错误信息应点名状态机: %v", err)
			}
		})
	}
}

func TestNextStateUsesKindTable(t *testing.T) {
	if next, ok := nextState(PendingApproval, StatePresented, eventAllowLongTerm); !ok || next != StateAwaitingLongTerm {
		t.Fatalf("审批长期允许迁移不对: %s %v", next, ok)
	}
	if _, ok := nextState(PendingQuestion, StatePresented, eventAllowLongTerm); ok {
		t.Fatal("提问不应接受长期允许迁移")
	}
	if _, ok := nextState(PendingApproval, StateAnswered, eventSettled); ok {
		t.Fatal("审批表没有已答状态")
	}
	if next, ok := nextState(PendingQuestion, StateAnswered, eventSettled); !ok || next != StateNone {
		t.Fatalf("提问结算迁移不对: %s %v", next, ok)
	}
}
