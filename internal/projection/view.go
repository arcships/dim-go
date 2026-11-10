// Package projection 是会话投影（D2）：把服务端事实折叠成一份内部一致的会话视图。
//
// 事实只有三个来源：历史页、实时通知、按固定截止点读到的未完成事项清单。本包不读
// 服务端存储、不把历史写盘、不做乐观更新，也不生成正式标识：已落库的行用服务端
// 标识，未落库的行只有包内临时键，且临时键只在本包内可见。
//
// 折叠只有一处实现：Folder.ApplyHistory 与 Folder.ApplyNotice 都落到
// branchState.apply，再落到同一个 foldState.apply。历史与实时的差别只在到达顺序，
// 顺序差由"按序号应用、乱序即整体重建"消除，因此同一批事实无论走哪条路径都收敛到
// 同一份视图。
package projection

import "encoding/json"

// RowKind 是行类型。行是视图的最小单元：一条消息可以拆成多个行，一行也可以只由
// 一条工具事实构成。
type RowKind string

const (
	// RowUserMessage 是用户的一次输入。
	RowUserMessage RowKind = "user_message"
	// RowAssistantText 是助手文本的一段连续输出。
	RowAssistantText RowKind = "assistant_text"
	// RowThinking 是助手思考的一段连续输出。
	RowThinking RowKind = "thinking"
	// RowTool 是一次工具调用与其结果。
	RowTool RowKind = "tool"
	// RowSystem 是服务端产生的系统提示，例如上下文压缩。
	RowSystem RowKind = "system"
	// RowGoalChange 是一次目标变更。
	RowGoalChange RowKind = "goal_change"
)

// RowStatus 是行状态。
type RowStatus string

const (
	// StatusPending 表示用户已提交、服务端尚未结算。
	StatusPending RowStatus = "pending"
	// StatusRunning 表示这一行还在增长。
	StatusRunning RowStatus = "running"
	// StatusComplete 表示这一行已结束。
	StatusComplete RowStatus = "complete"
	// StatusError 表示这一行以失败结束。
	StatusError RowStatus = "error"
	// StatusCancelled 表示这一行被取消。
	StatusCancelled RowStatus = "cancelled"
)

// Row 是视图里的一行。
//
// 身份规则：Pending 为真时这一行还没有落库，ID 为空；落库后原地换成服务端标识，
// 行在序列中的位置不变。客户端不生成正式标识。
type Row struct {
	ID        string      `json:"id,omitempty"`
	Kind      RowKind     `json:"kind"`
	Pending   bool        `json:"pending,omitempty"`
	BranchID  string      `json:"branchId"`
	Sequence  int64       `json:"sequence"`
	TurnIndex int         `json:"turnIndex"`
	TurnID    string      `json:"turnId,omitempty"`
	CreatedAt int64       `json:"createdAt"`
	UpdatedAt int64       `json:"updatedAt"`
	Status    RowStatus   `json:"status"`
	Text      string      `json:"text,omitempty"`
	Tool      *ToolCall   `json:"tool,omitempty"`
	Goal      *GoalChange `json:"goal,omitempty"`
}

// ToolCall 是一次工具调用的只读结果。
type ToolCall struct {
	CallID  string          `json:"callId"`
	Name    string          `json:"name"`
	Input   json.RawMessage `json:"input,omitempty"`
	Result  string          `json:"result,omitempty"`
	IsError bool            `json:"isError,omitempty"`
}

// GoalChange 是一次目标变更的事实摘要。
type GoalChange struct {
	// Action 取 set、pause、resume、completed、blocked、cleared 之一。
	Action string `json:"action"`
	// Status 是这次变更之后的目标状态。
	Status GoalStatus `json:"status"`
}

// TurnStatus 是轮次状态。
type TurnStatus string

const (
	// TurnRunning 表示这一轮还在进行。
	TurnRunning TurnStatus = "running"
	// TurnEnded 表示这一轮已经结束。
	TurnEnded TurnStatus = "ended"
)

// Turn 是轮索引里的一项。
type Turn struct {
	// Index 是分支内的轮序号，从 1 开始；0 表示尚未归属任何轮次。
	Index     int
	ID        string
	Status    TurnStatus
	Outcome   string
	Sequence  int64
	StartedAt int64
	EndedAt   int64
	// EndSequence 是结束事实（turn/end）的序号；0 表示这一轮没有结束事实。
	// 切点只认这个序号，没有它就明确说不可用，不拿别的序号凑。
	EndSequence int64
}

// GoalStatus 是目标状态。
type GoalStatus string

const (
	// GoalNone 表示目标存在但没有任何状态在推动它。
	GoalNone GoalStatus = "none"
	// GoalActive 表示目标在进行中。
	GoalActive GoalStatus = "active"
	// GoalPaused 表示目标被暂停。
	GoalPaused GoalStatus = "paused"
	// GoalComplete 表示目标已达成。
	GoalComplete GoalStatus = "complete"
	// GoalBlocked 表示目标被阻塞。
	GoalBlocked GoalStatus = "blocked"
)

// Goal 是折叠出来的目标。
type Goal struct {
	Objective string
	Status    GoalStatus
	Judge     string
	Reason    string
	Result    string
	Blocker   string
	Revision  int
	UpdatedAt int64
	// InputID 是最后一次写入目标的输入事件标识（服务端标识）。
	InputID string
}

// PendingItems 是四类未完成事项。装载按固定截止点进行，之后按事件结算。
type PendingItems struct {
	Inputs         []PendingInput
	Questions      []PendingQuestion
	Tasks          []PendingTask
	Collaborations []PendingCollaboration
}

