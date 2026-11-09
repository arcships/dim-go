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
	// composerSendSpan is the Send button's diameter in theme spacing units:
	// the spec's h-7 w-7 (28 DIPs at the 4 DIP unit), so a compact density
	// shrinks the round button with the rest of the frame.
	composerSendSpan = 7
	// composerToolSpan is one toolbar glyph button's square in theme spacing
	// units: h-7 w-7, the reference's icon-button size beside the send disc
	// (ConversationComposer.vue:79-90).
	composerToolSpan = 7
	// composerContextSpan is the context row's minimum height in theme spacing
	// units: the reference card's last band is ≈36 DIPs (h-9 at the 4 DIP unit).
	composerContextSpan = 9
	// composerEmptyWidth and composerWideWidth are the card's measures in
	// DIPs, as the reference draws them: the workspace's empty state keeps the
	// 461 DIP card of window-workspace.png, and a conversation follows the
	// wider 520 DIP card of window-conversation.png. Both are caps, not
	// filler: a narrower column draws the card across its own width, and the
	// card is centred in the room it has rather than running edge to edge.
	composerEmptyWidth = 461
	composerWideWidth  = 520
	// composerIconSize is the square every toolbar glyph is drawn at, the same
	// 16 DIP artwork the sidebar and the toolbar use.
	composerIconSize = 16
)

// composerSendGlyph is what the round Send button shows. It is a decoration:
// the button's accessible name is the localized Send text, so the arrow is
// hidden from assistive technology and the entry is never read as "up arrow".
const composerSendGlyph = "↑"

// composerHint is the shortcut help the editor carries. The plain Enter and
// Shift+Enter pair is the region's own: Enter submits when the window's gate
// and a non-empty draft allow it, and Shift+Enter always falls through to the
// editor's newline. ComposerView.Shortcut (Cmd/Ctrl+Enter, from the shell) also
// submits, so the shell's documented shortcut keeps working. The reference has
// no hint line under the editor, so the pair is the editor's Description and
// its KeyShortcuts rather than a visible row that takes room from the card.
const composerHint = "Enter to send · Shift+Enter for a new line"

// The reasons the composer's entries show while no data source is wired. The
// model and mode selectors have no field in ComposerView: App Server model
// facts and the agent mode are not passed to the region, so it has nothing true
// to list and no choice to write. They stay visible, disabled and explained
// rather than inventing a model list or a mode.
const (
	reasonModel = "Unavailable: ComposerView carries no model list or current model; App Server model facts are not passed to the composer yet."
	reasonMode  = "Unavailable: ComposerView carries no agent mode; the shell exposes no agent-mode fact to the composer yet."
	// reasonWorkspace and reasonPermission explain the context row's WIP
	// entries. The shell passes the facts it read, and has none today, so the
	// region draws the WIP state and this reason instead of a path or a mode it
	// invented.
	reasonWorkspace  = "Unavailable: ComposerView carries no workspace; the shell has no App Server workspace fact for the composer yet."
	reasonPermission = "Unavailable: ComposerView carries no default permission mode; the shell has no App Server permission fact for the composer yet."
)

// composerShadow is the card's soft outer shadow, the reference's low-alpha
// ring under the composer (ConversationComposer.vue:61-65). It is a black wash
// at about 6%, so it stays a lift in both appearances instead of a second
// border.
var composerShadow = ui.Hex("#00000010")

// composerGlyphs is the icon set the card's toolbar and context row draw, in
// the same 16×16 outline frame as the sidebar's (icons.go). The cube is the
// model entry's mark, the shield is the default permission mode's, the down
// chevron is the workspace picker's disclosure, and the double chevron is the
// mode entry's; they are drawn here because no other region uses them yet.
var composerGlyphs = struct {
	cube        *ui.SVG
	shield      *ui.SVG
	chevronDown *ui.SVG
	modeChevron *ui.SVG
}{
	cube:   sidebarIcon(`<path d="M8 1.8 13.2 4.9v6.2L8 14.2 2.8 11.1V4.9Z"/><path d="M2.8 4.9 8 8l5.2-3.1M8 8v6.2"/>`),
	shield: sidebarIcon(`<path d="M8 1.8 13 3.6v4.3c0 3.1-2 5.4-5 6.3-3-.9-5-3.2-5-6.3V3.6Z"/>`),
	// chevronDown is the workspace picker's disclosure, the reference's ⌄
	// beside the workspace name rather than the sidebar's forward chevron.
	chevronDown: sidebarIcon(`<path d="M3.5 6 8 10.5 12.5 6"/>`),
	// modeChevron is the Mode entry's mark: the reference draws the agent-mode
	// picker as the » pair of chevrons, not as the single forward chevron the
	// sidebar uses for its disclosure. Two strokes, so the entry reads as a
	// choice rather than as a link.
	modeChevron: sidebarIcon(`<path d="M3.5 4 6.5 8l-3 4"/><path d="M8.5 4 11.5 8l-3 4"/>`),
}

