package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// 未完成事项的读取方法名。
const (
	MethodSessionInputPending         = "session.input.pending"
	MethodSessionExecPending          = "session.exec.pending"
	MethodSessionQuestionPending      = "session.question.pending"
	MethodSessionCollaborationPending = "session.collaboration.pending"
)

// PendingQuery 是四类未完成事项读取的共同参数。
//
// BranchID 必填；Through 是固定的截止点（省略即当前头部）；After 是不透明的
// 分页游标，必须把上一页的 `next` 原样带回。
type PendingQuery struct {
	SessionID string          `json:"sessionId"`
	BranchID  string          `json:"branchId"`
	Through   *int64          `json:"through,omitempty"`
	After     json.RawMessage `json:"after,omitempty"`
}

// PendingPage 是四类未完成事项回执的共同部分。Through 是本页的截止点；
// Next 非空（且不是 null）时表示还有下一页。
type PendingPage struct {
	BranchID string          `json:"branchId"`
	Through  int64           `json:"through"`
	Next     json.RawMessage `json:"next,omitempty"`
}

// HasNext 判断是否还有下一页。
func (p PendingPage) HasNext() bool {
	return len(p.Next) > 0 && string(p.Next) != "null"
}

// PendingInputUser 是未消费输入里的用户内容；块类型由上层解释。
type PendingInputUser struct {
	Content []json.RawMessage `json:"content"`
}

// 输入投递方式。
const (
	InputModeQueue  = "queue"
	InputModeSteer  = "steer"
	InputModeAppend = "append"
)

// PendingInput 是一个尚未被消费的输入。
type PendingInput struct {
	InputID  string            `json:"inputId"`
	Sequence int64             `json:"sequence"`
	Mode     string            `json:"mode"`
	User     *PendingInputUser `json:"user,omitempty"`
	// Applied 表示已应用到上下文但尚未消费，因而不允许撤回。
	Applied bool `json:"applied"`
}

// SessionInputPendingResult 是 `session.input.pending` 回执。
type SessionInputPendingResult struct {
	PendingPage
	Inputs []PendingInput `json:"inputs"`
}

// ParseSessionInputPendingResult 解码并校验一页未消费输入；未知字段被忽略。
func ParseSessionInputPendingResult(raw json.RawMessage) (SessionInputPendingResult, error) {
	var result SessionInputPendingResult
	if err := decodeJSON(raw, &result, "session.input.pending"); err != nil {
		return result, err
	}
	if err := result.PendingPage.validate("session.input.pending"); err != nil {
		return result, err
	}
	if result.Inputs == nil {
		return result, errors.New("session.input.pending is missing inputs")
	}
	for index, input := range result.Inputs {
		switch {
		case input.InputID == "":
			return result, fmt.Errorf("session.input.pending inputs[%d] is missing inputId", index)
		case input.Mode != InputModeQueue && input.Mode != InputModeSteer && input.Mode != InputModeAppend:
			return result, fmt.Errorf("session.input.pending inputs[%d] has an invalid mode %q", index, input.Mode)
		}
	}
	return result, nil
}

// PendingExecTask 是一个尚未结束的后台任务。
type PendingExecTask struct {
	EventID     string `json:"eventId"`
	Sequence    int64  `json:"sequence"`
	StartedAt   int64  `json:"startedAt"`
	TaskID      string `json:"taskId"`
	TurnID      string `json:"turnId"`
	RequestID   string `json:"requestId"`
	CallID      string `json:"callId"`
	Description string `json:"description"`
}

// SessionExecPendingResult 是 `session.exec.pending` 回执。
type SessionExecPendingResult struct {
	PendingPage
	Tasks []PendingExecTask `json:"tasks"`
}

// ParseSessionExecPendingResult 解码并校验一页未结束后台任务；未知字段被忽略。
func ParseSessionExecPendingResult(raw json.RawMessage) (SessionExecPendingResult, error) {
	var result SessionExecPendingResult
	if err := decodeJSON(raw, &result, "session.exec.pending"); err != nil {
		return result, err
	}
	if err := result.PendingPage.validate("session.exec.pending"); err != nil {
		return result, err
	}
	if result.Tasks == nil {
		return result, errors.New("session.exec.pending is missing tasks")
	}
	for index, task := range result.Tasks {
		switch {
		case task.EventID == "":
			return result, fmt.Errorf("session.exec.pending tasks[%d] is missing eventId", index)
		case task.TaskID == "":
			return result, fmt.Errorf("session.exec.pending tasks[%d] is missing taskId", index)
		}
	}
	return result, nil
}

// PendingQuestionCall 是 `ask_user` 批次里的一个工具调用。
type PendingQuestionCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// PendingQuestionBatch 是一个还没被结算的 `ask_user` 批次。
type PendingQuestionBatch struct {
	EventID   string                `json:"eventId"`
	Sequence  int64                 `json:"sequence"`
	RequestID string                `json:"requestId"`
	Calls     []PendingQuestionCall `json:"calls"`
}

// SessionQuestionPendingResult 是 `session.question.pending` 回执。
type SessionQuestionPendingResult struct {
	PendingPage
	Batches []PendingQuestionBatch `json:"batches"`
}

