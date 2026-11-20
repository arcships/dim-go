package ui

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是长历史与长输出的实测口径（U4 §3）。与 perf_test.go 的分工：
// perf_test.go 守"一屏 + 200 轮样例"的三条预算（PF-250 / PF-16 / PF-60）；
// 本文件补测三件更重的事，并把实测值写回 docs/domains/U4-technical.md：
//
//  1. 200 轮快照（每轮 = 用户 + 思考 + 2 条工具 + 正文，共 1000 行）的首帧、
//     稳态帧、滚到顶 / 底、跳到中间轮的帧时间与常驻堆；
//  2. 渲染窗口（虚拟化）：行数超过门槛后只画视口内的行，搜索定位仍可用；
//  3. 长输出：单条 1 MB 工具输出与 1000 行代码块的渲染与滚动。
//
// 预算（docs/domains/requirements/design-baseline.md §11.8）：首帧 ≤ 250ms
// （PF-250）、稳态帧 ≤ 16ms（PF-16）、长列表一帧 ≤ 60ms（PF-60）。
// 复跑：go test ./internal/ui -run 'TestPerf' -v

const (
	// budgetLongHistoryHeapMB 是 200 轮（1000 行）渲染完后的常驻堆预算。
	budgetLongHistoryHeapMB = 128
	// budgetLongOutputMS 是长输出 / 长代码块一帧的预算：与长列表同口径
	// （PF-60），因为展开的长内容本身就是一段长列表。
	budgetLongOutputMS = 60.0
)

// perfThinking 是每轮思考块的原文（取样例里的一段真实思考）。
const perfThinking = "先把发送路径读一遍：入队在 composer 里，提交在 intents 里，" +
	"两处都有 running 判断；改完要确认没有第二处写 running。"

// perfReply 是每轮助手正文（一段带行内代码的 Markdown）。
const perfReply = "改好了：`send()` 先看 `running`，为真就入队；轮结束后由\n" +
	"`drainQueue()` 发出下一条。相关测试在 `internal/ui/composer_test.go`。\n\n" +
	"- 入队路径不写事实，只调 `Intents.QueueSubmit`\n- 排队消息面板照旧显示条数"

// perfLongRows 造 n 轮快照：每轮 5 行（用户、思考、两条工具、正文）。
func perfLongRows(n int) []RowSnapshot {
	rows := make([]RowSnapshot, 0, n*5)
	for i := 0; i < n; i++ {
		turn := i + 1
		rows = append(rows,
			RowSnapshot{
				Kind: RowUser, TurnIndex: turn, Status: RowStatusDone,
				Text: fmt.Sprintf("第 %d 轮：把发送路径改成排队，跑一遍相关测试，逐条核对失败原因。", turn),
			},
			RowSnapshot{
				Kind: RowThinking, TurnIndex: turn, Status: RowStatusDone,
				Text: perfThinking, Duration: 2 * time.Second,
			},
			RowSnapshot{
				Kind: RowTool, TurnIndex: turn, Status: RowStatusDone,
				Title: "读取", Args: "internal/ui/composer.go", Text: "完成",
			},
			RowSnapshot{
				Kind: RowTool, TurnIndex: turn, Status: RowStatusDone,
				Title: "执行", Args: "go test ./internal/ui -count=1", Text: "ok",
			},
			RowSnapshot{
				Kind: RowAssistant, TurnIndex: turn, Status: RowStatusDone, Text: perfReply,
			},
		)
	}
	return rows
}

// perfLongSource 是一个带 n 轮快照的只读 Source。
func perfLongSource(n int) *fakeSource {
	src := wiredSource()
	src.names = []string{"长历史"}
	src.hasSess = true
	src.session = SessionSnapshot{
		ID: "s1", Title: "长历史", Branch: "main", TurnCount: n,
		Rows: perfLongRows(n),
	}
	return src
}

// perfLongTester 建 App 与 Tester，并返回首帧耗时（毫秒）。
func perfLongTester(t *testing.T, n int) (*App, *gui.Tester, float64) {
	t.Helper()
	start := timeNow()
	app := NewApp(Options{State: StateDefault, Source: perfLongSource(n)})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	return app, tt, msSince(start)
}

