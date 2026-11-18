package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 右栏两页接的是 PanelFacts 的真实读数：这里用一个注入源证明
//   - 连着读数时不再画样例；
//   - 读态（加载 / 失败 / 截断 / 权限）都来自快照；
//   - 没有行数事实时不画 ± 数字；
//   - git 写动作只画禁用项并给出原因。
// 真实文件系统与 git 的读数在 internal/uiwire 的 panel_test.go 里验。

// fakePanelSource 是实现了 Source + PanelFacts 的注入源。
type fakePanelSource struct {
	*fakeSource
	// trees 是按目录路径给的快照（"" 是根）。
	trees   map[string]FileTreeSnapshot
	search  FileSearchSnapshot
	changes ChangeSet

	lastTree   FileTreeRequest
	lastSearch FileSearchRequest
	lastChange ChangesRequest
	// searchRefreshed 记录搜索请求里出现过一次 Refresh（点了刷新 / 重试）。
	searchRefreshed bool
}

func newFakePanelSource(cwd string) *fakePanelSource {
	return &fakePanelSource{
		fakeSource: &fakeSource{
			conn:    Connection{Phase: "connected"},
			names:   []string{"s1"},
			session: SessionSnapshot{ID: "s1", Title: "接读数", Cwd: cwd},
			hasSess: true,
		},
		trees: map[string]FileTreeSnapshot{},
	}
}

func (f *fakePanelSource) FileTree(request FileTreeRequest) FileTreeSnapshot {
	f.lastTree = request
	snapshot, ok := f.trees[request.Dir]
	if !ok {
		return FileTreeSnapshot{Cwd: request.Cwd, Dir: request.Dir, Phase: PanelReadReady}
	}
	snapshot.Cwd, snapshot.Dir = request.Cwd, request.Dir
	snapshot.Offset, snapshot.Limit = request.Offset, request.Limit
	return snapshot
}

func (f *fakePanelSource) FileSearch(request FileSearchRequest) FileSearchSnapshot {
	f.lastSearch = request
	if request.Refresh {
		f.searchRefreshed = true
	}
	snapshot := f.search
	snapshot.Query = request.Query
	if snapshot.Limit == 0 {
		snapshot.Limit = request.Limit
	}
	return snapshot
}

func (f *fakePanelSource) Changes(request ChangesRequest) ChangeSet {
	f.lastChange = request
	return f.changes
}

// rightWiredTester 造一个连着注入读数、打开在指定标签上的右栏。
func rightWiredTester(t *testing.T, tab int, src *fakePanelSource) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: StateDefault, Source: src})
	app.rightOpen, app.rightTab = true, tab
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// readyTree 是一页读到的目录快照。
func readyTree(entries ...FileEntrySnapshot) FileTreeSnapshot {
	return FileTreeSnapshot{Phase: PanelReadReady, GitKnown: true,
		Entries: entries, Total: len(entries)}
}