// ParseSessionQuestionPendingResult 解码并校验一页待答提问；未知字段被忽略。
func ParseSessionQuestionPendingResult(raw json.RawMessage) (SessionQuestionPendingResult, error) {
	var result SessionQuestionPendingResult
	if err := decodeJSON(raw, &result, "session.question.pending"); err != nil {
		return result, err
	}
	if err := result.PendingPage.validate("session.question.pending"); err != nil {
		return result, err
	}
	if result.Batches == nil {
		return result, errors.New("session.question.pending is missing batches")
	}
	for index, batch := range result.Batches {
		if batch.EventID == "" || batch.RequestID == "" {
			return result, fmt.Errorf("session.question.pending batches[%d] is missing identity", index)
		}
		if batch.Calls == nil {
			return result, fmt.Errorf("session.question.pending batches[%d] is missing calls", index)
		}
		for _, call := range batch.Calls {
			if call.ID == "" || call.Name == "" {
				return result, fmt.Errorf("session.question.pending batches[%d] has an invalid call", index)
			}
		}
	}
	return result, nil
}

// CollaborationTarget 是协作请求的接收方。字段为 snake_case，与服务端一致。
type CollaborationTarget struct {
	Kind      string          `json:"kind"`
	AgentID   string          `json:"agent_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Profile   json.RawMessage `json:"profile,omitempty"`
}

// CollaborationRequestData 是 `collaboration/request` 事实的数据。
type CollaborationRequestData struct {
	TurnID    string              `json:"turn_id"`
	RequestID string              `json:"request_id"`
	CallID    string              `json:"call_id"`
	Target    CollaborationTarget `json:"target"`
	Prompt    string              `json:"prompt"`
}

// 协作投递状态。
const (
	CollaborationDeliveryOK    = "ok"
	CollaborationDeliveryError = "error"
)

// CollaborationDelivery 是一次成员投递的结果；字段为 snake_case。
type CollaborationDelivery struct {
	Status       string `json:"status"`
	AgentID      string `json:"agent_id,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	InputEventID string `json:"input_event_id,omitempty"`
	Outcome      string `json:"outcome,omitempty"`
	Error        string `json:"error,omitempty"`
}

// PendingCollaboration 是一个尚未结束的协作请求。
type PendingCollaboration struct {
	EventID   string                   `json:"eventId"`
	Sequence  int64                    `json:"sequence"`
	StartedAt int64                    `json:"startedAt"`
	Request   CollaborationRequestData `json:"request"`
	Delivery  *CollaborationDelivery   `json:"delivery,omitempty"`
}

// SessionCollaborationPendingResult 是 `session.collaboration.pending` 回执。
type SessionCollaborationPendingResult struct {
	PendingPage
	Requests []PendingCollaboration `json:"requests"`
}

// ParseSessionCollaborationPendingResult 解码并校验一页协作请求；未知字段被忽略。
func ParseSessionCollaborationPendingResult(raw json.RawMessage) (SessionCollaborationPendingResult, error) {
	var result SessionCollaborationPendingResult
	if err := decodeJSON(raw, &result, "session.collaboration.pending"); err != nil {
		return result, err
	}
	if err := result.PendingPage.validate("session.collaboration.pending"); err != nil {
		return result, err
	}
	if result.Requests == nil {
		return result, errors.New("session.collaboration.pending is missing requests")
	}
	for index, request := range result.Requests {
		if request.EventID == "" {
			return result, fmt.Errorf("session.collaboration.pending requests[%d] is missing eventId", index)
		}
		if err := request.Request.validate(); err != nil {
			return result, fmt.Errorf("session.collaboration.pending requests[%d]: %w", index, err)
		}
		if request.Delivery != nil {
			if err := request.Delivery.validate(); err != nil {
				return result, fmt.Errorf("session.collaboration.pending requests[%d]: %w", index, err)
			}
		}
	}
	return result, nil
}

func (p PendingPage) validate(label string) error {
	if p.BranchID == "" {
		return fmt.Errorf("%s is missing branchId", label)
	}
	return nil
}

func (r CollaborationRequestData) validate() error {
	switch {
	case r.TurnID == "":
		return errors.New("request is missing turn_id")
	case r.RequestID == "":
		return errors.New("request is missing request_id")
	case r.CallID == "":
		return errors.New("request is missing call_id")
	}
	switch r.Target.Kind {
	case "agent":
		if r.Target.AgentID == "" {
			return errors.New("request target is missing agent_id")
		}
	case "member":
		if r.Target.AgentID == "" || r.Target.SessionID == "" {
			return errors.New("request target is missing member identity")
		}
	case "temporary":
		if len(r.Target.Profile) == 0 {
			return errors.New("request target is missing profile")
		}
	default:
		return fmt.Errorf("request target has an invalid kind %q", r.Target.Kind)
	}
	return nil
}

func (d CollaborationDelivery) validate() error {
	switch d.Status {
	case CollaborationDeliveryOK:
		if d.AgentID == "" || d.SessionID == "" {
			return errors.New("delivery is missing member identity")
		}
	case CollaborationDeliveryError:
		if d.Error == "" {
			return errors.New("delivery is missing error")
		}
	default:
		return fmt.Errorf("delivery has an invalid status %q", d.Status)
	}
	return nil
}
