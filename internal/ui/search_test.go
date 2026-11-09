package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// searchPaletteState is the fixture with a session callback that records the
// choice, as the window's controller does, so a test can tell a highlighted row
// from an opened one.
func searchPaletteState() (*State, *string) {
	s := searchState()
	chosen := ""
	s.OnSelect = func(sessionID string) {
		chosen = sessionID
		s.Selected = sessionID
	}
	return s, &chosen
}

// paletteDialog returns the open palette's dialog and whether it is showing.
func paletteDialog(tt *ui.Tester) (ui.AccessInfo, bool) {
	return tt.AccessByRole(ui.RoleDialog, "")
}

// openPalette opens the palette with the window's shortcut, checks that its
// input took the focus, and lets the reference's 200 ms enter finish so the
// frames and boxes the tests read are the settled ones.
func openPalette(t *testing.T, tt *ui.Tester) {
	t.Helper()
	tt.Key(searchShortcutMods, searchShortcutKey)
	if _, ok := paletteDialog(tt); !ok {
		t.Fatalf("the shortcut did not open the palette, texts %q", tt.Texts())
	}
	if !tt.Focused(tr(LocaleEN, keySearchPlaceholder)) {
		t.Fatalf("the palette's input did not take the focus, texts %q", tt.Texts())
	}
	tt.Advance(searchMotion)
}

// paletteResult returns one result row of the open palette. The sidebar's rows
// carry the same titles, so the row is addressed by its button role rather than
// by the first text on the frame.
func paletteResult(t *testing.T, tt *ui.Tester, title string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleButton, title)
	if !ok {
		t.Fatalf("the palette has no %q row, texts %q", title, tt.Texts())
	}
	return info
}

// clickPaletteResult clicks one result row by its button box.
func clickPaletteResult(t *testing.T, tt *ui.Tester, title string) {
	t.Helper()
	info := paletteResult(t, tt, title)
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
}

// TestSearchPaletteOpensCentered checks the reference's dialog: Cmd+G opens a
// dialog of 448 DIPs centred in the window, the frame stays drawn under it, and
// the input has the focus with the placeholder the reference shows.
func TestSearchPaletteOpensCentered(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 1000)
	if _, ok := paletteDialog(tt); ok {
		t.Fatal("the palette is open before the shortcut")
	}

	openPalette(t, tt)
	dialog, _ := paletteDialog(tt)
	if dialog.Bounds.W != searchPanelW {
		t.Errorf("the palette is %.1f wide, want the reference's %d", dialog.Bounds.W, searchPanelW)
	}
	if want := (900 - float32(searchPanelW)) / 2; dialog.Bounds.X != want {
		t.Errorf("the palette starts at x=%.1f, want the window's centre %v", dialog.Bounds.X, want)
	}
	if dialog.Bounds.Y <= 0 || dialog.Bounds.Y+dialog.Bounds.H > 1000 {
		t.Errorf("the palette at y=%.1f..%.1f is not inside the window", dialog.Bounds.Y, dialog.Bounds.Y+dialog.Bounds.H)
	}
	// The reference measures 174 DIPs for a two-row dialog: the 56 input row,
	// the 22 heading, two 42 rows, the room below them and the panel's hairline.
	if dialog.Bounds.H != 174 {
		t.Errorf("the two-row palette is %.1f high, want the reference's 174", dialog.Bounds.H)
	}
	if !tt.HasText("hello from the fixture") || !tt.HasText("Sign in to Dim") {
		t.Errorf("the frame under the palette was dropped, texts %q", tt.Texts())
	}
	if info, ok := tt.AccessByRole(ui.RoleTextField, tr(LocaleEN, keySearchPlaceholder)); !ok {
		t.Errorf("the palette has no named search field, texts %q", tt.Texts())
	} else if info.Placeholder != tr(LocaleEN, keySearchPlaceholder) {
		t.Errorf("the search field shows %q, want %q", info.Placeholder, tr(LocaleEN, keySearchPlaceholder))
	}
}

