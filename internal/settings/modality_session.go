package settings

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// 会话贡献的模态方法名。能力门禁按这些名字查询会话方法表；集成层负责把它们
// 映射到协议常量，本包不定义 wire 结构。
const (
	ModalityMethodTaskSubmit    = "modality.task.submit"
	ModalityMethodTaskStatus    = "modality.task.status"
	ModalityMethodTaskList      = "modality.task.list"
	ModalityMethodTaskCancel    = "modality.task.cancel"
	ModalityMethodTaskResult    = "modality.task.result"
	ModalityMethodRealtimeOpen  = "modality.realtime.open"
	ModalityMethodRealtimeSend  = "modality.realtime.send"
	ModalityMethodRealtimeClose = "modality.realtime.close"
)

// ModalitySession 是集成层提供的模态调用面：能力门禁、任务读取与实时帧发送。
// 实现方按服务端实际路由调用方法并映射回执；本机真实服务端把 modality.* 作为
// daemon 级方法声明在 hello.features，不贡献 session 方法，因此门禁必须接受
// hello 能力。实现方不解释业务、不重试、不缓存；任务提交、取消与实时打开、
// 关闭不走本接口，它们经写登记转达。
type ModalitySession interface {
	// SupportsModalityMethod 回答该会话是否声明了某个方法。实现方应先看 hello
	// 能力，再看会话方法表；两者都不能确认时按缺失处理。本机真实服务端只在
	// hello.features 声明 modality.*，会话方法表不含这些方法。
	SupportsModalityMethod(ctx context.Context, sessionID, method string) (bool, error)
	ReadModalityTask(ctx context.Context, sessionID, taskID string) (ModalityTask, error)
	ListModalityTasks(ctx context.Context, sessionID string, query ModalityTaskQuery) (ModalityTaskPage, error)
	ReadModalityTaskResult(ctx context.Context, sessionID string, query ModalityTaskResultQuery) (ModalityTaskResultPage, error)
	SendModalityRealtime(ctx context.Context, sessionID, connectionID string, frame ModalityRealtimeFrame) error
}

// requireModalityMethod 在发出任何会话请求或写登记之前检查方法表。缺失或读取
// 失败都返回能力缺失，并且不发请求。
func (m *Manager) requireModalityMethod(ctx context.Context, sessionID, method string) error {
	if m.cfg.ModalitySession == nil {
		return missing(method, "未提供会话模态调用面")
	}
	supported, err := m.cfg.ModalitySession.SupportsModalityMethod(ctx, sessionID, method)
	if err != nil {
		return missing(method, "读取会话方法表失败: "+err.Error())
	}
	if !supported {
		return missing(method, "会话未声明该方法")
	}
	return nil
}

// requireModalityMethods 依次检查多个方法，全部存在才继续。
func (m *Manager) requireModalityMethods(ctx context.Context, sessionID string, methods ...string) error {
	for _, method := range methods {
		if err := m.requireModalityMethod(ctx, sessionID, method); err != nil {
			return err
		}
	}
	return nil
}

// normalizeSessionID 校验会话标识；模态任务与实时传输都按会话隔离。
func normalizeSessionID(sessionID string) (string, error) {
	trimmed, ok := trimNonEmpty(sessionID)
	if !ok {
		return "", fmt.Errorf("%w: 缺少会话标识", ErrInvalidModalityRequest)
	}
	return trimmed, nil
}

// normalizeOptionalObject 校验一段可选的 JSON 对象。空内容表示调用方没有提供，
// 本包不补默认对象；非空内容必须是对象。
func normalizeOptionalObject(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}
	if trimmed[0] != '{' {
		return nil, fmt.Errorf("%w: 参数必须是 JSON 对象", ErrInvalidModalityRequest)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &object); err != nil {
		return nil, fmt.Errorf("%w: 参数无法解析: %v", ErrInvalidModalityRequest, err)
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err != nil {
		return nil, fmt.Errorf("%w: 参数无法解析: %v", ErrInvalidModalityRequest, err)
	}
	return append(json.RawMessage(nil), compacted.Bytes()...), nil
}

