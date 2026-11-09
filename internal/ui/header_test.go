package ui

import (
	"image/color"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// headerTestView renders one HeaderView directly, with no App Server and no
// callbacks, so a test can measure the bar without the rest of the shell.
func headerTestView(v HeaderView, width, height int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		header(c, newTokens(c.Theme()), v, Actions{})
	}, width, height)
}

// clickMenuItem clicks the menu item with label, found by its menu role so a
// page text with the same words cannot take the click.
func clickMenuItem(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleMenuItem, label)
	if !ok {
		t.Fatalf("the menu has no item %q, texts %q", label, tt.Texts())
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
}

// TestHeaderBreadcrumbShowsProjectThenTitle checks the reference's one-line
// breadcrumb: the catalog's project label, the separator and the selected
// title, with both parts single-line and inside the bar.
func TestHeaderBreadcrumbShowsProjectThenTitle(t *testing.T) {
	v := fixture().headerView()
	v.Project = "dimagent-mygo"
	v.Title = "Explain what this repository does in one paragraph."
	tt := headerTestView(v, 800, 200)

	project, ok := tt.Find(v.Project)
	if !ok {
		t.Fatalf("the project name is not drawn, texts %q", tt.Texts())
	}
	title, ok := tt.Find(v.Title)
	if !ok {
		t.Fatalf("the title is not drawn, texts %q", tt.Texts())
	}
	slash, ok := tt.Find(breadcrumbSeparator)
	if !ok {
		t.Fatalf("the project separator is not drawn, texts %q", tt.Texts())
	}
	if !(project.X < slash.X && slash.X < title.X) {
		t.Errorf("breadcrumb order project %.1f, separator %.1f, title %.1f", project.X, slash.X, title.X)
	}
	if _, ok := tt.AccessByRole(ui.RoleToolbar, tr(LocaleEN, keyHeaderRegion)); !ok {
		t.Errorf("the bar is not exposed as the conversation header, texts %q", tt.Texts())
	}
}

// TestHeaderBreadcrumbTruncatesBeforeTheMenu checks that a long project and
// title stay on one line in a narrow bar: both end before the session menu
// rather than pushing it, the editor entry or the panel toggle out of the
// window.
func TestHeaderBreadcrumbTruncatesBeforeTheMenu(t *testing.T) {
	v := fixture().headerView()
	v.Project = "dimagent-mygo-with-a-very-long-project-name"
	v.Title = "Explain what this repository does in one paragraph, in detail, without stopping"
	tt := headerTestView(v, 420, 200)

	title, ok := tt.Find(v.Title)
	if !ok {
		t.Fatalf("the title is not drawn, texts %q", tt.Texts())
	}
	menu, ok := tt.AccessByRole(ui.RoleMenuButton, tr(LocaleEN, keyHeaderMenu))
	if !ok {
		t.Fatalf("the session menu is not drawn, texts %q", tt.Texts())
	}
	if title.X+title.W > menu.Bounds.X+0.5 {
		t.Errorf("the title ends at %.1f, past the menu at %.1f", title.X+title.W, menu.Bounds.X)
	}
	if menu.Bounds.X+menu.Bounds.W > 420.5 {
		t.Errorf("the menu ends at %.1f, past the 420 DIP window", menu.Bounds.X+menu.Bounds.W)
	}
	if title.H > newTokens(ui.LightTheme()).headerH {
		t.Errorf("the title is %.1f high, taller than the bar", title.H)
	}
}

// TestHeaderOmitsTheBreadcrumbWithoutAProject checks that an empty catalog
// group draws neither a project name nor its separator: the shell never invents
// a project.
func TestHeaderOmitsTheBreadcrumbWithoutAProject(t *testing.T) {
	v := fixture().headerView()
	if v.Project != "" {
		t.Fatalf("the fixture carries a project %q", v.Project)
	}
	tt := headerTestView(v, 800, 200)

	if _, ok := tt.Find(breadcrumbSeparator); ok {
		t.Errorf("a bar without a project draws the separator, texts %q", tt.Texts())
	}
	if !tt.HasText(v.Title) {
		t.Errorf("a bar without a project lost the title, texts %q", tt.Texts())
	}
}

// TestHeaderHasNoStatusText checks the reference's conversation bar: no
// "Session: …" run row, no "App Server: …" reading and no Settings button in
// the bar itself. The sidebar keeps its own Settings entry.
func TestHeaderHasNoStatusText(t *testing.T) {
	s := fixture()
	s.Connection = ConnReady
	tt := ui.NewTester(s.View, 900, 600)

	header, ok := tt.AccessByRole(ui.RoleToolbar, tr(LocaleEN, keyHeaderRegion))
	if !ok {
		t.Fatalf("the conversation header is not drawn, texts %q", tt.Texts())
	}
	for _, gone := range []string{"Session:", "App Server:"} {
		if tt.HasText(gone) {
			t.Errorf("the bar still draws %q, texts %q", gone, tt.Texts())
		}
	}
	settings, ok := tt.Find(tr(LocaleEN, keyHeaderSettings))
	if ok && containsRect(header.Bounds, settings) {
		t.Errorf("the bar still draws a Settings entry at %+v", settings)
	}
}

