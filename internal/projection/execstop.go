package projection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

// 后台任务停止：按任务标识发一次停止，结果由结束事实收敛。
//
// 停止是一次写入，走写登记：本包只给登记内容（op、幂等键、参数）与发送接口，
// 不自己登记、不自己重发。集成层把 Intent 交给写登记，用已有的 KindStop
// （「停止会话或分支上的运行」）登记，发送方实现 ExecStopSender。

// 任务停止的错误值。
var (
	// ErrExecStopNoSender 表示没有配置停止发送方。
	ErrExecStopNoSender = errors.New("任务停止：缺少发送方")
	// ErrExecStopNoTask 表示没有任务标识。
	ErrExecStopNoTask = errors.New("任务停止：缺少任务标识")
)

// execStopOp 是停止意图的操作名；与写登记里的其他 op 一样放进参数。
const execStopOp = "exec.stop"

// ExecStopRequest 是一次任务停止请求。
type ExecStopRequest struct {
	SessionID string
	BranchID  string
	TaskID    string
}

// ExecStopResult 是服务端给出的结束事实（与 `exec/ended` 同形状的关键字段）。
type ExecStopResult struct {
	TaskID string
	// Status 取 completed、failed、cancelled 之一。
	Status string
}

// Stopped 报告这次停止是否以取消收口。
func (r ExecStopResult) Stopped() bool { return r.Status == "cancelled" }

// ExecStopSender 发送一次已经登记的后台任务停止。集成层用写登记的发送方实现，
// 由它决定走哪条连接。
type ExecStopSender interface {
	StopExecTask(ctx context.Context, req ExecStopRequest) (ExecStopResult, error)
}

// ExecStopIntent 是登记到写登记的停止意图。它只描述"停哪一个任务"，不发送。
type ExecStopIntent struct {
	SessionID string
	BranchID  string
	TaskID    string
}

// Op 是写登记参数里的操作名。
func (i ExecStopIntent) Op() string { return execStopOp }

// Key 是幂等键：按会话、分支与任务派生，同一任务重复点击只登记一次。
func (i ExecStopIntent) Key() string {
	sum := sha256.Sum256([]byte(i.SessionID + "\x00" + i.BranchID + "\x00" + i.TaskID))
	return "projection.exec_stop:" + hex.EncodeToString(sum[:16])
}

// Params 是写登记参数。字段与 Intent 一致，发送方按它拼请求。
func (i ExecStopIntent) Params() (json.RawMessage, error) {
	return json.Marshal(struct {
		Op       string `json:"op"`
		BranchID string `json:"branchId"`
		TaskID   string `json:"taskId"`
	}{Op: execStopOp, BranchID: i.BranchID, TaskID: i.TaskID})
}

// ExecTaskStop 按任务标识停止一个后台任务。
type ExecTaskStop struct {
	folder *Folder
	intent ExecStopIntent
	sender ExecStopSender
}

// NewExecTaskStop 创建一次任务停止。folder 用来把"正在停止"标到清单上，可空。
func NewExecTaskStop(folder *Folder, branchID, taskID string, sender ExecStopSender) (*ExecTaskStop, error) {
	if sender == nil {
		return nil, ErrExecStopNoSender
	}
	if taskID == "" {
		return nil, ErrExecStopNoTask
	}
	if branchID == "" {
		branchID = mainBranch
	}
	stop := &ExecTaskStop{
		folder: folder,
		intent: ExecStopIntent{BranchID: branchID, TaskID: taskID},
		sender: sender,
	}
	if folder != nil {
		stop.intent.SessionID = folder.sessionID
	}
	return stop, nil
}

// Intent 返回要登记到写登记的停止意图。
func (s *ExecTaskStop) Intent() ExecStopIntent { return s.intent }

// Stop 发送停止并等它收敛。服务端在任务结束后才回执，所以返回时状态已是非运行；
// 结束事实（`exec/ended`）随后到达，清单由它结算。回执先到时，本地把任务标成
// 正在停止，直到结束事实到达。
func (s *ExecTaskStop) Stop(ctx context.Context) (ExecStopResult, error) {
	result, err := s.sender.StopExecTask(ctx, ExecStopRequest{
		SessionID: s.intent.SessionID,
		BranchID:  s.intent.BranchID,
		TaskID:    s.intent.TaskID,
	})
	if err != nil {
		return ExecStopResult{}, err
	}
	if s.folder != nil {
		s.folder.MarkTaskStopping(s.intent.BranchID, s.intent.TaskID)
	}
	return result, nil
}
