package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// fixture is the shell's placeholder state: no App Server, no session facts.
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

// TestBranchListIsReadOnly checks that observed branches are shown as metadata
// only: the session.branches RPC is blocked/unverified, so there is no branch
// switch, cut point or side-chat action to click.
func TestBranchListIsReadOnly(t *testing.T) {
	s := fixture()
	s.Branches = []Branch{{ID: "main", Label: "main"}, {ID: "side", Label: "side"}}
	s.SelectedBranch = "main"
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Branches (read-only)"); !ok {
		t.Fatalf("branches are not grouped as read-only, texts %q", tt.Texts())
	}
	if !tt.HasText("main (current)") || !tt.HasText("side") {
		t.Errorf("branches are missing, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "side"); ok {
		t.Error("a branch is exposed as an action")
	}
	if !tt.HasText(reasonBranches) {
		t.Errorf("branch WIP reason is missing, texts %q", tt.Texts())
	}
	if err := tt.Click("side"); err != nil {
		t.Fatal(err)
	}
	if s.SelectedBranch != "main" {
		t.Errorf("branch selection changed to %q without a branch action", s.SelectedBranch)
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
	corner := img.RGBAAt(int(card.X)+1, int(card.Y)+1)
	if inside == outside {
		t.Errorf("the card face is the window background %v, so no card is drawn", inside)
	}
	if border == inside || border == outside {
		t.Errorf("the card outline is %v, the face %v and the background %v: no hairline is drawn", border, inside, outside)
	}
	if corner == inside {
		t.Errorf("the top-left corner is card face %v, so the corner is not rounded", corner)
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
	keysTT := ui.NewTester(keys.View, 900, 600)
	if err := keysTT.Click("Session keys"); err != nil {
		t.Fatal(err)
	}
	assertFramed(t, keysTT)
	if _, ok := keysTT.AccessByRole(ui.RoleRegion, "Keys panel"); !ok {
		t.Errorf("the keys browser is not drawn, texts %q", keysTT.Texts())
	}
}

// TestShellHeaderWeakensWithoutASession checks the toolbar's empty form: with
// no selected session it repeats no title and reports no run, and keeps the
// connection status and the appearance entry.
func TestShellHeaderWeakensWithoutASession(t *testing.T) {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	s.Messages = nil
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.Find(tr(LocaleEN, keyNoSession)); ok {
		t.Errorf("the toolbar repeats the no-session title, texts %q", tt.Texts())
	}
	if tt.HasText("Session: Idle") {
		t.Errorf("the toolbar reports a run with no session, texts %q", tt.Texts())
	}
	if !tt.HasText("App Server: Not connected") {
		t.Errorf("the toolbar lost the connection status, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyHeaderSettings)); !ok {
		t.Errorf("the toolbar lost the settings entry, texts %q", tt.Texts())
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
	if err := tt.Click("Second session"); err != nil {
		t.Fatal(err)
	}
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
// the sidebar's brand and a wrapped body paragraph are drawn larger on a larger
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

	smallBrand := textBox(t, small, "DimAgent")
	largeBrand := textBox(t, large, "DimAgent")
	if largeBrand.H <= smallBrand.H {
		t.Errorf("an 18 title is %.1f high, no taller than a 12 title's %.1f", largeBrand.H, smallBrand.H)
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
	brandBefore := textBox(t, tt, "DimAgent")

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
	if got := textBox(t, tt, "DimAgent"); got.H <= brandBefore.H {
		t.Errorf("the brand is %.1f high after choosing 18, no taller than %.1f", got.H, brandBefore.H)
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
