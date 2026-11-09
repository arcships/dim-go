// This file owns the main area's conversation bar and banner: the project and
// session-title breadcrumb, the session menu, the conditional Stop and Reconnect
// entries, the model entry, the context panel toggle, the delete confirmation
// and the error banner below them. It reads HeaderView and Actions only (see
// protocols.go).
//
// The reference's conversation bar is one line with the breadcrumb and a "…"
// menu on the left and the surface entries on the right. Run and connection
// status are not drawn here: the reference has no "Session: Idle" or
// "App Server: …" row, and this shell no longer adds one.

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

// contextPanelGlyph is the entry's two-pane mark, as the Electron reference
// draws the panel toggle at the card's top right: a rounded frame split into
// two panes with the right one filled. It strokes and fills currentColor, so
// it follows the theme like the sidebar's glyphs.
var contextPanelGlyph = sidebarIcon(`<rect x="1.8" y="2.6" width="12.4" height="10.8" rx="2"/><path d="M9.2 2.6v10.8"/><path d="M9.2 3.4h4.2a.8.8 0 0 1 .8.8v7.6a.8.8 0 0 1-.8.8H9.2Z" fill="currentColor" stroke="none"/>`)

// header shows the selected session's breadcrumb and entries, then the banner
// that belongs under them. It draws no region of its own beyond those rows.
//
// With no selected session there is no title to repeat and no run to report, so
// the toolbar keeps only what still applies: the connection status and the
// appearance entry. The conversation's empty state carries the heading instead.
func header(c *ui.Context, k tokens, v HeaderView, a Actions) {
	locale := v.Locale
	ui.Row(c).Role(ui.RoleToolbar).Label(tr(locale, keyHeaderRegion)).
		Gap(k.gapWide).MinHeight(k.headerH).Padding(k.toolbarPadY, k.toolbarPadX).AlignItems(ui.Center).
		BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border).Children(func() {
		if !v.HasSession {
			// No session: no breadcrumb, no menu and no run row, only the space
			// that keeps what remains at the toolbar's end.
			ui.Box(c).Grow(1)
			return
		}
		// The breadcrumb takes the room left of the right-hand entries and
		// truncates each part on one line, as the reference truncates its project
		// and title.
		breadcrumb(c, k, v, locale)
		sessionMenu(c, k, v, a)
		// Stopping is the App Server's to do once the callback sends session.stop,
		// so the entry belongs beside the status only while a turn still runs.
		if !v.Closed && v.Run == RunRunning {
			if a.OnStop == nil {
				wipAction(c, k, tr(locale, keyHeaderStop), tr(locale, keyReasonStop))
			} else {
				ui.Button(c, tr(locale, keyHeaderStop)).OnClick(a.OnStop)
			}
		}
		// A healthy connection has nothing to reconnect, a reconnect in flight
		// already shows its loading state, and a closed session has no attachment
		// to re-open.
		if !v.Closed && v.Connection != ConnReady && !v.Loading {
			if a.OnReconnect == nil {
				wipAction(c, k, tr(locale, keyHeaderReconnect), tr(locale, keyReasonReconnect))
			} else {
				ui.Button(c, tr(locale, keyHeaderReconnect)).OnClick(a.OnReconnect)
			}
		}
		// The model entry is a picker the reference draws beside the editor
		// entry. The shell has no model list or current-model fact from the App
		// Server, so it stays a visible, unavailable entry that explains the gap
		// instead of opening an empty picker.
		modelEntry(c, k, v, locale)
		headerDivider(c, k)
		// The context panel is the conversation's, so its entry appears only
		// while neither surface that replaces the conversation is showing. With
		// no workspace to show changed files for, the entry is absent rather than
		// disabled.
		if !v.SettingsOpen && !v.KeysOpen && a.OnToggleContextPanel != nil {
			contextPanelToggle(c, k, v, a)
		}
	})
	banner(c, k, v)
	deleteConfirmation(c, k, v, a)
}

// breadcrumb draws "Project / Title" as the reference's one line: each part is
// single-line and truncated, and the title takes the room that is left. An
// empty project draws no project name and no separator, because the shell has
// no project fact to show and never invents one.
func breadcrumb(c *ui.Context, k tokens, v HeaderView, locale Locale) {
	ui.Row(c).Grow(1).MinWidth(0).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		if v.Project != "" {
			ui.Text(c, v.Project).FontSize(k.bodySize).TextColor(k.mutedForeground).
				SingleLine().Shrink(1).MinWidth(0)
			// The separator is punctuation between two labels, so assistive
			// technology reads the two names rather than a bare slash.
			ui.Text(c, breadcrumbSeparator).FontSize(k.bodySize).TextColor(k.mutedForeground).
				SingleLine().Shrink(0).AccessHidden(true)
		}
		ui.Text(c, v.Title).Bold().FontSize(k.bodySize).SingleLine().
			Grow(1).MinWidth(0)
	})
}

