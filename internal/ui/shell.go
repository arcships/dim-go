// Package ui builds the DimAgent native view tree from plain Go view state.
//
// It is the static shell of Milestone 1: a session sidebar, a status row, a
// keyed conversation list and a composer. It holds no session, run, timeline or
// approval facts; the real catalog, timeline and connection come from the App
// Server and its supervisor, so everything here is throwaway view state that
// the fixture fills by hand.
//
// Actions the shell cannot carry out yet stay visible and unavailable: they are
// marked in the accessibility tree, dimmed, and explained with the upstream
// method they wait on, and no view reports a click on one. See wipAction.
package ui

import (
	"strconv"
	"strings"

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

// Branch is read-only branch metadata supplied by the App Server. Branch
// switching and cut point are not wired yet, so this is not an action list.
type Branch struct {
	ID       string
	Label    string
	State    string
	Head     int64
	Selected bool
}

// Author says who wrote a conversation row, which decides how it renders.
type Author uint8

const (
	// AuthorUser is a row of what the user submitted, shown as plain text.
	AuthorUser Author = iota
	// AuthorAssistant is a row of agent output, shown as Markdown.
	AuthorAssistant
	// AuthorApproval is a pending approval request from the transient
	// projection: plain text with the decisions the shell cannot take yet.
	AuthorApproval
	// AuthorTool is a tool call, drawn as a compact card with its name, its
	// status and one bounded line each of its params and its result.
	AuthorTool
)

// String is the author's name, as a row shows it.
func (a Author) String() string {
	switch a {
	case AuthorAssistant:
		return "DimAgent"
	case AuthorApproval:
		return "Approval request"
	case AuthorTool:
		return "Tool"
	default:
		return "You"
	}
}

// Message is one conversation row of the fixture. It is view state only: the
// TimelineFold projection owns message identity, order, streaming and tool
// facts, and this struct is replaced by its output in Milestone 3.
// ModelProgress is the transient, non-durable model preview shown in the status row.
type ModelProgress struct {
	Text     string
	Thinking string
	Tools    []string
}

// Notice is a transient notification copied from the session projection.
type Notice struct {
	Kind string
	ID   string
}

type Message struct {
	// ID keys the list row, so a row's state and the list's place survive
	// rows being added, removed or measured.
	ID string
	// Author decides whether the row renders as Markdown, plain text or an
	// approval request with its decisions.
	Author Author
	// Content is the row's text: Markdown for an assistant row.
	Content string
	// Tool is the call of an AuthorTool row: its name, previews and status.
	// It stays a pure view model, filled by the fixture, and is zero on a row
	// of any other author.
	Tool Tool
}

// State is the view state of the shell.
type State struct {
	Sessions []Session
	Selected string
	// Branches is read-only observed branch metadata for the selected session.
	// The daemon branch directory RPC is blocked/unverified, so this view does
	// not switch or create branches.
	Branches       []Branch
	SelectedBranch string

	// Connection, ConnectionError and ConversationLoading stand in for what
	// the App Server supervisor reports: its connection, the last error it
	// raised and whether history is still loading. They are throwaway view
	// state, not a source of facts, and this package reads no daemon state,
	// SQLite, JSONL or log to fill them.
	Connection          Connection
	ConnectionError     string
	ConversationLoading bool
	// Run is what the selected session is doing, as the supervisor reports it.
	// The shell draws it and offers no stop while nothing runs; it never
	// derives a run from the rows it shows.
	Run RunState
	// Transient model progress and notices are throwaway status-row inputs.
	ModelProgress *ModelProgress
	Notices       []Notice
	// ActionNotices and ActionError are window-owned outcomes of actions this
	// shell sent (submit/stop/select/reconnect). They are not session projection
	// facts, so a live refresh must leave them alone.
	ActionNotices []Notice
	ActionError   string

	// Messages are fixture conversation rows, not session facts.
	Messages []Message
	Draft    string

	// OnSelect, OnSubmit, OnStop and OnReconnect connect the shell's actions to
	// the caller's session controller. A nil field leaves its action in the view
	// without an effect, as a fixture does; the callback owns the App Server call
	// and the facts it returns, so the shell never reports an action it did not
	// send.
	OnSelect    func(sessionID string)
	OnSubmit    func(text string)
	OnStop      func()
	OnReconnect func()

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
			s.sidebarActions(c, k)
			ui.Sidebar(c, &s.Selected, func() {
				for _, session := range s.Sessions {
					ui.SidebarItem(c, session.ID, nil, session.Title)
				}
			}).Grow(1).Label("Sessions").OnChange(s.selectSession)
			if len(s.Branches) > 0 {
				s.branchList(c, k)
			}
			s.capabilityCards(c, k)
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

// sidebarActions holds the session list's actions above it. Creating a session
// needs session.create, which the Go client exposes but the shell does not call
// yet, so the entry stays visible and unavailable rather than disappearing.
func (s *State) sidebarActions(c *ui.Context, k tokens) {
	ui.Row(c).PaddingX(k.sidebarPadX).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Grow(1)
		wipAction(c, k, "New session", reasonCreateSession)
	})
}

