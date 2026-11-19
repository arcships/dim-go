package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 左栏按参考产品重建：栏头（收起）→ 四个全局入口（新建会话 / 搜索 / 定时任务 /
// 技能/插件）→ 列表（已置顶、列表头「会话」+ 添加工作区 + 视图菜单、工作区分组、
// 会话行）→ 底部账户行（头像 + 名字 + 设置）。
//
// 结构对应清单 docs/domains/requirements/inventory-main-rt.md 区 2：
// 2.1 栏头（LB-01、LB-10）、2.2 全局入口（LB-11–15）、2.3 置顶区（LB-16–23）、
// 2.4 列表头（LB-24–33）、2.5 工作区组（LB-34–57）、2.7 会话行（LB-66–82）、
// 2.7.1 行右键菜单（LB-107–112）、2.8 账户行（LB-83–100）。
//
// 参考产品的侧栏宽 360（DOM 实测）；本项目左栏宽是布局契约的 256（tokens.go 的
// SidebarWidth），所以行宽按 256 收窄，行高、间距、字号、字号层级仍取基线常量。
// 收起时留 12px 边条，点击边条展开。
//
// 窗口模式下 macOS 的红绿灯落在左栏顶部上方：顶部先留一条与控件同高的空带
// （c.TitleBar().Height），这一条也是拖动窗口的把手。
func (a *App) sidebar(c *gui.Context) {
	if !a.palette.open && c.Shortcut(gui.Cmd, gui.KeyG) {
		a.toggleSearch()
	}
	if a.sidebarCollapsed {
		a.sidebarRail(c)
		return
	}
	t := c.Theme()
	// 与主区之间靠面差（surface 对 background）分开，不画竖线（UI-18 第一档）。
	gui.Column(c).Width(a.sidebarWidthValue()).Shrink(0).
		Background(t.Surface).
		Label("左栏 · 会话列表").Role(gui.RoleNavigation).Focusable().TabStop(true).
		Children(func() {
			a.sidebarHeader(c)
			a.sidebarEntries(c)
			if a.searchOpen {
				a.searchPane(c)
			}
			a.sidebarList(c)
			a.accountRow(c)
		})
}

// sidebarHeader 是栏头：macOS 上让出窗口控件那条（只作拖动把手，不放控件），
// 右端一个收起按钮（LB-10）。
func (a *App) sidebarHeader(c *gui.Context) {
	if inset := titleBarInset(c.TitleBar()); inset > 0 {
		gui.Row(c).Height(inset).Shrink(0).DragWindow().Label("窗口控件让位")
	}
	gui.Row(c).Height(sidebarHeaderHeight).Shrink(0).
		Padding(space1, space2).AlignItems(gui.Center).Children(func() {
		gui.Spacer(c)
		if iconButton(c, iconButtonSpec{
			icon: iconSidebarPanelClose, label: "收起侧边栏 ⌘B", tip: "收起侧边栏 ⌘B",
			size: rowRegular,
		}) {
			a.setSidebarCollapsed(true)
		}
	})
}

// sidebarEntrySpec 是一个全局入口行：图标、检查器标记、文案、行尾键帽与动作。
type sidebarEntrySpec struct {
	icon     *gui.SVG
	mark     string
	label    string
	keycap   string
	tip      string
	disabled bool
	run      func()
}

// sidebarEntries 是四个全局入口（LB-11–15）：28 高一行的图标 + 文字，悬停整行
// 变浅一档；「搜索」行尾在悬停时出现 ⌘G 键帽（LB-13、C19）。
func (a *App) sidebarEntries(c *gui.Context) {
	gui.Column(c).Shrink(0).Padding(0, space2).Gap(space1).Children(func() {
		a.sidebarEntry(c, sidebarEntrySpec{
			icon: iconPlus, mark: "plus", label: "新建会话", run: a.createSession,
		})
		a.sidebarEntry(c, sidebarEntrySpec{
			icon: iconSearch, mark: "search", label: "搜索", keycap: "⌘G",
			tip: "搜索 · ⌘G", run: a.toggleSearch,
		})
		// 定时任务与技能/插件是独立路由页（清单 §7.3、§7.4），本批次只建入口。
		// 定时任务的创建与唤醒事实没有服务端方法（DEC-16），入口置灰并写原因。
		a.sidebarEntry(c, sidebarEntrySpec{
			icon: iconClock, mark: "clock", label: "定时任务",
			tip: noteSchedule, disabled: true,
		})
		a.sidebarEntry(c, sidebarEntrySpec{
			icon: iconSidebarGrid, mark: "layout-grid", label: "技能/插件",
			tip: "技能/插件：设置里的插件设置", run: a.openSkillsPage,
		})
	})
}

