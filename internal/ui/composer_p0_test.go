package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// composerTestView builds a composer in isolation, with the submission gate the
// tests need. The harness owns the draft pointer and the focus handle, so the
// region reads and writes the same values across frames as it does in the shell.
func composerTestView(t *testing.T, canSubmit, sending bool, draft string, onSubmit func(string)) (ComposerView, *ui.Tester) {
	t.Helper()
	view := ComposerView{
		Draft:       new(string),
		CanSubmit:   canSubmit,
		Sending:     sending,
		Placeholder: composerPlaceholder,
		Shortcut:    KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
		Focus:       new(ui.Handle),
	}
	*view.Draft = draft
	a := Actions{OnSubmit: onSubmit}
	tt := ui.NewTester(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, a)
	}, 720, 480)
	return view, tt
}

// TestComposerGrowsWithTextAndCaps checks the editor's adaptive height: it
// starts at the spec's 60 DIP floor, grows as lines are added, and stops at the
// 300 DIP cap, past which the text area scrolls inside instead of growing the
// card.
func TestComposerGrowsWithTextAndCaps(t *testing.T) {
	_, tt := composerTestView(t, false, false, "", nil)
	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}

	empty, ok := tt.Find("Composer")
	if !ok {
		t.Fatalf("the composer is not drawn, texts %q", tt.Texts())
	}
	if empty.H < composerMinH-0.5 {
		t.Fatalf("empty editor is %.1f high, want at least the %d DIP floor", empty.H, composerMinH)
	}

	// Shift+Enter always inserts a newline, so it is the key that grows the
	// draft without the submit gate taking it.
	tt.Type("first line")
	for range 4 {
		tt.Key(ui.Shift, ui.KeyEnter)
		tt.Type("another line")
	}
	tt.Frame()
	grown, ok := tt.Find("Composer")
	if !ok {
		t.Fatalf("the composer is not drawn after typing, texts %q", tt.Texts())
	}
	if grown.H <= empty.H {
		t.Fatalf("editor is %.1f high after five lines, want more than the empty %.1f", grown.H, empty.H)
	}
	if grown.H > composerMaxH+0.5 {
		t.Fatalf("editor is %.1f high with five lines, above the %d DIP cap", grown.H, composerMaxH)
	}

	for range 80 {
		tt.Key(ui.Shift, ui.KeyEnter)
		tt.Type("more")
	}
	tt.Frame()
	capped, ok := tt.Find("Composer")
	if !ok {
		t.Fatalf("the composer is not drawn after many lines, texts %q", tt.Texts())
	}
	if capped.H > composerMaxH+0.5 {
		t.Fatalf("editor is %.1f high past the cap, want at most %d", capped.H, composerMaxH)
	}
	if capped.H < composerMaxH-1 {
		t.Fatalf("editor is %.1f high past the cap, want it to reach %d before scrolling", capped.H, composerMaxH)
	}
}

// TestComposerEnterSubmitsOnce checks that plain Enter submits the trimmed
// draft exactly once, that Shift+Enter inserts a newline instead, and that the
// shell leaves the draft for the caller to clear.
func TestComposerEnterSubmitsOnce(t *testing.T) {
	s := fixture()
	s.Connection = ConnReady
	s.Draft = "  hello  "
	var got []string
	s.OnSubmit = func(text string) { got = append(got, text) }
	tt := ui.NewTester(s.View, 900, 600)

	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if len(got) != 1 || got[0] != "hello" {
		t.Fatalf("Enter submitted %q, want [hello]", got)
	}
	if s.Draft != "  hello  " {
		t.Fatalf("draft %q after Enter, want the caller to own the clear", s.Draft)
	}

	tt.Key(ui.Shift, ui.KeyEnter)
	if len(got) != 1 {
		t.Fatalf("Shift+Enter submitted again: %q", got)
	}
	if s.Draft != "  hello  \n" {
		t.Fatalf("draft %q after Shift+Enter, want a newline and no clear", s.Draft)
	}
}

// TestComposerSendDisabled checks the two gates that disable Send: an empty or
// whitespace-only draft, and a submission already in flight. Enter takes the
// message only when both allow it, and otherwise leaves the editor's newline.
func TestComposerSendDisabled(t *testing.T) {
	t.Run("empty draft", func(t *testing.T) {
		s := fixture()
		s.Connection = ConnReady
		calls := 0
		s.OnSubmit = func(string) { calls++ }
		tt := ui.NewTester(s.View, 900, 600)

		info, ok := tt.AccessByRole(ui.RoleButton, "Send")
		if !ok {
			t.Fatalf("the composer offers no Send entry, texts %q", tt.Texts())
		}
		if !info.Disabled {
			t.Fatal("Send is available with an empty draft")
		}
		if err := tt.Click("Composer"); err != nil {
			t.Fatal(err)
		}
		tt.Type("   \n")
		tt.Key(0, ui.KeyEnter)
		if calls != 0 {
			t.Fatalf("a whitespace-only draft submitted %d times", calls)
		}
	})

	t.Run("sending", func(t *testing.T) {
		calls := 0
		_, tt := composerTestView(t, true, true, "hello", func(string) { calls++ })

		info, ok := tt.AccessByRole(ui.RoleButton, "Send")
		if !ok {
			t.Fatalf("the composer offers no Send entry, texts %q", tt.Texts())
		}
		if !info.Disabled {
			t.Fatal("Send is available while a submission is in flight")
		}
		if err := tt.Click("Composer"); err != nil {
			t.Fatal(err)
		}
		tt.Key(0, ui.KeyEnter)
		if calls != 0 {
			t.Fatalf("Enter submitted %d times while Sending", calls)
		}
	})
}

