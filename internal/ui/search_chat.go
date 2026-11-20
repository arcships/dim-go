package ui

import (
	"fmt"
	"runtime"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 搜索聊天对话框（requirements-global §2.2，G-A2-01–G-A2-14）：⌘G 或左栏「搜索」
// 打开一个模态对话框，在**跨会话**的会话目录里找一条聊天并打开它。
//
// 事实来源只有一个：会话目录（Source.SessionNames / SessionTitles /
// SessionPlaces）。protocol v3 没有跨会话检索方法（会话内检索也是本端在已加载
// 的行文里做的，见 search.go），所以这里只在**已加载的目录条目**上做本地过滤，
// 对话框里逐字写「仅搜索已加载的会话」——不把结果说成整段历史的命中，也不假装
// 检索在服务端进行。
//
// 因此读态只有三种，且都由事实说：
//   - 正在加载：连着 Source，但目录还没读回（SessionNames 是 nil）；
//   - 读取失败：控制连接已断开（目录不会再刷新），给原因与「重试」（重连接口）；
//   - 就绪：目录已读回（空切片也算读回：没有会话，走空态）。
//
// 命中摘要：目录只带标题（没有标题时是首条消息预览）与工作区，所以「摘要」就是
// 标题里命中的那一段；标题没命中而工作区名命中时，摘要落在工作区那一段上。
// 会话正文不参与检索，这一点写在对话框底部的口径行里。

const (
	// 尺寸按参考产品的采集值：max-w-xl 448；输入行 44，结果行 40。
	chatSearchWidth       = 448
	chatSearchInputHeight = 44
	chatSearchRowHeight   = 40
	chatSearchHeaderH     = rowCompact
	chatSearchEmptyHeight = 74
	chatSearchNoteHeight  = rowCompact

	// 每页 8 条（G-A2-04「结果最多 8 条」）；⌘1–⌘8 的提示也按这个上限。
	chatSearchPageSize = 8

	// 文案（逐字，见 requirements-global 的期望列）。
	chatSearchTitle       = "搜索聊天"
	chatSearchDescription = "搜索并打开已有聊天"
	// chatSearchLabel 是输入框的读屏名（与对话框名区分：对话框叫「搜索聊天」，
	// 输入框读作搜索输入框，见 G-A2-04 的无障碍口径）。
	chatSearchLabel       = "搜索输入框"
	chatSearchPlaceholder = "搜索聊天"
	chatSearchEmptyText   = "未找到聊天"
	chatSearchLoadingText = "正在加载..."
	chatSearchFailedText  = "会话列表读取失败。"
	chatSearchMoreText    = "显示更多"
	chatSearchPrevText    = "上一页"
	chatSearchRetryText   = "重试"
	chatSearchUnnamedText = "未命名会话"

	// chatSearchScopeNote 是这次检索的口径：只覆盖已加载的目录条目。
	chatSearchScopeNote = "仅搜索已加载的会话"
)

// chatSearchGroup 是结果分组，顺序即渲染顺序（G-A2-08：置顶在上、最近在下）。
type chatSearchGroup int

const (
	chatGroupPinned chatSearchGroup = iota
	chatGroupRecent
	chatGroupCount
)

var chatSearchGroupLabels = [chatGroupCount]string{"置顶聊天", "最近聊天"}

// chatSearchPhase 是对话框的读态。三个值互斥。
type chatSearchPhase int

const (
	chatSearchReady chatSearchPhase = iota
	chatSearchLoading
	chatSearchFailed
)

// chatSearchHit 是一条结果：会话在目录里的下标、显示名、工作区与命中区间。
type chatSearchHit struct {
	index int
	id    string
	title string
	// place 是工作区显示名（目录的 basename）；空表示服务端没有给工作目录。
	place string
	group chatSearchGroup
	// start / end 是标题里的命中字节区间；-1 表示标题没有命中。
	start, end int
	// placeStart / placeEnd 是工作区名里的命中字节区间。
	placeStart, placeEnd int
}

// chatSearchState 是对话框的全部视图状态（可丢）：查询、已加载页数、选中行、
// 焦点与关闭后要还回去的焦点。
type chatSearchState struct {
	open  bool
	query string
	// page 是已加载的页数（从 1 起）：「显示更多」加一页，「上一页」减一页，
	// 已显示的行不重排（G-A2-07）。
	page     int
	selected int

	// hits 是本帧按查询算出的全部命中；rows 是已加载页里的那几行。
	hits []chatSearchHit
	rows []chatSearchHit

	phase      chatSearchPhase
	failReason string

	focus     bool
	selectAll bool
	// returnFocus 在关闭后把焦点还给时间线。
	returnFocus bool
}

// chatSearchKeys 是窗口级键位：键位取自「设置 › 快捷键」的 search-chat（出厂值 ⌘G），
// 打开对话框；已打开时不重复打开。确认框与命令面板在场时让给它们。
//
// 键位走 App.keybindingHit：改过键之后这里跟着改，不再写死 ⌘G。
func (a *App) chatSearchKeys(c *gui.Context) {
	if a.screen != ScreenMain || a.palette.open || a.sessionSearch.open {
		return
	}
	if !a.keybindingHit(c, "search-chat") {
		return
	}
	a.openChatSearch()
}

// openChatSearch 打开对话框：清查询与分页、把焦点排到输入框。
func (a *App) openChatSearch() {
	s := &a.chatSearch
	s.open = true
	s.query = ""
	s.page = 1
	s.selected = 0
	s.hits, s.rows = nil, nil
	s.focus = true
	s.selectAll = true
	s.returnFocus = false
}

// closeChatSearch 关闭对话框：查询与命中都清掉，再次打开是空的（G-A2-01 的
// 「不记忆上次查询」）；焦点还给时间线。
func (a *App) closeChatSearch() {
	s := &a.chatSearch
	if !s.open {
		return
	}
	s.open = false
	s.query = ""
	s.page = 1
	s.selected = 0
	s.hits, s.rows = nil, nil
	s.phase = chatSearchReady
	s.failReason = ""
	s.focus = false
	s.selectAll = false
	s.returnFocus = true
}

// chatSearchFrame 每帧读一次目录并按查询算命中。
func (a *App) chatSearchFrame(c *gui.Context) {
	s := &a.chatSearch
	if !s.open {
		return
	}
	if a.screen != ScreenMain {
		a.closeChatSearch()
		return
	}
	s.hits, s.phase, s.failReason = a.chatSearchMatches(s.query)
	// 分页位置按命中总数收敛：结果变少时不留在空页上。
	if pages := chatSearchPageCount(len(s.hits)); s.page > pages {
		s.page = pages
	}
	if s.page < 1 {
		s.page = 1
	}
	shown := s.page * chatSearchPageSize
	if shown > len(s.hits) {
		shown = len(s.hits)
	}
	s.rows = s.hits[:shown]
	// 选中行跟着结果走：原来的行还在就保持，消失了就回到第 1 行（G-A2-13 的
	// 「结果刷新后原选中项消失 → 高亮回到第 1 行」）。
	if len(s.rows) == 0 {
		s.selected = 0
	} else if s.selected < 0 || s.selected >= len(s.rows) {
		s.selected = 0
	}
}

// chatSearchPageCount 是总页数：至少一页（空结果也是一页）。
func chatSearchPageCount(total int) int {
	if total <= 0 {
		return 1
	}
	return (total + chatSearchPageSize - 1) / chatSearchPageSize
}

// chatSearchMatches 按查询过滤已加载的目录：置顶在前、其余按目录顺序。
func (a *App) chatSearchMatches(query string) ([]chatSearchHit, chatSearchPhase, string) {
	names, titles, places, phase, reason := a.chatSearchCatalog()
	if phase != chatSearchReady {
		return nil, phase, reason
	}
	q := strings.TrimSpace(query)
	var hits []chatSearchHit
	for i, id := range names {
		title := ""
		if i < len(titles) {
			title = titles[i]
		}
		if strings.TrimSpace(title) == "" {
			title = chatSearchUnnamedText
		}
		place := ""
		if i < len(places) {
			place = workspaceBaseName(places[i].Cwd)
		}
		start, end, placeStart, placeEnd := -1, -1, -1, -1
		if q != "" {
			if spans := searchFindAll(title, q); len(spans) > 0 {
				start, end = spans[0][0], spans[0][1]
			} else if spans := searchFindAll(place, q); len(spans) > 0 {
				placeStart, placeEnd = spans[0][0], spans[0][1]
			} else {
				// 标题与工作区都不含查询：这条不是命中。
				continue
			}
		}
		group := chatGroupRecent
		if a.sidebarView.pinned[id] {
			group = chatGroupPinned
		}
		hits = append(hits, chatSearchHit{
			index: i, id: id, title: title, place: place, group: group,
			start: start, end: end, placeStart: placeStart, placeEnd: placeEnd,
		})
	}
	// 置顶组在前：同组内保持目录顺序（服务端给的活动时间顺序）。
	pinned := make([]chatSearchHit, 0, len(hits))
	recent := make([]chatSearchHit, 0, len(hits))
	for _, hit := range hits {
		if hit.group == chatGroupPinned {
			pinned = append(pinned, hit)
			continue
		}
		recent = append(recent, hit)
	}
	return append(pinned, recent...), chatSearchReady, ""
}

// chatSearchCatalog 读一次目录：显示名、工作区与读态。
//
// 三支互斥（G-A2-01 的异常列：不空白无反馈）：断线 → 失败；连着但目录还没读回
// → 加载中；其余 → 就绪。
func (a *App) chatSearchCatalog() (
	names, titles []string, places []SessionPlaceSnapshot,
	phase chatSearchPhase, reason string,
) {
	if a.source == nil {
		// -demo 与截图：样例目录，不是事实。
		names, _ = sessionSamples(a.state)
		return names, a.sessionTitles(), a.sessionPlaces(), chatSearchReady, ""
	}
	if a.disconnected() {
		return nil, nil, nil, chatSearchFailed, a.liveConnection().Reason
	}
	names = a.source.SessionNames()
	if names == nil {
		// nil 与「读到了、但没有会话」不同：nil 表示目录还没读回。
		return nil, nil, nil, chatSearchLoading, ""
	}
	return names, a.source.SessionTitles(), a.source.SessionPlaces(), chatSearchReady, ""
}

// chatSearchTextKey 接管输入框里的键：↑↓ 选择、Enter 打开、Esc 关闭、
// ⌘1–⌘8（其它平台 Ctrl+1–8）直接打开第 n 条。
func (a *App) chatSearchTextKey(k gui.TextKey) bool {
	s := &a.chatSearch
	if !s.open {
		return false
	}
	switch {
	case k.Mods == 0 && k.Key == gui.KeyEscape:
		a.closeChatSearch()
		return true
	case k.Mods == 0 && k.Key == gui.KeyUp:
		a.moveChatSelection(-1)
		return true
	case k.Mods == 0 && k.Key == gui.KeyDown:
		a.moveChatSelection(1)
		return true
	case k.Mods == 0 && k.Key == gui.KeyEnter:
		a.openSelectedChat()
		return true
	case k.Mods == gui.Cmd && chatSearchDigit(k.Key) > 0:
		if n := chatSearchDigit(k.Key); n <= len(s.rows) {
			a.openChatSearchHit(s.rows[n-1])
		}
		return true
	}
	return false
}

// chatSearchDigit 把数字键换成 1–8；不在这个范围返回 0。
func chatSearchDigit(key gui.Key) int {
	switch key {
	case gui.Key1:
		return 1
	case gui.Key2:
		return 2
	case gui.Key3:
		return 3
	case gui.Key4:
		return 4
	case gui.Key5:
		return 5
	case gui.Key6:
		return 6
	case gui.Key7:
		return 7
	case gui.Key8:
		return 8
	}
	return 0
}

// moveChatSelection 上下移动选中行，越界不跨组回卷（G-A2-13 的「列表只有 1 行时
// ↑↓ 不改变」）。
func (a *App) moveChatSelection(delta int) {
	s := &a.chatSearch
	n := len(s.rows)
	if n == 0 {
		return
	}
	next := s.selected + delta
	if next < 0 {
		next = 0
	}
	if next >= n {
		next = n - 1
	}
	s.selected = next
}

// openSelectedChat 打开当前选中行（Enter / 点击同一落点）。
func (a *App) openSelectedChat() {
	s := &a.chatSearch
	if s.selected < 0 || s.selected >= len(s.rows) {
		return
	}
	a.openChatSearchHit(s.rows[s.selected])
}

// openChatSearchHit 打开一条结果：会话标识先换成左栏的行标识，再交给既有的
// 「选中会话」通道（连着 Source 时由它发 OpenSession，不另开协议通道）。
func (a *App) openChatSearchHit(hit chatSearchHit) {
	a.selectedSession = fmt.Sprintf("s%d", hit.index+1)
	a.screen = ScreenMain
	a.closeChatSearch()
}

// chatSearchDialog 画对话框本体：模态、背景压暗、Esc / 点遮罩关闭。
func (a *App) chatSearchDialog(c *gui.Context) {
	s := &a.chatSearch
	if !s.open || a.screen != ScreenMain {
		return
	}
	t := c.Theme()
	open := true
	// G-A2-01 搜索聊天对话框（模态，背景被遮罩压暗）
	dialog := modalShell(gui.Modal(c, &open, func() {
		// G-A2-02 标题 + 描述（只作可读名与说明，屏上不写可见行）
		panel := gui.Column(c).Width(chatSearchWidth).Role(gui.RoleDialog).
			Label(chatSearchTitle).Description(chatSearchDescription).
			Background(t.Surface).Border(strokeHairline, t.Border).
			Radius(radiusCard).Clip().Shadow(0, 8, 24, 0, colorShadow.in(t))
		panel.Children(func() {
			a.chatSearchInputRow(c)
			a.chatSearchResults(c)
		})
	}))
	if dialog.Dismissed() {
		a.closeChatSearch()
	}
}

// chatSearchInputRow 是输入行（44）：放大镜 + 输入框，没有右上关闭按钮。
func (a *App) chatSearchInputRow(c *gui.Context) {
	t := c.Theme()
	s := &a.chatSearch
	// 输入行自带一档面差（surfacePressed）：输入行与结果区靠面差分开，不画分隔线
	// （UI-18、X14）。
	gui.Row(c).Shrink(0).Height(chatSearchInputHeight).Padding(space1, space3).
		Gap(space2).AlignItems(gui.Center).Background(t.SurfacePressed).Children(func() {
		// G-A2-03 搜索图标
		iconGlyph(c, iconSearch, t.TextMuted, "search")
		// G-A2-04 搜索聊天（输入框占位与可读名）
		input := gui.TextInputBase(c, &s.query).
			Placeholder(chatSearchPlaceholder).Label(chatSearchLabel).Grow(1).
			FontSize(typeBody.sizeIn(t)).TextColor(t.Text).
			OnTextKey(a.chatSearchTextKey)
		if s.focus {
			input = input.Focus()
			s.focus = false
		}
		if s.selectAll {
			input.SetTextSelection(0, len([]rune(s.query)))
			s.selectAll = false
		}
	})
}

// chatSearchResults 是结果区：读态、分组标题、结果行、翻页行与口径行。
// G-A2-05 加载中 / G-A2-08 分组标题（置顶聊天 / 最近聊天）/
// G-A2-06 上一页 / G-A2-07 显示更多
func (a *App) chatSearchResults(c *gui.Context) {
	t := c.Theme()
	s := &a.chatSearch
	gui.Column(c).Shrink(0).Children(func() {
		switch s.phase {
		case chatSearchLoading:
			// G-A2-05 正在加载...
			gui.Row(c).Shrink(0).Height(rowRegular).Padding(space2, space3).
				Gap(space2).AlignItems(gui.Center).Label(chatSearchLoadingText).
				Role(gui.RoleStatus).Children(func() {
				gui.Spinner(c).Label("正在加载搜索聊天")
				typeSecondary.style(t, gui.Text(c, chatSearchLoadingText)).
					TextColor(t.TextMuted).SingleLine()
			})
			return
		case chatSearchFailed:
			a.chatSearchFailedRow(c, s.failReason)
			return
		}
		if len(s.hits) == 0 {
			a.chatSearchEmptyRow(c)
			return
		}
		if s.page > 1 {
			// G-A2-06 上一页（第 2 页之后才出现）
			a.chatSearchTextRow(c, chatSearchPrevText, "回到上一页结果", func() {
				s.page--
				s.selected = 0
			})
		}
		index := 0
		lastGroup := chatSearchGroup(-1)
		for _, hit := range s.rows {
			if hit.group != lastGroup {
				// G-A2-08 分组标题（置顶聊天 / 最近聊天）
				lastGroup = hit.group
				chatSearchGroupHeader(c, chatSearchGroupLabels[hit.group])
			}
			a.chatSearchHitRow(c, index, hit)
			index++
		}
		if s.page*chatSearchPageSize < len(s.hits) {
			// G-A2-07 显示更多（还有下一页才出现）
			a.chatSearchTextRow(c, chatSearchMoreText, "显示下一页结果", func() {
				s.page++
			})
		}
	})
	// 口径行：只在连着 Source 时写（样例目录不是事实，不能拿来宣称覆盖范围）。
	if a.wired() {
		// G-A2-05 仅搜索已加载的会话
		gui.Row(c).Shrink(0).Height(chatSearchNoteHeight).Padding(0, space3).
			AlignItems(gui.Center).Children(func() {
			typeMicro.style(t, gui.Text(c, chatSearchScopeNote)).TextColor(t.TextMuted).
				SingleLine()
		})
	}
}

// chatSearchFailedRow 是读失败：原因原文 + 重试（重连接口），不停在加载中。
func (a *App) chatSearchFailedRow(c *gui.Context, reason string) {
	t := c.Theme()
	// G-A2-01 会话列表读取失败。
	gui.Row(c).Shrink(0).MinHeight(rowRegular).Padding(space2, space3).
		Gap(space2).AlignItems(gui.Center).Children(func() {
		gui.Column(c).Grow(1).Children(func() {
			typeSecondary.style(t, gui.Text(c, chatSearchFailedText)).TextColor(t.Danger).
				Role(gui.RoleAlert).SingleLine()
			if reason != "" {
				typeMicro.style(t, gui.Text(c, reason)).TextColor(t.Danger).MaxLines(2)
			}
		})
		disabled := a.intents == nil
		label := chatSearchRetryText
		if disabled {
			label = chatSearchRetryText + "（未连接）"
		}
		// G-A2-01 重试
		if gui.Button(c, label).Label(chatSearchRetryText).Disabled(disabled).Clicked() {
			a.dispatch("重新连接", a.intents.Reconnect, nil)
		}
	})
}

// chatSearchEmptyRow 是空态「未找到聊天」：居中、上下留白（G-A2-14）。
func (a *App) chatSearchEmptyRow(c *gui.Context) {
	t := c.Theme()
	// G-A2-14 未找到聊天
	gui.Column(c).Shrink(0).Height(chatSearchEmptyHeight).Center().Label(chatSearchEmptyText).
		Children(func() {
			typeBody.style(t, gui.Text(c, chatSearchEmptyText)).TextColor(t.TextMuted)
		})
}

// chatSearchGroupHeader 是分组标题：12px 次要色，与结果行文字左缘对齐。
func chatSearchGroupHeader(c *gui.Context, label string) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Height(chatSearchHeaderH).Padding(space1, space3).
		AlignItems(gui.Center).Label("结果分组：" + label).Children(func() {
		typeSecondary.style(t, gui.Text(c, label)).TextColor(t.TextMuted).SingleLine()
	})
}