// TestSessionMenuItemsAndCallbacks checks the menu's entries in the reference's
// order and that each entry reaches its callback exactly once: the keys browser
// opens, the branch entries address the displayed branch, the delete entry
// opens a confirmation instead of deleting, and the settings entry toggles the
// appearance surface both ways.
func TestSessionMenuItemsAndCallbacks(t *testing.T) {
	s := fixture()
	s.SelectedBranch = "side"
	s.Keys = KeysPanelView{Loaded: true}
	ends, refreshes, deletes := 0, 0, 0
	s.OnEndBranch = func(branchID string) {
		if branchID != "side" {
			t.Errorf("end branch got %q, want side", branchID)
		}
		ends++
	}
	s.OnRefreshBranches = func() { refreshes++ }
	s.OnDeleteSession = func(string) { deletes++ }
	tt := ui.NewTester(s.View, 900, 600)

	openMenu := func() {
		t.Helper()
		if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
			t.Fatalf("open the session menu: %v", err)
		}
	}

	openMenu()
	labels := []string{
		menuText(LocaleEN, keySidebarPin),
		menuText(LocaleEN, keySidebarRename),
		menuText(LocaleEN, keyHeaderMenuOpenInNewWindow),
		menuText(LocaleEN, keyHeaderMenuExport),
		menuText(LocaleEN, keySidebarArchive),
		menuText(LocaleEN, keyHeaderMenuCopyLink),
		menuText(LocaleEN, keyHeaderMenuCopyID),
		menuText(LocaleEN, keyHeaderMenuCopyDebug),
		tr(LocaleEN, keySidebarKeys),
		tr(LocaleEN, keySidebarEndBranch),
		tr(LocaleEN, keyHeaderMenuRefresh),
		tr(LocaleEN, keySidebarDelete),
		tr(LocaleEN, keyHeaderSettings),
	}
	previous := float32(-1)
	for _, label := range labels {
		info, ok := tt.AccessByRole(ui.RoleMenuItem, label)
		if !ok {
			t.Fatalf("the menu has no %q item, texts %q", label, tt.Texts())
		}
		if info.Bounds.Y <= previous {
			t.Errorf("%q sits at y=%.1f, not below the item before it at %.1f", label, info.Bounds.Y, previous)
		}
		previous = info.Bounds.Y
	}

	clickMenuItem(t, tt, tr(LocaleEN, keySidebarKeys))
	if !s.ShowKeys {
		t.Fatal("the keys entry did not open the keys browser")
	}

	openMenu()
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarEndBranch))
	if ends != 1 {
		t.Errorf("End branch ran %d times, want 1", ends)
	}

	openMenu()
	clickMenuItem(t, tt, tr(LocaleEN, keyHeaderMenuRefresh))
	if refreshes != 1 {
		t.Errorf("Refresh branches ran %d times, want 1", refreshes)
	}

	openMenu()
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarDelete))
	if deletes != 0 {
		t.Fatalf("the delete menu item deleted without confirmation (%d calls)", deletes)
	}
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, tr(LocaleEN, keyHeaderDeleteTitle)); !ok {
		t.Fatalf("the delete confirmation is not drawn, texts %q", tt.Texts())
	}
	if err := tt.Click(tr(LocaleEN, keyHeaderDeleteCancel)); err != nil {
		t.Fatal(err)
	}
	if deletes != 0 || s.DeleteConfirm {
		t.Fatalf("Cancel: %d deletes, confirmation %v", deletes, s.DeleteConfirm)
	}

	openMenu()
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarDelete))
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, tr(LocaleEN, keyHeaderDeleteTitle)); !ok {
		t.Fatalf("the second confirmation is not drawn, texts %q", tt.Texts())
	}
	if err := tt.Click(tr(LocaleEN, keyHeaderDeleteConfirm)); err != nil {
		t.Fatal(err)
	}
	if deletes != 1 {
		t.Fatalf("the confirmed delete ran %d times, want 1", deletes)
	}

	openMenu()
	clickMenuItem(t, tt, tr(LocaleEN, keyHeaderSettings))
	if !s.ShowSettings {
		t.Fatal("the settings entry did not open the appearance route")
	}
	// The route replaces the whole window, so the return is its Back row rather
	// than a second header-menu item.
	if err := tt.Click(tr(LocaleEN, keySettingsBack)); err != nil {
		t.Fatal(err)
	}
	if s.ShowSettings {
		t.Fatal("Back did not return from the appearance route")
	}
}

// TestSessionMenuClosesOnEscape checks that Escape closes the menu and gives
// the focus back to its button, so a keyboard user can leave it.
func TestSessionMenuClosesOnEscape(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.AccessByRole(ui.RoleMenuItem, tr(LocaleEN, keySidebarKeys)); !ok {
		t.Fatalf("the menu did not open, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if _, ok := tt.AccessByRole(ui.RoleMenuItem, tr(LocaleEN, keySidebarKeys)); ok {
		t.Errorf("Escape left the menu open, texts %q", tt.Texts())
	}
	if !tt.Focused(tr(LocaleEN, keyHeaderMenu)) {
		t.Errorf("Escape did not return the focus to the menu button, texts %q", tt.Texts())
	}
}

// TestSessionMenuDeleteEscapeCancels checks that Escape closes the delete
// confirmation without deleting the session.
func TestSessionMenuDeleteEscapeCancels(t *testing.T) {
	s := fixture()
	deletes := 0
	s.OnDeleteSession = func(string) { deletes++ }
	tt := ui.NewTester(s.View, 900, 600)

	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarDelete))
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, tr(LocaleEN, keyHeaderDeleteTitle)); !ok {
		t.Fatalf("the delete confirmation is not drawn, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, tr(LocaleEN, keyHeaderDeleteTitle)); ok {
		t.Errorf("Escape left the confirmation open, texts %q", tt.Texts())
	}
	if deletes != 0 || s.DeleteConfirm {
		t.Fatalf("Escape: %d deletes, confirmation %v", deletes, s.DeleteConfirm)
	}
}

// TestSessionMenuEndsOnlyANonMainBranch checks the visible condition of the
// "End branch" entry: it is absent for no branch and for main, and present for
// a side branch. Without a callback it is present but unavailable and marked
// WIP with the session.branch_end reason.
func TestSessionMenuEndsOnlyANonMainBranch(t *testing.T) {
	for _, tc := range []struct {
		branch string
		want   bool
	}{
		{"", false},
		{mainBranchID, false},
		{"side", true},
	} {
		s := fixture()
		s.SelectedBranch = tc.branch
		tt := ui.NewTester(s.View, 900, 600)
		if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
			t.Fatal(err)
		}
		info, ok := tt.AccessByRole(ui.RoleMenuItem, tr(LocaleEN, keySidebarEndBranch))
		if ok != tc.want {
			t.Errorf("branch %q: End branch shown=%v, want %v, texts %q", tc.branch, ok, tc.want, tt.Texts())
			continue
		}
		if !tc.want {
			continue
		}
		if !info.Disabled {
			t.Errorf("branch %q: End branch is available without a callback", tc.branch)
		}
		if !strings.Contains(info.Description, tr(LocaleEN, keyReasonEndBranch)) {
			t.Errorf("branch %q: description %q does not name the gap", tc.branch, info.Description)
		}
		if !tt.HasText(wipBadge) {
			t.Errorf("branch %q: the WIP entry carries no visible badge, texts %q", tc.branch, tt.Texts())
		}
	}
}

