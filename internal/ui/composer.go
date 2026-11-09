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
	// composerMeterW and composerMeterH are the context meter's track, in DIPs:
	// the thin line the reference draws beside the branch. They are its h-1
	// w-10 at the 14px root font size, 3.5 × 35 DIPs (ContextUsageMeter.vue:30),
	// the same element the shell's status bar draws: a faint rail, not a
	// progress widget's body, so the status band stays one line.
	composerMeterW = 35
	composerMeterH = 3.5
	// composerMeterGap and composerMeterLabelW are the room the meter's reading
	// keeps beside its track, in DIPs: the reference's gap-1.5 between the rail
	// and the reading, and its min-w-7 cell (5.25 and 24.5 at the 14px root).
	// The reading is right-aligned in that cell, so the dash ends at the band's
	// inset no matter how many digits a known share prints.
	composerMeterGap    = 5.25
	composerMeterLabelW = 24.5
	// composerStatusH is the status band's height in a conversation, in DIPs:
	// the reference's h-10 at the 14px root (35), the room the card leaves above
	// the main column's bottom edge. It is a literal, not a spacing step,
	// because the reference measures the gap from its own frame.
	composerStatusH = 35
	// composerStatusPadX and composerStatusEntryPadX are the status band's
	// horizontal insets, in DIPs: the reference's px-5 on the footer (17.5 at
	// the 14px root) and its px-2 on each entry inside the band (7). The band
	// spans the main column, not the card, so in the reference's 800 DIP window
	// the permission shield's ink starts at x≈284 and the meter's dash ends at
	// x≈771 while the composer card itself runs from x≈266 to x≈785. See
	// docs/l5-reference.md §6.
	composerStatusPadX      = 17.5
	composerStatusEntryPadX = 7
	// composerStatusGap is the room between the band's entries, in DIPs. The
	// reference's row gap-1 is 3.5, but its branch label and brand mark are
	// slightly wider than this shell's 13 DIP type and 16 DIP glyph frame, so 6
	// keeps the branch's ink on the reference's x≈646 instead of drifting right
	// against the fixed meter.
	composerStatusGap = 6
)

// composerMeterDash is what the context meter shows while no usage fact is
// known. It is a dash rather than a zero or a filled track: an empty bar would
// read as "0 used", a number the composer has not been given.
const composerMeterDash = "—"

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
	// reasonContext explains the context meter while no token-usage fact is
	// known. The row draws "—" and keeps the number out rather than filling the
	// track with a guess.
	reasonContext = "Unavailable: ComposerView carries no context-usage fact; the shell has no App Server token-usage fact for the composer yet."
	// reasonBranch names what the branch entry is: the selected session's
	// branch, which the App Server owns, not the workspace's git branch. It is
	// not a gap, so it reads as a description rather than an unavailable note.
	reasonBranch = "The selected session's branch, as the App Server reports it; this is a session branch, not the workspace's git branch."
)

// composerBranchKey and composerContextUsageKey are the message keys for the
// branch entry's and the context meter's accessible names. i18n.go does not
// carry them yet; the composer owner reports them as needed entries, and each
// lookup falls back to its key until the table has the localized name. The
// branch key is deliberately not keySidebarBranches: the sidebar's list and the
// composer's single session branch are different things.
const (
	composerBranchKey       = "composer.branch"
	composerContextUsageKey = "composer.contextUsage"
)

// composerShadow is the card's soft outer shadow, the reference's low-alpha
// ring under the composer (ConversationComposer.vue:61-65). It is a black wash
// at about 6%, so it stays a lift in both appearances instead of a second
// border.
var composerShadow = ui.Hex("#00000010")

// composerGlyphs is the icon set the card's toolbar and context row draw, in
// the same 16×16 outline frame as the sidebar's (icons.go). The cube is the
// model entry's mark, the shield is the default permission mode's, the branch
// mark is the session branch's, the down chevron is the workspace picker's
// disclosure, and the double chevron is the mode entry's; they are drawn here
// because no other region uses them yet.
var composerGlyphs = struct {
	cube        *ui.SVG
	shield      *ui.SVG
	branch      *ui.SVG
	chevronDown *ui.SVG
	modeChevron *ui.SVG
}{
	cube:   sidebarIcon(`<path d="M8 1.8 13.2 4.9v6.2L8 14.2 2.8 11.1V4.9Z"/><path d="M2.8 4.9 8 8l5.2-3.1M8 8v6.2"/>`),
	shield: sidebarIcon(`<path d="M8 1.8 13 3.6v4.3c0 3.1-2 5.4-5 6.3-3-.9-5-3.2-5-6.3V3.6Z"/>`),
	// branch is the reference's branch mark: two stacked nodes on the left and
	// one offset node on the right, joined by the branch's rail. It is drawn
	// beside the selected session branch, not a git branch.
	branch: sidebarIcon(`<circle cx="4.4" cy="3.6" r="1.7"/><circle cx="4.4" cy="12.4" r="1.7"/><circle cx="11.6" cy="5.6" r="1.7"/><path d="M4.4 5.3v5.4M11.6 7.3c0 2.3-2.1 2.8-3.8 3.3"/>`),
	// chevronDown is the workspace picker's disclosure, the reference's ⌄
	// beside the workspace name rather than the sidebar's forward chevron.
	chevronDown: sidebarIcon(`<path d="M3.5 6 8 10.5 12.5 6"/>`),
	// modeChevron is the Mode entry's mark: the reference draws the agent-mode
	// picker as the » pair of chevrons, not as the single forward chevron the
	// sidebar uses for its disclosure. Two strokes, so the entry reads as a
	// choice rather than as a link.
	modeChevron: sidebarIcon(`<path d="M3.5 4 6.5 8l-3 4"/><path d="M8.5 4 11.5 8l-3 4"/>`),
}

