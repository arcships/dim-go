package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 右栏四面板（终端 / 后台任务 / 侧聊 / 子代理）的走查断言：按清单与需求核对
// 在场元素、文案逐字与主要分支。像素与位置由截图（docs/domains/screens 的
// panel-*）负责，这里判结构与状态。

// panelTester 造一个右栏打开在指定标签上的界面。
func panelTester(t *testing.T, tab int) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: StateDefault})
	app.rightOpen, app.rightTab = true, tab
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// 终端：标签、新建 / 关闭终端、清屏、右键菜单五项（复制选中内容为禁用态）。
func TestTerminalPanelElements(t *testing.T) {
	app, tt := panelTester(t, tabTerminal)
	if _, ok := tt.Find("终端"); !ok {
		t.Fatal("工具条没有终端标签")
	}
	if _, ok := tt.Find(app.termTitle()); !ok {
		t.Fatalf("终端标签没有目录名 %q", app.termTitle())
	}
	for _, label := range []string{"关闭进程", "新建终端", "关闭终端", "清屏", "复制输出"} {
		if _, ok := tt.Access(label); !ok {
			t.Fatalf("终端工具条缺少 %q", label)
		}
	}
	// 提示行：环境名前缀 + 目录名。
	if _, ok := tt.Find(termEnvDefault); !ok {
		t.Fatalf("提示行没有环境名前缀 %q", termEnvDefault)
	}
	// 右键菜单：五项，两组分隔线，复制选中内容禁用。右键点在画布中间
	// （标签栏上也有「终端」两个字，按名字点会落到标签上）。
	tt.RightClickAt(WindowWidth-100, WindowHeight/2)
	want := []string{"复制选中内容", "粘贴", "-", "清屏", "新建终端", "-", "关闭进程"}
	got := tt.Menu()
	if len(got) != len(want) {
		t.Fatalf("右键菜单 = %v，想要 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("右键菜单第 %d 项 = %q，想要 %q（全表 %v）", i+1, got[i], want[i], got)
		}
	}
}

// 终端：关闭进程 → 提示行；新建终端 → 回到空画布与提示行。
func TestTerminalCloseAndNew(t *testing.T) {
	app, tt := panelTester(t, tabTerminal)
	if err := tt.Click("关闭进程"); err != nil {
		t.Fatalf("点不到关闭进程：%v", err)
	}
	tt.Frame()
	if !app.panels.termClosed {
		t.Fatal("关闭进程没有进入已关闭态")
	}
	if _, ok := tt.Find(termClosedNotice); !ok {
		t.Fatalf("已关闭态没有提示行 %q", termClosedNotice)
	}
	if err := tt.Click("新建终端"); err != nil {
		t.Fatalf("点不到新建终端：%v", err)
	}
	tt.Frame()
	if app.panels.termClosed {
		t.Fatal("新建终端后仍是已关闭态")
	}
	if _, ok := tt.Find(termEmptyText); !ok {
		t.Fatalf("新终端没有空画布说明 %q", termEmptyText)
	}
}

// 终端：清屏只清可见输出，不动状态与提示行。
func TestTerminalClearKeepsPrompt(t *testing.T) {
	app, tt := panelTester(t, tabTerminal)
	if len(app.termLines) == 0 {
		t.Fatal("样例终端没有输出行")
	}
	if err := tt.Click("清屏"); err != nil {
		t.Fatalf("点不到清屏：%v", err)
	}
	tt.Frame()
	if len(app.termLines) != 0 {
		t.Fatalf("清屏后仍有 %d 行", len(app.termLines))
	}
	if _, ok := tt.Find(termEmptyText); !ok {
		t.Fatal("清屏后没有空画布说明")
	}
	if _, ok := tt.Find(app.termTitle()); !ok {
		t.Fatal("清屏后提示行没了")
	}
}

// 任务：行在场（状态 + 名称 + 行尾停止）、空态两种、选中行给详情。
func TestTasksPanelListAndDetail(t *testing.T) {
	app, tt := panelTester(t, tabTasks)
	if _, ok := tt.Access("后台任务：go test ./... · 运行中"); !ok {
		t.Fatal("列表里没有运行中的任务行")
	}
	if _, ok := tt.Access("后台任务：npm run build · 已完成"); !ok {
		t.Fatal("列表里没有已完成的任务行")
	}
	if _, ok := tt.Access("取消任务"); !ok {
		t.Fatal("运行中的行没有停止入口")
	}
	// 选中第一行：详情出现命令行与输出框。
	if err := tt.Click("go test ./..."); err != nil {
		t.Fatalf("点不到任务行：%v", err)
	}
	tt.Frame()
	if app.panels.taskSel != 0 {
		t.Fatalf("选中行 = %d，想要 0", app.panels.taskSel)
	}
	if _, ok := tt.Find("$ go test ./..."); !ok {
		t.Fatal("详情没有命令行")
	}
	if _, ok := tt.Access("复制命令"); !ok {
		t.Fatal("详情没有复制命令入口")
	}
	if _, ok := tt.Access("后台输出"); !ok {
		t.Fatal("详情没有只读输出框")
	}
	// 停止走两步：第一次进待确认，第二次才发。
	if err := tt.Click("取消任务"); err != nil {
		t.Fatalf("点不到停止入口：%v", err)
	}
	tt.Frame()
	if app.panels.taskArmed != 0 {
		t.Fatalf("待确认的行 = %d，想要 0", app.panels.taskArmed)
	}
	if _, ok := tt.Access("确认停止"); !ok {
		t.Fatal("待确认态没有确认停止入口")
	}
}

