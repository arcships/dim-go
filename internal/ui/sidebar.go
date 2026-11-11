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
		gui.Row(c).Height(sidebarHeaderHeight).Shrink(0).Padding(spaceXS, spaceM).
			AlignItems(gui.Center).Gap(spaceS).Children(func() {
			gui.Text(c, "会话").TextColor(t.TextMuted).FontSize(t.FontSize - smallFontSteps)
			gui.Spacer(c)
			gui.Button(c, "＋").Label("新建会话").Focusable()
			if gui.Button(c, "‹").Label("收起左栏").Focusable().Clicked() {
				a.sidebarCollapsed = true
			}
		})
	}

	footer := func() {
		gui.Column(c).Shrink(0).Padding(spaceM).Gap(spaceXXS).
			BorderWidth(hairline, 0, 0, 0).BorderColor(t.Border).Children(func() {
			gui.Text(c, "已连接").TextColor(t.TextMuted).
				FontSize(t.FontSize - smallFontSteps).Label("连接状态")
			gui.Text(c, "账号 · 已登录").TextColor(t.TextMuted).
				FontSize(t.FontSize - smallFontSteps).Label("账号")
		})
	}

	gui.Column(c).Width(SidebarWidth).Shrink(0).
		Background(t.Surface).BorderWidth(0, hairline, 0, 0).BorderColor(t.Border).
		Label("左栏 · 会话列表").Role(gui.RoleNavigation).Focusable().TabStop(true).
		Children(func() {
			header()
			a.sessionList(c)
			footer()
		})
}

// sessionList 用框架的 Sidebar：一个 Tab 停靠点，列表自身滚动，
// 上下键在会话之间移动。行首标记按状态取一个，不新增状态容器。
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
				label := sessionLabel(sessions[i], markers[i])
				gui.SidebarItem(c, "s"+strconv.Itoa(i+1), nil, label)
			}
		})
		gui.SidebarSection(c, "项目 B", nil, func() {
			label := sessionLabel(sessions[2], markers[2])
			gui.SidebarItem(c, "s3", nil, label)
		})
	}).Grow(1).Label("会话列表")
}

// sidebarRail 是收起后的 12px 边条：不显示会话，点击展开。
func (a *App) sidebarRail(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Width(SidebarRailWidth).Shrink(0).
		Background(t.Surface).BorderWidth(0, hairline, 0, 0).BorderColor(t.Border).
		AlignItems(gui.Center).Justify(gui.Center).Gap(spaceXS).
		Label("展开左栏").Role(gui.RoleButton).Focusable().TabStop(true).
		OnClick(func() { a.sidebarCollapsed = false }).
		Children(func() {
			for i := 0; i < 3; i++ {
				gui.Box(c).Width(railMarkWidth).Height(railMarkHeight).Radius(1).Background(t.TextMuted)
			}
		})
}

func sessionLabel(name, marker string) string {
	if marker == "" {
		return name
	}
	return marker + " " + name
}