// TestSessionMenuMarksMissingCallbacksAsWIP checks every menu entry the shell
// cannot carry out: it stays visible, disabled, marked WIP and explained with
// the method or fact it waits on, while the desktop-local settings entry and
// the local copy entries stay usable. The first group's entries wait on a client
// or callback the shell does not have at all; Pin, Rename and Archive reuse the
// sidebar's reasons.
func TestSessionMenuMarksMissingCallbacksAsWIP(t *testing.T) {
	s := fixture()
	s.SelectedBranch = "side"
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ label, reason string }{
		{menuText(LocaleEN, keySidebarPin), tr(LocaleEN, keyReasonPin)},
		{menuText(LocaleEN, keySidebarRename), tr(LocaleEN, keyReasonRename)},
		{menuText(LocaleEN, keyHeaderMenuOpenInNewWindow), menuText(LocaleEN, keyReasonOpenInNewWindow)},
		{menuText(LocaleEN, keyHeaderMenuExport), menuText(LocaleEN, keyReasonExport)},
		{menuText(LocaleEN, keySidebarArchive), tr(LocaleEN, keyReasonArchive)},
		{menuText(LocaleEN, keyHeaderMenuCopyDebug), menuText(LocaleEN, keyReasonCopyDebug)},
		{tr(LocaleEN, keySidebarKeys), tr(LocaleEN, keyReasonKeysEntry)},
		{tr(LocaleEN, keySidebarEndBranch), tr(LocaleEN, keyReasonEndBranch)},
		{tr(LocaleEN, keyHeaderMenuRefresh), tr(LocaleEN, keyReasonRefreshBranches)},
		{tr(LocaleEN, keySidebarDelete), tr(LocaleEN, keyReasonDelete)},
	} {
		info, ok := tt.AccessByRole(ui.RoleMenuItem, tc.label)
		if !ok {
			t.Fatalf("the menu has no %q item, texts %q", tc.label, tt.Texts())
		}
		if !info.Disabled {
			t.Errorf("%q is available without its callback", tc.label)
		}
		if !strings.Contains(info.Description, tc.reason) {
			t.Errorf("%q description %q does not name %q", tc.label, info.Description, tc.reason)
		}
		if math.Abs(float64(info.Bounds.H-menuRowH)) > 0.5 {
			t.Errorf("%q is %.1f high, want the reference's %v DIP row", tc.label, info.Bounds.H, menuRowH)
		}
	}
	if settings, ok := tt.AccessByRole(ui.RoleMenuItem, tr(LocaleEN, keyHeaderSettings)); !ok || settings.Disabled {
		t.Errorf("the desktop-local settings entry is missing or disabled: %+v", settings)
	}
	if !tt.HasText(wipBadge) {
		t.Errorf("the WIP entries carry no visible badge, texts %q", tt.Texts())
	}
}

// TestSessionMenuFollowsTheReferenceStructure checks the menu's two groups and
// the rule between them against the reference: pin, rename, open in a new
// window, export, archive, the three copy entries, then the keys, branch and
// delete/settings entries the shell has. Separators sit between the groups, not
// between every item.
func TestSessionMenuFollowsTheReferenceStructure(t *testing.T) {
	s := fixture()
	s.SelectedBranch = "side"
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}

	want := []string{
		menuText(LocaleEN, keySidebarPin),
		menuText(LocaleEN, keySidebarRename),
		menuText(LocaleEN, keyHeaderMenuOpenInNewWindow),
		menuText(LocaleEN, keyHeaderMenuExport),
		menuText(LocaleEN, keySidebarArchive),
		menuText(LocaleEN, keyHeaderMenuCopyLink),
		menuText(LocaleEN, keyHeaderMenuCopyID),
		menuText(LocaleEN, keyHeaderMenuCopyDebug),
		"-",
		tr(LocaleEN, keySidebarKeys),
		tr(LocaleEN, keySidebarEndBranch),
		tr(LocaleEN, keyHeaderMenuRefresh),
		"-",
		tr(LocaleEN, keySidebarDelete),
		tr(LocaleEN, keyHeaderSettings),
	}
	if got := tt.Menu(); !slices.Equal(got, want) {
		t.Errorf("the session menu draws %q, want %q", got, want)
	}
	// The rule keeps every row, usable or not, on the reference's 28 DIP step.
	for _, label := range want {
		if label == "-" {
			continue
		}
		info, ok := tt.AccessByRole(ui.RoleMenuItem, label)
		if !ok {
			t.Fatalf("the menu has no %q item, texts %q", label, tt.Texts())
		}
		if math.Abs(float64(info.Bounds.H-menuRowH)) > 0.5 {
			t.Errorf("%q is %.1f high, want the reference's %v DIP row", label, info.Bounds.H, menuRowH)
		}
	}
}

// TestSessionMenuClosesOnAnOutsideClick checks that pressing outside the menu
// closes it without choosing an entry: no callback runs and the menu's items
// leave the frame.
func TestSessionMenuClosesOnAnOutsideClick(t *testing.T) {
	s := fixture()
	refreshes := 0
	s.OnRefreshBranches = func() { refreshes++ }
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	if len(tt.Menu()) == 0 {
		t.Fatal("the menu did not open")
	}
	tt.ClickAt(20, 560)
	if got := tt.Menu(); len(got) != 0 {
		t.Errorf("an outside click left the menu open: %q", got)
	}
	if refreshes != 0 {
		t.Errorf("an outside click chose Refresh branches %d times", refreshes)
	}
}

