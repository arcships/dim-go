package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 斜杠命令弹层（requirements-global §3.1）：弹出、过滤、键盘、执行、空态、
// 滚动上限。命令清单是客户端静态表，写入走既有 Intents。

// newSlashTester 把焦点放进输入区并键入 `/`，返回已打开的弹层。
func newSlashTester(t *testing.T) (*App, *gui.Tester) {
	t.Helper()
	return newSlashTesterWith(t, NewApp(Options{State: StateDefault}))
}

// newSlashTesterWith 在给定 App 上打开弹层。
func newSlashTesterWith(t *testing.T, app *App) (*App, *gui.Tester) {
	t.Helper()
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("消息"); err != nil {
		t.Fatalf("点输入区：%v", err)
	}
	tt.Type("/")
	tt.Frame()
	if !app.slash.open {
		t.Fatalf("行首键入 / 没有打开弹层：draft=%q", app.draft)
	}
	return app, tt
}

// slashCommandNames 是弹层里的 13 条命令（顺序即静态表的顺序）。
var slashCommandNames = []string{
	"/compact", "/branch",
	"/agent", "/goal", "/pet",
	"/settings", "/models", "/mcp", "/skills",
	"/reload-skills", "/reload-plugins", "/fast", "/workflow",
}

// 行首键入 `/`：弹层出现、四组标题在场、13 条命令都在、宽 327（G-B1-01 /
// G-B1-04 / G-B1-05）。
func TestSlashMenuOpensWithCommandList(t *testing.T) {
	app, tt := newSlashTester(t)
	if got := len(app.slashMenuCommands()); got != len(slashCommandNames) {
		t.Fatalf("命令条数 = %d，想要 %d", got, len(slashCommandNames))
	}
	for _, label := range slashGroupLabels {
		if !tt.HasText("命令分组：" + label) {
			t.Fatalf("少了分组标题 %q：\n%s", label, strings.Join(tt.Texts(), "\n"))
		}
	}
	for _, name := range slashCommandNames {
		if !tt.HasText(name) {
			t.Fatalf("少了命令 %s：\n%s", name, strings.Join(tt.Texts(), "\n"))
		}
	}
	box, ok := tt.Find("斜杠命令")
	if !ok {
		t.Fatal("找不到弹层本体")
	}
	if box.W != slashMenuWidth {
		t.Fatalf("弹层宽 = %.1f，想要 %d", box.W, slashMenuWidth)
	}
	// 首行不默认高亮（参考 C37 的 10 行全为普通底）。
	if app.slash.selected != -1 {
		t.Fatalf("打开时就有选中行 %d", app.slash.selected)
	}
}

