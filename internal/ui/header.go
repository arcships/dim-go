// This file owns the main area's conversation bar and banner: the project and
// session-title breadcrumb, the session menu, the conditional Stop and Reconnect
// entries, the context panel toggle, the delete confirmation, the find bar's
// place at the bar's trailing end and the error banner below them. It reads
// HeaderView and Actions only (see protocols.go). The find bar itself is
// thread_search.go's nested region, drawn through renderThreadSearch.
//
// The reference's conversation bar is one line with the breadcrumb and a "…"
// menu on the left and the surface entries on the right. The bar carries a
// compact current-model chip between the menu and the panel toggle: it names
// the pair the App Server reported and opens the same model list the composer's
// chip opens, through the same Actions.OnToggleModelList. Its editor entry is
// absent: the shell has no editor client or workspace fact, so the mark, its
// disclosure arrow and the rule before the panel toggle would only pretend to
// open something. Run and connection status are not drawn here either: the
// reference has no "Session: Idle" or "App Server: …" row, and this shell no
// longer adds one.

package ui

import (
	"net/url"

	"github.com/egoist/mygo/ui"
)

// contextPanelToggleLabel is the toolbar's context panel entry: its accessible
// name, its tooltip, and the text the tests and assistive technology look for.
// The panel's own headings ("Changes", "No changed files") stay English until
// the panel takes a Locale like the other regions.
const contextPanelToggleLabel = "Context panel"

// breadcrumbSeparator sits between the project name and the session title, as
// the reference draws it. It is punctuation, not a translatable word; an empty
// project draws neither it nor a project name.
const breadcrumbSeparator = "/"

// breadcrumbGap is the room around the separator, in DIPs: the reference draws
// the project, the slash and the title as one run of adjacent text, so the
// breadcrumb keeps one tight step between them rather than the toolbar's gap.
const breadcrumbGap = 4

// breadcrumbProjectMaxW is the room the reference gives the project name before
// it truncates, in DIPs. Its span is max-w-[8rem] and shrinks inside the
// breadcrumb's flex row; at an 800 DIP window the flex row leaves it about 85
// DIPs, so window-conversation.png draws "dimagent-m…" with its last dot at
// x≈355 and starts the title's ink at x≈375, 15 DIPs right of where the whole
// name would leave it. The shell has no flex shrink in the bar, so it keeps the
// measured width as a literal rather than letting the full project name push
// the title left.
const breadcrumbProjectMaxW = 85

// titleEllipsisOverlap is the room the title's text takes past its own box, in
// DIPs. The bar's title ends where the session menu's target begins, and the
// menu keeps its 20 DIP target and glyph position; the reference's title still
// fits "one par…" in that room, while this shell's wider bold raster stops at
// "one pa…". The menu's target carries about 5 DIPs of bearing before its "…",
// so the title's text may run under that empty bearing: at 4.5 the reference's
// last glyph fits and the truncation lands where the reference draws it,
// without moving the menu or any control to its right. The 4 DIP estimate was
// the gap below the threshold; half a DIP more fits the "r".
const titleEllipsisOverlap = 4.5

// breadcrumbLeftPad is the bar's leading inset, in DIPs: the reference's
// ChatHeaderFrame keeps px-3 on the header and mx-3 on the breadcrumb's flex
// row (10.5 + 10.5 at the 14px root), so window-conversation.png starts the
// project's ink at x≈277.5, about 21 inside the main card's edge and 3.5 DIPs
// left of the conversation's own 24.5 DIP content inset. The shell's frame puts
// the bar's content edge at x≈258, so 19 keeps the project where the reference
// draws it while the 85 DIP cap keeps the title at x≈375.
const breadcrumbLeftPad = 19

// menuTargetW is the session menu's target width, in DIPs: a compact step that
// keeps the "…" close to the truncated title instead of pushing it into the
// toolbar's full 28 DIP square. The target keeps the toolbar's height.
const menuTargetW = 20

