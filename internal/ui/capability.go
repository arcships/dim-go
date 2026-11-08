package ui

import "github.com/egoist/mygo/ui"

// wipCapability is one read-only card in the sidebar's unavailable-capabilities
// list: the action it would back, the App Server or supervisor entry behind it,
// and which side of the gap is missing.
//
// It is product status, not session state. Its text is fixed and holds no
// session, run or approval fact; only whether a capability is present varies
// with the shell's callback wiring.
type wipCapability struct {
	action string
	method string
	gap    string
}

// wipCapabilities is the list the cards draw. It mirrors the disabled entries;
// keep each card's gap in step with the reason the entry passes to wipAction.
// "No Go client helper yet" means the method really is absent from
// internal/appserver or internal/supervisor; "helper exists; shell not wired"
// means the capability exists there and only this view has not called it.
//
// The list is per shell state: turn cancellation is wired once State.OnStop is
// set, so it is unavailable only while that callback is missing.
func wipCapabilities(s *State) []wipCapability {
	capabilities := []wipCapability{
		{"Session creation", "session.create", "no Go client helper yet"},
		{"Approvals", "approval.respond", "helper exists; shell not wired"},
		{"Attachments", "attachment.put", "no Go client helper yet"},
		{"Connection recovery", "supervisor.Reconnect", "helper exists; shell not wired"},
	}
	if s == nil || s.OnStop == nil {
		capabilities = append(capabilities,
			wipCapability{"Turn cancellation", "session.stop", "helper exists; shell not wired"})
	}
	return capabilities
}

// capabilityCards draws the read-only list of actions the shell cannot carry
// out yet. The cards say what is missing without offering a control, so no
// click can imply a capability the shell does not have. The list sits under the
// session sidebar, which is its only consumer.
func (s *State) capabilityCards(c *ui.Context, k tokens) {
	ui.Column(c).Padding(k.gap, k.sidebarPadX).Gap(k.gap).
		BorderWidth(k.hairline, 0, 0, 0).BorderColor(k.border).
		Role(ui.RoleGroup).Label("Unavailable capabilities").Children(func() {
		// The group's label names the list, so the visible heading stays out of
		// the accessibility tree rather than being read twice.
		ui.Text(c, "Unavailable capabilities").FontSize(k.smallSize).TextColor(k.muted).
			Unselectable().AccessHidden(true)
		for _, capability := range wipCapabilities(s) {
			capabilityCard(c, k, capability)
		}
	})
}

// capabilityCard draws one capability's status: its action, the method behind
// it, and the gap. Every part is text; the card has no button, link or field,
// so a user can read the status but cannot act on it.
func capabilityCard(c *ui.Context, k tokens, capability wipCapability) {
	ui.Column(c).Padding(k.cardPadY, k.cardPadX).Gap(k.gap).Radius(k.cardRadius).
		Background(k.wash(k.border)).BorderWidth(k.hairline).BorderColor(k.border).Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, capability.action).Bold().FontSize(k.smallSize)
			ui.Box(c).Grow(1)
			ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.muted).Unselectable()
		})
		ui.Text(c, capability.method).FontSize(k.smallSize).TextColor(k.muted).NoWrap()
		ui.Text(c, capability.gap).FontSize(k.smallSize).TextColor(k.muted)
	})
}
