package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	gui "github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// 会话内搜索（G-A1-01–09、G-A5-01–02）：标题栏右上方的一块浮层，⌘F 打开，
// 输入即时在本帧时间线已经拿到的文本里找命中。
//
// 三件事都不写事实：
//   - 命中来自 RowSnapshot / 样例轮次（用户消息、助手正文、工具参数与输出、
//     思考块），没有新增协议方法：App Server 不提供会话内检索（gaps.md G-02）；
//   - 「折叠区自动展开」只改本地的折叠位与气泡展开位；
//   - 当前命中与其余命中只用行内底色区分，文本本身不改（A5 的 368 条）。
//
// 浮层的一次搜索全部是可丢的界面状态：不落盘，切会话、关浮层即清空。
const (
	// 浮层尺寸与位置（A1-01）：宽取 min(420, 主区 − 2×内边距)，浮在标题栏上，
	// 不推挤标题栏与时间线。输入行 44、结果行 36（A1-01、A1-06）。
	sessionSearchWidth        = 420
	sessionSearchMinWidth     = 240
	sessionSearchTop          = 10
	sessionSearchInputHeight  = 44
	sessionSearchResultHeight = 36
	sessionSearchEmptyHeight  = 24

	// 文案（A1-03、A1-04、G-B2-08 的空态口径）。
	sessionSearchPlaceholder = "搜索聊天..."
	sessionSearchLabel       = "搜索聊天消息"
	sessionSearchEmptyText   = "没有匹配项"

	// 跳转后继续把当前命中滚进视口的时长：折叠区展开是 300ms，留一倍余量。
	sessionSearchRevealWindow = 600 * time.Millisecond

	// 关闭后继续写回滚动位置的帧数：布局那一帧的滚动（上一帧排上的
	// ScrollIntoView）先落地，还原跟在后面。
	sessionSearchRestoreFrames = 8
)

// colorFloat 是浮层的底色（design-baseline §5.3 的层级：底 → 卡 → 浮层 →
// 交互面；明 #ffffff / 暗 #222222）。
var colorFloat = colorSet{gui.Hex("#ffffff"), gui.Hex("#222222")}

// colorSearchHitCurrent 是当前命中片段的底色（A5-02：橙色 70%）；其余命中沿用
// colorSearchHit 的黄底（A5-01）。
var colorSearchHitCurrent = colorSet{gui.RGBA(255, 132, 0, 0.70), gui.RGBA(255, 170, 60, 0.55)}

// searchScope 是一条命中落在哪类文本上：决定跳过去时要展开什么。
type searchScope int

const (
	searchScopeUser searchScope = iota
	searchScopeAssistant
	searchScopeTool
	searchScopeThinking
)

// searchUnit 是一段可搜索的文本。key 是渲染端与索引端共用的标识（同一次
// 渲染里两边用同一套 key，命中才能落到行上）。
type searchUnit struct {
	key   string
	text  string
	row   int
	scope searchScope
	// group 为真表示这段文本在轮次组里：跳过去要先展开 openOf(row)。
	group bool
	// blocks 是还需要展开的块键（思考块、工具详情）：见 blockOf。
	blocks []string
}

// searchHit 是一条命中：unit 是 units 的下标，start / end 是文本里的字节区间。
type searchHit struct {
	unit  int
	start int
	end   int
}

// searchRange 是画高亮用的区间；current 标出当前定位项（A5-02）。
type searchRange struct {
	start   int
	end     int
	current bool
}

// sessionSearchState 是浮层的全部视图状态（可丢）。
type sessionSearchState struct {
	open  bool
	query string
	// think 是「搜索范围」里的思考块开关（A1 的需求 5：可开关，默认开）。
	think bool

	units   []searchUnit
	hits    []searchHit
	byUnit  map[string][]searchRange
	current int

	// session 是打开时的会话标识：切会话就关掉，不把上一个会话的查询带过去。
	session string

	// savedY 是打开前的滚动位置，关闭时恢复（需求 6）；restoreFrames 是还要
	// 写几帧（关闭的那几帧里可能还有上一帧排上的 ScrollIntoView 落到布局上，
	// 所以在布局之后写，才是这一帧的最后一句）。
	savedY        float32
	saved         bool
	restoreFrames int

	// focus / selectAll 在下一帧把焦点与全选落到输入框；returnFocus 在关闭后
	// 把焦点还给时间线。
	focus       bool
	selectAll   bool
	returnFocus bool

	// revealAt 是最近一次跳转的帧时钟：从它起的一小段时间里每帧把当前命中
	// 滚进视口（等折叠展开之后再定位）。
	revealAt time.Time
	// now 是本帧的帧时钟（c.Now()），供没有 Context 的回调使用。
	now time.Time
	// lastCurrent 是上一帧的当前项，用来判断「当前项换了」。
	lastCurrent string

	// opened* 记下搜索期间展开过的折叠位，关闭时复原。
	openedBlocks map[string]bool
	openedRounds map[int]bool
	openedUsers  map[int]bool
}

