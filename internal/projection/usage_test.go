package projection

import (
	"testing"
)

// TestUsageAccumulates 验证用量按 model/usage 累加。
func TestUsageAccumulates(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(2,
		ev("u1", 1, "model", "usage", `{"turn_id":"t1","request_id":"r1","usage":{"promptTokens":100,"completionTokens":20,"totalTokens":120,"cacheReadTokens":30}}`),
		ev("u2", 2, "model", "usage", `{"turn_id":"t1","request_id":"r2","usage":{"promptTokens":50,"completionTokens":10,"totalTokens":60,"cacheWriteTokens":5,"cost":0.25}}`),
	))
	usage := folder.View().Usage
	if usage == nil {
		t.Fatalf("应有用量")
	}
	if usage.Requests != 2 || usage.PromptTokens != 150 || usage.CompletionTokens != 30 || usage.TotalTokens != 180 {
		t.Fatalf("用量累计不对: %+v", usage)
	}
	if usage.CacheReadTokens != 30 || usage.CacheWriteTokens != 5 || usage.Cost != 0.25 {
		t.Fatalf("缓存与费用不对: %+v", usage)
	}

	// 没有用量事实时是 nil，不是零值。
	empty := newFolder(t)
	if empty.View().Usage != nil {
		t.Fatalf("没有用量事实时不该给用量: %+v", empty.View().Usage)
	}
}

// TestContextBudgetKeepsLatest 验证上下文占用取最新一条并给出剩余与比例。
func TestContextBudgetKeepsLatest(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(2,
		ev("b1", 1, "context", "budget", `{"turn_id":"t1","request_id":"r1","estimatedInputTokens":1000,"contextWindow":10000,"outputTokens":400}`),
		ev("b2", 2, "context", "budget", `{"turn_id":"t1","request_id":"r2","estimatedInputTokens":2500,"baselinePromptTokens":1000,"deltaTokens":1500,"contextWindow":10000,"outputTokens":400}`),
	))
	usage := folder.View().Context
	if usage == nil {
		t.Fatalf("应有上下文占用")
	}
	if usage.EstimatedInputTokens != 2500 || usage.ContextWindow != 10000 || usage.OutputTokens != 400 {
		t.Fatalf("上下文占用应取最新一条: %+v", usage)
	}
	if usage.BaselinePromptTokens != 1000 || usage.DeltaTokens != 1500 || usage.Sequence != 2 {
		t.Fatalf("基线与增量不对: %+v", usage)
	}
	if usage.Remaining() != 7500 {
		t.Fatalf("剩余应为 7500，实际 %d", usage.Remaining())
	}
	if percent := usage.Percent(); percent != 25 {
		t.Fatalf("占用比例应为 25，实际 %v", percent)
	}
}

// TestCompactionSummaryAndCount 验证压缩摘要、状态与次数。
func TestCompactionSummaryAndCount(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(2,
		ev("c1", 1, "context", "compaction", `{"status":"completed","through":1,"strategy":"summarize","retain_from":"e0","replacement":[{"role":"user","content":[{"type":"text","text":"摘要正文"}]}],"replacement_parts":{"strategy":1}}`),
		ev("c2", 2, "context", "compaction", `{"status":"cancelled","reason":"interrupted"}`),
	))
	view := folder.View()
	if view.Compactions != 2 {
		t.Fatalf("压缩次数应为 2，实际 %d", view.Compactions)
	}
	compaction := view.Compaction
	if compaction == nil {
		t.Fatalf("应有最近一次压缩")
	}
	if compaction.Status != "cancelled" || compaction.Summary != "" {
		t.Fatalf("取消的压缩不该有摘要: %+v", compaction)
	}

	// 再压一次：摘要与策略跟着最新一条走。
	mustNotice(t, folder, mainBranch, ev("c3", 3, "context", "compaction",
		`{"status":"completed","through":3,"strategy":"summarize","replacement":[{"role":"user","content":[{"type":"text","text":"第二次摘要"}]}],"replacement_parts":{"strategy":1}}`))
	compaction = folder.View().Compaction
	if compaction.Status != "completed" || compaction.Summary != "第二次摘要" || compaction.Strategy != "summarize" {
		t.Fatalf("最近一次压缩不对: %+v", compaction)
	}
	if folder.View().Compactions != 3 {
		t.Fatalf("压缩次数应为 3，实际 %d", folder.View().Compactions)
	}

	// 压缩事实仍然产生一条系统提示行，视图多出来的只是摘要。
	rows := 0
	for _, row := range folder.View().Rows {
		if row.Kind == RowSystem {
			rows++
		}
	}
	if rows != 3 {
		t.Fatalf("三次压缩应有三条系统提示行，实际 %d", rows)
	}
}

// TestCompactionSummaryTextVariant 验证 replacement 里的纯文本消息也能取到摘要。
func TestCompactionSummaryTextVariant(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(1,
		ev("c1", 1, "context", "compaction",
			`{"status":"completed","through":1,"strategy":"summary","replacement":[{"role":"user","content":"纯文本摘要"},{"role":"assistant","content":[{"type":"text","text":"补充"}]}],"replacement_parts":{"strategy":1,"digest":1}}`),
	))
	compaction := folder.View().Compaction
	// replacement_parts.strategy=1：只有第一条是策略产出的摘要；第二条是工具摘要条目。
	if compaction == nil || compaction.Summary != "纯文本摘要" {
		t.Fatalf("摘要应取策略消息（字符串形式）: %+v", compaction)
	}
}

// TestCompactionFailedStatus 验证失败的压缩保留状态与错误，不给摘要。
func TestCompactionFailedStatus(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(1,
		ev("c1", 1, "context", "compaction", `{"status":"failed","through":1,"error":"Compacted context does not reduce input.","diagnostics":{"entriesReplaced":0}}`),
	))
	compaction := folder.View().Compaction
	if compaction == nil || compaction.Status != "failed" || compaction.Summary != "" {
		t.Fatalf("失败的压缩不该有摘要: %+v", compaction)
	}
	if folder.View().Compactions != 1 {
		t.Fatalf("失败的压缩也计入次数: %d", folder.View().Compactions)
	}
}

// TestContextReadingHelpers 验证按需读数的剩余量、占比与阈值判定。
func TestContextReadingHelpers(t *testing.T) {
	reading := ContextReading{
		ContextWindow:        258000,
		OutputTokens:         128000,
		ThresholdTokens:      114000,
		EstimatedInputTokens: 33569,
	}
	if reading.Remaining() != 224431 {
		t.Fatalf("剩余量不对: %d", reading.Remaining())
	}
	if percent := reading.Percent(); percent < 12.9 || percent > 13.1 {
		t.Fatalf("占比不对: %v", percent)
	}
	if reading.OverThreshold() {
		t.Fatalf("没到阈值不该判过线")
	}
	reading.EstimatedInputTokens = 114000
	if !reading.OverThreshold() {
		t.Fatalf("到阈值应判过线")
	}
	unknown := ContextReading{}
	if unknown.Remaining() != 0 || unknown.Percent() != 0 || unknown.OverThreshold() {
		t.Fatalf("窗口未知时助手应给零值: %+v", unknown)
	}
}
