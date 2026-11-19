package uiwire

import (
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// P1-12：adapter 不产重试行，也不把"已耗时"写成倒计时。
//
// 协议 v3 没有"还剩多久重试"这条事实：界面按 ui.RowSnapshot.RetryIn 画倒计时，
// RetryIn 为 0 就整行不画。这里守的是生产端——任何一行投射出来都不能变成
// RowRetry，起止时间的差值只能落在 Duration（已耗时）上，不能落到 RetryIn。
func TestRowSnapshotNeverFabricatesRetryCountdown(t *testing.T) {
	kinds := []projection.RowKind{
		projection.RowUserMessage, projection.RowAssistantText, projection.RowThinking,
		projection.RowTool, projection.RowSystem,
	}
	for _, kind := range kinds {
		row, _ := rowSnapshot(projection.Row{
			Kind: kind, Text: "正文", CreatedAt: 1000, UpdatedAt: 6000,
			Status: projection.StatusRunning,
		})
		if row.Kind == ui.RowRetry {
			t.Fatalf("%s 行被投射成重试行：%+v", kind, row)
		}
		if row.RetryIn != 0 {
			t.Fatalf("%s 行给出了没有事实来源的剩余时间：%+v", kind, row.RetryIn)
		}
	}

	// 起止时间的差值是"已耗时"，只进 Duration；它不会被界面当成"还剩"。
	row, ok := rowSnapshot(projection.Row{
		Kind: projection.RowThinking, Text: "先读时间线", CreatedAt: 1000, UpdatedAt: 6000,
	})
	if !ok {
		t.Fatal("思考行没有投射出来")
	}
	if row.Duration != 5*time.Second {
		t.Fatalf("Duration = %v，想要 5s", row.Duration)
	}
	if row.RetryIn != 0 {
		t.Fatalf("RetryIn = %v，想要 0（没有剩余时间事实）", row.RetryIn)
	}
}
