package ui

import (
	"fmt"
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是会话内搜索（G-A1、G-A5）的行为测试：每一步都从屏幕上的入口做起
// （⌘F、输入、箭头、关闭键、点浮层外），断言落在看得见的结果与视图状态上。

// openSessionSearchForTest 打开搜索浮层并等一帧（输入框拿到焦点）。
func openSessionSearchForTest(t *testing.T, tt *gui.Tester) {
	t.Helper()
	tt.Key(gui.Cmd, gui.KeyF)
	tt.Frame()
}

// searchFor 在浮层里键入一段查询并等命中落地。
func searchFor(t *testing.T, tt *gui.Tester, query string) {
	t.Helper()
	if err := tt.Click(sessionSearchLabel); err != nil {
		t.Fatalf("点不到搜索输入框：%v", err)
	}
	// 全选后键入：每段查询都替换上一段，不叠加。
	tt.Command("selectAll")
	tt.Type(query)
	tt.Frame()
}

// TestSessionSearchOpensOnCommandF：会话里按 ⌘F 出现浮层（A1-01），输入框
// 拿焦点、有可读名与占位文案（A1-03、A1-04）。
func TestSessionSearchOpensOnCommandF(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if _, ok := tt.Find("会话内搜索"); ok {
		t.Fatal("没按 ⌘F 时不应有搜索浮层")
	}

	openSessionSearchForTest(t, tt)
	if !app.sessionSearch.open {
		t.Fatal("⌘F 没有打开搜索浮层")
	}
	if _, ok := tt.Find("会话内搜索"); !ok {
		t.Fatalf("屏幕上没有浮层：%v", tt.Texts())
	}
	// 查询为空：只有输入行，没有结果行（R-GS-055）。
	if tt.HasText("个结果") {
		t.Fatal("查询为空时不应有结果行")
	}
	if !tt.HasText(sessionSearchPlaceholder) && !tt.HasText(sessionSearchLabel) {
		t.Fatalf("输入框缺占位或可读名：%v", tt.Texts())
	}
	active, _ := tt.TextInputActive()
	if !active {
		t.Fatal("浮层打开后输入框没有拿焦点")
	}
}

// TestSessionSearchCommandFAgainKeepsOneOverlay：已有浮层时再按 ⌘F 不新开第二个
// （R-GS-003）。
func TestSessionSearchCommandFAgainKeepsOneOverlay(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	tt.Key(gui.Cmd, gui.KeyF)
	tt.Frame()
	if n := countSearchLabels(tt.Texts(), "会话内搜索"); n != 1 {
		t.Fatalf("浮层数量 = %d，期望 1", n)
	}
	if !app.sessionSearch.open {
		t.Fatal("再按 ⌘F 把浮层关掉了")
	}
	if active, _ := tt.TextInputActive(); !active {
		t.Fatal("再按 ⌘F 后焦点没有回到输入框")
	}
	if app.sessionSearch.query != "排队" {
		t.Fatalf("再按 ⌘F 改动了查询：%q", app.sessionSearch.query)
	}
}

// TestSessionSearchMatchesTimelineAndCounts：输入即时命中，结果行给出
// 「{当前} / {总数} 个结果」（R-GS-030、R-GS-079）。
func TestSessionSearchMatchesTimelineAndCounts(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")

	if got := len(app.sessionSearch.hits); got == 0 {
		t.Fatalf("「排队」在样例时间线里应有命中（%v）", unitKeys(app.sessionSearch.units))
	}
	want := fmt.Sprintf("%d / %d 个结果", app.sessionSearch.current+1, len(app.sessionSearch.hits))
	if !tt.HasText(want) {
		t.Fatalf("结果计数应为 %q：%v", want, tt.Texts())
	}
	// 命中来自多种文本：用户消息、助手正文、思考块与工具行都在语料里。
	scopes := map[searchScope]bool{}
	for _, unit := range app.sessionSearch.units {
		scopes[unit.scope] = true
	}
	for _, scope := range []searchScope{searchScopeUser, searchScopeAssistant} {
		if !scopes[scope] {
			t.Fatalf("语料里缺 %v 这一类的文本：%v", scope, unitKeys(app.sessionSearch.units))
		}
	}
}

// TestSessionSearchEnterMovesBetweenHits：回车到下一个、Shift+回车到上一个，
// 计数随之更新（A1-03、A1-07、A1-08）。
func TestSessionSearchEnterMovesBetweenHits(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	n := len(app.sessionSearch.hits)
	if n < 2 {
		t.Fatalf("样例里「排队」的命中数 = %d，测试需要至少 2 条", n)
	}

	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.sessionSearch.current != 1 {
		t.Fatalf("回车后当前项 = %d，期望 1", app.sessionSearch.current)
	}
	if !tt.HasText(fmt.Sprintf("2 / %d 个结果", n)) {
		t.Fatalf("回车后计数没更新：%v", tt.Texts())
	}

	tt.Key(gui.Shift, gui.KeyEnter)
	tt.Frame()
	if app.sessionSearch.current != 0 {
		t.Fatalf("Shift+回车后当前项 = %d，期望 0", app.sessionSearch.current)
	}

	// 在第一条上再往前：回卷到最后一条（R-GS-064）。
	tt.Key(gui.Shift, gui.KeyEnter)
	tt.Frame()
	if app.sessionSearch.current != n-1 {
		t.Fatalf("首条再往前应回卷到 %d，实际 %d", n-1, app.sessionSearch.current)
	}
	if !tt.HasText(fmt.Sprintf("%d / %d 个结果", n, n)) {
		t.Fatalf("回卷后计数不对：%v", tt.Texts())
	}
}

// TestSessionSearchArrowButtons：上下箭头按钮与回车等价，0 命中时禁用
// （A1-07、A1-08、R-GS-065、R-GS-073）。
func TestSessionSearchArrowButtons(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")

	if err := tt.Click("下一个搜索结果"); err != nil {
		t.Fatalf("点不到下一个：%v", err)
	}
	tt.Frame()
	if app.sessionSearch.current != 1 {
		t.Fatalf("点下一个后当前项 = %d，期望 1", app.sessionSearch.current)
	}
	if err := tt.Click("上一个搜索结果"); err != nil {
		t.Fatalf("点不到上一个：%v", err)
	}
	tt.Frame()
	if app.sessionSearch.current != 0 {
		t.Fatalf("点上一个后当前项 = %d，期望 0", app.sessionSearch.current)
	}

	searchFor(t, tt, "这一段查询一定没有命中")
	if len(app.sessionSearch.hits) != 0 {
		t.Fatalf("该查询不应有命中：%d", len(app.sessionSearch.hits))
	}
	for _, label := range []string{"上一个搜索结果", "下一个搜索结果"} {
		info, ok := tt.Access(label)
		if !ok {
			t.Fatalf("找不到按钮 %q", label)
		}
		if !info.Disabled {
			t.Fatalf("0 命中时 %q 应禁用", label)
		}
	}
}

// TestSessionSearchNoResults：无命中时计数为「0 个结果」并给空态文案
// （R-GS-061、R-GS-078）。
func TestSessionSearchNoResults(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "这段查询保证没有命中")

	if !tt.HasText("0 个结果") {
		t.Fatalf("无命中时计数应为「0 个结果」：%v", tt.Texts())
	}
	if !tt.HasText(sessionSearchEmptyText) {
		t.Fatalf("无命中时应给空态文案 %q：%v", sessionSearchEmptyText, tt.Texts())
	}
}

