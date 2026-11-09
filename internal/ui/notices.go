// This file owns the window's notice strip: the terminal close, connection and
// action failures, the window's confirmed action status, and the transient
// projection and action notices. It reads HeaderView and Actions only (see
// protocols.go).
//
// The strip keeps the precedence the shell already had: a terminal session
// close outranks a connection error, which outranks the last action error. The
// confirmed action status and the remaining action and projection notices
// follow as their own lines. A failure is drawn in the danger color and a
// confirmed outcome in the muted one, so a success or an unresolved status never
// reads like an error, and one dismiss entry clears the window-owned display
// state without writing any App Server fact.

package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// noticeLine is one line of the notice strip and whether it is a failure.
type noticeLine struct {
	text   string
	danger bool
}

// noticePrimary picks the strip's leading message: a terminal close first, then
// a connection failure, then the window's last action error. It is the exact
// text the shell showed before the strip carried more than one notice.
func noticePrimary(v HeaderView) string {
	switch {
	case v.Closed:
		message := tr(v.Locale, keyHeaderSessionClosed)
		if v.ClosedReason != "" {
			message += ": " + v.ClosedReason
		}
		return message
	case v.Connection == ConnError && v.ConnectionError != "":
		return v.Connection.String() + ": " + v.ConnectionError
	default:
		return v.ActionError
	}
}

// noticeText is one notice's line: its kind and, when it carries one, its
// identity or message. The kind is the window's own short name (for example
// "submit.accepted") and stays literal; the text after it is the caller's.
func noticeText(notice Notice) string {
	if notice.ID == "" {
		return notice.Kind
	}
	return notice.Kind + ": " + notice.ID
}

// noticeDanger reports whether one notice is a failure. The window's kinds end
// in ".error" or ".unknown"; everything else is a confirmed outcome and stays
// muted, so an accepted or cancelled action never looks like a failure.
func noticeDanger(kind string) bool {
	return strings.HasSuffix(kind, ".error") || strings.HasSuffix(kind, ".unknown")
}

// noticeLines splits the strip into its leading message and the lines under it.
// The window's confirmed action status comes first as a muted outcome, then the
// action notices; the action error that leads the strip is not repeated from
// ActionNotices, and the projection notices keep their order after the window's
// outcomes.
func noticeLines(v HeaderView) (string, []noticeLine) {
	primary := noticePrimary(v)
	extras := make([]noticeLine, 0, 1+len(v.ActionNotices)+len(v.Notices))
	if v.ActionStatus != "" && v.ActionStatus != primary {
		// ActionStatus is an outcome the App Server confirmed without a session
		// fact to show, such as a submission read back as accepted. It is always
		// muted: an unknown result still has its failure in ActionError, so the
		// status line never doubles as the danger row.
		extras = append(extras, noticeLine{text: v.ActionStatus})
	}
	for _, notice := range v.ActionNotices {
		if notice.ID != "" && notice.ID == v.ActionError {
			// The action error leads the strip when nothing outranks it; showing
			// the same text twice would read as two failures.
			continue
		}
		extras = append(extras, noticeLine{text: noticeText(notice), danger: noticeDanger(notice.Kind)})
	}
	for _, notice := range v.Notices {
		extras = append(extras, noticeLine{text: noticeText(notice), danger: noticeDanger(notice.Kind)})
	}
	return primary, extras
}

// banner draws the notice strip: the leading message, the action and projection
// notices under it, and one dismiss entry while the strip has window-owned
// display state to clear. The strip is a live region, so assistive technology
// reads an outcome as it appears.
func banner(c *ui.Context, k tokens, v HeaderView, a Actions) {
	primary, extras := noticeLines(v)
	if primary == "" && len(extras) == 0 {
		return
	}
	dismissible := a.OnDismissNotices != nil &&
		(v.ActionError != "" || v.ActionStatus != "" || len(v.ActionNotices) > 0 || len(v.Notices) > 0)
	ui.Column(c).Role(ui.RoleAlert).FillWidth().Children(func() {
		if primary != "" {
			noticeRow(c, k, primary, true)
		}
		for _, line := range extras {
			noticeRow(c, k, line.text, line.danger)
		}
		if dismissible {
			ui.Row(c).FillWidth().Padding(k.gap, k.edgePad).Justify(ui.End).Children(func() {
				ui.Button(c, tr(v.Locale, keyNoticesDismiss)).OnClick(a.OnDismissNotices).
					Transition(motionFastColors(c))
			})
		}
	})
}

// noticeRow draws one line of the strip. A failure takes the danger color and
// its wash; a confirmed outcome stays muted and unfilled, so the two never look
// alike. The text is the caller's and is never rewritten here.
func noticeRow(c *ui.Context, k tokens, text string, danger bool) {
	row := ui.Row(c).FillWidth().Padding(k.gap, k.edgePad).
		BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border)
	if danger {
		row.Background(k.wash(k.danger)).BorderColor(k.danger)
	}
	row.Children(func() {
		t := ui.Text(c, text).FontSize(k.labelSize)
		if danger {
			t.TextColor(k.danger).Bold()
		} else {
			t.TextColor(k.mutedForeground)
		}
	})
}
