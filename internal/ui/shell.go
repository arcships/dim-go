// Package ui builds the DimAgent native view tree from plain Go view state.
//
// It is the static shell of Milestone 1: a session sidebar, a keyed conversation
// list and a composer. It holds no session, run, timeline or
// approval facts; the real catalog, timeline and connection come from the App
// Server and its supervisor, so everything here is throwaway view state that
// the fixture fills by hand.
//
// The window is drawn by five regions — sidebar, header, timeline, approval row
// and composer — and this file is the only assembler: it copies State into each
// region's view struct, builds the Actions, and calls the render entries in
// draw order. A region reads its own view and the tokens, never State; the
// contract and its invariants are in protocols.go and
// docs/ui-component-protocols.md.
//
// Actions the shell cannot carry out yet stay visible and unavailable: they are
// marked in the accessibility tree, dimmed, and explained with the upstream
// method they wait on, and no view reports a click on one. See wipAction.
package ui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// ComposerShortcutKey is the key of the shell's shortcut that moves the
// keyboard focus to the composer, wherever the focus is in the window: Cmd+L
// on macOS, Ctrl+L elsewhere (ui.Cmd). A menu item can take the same pair once
// the shell has a menu bar.
const ComposerShortcutKey = ui.KeyL

// ComposerShortcutMods is the modifier of ComposerShortcutKey: the platform's
// own shortcut modifier.
var ComposerShortcutMods = ui.Cmd

// composerSubmitKey is the key that submits the draft from inside the composer.
const composerSubmitKey = ui.KeyEnter

// composerPlaceholder is the composer's empty-state text in the default
// language. The frame draws the active locale's text, whose English entry is
// this constant.
const composerPlaceholder = "Message DimAgent"

// conversationTopPad is the room above the first conversation row, in DIPs:
// the reference's first user bubble sits at y≈74 in an 800×720 window, 23 DIPs
// below this shell's old first row. It is a measured literal, not a spacing
// step, and belongs to the conversation rather than to every row.
const conversationTopPad = 23

// conversationInsetX is the room from the card's inner edge to the
// conversation's content, in DIPs: the reference's card runs from x=256.5 to
// x=796, starts its assistant text at x≈281 and ends its user bubble at x≈771,
// 24.5 DIPs inside the card's outer edge. One DIP of that is the card's own
// hairline, so the bar's breadcrumb and the timeline rows keep 23.5. The bar's
// trailing buttons keep their own px-3 inset, as the reference's do. The
// timeline rows already carry their own 14 DIP px-4 step, so the shell adds only
// the difference around the list. It is a measured literal, not a spacing step.
const conversationInsetX = 23.5

// conversationPadX is conversationInsetX at the window's density: it keeps the
// reference's 24.5 DIP edge at the regular step and tightens it with the timeline's
// own px-4 step in a compact window, so a density change keeps the bar and the
// rows in step instead of pinning one of them.
func (k tokens) conversationPadX() float32 {
	return k.timelinePadX * conversationInsetX / timelinePadX
}

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
	// UpdatedAt is session.catalog's updatedAt in epoch milliseconds, shown as
	// the row's relative time. Zero draws no time.
	UpdatedAt int64
}

// Branch is read-only branch metadata supplied by the App Server. Switching,
// ending and cut point are separate actions; this struct carries no control.
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
	// InputDisplay is the request's tool input, formatted for reading. It is
	// empty when the App Server sent no input; the panel then draws no block.
	InputDisplay string
	// Variant is the response option the request offers, when the window was
	// told one. The wire carries no options list, so this stays Unspecified
	// unless a caller proves a shape from the public contract.
	Variant ApprovalVariant
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
	// Sequence is the raw App Server event sequence that created a user row's
	// input. It is the durable fact a "Branch from here" cut point resolves
	// from: the row offers the entry only when it carries one. Rows of other
	// authors carry zero.
	Sequence int64
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
	// Search is the find bar's mark on this row, or nil when the query's current
	// match is elsewhere. The shell fills it on the copy it builds for the frame;
	// the projection reports no such fact, and no region searches the rows itself.
	Search *MessageSearch
}

// Route is the page the window shows in place of the conversation. Chat is the
// normal shell; the other routes take the whole window, as the reference's
// settings and management pages do. A route is display state only: switching it
// sends no App Server request and leaves the selection, draft and conversation
// list state untouched.
type Route uint8

const (
	// RouteChat is the conversation shell: sidebar, toolbar, timeline and
	// composer.
	RouteChat Route = iota
	// RouteSettings is the desktop-local appearance surface. It is the only
	// route that edits anything; the others are WIP pages.
	RouteSettings
	// RouteScheduled is the scheduled-tasks page. The shell has no client or
	// App Server method for it, so the page states the gap.
	RouteScheduled
	// RouteSkills is the skills and plugins page. The shell has no client or
	// App Server method for it, so the page states the gap.
	RouteSkills
)

// String is the route's name, for a log or a test.
func (r Route) String() string {
	switch r {
	case RouteSettings:
		return "settings"
	case RouteScheduled:
		return "scheduled"
	case RouteSkills:
		return "skills"
	default:
		return "chat"
	}
}

// ModelStatus is the window's record of the last model-catalog read. Available
// is false while the App Server does not advertise provider.management or the
// read failed, and Reason then names the gap; the composer shows the reason in
// place of a model name rather than inventing one. Selectable records whether
// the App Server advertises session.model.update: without it the list's rows stay
// disabled and explained as WIP even though the catalog itself is readable.
type ModelStatus struct {
	Available  bool
	Selectable bool
	Reason     string
}

// ModelChoice is one model as the App Server catalog serves it, copied for
// display. Enabled is the wire's enabled flag folded with the model's missing
// limits: a model whose declared limits are incomplete is drawn as unavailable,
// because the picker must not present it as selectable. Reason names why.
// Current marks the pair the attached session reported.
type ModelChoice struct {
	ProviderID   string
	ProviderName string
	ModelID      string
	DisplayName  string
	// ContextWindow and MaxOutputTokens are the model's declared limits, nil
	// when the App Server declared none. They are display facts copied from the
	// wire; a reader prints n/a for a nil value rather than guessing one.
	ContextWindow   *int64
	MaxOutputTokens *int64
	// ToolCalling, Vision and Reasoning are the model's declared capabilities,
	// as the catalog serves them. A model whose row the catalog did not carry
	// has no such fact, so the debug text writes n/a there instead of reading
	// this struct's zero value.
	ToolCalling bool
	Vision      bool
	Reasoning   bool
	// Source is the catalog's source label for the model, empty when the App
	// Server named none.
	Source  string
	Enabled bool
	Reason  string
	Current bool
}

// ModelCatalog is the model snapshot the composer displays: the current
// provider/model pair, whether the attach reply reported one, and the catalog's
// choices. A selection is written by the caller's OnSelectModel action; the
// window stores the pair only from App Server facts.
type ModelCatalog struct {
	Current    string
	HasCurrent bool
	// CurrentProviderID and CurrentModelID are the pair Current is made from,
	// so a reader can name the two halves separately even when the catalog
	// carries no row for it. They are empty while HasCurrent is false. Current
	// stays the display string the composer chip shows.
	CurrentProviderID string
	CurrentModelID    string
	Choices           []ModelChoice
}

