package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 命令面板（S-CP-01–12）：一个覆盖在整窗之上的浮层，不改变下方各区的布局。
//
// 四组候选：动作 / 会话 / 文件 / 设置项。动作走既有的 Intents（没有对应
// Intent 或当前条件下不可用的动作置灰并写原因），会话走左栏同一份列表，文件
// 走右栏已有的文件树，设置项走设置导航的静态表。视图状态全部可丢：查询词、
// 去抖后的过滤词、当前组与选中行、最近使用与搜索历史。
const (
	// 面板尺寸与位置（S-CP-01）：560×480，水平居中，顶部距视口 12%。
	commandPaletteWidth  = 560
	commandPaletteHeight = 480
	commandPaletteTop    = 0.12

	// 去抖与关闭时机（S-CP-02、S-CP-04、S-CP-09）。
	commandPaletteDebounce   = 120 * time.Millisecond
	commandPaletteCloseDelay = 300 * time.Millisecond
	// commandPaletteFocusGrace 是打开后等待输入框拿到焦点的时间：这段时间内
	// 不把"还没有焦点"当成失焦，避免刚打开就排上关闭。
	commandPaletteFocusGrace = 120 * time.Millisecond

	// 每个分组最多显示 8 条，整面板最多 20 条（S-CP-03）。
	commandPaletteGroupLimit = 8
	commandPaletteTotalLimit = 20

	// 文件扫描上限（S-CP-05）。
	commandPaletteScanLimit = 5000

	// 搜索历史与最近使用条数（S-CP-11、S-CP-12）。
	commandPaletteHistoryLimit = 10
	commandPaletteRecentLimit  = 5

	// 面板内的固定行高与输入行高。
	commandPaletteRowHeight   = rowRegular
	commandPaletteInputHeight = 36
	commandPaletteHeaderH     = rowCompact
	commandPaletteFooterH     = rowCompact
)

// 命令面板的分组，顺序即 Tab 的顺序。
const (
	commandGroupActions = iota
	commandGroupSessions
	commandGroupFiles
	commandGroupSettings
	commandGroupCount
)

var commandGroupLabels = [commandGroupCount]string{"动作", "会话", "文件", "设置项"}

var commandGroupIcons = [commandGroupCount]*gui.SVG{
	commandGroupActions:  iconCommand,
	commandGroupSessions: iconChat,
	commandGroupFiles:    iconDocument,
	commandGroupSettings: iconGear,
}

// commandPaletteState 是面板的全部视图状态（可丢，不承载事实）。
type commandPaletteState struct {
	open bool

	// query 绑定输入框；filter 是去抖后真正用于过滤的词。两者在输入后
	// 会短暂不同，deferred 期间显示的是上一份结果（S-CP-04）。
	query    string
	filter   string
	lastSeen string
	deferred bool
	dueAt    time.Time

	activeGroup int
	selected    int

	// groups 是本帧算出的候选；OnTextKey 的回调按组与行号执行它。
	groups [commandGroupCount]commandGroup

	// 打开、去抖与关闭的时间点都取帧时钟（c.Now），测试可用 Advance 走。
	now        time.Time
	openedAt   time.Time
	needsFocus bool
	hadFocus   bool
	blurAt     time.Time
	closeAt    time.Time
	feedback   string

	// history 是执行时记下的查询词（最多 10 条）；recent 是执行过的条目
	// 标识（最多 5 条），排在各组之首（S-CP-11、S-CP-12）。
	history []string
	recent  []string
}

// commandGroup 是一组候选：组标识、标题与行。
type commandGroup struct {
	label string
	items []commandItem
}

// commandItem 是一条候选。run / secondary 为当前条件下可执行的动作；
// disabled 时 reason 说明为什么不可用（S-CP-10）。
type commandItem struct {
	id       string
	title    string
	detail   string
	icon     *gui.SVG
	disabled bool
	reason   string

	run func()

	secondary         func()
	secondaryLabel    string
	secondaryDisabled bool
	secondaryReason   string

	// titleHit / titleHitEnd 是标题里命中的字节区间，用于高亮（-1 表示
	// 命中在描述或拼音首字母上，标题本身不画高亮）。
	titleHit    int
	titleHitEnd int
}

