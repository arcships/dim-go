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
	// composerModelChipPadX and composerModelChipPadY are the model chip's
	// inset in theme spacing units: the chip is a muted pill on the tool row,
	// and its padding keeps the label's ink on the reference's x, where
	// "未配置" runs into the right side of the card a few DIPs before the send
	// disc (window-conversation.png: the label ends at x≈741 and the disc
	// starts at x≈745 in the 800 DIP window).
	composerModelChipPadX = 1
	composerModelChipPadY = 0.5
	// composerModelChipMaxW bounds the chip when it carries a capability reason:
	// the reachable models still name the current pair, and a degraded read shows
	// its reason as one truncated line with the full text in the tooltip and the
	// model list.
	composerModelChipMaxW = 300
	// composerSendSpinner is the in-flight indicator's square in DIPs, drawn
	// inside the reference's 28 DIP disc: one toolbar glyph's 16 DIP frame
	// would crowd the disc's rim, so the spinner keeps the arrow's optical
	// size instead.
	composerSendSpinner = 14
	// composerSendHoverShare is how far the Send disc's face moves under the
	// pointer: the reference's density/hover step for its primary button, a
	// slight darkening in the light theme and a slight lightening in the dark
	// one, so the disc still reads as the same solid primary.
	composerSendHoverShare = 0.12
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
// model toolbar glyph has no selection action and the agent mode has no fact in
// ComposerView, so both stay visible, disabled and explained rather than
// inventing a selection. The model chip is the selection surface: it shows the
// current App Server pair and opens the model list.
const (
	reasonModel = "Unavailable: this toolbar entry has no model picker; the chip beside Send shows the current model and opens the list, where a model can be selected."
	reasonMode  = "Unavailable: ComposerView carries no agent mode; the shell exposes no agent-mode fact to the composer yet."
	// reasonAttachmentReady describes the attachment entry once the window wires
	// Actions.OnAttachFile: the picker takes a text file and the App Server
	// stores it, so a link can be added to the draft. Images are not offered
	// yet; they have no upload method and would need an inline image block.
	reasonAttachmentReady = "Add a text file: the window opens the native picker and the App Server stores the chosen file, then its link is added to the draft. Image attachments are WIP."
	// reasonModelPair explains an absent current model: the attach reply
	// reported no complete provider/model pair, so the chip shows the localized
	// unconfigured label rather than naming one.
	reasonModelPair = "Unavailable: the App Server reported no current provider/model pair for this session."
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

// composerBranchKey, composerContextUsageKey and composerSendBusyKey are the
// message keys for the branch entry's and the context meter's accessible names
// and for the in-flight Send's description. i18n.go does not carry them yet;
// the composer owner reports them as needed entries, and each lookup falls back
// to its key until the table has the localized name. The branch key is
// deliberately not keySidebarBranches: the sidebar's list and the composer's
// single session branch are different things. The busy key tells assistive
// technology what a disabled, in-flight Send is doing; the disc keeps the
// localized Send name.
const (
	composerBranchKey       = "composer.branch"
	composerContextUsageKey = "composer.contextUsage"
	composerSendBusyKey     = "composer.busy"
)

// composerGlyphs is the icon set the card's toolbar and context row draw, in
// the same Lucide frame as the sidebar's (lucide_icons.go). The box is the
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
	cube:   mustLucide("box"),
	shield: mustLucide("shield"),
	// branch is the reference's branch mark: two stacked nodes and one offset
	// node, joined by the branch's rail. It is drawn beside the selected
	// session branch, not a git branch.
	branch: mustLucide("git-branch"),
	// chevronDown is the workspace picker's disclosure, the reference's ⌄
	// beside the workspace name rather than the sidebar's forward chevron.
	chevronDown: mustLucide("chevron-down"),
	// modeChevron is the Mode entry's mark: the reference draws the agent-mode
	// picker as the » pair of chevrons, not as the single forward chevron the
	// sidebar uses for its disclosure. Two strokes, so the entry reads as a
	// choice rather than as a link.
	modeChevron: mustLucide("chevrons-right"),
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
	} else {
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
	composerModelDialog(c, k, v, a)
}

