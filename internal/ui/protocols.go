// This file freezes the contract between the shell and its regions.
//
// A region is one area of the window: the session sidebar, the toolbar with its
// banner, the timeline, one pending approval row, and the composer. Each region
// is drawn by one file and reads only three things: the shared design tokens,
// its own view struct from this file, and the Actions the window wired. It never
// reaches back into State, calls another region, or reads the App Server. The
// shell (shell.go) is the only assembler: it copies State into the view structs,
// builds the Actions from State's callbacks, and calls each render entry below
// in draw order. Two regions are nested: the approval row belongs to the
// timeline, which calls renderApproval for the row's body, and the find bar
// belongs to the toolbar, which calls renderThreadSearch at its trailing end
// while the window says it is open.
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
	// OnSelectBranch is the branch-switch request a branch list calls with the
	// branch the user chose. No region draws a branch list today: the session
	// menu offers End branch and Refresh branches, which neither switch the
	// displayed branch. When a branch list returns, switching is not its own
	// RPC: the caller reads that branch's history through the session
	// controller's OpenBranch and moves the displayed branch only after the App
	// Server answered. A failed read leaves the current branch shown and reports
	// through HeaderView.ActionError; the region writes no branch fact and
	// invents no branch.
	OnSelectBranch func(branchID string)
	// OnEndBranch is called by the toolbar's session menu with the displayed
	// branch's id. The caller sends session.branch_end, then re-reads
	// session.branches and the branch view. No row is removed here or by the
	// caller before the App Server confirms the end: the directory entry
	// disappears only after session.branches reports it gone, and an
	// already-ended reply is not turned into a local delete. A failed end keeps
	// the row and reports through HeaderView.ActionError. While this field is
	// nil the menu entry stays visible, disabled and explained as WIP.
	OnEndBranch func(branchID string)
	// OnRefreshBranches is called by the toolbar's session menu "Refresh
	// branches" entry. The caller re-reads session.branches into the directory
	// and publishes the new view; the region neither caches a directory nor
	// invents a row. A failed read leaves the previous directory and reports
	// through HeaderView.ActionError. While this field is nil the entry stays
	// visible, disabled and explained as WIP.
	OnRefreshBranches func()
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
	// OnDeleteSession is called by the toolbar's delete confirmation with the
	// selected session id. The caller sends session.delete and removes the row
	// only after the App Server confirmed; a rejected delete leaves the row and
	// reports through HeaderView.ActionError. The region calls it only after the
	// user confirmed the alert, never from the menu item itself.
	OnDeleteSession func(sessionID string)
	// OnRequestDelete is called by the toolbar's "Delete session" menu item. It
	// opens the window's delete confirmation; the session is deleted only when
	// the user confirms, through OnDeleteSession. While this field is nil the
	// menu entry stays visible, disabled and explained as WIP.
	OnRequestDelete func()
	// OnRequestRename is called by the toolbar's "Rename" menu item. It opens
	// the window's title dialog, prefilled from HeaderView.RenameDraft; the
	// rename is sent only when the user confirms, through OnRename. While this
	// field is nil the menu entry stays visible, disabled and explained as WIP.
	OnRequestRename func()
	// OnRename is called by the rename dialog's confirm button with the field's
	// text. An empty (or whitespace-only) title asks the caller to clear the
	// manual title; the caller sends session.rename and reports the outcome
	// through HeaderView.ActionError/ActionNotices. The region sends nothing on
	// Escape or Cancel and calls this at most once per confirm.
	OnRename func(title string)
	// OnPinSession is called by the toolbar's session menu with the selected
	// session id when the window keeps a desktop-local pinned list. The caller
	// toggles that id and republishes the lists; nothing here reaches the App
	// Server, and a pinned row changes only the sidebar's display order. While
	// this field is nil the menu entry stays visible, disabled and explained as
	// WIP.
	OnPinSession func(sessionID string)
	// OnArchiveSession is called by the toolbar's session menu with the selected
	// session id. The caller toggles the desktop-local archived list, which hides
	// the session from the sidebar without changing any App Server fact. While
	// this field is nil the menu entry stays visible, disabled and explained as
	// WIP.
	OnArchiveSession func(sessionID string)
	// OnOpenSettings is called by the toolbar's session menu settings entry and
	// toggles the appearance surface. The preferences it edits are desktop-local:
	// the caller applies the theme and the sidebar width and persists what it
	// can; nothing here reaches the App Server. While this field is nil the
	// entry is absent.
	OnOpenSettings func()
	// OnOpenScheduled is called by the sidebar's scheduled-tasks entry and opens
	// the whole-window WIP page. The page sends no App Server request: no public
	// method lists scheduled tasks. While this field is nil the entry stays
	// visible, disabled and explained as WIP.
	OnOpenScheduled func()
	// OnOpenSkills is called by the sidebar's skills entry and opens the
	// whole-window WIP page. The page sends no App Server request: no public
	// method lists skills or plugins. While this field is nil the entry stays
	// visible, disabled and explained as WIP.
	OnOpenSkills func()
	// OnBackToChat is called by a WIP route's Back entry and returns the window
	// to the conversation. It changes display state only: the selection, draft
	// and conversation list state are untouched.
	OnBackToChat func()
	// OnToggleModelList is called by the composer's model chip and toggles the
	// model list. The list displays State's ModelCatalog and ModelStatus copies;
	// a row is selectable only while OnSelectModel is wired.
	OnToggleModelList func()
	// OnSelectModel is called once when the model list's row is chosen: the
	// caller writes session.model.update and reports the outcome. The list
	// closes before this callback runs, so one click sends at most one request;
	// a failure reaches the window through ActionError and the chip keeps the
	// App Server's previous pair. While this field is nil every row stays
	// visible, disabled and explained as WIP.
	OnSelectModel func(providerID, modelID string)
	// OnDismissNotices is called by the notice strip's dismiss entry. It clears
	// the window-owned ActionError/ActionNotices and the transient Notices; those
	// are display state, so dismissing them changes no App Server fact.
	OnDismissNotices func()
	// OnToggleKeys is called by the toolbar's session menu "Session keys" entry
	// and toggles the session.keys browser in the main column. The shell owns
	// both the entry and the panel's visibility, so this field carries no caller
	// callback: it is wired exactly while a session.keys snapshot is present, and
	// the browser only displays what the caller already read. While this field is
	// nil the menu entry stays a WIP statement and opening the browser sends no
	// request.
	OnToggleKeys func()
	// OnToggleContextPanel is called by the toolbar's context panel entry and
	// toggles the changed-file panel beside the conversation. The shell owns the
	// entry and the panel's visibility, so this field carries no caller callback:
	// the panel displays the caller's ContextPanelView copy and starts no App
	// Server read. While this field is nil the entry is absent.
	OnToggleContextPanel func()
	// OnOpenSearch is called by the sidebar's Search entry and opens the
	// window's search palette, the same palette Cmd+G opens. The palette filters
	// the catalog copy the window already holds and sends no App Server request,
	// so the shell always wires this field; a nil field leaves the sidebar entry
	// a WIP statement for a caller that draws the column alone.
	OnOpenSearch func()
	// OnThreadSearchNext is called by the find bar's next entry and its Enter
	// key. The shell moves the highlight to the next match, wrapping to the first
	// one after the last, and the timeline scrolls to the row it lands on. The bar
	// counts the matches itself, so this field sends no request and rewrites no
	// row.
	OnThreadSearchNext func()
	// OnThreadSearchPrevious is called by the find bar's previous entry and its
	// Shift+Enter key. It moves the highlight back, wrapping to the last match
	// before the first, as OnThreadSearchNext moves it forward.
	OnThreadSearchPrevious func()
	// OnThreadSearchClose is called by the find bar's close entry, its Escape key
	// and the window's own Escape while the bar is open. The shell drops the
	// query and the highlight and gives the focus back to the control the bar took
	// it from. It sends nothing: the bar holds no fact the App Server owns.
	OnThreadSearchClose func()
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
	// Switching and ending a branch are wired through Actions.OnSelectBranch and
	// Actions.OnEndBranch; a cut point (the parent sequence a side line hangs
	// off) is still not offered here.
	SelectedBranch string
	// Branches is read-only branch metadata for the selected session. An empty
	// list draws no branch section at all. The list offers "Refresh" and, for
	// the displayed branch while it is neither main nor ended, "End branch";
	// both ask the window and neither writes a row, so an end removes an entry
	// only when a later refresh reports it gone.
	Branches []Branch
	// Creating is the window's record that session.create is in flight. It
	// holds no session fact; it only keeps the entry from being clicked twice
	// while one request is unanswered.
	Creating bool
	// Pinned and Archived are the desktop-local session id lists. They are
	// display state only: a pinned row is drawn under the pinned heading before
	// the catalog's project groups, and an archived row is not drawn at all,
	// while the catalog keeps its own order and no session fact changes. An id
	// the catalog does not hold draws nothing.
	Pinned   []string
	Archived []string
	// Capabilities are the read-only cards for the actions this window has not
	// wired. The shell maps the wired Actions to them (see wipCapabilities), so
	// a card appears exactly while its action is a WIP gap; the region draws
	// them verbatim and offers no control on them.
	Capabilities []wipCapability
	// KeysOpen is the window's record that the keys browser is showing. The
	// "Session keys" entry reports itself expanded from it; it is display state,
	// not a key fact.
	KeysOpen bool
	// Locale is the language of the region's own text: the entries, the branch
	// labels, the footer and the capability cards. Group labels and session
	// titles are catalog facts and stay as the caller sent them. An empty or
	// unknown value draws English.
	Locale Locale

	// state is the region's own view state: the thread list's place among its
	// rows, its choice, the selection that choice was last mapped from and the
	// rows that choice indexes into. The shell owns it for the window's lifetime
	// so a frame does not lose the list's scroll position; it holds no session
	// fact and cannot outlive the catalog it was laid out from.
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

