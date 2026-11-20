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

// TestSidebarGroupCollapseIsViewState：折叠记在视图状态里并随偏好落盘（重启后按
// 收起态渲染），点组头收起组内会话、再点展开。
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
		if len(wrote.View.CollapsedGroups) != 1 || wrote.View.CollapsedGroups[0] != group.id {
			t.Fatalf("折叠态没有随偏好落盘：%+v", wrote.View.CollapsedGroups)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("折叠态没有落盘")
	}

	if err := tt.Click("dim-agent"); err != nil {
		t.Fatalf("再点组头：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("评审甲"); !ok {
		t.Fatal("再点后组内会话没有回来")
	}
}

// TestChatSearchFindsSessionsAcrossWorkspaceGroups：跨会话检索不按工作区分组
// 切分——一条查询能把两个目录组里的会话一起找出来（G-A2-01 的「跨会话」）。
func TestChatSearchFindsSessionsAcrossWorkspaceGroups(t *testing.T) {
	app := cwdGroupApp(cwdGroupPlaces())
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Key(gui.Cmd, gui.KeyG)
	tt.Frame()
	if !app.chatSearch.open {
		t.Fatal("⌘G 没有打开搜索聊天对话框")
	}
	if len(app.chatSearch.rows) != 3 {
		t.Fatalf("空查询的结果 = %d 条，想要三条会话", len(app.chatSearch.rows))
	}
	if err := tt.Click(chatSearchLabel); err != nil {
		t.Fatalf("点不到搜索聊天输入框：%v", err)
	}
	tt.Type("评审")
	tt.Frame()
	for _, title := range []string{"评审甲", "评审乙", "评审丙"} {
		if !tt.HasText(title) {
			t.Fatalf("跨组命中 %q 没有出现在对话框里：\n%v", title, tt.Texts())
		}
	}
	// 副行是各自的工作区显示名（目录末段），跨组的归属不同。
	places := map[string]bool{}
	for _, row := range app.chatSearch.rows {
		places[row.place] = true
	}
	if !places["dim-agent"] || !places["rutis"] {
		t.Fatalf("结果的工作区归属 = %v，想要 dim-agent 与 rutis 都在", places)
	}

	// 只命中一组：另一组的会话与它无关，不在结果里。
	if err := tt.Click(chatSearchLabel); err != nil {
		t.Fatalf("点不到搜索聊天输入框：%v", err)
	}
	tt.Command("selectAll")
	tt.Type("丙")
	tt.Frame()
	if len(app.chatSearch.rows) != 1 || app.chatSearch.rows[0].title != "评审丙" {
		t.Fatalf("过滤后 = %+v", app.chatSearch.rows)
	}
}
