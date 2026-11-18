package ui

import (
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// liveFactSession 是一份来自 Source 的时间线：用户消息、思考、助手正文、
// 工具组（一成一败）、单个工具、失败行、重试、产物、改动与压缩分隔。
func liveFactSession() SessionSnapshot {
	return SessionSnapshot{
		ID: "s1", Title: "接入真实事实",
		Rows: []RowSnapshot{
			{Kind: RowUser, Text: "看看这次的失败", Status: RowStatusDone},
			{Kind: RowThinking, Text: "先读时间线，确认投影的字段。",
				Status: RowStatusDone, Duration: 400 * time.Millisecond},
			{Kind: RowAssistant, Text: "我把工具行接上了。", Status: RowStatusDone},
			{Kind: RowToolGroup, Status: RowStatusFailed, Children: []RowSnapshot{
				{Kind: RowTool, Title: "读取", Args: "internal/ui/timeline.go",
					Text: "读到了", Status: RowStatusDone, Duration: 1200 * time.Millisecond},
				{Kind: RowTool, Title: "改动", Args: "internal/ui/composer.go",
					Text: "权限不足，命令没有执行", Status: RowStatusFailed},
			}},
			{Kind: RowTool, Title: "执行", Args: "go test ./...",
				Text: "FAIL ./internal/ui", Status: RowStatusFailed},
			{Kind: RowError, Text: "failed"},
			{Kind: RowRetry, Duration: 5 * time.Second},
			{Kind: RowArtifact, Children: []RowSnapshot{
				{Kind: RowArtifact, Text: "排队说明.md"},
			}},
			{Kind: RowFileChanges, Children: []RowSnapshot{
				{Kind: RowFileChange, Text: "internal/ui/source.go"},
			}},
			{Kind: RowCompression, Text: "上下文已压缩，从这里开始新的窗口"},
		},
	}
}

// TestLiveTimelineDrawsFactBlocks：Source 给的每一类行都落到既有块上，
// 而不是一律按纯文本画。
func TestLiveTimelineDrawsFactBlocks(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.source = &fakeSource{
		conn:    Connection{Phase: "connected"},
		names:   []string{"接入真实事实"},
		session: liveFactSession(),
		hasSess: true,
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	// 用户气泡与助手正文。
	if _, ok := tt.Find("我的消息"); !ok {
		t.Fatal("用户行没有画成气泡")
	}
	if !tt.HasText("我把工具行接上了。") {
		t.Fatal("助手正文没有画出来")
	}
	// 思考块：标题里的时长来自事实。
	if _, ok := tt.Find("思考过程 思考了 0.4s"); !ok {
		t.Fatalf("思考块没有出现：%v", tt.Texts())
	}
	// 工具组：组头汇总两次调用（读 1 次、改 1 次），子行在展开里。
	if !tt.HasText("读1次文件、改1次文件") {
		t.Fatalf("工具组头没有汇总调用：%v", tt.Texts())
	}
	// 单个工具行：动作 + 参数摘要 + 失败标记都在行上。
	if _, ok := tt.Find("执行 go test ./..."); !ok {
		t.Fatalf("单个工具行没有出现：%v", tt.Texts())
	}
	// 失败行与重试倒计时。
	if !tt.HasText("运行失败") {
		t.Fatal("失败行没有出现")
	}
	if !tt.HasText("5s 后重试") {
		t.Fatal("重试倒计时没有出现")
	}
	// 改动块：收起时只有计数，点开后是文件与状态。
	if _, ok := tt.Find("1 个文件改动"); !ok {
		t.Fatalf("改动块没有出现：%v", tt.Texts())
	}
	if tt.HasText("改动文件 internal/ui/source.go") {
		t.Fatal("改动块收起时不该列出文件")
	}
	if err := tt.Click("1 个文件改动"); err != nil {
		t.Fatalf("点不到改动块：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.Find("改动文件 internal/ui/source.go"); !ok {
		t.Fatalf("改动块展开后没有文件行：%v", tt.Texts())
	}
	// 产物块：计数 + 展开后的产物卡。
	if _, ok := tt.Find("1 个产物"); !ok {
		t.Fatalf("产物块没有出现：%v", tt.Texts())
	}
	if err := tt.Click("1 个产物"); err != nil {
		t.Fatalf("点不到产物块：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.Find("产物 排队说明.md"); !ok {
		t.Fatalf("产物块展开后没有产物卡：%v", tt.Texts())
	}
	// 压缩分隔：服务端摘要原文写在分隔行上。
	if !tt.HasText("上下文已压缩，从这里开始新的窗口") {
		t.Fatal("压缩分隔没有出现")
	}
}

// TestLiveTimelineToolGroupExpands：工具组默认收起，点开后才列出工具行。
func TestLiveTimelineToolGroupExpands(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.source = &fakeSource{
		conn: Connection{Phase: "connected"}, names: []string{"接入真实事实"},
		session: liveFactSession(), hasSess: true,
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if tt.HasText("读到了") {
		t.Fatal("工具组收起时不该列出工具输出")
	}
	header, ok := tt.Find("轮次组头 读1次文件、改1次文件，状态：失败，" +
		"读取 timeline.go · 改动 composer.go")
	if !ok {
		t.Fatalf("找不到工具组头：%v", tt.Texts())
	}
	tt.ClickAt(header.X+header.W/2, header.Y+header.H/2)
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.Find("读取 timeline.go"); !ok {
		t.Fatalf("工具组展开后没有工具行：%v", tt.Texts())
	}
	// 工具行的结果只在行自己展开时出现。
	if err := tt.Click("改动 composer.go"); err != nil {
		t.Fatalf("点不到工具行：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("权限不足，命令没有执行") {
		t.Fatalf("工具行展开后没有结果原文：%v", tt.Texts())
	}
}

// TestLiveTimelineFallsBackToText：认不出的行种类仍按正文画；空行不画也不 panic。
func TestLiveTimelineFallsBackToText(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.source = &fakeSource{
		conn: Connection{Phase: "connected"}, names: []string{"接入真实事实"},
		hasSess: true,
		session: SessionSnapshot{ID: "s1", Title: "接入真实事实", Rows: []RowSnapshot{
			{Kind: "some_future_kind", Text: "认不出的行也照原文画"},
			{Kind: RowAssistant, Text: ""},
			{Kind: RowFileChanges},
		}},
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if !tt.HasText("认不出的行也照原文画") {
		t.Fatalf("认不出的行没有按正文画：%v", tt.Texts())
	}
}

// TestRowDurationText：块标题里的时长只写一档，没有时长事实时不写。
func TestRowDurationText(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, ""},
		{400 * time.Millisecond, "0.4s"},
		{12 * time.Second, "12s"},
		{90 * time.Second, "1m30s"},
	}
	for _, tc := range cases {
		if got := rowDurationText(tc.in); got != tc.want {
			t.Fatalf("rowDurationText(%v) = %q，想要 %q", tc.in, got, tc.want)
		}
	}
}
