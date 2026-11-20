package ui

import (
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「记忆」这一类需求的取证：哪些界面状态在切标签、切会话、关掉再开
// 之后还在，哪些按需求必须复位。每条用例对着需求表的一行（行尾注明 R-ID），
// 断言的是切换前后的可见结果，不是实现里有没有某个字段。

// TestTreeExpansionSurvivesTabSwitch：文件树在右栏内切到「改动」再回来，展开
// 态还在；收起态目录回来仍是朝右。
//
// 证据：R-RF-149、R-RF-166。
func TestTreeExpansionSurvivesTabSwitch(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	app.rp().dirOpen["assets"] = true
	tt.Frame()
	if info, ok := tt.Access("目录，assets，已展开"); !ok || !info.Expanded {
		t.Fatalf("目录没有按展开态渲染：%v", tt.Texts())
	}

	app.rpOpen(tabChanges)
	tt.Frame()
	app.rpOpen(tabFiles)
	tt.Frame()

	info, ok := tt.Access("目录，assets，已展开")
	if !ok {
		t.Fatalf("切标签再回来，展开态丢了：%v", tt.Texts())
	}
	if !info.Expanded {
		t.Fatal("切标签再回来，目录不再报展开态")
	}
	if !app.rp().dirOpen["assets"] {
		t.Fatal("切换标签清掉了目录展开表")
	}

	// 收起态目录回来仍是朝右（R-RF-166）：另找一个默认收起的目录。
	var collapsed string
	for _, node := range app.rp().tree {
		if node.dir && !app.rp().dirOpen[node.name] {
			collapsed = node.name
			break
		}
	}
	if collapsed == "" {
		t.Skip("样例树里没有第二个收起态目录")
	}
	app.rpOpen(tabChanges)
	tt.Frame()
	app.rpOpen(tabFiles)
	tt.Frame()
	if info, ok := tt.Access("目录，" + collapsed + "，已收起"); !ok {
		t.Fatalf("收起态目录 %q 回来不是收起态：%v", collapsed, tt.Texts())
	} else if info.Expanded {
		t.Fatalf("收起态目录 %q 回来报了展开态", collapsed)
	}
}

// TestTreeExpansionSurvivesPreviewRoundTrip：从文件页进预览、Esc 回到文件页，
// 树保持原展开状态。
//
// 证据：R-RF-335。
func TestTreeExpansionSurvivesPreviewRoundTrip(t *testing.T) {
	app, tt := panelTester(t, tabFiles)
	app.rp().dirOpen["assets"] = true
	tt.Frame()
	if _, ok := tt.Access("目录，assets，已展开"); !ok {
		t.Fatalf("目录没有按展开态渲染：%v", tt.Texts())
	}
	app.rp().previewPath = "AGENTS.md"
	app.rightTab = tabFilePreview
	tt.Frame()
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.rightTab != tabFiles {
		t.Fatalf("Esc 没有回到文件页：tab=%d", app.rightTab)
	}
	info, ok := tt.Access("目录，assets，已展开")
	if !ok || !info.Expanded {
		t.Fatalf("从预览回来，树的展开态丢了：%v", tt.Texts())
	}
}

// TestChangesScopeSurvivesTabSwitch：改动页的范围（未提交 / 当前分支 / Last Run）
// 在同一右栏会话内切走再回来保持上次所选。
//
// 证据：R-RD-024。
func TestChangesScopeSurvivesTabSwitch(t *testing.T) {
	app, tt := panelTester(t, tabChanges)
	if app.rp().scope != 0 {
		t.Fatalf("改动页默认范围 = %d，想要 0（未提交）", app.rp().scope)
	}
	app.rp().scope = 1
	tt.Frame()
	app.rpOpen(tabFiles)
	tt.Frame()
	app.rpOpen(tabChanges)
	tt.Frame()
	if app.rp().scope != 1 {
		t.Fatalf("切标签再回来范围回到 %d，想要仍为 1", app.rp().scope)
	}
}

// TestSessionSearchQueryDroppedOnClose：会话内搜索关掉再开，查询词清空、序号
// 从头开始。
//
// 证据：R-TB-063、R-TB-080。
func TestSessionSearchQueryDroppedOnClose(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	if app.sessionSearch.query != "排队" {
		t.Fatalf("查询没有进入浮层：%q", app.sessionSearch.query)
	}
	if app.sessionSearch.current == 0 && len(app.sessionSearch.hits) == 0 {
		t.Fatal("查询没有命中")
	}
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.sessionSearch.open {
		t.Fatal("Esc 没有关掉搜索")
	}
	openSessionSearchForTest(t, tt)
	if app.sessionSearch.query != "" {
		t.Fatalf("重开搜索沿用上次查询 %q", app.sessionSearch.query)
	}
	if app.sessionSearch.current > 0 {
		t.Fatalf("重开搜索沿用上次序号 %d", app.sessionSearch.current)
	}
	if len(app.sessionSearch.hits) != 0 {
		t.Fatalf("重开搜索沿用上次命中 %d 条", len(app.sessionSearch.hits))
	}
}

// TestViewStateLandsInPreferences：右栏页签、改动范围与文件筛选词写进视图偏好。
//
// 证据：R-TB-024、R-TB-301（落盘这一半）。
func TestViewStateLandsInPreferences(t *testing.T) {
	intents := newPrefsIntents()
	app := NewApp(Options{State: StateDefault, Intents: intents})
	app.rightOpen, app.rightTab = true, tabChanges
	app.rp().scope = 1
	app.rp().filter = "composer"
	app.persistPreferences()
	var got PreferencesSnapshot
	select {
	case got = <-intents.ch:
	case <-time.After(2 * time.Second):
		t.Fatal("没有写出偏好")
	}
	if got.View.RightTab != tabChanges {
		t.Fatalf("偏好里的右栏页签 = %d，想要 %d", got.View.RightTab, tabChanges)
	}
	if got.View.ChangesScope != 1 {
		t.Fatalf("偏好里的改动范围 = %d，想要 1", got.View.ChangesScope)
	}
	if got.View.FilesFilter != "composer" {
		t.Fatalf("偏好里的文件筛选词 = %q，想要 composer", got.View.FilesFilter)
	}
	if !got.View.RightOpen {
		t.Fatal("偏好里的右栏开合没有落盘")
	}
}

// TestRightPanelRestoresFromPreferences：启动按上次的右栏开合与页签铺回，
// 标题行按钮与显隐一致；上次关着就不自动打开。
//
// 证据：R-TB-024、R-TB-033、R-TB-301、R-TB-308。
func TestRightPanelRestoresFromPreferences(t *testing.T) {
	cases := []struct {
		name string
		view ViewPreferencesSnapshot
		open bool
		tab  int
	}{
		{"上次开着在改动页", ViewPreferencesSnapshot{RightOpen: true, RightTab: tabChanges}, true, tabChanges},
		{"上次关着", ViewPreferencesSnapshot{RightOpen: false, RightTab: tabFiles}, false, tabFiles},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := wiredSource()
			src.prefs = PreferencesSnapshot{View: c.view}
			app := NewApp(Options{State: StateDefault, Source: src})
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			tt.Frame()
			if app.rightOpen != c.open {
				t.Fatalf("右栏开合 = %v，想要 %v", app.rightOpen, c.open)
			}
			if app.rightTab != c.tab {
				t.Fatalf("右栏页签 = %d，想要 %d", app.rightTab, c.tab)
			}
			// 标题行按钮与显隐一致：开着是「关闭上下文面板」，关着是「打开上下文面板」。
			want := "打开上下文面板"
			if c.open {
				want = "关闭上下文面板"
			}
			if _, ok := tt.AccessByRole(gui.RoleButton, want); !ok {
				t.Fatalf("重启后标题行没有 %q：%v", want, tt.Texts())
			}
		})
	}
}

// TestRenameDialogDraftResetsOnReopen：重命名对话框关掉再开，草稿回到当前标题，
// 不沿用上次没提交的输入（R-TB-906 的「不沿用上次选择」口径）。
func TestRenameDialogDraftResetsOnReopen(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.openSessionRenameDialog(0)
	tt.Frame()
	if !app.renameDialog.open {
		t.Fatal("重命名对话框没有打开")
	}
	first := app.renameDialog.draft
	app.renameDialog.draft = "改到一半"
	tt.Frame()
	app.closeRenameDialog()
	tt.Frame()
	app.openSessionRenameDialog(0)
	tt.Frame()
	if got := app.renameDialog.draft; got == "改到一半" {
		t.Fatalf("重开对话框沿用了上次草稿 %q", got)
	} else if got != first {
		t.Fatalf("重开对话框的草稿 = %q，想要当前标题 %q", got, first)
	}
}
