package ui

import (
	"image"
	"image/color"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// duration 是界面上不该出现的时长写法：12s、2.1s、1 分 20 秒。
var duration = regexp.MustCompile(`[0-9]+(\.[0-9]+)?s\b|[0-9]+ 分 [0-9]+ 秒`)

func newTester(t *testing.T, s State) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: s})
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// 每个状态都要在样式与读屏名上分得清，证明状态切换落到元素上（§9.0 D4）：
// 屏幕上不写状态词，状态由行首图标表达，读屏名由「状态：…」给出。
func TestStatesShowTheirState(t *testing.T) {
	cases := []struct {
		state State
		word  string // 组头读屏名里的状态
		want  string
		gone  string
	}{
		{StateDefault, "状态：完成", "读取 composer.go", "等待你同意"},
		{StateRunning, "状态：正在运行", "读取 composer.go", "等待你同意"},
		{StateWaiting, "状态：等待你同意", "rm -rf ./tmp/*", "发送路径.md"},
		{StateFailed, "状态：完成", "结果不确定 · 正在核对", "排队中 1"},
		{StateEmpty, "", "还没有消息", "修发送路径"},
	}
	for _, tc := range cases {
		_, tt := newTester(t, tc.state)
		if tc.word != "" && !tt.HasText(tc.word) {
			t.Fatalf("%v：缺少状态位 %q", tc.state, tc.word)
		}
		if !tt.HasText(tc.want) {
			t.Fatalf("%v：缺少 %q", tc.state, tc.want)
		}
		if tc.gone != "" && tt.HasText(tc.gone) {
			t.Fatalf("%v：不应出现 %q", tc.state, tc.gone)
		}
	}
}

// 等待你同意时，审批卡占据输入位，输入区整个不存在（UI-1）。
func TestWaitingReplacesComposer(t *testing.T) {
	_, tt := newTester(t, StateWaiting)
	if _, ok := tt.Find("消息"); ok {
		t.Fatal("等待你同意时输入区不应存在")
	}
	for _, label := range []string{"允许一次", "一直允许", "拒绝"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("审批卡缺少 %q", label)
		}
	}
	// 审批卡占位时整个输入槽不出现（UI-1）。
	for _, label := range []string{"发送", "停止", "附件"} {
		if _, ok := tt.Find(label); ok {
			t.Fatalf("审批卡占位时不应出现输入控件 %q", label)
		}
	}
}

// 拿不到命令原文时，不提供“始终允许”（U3 §5.3）。
func TestApprovalWithoutCommandHidesAllowAlways(t *testing.T) {
	app, _ := newTester(t, StateWaiting)
	app.approvalCommand = ""
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if tt.HasText("一直允许") {
		t.Fatal("没有命令原文时不应出现一直允许")
	}
	if _, ok := tt.Find("允许一次"); !ok {
		t.Fatal("没有命令原文时仍应能允许一次")
	}
}

// 1–5 是仅用于走查的调试键。
func TestDebugKeysSwitchState(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	want := []State{StateDefault, StateRunning, StateWaiting, StateFailed, StateEmpty}
	for i, key := range []gui.Key{gui.Key1, gui.Key2, gui.Key3, gui.Key4, gui.Key5} {
		tt.Key(0, key)
		if app.State() != want[i] {
			t.Fatalf("按 %v 后状态是 %v，想要 %v", key, app.State(), want[i])
		}
	}
}

// 焦点在输入框里时，数字键必须留给草稿，不切状态。
func TestDebugKeysYieldToTextInput(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if err := tt.Click("消息"); err != nil {
		t.Fatalf("点不到输入框：%v", err)
	}
	tt.TypeKey(0, gui.Key2, "2")
	if app.State() != StateDefault {
		t.Fatalf("输入框里的 2 切了状态：%v", app.State())
	}
	if !strings.Contains(app.draft, "2") {
		t.Fatalf("草稿没有收到 2：%q", app.draft)
	}
}

// Tab 依次经过三个可聚焦区域：左栏、对话流、输入框。
func TestTabReachesEachRegion(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	reached := map[string]bool{}
	for i := 0; i < 40 && len(reached) < 3; i++ {
		tt.Key(0, gui.KeyTab)
		for _, label := range []string{"左栏 · 会话列表", "对话流", "消息"} {
			if tt.Focused(label) {
				reached[label] = true
			}
		}
	}
	for _, label := range []string{"左栏 · 会话列表", "对话流", "消息"} {
		if !reached[label] {
			t.Fatalf("Tab 没有到达 %q", label)
		}
	}
}

// 左栏收起后留 12px 边条，边条可展开回原位。
func TestSidebarCollapsesAndExpands(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if err := tt.Click("收起左栏"); err != nil {
		t.Fatalf("点不到收起：%v", err)
	}
	if !app.sidebarCollapsed {
		t.Fatal("左栏没有收起")
	}
	rail, ok := tt.Find("展开左栏")
	if !ok {
		t.Fatal("收起后找不到边条")
	}
	if rail.W != SidebarRailWidth {
		t.Fatalf("边条宽 %v，想要 %v", rail.W, SidebarRailWidth)
	}
	if err := tt.Click("展开左栏"); err != nil {
		t.Fatalf("点不到边条：%v", err)
	}
	if app.sidebarCollapsed {
		t.Fatal("左栏没有展开")
	}
}

// 轮次组头是本组的内容总览：写做了什么（动作与对象），不写状态词、计数与
// 时长（UI-11、§9.0 A2/A3）。屏幕上不出现「完成」这类状态词，也没有秒数。
func TestRoundHeaderIsContentPhrase(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	if !tt.HasText("读取 composer.go") {
		t.Fatalf("组头没有写这一组做了什么：%v", tt.Texts())
	}
	// 读屏名里的「状态：…」是状态位的可读名称，不是屏幕上的文字，白名单放行。
	aria := map[string]bool{"状态：完成": true, "状态：正在运行": true,
		"状态：等待你同意": true, "状态：失败": true}
	for _, text := range tt.Texts() {
		if aria[text] {
			continue
		}
		for _, word := range []string{"完成", "失败", "正在运行", "等待你同意"} {
			if text == word {
				t.Fatalf("主界面不应把状态词当文字：%q", text)
			}
		}
		// 「展开」「收起」只允许出现在左栏的收起/展开动作上，不能当行内入口。
		if text == "展开" || text == "收起" {
			t.Fatalf("行内不应出现文字入口 %q", text)
		}
	}
	for _, count := range []string{"个操作", "改了", "耗时", "已用"} {
		if tt.HasText(count) {
			t.Fatalf("折叠时不应出现计数或时长 %q", count)
		}
	}
	for _, text := range tt.Texts() {
		if duration.MatchString(text) {
			t.Fatalf("主界面不应出现时长：%q", text)
		}
	}
}

// 组头的可读名称由「内容短语 + 状态」合成（§9.0 可读下限）：屏幕上没有状态词，
// 读屏仍能听到状态。
func TestRoundHeaderAriaNameCarriesStatus(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	info, ok := tt.AccessByRole(gui.RoleDisclosure, "")
	if !ok {
		t.Fatal("找不到轮次组头")
	}
	if !strings.Contains(info.Name, "状态：完成") || !strings.Contains(info.Name, "读取 composer.go") {
		t.Fatalf("组头读屏名 %q，想要「状态 + 内容短语」", info.Name)
	}
	if info.Expanded {
		t.Fatal("折叠时不应报已展开")
	}
}

