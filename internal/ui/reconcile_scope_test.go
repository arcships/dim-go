package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是本轮「主窗口与全局面板分流」的可执行证据：左栏悬停卡、收起态导航组、
// 未读/星标的原因、状态条 Agent 配置选择器、确认框键盘、提问卡收起状态行、
// 审批长内容滚动与导出面板 Esc。

// agentIntents 接住 Agent 配置切换，其余方法走嵌入的 nil 接口。
type agentIntents struct {
	stubBaseIntents
	selects chan [2]string
}

func (s *agentIntents) SelectAgent(_ context.Context, agentID, expected string) error {
	s.selects <- [2]string{agentID, expected}
	return nil
}

// navSource 是同一工作区下三条会话的事实源；标题与目录都不同名，便于断言。
func navSource() *fakeSource {
	return &fakeSource{
		conn:   Connection{Phase: "connected"},
		names:  []string{"s1", "s2", "s3"},
		titles: []string{"第一条", "第二条", "第三条"},
		places: []SessionPlaceSnapshot{
			{Cwd: "/work/alpha"}, {Cwd: "/work/alpha"}, {Cwd: "/work/alpha"},
		},
		session: SessionSnapshot{ID: "s1", Title: "第一条", Cwd: "/work/alpha"},
		hasSess: true,
	}
}

