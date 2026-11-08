package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestRunStatusLabels checks the toolbar's run row for each state, and that it
// is a live region so a turn starting or settling is read out once.
func TestRunStatusLabels(t *testing.T) {
	cases := []struct {
		run   RunState
		label string
	}{
		{RunIdle, "Idle"},
		{RunRunning, "Running"},
		{RunStopping, "Stopping…"},
	}
	for _, tc := range cases {
		s := fixture()
		s.Run = tc.run
		tt := ui.NewTester(s.View, 900, 600)
		want := "Session: " + tc.label
		if !tt.HasText(want) {
			t.Errorf("%v: run row does not show %q, texts %q", tc.run, want, tt.Texts())
			continue
		}
		if _, ok := tt.AccessByRole(ui.RoleStatus, want); !ok {
			t.Errorf("%v: run row %q is not a status region", tc.run, want)
		}
	}
}

// TestStopEntryOnlyWhileRunning checks that the stop entry is there while a turn
// runs, visible and unavailable with the session.stop reason, and gone once
// nothing runs or the stop is settling.
func TestStopEntryOnlyWhileRunning(t *testing.T) {
	s := fixture()
	s.Run = RunRunning
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "Stop")
	if !ok {
		t.Fatalf("a running session offers no stop entry, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("the stop entry is not marked unavailable")
	}
	if !info.Focusable {
		t.Error("the stop entry is unreachable, so its reason is never read")
	}
	if !strings.Contains(info.Description, "session.stop") {
		t.Errorf("the stop entry's description %q does not name session.stop", info.Description)
	}
	if !strings.Contains(info.Description, wipBadge) {
		t.Errorf("the stop entry's description %q does not include the %q badge", info.Description, wipBadge)
	}
	if !tt.HasText("Stop") {
		t.Errorf("the stop entry is not drawn, texts %q", tt.Texts())
	}

	for _, run := range []RunState{RunIdle, RunStopping} {
		idle := fixture()
		idle.Run = run
		other := ui.NewTester(idle.View, 900, 600)
		if _, ok := other.AccessByRole(ui.RoleButton, "Stop"); ok {
			t.Errorf("%v offers a stop entry", run)
		}
		// Find matches the whole text, so the "Stopping…" status does not
		// count as the entry.
		if _, ok := other.Find("Stop"); ok {
			t.Errorf("%v draws a stop entry, texts %q", run, other.Texts())
		}
	}
}

// TestStopDoesNotPretendSuccess clicks the stop entry and checks that no view
// state changed: the run still runs, the rows are untouched and nothing was
// announced as stopped. The shell never reads Clicked for the entry, so it
// cannot report a stop the App Server did not perform.
func TestStopDoesNotPretendSuccess(t *testing.T) {
	s := fixture()
	s.Run = RunRunning
	s.Messages = append(s.Messages, Message{
		ID: "t1", Author: AuthorTool,
		Tool: Tool{Name: "bash", Params: "go test", Status: ToolRunning},
	})
	tt := ui.NewTester(s.View, 900, 600)

	if err := tt.Click("Stop"); err != nil {
		t.Fatalf("click Stop: %v", err)
	}

	if s.Run != RunRunning {
		t.Errorf("run is %v after clicking Stop, want it untouched", s.Run)
	}
	if len(s.Messages) != 3 {
		t.Errorf("%d rows after clicking Stop, want 3", len(s.Messages))
	}
	if s.Messages[2].Tool.Status != ToolRunning {
		t.Errorf("the tool row's status is %v after clicking Stop, want it untouched",
			s.Messages[2].Tool.Status)
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("clicking Stop raised an alert, as if the stop had failed or succeeded")
	}
	if tt.HasText("Stopped") || tt.HasText("Stopping…") {
		t.Errorf("the view claims the session stopped, texts %q", tt.Texts())
	}
}