// TestSearchPaletteDimsTheWindow checks the reference's scrim: the whole window
// is drawn under a black 80% layer, so a corner pixel keeps about a fifth of the
// face it had and the palette itself stays the only light surface.
func TestSearchPaletteDimsTheWindow(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 1000)
	before := tt.Image().RGBAAt(2, 2)

	openPalette(t, tt)
	after := tt.Image().RGBAAt(2, 2)
	for _, c := range []struct {
		name     string
		was, now uint8
	}{{"red", before.R, after.R}, {"green", before.G, after.G}, {"blue", before.B, after.B}} {
		want := int(c.was) / 5
		if got := int(c.now); got < want-4 || got > want+4 {
			t.Errorf("the scrim left %s at %d of %d, want about %d", c.name, got, c.was, want)
		}
	}
}

// TestSearchPaletteFilters checks the display copy the results come from: the
// query narrows the rows by title and by project, case-insensitively, and the
// catalog itself is not written.
func TestSearchPaletteFilters(t *testing.T) {
	s := searchState()
	before := append([]Session(nil), s.Sessions...)
	tt := ui.NewTester(s.View, 1200, 1000)
	openPalette(t, tt)

	if !tt.HasText(tr(LocaleEN, keySearchRecent)) {
		t.Errorf("the open palette has no recent-chats heading, texts %q", tt.Texts())
	}
	for _, title := range []string{"Fix flaky tests", "Plan release", "Read notes", "Untitled"} {
		paletteResult(t, tt, title)
	}

	tt.Type("PLAN")
	if _, ok := tt.AccessByRole(ui.RoleButton, "Plan release"); !ok {
		t.Fatalf("the query dropped the matching chat, texts %q", tt.Texts())
	}
	for _, gone := range []string{"Fix flaky tests", "Read notes", "Untitled"} {
		if _, ok := tt.AccessByRole(ui.RoleButton, gone); ok {
			t.Errorf("the query kept %q, texts %q", gone, tt.Texts())
		}
	}

	tt.Type("x")
	if _, ok := tt.AccessByRole(ui.RoleButton, "Plan release"); ok {
		t.Errorf("a query no chat matches still shows a row, texts %q", tt.Texts())
	}

	for i := range before {
		if before[i] != s.Sessions[i] {
			t.Errorf("the query rewrote the catalog: %+v, want %+v", s.Sessions, before)
		}
	}
}

