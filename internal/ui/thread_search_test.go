package ui

import (
	"fmt"
	"image"
	"testing"

	"github.com/egoist/mygo/ui"
)

// threadSearchFixture is a conversation the find bar can search: a user row, an
// assistant turn whose thinking, text and tool blocks each carry the word, and a
// second user row. A query therefore finds matches in every kind of text the
// timeline draws and in more than one row.
func threadSearchFixture() *State {
	s := fixture()
	s.Messages = []Message{
		{ID: "u1", Author: AuthorUser, Content: "Find the needle in the first row."},
		{ID: "a1", Author: AuthorAssistant, Blocks: []Block{
			{ID: "b-thinking", Kind: BlockThinking, Content: "The needle hides here too.", StartedAt: 1_000, EndedAt: 4_000},
			{ID: "b-text", Kind: BlockText, Content: "A Markdown answer with a needle inside it."},
			{ID: "b-tool", Kind: BlockTool, Tool: Tool{Name: "bash", Params: "grep needle", Result: "one hit", Status: ToolComplete}},
		}},
		{ID: "u2", Author: AuthorUser, Content: "And the last NEEDLE closes the loop."},
	}
	return s
}

// threadSearchFieldLabel is the find bar's field name in the default language,
// as every test addresses it.
var threadSearchFieldLabel = threadSearchLabel(LocaleEN, keyThreadSearchLabel)

// openThreadSearch opens the find bar with the shell's shortcut and checks that
// its field took the keyboard focus, as the reference's does.
func openThreadSearch(t *testing.T, tt *ui.Tester, s *State) ui.AccessInfo {
	t.Helper()
	tt.Key(ui.Cmd, ui.KeyF)
	if !s.ShowThreadSearch {
		t.Fatalf("Cmd+F did not open the find bar, texts %q", tt.Texts())
	}
	info, ok := tt.AccessByRole(ui.RoleTextField, threadSearchFieldLabel)
	if !ok {
		t.Fatalf("the find bar has no field, texts %q", tt.Texts())
	}
	if !info.Focused {
		t.Fatalf("the find bar's field did not take the focus, texts %q", tt.Texts())
	}
	return info
}

// threadSearchControl returns one of the bar's entries by its accessible name.
func threadSearchControl(t *testing.T, tt *ui.Tester, label string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatalf("the find bar has no %q entry, texts %q", label, tt.Texts())
	}
	return info
}

// pixelsDiffer reports whether the two frames draw different colors anywhere
// inside rect, which is how a test sees a mark on a row without reaching into the
// spans it was drawn with.
func pixelsDiffer(a, b *image.RGBA, rect ui.Rect) bool {
	width, height := a.Bounds().Dx(), a.Bounds().Dy()
	x0, y0 := max(0, int(rect.X)), max(0, int(rect.Y))
	x1, y1 := min(width, int(rect.X+rect.W)+1), min(height, int(rect.Y+rect.H)+1)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return true
			}
		}
	}
	return false
}

