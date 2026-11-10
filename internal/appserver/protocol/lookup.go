package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// MethodSessionInputLookup 是写登记核对用的提交标识查询。
const MethodSessionInputLookup = "session.input_lookup"

// InputLookupParams 是 `session.input_lookup` 参数。
//
// v3 要求 branchId 必填；SubmissionID 必须是 UUID，服务端对其他格式返回
// invalid_params。
type InputLookupParams struct {
	SessionID    string `json:"sessionId"`
	BranchID     string `json:"branchId"`
	SubmissionID string `json:"submissionId"`
}

// 提交标识查询的结果状态。
const (
	InputLookupAccepted = "accepted"
	InputLookupUnknown  = "unknown"
)

// InputLookupResult 是 `session.input_lookup` 回执：accepted 带落库的
// `input/message` 事实；unknown 表示服务端没有这次写入。
type InputLookupResult struct {
	Status string    `json:"status"`
	Event  *RawEvent `json:"event,omitempty"`
}

// ParseInputLookupResult 解码并校验一次提交标识查询；未知字段被忽略。
func ParseInputLookupResult(raw json.RawMessage) (InputLookupResult, error) {
	var result InputLookupResult
	if err := decodeJSON(raw, &result, "session.input_lookup"); err != nil {
		return result, err
	}
	switch result.Status {
	case InputLookupAccepted:
		if result.Event == nil {
			return result, errors.New("session.input_lookup accepted result is missing event")
		}
	case InputLookupUnknown:
		if result.Event != nil {
			return result, errors.New("session.input_lookup unknown result must not carry an event")
		}
	default:
		return result, fmt.Errorf("session.input_lookup has an invalid status %q", result.Status)
	}
	return result, nil
}
