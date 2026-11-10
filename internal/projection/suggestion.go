package projection

import (
	"encoding/json"
)

// 提示建议（PromptSuggestion）：服务端在某轮结束后给一条"用户接下来可能会说的话"。
// 它不进时间线，没有序号，也不重放；本包只折叠服务端的 offer / cleared 通知，
// 供界面读取。
//
// 请求与应答由集成层发：`session.suggestion.request {sessionId, turnId}` 触发一次
// 建议，`session.suggestion.respond {sessionId, turnId, action: accept|dismiss}`
// 表示用户接受或忽略。应答是一次写入，归写登记；本包不登记、不发送。

// suggestionNoticeSource 是提示建议通知的来源名。
const suggestionNoticeSource = "PromptSuggestion"

// Suggestion 是当前这一轮的一条提示建议。
type Suggestion struct {
	// TurnID 是这条建议所属的轮次。
	TurnID string `json:"turnId,omitempty"`
	// Text 是建议内容；服务端按 maxChars 截断过。
	Text string `json:"text"`
}

// sessionNotice 是 `session.notice` 通知的参数。它是服务端会话通知的公共信封，
// 本包只处理其中提示建议这一类。
type sessionNotice struct {
	SessionID string          `json:"sessionId"`
	BranchID  string          `json:"branchId,omitempty"`
	Source    string          `json:"source"`
	Kind      string          `json:"kind"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// applySessionNoticeLocked 折叠一条会话通知。只认提示建议；其他来源的通知属于别的
// 领域，本包不看也不改。
func (f *Folder) applySessionNoticeLocked(notice sessionNotice) []Rejection {
	if notice.SessionID != "" && notice.SessionID != f.sessionID {
		return []Rejection{{
			BranchID: notice.BranchID,
			Reason:   ReasonSessionMismatch,
			Detail:   "通知属于别的会话",
		}}
	}
	if notice.Source != suggestionNoticeSource {
		return nil
	}
	branch := f.branchLocked(notice.BranchID)
	data := decodeObject(notice.Data)
	turnID := firstString(data, "turnId", "turn_id")
	switch notice.Kind {
	case "offer":
		branch.suggestion = &Suggestion{TurnID: turnID, Text: stringValue(data["text"])}
	case "cleared":
		// 只清同一轮的建议；没带轮次标识时一律清掉。
		if branch.suggestion != nil && (turnID == "" || branch.suggestion.TurnID == turnID) {
			branch.suggestion = nil
		}
	}
	return nil
}

// TaskRunning 报告一个后台任务是否还在未结束清单里。任务输出用它判断"空页是不是
// EOF"：任务还在跑时，空页只表示暂时没有新输出。
func (f *Folder) TaskRunning(branchID, taskID string) bool {
	if branchID == "" {
		branchID = mainBranch
	}
	if taskID == "" {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	branch, ok := f.branches[branchID]
	if !ok {
		return false
	}
	return branch.foldState().inv.hasTask(taskID)
}

// MarkTaskStopping 把"已经发出停止"标到未结束任务上。它只影响展示：任务仍由结束
// 事实结算，本方法不改清单的成员。
func (f *Folder) MarkTaskStopping(branchID, taskID string) {
	if branchID == "" {
		branchID = mainBranch
	}
	if taskID == "" {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	branch, ok := f.branches[branchID]
	if !ok {
		return
	}
	branch.foldState().inv.markTaskStopping(taskID)
}
