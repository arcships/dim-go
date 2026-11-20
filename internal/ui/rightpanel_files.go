package ui

import (
	"fmt"
	"path"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 文件页：按 inventory-panels-rt.md §1 与 requirements-panels.md §1 重建。
//
// 组成：筛选框（与改动页共用一处，R-RF-002）→ 权限条（F-46）→ 主树 / 搜索
// 结果 / 空态 / 目录错误。行高 28；目录行的箭头 14、文件夹图标 16、名字与
// 文件行的名字同列（R-RF-160、R-RF-240）。
//
// 事实来源：连着 Source 且适配器实现 PanelFacts 时，树与搜索读真实读数
// （D7 的列目录与内容搜索）；只在 -demo（没有 Source）时用样例，样例见
// rightpanel_sample.go。搜索的读态（进行中 / 失败 / 部分失败 / 截断）都来自
// 快照，不在视图里猜。

// rpFilesPage 是文件页本体。
func (a *App) rpFilesPage(c *gui.Context) {
	s := a.rp()
	// 目录投影每帧在 View 里做一次（命令面板的文件组也读它）；这里只处理
	// 「刷新 / 重试」那一次额外重读。
	refresh := a.rpTakeRefresh()
	if refresh {
		a.syncPanelTree(true)
	}
	q := rpQuery(s.filter)
	gui.Column(c).Grow(1).Children(func() {
		a.rpOfflineBar(c)
		a.rpFilterRow(c)
		if s.permissionDenied {
			a.rpPermissionBar(c)
		}
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Padding(space1, rpTreePadH).
				Role(gui.RoleTree).Label("文件树").Children(func() {
				switch {
				case s.emptyTitle != "":
					a.rpEmptyBlock(c, s.emptyTitle, s.emptyDetail)
				case q != "":
					a.rpSearchResults(c, q, refresh)
				case s.loading && len(s.tree) == 0:
					a.rpSkeleton(c)
				case s.rootFailed:
					a.rpDirErrorRow(c, "")
				default:
					a.rpTreeRows(c, s.tree, "", 0)
					if s.rootPartial {
						a.rpLoadMoreRow(c, "")
					}
				}
			})
		})
	})
}

// rpTreePadH 是树区的左右留白（R 态行 410 对面板 405）。
const rpTreePadH = space1

// rpFacts 取面板读数：只有连着 Source 且适配器实现了 PanelFacts 时才为真。
// 不满足时（-demo 与截图）两页仍用样例，样例不是事实。
func (a *App) rpFacts() (PanelFacts, bool) {
	if a.source == nil {
		return nil, false
	}
	facts, ok := a.source.(PanelFacts)
	return facts, ok
}

// rpWired 表示两页现在读的是真实读数（而不是样例）。
func (a *App) rpWired() bool {
	_, ok := a.rpFacts()
	return ok
}

// rpCwd 是会话的工作目录（目录事实）；空表示服务端没有给。
func (a *App) rpCwd() string {
	if snap, ok := a.liveSession(); ok {
		return snap.Cwd
	}
	return ""
}

// rpNoWorkspace 返回两页在没有可读工作目录时的空态标题与说明；标题为空表示
// 有工作目录可读。
//
// 连着 Source 时工作目录只来自会话事实：没打开会话是「未打开会话」，会话没有
// cwd 是「没有工作区」。两种情况都不回落到样例，也不拿工作区根顶替——那会把
// 别的目录的东西画成这个会话的文件。样例只在 -demo（没有 Source）时出现。
func (a *App) rpNoWorkspace() (string, string) {
	if a.source == nil {
		return "", ""
	}
	snap, ok := a.liveSession()
	if !ok {
		return "未打开会话", "打开一个会话后，这里显示它的工作目录。"
	}
	if strings.TrimSpace(snap.Cwd) == "" {
		return "没有工作区", "这个会话没有工作目录，文件与改动都读不到。"
	}
	return "", ""
}

// rpReloadNote 是「重新加载 / 刷新」不能用的原因：没有可读的工作目录时重读
// 也读不出东西，那是一个点了没反应的入口（U3 §13.0 第 4 条）。
func (a *App) rpReloadNote() string {
	if title, _ := a.rpNoWorkspace(); title != "" {
		return "没有可读的工作目录，暂时不能重新加载"
	}
	return ""
}

// rpTakeRefresh 取一次「刷新」意图：按钮只把计数加一，这里读一次后归零，
// 保证一次点击只触发一次重读，不是每帧重读。
func (a *App) rpTakeRefresh() bool {
	s := a.rp()
	if s.reload == 0 {
		return false
	}
	s.reload = 0
	return true
}

