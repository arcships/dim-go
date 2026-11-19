package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 窄窗档（< CompactWidth，验收档 720）的七处形态差异，每处一条断言：左栏
// 只留边条 + 覆盖式抽屉、右栏整屏替代、标题名截断、控制行竖排、设置导航
// 改下拉、命令面板满宽、对话框满宽。判据只看窄窗形态；1440 档的既有断言
// 在 app_test.go，不在这里重复。

// narrowWidth 是验收档：scripts/capture-windows.py 的第三档。
const narrowWidth = 720

// newNarrowTester 建一个 720 宽的窗口。
func newNarrowTester(t *testing.T, s State) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: s})
	return app, gui.NewTester(app.View, narrowWidth, WindowHeight)
}

// 窄窗左栏只留 12px 边条；点边条展开的是覆盖式抽屉（256 宽），主区仍在原处
// 不重排（覆盖式，不是挤压式）；Esc 收起。
func TestNarrowSidebarRailAndDrawer(t *testing.T) {
	app, tt := newNarrowTester(t, StateDefault)

	rail, ok := tt.Find("展开侧边栏 ⌘B")
	if !ok {
		t.Fatal("窄窗没有左栏边条")
	}
	if rail.W != SidebarRailWidth {
		t.Fatalf("边条宽 %v，想要 %v", rail.W, SidebarRailWidth)
	}
	if tt.HasText("左栏 · 会话列表") {
		t.Fatal("窄窗默认不应展开 256 宽的左栏")
	}

	main, ok := tt.Find("对话流")
	if !ok {
		t.Fatal("找不到对话流")
	}
	if main.X != SidebarRailWidth {
		t.Fatalf("主区左缘 %v，想要贴着边条 %v", main.X, SidebarRailWidth)
	}
	wideMain := main.W

	if err := tt.Click("展开侧边栏 ⌘B"); err != nil {
		t.Fatalf("点不到边条：%v", err)
	}
	tt.Frame()
	if !app.narrowDrawer {
		t.Fatal("窄窗里点边条没有打开抽屉")
	}
	drawer, ok := tt.Find("左栏 · 会话列表")
	if !ok {
		t.Fatal("抽屉没有出现")
	}
	if drawer.W != SidebarWidth {
		t.Fatalf("抽屉宽 %v，想要 %v", drawer.W, SidebarWidth)
	}
	if drawer.X != 0 {
		t.Fatalf("抽屉左缘 %v，想要贴着窗口左边 0", drawer.X)
	}
	// 覆盖式：主区尺寸不变（不因抽屉展开而收窄）。
	after, ok := tt.Find("对话流")
	if !ok {
		t.Fatal("抽屉展开后主区消失了")
	}
	if after.W != wideMain {
		t.Fatalf("抽屉展开后主区宽 %v，展开前 %v：抽屉应是覆盖式", after.W, wideMain)
	}
	if app.sidebarCollapsed {
		t.Fatal("窄窗抽屉不应改动左栏的常驻偏好")
	}

	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.narrowDrawer {
		t.Fatal("Esc 没有收起抽屉")
	}
	if tt.HasText("左栏 · 会话列表") {
		t.Fatal("抽屉收起后左栏还在")
	}
}

// 窄窗右栏整屏替代主区：不并排、不缩到 400；面板占满边条右侧的整个宽度。
func TestNarrowRightPanelReplacesMain(t *testing.T) {
	app, tt := newNarrowTester(t, StateDefault)
	if err := tt.Click("打开上下文面板"); err != nil {
		t.Fatalf("点不到右栏入口：%v", err)
	}
	tt.Frame()
	if !app.rightOpen {
		t.Fatal("右栏没有打开")
	}
	panel, ok := tt.Find("右栏")
	if !ok {
		t.Fatal("找不到右栏")
	}
	want := float32(narrowWidth - SidebarRailWidth)
	if panel.W != want {
		t.Fatalf("窄窗右栏宽 %v，想要 %v（整屏替代）", panel.W, want)
	}
	if panel.X != SidebarRailWidth {
		t.Fatalf("右栏左缘 %v，想要 %v", panel.X, SidebarRailWidth)
	}
	if _, ok := tt.Find("对话流"); ok {
		t.Fatal("窄窗右栏打开时主区应与它同一时刻只显示一处")
	}
}

