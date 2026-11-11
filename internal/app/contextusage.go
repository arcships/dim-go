package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// 上下文占用与手动压缩的真实方法（alpha.21 实测，见 D2 的调查结论）。
const (
	methodContextUsage   = "session.context.usage"
	methodContextCompact = "session.context.compact"
)

// contextReadingCache 是会话最近一次按需读到的上下文占用。
type contextReadingCache struct {
	mu      sync.Mutex
	reading projection.ContextReading
	known   bool
}

func (c *contextReadingCache) store(reading projection.ContextReading) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reading = reading
	c.known = true
}

func (c *contextReadingCache) load() (projection.ContextReading, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reading, c.known
}

// ReadContextUsage 读一次上下文占用（`session.context.usage`，会话流）。
//
// 协议 v3 不把 `context/budget` 事实发给客户端，这是等价读法。门禁先看 hello，
// 再看会话方法表；缺方法零请求。读数按会话缓存，只读不落盘。
func (a *App) ReadContextUsage(ctx context.Context, sessionID string) (projection.ContextReading, error) {
	runtime, err := a.openRuntime(sessionID, methodContextUsage)
	if err != nil {
		return projection.ContextReading{}, err
	}
	if !a.sessionSupportsMethod(ctx, runtime, methodContextUsage) {
		return projection.ContextReading{}, capabilityMissing(methodContextUsage)
	}
	raw, err := runtime.lease.Subscription().Call(ctx, methodContextUsage, map[string]any{
		"sessionId": sessionID,
		"branchId":  branchOrMain(runtime.folder.Selected()),
	})
	if err != nil {
		return projection.ContextReading{}, err
	}
	reading, err := parseContextReading(raw)
	if err != nil {
		return projection.ContextReading{}, err
	}
	runtime.contextReading.store(reading)
	a.record("上下文占用: 窗口=%d 输出预留=%d 阈值=%d 估算输入=%d 基线=%s",
		reading.ContextWindow, reading.OutputTokens, reading.ThresholdTokens,
		reading.EstimatedInputTokens, reading.BaselineSource)
	return reading, nil
}

// ContextReading 返回本会话最近一次读到的上下文占用；没有读过时为 false。
func (a *App) ContextReading(sessionID string) (projection.ContextReading, bool) {
	runtime, ok := a.session(sessionID)
	if !ok {
		return projection.ContextReading{}, false
	}
	return runtime.contextReading.load()
}

// parseContextReading 把回执映射成投影领域的读数；缺省字段按 0 处理。
func parseContextReading(raw json.RawMessage) (projection.ContextReading, error) {
	var reply struct {
		Through              *int64  `json:"through"`
		EstimatedInputTokens float64 `json:"estimatedInputTokens"`
		ContextWindow        float64 `json:"contextWindow"`
		OutputTokens         float64 `json:"outputTokens"`
		ThresholdTokens      float64 `json:"thresholdTokens"`
		BaselineSource       string  `json:"baselineSource"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return projection.ContextReading{}, fmt.Errorf("上下文占用回执无法解析: %w", err)
	}
	reading := projection.ContextReading{
		EstimatedInputTokens: int64(reply.EstimatedInputTokens),
		ContextWindow:        int64(reply.ContextWindow),
		OutputTokens:         int64(reply.OutputTokens),
		ThresholdTokens:      int64(reply.ThresholdTokens),
		BaselineSource:       reply.BaselineSource,
	}
	if reply.Through != nil {
		reading.Through = *reply.Through
	}
	return reading, nil
}

// CompactContext 触发一次手动压缩：登记一次 KindManagementWrite，再由发送方
// 转到 `session.context.compact`。压缩结果以时间线事实 `context/compaction` 为准，
// 由投影折叠。
func (a *App) CompactContext(ctx context.Context, sessionID string) (write.IntentView, error) {
	if err := a.Writable(); err != nil {
		return write.IntentView{}, err
	}
	runtime, err := a.openRuntime(sessionID, methodContextCompact)
	if err != nil {
		return write.IntentView{}, err
	}
	// 能力门禁放在入口：缺方法时零请求、不登记。
	if !a.sessionSupportsMethod(ctx, runtime, methodContextCompact) {
		return write.IntentView{}, capabilityMissing(methodContextCompact)
	}
	writes := a.Writes()
	if writes == nil {
		return write.IntentView{}, ErrNotConnected
	}
	branchID := branchOrMain(runtime.folder.Selected())
	params, err := json.Marshal(map[string]any{
		"op":        "session.context.compact",
		"sessionId": sessionID,
		"branchId":  branchID,
	})
	if err != nil {
		return write.IntentView{}, fmt.Errorf("编码压缩参数: %w", err)
	}
	return writes.Register(write.Intent{
		Key:       "context.compact:" + sessionID + ":" + branchID + ":" + time.Now().Format(time.RFC3339Nano),
		Kind:      write.KindManagementWrite,
		SessionID: sessionID,
		BranchID:  branchID,
		Params:    params,
	})
}

// summarizeCompactReply 只取回执里的状态与序号，供诊断与证据使用。
func summarizeCompactReply(raw json.RawMessage) string {
	var reply struct {
		Sequence int64  `json:"sequence"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return "回执无法解析"
	}
	return fmt.Sprintf("status=%s sequence=%d", reply.Status, reply.Sequence)
}

// 保证协议类型引用成立。
var _ = protocol.ProtocolVersion
