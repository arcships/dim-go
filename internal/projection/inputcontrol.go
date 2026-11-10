package projection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

// 排队输入的撤回与会话目标的暂停/恢复：两者都是"先探测方法名，再发一次写入"。
// 本包只给意图与发送接口，登记与发送归写登记与集成层；状态仍由事实结算——
// 撤回写成 `input/withdraw`（可能再写 `input/deleted`），目标控制写成一条
// `goal:state` 记录。

// 用到的会话方法名。探测来源是 `session.methods`。
const (
	inputWithdrawMethod = "session.input.withdraw"
	goalPauseMethod     = "session.goal.pause"
	goalResumeMethod    = "session.goal.resume"
)

// 撤回与会话控制的错误值。
var (
	// ErrInputWithdrawUnavailable 表示服务端没有 `session.input.withdraw`。
	ErrInputWithdrawUnavailable = errors.New("撤回排队输入：服务端未提供 session.input.withdraw")
	// ErrInputWithdrawNoSender 表示没有配置撤回发送方。
	ErrInputWithdrawNoSender = errors.New("撤回排队输入：缺少发送方")
	// ErrInputWithdrawNoInput 表示没有输入标识。
	ErrInputWithdrawNoInput = errors.New("撤回排队输入：缺少输入标识")
	// ErrGoalControlUnavailable 表示服务端没有对应的目标控制方法。
	ErrGoalControlUnavailable = errors.New("目标控制：服务端未提供该方法")
	// ErrGoalControlNoSender 表示没有配置目标控制发送方。
	ErrGoalControlNoSender = errors.New("目标控制：缺少发送方")
	// ErrGoalControlBadAction 表示动作不是 pause 或 resume。
	ErrGoalControlBadAction = errors.New("目标控制：动作只能是 pause 或 resume")
)

// hasSessionMethod 报告会话方法表里有没有某个方法。方法表来自 `session.methods`；
// 没有探测过（空表）时一律按"没有"处理：宁可不发，也不猜。
func hasSessionMethod(methods []string, name string) bool {
	for _, method := range methods {
		if method == name {
			return true
		}
	}
	return false
}

// ----- 排队输入撤回 -----

// InputWithdrawRequest 是一次排队输入撤回请求。
type InputWithdrawRequest struct {
	SessionID string
	BranchID  string
	InputID   string
}

// InputWithdrawResult 是服务端的撤回回执：`deleted`（还握着，已删）或 `notHeld`
// （已经被消费或已删除，什么也没写）。
type InputWithdrawResult struct {
	Status string
}

// Deleted 报告这次撤回是否真的删掉了还没消费的输入。
func (r InputWithdrawResult) Deleted() bool { return r.Status == "deleted" }

// InputWithdrawSender 发送一次已经登记的撤回。集成层用写登记的发送方实现。
type InputWithdrawSender interface {
	WithdrawInput(ctx context.Context, req InputWithdrawRequest) (InputWithdrawResult, error)
}

// InputWithdrawIntent 是登记到写登记的撤回意图。
type InputWithdrawIntent struct {
	SessionID string
	BranchID  string
	InputID   string
}

// Op 是写登记参数里的操作名。
func (i InputWithdrawIntent) Op() string { return "input.withdraw" }

// Key 是幂等键：同一条输入重复撤回只登记一次。
func (i InputWithdrawIntent) Key() string {
	sum := sha256.Sum256([]byte(i.SessionID + "\x00" + i.BranchID + "\x00" + i.InputID))
	return "projection.input_withdraw:" + hex.EncodeToString(sum[:16])
}

// Params 是写登记参数。
func (i InputWithdrawIntent) Params() (json.RawMessage, error) {
	return json.Marshal(struct {
		Op       string `json:"op"`
		BranchID string `json:"branchId"`
		InputID  string `json:"inputId"`
	}{Op: "input.withdraw", BranchID: i.BranchID, InputID: i.InputID})
}

// QueuedInputWithdraw 撤回一条还没被消费的排队输入。方法缺失时不发请求。
type QueuedInputWithdraw struct {
	folder *Folder
	intent InputWithdrawIntent
	sender InputWithdrawSender
}

// NewQueuedInputWithdraw 创建一次撤回。methods 是 `session.methods` 的只读快照；
// 里面没有 `session.input.withdraw` 就返回能力缺失，调用方据此零请求。
func NewQueuedInputWithdraw(
	folder *Folder,
	branchID, inputID string,
	methods []string,
	sender InputWithdrawSender,
) (*QueuedInputWithdraw, error) {
	if !hasSessionMethod(methods, inputWithdrawMethod) {
		return nil, ErrInputWithdrawUnavailable
	}
	if sender == nil {
		return nil, ErrInputWithdrawNoSender
	}
	if inputID == "" {
		return nil, ErrInputWithdrawNoInput
	}
	if branchID == "" {
		branchID = mainBranch
	}
	withdraw := &QueuedInputWithdraw{
		folder: folder,
		intent: InputWithdrawIntent{BranchID: branchID, InputID: inputID},
		sender: sender,
	}
	if folder != nil {
		withdraw.intent.SessionID = folder.sessionID
	}
	return withdraw, nil
}

