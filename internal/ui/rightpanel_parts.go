package ui

import (
	"fmt"
	"runtime"

	gui "github.com/egoist/mygo/ui"
)

// 右栏标签条与浮层：按参考产品的外壳清单重建
// （docs/domains/requirements/inventory-panels.md 区 A：RP-A-01…37）。
//
// 组成：拖拽手柄 → 已打开的标签页（图标 + 文案 + 悬停 ×/按住 ⌘ 的 ⌘n 徽标）
// → ＋新建标签页（六类，按可用性出现）→ 展开面板 / 恢复面板宽度。
// 标签页可关、可右键（在右侧新建 / 重新加载 / 关闭 / 关闭其他 / 关闭右侧）。

// 参考产品右栏的类型（RP-A-08…21）。0–6 沿用既有实现的下标（文件 / 改动 /
// 后台 / 目标 / 终端 / 侧聊 / 子代理），其余按清单顺序追加。
const (
	// tabBrowser 是浏览器页（RP-A-11）。
	tabBrowser = tabSubagent + 1
	// tabFilePreview 是文件预览页（RP-A-12）。
	tabFilePreview = tabBrowser + 1
	// tabWorkflow 是工作流页（RP-A-15）。
	tabWorkflow = tabFilePreview + 1
	// tabWakeup 是唤醒页（RP-A-17）。
	tabWakeup = tabWorkflow + 1
	// tabExtension 是扩展页（RP-A-19）。
	tabExtension = tabWakeup + 1
	// tabMCPApp 是 MCP App 页（RP-A-21）。
	tabMCPApp = tabExtension + 1
	// tabCount 是已登记的类型数。
	tabCount = tabMCPApp + 1
)

// 标签条与浮层的尺寸（design-baseline §3、§7）。
const (
	// panelHandleWidth 是拖拽手柄的宽（RP-A-01：4px）。
	panelHandleWidth = 4
	// panelTabHeight 是标签页高（实测 25，标签栏高 48）。
	panelTabHeight = 25
	// panelTabPadH 是标签页左右内边距（px-2）。
	panelTabPadH = space2
	// panelTabIcon 是标签页图标（h-3.5 = 14）。
	panelTabIcon = 14
	// panelBarAction 是标签条右端图标按钮的边长（实测 25）。
	panelBarAction = 25
	// panelFilterHeight 是筛选框高（行内 25）。
	panelFilterHeight = 25
	// panelRowHeight2 是文件树与改动列表的行高（实测 28）。
	panelRowHeight2 = rowRegular
	// panelRowPadH 是列表行的左右内边距（px-2）。
	panelRowPadH = space2
	// panelRowGap 是行内图标与文字之间的间距（gap-2）。
	panelRowGap = space2
	// panelIndent 是子级相对父级的缩进（ml-4 = 14）。
	panelIndent = space4
	// panelArrowSlot 是目录箭头与文件行占位的宽（w-3.5 = 14）。
	panelArrowSlot = panelTabIcon
	// panelRowIcon 是行内对象图标的边长（size-4 = 16）。
	panelRowIcon = iconInline
	// panelEmptyIcon 是空态圆底的直径（清单 F-09：48）。
	panelEmptyIcon = 48
	// panelSearchRowHeight 是搜索命中行的高（PL 态实测 44：文件名 + 次级目录）。
	panelSearchRowHeight = 44
	// panelGroupHeight 是改动分组标题的行高（实测 28）。
	panelGroupHeight = rowRegular
	// panelScopeHeight 是范围栏的行高（实测约 35）。
	panelScopeHeight = 35
	// panelFooterHeight 是底部统计条的高（实测 35）。
	panelFooterHeight = 35
	// panelMenuWidth 是浮层的最小宽（参考实测 145–168）。
	panelMenuWidth = 168
	// panelMenuNoteWidth 是浮层里两段文案的宽（168 − 内边距 − 图标）。
	panelMenuNoteWidth = 120
	// panelDiffLineHeight 是行内 diff 一行的高（代码行高 20）。
	panelDiffLineHeight = 20
	// panelDiffPadH 是行内 diff 的左右内边距。
	panelDiffPadH = space2
	// panelSkeletonHeight 是骨架条的高（清单 F-03：32）。
	panelSkeletonHeight = 32
	// panelBannerPadH 是权限条与错误条的左右内边距。
	panelBannerPadH = space2
	// panelBadgeSize 是「已暂存」圆点徽标的直径（清单 F-33：12–14，取 12）。
	panelBadgeSize = 12
	// panelStatWidth 是行尾 +n/−n 两列的定宽（右对齐成列）。
	panelStatWidth = 24
	// panelCloseZone 是标签页右端的关闭区宽（× 的命中盒）。
	panelCloseZone = addSpace
)