// TestSessionMenuIsKeyboardReachable checks the menu's keys: Escape closes it
// and returns the focus to the button, Enter on the focused button opens it
// again, and Down moves the highlight onto the first usable entry, where Enter
// chooses it and closes the menu.
func TestSessionMenuIsKeyboardReachable(t *testing.T) {
	s := fixture()
	refreshes := 0
	s.OnRefreshBranches = func() { refreshes++ }
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if got := tt.Menu(); len(got) != 0 {
		t.Fatalf("Escape left the menu open: %q", got)
	}
	if !tt.Focused(tr(LocaleEN, keyHeaderMenu)) {
		t.Fatalf("Escape did not return the focus to the session menu button, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if got := tt.Menu(); len(got) == 0 {
		t.Fatal("Enter on the focused session menu button did not open the menu")
	}
	// Walk the highlight with Down until it reaches the entry the shell can
	// run: the drawn menu may open on its first or last item, so the test
	// follows the keys instead of assuming one step.
	refreshLabel := tr(LocaleEN, keyHeaderMenuRefresh)
	found := false
	for range 12 {
		tt.Key(0, ui.KeyDown)
		info, ok := tt.AccessByRole(ui.RoleMenuItem, refreshLabel)
		if ok && info.Focused {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Down did not reach Refresh branches, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if refreshes != 1 {
		t.Errorf("Enter on the highlighted entry ran Refresh branches %d times, want 1", refreshes)
	}
	if got := tt.Menu(); len(got) != 0 {
		t.Errorf("choosing an entry left the menu open: %q", got)
	}
}

// TestHeaderDrawsNoEditorEntry checks the bar's trailing group: the reference's
// VS Code mark, its disclosure arrow and the rule before the panel toggle are
// gone, because the shell has no editor client or workspace fact to open
// anything with. The panel toggle stays; it is a real entry.
func TestHeaderDrawsNoEditorEntry(t *testing.T) {
	v := fixture().headerView()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		header(c, newTokens(c.Theme()), v, Actions{OnToggleContextPanel: func() {}})
	}, 900, 200)
	k := newTokens(theme)

	if _, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyHeaderOpenInEditor)); ok {
		t.Error("the bar still exposes the editor-open entry")
	}
	if tt.HasText(tr(LocaleEN, keyHeaderOpenInEditor)) {
		t.Errorf("the bar still draws the editor-open label, texts %q", tt.Texts())
	}
	toggle, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyContextPanelToggle))
	if !ok {
		t.Fatalf("the bar lost the panel toggle, texts %q", tt.Texts())
	}
	// The divider went with the editor entry: the room before the toggle is the
	// bar's own face, not a vertical rule.
	image := tt.Image()
	y := int(toggle.Bounds.Y + toggle.Bounds.H/2)
	for x := int(toggle.Bounds.X) - 12; x < int(toggle.Bounds.X); x++ {
		if got := image.RGBAAt(x, y); !sameColor(got, k.background) {
			t.Errorf("the bar paints %v at x=%d, before the panel toggle: the divider is still drawn", got, x)
			break
		}
	}
}

// TestHeaderDrawsNoModelEntry checks the reference's bar: the model picker is
// the composer's entry, not the toolbar's, so the bar draws no "Model ▾" and no
// model picker button.
func TestHeaderDrawsNoModelEntry(t *testing.T) {
	v := fixture().headerView()
	tt := ui.NewTester(func(c *ui.Context) {
		header(c, newTokens(c.Theme()), v, Actions{OnToggleContextPanel: func() {}})
	}, 800, 41)

	for _, gone := range []string{"Model", "Model picker"} {
		if tt.HasText(gone) {
			t.Errorf("the bar still draws %q, texts %q", gone, tt.Texts())
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Model picker"); ok {
		t.Error("the bar still exposes a model picker entry")
	}
	// The panel toggle stays; only the model entry left the bar.
	if _, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyContextPanelToggle)); !ok {
		t.Errorf("the bar lost the panel toggle, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyHeaderOpenInEditor)); ok {
		t.Error("the bar still exposes the editor entry")
	}
}

// TestHeaderTitleUsesTheRoomBeforeTheMenu checks where a long title stops in
// the 537 DIP bar the reference's card leaves at an 800×720 window. The title's
// box ends at the session menu's target, but its text may run
// titleEllipsisOverlap DIPs under the target's empty left bearing, so the
// truncation reaches the box's edge without moving the menu or any control to
// its right: the title's ink reaches its box's edge and stops before the menu's
// "…". The title is longer than the bar so the check still exercises the cut
// now that the editor entry no longer takes room from the breadcrumb.
func TestHeaderTitleUsesTheRoomBeforeTheMenu(t *testing.T) {
	v := fixture().headerView()
	v.Project = "dimagent-mygo"
	v.Title = "Explain what this repository does in one paragraph, then keep going until the bar cuts it."
	v.Connection = ConnReady
	v.Run = RunIdle
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		header(c, newTokens(c.Theme()), v, Actions{OnToggleContextPanel: func() {}})
	}, 537, 41)
	k := newTokens(theme)

	title, ok := tt.Find(v.Title)
	if !ok {
		t.Fatalf("the title is not drawn, texts %q", tt.Texts())
	}
	menu, ok := tt.AccessByRole(ui.RoleMenuButton, tr(LocaleEN, keyHeaderMenu))
	if !ok {
		t.Fatalf("the session menu is not drawn, texts %q", tt.Texts())
	}
	glyph, ok := tt.Find("…")
	if !ok {
		t.Fatalf("the menu glyph is not drawn, texts %q", tt.Texts())
	}
	if math.Abs(float64(menu.Bounds.X-(title.X+title.W))) > 0.5 {
		t.Errorf("the menu target starts at %.1f, want it at the title's box edge %.1f",
			menu.Bounds.X, title.X+title.W)
	}
	// Read the box plus the elision room: a title that stopped at its box would
	// leave the last few DIPs before the menu empty.
	area := ui.Rect{X: title.X, Y: title.Y, W: title.W + titleEllipsisOverlap, H: title.H}
	ink, pixels := glyphInkBox(tt, area, k.background)
	if pixels == 0 {
		t.Fatal("the title draws no ink")
	}
	if right := ink.X + ink.W; right < title.X+title.W-1 {
		t.Errorf("the title's ink stops at %.1f, %.1f DIPs before its box ends at %.1f; the elision room is unused",
			right, title.X+title.W-right, title.X+title.W)
	}
	if ink.X+ink.W > glyph.X {
		t.Errorf("the title's ink ends at %.1f, running into the menu glyph at %.1f", ink.X+ink.W, glyph.X)
	}
}

