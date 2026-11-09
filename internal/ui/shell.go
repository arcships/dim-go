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
	// Workspace and Permission are the composer context row's display copies:
	// the workspace the next turn would run in and the session's default
	// permission mode, as the App Server reported them. The shell has no such
	// fact yet, so they stay empty and the composer draws its WIP state; they
	// are never filled from a local guess.
	Workspace  string
	Permission string

	// OnSelect, OnSelectBranch, OnEndBranch, OnRefreshBranches,
	// OnCreateSession, OnSubmit, OnStop, OnReconnect and OnApproval connect the
	// shell's actions to the caller's session controller. They are the only
	// wiring source of the Actions a region receives; a nil field leaves its
	// action in the view without an effect, as a fixture does, and the region
	// keeps its WIP entry. The callback owns the App Server call and the facts
	// it returns, so the shell never reports an action it did not send.
	// OnCreateSession asks the caller to run session.create and to publish the
	// new session in the catalog and the selection once the App Server returned
	// it. OnSelectBranch asks the caller to read that branch's history and to
	// show it only once the App Server returned it. OnEndBranch asks the caller
	// to end one branch and to remove its row only after session.branches
	// reports it gone. OnRefreshBranches asks the caller to re-read that
	// directory. OnApproval receives one decision for one request: the caller
	// sends approval.respond and lets the session projection settle the row.
	OnSelect          func(sessionID string)
	OnSelectBranch    func(branchID string)
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
	// ShowSettings is the window's own record that the appearance surface is
	// showing instead of the conversation. It is display state only.
	ShowSettings bool
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
	a := s.actions()
	composer := s.composerView()
	// Escape leaves the keys browser wherever the keyboard focus is. It is read
	// before the regions build so the frame already shows the conversation the
	// key returns to.
	if s.ShowKeys && c.Shortcut(0, ui.KeyEscape) {
		s.ShowKeys = false
	}
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
		// appearance surface) is drawn inside it, so they share the frame.
		ui.Column(c).FillHeight().Padding(k.framePad, k.framePad, k.framePad, 0).Children(func() {
			ui.Column(c).Grow(1).FillWidth().
				Radius(k.frameRadius).BorderWidth(k.hairline).BorderColor(k.border).
				Background(k.card).Role(ui.RoleGroup).Label("Main area").
				Children(func() {
					// The reference has no application bar in the empty state: the
					// session column's card starts at the frame and the hero carries
					// the heading. With a session the toolbar is its own region, built
					// from HeaderView. An action or connection failure still gets its
					// banner in either state, so an empty window never hides why a
					// preference save or a request failed.
					if s.Selected != "" {
						renderHeader(c, k, s.headerView(), a)
					} else {
						banner(c, k, s.headerView())
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
							case s.ShowSettings:
								// The appearance surface takes the conversation's place; the
								// sidebar and toolbar stay, so a preference change is
								// visible around it. The capability list is assembled here,
								// from the window's own callback wiring, so the surface only
								// draws it.
								renderSettings(c, k, s.Settings, wipCapabilities(s), s.applySettings)
							case s.ShowKeys:
								// The keys browser takes the conversation's place. It reads the
								// snapshot the caller supplied and sends no request.
								s.Keys.Selected = &s.keysSelected
								renderKeysPanel(c, k, &s.Keys, a)
							default:
								// An empty, loaded and open session shows the hero instead of the
								// projection's empty row; a closed session or a pending approval
								// keeps the timeline's own states, so the hero never speaks over
								// facts that are still on screen.
								if !timeline.Loading && !timeline.Closed && len(timeline.Messages) == 0 && len(pending) == 0 {
									emptyState(c, k, composer, s.locale(), a)
								} else {
									renderTimeline(c, k, timeline, a)
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
	return s.ShowContextPanel && s.Selected != "" && !s.ShowSettings && !s.ShowKeys
}

// contextPanelColumn draws the context panel beside the conversation in the
// width the layout gives it. The view keeps the caller's pointers and fold map,
// and only its Width is filled here, so a click still writes the caller's
// selection and folds.
func (s *State) contextPanelColumn(c *ui.Context, k tokens, a Actions) {
	width := clampContextPanelWidth(contextPanelW)
	ui.Column(c).Width(width).Shrink(1).MinWidth(0).FillHeight().Clip().Children(func() {
		v := s.ContextPanel
		v.Width = width
		renderContextPanel(c, k, v, a)
	})
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
// folds the type size, family, code size and density into the frame's theme and
// tokens, and the sidebar width is held between its bounds before it is drawn.
func (s *State) applySettings(change SettingsChange) {
	s.Settings = SettingsView{
		Theme:        change.Theme,
		FontSize:     change.FontSize,
		FontFamily:   change.FontFamily,
		SidebarWidth: change.SidebarWidth,
		Density:      change.Density,
		CodeFontSize: change.CodeFontSize,
		Locale:       change.Locale,
	}
	if change.SidebarWidth > 0 {
		s.SidebarWidth = change.SidebarWidth
	}
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
		// The settings toggle is the shell's own state, not a caller callback, so
		// it is always wired. Opening it takes the main column from the keys
		// browser, so the two never compete for it.
		OnOpenSettings: func() {
			s.ShowSettings = !s.ShowSettings
			if s.ShowSettings {
				s.ShowKeys = false
			}
		},
		// The context panel toggle is the shell's own state too: the panel
		// displays the caller's ContextPanel copy and sends no request, so the
		// entry needs no caller callback. It stays wired and the toolbar offers
		// it only while a session is selected and the conversation is showing.
		OnToggleContextPanel: func() {
			s.ShowContextPanel = !s.ShowContextPanel
		},
	}
	// The keys toggle is the shell's own state too, but it is wired only when the
	// caller supplied a session.keys snapshot. The sidebar no longer draws the
	// entry the reference lacks; the toggle stays for a caller that sets
	// State.ShowKeys or re-introduces an entry elsewhere.
	if s.Keys.Loaded {
		a.OnToggleKeys = func() {
			s.ShowKeys = !s.ShowKeys
			if s.ShowKeys {
				s.ShowSettings = false
			}
		}
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
		Locale:   s.locale(),
		state:    &s.sidebar,
	}
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
		Title:            s.selectedTitle(),
		SessionID:        s.Selected,
		HasSession:       s.Selected != "",
		SelectedBranch:   s.SelectedBranch,
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
		SettingsOpen:     s.ShowSettings,
		KeysOpen:         s.ShowKeys,
		ContextPanelOpen: s.contextPanelShown(),
		Locale:           s.locale(),
	}
	// The confirmation belongs to the selection it was opened for. A selection
	// change is dropped by syncDeleteConfirm before the frame builds; the guard
	// keeps a stale alert out of a frame that has not run it.
	if s.DeleteConfirm && s.deleteFor == s.Selected {
		v.DeleteConfirm = &s.DeleteConfirm
	}
	return v
}

// timelineView copies the rows and the window's loading and close facts into
// the timeline's view, with pointers to the list, fold and Markdown state the
// region keeps across frames. Pending approvals are not rows: they take the
// composer's slot, so they are left out here and read by pendingApprovals.
func (s *State) timelineView() TimelineView {
	rows := make([]Message, 0, len(s.Messages))
	for _, m := range s.Messages {
		if m.Author == AuthorApproval {
			continue
		}
		rows = append(rows, m)
	}
	return TimelineView{
		Messages: rows,
		Loading:  s.ConversationLoading,
		Closed:   s.SessionClosed,
		list:     &s.conversation,
		folds:    &s.folds,
	}
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
	}
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
	return ""
}
