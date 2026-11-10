package settings

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// ModalityTaskState 是模态任务状态。
type ModalityTaskState string

const (
	ModalityTaskQueued      ModalityTaskState = "queued"
	ModalityTaskRunning     ModalityTaskState = "running"
	ModalityTaskSucceeded   ModalityTaskState = "succeeded"
	ModalityTaskFailed      ModalityTaskState = "failed"
	ModalityTaskCancelled   ModalityTaskState = "cancelled"
	ModalityTaskInterrupted ModalityTaskState = "interrupted"
)

// ModalityTaskResultMaxBytes 是单次结果读取的字节上限，与服务端所有者约束一致。
const ModalityTaskResultMaxBytes = 4 * 1024 * 1024

// ModalityModelRef 是一次调用使用的模型引用；两个标识都由调用方给出。
type ModalityModelRef struct {
	ProviderID string
	ModelID    string
}

// ModalityTaskFileInput 是任务输入文件。DataBase64 与 NodePath 只能给一个。
type ModalityTaskFileInput struct {
	Name       string
	MediaType  string
	DataBase64 string
	NodePath   string
}

// ModalityTaskFile 是任务结果里的一个文件。
type ModalityTaskFile struct {
	Name      string
	MediaType string
	Bytes     int64
	SHA256    string
}

// ModalityTaskProgress 是任务进度；Fraction 为空表示服务端没有给。
type ModalityTaskProgress struct {
	Fraction *float64
	Message  string
}

// ModalityTaskResultData 是任务结果数据；Data 原样保留服务端 JSON。
type ModalityTaskResultData struct {
	Files []ModalityTaskFile
	Data  json.RawMessage
}

// ModalityTaskError 是任务失败原因。
type ModalityTaskError struct {
	Code    string
	Message string
}

// ModalityTask 是服务端确认的任务记录视图。可选字段用指针或空值保留「没有给」。
type ModalityTask struct {
	// SessionID 是任务所属会话；由本包在读取时填入，服务端任务记录本身不带会话。
	SessionID    string
	ID           string
	ClientTaskID string
	Revision     int64
	Category     string
	Operation    string
	State        ModalityTaskState
	Progress     *ModalityTaskProgress
	Result       *ModalityTaskResultData
	Error        *ModalityTaskError
	CreatedAt    string
	UpdatedAt    string
	FinishedAt   string
}

// ModalityTaskSubmitRequest 是一次任务提交。Params 为空表示调用方没有提供参数，
// 本包不补默认对象；Category 与 Operation 必填。
type ModalityTaskSubmitRequest struct {
	SessionID    string
	Category     string
	Operation    string
	Params       json.RawMessage
	Files        []ModalityTaskFileInput
	Model        *ModalityModelRef
	Node         string
	ClientTaskID string
}

// ModalityTaskQuery 是一次任务清单查询。零值表示不限定/不传该参数，本包不猜。
type ModalityTaskQuery struct {
	State    ModalityTaskState
	Category string
	Limit    int
	After    string
}

// ModalityTaskPage 是一页任务清单；Next 为空表示没有下一页。
type ModalityTaskPage struct {
	Tasks []ModalityTask
	Next  string
}

// ModalityTaskResultQuery 是一次任务结果分页读取。
type ModalityTaskResultQuery struct {
	TaskID string
	Name   string
	Offset int64
	Length int64
}

// ModalityTaskResultPage 是一页结果数据；EOF 为真表示已读完。
type ModalityTaskResultPage struct {
	DataBase64 string
	EOF        bool
}

// modalityTaskSubmitParams 是交给写登记的透明参数形状。
type modalityTaskSubmitParams struct {
	Op           string                  `json:"op"`
	SessionID    string                  `json:"sessionId"`
	Category     string                  `json:"category"`
	Operation    string                  `json:"operation"`
	Params       json.RawMessage         `json:"params,omitempty"`
	Files        []modalityTaskFileInput `json:"files,omitempty"`
	Model        *modalityModelRef       `json:"model,omitempty"`
	Node         string                  `json:"node,omitempty"`
	ClientTaskID string                  `json:"clientTaskId,omitempty"`
}

type modalityTaskFileInput struct {
	Name       string `json:"name"`
	MediaType  string `json:"mediaType"`
	DataBase64 string `json:"dataBase64,omitempty"`
	NodePath   string `json:"nodePath,omitempty"`
}

type modalityModelRef struct {
	ProviderID string `json:"providerId"`
	ModelID    string `json:"modelId"`
}