// TestHeaderPanelToggleDrawsTheOutlinedTwoPaneMark checks the toggle's mark
// against window-conversation.png: a rounded frame split into two panes, both
// pane interiors left empty and only the strokes painted. The mark is Lucide's
// panel-right, whose 18-unit frame and 2-unit stroke span 20 of 24 units, so it
// covers about 13.3 DIPs at the 16 DIP step and rasterizes to 14.
func TestHeaderPanelToggleDrawsTheOutlinedTwoPaneMark(t *testing.T) {
	v := fixture().headerView()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		header(c, newTokens(c.Theme()), v, Actions{OnToggleContextPanel: func() {}})
	}, 900, 41)
	k := newTokens(theme)

	entry, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyContextPanelToggle))
	if !ok {
		t.Fatalf("the context panel toggle is not drawn, texts %q", tt.Texts())
	}
	box := ui.Rect{
		X: entry.Bounds.X + (entry.Bounds.W-sidebarIconSize)/2,
		Y: entry.Bounds.Y + (entry.Bounds.H-sidebarIconSize)/2,
		W: sidebarIconSize, H: sidebarIconSize,
	}
	ink, pixels := glyphInkBox(tt, box, k.background)
	if pixels == 0 {
		t.Fatal("the panel toggle draws no mark")
	}
	if ink.W < 13 || ink.W > 14 || ink.H < 13 || ink.H > 14 {
		t.Errorf("the panel mark's ink is %.1fx%.1f, want the Lucide frame's 13–14 DIPs", ink.W, ink.H)
	}
	image := tt.Image()
	at := func(dx, dy float32) color.RGBA {
		return image.RGBAAt(int(box.X+dx), int(box.Y+dy))
	}
	if left := at(4, 8); !sameColor(left, k.background) {
		t.Errorf("the left pane's centre is %v, want the bar's face", left)
	}
	if right := at(12, 8); !sameColor(right, k.background) {
		t.Errorf("the right pane's centre is %v, want the bar's face and no fill", right)
	}
	if border := at(8, 2); sameColor(border, k.background) {
		t.Error("the frame's top edge draws no stroke")
	}
	if divider := at(10, 8); sameColor(divider, k.background) {
		t.Error("the mark has no centre divider")
	}
}

// TestHeaderBreadcrumbStartsAtTheReferenceInset checks the bar's left edge and
// the room before its menu against the reference's 800×720 frame: the project
// starts where the reference's chat header starts (px-3 plus the breadcrumb
// row's mx-3, x≈277.5 in the window, 19 DIPs inside the shell's frame), and the
// menu's target begins where the truncated title's box ends, so "…" sits about
// 13 DIPs after the last glyph instead of at the far end of an elastic
// breadcrumb.
func TestHeaderBreadcrumbStartsAtTheReferenceInset(t *testing.T) {
	v := fixture().headerView()
	v.Project = "dimagent-mygo"
	v.Title = "Explain what this repository does in one paragraph."
	v.Connection = ConnReady
	v.Run = RunIdle
	tt := ui.NewTester(func(c *ui.Context) {
		header(c, newTokens(c.Theme()), v, Actions{OnToggleContextPanel: func() {}})
	}, 537, 41)

	project, ok := tt.Find(v.Project)
	if !ok {
		t.Fatalf("the project is not drawn, texts %q", tt.Texts())
	}
	if math.Abs(float64(project.X-breadcrumbLeftPad)) > 1 {
		t.Errorf("the breadcrumb starts at x=%.1f, want the reference's %v DIP inset",
			project.X, breadcrumbLeftPad)
	}
	title, ok := tt.Find(v.Title)
	if !ok {
		t.Fatalf("the title is not drawn, texts %q", tt.Texts())
	}
	menu, ok := tt.Find("…")
	if !ok {
		t.Fatalf("the menu glyph is not drawn, texts %q", tt.Texts())
	}
	if gap := menu.X - (title.X + title.W); gap < 0 || gap > 8 {
		t.Errorf("the menu glyph sits %.1f after the title's box, want the menu to hug the title", gap)
	}
	button, ok := tt.AccessByRole(ui.RoleMenuButton, tr(LocaleEN, keyHeaderMenu))
	if !ok {
		t.Fatalf("the menu target is not drawn, texts %q", tt.Texts())
	}
	if gap := button.Bounds.X - (title.X + title.W); math.Abs(float64(gap)) > 0.5 {
		t.Errorf("the menu target sits %.1f after the title, want it at the title's edge", gap)
	}
	if button.Bounds.W > menuTargetW+0.5 {
		t.Errorf("the menu target is %.1f wide, want the compact %v DIP step", button.Bounds.W, menuTargetW)
	}
}

// TestHeaderTruncatesTheProjectAtTheReferenceWidth checks the breadcrumb's
// project name against window-conversation.png: the reference's workspace span
// is max-w-[8rem] and shrinks inside the bar's flex row, so in an 800 DIP
// window it leaves the project about 85 DIPs, draws "dimagent-m…" and starts
// the title's ink at x≈375. The bar keeps that measured width with its
// ellipsis, so the title starts where the reference draws it instead of being
// pushed right by the whole project name; the slash stays between the two and
// an empty project still draws no slash.
func TestHeaderTruncatesTheProjectAtTheReferenceWidth(t *testing.T) {
	v := fixture().headerView()
	v.Project = "dimagent-mygo"
	v.Title = "Explain what this repository does in one paragraph."
	tt := headerTestView(v, 800, 41)
	k := newTokens(ui.LightTheme())

	project, ok := tt.Find(v.Project)
	if !ok {
		t.Fatalf("the project name is not drawn, texts %q", tt.Texts())
	}
	if math.Abs(float64(project.W-breadcrumbProjectMaxW)) > 0.5 {
		t.Errorf("the project box is %.1f wide, want the reference's %v DIPs", project.W, breadcrumbProjectMaxW)
	}
	// The 85 DIP cap leaves the reference's one extra glyph: the drawn ink runs
	// past the old 83 DIP cap's "dimagent-…" (≈70 DIPs at this type step) to the
	// reference's "dimagent-m…", and still stops short of the whole name (≈96).
	ink, pixels := glyphInkBox(tt, project, k.background)
	if pixels == 0 {
		t.Fatal("the project name draws no ink")
	}
	if ink.W < 78 || ink.W > project.W {
		t.Errorf("the project's ink is %.1f wide in a %.1f box, want the reference's truncated name", ink.W, project.W)
	}
	title, ok := tt.Find(v.Title)
	if !ok {
		t.Fatalf("the title is not drawn, texts %q", tt.Texts())
	}
	// The reference's card starts at x=256.5, so its title ink at x≈375 is
	// 118.5 DIPs inside the card. This bar is measured from the shell's own
	// frame edge, about 1.5 DIPs right of the reference card's, so the same ink
	// lands at ≈117 here; the ±3 covers that offset and the glyph's bearing.
	titleInk, titlePixels := glyphInkBox(tt, title, k.background)
	if titlePixels == 0 {
		t.Fatal("the title draws no ink")
	}
	if math.Abs(float64(titleInk.X-118.5)) > 3 {
		t.Errorf("the title's ink starts at x=%.1f, want the reference's 118.5 ±3", titleInk.X)
	}
	if slash, ok := tt.Find(breadcrumbSeparator); !ok || slash.X <= project.X+project.W {
		t.Errorf("the separator at x=%.1f does not follow the truncated project ending at %.1f",
			slash.X, project.X+project.W)
	}
}

