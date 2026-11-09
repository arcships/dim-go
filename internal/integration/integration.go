package integration

import (
	"context"
	"fmt"
	"sync"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/session"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	"github.com/arcships/dimagent-mygo/internal/ui"
	"github.com/arcships/dimagent-mygo/internal/viewmodel"
)

// SessionController owns the transport and in-memory session view for one
// selected session. It is the narrow supervisor -> session -> viewmodel bridge;
// no daemon files or local persistence are consulted.
type SessionController struct {
	Supervisor *supervisor.Supervisor
	Conn       *supervisor.Connection
	Session    *session.Session

	// mu protects the active connection/session pair. Reattach swaps both
	// pointers atomically; readers hold the read lock for the duration of any
	// operation that uses the pair so they cannot observe a half-swapped view.
	mu sync.RWMutex
}

// CreateSession creates a session over the public control connection, then
// opens its dedicated stream and loads history. No local session is persisted.
func CreateSession(ctx context.Context, sup *supervisor.Supervisor, params appserver.CreateParams) (*SessionController, appserver.CreateResult, error) {
	if ctx == nil {
		return nil, appserver.CreateResult{}, fmt.Errorf("integration: nil context")
	}
	if sup == nil {
		return nil, appserver.CreateResult{}, fmt.Errorf("integration: nil supervisor")
	}
	conn, ok := sup.Control()
	if !ok {
		return nil, appserver.CreateResult{}, supervisor.ErrNotStarted
	}
	result, err := appserver.Create(ctx, conn.Client(), params)
	if err != nil {
		return nil, appserver.CreateResult{}, err
	}
	controller, err := OpenSession(ctx, sup, result.SessionID)
	if err != nil {
		return nil, result, err
	}
	return controller, result, nil
}

// OpenSession opens a dedicated stream through sup and loads the first history
// window. The returned controller owns the stream connection until Close.
func OpenSession(ctx context.Context, sup *supervisor.Supervisor, sessionID string) (*SessionController, error) {
	if ctx == nil {
		return nil, fmt.Errorf("integration: nil context")
	}
	if sup == nil {
		return nil, fmt.Errorf("integration: nil supervisor")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("integration: empty session id")
	}
	conn, err := sup.OpenStream(ctx)
	if err != nil {
		return nil, err
	}
	s, err := session.OpenWithOptions(ctx, conn.Client(), conn.Features(), sessionID, session.OpenOptions{CloseTransport: conn.Close})
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &SessionController{Supervisor: sup, Conn: conn, Session: s}, nil
}

// Close releases the session attachment and its supervisor-tracked stream. It
// takes the write lock so it cannot race with an in-flight Submit/Stop read or
// a Reattach swap; the session is detached only after those holders finish.
func (c *SessionController) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("integration: nil context")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Session == nil {
		return nil
	}
	return c.Session.Detach(ctx)
}

// State builds throwaway UI state from catalog and the current session view.
// Facts remain owned by App Server/projection; this function only adapts them.
// A controller without a session returns an empty, disconnected state.
func (c *SessionController) State(summary appserver.SessionSummary) *ui.State {
	return c.StateForBranch(summary, c.SessionBranchID())
}

// SessionBranchID returns the session's selected branch view id.
func (c *SessionController) SessionBranchID() string {
	if c == nil {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return ""
	}
	return c.Session.BranchID()
}

// terminalStatus is the public session lifecycle surface the bridge reads. It
// is intentionally narrow so a session stub can test the closed health state.
type terminalStatus interface {
	Closed() (appserver.ClosedNotice, bool)
	TerminalState() appserver.StreamState
	TerminalReason() string
}

// terminalClosed reports the terminal session.closed health. A transport loss is
// not closed: it is a resync trigger, while session.closed is final.
func terminalClosed(s terminalStatus) (bool, string) {
	if closed, ok := s.Closed(); ok {
		reason := closed.Reason
		if reason == "" {
			reason = s.TerminalReason()
		}
		return true, reason
	}
	if s.TerminalState() == appserver.StreamStateClosed {
		return true, s.TerminalReason()
	}
	return false, ""
}

