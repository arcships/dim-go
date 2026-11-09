package ui

import (
	"math"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
)

// luminance is a color's perceived lightness between 0 and 1, the measure the
// tests use to compare two faces: the semantic layer promises a direction, not
// an exact value.
func luminance(c ui.Color) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
}

// namedColor is one semantic token, named as its view asks for it, so a
// failure says which token is missing rather than which field. opaque is false
// for the tokens that are meant to be a translucent edge or ring: a wash may
// keep its alpha, a face may not.
type namedColor struct {
	name   string
	color  ui.Color
	opaque bool
}

// semanticColors lists every color the semantic layer promises. A view may only
// draw colors from this set, so a missing or transparent one is a hole in the
// layer, not a cosmetic detail.
func semanticColors(k tokens) []namedColor {
	return []namedColor{
		{"background", k.background, true},
		{"foreground", k.foreground, true},
		{"sidebar", k.sidebar, true},
		{"sidebarForeground", k.sidebarForeground, true},
		{"sidebarPrimary", k.sidebarPrimary, true},
		{"sidebarPrimaryForeground", k.sidebarPrimaryForeground, true},
		{"sidebarAccent", k.sidebarAccent, true},
		{"sidebarAccentForeground", k.sidebarAccentForeground, true},
		{"sidebarHover", k.sidebarHover, true},
		{"sidebarBorder", k.sidebarBorder, true},
		{"sidebarRing", k.sidebarRing, false},
		{"card", k.card, true},
		{"cardForeground", k.cardForeground, true},
		{"popover", k.popover, true},
		{"popoverForeground", k.popoverForeground, true},
		{"primary", k.primary, true},
		{"onPrimary", k.onPrimary, true},
		{"secondary", k.secondary, true},
		{"secondaryForeground", k.secondaryForeground, true},
		{"muted", k.muted, true},
		{"mutedForeground", k.mutedForeground, true},
		{"accent", k.accent, true},
		{"accentForeground", k.accentForeground, true},
		{"danger", k.danger, true},
		{"onDanger", k.onDanger, true},
		{"border", k.border, true},
		{"input", k.input, true},
		{"ring", k.ring, false},
		{"success", k.success, true},
		{"warning", k.warning, true},
		{"tooltip", k.tooltip, true},
		{"tooltipForeground", k.tooltipForeground, true},
		{"tooltipMutedForeground", k.tooltipMutedForeground, true},
		{"tooltipBorder", k.tooltipBorder, true},
		{"tooltipChip", k.tooltipChip, true},
	}
}

// themes are the two appearances every token assertion runs against: the
// semantic layer must not hold for one and break in the other.
func themes() []struct {
	name  string
	theme *ui.Theme
} {
	return []struct {
		name  string
		theme *ui.Theme
	}{
		{"light", ui.LightTheme()},
		{"dark", ui.DarkTheme()},
	}
}

// TestSemanticTokensExist checks that every semantic token is present and, for
// a face, opaque in both appearances: a transparent face would draw nothing,
// which is exactly what the old single surface plus a border wash looked like.
func TestSemanticTokensExist(t *testing.T) {
	for _, tc := range themes() {
		k := newTokens(tc.theme)
		for _, token := range semanticColors(k) {
			if token.color.A == 0 {
				t.Errorf("%s: %s is transparent", tc.name, token.name)
				continue
			}
			if token.opaque && token.color.A != 255 {
				t.Errorf("%s: %s has alpha %d, want an opaque face", tc.name, token.name, token.color.A)
			}
		}
	}
}

// TestSemanticTokensComeFromTheTheme checks that the layer derives from the
// theme rather than carrying colors of its own, except the reference's brand
// green: the focus ring and the neutral faces are the theme's, so a theme swap
// reaches every view through the tokens.
func TestSemanticTokensComeFromTheTheme(t *testing.T) {
	for _, tc := range themes() {
		k := newTokens(tc.theme)
		cases := []struct {
			name  string
			token ui.Color
			want  ui.Color
		}{
			{"primary", k.primary, brandGreenLight},
			{"onPrimary", k.onPrimary, ui.Hex("#ffffff")},
			{"accent", k.accent, tc.theme.SurfaceHover},
			{"secondary", k.secondary, tc.theme.Surface},
			{"muted", k.muted, tc.theme.Surface},
			{"mutedForeground", k.mutedForeground, tc.theme.TextMuted},
			{"danger", k.danger, tc.theme.Danger},
			{"border", k.border, tc.theme.Border},
			{"ring", k.ring, tc.theme.Focus},
			{"foreground", k.foreground, tc.theme.Text},
			{"background", k.background, tc.theme.Background},
		}
		if tc.theme.Dark {
			cases[0] = struct {
				name  string
				token ui.Color
				want  ui.Color
			}{"primary", k.primary, brandGreenDark}
			cases[1] = struct {
				name  string
				token ui.Color
				want  ui.Color
			}{"onPrimary", k.onPrimary, ui.Hex("#07140a")}
		}
		for _, c := range cases {
			if c.token != c.want {
				t.Errorf("%s: %s = %+v, want the theme's %+v", tc.name, c.name, c.token, c.want)
			}
		}
	}
}

