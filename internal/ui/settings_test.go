package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// settingsHarness drives the settings surface on its own: each test builds the
// region and folds the one change it reports back into the view, as the window
// will. The surface's section selection is its own frame-to-frame state, so it
// is not part of the view the harness folds.
type settingsHarness struct {
	view         SettingsView
	capabilities []wipCapability
	changes      []SettingsChange
	tt           *ui.Tester
}

// newSettingsHarness draws view and records every change the surface reports.
func newSettingsHarness(view SettingsView) *settingsHarness {
	return newSettingsHarnessAt(view, 900, 900)
}

// newSettingsHarnessAt is newSettingsHarness at a chosen window size, for the
// narrow-window cases.
func newSettingsHarnessAt(view SettingsView, width, height int) *settingsHarness {
	h := &settingsHarness{view: view}
	h.tt = ui.NewTester(func(c *ui.Context) {
		renderSettings(c, newTokens(c.Theme()), h.view, h.capabilities, func(change SettingsChange) {
			h.changes = append(h.changes, change)
			h.view = applySettingsChange(h.view, change)
		})
	}, width, height)
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

	if v.Section != SettingsSectionAppearance {
		t.Errorf("default section is %v, want the appearance page", v.Section)
	}
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
	if info, ok := h.tt.AccessByRole(ui.RoleListItem, "Appearance"); !ok || !info.Selected {
		t.Errorf("the default section's navigation row is not selected: %+v, found %v", info, ok)
	}
}

// TestSettingsSectionsOrder checks the left column against the reference: all
// 17 sections, in the reference's order, from Account to External Agent, with
// the appearance page open and marked as the selected row.
func TestSettingsSectionsOrder(t *testing.T) {
	want := []string{
		"Account", "Usage", "Personalization", "Appearance", "Unavailable capabilities", "General",
		"Shortcuts", "Models", "Multimodal", "Runtime", "MCP",
		"Notifications", "Pet", "Permissions", "Git", "Hooks", "Plugins",
		"External Agent",
	}
	if len(settingsSections) != len(want) {
		t.Fatalf("the navigation has %d sections, want %d", len(settingsSections), len(want))
	}
	for i, name := range want {
		if got := settingsSectionName(settingsSections[i]); got != name {
			t.Fatalf("section %d is %q, want %q", i+1, got, name)
		}
	}

	h := newSettingsHarness(defaultSettingsView(newTokens(ui.LightTheme())))
	if _, ok := h.tt.AccessByRole(ui.RoleNavigation, settingsNavigationLabel); !ok {
		t.Fatalf("the section column is not exposed as a navigation, texts %q", h.tt.Texts())
	}
	lastY := float32(-1)
	for i, name := range want {
		info, ok := h.tt.AccessByRole(ui.RoleListItem, name)
		if !ok {
			t.Fatalf("navigation row %d (%q) is missing, texts %q", i+1, name, h.tt.Texts())
		}
		if info.Bounds.Y <= lastY {
			t.Errorf("row %d (%q) sits at y=%.1f, not below the row before it at %.1f", i+1, name, info.Bounds.Y, lastY)
		}
		lastY = info.Bounds.Y
		if got := info.Selected; got != (name == "Appearance") {
			t.Errorf("row %q is selected=%v, want only Appearance selected", name, got)
		}
	}
}

