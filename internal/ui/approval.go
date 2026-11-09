// This file owns a pending approval row and the two decisions it
// offers, as plain view state over the App Server's request identity. It reads
// ApprovalView and Actions only (see protocols.go); the timeline calls it for an
// AuthorApproval row.

package ui

import "github.com/egoist/mygo/ui"

// approval draws a pending approval request: its author line, its text and the
// two decisions it asks for.
func approval(c *ui.Context, k tokens, v ApprovalView, a Actions) {
	ui.Column(c).Padding(k.rowPadY, k.timelinePadX).Gap(k.gap).Children(func() {
		ui.Text(c, AuthorApproval.String()).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
		ui.Text(c, v.Text).FontSize(k.bodySize)
		approvalActions(c, k, v, a)
	})
}

// approvalActions are the decisions an approval row asks for. Both are real
// actions only when the window wires OnApproval and the row still carries the
// App Server's request identity; otherwise they stay visible, dimmed and
// unavailable, so a click cannot report a decision the App Server never
// received. The callback sends one decision and the row stays until the
// session projection reports the resolution: the shell never removes it or
// claims an outcome on its own.
func approvalActions(c *ui.Context, k tokens, v ApprovalView, a Actions) {
	ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		if a.OnApproval == nil {
			wipAction(c, k, "Approve", reasonApproval)
			wipAction(c, k, "Deny", reasonApproval)
			return
		}
		decide := a.OnApproval
		addressable := false
		for _, request := range v.Pending {
			if request.RequestID == "" {
				continue
			}
			addressable = true
			ui.Button(c, "Approve").OnClick(func() {
				decide(request.BranchID, request.RequestID, ApprovalAllow)
			})
			ui.Button(c, "Deny").OnClick(func() {
				decide(request.BranchID, request.RequestID, ApprovalDeny)
			})
		}
		if !addressable {
			wipAction(c, k, "Approve", reasonApproval)
			wipAction(c, k, "Deny", reasonApproval)
		}
	})
}
