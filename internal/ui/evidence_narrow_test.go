package ui

import (
	"fmt"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「窄窗」（验收档 720）这一类需求的批量取证：一屏一条，断言的是这一屏
// 在 720 档的形态（整屏替代、满宽、顶栏下拉、控制行竖排），不是与 1440 档的像素差。
// 结构性的七处差异在 narrow_test.go；这里补的是「逐屏都成立」的那一层。

// TestNarrowRightPanelReplacesMainForEveryTab：720 档右栏每一页都整屏替代主区
// （不并排、不缩宽），面板占满边条右侧的全部宽度。对应各页的窄窗户型条目。
func TestNarrowRightPanelReplacesMainForEveryTab(t *testing.T) {
	want := float32(narrowWidth - SidebarRailWidth)
	for tab := 0; tab < tabCount; tab++ {
		t.Run(fmt.Sprintf("tab-%d", tab), func(t *testing.T) {
			app := NewApp(Options{State: StateDefault})
			app.rightOpen, app.rightTab = true, tab
			tt := gui.NewTester(app.View, narrowWidth, WindowHeight)
			tt.Frame()
			panel, ok := tt.Find("右栏")
			if !ok {
				t.Fatalf("窄窗右栏第 %d 页没有出现：%v", tab, tt.Texts())
			}
			if panel.X != SidebarRailWidth || panel.W != want {
				t.Fatalf("第 %d 页面板 %+v，想要 X=%v W=%v（整屏替代主区）",
					tab, panel, SidebarRailWidth, want)
			}
			if _, ok := tt.Find("对话流"); ok {
				t.Fatalf("第 %d 页打开时主区仍在屏上", tab)
			}
		})
	}
}

// TestNarrowMainKeepsRailAndStacksControls：720 档每个主界面状态都只留 12 的边条，
// 主区贴着边条占满其余宽度；输入区控制行竖排（工作区一行、权限一行）。
func TestNarrowMainKeepsRailAndStacksControls(t *testing.T) {
	for _, state := range []State{StateDefault, StateRunning, StateWaiting, StateFailed, StateEmpty} {
		t.Run(fmt.Sprintf("state-%d", state), func(t *testing.T) {
			app, tt := newNarrowTester(t, state)
			rail, ok := tt.Find("展开侧边栏 ⌘B")
			if !ok {
				t.Fatal("窄窗没有左栏边条")
			}
			if rail.W != SidebarRailWidth {
				t.Fatalf("边条宽 %v，想要 %v", rail.W, SidebarRailWidth)
			}
			if tt.HasText("左栏 · 会话列表") {
				t.Fatal("窄窗默认展开了 256 的左栏")
			}
			// 空态的主区是空态面板，其余状态是对话流；两者都贴着边条。
			main, ok := tt.Find("对话流")
			if !ok {
				main, ok = tt.Find("空态")
			}
			if !ok {
				t.Fatalf("找不到主区：%v", tt.Texts())
			}
			if main.X != SidebarRailWidth {
				t.Fatalf("主区左缘 %v，想要 %v", main.X, SidebarRailWidth)
			}
			if main.X+main.W > narrowWidth {
				t.Fatalf("主区右缘 %v 越出窗口 %d", main.X+main.W, narrowWidth)
			}
			if state != StateEmpty {
				return // 控制行只在空态输入区（工作区选择器）出现
			}
			workspace, ok := tt.Find("选择工作区")
			if !ok {
				t.Fatalf("找不到工作区选择器：%v", tt.Texts())
			}
			permission, ok := tt.Find("权限：" + app.permissionLabel())
			if !ok {
				t.Fatalf("找不到权限选择器：%v", tt.Texts())
			}
			if permission.Y < workspace.Y+workspace.H {
				t.Fatalf("控制行没有竖排：工作区 %+v、权限 %+v", workspace, permission)
			}
			if workspace.X != permission.X {
				t.Fatalf("竖排后两行左缘不同：%v / %v", workspace.X, permission.X)
			}
		})
	}
}

// TestNarrowSettingsSectionUsesFullWidth：720 档设置每一节都把 224 的常驻导航换
// 成顶部分节下拉，内容列占满窗口宽度；导航不再是左侧列表。
//
// 证据：R-SHL-039（19 节逐节成立）。
func TestNarrowSettingsSectionUsesFullWidth(t *testing.T) {
	for _, def := range settingsSectionDefs {
		t.Run(def.slug, func(t *testing.T) {
			app := NewApp(Options{State: StateDefault})
			app.screen, app.settingsSection = ScreenSettings, def.id
			tt := gui.NewTester(app.View, narrowWidth, WindowHeight)
			tt.Frame()

			nav, ok := tt.AccessByRole(gui.RolePopUpButton, "设置分节")
			if !ok {
				t.Fatalf("窄窗设置没有分节下拉：%v", tt.Texts())
			}
			if nav.Bounds.W < narrowWidth/2 {
				t.Fatalf("分节下拉宽 %v，应占满一行", nav.Bounds.W)
			}
			if _, ok := tt.AccessByRole(gui.RoleList, "设置分节"); ok {
				t.Fatal("窄窗仍画着 224 的常驻分节列表")
			}
			// 内容列从边距起占满一行：屏上第一个标题的右缘不越界。
			var found bool
			for _, text := range tt.Texts() {
				info, ok := tt.AccessByRole(gui.RoleHeading, text)
				if !ok {
					continue
				}
				found = true
				if info.Bounds.X < 0 || info.Bounds.X+info.Bounds.W > narrowWidth {
					t.Fatalf("标题 %q 越出窗口：%+v", info.Name, info.Bounds)
				}
				break
			}
			if !found {
				t.Fatalf("%s：这一节没有可读标题（内容列没铺开）", def.slug)
			}
			_ = app
		})
	}
}

// TestNarrowSessionSearchFitsWindow：720 档会话内搜索浮层不出窗口，命中行与
// 输入框都在窗口内（窄窗浮层按窗口宽度收）。
//
// 证据：R-GS-094。
func TestNarrowSessionSearchFitsWindow(t *testing.T) {
	app, tt := newNarrowTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	if !app.sessionSearch.open {
		t.Fatal("窄窗按 ⌘F 没有打开搜索")
	}
	box, ok := tt.Find(sessionSearchLabel)
	if !ok {
		t.Fatalf("找不到搜索输入框：%v", tt.Texts())
	}
	if box.X < 0 || box.X+box.W > narrowWidth {
		t.Fatalf("搜索输入框越出窗口：%+v", box)
	}
	for _, text := range tt.Texts() {
		info, ok := tt.Access(text)
		if !ok {
			continue
		}
		if info.Bounds.W == 0 {
			continue
		}
		if info.Bounds.X < -0.5 || info.Bounds.X+info.Bounds.W > narrowWidth+0.5 {
			t.Fatalf("浮层里的 %q 越出窗口：%+v", text, info.Bounds)
		}
	}
}

// TestNarrowCommandPaletteStaysInside：720 档命令面板满宽且不出窗口（R-GC-039）。
func TestNarrowCommandPaletteStaysInside(t *testing.T) {
	app, tt := newNarrowTester(t, StateDefault)
	tt.Key(gui.Cmd, gui.KeyK)
	tt.Frame()
	if !app.palette.open {
		t.Fatal("⌘K 没有打开命令面板")
	}
	panel, ok := tt.Find("命令面板")
	if !ok {
		t.Fatal("找不到命令面板")
	}
	if panel.X != narrowDialogMargin {
		t.Fatalf("命令面板左缘 %v，想要 %v", panel.X, narrowDialogMargin)
	}
	if panel.X+panel.W != narrowWidth-narrowDialogMargin {
		t.Fatalf("命令面板右缘 %v，想要 %v", panel.X+panel.W, narrowWidth-narrowDialogMargin)
	}
}
