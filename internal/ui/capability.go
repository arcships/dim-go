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

// wipCapabilities is the list the cards draw. It mirrors the entries that stay
// unavailable; keep each card's gap in step with the reason the entry passes to
// wipAction. "No Go client helper yet" means the method really is absent from
// internal/appserver or internal/supervisor; "helper exists; ..." means the
// capability exists there and names what this shell still lacks; "no verified
// ... RPC" means the upstream method itself is unconfirmed, so the shell must
// not invent one.
//
// The list is per shell state: session creation is wired once
// State.OnCreateSession is set, reconnect once State.OnReconnect is set, turn
// cancellation once State.OnStop is set and approval decisions once
// State.OnApproval is set, so each is unavailable only while its callback is
// missing. session.branches and session.keys have read helpers; switching a
// branch reads that branch's history and is wired once State.OnSelectBranch is
// set, ending one is wired once State.OnEndBranch is set, refreshing the
// directory once State.OnRefreshBranches is set, while the keys browser is
// wired once State.Keys carries a session.keys snapshot (the shell then wires
// Actions.OnToggleKeys and the sidebar entry replaces this card).
func wipCapabilities(s *State) []wipCapability {
	capabilities := make([]wipCapability, 0, 9)
	if s == nil || s.OnCreateSession == nil {
		capabilities = append(capabilities,
			wipCapability{"Session creation", "session.create", "helper exists; shell has no create callback"})
	}
	if s == nil || s.OnSelectBranch == nil {
		capabilities = append(capabilities,
			wipCapability{"Branch switching", "session.branches", "helper exists; shell has no branch selection callback"})
	}
	if s == nil || s.OnEndBranch == nil {
		capabilities = append(capabilities,
			wipCapability{"Branch end", "session.branch_end", "helper exists; shell has no branch end callback"})
	}
	if s == nil || s.OnRefreshBranches == nil {
		capabilities = append(capabilities,
			wipCapability{"Branch refresh", "session.branches", "helper exists; shell has no branch refresh callback"})
	}
	if s == nil || !s.Keys.Loaded {
		capabilities = append(capabilities,
			wipCapability{"Session keys", "session.keys", "helper exists; shell has no key browser callback"})
	}
	if s == nil || s.OnApproval == nil {
		capabilities = append(capabilities,
			wipCapability{"Approvals", "approval.respond", "helper exists; window has no approval callback"})
	}
	capabilities = append(capabilities,
		wipCapability{"Attachments", "attachment.put", "helper exists; shell has no file picker and keeps unknown upload results disabled"})
	if s == nil || s.OnReconnect == nil {
		capabilities = append(capabilities,
			wipCapability{"Connection recovery", "supervisor.Reconnect", "helper exists; shell not wired"})
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
// session sidebar, which is its only consumer. It takes the room the thread
// list leaves and scrolls when the window is too short for every card, so the
// catalog above it keeps its rows.
func capabilityCards(c *ui.Context, k tokens, capabilities []wipCapability) {
	ui.Scroll(c).Shrink(1).Children(func() {
		ui.Column(c).Padding(k.sidebarPadX).Gap(k.sidebarGap).
			BorderWidth(k.hairline, 0, 0, 0).BorderColor(k.border).
			Role(ui.RoleGroup).Label("Unavailable capabilities").Children(func() {
			// The group's label names the list, so the visible heading stays out of
			// the accessibility tree rather than being read twice.
			ui.Text(c, "Unavailable capabilities").FontSize(k.smallSize).TextColor(k.mutedForeground).
				Unselectable().AccessHidden(true)
			for _, capability := range capabilities {
				capabilityCard(c, k, capability)
			}
		})
	})
}

// capabilityCard draws one capability's status: its action, the method behind
// it, and the gap. Every part is text; the card has no button, link or field,
// so a user can read the status but cannot act on it. It is drawn as a popover
// rather than a card because it floats over the session list instead of
// belonging to the conversation.
func capabilityCard(c *ui.Context, k tokens, capability wipCapability) {
	ui.Column(c).Padding(k.cardPadY, k.cardPadX).Gap(k.gap).Radius(k.cardRadius).
		Background(k.popover).BorderWidth(k.hairline).BorderColor(k.border).Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, capability.action).Bold().FontSize(k.smallSize).TextColor(k.popoverForeground)
			ui.Box(c).Grow(1)
			ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
		})
		ui.Text(c, capability.method).FontSize(k.smallSize).TextColor(k.mutedForeground).NoWrap()
		ui.Text(c, capability.gap).FontSize(k.smallSize).TextColor(k.mutedForeground)
	})
}