// TestSearchPaletteMatchesTheProject checks the subtitle the reference shows:
// a project label is searched as well as a title, so the heading names the chat
// the query found.
func TestSearchPaletteMatchesTheProject(t *testing.T) {
	s := searchState()
	tt := ui.NewTester(s.View, 1200, 1000)
	openPalette(t, tt)

	tt.Type("docs")
	if _, ok := tt.AccessByRole(ui.RoleButton, "Read notes"); !ok {
		t.Fatalf("the project did not match, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Plan release"); ok {
		t.Errorf("the project query kept another project's chat, texts %q", tt.Texts())
	}
}

// TestSearchPaletteMatchesTheSessionID checks the third fact the input
// searches: a pasted session id finds its chat even when neither its title nor
// its project carries the text.
func TestSearchPaletteMatchesTheSessionID(t *testing.T) {
	s := searchState()
	tt := ui.NewTester(s.View, 1200, 1000)
	openPalette(t, tt)

	tt.Type("S3")
	if _, ok := tt.AccessByRole(ui.RoleButton, "Read notes"); !ok {
		t.Fatalf("the session id did not match, texts %q", tt.Texts())
	}
	for _, gone := range []string{"Fix flaky tests", "Plan release", "Untitled"} {
		if _, ok := tt.AccessByRole(ui.RoleButton, gone); ok {
			t.Errorf("the id query kept %q, texts %q", gone, tt.Texts())
		}
	}
}

// TestSearchPaletteHoverMovesTheSelection checks the pointer as a way to pick a
// row before Enter: pointing at a chat moves the highlight there, and it stays
// when the pointer leaves, while nothing is opened until Enter or a click.
func TestSearchPaletteHoverMovesTheSelection(t *testing.T) {
	s, chosen := searchPaletteState()
	tt := ui.NewTester(s.View, 1200, 1000)
	openPalette(t, tt)

	if s.SearchSelected != 0 {
		t.Fatalf("the palette highlighted %d, want the first row", s.SearchSelected)
	}
	row := paletteResult(t, tt, "Plan release")
	tt.Move(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	tt.Frame()
	if s.SearchSelected != 1 {
		t.Fatalf("the pointer highlighted %d, want the second row", s.SearchSelected)
	}
	if s.Selected != "s1" || *chosen != "" {
		t.Errorf("pointing at a row chose a chat: selected=%q chosen=%q", s.Selected, *chosen)
	}

	tt.Move(2, 2)
	tt.Frame()
	if s.SearchSelected != 1 {
		t.Errorf("leaving the row moved the highlight to %d, want it kept", s.SearchSelected)
	}
	// A pointer resting over a row does not pin the highlight: the keyboard still
	// moves it between rows.
	tt.Move(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyUp)
	if s.SearchSelected != 1 {
		t.Errorf("the arrow keys left the highlight at %d, want the second row", s.SearchSelected)
	}
	tt.Key(0, ui.KeyEnter)
	if *chosen != "s2" || s.Selected != "s2" {
		t.Errorf("Enter opened %q and selected %q, want the pointed row s2", *chosen, s.Selected)
	}
}

// TestSearchPaletteResultStatuses checks the 32×32 tile's three states: a row
// with no reported work draws the chat mark, a loading row the spinner and a
// failed row the failed mark on the danger wash. The palette derives no state:
// every row searchResults makes from the catalog is normal, because the shell
// holds no per-session run, unread or pin fact.
func TestSearchPaletteResultStatuses(t *testing.T) {
	for _, result := range searchResults(searchSessions(), "") {
		if result.Status != SearchResultNormal {
			t.Errorf("searchResults gave %q status %d, want the normal row", result.SessionID, result.Status)
		}
	}

	open, query, selected := true, "", 0
	results := []SearchResult{
		{SessionID: "n", Title: "Normal chat", Status: SearchResultNormal},
		{SessionID: "l", Title: "Loading chat", Status: SearchResultLoading},
		{SessionID: "e", Title: "Failed chat", Status: SearchResultError},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		searchPalette(c, newTokens(c.Theme()), SearchPaletteView{
			Open: &open, Query: &query, Results: results, Selected: &selected, Locale: LocaleEN,
		}, Actions{})
	}, 900, 700)
	tt.Advance(searchMotion)

	patterns := make(map[string]string, len(results))
	for _, result := range results {
		info := paletteResult(t, tt, result.Title)
		patterns[result.Title] = searchTilePattern(tt, info.Bounds)
	}
	for i, a := range results {
		for _, b := range results[i+1:] {
			if patterns[a.Title] == patterns[b.Title] {
				t.Errorf("the %q and %q tiles draw the same pixels", a.Title, b.Title)
			}
		}
	}
}

// searchTilePattern copies the 32×32 tile at the start of a result row, as the
// bytes it paints, so a test can tell the three status marks apart.
func searchTilePattern(tt *ui.Tester, row ui.Rect) string {
	img := tt.Image()
	out := make([]byte, 0, searchResultIcon*searchResultIcon*4)
	for y := 0; y < searchResultIcon; y++ {
		for x := 0; x < searchResultIcon; x++ {
			p := img.RGBAAt(int(row.X+searchRowPadX)+x, int(row.Y)+(searchRowH-searchResultIcon)/2+y)
			out = append(out, p.R, p.G, p.B, p.A)
		}
	}
	return string(out)
}

// TestSearchPaletteFadesAndZooms checks the reference's duration-200 entry and
// exit: the panel starts at 95% and transparent, passes through neither end at
// 100 ms, and settles at its measured size; closing keeps the copy on screen,
// fading and shrinking it over the same step. The scrim fades with it.
func TestSearchPaletteFadesAndZooms(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 1000)
	clear := tt.Image().RGBAAt(2, 2)

	tt.Key(searchShortcutMods, searchShortcutKey)
	start, ok := paletteDialog(tt)
	if !ok {
		t.Fatalf("the shortcut did not open the palette, texts %q", tt.Texts())
	}
	if want := float32(searchPanelW) * 0.95; start.Bounds.W < want-0.5 || start.Bounds.W > want+0.5 {
		t.Errorf("the palette starts %.1f wide, want the reference's 95%% of %d", start.Bounds.W, searchPanelW)
	}
	if got := tt.Image().RGBAAt(2, 2); got != clear {
		t.Errorf("the scrim is already %v on the first frame, want it to fade from the window's %v", got, clear)
	}

	tt.Advance(searchMotion / 2)
	mid, _ := paletteDialog(tt)
	if !(start.Bounds.W < mid.Bounds.W && mid.Bounds.W < searchPanelW) {
		t.Errorf("the palette is %.1f wide at 100 ms, want between %.1f and %d",
			mid.Bounds.W, start.Bounds.W, searchPanelW)
	}
	midScrim := tt.Image().RGBAAt(2, 2)
	if midScrim == clear {
		t.Errorf("the scrim at 100 ms is still the window's %v, want it on the way", midScrim)
	}

	tt.Advance(searchMotion / 2)
	settled, _ := paletteDialog(tt)
	if settled.Bounds.W != searchPanelW {
		t.Errorf("the settled palette is %.1f wide, want %d", settled.Bounds.W, searchPanelW)
	}
	dim := tt.Image().RGBAAt(2, 2)
	if midScrim == dim {
		t.Errorf("the scrim at 100 ms is already the settled %v", dim)
	}
	panel := [2]int{int(settled.Bounds.X) + 4, int(settled.Bounds.Y) + 4}
	shown := tt.Image().RGBAAt(panel[0], panel[1])

	tt.Key(0, ui.KeyEscape)
	if _, ok := paletteDialog(tt); ok {
		t.Fatal("Escape left the palette open")
	}
	frames := []uint8{shown.R}
	at := func() uint8 { return tt.Image().RGBAAt(panel[0], panel[1]).R }
	for i := 0; i < 2; i++ {
		tt.Advance(searchMotion / 2)
		frames = append(frames, at())
	}
	if frames[1] == frames[0] || frames[1] == frames[2] || frames[0] == frames[2] {
		t.Errorf("the closing panel at 0/100/200 ms is %v, want it on the way from the panel to the window", frames)
	}
	if got := tt.Image().RGBAAt(2, 2); got != clear {
		t.Errorf("the scrim ended at %v, want the window's %v back", got, clear)
	}
}

