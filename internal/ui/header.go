package ui

import (
	"context"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// header 是标题栏（参考产品的 ChatHeaderFrame / NewThreadHeader）：会话态为
// 「工作区 / 会话名 ⋯    打开方式 ▾ | 面板开关」，空态只留右侧面板开关。
//
// 高度取 max(c.TitleBar().Height, 48)：窗口控件由系统画，界面只让位。标题行
// 同时是窗口拖动区，双击最大化由框架给。

// appOpenWith 是「打开方式」菜单里的一款应用。
type appOpenWith struct{ name, label string }

// openWithApps 是采集到本机可用的四款应用（C4）。
func openWithApps() []appOpenWith {
	return []appOpenWith{
		{"VS Code", "在 VS Code 中打开"},
		{"Zed", "在 Zed 中打开"},
		{"Terminal", "在 Terminal 中打开"},
		{"Ghostty", "在 Ghostty 中打开"},
	}
}

// header 画标题栏。
func (a *App) header(c *gui.Context) {
	t := c.Theme()
	bar := c.TitleBar()
	name := a.sessionTitle()

	// 左内边距让开窗口控件：左栏收起成边条时控件会压到这一行。
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
		// 空态：参考产品只有右侧面板开关，不写会话名。
		if a.state == StateEmpty && !a.wired() {
			gui.Spacer(c)
			a.panelToggle(c)
			return
		}

		typeSecondary.style(t, gui.Text(c, a.workspaceTitle())).
			TextColor(t.TextMuted).SingleLine().Label("工作区 " + a.workspaceTitle())
		typeSecondary.style(t, gui.Text(c, "/")).TextColor(t.TextMuted)
		typeTitleRow.style(t, gui.Text(c, name)).SingleLine().Label("会话 " + name)
		a.chatMenu(c)
		if a.wired() {
			a.branchChip(c)
		} else {
			a.branchMarker(c)
		}
		gui.Spacer(c)
		a.openWithGroup(c)
		gui.Box(c).Width(strokeHairline).Height(space4).Background(t.Border).Label("分隔线")
		a.panelToggle(c)
	})
}

// sessionTitle 取当前会话名：接 Source 时用事实，否则用样例。
func (a *App) sessionTitle() string {
	if snap, ok := a.liveSession(); ok {
		if title := strings.TrimSpace(snap.Title); title != "" {
			return title
		}
	}
	if a.wired() {
		return "DimAgent"
	}
	return sampleSessionName(a.state)
}

// workspaceTitle 取工作区名：会话快照没有这一项，样例里用固定名。
func (a *App) workspaceTitle() string {
	if a.workspaceName != "" {
		return a.workspaceName
	}
	return sampleWorkspaceTitle
}

// panelToggle 是标题栏右端的面板开关：开时显示关闭形，关时显示打开形。
func (a *App) panelToggle(c *gui.Context) {
	icon, label := iconPanelRight, "打开上下文面板"
	if a.rightOpen {
		icon, label = iconPanelClose, "关闭上下文面板"
	}
	if iconButton(c, iconButtonSpec{icon: icon, label: label, tip: label, size: rowRegular}) {
		a.rightOpen = !a.rightOpen
	}
}

// branchMarker 是标题栏里的分支标记：参考产品本行不显示分支，这里只在
// 非 main 分支（或接上事实）时给一个不可点的标记，避免把分支入口丢掉。
func (a *App) branchMarker(c *gui.Context) {
	branch, _, wired := a.branchState()
	if branch == "" || (!wired && branch == "main") {
		return
	}
	t := c.Theme()
	gui.Row(c).Gap(space1).AlignItems(gui.Center).Label("分支 " + branch).Children(func() {
		iconGlyph(c, iconBranch, t.TextMuted, "branch")
		typeMicro.style(t, gui.Text(c, branch)).TextColor(t.TextMuted).SingleLine()
	})
}