// frameMS 画一帧并返回耗时（毫秒）。
func frameMS(tt *gui.Tester) float64 {
	start := timeNow()
	tt.Frame()
	return msSince(start)
}

// medFrames 连画 n 帧，返回中位数与最大值（毫秒）。
func medFrames(tt *gui.Tester, n int) (med float64, max float64) {
	frames := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		f := frameMS(tt)
		frames = append(frames, f)
		if f > max {
			max = f
		}
	}
	return median(frames), max
}

// TestPerfLongHistory 是 200 轮（1000 行）的四件实测：首帧、稳态帧、滚到顶 /
// 底、跳到中间轮；每条都对着 §11.8 的预算判定。
func TestPerfLongHistory(t *testing.T) {
	const rounds = 200
	total := len(perfLongRows(rounds))
	app, tt, first := perfLongTester(t, rounds)
	if first > budgetFirstFrameMS {
		t.Fatalf("200 轮首帧 %.2fms 超过预算 %.0fms", first, budgetFirstFrameMS)
	}

	// 稳态：视口没动，连画 20 帧。
	steady, steadyMax := medFrames(tt, 20)
	if steady > budgetSteadyFrameMS {
		t.Fatalf("200 轮稳态帧 %.2fms 超过预算 %.0fms", steady, budgetSteadyFrameMS)
	}

	// 滚到底（初始位置在顶部，先滚到最后一页）。
	app.feed.Y = app.feed.MaxY
	bottom, bottomMax := medFrames(tt, 10)
	if bottom > budgetLongFeedMS {
		t.Fatalf("滚到底一帧 %.2fms 超过预算 %.0fms", bottom, budgetLongFeedMS)
	}

	// 滚到顶：一次跨越整段历史的帧（最坏情况：整段内容都不在缓存里）。
	app.feed.Y = 0
	top, topMax := medFrames(tt, 10)
	if top > budgetLongFeedMS {
		t.Fatalf("滚到顶一帧 %.2fms 超过预算 %.0fms", top, budgetLongFeedMS)
	}

	// 跳到中间轮：滚动位置取一半，连画两帧（第一帧排版，第二帧稳态）。
	app.feed.Y = app.feed.MaxY / 2
	mid := frameMS(tt)
	mid2 := frameMS(tt)
	if mid2 > budgetLongFeedMS {
		t.Fatalf("跳到中间轮一帧 %.2fms 超过预算 %.0fms", mid2, budgetLongFeedMS)
	}

	// 常驻堆：渲染完 1000 行后的 HeapAlloc。
	runtime.GC()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	heap := float64(mem.HeapAlloc) / (1 << 20)
	if heap > budgetLongHistoryHeapMB {
		t.Fatalf("200 轮常驻堆 %.1fMB 超过预算 %.0fMB", heap, float64(budgetLongHistoryHeapMB))
	}

	t.Logf("200 轮 / %d 行：首帧 %.2fms（预算 %.0f）· 稳态 %.2fms 最大 %.2f（预算 %.0f）"+
		"· 滚到底 %.2fms 最大 %.2f · 滚到顶 %.2fms 最大 %.2f · 跳中间 %.2f/%.2fms（预算 %.0f）"+
		"· 常驻堆 %.1fMB（预算 %.0f）· 内容高 %.0fDIP",
		total, first, budgetFirstFrameMS, steady, steadyMax, budgetSteadyFrameMS,
		bottom, bottomMax, top, topMax, mid, mid2, budgetLongFeedMS,
		heap, float64(budgetLongHistoryHeapMB), app.feed.MaxY)
}

