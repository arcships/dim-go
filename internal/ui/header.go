package ui

import gui "github.com/egoist/mygo/ui"

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
		if a.model != "" {
			// 偏离默认的模型：最小形式，不写“模型：”前缀（UI-9）。
			typeSecondary.style(t, gui.Text(c, a.model)).TextColor(t.TextMuted).
				Label("模型 " + a.model)
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
