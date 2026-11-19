package ui

import (
	"context"
	"strings"

	gui "github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/deeplink"
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

	// 左内边距让开窗口控件：左栏收起成边条（窄窗总是如此）时控件会压到这一行。
	left := float32(space4)
	if (a.sidebarCollapsed || narrowWindow(c)) && bar.Left > left {
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

		// 窄窗只留「会话名 + 分支 + 面板开关」：工作区前缀会把会话名挤到截断
		// 之外，去掉它；会话名给一个宽度上限单行截断，全名仍在读屏名里。
		narrow := narrowWindow(c)
		if !narrow {
			typeSecondary.style(t, gui.Text(c, a.workspaceTitle())).
				// TB-17 工作区名
				TextColor(t.TextMuted).SingleLine().Label("工作区 " + a.workspaceTitle())
			// 斜杠只是视觉分隔，不进读屏顺序（R-TB-153）。
			// TB-18 斜杠分隔（视觉分隔，不进读屏）
			typeSecondary.style(t, gui.Text(c, "/")).TextColor(t.TextMuted).
				AccessHidden(true)
		}
		// TB-19 会话名
		title := typeTitleRow.style(t, gui.Text(c, name)).SingleLine().Label("会话 " + name)
		if narrow {
			title = title.MaxWidth(narrowTitleMaxWidth)
		}
		// 标题栏双击会话名打开重命名弹窗（行内快捷仍在左栏会话行双击）。
		if title.DoubleClicked() {
			a.openSessionRenameDialog(a.currentSessionIndex())
		}
		a.chatMenu(c)
		if a.wired() {
			a.branchChip(c)
		} else {
			a.branchMarker(c)
		}
		gui.Spacer(c)
		a.openWithGroup(c)
		// TB-38 竖分隔线
		hairlineBox(c, false, "分隔线").Height(space4)
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

// workspaceTitle 取工作区名：会话快照没有这一项，接上事实时用内置工作区名，
// 不把样例目录名漏到真实会话上；样例里用固定名。
func (a *App) workspaceTitle() string {
	if a.workspaceName != "" {
		return a.workspaceName
	}
	if a.wired() {
		return sidebarBuiltinName
	}
	return sampleWorkspaceTitle
}

// panelToggle 是标题栏右端的面板开关：开时显示关闭形，关时显示打开形。
func (a *App) panelToggle(c *gui.Context) {
	// TB-03 / TB-39 上下文档板开关（打开形）
	icon, label := iconPanelRight, "打开上下文面板"
	if a.rightOpen {
		// TB-04 / TB-40 上下文档板开关（关闭形）
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
		// 置顶：与左栏会话行的「置顶」是同一份视图状态（按会话标识记，置顶的
		// 会话进左栏「已置顶」）。取不到当前会话时置灰并写原因，不翻一个只让
		// 菜单文案变、别处看不到的布尔值。
		pinIndex := a.currentSessionIndex()
		if pinIndex < 0 && !a.wired() {
			pinIndex = sessionIndex(a.selectedSession)
		}
		pinNote := ""
		if pinIndex < 0 {
			pinNote = noteNoSessionID
		}
		pinIcon, pinText := iconPin, "置顶"
		if pinIndex >= 0 && a.pinned(pinIndex) {
			pinIcon, pinText = iconPinOff, "取消置顶"
		}
		if menuItem(c, m, pinIcon, pinText, pinText, pinNote, pinIndex < 0, false) {
			a.togglePin(pinIndex)
		}
		// 重命名：落到左栏的行内改名（同一个会话、同一条 RenameSession 写入）。
		// 只有"取不到会话"才禁用并写真实原因——RenameSession 一直都在。
		renameNote := ""
		if a.wired() && a.currentSessionIndex() < 0 {
			renameNote = noteNoSessionID
		}
		if menuItem(c, m, iconPencil, "重命名", "重命名", renameNote,
			renameNote != "", false) {
			a.openSessionRenameDialog(a.currentSessionIndex())
		}
		// 多窗口：在新窗口里打开当前会话；已经有一个窗口显示它时切换到那个窗口
		// （回执写明是哪一个），不重复开。
		if menuItem(c, m, iconExternal, "在新窗口中打开", "在新窗口中打开", "", false, false) {
			a.openCurrentInNewWindow()
		}
		if menuItem(c, m, iconDownload, "导出", "导出", "", false, false) {
			a.exportCurrentSession()
		}
		// 在新分支继续：以最后一轮结束处为切点建分支（Intents.CreateBranch），
		// 成功后切到新分支并写切点。最后一轮没结束、没有轮次或没有写通道时置灰，
		// 原因写在菜单项里（不往前找别的轮次）。
		forkTurn, forkReason := a.forkLatest()
		forkReason = a.forkReason(forkTurn, forkReason)
		forkLabel := "在新分支继续"
		if forkReason != "" {
			forkLabel += "（" + forkReason + "）"
		}
		if menuItem(c, m, iconBranch, forkLabel, "在新分支继续", forkReason,
			forkReason != "", false) {
			a.forkLatestFromHeader()
		}
		// 归档：接上 Intents 时走 ArchiveSession（左栏与命令面板同一落点），
		// -demo 才写视图回执；连着 Source 却没有写通道时置灰写原因。
		archiveNote := ""
		if a.wired() && a.intents == nil {
			archiveNote = noteNoWriteChannel
		} else if a.intents != nil && a.currentSessionIndex() < 0 {
			archiveNote = noteNoSessionID
		}
		if menuItem(c, m, iconArchive, "归档", "归档", archiveNote,
			archiveNote != "", false) {
			a.archiveCurrentSession()
		}
		hairlineBox(c, true, "")
		// 复制项：先取事实，再决定可用性——没有标识 / 目录时禁用并写原因，不复制
		// 空串；写剪贴板失败给失败原因，不显示「已复制」。
		id, cwd, _ := a.sessionCopyFacts()
		idNote := ""
		if id == "" {
			idNote = noteNoSessionID
		}
		if menuItem(c, m, iconLink, "复制会话链接", "复制会话链接", idNote, id == "", false) {
			a.copySessionLink(c, id)
		}
		if menuItem(c, m, iconFileCode, "复制会话 ID", "复制会话 ID",
			idNote, id == "", false) {
			a.copySessionField(c, id, "已复制会话 ID")
		}
		if cwd != "" && menuItem(c, m, iconFolderOpen, "复制会话路径", "复制会话路径",
			"", false, false) {
			a.copySessionField(c, cwd, "已复制会话路径")
		}
		if menuItem(c, m, iconFileCode, "复制调试信息", "复制调试信息",
			idNote, id == "", false) {
			a.copySessionField(c, a.debugInfo(), "已复制调试信息")
		}
	})
}

// startInlineRename 是会话行的行内快捷改名：与弹窗同一条写入路径
// （Intents.RenameSession），只是就地替换名称。双击会话行进入。
func (a *App) startInlineRename(index int) {
	if index < 0 {
		a.headerNote = noteNoSessionID
		return
	}
	a.renaming = index
	a.renameDraft = a.sessionName(index, a.sessionIDAt(index))
}

// archiveCurrentSession 归档当前会话：接上 Intents 时发 session.archive；样例
// （没有 Source）只写视图回执。连着 Source 却没有写通道时写原因，不冒充已归档。
func (a *App) archiveCurrentSession() {
	index := a.currentSessionIndex()
	if !a.wired() {
		a.headerNote = "已归档"
		return
	}
	if a.intents == nil {
		a.headerNote = noteNoWriteChannel
		return
	}
	if index < 0 && !a.wired() {
		index = sessionIndex(a.selectedSession)
	}
	if index < 0 {
		a.headerNote = noteNoSessionID
		return
	}
	a.archiveSession(index)
}

// exportCurrentSession 打开当前会话的导出面板（标题栏 ⋯ 菜单的「导出」）：索引
// 从会话目录换算；样例里用左栏选中的那一行，不在标题栏里按样例位置猜。
func (a *App) exportCurrentSession() {
	index := a.currentSessionIndex()
	if index < 0 && !a.wired() {
		index = sessionIndex(a.selectedSession)
	}
	if index < 0 {
		a.headerNote = noteNoSessionID
		return
	}
	a.exportSession(index)
}

// openCurrentInNewWindow 在第二个窗口里打开当前会话（标题栏 ⋯ 的「在新窗口中
// 打开」）：会话标识从目录里的当前项取，与左栏行菜单同一落点。
func (a *App) openCurrentInNewWindow() {
	index := a.currentSessionIndex()
	if index < 0 {
		// 样例里没有 Source 的当前会话：用左栏选中的那一行（截图与走查要能开窗）。
		index = sessionIndex(a.selectedSession)
	}
	if index < 0 {
		a.headerNote = noteNoSessionID
		return
	}
	names := a.sessionNames()
	if index >= len(names) {
		a.headerNote = noteNoSessionID
		return
	}
	a.openSessionInNewWindow(names[index])
}

// openSessionInNewWindow 是「在新窗口中打开」的落点：窗口由入口管，这里只把
// 会话标识交出去并把回执写回来；没有窗口管理时给可读原因，不冒充已经开窗。
func (a *App) openSessionInNewWindow(sessionID string) {
	if sessionID == "" {
		a.headerNote = noteNoSessionID
		return
	}
	if a.openInNewWindow == nil {
		a.headerNote = noteMultiWindow
		return
	}
	receipt, err := a.openInNewWindow(sessionID)
	if err != nil {
		a.headerNote = "无法打开新窗口：" + err.Error()
		return
	}
	a.headerNote = receipt
}

// copySessionLink 把会话深链写进剪贴板（dimagent://session/<会话>）：链接能被
// 本机自己的「深链」入口打开，不是本机路径。
func (a *App) copySessionLink(c *gui.Context, sessionID string) {
	if sessionID == "" {
		a.headerNote = noteNoSessionID
		return
	}
	a.copySessionField(c, deeplink.SessionURL(sessionID), "已复制会话链接")
}

// currentSessionIndex 把当前打开的会话换成目录里的序号；没有打开会话时返回 -1。
func (a *App) currentSessionIndex() int {
	id := ""
	if snap, ok := a.liveSession(); ok {
		id = snap.ID
	}
	if id == "" {
		return -1
	}
	for i, name := range a.sessionNames() {
		if name == id {
			return i
		}
	}
	return -1
}

// sessionCopyFacts 取出复制项要用的事实：会话标识与工作目录。两者都来自当前
// 会话快照；取不到时返回空串，对应的菜单项禁用。
func (a *App) sessionCopyFacts() (id, cwd, title string) {
	if snap, ok := a.liveSession(); ok {
		return snap.ID, snap.Cwd, snap.Title
	}
	if !a.wired() {
		// 样例：标识取样例行的序号，目录取样例工作区。
		return "s1", "", a.sessionTitle()
	}
	return "", "", ""
}

// debugInfo 是「复制调试信息」的内容：只有诊断字段（会话标识、标题、模型、
// 视图状态），不含凭据、token、路径之外的本机信息。
func (a *App) debugInfo() string {
	id, cwd, _ := a.sessionCopyFacts()
	lines := []string{
		"会话: " + id,
		"标题: " + a.sessionTitle(),
		"模型: " + a.currentModel(),
	}
	if cwd != "" {
		lines = append(lines, "工作目录: "+cwd)
	}
	return strings.Join(lines, "\n")
}

// copySessionField 把一段文字写进剪贴板，并在标题下方给回执；写失败时给原因，
// 不冒充成功。
func (a *App) copySessionField(c *gui.Context, value, receipt string) {
	if strings.TrimSpace(value) == "" {
		a.headerNote = noteNothingToCopy
		return
	}
	a.headerNote = "正在复制…"
	c.WriteClipboardThen(value, func(err error) {
		if err != nil {
			a.headerNote = "复制失败：" + err.Error()
			return
		}
		a.headerNote = receipt
	})
}

// openWithGroup 是「打开方式」组：主按钮（默认应用）+ 分隔线 + 更多方式。
//
// 连着 Source 时没有外部应用清单，也没有打开调用：主按钮改成中性的「打开方式」，
// 置灰并写原因，不冒充"正在用某个应用打开"。
func (a *App) openWithGroup(c *gui.Context) {
	t := c.Theme()
	live := a.wired()
	app := a.openWithApp
	label, icon := "打开方式", iconExternal
	if !live {
		if app == "" {
			app = "VS Code"
		}
		label, icon = "在 "+app+" 中打开", openWithIcon(app)
	}
	gui.Row(c).Height(rowRegular).Radius(radius).Border(strokeHairline, t.Border).
		Background(t.Surface).AlignItems(gui.Center).Children(func() {
		// TB-31 主按钮「{打开方式}」
		main := gui.ButtonBase(c).Width(composerToolSize).Height(space6).Radius(radius).
			Label(label).Tooltip(label).Cursor(gui.CursorPointer).Disabled(live)
		if live {
			main = main.Tooltip(noteOpenWith).Opacity(0.5)
		}
		if main.Hovered() && !live {
			main = main.Background(t.SurfaceHover)
		}
		main.Children(func() {
			ink := openWithColor(app, t)
			if live {
				ink = t.TextMuted
			}
			iconBox(c, icon, ink, "open-with")
		})
		if main.Clicked() && !live {
			a.headerNote = "正在用 " + app + " 打开"
		}
		hairlineBox(c, false, "").Height(space4)
		a.openWithMenu(c)
	})
}

// openWithMenu 是「更多打开方式」的浮层。
func (a *App) openWithMenu(c *gui.Context) {
	t := c.Theme()
	m := gui.MenuBase(c, &a.openWithOpen)
	tr := m.Trigger.Width(space6 - space1).Height(space6).Radius(radius).
		// TB-32 更多打开方式
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
		case a.wired():
			// 没有探测通道：不列出写死的四款应用，只写"没有清单"。
			menuItem(c, m, nil, "没有可用的外部应用", "没有可用的外部应用",
				noteOpenWith, true, false)
		case a.openWithLoading:
			// TB-33 正在检测已安装应用…
			menuItem(c, m, nil, "正在检测已安装应用…", "正在检测已安装应用", "", true, false)
		case len(openWithApps()) == 0:
			// TB-34 未检测到可用应用
			menuItem(c, m, nil, "未检测到可用应用", "未检测到可用应用", "", true, false)
		default:
			for _, app := range openWithApps() {
				app := app
				// TB-35 应用名菜单项
				if menuItem(c, m, openWithIcon(app.name), app.label, app.label, "", false, false) {
					a.openWithApp = app.name
					a.headerNote = "正在用 " + app.name + " 打开"
				}
			}
		}
		hairlineBox(c, true, "")
		// 这两项要宿主的打开 / 定位能力：连着 Source 时没有通道，置灰并写原因，
		// 不给"已显示""正在打开"这类编出来的回执。
		finderNote, defaultNote := "", ""
		if a.wired() {
			finderNote, defaultNote = noteOpenWith, noteOpenWith
		}
		// TB-36 在文件夹中显示
		if menuItem(c, m, iconFolderOpen, "在 Finder 中显示", "在 Finder 中显示",
			finderNote, finderNote != "", false) {
			a.headerNote = "已在 Finder 中显示"
		}
		// TB-37 用系统默认方式打开
		if menuItem(c, m, iconExternal, "用系统默认方式打开", "用系统默认方式打开",
			defaultNote, defaultNote != "", false) {
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