// TestSettingsNavigationSwitchesSection checks a click on a navigation row:
// it changes only the surface's chosen section, reports no preference change,
// and the page below switches from the appearance controls to the WIP
// explanation and back.
func TestSettingsNavigationSwitchesSection(t *testing.T) {
	h := newSettingsHarness(defaultSettingsView(newTokens(ui.LightTheme())))
	before := h.view

	if err := h.tt.Click("General"); err != nil {
		t.Fatalf("click General: %v", err)
	}
	if len(h.changes) != 0 {
		t.Fatalf("a navigation click reported %+v, want no preference change", h.changes)
	}
	if h.view != before {
		t.Errorf("a navigation click changed the view to %+v, want %+v", h.view, before)
	}
	if info, ok := h.tt.AccessByRole(ui.RoleListItem, "General"); !ok || !info.Selected {
		t.Errorf("the General row is not selected after its click: %+v, found %v", info, ok)
	}
	if info, ok := h.tt.AccessByRole(ui.RoleListItem, "Appearance"); !ok || info.Selected {
		t.Errorf("the Appearance row stayed selected after choosing General: %+v, found %v", info, ok)
	}
	if !h.tt.HasText(settingsSectionGaps[SettingsSectionGeneral]) {
		t.Errorf("the General page does not explain its gap, texts %q", h.tt.Texts())
	}
	if h.tt.HasText(tr(LocaleEN, keySettingsFontSize)) {
		t.Errorf("the General page still draws the appearance font-size row, texts %q", h.tt.Texts())
	}

	if err := h.tt.Click("Appearance"); err != nil {
		t.Fatalf("click Appearance: %v", err)
	}
	if !h.tt.HasText(tr(LocaleEN, keySettingsFontSize)) {
		t.Fatalf("the appearance page did not come back, texts %q", h.tt.Texts())
	}

	// The controls still work after the round trip: one choice, one change.
	if err := h.tt.Click("Dark"); err != nil {
		t.Fatalf("click Dark: %v", err)
	}
	if len(h.changes) != 1 || h.changes[0].Field != SettingsFieldTheme || h.changes[0].Theme != ThemeDark {
		t.Fatalf("click Dark reported %+v, want one theme change to Dark", h.changes)
	}
}

// TestSettingsWIPPagesOfferNoControls checks every section without a page: it
// keeps its navigation row and title, shows the WIP badge and the missing
// client or page, and draws no control, current value or fake data that could
// look like a saved preference.
func TestSettingsWIPPagesOfferNoControls(t *testing.T) {
	for _, section := range settingsSections {
		if section == SettingsSectionAppearance || section == SettingsSectionCapabilities {
			continue
		}
		name := settingsSectionName(section)
		t.Run(name, func(t *testing.T) {
			h := newSettingsHarness(SettingsView{Section: section})
			if info, ok := h.tt.AccessByRole(ui.RoleHeading, name); !ok || info.Level != 1 {
				t.Fatalf("the %s page has no level-1 heading %q: %+v, found %v", name, name, info, ok)
			}
			gap := settingsSectionGaps[section]
			if gap == "" {
				t.Fatalf("the %s page has no gap explanation", name)
			}
			if !h.tt.HasText(gap) {
				t.Fatalf("the %s page does not show its gap %q, texts %q", name, gap, h.tt.Texts())
			}
			if !strings.Contains(gap, "Unavailable") || !strings.Contains(gap, "client") || !strings.Contains(gap, "page") {
				t.Errorf("the %s gap %q does not name the missing client or page", name, gap)
			}
			if !h.tt.HasText(wipBadge) {
				t.Errorf("the %s page shows no visible %q badge, texts %q", name, wipBadge, h.tt.Texts())
			}
			if info, ok := h.tt.AccessByRole(ui.RoleRadio, ""); ok {
				t.Errorf("the %s page offers a radio control: %+v", name, info)
			}
			if info, ok := h.tt.AccessByRole(ui.RoleButton, ""); ok {
				t.Errorf("the %s page offers a button: %+v", name, info)
			}
			if h.tt.HasText(tr(LocaleEN, keySettingsCurrentPrefix)) {
				t.Errorf("the %s page draws a current-value row, texts %q", name, h.tt.Texts())
			}
			if len(h.changes) != 0 {
				t.Errorf("the %s page reported %+v, want no change", name, h.changes)
			}
		})
	}
}