// The header model chip's geometry. Its inset is one tight step of the theme's
// spacing and its height comes from the label's line plus that inset, so it
// stays a compact pill beside the panel toggle rather than a second toolbar
// button. headerModelMaxW bounds the label when the provider/model pair is
// long: the chip truncates its text with an ellipsis and shrinks inside the
// bar's trailing group instead of pushing the breadcrumb, the session menu or
// the panel toggle aside. It is a display measure, not a model fact.
const (
	headerModelPadX = 2
	headerModelPadY = 1
	headerModelMaxW = 160
)

// contextPanelGlyph is the entry's two-pane mark, as the Electron reference
// draws the panel toggle at the card's top right: Lucide's panel-right, a
// rounded frame with its split line separating the conversation from the
// context column. It strokes currentColor and paints no fill, so it follows the
// theme like the sidebar's glyphs.
var contextPanelGlyph = mustLucide("panel-right")

// reasonOpenInEditor is the explanation the localization table still carries
// for the reference's editor-open entry. The bar no longer draws that entry:
// with no editor client and no workspace fact, the mark, its arrow and the rule
// before the panel toggle were a fake entry rather than a usable one. The text
// stays so a real editor client can bring the entry back without inventing a
// new reason.
const reasonOpenInEditor = "Unavailable: the shell has no editor-open client or workspace fact for opening this session in VS Code yet."

// reasonHeaderModel explains the bar's model chip while the window wired no
// model-list action. The chip keeps its place and its name instead of
// disappearing, so the missing action is visible rather than silent; a window
// that wires Actions.OnToggleModelList never draws this reason. It is the
// window's own text, not a wire error, and stays English until i18n.go carries
// a key for it.
const reasonHeaderModel = "Unavailable: the window has no model-list action, so the model list cannot open."

// menuRowH is a session-menu item's height in the drawn menu, in DIPs: the
// reference's 28 DIP step. The menu adds Space(1.5) above and below the label
// line, so menuRowTextH is the line box that leaves the row at that height; the
// theme's own 15.3 DIP line would make it 27.3.
const (
	menuRowH     = 28
	menuRowTextH = 16
)

// The session menu's entries the App Server does not offer yet. The menu draws
// them as WIP so the reference's structure is visible without pretending an
// action the shell cannot carry out. The three copy entries are not among them:
// the id is a fact the App Server already reported, the link is that id in the
// scheme the app registers, and the debug text is built from the model facts the
// window displays, so all three are copied locally. Only their empty states are
// WIP: no selected identity for the link and the id, and no current model pair
// for the debug text. The message entries belong in i18n.go beside the other
// header and reason keys; until that table carries them, menuText falls back to
// the English constants below rather than drawing a raw key, and the fallback
// goes away with the table entries.
const (
	keyHeaderMenuOpenInNewWindow = "header.menu.openInNewWindow"
	keyHeaderMenuExport          = "header.menu.export"
	keyHeaderMenuCopyLink        = "header.menu.copyLink"
	keyHeaderMenuCopyID          = "header.menu.copyId"
	keyHeaderMenuCopyDebug       = "header.menu.copyDebug"

	keyReasonOpenInNewWindow = "reason.openInNewWindow"
	keyReasonExport          = "reason.export"
	keyReasonCopyLink        = "reason.copyLink"
	keyReasonCopyID          = "reason.copyId"
	keyReasonCopyDebug       = "reason.copyDebug"
)

// The English text of the entries above: the labels the reference's menu draws
// and the reason each one waits on. Pin, Rename and Archive already have their
// text in i18n.go; only these entries and their reasons are new. The link's and
// the id's reasons are what their no-identity state says: both entries are local
// actions now, so their reasons name the only gap left.
const (
	reasonOpenInNewWindow = "Unavailable: the shell has no second-window client for opening a session yet."
	reasonExport          = "Unavailable: the shell has no conversation export client or App Server export method yet."
	reasonCopyLink        = "Unavailable: no session identity is selected, so there is no conversation link to copy."
	reasonCopyID          = "Unavailable: no session identity is selected, so there is no conversation id to copy."
	reasonCopyDebug       = "Unavailable: no model facts are selected, so there is no runtime config to copy."
)