// composerCard draws the surface the composer edits in: the editor over the
// tool row, and, in the empty state, the context row as the card's last band.
// The card is one group centred at the reference's measure, and it keeps
// everything it repeats inside its own border and rounded corners.
func composerCard(c *ui.Context, k tokens, v ComposerView, a Actions) {
	// The card lifts by the card shadow step: the reference rings the
	// composer with a 1 DIP low-alpha edge (ConversationComposer.vue:63), which
	// the shell draws as the same soft, downward shadow a card carries, so the
	// border stays the edge and the shadow only reads as a lift off the column.
	card := ui.Column(c).Role(ui.RoleGroup).FillWidth().
		Radius(k.cardRadius).BorderWidth(k.hairline).BorderColor(k.border).
		Background(k.background).Transition(motionLayoutColors(c))
	k.shadowCard.paint(card).
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
// the accessible name and either the action it starts or the reason it cannot
// run yet.
func composerActions(c *ui.Context, k tokens, v ComposerView, a Actions) {
	locale := v.Locale
	ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		// The attachment entry keeps the first place in the Tab order, as the
		// shell's keyboard contract documents it. With Actions.OnAttachFile wired
		// it is a real, keyboard-reachable button; without it, it stays visible
		// and focusable but reads as unavailable, and a click reports nothing the
		// App Server did not do. The mode and model entries use Disabled instead:
		// they are new entries, and a tab stop would insert itself between
		// "Attach file" and the editor, changing that documented order.
		attachReason := tr(locale, keyReasonAttachment)
		if a.OnAttachFile != nil {
			attachReason = tr(locale, keyReasonAttachmentReady)
		}
		composerTool(c, k, sidebarIcons.plus, tr(locale, keyComposerAttach),
			attachReason, true, a.OnAttachFile)
		composerTool(c, k, composerGlyphs.modeChevron, tr(locale, keyComposerMode),
			tr(locale, keyReasonMode), false, nil)
		composerTool(c, k, composerGlyphs.cube, tr(locale, keyComposerModel),
			tr(locale, keyReasonModel), false, nil)
		ui.Box(c).Grow(1)
		// The chip shows the current App Server model pair, or the localized
		// unconfigured label with the read's reason. It opens the model list,
		// where a selectable row calls the caller's OnSelectModel; nothing here
		// invents a model or stores a selection.
		composerModelChip(c, k, v, a)
		composerSend(c, k, v, a)
	})
}

// composerModelChip draws the model status beside Send: the current
// provider/model the App Server reported, or the reference's muted "未配置"
// while no complete pair is known. The chip is a button that opens the model
// list through Actions.OnToggleModelList; a caller without that action keeps it
// visible and unavailable with the reason it waits on. It sits left of the send
// disc, where the reference draws the label, and the tool row's gap keeps that
// distance.
func composerModelChip(c *ui.Context, k tokens, v ComposerView, a Actions) {
	label := v.ModelLabel
	reason := v.ModelReason
	if label == "" {
		// No current pair: the chip states the read's reason when there is one,
		// so a missing capability is visible on the frame and not only in the
		// list. A session whose attach reply carried no pair keeps the
		// reference's unconfigured label.
		label = reason
		if label == "" {
			label = tr(v.Locale, keyComposerUnconfigured)
			reason = reasonModelPair
		}
	}
	chip := ui.ButtonBase(c).Key("composer-model").Padding(k.space(composerModelChipPadY), k.space(composerModelChipPadX)).
		Radius(k.radiusSM).Background(k.muted).AlignItems(ui.Center).MaxWidth(composerModelChipMaxW).
		Role(ui.RoleButton).Label(tr(v.Locale, keyComposerModelChip)).
		Transition(motionFastColors(c))
	if reason != "" {
		chip.Description(reason).Tooltip(reason)
	}
	if a.OnToggleModelList != nil {
		chip.OnClick(a.OnToggleModelList)
		if chip.Hovered() {
			chip.Background(k.muted)
		}
	} else {
		chip.AccessDisabled(true)
	}
	chip.Children(func() {
		ui.Text(c, label).FontSize(k.smallSize).TextColor(k.mutedForeground).SingleLine().Ellipsis("…").Unselectable()
	})
}

