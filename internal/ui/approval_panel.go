// This file owns the approval decision panel: the prompt the window shows in
// the composer slot while the App Server has a request pending. It reads
// ApprovalView and Actions only (see protocols.go), and it keeps the click
// contract: one decision per click, sent through Actions.OnApproval with the
// App Server's request identity, and no local claim that the request was
// resolved.
//
// The panel follows the reference prompt (docs/ui-layout-spec.md §5). It takes
// the composer's slot at the conversation card's 520 DIP measure, states the
// title and the reason, draws the primary request's input as a read-only
// monospaced block, and ends in the reference's three decisions with reject
// first. Reject is the danger-colored decision. The actions wrap onto a new
// line in a narrow window, so the card never grows past the slot it is given,
// and Tab reaches every decision and Enter sends the focused one.
//
// The always-allow decision is a WIP entry: its persistent grant is
// session.grants.add, which the shell does not call, so it stays visible,
// dimmed and explained instead of sending the one-off allow under a label that
// promises more. See reasonApprovalGrant.
//
// Residual gaps outside this file:
//   - The panel draws the reference's whole option set instead of selecting a
//     pair of labels from ApprovalView.Variant. The wire carries no option
//     list (docs/protocol-contract.md §12) and viewmodel.BuildApprovals passes
//     ApprovalVariantUnspecified today, so the variant cannot say which
//     entries a request really offers; the panel keeps the entries it can send
//     (deny, allow) and marks the grant entry WIP. protocols.go's field comment
//     still describes the old label fallback and needs its owner to reconcile
//     it.
//   - ApprovalView has no in-flight state, so a decision that was sent cannot
//     be disabled until the next projection frame. Only that frame may settle
//     the request; this region must not claim the decision was accepted.
//   - The reference's Enter/y = allow and Esc/n = deny shortcuts need a
//     verified way to ignore shortcuts while an input has focus. Tab and Enter
//     stay the reachable decisions, which is what mygo buttons expose without
//     a global key handler.

package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// The approval panel's geometry in DIPs, from docs/ui-layout-spec.md §5.2:
// the root is px-3 pb-3, the card is p-4 with mt-3 between its parts, and the
// action row is gap-2. Under the 14px root font size those Tailwind steps are
// 10.5, 14, 10.5 and 7 DIPs. They stay local until tokens.go's owner adds the
// panel's tokens: this branch only owns the approval files, and a region may
// not change the shared token contract on its own.
//
// The input block's own steps are §5.1's ApprovalInputPanel and
// CommandApprovalPanel: max-h-[240px] with px-3 py-2 and a monospaced body.
// The 240 is a px literal in the renderer, so it stays literal here.
const (
	approvalPanelPadX = 10.5 // px-3: the panel's left and right
	approvalPanelPadY = 10.5 // pb-3: below the panel
	approvalCardPad   = 14   // p-4: inside the card
	approvalCardGap   = 10.5 // mt-3: between the card's parts
	approvalActionGap = 7    // gap-2: between two decisions

	// approvalPanelWidth is the card's measure in DIPs, the same 520 DIP
	// conversation card the composer draws (composerWideWidth): the panel takes
	// the composer's slot, so it keeps the slot's measure and is centred in the
	// room it has. It is a cap, not a filler, so a narrower window draws the
	// card across that window instead of overflowing it.
	approvalPanelWidth = 520

	approvalInputPadX = 10.5 // px-3: inside the input block
	approvalInputPadY = 7    // py-2: above and below the input block's body
	approvalInputMaxH = 240  // max-h-[240px]: the input block's cap
)

// The decisions the reference's prompt offers, in its order: reject first, then
// the one-off allow, then the always-allow entry (docs/ui-layout-spec.md §5.2,
// "选项顺序 = reject → allow_once → 其余"). The first two collapse onto
// approval.respond's deny and allow. The last one would persist a grant through
// session.grants.add, which the shell does not call, so it is always drawn as a
// WIP entry and sends nothing.
const (
	approvalLabelReject   = "Reject once"
	approvalLabelAllow    = "Allow once"
	approvalLabelAllowAll = "Allow all"
)

// approvalInputFont is the monospaced family the input block uses, as §5.1's
// font-mono body does. mygo resolves "monospace" to the platform's family.
const approvalInputFont = "monospace"

// approvalInputMaxRunes bounds the input text itself, before it reaches the
// scroll block: an arbitrarily large Call.Input must not become an
// arbitrarily large text element. The block still scrolls input up to this
// cap; past it the text ends in an ellipsis, so the block never claims to show
// all of it. The cap is a display bound, not a protocol one.
const approvalInputMaxRunes = 2000

// reasonApprovalGrant is the gap behind "Allow all": §12 sends a durable grant
// through session.grants.add, after session.methods advertises it. The shell
// calls neither, so the option stays visible and dimmed rather than answering
// allow_once under the always-allow label.
const reasonApprovalGrant = "Unavailable: always-allow needs session.grants.add, which the shell does not call yet."

