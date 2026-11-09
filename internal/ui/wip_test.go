package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// wipCase names one entry the shell cannot carry out yet and the upstream
// method or capability its reason must name.
type wipCase struct {
	label  string
	reason string
}

// TestWIPEntriesAreVisibleDisabledAndExplainThemselves checks the shape of
// every WIP entry: it is drawn, its name is the action, assistive technology
// reads it as unavailable but still reaches it, and its description says what
// upstream work it waits on. The badge beside it makes the gap visible to
// sighted users too.
func TestWIPEntriesAreVisibleDisabledAndExplainThemselves(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	cases := []wipCase{
		{"New session", "session.create"},
		{"Reconnect", "reconnect"},
		{"Attach file", "attachment.put"},
	}
	for _, tc := range cases {
		info, ok := tt.AccessByRole(ui.RoleButton, tc.label)
		if !ok {
			t.Errorf("%q is not exposed as a button, texts %q", tc.label, tt.Texts())
			continue
		}
		if !info.Disabled {
			t.Errorf("%q is not marked unavailable", tc.label)
		}
		if !info.Focusable {
			t.Errorf("%q is unreachable, so its reason is never read", tc.label)
		}
		if !strings.Contains(info.Description, tc.reason) {
			t.Errorf("%q description %q does not name %q", tc.label, info.Description, tc.reason)
		}
		if !strings.Contains(info.Description, wipBadge) {
			t.Errorf("%q description %q does not include the visible %q badge", tc.label, info.Description, wipBadge)
		}
		if !tt.HasText(tc.label) {
			t.Errorf("%q is not drawn, texts %q", tc.label, tt.Texts())
		}
	}
	if !tt.HasText(wipBadge) {
		t.Errorf("the WIP entries show no visible WIP badge, texts %q", tt.Texts())
	}
}

// TestApprovalWIPEntries checks that an approval row keeps the two decisions,
// visible and unavailable, with the reason approval.respond is missing. The
// row itself is view state; the request and its decision stay the App Server's.
func TestApprovalWIPEntries(t *testing.T) {
	s := fixture()
	s.Messages = append(s.Messages, Message{
		ID:      "m3",
		Author:  AuthorApproval,
		Content: "Run the deployment?",
	})
	tt := ui.NewTester(s.View, 900, 600)

	if !tt.HasText("Approval request") || !tt.HasText("Run the deployment?") {
		t.Fatalf("approval row is not drawn, texts %q", tt.Texts())
	}
	for _, label := range []string{"Approve", "Deny"} {
		info, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok {
			t.Errorf("%q is not exposed as a button, texts %q", label, tt.Texts())
			continue
		}
		if !info.Disabled {
			t.Errorf("%q is not marked unavailable", label)
		}
		if !info.Focusable {
			t.Errorf("%q is unreachable, so its reason is never read", label)
		}
		if !strings.Contains(info.Description, "approval.respond") {
			t.Errorf("%q description %q does not name approval.respond", label, info.Description)
		}
		if !strings.Contains(info.Description, wipBadge) {
			t.Errorf("%q description %q does not include the visible %q badge", label, info.Description, wipBadge)
		}
	}
}

// TestWIPEntriesDoNotPretendSuccess clicks every WIP entry and checks that the
// view state is untouched: no session was created, no reconnection started, no
// attachment added and no approval answered. The shell never reads Clicked for
// these entries, so it has no way to report an action the App Server did not
// perform.
func TestWIPEntriesDoNotPretendSuccess(t *testing.T) {
	s := fixture()
	s.Messages = append(s.Messages, Message{ID: "m3", Author: AuthorApproval, Content: "Run it?"})
	tt := ui.NewTester(s.View, 900, 600)

	for _, label := range []string{"New session", "Reconnect", "Attach file", "Approve", "Deny"} {
		if err := tt.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
	}

	if s.Selected != "s1" {
		t.Errorf("selected %q after clicking WIP entries, want s1", s.Selected)
	}
	if s.Draft != "" {
		t.Errorf("draft %q after clicking WIP entries, want empty", s.Draft)
	}
	if len(s.Messages) != 3 {
		t.Errorf("%d rows after clicking WIP entries, want 3", len(s.Messages))
	}
	if s.Connection != ConnUnknown || s.ConnectionError != "" {
		t.Errorf("connection %v %q after clicking WIP entries, want it untouched",
			s.Connection, s.ConnectionError)
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("clicking WIP entries raised an alert, as if an action had failed or succeeded")
	}
}

// TestReconnectEntryHidesWhenConnected checks that a healthy connection shows
// no reconnect entry: there is nothing to reconnect to, and a permanently
// unavailable action beside "Connected" would be noise.
func TestReconnectEntryHidesWhenConnected(t *testing.T) {
	s := fixture()
	s.Connection = ConnReady
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleButton, "Reconnect"); ok {
		t.Error("the reconnect entry shows while connected")
	}
	// Find matches the whole text: the read-only capability card's
	// "supervisor.Reconnect" is not the entry.
	if _, ok := tt.Find("Reconnect"); ok {
		t.Errorf("the reconnect entry is drawn while connected, texts %q", tt.Texts())
	}
}

// TestReconnectEntryHidesWhileLoading checks that a reconnect in flight cannot
// be started twice: the loading state owns the one attempt until it settles.
func TestReconnectEntryHidesWhileLoading(t *testing.T) {
	s := fixture()
	s.Connection = ConnError
	s.ConversationLoading = true
	s.OnReconnect = func() {}
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleButton, "Reconnect"); ok {
		t.Error("the reconnect entry shows while a reconnect is loading")
	}

	s.ConversationLoading = false
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleButton, "Reconnect"); !ok {
		t.Errorf("the reconnect entry is missing after loading, texts %q", tt.Texts())
	}
}

// TestReconnectEntryHidesWithoutSelection checks that no reconnect action is
// offered when there is no selected session to re-open, even if the connection
// is not ready.
func TestReconnectEntryHidesWithoutSelection(t *testing.T) {
	s := fixture()
	s.Selected = ""
	s.Connection = ConnError
	s.OnReconnect = func() {}
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleButton, "Reconnect"); ok {
		t.Error("the reconnect entry shows without a selected session")
	}
}
