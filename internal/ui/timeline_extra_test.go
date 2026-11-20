package ui

import (
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// labelCount 数一帧里某个可定位标签出现了几次（同一元素每帧只记一次）。
func labelCount(tt *gui.Tester, label string) int {
	return strings.Count(strings.Join(tt.Texts(), "\n"), label)
}

// 用户消息轨道与标记：每条用户消息一个标记；窄窗里轨道让位给正文。
func TestUserMessageRailMarksEveryUserMessage(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if got := labelCount(tt, "用户消息轨道"); got != len(app.rounds) {
		t.Fatalf("轨道段数 = %d，想要 %d（每条用户消息一段）", got, len(app.rounds))
	}
	if got := labelCount(tt, "用户消息标记"); got != len(app.rounds) {
		t.Fatalf("标记数 = %d，想要 %d（每条用户消息一个）", got, len(app.rounds))
	}
	// 轨道不进读屏：标签是给检查器定位的装饰。
	if info, ok := tt.AccessByRole(gui.RoleButton, "用户消息轨道"); ok {
		t.Fatalf("轨道进了读屏：%+v", info)
	}
}

func TestUserMessageRailHiddenInNarrowWindow(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, CompactWidth-40, WindowHeight)
	if got := labelCount(tt, "用户消息轨道"); got != 0 {
		t.Fatalf("窄窗里轨道计数 = %d，想要 0", got)
	}
	if got := labelCount(tt, "用户消息标记"); got != 0 {
		t.Fatalf("窄窗里标记计数 = %d，想要 0", got)
	}
}

