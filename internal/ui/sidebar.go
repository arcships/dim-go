package ui

import (
	"strconv"

	gui "github.com/egoist/mygo/ui"
)

// sidebar 是 R1 左栏：入口区、会话列表（滚动）、账号与连接。
// 收起时留 12px 边条，点击边条展开。
func (a *App) sidebar(c *gui.Context) {
	if a.sidebarCollapsed {
		a.sidebarRail(c)
		return
	}
	t := c.Theme()
	header := func() {
		gui.Row(c).Height(sidebarHeaderHeight).Shrink(0).Padding(space1, space3).
			AlignItems(gui.Center).Gap(space2).Children(func() {
			typeSecondary.style(t, gui.Text(c, "会话")).TextColor(t.TextMuted)
			gui.Spacer(c)
			actionButton(c, "+", "新建会话", false, false)
			if actionButton(c, "‹", "收起左栏", false, false).Clicked() {
				a.sidebarCollapsed = true
			}
		})
	}

	footer := func() {
		gui.Column(c).Shrink(0).Padding(space3).Gap(space1).
			BorderWidth(strokeHairline, 0, 0, 0).BorderColor(t.Border).Children(func() {
			typeSecondary.style(t, gui.Text(c, "已连接")).TextColor(t.TextMuted).
				Label("连接状态")
			typeSecondary.style(t, gui.Text(c, "账号 · 已登录")).TextColor(t.TextMuted).
				Label("账号")
		})
	}

	gui.Column(c).Width(SidebarWidth).Shrink(0).
		Background(t.Surface).
		BorderWidth(0, strokeHairline, 0, 0).BorderColor(t.Border).
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
		gui.SidebarItem(c, "new", nil, "新会话")
		gui.SidebarItem(c, "search", nil, "搜索")
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
		BorderWidth(0, strokeHairline, 0, 0).BorderColor(t.Border).
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
