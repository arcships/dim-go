package ui

import (
	"fmt"
	"path"

	gui "github.com/egoist/mygo/ui"
)

// 改动页：按 inventory-panels-rt.md §2 与 requirements-panels.md §2 重建。
//
// 组成：范围栏（未提交改动 / 当前分支改动 / Last Run 改动）→ 筛选框（与文件页
// 共用）→ 已暂存组 / 未暂存组 → 行内 diff → 底部统计；提交或推送菜单与分支
// 子菜单在范围栏右端，权限不足时各组动作禁用。
//
// 事实来源：本产品还没有 git 改动与写动作的读数（Source 见 source.go），所以
// 列表用样例、暂存与还原只改样例；提交 / 推送 / 分支切换在本批不动真实仓库。

// 改动范围的三种取值（D-02–D-04）。
const (
	scopeUncommitted = iota
	scopeBranch
	scopeLastRun
)

// rpChangesPage 是改动页本体。
func (a *App) rpChangesPage(c *gui.Context) {
	gui.Column(c).Grow(1).Children(func() {
		a.rpScopeBar(c)
		a.rpFilterRow(c)
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Padding(space1, rpTreePadH).Children(func() {
				a.rpChangeGroups(c)
			})
		})
		if len(a.rpVisibleChanges()) > 0 {
			a.rpChangesFooter(c)
		}
	})
	a.rpRevertDialog(c)
}

// rpVisibleChanges 是当前范围下要显示的改动：样例只有「未提交」这一路；
// 其余两个范围按参考产品的空态处理。
func (a *App) rpVisibleChanges() []*rpChange {
	s := a.rp()
	if s.scope != scopeUncommitted {
		return nil
	}
	return s.changes
}

// rpScopeBar 是范围栏：左端范围下拉（D-01–D-07），右端刷新（D-15）与
// 提交或推送（D-08–D-14）。
func (a *App) rpScopeBar(c *gui.Context) {
	t := c.Theme()
	s := a.rp()
	gui.Row(c).Height(panelScopeHeight).Shrink(0).Padding(space1, space3).
		AlignItems(gui.Center).Gap(space1).Children(func() {
		m := gui.MenuBase(c, &s.scopeOpen)
		tr := m.Trigger.Height(panelScopeActionHeight).Padding(0, space1).
			Radius(radiusSmall).Gap(space1).AlignItems(gui.Center).
			Label("改动范围：" + rpScopeName(s.scope)).Tooltip("改动范围").
			Cursor(gui.CursorPointer)
		switch {
		case tr.Pressed():
			tr = tr.Background(t.SurfacePressed)
		case tr.Hovered():
			tr = tr.Background(t.SurfaceHover)
		}
		tr.Children(func() {
			typeSecondary.style(t, gui.Text(c, rpScopeName(s.scope))).TextColor(t.Text).
				SingleLine()
			rpGlyph(c, iconChevronDown, panelTabIcon, t.TextMuted, "")
		})
		m.Popup(func(panel gui.Element) {
			rpMenuPanel(panel, t, panelMenuWidth)
			for i, item := range rpScopeItems {
				scope := i
				if menuItem(c, m, nil, item.name, "改动范围："+item.name, item.note,
					false, s.scope == scope) {
					s.scope = scope
				}
			}
		})
		gui.Spacer(c)
		if iconButton(c, iconButtonSpec{icon: iconRetry, label: "刷新", tip: "刷新改动",
			size: panelBarAction}) {
			s.reload++
		}
		if a.rpShowGitAction() {
			a.rpSubmitButton(c)
		}
	})
}

// panelScopeActionHeight 是范围栏里各入口的高（实测 25）。
const panelScopeActionHeight = 25

// rpScopeItem 是范围下拉的一项（两段文案，D-05–D-07）。
type rpScopeItem struct {
	name string
	note string
}

var rpScopeItems = []rpScopeItem{
	{name: "未提交改动", note: "当前工作区改动"},
	{name: "当前分支改动", note: "分支与基线差异"},
	{name: "Last Run 改动", note: "最近一次 agent run 触达的文件"},
}

// rpScopeName 是范围标签的文案（D-02–D-04）。
func rpScopeName(scope int) string {
	if scope >= 0 && scope < len(rpScopeItems) {
		return rpScopeItems[scope].name
	}
	return rpScopeItems[0].name
}

// rpShowGitAction 是「提交或推送」的出现条件（D-08）：有会话 ∧ 范围不是
// Last Run。
func (a *App) rpShowGitAction() bool {
	return a.selectedSession != "" && a.rp().scope != scopeLastRun
}

