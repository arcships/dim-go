package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// SessionCollaborationCancelParams 是 `session.collaboration.cancel` 参数：
// 停止本分支向 memberSessionId 发起的那次协作运行。
type SessionCollaborationCancelParams struct {
	SessionID       string  `json:"sessionId"`
	BranchID        *string `json:"branchId,omitempty"`
	MemberSessionID string  `json:"memberSessionId"`
}

// 协作成员停止的结果。
const (
	CollaborationCancelCancelled = "cancelled"
	CollaborationCancelIdle      = "idle"
)

// SessionCollaborationCancelResult 是 `session.collaboration.cancel` 回执：
// cancelled 表示确实停了一次运行，idle 表示已经没有可停的运行。
type SessionCollaborationCancelResult struct {
	Status string `json:"status"`
}

// ParseSessionCollaborationCancelResult 解码并校验一次成员停止；未知字段被忽略。
func ParseSessionCollaborationCancelResult(raw json.RawMessage) (SessionCollaborationCancelResult, error) {
	var result SessionCollaborationCancelResult
	if err := decodeJSON(raw, &result, "session.collaboration.cancel"); err != nil {
		return result, err
	}
	switch result.Status {
	case CollaborationCancelCancelled, CollaborationCancelIdle:
		return result, nil
	default:
		return result, fmt.Errorf("session.collaboration.cancel has an invalid status %q", result.Status)
	}
}

// CollaborationDeliveredData 是 `collaboration/delivered` 事实的数据；成员
// 会话标识从这里取得（status=ok 时的 session_id）。
type CollaborationDeliveredData struct {
	OpEventID string
	Delivery  CollaborationDelivery
}

// ParseCollaborationDeliveredData 解码一次协作投递事实；未知字段被忽略。
func ParseCollaborationDeliveredData(raw json.RawMessage) (CollaborationDeliveredData, error) {
	var wire struct {
		OpEventID string `json:"op_event_id"`
		Status    string `json:"status"`
		AgentID   string `json:"agent_id"`
		SessionID string `json:"session_id"`
		InputID   string `json:"input_event_id"`
		Outcome   string `json:"outcome"`
		Error     string `json:"error"`
	}
	if err := decodeJSON(raw, &wire, "collaboration/delivered"); err != nil {
		return CollaborationDeliveredData{}, err
	}
	if wire.OpEventID == "" {
		return CollaborationDeliveredData{}, errors.New("collaboration/delivered is missing op_event_id")
	}
	delivery := CollaborationDelivery{
		Status:       wire.Status,
		AgentID:      wire.AgentID,
		SessionID:    wire.SessionID,
		InputEventID: wire.InputID,
		Outcome:      wire.Outcome,
		Error:        wire.Error,
	}
	if err := delivery.validate(); err != nil {
		return CollaborationDeliveredData{}, err
	}
	return CollaborationDeliveredData{OpEventID: wire.OpEventID, Delivery: delivery}, nil
}
