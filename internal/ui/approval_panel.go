// This file owns the approval decision panel: the prompt the window shows in
// the composer slot while the App Server has a request pending. It reads
// ApprovalView and Actions only (see protocols.go), like the inline approval
// row in approval.go, and it keeps the same click contract: one decision per
// click, sent through Actions.OnApproval with the App Server's request
// identity, and no local claim that the request was resolved.
//
// TODO(protocol): ApprovalView carries only the projection's text and the
// pending request identities. It has no tool-input payload, no approval variant
// and no response-option list, so the panel cannot draw the command/file input
// block or the allow_once/reject_once variants docs/ui-layout-spec.md §5.1
// describes. Extending the view is a protocol change (protocols.go and
// docs/ui-component-protocols.md §2.4) and belongs to that file's owner. This
// entry is not registered beside renderApproval in protocols.go yet for the same
// reason; the shell can call it directly once the owner adds the wrapper.
//
// TODO(keyboard): the panel keeps its decisions reachable with Tab and Enter,
// which is what mygo buttons expose without a global key handler. The prompt's
// Enter/y = allow and Esc/n = deny shortcuts (docs/ui-layout-spec.md §5.2) need
// a verified way to ignore shortcuts while an input has the focus; until then
// they stay unavailable rather than firing a decision from anywhere.

package ui

import "github.com/egoist/mygo/ui"

// The approval panel's geometry in DIPs, from docs/ui-layout-spec.md §5.2:
// the root is px-3 pb-3, the card is p-4 with mt-3 between its parts, and the
// action row is gap-2. Under the 14px root font size those Tailwind steps are
// 10.5, 14, 10.5 and 7 DIPs. They stay local until tokens.go's owner adds the
// panel's tokens: this branch only owns the approval files, and a region may
// not change the shared token contract on its own.
const (
	approvalPanelPadX = 10.5 // px-3: the panel's left and right
	approvalPanelPadY = 10.5 // pb-3: below the panel
	approvalCardPad   = 14   // p-4: inside the card
	approvalCardGap   = 10.5 // mt-3: between the card's parts
	approvalActionGap = 7    // gap-2: between two decisions
)

// renderApprovalPanel draws the pending approval requests in the composer's
// slot. A view with no pending request draws nothing at all: the slot is free
// for the composer, not an empty card. The first pending request is the primary
// one: its call name is the heading, and every request keeps its own call,
// reason, source and decisions so a click can never address a different request
// than the one it belongs to.
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
					approvalPanelRequest(c, k, request, a, i == 0)
				}
			})
		})
	})
}

// approvalPanelRequest draws one pending request: its call name, why it asks,
// where it came from, and the two decisions. The first request is the primary
// one and names the panel with its call; the rest are drawn the same way so a
// request the projection still reports is never silently dropped.
func approvalPanelRequest(c *ui.Context, k tokens, request Approval, a Actions, primary bool) {
	ui.Column(c).Gap(k.gap).Children(func() {
		if request.Call != "" {
			name := ui.Text(c, request.Call).FontSize(k.bodySize).Bold()
			if primary {
				name.FontSize(k.titleSize)
			}
		}
		if request.Reason != "" {
			ui.Text(c, request.Reason).FontSize(k.bodySize)
		}
		if request.Source != "" {
			ui.Text(c, "Source: "+request.Source).FontSize(k.smallSize).TextColor(k.mutedForeground)
		}
		approvalPanelActions(c, k, request, a)
	})
}

// approvalPanelActions are one request's decisions. They are real only when the
// window wired OnApproval and the request carries the App Server's request
// identity; otherwise they stay visible, dimmed and explained, so a click
// cannot report a decision nobody sent. A click sends exactly one decision; the
// panel never removes the request or shows a success, because only the next
// session projection frame may settle it.
func approvalPanelActions(c *ui.Context, k tokens, request Approval, a Actions) {
	ui.Row(c).Gap(approvalActionGap).Justify(ui.End).Children(func() {
		if a.OnApproval == nil || request.RequestID == "" {
			wipAction(c, k, "Approve", reasonApproval)
			wipAction(c, k, "Deny", reasonApproval)
			return
		}
		decide := a.OnApproval
		branchID, requestID := request.BranchID, request.RequestID
		ui.Button(c, "Approve").OnClick(func() {
			decide(branchID, requestID, ApprovalAllow)
		})
		ui.Button(c, "Deny").OnClick(func() {
			decide(branchID, requestID, ApprovalDeny)
		})
	})
}
