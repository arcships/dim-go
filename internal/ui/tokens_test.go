package ui

import (
	"image/color"
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
		{"明 bg", light.Background, "#ffffff"},
		{"明 surface", light.Surface, "#f4f4f5"},
		{"明 surface-hover", light.SurfaceHover, "#e9e9ec"},
		{"明 surface-pressed", light.SurfacePressed, "#dddde1"},
		{"明 border", light.Border, "#d9d9de"},
		{"明 text", light.Text, "#18181b"},
		{"明 text-secondary", light.TextMuted, "#5f5f68"},
		{"明 accent", light.Accent, "#2563eb"},
		{"明 accent-hover", light.AccentHover, "#1d4ed8"},
		{"明 accent-pressed", light.AccentPressed, "#1e40af"},
		{"明 accent-on", light.AccentText, "#ffffff"},
		{"明 success", light.Success, "#15803d"},
		{"明 warning", light.Warning, "#b45309"},
		{"明 danger", light.Danger, "#b91c1c"},
		{"明 focus", light.Focus, "#2563eb"},
		{"明 inverse", light.Inverse, "#18181b"},
		{"明 inverse-text", light.InverseText, "#ffffff"},

		{"暗 bg", dark.Background, "#18181b"},
		{"暗 surface", dark.Surface, "#27272a"},
		{"暗 surface-hover", dark.SurfaceHover, "#323236"},
		{"暗 surface-pressed", dark.SurfacePressed, "#3c3c41"},
		{"暗 border", dark.Border, "#3f3f46"},
		{"暗 text", dark.Text, "#f4f4f5"},
		{"暗 text-secondary", dark.TextMuted, "#a1a1aa"},
		{"暗 accent", dark.Accent, "#2563eb"},
		{"暗 accent-on", dark.AccentText, "#ffffff"},
		{"暗 success", dark.Success, "#22c55e"},
		{"暗 warning", dark.Warning, "#f59e0b"},
		{"暗 danger", dark.Danger, "#f87171"},
		{"暗 focus", dark.Focus, "#60a5fa"},

		// 框架没有字段的两个 token（§8.1）。
		{"明 border-strong", colorBorderStrong.in(light), "#85858b"},
		{"暗 border-strong", colorBorderStrong.in(dark), "#8b8b94"},
		{"明 accent-ink", colorAccentInk.in(light), "#1d4ed8"},
		{"暗 accent-ink", colorAccentInk.in(dark), "#60a5fa"},
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
		{"Theme.Spacing", light.Spacing, 4},
		{"Theme.FontSize", light.FontSize, 14},
		{"Theme.Radius", light.Radius, 6},
		{"Theme.FocusRingWidth", light.FocusRingWidth, 2},
		{"Theme.FocusRingOffset", light.FocusRingOffset, 1},
		{"Theme.ScrollbarWidth", light.ScrollbarWidth, 6},
		{"space-12", space12, 48},
		{"row-regular", rowRegular, 28},
		{"row-compact", rowCompact, 24},
		{"control-mini", controlMini, 20},
		{"icon-inline", iconInline, 16},
		{"radius", radius, 6},
		{"radius-code", radiusCode, 8},
		{"hit-min", hitMin, 24},
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
		{"正文", typeBody, 14, 21, 400},
		{"正文强调", typeBodyStrong, 14, 21, 600},
		{"次要", typeSecondary, 12.25, 0, 400},
		{"次要强调", typeSecondaryStrong, 12.25, 0, 600},
		{"极小", typeMicro, 10.5, 0, 400},
		{"代码", typeCode, 12.25, 17.15, 400},
		{"标题行会话名", typeTitleRow, 14, 0, 600},
		{"h1", typeH1, 22.4, 33.6, 700},
		{"h2", typeH2, 18.9, 28.35, 650},
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
	if scaled.FontSize != 21 {
		t.Fatalf("文本缩放 1.5 的主题字号 = %v，想要 21", scaled.FontSize)
	}
	if got := typeBody.sizeIn(scaled); got != 21 {
		t.Fatalf("文本缩放 1.5 的正文 = %v，想要 21", got)
	}
	if got := typeSecondary.sizeIn(scaled); got != 18.375 {
		t.Fatalf("文本缩放 1.5 的次要 = %v，想要 18.375", got)
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
		{"明", false, gui.Hex("#ffffff")},
		{"暗", true, gui.Hex("#18181b")},
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