// —— 键位 ——

// sessionSearchKeys 是关闭态的全局键：主界面上 ⌘F 打开浮层（A1-01）。已经
// 打开时只重新聚焦输入框并全选，不新开第二个（R-GS-003）。
func (a *App) sessionSearchKeys(c *gui.Context) {
	if a.screen != ScreenMain || a.palette.open {
		return
	}
	if !c.Shortcut(gui.Cmd, gui.KeyF) {
		return
	}
	// 条件：主区有活跃会话（R-GS-001）；没有会话时不给搜索入口。
	if a.wired() {
		if _, ok := a.liveSession(); !ok {
			return
		}
	}
	if a.sessionSearch.open {
		a.sessionSearch.focus = true
		a.sessionSearch.selectAll = true
		return
	}
	a.openSessionSearch()
}

// openSessionSearch 打开浮层：清空查询（除时间线选中的文本预填）、记下打开前
// 的滚动位置、把焦点排到输入框。
func (a *App) openSessionSearch() {
	s := &a.sessionSearch
	s.open = true
	s.think = true
	s.current = -1
	s.units, s.hits, s.byUnit = nil, nil, nil
	s.query = prefillSearchQuery(a.timelineSelected)
	s.savedY, s.saved = a.feed.Y, true
	s.restoreFrames = 0
	s.focus = true
	s.selectAll = true
	s.returnFocus = false
	s.revealAt = time.Time{}
	s.lastCurrent = ""
	s.openedBlocks = map[string]bool{}
	s.openedRounds = map[int]bool{}
	s.openedUsers = map[int]bool{}
	if snap, ok := a.liveSession(); ok {
		s.session = snap.ID
	} else {
		s.session = ""
	}
}

// closeSessionSearch 关掉浮层：清查询与命中、复原搜索期间展开的折叠位、把滚动
// 位置还原到打开前（需求 6），输入框随浮层卸载、焦点还给时间线（R-GS-045、
// R-GS-052）。
func (a *App) closeSessionSearch() {
	s := &a.sessionSearch
	if !s.open {
		return
	}
	for key := range s.openedBlocks {
		if p := a.blockOf(key, false); *p {
			*p = false
		}
	}
	for row := range s.openedRounds {
		if p := a.openOf(row); *p {
			*p = false
		}
	}
	for row := range s.openedUsers {
		delete(a.userExpanded, row)
	}
	s.open = false
	s.query = ""
	// 预填的选中文本只用一次：不把上一次的选区带到下一次打开（R-GS-009）。
	a.timelineSelected = ""
	s.units, s.hits, s.byUnit = nil, nil, nil
	s.current = -1
	s.focus = false
	s.selectAll = false
	s.returnFocus = true
	s.revealAt = time.Time{}
	s.lastCurrent = ""
	s.session = ""
	if s.saved {
		s.restoreFrames = sessionSearchRestoreFrames
		s.saved = false
	}
}

// sessionSearchTextKey 接管输入框里的键（A1-03）：Esc 关闭、回车下一个、
// Shift+回车上一个、↑ / ↓ 同上两个按钮、Home / End 跳到首 / 末条命中、
// ⌘F 重新聚焦并全选。
func (a *App) sessionSearchTextKey(k gui.TextKey) bool {
	switch {
	case k.Mods == 0 && k.Key == gui.KeyEscape:
		a.closeSessionSearch()
		return true
	case k.Mods == 0 && k.Key == gui.KeyEnter, k.Mods == 0 && k.Key == gui.KeyDown:
		a.moveSessionSearchHit(1)
		return true
	case k.Mods == gui.Shift && k.Key == gui.KeyEnter, k.Mods == 0 && k.Key == gui.KeyUp:
		a.moveSessionSearchHit(-1)
		return true
	case k.Mods == 0 && k.Key == gui.KeyHome:
		a.sessionSearchJump(-1)
		return true
	case k.Mods == 0 && k.Key == gui.KeyEnd:
		a.sessionSearchJump(1)
		return true
	case k.Mods == gui.Cmd && k.Key == gui.KeyF:
		a.sessionSearch.focus = true
		a.sessionSearch.selectAll = true
		return true
	}
	return false
}