// validateModelRef 校验一次模型引用。两个标识都必须由调用方给出，本包不猜。
func validateModelRef(model *ModalityModelRef) error {
	if model == nil {
		return nil
	}
	if _, ok := trimNonEmpty(model.ProviderID); !ok {
		return fmt.Errorf("%w: 模型引用缺少 provider 标识", ErrInvalidModalityRequest)
	}
	if _, ok := trimNonEmpty(model.ModelID); !ok {
		return fmt.Errorf("%w: 模型引用缺少模型标识", ErrInvalidModalityRequest)
	}
	return nil
}

// normalizeFrameKind 校验实时帧类型。
func normalizeFrameKind(kind string) (string, error) {
	trimmed, ok := trimNonEmpty(kind)
	if !ok {
		return "", fmt.Errorf("%w: 实时帧缺少类型", ErrInvalidModalityRequest)
	}
	switch trimmed {
	case "text", "audio", "control":
		return trimmed, nil
	default:
		return "", fmt.Errorf("%w: 未知实时帧类型 %q", ErrInvalidModalityRequest, kind)
	}
}

// normalizeFrameData 校验帧数据非空且是合法 JSON；音频帧必须是 JSON 字符串。
func normalizeFrameData(kind string, raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("%w: 实时帧缺少数据", ErrInvalidModalityRequest)
	}
	if !json.Valid(trimmed) {
		return nil, fmt.Errorf("%w: 实时帧数据不是合法 JSON", ErrInvalidModalityRequest)
	}
	if kind == "audio" {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil || text == "" {
			return nil, fmt.Errorf("%w: 音频帧数据必须是 base64 字符串", ErrInvalidModalityRequest)
		}
		if _, err := base64.StdEncoding.DecodeString(text); err != nil {
			return nil, fmt.Errorf("%w: 音频帧数据不是合法 base64", ErrInvalidModalityRequest)
		}
	}
	return append(json.RawMessage(nil), trimmed...), nil
}

// modalityWriteKey 生成写意图的幂等键。有调用方标识时按该标识去重；否则按
// 进程内序号生成唯一键，保证每次调用只登记一条。
func (m *Manager) modalityWriteKey(op, sessionID, identity string) string {
	if identity != "" {
		return "modality:" + op + ":" + sessionID + ":" + identity
	}
	m.mu.Lock()
	m.writeSeq++
	seq := m.writeSeq
	m.mu.Unlock()
	return fmt.Sprintf("modality:%s:%d", op, seq)
}

// registerModalityWrite 登记一条模态写入并等待终态。它只登记一次，不重试。
func (m *Manager) registerModalityWrite(ctx context.Context, key, sessionID, inputID string, params any) (writeView, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return writeView{}, fmt.Errorf("编码模态写入参数: %w", err)
	}
	writer := m.writer()
	if writer == nil {
		return writeView{}, fmt.Errorf("%w: %s", ErrWriteUnavailable, "未提供写登记")
	}
	view, err := writer.Register(write.Intent{
		Key:       key,
		Kind:      write.KindModalityWrite,
		SessionID: sessionID,
		InputID:   inputID,
		Params:    raw,
	})
	if err != nil {
		return writeView{}, fmt.Errorf("登记模态写入: %w", err)
	}
	view, err = writer.Wait(ctx, view.ID)
	if err != nil && view.Status == "" {
		return writeView{}, err
	}
	return view, nil
}

// writeView 是写登记视图的别名，避免在多个文件里重复长类型名。
type writeView = write.IntentView

// failureReason 取写登记给出的原因；没有时退回状态文本。
func failureReason(view writeView) string {
	if strings.TrimSpace(view.Reason) != "" {
		return view.Reason
	}
	return string(view.Status)
}