// State is the view state of the shell. It is the window's own record plus
// display copies of what the App Server reported; no region reads it directly.
// The shell copies it into a view struct per region and builds the Actions from
// the callbacks below (see protocols.go).
type State struct {
	Sessions []Session
	Selected string
	// SidebarWidth is the width the sidebar opens at and the width a drag
	// leaves it: a desktop-local preference in DIPs, 256 by default and held
	// between 200 and 360. Zero means the default. It is not a session fact.
	SidebarWidth float32
	// Branches is read-only observed branch metadata for the selected session.
	// The directory is displayed; switching, ending and cut point are requests
	// the window sends through the callbacks below, never local edits.
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
	// ClosedSessionTitle and ClosedSessionProject are display copies of the
	// catalog labels of a session whose delete was confirmed. The row leaves the
	// catalog, but the toolbar keeps naming the session while the main area shows
	// its terminal closed state. They are display state, never session facts.
	ClosedSessionTitle   string
	ClosedSessionProject string
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
	// Workspace and Permission are the composer context row's display copies:
	// the workspace the next turn would run in and the session's default
	// permission mode, as the App Server reported them. The shell has no such
	// fact yet, so they stay empty and the composer draws its WIP state; they
	// are never filled from a local guess.
	Workspace  string
	Permission string

	// OnSelect, OnSelectBranch, OnBranchFrom, OnEndBranch, OnRefreshBranches,
	// OnCreateSession, OnSubmit, OnStop, OnReconnect and OnApproval connect the
	// shell's actions to the caller's session controller. They are the only
	// wiring source of the Actions a region receives; a nil field leaves its
	// action in the view without an effect, as a fixture does, and the region
	// keeps its WIP entry. The callback owns the App Server call and the facts
	// it returns, so the shell never reports an action it did not send.
	// OnCreateSession asks the caller to run session.create and to publish the
	// new session in the catalog and the selection once the App Server returned
	// it. OnSelectBranch asks the caller to read that branch's history and to
	// show it only once the App Server returned it. OnBranchFrom asks the caller
	// to run session.branch_create for a cut point on the displayed branch and
	// to switch to the created branch only after the App Server returned it.
	// OnEndBranch asks the caller to end one branch and to remove its row only
	// after session.branches reports it gone. OnRefreshBranches asks the caller
	// to re-read that directory. OnApproval receives one decision for one
	// request: the caller sends approval.respond and lets the session projection
	// settle the row.
	OnSelect          func(sessionID string)
	OnSelectBranch    func(branchID string)
	OnBranchFrom      func(sessionID, branchID string, at int64, prompt string)
	OnEndBranch       func(branchID string)
	OnRefreshBranches func()
	OnCreateSession   func()
	OnSubmit          func(text string)
	OnStop            func()
	OnReconnect       func()
	OnApproval        func(branchID, requestID string, decision ApprovalDecision)
	// OnDeleteSession asks the caller to delete one session. The row leaves the
	// catalog only after the App Server confirmed the delete.
	OnDeleteSession func(sessionID string)
	// OnRename asks the caller to rename the selected session through
	// session.rename. An empty title clears the manual title. The caller sends
	// the write and refreshes the catalog/title; a failure is reported through
	// ActionError, and the shell never changes the title on its own.
	OnRename func(title string)
	// OnSelectModel asks the caller to switch the selected session's model
	// through session.model.update. The list closes before this runs, so one
	// click sends at most one request. The caller owns the write and the
	// read-back: a success updates the chip from the App Server's pair, a
	// failure is reported through ActionError and leaves the previous pair.
	OnSelectModel func(providerID, modelID string)
	// OnPinSession and OnArchiveSession ask the caller to toggle the selected
	// session in its desktop-local pinned or archived list. They are display
	// state only: the App Server owns the session, and a failure is reported
	// through ActionError. OnRestoreSession asks the caller to remove an id from
	// the archived list, which is what the archived-sessions page's Restore
	// entry does.
	OnPinSession     func(sessionID string)
	OnArchiveSession func(sessionID string)
	OnRestoreSession func(sessionID string)

	// Pinned and Archived are display copies of the desktop-local session id
	// lists. They carry no session fact: the sidebar orders pinned ids first,
	// hides archived ones, and the search palette groups the pinned ids. An id
	// the catalog does not hold draws nothing.
	Pinned   []string
	Archived []string

	// conversation is the list's place among its rows and the identity of each
	// row, kept across frames so that appended rows follow the end and rows keep
	// their state as the conversation changes.
	conversation ui.ListState
	// sidebar is the thread list's place, choice and row layout: the list's own
	// view state, not session facts. See sidebarState.
	sidebar sidebarState
	// composerFocus is the composer's identity, for the shortcut that moves the
	// keyboard focus to it.
	composerFocus ui.Handle
	// folds is the shell's own fold state of thinking and tool blocks, keyed by
	// row and block (see blockFolds.state). It holds no timeline fact: the
	// projection does not own whether a user opened a fold, so a rebuild keeps it
	// here. It lives as long as the window; a fold left behind by a row that left
	// the timeline stays as one unused entry and is never re-keyed.
	folds blockFolds
	// ShowContextPanel is the window's own record that the context panel is
	// showing beside the conversation. It is display state only and holds no
	// workspace, file or diff fact.
	ShowContextPanel bool
	// ContextPanel is the changed-file tree and the selected file's detail, as
	// the caller read them. It is a display copy: every node, diff line and
	// preview is the caller's, and a zero view keeps FilesReported false, so
	// the panel draws its WIP statement instead of inventing files. A live
	// refresh replaces the copy and leaves ShowContextPanel alone.
	ContextPanel ContextPanelView
	// ShowSettings is the window's own record that the settings route is
	// showing instead of the shell. It is display state only: while it is set,
	// View draws the route in the whole window and no sidebar, toolbar,
	// conversation or composer. Closing it restores the frame the window held.
	// It mirrors RouteSettings for callers that predate Route; setting it alone
	// still opens the appearance surface.
	ShowSettings bool
	// Route is the page the window shows. RouteChat is the shell, RouteSettings
	// the appearance surface, and RouteScheduled/RouteSkills the WIP pages the
	// sidebar's management entries open. It is display state only: the
	// selection, draft and conversation list state survive every switch.
	Route Route
	// ModelCatalog and ModelStatus are display copies of the App Server model
	// facts the caller read through the session controller. The composer draws
	// the current pair and the catalog; it never writes a selection from them.
	ModelCatalog ModelCatalog
	ModelStatus  ModelStatus
	// ModelListOpen is the window's own record that the composer's model list
	// is showing. It is display state only; closing it sends nothing, and a
	// row sends a selection only through the caller's OnSelectModel.
	ModelListOpen bool
	// RenameConfirm is the window's own record that the toolbar's rename dialog
	// is showing, and RenameDraft is the text its field edits. RenameConfirm is
	// dropped when the selection changes, so one session's dialog never renames
	// another.
	RenameConfirm bool
	RenameDraft   string
	// renameFor is the selection RenameConfirm was opened for. See
	// syncRenameConfirm.
	renameFor string
	// BranchFromConfirm is the window's own record that a user row's "Branch
	// from here" dialog is showing, and BranchFromDraft is the prompt its field
	// edits. branchFromAt is the parent sequence the row resolved and
	// branchFromFor/branchFromBranch are the row's session and displayed branch;
	// a window that changes either drops the dialog so one branch's cut point is
	// never sent for another.
	BranchFromConfirm bool
	BranchFromDraft   string
	branchFromAt      int64
	branchFromFor     string
	branchFromBranch  string
	// ShowSearch is the window's own record that the centered search palette is
	// showing over the frame. It is display state only: the palette draws above
	// the sidebar and main column without replacing them, and Cmd+G or the
	// sidebar's Search entry opens it. SearchQuery is the text its input edits
	// and SearchSelected the result its arrows highlight; both are view state
	// the palette writes in place, and a closed palette clears them so the next
	// open starts from the whole recent list. No field here is a session fact:
	// the results are filtered from Sessions and a chosen row goes through
	// OnSelect.
	ShowSearch     bool
	SearchQuery    string
	SearchSelected int
	// searchSynced is the query SearchSelected was last held against: when the
	// input's text changes, the highlight returns to the first result instead of
	// pointing into a list the query no longer produces.
	searchSynced string
	// ShowThreadSearch is the window's own record that the conversation's find
	// bar is showing in the toolbar, and ThreadSearchQuery is the text its field
	// edits. They are display state only: Cmd+F opens the bar, its matches are
	// counted from the rows the window already displays, and closing it forgets
	// the query and the highlight. No field here is a session fact, and nothing
	// is sent to the App Server.
	ShowThreadSearch  bool
	ThreadSearchQuery string
	// ThreadSearchIndex is the match the timeline highlights, as an index into
	// threadSearchFound; -1 while the query matches nothing, so the bar reads
	// "0/0" rather than pointing at a match that is not there.
	ThreadSearchIndex int
	// threadSearchSynced is the query ThreadSearchIndex was last held against:
	// changed text returns the highlight to the first match instead of pointing
	// into matches the query no longer produces. threadSearchFound is the match
	// list counted for the frame the bar and the timeline draw from, and
	// threadSearchRow the row it was last scrolled to, so a frame that keeps the
	// same match does not fight the user's scrolling.
	threadSearchSynced string
	threadSearchFound  []threadMatch
	threadSearchRow    int
	// threadSearchFocus is the field's identity, so Cmd+F again and a step entry
	// put the keyboard focus back in it. threadSearchReturn is the control the
	// bar took the focus from, to give it back as the bar closes.
	threadSearchFocus  ui.Handle
	threadSearchReturn *ui.Handle
	// threadSearchFor is the selection the bar was opened on and
	// threadSearchTracked says whether that comparison has started: a change of
	// session closes the bar, so one session's matches never highlight another's
	// rows. See syncThreadSearchSession.
	threadSearchFor     string
	threadSearchTracked bool
	// DeleteConfirm is the window's own record that the toolbar's delete
	// confirmation is showing. It holds no session fact: the alert is confirmed
	// through OnDeleteSession, and the row leaves the catalog only after the App
	// Server confirmed the delete. It is dropped when the selection changes, so
	// one session's confirmation never addresses another.
	DeleteConfirm bool
	// deleteFor is the selection DeleteConfirm was opened for. See
	// syncDeleteConfirm.
	deleteFor string
	// Settings holds the desktop-local appearance and language preferences the
	// surface reports. They are the desktop's, not a session's: nothing here is
	// sent to the App Server or stored as a session fact. Settings.Locale is the
	// language every region draws on the frame after a choice.
	Settings SettingsView
	// ShowKeys is the window's own record that the session keys browser is
	// showing instead of the conversation. It is display state only.
	ShowKeys bool
	// Keys is the session.keys snapshot the caller read for the selected
	// session, plus the browser's own selection and filter. The shell passes it
	// to the panel and draws no key of its own; the panel writes only Selected
	// and Query, which are view state. Nothing here is sent to the App Server.
	Keys KeysPanelView
	// keysSelected is the browser's selected key. Keys.Selected points at it on
	// every frame, so the panel's click writes here and the selection survives
	// the caller refreshing the snapshot.
	keysSelected string
	// keysSession is the selection the Keys snapshot was last assembled for and
	// keysTracked says whether that comparison has started. The first frame
	// adopts the caller's selection together with the snapshot it supplied; a
	// later selection change drops the snapshot and closes the browser, so one
	// session's keys are never shown for another.
	keysSession string
	keysTracked bool
}