// Total 返回四类未完成事项的总条数。
func (p PendingItems) Total() int {
	return len(p.Inputs) + len(p.Questions) + len(p.Tasks) + len(p.Collaborations)
}

// PendingInput 是一条已提交、尚未被服务端消费的输入。
type PendingInput struct {
	InputID  string
	Sequence int64
	Mode     string
	Text     string
	// Applied 为真表示已被应用但未消费：仍在清单里，但不可撤回。
	Applied bool
}

// PendingQuestion 是一批尚未回答的提问。提问内容原样保留，由交互裁决解释。
type PendingQuestion struct {
	RequestID string
	CallID    string
	EventID   string
	Sequence  int64
	ToolName  string
	Input     json.RawMessage
}

// PendingTask 是一个尚未结束的后台任务。
type PendingTask struct {
	TaskID      string `json:"taskId"`
	EventID     string `json:"eventId"`
	Sequence    int64  `json:"sequence"`
	Description string `json:"description,omitempty"`
	StartedAt   int64  `json:"startedAt"`
	// Stopping 表示已经发出停止、还在等结束事实。它只影响展示，不参与结算。
	Stopping bool `json:"stopping,omitempty"`
}

// PendingCollaboration 是一条尚未结算的协作请求。
type PendingCollaboration struct {
	EventID    string
	Sequence   int64
	CallID     string
	AgentID    string
	TargetKind string
	SessionID  string
	TurnID     string
	Prompt     string
	StartedAt  int64
	// Delivered 为真表示成员已接入，但请求仍未结算。
	Delivered bool
}

// BranchStatus 是分支状态。
type BranchStatus string

const (
	// BranchOpen 表示分支还在接收事实。
	BranchOpen BranchStatus = "open"
	// BranchEnded 表示分支已结束：之后的事实一律丢弃。
	BranchEnded BranchStatus = "ended"
)

// BranchInfo 是分支目录的一项。
type BranchInfo struct {
	BranchID    string
	ParentID    string
	BranchPoint int64
	Status      BranchStatus
	Head        int64
}

// LoadPhase 是"打开与装载"状态机的状态。
type LoadPhase string

const (
	PhaseClosed       LoadPhase = "closed"
	PhaseLoadingFirst LoadPhase = "loading_first_turn"
	PhaseReady        LoadPhase = "ready"
	PhasePaging       LoadPhase = "paging"
)

// LoadEvent 是"打开与装载"状态机的事件。
type LoadEvent string

const (
	LoadOpenRequested    LoadEvent = "open_requested"
	LoadFirstTurnArrived LoadEvent = "first_turn_arrived"
	LoadPageRequested    LoadEvent = "page_requested"
	LoadPageArrived      LoadEvent = "page_arrived"
	LoadConnectionLost   LoadEvent = "connection_lost"
	LoadSessionClosed    LoadEvent = "session_closed"
)

// Selection 是"分支选择"状态机的状态。
type Selection string

const (
	// SelectionMain 表示当前看主线。
	SelectionMain Selection = "main"
	// SelectionBranch 表示当前只看某个分支，且只读。
	SelectionBranch Selection = "branch_readonly"
)

// SelectionEvent 是"分支选择"状态机的事件。
type SelectionEvent string

const (
	SelectBranchEvent    SelectionEvent = "select_branch"
	BackToMainEvent      SelectionEvent = "back_to_main"
	EndedBranchFactEvent SelectionEvent = "fact_on_ended_branch"
)

// Transition 是一次状态机驱动结果。OK 为假表示该迁移非法：事件被丢弃、状态不变。
type Transition struct {
	OK      bool
	Machine string
	From    string
	To      string
	Event   string
	Reason  string
}

// Diagnostic 是一条诊断：非法事件或非法迁移的原因。怎么报由调用方决定。
type Diagnostic struct {
	Machine string
	From    string
	Event   string
	Reason  string
}

// Rejection 是一条被丢弃的事实及其原因。
type Rejection struct {
	BranchID string
	Sequence int64
	Type     string
	Name     string
	Reason   string
	Detail   string
}

// 丢弃原因。
const (
	ReasonInvalidEvent    = "invalid_event"
	ReasonBranchEnded     = "branch_ended"
	ReasonSessionMismatch = "session_mismatch"
	ReasonUnknownBranch   = "unknown_branch"
	ReasonMalformedNotice = "malformed_notice"
	ReasonOutOfWindow     = "out_of_window"
)

// View 是一份只读会话视图。它不含内部临时键，也不含可写引用。
type View struct {
	SessionID    string
	BranchID     string
	BranchStatus BranchStatus
	Head         int64
	Rows         []Row
	Turns        []Turn
	Pending      PendingItems
	Goal         *Goal
	LoadPhase    LoadPhase
	Selection    Selection
	// Unlanded 是尾部行的条数：这些行来自实时事实，还没有被历史覆盖。
	Unlanded int
	// Anchor 是最早已装载轮次的起始序号；0 表示还没装载过历史窗口。
	Anchor int64
	// HasOlder 表示还有更早的轮次没装载。
	HasOlder bool
	// Suggestion 是当前这一轮的一条提示建议；没有就是 nil。它不进时间线，
	// 断线即清空（服务端不重放）。
	Suggestion *Suggestion
	// Usage 是本分支累计用量（model/usage）；还没用量事实时为 nil。
	Usage *Usage
	// Context 是最近一次模型调用的上下文占用（context/budget）；没有事实时为 nil。
	Context *ContextUsage
	// Compaction 是最近一次上下文压缩的摘要；没有压缩事实时为 nil。
	Compaction *Compaction
	// Compactions 是压缩次数。
	Compactions int64
}
