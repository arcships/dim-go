package ui

import (
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是左栏按会话工作目录（cwd）分组的测试：三条会话两个目录 → 两个组、组内
// 顺序、组的排序、折叠只进可丢的视图状态、搜索命中跨组显示并自动展开。

// cwdGroupApp 建一个"连着 Source"的左栏：三条会话分布在两个目录（dim-agent 两条、
// rutis 一条），显示名取标题，便于按名字找行。
func cwdGroupApp(places []SessionPlaceSnapshot) *App {
	return NewApp(Options{
		State: StateDefault,
		Source: &fakeSource{
			conn:    Connection{Phase: "connected"},
			names:   []string{"s1", "s2", "s3"},
			titles:  []string{"评审甲", "评审乙", "评审丙"},
			places:  places,
			session: SessionSnapshot{ID: "s1", Title: "评审甲"},
			hasSess: true,
		},
	})
}

// cwdGroupPlaces 是三条会话两个目录的归属：s1、s3 在 dim-agent，s2 在 rutis。
func cwdGroupPlaces() []SessionPlaceSnapshot {
	return []SessionPlaceSnapshot{
		{Cwd: "/work/dim-agent", UpdatedAt: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)},
		{Cwd: "/work/rutis", UpdatedAt: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)},
		{Cwd: "/work/dim-agent", UpdatedAt: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)},
	}
}

// threadNames 取一组会话行的显示名，按渲染顺序。
func threadNames(threads []sidebarThread) []string {
	out := make([]string, 0, len(threads))
	for _, thread := range threads {
		out = append(out, thread.name)
	}
	return out
}

// TestSidebarGroupsBySessionCwd：三条会话两个 cwd → 两个目录组（组名取目录末段），
// 每条会话归到自己的组、组内沿用目录顺序；没有 cwd 的会话归「工作区」。
func TestSidebarGroupsBySessionCwd(t *testing.T) {
	app := cwdGroupApp(cwdGroupPlaces())
	all := app.sidebarThreads()
	groups := app.sidebarWorkspaces(all)

	if len(groups) != 3 {
		t.Fatalf("分组数 = %d，想要 3（工作区 + 两个目录）：%+v", len(groups), groups)
	}
	if !groups[0].builtin || groups[0].name != sidebarBuiltinName {
		t.Fatalf("第一组 = %+v，想要内置「工作区」", groups[0])
	}
	want := []struct {
		name  string
		count int
	}{
		{"dim-agent", 2},
		{"rutis", 1},
	}
	for i, w := range want {
		if groups[i+1].name != w.name || groups[i+1].count != w.count {
			t.Fatalf("第 %d 组 = %+v，想要 %s（%d 条）", i+1, groups[i+1], w.name, w.count)
		}
	}

	// 组内顺序：dim-agent 是 s1、s3，rutis 是 s2。
	if got := threadNames(app.sidebarGroupThreads(groups[1], all)); len(got) != 2 ||
		got[0] != "评审甲" || got[1] != "评审丙" {
		t.Fatalf("dim-agent 组内顺序 = %v，想要 [评审甲 评审丙]", got)
	}
	if got := threadNames(app.sidebarGroupThreads(groups[2], all)); len(got) != 1 || got[0] != "评审乙" {
		t.Fatalf("rutis 组内顺序 = %v，想要 [评审乙]", got)
	}
	if got := app.sidebarGroupThreads(groups[0], all); len(got) != 0 {
		t.Fatalf("没有 cwd 的会话时「工作区」仍有 %v", threadNames(got))
	}

	// 屏幕上两组都在，会话行在各自的组里。
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	for _, text := range []string{"dim-agent", "rutis", "评审甲", "评审乙", "评审丙"} {
		if _, ok := tt.Find(text); !ok {
			t.Fatalf("左栏没有出现 %q：\n%v", text, tt.Texts())
		}
	}
	if header, ok := tt.Find("dim-agent"); ok {
		if row, ok := tt.Find("评审甲"); ok && row.Y < header.Y {
			t.Fatalf("dim-agent 组头 %+v 排在自己的会话行 %+v 之后", header, row)
		}
	}
}

// TestSidebarSessionWithoutCwdFallsBackToWorkspace：没有 cwd 的会话归「工作区」，
// 有 cwd 的另成一组。
func TestSidebarSessionWithoutCwdFallsBackToWorkspace(t *testing.T) {
	app := cwdGroupApp([]SessionPlaceSnapshot{
		{},
		{Cwd: "/work/dim-agent"},
		{Cwd: ""},
	})
	all := app.sidebarThreads()
	groups := app.sidebarWorkspaces(all)
	if len(groups) != 2 {
		t.Fatalf("分组数 = %d，想要 2（工作区 + dim-agent）", len(groups))
	}
	if groups[0].count != 2 || groups[1].count != 1 || groups[1].name != "dim-agent" {
		t.Fatalf("分组 = %+v，想要工作区 2 条 + dim-agent 1 条", groups)
	}
	if got := threadNames(app.sidebarGroupThreads(groups[0], all)); len(got) != 2 ||
		got[0] != "评审甲" || got[1] != "评审丙" {
		t.Fatalf("「工作区」组内顺序 = %v，想要 [评审甲 评审丙]", got)
	}
}