// StateForBranch builds UI state from an existing session branch view/reducer.
// It does not create or mutate branches.
func (c *SessionController) StateForBranch(summary appserver.SessionSummary, branchID string) *ui.State {
	if c == nil {
		return &ui.State{Connection: ui.ConnUnknown}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return &ui.State{Connection: ui.ConnUnknown}
	}
	connection := ui.ConnReady
	connectionError := ""
	if c.Session.Lost() {
		connection = ui.ConnError
		// StreamState transitions to lost before Session.drain publishes Err.
		// Keep the banner stable across that small synchronization window rather
		// than exposing a blank error during a live refresh tick.
		if err := c.Session.Err(); err != nil {
			connectionError = err.Error()
		} else if reason := c.Session.TerminalReason(); reason != "" {
			connectionError = fmt.Sprintf("session transport lost (%s)", reason)
		} else {
			connectionError = "session transport lost"
		}
	}
	// Only expose a branch that exists in the daemon directory or has been
	// observed by the session reducer. Unknown IDs must not silently render an
	// empty synthetic branch.
	available := map[string]bool{}
	for _, branch := range c.Session.BranchDirectory().Branches {
		available[branch.BranchID] = true
	}
	for _, observed := range c.Session.Branches() {
		available[observed] = true
	}
	if branchID == "" || !available[branchID] {
		branchID = c.Session.BranchID()
	}
	branchView := c.Session.View(branchID)
	snapshot := branchView.Snapshot
	progress, notices := viewmodel.Transient(branchView.Transient)
	sessionClosed, closedReason := terminalClosed(c.Session)
	state := &ui.State{
		Sessions:            []ui.Session{{ID: summary.SessionID, Title: sessionTitle(summary)}},
		Selected:            summary.SessionID,
		SelectedBranch:      branchID,
		Connection:          connection,
		ConnectionError:     connectionError,
		SessionClosed:       sessionClosed,
		SessionClosedReason: closedReason,
		Messages:            append(viewmodel.Build(snapshot), viewmodel.BuildApprovals(c.Session.PendingApprovals(branchID)...)...),
		Run:                 ui.RunIdle,
		ModelProgress:       progress,
		Notices:             notices,
	}
	for _, branch := range c.Session.BranchDirectory().Branches {
		label := branch.BranchID
		if branch.Purpose != nil && *branch.Purpose != "" {
			label = *branch.Purpose
		}
		state.Branches = append(state.Branches, ui.Branch{ID: branch.BranchID, Label: label, State: branch.State, Head: branch.Head, Selected: branch.BranchID == branchID})
	}
	if view := branchView; view.State != nil {
		if view.State.Running {
			state.Run = ui.RunRunning
		}
		if view.State.Stopping {
			state.Run = ui.RunStopping
		}
	} else {
		state.Run = viewmodel.RunStateFrom(branchView.Snapshot)
	}
	return state
}

func sessionTitle(s appserver.SessionSummary) string {
	if s.Title != nil && *s.Title != "" {
		return *s.Title
	}
	if s.Preview != nil && *s.Preview != "" {
		return *s.Preview
	}
	return s.SessionID
}

// LiveState returns a fresh view of the selected session. Session's drain
// goroutine continuously folds live stream events into its in-memory projection;
// this method intentionally rebuilds throwaway UI rows on each call rather than
// introducing a second event/fact source. Callers can invoke it after a render
// tick or event notification to refresh the UI.
func (c *SessionController) LiveState(summary appserver.SessionSummary) *ui.State {
	return c.State(summary)
}