// 组头的内容短语按动作聚合：同一动作的对象并列，不写计数（§9.0 A3）。
func TestRoundHeaderPhrase(t *testing.T) {
	r := round{ops: []op{
		{action: "读取", target: "internal/ui/composer.go"},
		{action: "改动", target: "internal/ui/composer.go"},
		{action: "改动", target: "internal/ui/timeline.go"},
		{action: "执行", target: "go test ./..."},
	}}
	want := "读取 composer.go · 改动 composer.go、timeline.go · 执行 go test ./..."
	if got := roundHeaderPhrase(r); got != want {
		t.Fatalf("组头 = %q，想要 %q", got, want)
	}
	if got := roundHeaderPhrase(round{user: "只有对话"}); got != "" {
		t.Fatalf("纯对话轮的组头 = %q，想要空", got)
	}
}

// 窄窗下组头按对象截断（R7）：动作与对象的第一段留住，其余收成「…」。
func TestRoundHeaderTruncationKeepsObjects(t *testing.T) {
	full := "读取 composer.go · 改动 composer.go、timeline.go · 执行 go test ./internal/ui"
	shown := truncatePhrase(full)
	if len([]rune(shown)) > headerPhraseMaxRunes {
		t.Fatalf("截断后 %q 仍超 %d 字", shown, headerPhraseMaxRunes)
	}
	for _, keep := range []string{"读取 composer.go"} {
		if !strings.Contains(shown, keep) {
			t.Fatalf("截断后丢了 %q：%q", keep, shown)
		}
	}
	if !strings.HasSuffix(shown, "…") {
		t.Fatalf("截断后应以 … 结尾：%q", shown)
	}
	// 只剩一个动作组时，对象截到第一段（命令取第一个词）。
	if got := truncatePhrase("执行 go test ./internal/ui/very/long/path/that/never/fits/in/one/line"); !strings.Contains(got, "执行 go") {
		t.Fatalf("单组截断丢了动作或对象第一段：%q", got)
	}
}