// composerModelDialog draws the model list over the window while
// ComposerView.ModelOpen is set: the current pair and, for every model the
// catalog carries, its provider/model name, availability and the reason it is
// unavailable. A row is selectable while Actions.OnSelectModel is wired; the
// row closes the list first and then calls it once, so one click sends at most
// one request. A caller that wires no selection keeps every row visible,
// disabled and explained. Escape and a click outside the dialog close it through
// DialogBase, so no request is sent.
func composerModelDialog(c *ui.Context, k tokens, v ComposerView, a Actions) {
	if v.ModelOpen == nil || !*v.ModelOpen {
		return
	}
	ui.DialogBase(c, v.ModelOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(renameScrim)
		panel = panel.Width(renameDialogMaxW).Padding(k.cardPadY, k.cardPadX).Gap(k.gap).
			Radius(k.cardRadius).Background(k.background).
			BorderWidth(k.hairline).BorderColor(k.border).MaxHeightPercent(80).Clip()
		// The list is a dialog over a dimmed window, so it lifts by the dialog
		// step, as the reference's DialogContent does with shadow-lg.
		panel = k.shadowDialog.paint(panel)
		panel.Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, tr(v.Locale, keyModelTitle)).Bold().FontSize(k.titleSize).TextColor(k.foreground)
				ui.Box(c).Grow(1)
				ui.Text(c, v.ModelLabel).FontSize(k.smallSize).TextColor(k.mutedForeground).SingleLine()
			})
			if v.ModelReason != "" {
				ui.Text(c, v.ModelReason).FontSize(k.smallSize).TextColor(k.danger)
			}
			if len(v.ModelChoices) == 0 {
				ui.Text(c, tr(v.Locale, keyModelEmpty)).FontSize(k.smallSize).TextColor(k.mutedForeground)
			}
			ui.Column(c).FillWidth().Gap(k.gap).Children(func() {
				for _, choice := range v.ModelChoices {
					modelChoiceRow(c, k, v, a, choice)
				}
			})
			if a.OnSelectModel == nil {
				ui.Text(c, tr(v.Locale, keyModelReadOnly)).FontSize(k.smallSize).TextColor(k.mutedForeground)
			}
			ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
				ui.Button(c, tr(v.Locale, keyModelClose)).OnClick(func() {
					*v.ModelOpen = false
				}).Transition(motionFastColors(c))
			})
		})
	})
}

// modelChoiceRow draws one model: its provider and model names, the current
// marker, and its availability or the reason it is not selectable. With
// Actions.OnSelectModel wired and the model available the row is a button that
// closes the list and then calls the action once; an unavailable model stays a
// disabled row with its App Server reason. A caller without the action keeps
// every row disabled, because a click would pretend a selection it cannot
// write.
func modelChoiceRow(c *ui.Context, k tokens, v ComposerView, a Actions, choice ModelChoice) {
	locale := v.Locale
	name := choice.DisplayName
	if name == "" {
		name = choice.ModelID
	}
	if name == "" {
		name = choice.ProviderID
	}
	selectable := a.OnSelectModel != nil && choice.Enabled && choice.Reason == ""
	row := ui.ButtonBase(c).Key("composer-model:"+choice.ProviderID+"/"+choice.ModelID).
		FillWidth().Padding(k.gap, k.edgePad).Radius(k.radiusSM).
		Background(k.muted).AlignItems(ui.Center).
		Role(ui.RoleButton).Label(name).Selected(choice.Current)
	if selectable {
		row.OnClick(func() {
			// Close first: the dialog is gone before the request starts, so a
			// second click cannot send the same selection twice.
			if v.ModelOpen != nil {
				*v.ModelOpen = false
			}
			a.OnSelectModel(choice.ProviderID, choice.ModelID)
		})
		if row.Hovered() {
			row.Background(k.sidebarHover)
		}
	} else {
		row.AccessDisabled(true)
		if !choice.Enabled || choice.Reason != "" {
			reason := tr(locale, keyModelUnavailable)
			if choice.Reason != "" {
				reason += ": " + choice.Reason
			}
			row.Description(reason).Tooltip(reason)
		}
	}
	row.Children(func() {
		ui.Column(c).FillWidth().Children(func() {
			ui.Row(c).FillWidth().Gap(k.gap).AlignItems(ui.Center).Children(func() {
				ui.Text(c, name).FontSize(k.bodySize).TextColor(k.foreground).SingleLine()
				ui.Box(c).Grow(1)
				if choice.Current {
					ui.Text(c, tr(locale, keyModelCurrent)).FontSize(k.smallSize).TextColor(k.mutedForeground)
				}
			})
			ui.Text(c, choice.ProviderID+"/"+choice.ModelID).FontSize(k.smallSize).TextColor(k.mutedForeground).SingleLine()
			if choice.Enabled && choice.Reason == "" {
				ui.Text(c, tr(locale, keyModelAvailable)).FontSize(k.smallSize).TextColor(k.mutedForeground)
				return
			}
			reason := tr(locale, keyModelUnavailable)
			if choice.Reason != "" {
				reason += ": " + choice.Reason
			}
			ui.Text(c, reason).FontSize(k.smallSize).TextColor(k.danger)
		})
	})
}

