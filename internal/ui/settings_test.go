package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// settingsHarness drives the appearance surface on its own: renderSettings is
// not wired into the shell yet, so each test builds the region and folds the
// one change it reports back into the view, as the window will.
type settingsHarness struct {
	view    SettingsView
	changes []SettingsChange
	tt      *ui.Tester
}

// newSettingsHarness draws view and records every change the surface reports.
func newSettingsHarness(view SettingsView) *settingsHarness {
	h := &settingsHarness{view: view}
	h.tt = ui.NewTester(func(c *ui.Context) {
		renderSettings(c, newTokens(c.Theme()), h.view, func(change SettingsChange) {
			h.changes = append(h.changes, change)
			h.view = applySettingsChange(h.view, change)
		})
	}, 900, 900)
	return h
}

// applySettingsChange folds one reported change into the view, as the window's
// preferences would.
func applySettingsChange(v SettingsView, change SettingsChange) SettingsView {
	switch change.Field {
	case SettingsFieldTheme:
		v.Theme = change.Theme
	case SettingsFieldFontSize:
		v.FontSize = change.FontSize
	case SettingsFieldFontFamily:
		v.FontFamily = change.FontFamily
	case SettingsFieldSidebarWidth:
		v.SidebarWidth = change.SidebarWidth
	case SettingsFieldDensity:
		v.Density = change.Density
	case SettingsFieldCodeFontSize:
		v.CodeFontSize = change.CodeFontSize
	}
	return v
}

// TestSettingsDefaultsMatchTokens checks the appearance a window that has not
// saved a preference draws: the theme follows the desktop, the type sits on the
// tokens' own default steps, the sidebar is its default width, and the density
// is regular. The surface must draw those values, not empty rows.
func TestSettingsDefaultsMatchTokens(t *testing.T) {
	k := newTokens(ui.LightTheme())
	v := defaultSettingsView(k)

	if v.Theme != ThemeSystem {
		t.Errorf("default theme is %v, want the desktop's (System)", v.Theme)
	}
	if v.FontSize != k.uiFontSize {
		t.Errorf("default UI font size is %.0f, want the token's %.0f", v.FontSize, k.uiFontSize)
	}
	if v.CodeFontSize != k.codeFontSize {
		t.Errorf("default code font size is %.0f, want the token's %.0f", v.CodeFontSize, k.codeFontSize)
	}
	if v.FontFamily != FontSystem {
		t.Errorf("default font family is %v, want the system font", v.FontFamily)
	}
	if v.SidebarWidth != k.sidebarW {
		t.Errorf("default sidebar width is %.0f, want the token's %.0f", v.SidebarWidth, k.sidebarW)
	}
	if v.Density != DensityRegular {
		t.Errorf("default density is %v, want regular", v.Density)
	}

	h := newSettingsHarness(SettingsView{})
	for _, want := range []string{
		"Appearance",
		settingsLocalNote,
		"Color", "Theme", "Current: System",
		"Text", "Font size", "Current: " + settingsPx(k.uiFontSize),
		"Font family", "Current: System font",
		"Code font size", "Current: " + settingsPx(k.codeFontSize),
		"Layout", "Sidebar width", "Current: " + settingsPx(k.sidebarW),
		"Density", "Current: Regular",
		"Skin and chrome", settingsSkinNote,
	} {
		if !h.tt.HasText(want) {
			t.Errorf("the default surface is missing %q, texts %q", want, h.tt.Texts())
		}
	}
	if info, ok := h.tt.AccessByRole(ui.RoleRadio, "System"); !ok || !info.Checked {
		t.Errorf("the default theme's System choice is not selected: %+v, found %v", info, ok)
	}
	if info, ok := h.tt.AccessByRole(ui.RoleRadio, strconv.Itoa(int(k.uiFontSize))); !ok || !info.Checked {
		t.Errorf("the default font-size choice is not selected: %+v, found %v", info, ok)
	}
}

