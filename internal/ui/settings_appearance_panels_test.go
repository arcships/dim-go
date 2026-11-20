package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 外观」本版可做条目的回归：主题回滚与回落、预览图标的兜底、
// 字体面板的搜索 / 应用 / 重试、字号与代码字号的键盘、代码主题与预览同步。
//
// 证据：每个用例对应 docs/domains/requirements/requirements-settings.md 的条目，
// 注释里写需求 ID；文案断言用屏上原文。

// failingPrefsIntents 让 SetPreferences 失败，其余写入不实现（调到就 panic）。
type failingPrefsIntents struct {
	Intents
	fail bool
}

func (f *failingPrefsIntents) SetPreferences(context.Context, PreferencesSnapshot) error {
	if f.fail {
		return errors.New("写入被拒绝")
	}
	return nil
}

// TestAppearanceThemeSaveFailureRollsBack 断言主题落盘失败后角标回滚到原主题
// （R-APP-028 / R-APP-047）。
func TestAppearanceThemeSaveFailureRollsBack(t *testing.T) {
	src := wiredSource()
	src.prefs = PreferencesSnapshot{Theme: "明"}
	app := NewApp(Options{State: StateDefault, Source: src,
		Intents: &failingPrefsIntents{fail: true}})
	tt := openSettingsSection(t, app, secAppearance)

	if err := tt.Click("主题：深色"); err != nil {
		t.Fatalf("点不到深色卡：%v", err)
	}
	// 回滚在失败回执交回渲染线程后发生：等观察值，不数帧。
	waitFrame(t, tt, "主题回滚", func() bool { return app.themePref == "明" })
	if app.themePref != "明" {
		t.Fatalf("落盘失败后主题 = %q，想要回滚到「明」", app.themePref)
	}
	if info, ok := tt.Access("主题：浅色"); !ok || !info.Checked {
		t.Fatalf("角标没有回到浅色卡：%+v（找到 %v）", info, ok)
	}
	if _, ok := tt.Find("操作未完成"); !ok {
		t.Fatal("落盘失败没有提示")
	}
}

// TestAppearanceThemePreviewIconFallsBack 断言主题预览图标缺资源时留同样高度的
// 底色块（R-APP-038），选中标记缺资源时用纯色圆点兜底（R-APP-065）。
func TestAppearanceThemePreviewIconFallsBack(t *testing.T) {
	icons := appearanceThemeOptions[0].icon
	appearanceThemeOptions[0].icon = nil
	defer func() { appearanceThemeOptions[0].icon = icons }()

	src := wiredSource()
	src.prefs = PreferencesSnapshot{Theme: "明"}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secAppearance)
	for _, want := range []string{"浅色", "深色", "跟随系统"} {
		if _, ok := tt.Find("主题：" + want); !ok {
			t.Fatalf("预览图标缺失时 %q 卡不见了", want)
		}
	}

	screenCornerDotGlyphMissing = true
	defer func() { screenCornerDotGlyphMissing = false }()
	tt.Frame()
	if _, ok := tt.Find("主题：浅色"); !ok {
		t.Fatal("选中标记缺资源时主题卡不见了")
	}
	if info, ok := tt.Access("主题：浅色"); !ok || !info.Checked {
		t.Fatalf("兜底圆点没有保留选中态：%+v（找到 %v）", info, ok)
	}
}

// TestAppearanceSystemUnknownFallsBackLight 断言系统报不出外观时按浅色渲染、卡
// 仍选中，并在行下写原因（R-APP-055）。
func TestAppearanceSystemUnknownFallsBackLight(t *testing.T) {
	src := wiredSource()
	src.prefs = PreferencesSnapshot{Theme: "跟随系统"}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secAppearance)

	// 宿主报不出外观：按浅色渲染，且卡仍停在「跟随系统」（卡选的是偏好，不是
	// 系统报出来的深浅）。
	appearanceSystemKnown = false
	defer func() { appearanceSystemKnown = true }()
	if got := appearanceSystemAppearanceError(nil, "跟随系统"); got != noteSystemAppearanceUnknown {
		t.Fatalf("系统外观未知时的说明 = %q，想要 %q", got, noteSystemAppearanceUnknown)
	}
	if got := appearanceSystemAppearanceError(nil, "明"); got != "" {
		t.Fatalf("不跟随时不该写系统外观的说明：%q", got)
	}
	if info, ok := tt.Access("主题：跟随系统"); !ok || !info.Checked {
		t.Fatalf("回落时没有停在「跟随系统」卡：%+v（找到 %v）", info, ok)
	}
	if _, ok := tt.Find("主题：浅色"); !ok {
		t.Fatal("回落时浅色卡不见了")
	}
}