// composer draws dim-electron's composer: the card with the editor and the tool
// row, and, in a conversation, the status band that sits under the card against
// the main column's bottom edge. The card is centred at the reference's
// measure. The workspace's empty state keeps the context row inside the card,
// as the reference draws it; a conversation moves the permission, branch and
// context entries out of the card, so the card no longer repeats the permission
// row. Send and the Enter key both submit the trimmed draft; the caller clears
// it once the App Server accepted the submission, so a failed send keeps the
// text.
func composer(c *ui.Context, k tokens, v ComposerView, a Actions) {
	if !v.Wide {
		ui.Column(c).FillWidth().MaxWidth(composerWidth(v.Wide)).AlignSelf(ui.Center).
			Children(func() {
				composerCard(c, k, v, a)
			})
		return
	}
	// A conversation's card keeps the reference's 520 DIP measure, while the
	// status band under it spans the main column, as the reference's footer
	// does: its px-5 inset is measured from the column's edge, not the card's,
	// so the band's entries sit further in than the card's own border.
	ui.Column(c).FillWidth().Children(func() {
		ui.Column(c).FillWidth().MaxWidth(composerWidth(v.Wide)).AlignSelf(ui.Center).
			Children(func() {
				composerCard(c, k, v, a)
			})
		composerStatusBand(c, k, v)
	})
}

// composerCard draws the surface the composer edits in: the editor over the
// tool row, and, in the empty state, the context row as the card's last band.
// The card is one group centred at the reference's measure, and it keeps
// everything it repeats inside its own border and rounded corners.
func composerCard(c *ui.Context, k tokens, v ComposerView, a Actions) {
	ui.Column(c).Role(ui.RoleGroup).FillWidth().
		Radius(k.cardRadius).BorderWidth(k.hairline).BorderColor(k.border).
		Background(k.background).Shadow(0, 1, 4, 0, composerShadow).
		Transition(motionLayoutColors(c)).
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
			if v.Wide {
				// A conversation's context entries sit in the status band under
				// the card, not inside it.
				return
			}
			// The divider and the context row are the card's last band, so they
			// run to the card's edges rather than to the editor's inset.
			composerDivider(c, k)
			composerContextRow(c, k, v)
		})
}

