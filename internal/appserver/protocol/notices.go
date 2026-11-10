package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// 通知方法名。
const (
	MethodApprovalRequest   = "approval.request"
	MethodApprovalCancelled = "approval.cancelled"
	MethodApprovalResolved  = "approval.resolved"
	MethodSessionNotice     = "session.notice"
	MethodSessionClosed     = "session.closed"
)

// ApprovalDecision 是审批的最终决定，与服务端 wire 一致。
const (
	ApprovalDecisionAllow = "allow"
	ApprovalDecisionDeny  = "deny"
)

// ApprovalSource 是 v3 审批来源（服务端 BranchSource）。
//
// tool 来源带 turn_id / request_id / call_id；host 来源带 operation_id。
// 当前服务端没有其他 kind；未知字段按容忍处理，不参与解码。
type ApprovalSource struct {
	Kind        string `json:"kind"`
	TurnID      string `json:"turn_id,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
	CallID      string `json:"call_id,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
}

// ApprovalCall 是审批指向的工具调用。
type ApprovalCall struct {
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ApprovalRequestNotice 是 `approval.request` 通知。
type ApprovalRequestNotice struct {
	SessionID string         `json:"sessionId"`
	BranchID  string         `json:"branchId"`
	RequestID string         `json:"requestId"`
	Source    ApprovalSource `json:"source"`
	Call      ApprovalCall   `json:"call"`
	Reason    string         `json:"reason"`
}

// ParseApprovalRequestNotice 解码并校验一条审批请求；未知字段被忽略。
func ParseApprovalRequestNotice(raw json.RawMessage) (ApprovalRequestNotice, error) {
	var notice ApprovalRequestNotice
	if err := decodeJSON(raw, &notice, "approval.request"); err != nil {
		return notice, err
	}
	switch {
	case notice.SessionID == "":
		return notice, errors.New("approval.request is missing sessionId")
	case notice.BranchID == "":
		return notice, errors.New("approval.request is missing branchId")
	case notice.RequestID == "":
		return notice, errors.New("approval.request is missing requestId")
	case notice.Source.Kind == "":
		return notice, errors.New("approval.request is missing source.kind")
	case notice.Call.Name == "":
		return notice, errors.New("approval.request is missing call.name")
	}
	return notice, nil
}

// ApprovalCancelledNotice 是 `approval.cancelled` 通知：审批在开始执行前被取消。
type ApprovalCancelledNotice struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	RequestID string `json:"requestId"`
	Reason    string `json:"reason"`
}

// ParseApprovalCancelledNotice 解码并校验一条审批取消通知；未知字段被忽略。
func ParseApprovalCancelledNotice(raw json.RawMessage) (ApprovalCancelledNotice, error) {
	var notice ApprovalCancelledNotice
	if err := decodeJSON(raw, &notice, "approval.cancelled"); err != nil {
		return notice, err
	}
	switch {
	case notice.SessionID == "":
		return notice, errors.New("approval.cancelled is missing sessionId")
	case notice.BranchID == "":
		return notice, errors.New("approval.cancelled is missing branchId")
	case notice.RequestID == "":
		return notice, errors.New("approval.cancelled is missing requestId")
	}
	return notice, nil
}

// ApprovalResolvedNotice 是 `approval.resolved` 通知：审批已被结算。
// 同一 requestId 的第一条携带被接受的决定；后续 `approval.cancelled`
// 表示调用没有开始。
type ApprovalResolvedNotice struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	RequestID string `json:"requestId"`
	Decision  string `json:"decision"`
}

// ParseApprovalResolvedNotice 解码并校验一条审批结算通知；未知字段被忽略。
func ParseApprovalResolvedNotice(raw json.RawMessage) (ApprovalResolvedNotice, error) {
	var notice ApprovalResolvedNotice
	if err := decodeJSON(raw, &notice, "approval.resolved"); err != nil {
		return notice, err
	}
	switch {
	case notice.SessionID == "":
		return notice, errors.New("approval.resolved is missing sessionId")
	case notice.BranchID == "":
		return notice, errors.New("approval.resolved is missing branchId")
	case notice.RequestID == "":
		return notice, errors.New("approval.resolved is missing requestId")
	case notice.Decision != ApprovalDecisionAllow && notice.Decision != ApprovalDecisionDeny:
		return notice, fmt.Errorf("approval.resolved has an invalid decision %q", notice.Decision)
	}
	return notice, nil
}

// `session.notice` 的已知来源与类别。其它来源/类别原样保留给上层。
const (
	NoticeSourceSessionStore = "SessionStore"
	NoticeKindTitle          = "title"
)

// SessionNotice 是 `session.notice` 通知：某个 owner 的进程级状态变化。
// 它没有序号、不会被重放；事实以 owner 的读取为准。
type SessionNotice struct {
	SessionID string          `json:"sessionId"`
	Source    string          `json:"source"`
	Kind      string          `json:"kind"`
	Data      json.RawMessage `json:"data"`
}

// ParseSessionNotice 解码并校验一条会话通知；未知字段被忽略。
func ParseSessionNotice(raw json.RawMessage) (SessionNotice, error) {
	var notice SessionNotice
	if err := decodeJSON(raw, &notice, "session.notice"); err != nil {
		return notice, err
	}
	switch {
	case notice.SessionID == "":
		return notice, errors.New("session.notice is missing sessionId")
	case notice.Source == "":
		return notice, errors.New("session.notice is missing source")
	case notice.Kind == "":
		return notice, errors.New("session.notice is missing kind")
	}
	return notice, nil
}

// TitleFields 是标题通知的数据，也是 v3 会话摘要里的标题字段。
// Title 与 TitleSource 为 null 表示标题被清空。
type TitleFields struct {
	Title         *string `json:"title"`
	TitleSource   *string `json:"titleSource"`
	TitleRevision int64   `json:"titleRevision"`
}

// TitleChanged 是从 `session.notice` 里解出的标题变更。
type TitleChanged struct {
	SessionID     string
	Title         *string
	TitleSource   *string
	TitleRevision int64
}

// IsTitle 判断这条通知是否是标题变更。
func (n SessionNotice) IsTitle() bool {
	return n.Source == NoticeSourceSessionStore && n.Kind == NoticeKindTitle
}

// ParseTitle 解码标题变更；不是标题通知或数据不合法时返回错误。
func (n SessionNotice) ParseTitle() (TitleChanged, error) {
	if !n.IsTitle() {
		return TitleChanged{}, fmt.Errorf("session.notice is %s/%s, not a title change", n.Source, n.Kind)
	}
	var fields TitleFields
	if err := decodeJSON(n.Data, &fields, "session.notice title"); err != nil {
		return TitleChanged{}, err
	}
	if fields.TitleRevision < 1 {
		return TitleChanged{}, errors.New("title notice is missing titleRevision")
	}
	return TitleChanged{
		SessionID:     n.SessionID,
		Title:         fields.Title,
		TitleSource:   fields.TitleSource,
		TitleRevision: fields.TitleRevision,
	}, nil
}

// decodeJSON 是各 wire 类型的统一入口：未知字段被忽略，载荷为空直接拒绝。
func decodeJSON(raw json.RawMessage, out any, label string) error {
	if len(raw) == 0 {
		return fmt.Errorf("%s: empty payload", label)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	return nil
}
