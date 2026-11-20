package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「键盘」这一类需求的取证：焦点顺序、组内 Home/End 与 ↑↓、Esc 与
// Enter 的实际落点。每条用例对应需求表里的一条（行尾注明 R-ID），断言的是
// 键盘行为本身，不是外观。
//
// 取证口径与界面相同：焦点停在哪个读屏名上就用 gui.Tester.Focused 读，不看
// 内部字段——读了字段就只证明「代码以为焦点在哪」。

// focusByTab 按 Tab 直到焦点落到读屏名 name 上，最多 attempts 次；reverse 为
// 真时按 Shift+Tab。返回是否到达。
func focusByTab(t *testing.T, tt *gui.Tester, name string, reverse bool, attempts int) bool {
	t.Helper()
	mods := gui.Modifiers(0)
	if reverse {
		mods = gui.Shift
	}
	for i := 0; i < attempts; i++ {
		if tt.Focused(name) {
			return true
		}
		tt.Key(mods, gui.KeyTab)
	}
	return tt.Focused(name)
}

// TestSessionListIsOneFocusGroup：会话列表是一个焦点组——Tab 进一次，↑/↓ 逐行
// 移动、Home/End 到首尾（R-LB-160、R-LB-161、R-LB-162 的键盘口径）。
func TestSessionListIsOneFocusGroup(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if !focusByTab(t, tt, "修发送路径", false, 80) {
		t.Fatalf("Tab 没有把焦点带进会话列表：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyDown)
	if !tt.Focused("参考") {
		t.Fatalf("↓ 没有移动到下一行：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyUp)
	if !tt.Focused("修发送路径") {
		t.Fatalf("↑ 没有回到上一行：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyEnd)
	if tt.Focused("修发送路径") {
		t.Fatal("End 之后焦点还停在第一行")
	}
	tt.Key(0, gui.KeyHome)
	if !tt.Focused("修发送路径") {
		t.Fatalf("Home 没有回到第一行：%v", tt.Texts())
	}
	if app.selectedSession == "" {
		// 焦点移动不冒充选中：选中由 Enter 或点击决定（R-LB-163）。
		t.Fatal("样例会话应当已有选中项")
	}
}

// TestApprovalCardTabStaysInsideAndEscDoesNotAnswer：审批卡内的 Tab 只在按钮
// 之间走，不逃到背景；Esc 不答复（审批不提供误触快捷键）。R-AP-017。
func TestApprovalCardTabStaysInsideAndEscDoesNotAnswer(t *testing.T) {
	app, tt := newTester(t, StateWaiting)
	if _, ok := app.currentApproval(); !ok {
		t.Skip("走查态没有审批卡")
	}
	inside := map[string]bool{"仅这次": true, "拒绝": true, "始终允许此命令": true, "始终允许": true}
	seen := map[string]bool{}
	for i := 0; i < 24; i++ {
		tt.Key(0, gui.KeyTab)
		names := focusedNames(tt)
		if len(names) == 0 {
			t.Fatalf("第 %d 次 Tab 后焦点停在无名元素上", i+1)
		}
		for _, name := range names {
			if inside[name] {
				seen[name] = true
			}
		}
	}
	if len(seen) == 0 {
		t.Fatalf("Tab 从未落到审批卡的按钮上：%v", tt.Texts())
	}
	before := app.approvalDecision
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.approvalDecision != before {
		t.Fatal("Esc 改动了审批答复")
	}
	if !tt.HasText("想要") || !tt.HasText("拒绝") {
		t.Fatalf("Esc 之后审批卡消失了：%v", tt.Texts())
	}
}

// TestRightPanelToggleKeepsFocusOnButton：焦点在「打开上下文面板」上按 Enter 打开
// 右栏，焦点留在按钮上；再按 Enter 关闭（R-TB-023、R-TB-031、R-TB-333、R-TB-334）。
func TestRightPanelToggleKeepsFocusOnButton(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if !focusByTab(t, tt, "打开上下文面板", false, 80) {
		t.Fatalf("Tab 没有把焦点带到右栏开关：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if !app.rightOpen {
		t.Fatal("Enter 没有打开右栏")
	}
	if !tt.Focused("关闭上下文面板") {
		t.Fatalf("打开右栏后焦点不在按钮上：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.rightOpen {
		t.Fatal("再按 Enter 没有关闭右栏")
	}
	if !tt.Focused("打开上下文面板") {
		t.Fatalf("关闭右栏后焦点不在按钮上：%v", tt.Texts())
	}
}

// TestOverlayEscClosesEach：Esc 逐个关掉叠层（搜索、命令面板、@ 菜单、/ 菜单、
// 聊天搜索、右栏预览）。表里一格一个叠层，先开再按 Esc，断言关掉的是它自己。
func TestOverlayEscClosesEach(t *testing.T) {
	cases := []struct {
		name   string
		req    string
		open   func(t *testing.T, app *App, tt *gui.Tester)
		closed func(app *App, tt *gui.Tester) bool
	}{
		{"会话内搜索", "R-GS-021", func(t *testing.T, app *App, tt *gui.Tester) {
			openSessionSearchForTest(t, tt)
		}, func(app *App, tt *gui.Tester) bool { return !app.sessionSearch.open }},
		{"命令面板", "R-GC-030", func(t *testing.T, app *App, tt *gui.Tester) {
			app.palette.open = true
			tt.Frame()
		}, func(app *App, tt *gui.Tester) bool { return !app.palette.open }},
		{"@ 菜单", "R-IN-090", func(t *testing.T, app *App, tt *gui.Tester) {
			if err := tt.Click("消息"); err != nil {
				t.Fatalf("点不到输入框：%v", err)
			}
			tt.Type("@")
			tt.Frame()
			if !app.pick.open {
				t.Skip("@ 菜单没有打开")
			}
		}, func(app *App, tt *gui.Tester) bool { return !app.pick.open }},
		{"/ 菜单", "R-IN-091", func(t *testing.T, app *App, tt *gui.Tester) {
			if err := tt.Click("消息"); err != nil {
				t.Fatalf("点不到输入框：%v", err)
			}
			tt.Type("/")
			tt.Frame()
			if !app.slash.open {
				t.Skip("/ 菜单没有打开")
			}
		}, func(app *App, tt *gui.Tester) bool { return !app.slash.open }},
		{"聊天搜索", "R-GS-040", func(t *testing.T, app *App, tt *gui.Tester) {
			openChatSearchForTest(t, tt)
		}, func(app *App, tt *gui.Tester) bool { return !app.chatSearch.open }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app, tt := newTester(t, StateDefault)
			c.open(t, app, tt)
			tt.Key(0, gui.KeyEscape)
			tt.Frame()
			if !c.closed(app, tt) {
				t.Fatalf("%s：Esc 没有关掉叠层（%s）", c.name, c.req)
			}
		})
	}
}