// TestRightPanelFilesReadsPanelFacts 断言文件页画的是读数里的条目，且不是样例。
func TestRightPanelFilesReadsPanelFacts(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = readyTree(
		FileEntrySnapshot{Name: "docs", Dir: true, Change: "D"},
		FileEntrySnapshot{Name: "main.go"},
		FileEntrySnapshot{Name: "notes.md", Change: "M"},
	)
	src.trees["docs"] = readyTree(FileEntrySnapshot{Name: "guide.md"})
	app, tt := rightWiredTester(t, tabFiles, src)

	if src.lastTree.Cwd != "/work" {
		t.Fatalf("目录读的 cwd = %q，想要会话目录 /work", src.lastTree.Cwd)
	}
	for _, name := range []string{"目录，docs，已收起", "文件，notes.md，有改动", "文件，main.go"} {
		if _, ok := tt.Access(name); !ok {
			t.Fatalf("文件页缺少读数里的行 %q", name)
		}
	}
	// 样例树里的目录与文件一个都不该出现。
	for _, name := range []string{"assets", "AGENTS.md", "CHANGELOG.md"} {
		if _, ok := tt.Find(name); ok {
			t.Fatalf("连着读数时仍画了样例 %q", name)
		}
	}
	// 没有行数事实：行尾不画 +0/−0。
	for _, text := range []string{"+0", "-0"} {
		if _, ok := tt.Find(text); ok {
			t.Fatalf("没有行数事实时画了 %q", text)
		}
	}

	// 展开目录：按目录路径读下一页。
	if err := tt.Click("目录，docs，已收起"); err != nil {
		t.Fatalf("点不到目录行：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("guide.md"); !ok {
		t.Fatal("展开目录后没有读到子项")
	}
	if src.lastTree.Dir != "docs" {
		t.Fatalf("子级请求的目录 = %q，想要 docs", src.lastTree.Dir)
	}
	// 目录的改动字母来自读数（docs 聚合出 D），行仍然带着它。
	if got := app.right.dirOpen["docs"]; !got {
		t.Fatal("目录展开态没有落到视图状态上")
	}
}

// TestRightPanelFilesPaging 断言「加载更多」把分页位置加进下一次请求：
// 根目录存在 dirPages 的 "" 键上，子目录存在自己的路径上。
func TestRightPanelFilesPaging(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = FileTreeSnapshot{Phase: PanelReadReady, GitKnown: true,
		Entries: []FileEntrySnapshot{{Name: "a.go"}}, Total: 5, HasMore: true}
	app, tt := rightWiredTester(t, tabFiles, src)

	if src.lastTree.Limit != rpPageSize {
		t.Fatalf("第一页的条数 = %d，想要 %d", src.lastTree.Limit, rpPageSize)
	}
	if _, ok := tt.Find("加载更多"); !ok {
		t.Fatal("还有条目时没有「加载更多」")
	}
	if err := tt.Click("加载更多"); err != nil {
		t.Fatalf("点不到加载更多：%v", err)
	}
	tt.Frame()
	if src.lastTree.Limit != rpPageSize*2 {
		t.Fatalf("加载更多之后请求的条数 = %d，想要 %d", src.lastTree.Limit, rpPageSize*2)
	}
	if app.right.dirPages[""] != rpPageSize*2 {
		t.Fatalf("根目录的分页位置 = %d，想要 %d", app.right.dirPages[""], rpPageSize*2)
	}
}

// TestRightPanelFilesSearchStates 断言搜索四态都来自快照。
func TestRightPanelFilesSearchStates(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = readyTree(FileEntrySnapshot{Name: "main.go"})
	app, tt := rightWiredTester(t, tabFiles, src)

	// 进行中：状态行。
	src.search = FileSearchSnapshot{Phase: PanelReadLoading}
	tt.Frame()
	setFilter(app, tt, src, "token", FileSearchSnapshot{Phase: PanelReadLoading})
	if _, ok := tt.Find("正在搜索整个工作区…"); !ok {
		t.Fatal("搜索进行中时没有状态行")
	}
	if src.lastSearch.Query != "token" {
		t.Fatalf("搜索请求的关键词 = %q，想要 token", src.lastSearch.Query)
	}

	// 失败：危险色文案 + 重试。
	setFilter(app, tt, src, "token", FileSearchSnapshot{Phase: PanelReadFailed, Error: "读取失败"})
	if _, ok := tt.Find("文件搜索失败。"); !ok {
		t.Fatal("搜索失败时没有错误行")
	}
	if _, ok := tt.Find("重试"); !ok {
		t.Fatal("搜索失败时没有重试入口")
	}
	// 重试让下一次请求带上 Refresh，跳过缓存立刻重扫。
	src.searchRefreshed = false
	if err := tt.Click("重试"); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	tt.Frame()
	if !src.searchRefreshed {
		t.Fatal("点重试后的一次搜索没有要求刷新")
	}

	// 有命中：同一路径的多处命中只占一行，并给次级目录。
	setFilter(app, tt, src, "token", FileSearchSnapshot{Phase: PanelReadReady, Limit: rpSearchLimit,
		Hits: []FileHitSnapshot{
			{Path: "internal/ui/source.go", Line: 10, Snippet: "token"},
			{Path: "internal/ui/source.go", Line: 20, Snippet: "token"},
			{Path: "docs/README.md", Line: 3, Snippet: "token"},
		}})
	if got := countText(tt, "source.go"); got != 1 {
		t.Fatalf("同一路径的命中占了 %d 行，想要 1 行", got)
	}
	if _, ok := tt.Find("internal/ui"); !ok {
		t.Fatal("命中行没有次级目录")
	}
	if _, ok := tt.Find("docs"); !ok {
		t.Fatal("根下命中的目录前缀没有画出来")
	}

	// 截断：给上限提示。
	setFilter(app, tt, src, "token", FileSearchSnapshot{Phase: PanelReadReady,
		Limit: rpSearchLimit, Truncated: true, StopReason: "max-matches",
		Hits: []FileHitSnapshot{{Path: "main.go", Line: 1}}})
	if _, ok := tt.Find("仅显示前 200 个匹配项"); !ok {
		t.Fatal("截断时没有上限提示")
	}

	// 无命中：空态。
	setFilter(app, tt, src, "token", FileSearchSnapshot{Phase: PanelReadReady, Limit: rpSearchLimit})
	if _, ok := tt.Find("无匹配文件"); !ok {
		t.Fatal("无命中时没有空态")
	}
	if _, ok := tt.Find("没有找到包含“token”的文件"); !ok {
		t.Fatal("无命中时没有说明文案")
	}
}

// setFilter 换一份搜索快照并让视图进搜索态（筛选词是视图状态）。
func setFilter(app *App, tt *gui.Tester, src *fakePanelSource, query string, snapshot FileSearchSnapshot) {
	src.search = snapshot
	app.right.filter = query
	tt.Frame()
}

// countText 数一个文案在同一帧里出现了几次。
func countText(tt *gui.Tester, text string) int {
	count := 0
	for _, got := range tt.Texts() {
		if got == text {
			count++
		}
	}
	return count
}

// TestRightPanelFilesDeniedAndFailed 断言权限不足与读取失败来自快照。
func TestRightPanelFilesDeniedAndFailed(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = FileTreeSnapshot{Phase: PanelReadFailed, Denied: true, Error: "读取目录: permission denied"}
	_, tt := rightWiredTester(t, tabFiles, src)
	if _, ok := tt.Find("没有写入权限，部分操作不可用"); !ok {
		t.Fatal("权限不足时没有权限条")
	}
	if _, ok := tt.Find("无法加载此文件夹。"); !ok {
		t.Fatal("读取失败时没有错误行")
	}
	if _, ok := tt.Find("无匹配文件"); ok {
		t.Fatal("读取失败时不该画成空态")
	}
}

// TestRightPanelChangesUnavailable 断言条目级读数缺失时画「不可用」并给原因
// 与计数，而不是空列表或样例。
func TestRightPanelChangesUnavailable(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = readyTree()
	src.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main",
		Dirty: true, ChangedFileCount: 3,
		Unavailable: "git 只读口径只给改动计数，没有按文件与暂存态的条目、± 行数与 diff 文本。"}
	_, tt := rightWiredTester(t, tabChanges, src)

	if _, ok := tt.Find("改动明细不可用"); !ok {
		t.Fatal("没有画不可用态")
	}
	if _, ok := tt.Find(src.changes.Unavailable + "；Git 读到 3 个改动文件。"); !ok {
		t.Fatal("不可用态没有给出原因与计数")
	}
	for _, name := range []string{"composer.go", "timeline.go", "已暂存 (1)"} {
		if _, ok := tt.Find(name); ok {
			t.Fatalf("连着读数时仍画了样例 %q", name)
		}
	}
	if _, ok := tt.Find("没有未提交的改动"); ok {
		t.Fatal("有改动却画成了「没有改动」")
	}
}

