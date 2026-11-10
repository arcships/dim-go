package ui

import "github.com/egoist/mygo/ui"

// wipCapability is one read-only card in the settings surface's
// unavailable-capabilities list: the action it would back, the App Server or
// supervisor entry behind it, and which side of the gap is missing.
//
// It is product status, not session state. Its text is fixed and holds no
// session, run or approval fact; only whether a capability is present varies
// with the shell's callback wiring.
//
// action and gap are the English text the tests and the default frame pin;
// actionKey and gapKey are the message keys the active locale reads. method is
// the wire name behind the card and stays literal in every language.
type wipCapability struct {
	action    string
	method    string
	gap       string
	actionKey string
	gapKey    string
}

// wipCapabilityOf builds one card from its message keys. The English fields are
// read from the English table, so the card the tests inspect and the card an
// English frame draws cannot drift apart.
func wipCapabilityOf(actionKey, method, gapKey string) wipCapability {
	return wipCapability{
		action:    stringsEN.text(actionKey),
		method:    method,
		gap:       stringsEN.text(gapKey),
		actionKey: actionKey,
		gapKey:    gapKey,
	}
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
// directory once State.OnRefreshBranches is set, cutting a side branch from a
// message once State.OnBranchFrom is set, while the keys browser is wired once
// State.Keys carries a session.keys snapshot (the shell then wires
// Actions.OnToggleKeys and the sidebar entry replaces this card). The
// attachment entry is wired once State.OnAttachFile is set, so its card is only
// drawn for a window that has no picker callback.
func wipCapabilities(s *State) []wipCapability {
	capabilities := make([]wipCapability, 0, 10)
	if s == nil || s.OnCreateSession == nil {
		capabilities = append(capabilities,
			wipCapabilityOf(keyCapabilityCreate, "session.create", keyCapabilityCreateGap))
	}
	if s == nil || s.OnSelectBranch == nil {
		capabilities = append(capabilities,
			wipCapabilityOf(keyCapabilityBranchSwitch, "session.branches", keyCapabilityBranchSwitchGap))
	}
	if s == nil || s.OnEndBranch == nil {
		capabilities = append(capabilities,
			wipCapabilityOf(keyCapabilityBranchEnd, "session.branch_end", keyCapabilityBranchEndGap))
	}
	if s == nil || s.OnRefreshBranches == nil {
		capabilities = append(capabilities,
			wipCapabilityOf(keyCapabilityBranchRefresh, "session.branches", keyCapabilityBranchRefreshGap))
	}
	if s == nil || s.OnBranchFrom == nil {
		capabilities = append(capabilities,
			wipCapabilityOf(keyCapabilityBranchCut, "session.branch_create", keyCapabilityBranchCutGap))
	}
	if s == nil || !s.Keys.Loaded {
		capabilities = append(capabilities,
			wipCapabilityOf(keyCapabilityKeys, "session.keys", keyCapabilityKeysGap))
	}
	if s == nil || s.OnApproval == nil {
		capabilities = append(capabilities,
			wipCapabilityOf(keyCapabilityApprovals, "approval.respond", keyCapabilityApprovalsGap))
	}
	if s == nil || s.OnAttachFile == nil {
		capabilities = append(capabilities,
			wipCapabilityOf(keyCapabilityAttachments, "attachment.put", keyCapabilityAttachmentsGap))
	}
	if s == nil || s.OnReconnect == nil {
		capabilities = append(capabilities,
			wipCapabilityOf(keyCapabilityConnectionRecover, "supervisor.Reconnect", keyCapabilityConnectionGap))
	}
	if s == nil || s.OnStop == nil {
		capabilities = append(capabilities,
			wipCapabilityOf(keyCapabilityTurnCancel, "session.stop", keyCapabilityTurnCancelGap))
	}
	return capabilities
}

// capabilityCard draws one capability's status: its action, the method behind
// it, and the gap. Every part is text; the card has no button, link or field,
// so a user can read the status but cannot act on it. It is drawn as a popover
// rather than a card because it floats over the session list instead of
// belonging to the conversation. The method is the wire name and stays literal;
// action and gap are read in the active locale.
func capabilityCard(c *ui.Context, k tokens, locale Locale, capability wipCapability) {
	ui.Column(c).Padding(k.cardPadY, k.cardPadX).Gap(k.gap).Radius(k.cardRadius).
		Background(k.popover).BorderWidth(k.hairline).BorderColor(k.border).Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, tr(locale, capability.actionKey)).Bold().FontSize(k.smallSize).TextColor(k.popoverForeground)
			ui.Box(c).Grow(1)
			ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
		})
		ui.Text(c, capability.method).FontSize(k.smallSize).TextColor(k.mutedForeground).NoWrap()
		ui.Text(c, tr(locale, capability.gapKey)).FontSize(k.smallSize).TextColor(k.mutedForeground)
	})
}