// sidebarEntry 画一行全局入口。行本身是可点入口（读屏名就是文案），悬停时
// 有键帽的行在行尾补出键帽，行宽不变（键帽占的是文字后面的剩余宽度）。
func (a *App) sidebarEntry(c *gui.Context, spec sidebarEntrySpec) {
	t := c.Theme()
	row := gui.ButtonBase(c).Height(sidebarRowHeight).Padding(space1, space2).
		Radius(radiusSmall).Gap(space1).AlignItems(gui.Center).
		Label(spec.label).Cursor(gui.CursorPointer)
	if spec.tip != "" {
		row = row.Tooltip(spec.tip)
	}
	if spec.disabled {
		row = row.Disabled(true).Opacity(0.5)
	}
	hovered := row.Hovered()
	row = sidebarRowPaint(row, t, false)
	if row.Clicked() && !spec.disabled && spec.run != nil {
		spec.run()
	}
	row.Children(func() {
		iconGlyph(c, spec.icon, t.Text, spec.mark)
		typeSecondary.style(t, gui.Text(c, spec.label)).TextColor(t.Text).Grow(1).SingleLine()
		if hovered && spec.keycap != "" {
			sidebarKeycap(c, t, spec.keycap)
		}
	})
}

// openSkillsPage 打开设置 › 插件设置：左栏「技能/插件」的落点。技能与插件的
// 安装、启用、卸载都在那一节，而本版没有独立的技能路由页（清单 §7.4 的 PG-12
// 组尚未按参考产品重建）。
func (a *App) openSkillsPage() {
	a.screen = ScreenSettings
	a.settingsSection = secPluginSettings
}

// toggleSearch 开合左栏的搜索面（空白查询给全部）。
func (a *App) toggleSearch() {
	a.searchOpen = !a.searchOpen
	if !a.searchOpen {
		a.searchQuery = ""
	}
}

// searchPane 是左栏的搜索面（F37）：一行面差输入，Esc 或再点搜索收起；
// 过滤只作用于左栏列表，不改会话本身。参考产品把入口做成全窗搜索弹窗
// （清单 §7.2），弹窗不在本批次，这里保留行内过滤。
func (a *App) searchPane(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Padding(space1, space2).Gap(space2).
		AlignItems(gui.Center).Label("搜索会话").Children(func() {
		iconGlyph(c, iconSearch, t.TextMuted, "search-pane")
		gui.TextInputBase(c, &a.searchQuery).Grow(1).Label("搜索会话").
			Padding(space1, space2).Radius(radius).Background(t.SurfacePressed).
			Placeholder("搜索")
	})
}

// sidebarList 是列表本体：已置顶（有置顶时）→ 列表头「会话」→ 工作区分组；
// 平铺视图换成日期分组。列表自己滚动，账户行贴底不动。
func (a *App) sidebarList(c *gui.Context) {
	gui.Scroll(c).Grow(1).Label("会话列表").Children(func() {
		gui.Column(c).Shrink(0).Padding(0, space2).Gap(space2).Children(func() {
			all := a.sidebarThreads()
			if pinned := a.sidebarPinnedThreads(all); len(pinned) > 0 {
				gui.Column(c).Shrink(0).Children(func() {
					a.sidebarHeading(c, "已置顶")
					for _, thread := range pinned {
						a.sessionItem(c, thread.index, thread.name, thread.marker, thread.when)
					}
				})
			}
			a.listHeader(c)
			if len(all) == 0 && a.searchQuery != "" {
				a.sidebarNote(c, "没有匹配的会话", false)
				return
			}
			if a.sidebarView.flat {
				a.flatGroups(c, all)
				return
			}
			for _, ws := range a.sidebarWorkspaces(all) {
				// 搜索命中时只画有命中的组：没有命中的空组（含内置「工作区」）
				// 不占位，避免命中列表里夹一行「暂无会话」。
				if a.sidebarSearching() && len(a.sidebarGroupThreads(ws, all)) == 0 {
					continue
				}
				a.workspaceGroup(c, ws, all)
				if ws.builtin {
					// 内置工作区与其余工作区之间一条分隔线（LB-57）。
					gui.Divider(c)
				}
			}
		})
	})
}

// sidebarHeading 是只读分组标题（「已置顶」）：10.5px、侧栏弱文字色，不加计数。
func (a *App) sidebarHeading(c *gui.Context, text string) {
	t := c.Theme()
	gui.Row(c).Height(sidebarHeadingHeight).Shrink(0).Padding(space1, space1).
		AlignItems(gui.Center).Children(func() {
		typeMicro.style(t, gui.Text(c, text)).TextColor(colorSidebarHeading.in(t)).SingleLine()
	})
}