// composerStatusBand draws the conversation's status under the card: the
// default permission mode on the left and the selected session's branch with
// the context meter on the right, as the reference's status band lays them out.
// It is outside the card and flush with the column's bottom, and spans the
// main column rather than the card's 520 DIP measure, so it keeps only the
// editor and the tool row above it. The workspace entry stays in the empty
// state's card: the reference's conversation band shows no workspace.
func composerStatusBand(c *ui.Context, k tokens, v ComposerView) {
	locale := v.Locale
	ui.Row(c).FillWidth().Height(composerStatusH).
		Padding(0, composerStatusPadX, 0, composerStatusPadX).
		Gap(composerStatusGap).AlignItems(ui.Center).
		Role(ui.RoleGroup).Label(tr(locale, keyComposerStatus)).Children(func() {
		composerPermissionEntry(c, k, locale, v.Permission, composerStatusEntryPadX)
		ui.Box(c).Grow(1)
		composerBranchEntry(c, k, locale, v.Branch, composerStatusEntryPadX)
		composerContextMeter(c, k, locale, v.ContextKnown, v.ContextUsed, composerStatusEntryPadX, true)
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
	ui.Box(c).FillWidth().Height(k.hairline).Background(k.border).Transition(motionLayoutColors(c)).AccessHidden(true)
}

// composerContextRow draws the empty state's card bottom band: the workspace
// the next turn would run in on the left, then, on the right, the session's
// default permission mode, the selected session's branch and the context-usage
// meter. A conversation moves the entries into composerStatusBand under the
// card, as the reference does. None of them is a control here: an empty field
// draws nothing beside its label and carries the reason it waits on in the
// group's description and tooltip rather than as visible WIP text. The band
// keeps the reference's ≈36 DIP height, so the card does not collapse to the
// text when every value is short.
func composerContextRow(c *ui.Context, k tokens, v ComposerView) {
	locale := v.Locale
	ui.Row(c).FillWidth().MinHeight(k.space(composerContextSpan)).
		PaddingX(k.edgePad).PaddingY(k.gap).Gap(k.gapWide).
		AlignItems(ui.Center).Children(func() {
		composerWorkspaceEntry(c, k, locale, v.Workspace)
		ui.Box(c).Grow(1)
		composerPermissionEntry(c, k, locale, v.Permission, 0)
		composerBranchEntry(c, k, locale, v.Branch, 0)
		composerContextMeter(c, k, locale, v.ContextKnown, v.ContextUsed, 0, false)
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
// padX is the entry's own horizontal inset: the status band's entries keep the
// reference's px-2 (7 DIPs) inside the band, while the empty state's card row
// already sits on the card's inset and passes none.
func composerPermissionEntry(c *ui.Context, k tokens, locale Locale, value string, padX float32) {
	entry := ui.Row(c).Gap(k.gap).AlignItems(ui.Center).PaddingX(padX).
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

// composerBranchEntry draws the selected session's branch beside its branch
// mark, as the reference's status band draws "main". The reference's value is
// the workspace's git branch; this shell has no git fact, so the entry shows
// the session branch the App Server reports and says so in its description. An
// empty branch draws no entry at all rather than the reference's "main", which
// would be a git branch the shell never read. padX is the band entry's inset,
// as above.
func composerBranchEntry(c *ui.Context, k tokens, locale Locale, branch string, padX float32) {
	if branch == "" {
		return
	}
	ui.Row(c).Gap(k.gap).AlignItems(ui.Center).PaddingX(padX).
		Role(ui.RoleGroup).Label(tr(locale, composerBranchKey)).
		Description(reasonBranch).Tooltip(reasonBranch).Children(func() {
		composerGlyph(c, k, composerGlyphs.branch)
		ui.Text(c, branch).FontSize(k.smallSize).TextColor(k.foreground).Unselectable().SingleLine()
	})
}

// composerContextMeter draws the context-usage meter: the reference's thin
// rail, filled to the known share, and the reading's cell beside it. With rail
// set — the conversation's status band — an unknown share still draws the empty
// rail and the reference's dash in a right-aligned cell, so the band reads as a
// meter waiting for a fact. The empty state's card row keeps only the dash,
// because the reference's NewThread row has no meter there. The dash is a
// decoration and stays out of the accessible tree; the group's description says
// what the entry waits on, and no percentage is drawn until the shell formats
// one. used is a fraction of the context window in 0..1 and is read only when
// known. padX is the band entry's inset, as above.
func composerContextMeter(c *ui.Context, k tokens, locale Locale, known bool, used float32, padX float32, rail bool) {
	entry := ui.Row(c).Gap(composerMeterGap).AlignItems(ui.Center).PaddingX(padX).
		Role(ui.RoleGroup).Label(tr(locale, composerContextUsageKey))
	if !known {
		entry.Description(reasonContext).Tooltip(reasonContext).Children(func() {
			if rail {
				composerMeterRail(c, k)
				ui.Text(c, composerMeterDash).FontSize(k.smallSize).
					TextColor(k.mutedForeground).Width(composerMeterLabelW).TextAlign(ui.End).
					Unselectable().AccessHidden(true)
				return
			}
			ui.Text(c, composerMeterDash).FontSize(k.smallSize).
				TextColor(k.mutedForeground).Unselectable().AccessHidden(true)
		})
		return
	}
	fill := composerMeterW * clampShare(used)
	entry.Children(func() {
		ui.Box(c).Width(composerMeterW).Height(composerMeterH).
			Radius(composerMeterH / 2).Background(k.muted).Clip().Children(func() {
			if fill <= 0 {
				return
			}
			ui.Box(c).Width(fill).Height(composerMeterH).Background(k.primary)
		})
	})
}

// composerMeterRail draws the meter's empty track: the reference's h-1 w-10
// rail in the muted face, which window-conversation.png measures at x 706.5–741
// and 3.5 DIPs high beside the dash. It is the rail alone, so an unknown share
// reads as "no reading yet" rather than as a zero-filled bar.
func composerMeterRail(c *ui.Context, k tokens) {
	ui.Box(c).Width(composerMeterW).Height(composerMeterH).
		Radius(composerMeterH / 2).Background(k.muted)
}

// clampShare keeps a meter's fill inside its track: a share below zero reads as
// empty and one above one as full, so a bad caller cannot draw past the rail.
func clampShare(share float32) float32 {
	switch {
	case share < 0:
		return 0
	case share > 1:
		return 1
	default:
		return share
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
		// No model is configured because the shell has no model fact to show.
		// The reference puts the muted "未配置" label beside the send disc, and
		// this text states that absence rather than inventing a model name.
		ui.Text(c, tr(locale, keyComposerUnconfigured)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).SingleLine().Unselectable().
			Description(tr(locale, keyReasonModel))
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
	button := ui.ButtonBase(c).Key("composer-tool:"+label).Size(size, size).Radius(radiusSM).
		AlignItems(ui.Center).Justify(ui.Center).
		Label(label).Description(reason).Tooltip(reason).Transition(motionFastColors(c))
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
	send := ui.ButtonBase(c).Key("composer-send").Label(tr(v.Locale, keyComposerSend)).
		Width(size).Height(size).Radius(size / 2).Padding(0).
		Background(k.primary).Transition(motionFastColors(c)).Children(func() {
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