// TestSessionSearchEscapeAndCloseButton：Esc 与关闭键都收起浮层、清空查询与
// 命中，焦点回到时间线（R-GS-034、R-GS-047）。
func TestSessionSearchEscapeAndCloseButton(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	if len(app.sessionSearch.hits) == 0 {
		t.Fatal("打样查询应有命中")
	}

	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.sessionSearch.open {
		t.Fatal("Esc 没有关闭浮层")
	}
	if app.sessionSearch.query != "" || len(app.sessionSearch.hits) != 0 {
		t.Fatal("关闭后查询与命中没有清空")
	}
	if tt.HasText("个结果") {
		t.Fatalf("关闭后计数仍在屏幕上：%v", tt.Texts())
	}

	// 关闭键：再开一次，点「关闭搜索」。
	openSessionSearchForTest(t, tt)
	if err := tt.Click("关闭搜索"); err != nil {
		t.Fatalf("点不到关闭键：%v", err)
	}
	tt.Frame()
	if app.sessionSearch.open {
		t.Fatal("关闭键没有关闭浮层")
	}
	active, _ := tt.TextInputActive()
	if active {
		t.Fatal("关闭后焦点仍停在输入框里")
	}
}

// TestSessionSearchOutsideClickCloses：点浮层外关闭（R-GS-002）。
func TestSessionSearchOutsideClickCloses(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	if !app.sessionSearch.open {
		t.Fatal("⌘F 没有打开浮层")
	}
	// 左栏中部：浮层之外。
	tt.ClickAt(120, 400)
	tt.Frame()
	if app.sessionSearch.open {
		t.Fatal("点浮层外没有关闭")
	}
}

