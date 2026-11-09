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
// reasonDelete is what the delete entry says while the window has no delete
// callback: the client helper exists, but no window action is wired.
const reasonDelete = "Unavailable: the window has no session delete callback."

const (
	reasonCreateSession = "Unavailable: the window has no session.create callback yet."
	reasonApproval      = "Unavailable: the shell has no approval callback wired to approval.respond."
	reasonAttachment    = "Unavailable: attachment.put is not wired to a file picker yet; upload results may be unknown."
	reasonReconnect     = "Unavailable: the shell is not wired to supervisor reconnect yet."
	reasonStop          = "Unavailable: the shell has no session.stop callback yet."
	reasonBranches      = "Unavailable: branch switching/cut point is not wired yet."
	reasonUpdate        = "Unavailable: the shell has no update check wired to the app's updater yet."
	reasonSearch        = "Unavailable: the shell has no session search yet; dim-electron opens a search dialog over App Server history, and neither is wired here."
	reasonGroup         = "Unavailable: session rows carry no project or updated time yet, so the shell cannot group by project or date."
	reasonPin           = "Unavailable: no verified App Server RPC pins a session, and the shell keeps no local pin list."
	reasonRename        = "Unavailable: session.rename is a session method reached through session.methods, which the shell does not call yet."
	reasonArchive       = "Unavailable: the shell has no desktop-local hidden session list yet."

	// The global entries dim-electron's sidebar carries beside sessions. None of
	// them has a client, page or wire method here yet, so each entry keeps its
	// place, dimmed and explained, instead of opening a page that would only be
	// empty.
	reasonScheduled = "Unavailable: the shell has no scheduled-task client or page yet, and no public App Server method lists scheduled tasks."
	reasonSkills    = "Unavailable: the shell has no skills or plugins client or page yet, and no public App Server method lists skills or plugins."
	reasonAccount   = "Unavailable: the shell has no account client yet, so it cannot show an account or sign in."
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
	badge := ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).
		Unselectable().AccessHidden(true)
	button.DescribedBy(badge)
}
