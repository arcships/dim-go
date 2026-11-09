package ui

import (
	"image/color"
	"math"
	"strings"
	"testing"

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

	if err := tt.Click("Second session"); err != nil {
		t.Fatal(err)
	}
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
		"Update":          "updater",
		"Search sessions": "search",
		"Group":           "project",
		"Pin":             "pin",
		"Rename":          "session.rename",
		"Archive":         "hidden session list",
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

// TestSidebarSelectsABranch checks the branch rows once the window wires the
// action: a click on another branch asks for it exactly once, the region writes
// no selection of its own, and the rows mark the branch the view names. The
// caller moves SelectedBranch only after the App Server opened it.
func TestSidebarSelectsABranch(t *testing.T) {
	s := fixture()
	s.Branches = []Branch{{ID: "main", Label: "main"}, {ID: "side", Label: "side"}}
	s.SelectedBranch = "main"
	var got []string
	s.OnSelectBranch = func(branchID string) { got = append(got, branchID) }
	tt := ui.NewTester(s.View, 900, 600)

	mainRow, ok := tt.AccessByRole(ui.RoleListItem, "main")
	if !ok {
		t.Fatalf("the current branch is not exposed as a row, texts %q", tt.Texts())
	}
	if !mainRow.Selected {
		t.Error("the current branch is not marked selected")
	}
	sideRow, ok := tt.AccessByRole(ui.RoleListItem, "side")
	if !ok {
		t.Fatalf("the other branch is not exposed as a row, texts %q", tt.Texts())
	}
	if sideRow.Selected {
		t.Error("an unselected branch is marked selected")
	}

	if err := tt.Click("side"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "side" {
		t.Fatalf("OnSelectBranch ran with %q, want [side]", got)
	}
	if s.SelectedBranch != "main" {
		t.Fatalf("the region moved SelectedBranch to %q before the App Server opened it", s.SelectedBranch)
	}
	if info, ok := tt.AccessByRole(ui.RoleListItem, "side"); !ok || info.Selected {
		t.Errorf("the clicked row is marked selected before the open: %+v, found %v", info, ok)
	}

	// The caller publishes the opened branch; the rows follow it.
	s.SelectedBranch = "side"
	tt.Frame()
	if info, ok := tt.AccessByRole(ui.RoleListItem, "side"); !ok || !info.Selected {
		t.Errorf("the opened branch is not marked selected: %+v, found %v", info, ok)
	}
	if info, ok := tt.AccessByRole(ui.RoleListItem, "main"); !ok || info.Selected {
		t.Errorf("the previous branch is still marked selected: %+v, found %v", info, ok)
	}

	// A click on the branch already shown asks for no read.
	if err := tt.Click("side"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("a click on the shown branch ran OnSelectBranch %d times, want 1", len(got))
	}
}

// TestSidebarBranchWithoutCallbackStaysReadOnly checks the WIP form: without
// the callback the branch list keeps its read-only group, its reason and no
// selectable row, so an unwired window offers no switch it cannot perform.
func TestSidebarBranchWithoutCallbackStaysReadOnly(t *testing.T) {
	s := fixture()
	s.Branches = []Branch{{ID: "main", Label: "main"}, {ID: "side", Label: "side"}}
	s.SelectedBranch = "main"
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Branches (read-only)"); !ok {
		t.Errorf("branches are not grouped as read-only, texts %q", tt.Texts())
	}
	if !tt.HasText(reasonBranches) {
		t.Errorf("branch WIP reason is missing, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleListItem, "side"); ok {
		t.Error("an unwired branch list exposes a selectable row")
	}
	if err := tt.Click("side"); err != nil {
		t.Fatal(err)
	}
	if s.SelectedBranch != "main" {
		t.Fatalf("a read-only branch click changed SelectedBranch to %q", s.SelectedBranch)
	}
}

// TestSidebarBranchesWithoutDirectoryDrawNothing checks that a session with no
// branch directory draws no branch section: the shell invents no branch row and
// offers no switch for a directory the App Server has not supplied.
func TestSidebarBranchesWithoutDirectoryDrawNothing(t *testing.T) {
	s := fixture()
	s.OnSelectBranch = func(string) { t.Error("a branch row was offered without a directory") }
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Branches"); ok {
		t.Errorf("an empty branch directory draws a section, texts %q", tt.Texts())
	}
	if tt.HasText(reasonBranches) {
		t.Errorf("an empty branch directory shows the WIP reason, texts %q", tt.Texts())
	}
}

// TestSidebarEndsTheCurrentBranch checks the end entry: the displayed non-main
// branch offers it, a click asks the window to end exactly that branch, and the
// region deletes no row and moves no selection before the App Server confirms.
func TestSidebarEndsTheCurrentBranch(t *testing.T) {
	s := fixture()
	s.Branches = []Branch{
		{ID: "main", Label: "main", State: "open"},
		{ID: "side", Label: "side", State: "open"},
	}
	s.SelectedBranch = "side"
	var got []string
	s.OnEndBranch = func(branchID string) { got = append(got, branchID) }
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "End branch")
	if !ok || info.Disabled {
		t.Fatalf("the current branch offers no enabled End branch entry: %+v, found %v", info, ok)
	}
	if err := tt.Click("End branch"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "side" {
		t.Fatalf("OnEndBranch ran with %q, want [side]", got)
	}
	if len(s.Branches) != 2 {
		t.Fatalf("the sidebar deleted a branch row before the App Server confirmed: %#v", s.Branches)
	}
	if s.SelectedBranch != "side" {
		t.Fatalf("the sidebar moved the displayed branch to %q before the App Server confirmed", s.SelectedBranch)
	}
}

// TestSidebarEndsNoMainOrEndedBranch checks the entry's two App Server facts:
// main is the session's own line and an ended branch is already closed, so
// neither offers an end action.
func TestSidebarEndsNoMainOrEndedBranch(t *testing.T) {
	cases := []struct {
		name     string
		branches []Branch
		current  string
	}{
		{"main", []Branch{{ID: "main", Label: "main", State: "open"}}, "main"},
		{"ended", []Branch{
			{ID: "main", Label: "main", State: "open"},
			{ID: "side", Label: "side", State: "ended"},
		}, "side"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := fixture()
			s.Branches = tc.branches
			s.SelectedBranch = tc.current
			s.OnEndBranch = func(string) { t.Error("an endable row was offered where the directory does not allow it") }
			tt := ui.NewTester(s.View, 900, 600)

			if _, ok := tt.AccessByRole(ui.RoleButton, "End branch"); ok {
				t.Errorf("the %s branch offers End branch, texts %q", tc.name, tt.Texts())
			}
		})
	}
}