// openCommandPalette 打开面板并清空本次查询；历史与最近使用跨次保留。
func (a *App) openCommandPalette() {
	a.palette.open = true
	a.palette.query = ""
	a.palette.filter = ""
	a.palette.lastSeen = ""
	a.palette.deferred = false
	a.palette.dueAt = time.Time{}
	a.palette.activeGroup = commandGroupActions
	a.palette.selected = 0
	a.palette.feedback = ""
	a.palette.closeAt = time.Time{}
	a.palette.blurAt = time.Time{}
	a.palette.hadFocus = false
	a.palette.needsFocus = true
	a.palette.openedAt = a.palette.now
}

// closeCommandPalette 收起面板；下次打开重新开始一次查询。
func (a *App) closeCommandPalette() {
	a.palette.open = false
	a.palette.deferred = false
	a.palette.dueAt = time.Time{}
	a.palette.closeAt = time.Time{}
	a.palette.blurAt = time.Time{}
	a.palette.feedback = ""
	a.palette.needsFocus = false
}

// commandPaletteKeys 是面板关上时的全局键：Cmd+K 打开（S-CP-02）。打开后
// 的键由输入框的 OnTextKey 接管，这里不再重复处理，避免一次按键开关两次。
func (a *App) commandPaletteKeys(c *gui.Context) {
	if a.palette.open {
		return
	}
	// 终端的 Cmd+K 是清屏（既有键位）；在终端页让给它，不在其上再叠面板。
	if a.rightOpen && a.rightTab == 4 {
		return
	}
	if c.Shortcut(gui.Cmd, gui.KeyK) {
		a.openCommandPalette()
	}
}

// commandPalette 每帧画一次覆盖层：先走时间去抖与关闭，再画面板。
func (a *App) commandPalette(c *gui.Context) {
	a.palette.now = c.Now()
	if !a.palette.open {
		return
	}
	if a.palette.openedAt.IsZero() {
		a.palette.openedAt = a.palette.now
	}

	// 输入变化 → 120ms 后换结果（S-CP-04）。期间保持上一份结果，并请求帧。
	if a.palette.query != a.palette.lastSeen {
		a.palette.lastSeen = a.palette.query
		a.palette.deferred = true
		a.palette.dueAt = a.palette.now.Add(commandPaletteDebounce)
	}
	if a.palette.deferred {
		if a.palette.now.Before(a.palette.dueAt) {
			c.AnimationFrame()
		} else {
			a.palette.filter = a.palette.query
			a.palette.deferred = false
			a.palette.selected = 0
		}
	}

	// 执行后 300ms 内关闭（S-CP-09）。
	if !a.palette.closeAt.IsZero() {
		if !a.palette.now.Before(a.palette.closeAt) {
			a.closeCommandPalette()
			return
		}
		c.AnimationFrame()
	}

	a.palette.groups = a.commandPaletteGroups()
	a.normalizePaletteSelection()

	gui.Overlay(c, func() {
		a.commandPaletteLayer(c)
	})
}

// commandPaletteGroups 算四组候选：过滤 → 最近使用排组首 → 每组 ≤8 →
// 整面板 ≤20（各组轮转取，保证靠后的「设置项」也能露出来）。
func (a *App) commandPaletteGroups() [commandGroupCount]commandGroup {
	var groups [commandGroupCount]commandGroup
	groups[commandGroupActions] = commandGroup{label: commandGroupLabels[commandGroupActions], items: a.commandPaletteActions()}
	groups[commandGroupSessions] = commandGroup{label: commandGroupLabels[commandGroupSessions], items: a.commandPaletteSessions()}
	groups[commandGroupFiles] = commandGroup{label: commandGroupLabels[commandGroupFiles], items: a.commandPaletteFiles()}
	groups[commandGroupSettings] = commandGroup{label: commandGroupLabels[commandGroupSettings], items: a.commandPaletteSettings()}

	q := strings.TrimSpace(a.palette.filter)
	for i := range groups {
		items := groups[i].items
		if q != "" {
			kept := items[:0]
			for _, item := range items {
				ok, at, end := commandPaletteMatch(item.title, item.detail, q)
				if !ok {
					continue
				}
				item.titleHit, item.titleHitEnd = at, end
				kept = append(kept, item)
			}
			items = kept
		}
		items = a.commandPaletteRecentFirst(items)
		if len(items) > commandPaletteGroupLimit {
			items = items[:commandPaletteGroupLimit]
		}
		groups[i].items = items
	}

	// 全局上限：轮转取，最多 20 条（S-CP-03）。
	var kept [commandGroupCount][]commandItem
	total := 0
	for total < commandPaletteTotalLimit {
		added := false
		for i := 0; i < commandGroupCount && total < commandPaletteTotalLimit; i++ {
			if len(kept[i]) >= len(groups[i].items) {
				continue
			}
			kept[i] = append(kept[i], groups[i].items[len(kept[i])])
			total++
			added = true
		}
		if !added {
			break
		}
	}
	for i := 0; i < commandGroupCount; i++ {
		groups[i].items = kept[i]
	}
	return groups
}

