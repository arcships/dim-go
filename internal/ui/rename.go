// This file owns the title dialog the toolbar's session menu opens: the
// prefilled single-line field, the confirm and cancel entries, and the one
// session.rename request a confirm sends through Actions.OnRename. It reads
// HeaderView and Actions only (see protocols.go).
//
// Escape, Cancel and a click outside the dialog only close it; they send
// nothing. The confirm entry closes the dialog first and then calls OnRename
// once with the field's text, so a frame can never send the request twice. An
// empty or whitespace-only text is the caller's signal to clear the manual
// title, and the region never changes the title it shows.

package ui

import (
	"github.com/egoist/mygo/ui"
)

// renameDialogMaxW is the dialog panel's width in DIPs: wide enough for a
// readable title and the two entries, narrow enough to stay a compact dialog in
// the reference's 800 DIP window.
const renameDialogMaxW = 360

// renameDialog draws the window's rename dialog while HeaderView.RenameConfirm
// is set. The field edits HeaderView.RenameDraft in place; the confirm entry
// writes the open flag false and calls Actions.OnRename with the text it holds.
func renameDialog(c *ui.Context, k tokens, v HeaderView, a Actions) {
	if v.RenameConfirm == nil || !*v.RenameConfirm || v.RenameDraft == nil {
		return
	}
	locale := v.Locale
	ui.DialogBase(c, v.RenameConfirm, func(backdrop, panel ui.Element) {
		backdrop.Background(renameScrim)
		panel.Width(renameDialogMaxW).Padding(k.cardPadY, k.cardPadX).Gap(k.gap).
			Radius(k.cardRadius).Background(k.background).
			BorderWidth(k.hairline).BorderColor(k.border)
		panel.Children(func() {
			ui.Text(c, tr(locale, keyHeaderRenameTitle)).Bold().FontSize(k.titleSize).TextColor(k.foreground)
			ui.Text(c, tr(locale, keyHeaderRenameMessage)).FontSize(k.smallSize).TextColor(k.mutedForeground)
			ui.TextInput(c, v.RenameDraft).Label(tr(locale, keyHeaderRenameLabel)).
				Placeholder(tr(locale, keyHeaderRenameLabel)).FillWidth()
			ui.Row(c).FillWidth().Gap(k.gap).Justify(ui.End).Children(func() {
				ui.Button(c, tr(locale, keyHeaderRenameCancel)).OnClick(func() {
					*v.RenameConfirm = false
				}).Transition(motionFastColors(c))
				ui.Button(c, tr(locale, keyHeaderRenameConfirm)).OnClick(func() {
					// Close first: the request is sent once, and the next frame
					// already shows the dialog gone even while the App Server
					// call is in flight.
					text := *v.RenameDraft
					*v.RenameConfirm = false
					if a.OnRename != nil {
						a.OnRename(text)
					}
				}).Transition(motionFastColors(c))
			})
		})
	})
}

// renameScrim is the dialog's backdrop: the same dimmed window the theme's own
// modal uses, so the title dialog reads as modal over the conversation.
var renameScrim = ui.RGBA(0, 0, 0, 0.4)