// View builds one frame of the AppFrame: sidebar, toolbar, banner, conversation
// and composer. It only assembles each region's view and the Actions, then
// calls the region's render entry; it draws no region itself.
func (s *State) View(c *ui.Context) {
	// The appearance preference forces a theme when it is light or dark; system
	// leaves the window's own appearance alone. The tokens are then read from the
	// theme in force, so a forced theme carries through every region.
	switch s.Settings.Theme {
	case ThemeLight:
		c.SetTheme(ui.LightTheme())
	case ThemeDark:
		c.SetTheme(ui.DarkTheme())
	}
	// The saved appearance builds the frame's theme and tokens: the type size,
	// family and density are folded into the theme, so mygo's own widgets step
	// with the shell's views, and the tokens carry the same appearance into
	// every region. Both are read again each frame, so a choice made on the
	// settings surface shows on the next one.
	k := newTokensFor(c.Theme(), s.Settings)
	c.SetTheme(k.theme)
	// The divider writes the width it drags to; the shell keeps the sidebar
	// inside its own bounds before every frame, so a drag can never take the
	// session list or the conversation off the window.
	s.SidebarWidth = k.resolveSidebarWidth(s.SidebarWidth)
	s.syncKeysSession()
	s.syncDeleteConfirm()
	s.syncRenameConfirm()
	s.syncBranchFrom()
	s.syncThreadSearchSession()
	a := s.actions()
	// The settings route takes the whole window: no sidebar, toolbar,
	// conversation, composer or status band is drawn while it is open, and a
	// close returns to the frame the window held, with its selection, draft and
	// conversation list state untouched. The search palette never shows over the
	// settings route, so a palette that was open is dropped here, and the find
	// bar, which belongs to the conversation, goes with it.
	route := s.currentRoute()
	if route == RouteSettings {
		s.closeSearch()
		s.dropThreadSearch()
		renderSettings(c, k, s.settingsView(), wipCapabilities(s), s.applySettings, s.closeSettings, s.OnRestoreSession)
		return
	}
	// The scheduled-tasks and skills pages are WIP routes: they take the whole
	// window like settings, explain the missing client or method, and keep the
	// selection and draft for the conversation the Back entry returns to. The
	// window's notices stay visible on top so an action outcome is never hidden
	// behind a page.
	if route != RouteChat {
		s.closeSearch()
		s.dropThreadSearch()
		renderWIPRoute(c, k, route, s.headerView(), a)
		return
	}
	composer := s.composerView()
	// Escape leaves the keys browser wherever the keyboard focus is. It is read
	// before the regions build so the frame already shows the conversation the
	// key returns to.
	if s.ShowKeys && c.Shortcut(0, ui.KeyEscape) {
		s.ShowKeys = false
	}
	// The find bar belongs to the conversation. The keys browser and the search
	// palette take the conversation away, so they close it; Cmd+F opens it on the
	// selected session and, pressed again, puts the focus back in its field;
	// Escape leaves it wherever the focus is. The states are the window's own and
	// the matches are counted from the rows it already displays, so nothing here
	// sends a request. s.syncThreadSearch then holds the highlight against the
	// current text and scrolls the timeline to it.
	if s.ShowKeys || s.ShowSearch || s.Selected == "" {
		s.dropThreadSearch()
	}
	if s.Selected != "" && !s.ShowKeys && !s.ShowSearch && c.Shortcut(ui.Cmd, ui.KeyF) {
		s.openThreadSearch()
	}
	if s.ShowThreadSearch && c.Shortcut(0, ui.KeyEscape) {
		s.closeThreadSearch()
	}
	s.syncThreadSearch()
	ui.Split(c, &s.SidebarWidth, func() {
		renderSidebar(c, k, s.sidebarView(), a)
		// A click on a session row moves the selection while the sidebar builds;
		// checking again here drops the old snapshot before the main column draws.
		s.syncKeysSession()
	}, func() {
		// The main column is a card inset from the window edge, as
		// dim-electron's framed chat workspace draws it: 4.5 DIPs on the top,
		// right and bottom, none on the left while the sidebar is open, and a
		// rounded face with a hairline outline. Every surface the column shows
		// (the conversation, the approval panel, the keys browser and the
		// appearance surface) is drawn inside it, so they share the frame. The
		// reference frames the same card with a 0.5 DIP outer edge
		// (global.css:35), which the shell draws as the card step of its shadow
		// scale: a shadow outside the box, so the frame keeps its measured 4.5
		// DIP gap.
		ui.Column(c).FillHeight().Padding(k.framePad, k.framePad, k.framePad, 0).Children(func() {
			mainFrame(c, k, func() {
				// The reference has no application bar in the empty state: the
				// session column's card starts at the frame and the hero carries
				// the heading. With a session the toolbar is its own region, built
				// from HeaderView. An action or connection failure still gets its
				// banner in either state, so an empty window never hides why a
				// preference save or a request failed.
				if s.Selected != "" {
					renderHeader(c, k, s.headerView(), a)
				} else {
					banner(c, k, s.headerView(), a)
				}
				// The conversation and the context panel share the row below
				// the toolbar, so the panel sits beside the session column
				// and under the same card. The conversation takes the room
				// that is left; the panel keeps its own width and gives way
				// when the window cannot show both.
				ui.Row(c).Grow(1).FillWidth().Children(func() {
					ui.Column(c).Grow(1).Shrink(1).MinWidth(0).FillHeight().Children(func() {
						timeline := s.timelineView()
						pending := s.pendingApprovals()
						switch {
						case s.ShowKeys:
							// The keys browser takes the conversation's place. It reads the
							// snapshot the caller supplied and sends no request. It keeps
							// its hairline edge rather than the popover shadow: its box is
							// the frame's own inner edge, so an outer shadow would paint
							// over the frame's outline and onto the sidebar.
							s.Keys.Selected = &s.keysSelected
							ui.Column(c).Key("keys-panel").Grow(1).FillHeight().FillWidth().
								Transition(motionPanel(c)).Children(func() {
								renderKeysPanel(c, k, &s.Keys, a)
							})
						default:
							// An empty, loaded and open session shows the hero instead of the
							// projection's empty row; a closed session or a pending approval
							// keeps the timeline's own states, so the hero never speaks over
							// facts that are still on screen.
							if !timeline.Loading && !timeline.Closed && len(timeline.Messages) == 0 && len(pending) == 0 {
								emptyState(c, k, composer, s.locale(), a)
							} else {
								// The reference's conversation starts 23 DIPs lower than
								// this shell did: its first user bubble sits at y≈74 in an
								// 800×720 window, below the 41 DIP bar and the frame. The
								// room is the conversation's, so it is added around the
								// timeline only; the composer below keeps its place.
								//
								// The timeline rows keep their own 14 DIP px-4 inset, so
								// the conversation adds only the difference to the
								// reference's 25 DIP content edge. That covers every row
								// kind, including the tool card, which is built outside
								// this file.
								ui.Column(c).Grow(1).
									Padding(conversationTopPad, k.conversationPadX()-k.timelinePadX, 0, k.conversationPadX()-k.timelinePadX).
									Children(func() {
										renderTimeline(c, k, timeline, a)
									})
								// A pending approval takes the composer's slot: the user answers the
								// request before typing again, as the spec's composer replacement does.
								// The request set comes from the rows the projection produced, never
								// from a click.
								if len(pending) > 0 {
									renderApprovalPanel(c, k, ApprovalView{
										Pending:      pending,
										InputDisplay: pending[0].InputDisplay,
										Variant:      pending[0].Variant,
									}, a)
								} else {
									renderComposer(c, k, composer, a)
								}
							}
						}
					})
					if s.contextPanelShown() {
						s.contextPanelColumn(c, k, a)
					}
				})
			})
		})
	}).SplitReset(k.sidebarW).Fill()
	if c.Shortcut(ComposerShortcutMods, ComposerShortcutKey) {
		composer.Focus.Focus()
	}
	// The search palette is the window's own display state: Cmd+G and the
	// sidebar's Search entry open it, the results come from the catalog it
	// already holds, and it starts no App Server read. It is built last so it
	// paints over the frame it opened from rather than replacing it, and it is
	// absent while the settings route is showing.
	if c.Shortcut(searchShortcutMods, searchShortcutKey) {
		s.openSearch()
	}
	s.syncSearch()
	if s.ShowSearch {
		results := searchResults(s.Sessions, s.SearchQuery, s.Pinned, s.Archived)
		s.SearchSelected = max(0, min(s.SearchSelected, len(results)-1))
		renderSearchPalette(c, k, s.searchPaletteView(results), a)
	}
	// The control the find bar gives the focus back to is recorded here, at the
	// end of every frame the bar is closed and the conversation is showing: the
	// bar opens at the top of a frame, before the regions are built, and mygo
	// answers which control has the focus from the frame that is committed, so
	// the answer has to be taken while that frame is still the last one.
	if !s.ShowThreadSearch && s.Selected != "" {
		s.threadSearchReturn = s.threadSearchReturnTarget(c)
	}
}