// TestSearchPaletteReducedMotionSkipsTheFade checks the reduce-motion path: with
// the desktop asking for less motion the panel is settled and the scrim dimmed
// on the frame the shortcut opens it, and both are gone on the frame Escape
// closes it.
func TestSearchPaletteReducedMotionSkipsTheFade(t *testing.T) {
	s := fixture()
	tt := ui.NewTesterWith(s.View, 900, 1000, ui.TesterOptions{
		Preferences: ui.Preferences{ReduceMotion: true},
	})
	clear := tt.Image().RGBAAt(2, 2)
	tt.Key(searchShortcutMods, searchShortcutKey)
	dialog, ok := paletteDialog(tt)
	if !ok {
		t.Fatalf("the shortcut did not open the palette, texts %q", tt.Texts())
	}
	if dialog.Bounds.W != searchPanelW {
		t.Errorf("the reduced-motion palette is %.1f wide on its first frame, want %d", dialog.Bounds.W, searchPanelW)
	}
	if got := tt.Image().RGBAAt(2, 2); got == clear {
		t.Errorf("the scrim is still the window's %v on the frame the palette opens", got)
	}

	tt.Key(0, ui.KeyEscape)
	if _, ok := paletteDialog(tt); ok {
		t.Fatal("Escape left the palette open")
	}
	if got := tt.Image().RGBAAt(2, 2); got != clear {
		t.Errorf("the closing copy left the scrim at %v, want the window's %v at once", got, clear)
	}
}

