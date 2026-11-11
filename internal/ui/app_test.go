package ui

import (
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func newTester(t *testing.T, s State) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: s})
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// 每个状态都要画出自己的签名文案，证明状态切换落到区域与元素上。
func TestStatesShowTheirState(t *testing.T) {
	cases := []struct {
		state State
		want  string
		gone  string
	}{
		{StateDefault, "完成", "等待你同意"},
		{StateRunning, "正在运行", "等待你同意"},
		{StateWaiting, "需要你同意：跑一条命令", "发送路径.md"},
		{StateFailed, "失败", "排队中 1"},
		{StateEmpty, "还没有消息。写点什么开始，或把文件拖进来。", "修发送路径"},
	}
	for _, tc := range cases {
		_, tt := newTester(t, tc.state)
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
	for _, label := range []string{"允许一次", "始终允许", "拒绝"} {
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
	if tt.HasText("始终允许") {
		t.Fatal("没有命令原文时不应出现始终允许")
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
	for i := 0; i < 16 && len(reached) < 3; i++ {
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

// 过程摘要行折叠时只留状态判断；条数、文件数与耗时进展开（UI-11）。
func TestSummaryKeepsCountsInExpansion(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	for _, s := range []string{"个操作", "改了", "12s", "耗时"} {
		if tt.HasText(s) {
			t.Fatalf("折叠时不应出现明细 %q", s)
		}
	}
	if !tt.HasText("完成") {
		t.Fatal("折叠时应保留状态判断")
	}
	if err := tt.Click("展开"); err != nil {
		t.Fatalf("点不到展开：%v", err)
	}
	if app.summaryOpen[0] == false {
		t.Fatal("第一轮摘要没有展开")
	}
	if !tt.HasText("4 个操作 · 改了 2 个文件 · 耗时 12s") {
		t.Fatal("展开后缺少操作数、文件数与耗时")
	}
}

// 过程摘要行展开后按发生顺序给出操作与失败原因（UI-11）。
func TestSummaryExpands(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if tt.HasText("原因：权限不足，命令没有执行") {
		t.Fatal("折叠时不应给出展开内容")
	}
	if err := tt.Click("展开"); err != nil {
		t.Fatalf("点不到展开：%v", err)
	}
	if !app.summaryOpen[0] {
		t.Fatal("第一轮摘要没有展开")
	}
	if !tt.HasText("原因：权限不足，命令没有执行") {
		t.Fatal("展开后缺少失败原因")
	}
	if !tt.HasText("go test ./...") {
		t.Fatal("展开后缺少操作对象")
	}
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

// 内容上限 784 居中；左栏固定 256；标题行 48。
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
	if !ok || content.W > ContentMaxWidth {
		t.Fatalf("内容 %+v，上限 %v", content, ContentMaxWidth)
	}
	if content.X+content.W/2 != SidebarWidth+(1440-SidebarWidth)/2 {
		t.Fatalf("内容没有居中：%+v", content)
	}
}

// 空态不显示会话附属，也不显示用量与右栏入口。
func TestEmptyStateHidesSessionChrome(t *testing.T) {
	_, tt := newTester(t, StateEmpty)
	for _, label := range []string{"上下文用量", "打开右栏", "项目 A"} {
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
	if !tt.HasText("排队中 1") {
		t.Fatal("运行中缺少队列条")
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
			composer = "需要你同意：跑一条命令"
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

// C1：禁用的主按钮保留 accent 填充，由框架整体压 0.5；与 surface 次按钮靠面区分。
func TestDisabledPrimaryKeepsAccentFill(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	btn, ok := tt.Find("发送消息")
	if !ok {
		t.Fatal("找不到禁用的发送按钮")
	}
	img := tt.Image()
	// 文字左侧的按钮填充，避开字形。
	x, y := int(btn.X)+4, int(btn.Y+btn.H/2)
	fill := img.RGBAAt(x, y)
	if fill.R == 244 && fill.G == 244 && fill.B == 245 {
		t.Fatalf("禁用主按钮被换成 surface：%v", fill)
	}
	if !(fill.B > fill.R && fill.B > fill.G) {
		t.Fatalf("禁用主按钮没有 accent 填充：%v", fill)
	}
}

// C2：按钮高度取框架默认（6/14 内边距）；窗口未强调时主按钮由框架内缩 1px，
// 因此禁用主按钮 28.5、次按钮 30.5，都不是我们覆盖出来的值。
func TestButtonUsesFrameworkHeight(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	send, ok := tt.Find("发送消息")
	if !ok {
		t.Fatal("找不到发送按钮")
	}
	attach, ok := tt.Find("附件")
	if !ok {
		t.Fatal("找不到附件按钮")
	}
	if send.H < 28 || send.H > 29 {
		t.Fatalf("禁用主按钮高 %v，想要框架未强调窗口的 28.5", send.H)
	}
	if attach.H < 30 || attach.H > 31 {
		t.Fatalf("次按钮高 %v，想要框架默认的 30.5", attach.H)
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