// HeaderView is what the conversation bar and its banner draw: the selected
// session's project and title, the connection and run the supervisor reported,
// the window's own action outcomes, and the menu entries for the actions the
// reference keeps behind the bar's "…" button.
type HeaderView struct {
	// Project is the selected session's project label, as the catalog's Group
	// carries it (a cwd base name). The toolbar draws it before the title with
	// a separator while it is set; an empty Project draws no project name and
	// no separator, rather than an invented one.
	Project string
	// Title is the selected session's title, already resolved by the shell:
	// "No session" stands in when the catalog has none. The region does no
	// lookup of its own.
	Title string
	// SessionID is the selected session's identity, which the delete
	// confirmation addresses. The region never invents one.
	SessionID string
	// HasSession is true while a session is selected. The toolbar, its menu and
	// the banner belong to a selected session; the stop and reconnect entries
	// are offered only with one to attach to.
	HasSession bool
	// SelectedBranch is the branch the window displays. The menu's "End branch"
	// entry is offered only while it names a non-main branch; the App Server
	// owns the branch and the outcome of ending it.
	SelectedBranch string
	// Pinned and Archived report the selected session's desktop-local display
	// state, so the menu's pin and archive entries read "Pin" or "Unpin" and
	// "Archive" or "Unarchive". They are never session facts.
	Pinned   bool
	Archived bool
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
	// ModelProgress is the transient, non-durable model preview. The bar draws
	// no status line today, so it carries the preview without presenting it; a
	// future status surface reads it rather than inventing progress.
	ModelProgress *ModelProgress
	// Notices and ActionNotices are transient notifications: projection notices
	// and the window's own outcomes of actions it sent. The bar draws no count
	// today; they stay in the contract for the status surface that reads them.
	// They are not projection facts, so a live refresh must leave the
	// window-owned ones alone.
	Notices       []Notice
	ActionNotices []Notice
	// ActionError is the window-owned failure of the last action it sent. It
	// shows in the banner only when no terminal close and no connection error
	// outranks it. A failure is reported here; it is never shown as success.
	ActionError string
	// ActionStatus is the window-owned line for an outcome the App Server
	// confirmed without a session fact to show, such as a submission resolved by
	// reading it back. The bar draws no status line today; it stays in the
	// contract for the surface that reads it. A later action that settles
	// another outcome replaces it.
	ActionStatus string
	// Loading is the window's record that history or a reconnect is still
	// loading. It hides the reconnect entry so one attempt cannot be started
	// twice; it never stands in for history.
	Loading bool
	// SettingsOpen is the window's record that the appearance surface is
	// showing. The menu's settings entry reads "Settings" or "Conversation"
	// from it, so the same entry always returns.
	SettingsOpen bool
	// KeysOpen is the window's record that the keys browser is showing in the
	// main column. The context panel entry is absent while it is set, since the
	// conversation the panel sits beside is not on screen.
	KeysOpen bool
	// ContextPanelOpen is the window's record that the context panel is showing
	// beside the conversation. The toolbar's entry reads its state from it and
	// toggles it through Actions.OnToggleContextPanel.
	ContextPanelOpen bool
	// DeleteConfirm is the window's own record that the delete confirmation is
	// showing. The toolbar draws the alert while *DeleteConfirm is true and
	// writes false as the alert closes; its confirm button calls
	// Actions.OnDeleteSession with SessionID. A nil pointer draws no alert.
	DeleteConfirm *bool
	// RenameConfirm and RenameDraft are the window's own record of the rename
	// dialog: while *RenameConfirm is true the toolbar draws it over the frame,
	// its text field edits *RenameDraft (prefilled with the current title), and
	// Escape or Cancel only writes false. Its confirm button calls
	// Actions.OnRename with the draft text. A nil pointer draws no dialog.
	RenameConfirm *bool
	RenameDraft   *string
	// ThreadSearch is the conversation's find bar, drawn at the end of the bar
	// while the window says it is open. The bar is a nested region of the
	// toolbar, as the approval row is one of the timeline: header.go draws it
	// through renderThreadSearch, and it reads the matches the shell counted from
	// the rows the window already displays. It is display state only and starts
	// no App Server read.
	ThreadSearch ThreadSearchView
	// Locale is the language of the toolbar's own text: the breadcrumb's
	// accessible name, the menu entries, the editor-open entry, the Stop and
	// Reconnect entries and the terminal-close banner. The run and connection
	// names, the session title and an error string are not the shell's words and
	// stay as they came. An empty or unknown value draws English.
	Locale Locale
}