// TestSessionSearchReopenClearsQuery：关闭再打开不记忆上次查询（R-GS-009）。
func TestSessionSearchReopenClearsQuery(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	tt.Key(0, gui.KeyEscape)
	tt.Frame()

	app.timelineSelected = ""
	openSessionSearchForTest(t, tt)
	if app.sessionSearch.query != "" {
		t.Fatalf("重开后查询 = %q，期望空", app.sessionSearch.query)
	}
}

// TestSessionSearchPrefillsSelection：打开时时间线有选中文本就预填并全选
// （R-GS-028）。
func TestSessionSearchPrefillsSelection(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.timelineSelected = "看看这次的失败"
	openSessionSearchForTest(t, tt)
	if app.sessionSearch.query != "看看这次的失败" {
		t.Fatalf("预填 = %q", app.sessionSearch.query)
	}
	if len(app.sessionSearch.hits) == 0 {
		t.Fatal("预填的查询应有命中")
	}
}

// TestSessionSearchDoesNotStealComposer：搜索开着时键入落在搜索框，输入区草稿
// 不动（「搜索时输入区快捷键不抢焦点」）。
func TestSessionSearchDoesNotStealComposer(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.draft = "草稿还在"
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	if app.draft != "草稿还在" {
		t.Fatalf("搜索改动输入区草稿：%q", app.draft)
	}
	if !strings.Contains(app.sessionSearch.query, "排队") {
		t.Fatalf("查询没有落到搜索框：%q", app.sessionSearch.query)
	}
	// ⌘B 这类界面键仍可用，但不会把焦点挪进输入区。
	tt.Key(gui.Cmd, gui.KeyB)
	tt.Frame()
	if !app.sidebarCollapsed {
		t.Fatal("搜索开着时 ⌘B 失效")
	}
	active, _ := tt.TextInputActive()
	if !active {
		t.Fatal("⌘B 之后焦点离开了搜索框")
	}
}

// TestSessionSearchThinkingToggle：思考块的命中可开关（需求 5）。
func TestSessionSearchThinkingToggle(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	// 只有思考块里出现的词。
	searchFor(t, tt, "确认排队判定覆盖了草稿")
	withThinking := len(app.sessionSearch.hits)
	if withThinking == 0 {
		t.Fatal("思考块里应有命中")
	}
	if !tt.HasText("含思考块") {
		t.Fatalf("缺少范围开关：%v", tt.Texts())
	}
	if err := tt.Click("搜索范围：含思考块"); err != nil {
		t.Fatalf("点不到范围开关：%v", err)
	}
	tt.Frame()
	if len(app.sessionSearch.hits) != 0 {
		t.Fatalf("关掉思考块后仍有命中：%d", len(app.sessionSearch.hits))
	}
	if !tt.HasText("0 个结果") {
		t.Fatalf("关掉思考块后计数应为 0：%v", tt.Texts())
	}
	if err := tt.Click("搜索范围：不含思考块"); err != nil {
		t.Fatalf("点不到范围开关：%v", err)
	}
	tt.Frame()
	if got := len(app.sessionSearch.hits); got != withThinking {
		t.Fatalf("再开思考块后命中 = %d，期望 %d", got, withThinking)
	}
}