// 任务：两个子标签各自的空态文案。
func TestTasksPanelEmptyStates(t *testing.T) {
	app, tt := panelTester(t, tabTasks)
	app.tasks = nil
	tt.Frame()
	if _, ok := tt.Find("暂无运行进程"); !ok {
		t.Fatal("后台进程空态文案不对")
	}
	if err := tt.Click("唤醒"); err != nil {
		t.Fatalf("点不到唤醒子标签：%v", err)
	}
	tt.Frame()
	if app.panels.taskKind != 1 {
		t.Fatalf("子标签 = %d，想要 1", app.panels.taskKind)
	}
	if _, ok := tt.Find("暂无待唤醒事项（" + noteSchedule + "）"); !ok {
		t.Fatal("唤醒空态文案不对")
	}
}

// 侧聊：空态文案、输入占位、发送与结束分支确认。
func TestSideChatPanelFlow(t *testing.T) {
	app, tt := panelTester(t, tabSideChat)
	if _, ok := tt.Find(sideChatEmptyThread); !ok {
		t.Fatal("侧聊没有空态文案")
	}
	input, ok := tt.Access("侧边对话输入")
	if !ok {
		t.Fatal("侧聊没有输入框")
	}
	if input.Description == "" && input.Name == "" {
		t.Fatal("输入框没有可读名")
	}
	in := input.Bounds
	app.panels.sideChatDraft = "只看 internal/ui 的发送路径"
	tt.Frame()
	// 主输入区也有一个「发送」，侧聊的发送按钮按位置点（输入框右端的圆钮）。
	sendX, sendY := in.X+in.W+space1+panelSendSize/2, in.Y+in.H/2
	tt.ClickAt(sendX, sendY)
	tt.Frame()
	if len(app.panels.sideChatSent) != 1 {
		t.Fatalf("发出 %d 条，想要 1", len(app.panels.sideChatSent))
	}
	if app.panels.sideChatDraft != "" {
		t.Fatal("发送后草稿没有清空")
	}
	if _, ok := tt.Find("只看 internal/ui 的发送路径"); !ok {
		t.Fatal("时间线里没有发出的消息")
	}
	// 结束分支：确认框 → 确认 → 状态条 + 输入框禁用。
	if err := tt.Click("结束分支"); err != nil {
		t.Fatalf("点不到结束分支：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find(sideChatEndTitle); !ok {
		t.Fatal("没有结束分支的确认框")
	}
	if err := tt.Click("确认"); err != nil {
		t.Fatalf("点不到确认：%v", err)
	}
	tt.Frame()
	if !app.panels.sideChatEnded {
		t.Fatal("确认后分支没有结束")
	}
	if _, ok := tt.Find(sideChatClosedNotice); !ok {
		t.Fatal("结束后没有状态条")
	}
	if info, ok := tt.Access("侧边对话输入"); !ok || !info.Disabled {
		t.Fatal("结束后输入框没有禁用")
	}
}

// 侧聊：错误横幅带重试，重试后横幅消失。
func TestSideChatErrorBanner(t *testing.T) {
	app, tt := panelTester(t, tabSideChat)
	app.panels.sideChatError = "读取侧边对话失败"
	tt.Frame()
	if _, ok := tt.Find("读取侧边对话失败"); !ok {
		t.Fatal("没有错误横幅")
	}
	if err := tt.Click(sideChatRetry); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	tt.Frame()
	if app.panels.sideChatError != "" {
		t.Fatal("重试后横幅没有消失")
	}
}

// 子代理：加载态、错误态与就绪后的只读时间线。
func TestSubagentPanelStates(t *testing.T) {
	app, tt := panelTester(t, tabSubagent)
	if _, ok := tt.Find(subagentLoadingText); !ok {
		t.Fatalf("没有加载态文案 %q", subagentLoadingText)
	}
	app.panels.subagentError = "读取子代理会话失败"
	tt.Frame()
	if _, ok := tt.Find("读取子代理会话失败"); !ok {
		t.Fatal("没有首屏错误文本")
	}
	if _, ok := tt.Access(subagentRetry); !ok {
		t.Fatal("首屏错误没有重试入口")
	}
	app.panels.subagentError = ""
	app.panels.subagentReady = true
	tt.Frame()
	if _, ok := tt.Find(subagentSamples()[0]); !ok {
		t.Fatal("就绪后没有只读时间线")
	}
	if _, ok := tt.Find(subagentLoadingText); ok {
		t.Fatal("就绪后仍显示加载文案")
	}
}
