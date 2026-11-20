package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件覆盖标题栏与状态条这一侧的三件事：会话名截断后的悬停全名、标题栏
// 分支标记的菜单（切换 / 新建）、以及权限与用量两个浮层的键盘收起。

// TestHeaderSessionTitleKeepsFullName：屏上会话名单行截断，完整名字留在悬停
// 提示与读屏名里，截断不影响读到的内容（TB-19）。
func TestHeaderSessionTitleKeepsFullName(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	name := app.sessionTitle()
	info, ok := tt.Access("会话 " + name)
	if !ok {
		t.Fatalf("标题栏没有会话名：%v", tt.Texts())
	}
	if info.Description != name {
		t.Fatalf("会话名的提示是 %q，想要全名 %q", info.Description, name)
	}

	// 窄窗：会话名给上限宽，仍读得到全名。
	app.sidebarCollapsed = true
	tt.Frame()
	info, ok = tt.Access("会话 " + name)
	if !ok {
		t.Fatalf("窄窗下没有会话名：%v", tt.Texts())
	}
	if info.Bounds.W > narrowTitleMaxWidth+1 {
		t.Fatalf("窄窗会话名宽 %v，超过上限 %v", info.Bounds.W, float32(narrowTitleMaxWidth))
	}
	if info.Description != name {
		t.Fatal("窄窗截断后读不到全名")
	}
}

// TestHeaderBranchMarkerMenu：标题栏分支标记点开是分支菜单，能切换与新建；
// 新建沿用「{当前}-1」的样例规则，切完标记跟着变（TB-19 的分支入口）。
func TestHeaderBranchMarkerMenu(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.branch = "feature/queue"
	app.branches = []string{"main", "feature/queue"}
	tt.Frame()
	if err := tt.Click("分支 feature/queue"); err != nil {
		t.Fatalf("点不到标题栏分支标记：%v", err)
	}
	tt.Frame()
	if !tt.HasText("创建新分支") {
		t.Fatalf("分支标记没有菜单：%v", tt.Texts())
	}
	if err := tt.Click("分支：main"); err != nil {
		t.Fatalf("点不到另一个分支：%v", err)
	}
	tt.Frame()
	if app.branch != "main" {
		t.Fatalf("切分支后是 %q，想要 main", app.branch)
	}
	// main 在样例里按 UI-9 不上屏：标记消失（不画一个恒亮的分支芯片）。
	if _, ok := tt.Find("分支 main"); ok {
		t.Fatal("main 分支不该上屏")
	}
}

// TestHeaderBranchMarkerCreatesBranch：分支菜单的「创建新分支」按当前分支加一条，
// 并切到新分支上。
func TestHeaderBranchMarkerCreatesBranch(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.branch = "feature/queue"
	app.branches = []string{"main", "feature/queue"}
	tt.Frame()
	if err := tt.Click("分支 feature/queue"); err != nil {
		t.Fatalf("点不到标题栏分支标记：%v", err)
	}
	tt.Frame()
	if err := tt.Click("创建新分支"); err != nil {
		t.Fatalf("点不到创建新分支：%v", err)
	}
	tt.Frame()
	if app.branch != "feature/queue-1" {
		t.Fatalf("新建后分支是 %q，想要 feature/queue-1", app.branch)
	}
}

// TestStatusPopoversCloseOnEscape：权限与用量两个浮层都能用 Esc 收起，收起后
// 触发按钮回到原样（键盘可达的开合口径）。
func TestStatusPopoversCloseOnEscape(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	if err := tt.Click("上下文用量 ~12%"); err != nil {
		t.Fatalf("点不到用量表：%v", err)
	}
	tt.Frame()
	if !tt.HasText("本轮上下文") {
		t.Fatalf("用量浮层没打开：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if tt.HasText("本轮上下文") {
		t.Fatal("Esc 没有收起用量浮层")
	}

	if err := tt.Click("权限：默认权限"); err != nil {
		t.Fatalf("点不到权限档位：%v", err)
	}
	tt.Frame()
	if !tt.HasText("完全访问权限") {
		t.Fatalf("权限菜单没打开：%v", tt.Texts())
	}
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if tt.HasText("完全访问权限") {
		t.Fatal("Esc 没有收起权限菜单")
	}
}