// 窄窗标题行去掉工作区前缀，会话名单行截断到 narrowTitleMaxWidth，全名仍在
// 读屏名里；面板开关不被挤出窗口。
func TestNarrowHeaderTruncatesSessionName(t *testing.T) {
	long := "把排队判定挪到写通道之后再重放一次的重构会话"
	src := wiredSource()
	src.session.Title = long
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, narrowWidth, WindowHeight)

	if tt.HasText("工作区 " + app.workspaceTitle()) {
		t.Fatalf("窄窗标题行仍写着工作区前缀：%v", tt.Texts())
	}
	title, ok := tt.Find("会话 " + long)
	if !ok {
		t.Fatalf("找不到会话名（读屏名应带全名）：%v", tt.Texts())
	}
	if title.W > narrowTitleMaxWidth {
		t.Fatalf("会话名宽 %v，超过窄窗上限 %v（应单行截断）", title.W, narrowTitleMaxWidth)
	}
	toggle, ok := tt.Find("关闭上下文面板")
	if !ok {
		toggle, ok = tt.Find("打开上下文面板")
	}
	if !ok {
		t.Fatal("找不到面板开关")
	}
	if title.X+title.W > toggle.X {
		t.Fatalf("会话名右缘 %.1f 压到面板开关左缘 %.1f", title.X+title.W, toggle.X)
	}
	if toggle.X+toggle.W > narrowWidth {
		t.Fatalf("面板开关右缘 %.1f 越出窗口 %d", toggle.X+toggle.W, narrowWidth)
	}
}

// 窄窗标题行让开窗口控件：左栏在窄窗里只剩 12px 边条，红绿灯会压到标题行，
// 标题行左内边距必须取 TitleBar().Left，会话名不能画在控件上。
func TestNarrowHeaderClearsWindowControls(t *testing.T) {
	app, tt := newNarrowTester(t, StateDefault)
	tt.SetTitleBar(gui.TitleBar{Height: 28, Left: 78})
	tt.Frame()
	title, ok := tt.Find("会话 " + app.sessionTitle())
	if !ok {
		t.Fatalf("找不到会话名：%v", tt.Texts())
	}
	if title.X < 78 {
		t.Fatalf("会话名左缘 %v 落在窗口控件里（控件让位 78）", title.X)
	}
}

// 窄窗输入区控制行竖排：工作区一行、权限一行，两行左缘相同。
func TestNarrowComposerControlsStack(t *testing.T) {
	app, tt := newNarrowTester(t, StateEmpty)
	workspace, ok := tt.Find("选择工作区")
	if !ok {
		t.Fatal("找不到工作区选择器")
	}
	permission, ok := tt.Find("权限：" + app.permissionLabel())
	if !ok {
		t.Fatalf("找不到权限选择器：%v", tt.Texts())
	}
	if workspace.X != permission.X {
		t.Fatalf("两行左缘不同：工作区 %v、权限 %v", workspace.X, permission.X)
	}
	if permission.Y < workspace.Y+workspace.H {
		t.Fatalf("两个选择器仍并排：工作区 %+v、权限 %+v", workspace, permission)
	}
}

// 窄窗设置导航收成顶部下拉：不再画 224 的常驻列，下拉占满一行；从中选一节能
// 切到该分节。
func TestNarrowSettingsNavDropdown(t *testing.T) {
	app, tt := newNarrowTester(t, StateDefault)
	tt.Key(gui.Cmd, gui.KeyComma)
	tt.Frame()
	if app.screen != ScreenSettings {
		t.Fatal("⌘, 没有打开设置")
	}
	trigger, ok := tt.Find("设置分节")
	if !ok {
		t.Fatalf("窄窗设置没有分节下拉：%v", tt.Texts())
	}
	if trigger.W < SettingsNavWidth {
		t.Fatalf("分节下拉宽 %v，应占满一行（窄窗没有 224 的常驻导航）", trigger.W)
	}
	if err := tt.Click("设置分节"); err != nil {
		t.Fatalf("点不到分节下拉：%v", err)
	}
	tt.Frame()
	if err := tt.Click("归档会话"); err != nil {
		t.Fatalf("点不到分节选项：%v", err)
	}
	tt.Frame()
	if app.settingsSection != secArchived {
		t.Fatalf("选「归档会话」后分节是 %v，想要 %v", app.settingsSection, secArchived)
	}
}

// 窄窗命令面板改满宽、左右留 narrowDialogMargin。
func TestNarrowCommandPaletteFullWidth(t *testing.T) {
	app, tt := newNarrowTester(t, StateDefault)
	tt.Key(gui.Cmd, gui.KeyK)
	if !app.palette.open {
		t.Fatal("Cmd+K 没有打开命令面板")
	}
	tt.Frame()
	panel, ok := tt.Find("命令面板")
	if !ok {
		t.Fatal("找不到命令面板")
	}
	want := float32(narrowWidth) - 2*narrowDialogMargin
	if panel.W != want {
		t.Fatalf("命令面板宽 %v，想要 %v", panel.W, want)
	}
	if panel.X != narrowDialogMargin {
		t.Fatalf("命令面板左缘 %v，想要 %v", panel.X, narrowDialogMargin)
	}
}