// chatSearchHitRow 是一条结果：图标 + 标题（命中高亮）+ 工作区 + 快捷键提示。
// G-A2-09 主行（会话标题）/ G-A2-10 副行（工作区显示名）/ G-A2-12 快捷键提示 /
// G-A2-13 选中行
func (a *App) chatSearchHitRow(c *gui.Context, index int, hit chatSearchHit) {
	t := c.Theme()
	s := &a.chatSearch
	selected := index == s.selected
	body := hit.title
	if hit.place != "" {
		body += "，" + hit.place
	}
	row := gui.Row(c).Shrink(0).Height(chatSearchRowHeight).Padding(space1, space3).
		Gap(space2).AlignItems(gui.Center).Radius(radiusSmall).
		Label(body).Role(gui.RoleOption).AccessSelected(selected).
		Cursor(gui.CursorPointer)
	if selected {
		row = row.Background(t.SurfaceHover)
	} else if row.Hovered() {
		row = row.Background(t.SurfaceHover)
	}
	row = row.OnClick(func() { a.openChatSearchHit(hit) })
	row.Children(func() {
		a.chatSearchRowIcon(c, hit)
		gui.Column(c).Grow(1).Children(func() {
			// G-A2-09 结果行主行（会话标题，命中高亮）
			if hit.start >= 0 {
				spans := searchSpans(t, hit.title,
					[]searchRange{{start: hit.start, end: hit.end}},
					typeBodyStrong, t.Text)
				gui.RichText(c, spans...).Label(hit.title).SingleLine()
			} else {
				typeBodyStrong.style(t, gui.Text(c, hit.title)).TextColor(t.Text).SingleLine()
			}
			// G-A2-10 结果行副行（工作区显示名；没有工作目录时不出现）
			if hit.place != "" {
				if hit.placeStart >= 0 {
					spans := searchSpans(t, hit.place,
						[]searchRange{{start: hit.placeStart, end: hit.placeEnd}},
						typeMicro, t.TextMuted)
					gui.RichText(c, spans...).Label(hit.place).SingleLine()
				} else {
					typeMicro.style(t, gui.Text(c, hit.place)).TextColor(t.TextMuted).
						SingleLine()
				}
			}
		})
		// G-A2-12 行尾快捷键提示（⌘1–⌘8）
		if hint := chatSearchHint(index); hint != "" {
			typeMicro.style(t, gui.Text(c, hint)).TextColor(t.TextMuted).SingleLine()
		}
	})
	if selected {
		row.ScrollIntoView()
	}
}

