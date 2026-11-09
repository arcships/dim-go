package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestSidebarKeyboardNavigation checks that the session sidebar takes the
// keyboard: Up, Down, Home and End move the choice while it has the focus, as
// ui.Sidebar documents, and the header follows it.
func TestSidebarKeyboardNavigation(t *testing.T) {
	s := fixture()
	s.Sessions = []Session{
		{ID: "s1", Title: "First session"},
		{ID: "s2", Title: "Second session"},
		{ID: "s3", Title: "Third session"},
	}
	s.Selected = "s1"
	tt := ui.NewTester(s.View, 900, 600)
	clickRow(t, tt, "First session")

	steps := []struct {
		key  ui.Key
		want string
	}{
		{ui.KeyDown, "s2"},
		{ui.KeyDown, "s3"},
		{ui.KeyDown, "s3"}, // the last item holds the choice
		{ui.KeyUp, "s2"},
		{ui.KeyHome, "s1"},
		{ui.KeyEnd, "s3"},
	}
	for _, step := range steps {
		tt.Key(0, step.key)
		if s.Selected != step.want {
			t.Fatalf("after %v selected %q, want %q", step.key, s.Selected, step.want)
		}
	}
	if !tt.HasText("Third session") {
		t.Errorf("header does not follow the keyboard choice, texts %q", tt.Texts())
	}
}

// TestTabOrder checks the shell's Tab stops in tree order: the sidebar's
// header, entries, thread actions and account footer, the thread list, the
// split divider, the conversation bar's session menu, its conditional
// reconnect entry and its context panel toggle, the composer's attachment
// entry, then the composer, round again. The editor entry is unavailable and
// takes no focus, so it is skipped; the reconnect WIP entry stays a Tab stop so
// its reason is read, and the send entry is unavailable too—the fixture's draft
// is empty and it has no submit callback. The WIP entries stay in the order on
// purpose: a keyboard or screen-reader user finds them, reads that they are
// unavailable and why, and moves on.
//
// The sidebar's settings entry and the conversation menu's share no label; the
// tab order visits the sidebar's first because the sidebar is drawn first.
func TestTabOrder(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	for _, want := range []string{
		"New session", "Search", "Scheduled tasks", "Skills and plugins",
		"Sessions", "Sign in to Dim", "Settings", "Divider", "Session menu", "Reconnect",
		"Context panel", "Attach file", tr(LocaleEN, keyComposerModelChip), "Composer", "New session",
	} {
		tt.Key(0, ui.KeyTab)
		if !tt.Focused(want) {
			t.Fatalf("Tab does not reach %q, texts %q", want, tt.Texts())
		}
	}
}

// TestComposerShortcut checks that the shell's documented shortcut moves the
// keyboard focus to the composer, from no focus and from the sidebar, and that
// the composer then takes the typing.
func TestComposerShortcut(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	tt.Key(ComposerShortcutMods, ComposerShortcutKey)
	info, ok := tt.AccessByRole(ui.RoleTextField, "Composer")
	if !ok {
		t.Fatalf("composer is not exposed to accessibility, texts %q", tt.Texts())
	}
	if !info.Focused {
		t.Fatalf("the composer does not have the focus after %v+%v", ComposerShortcutMods, ComposerShortcutKey)
	}
	tt.Type("from the shortcut")
	if s.Draft != "from the shortcut" {
		t.Fatalf("draft %q after the shortcut, want %q", s.Draft, "from the shortcut")
	}

	// From the sidebar, where the arrows move the choice.
	clickRow(t, tt, "Second session")
	tt.Key(0, ui.KeyUp)
	if s.Selected != "s1" {
		t.Fatalf("selected %q after the arrow, want s1", s.Selected)
	}
	tt.Key(ComposerShortcutMods, ComposerShortcutKey)
	if !tt.Focused("Composer") {
		t.Fatalf("the composer does not have the focus after the shortcut from the sidebar")
	}
	tt.Type("!")
	if s.Draft != "from the shortcut!" {
		t.Fatalf("draft %q after the second shortcut, want %q", s.Draft, "from the shortcut!")
	}
}