// addSpace / flushSpace 是拼尺寸用的常量（避免在排布里写表达式）。
const (
	addSpace   = 16
	flushSpace = 2 * space2
)

// rightPanelState 是右栏可丢的界面状态：开着的标签、筛选词、展开的目录、
// 改动范围、浮层与确认框，外加样例树与样例改动（样例不是事实）。
//
// 与 panelView 分开：panelView 是终端 / 任务 / 侧聊 / 子代理四页的视图状态，
// 本结构是标签条、文件页与改动页的状态。
type rightPanelState struct {
	// tabs 是打开的标签（顺序即 ⌘1…⌘9 的编号）；a.rightTab 指当前页。
	tabs []int

	filter string

	// dirOpen 是目录展开态（键是目录路径）；dirPages 是每个目录已加载的子项数。
	dirOpen  map[string]bool
	dirPages map[string]int
	// dirLoading / dirFailed 是「正在加载...」与目录错误行。
	dirLoading map[string]bool
	dirFailed  map[string]bool

	// tree 是文件树样例；changes 是改动列表样例。
	tree    []*rpNode
	changes []*rpChange

	// scope 是改动范围（0 未提交 / 1 当前分支 / 2 Last Run）。
	scope int
	// diffOpen 是行内 diff 展开的文件集合。
	diffOpen map[string]bool

	// 搜索态与读取失败：本机对样例做同步匹配，「正在搜索」只在一帧内出现；
	// 错误 / 部分失败 / 权限不足要有真实的文件读数才会出现。
	searching        bool
	searchFailed     bool
	searchPartial    bool
	permissionDenied bool
	rootPartial      bool
	// groupsClosed 是改动分组的收起态（默认展开）；键 "staged" / "unstaged"。
	groupsClosed map[string]bool

	// 浮层： ＋新建（可指定插入位置）、标签右键、行右键、范围下拉、提交菜单、
	// 分支子菜单。
	newMenu      bool
	pendingAfter int
	tabMenu      map[int]*bool
	rowMenu      map[string]*bool
	scopeOpen    bool
	gitOpen      bool
	branchOpen   bool

	// revertPath 非空时「还原未暂存改动」的确认框在场。
	revertPath string

	// width 是拖出来的宽度（0 = 布局契约的默认宽）；expanded 是「展开面板」。
	width    float32
	expanded bool

	// reload 是「重新加载」的计数（样例里只用于让行重建）。
	reload int
}

// rp 取右栏状态并在第一次用到时补默认值。
func (a *App) rp() *rightPanelState {
	s := &a.right
	if s.dirOpen == nil {
		s.dirOpen = map[string]bool{}
		s.dirPages = map[string]int{}
		s.dirLoading = map[string]bool{}
		s.dirFailed = map[string]bool{}
		s.diffOpen = map[string]bool{}
		s.tabMenu = map[int]*bool{}
		s.rowMenu = map[string]*bool{}
		s.tree, s.changes = rightSample()
		s.pendingAfter = -1
		s.expanded = false
	}
	if a.rightTab < 0 || a.rightTab >= tabCount {
		a.rightTab = tabFiles
	}
	if len(s.tabs) == 0 {
		s.tabs = []int{a.rightTab}
	}
	if !rpHasTab(s.tabs, a.rightTab) {
		s.tabs = append(s.tabs, a.rightTab)
	}
	return s
}