// TestPerfVirtualWindow 守"只画视口内的行"：2000 轮（10000 行）时渲染的行
// 数量与总行数无关，滚动到最后一屏后末行出现、首行消失。
func TestPerfVirtualWindow(t *testing.T) {
	const rounds = 2000
	app, tt, first := perfLongTester(t, rounds)
	total := len(perfLongRows(rounds))
	if first > budgetFirstFrameMS {
		t.Fatalf("2000 轮首帧 %.2fms 超过预算 %.0fms", first, budgetFirstFrameMS)
	}
	if got := app.renderedRows(); got > timelineWindowMaxRendered {
		t.Fatalf("首帧渲染了 %d 行，超过窗口上界 %d（总行数 %d）",
			got, timelineWindowMaxRendered, total)
	}
	// 首屏：最后一段历史不该在元素树里。
	if tt.HasText("第 2000 轮") {
		t.Fatal("首帧把最末一轮也渲染了：渲染窗口没有生效")
	}
	// 滚到底：末轮出现，首轮退出元素树。
	app.feed.Y = app.feed.MaxY
	tt.Frame()
	tt.Frame()
	if !tt.HasText("第 2000 轮") {
		t.Fatalf("滚到底后没有末轮的正文（feed.Y=%.0f MaxY=%.0f 渲染 %d 行）",
			app.feed.Y, app.feed.MaxY, app.renderedRows())
	}
	if tt.HasText("第 1 轮") {
		t.Fatal("滚到底后首轮仍在元素树里：视口外的行没有退出渲染")
	}
	bottom := medFramesOnly(tt, 10)
	if bottom > budgetLongFeedMS {
		t.Fatalf("10000 行滚到底一帧 %.2fms 超过预算 %.0fms", bottom, budgetLongFeedMS)
	}
	// 滚到顶：回到第一轮。
	app.feed.Y = 0
	tt.Frame()
	tt.Frame()
	if !tt.HasText("第 1 轮") {
		t.Fatal("滚到顶后没有首轮的正文")
	}
	t.Logf("2000 轮 / %d 行：首帧 %.2fms · 渲染 %d 行 · 滚到底一帧 %.2fms · 内容高 %.0fDIP（占位估算）",
		total, first, app.renderedRows(), bottom, app.feed.MaxY)
}

// medFramesOnly 只取中位数（滚动后的稳态）。
func medFramesOnly(tt *gui.Tester, n int) float64 {
	m, _ := medFrames(tt, n)
	return m
}

// TestPerfVirtualSearchReveal 守"搜索定位仍可用"：命中不在渲染窗口里时，
// 跳过去要把那一行拉回窗口，并把它滚进视口。
func TestPerfVirtualSearchReveal(t *testing.T) {
	app, tt, _ := perfLongTester(t, 2000)
	app.feed.Y = 0
	tt.Frame()

	// 打开会话内搜索并查一个只在第 1500 轮出现过的词。
	app.sessionSearch.open = true
	app.sessionSearch.think = true
	app.sessionSearch.query = "第 1500 轮"
	app.sessionSearch.focus = false
	app.sessionSearch.current = -1
	tt.Frame()
	tt.Frame()
	if len(app.sessionSearch.hits) == 0 {
		t.Fatalf("没有命中：units=%d query=%q", len(app.sessionSearch.units), app.sessionSearch.query)
	}
	// 跳到最后一条命中（这句话只有一条）：那一行要先进入窗口才滚得到。
	app.sessionSearchJump(1)
	for i := 0; i < 6; i++ {
		tt.Frame()
	}
	if app.feed.Y <= 0 {
		t.Fatalf("命中没有滚进视口（feed.Y = %.0f）", app.feed.Y)
	}
	if !tt.HasText("第 1500 轮") {
		t.Fatalf("命中所在行没有被拉进渲染窗口（渲染 %d 行）", app.renderedRows())
	}
}