// TestSearchPaletteNarrowWindow checks the reference's width cap: in a window
// too narrow for 448 DIPs the panel takes 90% of it, stays centered and keeps
// its 42 DIP rows.
func TestSearchPaletteNarrowWindow(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 360, 800)
	openPalette(t, tt)

	dialog, _ := paletteDialog(tt)
	if want := float32(360) * 0.9; dialog.Bounds.W > want+0.5 || dialog.Bounds.W < want-0.5 {
		t.Errorf("the narrow palette is %.1f wide, want the reference's 90%% of 360 (%.1f)", dialog.Bounds.W, want)
	}
	if want := (360 - dialog.Bounds.W) / 2; dialog.Bounds.X != want {
		t.Errorf("the narrow palette starts at x=%.1f, want the window's centre %v", dialog.Bounds.X, want)
	}
	row := paletteResult(t, tt, "First session")
	if row.Bounds.H != searchRowH {
		t.Errorf("the narrow row is %.1f high, want %d", row.Bounds.H, searchRowH)
	}
}

// TestSearchPaletteEmptyState checks the reference's empty state: a query that
// leaves no chat shows the status line instead of a heading or a row, and the
// input keeps the text.
func TestSearchPaletteEmptyState(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 1200, 1000)
	openPalette(t, tt)

	tt.Type("zzzz-no-such-chat")
	if _, ok := tt.AccessByRole(ui.RoleStatus, tr(LocaleEN, keySearchEmpty)); !ok {
		t.Fatalf("the empty state is missing, texts %q", tt.Texts())
	}
	if tt.HasText(tr(LocaleEN, keySearchRecent)) {
		t.Errorf("the empty palette still draws its heading, texts %q", tt.Texts())
	}
	if info, ok := tt.AccessByRole(ui.RoleTextField, tr(LocaleEN, keySearchPlaceholder)); !ok || info.Value != "zzzz-no-such-chat" {
		t.Errorf("the empty state lost the query: %+v", info)
	}
}

// TestSearchPaletteKeyboard checks the reference's keys: Down and Up move the
// highlight without choosing, Enter opens the highlighted chat, and Cmd+1 opens
// the first result directly.
func TestSearchPaletteKeyboard(t *testing.T) {
	s, chosen := searchPaletteState()
	tt := ui.NewTester(s.View, 1200, 1000)

	openPalette(t, tt)
	if s.SearchSelected != 0 {
		t.Fatalf("the palette highlighted %d, want the first row", s.SearchSelected)
	}
	if s.Selected != "s1" || *chosen != "" {
		t.Fatalf("opening the palette chose a chat: selected=%q chosen=%q", s.Selected, *chosen)
	}

	tt.Key(0, ui.KeyDown)
	if s.SearchSelected != 1 {
		t.Fatalf("Down highlighted %d, want the second row", s.SearchSelected)
	}
	if s.Selected != "s1" || *chosen != "" {
		t.Errorf("moving the highlight chose a chat: selected=%q chosen=%q", s.Selected, *chosen)
	}
	tt.Key(0, ui.KeyUp)
	if s.SearchSelected != 0 {
		t.Errorf("Up highlighted %d, want the first row", s.SearchSelected)
	}

	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if *chosen != "s2" || s.Selected != "s2" {
		t.Errorf("Enter opened %q and selected %q, want s2", *chosen, s.Selected)
	}
	if _, ok := paletteDialog(tt); ok {
		t.Error("Enter left the palette open")
	}

	openPalette(t, tt)
	tt.Key(0, ui.KeyDown)
	tt.Key(ui.Cmd, ui.Key1)
	if *chosen != "s1" || s.Selected != "s1" {
		t.Errorf("Cmd+1 opened %q and selected %q, want s1", *chosen, s.Selected)
	}
	if _, ok := paletteDialog(tt); ok {
		t.Error("Cmd+1 left the palette open")
	}
}

// TestSearchPaletteStartsEachOpenFresh checks the reference's reopened dialog:
// closing clears the query and the highlight, so the next open lists the whole
// recent set with its first row selected.
func TestSearchPaletteStartsEachOpenFresh(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 1200, 1000)

	openPalette(t, tt)
	tt.Type("second")
	tt.Key(0, ui.KeyEscape)
	if s.ShowSearch || s.SearchQuery != "" || s.SearchSelected != 0 {
		t.Fatalf("closing left search=%v query=%q selected=%d", s.ShowSearch, s.SearchQuery, s.SearchSelected)
	}

	openPalette(t, tt)
	if s.SearchQuery != "" || s.SearchSelected != 0 {
		t.Errorf("reopening kept query=%q selected=%d", s.SearchQuery, s.SearchSelected)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "First session"); !ok {
		t.Errorf("reopening did not list the recent chats, texts %q", tt.Texts())
	}
}

