package ui

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// fixture is the shell's placeholder state: no App Server, no session facts.

// clickRow clicks the center of a sidebar row by the text it shows. The row is
// the input target for a click, so Click on the inner text is rejected; the
// coordinates reach the row's own click handler.
func clickRow(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	rect, ok := tt.Find(label)
	if !ok {
		t.Fatalf("row %q is not drawn, texts %q", label, tt.Texts())
	}
	tt.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
}

func fixture() *State {
	return &State{
		Sessions: []Session{
			{ID: "s1", Title: "First session"},
			{ID: "s2", Title: "Second session"},
		},
		Selected: "s1",
		Messages: []Message{
			{ID: "m1", Author: AuthorUser, Content: "hello from the fixture"},
			{ID: "m2", Author: AuthorAssistant, Content: "second fixture row"},
		},
	}
}

// shellContextPanelState is the fixture with the changed-file facts the panel
// displays, as the caller read them. The selection and the fold map are the
// caller's view state, so a click in the shell test writes back into this
// state, exactly as a window's controller would keep it.
func shellContextPanelState() (*State, *string) {
	selected := "internal/ui/context_panel.go"
	s := fixture()
	s.ContextPanel = ContextPanelView{
		Files:         contextPanelFiles(),
		FilesReported: true,
		Selected:      &selected,
		Expanded:      map[string]*bool{},
	}
	return s, &selected
}

// TestShellRenders runs the whole view without a window and checks the shell's
// static parts.
func TestShellRenders(t *testing.T) {
	tt := ui.NewTester(fixture().View, 900, 600)
	for _, want := range []string{"First session", "Second session", "hello from the fixture", "second fixture row"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q, texts %q", want, tt.Texts())
		}
	}
}

// TestBranchListIsNotInTheSidebar checks that the observed branch metadata is
// no longer drawn in the reference's session column. The facts and callbacks
// stay in State for the session header menu that will carry them.
func TestBranchListIsNotInTheSidebar(t *testing.T) {
	s := fixture()
	s.Branches = []Branch{{ID: "main", Label: "main"}, {ID: "side", Label: "side"}}
	s.SelectedBranch = "main"
	tt := ui.NewTester(s.View, 900, 900)

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Branches (read-only)"); ok {
		t.Errorf("the sidebar still groups a branch list, texts %q", tt.Texts())
	}
	for _, gone := range []string{"main (current)", reasonBranches} {
		if tt.HasText(gone) {
			t.Errorf("the sidebar still draws %q, texts %q", gone, tt.Texts())
		}
	}
}

// TestShellFramesTheMainArea checks the main column's frame: the card sits
// 4.5 DIPs in from the window's top, right and bottom edges and against the
// sidebar on the left, and it draws a face, a hairline outline and a rounded
// corner rather than one flat window. The numbers come from
// docs/l5-reference.md §6 and dim-electron's chat-workspace utilities
// (global.css:19-31).
func TestShellFramesTheMainArea(t *testing.T) {
	const width, height = 900, 600
	tt := ui.NewTester(fixture().View, width, height)

	card, ok := tt.Find("Main area")
	if !ok {
		t.Fatalf("the main area is not framed, texts %q", tt.Texts())
	}
	if card.X < 255 || card.X > 257 {
		t.Errorf("the card starts at %.1f, want the sidebar's 256", card.X)
	}
	if top := card.Y; top < 4 || top > 5 {
		t.Errorf("the card starts %.1f below the top, want the 4.5 DIP frame", top)
	}
	if right := width - (card.X + card.W); right < 4 || right > 5 {
		t.Errorf("the card leaves %.1f at the right, want the 4.5 DIP frame", right)
	}
	if bottom := height - (card.Y + card.H); bottom < 4 || bottom > 5 {
		t.Errorf("the card leaves %.1f at the bottom, want the 4.5 DIP frame", bottom)
	}

	img := tt.Image()
	inside := img.RGBAAt(int(card.X+card.W-8), int(card.Y+card.H-8))
	outside := img.RGBAAt(int(card.X+card.W/2), 1)
	border := img.RGBAAt(int(card.X), int(card.Y+card.H/2))
	// The reference card is white on a white window in the light theme (this
	// tester draws the light one), so the fill cannot prove the frame; the
	// hairline does. The dark theme's lifted face is covered by tokens_test.
	if inside != outside {
		t.Errorf("the light card face %v differs from the white window %v", inside, outside)
	}
	if border == inside {
		t.Errorf("the card outline is %v, the face %v: no hairline is drawn", border, inside)
	}
}

