package ui

import (
	"strings"
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
	if s.ActionError != "" || s.ActionStatus != "" || len(s.ActionNotices) != 0 || len(s.Notices) != 0 {
		t.Errorf("dismiss left state: error=%q status=%q notices=%v projection=%v",
			s.ActionError, s.ActionStatus, s.ActionNotices, s.Notices)
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("the notice strip is still drawn after dismiss")
	}
}

// TestNoticeStripShowsTheActionStatusAsMuted checks the window's confirmed
// outcome line: ActionStatus is drawn as its own muted status row and its
// dismiss entry clears it without touching an App Server fact.
func TestNoticeStripShowsTheActionStatusAsMuted(t *testing.T) {
	s := fixture()
	s.ActionStatus = "Submission accepted"
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 700)
	k := newTokens(theme)

	if !tt.HasText(s.ActionStatus) {
		t.Fatalf("the action status is missing, texts %q", tt.Texts())
	}
	status := textBox(t, tt, s.ActionStatus)
	if _, muted := colorBox(tt, status, k.mutedForeground); muted == 0 {
		t.Error("the action status is not drawn muted")
	}
	if _, danger := colorBox(tt, status, k.danger); danger != 0 {
		t.Error("the action status is drawn in the danger color")
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); !ok {
		t.Fatalf("the action status is not a live-region alert, texts %q", tt.Texts())
	}

	if err := tt.Click(tr(LocaleEN, keyNoticesDismiss)); err != nil {
		t.Fatalf("dismiss the action status: %v", err)
	}
	tt.Frame()
	if s.ActionStatus != "" {
		t.Errorf("dismiss left the action status %q", s.ActionStatus)
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("the notice strip is still drawn after dismissing the status")
	}
}

// TestNoticeStripKeepsTheUnknownStatusOutOfTheDangerRow checks the unknown-result
// pair: the failure detail is the danger row and the bare "Submission result
// unknown" status is the muted line under it, so the status never reads as a
// second error.
func TestNoticeStripKeepsTheUnknownStatusOutOfTheDangerRow(t *testing.T) {
	s := fixture()
	s.ActionError = "Submission result unknown: the App Server does not hold this submission."
	s.ActionStatus = "Submission result unknown"
	s.ActionNotices = []Notice{{Kind: "submit.unknown", ID: s.ActionError}}
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 700)
	k := newTokens(theme)

	failure := textBox(t, tt, s.ActionError)
	if _, danger := colorBox(tt, failure, k.danger); danger == 0 {
		t.Error("the unknown result's failure is not drawn in the danger color")
	}
	status := textBox(t, tt, s.ActionStatus)
	if _, muted := colorBox(tt, status, k.mutedForeground); muted == 0 {
		t.Error("the unknown result's status is not drawn muted")
	}
	if _, danger := colorBox(tt, status, k.danger); danger != 0 {
		t.Error("the unknown result's status is drawn in the danger color")
	}
	if strings.Count(strings.Join(tt.Texts(), "\n"), s.ActionError) != 1 {
		t.Errorf("the unknown result's failure is repeated, texts %q", tt.Texts())
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