// branchList shows the branch metadata supplied by the App Server as read-only
// rows. Branch switching and cut point are not wired yet, so the list offers no
// switch or side-chat action.
func (s *State) branchList(c *ui.Context, k tokens) {
	ui.Column(c).Padding(k.gap, k.sidebarPadX).Gap(k.gap).
		Role(ui.RoleGroup).Label("Branches (read-only)").Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Branches").FontSize(k.smallSize).TextColor(k.muted).Bold().AccessHidden(true)
			ui.Box(c).Grow(1)
			ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.muted).Unselectable()
		})
		for _, branch := range s.Branches {
			label := branch.Label
			if label == "" {
				label = branch.ID
			}
			if branch.Selected || branch.ID == s.SelectedBranch {
				label += " (current)"
			}
			ui.Text(c, label).FontSize(k.smallSize).TextColor(k.muted).NoWrap()
		}
		ui.Text(c, reasonBranches).FontSize(k.smallSize).TextColor(k.muted)
	})
}

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
		if s.Selected != "" && s.Run == RunRunning {
			if s.OnStop == nil {
				wipAction(c, k, "Stop", reasonStop)
			} else {
				ui.Button(c, "Stop").OnClick(s.stopRun)
			}
		}
		// A healthy connection has nothing to reconnect, a reconnect in flight
		// already shows its loading state, and with no selected session there is
		// no attachment to re-open.
		if s.Selected != "" && s.Connection != ConnReady && !s.ConversationLoading {
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
			Background(k.runColor(s.Run)).Unselectable()
		ui.Text(c, "Session: "+s.Run.String()).FontSize(k.smallSize).TextColor(k.muted)
		if s.ModelProgress != nil {
			text := s.ModelProgress.Text
			if text == "" {
				text = s.ModelProgress.Thinking
			}
			if text != "" {
				ui.Text(c, text).FontSize(k.smallSize).TextColor(k.muted)
			}
		}
		if notices := len(s.Notices) + len(s.ActionNotices); notices > 0 {
			ui.Text(c, "Notices: "+strconv.Itoa(notices)).FontSize(k.smallSize).TextColor(k.muted)
		}
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

// banner shows the last window-owned action error, or the supervisor's last
// connection error, in a live region assistive technology reads out as it
// appears. Connection failures take precedence over older action failures so
// the latest transport problem remains visible; live refreshes still leave
// ActionError alone.
func (s *State) banner(c *ui.Context, k tokens) {
	message := ""
	if s.Connection == ConnError && s.ConnectionError != "" {
		message = s.Connection.String() + ": " + s.ConnectionError
	} else {
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
// for the assistant, as plain text for the user, as an approval request with the
// decisions the shell cannot take yet, or as a tool call's card.
func (s *State) messageRow(c *ui.Context, k tokens, m Message) {
	ui.Column(c).Padding(k.rowPadY, k.rowPadX).Gap(k.gap).Children(func() {
		ui.Text(c, m.Author.String()).FontSize(k.smallSize).TextColor(k.muted).Unselectable()
		switch m.Author {
		case AuthorAssistant:
			markdown.View(c, m.Content, &s.markdown)
		case AuthorApproval:
			ui.Text(c, m.Content).FontSize(k.bodySize)
			s.approvalActions(c, k)
		case AuthorTool:
			s.toolRow(c, k, m)
		default:
			ui.Text(c, m.Content).FontSize(k.bodySize)
		}
	})
}

// approvalActions are the decisions an approval row asks for. The shell does
// not call approval.respond yet, so both entries stay visible, dimmed and
// unavailable: a click cannot report a decision the App Server never received.
func (s *State) approvalActions(c *ui.Context, k tokens) {
	ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		wipAction(c, k, "Approve", reasonApproval)
		wipAction(c, k, "Deny", reasonApproval)
	})
}

// composer holds the draft, the attachment entry and the send entry. It is a
// TextArea for now; IME composition is not claimed by this shell, only checked
// headless as draft text that survives a rebuild. ComposerShortcutMods with
// ComposerShortcutKey gives it the focus; the same modifier with Enter submits
// the draft.
func (s *State) composer(c *ui.Context, k tokens) {
	ui.Column(c).Padding(k.edgePad).Gap(k.gap).Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			wipAction(c, k, "Attach file", reasonAttachment)
			ui.Box(c).Grow(1)
			send := ui.Button(c, "Send").OnClick(s.submitDraft)
			if !s.canSubmit() {
				send.Disabled(true)
			}
		})
		ui.TextArea(c.Key("composer"), &s.Draft).Bind(&s.composerFocus).
			Height(k.composerH).Placeholder("Message DimAgent").Label("Composer").
			OnShortcut(ComposerShortcutMods, ui.KeyEnter, s.submitDraft)
	})
}

// selectSession tells the caller the sidebar chose another session. The shell
// changed Selected first; the caller opens that session and reports what it
// finds.
func (s *State) selectSession() {
	if s.OnSelect != nil {
		s.OnSelect(s.Selected)
	}
}

// canSubmit reports whether the shell has a selected session on a ready
// connection, text to send, and a callback to send it through. A fixture
// without those cannot pretend a submission the App Server never received.
func (s *State) canSubmit() bool {
	return s.OnSubmit != nil && s.Connection == ConnReady && !s.ConversationLoading &&
		s.Selected != "" && strings.TrimSpace(s.Draft) != ""
}

// submitDraft sends the draft through the caller's submit callback when the
// shell can submit. The shell leaves the draft in place: the caller clears it
// once the App Server accepts the submission, so a failed send keeps the text.
func (s *State) submitDraft() {
	if !s.canSubmit() {
		return
	}
	s.OnSubmit(strings.TrimSpace(s.Draft))
}

// stopRun asks the caller to stop the running turn.
func (s *State) stopRun() {
	if s.OnStop != nil {
		s.OnStop()
	}
}

func (s *State) reconnect() {
	if s.OnReconnect != nil {
		s.OnReconnect()
	}
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