// 虚拟化占位：还有更早的行没装载时给一个不含文字的空占位；没有该事实时不画。
func TestVirtualTopPlaceholderFollowsHasOlder(t *testing.T) {
	src := &fakeSource{
		conn: Connection{Phase: "connected"}, names: []string{"s1"}, hasSess: true,
		session: SessionSnapshot{ID: "s1", HasOlder: true,
			Rows: []RowSnapshot{{Kind: RowUser, Text: "已加载的一段"}}},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("更早的行占位"); !ok {
		t.Fatalf("有更早的行时没有占位：%v", tt.Texts())
	}
	src.session.HasOlder = false
	tt.Frame()
	if _, ok := tt.Find("更早的行占位"); ok {
		t.Fatal("没有更早的行时仍画了占位")
	}
}

// 恢复遮罩：正在装载第一轮时画品牌标记与说明，不把「还没有消息」当结论。
func TestRestoreMaskWhileLoadingFirstTurn(t *testing.T) {
	src := &fakeSource{
		conn: Connection{Phase: "connected"}, names: []string{"s1"}, hasSess: true,
		session: SessionSnapshot{ID: "s1", LoadPhase: "loading_first_turn"},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("恢复遮罩"); !ok {
		t.Fatalf("装载第一轮时没有恢复遮罩：%v", tt.Texts())
	}
	if !tt.HasText(noteRestore) {
		t.Fatalf("恢复遮罩没有说明：%v", tt.Texts())
	}
	if tt.HasText("当前会话还没有消息") {
		t.Fatal("恢复中把空态当成了结论")
	}
	src.session.LoadPhase = "ready"
	src.session.Rows = []RowSnapshot{{Kind: RowUser, Text: "到了"}}
	tt.Frame()
	if _, ok := tt.Find("恢复遮罩"); ok {
		t.Fatal("装载完成后遮罩还在")
	}
}

// 工作过程变体：完成写「工作过程 · 动作汇总」；被停止写「已停止 · 时长」并给
// 一行「用户中断」；失败按有没有终端现场分成两种。
func TestWorkProcessVariants(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if !tt.HasText("工作过程 · ") {
		t.Fatalf("完成的一轮没有「工作过程 · 动作」：%v", tt.Texts())
	}

	app.rounds[0].status = roundStopped
	app.rounds[0].duration = "12s"
	tt.Frame()
	if !tt.HasText("已停止 · 12s") {
		t.Fatalf("停止的一轮标题不对：%v", tt.Texts())
	}
	if _, ok := tt.Find("用户中断"); !ok {
		t.Fatalf("停止的一轮没有「用户中断」：%v", tt.Texts())
	}

	app.rounds[0].status = roundFailed
	app.rounds[0].duration = "1m"
	app.rounds[0].ops = []op{{
		action: "执行", target: "go test ./...", output: "FAIL", failed: true,
	}}
	tt.Frame()
	if !tt.HasText("1m 后失败") {
		t.Fatalf("失败的一轮没有「{时长} 后失败」：%v", tt.Texts())
	}

	app.rounds[0].ops = []op{{action: "读取", target: "internal/ui/timeline.go"}}
	tt.Frame()
	if !tt.HasText("出错前的工作过程") {
		t.Fatalf("没有终端现场时标题不是「出错前的工作过程」：%v", tt.Texts())
	}
}

// 子代理通知：工具名认得出是子代理时画 chip，名称与通知都来自事实行。
func TestSubagentNotificationChip(t *testing.T) {
	src := &fakeSource{
		conn: Connection{Phase: "connected"}, names: []string{"s1"}, hasSess: true,
		session: SessionSnapshot{ID: "s1", Rows: []RowSnapshot{
			{Kind: RowTool, Title: "task", Args: "调研投影", Text: "子代理已完成",
				Status: RowStatusDone},
		}},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if !tt.HasText("子代理：调研投影") || !tt.HasText("子代理已完成") {
		t.Fatalf("子代理 chip 没有名称或通知：%v", tt.Texts())
	}
	if err := tt.Click("子代理：调研投影，通知：子代理已完成"); err != nil {
		t.Fatalf("点不到子代理 chip：%v", err)
	}
	if !app.rightOpen || app.rightTab != tabSubagent {
		t.Fatalf("点子代理 chip 没有打开子代理页：open=%v tab=%d", app.rightOpen, app.rightTab)
	}
}

// 终端错误详情：执行类工具失败后，展开详情给标题、复制与输出原文。
func TestTerminalErrorDetail(t *testing.T) {
	src := &fakeSource{
		conn: Connection{Phase: "connected"}, names: []string{"s1"}, hasSess: true,
		session: SessionSnapshot{ID: "s1", Rows: []RowSnapshot{
			{Kind: RowTool, Title: "执行", Args: "go test ./...", Text: "FAIL ./internal/ui",
				Status: RowStatusFailed},
		}},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("执行 go test ./..."); err != nil {
		t.Fatalf("点不到工具行：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.Find("工具详情：终端错误详情"); !ok {
		t.Fatalf("展开后没有终端错误详情：%v", tt.Texts())
	}
	if _, ok := tt.Find("复制终端错误详情"); !ok {
		t.Fatalf("终端错误详情没有复制入口：%v", tt.Texts())
	}
	if !tt.HasText("FAIL ./internal/ui") {
		t.Fatalf("终端错误详情没有输出原文：%v", tt.Texts())
	}
}

// 行内提及：技能 chip、文件按钮（打开右栏预览）、提交 chip（不可点）与链接
// （本版没有外部打开通道，置灰写原因）。
func TestInlineMentions(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.rounds[0].user = "看 @internal/ui/timeline.go 和 $deep-read 以及 #deadbeef 与 https://example.com/x"
	tt.Frame()
	if _, ok := tt.Find("打开文件：internal/ui/timeline.go"); !ok {
		t.Fatalf("没有文件提及：%v", tt.Texts())
	}
	if _, ok := tt.Find("技能：deep-read"); !ok {
		t.Fatalf("没有技能提及：%v", tt.Texts())
	}
	if _, ok := tt.Find("提交 deadbeef"); !ok {
		t.Fatalf("没有提交提及：%v", tt.Texts())
	}
	if !tt.HasText("在浏览器打开：example.com") {
		t.Fatalf("没有链接提及：%v", tt.Texts())
	}
	// 文件提及点开右栏预览页。
	if err := tt.Click("打开文件：internal/ui/timeline.go"); err != nil {
		t.Fatalf("点不到文件提及：%v", err)
	}
	if !app.rightOpen || app.rightTab != tabFilePreview || app.rp().previewPath != "internal/ui/timeline.go" {
		t.Fatalf("文件提及没有打开预览：open=%v tab=%d path=%q",
			app.rightOpen, app.rightTab, app.rp().previewPath)
	}
	// 链接没有外部打开通道：入口置灰并写原因。
	info, ok := tt.Access("在浏览器打开：example.com（" + noteOpenWith + "）")
	if !ok {
		t.Fatalf("链接没有可读名：%v", tt.Texts())
	}
	if !info.Disabled {
		t.Fatalf("没有外部打开通道时链接仍可点：%+v", info)
	}
}

// 折叠态仍按原文显示（MaxLines 对整段生效），展开后给出提及。
func TestInlineMentionsAfterExpand(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	long := strings.Repeat("这是一段很长的正文。\n", 10) + "看 @internal/ui/timeline.go"
	app.rounds[0].user = long
	tt.Frame()
	if _, ok := tt.Find("打开文件：internal/ui/timeline.go"); ok {
		t.Fatal("折叠态不该把正文拆成提及")
	}
	if err := tt.Click("展示全部"); err != nil {
		t.Fatalf("点不到展示全部：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("打开文件：internal/ui/timeline.go"); !ok {
		t.Fatalf("展开后没有给出提及：%v", tt.Texts())
	}
}

// 站点图标批次：网页工具行按主机名画 16×16 的图标位，并写明本版不取 favicon。
func TestSiteIconBatchForWebTool(t *testing.T) {
	src := &fakeSource{
		conn: Connection{Phase: "connected"}, names: []string{"s1"}, hasSess: true,
		session: SessionSnapshot{ID: "s1", Rows: []RowSnapshot{
			{Kind: RowTool, Title: "抓取", Args: "https://example.com/a https://docs.example.com/b",
				Text: "读到了", Status: RowStatusDone},
		}},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("站点图标：example.com"); !ok {
		t.Fatalf("没有站点图标：%v", tt.Texts())
	}
	if _, ok := tt.Find("站点图标：docs.example.com"); !ok {
		t.Fatalf("第二个站点没有图标：%v", tt.Texts())
	}
	// 主机名只取 scheme 后、路径前的一段，并按出现顺序去重。
	if got := webHosts("https://example.com/a https://docs.example.com/b https://example.com/c"); len(got) != 2 ||
		got[0] != "example.com" || got[1] != "docs.example.com" {
		t.Fatalf("webHosts = %v，想要 [example.com docs.example.com]", got)
	}
	if !strings.Contains(noteSiteIcon, "站点图标") {
		t.Fatalf("站点图标缺少原因文案：%q", noteSiteIcon)
	}
}