// TestSearchPaletteEscapeReturnsTheFocus checks the reference's Escape: the
// palette closes, the focus goes back to the sidebar entry that opened it, and
// nothing is chosen.
func TestSearchPaletteEscapeReturnsTheFocus(t *testing.T) {
	s, chosen := searchPaletteState()
	tt := ui.NewTester(s.View, 1200, 1000)

	if err := tt.Click(tr(LocaleEN, keySidebarSearch)); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused(tr(LocaleEN, keySearchPlaceholder)) {
		t.Fatalf("the sidebar entry did not focus the palette's input, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if _, ok := paletteDialog(tt); ok {
		t.Fatal("Escape left the palette open")
	}
	if !tt.Focused(tr(LocaleEN, keySidebarSearch)) {
		t.Errorf("Escape did not return the focus to the search entry, texts %q", tt.Texts())
	}
	if *chosen != "" || s.Selected != "s1" {
		t.Errorf("Escape chose a chat: chosen=%q selected=%q", *chosen, s.Selected)
	}
}

// TestSearchPaletteShortcutReturnsTheFocus checks the other way in: opened from
// a shortcut while another control had the focus, the palette gives that
// control the focus back as it closes, not the sidebar entry.
func TestSearchPaletteShortcutReturnsTheFocus(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 1200, 1000)

	if err := tt.Click(tr(LocaleEN, keyComposerLabel)); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused(tr(LocaleEN, keyComposerLabel)) {
		t.Fatalf("the composer did not take the focus, texts %q", tt.Texts())
	}
	openPalette(t, tt)
	tt.Key(0, ui.KeyEscape)
	if _, ok := paletteDialog(tt); ok {
		t.Fatal("Escape left the palette open")
	}
	if !tt.Focused(tr(LocaleEN, keyComposerLabel)) {
		t.Errorf("Escape did not return the focus to the composer, texts %q", tt.Texts())
	}
}

// TestSearchPaletteBackdropCloses checks the scrim as a way out: a click on the
// window outside the panel closes the palette and returns the focus, as a click
// on the panel itself does not.
func TestSearchPaletteBackdropCloses(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 1200, 1000)

	openPalette(t, tt)
	dialog, _ := paletteDialog(tt)
	tt.ClickAt(dialog.Bounds.X+dialog.Bounds.W/2, dialog.Bounds.Y+searchInputH/2)
	if _, ok := paletteDialog(tt); !ok {
		t.Fatal("a click inside the panel closed the palette")
	}
	tt.ClickAt(10, 500)
	if _, ok := paletteDialog(tt); ok {
		t.Fatal("a click on the scrim left the palette open")
	}
	if s.ShowSearch {
		t.Error("the scrim did not clear the window's open record")
	}
}

// TestSearchPaletteClickOpens checks the reference's result click: it chooses
// that chat, closes the palette and asks the window for the session, while the
// sidebar's own selection changes only through that callback.
func TestSearchPaletteClickOpens(t *testing.T) {
	s, chosen := searchPaletteState()
	tt := ui.NewTester(s.View, 1200, 1000)
	openPalette(t, tt)

	clickPaletteResult(t, tt, "Plan release")
	if *chosen != "s2" || s.Selected != "s2" {
		t.Errorf("the click opened %q and selected %q, want s2", *chosen, s.Selected)
	}
	if _, ok := paletteDialog(tt); ok {
		t.Error("the click left the palette open")
	}
}

// TestSearchPaletteAbsentOnTheSettingsRoute checks the two surfaces never
// overlap: the appearance route takes the whole window, so the search shortcut
// opens nothing there and no palette is drawn over it.
func TestSearchPaletteAbsentOnTheSettingsRoute(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 1200, 1000)

	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Appearance") {
		t.Fatalf("the settings route is not showing, texts %q", tt.Texts())
	}
	tt.Key(searchShortcutMods, searchShortcutKey)
	if _, ok := paletteDialog(tt); ok {
		t.Error("the shortcut opened the palette over the settings route")
	}
	if s.ShowSearch {
		t.Error("the shortcut set the window's open record on the settings route")
	}
}