// Intent 返回要登记到写登记的撤回意图。
func (w *QueuedInputWithdraw) Intent() InputWithdrawIntent { return w.intent }

// Withdraw 发送撤回。清单与视图的成员由 `input/withdraw` / `input/deleted`
// 事实结算，本方法不做本地预删。
func (w *QueuedInputWithdraw) Withdraw(ctx context.Context) (InputWithdrawResult, error) {
	return w.sender.WithdrawInput(ctx, InputWithdrawRequest{
		SessionID: w.intent.SessionID,
		BranchID:  w.intent.BranchID,
		InputID:   w.intent.InputID,
	})
}

// ----- 目标暂停与恢复 -----

// GoalAction 是目标控制动作。
type GoalAction string

const (
	// GoalPause 暂停目标。
	GoalPause GoalAction = "pause"
	// GoalResume 恢复目标。
	GoalResume GoalAction = "resume"
)

// goalMethod 返回动作对应的会话方法名。
func (a GoalAction) method() (string, bool) {
	switch a {
	case GoalPause:
		return goalPauseMethod, true
	case GoalResume:
		return goalResumeMethod, true
	}
	return "", false
}

// GoalControlRequest 是一次目标控制请求。
type GoalControlRequest struct {
	SessionID string
	BranchID  string
	Action    GoalAction
}

// GoalControlResult 是服务端的回执：控制事实的序号。
type GoalControlResult struct {
	Sequence int64
}

// GoalControlSender 发送一次已经登记的目标控制。集成层用写登记的发送方实现。
type GoalControlSender interface {
	ControlGoal(ctx context.Context, req GoalControlRequest) (GoalControlResult, error)
}

// GoalControlIntent 是登记到写登记的目标控制意图。
type GoalControlIntent struct {
	SessionID string
	BranchID  string
	Action    GoalAction
}

// Op 是写登记参数里的操作名（goal.pause / goal.resume）。
func (i GoalControlIntent) Op() string { return "goal." + string(i.Action) }

// Key 是幂等键：同一分支的同一动作重复点击只登记一次。
func (i GoalControlIntent) Key() string {
	sum := sha256.Sum256([]byte(i.SessionID + "\x00" + i.BranchID + "\x00" + string(i.Action)))
	return "projection.goal_control:" + hex.EncodeToString(sum[:16])
}

// Params 是写登记参数。
func (i GoalControlIntent) Params() (json.RawMessage, error) {
	return json.Marshal(struct {
		Op       string `json:"op"`
		BranchID string `json:"branchId"`
	}{Op: i.Op(), BranchID: i.BranchID})
}

// GoalControl 暂停或恢复一个分支的目标。方法缺失时不发请求。
type GoalControl struct {
	folder *Folder
	intent GoalControlIntent
	sender GoalControlSender
}

// NewGoalControl 创建一次目标控制。methods 是 `session.methods` 的只读快照；
// 里面没有对应方法就返回能力缺失。
func NewGoalControl(
	folder *Folder,
	branchID string,
	action GoalAction,
	methods []string,
	sender GoalControlSender,
) (*GoalControl, error) {
	method, ok := action.method()
	if !ok {
		return nil, ErrGoalControlBadAction
	}
	if !hasSessionMethod(methods, method) {
		return nil, ErrGoalControlUnavailable
	}
	if sender == nil {
		return nil, ErrGoalControlNoSender
	}
	if branchID == "" {
		branchID = mainBranch
	}
	control := &GoalControl{
		folder: folder,
		intent: GoalControlIntent{BranchID: branchID, Action: action},
		sender: sender,
	}
	if folder != nil {
		control.intent.SessionID = folder.sessionID
	}
	return control, nil
}

// Intent 返回要登记到写登记的控制意图。
func (g *GoalControl) Intent() GoalControlIntent { return g.intent }

// Apply 发送控制。目标状态由服务端写下的 `goal:state` 记录结算，本方法不做本地预改。
func (g *GoalControl) Apply(ctx context.Context) (GoalControlResult, error) {
	return g.sender.ControlGoal(ctx, GoalControlRequest{
		SessionID: g.intent.SessionID,
		BranchID:  g.intent.BranchID,
		Action:    g.intent.Action,
	})
}