// chatSearchRowIcon 是行首 32px 底托里的状态图标：置顶的会话给琥珀星标（读作
// 「已置顶」），其余给默认的对话图标（不单独播报）。
//
// 参考产品的这一位还有「失败 / 加载中 / 未读」三种（G-A2-11a–11c）：那三种要有
// 每一行的读态与未读事实，协议 v3 的会话目录只有标题 / 工作区 / 时间，没有未读
// 标记、也没有逐行的加载状态，所以这里不画——按 G-A2-11e 的兜底落到默认图标，
// 而不是编一个状态出来。
func (a *App) chatSearchRowIcon(c *gui.Context, hit chatSearchHit) {
	t := c.Theme()
	box := gui.Box(c).Width(chatSearchRowHeight - 2*space1).Height(chatSearchRowHeight - 2*space1).
		Shrink(0).Center().Radius(radiusSmall).Background(t.SurfacePressed)
	if hit.group == chatGroupPinned {
		// G-A2-11d 置顶星标（底托内、琥珀色）
		box = box.Label("已置顶").Tooltip("已置顶")
		box.DrawOver(func(p *gui.Painter, r gui.Rect) {
			paintIcon(p, iconPin, r, t.Warning)
		})
		return
	}
	// G-A2-11e 默认图标（不播报额外状态）
	box = box.Label("会话状态图标").AccessHidden(true)
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		paintIcon(p, iconChat, r, t.TextMuted)
	})
}

// chatSearchHint 是第 index 条的快捷键提示；超过 8 条不给提示。
func chatSearchHint(index int) string {
	if index < 0 || index >= chatSearchPageSize {
		return ""
	}
	if runtime.GOOS == "darwin" {
		return fmt.Sprintf("⌘%d", index+1)
	}
	return fmt.Sprintf("Ctrl+%d", index+1)
}

// chatSearchTextRow 是整行的文字按钮（上一页 / 显示更多）。
// G-A2-06 上一页 / G-A2-07 显示更多（整行文字按钮、32 高）
func (a *App) chatSearchTextRow(c *gui.Context, text, label string, run func()) {
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Height(rowCompact).Padding(0, space3).
		Radius(radiusSmall).Label(label).Cursor(gui.CursorPointer)
	if b.Hovered() {
		b = b.Background(t.SurfaceHover)
	}
	if b.Clicked() {
		run()
	}
	b.Children(func() {
		typeSecondary.style(t, gui.Text(c, text)).TextColor(t.TextMuted).SingleLine()
	})
}