// listHeader 是列表头（LB-24–26）：「会话」标题 + 添加工作区 + 视图菜单。
func (a *App) listHeader(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Height(sidebarHeadingHeight).Shrink(0).Padding(space1, space1).
		Margin(space2, 0, space1, 0).
		AlignItems(gui.Center).Gap(space1).Children(func() {
		typeMicro.style(t, gui.Text(c, "会话")).TextColor(colorSidebarHeading.in(t)).SingleLine()
		gui.Spacer(c)
		// 添加工作区：打开系统目录选择器属于平台能力，本批次只留入口与读屏名。
		if iconButton(c, iconButtonSpec{icon: iconSidebarFolderPlus, label: "添加工作区", size: rowCompact}) {
			a.addWorkspace()
		}
		a.sidebarViewButton(c)
	})
}

// sidebarViewButton 是列表头的视图切换（LB-26）：分组视图显示列表图标，平铺
// 显示宫格；点开是排序 / 视图 / 折叠全部（LB-27–33、C11）。
func (a *App) sidebarViewButton(c *gui.Context) {
	t := c.Theme()
	icon := iconSidebarList
	label := "视图：按工作区分组"
	if a.sidebarView.flat {
		icon = iconSidebarGrid
		label = "视图：平铺"
	}
	b := gui.ButtonBase(c).Width(rowCompact).Height(rowCompact).Radius(radiusSmall).
		Cursor(gui.CursorPointer).Label(label).Tooltip("视图")
	b = b.PopoverMenuButton(a.viewMenu, gui.PopoverMenuButtonOptions{Open: &a.sidebarView.viewMenu})
	switch {
	case b.Pressed() || a.sidebarView.viewMenu:
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		iconGlyph(c, icon, t.TextMuted, "view")
	})
}

// viewMenu 是视图菜单：排序两项、视图两项、折叠全部。参考产品里「排序」「视图」
// 是不可点的分组标题；框架的菜单项只有"项"一种，这里用禁用的项表达。
func (a *App) viewMenu(m *gui.Menu) {
	m.Item("排序").Disabled(true)
	if m.Item("最近活跃").Radio(!a.sidebarView.byCreated).Chosen() {
		a.sidebarView.byCreated = false
	}
	if m.Item("最近创建").Radio(a.sidebarView.byCreated).Chosen() {
		a.sidebarView.byCreated = true
	}
	m.Separator()
	m.Item("视图").Disabled(true)
	if m.Item("按工作区分组").Radio(!a.sidebarView.flat).Chosen() {
		a.sidebarView.flat = false
	}
	if m.Item("平铺").Radio(a.sidebarView.flat).Chosen() {
		a.sidebarView.flat = true
	}
	m.Separator()
	if m.Item("折叠全部").Chosen() {
		a.collapseAllGroups()
	}
}

// addWorkspace 是「添加工作区」的落点：macOS 弹系统目录选择器，其余平台在同一个
// 小框里手输路径；选中后目录登记进本地偏好（左栏多一组），取消不改任何状态。
// 见 dirpick.go。
func (a *App) addWorkspace() {
	a.startDirPick(dirPickWorkspace)
}

// addWorkspacePath 把选中的目录登记成一个附加工作区分组（本地偏好里的一项）。
// 已经登记过、或服务端本来就授权了同一个目录时不再加一组。
func (a *App) addWorkspacePath(path string) {
	for _, existing := range a.workspaces {
		if existing == path {
			return
		}
	}
	if a.source != nil {
		for _, ws := range a.source.Workspaces() {
			if ws.Path == path {
				return
			}
		}
	}
	a.workspaces = append(a.workspaces, path)
	// 分组的标识按归一后的目录记（与 sidebarWorkspaces 建组时同一口径）。
	a.expandGroup(workspaceGroupID(cleanGroupPath(path)))
}