// rpSubmitButton 是「提交或推送」：图标 + 文案 + 下拉箭头，菜单里先是分支
// 子菜单，再是提交 / 提交并推送 / 推送（D-08–D-14）。
func (a *App) rpSubmitButton(c *gui.Context) {
	t := c.Theme()
	s := a.rp()
	current, branches, _ := a.branchState()
	if current == "" {
		current = "main"
	}
	b := gui.MenuBase(c, &s.gitOpen)
	tr := b.Trigger.Height(panelScopeActionHeight).Padding(0, space2).Radius(radius).
		Gap(space1).AlignItems(gui.Center).Label("提交或推送").Tooltip("提交或推送").
		Cursor(gui.CursorPointer)
	tr.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, radius, strokeHairline)
	})
	switch {
	case tr.Pressed():
		tr = tr.Background(t.SurfacePressed)
	case tr.Hovered():
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() {
		rpGlyph(c, iconCloudUpload, panelRowIcon, t.TextMuted, "")
		typeSecondary.style(t, gui.Text(c, "提交或推送")).TextColor(t.Text).SingleLine()
		rpGlyph(c, iconChevronDown, panelTabIcon, t.TextMuted, "")
	})
	b.Popup(func(panel gui.Element) {
		rpMenuPanel(panel, t, panelMenuWidth)
		sub := b.SubmenuItem().MinHeight(panelRowHeight2).Padding(space1, space2).
			Radius(radius).Gap(space2).AlignItems(gui.Center).Label("切换分支")
		if sub.Highlighted() {
			sub = sub.Background(t.SurfaceHover)
		}
		sub.Children(func() {
			rpGlyph(c, iconBranch, panelRowIcon, t.TextMuted, "")
			typeSecondary.style(t, gui.Text(c, current)).TextColor(t.Text).Grow(1).SingleLine()
			rpGlyph(c, iconChevronR, panelTabIcon, t.TextMuted, "")
		})
		b.Submenu(sub, func(panel gui.Element) {
			rpMenuPanel(panel, t, panelMenuWidth)
			for _, name := range branches {
				name := name
				if menuItem(c, b, iconBranch, name, "分支："+name, "", name == current, name == current) {
					// 分支切换要写仓库，本批不动事实：菜单只收起。
					s.branchOpen = false
				}
			}
		})
		rpMenuSeparator(c)
		noStaged := len(a.rpStaged()) == 0
		menuItem(c, b, iconGitCommit, "提交", "提交", "", noStaged, false)
		menuItem(c, b, iconCloudUpload, "提交并推送", "提交并推送", "", noStaged, false)
		menuItem(c, b, iconCloudUpload, "推送", "推送", "", false, false)
	})
}

// rpStaged / rpUnstaged 按暂存态分两组。
func (a *App) rpStaged() []*rpChange {
	var out []*rpChange
	for _, change := range a.rpVisibleChanges() {
		if change.staged {
			out = append(out, change)
		}
	}
	return out
}

func (a *App) rpUnstaged() []*rpChange {
	var out []*rpChange
	for _, change := range a.rpVisibleChanges() {
		if !change.staged {
			out = append(out, change)
		}
	}
	return out
}

// rpChangeGroups 是空态或两组列表（D-17–D-72）。
func (a *App) rpChangeGroups(c *gui.Context) {
	s := a.rp()
	changes := a.rpVisibleChanges()
	if len(changes) == 0 {
		title, detail := rpEmptyText(s.scope)
		a.rpEmptyBlockIcon(c, iconCircleCheck, title, detail)
		return
	}
	staged, unstaged := a.rpStaged(), a.rpUnstaged()
	if len(staged) > 0 {
		a.rpGroup(c, "staged", fmt.Sprintf("已暂存 (%d)", len(staged)), iconMinus,
			"全部取消暂存", false)
	}
	if len(unstaged) > 0 {
		a.rpGroup(c, "unstaged", fmt.Sprintf("未暂存 (%d)", len(unstaged)), iconCheck,
			"全部暂存", true)
	}
}

// rpEmptyText 是五个空态的标题与说明（D-18–D-27）。
func rpEmptyText(scope int) (string, string) {
	switch scope {
	case scopeBranch:
		return "当前分支没有改动", "当前分支与其基线一致。"
	case scopeLastRun:
		return "没有 Last Run 改动", "最近一次 agent run 没有记录改动文件。"
	}
	return "没有未提交的改动", "工作区是干净的"
}

