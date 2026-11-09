// This file owns the composer region and the submit it sends through the
// window when the shell can carry it out. It reads ComposerView and Actions only
// (see protocols.go).

package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// The composer's geometry, from docs/ui-layout-spec.md §3.2: the editor's
// min-height is 60px and its max-height 300px, both px literals that do not
// scale with the root font size, as dim-electron's own stylesheet keeps them
// (A/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1491-1497).
// Past the cap the text area scrolls internally: mygo's Lines does that on its
// own, so the card around it does not grow.
const (
	composerMinH = 60
	composerMaxH = 300
	// composerMaxLines is the line cap mygo needs before MaxHeight can clamp.
	// It is deliberately above the cap, so the DIP limit decides where the
	// area stops growing rather than an estimate of the theme's line height.
	composerMaxLines = 64
)

// composerHint is the shortcut hint under the editor. The plain Enter and
// Shift+Enter pair is the region's own: Enter submits when the window's gate
// and a non-empty draft allow it, and Shift+Enter always falls through to the
// editor's newline. ComposerView.Shortcut (Cmd/Ctrl+Enter, from the shell) also
// submits, so the shell's documented shortcut keeps working.
const composerHint = "Enter to send · Shift+Enter for a new line"

// The reasons the composer's entries show while no data source is wired. The
// model and mode selectors have no field in ComposerView: App Server model
// facts and the agent mode are not passed to the region, so it has nothing true
// to list and no choice to write. They stay visible, disabled and explained
// rather than inventing a model list or a mode.
const (
	reasonModel = "Unavailable: ComposerView carries no model list or current model; App Server model facts are not passed to the composer yet."
	reasonMode  = "Unavailable: ComposerView carries no agent mode; the shell exposes no agent-mode fact to the composer yet."
)

// composer draws the draft editor and the action row under it: the attachment
// entry and the mode selector on the left, and the model selector and Send on
// the right, as dim-electron's composer does (ConversationComposer.vue:79-90,
// InputArea.vue:2456-2488). Send and the Enter key both submit the trimmed
// draft; the caller clears it once the App Server accepted the submission, so a
// failed send keeps the text.
func composer(c *ui.Context, k tokens, v ComposerView, a Actions) {
	ui.Column(c).Padding(k.edgePad).Gap(k.gap).
		Radius(k.cardRadius).BorderWidth(k.hairline).BorderColor(k.border).
		Background(k.background).Children(func() {
		// The row is built first so the keyboard order stays the one the shell
		// contract documents (Attach file, then the editor), while Reverse draws
		// it at the card's bottom, the editor first, as the spec lays it out.
		ui.Column(c).Gap(k.gap).Reverse().Children(func() {
			composerActions(c, k, v, a)
			composerHintRow(c, k)
			composerEditor(c, k, v, a)
		})
	})
}

// composerEditor draws the draft field, as high as its wrapped text between
// the spec's 60 and 300 DIPs, scrolling inside past the cap. Enter submits, and
// Shift+Enter inserts a newline; a modified Enter stays for ComposerView's own
// shortcut, which the shell wires to Cmd/Ctrl+Enter.
func composerEditor(c *ui.Context, k tokens, v ComposerView, a Actions) {
	ui.TextArea(c.Key("composer"), v.Draft).Bind(v.Focus).
		Lines(1, composerMaxLines).MinHeight(composerMinH).MaxHeight(composerMaxH).
		Placeholder(v.Placeholder).Label("Composer").
		OnShortcut(v.Shortcut.Mods, v.Shortcut.Key, func() {
			submitDraft(v, a)
		}).
		OnTextKey(func(key ui.TextKey) bool {
			if key.Key != ui.KeyEnter || key.Mods != 0 {
				// Shift+Enter and Cmd/Ctrl+Enter fall through: the first is the
				// editor's newline, the second is ComposerView.Shortcut above.
				return false
			}
			text := strings.TrimSpace(key.Text())
			if !canSubmit(v, a) || text == "" {
				// Nothing can be sent: keep the editor's newline rather than
				// swallowing the key, as an empty message field does. The live
				// text, not the frame's draft, is what the key came with.
				return false
			}
			a.OnSubmit(text)
			return true
		})
}

// composerActions draws the bottom row: the attachment entry and the mode
// selector on the left, the model selector and Send on the right.
func composerActions(c *ui.Context, k tokens, v ComposerView, a Actions) {
	ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		// The attachment entry keeps wipAction's place in the Tab order, as the
		// shell's keyboard contract documents it. The model and mode selectors
		// below use Disabled instead: they are new entries, and a tab stop would
		// insert itself between "Attach file" and the editor, changing that
		// documented order. They are still visible, read as disabled with their
		// reason, and take no focus.
		wipAction(c, k, "Attach file", reasonAttachment)
		composerWIP(c, k, "Mode", reasonMode)
		ui.Box(c).Grow(1)
		composerWIP(c, k, "Model", reasonModel)
		send := ui.Button(c, "Send").OnClick(func() {
			submitDraft(v, a)
		})
		if !submitReady(v, a) {
			send.Disabled(true)
		}
	})
}

// composerWIP draws one composer entry the region cannot offer yet: visible,
// disabled, with the reason it waits on, and a badge that says so without being
// read as a control of its own. Disabled keeps it out of the Tab order, unlike
// wipAction's sidebar entries, because the composer's order is pinned by the
// shell contract; a click cannot report anything the App Server did not do.
func composerWIP(c *ui.Context, k tokens, label, reason string) {
	button := ui.Button(c, label).Disabled(true).AccessDisabled(true).Description(reason)
	badge := ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).
		Unselectable().AccessHidden(true)
	button.DescribedBy(badge)
}

// composerHintRow draws the shortcut hint under the editor. It is text, not a
// control, so it adds no Tab stop.
func composerHintRow(c *ui.Context, k tokens) {
	ui.Text(c, composerHint).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
}

// canSubmit reports whether the window would take a submission now: a wired
// submit callback, the window's session and connection gate, no submission
// already in flight, and a draft to read. It says nothing about the draft's
// text, because a key can arrive before the frame's value catches up.
func canSubmit(v ComposerView, a Actions) bool {
	return a.OnSubmit != nil && v.CanSubmit && !v.Sending && v.Draft != nil
}

// submitReady reports whether one submission could be sent from the frame's
// draft: canSubmit plus draft text that is not only whitespace. The draft is
// read live, not from the frame, because a keystroke can land between two
// frames; a fixture without the rest cannot pretend a submission the App
// Server never received.
func submitReady(v ComposerView, a Actions) bool {
	return canSubmit(v, a) && strings.TrimSpace(*v.Draft) != ""
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
