package ui

import (
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 设置壳的两条尺寸不是间距档位，是窗框留白，故单列常量并写明推导：
// 参考产品的设置面是窗框里的一块白面（1030×725 视口上量得白面从 (4,4) 起、
// 到 (1026,721) 止，四周 4），导航与内容之间一条 1px 分隔线（导航最后一像素）。
const (
	settingsShellInset = 4
	settingsShellRule  = 1
)

// settings 是 P2 设置屏（独立成屏，U1-ia §4）：左导航 224 + 内容区（上限 1008）。
// 导航是参考产品的 19 节（settings_sections.go）；内容按标识路由：已重建的分节
// 显示真内容，其余显示占位，不摆编造的控件。
//
// 面差：整块设置面是 t.Surface（白面 / 暗色 0c0c0c），窗框留 settingsShellInset
// 让出 t.Background。参考产品同构——导航与内容同在白面上，两者之间靠 1px 的
// t.Border 分开（同色面之间必须有这条线，否则导航列看不出边界）。
//
// 顶部让位与标题栏一致：窗口控件（红绿灯）落在导航列上方的那条里（UI-19），
// 导航列顶部留出 c.TitleBar().Height，空白区可拖动窗口。
func (a *App) settings(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Fill().AlignItems(gui.Stretch).
		Background(t.Background).Padding(settingsShellInset).Label("设置").Children(func() {
		gui.Row(c).Fill().AlignItems(gui.Stretch).Radius(radius).
			Background(t.Surface).Children(func() {
			a.settingsNav(c)
			gui.Box(c).Width(settingsShellRule).Shrink(0).
				Background(t.Border).Label("导航与内容的分隔")
			gui.Column(c).Grow(1).MinWidth(MainMinWidth).Children(func() {
				gui.Column(c).Grow(1).MaxWidth(SettingsContentMaxWidth).
					Padding(space8, space8).Gap(space4).Children(func() {
					a.actionErrorView(c)
					a.settingsContent(c)
				})
			})
		})
	})
}

// settingsContent 按当前分节路由内容：已重建的分节显示真内容，其余一律占位。
func (a *App) settingsContent(c *gui.Context) {
	section := settingsSectionByID(a.settingsSection)
	switch section.id {
	case secAccount:
		a.settingsAccountScreen(c)
	case secUsage:
		renderSettingsUsageScreen(c)
	case secPersonalization:
		renderSettingsPersonalizationScreen(c)
	case secAppearance:
		renderAppearance(c)
	case secGeneral:
		renderGeneral(c)
	case secKeybindings:
		a.renderKeybindings(c)
	case secProviders:
		renderConfiguration(c, a)
	case secModality:
		renderModality(c)
	case secRuntime:
		renderRuntime(c)
	case secMCP:
		renderMCP(c, a)
	case secNotifications:
		renderNotifications(c)
	case secPets:
		renderPets(c)
	case secPermissions:
		renderPermissions(c)
	case secGit:
		renderGit(c)
	case secHooks:
		renderHooks(c, a)
	case secPluginSettings:
		renderPluginSettings(c)
	case secExternalAgents:
		renderExternalAgents(c)
	case secArchived:
		renderArchived(c)
	case secAbout:
		renderAbout(c)
	default:
		settingsPlaceholder(c, section)
	}
}

// settingsPlaceholder 是尚未重建的分节：只有分节标题与一句状态。这里不放任何
// 控件——没有实现的分节不摆看起来能用的东西（占位不是假的设置面）。
func settingsPlaceholder(c *gui.Context, section settingsSectionDef) {
	t := c.Theme()
	settingsSectionTitle(c, section.label)
	typeSecondary.style(t, gui.Text(c, "尚未按参考产品重建")).
		TextColor(t.TextMuted)
}

// settingsNav 是左导航：返回主界面的图标入口 + 19 节分节列表。
//
// 导航列自己不铺底色：白面由设置壳铺（同一块面），导航再铺一次会把圆角的
// 四个内角盖成直角。左右留 space2：参考产品的分节行宽 210（224 − 2×7）。
func (a *App) settingsNav(c *gui.Context) {
	gui.Column(c).Width(SettingsNavWidth).Shrink(0).
		Padding(0, space2).Children(func() {
		if inset := titleBarInset(c.TitleBar()); inset > 0 {
			gui.Row(c).Height(inset).Shrink(0).DragWindow().Label("窗口控件让位")
		}
		gui.Row(c).Height(sidebarHeaderHeight).Shrink(0).
			Padding(space1, space2).AlignItems(gui.Center).Gap(space1).Children(func() {
			if iconButton(c, iconButtonSpec{icon: iconChevronL, label: "返回主界面", size: rowCompact}) {
				a.screen = ScreenMain
			}
			gui.Spacer(c)
		})
		// 19 节在矮窗口里放不下：导航列自己滚，内容区不受影响。
		gui.Scroll(c).Grow(1).Label("设置分节").Children(func() {
			gui.Column(c).Shrink(0).Children(func() {
				for _, section := range settingsSectionDefs {
					a.navRow(c, section)
				}
			})
		})
	})
}

// navRow 是导航里的一节：图标 + 名称。自绘行（无外观 ButtonBase），选中态由
// 底色与字重表达（参考产品同构：选中用强调底色与前景色，其余用次要文字色）。
// 行高取 SettingsNavRowHeight（35）：相邻两节的文字 y 步进与行高同值，行间不再留缝。
func (a *App) navRow(c *gui.Context, section settingsSectionDef) {
	t := c.Theme()
	row := gui.ButtonBase(c).Height(SettingsNavRowHeight).Padding(0, space3).
		Radius(radius).Gap(space3).AlignItems(gui.Center).
		Label(section.label).Cursor(gui.CursorPointer).Justify(gui.Start)
	selected := a.settingsSection == section.id
	switch {
	case selected:
		row = row.Background(t.SurfacePressed)
	case row.Hovered():
		row = row.Background(t.SurfaceHover)
	}
	if row.Clicked() {
		a.settingsSection = section.id
	}
	ink := t.TextMuted
	style := typeControl
	if selected {
		ink = t.Text
		style = typeControlStrong
	}
	row.Children(func() {
		iconGlyph(c, section.icon, ink, section.label)
		style.style(t, gui.Text(c, section.label)).TextColor(ink)
	})
}

// settingsSectionTitle 是分节标题：标题层级由字重与字号给（U2 §1），字号与
// 已重建分节的 h2 同档（20/25），19 节的页首不再一大一小。
func settingsSectionTitle(c *gui.Context, title string) {
	t := c.Theme()
	typeH2.style(t, gui.Text(c, title)).TextColor(t.Text).
		Role(gui.RoleHeading).Level(2)
}

// dashIfEmpty 把空值显示成「—」：不发明内容，也不留空。
func dashIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}
