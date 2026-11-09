package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestRunStatusRowLeavesTheToolbar checks that the toolbar no longer repeats
// the session lifecycle: the reference's conversation bar has a breadcrumb and
// a menu, not a "Session: Idle" row. The Stop entry stays conditional on a
// running turn.
func TestRunStatusRowLeavesTheToolbar(t *testing.T) {
	for _, run := range []RunState{RunIdle, RunRunning, RunStopping} {
		s := fixture()
		s.Run = run
		tt := ui.NewTester(s.View, 900, 600)
		for _, gone := range []string{"Session: Idle", "Session: Running", "Session: Stopping…"} {
			if tt.HasText(gone) {
				t.Errorf("%v: the toolbar still draws the run row %q, texts %q", run, gone, tt.Texts())
			}
		}
	}
}

// TestStopEntryOnlyWhileRunning checks that the stop entry is there while a turn
// runs and gone once nothing runs or the stop is settling. It is a real action:
// the caller's callback sends session.stop, so the entry is enabled and carries
// no WIP reason.
func TestStopEntryOnlyWhileRunning(t *testing.T) {
	s := fixture()
	s.Run = RunRunning
	s.OnStop = func() {}
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "Stop")
	if !ok {
		t.Fatalf("a running session offers no stop entry, texts %q", tt.Texts())
	}
	if info.Disabled {
		t.Error("the stop entry is disabled while a turn runs")
	}
	if !info.Focusable {
		t.Error("the stop entry is unreachable")
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

// TestStopCallsCallback checks that clicking Stop reaches the caller's callback
// once, and that the shell itself reports nothing: the run state and rows stay
// as they were until the App Server's state arrives through the supervisor.
func TestStopCallsCallback(t *testing.T) {
	s := fixture()
	s.Run = RunRunning
	s.Messages = append(s.Messages, Message{
		ID: "t1", Author: AuthorTool,
		Tool: Tool{Name: "bash", Params: "go test", Status: ToolRunning},
	})
	stops := 0
	s.OnStop = func() { stops++ }
	tt := ui.NewTester(s.View, 900, 600)

	if err := tt.Click("Stop"); err != nil {
		t.Fatalf("click Stop: %v", err)
	}

	if stops != 1 {
		t.Errorf("OnStop ran %d times, want 1", stops)
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

// TestStopWithoutCallbackDoesNotPretendSuccess checks that a fixture shell with
// no callback keeps the Stop entry visible and unavailable, explaining the
// session.stop gap, and leaves its state alone: a click cannot report a stop the
// App Server did not perform.
func TestStopWithoutCallbackDoesNotPretendSuccess(t *testing.T) {
	s := fixture()
	s.Run = RunRunning
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleButton, "Stop")
	if !ok {
		t.Fatalf("a running session offers no stop entry, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("the stop entry is available without a callback")
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

	if err := tt.Click("Stop"); err != nil {
		t.Fatalf("click Stop: %v", err)
	}
	if s.Run != RunRunning {
		t.Errorf("run is %v after clicking Stop, want it untouched", s.Run)
	}
	if tt.HasText("Stopped") || tt.HasText("Stopping…") {
		t.Errorf("the view claims the session stopped, texts %q", tt.Texts())
	}
}