// TestShellFrameSurroundsTheMainColumnPanels checks that the frame is the main
// column's, not the conversation's: the appearance surface and the keys browser
// draw inside the same card.
func TestShellFrameSurroundsTheMainColumnPanels(t *testing.T) {
	assertFramed := func(t *testing.T, tt *ui.Tester) {
		t.Helper()
		if _, ok := tt.Find("Main area"); !ok {
			t.Errorf("the main column lost its frame, texts %q", tt.Texts())
		}
	}

	settings := fixture()
	settingsTT := ui.NewTester(settings.View, 900, 600)
	if err := settingsTT.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	assertFramed(t, settingsTT)
	if !settingsTT.HasText("Appearance") {
		t.Errorf("the settings surface is not drawn, texts %q", settingsTT.Texts())
	}

	keys := shellKeysState()
	keys.ShowKeys = true
	keysTT := ui.NewTester(keys.View, 900, 600)
	assertFramed(t, keysTT)
	if _, ok := keysTT.AccessByRole(ui.RoleRegion, "Keys panel"); !ok {
		t.Errorf("the keys browser is not drawn, texts %q", keysTT.Texts())
	}
}

// TestShellHasNoHeaderWithoutASession checks the reference's empty state: no
// application bar with the connection or a Settings entry, just the hero. The
// sidebar's own Settings entry and the composer stay.
func TestShellHasNoHeaderWithoutASession(t *testing.T) {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	s.Messages = nil
	tt := ui.NewTester(s.View, 900, 900)

	for _, gone := range []string{"App Server: Not connected", "Session: Idle"} {
		if tt.HasText(gone) {
			t.Errorf("the empty state draws the application bar text %q, texts %q", gone, tt.Texts())
		}
	}
	if !tt.HasText(tr(LocaleEN, keyHeroTitle)) {
		t.Errorf("the empty state lost the hero, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keySidebarSettingsEntry)); !ok {
		t.Errorf("the sidebar lost its settings entry, texts %q", tt.Texts())
	}
}

// TestShellHeaderKeepsTheSessionTitle checks that the toolbar is unchanged
// while a session is selected.
func TestShellHeaderKeepsTheSessionTitle(t *testing.T) {
	tt := ui.NewTester(fixture().View, 900, 600)
	if !tt.HasText("First session") {
		t.Errorf("the toolbar lost the selected title, texts %q", tt.Texts())
	}
	if !tt.HasText("Session: Idle") {
		t.Errorf("the toolbar lost the run row, texts %q", tt.Texts())
	}
}

func TestShellSelectsSession(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)
	clickRow(t, tt, "Second session")
	if s.Selected != "s2" {
		t.Fatalf("selected %q, want s2", s.Selected)
	}
	if !tt.HasText("Second session") {
		t.Errorf("header after the click, texts %q", tt.Texts())
	}
}

// TestShellComposer checks that the composer edits the draft text.
func TestShellComposer(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}
	tt.Type("hi")
	if s.Draft != "hi" {
		t.Fatalf("draft %q, want hi", s.Draft)
	}
}

// textBox is the box of a text in the frame, failing the test when the text is
// not shown: the appearance tests measure a box, so a missing one is the thing
// they check, not a detail to skip.
func textBox(t *testing.T, tt *ui.Tester, text string) ui.Rect {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("%q is not in the frame, texts %q", text, tt.Texts())
	}
	return r
}

