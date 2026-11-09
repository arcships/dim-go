// This file freezes the contract between the shell and its regions.
//
// A region is one area of the window: the session sidebar, the toolbar with its
// banner, the timeline, one pending approval row, and the composer. Each region
// is drawn by one file and reads only three things: the shared design tokens,
// its own view struct from this file, and the Actions the window wired. It never
// reaches back into State, calls another region, or reads the App Server. The
// shell (shell.go) is the only assembler: it copies State into the view structs,
// builds the Actions from State's callbacks, and calls each render entry below
// in draw order. The approval row is the one nested region: the timeline owns
// its place in the row list and calls renderApproval for the row's body.
//
// This is what lets regions be worked on in parallel worktrees: a region may
// change its own internals and helpers freely, but the fields it reads, its
// entry signature and what it does on a click are fixed here and in
// docs/ui-component-protocols.md.
//
// Every region keeps these invariants:
//
//   - The App Server owns session, run, timeline, branch and approval facts. A
//     view carries display copies and the window's own view state; a region
//     renders them and derives no fact from them.
//   - A click is a request, never a result. A region calls the matching Actions
//     field at most once and changes no session fact itself; the window sends
//     the App Server method and reports the outcome through HeaderView's
//     ActionError or ActionStatus, or through a later projection frame.
//   - A nil Actions field means the window has not wired that action. The
//     calling region keeps its entry visible, disabled and explained with
//     wipAction and its reason, and never simulates a result.
package ui

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// Actions is everything a region may ask the window to do. Each field that is
// set carries one request from one user action; each that is nil is a WIP gap
// the region draws rather than hides. The window owns the App Server call: no
// region reports an action it did not send, and a callback that fails must
// surface the failure through HeaderView.ActionError or ActionStatus instead of
// letting the view look successful.
type Actions struct {
	// OnCreateSession is called by the sidebar's "New session" entry when the
	// window has wired it. The caller runs session.create and publishes the new
	// session in the catalog and the selection only once the App Server
	// returned it; a rejected or failed create leaves the catalog as it was and
	// reports through HeaderView.ActionError. While SidebarView.Creating is
	// true the entry is disabled and the region does not call this again.
	OnCreateSession func()
	// OnSelect is called by the sidebar after a click or an arrow key moved the
	// choice, with the id it just wrote to SidebarView.Selected. The caller
	// opens that session and reports what it finds; the sidebar adds no row and
	// invents no history. A session that fails to open is shown through the
	// connection and banner fields, never as a local placeholder session.
	OnSelect func(sessionID string)
	// OnSelectBranch is called by the sidebar's branch list with the branch the
	// user chose. Branch switching is not its own RPC: the caller reads that
	// branch's history through the session controller's OpenBranch and moves
	// SidebarView.SelectedBranch only after the App Server answered. A failed
	// read leaves the current branch shown and reports through
	// HeaderView.ActionError; the region writes no branch fact and invents no
	// branch. While this field is nil the list keeps its read-only WIP form.
	OnSelectBranch func(branchID string)
	// OnSubmit is called by the composer's Send entry and submit shortcut with
	// the trimmed draft. The caller sends the turn and clears the draft only
	// after the App Server accepted it; on failure the draft stays and the
	// failure is reported through HeaderView.ActionError, so a send the App
	// Server never received never looks delivered.
	OnSubmit func(text string)
	// OnStop is called by the toolbar's Stop entry while a turn runs. The caller
	// sends session.stop and leaves the run state to the App Server's next
	// report; the view does not claim the turn stopped.
	OnStop func()
	// OnReconnect is called by the toolbar's Reconnect entry. The caller asks
	// the supervisor to reopen the attachment and reports its outcome; the view
	// does not claim the connection recovered.
	OnReconnect func()
	// OnApproval is called by an approval row with one decision for one
	// request, addressed by the App Server's branch and request identity. The
	// caller sends approval.respond and lets the session projection settle the
	// row; the region never removes the row or claims an outcome on its own,
	// and a row without a request identity never reaches this field.
	OnApproval func(branchID, requestID string, decision ApprovalDecision)
	// OnOpenSettings is called by the toolbar's Settings entry and toggles the
	// appearance surface. The preferences it edits are desktop-local: the caller
	// applies the theme and the sidebar width and persists what it can; nothing
	// here reaches the App Server. While this field is nil the entry is absent.
	OnOpenSettings func()
	// OnToggleKeys is called by the sidebar's "Session keys" entry and toggles
	// the session.keys browser in the main column. The shell owns both the entry
	// and the panel's visibility, so this field carries no caller callback: it is
	// wired exactly while a session.keys snapshot is present, and the browser
	// only displays what the caller already read. While this field is nil the
	// entry stays a WIP statement and opening the browser sends no request.
	OnToggleKeys func()
}