// commandPaletteRecentFirst 把最近用过的条目按最近顺序排到组首，其余保持
// 原顺序（S-CP-12）；不在最近 5 条里的条目不重排。
func (a *App) commandPaletteRecentFirst(items []commandItem) []commandItem {
	if len(a.palette.recent) == 0 || len(items) < 2 {
		return items
	}
	rank := make(map[string]int, len(a.palette.recent))
	for i, id := range a.palette.recent {
		rank[id] = i
	}
	sort.SliceStable(items, func(i, j int) bool {
		ri, iok := rank[items[i].id]
		rj, jok := rank[items[j].id]
		switch {
		case iok && jok:
			return ri < rj
		case iok:
			return true
		case jok:
			return false
		}
		return false
	})
	return items
}

// normalizePaletteSelection 让当前组与选中行落在有内容的位置。
func (a *App) normalizePaletteSelection() {
	if a.palette.activeGroup < 0 || a.palette.activeGroup >= commandGroupCount ||
		len(a.palette.groups[a.palette.activeGroup].items) == 0 {
		a.palette.activeGroup = -1
		for i := 0; i < commandGroupCount; i++ {
			if len(a.palette.groups[i].items) > 0 {
				a.palette.activeGroup = i
				break
			}
		}
		a.palette.selected = 0
	}
	if a.palette.activeGroup < 0 {
		return
	}
	if n := len(a.palette.groups[a.palette.activeGroup].items); a.palette.selected >= n {
		a.palette.selected = max(0, n-1)
	}
}

// commandPaletteLayer 画背景变暗层与面板本体。
func (a *App) commandPaletteLayer(c *gui.Context) {
	t := c.Theme()
	windowWidth, windowHeight := c.Size()
	x := (windowWidth - commandPaletteWidth) / 2
	if x < space4 {
		x = space4
	}
	y := windowHeight * commandPaletteTop

	gui.Box(c).Absolute().Fill().Background(gui.RGBA(0, 0, 0, 0.28)).
		AccessHidden(true).OnClick(a.closeCommandPalette)

	panel := gui.Column(c).Absolute().Left(x).Top(y).
		Width(commandPaletteWidth).Height(commandPaletteHeight).
		Background(t.Surface).Border(strokeHairline, t.Border).
		Radius(radiusWindowCard).Clip().
		Shadow(0, 10, 30, 0, colorShadow.in(t)).
		Label("命令面板").Role(gui.RoleDialog)

	panel.Children(func() {
		a.commandPaletteInput(c)
		a.commandPaletteHistory(c)
		a.commandPaletteResults(c)
		a.commandPaletteFooter(c)
	})

	// 失焦 300ms 关（S-CP-02）。打开后的短暂宽限期内不判失焦。
	if c.WindowActive() && panel.FocusWithin() {
		a.palette.hadFocus = true
		a.palette.blurAt = time.Time{}
	} else if a.palette.hadFocus && a.palette.closeAt.IsZero() {
		if a.palette.blurAt.IsZero() {
			a.palette.blurAt = a.palette.now.Add(commandPaletteCloseDelay)
		}
		if !a.palette.now.Before(a.palette.blurAt) {
			a.closeCommandPalette()
			return
		}
		c.AnimationFrame()
	} else if !a.palette.hadFocus && a.palette.now.Sub(a.palette.openedAt) > commandPaletteFocusGrace {
		// 输入框始终没拿到焦点（例如窗口不在前台）：不无限挂着。
		if a.palette.blurAt.IsZero() {
			a.palette.blurAt = a.palette.now.Add(commandPaletteCloseDelay)
		}
		if !a.palette.now.Before(a.palette.blurAt) {
			a.closeCommandPalette()
			return
		}
		c.AnimationFrame()
	}

	// 焦点不在输入框时（例如点了别的区域），覆盖层仍接住 Esc 与 Cmd+K。
	if panel.OverlayShortcut(0, gui.KeyEscape) || panel.OverlayShortcut(gui.Cmd, gui.KeyK) {
		a.closeCommandPalette()
	}
}