// TestRightPanelChangesLastRunRows 断言 Last Run 范围画真实路径行，且没有
// 行数事实时不画 ± 两列。
func TestRightPanelChangesLastRunRows(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = readyTree()
	src.changes = ChangeSet{Phase: PanelReadReady, Scope: ChangeScopeLastRun, RunID: "t1",
		Note: "覆盖级别 tool_reports：只含工具结果声明的路径",
		Files: []ChangeFileSnapshot{
			{Path: "internal/ui/source.go", Exists: true},
			{Path: "gone/file.go", Status: "D"},
		}}
	app, tt := rightWiredTester(t, tabChanges, src)
	app.right.scope = scopeLastRun
	tt.Frame()

	if src.lastChange.Scope != ChangeScopeLastRun {
		t.Fatalf("改动读的范围 = %q，想要 lastRun", src.lastChange.Scope)
	}
	if _, ok := tt.Find("source.go"); !ok {
		t.Fatal("Last Run 的行没有画出来")
	}
	if _, ok := tt.Find("file.go"); !ok {
		t.Fatal("声明删除的文件没有画出来")
	}
	if _, ok := tt.Find("2 个文件改动"); !ok {
		t.Fatal("底部统计没有按行数算")
	}
	for _, text := range []string{"+0", "-0"} {
		if _, ok := tt.Find(text); ok {
			t.Fatalf("没有行数事实时画了 %q", text)
		}
	}
	// 未提交范围（没有行）回到真实空态：干净工作区就是「没有改动」。
	src.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main"}
	app.right.scope = scopeUncommitted
	tt.Frame()
	if _, ok := tt.Find("没有未提交的改动"); !ok {
		t.Fatal("干净工作区没有画空态")
	}
}

