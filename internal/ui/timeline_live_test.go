package ui

import (
	"slices"
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

// 事实时间线里的用户消息也走同一套悬停动作与右键菜单（TL-37–TL-43）：
// 消息来自 RowSnapshot，动作不另画一份。
func TestLiveUserMessageActionsAndMenu(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.source = &fakeSource{
		conn: Connection{Phase: "connected"}, names: []string{"接入真实事实"},
		session: liveFactSession(), hasSess: true,
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	bubble, ok := tt.Find("我的消息")
	if !ok {
		t.Fatalf("用户行没有画成气泡：%v", tt.Texts())
	}
	tt.Move(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()
	if _, ok := tt.Find("复制整个消息 1"); !ok {
		t.Fatalf("悬停后没有消息动作：%v", tt.Texts())
	}
	// 快照没有给轮次（TurnIndex 为 0）：分叉置灰并写原因，不拿别的轮次顶替。
	if info, ok := tt.Access("从此处分叉 1（" + forkNoteNoTurn + "）"); !ok || !info.Disabled {
		t.Fatalf("分叉没有置灰：%+v（找到 %v）", info, ok)
	}
	if err := tt.Click("复制整个消息 1"); err != nil {
		t.Fatalf("点不到复制：%v", err)
	}
	if tt.Clipboard() != "看看这次的失败" {
		t.Fatalf("复制的是 %q", tt.Clipboard())
	}

	tt.RightClickAt(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()
	if !slices.Contains(tt.Menu(), "复制整个消息") {
		t.Fatalf("事实消息没有右键菜单：%v", tt.Menu())
	}
	if err := tt.ChooseMenuItem("复制整个消息"); err != nil {
		t.Fatalf("点不到菜单复制：%v", err)
	}
	if tt.Clipboard() != "看看这次的失败" {
		t.Fatalf("菜单复制的是 %q", tt.Clipboard())
	}
}

// 助手正文右键菜单（TL-65）：复制整个消息。
func TestLiveAssistantContextMenu(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.source = &fakeSource{
		conn: Connection{Phase: "connected"}, names: []string{"接入真实事实"},
		session: liveFactSession(), hasSess: true,
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	body, ok := tt.Find("助手正文")
	if !ok {
		t.Fatalf("没有助手正文：%v", tt.Texts())
	}
	tt.RightClickAt(body.X+body.W/2, body.Y+body.H/2)
	tt.Frame()
	menu := tt.Menu()
	if !slices.Contains(menu, "复制整个消息") {
		t.Fatalf("助手正文没有右键菜单：%v", menu)
	}
	if err := tt.ChooseMenuItem("复制整个消息"); err != nil {
		t.Fatalf("点不到复制整个消息：%v", err)
	}
	if tt.Clipboard() != "我把工具行接上了。" {
		t.Fatalf("复制整个消息得到 %q", tt.Clipboard())
	}
}

// 工具详情三段在事实时间线上同样成立：失败原因没有单独进快照时给出占位，
// 不把输出冒充原因。
func TestLiveToolDetailShowsFailureSection(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	session := liveFactSession()
	app.source = &fakeSource{
		conn: Connection{Phase: "connected"}, names: []string{"接入真实事实"},
		session: session, hasSess: true,
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	group := session.Rows[3]
	header := "轮次组头 " + groupSummary(toolGroupRound(group)) + "，状态：" +
		statusWord(liveRoundStatus(group.Status)) + "，" + roundHeaderPhrase(toolGroupRound(group))
	if err := tt.Click(header); err != nil {
		t.Fatalf("点不到工具组头：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if err := tt.Click("改动 composer.go"); err != nil {
		t.Fatalf("点不到失败的工具行：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.Find("工具详情：输出"); !ok {
		t.Fatalf("没有输出段：%v", tt.Texts())
	}
	if _, ok := tt.Find("工具详情：失败原因"); !ok {
		t.Fatalf("没有失败原因段：%v", tt.Texts())
	}
	if !tt.HasText("失败原因还没有单独进快照（等事实）") {
		t.Fatalf("失败原因段没有写明等事实：%v", tt.Texts())
	}
	if !tt.HasText("权限不足，命令没有执行") {
		t.Fatalf("输出原文没有上屏：%v", tt.Texts())
	}
}

// 真实消息的附件与引用字段还没进 RowSnapshot（DEC-07）：画一行「等事实」的
// 占位，不画假 chip。
func TestUserMessageAttachmentPlaceholder(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.source = &fakeSource{
		conn:    Connection{Phase: "connected"},
		names:   []string{"接入真实事实"},
		session: liveFactSession(),
		hasSess: true,
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if _, ok := tt.Find("附件与引用占位"); !ok {
		t.Fatalf("真实消息没有附件占位：%v", tt.Texts())
	}
	if !tt.HasText(noteAttachmentFacts) {
		t.Fatalf("占位没有写明等事实：%v", tt.Texts())
	}
	// 产物块来自事实，但「打开」入口没有协议通道：置灰并写原因（DEC-06）。
	if err := tt.Click("1 个产物"); err != nil {
		t.Fatalf("点不到产物块：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	info, ok := tt.Access("打开产物 排队说明.md（" + noteArtifact + "）")
	if !ok || !info.Disabled {
		t.Fatalf("产物打开入口没有置灰：%+v（找到 %v）", info, ok)
	}
}