// rpHasTab 判断标签是否已打开。
func rpHasTab(tabs []int, tab int) bool {
	for _, t := range tabs {
		if t == tab {
			return true
		}
	}
	return false
}

// rpIndex 取标签在标签条里的序号，找不到返回 -1。
func rpIndex(tabs []int, tab int) int {
	for i, t := range tabs {
		if t == tab {
			return i
		}
	}
	return -1
}

// rpOpen 打开（或聚焦）一页，新页排在最后。
func (a *App) rpOpen(tab int) {
	s := a.rp()
	if !rpHasTab(s.tabs, tab) {
		s.tabs = append(s.tabs, tab)
	}
	a.rightTab = tab
}

// rpOpenAfter 在 index 之后插入一页（右键菜单「在右侧新建标签页」）。
func (a *App) rpOpenAfter(tab, index int) {
	s := a.rp()
	if !rpHasTab(s.tabs, tab) {
		if index < 0 || index >= len(s.tabs) {
			s.tabs = append(s.tabs, tab)
		} else {
			next := make([]int, 0, len(s.tabs)+1)
			next = append(next, s.tabs[:index+1]...)
			next = append(next, tab)
			next = append(next, s.tabs[index+1:]...)
			s.tabs = next
		}
	}
	a.rightTab = tab
}

// rpClose 关掉一页：关当前页时切到左邻（没有左邻取右邻）；全部关完右栏收起
// （R-TB-029：最后一个标签关掉，右栏自动收起）。
func (a *App) rpClose(tab int) {
	s := a.rp()
	index := rpIndex(s.tabs, tab)
	if index < 0 {
		return
	}
	s.tabs = append(s.tabs[:index:index], s.tabs[index+1:]...)
	if len(s.tabs) == 0 {
		a.rightOpen = false
		a.rightTab = tabFiles
		return
	}
	if a.rightTab != tab {
		return
	}
	if index > 0 {
		a.rightTab = s.tabs[index-1]
	} else {
		a.rightTab = s.tabs[0]
	}
}

// rpCloseOthers 关掉除 tab 之外的页。
func (a *App) rpCloseOthers(tab int) {
	s := a.rp()
	s.tabs = []int{tab}
	a.rightTab = tab
}

// rpCloseRight 关掉 tab 右侧的页。
func (a *App) rpCloseRight(tab int) {
	s := a.rp()
	index := rpIndex(s.tabs, tab)
	if index < 0 {
		return
	}
	s.tabs = s.tabs[:index+1]
	a.rightTab = tab
}

// rpWidth 是右栏实际占的宽度：展开态占满主区，否则取拖出来的宽度（默认
// 320），下限 280、上限让主区保住 400。
func (a *App) rpWidth(windowWidth, sidebar float32) float32 {
	if !a.rightOpen {
		return 0
	}
	s := a.rp()
	if s.expanded {
		if rest := windowWidth - sidebar; rest > 0 {
			return rest
		}
		return 0
	}
	maxPanel := windowWidth - sidebar - RightPanelMainMinWidth
	if maxPanel < RightPanelMinWidth {
		return 0
	}
	want := float32(RightPanelPreferredWidth)
	if s.width > 0 {
		want = s.width
	}
	return clamp(want, RightPanelMinWidth, maxPanel)
}

// rpHandle 是面板左缘的竖向拖拽手柄（RP-A-01：4px，悬停变主色）。
func (a *App) rpHandle(c *gui.Context) {
	t := c.Theme()
	h := gui.Box(c).Width(panelHandleWidth).FillHeight().Shrink(0).
		Label("拖拽调整面板宽度").Cursor(gui.CursorResizeEW)
	if h.Hovered() || h.Pressed() {
		h = h.Background(colorAccent.in(t))
	}
	if dx, _, dragging := h.Dragged(); dragging {
		s := a.rp()
		base := s.width
		if base <= 0 {
			base = RightPanelPreferredWidth
		}
		s.width = base - dx
	}
}