// workspaceGroup 是一个工作区分组（LB-34–57）：一行工作区（图标 / 折叠箭头 +
// 名称 + 悬停出现的新建与更多）与其会话行；展开且没有会话时写「暂无会话」。
// 组名取目录末段，行上悬停给出的说明是完整目录。搜索命中时组一律展开（命中
// 跨组显示），但不改写折叠态。
//
// 行宽按参考产品的两段：工作区行缩进 7，会话行占满内容宽（DOM 实测内置工作区
// 的会话行 x=7 w=346、其他分组的会话行缩进到 x=14）。
func (a *App) workspaceGroup(c *gui.Context, ws sidebarWorkspace, all []sidebarThread) {
	t := c.Theme()
	open := a.sidebarSearching() || !a.groupCollapsed(ws)
	threads := a.sidebarGroupThreads(ws, all)
	gui.Column(c).Shrink(0).Children(func() {
		gui.Row(c).Shrink(0).PaddingStart(space2).Children(func() {
			if a.renamingGroup == ws.id {
				a.groupRenameRow(c, ws)
				return
			}
			row := gui.ButtonBase(c).Height(sidebarGroupRowHeight).Padding(space1, space2).
				Radius(radiusSmall).Gap(space1).AlignItems(gui.Center).
				Grow(1).Label(ws.name).Cursor(gui.CursorPointer)
			if !ws.builtin {
				row = row.Tooltip(groupTooltip(ws))
			}
			hovered := row.Hovered()
			row = sidebarRowPaint(row, t, false)
			if row.Clicked() {
				a.toggleGroup(ws)
			}
			row.Children(func() {
				// 悬停时图标槽换成折叠箭头（LB-35）；静止时是 🏠 或文件夹。
				sidebarDisclosure(c, t, ws, open, hovered)
				typeSecondaryStrong.style(t, gui.Text(c, ws.name)).TextColor(t.Text).
					Grow(1).SingleLine()
			})
			// 行尾动作在工作区行之外（参考产品的 21×21 两个按钮），静止时不接收
			// 指针，也不改变行按钮的宽度。
			sidebarIconAction(c, sidebarIconSpec{
				icon: iconPlus, label: "在工作区中新建会话：" + ws.name,
				hover: hovered, run: func() { a.createSessionIn(ws) },
			})
			sidebarIconAction(c, sidebarIconSpec{
				icon: iconEllipsis, label: "工作区菜单：" + ws.name,
				hover: hovered || *a.groupMenuPtr(ws.id),
				menu: func(m *gui.Menu) { a.workspaceMenu(m, ws, threads) },
				open: a.groupMenuPtr(ws.id),
			})
		})
		if !open {
			return
		}
		body := func() {
			if len(threads) == 0 {
				a.sidebarNote(c, "暂无会话", true)
				return
			}
			for _, thread := range threads {
				a.sessionItem(c, thread.index, thread.name, thread.marker, thread.when)
			}
		}
		if ws.builtin {
			body()
			return
		}
		gui.Column(c).Shrink(0).Padding(0, space2).Children(body)
	})
}

// workspaceMenu 是工作区行的菜单（C16 / C17 / C45）：置顶 / 在文件管理器中显示 /
// 归档会话 / 重命名 / 移除。
//
// 能做的都在本地做完：置顶与改名是本地视图状态，移除会把本端登记的目录一并删掉；
// 「在文件管理器中显示」要宿主能力，本版没有通道，按 DEC-09 禁用并写原因。
func (a *App) workspaceMenu(m *gui.Menu, ws sidebarWorkspace, threads []sidebarThread) {
	if !ws.builtin {
		pin := "置顶工作区"
		if a.groupPinned(ws) {
			pin = "取消置顶工作区"
		}
		if m.Item(pin).Subtitle(ws.name).Chosen() {
			a.toggleGroupPin(ws)
		}
	}
	m.Item("在文件管理器中显示").Disabled(true).Subtitle(notePreviewIO)
	if len(threads) > 0 {
		if m.Item("归档会话").Chosen() {
			for _, thread := range threads {
				a.archiveSession(thread.index)
			}
		}
	}
	m.Separator()
	if m.Item("重命名").Chosen() {
		a.renamingGroup, a.groupRenameDraft = ws.id, ws.name
	}
	if !ws.builtin {
		if m.Item("移除工作区").Chosen() {
			a.removeWorkspaceGroup(ws)
		}
	}
}

// groupRenameRow 是组头的行内改名（LB-48）：输入框就地替换名称，Enter 保存、
// Esc 放弃（Esc 在 viewKeys 里统一处理）。名称留空时回落原名称。
func (a *App) groupRenameRow(c *gui.Context, ws sidebarWorkspace) {
	t := c.Theme()
	gui.Column(c).Grow(1).Shrink(0).Padding(space1, space1).Gap(space1).
		Label("重命名工作区").Children(func() {
		input := gui.TextInputBase(c, &a.groupRenameDraft).Label("工作区名称").
			Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
		input.OnSubmit(func() { a.renameGroup(ws) })
		gui.Row(c).Gap(space2).Children(func() {
			gui.Spacer(c)
			if actionButton(c, "保存", "保存工作区名称", true, false).Clicked() {
				a.renameGroup(ws)
			}
			if actionButton(c, "取消", "取消重命名工作区", false, false).Clicked() {
				a.renamingGroup, a.groupRenameDraft = "", ""
			}
		})
	})
}

// groupTooltip 是分组行的悬停说明：有目录时给完整目录，样例分组退回组名。
func groupTooltip(ws sidebarWorkspace) string {
	if ws.path != "" {
		return ws.path
	}
	return ws.name
}

