package ui

import (
	"image/color"
	"math"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// TestThemeTokens 逐项对 docs/domains/U2-visual.md §8 的 token 表：
// 主题字段（§3.4 映射）、框架没有字段的四个视图 token、间距与度量。
func TestThemeTokens(t *testing.T) {
	light := theme(false, gui.Preferences{})
	dark := theme(true, gui.Preferences{})

	colors := []struct {
		name string
		got  gui.Color
		want string
	}{
		{"明 bg", light.Background, "#f6fdff"},
		{"明 surface", light.Surface, "#ffffff"},
		{"明 surface-hover", light.SurfaceHover, "#f5f5f5"},
		{"明 surface-pressed", light.SurfacePressed, "#ebebeb"},
		{"明 border", light.Border, "#e5e5e5"},
		{"明 text", light.Text, "#0a0a0a"},
		{"明 text-secondary", light.TextMuted, "#737373"},
		{"明 accent", light.Accent, "#349651"},
		{"明 accent-hover", light.AccentHover, "#40a35c"},
		{"明 accent-pressed", light.AccentPressed, "#218a45"},
		{"明 accent-on", light.AccentText, "#ffffff"},
		{"明 success", light.Success, "#009966"},
		{"明 warning", light.Warning, "#e17100"},
		{"明 danger", light.Danger, "#e7000b"},
		{"明 focus", light.Focus, "#a1a1a1"},
		{"明 inverse", light.Inverse, "#0a0a0a"},
		{"明 inverse-text", light.InverseText, "#ffffff"},

		{"暗 bg", dark.Background, "#090c0d"},
		{"暗 surface", dark.Surface, "#0c0c0c"},
		{"暗 surface-hover", dark.SurfaceHover, "#262626"},
		{"暗 surface-pressed", dark.SurfacePressed, "#2b2b2b"},
		{"暗 border", dark.Border, "#2e2e2e"},
		{"暗 text", dark.Text, "#ececec"},
		{"暗 text-secondary", dark.TextMuted, "#8a8a8a"},
		{"暗 accent", dark.Accent, "#63cb7b"},
		{"暗 accent-on", dark.AccentText, "#010602"},
		{"暗 success", dark.Success, "#00d492"},
		{"暗 warning", dark.Warning, "#ffb900"},
		{"暗 danger", dark.Danger, "#ff6467"},
		{"暗 focus", dark.Focus, "#63cb7b"},

		// 框架没有字段的 accent-ink（§8.1）。边界只有主题 Border 一套（UI-18）。
		{"明 accent-ink", colorAccentInk.in(light), "#197037"},
		{"暗 accent-ink", colorAccentInk.in(dark), "#63cb7b"},
	}
	for _, c := range colors {
		if want := gui.Hex(c.want); c.got != want {
			t.Errorf("%s = %v，想要 %s", c.name, c.got, c.want)
		}
	}

	// 度量（§8.3、§8.5）。
	metrics := []struct {
		name      string
		got, want float32
	}{
		{"Theme.Spacing", light.Spacing, 3.5},
		// 字号不覆盖：跟框架的平台默认（macOS 13、其它 14），所以这里比的是同一个值。
		{"Theme.FontSize", light.FontSize, gui.LightTheme().FontSize},
		{"Theme.Radius", light.Radius, 6.75},
		{"Theme.FocusRingWidth", light.FocusRingWidth, 2},
		{"Theme.FocusRingOffset", light.FocusRingOffset, 1},
		{"Theme.ScrollbarWidth", light.ScrollbarWidth, 6},
		{"space-12", space12, 42},
		{"row-regular", rowRegular, 28},
		{"row-compact", rowCompact, 24},
		{"control-mini", controlMini, 20},
		{"icon-inline", iconInline, 16},
		{"control-pad-v", controlPadV, 6},
		{"control-pad-h", controlPadH, 14},
		{"radius", radius, 6.75},
		{"radius-small", radiusSmall, 4.75},
		{"radius-card", radiusCard, 8.75},
		{"radius-window-card", radiusWindowCard, 12.75},
		{"radius-code", radiusCode, 8.75},
		{"hit-min", hitMin, 24},
		{"stroke-chevron", strokeChevron, 1.5},
		{"context-near-percent", contextNearPercent, 90},
		{"设置导航宽", SettingsNavWidth, 224},
		{"设置导航行高", SettingsNavRowHeight, 35},
	}
	for _, m := range metrics {
		if m.got != m.want {
			t.Errorf("%s = %v，想要 %v", m.name, m.got, m.want)
		}
	}
	if light.TooltipDelay != tooltipDelay || tooltipDelay != 600*time.Millisecond {
		t.Errorf("TooltipDelay = %v，想要 %v", light.TooltipDelay, tooltipDelay)
	}
	if tooltipWarm != 400*time.Millisecond {
		t.Errorf("tooltip-warm = %v，想要 400ms", tooltipWarm)
	}
	if motionTransition != 200*time.Millisecond || motionSpring != 500*time.Millisecond ||
		motionContent != 350*time.Millisecond {
		t.Errorf("动效 = %v/%v/%v，想要 200/500/350ms",
			motionTransition, motionSpring, motionContent)
	}
	if caretCycle != 1060*time.Millisecond || caretWidth != 2 || caretHeight != 14 {
		t.Errorf("光标 = %v/%v×%v，想要 1060ms、2×14", caretCycle, caretWidth, caretHeight)
	}
}

// TestFontSteps 对 §8.2 的字阶：正文/代码给倍率，其余交给框架度量。
func TestFontSteps(t *testing.T) {
	steps := []struct {
		name string
		step fontStep
		size float32
		line float32
		w    int
	}{
		{"正文", typeBody, 12.25, 18, 400},
		{"正文强调", typeBodyStrong, 12.25, 18, 600},
		{"次要", typeSecondary, 12.25, 0, 400},
		{"次要强调", typeSecondaryStrong, 12.25, 0, 500},
		{"标题行会话名", typeTitleRow, 15, 0, 600},
		{"极小", typeMicro, 10.5, 14, 400},
		{"代码", typeCode, 13, 20, 400},
		{"h1", typeH1, 22.4, 33.6, 700},
		{"h2", typeH2, 20, 25, 600},
		{"h3", typeH3, 16.1, 24.15, 600},
		{"h4", typeH4, 14, 21, 600},
		{"h5", typeH5, 13.3, 19.95, 600},
		{"h6", typeH6, 12.6, 18.9, 600},
	}
	for _, s := range steps {
		if s.step.size != s.size || s.step.line != s.line || s.step.weight != s.w {
			t.Errorf("%s = %v/%v/%v，想要 %v/%v/%v",
				s.name, s.step.size, s.step.line, s.step.weight, s.size, s.line, s.w)
		}
	}
	if typeCode.family != codeFont {
		t.Errorf("代码字族 = %q，想要 %q", typeCode.family, codeFont)
	}
}

// TestThemeFollowsPreferences 对 §7.4 的文本缩放与 §7.5 的高对比。
func TestThemeFollowsPreferences(t *testing.T) {
	scaled := theme(false, gui.Preferences{TextScale: 1.5})
	if want := gui.LightTheme().FontSize * 1.5; scaled.FontSize != want {
		t.Fatalf("文本缩放 1.5 的主题字号 = %v，想要 %v", scaled.FontSize, want)
	}
	// 正文取参考的 text-sm：平台默认字号（参考的根字号 14）的 0.875 倍，随平台同比缩放
	// （macOS 13 → 11.375）。
	if got, want := typeBody.sizeIn(scaled), gui.LightTheme().FontSize*1.5*(12.25/14); got != want {
		t.Fatalf("文本缩放 1.5 的正文 = %v，想要 %v", got, want)
	}
	// 次要级与正文同字号（参考同为 text-sm），随平台字号同比缩放。
	if got, want := typeSecondary.sizeIn(scaled), gui.LightTheme().FontSize*1.5*(12.25/14); got != want {
		t.Fatalf("文本缩放 1.5 的次要 = %v，想要 %v", got, want)
	}

	contrast := theme(false, gui.Preferences{HighContrast: true})
	base := theme(false, gui.Preferences{})
	// 沿用框架 Theme.follow 的同一套做法，而不是另写 v1 的覆盖规则（U2 §7.5）。
	if want := base.Border.Mix(base.Text, 0.45); contrast.Border != want {
		t.Errorf("高对比 border = %v，想要 %v", contrast.Border, want)
	}
	if want := base.TextMuted.Mix(base.Text, 0.4); contrast.TextMuted != want {
		t.Errorf("高对比 text-secondary = %v，想要 %v", contrast.TextMuted, want)
	}
	if contrast.Focus.A != 255 {
		t.Errorf("高对比 focus 透明度 = %v，想要不透明", contrast.Focus.A)
	}
	if contrast.FocusRingWidth != 2 {
		t.Errorf("高对比焦点环 = %v，想要沿用框架的 2", contrast.FocusRingWidth)
	}
}

// TestViewAppliesTheme 证明界面真的用这套 token：窗口底色取 bg，暗色随系统。
func TestViewAppliesTheme(t *testing.T) {
	for _, c := range []struct {
		name string
		dark bool
		want gui.Color
	}{
		{"明", false, gui.Hex("#f6fdff")},
		{"暗", true, gui.Hex("#090c0d")},
	} {
		app := NewApp(Options{State: StateDefault})
		tt := gui.NewTesterWith(app.View, WindowWidth, WindowHeight,
			gui.TesterOptions{Dark: c.dark})
		// 主区左上（左栏右侧、标题行下方）是窗口底。
		got := tt.Image().RGBAAt(SidebarWidth+8, HeaderHeight+8)
		want := color.RGBA{R: c.want.R, G: c.want.G, B: c.want.B, A: 0xff}
		if got != want {
			t.Fatalf("%s：主区底色 = %v，想要 %v", c.name, got, want)
		}
	}
}

// TestDisabledContrastUsesFrameworkOpacity 按框架算法重算禁用态（C1）：
// 不换面，整棵子树压 0.5。禁用次按钮标签 = text 叠 surface 一半 → 3.69:1；
// 禁用主按钮填充 = accent 叠 surface 一半 → #9acba8 一类的淡强调面。
func TestDisabledContrastUsesFrameworkOpacity(t *testing.T) {
	light := theme(false, gui.Preferences{})

	label := mixOver(light.Text, light.Surface, 0.5)
	if got := contrastRatio(label, light.Surface); math.Abs(got-3.69) > 0.02 {
		t.Errorf("禁用次按钮标签对 surface = %.2f:1，想要 3.69:1", got)
	}

	fill := mixOver(light.Accent, light.Surface, 0.5)
	want := [3]int{154, 203, 168}
	got := [3]int{int(fill.R), int(fill.G), int(fill.B)}
	for i := range want {
		if d := got[i] - want[i]; d < -2 || d > 2 {
			t.Fatalf("禁用主按钮填充 = %v，想要接近 %v", got, want)
		}
	}
}

// mixOver 把 fg 以 alpha 叠在 bg 上，即框架禁用态的半透明合成。
func mixOver(fg, bg gui.Color, alpha float32) gui.Color {
	mix := func(f, b uint8) uint8 {
		return uint8(math.Round(float64(alpha)*float64(f) + (1-float64(alpha))*float64(b)))
	}
	return gui.Color{R: mix(fg.R, bg.R), G: mix(fg.G, bg.G), B: mix(fg.B, bg.B), A: 255}
}

// contrastRatio 是 WCAG 2.1 的相对亮度对比度。
func contrastRatio(a, b gui.Color) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(c gui.Color) float64 {
	lin := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}