// moveSessionSearchHit 把当前命中前后移动一格，首尾回卷（A1-07、A1-08）。
func (a *App) moveSessionSearchHit(delta int) {
	s := &a.sessionSearch
	n := len(s.hits)
	if n == 0 {
		return
	}
	if s.current < 0 || s.current >= n {
		s.current = 0
	} else {
		s.current = ((s.current+delta)%n + n) % n
	}
	// 旧命中降为普通底色（R-GS-374）；只有一条时回卷到它自己（R-GS-069、R-GS-077）。
	a.rebuildSearchRanges()
	a.openFoldsForCurrent()
	s.revealAt = s.now
}

// sessionSearchJump 把当前命中跳到首条（edge<0）或末条（edge>0），与点
// G-A1-07 / G-A1-08 按钮走到同一个落点；没有命中时不动。
func (a *App) sessionSearchJump(edge int) {
	s := &a.sessionSearch
	n := len(s.hits)
	if n == 0 {
		return
	}
	if edge < 0 {
		s.current = 0
	} else {
		s.current = n - 1
	}
	a.rebuildSearchRanges()
	a.openFoldsForCurrent()
	s.revealAt = s.now
}

// sessionSearchRestore 在关闭后的几帧里把时间线的滚动位置写回打开前（需求 6）。
// 在布局之后写：这一帧里别的滚动（例如上一帧排上的 ScrollIntoView）先落地，
// 最后一句才是还原。
func (a *App) sessionSearchRestore(c *gui.Context) {
	s := &a.sessionSearch
	if s.open || s.restoreFrames <= 0 {
		return
	}
	a.feed.Y = s.savedY
	s.restoreFrames--
	c.AnimationFrame()
}

// —— 索引 ——

// sessionSearchFrame 每帧准备一次命中：主界面上浮层开着时重建索引，关着时把
// 滚动位置还原一次。放在 layout 之前，时间线这一帧就能按新命中画高亮。
func (a *App) sessionSearchFrame(c *gui.Context) {
	s := &a.sessionSearch
	if !s.open {
		return
	}
	// 进了设置等其他屏：浮层不在场，状态一并清掉。
	if a.screen != ScreenMain {
		a.closeSessionSearch()
		return
	}
	// 切了会话：查询与命中都不带过去（R-TB-049）。
	if snap, ok := a.liveSession(); ok {
		if s.session != "" && snap.ID != s.session {
			a.closeSessionSearch()
			return
		}
		s.session = snap.ID
	}
	s.now = c.Now()
	a.reindexSessionSearch()
}

// reindexSessionSearch 按当前查询重建命中。每帧重建：时间线在流式增长时命中
// 要跟着走；文本量是当前会话的行文，重建比按帧找差异便宜。
func (a *App) reindexSessionSearch() {
	s := &a.sessionSearch
	// 本机同步重建：计数与命中同帧更新，不存在「进行中」的假 0（R-GS-021、R-GS-081）。
	// 先记下上一帧的当前项（按单元 key + 起点认回来），再重建语料与命中。
	prevKey, prevStart := "", -1
	if h, ok := s.currentHit(); ok && h.unit >= 0 && h.unit < len(s.units) {
		prevKey, prevStart = s.units[h.unit].key, h.start
	}
	s.units = a.sessionSearchUnits()
	s.hits = nil
	s.byUnit = map[string][]searchRange{}

	query := strings.TrimSpace(s.query)
	if query != "" {
		for i := range s.units {
			for _, span := range searchFindAll(s.units[i].text, query) {
				s.hits = append(s.hits, searchHit{unit: i, start: span[0], end: span[1]})
			}
		}
		// 当前项按 (unit key, start) 认回来；认不回来就落到第一条。
		s.current = -1
		if len(s.hits) > 0 {
			s.current = 0
			if prevStart >= 0 {
				for i, h := range s.hits {
					if s.units[h.unit].key == prevKey && h.start == prevStart {
						s.current = i
						break
					}
				}
			}
		}
	} else {
		s.current = -1
	}
	// 当前项换了（新查询、或原项被流式内容挤走）：重新给一段滚动时间，
	// 等折叠展开之后再定位。
	if key := s.currentHitKey(); key != s.lastCurrent {
		s.lastCurrent = key
		s.revealAt = s.now
	}
	a.rebuildSearchRanges()
	a.openFoldsForCurrent()
}

// currentHit 取当前命中。
func (s *sessionSearchState) currentHit() (searchHit, bool) {
	if s.current < 0 || s.current >= len(s.hits) {
		return searchHit{}, false
	}
	return s.hits[s.current], true
}

// currentHitKey 取当前命中所在单元的 key。
func (s *sessionSearchState) currentHitKey() string {
	h, ok := s.currentHit()
	if !ok || h.unit < 0 || h.unit >= len(s.units) {
		return ""
	}
	return s.units[h.unit].key
}

