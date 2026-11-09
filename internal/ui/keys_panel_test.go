package ui

import (
	"strconv"
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

// keysPanelTextInfo returns what assistive technology sees of the text s in
// tt's last frame, failing the test when the frame does not draw it.
func keysPanelTextInfo(t *testing.T, tt *ui.Tester, s string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleText, s)
	if !ok {
		t.Fatalf("the text %q is not drawn, texts %q", s, tt.Texts())
	}
	return info
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
// the key heading, the block's value and the note that names the block.
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
	entries := keysPanelEntries()
	for _, want := range []string{
		"Keys", "3 keys", "through 12",
		entries[0].Key, keysPanelNote(entries[0]), entries[0].Text,
		entries[1].Key, keysPanelNote(entries[1]), entries[1].Text,
		entries[2].Key, keysPanelNote(entries[2]), entries[2].Text,
	} {
		if !tt.HasText(want) {
			t.Errorf("keys panel is missing %q, texts %q", want, tt.Texts())
		}
	}
	if tt.HasText(wipBadge) {
		t.Errorf("a loaded snapshot is drawn as a WIP gap, texts %q", tt.Texts())
	}

	for _, entry := range entries {
		row, ok := tt.AccessByRole(ui.RoleListItem, entry.Key)
		if !ok {
			t.Errorf("%q is not a list item, texts %q", entry.Key, tt.Texts())
			continue
		}
		if row.Bounds.H != keysPanelRowH {
			t.Errorf("%q is %.1f high, want the %d DIP row", entry.Key, row.Bounds.H, keysPanelRowH)
		}
	}
}

// TestKeysPanelRowSections checks the row's three parts and their places: the
// key heading and the note share the first line, the note ends at the row's
// right edge, and the value takes the second line inside the row's padding.
func TestKeysPanelRowSections(t *testing.T) {
	selected := ""
	tt := keysPanelTester(fullKeysPanelView(&selected), 900, 600)

	row, ok := tt.AccessByRole(ui.RoleListItem, "task")
	if !ok {
		t.Fatalf("the row is not a list item, texts %q", tt.Texts())
	}
	note, ok := tt.AccessByRole(ui.RoleText, keysPanelNote(keysPanelEntries()[0]))
	if !ok {
		t.Fatalf("the row's note is not drawn, texts %q", tt.Texts())
	}
	value, ok := tt.AccessByRole(ui.RoleText, "implement the keys panel")
	if !ok {
		t.Fatalf("the row's value is not drawn, texts %q", tt.Texts())
	}
	key, ok := tt.AccessByRole(ui.RoleText, "task")
	if !ok {
		t.Fatalf("the row's key heading is not drawn, texts %q", tt.Texts())
	}

	if key.Bounds.Y != note.Bounds.Y {
		t.Errorf("the heading is at y=%.1f and the note at y=%.1f, want one line",
			key.Bounds.Y, note.Bounds.Y)
	}
	if value.Bounds.Y <= note.Bounds.Y {
		t.Errorf("the value is at y=%.1f, not below the note's y=%.1f",
			value.Bounds.Y, note.Bounds.Y)
	}
	if key.Bounds.X < row.Bounds.X+keysPanelPadX || value.Bounds.X != key.Bounds.X {
		t.Errorf("the parts do not start at the row's %.1f DIP inset: key x=%.1f value x=%.1f row x=%.1f",
			float32(keysPanelPadX), key.Bounds.X, value.Bounds.X, row.Bounds.X)
	}
	if right := row.Bounds.X + row.Bounds.W - keysPanelPadX; note.Bounds.X+note.Bounds.W > right+0.5 {
		t.Errorf("the note ends at %.1f, past the row's right edge %.1f", note.Bounds.X+note.Bounds.W, right)
	}
	if value.Bounds.W != row.Bounds.W-2*keysPanelPadX {
		t.Errorf("the value is %.1f wide in a %.1f-wide row, want the row less its padding",
			value.Bounds.W, row.Bounds.W)
	}
}

