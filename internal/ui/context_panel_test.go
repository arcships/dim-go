package ui

import (
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// contextPanelTester renders one context panel in a window of width×height
// DIPs, with the shell's light tokens as its theme. The view is captured once:
// the panel writes only through the pointers and the map the view carries, so a
// click is visible to the caller and to the next frame in the same way. Call
// this again with a new view to check a different frame.
func contextPanelTester(v ContextPanelView, width, height int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		renderContextPanel(c, newTokens(c.Theme()), v, Actions{})
	}, width, height)
}

// contextPanelFiles is the tree the tests draw: two directories with one file
// each and a file at the top, so a fold hides and shows a nested row and the
// three statuses each appear once.
func contextPanelFiles() []ContextPanelFile {
	return []ContextPanelFile{
		{Path: "internal", Children: []ContextPanelFile{
			{Path: "internal/ui", Children: []ContextPanelFile{
				{Path: "internal/ui/context_panel.go", Status: ContextPanelChanged},
			}},
			{Path: "internal/appserver", Children: []ContextPanelFile{
				{Path: "internal/appserver/client.go", Status: ContextPanelAdded},
			}},
		}},
		{Path: "README.md", Status: ContextPanelDeleted},
	}
}

// contextPanelDiffLines is a diff with all three line kinds. The second pair shares
// line number 13, as a replaced line does: the old side numbers the removed
// line and the new side numbers the added one.
func contextPanelDiffLines() []ContextPanelDiffLine {
	return []ContextPanelDiffLine{
		{Kind: ContextPanelDiffContext, Text: "func main() {", OldLine: 12, NewLine: 12},
		{Kind: ContextPanelDiffRemoved, Text: "old statement", OldLine: 13},
		{Kind: ContextPanelDiffAdded, Text: "new statement", NewLine: 13},
		{Kind: ContextPanelDiffAdded, Text: "second new statement", NewLine: 14},
	}
}

// fullContextPanelView is a reported tree with a selected file, its diff and a
// caller-held fold map. selected is a pointer into the caller's own state, so a
// test can read what a click wrote.
func fullContextPanelView(selected *string) ContextPanelView {
	return ContextPanelView{
		Files:         contextPanelFiles(),
		FilesReported: true,
		Selected:      selected,
		Expanded:      map[string]*bool{},
		Diff:          contextPanelDiffLines(),
		DiffReported:  true,
		Width:         900,
	}
}

