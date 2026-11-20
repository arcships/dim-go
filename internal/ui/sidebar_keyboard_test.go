package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是左栏会话列表的键盘取证：组内会话行是一个焦点组——Tab 进组一次、
// ↑/↓ 逐行移动、Home/End 到组内首末行、Enter / 空格打开。断言的是焦点落到哪
// 一行、以及打开后选中行变成谁，不是 hasRole 这类静态标注。
//
// 对应用例：R-LB-338（组内 ↑/↓ 移动、Enter 打开）、R-LB-538（Home/End 首末行）。

// focusRowByTab 按 Tab 直到焦点真的落在可读名 name 的元素自己身上（焦点停在
// 包着这一行的容器上时不算——那是左栏容器而不是行）。
func focusRowByTab(t *testing.T, tt *gui.Tester, name string, attempts int) bool {
	t.Helper()
	for i := 0; i < attempts; i++ {
		if focusOn(tt, name) {
			return true
		}
		tt.Key(0, gui.KeyTab)
	}
	return focusOn(tt, name)
}

// focusedRow 返回焦点所在的会话行名（不在任何会话行上时为空）。
func focusedRow(tt *gui.Tester, names ...string) string {
	for _, name := range names {
		if focusOn(tt, name) {
			return name
		}
	}
	return ""
}

// TestSidebarSessionRowsAreOneFocusGroup：Tab 进组一次停在组内第一行，↑/↓ 逐行
// 移动（不跨组、不跳到别的区域），Home/End 到组内首末行。
func TestSidebarSessionRowsAreOneFocusGroup(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	rows := []string{"修发送路径", "旧实现对照"}

	if !focusRowByTab(t, tt, "修发送路径", 80) {
		t.Fatalf("Tab 没有把焦点带进会话行组：%v", tt.Texts())
	}
	if got := focusedRow(tt, rows...); got != "修发送路径" {
		t.Fatalf("进组后焦点在 %q，想要组内第一行「修发送路径」", got)
	}

	// ↓ 逐行移动：焦点与选中行都跟着走到下一行。
	tt.Key(0, gui.KeyDown)
	if got := focusedRow(tt, rows...); got != "旧实现对照" {
		t.Fatalf("↓ 之后焦点在 %q，想要「旧实现对照」", got)
	}
	// ↑ 回到上一行。
	tt.Key(0, gui.KeyUp)
	if got := focusedRow(tt, rows...); got != "修发送路径" {
		t.Fatalf("↑ 之后焦点在 %q，想要「修发送路径」", got)
	}
	// End / Home 到组内末行 / 首行。
	tt.Key(0, gui.KeyEnd)
	if got := focusedRow(tt, rows...); got != "旧实现对照" {
		t.Fatalf("End 之后焦点在 %q，想要组内末行「旧实现对照」", got)
	}
	tt.Key(0, gui.KeyHome)
	if got := focusedRow(tt, rows...); got != "修发送路径" {
		t.Fatalf("Home 之后焦点在 %q，想要组内首行「修发送路径」", got)
	}
	// 移动只在组内：另一个工作区的会话行不在这条路径上。
	if focusOn(tt, "参考") {
		t.Fatal("方向键跨组跳到了别的工作区的会话行")
	}
	if app.selectedSession != "s1" {
		t.Fatalf("只移动焦点改动选中行：selectedSession = %q", app.selectedSession)
	}

	// 空格与 Enter 打开焦点所在行：选中行随之变化。
	tt.Key(0, gui.KeyEnd)
	tt.Key(0, gui.KeySpace)
	tt.Frame()
	if app.selectedSession != "s3" {
		t.Fatalf("空格没有打开焦点所在行：selectedSession = %q，想要 s3", app.selectedSession)
	}
	tt.Key(0, gui.KeyHome)
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.selectedSession != "s1" {
		t.Fatalf("Enter 没有打开焦点所在行：selectedSession = %q，想要 s1", app.selectedSession)
	}
}

// TestSidebarSessionRowGroupEntryIsOneTabStop：组内会话行只占一个 Tab 停靠点——
// 从最后一行再按 Tab 不会停在组内其它行上，而是离开这一组。
func TestSidebarSessionRowGroupEntryIsOneTabStop(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	if !focusRowByTab(t, tt, "修发送路径", 80) {
		t.Fatalf("Tab 没有把焦点带进会话行组：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyEnd)
	tt.Key(0, gui.KeyTab)
	if focusOn(tt, "修发送路径") || focusOn(tt, "旧实现对照") {
		t.Fatal("Tab 之后焦点仍在组内会话行上：这一组不止一个 Tab 停靠点")
	}
}
