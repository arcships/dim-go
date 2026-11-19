package ui

import (
	"context"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是 docs/audit-fakes.md §2「P1 空壳交互」的回归断言：一条 P1 一个用例，
// 断言的都是同一件事——**点了要么真的做了，要么置灰并写原因、状态不变**。
//
// 用例都连着注入的 Source（连接模式）：空壳交互只在连接模式里才算假，-demo 的
// 样例走查不在本文件的口径里。
//
// 已由既有用例覆盖的 P1（不在这里重复）：
//   - P1-1 归档：TestHeaderMenuUsesRealSessionActions
//   - P1-3 重命名：TestHeaderRenameIsAvailable
//   - P1-7 通用：TestGeneralScreenDisablesSampleToggles
//   - P1-8 外观：TestAppearanceThemeCardAppliesAndFollowsPreference / TestAppearancePreferencesPersist
//   - P1-9 个性化：TestPersonalizationControlsDisabledWithoutChannel
//   - P1-10 权限：TestPermissionsIsolationDisablesSampleSwitches
//   - P1-11 Git：TestGitScreenKeepsFormWithoutSampleRemotes
//   - P1-14 文件页：TestRightPanelUnopenedEmptyState

// p1Intents 接住本批断言用到的分支写入；其余方法走嵌入的 nil 接口，被调用即
// panic：说明测试点到了一个没接的动作。
type p1Intents struct {
	Intents
	branches chan p1BranchCut
}

// p1BranchCut 是一次 CreateBranch 的实参。
type p1BranchCut struct {
	parent string
	turn   int
	prompt string
}

func (p *p1Intents) CreateBranch(
	_ context.Context, parentID string, turnIndex int, prompt string,
) error {
	if p.branches != nil {
		p.branches <- p1BranchCut{parent: parentID, turn: turnIndex, prompt: prompt}
	}
	return nil
}

// P1-7：连上 Source 后这一屏的桌面项没有读写通道：开关、根目录输入与「重启
// App」、代理分段、浏览器数据两行动作都置灰并写原因（开关三条由
// TestGeneralScreenDisablesSampleToggles 覆盖，这里补齐其余入口）。
func TestGeneralScreenDesktopActionsDisabled(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secGeneral)

	if info, ok := tt.AccessByRole(gui.RoleButton, "重启 App"); !ok || !info.Disabled {
		t.Fatalf("「重启 App」没有置灰：%+v（找到 %v）", info, ok)
	}
	for _, label := range []string{"导入…", "清理数据…"} {
		info, ok := tt.Access(label)
		if !ok || !info.Disabled {
			t.Fatalf("浏览器数据动作 %q 没有置灰：%+v（找到 %v）", label, info, ok)
		}
	}
	if info, ok := tt.Access("代理模式"); !ok || !info.Disabled {
		t.Fatalf("代理分段没有置灰：%+v（找到 %v）", info, ok)
	}
	_ = tt.Click("导入…")
	tt.Frame()
	if generalScreen.importOpen || generalScreen.clearOpen || generalScreen.proxyDialog {
		t.Fatalf("置灰的动作仍打开了对话框：%+v", generalScreen)
	}
}

// P1-2 / P1-4：标题栏 ⋯ 的「置顶」与左栏会话行的置顶是同一份视图状态，点它
// 真的把会话放进「已置顶」；置顶按会话标识记，列表重排后不会跟到别人的行上。
func TestHeaderPinPinsSidebarSession(t *testing.T) {
	src := wiredSource()
	src.names = []string{"s1", "s2"}
	src.titles = []string{"会话一", "会话二"}
	src.session = SessionSnapshot{ID: "s1", Title: "会话一"}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if err := tt.Click("更多操作"); err != nil {
		t.Fatalf("点不到 ⋯ 菜单：%v", err)
	}
	tt.Frame()
	if err := tt.Click("置顶"); err != nil {
		t.Fatalf("点不到置顶：%v", err)
	}
	tt.Frame()

	if !app.pinned(0) {
		t.Fatal("标题栏置顶没有落到会话上")
	}
	// 置顶真的落到左栏「已置顶」：不是只改菜单文案。
	pinned := app.sidebarPinnedThreads(app.sidebarThreads())
	if len(pinned) != 1 || pinned[0].index != 0 {
		t.Fatalf("左栏「已置顶」= %+v，想要只有 s1", pinned)
	}
	if _, ok := tt.Find("已置顶"); !ok {
		t.Fatalf("左栏没有出现「已置顶」：%v", tt.Texts())
	}

	// 目录重排：s1 换到第 1 行。置顶要跟着会话走，不能跟到新占第 0 行的 s2。
	src.names = []string{"s2", "s1"}
	src.titles = []string{"会话二", "会话一"}
	tt.Frame()
	if app.pinned(0) {
		t.Fatal("置顶跟到了重排后第 0 行的 s2")
	}
	if !app.pinned(1) {
		t.Fatal("s1 换行后丢了置顶")
	}
	pinned = app.sidebarPinnedThreads(app.sidebarThreads())
	if len(pinned) != 1 || pinned[0].index != 1 {
		t.Fatalf("重排后「已置顶」= %+v，想要原来的 s1", pinned)
	}
}