// commandPaletteInput 是搜索输入行：放大镜 + 无边框输入框。
func (a *App) commandPaletteInput(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Height(commandPaletteInputHeight).
		Padding(space1, space3).Gap(space2).AlignItems(gui.Center).Children(func() {
		iconGlyph(c, iconSearch, t.TextMuted, "search")
		input := gui.TextInputBase(c, &a.palette.query).
			Placeholder("搜索命令、会话、文件或设置...").
			Label("搜索命令").Grow(1).
			FontSize(typeBody.sizeIn(t)).TextColor(t.Text).
			OnTextKey(a.commandPaletteTextKey)
		if a.palette.needsFocus {
			input.AutoFocus()
			a.palette.needsFocus = false
		}
	})
	gui.Box(c).Height(strokeHairline).Shrink(0).Background(t.Border)
}

// commandPaletteTextKey 接管输入框里的键（S-CP-06）：上下选择、回车执行、
// Cmd+回车次级动作、Tab / Shift+Tab 换组、Esc 与 Cmd+K 关闭。
func (a *App) commandPaletteTextKey(k gui.TextKey) bool {
	switch {
	case k.Mods == gui.Cmd && k.Key == gui.KeyK:
		a.closeCommandPalette()
		return true
	case k.Mods == 0 && k.Key == gui.KeyEscape:
		a.closeCommandPalette()
		return true
	case k.Mods == 0 && k.Key == gui.KeyTab:
		a.movePaletteGroup(1)
		return true
	case k.Mods == gui.Shift && k.Key == gui.KeyTab:
		a.movePaletteGroup(-1)
		return true
	case k.Mods == 0 && k.Key == gui.KeyUp:
		a.movePaletteSelection(-1)
		return true
	case k.Mods == 0 && k.Key == gui.KeyDown:
		a.movePaletteSelection(1)
		return true
	case k.Mods == gui.Cmd && k.Key == gui.KeyEnter:
		a.executePaletteSelection(true)
		return true
	case k.Mods == 0 && k.Key == gui.KeyEnter:
		a.executePaletteSelection(false)
		return true
	}
	return false
}

// movePaletteGroup 切到下一个 / 上一个有内容的组（S-CP-06）。
func (a *App) movePaletteGroup(delta int) {
	start := a.palette.activeGroup
	if start < 0 {
		start = 0
	}
	for step := 1; step <= commandGroupCount; step++ {
		i := ((start+delta*step)%commandGroupCount + commandGroupCount) % commandGroupCount
		if len(a.palette.groups[i].items) > 0 {
			a.palette.activeGroup = i
			a.palette.selected = 0
			return
		}
	}
}

// movePaletteSelection 在当前组内上下移动，越过首尾时回卷。
func (a *App) movePaletteSelection(delta int) {
	group := a.palette.groups[a.palette.activeGroup]
	if len(group.items) == 0 {
		return
	}
	n := len(group.items)
	a.palette.selected = (a.palette.selected + delta + n) % n
}

// executePaletteSelection 执行当前行；secondary 为 Cmd+回车（S-CP-06）。
// 不可用的动作不执行，原因写回面板底部（S-CP-10）；执行成功即排上 300ms
// 后的关闭（S-CP-09）。
func (a *App) executePaletteSelection(secondary bool) {
	if a.palette.activeGroup < 0 {
		return
	}
	group := a.palette.groups[a.palette.activeGroup]
	if a.palette.selected < 0 || a.palette.selected >= len(group.items) {
		return
	}
	item := group.items[a.palette.selected]
	if secondary {
		switch {
		case item.secondary == nil:
			a.palette.feedback = "这一项没有次级动作"
			return
		case item.secondaryDisabled:
			a.palette.feedback = item.secondaryReason
			return
		}
		a.palette.feedback = "已执行：" + item.title
		item.secondary()
	} else {
		if item.disabled {
			a.palette.feedback = item.reason
			return
		}
		a.palette.feedback = "已执行：" + item.title
		item.run()
	}
	a.recordPaletteExecution(item)
	a.palette.closeAt = a.palette.now.Add(commandPaletteCloseDelay)
}

// recordPaletteExecution 记下最近使用（≤5）与搜索历史（≤10）。
func (a *App) recordPaletteExecution(item commandItem) {
	a.palette.recent = commandPalettePush(a.palette.recent, item.id, commandPaletteRecentLimit)
	if q := strings.TrimSpace(a.palette.query); q != "" {
		a.palette.history = commandPalettePush(a.palette.history, q, commandPaletteHistoryLimit)
	}
}

