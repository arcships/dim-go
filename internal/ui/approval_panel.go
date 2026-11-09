// This file owns the approval decision panel: the prompt the window shows in
// the composer slot while the App Server has a request pending. It reads
// ApprovalView and Actions only (see protocols.go), like the inline approval
// row in approval.go, and it keeps the same click contract: one decision per
// click, sent through Actions.OnApproval with the App Server's request
// identity, and no local claim that the request was resolved.
//
// The view now carries the primary request's rendered input (InputDisplay) and
// the UI option it is drawn with (Variant), both described in protocols.go. Two
// gaps remain outside this file: shell.View still passes only Pending, and
// viewmodel.BuildApprovals copies neither Call.Input nor an option, so the app
// shows neither the input block nor a variant label until those owners wire
// them. The region must not read the request itself to fill them.
//
// TODO(grants): allow_all promises a persistent grant, which §12 routes through
// session.grants.add. The shell does not call that method, so the panel keeps
// "Always allow" visible, dimmed and explained instead of sending the one-off
// allow that would silently be a different decision. See reasonApprovalGrant.
//
// TODO(keyboard): the panel keeps its decisions reachable with Tab and Enter,
// which is what mygo buttons expose without a global key handler. The prompt's
// Enter/y = allow and Esc/n = deny shortcuts (docs/ui-layout-spec.md §5.2) need
// a verified way to ignore shortcuts while an input has the focus; until then
// they stay unavailable rather than firing a decision from anywhere.

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

	approvalInputPadX = 10.5 // px-3: inside the input block
	approvalInputPadY = 7    // py-2: above and below the input block's body
	approvalInputMaxH = 240  // max-h-[240px]: the input block's cap
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

// reasonApprovalGrant is the gap behind "Always allow": §12 sends a durable
// grant through session.grants.add, after session.methods advertises it. The
// shell calls neither, so the option stays visible and dimmed rather than
// answering allow_once under the always-allow label.
const reasonApprovalGrant = "Unavailable: always-allow needs session.grants.add, which the shell does not call yet."

// renderApprovalPanel draws the pending approval requests in the composer's
// slot. A view with no pending request draws nothing at all: the slot is free
// for the composer, not an empty card. The first pending request is the primary
// one: its call name is the heading, its input is the view's input block, and
// every request keeps its own call, reason, source and decisions so a click can
// never address a different request than the one it belongs to.
func renderApprovalPanel(c *ui.Context, k tokens, v ApprovalView, a Actions) {
	if len(v.Pending) == 0 {
		return
	}
	ui.Column(c).Padding(0, approvalPanelPadX, approvalPanelPadY, approvalPanelPadX).Children(func() {
		ui.Column(c).Padding(approvalCardPad).Radius(radiusMD).
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
					approvalPanelRequest(c, k, request, a, i == 0, input, v.Variant)
				}
			})
		})
	})
}

// approvalPanelRequest draws one pending request: its call name, its input when
// the view carries the primary one, why it asks, where it came from, and the
// two decisions. The first request is the primary one and names the panel with
// its call; the rest are drawn the same way so a request the projection still
// reports is never silently dropped. Only the primary request has an input
// payload in the view, so the block appears once.
func approvalPanelRequest(c *ui.Context, k tokens, request Approval, a Actions, primary bool, input string, variant ApprovalVariant) {
	ui.Column(c).Gap(k.gap).Children(func() {
		if request.Call != "" {
			name := ui.Text(c, request.Call).FontSize(k.bodySize).Bold()
			if primary {
				name.FontSize(k.titleSize)
			}
		}
		if input != "" {
			approvalPanelInput(c, k, input)
		}
		if request.Reason != "" {
			ui.Text(c, request.Reason).FontSize(k.bodySize)
		}
		if request.Source != "" {
			ui.Text(c, "Source: "+request.Source).FontSize(k.smallSize).TextColor(k.mutedForeground)
		}
		approvalPanelActions(c, k, request, a, variant)
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
		Radius(radiusMD).Border(k.hairline, k.border).Background(k.card).Clip().
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

// approvalLabels are the two decisions' labels for a variant. Only the option
// names the contract documents change the wording; an empty or unknown variant
// keeps plain Approve/Deny rather than guessing at allow_once or allow_all.
func approvalLabels(variant ApprovalVariant) (allow, deny string) {
	switch variant {
	case ApprovalVariantAllowOnce:
		return "Allow once", "Deny"
	case ApprovalVariantAllowAll:
		return "Always allow", "Deny"
	case ApprovalVariantRejectOnce:
		return "Approve", "Deny once"
	default:
		return "Approve", "Deny"
	}
}

// approvalPanelActions are one request's decisions. They are real only when the
// window wired OnApproval and the request carries the App Server's request
// identity; otherwise they stay visible, dimmed and explained, so a click
// cannot report a decision nobody sent. A click sends exactly one decision; the
// panel never removes the request or shows a success, because only the next
// session projection frame may settle it.
//
// The variant changes the labels, not the decision: allow_once and allow_all
// both send ApprovalAllow, reject_once sends ApprovalDeny. allow_all's allow
// entry is the one exception: its persistent grant is session.grants.add, which
// is not implemented, so it stays a WIP entry instead of sending the one-off
// allow that would not keep the promise its label makes.
func approvalPanelActions(c *ui.Context, k tokens, request Approval, a Actions, variant ApprovalVariant) {
	allowLabel, denyLabel := approvalLabels(variant)
	ui.Row(c).Gap(approvalActionGap).Justify(ui.End).Children(func() {
		if a.OnApproval == nil || request.RequestID == "" {
			wipAction(c, k, allowLabel, reasonApproval)
			wipAction(c, k, denyLabel, reasonApproval)
			return
		}
		decide := a.OnApproval
		branchID, requestID := request.BranchID, request.RequestID
		if variant == ApprovalVariantAllowAll {
			wipAction(c, k, allowLabel, reasonApprovalGrant)
		} else {
			ui.Button(c, allowLabel).OnClick(func() {
				decide(branchID, requestID, ApprovalAllow)
			})
		}
		ui.Button(c, denyLabel).OnClick(func() {
			decide(branchID, requestID, ApprovalDeny)
		})
	})
}
