package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// branchFromFixture is the shell with one user row that carries an App Server
// input sequence, so the row offers the cut point entry. It is view state only:
// no daemon or storage is read.
func branchFromFixture() *State {
	s := fixture()
	s.SelectedBranch = "main"
	s.Messages = []Message{
		{ID: "u1", Author: AuthorUser, Content: "why does this fail?", Sequence: 5},
		{ID: "a1", Author: AuthorAssistant, Content: "the frame is settled"},
	}
	return s
}

// TestBranchFromEntryOpensTheDialogAndSendsOnce checks the wired path: the user
// row offers "Branch from here", the entry is reachable from the keyboard, and
// the dialog's confirm calls the window once with the sequence immediately
// before the input and the typed prompt. The dialog closes first, and the shell
// changes no branch fact itself.
func TestBranchFromEntryOpensTheDialogAndSendsOnce(t *testing.T) {
	s := branchFromFixture()
	type request struct {
		at     int64
		prompt string
	}
	var got []request
	s.OnBranchFrom = func(sessionID, branchID string, at int64, prompt string) {
		if sessionID != "s1" || branchID != "main" {
			t.Errorf("OnBranchFrom got session=%q branch=%q, want s1 on main", sessionID, branchID)
		}
		got = append(got, request{at: at, prompt: prompt})
	}
	tt := ui.NewTester(s.View, 900, 700)

	entry, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyTimelineBranchFrom))
	if !ok {
		t.Fatalf("the user row has no branch entry, texts %q", tt.Texts())
	}
	if !entry.Focusable {
		t.Error("the branch entry is not reachable from the keyboard")
	}
	if err := tt.Click(tr(LocaleEN, keyTimelineBranchFrom)); err != nil {
		t.Fatalf("click the branch entry: %v", err)
	}
	tt.Frame()
	if !s.BranchFromConfirm {
		t.Fatalf("the entry did not open the prompt dialog")
	}
	field, ok := tt.AccessByRole(ui.RoleTextField, tr(LocaleEN, keyBranchFromPromptLabel))
	if !ok {
		t.Fatalf("the dialog has no prompt field, texts %q", tt.Texts())
	}
	tt.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tt.Type("side question")
	if err := tt.Click(tr(LocaleEN, keyBranchFromConfirm)); err != nil {
		t.Fatalf("confirm the branch: %v", err)
	}
	if len(got) != 1 || got[0].at != 4 || got[0].prompt != "side question" {
		t.Fatalf("OnBranchFrom got %#v, want one request at 4 with the typed prompt", got)
	}
	if s.BranchFromConfirm {
		t.Error("the dialog stayed open after the confirm")
	}
	if s.SelectedBranch != "main" {
		t.Errorf("the shell changed the displayed branch to %q", s.SelectedBranch)
	}
	// The dialog is gone, so a second confirm cannot send another request.
	if err := tt.Click(tr(LocaleEN, keyBranchFromConfirm)); err == nil {
		t.Error("the confirm entry is still clickable after the dialog closed")
	}
	if len(got) != 1 {
		t.Fatalf("OnBranchFrom got %d requests, want exactly one", len(got))
	}
}

// TestBranchFromEntryIsKeyboardReachable checks the entry from the keyboard:
// Tab reaches it and Enter opens the prompt dialog. The row action is a real
// button, not a pointer-only affordance.
func TestBranchFromEntryIsKeyboardReachable(t *testing.T) {
	s := branchFromFixture()
	s.OnBranchFrom = func(string, string, int64, string) {}
	tt := ui.NewTester(s.View, 900, 700)

	reached := false
	for i := 0; i < 60 && !reached; i++ {
		tt.Key(0, ui.KeyTab)
		reached = tt.Focused(tr(LocaleEN, keyTimelineBranchFrom))
	}
	if !reached {
		t.Fatalf("Tab does not reach the branch entry, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !s.BranchFromConfirm {
		t.Error("Enter on the branch entry did not open the prompt dialog")
	}
}

// TestBranchFromDialogCancelAndEscapeSendNothing checks both ways out: Cancel
// and Escape close the dialog without calling the window.
func TestBranchFromDialogCancelAndEscapeSendNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		quit func(t *testing.T, tt *ui.Tester)
	}{
		{"cancel", func(t *testing.T, tt *ui.Tester) {
			t.Helper()
			if err := tt.Click(tr(LocaleEN, keyBranchFromCancel)); err != nil {
				t.Fatalf("cancel the dialog: %v", err)
			}
		}},
		{"escape", func(t *testing.T, tt *ui.Tester) {
			t.Helper()
			tt.Key(0, ui.KeyEscape)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := branchFromFixture()
			calls := 0
			s.OnBranchFrom = func(string, string, int64, string) { calls++ }
			tt := ui.NewTester(s.View, 900, 700)

			if err := tt.Click(tr(LocaleEN, keyTimelineBranchFrom)); err != nil {
				t.Fatalf("open the dialog: %v", err)
			}
			tt.Frame()
			tc.quit(t, tt)
			tt.Frame()
			if calls != 0 {
				t.Errorf("%s sent %d branch requests, want none", tc.name, calls)
			}
			if s.BranchFromConfirm {
				t.Errorf("%s left the dialog open", tc.name)
			}
		})
	}
}

