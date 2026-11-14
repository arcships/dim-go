package ui

import (
	"context"
	"fmt"
	"strings"

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
	id      string
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

// extItems 返回某一类的列表：连着 Source 时每帧从快照重新映射，否则用样例。
func (a *App) extItems(kind extKind) []extItem {
	if snapshot, ok := a.liveExtensions(); ok {
		return extItemsFrom(snapshot, kind)
	}
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

// extItemsFrom 把四类读数里的一类映射成列表行；不认识的字段留空。
func extItemsFrom(snapshot ExtensionsSnapshot, kind extKind) []extItem {
	var list ExtListSnapshot
	switch kind {
	case extPlugins:
		list = snapshot.Plugins
	case extSkills:
		list = snapshot.Skills
	case extMCP:
		list = snapshot.MCP
	default:
		list = snapshot.Hooks
	}
	items := make([]extItem, 0, len(list.Items))
	for _, entry := range list.Items {
		items = append(items, extItem{
			id:      entry.ID,
			name:    entry.Name,
			version: entry.Version,
			status:  extStatusFrom(entry),
			on:      entry.Enabled,
		})
	}
	return items
}

// extensionID 取服务端标识；样例没有单独标识时用名称。
func extensionID(item *extItem) string {
	if item.id != "" {
		return item.id
	}
	return item.name
}

// extKindKey 是 Intent 里的扩展类型取值。
func extKindKey(kind extKind) string {
	switch kind {
	case extPlugins:
		return "plugin"
	case extSkills:
		return "skill"
	case extMCP:
		return "mcp"
	default:
		return "hook"
	}
}

// extStatusFrom 把服务端的错误与状态文本映射成列表点的状态。
func extStatusFrom(entry ExtItemSnapshot) extStatus {
	text := strings.ToLower(entry.Error + " " + entry.Status)
	switch {
	case strings.Contains(text, "fail"), strings.Contains(text, "error"):
		return extError
	case strings.Contains(text, "update"), strings.Contains(text, "outdated"):
		return extUpdate
	}
	return extOK
}

// extensionState 是某一类扩展在接上事实后的状态与原因。
func (a *App) extensionState(kind extKind) (extState, string) {
	snapshot, ok := a.liveExtensions()
	if !ok {
		return a.extState, ""
	}
	var list ExtListSnapshot
	switch kind {
	case extPlugins:
		list = snapshot.Plugins
	case extSkills:
		list = snapshot.Skills
	case extMCP:
		list = snapshot.MCP
	default:
		list = snapshot.Hooks
	}
	switch {
	case list.CapabilityMissing:
		return extFailed, "服务端未声明该类扩展"
	case list.ReadError != "":
		return extFailed, list.ReadError
	case len(list.Items) == 0:
		return extEmpty, ""
	}
	return extLoaded, ""
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

	state, readErr := a.extensionState(extKind(a.extTab))
	switch state {
	case extLoading:
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label("扩展加载中").Children(func() {
			iconGlyph(c, iconClock, t.TextMuted, "clock")
			typeSecondary.style(t, gui.Text(c, "正在读取…")).TextColor(t.TextMuted)
		})
		return
	case extFailed:
		// 说明与重试是同一件事的两半：重试紧跟在文字后面（A3：关系归布局），
		// 不用空白把入口推到窗口另一边。接上事实时不给重试（重读入口未接）。
		reason := "读取失败：连接已断开"
		if a.wired() {
			reason = readErr
			if reason == "" {
				reason = "读取失败"
			}
		}
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).
			Label("扩展读取失败").Children(func() {
			iconGlyph(c, iconWarn, t.Danger, "exclamation")
			typeSecondary.style(t, gui.Text(c, reason)).TextColor(t.Danger)
			if !a.wired() && iconButton(c, iconButtonSpec{icon: iconUndo, label: "重试读取", size: rowCompact}) {
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
		if snapshot, ok := a.liveExtensions(); ok {
			a.mcpImportSources(c, snapshot)
		}
		// MCP 页签：每行带认证状态、导入与连接测试（F49–F51）。
		for i := range items {
			a.mcpRow(c, &items[i])
		}
		return
	}
	if a.selectMode {
		a.selectToolbar(c)
	}
	for i := range items {
		a.extRow(c, &items[i], extKind(a.extTab))
	}
}

// mcpImportSources 是 MCP 页签上方的导入来源：来源来自 mcp.import.sources，
// 点一个来源先预览再应用，不拿 MCP 名称冒充来源。
func (a *App) mcpImportSources(c *gui.Context, snapshot ExtensionsSnapshot) {
	t := c.Theme()
	if snapshot.ImportError != "" {
		a.stateNote(c, "导入来源读取失败", iconWarn, "exclamation", snapshot.ImportError, t.Danger)
		return
	}
	ready := make([]MCPImportSourceSnapshot, 0, len(snapshot.MCPImportSources))
	for _, source := range snapshot.MCPImportSources {
		if source.State == "" || source.State == "ready" {
			ready = append(ready, source)
		}
	}
	if len(ready) == 0 {
		return
	}
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label("MCP 导入来源").Children(func() {
		typeSecondary.style(t, gui.Text(c, "导入来源")).TextColor(t.TextMuted)
		for _, source := range ready {
			source := source
			label := "从 " + source.Client + " 导入"
			if actionButton(c, label, "导入来源："+source.ID, false, false).Clicked() && a.intents != nil {
				id, client := source.ID, source.Client
				var applied int
				a.dispatch("导入 MCP", func(ctx context.Context) error {
					n, err := a.intents.ImportMCP(ctx, id)
					applied = n
					return err
				}, func() {
					a.importMCP = client
					a.importResult = fmt.Sprintf("已导入 %d 项", applied)
				})
			}
		}
		gui.Spacer(c)
	})
	if a.importResult != "" {
		a.stateNote(c, "导入回执", iconCheck, "check", a.importResult, t.TextMuted)
	}
}

// extRow 是一行扩展：名称 + 版本（内容）、状态点（样式）、启用开关（框架 Switch）。
// 窄窗时改为两行：名称一行，版本与开关一行（U2-forms F26 的退化规则）。
func (a *App) extRow(c *gui.Context, it *extItem, kind extKind) {
	t := c.Theme()
	width, _ := c.Size()
	inner := width - SettingsNavWidth - 2*space8
	stacked := inner < extRowMinWidth

	status := func() {
		extStatusDot(c, statusColor(t, it.status), it.name)
	}
	sw := func() {
		before := it.on
		gui.Switch(c, &it.on).Label("启用：" + it.name)
		if a.intents != nil && it.on != before {
			id, enabled := extensionID(it), it.on
			a.dispatch("启用扩展", func(ctx context.Context) error {
				return a.intents.SetExtensionEnabled(ctx, extKindKey(kind), id, enabled)
			}, nil)
		}
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
	// 选择态与普通态是两套操作模型：选择态收起行内动作（开关、详情），整行就是
	// 勾选；否则"点开关改这一行"与"点行选这一行"会并存（F48 的处置）。
	if a.selectMode {
		checked := a.extSelected[it.name]
		row := gui.ButtonBase(c).Gap(space2).AlignItems(gui.Center).Shrink(0).
			Label("选择：" + it.name).Cursor(gui.CursorPointer)
		if row.Hovered() {
			row = row.Background(t.SurfaceHover)
		}
		if row.Clicked() {
			a.extSelected[it.name] = !checked
		}
		row.Children(func() {
			if checked {
				iconGlyph(c, iconCheck, colorAccentInk.in(t), "select:"+it.name)
			} else {
				iconGlyph(c, iconStop, t.TextMuted, "select:"+it.name)
			}
			status()
			typeBody.style(t, gui.Text(c, it.name)).TextColor(t.Text).Width(extNameWidth).SingleLine()
			typeSecondary.style(t, gui.Text(c, it.version)).TextColor(t.TextMuted).
				Width(extVersionWidth)
			gui.Spacer(c)
		})
		return
	}

	base := gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).
		Label("扩展：" + it.name + " · " + it.version + " · " + statusName(it.status))
	base = base.ContextMenu(func(m *gui.Menu) {
		if m.Item("详情").Chosen() {
			a.openExtDetail(it.name)
		}
	})
	base.Children(func() {
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

// enabledText 是启停状态的值文本（设置行里的右值，不是主界面状态词）。
func enabledText(on bool) string {
	if on {
		return "已启用"
	}
	return "已停用"
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
