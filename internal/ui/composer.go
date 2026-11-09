// This file owns the composer region and the submit it sends through the
// window when the shell can carry it out. It reads ComposerView and Actions only
// (see protocols.go).

package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// composer holds the draft, the attachment entry and the send entry. It is a
// TextArea for now; IME composition is not claimed by this shell, only checked
// headless as draft text that survives a rebuild. ComposerView.Shortcut submits
// the draft; the shell's own focus shortcut reaches ComposerView.Focus.
func composer(c *ui.Context, k tokens, v ComposerView, a Actions) {
	ui.Column(c).Padding(k.edgePad).Gap(k.gap).Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			wipAction(c, k, "Attach file", reasonAttachment)
			ui.Box(c).Grow(1)
			send := ui.Button(c, "Send").OnClick(func() {
				submitDraft(v, a)
			})
			if !submitReady(v, a) {
				send.Disabled(true)
			}
		})
		ui.TextArea(c.Key("composer"), v.Draft).Bind(v.Focus).
			Height(k.composerH).Placeholder(v.Placeholder).Label("Composer").
			OnShortcut(v.Shortcut.Mods, v.Shortcut.Key, func() {
				submitDraft(v, a)
			})
	})
}

// submitReady reports whether one submission could be sent now: a wired submit
// callback, the window's session and connection gate, and draft text that is
// not only whitespace. The draft is read live, not from the frame, because a
// keystroke can land between two frames; a fixture without the rest cannot
// pretend a submission the App Server never received.
func submitReady(v ComposerView, a Actions) bool {
	return a.OnSubmit != nil && v.CanSubmit && v.Draft != nil && strings.TrimSpace(*v.Draft) != ""
}

// submitDraft sends the draft through the caller's submit callback when the
// shell can submit. The shell leaves the draft in place: the caller clears it
// once the App Server accepts the submission, so a failed send keeps the text.
func submitDraft(v ComposerView, a Actions) {
	if !submitReady(v, a) {
		return
	}
	a.OnSubmit(strings.TrimSpace(*v.Draft))
}