// rpOfflineBar 是断线横幅：面板的本机读数照常显示，但连接阶段与原因要看得见。
func (a *App) rpOfflineBar(c *gui.Context) {
	conn := a.liveConnection()
	if conn.Phase != "disconnected" {
		return
	}
	panelBanner(c, "连接已断开："+conn.Reason, "重新连接", func() {
		if a.intents != nil {
			a.dispatch("重新连接", a.intents.Reconnect, nil)
		}
	})
}

// rpMaxDepth 是树的展开深度上限：异常深的目录不再往下读，避免一次读光整棵树。
const rpMaxDepth = 16

// syncPanelTree 每帧把文件页的目录投影一次。
//
// 文件页与命令面板的文件组读同一份投影（都读 s.tree），所以这里在 View 里做，
// 而不是只在文件页渲染时做。右栏没开、命令面板也没开时不读，省掉没人看的读。
func (a *App) syncPanelTree(refresh bool) {
	facts, ok := a.rpFacts()
	if !ok {
		return
	}
	// 提及弹层的文件候选读同一份目录投影：它在场时也要读（见 mention.go）。
	if !refresh && !a.rightOpen && !a.palette.open && !a.pick.open {
		return
	}
	a.rpLiveTree(facts, refresh)
}

// rpLiveTree 用目录读数重建这一帧的树。
//
// 每帧重建，视图只留可丢状态（展开态、分页位置）。读不到时把原因落到
// rootFailed / permissionDenied / dirFailed 上，由既有的错误行画出来。
// 没有可读的工作目录时不派发读数：树清空、两页都给空态。
func (a *App) rpLiveTree(facts PanelFacts, refresh bool) {
	s := a.rp()
	if title, detail := a.rpNoWorkspace(); title != "" {
		s.loading, s.rootFailed, s.permissionDenied, s.rootPartial = false, false, false, false
		s.emptyTitle, s.emptyDetail = title, detail
		s.tree = nil
		return
	}
	s.emptyTitle, s.emptyDetail = "", ""
	cwd := a.rpCwd()
	// 根目录的分页位置也存在 dirPages 里（键 "" 是根），「加载更多」把它加上一页。
	limit := s.dirPages[""]
	if limit < rpPageSize {
		limit = rpPageSize
		s.dirPages[""] = limit
	}
	root := facts.FileTree(FileTreeRequest{
		Cwd: cwd, Offset: 0, Limit: limit, Refresh: refresh,
	})
	s.loading = root.Phase == PanelReadLoading
	s.rootFailed = root.Phase == PanelReadFailed
	s.rootError = root.Error
	s.dirError[""] = root.Error
	s.permissionDenied = root.Denied
	s.rootPartial = root.HasMore
	s.tree = a.rpNodes(facts, cwd, "", root, refresh, 0)
}

