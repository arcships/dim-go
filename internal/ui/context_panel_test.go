package ui

import (
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
	tt := contextPanelTester(v, 900, 600)

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
// panel says so and draws no file, rather than a WIP statement or an invented
// row.
func TestContextPanelEmptyState(t *testing.T) {
	v := ContextPanelView{FilesReported: true, Width: 900}
	tt := contextPanelTester(v, 900, 600)

	if !tt.HasText("No changed files") {
		t.Fatalf("the empty state is missing, texts %q", tt.Texts())
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
	if tt.HasText("No changed files") {
		t.Errorf("the panel claims a clean workspace without facts, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleTree, "Changed files"); ok {
		t.Errorf("the panel draws a tree without facts, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); !ok {
		t.Errorf("the WIP panel is not a named region, texts %q", tt.Texts())
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
// the tree past it.
func TestContextPanelNarrowWidthDropsLineNumbers(t *testing.T) {
	longPath := "internal/ui/a_very_long_context_panel_file_name_that_cannot_fit.go"
	selected := longPath
	v := ContextPanelView{
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
		Width:         contextPanelNarrowW - 1,
	}
	tt := contextPanelTester(v, int(contextPanelNarrowW)-1, 600)

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

	// The same view at full width keeps the numbers, so the narrow step is a
	// choice about room rather than a column that is never drawn.
	v.Width = 900
	tt = contextPanelTester(v, 900, 600)
	for _, number := range []string{"12", "13", "14"} {
		if !tt.HasText(number) {
			t.Errorf("the wide panel dropped line number %q, texts %q", number, tt.Texts())
		}
	}
}
