// This file owns a pending approval row and the two decisions it
// offers, as plain view state over the App Server's request identity.

package ui

import "github.com/egoist/mygo/ui"

// approvalRow draws a pending approval request: its author line, its text and
// the two decisions it asks for.
func (s *State) approvalRow(c *ui.Context, k tokens, m Message) {
	ui.Column(c).Padding(k.rowPadY, k.timelinePadX).Gap(k.gap).Children(func() {
		ui.Text(c, m.Author.String()).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
		ui.Text(c, m.Content).FontSize(k.bodySize)
		s.approvalActions(c, k, m)
	})
}

// approvalActions are the decisions an approval row asks for. Both are real
// actions only when the window wires OnApproval and the row still carries the
// App Server's request identity; otherwise they stay visible, dimmed and
// unavailable, so a click cannot report a decision the App Server never
// received. The callback sends one decision and the row stays until the
// session projection reports the resolution: the shell never removes it or
// claims an outcome on its own.
func (s *State) approvalActions(c *ui.Context, k tokens, m Message) {
	ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		if m.Approval == nil || m.Approval.RequestID == "" || s.OnApproval == nil {
			wipAction(c, k, "Approve", reasonApproval)
			wipAction(c, k, "Deny", reasonApproval)
			return
		}
		approval, decide := *m.Approval, s.OnApproval
		ui.Button(c, "Approve").OnClick(func() { decide(approval.BranchID, approval.RequestID, ApprovalAllow) })
		ui.Button(c, "Deny").OnClick(func() { decide(approval.BranchID, approval.RequestID, ApprovalDeny) })
	})
}