// sessionMenu draws the "…" entry at the end of the breadcrumb and its menu.
// The items are the entries the sidebar no longer draws but the session still
// has: the keys browser, ending a non-main branch, refreshing the branch
// directory, deleting the session behind a confirmation, and the desktop-local
// appearance surface as a fallback. An entry whose callback is missing stays
// visible, disabled and marked WIP with the reason it waits on, so the menu
// never pretends an action the shell cannot carry out.
func sessionMenu(c *ui.Context, k tokens, v HeaderView, a Actions) {
	locale := v.Locale
	label := tr(locale, keyHeaderMenu)
	button := ui.ButtonBase(c).Size(k.sidebarEntryH, k.sidebarEntryH).
		Radius(radiusSM).AlignItems(ui.Center).Justify(ui.Center).
		Label(label).Tooltip(label)
	if button.Hovered() {
		button.Background(k.sidebarHover)
	}
	button.Children(func() {
		ui.Text(c, "…").FontSize(k.titleSize).TextColor(k.mutedForeground).AccessHidden(true)
	})
	button.PopoverMenuButton(func(m *ui.Menu) {
		if a.OnToggleKeys == nil {
			menuWIP(m, tr(locale, keySidebarKeys), tr(locale, keyReasonKeysEntry))
		} else if m.Item(tr(locale, keySidebarKeys)).Chosen() {
			a.OnToggleKeys()
		}
		m.Separator()
		// Only a non-main branch can be ended: main is the branch the App
		// Server always keeps, and with no branch reported there is nothing to
		// address. A missing callback keeps the visible entry explained as WIP.
		if v.SelectedBranch != "" && v.SelectedBranch != mainBranchID {
			if a.OnEndBranch == nil {
				menuWIP(m, tr(locale, keySidebarEndBranch), tr(locale, keyReasonEndBranch))
			} else if m.Item(tr(locale, keySidebarEndBranch)).Chosen() {
				a.OnEndBranch(v.SelectedBranch)
			}
		}
		if a.OnRefreshBranches == nil {
			menuWIP(m, tr(locale, keyHeaderMenuRefresh), tr(locale, keyReasonRefreshBranches))
		} else if m.Item(tr(locale, keyHeaderMenuRefresh)).Chosen() {
			a.OnRefreshBranches()
		}
		m.Separator()
		if a.OnRequestDelete == nil || a.OnDeleteSession == nil || v.SessionID == "" {
			menuWIP(m, tr(locale, keySidebarDelete), tr(locale, keyReasonDelete))
		} else if m.Item(tr(locale, keySidebarDelete)).Danger().Chosen() {
			// The menu item only opens the confirmation; the session is deleted
			// from the alert's confirm button, through Actions.OnDeleteSession.
			a.OnRequestDelete()
		}
		if a.OnOpenSettings != nil {
			settingsKey := keyHeaderSettings
			if v.SettingsOpen {
				settingsKey = keyHeaderConversation
			}
			if m.Item(tr(locale, settingsKey)).Chosen() {
				a.OnOpenSettings()
			}
		}
	})
}

// menuWIP adds one menu entry the shell cannot carry out yet: disabled, marked
// WIP, and explained with the upstream method it waits on. The subtitle is the
// reason, so both a reader and assistive technology see it rather than a
// silently inert item.
func menuWIP(m *ui.Menu, label, reason string) {
	m.Item(label).Disabled(true).Trailing(wipBadge).Subtitle(reason)
}

// modelEntry draws the reference's model picker position as an unavailable
// entry: the model label with a disclosure arrow, dimmed and explained with the
// missing App Server model fact. It takes no keyboard focus, like the
// composer's mode and model entries, so the documented Tab order stays short.
func modelEntry(c *ui.Context, k tokens, v HeaderView, locale Locale) {
	reason := tr(locale, keyReasonModel)
	button := ui.ButtonBase(c).Height(k.sidebarEntryH).Padding(0, k.gapWide).
		Radius(radiusSM).Gap(k.gap).AlignItems(ui.Center).
		Label(tr(locale, keyHeaderModelPicker)).Description(reason).Tooltip(reason).
		Disabled(true).AccessDisabled(true)
	button.Children(func() {
		// The arrow is part of the same text run, so the model label stays one
		// node and does not read as a second "Model" beside the composer's.
		ui.Text(c, tr(locale, keyHeaderModel)+" ▾").FontSize(k.bodySize).
			TextColor(k.mutedForeground).SingleLine()
	})
}

// headerDivider is the hairline between the model entry and the context panel
// toggle, as the reference draws a vertical rule before its panel switch.
func headerDivider(c *ui.Context, k tokens) {
	ui.Box(c).Width(k.hairline).Height(k.sidebarEntryH).Shrink(0).
		Background(k.border).AccessHidden(true)
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
	toggle := ui.ButtonBase(c).Size(k.sidebarEntryH, k.sidebarEntryH).
		Radius(radiusSM).AlignItems(ui.Center).Justify(ui.Center).
		Label(tr(v.Locale, keyContextPanelToggle)).Tooltip(tr(v.Locale, keyContextPanelToggle))
	if v.ContextPanelOpen || toggle.Hovered() {
		toggle.Background(k.sidebarHover)
	}
	toggle.OnClick(a.OnToggleContextPanel).Children(func() {
		sidebarGlyph(c, contextPanelGlyph, k.mutedForeground)
	})
}

// banner shows the terminal session.closed state, the last window-owned action
// error, or the supervisor's last connection error, in a live region assistive
// technology reads out as it appears. A closed session outranks transport
// errors because it is terminal; connection failures outrank older action
// failures. Live refreshes still leave ActionError alone.
func banner(c *ui.Context, k tokens, v HeaderView) {
	message := ""
	switch {
	case v.Closed:
		message = tr(v.Locale, keyHeaderSessionClosed)
		if v.ClosedReason != "" {
			message += ": " + v.ClosedReason
		}
	case v.Connection == ConnError && v.ConnectionError != "":
		message = v.Connection.String() + ": " + v.ConnectionError
	default:
		message = v.ActionError
	}
	if message == "" {
		return
	}
	ui.Row(c).Role(ui.RoleAlert).Gap(k.gap).Padding(k.gap, k.edgePad).
		Background(k.wash(k.danger)).
		BorderWidth(0, 0, k.hairline, 0).BorderColor(k.danger).Children(func() {
		ui.Text(c, message).FontSize(k.labelSize).TextColor(k.danger).Bold()
	})
}