// commandPalettePush 把 v 放到列表首位并去重，超过 limit 的截掉。
func commandPalettePush(list []string, v string, limit int) []string {
	if v == "" {
		return list
	}
	out := make([]string, 0, len(list)+1)
	out = append(out, v)
	for _, old := range list {
		if old != v {
			out = append(out, old)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// commandPaletteHistory 是查询为空时的一行搜索历史（存 10 条，显示前几条，
// 单行截断）。
func (a *App) commandPaletteHistory(c *gui.Context) {
	if strings.TrimSpace(a.palette.query) != "" || len(a.palette.history) == 0 {
		return
	}
	t := c.Theme()
	typeMicro.style(t, gui.Text(c, "最近搜索："+strings.Join(a.palette.history, " · "))).
		TextColor(t.TextMuted).Padding(0, space3).SingleLine()
}

// commandPaletteResults 是四组的滚动列表；没有匹配项时给空态（S-CP-08）。
func (a *App) commandPaletteResults(c *gui.Context) {
	total := 0
	for i := 0; i < commandGroupCount; i++ {
		total += len(a.palette.groups[i].items)
	}
	if total == 0 {
		t := c.Theme()
		gui.Row(c).Grow(1).AlignItems(gui.Center).Justify(gui.Center).Children(func() {
			typeSecondary.style(t, gui.Text(c, "没有匹配项")).TextColor(t.TextMuted)
		})
		return
	}

	gui.Scroll(c).Grow(1).Children(func() {
		gui.Column(c).Padding(space1, space2).Gap(space1).Children(func() {
			for gi := 0; gi < commandGroupCount; gi++ {
				group := a.palette.groups[gi]
				if len(group.items) == 0 {
					continue
				}
				a.commandPaletteGroupHeader(c, gi, group)
				for ii := range group.items {
					a.commandPaletteRow(c, gi, ii, group.items[ii])
				}
			}
		})
	})
}

// commandPaletteGroupHeader 是组标题：图标 + 名称 + 条数。
func (a *App) commandPaletteGroupHeader(c *gui.Context, gi int, group commandGroup) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Height(commandPaletteHeaderH).Padding(0, space1).
		Gap(space1).AlignItems(gui.Center).Label("分组：" + group.label).Children(func() {
		iconGlyph(c, commandGroupIcons[gi], t.TextMuted, "group")
		typeMicro.style(t, gui.Text(c, group.label)).TextColor(t.TextMuted)
		typeMicro.style(t, gui.Text(c, fmt.Sprintf("%d", len(group.items)))).
			TextColor(t.TextMuted)
	})
}

// commandPaletteRow 是一行候选：图标 + 标题（命中高亮）+ 说明 / 不可用原因。
// 选中行用悬停面；不可用行整行 0.5 并写原因（S-CP-05、S-CP-10）。
func (a *App) commandPaletteRow(c *gui.Context, gi, ii int, item commandItem) {
	t := c.Theme()
	selected := gi == a.palette.activeGroup && ii == a.palette.selected
	row := gui.Row(c).Shrink(0).Height(commandPaletteRowHeight).Padding(0, space2).
		Gap(space2).Radius(radius).AlignItems(gui.Center).
		Label(item.title + "：" + item.detail).
		Role(gui.RoleOption).AccessSelected(selected)
	switch {
	case selected:
		row = row.Background(t.SurfaceHover)
	case row.Hovered():
		row = row.Background(t.SurfaceHover)
	}
	if item.disabled {
		row = row.Opacity(0.5)
	}
	row.Children(func() {
		iconGlyph(c, item.icon, t.TextMuted, "item")
		gui.Column(c).Grow(1).Children(func() {
			if item.titleHit >= 0 && item.titleHitEnd > item.titleHit {
				commandPaletteHighlighted(c, item.title, item.titleHit, item.titleHitEnd, t)
			} else {
				typeSecondary.style(t, gui.Text(c, item.title)).TextColor(t.Text).SingleLine()
			}
			note := item.detail
			if item.disabled && item.reason != "" {
				note = "不可用：" + item.reason
			} else if item.secondaryLabel != "" {
				note = strings.TrimSpace(note + " · Cmd+回车：" + item.secondaryLabel)
			}
			if note != "" {
				typeMicro.style(t, gui.Text(c, note)).TextColor(t.TextMuted).SingleLine()
			}
		})
	})
	if item.disabled {
		return
	}
	row.OnClick(func() { a.executePaletteRow(gi, ii, false) })
}

// commandPaletteHighlighted 画带命中高亮的标题（与右栏搜索同一种底色）。
func commandPaletteHighlighted(c *gui.Context, text string, at, end int, t *gui.Theme) {
	spans := make([]gui.Span, 0, 3)
	add := func(s string, bg gui.Color) {
		if s == "" {
			return
		}
		spans = append(spans, gui.Span{
			Text: s, Size: typeSecondary.sizeIn(t), Weight: typeSecondary.weight,
			Color: t.Text, Background: bg,
		})
	}
	add(text[:at], gui.Color{})
	add(text[at:end], colorSearchHit.in(t))
	add(text[end:], gui.Color{})
	gui.RichText(c, spans...).SingleLine()
}

// commandPaletteFooter 是底部提示行：固定键位、当前行的次级动作与执行反馈。
func (a *App) commandPaletteFooter(c *gui.Context) {
	t := c.Theme()
	gui.Box(c).Height(strokeHairline).Shrink(0).Background(t.Border)
	gui.Row(c).Shrink(0).Height(commandPaletteFooterH).Padding(space1, space3).
		Gap(space2).AlignItems(gui.Center).Children(func() {
		hint := "上下选择 · 回车执行 · Tab 换组 · Cmd+回车次级 · Esc 关闭"
		if a.palette.feedback != "" {
			hint = a.palette.feedback
		}
		typeMicro.style(t, gui.Text(c, hint)).TextColor(t.TextMuted).Grow(1).SingleLine()
	})
}

// executePaletteRow 是鼠标点一行：与回车同一条执行路径。
func (a *App) executePaletteRow(gi, ii int, secondary bool) {
	a.palette.activeGroup = gi
	a.palette.selected = ii
	a.executePaletteSelection(secondary)
}

// ---- 四组候选 ----

// commandPaletteActions 是动作组：全部走已有 Intents；不可用的置灰并写原因。
func (a *App) commandPaletteActions() []commandItem {
	session, hasSession := a.liveSession()
	branch, _, _ := a.branchState()
	goal := a.goal
	if hasSession {
		goal = session.Goal
	}
	draft := strings.TrimSpace(a.draft)
	running := a.liveRunning()
	connected := !a.disconnected()

	noIntents := "未连接 App Server，无法执行"
	return []commandItem{
		{
			id: "action:stop", title: "停止这次运行", detail: "中止当前回复",
			icon:     iconStop,
			disabled: !running || a.intents == nil,
			reason:   commandPaletteUnavailable(!running, "当前没有正在运行的回复", noIntents),
			run: func() {
				a.dispatch("停止", a.intents.Stop, nil)
			},
		},
		{
			id: "action:send", title: "发送当前草稿", detail: "提交输入区里的内容",
			icon:     iconArrowUp,
			disabled: draft == "" || a.intents == nil,
			reason:   commandPaletteUnavailable(draft == "", "输入区没有内容", noIntents),
			run:      a.submitDraft,
			secondary: func() {
				text := a.draft
				a.dispatch("排队", func(ctx context.Context) error {
					return a.intents.Queue(ctx, text)
				}, func() { a.draft = "" })
			},
			secondaryLabel:    "排队发送",
			secondaryDisabled: a.intents == nil || draft == "",
			secondaryReason:   commandPaletteUnavailable(a.intents == nil, "未连接 App Server，无法排队", "输入区没有内容"),
		},
		{
			id: "action:new-session", title: "新建会话", detail: "在工作区里开一个会话",
			icon:     iconPlus,
			disabled: true,
			reason:   "工作区路径尚未接入界面（CreateSession 需要 cwd）",
		},
		{
			id: "action:end-branch", title: "结束当前分支", detail: "结束正在查看的分支",
			icon:     iconBranch,
			disabled: a.intents == nil || branch == "" || branch == "main",
			reason:   commandPaletteUnavailable(a.intents == nil, noIntents, "当前在主干分支，没有可结束的分支"),
			run: func() {
				a.dispatch("结束分支", func(ctx context.Context) error {
					return a.intents.EndBranch(ctx, branch)
				}, nil)
			},
		},
		{
			id: "action:clear-goal", title: "清除目标", detail: "去掉当前目标",
			icon:     iconTarget,
			disabled: a.intents == nil || strings.TrimSpace(goal) == "",
			reason:   commandPaletteUnavailable(a.intents == nil, noIntents, "当前没有目标"),
			run: func() {
				a.dispatch("清除目标", a.intents.ClearGoal, nil)
			},
		},
		{
			id: "action:archive", title: "归档当前会话", detail: "从列表移入已归档",
			icon:     iconArchive,
			disabled: a.intents == nil || !hasSession,
			reason:   commandPaletteUnavailable(a.intents == nil, noIntents, "还没有打开会话"),
			run: func() {
				id := session.ID
				a.dispatch("归档会话", func(ctx context.Context) error {
					return a.intents.ArchiveSession(ctx, id, true)
				}, nil)
			},
		},
		{
			id: "action:export", title: "导出当前会话", detail: "写到本地文件",
			icon:     iconDocument,
			disabled: a.intents == nil || !hasSession,
			reason:   commandPaletteUnavailable(a.intents == nil, noIntents, "还没有打开会话"),
			run: func() {
				id := session.ID
				a.dispatch("导出会话", func(ctx context.Context) error {
					path, err := a.intents.ExportSession(ctx, id)
					if err == nil && path != "" {
						a.palette.feedback = "已导出到 " + path
					}
					return err
				}, nil)
			},
		},
		{
			id: "action:reconnect", title: "重新连接", detail: "重连 App Server 控制连接",
			icon:     iconPlug,
			disabled: a.intents == nil || connected,
			reason:   commandPaletteUnavailable(a.intents == nil, noIntents, "当前已连接"),
			run: func() {
				a.dispatch("重新连接", a.intents.Reconnect, nil)
			},
		},
	}
}

// commandPaletteUnavailable 在几个原因里取第一个命中的；都没命中时给通用原因。
func commandPaletteUnavailable(first bool, firstReason, secondReason string) string {
	if first {
		return firstReason
	}
	if secondReason != "" {
		return secondReason
	}
	return "当前条件下不可用"
}

// commandPaletteSessions 是会话组：列表来自 Source（或样例），选中即切换。
func (a *App) commandPaletteSessions() []commandItem {
	names := a.sessionNames()
	items := make([]commandItem, 0, len(names))
	for i, name := range names {
		index := i
		title := name
		secondaryReason := "未连接 App Server，无法归档"
		secondaryDisabled := a.intents == nil
		var secondary func()
		if a.intents != nil {
			secondary = func() {
				id := name
				if snap, ok := a.liveSession(); ok && snap.ID != "" {
					id = snap.ID
				}
				a.dispatch("归档会话", func(ctx context.Context) error {
					return a.intents.ArchiveSession(ctx, id, true)
				}, nil)
			}
		}
		items = append(items, commandItem{
			id:     "session:" + name,
			title:  title,
			detail: "会话",
			icon:   iconChat,
			run: func() {
				a.selectedSession = "s" + fmt.Sprintf("%d", index+1)
				a.screen = ScreenMain
			},
			secondary:         secondary,
			secondaryLabel:    "归档此会话",
			secondaryDisabled: secondaryDisabled,
			secondaryReason:   secondaryReason,
		})
	}
	return items
}

// commandPaletteFiles 是文件组：扁平扫描右栏已有的文件树，最多 5000 个节点
// （S-CP-05）。回车在文件页里筛出该文件；有改动的文件可用 Cmd+回车看改动。
func (a *App) commandPaletteFiles() []commandItem {
	s := a.rp()
	var items []commandItem
	scanned := 0
	var walk func(nodes []*rpNode, dir string)
	walk = func(nodes []*rpNode, dir string) {
		for _, node := range nodes {
			if scanned >= commandPaletteScanLimit {
				return
			}
			scanned++
			if node.dir {
				next := node.name
				if dir != "" {
					next = dir + "/" + node.name
				}
				walk(node.children, next)
				continue
			}
			path := rpJoinPath(dir, node.name)
			name := node.name
			change := node.change
			item := commandItem{
				id:     "file:" + path,
				title:  name,
				detail: path,
				icon:   rpFileIcon(name),
				run: func() {
					a.screen = ScreenMain
					a.rightOpen, a.rightTab = true, tabFiles
					a.rp().filter = name
				},
			}
			if change != nil {
				item.secondary = func() {
					a.screen = ScreenMain
					a.rightOpen, a.rightTab = true, tabChanges
				}
				item.secondaryLabel = "查看改动"
			} else {
				item.secondaryLabel = "查看改动"
				item.secondaryDisabled = true
				item.secondaryReason = "这个文件没有改动"
			}
			items = append(items, item)
		}
	}
	walk(s.tree, "")
	return items
}

// commandPaletteSettings 是设置项组：静态表来自设置导航（同一份 19 节）。
func (a *App) commandPaletteSettings() []commandItem {
	items := make([]commandItem, 0, len(settingsSectionDefs))
	for _, section := range settingsSectionDefs {
		section := section
		items = append(items, commandItem{
			id:     "setting:" + fmt.Sprintf("%d", section.id),
			title:  "设置 · " + section.label,
			detail: "打开设置分节",
			icon:   section.icon,
			run: func() {
				a.screen = ScreenSettings
				a.settingsSection = section.id
			},
		})
	}
	return items
}

// ---- 匹配与拼音首字母 ----

// commandPaletteMatch 判断一条候选是否命中查询：标题子串优先（给出高亮区间），
// 其次是路径 / 说明，最后是标题的拼音首字母（S-CP-07）。
func commandPaletteMatch(title, detail, query string) (bool, int, int) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true, -1, -1
	}
	lowerTitle := strings.ToLower(title)
	if at := strings.Index(lowerTitle, q); at >= 0 {
		return true, at, at + len(q)
	}
	if strings.Contains(strings.ToLower(detail), q) {
		return true, -1, -1
	}
	initials := strings.ToLower(commandPaletteInitials(title))
	if strings.Contains(initials, q) {
		return true, -1, -1
	}
	return false, -1, -1
}

