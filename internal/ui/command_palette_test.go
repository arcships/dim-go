package ui

import (
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// newPaletteTester 建一个已打开命令面板的窗口。
func newPaletteTester(t *testing.T, s State) (*App, *gui.Tester) {
	t.Helper()
	app := NewApp(Options{State: s})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Key(gui.Cmd, gui.KeyK)
	if !app.palette.open {
		t.Fatal("Cmd+K 没有打开命令面板")
	}
	return app, tt
}

// settlePaletteQuery 让去抖到期：120ms 后换结果（S-CP-04）。
func settlePaletteQuery(tt *gui.Tester) {
	tt.Advance(commandPaletteDebounce + 20*time.Millisecond)
}

// Cmd+K 开关、Esc 收、焦点进搜索框（S-CP-02）。
func TestCommandPaletteTogglesAndFocusesSearch(t *testing.T) {
	app, tt := newPaletteTester(t, StateDefault)
	if !tt.Focused("搜索命令") {
		t.Fatal("打开后焦点不在搜索框")
	}
	if !tt.HasText("动作") || !tt.HasText("会话") || !tt.HasText("文件") || !tt.HasText("设置项") {
		t.Fatalf("四组没有同时出现：%v", tt.Texts())
	}
	tt.Key(gui.Cmd, gui.KeyK)
	if app.palette.open {
		t.Fatal("再按 Cmd+K 没有关闭")
	}
	if _, ok := tt.Find("搜索命令"); ok {
		t.Fatal("关闭后搜索框还在")
	}

	tt.Key(gui.Cmd, gui.KeyK)
	if !app.palette.open {
		t.Fatal("Cmd+K 没有重新打开")
	}
	tt.Key(0, gui.KeyEscape)
	if app.palette.open {
		t.Fatal("Esc 没有关闭")
	}
}

// 输入去抖 120ms，过滤在到期后才换结果（S-CP-04）。
func TestCommandPaletteDebouncesInput(t *testing.T) {
	app, tt := newPaletteTester(t, StateDefault)
	tt.Type("账户")
	if app.palette.query != "账户" {
		t.Fatalf("输入没有进查询词：%q", app.palette.query)
	}
	if app.palette.filter != "" {
		t.Fatalf("输入后立即换了结果：%q", app.palette.filter)
	}
	tt.Advance(119 * time.Millisecond)
	if app.palette.filter != "" {
		t.Fatalf("119ms 就去抖了：%q", app.palette.filter)
	}
	settlePaletteQuery(tt)
	if app.palette.filter != "账户" {
		t.Fatalf("去抖到期后没有过滤：%q", app.palette.filter)
	}
	if !tt.HasText("设置 · 账户") {
		t.Fatalf("没有「设置 · 账户」：%v", tt.Texts())
	}
	if tt.HasText("发送当前草稿") {
		t.Fatal("过滤后仍显示不匹配的动作")
	}
}

// 上下键改选中行，回车执行当前行；执行后 300ms 内关闭（S-CP-06、S-CP-09）。
func TestCommandPaletteArrowsAndEnter(t *testing.T) {
	app, tt := newPaletteTester(t, StateDefault)
	if app.palette.activeGroup != commandGroupActions {
		t.Fatalf("打开时不在动作组：%d", app.palette.activeGroup)
	}
	tt.Key(0, gui.KeyDown)
	if app.palette.selected != 1 {
		t.Fatalf("下键后选中 %d", app.palette.selected)
	}
	tt.Key(0, gui.KeyUp)
	if app.palette.selected != 0 {
		t.Fatalf("上键后选中 %d", app.palette.selected)
	}

	// 过滤到唯一的设置项，回车打开设置。
	tt.Type("账户")
	settlePaletteQuery(tt)
	tt.Key(0, gui.KeyEnter)
	if app.screen != ScreenSettings || app.settingsSection != secAccount {
		t.Fatalf("回车没有打开账户设置：%v %v", app.screen, app.settingsSection)
	}
	if app.palette.closeAt.IsZero() {
		t.Fatal("执行后没有排上关闭")
	}
	tt.Advance(200 * time.Millisecond)
	if !app.palette.open {
		t.Fatal("执行后不到 300ms 就关了")
	}
	tt.Advance(150 * time.Millisecond)
	if app.palette.open {
		t.Fatal("执行后 300ms 内没有关闭")
	}
}

// Tab / Shift+Tab 换组（S-CP-06），选中行回到组首。
func TestCommandPaletteTabsBetweenGroups(t *testing.T) {
	app, tt := newPaletteTester(t, StateDefault)
	tt.Key(0, gui.KeyDown)
	tt.Key(0, gui.KeyTab)
	if app.palette.activeGroup != commandGroupSessions {
		t.Fatalf("Tab 后是第 %d 组", app.palette.activeGroup)
	}
	if app.palette.selected != 0 {
		t.Fatalf("换组后选中行没有回到组首：%d", app.palette.selected)
	}
	tt.Key(gui.Shift, gui.KeyTab)
	if app.palette.activeGroup != commandGroupActions {
		t.Fatalf("Shift+Tab 后是第 %d 组", app.palette.activeGroup)
	}
}

// 无匹配项显示空态（S-CP-08）。
func TestCommandPaletteEmptyState(t *testing.T) {
	app, tt := newPaletteTester(t, StateDefault)
	tt.Type("zzzz")
	settlePaletteQuery(tt)
	total := 0
	for i := 0; i < commandGroupCount; i++ {
		total += len(app.palette.groups[i].items)
	}
	if total != 0 {
		t.Fatalf("无匹配时仍有 %d 条", total)
	}
	if !tt.HasText("没有匹配项") {
		t.Fatalf("没有空态文案：%v", tt.Texts())
	}
	// 空态里按上下与回车不应越界。
	tt.Key(0, gui.KeyDown)
	tt.Key(0, gui.KeyEnter)
	if !app.palette.open {
		t.Fatal("空态回车不应关闭面板")
	}
}

// 失焦 300ms 关闭（S-CP-02）。
func TestCommandPaletteClosesAfterBlur(t *testing.T) {
	app, tt := newPaletteTester(t, StateDefault)
	tt.SetWindowActive(false)
	tt.Advance(200 * time.Millisecond)
	if !app.palette.open {
		t.Fatal("失焦不到 300ms 就关了")
	}
	tt.Advance(150 * time.Millisecond)
	if app.palette.open {
		t.Fatal("失焦后没有关闭")
	}
}

// 拼音首字母可命中（S-CP-07）。
func TestCommandPaletteMatchesPinyinInitials(t *testing.T) {
	app, tt := newPaletteTester(t, StateDefault)
	tt.Type("zh")
	settlePaletteQuery(tt)
	found := false
	for _, item := range app.palette.groups[commandGroupSettings].items {
		if item.title == "设置 · 账户" {
			found = true
		}
	}
	if !found {
		t.Fatalf("拼音首字母 zh 没有命中账户：%v", paletteTitles(app.palette.groups[commandGroupSettings].items))
	}
}

// 分组 ≤8、整面板 ≤20（S-CP-03）。
func TestCommandPaletteGroupAndTotalLimits(t *testing.T) {
	app, _ := newPaletteTester(t, StateDefault)
	total := 0
	for i := 0; i < commandGroupCount; i++ {
		n := len(app.palette.groups[i].items)
		if n > commandPaletteGroupLimit {
			t.Fatalf("第 %d 组有 %d 条，超过 %d", i, n, commandPaletteGroupLimit)
		}
		total += n
	}
	if total > commandPaletteTotalLimit {
		t.Fatalf("整面板 %d 条，超过 %d", total, commandPaletteTotalLimit)
	}
}

// 最近使用 5 条排到组首，搜索历史最多 10 条（S-CP-11、S-CP-12）。
func TestCommandPaletteRecentAndHistory(t *testing.T) {
	app, _ := newPaletteTester(t, StateDefault)
	for i := 0; i < commandPaletteHistoryLimit+3; i++ {
		app.palette.query = "q" + string(rune('a'+i))
		app.recordPaletteExecution(commandItem{id: "action:stop", title: "停止这次运行"})
	}
	if len(app.palette.history) != commandPaletteHistoryLimit {
		t.Fatalf("历史 %d 条，想要 %d", len(app.palette.history), commandPaletteHistoryLimit)
	}
	if app.palette.history[0] != "qm" {
		t.Fatalf("历史最新一条是 %q", app.palette.history[0])
	}
	for i := 0; i < commandPaletteRecentLimit+2; i++ {
		app.recordPaletteExecution(commandItem{id: "file:" + string(rune('a'+i)), title: "f"})
	}
	if len(app.palette.recent) != commandPaletteRecentLimit {
		t.Fatalf("最近使用 %d 条，想要 %d", len(app.palette.recent), commandPaletteRecentLimit)
	}
	// 用文件组里真实存在的一条做最近使用，验证它排到组首。
	before := app.commandPaletteGroups()[commandGroupFiles].items
	if len(before) < 2 {
		t.Fatalf("文件组只有 %d 条", len(before))
	}
	picked := before[len(before)-1].id
	app.recordPaletteExecution(commandItem{id: picked, title: "文件"})
	groups := app.commandPaletteGroups()
	if got := groups[commandGroupFiles].items[0].id; got != picked {
		t.Fatalf("最近使用的文件没有排到组首：%q，想要 %q", got, picked)
	}
}

// 尺寸与位置（S-CP-01）：560×480，水平居中，顶部 12%。
func TestCommandPaletteGeometry(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Variant: VariantPalette})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	info, ok := tt.Access("命令面板")
	if !ok {
		t.Fatal("找不到命令面板")
	}
	b := info.Bounds
	if b.W != commandPaletteWidth || b.H != commandPaletteHeight {
		t.Fatalf("面板 %.0f×%.0f，想要 %d×%d", b.W, b.H, commandPaletteWidth, commandPaletteHeight)
	}
	wantX := (float32(WindowWidth) - commandPaletteWidth) / 2
	wantY := float32(WindowHeight) * commandPaletteTop
	if b.X != wantX || b.Y != wantY {
		t.Fatalf("面板在 (%.0f,%.0f)，想要 (%.0f,%.0f)", b.X, b.Y, wantX, wantY)
	}
}