// rpNodes 把一个目录快照映射成树节点；展开的目录再把子级读一页。
func (a *App) rpNodes(facts PanelFacts, cwd, dir string, snapshot FileTreeSnapshot, refresh bool, depth int) []*rpNode {
	s := a.rp()
	nodes := make([]*rpNode, 0, len(snapshot.Entries))
	for _, entry := range snapshot.Entries {
		p := rpJoinPath(dir, entry.Name)
		node := &rpNode{name: entry.Name, dir: entry.Dir, change: rpEntryChange(p, entry.Change)}
		if entry.Dir && s.dirOpen[p] && depth < rpMaxDepth {
			limit := s.dirPages[p]
			if limit < rpPageSize {
				limit = rpPageSize
				s.dirPages[p] = limit
			}
			// 这个目录刚点过「重试」时带 Refresh 重读一次，然后清掉标记：只重读
			// 这一格，不做整树刷新（F-26 / R-RF-219）。
			childRefresh := refresh
			if s.dirRefresh[p] {
				childRefresh = true
				delete(s.dirRefresh, p)
			}
			child := facts.FileTree(FileTreeRequest{
				Cwd: cwd, Dir: p, Offset: 0, Limit: limit, Refresh: childRefresh,
			})
			s.dirFailed[p] = child.Phase == PanelReadFailed
			s.dirError[p] = child.Error
			s.dirLoading[p] = child.Phase == PanelReadLoading
			node.hasMore = child.HasMore
			node.children = a.rpNodes(facts, cwd, p, child, refresh, depth+1)
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// rpEntryChange 把条目上的 git 状态字母映射成行上的改动标记。
//
// 只有合并状态（没有暂存列、没有 ± 行数），所以行上不再画数字：statUnknown
// 让行尾的 +n/−n 不出现，而不是画一个假的 0。
func rpEntryChange(p, letter string) *rpChange {
	if letter == "" {
		return nil
	}
	return &rpChange{path: p, status: letter[0], statUnknown: true}
}

// rpGitWriteNote 是 git 写动作不可用的原因（暂存 / 取消暂存 / 还原 / 提交 /
// 推送 / 切分支）。技术口径是「缺协议方法，D7 不做 git 写」，界面上只写结果
// （DEC-03、UI-3）。
const rpGitWriteNote = noteGitWrite

// rpGitWriteDisabled 表示这些写动作现在不能发：连着真实读数时一律禁用。
func (a *App) rpGitWriteDisabled() bool { return a.rpWired() }

// rpQuery 是筛选词：去首尾空白；空串表示不在搜索态。
func rpQuery(q string) string { return strings.TrimSpace(q) }

// rpFilterRow 是筛选框：前置放大镜、占位「筛选文件...」、无边框无底色，高 25
// （RP-B-01 / F-01）。
//
// 没有可读的工作目录时筛选框禁用并写原因：输入了也筛不出东西——那是一个
// "看起来能用、按了没反应"的入口（U3 §13.1）。
func (a *App) rpFilterRow(c *gui.Context) {
	t := c.Theme()
	s := a.rp()
	note := a.rpFilterNote()
	gui.Column(c).Shrink(0).Padding(space1, space3).Children(func() {
		gui.Row(c).Height(panelFilterHeight).Gap(space1).AlignItems(gui.Center).
			Children(func() {
				rpGlyph(c, iconSearch, iconInline, t.TextMuted, "search")
				input := gui.TextInputBase(c, &s.filter).Grow(1).Label("筛选文件").
					Placeholder("筛选文件...").FontSize(typeSecondary.sizeIn(t)).
					TextColor(t.Text).Disabled(note != "")
				if note != "" {
					input = input.Tooltip(note)
				}
			})
	})
	// 焦点在框内按 Esc：退出搜索态并清空查询（F-01、R-RF-007）。
	if s.filter != "" && c.FocusTakesText() && c.Shortcut(0, gui.KeyEscape) {
		s.filter = ""
		s.searching = false
	}
}

// rpFilterNote 是筛选框不可用的原因；空串表示可以筛选。
//
// 两种"没有工作目录"分开写：没打开会话与会话没有 cwd 是两件事，改法也不同。
func (a *App) rpFilterNote() string {
	if a.source == nil {
		return ""
	}
	if _, ok := a.liveSession(); !ok {
		return rpFilterNoSessionNote
	}
	if strings.TrimSpace(a.rpCwd()) == "" {
		return rpFilterNoCwdNote
	}
	return ""
}

const (
	// rpFilterNoSessionNote 是没打开会话时筛选框的禁用原因。
	rpFilterNoSessionNote = "打开一个会话后才能筛选文件"
	// rpFilterNoCwdNote 是会话没有工作目录时筛选框的禁用原因。
	rpFilterNoCwdNote = "这个会话没有工作目录，暂时不能筛选文件"
)

// rpSkeleton 是文件树与改动列表首次加载的三条骨架（F-03 / D-16）：每条高 32、
// 左缘与树行同级；中性浅块、深色主题换深灰块，不用品牌色。容器用状态角色表达
// 「正在加载」（读屏只说一次），三条骨架本身不单独播报。
func (a *App) rpSkeleton(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Label("正在加载文件树").Role(gui.RoleStatus).Children(func() {
		for i := 0; i < rpSkeletonRows; i++ {
			// D-16 骨架条：中性浅块，宽度撑满列表宽（窄窗下随列表收缩）；
			// 三条同步脉冲（同一 key、同一周期），减少动效时框架直接停在稳态。
			row := gui.Row(c).Height(panelSkeletonHeight).Padding(0, panelRowPadH).
				AlignItems(gui.Center)
			pulse := row.Loop("skeleton-pulse", rpSkeletonPulse, gui.EaseInOut)
			row.Opacity(0.55 + 0.45*(1-abs32(2*pulse-1))).Children(func() {
				gui.Box(c).Height(panelRowHeight2).Grow(1).
					Radius(radiusSmall).Background(t.SurfaceHover)
			})
		}
	})
}

// rpSkeletonRows 是骨架条数（F-03：3 条）。
const rpSkeletonRows = 3

// rpSkeletonPulse 是骨架脉冲的周期（参考产品的骨架呼吸约 1.2s）。
const rpSkeletonPulse = 1200 * time.Millisecond

// abs32 取绝对值（脉冲按三角波来回）。
func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// rpPermissionBar 是没有写入权限时的警告条（F-46–F-48）：琥珀底、告警图标、
// 文案与行尾「重试」。
func (a *App) rpPermissionBar(c *gui.Context) {
	t := c.Theme()
	s := a.rp()
	bar := gui.Row(c).Shrink(0).Padding(space1, panelBannerPadH).Gap(panelRowGap).
		AlignItems(gui.Center).Role(gui.RoleAlert).Label("没有写入权限，部分操作不可用").
		Background(t.Warning.Alpha(0.12))
	bar.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Warning.Alpha(0.35), radiusSmall, strokeHairline)
	})
	bar.Children(func() {
		rpGlyph(c, iconWarn, panelRowIcon, t.Warning, "warning")
		typeSecondary.style(t, gui.Text(c, "没有写入权限，部分操作不可用")).
			TextColor(t.Warning).Grow(1).SingleLine()
		if rpTextAction(c, iconRetry, "重试", "重试读取文件树", false) {
			s.permissionDenied = false
			s.reload++
		}
	})
}

