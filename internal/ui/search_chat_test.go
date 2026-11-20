package ui

import (
	"strconv"
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 搜索聊天对话框（requirements-global §2.2，G-A2-01–G-A2-14）的行为测试：每一步
// 都从屏幕上的入口做起（⌘G、左栏「搜索」、输入框、箭头、回车、Esc），断言落在
// 看得见的结果与视图状态上。

// openChatSearchForTest 按 ⌘G 打开对话框并等一帧（输入框拿到焦点）。
func openChatSearchForTest(t *testing.T, tt *gui.Tester) {
	t.Helper()
	tt.Key(gui.Cmd, gui.KeyG)
	tt.Frame()
}

// chatSearchFor 在对话框里键入一段查询并等命中落地。
func chatSearchFor(t *testing.T, tt *gui.Tester, query string) {
	t.Helper()
	if err := tt.Click(chatSearchLabel); err != nil {
		t.Fatalf("点不到搜索聊天输入框：%v", err)
	}
	tt.Command("selectAll")
	tt.Type(query)
	tt.Frame()
}

// chatSearchWiredSource 是一份带若干会话的目录读数：s1（置顶）、s2…s10。
func chatSearchWiredSource() *fakeSource {
	names := make([]string, 0, 10)
	titles := make([]string, 0, 10)
	places := make([]SessionPlaceSnapshot, 0, 10)
	titleList := []string{
		"修复发送路径", "评审排队的实现", "资产图标重画", "参考产品对照", "旧实现拆解",
		"会话目录分页", "文件树读数", "终端粘贴", "设置快捷键", "导出会话",
	}
	for i := 0; i < 10; i++ {
		names = append(names, "s"+string(rune('1'+i)))
		titles = append(titles, titleList[i])
		places = append(places, SessionPlaceSnapshot{Cwd: "/work/project-" + string(rune('a'+i))})
	}
	return &fakeSource{
		conn:   Connection{Phase: "connected"},
		names:  names,
		titles: titles,
		places: places,
	}
}

// ⌘G 打开模态对话框：role=dialog、输入框拿焦点、底部写明这次检索的口径（G-A2-01、
// G-A2-04、G-A2-05）。
func TestChatSearchOpensWithCommandG(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: chatSearchWiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find(chatSearchTitle); ok {
		t.Fatal("没按 ⌘G 时不应有搜索聊天对话框")
	}

	openChatSearchForTest(t, tt)
	if !app.chatSearch.open {
		t.Fatalf("⌘G 没有打开对话框：draft=%q", app.draft)
	}
	if _, ok := tt.AccessByRole(gui.RoleDialog, chatSearchTitle); !ok {
		t.Fatalf("对话框没有以 role=dialog 暴露：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if !tt.HasText(chatSearchPlaceholder) {
		t.Fatalf("输入框缺占位文案 %q：\n%s", chatSearchPlaceholder,
			strings.Join(tt.Texts(), "\n"))
	}
	if active, _ := tt.TextInputActive(); !active {
		t.Fatal("对话框打开后输入框没有拿焦点")
	}
	// 口径行：只在已有的目录条目上过滤，不把结果说成全量历史。
	if !tt.HasText(chatSearchScopeNote) {
		t.Fatalf("缺口径行 %q：\n%s", chatSearchScopeNote, strings.Join(tt.Texts(), "\n"))
	}
	// 空查询给全部会话：每组最多 8 条，第一页 8 行。
	if len(app.chatSearch.rows) != chatSearchPageSize {
		t.Fatalf("空查询的结果行 = %d，想要第一页 %d 条", len(app.chatSearch.rows),
			chatSearchPageSize)
	}
}

// 已有对话框时再按 ⌘G 不新开第二个，查询保留（G-A2-01 的「对话框计数恒为 1」）。
func TestChatSearchOpenTwiceKeepsOneDialog(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: chatSearchWiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	openChatSearchForTest(t, tt)
	chatSearchFor(t, tt, "排队")
	openChatSearchForTest(t, tt)
	if n := countSearchLabels(tt.Texts(), chatSearchTitle); n != 1 {
		t.Fatalf("对话框数量 = %d，想要 1", n)
	}
	if app.chatSearch.query != "排队" {
		t.Fatalf("再按 ⌘G 改动了查询：%q", app.chatSearch.query)
	}
}

// 输入即过滤；置顶会话排在「置顶聊天」组，其余在「最近聊天」组；没有命中的组
// 不画标题（G-A2-08、G-A2-09、G-A2-10）。
func TestChatSearchGroupsAndFilters(t *testing.T) {
	src := chatSearchWiredSource()
	app := NewApp(Options{State: StateDefault, Source: src})
	app.sidebarView.pinned = map[string]bool{"s2": true}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	openChatSearchForTest(t, tt)

	// 空查询：两组都在，置顶的一条在最前面。
	if !tt.HasText("结果分组：" + chatSearchGroupLabels[chatGroupPinned]) {
		t.Fatalf("缺置顶分组标题：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if !tt.HasText("结果分组：" + chatSearchGroupLabels[chatGroupRecent]) {
		t.Fatalf("缺最近分组标题：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if got := app.chatSearch.rows[0].title; got != "评审排队的实现" {
		t.Fatalf("第一行 = %q，想要置顶的那条", got)
	}

	// 过滤：只剩置顶那一条，最近组随之消失。
	chatSearchFor(t, tt, "排队")
	if len(app.chatSearch.rows) != 1 || app.chatSearch.rows[0].title != "评审排队的实现" {
		t.Fatalf("过滤后 = %v", app.chatSearch.rows)
	}
	if tt.HasText("结果分组：" + chatSearchGroupLabels[chatGroupRecent]) {
		t.Fatal("没有命中的组不该画标题")
	}
	// 命中摘要：标题里的命中区间落在查询词上。
	hit := app.chatSearch.rows[0]
	if hit.start < 0 || hit.title[hit.start:hit.end] != "排队" {
		t.Fatalf("命中区间 = (%d,%d) 在 %q", hit.start, hit.end, hit.title)
	}
	// 副行是工作区显示名（目录末段），不出现绝对路径。
	if hit.place != "project-b" {
		t.Fatalf("副行 = %q，想要工作区显示名", hit.place)
	}
	if tt.HasText("/work/project-b") {
		t.Fatal("副行画出了绝对路径")
	}
}

// ↑↓ 移动选中行，回车打开当前会话并关闭对话框（G-A2-13、G-A2-04）。
func TestChatSearchKeyboardOpensSession(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: chatSearchWiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	openChatSearchForTest(t, tt)
	if app.chatSearch.selected != 0 {
		t.Fatalf("默认选中行 = %d，想要第 0 行", app.chatSearch.selected)
	}
	tt.Key(0, gui.KeyDown)
	tt.Frame()
	if app.chatSearch.selected != 1 {
		t.Fatalf("↓ 之后选中行 = %d，想要 1", app.chatSearch.selected)
	}
	tt.Key(0, gui.KeyUp)
	tt.Frame()
	if app.chatSearch.selected != 0 {
		t.Fatalf("↑ 之后选中行 = %d，想要 0", app.chatSearch.selected)
	}
	// 首行再按 ↑ 停在首行（不跨组回卷）。
	tt.Key(0, gui.KeyUp)
	tt.Frame()
	if app.chatSearch.selected != 0 {
		t.Fatalf("首行按 ↑ 后选中行 = %d，想要 0", app.chatSearch.selected)
	}

	tt.Key(0, gui.KeyDown)
	tt.Frame()
	want := app.chatSearch.rows[1]
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.chatSearch.open {
		t.Fatal("回车后对话框没有关闭")
	}
	// 跳转走的是左栏「选中会话」通道：行标识按目录下标换算。
	if got, wantID := app.selectedSession, "s"+strconv.Itoa(want.index+1); got != wantID {
		t.Fatalf("选中会话 = %q，想要 %q", got, wantID)
	}
}

// ⌘1–⌘8 直接打开第 n 条（G-A2-04、G-A2-12）。
func TestChatSearchNumberShortcutOpensRow(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: chatSearchWiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	openChatSearchForTest(t, tt)
	want := app.chatSearch.rows[2]
	tt.Key(gui.Cmd, gui.Key3)
	tt.Frame()
	if app.chatSearch.open {
		t.Fatal("⌘3 后对话框没有关闭")
	}
	if got, wantID := app.selectedSession, "s"+strconv.Itoa(want.index+1); got != wantID {
		t.Fatalf("⌘3 打开的会话 = %q，想要 %q", got, wantID)
	}
}

// 翻页：第一页 8 条时「显示更多」追加下一页，回退时「上一页」出现（G-A2-06、G-A2-07）。
func TestChatSearchPaging(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: chatSearchWiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	openChatSearchForTest(t, tt)
	if len(app.chatSearch.hits) != 10 {
		t.Fatalf("命中总数 = %d，想要 10", len(app.chatSearch.hits))
	}
	if len(app.chatSearch.rows) != 8 {
		t.Fatalf("第一页行数 = %d，想要 8", len(app.chatSearch.rows))
	}
	if tt.HasText("回到上一页结果") {
		t.Fatal("第 1 页不该有「上一页」")
	}
	if err := tt.Click("显示下一页结果"); err != nil {
		t.Fatalf("点不到「显示更多」：%v", err)
	}
	tt.Frame()
	if len(app.chatSearch.rows) != 10 {
		t.Fatalf("显示更多后行数 = %d，想要 10", len(app.chatSearch.rows))
	}
	if err := tt.Click("回到上一页结果"); err != nil {
		t.Fatalf("点不到「上一页」：%v", err)
	}
	tt.Frame()
	if len(app.chatSearch.rows) != 8 {
		t.Fatalf("上一页后行数 = %d，想要 8", len(app.chatSearch.rows))
	}
	if app.chatSearch.selected != 0 {
		t.Fatalf("回退页后选中行 = %d，想要第 0 行", app.chatSearch.selected)
	}
}

// 三个读态互斥：无命中给空态、目录还没读回给「正在加载...」、断线给原因与重试
// （G-A2-01 的异常列、G-A2-05、G-A2-14）。
func TestChatSearchStates(t *testing.T) {
	t.Run("无命中", func(t *testing.T) {
		app := NewApp(Options{State: StateDefault, Source: chatSearchWiredSource()})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		openChatSearchForTest(t, tt)
		chatSearchFor(t, tt, "zzzzz")
		if !tt.HasText(chatSearchEmptyText) {
			t.Fatalf("无命中没有空态 %q：\n%s", chatSearchEmptyText,
				strings.Join(tt.Texts(), "\n"))
		}
		// 空态句不嵌查询词（G-A2-14 的「查询非空且无命中」）。
		if tt.HasText("zzzzz" + chatSearchEmptyText) {
			t.Fatal("空态句里嵌了查询词")
		}
	})
	t.Run("目录未读回", func(t *testing.T) {
		// names 为 nil：连着 Source，但目录还没读回来。
		src := &fakeSource{conn: Connection{Phase: "connected"}}
		app := NewApp(Options{State: StateDefault, Source: src})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		openChatSearchForTest(t, tt)
		if app.chatSearch.phase != chatSearchLoading {
			t.Fatalf("读态 = %v，想要加载中", app.chatSearch.phase)
		}
		if !tt.HasText(chatSearchLoadingText) {
			t.Fatalf("缺加载文案：\n%s", strings.Join(tt.Texts(), "\n"))
		}
		if tt.HasText(chatSearchEmptyText) {
			t.Fatal("加载中不该同时显示空态")
		}
	})
	t.Run("断线", func(t *testing.T) {
		src := &fakeSource{conn: Connection{Phase: "disconnected", Reason: "连接已关闭"}}
		app := NewApp(Options{State: StateDefault, Source: src})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		openChatSearchForTest(t, tt)
		if app.chatSearch.phase != chatSearchFailed {
			t.Fatalf("读态 = %v，想要失败", app.chatSearch.phase)
		}
		if !tt.HasText(chatSearchFailedText) || !tt.HasText("连接已关闭") {
			t.Fatalf("失败行缺通用句或原因：\n%s", strings.Join(tt.Texts(), "\n"))
		}
		if !tt.HasText(chatSearchRetryText) {
			t.Fatalf("失败行缺「重试」：\n%s", strings.Join(tt.Texts(), "\n"))
		}
	})
}

// Esc 关闭对话框，再次打开不记忆上次查询（G-A2-01、G-A2-04 的「每次打开为空」）。
func TestChatSearchEscapeClosesAndForgetsQuery(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: chatSearchWiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	openChatSearchForTest(t, tt)
	chatSearchFor(t, tt, "排队")
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.chatSearch.open {
		t.Fatal("Esc 没有关闭对话框")
	}
	openChatSearchForTest(t, tt)
	if app.chatSearch.query != "" {
		t.Fatalf("再次打开带回了上次的查询：%q", app.chatSearch.query)
	}
}

// 左栏「搜索」行与 ⌘G 是同一个入口（G-A2-01）。
func TestChatSearchSidebarEntryOpensDialog(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("搜索"); err != nil {
		t.Fatalf("点不到左栏「搜索」：%v", err)
	}
	tt.Frame()
	if !app.chatSearch.open {
		t.Fatalf("左栏「搜索」没有打开对话框：\n%s", strings.Join(tt.Texts(), "\n"))
	}
}

// ⌘G（跨会话的搜索聊天对话框）与 ⌘F（会话内搜索浮层）互不打架：各自开自己那
// 一个，同一时刻不同时开两个；关掉之后另一个照常能开（G-A2-01 与 G-A1-01）。
func TestChatSearchAndSessionSearchDoNotCollide(t *testing.T) {
	src := chatSearchWiredSource()
	// 会话内搜索要有打开的会话才给入口：这份读数补上当前会话。
	src.session = SessionSnapshot{ID: "s1", Title: "修复发送路径"}
	src.hasSess = true
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	// ⌘G 只开对话框，不开会话内搜索。
	openChatSearchForTest(t, tt)
	if !app.chatSearch.open || app.sessionSearch.open {
		t.Fatalf("⌘G 之后 chatSearch=%v sessionSearch=%v，想要只开对话框",
			app.chatSearch.open, app.sessionSearch.open)
	}
	// 对话框开着时 ⌘F 不抢：焦点在对话框里，会话内搜索不出现。
	tt.Key(gui.Cmd, gui.KeyF)
	tt.Frame()
	if app.sessionSearch.open {
		t.Fatal("对话框开着时 ⌘F 又开了会话内搜索")
	}
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.chatSearch.open {
		t.Fatal("Esc 没有关掉搜索聊天对话框")
	}

	// 关掉对话框之后 ⌘F 照旧开会话内搜索。
	tt.Key(gui.Cmd, gui.KeyF)
	tt.Frame()
	if !app.sessionSearch.open {
		t.Fatal("关掉对话框之后 ⌘F 打不开会话内搜索了")
	}
	// 会话内搜索开着时 ⌘G 不开对话框。
	tt.Key(gui.Cmd, gui.KeyG)
	tt.Frame()
	if app.chatSearch.open {
		t.Fatal("会话内搜索开着时 ⌘G 又开了搜索聊天对话框")
	}
}

// 行首状态：置顶的会话在底托里给星标并读作「已置顶」，没有置顶时这一句不在任何
// 地方出现（G-A2-11d、G-A2-11e）。失败 / 加载中 / 未读三种要有逐行读态与未读
// 事实，协议 v3 的会话目录没有，所以不画，按兜底落到默认图标。
func TestChatSearchRowStates(t *testing.T) {
	pinned := NewApp(Options{State: StateDefault, Source: chatSearchWiredSource()})
	pinned.sidebarView.pinned = map[string]bool{"s2": true}
	ptt := gui.NewTester(pinned.View, WindowWidth, WindowHeight)
	openChatSearchForTest(t, ptt)
	if !ptt.HasText("已置顶") {
		t.Fatalf("置顶行缺星标的读屏名：\n%s", strings.Join(ptt.Texts(), "\n"))
	}
	var starRow bool
	for _, row := range pinned.chatSearch.rows {
		if row.group == chatGroupPinned {
			starRow = true
		}
	}
	if !starRow {
		t.Fatal("置顶的会话没有进「置顶聊天」组")
	}

	// 没有置顶：星标与它的读屏名都不出现（默认图标不单独播报）。
	plain := NewApp(Options{State: StateDefault, Source: chatSearchWiredSource()})
	tt := gui.NewTester(plain.View, WindowWidth, WindowHeight)
	openChatSearchForTest(t, tt)
	if tt.HasText("已置顶") {
		t.Fatalf("没有置顶却出现了星标文案：\n%s", strings.Join(tt.Texts(), "\n"))
	}
}

// chatSearchFocusByTab 在对话框里按 Tab 直到某个可读名拿到焦点：键盘能走到哪
// 个控件，就证明那个控件在焦点链上（分页按钮不是只能点）。
func chatSearchFocusByTab(tt *gui.Tester, name string, limit int) bool {
	for i := 0; i < limit; i++ {
		if info, ok := tt.Access(name); ok && info.Focused {
			return true
		}
		tt.Key(0, gui.KeyTab)
		tt.Frame()
	}
	info, ok := tt.Access(name)
	return ok && info.Focused
}

// 翻页按钮的键盘通道：Tab 走到「显示更多」/「上一页」，Enter 与点击同一落点
// （G-A2-06、G-A2-07 的键盘列）。
func TestChatSearchPagingByKeyboard(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: chatSearchWiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	openChatSearchForTest(t, tt)

	if !chatSearchFocusByTab(tt, "显示下一页结果", 4) {
		t.Fatalf("Tab 走不到「显示更多」：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.chatSearch.page != 2 || len(app.chatSearch.rows) != 10 {
		t.Fatalf("Enter 翻页后 page=%d rows=%d，想要第 2 页 10 行",
			app.chatSearch.page, len(app.chatSearch.rows))
	}

	if !chatSearchFocusByTab(tt, "回到上一页结果", 4) {
		t.Fatalf("Tab 走不到「上一页」：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.chatSearch.page != 1 || len(app.chatSearch.rows) != 8 {
		t.Fatalf("Enter 回退后 page=%d rows=%d，想要第 1 页 8 行",
			app.chatSearch.page, len(app.chatSearch.rows))
	}
	if app.chatSearch.selected != 0 {
		t.Fatalf("回退页后选中行 = %d，想要第 0 行", app.chatSearch.selected)
	}
}

// Home / End 把选中行跳到首条 / 末条，输入框里的查询不变（与 ↑↓ 同一族键位，
// G-A2-13 的选中行迁移）。
func TestChatSearchHomeEndSelection(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: chatSearchWiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	openChatSearchForTest(t, tt)

	tt.Key(0, gui.KeyEnd)
	tt.Frame()
	if got, want := app.chatSearch.selected, len(app.chatSearch.rows)-1; got != want {
		t.Fatalf("End 之后选中行 = %d，想要末条 %d", got, want)
	}
	tt.Key(0, gui.KeyHome)
	tt.Frame()
	if app.chatSearch.selected != 0 {
		t.Fatalf("Home 之后选中行 = %d，想要首条 0", app.chatSearch.selected)
	}
	if app.chatSearch.query != "" {
		t.Fatalf("Home / End 改了查询词：%q", app.chatSearch.query)
	}
}

// 目录读回后加载行消失、结果行出现：加载态不残留成常驻文案（G-A2-05 的
// 「结果返回后整行消失」），空目录则换成空态而不是停在加载中（G-A2-14）。
func TestChatSearchLoadingSettlesIntoResults(t *testing.T) {
	// names 为 nil：连着 Source，目录还没读回。
	src := &fakeSource{conn: Connection{Phase: "connected"}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	openChatSearchForTest(t, tt)
	if !tt.HasText(chatSearchLoadingText) {
		t.Fatalf("加载中没有「正在加载...」：\n%s", strings.Join(tt.Texts(), "\n"))
	}

	// 目录读回（一条会话）：加载行让位给结果行。
	src.names = []string{"s1"}
	src.titles = []string{"修复发送路径"}
	tt.Frame()
	if app.chatSearch.phase != chatSearchReady {
		t.Fatalf("目录读回后读态 = %v，想要就绪", app.chatSearch.phase)
	}
	if tt.HasText(chatSearchLoadingText) {
		t.Fatalf("目录读回后加载行还在：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if !tt.HasText("修复发送路径") {
		t.Fatalf("目录读回后没有结果行：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if len(app.chatSearch.rows) != 1 {
		t.Fatalf("目录读回后行数 = %d，想要 1", len(app.chatSearch.rows))
	}

	// 目录读回但没有会话：空态出场，而不是停在加载中。
	src.names, src.titles = []string{}, []string{}
	tt.Frame()
	if tt.HasText(chatSearchLoadingText) {
		t.Fatal("空目录仍停在加载行")
	}
	if !tt.HasText(chatSearchEmptyText) {
		t.Fatalf("空目录没有空态 %q：\n%s", chatSearchEmptyText, strings.Join(tt.Texts(), "\n"))
	}

	// 点空态没有动作：不关对话框、不打开任何会话（G-A2-14 的交互列）。
	before := app.selectedSession
	if err := tt.Click(chatSearchEmptyText); err == nil {
		tt.Frame()
	}
	if !app.chatSearch.open {
		t.Fatal("点空态把对话框关掉了")
	}
	if app.selectedSession != before {
		t.Fatalf("点空态换了选中会话：%q → %q", before, app.selectedSession)
	}
}