// copyIDStatus is the muted line the notice strip shows after the menu copied
// the selected session's id. It is the window's own outcome, not an App Server
// fact: the region writes the clipboard and tells the shell through
// Actions.OnCopySessionID, which records this line and sends nothing. It stays
// an English constant until i18n.go carries a key for it.
const copyIDStatus = "Copied conversation ID"

// copyLinkStatus is the muted line the notice strip shows after the menu copied
// the selected session's conversation link. Like copyIDStatus it is the window's
// own outcome: the region writes the clipboard and tells the shell through
// Actions.OnCopyConversationLink, which records this line and sends nothing. It
// stays an English constant until i18n.go carries a key for it.
const copyLinkStatus = "Copied conversation link"

// ConversationLinkScheme is the URL scheme the app registers for its
// conversation deep links. cmd/dimagent registers this scheme with the system
// and parses the same shape back, so the menu and the handler cannot disagree
// about what a link is.
const ConversationLinkScheme = "dimagent"

// ConversationLinkHost is the route that names a conversation in a deep link.
// Any other host is a different entry point and is not a conversation link.
const ConversationLinkHost = "thread"

// ConversationLink is the deep link that opens one session. The id is an opaque
// App Server fact, so it is escaped as one path segment and the link stays a
// single token the system hands back as a URL. An empty id is not a link; the
// menu keeps the entry as its WIP statement instead.
func ConversationLink(sessionID string) string {
	return ConversationLinkScheme + "://" + ConversationLinkHost + "/" + url.PathEscape(sessionID)
}

// copyDebugStatus is the muted line the notice strip shows after the menu
// copied the model's public runtime facts. Like copyIDStatus it is the window's
// own outcome: the region writes the clipboard and tells the shell through
// Actions.OnCopyDebugInfo, which records this line and sends nothing. It stays
// an English constant until i18n.go carries a key for it.
const copyDebugStatus = "Copied debug info"

// headerMenuFallbacks is the English text of the menu keys i18n.go does not
// carry yet. It is a stand-in for the table entries, not a second language
// table: menuText consults it only when tr reports a missing key.
var headerMenuFallbacks = map[string]string{
	keyHeaderMenuOpenInNewWindow: "Open in new window",
	keyHeaderMenuExport:          "Export",
	keyHeaderMenuCopyLink:        "Copy conversation link",
	keyHeaderMenuCopyID:          "Copy conversation ID",
	keyHeaderMenuCopyDebug:       "Copy debug info",
	keyReasonOpenInNewWindow:     reasonOpenInNewWindow,
	keyReasonExport:              reasonExport,
	keyReasonCopyLink:            reasonCopyLink,
	keyReasonCopyID:              reasonCopyID,
	keyReasonCopyDebug:           reasonCopyDebug,
}

// menuText resolves one session-menu label or reason. The lookup goes through
// the locale table first, so a translated entry wins as soon as i18n.go carries
// the key; a missing entry draws the English fallback instead of the key.
func menuText(locale Locale, key string) string {
	if text := tr(locale, key); text != key {
		return text
	}
	if text, ok := headerMenuFallbacks[key]; ok {
		return text
	}
	return key
}

