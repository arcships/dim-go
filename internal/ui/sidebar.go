package ui

import (
	"strconv"

	gui "github.com/egoist/mygo/ui"
)

// sidebar 是 R1 左栏：入口区、会话列表（滚动）、账号与连接。
// 收起时留 12px 边条，点击边条展开。
//
// 窗口模式下，macOS 的红绿灯落在左栏顶部上方：顶部先留一条与控件同高的空带
// （c.TitleBar().Height），这一条也是拖动窗口的把手，控件不会压住任何可点元素。
// 全屏时 c.TitleBar() 为零，这条空带随之消失；标题行与主区不因此变动（UI-19）。
func (a *App) sidebar(c *gui.Context) {
	if a.sidebarCollapsed {
		a.sidebarRail(c)
		return
	}
	t := c.Theme()
	header := func() {
		if inset := titleBarInset(c.TitleBar()); inset > 0 {
			gui.Row(c).Height(inset).Shrink(0).DragWindow().Label("窗口控件让位")
		}
		// 入口只有图标（UI-23：界面语是最后手段）：新建、搜索、收起左栏。
		// 三个入口右对齐，读屏名照写，屏幕上不出现字。
		gui.Row(c).Height(sidebarHeaderHeight).Shrink(0).
			Padding(space1, space2, space1, space2).
			AlignItems(gui.Center).Gap(space1).Children(func() {
			gui.Spacer(c)
			iconButton(c, iconButtonSpec{icon: iconPlus, label: "新建会话", size: rowCompact})
			iconButton(c, iconButtonSpec{icon: iconSearch, label: "搜索", size: rowCompact})
			if iconButton(c, iconButtonSpec{icon: iconChevronL, label: "收起左栏", size: rowCompact}) {
				a.sidebarCollapsed = true
			}
		})
	}

	footer := func() {
		// 连接状态用状态点、账号用首字母头像（UI-23：先换形状与图形）；两者都
		// 保留读屏名，屏幕上不出现「已连接」「账号 · 已登录」这类界面语。
		gui.Row(c).Shrink(0).Padding(space3, space2, space3, space2).
			AlignItems(gui.Center).Gap(space2).Children(func() {
			statusDot(c, t.Success, a.connected)
			gui.Spacer(c)
			// 设置入口（P2）：齿轮图标，点开是独立成屏的设置。
			if iconButton(c, iconButtonSpec{icon: iconGear, label: "设置", size: rowCompact}) {
				a.screen = ScreenSettings
			}
			avatar(c, t, a.accountInitial)
		})
	}

	// 与主区之间靠面差（surface 对 background）分开，不画竖线（UI-18 第一档）。
	gui.Column(c).Width(SidebarWidth).Shrink(0).
		Background(t.Surface).
		Label("左栏 · 会话列表").Role(gui.RoleNavigation).Focusable().TabStop(true).
		Children(func() {
			header()
			a.sessionList(c)
			footer()
		})
}

// sessionList 用框架的 Sidebar：一个 Tab 停靠点，列表自身滚动，
// 上下键在会话之间移动。状态标记走固定的 16px icon 槽，不拼进 label，
// 因此带标记与不带标记的行文字左边界相同（U2 校准 C6）。
func (a *App) sessionList(c *gui.Context) {
	sessions, markers := sessionSamples(a.state)
	gui.Sidebar(c, &a.selectedSession, func() {
		// 新建与搜索已移进左栏头部的图标入口（UI-23），列表从这里直接分组。
		if len(sessions) == 0 {
			return
		}
		gui.Divider(c)
		gui.SidebarSection(c, "项目 A", nil, func() {
			for i := range sessions[:2] {
				a.sessionItem(c, "s"+strconv.Itoa(i+1), sessions[i], markers[i])
			}
		})
		gui.SidebarSection(c, "项目 B", nil, func() {
			a.sessionItem(c, "s3", sessions[2], markers[2])
		})
	}).Grow(1).Label("会话列表")
}

// sessionItem 画一条会话行：选中样式完全交给框架（聚焦 = accent 填充 +
// accent-on，未聚焦 = surface-pressed；U2 校准 C4 取框架语义）；状态标记走
// SidebarItem 的 16px icon 槽，所有会话行都占这个槽，文字左边界一致。
func (a *App) sessionItem(c *gui.Context, id, name, marker string) {
	icon := statusIcon(marker)
	if icon == nil {
		icon = iconDotEmpty // 只占位，不画：让所有会话行从同一列起
	}
	gui.SidebarItem(c, id, icon, name)
}

// markerColor 是侧栏状态标记的颜色：运行/等待用 accent-ink，失败用 danger。
func markerColor(t *gui.Theme, marker string) gui.Color {
	if marker == "✕" {
		return t.Danger
	}
	return colorAccentInk.in(t)
}

// sidebarRail 是收起后的 12px 边条：不显示会话，点击展开。
func (a *App) sidebarRail(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Width(SidebarRailWidth).Shrink(0).
		Background(t.Surface).
		AlignItems(gui.Center).Justify(gui.Center).Gap(space1).
		Label("展开左栏").Role(gui.RoleButton).Focusable().TabStop(true).
		OnClick(func() { a.sidebarCollapsed = false }).
		Children(func() {
			for i := 0; i < 3; i++ {
				gui.Box(c).Width(railMarkWidth).Height(railMarkHeight).
					Radius(railMarkHeight / 2).Background(t.TextMuted)
			}
		})
}
