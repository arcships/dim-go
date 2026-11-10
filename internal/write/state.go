package write

import (
	"fmt"
	"sort"
)

// transitionEvent 是驱动「在途写」状态机的事件。它不出包。
type transitionEvent string

const (
	eventSend              transitionEvent = "send"
	eventSucceed           transitionEvent = "succeed"
	eventFail              transitionEvent = "fail"
	eventUnknown           transitionEvent = "unknown"
	eventVerifyHit         transitionEvent = "verify_hit"
	eventVerifyMiss        transitionEvent = "verify_miss"
	eventVerifyUnconfirmed transitionEvent = "verify_unconfirmed"
	eventVerifyExhausted   transitionEvent = "verify_exhausted"
	eventClose             transitionEvent = "close"
)

// transitionTable 逐条对应 docs/domains/A.md 的「在途写」迁移表。
//
// 已登记 → 在途（发出请求）
// 在途 → 成功（明确成功）
// 在途 → 失败（明确失败）
// 在途 → 未知（超时、断线或重启后发现已发出）
// 未知 → 成功（核对命中且归属一致）
// 未知 → 未发生（核对确认服务端没有这次写入）
// 未知 → 未知（核对无法确认，按上限再核对）
// 未知 → 需人工确认（核对次数到上限）
//
// 另外两条是本领域的关闭处理：登记后未发出就关闭，按未发生处理；在途或未知
// 时关闭，按「需人工确认」处理，与 A.md「重启后一律按需人工确认、不补发」一致。
var transitionTable = map[Status]map[transitionEvent]Status{
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

var allStatuses = []Status{
	StatusRegistered,
	StatusInFlight,
	StatusUnknown,
	StatusSucceeded,
	StatusFailed,
	StatusNotOccurred,
	StatusNeedsManualConfirmation,
}

var allEvents = []transitionEvent{
	eventSend,
	eventSucceed,
	eventFail,
	eventUnknown,
	eventVerifyHit,
	eventVerifyMiss,
	eventVerifyUnconfirmed,
	eventVerifyExhausted,
	eventClose,
}

// transition 查一步迁移。非法迁移返回 false，由调用方丢弃事件。
func transition(from Status, event transitionEvent) (Status, bool) {
	events, ok := transitionTable[from]
	if !ok {
		return "", false
	}
	next, ok := events[event]
	return next, ok
}

// validateTransitionTable 在启动时校验迁移表：每个来源与目标都在状态集合内，
// 每个事件都在事件集合内，终态没有出边，非终态都有「关闭」出边。
func validateTransitionTable() error {
	return validateTransitionTableFor(transitionTable)
}

// validateTransitionTableFor 校验一份迁移表；抽出来便于测试坏表。
func validateTransitionTableFor(table map[Status]map[transitionEvent]Status) error {
	statuses := make(map[Status]bool, len(allStatuses))
	for _, status := range allStatuses {
		statuses[status] = true
	}
	events := make(map[transitionEvent]bool, len(allEvents))
	for _, event := range allEvents {
		events[event] = true
	}

	sources := make([]string, 0, len(table))
	for status := range table {
		sources = append(sources, string(status))
	}
	sort.Strings(sources)

	for _, raw := range sources {
		from := Status(raw)
		if !statuses[from] {
			return fmt.Errorf("写登记状态迁移表不合法：未知来源状态 %q", from)
		}
		if from.terminal() {
			return fmt.Errorf("写登记状态迁移表不合法：终态 %q 不应有出边", from)
		}
		edges := table[from]
		for event, to := range edges {
			if !events[event] {
				return fmt.Errorf("写登记状态迁移表不合法：%q 上的未知事件 %q", from, event)
			}
			if !statuses[to] {
				return fmt.Errorf("写登记状态迁移表不合法：%q 经 %q 到未知状态 %q", from, event, to)
			}
		}
	}

	for _, status := range allStatuses {
		if status.terminal() {
			continue
		}
		if _, ok := table[status][eventClose]; !ok {
			return fmt.Errorf("写登记状态迁移表不合法：非终态 %q 缺少关闭迁移", status)
		}
	}
	return nil
}
