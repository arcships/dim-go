package ui

import (
	"strconv"
	"strings"

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
			if iconButton(c, iconButtonSpec{icon: iconSearch, label: "搜索", size: rowCompact}) {
				a.searchOpen = !a.searchOpen
				a.searchQuery = ""
			}
			// 会话操作入口放在头部：框架的 SidebarItem 会把子元素的点击吃掉，行内
			// 「…」按不了（见 F34 的被否替代）。它作用于当前选中的会话。
			a.sessionActionsEntry(c)
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
	if a.searchOpen {
		a.searchPane(c)
	}
	sessions, markers := a.sessionNames(), a.sessionMarkers()
	// 归档与删除的会话不再出现；搜索面开着时按名称过滤（空白查询给全部）。
	visible := make([]bool, len(sessions))
	any := false
	for i := range sessions {
		if a.archived[i] {
			continue
		}
		name := sessions[i]
		if r, ok := a.renamed[i]; ok {
			name = r
		}
		if a.searchOpen && a.searchQuery != "" && !containsFold(name, a.searchQuery) {
			continue
		}
		visible[i] = true
		any = true
	}
	gui.Sidebar(c, &a.selectedSession, func() {
		if len(sessions) == 0 {
			return
		}
		gui.Divider(c)
		if !any {
			gui.Row(c).Padding(space1, space2).Shrink(0).Label("搜索无结果").Children(func() {
				typeSecondary.style(c.Theme(), gui.Text(c, "没有匹配的会话")).
					TextColor(c.Theme().TextMuted)
			})
			return
		}
		gui.SidebarSection(c, "项目 A", nil, func() {
			for i := 0; i < 2 && i < len(sessions); i++ {
				if visible[i] {
					a.sessionItem(c, "s"+strconv.Itoa(i+1), a.sessionName(i, sessions[i]),
						markers[i], i)
				}
			}
		})
		gui.SidebarSection(c, "项目 B", nil, func() {
			for i := 2; i < len(sessions); i++ {
				if visible[i] {
					a.sessionItem(c, "s"+strconv.Itoa(i+1), a.sessionName(i, sessions[i]),
						markers[i], i)
				}
			}
		})
	}).Grow(1).Label("会话列表")
}

// searchPane 是左栏的搜索面（形式库 F37）：一行面差输入，Esc 或再点搜索收起；
// 空白查询给全部。过滤只作用于左栏列表，不改会话本身。
func (a *App) searchPane(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Padding(space1, space2).Gap(space2).
		AlignItems(gui.Center).Label("搜索会话").Children(func() {
		iconGlyph(c, iconSearch, t.TextMuted, "search")
		gui.TextInputBase(c, &a.searchQuery).Grow(1).Label("搜索会话").
			Padding(space1, space2).Radius(radius).Background(t.SurfacePressed).
			Placeholder("搜索")
	})
}

// sessionItem 画一条会话行：选中样式完全交给框架（聚焦 = accent 填充 +
// accent-on，未聚焦 = surface-pressed；U2 校准 C4 取框架语义）；状态标记走
// SidebarItem 的 16px icon 槽，所有会话行都占这个槽，文字左边界一致。
//
// 目录动作（重命名 / 归档 / 导出 / 删除）的可见入口在左栏头部的「会话操作」
// （框架原生弹出菜单，画在窗口里、可截图、可被无窗口 Tester 驱动）；行上还有
// 同一个菜单的右键路径。重命名与删除确认在行内完成。
func (a *App) sessionItem(c *gui.Context, id, name, marker string, index int) {
	icon := statusIcon(marker)
	if icon == nil {
		icon = iconDotEmpty // 只占位，不画：让所有会话行从同一列起
	}
	// 行高对齐官方示例的导航行（gallery-overview.png 的选中条约 26–28 高）：
	// 我们原来 22.1，列表过密。
	if a.renaming == index {
		a.renameRow(c, index, name)
		return
	}
	if a.deleting == index {
		a.deleteConfirmRow(c, index, name)
		return
	}
	menu := func(m *gui.Menu) {
		if m.Item("重命名").Chosen() {
			a.renaming = index
			a.renameDraft = name
		}
		if m.Item("归档").Chosen() {
			a.archived[index] = true
		}
		if m.Item("导出").Chosen() {
			a.exported = name
		}
		if m.Item("删除").Chosen() {
			a.deleting = index
		}
	}
	// 行的右键菜单与头部的「会话操作」是同一份菜单；行的子元素点击会被框架的
	// SidebarItem 吃掉，所以可见入口只放头部。
	gui.SidebarItem(c, id, icon, name).MinHeight(rowRegular).ContextMenu(menu)
}

// sessionActionsEntry 是左栏头部的「会话操作」入口：作用于选中的会话。
func (a *App) sessionActionsEntry(c *gui.Context) {
	if a.selectedSession == "" {
		return
	}
	index := sessionIndex(a.selectedSession)
	sessions, _ := sessionSamples(a.state)
	if index < 0 || index >= len(sessions) {
		return
	}
	name := a.sessionName(index, sessions[index])
	menu := func(m *gui.Menu) {
		if m.Item("重命名").Chosen() {
			a.renaming, a.renameDraft = index, name
		}
		if m.Item("归档").Chosen() {
			a.archived[index] = true
		}
		if m.Item("导出").Chosen() {
			a.exported = name
		}
		if m.Item("删除").Chosen() {
			a.deleting = index
		}
	}
	b := gui.ButtonBase(c).Width(rowCompact).Height(rowCompact).Radius(radius).
		Cursor(gui.CursorPointer).Label("会话操作：" + name)
	b = b.PopoverMenuButton(menu, gui.PopoverMenuButtonOptions{Open: a.menuOpenPtr(index)})
	b.Children(func() {
		iconGlyph(c, iconEllipsis, c.Theme().TextMuted, "ellipsis")
	})
}

// renameRow 是重命名的行内编辑（F35）：面差输入 + 保存 / 取消两个文字按钮，
// 与目标行的编辑态同一套形式；左栏只有 256 宽，所以分两行（输入一行、按钮一行）。
func (a *App) renameRow(c *gui.Context, index int, old string) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Padding(space1, space2).Gap(space1).
		Label("重命名会话").Children(func() {
		gui.TextInputBase(c, &a.renameDraft).Label("新名称").
			Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
		gui.Row(c).Gap(space2).Children(func() {
			gui.Spacer(c)
			if actionButton(c, "保存", "保存名称", true, false).Clicked() {
				if a.renameDraft != "" {
					a.renamed[index] = a.renameDraft
				}
				a.renaming = -1
			}
			if actionButton(c, "取消", "取消重命名", false, false).Clicked() {
				a.renaming = -1
			}
		})
		_ = old
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
				a.archived[index] = true
				a.deleting = -1
			}
			if actionButton(c, "取消", "取消删除", false, false).Clicked() {
				a.deleting = -1
			}
		})
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

// sessionMarkers 是会话行的状态标记：连着 Source 时不画样例标记（真实状态由
// 会话快照决定，后续批次接）；样例模式下沿用 sample。
func (a *App) sessionMarkers() []string {
	if a.source != nil {
		return make([]string, len(a.sessionNames()))
	}
	_, markers := sessionSamples(a.state)
	return markers
}