// TestAccentAndPrimaryDiffer checks the rename the layer makes: dim-electron's
// accent is a neutral hover face, while the brand color is primary. They were
// the same field before, so a view asking for a hover tint drew the brand fill.
func TestAccentAndPrimaryDiffer(t *testing.T) {
	for _, tc := range themes() {
		k := newTokens(tc.theme)
		if k.accent == k.primary {
			t.Errorf("%s: accent and primary are both %+v", tc.name, k.accent)
		}
		if k.accent == k.background {
			t.Errorf("%s: accent is the background, so a hover draws nothing", tc.name)
		}
		if k.accentForeground == k.accent {
			t.Errorf("%s: accentForeground equals its face", tc.name)
		}
	}
}

// TestSurfaceLadder checks the elevation the layer derives: a card lifts toward
// white off the background, the popover another step on top, and neither is a
// translucent border wash. The reference's light appearance keeps the card on
// the window's white, so equality is allowed there; a card must never sit darker
// than the window it covers.
func TestSurfaceLadder(t *testing.T) {
	for _, tc := range themes() {
		k := newTokens(tc.theme)
		base := luminance(k.background)

		if k.card.A != 255 {
			t.Errorf("%s: card is a wash, alpha %d", tc.name, k.card.A)
		}
		if wash := k.border.Alpha(washAlpha); k.card == wash {
			t.Errorf("%s: card is the border wash %+v", tc.name, wash)
		}

		cardRise := luminance(k.card) - base
		popoverRise := luminance(k.popover) - base
		if cardRise < 0 {
			t.Errorf("%s: card is darker than the background by %.3f", tc.name, -cardRise)
		}
		if popoverRise < cardRise {
			t.Errorf("%s: popover lifts %.3f, card %.3f: the ladder points down", tc.name, popoverRise, cardRise)
		}
		if tc.theme.Dark {
			if cardRise <= 0.01 {
				t.Errorf("%s: card is %.3f from the background, too close to read as a face", tc.name, cardRise)
			}
			if popoverRise <= cardRise {
				t.Errorf("%s: popover lifts %.3f, card %.3f: the ladder is not monotone", tc.name, popoverRise, cardRise)
			}
		}
	}
}

// TestTextOnFilledTokensIsReadable checks the paired *-on tokens: the text on
// the primary fill and on danger stays far enough from its face to read, in
// both appearances.
func TestTextOnFilledTokensIsReadable(t *testing.T) {
	for _, tc := range themes() {
		k := newTokens(tc.theme)
		for _, pair := range []struct {
			face, text ui.Color
			name       string
		}{
			{k.primary, k.onPrimary, "onPrimary"},
			{k.danger, k.onDanger, "onDanger"},
			{k.tooltip, k.tooltipForeground, "tooltipForeground"},
		} {
			if diff := math.Abs(luminance(pair.face) - luminance(pair.text)); diff < 0.3 {
				t.Errorf("%s: %s is %.2f from its face %+v", tc.name, pair.name, diff, pair.face)
			}
		}
	}
}

// TestTooltipIsTheInversePair checks that the tip stands out from the window:
// it is the theme turned over, and its secondary text and chips step toward it
// rather than away.
func TestTooltipIsTheInversePair(t *testing.T) {
	for _, tc := range themes() {
		k := newTokens(tc.theme)
		if k.tooltip == k.background {
			t.Errorf("%s: tooltip is the window's own face", tc.name)
		}
		muted := math.Abs(luminance(k.tooltipMutedForeground) - luminance(k.tooltip))
		text := math.Abs(luminance(k.tooltipForeground) - luminance(k.tooltip))
		if muted >= text {
			t.Errorf("%s: tooltip's secondary text is as far from the face as its text", tc.name)
		}
	}
}

