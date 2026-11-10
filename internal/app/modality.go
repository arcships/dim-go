package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
	"github.com/arcships/dimagent-mygo/internal/settings"
)

// modalitySession 是 settings 模态调用面的集成层实现：能力门禁、任务读取与实时帧发送。
//
// 门禁顺序固定为 hello 能力 → 会话方法表：真实服务端把 modality.* 声明在
// hello.features，但不贡献会话方法；只查会话方法表会误判缺失。两者都没有才算缺失。
type modalitySession struct {
	app *App
}

func (s *modalitySession) SupportsModalityMethod(
	ctx context.Context,
	sessionID string,
	method string,
) (bool, error) {
	if method == "" {
		return false, nil
	}
	// 先看 hello：daemon 级贡献方法只在这里声明。
	if conn := s.app.connection(); conn != nil && conn.HasFeature(method) {
		return true, nil
	}
	runtime, err := s.app.openRuntime(sessionID, method)
	if err != nil {
		return false, err
	}
	methods, err := s.app.sessionMethods(ctx, runtime)
	if err != nil {
		// 读不到方法表也按缺失处理（与 hello 都确认不了）；记一条诊断便于排障。
		s.app.record("读取会话方法清单失败，按时缺失处理: %v", err)
		return false, nil
	}
	return methods.Has(method), nil
}

// ReadModalityTask 读一条任务记录（`modality.task.status`，会话流）。
func (s *modalitySession) ReadModalityTask(
	ctx context.Context,
	sessionID string,
	taskID string,
) (settings.ModalityTask, error) {
	runtime, err := s.app.openRuntime(sessionID, settings.ModalityMethodTaskStatus)
	if err != nil {
		return settings.ModalityTask{}, err
	}
	raw, err := runtime.lease.Subscription().Call(ctx, settings.ModalityMethodTaskStatus,
		protocol.ModalityTaskStatusParams{TaskID: taskID})
	if err != nil {
		return settings.ModalityTask{}, err
	}
	reply, err := protocol.ParseModalityTaskReply(raw)
	if err != nil {
		return settings.ModalityTask{}, fmt.Errorf("任务状态回执无法解析: %w", err)
	}
	return modalityTaskFromRecord(reply.Task)
}

// ListModalityTasks 读一页任务清单（`modality.task.list`，会话流）。
func (s *modalitySession) ListModalityTasks(
	ctx context.Context,
	sessionID string,
	query settings.ModalityTaskQuery,
) (settings.ModalityTaskPage, error) {
	runtime, err := s.app.openRuntime(sessionID, settings.ModalityMethodTaskList)
	if err != nil {
		return settings.ModalityTaskPage{}, err
	}
	params := protocol.ModalityTaskListParams{}
	if query.State != "" {
		state := string(query.State)
		params.State = &state
	}
	if query.Category != "" {
		category := query.Category
		params.Category = &category
	}
	if query.Limit > 0 {
		limit := query.Limit
		params.Limit = &limit
	}
	if query.After != "" {
		after := query.After
		params.After = &after
	}
	raw, err := runtime.lease.Subscription().Call(ctx, settings.ModalityMethodTaskList, params)
	if err != nil {
		return settings.ModalityTaskPage{}, err
	}
	result, err := protocol.ParseModalityTaskListResult(raw)
	if err != nil {
		return settings.ModalityTaskPage{}, fmt.Errorf("任务清单回执无法解析: %w", err)
	}
	page := settings.ModalityTaskPage{Tasks: make([]settings.ModalityTask, 0, len(result.Tasks))}
	for _, record := range result.Tasks {
		task, err := modalityTaskFromRecord(record)
		if err != nil {
			return settings.ModalityTaskPage{}, err
		}
		page.Tasks = append(page.Tasks, task)
	}
	if result.Next != nil {
		page.Next = *result.Next
	}
	return page, nil
}

// ReadModalityTaskResult 读一页任务结果（`modality.task.result`，会话流）。
func (s *modalitySession) ReadModalityTaskResult(
	ctx context.Context,
	sessionID string,
	query settings.ModalityTaskResultQuery,
) (settings.ModalityTaskResultPage, error) {
	runtime, err := s.app.openRuntime(sessionID, settings.ModalityMethodTaskResult)
	if err != nil {
		return settings.ModalityTaskResultPage{}, err
	}
	params := protocol.ModalityTaskResultParams{
		TaskID: query.TaskID,
		Name:   query.Name,
		Length: query.Length,
	}
	if query.Offset > 0 {
		offset := query.Offset
		params.Offset = &offset
	}
	raw, err := runtime.lease.Subscription().Call(ctx, settings.ModalityMethodTaskResult, params)
	if err != nil {
		return settings.ModalityTaskResultPage{}, err
	}
	reply, err := protocol.ParseModalityTaskResultReply(raw)
	if err != nil {
		return settings.ModalityTaskResultPage{}, fmt.Errorf("任务结果回执无法解析: %w", err)
	}
	return settings.ModalityTaskResultPage{DataBase64: reply.DataBase64, EOF: reply.EOF}, nil
}