// rpTreeRows 画一层树：目录行（可展开）与文件行；子级再缩进 14（F-22）。
func (a *App) rpTreeRows(c *gui.Context, nodes []*rpNode, prefix string, depth int) {
	s := a.rp()
	if len(nodes) == 0 {
		if depth == 0 {
			// 读到了、但这一层没有条目：空态，不是错误态，也不是搜索无命中。
			a.rpEmptyBlock(c, rpEmptyDirTitle, rpEmptyDirDetail)
		}
		return
	}
	for _, node := range nodes {
		node := node
		p := node.name
		if prefix != "" {
			p = prefix + "/" + node.name
		}
		if node.dir {
			a.rpDirRow(c, node, p, depth)
			if !s.dirOpen[p] {
				continue
			}
			if s.dirFailed[p] {
				a.rpDirErrorRow(c, p)
				continue
			}
			gui.Column(c).PaddingStart(panelIndent).Children(func() {
				loaded := a.rpDirLoaded(node, p)
				a.rpTreeRows(c, node.children[:loaded], p, depth+1)
				if loaded < len(node.children) || node.hasMore {
					a.rpLoadMoreRow(c, p)
				}
			})
			continue
		}
		a.rpFileRow(c, node, p, depth)
	}
}

// rpDirLoaded 是目录已加载的子项数：第一次展开给一页，之后按「加载更多」累加。
func (a *App) rpDirLoaded(node *rpNode, p string) int {
	s := a.rp()
	loaded, ok := s.dirPages[p]
	if !ok {
		loaded = rpPageSize
		if loaded > len(node.children) {
			loaded = len(node.children)
		}
		s.dirPages[p] = loaded
	}
	if loaded > len(node.children) {
		loaded = len(node.children)
	}
	return loaded
}

// rpPageSize 是目录与树的分页大小（清单 R-RF-190 的一批子项）。
const rpPageSize = 20

// 空目录的文案：读到空目录与"搜索没有命中"是两种空，不能共用一句
// （U3 §13.1）。
const (
	rpEmptyDirTitle  = "这个文件夹是空的"
	rpEmptyDirDetail = "这里还没有文件。"
)

// rpDirRow 是目录行：箭头 + 文件夹图标 + 目录名（F-17–F-21）。
func (a *App) rpDirRow(c *gui.Context, node *rpNode, p string, depth int) {
	t := c.Theme()
	s := a.rp()
	open := s.dirOpen[p]
	state := "已收起"
	if open {
		state = "已展开"
	}
	row := gui.ButtonBase(c).Height(panelRowHeight2).Padding(0, panelRowPadH).
		Gap(panelRowGap).AlignItems(gui.Center).Radius(radiusSmall).
		Role(gui.RoleTreeItem).Expanded(open).Level(depth + 1).
		Label("目录，" + node.name + "，" + state).Cursor(gui.CursorPointer)
	switch {
	case row.Pressed():
		row = row.Background(t.SurfacePressed)
	case row.Hovered():
		row = row.Background(t.SurfaceHover)
	}
	if row.Clicked() {
		s.dirOpen[p] = !open
	}
	row.Children(func() {
		arrow := iconChevronR
		if open {
			arrow = iconChevronDown
		}
		rpGlyph(c, arrow, panelArrowSlot, t.TextMuted, "")
		folder := iconSidebarFolder
		if open {
			folder = iconSidebarFolderOpen
		}
		rpGlyph(c, folder, panelRowIcon, t.TextMuted, "")
		color := t.Text
		switch {
		case strings.HasPrefix(node.name, "."):
			color = t.TextMuted
		case rpDirChanged(node) || node.change != nil:
			color = t.Warning
		}
		typeControl.style(t, gui.Text(c, node.name)).TextColor(color).Grow(1).SingleLine()
	})
}

// rpDirChanged 判断目录的后代里有没有改动（有则目录名转琥珀色，F-21）。
func rpDirChanged(node *rpNode) bool {
	for _, child := range node.children {
		if child.dir {
			if rpDirChanged(child) {
				return true
			}
			continue
		}
		if child.change != nil {
			return true
		}
	}
	return false
}

