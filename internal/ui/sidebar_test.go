package ui

import (
	"image/color"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestSidebarGroupsThreads checks the grouped view: rows keep the catalog's
// order inside the group the caller labelled, each heading counts its rows, and
// a session the caller left ungrouped falls under the explicit "No project"
// heading instead of being dropped or given an invented project.
func TestSidebarGroupsThreads(t *testing.T) {
	s := fixture()
	s.Sessions = []Session{
		{ID: "s1", Title: "First session", Group: "alpha"},
		{ID: "s2", Title: "Second session", Group: "beta"},
		{ID: "s3", Title: "Third session", Group: "beta"},
		{ID: "s4", Title: "Fourth session"},
	}
	s.Selected = "s1"
	tt := ui.NewTester(s.View, 900, 600)

	for _, want := range []string{"alpha", "beta", ungroupedLabel, "First session", "Second session", "Third session", "Fourth session"} {
		if !tt.HasText(want) {
			t.Errorf("grouped sidebar is missing %q, texts %q", want, tt.Texts())
		}
	}
	heading, ok := tt.Find("alpha")
	if !ok {
		t.Fatalf("alpha heading is not drawn, texts %q", tt.Texts())
	}
	first, ok := tt.Find("First session")
	if !ok {
		t.Fatalf("first row is not drawn, texts %q", tt.Texts())
	}
	if first.Y < heading.Y+heading.H {
		t.Errorf("row %+v is not under its heading %+v", first, heading)
	}
	second, _ := tt.Find("Second session")
	third, _ := tt.Find("Third session")
	fourth, _ := tt.Find("Fourth session")
	if !(first.Y < second.Y && second.Y < third.Y && third.Y < fourth.Y) {
		t.Errorf("rows are out of catalog order: %+v %+v %+v %+v", first, second, third, fourth)
	}
}

// TestSidebarStaysFlatWithoutGroups checks that the shell draws no headings
// while the catalog carries no group label: it has no project or date fact of
// its own and must not invent the "No project" heading for everyone.
func TestSidebarStaysFlatWithoutGroups(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	if tt.HasText(ungroupedLabel) {
		t.Errorf("an ungrouped catalog draws a %q heading, texts %q", ungroupedLabel, tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleListItem, "First session"); !ok {
		t.Errorf("flat sidebar does not expose its rows, texts %q", tt.Texts())
	}
}

// TestSidebarSelectsASession checks the row's selected state and the choice it
// publishes: the list marks the shell's selection for assistive technology, a
// click on another row moves it, and the header follows.
func TestSidebarSelectsASession(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	first, ok := tt.AccessByRole(ui.RoleListItem, "First session")
	if !ok {
		t.Fatalf("first row is not exposed, texts %q", tt.Texts())
	}
	if !first.Selected {
		t.Error("the selected session is not marked selected")
	}
	second, ok := tt.AccessByRole(ui.RoleListItem, "Second session")
	if !ok {
		t.Fatalf("second row is not exposed, texts %q", tt.Texts())
	}
	if second.Selected {
		t.Error("an unselected session is marked selected")
	}

	clickRow(t, tt, "Second session")
	if s.Selected != "s2" {
		t.Fatalf("selected %q after the click, want s2", s.Selected)
	}
	if info, ok := tt.AccessByRole(ui.RoleListItem, "Second session"); !ok || !info.Selected {
		t.Errorf("the clicked row is not marked selected: %+v, found %v", info, ok)
	}
	if info, ok := tt.AccessByRole(ui.RoleListItem, "First session"); !ok || info.Selected {
		t.Errorf("the first row is still marked selected: %+v, found %v", info, ok)
	}
	if !tt.HasText("Second session") {
		t.Errorf("the header does not follow the choice, texts %q", tt.Texts())
	}
}

// TestSidebarDefaultWidth checks the width the sidebar opens at: 256 DIPs, the
// value docs/ui-layout-spec.md gives, not the theme's spacing unit.
func TestSidebarDefaultWidth(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 1440, 900)

	info, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar")
	if !ok {
		t.Fatalf("the sidebar is not exposed, texts %q", tt.Texts())
	}
	if math.Abs(float64(info.Bounds.W-sidebarDefaultW)) > 0.5 {
		t.Errorf("sidebar width is %.2f, want %d", info.Bounds.W, sidebarDefaultW)
	}
}

// TestSidebarResizeClamps checks the drag handle: a drag inside the bounds
// sets the width, and a drag past either bound leaves the sidebar at 200 or
// 360 DIPs instead of letting it collapse or take the conversation's room.
func TestSidebarResizeClamps(t *testing.T) {
	cases := []struct {
		name string
		dx   float32
		want float32
	}{
		{"inside", 40, sidebarDefaultW + 40},
		{"past the widest", 400, sidebarMaxW},
		{"past the narrowest", -400, sidebarMinW},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := fixture()
			tt := ui.NewTester(s.View, 1440, 900)

			divider, ok := tt.AccessByRole(ui.RoleSplitter, "")
			if !ok {
				t.Fatalf("the split divider is not exposed, texts %q", tt.Texts())
			}
			x := float32(divider.Bounds.X + divider.Bounds.W/2)
			y := float32(divider.Bounds.Y + divider.Bounds.H/2)
			tt.Press(x, y)
			tt.Move(x+tc.dx, y)
			tt.Release(x+tc.dx, y)
			tt.Frame()

			if s.SidebarWidth != tc.want {
				t.Errorf("sidebar width after dragging %+.0f is %.0f, want %.0f", tc.dx, s.SidebarWidth, tc.want)
			}
			info, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar")
			if !ok {
				t.Fatalf("the sidebar is not exposed after the drag, texts %q", tt.Texts())
			}
			if math.Abs(float64(info.Bounds.W-tc.want)) > 0.5 {
				t.Errorf("the drawn sidebar is %.2f wide, want %.0f", info.Bounds.W, tc.want)
			}
		})
	}
}