// TestSessionSearchExpandsFoldedRegions：命中落在折叠的思考块与工具详情里时，
// 自动展开（R-GS-378）。
func TestSessionSearchExpandsFoldedRegions(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if !*app.blockOf("think:0", false) {
		// 默认收起着，命中应当把它打开。
	}
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "先读输入区与时间线")
	if len(app.sessionSearch.hits) == 0 {
		t.Fatal("思考块里应有命中")
	}
	if !*app.blockOf("think:0", false) {
		t.Fatal("命中所在的思考块没有展开")
	}

	// 工具输出：先命中输出原文，工具详情要展开。
	searchFor(t, tt, "FAIL")
	if len(app.sessionSearch.hits) == 0 {
		t.Fatalf("工具输出里应有命中：%v", tt.Texts())
	}
	if !*app.blockOf("tool:0:3", false) {
		t.Fatal("命中所在的工具详情没有展开")
	}

	// 收起浮层：搜索期间展开的折叠位复原。
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if *app.blockOf("think:0", false) || *app.blockOf("tool:0:3", false) {
		t.Fatal("关闭后折叠位没有复原")
	}
}

// TestSessionSearchRestoresScroll：跳转会把当前命中滚进视口，关闭后回到打开前
// 的滚动位置（需求 6、R-GS-375）。
func TestSessionSearchRestoresScroll(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	// 把样例拉长，让时间线确实可滚。
	base := app.rounds
	for i := 0; i < 12; i++ {
		app.rounds = append(app.rounds, base...)
	}
	tt := gui.NewTester(app.View, WindowWidth, 520)
	app.feed.Y = 0
	tt.Frame()

	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "已经改成排队")
	if len(app.sessionSearch.hits) == 0 {
		t.Fatal("末尾一轮应有命中")
	}
	if app.sessionSearch.current != 0 {
		t.Fatalf("当前项 = %d，期望 0", app.sessionSearch.current)
	}
	for i := 0; i < 4; i++ {
		tt.Frame()
	}
	if app.feed.Y <= 0 {
		t.Fatalf("当前命中没有滚进视口（feed.Y = %.1f）", app.feed.Y)
	}

	saved := app.feed.Y
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.feed.Y != 0 {
		t.Fatalf("关闭后滚动位置 = %.1f，期望回到 0（跳转时为 %.1f）", app.feed.Y, saved)
	}
}

// TestSessionSearchHighlightSpans：命中片段有底色；当前项用更深的橙底，其余用
// 黄底（A5-01、A5-02）。
func TestSessionSearchHighlightSpans(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	if len(app.sessionSearch.hits) < 2 {
		t.Fatalf("需要至少 2 条命中，实际 %d", len(app.sessionSearch.hits))
	}

	c := theme(false, gui.Preferences{})
	key := app.sessionSearch.units[app.sessionSearch.hits[0].unit].key
	text := app.sessionSearch.units[app.sessionSearch.hits[0].unit].text
	ranges := app.searchRanges(key)
	if len(ranges) == 0 {
		t.Fatalf("单元 %s 没有命中区间", key)
	}
	spans := searchSpans(c, text, ranges, typeBody, c.Text)
	var current, other int
	for _, span := range spans {
		got := span.Background
		switch got {
		case colorSearchHitCurrent.in(c):
			current++
		case colorSearchHit.in(c):
			other++
		}
	}
	if current+other != len(ranges) {
		t.Fatalf("高亮 span = %d（当前 %d / 其余 %d），命中 %d", current+other, current, other, len(ranges))
	}
	if current != 1 {
		t.Fatalf("当前项高亮 span = %d，期望 1", current)
	}
}

// TestSearchMarkedMarkdown：Markdown 正文的命中包 <mark>，当前项再包 <u>；
// 代码块、行内代码与链接目标里的命中不包（不破坏原有结构）。
func TestSearchMarkedMarkdown(t *testing.T) {
	src := "先看结论：**排队不再覆盖队列**。\n\n`排队的实现`见[说明](https://example/排队)。\n\n```\n排队\n```\n"
	at := strings.Index(src, "**排队")
	ranges := []searchRange{
		{start: at + 2, end: at + 2 + len("排队"), current: true},
		{start: strings.Index(src, "排队的实现"), end: strings.Index(src, "排队的实现") + len("排队")},
		{start: strings.Index(src, "example/排队"), end: strings.Index(src, "example/排队") + len("排队")},
		{start: strings.LastIndex(src, "排队"), end: strings.LastIndex(src, "排队") + len("排队")},
	}
	got := searchMarkedMarkdown(src, ranges)
	if !strings.Contains(got, "**<mark><u>排队</u></mark>不再覆盖队列**") {
		t.Fatalf("粗体内的命中没有按行内标记包好：%q", got)
	}
	if strings.Count(got, "<mark>") != 1 {
		t.Fatalf("只该包正文里的那一条，实际 %q", got)
	}
}