// header shows the selected session's breadcrumb and entries, then the banner
// that belongs under them. It draws no region of its own beyond those rows.
//
// With no selected session there is no title to repeat and no run to report, so
// the toolbar keeps only what still applies: the connection status and the
// appearance entry. The conversation's empty state carries the heading instead.
func header(c *ui.Context, k tokens, v HeaderView, a Actions) {
	locale := v.Locale
	ui.Row(c).Role(ui.RoleToolbar).Label(tr(locale, keyHeaderRegion)).
		Gap(k.gapWide).MinHeight(k.headerH).Padding(k.toolbarPadY, k.toolbarPadX, k.toolbarPadY, breadcrumbLeftPad).AlignItems(ui.Center).
		BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border).Transition(motionLayoutColors(c)).Children(func() {
		if !v.HasSession {
			// No session: no breadcrumb, no menu and no run row, only the space
			// that keeps what remains at the toolbar's end.
			ui.Box(c).Grow(1)
			return
		}
		// The breadcrumb and its "…" menu are the bar's leading group: the title
		// truncates against the menu, whose target starts where the title's box
		// ends, as the reference draws them. The menu's own glyph bearing and the
		// room a cut title keeps after its ellipsis make the visible step of about
		// 13 DIPs; the trailing entries keep the bar's wider toolbar gap.
		ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Center).Children(func() {
			breadcrumb(c, k, v, locale)
			sessionMenu(c, k, v, a)
		})
		// The find bar takes the toolbar's trailing room while it shows, so its
		// field, count and entries stay on the bar instead of competing with the
		// entries below for a narrow window's last DIPs. Closing it brings them
		// back; nothing about the session changes while it is open.
		if v.ThreadSearch.IsOpen() {
			renderThreadSearch(c, k, v.ThreadSearch, a)
			return
		}
		// Stopping is the App Server's to do once the callback sends session.stop,
		// so the entry belongs beside the status only while a turn still runs.
		if !v.Closed && v.Run == RunRunning {
			if a.OnStop == nil {
				wipAction(c, k, tr(locale, keyHeaderStop), tr(locale, keyReasonStop))
			} else {
				ui.Button(c, tr(locale, keyHeaderStop)).OnClick(a.OnStop).Transition(motionFastColors(c))
			}
		}
		// A healthy connection has nothing to reconnect, a reconnect in flight
		// already shows its loading state, and a closed session has no attachment
		// to re-open.
		if !v.Closed && v.Connection != ConnReady && !v.Loading {
			if a.OnReconnect == nil {
				wipAction(c, k, tr(locale, keyHeaderReconnect), tr(locale, keyReasonReconnect))
			} else {
				ui.Button(c, tr(locale, keyHeaderReconnect)).OnClick(a.OnReconnect).Transition(motionFastColors(c))
			}
		}
		// The current model is the conversation's, so its chip waits for the
		// conversation like the panel entry does: with Settings or the keys
		// browser showing, the composer that draws the model list is not on
		// screen, and opening it would only write a flag. It sits before the
		// panel toggle, where the reference draws its trailing model entry, and
		// names the pair the App Server reported. An empty label draws no chip,
		// so a window without a model fact keeps the bar's old shape.
		if !v.SettingsOpen && !v.KeysOpen {
			headerModelChip(c, k, v, a)
		}
		// The context panel is the conversation's, so its entry appears only
		// while neither surface that replaces the conversation is showing. With
		// no workspace to show changed files for, the entry is absent rather than
		// disabled. It is the bar's trailing entry after the model chip, and the
		// reference's editor control and the rule it puts before the panel toggle
		// are not drawn, as the shell has no editor client to open anything with.
		if !v.SettingsOpen && !v.KeysOpen && a.OnToggleContextPanel != nil {
			contextPanelToggle(c, k, v, a)
		}
	})
	banner(c, k, v, a)
	deleteConfirmation(c, k, v, a)
	renameDialog(c, k, v, a)
}