type modalityTaskCancelParams struct {
	Op        string `json:"op"`
	SessionID string `json:"sessionId"`
	TaskID    string `json:"taskId"`
}

// SubmitTask 提交一个模态任务。
//
// 提交与随后的状态读取都需要会话声明对应方法；缺任一方法即返回能力缺失，
// 不发出写请求。提交经写登记登记一次 KindModalityWrite（op=task.submit），
// 成功后用回执里的任务标识重读服务端任务，不做乐观更新。
func (m *Manager) SubmitTask(ctx context.Context, request ModalityTaskSubmitRequest) (ModalityTask, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID, err := normalizeSessionID(request.SessionID)
	if err != nil {
		return ModalityTask{}, err
	}
	category, ok := trimNonEmpty(request.Category)
	if !ok {
		return ModalityTask{}, fmt.Errorf("%w: 任务缺少 category", ErrInvalidModalityRequest)
	}
	operation, ok := trimNonEmpty(request.Operation)
	if !ok {
		return ModalityTask{}, fmt.Errorf("%w: 任务缺少 operation", ErrInvalidModalityRequest)
	}
	params, err := normalizeOptionalObject(request.Params)
	if err != nil {
		return ModalityTask{}, err
	}
	files, err := normalizeTaskFiles(request.Files)
	if err != nil {
		return ModalityTask{}, err
	}
	if err := validateModelRef(request.Model); err != nil {
		return ModalityTask{}, err
	}
	node, _ := trimNonEmpty(request.Node)
	clientTaskID, _ := trimNonEmpty(request.ClientTaskID)

	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	if err := m.requireModalityMethods(ctx, sessionID, ModalityMethodTaskSubmit, ModalityMethodTaskStatus); err != nil {
		return ModalityTask{}, err
	}
	key := m.modalityWriteKey("task.submit", sessionID, clientTaskID)
	view, err := m.registerModalityWrite(ctx, key, sessionID, "", modalityTaskSubmitParams{
		Op:           "task.submit",
		SessionID:    sessionID,
		Category:     category,
		Operation:    operation,
		Params:       params,
		Files:        files,
		Model:        toWireModelRef(request.Model),
		Node:         node,
		ClientTaskID: clientTaskID,
	})
	if err != nil {
		return ModalityTask{}, err
	}
	if view.Status != write.StatusSucceeded {
		return ModalityTask{}, fmt.Errorf("%w: %s", ErrTaskSubmitFailed, failureReason(view))
	}
	if strings.TrimSpace(view.InputID) == "" {
		return ModalityTask{}, fmt.Errorf("%w: 任务提交回执缺少任务标识", ErrMalformedWriteResult)
	}
	task, err := m.readTask(ctx, sessionID, strings.TrimSpace(view.InputID))
	if err != nil {
		return ModalityTask{}, fmt.Errorf("任务已提交（%s），但读取状态失败: %w", view.InputID, err)
	}
	return task, nil
}

// CancelTask 取消一个模态任务。它与提交共用能力门禁与写登记；成功后重读任务，
// 以服务端状态为准。
func (m *Manager) CancelTask(ctx context.Context, sessionID, taskID string) (ModalityTask, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID, err := normalizeSessionID(sessionID)
	if err != nil {
		return ModalityTask{}, err
	}
	taskID, ok := trimNonEmpty(taskID)
	if !ok {
		return ModalityTask{}, fmt.Errorf("%w: 任务缺少标识", ErrInvalidModalityRequest)
	}

	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	if err := m.requireModalityMethods(ctx, sessionID, ModalityMethodTaskCancel, ModalityMethodTaskStatus); err != nil {
		return ModalityTask{}, err
	}
	key := m.modalityWriteKey("task.cancel", sessionID, "")
	view, err := m.registerModalityWrite(ctx, key, sessionID, taskID, modalityTaskCancelParams{
		Op:        "task.cancel",
		SessionID: sessionID,
		TaskID:    taskID,
	})
	if err != nil {
		return ModalityTask{}, err
	}
	if view.Status != write.StatusSucceeded {
		return ModalityTask{}, fmt.Errorf("%w: %s", ErrTaskCancelFailed, failureReason(view))
	}
	task, err := m.readTask(ctx, sessionID, taskID)
	if err != nil {
		return ModalityTask{}, fmt.Errorf("任务已取消（%s），但读取状态失败: %w", taskID, err)
	}
	return task, nil
}