// 窄窗对话框改满宽、左右留边距；1440 档仍是原宽（这里用重命名弹窗对照）。
func TestNarrowDialogsFullWidth(t *testing.T) {
	title := renameCopySessionTitle
	cases := []struct {
		width int
		want  float32
	}{
		{narrowWidth, narrowWidth - 2*narrowDialogMargin},
		{WindowWidth, 399},
	}
	for _, tc := range cases {
		app := NewApp(Options{State: StateDefault, Source: wiredSource()})
		tt := gui.NewTester(app.View, tc.width, WindowHeight)
		openRenameFromHeader(t, tt)
		panel, ok := tt.Find(title)
		if !ok {
			t.Fatalf("%d 宽：找不到重命名弹窗", tc.width)
		}
		if panel.W != tc.want {
			t.Fatalf("%d 宽：弹窗宽 %v，想要 %v", tc.width, panel.W, tc.want)
		}
		if tc.width == narrowWidth && panel.X != narrowDialogMargin {
			t.Fatalf("窄窗弹窗左缘 %v，想要 %v", panel.X, narrowDialogMargin)
		}
	}
}

// 分档边界：CompactWidth 上仍是常驻左栏 + 横排控制行；少一像素才切窄窗。
func TestCompactWidthBoundary(t *testing.T) {
	for _, tc := range []struct {
		width    int
		sidebarW float32
	}{
		{CompactWidth, SidebarWidth},
		{CompactWidth - 1, SidebarRailWidth},
	} {
		app := NewApp(Options{State: StateEmpty})
		tt := gui.NewTester(app.View, tc.width, WindowHeight)
		label := "左栏 · 会话列表"
		if tc.sidebarW == SidebarRailWidth {
			label = "展开侧边栏 ⌘B"
		}
		bar, ok := tt.Find(label)
		if !ok {
			t.Fatalf("%d 宽：找不到 %q：%v", tc.width, label, tt.Texts())
		}
		if bar.W != tc.sidebarW {
			t.Fatalf("%d 宽：左栏宽 %v，想要 %v", tc.width, bar.W, tc.sidebarW)
		}
		workspace, ok := tt.Find("选择工作区")
		if !ok {
			t.Fatalf("%d 宽：找不到工作区选择器", tc.width)
		}
		permission, ok := tt.Find("权限：" + app.permissionLabel())
		if !ok {
			t.Fatalf("%d 宽：找不到权限选择器", tc.width)
		}
		stacked := permission.Y >= workspace.Y+workspace.H
		if wantStacked := tc.sidebarW == SidebarRailWidth; stacked != wantStacked {
			t.Fatalf("%d 宽：控制行竖排=%v，想要 %v", tc.width, stacked, wantStacked)
		}
	}
}

// 设置 › 权限的规则表头只在三列并排时出现（U2-forms F23）：720 档内容列 664
// 够三列，表头在；500 档内容列 444 < 512，规则行按行堆叠，表头不画。
func TestNarrowPermissionRuleTableHeader(t *testing.T) {
	for _, tc := range []struct {
		width      int
		wantHeader bool
	}{
		{narrowWidth, true},
		{500, false},
	} {
		app := NewApp(Options{State: StateDefault})
		app.screen = ScreenSettings
		app.settingsSection = secPermissions
		tt := gui.NewTester(app.View, tc.width, WindowHeight)
		_, ok := tt.Find("命令规则表头")
		if ok != tc.wantHeader {
			t.Fatalf("%d 宽：规则表头出现=%v，想要 %v", tc.width, ok, tc.wantHeader)
		}
		if !tc.wantHeader {
			continue
		}
		command, ok1 := tt.Find(permCopyColCmd)
		match, ok2 := tt.Find(permCopyColMatch)
		decision, ok3 := tt.Find(permCopyColAction)
		if !ok1 || !ok2 || !ok3 {
			t.Fatalf("%d 宽：三列表头缺格：%v %v %v", tc.width, ok1, ok2, ok3)
		}
		if command.Y != match.Y || decision.Y != match.Y {
			t.Fatalf("%d 宽：三列表头不在同一行：%v %v %v", tc.width, command.Y, match.Y, decision.Y)
		}
	}
}