// TestSettingsReportsOneChangePerChoice checks each control: one click reports
// exactly one change, with the field and value the user chose. A redraw does
// not report the choice again, and the change carries the resolved view, so the
// window can store either the one value or the whole snapshot.
func TestSettingsReportsOneChangePerChoice(t *testing.T) {
	h := newSettingsHarness(defaultSettingsView(newTokens(ui.LightTheme())))

	cases := []struct {
		label string
		field SettingsField
		check func(SettingsChange) bool
	}{
		{"Dark", SettingsFieldTheme, func(c SettingsChange) bool { return c.Theme == ThemeDark }},
		{"Monospace", SettingsFieldFontFamily, func(c SettingsChange) bool { return c.FontFamily == FontMonospace }},
		{"Compact", SettingsFieldDensity, func(c SettingsChange) bool { return c.Density == DensityCompact }},
		{"18", SettingsFieldFontSize, func(c SettingsChange) bool { return c.FontSize == 18 }},
		{"Larger code", SettingsFieldCodeFontSize, func(c SettingsChange) bool { return c.CodeFontSize == defaultCodeFontSize+settingsCodeStep }},
		{"Narrower sidebar", SettingsFieldSidebarWidth, func(c SettingsChange) bool {
			return c.SidebarWidth == sidebarDefaultW-settingsWidthStep
		}},
	}
	for _, tc := range cases {
		before := len(h.changes)
		if err := h.tt.Click(tc.label); err != nil {
			t.Fatalf("click %q: %v", tc.label, err)
		}
		if got := len(h.changes); got != before+1 {
			t.Fatalf("click %q reported %d changes, want exactly 1", tc.label, got-before)
		}
		change := h.changes[len(h.changes)-1]
		if change.Field != tc.field {
			t.Errorf("click %q reported field %v, want %v", tc.label, change.Field, tc.field)
		}
		if !tc.check(change) {
			t.Errorf("click %q reported %+v", tc.label, change)
		}
		// The callback carries the whole resolved view, and applying it leaves
		// the other preferences alone.
		if change.Theme != h.view.Theme || change.FontSize != h.view.FontSize ||
			change.FontFamily != h.view.FontFamily || change.SidebarWidth != h.view.SidebarWidth ||
			change.Density != h.view.Density || change.CodeFontSize != h.view.CodeFontSize {
			t.Errorf("click %q reported %+v, which does not match the applied view %+v", tc.label, change, h.view)
		}

		h.tt.Frame()
		if got := len(h.changes); got != before+1 {
			t.Errorf("click %q reported %d changes after a redraw, want 1", tc.label, got-before)
		}
	}
}

// TestSettingsFontSizeStepsCycle checks the UI font-size control as one radio
// group: the arrows step through the offered sizes and wrap at either end, and
// each step reports its own value once. The surface never lands between steps.
func TestSettingsFontSizeStepsCycle(t *testing.T) {
	h := newSettingsHarness(defaultSettingsView(newTokens(ui.LightTheme())))
	h.view.FontSize = fontSizeSteps[len(fontSizeSteps)-1]
	h.tt.Frame()

	// Put the keyboard focus on the group's chosen step, then step with the
	// arrows, as a keyboard user does.
	if err := h.tt.Click(strconv.Itoa(int(h.view.FontSize))); err != nil {
		t.Fatal(err)
	}
	if len(h.changes) != 0 {
		t.Fatalf("clicking the chosen step reported %d changes, want 0", len(h.changes))
	}

	steps := []struct {
		key  ui.Key
		want float32
	}{
		{ui.KeyRight, fontSizeSteps[0]},                   // wraps past the largest
		{ui.KeyRight, fontSizeSteps[1]},                   // the next step
		{ui.KeyLeft, fontSizeSteps[0]},                    // back one
		{ui.KeyLeft, fontSizeSteps[len(fontSizeSteps)-1]}, // wraps past the smallest
	}
	for i, step := range steps {
		before := len(h.changes)
		h.tt.Key(0, step.key)
		if got := len(h.changes); got != before+1 {
			t.Fatalf("step %d (%v) reported %d changes, want exactly 1", i+1, step.key, got-before)
		}
		change := h.changes[len(h.changes)-1]
		if change.Field != SettingsFieldFontSize || change.FontSize != step.want {
			t.Errorf("step %d (%v) reported %+v, want font size %.0f", i+1, step.key, change, step.want)
		}
		h.tt.Frame()
	}
}

