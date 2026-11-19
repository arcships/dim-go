package ui

import (
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 外观」按本地偏好做的回归断言（裁决 2）。
//
// 语言、字体、字号、代码主题、皮肤五项里，能真正渲染的四项（界面字体、界面字号、
// 代码字体、代码字号）与代码主题都走 preferences.json 的 appearance.*：改一次
// 落盘一次、重启恢复、下一帧生效。语言与皮肤本版没有生效路径，置灰并写原因
// （unavailable.go 的 noteAppearanceLanguage / noteSkinMissing）。

// stubFontFamilies 把"本机字体清单"换成固定的一份（headless 渲染没有真字体表），
// 返回还原函数。
func stubFontFamilies(families ...gui.FontFamily) func() {
	prev := systemFontFamilies
	systemFontFamilies = func() []gui.FontFamily { return families }
	return func() { systemFontFamilies = prev }
}

// clickRole 点一个按角色定位的控件：同一行里的标签文字与控件同名时，只按名字点
// 会落到标签上。
func clickRole(t *testing.T, tt *gui.Tester, role gui.Role, name string) {
	t.Helper()
	info, ok := tt.AccessByRole(role, name)
	if !ok {
		t.Fatalf("找不到 %v %q：%v", role, name, tt.Texts())
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Frame()
}

// waitPrefsOfApp 等一次落盘的本地偏好。
func waitPrefsOfApp(t *testing.T, ch chan PreferencesSnapshot) PreferencesSnapshot {
	t.Helper()
	select {
	case prefs := <-ch:
		return prefs
	case <-time.After(2 * time.Second):
		t.Fatal("没有落盘本地偏好")
		return PreferencesSnapshot{}
	}
}

// TestAppearancePreferencesRestore：preferences.json 里的 appearance.* 在启动时
// 进入渲染——界面字体写进主题、界面字号乘进主题字号、代码字体与字号进代码字级、
// 代码主题进 markdown 的代码配色。
func TestAppearancePreferencesRestore(t *testing.T) {
	src := wiredSource()
	src.prefs.Appearance = AppearancePreferences{
		UIFont: "PingFang SC", CodeFont: "Menlo", UIScale: 1.16, CodeSize: 18,
		CodeTheme: "One Dark Pro",
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	if app.appearance != src.prefs.Appearance {
		t.Fatalf("外观偏好没有恢复：%+v", app.appearance)
	}
	_ = gui.NewTester(app.View, WindowWidth, WindowHeight)

	// 界面字体与字号进主题。
	drawn := themeFor(false, gui.Preferences{}, app.appearanceLook())
	base := theme(false, gui.Preferences{})
	if drawn.Font != "PingFang SC" {
		t.Fatalf("主题字体 = %q，想要 PingFang SC", drawn.Font)
	}
	if drawn.FontSize != base.FontSize*1.16 {
		t.Fatalf("主题字号 = %v，想要 %v（默认 %v × 1.16）",
			drawn.FontSize, base.FontSize*1.16, base.FontSize)
	}

	// 代码字体与字号进代码字级。
	if want := codeFamilyFor("Menlo"); currentCodeLook.family != want {
		t.Fatalf("代码字体栈 = %q，想要 %q", currentCodeLook.family, want)
	}
	if want := float32(18) / defaultCodeSize; currentCodeLook.scale != want {
		t.Fatalf("代码字号倍率 = %v，想要 %v", currentCodeLook.scale, want)
	}

	// 代码主题进 markdown 的代码配色。
	md := markdownTheme(theme(false, gui.Preferences{}))
	if md.Code != gui.Hex("#abb2bf") || md.CodeBlockBackground != gui.Hex("#282c34") {
		t.Fatalf("代码配色没有跟着代码主题：%v / %v", md.Code, md.CodeBlockBackground)
	}
}

// TestAppearanceCodeSizeChangesCodeText：改代码字号后代码字级的实际字号跟着变
// （不是只改屏上的一个读数）。
func TestAppearanceCodeSizeChangesCodeText(t *testing.T) {
	src := wiredSource()
	app := NewApp(Options{State: StateDefault, Source: src})
	_ = gui.NewTester(app.View, WindowWidth, WindowHeight)
	before := typeCode.sizeIn(theme(false, gui.Preferences{}))

	app.appearance.CodeSize = 20
	_ = gui.NewTester(app.View, WindowWidth, WindowHeight)
	after := typeCode.sizeIn(theme(false, gui.Preferences{}))
	if after <= before {
		t.Fatalf("代码字号调到 20 后代码字级没有变大：%v → %v", before, after)
	}
	if want := before * 20 / defaultCodeSize; after != want {
		t.Fatalf("代码字级 = %v，想要 %v", after, want)
	}
}

// TestAppearanceFontChoiceIsPersisted：在屏上换代码字体 → 写进 App 的外观偏好并
// 落盘，下一帧代码字体栈跟着换。
func TestAppearanceFontChoiceIsPersisted(t *testing.T) {
	defer stubFontFamilies(gui.FontFamily{Name: "Menlo", Monospaced: true},
		gui.FontFamily{Name: "PingFang SC"})()

	intents := &p0Intents{prefs: make(chan PreferencesSnapshot, 8)}
	app := NewApp(Options{State: StateDefault, Source: wiredSource(), Intents: intents})
	tt := openSettingsSection(t, app, secAppearance)

	clickRole(t, tt, gui.RolePopUpButton, "代码字体")
	if err := tt.Click("Menlo"); err != nil {
		t.Fatalf("点不到 Menlo 选项：%v", err)
	}
	tt.Frame()
	if app.appearance.CodeFont != "Menlo" {
		t.Fatalf("代码字体 = %q，想要 Menlo", app.appearance.CodeFont)
	}
	prefs := waitPrefsOfApp(t, intents.prefs)
	if prefs.Appearance.CodeFont != "Menlo" {
		t.Fatalf("落盘的代码字体 = %q，想要 Menlo", prefs.Appearance.CodeFont)
	}
	if want := codeFamilyFor("Menlo"); currentCodeLook.family != want {
		t.Fatalf("代码字体栈 = %q，想要 %q", currentCodeLook.family, want)
	}
}

// TestAppearanceFontOptionDisabledWhenMissing：本机没装的字体在弹层里置灰并写原因
// （选了不生效，所以不让选）；装着的字体照常可选。
func TestAppearanceFontOptionDisabledWhenMissing(t *testing.T) {
	defer stubFontFamilies(gui.FontFamily{Name: "Menlo", Monospaced: true},
		gui.FontFamily{Name: "PingFang SC"})()

	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secAppearance)
	clickRole(t, tt, gui.RolePopUpButton, "代码字体")

	info, ok := tt.Access("JetBrains Mono")
	if !ok {
		t.Fatalf("弹层里没有列出 JetBrains Mono：%v", tt.Texts())
	}
	if !info.Disabled {
		t.Fatalf("本机没有的字体仍可选：%+v", info)
	}
	if info.Description != noteFontMissing {
		t.Fatalf("缺字体没有写原因：%q", info.Description)
	}
	if okInfo, found := tt.Access("Menlo"); !found || okInfo.Disabled {
		t.Fatalf("本机有的字体反而不可选：%+v（找到 %v）", okInfo, found)
	}
}

// TestAppearanceUISizeSegmentIsPersisted：UI 字号五档真的换字号并落盘。
func TestAppearanceUISizeSegmentIsPersisted(t *testing.T) {
	intents := &p0Intents{prefs: make(chan PreferencesSnapshot, 8)}
	app := NewApp(Options{State: StateDefault, Source: wiredSource(), Intents: intents})
	tt := openSettingsSection(t, app, secAppearance)

	clickRole(t, tt, gui.RoleRadio, "大")
	if want := float32(1.08); app.appearance.UIScale != want {
		t.Fatalf("界面字号倍率 = %v，想要 %v", app.appearance.UIScale, want)
	}
	if prefs := waitPrefsOfApp(t, intents.prefs); prefs.Appearance.UIScale != 1.08 {
		t.Fatalf("落盘的界面字号倍率 = %v，想要 1.08", prefs.Appearance.UIScale)
	}
}

// TestAppearanceLanguageAndSkinDisabled：语言与皮肤本版没有生效路径，置灰并写原因。
func TestAppearanceLanguageAndSkinDisabled(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secAppearance)

	if info, ok := tt.AccessByRole(gui.RolePopUpButton, "程序语言"); !ok || !info.Disabled {
		t.Fatalf("程序语言下拉没有置灰：%+v（找到 %v）", info, ok)
	}
	if !tt.HasText(noteAppearanceLanguage) {
		t.Fatalf("没有写语言不可用的原因：%v", tt.Texts())
	}
	if !tt.HasText(noteSkinMissing) {
		t.Fatalf("没有写皮肤不可用的原因：%v", tt.Texts())
	}
	for _, label := range []string{"皮肤：Windows 98", "皮肤：默认"} {
		info, ok := tt.AccessByRole(gui.RoleRadio, label)
		if !ok {
			t.Fatalf("%q 不见了（形态要留）", label)
		}
		if !info.Disabled {
			t.Fatalf("%q 仍可选（选了不生效）：%+v", label, info)
		}
	}
}