// TestSessionMenuTogglesPinAndArchive checks the two desktop-local entries: with
// the window's callbacks wired they are usable, each click reaches the callback
// exactly once with the selected id, and the label follows the state the next
// frame reports ("Pin"/"Unpin", "Archive"/"Unarchive"). Nothing here is a
// session fact; the App Server is not addressed at all.
func TestSessionMenuTogglesPinAndArchive(t *testing.T) {
	s := fixture()
	pins, archives := 0, 0
	s.OnPinSession = func(sessionID string) {
		if sessionID != "s1" {
			t.Errorf("pin got %q, want s1", sessionID)
		}
		pins++
		if len(s.Pinned) == 0 {
			s.Pinned = []string{sessionID}
		} else {
			s.Pinned = nil
		}
	}
	s.OnArchiveSession = func(sessionID string) {
		if sessionID != "s1" {
			t.Errorf("archive got %q, want s1", sessionID)
		}
		archives++
		if len(s.Archived) == 0 {
			s.Archived = []string{sessionID}
		} else {
			s.Archived = nil
		}
	}
	tt := ui.NewTester(s.View, 900, 600)
	openMenu := func() {
		t.Helper()
		if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
			t.Fatalf("open the session menu: %v", err)
		}
	}
	item := func(label string) ui.AccessInfo {
		t.Helper()
		info, ok := tt.AccessByRole(ui.RoleMenuItem, label)
		if !ok {
			t.Fatalf("the menu has no %q item, texts %q", label, tt.Texts())
		}
		if info.Disabled {
			t.Errorf("%q is disabled although its callback is wired", label)
		}
		return info
	}

	openMenu()
	item(tr(LocaleEN, keySidebarPin))
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarPin))
	if pins != 1 || len(s.Pinned) != 1 {
		t.Fatalf("Pin ran %d time(s), pinned=%v, want one pin", pins, s.Pinned)
	}

	openMenu()
	item(tr(LocaleEN, keySidebarUnpin))
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarUnpin))
	if pins != 2 || len(s.Pinned) != 0 {
		t.Fatalf("Unpin ran %d time(s) total, pinned=%v, want the second toggle", pins, s.Pinned)
	}

	openMenu()
	item(tr(LocaleEN, keySidebarArchive))
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarArchive))
	if archives != 1 || len(s.Archived) != 1 {
		t.Fatalf("Archive ran %d time(s), archived=%v, want one archive", archives, s.Archived)
	}

	openMenu()
	item(tr(LocaleEN, keySidebarUnarchive))
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarUnarchive))
	if archives != 2 || len(s.Archived) != 0 {
		t.Fatalf("Unarchive ran %d time(s) total, archived=%v, want the second toggle", archives, s.Archived)
	}
}

// TestSessionMenuCopiesTheConversationID checks the menu's one local entry end
// to end: Copy conversation ID is usable while a session identity is selected,
// a click puts exactly that id on the system clipboard, the menu closes, and
// the banner shows the muted "Copied conversation ID" outcome. The id is a fact
// the shell already displays, so no App Server request is behind it.
func TestSessionMenuCopiesTheConversationID(t *testing.T) {
	s := fixture()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 600)
	if got := tt.Clipboard(); got != "" {
		t.Fatalf("the clipboard starts with %q", got)
	}

	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	label := menuText(LocaleEN, keyHeaderMenuCopyID)
	info, ok := tt.AccessByRole(ui.RoleMenuItem, label)
	if !ok {
		t.Fatalf("the menu has no %q item, texts %q", label, tt.Texts())
	}
	if info.Disabled {
		t.Fatalf("%q is disabled although a session identity is selected", label)
	}
	clickMenuItem(t, tt, label)
	if got, want := tt.Clipboard(), s.Selected; got != want {
		t.Errorf("copying put %q on the clipboard, want the selected session's id %q", got, want)
	}
	tt.Frame()
	if s.ActionStatus != copyIDStatus {
		t.Errorf("the copy recorded %q as the action status, want %q", s.ActionStatus, copyIDStatus)
	}
	if !tt.HasText(copyIDStatus) {
		t.Errorf("the banner does not say %q, texts %q", copyIDStatus, tt.Texts())
	}
	k := newTokens(theme)
	status := textBox(t, tt, copyIDStatus)
	if _, muted := colorBox(tt, status, k.mutedForeground); muted == 0 {
		t.Error("the copy notice is not drawn muted")
	}
	if _, danger := colorBox(tt, status, k.danger); danger != 0 {
		t.Error("the copy notice is drawn as a failure")
	}
	if got := tt.Menu(); len(got) != 0 {
		t.Errorf("choosing the entry left the menu open: %q", got)
	}
}

