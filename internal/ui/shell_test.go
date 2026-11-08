package ui

import (
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

// TestShellSelectsSession checks that the sidebar drives the header, which is
// the only interaction the shell claims.
func TestShellSelectsBranch(t *testing.T) {
	s := fixture()
	s.Branches = []Branch{{ID: "main", Label: "main"}, {ID: "side", Label: "side"}}
	s.SelectedBranch = "main"
	var got string
	s.OnBranchSelect = func(id string) { got = id }
	tt := ui.NewTester(s.View, 900, 600)
	if err := tt.Click("side"); err != nil {
		t.Fatal(err)
	}
	if got != "side" {
		t.Fatalf("branch callback = %q, want side", got)
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