// chatMenu 是会话 `⋯` 菜单（TB-20–TB-30）：9 个可见项 + 置顶/取消置顶两种状态。
func (a *App) chatMenu(c *gui.Context) {
	t := c.Theme()
	m := gui.MenuBase(c, &a.chatMenuOpen)
	tr := m.Trigger.Width(space6).Height(space6).Radius(radius).
		Label("更多操作").Tooltip("更多操作").Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() { iconBox(c, iconEllipsis, t.TextMuted, "more") })
	m.Popup(func(panel gui.Element) {
		panel.MinWidth(composerMenuWidth).Padding(space1).Radius(radiusCard).
			Background(t.Surface).Border(strokeHairline, t.Border).
			Shadow(0, 4, 18, 0, colorShadow.in(t))
		pinIcon, pinText := iconPin, "置顶"
		if a.chatPinned {
			pinIcon, pinText = iconPinOff, "取消置顶"
		}
		if menuItem(c, m, pinIcon, pinText, pinText, "", false, false) {
			a.chatPinned = !a.chatPinned
		}
		// 重命名：采集里 SessionTitle 能力不可用时禁用，并在同一行给括注。
		menuItem(c, m, iconPencil, "重命名", "重命名（SessionTitle 不可用）",
			"（SessionTitle 不可用）", true, false)
		if menuItem(c, m, iconExternal, "在新窗口中打开", "在新窗口中打开", "", false, false) {
			a.headerNote = "已在新窗口中打开"
		}
		if menuItem(c, m, iconDownload, "导出", "导出", "", false, false) {
			a.exported = a.sessionTitle()
		}
		if menuItem(c, m, iconArchive, "归档", "归档", "", false, false) {
			a.headerNote = "已归档"
		}
		gui.Box(c).Height(strokeHairline).Background(t.Border)
		if menuItem(c, m, iconLink, "复制会话链接", "复制会话链接", "", false, false) {
			a.headerNote = "已复制会话链接"
		}
		if menuItem(c, m, iconFileCode, "复制会话 ID", "复制会话 ID", "", false, false) {
			a.headerNote = "已复制会话 ID"
		}
		if menuItem(c, m, iconFolderOpen, "复制会话路径", "复制会话路径", "", false, false) {
			a.headerNote = "已复制会话路径"
		}
		if menuItem(c, m, iconFileCode, "复制调试信息", "复制调试信息", "", false, false) {
			a.headerNote = "已复制调试信息"
		}
	})
}

// openWithGroup 是「打开方式」组：主按钮（默认应用）+ 分隔线 + 更多方式。
func (a *App) openWithGroup(c *gui.Context) {
	t := c.Theme()
	app := a.openWithApp
	if app == "" {
		app = "VS Code"
	}
	gui.Row(c).Height(rowRegular).Radius(radius).Border(strokeHairline, t.Border).
		Background(t.Surface).AlignItems(gui.Center).Children(func() {
		main := gui.ButtonBase(c).Width(composerToolSize).Height(space6).Radius(radius).
			Label("在 " + app + " 中打开").Tooltip("在 " + app + " 中打开").
			Cursor(gui.CursorPointer)
		if main.Hovered() {
			main = main.Background(t.SurfaceHover)
		}
		main.Children(func() {
			iconBox(c, openWithIcon(app), openWithColor(app, t), "open-with")
		})
		if main.Clicked() {
			a.headerNote = "正在用 " + app + " 打开"
		}
		gui.Box(c).Width(strokeHairline).Height(space4).Background(t.Border)
		a.openWithMenu(c)
	})
}

// openWithMenu 是「更多打开方式」的浮层。
func (a *App) openWithMenu(c *gui.Context) {
	t := c.Theme()
	m := gui.MenuBase(c, &a.openWithOpen)
	tr := m.Trigger.Width(space6 - space1).Height(space6).Radius(radius).
		Label("更多打开方式").Tooltip("更多打开方式").Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() { iconBox(c, iconChevronDown, t.TextMuted, "chevron") })
	m.Popup(func(panel gui.Element) {
		panel.MinWidth(composerMenuWidth).Padding(space1).Radius(radiusCard).
			Background(t.Surface).Border(strokeHairline, t.Border).
			Shadow(0, 4, 18, 0, colorShadow.in(t))
		switch {
		case a.openWithLoading:
			menuItem(c, m, nil, "正在检测已安装应用…", "正在检测已安装应用", "", true, false)
		case len(openWithApps()) == 0:
			menuItem(c, m, nil, "未检测到可用应用", "未检测到可用应用", "", true, false)
		default:
			for _, app := range openWithApps() {
				app := app
				if menuItem(c, m, openWithIcon(app.name), app.label, app.label, "", false, false) {
					a.openWithApp = app.name
					a.headerNote = "正在用 " + app.name + " 打开"
				}
			}
		}
		gui.Box(c).Height(strokeHairline).Background(t.Border)
		if menuItem(c, m, iconFolderOpen, "在 Finder 中显示", "在 Finder 中显示", "", false, false) {
			a.headerNote = "已在 Finder 中显示"
		}
		if menuItem(c, m, iconExternal, "用系统默认方式打开", "用系统默认方式打开", "", false, false) {
			a.headerNote = "正在用系统默认方式打开"
		}
	})
}