// composer draws the card dim-electron's composer is: the editor on top, the
// tool row under it, and the context row as the card's last band. The card is
// one group centred at the reference's measure, and it keeps the toolbar, the
// editor and the context row inside its own border and rounded corners. Send
// and the Enter key both submit the trimmed draft; the caller clears it once
// the App Server accepted the submission, so a failed send keeps the text.
func composer(c *ui.Context, k tokens, v ComposerView, a Actions) {
	ui.Column(c).Role(ui.RoleGroup).FillWidth().MaxWidth(composerWidth(v.Wide)).AlignSelf(ui.Center).
		Radius(k.cardRadius).BorderWidth(k.hairline).BorderColor(k.border).
		Background(k.background).Shadow(0, 1, 4, 0, composerShadow).
		Children(func() {
			// The editor and the tool row keep the card's inset. The row is built
			// first so the keyboard order stays the one the shell contract
			// documents (Attach file, then the editor), while Reverse draws it at
			// the card's bottom, the editor first, as the spec lays it out.
			ui.Column(c).Padding(k.edgePad).Gap(k.gap).Children(func() {
				ui.Column(c).Gap(k.gap).Reverse().Children(func() {
					composerActions(c, k, v, a)
					composerEditor(c, k, v, a)
				})
			})
			// The divider and the context row are the card's last band, so they
			// run to the card's edges rather than to the editor's inset.
			composerDivider(c, k)
			composerContextRow(c, k, v)
		})
}

// composerWidth is the card's cap: the conversation's wider measure when the
// shell says a conversation is showing, and the workspace's narrower one in the
// empty state. Both stay under the column's width until it is narrower still.
func composerWidth(wide bool) float32 {
	if wide {
		return composerWideWidth
	}
	return composerEmptyWidth
}

// composerDivider is the hairline between the tool row and the context row. It
// spans the card, so it is drawn outside the editor's padded column.
func composerDivider(c *ui.Context, k tokens) {
	ui.Box(c).FillWidth().Height(k.hairline).Background(k.border).AccessHidden(true)
}

// composerContextRow draws the card's bottom band: the workspace the next turn
// would run in on the left, the session's default permission mode on the right,
// as dim-electron's composer does. Neither is a control here: there is no
// workspace or permission fact and no action to change one, so an empty field
// draws nothing beside its label and carries the reason it waits on in the
// group's description and tooltip rather than as visible WIP text. The band
// keeps the reference's ≈36 DIP height, so the card does not collapse to the
// text when both values are short.
func composerContextRow(c *ui.Context, k tokens, v ComposerView) {
	locale := v.Locale
	ui.Row(c).FillWidth().MinHeight(k.space(composerContextSpan)).
		PaddingX(k.edgePad).PaddingY(k.gap).Gap(k.gapWide).
		AlignItems(ui.Center).Children(func() {
		composerWorkspaceEntry(c, k, locale, v.Workspace)
		ui.Box(c).Grow(1)
		composerPermissionEntry(c, k, locale, v.Permission)
	})
}

// composerWorkspaceEntry draws the workspace's display: its icon, its label,
// the value the window supplied when it supplied one, and the disclosure
// chevron the reference's picker shows. It is a group, not a button: there is
// no action to open a picker yet, and it adds no Tab stop to the composer's
// documented order. With no value the entry reads as the reference's plain
// "Workspace ⌄" and the group's description and tooltip say why it is empty.
func composerWorkspaceEntry(c *ui.Context, k tokens, locale Locale, value string) {
	entry := ui.Row(c).Gap(k.gap).AlignItems(ui.Center).
		Role(ui.RoleGroup).Label(tr(locale, keyComposerWorkspace)).Children(func() {
		composerGlyph(c, k, sidebarIcons.folder)
		ui.Text(c, tr(locale, keyComposerWorkspace)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).Unselectable()
		composerContextValue(c, k, value)
		composerGlyph(c, k, composerGlyphs.chevronDown)
	})
	if value == "" {
		entry.Description(reasonWorkspace).Tooltip(reasonWorkspace)
	}
}

// composerPermissionEntry draws the session's default permission mode beside
// its shield, as the reference's status band does. Like the workspace it is a
// display group: the composer offers no control that would change the mode.
func composerPermissionEntry(c *ui.Context, k tokens, locale Locale, value string) {
	entry := ui.Row(c).Gap(k.gap).AlignItems(ui.Center).
		Role(ui.RoleGroup).Label(tr(locale, keyComposerPermission)).Children(func() {
		composerGlyph(c, k, composerGlyphs.shield)
		ui.Text(c, tr(locale, keyComposerPermission)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).Unselectable()
		composerContextValue(c, k, value)
	})
	if value == "" {
		entry.Description(reasonPermission).Tooltip(reasonPermission)
	}
}

// composerContextValue draws a context entry's value when the window supplied
// one. An empty value draws nothing: the entry keeps its muted label, and the
// group's description and tooltip explain the gap, so the card shows no WIP
// text the reference does not have and nothing that looks like a fact.
func composerContextValue(c *ui.Context, k tokens, value string) {
	if value == "" {
		return
	}
	ui.Text(c, value).FontSize(k.smallSize).TextColor(k.foreground).Unselectable()
}