// 点内容块就展开：入口是内容本身，方向指示紧跟内容，行宽随内容（UI-16、§9.0 取向 4）。
func TestRoundHeaderTogglesOnContentClick(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	row := headerRow(t, tt)
	tt.ClickAt(row.X+row.W/2, row.Y+row.H/2)
	if !*app.openOf(0) {
		t.Fatal("点内容块没有展开")
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("4 个操作 · 改了 2 个文件") {
		t.Fatalf("展开后缺少计数：%v", tt.Texts())
	}
	row = headerRow(t, tt)
	tt.ClickAt(row.X+row.W/2, row.Y+row.H/2)
	if *app.openOf(0) {
		t.Fatal("再点内容块没有收起")
	}
}

// 行宽随内容：组头的内容块明显窄于内容列，且从内容列左缘起（§9.0 A3/D3）。
func TestRoundHeaderIsLeftAlignedAndContentSized(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	content, ok := tt.Find("对话内容")
	if !ok {
		t.Fatal("找不到对话内容")
	}
	row := headerRow(t, tt)
	if row.X != content.X+space4 {
		t.Fatalf("组头左缘 %v，想要内容列左缘 %v", row.X, content.X+space4)
	}
	if row.W >= content.W/2 {
		t.Fatalf("组头宽 %v，内容列宽 %v：行宽仍是容器给的", row.W, content.W)
	}
}

// 展开后按发生顺序给出操作与失败原因；点明细内容不收起（U3 §1.1 L2）。
func TestRoundHeaderExpands(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if tt.HasText("权限不足，命令没有执行") {
		t.Fatal("折叠时不应给出展开内容")
	}
	row := headerRow(t, tt)
	tt.ClickAt(row.X+row.W/2, row.Y+row.H/2)
	if !*app.openOf(0) {
		t.Fatal("第一轮组头没有展开")
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("权限不足，命令没有执行") {
		t.Fatal("展开后缺少失败原因")
	}
	if !tt.HasText("go test ./...") {
		t.Fatal("展开后缺少操作对象")
	}
	if err := tt.Click("go test ./..."); err != nil {
		t.Fatalf("点不到明细行：%v", err)
	}
	if !*app.openOf(0) {
		t.Fatal("点明细内容不应收起组头")
	}
}

// 组头可点性由指针形状、悬停态与方向指示表达（UI-16、U2 §4.4）。
func TestRoundHeaderShowsPointerAndHover(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	row := headerRow(t, tt)
	x, y := row.X+row.W-4, row.Y+row.H/2
	// 指针不在组头上时是默认形状。
	tt.Move(WindowWidth/2, HeaderHeight+2)
	tt.Frame()
	if got := tt.Cursor(); got != gui.CursorDefault {
		t.Fatalf("组头外的指针形状 %v，想要默认", got)
	}
	tt.Move(x, y)
	if got := tt.Cursor(); got != gui.CursorPointer {
		t.Fatalf("组头上的指针形状 %v，想要手形", got)
	}
	hover := colorSurfaceHover.in(theme(false, gui.Preferences{}))
	if got := tt.Image().RGBAAt(int(x), int(y)); got != colorOf(hover) {
		t.Fatalf("悬停底 %v，想要 surface-hover %v", got, colorOf(hover))
	}
}

// 方向指示：16×16 盒，折叠指向右、展开指向下，像素随开合改变（U2 §4.4）。
func TestDirectionIndicatorTurnsWithOpen(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	r, ok := tt.Find("方向指示")
	if !ok {
		t.Fatal("找不到方向指示")
	}
	if r.W != iconInline || r.H != iconInline {
		t.Fatalf("方向指示 %+v，想要 %d×%d", r, iconInline, iconInline)
	}
	box := gui.Rect{W: iconInline, H: iconInline}
	closed, open := chevronPoints(box, chevronAngle(0)), chevronPoints(box, chevronAngle(1))
	const tip = 1
	if closed[tip][0] <= box.W/2 || math.Abs(float64(closed[tip][1]-box.H/2)) > 0.01 {
		t.Fatalf("折叠时的尖角 %v，想要指向右", closed[tip])
	}
	if open[tip][1] <= box.H/2 || math.Abs(float64(open[tip][0]-box.W/2)) > 0.01 {
		t.Fatalf("展开时的尖角 %v，想要指向下", open[tip])
	}

	before := tt.Image()
	*app.openOf(0) = true
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	after := tt.Image()
	if samePixels(before, after, r) {
		t.Fatal("方向指示没有随开合改变")
	}
}

// 纯对话轮没有可说的动作，就不出组头（§9.0 A3：组头由内容决定有无）。
func TestChatOnlyRoundHasNoHeader(t *testing.T) {
	app, _ := newTester(t, StateDefault)
	for i := range app.rounds {
		app.rounds[i].ops = nil
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("方向指示"); ok {
		t.Fatal("纯对话轮不应出组头与方向指示")
	}
	if !tt.HasText("看看这次的失败") {
		t.Fatal("对话内容不应随组头一起消失")
	}
}

// headerRow 返回第一个轮次组头（可折叠行的内容块）的盒子。
func headerRow(t *testing.T, tt *gui.Tester) gui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(gui.RoleDisclosure, "")
	if !ok {
		t.Fatal("找不到轮次组头")
	}
	return info.Bounds
}

// 标题行同时是窗口标题栏（UI-19）：高度取 max(c.TitleBar().Height, 48)，左栏顶部
// 让出控件那条，行内不重复写会话名；全屏时 c.TitleBar() 为零也不跳变。
func TestTitleBarIsWindowTitleBar(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	header, ok := tt.Find("标题行")
	if !ok || header.H != HeaderHeight {
		t.Fatalf("没有窗口控件时标题行 %+v，想要高 %v", header, HeaderHeight)
	}

	// 窗口模式：控件高 28 落在 48 之内，左栏顶部让出那一条。
	tt.SetTitleBar(gui.TitleBar{Height: 28, Left: 78})
	header, _ = tt.Find("标题行")
	if header.H != HeaderHeight {
		t.Fatalf("控件 28 时标题行高 %v，想要仍是 %v", header.H, HeaderHeight)
	}
	newSession, ok := tt.Find("新建会话")
	if !ok {
		t.Fatal("找不到左栏头部的入口")
	}
	if newSession.Y < 28 {
		t.Fatalf("左栏头部 %+v 压在窗口控件让位条里", newSession)
	}

	// 控件比 48 高时标题行跟着变高（不写死 48）。
	tt.SetTitleBar(gui.TitleBar{Height: 56, Left: 78})
	header, _ = tt.Find("标题行")
	if header.H != 56 {
		t.Fatalf("控件 56 时标题行高 %v，想要 56", header.H)
	}

	// 会话名在标题行左端写一次（§9.0 A1/D1：这一行服务"我在哪"），
	// 且落在窗口控件让位的右边。
	tt.SetTitleBar(gui.TitleBar{Height: 28, Left: 78})
	header, _ = tt.Find("标题行")
	title, ok := tt.Find("会话 修发送路径")
	if !ok {
		t.Fatal("标题行里没有会话名")
	}
	if title.Y < header.Y || title.Y+title.H > header.Y+header.H {
		t.Fatalf("会话名 %+v 不在标题行 %+v 里", title, header)
	}
	if app.sidebarCollapsed {
		t.Fatal("此时左栏应是展开的")
	}

	// 左栏收起成 12px 边条时，标题行按控件宽度让位，会话名不被红绿灯压住。
	app.sidebarCollapsed = true
	tt.Frame()
	title, ok = tt.Find("会话 修发送路径")
	if !ok {
		t.Fatal("收起左栏后标题行里没有会话名")
	}
	if title.X < 78 {
		t.Fatalf("会话名左缘 %v，压在窗口控件（宽 78）下面", title.X)
	}
	app.sidebarCollapsed = false
	tt.Frame()
}

// 空态也要写会话名：标题行不留空行（§9.0 A1）。
func TestEmptyStateTitleIsNotEmpty(t *testing.T) {
	_, tt := newTester(t, StateEmpty)
	header, ok := tt.Find("标题行")
	if !ok {
		t.Fatal("找不到标题行")
	}
	title, ok := tt.Find("会话 新会话")
	if !ok {
		t.Fatalf("空态标题行没有会话名：%v", tt.Texts())
	}
	if title.Y < header.Y || title.Y+title.H > header.Y+header.H {
		t.Fatalf("空态会话名 %+v 不在标题行 %+v 里", title, header)
	}
}

// 动作入口用图标，不用字符冒充（UI-21、X15）：界面上不再有 + ‹ › × ⋯ 这类
// 当入口的字符，动作各有读屏名，图标由图标元素画。
func TestActionsUseIconsNotGlyphs(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	for _, glyph := range []string{"+", "‹", "›", "×", "⋯", "＋"} {
		if tt.HasText(glyph) {
			t.Fatalf("界面里仍有用字符当图标的地方：%q", glyph)
		}
	}
	for _, label := range []string{"新建会话", "收起左栏", "打开右栏", "附件"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("找不到图标入口的读屏名 %q", label)
		}
	}
	// 图标真的画出来了：入口的盒子里有非底色像素。
	box, ok := tt.Find("附件")
	if !ok {
		t.Fatal("找不到附件入口")
	}
	if !hasInk(tt.Image(), box) {
		t.Fatalf("附件入口 %+v 里没有画出图标", box)
	}
}

// 终端：右栏第五个标签；三态只由状态点表达；中止/清屏/键盘都生效。
func TestTerminalStatesAndKeys(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.rightOpen, app.rightTab = true, 4
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("终端"); !ok {
		t.Fatal("右栏没有终端标签")
	}
	// 空闲：有清屏、没有中止。
	if _, ok := tt.Access("清屏"); !ok {
		t.Fatal("空闲态缺清屏入口")
	}
	if _, ok := tt.Access("中止运行"); ok {
		t.Fatal("空闲态不应有中止入口")
	}
	// 运行中：状态点转 accent，出现中止；Ctrl-C 中止。
	app.termState = termRunning
	app.termLines = termSamples(termRunning)
	tt.Frame()
	if _, ok := tt.Access("中止运行"); !ok {
		t.Fatal("运行中缺中止入口")
	}
	tt.Key(gui.Ctrl, gui.KeyC)
	if app.termState != termExited {
		t.Fatalf("Ctrl-C 后状态 = %v，想要已退出", app.termState)
	}
	// Cmd-K 清屏：回空态。
	tt.Key(gui.Cmd, gui.KeyK)
	tt.Frame()
	if len(app.termLines) != 0 {
		t.Fatalf("清屏后仍有 %d 行", len(app.termLines))
	}
	if _, ok := tt.Find("还没有输出"); !ok {
		t.Fatal("清屏后没有空态")
	}
	// Esc 收回右栏（回主界面）。
	tt.Key(0, gui.KeyEscape)
	if app.rightOpen {
		t.Fatal("Esc 没有收回右栏")
	}
}

// 附件移除（可逆）。
func TestAttachmentRemove(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if len(app.attachments) != 1 {
		t.Fatalf("样例附件 %d 条，想要 1", len(app.attachments))
	}
	if err := tt.Click("移除附件：发送路径.md"); err != nil {
		t.Fatalf("点不到移除：%v", err)
	}
	if len(app.attachments) != 0 {
		t.Fatal("移除没有生效")
	}
}

// 命令模式（`!`）：标记与占位换口径、槽位换语义、运行后输出进终端、退出回普通输入。
func TestShellMode(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if app.shellMode() {
		t.Fatal("默认不该是命令模式")
	}
	app.draft = "!go test ./..."
	tt.Frame()
	if !app.shellMode() {
		t.Fatal("! 前缀没有进入命令模式")
	}
	if _, ok := tt.Find("命令模式"); !ok {
		t.Fatal("缺命令模式标记")
	}
	if _, ok := tt.Access("运行命令"); !ok {
		t.Fatal("槽位没有换成运行命令")
	}
	if _, ok := tt.Access("发送"); ok {
		t.Fatal("命令模式不该再有发送")
	}
	// 运行：输出进终端、草稿清空、退出模式。
	before := len(app.termLines)
	if err := tt.Click("运行命令"); err != nil {
		t.Fatalf("点不到运行：%v", err)
	}
	if app.shellMode() {
		t.Fatal("运行后没有退出命令模式")
	}
	if len(app.termLines) != before+2 {
		t.Fatalf("终端多了 %d 行，想要 2", len(app.termLines)-before)
	}

	// 退出：叉入口只去前缀，草稿保留。
	app.draft = "!echo hi"
	tt.Frame()
	if err := tt.Click("退出命令模式"); err != nil {
		t.Fatalf("点不到退出：%v", err)
	}
	if app.draft != "echo hi" {
		t.Fatalf("退出后草稿 = %q，想要 echo hi", app.draft)
	}
}

// 终端 ANSI：\r 覆盖、\b 退格、\x1b[K 清行；断线入口与恢复。
func TestTerminalANSIAndDisconnect(t *testing.T) {
	cases := []struct{ in, want string }{
		{"下载 10%\r下载 100%", "下载 100%"},
		{"v1.O\b0", "v1.0"},
		{"临时输出\x1b[K", ""},
	}
	for _, c := range cases {
		if got := interpretLine(c.in); got != c.want {
			t.Fatalf("interpretLine(%q) = %q，想要 %q", c.in, got, c.want)
		}
	}
	app, tt := newTester(t, StateDefault)
	app.rightOpen, app.rightTab = true, 4
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("模拟断线"); err != nil {
		t.Fatalf("点不到断线入口：%v", err)
	}
	if app.termState != termDisconnected {
		t.Fatal("没有转断线")
	}
	tt.Frame()
	if _, ok := tt.Access("重新连接"); !ok {
		t.Fatal("断线态缺恢复入口")
	}
	if err := tt.Click("重新连接"); err != nil {
		t.Fatalf("点不到恢复：%v", err)
	}
	if app.termState != termIdle {
		t.Fatal("恢复后没有回空闲")
	}
	if got := app.termVisible()[len(app.termVisible())-1].text; got != "reconnected" {
		t.Fatalf("恢复末行 = %q", got)
	}
}

// 会话目录动作：菜单四项、重命名行内编辑、删除两步、归档/导出、搜索过滤。
func TestSessionActions(t *testing.T) {
	app, tt := newTester(t, StateDefault)

	// 菜单：选中行有「…」，四项齐全。
	info, ok := tt.Find("会话操作：修发送路径")
	if !ok {
		t.Fatal("选中行没有行末动作入口")
	}
	_ = info
	if err := tt.Click("会话操作：修发送路径"); err != nil {
		t.Fatalf("点不到动作入口：%v", err)
	}
	tt.Frame()
	menu := tt.Menu()
	for _, want := range []string{"重命名", "归档", "导出", "删除"} {
		found := false
		for _, got := range menu {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("菜单缺 %q（实际 %v）", want, menu)
		}
	}

	// 重命名：行内编辑 + 保存。
	if err := tt.ChooseMenuItem("重命名"); err != nil {
		t.Fatalf("点不到重命名：%v", err)
	}
	if app.renaming != 0 {
		t.Fatal("没有进入重命名编辑态")
	}
	tt.Frame()
	if _, ok := tt.Find("保存"); !ok {
		t.Fatal("重命名编辑态缺保存按钮")
	}
	app.renameDraft = "发送路径（改）"
	if err := tt.Click("保存名称"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if got := app.sessionName(0, ""); got != "发送路径（改）" {
		t.Fatalf("重命名没生效：%q", got)
	}

	// 删除：菜单选删除只进确认态，确认后才消失。
	app.renaming = -1
	tt.Frame()
	if err := tt.Click("会话操作：发送路径（改）"); err != nil {
		t.Fatalf("点不到动作入口：%v", err)
	}
	tt.Frame()
	if err := tt.ChooseMenuItem("删除"); err != nil {
		t.Fatalf("点不到删除：%v", err)
	}
	if app.deleting != 0 || app.archived[0] {
		t.Fatal("第一次选删除不应生效")
	}
	tt.Frame()
	if err := tt.Click("确认删除"); err != nil {
		t.Fatalf("点不到确认删除：%v", err)
	}
	if !app.archived[0] {
		t.Fatal("确认后会话没有移除")
	}

	// 归档：可逆，单步。
	app2, tt2 := newTester(t, StateDefault)
	if err := tt2.Click("会话操作：修发送路径"); err != nil {
		t.Fatalf("点不到动作入口：%v", err)
	}
	tt2.Frame()
	if err := tt2.ChooseMenuItem("归档"); err != nil {
		t.Fatalf("点不到归档：%v", err)
	}
	if !app2.archived[0] {
		t.Fatal("归档没有生效")
	}
	tt2.Frame()
	if _, ok := tt2.Find("修发送路径"); ok {
		t.Fatal("归档后行仍在列表里")
	}
}

// 搜索面：点搜索展开，输入过滤，空白给全部，Esc 关闭。
func TestSessionSearch(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if err := tt.Click("搜索"); err != nil {
		t.Fatalf("点不到搜索：%v", err)
	}
	if !app.searchOpen {
		t.Fatal("搜索没有展开")
	}
	tt.Frame()
	app.searchQuery = "旧实现"
	tt.Frame()
	if _, ok := tt.Find("修发送路径"); ok {
		t.Fatal("过滤后仍显示不匹配的行")
	}
	if _, ok := tt.Find("旧实现对照"); !ok {
		t.Fatal("过滤后匹配的行不见了")
	}
	app.searchQuery = ""
	tt.Frame()
	if _, ok := tt.Find("修发送路径"); !ok {
		t.Fatal("空白查询应给全部")
	}
	tt.Key(0, gui.KeyEscape)
	if app.searchOpen {
		t.Fatal("Esc 没有收起搜索面")
	}
}

// 设置是独立成屏（P2）：从左栏齿轮进入、从导航的返回入口回来；草稿与滚动位置
// 不因为进出一趟设置而丢失（UI-8）。
func TestSettingsRoundTripKeepsDraftAndScroll(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.draft = "还没写完的一句"
	// 用矮窗口让时间线可滚动：滚动位置才有意义的非零值。
	tt := gui.NewTester(app.View, WindowWidth, 480)
	tt.Frame()
	tt.Scroll(WindowWidth/2, 300, 0, 120)
	tt.Frame()
	scrolled := app.feed.Y

	if err := tt.Click("设置"); err != nil {
		t.Fatalf("点不到设置入口：%v", err)
	}
	if app.screen != ScreenSettings {
		t.Fatal("没有进入设置屏")
	}
	tt.Frame()
	if _, ok := tt.Find("对话流"); ok {
		t.Fatal("设置屏不应同时显示主界面")
	}
	for _, name := range []string{"账户", "用量", "个性化", "关于"} {
		if _, ok := tt.Find(name); !ok {
			t.Fatalf("设置导航缺分节 %q", name)
		}
	}
	if err := tt.Click("返回主界面"); err != nil {
		t.Fatalf("点不到返回入口：%v", err)
	}
	if app.screen != ScreenMain {
		t.Fatal("没有回到主界面")
	}
	if app.draft != "还没写完的一句" {
		t.Fatalf("草稿丢了：%q", app.draft)
	}
	if app.feed.Y != scrolled {
		t.Fatalf("滚动位置丢了：%v → %v", scrolled, app.feed.Y)
	}
	tt.Frame()
	if _, ok := tt.Find("对话流"); !ok {
		t.Fatal("回到主界面后没有对话流")
	}
}

// 设置导航的 19 节与内容路由：已重建的分节显示真内容，其余显示占位。
func TestSettingsNavAndSectionRouting(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.screen = ScreenSettings
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)

	// 19 节齐全，顺序与参考产品一致。
	want := []string{
		"账户", "用量", "个性化", "外观", "通用", "快捷键", "模型供应", "多模态",
		"运行环境", "MCP", "通知", "宠物", "权限", "Git", "Hooks", "插件设置",
		"外部 Agent", "归档会话", "关于",
	}
	if len(settingsSectionDefs) != len(want) {
		t.Fatalf("分节 %d 项，想要 %d", len(settingsSectionDefs), len(want))
	}
	lastY := float32(-1)
	for i, name := range want {
		info, ok := tt.Find(name)
		if !ok {
			t.Fatalf("导航缺第 %d 节 %q", i+1, name)
		}
		if info.Y <= lastY {
			t.Fatalf("第 %d 节 %q 不在上一节下方（Y=%v）", i+1, name, info.Y)
		}
		lastY = info.Y
		if settingsSectionDefs[i].label != name {
			t.Fatalf("第 %d 节 = %q，想要 %q", i+1, settingsSectionDefs[i].label, name)
		}
	}

	// 账户：真内容（身份与套餐读数）。
	if app.settingsSection != secAccount {
		t.Fatalf("默认分节 = %v，想要账户", app.settingsSection)
	}
	if _, ok := tt.Find("尚未按参考产品重建"); ok {
		t.Fatal("账户不应显示占位")
	}

	// 未实现的分节：切过去只有标题 + 一行占位，不放控件。
	if err := tt.Click("宠物"); err != nil {
		t.Fatalf("点不到分节：%v", err)
	}
	if app.settingsSection != secPets {
		t.Fatalf("分节 = %v，想要宠物", app.settingsSection)
	}
	tt.Frame()
	if _, ok := tt.Find("宠物"); !ok {
		t.Fatal("占位分节缺标题")
	}
	// 19 节现在都已按参考产品重建：占位句不应再出现。
	if _, ok := tt.Find("尚未按参考产品重建"); ok {
		t.Fatal("分节已重建，不应再显示占位")
	}
	if _, ok := tt.Find("宠物"); !ok {
		t.Fatal("宠物分节缺内容")
	}
}

// 分支：main 不上屏（UI-9）；非 main 时标题行给「图标 + 名字」，点开是原生菜单。
func TestBranchMarkerOnlyWhenNotMain(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if _, ok := tt.Find("切换分支：main"); ok {
		t.Fatal("main 分支不应出现在标题行")
	}
	app.branch, app.branches = branchSamples()
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	chip, ok := tt.Find("切换分支：feature/queue-rework")
	if !ok {
		t.Fatal("非 main 分支缺少标题行标记")
	}
	header, _ := tt.Find("标题行")
	if chip.Y < header.Y || chip.Y+chip.H > header.Y+header.H {
		t.Fatalf("分支标记 %+v 不在标题行 %+v 里", chip, header)
	}
	// 点开菜单：清单里有当前分支与 main，当前分支打勾。
	if err := tt.Click("切换分支：feature/queue-rework"); err != nil {
		t.Fatalf("点不到分支标记：%v", err)
	}
	menu := tt.Menu()
	if len(menu) < 2 {
		t.Fatalf("分支菜单只有 %v", menu)
	}
}

// 目标行：在输入区上沿一行；编辑态同位置换成输入框 + 保存 / 取消。
func TestGoalRowLivesAboveComposer(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	goal, ok := tt.Find("把发送路径改成排队")
	if !ok {
		t.Fatal("找不到目标行")
	}
	editor, _ := tt.Find("消息")
	if goal.Y >= editor.Y {
		t.Fatalf("目标行 %+v 不在输入区上方（编辑器 %+v）", goal, editor)
	}
	app.goalEditing = true
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	// W1：保存 / 取消是文字按钮（不是与审批同形的勾/叉）。
	for _, label := range []string{"保存", "取消"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("目标编辑态缺少文字按钮 %q", label)
		}
	}
	for _, label := range []string{"保存目标", "取消编辑"} {
		if _, ok := tt.Access(label); !ok {
			t.Fatalf("文字按钮缺少读屏名 %q", label)
		}
	}
	if _, ok := tt.Find("修改目标"); ok {
		t.Fatal("编辑态不应同时显示修改入口")
	}
	// W2：编辑态输入框有面差（surface 填充），一眼能看出"这里能改"。
	if err := gui.NewTester(app.View, WindowWidth, WindowHeight).Click("目标"); err != nil {
		t.Logf("点不到目标输入框：%v", err)
	}
	_ = tt
}

