package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// noticesState is the fixture with one failure, one confirmed outcome and one
// projection notice, so the strip has all three sources at once.
func noticesState() *State {
	s := fixture()
	s.ActionError = "Submission result unknown: the App Server did not answer"
	s.ActionNotices = []Notice{
		{Kind: "submit.unknown", ID: s.ActionError},
		{Kind: "submit.accepted", ID: "submission-1"},
	}
	s.Notices = []Notice{{Kind: "approval"}}
	return s
}

// TestNoticeStripShowsFailuresAndOutcomes checks the strip's content and color:
// the window's failure is drawn in the danger color, the confirmed outcome
// stays muted, the projection notice is visible, and the action error is not
// repeated from ActionNotices.
func TestNoticeStripShowsFailuresAndOutcomes(t *testing.T) {
	s := noticesState()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 700)
	k := newTokens(theme)

	for _, want := range []string{s.ActionError, "submit.accepted: submission-1", "approval"} {
		if !tt.HasText(want) {
			t.Errorf("the notice strip is missing %q, texts %q", want, tt.Texts())
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); !ok {
		t.Fatalf("the notices are not a live-region alert, texts %q", tt.Texts())
	}
	failure := textBox(t, tt, s.ActionError)
	if _, danger := colorBox(tt, failure, k.danger); danger == 0 {
		t.Error("the failure row is not drawn in the danger color")
	}
	outcome := textBox(t, tt, "submit.accepted: submission-1")
	if _, muted := colorBox(tt, outcome, k.mutedForeground); muted == 0 {
		t.Error("the confirmed outcome is not drawn muted")
	}
}

// TestNoticeStripDismissClearsTheWindowOwnedState checks the close entry: one
// click clears the action error, the action notices and the transient
// projection notices, and the strip leaves the frame.
func TestNoticeStripDismissClearsTheWindowOwnedState(t *testing.T) {
	s := noticesState()
	tt := ui.NewTester(s.View, 900, 700)

	if err := tt.Click(tr(LocaleEN, keyNoticesDismiss)); err != nil {
		t.Fatalf("dismiss the notices: %v", err)
	}
	tt.Frame()
	if s.ActionError != "" || len(s.ActionNotices) != 0 || len(s.Notices) != 0 {
		t.Errorf("dismiss left state: error=%q notices=%v projection=%v",
			s.ActionError, s.ActionNotices, s.Notices)
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("the notice strip is still drawn after dismiss")
	}
}

// TestNoticeStripKeepsTheTerminalCloseAboveTheActionError checks the existing
// precedence: a closed session outranks the older action failure, which is not
// shown beside it, so the strip never claims two different terminal states.
func TestNoticeStripKeepsTheTerminalCloseAboveTheActionError(t *testing.T) {
	s := fixture()
	s.SessionClosed = true
	s.SessionClosedReason = "done"
	s.ActionError = "Rename session: refused"
	tt := ui.NewTester(s.View, 900, 700)

	if !tt.HasText("Session closed: done") {
		t.Errorf("the terminal close is not drawn, texts %q", tt.Texts())
	}
	if tt.HasText("Rename session: refused") {
		t.Errorf("the older action error outranks the terminal close, texts %q", tt.Texts())
	}
}