// TestSidebarWIPEntriesDoNothing checks every sidebar entry the shell cannot
// carry out yet: it keeps its accessible name, is marked unavailable, explains
// the real gap, and a click changes no view state.
func TestSidebarWIPEntriesDoNothing(t *testing.T) {
	s := fixture()
	s.OnCreateSession = nil
	tt := ui.NewTester(s.View, 900, 600)
	width := s.SidebarWidth

	entries := map[string]string{
		"Scheduled tasks":    "scheduled",
		"Skills and plugins": "skills",
		"Sign in to Dim":     "account",
	}
	for name, gap := range entries {
		info, ok := tt.AccessByRole(ui.RoleButton, name)
		if !ok {
			t.Errorf("%q is not exposed as an entry, texts %q", name, tt.Texts())
			continue
		}
		if !info.Disabled {
			t.Errorf("%q is available although the shell cannot carry it out", name)
		}
		if !strings.Contains(info.Description, "Unavailable") {
			t.Errorf("%q description %q does not say it is unavailable", name, info.Description)
		}
		if !strings.Contains(info.Description, gap) {
			t.Errorf("%q description %q does not name %q", name, info.Description, gap)
		}
		if err := tt.Click(name); err != nil {
			t.Fatalf("click %q: %v", name, err)
		}
	}

	if s.Selected != "s1" || s.Draft != "" || len(s.Sessions) != 2 || s.SidebarWidth != width {
		t.Errorf("a WIP click changed the shell: selected=%q draft=%q sessions=%d width=%.0f (was %.0f)",
			s.Selected, s.Draft, len(s.Sessions), s.SidebarWidth, width)
	}
}

// TestSidebarRowFaces checks the row states the spec names: the selected row
// is painted with the sidebar's accent face, and the row under the pointer with
// its hover face. Both are sampled beside the title, in the row's own padding.
func TestSidebarRowFaces(t *testing.T) {
	s := fixture()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 600)
	k := newTokens(theme)

	rowFace := func(name string) color.RGBA {
		rect, ok := tt.Find(name)
		if !ok {
			t.Fatalf("%q is not drawn, texts %q", name, tt.Texts())
		}
		return tt.Image().RGBAAt(int(rect.X)+3, int(rect.Y+rect.H/2))
	}
	selected := rowFace("First session")
	if !sameColor(selected, k.sidebarAccent) {
		t.Errorf("the selected row's face is %v, want the sidebar accent %+v", selected, k.sidebarAccent)
	}

	if hovered := rowFace("Second session"); sameColor(hovered, k.sidebarAccent) {
		t.Fatalf("the unselected row already paints the selected face")
	}
	second, _ := tt.Find("Second session")
	tt.Move(second.X+second.W/2, second.Y+second.H/2)
	tt.Frame()
	hovered := rowFace("Second session")
	if !sameColor(hovered, k.sidebarHover) {
		t.Errorf("the hovered row's face is %v, want the sidebar hover face %+v", hovered, k.sidebarHover)
	}
}