// rebuildSearchRanges 把命中按单元归拢，并标出哪一段是当前项。
func (a *App) rebuildSearchRanges() {
	s := &a.sessionSearch
	s.byUnit = make(map[string][]searchRange, len(s.units))
	for i, h := range s.hits {
		if h.unit < 0 || h.unit >= len(s.units) {
			continue
		}
		key := s.units[h.unit].key
		s.byUnit[key] = append(s.byUnit[key], searchRange{
			start:   h.start,
			end:     h.end,
			current: i == s.current,
		})
	}
}

// openFoldsForCurrent 展开当前命中所在的折叠区，等渲染时就能定位（R-GS-378）。
func (a *App) openFoldsForCurrent() {
	s := &a.sessionSearch
	h, ok := s.currentHit()
	if !ok {
		return
	}
	unit := s.units[h.unit]
	if unit.group {
		if p := a.openOf(unit.row); !*p {
			*p = true
			s.openedRounds[unit.row] = true
		}
	}
	for _, key := range unit.blocks {
		if p := a.blockOf(key, false); !*p {
			*p = true
			s.openedBlocks[key] = true
		}
	}
	if unit.scope == searchScopeUser && bubbleOverflows(unit.text) {
		if a.userExpanded == nil {
			a.userExpanded = map[int]bool{}
		}
		if !a.userExpanded[unit.row] {
			a.userExpanded[unit.row] = true
			s.openedUsers[unit.row] = true
		}
	}
}

// sessionSearchUnits 把本帧时间线上的文本按显示顺序摊成可搜索单元。
func (a *App) sessionSearchUnits() []searchUnit {
	if snap, ok := a.liveSession(); ok {
		return a.liveSearchUnits(snap)
	}
	return a.roundSearchUnits()
}

// liveSearchUnits 是连着事实时的语料：行的种类与 liveRow 的渲染一一对应。
func (a *App) liveSearchUnits(snap SessionSnapshot) []searchUnit {
	var units []searchUnit
	for i := range snap.Rows {
		row := snap.Rows[i]
		switch row.Kind {
		case RowUser:
			units = append(units, searchUnit{
				key: searchKeyUser(i), text: row.Text, row: i, scope: searchScopeUser,
			})
		case RowThinking:
			if !a.sessionSearch.think {
				continue
			}
			units = append(units, searchUnit{
				key: searchKeyThinking(i), text: row.Text, row: i, scope: searchScopeThinking,
				blocks: []string{searchThinkBlock(i)},
			})
		case RowToolGroup:
			for n := range row.Children {
				units = append(units, a.toolSearchUnits(i, n, toolOp(row.Children[n]), true)...)
			}
		case RowTool:
			units = append(units, a.toolSearchUnits(i, 0, toolOp(row), false)...)
		case RowCompression:
			// 压缩摘要是系统写的分隔行，不是对话正文，不参与检索。
		default:
			units = append(units, searchUnit{
				key: searchKeyAssistant(i), text: row.Text, row: i, scope: searchScopeAssistant,
			})
		}
	}
	return units
}

// roundSearchUnits 是样例模式的语料：与 round / assistant 的渲染一一对应。
func (a *App) roundSearchUnits() []searchUnit {
	var units []searchUnit
	for i := range a.rounds {
		r := a.rounds[i]
		units = append(units, searchUnit{
			key: searchKeyUser(i), text: r.user, row: i, scope: searchScopeUser,
		})
		if r.unknown {
			// 未知提交只画占位与说明行：没有正文、没有工具行。
			continue
		}
		if r.thinking != "" && a.sessionSearch.think {
			units = append(units, searchUnit{
				key: searchKeyThinking(i), text: r.thinking, row: i, scope: searchScopeThinking,
				blocks: []string{searchThinkBlock(i)},
			})
		}
		units = append(units, searchUnit{
			key: searchKeyAssistant(i), text: r.reply, row: i, scope: searchScopeAssistant,
		})
		for n := range r.ops {
			units = append(units, a.toolSearchUnits(i, n, r.ops[n], true)...)
		}
	}
	return units
}