// mainFrame draws the card the main column's surfaces sit in: the framed
// workspace, its hairline outline and its corner, lifted by the card step of
// the shadow scale. The reference frames the same card with a 0.5 DIP outer edge
// (global.css:35); mygo paints that as a soft shadow outside the box, so the
// frame keeps its measured 4.5 DIP gap and takes no room in the layout. body
// draws the column's own surfaces inside the card.
func mainFrame(c *ui.Context, k tokens, body func()) {
	frame := ui.Column(c).Grow(1).FillWidth().
		Radius(k.frameRadius).BorderWidth(k.hairline).BorderColor(k.border).
		Background(k.card).Transition(motionLayoutColors(c)).Role(ui.RoleGroup).Label("Main area")
	k.shadowCard.paint(frame).Children(body)
}

// openSearch opens the palette from the shortcut or the sidebar entry. It
// starts from an empty query, as the reference's reopened dialog does, and
// highlights the first result; the list itself is filtered on every frame from
// State.Sessions. The find bar belongs to the conversation the palette covers,
// so the palette closes it rather than leaving it behind the dialog.
func (s *State) openSearch() {
	s.dropThreadSearch()
	s.ShowSearch = true
	s.SearchQuery = ""
	s.SearchSelected = 0
	s.searchSynced = ""
}

// closeSearch drops the palette and its query, so closing it leaves no filter
// behind and the next open starts from the whole recent list. The focus returns
// to the control that opened the palette by itself: the dialog is an overlay,
// and mygo gives the focus back to the element that had it as it opened.
func (s *State) closeSearch() {
	s.ShowSearch = false
	s.SearchQuery = ""
	s.SearchSelected = 0
	s.searchSynced = ""
}

// syncSearch keeps the highlight on a row the current results still have: a
// changed query returns it to the first row, and a closed palette clears the
// text and the highlight. The input edits the query while the frame builds, so
// this runs at the top of the next one.
func (s *State) syncSearch() {
	if !s.ShowSearch {
		s.closeSearch()
		return
	}
	if s.SearchQuery != s.searchSynced {
		s.searchSynced = s.SearchQuery
		s.SearchSelected = 0
	}
}

// searchPaletteView copies the palette's view state and the results the shell
// filtered. Open, Query and Selected are pointers: the dialog writes Open when
// it is dismissed, the input writes Query, and the arrows write Selected, so
// the shell reads them on the next frame without a second copy.
func (s *State) searchPaletteView(results []SearchResult) SearchPaletteView {
	return SearchPaletteView{
		Open:     &s.ShowSearch,
		Query:    &s.SearchQuery,
		Results:  results,
		Selected: &s.SearchSelected,
		Locale:   s.locale(),
	}
}

// emptyState draws the workspace's empty state: the hero and the composer as
// one group in the middle of the card, the composer directly under the hero, as
// the reference's NewThread lays them out. They are centred together rather
// than the hero centred alone over a composer pinned to the bottom, so the pair
// keeps its distance from the window's edges in every window height.
func emptyState(c *ui.Context, k tokens, v ComposerView, locale Locale, a Actions) {
	ui.Column(c).Grow(1).FillWidth().Justify(ui.Center).Children(func() {
		// The hero grows to fill whatever it is given (conversation.go), so the
		// group gives it exactly its own box: the mark, the gap, the heading and
		// the p-8 inset. That leaves the composer to follow it inside the
		// centered pair instead of taking the free space.
		ui.Column(c).Height(emptyHeroHeight(k)).Shrink(1).Children(func() {
			renderHero(c, k, locale)
		})
		renderComposer(c, k, v, a)
	})
}

// emptyHeroHeight is the hero's own height: renderHero's p-8 padding, the
// brand mark, the gap under it and the heading's line box. It mirrors that
// region's box, so the composer follows the hero directly.
func emptyHeroHeight(k tokens) float32 {
	return 2*k.heroPad + k.heroMarkH + k.heroGap + k.heroTitleLineH
}

// contextPanelShown reports whether the context panel is drawn this frame. It
// is the window's own toggle, and it needs a selected session because the
// panel shows that session's changed files. The appearance surface and the
// keys browser take the main column from the conversation, so the panel stays
// closed while either is showing even though its toggle keeps the choice for
// the conversation the user returns to.
func (s *State) contextPanelShown() bool {
	return s.ShowContextPanel && s.Selected != "" && s.currentRoute() == RouteChat && !s.ShowKeys
}

// contextPanelColumn draws the context panel beside the conversation in the
// width the layout gives it. The view keeps the caller's pointers and fold map,
// and only its Width is filled here, so a click still writes the caller's
// selection and folds.
func (s *State) contextPanelColumn(c *ui.Context, k tokens, a Actions) {
	width := clampContextPanelWidth(contextPanelW)
	// The panel is a side surface over the conversation, so it lifts by the
	// popover step: its shadow falls left onto the conversation, while its own
	// start hairline stays the edge the reference draws. The shadow paints
	// outside the panel's box and takes no room, so the panel keeps its width.
	panel := ui.Column(c).Key("context-panel").Width(width).Shrink(1).MinWidth(0).FillHeight().Clip().
		Transition(motionSidePanel(c))
	k.shadowPopover.paint(panel).Children(func() {
		v := s.ContextPanel
		v.Width = width
		renderContextPanel(c, k, v, a)
	})
}