// openWithIcon 是一款应用的图标形状：上游没有品牌图标，按对象类型取同网格形状。
func openWithIcon(name string) *gui.SVG {
	switch name {
	case "VS Code":
		return iconVSCode
	case "Zed":
		return iconZed
	case "Terminal":
		return iconTerminal
	case "Ghostty":
		return iconGhost
	}
	return iconExternal
}

// openWithColor 是应用图标的颜色：VS Code 取品牌蓝，其余取次要前景色。
func openWithColor(name string, t *gui.Theme) gui.Color {
	if name == "VS Code" {
		return gui.Hex("#007acc")
	}
	return t.TextMuted
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

// branchChip 是旧标题行的分支菜单（保留给分支确认态走查）：分叉图标 + 分支名，
// 点开是原生菜单；「结束分支」是高后果，两步确认。
func (a *App) branchChip(c *gui.Context) {
	t := c.Theme()
	current, branches, wired := a.branchState()

	if a.endBranchConfirm {
		b := gui.ButtonBase(c).Padding(space1, space2).Radius(radius).
			Cursor(gui.CursorPointer).Label("确认结束分支").
			Tooltip("再点一次：结束这条分支，之后不能再提交")
		if b.Hovered() {
			b = b.Background(t.SurfaceHover)
		}
		if b.Clicked() {
			if wired && a.intents != nil {
				ended := current
				a.dispatch("结束分支", func(ctx context.Context) error {
					return a.intents.EndBranch(ctx, ended)
				}, func() { a.endBranchConfirm = false })
			} else {
				ended := a.branch
				a.branch = "main"
				kept := a.branches[:0]
				for _, name := range a.branches {
					if name != ended {
						kept = append(kept, name)
					}
				}
				a.branches = kept
				a.endBranchConfirm = false
			}
		}
		b.Children(func() {
			iconGlyph(c, iconBranch, t.Danger, "branch")
			typeSecondary.style(t, gui.Text(c, "确认结束分支？")).TextColor(t.Danger).SingleLine()
		})
		if actionButton(c, "取消", "取消结束分支", false, false).Clicked() {
			a.endBranchConfirm = false
		}
		return
	}

	label := "切换分支：" + current
	b := gui.ButtonBase(c).Padding(space1, space2).Radius(radius).
		Cursor(gui.CursorPointer).Label(label).Tooltip("分支 · 点开切换")
	b = b.PopoverMenuButton(func(m *gui.Menu) {
		for _, name := range branches {
			n := name
			if m.Item(n).Checked(n == current).Chosen() {
				if wired && a.intents != nil {
					a.dispatch("切换分支", func(ctx context.Context) error {
						return a.intents.SelectBranch(ctx, n)
					}, nil)
				} else {
					a.branch = n
				}
			}
		}
		m.Separator()
		if m.Item("从当前分支切出").Chosen() {
			if wired && a.intents != nil {
				snap, _ := a.liveSession()
				prompt := strings.TrimSpace(a.draft)
				if prompt == "" {
					prompt = "从当前分支切出"
				}
				parent, turn := current, snap.TurnCount
				a.dispatch("切出分支", func(ctx context.Context) error {
					return a.intents.CreateBranch(ctx, parent, turn, prompt)
				}, nil)
			} else {
				a.branch = a.branch + "-1"
			}
		}
		if m.Item("结束分支").Disabled(current == "main").Chosen() {
			a.endBranchConfirm = true
		}
	}, gui.PopoverMenuButtonOptions{Open: &a.branchMenuOpen})
	if b.Hovered() {
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		iconGlyph(c, iconBranch, t.TextMuted, "branch")
		typeSecondary.style(t, gui.Text(c, current)).TextColor(t.TextMuted).SingleLine()
	})
}