// rpTabBar 是标签条：标签页（可横向滚动）+ ＋新建标签页 + 展开面板。
func (a *App) rpTabBar(c *gui.Context, width float32) {
	s := a.rp()
	a.rpTabKeys(c)
	gui.Row(c).Height(HeaderHeight).Shrink(0).Padding(space1, space3).
		AlignItems(gui.Center).Gap(space1).Label("右栏标签").Children(func() {
		gui.Row(c).Grow(1).AlignItems(gui.Center).Gap(space1).Children(func() {
			for i, tab := range s.tabs {
				a.rpTabItem(c, tab, i)
			}
		})
		a.rpNewTabButton(c)
		a.rpExpandButton(c)
	})
	_ = width
}

// rpTabKeys 是 ⌘1…⌘9 切页（RP-A-22）：标签多于 1 个时才生效。
func (a *App) rpTabKeys(c *gui.Context) {
	s := a.rp()
	if len(s.tabs) < 2 {
		return
	}
	keys := [...]gui.Key{gui.Key1, gui.Key2, gui.Key3, gui.Key4, gui.Key5,
		gui.Key6, gui.Key7, gui.Key8, gui.Key9}
	for i, key := range keys {
		if i >= len(s.tabs) {
			break
		}
		if c.Shortcut(gui.Cmd, key) {
			a.rightTab = s.tabs[i]
			return
		}
	}
}

// rpTabItem 是一个标签页：图标 + 文案 + 悬停 ×（RP-A-23）或按住 ⌘ 的徽标
// （RP-A-22）；单击切页，单击 × 关页，右键出标签菜单（RP-A-31…35）。
func (a *App) rpTabItem(c *gui.Context, tab, index int) {
	t := c.Theme()
	s := a.rp()
	active := tab == a.rightTab
	label := a.rpTabLabel(tab)
	b := gui.ButtonBase(c).Height(panelTabHeight).Padding(space1, panelTabPadH).
		Radius(radiusSmall).Gap(space1).AlignItems(gui.Center).
		Label(label).Tooltip(label).Cursor(gui.CursorPointer)
	switch {
	case active:
		b = b.Background(t.SurfaceHover)
		// 当前页的轮廓：自绘 1px，颜色取主题边框（不新增第二级边框色）。
		b.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Stroke(r, t.Border, radiusSmall, strokeHairline)
		})
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	hovered := b.Hovered()
	badge := c.Modifiers()&gui.Cmd != 0 && len(s.tabs) > 1 && index < 9
	var closeBox gui.Element
	b.Children(func() {
		rpGlyph(c, rpTabIcon(tab), panelTabIcon, t.TextMuted, "")
		typeSecondary.style(t, gui.Text(c, label)).TextColor(t.Text).SingleLine()
		switch {
		case badge:
			typeMicro.style(t, gui.Text(c, rpShortcut(index+1))).TextColor(t.TextMuted)
		case hovered:
			closeBox = gui.Box(c).Width(panelTabIcon).Height(panelTabIcon).Shrink(0).
				Label("关闭标签页：" + label)
			closeBox.DrawOver(func(p *gui.Painter, r gui.Rect) {
				p.Icon(iconXMark, r, t.TextMuted)
			})
		}
	})
	clicked := b.Clicked()
	closing := closeBox.Hovered()
	if clicked {
		if closing && !badge {
			a.rpClose(tab)
		} else {
			a.rightTab = tab
		}
	}
	a.rpTabMenu(c, b, tab)
}