// TestRightPanelGitWritesDisabled 断言 git 写动作只出现禁用项并带原因。
func TestRightPanelGitWritesDisabled(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = readyTree(FileEntrySnapshot{Name: "notes.md", Change: "M"})
	src.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main"}
	_, tt := rightWiredTester(t, tabFiles, src)

	// 文件右键：暂存禁用并给原因。
	if err := tt.RightClick("文件，notes.md，有改动"); err != nil {
		t.Fatalf("右键文件行：%v", err)
	}
	tt.Frame()
	item, ok := tt.Access("暂存")
	if !ok {
		t.Fatal("文件菜单里没有「暂存」")
	}
	if !item.Disabled {
		t.Fatal("连着读数时「暂存」不该可点")
	}
	if _, ok := tt.Find(rpGitWriteNote); !ok {
		t.Fatal("文件菜单没有给出不可用原因")
	}
	// 点它也不该改任何东西。
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	tt.Frame()
	if got := countText(tt, "已暂存"); got > 0 {
		t.Fatal("点了禁用的「暂存」还标了已暂存")
	}
}

// TestRightPanelSubmitMenuDisabled 断言提交 / 推送 / 切分支在真实读数下禁用
// 并给原因。
func TestRightPanelSubmitMenuDisabled(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = readyTree()
	src.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main"}
	_, tt := rightWiredTester(t, tabChanges, src)

	if err := tt.Click("提交或推送"); err != nil {
		t.Fatalf("点不到提交或推送：%v", err)
	}
	tt.Frame()
	for _, label := range []string{"提交", "提交并推送", "推送"} {
		info, ok := tt.Access(label)
		if !ok {
			t.Fatalf("菜单里没有 %q", label)
		}
		if !info.Disabled {
			t.Fatalf("连着读数时 %q 不该可点", label)
		}
	}
	// 原因写在菜单项的副行里（与既有两段文案一致）。
	if got := countText(tt, rpGitWriteNote); got < 3 {
		t.Fatalf("菜单里带原因的项只有 %d 条，想要至少 3 条", got)
	}
}

// TestRightPanelOfflineBanner 断言断线时两页都在页头给原因与重连入口，
// 同时仍画本机读数（本机读不依赖连接）。
func TestRightPanelOfflineBanner(t *testing.T) {
	cases := []struct {
		tab   int
		proof string // 本机读数里必然出现的一行
	}{
		{tab: tabFiles, proof: "文件，notes.md"},
		{tab: tabChanges, proof: "没有未提交的改动"},
	}
	for _, tc := range cases {
		src := newFakePanelSource("/work")
		src.conn = Connection{Phase: "disconnected", Reason: "socket closed"}
		src.trees[""] = readyTree(FileEntrySnapshot{Name: "notes.md"})
		src.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main"}
		_, tt := rightWiredTester(t, tc.tab, src)
		if _, ok := tt.Find("连接已断开：socket closed"); !ok {
			t.Fatalf("标签 %d 没有画断线原因", tc.tab)
		}
		if _, ok := tt.Find("重新连接"); !ok {
			t.Fatalf("标签 %d 没有重连入口", tc.tab)
		}
		if _, ok := tt.Find(tc.proof); !ok {
			t.Fatalf("标签 %d 断线后没有画本机读数（缺 %q）", tc.tab, tc.proof)
		}
	}
}

// TestCommandPaletteReadsWiredTree 断言命令面板的文件组读到的是右栏的目录投影
// （连着读数时是真文件，不是样例）。
func TestCommandPaletteReadsWiredTree(t *testing.T) {
	src := newFakePanelSource("/work")
	src.trees[""] = readyTree(
		FileEntrySnapshot{Name: "notes.md", Change: "M"},
		FileEntrySnapshot{Name: "main.go"},
	)
	app := NewApp(Options{State: StateDefault, Source: src})
	app.openCommandPalette()
	// 面板开着、右栏关着时也要有投影：面板读的就是这份。
	_ = gui.NewTester(app.View, WindowWidth, WindowHeight)

	got := map[string]bool{}
	for _, item := range app.commandPaletteFiles() {
		got[item.title] = true
	}
	for _, want := range []string{"notes.md", "main.go"} {
		if !got[want] {
			t.Fatalf("命令面板的文件组缺少读数里的 %q（拿到 %v）", want, got)
		}
	}
	for _, sample := range []string{"AGENTS.md", "fib57.py", "assets"} {
		if got[sample] {
			t.Fatalf("命令面板的文件组仍含样例 %q", sample)
		}
	}
}
