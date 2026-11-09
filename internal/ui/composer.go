// This file owns the composer region and the submit, stop and
// reconnect callbacks it sends through the window when the shell can carry them
// out.

package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// composer holds the draft, the attachment entry and the send entry. It is a
// TextArea for now; IME composition is not claimed by this shell, only checked
// headless as draft text that survives a rebuild. ComposerShortcutMods with
// ComposerShortcutKey gives it the focus; the same modifier with Enter submits
// the draft.
func (s *State) composer(c *ui.Context, k tokens) {
	ui.Column(c).Padding(k.edgePad).Gap(k.gap).Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			wipAction(c, k, "Attach file", reasonAttachment)
			ui.Box(c).Grow(1)
			send := ui.Button(c, "Send").OnClick(s.submitDraft)
			if !s.canSubmit() {
				send.Disabled(true)
			}
		})
		ui.TextArea(c.Key("composer"), &s.Draft).Bind(&s.composerFocus).
			Height(k.composerH).Placeholder("Message DimAgent").Label("Composer").
			OnShortcut(ComposerShortcutMods, ui.KeyEnter, s.submitDraft)
	})
}

// canSubmit reports whether the shell has a selected session on a ready
// connection, text to send, and a callback to send it through. A fixture
// without those cannot pretend a submission the App Server never received.
func (s *State) canSubmit() bool {
	return s.OnSubmit != nil && s.Connection == ConnReady && !s.ConversationLoading &&
		!s.SessionClosed && s.Selected != "" && strings.TrimSpace(s.Draft) != ""
}

// submitDraft sends the draft through the caller's submit callback when the
// shell can submit. The shell leaves the draft in place: the caller clears it
// once the App Server accepts the submission, so a failed send keeps the text.
func (s *State) submitDraft() {
	if !s.canSubmit() {
		return
	}
	s.OnSubmit(strings.TrimSpace(s.Draft))
}

// stopRun asks the caller to stop the running turn.
func (s *State) stopRun() {
	if s.OnStop != nil {
		s.OnStop()
	}
}

func (s *State) reconnect() {
	if s.OnReconnect != nil {
		s.OnReconnect()
	}
}