// rpTabMenu 是标签的右键菜单（RP-A-31…35）。
func (a *App) rpTabMenu(c *gui.Context, target gui.Element, tab int) {
	s := a.rp()
	m := gui.ContextMenuBase(c, target, s.tabMenuPtr(tab))
	m.Popup(func(panel gui.Element) {
		rpMenuPanel(panel, c.Theme(), panelMenuWidth)
		if menuItem(c, m, iconPanelRight, "在右侧新建标签页", "在右侧新建标签页", "", false, false) {
			s.pendingAfter = rpIndex(s.tabs, tab)
			s.newMenu = true
		}
		if rpReloadable(tab) {
			if menuItem(c, m, iconRetry, "重新加载", "重新加载", "", false, false) {
				s.reload++
			}
		}
		if menuItem(c, m, iconXMark, "关闭", "关闭标签页", "", false, false) {
			a.rpClose(tab)
		}
		if menuItem(c, m, iconPanelClose, "关闭其他标签页", "关闭其他标签页", "",
			len(s.tabs) < 2, false) {
			a.rpCloseOthers(tab)
		}
		index := rpIndex(s.tabs, tab)
		if menuItem(c, m, iconPanelClose, "关闭右侧标签页", "关闭右侧标签页", "",
			index < 0 || index >= len(s.tabs)-1, false) {
			a.rpCloseRight(tab)
		}
	})
}

// rpNewTabItem 是「＋」菜单的一项（RP-A-25…30）。
type rpNewTabItem struct {
	tab     int
	icon    *gui.SVG
	text    string
	enabled func(*App, *rightPanelState) bool
}

// rpNewTabItems 是参考产品的六项。可用性按参考产品：已有同类页时不再出现；
// 终端要有工作目录；侧边对话要有活跃会话；浏览器与扩展要有对应能力——
// 本产品还没有这两项能力，故不出现。
func rpNewTabItems() []rpNewTabItem {
	notOpen := func(tab int) func(*App, *rightPanelState) bool {
		return func(_ *App, s *rightPanelState) bool { return !rpHasTab(s.tabs, tab) }
	}
	return []rpNewTabItem{
		{tab: tabFiles, icon: iconFolders, text: "文件", enabled: notOpen(tabFiles)},
		{tab: tabChanges, icon: iconDiff, text: "Review", enabled: notOpen(tabChanges)},
		{tab: tabBrowser, icon: iconGlobe, text: "浏览器", enabled: func(*App, *rightPanelState) bool { return false }},
		{tab: tabTerminal, icon: iconTerminal, text: "终端", enabled: notOpen(tabTerminal)},
		{tab: tabSideChat, icon: iconChat, text: "侧边对话", enabled: func(a *App, s *rightPanelState) bool {
			return !rpHasTab(s.tabs, tabSideChat) && a.selectedSession != ""
		}},
		{tab: tabExtension, icon: iconSidebarGrid, text: "扩展", enabled: func(*App, *rightPanelState) bool { return false }},
		// 后台进程在参考产品里由输入区的活动条打开，本产品还没有那个入口，
		// 暂列进 ＋ 菜单（有任务时才出现）；接入活动条后从这里撤掉。
		{tab: tabTasks, icon: iconTerminal, text: "后台进程", enabled: func(a *App, s *rightPanelState) bool {
			return !rpHasTab(s.tabs, tabTasks) && len(a.sessionTaskList()) > 0
		}},
	}
}

// rpNewTabButton 是标签条右端的「＋新建标签页」（RP-A-24）。
func (a *App) rpNewTabButton(c *gui.Context) {
	t := c.Theme()
	s := a.rp()
	m := gui.MenuBase(c, &s.newMenu)
	tr := m.Trigger.Width(panelBarAction).Height(panelBarAction).Radius(radiusSmall).
		Center().Label("新建标签页").Tooltip("新建标签页").Cursor(gui.CursorPointer)
	switch {
	case tr.Pressed():
		tr = tr.Background(t.SurfacePressed)
	case tr.Hovered():
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() {
		rpGlyph(c, iconPlus, iconInline, t.TextMuted, "")
	})
	m.Popup(func(panel gui.Element) {
		rpMenuPanel(panel, t, panelMenuWidth)
		for _, item := range rpNewTabItems() {
			if !item.enabled(a, s) {
				continue
			}
			tab := item.tab
			text := item.text
			if tab == tabTasks {
				text = fmt.Sprintf("后台进程 %d", len(a.sessionTaskList()))
			}
			if menuItem(c, m, item.icon, text, text, "", false, false) {
				if s.pendingAfter >= 0 {
					a.rpOpenAfter(tab, s.pendingAfter)
				} else {
					a.rpOpen(tab)
				}
				s.pendingAfter = -1
			}
		}
	})
}