// TestAppearanceFontPanelSearchesAndApplies 断言字体面板的搜索、过滤、应用与收起
// （R-APP-130 / 131 / 173 / 198）。
func TestAppearanceFontPanelSearchesAndApplies(t *testing.T) {
	defer stubFontFamilies(gui.FontFamily{Name: "Menlo", Monospaced: true},
		gui.FontFamily{Name: "PingFang SC"},
		gui.FontFamily{Name: "Noto Sans SC"})()

	intents := &p0Intents{prefs: make(chan PreferencesSnapshot, 8)}
	app := NewApp(Options{State: StateDefault, Source: wiredSource(), Intents: intents})
	tt := openSettingsSection(t, app, secAppearance)

	clickRole(t, tt, gui.RolePopUpButton, "代码字体")
	if _, ok := tt.Find("搜索字体"); !ok {
		t.Fatalf("字体面板没有搜索框：%v", tt.Texts())
	}
	if _, ok := tt.Find("Noto Sans SC"); !ok {
		t.Fatal("字体面板没有列出本机字体")
	}
	appearanceScreen.fontQuery = "men"
	tt.Frame()
	if _, ok := tt.Find("Menlo"); !ok {
		t.Fatalf("搜索 men 没有留下 Menlo：%v", tt.Texts())
	}
	if _, ok := tt.Find("Noto Sans SC"); ok {
		t.Fatal("搜索 men 仍列出 Noto Sans SC")
	}
	if err := tt.Click("Menlo"); err != nil {
		t.Fatalf("点不到 Menlo：%v", err)
	}
	tt.Frame()
	if app.appearance.CodeFont != "Menlo" {
		t.Fatalf("代码字体 = %q，想要 Menlo", app.appearance.CodeFont)
	}
	if _, ok := tt.Find("搜索字体"); ok {
		t.Fatal("选定后字体面板没有收起")
	}
	prefs := waitPrefsOfApp(t, intents.prefs)
	if prefs.Appearance.CodeFont != "Menlo" {
		t.Fatalf("落盘的代码字体 = %q", prefs.Appearance.CodeFont)
	}
}

// TestAppearanceFontListRetry 断言系统字体清单读不到时写原因并给「重试」，重试后
// 清单回来（R-APP-224）。
func TestAppearanceFontListRetry(t *testing.T) {
	families := []gui.FontFamily{}
	defer stubFontFamilies(families...)()
	restore := stubFontFamilies(families...)
	defer restore()

	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secAppearance)
	clickRole(t, tt, gui.RolePopUpButton, "UI 字体")
	if !tt.HasText(noteFontListMissing) {
		t.Fatalf("读不到字体清单时没有写原因：%v", tt.Texts())
	}
	// 清单回来后重试：面板换成字体列表。
	systemFontFamilies = func() []gui.FontFamily {
		return []gui.FontFamily{{Name: "Noto Sans SC"}}
	}
	if err := tt.Click("重试"); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	tt.Frame()
	if tt.HasText(noteFontListMissing) {
		t.Fatalf("重试后仍写着读不到：%v", tt.Texts())
	}
	if _, ok := tt.Find("Noto Sans SC"); !ok {
		t.Fatalf("重试后没有列出字体：%v", tt.Texts())
	}
}

// TestAppearanceFontSearchEmpty 断言没有匹配字体时有空态（R-APP-20 的空态行）。
func TestAppearanceFontSearchEmpty(t *testing.T) {
	defer stubFontFamilies(gui.FontFamily{Name: "Menlo", Monospaced: true})()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secAppearance)
	clickRole(t, tt, gui.RolePopUpButton, "代码字体")
	appearanceScreen.fontQuery = "zzz"
	tt.Frame()
	if !tt.HasText("没有匹配的字体") {
		t.Fatalf("没有匹配字体时没有空态：%v", tt.Texts())
	}
}