// TestSidebarEmptyCatalog checks the empty state: an empty catalog says so in a
// status region and offers no thread list, rather than an empty focus stop.
func TestSidebarEmptyCatalog(t *testing.T) {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	tt := ui.NewTester(s.View, 900, 600)

	if !tt.HasText("No sessions yet") {
		t.Fatalf("the empty sidebar says nothing, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleStatus, "No sessions yet"); !ok {
		t.Errorf("the empty sidebar is not a status region, texts %q", tt.Texts())
	}
}

// sameColor reports whether a sampled pixel is exactly the token's color.
func sameColor(pixel color.RGBA, token ui.Color) bool {
	return pixel.R == token.R && pixel.G == token.G && pixel.B == token.B && pixel.A == token.A
}

// TestSidebarHidesTheBranchDirectory checks the reference's session column:
// it draws no branch section, so no branch row, switch, refresh or end entry can
// appear there. The callbacks stay in State and Actions for the session header
// menu that will carry them.
func TestSidebarHidesTheBranchDirectory(t *testing.T) {
	s := fixture()
	s.Branches = []Branch{
		{ID: "main", Label: "main", State: "open"},
		{ID: "side", Label: "side", State: "open"},
	}
	s.SelectedBranch = "side"
	s.OnSelectBranch = func(string) { t.Error("the sidebar offered a branch switch") }
	s.OnEndBranch = func(string) { t.Error("the sidebar offered a branch end") }
	s.OnRefreshBranches = func() { t.Error("the sidebar offered a branch refresh") }
	tt := ui.NewTester(s.View, 900, 900)

	for _, gone := range []string{"Branches", "Branches (read-only)", "main (current)", "Refresh", "End branch", reasonBranches} {
		if tt.HasText(gone) {
			t.Errorf("the sidebar still draws %q, texts %q", gone, tt.Texts())
		}
	}

	actions := s.actions()
	if actions.OnSelectBranch == nil || actions.OnEndBranch == nil || actions.OnRefreshBranches == nil {
		t.Errorf("the branch callbacks were dropped: %+v", actions)
	}
}

// TestSidebarHidesTheThreadActions checks that the reference's column carries
// no pin, rename, archive or delete row: the first three have no verified App
// Server RPC, and delete moves to the session header menu later. The delete
// callback stays wired in Actions.
func TestSidebarHidesTheThreadActions(t *testing.T) {
	s := fixture()
	s.OnDeleteSession = func(string) { t.Error("the sidebar offered a delete") }
	tt := ui.NewTester(s.View, 900, 900)

	for _, gone := range []string{"Pin", "Rename", "Archive", "Delete session", "Thread actions (unavailable)"} {
		if tt.HasText(gone) {
			t.Errorf("the sidebar still draws %q, texts %q", gone, tt.Texts())
		}
	}
	if s.actions().OnDeleteSession == nil {
		t.Error("the delete callback was dropped")
	}
}

// TestSidebarRelativeTime checks the row stamp the reference shows: catalog
// milliseconds become minutes, hours and days, and a zero time draws nothing.
func TestSidebarRelativeTime(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		in   int64
		want string
	}{
		{"zero", 0, ""},
		{"seconds", now.Add(-5 * time.Minute).Unix(), "5m"},
		{"millis", now.Add(-2 * time.Hour).UnixMilli(), "2h"},
		{"days", now.Add(-3 * 24 * time.Hour).UnixMilli(), "3d"},
		{"future", now.Add(time.Minute).UnixMilli(), ""},
	}
	for _, tc := range cases {
		if got := relativeStamp(tc.in, now); got != tc.want {
			t.Errorf("%s: relativeStamp(%d) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

// searchSessions is the catalog the search tests filter: the titles, the ids
// and the groups each give a query something different to match.
func searchSessions() []Session {
	return []Session{
		{ID: "s1", Title: "Fix flaky tests", Group: "dim-electron"},
		{ID: "s2", Title: "Plan release", Group: "dim-electron"},
		{ID: "s3", Title: "Read notes", Group: "docs"},
		{ID: "s4", Title: "Untitled"},
	}
}

// searchState is a shell holding that catalog, with the first session selected.
func searchState() *State {
	s := fixture()
	s.Sessions = searchSessions()
	s.Selected = "s1"
	return s
}

// sidebarRowShown reports whether the thread list exposes a row for the
// session title. Rows are list items; the window's title bar shows the same
// title, so a plain text search would not tell them apart.
func sidebarRowShown(tt *ui.Tester, title string) bool {
	_, ok := tt.AccessByRole(ui.RoleListItem, title)
	return ok
}

// openSearch opens the sidebar's local filter box, as a user does before
// typing.
func openSearch(t *testing.T, tt *ui.Tester) {
	t.Helper()
	if err := tt.Click("Search"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Filter sessions"); !ok {
		t.Fatalf("the search entry did not open its box, texts %q", tt.Texts())
	}
	if err := tt.Click("Filter sessions"); err != nil {
		t.Fatal(err)
	}
}

// TestSidebarSearchesSessionsLocally checks the local filter: the box narrows
// the thread list to the sessions whose title contains the query, ignoring
// case, and the catalog the window passed is left exactly as it was.
func TestSidebarSearchesSessionsLocally(t *testing.T) {
	s := searchState()
	before := slices.Clone(s.Sessions)
	tt := ui.NewTester(s.View, 900, 1000)
	openSearch(t, tt)

	tt.Type("PLAN")
	if !sidebarRowShown(tt, "Plan release") {
		t.Fatalf("the filter dropped the matching session, texts %q", tt.Texts())
	}
	for _, gone := range []string{"Fix flaky tests", "Read notes", "Untitled"} {
		if sidebarRowShown(tt, gone) {
			t.Errorf("the filter kept %q, texts %q", gone, tt.Texts())
		}
	}
	if !slices.Equal(before, s.Sessions) {
		t.Errorf("the filter rewrote the catalog: %+v, want %+v", s.Sessions, before)
	}
	if s.Selected != "s1" {
		t.Errorf("the filter moved the selection to %q", s.Selected)
	}
}

// TestSidebarSearchesBySessionID checks that the id is searched as well as the
// title: a session whose title the user does not remember is still found.
func TestSidebarSearchesBySessionID(t *testing.T) {
	s := searchState()
	tt := ui.NewTester(s.View, 900, 1200)
	openSearch(t, tt)

	tt.Type("s3")
	if !sidebarRowShown(tt, "Read notes") {
		t.Fatalf("the filter did not match the session id, texts %q", tt.Texts())
	}
	for _, gone := range []string{"Plan release", "Fix flaky tests"} {
		if sidebarRowShown(tt, gone) {
			t.Errorf("the id filter kept %q, texts %q", gone, tt.Texts())
		}
	}
}

// TestSidebarSearchWithoutMatches checks the empty state of a filter that
// leaves no row: it says so, in a status region, and invents no session.
func TestSidebarSearchWithoutMatches(t *testing.T) {
	s := searchState()
	tt := ui.NewTester(s.View, 900, 1200)
	openSearch(t, tt)

	tt.Type("zzz")
	if !tt.HasText("No matching sessions") {
		t.Fatalf("a filter with no match says nothing, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleStatus, "No matching sessions"); !ok {
		t.Errorf("the no-match state is not a status region, texts %q", tt.Texts())
	}
	if tt.HasText("No sessions yet") {
		t.Errorf("a filtered catalog reads as an empty catalog, texts %q", tt.Texts())
	}
	if sidebarRowShown(tt, "Plan release") {
		t.Error("a filter with no match still shows a row")
	}
	if len(s.Sessions) != len(searchSessions()) {
		t.Errorf("the filter dropped catalog rows: %+v", s.Sessions)
	}
}

// TestSidebarSearchEscape checks the ways out: Escape with an empty box closes
// it, and with text in the focused box it first clears the filter, which brings
// the whole catalog back; a second Escape closes the box.
func TestSidebarSearchEscape(t *testing.T) {
	s := searchState()
	tt := ui.NewTester(s.View, 900, 1200)

	if err := tt.Click("Search"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Filter sessions"); ok {
		t.Fatalf("Escape left an empty search box open, texts %q", tt.Texts())
	}

	openSearch(t, tt)
	tt.Type("plan")
	if !sidebarRowShown(tt, "Plan release") || sidebarRowShown(tt, "Fix flaky tests") {
		t.Fatalf("the filter did not narrow the list, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if s.sidebar.query != "" {
		t.Errorf("Escape left the query %q", s.sidebar.query)
	}
	if !sidebarRowShown(tt, "Plan release") || !sidebarRowShown(tt, "Fix flaky tests") {
		t.Fatalf("Escape did not clear the filter, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Filter sessions"); ok {
		t.Fatalf("a second Escape did not close the box, texts %q", tt.Texts())
	}
}

// TestSidebarSearchClosesAndClears checks the entry as a toggle: a second
// click closes the box and clears the query, so reopening it starts from the
// whole catalog instead of the last filter.
func TestSidebarSearchClosesAndClears(t *testing.T) {
	s := searchState()
	// The window is taller than the fixture's: the account footer and the two
	// management entries take room that the thread list used to have at 1000,
	// and this test needs all four rows built to tell the closed filter from an
	// open one.
	tt := ui.NewTester(s.View, 900, 1100)
	openSearch(t, tt)
	tt.Type("plan")
	if !sidebarRowShown(tt, "Plan release") || sidebarRowShown(tt, "Fix flaky tests") {
		t.Fatalf("the filter did not narrow the list, texts %q", tt.Texts())
	}

	if err := tt.Click("Search"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Filter sessions"); ok {
		t.Fatalf("a second click left the box open, texts %q", tt.Texts())
	}
	if s.sidebar.query != "" {
		t.Errorf("closing the box left the query %q", s.sidebar.query)
	}
	for _, want := range []string{"Fix flaky tests", "Plan release", "Read notes"} {
		if !sidebarRowShown(tt, want) {
			t.Errorf("the closed filter still hides %q, texts %q", want, tt.Texts())
		}
	}
	if s.sidebar.filter != "" {
		t.Errorf("the closed filter is still %q", s.sidebar.filter)
	}

	openSearch(t, tt)
	if s.sidebar.query != "" {
		t.Errorf("reopening the box kept the query %q", s.sidebar.query)
	}
	if s.sidebar.filter != "" {
		t.Errorf("reopening the box kept the filter %q", s.sidebar.filter)
	}
}

// TestSidebarSearchRebuildsGroups checks that the headings are laid out from
// the filtered copy: a group the query leaves keeps its heading, and a group it
// hides draws no heading at all.
func TestSidebarSearchRebuildsGroups(t *testing.T) {
	s := searchState()
	tt := ui.NewTester(s.View, 900, 1200)
	openSearch(t, tt)

	tt.Type("release")
	if !tt.HasText("dim-electron") {
		t.Fatalf("the filtered heading is missing, texts %q", tt.Texts())
	}
	for _, gone := range []string{"docs", ungroupedLabel} {
		if tt.HasText(gone) {
			t.Errorf("the filter kept the %q heading, texts %q", gone, tt.Texts())
		}
	}
	if !sidebarRowShown(tt, "Plan release") || sidebarRowShown(tt, "Read notes") {
		t.Errorf("the filtered group still shows other rows, texts %q", tt.Texts())
	}
}

// TestSidebarSearchTabReachable checks the keyboard: the opened box is the
// entry's next Tab stop, and Tab leaves it for the entry after it.
func TestSidebarSearchTabReachable(t *testing.T) {
	s := searchState()
	tt := ui.NewTester(s.View, 900, 1200)

	if err := tt.Click("Search"); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused("Search") {
		t.Fatalf("the clicked entry did not take the focus, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Filter sessions") {
		t.Fatalf("Tab does not reach the search box, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Scheduled tasks") {
		t.Fatalf("Tab does not leave the search box for the next entry, texts %q", tt.Texts())
	}
}

// TestSidebarSearchShortcut checks the shortcut the entry's hint names: it
// opens the box and closes it again, and sends nothing either way.
func TestSidebarSearchShortcut(t *testing.T) {
	s := searchState()
	tt := ui.NewTester(s.View, 900, 1200)

	tt.Key(searchShortcutMods, searchShortcutKey)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Filter sessions"); !ok {
		t.Fatalf("the shortcut did not open the box, texts %q", tt.Texts())
	}
	tt.Key(searchShortcutMods, searchShortcutKey)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Filter sessions"); ok {
		t.Fatalf("the shortcut did not close the box, texts %q", tt.Texts())
	}
}

// TestSidebarSearchUsesTheCallerQuery checks the view contract for a caller
// that keeps the text itself: Searching opens the box, SearchQuery seeds it and
// takes the edits in place, and the filter reads that text.
func TestSidebarSearchUsesTheCallerQuery(t *testing.T) {
	selected := "s1"
	query := "notes"
	v := SidebarView{
		Sessions:    searchSessions(),
		Selected:    &selected,
		Searching:   true,
		SearchQuery: &query,
		state:       &sidebarState{},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		renderSidebar(c, newTokens(c.Theme()), v, Actions{})
	}, 900, 1000)

	if !sidebarRowShown(tt, "Read notes") || sidebarRowShown(tt, "Plan release") {
		t.Fatalf("the caller's query did not filter, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Filter sessions"); !ok {
		t.Fatalf("Searching did not open the box, texts %q", tt.Texts())
	}
	if err := tt.Click("Filter sessions"); err != nil {
		t.Fatal(err)
	}
	tt.Type("!")
	if query != "notes!" {
		t.Errorf("the box did not edit the caller's query: %q", query)
	}
}

// TestSidebarSearchNeedsNoCallback checks the entry without any Actions wired:
// local filtering asks the window for nothing, so the entry is offered and the
// box opens, unlike the shell's WIP entries.
func TestSidebarSearchNeedsNoCallback(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 1000)

	info, ok := tt.AccessByRole(ui.RoleButton, "Search")
	if !ok {
		t.Fatalf("the search entry is not exposed, texts %q", tt.Texts())
	}
	if info.Disabled {
		t.Error("the search entry is unavailable although the filter is local")
	}
	if info.Description != "" {
		t.Errorf("the search entry still carries a WIP reason %q", info.Description)
	}
	if err := tt.Click("Search"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Filter sessions"); !ok {
		t.Errorf("the entry did not open the box without a callback, texts %q", tt.Texts())
	}
	if s.Selected != "s1" || len(s.Sessions) != 2 {
		t.Errorf("opening the box changed the shell: selected=%q sessions=%d", s.Selected, len(s.Sessions))
	}
}

// TestSidebarGlobalEntryOrder checks the global entries in dim-electron's
// order: New session, Search, Scheduled tasks and Skills and plugins. The
// account footer sits below the session content, with the settings entry on its
// tail.
func TestSidebarGlobalEntryOrder(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	order := []string{"New session", "Search", "Scheduled tasks", "Skills and plugins"}
	var previous ui.Rect
	for i, want := range order {
		rect, ok := tt.Find(want)
		if !ok {
			t.Fatalf("%q is not drawn, texts %q", want, tt.Texts())
		}
		if i > 0 && rect.Y <= previous.Y {
			t.Errorf("%q at y=%.1f is not below the entry before it at y=%.1f", want, rect.Y, previous.Y)
		}
		previous = rect
	}

	account, ok := tt.Find("Sign in to Dim")
	if !ok {
		t.Fatalf("the account row is not drawn, texts %q", tt.Texts())
	}
	content, ok := tt.Find("Sessions")
	if !ok {
		t.Fatalf("the thread list heading is not drawn, texts %q", tt.Texts())
	}
	if account.Y <= content.Y {
		t.Errorf("the account row at y=%.1f is not below the sidebar content at y=%.1f", account.Y, content.Y)
	}
	settings, ok := tt.Find("Settings")
	if !ok {
		t.Fatalf("the settings entry is not drawn, texts %q", tt.Texts())
	}
	if settings.Y < account.Y || settings.Y >= account.Y+account.H {
		t.Errorf("the settings entry %+v is not on the account row %+v", settings, account)
	}
	nav, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar")
	if !ok {
		t.Fatalf("the sidebar is not exposed, texts %q", tt.Texts())
	}
	if account.Y+account.H > nav.Bounds.Y+nav.Bounds.H {
		t.Errorf("the account row %+v runs past the sidebar %+v", account, nav.Bounds)
	}
}

// TestSidebarAccountRowStaysWIP checks the account entry: the shell has no
// account client, so there is no identity or sign-in fact to show. The entry
// stays visible and unavailable, names the real gap, and a click changes no
// view state.
func TestSidebarAccountRowStaysWIP(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "Sign in to Dim")
	if !ok {
		t.Fatalf("the account entry is not exposed as a button, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("the account entry is available although the shell has no account client")
	}
	if !info.Focusable {
		t.Error("the account entry is unreachable, so its reason is never read")
	}
	if strings.Contains(info.Description, wipBadge) {
		t.Errorf("the account entry still carries a visible %q badge in its description: %q", wipBadge, info.Description)
	}
	if !strings.Contains(info.Description, "account client") {
		t.Errorf("the account entry description %q does not name the missing account client", info.Description)
	}
	if err := tt.Click("Sign in to Dim"); err != nil {
		t.Fatal(err)
	}
	if s.ShowSettings || s.Selected != "s1" || s.Draft != "" || len(s.Sessions) != 2 {
		t.Errorf("a click on the account entry changed the shell: settings=%v selected=%q draft=%q sessions=%d",
			s.ShowSettings, s.Selected, s.Draft, len(s.Sessions))
	}
}

// TestSidebarSettingsOpensTheSurface checks the settings entry at the account
// row's tail: it is the sidebar's own Settings button, and clicking it opens
// the same appearance surface the toolbar's entry toggles, with no App Server
// request.
func TestSidebarSettingsOpensTheSurface(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	nav, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar")
	if !ok {
		t.Fatalf("the sidebar is not exposed, texts %q", tt.Texts())
	}
	info, ok := tt.AccessByRole(ui.RoleButton, "Settings")
	if !ok {
		t.Fatalf("the settings entry is not exposed as a button, texts %q", tt.Texts())
	}
	if info.Bounds.X < nav.Bounds.X || info.Bounds.X+info.Bounds.W > nav.Bounds.X+nav.Bounds.W {
		t.Fatalf("the first Settings button %+v is not the sidebar's, sidebar %+v", info.Bounds, nav.Bounds)
	}
	if info.Disabled {
		t.Fatal("the settings entry is unavailable although settings is desktop-local")
	}

	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	if !s.ShowSettings {
		t.Fatalf("clicking the sidebar settings entry did not open the surface: %+v", s.ShowSettings)
	}
	tt.Frame()
	if !tt.HasText("Appearance") || !tt.HasText("Interface language") {
		t.Errorf("the appearance surface is not drawn after the click, texts %q", tt.Texts())
	}
	if !tt.HasText("Sign in to Dim") {
		t.Errorf("the account row disappeared behind the settings surface, texts %q", tt.Texts())
	}
}

// TestSidebarStartsWithTheEntries checks the reference's top of the column:
// there is no wordmark and no update entry, and the first entry begins below
// the traffic lights rather than at the window's top edge.
func TestSidebarStartsWithTheEntries(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 800, 720)

	if tt.HasText("DimAgent") {
		t.Errorf("the sidebar still draws its wordmark, texts %q", tt.Texts())
	}
	if tt.HasText("Update") {
		t.Errorf("the sidebar still draws the update entry, texts %q", tt.Texts())
	}
	first, ok := tt.Find("New session")
	if !ok {
		t.Fatalf("the first entry is not drawn, texts %q", tt.Texts())
	}
	if first.Y < 40 {
		t.Errorf("the first entry starts at y=%.1f, under the traffic lights", first.Y)
	}
	if first.Y > 80 {
		t.Errorf("the first entry starts at y=%.1f, too far below the reference's ~64", first.Y)
	}
}

// TestSidebarDropsTheSessionCountAndGroupEntry checks the thread heading: the
// reference draws "Sessions" with its icons, not a count and a Group entry.
func TestSidebarDropsTheSessionCountAndGroupEntry(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 800, 720)

	if !tt.HasText("Sessions") {
		t.Fatalf("the thread heading is missing, texts %q", tt.Texts())
	}
	for _, gone := range []string{"Group", "Sessions 2", reasonGroup} {
		if tt.HasText(gone) {
			t.Errorf("the sidebar still draws %q, texts %q", gone, tt.Texts())
		}
	}
}

// TestSidebarDrawsNoCapabilityCards checks that the unavailable capabilities
// moved to settings: the sidebar itself exposes no capability group, method or
// gap.
func TestSidebarDrawsNoCapabilityCards(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 900)

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Unavailable capabilities"); ok {
		t.Error("the sidebar still groups the capability cards")
	}
	for _, gone := range []string{"session.create", "attachment.put", "Turn cancellation"} {
		if tt.HasText(gone) {
			t.Errorf("the sidebar still draws the capability %q, texts %q", gone, tt.Texts())
		}
	}
}

// TestSidebarHidesThreadActionsAtZeroSessions checks the empty catalog: the
// delete and thread-action rows never appear, whichever callbacks the window
// wires.
func TestSidebarHidesThreadActionsAtZeroSessions(t *testing.T) {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	s.OnDeleteSession = func(string) { t.Error("a delete entry was offered with no session") }
	tt := ui.NewTester(s.View, 900, 900)

	if !tt.HasText("No sessions yet") {
		t.Errorf("the empty catalog lost its message, texts %q", tt.Texts())
	}
	for _, gone := range []string{"Delete session", "Pin", "Rename", "Archive"} {
		if tt.HasText(gone) {
			t.Errorf("the empty sidebar draws %q, texts %q", gone, tt.Texts())
		}
	}
}

// TestSidebarEmptyCatalogKeepsTheReferenceStructure checks the empty column's
// structure, as the reference draws it: the session heading with its icons, the
// workspace heading, the empty message under it and a hairline closing the
// section, with no WIP text or capability group.
func TestSidebarEmptyCatalogKeepsTheReferenceStructure(t *testing.T) {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	s.Settings = SettingsView{Locale: LocaleZHCN}
	const width, height = 800, 720
	tt := ui.NewTester(s.View, width, height)
	k := newTokens(ui.LightTheme())

	sessions := textBox(t, tt, "会话")
	workspace := textBox(t, tt, "工作区")
	empty := textBox(t, tt, "暂无会话")
	if sessions.Y >= workspace.Y || workspace.Y >= empty.Y {
		t.Errorf("the empty column's order is off: 会话 y=%.1f, 工作区 y=%.1f, 暂无会话 y=%.1f",
			sessions.Y, workspace.Y, empty.Y)
	}
	if _, ok := tt.AccessByRole(ui.RoleStatus, "暂无会话"); !ok {
		t.Errorf("the empty message is not a status region, texts %q", tt.Texts())
	}

	// The section closes with a hairline under the message.
	face := color.RGBA{R: k.border.R, G: k.border.G, B: k.border.B, A: k.border.A}
	divider := false
	for y := int(empty.Y + empty.H); y < min(int(empty.Y+empty.H+40), height); y++ {
		run := 0
		for x := 0; x < int(k.sidebarW); x++ {
			if tt.Image().RGBAAt(x, y) == face {
				run++
			}
		}
		if run > int(k.sidebarW)/2 {
			divider = true
			break
		}
	}
	if !divider {
		t.Errorf("no divider closes the empty section under the message")
	}

	for _, gone := range []string{wipBadge, "Unavailable capabilities"} {
		if tt.HasText(gone) {
			t.Errorf("the empty sidebar still draws %q, texts %q", gone, tt.Texts())
		}
	}
}

// TestSidebarListsGroupsUnderTheWorkspaceHeading checks the populated column's
// structure, as the reference draws it: the session heading, the workspace
// heading, the built-in workspace's empty message and its hairline, then the
// project group and its rows. The group heading carries its folder and its name
// only; the reference has no count badge. The empty message is inset like the
// reference's pl-[26px], under the group's text rather than its icon.
func TestSidebarListsGroupsUnderTheWorkspaceHeading(t *testing.T) {
	s := fixture()
	s.Sessions = []Session{
		{ID: "s1", Title: "Explain what this repository does", Group: "dimagent-mygo", UpdatedAt: time.Now().Add(-5 * time.Minute).UnixMilli()},
		{ID: "s2", Title: "Plan the next sidebar step", Group: "dimagent-mygo"},
	}
	s.Selected = "s1"
	s.Settings = SettingsView{Locale: LocaleZHCN}
	tt := ui.NewTester(s.View, 800, 720)
	k := newTokens(ui.LightTheme())

	sessions := textBox(t, tt, "会话")
	workspace := textBox(t, tt, "工作区")
	empty := textBox(t, tt, "暂无会话")
	group := textBox(t, tt, "dimagent-mygo")
	row := textBox(t, tt, "Explain what this repository does")
	if sessions.Y >= workspace.Y || workspace.Y >= empty.Y || empty.Y >= group.Y || group.Y >= row.Y {
		t.Errorf("the populated column's order is off: 会话 y=%.1f, 工作区 y=%.1f, 暂无会话 y=%.1f, dimagent-mygo y=%.1f, row y=%.1f",
			sessions.Y, workspace.Y, empty.Y, group.Y, row.Y)
	}
	if _, ok := tt.AccessByRole(ui.RoleStatus, "暂无会话"); ok {
		t.Error("the populated column reads the built-in workspace message as the list's status")
	}
	if tt.HasText("2") {
		t.Errorf("the group heading still draws its row count, texts %q", tt.Texts())
	}
	if want := k.sidebarPadX + sidebarEmptyIndent; math.Abs(float64(empty.X-want)) > 0.5 {
		t.Errorf("暂无会话 starts at x=%.1f, want the reference's pl-[26px] at %.1f", empty.X, want)
	}
	if math.Abs(float64(empty.X-group.X-k.sidebarPadX-sidebarIconSize-k.sidebarGap)) > 0.5 {
		t.Errorf("暂无会话 at x=%.1f does not align with the group heading's text at x=%.1f",
			empty.X, group.X+k.sidebarPadX+sidebarIconSize+k.sidebarGap)
	}
	// The section closes with a hairline between the message and the group.
	face := color.RGBA{R: k.border.R, G: k.border.G, B: k.border.B, A: k.border.A}
	divider := false
	for y := int(empty.Y + empty.H); y < int(group.Y); y++ {
		run := 0
		for x := int(k.sidebarPadX * 2); x < int(k.sidebarW-k.sidebarPadX*2); x++ {
			if tt.Image().RGBAAt(x, y) == face {
				run++
			}
		}
		if run > int(k.sidebarW-4*k.sidebarPadX)/2 {
			divider = true
			break
		}
	}
	if !divider {
		t.Error("no hairline closes the built-in workspace section above the project group")
	}
	if !tt.HasText("5m") {
		t.Errorf("the session row lost its relative stamp, texts %q", tt.Texts())
	}
}

// TestSidebarSessionRowStartsWithItsTitle checks the reference's row: the title
// begins at the row's own padding, with no status dot in a leading cell. The
// selected row's face runs into the title's first glyph, so a dot would show as
// a colored block before the text.
func TestSidebarSessionRowStartsWithItsTitle(t *testing.T) {
	s := fixture()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 600)
	k := newTokens(theme)

	nav, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar")
	if !ok {
		t.Fatalf("the sidebar is not exposed, texts %q", tt.Texts())
	}
	row := textBox(t, tt, "First session")
	if row.X < nav.Bounds.X || row.X+row.W > nav.Bounds.X+nav.Bounds.W {
		t.Fatalf("the session row %+v is not inside the sidebar %+v", row, nav.Bounds)
	}
	y := int(row.Y + row.H/2)
	ink := -1
	for x := int(row.X); x < int(row.X+row.W); x++ {
		if got := tt.Image().RGBAAt(x, y); !sameColor(got, k.sidebarAccent) {
			ink = x
			break
		}
	}
	if ink < 0 {
		t.Fatalf("the selected row paints no title")
	}
	if inset := float32(ink) - row.X; inset > k.sidebarGap+2 {
		t.Errorf("the row's first ink is %.1f DIPs in, want the title at the row's %.1f padding: a leading status cell is still drawn",
			inset, k.sidebarGap)
	}
}

// TestSidebarSearchRowShowsNoHint checks the reference's search entry: it is
// the word alone, with no shortcut hint at the row's tail. The shortcut itself
// stays wired, as TestSidebarSearchShortcut checks.
func TestSidebarSearchRowShowsNoHint(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	for _, hint := range []string{"⌘G", "Ctrl+G"} {
		if tt.HasText(hint) {
			t.Errorf("the search row still draws the shortcut hint %q, texts %q", hint, tt.Texts())
		}
	}
	info, ok := tt.AccessByRole(ui.RoleButton, "Search")
	if !ok {
		t.Fatalf("the search entry is not exposed, texts %q", tt.Texts())
	}
	if info.Name != "Search" {
		t.Errorf("the search entry is exposed as %q, want %q", info.Name, "Search")
	}
	if info.Disabled {
		t.Error("the search entry is unavailable although its filter is local")
	}
}

// TestSidebarFooterKeepsOnlyTheGear checks the reference's footer: the gear
// alone opens the settings surface. Its label stays the accessible name and the
// tooltip, and no word is drawn beside the glyph.
func TestSidebarFooterKeepsOnlyTheGear(t *testing.T) {
	s := fixture()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 600)
	k := newTokens(theme)

	nav, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar")
	if !ok {
		t.Fatalf("the sidebar is not exposed, texts %q", tt.Texts())
	}
	info, ok := tt.AccessByRole(ui.RoleButton, "Settings")
	if !ok {
		t.Fatalf("the settings entry is not exposed, texts %q", tt.Texts())
	}
	if info.Bounds.X < nav.Bounds.X || info.Bounds.X+info.Bounds.W > nav.Bounds.X+nav.Bounds.W {
		t.Fatalf("the settings button %+v is not the sidebar's, sidebar %+v", info.Bounds, nav.Bounds)
	}
	if want := 2*k.sidebarPadX + sidebarIconSize; info.Bounds.W > want+0.5 {
		t.Errorf("the settings button is %.1f wide, want the gear's %.1f: a word is still drawn beside it",
			info.Bounds.W, want)
	}
	if info.Name != tr(LocaleEN, keySidebarSettingsEntry) {
		t.Errorf("the settings button is named %q, want %q", info.Name, tr(LocaleEN, keySidebarSettingsEntry))
	}
	if info.Description != info.Name {
		t.Errorf("the settings button's description is %q, want the gear's accessible name %q", info.Description, info.Name)
	}

	// The glyph sits at the button's left padding; the rest of the row is the
	// sidebar's face, so no label is painted there.
	face := color.RGBA{R: k.sidebar.R, G: k.sidebar.G, B: k.sidebar.B, A: k.sidebar.A}
	image := tt.Image()
	glyphRight := int(info.Bounds.X+k.sidebarPadX) + sidebarIconSize
	right := int(info.Bounds.X + info.Bounds.W)
	y := int(info.Bounds.Y + info.Bounds.H/2)
	for x := glyphRight + 1; x < right; x++ {
		if got := image.RGBAAt(x, y); got != face {
			t.Errorf("the settings row paints %v at x=%d, after the gear: a label is still drawn", got, x)
			break
		}
	}
	if !tt.HasText("Sign in to Dim") {
		t.Errorf("the account row is missing from the footer, texts %q", tt.Texts())
	}
}