// rpGroup 画一组：分组标题（箭头 + 计数 + 行尾动作）+ 行 + 行内 diff。
// key 取 "staged" / "unstaged"；stage 为真时行尾动作是「全部暂存」。
func (a *App) rpGroup(c *gui.Context, key, title string, actionIcon *gui.SVG, action string, stage bool) {
	t := c.Theme()
	s := a.rp()
	rows := a.rpUnstaged()
	staged := false
	if !stage {
		rows = a.rpStaged()
		staged = true
	}
	open := s.groupOpen(key)
	gui.Row(c).Height(panelGroupHeight).Padding(0, panelRowPadH).Gap(space1).
		AlignItems(gui.Center).Children(func() {
		label := gui.ButtonBase(c).Grow(1).Height(panelGroupHeight).Radius(radiusSmall).
			Gap(space1).AlignItems(gui.Center).Label(title + "，切换展开").
			Cursor(gui.CursorPointer)
		if label.Hovered() {
			label = label.Background(t.SurfaceHover)
		}
		if label.Clicked() {
			s.setGroupOpen(key, !open)
		}
		label.Children(func() {
			arrow := iconChevronR
			if open {
				arrow = iconChevronDown
			}
			rpGlyph(c, arrow, panelTabIcon, t.TextMuted, "")
			typeSecondaryStrong.style(t, gui.Text(c, title)).TextColor(t.TextMuted).SingleLine()
		})
		if rpTextAction(c, actionIcon, action, action, len(rows) == 0) {
			for _, change := range rows {
				change.staged = stage
			}
		}
	})
	if !open {
		return
	}
	for _, change := range rows {
		a.rpChangeRow(c, change, staged)
	}
}

// rpChangeRow 是改动行：文件图标 + 文件名（title = 完整路径）+ +n/−n +
// 行内动作；单击展开行内 diff（D-33–D-49 / D-55–D-72）。
func (a *App) rpChangeRow(c *gui.Context, change *rpChange, staged bool) {
	t := c.Theme()
	s := a.rp()
	p := change.path
	row := gui.Row(c).Height(panelRowHeight2).Padding(0, panelRowPadH).
		Gap(panelRowGap).AlignItems(gui.Center).Label(
		fmt.Sprintf("文件，%s，新增 %d 行，删除 %d 行", p, change.added, change.removed))
	if row.Hovered() {
		row = row.Background(t.SurfaceHover)
	}
	row.Children(func() {
		rpGlyph(c, rpFileIcon(p), panelRowIcon, t.TextMuted, "")
		name := gui.ButtonBase(c).Grow(1).Height(panelRowHeight2).
			Label(path.Base(p)).Tooltip(p).Cursor(gui.CursorPointer)
		if name.Clicked() {
			s.diffOpen[p] = !s.diffOpen[p]
		}
		name.Children(func() {
			typeControl.style(t, gui.Text(c, path.Base(p))).TextColor(t.Text).SingleLine()
		})
		rpStatPair(c, change.added, change.removed)
		if staged {
			if iconButton(c, iconButtonSpec{icon: iconMinus, label: "取消暂存",
				tip: "取消暂存", size: rowCompact}) {
				change.staged = false
			}
		} else {
			if iconButton(c, iconButtonSpec{icon: iconPlus, label: "暂存", tip: "暂存",
				size: rowCompact}) {
				change.staged = true
			}
			if iconButton(c, iconButtonSpec{icon: iconUndo, label: "还原", tip: "还原",
				size: rowCompact}) {
				s.revertPath = p
			}
		}
	})
	if s.diffOpen[p] {
		a.rpDiffBlock(c, change)
	}
	a.rpChangeMenu(c, row, change, staged)
}

// rpDiffBlock 是行内 unified diff（D-47–D-49）：等宽、按行着色，行数多时
// 只画前若干行并给「已截断」。
func (a *App) rpDiffBlock(c *gui.Context, change *rpChange) {
	t := c.Theme()
	if len(change.diff) == 0 {
		return
	}
	lines := change.diff
	truncated := false
	if len(lines) > rpDiffMaxLines {
		lines = lines[:rpDiffMaxLines]
		truncated = true
	}
	block := gui.Column(c).Shrink(0).Padding(space1, panelDiffPadH).
		Radius(radiusSmall).Background(t.Surface)
	block.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, radiusSmall, strokeHairline)
	})
	block.Children(func() {
		for _, line := range lines {
			color := t.TextMuted
			switch {
			case len(line) > 0 && line[0] == '+':
				color = t.Success
			case len(line) > 0 && line[0] == '-':
				color = t.Danger
			}
			typeCode.style(t, gui.Text(c, line)).TextColor(color).
				FontSize(typeMicro.sizeIn(t)).SingleLine()
		}
		if truncated {
			typeSecondary.style(t, gui.Text(c, "已截断")).TextColor(t.TextMuted).SingleLine()
		}
	})
}

// rpDiffMaxLines 是行内 diff 一次画出的最多行数（其余折叠为「已截断」）。
const rpDiffMaxLines = 60

