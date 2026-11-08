package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestSelectCallsCallback checks that choosing a session, by pointer or by
// keyboard, reaches the caller's callback with the chosen id. The shell changes
// Selected first and lets the callback open the session; it invents no rows.
func TestSelectCallsCallback(t *testing.T) {
	s := fixture()
	var got []string
	s.OnSelect = func(id string) { got = append(got, id) }
	tt := ui.NewTester(s.View, 900, 600)

	if err := tt.Click("Second session"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyHome)

	want := []string{"s2", "s1"}
	if len(got) != len(want) {
		t.Fatalf("OnSelect ran %d times with %q, want %q", len(got), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("OnSelect ran with %q, want %q", got, want)
		}
	}
}

// TestSubmitCallsCallback checks that the platform shortcut with Enter in the
// composer and the Send button both reach the caller's callback with the
// trimmed draft, and that the shell leaves the draft for the caller to clear
// once the App Server accepts it.
func TestSubmitCallsCallback(t *testing.T) {
	s := fixture()
	s.Connection = ConnReady
	var got []string
	s.OnSubmit = func(text string) { got = append(got, text) }
	tt := ui.NewTester(s.View, 900, 600)

	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}
	tt.Type("  hello  ")
	tt.Key(ComposerShortcutMods, ui.KeyEnter)
	if len(got) != 1 || got[0] != "hello" {
		t.Fatalf("the submit shortcut submitted %q, want [hello]", got)
	}
	if s.Draft != "  hello  " {
		t.Fatalf("the shell cleared the draft to %q before the caller accepted it", s.Draft)
	}

	if err := tt.Click("Send"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1] != "hello" {
		t.Fatalf("Send submitted %q, want [hello hello]", got)
	}
}

// TestSubmitIgnoresEmptyDraft checks that an empty or whitespace-only draft
// submits nothing, and that the Send entry is unavailable while there is
// nothing to send.
func TestSubmitIgnoresEmptyDraft(t *testing.T) {
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
		t.Error("Send is available with an empty draft")
	}
	if err := tt.Click("Send"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}
	tt.Type("   \n")
	tt.Key(ComposerShortcutMods, ui.KeyEnter)
	if calls != 0 {
		t.Fatalf("an empty draft submitted %d times", calls)
	}
}

// TestSubmitWithoutCallbackDoesNotPretendSuccess checks that a fixture shell
// with text but no submit callback offers no enabled Send and sends nothing:
// the entry cannot imply a submission the App Server never received.
func TestSubmitWithoutCallbackDoesNotPretendSuccess(t *testing.T) {
	s := fixture()
	s.Connection = ConnReady
	tt := ui.NewTester(s.View, 900, 600)

	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}
	tt.Type("hello")
	info, ok := tt.AccessByRole(ui.RoleButton, "Send")
	if !ok {
		t.Fatalf("the composer offers no Send entry, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("Send is available without a submit callback")
	}
	if err := tt.Click("Send"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ComposerShortcutMods, ui.KeyEnter)
	if s.Draft != "hello" {
		t.Errorf("draft %q after a send without a callback, want it untouched", s.Draft)
	}
	if len(s.Messages) != 2 {
		t.Errorf("%d rows after a send without a callback, want 2", len(s.Messages))
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("a send without a callback raised an alert, as if it had succeeded or failed")
	}
}

// TestSubmitUnavailableWithoutReadySession checks that Send stays unavailable
// until there is a selected session on a ready connection and history is not
// loading: the shell cannot send through a controller the window does not have.
func TestSubmitUnavailableWithoutReadySession(t *testing.T) {
	cases := []struct {
		name string
		edit func(*State)
	}{
		{"not connected", func(s *State) { s.Connection = ConnUnknown }},
		{"connecting", func(s *State) { s.Connection = ConnConnecting }},
		{"connection error", func(s *State) { s.Connection = ConnError }},
		{"loading history", func(s *State) { s.ConversationLoading = true }},
		{"no selection", func(s *State) { s.Selected = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := fixture()
			s.Connection = ConnReady
			calls := 0
			s.OnSubmit = func(string) { calls++ }
			tc.edit(s)
			tt := ui.NewTester(s.View, 900, 600)

			if err := tt.Click("Composer"); err != nil {
				t.Fatal(err)
			}
			tt.Type("hello")
			info, ok := tt.AccessByRole(ui.RoleButton, "Send")
			if !ok {
				t.Fatalf("the composer offers no Send entry, texts %q", tt.Texts())
			}
			if !info.Disabled {
				t.Error("Send is available without a ready selected session")
			}
			tt.Key(ComposerShortcutMods, ui.KeyEnter)
			if calls != 0 {
				t.Fatalf("the submit shortcut ran %d times without a ready selected session", calls)
			}
			if err := tt.Click("Send"); err != nil {
				t.Fatal(err)
			}
			if calls != 0 {
				t.Errorf("Send ran %d times without a ready selected session", calls)
			}
			if s.Draft != "hello" {
				t.Errorf("draft %q after an unavailable send, want it untouched", s.Draft)
			}
		})
	}
}
