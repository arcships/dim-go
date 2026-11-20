package ui

import (
	"context"
	"fmt"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/shortcut"
)

// shortcutIntents 记录设置 › 快捷键 › 全局发出的写入；其余方法留在嵌入的
// Intents 上（没被调到的方法不会生效）。
type shortcutIntents struct {
	Intents
	calls chan [2]string
	err   error
}

func newShortcutIntents() *shortcutIntents {
	return &shortcutIntents{calls: make(chan [2]string, 4)}
}

func (s *shortcutIntents) SetGlobalShortcut(_ context.Context, enabled bool, accelerator string) error {
	s.calls <- [2]string{fmt.Sprintf("%v", enabled), accelerator}
	return s.err
}

// expectShortcut 等一条写入记录：写入在后台协程执行，这里给它两秒。
func (s *shortcutIntents) expectShortcut(t *testing.T, wantEnabled, wantAccelerator string) {
	t.Helper()
	select {
	case got := <-s.calls:
		if got[0] != wantEnabled || got[1] != wantAccelerator {
			t.Fatalf("写入了 %v，想要 [%s %s]", got, wantEnabled, wantAccelerator)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("没有等到 SetGlobalShortcut(%s, %s)", wantEnabled, wantAccelerator)
	}
}

// TestKeybindingsGlobalShortcutReadsStatus：分组「全局」的开关、键位与注册结果
// 都来自 Source，不是样例。
func TestKeybindingsGlobalShortcutReadsStatus(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.shortcut = GlobalShortcutSnapshot{
		Enabled: true, Accelerator: "CmdOrCtrl+Shift+Space", Label: "⌘⇧Space", Registered: true,
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secKeybindings)

	if !tt.HasText("全局") {
		t.Fatal("缺少分组「全局」")
	}
	if !tt.HasText("全局唤起") {
		t.Fatal("缺少全局唤起这一行")
	}
	if !tt.HasText("⌘⇧Space") {
		t.Fatal("键位没有来自 Source 的读数")
	}
}

// TestKeybindingsGlobalShortcutShowsRegisterFailure：注册失败（键位被占用）时
// 原位给出原因，不静默。
func TestKeybindingsGlobalShortcutShowsRegisterFailure(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.shortcut = GlobalShortcutSnapshot{
		Enabled: true, Accelerator: "CmdOrCtrl+Shift+Space", Label: "⌘⇧Space",
		Reason: "it may be used by another application (-9868)",
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secKeybindings)

	if !tt.HasText("无法注册 ⌘⇧Space：it may be used by another application (-9868)") {
		t.Fatalf("注册失败没有给出原因：%v", tt.Texts())
	}
}

// TestKeybindingsGlobalShortcutToggleSendsIntent：关掉开关走 SetGlobalShortcut，
// 带着当前键位。
func TestKeybindingsGlobalShortcutToggleSendsIntent(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.shortcut = GlobalShortcutSnapshot{
		Enabled: true, Accelerator: "CmdOrCtrl+Shift+Space", Label: "⌘⇧Space", Registered: true,
	}
	intents := newShortcutIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := openSettingsSection(t, app, secKeybindings)

	if err := tt.Click("全局唤起开关"); err != nil {
		t.Fatalf("点不到全局唤起开关：%v", err)
	}
	intents.expectShortcut(t, "false", "CmdOrCtrl+Shift+Space")
}

// TestKeybindingsGlobalShortcutChangeKey：点键位进入改键，输入字形写法后应用，
// 发出去的是规范写法。
func TestKeybindingsGlobalShortcutChangeKey(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.shortcut = GlobalShortcutSnapshot{
		Enabled: true, Accelerator: "CmdOrCtrl+Shift+Space", Label: "⌘⇧Space", Registered: true,
	}
	intents := newShortcutIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := openSettingsSection(t, app, secKeybindings)

	if err := tt.Click("全局唤起快捷键"); err != nil {
		t.Fatalf("点不到键位按钮：%v", err)
	}
	tt.Frame()
	if !globalShortcutEdit.editing {
		t.Fatal("点键位没有进入改键态")
	}
	// 输入框里改成 ⌘⇧K（走字形写法），回车等同「应用」。
	globalShortcutEdit.draft = "⌘⇧K"
	if err := tt.Click("应用全局快捷键"); err != nil {
		t.Fatalf("点不到「应用」：%v", err)
	}
	intents.expectShortcut(t, "true", "Cmd+Shift+K")
	tt.Frame()
	if globalShortcutEdit.editing {
		t.Fatal("保存后仍停在改键态")
	}
}

// TestKeybindingsGlobalShortcutRejectsInvalidKey：认不出的键位就地写原因，不发
// 写入，开关不动。
func TestKeybindingsGlobalShortcutRejectsInvalidKey(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.shortcut = GlobalShortcutSnapshot{
		Enabled: true, Accelerator: "CmdOrCtrl+Shift+Space", Label: "⌘⇧Space", Registered: true,
	}
	intents := newShortcutIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := openSettingsSection(t, app, secKeybindings)

	if err := tt.Click("全局唤起快捷键"); err != nil {
		t.Fatalf("点不到键位按钮：%v", err)
	}
	tt.Frame()
	globalShortcutEdit.draft = "K"
	if err := tt.Click("应用全局快捷键"); err != nil {
		t.Fatalf("点不到「应用」：%v", err)
	}
	tt.Frame()
	if !tt.HasText("键位无法识别：快捷键至少要有一个修饰键") {
		t.Fatalf("没有就地给出原因：%v", tt.Texts())
	}
	select {
	case got := <-intents.calls:
		t.Fatalf("不合法的键位不应写入：%v", got)
	default:
	}
}

// TestKeybindingsGlobalShortcutDemoTogglesLocally：没有 Source（-demo）时开关与
// 键位只改本端界面状态，并给出关闭说明。
func TestKeybindingsGlobalShortcutDemoTogglesLocally(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault})
	tt := openSettingsSection(t, app, secKeybindings)
	if want := shortcut.Label(shortcut.DefaultAccelerator); !tt.HasText(want) {
		t.Fatalf("样例模式没有默认键位 %q：%v", want, tt.Texts())
	}

	if err := tt.Click("全局唤起开关"); err != nil {
		t.Fatalf("点不到全局唤起开关：%v", err)
	}
	tt.Frame()
	if app.globalShortcutOn {
		t.Fatal("样例模式关掉开关没有生效")
	}
	if !tt.HasText("已关闭：在其它应用里按下不会唤起。") {
		t.Fatalf("关掉后没有给说明：%v", tt.Texts())
	}
}

// TestFocusComposerGivesInputFocus：全局唤起后输入区拿到焦点（请求只消费一次）。
func TestFocusComposerGivesInputFocus(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	app.FocusComposer()
	tt.Frame()
	if !tt.Focused("消息") {
		t.Fatal("FocusComposer 后输入区没有焦点")
	}
	if app.focusComposer.Load() {
		t.Fatal("焦点请求没有被消费，下一帧还会再抢一次")
	}
}

// TestSettingsKeybindingsVariantOpensSection：-variant settings-keybindings 直接
// 停在快捷键节（截图与走查用）。
func TestSettingsKeybindingsVariantOpensSection(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Variant: VariantSettingsKeybindings})
	if app.screen != ScreenSettings || app.settingsSection != secKeybindings {
		t.Fatalf("变体没有打开快捷键节：screen=%v section=%v", app.screen, app.settingsSection)
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if !tt.HasText("全局唤起") {
		t.Fatal("快捷键节没有画出全局唤起")
	}
}