// TestSearchPaletteLimitsToEight checks the reference's Cmd+1..Cmd+8: the
// palette lists at most eight recent chats and numbers those rows from one.
func TestSearchPaletteLimitsToEight(t *testing.T) {
	sessions := make([]Session, 0, 12)
	for i := 0; i < 12; i++ {
		sessions = append(sessions, Session{ID: "s" + strconv.Itoa(i), Title: "Session " + strconv.Itoa(i)})
	}
	results := searchResults(sessions, "")
	if len(results) != searchMaxResults {
		t.Fatalf("the palette lists %d recent chats, want %d", len(results), searchMaxResults)
	}
	if results[0].SessionID != "s0" || results[searchMaxResults-1].SessionID != "s7" {
		t.Errorf("the palette kept %q..%q, want the catalog's first eight",
			results[0].SessionID, results[searchMaxResults-1].SessionID)
	}
	if hint := searchShortcutHint(searchMaxResults - 1); !strings.HasSuffix(hint, "8") {
		t.Errorf("the last hint is %q, want the eighth row's", hint)
	}
}

// TestSearchPaletteChineseLocale checks that the palette is translated with the
// rest of the shell: the placeholder, the heading and the empty state follow the
// saved language, while a session title stays the catalog's.
func TestSearchPaletteChineseLocale(t *testing.T) {
	s := fixture()
	s.Settings = SettingsView{Locale: LocaleZHCN}
	tt := ui.NewTester(s.View, 1200, 1000)
	tt.Key(searchShortcutMods, searchShortcutKey)
	if _, ok := paletteDialog(tt); !ok {
		t.Fatalf("the shortcut did not open the palette, texts %q", tt.Texts())
	}
	if !tt.Focused(tr(LocaleZHCN, keySearchPlaceholder)) {
		t.Fatalf("the zh-CN input did not take the focus, texts %q", tt.Texts())
	}
	tt.Advance(searchMotion)

	if info, ok := tt.AccessByRole(ui.RoleTextField, tr(LocaleZHCN, keySearchPlaceholder)); !ok ||
		info.Placeholder != tr(LocaleZHCN, keySearchPlaceholder) {
		t.Errorf("the zh-CN search field is %+v, texts %q", info, tt.Texts())
	}
	if !tt.HasText(tr(LocaleZHCN, keySearchRecent)) {
		t.Errorf("the zh-CN heading is missing, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "First session"); !ok {
		t.Errorf("the zh-CN palette dropped the catalog's title, texts %q", tt.Texts())
	}
	tt.Type("zzzz")
	if _, ok := tt.AccessByRole(ui.RoleStatus, tr(LocaleZHCN, keySearchEmpty)); !ok {
		t.Errorf("the zh-CN empty state is missing, texts %q", tt.Texts())
	}
}

// TestSearchPaletteShortcutHints checks the reference's row hints: each result
// names the shortcut that opens it, numbered from 1 in the order shown.
func TestSearchPaletteShortcutHints(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 1200, 1000)
	openPalette(t, tt)

	for i, title := range []string{"First session", "Second session"} {
		info := paletteResult(t, tt, title)
		if want := searchShortcutHint(i); !rowShowsText(tt, info.Bounds, want) {
			t.Errorf("the row for %q does not show the %q hint, texts %q", title, want, tt.Texts())
		}
	}
}

// rowShowsText reports whether the frame shows s inside box, which is how a
// test tells one row's hint from another's.
func rowShowsText(tt *ui.Tester, box ui.Rect, s string) bool {
	for _, text := range tt.Texts() {
		if text != s {
			continue
		}
		if rect, ok := tt.Find(text); ok && rect.X >= box.X && rect.X+rect.W <= box.X+box.W &&
			rect.Y >= box.Y && rect.Y+rect.H <= box.Y+box.H {
			return true
		}
	}
	return false
}