// TestThreadSearchShortcutOpensTheField checks the shortcut the reference binds:
// Cmd+F draws the field in the toolbar, focused, with its placeholder; a second
// press puts the focus back in it and keeps the query the first one typed.
func TestThreadSearchShortcutOpensTheField(t *testing.T) {
	s := threadSearchFixture()
	tt := ui.NewTester(s.View, 900, 600)
	if _, ok := tt.AccessByRole(ui.RoleTextField, threadSearchFieldLabel); ok {
		t.Fatalf("the find bar is open before the shortcut, texts %q", tt.Texts())
	}

	info := openThreadSearch(t, tt, s)
	if info.Placeholder != tr(LocaleEN, keyThreadSearchPlaceholder) {
		// i18n.go does not carry the key yet, so the bar draws its English text.
		if info.Placeholder != threadSearchFallbacks[keyThreadSearchPlaceholder] {
			t.Errorf("the field shows %q, want the find placeholder", info.Placeholder)
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleGroup, threadSearchLabel(LocaleEN, keyThreadSearchRegion)); !ok {
		t.Errorf("the bar is not exposed as a group, texts %q", tt.Texts())
	}

	tt.Type("first row")
	if s.ThreadSearchQuery != "first row" {
		t.Fatalf("the field edited %q, want the typed text", s.ThreadSearchQuery)
	}
	// Tab leaves the field for the bar's entries; Cmd+F comes back to it.
	tt.Key(0, ui.KeyTab)
	if tt.Focused(threadSearchFieldLabel) {
		t.Fatalf("Tab left the field in the bar")
	}
	tt.Key(ui.Cmd, ui.KeyF)
	if !tt.Focused(threadSearchFieldLabel) {
		t.Fatalf("Cmd+F did not put the focus back in the field")
	}
	if s.ThreadSearchQuery != "first row" {
		t.Errorf("the second Cmd+F reset the query to %q", s.ThreadSearchQuery)
	}
}

// TestThreadSearchCountsMatches checks the counter: "0/0" while the field is
// empty or the query matches nothing, then the number of the current match over
// how many the loaded rows hold, matched case-insensitively across the user,
// thinking, text and tool texts of the conversation.
func TestThreadSearchCountsMatches(t *testing.T) {
	s := threadSearchFixture()
	tt := ui.NewTester(s.View, 900, 700)
	openThreadSearch(t, tt, s)

	if !tt.HasText("0/0") {
		t.Fatalf("an empty query does not read 0/0, texts %q", tt.Texts())
	}
	tt.Type("NEEDLE")
	if !tt.HasText("1/5") {
		t.Fatalf("a query with five matches does not read 1/5, texts %q", tt.Texts())
	}
	counter, ok := tt.AccessByRole(ui.RoleStatus, "1/5")
	if !ok {
		t.Fatalf("the counter is not drawn as a status, texts %q", tt.Texts())
	}
	if want := threadSearchCountLabel(LocaleEN, 1, 5); counter.Description != want {
		t.Errorf("the counter describes itself as %q, want %q", counter.Description, want)
	}

	tt.Type(" absent")
	field, ok := tt.AccessByRole(ui.RoleTextField, threadSearchFieldLabel)
	if !ok {
		t.Fatalf("the field left the bar, texts %q", tt.Texts())
	}
	if field.Value != "NEEDLE absent" {
		t.Errorf("an unmatched query left %q, want the typed text kept", field.Value)
	}
	if !tt.HasText("0/0") {
		t.Errorf("an unmatched query does not read 0/0, texts %q", tt.Texts())
	}
	for _, label := range []string{threadSearchLabel(LocaleEN, keyThreadSearchPrevious), threadSearchLabel(LocaleEN, keyThreadSearchNext)} {
		if entry := threadSearchControl(t, tt, label); !entry.Disabled {
			t.Errorf("%q stays available with no match", label)
		}
	}
}

// TestThreadSearchStepsAndWraps checks the two ways through the matches: Enter
// and the next entry move forward, Shift+Enter and the previous entry move back,
// and both wrap at the ends. A step taken with the pointer puts the focus back in
// the field so Enter carries on.
func TestThreadSearchStepsAndWraps(t *testing.T) {
	s := threadSearchFixture()
	tt := ui.NewTester(s.View, 900, 700)
	openThreadSearch(t, tt, s)
	tt.Type("needle")

	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("2/5") {
		t.Fatalf("Enter did not step to 2/5, texts %q", tt.Texts())
	}
	tt.Key(ui.Shift, ui.KeyEnter)
	if !tt.HasText("1/5") {
		t.Fatalf("Shift+Enter did not step back to 1/5, texts %q", tt.Texts())
	}
	tt.Key(ui.Shift, ui.KeyEnter)
	if !tt.HasText("5/5") {
		t.Fatalf("Shift+Enter at the first match did not wrap to 5/5, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("1/5") {
		t.Fatalf("Enter at the last match did not wrap to 1/5, texts %q", tt.Texts())
	}

	next := threadSearchControl(t, tt, threadSearchLabel(LocaleEN, keyThreadSearchNext))
	tt.ClickAt(next.Bounds.X+next.Bounds.W/2, next.Bounds.Y+next.Bounds.H/2)
	if !tt.HasText("2/5") {
		t.Fatalf("the next entry did not step to 2/5, texts %q", tt.Texts())
	}
	if !tt.Focused(threadSearchFieldLabel) {
		t.Errorf("a step entry kept the focus instead of giving it back to the field")
	}
	previous := threadSearchControl(t, tt, threadSearchLabel(LocaleEN, keyThreadSearchPrevious))
	tt.ClickAt(previous.Bounds.X+previous.Bounds.W/2, previous.Bounds.Y+previous.Bounds.H/2)
	if !tt.HasText("1/5") {
		t.Fatalf("the previous entry did not step back to 1/5, texts %q", tt.Texts())
	}
}

// TestThreadSearchHighlightsTheCurrentMatch checks the mark on the row the
// current match is in: the row is drawn differently while it is the current one,
// the mark moves with the query, and a match in a folded thinking block opens
// that fold so the run is on screen.
func TestThreadSearchHighlightsTheCurrentMatch(t *testing.T) {
	s := threadSearchFixture()
	tt := ui.NewTester(s.View, 900, 700)
	first, ok := tt.Find(s.Messages[0].Content)
	if !ok {
		t.Fatalf("the first row is not drawn, texts %q", tt.Texts())
	}
	if tt.HasText(s.Messages[1].Blocks[0].Content) {
		t.Fatalf("the thinking block is open before the search, texts %q", tt.Texts())
	}
	plain := tt.Image()

	// The query matches the first row: that row is marked.
	openThreadSearch(t, tt, s)
	tt.Type("first row")
	withFirst := tt.Image()
	if !pixelsDiffer(plain, withFirst, first) {
		t.Errorf("the current match does not mark its row")
	}

	// A query matching the last row moves the mark off the first one. Both
	// queries leave the thinking block folded, so the rows keep their layout and
	// the difference is the mark alone.
	tt.Key(0, ui.KeyEscape)
	openThreadSearch(t, tt, s)
	tt.Type("closes the loop")
	withLast := tt.Image()
	if !pixelsDiffer(withFirst, withLast, first) {
		t.Errorf("the mark stayed on the first row while the match moved to the last one")
	}

	// A match in a folded thinking block opens the fold, so the marked run is on
	// screen rather than hidden behind the disclosure.
	tt.Key(0, ui.KeyEscape)
	openThreadSearch(t, tt, s)
	tt.Type("hides here")
	if !tt.HasText(s.Messages[1].Blocks[0].Content) {
		t.Fatalf("the match in a folded block did not open it, texts %q", tt.Texts())
	}
	if !tt.HasText("1/1") {
		t.Errorf("the query in the thinking block does not read 1/1, texts %q", tt.Texts())
	}
}

// TestThreadSearchClosesClearsTheMark checks Escape: it closes the bar, forgets
// the query and the highlight, and draws the conversation the frame before the
// search did.
func TestThreadSearchClosesClearsTheMark(t *testing.T) {
	s := threadSearchFixture()
	tt := ui.NewTester(s.View, 900, 700)
	rows := make([]ui.Rect, 0, len(s.Messages))
	for _, m := range s.Messages {
		label := m.Content
		if label == "" {
			label = m.Blocks[1].Content
		}
		if rect, ok := tt.Find(label); ok {
			rows = append(rows, rect)
		}
	}
	before := tt.Image()

	openThreadSearch(t, tt, s)
	tt.Type("row")
	highlighted := tt.Image()
	marked := false
	for _, rect := range rows {
		if pixelsDiffer(before, highlighted, rect) {
			marked = true
		}
	}
	if !marked {
		t.Fatalf("the open bar marks no row, texts %q", tt.Texts())
	}

	tt.Key(0, ui.KeyEscape)
	if s.ShowThreadSearch || s.ThreadSearchQuery != "" {
		t.Fatalf("Escape left the bar open with %q", s.ThreadSearchQuery)
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, threadSearchFieldLabel); ok {
		t.Fatalf("the field is still drawn after Escape, texts %q", tt.Texts())
	}
	after := tt.Image()
	for i, rect := range rows {
		if pixelsDiffer(before, after, rect) {
			t.Errorf("row %d still differs from the frame before the search", i)
		}
	}
}

// TestThreadSearchScrollsToTheMatch checks that a match off screen is scrolled
// into view: the conversation follows its end, so a match in the first row is
// drawn only once the bar asks the list for it.
func TestThreadSearchScrollsToTheMatch(t *testing.T) {
	s := fixture()
	s.Messages = nil
	for i := range 60 {
		s.Messages = append(s.Messages, Message{ID: fmt.Sprintf("m%d", i), Author: AuthorUser,
			Content: fmt.Sprintf("row %d of the conversation", i)})
	}
	s.Messages[0].Content = "the needle sits at the first row"
	tt := ui.NewTester(s.View, 900, 600)
	if tt.HasText(s.Messages[0].Content) {
		t.Fatalf("the first row is on screen before the search, texts %q", tt.Texts())
	}

	openThreadSearch(t, tt, s)
	tt.Type("needle")
	if !tt.HasText(s.Messages[0].Content) {
		t.Fatalf("the match off screen was not scrolled into view, texts %q", tt.Texts())
	}
	if !tt.HasText("1/1") {
		t.Errorf("the single match does not read 1/1, texts %q", tt.Texts())
	}

	// Wrapping from the last match back to the first scrolls back as well.
	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("1/1") {
		t.Errorf("a step around a single match left it at %q", tt.Texts())
	}
	if !tt.HasText(s.Messages[0].Content) {
		t.Errorf("the wrapped match is not on screen, texts %q", tt.Texts())
	}
}

// TestThreadSearchClosesWithTheConversation checks the places the bar cannot stay
// open: another session's rows are not the ones it counted, the appearance
// surface takes the whole window, the keys browser takes the conversation, and
// the search palette covers it.
func TestThreadSearchClosesWithTheConversation(t *testing.T) {
	s := threadSearchFixture()
	s.Keys = KeysPanelView{Loaded: true}
	tt := ui.NewTester(s.View, 900, 700)
	openThreadSearch(t, tt, s)
	tt.Type("needle")

	// The search palette covers the conversation and closes the bar.
	tt.Key(searchShortcutMods, searchShortcutKey)
	if s.ShowThreadSearch {
		t.Fatalf("the search palette left the find bar open")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleTextField, threadSearchFieldLabel); ok {
		t.Fatalf("the find bar outlived the palette, texts %q", tt.Texts())
	}

	// The appearance surface takes the window; a closed bar stays closed when the
	// conversation returns.
	openThreadSearch(t, tt, s)
	tt.Type("needle")
	s.ShowSettings = true
	tt.Frame()
	if s.ShowThreadSearch {
		t.Fatalf("the settings route left the find bar open")
	}
	s.ShowSettings = false
	tt.Frame()
	if s.ShowThreadSearch {
		t.Fatalf("the conversation returned with the find bar open")
	}

	// The keys browser takes the conversation.
	openThreadSearch(t, tt, s)
	tt.Type("needle")
	s.ShowKeys = true
	tt.Frame()
	if s.ShowThreadSearch {
		t.Fatalf("the keys browser left the find bar open")
	}
	s.ShowKeys = false
	tt.Frame()

	// Another session's rows are another conversation.
	openThreadSearch(t, tt, s)
	tt.Type("needle")
	clickRow(t, tt, "Second session")
	tt.Frame()
	if s.ShowThreadSearch || s.ThreadSearchQuery != "" {
		t.Fatalf("a session change left the bar open with %q", s.ThreadSearchQuery)
	}
}

// TestThreadSearchNarrowWindowKeepsTheControlsInside checks the bar's share of a
// narrow toolbar: it gives its width up, its field shrinks, the counter and the
// three entries stay on the bar, and nothing runs past the window's edge.
func TestThreadSearchNarrowWindowKeepsTheControlsInside(t *testing.T) {
	const width = 560
	s := threadSearchFixture()
	tt := ui.NewTester(s.View, width, 600)
	openThreadSearch(t, tt, s)
	tt.Type("needle")

	header, ok := tt.AccessByRole(ui.RoleToolbar, tr(LocaleEN, keyHeaderRegion))
	if !ok {
		t.Fatalf("the conversation bar is not drawn, texts %q", tt.Texts())
	}
	bar, ok := tt.AccessByRole(ui.RoleGroup, threadSearchLabel(LocaleEN, keyThreadSearchRegion))
	if !ok {
		t.Fatalf("the find bar is not drawn, texts %q", tt.Texts())
	}
	if bar.Bounds.W > threadSearchW+0.5 {
		t.Errorf("the find bar is %.1f wide, want at most %d", bar.Bounds.W, threadSearchW)
	}
	if bar.Bounds.W >= threadSearchW {
		t.Errorf("the find bar kept its full %d DIPs in a %d DIP window", threadSearchW, width)
	}
	field, ok := tt.AccessByRole(ui.RoleTextField, threadSearchFieldLabel)
	if !ok {
		t.Fatalf("the field left the bar, texts %q", tt.Texts())
	}
	if field.Bounds.W < 100 {
		t.Errorf("the field is %.1f wide in a %d DIP window, too narrow to read", field.Bounds.W, width)
	}
	counter, ok := tt.AccessByRole(ui.RoleStatus, "1/5")
	if !ok {
		t.Fatalf("the counter left the bar, texts %q", tt.Texts())
	}
	parts := []struct {
		name string
		rect ui.Rect
	}{
		{"field", field.Bounds},
		{"counter", counter.Bounds},
		{"previous", threadSearchControl(t, tt, threadSearchLabel(LocaleEN, keyThreadSearchPrevious)).Bounds},
		{"next", threadSearchControl(t, tt, threadSearchLabel(LocaleEN, keyThreadSearchNext)).Bounds},
		{"close", threadSearchControl(t, tt, threadSearchLabel(LocaleEN, keyThreadSearchClose)).Bounds},
	}
	for _, part := range parts {
		if part.rect.X < bar.Bounds.X-0.5 || part.rect.X+part.rect.W > bar.Bounds.X+bar.Bounds.W+0.5 {
			t.Errorf("the %s at %.1f..%.1f is not inside the find bar %.1f..%.1f",
				part.name, part.rect.X, part.rect.X+part.rect.W, bar.Bounds.X, bar.Bounds.X+bar.Bounds.W)
		}
		if !containsRect(header.Bounds, part.rect) {
			t.Errorf("the %s at %+v is not inside the toolbar %+v", part.name, part.rect, header.Bounds)
		}
		if part.rect.X+part.rect.W > width+0.5 {
			t.Errorf("the %s ends at %.1f, past the %d DIP window", part.name, part.rect.X+part.rect.W, width)
		}
	}
	// The parts keep their order and do not overlap.
	for i := 1; i < len(parts); i++ {
		previous := parts[i-1]
		if parts[i].rect.X+0.5 < previous.rect.X+previous.rect.W {
			t.Errorf("the %s at %.1f overlaps the %s ending at %.1f",
				parts[i].name, parts[i].rect.X, previous.name, previous.rect.X+previous.rect.W)
		}
	}
}

// TestThreadSearchTabOrderAndFocusReturn checks the bar's place in the keyboard
// order and where Escape gives the focus back: Tab moves from the field to the two
// step entries and the close entry, and Escape closes the bar and returns the
// focus to the control it took it from — the composer at first, then the sidebar's
// thread list once that has it.
func TestThreadSearchTabOrderAndFocusReturn(t *testing.T) {
	s := threadSearchFixture()
	tt := ui.NewTester(s.View, 900, 700)
	tt.Key(ComposerShortcutMods, ComposerShortcutKey)
	if !tt.Focused(tr(LocaleEN, keyComposerLabel)) {
		t.Fatalf("the composer did not take the focus, texts %q", tt.Texts())
	}
	openThreadSearch(t, tt, s)
	tt.Type("needle")

	for _, want := range []string{
		threadSearchLabel(LocaleEN, keyThreadSearchPrevious),
		threadSearchLabel(LocaleEN, keyThreadSearchNext),
		threadSearchLabel(LocaleEN, keyThreadSearchClose),
	} {
		tt.Key(0, ui.KeyTab)
		if !tt.Focused(want) {
			t.Fatalf("Tab does not reach %q, texts %q", want, tt.Texts())
		}
	}

	tt.Key(0, ui.KeyEscape)
	if s.ShowThreadSearch {
		t.Fatalf("Escape with the focus on an entry left the bar open")
	}
	if s.ThreadSearchQuery != "" {
		t.Errorf("Escape left the query %q", s.ThreadSearchQuery)
	}
	if !tt.Focused(tr(LocaleEN, keyComposerLabel)) {
		t.Errorf("Escape did not give the focus back to the composer, texts %q", tt.Texts())
	}

	// Opened from the sidebar's thread list, the bar gives the focus back there
	// rather than to the composer: it returns the focus to the control it took it
	// from. The tester's window takes the keyboard first, because mygo answers
	// which control has it from the window that is focused.
	tt.SetFocused(true)
	clickRow(t, tt, "First session")
	openThreadSearch(t, tt, s)
	tt.Key(0, ui.KeyEscape)
	if !tt.Focused(tr(LocaleEN, keySidebarSessions)) {
		t.Errorf("Escape did not give the focus back to the thread list, texts %q", tt.Texts())
	}
}

// TestThreadSearchMatchesTheRowsItDraws checks the search's scope and order: the
// user's text, an assistant turn's text, thinking and tool calls but not its file
// changes, with the matches in reading order and no query matching nothing.
func TestThreadSearchMatchesTheRowsItDraws(t *testing.T) {
	s := threadSearchFixture()
	s.Messages[1].Blocks = append(s.Messages[1].Blocks, Block{ID: "b-files", Kind: BlockFileChanges,
		Files: []FileChange{{Path: "needle.txt", Status: FileModified}}})

	matches := threadSearchMatches(s.Messages, "needle")
	want := []threadMatch{
		{Row: 0, Block: -1, Start: 9, End: 15},
		{Row: 1, Block: 0},
		{Row: 1, Block: 1},
		{Row: 1, Block: 2},
		{Row: 2, Block: -1},
	}
	if len(matches) != len(want) {
		t.Fatalf("the query found %d matches, want %d: %+v", len(matches), len(want), matches)
	}
	for i, match := range matches {
		if match.Row != want[i].Row || match.Block != want[i].Block {
			t.Errorf("match %d is in row %d block %d, want row %d block %d",
				i, match.Row, match.Block, want[i].Row, want[i].Block)
		}
	}
	if match := matches[0]; match.Start != 9 || match.End != 15 {
		t.Errorf("the first match is %d..%d, want the run 9..15", match.Start, match.End)
	}
	if got := s.Messages[0].Content[matches[0].Start:matches[0].End]; got != "needle" {
		t.Errorf("the match slices %q, want the drawn run", got)
	}

	if got := threadSearchMatches(s.Messages, "   "); got != nil {
		t.Errorf("a blank query found %+v, want nothing", got)
	}
	if got := threadSearchMatches(s.Messages, "  NEEDLE "); len(got) != 5 {
		t.Errorf("a padded query found %d matches, want 5", len(got))
	}
}

// TestThreadMatchRunsKeepTheDrawnText checks the matcher behind the counter: it
// folds case, keeps the byte offsets of the text it was given, finds
// non-overlapping runs, and leaves a text with no match alone.
func TestThreadMatchRunsKeepTheDrawnText(t *testing.T) {
	text := "Ärger über ÄRGER und nichts sonst"
	runs := threadMatchRuns(text, foldedRunes("ärger"))
	if len(runs) != 2 {
		t.Fatalf("the query found %d runs, want 2", len(runs))
	}
	for _, run := range runs {
		if got := text[run[0]:run[1]]; got != "Ärger" && got != "ÄRGER" {
			t.Errorf("a run slices %q, want the drawn word", got)
		}
	}
	if got := threadMatchRuns("hello", foldedRunes("needle")); len(got) != 0 {
		t.Errorf("a text without the query found %+v", got)
	}
	// Two matches that touch count once each, not as one overlapping run.
	if got := threadMatchRuns("aaaa", foldedRunes("aa")); len(got) != 2 {
		t.Errorf("the overlapping query found %d runs, want 2", len(got))
	}
}

// TestTextRunsCutTheHighlightedRun checks the pieces a marked text is drawn from:
// the run the query matched and the text around it, with a range outside the text
// leaving it whole.
func TestTextRunsCutTheHighlightedRun(t *testing.T) {
	runs := textRuns("hello needle here", TextHighlight{Start: 6, End: 12})
	if len(runs) != 3 {
		t.Fatalf("the mark cut the text into %d runs, want 3: %+v", len(runs), runs)
	}
	want := []textRun{{Text: "hello "}, {Text: "needle", Highlight: true}, {Text: " here"}}
	for i, run := range runs {
		if run != want[i] {
			t.Errorf("run %d is %+v, want %+v", i, run, want[i])
		}
	}
	if got := textRuns("short", TextHighlight{Start: 5, End: 9}); len(got) != 1 || got[0].Highlight {
		t.Errorf("a range outside the text did not leave it whole: %+v", got)
	}
	if got := textRuns("", TextHighlight{Start: 0, End: 0}); len(got) != 1 {
		t.Errorf("an empty text was cut into %+v", got)
	}
}

// foldedRunes is the folded query the matcher compares against.
func foldedRunes(s string) []rune {
	folded, _ := foldRunes(s)
	return folded
}