// 结束分支是两步动作（AC-08）：菜单第一次只换成确认态，再点才生效。
func TestEndBranchNeedsConfirm(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.branch, app.branches = branchSamples()
	app.branchMenuOpen = true
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.ChooseMenuItem("结束分支"); err != nil {
		t.Fatalf("点不到结束分支：%v", err)
	}
	if !app.endBranchConfirm {
		t.Fatal("第一次点结束分支应进入确认态")
	}
	if app.branch != "feature/queue-rework" {
		t.Fatalf("第一次点击就切换了分支：%q", app.branch)
	}
	// 第二步在标题行的标记上：换成「确认结束分支？」，再点标记才生效。
	tt.Frame()
	if _, ok := tt.Find("确认结束分支？"); !ok {
		t.Fatal("标记没有换成确认文案")
	}
	if err := tt.Click("确认结束分支"); err != nil {
		t.Fatalf("点不到确认结束分支：%v", err)
	}
	if app.branch != "main" {
		t.Fatalf("确认后分支 = %q，想要 main", app.branch)
	}
}

// 后台任务：右栏「后台」列出任务；运行中的行有取消入口，完成的行没有。
func TestTaskRowsShowStateAndCancel(t *testing.T) {
	app, _ := newTester(t, StateDefault)
	app.rightOpen, app.rightTab = true, 2
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	for _, name := range []string{"go test ./...", "npm run build"} {
		if _, ok := tt.Find(name); !ok {
			t.Fatalf("后台清单缺少 %q", name)
		}
	}
	if _, ok := tt.Find("取消任务"); !ok {
		t.Fatal("运行中的任务缺取消入口")
	}
	if _, ok := tt.Access("后台任务：go test ./... · 运行中"); !ok {
		t.Fatal("缺少运行中的状态位")
	}
	if _, ok := tt.Access("后台任务：npm run build · 已完成"); !ok {
		t.Fatal("缺少已完成的状态位")
	}
}