// rpExpandButton 是「展开面板 / 恢复面板宽度」（RP-A-36/37）。
func (a *App) rpExpandButton(c *gui.Context) {
	s := a.rp()
	icon, label := iconMaximize, "展开面板"
	if s.expanded {
		icon, label = iconMinimize, "恢复面板宽度"
	}
	if iconButton(c, iconButtonSpec{icon: icon, label: label, tip: label, size: panelBarAction}) {
		s.expanded = !s.expanded
	}
}

// rpTabLabel 是标签文案（RP-A-09…21）：后台进程与唤醒带计数。
func (a *App) rpTabLabel(tab int) string {
	switch tab {
	case tabFiles:
		return "文件"
	case tabChanges:
		return "Review"
	case tabTasks:
		return fmt.Sprintf("后台进程 %d", len(a.sessionTaskList()))
	case tabGoal:
		return "目标"
	case tabTerminal:
		return "终端"
	case tabSideChat:
		return "侧边对话"
	case tabSubagent:
		return "子代理"
	case tabBrowser:
		return "浏览器"
	case tabFilePreview:
		return "打开文件"
	case tabWorkflow:
		return "工作流"
	case tabWakeup:
		return fmt.Sprintf("唤醒 %d", len(a.sessionTaskList()))
	case tabExtension:
		return "扩展"
	case tabMCPApp:
		return "MCP App"
	}
	return "标签页"
}

// rpTabIcon 是标签图标（RP-A-08、参考产品 :324–:339）。
func rpTabIcon(tab int) *gui.SVG {
	switch tab {
	case tabFiles:
		return iconFolders
	case tabChanges:
		return iconDiff
	case tabTasks, tabTerminal:
		return iconTerminal
	case tabGoal:
		return iconTarget
	case tabSideChat, tabSubagent:
		return iconChat
	case tabBrowser:
		return iconGlobe
	case tabFilePreview:
		return iconDocument
	case tabWorkflow:
		return iconWorkflow
	case tabWakeup:
		return iconClock
	case tabExtension:
		return iconSidebarGrid
	case tabMCPApp:
		return iconAppWindow
	}
	return iconDocument
}

// rpReloadable 是「重新加载」适用的标签类型（RP-A-32）。
func rpReloadable(tab int) bool {
	switch tab {
	case tabFiles, tabChanges, tabBrowser, tabFilePreview:
		return true
	}
	return false
}

// rpShortcut 是标签上的快捷键徽标（RP-A-22）：macOS 用 ⌘n，其它平台 Ctrl+n。
func rpShortcut(n int) string {
	if runtime.GOOS == "darwin" {
		return fmt.Sprintf("⌘%d", n)
	}
	return fmt.Sprintf("Ctrl+%d", n)
}

// rpMenuPanel 是浮层与菜单的面板样式：surface 面、卡片圆角、1px 主题描边、投影。
func rpMenuPanel(panel gui.Element, t *gui.Theme, width float32) {
	panel.MinWidth(width).Padding(space1).Radius(radiusCard).Background(t.Surface).
		Shadow(0, 4, 16, 0, colorShadow.in(t))
	panel.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, radiusCard, strokeHairline)
	})
}

// rpRule 是一条 1px 分隔线：用 1px 高的面差表达，不新增自绘描边。
func rpRule(c *gui.Context) {
	t := c.Theme()
	gui.Box(c).Shrink(0).Height(strokeHairline).Background(t.Border).
		Label("分隔线").AccessHidden(true)
}

