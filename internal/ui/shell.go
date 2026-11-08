// Package ui builds the DimAgent native view tree from plain Go view state.
//
// It is the static shell of Milestone 1: a session sidebar, a status row, a
// keyed conversation list and a composer. It holds no session, run, timeline or
// approval facts; the real catalog, timeline and connection come from the App
// Server and its supervisor, so everything here is throwaway view state that
// the fixture fills by hand.
package ui

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// ComposerShortcutKey is the key of the shell's shortcut that moves the
// keyboard focus to the composer, wherever the focus is in the window: Cmd+L
// on macOS, Ctrl+L elsewhere (ui.Cmd). A menu item can take the same pair once
// the shell has a menu bar.
const ComposerShortcutKey = ui.KeyL

// ComposerShortcutMods is the modifier of ComposerShortcutKey: the platform's
// own shortcut modifier.
var ComposerShortcutMods = ui.Cmd

// Session is a sidebar row. The fixture fills it by hand until the App Server
// session catalog lands.
type Session struct {
	ID    string
	Title string
}

// Author says who wrote a conversation row, which decides how it renders.
type Author uint8

const (
	// AuthorUser is a row of what the user submitted, shown as plain text.
	AuthorUser Author = iota
	// AuthorAssistant is a row of agent output, shown as Markdown.
	AuthorAssistant
)

// String is the author's name, as a row shows it.
func (a Author) String() string {
	if a == AuthorAssistant {
		return "DimAgent"
	}
	return "You"
}

// Message is one conversation row of the fixture. It is view state only: the
// TimelineFold projection owns message identity, order, streaming and tool
// facts, and this struct is replaced by its output in Milestone 3.
type Message struct {
	// ID keys the list row, so a row's state and the list's place survive
	// rows being added, removed or measured.
	ID string
	// Author decides whether the row renders as Markdown or plain text.
	Author Author
	// Content is the row's text: Markdown for an assistant row.
	Content string
}

// State is the view state of the shell.
type State struct {
	Sessions []Session
	Selected string

	// Connection, ConnectionError and ConversationLoading stand in for what
	// the App Server supervisor reports: its connection, the last error it
	// raised and whether history is still loading. They are throwaway view
	// state, not a source of facts, and this package reads no daemon state,
	// SQLite, JSONL or log to fill them.
	Connection          Connection
	ConnectionError     string
	ConversationLoading bool

	// Messages are fixture conversation rows, not session facts.
	Messages []Message
	Draft    string

	// conversation is the list's place among its rows and the identity of each
	// row, kept across frames so that appended rows follow the end and rows keep
	// their state as the conversation changes.
	conversation ui.ListState
	// markdown is the shared Markdown options of every assistant row, whose
	// theme follows the window's.
	markdown markdown.Options
	// composerFocus is the composer's identity, for the shortcut that moves the
	// keyboard focus to it.
	composerFocus ui.Handle
}

// View builds one frame of the AppFrame: sidebar, toolbar, banner, conversation
// and composer.
func (s *State) View(c *ui.Context) {
	k := newTokens(c.Theme())
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Width(k.sidebarW).Shrink(0).PaddingY(k.sidebarPadY).Background(k.surface).Children(func() {
			ui.Sidebar(c, &s.Selected, func() {
				for _, session := range s.Sessions {
					ui.SidebarItem(c, session.ID, nil, session.Title)
				}
			}).Grow(1).Label("Sessions")
		})
		ui.Column(c).Grow(1).FillHeight().Children(func() {
			s.header(c, k)
			s.banner(c, k)
			s.conversationList(c, k)
			s.composer(c, k)
		})
	})
	if c.Shortcut(ComposerShortcutMods, ComposerShortcutKey) {
		s.composerFocus.Focus()
	}
}

// header shows the selected session's title beside the connection status.
func (s *State) header(c *ui.Context, k tokens) {
	ui.Row(c).Gap(k.gapWide).Padding(k.toolbarPadY, k.toolbarPadX).AlignItems(ui.Center).
		BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border).Children(func() {
		ui.Text(c, s.selectedTitle()).Bold().FontSize(k.titleSize)
		ui.Box(c).Grow(1)
		s.status(c, k)
	})
}

// status is the toolbar's inspector row: the connection's state as a dot and a
// label in a live region, so assistive technology reads a change as it comes.
func (s *State) status(c *ui.Context, k tokens) {
	ui.Row(c).Role(ui.RoleStatus).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(k.statusDot, k.statusDot).Radius(k.statusDot / 2).
			Background(k.connectionColor(s.Connection)).Unselectable()
		ui.Text(c, "App Server: "+s.Connection.String()).FontSize(k.smallSize).TextColor(k.muted)
	})
}

// banner shows the supervisor's last error, in a live region assistive
// technology reads out as it appears.
func (s *State) banner(c *ui.Context, k tokens) {
	if s.Connection != ConnError || s.ConnectionError == "" {
		return
	}
	ui.Row(c).Role(ui.RoleAlert).Gap(k.gap).Padding(k.gap, k.edgePad).
		Background(k.wash(k.danger)).
		BorderWidth(0, 0, k.hairline, 0).BorderColor(k.danger).Children(func() {
		ui.Text(c, s.Connection.String()+": "+s.ConnectionError).
			FontSize(k.labelSize).TextColor(k.danger).Bold()
	})
}

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
			ui.Text(c, "Loading conversation…").FontSize(k.bodySize).TextColor(k.muted)
		})
	})
}

// emptyConversation is what shows once history has loaded and holds no row.
// Its one line is a status region, so assistive technology hears the end of
// loading rather than finding the busy region gone and nothing said.
func (s *State) emptyConversation(c *ui.Context, k tokens) {
	ui.Column(c).Grow(1).Padding(k.panelPad).Gap(k.gap).Children(func() {
		ui.Text(c, "No messages yet").FontSize(k.bodySize).TextColor(k.muted).Role(ui.RoleStatus)
		ui.Text(c, "The conversation fills in when the App Server projection arrives.").
			FontSize(k.smallSize).TextColor(k.muted)
	})
}

// messageRow builds one row: the author's name, then the content as Markdown
// for the assistant and as plain text for the user. Tool and approval cards
// take the same place once the projection carries them.
func (s *State) messageRow(c *ui.Context, k tokens, m Message) {
	ui.Column(c).Padding(k.rowPadY, k.rowPadX).Gap(k.gap).Children(func() {
		ui.Text(c, m.Author.String()).FontSize(k.smallSize).TextColor(k.muted).Unselectable()
		if m.Author == AuthorAssistant {
			markdown.View(c, m.Content, &s.markdown)
			return
		}
		ui.Text(c, m.Content).FontSize(k.bodySize)
	})
}

// composer holds the draft. It is a TextArea for now; IME composition is not
// claimed by this shell, only checked headless as draft text that survives a
// rebuild. ComposerShortcutMods with ComposerShortcutKey gives it the focus.
func (s *State) composer(c *ui.Context, k tokens) {
	ui.Column(c).Padding(k.edgePad).Gap(k.gap).Children(func() {
		ui.TextArea(c.Key("composer"), &s.Draft).Bind(&s.composerFocus).
			Height(k.composerH).Placeholder("Message DimAgent").Label("Composer")
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