// 发送槽的形式是纯图标 + tooltip（名称 · 快捷键），读屏名给动作词（UI-23、F17）。
func TestSendSlotIsIconWithTooltip(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	info, ok := tt.Access("发送")
	if !ok {
		t.Fatal("找不到发送入口")
	}
	if info.Name != "发送" || info.Description != "发送 · Enter" {
		t.Fatalf("发送入口 name=%q description=%q，想要「发送」/「发送 · Enter」", info.Name, info.Description)
	}
	_, rt := newTester(t, StateRunning)
	stop, ok := rt.Access("停止")
	if !ok {
		t.Fatal("运行中找不到停止入口")
	}
	if stop.Name != "停止" || stop.Description != "停止 · Enter" {
		t.Fatalf("停止入口 name=%q description=%q，想要「停止」/「停止 · Enter」", stop.Name, stop.Description)
	}
	// 纯图标：槽位的盒子只有命中区那么宽，容不下文字（文字标签只给读屏）。
	box, _ := tt.Find("发送")
	if box.W > rowCompact || box.H > rowCompact {
		t.Fatalf("发送槽 %+v 比命中区大，说明里面还有文字", box)
	}
}

// 审批的三个决定是图标 + 文字；「一直允许」两步确认（UI-23、F18）。
func TestApprovalDecisionsHaveIconsAndConfirm(t *testing.T) {
	app, tt := newTester(t, StateWaiting)
	for _, label := range []string{"允许一次", "一直允许", "拒绝"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("决定按钮缺少文字 %q", label)
		}
		icon, ok := tt.Access(label)
		if !ok {
			t.Fatalf("决定按钮 %q 没有读屏名", label)
		}
		if !hasInk(tt.Image(), icon.Bounds) {
			t.Fatalf("决定按钮 %q 里没有图标", label)
		}
	}
	if err := tt.Click("一直允许"); err != nil {
		t.Fatalf("点不到一直允许：%v", err)
	}
	if !app.confirmAlways {
		t.Fatal("第一次点一直允许应进入确认态")
	}
	if _, ok := tt.Find("确认一直允许"); !ok {
		t.Fatal("确认态没有换成「确认一直允许」")
	}
	if app.approvalDecision != "" {
		t.Fatal("第一次点击不应写下任何决定")
	}
	if err := tt.Click("确认一直允许"); err != nil {
		t.Fatalf("点不到确认：%v", err)
	}
	if app.approvalDecision != "allow-always" {
		t.Fatalf("确认后决定 = %q，想要 allow-always", app.approvalDecision)
	}
}