// flatGroups 是平铺视图（C12）：按日期分组，不按工作区分组。日期分组用
// 「今天」一档（目录快照没有时间事实时全部落在今天）。
func (a *App) flatGroups(c *gui.Context, all []sidebarThread) {
	threads := a.unpinnedThreads(all)
	gui.Column(c).Shrink(0).Children(func() {
		if len(threads) == 0 {
			a.sidebarNote(c, "暂无会话", true)
			return
		}
		a.sidebarHeading(c, "今天")
		for _, thread := range threads {
			a.sessionItem(c, thread.index, thread.name, thread.marker, thread.when)
		}
	})
}

// sidebarNote 是一行说明文字（「暂无会话」「没有匹配的会话」）：10.5px、最淡的
// 侧栏文字；hint 为真时缩进到会话标题那一列。
func (a *App) sidebarNote(c *gui.Context, text string, indented bool) {
	t := c.Theme()
	pad := float32(space1)
	if indented {
		pad = sidebarEmptyIndent
	}
	gui.Row(c).Shrink(0).Padding(space1, pad).AlignItems(gui.Center).Children(func() {
		typeMicro.style(t, gui.Text(c, text)).TextColor(colorSidebarFaint.in(t)).SingleLine()
	})
}

// sessionItem 画一条会话行（LB-66–82）：状态槽 + 标题 +（等待操作）标签 +
// 时间 / 悬停动作。选中态是 12% 底的整行；悬停时行尾换成置顶、归档与更多三个
// 动作（参考产品 C18 的悬停动作，第三项是本实现的行内菜单入口）。
//
// 行右键与行内「更多」是同一份菜单（LB-107–112）：置顶 / 重命名 / 在新窗口中
// 打开 / 导出 / 归档 / 删除。
func (a *App) sessionItem(c *gui.Context, index int, name, marker, when string) {
	t := c.Theme()
	if a.renaming == index {
		a.renameRow(c, index)
		return
	}
	if a.deleting == index {
		a.deleteConfirmRow(c, index, name)
		return
	}
	id := "s" + strconv.Itoa(index+1)
	chosen := a.selectedSession == id
	row := gui.ButtonBase(c).Height(sidebarRowHeight).Padding(space1, space1).
		Radius(radiusSmall).Gap(space1).AlignItems(gui.Center).
		Label(name).Cursor(gui.CursorPointer).
		ContextMenu(func(m *gui.Menu) { a.sessionMenu(m, index, name) })
	hovered := row.Hovered()
	menuOpen := *a.menuOpenPtr(index)
	row = sidebarRowPaint(row, t, chosen)
	if row.Clicked() && !chosen {
		a.selectedSession = id
	}
	row.Children(func() {
		sidebarStatusSlot(c, t, marker, a.pinned(index))
		typeSecondary.style(t, gui.Text(c, name)).TextColor(t.Text).Grow(1).SingleLine()
		if marker == "◷" {
			sidebarBadge(c, t, "等待操作")
		}
		if hovered || menuOpen {
			sidebarIconAction(c, sidebarIconSpec{
				icon: iconEllipsis, label: "会话操作：" + name, color: t.TextMuted,
				hover: true, menu: func(m *gui.Menu) { a.sessionMenu(m, index, name) },
				open: a.menuOpenPtr(index),
			})
			pinIcon, pinLabel := iconSidebarPin, "置顶会话："+name
			if a.pinned(index) {
				pinIcon, pinLabel = iconSidebarPinOff, "取消置顶会话："+name
			}
			sidebarIconAction(c, sidebarIconSpec{
				icon: pinIcon, label: pinLabel, hover: true,
				run: func() { a.togglePin(index) },
			})
			sidebarIconAction(c, sidebarIconSpec{
				icon: iconArchive, label: "归档会话：" + name, hover: true,
				run: func() { a.archiveSession(index) },
			})
			return
		}
		if when != "" {
			typeMicro.style(t, gui.Text(c, when)).TextColor(t.TextMuted).SingleLine()
		}
	})
}

// sessionMenu 是会话行的六项菜单（LB-107–112 与 C2 的同构菜单）：置顶 / 重命名 /
// 在新窗口中打开 / 导出 / 归档 / 删除。已有置顶的会话首项是「取消置顶」。
func (a *App) sessionMenu(m *gui.Menu, index int, name string) {
	pin := "置顶"
	if a.pinned(index) {
		pin = "取消置顶"
	}
	if m.Item(pin).Chosen() {
		a.togglePin(index)
	}
	if m.Item("重命名").Chosen() {
		a.renaming, a.renameDraft = index, name
	}
	// 在新窗口中打开：弹窗窗口不在本批次（清单 §7 之外的窗口形态）。
	m.Item("在新窗口中打开")
	if m.Item("导出").Chosen() {
		a.exportSession(index)
	}
	m.Separator()
	if m.Item("归档").Chosen() {
		a.archiveSession(index)
	}
	if m.Item("删除").Chosen() {
		a.deleting = index
	}
}

