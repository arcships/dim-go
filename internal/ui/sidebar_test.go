package ui

import (
	"image/color"
	"math"
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
// the real gap, and a click changes no view state. The scheduled-tasks and
// skills entries are real routes now; their own test checks the pages they open.
func TestSidebarWIPEntriesDoNothing(t *testing.T) {
	s := fixture()
	s.OnCreateSession = nil
	tt := ui.NewTester(s.View, 900, 600)
	width := s.SidebarWidth

	entries := map[string]string{
		"Sign in to Dim": "account",
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
		// The list insets its rows by sidebarRowInset, so the sample sits just
		// inside the row's face, left of the title's ink.
		return tt.Image().RGBAAt(int(rect.X)+2, int(rect.Y+rect.H/2))
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
	// The face eases from the sidebar's tint to the hover tint over the fast
	// step, as the reference's button transition does; sample it arrived.
	tt.Advance(motionFast + time.Millisecond)
	hovered := rowFace("Second session")
	if !sameColor(hovered, k.sidebarHover) {
		t.Errorf("the hovered row's face is %v, want the sidebar hover face %+v", hovered, k.sidebarHover)
	}
}

// TestSidebarThreadRowsAreInset checks the reference's session row face: it
// does not run edge to edge in the column. window-conversation.png draws the
// selected row from x≈11.5 to 248.5, so its face starts 4.5 DIPs inside the
// list's own 7 DIP padding and stops half a DIP inside it on the right; the
// hover face of an unselected row is inset the same way. The row's title and
// relative stamp move with the face.
func TestSidebarThreadRowsAreInset(t *testing.T) {
	s := fixture()
	tt, k := glyphTester(t, s)
	nav, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar")
	if !ok {
		t.Fatalf("the sidebar is not exposed, texts %q", tt.Texts())
	}
	insetLeft := nav.Bounds.X + k.sidebarPadX + sidebarRowInset
	insetRight := nav.Bounds.X + nav.Bounds.W - k.sidebarPadX - sidebarRowEndInset

	// faceSpan returns the first and last pixel of the face on row y, scanning
	// the row's width in the rendered frame.
	faceSpan := func(row ui.Rect, y int, face ui.Color) (int, int, bool) {
		img := tt.Image()
		left, right := -1, -1
		for x := int(row.X) - 1; x <= int(row.X+row.W); x++ {
			if x < 0 || x >= img.Bounds().Dx() {
				continue
			}
			if sameColor(img.RGBAAt(x, y), face) {
				if left < 0 {
					left = x
				}
				right = x
			}
		}
		return left, right, left >= 0
	}
	// checkInset compares the face's edges with the column's content edge plus
	// the reference's row inset.
	checkInset := func(name string, row ui.Rect, left, right float32) {
		t.Helper()
		if math.Abs(float64(left-insetLeft)) > 1 {
			t.Errorf("%s's face starts at %.1f, want it %.1f DIPs inside the column's edge", name, left, insetLeft)
		}
		if math.Abs(float64(right-insetRight)) > 1 {
			t.Errorf("%s's face ends at %.1f, want it %.1f DIPs inside the column's other edge", name, right, insetRight)
		}
		if math.Abs(float64(row.X-insetLeft)) > 0.5 {
			t.Errorf("%s's box starts at %.1f, want the inset edge at %.1f so the list's own choice face stays under the row", name, row.X, insetLeft)
		}
		if math.Abs(float64(row.X+row.W-insetRight)) > 0.5 {
			t.Errorf("%s's box ends at %.1f, want the inset edge at %.1f", name, row.X+row.W, insetRight)
		}
	}

	first, ok := tt.Find(s.Sessions[0].Title)
	if !ok {
		t.Fatalf("%q is not drawn, texts %q", s.Sessions[0].Title, tt.Texts())
	}
	left, right, ok := faceSpan(first, int(first.Y+first.H/2), k.sidebarAccent)
	if !ok {
		t.Fatalf("the selected row paints no accent face, texts %q", tt.Texts())
	}
	checkInset("the selected row", first, float32(left), float32(right))

	second, ok := tt.Find(s.Sessions[1].Title)
	if !ok {
		t.Fatalf("%q is not drawn, texts %q", s.Sessions[1].Title, tt.Texts())
	}
	tt.Move(second.X+second.W/2, second.Y+second.H/2)
	tt.Frame()
	tt.Advance(motionFast + time.Millisecond)
	left, right, ok = faceSpan(second, int(second.Y+second.H/2), k.sidebarHover)
	if !ok {
		t.Fatalf("the hovered row paints no hover face, texts %q", tt.Texts())
	}
	checkInset("the hovered row", second, float32(left), float32(right))
}

// TestSidebarDrawsNoCollapseEntry checks that the column no longer draws the
// reference's panel-collapse target. The shell has no collapse behavior, so the
// mark at the column's top right was a fake entry; the entries and headings it
// floated over still start where the reference draws them.
func TestSidebarDrawsNoCollapseEntry(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleButton, "Collapse sidebar"); ok {
		t.Error("the sidebar still exposes a collapse entry")
	}
	if tt.HasText("Collapse sidebar") {
		t.Errorf("the sidebar still draws the collapse label, texts %q", tt.Texts())
	}
	if !tt.HasText("New session") {
		t.Errorf("the entries are missing above the removed mark, texts %q", tt.Texts())
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

// searchSessions is the catalog the search palette tests filter: the titles
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

// openSearch opens the window's search palette from the sidebar entry, as a
// user does before typing, and checks that its input has the focus.
func openSearch(t *testing.T, tt *ui.Tester) {
	t.Helper()
	if err := tt.Click(tr(LocaleEN, keySidebarSearch)); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.AccessByRole(ui.RoleDialog, ""); !ok {
		t.Fatalf("the search entry did not open the palette, texts %q", tt.Texts())
	}
	if !tt.Focused(tr(LocaleEN, keySearchPlaceholder)) {
		t.Fatalf("the palette's input does not have the focus, texts %q", tt.Texts())
	}
}

// TestSidebarSearchEntryOpensThePalette checks the entry that replaced the
// column's inline filter: a click opens the centered palette, the sidebar keeps
// its rows under it, and the catalog is left exactly as it was.
func TestSidebarSearchEntryOpensThePalette(t *testing.T) {
	s := searchState()
	before := append([]Session(nil), s.Sessions...)
	tt := ui.NewTester(s.View, 900, 1000)

	if _, ok := tt.AccessByRole(ui.RoleTextField, tr(LocaleEN, keySearchPlaceholder)); ok {
		t.Fatalf("the closed sidebar already draws a search field, texts %q", tt.Texts())
	}
	openSearch(t, tt)
	if !tt.HasText("Fix flaky tests") || !tt.HasText("Plan release") {
		t.Errorf("the sidebar's rows are missing under the palette, texts %q", tt.Texts())
	}
	for i := range before {
		if before[i] != s.Sessions[i] {
			t.Errorf("opening the palette rewrote the catalog: %+v, want %+v", s.Sessions, before)
		}
	}
	if s.Selected != "s1" {
		t.Errorf("opening the palette moved the selection to %q", s.Selected)
	}
}

// TestSidebarDrawsNoFilterField checks that the reference's search entry is a
// plain row now: the column keeps no text field of its own and no filter copy,
// so nothing can narrow the thread list behind the window's back.
func TestSidebarDrawsNoFilterField(t *testing.T) {
	s := searchState()
	tt := ui.NewTester(s.View, 900, 1000)

	for _, gone := range []string{"Filter sessions", "Title or id", "No matching sessions"} {
		if tt.HasText(gone) {
			t.Errorf("the sidebar still draws the old filter %q, texts %q", gone, tt.Texts())
		}
	}
	info, ok := tt.AccessByRole(ui.RoleTextField, tr(LocaleEN, keyComposerLabel))
	if !ok || info.Placeholder != tr(LocaleEN, keyComposerPlaceholder) {
		t.Errorf("the only field on the frame is not the composer's: %+v", info)
	}
}

// TestSidebarSearchEntryIsTheCallersAction checks the region contract: the
// entry calls Actions.OnOpenSearch and draws nothing itself, and a caller that
// wires no opener keeps the row visible and explained instead of silently dead.
func TestSidebarSearchEntryIsTheCallersAction(t *testing.T) {
	selected := "s1"
	v := SidebarView{
		Sessions: searchSessions(),
		Selected: &selected,
		state:    &sidebarState{},
	}
	opened := 0
	tt := ui.NewTester(func(c *ui.Context) {
		renderSidebar(c, newTokens(c.Theme()), v, Actions{OnOpenSearch: func() { opened++ }})
	}, 900, 1000)
	if err := tt.Click(tr(LocaleEN, keySidebarSearch)); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Errorf("the entry called OnOpenSearch %d times, want 1", opened)
	}

	tt = ui.NewTester(func(c *ui.Context) {
		renderSidebar(c, newTokens(c.Theme()), v, Actions{})
	}, 900, 1000)
	info, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keySidebarSearch))
	if !ok {
		t.Fatalf("the unopened entry is not exposed, texts %q", tt.Texts())
	}
	if !info.Disabled || info.Description == "" {
		t.Errorf("the entry without a callback is %+v, want a visible WIP row", info)
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
// row's tail: it is the sidebar's own Settings button, and clicking it opens the
// full-window appearance route with no App Server request. The route replaces
// the sidebar, and Back brings the same selection and draft back.
func TestSidebarSettingsOpensTheSurface(t *testing.T) {
	s := fixture()
	s.Draft = "keep me"
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
		t.Fatalf("clicking the sidebar settings entry did not open the route: %+v", s.ShowSettings)
	}
	tt.Frame()
	if !tt.HasText("Appearance") || !tt.HasText("Interface language") {
		t.Errorf("the appearance route is not drawn after the click, texts %q", tt.Texts())
	}
	if tt.HasText("Sign in to Dim") {
		t.Errorf("the account row is still drawn behind the whole-window route, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar"); ok {
		t.Errorf("the session sidebar is still drawn behind the route, texts %q", tt.Texts())
	}

	if err := tt.Click(tr(LocaleEN, keySettingsBack)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Sign in to Dim") {
		t.Errorf("the account row did not return with the sidebar, texts %q", tt.Texts())
	}
	if s.Selected != "s1" || s.Draft != "keep me" {
		t.Errorf("closing the route changed the window: selected=%q draft=%q", s.Selected, s.Draft)
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
// reference draws "Sessions", not a count and a Group entry, and the add and
// list marks at its tail are gone too: they took no click, and the add repeated
// the new-session entry. The heading's tail is the sidebar's own face.
func TestSidebarDropsTheSessionCountAndGroupEntry(t *testing.T) {
	s := fixture()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 800, 720)
	k := newTokens(theme)

	if !tt.HasText("Sessions") {
		t.Fatalf("the thread heading is missing, texts %q", tt.Texts())
	}
	for _, gone := range []string{"Group", "Sessions 2", reasonGroup} {
		if tt.HasText(gone) {
			t.Errorf("the sidebar still draws %q, texts %q", gone, tt.Texts())
		}
	}
	heading := textBox(t, tt, "Sessions")
	image := tt.Image()
	center := heading.Y + heading.H/2
	top, bottom := int(center-k.sidebarSectionH/2), int(center+k.sidebarSectionH/2)
	for y := top; y < bottom; y++ {
		for x := int(heading.X+heading.W) + 4; x < int(k.sidebarW); x++ {
			if got := image.RGBAAt(x, y); !sameColor(got, k.sidebar) {
				t.Fatalf("the session heading paints %v at x=%d, y=%d: a decoration is still drawn", got, x, y)
			}
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
// structure: the session heading, the workspace heading, the hairline closing
// its section, then the project group and its rows. The built-in workspace's
// "No sessions yet" message is absent above real rows: it contradicted them.
// The group heading carries its folder and its name only; the reference has no
// count badge.
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
	group := textBox(t, tt, "dimagent-mygo")
	row := textBox(t, tt, "Explain what this repository does")
	if sessions.Y >= workspace.Y || workspace.Y >= group.Y || group.Y >= row.Y {
		t.Errorf("the populated column's order is off: 会话 y=%.1f, 工作区 y=%.1f, dimagent-mygo y=%.1f, row y=%.1f",
			sessions.Y, workspace.Y, group.Y, row.Y)
	}
	if tt.HasText("暂无会话") {
		t.Errorf("the populated column still draws the empty message, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleStatus, "暂无会话"); ok {
		t.Error("the populated column reads the built-in workspace message as the list's status")
	}
	if tt.HasText("2") {
		t.Errorf("the group heading still draws its row count, texts %q", tt.Texts())
	}
	// The section still closes with a hairline between the workspace heading and
	// the first group, even though the empty message above the rows is gone: the
	// message's room stays, without a made-up message, so the rule keeps the
	// reference's y≈295.5 and the project group starts under it.
	face := color.RGBA{R: k.border.R, G: k.border.G, B: k.border.B, A: k.border.A}
	dividerY := -1
	for y := int(workspace.Y + workspace.H); y < int(group.Y); y++ {
		run := 0
		for x := int(k.sidebarPadX * 2); x < int(k.sidebarW-k.sidebarPadX*2); x++ {
			if tt.Image().RGBAAt(x, y) == face {
				run++
			}
		}
		if run > int(k.sidebarW-4*k.sidebarPadX)/2 {
			dividerY = y
			break
		}
	}
	if dividerY < 0 {
		t.Error("no hairline closes the workspace section above the project group")
	} else if math.Abs(float64(dividerY)-295.5) > 1 {
		t.Errorf("the hairline closes the workspace section at y=%d, want the reference's 295.5", dividerY)
	}
	if !tt.HasText("5m") {
		t.Errorf("the session row lost its relative stamp, texts %q", tt.Texts())
	}
}

// TestSidebarSessionRowStartsWithItsTitle checks the reference's row: the title
// begins after the row's leading cell, px-2 plus the cell's 17.5 DIPs, which is
// 24.5 DIPs inside the row's face. This shell has no status or icon fact for a
// session, so the cell stays empty: no ink sits in it, and no made-up dot is
// drawn at the face's edge.
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
	if math.Abs(float64(row.H-sidebarThreadRowH)) > 0.5 {
		t.Errorf("the session row is %.1f high, want the reference's %d", row.H, sidebarThreadRowH)
	}
	y := int(row.Y + row.H/2)
	ink := -1
	for x := int(row.X) + 1; x < int(row.X+row.W); x++ {
		if got := tt.Image().RGBAAt(x, y); !sameColor(got, k.sidebarAccent) {
			ink = x
			break
		}
	}
	if ink < 0 {
		t.Fatalf("the selected row paints no title")
	}
	want := row.X + k.sidebarPadX + sidebarThreadCell
	if math.Abs(float64(float32(ink)-want)) > 1 {
		t.Errorf("the row's first ink is at x=%d, want the title after its %.1f leading cell at x=%.1f: the slot is not the reference's",
			ink, k.sidebarPadX+sidebarThreadCell, want)
	}
}

// TestSidebarSearchRowShowsNoHint checks the reference's search entry: it is
// the word alone, with no shortcut hint at the row's tail. The window's own
// palette shortcut stays wired in the shell, as the search palette tests check.
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
// 2x, so pixels ÷ 2 are DIPs): the four entries, the two headings, the project
// heading, the session row and the account footer. Each must land within 3 DIPs,
// the session row keeps the reference's 28 DIP face, and the account row must
// still sit on the sidebar's bottom edge, so the room is added above the list
// rather than by pushing the footer up. The built-in workspace's message room
// stays above the project group even though this shell draws no message there:
// without it the rule and the projects would sit a message's height high.
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
		{"dimagent-mygo", 314.0},
		{"Explain what this repository does in one paragraph.", 345.8},
		{tr(LocaleZHCN, keySidebarAccount), 698.0},
	}
	for _, tc := range cases {
		box := textBox(t, tt, tc.label)
		if center := float64(box.Y + box.H/2); math.Abs(center-tc.want) > 3 {
			t.Errorf("%q centers at %.1f, want the reference's %.1f ±3", tc.label, center, tc.want)
		}
	}

	// The selected session keeps the reference's row face: 28 DIPs from y≈331.5.
	row := textBox(t, tt, "Explain what this repository does in one paragraph.")
	if math.Abs(float64(row.H-sidebarThreadRowH)) > 0.5 {
		t.Errorf("the session row is %.1f high, want the reference's %d", row.H, sidebarThreadRowH)
	}
	if math.Abs(float64(row.Y-331.5)) > 0.5 {
		t.Errorf("the session row starts at y=%.1f, want the reference's 331.5", row.Y)
	}

	k := newTokens(ui.LightTheme())
	account := textBox(t, tt, tr(LocaleZHCN, keySidebarAccount))
	if gap := height - (account.Y + account.H); math.Abs(float64(gap-k.sidebarPadX)) > 0.5 {
		t.Errorf("the account row leaves %.1f below it, want the sidebar's %.1f DIP edge", gap, k.sidebarPadX)
	}
}
