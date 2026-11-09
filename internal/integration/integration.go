package integration

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/session"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	"github.com/arcships/dimagent-mygo/internal/ui"
	"github.com/arcships/dimagent-mygo/internal/viewmodel"
)

// Fail-closed reasons a re-attach or a replay gap is refused. They are exported
// so callers can report the WIP boundary instead of retrying blindly.
var (
	// ErrSessionClosed means the daemon sent the terminal session.closed; a
	// closed session is final and is never re-attached (contract §8).
	ErrSessionClosed = errors.New("integration: session is closed")
	// ErrSessionDetached means the attachment was released locally; re-attach
	// would resurrect a deliberately released controller.
	ErrSessionDetached = errors.New("integration: session was detached")
	// ErrSessionNotLost means a resync is not legal yet: the old transport is
	// still live (or its loss has not been observed), so re-attaching would
	// double-attach the same session.
	ErrSessionNotLost = errors.New("integration: session transport has not been lost")
	// ErrDaemonRestarted means the daemon identity changed while a re-attach was
	// in flight; the fresh attachment belongs to a superseded instance and is
	// discarded instead of installed as a stale view.
	ErrDaemonRestarted = errors.New("integration: daemon instance changed during re-attach")
	// ErrReplayGap marks a subscription_changed the bridge refuses to replay.
	// The upstream recovery contract is unverified, so the bridge fails closed
	// instead of looping or inventing local facts.
	ErrReplayGap = errors.New("integration: subscription_changed replay gap")
)

// ReplayGapError reports a subscription_changed that was not recovered. A
// read-only request may re-attach and retry exactly once (contract §13); a
// write is never replayed. Anything that survives that allowance is surfaced
// here so the UI can show the WIP boundary.
type ReplayGapError struct {
	Op       string
	ReadOnly bool
	Retried  bool
	Err      error
}