// renderApprovalPanel draws the pending approval requests in the composer's
// slot. A view with no pending request draws nothing at all: the slot is free
// for the composer, not an empty card. The first pending request is the primary
// one: its input is the view's input block, and every request keeps its own
// call, reason, source and decisions so a click can never address a different
// request than the one it belongs to.
//
// The panel is capped and centred at the composer's 520 DIP measure, so it sits
// where the composer sat and narrows with the window instead of overflowing the
// slot.
func renderApprovalPanel(c *ui.Context, k tokens, v ApprovalView, a Actions) {
	if len(v.Pending) == 0 {
		return
	}
	ui.Column(c).FillWidth().Padding(0, approvalPanelPadX, approvalPanelPadY, approvalPanelPadX).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(approvalPanelWidth).AlignSelf(ui.Center).
			Padding(approvalCardPad).Radius(k.radiusMD).
			Background(k.card).Border(k.hairline, k.border).
			Role(ui.RoleGroup).Label("Approval request").Children(func() {
			ui.Column(c).Gap(approvalCardGap).Children(func() {
				ui.Text(c, "Approval required").Bold().FontSize(k.titleSize)
				if v.Text != "" {
					ui.Text(c, v.Text).FontSize(k.bodySize)
				}
				for i, request := range v.Pending {
					input := ""
					if i == 0 {
						input = v.InputDisplay
					}
					approvalPanelRequest(c, k, request, a, input)
				}
			})
		})
	})
}

// approvalPanelRequest draws one pending request: its call name, its input when
// the view carries the primary one, why it asks, where it came from, and the
// reference's three decisions. The first request's call names the panel's
// action and the others are drawn the same way, so a request the projection
// still reports is never silently dropped. Only the primary request has an
// input payload in the view, so the block appears once.
func approvalPanelRequest(c *ui.Context, k tokens, request Approval, a Actions, input string) {
	ui.Column(c).Gap(k.gap).Children(func() {
		if request.Call != "" {
			ui.Text(c, request.Call).FontSize(k.bodySize).Bold()
		}
		if input != "" {
			approvalPanelInput(c, k, input)
		}
		if request.Reason != "" {
			ui.Text(c, request.Reason).FontSize(k.bodySize).TextColor(k.mutedForeground)
		}
		if request.Source != "" {
			ui.Text(c, "Source: "+request.Source).FontSize(k.smallSize).TextColor(k.mutedForeground)
		}
		approvalPanelActions(c, k, request, a)
	})
}

// approvalPanelInput draws the primary request's input as a read-only block:
// a bounded, scrolled, monospaced body inside the card's own border, as §5.1's
// ApprovalInputPanel does. It is named as a group so the block is one thing to
// assistive technology; a view without input draws no block at all, not an
// empty frame. The text is trimmed and capped (approvalInputMaxRunes); the
// region neither parses the input nor edits it, and no text field is created,
// so it cannot be mistaken for an editor.
func approvalPanelInput(c *ui.Context, k tokens, input string) {
	text := approvalInputText(input)
	if text == "" {
		return
	}
	ui.Column(c).Padding(approvalInputPadY, approvalInputPadX, approvalInputPadY, approvalInputPadX).
		Radius(k.radiusMD).Border(k.hairline, k.border).Background(k.card).Clip().
		Role(ui.RoleGroup).Label("Approval input").Children(func() {
		ui.Scroll(c).MaxHeight(approvalInputMaxH).Children(func() {
			ui.Text(c, text).Font(approvalInputFont).FontSize(k.codeFontSize).TextColor(k.cardForeground)
		})
	})
}

// approvalInputText is the input block's body: the view's display string with
// its outer whitespace removed, cut to approvalInputMaxRunes runes with an
// ellipsis when it is longer. Blank input gives "", which draws no block.
func approvalInputText(input string) string {
	text := strings.TrimSpace(input)
	runes := []rune(text)
	if len(runes) <= approvalInputMaxRunes {
		return text
	}
	return string(runes[:approvalInputMaxRunes]) + "…"
}

// approvalPanelActions are one request's decisions, in the reference's order:
// reject, allow once, and the always-allow WIP entry. They are real only when
// the window wired OnApproval and the request carries the App Server's request
// identity; otherwise they stay visible, dimmed and explained, so a click
// cannot report a decision nobody sent. A click sends exactly one decision; the
// panel never removes the request or shows a success, because only the next
// session projection frame may settle it.
//
// Reject is drawn in the danger color: it is the decision that refuses the
// request, and the color is the shell's token for that meaning, not a second
// claim about the request. The row wraps and stays end-aligned, so the three
// decisions keep their order on one line at the composer's measure and stack
// in a narrow window without running past the card.
func approvalPanelActions(c *ui.Context, k tokens, request Approval, a Actions) {
	enabled := a.OnApproval != nil && request.RequestID != ""
	decide := a.OnApproval
	branchID, requestID := request.BranchID, request.RequestID
	ui.Row(c).Gap(approvalActionGap).AlignItems(ui.Center).Justify(ui.End).Wrap().Children(func() {
		approvalDecision(c, k, approvalLabelReject, reasonApproval, enabled, true, func() {
			decide(branchID, requestID, ApprovalDeny)
		})
		approvalDecision(c, k, approvalLabelAllow, reasonApproval, enabled, false, func() {
			decide(branchID, requestID, ApprovalAllow)
		})
		// Always a WIP entry: the grant it promises needs session.grants.add,
		// even while the two one-off decisions are wired.
		approvalDecision(c, k, approvalLabelAllowAll, reasonApprovalGrant, false, false, nil)
	})
}

// approvalDecision draws one decision as one unit of the wrapping action row:
// a real button when the decision can be sent, or the visible-disabled WIP
// entry with its badge when it cannot. The row around it keeps the badge beside
// its button when the actions wrap, instead of leaving it alone on the next
// line. danger draws the button's text and outline in the shell's danger color.
func approvalDecision(c *ui.Context, k tokens, label, reason string, enabled, danger bool, decide func()) {
	ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		if !enabled {
			wipAction(c, k, label, reason)
			return
		}
		button := ui.Button(c, label)
		if danger {
			button.TextColor(k.danger).BorderColor(k.danger)
		}
		button.OnClick(decide)
	})
}
