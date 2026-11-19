package ui

import (
	"context"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「视图状态落盘」这一类的测试：左栏收起 / 视图口径 / 分组折叠 / 右栏
// 开合与标签页 / 两层筛选都进 PreferencesSnapshot，启动时按上次的样子恢复。

// viewIntents 记录写回的偏好（视图状态变化会触发一次保存）。
type viewIntents struct {
	Intents
	prefs chan PreferencesSnapshot
}

func (v *viewIntents) SetPreferences(_ context.Context, prefs PreferencesSnapshot) error {
	v.prefs <- prefs
	return nil
}

// waitPrefsOf 等一条偏好写入；视图变化在后台协程里写，这里给它两秒。
func waitPrefsOf(t *testing.T, ch chan PreferencesSnapshot) PreferencesSnapshot {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(2 * time.Second):
		t.Fatal("视图状态没有落盘")
		return PreferencesSnapshot{}
	}
}

// renameGroupDraft 走一次组头改名的落点（与菜单里的「重命名」同一路径）。
func (a *App) renameGroupDraft(ws sidebarWorkspace, name string) {
	a.renamingGroup, a.groupRenameDraft = ws.id, name
	a.renameGroup(ws)
}

// TestViewPreferencesRestoreOnStart：PreferencesSnapshot 里的视图状态在启动时
// 铺回视图（左栏收起、分组折叠、右栏标签页与范围、文件页筛选）。
func TestViewPreferencesRestoreOnStart(t *testing.T) {
	src := &fakeSource{
		conn:   Connection{Phase: "connected"},
		names:  []string{"s1", "s2"},
		titles: []string{"评审甲", "评审乙"},
		places: []SessionPlaceSnapshot{{Cwd: "/work/dim-agent"}, {Cwd: "/work/rutis"}},
		prefs: PreferencesSnapshot{
			Theme: "暗",
			View: ViewPreferencesSnapshot{
				SidebarCollapsed:    true,
				SidebarFlat:         true,
				SidebarByCreated:    true,
				CollapsedGroups:     []string{"builtin"},
				PinnedGroups:        []string{"local:/work/rutis"},
				WorkspaceNames:      map[string]string{"local:/work/rutis": "评审目录"},
				RightOpen:           true,
				RightTab:            tabChanges,
				RightTabs:           []int{tabFiles, tabChanges},
				RightExpanded:       true,
				RightWidth:          420,
				FilesFilter:         "comp",
				ChangesScope:        1,
				ChangesGroupsClosed: []string{"staged"},
			},
		},
	}
	app := NewApp(Options{State: StateDefault, Source: src})

	if !app.sidebarCollapsed {
		t.Fatal("左栏收起态没有恢复")
	}
	if !app.sidebarView.flat || !app.sidebarView.byCreated {
		t.Fatalf("视图口径没有恢复：flat=%v byCreated=%v", app.sidebarView.flat, app.sidebarView.byCreated)
	}
	if !app.sidebarView.collapsed["builtin"] {
		t.Fatalf("分组折叠态没有恢复：%+v", app.sidebarView.collapsed)
	}
	if !app.rightOpen || app.rightTab != tabChanges {
		t.Fatalf("右栏开合 / 当前页没有恢复：open=%v tab=%v", app.rightOpen, app.rightTab)
	}
	if !app.right.expanded || app.right.width != 420 {
		t.Fatalf("右栏展开 / 宽度没有恢复：%+v", app.right)
	}
	if app.right.filter != "comp" || app.right.scope != 1 {
		t.Fatalf("筛选词 / 范围没有恢复：%q %v", app.right.filter, app.right.scope)
	}
	if !app.sidebarView.pinnedGroups["local:/work/rutis"] {
		t.Fatalf("工作区置顶没有恢复：%+v", app.sidebarView.pinnedGroups)
	}
	if app.sidebarView.groupNames["local:/work/rutis"] != "评审目录" {
		t.Fatalf("工作区改名没有恢复：%+v", app.sidebarView.groupNames)
	}
	if !app.right.groupsClosed["staged"] {
		t.Fatalf("改动分组折叠态没有恢复：%+v", app.right.groupsClosed)
	}
}

// TestViewStateChangesPersist：收起左栏、切视图口径、折叠分组、切右栏标签页都
// 各写一次偏好，写出去的是当前状态（不是上一次的快照）。
func TestViewStateChangesPersist(t *testing.T) {
	src := wiredSource()
	src.names = []string{"s1", "s2"}
	src.places = []SessionPlaceSnapshot{{Cwd: "/work/dim-agent"}, {Cwd: "/work/rutis"}}
	intents := &viewIntents{prefs: make(chan PreferencesSnapshot, 8)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	tt.Key(gui.Cmd, gui.KeyB)
	tt.Frame()
	got := waitPrefsOf(t, intents.prefs)
	if !got.View.SidebarCollapsed {
		t.Fatalf("收起左栏后偏好里仍是展开：%+v", got.View)
	}

	app.sidebarView.flat = true
	tt.Frame()
	got = waitPrefsOf(t, intents.prefs)
	if !got.View.SidebarFlat {
		t.Fatalf("切到平铺后偏好里没有记：%+v", got.View)
	}

	group := app.sidebarWorkspaces(app.sidebarThreads())[1] // 目录组 dim-agent
	app.toggleGroup(group)
	tt.Frame()
	got = waitPrefsOf(t, intents.prefs)
	if len(got.View.CollapsedGroups) != 1 || got.View.CollapsedGroups[0] != group.id {
		t.Fatalf("折叠分组后偏好里没有记：%+v", got.View.CollapsedGroups)
	}

	app.toggleGroupPin(group)
	got = waitPrefsOf(t, intents.prefs)
	if len(got.View.PinnedGroups) != 1 || got.View.PinnedGroups[0] != group.id {
		t.Fatalf("置顶工作区后偏好里没有记：%+v", got.View.PinnedGroups)
	}

	app.renameGroupDraft(group, "改名后")
	got = waitPrefsOf(t, intents.prefs)
	if got.View.WorkspaceNames[group.id] != "改名后" {
		t.Fatalf("改名后偏好里没有记：%+v", got.View.WorkspaceNames)
	}

	app.rightOpen = true
	app.rightTab = tabFiles
	tt.Frame()
	got = waitPrefsOf(t, intents.prefs)
	if !got.View.RightOpen || got.View.RightTab != tabFiles {
		t.Fatalf("右栏状态没有落盘：%+v", got.View)
	}

	// 没变化的一帧不写。
	tt.Frame()
	select {
	case extra := <-intents.prefs:
		t.Fatalf("状态没变却又写了一次：%+v", extra.View)
	default:
	}
}