// sidebarIconSpec 是行内小动作（21×21 的图标入口）：图标、读屏名、是否显形，
// 以及"点一下做什么"或"打开哪个菜单"。不显形时仍占位、但不接收指针（框架的
// IgnoreInput 约定），因此行内其余部分的点击与命中区不变。
type sidebarIconSpec struct {
	icon  *gui.SVG
	label string
	color gui.Color
	hover bool
	menu  func(*gui.Menu)
	open  *bool
	run   func()
}

// sidebarIconAction 画一个行内小动作；返回它是否被按下。
func sidebarIconAction(c *gui.Context, spec sidebarIconSpec) bool {
	t := c.Theme()
	color := spec.color
	if color == (gui.Color{}) {
		color = t.TextMuted
	}
	b := gui.ButtonBase(c).Width(sidebarActionSize).Height(sidebarActionSize).
		Radius(radiusSmall).Cursor(gui.CursorPointer).Label(spec.label).
		Tooltip(spec.label)
	if spec.menu != nil {
		b = b.PopoverMenuButton(spec.menu, gui.PopoverMenuButtonOptions{Open: spec.open})
	}
	if spec.hover {
		switch {
		case b.Pressed():
			b = b.Background(t.SurfacePressed)
		case b.Hovered():
			b = b.Background(t.SurfaceHover)
		}
	} else {
		b = b.Opacity(0).IgnoreInput(true)
	}
	b.Children(func() {
		gui.Icon(c, spec.icon).Size(iconInline, iconInline).TextColor(color)
	})
	clicked := b.Clicked()
	if clicked && spec.run != nil {
		spec.run()
	}
	return clicked
}

// accountRow 是底部账户行（LB-83–88、LB-100）：18px 首字母头像 + 账户名 +
// 右侧设置齿轮；整行点开账户菜单（C32）。退出登录的确认行占同一位置（LB-101）。
func (a *App) accountRow(c *gui.Context) {
	if a.logoutConfirm {
		a.logoutConfirmRow(c)
		return
	}
	t := c.Theme()
	initial := a.accountInitialValue()
	name := a.accountNameValue()
	gui.Row(c).Shrink(0).Height(sidebarAccountHeight+space2).
		Padding(space1, space2).AlignItems(gui.Center).Gap(space1).Children(func() {
		row := gui.ButtonBase(c).Grow(1).Height(sidebarAccountHeight).
			Padding(space1, space2).Radius(radiusSmall).Gap(space2).
			AlignItems(gui.Center).Label(name).Cursor(gui.CursorPointer)
		row = row.PopoverMenuButton(a.accountMenu, gui.PopoverMenuButtonOptions{Open: &a.sidebarView.accountMenu})
		row = sidebarRowPaint(row, t, false)
		row.Children(func() {
			sidebarAvatar(c, t, initial)
			typeSecondary.style(t, gui.Text(c, name)).TextColor(t.Text).Grow(1).SingleLine()
		})
		if iconButton(c, iconButtonSpec{icon: iconGear, label: "设置", tip: "设置 ⌘,", size: sidebarAccountHeight}) {
			a.screen = ScreenSettings
		}
	})
}

// sidebarAvatar 是账户行头像：18px 整圆，里面是账户名首字母（内容，不是界面语）。
func sidebarAvatar(c *gui.Context, t *gui.Theme, initial string) {
	box := gui.Box(c).Width(sidebarAvatarSize).Height(sidebarAvatarSize).Shrink(0).
		Radius(radiusFull).Background(t.SurfacePressed).Label("账号：" + initial).
		AlignItems(gui.Center).Justify(gui.Center)
	box.Children(func() {
		if initial == "" {
			return
		}
		typeMicro.style(t, gui.Text(c, initial)).TextColor(t.Text).TextAlign(gui.Center)
	})
}

// accountMenu 是账户菜单（C32）：身份项 + 用量 / 显示 Pet / 邀请朋友 + 设置 /
// 退出登录。用量带着账户事实里的剩余额度；Pet 与邀请没有通道，按 DEC-09 的口径
// 禁用并写原因；退出登录走已有的 account.logout（确认后提交）。
func (a *App) accountMenu(m *gui.Menu) {
	m.Item(a.accountInitialValue() + "   " + a.accountNameValue()).Disabled(true)
	if m.Item(a.usageLabel()).Chosen() {
		a.screen = ScreenSettings
		a.settingsSection = secUsage
	}
	m.Item("显示 Pet").Disabled(true).Subtitle(notePet)
	m.Item("邀请朋友").Disabled(true).Subtitle(noteInvite)
	m.Separator()
	if m.Item("设置").Chosen() {
		a.screen = ScreenSettings
	}
	if m.Item("退出登录").Chosen() {
		a.logoutConfirm = true
	}
}