// P1-5：状态条「创建新分支」在连接模式下真的发 session.branch_create（切点是
// 当前分支已加载的最后一轮），不再点了没反应。
func TestStatusBarCreateBranchSendsIntent(t *testing.T) {
	src := wiredSource()
	src.session = SessionSnapshot{
		ID: "s1", Title: "真实会话", Branch: "feature-x",
		Branches:  []BranchSnapshot{{ID: "feature-x"}},
		TurnCount: 4,
	}
	intents := &p1Intents{branches: make(chan p1BranchCut, 1)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if err := tt.Click("分支 feature-x"); err != nil {
		t.Fatalf("点不到分支菜单：%v", err)
	}
	tt.Frame()
	if err := tt.Click("创建新分支"); err != nil {
		t.Fatalf("点不到创建新分支：%v", err)
	}
	select {
	case got := <-intents.branches:
		if got.parent != "feature-x" || got.turn != 4 || got.prompt == "" {
			t.Fatalf("CreateBranch 收到 %+v，想要 feature-x/4/非空提示词", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("创建新分支没有发出 session.branch_create")
	}
}

// P1-6：空态工作区菜单的「添加新工作区」走真正的目录选择，选中的目录登记成
// 工作区并落盘；不再只把标题改成「新工作区」。
func TestComposerAddWorkspaceOpensDirectoryPicker(t *testing.T) {
	dir := pickedDir(t, "composer-workspace")
	src := wiredSource()
	// 接上事实但没有打开会话：输入卡下面带工作区与权限两处入口。
	src.names, src.hasSess, src.session = nil, false, SessionSnapshot{}
	changed := make(chan struct{}, 8)
	prefs := make(chan PreferencesSnapshot, 8)
	app := NewApp(Options{
		State: StateEmpty, Source: src,
		Intents: &dirPickIntents{prefs: prefs},
		Changed: func() {
			select {
			case changed <- struct{}{}:
			default:
			}
		},
	})
	app.dirPickGOOS = "darwin"
	app.dirPickRunner = func(context.Context, string, ...string) (string, error) {
		return dir, nil
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if err := tt.Click("选择工作区"); err != nil {
		t.Fatalf("点不到工作区菜单：%v", err)
	}
	tt.Frame()
	if err := tt.Click("添加新工作区"); err != nil {
		t.Fatalf("点不到添加新工作区：%v", err)
	}
	waitChanged(t, changed, "添加新工作区")
	tt.Frame()

	if len(app.workspaces) != 1 || app.workspaces[0] != dir {
		t.Fatalf("登记的目录 = %v，想要 %q", app.workspaces, dir)
	}
	if app.workspaceName == "新工作区" {
		t.Fatal("仍然只是把上方标题改成占位名字")
	}
	got := waitPrefs(t, prefs, "添加新工作区")
	if len(got.Workspaces) != 1 || got.Workspaces[0] != dir {
		t.Fatalf("落盘的目录 = %v，想要 %q", got.Workspaces, dir)
	}
}

// P1-12：自动重试行只有倒计时是事实；「立即重试」没有通道，置灰并把原因写在
// tooltip 上，点了不改任何状态。
func TestTimelineRetryNowDisabledWithoutChannel(t *testing.T) {
	src := wiredSource()
	src.session = SessionSnapshot{
		ID: "s1", Title: "真实会话",
		Rows: []RowSnapshot{{Kind: RowRetry, Duration: 5 * time.Second}},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if !tt.HasText("5s 后重试") {
		t.Fatalf("没有画倒计时：%v", tt.Texts())
	}
	info, ok := tt.Access("立即重试")
	if !ok {
		t.Fatalf("没有「立即重试」入口：%v", tt.Texts())
	}
	if !info.Disabled {
		t.Fatalf("「立即重试」可点（点了没有任何请求）：%+v", info)
	}
	if info.Description != noteRetryNow {
		t.Fatalf("「立即重试」没写原因：%+v", info)
	}
	_ = tt.Click("立即重试")
	tt.Frame()
	if app.retryIntent != nil {
		t.Fatal("点「立即重试」排上了一次重试动作")
	}
}

// P1-13：侧聊的「结束分支」「停止」没有通道：入口照留、置灰、原因进 tooltip，
// 点了既不弹确认框也不改结束 / 发送中的布尔值。
func TestSideChatStopAndEndDisabledWithoutChannel(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	app.rightOpen, app.rightTab = true, tabSideChat
	app.panels.sideChatSent = []string{"历史问题"}
	app.panels.sideChatSending = true
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	end, ok := tt.Access("结束分支")
	if !ok {
		t.Fatalf("侧聊没有结束分支入口：%v", tt.Texts())
	}
	if !end.Disabled || end.Description != noteSideChatWrite {
		t.Fatalf("「结束分支」没有置灰或没写原因：%+v", end)
	}
	stop, ok := tt.Access("停止")
	if !ok {
		t.Fatalf("侧聊没有停止入口：%v", tt.Texts())
	}
	if !stop.Disabled || stop.Description != noteSideChatWrite {
		t.Fatalf("「停止」没有置灰或没写原因：%+v", stop)
	}

	_ = tt.Click("结束分支")
	_ = tt.Click("停止")
	tt.Frame()
	if app.panels.sideChatConfirm {
		t.Fatal("点「结束分支」仍弹出一个改了也没用的确认框")
	}
	if app.panels.sideChatEnded {
		t.Fatal("点「结束分支」仍把本地状态写成已结束")
	}
	if !app.panels.sideChatSending {
		t.Fatal("点「停止」改了本地发送态（服务端并不受影响）")
	}
	if _, ok := tt.Find(sideChatEndTitle); ok {
		t.Fatal("连接模式下仍出现结束分支确认框")
	}
}