// TestComposerWIPEntriesDoNothing checks that the model and mode selectors are
// visible, disabled and explained with their real gap, and that clicking them
// runs nothing and leaves the draft and the window's state alone.
func TestComposerWIPEntriesDoNothing(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	for _, tc := range []struct {
		label, reason string
	}{
		{"Mode", reasonMode},
		{"Model", reasonModel},
	} {
		info, ok := tt.AccessByRole(ui.RoleButton, tc.label)
		if !ok {
			t.Fatalf("%s is not drawn as an entry, texts %q", tc.label, tt.Texts())
		}
		if !info.Disabled {
			t.Errorf("%s is not disabled", tc.label)
		}
		if info.Focusable {
			t.Errorf("%s takes the keyboard focus, so it is a Tab stop", tc.label)
		}
		if !strings.Contains(info.Description, tc.reason) {
			t.Errorf("%s description %q does not explain the gap %q", tc.label, info.Description, tc.reason)
		}
		rect, ok := tt.Find(tc.label)
		if !ok {
			t.Fatalf("%s has no drawn box", tc.label)
		}
		tt.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
		if tt.Focused(tc.label) {
			t.Errorf("clicking %s focused it", tc.label)
		}
	}

	if s.Draft != "" || s.Selected != "s1" {
		t.Errorf("a WIP click changed the shell: draft=%q selected=%q", s.Draft, s.Selected)
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("a WIP click raised an alert, as if an action had failed or succeeded")
	}
}

// TestComposerTabOrder checks the composer's Tab stops after the P0 layout: the
// attachment entry, then the editor, with the disabled model and mode entries
// and the hint skipped. The order is the shell's documented one, so adding the
// two selectors did not insert a stop into it.
func TestComposerTabOrder(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	for _, want := range []string{
		"Update", "New session", "Search sessions", "Scheduled tasks", "Skills and plugins",
		"Session keys", "Group", "Sessions", "Pin", "Rename", "Archive",
		"Sign in to Dim", "Settings", "Divider", "Reconnect", "Settings", "Context panel",
	} {
		tt.Key(0, ui.KeyTab)
		if !tt.Focused(want) {
			t.Fatalf("Tab does not reach %q, texts %q", want, tt.Texts())
		}
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Attach file") {
		t.Fatalf("Tab does not reach Attach file, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Composer") {
		t.Fatalf("Tab does not reach the editor next, texts %q", tt.Texts())
	}
	if tt.Focused("Mode") || tt.Focused("Model") || tt.Focused("Send") {
		t.Errorf("a disabled composer entry took the focus")
	}
}

// TestComposerActionRowSitsBelowEditor checks the P0 layout: the editor is on
// top, the shortcut hint under it, and the action row at the card's bottom with
// the attachment and mode entries on the left and the model entry and Send on
// the right.
func TestComposerActionRowSitsBelowEditor(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	editor, ok := tt.Find("Composer")
	if !ok {
		t.Fatalf("the editor is not drawn, texts %q", tt.Texts())
	}
	if !tt.HasText(composerHint) {
		t.Errorf("the shortcut hint %q is missing, texts %q", composerHint, tt.Texts())
	}
	left := []string{"Attach file", "Mode"}
	right := []string{"Model", "Send"}
	for _, label := range append(append([]string{}, left...), right...) {
		rect, ok := tt.Find(label)
		if !ok {
			t.Fatalf("%q is not drawn, texts %q", label, tt.Texts())
		}
		if rect.Y < editor.Y+editor.H {
			t.Errorf("%q sits at y=%.1f, inside or above the editor ending at %.1f", label, rect.Y, editor.Y+editor.H)
		}
	}
	attach, _ := tt.Find("Attach file")
	model, _ := tt.Find("Model")
	if attach.X > editor.X+editor.W/2 || model.X < editor.X+editor.W/2 {
		t.Errorf("the action row is not split left tools / right model: attach x=%.1f, model x=%.1f, editor %.1f..%.1f",
			attach.X, model.X, editor.X, editor.X+editor.W)
	}
}