// SendModalityRealtime 发一帧实时数据（`modality.realtime.send`，会话流）。
func (s *modalitySession) SendModalityRealtime(
	ctx context.Context,
	sessionID string,
	connectionID string,
	frame settings.ModalityRealtimeFrame,
) error {
	runtime, err := s.app.openRuntime(sessionID, settings.ModalityMethodRealtimeSend)
	if err != nil {
		return err
	}
	_, err = runtime.lease.Subscription().Call(ctx, settings.ModalityMethodRealtimeSend,
		protocol.ModalityRealtimeSendParams{
			ConnectionID: connectionID,
			Frame: protocol.ModalityRealtimeFrame{
				Kind: frame.Kind,
				Data: frame.Data,
			},
		})
	return err
}

// routeModalityNotice 把实时事件与结束通知交给 settings 的实时面。
func (a *App) routeModalityNotice(sessionID string, event session.Event) {
	domain := a.Settings()
	if domain == nil {
		return
	}
	switch event.Method {
	case "modality.realtime.event":
		notice, err := protocol.ParseModalityRealtimeEventNotice(event.Params)
		if err != nil {
			a.record("实时事件无法解析: %v", err)
			return
		}
		if err := domain.DeliverRealtimeEvent(settings.ModalityRealtimeEvent{
			SessionID:    sessionID,
			ConnectionID: notice.ConnectionID,
			Event:        notice.Event,
		}); err != nil {
			a.record("实时事件投递失败: %v", err)
		}
	case "modality.realtime.ended":
		notice, err := protocol.ParseModalityRealtimeEndedNotice(event.Params)
		if err != nil {
			a.record("实时结束通知无法解析: %v", err)
			return
		}
		if err := domain.DeliverRealtimeEnded(settings.ModalityRealtimeEnded{
			SessionID:    sessionID,
			ConnectionID: notice.ConnectionID,
			Reason:       notice.Reason,
		}); err != nil {
			a.record("实时结束投递失败: %v", err)
		}
	}
}

// modalityTaskFromRecord 把 owner 的任务记录映射成设置领域的只读视图。
func modalityTaskFromRecord(record protocol.ModalityTaskRecord) (settings.ModalityTask, error) {
	task := settings.ModalityTask{
		ID:         record.TaskID,
		Revision:   record.Revision,
		Category:   record.Category,
		Operation:  record.Operation,
		State:      settings.ModalityTaskState(record.State),
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
		FinishedAt: "",
	}
	if record.ClientTaskID != nil {
		task.ClientTaskID = *record.ClientTaskID
	}
	if record.FinishedAt != nil {
		task.FinishedAt = *record.FinishedAt
	}
	if task.ID == "" {
		return settings.ModalityTask{}, fmt.Errorf("任务记录缺少标识")
	}
	if progress := parseModalityProgress(record.Progress); progress != nil {
		task.Progress = progress
	}
	if result := parseModalityResult(record.Result); result != nil {
		task.Result = result
	}
	if failure := parseModalityError(record.Error); failure != nil {
		task.Error = failure
	}
	return task, nil
}

// parseModalityProgress 解析进度载荷：`{fraction?, message?}`，认不出的字段忽略。
func parseModalityProgress(raw json.RawMessage) *settings.ModalityTaskProgress {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var payload struct {
		Fraction *float64 `json:"fraction"`
		Message  string   `json:"message"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	return &settings.ModalityTaskProgress{Fraction: payload.Fraction, Message: payload.Message}
}

// parseModalityResult 解析结果载荷：`{files[{name,mediaType,bytes,sha256}], data}`。
func parseModalityResult(raw json.RawMessage) *settings.ModalityTaskResultData {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var payload struct {
		Files []map[string]any `json:"files"`
		Data  json.RawMessage  `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	result := &settings.ModalityTaskResultData{Data: append(json.RawMessage(nil), payload.Data...)}
	for _, file := range payload.Files {
		mapped := settings.ModalityTaskFile{
			// 真实回执用 snake_case，旧夹具用 camelCase；两种都接受。
			Name:      firstNonEmpty(stringValue(file, "name"), ""),
			MediaType: firstNonEmpty(stringValue(file, "mediaType"), stringValue(file, "media_type"), stringValue(file, "mimeType")),
			SHA256:    firstNonEmpty(stringValue(file, "sha256"), stringValue(file, "digest")),
		}
		if bytesValue, ok := file["bytes"].(float64); ok {
			mapped.Bytes = int64(bytesValue)
		}
		result.Files = append(result.Files, mapped)
	}
	return result
}

// stringValue 取 map 里的字符串字段；不存在或类型不对时返回空。
func stringValue(values map[string]any, key string) string {
	if text, ok := values[key].(string); ok {
		return text
	}
	return ""
}

// parseModalityError 解析失败载荷：`{code, message}`。
func parseModalityError(raw json.RawMessage) *settings.ModalityTaskError {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	if payload.Code == "" && payload.Message == "" {
		return nil
	}
	return &settings.ModalityTaskError{Code: payload.Code, Message: payload.Message}
}