// toolSearchUnits 是一条工具行的三段文本：参数（行上）、输入与输出（展开里）。
func (a *App) toolSearchUnits(row, n int, o op, grouped bool) []searchUnit {
	units := make([]searchUnit, 0, 3)
	if param := primaryParam(o); param != "" {
		units = append(units, searchUnit{
			key: searchKeyToolParam(row, n), text: param, row: row,
			scope: searchScopeTool, group: grouped,
		})
	}
	if o.input != "" {
		units = append(units, searchUnit{
			key: searchKeyToolInput(row, n), text: o.input, row: row,
			scope: searchScopeTool, group: grouped,
			blocks: []string{searchToolBlock(row, n)},
		})
	}
	if o.output != "" {
		units = append(units, searchUnit{
			key: searchKeyToolOutput(row, n), text: o.output, row: row,
			scope: searchScopeTool, group: grouped,
			blocks: []string{searchToolBlock(row, n)},
		})
	}
	return units
}

// —— 命中计算 ——

// searchFindAll 在 text 里找 query 的全部不重叠出现，返回字节区间。比较按
// Unicode 小写走（与界面上的等宽/中英混排无关），区间给回原文本的字节偏移。
func searchFindAll(text, query string) [][2]int {
	qr := []rune(query)
	if len(qr) == 0 || text == "" {
		return nil
	}
	tr := []rune(text)
	if len(tr) < len(qr) {
		return nil
	}
	// 每个 rune 的字节起点；末尾多一格给「区间右端」。
	offs := make([]int, len(tr)+1)
	at := 0
	for i, r := range tr {
		offs[i] = at
		at += utf8.RuneLen(r)
	}
	offs[len(tr)] = at

	for i := range qr {
		qr[i] = unicode.ToLower(qr[i])
	}
	var out [][2]int
	for i := 0; i+len(qr) <= len(tr); {
		if searchEqualFold(tr[i:i+len(qr)], qr) {
			out = append(out, [2]int{offs[i], offs[i+len(qr)]})
			i += len(qr)
			continue
		}
		i++
	}
	return out
}

// searchEqualFold 比较同长度的一段 rune 与小写化后的查询。
func searchEqualFold(runs, lower []rune) bool {
	for i := range runs {
		if unicode.ToLower(runs[i]) != lower[i] {
			return false
		}
	}
	return true
}

// searchRanges 取某单元本帧的命中区间。
func (a *App) searchRanges(key string) []searchRange {
	if !a.sessionSearch.open {
		return nil
	}
	return a.sessionSearch.byUnit[key]
}

// —— 渲染：行内高亮 ——

// searchText 画一段可搜索的文本：有命中时按区间拆成带底色的 span，当前项用
// 更深的橙底（A5-01、A5-02）。没有命中时与原来的 Text 完全一样，不动排版。
func (a *App) searchText(c *gui.Context, key, text string, step fontStep, color gui.Color) gui.Element {
	ranges := a.searchRanges(key)
	if len(ranges) == 0 {
		return step.style(c.Theme(), gui.Text(c, text)).TextColor(color)
	}
	return gui.RichText(c, searchSpans(c.Theme(), text, ranges, step, color)...)
}

// searchSpans 把文本按命中区间拆成 span：命中片段带底色，当前项底色更深。
func searchSpans(t *gui.Theme, text string, ranges []searchRange, step fontStep, color gui.Color) []gui.Span {
	family := step.family
	if family == codeFont {
		family = codeFamily()
	}
	// span 不带点击动作（R-GS-370）；span 的文本就是原文，读屏按原文读出（R-GS-371）；
	// 当前项按索引标出，不随可见性撤销（R-GS-381）。
	spans := make([]gui.Span, 0, 2*len(ranges)+1)
	at := 0
	add := func(s string) {
		if s == "" {
			return
		}
		spans = append(spans, gui.Span{
			Text: s, Font: family, Size: step.sizeIn(t), Weight: step.weight, Color: color,
		})
	}
	for _, r := range ranges {
		start, end := r.start, r.end
		if start < at || start > len(text) {
			continue
		}
		if end > len(text) {
			end = len(text)
		}
		if start >= end {
			continue
		}
		add(text[at:start])
		hitColor := colorSearchHit.in(t)
		if r.current {
			hitColor = colorSearchHitCurrent.in(t)
		}
		spans = append(spans, gui.Span{
			Text: text[start:end], Font: family, Size: step.sizeIn(t),
			Weight: step.weight, Color: color, Background: hitColor, Radius: radiusSmall,
		})
		at = end
	}
	add(text[at:])
	return spans
}

// searchCodeBlock 是带命中高亮的等宽浅底块：工具输入 / 输出用（与 codeBlock
// 同一个外观，只多一层行内高亮）。
func (a *App) searchCodeBlock(c *gui.Context, key, text string) {
	t := c.Theme()
	gui.Box(c).Shrink(0).Padding(space2).Radius(radius).Background(t.SurfacePressed).
		Children(func() {
			a.searchText(c, key, text, typeCode, t.Text).Selectable()
		})
}