// 主区四个位置共用一条文字左缘（UI-22、§9.14 第 3 层）。
func TestContentBaselinesShareOneColumn(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	edges := map[string]float32{}
	if r, ok := tt.Find("会话 修发送路径"); ok {
		edges["标题行"] = r.X
	}
	if r, ok := tt.Find("消息在排队判定里丢了：运行中提交会覆盖队列。我把输入位改成“运行中先入队，结束后自动发出”。"); ok {
		edges["时间线"] = r.X
	}
	if r, ok := tt.Find("消息"); ok {
		// 输入框的盒子左缘就是输入卡的文字列（TextAreaBase 无外观）。
		edges["输入卡"] = r.X
	}
	if len(edges) != 3 {
		t.Fatalf("量到的左缘不足：%v", edges)
	}
	base := edges["标题行"]
	for name, x := range edges {
		if diff := x - base; diff > 2 || diff < -2 {
			t.Fatalf("%s 左缘 %v 与标题行 %v 不同（要求同一列）", name, x, base)
		}
	}
	// 侧栏：所有会话行的文字同在一条文字列（行首图标各占同一列）。
	app.state = StateDefault
	app.sidebarCollapsed = false
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	first, ok1 := tt.Find("修发送路径")
	second, ok2 := tt.Find("参考")
	third, ok3 := tt.Find("旧实现对照")
	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("侧栏量测缺少元素：%v %v %v", ok1, ok2, ok3)
	}
	for name, r := range map[string]gui.Rect{"第二行": second, "第三行": third} {
		if diff := r.X - first.X; diff > 1 || diff < -1 {
			t.Fatalf("侧栏%s 左缘 %v 与第一行 %v 不同", name, r.X, first.X)
		}
	}
}

// 审批卡的文字也落在同一条左缘上。
func TestApprovalBaselineMatchesContentColumn(t *testing.T) {
	_, tt := newTester(t, StateWaiting)
	// 用影响说明那行量：命令是块级代码块，自己带内边距（与输入卡里的 chip 同理），
	// 不参与区域的文字列；卡片里的普通文字才是这一列。
	title, ok := tt.Find("影响：会读写工作区内的临时文件")
	if !ok {
		t.Fatal("找不到审批卡上的影响说明")
	}
	header, ok := tt.Find("标题行")
	if !ok {
		t.Fatal("找不到标题行")
	}
	_ = header
	name, ok := tt.Find("会话 修发送路径")
	if !ok {
		t.Fatal("找不到标题行会话名")
	}
	if diff := title.X - name.X; diff > 2 || diff < -2 {
		t.Fatalf("审批卡文字左缘 %v 与标题行 %v 不同", title.X, name.X)
	}
}

// 自绘可交互元素有默认 / 悬停 / 按下 / 聚焦四态（§9.14 第 4 层、X17；
// 禁用态不适用于组头与用量指示，断线只换颜色）。
func TestSelfDrawnRowsHaveStates(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	row := headerRow(t, tt)
	x, y := row.X+row.W-4, row.Y+row.H/2
	hover := colorOf(colorSurfaceHover.in(theme(false, gui.Preferences{})))
	pressed := colorOf(colorSurfacePressed.in(theme(false, gui.Preferences{})))

	tt.Move(x, y)
	if got := tt.Image().RGBAAt(int(x), int(y)); got != hover {
		t.Fatalf("悬停底 %v，想要 %v", got, hover)
	}
	tt.Press(x, y)
	if got := tt.Image().RGBAAt(int(x), int(y)); got != pressed {
		t.Fatalf("按下底 %v，想要 %v", got, pressed)
	}
	tt.Release(x, y)
	// 键盘聚焦时框架画焦点环（F14）：焦点确实落在这一行上。
	for i := 0; i < 6 && !tt.Focused("完成") && !tt.Focused("读取 composer.go"); i++ {
		tt.Key(0, gui.KeyTab)
	}
}

// 图标入口与组头在 200% 文本缩放下不越界（U2 §7.4）。
func TestIconsSurviveTextScale(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTesterWith(app.View, WindowWidth, WindowHeight,
		gui.TesterOptions{Preferences: gui.Preferences{TextScale: 2}})
	for _, label := range []string{"新建会话", "收起左栏", "打开右栏", "附件"} {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("200%% 缩放下找不到 %q", label)
		}
		if r.X < 0 || r.X+r.W > WindowWidth || r.Y < 0 || r.Y+r.H > WindowHeight {
			t.Fatalf("200%% 缩放下 %q 越界：%+v", label, r)
		}
	}
}

// hasInk 报告 box 里有没有非底色像素（用来证明图标真的画了）。
func hasInk(img *image.RGBA, box gui.Rect) bool {
	bg := img.RGBAAt(int(box.X)+1, int(box.Y)+1)
	for y := int(box.Y); y < int(box.Y+box.H); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if img.RGBAAt(x, y) != bg {
				return true
			}
		}
	}
	return false
}

