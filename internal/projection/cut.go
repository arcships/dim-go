package projection

// 切点查询：目录领域从父分支的某一轮结束处开分支，需要一个序号。
//
// 本查询只读现有视图里的轮次与它们的结束事实，不折叠、不读取、不猜：轮次不在
// 已加载范围内就是"未加载"，轮次没有结束事实就是"不可用"。

// CutStatus 是切点查询的结果状态。
type CutStatus string

const (
	// CutReady 表示该轮已结束，切点可用。
	CutReady CutStatus = "ready"
	// CutNotEnded 表示该轮没有结束事实（还在进行或没收到 turn/end），切点不可用。
	CutNotEnded CutStatus = "not_ended"
	// CutNotLoaded 表示该轮不在已加载范围内，调用方应先向前翻页而不是猜切点。
	CutNotLoaded CutStatus = "not_loaded"
	// CutNoBranch 表示分支不在折叠里（未知分支或不是本会话）。
	CutNoBranch CutStatus = "no_branch"
)

// CutPoint 是一次切点查询的结果。
type CutPoint struct {
	// Status 说明能不能用这个切点，以及为什么不能用。
	Status CutStatus
	// Sequence 是该轮结束事实的序号；只有 Status 为 CutReady 时有意义。
	Sequence int64
	// TurnIndex 是查询的轮次序号。
	TurnIndex int
	// TurnID 是该轮的服务端标识（已知时给出）。
	TurnID string
}

// Usable 报告切点是否可用。
func (c CutPoint) Usable() bool { return c.Status == CutReady }

// CutPoint 查询某一轮结束处的序号。它只看已加载的事实，不触发任何读取。
func (f *Folder) CutPoint(branchID string, turnIndex int) CutPoint {
	if branchID == "" {
		branchID = mainBranch
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	branch, ok := f.branches[branchID]
	if !ok {
		return CutPoint{Status: CutNoBranch, TurnIndex: turnIndex}
	}
	view := f.viewLocked(branch)
	for _, turn := range view.Turns {
		if turn.Index != turnIndex {
			continue
		}
		if turn.EndSequence <= 0 {
			// 没有结束事实：不拿行序号或本轮起点凑一个切点。
			return CutPoint{Status: CutNotEnded, TurnIndex: turnIndex, TurnID: turn.ID}
		}
		return CutPoint{
			Status:    CutReady,
			Sequence:  turn.EndSequence,
			TurnIndex: turnIndex,
			TurnID:    turn.ID,
		}
	}
	return CutPoint{Status: CutNotLoaded, TurnIndex: turnIndex}
}

// TurnCut 返回某一轮结束处的父分支序号，供目录领域开分支直接使用。
//
// 签名与目录领域的分支来源接口一致（不导入该包）：ok 为 false 表示拿不到切点，
// 可能是轮次还没结束，也可能是不在已加载范围内；需要区分原因时用 CutPoint。
// 会话标识与本折叠器不一致时同样返回不可用。
func (f *Folder) TurnCut(sessionID, branchID string, turnIndex int) (int64, bool) {
	if sessionID != "" && sessionID != f.sessionID {
		return 0, false
	}
	point := f.CutPoint(branchID, turnIndex)
	if !point.Usable() {
		return 0, false
	}
	return point.Sequence, true
}
