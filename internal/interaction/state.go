package interaction

import "fmt"

// State 是一条待决项在推进状态机里的状态。名字逐条对应 docs/domains/A.md 的
// 「审批推进」与「提问应答」两张表。
type State string

const (
	// StateNone 表示无待决：条目不存在。
	StateNone State = "none"
	// StatePresented 表示待呈现（审批）或待答（提问）：条目在队首等用户决定。
	StatePresented State = "presented"
	// StateAwaitingLongTerm 表示待写长期：用户选了长期允许，规则正在写入。
	StateAwaitingLongTerm State = "awaiting_long_term"
	// StateAwaitingResponse 表示待应答：决定已作出，应答已发出或长期规则已写入。
	StateAwaitingResponse State = "awaiting_response"
	// StateAnswered 表示提问已答，等待服务端结算。
	StateAnswered State = "answered"
)

// stateEvent 驱动推进状态机的事件。它不出包。
type stateEvent string

const (
	// eventIncoming 是「审批事件」或「提问清单装载 / 截止点之后的 ask_user 事实」。
	eventIncoming stateEvent = "incoming"
	// eventDecide 是用户选出同意、拒绝、应答或取消。
	eventDecide stateEvent = "decide"
	// eventAllowLongTerm 是用户选长期允许。
	eventAllowLongTerm stateEvent = "allow_long_term"
	// eventLongTermWritten 是长期允许写入成功。
	eventLongTermWritten stateEvent = "long_term_written"
	// eventLongTermFailed 是长期允许写入失败或无法确认。
	eventLongTermFailed stateEvent = "long_term_failed"
	// eventSettled 是服务端结算：审批回执/取消、提问的工具结果。
	eventSettled stateEvent = "settled"
)

// machine 是两张推进状态机的名字，只用于校验与错误信息。
type machine string

const (
	machineApproval machine = "审批推进"
	machineQuestion machine = "提问应答"
)

// approvalTransitions 逐条对应 A.md「审批推进」，并补一条服务端在用户作答前
// 结算（审批被取消）的出边：
//
//	无待决 + 审批事件或重放快照 → 待呈现
//	待呈现 + 用户选同意或拒绝 → 待应答
//	待呈现 + 用户选长期允许 → 待写长期
//	待写长期 + 长期允许写入成功 → 待应答
//	待写长期 + 长期允许写入失败 → 待呈现（且不发应答）
//	待应答 + 应答发出且服务端结算 → 无待决（推进到下一条）
//	待呈现/待写长期 + 服务端结算 → 无待决（审批在作答前被取消）
var approvalTransitions = map[State]map[stateEvent]State{
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

// questionTransitions 逐条对应 A.md「提问应答」，并补一条服务端在用户作答前
// 结算（提问被取消）的出边：
//
//	无待决 + 提问事件或重放快照 → 待答（只取队首）
//	待答 + 用户应答、取消或拒绝 → 已答
//	已答 + 服务端结算 → 无待决（推进到下一条）
//	待答 + 服务端结算 → 无待决（提问在作答前被取消）
var questionTransitions = map[State]map[stateEvent]State{
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

// allStates 是两张表允许出现的状态全集。
var allStates = []State{
	StateNone,
	StatePresented,
	StateAwaitingLongTerm,
	StateAwaitingResponse,
	StateAnswered,
}

// allStateEvents 是两张表允许出现的事件全集。
var allStateEvents = []stateEvent{
	eventIncoming,
	eventDecide,
	eventAllowLongTerm,
	eventLongTermWritten,
	eventLongTermFailed,
	eventSettled,
}

// transitionTable 按待决类型取迁移表。
func transitionTable(kind PendingKind) (machine, map[State]map[stateEvent]State) {
	if kind == PendingQuestion {
		return machineQuestion, questionTransitions
	}
	return machineApproval, approvalTransitions
}

// nextState 查一步迁移。非法迁移返回 false，由调用方丢弃事件。
func nextState(kind PendingKind, from State, event stateEvent) (State, bool) {
	_, table := transitionTable(kind)
	events, ok := table[from]
	if !ok {
		return "", false
	}
	next, ok := events[event]
	return next, ok
}

// validateTransitionTables 在启动时校验两张推进表：来源与目标都在状态集合内、
// 事件都在事件集合内、无待决只作为入口、每个非无待决状态都有服务端结算出边。
func validateTransitionTables() error {
	if err := validateTransitionTableFor(machineApproval, approvalTransitions); err != nil {
		return err
	}
	return validateTransitionTableFor(machineQuestion, questionTransitions)
}

// validateTransitionTableFor 校验一份迁移表；抽出来便于测试坏表。
func validateTransitionTableFor(name machine, table map[State]map[stateEvent]State) error {
	states := make(map[State]bool, len(allStates))
	for _, state := range allStates {
		states[state] = true
	}
	events := make(map[stateEvent]bool, len(allStateEvents))
	for _, event := range allStateEvents {
		events[event] = true
	}

	for _, from := range allStates {
		edges, listed := table[from]
		if !listed {
			continue
		}
		for event, to := range edges {
			if !events[event] {
				return fmt.Errorf("%s状态迁移表不合法：%q 上的未知事件 %q", name, from, event)
			}
			if !states[to] {
				return fmt.Errorf("%s状态迁移表不合法：%q 经 %q 到未知状态 %q", name, from, event, to)
			}
		}
	}

	incoming, ok := table[StateNone]
	if !ok {
		return fmt.Errorf("%s状态迁移表不合法：缺少「无待决」入口", name)
	}
	if _, ok := incoming[eventIncoming]; !ok {
		return fmt.Errorf("%s状态迁移表不合法：「无待决」缺少进入待决的迁移", name)
	}
	for event := range incoming {
		if event != eventIncoming {
			return fmt.Errorf("%s状态迁移表不合法：「无待决」只能作为入口，出现了事件 %q", name, event)
		}
	}

	// 只有本状态机可达的状态需要结算出边；另一张表的状态不算。
	reachable := map[State]bool{StateNone: true}
	queue := []State{StateNone}
	for len(queue) > 0 {
		from := queue[0]
		queue = queue[1:]
		for _, to := range table[from] {
			if reachable[to] {
				continue
			}
			reachable[to] = true
			queue = append(queue, to)
		}
	}
	for state := range reachable {
		if state == StateNone {
			continue
		}
		if _, ok := table[state][eventSettled]; !ok {
			return fmt.Errorf("%s状态迁移表不合法：%q 缺少服务端结算出边", name, state)
		}
	}
	return nil
}