// TestShellAppliesSavedFontSize checks that a saved UI size reaches the frame:
// a sidebar entry and a wrapped body paragraph are drawn larger on a larger
// step, so the scale moves rather than one label.
func TestShellAppliesSavedFontSize(t *testing.T) {
	body := strings.Repeat("word ", 40) + "tail"
	render := func(size float32) *ui.Tester {
		s := fixture()
		s.Settings = SettingsView{FontSize: size}
		s.Messages = append(s.Messages, Message{ID: "m3", Author: AuthorAssistant, Content: body})
		return ui.NewTester(s.View, 900, 600)
	}
	small, large := render(12), render(18)

	smallEntry := textBox(t, small, "Sessions")
	largeEntry := textBox(t, large, "Sessions")
	if largeEntry.H <= smallEntry.H {
		t.Errorf("an 18 entry is %.1f high, no taller than a 12 entry's %.1f", largeEntry.H, smallEntry.H)
	}
	smallBody := textBox(t, small, body)
	largeBody := textBox(t, large, body)
	if largeBody.H <= smallBody.H || largeBody.W != smallBody.W {
		t.Errorf("an 18 paragraph is %.1f×%.1f, want it taller than the 12 one's %.1f×%.1f",
			largeBody.W, largeBody.H, smallBody.W, smallBody.H)
	}
}

// TestShellSettingsChoiceShowsOnTheNextFrame checks the wiring the appearance
// surface has into the frame: a choice made on the page changes what the next
// frame draws, in the same window, without the caller rebuilding the state.
func TestShellSettingsChoiceShowsOnTheNextFrame(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)
	entryBefore := textBox(t, tt, "Sessions")

	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	titleBefore := textBox(t, tt, "Appearance")
	if err := tt.Click("18"); err != nil {
		t.Fatal(err)
	}
	if s.Settings.FontSize != 18 {
		t.Fatalf("the surface reported %.0f, want 18", s.Settings.FontSize)
	}
	if got := textBox(t, tt, "Sessions"); got.H <= entryBefore.H {
		t.Errorf("the entry is %.1f high after choosing 18, no taller than %.1f", got.H, entryBefore.H)
	}
	if got := textBox(t, tt, "Appearance"); got.H <= titleBefore.H {
		t.Errorf("the page title is %.1f high after choosing 18, no taller than %.1f", got.H, titleBefore.H)
	}
}

// TestShellCompactDensityTightensTheRows checks the density end to end: the
// conversation's rows lose their outer padding and the gap above them, so the
// same rows start closer to the edge and higher up.
func TestShellCompactDensityTightensTheRows(t *testing.T) {
	render := func(density Density) *ui.Tester {
		s := fixture()
		s.Settings = SettingsView{Density: density}
		return ui.NewTester(s.View, 900, 600)
	}
	regular, compact := render(DensityRegular), render(DensityCompact)

	regularRow := textBox(t, regular, "second fixture row")
	compactRow := textBox(t, compact, "second fixture row")
	if compactRow.X >= regularRow.X {
		t.Errorf("compact starts the row at %.1f, not left of the regular %.1f", compactRow.X, regularRow.X)
	}
	if compactRow.Y > regularRow.Y {
		t.Errorf("compact starts the row at %.1f, below the regular %.1f", compactRow.Y, regularRow.Y)
	}
}

// TestShellCodeFontSizeReachesCodeText checks the code size end to end: an
// approval's tool input is drawn at the saved code size, while the body text
// beside it keeps the UI step.
func TestShellCodeFontSizeReachesCodeText(t *testing.T) {
	render := func(size float32) *ui.Tester {
		s := fixture()
		s.Settings = SettingsView{CodeFontSize: size}
		s.Messages = append(s.Messages, Message{
			ID:      "a1",
			Author:  AuthorApproval,
			Content: "exec — need a decision",
			Approval: &Approval{
				BranchID:     "main",
				RequestID:    "req-1",
				Call:         "exec",
				Reason:       "need a decision",
				InputDisplay: "inputmarker",
			},
		})
		return ui.NewTester(s.View, 900, 600)
	}
	small, large := render(settingsCodeMin), render(settingsCodeMax)

	smallCode := textBox(t, small, "inputmarker")
	largeCode := textBox(t, large, "inputmarker")
	if largeCode.H <= smallCode.H {
		t.Errorf("20px code is %.1f high, no taller than 12px code at %.1f", largeCode.H, smallCode.H)
	}
	smallBody := textBox(t, small, "need a decision")
	largeBody := textBox(t, large, "need a decision")
	if largeBody.H != smallBody.H {
		t.Errorf("the code size moved the body text from %.1f to %.1f", smallBody.H, largeBody.H)
	}
}

