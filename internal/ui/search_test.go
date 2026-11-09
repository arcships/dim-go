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

// openPalette opens the palette with the window's shortcut and checks that its
// input took the focus.
func openPalette(t *testing.T, tt *ui.Tester) {
	t.Helper()
	tt.Key(searchShortcutMods, searchShortcutKey)
	if _, ok := paletteDialog(tt); !ok {
		t.Fatalf("the shortcut did not open the palette, texts %q", tt.Texts())
	}
	if !tt.Focused(tr(LocaleEN, keySearchPlaceholder)) {
		t.Fatalf("the palette's input did not take the focus, texts %q", tt.Texts())
	}
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
