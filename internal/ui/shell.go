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
	// Group is the heading the sidebar draws the row under, as a display
	// label the window derives from a catalog fact, such as the project of
	// session.catalog's cwd or a date bucket of updatedAt. The shell keeps
	// no grouping fact of its own and renders one flat list while every
	// Group is empty; it never invents a project name.
	Group string
}

// Branch is read-only branch metadata supplied by the App Server. Branch
// switching and cut point remain disabled, so this is not an action list.
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
	// projection: plain text with its two decisions. The decisions are real
	// actions only while the window supplies OnApproval and the request keeps
	// its App Server identity; otherwise they stay visible WIP entries.
	AuthorApproval
	// AuthorTool is a tool call, drawn as a compact, expandable card with its
	// name, its status, and its params and result in a bounded panel.
	AuthorTool
	// AuthorError is a turn that ended in failure, drawn as its own row in the
	// danger color inside an alert region. The row's text is the caller's; the
	// shell adds no reason of its own.
	AuthorError
	// AuthorRetry is a retry notice, drawn as its own row in the warning color:
	// a turn was retried after it failed.
	AuthorRetry
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
	case AuthorError:
		return "Error"
	case AuthorRetry:
		return "Retry"
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

// ApprovalDecision is the shell's decision vocabulary for one pending
// approval. It is not the wire value: the session layer folds it onto
// approval.respond's allow/deny and reports what the App Server answered.
type ApprovalDecision string

const (
	ApprovalAllow ApprovalDecision = "allow"
	ApprovalDeny  ApprovalDecision = "deny"
)

// Approval is what an AuthorApproval row needs to offer its decisions: the
// App Server's identity of the request, copied from the session projection.
// The row holds no state of its own; whether the request is still pending comes
// from the next projection frame, not from a click.
type Approval struct {
	BranchID  string
	RequestID string
	Call      string
	Reason    string
	Source    string
}

type Message struct {
	// ID keys the list row, so a row's state and the list's place survive
	// rows being added, removed or measured.
	ID string
	// Author decides whether the row renders as Markdown, plain text or an
	// approval request with its decisions.
	Author Author
	// Content is the row's text: Markdown for an assistant row whose Blocks are
	// empty, plain text for a user, approval, error or retry row.
	Content string
	// Blocks are an assistant turn's parts in the projection's order: text,
	// thinking, tools and file changes. A row with no blocks renders Content as
	// one Markdown block, so a fixture or an adapter that keeps only text still
	// draws the turn.
	Blocks []Block
	// Tool is the call of an AuthorTool row: its name, previews and status.
	// It stays a pure view model, filled by the fixture, and is zero on a row
	// of any other author.
	Tool     Tool
	Approval *Approval
}

// State is the view state of the shell.
type State struct {
	Sessions []Session
	Selected string
	// SidebarWidth is the width the sidebar opens at and the width a drag
	// leaves it: a desktop-local preference in DIPs, 256 by default and held
	// between 200 and 360. Zero means the default. It is not a session fact.
	SidebarWidth float32
	// Branches is read-only observed branch metadata for the selected session.
	// The directory is displayed, while switching and cut point remain disabled.
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
	// SessionClosed is the terminal session.closed health fact. It is separate
	// from the connection: a closed session may still have a healthy transport,
	// but no submit, stop or reconnect is valid.
	SessionClosed       bool
	SessionClosedReason string
	// Run is what the selected session is doing, as the supervisor reports it.
	// The shell draws it and offers no stop while nothing runs; it never
	// derives a run from the rows it shows.
	Run RunState
	// Transient model progress and notices are throwaway status-row inputs.
	ModelProgress *ModelProgress
	Notices       []Notice
	// ActionNotices and ActionError are window-owned outcomes of actions this
	// shell sent (create/submit/stop/select/reconnect). They are not session
	// projection facts, so a live refresh must leave them alone.
	ActionNotices []Notice
	ActionError   string
	// ActionStatus is the window-owned line for an outcome the App Server
	// confirmed without a session fact to show, such as a submission resolved by
	// reading it back. It is not projection state either: a later action that
	// settles another outcome replaces it, and a selection or a creation clears
	// it so it never describes an older action.
	ActionStatus string
	// CreatingSession is the window's own record that session.create is in
	// flight. It holds no session fact; it only keeps the entry from being
	// clicked twice while one request is unanswered.
	CreatingSession bool

	// Messages are fixture conversation rows, not session facts.
	Messages []Message
	Draft    string

	// OnSelect, OnCreateSession, OnSubmit, OnStop, OnReconnect and OnApproval
	// connect the shell's actions to the caller's session controller. A nil field
	// leaves its action in the view without an effect, as a fixture does; the
	// callback owns the App Server call and the facts it returns, so the shell
	// never reports an action it did not send. OnCreateSession asks the caller to
	// run session.create and to publish the new session in the catalog and the
	// selection once the App Server returned it. OnApproval receives one decision
	// for one request: the caller sends approval.respond and lets the session
	// projection settle the row.
	OnSelect        func(sessionID string)
	OnCreateSession func()
	OnSubmit        func(text string)
	OnStop          func()
	OnReconnect     func()
	OnApproval      func(branchID, requestID string, decision ApprovalDecision)

	// conversation is the list's place among its rows and the identity of each
	// row, kept across frames so that appended rows follow the end and rows keep
	// their state as the conversation changes.
	conversation ui.ListState
	// sidebarList is the thread list's place and focus, sidebarChoice is the
	// row it shows as chosen, sidebarSynced is the session that choice was
	// last mapped from, and sidebarRows is the layout the choice indexes into.
	// They are the list's own view state, not session facts.
	sidebarList   ui.ListState
	sidebarChoice int
	sidebarSynced string
	sidebarRows   []sidebarRow
	// markdown is the shared Markdown options of every assistant row, whose
	// theme follows the window's.
	markdown markdown.Options
	// composerFocus is the composer's identity, for the shortcut that moves the
	// keyboard focus to it.
	composerFocus ui.Handle
	// expandedBlocks is the shell's own fold state of thinking and tool blocks,
	// keyed by row and block (see blockOpen). It holds no timeline fact: the
	// projection does not own whether a user opened a fold, so a rebuild keeps
	// it here. It lives as long as the window; a fold left behind by a row that
	// left the timeline stays as one unused entry and is never re-keyed.
	expandedBlocks map[string]*bool
}

