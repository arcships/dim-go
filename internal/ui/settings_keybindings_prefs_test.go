package ui

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/shortcut"
)

// prefsIntents 只接 SetPreferences：记下每次落盘的偏好，可按需让写入失败。
type prefsIntents struct {
	Intents
	mu    sync.Mutex
	saved []PreferencesSnapshot
	fail  error
	ch    chan PreferencesSnapshot
}

func newPrefsIntents() *prefsIntents {
	return &prefsIntents{ch: make(chan PreferencesSnapshot, 4)}
}

func (p *prefsIntents) SetPreferences(_ context.Context, prefs PreferencesSnapshot) error {
	p.mu.Lock()
	p.saved = append(p.saved, prefs)
	fail := p.fail
	p.mu.Unlock()
	if p.ch != nil {
		select {
		case p.ch <- prefs:
		default:
		}
	}
	return fail
}

// expectBinding 等一次落盘，并核对某个动作写进去的键位（空串表示这一项被删掉）。
func (p *prefsIntents) expectBinding(t *testing.T, id, want string) {
	t.Helper()
	select {
	case prefs := <-p.ch:
		if got := prefs.Keybindings[id]; got != want {
			t.Fatalf("落盘的 %s = %q，想要 %q（整份：%v）", id, got, want, prefs.Keybindings)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("没有等到 SetPreferences（%s）", id)
	}
}

// openKeybindings 打开设置 › 快捷键并渲染一帧。
func openKeybindings(t *testing.T, src *fakeSource, intents Intents) (*App, *gui.Tester) {
	t.Helper()
	resetSettingsScreenState()
	keybindingsScreen = keybindingsState{}
	globalShortcutEdit = globalShortcutEditState{}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	return app, openSettingsSection(t, app, secKeybindings)
}

// TestKeybindingEditPersistsToPreferences：改键落进本地偏好的 keybindings，
// 键位按规范写法保存，屏上换成新键位，并且出现「重置」。
func TestKeybindingEditPersistsToPreferences(t *testing.T) {
	intents := newPrefsIntents()
	app, tt := openKeybindings(t, wiredSource(), intents)

	if err := tt.Click("快捷键：切换终端"); err != nil {
		t.Fatalf("点不到键位按钮：%v", err)
	}
	tt.Frame()
	if keybindingsScreen.editing != "toggle-terminal" {
		t.Fatalf("点击后没有进入改键态：%q", keybindingsScreen.editing)
	}
	// 输入框里用字形写法：⌘⌥J → Cmd+Alt+J。
	keybindingsScreen.draft = "⌘⌥J"
	tt.Frame()
	if err := tt.Click("应用切换终端快捷键"); err != nil {
		t.Fatalf("点不到「应用」：%v", err)
	}
	intents.expectBinding(t, "toggle-terminal", "Cmd+Alt+J")

	tt.Frame()
	if keybindingsScreen.editing != "" {
		t.Fatal("保存后仍停在改键态")
	}
	if got := app.keybindingAccel("toggle-terminal"); got != "Cmd+Alt+J" {
		t.Fatalf("本地镜像没有跟上：%q", got)
	}
	if !tt.HasText(shortcut.Label("Cmd+Alt+J")) {
		t.Fatalf("屏上没有新键位 %q：%v", shortcut.Label("Cmd+Alt+J"), tt.Texts())
	}
	if _, ok := tt.Access("重置 切换终端"); !ok {
		t.Fatal("改过的行没有出现「重置」")
	}
}

// TestKeybindingInvalidKeepsOldValue：认不出的键位不落盘、原值保留，行下写原因。
func TestKeybindingInvalidKeepsOldValue(t *testing.T) {
	intents := newPrefsIntents()
	app, tt := openKeybindings(t, wiredSource(), intents)

	if err := tt.Click("快捷键：切换侧边栏"); err != nil {
		t.Fatalf("点不到键位按钮：%v", err)
	}
	keybindingsScreen.draft = "K" // 没有修饰键
	tt.Frame()
	if err := tt.Click("应用切换侧边栏快捷键"); err != nil {
		t.Fatalf("点不到「应用」：%v", err)
	}
	tt.Frame()

	if got := app.keybindingAccel("toggle-sidebar"); got != shortcut.DefaultBinding("toggle-sidebar") {
		t.Fatalf("无效键位改动了当前值：%q", got)
	}
	if !tt.HasText(keybindingErrInvalid) {
		t.Fatalf("行下没有写原因：%v", tt.Texts())
	}
	select {
	case prefs := <-intents.ch:
		t.Fatalf("无效键位不该落盘：%v", prefs.Keybindings)
	case <-time.After(50 * time.Millisecond):
	}
}

// TestKeybindingConflictKeepsNewValue：与另一行撞键时保留新值，并在行下写冲突。
func TestKeybindingConflictKeepsNewValue(t *testing.T) {
	intents := newPrefsIntents()
	app, tt := openKeybindings(t, wiredSource(), intents)

	if err := tt.Click("快捷键：切换终端"); err != nil {
		t.Fatalf("点不到键位按钮：%v", err)
	}
	keybindingsScreen.draft = "⌘B" // 与「切换侧边栏」撞键
	tt.Frame()
	if err := tt.Click("应用切换终端快捷键"); err != nil {
		t.Fatalf("点不到「应用」：%v", err)
	}
	intents.expectBinding(t, "toggle-terminal", "Cmd+B")

	tt.Frame()
	if got := app.keybindingAccel("toggle-terminal"); got != "Cmd+B" {
		t.Fatalf("撞键后没有保留新值：%q", got)
	}
	want := strings.Replace(keybindingErrConflict, "%s", "切换侧边栏", 1)
	if !tt.HasText(want) {
		t.Fatalf("行下没有写冲突提示：%v", tt.Texts())
	}
}

// TestKeybindingResetDropsOverride：改过的行点「重置」回到出厂值，偏好里删掉这一项。
func TestKeybindingResetDropsOverride(t *testing.T) {
	src := wiredSource()
	src.prefs.Keybindings = map[string]string{"toggle-terminal": "Cmd+Alt+J"}
	intents := newPrefsIntents()
	app, tt := openKeybindings(t, src, intents)

	if got := app.keybindingAccel("toggle-terminal"); got != "Cmd+Alt+J" {
		t.Fatalf("启动没有读回偏好里的键位：%q", got)
	}
	if err := tt.Click("重置 切换终端"); err != nil {
		t.Fatalf("点不到「重置」：%v", err)
	}
	intents.expectBinding(t, "toggle-terminal", "")

	tt.Frame()
	if got := app.keybindingAccel("toggle-terminal"); got != shortcut.DefaultBinding("toggle-terminal") {
		t.Fatalf("重置后没有回到出厂值：%q", got)
	}
	if _, ok := tt.Access("重置 切换终端"); ok {
		t.Fatal("重置后「重置」还留在行上")
	}
}

// TestKeybindingSaveFailureRollsBack：落盘失败时回到保存前的键位，并在行下写原因。
func TestKeybindingSaveFailureRollsBack(t *testing.T) {
	intents := newPrefsIntents()
	intents.fail = errors.New("偏好文件不可写")
	app, tt := openKeybindings(t, wiredSource(), intents)

	if err := tt.Click("快捷键：切换终端"); err != nil {
		t.Fatalf("点不到键位按钮：%v", err)
	}
	keybindingsScreen.draft = "⌘⌥J"
	tt.Frame()
	if err := tt.Click("应用切换终端快捷键"); err != nil {
		t.Fatalf("点不到「应用」：%v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if keybindingsScreen.errors["toggle-terminal"] == keybindingErrSave {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := keybindingsScreen.errors["toggle-terminal"]; got != keybindingErrSave {
		t.Fatalf("失败原因没有回到行内：%q", got)
	}
	if got := app.keybindingAccel("toggle-terminal"); got != shortcut.DefaultBinding("toggle-terminal") {
		t.Fatalf("落盘失败没有回滚：%q", got)
	}
	if !tt.HasText("切换终端：" + keybindingErrSave) {
		t.Fatalf("失败原因没有上屏：%v", tt.Texts())
	}
}

// TestKeybindingSubmitChangesHasNoAction：「提交更改」本版没有落点：按钮置灰、
// 写原因，点了不进入改键。
func TestKeybindingSubmitChangesHasNoAction(t *testing.T) {
	_, tt := openKeybindings(t, wiredSource(), newPrefsIntents())
	if !tt.HasText(shortcut.Label("CmdOrCtrl+Shift+Enter")) {
		t.Fatalf("没有落点的行也要显示键位：%v", tt.Texts())
	}
	info, ok := tt.Access("快捷键：提交更改")
	if !ok {
		t.Fatal("没有落点的行也要保留入口")
	}
	if !info.Disabled {
		t.Fatal("没有落点的行应当置灰")
	}
}

// TestKeybindingDrivesMainWindowKeys：改过的键位真的接管主界面，出厂键位不再生效。
func TestKeybindingDrivesMainWindowKeys(t *testing.T) {
	src := wiredSource()
	src.prefs.Keybindings = map[string]string{"toggle-terminal": "Cmd+Alt+J"}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	// 出厂键位 ⌘J 已经不接管。
	tt.Key(gui.Cmd, gui.KeyJ)
	tt.Frame()
	if app.rightOpen && app.rightTab == tabTerminal {
		t.Fatal("旧键位 ⌘J 仍在开终端页")
	}
	// 改过的键位 ⌘⌥J 打开终端页。
	tt.Key(gui.Cmd|gui.Alt, gui.KeyJ)
	tt.Frame()
	if !app.rightOpen || app.rightTab != tabTerminal {
		t.Fatalf("改过的键位没有打开终端页：open=%v tab=%d", app.rightOpen, app.rightTab)
	}
	// 同一键位再按一次收起。
	tt.Key(gui.Cmd|gui.Alt, gui.KeyJ)
	tt.Frame()
	if app.rightOpen {
		t.Fatal("同一键位第二次没有收起终端页")
	}
}

// TestKeybindingToggleSidebarAndSearch：出厂键位下，⌘B 开合左栏、⌘G 打开搜索对话、
// ⌘L 排焦点；这三条都走同一条「界面内快捷键」通路。
func TestKeybindingToggleSidebarAndSearch(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	before := app.sidebarCollapsed
	tt.Key(gui.Cmd, gui.KeyB)
	tt.Frame()
	if app.sidebarCollapsed == before {
		t.Fatal("⌘B 没有切换左栏")
	}
	tt.Key(gui.Cmd, gui.KeyG)
	tt.Frame()
	if !app.chatSearch.open {
		t.Fatal("⌘G 没有打开搜索对话")
	}
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	tt.Key(gui.Cmd, gui.KeyL)
	tt.Frame()
	if !tt.Focused("消息") {
		t.Fatal("⌘L 没有把焦点落到输入区")
	}
	if app.focusComposer.Load() {
		t.Fatal("焦点请求没有被消费：下一帧还会再抢一次")
	}
}

// TestParseAcceleratorRejectsUnknown：表外的键位按不到，不猜一个别的键。
func TestParseAcceleratorRejectsUnknown(t *testing.T) {
	cases := []string{"", "B", "Cmd+Bogus", "CmdOrCtrl+", "Cmd+NumDec"}
	for _, text := range cases {
		if _, _, ok := parseAccelerator(text); ok {
			t.Fatalf("%q 不应当能换算成框架按键", text)
		}
	}
	mods, key, ok := parseAccelerator("CmdOrCtrl+Shift+B")
	if !ok || mods != gui.Cmd|gui.Shift || key != gui.KeyB {
		t.Fatalf("CmdOrCtrl+Shift+B 换算成了 %v/%v（ok=%v）", mods, key, ok)
	}
}