// searchReveal 在跳转后的一小段时间里把当前命中所在的行滚进视口（A5-02 的
// 375）。调用点就在画那一行的地方：折叠展开的那一帧也会跟着滚。
func (a *App) searchReveal(c *gui.Context, el gui.Element, row int, scope searchScope) {
	s := &a.sessionSearch
	if !s.open || s.revealAt.IsZero() {
		return
	}
	if c.Now().Sub(s.revealAt) > sessionSearchRevealWindow {
		s.revealAt = time.Time{}
		return
	}
	h, ok := s.currentHit()
	if !ok || h.unit < 0 || h.unit >= len(s.units) {
		return
	}
	if unit := s.units[h.unit]; unit.row != row || unit.scope != scope {
		return
	}
	el.ScrollIntoView(gui.ScrollIntoViewOptions{Block: gui.ScrollCenter})
	c.AnimationFrame()
}

// assistantMarkdown 画一段助手正文：搜索时把命中片段包一层 <mark>（A5 的行内
// 高亮），当前项再包一层 <u> 以便与同一条消息里的其余命中区分。
func (a *App) assistantMarkdown(c *gui.Context, key, text string) gui.Element {
	ranges := a.searchRanges(key)
	if len(ranges) == 0 {
		return markdown.View(c, text, &a.md)
	}
	return markdown.View(c, searchMarkedMarkdown(text, ranges), &a.md)
}

// searchMarkedMarkdown 把命中片段包成 <mark>…</mark>（当前项再包一层 <u>），
// 返回新的 Markdown 源。代码块与行内代码、链接目标、HTML 标签里的命中不包，
// 避免把标记塞进不该解析成行内标签的地方。
func searchMarkedMarkdown(text string, ranges []searchRange) string {
	if len(ranges) == 0 {
		return text
	}
	protected := searchProtectedRanges(text)
	var b strings.Builder
	at := 0
	for _, r := range ranges {
		start, end := r.start, r.end
		if start < at || start >= len(text) || end > len(text) || start >= end {
			continue
		}
		if strings.ContainsRune(text[start:end], '\n') {
			continue
		}
		if searchOverlaps(protected, start, end) {
			continue
		}
		b.WriteString(text[at:start])
		if r.current {
			b.WriteString("<mark><u>")
			b.WriteString(text[start:end])
			b.WriteString("</u></mark>")
		} else {
			b.WriteString("<mark>")
			b.WriteString(text[start:end])
			b.WriteString("</mark>")
		}
		at = end
	}
	if at == 0 {
		return text
	}
	b.WriteString(text[at:])
	return b.String()
}

// searchProtectedRanges 找出不该往里插行内标签的区间：围栏代码块、行内代码、
// 链接目标、HTML 标签与转义字符。
func searchProtectedRanges(text string) [][2]int {
	var out [][2]int
	fence := 0 // 0 不在围栏里；'`' / '~' 是围栏字符
	offset := 0
	for _, line := range strings.SplitAfter(text, "\n") {
		lineStart := offset
		offset += len(line)
		body := strings.TrimRight(line, "\n")
		trimmed := strings.TrimLeft(body, " \t")
		indent := len(body) - len(trimmed)
		if fence == 0 {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				fence = int(trimmed[0])
				out = append(out, [2]int{lineStart + indent, lineStart + len(body)})
				continue
			}
		} else if len(trimmed) >= 3 && trimmed[0] == byte(fence) &&
			strings.Trim(trimmed, string(trimmed[0])) == "" {
			fence = 0
			out = append(out, [2]int{lineStart + indent, lineStart + len(body)})
			continue
		}
		if fence != 0 {
			out = append(out, [2]int{lineStart, lineStart + len(body)})
			continue
		}
		out = append(out, searchLineProtected(body, lineStart)...)
	}
	return out
}

// searchLineProtected 找一行的行内代码、链接目标、HTML 标签与转义字符。
func searchLineProtected(line string, base int) [][2]int {
	var out [][2]int
	for i := 0; i < len(line); {
		switch line[i] {
		case '\\':
			if i+1 < len(line) {
				out = append(out, [2]int{base + i, base + i + 2})
				i += 2
				continue
			}
			i++
		case '`':
			run := searchRunLen(line[i:], '`')
			end := strings.Index(line[i+run:], strings.Repeat("`", run))
			if end < 0 {
				i += run
				continue
			}
			stop := i + run + end + run
			out = append(out, [2]int{base + i, base + stop})
			i = stop
		case '<':
			end := strings.IndexByte(line[i:], '>')
			if end < 0 {
				i++
				continue
			}
			out = append(out, [2]int{base + i, base + i + end + 1})
			i += end + 1
		case ']':
			// 链接目标：]( ... )
			if i+1 < len(line) && line[i+1] == '(' {
				end := strings.IndexByte(line[i+2:], ')')
				if end < 0 {
					i++
					continue
				}
				stop := i + 2 + end + 1
				out = append(out, [2]int{base + i, base + stop})
				i = stop
				continue
			}
			i++
		default:
			i++
		}
	}
	return out
}