// rpFileRow 是文件行：14px 占位 + 文件图标 + 文件名 + 统计 + 暂存点
// （F-27–F-33），右键出文件菜单（F-34–F-44）。
func (a *App) rpFileRow(c *gui.Context, node *rpNode, p string, depth int) {
	t := c.Theme()
	label := "文件，" + node.name
	if node.change != nil {
		if node.change.statUnknown {
			// 只有合并状态、没有行数与暂存列：读屏文案里也不声称没有的事实。
			label += "，有改动"
		} else {
			label += fmt.Sprintf("，新增 %d 行，删除 %d 行", node.change.added, node.change.removed)
			if node.change.staged {
				label += "，已暂存"
			} else {
				label += "，未暂存"
			}
		}
	}
	row := gui.ButtonBase(c).Height(panelRowHeight2).Padding(0, panelRowPadH).
		Gap(panelRowGap).AlignItems(gui.Center).Radius(radiusSmall).
		Role(gui.RoleTreeItem).Level(depth + 1).
		Label(label).Cursor(gui.CursorPointer)
	switch {
	case row.Pressed():
		row = row.Background(t.SurfacePressed)
	case row.Hovered():
		row = row.Background(t.SurfaceHover)
	}
	row.Children(func() {
		// 与目录箭头同宽的占位，让文件名与目录名同列（F-28）。
		gui.Box(c).Width(panelArrowSlot).Height(panelArrowSlot).Shrink(0)
		rpGlyph(c, rpFileIcon(node.name), panelRowIcon, t.TextMuted, "")
		color := t.Text
		switch {
		case strings.HasPrefix(node.name, "."):
			color = t.TextMuted
		case node.change != nil:
			color = t.Warning
		}
		typeControl.style(t, gui.Text(c, node.name)).TextColor(color).Grow(1).SingleLine()
		if node.change != nil && !node.change.statUnknown {
			rpStatPair(c, node.change.added, node.change.removed)
		}
		if node.change != nil && node.change.staged {
			rpStagedDot(c)
		}
	})
	// 文件行的单击是「打开预览」（F-27）：在「打开文件」页看这个文件。
	if row.Clicked() {
		a.rpOpenPreview(p)
	}
	a.rpFileMenu(c, row, node, p)
}

// rpStatPair 是行尾的 +n/−n：两列定宽、等宽数字、右对齐成列（F-32）。
func rpStatPair(c *gui.Context, added, removed int) {
	t := c.Theme()
	typeCode.style(t, gui.Text(c, fmt.Sprintf("+%d", added))).TextColor(t.Success).
		FontSize(typeMicro.sizeIn(t)).Width(panelStatWidth).TextAlign(gui.End).SingleLine()
	typeCode.style(t, gui.Text(c, fmt.Sprintf("-%d", removed))).TextColor(t.Danger).
		FontSize(typeMicro.sizeIn(t)).Width(panelStatWidth).TextAlign(gui.End).SingleLine()
}

// rpStagedDot 是「已暂存」圆点徽标：蓝底白圈的 12px 圆（F-33）。
func rpStagedDot(c *gui.Context) {
	t := c.Theme()
	box := gui.Box(c).Width(panelBadgeSize).Height(panelBadgeSize).Shrink(0).
		Label("已暂存")
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, colorStagedDot.in(t), panelBadgeSize/2)
		inner := gui.Rect{
			X: r.X + panelBadgeInset, Y: r.Y + panelBadgeInset,
			W: r.W - 2*panelBadgeInset, H: r.H - 2*panelBadgeInset,
		}
		p.Fill(inner, t.Surface, panelBadgeSize/2)
	})
}

// panelBadgeInset 是暂存点里白圈的半径收缩（12px 圆里留 3px 白心）。
const panelBadgeInset = 3

// rpFileIcon 是按扩展名取的文件图标（F-29）：未知扩展名退化为通用文件。
func rpFileIcon(name string) *gui.SVG {
	switch strings.ToLower(path.Ext(name)) {
	case ".go", ".py", ".sh", ".js", ".ts", ".rs":
		return iconFileCode
	case ".md", ".txt", ".docx":
		return iconDocument
	case ".json", ".toml", ".yml", ".yaml":
		return iconFileCode
	case ".html":
		return iconGlobe
	case ".png", ".jpg", ".jpeg", ".gif", ".svg":
		return iconEye
	case ".mp3", ".wav":
		return iconPlay
	}
	return iconDocument
}