// rpGlyph 画一个固定尺寸的装饰图标（不随字号变化），mark 非空时带检查器标记。
func rpGlyph(c *gui.Context, icon *gui.SVG, size float32, color gui.Color, mark string) {
	box := gui.Box(c).Width(size).Height(size).Shrink(0)
	if mark != "" {
		box = box.Label("图标：" + mark).AccessHidden(true)
	}
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Icon(icon, r, color)
	})
}

// rpTextAction 是行内的「图标 + 文字」小按钮（「全部暂存」「全部取消暂存」这类）：
// 默认透明，悬停浅底，禁用整棵子树 0.5。
func rpTextAction(c *gui.Context, icon *gui.SVG, text, label string, disabled bool) bool {
	t := c.Theme()
	if label == "" {
		label = text
	}
	b := gui.ButtonBase(c).Shrink(0).Height(rowCompact).Padding(0, space1).
		Radius(radiusSmall).Gap(space1).AlignItems(gui.Center).
		Label(label).Cursor(gui.CursorPointer).Disabled(disabled)
	switch {
	case disabled:
		b = b.Opacity(0.5)
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		if icon != nil {
			rpGlyph(c, icon, panelRowIcon, t.TextMuted, "")
		}
		typeSecondary.style(t, gui.Text(c, text)).TextColor(t.TextMuted).SingleLine()
	})
	return b.Clicked()
}

// rpOutlineButton 是带 1px 轮廓的次按钮（对话框的「取消」）：描边用 Draw，
// 颜色取主题边框。
func rpOutlineButton(c *gui.Context, text, label string) bool {
	t := c.Theme()
	if label == "" {
		label = text
	}
	b := gui.ButtonBase(c).Shrink(0).Height(rowRegular).Padding(space1, space2).
		Radius(radius).Label(label).Cursor(gui.CursorPointer)
	b.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, radius, strokeHairline)
	})
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		typeSecondary.style(t, gui.Text(c, text)).TextColor(t.Text).SingleLine()
	})
	return b.Clicked()
}

// rpDangerButton 是危险色的主按钮（「还原」）。
func rpDangerButton(c *gui.Context, text, label string) bool {
	t := c.Theme()
	if label == "" {
		label = text
	}
	b := gui.ButtonBase(c).Shrink(0).Height(rowRegular).Padding(space1, space2).
		Radius(radius).Background(t.Danger).Label(label).Cursor(gui.CursorPointer)
	switch {
	case b.Pressed():
		b = b.Opacity(0.85)
	case b.Hovered():
		b = b.Opacity(0.9)
	}
	b.Children(func() {
		typeSecondary.style(t, gui.Text(c, text)).TextColor(t.AccentText).SingleLine()
	})
	return b.Clicked()
}

// rpTabMenuPtr 取某个标签右键菜单的开合指针（视图状态，按需创建）。
func (s *rightPanelState) tabMenuPtr(tab int) *bool {
	if s.tabMenu == nil {
		s.tabMenu = map[int]*bool{}
	}
	if p, ok := s.tabMenu[tab]; ok {
		return p
	}
	open := false
	s.tabMenu[tab] = &open
	return &open
}

// rpRowMenuPtr 取某一行右键菜单的开合指针。
func (s *rightPanelState) rowMenuPtr(key string) *bool {
	if s.rowMenu == nil {
		s.rowMenu = map[string]*bool{}
	}
	if p, ok := s.rowMenu[key]; ok {
		return p
	}
	open := false
	s.rowMenu[key] = &open
	return &open
}

// sessionTaskList 是标签文案里后台任务 / 唤醒的计数来源（与任务页同源）。
func (a *App) sessionTaskList() []task {
	tasks, _ := a.sessionTasks()
	return tasks
}