// TestSearchFindAll：大小写不敏感、按字节给区间、不重叠。
func TestSearchFindAll(t *testing.T) {
	text := "Go Test GO go"
	got := searchFindAll(text, "go")
	if len(got) != 3 {
		t.Fatalf("命中数 = %d，期望 3（%v）", len(got), got)
	}
	for _, span := range got {
		if text[span[0]:span[1]] != "Go" && text[span[0]:span[1]] != "GO" && text[span[0]:span[1]] != "go" {
			t.Fatalf("区间切出来的文本不对：%q", text[span[0]:span[1]])
		}
	}
	if got := searchFindAll("中文排队中", "排队"); len(got) != 1 ||
		"中文排队中"[got[0][0]:got[0][1]] != "排队" {
		t.Fatalf("中文命中不对：%v", got)
	}
	if got := searchFindAll("abc", ""); got != nil {
		t.Fatalf("空查询不应有命中：%v", got)
	}
}

// TestSessionSearchLiveRowsSearchable：连着事实时，用户行、助手行、思考行、
// 工具行的参数与输出都进语料，键与渲染端一致。
func TestSessionSearchLiveRowsSearchable(t *testing.T) {
	src := wiredSource()
	src.session.Rows = []RowSnapshot{
		{Kind: RowUser, Text: "看看 internal/ui 的排布"},
		{Kind: RowThinking, Text: "先读 composer.go，再改"},
		{Kind: RowToolGroup, Children: []RowSnapshot{
			{Kind: RowTool, Title: "读取", Args: "internal/ui/composer.go", Text: "package ui"},
		}},
		{Kind: RowAssistant, Text: "internal/ui 已经改好。"},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "internal")
	if len(app.sessionSearch.hits) == 0 {
		t.Fatalf("事实行里应有命中：%v", app.sessionSearch.units)
	}
	hasTool, hasAssistant := false, false
	for _, h := range app.sessionSearch.hits {
		switch app.sessionSearch.units[h.unit].scope {
		case searchScopeTool:
			hasTool = true
		case searchScopeAssistant:
			hasAssistant = true
		}
	}
	if !hasTool && !hasAssistant {
		t.Fatalf("命中没有落到工具行或助手行：%v", unitKeys(app.sessionSearch.units))
	}
}

// countSearchLabels 数屏幕上逐字等于 text 的标签个数。
func countSearchLabels(texts []string, text string) int {
	n := 0
	for _, got := range texts {
		if got == text {
			n++
		}
	}
	return n
}

// unitKeys 是语料的 key 列表（失败时打印用）。
func unitKeys(units []searchUnit) []string {
	out := make([]string, 0, len(units))
	for _, unit := range units {
		out = append(out, unit.key)
	}
	return out
}

// TestSessionSearchMarkdownHighlightKeepsText：助手正文的命中走行内标记，屏幕
// 上的文字不变、标记本身不上屏（A5-01、R-GS-368）。
func TestSessionSearchMarkdownHighlightKeepsText(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队不再覆盖队列")

	hitAssistant := false
	for _, h := range app.sessionSearch.hits {
		if app.sessionSearch.units[h.unit].scope == searchScopeAssistant {
			hitAssistant = true
		}
	}
	if !hitAssistant {
		t.Fatal("富文本回复里应有命中")
	}
	if !tt.HasText("排队不再覆盖队列") {
		t.Fatalf("命中之后正文文字变了：%v", tt.Texts())
	}
	for _, text := range tt.Texts() {
		if strings.Contains(text, "<mark") || strings.Contains(text, "</u>") {
			t.Fatalf("行内标记漏到屏幕上：%q", text)
		}
	}
}
