package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Modality 能力快照（与 wire 的 camelCase 一致）。
type ModalityReason struct {
	Dimension string `json:"dimension"`
	Detail    string `json:"detail"`
}

// ModalityOption 是一个可选的模型。
type ModalityOption struct {
	ProviderID string          `json:"providerId"`
	ModelID    string          `json:"modelId"`
	Available  bool            `json:"available"`
	Reason     *ModalityReason `json:"reason,omitempty"`
}

// ModalityEffective 是当前会实际使用的模型。
type ModalityEffective struct {
	ProviderID string `json:"providerId"`
	ModelID    string `json:"modelId"`
	Source     string `json:"source"`
}

// ModalityOperation 是一个操作的可用性。
type ModalityOperation struct {
	Operation string             `json:"operation"`
	Key       *string            `json:"key,omitempty"`
	Available bool               `json:"available"`
	Reasons   []ModalityReason   `json:"reasons"`
	Default   *ModalityOption    `json:"default"`
	Effective *ModalityEffective `json:"effective"`
	Options   []ModalityOption   `json:"options"`
}

// ModalityCategory 是一个类别。
type ModalityCategory struct {
	Category   string              `json:"category"`
	Available  bool                `json:"available"`
	Reasons    []ModalityReason    `json:"reasons"`
	Operations []ModalityOperation `json:"operations"`
}

// ModalityCapabilitiesResult 是 `modality.capabilities` / `configure` 回执。
type ModalityCapabilitiesResult struct {
	Revision   string             `json:"revision"`
	Categories []ModalityCategory `json:"categories"`
}

// ParseModalityCapabilitiesResult 解码并校验能力快照；未知字段被忽略。
func ParseModalityCapabilitiesResult(raw json.RawMessage) (ModalityCapabilitiesResult, error) {
	var result ModalityCapabilitiesResult
	if err := decodeJSON(raw, &result, "modality.capabilities"); err != nil {
		return result, err
	}
	if result.Revision == "" {
		return result, errors.New("modality.capabilities is missing revision")
	}
	if result.Categories == nil {
		return result, errors.New("modality.capabilities is missing categories")
	}
	for index, category := range result.Categories {
		if category.Category == "" || category.Reasons == nil || category.Operations == nil {
			return result, fmt.Errorf("modality.capabilities categories[%d] is missing fields", index)
		}
		for _, operation := range category.Operations {
			if operation.Operation == "" || operation.Reasons == nil || operation.Options == nil {
				return result, fmt.Errorf("modality.capabilities categories[%d] has an invalid operation", index)
			}
		}
	}
	return result, nil
}

// ModalityConfigureParams 是 `modality.configure` 参数；`model` 为 null 表示清空。
type ModalityConfigureParams struct {
	ExpectedRevision string          `json:"expectedRevision"`
	Category         string          `json:"category"`
	Operation        *string         `json:"operation,omitempty"`
	Model            json.RawMessage `json:"model"`
}

// ModalityModelRef 是模型引用。
type ModalityModelRef struct {
	ProviderID string `json:"providerId"`
	ModelID    string `json:"modelId"`
}

// ModalityTaskState 是任务状态（snake_case）。
const (
	ModalityTaskQueued      = "queued"
	ModalityTaskRunning     = "running"
	ModalityTaskSucceeded   = "succeeded"
	ModalityTaskFailed      = "failed"
	ModalityTaskCancelled   = "cancelled"
	ModalityTaskInterrupted = "interrupted"
)

// ModalityTaskSubmitParams 是 `modality.task.submit` 参数（snake_case 字段）。
type ModalityTaskSubmitParams struct {
	Category     string            `json:"category"`
	Operation    string            `json:"operation"`
	Params       json.RawMessage   `json:"params,omitempty"`
	Files        []json.RawMessage `json:"files,omitempty"`
	Model        *ModalityModelRef `json:"model,omitempty"`
	Node         *string           `json:"node,omitempty"`
	ClientTaskID *string           `json:"client_task_id,omitempty"`
}

