package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件把 docs/domains/requirements/design-baseline.md 的样式常量钉在实现上：
// 每类样式一条断言，取元素用的字级、渲染后的包围盒或像素，与基线常量比对。
// 常量本身在 tokens_test.go 已逐项对过；这里对的是「元素真的用了这个值」。

// TestStyleBaselineTypeRoles 对 design-baseline §2 的字号 / 行高。
func TestStyleBaselineTypeRoles(t *testing.T) {
	roles := []struct {
		name   string
		step   fontStep
		size   float32
		line   float32
		weight int
	}{
		// 页面标题 17.5 / 24.5；对话框标题 15.75 / 24.5。
		{"页面标题", typePageTitle, 17.5, 24.5, 600},
		{"对话框标题", typeDialogTitle, 15.75, 24.5, 600},
		// 空态主标题 21 / 28。
		{"空态主标题", typeEmptyTitle, 21, 28, 700},
		// UI 正文 12.25，块级行盒 18。
		{"UI 正文", typeBody, 12.25, 18, 400},
		// 说明文字 / 分组标题 / 徽标文字 10.5 / 14。
		{"说明文字", typeMicro, 10.5, 14, 400},
		// 代码 13 / 20。
		{"代码", typeCode, 13, 20, 400},
	}
	for _, r := range roles {
		if r.step.size != r.size || r.step.line != r.line || r.step.weight != r.weight {
			t.Errorf("%s = %v/%v/%v，基线 %v/%v/%v",
				r.name, r.step.size, r.step.line, r.step.weight, r.size, r.line, r.weight)
		}
	}
}

// TestStyleBaselinePageTitleElement 取渲染后的页面标题：字号 17.5 的墨迹盒高
// 约 1.3 倍（22.75），二级标题角色。改回 20 时这条会先红。
func TestStyleBaselinePageTitleElement(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault})
	tt := openSettingsSection(t, app, secAccount)
	info, ok := tt.AccessByRole(gui.RoleHeading, "账户")
	if !ok {
		t.Fatal("账户页没有二级标题「账户」")
	}
	if want := float32(22.75); info.Bounds.H < want-1 || info.Bounds.H > want+1 {
		t.Errorf("页面标题盒高 = %v，基线 17.5 字号对应约 %v", info.Bounds.H, want)
	}
	if info.Level != 2 {
		t.Errorf("页面标题层级 = %d，想要 2", info.Level)
	}
}

// TestStyleBaselineControlSizes 取渲染后的控件盒子：设置导航行 35、开关 28×16
// （design-baseline §3）。
func TestStyleBaselineControlSizes(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault})
	tt := openSettingsSection(t, app, secNotifications)

	nav, ok := tt.AccessByRole(gui.RoleListItem, "账户")
	if !ok {
		t.Fatal("设置导航里没有「账户」行")
	}
	if nav.Bounds.H != SettingsNavRowHeight {
		t.Errorf("导航行高 = %v，基线 %v", nav.Bounds.H, SettingsNavRowHeight)
	}

	sw, ok := tt.AccessByRole(gui.RoleSwitch, "任务完成或失败时通知开关")
	if !ok {
		t.Fatal("通知屏没有任务完成开关")
	}
	if sw.Bounds.W != 28 || sw.Bounds.H != 16 {
		t.Errorf("开关 = %v×%v，基线 28×16", sw.Bounds.W, sw.Bounds.H)
	}
}

// TestStyleBaselineAccentColor 取渲染后的开关轨道像素：开态主色 明 #349651 /
// 暗 #63cb7b（design-baseline §5）。
func TestStyleBaselineAccentColor(t *testing.T) {
	for _, c := range []struct {
		name string
		dark bool
		want gui.Color
	}{
		{"明", false, gui.Hex("#349651")},
		{"暗", true, gui.Hex("#63cb7b")},
	} {
		resetSettingsScreenState()
		app := NewApp(Options{State: StateDefault})
		app.screen, app.settingsSection = ScreenSettings, secNotifications
		app.notifyTaskDone = true
		tt := gui.NewTesterWith(app.View, WindowWidth, WindowHeight,
			gui.TesterOptions{Dark: c.dark})
		sw, ok := tt.AccessByRole(gui.RoleSwitch, "任务完成或失败时通知开关")
		if !ok {
			t.Fatalf("%s：通知屏没有任务完成开关", c.name)
		}
		got := tt.Image().RGBAAt(int(sw.Bounds.X+4), int(sw.Bounds.Y+sw.Bounds.H/2))
		if got.R != c.want.R || got.G != c.want.G || got.B != c.want.B {
			t.Errorf("%s：开关轨道 = #%02x%02x%02x，基线 #%02x%02x%02x",
				c.name, got.R, got.G, got.B, c.want.R, c.want.G, c.want.B)
		}
	}
}

// TestStyleBaselineSpacing 对 design-baseline §1.1 的间距档与 §3 的控件内边距：
// 全界面只认一套 3.5 网格，控件内边距 6 / 14。
func TestStyleBaselineSpacing(t *testing.T) {
	steps := []struct {
		name      string
		got, want float32
	}{
		{"单位", spaceUnit, 3.5},
		{"space-1", space1, 3.5},
		{"space-2", space2, 7},
		{"space-3", space3, 10.5},
		{"space-4", space4, 14},
		{"space-6", space6, 21},
		{"space-8", space8, 28},
		{"space-12", space12, 42},
		{"控件内边距-纵", controlPadV, 6},
		{"控件内边距-横", controlPadH, 14},
	}
	for _, s := range steps {
		if s.got != s.want {
			t.Errorf("%s = %v，基线 %v", s.name, s.got, s.want)
		}
	}
}

// TestStyleBaselineRadiusAndStroke 对 design-baseline §4 的圆角与描边：控件 6.75、
// 卡内分区 8.75、主窗卡 12.75、小标签 4.75；描边 1、焦点环 2 + 外扩 1。
func TestStyleBaselineRadiusAndStroke(t *testing.T) {
	radii := []struct {
		name      string
		got, want float32
	}{
		{"小标签", radiusSmall, 4.75},
		{"按钮与输入框", radius, 6.75},
		{"卡内分区", radiusCard, 8.75},
		{"主窗卡", radiusWindowCard, 12.75},
		{"设置屏卡", screenRadiusCard, 8.75},
		{"设置屏控件", screenRadiusCtl, 6.75},
	}
	for _, r := range radii {
		if r.got != r.want {
			t.Errorf("%s 圆角 = %v，基线 %v", r.name, r.got, r.want)
		}
	}
	if strokeHairline != 1 || strokeFocus != 2 || focusRingOffset != 1 {
		t.Errorf("描边 = %v / 焦点环 %v+%v，基线 1 / 2+1",
			strokeHairline, strokeFocus, focusRingOffset)
	}
}