// TestTokenDimensions pins the dimensionless part of the layer: the sidebar's
// width, the font-size steps, the radius ladder and the type scale. Changing
// one of these is a deliberate decision about the shell's density, so the
// numbers are checked here rather than discovered in a view.
func TestTokenDimensions(t *testing.T) {
	// Pin the reference's 14px root: mygo's platform default is 13 on macOS, and
	// the rem-derived ladder is stated at the reference's size.
	theme := ui.LightTheme()
	theme.FontSize = defaultUIFontSize
	k := newTokens(theme)

	if k.sidebarW != 256 || k.sidebarMinW != 200 || k.sidebarMaxW != 360 {
		t.Errorf("sidebar width is %.0f (%.0f–%.0f), want 256 (200–360)",
			k.sidebarW, k.sidebarMinW, k.sidebarMaxW)
	}

	wantSteps := []float32{12, 13, 14, 16, 18}
	if len(fontSizeSteps) != len(wantSteps) {
		t.Fatalf("font-size steps are %v, want %v", fontSizeSteps, wantSteps)
	}
	for i, want := range wantSteps {
		if fontSizeSteps[i] != want {
			t.Errorf("font-size step %d is %.0f, want %.0f", i, fontSizeSteps[i], want)
		}
	}

	radii := []struct {
		name string
		got  float32
		want float32
	}{
		{"radiusSM", radiusSM, 4.75},
		{"radiusMD", radiusMD, 6.75},
		{"radiusLG", radiusLG, 8.75},
		{"radiusXL", radiusXL, 12.75},
	}
	for _, r := range radii {
		if r.got != r.want {
			t.Errorf("%s is %.2f, want %.2f", r.name, r.got, r.want)
		}
	}
	if k.cardRadius < radiusMD || k.cardRadius > radiusXL {
		t.Errorf("card radius %.2f is off the ladder", k.cardRadius)
	}
	if k.codeFontSize != 13 {
		t.Errorf("code font size is %.0f, want 13", k.codeFontSize)
	}
	if k.uiFontSize != k.bodySize {
		t.Errorf("body text is %.0f on the %.0f step", k.bodySize, k.uiFontSize)
	}
	if !isFontStep(k.uiFontSize) {
		t.Errorf("body text sits on %.0f, which is not one of the steps", k.uiFontSize)
	}

	// The hero is the reference's 45:32 mark at h-14, a 22 DIP heading on a 28
	// DIP line box, a gap-4 between them and a p-8 inset.
	if math.Abs(float64(k.heroMarkH-heroMarkH)) > 1e-4 || math.Abs(float64(k.heroMarkW-heroMarkW)) > 1e-4 {
		t.Errorf("hero mark is %.2fx%.2f, want %.2f", k.heroMarkW, k.heroMarkH, heroMarkW)
	}
	if want := k.heroMarkH * 45.0 / 32.0; math.Abs(float64(k.heroMarkW-want)) > 1e-4 {
		t.Errorf("hero mark is %.2f wide for a %.2f high 45:32 shape, want %.2f", k.heroMarkW, k.heroMarkH, want)
	}
	if math.Abs(float64(k.heroTitleSize-22)) > 1e-4 || math.Abs(float64(k.heroTitleLineH-28)) > 1e-4 {
		t.Errorf("hero title is %.1f on a %.1f line box, want 22/28", k.heroTitleSize, k.heroTitleLineH)
	}
	if k.heroGap != 14 || k.heroPad != 28 {
		t.Errorf("hero gap/inset is %.1f/%.1f, want 14/28", k.heroGap, k.heroPad)
	}

	// The card and row insets are the reference's px-3 and px-4 at its 14px root,
	// not the theme's 4 DIP spacing unit.
	if k.cardPadX != 10.5 || k.rowPadX != 14 {
		t.Errorf("card/row inset is %.1f/%.1f, want 10.5/14", k.cardPadX, k.rowPadX)
	}
}

// isFontStep reports whether size is one of the steps the appearance settings
// offer.
func isFontStep(size float32) bool {
	for _, step := range fontSizeSteps {
		if step == size {
			return true
		}
	}
	return false
}