// composerGlyph draws one 16 DIP decoration of the card, in the muted ink the
// reference uses for its toolbar and status glyphs. The row's label already
// names the entry beside it, so assistive technology skips the glyph.
func composerGlyph(c *ui.Context, k tokens, icon *ui.SVG) {
	ui.Icon(c, icon).Size(composerIconSize, composerIconSize).
		TextColor(k.mutedForeground).AccessHidden(true)
}

// composerEditor draws the draft field, as high as its wrapped text between
// the spec's 60 and 300 DIPs, scrolling inside past the cap. Enter submits, and
// Shift+Enter inserts a newline; a modified Enter stays for ComposerView's own
// shortcut, which the shell wires to Cmd/Ctrl+Enter. The shortcut pair is the
// field's description and key shortcut rather than a visible hint row, which
// the reference does not draw.
func composerEditor(c *ui.Context, k tokens, v ComposerView, a Actions) {
	ui.TextAreaBase(c.Key("composer"), v.Draft).Bind(v.Focus).
		Lines(1, composerMaxLines).MinHeight(composerMinH).MaxHeight(composerMaxH).
		Placeholder(v.Placeholder).Label(tr(v.Locale, keyComposerLabel)).
		Description(tr(v.Locale, keyComposerHint)).
		KeyShortcuts(0, composerSubmitKey).
		KeyShortcuts(v.Shortcut.Mods, v.Shortcut.Key).
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

// composerActions draws the tool row: the attachment, mode and model entries as
// icon buttons on the left and the round Send entry on the right, as the
// reference's composer toolbar does. No entry shows a visible WIP badge or a
// text label: the glyph names it, and its Label, Description and Tooltip carry
// the accessible name and the reason it cannot run yet.
func composerActions(c *ui.Context, k tokens, v ComposerView, a Actions) {
	locale := v.Locale
	ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		// The attachment entry keeps the first place in the Tab order, as the
		// shell's keyboard contract documents it: it stays focusable and reads
		// as unavailable, while a click reports nothing the App Server did not
		// do. The mode and model entries use Disabled instead: they are new
		// entries, and a tab stop would insert itself between "Attach file" and
		// the editor, changing that documented order.
		composerTool(c, k, sidebarIcons.plus, tr(locale, keyComposerAttach),
			tr(locale, keyReasonAttachment), true)
		composerTool(c, k, composerGlyphs.modeChevron, tr(locale, keyComposerMode),
			tr(locale, keyReasonMode), false)
		composerTool(c, k, composerGlyphs.cube, tr(locale, keyComposerModel),
			tr(locale, keyReasonModel), false)
		ui.Box(c).Grow(1)
		composerSend(c, k, v, a)
	})
}

// composerTool draws one toolbar glyph button: a square, borderless button that
// carries no action, because the shell has no attachment picker, agent mode or
// model list to open yet. Its accessible name is the localized label, and its
// Description and Tooltip say what it waits on. It keeps the reference's glyph
// weight rather than dimming to the old WIP opacity, so the toolbar still looks
// like the one it copies; the unavailable state is what assistive technology
// reads and every click is ignored. tabStop decides whether it keeps a place in
// the Tab order: only the attachment entry does, so the documented order does
// not change.
func composerTool(c *ui.Context, k tokens, icon *ui.SVG, label, reason string, tabStop bool) {
	size := k.space(composerToolSpan)
	button := ui.ButtonBase(c).Size(size, size).Radius(radiusSM).
		AlignItems(ui.Center).Justify(ui.Center).
		Label(label).Description(reason).Tooltip(reason)
	if tabStop {
		button.AccessDisabled(true)
	} else {
		button.Disabled(true).AccessDisabled(true)
	}
	button.Children(func() {
		composerGlyph(c, k, icon)
	})
}

// composerSend draws Send as the compact round button of the composer: a
// primary disc with the arrow decoration, named by the localized Send text. A
// draft that cannot be sent now leaves it disabled and unclickable but keeps
// the reference's solid primary face: the empty state's Send is the green disc
// the reference draws, not a faded one, and the disabled state is what
// assistive technology reads.
func composerSend(c *ui.Context, k tokens, v ComposerView, a Actions) {
	size := k.space(composerSendSpan)
	send := ui.ButtonBase(c).Label(tr(v.Locale, keyComposerSend)).
		Width(size).Height(size).Radius(size / 2).Padding(0).
		Background(k.primary).Children(func() {
		ui.Text(c, composerSendGlyph).FontSize(k.bodySize).TextColor(k.onPrimary).
			Unselectable().AccessHidden(true)
	}).OnClick(func() {
		submitDraft(v, a)
	})
	if !submitReady(v, a) {
		// Disabled takes the click and the Tab stop; DimDisabled(false) keeps the
		// reference's solid disc instead of mygo's half-opacity disabled face.
		send.Disabled(true).DimDisabled(false)
	}
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
