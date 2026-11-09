package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestCapabilityCardsAreReadOnly checks the sidebar's unavailable-capabilities
// list: every card is drawn with its action, method and gap, and none of them
// is a control. A reader can learn what is missing, but there is nothing to
// click that could imply the shell has the capability.
func TestCapabilityCardsAreReadOnly(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Unavailable capabilities"); !ok {
		t.Fatalf("the cards are not grouped for assistive technology, texts %q", tt.Texts())
	}
	for _, capability := range wipCapabilities(s) {
		for _, want := range []string{capability.action, capability.method, capability.gap} {
			if !tt.HasText(want) {
				t.Errorf("%q: card is missing %q, texts %q", capability.action, want, tt.Texts())
			}
		}
		if _, ok := tt.AccessByRole(ui.RoleButton, capability.action); ok {
			t.Errorf("%q is an action, not a read-only card", capability.action)
		}
		if _, ok := tt.AccessByRole(ui.RoleButton, capability.method); ok {
			t.Errorf("%q is an action, not a read-only card", capability.method)
		}
		if _, ok := tt.AccessByRole(ui.RoleText, capability.action); !ok {
			t.Errorf("%q is not exposed as read-only text", capability.action)
		}
	}
	if !tt.HasText(wipBadge) {
		t.Errorf("the cards show no visible %q badge, texts %q", wipBadge, tt.Texts())
	}
}

// TestCapabilityCardsReportTheActualGap pins each card's gap to the fact in
// this tree: approval.respond, session.stop, session.create, session.branches,
// session.keys and supervisor.Reconnect have helpers in the Go client, session
// or supervisor, so no card may claim they are missing; only attachment.put has
// no Go helper. Branches and keys are still read-only, and approval decisions
// stay unavailable only while the window has no callback.
func TestCapabilityCardsReportTheActualGap(t *testing.T) {
	cards := map[string]wipCapability{}
	for _, capability := range wipCapabilities(&State{}) {
		cards[capability.action] = capability
	}

	for _, action := range []string{
		"Session creation", "Branch switching", "Session keys", "Approvals",
		"Connection recovery", "Turn cancellation",
	} {
		capability, ok := cards[action]
		if !ok {
			t.Errorf("no card for %q", action)
			continue
		}
		if !strings.Contains(capability.gap, "helper exists") {
			t.Errorf("%q reports %q, but its helper exists", action, capability.gap)
		}
	}
	if capability := cards["Attachments"]; !strings.Contains(capability.gap, "no Go client helper") {
		t.Errorf("attachments reports %q, want missing helper", capability.gap)
	}
	// The branch and key reads are wired: their cards must name the real gap
	// (read-only display, no switch RPC or key browser), not a missing helper.
	if gap := cards["Branch switching"].gap; !strings.Contains(gap, "read-only") {
		t.Errorf("branch switching reports %q, want the read-only gap", gap)
	}
	if gap := cards["Session keys"].gap; !strings.Contains(gap, "key browser") {
		t.Errorf("session keys reports %q, want the missing key browser", gap)
	}
	if gap := cards["Approvals"].gap; !strings.Contains(gap, "callback") {
		t.Errorf("approvals reports %q, want the missing callback", gap)
	}
	// session.create has a helper, so its card may only name the shell's missing
	// callback, never a missing client method.
	if gap := cards["Session creation"].gap; !strings.Contains(gap, "callback") {
		t.Errorf("session creation reports %q, want the missing create callback", gap)
	}
	if !strings.Contains(reasonCreateSession, "session.create") {
		t.Errorf("create reason does not name session.create: %q", reasonCreateSession)
	}

	// Reconnect used to say the App Server had not implemented it. The
	// supervisor now has Reconnect, so the reason must name the shell's wiring
	// gap instead of claiming an upstream gap that no longer exists.
	if strings.Contains(reasonReconnect, "not implemented") {
		t.Errorf("reconnect reason claims the capability is missing: %q", reasonReconnect)
	}
	if !strings.Contains(reasonReconnect, "supervisor") {
		t.Errorf("reconnect reason does not name the supervisor: %q", reasonReconnect)
	}
	if !strings.Contains(reasonApproval, "shell") || !strings.Contains(reasonApproval, "approval.respond") {
		t.Errorf("approval %q reason must name the shell's missing approval.respond callback", reasonApproval)
	}
	if !strings.Contains(reasonBranches, "switching") {
		t.Errorf("branch reason does not name the missing switch: %q", reasonBranches)
	}
}

// TestCapabilityCardsDropWiredReconnect checks that connection recovery leaves
// the unavailable list once State.OnReconnect is set: the Reconnect entry is the
// real action then, and a stale WIP card would contradict the toolbar.
func TestCapabilityCardsDropWiredReconnect(t *testing.T) {
	s := fixture()
	s.Connection = ConnError
	s.OnReconnect = func() {}
	tt := ui.NewTester(s.View, 900, 600)

	for _, capability := range wipCapabilities(s) {
		if capability.action == "Connection recovery" {
			t.Errorf("a wired reconnect still has a WIP card: %+v", capability)
		}
	}
	if tt.HasText("Connection recovery") {
		t.Errorf("the unavailable list still shows connection recovery, texts %q", tt.Texts())
	}
	if info, ok := tt.AccessByRole(ui.RoleButton, "Reconnect"); !ok || info.Disabled {
		t.Errorf("wired reconnect is not an enabled action: %+v, found %v", info, ok)
	}
}