// ThreadSearchView is what the conversation bar's find bar draws: the text it
// edits, the match the timeline highlights and the two entries that step through
// the matches.
type ThreadSearchView struct {
	// Open is the window's record that the bar is showing. A nil pointer draws
	// nothing, so a window that never opened the bar draws the toolbar the
	// reference shows.
	Open *bool
	// Query is the search text the field edits in place. The shell passes a
	// pointer, so the text survives the next frame while the matches are counted
	// from it; a nil pointer draws nothing.
	Query *string
	// Current is the one-based number of the match the timeline highlights, and
	// Total is how many the query found in the loaded rows. Both are zero while
	// nothing matches, and the bar then reads "0/0". The count is a display copy:
	// it numbers matches in the rows the window shows and claims no history the
	// App Server has not sent.
	Current int
	Total   int
	// Focus is the field's identity, so the shortcut that opens the bar again and
	// the entries that step through the matches put the keyboard focus back in it.
	// A nil pointer leaves the field without one.
	Focus *ui.Handle
	// Locale is the language of the bar's own text: the field's label and
	// placeholder, the counter's description and the three entries. The query
	// itself is the user's text and stays as typed.
	Locale Locale
}

// IsOpen reports whether the window's find bar is showing.
func (v ThreadSearchView) IsOpen() bool { return v.Open != nil && *v.Open }

