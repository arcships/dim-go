package ui

import (
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
// rather than pushing it, the model entry or the panel toggle out of the
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