// searchRunLen 是 s 开头连续 c 的长度。
func searchRunLen(s string, c byte) int {
	n := 0
	for n < len(s) && s[n] == c {
		n++
	}
	return n
}

// searchOverlaps 判断区间与保护区相交。
func searchOverlaps(protected [][2]int, start, end int) bool {
	for _, p := range protected {
		if start < p[1] && p[0] < end {
			return true
		}
	}
	return false
}

// —— 单元标识（渲染端与索引端共用）——

func searchKeyUser(row int) string          { return fmt.Sprintf("u:%d", row) }
func searchKeyAssistant(row int) string     { return fmt.Sprintf("a:%d", row) }
func searchKeyThinking(row int) string      { return fmt.Sprintf("t:%d", row) }
func searchKeyToolParam(row, n int) string  { return fmt.Sprintf("p:%d:%d", row, n) }
func searchKeyToolInput(row, n int) string  { return fmt.Sprintf("n:%d:%d", row, n) }
func searchKeyToolOutput(row, n int) string { return fmt.Sprintf("o:%d:%d", row, n) }
func searchThinkBlock(row int) string       { return fmt.Sprintf("think:%d", row) }
func searchToolBlock(row, n int) string     { return fmt.Sprintf("tool:%d:%d", row, n) }

// prefillSearchQuery 是打开浮层时的预填：时间线里选中的文本（单行化）。
func prefillSearchQuery(selection string) string {
	selection = strings.TrimSpace(selection)
	if selection == "" {
		return ""
	}
	selection = strings.Join(strings.Fields(selection), " ")
	if runes := []rune(selection); len(runes) > 120 {
		selection = string(runes[:120])
	}
	return selection
}

// —— 浮层 ——

// sessionSearchOverlay 画浮层本体：整窗一层透明的点击区（点浮层外关闭），
// 上面是浮层。位置贴着主区右缘，宽度按窗口收窄，不溢出（R-GS-010）。
func (a *App) sessionSearchOverlay(c *gui.Context) {
	s := &a.sessionSearch
	if !s.open || a.screen != ScreenMain {
		return
	}
	t := c.Theme()
	windowWidth, _ := c.Size()
	sidebar := a.sidebarWidthValue()
	if a.sidebarCollapsed || narrowWindow(c) {
		// 窄窗里左栏只留边条（与 layout 同一判据）：浮层跟着主区的实际左缘走。
		sidebar = SidebarRailWidth
	}
	panel := a.rpWidth(windowWidth, sidebar)
	width := float32(sessionSearchWidth)
	if avail := windowWidth - sidebar - panel - 2*space4; avail < width {
		width = avail
	}
	if width < sessionSearchMinWidth {
		width = sessionSearchMinWidth
	}
	x := windowWidth - panel - width - space4
	if x < space4 {
		x = space4
	}

	gui.Overlay(c, func() {
		// 点击浮层外关闭（A1-01 的 R-GS-002）：透明的一层，接住指针但不画东西。
		gui.Box(c).Absolute().Fill().AccessHidden(true).OnClick(a.closeSessionSearch)

		// TB-06 消息内搜索浮层容器
		box := gui.Column(c).Absolute().Left(x).Top(sessionSearchTop).Width(width).
			Background(colorFloat.in(t)).Border(strokeHairline, t.Border).
			Radius(radiusCard).Shadow(0, 6, 18, 0, colorShadow.in(t)).
			Label("会话内搜索").Role(gui.RoleSearch)
		box.Children(func() {
			a.sessionSearchInputRow(c)
			if strings.TrimSpace(s.query) != "" {
				a.sessionSearchResultRow(c)
				if len(s.hits) == 0 {
					a.sessionSearchEmptyRow(c)
				}
			}
		})
		// 焦点不在输入框时（点了别处），浮层仍然接住 Esc。
		if box.OverlayShortcut(0, gui.KeyEscape) {
			a.closeSessionSearch()
		}
	})
}

