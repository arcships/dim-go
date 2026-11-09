package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// keysPanelTester renders one keys panel in a window of width×height DIPs, with
// the shell's light tokens as its theme. The view is a pointer because the
// search field edits its Query and a click writes its Selected; the same view
// passed back gives the next frame the user's state.
func keysPanelTester(v *KeysPanelView, width, height int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		renderKeysPanel(c, newTokens(c.Theme()), v, Actions{})
	}, width, height)
}

// keysPanelEntries is the snapshot the tests draw: three keys whose names,
// input ids and text each give the filter something different to match.
func keysPanelEntries() []KeysPanelEntry {
	return []KeysPanelEntry{
		{Key: "task", Sequence: 12, InputID: "input-1", Text: "implement the keys panel"},
		{Key: "notes", Sequence: 9, InputID: "input-2", Text: "the daemon owns the snapshot"},
		{Key: "goroutine", Sequence: 4, InputID: "input-3", Text: "check the scheduler"},
	}
}

// fullKeysPanelView is a loaded snapshot with a caller-held selection.
func fullKeysPanelView(selected *string) *KeysPanelView {
	return &KeysPanelView{
		Entries:  keysPanelEntries(),
		Through:  12,
		Loaded:   true,
		Selected: selected,
	}
}

// TestKeysPanelDrawsTheSnapshot checks the reading the panel exists for: its
// title, the through of the read, the local filter, and one row per key with
// the sequence, the input id and the block text.
func TestKeysPanelDrawsTheSnapshot(t *testing.T) {
	selected := ""
	tt := keysPanelTester(fullKeysPanelView(&selected), 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleRegion, "Keys panel"); !ok {
		t.Fatalf("the panel is not a named region, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Search keys"); !ok {
		t.Fatalf("the filter is not a named search field, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleList, "Keys"); !ok {
		t.Fatalf("the keys are not a named list, texts %q", tt.Texts())
	}
	for _, want := range []string{
		"Keys", "3 keys", "through 12",
		"task", "12", "input-1", "implement the keys panel",
		"notes", "9", "input-2", "the daemon owns the snapshot",
		"goroutine", "4", "input-3", "check the scheduler",
	} {
		if !tt.HasText(want) {
			t.Errorf("keys panel is missing %q, texts %q", want, tt.Texts())
		}
	}
	if tt.HasText(wipBadge) {
		t.Errorf("a loaded snapshot is drawn as a WIP gap, texts %q", tt.Texts())
	}

	for _, key := range []string{"task", "notes", "goroutine"} {
		if _, ok := tt.AccessByRole(ui.RoleListItem, key); !ok {
			t.Errorf("%q is not a list item, texts %q", key, tt.Texts())
		}
	}
}

// TestKeysPanelFiltersTheDisplayCopy checks the local filter: it keeps the
// matching rows, drops the others, and leaves the caller's snapshot as it was
// instead of asking for another read or rewriting Entries.
func TestKeysPanelFiltersTheDisplayCopy(t *testing.T) {
	selected := ""
	v := fullKeysPanelView(&selected)
	v.Query = "daemon"
	tt := keysPanelTester(v, 900, 600)

	if !tt.HasText("notes") || !tt.HasText("the daemon owns the snapshot") {
		t.Fatalf("the filter dropped the matching key, texts %q", tt.Texts())
	}
	for _, gone := range []string{"task", "goroutine", "input-1", "input-3"} {
		if tt.HasText(gone) {
			t.Errorf("the filter kept %q, texts %q", gone, tt.Texts())
		}
	}
	if len(v.Entries) != 3 {
		t.Errorf("the filter changed the caller's snapshot: %d entries", len(v.Entries))
	}
	if !tt.HasText("3 keys") {
		t.Errorf("the header stopped counting the snapshot's entries, texts %q", tt.Texts())
	}

	// A query that matches nothing names the filter rather than claiming the
	// branch has no keys.
	v.Query = "nothing-matches"
	tt = keysPanelTester(v, 900, 600)
	if !tt.HasText(`No keys match "nothing-matches".`) {
		t.Fatalf("an empty filter result is not named, texts %q", tt.Texts())
	}
	if tt.HasText("No keys yet") {
		t.Errorf("an empty filter result claims the branch has no keys, texts %q", tt.Texts())
	}
}

// TestKeysPanelEmptyState checks a loaded snapshot with no keys: the panel
// says so and draws no row, rather than a WIP statement or an invented key.
func TestKeysPanelEmptyState(t *testing.T) {
	v := &KeysPanelView{Loaded: true, Through: 12}
	tt := keysPanelTester(v, 900, 600)

	if !tt.HasText("No keys yet") {
		t.Fatalf("the empty state is missing, texts %q", tt.Texts())
	}
	if !tt.HasText("0 keys") || !tt.HasText("through 12") {
		t.Errorf("the header does not report the snapshot's count and through, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleList, "Keys"); ok {
		t.Errorf("an empty snapshot draws a key list, texts %q", tt.Texts())
	}
	if tt.HasText(wipBadge) {
		t.Errorf("a loaded, empty snapshot is drawn as a WIP gap, texts %q", tt.Texts())
	}
}

// TestKeysPanelWIPWithoutSnapshot checks the missing-data state: without a
// session.keys read the whole panel is a WIP statement, not "No keys yet",
// because the panel cannot tell a branch with no keys from a read it never
// received. Entries the caller happened to pass are not drawn as facts.
func TestKeysPanelWIPWithoutSnapshot(t *testing.T) {
	v := &KeysPanelView{Entries: keysPanelEntries(), Through: 12}
	tt := keysPanelTester(v, 900, 600)

	if !tt.HasText(wipBadge) || !tt.HasText(reasonKeysUnloaded) {
		t.Fatalf("the panel does not explain its missing data, texts %q", tt.Texts())
	}
	if tt.HasText("No keys yet") {
		t.Errorf("the panel claims a branch with no keys without a read, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleList, "Keys"); ok {
		t.Errorf("the panel draws keys without a read, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, "Search keys"); ok {
		t.Errorf("the panel offers a filter without a read, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Keys panel"); !ok {
		t.Errorf("the WIP panel is not a named region, texts %q", tt.Texts())
	}
}

// TestKeysPanelSelectsARow checks a click: the caller's selection pointer
// moves, the new row is marked selected and the old one is not. The panel
// writes no other state.
func TestKeysPanelSelectsARow(t *testing.T) {
	selected := ""
	v := fullKeysPanelView(&selected)
	tt := keysPanelTester(v, 900, 600)

	if err := tt.Click("notes"); err != nil {
		t.Fatal(err)
	}
	if selected != "notes" {
		t.Fatalf("the click selected %q, want notes", selected)
	}
	chosen, ok := tt.AccessByRole(ui.RoleListItem, "notes")
	if !ok || !chosen.Selected {
		t.Errorf("the clicked key is not marked selected: %+v, found %v", chosen, ok)
	}
	previous, ok := tt.AccessByRole(ui.RoleListItem, "task")
	if !ok || previous.Selected {
		t.Errorf("the key selected before is still marked: %+v, found %v", previous, ok)
	}
	if v.Entries[1].Key != "notes" {
		t.Errorf("the click rewrote the snapshot: %+v", v.Entries[1])
	}
}

// TestKeysPanelClickWithoutSelectionKeepsNoState checks a caller that keeps no
// selection: a click writes nothing and does not invent a field for it.
func TestKeysPanelClickWithoutSelectionKeepsNoState(t *testing.T) {
	tt := keysPanelTester(fullKeysPanelView(nil), 900, 600)

	if err := tt.Click("task"); err != nil {
		t.Fatal(err)
	}
	if info, ok := tt.AccessByRole(ui.RoleListItem, "task"); !ok || info.Selected {
		t.Errorf("a nil Selected still marked a row: %+v, found %v", info, ok)
	}
}

// TestKeysPanelKeyboardReachability checks that Tab reaches the filter and the
// rows without a pointer, Enter selects the focused row, and typing in the
// filter narrows the list on the next frame.
func TestKeysPanelKeyboardReachability(t *testing.T) {
	selected := ""
	v := fullKeysPanelView(&selected)
	tt := keysPanelTester(v, 900, 600)

	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Search keys") {
		t.Fatalf("Tab does not reach the filter, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("task") {
		t.Fatalf("Tab does not reach the first key, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if selected != "task" {
		t.Fatalf("Enter on the focused key selected %q, want task", selected)
	}

	// The filter keeps the keyboard working: type a query, then Tab to the one
	// row it leaves and select it.
	if err := tt.Click("Search keys"); err != nil {
		t.Fatal(err)
	}
	tt.Type("goroutine")
	if v.Query != "goroutine" {
		t.Fatalf("typing left the query %q", v.Query)
	}
	if tt.HasText("task") || tt.HasText("notes") {
		t.Fatalf("the filter kept other keys, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("goroutine") {
		t.Fatalf("Tab does not reach the filtered key, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if selected != "goroutine" {
		t.Fatalf("Enter on the filtered key selected %q, want goroutine", selected)
	}
}

// TestKeysPanelTruncatesLongText checks a long block body: it stays one row,
// its whitespace is folded to single spaces, and text past the bound ends in
// an ellipsis rather than growing the row or disappearing.
func TestKeysPanelTruncatesLongText(t *testing.T) {
	long := "first line\nsecond line " + strings.Repeat("long", 100)
	v := &KeysPanelView{
		Entries: []KeysPanelEntry{
			{Key: "long", Sequence: 3, InputID: "input-9", Text: long},
		},
		Through: 3,
		Loaded:  true,
	}
	tt := keysPanelTester(v, 900, 600)

	if tt.HasText(long) {
		t.Fatalf("the row drew the whole long text, texts %q", tt.Texts())
	}
	want := keysPanelText(long)
	if !tt.HasText(want) {
		t.Fatalf("the row did not draw the bounded text %q, texts %q", want, tt.Texts())
	}
	if !strings.HasSuffix(want, "…") {
		t.Errorf("the bounded text %q has no ellipsis", want)
	}
	if !tt.HasText("first line second line ") {
		t.Errorf("the row did not fold the block's newline into one line, texts %q", tt.Texts())
	}
	row, ok := tt.Find(want)
	if !ok {
		t.Fatalf("the bounded text is not drawn, texts %q", tt.Texts())
	}
	if row.W > 900 {
		t.Errorf("the bounded text is %.1f wide in a 900-wide panel", row.W)
	}
}

// shellKeysState is the shell fixture with a loaded session.keys snapshot for
// the selected session, so the sidebar's "Session keys" entry is a real action.
func shellKeysState() *State {
	s := fixture()
	s.Keys = KeysPanelView{Entries: keysPanelEntries(), Through: 12, Loaded: true}
	return s
}

// TestShellOpensAndClosesTheKeysPanel checks the wiring: the sidebar entry is a
// real action while the window holds a snapshot, opening it swaps the main
// column for the browser, a row click writes only the shell's view state, and a
// second click returns to the conversation.
func TestShellOpensAndClosesTheKeysPanel(t *testing.T) {
	s := shellKeysState()
	tt := ui.NewTester(s.View, 900, 600)

	if tt.HasText("implement the keys panel") {
		t.Fatalf("the keys rows are drawn before the browser opens, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Keys panel"); ok {
		t.Fatalf("the keys panel is drawn before it is opened, texts %q", tt.Texts())
	}
	info, ok := tt.AccessByRole(ui.RoleButton, "Session keys")
	if !ok || info.Disabled {
		t.Fatalf("the keys entry is not an enabled action: %+v, found %v", info, ok)
	}

	if err := tt.Click("Session keys"); err != nil {
		t.Fatal(err)
	}
	if !s.ShowKeys {
		t.Fatal("the keys entry did not open the browser")
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Keys panel"); !ok {
		t.Fatalf("the opener did not draw the keys panel, texts %q", tt.Texts())
	}
	for _, want := range []string{"3 keys", "through 12", "implement the keys panel"} {
		if !tt.HasText(want) {
			t.Errorf("the browser is missing %q, texts %q", want, tt.Texts())
		}
	}
	if tt.HasText("hello from the fixture") {
		t.Errorf("the conversation is still drawn under the browser, texts %q", tt.Texts())
	}
	if entry, ok := tt.AccessByRole(ui.RoleButton, "Session keys"); !ok || !entry.Expanded {
		t.Errorf("the keys entry does not report its surface expanded: %+v, found %v", entry, ok)
	}

	// A row click moves the shell's selection and nothing else.
	if err := tt.Click("notes"); err != nil {
		t.Fatal(err)
	}
	if s.Keys.Selected == nil || *s.Keys.Selected != "notes" {
		t.Fatalf("the row click selected %v, want notes", s.Keys.Selected)
	}
	if s.Selected != "s1" || !s.ShowKeys {
		t.Errorf("the row click changed the session view: selected=%q showing=%v", s.Selected, s.ShowKeys)
	}

	// Clicking the entry again returns to the conversation.
	if err := tt.Click("Session keys"); err != nil {
		t.Fatal(err)
	}
	if s.ShowKeys {
		t.Fatal("the second click left the browser open")
	}
	tt.Frame()
	if !tt.HasText("hello from the fixture") {
		t.Errorf("the conversation did not come back, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Keys panel"); ok {
		t.Errorf("the browser is still drawn after closing, texts %q", tt.Texts())
	}
}

// TestKeysPanelClosesOnEscape checks the keyboard way back: Escape closes the
// browser and the frame that follows shows the conversation again.
func TestKeysPanelClosesOnEscape(t *testing.T) {
	s := shellKeysState()
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click("Session keys"); err != nil {
		t.Fatal(err)
	}
	if !s.ShowKeys {
		t.Fatal("the keys entry did not open the browser")
	}

	tt.Key(0, ui.KeyEscape)
	if s.ShowKeys {
		t.Fatal("Escape left the keys browser open")
	}
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Keys panel"); ok {
		t.Errorf("Escape did not return to the conversation, texts %q", tt.Texts())
	}
	if !tt.HasText("hello from the fixture") {
		t.Errorf("the conversation is missing after Escape, texts %q", tt.Texts())
	}
}

// TestShellKeysEntryIsWIPWithoutSnapshot checks the unwired state: without a
// session.keys read the entry keeps its place, dimmed, explained and inert, and
// no browser opens.
func TestShellKeysEntryIsWIPWithoutSnapshot(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "Session keys")
	if !ok {
		t.Fatalf("the keys entry is not drawn, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("the keys entry is enabled without a session.keys snapshot")
	}
	if !strings.Contains(info.Description, "key browser callback") {
		t.Errorf("the keys entry description %q does not name the missing wiring", info.Description)
	}
	if err := tt.Click("Session keys"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if s.ShowKeys {
		t.Error("clicking the WIP entry opened a browser with no snapshot")
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Keys panel"); ok {
		t.Errorf("a browser opened without a session.keys read, texts %q", tt.Texts())
	}
}

// TestShellKeysPanelFiltersTheSnapshot checks the filter through the shell: the
// panel edits State.Keys.Query in place, so the narrowed list survives the next
// frame and the caller's entries are untouched.
func TestShellKeysPanelFiltersTheSnapshot(t *testing.T) {
	s := shellKeysState()
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click("Session keys"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Search keys"); err != nil {
		t.Fatal(err)
	}
	tt.Type("daemon")
	if s.Keys.Query != "daemon" {
		t.Fatalf("the filter query is %q, want daemon", s.Keys.Query)
	}
	if _, ok := tt.AccessByRole(ui.RoleListItem, "notes"); !ok {
		t.Fatalf("the filter dropped the matching key, texts %q", tt.Texts())
	}
	for _, gone := range []string{"task", "goroutine"} {
		if _, ok := tt.AccessByRole(ui.RoleListItem, gone); ok {
			t.Fatalf("the filter kept the key %q, texts %q", gone, tt.Texts())
		}
	}
	if len(s.Keys.Entries) != 3 {
		t.Errorf("the filter rewrote the caller's snapshot: %d entries", len(s.Keys.Entries))
	}
}

// TestKeysPanelClosesOnSessionSwitch checks that a selection change closes the
// browser and drops its snapshot: one session's keys are never drawn for
// another, and the entry stays WIP until the window reads the new session's.
func TestKeysPanelClosesOnSessionSwitch(t *testing.T) {
	s := shellKeysState()
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click("Session keys"); err != nil {
		t.Fatal(err)
	}

	clickRow(t, tt, "Second session")
	if s.Selected != "s2" {
		t.Fatalf("selected %q, want s2", s.Selected)
	}
	if s.ShowKeys {
		t.Error("the browser stayed open after the session changed")
	}
	if s.Keys.Loaded || len(s.Keys.Entries) != 0 {
		t.Errorf("the previous session's snapshot survived the switch: %+v", s.Keys)
	}
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Keys panel"); ok {
		t.Errorf("the previous session's browser is drawn, texts %q", tt.Texts())
	}
	if tt.HasText("implement the keys panel") {
		t.Errorf("the previous session's keys are drawn for s2, texts %q", tt.Texts())
	}
	if info, ok := tt.AccessByRole(ui.RoleButton, "Session keys"); !ok || !info.Disabled {
		t.Errorf("the keys entry is a real action for a session with no snapshot: %+v, found %v", info, ok)
	}

	// The window's read for the new session fills the browser again, and only
	// that session's keys are shown.
	s.Keys = KeysPanelView{
		Entries: []KeysPanelEntry{{Key: "other", Sequence: 2, InputID: "input-9", Text: "second session key"}},
		Through: 2,
		Loaded:  true,
	}
	tt.Frame()
	if err := tt.Click("Session keys"); err != nil {
		t.Fatal(err)
	}
	if !s.ShowKeys {
		t.Fatal("the keys entry did not open the new session's browser")
	}
	if !tt.HasText("second session key") {
		t.Errorf("the new session's key is missing, texts %q", tt.Texts())
	}
	if tt.HasText("implement the keys panel") {
		t.Errorf("the old session's key is drawn for s2, texts %q", tt.Texts())
	}
}
