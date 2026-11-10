package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// ModalityOp 是模态写入的操作名，取值与写登记的参数一致。
type ModalityOp string

const (
	// ModalityTaskSubmit 提交一次模态任务。
	ModalityTaskSubmit ModalityOp = "task.submit"
	// ModalityTaskCancel 取消一次模态任务。
	ModalityTaskCancel ModalityOp = "task.cancel"
	// ModalityRealtimeOpen 打开实时传输。
	ModalityRealtimeOpen ModalityOp = "realtime.open"
	// ModalityRealtimeClose 关闭实时传输。
	ModalityRealtimeClose ModalityOp = "realtime.close"
)

// ShellRunRequest 是一次会话内命令执行（`!` 命令由服务端执行）。
type ShellRunRequest struct {
	// Key 是幂等键；空时按会话、分支与一次随机标识生成。
	Key string
	// SessionID 是目标会话，必填。
	SessionID string
	// BranchID 是目标分支；空按主线。
	BranchID string
	// Command 是要执行的命令，必填。
	Command string
	// RunID 是客户端给出的运行标识；空时不带，服务端自行生成。
	RunID string
}

// ShellStopRequest 是一次会话内命令停止。
type ShellStopRequest struct {
	// Key 是幂等键；空时按运行标识派生。
	Key string
	// SessionID 是目标会话，必填。
	SessionID string
	// BranchID 是目标分支；空按主线。
	BranchID string
	// RunID 是要停止的运行标识，必填。
	RunID string
}

// ModalityWriteRequest 是一次模态写入（任务提交/取消、实时传输打开/关闭）。
type ModalityWriteRequest struct {
	// Key 是幂等键；空时按会话与操作生成。
	Key string
	// SessionID 是目标会话，必填。
	SessionID string
	// BranchID 是目标分支；空按主线。
	BranchID string
	// Op 是操作名，必须是四个取值之一。
	Op ModalityOp
	// Params 是该操作自己的载荷，原样下发（例如 taskId、模型引用）。
	Params map[string]any
	// InputID 供结果未知时核对；空时由写登记生成。
	InputID string
}

// RunShell 登记一次会话内命令执行意图。写闸门未放行时直接拒绝，不登记。
func (a *App) RunShell(ctx context.Context, request ShellRunRequest) (write.IntentView, error) {
	if strings.TrimSpace(request.SessionID) == "" {
		return write.IntentView{}, fmt.Errorf("%w: 缺少会话标识", ErrInvalidConfig)
	}
	if strings.TrimSpace(request.Command) == "" {
		return write.IntentView{}, fmt.Errorf("%w: 命令为空", write.ErrInvalidIntent)
	}
	if err := a.Writable(); err != nil {
		return write.IntentView{}, err
	}
	writes := a.Writes()
	if writes == nil {
		return write.IntentView{}, ErrNotConnected
	}
	params := map[string]any{"op": "run", "command": request.Command}
	if request.RunID != "" {
		params["runId"] = request.RunID
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return write.IntentView{}, fmt.Errorf("编码命令执行参数: %w", err)
	}
	key := request.Key
	if key == "" {
		id, err := uuidV4()
		if err != nil {
			return write.IntentView{}, fmt.Errorf("生成命令执行标识: %w", err)
		}
		key = "shell.run:" + request.SessionID + ":" + request.BranchID + ":" + id
	}
	return writes.Register(write.Intent{
		Key:       key,
		Kind:      write.KindShellRun,
		SessionID: request.SessionID,
		BranchID:  request.BranchID,
		InputID:   request.RunID,
		Params:    raw,
	})
}

// StopShell 登记一次会话内命令停止意图。写闸门未放行时直接拒绝，不登记。
func (a *App) StopShell(ctx context.Context, request ShellStopRequest) (write.IntentView, error) {
	if strings.TrimSpace(request.SessionID) == "" {
		return write.IntentView{}, fmt.Errorf("%w: 缺少会话标识", ErrInvalidConfig)
	}
	if strings.TrimSpace(request.RunID) == "" {
		return write.IntentView{}, fmt.Errorf("%w: 缺少运行标识", write.ErrInvalidIntent)
	}
	if err := a.Writable(); err != nil {
		return write.IntentView{}, err
	}
	writes := a.Writes()
	if writes == nil {
		return write.IntentView{}, ErrNotConnected
	}
	raw, err := json.Marshal(map[string]any{"op": "stop", "runId": request.RunID})
	if err != nil {
		return write.IntentView{}, fmt.Errorf("编码命令停止参数: %w", err)
	}
	key := request.Key
	if key == "" {
		key = "shell.stop:" + request.SessionID + ":" + request.RunID
	}
	return writes.Register(write.Intent{
		Key:       key,
		Kind:      write.KindShellRun,
		SessionID: request.SessionID,
		BranchID:  request.BranchID,
		InputID:   request.RunID,
		Params:    raw,
	})
}

// ModalityWrite 登记一次模态写入意图。写闸门未放行时直接拒绝，不登记。
func (a *App) ModalityWrite(ctx context.Context, request ModalityWriteRequest) (write.IntentView, error) {
	if strings.TrimSpace(request.SessionID) == "" {
		return write.IntentView{}, fmt.Errorf("%w: 缺少会话标识", ErrInvalidConfig)
	}
	switch request.Op {
	case ModalityTaskSubmit, ModalityTaskCancel, ModalityRealtimeOpen, ModalityRealtimeClose:
	default:
		return write.IntentView{}, fmt.Errorf("%w: 未知模态操作 %q", write.ErrInvalidIntent, request.Op)
	}
	if err := a.Writable(); err != nil {
		return write.IntentView{}, err
	}
	writes := a.Writes()
	if writes == nil {
		return write.IntentView{}, ErrNotConnected
	}
	params := make(map[string]any, len(request.Params)+1)
	for key, value := range request.Params {
		params[key] = value
	}
	params["op"] = string(request.Op)
	raw, err := json.Marshal(params)
	if err != nil {
		return write.IntentView{}, fmt.Errorf("编码模态写入参数: %w", err)
	}
	key := request.Key
	if key == "" {
		key = "modality:" + request.SessionID + ":" + string(request.Op)
	}
	return writes.Register(write.Intent{
		Key:       key,
		Kind:      write.KindModalityWrite,
		SessionID: request.SessionID,
		BranchID:  request.BranchID,
		InputID:   request.InputID,
		Params:    raw,
	})
}