// TestContextPanelDrawsTheFileTree checks the reading the panel exists for: the
// changed-file tree with every node, its directory levels and folds, and the
// three status letters, above the selected file's detail.
func TestContextPanelDrawsTheFileTree(t *testing.T) {
	selected := "internal/ui/context_panel.go"
	v := fullContextPanelView(&selected)
	tt, k := contextPanelThemedTester(v, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); !ok {
		t.Fatalf("the panel is not a named region, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleTree, "Changed files"); !ok {
		t.Fatalf("the file tree is not drawn, texts %q", tt.Texts())
	}
	for _, want := range []string{
		"Changes", "3 files",
		"internal", "ui", "appserver",
		"context_panel.go", "client.go", "README.md",
		// One status letter per kind, as git's.
		"M", "A", "D",
		// The selected file's detail: the full path and its status.
		"internal/ui/context_panel.go", "changed",
		"func main() {", "old statement", "new statement",
	} {
		if !tt.HasText(want) {
			t.Errorf("context panel is missing %q, texts %q", want, tt.Texts())
		}
	}

	for _, want := range []struct {
		name  string
		level int
	}{
		{"internal", 1}, {"ui", 2}, {"context_panel.go", 3}, {"README.md", 1},
	} {
		info, ok := tt.AccessByRole(ui.RoleTreeItem, want.name)
		if !ok {
			t.Errorf("%s is not a tree item, texts %q", want.name, tt.Texts())
			continue
		}
		if info.Level != want.level {
			t.Errorf("%s is at level %d, want %d", want.name, info.Level, want.level)
		}
	}

	dir, ok := tt.AccessByRole(ui.RoleTreeItem, "internal")
	if !ok || !dir.Expandable || !dir.Expanded {
		t.Errorf("internal is not an expanded directory: %+v, found %v", dir, ok)
	}
	// A row is one tree step high, and each directory level indents its row by
	// one step, so a nested row's ink starts one indent right of its parent's.
	if info, ok := tt.AccessByRole(ui.RoleTreeItem, "README.md"); !ok || info.Bounds.H != contextPanelRowH {
		t.Errorf("a top-level row is %.1f high, want %d: %+v, found %v",
			info.Bounds.H, contextPanelRowH, info, ok)
	}
	inkX := func(name string) float32 {
		info, ok := tt.AccessByRole(ui.RoleTreeItem, name)
		if !ok {
			t.Fatalf("%q is not a tree item, texts %q", name, tt.Texts())
		}
		ink, pixels := glyphInkBox(tt, info.Bounds, k.background)
		if pixels == 0 {
			t.Fatalf("%q paints no ink", name)
		}
		return ink.X
	}
	if step := inkX("ui") - inkX("internal"); step != contextPanelIndent {
		t.Errorf("one directory level indents by %.1f, want %d", step, contextPanelIndent)
	}
	file, ok := tt.AccessByRole(ui.RoleTreeItem, "context_panel.go")
	if !ok {
		t.Fatalf("the selected file is not a tree item, texts %q", tt.Texts())
	}
	if file.Expandable {
		t.Errorf("a file is drawn as expandable: %+v", file)
	}
	if !file.Selected {
		t.Errorf("the selected file is not marked selected: %+v", file)
	}
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Selected file"); !ok {
		t.Errorf("the detail is not a named group, texts %q", tt.Texts())
	}
	if tt.HasText("No changed files") {
		t.Errorf("the panel claims a clean workspace while it draws files, texts %q", tt.Texts())
	}
	if tt.HasText("WIP") {
		t.Errorf("the panel is WIP while it has facts, texts %q", tt.Texts())
	}
}