// rpChangeMenu 是改动行的两套右键菜单（D-39–D-46 / D-62–D-69）。
func (a *App) rpChangeMenu(c *gui.Context, target gui.Element, change *rpChange, staged bool) {
	s := a.rp()
	m := gui.ContextMenuBase(c, target, s.rowMenuPtr("change:"+change.path))
	m.Popup(func(panel gui.Element) {
		rpMenuPanel(panel, c.Theme(), panelMenuNoteWidth+panelRowIcon+2*space2)
		if staged {
			if menuItem(c, m, iconMinus, "取消暂存", "取消暂存", "", false, false) {
				change.staged = false
			}
		} else {
			if menuItem(c, m, iconPlus, "暂存", "暂存", "", false, false) {
				change.staged = true
			}
		}
		rpMenuSeparator(c)
		if menuItem(c, m, iconUndo, "还原", "还原", "", false, false) {
			if staged {
				change.staged = false
			} else {
				s.revertPath = change.path
			}
		}
		rpMenuSeparator(c)
		menuItem(c, m, iconEye, "预览", "预览", "", false, false)
		menuItem(c, m, iconExternal, "用外部工具打开", "用外部工具打开", "", false, false)
		rpMenuSeparator(c)
		menuItem(c, m, iconCopy, "复制路径", "复制路径", "", false, false)
	})
}

// rpChangesFooter 是底部统计条（D-77–D-79）：{count} 个文件改动 + 合计。
func (a *App) rpChangesFooter(c *gui.Context) {
	t := c.Theme()
	changes := a.rpVisibleChanges()
	added, removed := 0, 0
	for _, change := range changes {
		added += change.added
		removed += change.removed
	}
	rpRule(c)
	gui.Row(c).Height(panelFooterHeight).Shrink(0).Padding(0, space3).
		AlignItems(gui.Center).Gap(space2).Children(func() {
		typeSecondary.style(t, gui.Text(c, fmt.Sprintf("%d 个文件改动", len(changes)))).
			TextColor(t.TextMuted).Grow(1).SingleLine()
		typeCode.style(t, gui.Text(c, fmt.Sprintf("+%d", added))).TextColor(t.Success).
			FontSize(typeSecondary.sizeIn(t)).SingleLine()
		typeCode.style(t, gui.Text(c, fmt.Sprintf("-%d", removed))).TextColor(t.Danger).
			FontSize(typeSecondary.sizeIn(t)).SingleLine()
	})
}

// rpRevertDialog 是「确认还原未暂存改动？」（D-73–D-76）：危险色的主按钮。
func (a *App) rpRevertDialog(c *gui.Context) {
	s := a.rp()
	if s.revertPath == "" {
		return
	}
	open := true
	m := gui.Modal(c, &open, func() {
		t := c.Theme()
		panel := gui.Column(c).Gap(space2).Padding(space3).Width(panelDialogWidth).
			Background(t.Surface).Radius(radiusCard)
		panel.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Stroke(r, t.Border, radiusCard, strokeHairline)
		})
		panel.Children(func() {
			typeH3.style(t, gui.Text(c, "确认还原未暂存改动？")).TextColor(t.Text).
				Role(gui.RoleHeading).Level(3).SingleLine()
			typeSecondary.style(t, gui.Text(c,
				fmt.Sprintf("这会永久丢弃“%s”中的未暂存改动。", s.revertPath))).
				TextColor(t.TextMuted).MaxLines(3)
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				if rpOutlineButton(c, "取消", "取消还原") {
					s.revertPath = ""
					open = false
				}
				if rpDangerButton(c, "还原", "确认还原") {
					a.rpDropChange(s.revertPath)
					s.revertPath = ""
					open = false
				}
			})
		})
	})
	if !open || m.Dismissed() {
		s.revertPath = ""
	}
}

// panelDialogWidth 是还原确认框的宽（参考产品对话框的常用宽）。
const panelDialogWidth = 360

// rpDropChange 丢掉一条未暂存改动（样例里直接从改动表与树上摘掉）。
func (a *App) rpDropChange(p string) {
	s := a.rp()
	kept := s.changes[:0]
	for _, change := range s.changes {
		if change.path != p {
			kept = append(kept, change)
		}
	}
	s.changes = kept
	delete(s.diffOpen, p)
	rpClearChange(s.tree, p)
}

// rpClearChange 把树里对应节点的改动清掉（还原后行不再显示统计）。
func rpClearChange(nodes []*rpNode, p string) {
	for _, node := range nodes {
		if node.dir {
			rpClearChange(node.children, p)
			continue
		}
		if node.change != nil && node.change.path == p {
			node.change = nil
		}
	}
}

// groupOpen / setGroupOpen 是分组标题的展开态：默认展开，键为 "staged" /
// "unstaged"。
func (s *rightPanelState) groupOpen(key string) bool {
	if s.groupsClosed == nil {
		return true
	}
	return !s.groupsClosed[key]
}

func (s *rightPanelState) setGroupOpen(key string, open bool) {
	if s.groupsClosed == nil {
		s.groupsClosed = map[string]bool{}
	}
	if open {
		delete(s.groupsClosed, key)
		return
	}
	s.groupsClosed[key] = true
}