// logoutConfirmRow 是退出登录的两步确认（LB-101）：与删除会话同一套原地确认，
// 不可逆动作不在菜单里一步完成。确认后走 account.logout；失败原因回到错误行。
func (a *App) logoutConfirmRow(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Padding(space1, space2).Gap(space1).
		Label("退出登录确认").Role(gui.RoleAlert).Children(func() {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
			iconGlyph(c, iconWarn, t.Warning, "warning")
			typeSecondary.style(t, gui.Text(c, "退出 Dim？")).TextColor(t.Text).Grow(1).SingleLine()
		})
		typeMicro.style(t, gui.Text(c, "退出后，需要重新登录才能使用 Dim 账户。")).
			TextColor(t.TextMuted).MaxLines(2)
		gui.Row(c).Gap(space2).Children(func() {
			gui.Spacer(c)
			if actionButton(c, "退出登录", "确认退出登录", true, false).Clicked() {
				a.logoutConfirm = false
				a.dispatch("退出登录", func(ctx context.Context) error {
					return a.intents.Logout(ctx)
				}, nil)
			}
			if actionButton(c, "取消", "取消退出登录", false, false).Clicked() {
				a.logoutConfirm = false
			}
		})
	})
}

// usageLabel 是账户菜单里的用量项：有账户用量事实时把剩余额度写在后面
// （LB-91 / LB-92），没有事实只写「用量」，不编造百分比。
func (a *App) usageLabel() string {
	usage, ok := a.liveUsage()
	if !ok || !usage.Available || usage.Limit <= 0 {
		return "用量"
	}
	left := usage.Limit - usage.Used
	if left < 0 {
		left = 0
	}
	return fmt.Sprintf("用量   剩余 %d%%", int(left*100/usage.Limit+0.5))
}

// renameRow 是重命名的行内编辑（F35）：面差输入 + 保存 / 取消两个文字按钮，
// 与目标行的编辑态同一套形式；左栏只有 256 宽，所以分两行（输入一行、按钮一行）。
func (a *App) renameRow(c *gui.Context, index int) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Padding(space1, space2).Gap(space1).
		Label("重命名会话").Children(func() {
		gui.TextInputBase(c, &a.renameDraft).Label("新名称").
			Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
		gui.Row(c).Gap(space2).Children(func() {
			gui.Spacer(c)
			if actionButton(c, "保存", "保存名称", true, false).Clicked() {
				a.saveRename(index)
			}
			if actionButton(c, "取消", "取消重命名", false, false).Clicked() {
				a.renaming = -1
			}
		})
	})
}

// deleteConfirmRow 是删除的两步确认（F36）：同一行原地换成「确认删除「名」？」+
// 删除 / 取消，与结束分支的确认同一套形态；不可逆动作不在菜单里一步完成。
func (a *App) deleteConfirmRow(c *gui.Context, index int, name string) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Padding(space1, space2).Gap(space1).
		Label("删除会话确认").Children(func() {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
			iconGlyph(c, iconXMark, t.Danger, "x-mark")
			typeSecondary.style(t, gui.Text(c, "确认删除「"+name+"」？")).TextColor(t.Danger).
				SingleLine()
		})
		gui.Row(c).Gap(space2).Children(func() {
			gui.Spacer(c)
			if actionButton(c, "删除", "确认删除", true, false).Clicked() {
				a.deleteSession(index)
			}
			if actionButton(c, "取消", "取消删除", false, false).Clicked() {
				a.deleting = -1
			}
		})
	})
}

// sessionIDAt 把视图里的行序号换成目录里的真实会话标识。
func (a *App) sessionIDAt(index int) string {
	names := a.sessionNames()
	if index < 0 || index >= len(names) {
		return ""
	}
	return names[index]
}

// createSession 新建会话：工作目录取当前工作区（空态选择器选中的工作区，其次
// 是当前会话的目录）。样例模式下没有写入路径，切到空态（参考产品的「新建会话」
// 进入空会话）。
func (a *App) createSession() {
	a.createSessionAt(a.currentWorkspaceCwd())
}

// createSessionIn 在指定工作区分组下新建会话（LB-42）：组先展开（新会话落到
// 哪一组要看得见），cwd 取该组的目录，不拿当前会话的目录顶替。
func (a *App) createSessionIn(ws sidebarWorkspace) {
	a.expandGroup(ws.id)
	a.createSessionAt(ws.path)
}