// View builds one frame of the AppFrame: sidebar, toolbar, banner, conversation
// and composer.
func (s *State) View(c *ui.Context) {
	k := newTokens(c.Theme())
	// The divider writes the width it drags to; the shell keeps the sidebar
	// inside its own bounds before every frame, so a drag can never take the
	// session list or the conversation off the window.
	s.SidebarWidth = k.resolveSidebarWidth(s.SidebarWidth)
	ui.Split(c, &s.SidebarWidth, func() {
		s.sidebar(c, k)
	}, func() {
		s.main(c, k)
	}).SplitReset(k.sidebarW).Fill()
	if c.Shortcut(ComposerShortcutMods, ComposerShortcutKey) {
		s.composerFocus.Focus()
	}
}

// main is the conversation area beside the sidebar: toolbar, banner,
// conversation list and composer.
func (s *State) main(c *ui.Context, k tokens) {
	ui.Column(c).FillHeight().Children(func() {
		s.header(c, k)
		s.banner(c, k)
		s.conversationList(c, k)
		s.composer(c, k)
	})
}

// createSession asks the window to create a session. The shell adds no row and
// selects nothing: the new session appears only when the window publishes what
// session.create returned, so a click the App Server rejected leaves the
// catalog as it was.
func (s *State) createSession() {
	if s.OnCreateSession == nil || s.CreatingSession {
		return
	}
	s.OnCreateSession()
}

// branchList shows the branch metadata supplied by the App Server as read-only
// rows. Branch switching and cut point remain disabled, so the list offers no
// switch or side-chat action.
func (s *State) branchList(c *ui.Context, k tokens) {
	ui.Column(c).Padding(k.gap, k.sidebarPadX).Gap(k.gap).
		Role(ui.RoleGroup).Label("Branches (read-only)").Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Branches").FontSize(k.smallSize).TextColor(k.mutedForeground).Bold().AccessHidden(true)
			ui.Box(c).Grow(1)
			ui.Text(c, wipBadge).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
		})
		for _, branch := range s.Branches {
			label := branch.Label
			if label == "" {
				label = branch.ID
			}
			if branch.Selected || branch.ID == s.SelectedBranch {
				label += " (current)"
			}
			ui.Text(c, label).FontSize(k.smallSize).TextColor(k.mutedForeground).NoWrap()
		}
		ui.Text(c, reasonBranches).FontSize(k.smallSize).TextColor(k.mutedForeground)
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

// approvalRow draws a pending approval request: its author line, its text and
// the two decisions it asks for.
func (s *State) approvalRow(c *ui.Context, k tokens, m Message) {
	ui.Column(c).Padding(k.rowPadY, k.timelinePadX).Gap(k.gap).Children(func() {
		ui.Text(c, m.Author.String()).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
		ui.Text(c, m.Content).FontSize(k.bodySize)
		s.approvalActions(c, k, m)
	})
}

// approvalActions are the decisions an approval row asks for. Both are real
// actions only when the window wires OnApproval and the row still carries the
// App Server's request identity; otherwise they stay visible, dimmed and
// unavailable, so a click cannot report a decision the App Server never
// received. The callback sends one decision and the row stays until the
// session projection reports the resolution: the shell never removes it or
// claims an outcome on its own.
func (s *State) approvalActions(c *ui.Context, k tokens, m Message) {
	ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
		if m.Approval == nil || m.Approval.RequestID == "" || s.OnApproval == nil {
			wipAction(c, k, "Approve", reasonApproval)
			wipAction(c, k, "Deny", reasonApproval)
			return
		}
		approval, decide := *m.Approval, s.OnApproval
		ui.Button(c, "Approve").OnClick(func() { decide(approval.BranchID, approval.RequestID, ApprovalAllow) })
		ui.Button(c, "Deny").OnClick(func() { decide(approval.BranchID, approval.RequestID, ApprovalDeny) })
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
		!s.SessionClosed && s.Selected != "" && strings.TrimSpace(s.Draft) != ""
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