// TestShellThemePreferenceForcesTheTheme checks the theme choice: a forced
// light theme draws the same frame whatever the desktop's appearance is, and a
// forced dark one draws a different frame, so the choice still wins over the
// system theme while the saved type, family and density are applied on top.
func TestShellThemePreferenceForcesTheTheme(t *testing.T) {
	render := func(choice ThemeChoice, systemDark bool) *ui.Tester {
		s := fixture()
		s.Settings = SettingsView{Theme: choice}
		tt := ui.NewTester(s.View, 900, 600)
		tt.SetDark(systemDark)
		return tt
	}
	if a, b := render(ThemeLight, true), render(ThemeLight, false); !bytes.Equal(a.Image().Pix, b.Image().Pix) {
		t.Error("a forced light theme does not draw the same frame on a dark desktop")
	}
	if a, b := render(ThemeDark, true), render(ThemeDark, false); !bytes.Equal(a.Image().Pix, b.Image().Pix) {
		t.Error("a forced dark theme does not draw the same frame on a light desktop")
	}
	light, dark := render(ThemeLight, false), render(ThemeDark, false)
	if bytes.Equal(light.Image().Pix, dark.Image().Pix) {
		t.Error("the light and dark choices draw the same frame")
	}
}

// TestShellTogglesTheContextPanel checks the toolbar entry end to end: it is
// named "Context panel", it is absent while the panel is closed, one click
// draws the panel with the caller's changed-file facts, and the next click
// returns the room to the conversation.
func TestShellTogglesTheContextPanel(t *testing.T) {
	s, selected := shellContextPanelState()
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleButton, contextPanelToggleLabel); !ok {
		t.Fatalf("the toolbar has no context panel entry, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); ok {
		t.Fatalf("the panel is drawn before it is opened, texts %q", tt.Texts())
	}
	if tt.HasText("context_panel.go") {
		t.Fatalf("the changed files are drawn before the panel opens, texts %q", tt.Texts())
	}

	if err := tt.Click(contextPanelToggleLabel); err != nil {
		t.Fatal(err)
	}
	if !s.ShowContextPanel {
		t.Fatal("the entry did not open the panel")
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); !ok {
		t.Fatalf("the opener did not draw the panel, texts %q", tt.Texts())
	}
	for _, want := range []string{"Changes", "3 files", "context_panel.go", "README.md", "internal/ui/context_panel.go"} {
		if !tt.HasText(want) {
			t.Errorf("the panel is missing %q, texts %q", want, tt.Texts())
		}
	}
	if !tt.HasText("hello from the fixture") {
		t.Errorf("the conversation left the main column when the panel opened, texts %q", tt.Texts())
	}
	// The panel still writes the caller's own view state: a file click lands
	// in the pointer the caller passed and the detail follows it.
	if err := tt.Click("client.go"); err != nil {
		t.Fatal(err)
	}
	if *selected != "internal/appserver/client.go" {
		t.Errorf("the file click wrote %q, want internal/appserver/client.go", *selected)
	}
	if !tt.HasText("internal/appserver/client.go") || !tt.HasText("added") {
		t.Errorf("the detail did not follow the shell's selection, texts %q", tt.Texts())
	}

	if err := tt.Click(contextPanelToggleLabel); err != nil {
		t.Fatal(err)
	}
	if s.ShowContextPanel {
		t.Fatal("the entry did not close the panel")
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); ok {
		t.Fatalf("the panel is still drawn after it was closed, texts %q", tt.Texts())
	}
}

// TestShellContextPanelWIPWithoutFacts checks the panel's missing-data state
// inside the shell: a zero ContextPanel draws the WIP statement and its reason,
// never an empty tree or a fabricated file.
func TestShellContextPanelWIPWithoutFacts(t *testing.T) {
	s := fixture()
	s.ShowContextPanel = true
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); !ok {
		t.Fatalf("the panel is not drawn, texts %q", tt.Texts())
	}
	if !tt.HasText("WIP") || !tt.HasText(reasonContextFiles) {
		t.Errorf("the panel does not explain its missing data, texts %q", tt.Texts())
	}
	if tt.HasText("No changed files") {
		t.Errorf("the panel claims a clean workspace without facts, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleTree, "Changed files"); ok {
		t.Errorf("the panel draws a tree without facts, texts %q", tt.Texts())
	}
}