// createSessionAt 按给定工作目录新建会话；目录为空时仍走当前工作区的回落。
func (a *App) createSessionAt(cwd string) {
	if a.intents == nil {
		a.SetState(StateEmpty)
		return
	}
	if cwd == "" {
		cwd = a.currentWorkspaceCwd()
	}
	a.dispatch("新建会话", func(ctx context.Context) error {
		return a.intents.CreateSession(ctx, cwd)
	}, nil)
}

// saveRename 保存行内改名：接上事实时走目录改名，失败原因回到错误行。
func (a *App) saveRename(index int) {
	if strings.TrimSpace(a.renameDraft) == "" {
		return
	}
	title := a.renameDraft
	if a.intents == nil {
		a.renamed[index] = title
		a.renaming = -1
		return
	}
	sessionID := a.sessionIDAt(index)
	if sessionID == "" {
		return
	}
	a.dispatch("重命名", func(ctx context.Context) error {
		return a.intents.RenameSession(ctx, sessionID, title)
	}, func() {
		a.renamed[index] = title
		a.renaming = -1
	})
}

// archiveSession 归档（本地隐藏）一行；样例模式只改视图状态。
func (a *App) archiveSession(index int) {
	if a.intents == nil {
		a.archived[index] = true
		return
	}
	sessionID := a.sessionIDAt(index)
	if sessionID == "" {
		return
	}
	a.dispatch("归档", func(ctx context.Context) error {
		return a.intents.ArchiveSession(ctx, sessionID, true)
	}, nil)
}

// exportSession 导出一行到本地文件；回执路径显示在同一位置。
func (a *App) exportSession(index int) {
	name := a.sessionName(index, a.sessionIDAt(index))
	if a.intents == nil {
		a.exported = name
		return
	}
	sessionID := a.sessionIDAt(index)
	if sessionID == "" {
		return
	}
	var path string
	a.dispatch("导出", func(ctx context.Context) error {
		written, err := a.intents.ExportSession(ctx, sessionID)
		if err == nil {
			path = written
		}
		return err
	}, func() {
		if path != "" {
			a.exported = path
		}
	})
}

// deleteSession 确认删除一行：成功后目录条目消失，当前会话由适配器切走。
func (a *App) deleteSession(index int) {
	if a.intents == nil {
		a.archived[index] = true
		a.deleting = -1
		return
	}
	sessionID := a.sessionIDAt(index)
	if sessionID == "" {
		return
	}
	a.dispatch("删除会话", func(ctx context.Context) error {
		return a.intents.DeleteSession(ctx, sessionID)
	}, func() {
		a.deleting = -1
	})
}

// menuOpenPtr 取某一行菜单的开关指针（视图状态，按需创建并缓存指针——
// 每次返回新指针的话，框架把开关写进去后 map 里看不到）。
func (a *App) menuOpenPtr(index int) *bool {
	if a.menuOpen == nil {
		a.menuOpen = map[int]*bool{}
	}
	if p, ok := a.menuOpen[index]; ok {
		return p
	}
	v := false
	p := &v
	a.menuOpen[index] = p
	return p
}

// sessionName 取这一行显示的会话名（重命名后的值优先；是视图状态，不是事实）。
func (a *App) sessionName(i int, def string) string {
	if r, ok := a.renamed[i]; ok {
		return r
	}
	return def
}

// containsFold 是大小写无关的子串匹配（搜索面用）。
func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// sessionIndex 把会话 id（s1 / s2 / s3）换成序号。
func sessionIndex(id string) int {
	if len(id) < 2 {
		return -1
	}
	n, err := strconv.Atoi(id[1:])
	if err != nil {
		return -1
	}
	return n - 1
}

// sidebarRail 是收起后的 12px 边条：不显示会话，点击展开（LB-10 / TB-13）。
// 读屏名与 tooltip 带 ⌘B，键盘焦点落在边条上按回车同效。
func (a *App) sidebarRail(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Width(SidebarRailWidth).Shrink(0).
		Background(t.Surface).
		AlignItems(gui.Center).Justify(gui.Center).Gap(space1).
		Label("展开侧边栏 ⌘B").Tooltip("展开侧边栏 ⌘B").
		Role(gui.RoleButton).Focusable().TabStop(true).
		OnClick(func() { a.setSidebarCollapsed(false) }).
		Children(func() {
			for i := 0; i < 3; i++ {
				gui.Box(c).Width(railMarkWidth).Height(railMarkHeight).
					Radius(railMarkHeight / 2).Background(t.TextMuted)
			}
		})
}

// sessionMarkers 是会话行的状态标记：连着 Source 时不画样例标记（真实状态由
// 会话快照决定，后续批次接）；样例模式下沿用 sample。
func (a *App) sessionMarkers() []string {
	if a.source != nil {
		return make([]string, len(a.sessionNames()))
	}
	_, markers := sessionSamples(a.state)
	return markers
}
