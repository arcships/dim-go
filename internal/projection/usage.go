package projection

import (
	"math"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// 上下文与用量：只折叠服务端给出的事实，算的是事实的累计与最新值，不推导服务端
// 没给的数字。

// Usage 是会话累计用量，按 `model/usage` 逐条累加。
type Usage struct {
	// Requests 是累加了用量的模型请求数。
	Requests         int64   `json:"requests"`
	PromptTokens     int64   `json:"promptTokens"`
	CompletionTokens int64   `json:"completionTokens"`
	TotalTokens      int64   `json:"totalTokens"`
	CacheReadTokens  int64   `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens int64   `json:"cacheWriteTokens,omitempty"`
	Cost             float64 `json:"cost,omitempty"`
}

// ContextUsage 是最近一次模型调用的上下文占用，来自 `context/budget`。
type ContextUsage struct {
	// EstimatedInputTokens 是服务端估算的本次输入 token 数。
	EstimatedInputTokens int64 `json:"estimatedInputTokens"`
	// ContextWindow 是模型的上下文窗口。
	ContextWindow int64 `json:"contextWindow"`
	// OutputTokens 是为输出预留的 token 数。
	OutputTokens int64 `json:"outputTokens"`
	// BaselinePromptTokens 与 DeltaTokens 是服务端给出的基线与其增量；
	// 服务端没给就是 0。
	BaselinePromptTokens int64 `json:"baselinePromptTokens,omitempty"`
	DeltaTokens          int64 `json:"deltaTokens,omitempty"`
	// Sequence 是这条事实的序号。
	Sequence int64 `json:"sequence"`
}

// Remaining 是估算还可用的输入 token 数；窗口未知时为 0。
func (c ContextUsage) Remaining() int64 {
	remaining := c.ContextWindow - c.EstimatedInputTokens
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Percent 是占用比例（0–100）；窗口未知时为 0。
func (c ContextUsage) Percent() float64 {
	if c.ContextWindow <= 0 {
		return 0
	}
	percent := float64(c.EstimatedInputTokens) / float64(c.ContextWindow) * 100
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}

// Compaction 是一次上下文压缩的只读摘要。
type Compaction struct {
	// Sequence 是压缩事实的序号。
	Sequence int64 `json:"sequence"`
	// Status 取 completed 或 cancelled。
	Status string `json:"status"`
	// Summary 是压缩后的摘要正文；取消的压缩没有摘要。
	Summary string `json:"summary,omitempty"`
	// Strategy 是压缩策略名，服务端没给就是空。
	Strategy string `json:"strategy,omitempty"`
	// RetainFrom 是保留边界的事实标识，服务端没给就是空。
	RetainFrom string `json:"retainFrom,omitempty"`
	// At 是压缩事实的时间。
	At int64 `json:"at"`
}

// usageState 是折叠里的用量累计。
type usageState struct {
	usage Usage
	seen  bool
}

// contextState 是折叠里最新的上下文占用。
type contextState struct {
	usage ContextUsage
	seen  bool
}

// compactionState 是折叠里最近一次压缩与压缩次数。
type compactionState struct {
	latest Compaction
	count  int64
}

// applyUsage 累加一条 `model/usage`。
func (s *foldState) applyUsage(data map[string]any) {
	usage := object(data["usage"])
	if len(usage) == 0 {
		return
	}
	state := &s.usage
	state.usage.Requests++
	state.usage.PromptTokens += roundedInt64(usage["promptTokens"])
	state.usage.CompletionTokens += roundedInt64(usage["completionTokens"])
	state.usage.TotalTokens += roundedInt64(usage["totalTokens"])
	state.usage.CacheReadTokens += roundedInt64(usage["cacheReadTokens"])
	state.usage.CacheWriteTokens += roundedInt64(usage["cacheWriteTokens"])
	state.usage.Cost += roundedFloat(usage["cost"])
	state.seen = true
}

// applyBudget 记录一次 `context/budget`：后来的覆盖先来的。
func (s *foldState) applyBudget(event protocol.RawEvent, data map[string]any) {
	s.context.usage = ContextUsage{
		EstimatedInputTokens: roundedInt64(data["estimatedInputTokens"]),
		ContextWindow:        roundedInt64(data["contextWindow"]),
		OutputTokens:         roundedInt64(data["outputTokens"]),
		BaselinePromptTokens: roundedInt64(data["baselinePromptTokens"]),
		DeltaTokens:          roundedInt64(data["deltaTokens"]),
		Sequence:             event.Sequence,
	}
	s.context.seen = true
}

// applyCompactionRecord 记录一次压缩事实（系统提示行仍由 applyCompaction 建）。
func (s *foldState) applyCompactionRecord(event protocol.RawEvent, data map[string]any) {
	status := firstString(data, "status")
	s.compaction.latest = Compaction{
		Sequence:   event.Sequence,
		Status:     status,
		Summary:    compactionSummary(data),
		Strategy:   firstString(data, "strategy"),
		RetainFrom: firstString(data, "retain_from", "retainFrom"),
		At:         event.CreateAt,
	}
	s.compaction.count++
}

// usageView 输出用量；没有事实时返回 nil。
func (s *foldState) usageView() *Usage {
	if !s.usage.seen {
		return nil
	}
	usage := s.usage.usage
	return &usage
}

// contextView 输出上下文占用；没有事实时返回 nil。
func (s *foldState) contextView() *ContextUsage {
	if !s.context.seen {
		return nil
	}
	usage := s.context.usage
	return &usage
}

// compactionView 输出最近一次压缩；没有事实时返回 nil。
func (s *foldState) compactionView() *Compaction {
	if s.compaction.count == 0 {
		return nil
	}
	compaction := s.compaction.latest
	return &compaction
}

// roundedInt64 把 JSON 数值四舍五入成 int64；缺项按 0 算。
func roundedInt64(value any) int64 {
	if number, ok := value.(float64); ok {
		return int64(math.Round(number))
	}
	return toInt64(value)
}

// roundedFloat 把 JSON 数值读成 float64；缺项按 0 算。
func roundedFloat(value any) float64 {
	if number, ok := value.(float64); ok {
		return number
	}
	return 0
}

// ContextReading 是服务端按需读到的上下文占用（会话方法 `session.context.usage`）。
//
// 协议 v3 不把 `context/budget` 事实发给客户端（服务端的事件可见性白名单把它排除），
// 所以客户端拿到的占用来自这个读取方法。集成层负责发请求，本包负责换算与判断。
type ContextReading struct {
	// Through 是这次读数覆盖到的序号。
	Through int64 `json:"through"`
	// EstimatedInputTokens 是服务端估算的输入 token 数。
	EstimatedInputTokens int64 `json:"estimatedInputTokens"`
	// ContextWindow 是模型的上下文窗口。
	ContextWindow int64 `json:"contextWindow"`
	// OutputTokens 是为输出预留的 token 数。
	OutputTokens int64 `json:"outputTokens"`
	// ThresholdTokens 是自动压缩的阈值。
	ThresholdTokens int64 `json:"thresholdTokens"`
	// BaselineSource 是估算基线的来源。
	BaselineSource string `json:"baselineSource,omitempty"`
}

// Remaining 是估算还可用的输入 token 数。
func (c ContextReading) Remaining() int64 {
	remaining := c.ContextWindow - c.EstimatedInputTokens
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Percent 是占用比例（0–100）；窗口未知时为 0。
func (c ContextReading) Percent() float64 {
	if c.ContextWindow <= 0 {
		return 0
	}
	percent := float64(c.EstimatedInputTokens) / float64(c.ContextWindow) * 100
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}

// OverThreshold 报告估算输入是否已达到自动压缩的阈值。
func (c ContextReading) OverThreshold() bool {
	return c.ThresholdTokens > 0 && float64(c.EstimatedInputTokens) >= float64(c.ThresholdTokens)
}