// 空态截图变体：打开面板并停在「没有匹配项」。
func TestCommandPaletteEmptyVariant(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Variant: VariantPaletteEmpty})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if !app.palette.open {
		t.Fatal("走查变体没有打开面板")
	}
	if !tt.HasText("没有匹配项") {
		t.Fatalf("空态变体没有空态文案：%v", tt.Texts())
	}
}

// paletteTitles 是调试用的标题列表。
func paletteTitles(items []commandItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.title)
	}
	return out
}

// TestCommandPaletteRowShortcut：有界面内快捷键的动作在行尾画实际键位（改过键
// 就画改后的），没有绑定的行不画（G-B2-06）。
func TestCommandPaletteRowShortcut(t *testing.T) {
	app, tt := newPaletteTester(t, StateDefault)
	want := app.keybindingLabel("new-thread")
	if want == "" {
		t.Fatal("new-thread 没有可读键位")
	}
	if _, ok := tt.Find("新建会话"); !ok {
		t.Fatalf("命令面板没有「新建会话」行：%v", tt.Texts())
	}
	if !tt.HasText(want) {
		t.Fatalf("行尾没有快捷键 %q：%v", want, tt.Texts())
	}
	// 读屏名带上同一个键位；样例模式这一行不可用，快捷键仍要画（R-GM-210）。
	if info, ok := tt.Access("新建会话：在工作区里开一个会话，快捷键 " + want); !ok || !info.Disabled {
		t.Fatalf("行读屏名没有带快捷键：%+v（找到 %v）", info, ok)
	}
	app.keybindings = map[string]string{"new-thread": "Cmd+Alt+N"}
	tt.Frame()
	changed := app.keybindingLabel("new-thread")
	if changed == want {
		t.Fatal("改键后键位没有变")
	}
	if !tt.HasText(changed) {
		t.Fatalf("改键后行尾没有换成 %q：%v", changed, tt.Texts())
	}
	// 没有绑定的行不画快捷键：行读屏名里没有「快捷键」这一段。
	if info, ok := tt.Access("停止这次运行：中止当前回复"); !ok {
		t.Fatalf("没有绑定的行读屏名不对：%+v（找到 %v）", info, ok)
	}
}