// syncThreadSearchSession drops the find bar and its highlight when the selected
// session changed under it. The bar counts one session's rows, so keeping it
// would search one session's text against another's conversation; the window
// opens the next session with no bar, as it opens it with no query. The focus is
// left where the new frame puts it, since the control the bar took it from
// belonged to the conversation that is gone.
func (s *State) syncThreadSearchSession() {
	if !s.threadSearchTracked {
		s.threadSearchTracked = true
		s.threadSearchFor = s.Selected
		return
	}
	if s.threadSearchFor == s.Selected {
		return
	}
	s.threadSearchFor = s.Selected
	s.dropThreadSearch()
}

// openThreadSearch shows the find bar, or puts the focus back in its field when
// it already shows, as Cmd+F does: the query it holds stays, so a second press
// carries on where the first left off. The control the bar gives the focus back
// to is the one threadSearchReturn recorded on the last frame the bar was closed,
// taken in State.View once the regions of that frame were built: mygo answers
// which control has the focus from the frame that is committed, so it cannot be
// asked here, before this frame's regions exist.
func (s *State) openThreadSearch() {
	s.threadSearchFor = s.Selected
	s.threadSearchTracked = true
	s.ShowThreadSearch = true
	s.threadSearchFocus.Focus()
}

// threadSearchReturnTarget is the control the find bar gives the focus back to as
// it closes: the conversation's list, the sidebar's thread list, or the composer.
// Those are the three the shell can name, and they are where the keyboard usually
// was when Cmd+F came; a row they hold counts, so a click on a session row hands
// the focus back to the list around it. The composer is the fallback, since it is
// the shell's own text entry and the bar opens from the keyboard.
func (s *State) threadSearchReturnTarget(c *ui.Context) *ui.Handle {
	switch {
	case s.conversation.FocusWithin(c):
		return &s.conversation.Handle
	case s.sidebar.list.FocusWithin(c):
		return &s.sidebar.list.Handle
	default:
		return &s.composerFocus
	}
}

// closeThreadSearch leaves the find bar: it forgets the query and the highlight
// and gives the focus back to the control the bar took it from, or to the
// composer when there was none to record. Escape, the bar's close entry and the
// window's own Escape while the bar is open all end here. The conversation is
// drawn again from the rows it held, so the frame after the close is the frame
// before the open.
func (s *State) closeThreadSearch() {
	target := s.threadSearchReturn
	s.dropThreadSearch()
	if target == nil {
		target = &s.composerFocus
	}
	target.Focus()
}

// dropThreadSearch forgets the bar, its query and the highlight without touching
// the keyboard focus. It is what a change of session, page or surface uses: the
// control the bar took the focus from belongs to a frame that is no longer
// drawn, so there is nothing to give the focus back to.
func (s *State) dropThreadSearch() {
	s.ShowThreadSearch = false
	s.ThreadSearchQuery = ""
	s.ThreadSearchIndex = -1
	s.threadSearchSynced = ""
	s.threadSearchFound = nil
	s.threadSearchRow = -1
	s.threadSearchReturn = nil
}

// stepThreadSearch moves the highlight by delta through the matches the current
// query finds, wrapping from the last back to the first and from the first back
// to the last. A query that matches nothing leaves the highlight where it is, so
// an entry that cannot step claims nothing. The request to scroll to the row is
// taken again on the next frame, even when the match stays in the same row.
func (s *State) stepThreadSearch(delta int) {
	count := len(threadSearchMatches(s.timelineRows(), s.ThreadSearchQuery))
	if count == 0 {
		return
	}
	index := s.ThreadSearchIndex
	switch {
	case index < 0:
		index = 0
	case index+delta < 0:
		index = count - 1
	case index+delta >= count:
		index = 0
	default:
		index += delta
	}
	s.ThreadSearchIndex = index
	s.threadSearchRow = -1
}

// syncThreadSearch holds the highlight against the text the timeline shows. It
// counts the query's occurrences once a frame, keeps the highlight on a match
// that still exists and returns it to the first when the text changes, opens a
// folded thinking block the current match sits in so the match is on screen, and
// asks the conversation to scroll to its row. The count is a display copy of the
// loaded rows: it holds no App Server fact and starts no read.
func (s *State) syncThreadSearch() {
	if !s.ShowThreadSearch {
		s.ThreadSearchIndex = -1
		s.threadSearchSynced = ""
		s.threadSearchFound = nil
		s.threadSearchRow = -1
		return
	}
	if s.ThreadSearchQuery != s.threadSearchSynced {
		s.threadSearchSynced = s.ThreadSearchQuery
		s.ThreadSearchIndex = 0
		s.threadSearchRow = -1
	}
	rows := s.timelineRows()
	s.threadSearchFound = threadSearchMatches(rows, s.ThreadSearchQuery)
	if len(s.threadSearchFound) == 0 {
		s.ThreadSearchIndex = -1
		s.threadSearchRow = -1
		return
	}
	s.ThreadSearchIndex = max(0, min(s.ThreadSearchIndex, len(s.threadSearchFound)-1))
	match := s.threadSearchFound[s.ThreadSearchIndex]
	// A match inside a thinking block is only on screen while the block shows its
	// text, so the bar opens that fold. It stays open afterwards, as a fold the
	// user opened does: the shell keeps the fold state, not the search.
	if match.Block >= 0 && match.Row < len(rows) && match.Block < len(rows[match.Row].Blocks) {
		row := rows[match.Row]
		block := row.Blocks[match.Block]
		if block.Kind == BlockThinking {
			*s.folds.state(blockKey(row.ID, block, match.Block)) = true
		}
	}
	if match.Row != s.threadSearchRow {
		s.conversation.ScrollTo(match.Row, ui.Center)
		s.threadSearchRow = match.Row
	}
}

// syncKeysSession drops the keys snapshot and closes the browser when the
// selected session changed. The snapshot belongs to one session, so keeping it
// would show one session's keys under another; the caller re-reads session.keys
// and supplies a new KeysPanelView for the selection it shows.
func (s *State) syncKeysSession() {
	if !s.keysTracked {
		s.keysTracked = true
		s.keysSession = s.Selected
		return
	}
	if s.keysSession == s.Selected {
		return
	}
	s.keysSession = s.Selected
	s.ShowKeys = false
	s.Keys = KeysPanelView{}
	s.keysSelected = ""
}

// syncDeleteConfirm drops the delete confirmation when the selected session
// changed under it. The alert addresses one session; keeping it open would let
// its confirm button delete a different selection. A closed alert clears the
// identity it was opened for.
func (s *State) syncDeleteConfirm() {
	if !s.DeleteConfirm {
		s.deleteFor = ""
		return
	}
	if s.deleteFor != s.Selected {
		s.DeleteConfirm = false
		s.deleteFor = ""
	}
}

// applySettings stores the appearance snapshot the surface reported and hands it
// to OnSettingsChange for the window to persist. The shell applies it on the
// next frame through State.View: the forced theme is read there, newTokensFor
// folds the type size, family, code size, density and chrome palette into the
// frame's theme and tokens, and the sidebar width is held between its bounds
// before it is drawn.
func (s *State) applySettings(change SettingsChange) {
	// The build version is a fact of this binary, not a preference, so a
	// settings choice keeps it for the About page.
	appVersion := s.Settings.AppVersion
	s.Settings = SettingsView{
		Theme:        change.Theme,
		Chrome:       change.Chrome,
		FontSize:     change.FontSize,
		FontFamily:   change.FontFamily,
		SidebarWidth: change.SidebarWidth,
		Density:      change.Density,
		CodeFontSize: change.CodeFontSize,
		Locale:       change.Locale,
		AppVersion:   appVersion,
	}
	if change.SidebarWidth > 0 {
		s.SidebarWidth = change.SidebarWidth
	}
}

// closeSettings is the route's return: the Back row and Escape both call it,
// and so does the toolbar's Settings entry through its own toggle. It only
// clears the window's route flag; the selection, draft, catalog and conversation
// list state stay where they were.
func (s *State) closeSettings() {
	s.setRoute(RouteChat)
}

// currentRoute is the page the frame draws. RouteSettings is the canonical
// value for the appearance surface, but callers that predate Route set
// ShowSettings alone; such a window still opens settings rather than chat.
func (s *State) currentRoute() Route {
	if s.Route != RouteChat {
		return s.Route
	}
	if s.ShowSettings {
		return RouteSettings
	}
	return RouteChat
}