// TestFontSizeSnapsToAStep checks that a theme size between two steps lands on
// one of them, so every heading of the type scale moves together and the shell
// never draws text between the sizes the appearance settings offer.
func TestFontSizeSnapsToAStep(t *testing.T) {
	cases := []struct {
		theme, want float32
	}{
		{0, defaultUIFontSize},
		{11, 12},
		{13, 13},
		{15.9, 16},
		{40, 18},
	}
	for _, tc := range cases {
		if got := snapFontSize(tc.theme); got != tc.want {
			t.Errorf("snapFontSize(%.1f) = %.0f, want %.0f", tc.theme, got, tc.want)
		}
	}

	theme := ui.LightTheme()
	theme.FontSize = 17.5
	k := newTokens(theme)
	if k.bodySize != 18 {
		t.Errorf("body text on a 17.5 theme is %.0f, want the 18 step", k.bodySize)
	}
	if !(k.titleSize > k.bodySize && k.bodySize > k.labelSize && k.labelSize > k.smallSize) {
		t.Errorf("type scale is not ordered: %.1f/%.1f/%.1f/%.1f",
			k.titleSize, k.bodySize, k.labelSize, k.smallSize)
	}
}

// TestFontSizePreferenceSetsTheScale checks the font-size steps: each one the
// appearance surface offers moves the body and every step of the type scale to
// that size, with the scale keeping its order and growing from step to step.
func TestFontSizePreferenceSetsTheScale(t *testing.T) {
	var previous float32
	for i, step := range fontSizeSteps {
		k := newTokensFor(ui.LightTheme(), SettingsView{FontSize: step})
		if k.uiFontSize != step || k.bodySize != step {
			t.Errorf("a %.0f preference draws %.0f body text on the %.0f step", step, k.bodySize, k.uiFontSize)
		}
		if k.codeFontSize != defaultCodeFontSize {
			t.Errorf("a %.0f preference moved the code size to %.0f", step, k.codeFontSize)
		}
		if !(k.titleSize > k.bodySize && k.bodySize > k.labelSize && k.labelSize > k.smallSize) {
			t.Errorf("a %.0f preference breaks the scale order: %.1f/%.1f/%.1f/%.1f",
				step, k.titleSize, k.bodySize, k.labelSize, k.smallSize)
		}
		if i > 0 && k.titleSize <= previous {
			t.Errorf("step %.0f draws the title at %.1f, no larger than the step below's %.1f",
				step, k.titleSize, previous)
		}
		previous = k.titleSize
	}
}

// TestFontSizePreferenceOverridesTheTheme checks which size wins when the theme
// names one: the saved preference, which also lands on the frame's theme so
// mygo's own widgets and text take the same size.
func TestFontSizePreferenceOverridesTheTheme(t *testing.T) {
	theme := ui.LightTheme()
	theme.FontSize = 12

	k := newTokensFor(theme, SettingsView{FontSize: 18})
	if k.bodySize != 18 {
		t.Errorf("a saved 18 over a 12 theme draws body text at %.0f", k.bodySize)
	}
	if k.theme.FontSize != 18 {
		t.Errorf("the frame's theme has a %.0f type size, want the saved 18", k.theme.FontSize)
	}
	if theme.FontSize != 12 {
		t.Errorf("the window's own theme was changed to %.0f", theme.FontSize)
	}
}

