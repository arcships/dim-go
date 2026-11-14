package ui

import (
	"context"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// header 是 R2 标题行，同时也是窗口的标题栏（UI-19）：窗口控件（macOS 的红绿灯、
// Windows 的最小化/最大化/关闭）由系统画在我们这一行的上方，界面不再保留系统
// 标题栏，也不在旁边再画一条自己的标题栏。
//
// 高度取 max(c.TitleBar().Height, 48)：窗口模式里红绿灯要占的那条在 48 之内，
// 全屏时 c.TitleBar() 为 0，高度仍是 48，布局不跳变。空白区是拖动窗口的把手，
// 双击最大化由框架给。
//
// 会话名在左端写一次，是界面上唯一的可见标题（红绿灯让位之后；收起左栏时按
// c.TitleBar().Left 让位，不被控件压住）。窗口 Title 给任务栏、Mission Control
// 与窗口切换器用，不是"界面上再写一遍"（§9.0 A1/D1：这一行服务"我在哪"）。
//
// 分支、权限、模型取默认值时不出现（UI-9）；偏离默认的模型只在行末留一个淡
// 标记，不占输入区。上下文用量不在这一行，它属于输入区（UI-15）。空态也写会话名
// （给的是"新会话"），标题行不留空行。
func (a *App) header(c *gui.Context) {
	t := c.Theme()
	bar := c.TitleBar()
	name := "修发送路径"
	if a.state == StateEmpty {
		name = "新会话"
	}

	// 左内边距让开窗口控件：红绿灯落在左栏上方时这一行不必让，左栏收起为
	// 12px 边条时控件会压到这一行的左缘。
	left := float32(space4)
	if a.sidebarCollapsed && bar.Left > left {
		left = bar.Left
	}
	right := float32(space4)
	if bar.Right > right {
		right = bar.Right
	}

	gui.Row(c).Height(titleBarHeight(bar)).Shrink(0).
		Padding(0, right, 0, left).
		Gap(space2).AlignItems(gui.Center).DragWindow().
		Label("标题行").Role(gui.RoleGroup).Children(func() {
		// 读屏名带上"会话"，与左栏的"新会话"这类行文区分开（同一段文字在别处
		// 可能指动作，不是标题）。
		title := typeTitleRow.style(t, gui.Text(c, name)).SingleLine().
			Label("会话 " + name)
		if a.state == StateEmpty {
			title.TextColor(t.TextMuted)
		}
		gui.Spacer(c)
		if a.state == StateEmpty {
			return
		}
		// 分支：样例里 main 不上屏（UI-9）；接上事实时始终给入口，便于切出/结束。
		currentBranch, _, wiredBranch := a.branchState()
		if wiredBranch || (currentBranch != "" && currentBranch != "main") {
			a.branchChip(c)
		}
		if model := a.currentModel(); model != "" {
			// 偏离默认的模型：最小形式，不写“模型：”前缀（UI-9）。
			typeSecondary.style(t, gui.Text(c, model)).TextColor(t.TextMuted).
				Label("模型 " + model)
		}
		label := "打开右栏"
		if a.rightOpen {
			label = "收起右栏"
		}
		if iconMenuButton(c, iconPanelRight, label, a.rightOpen) {
			a.rightOpen = !a.rightOpen
		}
	})
}

// titleBarHeight 是标题行的高度：窗口控件占的那条与固定 48 取大者。全屏时
// 控件隐藏（c.TitleBar() 为零），高度仍是 48，主区不因此跳动。
func titleBarHeight(bar gui.TitleBar) float32 {
	if bar.Height > HeaderHeight {
		return bar.Height
	}
	return HeaderHeight
}

// titleBarInset 是左栏顶部让给窗口控件的高度：窗口模式里等于控件那条的高度，
// 全屏时为 0（控件已隐藏）。它只影响左栏顶部那一条空带，不影响别的区域。
func titleBarInset(bar gui.TitleBar) float32 { return bar.Height }

// branchChip 是标题行里的分支标记：分叉图标 + 分支名，点开是分支清单（框架的原生
// 弹出菜单，不占版式）。清单里当前分支打勾；「结束分支」是高后果（AC-08）：
// 选它只进入确认态——标记换成「确认结束分支？」并出现「取消」，再点标记才生效。
func (a *App) branchChip(c *gui.Context) {
	t := c.Theme()
	current, branches, wired := a.branchState()

	if a.endBranchConfirm {
		b := gui.ButtonBase(c).Padding(space1, space2).Radius(radius).
			Cursor(gui.CursorPointer).Label("确认结束分支").
			Tooltip("再点一次：结束这条分支，之后不能再提交")
		if b.Hovered() {
			b = b.Background(t.SurfaceHover)
		}
		if b.Clicked() {
			if wired && a.intents != nil {
				ended := current
				a.dispatch("结束分支", func(ctx context.Context) error {
					return a.intents.EndBranch(ctx, ended)
				}, func() { a.endBranchConfirm = false })
			} else {
				ended := a.branch
				a.branch = "main"
				kept := a.branches[:0]
				for _, name := range a.branches {
					if name != ended {
						kept = append(kept, name)
					}
				}
				a.branches = kept
				a.endBranchConfirm = false
			}
		}
		b.Children(func() {
			iconGlyph(c, iconBranch, t.Danger, "branch")
			typeSecondary.style(t, gui.Text(c, "确认结束分支？")).TextColor(t.Danger).SingleLine()
		})
		if actionButton(c, "取消", "取消结束分支", false, false).Clicked() {
			a.endBranchConfirm = false
		}
		return
	}

	label := "切换分支：" + current
	b := gui.ButtonBase(c).Padding(space1, space2).Radius(radius).
		Cursor(gui.CursorPointer).Label(label).Tooltip("分支 · 点开切换")
	b = b.PopoverMenuButton(func(m *gui.Menu) {
		for _, name := range branches {
			n := name
			if m.Item(n).Checked(n == current).Chosen() {
				if wired && a.intents != nil {
					a.dispatch("切换分支", func(ctx context.Context) error {
						return a.intents.SelectBranch(ctx, n)
					}, nil)
				} else {
					a.branch = n
				}
			}
		}
		m.Separator()
		if m.Item("从当前分支切出").Chosen() {
			if wired && a.intents != nil {
				snap, _ := a.liveSession()
				prompt := strings.TrimSpace(a.draft)
				if prompt == "" {
					prompt = "从当前分支切出"
				}
				parent, turn := current, snap.TurnCount
				a.dispatch("切出分支", func(ctx context.Context) error {
					return a.intents.CreateBranch(ctx, parent, turn, prompt)
				}, nil)
			} else {
				a.branch = a.branch + "-1"
			}
		}
		if m.Item("结束分支").Disabled(current == "main").Chosen() {
			a.endBranchConfirm = true
		}
	}, gui.PopoverMenuButtonOptions{Open: &a.branchMenuOpen})
	if b.Hovered() {
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		iconGlyph(c, iconBranch, t.TextMuted, "branch")
		typeSecondary.style(t, gui.Text(c, current)).TextColor(t.TextMuted).SingleLine()
	})
}