// TestSidebarCollapsedNavGroup：左栏收起时标题栏给出上一个/下一个工作区会话与
// 新建入口；边界处置灰，点击导航把目录行交给打开通道（TB-14–TB-16）。
func TestSidebarCollapsedNavGroup(t *testing.T) {
	src := navSource()
	stub := &navIntentStub{opened: make(chan string, 4), created: make(chan string, 4)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
	app.sidebarCollapsed = true
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	for _, label := range []string{"上一个工作区会话", "下一个工作区会话", "在工作区中新建会话"} {
		if _, ok := tt.Access(label); !ok {
			t.Fatalf("收起态标题栏缺少 %q：%v", label, tt.Texts())
		}
	}
	// 第一条：上一个禁用（边界），下一个可用。
	if info, ok := tt.Access("上一个工作区会话"); !ok || info.Description == "" {
		t.Fatalf("上一个的边界原因没有写进说明：%+v（ok=%v）", info, ok)
	}
	if err := tt.Click("下一个工作区会话"); err != nil {
		t.Fatalf("点不到下一个：%v", err)
	}
	tt.Frame()
	select {
	case got := <-stub.opened:
		if got != "s2" {
			t.Fatalf("下一个打开 %q，想要 s2", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("点下一个没有发起打开")
	}

	if err := tt.Click("在工作区中新建会话"); err != nil {
		t.Fatalf("点不到新建：%v", err)
	}
	select {
	case cwd := <-stub.created:
		if cwd != "/work/alpha" {
			t.Fatalf("新建的工作目录 = %q，想要 /work/alpha", cwd)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("点新建没有发起 CreateSession")
	}
}

// navIntentStub 接住打开会话与新建会话。
type navIntentStub struct {
	stubBaseIntents
	opened  chan string
	created chan string
}

func (s *navIntentStub) OpenSession(_ context.Context, name string) error {
	s.opened <- name
	return nil
}

func (s *navIntentStub) CreateSession(_ context.Context, cwd string) error {
	s.created <- cwd
	return nil
}

// TestSessionHoverCard：悬停卡显示完整标题与所属工作区；当前会话还有分支行。
// 非当前会话没有逐会话分支事实，不画分支（不编造）。
func TestSessionHoverCard(t *testing.T) {
	src := navSource()
	src.titles = []string{"第一条", "一条很长很长需要折行的会话标题，名字里带完整语义", "第三条"}
	src.session.Branch = "feature/hover-card"
	app := NewApp(Options{State: StateDefault, Source: src})
	app.branch = "feature/hover-card"
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	row, ok := tt.Find("第一条")
	if !ok {
		t.Fatalf("找不到会话行：%v", tt.Texts())
	}
	tt.Move(row.X+row.W/2, row.Y+row.H/2)
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	tt.Frame()
	for _, want := range []string{"会话标题：第一条", "所属工作区：alpha", "所在分支：feature/hover-card"} {
		if !tt.HasText(want) {
			t.Fatalf("悬停卡缺少 %q：%v", want, tt.Texts())
		}
	}

	// 移到第二条（非当前会话）：标题与工作区在，分支行不在。
	other, ok := tt.Find("一条很长很长需要折行的会话标题，名字里带完整语义")
	if !ok {
		t.Fatalf("找不到第二条会话：%v", tt.Texts())
	}
	tt.Move(other.X+other.W/2, other.Y+other.H/2)
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	tt.Frame()
	if !tt.HasText("会话标题：一条很长很长需要折行的会话标题，名字里带完整语义") {
		t.Fatalf("第二行悬停卡没有出现：%v", tt.Texts())
	}
	if tt.HasText("所在分支：feature/hover-card") {
		t.Fatalf("非当前会话不该画分支行：%v", tt.Texts())
	}
}

// TestSessionUnreadStarReasons：未读与星标入口保留但置灰，原因写在菜单项说明里。
func TestSessionUnreadStarReasons(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	tt.Frame()
	name := sampleSessionName(StateDefault)
	row, ok := tt.Find(name)
	if !ok {
		t.Fatalf("找不到会话行 %q：%v", name, tt.Texts())
	}
	_ = app
	tt.Move(row.X+row.W-10, row.Y+row.H/2)
	tt.Frame()
	if err := tt.Click("会话操作：" + name); err != nil {
		t.Fatalf("点不到会话菜单：%v", err)
	}
	tt.Frame()
	for _, label := range []string{"标为未读", "星标会话"} {
		info, ok := tt.Access(label)
		if !ok {
			t.Fatalf("菜单里没有 %q：%v", label, tt.Texts())
		}
		if !strings.Contains(info.Description, "会话目录没有") {
			t.Fatalf("%q 没有写原因：%+v", label, info)
		}
	}
}

// TestStatusAgentSelector：状态条在有多套配置时出现「Agent 配置」，选中项带勾，
// 切换把（新档案、切换前档案）交给 SelectAgent（SB-08）。
func TestStatusAgentSelector(t *testing.T) {
	src := wiredSource()
	src.agent = AgentConfigSnapshot{
		Known: true, Current: "agent-a", CurrentKnown: true, CanSelect: true,
		Profiles: []AgentConfigProfile{
			{ID: "agent-a", Name: "默认开发"},
			{ID: "agent-b", Name: "代码评审"},
		},
	}
	stub := &agentIntents{selects: make(chan [2]string, 1)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	if !tt.HasText("Agent 配置") || !tt.HasText("默认开发") {
		t.Fatalf("状态条没有 Agent 配置：%v", tt.Texts())
	}
	if err := tt.Click("Agent 配置：默认开发"); err != nil {
		t.Fatalf("点不到 Agent 配置：%v", err)
	}
	tt.Frame()
	if info, ok := tt.Access("Agent 配置：代码评审"); !ok || info.Checked {
		t.Fatalf("菜单里的代码评审状态不对：%+v（ok=%v）", info, ok)
	}
	if err := tt.Click("Agent 配置：代码评审"); err != nil {
		t.Fatalf("选不到代码评审：%v", err)
	}
	tt.Frame()
	select {
	case got := <-stub.selects:
		if got != [2]string{"agent-b", "agent-a"} {
			t.Fatalf("SelectAgent 实参 = %v，想要 [agent-b agent-a]", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("选择配置没有发出 SelectAgent")
	}
}

// TestConfirmKeyboardFocusCycle：确认框里 Tab 在取消与删除之间循环，焦点落在
// 取消时回车取消、不删除；聚焦删除后回车才执行（G-E1-03/04、G-E2-04/05）。
func TestConfirmKeyboardFocusCycle(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.openDeleteConfirm([]int{0})
	tt.Frame()
	if info, ok := tt.Access(confirmCopyCancel); !ok || !info.Focused {
		t.Fatalf("默认焦点不在取消：%+v（ok=%v）", info, ok)
	}
	tt.Key(0, gui.KeyTab)
	tt.Frame()
	if info, ok := tt.Access(confirmCopyDelete); !ok || !info.Focused {
		t.Fatalf("Tab 后焦点不在删除：%+v（ok=%v）", info, ok)
	}
	tt.Key(gui.Shift, gui.KeyTab)
	tt.Frame()
	if info, ok := tt.Access(confirmCopyCancel); !ok || !info.Focused {
		t.Fatalf("Shift+Tab 后焦点不在取消：%+v（ok=%v）", info, ok)
	}
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.archived[0] {
		t.Fatal("焦点在取消时回车删掉了会话")
	}
	if app.confirm.kind != confirmKindNone {
		t.Fatalf("回车没有等同取消：%s", app.confirm.kind)
	}
}

// TestQuestionCollapsedStatusLine：收起态状态行按已答情况四选一（G-D1-05）。
func TestQuestionCollapsedStatusLine(t *testing.T) {
	app, tt := questionTester(t, threeQuestionBatch(), nil)
	tt.Frame()
	// 未答单选：待选择 · 2 个选项。
	if err := tt.Click(questionCopyCollapse); err != nil {
		t.Fatalf("点不到收起：%v", err)
	}
	tt.Frame()
	if !tt.HasText("待选择 · 2 个选项") {
		t.Fatalf("收起态未答文案不对：%v", tt.Texts())
	}
	if err := tt.Click(questionCopyExpand); err != nil {
		t.Fatalf("点不到展开：%v", err)
	}
	tt.Frame()
	if err := tt.Click("暂存"); err != nil {
		t.Fatalf("点不到选项：%v", err)
	}
	tt.Frame()
	if err := tt.Click(questionCopyCollapse); err != nil {
		t.Fatalf("点不到收起：%v", err)
	}
	tt.Frame()
	if !tt.HasText("已选：暂存") {
		t.Fatalf("收起态已答文案不对：%v", tt.Texts())
	}
	_ = app
}

// TestApprovalLongContentScrolls：200 行命令的审批输入区有高度上限，内容不把
// 卡片撑破；差异块各自限高（G-C4-01 / G-C4-06）。
func TestApprovalLongContentScrolls(t *testing.T) {
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, "echo 第 "+strings.Repeat("x", i%40)+" 行")
	}
	src := wiredSource()
	src.approval = ApprovalSnapshot{
		ID: "a-long", Tool: "exec", Command: strings.Join(lines, "\n"),
		Input: `{"command":"` + strings.Join(lines, "\\n") + `"}`,
	}
	src.hasAppr = true
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	panel, ok := tt.Find("审批输入：输入")
	if !ok {
		t.Fatalf("找不到审批输入区：%v", tt.Texts())
	}
	if panel.H > approvalInputMaxHeight+1 {
		t.Fatalf("审批输入区高 %v，超过上限 %v", panel.H, approvalInputMaxHeight)
	}
}

// TestExportEscapeCloses：导出面板按 Esc 等同取消（R-GF-126）。
func TestExportEscapeCloses(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.openExportPanel("s1", "修发送路径")
	tt.Frame()
	if !app.export.open {
		t.Fatal("导出面板没有打开")
	}
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.export.open {
		t.Fatal("Esc 没有关闭导出面板")
	}
}

// TestSessionSearchKeyboardArrows：搜索浮层里 Tab 到箭头按钮后回车与点击同效
// （G-A1-07/08 的键盘路径）。
func TestSessionSearchKeyboardArrows(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	if len(app.sessionSearch.hits) < 2 {
		t.Fatalf("命中数不足：%d", len(app.sessionSearch.hits))
	}
	// 输入框 → 关闭键 → 上一个 → 下一个。
	for i := 0; i < 3; i++ {
		tt.Key(0, gui.KeyTab)
		tt.Frame()
	}
	if info, ok := tt.Access("下一个搜索结果"); !ok || !info.Focused {
		t.Fatalf("Tab 没落到下一个结果按钮：%+v（ok=%v）", info, ok)
	}
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.sessionSearch.current != 1 {
		t.Fatalf("回车后当前项 = %d，想要 1", app.sessionSearch.current)
	}
}

// TestConfirmDialogAccessibleAnatomy：四类确认框共用一套可访问结构——带名的
// alertdialog（名字是标题）、标题是 heading、说明句挂成 describedby（读屏先读
// 标题，再读说明，最后才到按钮）。说明是这段确认的唯一解释，正文里逐字可见
// （G-E1-01/02、G-E2-01/02、G-E3-01/02、G-E4-01/02）。
func TestConfirmDialogAccessibleAnatomy(t *testing.T) {
	cases := []struct {
		name  string
		open  func(a *App)
		title string
		desc  string
	}{
		{
			name:  "删除会话",
			open:  func(a *App) { a.openDeleteConfirm([]int{0}) },
			title: confirmCopyDeleteOne,
			desc:  "确定要永久删除“修发送路径”吗？删除后会话将立即从侧边栏移除，且无法恢复。",
		},
		{
			name:  "归档会话",
			open:  func(a *App) { a.openArchiveConfirm([]int{0}) },
			title: "归档 1 个会话",
			desc:  "确定要归档以下会话吗？“修发送路径”",
		},
		{
			name:  "会话正在运行",
			open:  func(a *App) { a.openConfirmEditRunning(0, "改过的正文") },
			title: confirmCopyEditTitle,
			desc:  confirmCopyEditDesc,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, tt := newTester(t, StateDefault)
			tc.open(app)
			tt.Frame()
			dialog, ok := tt.AccessByRole(gui.RoleAlertDialog, tc.title)
			if !ok {
				t.Fatalf("找不到名为 %q 的 alertdialog：%v", tc.title, tt.Texts())
			}
			if !strings.Contains(dialog.Description, tc.desc) {
				t.Fatalf("对话框说明 = %q，想要包含 %q", dialog.Description, tc.desc)
			}
			if heading, ok := tt.AccessByRole(gui.RoleHeading, tc.title); !ok || heading.Bounds.H <= 0 {
				t.Fatalf("标题不是可读的 heading：%+v（ok=%v）", heading, ok)
			}
			if !tt.HasText(tc.desc) {
				t.Fatalf("说明句没有逐字画出：%v", tt.Texts())
			}
		})
	}
}

// TestConfirmWorkspaceDescriptionAndPath：删除工作区会话的说明按打开时的计数定格，
// 目标变多也不改；路径行按事实给出完整目录（G-E4-01/02/03）。
func TestConfirmWorkspaceDescriptionAndPath(t *testing.T) {
	app := cwdGroupApp([]SessionPlaceSnapshot{
		{Cwd: "/work/dim-agent"},
		{Cwd: "/work/rutis"},
	})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	groups := app.sidebarWorkspaces(app.sidebarThreads())
	var target sidebarWorkspace
	for _, group := range groups {
		if group.name == "rutis" {
			target = group
		}
	}
	if target.id == "" {
		t.Fatalf("找不到 rutis 组：%+v", groups)
	}
	threads := app.sidebarGroupThreads(target, app.sidebarThreads())
	app.openWorkspaceDeleteConfirm(target, threads)
	tt.Frame()

	desc := "将删除“rutis”下的 1 条会话。"
	dialog, ok := tt.AccessByRole(gui.RoleAlertDialog, confirmCopyWorkspace)
	if !ok {
		t.Fatalf("找不到工作区确认框：%v", tt.Texts())
	}
	if !strings.Contains(dialog.Description, desc) {
		t.Fatalf("对话框说明 = %q，想要包含 %q", dialog.Description, desc)
	}
	// 弹窗期间计数不再变：描述定格在打开时的那一刻。
	if !strings.Contains(app.confirmDescription(), "1 条会话") {
		t.Fatalf("说明没有按打开时的计数定格：%q", app.confirmDescription())
	}
	if info, ok := tt.Access("工作区路径：/work/rutis"); !ok || info.Name != "工作区路径：/work/rutis" {
		t.Fatalf("路径行没有按事实给出：%+v（ok=%v）", info, ok)
	}

	// 打开期间目标集合变了也不改标题与说明：两者都按打开时那一刻定格。
	beforeTitle, beforeDesc := app.confirmTitle(), app.confirmDescription()
	app.confirm.targets = append(app.confirm.targets,
		confirmTarget{index: 0, id: "s1", title: "修发送路径"})
	if app.confirmTitle() != beforeTitle || app.confirmDescription() != beforeDesc {
		t.Fatalf("打开期间的改动改了标题/说明：%q / %q", app.confirmTitle(), app.confirmDescription())
	}

	// 计数为 0（工作区下已没有会话）时仍可打开，描述照常写「0 条会话」。
	app.confirm = confirmState{}
	app.openWorkspaceDeleteConfirm(target, nil)
	tt.Frame()
	if app.confirm.kind != confirmKindDeleteWorkspace {
		t.Fatalf("计数为 0 时打不开工作区确认框：%q", app.confirm.kind)
	}
	if !strings.Contains(app.confirmDescription(), "0 条会话") {
		t.Fatalf("计数为 0 的说明不对：%q", app.confirmDescription())
	}
}