// TestCapabilityCardsDropWiredStop checks that turn cancellation leaves the
// unavailable list once State.OnStop is set: the Stop entry is the real action
// then, and keeping a WIP card beside it would contradict the toolbar.
func TestCapabilityCardsDropWiredStop(t *testing.T) {
	s := fixture()
	s.Run = RunRunning
	s.OnStop = func() {}
	tt := ui.NewTester(s.View, 900, 600)

	for _, capability := range wipCapabilities(s) {
		if capability.action == "Turn cancellation" {
			t.Errorf("a wired stop still has a WIP card: %+v", capability)
		}
	}
	if tt.HasText("Turn cancellation") {
		t.Errorf("the unavailable list still shows turn cancellation, texts %q", tt.Texts())
	}
	if info, ok := tt.AccessByRole(ui.RoleButton, "Stop"); !ok || info.Disabled {
		t.Errorf("wired stop is not an enabled action: %+v, found %v", info, ok)
	}
}

// TestCapabilityCardsDropWiredApprovals checks that approval decisions leave the
// unavailable list once State.OnApproval is set: the row's own entries are the
// real actions then, and a stale WIP card would contradict them.
func TestCapabilityCardsDropWiredApprovals(t *testing.T) {
	s := approvalFixture()
	s.OnApproval = func(string, string, ApprovalDecision) {}
	tt := ui.NewTester(s.View, 900, 600)

	for _, capability := range wipCapabilities(s) {
		if capability.action == "Approvals" {
			t.Errorf("a wired approval still has a WIP card: %+v", capability)
		}
	}
	if tt.HasText("Approvals") {
		t.Errorf("the unavailable list still shows approvals, texts %q", tt.Texts())
	}
	if info, ok := tt.AccessByRole(ui.RoleButton, "Approve"); !ok || info.Disabled {
		t.Errorf("wired approval is not an enabled action: %+v, found %v", info, ok)
	}
}

// TestCapabilityCardsDropWiredCreateSession checks that session creation leaves
// the unavailable list once State.OnCreateSession is set: the sidebar's entry is
// the real action then, and a stale WIP card would contradict it.
func TestCapabilityCardsDropWiredCreateSession(t *testing.T) {
	s := fixture()
	s.OnCreateSession = func() {}
	tt := ui.NewTester(s.View, 900, 600)

	for _, capability := range wipCapabilities(s) {
		if capability.action == "Session creation" {
			t.Errorf("a wired creation still has a WIP card: %+v", capability)
		}
	}
	if tt.HasText("Session creation") {
		t.Errorf("the unavailable list still shows session creation, texts %q", tt.Texts())
	}
	if info, ok := tt.AccessByRole(ui.RoleButton, "New session"); !ok || info.Disabled {
		t.Errorf("wired session creation is not an enabled action: %+v, found %v", info, ok)
	}
}

// TestCapabilityCardsFitTheSidebar checks that the cards are actually placed
// under the session list and inside the window, not merely present in the
// label list: the list above them keeps its rows and the last card's bottom
// stays on screen.
func TestCapabilityCardsFitTheSidebar(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	cards := wipCapabilities(s)
	first, ok := tt.Find(cards[0].action)
	if !ok {
		t.Fatalf("first card %q is not drawn, texts %q", cards[0].action, tt.Texts())
	}
	session, ok := tt.Find("Second session")
	if !ok {
		t.Fatalf("second session row is not drawn above the cards, texts %q", tt.Texts())
	}
	if session.Y+session.H > first.Y {
		t.Errorf("session row %+v overlaps the first card %+v", session, first)
	}

	previous := first
	for _, capability := range cards[1:] {
		card, ok := tt.Find(capability.action)
		if !ok {
			t.Fatalf("card %q is not drawn, texts %q", capability.action, tt.Texts())
		}
		if card.Y < previous.Y+previous.H {
			t.Errorf("card %q at %+v overlaps the card before it at %+v", capability.action, card, previous)
		}
		previous = card
	}
	if previous.Y+previous.H > 600 {
		t.Errorf("the last card ends at %.2f, below the 600-high window", previous.Y+previous.H)
	}
}

// TestCapabilityCardsDoNotChangeState checks that the read-only cards add no
// focus stop and leave the shell's view state alone: only the existing WIP
// actions and the composer take the keyboard.
func TestCapabilityCardsDoNotChangeState(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	before := s.Selected
	for _, want := range []string{"New session", "Sessions", "Reconnect", "Attach file", "Composer"} {
		tt.Key(0, ui.KeyTab)
		if !tt.Focused(want) {
			t.Fatalf("Tab does not reach %q, texts %q", want, tt.Texts())
		}
	}
	if s.Selected != before {
		t.Errorf("selected %q while tabbing through the shell, want %q", s.Selected, before)
	}
	if s.Draft != "" {
		t.Errorf("draft %q while tabbing through the shell, want empty", s.Draft)
	}
}
