package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// settingsHarness drives the settings route on its own: each test builds the
// route and folds the one change it reports back into the view, as the window
// will. The route's section selection is its own frame-to-frame state, so it is
// not part of the view the harness folds; backs counts the Back row and the
// Escape key, which both call the same callback.
type settingsHarness struct {
	view         SettingsView
	capabilities []wipCapability
	changes      []SettingsChange
	backs        int
	tt           *ui.Tester
}

// newSettingsHarness draws view and records every change the route reports.
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
		}, func() {
			h.backs++
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
	case SettingsFieldLocale:
		v.Locale = change.Locale
	}
	return v
}

// chooseOption opens a select by its accessible name and chooses one option. It
// clicks the select trigger's bounds rather than the row's label text, since
// both are found by the same name; callers choose an option other than the
// current one, as a reader would.
func chooseOption(t *testing.T, tt *ui.Tester, control, option string) {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RolePopUpButton, control)
	if !ok {
		t.Fatalf("the %q select is missing, texts %q", control, tt.Texts())
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	if err := tt.Click(option); err != nil {
		t.Fatalf("choose %q in %q: %v", option, control, err)
	}
}

// clickAccess clicks the center of the first element with the role and
// accessible name. It reaches a control whose visible row label shares its name,
// as the code-size slider's group heading does.
func clickAccess(t *testing.T, tt *ui.Tester, role ui.Role, name string) {
	t.Helper()
	info, ok := tt.AccessByRole(role, name)
	if !ok {
		t.Fatalf("the %q %v control is missing, texts %q", name, role, tt.Texts())
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
}

// TestSettingsDefaultsMatchTokens checks the appearance a window that has not
// saved a preference draws: the theme follows the desktop, the type sits on the
// tokens' own default steps, the sidebar is its default width, and the density
// is regular. The page must draw those values, not empty controls.
func TestSettingsDefaultsMatchTokens(t *testing.T) {
	k := newTokens(ui.LightTheme())
	v := defaultSettingsView(k)

	if v.Section != SettingsSectionPersonalization {
		t.Errorf("default section is %v, want the personalization page", v.Section)
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
}

// TestSettingsOpensOnPersonalization checks the route's opening page: the
// reference opens settings on Personalization, so a view that names no section
// draws that page, its navigation row is the selected one, and the appearance
// controls stay on the appearance page behind their own navigation row.
func TestSettingsOpensOnPersonalization(t *testing.T) {
	h := newSettingsHarness(SettingsView{})

	if info, ok := h.tt.AccessByRole(ui.RoleHeading, "Personalization"); !ok || info.Level != 1 {
		t.Errorf("the opening page has no level-1 Personalization heading: %+v, found %v", info, ok)
	}
	if !h.tt.HasText(settingsSectionGaps[SettingsSectionPersonalization]) {
		t.Errorf("the personalization page does not explain its gap, texts %q", h.tt.Texts())
	}
	if !h.tt.HasText(wipBadge) {
		t.Errorf("the personalization page shows no visible %q badge, texts %q", wipBadge, h.tt.Texts())
	}
	if info, ok := h.tt.AccessByRole(ui.RoleListItem, "Personalization"); !ok || !info.Selected {
		t.Errorf("the opening section's navigation row is not selected: %+v, found %v", info, ok)
	}
	if info, ok := h.tt.AccessByRole(ui.RoleListItem, "Appearance"); !ok || info.Selected {
		t.Errorf("the Appearance row is selected although the route opened on Personalization: %+v, found %v", info, ok)
	}
	if _, ok := h.tt.AccessByRole(ui.RoleRadioGroup, "Theme"); ok {
		t.Errorf("the opening page draws the appearance theme cards, texts %q", h.tt.Texts())
	}
	for _, want := range []string{"Interface language", "UI font", "Code size"} {
		if h.tt.HasText(want) {
			t.Errorf("the opening page draws the appearance control %q, texts %q", want, h.tt.Texts())
		}
	}
}

// TestSettingsAppearancePageDrawsDefaults checks the appearance page a reader
// reaches from the navigation: it draws the resolved preference values, not
// empty controls, and keeps its own navigation row selected.
func TestSettingsAppearancePageDrawsDefaults(t *testing.T) {
	k := newTokens(ui.LightTheme())
	h := newSettingsHarness(SettingsView{Section: SettingsSectionAppearance})
	for _, want := range []string{
		"Appearance",
		settingsLocalNote,
		"Theme", "System", "Light", "Dark",
		"Language", "Interface language", "English",
		"Font", "UI font", "System font", "Code font", "Follows the UI font",
		"UI size",
		"Code size",
		"Code theme", "Same as interface",
		"My skin", settingsSkinNote, "Skin 1", "Skin 6",
		wipBadge,
	} {
		if !h.tt.HasText(want) {
			t.Errorf("the appearance page is missing %q, texts %q", want, h.tt.Texts())
		}
	}
	if h.tt.HasText("Current:") {
		t.Errorf("the page still draws a Current: value, texts %q", h.tt.Texts())
	}
	if h.tt.HasText("Density") {
		t.Errorf("the page still draws the density control, texts %q", h.tt.Texts())
	}
	if info, ok := h.tt.AccessByRole(ui.RoleHeading, "Appearance"); !ok || info.Level != 1 {
		t.Errorf("the page has no level-1 Appearance heading: %+v, found %v", info, ok)
	}
	if _, ok := h.tt.AccessByRole(ui.RoleRadioGroup, "Theme"); !ok {
		t.Errorf("the theme cards are not a radio group, texts %q", h.tt.Texts())
	}
	for _, name := range []string{"System", "Light", "Dark"} {
		info, ok := h.tt.AccessByRole(ui.RoleRadio, name)
		if !ok {
			t.Errorf("the %s theme card is not a radio, texts %q", name, h.tt.Texts())
			continue
		}
		if info.Checked != (name == "System") {
			t.Errorf("the %s theme card is checked=%v, want only System checked", name, info.Checked)
		}
	}
	if info, ok := h.tt.AccessByRole(ui.RoleRadio, "S"); !ok || !info.Checked {
		t.Errorf("the default UI size's S segment is not selected: %+v, found %v", info, ok)
	}
	for _, tc := range []struct{ label, value string }{
		{"Interface language", "English"},
		{"UI font", "System font"},
	} {
		info, ok := h.tt.AccessByRole(ui.RolePopUpButton, tc.label)
		if !ok {
			t.Errorf("the %s select is missing, texts %q", tc.label, h.tt.Texts())
			continue
		}
		if info.Value != tc.value {
			t.Errorf("the %s select shows %q, want %q", tc.label, info.Value, tc.value)
		}
		if info.Disabled {
			t.Errorf("the %s select is disabled although its preference is settable", tc.label)
		}
	}
	for _, tc := range []struct{ label, value string }{
		{"Code font", "Follows the UI font"},
		{"Code theme", "Same as interface"},
	} {
		info, ok := h.tt.AccessByRole(ui.RolePopUpButton, tc.label)
		if !ok {
			t.Errorf("the read-only %s select is missing, texts %q", tc.label, h.tt.Texts())
			continue
		}
		if info.Value != tc.value {
			t.Errorf("the %s select shows %q, want %q", tc.label, info.Value, tc.value)
		}
		if !info.Disabled {
			t.Errorf("the read-only %s select is available although the shell has no such preference", tc.label)
		}
	}
	slider, ok := h.tt.AccessByRole(ui.RoleSlider, "Code size")
	if !ok {
		t.Fatalf("the code-size slider is missing, texts %q", h.tt.Texts())
	}
	if slider.Now != float64(k.codeFontSize) || slider.Min != settingsCodeMin || slider.Max != settingsCodeMax {
		t.Errorf("the code-size slider is %+v, want %.0f between %d and %d", slider, k.codeFontSize, settingsCodeMin, settingsCodeMax)
	}
	if info, ok := h.tt.AccessByRole(ui.RoleListItem, "Appearance"); !ok || !info.Selected {
		t.Errorf("the appearance page's navigation row is not selected: %+v, found %v", info, ok)
	}
}

// TestSettingsSectionsOrder checks the left column against the reference: the
// 17 reference sections in order, then the shell's own archived-sessions and
// about pages, with the Back row above them, the personalization page marked as
// the opening selection and no capability row of its own.
func TestSettingsSectionsOrder(t *testing.T) {
	want := []string{
		"Account", "Usage", "Personalization", "Appearance", "General",
		"Shortcuts", "Models", "Multimodal", "Runtime", "MCP",
		"Notifications", "Pet", "Permissions", "Git", "Hooks", "Plugins",
		"External Agent", "Archived sessions", "About",
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
	back, ok := h.tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keySettingsBack))
	if !ok {
		t.Fatalf("the navigation has no Back row, texts %q", h.tt.Texts())
	}
	lastY := back.Bounds.Y
	for i, name := range want {
		info, ok := h.tt.AccessByRole(ui.RoleListItem, name)
		if !ok {
			t.Fatalf("navigation row %d (%q) is missing, texts %q", i+1, name, h.tt.Texts())
		}
		if info.Bounds.Y <= lastY {
			t.Errorf("row %d (%q) sits at y=%.1f, not below the row before it at %.1f", i+1, name, info.Bounds.Y, lastY)
		}
		lastY = info.Bounds.Y
		if got := info.Selected; got != (name == "Personalization") {
			t.Errorf("row %q is selected=%v, want only Personalization selected", name, got)
		}
	}
}

// TestSettingsNavigationSwitchesSection checks a click on a navigation row:
// it changes only the route's chosen section, reports no preference change,
// and the page below switches from the opening personalization page to another
// WIP explanation and then to the appearance controls.
func TestSettingsNavigationSwitchesSection(t *testing.T) {
	h := newSettingsHarness(defaultSettingsView(newTokens(ui.LightTheme())))
	before := h.view

	if !h.tt.HasText(settingsSectionGaps[SettingsSectionPersonalization]) {
		t.Fatalf("the route did not open on the personalization page, texts %q", h.tt.Texts())
	}
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
	if info, ok := h.tt.AccessByRole(ui.RoleListItem, "Personalization"); !ok || info.Selected {
		t.Errorf("the Personalization row stayed selected after choosing General: %+v, found %v", info, ok)
	}
	if !h.tt.HasText(settingsSectionGaps[SettingsSectionGeneral]) {
		t.Errorf("the General page does not explain its gap, texts %q", h.tt.Texts())
	}
	if h.tt.HasText(tr(LocaleEN, keySettingsFontFamily)) {
		t.Errorf("the General page still draws the appearance font row, texts %q", h.tt.Texts())
	}

	if err := h.tt.Click("Appearance"); err != nil {
		t.Fatalf("click Appearance: %v", err)
	}
	if !h.tt.HasText(tr(LocaleEN, keySettingsFontFamily)) {
		t.Fatalf("the appearance page did not open, texts %q", h.tt.Texts())
	}
	if h.tt.HasText(settingsSectionGaps[SettingsSectionPersonalization]) {
		t.Errorf("the appearance page still draws the personalization gap, texts %q", h.tt.Texts())
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
// client or page, and draws no select, slider, radio or fake data that could
// look like a saved preference.
func TestSettingsWIPPagesOfferNoControls(t *testing.T) {
	for _, section := range settingsSections {
		if section == SettingsSectionAppearance {
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
			if !strings.Contains(gap, "Unavailable") || !strings.Contains(gap, "page") {
				t.Errorf("the %s gap %q does not name the missing client or page", name, gap)
			}
			if !h.tt.HasText(wipBadge) {
				t.Errorf("the %s page shows no visible %q badge, texts %q", name, wipBadge, h.tt.Texts())
			}
			for _, role := range []ui.Role{ui.RolePopUpButton, ui.RoleSlider, ui.RoleSwitch, ui.RoleRadio} {
				if info, ok := h.tt.AccessByRole(role, ""); ok {
					t.Errorf("the %s page offers a %v control: %+v", name, role, info)
				}
			}
			if len(h.changes) != 0 {
				t.Errorf("the %s page reported %+v, want no change", name, h.changes)
			}
		})
	}
}

// TestSettingsNarrowNavigationFits checks the settings route in a narrow
// window: it shows one level at a time instead of squeezing the section column
// and the page together. The section list takes the card first and every row
// stays inside the window; a row click opens that section's page as the second
// level, with a return row above it and its controls inside the window. The
// return row goes back to the list, Escape from the page does the same, and a
// second Escape closes the route.
func TestSettingsNarrowNavigationFits(t *testing.T) {
	const width, height = 320, 640
	h := newSettingsHarnessAt(defaultSettingsView(newTokens(ui.LightTheme())), width, height)

	nav, ok := h.tt.AccessByRole(ui.RoleNavigation, settingsNavigationLabel)
	if !ok {
		t.Fatalf("the section list is not exposed in a narrow window, texts %q", h.tt.Texts())
	}
	if nav.Bounds.X < 0 || nav.Bounds.X+nav.Bounds.W > width+0.5 || nav.Bounds.W <= 0 {
		t.Fatalf("the section list %+v does not fit the %d-wide window", nav.Bounds, width)
	}
	if nav.Bounds.W < width/2 {
		t.Errorf("the section list takes %.0f of the %d-wide window, want the first level to fill it", nav.Bounds.W, width)
	}
	for _, section := range settingsSections {
		name := settingsSectionName(section)
		info, ok := h.tt.AccessByRole(ui.RoleListItem, name)
		if !ok {
			t.Fatalf("the %s row is missing in a narrow window, texts %q", name, h.tt.Texts())
		}
		if info.Bounds.X < nav.Bounds.X-0.5 || info.Bounds.X+info.Bounds.W > nav.Bounds.X+nav.Bounds.W+0.5 {
			t.Errorf("the %s row %+v spills outside the list %+v", name, info.Bounds, nav.Bounds)
		}
	}
	if _, ok := h.tt.AccessByRole(ui.RoleScroll, ""); !ok {
		t.Errorf("the settings route has no scroll area, texts %q", h.tt.Texts())
	}
	if info, ok := h.tt.AccessByRole(ui.RoleListItem, "Personalization"); !ok || !info.Selected {
		t.Errorf("the narrow list does not open on the personalization selection: %+v, found %v", info, ok)
	}
	if _, ok := h.tt.AccessByRole(ui.RoleHeading, "Personalization"); ok {
		t.Errorf("the personalization page is drawn beside the list in a narrow window, texts %q", h.tt.Texts())
	}

	// A row click opens that section's page as the second level.
	if err := h.tt.Click("Appearance"); err != nil {
		t.Fatalf("click Appearance: %v", err)
	}
	if _, ok := h.tt.AccessByRole(ui.RoleNavigation, settingsNavigationLabel); ok {
		t.Errorf("the section list stayed beside the page in a narrow window, texts %q", h.tt.Texts())
	}
	heading, ok := h.tt.AccessByRole(ui.RoleHeading, "Appearance")
	if !ok {
		t.Fatalf("the appearance page heading is missing, texts %q", h.tt.Texts())
	}
	if heading.Bounds.X+heading.Bounds.W > width+0.5 {
		t.Errorf("the page heading %+v spills past the %d-wide window", heading.Bounds, width)
	}
	for _, name := range []string{"Light", "Dark", "System"} {
		info, ok := h.tt.AccessByRole(ui.RoleRadio, name)
		if !ok {
			t.Fatalf("the %s theme card is missing in a narrow window, texts %q", name, h.tt.Texts())
		}
		if info.Bounds.X+info.Bounds.W > width+0.5 {
			t.Errorf("the %s theme card %+v spills past the %d-wide window", name, info.Bounds, width)
		}
	}
	for _, name := range []string{"XS", "S", "M", "L", "XL"} {
		info, ok := h.tt.AccessByRole(ui.RoleRadio, name)
		if !ok {
			t.Fatalf("the %s size segment is missing in a narrow window, texts %q", name, h.tt.Texts())
		}
		if info.Bounds.X+info.Bounds.W > width+0.5 {
			t.Errorf("the %s size segment %+v spills past the %d-wide window", name, info.Bounds, width)
		}
	}
	for _, name := range []string{tr(LocaleEN, keySettingsSkinNew), tr(LocaleEN, keySettingsSkinOpen)} {
		info, ok := h.tt.AccessByRole(ui.RoleButton, name)
		if !ok {
			t.Fatalf("the %s action is missing in a narrow window, texts %q", name, h.tt.Texts())
		}
		if info.Bounds.X+info.Bounds.W > width+0.5 {
			t.Errorf("the %s action %+v spills past the %d-wide window", name, info.Bounds, width)
		}
	}
	if len(h.changes) != 0 {
		t.Errorf("opening a section page reported %+v, want no preference change", h.changes)
	}

	// Tab in the compact page starts at its return row.
	h.tt.Key(0, ui.KeyTab)
	if !h.tt.Focused(settingsSectionsBack) {
		t.Errorf("the first Tab in the section page reached %q, want the %q return row", focusedName(h.tt), settingsSectionsBack)
	}

	// The return row is the way back to the list; it is not the route's Back.
	backToList, ok := h.tt.AccessByRole(ui.RoleButton, settingsSectionsBack)
	if !ok {
		t.Fatalf("the section page has no %q return row, texts %q", settingsSectionsBack, h.tt.Texts())
	}
	if backToList.Bounds.X+backToList.Bounds.W > width+0.5 {
		t.Errorf("the return row %+v spills past the %d-wide window", backToList.Bounds, width)
	}
	if err := h.tt.Click(settingsSectionsBack); err != nil {
		t.Fatalf("click %s: %v", settingsSectionsBack, err)
	}
	if h.backs != 0 {
		t.Errorf("the return row closed the route %d times, want only the list back", h.backs)
	}
	if _, ok := h.tt.AccessByRole(ui.RoleNavigation, settingsNavigationLabel); !ok {
		t.Fatalf("the return row did not bring the section list back, texts %q", h.tt.Texts())
	}
	if info, ok := h.tt.AccessByRole(ui.RoleListItem, "Appearance"); !ok || !info.Selected {
		t.Errorf("the returned-to list lost the Appearance selection: %+v, found %v", info, ok)
	}

	// Escape from the page returns to the list first and closes the route only
	// from the list.
	if err := h.tt.Click("Appearance"); err != nil {
		t.Fatalf("click Appearance: %v", err)
	}
	h.tt.Key(0, ui.KeyEscape)
	if h.backs != 0 {
		t.Errorf("Escape closed the route from the section page, want the return to the list")
	}
	if _, ok := h.tt.AccessByRole(ui.RoleNavigation, settingsNavigationLabel); !ok {
		t.Fatalf("Escape did not bring the section list back, texts %q", h.tt.Texts())
	}
	h.tt.Key(0, ui.KeyEscape)
	if h.backs != 1 {
		t.Errorf("Escape from the section list called the back callback %d times, want 1", h.backs)
	}
}

// TestSettingsCompactBreakpoint pins where the route changes layout: a window
// at settingsCompactW still gives the section column and the page their
// side-by-side room, and one DIP narrower shows the section list alone with the
// page behind a row click.
func TestSettingsCompactBreakpoint(t *testing.T) {
	wide := newSettingsHarnessAt(defaultSettingsView(newTokens(ui.LightTheme())), settingsCompactW, 900)
	if _, ok := wide.tt.AccessByRole(ui.RoleNavigation, settingsNavigationLabel); !ok {
		t.Fatalf("the section column is missing at the breakpoint, texts %q", wide.tt.Texts())
	}
	if _, ok := wide.tt.AccessByRole(ui.RoleHeading, "Personalization"); !ok {
		t.Errorf("the page is not drawn beside the column at %d wide, texts %q", settingsCompactW, wide.tt.Texts())
	}

	narrow := newSettingsHarnessAt(defaultSettingsView(newTokens(ui.LightTheme())), settingsCompactW-1, 900)
	if _, ok := narrow.tt.AccessByRole(ui.RoleNavigation, settingsNavigationLabel); !ok {
		t.Fatalf("the section list is missing just below the breakpoint, texts %q", narrow.tt.Texts())
	}
	if _, ok := narrow.tt.AccessByRole(ui.RoleHeading, "Personalization"); ok {
		t.Errorf("the page still shares the card just below %d, texts %q", settingsCompactW, narrow.tt.Texts())
	}
}

// TestSettingsReportsOneChangePerChoice checks each control: one choice reports
// exactly one change, with the field and value the user chose. A redraw does
// not report the choice again, and the change carries the resolved view, so the
// window can store either the one value or the whole snapshot.
func TestSettingsReportsOneChangePerChoice(t *testing.T) {
	h := newSettingsHarness(SettingsView{Section: SettingsSectionAppearance})

	cases := []struct {
		choose func()
		field  SettingsField
		check  func(SettingsChange) bool
	}{
		{func() { h.tt.Click("Dark") }, SettingsFieldTheme, func(c SettingsChange) bool { return c.Theme == ThemeDark }},
		{func() { chooseOption(t, h.tt, "UI font", "Monospace") }, SettingsFieldFontFamily, func(c SettingsChange) bool { return c.FontFamily == FontMonospace }},
		{func() { h.tt.Click("L") }, SettingsFieldFontSize, func(c SettingsChange) bool { return c.FontSize == 16 }},
		{func() { clickAccess(t, h.tt, ui.RoleSlider, "Code size") }, SettingsFieldCodeFontSize, func(c SettingsChange) bool {
			return c.CodeFontSize >= settingsCodeMin && c.CodeFontSize <= settingsCodeMax
		}},
	}
	for i, tc := range cases {
		before := len(h.changes)
		tc.choose()
		if got := len(h.changes); got != before+1 {
			t.Fatalf("choice %d reported %d changes, want exactly 1", i+1, got-before)
		}
		change := h.changes[len(h.changes)-1]
		if change.Field != tc.field {
			t.Errorf("choice %d reported field %v, want %v", i+1, change.Field, tc.field)
		}
		if !tc.check(change) {
			t.Errorf("choice %d reported %+v", i+1, change)
		}
		h.tt.Frame()
		if got := len(h.changes); got != before+1 {
			t.Errorf("choice %d reported %d changes after a redraw, want 1", i+1, got-before)
		}
	}

	// The language select is last because it redraws the page in the chosen
	// language; choosing it reports one change and the next frame is Chinese.
	before := len(h.changes)
	chooseOption(t, h.tt, "Interface language", "简体中文")
	if got := len(h.changes); got != before+1 {
		t.Fatalf("the language choice reported %d changes, want 1", got-before)
	}
	change := h.changes[len(h.changes)-1]
	if change.Field != SettingsFieldLocale || change.Locale != LocaleZHCN {
		t.Errorf("the language choice reported %+v, want zh-CN", change)
	}
	if !h.tt.HasText("外观") || !h.tt.HasText("界面语言") {
		t.Errorf("the next frame is not Chinese, texts %q", h.tt.Texts())
	}
}

// TestSettingsThemeCardsFollowReferenceOrder checks the appearance page's theme
// cards: Light, Dark and System from left to right, as the reference draws them,
// and each card saving its own ThemeChoice rather than its position. Light is
// the first card and still saves ThemeLight, and System is the last and saves
// the enum's zero value.
func TestSettingsThemeCardsFollowReferenceOrder(t *testing.T) {
	h := newSettingsHarness(SettingsView{Section: SettingsSectionAppearance})

	order := []string{"Light", "Dark", "System"}
	lastRight := float32(-1)
	for _, name := range order {
		info, ok := h.tt.AccessByRole(ui.RoleRadio, name)
		if !ok {
			t.Fatalf("the %s theme card is missing, texts %q", name, h.tt.Texts())
		}
		if info.Bounds.X < lastRight {
			t.Errorf("the %s card starts at x=%.1f, left of the card before it ending at %.1f", name, info.Bounds.X, lastRight)
		}
		lastRight = info.Bounds.X + info.Bounds.W
	}

	for _, tc := range []struct {
		name string
		want ThemeChoice
	}{
		{"Light", ThemeLight},
		{"Dark", ThemeDark},
		{"System", ThemeSystem},
	} {
		before := len(h.changes)
		if err := h.tt.Click(tc.name); err != nil {
			t.Fatalf("click %s: %v", tc.name, err)
		}
		if got := len(h.changes); got != before+1 {
			t.Fatalf("clicking %s reported %d changes, want 1", tc.name, got-before)
		}
		change := h.changes[len(h.changes)-1]
		if change.Field != SettingsFieldTheme || change.Theme != tc.want {
			t.Errorf("clicking %s reported %+v, want %v", tc.name, change, tc.want)
		}
		if info, ok := h.tt.AccessByRole(ui.RoleRadio, tc.name); !ok || !info.Checked {
			t.Errorf("the %s card is not checked after its click: %+v, found %v", tc.name, info, ok)
		}
	}
}

// TestSettingsUISizeSegmentsChooseEachStep checks the five UI-size segments:
// each one reports its own step, and the chosen segment is the one selected on
// the next frame.
func TestSettingsUISizeSegmentsChooseEachStep(t *testing.T) {
	h := newSettingsHarness(SettingsView{Section: SettingsSectionAppearance})
	labels := []string{
		tr(LocaleEN, keySettingsSizeXS),
		tr(LocaleEN, keySettingsSizeSmall),
		tr(LocaleEN, keySettingsSizeMedium),
		tr(LocaleEN, keySettingsSizeLarge),
		tr(LocaleEN, keySettingsSizeXL),
	}
	for i, label := range labels {
		before := len(h.changes)
		if err := h.tt.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
		if got := len(h.changes); got != before+1 {
			t.Fatalf("segment %q reported %d changes, want 1", label, got-before)
		}
		change := h.changes[len(h.changes)-1]
		if change.Field != SettingsFieldFontSize || change.FontSize != fontSizeSteps[i] {
			t.Errorf("segment %q reported %+v, want font size %.0f", label, change, fontSizeSteps[i])
		}
		if info, ok := h.tt.AccessByRole(ui.RoleRadio, label); !ok || !info.Checked {
			t.Errorf("segment %q is not selected after its click: %+v, found %v", label, info, ok)
		}
	}
}

// TestSettingsCodeSizeSliderStaysInRange checks the code-size slider: a saved
// value outside 12–20 is drawn at the nearest bound, and the slider exposes
// that range and step to assistive technology.
func TestSettingsCodeSizeSliderStaysInRange(t *testing.T) {
	for _, tc := range []struct {
		name  string
		saved float32
		want  float64
	}{
		{"above the largest", 99, settingsCodeMax},
		{"below the smallest", 1, settingsCodeMin},
		{"inside the range", 16, 16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newSettingsHarnessAt(SettingsView{Section: SettingsSectionAppearance, CodeFontSize: tc.saved}, 900, 900)
			slider, ok := h.tt.AccessByRole(ui.RoleSlider, "Code size")
			if !ok {
				t.Fatalf("the code-size slider is missing, texts %q", h.tt.Texts())
			}
			if slider.Now != tc.want {
				t.Errorf("the slider shows %.0f, want %.0f", slider.Now, tc.want)
			}
			if slider.Min != settingsCodeMin || slider.Max != settingsCodeMax || slider.Step != settingsCodeStep {
				t.Errorf("the slider range is %+v, want %d–%d step %d", slider, settingsCodeMin, settingsCodeMax, settingsCodeStep)
			}
			if len(h.changes) != 0 {
				t.Errorf("drawing the slider reported %+v, want no change", h.changes)
			}
		})
	}
}

// TestSettingsWIPEntriesDoNothing checks the WIP parts of the appearance page:
// the gallery's covers and its two actions stay visible, unavailable and
// explained, the read-only code controls are disabled, and a click changes no
// preference. The page offers no fake skin, no fake code preview and no third
// color theme.
func TestSettingsWIPEntriesDoNothing(t *testing.T) {
	// The gallery sits below the fold of a 900-high window, so this test draws
	// a tall window and can click every cover and action.
	h := newSettingsHarnessAt(SettingsView{Section: SettingsSectionAppearance}, 900, 1600)

	for _, label := range []string{"New from image", "Open folder"} {
		info, ok := h.tt.AccessByRole(ui.RoleButton, label)
		if !ok {
			t.Errorf("%q is not exposed as an entry, texts %q", label, h.tt.Texts())
			continue
		}
		if !info.Disabled {
			t.Errorf("%q is available although the shell cannot open a skin", label)
		}
		if !strings.Contains(info.Description, "skin") {
			t.Errorf("%q description %q does not name the missing skin support", label, info.Description)
		}
		if err := h.tt.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
	}
	if len(h.changes) != 0 {
		t.Errorf("a WIP click reported %+v, want no change", h.changes)
	}
	for i := 1; i <= 6; i++ {
		if !h.tt.HasText(tr(LocaleEN, keySettingsSkinCard) + " " + strconv.Itoa(i)) {
			t.Errorf("skin cover %d is missing, texts %q", i, h.tt.Texts())
		}
	}
	if !h.tt.HasText(wipBadge) {
		t.Errorf("the WIP entries show no visible %q badge, texts %q", wipBadge, h.tt.Texts())
	}
	for _, fake := range []string{"Windows 98", "Windows XP", "Windows Vista", "macOS Big Sur", "OSX Tiger"} {
		if h.tt.HasText(fake) {
			t.Errorf("the surface offers the fake chrome theme %q", fake)
		}
	}
}

// TestSettingsNavigationWorksInShell checks the navigation inside the shell's
// own frame, not just the route's harness: opening settings starts on the
// personalization page, a click on another section replaces the page, the
// appearance page is still reachable, and the selection survives the next
// frame.
func TestSettingsNavigationWorksInShell(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(settingsSectionGaps[SettingsSectionPersonalization]) {
		t.Fatalf("the settings route did not open on the personalization page, texts %q", tt.Texts())
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
	if !tt.HasText(tr(LocaleEN, keySettingsFontFamily)) {
		t.Fatalf("the appearance page did not open in the shell, texts %q", tt.Texts())
	}
}

// TestSettingsKeyboardReachable checks the Tab order: the Back row, then the
// section navigation in the reference's order, then the appearance controls in
// reading order, including the read-only controls whose reasons a screen-reader
// user must still find. Escape calls the same callback as Back.
func TestSettingsKeyboardReachable(t *testing.T) {
	h := newSettingsHarnessAt(SettingsView{Section: SettingsSectionAppearance}, 900, 1600)

	want := []string{
		tr(LocaleEN, keySettingsBack),
		"Account", "Usage", "Personalization", "Appearance", "General",
		"Shortcuts", "Models", "Multimodal", "Runtime", "MCP",
		"Notifications", "Pet", "Permissions", "Git", "Hooks", "Plugins",
		"External Agent", "Archived sessions", "About",
		"Light", "Dark", "System",
		tr(LocaleEN, keySettingsLanguageRow),
		tr(LocaleEN, keySettingsFontFamily),
		"S",
		tr(LocaleEN, keySettingsCodeFontSize),
		tr(LocaleEN, keySettingsSkinNew),
		tr(LocaleEN, keySettingsSkinOpen),
	}
	for i, name := range want {
		h.tt.Key(0, ui.KeyTab)
		if !h.tt.Focused(name) {
			t.Fatalf("Tab %d did not reach %q, texts %q; focused %q", i+1, name, h.tt.Texts(), focusedName(h.tt))
		}
	}
	h.tt.Key(0, ui.KeyEscape)
	if h.backs != 1 {
		t.Fatalf("Escape called the back callback %d times, want 1", h.backs)
	}
}

// focusedName reports the focused accessible name, for a failed Tab assertion.
func focusedName(tt *ui.Tester) string {
	for _, name := range []string{
		tr(LocaleEN, keySettingsBack),
		"Account", "Usage", "Personalization", "Appearance", "General",
		"Shortcuts", "Models", "Multimodal", "Runtime", "MCP",
		"Notifications", "Pet", "Permissions", "Git", "Hooks", "Plugins",
		"External Agent", "Archived sessions", "About",
		"Light", "Dark", "System",
		tr(LocaleEN, keySettingsLanguageRow),
		tr(LocaleEN, keySettingsFontFamily),
		"S", "XS", "M", "L", "XL",
		tr(LocaleEN, keySettingsCodeFontSize),
		tr(LocaleEN, keySettingsSkinNew),
		tr(LocaleEN, keySettingsSkinOpen),
		settingsSectionsBack,
	} {
		if tt.Focused(name) {
			return name
		}
	}
	for _, role := range []ui.Role{ui.RoleButton, ui.RoleListItem, ui.RoleRadio, ui.RolePopUpButton, ui.RoleSlider, ui.RoleSwitch} {
		if info, ok := tt.AccessByRole(role, ""); ok && info.Focused {
			return string(roleName(role)) + ":" + info.Name
		}
	}
	return ""
}

// roleName keeps focusedName's failure text short without depending on the
// platform's role name.
func roleName(role ui.Role) string {
	switch role {
	case ui.RoleListItem:
		return "listitem"
	case ui.RolePopUpButton:
		return "select"
	case ui.RoleRadio:
		return "radio"
	case ui.RoleSlider:
		return "slider"
	default:
		return "control"
	}
}

// TestSettingsAboutPageListsCapabilities checks where the shell's capability
// gaps live now: the settings navigation has no "Unavailable capabilities" row,
// and the About page carries the list beside its own WIP explanation. The
// sidebar itself draws none of the cards.
func TestSettingsAboutPageListsCapabilities(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 900)

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Unavailable capabilities"); ok {
		t.Fatalf("the sidebar still draws the capability cards, texts %q", tt.Texts())
	}
	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.AccessByRole(ui.RoleListItem, "Unavailable capabilities"); ok {
		t.Fatalf("the navigation still carries an unavailable-capabilities row, texts %q", tt.Texts())
	}
	if err := tt.Click("About"); err != nil {
		t.Fatal(err)
	}
	if info, ok := tt.AccessByRole(ui.RoleHeading, "About"); !ok || info.Level != 1 {
		t.Fatalf("the About page has no level-1 heading: %+v, found %v", info, ok)
	}
	if !tt.HasText(settingsSectionGaps[SettingsSectionAbout]) || !tt.HasText(wipBadge) {
		t.Errorf("the About page lost its WIP explanation, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Unavailable capabilities"); !ok {
		t.Fatalf("the About page does not group the capability cards, texts %q", tt.Texts())
	}
	for _, capability := range wipCapabilities(s) {
		for _, want := range []string{capability.action, capability.method, capability.gap} {
			if !tt.HasText(want) {
				t.Errorf("the About page is missing %q, texts %q", want, tt.Texts())
			}
		}
	}
	if !tt.HasText(settingsCapabilitiesNote) {
		t.Errorf("the About page lost the capability note, texts %q", tt.Texts())
	}
}

// TestSettingsRouteTakesTheWholeWindow checks the route boundary: while
// settings is open the frame has no session sidebar, conversation header,
// conversation rows or composer, and closing it through Back restores the frame
// with the selection and draft untouched.
func TestSettingsRouteTakesTheWholeWindow(t *testing.T) {
	s := fixture()
	s.Draft = "keep this draft"
	tt := ui.NewTester(s.View, 900, 600)
	if !tt.HasText("hello from the fixture") {
		t.Fatalf("the fixture has no conversation, texts %q", tt.Texts())
	}

	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.AccessByRole(ui.RoleNavigation, tr(LocaleEN, keySidebarNavigation)); ok {
		t.Errorf("the session sidebar is still drawn behind the settings route, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleToolbar, tr(LocaleEN, keyHeaderRegion)); ok {
		t.Errorf("the conversation header is still drawn behind the settings route, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, tr(LocaleEN, keyComposerLabel)); ok {
		t.Errorf("the composer is still drawn behind the settings route, texts %q", tt.Texts())
	}
	if tt.HasText("hello from the fixture") {
		t.Errorf("the conversation is still drawn behind the settings route, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, tr(LocaleEN, keySettingsRegion)); !ok {
		t.Errorf("the settings route is not the window's region, texts %q", tt.Texts())
	}

	if err := tt.Click(tr(LocaleEN, keySettingsBack)); err != nil {
		t.Fatal(err)
	}
	if s.ShowSettings {
		t.Fatal("Back did not close the settings route")
	}
	for _, want := range []string{"hello from the fixture", tr(LocaleEN, keyComposerLabel)} {
		if !tt.HasText(want) {
			t.Errorf("the conversation did not return with %q, texts %q", want, tt.Texts())
		}
	}
	if s.Selected != "s1" || s.Draft != "keep this draft" {
		t.Errorf("closing settings changed the window: selected=%q draft=%q", s.Selected, s.Draft)
	}

	// Escape is the same return.
	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if s.ShowSettings {
		t.Fatal("Escape did not close the settings route")
	}
	if !tt.HasText("hello from the fixture") {
		t.Errorf("Escape did not restore the conversation, texts %q", tt.Texts())
	}
}
