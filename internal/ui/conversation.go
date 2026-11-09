// This file owns the timeline region: the keyed conversation list that follows
// its end, the loading and empty states, and the per-author row dispatch. It
// reads TimelineView and Actions only (see protocols.go).

package ui

import "github.com/egoist/mygo/ui"

// timeline shows the projection's rows in a keyed, virtualized list that
// follows its end, the shape the TimelineFold-driven conversation takes: only
// the rows in view are built, a row keeps its state by message ID, and a
// streaming append stays in view until the user scrolls away from the end.
func timeline(c *ui.Context, k tokens, v TimelineView, a Actions) {
	switch {
	case v.Loading:
		loadingConversation(c, k)
	case len(v.Messages) == 0:
		emptyConversation(c, k)
	default:
		list := v.list
		list.FollowEnd = true
		list.Key = func(row int) any { return v.Messages[row].ID }
		ui.List(c, list, len(v.Messages), func(row int) {
			messageRow(c, k, v, a, v.Messages[row])
		}).Grow(1).Label("Conversation")
	}
}

// loadingConversation is what shows while history is still coming. The row is a
// busy status region, so assistive technology waits for the rows rather than
// reading the placeholder, and reads the one label of the work.
func loadingConversation(c *ui.Context, k tokens) {
	ui.Column(c).Grow(1).Padding(k.panelPad).Gap(k.gap).Children(func() {
		ui.Row(c).Role(ui.RoleStatus).Busy(true).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).AccessHidden(true) // the text beside it says what the work is
			ui.Text(c, "Loading conversation…").FontSize(k.bodySize).TextColor(k.mutedForeground)
		})
	})
}

// emptyConversation is what shows once history has loaded and holds no row.
// Its one line is a status region, so assistive technology hears the end of
// loading rather than finding the busy region gone and nothing said.
func emptyConversation(c *ui.Context, k tokens) {
	ui.Column(c).Grow(1).Padding(k.panelPad).Gap(k.gap).Children(func() {
		ui.Text(c, "No messages yet").FontSize(k.bodySize).TextColor(k.mutedForeground).Role(ui.RoleStatus)
		ui.Text(c, "The conversation fills in when the App Server projection arrives.").
			FontSize(k.smallSize).TextColor(k.mutedForeground)
	})
}

// messageRow builds one row by its author, the way docs/ui-layout-spec.md §4
// splits dim-electron's timeline: a user bubble, an assistant turn with its
// blocks in order, a tool card, or an error/retry notice. A pending approval is
// not a row: it takes the composer's slot (renderApprovalPanel). A row's own text
// stands in for an unknown author, so a caller cannot lose text to a row type the
// shell does not know.
func messageRow(c *ui.Context, k tokens, v TimelineView, a Actions, m Message) {
	switch m.Author {
	case AuthorUser:
		userRow(c, k, m)
	case AuthorAssistant:
		assistantRow(c, k, v, m)
	case AuthorTool:
		toolRow(c, k, v.folds, m)
	case AuthorError:
		errorRow(c, k, m)
	case AuthorRetry:
		retryRow(c, k, m)
	default:
		ui.Column(c).Padding(k.rowPadY, k.timelinePadX).Children(func() {
			ui.Text(c, m.Content).FontSize(k.bodySize)
		})
	}
}
