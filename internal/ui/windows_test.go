package ui

import (
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// TestOpenCurrentSessionInNewWindow 核对标题栏 ⋯ 的「在新窗口中打开」：会话标识
// 交给入口的窗口管理，回执写在标题栏，不自己冒充开了窗口。
func TestOpenCurrentSessionInNewWindow(t *testing.T) {
	var called []string
	src := &fakeSource{
		conn:    Connection{Phase: "connected"},
		names:   []string{"sess-a", "sess-b"},
		session: SessionSnapshot{ID: "sess-a", Title: "发送路径"},
		hasSess: true,
	}
	app := NewApp(Options{
		State:  StateDefault,
		Source: src,
		OpenInNewWindow: func(sessionID string) (string, error) {
			called = append(called, sessionID)
			return "已在新窗口中打开", nil
		},
	})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	if err := tt.Click("更多操作"); err != nil {
		t.Fatalf("点不到 ⋯：%v", err)
	}
	tt.Frame()
	if err := tt.Click("在新窗口中打开"); err != nil {
		t.Fatalf("点不到「在新窗口中打开」：%v", err)
	}
	tt.Frame()
	if len(called) != 1 || called[0] != "sess-a" {
		t.Fatalf("交给窗口管理的会话 = %v，想要 [sess-a]", called)
	}
	if _, ok := tt.Find("已在新窗口中打开"); !ok {
		t.Fatal("标题栏没有写回执")
	}
}

// TestOpenRowSessionInNewWindow 核对左栏行菜单的「在新窗口中打开」：交出去的是
// 这一行的会话标识，不是当前选中的那一条。
func TestOpenRowSessionInNewWindow(t *testing.T) {
	var called []string
	app := NewApp(Options{
		State: StateDefault,
		OpenInNewWindow: func(sessionID string) (string, error) {
			called = append(called, sessionID)
			return "已在新窗口中打开", nil
		},
	})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	// 悬停第二行，点行尾的「更多」。
	row, ok := tt.Find("参考")
	if !ok {
		t.Fatal("找不到第二行会话")
	}
	tt.Move(row.X+row.W-sidebarActionSize, row.Y+row.H/2)
	tt.Frame()
	if err := tt.Click("会话操作：参考"); err != nil {
		t.Fatalf("点不到行尾动作入口：%v", err)
	}
	tt.Frame()
	if err := tt.Click("在新窗口中打开"); err != nil {
		t.Fatalf("点不到「在新窗口中打开」：%v", err)
	}
	tt.Frame()
	if len(called) != 1 || called[0] != "参考" {
		t.Fatalf("交给窗口管理的会话 = %v，想要 [参考]", called)
	}
}

// TestOpenInNewWindowWithoutManager 核对没有窗口管理时（界面测试 / 单屏截图）
// 给的是可读原因，而不是静默什么也不做。
func TestOpenInNewWindowWithoutManager(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	tt.Frame()
	if err := tt.Click("更多操作"); err != nil {
		t.Fatalf("点不到 ⋯：%v", err)
	}
	tt.Frame()
	if err := tt.Click("在新窗口中打开"); err != nil {
		t.Fatalf("点不到「在新窗口中打开」：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find(noteMultiWindow); !ok {
		t.Fatalf("没有窗口管理时应写回执 %q", noteMultiWindow)
	}
	if app.headerNote != noteMultiWindow {
		t.Fatalf("回执行 = %q，想要 %q", app.headerNote, noteMultiWindow)
	}
}

// TestShowSettingsSectionBySlug 核对深链的分节落点：已知标识进设置屏的那一节，
// 未知标识给可读原因且不改屏。
func TestShowSettingsSectionBySlug(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	tt.Frame()

	if err := app.ShowSettingsSection("appearance"); err != nil {
		t.Fatalf("已知分节应能打开：%v", err)
	}
	if app.screen != ScreenSettings || app.settingsSection != secAppearance {
		t.Fatalf("屏幕 = %v、分节 = %v，想要设置 › 外观", app.screen, app.settingsSection)
	}
	tt.Frame()
	if _, ok := tt.Find("设置"); !ok {
		t.Fatal("没有进设置屏")
	}

	// 空分节按第一节处理（dimagent://settings）。
	if err := app.ShowSettingsSection(""); err != nil {
		t.Fatalf("空分节应回落到第一节：%v", err)
	}
	if app.settingsSection != secAccount {
		t.Fatalf("空分节落在 %v，想要账户", app.settingsSection)
	}

	app.screen, app.settingsSection = ScreenMain, secAccount
	err := app.ShowSettingsSection("no-such-section")
	if err == nil || !strings.Contains(err.Error(), "未知的设置分节") {
		t.Fatalf("未知分节的原因 = %v", err)
	}
	if app.screen != ScreenMain {
		t.Fatal("未知分节不该改屏")
	}
}
