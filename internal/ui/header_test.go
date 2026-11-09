package ui

import (
	"image/color"
	"math"
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
		t.Fatal("the settings entry did not open the appearance surface")
	}
	openMenu()
	clickMenuItem(t, tt, tr(LocaleEN, keyHeaderConversation))
	if s.ShowSettings {
		t.Fatal("the conversation entry did not return from the appearance surface")
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
// the method it waits on, while the desktop-local settings entry stays usable.
func TestSessionMenuMarksMissingCallbacksAsWIP(t *testing.T) {
	s := fixture()
	s.SelectedBranch = "side"
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ label, reason string }{
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
	}
	if settings, ok := tt.AccessByRole(ui.RoleMenuItem, tr(LocaleEN, keyHeaderSettings)); !ok || settings.Disabled {
		t.Errorf("the desktop-local settings entry is missing or disabled: %+v", settings)
	}
}

// TestHeaderOffersTheEditorOpenEntryAsWIP checks the reference's editor-open
// control: a VS Code mark and its disclosure arrow stay visible at the bar's
// right end, before the divider and the panel toggle, but take no click and no
// Tab stop while the shell has no editor client or workspace fact.
func TestHeaderOffersTheEditorOpenEntryAsWIP(t *testing.T) {
	v := fixture().headerView()
	tt := ui.NewTester(func(c *ui.Context) {
		header(c, newTokens(c.Theme()), v, Actions{OnToggleContextPanel: func() {}})
	}, 900, 200)
	label := tr(LocaleEN, keyHeaderOpenInEditor)
	info, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatalf("the editor-open entry is not drawn, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("the editor-open entry is available although the shell has no editor client")
	}
	if info.Focusable {
		t.Error("the editor-open entry takes a Tab stop")
	}
	if !strings.Contains(info.Description, tr(LocaleEN, keyReasonOpenInEditor)) {
		t.Errorf("the entry description %q does not name the reason %q",
			info.Description, tr(LocaleEN, keyReasonOpenInEditor))
	}
	entry, ok := tt.Find(label)
	if !ok {
		t.Fatalf("the editor-open entry has no drawn box, texts %q", tt.Texts())
	}
	if entry.H < newTokens(ui.LightTheme()).sidebarEntryH-0.5 {
		t.Errorf("the entry is %.1f high, want the toolbar's 28 DIP control", entry.H)
	}
	if want := float32(2*openInEditorCell) + 2*newTokens(ui.LightTheme()).hairline; entry.W > want+0.5 {
		t.Errorf("the entry is %.1f wide, want the reference's %v DIP frame", entry.W, want)
	}
	toggle, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyContextPanelToggle))
	if !ok {
		t.Fatalf("the context panel toggle is not drawn, texts %q", tt.Texts())
	}
	if toggle.Bounds.X < entry.X+entry.W-0.5 {
		t.Errorf("the context panel toggle at x=%.1f is not after the editor entry ending at %.1f",
			toggle.Bounds.X, entry.X+entry.W)
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
	// The editor entry and the panel toggle stay; only the model entry left the bar.
	if _, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyHeaderOpenInEditor)); !ok {
		t.Errorf("the bar lost the editor entry, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyContextPanelToggle)); !ok {
		t.Errorf("the bar lost the panel toggle, texts %q", tt.Texts())
	}
}

// TestHeaderTitleUsesTheRoomBeforeTheMenu checks where the long title stops in
// the 537 DIP bar the reference's card leaves at an 800×720 window. The title's
// box ends at the session menu's target, but its text may run
// titleEllipsisOverlap DIPs under the target's empty left bearing, so the
// truncation lands on the reference's last glyph ("one par…") without moving
// the menu or any control to its right: the title's ink reaches its box's edge
// and stops before the menu's "…".
func TestHeaderTitleUsesTheRoomBeforeTheMenu(t *testing.T) {
	v := fixture().headerView()
	v.Project = "dimagent-mygo"
	v.Title = "Explain what this repository does in one paragraph."
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
// against window-conversation.png: a 10×10 rounded frame split down the middle,
// both panes left empty and only the strokes painted. The reference's mark is
// 12 DIPs of ink at the 16 DIP step; the earlier mark filled the right pane,
// which shows as the toggle's ink at the pane's centre.
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
	if math.Abs(float64(ink.W-12)) > 1 || math.Abs(float64(ink.H-12)) > 1 {
		t.Errorf("the panel mark's ink is %.1fx%.1f, want the reference's 12×12", ink.W, ink.H)
	}
	image := tt.Image()
	at := func(dx, dy float32) color.RGBA {
		return image.RGBAAt(int(box.X+dx), int(box.Y+dy))
	}
	if left := at(5, 8); !sameColor(left, k.background) {
		t.Errorf("the left pane's centre is %v, want the bar's face", left)
	}
	if right := at(11, 8); !sameColor(right, k.background) {
		t.Errorf("the right pane's centre is %v, want the bar's face and no fill", right)
	}
	if border := at(8, 3); sameColor(border, k.background) {
		t.Error("the frame's top edge draws no stroke")
	}
	if divider := at(8, 8); sameColor(divider, k.background) {
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

// TestHeaderEditorMarkUsesTheVSCodeBrandBlue checks the drawn editor entry: the
// VS Code mark keeps the brand's blue even though the WIP entry is unavailable,
// while the disclosure arrow beside it stays the bar's muted ink. The entry is
// unavailable without dimming: it takes no click and no Tab stop, and the mark
// paints the reference's #2563EB exactly instead of a half-opacity tint. The
// frame has no internal rule between the two cells.
func TestHeaderEditorMarkUsesTheVSCodeBrandBlue(t *testing.T) {
	v := fixture().headerView()
	tt := ui.NewTester(func(c *ui.Context) {
		header(c, newTokens(c.Theme()), v, Actions{})
	}, 800, 41)
	label := tr(LocaleEN, keyHeaderOpenInEditor)
	entry, ok := tt.Find(label)
	if !ok {
		t.Fatalf("the editor entry is not drawn, texts %q", tt.Texts())
	}
	info, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatalf("the editor entry is not a named button, texts %q", tt.Texts())
	}
	if !info.Disabled || info.Focusable {
		t.Errorf("the editor entry is not unavailable without a Tab stop: disabled=%v focusable=%v",
			info.Disabled, info.Focusable)
	}
	image := tt.Image()
	brand := color.RGBA{R: openInEditorBrand.R, G: openInEditorBrand.G, B: openInEditorBrand.B, A: openInEditorBrand.A}
	blue, exact := 0, 0
	for y := int(entry.Y) + 2; y < int(entry.Y+entry.H)-2; y++ {
		for x := int(entry.X) + 1; x < int(entry.X+openInEditorCell); x++ {
			p := image.RGBAAt(x, y)
			if d := int(p.B) - int(p.R); d > blue {
				blue = d
			}
			if p == brand {
				exact++
			}
		}
	}
	if exact == 0 {
		t.Errorf("the mark never paints the reference's %v brand blue, strongest pixel is %d bluer than red", brand, blue)
	}
	muted := 0
	for y := int(entry.Y) + 2; y < int(entry.Y+entry.H)-2; y++ {
		for x := int(entry.X+openInEditorCell) + 1; x < int(entry.X+entry.W); x++ {
			p := image.RGBAAt(x, y)
			if d := int(p.B) - int(p.R); d > muted {
				muted = d
			}
		}
	}
	if muted >= blue/2 {
		t.Errorf("the arrow is %d bluer than red, want it muted beside the %d mark", muted, blue)
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
