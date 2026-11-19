package ui

import (
	"context"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「元素在场但点了没反应」这一类的测试：键位落点、菜单动作、行内改名、
// 移除工作区、退出登录确认。每条都从屏幕上点到那个入口，不直接改视图状态。

// TestViewShortcutsToggleSidebarAndOpenSettings：⌘B 收起 / 展开左栏（连按两次回到
// 收起态），⌘, 打开设置（LB-97），焦点在输入框里也不吃字符。
func TestViewShortcutsToggleSidebarAndOpenSettings(t *testing.T) {
	app, tt := newTester(t, StateDefault)

	tt.Key(gui.Cmd, gui.KeyB)
	tt.Frame()
	if !app.sidebarCollapsed {
		t.Fatal("⌘B 没有收起左栏")
	}
	tt.Key(gui.Cmd, gui.KeyB)
	tt.Frame()
	if app.sidebarCollapsed {
		t.Fatal("再按 ⌘B 没有展开左栏")
	}

	// 焦点在输入框里时 ⌘B 仍生效，草稿不受影响。
	app.draft = "草稿还在"
	if err := tt.Click("消息"); err != nil {
		t.Fatalf("点不到输入框：%v", err)
	}
	tt.Key(gui.Cmd, gui.KeyB)
	tt.Frame()
	if !app.sidebarCollapsed {
		t.Fatal("焦点在输入框时 ⌘B 失效")
	}
	if app.draft != "草稿还在" {
		t.Fatalf("⌘B 改动了草稿：%q", app.draft)
	}

	tt.Key(gui.Cmd, gui.KeyComma)
	tt.Frame()
	if app.screen != ScreenSettings {
		t.Fatal("⌘, 没有打开设置")
	}
}

// stubBaseIntents 接住视图状态落盘（视图变化会写一次偏好）；其余方法走嵌入的
// nil 接口，被调用即 panic，说明测试点到了一个没接的动作。
type stubBaseIntents struct{ Intents }

func (stubBaseIntents) SetPreferences(context.Context, PreferencesSnapshot) error { return nil }

// stubAccountIntents 只接退出登录：其余方法走嵌入的 nil 接口（被调用即 panic）。
type stubAccountIntents struct {
	stubBaseIntents
	loggedOut chan struct{}
}

func (s *stubAccountIntents) Logout(_ context.Context) error {
	s.loggedOut <- struct{}{}
	return nil
}

// TestAccountMenuLogoutAsksTwiceAndCallsIntent：账户菜单的「退出登录」先出确认行
// （LB-101），确认后才发 account.logout；Pet 与邀请项禁用并写原因（LB-94）。
func TestAccountMenuLogoutAsksTwiceAndCallsIntent(t *testing.T) {
	src := wiredSource()
	stub := &stubAccountIntents{loggedOut: make(chan struct{}, 1)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	row, ok := tt.Find(app.accountNameValue())
	if !ok {
		t.Fatalf("找不到账户行：%v", tt.Texts())
	}
	if err := tt.Click(app.accountNameValue()); err != nil {
		t.Fatalf("点账户行：%v", err)
	}
	tt.Frame()
	menu := tt.Menu()
	for _, want := range []string{"显示 Pet", "邀请朋友", "设置", "退出登录"} {
		if !menuHas(menu, want) {
			t.Fatalf("账户菜单缺 %q（实际 %v）", want, menu)
		}
	}
	if err := tt.ChooseMenuItem("退出登录"); err != nil {
		t.Fatalf("点不到退出登录：%v", err)
	}
	tt.Frame()
	if !app.logoutConfirm {
		t.Fatal("退出登录没有进入确认态")
	}
	if _, ok := tt.Find("退出 Dim？"); !ok {
		t.Fatalf("确认行没有出现：%v", tt.Texts())
	}
	select {
	case <-stub.loggedOut:
		t.Fatal("确认前就登出了")
	default:
	}

	if err := tt.Click("取消退出登录"); err != nil {
		t.Fatalf("点不到取消：%v", err)
	}
	tt.Frame()
	if app.logoutConfirm {
		t.Fatal("取消后仍在确认态")
	}
	if _, ok := tt.Find("退出 Dim？"); ok {
		t.Fatal("取消后确认行还在")
	}

	// 再走一遍并确认：只有确认后才发 account.logout。
	if err := tt.Click(app.accountNameValue()); err != nil {
		t.Fatalf("再点账户行：%v", err)
	}
	tt.Frame()
	if err := tt.ChooseMenuItem("退出登录"); err != nil {
		t.Fatalf("再点退出登录：%v", err)
	}
	tt.Frame()
	if err := tt.Click("确认退出登录"); err != nil {
		t.Fatalf("点不到确认：%v", err)
	}
	select {
	case <-stub.loggedOut:
	case <-time.After(2 * time.Second):
		t.Fatal("确认后没有发 account.logout")
	}
	_ = row
}

// menuHas 报告菜单里有没有这一项（tt.Menu 只给文案）。
func menuHas(items []string, want string) bool {
	for _, got := range items {
		if got == want {
			return true
		}
	}
	return false
}

// TestSidebarSkillsEntryOpensPluginSettings：左栏「技能/插件」打开设置 › 插件设置
// （LB-15），不留在会话界面。
func TestSidebarSkillsEntryOpensPluginSettings(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	if err := tt.Click("技能/插件"); err != nil {
		t.Fatalf("点不到技能/插件：%v", err)
	}
	tt.Frame()
	if app.screen != ScreenSettings || app.settingsSection != secPluginSettings {
		t.Fatalf("落到 screen=%v section=%v，想要设置 › 插件设置", app.screen, app.settingsSection)
	}
}

// stubCreateIntents 只接新建会话：记下 cwd。
type stubCreateIntents struct {
	stubBaseIntents
	created chan string
}

func (s *stubCreateIntents) CreateSession(_ context.Context, cwd string) error {
	s.created <- cwd
	return nil
}

// TestWorkspaceGroupPlusCreatesInThatWorkspace：组头的新建按钮用该组的目录新建
// （LB-42），不拿当前会话的目录顶替，并把该组展开。
func TestWorkspaceGroupPlusCreatesInThatWorkspace(t *testing.T) {
	app := cwdGroupApp([]SessionPlaceSnapshot{
		{Cwd: "/work/dim-agent"},
		{Cwd: "/work/rutis"},
		{Cwd: "/work/dim-agent"},
	})
	stub := &stubCreateIntents{created: make(chan string, 1)}
	app.intents = stub
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	group := app.sidebarWorkspaces(app.sidebarThreads())[2] // rutis
	app.sidebarView.collapsed = map[string]bool{group.id: true}
	tt.Frame()

	row, ok := tt.Find("rutis")
	if !ok {
		t.Fatalf("找不到 rutis 组头：%v", tt.Texts())
	}
	tt.Move(row.X+row.W-20, row.Y+row.H/2)
	if err := tt.Click("在工作区中新建会话：rutis"); err != nil {
		t.Fatalf("点不到新建按钮：%v", err)
	}
	select {
	case got := <-stub.created:
		if got != "/work/rutis" {
			t.Fatalf("CreateSession 收到 cwd=%q，想要 /work/rutis", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("点新建没有发 CreateSession")
	}
	if app.groupCollapsed(group) {
		t.Fatal("新建后该组仍是收起的")
	}
}

// TestWorkspaceMenuPinsRenamesAndRemoves：工作区菜单的置顶 / 重命名 / 移除都有落点
// （LB-44…LB-49）；置顶把组排到最前，改名只改显示名，移除后该组与它的会话都不
// 再出现在列表里。
func TestWorkspaceMenuPinsRenamesAndRemoves(t *testing.T) {
	app := cwdGroupApp(cwdGroupPlaces())
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	// 置顶 rutis：组排到 dim-agent 之前。
	openGroupMenu(t, tt, "rutis")
	if err := tt.ChooseMenuItem("置顶工作区"); err != nil {
		t.Fatalf("点不到置顶工作区：%v", err)
	}
	tt.Frame()
	groups := app.sidebarWorkspaces(app.sidebarThreads())
	if groups[1].name != "rutis" {
		t.Fatalf("置顶后首组 = %q，想要 rutis", groups[1].name)
	}
	// 置顶的工作区进「已置顶」区（LB-22），不再在下面的分组列表里重复。
	heading, ok := tt.Find("已置顶")
	if !ok {
		t.Fatalf("置顶工作区后没有「已置顶」区：%v", tt.Texts())
	}
	if row, ok := tt.Find("rutis"); !ok || row.Y < heading.Y {
		t.Fatalf("置顶的 rutis 不在「已置顶」区里：heading=%+v row=%+v", heading, row)
	}

	// 再开菜单：首项换成取消置顶。
	openGroupMenu(t, tt, "rutis")
	if !menuHas(tt.Menu(), "取消置顶工作区") {
		t.Fatalf("已置顶的组菜单没有「取消置顶工作区」：%v", tt.Menu())
	}
	if err := tt.ChooseMenuItem("取消置顶工作区"); err != nil {
		t.Fatalf("点不到取消置顶：%v", err)
	}
	tt.Frame()
	if groups := app.sidebarWorkspaces(app.sidebarThreads()); groups[1].name != "dim-agent" {
		t.Fatalf("取消置顶后首组 = %q，想要 dim-agent", groups[1].name)
	}

	// 改名：组头换成输入框，保存后显示新名字。
	openGroupMenu(t, tt, "dim-agent")
	if err := tt.ChooseMenuItem("重命名"); err != nil {
		t.Fatalf("点不到重命名：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("工作区名称"); !ok {
		t.Fatalf("组头没有进入行内编辑：%v", tt.Texts())
	}
	app.groupRenameDraft = "评审目录"
	if err := tt.Click("保存工作区名称"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("评审目录"); !ok {
		t.Fatalf("改名后组头没有新名字：%v", tt.Texts())
	}

	// 移除：组与组内会话都不再出现。
	openGroupMenu(t, tt, "评审目录")
	if err := tt.ChooseMenuItem("移除工作区"); err != nil {
		t.Fatalf("点不到移除工作区：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("评审目录"); ok {
		t.Fatal("移除后组头还在")
	}
	if _, ok := tt.Find("评审甲"); ok {
		t.Fatal("移除后组内会话还在")
	}
	if _, ok := tt.Find("评审丙"); ok {
		t.Fatal("移除后同目录的另一条会话还在")
	}
}

// openGroupMenu 把指针挪到组头行上再点行末菜单入口（悬停才显形）。
func openGroupMenu(t *testing.T, tt *gui.Tester, name string) {
	t.Helper()
	row, ok := tt.Find(name)
	if !ok {
		t.Fatalf("找不到组头 %q：%v", name, tt.Texts())
	}
	tt.Move(row.X+row.W-20, row.Y+row.H/2)
	tt.Frame()
	if err := tt.Click("工作区菜单：" + name); err != nil {
		t.Fatalf("点不到 %s 的菜单入口：%v", name, err)
	}
	tt.Frame()
}