// TestKeysPanelNarrowWindowClips checks a narrow main column: the row, the
// heading, the note and the value all stay inside the window and clip their
// single lines instead of widening the panel or wrapping onto a second row.
func TestKeysPanelNarrowWindowClips(t *testing.T) {
	const width = 320
	longKey := "a-rather-long-key-name-that-cannot-fit-into-a-narrow-keys-panel"
	longText := strings.Repeat("value ", 80) + "end"
	v := &KeysPanelView{
		Entries: []KeysPanelEntry{{
			Key: longKey, Sequence: 12345, InputID: "input-that-never-ends", Text: longText,
		}},
		Through: 12345,
		Loaded:  true,
	}
	tt := keysPanelTester(v, width, 600)

	row, ok := tt.AccessByRole(ui.RoleListItem, longKey)
	if !ok {
		t.Fatalf("the row is not a list item, texts %q", tt.Texts())
	}
	if row.Bounds.X < 0 || row.Bounds.X+row.Bounds.W > width {
		t.Errorf("the row spans %.1f–%.1f in a %d-wide window", row.Bounds.X, row.Bounds.X+row.Bounds.W, width)
	}
	if row.Bounds.H != keysPanelRowH {
		t.Errorf("the narrow row is %.1f high, want the %d DIP row", row.Bounds.H, keysPanelRowH)
	}

	key, ok := tt.AccessByRole(ui.RoleText, longKey)
	if !ok {
		t.Fatalf("the key heading is not drawn, texts %q", tt.Texts())
	}
	parts := []struct {
		name string
		text string
		info ui.AccessInfo
	}{
		{"key", longKey, key},
		{"note", keysPanelNote(v.Entries[0]), keysPanelTextInfo(t, tt, keysPanelNote(v.Entries[0]))},
		{"value", keysPanelText(longText), keysPanelTextInfo(t, tt, keysPanelText(longText))},
	}
	for _, part := range parts {
		if part.info.Bounds.X < row.Bounds.X || part.info.Bounds.X+part.info.Bounds.W > row.Bounds.X+row.Bounds.W {
			t.Errorf("the %s spans %.1f–%.1f, outside the row at %.1f–%.1f",
				part.name, part.info.Bounds.X, part.info.Bounds.X+part.info.Bounds.W,
				row.Bounds.X, row.Bounds.X+row.Bounds.W)
		}
		if part.info.Bounds.H != keysPanelLineH {
			t.Errorf("the %s is %.1f high, want one %d DIP line", part.name, part.info.Bounds.H, keysPanelLineH)
		}
	}

	// The same snapshot in a roomy window draws the whole key heading; the
	// narrow one clips it rather than pushing the note off the row.
	wideKey := keysPanelTextInfo(t, keysPanelTester(v, 900, 600), longKey)
	if key.Bounds.W >= wideKey.Bounds.W {
		t.Errorf("the narrow key heading is %.1f wide, not clipped from the wide %.1f",
			key.Bounds.W, wideKey.Bounds.W)
	}

	// The narrow row is still one click away, and the click still writes only
	// the caller's selection.
	selected := ""
	v.Selected = &selected
	if err := tt.Click(longKey); err != nil {
		t.Fatal(err)
	}
	if selected != longKey {
		t.Errorf("the narrow row's click selected %q, want %q", selected, longKey)
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

// TestKeysPanelNoteNamesTheBlock checks the note's two forms: the input that
// wrote the block and the block's sequence while the block names an input, and
// the sequence alone when it does not. The note joins the caller's fields and
// invents no third one.
func TestKeysPanelNoteNamesTheBlock(t *testing.T) {
	withInput := keysPanelNote(KeysPanelEntry{Sequence: 12, InputID: "input-1"})
	if withInput != "input-1 · seq 12" {
		t.Errorf("the note is %q, want the input and the sequence", withInput)
	}
	withoutInput := keysPanelNote(KeysPanelEntry{Sequence: 9})
	if withoutInput != "seq 9" {
		t.Errorf("the note without an input is %q, want the sequence alone", withoutInput)
	}
}

// keysPanelScrollEntries is a snapshot longer than a short panel: enough rows
// that the last ones start below the window and only the keyboard reaches them.
func keysPanelScrollEntries(n int) []KeysPanelEntry {
	entries := make([]KeysPanelEntry, 0, n)
	for i := 0; i < n; i++ {
		entries = append(entries, KeysPanelEntry{
			Key:      "key-" + strconv.Itoa(i),
			Sequence: int64(i + 1),
			InputID:  "input-" + strconv.Itoa(i),
			Text:     "value " + strconv.Itoa(i),
		})
	}
	return entries
}

// TestKeysPanelKeyboardScrollsTheList checks the keyboard path through a list
// taller than the panel: Tab reaches a row below the edge and brings it into
// view, Page Down, End and Home scroll the list while the keyboard has it.
func TestKeysPanelKeyboardScrollsTheList(t *testing.T) {
	v := &KeysPanelView{Entries: keysPanelScrollEntries(20), Through: 20, Loaded: true}
	tt := keysPanelTester(v, 900, 320)

	// The last key starts below the panel: its box is clipped to nothing.
	if rect, ok := tt.Find("key-19"); !ok || rect.W != 0 {
		t.Fatalf("the last key is already on screen: %+v, found %v", rect, ok)
	}

	// Tab: the filter, then the first row, and on to a row the first frame did
	// not show. Focusing it scrolls the list instead of leaving it out of view.
	tt.Key(0, ui.KeyTab)
	for i := 0; i < len(v.Entries) && !tt.Focused("key-12"); i++ {
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused("key-12") {
		t.Fatalf("Tab did not reach key-12, texts %q", tt.Texts())
	}
	if rect, ok := tt.Find("key-12"); !ok || rect.W == 0 || rect.Y+rect.H > 320 {
		t.Fatalf("the focused key-12 is not in view: %+v, found %v", rect, ok)
	}

	// End: the last row comes into view, the first leaves it.
	tt.Key(0, ui.KeyEnd)
	last, ok := tt.Find("key-19")
	if !ok || last.W == 0 || last.Y+last.H > 320 {
		t.Fatalf("End did not bring the last key into view: %+v, found %v", last, ok)
	}
	if first, ok := tt.Find("key-0"); !ok || first.W != 0 {
		t.Fatalf("End left the first key in view: %+v, found %v", first, ok)
	}

	// Home: back to the top.
	tt.Key(0, ui.KeyHome)
	if first, ok := tt.Find("key-0"); !ok || first.W == 0 || first.Y < 0 {
		t.Fatalf("Home did not return to the first key: %+v, found %v", first, ok)
	}

	// Page Down scrolls a page while the row holds the keyboard.
	before, _ := tt.Find("key-19")
	tt.Key(0, ui.KeyPageDown)
	after, _ := tt.Find("key-19")
	if after.Y >= before.Y {
		t.Fatalf("Page Down did not scroll the list: last key y %.1f -> %.1f", before.Y, after.Y)
	}
}

// shellKeysState is the shell fixture with a loaded session.keys snapshot for
// the selected session and the browser showing. The sidebar's Session keys entry
// is gone, as the reference has none, so the window's own ShowKeys is the state
// the panel reads.
func shellKeysState() *State {
	s := fixture()
	s.Keys = KeysPanelView{Entries: keysPanelEntries(), Through: 12, Loaded: true}
	s.ShowKeys = true
	return s
}

// TestShellDrawsAndClosesTheKeysPanel checks the wiring: a loaded snapshot with
// ShowKeys swaps the main column for the browser, a row click writes only the
// shell's view state, and clearing ShowKeys returns to the conversation. The
// sidebar entry is gone because the reference's column has none.
func TestShellDrawsAndClosesTheKeysPanel(t *testing.T) {
	s := shellKeysState()
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleRegion, "Keys panel"); !ok {
		t.Fatalf("the keys panel is not drawn, texts %q", tt.Texts())
	}
	for _, want := range []string{"3 keys", "through 12", "implement the keys panel"} {
		if !tt.HasText(want) {
			t.Errorf("the browser is missing %q, texts %q", want, tt.Texts())
		}
	}
	if tt.HasText("hello from the fixture") {
		t.Errorf("the conversation is still drawn under the browser, texts %q", tt.Texts())
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

	// Clearing the window's state returns to the conversation.
	s.ShowKeys = false
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

// TestShellHasNoKeysEntry checks the reference's column: no Session keys entry
// is drawn, with or without a snapshot, and nothing opens a browser the window
// did not already show.
func TestShellHasNoKeysEntry(t *testing.T) {
	for _, withSnapshot := range []bool{false, true} {
		s := fixture()
		if withSnapshot {
			s.Keys = KeysPanelView{Entries: keysPanelEntries(), Through: 12, Loaded: true}
		}
		tt := ui.NewTester(s.View, 900, 600)

		if _, ok := tt.AccessByRole(ui.RoleButton, "Session keys"); ok {
			t.Errorf("snapshot=%v: the sidebar still exposes a Session keys entry, texts %q", withSnapshot, tt.Texts())
		}
		if tt.HasText("Session keys") {
			t.Errorf("snapshot=%v: the sidebar still draws Session keys, texts %q", withSnapshot, tt.Texts())
		}
		if _, ok := tt.AccessByRole(ui.RoleRegion, "Keys panel"); ok {
			t.Errorf("snapshot=%v: a browser opened by itself, texts %q", withSnapshot, tt.Texts())
		}
	}
}

// TestShellKeysPanelFiltersTheSnapshot checks the filter through the shell: the
// panel edits State.Keys.Query in place, so the narrowed list survives the next
// frame and the caller's entries are untouched.
func TestShellKeysPanelFiltersTheSnapshot(t *testing.T) {
	s := shellKeysState()
	tt := ui.NewTester(s.View, 900, 600)
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
	if info, ok := tt.AccessByRole(ui.RoleButton, "Session keys"); ok {
		t.Errorf("the keys entry returned for a session with no snapshot: %+v", info)
	}

	// The window's read for the new session fills the browser again, and only
	// that session's keys are shown.
	s.Keys = KeysPanelView{
		Entries: []KeysPanelEntry{{Key: "other", Sequence: 2, InputID: "input-9", Text: "second session key"}},
		Through: 2,
		Loaded:  true,
	}
	s.ShowKeys = true
	tt.Frame()
	if !tt.HasText("second session key") {
		t.Errorf("the new session's key is missing, texts %q", tt.Texts())
	}
	if tt.HasText("implement the keys panel") {
		t.Errorf("the old session's key is drawn for s2, texts %q", tt.Texts())
	}
}
