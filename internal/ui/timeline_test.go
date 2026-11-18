package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 状态条只属于会话工作区：有会话时在底部出现，空态没有。
func TestStatusBarAppearsWithSession(t *testing.T) {
	_, session := newTester(t, StateDefault)
	if _, ok := session.Find("状态条"); !ok {
		t.Fatal("有会话时没有状态条")
	}
	if _, ok := session.Find("权限：默认权限"); !ok {
		t.Fatal("状态条没有权限选择器")
	}
	if _, ok := session.Find("上下文用量 ~12%"); !ok {
		t.Fatal("状态条没有上下文用量")
	}

	_, empty := newTester(t, StateEmpty)
	if _, ok := empty.Find("状态条"); ok {
		t.Fatal("空态不应出现状态条")
	}
}

// 思考块两态：收起只有标题，展开显示思考原文；高度过渡按 0.3s 走完。
func TestThinkingBlockExpands(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if tt.HasText("用户要看失败原因。先读输入区与时间线，确认排队判定覆盖了草稿。") {
		t.Fatal("收起时不应显示思考原文")
	}
	if err := tt.Click("思考过程 思考了 0.3s"); err != nil {
		t.Fatalf("点不到思考块：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("用户要看失败原因。先读输入区与时间线，确认排队判定覆盖了草稿。") {
		t.Fatal("展开后缺少思考原文")
	}
	if p := app.blockOf("think:0", false); !*p {
		t.Fatal("思考块没有记为展开")
	}
}

// 工具分组：运行中的最新一轮默认展开，工具行带参数摘要与差异计数；
// 点一行展开输入详情。
func TestToolGroupShowsToolRowsAndDetails(t *testing.T) {
	_, tt := newTester(t, StateRunning)
	if !tt.HasText("composer.go") {
		t.Fatalf("展开的工具分组里没有工具行：%v", tt.Texts())
	}
	if !tt.HasText("+19") || !tt.HasText("-2") {
		t.Fatal("改动行没有差异计数")
	}
	if err := tt.Click("读取 composer.go"); err != nil {
		t.Fatalf("点不到工具行：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("path: internal/ui/composer.go") {
		t.Fatal("展开后没有工具输入")
	}
}

// 上下文用量的百分比与短形式：上限未知时不假装有数值。
func TestContextUsageFormatting(t *testing.T) {
	if got := contextPercent(24_600, 200_000); got != 12 {
		t.Fatalf("百分比 %d，想要 12", got)
	}
	if got := contextPercent(0, 0); got != 0 {
		t.Fatalf("未知上限的百分比 %d，想要 0", got)
	}
	if got := formatTokens(1_048_576); got != "1.0M" {
		t.Fatalf("formatTokens = %q", got)
	}
	if got := pctLabel(12); got != "~12%" {
		t.Fatalf("pctLabel = %q", got)
	}
}

// 悬停用户消息才出现时间与动作；气泡本身位置不动（TL-35、TL-37–TL-40）。
func TestUserActionsShowOnHover(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	if tt.HasText("复制整个消息 1") {
		t.Fatal("未悬停时不应出现消息动作")
	}
	bubble, ok := tt.Find("我的消息")
	if !ok {
		t.Fatal("找不到用户气泡")
	}
	tt.Move(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()
	if _, ok := tt.Find("复制整个消息 1"); !ok {
		t.Fatal("悬停后没有复制动作")
	}
	if _, ok := tt.Find("发送时间 星期二 21:22"); !ok {
		t.Fatal("悬停后没有发送时间")
	}
}

// 上下文用量浮层：点开后有本轮上下文、缓存命中与套餐用量三组文字。
func TestContextUsagePopover(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	if err := tt.Click("上下文用量 ~12%"); err != nil {
		t.Fatalf("点不到用量表：%v", err)
	}
	if !tt.HasText("本轮上下文") || !tt.HasText("缓存命中 98%") || !tt.HasText("套餐用量") {
		t.Fatalf("用量浮层内容不全：%v", tt.Texts())
	}
}

// 用户气泡最大宽是内容列的 85%：窄窗下重算，不越界。
func TestUserBubbleMaxWidth(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.rounds[0].user = "这是一条很长的用户消息，用来验证气泡宽度上限。" +
		"它应该折行，而不是撑满整条内容列，也不应该越过内容列右缘。"
	tt.Frame()
	bubble, ok := tt.Find("我的消息")
	if !ok {
		t.Fatal("找不到用户气泡")
	}
	content, ok := tt.Find("对话内容")
	if !ok {
		t.Fatal("找不到对话内容")
	}
	maxWidth := content.W * userBubbleMaxPercent / 100
	if bubble.W > maxWidth+1 {
		t.Fatalf("气泡宽 %v，超过内容列 85%%（%v）", bubble.W, maxWidth)
	}
	if bubble.X+bubble.W > content.X+content.W+1 {
		t.Fatalf("气泡越出内容列：%v + %v > %v", bubble.X, bubble.W, content.X+content.W)
	}
}

// 回到最新只在滚离底部后出现；点击后滚回底部。
func TestBackToLatestAppearsWhenScrolledAway(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.feed.Y = 0
	app.feed.MaxY = 400
	tt.Frame()
	if _, ok := tt.Find("回到最新"); !ok {
		t.Fatal("滚离底部后没有回到最新")
	}
	if err := tt.Click("回到最新"); err != nil {
		t.Fatalf("点不到回到最新：%v", err)
	}
	if !app.feed.Scrolling() {
		t.Fatal("点击后没有开始滚动")
	}
}

// 完全放开权限是高后果：先出确认对话框，勾选风险后才能确认。
func TestFullAccessNeedsConfirmation(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("权限：默认权限"); err != nil {
		t.Fatalf("点不到权限选择器：%v", err)
	}
	tt.Frame()
	if err := tt.Click("权限：完全访问权限"); err != nil {
		t.Fatalf("点不到完全访问权限：%v", err)
	}
	tt.Frame()
	if !app.permissionArmOpen {
		t.Fatal("选完全访问权限没有进入确认")
	}
	if _, ok := tt.Find("确认开启完全放开？"); !ok {
		t.Fatal("没有完全放开的确认对话框")
	}
	if _, ok := tt.Find("我已了解风险，并愿意继续"); !ok {
		t.Fatal("没有风险勾选")
	}
	// 未勾选时确认不生效。
	_ = tt.Click("确认开启")
	if app.permission == "完全访问权限" {
		t.Fatal("未确认就切到了完全访问")
	}
	if err := tt.Click("我已了解风险，并愿意继续"); err != nil {
		t.Fatalf("点不到风险勾选：%v", err)
	}
	tt.Frame()
	if err := tt.Click("确认开启"); err != nil {
		t.Fatalf("勾选后点不到确认：%v", err)
	}
	if app.permission != "完全访问权限" {
		t.Fatalf("确认后权限 %q，想要完全访问权限", app.permission)
	}
}

// 用户消息悬停动作（TL-37–TL-40）：复制进剪贴板、编辑回填输入区、侧边对话
// 打开右栏；分叉没有写入口，置灰并写原因。
func TestUserActionsDoWhatTheySay(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	bubble, ok := tt.Find("我的消息")
	if !ok {
		t.Fatal("找不到用户气泡")
	}
	tt.Move(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()

	if info, ok := tt.Access("从此处分叉 1（消息分叉还没有写入口）"); !ok || !info.Disabled {
		t.Fatalf("分叉没有按不可用画：%+v（找到 %v）", info, ok)
	}
	if _, ok := tt.Find("分叉不可用的原因"); !ok {
		t.Fatalf("置灰的分叉没写原因：%v", tt.Texts())
	}

	if err := tt.Click("复制整个消息 1"); err != nil {
		t.Fatalf("点不到复制：%v", err)
	}
	if got := tt.Clipboard(); got != app.rounds[0].user {
		t.Fatalf("剪贴板是 %q，想要 %q", got, app.rounds[0].user)
	}

	if err := tt.Click("编辑消息 1"); err != nil {
		t.Fatalf("点不到编辑：%v", err)
	}
	if app.draft != app.rounds[0].user {
		t.Fatalf("编辑没有回填输入区：%q", app.draft)
	}

	if err := tt.Click("从此处开始侧边对话 1"); err != nil {
		t.Fatalf("点不到侧边对话：%v", err)
	}
	if !app.rightOpen || app.rightTab != tabSideChat {
		t.Fatalf("侧边对话没有打开右栏：open=%v tab=%d", app.rightOpen, app.rightTab)
	}
	if !app.panels.sideChatFromMessage {
		t.Fatal("侧边对话没有记下从消息之前开始")
	}
}

// 消息右键菜单（TL-41–TL-43）：侧边对话、复制整个消息、编辑都在；
// 没有写入口的删除与撤回置灰并写原因。
func TestUserMessageContextMenu(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	bubble, ok := tt.Find("我的消息")
	if !ok {
		t.Fatal("找不到用户气泡")
	}
	tt.RightClickAt(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()
	menu := tt.Menu()
	for _, want := range []string{
		"从此处开始侧边对话", "复制整个消息", "编辑消息",
		"复制（" + messageCopyNeedsSelection + "）",
		"删除消息（" + messageDeleteUnavailable + "）",
		"撤回（" + messageWithdrawUnavailable + "）",
	} {
		if !slices.Contains(menu, want) {
			t.Fatalf("菜单缺 %q（实际 %v）", want, menu)
		}
	}
	if err := tt.ChooseMenuItem("复制（" + messageCopyNeedsSelection + "）"); err == nil {
		t.Fatal("没有选区却可选复制")
	}
	if err := tt.ChooseMenuItem("删除消息（" + messageDeleteUnavailable + "）"); err == nil {
		t.Fatal("删除没有写入口却可选")
	}
	tt.CloseMenu()

	tt.RightClickAt(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()
	if err := tt.ChooseMenuItem("编辑消息"); err != nil {
		t.Fatalf("点不到菜单编辑：%v", err)
	}
	if app.draft != app.rounds[0].user {
		t.Fatalf("菜单编辑没有回填输入区：%q", app.draft)
	}

	tt.RightClickAt(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()
	if err := tt.ChooseMenuItem("复制整个消息"); err != nil {
		t.Fatalf("点不到复制整个消息：%v", err)
	}
	if got := tt.Clipboard(); got != app.rounds[0].user {
		t.Fatalf("菜单复制的是 %q，想要 %q", got, app.rounds[0].user)
	}
}

// 选区浮条（TL-44）：选中正文后出现复制与引用两项；引用把选区以 Markdown
// 引用行填进输入区。
func TestSelectionBarCopiesAndQuotes(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	bubble, ok := tt.Find("我的消息")
	if !ok {
		t.Fatal("找不到用户气泡")
	}
	tt.ClickAt(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Key(gui.Cmd, gui.KeyA)
	tt.Frame()
	if _, ok := tt.Find("选区浮条"); !ok {
		t.Fatalf("选中正文后没有选区浮条：%v", tt.Texts())
	}

	if err := tt.Click("引用到输入区"); err != nil {
		t.Fatalf("点不到引用到输入区：%v", err)
	}
	if !strings.HasPrefix(app.draft, "> ") || !strings.Contains(app.draft, app.rounds[0].user) {
		t.Fatalf("引用没有填进输入区：%q", app.draft)
	}

	// 再选一次，验证复制选区走剪贴板。
	tt.ClickAt(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Key(gui.Cmd, gui.KeyA)
	tt.Frame()
	if err := tt.Click("复制选区"); err != nil {
		t.Fatalf("点不到复制选区：%v", err)
	}
	if got := tt.Clipboard(); got != app.rounds[0].user {
		t.Fatalf("复制选区得到 %q，想要 %q", got, app.rounds[0].user)
	}
}

// 工具详情三段（TL-54 的展开）：输入 / 输出 / 失败原因各有段名，内容不混。
func TestToolDetailHasThreeSections(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	header := "轮次组头 " + groupSummary(app.rounds[0]) + "，状态：" +
		statusWord(app.rounds[0].status) + "，" + roundHeaderPhrase(app.rounds[0])
	if err := tt.Click(header); err != nil {
		t.Fatalf("点不到工具组头：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if err := tt.Click("执行 go test ./..."); err != nil {
		t.Fatalf("点不到失败的工具行：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	for _, want := range []string{"输入", "输出", "失败原因"} {
		if _, ok := tt.Find("工具详情：" + want); !ok {
			t.Fatalf("工具详情缺 %q 段：%v", want, tt.Texts())
		}
	}
	if !tt.HasText("command: go test ./...") || !tt.HasText("$ go test ./...") {
		t.Fatalf("输入与输出内容没有分别上屏：%v", tt.Texts())
	}
	if !tt.HasText("权限不足，命令没有执行") {
		t.Fatal("失败原因没有上屏")
	}
}

// 附件与引用 chip（TL-21–TL-26）：气泡里按数据画图片、chip、引用、标注与
// 定时来源；没有数据时一个都不画。
func TestUserMessageAttachmentsAndQuotes(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if _, ok := tt.Find("消息附件与引用"); ok {
		t.Fatal("样例没有附件事实，却画了附件区")
	}
	app.rounds[0].images = []string{"屏幕.png"}
	app.rounds[0].attachments = []messageAttachment{
		{name: "需求.md"},
		{name: "旧日志.txt", missing: true},
	}
	app.rounds[0].quotes = []messageQuote{{title: "引用 1", text: "发送路径要排队"}}
	app.rounds[0].marks = []messageMark{{num: 1, text: "这一行是标注"}}
	app.rounds[0].wakeup = "定时唤醒：每天 9:00"
	tt.Frame()
	for _, want := range []string{
		"消息附件与引用", "图片附件 屏幕.png", "附件 需求.md",
		"附件 旧日志.txt（不可用）", "引用 引用 1", "定时唤醒来源",
	} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("缺少 %q：%v", want, tt.Texts())
		}
	}
	if !tt.HasText("这一行是标注") {
		t.Fatalf("标注正文没有上屏：%v", tt.Texts())
	}
}

// quoteIntoDraft 只加一层引用；空草稿不留前导空行。
func TestQuoteIntoDraft(t *testing.T) {
	if got := quoteIntoDraft("", "第一段\n第二段"); got != "> 第一段\n> 第二段" {
		t.Fatalf("空草稿的引用是 %q", got)
	}
	if got := quoteIntoDraft("已有内容", "引用"); got != "已有内容\n\n> 引用" {
		t.Fatalf("已有内容时的引用是 %q", got)
	}
}