// TestCompactDensityTightensTheRoom checks what compact changes: every gap and
// padding the shell derives from the theme's spacing unit and every literal
// layout step shrinks by densityCompactScale, in the tokens and for the widgets
// through the theme's spacing unit, while row heights, type sizes and colors
// keep their values.
func TestCompactDensityTightensTheRoom(t *testing.T) {
	regular := newTokensFor(ui.LightTheme(), SettingsView{})
	compact := newTokensFor(ui.LightTheme(), SettingsView{Density: DensityCompact})

	room := []struct {
		name           string
		regular, tight float32
	}{
		{"gap", regular.gap, compact.gap},
		{"gapWide", regular.gapWide, compact.gapWide},
		{"rowPadY", regular.rowPadY, compact.rowPadY},
		{"rowPadX", regular.rowPadX, compact.rowPadX},
		{"toolbarPadY", regular.toolbarPadY, compact.toolbarPadY},
		{"toolbarPadX", regular.toolbarPadX, compact.toolbarPadX},
		{"edgePad", regular.edgePad, compact.edgePad},
		{"panelPad", regular.panelPad, compact.panelPad},
		{"cardPadY", regular.cardPadY, compact.cardPadY},
		{"cardPadX", regular.cardPadX, compact.cardPadX},
		{"heroGap", regular.heroGap, compact.heroGap},
		{"heroPad", regular.heroPad, compact.heroPad},
		{"timelinePadX", regular.timelinePadX, compact.timelinePadX},
		{"timelineBlockGap", regular.timelineBlockGap, compact.timelineBlockGap},
		{"timelinePadStart", regular.timelinePadStart, compact.timelinePadStart},
		{"timelineUserPadY", regular.timelineUserPadY, compact.timelineUserPadY},
		{"timelineUserPadX", regular.timelineUserPadX, compact.timelineUserPadX},
		{"timelineFileRowY", regular.timelineFileRowY, compact.timelineFileRowY},
		{"timelineFileRowX", regular.timelineFileRowX, compact.timelineFileRowX},
		{"timelineFileGap", regular.timelineFileGap, compact.timelineFileGap},
		{"sidebarHeaderTop", regular.sidebarHeaderTop, compact.sidebarHeaderTop},
		{"sidebarGap", regular.sidebarGap, compact.sidebarGap},
		{"sidebarPadX", regular.sidebarPadX, compact.sidebarPadX},
		{"sidebarSectionPadX", regular.sidebarSectionPadX, compact.sidebarSectionPadX},
	}
	for _, tc := range room {
		if tc.tight >= tc.regular {
			t.Errorf("compact %s is %.2f, not tighter than the regular %.2f", tc.name, tc.tight, tc.regular)
		}
		if want := tc.regular * densityCompactScale; math.Abs(float64(tc.tight-want)) > 1e-4 {
			t.Errorf("compact %s is %.2f, want %.4f of the regular %.2f", tc.name, tc.tight, densityCompactScale, tc.regular)
		}
	}
	if got, want := compact.space(3), regular.space(3)*densityCompactScale; math.Abs(float64(got-want)) > 1e-4 {
		t.Errorf("compact space(3) is %.2f, want %.2f", got, want)
	}
	if want := regular.theme.Spacing * densityCompactScale; math.Abs(float64(compact.theme.Spacing-want)) > 1e-4 {
		t.Errorf("the compact theme's spacing unit is %.2f, want %.2f", compact.theme.Spacing, want)
	}

	// What compact must leave alone: the parts are the same size, only the room
	// between them is tighter.
	unchanged := []struct {
		name           string
		regular, tight float32
	}{
		{"sidebarHeaderH", regular.sidebarHeaderH, compact.sidebarHeaderH},
		{"sidebarSectionH", regular.sidebarSectionH, compact.sidebarSectionH},
		{"sidebarRowH", regular.sidebarRowH, compact.sidebarRowH},
		{"sidebarEntryH", regular.sidebarEntryH, compact.sidebarEntryH},
		{"sidebarCell", regular.sidebarCell, compact.sidebarCell},
		{"timelineHeaderH", regular.timelineHeaderH, compact.timelineHeaderH},
		{"timelinePanelMaxH", regular.timelinePanelMaxH, compact.timelinePanelMaxH},
		{"timelineUserWidth", regular.timelineUserWidth, compact.timelineUserWidth},
		{"uiFontSize", regular.uiFontSize, compact.uiFontSize},
		{"titleSize", regular.titleSize, compact.titleSize},
		{"labelSize", regular.labelSize, compact.labelSize},
		{"heroTitleSize", regular.heroTitleSize, compact.heroTitleSize},
		{"heroTitleLineH", regular.heroTitleLineH, compact.heroTitleLineH},
		{"heroMarkH", regular.heroMarkH, compact.heroMarkH},
		{"codeFontSize", regular.codeFontSize, compact.codeFontSize},
		{"sidebarW", regular.sidebarW, compact.sidebarW},
	}
	for _, tc := range unchanged {
		if tc.tight != tc.regular {
			t.Errorf("compact changed %s from %.2f to %.2f", tc.name, tc.regular, tc.tight)
		}
	}
	if compact.background != regular.background || compact.card != regular.card {
		t.Error("compact changed the colors")
	}
}