// breadcrumb draws "Project / Title" as the reference's one line: each part is
// single-line and truncated, and the title takes the room that is left. An
// empty project draws no project name and no separator, because the shell has
// no project fact to show and never invents one.
func breadcrumb(c *ui.Context, k tokens, v HeaderView, locale Locale) {
	ui.Row(c).Grow(1).MinWidth(0).Gap(breadcrumbGap).AlignItems(ui.Center).Children(func() {
		if v.Project != "" {
			ui.Text(c, v.Project).FontSize(k.bodySize).TextColor(k.mutedForeground).
				SingleLine().Ellipsis("…").MaxWidth(breadcrumbProjectMaxW).Shrink(1).MinWidth(0)
			// The separator is punctuation between two labels, so assistive
			// technology reads the two names rather than a bare slash.
			ui.Text(c, breadcrumbSeparator).FontSize(k.bodySize).TextColor(k.mutedForeground).
				SingleLine().Shrink(0).AccessHidden(true)
		}
		ui.Text(c, v.Title).Bold().FontSize(k.bodySize).SingleLine().
			Padding(0, -titleEllipsisOverlap, 0, 0).Grow(1).MinWidth(0)
	})
}

// sessionMenu draws the "…" entry at the end of the breadcrumb and its menu.
// The menu follows the reference's structure in two groups. The first is the
// session's own entries: pin, rename, open in a new window, export, archive and
// the three copy entries. Pin and Archive are wired to the window's
// desktop-local display lists and toggle with the state HeaderView reports;
// Rename opens the window's title dialog when that callback exists; Copy
// conversation ID copies the selected id to the system clipboard and Copy debug
// info copies the model's public runtime facts. Both record a muted outcome,
// because the id and the model config are facts the shell already displays. An
// empty id or an empty debug text leaves the entry as the menu's WIP statement.
// The rest stay visible, disabled and explained as WIP with the gap they wait
// on. The second group keeps the entries the shell does have: the keys browser,
// ending a non-main branch, refreshing the branch directory, deleting the
// session behind a confirmation, and the desktop-local appearance surface. An
// entry whose callback is missing stays visible, disabled and marked WIP too, so
// the menu never pretends an action the shell cannot carry out.
//
// The menu's panel is mygo's own popover, which lifts by its theme's shadow; the
// shell's popover step is not applied here because the panel is not this view's
// element, and mygo offers no hook for its shadow.
func sessionMenu(c *ui.Context, k tokens, v HeaderView, a Actions) {
	locale := v.Locale
	label := tr(locale, keyHeaderMenu)
	button := ui.ButtonBase(c).Key("header-menu").Size(menuTargetW, k.sidebarEntryH).
		Radius(k.radiusSM).AlignItems(ui.Center).Justify(ui.Center).
		Label(label).Tooltip(label).Transition(motionFastColors(c))
	if button.Hovered() {
		button.Background(k.sidebarHover)
	}
	button.Children(func() {
		ui.Text(c, "…").FontSize(k.titleSize).TextColor(k.mutedForeground).AccessHidden(true)
	})
	button.PopoverMenuButton(func(m *ui.Menu) {
		// The reference's first group. Pin and Archive are desktop-local display
		// state: while the window wires them the entry toggles, and its label
		// reads the state the caller reports. Rename reaches the window's title
		// dialog once the caller supplies OnRename, and the remaining five carry
		// the menu's own keys until i18n.go picks them up.
		if a.OnPinSession == nil || v.SessionID == "" {
			menuWIP(m, menuText(locale, keySidebarPin), menuText(locale, keyReasonPin))
		} else {
			pinKey := keySidebarPin
			if v.Pinned {
				pinKey = keySidebarUnpin
			}
			if menuItem(m, menuText(locale, pinKey)).Chosen() {
				a.OnPinSession(v.SessionID)
			}
		}
		if a.OnRequestRename == nil {
			menuWIP(m, menuText(locale, keySidebarRename), menuText(locale, keyReasonRename))
		} else if menuItem(m, menuText(locale, keySidebarRename)).Chosen() {
			a.OnRequestRename()
		}
		menuWIP(m, menuText(locale, keyHeaderMenuOpenInNewWindow), menuText(locale, keyReasonOpenInNewWindow))
		menuWIP(m, menuText(locale, keyHeaderMenuExport), menuText(locale, keyReasonExport))
		if a.OnArchiveSession == nil || v.SessionID == "" {
			menuWIP(m, menuText(locale, keySidebarArchive), menuText(locale, keyReasonArchive))
		} else {
			archiveKey := keySidebarArchive
			if v.Archived {
				archiveKey = keySidebarUnarchive
			}
			if menuItem(m, menuText(locale, archiveKey)).Chosen() {
				a.OnArchiveSession(v.SessionID)
			}
		}
		// The conversation link is the selected id in the scheme the app
		// registers, so it is a window action like the copy entries: the region
		// writes the clipboard itself and tells the shell, which shows the muted
		// outcome. With no selected identity there is no link to build, and the
		// entry stays the menu's WIP statement.
		if v.SessionID == "" {
			menuWIP(m, menuText(locale, keyHeaderMenuCopyLink), menuText(locale, keyReasonCopyLink))
		} else if menuItem(m, menuText(locale, keyHeaderMenuCopyLink)).Chosen() {
			copyConversationLink(c, v.SessionID, a)
		}
		// Copying the id is the window's own action: the id is a fact the App
		// Server already reported, so the region writes it to the system
		// clipboard itself and tells the shell, which shows the muted outcome.
		// With no selected identity there is nothing to copy, and the entry
		// stays the menu's WIP statement.
		if v.SessionID == "" {
			menuWIP(m, menuText(locale, keyHeaderMenuCopyID), menuText(locale, keyReasonCopyID))
		} else if menuItem(m, menuText(locale, keyHeaderMenuCopyID)).Chosen() {
			copyConversationID(c, v.SessionID, a)
		}
		if v.DebugInfo == "" {
			menuWIP(m, menuText(locale, keyHeaderMenuCopyDebug), menuText(locale, keyReasonCopyDebug))
		} else if menuItem(m, menuText(locale, keyHeaderMenuCopyDebug)).Chosen() {
			copyDebugInfo(c, v.DebugInfo, a)
		}
		m.Separator()
		if a.OnToggleKeys == nil {
			menuWIP(m, tr(locale, keySidebarKeys), tr(locale, keyReasonKeysEntry))
		} else if menuItem(m, tr(locale, keySidebarKeys)).Chosen() {
			a.OnToggleKeys()
		}
		// Only a non-main branch can be ended: main is the branch the App
		// Server always keeps, and with no branch reported there is nothing to
		// address. A missing callback keeps the visible entry explained as WIP.
		if v.SelectedBranch != "" && v.SelectedBranch != mainBranchID {
			if a.OnEndBranch == nil {
				menuWIP(m, tr(locale, keySidebarEndBranch), tr(locale, keyReasonEndBranch))
			} else if menuItem(m, tr(locale, keySidebarEndBranch)).Chosen() {
				a.OnEndBranch(v.SelectedBranch)
			}
		}
		if a.OnRefreshBranches == nil {
			menuWIP(m, tr(locale, keyHeaderMenuRefresh), tr(locale, keyReasonRefreshBranches))
		} else if menuItem(m, tr(locale, keyHeaderMenuRefresh)).Chosen() {
			a.OnRefreshBranches()
		}
		// The rule between the branch entries and the destructive pair, as the
		// reference separates what acts on the branch from what acts on the
		// session.
		m.Separator()
		if a.OnRequestDelete == nil || a.OnDeleteSession == nil || v.SessionID == "" {
			menuWIP(m, tr(locale, keySidebarDelete), tr(locale, keyReasonDelete))
		} else if menuItem(m, tr(locale, keySidebarDelete)).Danger().Chosen() {
			// The menu item only opens the confirmation; the session is deleted
			// from the alert's confirm button, through Actions.OnDeleteSession.
			a.OnRequestDelete()
		}
		if a.OnOpenSettings != nil {
			settingsKey := keyHeaderSettings
			if v.SettingsOpen {
				settingsKey = keyHeaderConversation
			}
			if menuItem(m, tr(locale, settingsKey)).Chosen() {
				a.OnOpenSettings()
			}
		}
	})
}