// TestAppearanceUISizeKeyboardHomeEnd 断言 UI 字号段的分段控件用 Home / End 到
// 两端档（R-APP-265）。
func TestAppearanceUISizeKeyboardHomeEnd(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secAppearance)

	info, ok := tt.AccessByRole(gui.RoleRadio, "中")
	if !ok {
		t.Fatalf("找不到「中」段：%v", tt.Texts())
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Frame()
	tt.Key(0, gui.KeyEnd)
	tt.Frame()
	if want := appearanceUISizes[len(appearanceUISizes)-1].scale; app.appearance.UIScale != want {
		t.Fatalf("End 后界面字号倍率 = %v，想要 %v", app.appearance.UIScale, want)
	}
	tt.Key(0, gui.KeyHome)
	tt.Frame()
	if want := appearanceUISizes[0].scale; app.appearance.UIScale != want {
		t.Fatalf("Home 后界面字号倍率 = %v，想要 %v", app.appearance.UIScale, want)
	}
}

// TestAppearanceCodeSizeKeyboardHomeEnd 断言代码字号滑块的 Home / End 到两端
// （R-APP-338）。
func TestAppearanceCodeSizeKeyboardHomeEnd(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secAppearance)

	clickRole(t, tt, gui.RoleSlider, "代码字号")
	tt.Key(0, gui.KeyEnd)
	tt.Frame()
	if got := appearanceCodeSize(app.appearance.CodeSize); got != appearanceCodeSizeMax {
		t.Fatalf("End 后代码字号 = %v，想要 %v", got, appearanceCodeSizeMax)
	}
	tt.Key(0, gui.KeyHome)
	tt.Frame()
	if got := appearanceCodeSize(app.appearance.CodeSize); got != appearanceCodeSizeMin {
		t.Fatalf("Home 后代码字号 = %v，想要 %v", got, appearanceCodeSizeMin)
	}
}

// TestAppearanceCodeThemeAppliesToPreview 断言选一套深色主题后预览与代码区都用它
// 的深色配色（R-APP-384 / 403 / 419）。
func TestAppearanceCodeThemeAppliesToPreview(t *testing.T) {
	intents := &p0Intents{prefs: make(chan PreferencesSnapshot, 8)}
	app := NewApp(Options{State: StateDefault, Source: wiredSource(), Intents: intents})
	tt := openSettingsSection(t, app, secAppearance)

	clickRole(t, tt, gui.RolePopUpButton, "代码主题")
	if _, ok := tt.Find("One Dark Pro"); !ok {
		t.Fatalf("代码主题面板没有列出 One Dark Pro：%v", tt.Texts())
	}
	if err := tt.Click("One Dark Pro"); err != nil {
		t.Fatalf("点不到 One Dark Pro：%v", err)
	}
	tt.Frame()
	if app.appearance.CodeTheme != "One Dark Pro" {
		t.Fatalf("代码主题 = %q，想要 One Dark Pro", app.appearance.CodeTheme)
	}
	if _, ok := tt.Find("深色"); !ok {
		t.Fatal("代码主题面板没有分组标记")
	}
	want, _ := appearanceCodeInkFor("One Dark Pro", false)
	if want.bg != gui.Hex("#282c34") {
		t.Fatalf("深色主题在浅色界面下取到的底不是深色：%v", want.bg)
	}
	prefs := waitPrefsOfApp(t, intents.prefs)
	if prefs.Appearance.CodeTheme != "One Dark Pro" {
		t.Fatalf("落盘的代码主题 = %q", prefs.Appearance.CodeTheme)
	}
}

// TestAppearanceLanguageSwitchUnavailable 断言程序语言只有简体中文一套，下拉置灰
// 并写原因（R-APP-085 / 086 / 098 未实现：缺 i18n 层）。
func TestAppearanceLanguageSwitchUnavailable(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secAppearance)
	info, ok := tt.AccessByRole(gui.RolePopUpButton, "程序语言")
	if !ok || !info.Disabled {
		t.Fatalf("程序语言下拉没有置灰：%+v（找到 %v）", info, ok)
	}
	if !strings.Contains(joinedText(tt), noteAppearanceLanguage) {
		t.Fatalf("没有写语言不可用的原因：%v", tt.Texts())
	}
}