// SidebarView is what the sidebar region draws: the session catalog, the
// read-only branch metadata, the entry state the window owns and the list state
// the region itself keeps across frames.
type SidebarView struct {
	// Sessions is the session catalog in the App Server's order, with the
	// display group each row is drawn under. The region invents no row and no
	// group label: an empty Group is drawn under ungroupedLabel once another
	// row carries one, and a catalog with no group at all stays one flat list.
	Sessions []Session
	// Selected is the selected session's id. It is a pointer because the
	// sidebar owns the display choice: a click or an arrow key writes the
	// chosen row's id here and then calls Actions.OnSelect, so the next frame
	// shows the choice even when the window has no callback to open the
	// session. It is the only field the region writes.
	Selected *string
	// SelectedBranch is the observed current branch of the selected session.
	// It is display only: branch switching and cut point stay unavailable.
	SelectedBranch string
	// Branches is read-only branch metadata for the selected session. An empty
	// list draws no branch section at all.
	Branches []Branch
	// Creating is the window's record that session.create is in flight. It
	// holds no session fact; it only keeps the entry from being clicked twice
	// while one request is unanswered.
	Creating bool
	// Capabilities are the read-only cards for the actions this window has not
	// wired. The shell maps the wired Actions to them (see wipCapabilities), so
	// a card appears exactly while its action is a WIP gap; the region draws
	// them verbatim and offers no control on them.
	Capabilities []wipCapability
	// KeysOpen is the window's record that the keys browser is showing. The
	// "Session keys" entry reports itself expanded from it; it is display state,
	// not a key fact.
	KeysOpen bool

	// state is the region's own view state: the thread list's place among its
	// rows, its choice, the selection that choice was last mapped from and the
	// rows that choice indexes into. The shell owns it for the window's
	// lifetime so a frame does not lose the list's scroll position; it holds no
	// session fact and cannot outlive the catalog it was laid out from.
	state *sidebarState
}

// sidebarState is the sidebar's throwaway view state. Every row in it is
// derived from SidebarView.Sessions on the frame it is used; see sidebar.go.
type sidebarState struct {
	list   ui.ListState
	choice int
	synced string
	rows   []sidebarRow
}

// HeaderView is what the toolbar and its banner draw: the selected session's
// title, the connection and run the supervisor reported, the transient status
// line and the window's own action outcomes.
type HeaderView struct {
	// Title is the selected session's title, already resolved by the shell:
	// "No session" stands in when the catalog has none. The region does no
	// lookup of its own.
	Title string
	// HasSession is true while a session is selected. The stop and reconnect
	// entries are offered only with a selected session to attach to.
	HasSession bool
	// Connection and ConnectionError are the supervisor's last report and the
	// last error it raised. They are display state: the region reads no daemon
	// state, SQLite, JSONL or log.
	Connection      Connection
	ConnectionError string
	// Run is what the selected session is doing, as the supervisor reports it.
	// The region never derives a run from the rows it shows.
	Run RunState
	// Closed and ClosedReason are the terminal session.closed fact. A closed
	// session outranks the last run state, the connection error and any older
	// action error: it is shown, and no submit, stop or reconnect is offered.
	Closed       bool
	ClosedReason string
	// ModelProgress is the transient, non-durable model preview.
	ModelProgress *ModelProgress
	// Notices and ActionNotices are transient notifications: projection notices
	// and the window's own outcomes of actions it sent. The status row shows
	// their total only. They are not projection facts, so a live refresh must
	// leave the window-owned ones alone.
	Notices       []Notice
	ActionNotices []Notice
	// ActionError is the window-owned failure of the last action it sent. It
	// shows in the banner only when no terminal close and no connection error
	// outranks it. A failure is reported here; it is never shown as success.
	ActionError string
	// ActionStatus is the window-owned line for an outcome the App Server
	// confirmed without a session fact to show, such as a submission resolved by
	// reading it back. A later action that settles another outcome replaces it.
	ActionStatus string
	// Loading is the window's record that history or a reconnect is still
	// loading. It hides the reconnect entry so one attempt cannot be started
	// twice; it never stands in for history.
	Loading bool
	// SettingsOpen is the window's record that the appearance surface is
	// showing. The toolbar's entry reads "Settings" or "Conversation" from it.
	SettingsOpen bool
}