// TestSidebarMatchesTheReferenceRhythm checks the column's vertical rhythm
// against the reference window. The wanted centers are the label rows measured
// from docs/evidence/electron-reference/window-conversation.png (1600×1440 at
// 2x, so pixels ÷ 2 are DIPs): the four entries, the two headings, the built-in
// workspace's message, the project heading, the session row and the account
// footer. Each must land within 3 DIPs, and the account row must still sit on
// the sidebar's bottom edge, so the room is added above the list rather than by
// pushing the footer up.
func TestSidebarMatchesTheReferenceRhythm(t *testing.T) {
	s := fixture()
	s.Settings = SettingsView{Locale: LocaleZHCN}
	s.Sessions = []Session{
		{ID: "s1", Title: "Explain what this repository does in one paragraph.", Group: "dimagent-mygo", UpdatedAt: time.Now().Add(-5 * time.Minute).UnixMilli()},
	}
	s.Selected = "s1"
	const width, height = 800, 720
	tt := ui.NewTester(s.View, width, height)

	// The sidebar is drawn before the bar, so a label the breadcrumb repeats,
	// such as the project, is found in the column first.
	cases := []struct {
		label string
		want  float64
	}{
		{tr(LocaleZHCN, keySidebarNewSession), 72.8},
		{tr(LocaleZHCN, keySidebarSearch), 104.2},
		{tr(LocaleZHCN, keySidebarScheduled), 135.8},
		{tr(LocaleZHCN, keySidebarSkills), 167.2},
		{tr(LocaleZHCN, keySidebarSessions), 204.0},
		{tr(LocaleZHCN, keyComposerWorkspace), 237.0},
		{tr(LocaleZHCN, keySidebarEmpty), 272.0},
		{"dimagent-mygo", 314.5},
		{"Explain what this repository does in one paragraph.", 345.8},
		{tr(LocaleZHCN, keySidebarAccount), 698.0},
	}
	for _, tc := range cases {
		box := textBox(t, tt, tc.label)
		if center := float64(box.Y + box.H/2); math.Abs(center-tc.want) > 3 {
			t.Errorf("%q centers at %.1f, want the reference's %.1f ±3", tc.label, center, tc.want)
		}
	}

	k := newTokens(ui.LightTheme())
	account := textBox(t, tt, tr(LocaleZHCN, keySidebarAccount))
	if gap := height - (account.Y + account.H); math.Abs(float64(gap-k.sidebarPadX)) > 0.5 {
		t.Errorf("the account row leaves %.1f below it, want the sidebar's %.1f DIP edge", gap, k.sidebarPadX)
	}
}