// menuItem adds one session-menu entry with the reference's row height. The
// label is drawn through Content so its line box takes menuRowTextH instead of
// the theme's default line, which leaves the drawn row at 28 DIPs; the label
// stays the item's accessible name and the system's menus still show it.
func menuItem(m *ui.Menu, label string) *ui.MenuItem {
	return m.Item(label).Content(func(content ui.MenuItemContent) {
		ui.Text(content.Context, label).SingleLine().FixedLineHeight(menuRowTextH)
	})
}

// menuWIP adds one menu entry the shell cannot carry out yet: disabled, marked
// WIP, and explained with the upstream method it waits on. The reason is the
// tooltip, which assistive technology also reads as the item's description, so
// it explains the entry without a second line that would break the 28 DIP row.
func menuWIP(m *ui.Menu, label, reason string) {
	menuItem(m, label).Disabled(true).Trailing(wipBadge).Tooltip(reason)
}

// copyConversationID copies the selected session's id to the system clipboard.
// The id is a fact the App Server already reported, so this is a window action
// with no request behind it: the region writes the clipboard through the
// window's service and the shell records the muted outcome line through
// Actions.OnCopySessionID. It sends nothing and changes no session fact. A
// window without that callback still copies; only the status line is missing.
func copyConversationID(c *ui.Context, sessionID string, a Actions) {
	c.Services().WriteClipboard(sessionID)
	if a.OnCopySessionID != nil {
		a.OnCopySessionID(sessionID)
	}
}

