package projection

import (
	"strings"
	"testing"
)

// TestLoadPhaseMachineFollowsTable 验证"打开与装载"按 A 文档的迁移表走。
func TestLoadPhaseMachineFollowsTable(t *testing.T) {
	folder := newFolder(t)
	if folder.LoadPhase() != PhaseClosed {
		t.Fatalf("初始应未打开: %s", folder.LoadPhase())
	}

	steps := []struct {
		event LoadEvent
		want  LoadPhase
	}{
		{LoadOpenRequested, PhaseLoadingFirst},
		{LoadFirstTurnArrived, PhaseReady},
		{LoadPageRequested, PhasePaging},
		{LoadPageArrived, PhaseReady},
		{LoadConnectionLost, PhaseLoadingFirst},
		{LoadFirstTurnArrived, PhaseReady},
		{LoadSessionClosed, PhaseClosed},
	}
	for index, step := range steps {
		transition := folder.AdvanceLoad(step.event)
		if !transition.OK {
			t.Fatalf("第 %d 步迁移被拒: %+v", index+1, transition)
		}
		if folder.LoadPhase() != step.want {
			t.Fatalf("第 %d 步后阶段应为 %s，实际 %s", index+1, step.want, folder.LoadPhase())
		}
	}
	if diagnostics := folder.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("合法路径不该记诊断: %+v", diagnostics)
	}
}

// TestIllegalLoadTransitionIsDropped 验证非法迁移被丢弃、状态不变，并记下原因。
func TestIllegalLoadTransitionIsDropped(t *testing.T) {
	folder := newFolder(t)
	// 未打开时直接收到首轮历史：迁移表里没有这条。
	transition := folder.AdvanceLoad(LoadFirstTurnArrived)
	if transition.OK {
		t.Fatalf("非法迁移不该通过: %+v", transition)
	}
	if transition.Machine != machineLoad {
		t.Fatalf("诊断应带状态机名: %+v", transition)
	}
	if folder.LoadPhase() != PhaseClosed {
		t.Fatalf("非法迁移不该改状态: %s", folder.LoadPhase())
	}
	diagnostics := folder.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Machine != machineLoad {
		t.Fatalf("应记一条装载诊断: %+v", diagnostics)
	}
}

// TestCloseFromAnyState 验证"任意 → 关闭会话"从每个状态都成立。
func TestCloseFromAnyState(t *testing.T) {
	for _, phase := range loadPhases {
		folder := newFolder(t)
		folder.loadPhase = phase
		transition := folder.AdvanceLoad(LoadSessionClosed)
		if !transition.OK || folder.LoadPhase() != PhaseClosed {
			t.Fatalf("从 %s 关闭会话应成功: %+v", phase, transition)
		}
	}
}

// TestTransitionTablesMatchDocument 验证迁移表与 A 文档逐条对得上。
func TestTransitionTablesMatchDocument(t *testing.T) {
	if err := validateTransitionTables(); err != nil {
		t.Fatalf("迁移表校验失败: %v", err)
	}
	if len(loadTable) != 6 {
		t.Fatalf("装载迁移应为 6 条，实际 %d", len(loadTable))
	}
	if len(selectionTable) != 3 {
		t.Fatalf("分支选择迁移应为 3 条，实际 %d", len(selectionTable))
	}

	wanted := []transitionRow{
		{From: string(PhaseClosed), Event: string(LoadOpenRequested), To: string(PhaseLoadingFirst)},
		{From: string(PhaseLoadingFirst), Event: string(LoadFirstTurnArrived), To: string(PhaseReady)},
		{From: string(PhaseReady), Event: string(LoadPageRequested), To: string(PhasePaging)},
		{From: string(PhasePaging), Event: string(LoadPageArrived), To: string(PhaseReady)},
		{From: string(PhaseReady), Event: string(LoadConnectionLost), To: string(PhaseLoadingFirst)},
		{From: anyState, Event: string(LoadSessionClosed), To: string(PhaseClosed)},
	}
	for _, row := range wanted {
		if to, ok := lookupTransition(loadTable, row.From, row.Event); !ok || to != row.To {
			t.Fatalf("装载迁移缺条目或目标不符: %+v", row)
		}
	}
}

// TestBrokenTableIsRejected 验证不合格的迁移表会被逐条报出。
func TestBrokenTableIsRejected(t *testing.T) {
	broken := []transitionRow{
		{From: "不存在", Event: string(LoadOpenRequested), To: string(PhaseLoadingFirst)},
		{From: string(PhaseClosed), Event: "不存在的事件", To: string(PhaseReady)},
		{From: string(PhaseClosed), Event: string(LoadOpenRequested), To: "不存在的目标"},
		{From: string(PhaseReady), Event: string(LoadPageRequested), To: string(PhasePaging)},
		{From: string(PhaseReady), Event: string(LoadPageRequested), To: string(PhaseClosed)},
	}
	problems := validateTable(machineLoad, loadStates(), loadEventNames(), broken)
	if len(problems) != 4 {
		t.Fatalf("应报出 4 条问题，实际 %d: %v", len(problems), problems)
	}
	for _, problem := range problems {
		if !strings.HasPrefix(problem, machineLoad) {
			t.Fatalf("问题应带状态机名: %q", problem)
		}
	}

	if err := validateTransitionTables(); err != nil {
		t.Fatalf("真实迁移表应通过: %v", err)
	}
	// 校验失败时启动即失败。
	saved := loadTable
	loadTable = broken
	defer func() { loadTable = saved }()
	if err := validateTransitionTables(); err == nil {
		t.Fatalf("不合格迁移表应让启动失败")
	} else if !strings.Contains(err.Error(), machineLoad) {
		t.Fatalf("启动失败信息应点名状态机: %v", err)
	}
}

// TestNewRequiresSessionID 验证缺会话标识时启动失败。
func TestNewRequiresSessionID(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatalf("缺会话标识应失败")
	}
}