// TestBranchFromDialogKeepsAnEmptyPromptLocal checks the prompt gate: the
// confirm entry is disabled while the prompt is blank, so an empty request never
// reaches the App Server's validation.
func TestBranchFromDialogKeepsAnEmptyPromptLocal(t *testing.T) {
	s := branchFromFixture()
	calls := 0
	s.OnBranchFrom = func(string, string, int64, string) { calls++ }
	tt := ui.NewTester(s.View, 900, 700)

	if err := tt.Click(tr(LocaleEN, keyTimelineBranchFrom)); err != nil {
		t.Fatalf("open the dialog: %v", err)
	}
	tt.Frame()
	confirm, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyBranchFromConfirm))
	if !ok {
		t.Fatalf("the dialog has no confirm entry, texts %q", tt.Texts())
	}
	if !confirm.Disabled {
		t.Fatal("the confirm entry is enabled for an empty prompt")
	}
	if calls != 0 {
		t.Fatalf("OnBranchFrom got %d calls before any confirm", calls)
	}
	if !s.BranchFromConfirm {
		t.Error("the dialog closed on its own")
	}
}

// TestBranchFromEntryAppearsOnlyWithAVerifiedSequence checks the cut point fact:
// a row without an App Server sequence offers no action, because the shell has
// no sequence to cut from and must not guess one. With the window wired, the
// entry stays visible and explains the missing sequence instead.
func TestBranchFromEntryAppearsOnlyWithAVerifiedSequence(t *testing.T) {
	unwired := fixture()
	tt := ui.NewTester(unwired.View, 900, 700)
	if tt.HasText(tr(LocaleEN, keyTimelineBranchFrom)) {
		t.Fatalf("an unwired row without a sequence offered a cut point, texts %q", tt.Texts())
	}

	wired := fixture()
	wired.OnBranchFrom = func(string, string, int64, string) {}
	tt = ui.NewTester(wired.View, 900, 700)
	entry, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyTimelineBranchFrom))
	if !ok {
		t.Fatalf("the wired row with no sequence draws no explanation, texts %q", tt.Texts())
	}
	if !entry.Disabled {
		t.Errorf("the row with no sequence offered an enabled cut point: %+v", entry)
	}
	if !strings.Contains(entry.Description, reasonBranchCut) {
		t.Errorf("the row explains %q, want %q", entry.Description, reasonBranchCut)
	}
}

// TestBranchFromEntryStaysWIPWithoutTheCallback checks the region's WIP rule: a
// row that has a sequence but no wired callback keeps the entry visible,
// disabled and explained instead of pretending the branch can be created.
func TestBranchFromEntryStaysWIPWithoutTheCallback(t *testing.T) {
	s := branchFromFixture()
	tt := ui.NewTester(s.View, 900, 700)

	entry, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyTimelineBranchFrom))
	if !ok {
		t.Fatalf("the WIP branch entry is not drawn, texts %q", tt.Texts())
	}
	if !entry.Disabled {
		t.Errorf("the WIP branch entry is enabled: %+v", entry)
	}
	if !strings.Contains(entry.Description, reasonBranchFrom) {
		t.Errorf("the WIP branch entry explains %q, want %q", entry.Description, reasonBranchFrom)
	}
	if !tt.HasText(wipBadge) {
		t.Errorf("the WIP branch entry carries no %q badge", wipBadge)
	}
}

// TestBranchFromDialogClosesWhenTheBranchChanges checks the dialog's identity
// guard: a displayed branch that changed under the prompt drops it, so one
// branch's cut point is never sent for another.
func TestBranchFromDialogClosesWhenTheBranchChanges(t *testing.T) {
	s := branchFromFixture()
	s.OnBranchFrom = func(string, string, int64, string) {}
	tt := ui.NewTester(s.View, 900, 700)

	if err := tt.Click(tr(LocaleEN, keyTimelineBranchFrom)); err != nil {
		t.Fatalf("open the dialog: %v", err)
	}
	tt.Frame()
	if !s.BranchFromConfirm {
		t.Fatal("the dialog did not open")
	}
	s.SelectedBranch = "side"
	tt.Frame()
	if s.BranchFromConfirm {
		t.Error("the dialog survived a branch change")
	}
	if tt.HasText(tr(LocaleEN, keyBranchFromTitle)) {
		t.Errorf("the dialog is still drawn after a branch change, texts %q", tt.Texts())
	}
}