// rpSearchResults 是搜索态：状态行 / 错误行 / 部分失败 / 上限提示 / 结果行 /
// 空态（F-04–F-12）。
func (a *App) rpSearchResults(c *gui.Context, query string, refresh bool) {
	if facts, ok := a.rpFacts(); ok {
		a.rpLiveSearch(c, facts, query, refresh)
		return
	}
	t := c.Theme()
	s := a.rp()
	if s.searching {
		typeSecondary.style(t, gui.Text(c, "正在搜索整个工作区…")).
			TextColor(t.TextMuted).Role(gui.RoleStatus).SingleLine()
		return
	}
	if s.searchFailed {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
			typeSecondary.style(t, gui.Text(c, "文件搜索失败。")).TextColor(t.Danger).
				Role(gui.RoleAlert)
			if rpTextAction(c, nil, "重试", "重试搜索", false) {
				// 样例里的搜索是同步匹配：重试当帧出结果，不停在「正在搜索」上
				// ——那会是一个没有返回点的假等待（F-06 / R-RF-046）。
				s.searchFailed = false
				s.searching = false
			}
		})
		return
	}
	if s.searchPartial {
		typeSecondary.style(t, gui.Text(c, "部分文件夹无法搜索，结果可能不完整。")).
			TextColor(t.Warning).Role(gui.RoleStatus).SingleLine()
	}
	matches := rpSearch(s.tree, query)
	if len(matches) > rpSearchLimit {
		matches = matches[:rpSearchLimit]
		typeSecondary.style(t, gui.Text(c, fmt.Sprintf("仅显示前 %d 个匹配项", rpSearchLimit))).
			TextColor(t.TextMuted).SingleLine()
	}
	if len(matches) == 0 {
		a.rpEmptyBlock(c, "无匹配文件",
			fmt.Sprintf("没有找到包含“%s”的文件", query))
		return
	}
	for _, hit := range matches {
		a.rpSearchRow(c, hit, query)
	}
}

// rpLiveSearch 画真实的搜索读数：读态、失败与重试、部分失败、截断、命中与空态。
//
// 命中是内容命中（路径 + 行号），所以先按路径归并再画行：同一文件的多处命中
// 只占一行，行上的名字与目录取自路径。
func (a *App) rpLiveSearch(c *gui.Context, facts PanelFacts, query string, refresh bool) {
	t := c.Theme()
	s := a.rp()
	snapshot := facts.FileSearch(FileSearchRequest{
		Cwd: a.rpCwd(), Query: query, Limit: rpSearchLimit, Refresh: refresh,
	})
	switch snapshot.Phase {
	case PanelReadLoading, PanelReadIdle:
		typeSecondary.style(t, gui.Text(c, "正在搜索整个工作区…")).
			TextColor(t.TextMuted).Role(gui.RoleStatus).SingleLine()
		return
	case PanelReadFailed:
		s.searchError = snapshot.Error
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
			// 通用句 + 原因原文分两段：原因照读数写，不在界面这一层编。
			typeSecondary.style(t, gui.Text(c, "文件搜索失败。")).TextColor(t.Danger).
				Role(gui.RoleAlert).Shrink(0).MaxLines(1)
			if snapshot.Error != "" {
				typeSecondary.style(t, gui.Text(c, snapshot.Error)).TextColor(t.Danger).
					Grow(1).MaxLines(2)
			}
			if rpTextAction(c, nil, "重试", "重试搜索", false) {
				s.reload++
			}
		})
		return
	}
	if snapshot.Denied {
		typeSecondary.style(t, gui.Text(c, "部分文件夹无法搜索，结果可能不完整。")).
			TextColor(t.Warning).Role(gui.RoleStatus).SingleLine()
	}
	if snapshot.Truncated {
		line := fmt.Sprintf("仅显示前 %d 个匹配项", snapshot.Limit)
		// 截断的成因分开写：匹配数到上限与扫描规模到上限不是一回事，后者的
		// 结果可能连"前 N 个"都没覆盖到整个工作区。
		switch snapshot.StopReason {
		case "max-files":
			line += "：已扫到文件数上限，结果可能不完整"
		case "max-scan-bytes":
			line += "：已扫到扫描字节上限，结果可能不完整"
		}
		typeSecondary.style(t, gui.Text(c, line)).TextColor(t.TextMuted).SingleLine()
	}
	hits := rpPathHits(snapshot.Hits)
	if len(hits) == 0 {
		a.rpEmptyBlock(c, "无匹配文件",
			fmt.Sprintf("没有找到包含“%s”的文件", query))
		return
	}
	for _, hit := range hits {
		a.rpSearchRow(c, hit, query)
	}
}

// rpPathHits 把内容命中按路径归并成搜索行（同一文件的多处命中只占一行）。
func rpPathHits(hits []FileHitSnapshot) []rpSearchHit {
	out := make([]rpSearchHit, 0, len(hits))
	seen := make(map[string]bool, len(hits))
	for _, hit := range hits {
		if seen[hit.Path] {
			continue
		}
		seen[hit.Path] = true
		dir := path.Dir(hit.Path)
		if dir == "." {
			dir = ""
		}
		out = append(out, rpSearchHit{node: &rpNode{name: path.Base(hit.Path)}, dir: dir})
		if len(out) >= rpSearchLimit {
			break
		}
	}
	return out
}

// rpSearchLimit 是搜索结果的上限（清单 F-08：200）。
const rpSearchLimit = 200

// rpSearchHit 是一条搜索命中：文件节点与它所在的目录。
type rpSearchHit struct {
	node *rpNode
	dir  string
}