// TestSettingsSidebarWidthStaysInBounds checks the width control: a saved
// width outside 200–360 is drawn at the nearest bound, the button past that
// bound is unavailable, and every step the surface reports stays inside it
// even when a press would pass an edge.
func TestSettingsSidebarWidthStaysInBounds(t *testing.T) {
	t.Run("above the widest", func(t *testing.T) {
		h := newSettingsHarness(SettingsView{SidebarWidth: 999})
		if !h.tt.HasText("Current: " + settingsPx(sidebarMaxW)) {
			t.Fatalf("an out-of-range width is not drawn at %d, texts %q", sidebarMaxW, h.tt.Texts())
		}
		if info, ok := h.tt.AccessByRole(ui.RoleButton, "Wider sidebar"); !ok || !info.Disabled {
			t.Errorf("Wider sidebar is available at the widest width: %+v, found %v", info, ok)
		}
		if err := h.tt.Click("Wider sidebar"); err != nil {
			t.Fatal(err)
		}
		if len(h.changes) != 0 {
			t.Fatalf("a press past the widest reported %d changes, want 0", len(h.changes))
		}
		if err := h.tt.Click("Narrower sidebar"); err != nil {
			t.Fatal(err)
		}
		if len(h.changes) != 1 {
			t.Fatalf("a press inside the range reported %d changes, want 1", len(h.changes))
		}
		if got := h.changes[0].SidebarWidth; got != sidebarMaxW-settingsWidthStep {
			t.Errorf("narrowing from the widest reported %.0f, want %.0f", got, float32(sidebarMaxW-settingsWidthStep))
		}
	})

	t.Run("below the narrowest", func(t *testing.T) {
		h := newSettingsHarness(SettingsView{SidebarWidth: 120})
		if !h.tt.HasText("Current: " + settingsPx(sidebarMinW)) {
			t.Fatalf("an out-of-range width is not drawn at %d, texts %q", sidebarMinW, h.tt.Texts())
		}
		if info, ok := h.tt.AccessByRole(ui.RoleButton, "Narrower sidebar"); !ok || !info.Disabled {
			t.Errorf("Narrower sidebar is available at the narrowest width: %+v, found %v", info, ok)
		}
		if err := h.tt.Click("Narrower sidebar"); err != nil {
			t.Fatal(err)
		}
		if len(h.changes) != 0 {
			t.Fatalf("a press past the narrowest reported %d changes, want 0", len(h.changes))
		}
		if err := h.tt.Click("Wider sidebar"); err != nil {
			t.Fatal(err)
		}
		if len(h.changes) != 1 {
			t.Fatalf("a press inside the range reported %d changes, want 1", len(h.changes))
		}
		if got := h.changes[0].SidebarWidth; got != sidebarMinW+settingsWidthStep {
			t.Errorf("widening from the narrowest reported %.0f, want %.0f", got, float32(sidebarMinW+settingsWidthStep))
		}
	})

	t.Run("a step past the edge", func(t *testing.T) {
		h := newSettingsHarness(SettingsView{SidebarWidth: sidebarMaxW - 5})
		if err := h.tt.Click("Wider sidebar"); err != nil {
			t.Fatal(err)
		}
		if len(h.changes) != 1 {
			t.Fatalf("a press inside the range reported %d changes, want 1", len(h.changes))
		}
		if got := h.changes[0].SidebarWidth; got != sidebarMaxW {
			t.Errorf("a step past the widest reported %.0f, want it converged at %d", got, sidebarMaxW)
		}
	})
}

// TestSettingsWIPEntriesDoNothing checks the skin and chrome entries: they stay
// visible, are marked unavailable, explain the real gap, and a click changes no
// preference. The surface offers no fake skin or third color theme.
func TestSettingsWIPEntriesDoNothing(t *testing.T) {
	h := newSettingsHarness(defaultSettingsView(newTokens(ui.LightTheme())))

	for _, tc := range []struct{ label, gap string }{
		{"Skin", "skin"},
		{"Chrome theme", "chrome"},
	} {
		info, ok := h.tt.AccessByRole(ui.RoleButton, tc.label)
		if !ok {
			t.Errorf("%q is not exposed as an entry, texts %q", tc.label, h.tt.Texts())
			continue
		}
		if !info.Disabled {
			t.Errorf("%q is available although the shell cannot draw a skin", tc.label)
		}
		if !info.Focusable {
			t.Errorf("%q is unreachable, so its reason is never read", tc.label)
		}
		if !strings.Contains(info.Description, "Unavailable") {
			t.Errorf("%q description %q does not say it is unavailable", tc.label, info.Description)
		}
		if !strings.Contains(strings.ToLower(info.Description), tc.gap) {
			t.Errorf("%q description %q does not name %q", tc.label, info.Description, tc.gap)
		}
		if err := h.tt.Click(tc.label); err != nil {
			t.Fatalf("click %q: %v", tc.label, err)
		}
	}
	if len(h.changes) != 0 {
		t.Errorf("a WIP click reported %+v, want no change", h.changes)
	}
	if !h.tt.HasText(wipBadge) {
		t.Errorf("the skin entries show no visible %q badge, texts %q", wipBadge, h.tt.Texts())
	}
	for _, fake := range []string{"Windows 98", "Windows XP", "Windows Vista", "macOS Big Sur", "OSX Tiger"} {
		if h.tt.HasText(fake) {
			t.Errorf("the surface offers the fake chrome theme %q", fake)
		}
	}
}

// TestSettingsKeyboardReachable checks the Tab order: every control the surface
// offers is reachable from the keyboard, in reading order, including the
// unavailable skin entries whose reasons a screen-reader user must still find.
func TestSettingsKeyboardReachable(t *testing.T) {
	h := newSettingsHarness(defaultSettingsView(newTokens(ui.LightTheme())))

	want := []string{
		ThemeSystem.String(),
		strconv.Itoa(int(h.view.FontSize)),
		FontSystem.String(),
		"Smaller code",
		"Larger code",
		"Narrower sidebar",
		"Wider sidebar",
		DensityRegular.String(),
		"Skin",
		"Chrome theme",
	}
	for i, name := range want {
		h.tt.Key(0, ui.KeyTab)
		if !h.tt.Focused(name) {
			t.Fatalf("Tab %d did not reach %q, texts %q", i+1, name, h.tt.Texts())
		}
	}
}