// setRoute moves the window to one page. The route is display state only: the
// selection, draft, catalog and conversation list state are untouched, so a WIP
// page's Back entry returns to the conversation the user left. Opening a page
// also closes the palette and the keys browser, which belong to the shell.
func (s *State) setRoute(route Route) {
	s.Route = route
	s.ShowSettings = route == RouteSettings
	s.ModelListOpen = false
	if route != RouteChat {
		s.ShowKeys = false
		s.closeSearch()
		s.dropThreadSearch()
	}
}

// syncRenameConfirm drops the rename dialog when the selected session changed
// under it. The dialog addresses one session; keeping it open would let its
// confirm button rename a different selection.
func (s *State) syncRenameConfirm() {
	if !s.RenameConfirm {
		s.renameFor = ""
		return
	}
	if s.renameFor != s.Selected {
		s.RenameConfirm = false
		s.renameFor = ""
		s.RenameDraft = ""
	}
}

// syncBranchFrom drops the "Branch from here" dialog when the selected session
// or displayed branch changed under it. The cut point is a parent-branch
// sequence, so sending it after either moved would create a line from another
// branch than the row belongs to. Closing clears the identity and the prompt.
func (s *State) syncBranchFrom() {
	if !s.BranchFromConfirm {
		s.branchFromFor = ""
		s.branchFromBranch = ""
		return
	}
	if s.branchFromFor != s.Selected || s.branchFromBranch != s.SelectedBranch {
		s.BranchFromConfirm = false
		s.branchFromFor = ""
		s.branchFromBranch = ""
		s.branchFromAt = 0
		s.BranchFromDraft = ""
	}
}

// dismissNotices clears the window's notice strip: the last action error, the
// confirmed action status, the action-outcome list and the transient projection
// notices. They are display state only, so dismissing them changes no App Server
// fact; a later projection frame or action publishes new ones.
func (s *State) dismissNotices() {
	s.ActionError = ""
	s.ActionStatus = ""
	s.ActionNotices = nil
	s.Notices = nil
}

// locale is the language the frame draws: the saved preference, or English
// while none is saved. It is read at the top of the frame, so a language chosen
// on the settings surface reaches every region on the next one.
func (s *State) locale() Locale {
	return resolveLocale(s.Settings.Locale)
}

// actions copies the window's callbacks into the Actions a region receives. A
// nil callback stays nil, which is how a region learns the action is a WIP gap.
func (s *State) actions() Actions {
	a := Actions{
		OnCreateSession:   s.OnCreateSession,
		OnSelect:          s.OnSelect,
		OnSelectBranch:    s.OnSelectBranch,
		OnEndBranch:       s.OnEndBranch,
		OnRefreshBranches: s.OnRefreshBranches,
		OnSubmit:          s.OnSubmit,
		OnStop:            s.OnStop,
		OnReconnect:       s.OnReconnect,
		OnApproval:        s.OnApproval,
		OnDeleteSession:   s.OnDeleteSession,
		OnPinSession:      s.OnPinSession,
		OnArchiveSession:  s.OnArchiveSession,
		// The settings toggle is the shell's own state, not a caller callback, so
		// it is always wired. Opening it takes the main column from the keys
		// browser, so the two never compete for it.
		OnOpenSettings: func() {
			if s.currentRoute() == RouteSettings {
				s.setRoute(RouteChat)
				return
			}
			s.setRoute(RouteSettings)
		},
		// The scheduled-tasks and skills entries are the shell's own WIP routes:
		// they open a whole-window page that states the missing client or App
		// Server method and sends no request, so both entries are always wired.
		OnOpenScheduled: func() { s.setRoute(RouteScheduled) },
		OnOpenSkills:    func() { s.setRoute(RouteSkills) },
		OnBackToChat:    func() { s.setRoute(RouteChat) },
		// The model list is window-owned display state: opening it sends no
		// request. Its rows call the caller's OnSelectModel when that callback is
		// wired and the App Server advertises session.model.update; otherwise
		// they stay disabled and explained.
		OnToggleModelList: func() {
			s.ModelListOpen = !s.ModelListOpen
		},
		// Notices are window-owned display state; dismissing them clears the
		// banner without touching any App Server fact.
		OnDismissNotices: s.dismissNotices,
		// Copying the conversation id is the window's own action too: the
		// region writes the clipboard, since only a frame has the window's
		// services, and the shell records the muted outcome line here. It sends
		// no request and stores no session fact.
		OnCopySessionID: func(string) {
			s.ActionStatus = copyIDStatus
		},
		// The conversation link is the window's own text too: it is the selected
		// id in the scheme the app registers, so the region builds and copies it
		// and the shell records the muted outcome line. It sends no request.
		OnCopyConversationLink: func() {
			s.ActionStatus = copyLinkStatus
		},
		// Copying the debug info is the window's own action too: the region
		// writes the text it built from the model facts, and the shell records
		// the muted outcome line. It sends no request.
		OnCopyDebugInfo: func() {
			s.ActionStatus = copyDebugStatus
		},
		// The context panel toggle is the shell's own state too: the panel
		// displays the caller's ContextPanel copy and sends no request, so the
		// entry needs no caller callback. It stays wired and the toolbar offers
		// it only while a session is selected and the conversation is showing.
		OnToggleContextPanel: func() {
			s.ShowContextPanel = !s.ShowContextPanel
		},
		// The search palette is the shell's own state: its results are a
		// display copy of the catalog the window already holds and opening it
		// sends nothing, so the sidebar's Search entry is always wired.
		OnOpenSearch: s.openSearch,
		// The find bar is the window's own state too: it counts the rows the
		// conversation already shows and sends no request, so its entries are
		// always wired. A step moves the highlight and asks for the row it lands
		// on; closing forgets the query and the highlight and gives the focus back.
		OnThreadSearchNext:     func() { s.stepThreadSearch(1) },
		OnThreadSearchPrevious: func() { s.stepThreadSearch(-1) },
		OnThreadSearchClose:    s.closeThreadSearch,
	}
	// "Branch from here" is the window's own dialog: a row asks for it with the
	// cut point it resolved, and the dialog's confirm calls the caller once. It
	// is wired exactly while the caller supplied OnBranchFrom; without it the
	// row keeps a disabled WIP entry, so no region can claim a branch the window
	// never asked for.
	if s.OnBranchFrom != nil {
		a.OnRequestBranchFrom = func(at int64) {
			if s.Selected == "" || at < 0 {
				return
			}
			s.branchFromAt = at
			s.branchFromFor = s.Selected
			s.branchFromBranch = s.SelectedBranch
			s.BranchFromDraft = ""
			s.BranchFromConfirm = true
		}
		a.OnBranchFrom = func(at int64, prompt string) {
			if at < 0 {
				return
			}
			s.OnBranchFrom(s.Selected, s.SelectedBranch, at, prompt)
		}
	}
	// The model selection is wired only while the caller supplied a callback and
	// the App Server advertised session.model.update. A daemon without the
	// method keeps every row disabled, so the list states the WIP gap instead of
	// sending a write the daemon would refuse.
	if s.OnSelectModel != nil && s.ModelStatus.Selectable {
		a.OnSelectModel = s.OnSelectModel
	}
	// The keys toggle is the shell's own state too, but it is wired only when the
	// caller supplied a session.keys snapshot. The sidebar no longer draws the
	// entry the reference lacks; the toggle stays for a caller that sets
	// State.ShowKeys or re-introduces an entry elsewhere.
	if s.Keys.Loaded {
		a.OnToggleKeys = func() {
			s.ShowKeys = !s.ShowKeys
			if s.ShowKeys {
				s.dropThreadSearch()
				s.setRoute(RouteChat)
			}
		}
	}
	// The rename dialog is the window's own state too: the menu item only opens
	// it, and the dialog's confirm button calls the caller's OnRename. It is
	// wired exactly while that callback exists, so a fixture without one keeps
	// the entry explained as WIP.
	if s.OnRename != nil {
		a.OnRequestRename = func() {
			if s.Selected == "" {
				return
			}
			s.renameFor = s.Selected
			s.RenameDraft = s.selectedTitle()
			if s.RenameDraft == tr(s.locale(), keyNoSession) {
				s.RenameDraft = ""
			}
			s.RenameConfirm = true
		}
		a.OnRename = s.OnRename
	}
	// The delete confirmation is the window's own state too: the menu item only
	// opens it, and the alert's confirm button calls the caller's
	// OnDeleteSession. It is wired exactly while that callback exists, so a
	// fixture without one keeps the entry explained as WIP.
	if s.OnDeleteSession != nil {
		a.OnRequestDelete = func() {
			if s.Selected == "" {
				return
			}
			s.deleteFor = s.Selected
			s.DeleteConfirm = true
		}
	}
	return a
}

