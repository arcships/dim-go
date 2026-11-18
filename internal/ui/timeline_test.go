package ui

import (
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