// composerTool draws one toolbar glyph button: a square, borderless button
// whose accessible name is the localized label and whose Description and
// Tooltip carry either the action it starts or the reason it waits. A nil
// onClick leaves it available to the keyboard but unavailable to assistive
// technology: the click is ignored, so the region never reports a result the
// App Server did not produce. A wired onClick makes it a real button. It keeps
// the reference's glyph weight rather than dimming to the old WIP opacity, so
// the toolbar still looks like the one it copies. tabStop decides whether it
// keeps a place in the Tab order: only the attachment entry does, so the
// documented order does not change.
func composerTool(c *ui.Context, k tokens, icon *ui.SVG, label, reason string, tabStop bool, onClick func()) {
	size := k.space(composerToolSpan)
	button := ui.ButtonBase(c).Key("composer-tool:"+label).Size(size, size).Radius(k.radiusSM).
		AlignItems(ui.Center).Justify(ui.Center).
		Label(label).Description(reason).Tooltip(reason).Transition(motionFastColors(c))
	if onClick != nil {
		button.OnClick(onClick)
		if button.Hovered() {
			button.Background(k.muted)
		}
	} else if tabStop {
		button.AccessDisabled(true)
	} else {
		button.Disabled(true).AccessDisabled(true)
	}
	button.Children(func() {
		composerGlyph(c, k, icon)
	})
}

// composerSend draws Send as the compact round button of the composer: a
// primary disc around the arrow decoration, named by the localized Send text.
// It has three faces, all of them the reference's solid primary disc rather
// than a faded one:
//
//   - a draft that cannot be sent (empty or whitespace-only text, no submit
//     gate) keeps the solid face and takes no hover, so the disc still matches
//     window-conversation.png while it is disabled;
//   - a draft that can be sent keeps the same face and darkens slightly under
//     the pointer, which is the reference's hover step for its primary button;
//   - a submission in flight swaps the arrow for a spinner and stays disabled,
//     so the row shows work rather than a second send.
//
// The disabled state is what assistive technology reads; DimDisabled(false)
// keeps the disc solid instead of mygo's half-opacity disabled face.
func composerSend(c *ui.Context, k tokens, v ComposerView, a Actions) {
	size := k.space(composerSendSpan)
	ready := submitReady(v, a)
	send := ui.ButtonBase(c).Key("composer-send").Label(tr(v.Locale, keyComposerSend)).
		Width(size).Height(size).Radius(size / 2).Padding(0).
		Background(k.primary).Transition(motionFastColors(c)).Children(func() {
		if v.Sending {
			// A button's children are not separate nodes in the accessible
			// tree, so the spinner is a decoration: the disc itself carries the
			// busy state and the reason for assistive technology.
			ui.Spinner(c).Size(composerSendSpinner, composerSendSpinner).
				TextColor(k.onPrimary).AccessHidden(true)
			return
		}
		ui.Text(c, composerSendGlyph).FontSize(k.bodySize).TextColor(k.onPrimary).
			Unselectable().AccessHidden(true)
	}).OnClick(func() {
		submitDraft(v, a)
	})
	switch {
	case v.Sending:
		send.Disabled(true).DimDisabled(false).Busy(true).
			Description(tr(v.Locale, composerSendBusyKey))
	case !ready:
		send.Disabled(true).DimDisabled(false)
	case send.Hovered():
		// Only a disc that can be pressed answers the pointer: a disabled or
		// in-flight Send keeps the face the reference draws.
		send.Background(composerSendHover(k))
	}
}

// composerSendHover is the Send disc's face under the pointer: the solid
// primary moved one slight step, darker in the light theme and lighter in the
// dark one, so the reference's hover reads as a shade of the same button
// rather than a second color. It starts from the primary fill, not a theme
// field, so a theme with its own brand still gets the same step.
func composerSendHover(k tokens) ui.Color {
	step := ui.Hex("#000000")
	if k.theme != nil && k.theme.Dark {
		step = ui.Hex("#ffffff")
	}
	return k.primary.Mix(step, composerSendHoverShare)
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
