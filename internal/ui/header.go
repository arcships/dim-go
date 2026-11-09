// This file owns the main area's toolbar: the selected session's
// title, the run status row, the connection status row, the terminal banner and
// the title lookup the toolbar draws.

package ui

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

// header shows the selected session's title beside the connection status.
func (s *State) header(c *ui.Context, k tokens) {
	ui.Row(c).Gap(k.gapWide).Padding(k.toolbarPadY, k.toolbarPadX).AlignItems(ui.Center).
		BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border).Children(func() {
		ui.Text(c, s.selectedTitle()).Bold().FontSize(k.titleSize)
		ui.Box(c).Grow(1)
		s.runStatus(c, k)
		s.status(c, k)
		// Stopping is the App Server's to do once the callback sends session.stop,
		// so the entry belongs beside the status only while a turn still runs.
		if s.Selected != "" && !s.SessionClosed && s.Run == RunRunning {
			if s.OnStop == nil {
				wipAction(c, k, "Stop", reasonStop)
			} else {
				ui.Button(c, "Stop").OnClick(s.stopRun)
			}
		}
		// A healthy connection has nothing to reconnect, a reconnect in flight
		// already shows its loading state, and with no selected session there is
		// no attachment to re-open.
		if s.Selected != "" && !s.SessionClosed && s.Connection != ConnReady && !s.ConversationLoading {
			if s.OnReconnect == nil {
				wipAction(c, k, "Reconnect", reasonReconnect)
			} else {
				ui.Button(c, "Reconnect").OnClick(s.reconnect)
			}
		}
	})
}

// runStatus is the toolbar's run row: what the selected session is doing, as a
// dot and a label in a live region, so a turn starting or settling is read out
// once. It is fixture-driven until the supervisor reports session state.
func (s *State) runStatus(c *ui.Context, k tokens) {
	ui.Row(c).Role(ui.RoleStatus).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(k.statusDot, k.statusDot).Radius(k.statusDot / 2).
			Background(k.runColorFor(s)).Unselectable()
		ui.Text(c, "Session: "+s.runLabel()).FontSize(k.smallSize).TextColor(k.mutedForeground)
		if s.ModelProgress != nil {
			text := s.ModelProgress.Text
			if text == "" {
				text = s.ModelProgress.Thinking
			}
			if text != "" {
				ui.Text(c, text).FontSize(k.smallSize).TextColor(k.mutedForeground)
			}
		}
		if notices := len(s.Notices) + len(s.ActionNotices); notices > 0 {
			ui.Text(c, "Notices: "+strconv.Itoa(notices)).FontSize(k.smallSize).TextColor(k.mutedForeground)
		}
		if s.ActionStatus != "" {
			ui.Text(c, s.ActionStatus).FontSize(k.smallSize).TextColor(k.mutedForeground)
		}
	})
}

// runLabel names the session lifecycle in the status row. A terminal
// session.closed outranks the last run state, which must not claim work is
// still running.
func (s *State) runLabel() string {
	if s.SessionClosed {
		return "Closed"
	}
	return s.Run.String()
}

// status is the toolbar's inspector row: the connection's state as a dot and a
// label in a live region, so assistive technology reads a change as it comes.
func (s *State) status(c *ui.Context, k tokens) {
	ui.Row(c).Role(ui.RoleStatus).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(k.statusDot, k.statusDot).Radius(k.statusDot / 2).
			Background(k.connectionColor(s.Connection)).Unselectable()
		ui.Text(c, "App Server: "+s.Connection.String()).FontSize(k.smallSize).TextColor(k.mutedForeground)
	})
}

// banner shows the terminal session.closed state, the last window-owned action
// error, or the supervisor's last connection error, in a live region assistive
// technology reads out as it appears. A closed session outranks transport
// errors because it is terminal; connection failures outrank older action
// failures. Live refreshes still leave ActionError alone.
func (s *State) banner(c *ui.Context, k tokens) {
	message := ""
	switch {
	case s.SessionClosed:
		message = "Session closed"
		if s.SessionClosedReason != "" {
			message += ": " + s.SessionClosedReason
		}
	case s.Connection == ConnError && s.ConnectionError != "":
		message = s.Connection.String() + ": " + s.ConnectionError
	default:
		message = s.ActionError
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

// selectedTitle returns the title of the selected session, or a placeholder.
func (s *State) selectedTitle() string {
	for _, session := range s.Sessions {
		if session.ID == s.Selected {
			return session.Title
		}
	}
	return "No session"
}
