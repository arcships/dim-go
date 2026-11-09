// This file owns the main area's conversation bar and banner: the project and
// session-title breadcrumb, the session menu, the conditional Stop and Reconnect
// entries, the context panel toggle, the delete confirmation, the find bar's
// place at the bar's trailing end and the error banner below them. It reads
// HeaderView and Actions only (see protocols.go). The find bar itself is
// thread_search.go's nested region, drawn through renderThreadSearch.
//
// The reference's conversation bar is one line with the breadcrumb and a "…"
// menu on the left and the surface entries on the right. Its model picker lives
// in the composer, not in the bar, so the bar draws no model entry. Its editor
// entry is absent too: the shell has no editor client or workspace fact, so the
// mark, its disclosure arrow and the rule before the panel toggle would only
// pretend to open something. Run and connection status are not drawn here
// either: the reference has no "Session: Idle" or "App Server: …" row, and this
// shell no longer adds one.

package ui

import (
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

// contextPanelGlyph is the entry's two-pane mark, as the Electron reference
// draws the panel toggle at the card's top right: a 10×10 rounded frame split
// down the middle, both panes outlined. It strokes currentColor and paints no
// fill, so it follows the theme like the sidebar's glyphs.
var contextPanelGlyph = sidebarIcon(`<rect x="3" y="3" width="10" height="10" rx="1.7"/><path d="M8 3v10"/>`)

// reasonOpenInEditor is the explanation the localization table still carries
// for the reference's editor-open entry. The bar no longer draws that entry:
// with no editor client and no workspace fact, the mark, its arrow and the rule
// before the panel toggle were a fake entry rather than a usable one. The text
// stays so a real editor client can bring the entry back without inventing a
// new reason.
const reasonOpenInEditor = "Unavailable: the shell has no editor-open client or workspace fact for opening this session in VS Code yet."

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
// action the shell cannot carry out. Their message entries belong in i18n.go
// beside the other header and reason keys; until that table carries them,
// menuText falls back to the English constants below rather than drawing a raw
// key, and the fallback goes away with the table entries.
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
// text in i18n.go; only these five entries and their reasons are new.
const (
	reasonOpenInNewWindow = "Unavailable: the shell has no second-window client for opening a session yet."
	reasonExport          = "Unavailable: the shell has no conversation export client or App Server export method yet."
	reasonCopyLink        = "Unavailable: the shell has no conversation-link fact for the selected session yet."
	reasonCopyID          = "Unavailable: the window has no copy action for the selected session yet."
	reasonCopyDebug       = "Unavailable: the shell keeps no debug snapshot of the selected session yet."
)

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
		// The context panel is the conversation's, so its entry appears only
		// while neither surface that replaces the conversation is showing. With
		// no workspace to show changed files for, the entry is absent rather than
		// disabled. It is the bar's only trailing entry: the reference's editor
		// control and the rule it puts before the panel toggle are not drawn, as
		// the shell has no editor client to open anything with.
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
// the three copy entries. This shell has no client, callback or fact for any of
// them yet, so each stays visible, disabled and explained as WIP with the gap
// it waits on. The second group keeps the entries the shell does have: the keys
// browser, ending a non-main branch, refreshing the branch directory, deleting
// the session behind a confirmation, and the desktop-local appearance surface.
// An entry whose callback is missing stays visible, disabled and marked WIP
// too, so the menu never pretends an action the shell cannot carry out.
func sessionMenu(c *ui.Context, k tokens, v HeaderView, a Actions) {
	locale := v.Locale
	label := tr(locale, keyHeaderMenu)
	button := ui.ButtonBase(c).Key("header-menu").Size(menuTargetW, k.sidebarEntryH).
		Radius(radiusSM).AlignItems(ui.Center).Justify(ui.Center).
		Label(label).Tooltip(label).Transition(motionFastColors(c))
	if button.Hovered() {
		button.Background(k.sidebarHover)
	}
	button.Children(func() {
		ui.Text(c, "…").FontSize(k.titleSize).TextColor(k.mutedForeground).AccessHidden(true)
	})
	button.PopoverMenuButton(func(m *ui.Menu) {
		// The reference's first group. Pin and Archive keep their translated
		// labels and reasons from the sidebar's table; Rename reaches the
		// window's title dialog once the caller supplies OnRename, and the
		// remaining five carry the menu's own keys until i18n.go picks them up.
		menuWIP(m, menuText(locale, keySidebarPin), menuText(locale, keyReasonPin))
		if a.OnRequestRename == nil {
			menuWIP(m, menuText(locale, keySidebarRename), menuText(locale, keyReasonRename))
		} else if menuItem(m, menuText(locale, keySidebarRename)).Chosen() {
			a.OnRequestRename()
		}
		menuWIP(m, menuText(locale, keyHeaderMenuOpenInNewWindow), menuText(locale, keyReasonOpenInNewWindow))
		menuWIP(m, menuText(locale, keyHeaderMenuExport), menuText(locale, keyReasonExport))
		menuWIP(m, menuText(locale, keySidebarArchive), menuText(locale, keyReasonArchive))
		menuWIP(m, menuText(locale, keyHeaderMenuCopyLink), menuText(locale, keyReasonCopyLink))
		menuWIP(m, menuText(locale, keyHeaderMenuCopyID), menuText(locale, keyReasonCopyID))
		menuWIP(m, menuText(locale, keyHeaderMenuCopyDebug), menuText(locale, keyReasonCopyDebug))
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

// contextPanelToggle is the toolbar's panel entry: an icon button at the
// toolbar's end, so the mark sits where the reference draws it. Its accessible
// name and tooltip are the label; a click asks the window to toggle the panel,
// and the next frame draws the state it wrote. The entry keeps the hover face
// while the panel is open, so its state is visible without a tooltip.
func contextPanelToggle(c *ui.Context, k tokens, v HeaderView, a Actions) {
	toggle := ui.ButtonBase(c).Key("header-context-panel").Size(k.sidebarEntryH, k.sidebarEntryH).
		Radius(radiusSM).AlignItems(ui.Center).Justify(ui.Center).
		Label(tr(v.Locale, keyContextPanelToggle)).Tooltip(tr(v.Locale, keyContextPanelToggle)).
		Transition(motionFastColors(c))
	if v.ContextPanelOpen || toggle.Hovered() {
		toggle.Background(k.sidebarHover)
	}
	toggle.OnClick(a.OnToggleContextPanel).Children(func() {
		sidebarGlyph(c, contextPanelGlyph, k.mutedForeground)
	})
}

// banner, in notices.go, draws the window's notice strip: the terminal close,
// connection and action failures, the transient action and projection notices,
// and their dismiss entry. A closed session outranks transport errors because
// it is terminal; connection failures outrank older action failures. Live
// refreshes still leave ActionError alone.
