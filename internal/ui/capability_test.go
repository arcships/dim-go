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
// this tree: approval.respond, session.stop and supervisor.Reconnect exist in
// the Go client or supervisor, so no card may claim they are missing;
// session.create and attachment.put do not exist, so no card may claim the
// shell could wire them.
func TestCapabilityCardsReportTheActualGap(t *testing.T) {
	cards := map[string]wipCapability{}
	for _, capability := range wipCapabilities(&State{}) {
		cards[capability.action] = capability
	}

	for _, action := range []string{"Approvals", "Connection recovery", "Turn cancellation"} {
		capability, ok := cards[action]
		if !ok {
			t.Errorf("no card for %q", action)
			continue
		}
		if !strings.Contains(capability.gap, "helper exists") {
			t.Errorf("%q reports %q, but its helper exists", action, capability.gap)
		}
	}
	if capability := cards["Session creation"]; !strings.Contains(capability.gap, "helper exists") {
		t.Errorf("session creation reports %q, want helper exists", capability.gap)
	}
	if capability := cards["Attachments"]; !strings.Contains(capability.gap, "no Go client helper") {
		t.Errorf("attachments reports %q, want missing helper", capability.gap)
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
	if !strings.Contains(reasonApproval, "shell") {
		t.Errorf("approval %q reason must name the shell's wiring gap", reasonApproval)
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