// rpSearch 在样例树上按名字匹配文件（大小写不敏感），返回命中与所在目录。
func rpSearch(nodes []*rpNode, query string) []rpSearchHit {
	q := rpAsciiLower(query)
	var out []rpSearchHit
	var walk func(list []*rpNode, dir string)
	walk = func(list []*rpNode, dir string) {
		for _, node := range list {
			if node.dir {
				next := node.name
				if dir != "" {
					next = dir + "/" + node.name
				}
				walk(node.children, next)
				continue
			}
			if strings.Contains(rpAsciiLower(node.name), q) {
				out = append(out, rpSearchHit{node: node, dir: dir})
			}
		}
	}
	walk(nodes, "")
	return out
}

// rpSearchRow 是一条搜索命中行（F-12）：图标 + 文件名（命中高亮）+ 次级目录
// + 统计，行高 44。
func (a *App) rpSearchRow(c *gui.Context, hit rpSearchHit, query string) {
	t := c.Theme()
	row := gui.ButtonBase(c).Height(panelSearchRowHeight).Padding(space1, panelRowPadH).
		Gap(panelRowGap).AlignItems(gui.Center).Radius(radiusSmall).
		Role(gui.RoleListItem).
		Label("文件，" + hit.node.name + "，" + hit.dir).Cursor(gui.CursorPointer)
	switch {
	case row.Pressed():
		row = row.Background(t.SurfacePressed)
	case row.Hovered():
		row = row.Background(t.SurfaceHover)
	}
	row.Children(func() {
		rpGlyph(c, rpFileIcon(hit.node.name), panelRowIcon, t.TextMuted, "")
		gui.Column(c).Grow(1).Children(func() {
			rpHighlighted(c, hit.node.name, query, t.Text)
			if hit.dir != "" {
				typeMicro.style(t, gui.Text(c, hit.dir)).TextColor(t.TextMuted).SingleLine()
			}
		})
		if hit.node.change != nil && !hit.node.change.statUnknown {
			rpStatPair(c, hit.node.change.added, hit.node.change.removed)
		}
	})
	if row.Clicked() {
		a.rpOpenPreview(rpJoinPath(hit.dir, hit.node.name))
	}
	a.rpFileMenu(c, row, hit.node, rpJoinPath(hit.dir, hit.node.name))
}

// rpJoinPath 拼仓库内相对路径。
func rpJoinPath(dir, name string) string {
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

// rpAsciiLower 只折叠 ASCII 大小写：字节长度不变，命中段的字节下标可以直接
// 用回原文（Unicode 的大小写折叠会改字节长度）。
func rpAsciiLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

// rpHighlighted 画一行带命中高亮的文字：命中段黄底（RP-B-13）。
func rpHighlighted(c *gui.Context, text, query string, color gui.Color) {
	t := c.Theme()
	q := rpAsciiLower(query)
	lower := rpAsciiLower(text)
	at := strings.Index(lower, q)
	if q == "" || at < 0 {
		typeControl.style(t, gui.Text(c, text)).TextColor(color).SingleLine()
		return
	}
	before := text[:at]
	hit := text[at : at+len(q)]
	after := text[at+len(q):]
	spans := make([]gui.Span, 0, 3)
	add := func(s string, bg gui.Color) {
		if s == "" {
			return
		}
		spans = append(spans, gui.Span{
			Text: s, Size: typeControl.sizeIn(t), Weight: typeControl.weight,
			Color: color, Background: bg,
		})
	}
	add(before, gui.Color{})
	add(hit, colorSearchHit.in(t))
	add(after, gui.Color{})
	gui.RichText(c, spans...).SingleLine()
}

// rpEmptyBlock 是空态：48px 圆底图标 + 标题 + 说明（F-09–F-11 / F-15–F-17）。
func (a *App) rpEmptyBlock(c *gui.Context, title, detail string) {
	a.rpEmptyBlockIcon(c, iconDocument, title, detail)
}

// rpEmptyBlockIcon 是带指定图标的空态块（文件页用文件图标，改动页用对勾）。
func (a *App) rpEmptyBlockIcon(c *gui.Context, icon *gui.SVG, title, detail string) {
	t := c.Theme()
	gui.Column(c).Grow(1).Center().Padding(space8, space4).Gap(space2).Children(func() {
		circle := gui.Box(c).Width(panelEmptyIcon).Height(panelEmptyIcon).Shrink(0)
		circle.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Fill(r, t.SurfaceHover, panelEmptyIcon/2)
			inner := gui.Rect{
				X: r.X + panelEmptyGlyphInset, Y: r.Y + panelEmptyGlyphInset,
				W: r.W - 2*panelEmptyGlyphInset, H: r.H - 2*panelEmptyGlyphInset,
			}
			paintIcon(p, icon, inner, t.TextMuted)
		})
		typeControlStrong.style(t, gui.Text(c, title)).TextColor(t.Text).
			TextAlign(gui.Center).SingleLine()
		if detail != "" {
			typeSecondary.style(t, gui.Text(c, detail)).TextColor(t.TextMuted).
				TextAlign(gui.Center).MaxLines(2)
		}
	})
}