// 继续输入按前缀过滤（大小写不敏感），再给模糊匹配与空态（G-B1-01 / G-B1-03）。
func TestSlashMenuFiltersByPrefix(t *testing.T) {
	app, tt := newSlashTester(t)
	tt.Type("model")
	tt.Frame()
	tt.Frame()
	if app.slash.prefix != "model" {
		t.Fatalf("前缀 = %q", app.slash.prefix)
	}
	names := slashNames(app.slashMenuCommands())
	if len(names) != 1 || names[0] != "/models" {
		t.Fatalf("前缀 model 命中 %v，想要只有 /models", names)
	}
	if tt.HasText("命令分组：调试") {
		t.Fatal("组内项全被过滤后组标题还在")
	}

	// 模糊匹配：`rsk` 是 reload-skills 的子序列（reload-plugins 里没有 k）。
	app.draft = "/rsk"
	tt.Frame()
	names = slashNames(app.slashMenuCommands())
	if len(names) != 1 || names[0] != "/reload-skills" {
		t.Fatalf("模糊匹配 rsk 命中 %v，想要只有 /reload-skills", names)
	}

	// 大写前缀按不区分大小写匹配。
	app.draft = "/AG"
	tt.Frame()
	if names := slashNames(app.slashMenuCommands()); len(names) != 1 || names[0] != "/agent" {
		t.Fatalf("前缀 AG 命中 %v，想要只有 /agent", names)
	}

	// 无匹配：空态文案，两个分组标题都不在场。
	app.draft = "/zzzz"
	tt.Frame()
	if !tt.HasText("未找到命令或技能") {
		t.Fatalf("没有空态文案：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	for _, label := range slashGroupLabels {
		if tt.HasText("命令分组：" + label) {
			t.Fatalf("空态时仍画出分组标题 %q", label)
		}
	}

	// 清空前缀回到全量（`/` 后为空不显示空态）。
	app.draft = "/"
	tt.Frame()
	if tt.HasText("未找到命令或技能") {
		t.Fatal("`/` 后为空时仍显示空态")
	}
	if got := len(app.slashMenuCommands()); got != len(slashCommandNames) {
		t.Fatalf("清空前缀后只剩 %d 条", got)
	}
}

// slashDraft 直接改写草稿并把弹层推到稳定态：第一帧按前缀过滤，第二帧起
// 输入框才带着 OnTextKey（框架的按键接管从下一帧生效），之后的键盘操作才会
// 被弹层看到。
func slashDraft(t *testing.T, app *App, tt *gui.Tester, draft string) {
	t.Helper()
	app.draft = draft
	tt.Frame()
	tt.Frame()
}

// slashNames 取命令名，便于断言过滤结果。
func slashNames(commands []slashCommand) []string {
	names := make([]string, 0, len(commands))
	for _, cmd := range commands {
		names = append(names, "/"+cmd.name)
	}
	return names
}

// ↑↓ 选择（跳过置灰项）与 ↵ 执行：执行后输入框清空、弹层收起（G-B1-07 /
// G-B1-08）。
func TestSlashMenuArrowsAndEnter(t *testing.T) {
	// 样例模式没有 Intents：前两条置灰，所以第一次下键落在第三条（/branch）。
	app, tt := newSlashTester(t)
	selectable := app.slashSelectable()
	if len(selectable) == 0 {
		t.Fatal("没有可执行的命令")
	}
	tt.Key(0, gui.KeyDown)
	if app.slash.selected != selectable[0] {
		t.Fatalf("第一次下键选中 %d，想要 %d", app.slash.selected, selectable[0])
	}
	tt.Key(0, gui.KeyDown)
	if app.slash.selected != selectable[1] {
		t.Fatalf("第二次下键选中 %d，想要 %d", app.slash.selected, selectable[1])
	}
	tt.Key(0, gui.KeyUp)
	if app.slash.selected != selectable[0] {
		t.Fatalf("上键选中 %d，想要 %d", app.slash.selected, selectable[0])
	}

	// 过滤到 /agent，回车执行：模式切到 Agent，草稿清空，弹层收起；回执写明
	// 「本版模式位只改界面标记，尚未影响发送」，不许只回一句「已执行」。
	slashDraft(t, app, tt, "/agent")
	tt.Key(0, gui.KeyDown)
	tt.Key(0, gui.KeyEnter)
	if app.mode != "Agent" {
		t.Fatalf("执行 /agent 后模式 = %q", app.mode)
	}
	if app.draft != "" {
		t.Fatalf("执行后草稿 = %q，想要清空", app.draft)
	}
	if app.slash.open {
		t.Fatal("执行后弹层没有收起")
	}
	if tt.HasText("已执行：/agent") {
		t.Fatalf("/agent 仍回一句「已执行」：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if !tt.HasText(app.slash.notice) || !strings.Contains(app.slash.notice, "尚未影响发送") {
		t.Fatalf("/agent 回执没有写清生效范围：%q", app.slash.notice)
	}
}

// Esc 收起弹层但保留 `/` 与前缀；再键入字符重新弹出（G-B1-01）。
func TestSlashMenuEscapeKeepsDraft(t *testing.T) {
	app, tt := newSlashTester(t)
	tt.Type("ag")
	tt.Frame()
	tt.Frame()
	if !app.slash.open || app.slash.prefix != "ag" {
		t.Fatalf("前缀 = %q，open=%v", app.slash.prefix, app.slash.open)
	}
	tt.Key(0, gui.KeyEscape)
	if app.slash.open {
		t.Fatal("Esc 没有收起弹层")
	}
	if app.draft != "/ag" {
		t.Fatalf("Esc 后草稿 = %q，想要 /ag（不吞字）", app.draft)
	}
	tt.Frame()
	if app.slash.open {
		t.Fatal("同一前缀又自动弹出来了")
	}
	tt.Type("e")
	tt.Frame()
	tt.Frame()
	if !app.slash.open {
		t.Fatal("继续键入后弹层没有重新出现")
	}
	if app.draft != "/age" {
		t.Fatalf("继续键入后草稿 = %q", app.draft)
	}
}

// 没有实现路径的命令置灰、不可选中，原因写在行内（G-B1-05 / G-B1-12）。
func TestSlashMenuDisabledCommandsSayWhy(t *testing.T) {
	app, tt := newSlashTester(t)
	for _, name := range []string{"pet", "reload-plugins", "fast", "workflow"} {
		slashDraft(t, app, tt, "/"+name)
		commands := app.slashMenuCommands()
		if len(commands) != 1 || commands[0].name != name {
			t.Fatalf("过滤 %s 命中 %v", name, slashNames(commands))
		}
		if commands[0].runnable() {
			t.Fatalf("/%s 不该有可执行路径", name)
		}
		if !tt.HasText("不可用：") {
			t.Fatalf("/%s 行内没有写原因：\n%s", name, strings.Join(tt.Texts(), "\n"))
		}
		// 置灰项不被 ↑↓ 选中。
		tt.Key(0, gui.KeyDown)
		if app.slash.selected != -1 {
			t.Fatalf("/%s 是可选中项（%d）", name, app.slash.selected)
		}
	}
}

// 设置类命令直达对应分节（G-B1-11 / G-B1-12 / G-B1-13 / G-B1-14）。
func TestSlashMenuOpensSettingsSections(t *testing.T) {
	cases := []struct {
		command string
		section settingsSectionID
	}{
		{"settings", secAccount},
		{"models", secProviders},
		{"mcp", secMCP},
		{"skills", secPluginSettings},
	}
	for _, c := range cases {
		app, tt := newSlashTester(t)
		slashDraft(t, app, tt, "/"+c.command)
		tt.Key(0, gui.KeyDown)
		tt.Key(0, gui.KeyEnter)
		if app.screen != ScreenSettings || app.settingsSection != c.section {
			t.Fatalf("/%s 落在 %v/%v，想要设置 %v", c.command, app.screen, app.settingsSection, c.section)
		}
	}
}

// 鼠标点一行与回车同一条执行路径（G-B1-05）。
func TestSlashMenuClickRunsCommand(t *testing.T) {
	app, tt := newSlashTester(t)
	slashDraft(t, app, tt, "/branch")
	if err := tt.Click("/branch，侧边对话"); err != nil {
		t.Fatalf("点命令行：%v", err)
	}
	if !app.rightOpen || app.rightTab != tabSideChat {
		t.Fatalf("点 /branch 后右栏 = %v/%v，想要侧聊", app.rightOpen, app.rightTab)
	}
	if app.draft != "" || app.slash.open {
		t.Fatalf("点命令行后草稿 = %q，open=%v", app.draft, app.slash.open)
	}
}

// 候选多于可视高度时在弹层内滚动：高度不超过 8 行（G-B1-06）。
func TestSlashMenuScrollsWithinEightRows(t *testing.T) {
	_, tt := newSlashTester(t)
	box, ok := tt.Find("斜杠命令")
	if !ok {
		t.Fatal("找不到弹层本体")
	}
	limit := float32(slashMenuMaxHeight + 2*slashMenuPad + 2*strokeHairline)
	if box.H > limit {
		t.Fatalf("弹层高 %.1f 超过 8 行上限 %.1f（13 条命令应滚动）", box.H, limit)
	}
	if box.H < slashMenuMaxHeight {
		t.Fatalf("弹层高 %.1f 没有用满 8 行（%d）", box.H, slashMenuMaxHeight)
	}
}

// 焦点离开输入区（点弹层外）收起弹层，草稿不动（G-B1-01）。
func TestSlashMenuClosesWhenFocusLeavesComposer(t *testing.T) {
	app, tt := newSlashTester(t)
	if err := tt.Click("附件"); err != nil {
		t.Fatalf("点附件按钮：%v", err)
	}
	tt.Frame()
	if app.slash.open {
		t.Fatal("点了弹层外没有收起")
	}
	if app.draft != "/" {
		t.Fatalf("收起后草稿 = %q，想要 /", app.draft)
	}
}

// slashIntentStub 只实现两条斜杠命令的写入口：其余方法走嵌入的 nil 接口，
// 被调用会 panic（说明测试点到了一个没接的动作）。
type slashIntentStub struct {
	Intents
	compact chan struct{}
	skills  chan struct{}
}

// OpenSession 是窗口打开会话时的调用：这些用例不换会话，直接成功。
func (s *slashIntentStub) OpenSession(context.Context, string) error { return nil }

func (s *slashIntentStub) CompactContext(context.Context) error {
	s.compact <- struct{}{}
	return nil
}

func (s *slashIntentStub) RefreshSkills(context.Context) error {
	s.skills <- struct{}{}
	return nil
}

// /compact 与 /reload-skills 走既有 Intents（不另开通道）（G-B1-17 / G-B1-15）。
func TestSlashMenuWritesGoThroughIntents(t *testing.T) {
	stub := &slashIntentStub{
		compact: make(chan struct{}, 1),
		skills:  make(chan struct{}, 1),
	}
	app := NewApp(Options{State: StateDefault, Source: wiredSource(), Intents: stub})
	app, tt := newSlashTesterWith(t, app)

	for _, c := range []struct {
		command string
		ready   chan struct{}
	}{{"/compact", stub.compact}, {"/reload-skills", stub.skills}} {
		slashDraft(t, app, tt, c.command)
		tt.Key(0, gui.KeyDown)
		tt.Key(0, gui.KeyEnter)
		select {
		case <-c.ready:
		case <-time.After(2 * time.Second):
			t.Fatalf("%s 没有走 Intents", c.command)
		}
	}
}

// 连着 Source 但没有打开会话时，写入类命令置灰并写原因（G-B1-17 的异常分支）。
func TestSlashMenuWithoutSessionSaysWhy(t *testing.T) {
	src := wiredSource()
	src.hasSess = false
	stub := &slashIntentStub{compact: make(chan struct{}, 1), skills: make(chan struct{}, 1)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
	app, tt := newSlashTesterWith(t, app)
	slashDraft(t, app, tt, "/compact")
	commands := app.slashMenuCommands()
	if len(commands) != 1 || commands[0].runnable() {
		t.Fatalf("/compact 在没有会话时仍可执行：%v", slashNames(commands))
	}
	if !tt.HasText("还没有打开会话") {
		t.Fatalf("没有写原因：\n%s", strings.Join(tt.Texts(), "\n"))
	}
}

// /goal 与输入卡的模式菜单同一落点：切到 Goal 模式，没有目标时打开目标行的
// 编辑框（真正的写入仍走 SetGoal）；回执写明模式位本版尚未影响发送。
func TestSlashGoalSwitchesModeAndOpensGoalEditor(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	app, tt := newSlashTesterWith(t, app)
	slashDraft(t, app, tt, "/goal")
	commands := app.slashMenuCommands()
	if len(commands) != 1 || commands[0].name != "goal" {
		t.Fatalf("过滤 /goal 命中 %v", slashNames(commands))
	}
	if !commands[0].runnable() {
		t.Fatal("/goal 在已打开会话时不可执行")
	}
	tt.Key(0, gui.KeyDown)
	tt.Key(0, gui.KeyEnter)

	if app.mode != "Goal" {
		t.Fatalf("模式 = %q，想要 Goal", app.mode)
	}
	if !app.goalEditing {
		t.Fatal("没有目标时 /goal 没有打开目标编辑框（与模式菜单不同落点）")
	}
	if _, ok := tt.Access("保存目标"); !ok {
		t.Fatalf("目标编辑框没有上屏（找不到「保存目标」）：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if !strings.Contains(app.slash.notice, "目标编辑框") ||
		!strings.Contains(app.slash.notice, "尚未影响发送") {
		t.Fatalf("/goal 回执没有写清实际效果：%q", app.slash.notice)
	}
	if strings.HasPrefix(app.slash.notice, "已执行") {
		t.Fatalf("/goal 仍回一句「已执行」：%q", app.slash.notice)
	}
}

// /branch 在连接模式下置灰写原因：侧聊还没有事实通道，不把用户送进一个只会
// 本地回声的面板。
func TestSlashBranchDisabledWithoutSideChatChannel(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	app, tt := newSlashTesterWith(t, app)
	slashDraft(t, app, tt, "/branch")
	commands := app.slashMenuCommands()
	if len(commands) != 1 || commands[0].name != "branch" {
		t.Fatalf("过滤 /branch 命中 %v", slashNames(commands))
	}
	if commands[0].runnable() {
		t.Fatal("/branch 在连接模式下仍可执行")
	}
	if commands[0].reason != slashBranchNoChannel {
		t.Fatalf("/branch 的原因 = %q，想要 %q", commands[0].reason, slashBranchNoChannel)
	}
	if !tt.HasText("不可用：" + slashBranchNoChannel) {
		t.Fatalf("/branch 行内没有写原因：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	// 回车不会执行置灰项：选中行留在 -1，右栏不打开，回执仍是原因句。
	tt.Key(0, gui.KeyDown)
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.slash.selected != -1 {
		t.Fatalf("/branch 可选中（选中 %d）", app.slash.selected)
	}
	if app.rightOpen {
		t.Fatal("/branch 不可用却打开了右栏")
	}
	// 绕过键盘直接执行也不写任何东西：动作只回原因。
	for i, cmd := range app.slashMenuCommands() {
		if cmd.name == "branch" {
			app.slash.selected = i
		}
	}
	app.slash.notice = ""
	app.slashExecuteSelected()
	if app.rightOpen {
		t.Fatal("/branch 不可用却打开了右栏")
	}
	if app.slash.notice != "不可用："+slashBranchNoChannel {
		t.Fatalf("/branch 的回执 = %q", app.slash.notice)
	}
}
