package ui

import "github.com/egoist/mygo/ui"

// wipBadge is what the shell shows beside an action it cannot carry out yet,
// so that the gap is visible and not only a dimmed control.
const wipBadge = "WIP"

// wipOpacity fades an action the shell cannot carry out yet, as Disabled
// dims a control. The badge beside it says why it looks that way.
const wipOpacity = 0.5

// The reasons the shell's unsupported actions show. Each names the upstream
// method or capability the entry waits on, so the entry explains itself
// instead of pretending the action worked. Keep them in step with
// docs/protocol-contract.md and docs/development-plan.md; the App Server
// remains the only source of the facts those actions would change.
const (
	reasonCreateSession = "Unavailable: the shell does not call session.create yet."
	reasonApproval      = "Unavailable: the shell does not call approval.respond yet."
	reasonAttachment    = "Unavailable: the shell does not call attachment.put yet."
	reasonReconnect     = "Unavailable: App Server reconnect/resync is not implemented yet."
)

// wipAction draws one entry for an action the shell cannot carry out yet. It
// keeps the action's place and accessible name, dims it, tells assistive
// technology that it is unavailable, and reads out the reason it waits on.
// The view never asks whether it was clicked, so the shell cannot report a
// success that the App Server did not perform.
//
// Call it inside a Row: it builds the button and, after it, the visible badge
// that describes the button without being read as a control of its own.
func wipAction(c *ui.Context, k tokens, label, reason string) {
	button := ui.Button(c, label).AccessDisabled(true).Description(reason).Opacity(wipOpacity)
	badge := ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.muted).
		Unselectable().AccessHidden(true)
	button.DescribedBy(badge)
}