// TestSessionMenuCopiesWithoutAWindowCallback checks the region on its own: the
// clipboard write needs no window callback, so the entry copies the id even
// when Actions carries no copy action, and the missing status line is the only
// difference.
func TestSessionMenuCopiesWithoutAWindowCallback(t *testing.T) {
	v := fixture().headerView()
	tt := headerTestView(v, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	clickMenuItem(t, tt, menuText(LocaleEN, keyHeaderMenuCopyID))
	if got, want := tt.Clipboard(), v.SessionID; got != want {
		t.Errorf("copying put %q on the clipboard, want the selected session's id %q", got, want)
	}
	if tt.HasText(copyIDStatus) {
		t.Errorf("a region without copy callbacks drew the outcome %q", copyIDStatus)
	}
}

// TestSessionMenuCopyIDWithoutAnIdentityStaysWIP checks the entry's visible
// state with no selected identity: there is no id to copy, so the item keeps
// its WIP mark and reason and a click writes nothing.
func TestSessionMenuCopyIDWithoutAnIdentityStaysWIP(t *testing.T) {
	v := fixture().headerView()
	v.SessionID = ""
	tt := headerTestView(v, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	label := menuText(LocaleEN, keyHeaderMenuCopyID)
	info, ok := tt.AccessByRole(ui.RoleMenuItem, label)
	if !ok {
		t.Fatalf("the menu has no %q item, texts %q", label, tt.Texts())
	}
	if !info.Disabled {
		t.Errorf("%q is available without a session identity", label)
	}
	if reason := menuText(LocaleEN, keyReasonCopyID); !strings.Contains(info.Description, reason) {
		t.Errorf("%q description %q does not name %q", label, info.Description, reason)
	}
	clickMenuItem(t, tt, label)
	if got := tt.Clipboard(); got != "" {
		t.Errorf("clicking the WIP entry put %q on the clipboard", got)
	}
}

// TestConversationLinkIsTheDeepLinkTheAppRegisters checks the text the menu
// copies: the app's registered scheme, the thread route and the selected id as
// one path segment. The link is built from the id alone, so it carries no
// session fact of its own.
func TestConversationLinkIsTheDeepLinkTheAppRegisters(t *testing.T) {
	const id = "8a1e5b6c-7d8f-4a90-b1c2-d3e4f5061728"
	want := ConversationLinkScheme + "://" + ConversationLinkHost + "/" + id
	if got := ConversationLink(id); got != want {
		t.Errorf("ConversationLink(%q) = %q, want %q", id, got, want)
	}
	// An id that is not a plain token stays one path segment, so the link the
	// clipboard holds is still one URL.
	if got, want := ConversationLink("a/b"), "dimagent://thread/a%2Fb"; got != want {
		t.Errorf("ConversationLink(%q) = %q, want %q", "a/b", got, want)
	}
}

// TestSessionMenuCopiesTheConversationLink checks the menu's link entry end to
// end: Copy conversation link is usable while a session identity is selected, a
// click puts exactly dimagent://thread/<id> on the system clipboard, the menu
// closes, and the banner shows the muted "Copied conversation link" outcome.
// The link is built from the id the shell already displays, so no App Server
// request is behind it.
func TestSessionMenuCopiesTheConversationLink(t *testing.T) {
	s := fixture()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 600)
	if got := tt.Clipboard(); got != "" {
		t.Fatalf("the clipboard starts with %q", got)
	}

	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	label := menuText(LocaleEN, keyHeaderMenuCopyLink)
	info, ok := tt.AccessByRole(ui.RoleMenuItem, label)
	if !ok {
		t.Fatalf("the menu has no %q item, texts %q", label, tt.Texts())
	}
	if info.Disabled {
		t.Fatalf("%q is disabled although a session identity is selected", label)
	}
	clickMenuItem(t, tt, label)
	if got, want := tt.Clipboard(), "dimagent://thread/"+s.Selected; got != want {
		t.Errorf("copying put %q on the clipboard, want %q", got, want)
	}
	tt.Frame()
	if s.ActionStatus != copyLinkStatus {
		t.Errorf("the copy recorded %q as the action status, want %q", s.ActionStatus, copyLinkStatus)
	}
	if !tt.HasText(copyLinkStatus) {
		t.Errorf("the banner does not say %q, texts %q", copyLinkStatus, tt.Texts())
	}
	k := newTokens(theme)
	status := textBox(t, tt, copyLinkStatus)
	if _, muted := colorBox(tt, status, k.mutedForeground); muted == 0 {
		t.Error("the copy notice is not drawn muted")
	}
	if _, danger := colorBox(tt, status, k.danger); danger != 0 {
		t.Error("the copy notice is drawn as a failure")
	}
	if got := tt.Menu(); len(got) != 0 {
		t.Errorf("choosing the entry left the menu open: %q", got)
	}
}

// TestSessionMenuCopiesTheConversationLinkWithoutAWindowCallback checks the
// region on its own: the clipboard write needs no window callback, so the entry
// copies the link even when Actions carries no copy action, and the missing
// status line is the only difference.
func TestSessionMenuCopiesTheConversationLinkWithoutAWindowCallback(t *testing.T) {
	v := fixture().headerView()
	tt := headerTestView(v, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	clickMenuItem(t, tt, menuText(LocaleEN, keyHeaderMenuCopyLink))
	if got, want := tt.Clipboard(), ConversationLink(v.SessionID); got != want {
		t.Errorf("copying put %q on the clipboard, want %q", got, want)
	}
	if tt.HasText(copyLinkStatus) {
		t.Errorf("a region without copy callbacks drew the outcome %q", copyLinkStatus)
	}
}

// TestSessionMenuCopyLinkWithoutAnIdentityStaysWIP checks the entry's visible
// state with no selected identity: there is no id to build a link from, so the
// item keeps its WIP mark and reason and a click writes nothing.
func TestSessionMenuCopyLinkWithoutAnIdentityStaysWIP(t *testing.T) {
	v := fixture().headerView()
	v.SessionID = ""
	tt := headerTestView(v, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	label := menuText(LocaleEN, keyHeaderMenuCopyLink)
	info, ok := tt.AccessByRole(ui.RoleMenuItem, label)
	if !ok {
		t.Fatalf("the menu has no %q item, texts %q", label, tt.Texts())
	}
	if !info.Disabled {
		t.Errorf("%q is available without a session identity", label)
	}
	if reason := menuText(LocaleEN, keyReasonCopyLink); !strings.Contains(info.Description, reason) {
		t.Errorf("%q description %q does not name %q", label, info.Description, reason)
	}
	clickMenuItem(t, tt, label)
	if got := tt.Clipboard(); got != "" {
		t.Errorf("clicking the WIP entry put %q on the clipboard", got)
	}
}

// TestHeaderDebugInfoDescribesTheCurrentModel checks the text the menu copies:
// the reference's title and its public model rows, one per line, with the pair's
// name and id, the declared limits, the declared capabilities and the catalog's
// source. The shell builds it from the facts it already displays, so no App
// Server read is behind it.
func TestHeaderDebugInfoDescribesTheCurrentModel(t *testing.T) {
	s := modelFixtureState()
	contextWindow, maxOutput := int64(200000), int64(8192)
	current := &s.ModelCatalog.Choices[0]
	current.ContextWindow = &contextWindow
	current.MaxOutputTokens = &maxOutput
	current.ToolCalling = true
	current.Vision = false
	current.Reasoning = true
	current.Source = "default"

	v := s.headerView()
	lines := strings.Split(v.DebugInfo, "\n")
	want := []string{
		"Model runtime config",
		"Provider  Anthropic (anthropic)",
		"Model     Claude 3 (claude-3)",
		"Context   200000 / output 8192",
		"Caps      tools yes, vision no, reasoning yes",
		"Source    default",
	}
	if !slices.Equal(lines, want) {
		t.Errorf("the debug info reads %q, want %q", lines, want)
	}
}

// TestHeaderDebugInfoWritesNForUnknownFacts checks that a field the window does
// not hold is written n/a instead of being guessed: a current pair with no
// catalog row keeps the names n/a, a nil limit is n/a, and a row that declared
// no capability reads "no" rather than n/a.
func TestHeaderDebugInfoWritesNForUnknownFacts(t *testing.T) {
	t.Run("pair without a catalog row", func(t *testing.T) {
		s := fixture()
		s.ModelCatalog = ModelCatalog{
			Current:           "anthropic/claude-3",
			HasCurrent:        true,
			CurrentProviderID: "anthropic",
			CurrentModelID:    "claude-3",
		}
		v := s.headerView()
		for _, want := range []string{
			"Provider  anthropic",
			"Model     claude-3",
			"Context   n/a / output n/a",
			"Caps      tools n/a, vision n/a, reasoning n/a",
			"Source    n/a",
		} {
			if !strings.Contains(v.DebugInfo, want) {
				t.Errorf("the debug info is missing %q, text %q", want, v.DebugInfo)
			}
		}
	})
	t.Run("row without limits or source", func(t *testing.T) {
		s := fixture()
		s.ModelCatalog = ModelCatalog{
			Current:           "anthropic/claude-3",
			HasCurrent:        true,
			CurrentProviderID: "anthropic",
			CurrentModelID:    "claude-3",
			Choices: []ModelChoice{{
				ProviderID:   "anthropic",
				ProviderName: "Anthropic",
				ModelID:      "claude-3",
				DisplayName:  "Claude 3",
				Current:      true,
			}},
		}
		v := s.headerView()
		for _, want := range []string{
			"Context   n/a / output n/a",
			"Caps      tools no, vision no, reasoning no",
			"Source    n/a",
		} {
			if !strings.Contains(v.DebugInfo, want) {
				t.Errorf("the debug info is missing %q, text %q", want, v.DebugInfo)
			}
		}
	})
}

// TestSessionMenuCopiesTheDebugInfo checks the entry end to end: it is usable
// while the window holds a current model pair, a click puts exactly that text
// on the clipboard, the menu closes, and the banner shows the muted "Copied
// debug info" outcome. The text is built from facts the shell already displays,
// so no App Server request is behind the click.
func TestSessionMenuCopiesTheDebugInfo(t *testing.T) {
	s := modelFixtureState()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 600)
	want := s.headerView().DebugInfo
	if want == "" {
		t.Fatal("the model fixture produced no debug info")
	}

	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	label := menuText(LocaleEN, keyHeaderMenuCopyDebug)
	info, ok := tt.AccessByRole(ui.RoleMenuItem, label)
	if !ok {
		t.Fatalf("the menu has no %q item, texts %q", label, tt.Texts())
	}
	if info.Disabled {
		t.Fatalf("%q is disabled although the window holds model facts", label)
	}
	clickMenuItem(t, tt, label)
	if got := tt.Clipboard(); got != want {
		t.Errorf("copying put %q on the clipboard, want %q", got, want)
	}
	tt.Frame()
	if s.ActionStatus != copyDebugStatus {
		t.Errorf("the copy recorded %q as the action status, want %q", s.ActionStatus, copyDebugStatus)
	}
	if !tt.HasText(copyDebugStatus) {
		t.Errorf("the banner does not say %q, texts %q", copyDebugStatus, tt.Texts())
	}
	k := newTokens(theme)
	status := textBox(t, tt, copyDebugStatus)
	if _, muted := colorBox(tt, status, k.mutedForeground); muted == 0 {
		t.Error("the copy notice is not drawn muted")
	}
	if _, danger := colorBox(tt, status, k.danger); danger != 0 {
		t.Error("the copy notice is drawn as a failure")
	}
	if got := tt.Menu(); len(got) != 0 {
		t.Errorf("choosing the entry left the menu open: %q", got)
	}
}

// TestSessionMenuCopiesDebugInfoOnce checks the region on its own: one click
// calls the outcome callback exactly once and writes the whole text, and the
// write needs no window callback, so a region without one still copies.
func TestSessionMenuCopiesDebugInfoOnce(t *testing.T) {
	v := modelFixtureState().headerView()
	copies := 0
	tt := ui.NewTester(func(c *ui.Context) {
		header(c, newTokens(c.Theme()), v, Actions{OnCopyDebugInfo: func() { copies++ }})
	}, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	clickMenuItem(t, tt, menuText(LocaleEN, keyHeaderMenuCopyDebug))
	if copies != 1 {
		t.Errorf("one click ran the copy outcome %d time(s), want 1", copies)
	}
	if got := tt.Clipboard(); got != v.DebugInfo {
		t.Errorf("copying put %q on the clipboard, want %q", got, v.DebugInfo)
	}
	if tt.HasText(copyDebugStatus) {
		t.Errorf("a region without the shell drew the outcome %q", copyDebugStatus)
	}

	bare := modelFixtureState().headerView()
	bareTT := headerTestView(bare, 900, 600)
	if err := bareTT.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	clickMenuItem(t, bareTT, menuText(LocaleEN, keyHeaderMenuCopyDebug))
	if got := bareTT.Clipboard(); got != bare.DebugInfo {
		t.Errorf("a region without callbacks put %q on the clipboard, want %q", got, bare.DebugInfo)
	}
}

// TestSessionMenuCopyDebugWithoutModelFactsStaysWIP checks the entry's empty
// state: with no current model pair there is no runtime config to copy, so the
// item keeps its WIP mark and reason and a click writes nothing.
func TestSessionMenuCopyDebugWithoutModelFactsStaysWIP(t *testing.T) {
	v := fixture().headerView()
	if v.DebugInfo != "" {
		t.Fatalf("the fixture built debug info without model facts: %q", v.DebugInfo)
	}
	tt := headerTestView(v, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	label := menuText(LocaleEN, keyHeaderMenuCopyDebug)
	info, ok := tt.AccessByRole(ui.RoleMenuItem, label)
	if !ok {
		t.Fatalf("the menu has no %q item, texts %q", label, tt.Texts())
	}
	if !info.Disabled {
		t.Errorf("%q is available without model facts", label)
	}
	if reason := menuText(LocaleEN, keyReasonCopyDebug); !strings.Contains(info.Description, reason) {
		t.Errorf("%q description %q does not name %q", label, info.Description, reason)
	}
	clickMenuItem(t, tt, label)
	if got := tt.Clipboard(); got != "" {
		t.Errorf("clicking the WIP entry put %q on the clipboard", got)
	}
}
