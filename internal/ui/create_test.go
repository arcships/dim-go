package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestCreateSessionCallsCallbackOnce checks the wired entry: one click reaches
// the window's create callback exactly once, and the shell itself adds no row
// and changes no selection, because the new session is the App Server's answer
// and the window publishes it only after session.create returned.
func TestCreateSessionCallsCallbackOnce(t *testing.T) {
	s := fixture()
	calls := 0
	s.OnCreateSession = func() { calls++ }
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "New session")
	if !ok {
		t.Fatalf("the sidebar offers no New session entry, texts %q", tt.Texts())
	}
	if info.Disabled {
		t.Error("New session is unavailable although the window wired session.create")
	}
	if err := tt.Click("New session"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("New session ran the create callback %d times, want 1", calls)
	}
	if len(s.Sessions) != 2 || s.Selected != "s1" {
		t.Fatalf("the shell invented a session or moved the selection: sessions=%+v selected=%q", s.Sessions, s.Selected)
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("a click raised an alert before the App Server answered")
	}
}

// TestCreateSessionDisabledWhileCreating checks that one unanswered
// session.create owns the entry: while it is in flight the button is disabled
// and a click reaches nothing, so a double click cannot ask for two sessions.
func TestCreateSessionDisabledWhileCreating(t *testing.T) {
	s := fixture()
	calls := 0
	s.OnCreateSession = func() { calls++ }
	s.CreatingSession = true
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "New session")
	if !ok {
		t.Fatalf("the sidebar offers no New session entry, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("New session is available while a creation is in flight")
	}
	if err := tt.Click("New session"); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("a creation in flight started %d more, want 0", calls)
	}

	s.CreatingSession = false
	tt.Frame()
	info, ok = tt.AccessByRole(ui.RoleButton, "New session")
	if !ok || info.Disabled {
		t.Errorf("New session did not become available after the creation settled: %+v, found %v", info, ok)
	}
	if err := tt.Click("New session"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("New session ran %d times after the creation settled, want 1", calls)
	}
}

// TestCreateSessionWithoutCallbackStaysWIP checks the unwired shell: the entry
// keeps its place, stays unavailable, and names session.create in its
// description and tooltip instead of a visible badge.
func TestCreateSessionWithoutCallbackStaysWIP(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "New session")
	if !ok {
		t.Fatalf("the sidebar offers no New session entry, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("New session is available without a create callback")
	}
	if !strings.Contains(info.Description, "session.create") {
		t.Errorf("description %q does not name session.create", info.Description)
	}
	if err := tt.Click("New session"); err != nil {
		t.Fatal(err)
	}
	if len(s.Sessions) != 2 || s.Selected != "s1" {
		t.Fatalf("an unwired click changed the catalog: sessions=%+v selected=%q", s.Sessions, s.Selected)
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("an unwired click raised an alert, as if an action had been sent")
	}
}