// TextHighlight is one run of a text the find bar's query matched, in byte
// offsets of that text, as the shell's own matching produced them.
type TextHighlight struct {
	Start int
	End   int
}

// MessageSearch marks a conversation row the find bar's current match is in. The
// timeline draws a marked row washed, so the match shows even where the row's
// text is Markdown or a tool card; Match holds the run of the row's own text when
// that text is drawn as plain text, and is nil for a match in a block or in text
// the shell does not draw itself.
//
// The shell fills it on the copies it builds for the frame; it is never a fact
// the projection reports.
type MessageSearch struct {
	Match *TextHighlight
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
	// Wide is the card's width state, as the shell knows it: true while a
	// conversation is showing, false in the workspace's empty state. It is
	// layout only, read by the region to pick the reference's wider 520 DIP
	// card or the narrower 461 DIP one; the composer reads no message or
	// timeline fact itself.
	Wide bool
	// Workspace is the workspace the next turn would run in, as the shell read
	// it from an App Server fact. Empty means the shell has no such fact yet;
	// the context row then draws the WIP badge and its reason rather than a
	// made-up path.
	Workspace string
	// Permission is the session's default permission mode, as the shell read it
	// from an App Server fact. Empty draws the context row's WIP state rather
	// than an invented mode. The composer offers no control to change it.
	Permission string
	// Branch is the selected session's branch id, as the App Server reports it.
	// It is a session branch, not the workspace's git branch: the shell reads
	// no git fact here, and the region labels it as the session's. Empty draws
	// no branch entry rather than the reference's git "main".
	Branch string
	// ContextKnown reports whether ContextUsed is a real App Server usage fact.
	// The shell has no token-usage fact for the composer yet, so it always
	// passes false and the meter draws the reference's dash instead of a
	// filled track or a percentage.
	ContextKnown bool
	// ContextUsed is the share of the context window the session has used, in
	// 0..1, as the shell read it from an App Server fact. It is read only while
	// ContextKnown is true; an unknown share is never drawn as zero.
	ContextUsed float32
	// Shortcut is the modifier and key that submit the draft. The shell's own
	// focus shortcut (ComposerShortcutMods with ComposerShortcutKey) is not part
	// of this view: it focuses Focus and is handled by the shell.
	Shortcut KeyBinding
	// Focus is the composer's identity, for the shell's focus shortcut.
	Focus *ui.Handle
	// ModelLabel is the current model's display name, provider/model as the
	// shell read the attach reply, or empty when no complete pair was reported.
	// The chip draws the localized unconfigured label in its place.
	ModelLabel string
	// ModelReason explains an absent model: the catalog read's capability or
	// failure reason, or the no-pair fact. It is empty while ModelLabel is set.
	ModelReason string
	// ModelOpen is the window's own record that the model list is showing; the
	// chip toggles it through Actions.OnToggleModelList.
	ModelOpen *bool
	// ModelChoices are the App Server catalog's models in its order, copied for
	// display. The list marks the current pair and shows each model's
	// availability and reason; a selectable row calls Actions.OnSelectModel.
	ModelChoices []ModelChoice
	// Locale is the language of the composer's own text: the editor's label and
	// description, its entries' labels and reasons, and the context row.
	// Placeholder is already resolved by the shell. An empty or unknown value
	// draws English.
	Locale Locale
}