// RespondApproval sends one folded approval decision. It never retries; a cancelled outcome is authoritative.
func (c *SessionController) RespondApproval(ctx context.Context, branchID, requestID string, decision string) (appserver.ApprovalRespondOutcome, error) {
	if c == nil || c.Session == nil {
		return "", fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return "", fmt.Errorf("integration: nil context")
	}
	if branchID == "" || requestID == "" {
		return "", fmt.Errorf("integration: empty approval identity")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Session.RespondApproval(ctx, appserver.ApprovalRespondParams{SessionID: c.Session.SessionID(), BranchID: branchID, RequestID: requestID, Decision: decision})
}

// Submit sends text through the selected session's dedicated App Server
// connection. The session layer owns the submission id and unknown-result
// semantics; this bridge does not retry or manufacture local timeline facts.
func (c *SessionController) Submit(ctx context.Context, text string) (appserver.SubmitResult, error) {
	result, _, err := c.SubmitWithID(ctx, text)
	return result, err
}

// SubmitWithID is Submit with the idempotency key exposed for callers that
// need to read back an unknown result through the App Server protocol.
func (c *SessionController) SubmitWithID(ctx context.Context, text string) (appserver.SubmitResult, string, error) {
	if c == nil {
		return appserver.SubmitResult{}, "", fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return appserver.SubmitResult{}, "", fmt.Errorf("integration: nil context")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return appserver.SubmitResult{}, "", fmt.Errorf("integration: no session")
	}
	return c.Session.SubmitWithID(ctx, text)
}

// Stop asks the App Server to stop the selected session's default branch.
// Running state converges through the session.state stream notification.
// Methods returns the daemon-advertised session method table captured during Open.
func (c *SessionController) Methods() appserver.MethodsResult {
	if c == nil {
		return appserver.MethodsResult{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return appserver.MethodsResult{}
	}
	return c.Session.Methods()
}

// BranchDirectory is always empty: session.branches has no verified upstream
// RPC, so no daemon branch directory is requested or fabricated.
func (c *SessionController) BranchDirectory() appserver.BranchesResult {
	if c == nil {
		return appserver.BranchesResult{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return appserver.BranchesResult{}
	}
	return c.Session.BranchDirectory()
}

func (c *SessionController) Stop(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return fmt.Errorf("integration: nil context")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return fmt.Errorf("integration: no session")
	}
	return c.Session.Stop(ctx)
}

// Reattach reconnects the selected session after transport loss. It rebuilds
// the view from App Server history and never uses the old projection as truth.
func (c *SessionController) Reattach(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return fmt.Errorf("integration: nil context")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Supervisor == nil || c.Session == nil {
		return fmt.Errorf("integration: no session")
	}
	if !c.Session.Lost() {
		return fmt.Errorf("integration: session transport is not lost")
	}
	conn, err := c.Supervisor.OpenStream(ctx)
	if err != nil {
		return err
	}
	fresh, err := session.OpenWithOptions(ctx, conn.Client(), conn.Features(), c.Session.SessionID(), session.OpenOptions{CloseTransport: conn.Close})
	if err != nil {
		conn.Close()
		return err
	}
	old := c.Session
	c.Conn, c.Session = conn, fresh
	old.Release()
	return nil
}

// Catalog reads the authoritative session list over the supervisor control
// connection and adapts it to sidebar rows.
func Catalog(ctx context.Context, sup *supervisor.Supervisor) ([]ui.Session, error) {
	if ctx == nil {
		return nil, fmt.Errorf("integration: nil context")
	}
	if sup == nil {
		return nil, fmt.Errorf("integration: nil supervisor")
	}
	conn, ok := sup.Control()
	if !ok {
		return nil, supervisor.ErrNotStarted
	}
	limit := 200
	result, err := appserver.Catalog(ctx, conn.Client(), appserver.CatalogParams{Limit: &limit}, conn.Features())
	if err != nil {
		return nil, err
	}
	rows := make([]ui.Session, 0, len(result.Sessions))
	for _, summary := range result.Sessions {
		rows = append(rows, ui.Session{ID: summary.SessionID, Title: sessionTitle(summary)})
	}
	return rows, nil
}