// 标签条与面板用到的新形状：与既有图标同一 24 网格、1.5 描边。
var (
	// iconFolders 是「文件」标签（lucide folders）。
	iconFolders = mustSVG(`<path d="M9 3.75h3.75a1.5 1.5 0 0 1 1.5 1.5v.75M3.75 6.75A1.5 1.5 0 0 1 5.25 5.25h4.5l1.5 1.5h7.5a1.5 1.5 0 0 1 1.5 1.5v.75"/><path d="M3.75 6.75h13.5a1.5 1.5 0 0 1 1.5 1.5v2.25"/><path d="M3.75 6.75v10.5a1.5 1.5 0 0 0 1.5 1.5h12a1.5 1.5 0 0 0 1.5-1.5V12"/>`)
	// iconDiff 是「Review」标签（lucide git-compare）。
	iconDiff = mustSVG(`<circle cx="6" cy="6" r="2.25"/><circle cx="18" cy="18" r="2.25"/><path d="M6 8.25v5.25A2.25 2.25 0 0 0 8.25 15.75h7.5"/><path d="M18 15.75V10.5A2.25 2.25 0 0 0 15.75 8.25h-7.5"/>`)
	// iconMaximize / iconMinimize 是「展开面板 / 恢复面板宽度」。
	iconMaximize = mustSVG(`<path d="M3.75 9V5.25a1.5 1.5 0 0 1 1.5-1.5H9M15 3.75h3.75a1.5 1.5 0 0 1 1.5 1.5V9M20.25 15v3.75a1.5 1.5 0 0 1-1.5 1.5H15M9 20.25H5.25a1.5 1.5 0 0 1-1.5-1.5V15"/>`)
	iconMinimize = mustSVG(`<path d="M9 3.75V9H3.75M15 3.75V9h5.25M20.25 15H15v5.25M3.75 15H9v5.25"/>`)
	// iconMinus 是「取消暂存 / 全部取消暂存」。
	iconMinus = mustSVG(`<path d="M5.25 12h13.5"/>`)
	// iconGitCommit 是「提交」。
	iconGitCommit = mustSVG(`<circle cx="12" cy="12" r="3"/><path d="M3 12h6M15 12h6"/>`)
	// iconCloudUpload 是「提交或推送 / 推送」。
	iconCloudUpload = mustSVG(`<path d="M12 15.75V8.25m0 0-3 3m3-3 3 3"/><path d="M7.5 18.75h-.75a4.5 4.5 0 0 1-.6-8.96 6 6 0 0 1 11.7 0 4.5 4.5 0 0 1-.6 8.96h-.75"/>`)
	// iconAppWindow 是 MCP App 标签（lucide app-window）。
	iconAppWindow = mustSVG(`<rect x="3.75" y="4.5" width="16.5" height="15" rx="2.25"/><path d="M3.75 9h16.5"/><path d="M7.5 6.75h.008v.008H7.5V6.75Z"/>`)
	// iconCircleCheck 是改动空态的圆底图标（lucide circle-check）。
	iconCircleCheck = mustSVG(`<circle cx="12" cy="12" r="8.25"/><path d="m8.25 12.375 2.625 2.625 5.25-5.25"/>`)
	// iconFileDiff 是文件行「查看差异」（lucide file-diff）。
	iconFileDiff = mustSVG(`<path d="M14.25 3.75H6.75a1.5 1.5 0 0 0-1.5 1.5v13.5a1.5 1.5 0 0 0 1.5 1.5h10.5a1.5 1.5 0 0 0 1.5-1.5V7.5z"/><path d="M14.25 3.75V7.5h4.5"/><path d="M9 13.5h2.25M10.125 12.375v2.25M15 12.75h-2.25"/>`)
	// iconFolderOpenSmall 是展开目录的图标（复用左栏的同形）。
)

// colorSearchHit 是搜索命中的黄底（参考产品的命中高亮；设计基线未列此 token）。
var colorSearchHit = colorSet{gui.RGBA(255, 214, 0, 0.35), gui.RGBA(255, 200, 0, 0.28)}

// colorStagedDot 是「已暂存」圆点的蓝底（参考产品的信息色）。
var colorStagedDot = colorSet{gui.Hex("#2b7fff"), gui.Hex("#51a2ff")}
