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
		if _, ok := tt.AccessByRole(ui.RoleText, capability.action); !ok {
			t.Errorf("%q is not exposed as read-only text", capability.action)
		}
		// The sidebar's own "Session keys" entry shares the card's name; it is
		// a button, so the name alone no longer proves the card is not one. Its
		// method and gap stay text-only.
		if capability.action == "Session keys" {
			continue
		}
		if _, ok := tt.AccessByRole(ui.RoleButton, capability.action); ok {
			t.Errorf("%q is an action, not a read-only card", capability.action)
		}
		if _, ok := tt.AccessByRole(ui.RoleButton, capability.method); ok {
			t.Errorf("%q is an action, not a read-only card", capability.method)
		}
	}
	if !tt.HasText(wipBadge) {
		t.Errorf("the cards show no visible %q badge, texts %q", wipBadge, tt.Texts())
	}
}

// TestCapabilityCardsReportTheActualGap pins each card's gap to the fact in
// this tree: approval.respond, session.stop, session.create, session.branches,
// session.keys, attachment.put and supervisor.Reconnect have helpers in the Go
// client, session or supervisor, so no card may claim they are missing; an
// attachment card names the missing file picker instead. Branches and keys are
// still read-only, and approval decisions stay unavailable only while the
// window has no callback.
func TestCapabilityCardsReportTheActualGap(t *testing.T) {
	cards := map[string]wipCapability{}
	for _, capability := range wipCapabilities(&State{}) {
		cards[capability.action] = capability
	}

	for _, action := range []string{
		"Session creation", "Branch switching", "Branch end", "Branch refresh",
		"Session keys", "Approvals", "Attachments", "Connection recovery",
		"Turn cancellation",
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
	// attachment.put has a Go helper, so its card must name what the shell
	// still lacks: a file picker, and a way to be honest about an upload whose
	// result is unknown.
	if gap := cards["Attachments"].gap; !strings.Contains(gap, "file picker") || !strings.Contains(gap, "unknown") {
		t.Errorf("attachments reports %q, want the missing file picker and unknown results", gap)
	}
	// The branch and key reads are wired: their cards must name the real gap
	// (no branch-selection callback, no key browser callback), not a missing
	// helper.
	if gap := cards["Branch switching"].gap; !strings.Contains(gap, "branch selection callback") {
		t.Errorf("branch switching reports %q, want the missing selection callback", gap)
	}
	if gap := cards["Session keys"].gap; !strings.Contains(gap, "key browser callback") {
		t.Errorf("session keys reports %q, want the missing key browser callback", gap)
	}
	if gap := cards["Approvals"].gap; !strings.Contains(gap, "callback") {
		t.Errorf("approvals reports %q, want the missing callback", gap)
	}
	// session.create has a helper, so its card may only name the shell's missing
	// callback, never a missing client method.
	if gap := cards["Session creation"].gap; !strings.Contains(gap, "callback") {
		t.Errorf("session creation reports %q, want the missing create callback", gap)
	}
	if gap := cards["Branch end"].gap; !strings.Contains(gap, "branch end callback") {
		t.Errorf("branch end reports %q, want the missing end callback", gap)
	}
	if gap := cards["Branch refresh"].gap; !strings.Contains(gap, "branch refresh callback") {
		t.Errorf("branch refresh reports %q, want the missing refresh callback", gap)
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

// TestCapabilityCardsDropWiredBranchEndAndRefresh checks that the branch end and
// refresh callbacks leave the unavailable list once the window wires them: the
// branch directory's own entries are the real actions then, and a stale WIP card
// would contradict them.
func TestCapabilityCardsDropWiredBranchEndAndRefresh(t *testing.T) {
	s := fixture()
	s.Branches = []Branch{{ID: "main", Label: "main", State: "open"}, {ID: "side", Label: "side", State: "open"}}
	s.SelectedBranch = "side"
	s.OnEndBranch = func(string) {}
	s.OnRefreshBranches = func() {}
	tt := ui.NewTester(s.View, 900, 600)

	for _, capability := range wipCapabilities(s) {
		if capability.action == "Branch end" || capability.action == "Branch refresh" {
			t.Errorf("a wired branch action still has a WIP card: %+v", capability)
		}
	}
	if tt.HasText("Branch end") || tt.HasText("Branch refresh") {
		t.Errorf("the unavailable list still shows a wired branch action, texts %q", tt.Texts())
	}
	if info, ok := tt.AccessByRole(ui.RoleButton, "End branch"); !ok || info.Disabled {
		t.Errorf("wired branch end is not an enabled action: %+v, found %v", info, ok)
	}
	if info, ok := tt.AccessByRole(ui.RoleButton, "Refresh"); !ok || info.Disabled {
		t.Errorf("wired branch refresh is not an enabled action: %+v, found %v", info, ok)
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

// TestCapabilityCardsDropWiredKeys checks that the session keys browser leaves
// the unavailable list once the window has a session.keys snapshot: the
// sidebar's own entry is the real action then, and a stale WIP card would
// contradict it.
func TestCapabilityCardsDropWiredKeys(t *testing.T) {
	s := fixture()
	s.Keys = KeysPanelView{Loaded: true, Through: 12, Entries: keysPanelEntries()}
	tt := ui.NewTester(s.View, 900, 600)

	for _, capability := range wipCapabilities(s) {
		if capability.action == "Session keys" {
			t.Errorf("a wired keys browser still has a WIP card: %+v", capability)
		}
	}
	if tt.HasText("session.keys") {
		t.Errorf("the unavailable list still shows session.keys, texts %q", tt.Texts())
	}
	info, ok := tt.AccessByRole(ui.RoleButton, "Session keys")
	if !ok || info.Disabled {
		t.Errorf("wired keys entry is not an enabled action: %+v, found %v", info, ok)
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
// under the thread list and inside the window, not merely present in the label
// list: the list above them keeps its rows, the cards follow one another, and
// the cards' own scroll keeps the last one reachable when the window is too
// short to show every card at once.
func TestCapabilityCardsFitTheSidebar(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	cards := wipCapabilities(s)
	// cardText is the text a card is found by. The "Session keys" card shares
	// its action name with the sidebar's own entry, so it is found by its
	// method: the check must measure the card, not the entry above it.
	cardText := func(capability wipCapability) string {
		if capability.action == "Session keys" {
			return capability.method
		}
		return capability.action
	}
	first, ok := tt.Find(cardText(cards[0]))
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

	previous, lastLabel := first, cardText(cards[0])
	for _, capability := range cards[1:] {
		card, ok := tt.Find(cardText(capability))
		if !ok {
			t.Fatalf("card %q is not drawn, texts %q", capability.action, tt.Texts())
		}
		if card.Y < previous.Y+previous.H {
			t.Errorf("card %q at %+v overlaps the card before it at %+v", capability.action, card, previous)
		}
		previous, lastLabel = card, cardText(capability)
	}
	// The card area is a scroll container bounded by the sidebar, so a short
	// window clips the last card instead of pushing it past the window.
	if previous.Y+previous.H > 600 {
		tt.Scroll(first.X+4, first.Y+4, 0, 2000)
		tt.Frame()
		last, ok := tt.Find(lastLabel)
		if !ok {
			t.Fatalf("last card %q is not drawn after scrolling", lastLabel)
		}
		if last.Y+last.H > 600 {
			t.Errorf("the last card still ends at %.2f, below the 600-high window", last.Y+last.H)
		}
	}
}

// TestCapabilityCardsDrawThePopoverFace checks that a capability card is
// painted with the popover face rather than the border tint it used to be: the
// padding beside its action text is the opaque face, so the card's parts read
// against a surface of their own instead of whatever happens to be under it.
func TestCapabilityCardsDrawThePopoverFace(t *testing.T) {
	s := fixture()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 600)

	capability := wipCapabilities(s)[0]
	rect, ok := tt.Find(capability.action)
	if !ok {
		t.Fatalf("card %q is not drawn, texts %q", capability.action, tt.Texts())
	}
	face := newTokens(theme).popover
	pixel := tt.Image().RGBAAt(int(rect.X)-4, int(rect.Y+rect.H/2))
	if pixel.R != face.R || pixel.G != face.G || pixel.B != face.B || pixel.A != face.A {
		t.Errorf("the padding beside %q is %v, want the popover face %+v", capability.action, pixel, face)
	}
}

// TestCapabilityCardsDoNotChangeState checks that the read-only cards add no
// focus stop and leave the shell's view state alone: only the sidebar's own
// entries and the composer take the keyboard.
func TestCapabilityCardsDoNotChangeState(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 600)

	before := s.Selected
	for _, want := range []string{
		"Update", "New session", "Search sessions", "Session keys", "Group", "Sessions",
		"Pin", "Rename", "Archive", "Divider", "Reconnect", "Settings", "Attach file", "Composer",
	} {
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
