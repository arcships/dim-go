package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 扩展是设置的第四节（U1-ia §4 的「扩展」；MCP 在 U1-ia 是独立第 5 节，本批按
// 一个分节 + 四个页签实现，偏离已记在 U3 T16 与 U0 §3.1）。
//
// 结构沿用设置屏：左导航一行 + 内容区的框架 Tabs；列表行是"名称 + 版本 + 状态 +
// 启用开关"，开关用框架 Switch，不自绘。四种数据在同一个行组件里画，只是数据不同。

// extKind 是扩展的四类。
type extKind int

const (
	extSkills extKind = iota
	extPlugins
	extMCP
	extHooks
)

// extKindNames 是四类的页签名（也是列表的空态主语）。
var extKindNames = [...]string{"技能", "插件", "MCP", "钩子"}

// extState 是这一节的加载态。
type extState int

const (
	extLoaded extState = iota
	extLoading
	extEmpty
	extFailed
)

// extItem 是扩展列表的一行：名称与版本是内容，状态用样式（点 + 颜色）。
type extItem struct {
	name    string
	version string
	status  extStatus
	on      bool
}

// extStatus 是一行的状态：正常、有更新、加载失败。
type extStatus int

const (
	extOK extStatus = iota
	extUpdate
	extError
)

// extItems 返回某一类的样例数据（数量按本批要求：插件 8、技能 6、MCP 4、钩子 3）。
func (a *App) extItems(kind extKind) []extItem {
	switch kind {
	case extPlugins:
		return a.plugins
	case extSkills:
		return a.skills
	case extMCP:
		return a.mcps
	default:
		return a.hooks
	}
}

// settingsExtensions 是「扩展」分节：四个页签 + 列表。
func (a *App) settingsExtensions(c *gui.Context) {
	t := c.Theme()
	settingsSectionTitle(c, "扩展")

	// 详情是屏内二级页：整块内容换成详情，顶部一个返回入口。
	if a.extDetail >= 0 {
		a.extDetailView(c)
		return
	}

	labels := make([]string, 0, len(extKindNames))
	labels = append(labels, extKindNames[:]...)
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Children(func() {
		gui.Tabs(c, &a.extTab, labels...).Label("扩展页签")
		gui.Spacer(c)
		if actionButton(c, "选择", "进入选择态", false, a.extTab == int(extMCP)).Clicked() {
			a.selectMode = !a.selectMode
			a.extSelected = map[string]bool{}
		}
	})

	switch a.extState {
	case extLoading:
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label("扩展加载中").Children(func() {
			iconGlyph(c, iconClock, t.TextMuted, "clock")
			typeSecondary.style(t, gui.Text(c, "正在读取…")).TextColor(t.TextMuted)
		})
		return
	case extFailed:
		// 说明与重试是同一件事的两半：重试紧跟在文字后面（A3：关系归布局），
		// 不用空白把入口推到窗口另一边。
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).
			Label("扩展读取失败").Children(func() {
			iconGlyph(c, iconWarn, t.Danger, "exclamation")
			typeSecondary.style(t, gui.Text(c, "读取失败：连接已断开")).TextColor(t.Danger)
			if iconButton(c, iconButtonSpec{icon: iconUndo, label: "重试读取", size: rowCompact}) {
				a.extState = extLoaded
			}
			gui.Spacer(c)
		})
		return
	case extEmpty:
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label("扩展空态").Children(func() {
			iconGlyph(c, iconChat, t.TextMuted, "chat")
			typeSecondary.style(t, gui.Text(c, "还没有可用的"+extKindNames[a.extTab])).
				TextColor(t.TextMuted)
		})
		return
	}

	items := a.extItems(extKind(a.extTab))
	if len(items) == 0 {
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label("扩展空态").Children(func() {
			iconGlyph(c, iconChat, t.TextMuted, "chat")
			typeSecondary.style(t, gui.Text(c, "还没有可用的"+extKindNames[a.extTab])).
				TextColor(t.TextMuted)
		})
		return
	}
	if a.extTab == int(extMCP) {
		// MCP 页签：每行带认证状态、导入与连接测试（F49–F51）。
		for i := range a.mcps {
			a.mcpRow(c, &a.mcps[i])
		}
		return
	}
	if a.selectMode {
		a.selectToolbar(c)
	}
	for i := range items {
		a.extRow(c, &items[i])
	}
}

// extRow 是一行扩展：名称 + 版本（内容）、状态点（样式）、启用开关（框架 Switch）。
// 窄窗时改为两行：名称一行，版本与开关一行（U2-forms F26 的退化规则）。
func (a *App) extRow(c *gui.Context, it *extItem) {
	t := c.Theme()
	width, _ := c.Size()
	inner := width - SettingsNavWidth - 2*space8
	stacked := inner < extRowMinWidth

	status := func() {
		extStatusDot(c, statusColor(t, it.status), it.name)
	}
	sw := func() {
		gui.Switch(c, &it.on).Label("启用：" + it.name)
	}

	if stacked {
		gui.Column(c).Shrink(0).Gap(space1).Children(func() {
			// 窄窗两行：第一行名称 + 状态点（属性跟着名字），第二行版本 + 开关。
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
				status()
				typeBody.style(t, gui.Text(c, it.name)).TextColor(t.Text).SingleLine()
			})
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
				typeSecondary.style(t, gui.Text(c, it.version)).TextColor(t.TextMuted)
				gui.Spacer(c)
				sw()
			})
		})
		return
	}
	// 宽窗：状态是**这一项**的属性，放在名称左侧作行首（距名称 space-2），既紧贴名称
	// 又在跨行成列；版本跟在名称后；开关独占行末（A3：关系归布局）。原来状态点贴着
	// 开关、离名称 500px，看起来像属于开关。
	base := gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).
		Label("扩展：" + it.name + " · " + it.version + " · " + statusName(it.status))
	base = base.ContextMenu(func(m *gui.Menu) {
		if m.Item("详情").Chosen() {
			a.openExtDetail(it.name)
		}
	})
	base.Children(func() {
			if a.selectMode {
				// 选择态：行首复选框（框架 Checkbox），点它不打开详情。
				checked := a.extSelected[it.name]
				gui.Checkbox(c, &checked, "").Label("选择：" + it.name)
				if checked != a.extSelected[it.name] {
					a.extSelected[it.name] = checked
				}
			}
			status()
			typeBody.style(t, gui.Text(c, it.name)).TextColor(t.Text).Width(extNameWidth).SingleLine()
			typeSecondary.style(t, gui.Text(c, it.version)).TextColor(t.TextMuted).
				Width(extVersionWidth)
			gui.Spacer(c)
			if iconButton(c, iconButtonSpec{icon: iconChevronR, label: "详情：" + it.name, size: rowCompact}) {
				a.openExtDetail(it.name)
			}
			sw()
		})
}

// statusColor 是一行状态的颜色：可用 success、有更新 warning、加载失败 danger。
func statusColor(t *gui.Theme, s extStatus) gui.Color {
	switch s {
	case extUpdate:
		return t.Warning
	case extError:
		return t.Danger
	}
	return t.Success
}

// statusName 是状态的读屏名（屏幕上不写状态词，只给点）。
func statusName(s extStatus) string {
	switch s {
	case extUpdate:
		return "有更新"
	case extError:
		return "加载失败"
	}
	return "可用"
}