// perfToolSource 造一条带长输出的工具行（正文 = output），并把它展开。
func perfToolSource(output string) *App {
	src := wiredSource()
	src.names = []string{"长输出"}
	src.hasSess = true
	src.session = SessionSnapshot{
		ID: "s1", Title: "长输出", Branch: "main", TurnCount: 1,
		Rows: []RowSnapshot{
			{Kind: RowUser, TurnIndex: 1, Status: RowStatusDone, Text: "把这次命令的完整输出贴出来"},
			{Kind: RowTool, TurnIndex: 1, Status: RowStatusDone,
				Title: "执行", Args: "go test ./... -v", Text: output},
		},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	// 展开工具行（与点一下同一处视图状态）。
	open := app.blockOf("tool:1:0", false)
	*open = true
	return app
}

// TestPerfLongToolOutput 是单条 1 MB 工具输出的渲染与滚动。
func TestPerfLongToolOutput(t *testing.T) {
	output := perfBigOutput(1 << 20)
	app := perfToolSource(output)
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	// 展开是 300ms 的高度过渡：走完再量，量到的才是"整段都在"的帧。
	tt.Advance(400 * time.Millisecond)

	first := frameMS(tt)
	steady, max := medFrames(tt, 10)
	if steady > budgetLongOutputMS {
		t.Fatalf("1MB 输出稳态帧 %.2fms 超过预算 %.0fms", steady, budgetLongOutputMS)
	}
	if max > budgetLongOutputMS {
		t.Fatalf("1MB 输出帧最大 %.2fms 超过预算 %.0fms", max, budgetLongOutputMS)
	}
	// 滚动：视口里只有一段，滚到中间与末尾各画一帧。
	for _, y := range []float32{app.feed.MaxY / 2, app.feed.MaxY} {
		app.feed.Y = y
		if f := frameMS(tt); f > budgetLongOutputMS {
			t.Fatalf("1MB 输出滚动到 %.0f 一帧 %.2fms 超过预算 %.0fms", y, f, budgetLongOutputMS)
		}
	}
	runtime.GC()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	t.Logf("1MB 输出（%d 字节，%d 行）：首帧 %.2fms · 稳态 %.2fms 最大 %.2f（预算 %.0f）"+
		"· 单行内容高 %.0fDIP · 常驻堆 %.1fMB",
		len(output), strings.Count(output, "\n")+1, first, steady, max,
		budgetLongOutputMS, app.feed.MaxY, float64(mem.HeapAlloc)/(1<<20))
}

// TestPerfLongCodeBlock 是 1000 行代码块的渲染与滚动。
func TestPerfLongCodeBlock(t *testing.T) {
	output := perfCodeLines(1000)
	app := perfToolSource(output)
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Advance(400 * time.Millisecond)

	first := frameMS(tt)
	steady, max := medFrames(tt, 10)
	if steady > budgetLongOutputMS {
		t.Fatalf("1000 行代码块稳态帧 %.2fms 超过预算 %.0fms", steady, budgetLongOutputMS)
	}
	if max > budgetLongOutputMS {
		t.Fatalf("1000 行代码块帧最大 %.2fms 超过预算 %.0fms", max, budgetLongOutputMS)
	}
	for _, y := range []float32{app.feed.MaxY / 2, app.feed.MaxY} {
		app.feed.Y = y
		if f := frameMS(tt); f > budgetLongOutputMS {
			t.Fatalf("1000 行代码块滚动到 %.0f 一帧 %.2fms 超过预算 %.0fms", y, f, budgetLongOutputMS)
		}
	}
	t.Logf("1000 行代码块（%d 字节）：首帧 %.2fms · 稳态 %.2fms 最大 %.2f（预算 %.0f）· 内容高 %.0fDIP",
		len(output), first, steady, max, budgetLongOutputMS, app.feed.MaxY)
}

// perfBigOutput 造一段约 n 字节的命令输出（每行 64 字节，等宽、含路径与状态）。
func perfBigOutput(n int) string {
	var b strings.Builder
	b.Grow(n + 128)
	line := 0
	for b.Len() < n {
		line++
		fmt.Fprintf(&b, "ok   internal/pkg%04d  %.2fs  coverage: %.1f%% of statements\n",
			line%4000, float64(line%900)/100, float64(line%100))
	}
	return b.String()
}

// perfCodeLines 造 n 行代码（Go 片段，单行 44 字符左右）。
func perfCodeLines(n int) string {
	var b strings.Builder
	b.Grow(n * 48)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "\tif err := step%d(ctx, in); err != nil {\n", i)
		fmt.Fprintf(&b, "\t\treturn fmt.Errorf(\"step %d: %%w\", err)\n", i)
	}
	return b.String()
}

// TestPerfLongHistoryBench 是 200 轮首帧与滚动的基准（与断言同一套样例）。
func BenchmarkLongHistory200(b *testing.B) {
	src := perfLongSource(200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := NewApp(Options{State: StateDefault, Source: src})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		_ = tt.Image()
	}
}

// BenchmarkLongHistoryScroll 是 200 轮滚到顶的基准。
func BenchmarkLongHistoryScroll(b *testing.B) {
	app := NewApp(Options{State: StateDefault, Source: perfLongSource(200)})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.feed.Y = 0
		tt.Frame()
		app.feed.Y = app.feed.MaxY
		tt.Frame()
	}
}