// TestSettingsNarrowNavigationFits checks the section column in a narrow
// window: it shrinks with the room instead of pushing the page off the right
// edge, and every row stays inside the column.
func TestSettingsNarrowNavigationFits(t *testing.T) {
	const width, height = 320, 640
	h := newSettingsHarnessAt(defaultSettingsView(newTokens(ui.LightTheme())), width, height)

	nav, ok := h.tt.AccessByRole(ui.RoleNavigation, settingsNavigationLabel)
	if !ok {
		t.Fatalf("the section column is not exposed, texts %q", h.tt.Texts())
	}
	if nav.Bounds.X < 0 || nav.Bounds.X+nav.Bounds.W > width+0.5 || nav.Bounds.W <= 0 {
		t.Fatalf("the section column %+v does not fit the %d-wide window", nav.Bounds, width)
	}
	for _, section := range settingsSections {
		name := settingsSectionName(section)
		info, ok := h.tt.AccessByRole(ui.RoleListItem, name)
		if !ok {
			t.Fatalf("the %s row is missing in a narrow window, texts %q", name, h.tt.Texts())
		}
		if info.Bounds.X < nav.Bounds.X-0.5 || info.Bounds.X+info.Bounds.W > nav.Bounds.X+nav.Bounds.W+0.5 {
			t.Errorf("the %s row %+v spills outside the column %+v", name, info.Bounds, nav.Bounds)
		}
	}
	heading, ok := h.tt.AccessByRole(ui.RoleHeading, "Appearance")
	if !ok {
		t.Fatalf("the appearance page heading is missing, texts %q", h.tt.Texts())
	}
	if heading.Bounds.X+heading.Bounds.W > width+0.5 {
		t.Errorf("the page heading %+v spills past the %d-wide window", heading.Bounds, width)
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

// TestSettingsNavigationWorksInShell checks the navigation inside the shell's
// own frame, not just the surface's harness: opening settings starts on the
// appearance page, a click on another section replaces the page, and the
// selection survives the next frame.
func TestSettingsNavigationWorksInShell(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(tr(LocaleEN, keySettingsFontSize)) {
		t.Fatalf("the settings surface did not open on the appearance page, texts %q", tt.Texts())
	}
	if err := tt.Click("General"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(settingsSectionGaps[SettingsSectionGeneral]) {
		t.Fatalf("the General page is not drawn, texts %q", tt.Texts())
	}
	tt.Frame()
	if !tt.HasText(settingsSectionGaps[SettingsSectionGeneral]) {
		t.Fatalf("the General page did not survive the next frame, texts %q", tt.Texts())
	}
	if err := tt.Click("Appearance"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(tr(LocaleEN, keySettingsFontSize)) {
		t.Fatalf("the appearance page did not come back in the shell, texts %q", tt.Texts())
	}
}

// TestSettingsKeyboardReachable checks the Tab order: the section navigation
// comes first, in the reference's order, then every control of the appearance
// page in reading order, including the unavailable skin entries whose reasons
// a screen-reader user must still find.
func TestSettingsKeyboardReachable(t *testing.T) {
	h := newSettingsHarness(defaultSettingsView(newTokens(ui.LightTheme())))

	want := []string{
		"Account", "Usage", "Personalization", "Appearance", "Unavailable capabilities", "General",
		"Shortcuts", "Models", "Multimodal", "Runtime", "MCP",
		"Notifications", "Pet", "Permissions", "Git", "Hooks", "Plugins",
		"External Agent",
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

// TestSettingsCapabilitiesPage checks the page the sidebar's cards moved to: the
// settings navigation carries "Unavailable capabilities", the page lists every
// gap the shell has, and the sidebar itself draws none of them.
func TestSettingsCapabilitiesPage(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 900)

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Unavailable capabilities"); ok {
		t.Fatalf("the sidebar still draws the capability cards, texts %q", tt.Texts())
	}
	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Unavailable capabilities"); err != nil {
		t.Fatal(err)
	}
	if info, ok := tt.AccessByRole(ui.RoleHeading, "Unavailable capabilities"); !ok || info.Level != 1 {
		t.Fatalf("the capabilities page has no level-1 heading: %+v, found %v", info, ok)
	}
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Unavailable capabilities"); !ok {
		t.Fatalf("the capabilities page does not group its cards, texts %q", tt.Texts())
	}
	for _, capability := range wipCapabilities(s) {
		for _, want := range []string{capability.action, capability.method, capability.gap} {
			if !tt.HasText(want) {
				t.Errorf("the capabilities page is missing %q, texts %q", want, tt.Texts())
			}
		}
	}
	if !tt.HasText(settingsCapabilitiesNote) {
		t.Errorf("the capabilities page lost its note, texts %q", tt.Texts())
	}
}
