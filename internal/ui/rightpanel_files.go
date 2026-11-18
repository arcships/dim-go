package ui

import (
	"fmt"
	"path"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 文件页：按 inventory-panels-rt.md §1 与 requirements-panels.md §1 重建。
//
// 组成：筛选框（与改动页共用一处，R-RF-002）→ 权限条（F-46）→ 主树 / 搜索
// 结果 / 空态 / 目录错误。行高 28；目录行的箭头 14、文件夹图标 16、名字与
// 文件行的名字同列（R-RF-160、R-RF-240）。
//
// 事实来源：本产品还没有文件树读数，树与改动都用样例（rightpanel_sample.go）；
// 搜索在本机对样例做同步匹配，所以「正在搜索」只在一帧内出现，错误 / 部分
// 失败两态要等真实的搜索读数（见交回时的未实现分支表）。

// rpFilesPage 是文件页本体。
func (a *App) rpFilesPage(c *gui.Context) {
	s := a.rp()
	q := rpQuery(s.filter)
	gui.Column(c).Grow(1).Children(func() {
		a.rpFilterRow(c)
		if s.permissionDenied {
			a.rpPermissionBar(c)
		}
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Padding(space1, rpTreePadH).Children(func() {
				if q != "" {
					a.rpSearchResults(c, q)
					return
				}
				a.rpTreeRows(c, s.tree, "", 0)
			})
		})
	})
}

// rpTreePadH 是树区的左右留白（R 态行 410 对面板 405）。
const rpTreePadH = space1

// rpQuery 是筛选词：去首尾空白；空串表示不在搜索态。
func rpQuery(q string) string { return strings.TrimSpace(q) }

// rpFilterRow 是筛选框：前置放大镜、占位「筛选文件...」、无边框无底色，高 25
// （RP-B-01 / F-01）。
func (a *App) rpFilterRow(c *gui.Context) {
	t := c.Theme()
	s := a.rp()
	gui.Column(c).Shrink(0).Padding(space1, space3).Children(func() {
		gui.Row(c).Height(panelFilterHeight).Gap(space1).AlignItems(gui.Center).
			Children(func() {
				rpGlyph(c, iconSearch, iconInline, t.TextMuted, "search")
				gui.TextInputBase(c, &s.filter).Grow(1).Label("筛选文件").
					Placeholder("筛选文件...").FontSize(typeSecondary.sizeIn(t)).
					TextColor(t.Text)
			})
	})
	// 焦点在框内按 Esc：退出搜索态并清空查询（F-01、R-RF-007）。
	if s.filter != "" && c.FocusTakesText() && c.Shortcut(0, gui.KeyEscape) {
		s.filter = ""
		s.searching = false
	}
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
			a.rpEmptyBlock(c, "无匹配文件", "")
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
			a.rpDirRow(c, node, p)
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
				if loaded < len(node.children) {
					a.rpLoadMoreRow(c, p)
				}
			})
			continue
		}
		a.rpFileRow(c, node, p)
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

