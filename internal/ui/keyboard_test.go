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
	if err := tt.Click("First session"); err != nil {
		t.Fatal(err)
	}

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

// TestTabOrder checks the shell's Tab stops: the sidebar, then the composer,
// round again, so a keyboard user reaches both from the window.
func TestTabOrder(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Sessions") {
		t.Fatalf("the first Tab does not reach the session sidebar, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Composer") {
		t.Fatalf("the second Tab does not reach the composer, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Sessions") {
		t.Fatalf("Tab does not go round to the sidebar, texts %q", tt.Texts())
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
	if err := tt.Click("Second session"); err != nil {
		t.Fatal(err)
	}
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
