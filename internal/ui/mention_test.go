package ui

import (
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// `@` 文件提及与 `$` 技能提及弹层（requirements-global §2.3、§2.4）的行为测试：
// 每一步都在输入区里键入，断言落在弹层的候选、选中行与插进草稿的文本上。

// mentionTester 把焦点放进输入区，返回一个空的输入区。
func mentionTester(t *testing.T) (*App, *gui.Tester) {
	t.Helper()
	tc := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(tc.View, WindowWidth, WindowHeight)
	if err := tt.Click("消息"); err != nil {
		t.Fatalf("点输入区：%v", err)
	}
	return tc, tt
}

// typeDraft 直接在输入区键入（触发位由键入造成，不靠改状态）。
func typeDraft(t *testing.T, tt *gui.Tester, text string) {
	t.Helper()
	tt.Type(text)
	tt.Frame()
}

// 键入 `@` 弹出文件候选：分组「文件」在场，候选来自已读到的目录投影；`@` 在
// 行中同样触发（G-A3-01）。
func TestFilePickOpensOnAtSign(t *testing.T) {
	app, tt := mentionTester(t)
	typeDraft(t, tt, "@")
	if !app.pick.open || app.pick.kind != pickFile {
		t.Fatalf("键入 @ 没有打开文件弹层：open=%v kind=%v", app.pick.open, app.pick.kind)
	}
	if !tt.HasText("候选分组：" + pickGroupFile) {
		t.Fatalf("缺文件分组标题：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if len(app.pick.files) == 0 {
		t.Fatalf("没有文件候选：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	// 目录名以 '/' 收尾，文件不带：两者可区分（G-A3-05）。
	var dirs, files int
	for _, item := range app.pick.files {
		if item.dir {
			dirs++
			continue
		}
		files++
	}
	if dirs == 0 || files == 0 {
		t.Fatalf("候选里目录 %d 条、文件 %d 条，想要两类都有", dirs, files)
	}
}

// 行中的 `@` 也触发（不限行首，G-A3-01 的「@ 在行首 / 行中」）。
func TestFilePickTriggersMidLine(t *testing.T) {
	app, tt := mentionTester(t)
	typeDraft(t, tt, "看一下 @comp")
	if !app.pick.open {
		t.Fatalf("行中的 @ 没有触发弹层：draft=%q", app.draft)
	}
	if app.pick.query != "comp" {
		t.Fatalf("前缀 = %q，想要 comp", app.pick.query)
	}
}

// 输入过滤：`@comp` 只留路径里含 comp 的候选，`@zzz` 给空态「未找到文件或目录」
// （G-A3-03）。
func TestFilePickFiltersAndEmptyState(t *testing.T) {
	app, tt := mentionTester(t)
	typeDraft(t, tt, "@comp")
	if !app.pick.open {
		t.Fatalf("弹层没有打开：draft=%q", app.draft)
	}
	var found bool
	for _, item := range app.pick.files {
		if item.name == "composer.go" {
			found = true
			if item.nameStart < 0 || item.name[item.nameStart:item.nameEnd] != "comp" {
				t.Fatalf("主字没有命中区间：%+v", item)
			}
		}
	}
	if !found {
		t.Fatalf("过滤后没有 composer.go：%+v", app.pick.files)
	}
	for _, item := range app.pick.files {
		if !strings.Contains(strings.ToLower(item.path), "comp") {
			t.Fatalf("候选 %q 不含查询词", item.path)
		}
	}

	typeDraft(t, tt, "zzz")
	if len(app.pick.items) != 0 {
		t.Fatalf("无命中时仍有候选：%+v", app.pick.items)
	}
	if !tt.HasText(pickNoFileText) {
		t.Fatalf("缺空态文案 %q：\n%s", pickNoFileText, strings.Join(tt.Texts(), "\n"))
	}
	if !app.pick.open {
		t.Fatal("空态出现时弹层不该关闭")
	}
}

// ↑↓ 选择、回车插入：把 `@前缀` 换成 `@路径加空格` 并收起（G-A3-11）。
func TestFilePickArrowAndEnterInserts(t *testing.T) {
	app, tt := mentionTester(t)
	typeDraft(t, tt, "@internal")
	if len(app.pick.items) < 2 {
		t.Fatalf("候选不足两条：%+v", app.pick.items)
	}
	if app.pick.selected != 0 {
		t.Fatalf("默认选中行 = %d，想要第 0 行", app.pick.selected)
	}
	tt.Key(0, gui.KeyDown)
	tt.Frame()
	if app.pick.selected != 1 {
		t.Fatalf("↓ 之后选中行 = %d，想要 1", app.pick.selected)
	}
	// 选中的是一条文件（目录插完会留在弹层里，见下一个用例）。
	fileIndex := -1
	for i, item := range app.pick.items {
		if !item.dir {
			fileIndex = i
			break
		}
	}
	if fileIndex < 0 {
		t.Fatalf("没有文件候选：%+v", app.pick.items)
	}
	for app.pick.selected < fileIndex {
		tt.Key(0, gui.KeyDown)
		tt.Frame()
	}
	want := app.pick.items[fileIndex]
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.pick.open {
		t.Fatalf("回车之后弹层没有收起：draft=%q", app.draft)
	}
	if !strings.HasPrefix(app.draft, want.insert) {
		t.Fatalf("插入后草稿 = %q，想要以 %q 开头", app.draft, want.insert)
	}
	if strings.HasPrefix(app.draft, "@internal ") || !strings.HasPrefix(app.draft, "@") {
		t.Fatalf("触发位与前缀没有被替换：%q", app.draft)
	}
}

// 目录候选插完仍留在弹层里，可以接着追问子目录（G-A3-05 的「可继续追问子目录」）。
func TestFilePickDirectoryStaysOpen(t *testing.T) {
	app, tt := mentionTester(t)
	typeDraft(t, tt, "@internal")
	// 选中第一个目录候选（目录排在文件之前）。
	var dirIndex = -1
	for i, item := range app.pick.items {
		if item.dir {
			dirIndex = i
			break
		}
	}
	if dirIndex < 0 {
		t.Fatalf("没有目录候选：%+v", app.pick.items)
	}
	for app.pick.selected < dirIndex {
		tt.Key(0, gui.KeyDown)
		tt.Frame()
	}
	want := app.pick.items[dirIndex]
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if !app.pick.open {
		t.Fatalf("插入目录后弹层收起了：draft=%q", app.draft)
	}
	if !strings.HasSuffix(strings.TrimSpace(app.draft), want.insert) {
		t.Fatalf("草稿 = %q，想要以 %q 收尾", app.draft, want.insert)
	}
	if !strings.HasSuffix(app.pick.query, "/") {
		t.Fatalf("插入目录后的前缀 = %q，想要以 / 收尾", app.pick.query)
	}
}

// Esc 收起弹层，草稿里的 `@前缀` 保留；同样的前缀不再自动弹回，再键入字符回来
// （G-A3-01 的「输入空格 / Esc」）。
func TestFilePickEscapeDismisses(t *testing.T) {
	app, tt := mentionTester(t)
	typeDraft(t, tt, "@comp")
	if !app.pick.open {
		t.Fatal("弹层没有打开")
	}
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.pick.open {
		t.Fatal("Esc 没有收起弹层")
	}
	if app.draft != "@comp" {
		t.Fatalf("Esc 改了草稿：%q", app.draft)
	}
	// 同样的前缀：不再自动弹出。
	tt.Frame()
	if app.pick.open {
		t.Fatal("Esc 之后同样的前缀又弹回来了")
	}
	// 继续键入：候选取新前缀，弹层回来。
	typeDraft(t, tt, "o")
	if !app.pick.open || app.pick.query != "compo" {
		t.Fatalf("继续键入后没有回来：open=%v query=%q", app.pick.open, app.pick.query)
	}
}

// 键入空格结束这一段：弹层收起，`@前缀` 与空格都留在草稿里（不被吞）。
func TestFilePickClosesOnSpace(t *testing.T) {
	app, tt := mentionTester(t)
	typeDraft(t, tt, "@comp")
	tt.Type(" ")
	tt.Frame()
	if app.pick.open {
		t.Fatalf("键入空格后弹层仍在：draft=%q", app.draft)
	}
	if app.draft != "@comp " {
		t.Fatalf("草稿 = %q，想要 @comp 后面留一个空格", app.draft)
	}
}

// 会话没有工作目录：空态写「当前无工作区目录」，不留空弹层（G-A3-03）。
func TestFilePickWithoutWorkspace(t *testing.T) {
	src := &fakePanelSource{fakeSource: &fakeSource{
		conn:    Connection{Phase: "connected"},
		names:   []string{"s1"},
		session: SessionSnapshot{ID: "s1", Title: "没有工作区"},
		hasSess: true,
	}, trees: map[string]FileTreeSnapshot{}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("消息"); err != nil {
		t.Fatalf("点输入区：%v", err)
	}
	typeDraft(t, tt, "@")
	if !app.pick.open {
		t.Fatal("弹层没有打开")
	}
	if !tt.HasText(pickNoWorkspaceText) {
		t.Fatalf("缺空态文案 %q：\n%s", pickNoWorkspaceText, strings.Join(tt.Texts(), "\n"))
	}
}

// 技能候选从 Extensions().Skills 读：禁用的不列、同名只留一条（G-A3-10、G-A4-04）。
func TestFilePickListsSkillsFromSource(t *testing.T) {
	src := wiredSource()
	src.extensions.Skills = ExtListSnapshot{Items: []ExtItemSnapshot{
		{ID: "a", Name: "dispatch-work", Description: "真实技能说明", Enabled: true},
		{ID: "b", Name: "dispatch-work", Description: "同名重复", Enabled: true},
		{ID: "c", Name: "disabled-skill", Description: "禁用的技能", Enabled: false},
	}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("消息"); err != nil {
		t.Fatalf("点输入区：%v", err)
	}
	typeDraft(t, tt, "@dispatch")
	if !app.pick.open {
		t.Fatalf("弹层没有打开：draft=%q", app.draft)
	}
	if len(app.pick.skills) != 1 {
		t.Fatalf("技能候选 = %d 条，想要同名去重后的 1 条：%+v", len(app.pick.skills),
			app.pick.skills)
	}
	if !tt.HasText("候选分组：" + pickGroupSkill) {
		t.Fatalf("缺技能分组标题：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if app.pick.skills[0].detail != "真实技能说明" {
		t.Fatalf("技能右尾 = %q，想要描述", app.pick.skills[0].detail)
	}
	for _, item := range app.pick.skills {
		if item.name == "disabled-skill" {
			t.Fatal("禁用的技能进了候选")
		}
	}
}

// 键入 `$` 弹出技能弹层：名称 + 描述，回车插入 `$名字 ` 并收起（G-A4-01、G-A4-04、
// G-A4-05）。
func TestSkillPickOpensAndInserts(t *testing.T) {
	app, tt := mentionTester(t)
	typeDraft(t, tt, "$")
	if !app.pick.open || app.pick.kind != pickSkill {
		t.Fatalf("键入 $ 没有打开技能弹层：open=%v kind=%v", app.pick.open, app.pick.kind)
	}
	if !tt.HasText("候选分组：" + pickGroupSkill) {
		t.Fatalf("缺技能分组标题：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if len(app.pick.skills) == 0 {
		t.Fatalf("没有技能候选：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if app.pick.files != nil {
		t.Fatalf("技能弹层不该列文件：%+v", app.pick.files)
	}
	// 主字技能名、右尾描述：样例技能里有描述的那一条。
	first := app.pick.skills[0]
	if !tt.HasText(first.name) {
		t.Fatalf("屏上没有技能名 %q：\n%s", first.name, strings.Join(tt.Texts(), "\n"))
	}

	typeDraft(t, tt, "deep")
	want := app.pick.items[app.pick.selected]
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.pick.open {
		t.Fatalf("回车后弹层没有收起：draft=%q", app.draft)
	}
	if app.draft != want.insert {
		t.Fatalf("草稿 = %q，想要 %q", app.draft, want.insert)
	}
	if !strings.HasPrefix(app.draft, "$") || !strings.HasSuffix(app.draft, " ") {
		t.Fatalf("插入的格式不是 `$名字 `：%q", app.draft)
	}
}

// `$` 前缀无命中：空态「未找到技能」，弹层不关闭（G-A4-03）。
func TestSkillPickEmptyState(t *testing.T) {
	app, tt := mentionTester(t)
	typeDraft(t, tt, "$zzzzz")
	if !app.pick.open {
		t.Fatal("无命中时弹层不该关闭")
	}
	if !tt.HasText(pickNoSkillText) {
		t.Fatalf("缺空态文案 %q：\n%s", pickNoSkillText, strings.Join(tt.Texts(), "\n"))
	}
	if len(app.pick.items) != 0 {
		t.Fatalf("无命中时仍有候选：%+v", app.pick.items)
	}
	// 空态句不嵌查询词。
	if strings.Contains(pickNoSkillText, "zzzzz") {
		t.Fatal("空态句里嵌了查询词")
	}
}

// `@` 与 `$` 互斥：同一时刻只有一个弹层（G-A3-01、G-A4-01 的「弹层计数为 1」）。
func TestPickPopoversAreExclusive(t *testing.T) {
	app, tt := mentionTester(t)
	typeDraft(t, tt, "@comp")
	if app.pick.kind != pickFile {
		t.Fatalf("先键入 @ 后 kind = %v", app.pick.kind)
	}
	typeDraft(t, tt, " $dee")
	if !app.pick.open || app.pick.kind != pickSkill {
		t.Fatalf("键入 $ 后 kind = %v open=%v", app.pick.kind, app.pick.open)
	}
	// 屏上只有一个候选列表。
	if n := countSearchLabels(tt.Texts(), "候选列表："+pickGroupFile); n != 0 {
		t.Fatalf("文件弹层仍在屏上：%d", n)
	}
	if n := countSearchLabels(tt.Texts(), "候选列表："+pickGroupSkill); n != 1 {
		t.Fatalf("技能弹层层数 = %d，想要 1", n)
	}
}

// pickRetrySource 记下目录读数里出现过一次「跳过缓存」（点了重试）。
type pickRetrySource struct {
	*fakePanelSource
	treeRefreshed bool
}

func (s *pickRetrySource) FileTree(request FileTreeRequest) FileTreeSnapshot {
	if request.Refresh {
		s.treeRefreshed = true
	}
	return s.fakePanelSource.FileTree(request)
}

// 读失败：文件读数失败时给原因与「重试」，重试真的重读一次（跳过缓存）。
func TestFilePickFailureAndRetry(t *testing.T) {
	base := newFakePanelSource("/work/demo")
	base.trees[""] = FileTreeSnapshot{Phase: PanelReadFailed, Error: "读取目录失败"}
	src := &pickRetrySource{fakePanelSource: base}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("消息"); err != nil {
		t.Fatalf("点输入区：%v", err)
	}
	typeDraft(t, tt, "@")
	if !app.pick.open {
		t.Fatal("弹层没有打开")
	}
	if !tt.HasText(pickFileFailedText) || !tt.HasText("读取目录失败") {
		t.Fatalf("失败行缺通用句或原因：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if err := tt.Click(pickRetryFileText); err != nil {
		t.Fatalf("点不到「重试」：%v", err)
	}
	// 点击之后的那一帧读数要跳过缓存（重读一次）。
	tt.Frame()
	tt.Frame()
	if !src.treeRefreshed {
		t.Fatalf("重试没有让目录读数跳过缓存：tree=%+v", src.lastTree)
	}
}

// 候选到扫描上限：写一句上限说明（G-A3 的文件数量上限）。
func TestFilePickCapNote(t *testing.T) {
	src := newFakePanelSource("/work/demo")
	src.trees[""] = FileTreeSnapshot{Phase: PanelReadReady, Entries: []FileEntrySnapshot{
		{Name: "app.go"},
	}}
	src.search = FileSearchSnapshot{
		Phase: PanelReadReady, FilesScanned: pickFileCap + 1,
		Hits: []FileHitSnapshot{{Path: "internal/ui/app.go", Line: 1, Column: 1}},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("消息"); err != nil {
		t.Fatalf("点输入区：%v", err)
	}
	typeDraft(t, tt, "@app")
	if !app.pick.open {
		t.Fatal("弹层没有打开")
	}
	if !tt.HasText(pickCapNote) {
		t.Fatalf("缺上限说明 %q：\n%s", pickCapNote, strings.Join(tt.Texts(), "\n"))
	}
}

// G-A3-07：`@` 弹层本应把提交类候选并入文件组。提交清单没有读数通道（protocol
// v3 无 git 提交方法），所以整组不画；空态里补一行原因，不另立空分组标题。
func TestFilePickCommitCandidatesWriteReason(t *testing.T) {
	app, tt := mentionTester(t)
	typeDraft(t, tt, "@zzz")
	tt.Frame()
	if !tt.HasText(pickNoFileText) {
		t.Fatalf("缺文件空态文案：%v", tt.Texts())
	}
	if !tt.HasText(pickCommitNoChannel) {
		t.Fatalf("空态没有写提交候选缺失的原因：%v", tt.Texts())
	}
	if _, ok := tt.Find("候选分组：提交"); ok {
		t.Fatal("读不到提交清单却画了提交分组")
	}
	// 有文件候选时提交组仍不画，也不在候选下留一行原因（空态才写）。
	typeDraft(t, tt, " @comp")
	if app.pick.query != "comp" {
		t.Fatalf("前缀 = %q，想要 comp", app.pick.query)
	}
	if tt.HasText(pickCommitNoChannel) {
		t.Fatalf("有候选时仍写了提交缺失原因：%v", tt.Texts())
	}
}
