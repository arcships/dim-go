package ui

import (
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 性能预算（U4 §3）。测法与命令见 docs/domains/U4-technical.md：
//
//	go test ./internal/ui -run 'TestPerf|^$' -bench . -benchtime 20x -benchmem
//
// 预算是"上界"，不是目标：超过就说明这一帧做了不该做的事。

// 预算值（毫秒）。机器无关的宽松上界，取实测值的 3–5 倍。
const (
	budgetFirstFrameMS  = 250.0
	budgetSteadyFrameMS = 16.0
	budgetLongFeedMS    = 60.0
)

// timeNow / msSince / median 是三个小工具，避免在断言里散落时间换算。
func timeNow() time.Time { return time.Now() }

func msSince(t time.Time) float64 { return float64(time.Since(t).Microseconds()) / 1000 }

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[len(c)/2]
}

// benchRounds 造 n 轮样例：一半带展开的操作，一半只有回复。
func benchRounds(n int) []round {
	out := make([]round, 0, n)
	for i := 0; i < n; i++ {
		r := round{
			user:   fmt.Sprintf("第 %d 轮：把发送路径改成排队", i+1),
			reply:  "运行中的提交先入队，当前这一轮结束后自动发出。",
			status: roundDone,
			ops: []op{
				{action: "读取", target: "internal/ui/composer.go", result: "完成"},
				{action: "改动", target: "internal/ui/timeline.go", result: "完成"},
				{action: "执行", target: "go test ./internal/ui", result: "完成"},
			},
		}
		out = append(out, r)
	}
	return out
}

// TestPerfBudgets 把预算写成断言：超过就失败，命令可复跑。
func TestPerfBudgets(t *testing.T) {
	// 首帧：建窗口 + 第一帧。
	start := timeNow()
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	firstFrame := msSince(start)
	if firstFrame > budgetFirstFrameMS {
		t.Fatalf("首帧 %.2fms 超过预算 %.0fms", firstFrame, budgetFirstFrameMS)
	}

	// 稳态帧：已建好的窗口连画 20 帧取中位数。
	frames := make([]float64, 0, 20)
	for i := 0; i < 20; i++ {
		start := timeNow()
		tt.Frame()
		frames = append(frames, msSince(start))
	}
	steady := median(frames)
	if steady > budgetSteadyFrameMS {
		t.Fatalf("稳态帧 %.2fms 超过预算 %.0fms", steady, budgetSteadyFrameMS)
	}

	// 长列表：200 轮，画 10 帧取中位数。
	long := NewApp(Options{State: StateDefault})
	long.rounds = benchRounds(200)
	longTT := gui.NewTester(long.View, WindowWidth, WindowHeight)
	longFrames := make([]float64, 0, 10)
	for i := 0; i < 10; i++ {
		start := timeNow()
		longTT.Frame()
		longFrames = append(longFrames, msSince(start))
	}
	longFrame := median(longFrames)
	if longFrame > budgetLongFeedMS {
		t.Fatalf("200 轮的帧 %.2fms 超过预算 %.0fms", longFrame, budgetLongFeedMS)
	}
	t.Logf("首帧 %.2fms（预算 %.0f）· 稳态 %.2fms（预算 %.0f）· 200 轮 %.2fms（预算 %.0f）",
		firstFrame, budgetFirstFrameMS, steady, budgetSteadyFrameMS, longFrame, budgetLongFeedMS)
}

// TestPerfMemory 量大致的常驻内存：一屏 + 200 轮的 App。
func TestPerfMemory(t *testing.T) {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	long := NewApp(Options{State: StateDefault})
	long.rounds = benchRounds(200)
	tt := gui.NewTester(long.View, WindowWidth, WindowHeight)
	tt.Frame()
	runtime.GC()
	runtime.ReadMemStats(&after)
	heap := float64(after.HeapAlloc) / (1 << 20)
	if heap > 96 {
		t.Fatalf("200 轮时堆 %.1fMB 超过预算 96MB", heap)
	}
	t.Logf("200 轮常驻堆 %.1fMB（预算 96MB）", heap)
}

// BenchmarkFirstFrame 是首帧的基准（可与预算对照看趋势）。
func BenchmarkFirstFrame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		app := NewApp(Options{State: StateDefault})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		_ = tt.Image()
	}
}

// BenchmarkSteadyFrame 是稳态帧的基准。
func BenchmarkSteadyFrame(b *testing.B) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}

// BenchmarkLongFeed200 是 200 轮长列表的基准。
func BenchmarkLongFeed200(b *testing.B) {
	app := NewApp(Options{State: StateDefault})
	app.rounds = benchRounds(200)
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}