// copyConversationLink copies the selected session's deep link to the system
// clipboard. The link is ConversationLink's own text built from the id the App
// Server already reported and the scheme cmd/dimagent registers, so this is a
// window action with no request behind it: the region writes the clipboard and
// the shell records the muted "Copied conversation link" outcome through
// Actions.OnCopyConversationLink. A window without that callback still copies;
// only the status line is missing. The menu offers the entry only while a
// session identity is selected.
func copyConversationLink(c *ui.Context, sessionID string, a Actions) {
	c.Services().WriteClipboard(ConversationLink(sessionID))
	if a.OnCopyConversationLink != nil {
		a.OnCopyConversationLink()
	}
}

// copyDebugInfo copies the model's public runtime facts. The text is already in
// HeaderView.DebugInfo, built by the shell from the catalog facts the window
// displays, so the region writes it and sends nothing. The shell records the
// muted "Copied debug info" outcome through Actions.OnCopyDebugInfo; a window
// without that callback still copies, and only the status line is missing. The
// menu offers the entry only while the text is non-empty.
func copyDebugInfo(c *ui.Context, text string, a Actions) {
	c.Services().WriteClipboard(text)
	if a.OnCopyDebugInfo != nil {
		a.OnCopyDebugInfo()
	}
}

// deleteConfirmation draws the alert the menu's delete entry opens. The alert
// is the window's own confirmation: the menu never deletes a session, and the
// confirm button calls Actions.OnDeleteSession with the selected session's
// identity. Cancel, Escape and a click outside the alert delete nothing.
func deleteConfirmation(c *ui.Context, k tokens, v HeaderView, a Actions) {
	if !v.HasSession || v.DeleteConfirm == nil {
		return
	}
	locale := v.Locale
	switch ui.AlertDialog(c, v.DeleteConfirm,
		tr(locale, keyHeaderDeleteTitle), tr(locale, keyHeaderDeleteMessage),
		tr(locale, keyHeaderDeleteCancel), tr(locale, keyHeaderDeleteConfirm)) {
	case 1:
		if a.OnDeleteSession != nil && v.SessionID != "" {
			a.OnDeleteSession(v.SessionID)
		}
	}
}