// ReadTask 读取一个任务的当前状态。它走会话读取，不经过写登记。
func (m *Manager) ReadTask(ctx context.Context, sessionID, taskID string) (ModalityTask, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID, err := normalizeSessionID(sessionID)
	if err != nil {
		return ModalityTask{}, err
	}
	taskID, ok := trimNonEmpty(taskID)
	if !ok {
		return ModalityTask{}, fmt.Errorf("%w: 任务缺少标识", ErrInvalidModalityRequest)
	}
	if err := m.requireModalityMethod(ctx, sessionID, ModalityMethodTaskStatus); err != nil {
		return ModalityTask{}, err
	}
	return m.readTask(ctx, sessionID, taskID)
}

// ListTasks 读取一页任务清单。Next 原样返回，调用方按游标继续。
func (m *Manager) ListTasks(ctx context.Context, sessionID string, query ModalityTaskQuery) (ModalityTaskPage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID, err := normalizeSessionID(sessionID)
	if err != nil {
		return ModalityTaskPage{}, err
	}
	normalized, err := normalizeTaskQuery(query)
	if err != nil {
		return ModalityTaskPage{}, err
	}
	if err := m.requireModalityMethod(ctx, sessionID, ModalityMethodTaskList); err != nil {
		return ModalityTaskPage{}, err
	}
	if m.cfg.ModalitySession == nil {
		return ModalityTaskPage{}, missing(ModalityMethodTaskList, "未提供会话模态调用面")
	}
	page, err := m.cfg.ModalitySession.ListModalityTasks(ctx, sessionID, normalized)
	if err != nil {
		return ModalityTaskPage{}, readFailure(ModalityMethodTaskList, ErrTaskListFailed, err)
	}
	for index := range page.Tasks {
		page.Tasks[index].SessionID = sessionID
	}
	if err := validateTaskPage(page); err != nil {
		return ModalityTaskPage{}, err
	}
	m.cacheTasks(page.Tasks)
	return cloneTaskPage(page), nil
}

// ReadTaskResult 读取一页任务结果。按 Offset 递增、Length 不超上限，直到 EOF。
func (m *Manager) ReadTaskResult(ctx context.Context, sessionID string, query ModalityTaskResultQuery) (ModalityTaskResultPage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID, err := normalizeSessionID(sessionID)
	if err != nil {
		return ModalityTaskResultPage{}, err
	}
	normalized, err := normalizeTaskResultQuery(query)
	if err != nil {
		return ModalityTaskResultPage{}, err
	}
	if err := m.requireModalityMethod(ctx, sessionID, ModalityMethodTaskResult); err != nil {
		return ModalityTaskResultPage{}, err
	}
	if m.cfg.ModalitySession == nil {
		return ModalityTaskResultPage{}, missing(ModalityMethodTaskResult, "未提供会话模态调用面")
	}
	page, err := m.cfg.ModalitySession.ReadModalityTaskResult(ctx, sessionID, normalized)
	if err != nil {
		return ModalityTaskResultPage{}, readFailure(ModalityMethodTaskResult, ErrTaskResultFailed, err)
	}
	if _, err := base64.StdEncoding.DecodeString(page.DataBase64); err != nil {
		return ModalityTaskResultPage{}, fmt.Errorf("%w: 任务结果不是合法 base64", ErrMalformedModality)
	}
	return page, nil
}