func (e *ReplayGapError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.ReadOnly {
		return fmt.Sprintf("integration: %s: subscription_changed survived one read-only re-attach retry; upstream recovery contract is unverified (WIP), failing closed: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("integration: %s: subscription_changed on a write; not replayed automatically (WIP fail-closed): %v", e.Op, e.Err)
}

func (e *ReplayGapError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *ReplayGapError) Is(target error) bool { return target == ErrReplayGap }

// isSubscriptionChanged reports whether the daemon classified the failure as a
// replay gap.
func isSubscriptionChanged(err error) bool {
	return err != nil && appserver.IsAppError(err, appserver.AppErrorSubscriptionChanged)
}

// retryReadOnce runs one read-only attempt and, only when the daemon reports
// subscription_changed, retries exactly once. Writes never pass through here.
// A replay gap that survives the single allowed retry becomes a fail-closed
// *ReplayGapError rather than another loop.
func retryReadOnce(ctx context.Context, op string, attempt func() error) error {
	err := attempt()
	if !isSubscriptionChanged(err) {
		return err
	}
	if ctx != nil && ctx.Err() != nil {
		return err
	}
	retryErr := attempt()
	if retryErr == nil {
		return nil
	}
	if isSubscriptionChanged(retryErr) {
		return &ReplayGapError{Op: op, ReadOnly: true, Retried: true, Err: retryErr}
	}
	return retryErr
}

// replayGapWrite marks a write failure caused by subscription_changed. The
// write is reported, never resent: a retry could duplicate a submission or an
// approval decision.
func replayGapWrite(op string, err error) error {
	if !isSubscriptionChanged(err) {
		return err
	}
	return &ReplayGapError{Op: op, Err: err}
}

// SessionController owns the transport and in-memory session view for one
// selected session. It is the narrow supervisor -> session -> viewmodel bridge;
// no daemon files or local persistence are consulted.
type SessionController struct {
	Supervisor *supervisor.Supervisor
	Conn       *supervisor.Connection
	Session    *session.Session

	// mu protects the active connection/session pair, the terminal closed flag
	// and the re-attach record. Reattach swaps both pointers atomically; readers
	// hold the read lock for the duration of any operation that uses the pair so
	// they cannot observe a half-swapped view.
	mu sync.RWMutex

	// reattachMu serializes re-attach attempts without holding mu across dial,
	// attach and history I/O, so State/Submit readers and Close stay responsive.
	reattachMu sync.Mutex

	// closed is set by Close and is terminal. A Reattach that finishes its dial
	// after Close must discard its fresh attachment instead of resurrecting a
	// released controller.
	closed bool

	lastReattach    ReattachResult
	hasLastReattach bool
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
		// session.create is a write: a subscription_changed reply is reported, not
		// retried, so a duplicate session can never be created silently.
		return nil, appserver.CreateResult{}, replayGapWrite("session.create", err)
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
	var conn *supervisor.Connection
	var s *session.Session
	// attach + first history window is a read-only operation: a daemon
	// subscription_changed gets the contract's single re-attach retry.
	if err := retryReadOnce(ctx, "session.attach", func() error {
		var attemptErr error
		conn, s, attemptErr = openStreamSession(ctx, sup, sessionID)
		return attemptErr
	}); err != nil {
		return nil, err
	}
	return &SessionController{Supervisor: sup, Conn: conn, Session: s}, nil
}

// openStreamSession opens one stream connection and attaches a fresh session on
// it. A failed session.Open closes the connection it was handed: OpenStream
// already succeeded, so the stream must never outlive a failed attach.
func openStreamSession(ctx context.Context, sup *supervisor.Supervisor, sessionID string) (*supervisor.Connection, *session.Session, error) {
	conn, err := sup.OpenStream(ctx)
	if err != nil {
		return nil, nil, err
	}
	fresh, err := session.OpenWithOptions(ctx, conn.Client(), conn.Features(), sessionID, session.OpenOptions{CloseTransport: conn.Close})
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return conn, fresh, nil
}

// Close releases the session attachment and its supervisor-tracked stream. It
// takes the write lock so it cannot race with an in-flight Submit/Stop read or
// a Reattach swap; the session is detached only after those holders finish.
// Close is terminal: a Reattach that raced it must discard its fresh attachment
// and never resurrect the controller.
func (c *SessionController) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("integration: nil context")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
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

// RespondApproval sends one folded approval decision. It never retries; a
// cancelled outcome is authoritative. A subscription_changed reply is reported
// as a fail-closed replay gap instead of resending the decision.
func (c *SessionController) RespondApproval(ctx context.Context, branchID, requestID string, decision string) (appserver.ApprovalRespondOutcome, error) {
	if c == nil {
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
	if c.Session == nil {
		return "", fmt.Errorf("integration: no session")
	}
	outcome, err := c.Session.RespondApproval(ctx, appserver.ApprovalRespondParams{SessionID: c.Session.SessionID(), BranchID: branchID, RequestID: requestID, Decision: decision})
	return outcome, replayGapWrite("approval.respond", err)
}

// Submit sends text through the selected session's dedicated App Server
// connection. The session layer owns the submission id and unknown-result
// semantics; this bridge does not retry or manufacture local timeline facts.
func (c *SessionController) Submit(ctx context.Context, text string) (appserver.SubmitResult, error) {
	result, _, err := c.SubmitWithID(ctx, text)
	return result, err
}

// SubmitWithID is Submit with the idempotency key exposed for callers that
// need to read back an unknown result through the App Server protocol. A
// subscription_changed reply is never resent: the caller gets the fail-closed
// replay-gap error and must resolve the submission through session.input_lookup.
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
	result, submissionID, err := c.Session.SubmitWithID(ctx, text)
	return result, submissionID, replayGapWrite("session.submit", err)
}

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

// Stop asks the App Server to stop the selected session's default branch.
// Running state converges through the session.state notification. A
// subscription_changed reply is reported, never replayed.
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
	return replayGapWrite("session.stop", c.Session.Stop(ctx))
}

// ReattachResult reports the daemon identity observed by a successful
// Reattach. A restarted daemon is not an error: the fresh attachment is rebuilt
// from the new instance's history and the old projection is dropped.
type ReattachResult struct {
	SessionID          string
	PreviousGeneration uint64
	Generation         uint64
	PreviousInstanceID string
	InstanceID         string
	Restarted          bool
}

// LastReattach returns the identity observed by the most recent successful
// Reattach.
func (c *SessionController) LastReattach() (ReattachResult, bool) {
	if c == nil {
		return ReattachResult{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.hasLastReattach {
		return ReattachResult{}, false
	}
	return c.lastReattach, true
}

// Reattach reconnects the selected session after transport loss. It rebuilds
// the view from App Server history and never uses the old projection as truth.
//
// Fail-closed rules:
//
//   - session.closed and a local detach are terminal and never re-attached;
//   - the old transport must have ended (lost, or Connection.Done fired) so a
//     live attachment is never double-attached;
//   - a re-attach that observes a newer daemon instance is either rebuilt from
//     that instance (when the swap is still current) or discarded with
//     ErrDaemonRestarted when another connection superseded it mid-flight;
//   - a Close that wins the race makes this attempt discard its fresh
//     attachment instead of resurrecting the controller.
//
// The dial/attach/history work runs outside the controller lock; only the
// pointer swap is atomic.
func (c *SessionController) Reattach(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return fmt.Errorf("integration: nil context")
	}
	c.reattachMu.Lock()
	defer c.reattachMu.Unlock()

	c.mu.Lock()
	if c.Supervisor == nil || c.Session == nil {
		c.mu.Unlock()
		return fmt.Errorf("integration: no session")
	}
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("%w: the controller was closed", ErrSessionClosed)
	}
	old, oldConn := c.Session, c.Conn
	if err := reattachBlocked(old, oldConn, c.Supervisor); err != nil {
		c.mu.Unlock()
		return err
	}
	sessionID := old.SessionID()
	c.mu.Unlock()

	var conn *supervisor.Connection
	var fresh *session.Session
	// Attach + history is read-only, so subscription_changed gets the
	// contract's single re-attach retry; anything past that fails closed.
	err := retryReadOnce(ctx, "session.attach", func() error {
		var attemptErr error
		conn, fresh, attemptErr = openStreamSession(ctx, c.Supervisor, sessionID)
		return attemptErr
	})
	if err != nil {
		return err
	}
	// Another connection may have observed a newer daemon instance while this
	// re-attach was in flight. Installing the fresh attachment would present a
	// superseded instance's history, so discard it and fail closed.
	if conn.Stale() {
		generation := conn.Generation()
		discardStream(fresh, conn)
		return fmt.Errorf("%w: connection generation %d was superseded by generation %d", ErrDaemonRestarted, generation, c.Supervisor.Generation())
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		discardStream(fresh, conn)
		return fmt.Errorf("%w: the controller was closed during re-attach", ErrSessionClosed)
	}
	if c.Session != old {
		// Defensive: only Close and Reattach mutate the pair, and both are
		// excluded above. Never install over a pair this attempt did not read.
		c.mu.Unlock()
		discardStream(fresh, conn)
		return fmt.Errorf("%w: the session changed during re-attach", ErrSessionNotLost)
	}
	result := ReattachResult{
		SessionID:  sessionID,
		Generation: conn.Generation(),
		InstanceID: conn.InstanceID(),
	}
	if oldConn != nil {
		result.PreviousGeneration = oldConn.Generation()
		result.PreviousInstanceID = oldConn.InstanceID()
		result.Restarted = oldConn.InstanceID() != conn.InstanceID()
	} else {
		result.PreviousInstanceID = conn.PreviousInstanceID()
		result.Restarted = conn.Restarted()
	}
	c.Conn, c.Session = conn, fresh
	c.lastReattach, c.hasLastReattach = result, true
	c.mu.Unlock()

	// The swap is atomic: readers under mu now see the fresh pair. Release the
	// old attachment outside the lock; session.detach is bounded but must not
	// block State/Submit on the new pair.
	old.Release()
	return nil
}

// reattachBlocked reports why the current attachment cannot be replaced. A
// domain resync is only legal after the transport ended (contract §14);
// session.closed and a local detach are final and never re-attempted.
func reattachBlocked(s *session.Session, conn *supervisor.Connection, sup *supervisor.Supervisor) error {
	if closed, ok := s.Closed(); ok {
		reason := closed.Reason
		if reason == "" {
			reason = s.TerminalReason()
		}
		return fmt.Errorf("%w: %s", ErrSessionClosed, reason)
	}
	switch s.TerminalState() {
	case appserver.StreamStateClosed:
		return fmt.Errorf("%w: %s", ErrSessionClosed, s.TerminalReason())
	case appserver.StreamStateDetached:
		return fmt.Errorf("%w: the attachment was released", ErrSessionDetached)
	}
	if s.Lost() {
		return nil
	}
	// The transport can end before the session router publishes lost.
	// Connection.Done is the upstream resync trigger, so treat it as lost
	// instead of refusing a legitimate resync during that window.
	if conn != nil && conn.Closed() {
		return nil
	}
	if conn != nil && conn.Stale() {
		current := uint64(0)
		if sup != nil {
			current = sup.Generation()
		}
		return fmt.Errorf("%w: daemon generation advanced from %d to %d while the attachment is still live; wait for the old transport to end", ErrSessionNotLost, conn.Generation(), current)
	}
	return fmt.Errorf("%w: transport is still attached", ErrSessionNotLost)
}

// discardStream releases a fresh attachment that must not be installed. Session
// release closes the owned client; Connection.Close also drops supervisor
// tracking. Both are idempotent.
func discardStream(s *session.Session, conn *supervisor.Connection) {
	if s != nil {
		s.Release()
	}
	if conn != nil {
		conn.Close()
	}
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
	var result appserver.CatalogResult
	// session.catalog is a read-only call: subscription_changed gets exactly one
	// retry, then fails closed as a replay gap.
	if err := retryReadOnce(ctx, "session.catalog", func() error {
		var attemptErr error
		result, attemptErr = appserver.Catalog(ctx, conn.Client(), appserver.CatalogParams{Limit: &limit}, conn.Features())
		return attemptErr
	}); err != nil {
		return nil, err
	}
	rows := make([]ui.Session, 0, len(result.Sessions))
	for _, summary := range result.Sessions {
		rows = append(rows, ui.Session{ID: summary.SessionID, Title: sessionTitle(summary)})
	}
	return rows, nil
}
