// This file owns the main area's toolbar and banner: the selected session's
// title, the run status row, the connection status row, the Stop and Reconnect
// entries, and the terminal or error banner below them. It reads HeaderView and
// Actions only (see protocols.go).

package ui

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

// header shows the selected session's title beside the connection status, then
// the banner that belongs under it. It draws no region of its own beyond those
// two rows.
//
// With no selected session there is no title to repeat and no run to report, so
// the toolbar keeps only what still applies: the connection status and the
// appearance entry. The conversation's empty state carries the heading instead.
func header(c *ui.Context, k tokens, v HeaderView, a Actions) {
	locale := v.Locale
	ui.Row(c).Gap(k.gapWide).Padding(k.toolbarPadY, k.toolbarPadX).AlignItems(ui.Center).
		BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border).Children(func() {
		if v.HasSession {
			ui.Text(c, v.Title).Bold().FontSize(k.titleSize)
			ui.Box(c).Grow(1)
			runStatus(c, k, v)
		} else {
			// No session: no title and no run row, only the space that keeps the
			// remaining entries at the toolbar's end.
			ui.Box(c).Grow(1)
		}
		status(c, k, v)
		// Stopping is the App Server's to do once the callback sends session.stop,
		// so the entry belongs beside the status only while a turn still runs.
		if v.HasSession && !v.Closed && v.Run == RunRunning {
			if a.OnStop == nil {
				wipAction(c, k, tr(locale, keyHeaderStop), tr(locale, keyReasonStop))
			} else {
				ui.Button(c, tr(locale, keyHeaderStop)).OnClick(a.OnStop)
			}
		}
		// A healthy connection has nothing to reconnect, a reconnect in flight
		// already shows its loading state, and with no selected session there is
		// no attachment to re-open.
		if v.HasSession && !v.Closed && v.Connection != ConnReady && !v.Loading {
			if a.OnReconnect == nil {
				wipAction(c, k, tr(locale, keyHeaderReconnect), tr(locale, keyReasonReconnect))
			} else {
				ui.Button(c, tr(locale, keyHeaderReconnect)).OnClick(a.OnReconnect)
			}
		}
		// The appearance surface is one toggle away from the conversation; its
		// label says where the entry goes, so the same button always returns.
		if a.OnOpenSettings != nil {
			label := tr(locale, keyHeaderSettings)
			if v.SettingsOpen {
				label = tr(locale, keyHeaderConversation)
			}
			ui.Button(c, label).OnClick(a.OnOpenSettings)
		}
	})
	banner(c, k, v)
}

// runStatus is the toolbar's run row: what the selected session is doing, as a
// dot and a label in a live region, so a turn starting or settling is read out
// once. It is fixture-driven until the supervisor reports session state.
func runStatus(c *ui.Context, k tokens, v HeaderView) {
	ui.Row(c).Role(ui.RoleStatus).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		// A terminal closed session has no active run to color, even if the last
		// state notice said running.
		dot := k.runColor(v.Run)
		if v.Closed {
			dot = k.mutedForeground
		}
		ui.Box(c).Size(k.statusDot, k.statusDot).Radius(k.statusDot / 2).
			Background(dot).Unselectable()
		ui.Text(c, tr(v.Locale, keyHeaderSessionPrefix)+runLabel(v)).FontSize(k.smallSize).TextColor(k.mutedForeground)
		if v.ModelProgress != nil {
			text := v.ModelProgress.Text
			if text == "" {
				text = v.ModelProgress.Thinking
			}
			if text != "" {
				ui.Text(c, text).FontSize(k.smallSize).TextColor(k.mutedForeground)
			}
		}
		if notices := len(v.Notices) + len(v.ActionNotices); notices > 0 {
			ui.Text(c, tr(v.Locale, keyHeaderNoticesPrefix)+strconv.Itoa(notices)).FontSize(k.smallSize).TextColor(k.mutedForeground)
		}
		if v.ActionStatus != "" {
			ui.Text(c, v.ActionStatus).FontSize(k.smallSize).TextColor(k.mutedForeground)
		}
	})
}

// runLabel names the session lifecycle in the status row. A terminal
// session.closed outranks the last run state, which must not claim work is
// still running.
func runLabel(v HeaderView) string {
	if v.Closed {
		return tr(v.Locale, keyHeaderClosed)
	}
	return v.Run.String()
}

// status is the toolbar's inspector row: the connection's state as a dot and a
// label in a live region, so assistive technology reads a change as it comes.
func status(c *ui.Context, k tokens, v HeaderView) {
	ui.Row(c).Role(ui.RoleStatus).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(k.statusDot, k.statusDot).Radius(k.statusDot / 2).
			Background(k.connectionColor(v.Connection)).Unselectable()
		ui.Text(c, tr(v.Locale, keyHeaderConnectionPrefix)+v.Connection.String()).FontSize(k.smallSize).TextColor(k.mutedForeground)
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