// TimelineView is what the timeline region draws: the conversation rows, the
// loading and empty states, and the view state the region keeps across frames.
type TimelineView struct {
	// Messages are the rows in the projection's order. The projection owns
	// message identity, streaming and tool facts; a row is a display copy.
	Messages []Message
	// Loading is true while history is still coming. The region shows the
	// loading state instead of the rows, and reads it as the end of loading
	// only when the next frame clears it.
	Loading bool
	// Closed is the terminal session.closed fact. The region does not drop or
	// add a row on it: rows are projection facts, and a closed session's history
	// stays readable. It is carried so a region that grows a live tail can stop
	// claiming one; nothing branches on it today.
	Closed bool

	// list is the conversation list's place among its rows and the identity of
	// each row, kept across frames so appended rows follow the end and rows keep
	// their state as the conversation changes. It holds no timeline fact.
	list *ui.ListState
	// folds is the shell's own fold state of thinking and tool blocks, keyed by
	// row and block. The projection does not own whether a user opened a fold,
	// so it lives here for the window's lifetime rather than in the projection.
	folds *blockFolds
	// markdown is the shared Markdown options of every assistant row, whose
	// theme follows the window's.
	markdown *markdown.Options
}

// blockFolds is the shell's own fold state of thinking and tool blocks, keyed
// by row and block (see blockKey). A fold left behind by a row that left the
// timeline stays as one unused entry and is never re-keyed.
type blockFolds struct {
	open map[string]*bool
}

// state returns the fold state of one block, creating it closed. The pointer
// stays valid across frames, which is what ui.CollapsibleBase binds: whether a
// user opened a fold is the shell's own view state, not a timeline fact.
func (f *blockFolds) state(key string) *bool {
	if f.open == nil {
		f.open = map[string]*bool{}
	}
	if open, ok := f.open[key]; ok {
		return open
	}
	open := new(bool)
	f.open[key] = open
	return open
}

// ApprovalVariant is the UI option a pending request is drawn with. It names
// the option, not a wire decision: docs/protocol-contract.md §12 collapses
// allow_once and allow_all onto approval.respond's "allow" and everything else
// onto "deny", so the variant changes the region's wording and nothing it
// sends.
//
// The names are the ones the contract and docs/ui-layout-spec.md §5.2 use.
// approval.request carries no option list (appserver's ApprovalRequest has
// source, call, reason and no options), so a view may only carry a variant the
// shell was told about; it derives none from the call name or the source kind.
// An empty or unknown variant means the options were not named, and the region
// falls back to plain Approve/Deny instead of inventing allow_once/allow_all.
type ApprovalVariant string

const (
	// ApprovalVariantUnspecified is a request whose options the window was not
	// told: the region draws plain Approve/Deny.
	ApprovalVariantUnspecified ApprovalVariant = ""
	// ApprovalVariantAllowOnce is the allow-once option: one approval, no
	// persisted grant. It collapses to approval.respond's "allow".
	ApprovalVariantAllowOnce ApprovalVariant = "allow_once"
	// ApprovalVariantAllowAll is the always-allow option. It collapses to the
	// same "allow" on the wire, but the persistent grant it promises goes
	// through session.grants.add, which the shell does not call yet. The
	// region draws it as a WIP entry rather than answering once in its place.
	ApprovalVariantAllowAll ApprovalVariant = "allow_all"
	// ApprovalVariantRejectOnce is the reject-once option. It collapses to
	// approval.respond's "deny".
	ApprovalVariantRejectOnce ApprovalVariant = "reject_once"
)

