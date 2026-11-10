package write

// verdict 是一次核对读取的判定。
type verdict int

const (
	// verifyUnconfirmed 表示核对无法确认，需要继续核对。
	verifyUnconfirmed verdict = iota
	// verifyHit 表示历史已含该输入且归属一致。
	verifyHit
	// verifyMiss 表示读到了确定状态，且服务端没有这次写入。
	verifyMiss
)

// judge 把一次核对读取收敛成三选一：
//   - 读取失败或读取不可信 → 无法确认；
//   - 历史已含该输入且归属一致 → 命中；
//   - 读到确定状态且历史不含该输入 → 未发生。
//
// Running 是核对上下文，本版判定不使用它：运行状态不能单独证明某次写是否发生。
func judge(query InputQuery, state InputState, err error) verdict {
	if err != nil || !state.Confirmed {
		return verifyUnconfirmed
	}
	if !state.Present {
		return verifyMiss
	}
	if !ownershipMatches(query, state) {
		return verifyUnconfirmed
	}
	return verifyHit
}

// ownershipMatches 判断核对命中的输入是否属于这次意图的目标会话与分支。
// 意图声明的归属必须与读取结果完全一致；读取结果为空视为无法确认归属。
// 会话生命周期的新建意图在登记时没有会话标识，核对时允许读到服务端新建的
// 会话标识；其他无会话意图维持原有判定，读到会话即视为无法确认归属。
func ownershipMatches(query InputQuery, state InputState) bool {
	if query.SessionID != "" {
		if state.SessionID != query.SessionID {
			return false
		}
	} else if state.SessionID != "" && query.Kind != KindSessionLifecycle {
		return false
	}
	if query.BranchID != "" && state.BranchID != query.BranchID {
		return false
	}
	return true
}