// headerModelChip draws the bar's current-model entry: a compact pill that
// names the pair the App Server reported, with muted text and the toolbar's
// hover face. It is a button, so the Tab key reaches it and Enter or Space
// toggles the list like a click does; the click calls
// Actions.OnToggleModelList once, the same action the composer's chip calls,
// and changes no model fact itself. A window without that action keeps the
// chip visible, disabled and explained with the reason it waits on, rather
// than offering a click that would do nothing. An empty label draws no chip:
// the shell never invents a model to name.
//
// The label is the shell's display copy of the current provider/model pair. It
// truncates with an ellipsis inside headerModelMaxW and gives room back to the
// bar's other entries when the window cannot fit them all; the full pair stays
// the chip's tooltip.
func headerModelChip(c *ui.Context, k tokens, v HeaderView, a Actions) {
	if v.ModelLabel == "" {
		return
	}
	name := tr(v.Locale, keyModelTitle)
	// The chip's width is the label's own, capped by headerModelMaxW, and it is
	// set as the flex basis so a long pair does not claim the tail of the bar
	// before the layout clamps it: the bar's leading breadcrumb keeps its room
	// and the label truncates inside the chip. MinWidth(0) lets the chip give
	// that room back in a narrow window instead of squeezing the menu or the
	// panel toggle.
	padX := k.space(headerModelPadX)
	labelW, _ := c.MeasureText(0, ui.Span{Text: v.ModelLabel, Size: k.smallSize})
	width := min(labelW+2*padX, headerModelMaxW)
	chip := ui.ButtonBase(c).Key("header-model").
		Width(width).Shrink(1).MinWidth(0).
		Padding(k.space(headerModelPadY), padX).
		Radius(k.radiusSM).AlignItems(ui.Center).Role(ui.RoleButton).
		Label(name).Transition(motionFastColors(c))
	if a.OnToggleModelList != nil {
		chip.Tooltip(v.ModelLabel)
		chip.OnClick(a.OnToggleModelList)
		if chip.Hovered() {
			chip.Background(k.sidebarHover)
		}
	} else {
		chip.AccessDisabled(true).Description(reasonHeaderModel).Tooltip(reasonHeaderModel)
	}
	chip.Children(func() {
		ui.Text(c, v.ModelLabel).FontSize(k.smallSize).TextColor(k.mutedForeground).
			SingleLine().Ellipsis("…").Unselectable()
	})
}

// contextPanelToggle is the toolbar's panel entry: an icon button at the
// toolbar's end, so the mark sits where the reference draws it. Its accessible
// name and tooltip are the label; a click asks the window to toggle the panel,
// and the next frame draws the state it wrote. The entry keeps the hover face
// while the panel is open, so its state is visible without a tooltip.
func contextPanelToggle(c *ui.Context, k tokens, v HeaderView, a Actions) {
	toggle := ui.ButtonBase(c).Key("header-context-panel").Size(k.sidebarEntryH, k.sidebarEntryH).
		Radius(k.radiusSM).AlignItems(ui.Center).Justify(ui.Center).
		Label(tr(v.Locale, keyContextPanelToggle)).Tooltip(tr(v.Locale, keyContextPanelToggle)).
		Transition(motionFastColors(c))
	if v.ContextPanelOpen || toggle.Hovered() {
		toggle.Background(k.sidebarHover)
	}
	if a.OnToggleContextPanel != nil {
		toggle.OnClick(a.OnToggleContextPanel)
	} else {
		const reason = "Unavailable: the window has no context-panel toggle callback."
		toggle.AccessDisabled(true).Description(reason).Tooltip(reason)
	}
	toggle.Children(func() {
		sidebarGlyph(c, contextPanelGlyph, k.mutedForeground)
	})
}

// banner, in notices.go, draws the window's notice strip: the terminal close,
// connection and action failures, the transient action and projection notices,
// and their dismiss entry. A closed session outranks transport errors because
// it is terminal; connection failures outrank older action failures. Live
// refreshes still leave ActionError alone.