// sidebarView copies the catalog and the window-owned entry state into the
// sidebar's view. Selected is a pointer: the sidebar writes the choice it made,
// and the shell reads it on the next frame. Branch metadata stays in State for
// the session header menu; this column no longer draws it.
func (s *State) sidebarView() SidebarView {
	return SidebarView{
		Sessions: s.Sessions,
		Selected: &s.Selected,
		Creating: s.CreatingSession,
		Pinned:   s.Pinned,
		Archived: s.Archived,
		Locale:   s.locale(),
		state:    &s.sidebar,
	}
}

// settingsView copies the desktop preferences, the catalog rows the
// archived-sessions page lists and the model catalog the models page lists. The
// rows are a display copy: ids the catalog no longer holds draw nothing, and the
// catalog's title and order stay the only session facts. The model fields are
// the same snapshot the composer's model list reads; carrying them here adds no
// read and no second source.
func (s *State) settingsView() SettingsView {
	v := s.Settings
	v.ArchivedSessions = s.archivedSessionRows()
	v.ModelCatalog = s.ModelCatalog
	v.ModelStatus = s.ModelStatus
	return v
}

// archivedSessionRows lists the archived sessions the catalog still holds, in
// the catalog's own order. It is display state for the settings page: the id
// list hides rows, the catalog supplies their titles.
func (s *State) archivedSessionRows() []ArchivedSession {
	if len(s.Archived) == 0 {
		return nil
	}
	rows := make([]ArchivedSession, 0, len(s.Archived))
	for _, session := range s.Sessions {
		if containsID(s.Archived, session.ID) {
			rows = append(rows, ArchivedSession{SessionID: session.ID, Title: session.Title})
		}
	}
	return rows
}

// pendingApprovals collects the approval requests the current rows still show,
// keyed by request id so a duplicated row cannot offer the same decision twice.
// It copies from the rows the projection produced; a click never adds or removes
// an entry here.
func (s *State) pendingApprovals() []Approval {
	var pending []Approval
	seen := map[string]bool{}
	for _, m := range s.Messages {
		if m.Author != AuthorApproval || m.Approval == nil {
			continue
		}
		approval := *m.Approval
		if approval.RequestID == "" || seen[approval.RequestID] {
			continue
		}
		seen[approval.RequestID] = true
		pending = append(pending, approval)
	}
	return pending
}

// headerView copies the supervisor's report, the terminal close and the
// window-owned status into the toolbar's view. Project is the selected catalog
// row's Group, which the catalog already carries as the cwd's base name; the
// shell adds no project fact of its own.
func (s *State) headerView() HeaderView {
	v := HeaderView{
		Project:          s.selectedProject(),
		Title:            s.sessionTitleLabel(),
		SessionID:        s.Selected,
		HasSession:       s.Selected != "",
		SelectedBranch:   s.SelectedBranch,
		Pinned:           containsID(s.Pinned, s.Selected),
		Archived:         containsID(s.Archived, s.Selected),
		Connection:       s.Connection,
		ConnectionError:  s.ConnectionError,
		Run:              s.Run,
		Closed:           s.SessionClosed,
		ClosedReason:     s.SessionClosedReason,
		ModelProgress:    s.ModelProgress,
		Notices:          s.Notices,
		ActionNotices:    s.ActionNotices,
		ActionError:      s.ActionError,
		ActionStatus:     s.ActionStatus,
		Loading:          s.ConversationLoading,
		SettingsOpen:     s.currentRoute() == RouteSettings,
		KeysOpen:         s.ShowKeys,
		ContextPanelOpen: s.contextPanelShown(),
		ThreadSearch:     s.threadSearchView(),
		DebugInfo:        s.debugInfoText(),
		Locale:           s.locale(),
	}
	// The confirmation belongs to the selection it was opened for. A selection
	// change is dropped by syncDeleteConfirm before the frame builds; the guard
	// keeps a stale alert out of a frame that has not run it.
	if s.DeleteConfirm && s.deleteFor == s.Selected {
		v.DeleteConfirm = &s.DeleteConfirm
	}
	// The rename dialog belongs to its selection the same way. The region writes
	// only the text field and the open flag, and the confirm button calls
	// OnRename with the text it holds.
	if s.RenameConfirm && s.renameFor == s.Selected {
		v.RenameConfirm = &s.RenameConfirm
		v.RenameDraft = &s.RenameDraft
	}
	return v
}

// The session menu's "Copy debug info" text. Its title and row labels follow
// the reference's ChatHeader.buildDebugInfoText, one labeled value per line
// with the labels padded to one width. The text carries only the public model
// facts the window already displays: it is built from ModelCatalog's current
// pair and catalog row here, never from a private snapshot or a new App Server
// read. ModelStatus explains a catalog the window could not read, which leaves
// the row-specific facts n/a rather than turning them into guesses.
const (
	debugInfoTitle      = "Model runtime config"
	debugInfoLabelWidth = 9
	debugInfoMissing    = "n/a"
)

// debugInfoText builds the menu's copy payload from the current model's public
// facts. A field the window does not hold is written n/a rather than guessed;
// a model the catalog carries no row for still names the pair the session
// reported, and its limits, capabilities and source stay n/a. With no current
// pair there is no runtime config to report, so the text is empty and the menu
// keeps the entry as a WIP statement.
func (s *State) debugInfoText() string {
	providerID, modelID := s.ModelCatalog.CurrentProviderID, s.ModelCatalog.CurrentModelID
	choice, hasChoice := debugInfoChoice(s.ModelCatalog)
	if hasChoice {
		providerID, modelID = choice.ProviderID, choice.ModelID
	}
	if providerID == "" && modelID == "" {
		return ""
	}
	providerName, modelName := "", ""
	contextLimits := debugInfoMissing + " / output " + debugInfoMissing
	caps := "tools " + debugInfoMissing + ", vision " + debugInfoMissing + ", reasoning " + debugInfoMissing
	source := debugInfoMissing
	if hasChoice {
		providerName, modelName = choice.ProviderName, choice.DisplayName
		contextLimits = debugInfoNumber(choice.ContextWindow) + " / output " + debugInfoNumber(choice.MaxOutputTokens)
		caps = "tools " + debugInfoYesNo(choice.ToolCalling) +
			", vision " + debugInfoYesNo(choice.Vision) +
			", reasoning " + debugInfoYesNo(choice.Reasoning)
		if choice.Source != "" {
			source = choice.Source
		}
	}
	rows := []string{
		debugInfoRow("Provider", debugInfoPair(providerName, providerID)),
		debugInfoRow("Model", debugInfoPair(modelName, modelID)),
		debugInfoRow("Context", contextLimits),
		debugInfoRow("Caps", caps),
		debugInfoRow("Source", source),
	}
	return strings.Join(append([]string{debugInfoTitle}, rows...), "\n")
}

// debugInfoChoice finds the catalog row for the current pair: the row the
// caller flagged current, or the row matching the reported pair when a caller
// set the ids without the flag. It returns false when the catalog carries no
// row, which leaves the row's own facts n/a.
func debugInfoChoice(catalog ModelCatalog) (ModelChoice, bool) {
	for _, choice := range catalog.Choices {
		if choice.Current {
			return choice, true
		}
	}
	if catalog.CurrentProviderID == "" || catalog.CurrentModelID == "" {
		return ModelChoice{}, false
	}
	for _, choice := range catalog.Choices {
		if choice.ProviderID == catalog.CurrentProviderID && choice.ModelID == catalog.CurrentModelID {
			return choice, true
		}
	}
	return ModelChoice{}, false
}

// debugInfoPair names one provider or model by its display name and its id, as
// the reference does. A missing half is left out instead of drawn empty, and
// two missing halves are n/a.
func debugInfoPair(name, id string) string {
	if name == "" {
		if id == "" {
			return debugInfoMissing
		}
		return id
	}
	if id == "" {
		return name
	}
	return name + " (" + id + ")"
}

// debugInfoNumber prints a declared limit, or n/a when the wire declared none.
// A non-positive value is no limit the shell can report.
func debugInfoNumber(value *int64) string {
	if value == nil || *value <= 0 {
		return debugInfoMissing
	}
	return strconv.FormatInt(*value, 10)
}