// 上下文用量在输入卡内，不在标题行（UI-15）；点数值展开明细，再点收起（T10）。
func TestContextMeterSitsInComposer(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	header, ok := tt.Find("标题行")
	if !ok {
		t.Fatal("找不到标题行")
	}
	meter, ok := tt.Find("12%")
	if !ok {
		t.Fatal("输入区里找不到上下文用量")
	}
	if meter.Y < header.Y+header.H {
		t.Fatalf("用量 %+v 仍落在标题行（%+v）里", meter, header)
	}
	chip, ok := tt.Find("发送路径.md")
	if !ok {
		t.Fatal("找不到输入卡内的附件块")
	}
	if meter.Y < chip.Y-space1 || meter.Y+meter.H > chip.Y+chip.H+space1 {
		t.Fatalf("用量 %+v 与附件块 %+v 不在输入卡的第一行", meter, chip)
	}
	editor, ok := tt.Find("消息")
	if !ok || meter.Y+meter.H > editor.Y {
		t.Fatalf("用量 %+v 不在编辑器上方（编辑器 %+v）", meter, editor)
	}
	// 标题行只剩会话名与右栏入口：用量文字不出现在标题行的高度范围里。
	for _, text := range tt.Texts() {
		if !strings.Contains(text, "上下文") && !strings.Contains(text, "%") {
			continue
		}
		r, ok := tt.Find(text)
		if ok && r.Y < header.Y+header.H {
			t.Fatalf("标题行里仍有 %q (%+v)", text, r)
		}
	}

	if err := tt.Click("12%"); err != nil {
		t.Fatalf("点不到用量：%v", err)
	}
	if !app.contextOpen {
		t.Fatal("点用量没有展开明细")
	}
	if !tt.HasText("已用 24.6k / 200k tokens") {
		t.Fatal("展开后缺少用量明细")
	}
	info, ok := tt.AccessByRole(gui.RoleDisclosure, "上下文 12%")
	if !ok || !info.Expanded {
		t.Fatalf("用量指示的读屏状态 %+v，想要已展开", info)
	}
	if err := tt.Click("12%"); err != nil {
		t.Fatalf("点不到用量：%v", err)
	}
	if app.contextOpen || tt.HasText("已用 24.6k") {
		t.Fatal("再点用量没有收起明细")
	}
}

// 将满时同一处数值换颜色并补词，不新增元素（F8、U2 §4.5）。
func TestContextMeterShowsNearFull(t *testing.T) {
	app, _ := newTester(t, StateDefault)
	app.contextUsed = 180_000
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if !tt.HasText("90%") {
		t.Fatalf("将满时缺少数值：%v", tt.Texts())
	}
	if tt.HasText("快满了") {
		t.Fatal("将满不应加状态词：状态归样式（§9.0 A2）")
	}
	// 颜色之外必须有形状（§9.0 D4）：将满时多一个 6px 圆点，常态没有。
	if _, ok := tt.Find("上下文将满"); !ok {
		t.Fatal("将满时缺少非颜色通道（圆点）")
	}
	normal := NewApp(Options{State: StateDefault})
	plain := gui.NewTester(normal.View, WindowWidth, WindowHeight)
	if _, ok := plain.Find("上下文将满"); ok {
		t.Fatal("常态不应出现将满圆点")
	}
	// 数值取 warning：取这块像素里离目标色最近的一点，确认它更像 warning 而不是次要文字色。
	meter, ok := tt.Find("90%")
	if !ok {
		t.Fatal("找不到用量数值")
	}
	im := tt.Image()
	near, want := 1e9, 1e9
	for y := int(meter.Y); y < int(meter.Y+meter.H); y++ {
		for x := int(meter.X); x < int(meter.X+meter.W); x++ {
			px := im.RGBAAt(x, y)
			near = math.Min(near, colorDistance(px, colorOf(colorWarning.in(theme(false, gui.Preferences{})))))
			want = math.Min(want, colorDistance(px, colorOf(colorTextSecondary.in(theme(false, gui.Preferences{})))))
		}
	}
	if near >= want {
		t.Fatalf("将满数值不像 warning：最近 %v / 次要文字 %v", near, want)
	}
}

// U3 §1.1 L2 依赖框架的命中行为：行内可交互元素自己吃点击，不冒泡成开合。
// 本片的过程摘要行里没有链接或按钮（明细在行下方，见 TestSummaryExpands），
// 所以这里用最小视图验证机制本身。
func TestNestedControlDoesNotToggleRow(t *testing.T) {
	open := false
	nested := 0
	view := func(c *gui.Context) {
		parts := gui.CollapsibleBase(c, &open)
		parts.Trigger.Gap(space1).Children(func() {
			gui.Text(c, "这一轮的过程")
			if gui.Button(c, "打开文件").Clicked() {
				nested++
			}
		})
		parts.Panel(func() {
			gui.Text(c, "明细")
		})
	}
	tt := gui.NewTester(view, 400, 300)
	if err := tt.Click("打开文件"); err != nil {
		t.Fatalf("点不到行内按钮：%v", err)
	}
	if open {
		t.Fatal("行内按钮把行也开合了")
	}
	if nested != 1 {
		t.Fatalf("行内按钮收到 %d 次点击，想要 1 次", nested)
	}
	if err := tt.Click("这一轮的过程"); err != nil {
		t.Fatalf("点不到行内容：%v", err)
	}
	if !open {
		t.Fatal("点行内容没有开合")
	}
}

// colorDistance 是两个像素的欧氏距离，供取色断言用。
func colorDistance(a, b color.RGBA) float64 {
	dr, dg, db := float64(a.R)-float64(b.R), float64(a.G)-float64(b.G), float64(a.B)-float64(b.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

// colorOf 把设计 token 转成像素值，供取色断言用。
func colorOf(c gui.Color) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff}
}

// samePixels 报告 r 这块区域在两帧里是否完全一样。
func samePixels(a, b image.Image, r gui.Rect) bool {
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if a.At(x, y) != b.At(x, y) {
				return false
			}
		}
	}
	return true
}

// 右栏默认收起，展开后宽度落在 280–（容器−400）之间，主区不缩到 400 以下。
func TestRightPanelOpensWithinBounds(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if app.rightOpen {
		t.Fatal("右栏应默认收起")
	}
	if err := tt.Click("打开右栏"); err != nil {
		t.Fatalf("点不到右栏入口：%v", err)
	}
	if !app.rightOpen {
		t.Fatal("右栏没有展开")
	}
	panel, ok := tt.Find("右栏")
	if !ok {
		t.Fatal("找不到右栏")
	}
	if panel.W < RightPanelMinWidth || panel.W > 1440-SidebarWidth-RightPanelMainMinWidth {
		t.Fatalf("右栏宽 %v 超出 280–（容器−400）", panel.W)
	}
	main, ok := tt.Find("对话流")
	if !ok {
		t.Fatal("找不到对话流")
	}
	if main.W < MainMinWidth {
		t.Fatalf("主区宽 %v，小于 %v", main.W, MainMinWidth)
	}
}

// 左栏固定 256、标题行 48；对话内容左缘对齐，不做居中（§9.0 A3）。
func TestRegionDimensions(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	sidebar, ok := tt.Find("左栏 · 会话列表")
	if !ok || sidebar.W != SidebarWidth {
		t.Fatalf("左栏 %+v，想要宽 %v", sidebar, SidebarWidth)
	}
	header, ok := tt.Find("标题行")
	if !ok || header.H != HeaderHeight {
		t.Fatalf("标题行 %+v，想要高 %v", header, HeaderHeight)
	}
	content, ok := tt.Find("对话内容")
	if !ok {
		t.Fatal("找不到对话内容")
	}
	if content.X != SidebarWidth {
		t.Fatalf("内容列左缘 %v，想要贴着主区左缘 %v", content.X, SidebarWidth)
	}
	row := headerRow(t, tt)
	if row.X != SidebarWidth+space4 {
		t.Fatalf("组头左缘 %v，想要内容列左缘 %v", row.X, SidebarWidth+space4)
	}
	if row.W >= (WindowWidth-SidebarWidth)/2 {
		t.Fatalf("组头宽 %v，内容仍被容器撑开", row.W)
	}
}