// panelEmptyGlyphInset 是空态圆底里图标的内缩（48 圆里放 20 图标）。
const panelEmptyGlyphInset = 14

// rpLoadMoreRow 是目录内的「加载更多 / 正在加载...」（F-24）。
func (a *App) rpLoadMoreRow(c *gui.Context, p string) {
	s := a.rp()
	// F-45 加载更多 / 正在加载...
	label := "加载更多"
	if s.dirLoading[p] {
		label = "正在加载..."
	}
	if rpTextAction(c, iconPlus, label, label+"："+p, s.dirLoading[p]) {
		s.dirLoading[p] = true
		s.dirPages[p] += rpPageSize
		s.dirLoading[p] = false
	}
}

// rpDirErrorRow 是目录内的错误行（F-25/F-26）：告警图标 + 危险色文案 + 行尾
// 「重试」。
//
// 读屏名带上目录：读屏用户听到「无法加载文件夹：src/components」，才知道是哪
// 一格出错（R-RF-215）。根级失败说「无法加载文件树」——根不是某一个文件夹，
// 写「此文件夹」会指向一格不存在的东西。
func (a *App) rpDirErrorRow(c *gui.Context, p string) {
	s := a.rp()
	// 通用句后面接读取方给的原因：只有通用句时用户不知道是权限、断线还是路径
	// 没了；没有原因时这一行与原来一致。
	reason := s.dirError[p]
	who := "无法加载文件树"
	sentence := "无法加载文件树。"
	retryA11y := "重试读取文件树"
	if p != "" {
		who = "无法加载文件夹：" + p
		sentence = "无法加载此文件夹。"
		retryA11y = "重试读取文件夹：" + p
	}
	// 重试 = 清掉失败位并请求这一格的重读（适配器按 Refresh 跳过缓存）：
	// 根级仍走整树刷新，子目录只重读自己。
	panelErrorRow(c, who, sentence, reason, retryA11y, func() {
		s.dirFailed[p] = false
		if p == "" {
			s.reload++
		} else {
			s.dirRefresh[p] = true
		}
	})
}

// rpFileMenu 是文件行的右键菜单（F-34–F-44）：暂存 / 取消暂存、查看差异、
// 预览、用外部工具打开、在文件管理器中显示、另存为…、复制路径。
func (a *App) rpFileMenu(c *gui.Context, target gui.Element, node *rpNode, p string) {
	s := a.rp()
	m := gui.ContextMenuBase(c, target, s.rowMenuPtr("file:"+p))
	m.Popup(func(panel gui.Element) {
		rpMenuPanel(panel, c.Theme(), panelMenuNoteWidth+panelRowIcon+2*space2)
		change := node.change
		// git 写动作与差异读取现在没有通道：连着真实读数时只画禁用项与原因，
		// 不留"点了像生效"的假入口（见 D14 未接动作与 D7 §1「不做」）。
		locked := a.rpGitWriteDisabled()
		note := ""
		if locked {
			note = rpGitWriteNote
		}
		if change != nil && !change.staged {
			if menuItem(c, m, iconPlus, "暂存", "暂存", note, locked, false) && !locked {
				change.staged = true
			}
		}
		if change != nil && change.staged {
			if menuItem(c, m, iconMinus, "取消暂存", "取消暂存", note, locked, false) && !locked {
				change.staged = false
			}
		}
		if change != nil {
			rpMenuSeparator(c)
			if menuItem(c, m, iconFileDiff, "查看差异", "查看差异", note, locked, false) && !locked {
				a.rpOpen(tabChanges)
				s.diffOpen[p] = true
			}
		}
		rpMenuSeparator(c)
		if menuItem(c, m, iconEye, "预览", "预览", "", false, false) {
			a.rpOpenPreview(p)
		}
		// 外部工具、文件管理器与另存为要宿主/协议能力：本版不做，入口禁用并
		// 写原因（DEC-09）；复制路径是本地动作，照常可用。
		menuItem(c, m, iconExternal, "用外部工具打开", "用外部工具打开", notePreviewIO, true, false)
		menuItem(c, m, iconFolderOpen, "在文件管理器中显示", "在文件管理器中显示", notePreviewIO, true, false)
		menuItem(c, m, iconDownload, "另存为…", "另存为…", notePreviewIO, true, false)
		rpMenuSeparator(c)
		if menuItem(c, m, iconCopy, "复制路径", "复制路径", "", false, false) {
			c.WriteClipboard(p)
		}
	})
}

// rpMenuSeparator 是浮层里的一条分隔线（1px 高的面差，不新增自绘描边）。
func rpMenuSeparator(c *gui.Context) {
	hairlineBox(c, true, "分隔线").Margin(space1, 0, space1, 0)
}