// TestShellContextPanelHiddenWithoutSession checks that the panel is the
// conversation's: with no selected session the toolbar offers no entry and the
// panel stays out of the frame, whatever the window's own toggle says.
func TestShellContextPanelHiddenWithoutSession(t *testing.T) {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	s.Messages = nil
	s.ShowContextPanel = true
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleButton, contextPanelToggleLabel); ok {
		t.Errorf("the toolbar offers the panel with no session, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); ok {
		t.Errorf("the panel is drawn with no session, texts %q", tt.Texts())
	}
}

// contextPanelBounds is the panel region's box, failing the test when the panel
// is not drawn.
func contextPanelBounds(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleRegion, "Context panel")
	if !ok {
		t.Fatalf("the panel is not drawn, texts %q", tt.Texts())
	}
	return info.Bounds
}

// TestShellContextPanelWidth checks the panel's place and width: 320 DIPs wide
// against the main card's right edge, with the conversation left of it. A
// window too narrow for both shrinks the panel instead of pushing the card past
// the window's edge.
func TestShellContextPanelWidth(t *testing.T) {
	width := func(w int) (ui.Rect, ui.Rect, *ui.Tester) {
		s, _ := shellContextPanelState()
		s.ShowContextPanel = true
		tt := ui.NewTester(s.View, w, 600)
		card, ok := tt.Find("Main area")
		if !ok {
			t.Fatalf("%d-wide window: the main area is not framed, texts %q", w, tt.Texts())
		}
		return contextPanelBounds(t, tt), card, tt
	}

	panel, card, tt := width(900)
	if panel.W < contextPanelW-1 || panel.W > contextPanelW+1 {
		t.Errorf("the panel is %.1f wide, want the reference's %d", panel.W, contextPanelW)
	}
	if right := card.X + card.W; panel.X+panel.W > right+0.5 {
		t.Errorf("the panel ends at %.1f, past the card's %.1f", panel.X+panel.W, right)
	}
	if conversation := textBox(t, tt, "hello from the fixture"); conversation.X >= panel.X {
		t.Errorf("the conversation starts at %.1f, not left of the panel's %.1f", conversation.X, panel.X)
	}

	narrow, narrowCard, narrowTT := width(480)
	if narrow.W >= panel.W {
		t.Errorf("a 480-wide window keeps the panel at %.1f, no narrower than %.1f", narrow.W, panel.W)
	}
	if narrow.X+narrow.W > narrowCard.X+narrowCard.W+0.5 {
		t.Errorf("the narrow panel ends at %.1f, past the card's %.1f", narrow.X+narrow.W, narrowCard.X+narrowCard.W)
	}
	if narrowCard.X+narrowCard.W > 480.5 {
		t.Errorf("the card ends at %.1f, past the 480-wide window", narrowCard.X+narrowCard.W)
	}
	if narrow.W <= 0 || narrow.H <= 0 {
		t.Errorf("the narrow panel has no room: %+v", narrow)
	}
	if !narrowTT.HasText(contextPanelToggleLabel) {
		t.Errorf("the narrow window lost the panel entry, texts %q", narrowTT.Texts())
	}
}

// TestShellContextPanelYieldsToSettingsAndKeys checks the mutual exclusion: the
// appearance surface and the keys browser take the main column from the
// conversation, so the panel is not drawn and the toolbar hides its entry while
// either is open. The window's own toggle is kept, and the panel returns with
// the conversation.
func TestShellContextPanelYieldsToSettingsAndKeys(t *testing.T) {
	s, _ := shellContextPanelState()
	s.ShowContextPanel = true
	tt := ui.NewTester(s.View, 900, 600)
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); !ok {
		t.Fatalf("the panel is not drawn before Settings opens, texts %q", tt.Texts())
	}

	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	if !s.ShowSettings {
		t.Fatal("the appearance surface did not open")
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); ok {
		t.Errorf("the panel is drawn over the appearance surface, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, contextPanelToggleLabel); ok {
		t.Errorf("the panel entry is offered while the appearance surface shows, texts %q", tt.Texts())
	}
	if !s.ShowContextPanel {
		t.Error("opening the appearance surface dropped the window's panel toggle")
	}

	if err := tt.Click("Conversation"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); !ok {
		t.Fatalf("the panel did not return with the conversation, texts %q", tt.Texts())
	}

	keys := shellKeysState()
	keys.ShowContextPanel = true
	keys.ShowKeys = true
	keysTT := ui.NewTester(keys.View, 900, 600)
	if _, ok := keysTT.AccessByRole(ui.RoleRegion, "Keys panel"); !ok {
		t.Fatalf("the keys browser did not open, texts %q", keysTT.Texts())
	}
	if _, ok := keysTT.AccessByRole(ui.RoleRegion, "Context panel"); ok {
		t.Errorf("the panel is drawn over the keys browser, texts %q", keysTT.Texts())
	}
	if _, ok := keysTT.AccessByRole(ui.RoleButton, contextPanelToggleLabel); ok {
		t.Errorf("the panel entry is offered while the keys browser shows, texts %q", keysTT.Texts())
	}
}