// ApprovalView is what one approval row draws: the projection's text, the
// request identities the row still asks about, the primary request's input as
// a read-only block, and the UI option the requests are drawn with. The
// decisions go through Actions.OnApproval; the row itself stays until the
// session projection reports its resolution.
type ApprovalView struct {
	// Text is the row's body, as the transient projection gives it. The region
	// adds no reason of its own.
	Text string
	// Pending is the list of requests this row offers decisions for. The
	// transient projection gives one request per row today. A row with no
	// request, or a request with no App Server identity, keeps both decisions
	// visible, disabled and explained instead of sending a decision nobody can
	// address.
	Pending []Approval
	// InputDisplay is the primary request's tool input already rendered for
	// display: the command line or arguments of ApprovalRequest.Call.Input,
	// copied by the shell. The region draws it read-only, monospaced and
	// bounded; it parses no JSON and adds no field of its own. Empty means the
	// request had no input payload, and the region draws no block rather than an
	// empty one.
	InputDisplay string
	// Variant is the UI option the pending requests are drawn with, as named by
	// docs/protocol-contract.md §12 and docs/ui-layout-spec.md §5.2. It changes
	// the decisions' labels only; a click always sends the same allow/deny
	// decision through Actions.OnApproval, and allow_all stays a WIP entry since
	// its grant needs session.grants.add (not implemented).
	Variant ApprovalVariant
}

// ComposerView is what the composer region draws: the draft it edits and the
// state and shortcut the window gives it.
type ComposerView struct {
	// Draft is the draft text the composer edits in place. A submitted draft
	// stays here: the window clears it once the App Server accepts the
	// submission, so a failed send keeps the text.
	Draft *string
	// CanSubmit is the window's gate for a submission from the selected
	// session: a wired submit callback, a ready connection, loaded history, an
	// open session and a selection. It does not include the draft, because a
	// keystroke and the shortcut can land between two frames: the region reads
	// the draft live and sends nothing while its trimmed text is empty.
	CanSubmit bool
	// Sending is the window's record that a submission is in flight. No
	// supervisor or App Server fact reports one today, so the shell always
	// passes false; the region disables Send while it is true and changes
	// nothing else, so a later wiring cannot make the composer claim a
	// submission the App Server has not seen.
	Sending bool
	// Placeholder is the composer's empty-state text.
	Placeholder string
	// Shortcut is the modifier and key that submit the draft. The shell's own
	// focus shortcut (ComposerShortcutMods with ComposerShortcutKey) is not part
	// of this view: it focuses Focus and is handled by the shell.
	Shortcut KeyBinding
	// Focus is the composer's identity, for the shell's focus shortcut.
	Focus *ui.Handle
}

// KeyBinding is one keyboard shortcut as a region listens for it.
type KeyBinding struct {
	// Mods are the modifier keys held with the key; zero means none.
	Mods ui.Modifiers
	// Key is the key pressed with Mods.
	Key ui.Key
}

// renderSidebar draws the session sidebar: the brand row, the global entries,
// the grouped thread list, the read-only branch list, the thread actions and the
// WIP capability cards. It writes SidebarView.Selected on a choice, then calls
// Actions.OnSelect; it calls Actions.OnCreateSession from "New session".
func renderSidebar(c *ui.Context, k tokens, v SidebarView, a Actions) {
	sidebar(c, k, v, a)
}

// renderHeader draws the toolbar (title, run, connection, Stop, Reconnect) and
// the banner below it (a terminal close, the connection error, or the window's
// action error). It calls Actions.OnStop and Actions.OnReconnect from the
// entries, and only when HeaderView says the entry applies.
func renderHeader(c *ui.Context, k tokens, v HeaderView, a Actions) {
	header(c, k, v, a)
}

// renderTimeline draws the conversation list, or the loading or empty state. It
// calls Actions.OnApproval for an approval row's decisions; no other action
// starts from the timeline.
func renderTimeline(c *ui.Context, k tokens, v TimelineView, a Actions) {
	timeline(c, k, v, a)
}

// renderApproval draws the pending-approval panel that takes the composer's
// slot. It calls Actions.OnApproval, and only for a request that carries the
// App Server's identity.
func renderApproval(c *ui.Context, k tokens, v ApprovalView, a Actions) {
	renderApprovalPanel(c, k, v, a)
}

// renderComposer draws the attachment entry, the Send entry and the draft
// field. It calls Actions.OnSubmit with the trimmed draft when the window's gate
// and the live draft both allow a send.
func renderComposer(c *ui.Context, k tokens, v ComposerView, a Actions) {
	composer(c, k, v, a)
}

// renderKeysPanel draws the session.keys browser in the main column: the
// snapshot the caller read, its through, the local filter and one row per key.
// It writes KeysPanelView.Query while the filter is edited and
// KeysPanelView.Selected on a row click or Enter; it calls no Actions field,
// because the snapshot is a read and the panel starts no request.
func renderKeysPanel(c *ui.Context, k tokens, v *KeysPanelView, a Actions) {
	keysPanel(c, k, v, a)
}