// ModalityTaskStatusParams 是 `modality.task.status` / `cancel` / `wait` 的定位参数。
type ModalityTaskStatusParams struct {
	TaskID string `json:"task_id"`
}

// ModalityTaskWaitParams 是 `modality.task.wait` 参数。
type ModalityTaskWaitParams struct {
	TaskID    string `json:"task_id"`
	TimeoutMS *int64 `json:"timeout_ms,omitempty"`
}

// ModalityTaskResultParams 是 `modality.task.result` 参数。
type ModalityTaskResultParams struct {
	TaskID string `json:"task_id"`
	Name   string `json:"name"`
	Offset *int64 `json:"offset,omitempty"`
	Length int64  `json:"length"`
}

// ModalityTaskRecord 是 owner 的任务记录；执行器/参数/结果按原样保留。
type ModalityTaskRecord struct {
	Version      int64           `json:"version"`
	TaskID       string          `json:"task_id"`
	ClientTaskID *string         `json:"client_task_id"`
	Revision     int64           `json:"revision"`
	Category     string          `json:"category"`
	Operation    string          `json:"operation"`
	Executor     json.RawMessage `json:"executor"`
	Params       json.RawMessage `json:"params"`
	State        string          `json:"state"`
	RemoteHandle *string         `json:"remote_handle"`
	Progress     json.RawMessage `json:"progress"`
	Result       json.RawMessage `json:"result"`
	Error        json.RawMessage `json:"error"`
	RemoteCancel *string         `json:"remote_cancel"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
	FinishedAt   *string         `json:"finished_at"`
}

// ModalityTaskReply 是 submit/status/wait/cancel 的回执。
type ModalityTaskReply struct {
	Task ModalityTaskRecord `json:"task"`
}

// ParseModalityTaskReply 解码并校验一次任务回执；未知字段被忽略。
func ParseModalityTaskReply(raw json.RawMessage) (ModalityTaskReply, error) {
	var result ModalityTaskReply
	if err := decodeJSON(raw, &result, "modality.task"); err != nil {
		return result, err
	}
	if err := result.Task.validate("modality.task"); err != nil {
		return result, err
	}
	return result, nil
}

// ModalityTaskListParams 是 `modality.task.list` 参数。
type ModalityTaskListParams struct {
	State    *string `json:"state,omitempty"`
	Category *string `json:"category,omitempty"`
	Limit    *int    `json:"limit,omitempty"`
	After    *string `json:"after,omitempty"`
}

// ModalityTaskListResult 是 `modality.task.list` 回执。
type ModalityTaskListResult struct {
	Tasks []ModalityTaskRecord `json:"tasks"`
	Next  *string              `json:"next,omitempty"`
}

// ParseModalityTaskListResult 解码并校验任务清单；未知字段被忽略。
func ParseModalityTaskListResult(raw json.RawMessage) (ModalityTaskListResult, error) {
	var result ModalityTaskListResult
	if err := decodeJSON(raw, &result, "modality.task.list"); err != nil {
		return result, err
	}
	if result.Tasks == nil {
		return result, errors.New("modality.task.list is missing tasks")
	}
	for index, task := range result.Tasks {
		if err := task.validate("modality.task.list"); err != nil {
			return result, fmt.Errorf("tasks[%d]: %w", index, err)
		}
	}
	return result, nil
}

// ModalityTaskResultReply 是一页任务结果。
type ModalityTaskResultReply struct {
	DataBase64 string `json:"data_base64"`
	EOF        bool   `json:"eof"`
}

// ParseModalityTaskResultReply 解码并校验一页结果；未知字段被忽略。
func ParseModalityTaskResultReply(raw json.RawMessage) (ModalityTaskResultReply, error) {
	var result ModalityTaskResultReply
	if err := decodeJSON(raw, &result, "modality.task.result"); err != nil {
		return result, err
	}
	if result.DataBase64 == "" && !result.EOF {
		return result, errors.New("modality.task.result has neither data nor eof")
	}
	return result, nil
}

// ModalityRealtimeOpenParams 是 `modality.realtime.open` 参数。
type ModalityRealtimeOpenParams struct {
	Operation string            `json:"operation"`
	Model     *ModalityModelRef `json:"model,omitempty"`
	Params    json.RawMessage   `json:"params,omitempty"`
}

// ModalityRealtimeOpenResult 是一次实时会话的建立结果。
type ModalityRealtimeOpenResult struct {
	ConnectionID string          `json:"connectionId"`
	Transport    string          `json:"transport"`
	Connection   json.RawMessage `json:"connection,omitempty"`
}

// ParseModalityRealtimeOpenResult 解码并校验一次实时会话；未知字段被忽略。
func ParseModalityRealtimeOpenResult(raw json.RawMessage) (ModalityRealtimeOpenResult, error) {
	var result ModalityRealtimeOpenResult
	if err := decodeJSON(raw, &result, "modality.realtime.open"); err != nil {
		return result, err
	}
	if result.ConnectionID == "" {
		return result, errors.New("modality.realtime.open is missing connectionId")
	}
	switch result.Transport {
	case "relay", "direct":
	default:
		return result, fmt.Errorf("modality.realtime.open has an invalid transport %q", result.Transport)
	}
	return result, nil
}

// ModalityRealtimeFrame 是一帧文本、音频或控制数据。
type ModalityRealtimeFrame struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

// ModalityRealtimeSendParams 是 `modality.realtime.send` 参数。
type ModalityRealtimeSendParams struct {
	ConnectionID string                `json:"connectionId"`
	Frame        ModalityRealtimeFrame `json:"frame"`
}

// ModalityRealtimeCloseParams 是 `modality.realtime.close` 参数。
type ModalityRealtimeCloseParams struct {
	ConnectionID string `json:"connectionId"`
}

// ModalityRealtimeEventNotice 是 `modality.realtime.event` 通知。
type ModalityRealtimeEventNotice struct {
	ConnectionID string          `json:"connectionId"`
	Event        json.RawMessage `json:"event"`
}

// ParseModalityRealtimeEventNotice 解码并校验一条实时事件；未知字段被忽略。
func ParseModalityRealtimeEventNotice(raw json.RawMessage) (ModalityRealtimeEventNotice, error) {
	var notice ModalityRealtimeEventNotice
	if err := decodeJSON(raw, &notice, "modality.realtime.event"); err != nil {
		return notice, err
	}
	if notice.ConnectionID == "" || len(notice.Event) == 0 {
		return notice, errors.New("modality.realtime.event is missing connectionId or event")
	}
	return notice, nil
}

// ModalityRealtimeEndedNotice 是 `modality.realtime.ended` 通知。
type ModalityRealtimeEndedNotice struct {
	ConnectionID string `json:"connectionId"`
	Reason       string `json:"reason"`
}

// ParseModalityRealtimeEndedNotice 解码并校验一条实时结束通知；未知字段被忽略。
func ParseModalityRealtimeEndedNotice(raw json.RawMessage) (ModalityRealtimeEndedNotice, error) {
	var notice ModalityRealtimeEndedNotice
	if err := decodeJSON(raw, &notice, "modality.realtime.ended"); err != nil {
		return notice, err
	}
	if notice.ConnectionID == "" || notice.Reason == "" {
		return notice, errors.New("modality.realtime.ended is missing connectionId or reason")
	}
	return notice, nil
}

func (t ModalityTaskRecord) validate(label string) error {
	if t.TaskID == "" || t.Category == "" || t.Operation == "" {
		return fmt.Errorf("%s task is missing identity", label)
	}
	switch t.State {
	case ModalityTaskQueued, ModalityTaskRunning, ModalityTaskSucceeded,
		ModalityTaskFailed, ModalityTaskCancelled, ModalityTaskInterrupted:
		return nil
	default:
		return fmt.Errorf("%s task has an invalid state %q", label, t.State)
	}
}