// TestSidebarEndBranchWithoutCallbackStaysWIP checks the WIP form: without the
// callback the entry keeps its place, is disabled and explains the missing
// session.branch_end wiring, so an unwired window cannot imply an end.
func TestSidebarEndBranchWithoutCallbackStaysWIP(t *testing.T) {
	s := fixture()
	s.Branches = []Branch{
		{ID: "main", Label: "main", State: "open"},
		{ID: "side", Label: "side", State: "open"},
	}
	s.SelectedBranch = "side"
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "End branch")
	if !ok {
		t.Fatalf("the current branch offers no End branch entry, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("End branch is enabled without a callback")
	}
	if !strings.Contains(info.Description, reasonEndBranch) {
		t.Errorf("the end entry description %q does not carry %q", info.Description, reasonEndBranch)
	}
	if err := tt.Click("End branch"); err != nil {
		t.Fatal(err)
	}
}

// TestSidebarRefreshesTheBranchDirectory checks the directory's refresh entry:
// a click asks the window to re-read session.branches exactly once, and the
// region keeps its rows for the caller to replace.
func TestSidebarRefreshesTheBranchDirectory(t *testing.T) {
	s := fixture()
	s.Branches = []Branch{
		{ID: "main", Label: "main", State: "open"},
		{ID: "side", Label: "side", State: "open"},
	}
	s.SelectedBranch = "main"
	calls := 0
	s.OnRefreshBranches = func() { calls++ }
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "Refresh")
	if !ok || info.Disabled {
		t.Fatalf("the branch directory offers no enabled Refresh entry: %+v, found %v", info, ok)
	}
	if err := tt.Click("Refresh"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("OnRefreshBranches ran %d times, want 1", calls)
	}
	if len(s.Branches) != 2 {
		t.Fatalf("the sidebar rewrote the directory itself: %#v", s.Branches)
	}
}

// TestSidebarRefreshWithoutCallbackStaysWIP checks that the refresh entry keeps
// its place and its reason while the window has no callback.
func TestSidebarRefreshWithoutCallbackStaysWIP(t *testing.T) {
	s := fixture()
	s.Branches = []Branch{{ID: "main", Label: "main", State: "open"}}
	s.SelectedBranch = "main"
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "Refresh")
	if !ok {
		t.Fatalf("the branch directory offers no Refresh entry, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("Refresh is enabled without a callback")
	}
	if !strings.Contains(info.Description, reasonRefreshBranches) {
		t.Errorf("the refresh entry description %q does not carry %q", info.Description, reasonRefreshBranches)
	}
}