// sessionSearchInputRow 是输入行（44px）：放大镜 + 输入框 + 关闭键。
func (a *App) sessionSearchInputRow(c *gui.Context) {
	t := c.Theme()
	s := &a.sessionSearch
	gui.Row(c).Shrink(0).Height(sessionSearchInputHeight).
		Padding(space1, space3).Gap(space2).AlignItems(gui.Center).Children(func() {
		// G-A1-02a 输入框左端的放大镜（20px，空查询也常驻）
		gui.Row(c).Shrink(0).Width(iconBlock).Height(iconBlock).Center().
			OnClick(func() { s.focus = true }).Children(func() {
			iconGlyphSize(c, iconSearch, iconBlock, t.TextMuted, "search")
		})
		// 输入框以 textbox 语义暴露，名称取 Label（不是 placeholder）；Label 只作
		// 可读名，不进可见文本（R-GS-042、R-GS-044）。
		// TB-07 / G-A1-04 搜索聊天...
		input := gui.TextInputBase(c, &s.query).
			Placeholder(sessionSearchPlaceholder).Label(sessionSearchLabel).Grow(1).
			FontSize(typeBody.sizeIn(t)).TextColor(t.Text).
			OnTextKey(a.sessionSearchTextKey)
		if s.focus {
			// 打开、再按 ⌘F、点放大镜都用 Focus 落到输入框：这一帧就生效。
			input = input.Focus()
			s.focus = false
		}
		if s.selectAll {
			input.SetTextSelection(0, len([]rune(s.query)))
			s.selectAll = false
		}
		// 关闭键 28px 热区，查询为空时也常驻可用（R-GS-051、R-GS-053）。
		if iconButton(c, iconButtonSpec{
			// TB-08 关闭搜索
			icon: iconXMark, label: "关闭搜索", tip: "关闭搜索 · Esc", size: rowRegular,
		}) {
			a.closeSessionSearch()
		}
	})
}

// sessionSearchResultRow 是结果行（36px，上边框）：左按钮组、右结果计数。
func (a *App) sessionSearchResultRow(c *gui.Context) {
	t := c.Theme()
	s := &a.sessionSearch
	gui.Box(c).Shrink(0).Height(strokeHairline).Background(t.Border)
	// G-A1-06 结果行（36px，上边框）：左按钮组、右结果计数
	gui.Row(c).Shrink(0).Height(sessionSearchResultHeight).
		Padding(space1, space3).Gap(space1).AlignItems(gui.Center).
		Label("搜索结果").Role(gui.RoleGroup).Children(func() {
		// 前后箭头同尺寸同层级，只有图标朝向不同（R-GS-074）。
		// R-TB-091：零命中时没有可跳的对象，两个箭头整块不画——不渲染成禁用态。
		arrowsVisible := len(s.hits) > 0
		if arrowsVisible {
			if iconButton(c, iconButtonSpec{
				// TB-09 上一个搜索结果
				icon: iconChevronUp, label: "上一个搜索结果",
				tip: "上一个搜索结果 · Shift+Enter", size: rowRegular,
			}) {
				a.moveSessionSearchHit(-1)
			}
			if iconButton(c, iconButtonSpec{
				// TB-10 下一个搜索结果
				icon: iconChevronDown, label: "下一个搜索结果",
				tip: "下一个搜索结果 · Enter", size: rowRegular,
			}) {
				a.moveSessionSearchHit(1)
			}
		}
		label := "不含思考块"
		if s.think {
			label = "含思考块"
		}
		if gui.Button(c, label).Label("搜索范围：" + label).
			Tooltip("搜索范围：思考块是否参与匹配").Height(rowCompact).Clicked() {
			s.think = !s.think
			a.openFoldsForCurrent()
		}
		gui.Spacer(c)
		// 只在已加载的行里找：还有更早轮次没载入时把口径写在同一行，计数不被读成
		// 整段历史的命中数。
		if a.sessionPartial() {
			typeMicro.style(t, gui.Text(c, partialHistoryNote)).TextColor(t.TextMuted).
				Label(partialHistoryNote).SingleLine()
		}
		// TB-11 / TB-12 / G-A1-09 结果计数（0 个结果 / {current} / {total} 个结果）
		count := "0 个结果"
		if n := len(s.hits); n > 0 {
			count = fmt.Sprintf("%d / %d 个结果", s.current+1, n)
		}
		typeSecondary.style(t, gui.Text(c, count)).TextColor(t.TextMuted).
			Label(count).Role(gui.RoleStatus)
	})
}

// sessionSearchEmptyRow 是无命中时的一行空态文案（与命令面板同一个口径）。
func (a *App) sessionSearchEmptyRow(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Height(sessionSearchEmptyHeight).
		Padding(0, space3).AlignItems(gui.Center).Children(func() {
		typeMicro.style(t, gui.Text(c, sessionSearchEmptyText)).TextColor(t.TextMuted)
	})
}