// debugInfoYesNo prints one declared capability. A model whose row is missing
// never reaches it; the caller writes n/a instead of reading the false zero.
func debugInfoYesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// debugInfoRow puts one label and its value on a line, padded to the label
// column the reference uses so the values line up in a plain-text clipboard.
func debugInfoRow(label, value string) string {
	if pad := debugInfoLabelWidth - len(label); pad > 0 {
		return label + strings.Repeat(" ", pad) + " " + value
	}
	return label + " " + value
}

// timelineView copies the rows and the window's loading and close facts into
// the timeline's view, with pointers to the list, fold and Markdown state the
// region keeps across frames. Pending approvals are not rows: they take the
// composer's slot, so they are left out here and read by pendingApprovals. The
// rows are the shell's own copies, so the find bar's mark on the current match is
// written here without reaching the display copies the window holds.
func (s *State) timelineView() TimelineView {
	rows := s.timelineRows()
	if s.ShowThreadSearch && s.ThreadSearchIndex >= 0 && s.ThreadSearchIndex < len(s.threadSearchFound) {
		highlightThreadMatch(rows, s.threadSearchFound[s.ThreadSearchIndex])
	}
	return TimelineView{
		Messages: rows,
		Loading:  s.ConversationLoading,
		Closed:   s.SessionClosed,
		list:     &s.conversation,
		folds:    &s.folds,
		Locale:   s.locale(),
		// The branch-from dialog belongs to the row it was opened from: the view
		// carries its open flag, prompt and resolved cut point only while the
		// selection and displayed branch are still the ones it was opened on.
		BranchFromConfirm: s.branchFromConfirmView(),
		BranchFromDraft:   s.branchFromDraftView(),
		BranchFromAt:      s.branchFromAt,
	}
}

// branchFromConfirmView returns the open flag the timeline draws the
// "Branch from here" dialog from, or nil while the dialog is closed or belongs
// to another session or branch.
func (s *State) branchFromConfirmView() *bool {
	if !s.BranchFromConfirm || s.branchFromFor != s.Selected || s.branchFromBranch != s.SelectedBranch {
		return nil
	}
	return &s.BranchFromConfirm
}

// branchFromDraftView returns the prompt field the dialog edits, or nil while
// the dialog is not drawn.
func (s *State) branchFromDraftView() *string {
	if s.branchFromConfirmView() == nil {
		return nil
	}
	return &s.BranchFromDraft
}

// timelineRows is the rows the timeline draws, in the projection's order: every
// conversation row but the pending approvals, which take the composer's slot
// instead of a row. The find bar numbers its matches in this same list, so a
// match's row is the index the timeline builds.
func (s *State) timelineRows() []Message {
	rows := make([]Message, 0, len(s.Messages))
	for _, m := range s.Messages {
		if m.Author == AuthorApproval {
			continue
		}
		rows = append(rows, m)
	}
	return rows
}

// highlightThreadMatch marks the row the find bar's current match is in, and the
// block within it when the match is in one. The row is marked washed either way,
// because a Markdown turn and a tool card draw text the shell does not lay out
// itself; Match carries the run where the shell does draw the text. The block
// slice is cloned before it is marked: the row copies share it with the display
// copies the window holds.
func highlightThreadMatch(rows []Message, match threadMatch) {
	if match.Row < 0 || match.Row >= len(rows) {
		return
	}
	row := &rows[match.Row]
	highlight := &TextHighlight{Start: match.Start, End: match.End}
	row.Search = &MessageSearch{}
	if match.Block < 0 {
		row.Search.Match = highlight
		return
	}
	if match.Block >= len(row.Blocks) {
		return
	}
	blocks := make([]Block, len(row.Blocks))
	copy(blocks, row.Blocks)
	blocks[match.Block].Match = highlight
	row.Blocks = blocks
}

// threadSearchView copies the bar's own state and the count the last frame
// found. Open and Query are pointers: the bar's field writes the text in place
// and the close entry asks the window through Actions, so the shell reads them
// on the next frame without a second copy.
func (s *State) threadSearchView() ThreadSearchView {
	v := ThreadSearchView{
		Query:  &s.ThreadSearchQuery,
		Focus:  &s.threadSearchFocus,
		Locale: s.locale(),
	}
	if !s.ShowThreadSearch {
		return v
	}
	v.Open = &s.ShowThreadSearch
	if total := len(s.threadSearchFound); total > 0 && s.ThreadSearchIndex >= 0 {
		v.Current = s.ThreadSearchIndex + 1
		v.Total = total
	}
	return v
}

// composerView copies the draft and the window's submission gate into the
// composer's view. CanSubmit leaves the draft's text to the region, which reads
// it live when the shortcut or Send fires.
func (s *State) composerView() ComposerView {
	return ComposerView{
		Draft:       &s.Draft,
		CanSubmit:   s.canSubmit(),
		Placeholder: tr(s.locale(), keyComposerPlaceholder),
		// The card widens once the conversation has rows: the reference's empty
		// state is 461 DIPs and its conversation 520.
		Wide:       len(s.Messages) > 0,
		Workspace:  s.Workspace,
		Permission: s.Permission,
		// The context row's branch is the selected session's, not the
		// workspace's git branch: the App Server owns session branches and this
		// shell reads no git fact. Context usage has no fact yet, so the meter
		// stays unknown and draws its dash rather than a number.
		Branch:       s.SelectedBranch,
		ContextKnown: false,
		Shortcut:     KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
		Focus:        &s.composerFocus,
		Locale:       s.locale(),
		// The model chip displays the App Server facts the caller read. With no
		// current pair it keeps the unconfigured label and carries the status
		// reason; a row calls the caller's OnSelectModel when it is wired.
		ModelLabel:   s.ModelCatalog.Current,
		ModelReason:  s.modelReason(),
		ModelOpen:    &s.ModelListOpen,
		ModelChoices: s.ModelCatalog.Choices,
	}
}

// modelReason explains a model the composer cannot name or change: the catalog
// read's capability/failure reason, or the missing session.model.update
// capability. A current pair with a selectable catalog shows no reason; the
// chip's description carries the WIP reason when selection is unavailable.
func (s *State) modelReason() string {
	if s.ModelStatus.Reason == "" {
		return ""
	}
	if s.ModelCatalog.HasCurrent && s.ModelCatalog.Current != "" && s.ModelStatus.Selectable {
		return ""
	}
	return s.ModelStatus.Reason
}

// canSubmit reports whether the window would accept a submission from the
// selected session right now: a wired submit callback, a ready connection,
// loaded history, an open session and a selection. It says nothing about the
// draft: the composer reads that live. A fixture without those cannot pretend a
// submission the App Server never received.
func (s *State) canSubmit() bool {
	return s.OnSubmit != nil && s.Connection == ConnReady && !s.ConversationLoading &&
		!s.SessionClosed && s.Selected != ""
}

// selectedTitle returns the title of the selected session, or a placeholder.
func (s *State) selectedTitle() string {
	for _, session := range s.Sessions {
		if session.ID == s.Selected {
			return session.Title
		}
	}
	return tr(s.locale(), keyNoSession)
}

// sessionTitleLabel is the title the toolbar's breadcrumb draws. Unlike
// selectedTitle, which also seeds the rename dialog and must stay empty for a
// session without a title, this one names an untitled session with the locale's
// placeholder: the breadcrumb never shows an opaque id in place of a name.
func (s *State) sessionTitleLabel() string {
	for _, session := range s.Sessions {
		if session.ID == s.Selected {
			return sessionLabel(session.Title, s.locale())
		}
	}
	if s.SessionClosed && s.Selected != "" {
		// The row left the catalog with the confirmed delete, so the toolbar
		// keeps naming it from the labels it last read instead of claiming no
		// session is selected.
		return sessionLabel(s.ClosedSessionTitle, s.locale())
	}
	return tr(s.locale(), keyNoSession)
}

// selectedProject returns the selected session's project label, as the catalog
// carries it in Group (a cwd base name), or "" when the catalog has none. The
// toolbar draws no project name or separator for an empty value; it never
// invents a project.
func (s *State) selectedProject() string {
	for _, session := range s.Sessions {
		if session.ID == s.Selected {
			return session.Group
		}
	}
	if s.SessionClosed && s.Selected != "" {
		return s.ClosedSessionProject
	}
	return ""
}