// TestCodeFontSizePreference checks the code size: the saved value is the
// token's codeFontSize, a zero takes the default, and a value outside the
// surface's range is held to it, exactly as the appearance surface draws it.
func TestCodeFontSizePreference(t *testing.T) {
	cases := []struct {
		saved, want float32
	}{
		{0, defaultCodeFontSize},
		{13, 13},
		{16, 16},
		{19.6, 20},
		{5, settingsCodeMin},
		{40, settingsCodeMax},
	}
	for _, tc := range cases {
		k := newTokensFor(ui.LightTheme(), SettingsView{CodeFontSize: tc.saved})
		if k.codeFontSize != tc.want {
			t.Errorf("a %.1f code size draws code at %.0f, want %.0f", tc.saved, k.codeFontSize, tc.want)
		}
		if wantBody := newTokens(ui.LightTheme()).bodySize; k.bodySize != wantBody {
			t.Errorf("a %.1f code size moved the body text to %.0f, want the %.0f step", tc.saved, k.bodySize, wantBody)
		}
	}
}

// TestFontFamilyPreferenceReachesTheTheme checks the family: the system stack
// is mygo's empty family, the monospace one is the platform's monospaced
// family, and neither moves the type scale.
func TestFontFamilyPreferenceReachesTheTheme(t *testing.T) {
	system := newTokensFor(ui.LightTheme(), SettingsView{})
	if system.theme.Font != "" {
		t.Errorf("the system family is %q, want mygo's platform interface font", system.theme.Font)
	}

	mono := newTokensFor(ui.LightTheme(), SettingsView{FontFamily: FontMonospace})
	if mono.theme.Font != monospaceFamily {
		t.Errorf("the monospace family is %q, want %q", mono.theme.Font, monospaceFamily)
	}
	if mono.uiFontSize != system.uiFontSize || mono.codeFontSize != system.codeFontSize {
		t.Errorf("the family moved the type sizes to %.0f/%.0f", mono.uiFontSize, mono.codeFontSize)
	}
}

// TestDefaultAppearanceKeepsTheThemeTokens checks the fallback: a window that
// saved no preference draws exactly the tokens its theme alone produces, so a
// first run after this change looks as it did before one could be saved.
func TestDefaultAppearanceKeepsTheThemeTokens(t *testing.T) {
	for _, tc := range themes() {
		theme := newTokens(tc.theme)
		saved := newTokensFor(tc.theme, SettingsView{})
		theme.theme, saved.theme = nil, nil
		if !reflect.DeepEqual(theme, saved) {
			t.Errorf("%s: the default appearance differs from the theme's tokens", tc.name)
		}
	}
}

// TestSidebarWidthClamps checks the bounds a resize is held to: a drag can
// neither collapse the session list nor take the room the conversation needs.
func TestSidebarWidthClamps(t *testing.T) {
	k := newTokens(ui.LightTheme())
	cases := []struct{ in, want float32 }{
		{0, sidebarMinW},
		{120, sidebarMinW},
		{sidebarMinW, sidebarMinW},
		{300, 300},
		{sidebarMaxW, sidebarMaxW},
		{900, sidebarMaxW},
	}
	for _, tc := range cases {
		if got := k.clampSidebarWidth(tc.in); got != tc.want {
			t.Errorf("clampSidebarWidth(%.0f) = %.0f, want %.0f", tc.in, got, tc.want)
		}
	}
}

// faceShare is how much of r is painted exactly face, as the share of the
// pixels sampled: an opaque face covers the padding and the gaps between lines,
// while a tint of the border or of the background covers none of it.
func faceShare(tt *ui.Tester, r ui.Rect, face ui.Color) float64 {
	img := tt.Image()
	total, hits := 0, 0
	for y := int(r.Y); y < int(r.Y+r.H); y += 2 {
		for x := int(r.X); x < int(r.X+r.W); x += 2 {
			if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
				continue
			}
			total++
			pixel := img.RGBAAt(x, y)
			if pixel.R == face.R && pixel.G == face.G && pixel.B == face.B && pixel.A == face.A {
				hits++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

// TestShellRendersDark checks that the whole shell builds in the dark
// appearance: the tokens are derived from the window's theme each frame, so a
// dark window must not lose a row or a card to a color that only works on
// white.
func TestShellRendersDark(t *testing.T) {
	s := fixture()
	s.Messages = append(s.Messages, Message{
		ID:     "t1",
		Author: AuthorTool,
		Tool:   Tool{Name: "bash", Params: "go test ./...", Result: "ok", Status: ToolRunning},
	})
	tt := ui.NewTester(s.View, 900, 600)
	tt.SetDark(true)

	for _, want := range []string{"First session", "hello from the fixture", "bash", "Running", "Send"} {
		if !tt.HasText(want) {
			t.Errorf("dark shell is missing %q, texts %q", want, tt.Texts())
		}
	}
}