// TestShellContextPanelToggleTabStop checks that the toolbar entry is a real
// Tab stop: Tab reaches it, and Enter opens the panel, so a keyboard user does
// not need the pointer.
func TestShellContextPanelToggleTabStop(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	reached := false
	for i := 0; i < 40; i++ {
		tt.Key(0, ui.KeyTab)
		if tt.Focused(contextPanelToggleLabel) {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatalf("Tab does not reach %q, texts %q", contextPanelToggleLabel, tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if !s.ShowContextPanel {
		t.Fatalf("Enter on the entry did not open the panel, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Context panel"); !ok {
		t.Errorf("the keyboard opener did not draw the panel, texts %q", tt.Texts())
	}
}

// composerCardBottom is the composer card's bottom edge, read from the Send
// entry and the structure the region documents: the tool row's inset, the
// divider and the ≈36 DIP context band under it. It lets a shell test measure
// where the card sits without naming a group the accessibility tree does not.
func composerCardBottom(t *testing.T, tt *ui.Tester, k tokens) float32 {
	t.Helper()
	send := textBox(t, tt, "Send")
	return send.Y + send.H + k.edgePad + k.hairline + k.space(composerContextSpan) + k.hairline
}

// TestShellEmptyStateCentresTheHeroAndComposer checks the reference's empty
// workspace: the hero and the composer are one group in the middle of the card,
// the composer directly under the hero, not pinned to the window's bottom.
func TestShellEmptyStateCentresTheHeroAndComposer(t *testing.T) {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	s.Messages = nil
	const width, height = 800, 720
	tt := ui.NewTester(s.View, width, height)
	k := newTokens(ui.LightTheme())

	title := textBox(t, tt, tr(LocaleEN, keyHeroTitle))
	editor := textBox(t, tt, "Composer")
	if title.Y+title.H >= editor.Y {
		t.Errorf("the hero runs into the composer: heading ends at %.1f, the card starts at %.1f",
			title.Y+title.H, editor.Y)
	}
	mark, pixels := colorBox(tt, ui.Rect{X: k.sidebarW + k.framePad, W: width - k.sidebarW, H: title.Y}, k.primary)
	if pixels == 0 {
		t.Fatalf("no primary-green pixels above the heading, so the brand mark is missing")
	}

	// The pair is centred: the room above the mark and the room under the card
	// are the same, and the card is not resting on the window's bottom edge.
	topGap := mark.Y - k.heroPad - k.framePad
	bottomGap := float32(height) - k.framePad - composerCardBottom(t, tt, k)
	if math.Abs(float64(topGap-bottomGap)) > 2 {
		t.Errorf("the empty group is not centred: %.1f above the hero, %.1f below the composer", topGap, bottomGap)
	}
	if bottomGap < k.space(composerContextSpan)*2 {
		t.Errorf("the composer is pinned to the bottom: only %.1f DIP under it", bottomGap)
	}
}

// TestShellConversationKeepsTheComposerAtTheBottom checks the other half of the
// layout: once the conversation has rows the composer stays on the card's
// bottom edge, as the reference's conversation does.
func TestShellConversationKeepsTheComposerAtTheBottom(t *testing.T) {
	const height = 720
	tt := ui.NewTester(fixture().View, 800, height)
	k := newTokens(ui.LightTheme())

	cardBottom := composerCardBottom(t, tt, k)
	if gap := float32(height) - k.framePad - cardBottom; math.Abs(float64(gap)) > 1 {
		t.Errorf("the composer ends %.1f before the card's bottom, want it on the edge", gap)
	}
}