// TestSidebarGroupOrderByActivity：会话数相同时按最近活动时间排，晚的在前。
func TestSidebarGroupOrderByActivity(t *testing.T) {
	app := NewApp(Options{
		State: StateDefault,
		Source: &fakeSource{
			conn:   Connection{Phase: "connected"},
			names:  []string{"s1", "s2"},
			titles: []string{"旧会话", "新会话"},
			places: []SessionPlaceSnapshot{
				{Cwd: "/work/old", UpdatedAt: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)},
				{Cwd: "/work/new", UpdatedAt: time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)},
			},
		},
	})
	groups := app.sidebarWorkspaces(app.sidebarThreads())
	if len(groups) != 3 {
		t.Fatalf("分组数 = %d，想要 3（工作区 + 两个目录）", len(groups))
	}
	if groups[1].name != "new" || groups[2].name != "old" {
		t.Fatalf("同数会话的组顺序 = %s, %s，想要 new, old", groups[1].name, groups[2].name)
	}
}

// TestSidebarGroupCollapseIsViewState：折叠只记在视图状态里（不落盘），点组头收起
// 组内会话、再点展开。
func TestSidebarGroupCollapseIsViewState(t *testing.T) {
	prefs := make(chan PreferencesSnapshot, 4)
	app := cwdGroupApp(cwdGroupPlaces())
	app.intents = &dirPickIntents{prefs: prefs}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	group := app.sidebarWorkspaces(app.sidebarThreads())[1]

	// 有会话的组默认展开。
	if app.groupCollapsed(group) {
		t.Fatal("有会话的组默认被折叠")
	}
	if _, ok := tt.Find("评审甲"); !ok {
		t.Fatalf("dim-agent 组没有露出来：\n%v", tt.Texts())
	}

	if err := tt.Click("dim-agent"); err != nil {
		t.Fatalf("点组头：%v", err)
	}
	tt.Frame()
	if !app.groupCollapsed(group) {
		t.Fatal("点组头后仍是展开态")
	}
	if _, ok := tt.Find("评审甲"); ok {
		t.Fatal("折叠后组内会话还在屏幕上")
	}
	if got := app.sidebarView.collapsed[group.id]; !got {
		t.Fatalf("折叠态没有记进视图状态：%+v", app.sidebarView.collapsed)
	}
	select {
	case wrote := <-prefs:
		t.Fatalf("折叠态落了盘：%+v", wrote)
	default:
	}

	if err := tt.Click("dim-agent"); err != nil {
		t.Fatalf("再点组头：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("评审甲"); !ok {
		t.Fatal("再点后组内会话没有回来")
	}
}

// TestSidebarSearchShowsHitsAcrossGroupsAndExpands：搜索命中时跨组显示，组自动
// 展开；查询清空后回到原来的折叠态。
func TestSidebarSearchShowsHitsAcrossGroupsAndExpands(t *testing.T) {
	app := cwdGroupApp(cwdGroupPlaces())
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	groups := app.sidebarWorkspaces(app.sidebarThreads())
	if app.sidebarView.collapsed == nil {
		app.sidebarView.collapsed = map[string]bool{}
	}
	for _, group := range groups[1:] {
		app.sidebarView.collapsed[group.id] = true
	}
	tt.Frame()
	if _, ok := tt.Find("评审甲"); ok {
		t.Fatal("搜索前两组应处于折叠态")
	}

	app.searchOpen, app.searchQuery = true, "评审"
	tt.Frame()
	// 命中跨组显示，且折叠住的组被自动展开。
	for _, name := range []string{"评审甲", "评审乙", "评审丙"} {
		if _, ok := tt.Find(name); !ok {
			t.Fatalf("搜索命中 %q 没有跨组显示：\n%v", name, tt.Texts())
		}
	}
	if _, ok := tt.Find("暂无会话"); ok {
		t.Fatalf("搜索时仍画了没有命中的空组：\n%v", tt.Texts())
	}
	for _, group := range groups[1:] {
		if !app.sidebarView.collapsed[group.id] {
			t.Fatalf("%s 的折叠态被搜索改写了", group.name)
		}
	}

	// 只命中一组的查询：另一组的行与组头都不出现。
	app.searchQuery = "乙"
	tt.Frame()
	if _, ok := tt.Find("评审甲"); ok {
		t.Fatal("只命中一组时仍显示了别组的行")
	}
	if _, ok := tt.Find("dim-agent"); ok {
		t.Fatal("没有命中的组头还在")
	}
	if _, ok := tt.Find("评审乙"); !ok {
		t.Fatal("命中的行没出现")
	}

	// 清空查询：折叠态照旧。
	app.searchQuery = ""
	tt.Frame()
	if _, ok := tt.Find("评审甲"); ok {
		t.Fatal("清空查询后折叠态没有恢复")
	}
}