// Task 返回最近一次确认的任务；第二个返回值表示是否有缓存。
func (m *Manager) Task(sessionID, taskID string) (ModalityTask, bool) {
	sessionID, ok := trimNonEmpty(sessionID)
	if !ok {
		return ModalityTask{}, false
	}
	taskID, ok = trimNonEmpty(taskID)
	if !ok {
		return ModalityTask{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	task, found := m.tasks[taskCacheKey(sessionID, taskID)]
	if !found {
		return ModalityTask{}, false
	}
	return cloneTask(task), true
}

// readTask 只做会话读取、校验与缓存；调用前已通过方法门禁。
func (m *Manager) readTask(ctx context.Context, sessionID, taskID string) (ModalityTask, error) {
	if m.cfg.ModalitySession == nil {
		return ModalityTask{}, missing(ModalityMethodTaskStatus, "未提供会话模态调用面")
	}
	task, err := m.cfg.ModalitySession.ReadModalityTask(ctx, sessionID, taskID)
	if err != nil {
		return ModalityTask{}, readFailure(ModalityMethodTaskStatus, ErrTaskReadFailed, err)
	}
	task.SessionID = sessionID
	if err := validateTask(task); err != nil {
		return ModalityTask{}, err
	}
	m.cacheTasks([]ModalityTask{task})
	return cloneTask(task), nil
}

func (m *Manager) cacheTasks(tasks []ModalityTask) {
	if len(tasks) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, task := range tasks {
		if task.ID == "" {
			continue
		}
		sessionID, ok := trimNonEmpty(task.SessionID)
		if !ok {
			continue
		}
		m.tasks[taskCacheKey(sessionID, task.ID)] = cloneTask(task)
	}
}

func taskCacheKey(sessionID, taskID string) string {
	return sessionID + "\x00" + taskID
}

func normalizeTaskFiles(files []ModalityTaskFileInput) ([]modalityTaskFileInput, error) {
	if len(files) == 0 {
		return nil, nil
	}
	normalized := make([]modalityTaskFileInput, 0, len(files))
	for index, file := range files {
		name, ok := trimNonEmpty(file.Name)
		if !ok {
			return nil, fmt.Errorf("%w: 第 %d 个输入文件缺少名称", ErrInvalidModalityRequest, index)
		}
		mediaType, ok := trimNonEmpty(file.MediaType)
		if !ok {
			return nil, fmt.Errorf("%w: 第 %d 个输入文件缺少媒体类型", ErrInvalidModalityRequest, index)
		}
		data, hasData := trimNonEmpty(file.DataBase64)
		nodePath, hasNodePath := trimNonEmpty(file.NodePath)
		if hasData == hasNodePath {
			return nil, fmt.Errorf("%w: 第 %d 个输入文件必须且只能给 dataBase64 或 nodePath", ErrInvalidModalityRequest, index)
		}
		if hasData {
			if _, err := base64.StdEncoding.DecodeString(data); err != nil {
				return nil, fmt.Errorf("%w: 第 %d 个输入文件不是合法 base64", ErrInvalidModalityRequest, index)
			}
		}
		normalized = append(normalized, modalityTaskFileInput{
			Name:       name,
			MediaType:  mediaType,
			DataBase64: data,
			NodePath:   nodePath,
		})
	}
	return normalized, nil
}

func toWireModelRef(model *ModalityModelRef) *modalityModelRef {
	if model == nil {
		return nil
	}
	return &modalityModelRef{
		ProviderID: strings.TrimSpace(model.ProviderID),
		ModelID:    strings.TrimSpace(model.ModelID),
	}
}

func normalizeTaskQuery(query ModalityTaskQuery) (ModalityTaskQuery, error) {
	if query.State != "" {
		switch query.State {
		case ModalityTaskQueued, ModalityTaskRunning, ModalityTaskSucceeded,
			ModalityTaskFailed, ModalityTaskCancelled, ModalityTaskInterrupted:
		default:
			return ModalityTaskQuery{}, fmt.Errorf("%w: 未知任务状态 %q", ErrInvalidModalityRequest, query.State)
		}
	}
	if query.Limit < 0 || query.Limit > 100 {
		return ModalityTaskQuery{}, fmt.Errorf("%w: 任务清单上限 %d 超出范围", ErrInvalidModalityRequest, query.Limit)
	}
	query.Category, _ = trimNonEmpty(query.Category)
	query.After, _ = trimNonEmpty(query.After)
	return query, nil
}

func normalizeTaskResultQuery(query ModalityTaskResultQuery) (ModalityTaskResultQuery, error) {
	taskID, ok := trimNonEmpty(query.TaskID)
	if !ok {
		return ModalityTaskResultQuery{}, fmt.Errorf("%w: 结果读取缺少任务标识", ErrInvalidModalityRequest)
	}
	name, ok := trimNonEmpty(query.Name)
	if !ok {
		return ModalityTaskResultQuery{}, fmt.Errorf("%w: 结果读取缺少文件名称", ErrInvalidModalityRequest)
	}
	if query.Offset < 0 {
		return ModalityTaskResultQuery{}, fmt.Errorf("%w: 结果偏移为负 %d", ErrInvalidModalityRequest, query.Offset)
	}
	if query.Length <= 0 || query.Length > ModalityTaskResultMaxBytes {
		return ModalityTaskResultQuery{}, fmt.Errorf(
			"%w: 结果长度 %d 超出 1..%d", ErrInvalidModalityRequest, query.Length, ModalityTaskResultMaxBytes,
		)
	}
	query.TaskID = taskID
	query.Name = name
	return query, nil
}

// validateTask 校验任务记录的必要字段。缺字段或未知状态按事实不合法拒绝。
func validateTask(task ModalityTask) error {
	if _, ok := trimNonEmpty(task.ID); !ok {
		return fmt.Errorf("%w: 任务缺少标识", ErrMalformedModality)
	}
	if _, ok := trimNonEmpty(task.Category); !ok {
		return fmt.Errorf("%w: 任务 %q 缺少 category", ErrMalformedModality, task.ID)
	}
	if _, ok := trimNonEmpty(task.Operation); !ok {
		return fmt.Errorf("%w: 任务 %q 缺少 operation", ErrMalformedModality, task.ID)
	}
	if task.Revision < 0 {
		return fmt.Errorf("%w: 任务 %q 的修订号为负", ErrMalformedModality, task.ID)
	}
	switch task.State {
	case ModalityTaskQueued, ModalityTaskRunning, ModalityTaskSucceeded,
		ModalityTaskFailed, ModalityTaskCancelled, ModalityTaskInterrupted:
	default:
		return fmt.Errorf("%w: 任务 %q 的状态 %q 未知", ErrMalformedModality, task.ID, task.State)
	}
	if _, ok := trimNonEmpty(task.CreatedAt); !ok {
		return fmt.Errorf("%w: 任务 %q 缺少创建时间", ErrMalformedModality, task.ID)
	}
	if _, ok := trimNonEmpty(task.UpdatedAt); !ok {
		return fmt.Errorf("%w: 任务 %q 缺少更新时间", ErrMalformedModality, task.ID)
	}
	if task.Progress != nil && task.Progress.Fraction != nil {
		if *task.Progress.Fraction < 0 || *task.Progress.Fraction > 1 {
			return fmt.Errorf("%w: 任务 %q 的进度超出 0..1", ErrMalformedModality, task.ID)
		}
	}
	if task.Result != nil {
		if len(task.Result.Data) > 0 && !json.Valid(task.Result.Data) {
			return fmt.Errorf("%w: 任务 %q 的结果数据不是合法 JSON", ErrMalformedModality, task.ID)
		}
		for index, file := range task.Result.Files {
			if _, ok := trimNonEmpty(file.Name); !ok {
				return fmt.Errorf("%w: 任务 %q 的第 %d 个结果文件缺少名称", ErrMalformedModality, task.ID, index)
			}
			if _, ok := trimNonEmpty(file.MediaType); !ok {
				return fmt.Errorf("%w: 任务 %q 的第 %d 个结果文件缺少媒体类型", ErrMalformedModality, task.ID, index)
			}
			if _, ok := trimNonEmpty(file.SHA256); !ok {
				return fmt.Errorf("%w: 任务 %q 的第 %d 个结果文件缺少摘要", ErrMalformedModality, task.ID, index)
			}
			if file.Bytes < 0 {
				return fmt.Errorf("%w: 任务 %q 的第 %d 个结果文件字节为负", ErrMalformedModality, task.ID, index)
			}
		}
	}
	if task.Error != nil {
		if _, ok := trimNonEmpty(task.Error.Code); !ok {
			return fmt.Errorf("%w: 任务 %q 的错误缺少代码", ErrMalformedModality, task.ID)
		}
		if _, ok := trimNonEmpty(task.Error.Message); !ok {
			return fmt.Errorf("%w: 任务 %q 的错误缺少说明", ErrMalformedModality, task.ID)
		}
	}
	return nil
}

func validateTaskPage(page ModalityTaskPage) error {
	for _, task := range page.Tasks {
		if err := validateTask(task); err != nil {
			return err
		}
	}
	if page.Next != "" {
		if _, ok := trimNonEmpty(page.Next); !ok {
			return fmt.Errorf("%w: 任务清单游标为空", ErrMalformedModality)
		}
	}
	return nil
}

func cloneTask(task ModalityTask) ModalityTask {
	cloned := task
	if task.Progress != nil {
		progress := *task.Progress
		if task.Progress.Fraction != nil {
			fraction := *task.Progress.Fraction
			progress.Fraction = &fraction
		}
		cloned.Progress = &progress
	}
	if task.Result != nil {
		result := ModalityTaskResultData{
			Files: append([]ModalityTaskFile(nil), task.Result.Files...),
			Data:  cloneRaw(task.Result.Data),
		}
		cloned.Result = &result
	}
	if task.Error != nil {
		errorValue := *task.Error
		cloned.Error = &errorValue
	}
	return cloned
}

func cloneTaskPage(page ModalityTaskPage) ModalityTaskPage {
	cloned := ModalityTaskPage{Next: page.Next, Tasks: make([]ModalityTask, 0, len(page.Tasks))}
	for _, task := range page.Tasks {
		cloned.Tasks = append(cloned.Tasks, cloneTask(task))
	}
	return cloned
}