// KeyBinding is one keyboard shortcut as a region listens for it.
type KeyBinding struct {
	// Mods are the modifier keys held with the key; zero means none.
	Mods ui.Modifiers
	// Key is the key pressed with Mods.
	Key ui.Key
}

// renderSidebar draws the session sidebar: the global entries, the grouped
// thread list and the account footer. It writes SidebarView.Selected on a
// choice, then calls Actions.OnSelect; it calls Actions.OnCreateSession from
// "New session" and Actions.OnOpenSearch from the Search entry, which opens the
// window's search palette and sends no request of its own.
func renderSidebar(c *ui.Context, k tokens, v SidebarView, a Actions) {
	sidebar(c, k, v, a)
}

// renderHeader draws the conversation bar and the banner below it. The bar is
// the reference's one line: the project and title breadcrumb, the session menu,
// the conditional Stop and Reconnect entries, the editor-open entry, a divider
// and the context panel toggle. The editor-open entry is WIP: the shell has no
// editor client or workspace fact, so it stays visible, unavailable and
// explained; the model picker lives in the composer, not the bar. It calls
// Actions.OnToggleKeys, Actions.OnEndBranch, Actions.OnRefreshBranches,
// Actions.OnRequestDelete and Actions.OnOpenSettings from its menu,
// Actions.OnDeleteSession from the delete alert's confirm button, and
// Actions.OnStop and Actions.OnReconnect from the entries, each only when
// HeaderView says the entry applies. The banner shows a terminal close, the
// connection error, or the window's action error.
//
// While HeaderView says the find bar is open, the bar draws it at its trailing
// end through renderThreadSearch, and the Stop, Reconnect and context-panel
// entries step aside for it.
func renderHeader(c *ui.Context, k tokens, v HeaderView, a Actions) {
	header(c, k, v, a)
}

// renderThreadSearch draws the conversation bar's find bar: the field, the "n/m"
// counter and the entries that step through the matches and close the bar. It
// edits ThreadSearchView.Query in place and calls Actions.OnThreadSearchNext,
// Actions.OnThreadSearchPrevious and Actions.OnThreadSearchClose; it starts no
// App Server read, because the matches are the shell's display copy of the rows
// on screen. It is a nested region of the toolbar: header.go calls it at the
// bar's trailing end while the window says the bar is open.
func renderThreadSearch(c *ui.Context, k tokens, v ThreadSearchView, a Actions) {
	threadSearchBar(c, k, v, a)
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
// field, with the empty state's context row inside the card and a conversation's
// status band under it. It calls Actions.OnSubmit with the trimmed draft when
// the window's gate and the live draft both allow a send.
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

// renderSearchPalette draws the centered search palette over the window's
// content while SearchPaletteView.Open is set: the input over the scrim, the
// recent-chat heading and one row per result. It writes the palette's own
// highlight through SearchPaletteView.Selected, closes the palette when Escape,
// the scrim or a result asks it to, and calls Actions.OnSelect with the chosen
// session. The results are a display copy of the catalog the shell already
// holds: the palette starts no App Server read.
func renderSearchPalette(c *ui.Context, k tokens, v SearchPaletteView, a Actions) {
	searchPalette(c, k, v, a)
}

// renderContextPanel, in context_panel.go, is the frozen entry of the context
// panel: the changed-file tree and the selected file's diff or preview, drawn
// beside the conversation while the shell's own toggle is on. It writes
// ContextPanelView.Selected on a file click and the directory's fold through
// ContextPanelView.Expanded; it calls no Actions field, because the panel
// displays the caller's copy and starts no App Server read.
