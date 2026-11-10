package projection

import (
	"fmt"
	"strings"
)

// 状态机名，诊断与启动校验都按名字报出。
const (
	machineLoad      = "打开与装载"
	machineSelection = "分支选择"
)

// anyState 表示迁移表里的"任意当前状态"。
const anyState = "*"

// transitionRow 是迁移表的一行。From 为 anyState 时表示任意当前状态。
type transitionRow struct {
	From  string
	Event string
	To    string
}

// loadPhases 是"打开与装载"的全部状态。
var loadPhases = []LoadPhase{PhaseClosed, PhaseLoadingFirst, PhaseReady, PhasePaging}

// loadEvents 是"打开与装载"的全部事件。
var loadEvents = []LoadEvent{
	LoadOpenRequested,
	LoadFirstTurnArrived,
	LoadPageRequested,
	LoadPageArrived,
	LoadConnectionLost,
	LoadSessionClosed,
}

// loadTable 与 docs/domains/A.md 的"打开与装载"迁移表逐条对应：
//
//	未打开 + 打开请求 → 装载首轮
//	装载首轮 + 首轮历史到达 → 就绪
//	就绪 + 向前翻页请求 → 翻页中
//	翻页中 + 该页到达 → 就绪
//	就绪 + 连接中断 → 装载首轮（并清空未落库尾部）
//	任意 + 关闭会话 → 未打开
var loadTable = []transitionRow{
	{From: string(PhaseClosed), Event: string(LoadOpenRequested), To: string(PhaseLoadingFirst)},
	{From: string(PhaseLoadingFirst), Event: string(LoadFirstTurnArrived), To: string(PhaseReady)},
	{From: string(PhaseReady), Event: string(LoadPageRequested), To: string(PhasePaging)},
	{From: string(PhasePaging), Event: string(LoadPageArrived), To: string(PhaseReady)},
	{From: string(PhaseReady), Event: string(LoadConnectionLost), To: string(PhaseLoadingFirst)},
	{From: anyState, Event: string(LoadSessionClosed), To: string(PhaseClosed)},
}

// selections 是"分支选择"的全部状态。
var selections = []Selection{SelectionMain, SelectionBranch}

// selectionEvents 是"分支选择"的全部事件。
var selectionEvents = []SelectionEvent{SelectBranchEvent, BackToMainEvent, EndedBranchFactEvent}

// selectionTable 与 docs/domains/A.md 的"分支选择"迁移表逐条对应：
//
//	主线 + 选某个分支 → 分支只读
//	分支只读 + 切回主线 → 主线
//	分支只读 + 该分支已结束且收到新事件 → 分支只读（丢弃该事件）
var selectionTable = []transitionRow{
	{From: string(SelectionMain), Event: string(SelectBranchEvent), To: string(SelectionBranch)},
	{From: string(SelectionBranch), Event: string(BackToMainEvent), To: string(SelectionMain)},
	{From: string(SelectionBranch), Event: string(EndedBranchFactEvent), To: string(SelectionBranch)},
}

// validateTransitionTables 校验两张迁移表。启动时调用：任何一条不合法就失败，
// 报出具体状态机与不合法条目，不带病运行。
func validateTransitionTables() error {
	var problems []string
	problems = append(problems, validateTable(machineLoad, loadStates(), loadEventNames(), loadTable)...)
	problems = append(problems, validateTable(machineSelection, selectionStates(), selectionEventNames(), selectionTable)...)
	if len(problems) > 0 {
		return fmt.Errorf("状态机迁移表不合法: %s", strings.Join(problems, "; "))
	}
	return nil
}

func loadStates() []string {
	out := make([]string, 0, len(loadPhases))
	for _, phase := range loadPhases {
		out = append(out, string(phase))
	}
	return out
}

func loadEventNames() []string {
	out := make([]string, 0, len(loadEvents))
	for _, event := range loadEvents {
		out = append(out, string(event))
	}
	return out
}

func selectionStates() []string {
	out := make([]string, 0, len(selections))
	for _, selection := range selections {
		out = append(out, string(selection))
	}
	return out
}

func selectionEventNames() []string {
	out := make([]string, 0, len(selectionEvents))
	for _, event := range selectionEvents {
		out = append(out, string(event))
	}
	return out
}

// validateTable 检查一张表的每条迁移：当前状态与目标状态必须落在状态集合内，
// 事件必须在可接受事件内；同一（当前状态，事件）不允许两条不同目标。
func validateTable(machine string, states, events []string, table []transitionRow) []string {
	known := make(map[string]struct{}, len(states))
	for _, state := range states {
		known[state] = struct{}{}
	}
	accepts := make(map[string]struct{}, len(events))
	for _, event := range events {
		accepts[event] = struct{}{}
	}
	seen := make(map[string]string, len(table))
	var problems []string
	for index, row := range table {
		if row.From != anyState {
			if _, ok := known[row.From]; !ok {
				problems = append(problems, fmt.Sprintf("%s 第 %d 条: 当前状态 %q 不在状态集合内", machine, index+1, row.From))
			}
		}
		if _, ok := known[row.To]; !ok {
			problems = append(problems, fmt.Sprintf("%s 第 %d 条: 目标状态 %q 不在状态集合内", machine, index+1, row.To))
		}
		if _, ok := accepts[row.Event]; !ok {
			problems = append(problems, fmt.Sprintf("%s 第 %d 条: 事件 %q 不在可接受事件内", machine, index+1, row.Event))
		}
		key := row.From + "\x00" + row.Event
		if previous, ok := seen[key]; ok && previous != row.To {
			problems = append(problems, fmt.Sprintf("%s 第 %d 条: (%q, %q) 已有目标 %q，又声明 %q", machine, index+1, row.From, row.Event, previous, row.To))
		}
		seen[key] = row.To
	}
	return problems
}

// lookupTransition 在表里找一条迁移。任意状态的条目优先，避免被同名的具体条目遮住。
func lookupTransition(table []transitionRow, from, event string) (string, bool) {
	if to, ok := findRow(table, from, event); ok {
		return to, true
	}
	return findRow(table, anyState, event)
}

func findRow(table []transitionRow, from, event string) (string, bool) {
	for _, row := range table {
		if row.From == from && row.Event == event {
			return row.To, true
		}
	}
	return "", false
}