// rpDirRow 是目录行：箭头 + 文件夹图标 + 目录名（F-17–F-21）。
func (a *App) rpDirRow(c *gui.Context, node *rpNode, p string) {
	t := c.Theme()
	s := a.rp()
	open := s.dirOpen[p]
	state := "已收起"
	if open {
		state = "已展开"
	}
	row := gui.ButtonBase(c).Height(panelRowHeight2).Padding(0, panelRowPadH).
		Gap(panelRowGap).AlignItems(gui.Center).Radius(radiusSmall).
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
		case rpDirChanged(node):
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
func (a *App) rpFileRow(c *gui.Context, node *rpNode, p string) {
	t := c.Theme()
	label := "文件，" + node.name
	if node.change != nil {
		label += fmt.Sprintf("，新增 %d 行，删除 %d 行", node.change.added, node.change.removed)
		if node.change.staged {
			label += "，已暂存"
		} else {
			label += "，未暂存"
		}
	}
	row := gui.ButtonBase(c).Height(panelRowHeight2).Padding(0, panelRowPadH).
		Gap(panelRowGap).AlignItems(gui.Center).Radius(radiusSmall).
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
		if node.change != nil {
			rpStatPair(c, node.change.added, node.change.removed)
		}
		if node.change != nil && node.change.staged {
			rpStagedDot(c)
		}
	})
	// 文件行的单击是「打开预览」（F-27）；预览面板不在本批，故只保留行本身。
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
func (a *App) rpSearchResults(c *gui.Context, query string) {
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
				s.searchFailed = false
				s.searching = true
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

// rpSearchLimit 是搜索结果的上限（清单 F-08：200）。
const rpSearchLimit = 200

// rpSearchHit 是一条搜索命中：文件节点与它所在的目录。
type rpSearchHit struct {
	node *rpNode
	dir  string
}

// rpSearch 在样例树上按名字匹配文件（大小写不敏感），返回命中与所在目录。
func rpSearch(nodes []*rpNode, query string) []rpSearchHit {
	q := strings.ToLower(query)
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
			if strings.Contains(strings.ToLower(node.name), q) {
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
		added, removed := 0, 0
		if hit.node.change != nil {
			added, removed = hit.node.change.added, hit.node.change.removed
		}
		rpStatPair(c, added, removed)
	})
	a.rpFileMenu(c, row, hit.node, rpJoinPath(hit.dir, hit.node.name))
}

// rpJoinPath 拼仓库内相对路径。
func rpJoinPath(dir, name string) string {
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

// rpHighlighted 画一行带命中高亮的文字：命中段黄底（RP-B-13）。
func rpHighlighted(c *gui.Context, text, query string, color gui.Color) {
	t := c.Theme()
	q := strings.ToLower(query)
	lower := strings.ToLower(text)
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
			p.Icon(icon, inner, t.TextMuted)
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

// rpDirErrorRow 是目录内的错误行（F-25/F-26）：危险色文案 + 行尾「重试」。
func (a *App) rpDirErrorRow(c *gui.Context, p string) {
	t := c.Theme()
	s := a.rp()
	gui.Row(c).Height(panelRowHeight2).Padding(0, panelRowPadH).Gap(space1).
		AlignItems(gui.Center).Role(gui.RoleAlert).Children(func() {
		typeSecondary.style(t, gui.Text(c, "无法加载此文件夹。")).TextColor(t.Danger).
			Grow(1).SingleLine()
		if rpTextAction(c, nil, "重试", "重试读取文件夹："+p, false) {
			s.dirFailed[p] = false
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
		if change != nil && !change.staged {
			if menuItem(c, m, iconPlus, "暂存", "暂存", "", false, false) {
				change.staged = true
			}
		}
		if change != nil && change.staged {
			if menuItem(c, m, iconMinus, "取消暂存", "取消暂存", "", false, false) {
				change.staged = false
			}
		}
		if change != nil {
			rpMenuSeparator(c)
			if menuItem(c, m, iconFileDiff, "查看差异", "查看差异", "", false, false) {
				a.rpOpen(tabChanges)
				s.diffOpen[p] = true
			}
		}
		rpMenuSeparator(c)
		menuItem(c, m, iconEye, "预览", "预览", "", false, false)
		menuItem(c, m, iconExternal, "用外部工具打开", "用外部工具打开", "", false, false)
		menuItem(c, m, iconFolderOpen, "在文件管理器中显示", "在文件管理器中显示", "", false, false)
		menuItem(c, m, iconDownload, "另存为…", "另存为…", "", false, false)
		rpMenuSeparator(c)
		menuItem(c, m, iconCopy, "复制路径", "复制路径", "", false, false)
	})
}

// rpMenuSeparator 是浮层里的一条分隔线（1px 高的面差，不新增自绘描边）。
func rpMenuSeparator(c *gui.Context) {
	t := c.Theme()
	gui.Box(c).Shrink(0).Height(strokeHairline).Margin(space1, 0, space1, 0).
		Background(t.Border).Label("分隔线").AccessHidden(true)
}