// commandPaletteInitials 取文字的拼音首字母：汉字查表，ASCII 字母数字原样
// 保留，其余字符（空格、标点）跳过。没有收进表里的汉字退化为原字，查询仍可
// 按原字命中标题。
func commandPaletteInitials(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 128 {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			}
			continue
		}
		if initial, ok := commandPaletteInitialTable[r]; ok {
			b.WriteByte(initial)
			continue
		}
		if r >= 0x4E00 && r <= 0x9FFF {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// commandPaletteInitialTable 是界面语用到的汉字到拼音首字母的映射。它不是
// 完整拼音表：只覆盖本产品的命令、分组与设置名称；表外汉字仍可按原文匹配。
var commandPaletteInitialTable = map[rune]byte{
	'停': 't', '止': 'z', '这': 'z', '次': 'c', '运': 'y', '行': 'x',
	'发': 'f', '送': 's', '当': 'd', '前': 'q', '草': 'c', '稿': 'g',
	'新': 'x', '建': 'j', '会': 'h', '话': 'h',
	'结': 'j', '束': 's', '分': 'f', '支': 'z',
	'清': 'q', '除': 'c', '目': 'm', '标': 'b',
	'归': 'g', '档': 'd', '导': 'd', '出': 'c',
	'重': 'c', '连': 'l', '接': 'j',
	'切': 'q', '换': 'h', '设': 's', '置': 'z',
	'文': 'w', '件': 'j', '搜': 's', '索': 's',
	'命': 'm', '令': 'l', '账': 'z', '户': 'h',
	'用': 'y', '量': 'l', '个': 'g', '性': 'x', '化': 'h',
	'外': 'w', '观': 'g', '通': 't', '快': 'k', '键': 'j',
	'模': 'm', '型': 'x', '供': 'g', '应': 'y', '多': 'd', '态': 't',
	'时': 's', '知': 'z', '宠': 'c', '物': 'w', '权': 'q', '限': 'x',
	'插': 'c', '部': 'b', '代': 'd', '理': 'l', '已': 'y', '关': 'g', '于': 'y',
	'修': 'x', '路': 'l', '径': 'j', '参': 'c', '考': 'k',
	'旧': 'j', '实': 's', '现': 'x', '对': 'd', '照': 'z',
	'最': 'z', '近': 'j', '使': 's', '没': 'm', '有': 'y', '匹': 'p', '配': 'p',
	'项': 'x', '或': 'h', '选': 'x', '择': 'z', '执': 'z', '组': 'z',
	'级': 'j', '动': 'd', '作': 'z', '闭': 'b', '打': 'd', '开': 'k',
	'区': 'q', '尚': 's', '未': 'w', '入': 'r', '界': 'j', '面': 'm',
	'在': 'z', '的': 'd', '回': 'h', '复': 'f', '服': 'f', '务': 'w',
	'输': 's', '容': 'r', '干': 'g', '还': 'h', '改': 'g',
}
