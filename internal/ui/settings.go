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
	narrow := narrowWindow(c)
	gui.Row(c).Fill().AlignItems(gui.Stretch).
		Background(t.Background).Padding(settingsShellInset).Label("设置").Children(func() {
		gui.Row(c).Fill().AlignItems(gui.Stretch).Radius(radius).
			Background(t.Surface).Children(func() {
			if narrow {
				// 窄窗：224 的常驻导航放不下，收成顶部一行（返回 + 分节下拉），
				// 内容列占满整屏；导航与内容之间不再画竖线（上下已由一行分开）。
				gui.Column(c).Grow(1).Children(func() {
					if inset := titleBarInset(c.TitleBar()); inset > 0 {
						gui.Row(c).Height(inset).Shrink(0).DragWindow().Label("窗口控件让位").
							AccessHidden(true)
					}
					a.settingsNavCompact(c)
					hairlineBox(c, true, "导航与内容的分隔")
					gui.Column(c).Grow(1).Children(func() {
						gui.Column(c).Grow(1).MaxWidth(SettingsContentMaxWidth).
							Padding(space8, space8).Gap(space4).Children(func() {
							a.actionErrorView(c)
							if a.settingsNarrowList {
								a.settingsSectionList(c)
								return
							}
							a.settingsContent(c)
						})
					})
				})
				return
			}
			a.settingsNav(c)
			gui.Box(c).Width(settingsShellRule).Shrink(0).
				Background(t.Border).Role(gui.RoleSeparator).
				Orientation(gui.Vertical).AccessHidden(true)
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

// settingsNavCompact 是窄窗里的分节导航：返回 + 一个分节下拉（19 节的全集），
// 选中的名称换回分节标识。下拉的读屏名是「设置分节」，当前值由下拉自己显示。
func (a *App) settingsNavCompact(c *gui.Context) {
	current := settingsSectionByID(a.settingsSection)
	options := make([]string, len(settingsSectionDefs))
	for i, section := range settingsSectionDefs {
		options[i] = section.label
	}
	pick := current.label
	// S-shell-41：窄窗一点开设置就有返回按钮（取代常驻侧栏）；在看内容时它回
	// 节列表，在节列表时它回主界面。
	narrowState := narrowWindow(c)
	backVisible := narrowState
	errText := a.settingsSectionErrText(a.settingsSection)
	gui.Row(c).Shrink(0).Padding(space1, space2).Gap(space1).
		AlignItems(gui.Center).Children(func() {
		if backVisible {
			label := "返回节列表"
			if a.settingsNarrowList {
				label = "返回主界面"
			}
			// S-shell-41 / S-shell-42 返回（窄窗导航）：图标资源缺失也保留命中区。
			backPressed := iconButton(c, iconButtonSpec{icon: navBackIcon(), label: label,
				tip: label, size: rowCompact})
			if backPressed {
				if a.settingsNarrowList {
					a.screen = ScreenMain
				} else {
					// 回到节列表（窄屏列表态）：只换内容列，导航行留着。
					a.settingsNarrowList = true
				}
			}
			if errText != "" {
				// 这一节的读数失败：导航行上写原因，切节不受影响。
				navErrDot(c, current.label)
			}
		}
		gui.Select(c, &pick, options).Grow(1).Label("设置分节")
	})
	if pick != current.label {
		if section, ok := settingsSectionByLabel(pick); ok {
			a.settingsSection = section.id
			a.settingsNarrowList = false
		}
	}
}

// navBackIcon 是窄窗导航的返回图标：资源缺失时退回同一族的三角箭头，命中区不变
// （S-shell-42）。
func navBackIcon() *gui.SVG {
	if iconChevronL != nil {
		return iconChevronL
	}
	return iconChevronLFallback
}

// iconChevronLFallback 是返回图标的兜底图形：与 iconChevronL 同一网格与描边。
var iconChevronLFallback = mustSVG(`<path d="M15.75 19.5 8.25 12l7.5-7.5"/>`)

// settingsSectionList 是窄窗的节列表态：19 节逐行（图标 + 名称 + 选中态），点一行
// 进该节的内容。窄窗没有常驻侧栏，这份列表就是「回到节列表」看到的那一屏。
func (a *App) settingsSectionList(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Grow(1).Label("设置分节列表").
		Role(gui.RoleList).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Shrink(0).Gap(space1).Children(func() {
				for _, section := range settingsSectionDefs {
					row := gui.ButtonBase(c).Shrink(0).Height(SettingsNavRowHeight).
						Padding(0, space3).Radius(radius).Gap(space3).
						AlignItems(gui.Center).Justify(gui.Start).
						Label(section.label).Cursor(gui.CursorPointer).
						Role(gui.RoleListItem)
					selected := a.settingsSection == section.id
					row = row.AccessSelected(selected)
					switch {
					case selected:
						row = row.Background(t.SurfacePressed)
					case row.Hovered():
						row = row.Background(t.SurfaceHover)
					}
					if row.Clicked() {
						a.settingsSection = section.id
						a.settingsNarrowList = false
					}
					ink, style := t.TextMuted, typeControl
					if selected {
						ink, style = t.Text, typeControlStrong
					}
					row.Children(func() {
						navIconGlyph(c, section.icon, ink, section.label)
						style.style(t, gui.Text(c, section.label)).TextColor(ink).
							Grow(1).MinWidth(0).SingleLine()
					})
				}
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
		renderSettingsPersonalizationScreen(c, a)
	case secAppearance:
		renderAppearance(c, a)
	case secGeneral:
		renderGeneral(c, a)
	case secKeybindings:
		a.renderKeybindings(c)
	case secProviders:
		renderConfiguration(c, a)
	case secModality:
		renderModality(c, a)
	case secRuntime:
		renderRuntime(c, a)
	case secMCP:
		renderMCP(c, a)
	case secNotifications:
		a.renderNotifications(c)
	case secPets:
		renderPets(c, a)
	case secPermissions:
		renderPermissions(c, a)
	case secGit:
		renderGit(c, a)
	case secHooks:
		renderHooks(c, a)
	case secPluginSettings:
		renderPluginSettings(c, a)
	case secExternalAgents:
		renderExternalAgents(c)
	case secArchived:
		renderArchived(c, a)
	case secAbout:
		a.renderAbout(c)
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
			// S-shell-01 / S-shell-02 返回主界面
			if iconButton(c, iconButtonSpec{icon: iconChevronL, label: "返回主界面", size: rowCompact}) {
				a.screen = ScreenMain
			}
			gui.Spacer(c)
		})
		// 19 节在矮窗口里放不下：导航列自己滚，内容区不受影响。
		gui.Scroll(c).Grow(1).Label("设置分节").Role(gui.RoleList).Children(func() {
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
//
// 两处异常形态（S-shell 各节的「异常」行）：
//   - 图标资源缺失时只留同宽的图标位，文字左缘不动、行高不变；
//   - 该节的读数失败时行尾挂一个警示点，原因进 tooltip；行仍可点，点了照常切过去
//     （切过去以后节内自己写错误）。
func (a *App) navRow(c *gui.Context, section settingsSectionDef) {
	t := c.Theme()
	errText := a.settingsSectionErrText(section.id)
	row := gui.ButtonBase(c).Height(SettingsNavRowHeight).Padding(0, space3).
		Radius(radius).Gap(space3).AlignItems(gui.Center).
		Label(section.label).Cursor(gui.CursorPointer).Justify(gui.Start)
	selected := a.settingsSection == section.id
	row = row.Role(gui.RoleListItem).AccessSelected(selected)
	switch {
	case selected:
		row = row.Background(t.SurfacePressed)
	case row.Hovered():
		row = row.Background(t.SurfaceHover)
	}
	if errText != "" {
		row = row.Tooltip(errText)
	}
	if row.Clicked() {
		// 读数失败也不拦：导航项始终可切，失败由节内写（S-shell-03 / 07 / 11 …）。
		a.settingsSection = section.id
	}
	ink := t.TextMuted
	style := typeControl
	if selected {
		ink = t.Text
		style = typeControlStrong
	}
	row.Children(func() {
		// S-shell-04/06/08/10/12/14/16/18/20/22/24/26/28/30/32/34/36/38/40 分节图标
		navIconGlyph(c, section.icon, ink, section.label)
		// S-shell-03/05/07/09/11/13/15/17/19/21/23/25/27/29/31/33/35/37/39 分节名称
		style.style(t, gui.Text(c, section.label)).TextColor(ink)
		if errText != "" {
			// 读数失败：行尾一个警示点（不占文字列），原因在 tooltip。
			navErrDot(c, section.label)
		}
	})
}

// navIconGlyph 是分节图标的图标位：图标缺资源时只画空位，宽度仍是 iconInline，
// 文字左移不会发生（S-shell-04 等「异常」行）。
func navIconGlyph(c *gui.Context, icon *gui.SVG, color gui.Color, mark string) {
	if icon != nil {
		iconGlyph(c, icon, color, mark)
		return
	}
	gui.Box(c).Width(iconInline).Height(iconInline).Shrink(0).
		Label("图标：" + mark).Tooltip(noteSectionIconMissing).AccessHidden(true)
}

// navErrDot 是导航行尾的警示点：3.5 直径的实心点，读屏名带节名。
func navErrDot(c *gui.Context, mark string) {
	t := c.Theme()
	box := gui.Box(c).Size(space1, space1).Shrink(0).
		Label("读取失败：" + mark).Role(gui.RoleStatus).AccessHidden(true)
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, t.Danger, space1/2)
	})
}

// settingsSectionErrText 是一节的读数失败原因：有事实读数的节给出服务端原文，
// 其余节没有错误来源就返回空（不编原因）。
func (a *App) settingsSectionErrText(id settingsSectionID) string {
	if a.source == nil {
		return ""
	}
	settings := a.source.Settings()
	extensions := a.source.Extensions()
	switch id {
	case secProviders:
		return settings.ProviderError
	case secRuntime:
		return settings.RuntimesReason
	case secPermissions:
		return settings.RulesError
	case secMCP:
		return extensions.MCP.ReadError
	case secHooks:
		return extensions.Hooks.ReadError
	case secPluginSettings:
		return extensions.Plugins.ReadError
	case secExternalAgents:
		return extensions.Plugins.ReadError
	case secUsage:
		usage := a.source.Usage()
		if usage.Reason != "" && !usage.Available {
			return usage.Reason
		}
		return ""
	}
	return ""
}

// settingsSectionTitle 是分节标题：标题层级由字重与字号给（U2 §1），字号取
// design-baseline §2 的页面标题 17.5/24.5/600，19 节的页首同一个档。
func settingsSectionTitle(c *gui.Context, title string) {
	t := c.Theme()
	typePageTitle.style(t, gui.Text(c, title)).TextColor(t.Text).
		Role(gui.RoleHeading).Level(2)
}

// dashIfEmpty 把空值显示成「—」：不发明内容，也不留空。
func dashIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}
