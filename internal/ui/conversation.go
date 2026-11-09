// This file owns the conversation region between the banner and the
// composer: the keyed list that follows its end, the loading and empty states, and
// the per-author row dispatch.

package ui

import "github.com/egoist/mygo/ui"

// conversationList shows the fixture rows in a keyed, virtualized list that
// follows its end, the shape the TimelineFold-driven conversation takes: only
// the rows in view are built, a row keeps its state by message ID, and a
// streaming append stays in view until the user scrolls away from the end.
func (s *State) conversationList(c *ui.Context, k tokens) {
	switch {
	case s.ConversationLoading:
		s.loadingConversation(c, k)
	case len(s.Messages) == 0:
		s.emptyConversation(c, k)
	default:
		list := &s.conversation
		list.FollowEnd = true
		list.Key = func(row int) any { return s.Messages[row].ID }
		ui.List(c, list, len(s.Messages), func(row int) {
			s.messageRow(c, k, s.Messages[row])
		}).Grow(1).Label("Conversation")
	}
}

// loadingConversation is what shows while history is still coming. The row is a
// busy status region, so assistive technology waits for the rows rather than
// reading the placeholder, and reads the one label of the work.
func (s *State) loadingConversation(c *ui.Context, k tokens) {
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
func (s *State) emptyConversation(c *ui.Context, k tokens) {
	ui.Column(c).Grow(1).Padding(k.panelPad).Gap(k.gap).Children(func() {
		ui.Text(c, "No messages yet").FontSize(k.bodySize).TextColor(k.mutedForeground).Role(ui.RoleStatus)
		ui.Text(c, "The conversation fills in when the App Server projection arrives.").
			FontSize(k.smallSize).TextColor(k.mutedForeground)
	})
}

// messageRow builds one row by its author, the way docs/ui-layout-spec.md §4
// splits dim-electron's timeline: a user bubble, an assistant turn with its
// blocks in order, a tool card, an approval row with its decisions, or an
// error/retry notice. A row's own text stands in for an unknown author, so a
// caller cannot lose text to a row type the shell does not know.
func (s *State) messageRow(c *ui.Context, k tokens, m Message) {
	switch m.Author {
	case AuthorUser:
		s.userRow(c, k, m)
	case AuthorAssistant:
		s.assistantRow(c, k, m)
	case AuthorTool:
		s.toolRow(c, k, m)
	case AuthorApproval:
		s.approvalRow(c, k, m)
	case AuthorError:
		s.errorRow(c, k, m)
	case AuthorRetry:
		s.retryRow(c, k, m)
	default:
		ui.Column(c).Padding(k.rowPadY, k.timelinePadX).Children(func() {
			ui.Text(c, m.Content).FontSize(k.bodySize)
		})
	}
}