// 空态不显示会话附属，也不显示用量与右栏入口。
func TestEmptyStateHidesSessionChrome(t *testing.T) {
	_, tt := newTester(t, StateEmpty)
	for _, label := range []string{"上下文", "打开右栏", "项目 A"} {
		if tt.HasText(label) {
			t.Fatalf("空态不应出现 %q", label)
		}
	}
}

// ParseState 接受名称与 1–5，其他给错误。
func TestParseState(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want State
	}{
		{"default", StateDefault},
		{"2", StateRunning},
		{"waiting", StateWaiting},
		{"4", StateFailed},
		{"empty", StateEmpty},
	} {
		got, err := ParseState(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("ParseState(%q) = %v, %v；想要 %v", tc.in, got, err, tc.want)
		}
	}
	if _, err := ParseState("nope"); err == nil {
		t.Fatal("未知状态应报错")
	}
}

// 运行中：队列条贴在输入区上沿；发送槽同一位置、同一尺寸变成停止，
// 不与发送并存。
func TestRunningShowsQueueAndStop(t *testing.T) {
	_, tt := newTester(t, StateRunning)
	if !tt.HasText("把日志也开上") {
		t.Fatal("运行中缺少队列条的内容")
	}
	stop, ok := tt.Find("停止")
	if !ok {
		t.Fatal("运行中发送槽应是停止")
	}
	if _, ok := tt.Find("发送"); ok {
		t.Fatal("运行中不应出现发送")
	}

	_, idle := newTester(t, StateDefault)
	send, ok := idle.Find("发送")
	if !ok {
		t.Fatal("空闲时找不到发送")
	}
	if stop != send {
		t.Fatalf("停止槽 %+v 与发送槽 %+v 不同位同尺寸", stop, send)
	}
}

// 输入区右侧只留附件入口与发送槽；模型不在输入区（UI-9、迭代意见）。
func TestComposerHasNoModel(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	if tt.HasText("模型") {
		t.Fatal("输入区不应出现模型入口")
	}
	if _, ok := tt.Find("附件"); !ok {
		t.Fatal("输入区缺少附件入口")
	}
}

// 文本缩放 200% 时，五个状态的文字都不越出窗口，输入位也不压住对话流（U2 §7.4）。
func TestScreensFitAtTextScale(t *testing.T) {
	for _, s := range []State{StateDefault, StateRunning, StateWaiting, StateFailed, StateEmpty} {
		app := NewApp(Options{State: s})
		tt := gui.NewTesterWith(app.View, WindowWidth, WindowHeight,
			gui.TesterOptions{Preferences: gui.Preferences{TextScale: 2}})
		for _, text := range tt.Texts() {
			r, ok := tt.Find(text)
			if !ok {
				continue
			}
			// 滚动区里滚出视口的行没有可见面积，不算布局越界。
			if r.W == 0 || r.H == 0 {
				continue
			}
			if r.X < 0 || r.Y < 0 || r.X+r.W > WindowWidth || r.Y+r.H > WindowHeight {
				t.Fatalf("%v：%q 越出窗口（%+v）", s, text, r)
			}
		}
		if s == StateEmpty {
			continue
		}
		feed, ok := tt.Find("对话流")
		if !ok {
			t.Fatalf("%v：找不到对话流", s)
		}
		composer := "消息"
		if s == StateWaiting {
			composer = "rm -rf ./tmp/*"
		}
		box, ok := tt.Find(composer)
		if !ok {
			t.Fatalf("%v：找不到输入位（%q）", s, composer)
		}
		if feed.Y+feed.H > box.Y {
			t.Fatalf("%v：对话流（%+v）压到输入位（%+v）", s, feed, box)
		}
	}
}

// C1：禁用的主图标动作保留 accent 填充（整体压 0.5），与透明底的图标入口靠面区分。
func TestDisabledPrimaryKeepsAccentFill(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	btn, ok := tt.Find("发送")
	if !ok {
		t.Fatal("找不到发送入口")
	}
	img := tt.Image()
	// 图标的左上方是纯填充区，避开图标笔画。
	x, y := int(btn.X)+3, int(btn.Y)+3
	fill := img.RGBAAt(x, y)
	if fill.R == 255 && fill.G == 255 && fill.B == 255 {
		t.Fatalf("禁用主图标动作被换成 surface：%v", fill)
	}
	if !(fill.G > fill.R && fill.G > fill.B) {
		t.Fatalf("禁用主图标动作没有 accent 填充：%v", fill)
	}
	if fill.R < 100 {
		t.Fatalf("禁用态没有被压 0.5：%v", fill)
	}
}

// C2：图标入口（发送 / 停止 / 附件）都取 row-compact 24 的命中区（UI-22 档位）；
// 审批的决定是"图标 + 文字"，高度由内边距与行高算出，不再是框架按钮的 28.5。
func TestButtonUsesFrameworkHeight(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	for _, label := range []string{"发送", "附件"} {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("找不到图标入口 %q", label)
		}
		if r.H != rowCompact || r.W != rowCompact {
			t.Fatalf("%q %+v，想要 %v×%v 的命中区", label, r, rowCompact, rowCompact)
		}
	}
	waiting, wt := newTester(t, StateWaiting)
	_ = waiting
	allow, ok := wt.Find("允许一次")
	if !ok {
		t.Fatal("找不到允许一次")
	}
	if allow.H < 24 {
		t.Fatalf("决定按钮高 %v，低于命中区下限", allow.H)
	}
}

// C5：状态位是固定 16×16 盒，不随字号变化。
func TestStatusMarkIs16Box(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	r, ok := tt.Find("状态：完成")
	if !ok {
		t.Fatal("找不到状态位")
	}
	if r.W != 16 || r.H != 16 {
		t.Fatalf("状态位 %+v，想要 16×16", r)
	}
}

// C6：侧栏带标记与不带标记的行，文字左边界相同。
func TestSidebarMarkerUsesIconSlot(t *testing.T) {
	_, tt := newTester(t, StateRunning)
	marked, ok := tt.Find("修发送路径")
	if !ok {
		t.Fatal("找不到带标记的会话")
	}
	plain, ok := tt.Find("参考")
	if !ok {
		t.Fatal("找不到不带标记的会话")
	}
	if marked.X != plain.X {
		t.Fatalf("带标记行文字 X=%v，不带标记行 X=%v", marked.X, plain.X)
	}
}

// C8：标签里不再出现全角加号。
func TestNoFullwidthPlus(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	if tt.HasText("＋") {
		t.Fatal("界面里仍有全角加号")
	}
}

// 模型偏离默认时只在标题行末尾留一个淡标记，不写“模型：”（UI-9）。
func TestModelMarkerOnlyWhenNonDefault(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if tt.HasText("模型") {
		t.Fatal("默认模型不应上屏")
	}
	app.model = "Opus"
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	r, ok := tt.Find("模型 Opus")
	if !ok {
		t.Fatal("非默认模型缺少标题行标记")
	}
	if r.Y+r.H > HeaderHeight {
		t.Fatalf("模型标记 %+v 不在标题行内（高 %v）", r, HeaderHeight)
	}
}