// TestContextPanelCollapsesDirectories checks a directory's fold: a click
// closes it, its children leave the tree, the caller's fold state records it,
// and a second click opens it again.
func TestContextPanelCollapsesDirectories(t *testing.T) {
	v := ContextPanelView{
		Files:         contextPanelFiles(),
		FilesReported: true,
		Expanded:      map[string]*bool{},
		Width:         900,
	}
	tt := contextPanelTester(v, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleTreeItem, "context_panel.go"); !ok {
		t.Fatalf("the nested file is not drawn before the fold, texts %q", tt.Texts())
	}
	if err := tt.Click("internal"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("context_panel.go") || tt.HasText("client.go") {
		t.Fatalf("a closed directory kept its children, texts %q", tt.Texts())
	}
	info, ok := tt.AccessByRole(ui.RoleTreeItem, "internal")
	if !ok || info.Expanded {
		t.Errorf("internal is not reported closed: %+v, found %v", info, ok)
	}
	if info.Description != "2 files" {
		t.Errorf("the collapsed directory says %q, want the 2 files it holds", info.Description)
	}
	open, ok := v.Expanded["internal"]
	if !ok || open == nil || *open {
		t.Errorf("the click did not record the closed fold in the caller's map: %v, %v", open, ok)
	}

	if err := tt.Click("internal"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("context_panel.go") || !tt.HasText("client.go") {
		t.Fatalf("a reopened directory kept its children hidden, texts %q", tt.Texts())
	}
	if open := v.Expanded["internal"]; open == nil || !*open {
		t.Errorf("the second click did not reopen the caller's fold: %v", open)
	}
}

// TestContextPanelSelectsAFile checks a file click: the caller's selection
// pointer moves, the new row is marked selected, the old one is not, and the
// detail follows with the file's path and status.
func TestContextPanelSelectsAFile(t *testing.T) {
	selected := "internal/ui/context_panel.go"
	v := fullContextPanelView(&selected)
	tt := contextPanelTester(v, 900, 600)

	if err := tt.Click("client.go"); err != nil {
		t.Fatal(err)
	}
	if selected != "internal/appserver/client.go" {
		t.Fatalf("the click selected %q, want internal/appserver/client.go", selected)
	}
	chosen, ok := tt.AccessByRole(ui.RoleTreeItem, "client.go")
	if !ok || !chosen.Selected {
		t.Errorf("the clicked file is not marked selected: %+v, found %v", chosen, ok)
	}
	previous, ok := tt.AccessByRole(ui.RoleTreeItem, "context_panel.go")
	if !ok || previous.Selected {
		t.Errorf("the file clicked before is still marked selected: %+v, found %v", previous, ok)
	}
	if !tt.HasText("internal/appserver/client.go") || !tt.HasText("added") {
		t.Errorf("the detail did not follow the selection, texts %q", tt.Texts())
	}
}

// TestContextPanelDrawsTheThreeDiffKinds checks one row per line: the code
// text, both line-number columns, the +/− marks, and no cell invented for the
// side a line has no number on.
func TestContextPanelDrawsTheThreeDiffKinds(t *testing.T) {
	selected := "internal/ui/context_panel.go"
	tt := contextPanelTester(fullContextPanelView(&selected), 900, 600)

	for _, want := range []string{
		"func main() {", "old statement", "new statement", "second new statement",
		// Old and new line numbers, including the shared 13 of a replaced line.
		"12", "13", "14",
		// The gutter marks: the added and the removed line, and no mark for
		// context beyond the empty cell.
		"+", "-",
	} {
		if !tt.HasText(want) {
			t.Errorf("diff is missing %q, texts %q", want, tt.Texts())
		}
	}

	// The gutter marks and numbers are hidden from assistive technology: the
	// line's text is what it reads, not the drawing's columns.
	for _, hidden := range []string{"+", "-", "12", "13", "14"} {
		if _, ok := tt.Access(hidden); ok {
			t.Errorf("%q is read to assistive technology, texts %q", hidden, tt.Texts())
		}
	}
}

// TestContextPanelDrawsAPreview checks the preview when no diff was reported:
// the caller's text, drawn as given, with no diff gutter or invented line.
func TestContextPanelDrawsAPreview(t *testing.T) {
	selected := "internal/ui/context_panel.go"
	v := fullContextPanelView(&selected)
	v.Diff, v.DiffReported = nil, false
	v.Preview = "package ui\n\nfunc main() {}\n"
	v.PreviewReported = true
	tt := contextPanelTester(v, 900, 600)

	if !tt.HasText("package ui") || !tt.HasText("func main() {}") {
		t.Fatalf("the preview is not drawn, texts %q", tt.Texts())
	}
	if tt.HasText("No diff lines.") || tt.HasText("WIP") {
		t.Errorf("the panel still claims a missing detail, texts %q", tt.Texts())
	}
}

// TestContextPanelEmptyState checks a reported workspace with no change: the
// reference's two lines, the title over the muted detail, and no file drawn,
// rather than a WIP statement or an invented row.
func TestContextPanelEmptyState(t *testing.T) {
	v := ContextPanelView{FilesReported: true, Width: 900}
	tt := contextPanelTester(v, 900, 600)

	for _, want := range []string{"No changed files", "The workspace is clean."} {
		if !tt.HasText(want) {
			t.Errorf("the empty state is missing %q, texts %q", want, tt.Texts())
		}
	}
	if !tt.HasText("0 files") {
		t.Errorf("the header does not count zero files, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleTree, "Changed files"); ok {
		t.Errorf("an empty workspace draws a file tree, texts %q", tt.Texts())
	}
	if tt.HasText("WIP") {
		t.Errorf("an empty workspace is drawn as a WIP gap, texts %q", tt.Texts())
	}
}

// TestContextPanelWIPWithoutFacts checks the missing-data state: without
// changed-file facts the whole panel is a WIP statement, not a tree and not
// "No changed files", because the panel cannot tell a clean workspace from a
// read it never received.
func TestContextPanelWIPWithoutFacts(t *testing.T) {
	tt := contextPanelTester(ContextPanelView{Width: 900}, 900, 600)

	if !tt.HasText("WIP") || !tt.HasText(reasonContextFiles) {
		t.Fatalf("the panel does not explain its missing data, texts %q", tt.Texts())
	}
	if !tt.HasText("Changes") {
		t.Errorf("the WIP panel does not carry the header's title, texts %q", tt.Texts())
	}
	if tt.HasText("No changed files") {
		t.Errorf("the panel claims a clean workspace without facts, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleTree, "Changed files"); ok {
		t.Errorf("the panel draws a tree without facts, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); !ok {
		t.Errorf("the WIP panel is not a named region, texts %q", tt.Texts())
	}

	// The title keeps the header's size, so the panel does not re-type itself
	// when the caller's facts arrive.
	wipTitle, ok := tt.Find("Changes")
	if !ok {
		t.Fatalf("the WIP panel does not draw %q, texts %q", "Changes", tt.Texts())
	}
	reported := contextPanelTester(ContextPanelView{FilesReported: true, Width: 900}, 900, 600)
	headerTitle, ok := reported.Find("Changes")
	if !ok {
		t.Fatalf("the reported panel does not draw %q, texts %q", "Changes", reported.Texts())
	}
	if math.Abs(float64(wipTitle.H-headerTitle.H)) > 0.5 {
		t.Errorf("the WIP title is %v high, the header's %v: the panel changes type between states",
			wipTitle.H, headerTitle.H)
	}
}

// TestContextPanelDetailWIPWithoutDiffOrPreview checks a selected file whose
// detail the caller has not read: the tree stays, the detail names the gap, and
// a reported-but-empty diff says so instead of borrowing a preview.
func TestContextPanelDetailWIPWithoutDiffOrPreview(t *testing.T) {
	selected := "internal/ui/context_panel.go"
	v := fullContextPanelView(&selected)
	v.Diff, v.DiffReported = nil, false
	tt := contextPanelTester(v, 900, 600)

	if !tt.HasText(reasonContextDetail) {
		t.Fatalf("the detail does not explain its missing data, texts %q", tt.Texts())
	}
	if !tt.HasText("WIP") {
		t.Errorf("the detail carries no WIP badge, texts %q", tt.Texts())
	}

	v.Preview = "a preview the caller did pass"
	v.PreviewReported = true
	v.DiffReported = true
	v.Diff = nil
	tt = contextPanelTester(v, 900, 600)
	if !tt.HasText("No diff lines.") {
		t.Fatalf("an empty reported diff is not named, texts %q", tt.Texts())
	}
	if tt.HasText(v.Preview) {
		t.Errorf("the detail fell back to a preview while it had a diff read, texts %q", tt.Texts())
	}
}

// TestContextPanelKeyboardReachability checks that Tab reaches the tree and
// that, without a pointer, Down walks the items, Enter selects a file, and Left
// and Right close and open a directory.
func TestContextPanelKeyboardReachability(t *testing.T) {
	selected := ""
	v := ContextPanelView{
		Files:         contextPanelFiles(),
		FilesReported: true,
		Selected:      &selected,
		Expanded:      map[string]*bool{},
		Width:         900,
	}
	tt := contextPanelTester(v, 900, 600)

	tt.Key(0, ui.KeyTab)
	if !tt.Focused("internal") {
		t.Fatalf("Tab does not reach the tree's first item, texts %q", tt.Texts())
	}
	for _, step := range []struct {
		key   ui.Key
		focus string
	}{
		{ui.KeyDown, "ui"},
		{ui.KeyDown, "context_panel.go"},
	} {
		tt.Key(0, step.key)
		if !tt.Focused(step.focus) {
			t.Fatalf("%v did not move the focus to %s, texts %q", step.key, step.focus, tt.Texts())
		}
	}
	tt.Key(0, ui.KeyEnter)
	if selected != "internal/ui/context_panel.go" {
		t.Fatalf("Enter on the file selected %q, want internal/ui/context_panel.go", selected)
	}

	tt.Key(0, ui.KeyHome)
	if !tt.Focused("internal") {
		t.Fatalf("Home did not return to the first item, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyLeft)
	if info, ok := tt.AccessByRole(ui.RoleTreeItem, "internal"); !ok || info.Expanded {
		t.Fatalf("Left did not close the directory: %+v, found %v", info, ok)
	}
	tt.Key(0, ui.KeyRight)
	if info, ok := tt.AccessByRole(ui.RoleTreeItem, "internal"); !ok || !info.Expanded {
		t.Fatalf("Right did not open the directory: %+v, found %v", info, ok)
	}
}

// TestContextPanelNarrowWidthDropsLineNumbers checks the narrow step: below the
// panel's own threshold the diff keeps its code and drops the two number
// columns, and a long path stays on one row inside the panel instead of growing
// the tree past it. The same view at the reference's width keeps the numbers,
// so the step is a choice about room rather than a column that is never drawn,
// and a caller below the panel's floor is judged at that floor.
func TestContextPanelNarrowWidthDropsLineNumbers(t *testing.T) {
	longPath := "internal/ui/a_very_long_context_panel_file_name_that_cannot_fit.go"
	selected := longPath
	newView := func(width float32) ContextPanelView {
		return ContextPanelView{
			Files: []ContextPanelFile{
				{Path: "internal", Children: []ContextPanelFile{
					{Path: longPath, Status: ContextPanelChanged},
				}},
			},
			FilesReported: true,
			Selected:      &selected,
			Expanded:      map[string]*bool{},
			Diff:          contextPanelDiffLines(),
			DiffReported:  true,
			Width:         width,
		}
	}
	numbersShown := func(tt *ui.Tester) bool {
		for _, number := range []string{"12", "13", "14"} {
			if !tt.HasText(number) {
				return false
			}
		}
		return true
	}

	tt := contextPanelTester(newView(contextPanelNarrowW-1), int(contextPanelNarrowW)-1, 600)

	if !tt.HasText("func main() {") || !tt.HasText("old statement") {
		t.Fatalf("the narrow diff lost its code, texts %q", tt.Texts())
	}
	for _, number := range []string{"12", "13", "14"} {
		if tt.HasText(number) {
			t.Errorf("a narrow panel still draws line number %q, texts %q", number, tt.Texts())
		}
	}
	row, ok := tt.Find(ContextPanelFile{Path: longPath}.Name())
	if !ok {
		t.Fatalf("the long file name is not drawn, texts %q", tt.Texts())
	}
	if row.W > float32(contextPanelNarrowW) {
		t.Errorf("the long file name is %.1f wide in a %d-wide panel", row.W, contextPanelNarrowW)
	}
	if row.H > contextPanelRowH {
		t.Errorf("the long file name is %.1f high, want one %d-high row", row.H, contextPanelRowH)
	}

	// The floor is the narrowest the panel draws: the code keeps the room and
	// the assigned cell is not invented.
	tt = contextPanelTester(newView(contextPanelMinW), int(contextPanelMinW), 600)
	if !tt.HasText("new statement") || !tt.HasText("+") || !tt.HasText("-") {
		t.Errorf("the panel at its floor lost its code or its signs, texts %q", tt.Texts())
	}
	if numbersShown(tt) {
		t.Errorf("the panel at its floor still draws its line numbers, texts %q", tt.Texts())
	}
	row, ok = tt.Find(ContextPanelFile{Path: longPath}.Name())
	if !ok {
		t.Fatalf("the long file name is not drawn at the floor, texts %q", tt.Texts())
	}
	if row.W > contextPanelMinW || row.H > contextPanelRowH {
		t.Errorf("the long file name is %.1fx%.1f in a %d-wide panel, want one %d-high row",
			row.W, row.H, contextPanelMinW, contextPanelRowH)
	}

	// A caller that asks for less than the floor is judged at the floor.
	tt = contextPanelTester(newView(contextPanelMinW-40), int(contextPanelMinW), 600)
	if numbersShown(tt) {
		t.Errorf("a below-floor width still draws the numbers, texts %q", tt.Texts())
	}

	// An unmeasured width draws every column, as the reference does.
	tt = contextPanelTester(newView(0), 900, 600)
	if !numbersShown(tt) {
		t.Errorf("an unmeasured width dropped the numbers, texts %q", tt.Texts())
	}

	// The reference's width keeps the numbers.
	tt = contextPanelTester(newView(contextPanelW), int(contextPanelW), 600)
	if !numbersShown(tt) {
		t.Errorf("the %d-wide panel dropped the numbers, texts %q", contextPanelW, tt.Texts())
	}
}

// TestContextPanelWidthBounds checks the panel's width contract: the reference's
// 320 when the caller names no width, and the 240..400 bounds otherwise. The
// shell asks for the reference width; a future resize or preference is what the
// bounds keep from taking the conversation's room.
func TestContextPanelWidthBounds(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   float32
		want float32
	}{
		{"unmeasured", 0, contextPanelW},
		{"negative", -12, contextPanelW},
		{"not a number", float32(math.NaN()), contextPanelW},
		{"below the floor", 100, contextPanelMinW},
		{"at the floor", contextPanelMinW, contextPanelMinW},
		{"the reference", contextPanelW, contextPanelW},
		{"at the ceiling", contextPanelMaxW, contextPanelMaxW},
		{"above the ceiling", 900, contextPanelMaxW},
	} {
		if got := clampContextPanelWidth(tc.in); got != tc.want {
			t.Errorf("%s: clampContextPanelWidth(%v) = %v, want %v", tc.name, tc.in, got, tc.want)
		}
	}
}

// contextPanelThemedTester renders the panel while keeping the tokens it drew
// with, so a test can compare the faces and tints it actually painted.
func contextPanelThemedTester(v ContextPanelView, width, height int) (*ui.Tester, tokens) {
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		renderContextPanel(c, newTokens(theme), v, Actions{})
	}, width, height)
	return tt, newTokens(theme)
}

// contextPanelRowFace samples a tree row inside its start padding, left of the
// chevron or the status letter, so no glyph colours the sample.
func contextPanelRowFace(t *testing.T, tt *ui.Tester, name string) color.RGBA {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleTreeItem, name)
	if !ok {
		t.Fatalf("%q is not a tree item, texts %q", name, tt.Texts())
	}
	return tt.Image().RGBAAt(int(info.Bounds.X)+2, int(info.Bounds.Y+info.Bounds.H/2))
}

// TestContextPanelRowStates checks the three faces of a tree row: the selected
// file under the accent, a plain row on the panel's own face, and both the row
// under the pointer and the row the keyboard focus is on under the hover tint,
// so a keyboard user can see where the focus is.
func TestContextPanelRowStates(t *testing.T) {
	selected := "internal/ui/context_panel.go"
	tt, k := contextPanelThemedTester(fullContextPanelView(&selected), 900, 600)

	if got := contextPanelRowFace(t, tt, "context_panel.go"); !sameColor(got, k.accent) {
		t.Errorf("the selected file's face is %v, want the accent %+v", got, k.accent)
	}
	if got := contextPanelRowFace(t, tt, "client.go"); !sameColor(got, k.background) {
		t.Errorf("an unselected row's face is %v, want the panel's %+v", got, k.background)
	}

	chosen, ok := tt.AccessByRole(ui.RoleTreeItem, "client.go")
	if !ok {
		t.Fatalf("client.go is not a tree item, texts %q", tt.Texts())
	}
	tt.Move(chosen.Bounds.X+chosen.Bounds.W/2, chosen.Bounds.Y+chosen.Bounds.H/2)
	tt.Frame()
	if got := contextPanelRowFace(t, tt, "client.go"); !sameColor(got, k.sidebarHover) {
		t.Errorf("the pointed row's face is %v, want the hover tint %+v", got, k.sidebarHover)
	}

	tt.Move(1, 1)
	tt.Frame()
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("internal") {
		t.Fatalf("Tab does not reach the tree's first item, texts %q", tt.Texts())
	}
	if got := contextPanelRowFace(t, tt, "internal"); !sameColor(got, k.sidebarHover) {
		t.Errorf("the focused row's face is %v, want the hover tint %+v", got, k.sidebarHover)
	}
}

// TestContextPanelDiffRowTints checks the diff's +/− colors: an added line is
// washed with the success color, a removed line with the danger color, and an
// unchanged line keeps the panel's face.
func TestContextPanelDiffRowTints(t *testing.T) {
	selected := "internal/ui/context_panel.go"
	tt, k := contextPanelThemedTester(fullContextPanelView(&selected), 900, 600)

	detail, ok := tt.AccessByRole(ui.RoleGroup, "Selected file")
	if !ok {
		t.Fatalf("the detail is not a named group, texts %q", tt.Texts())
	}
	// Sample each row's face just inside its left padding, where no line
	// number, sign or code glyph is drawn.
	x := int(detail.Bounds.X+contextPanelPadX) + 1
	img := tt.Image()
	rowFace := func(text string) color.RGBA {
		t.Helper()
		box, ok := tt.Find(text)
		if !ok {
			t.Fatalf("%q is not drawn, texts %q", text, tt.Texts())
		}
		return img.RGBAAt(x, int(box.Y+box.H/2))
	}

	if got := rowFace("func main() {"); !sameColor(got, k.background) {
		t.Errorf("a context line's face is %v, want the panel's %+v", got, k.background)
	}
	if got, want := rowFace("new statement"), k.wash(k.success).Over(k.background); !sameColor(got, want) {
		t.Errorf("an added line's face is %v, want the success wash %+v", got, want)
	}
	if got, want := rowFace("old statement"), k.wash(k.danger).Over(k.background); !sameColor(got, want) {
		t.Errorf("a removed line's face is %v, want the danger wash %+v", got, want)
	}
}

// TestContextPanelNamesTheDiffAndPreview checks that each detail read is a named
// group, so assistive technology says whether the panel is showing a diff or a
// preview rather than one unnamed block of code.
func TestContextPanelNamesTheDiffAndPreview(t *testing.T) {
	selected := "internal/ui/context_panel.go"
	tt := contextPanelTester(fullContextPanelView(&selected), 900, 600)
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Diff"); !ok {
		t.Errorf("the diff is not a named group, texts %q", tt.Texts())
	}

	v := fullContextPanelView(&selected)
	v.Diff, v.DiffReported = nil, false
	v.Preview, v.PreviewReported = "package ui\n", true
	tt = contextPanelTester(v, 900, 600)
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Preview"); !ok {
		t.Errorf("the preview is not a named group, texts %q", tt.Texts())
	}
}

// TestContextPanelAsksForASelection checks a reported tree with no selection:
// the detail asks for one, and claims neither a diff nor a missing read, while
// the tree stays usable.
func TestContextPanelAsksForASelection(t *testing.T) {
	selected := ""
	v := fullContextPanelView(&selected)
	v.Diff, v.DiffReported = nil, false
	tt := contextPanelTester(v, 900, 600)

	if !tt.HasText("Select a file to see its diff.") {
		t.Fatalf("the detail does not ask for a selection, texts %q", tt.Texts())
	}
	if tt.HasText("WIP") || tt.HasText("No diff lines.") {
		t.Errorf("an unselected detail claims a read it never had, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleTree, "Changed files"); !ok {
		t.Errorf("the tree is not drawn beside the request, texts %q", tt.Texts())
	}
}
