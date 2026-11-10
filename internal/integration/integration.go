package integration

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
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
		Sessions:            []ui.Session{{ID: summary.SessionID, Title: sessionTitle(summary, c.Session.Title())}},
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
		Keys:                keysPanelView(c.Session),
	}
	for _, branch := range c.Session.BranchDirectory().Branches {
		// Label from contract fields only: branchId is the stable identity, and a
		// side chat's parent is the one relation the directory proves. There is no
		// purpose/title field in the public entry, so none is invented.
		label := branch.BranchID
		if branch.ParentBranchID != "" {
			label += " ← " + branch.ParentBranchID
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

// keysPanelView adapts the session's session.keys snapshot to the panel's
// display type. Only a snapshot the session actually read counts as loaded; a
// failed or unavailable read leaves Loaded false so the panel says WIP instead
// of drawing an empty list as if the branch had no keys.
func keysPanelView(s *session.Session) ui.KeysPanelView {
	if s == nil {
		return ui.KeysPanelView{}
	}
	result, ok := s.Keys()
	if !ok {
		return ui.KeysPanelView{}
	}
	view := ui.KeysPanelView{Loaded: true}
	if result.Through != nil {
		view.Through = *result.Through
	}
	view.Entries = make([]ui.KeysPanelEntry, 0, len(result.Entries))
	for _, entry := range result.Entries {
		view.Entries = append(view.Entries, ui.KeysPanelEntry{
			Key:      entry.Key,
			Sequence: entry.Sequence,
			InputID:  entry.InputID,
			Text:     entry.Text,
		})
	}
	return view
}

// sessionTitle picks the display label. The session's observed title wins when
// it is newer than the catalog summary's, so a rename reply or a title notice
// updates the row without waiting for the next catalog read; the summary is
// otherwise the seed.
func sessionTitle(summary appserver.SessionSummary, live session.TitleState) string {
	title := mergeTitle(summary, live)
	if title.Known && title.Text != nil && *title.Text != "" {
		return *title.Text
	}
	if summary.Preview != nil && *summary.Preview != "" {
		return *summary.Preview
	}
	// No observed label yet: the UI localizes the placeholder. Returning the
	// session id here would leak a truncated opaque id into the shell.
	return ""
}

// mergeTitle orders the catalog entry's title triple against the live title
// metadata by revision. The public contract keeps whichever revision is higher:
// a list can be read before a commit and still reach the client after that
// commit's notice.
func mergeTitle(summary appserver.SessionSummary, live session.TitleState) session.TitleState {
	base := summaryTitleState(summary)
	if live.Known && (!base.Known || live.Revision > base.Revision) {
		return live
	}
	return base
}

// summaryTitleState converts the optional v3 title fields of a summary entry. A
// daemon that predates the fields omits all three; that is unknown, not a
// cleared title.
func summaryTitleState(summary appserver.SessionSummary) session.TitleState {
	if summary.Title == nil && summary.TitleSource == nil && summary.TitleRevision == nil {
		return session.TitleState{}
	}
	state := session.TitleState{
		Text:   cloneString(summary.Title),
		Source: cloneString(summary.TitleSource),
		Known:  true,
	}
	if summary.TitleRevision != nil {
		state.Revision = *summary.TitleRevision
	}
	return state
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
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

// SubmitBranch sends text on one explicit branch of the selected session, which
// is what the window shows after it opened that branch. A caller that already
// knows the branch must not rely on the session's default: the displayed branch
// and the submitted branch have to be the same App Server fact.
func (c *SessionController) SubmitBranch(ctx context.Context, branchID, text string) (appserver.SubmitResult, error) {
	result, _, err := c.SubmitBranchWithID(ctx, branchID, text)
	return result, err
}

// SubmitBranchWithID is SubmitBranch with the idempotency key exposed, so an
// unknown result is read back on the same branch through session.input_lookup
// instead of being resent. A subscription_changed reply is never replayed.
func (c *SessionController) SubmitBranchWithID(ctx context.Context, branchID, text string) (appserver.SubmitResult, string, error) {
	if c == nil {
		return appserver.SubmitResult{}, "", fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return appserver.SubmitResult{}, "", fmt.Errorf("integration: nil context")
	}
	if branchID == "" {
		return appserver.SubmitResult{}, "", fmt.Errorf("integration: empty branch id")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return appserver.SubmitResult{}, "", fmt.Errorf("integration: no session")
	}
	result, submissionID, err := c.Session.SubmitBranchWithID(ctx, branchID, text)
	return result, submissionID, replayGapWrite("session.submit", err)
}

// LookupInput reads a submission back by its idempotency key after an unknown
// submit result (§10). It never resends the submission and never caches the
// answer: every call asks the App Server and decodes the tagged union, so an
// unverified payload can never be mistaken for an accepted input. An accepted
// lookup means the daemon found the submitted input, not that the turn
// succeeded.
func (c *SessionController) LookupInput(ctx context.Context, branchID, submissionID string) (appserver.InputLookupResult, error) {
	if c == nil {
		return appserver.InputLookupResult{}, fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return appserver.InputLookupResult{}, fmt.Errorf("integration: nil context")
	}
	if submissionID == "" {
		return appserver.InputLookupResult{}, fmt.Errorf("integration: empty submission id")
	}
	// An empty branchID is not an error: Session.LookupInput supplies the
	// session's current branch, matching how Submit does. A caller that knows
	// the branch it submitted on uses LookupInputBranch instead.
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return appserver.InputLookupResult{}, fmt.Errorf("integration: no session")
	}
	return decodeInputLookup(c.Session, ctx, branchID, submissionID)
}

// LookupInputBranch reads a submission back on one explicit branch. It is the
// read-back a branch-aware submit needs: the window names the branch the App
// Server received, so a side-branch submission is never resolved against the
// session's default branch. The branch id is required and the submission is
// never resent.
func (c *SessionController) LookupInputBranch(ctx context.Context, branchID, submissionID string) (appserver.InputLookupResult, error) {
	if c == nil {
		return appserver.InputLookupResult{}, fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return appserver.InputLookupResult{}, fmt.Errorf("integration: nil context")
	}
	if branchID == "" {
		return appserver.InputLookupResult{}, fmt.Errorf("integration: empty branch id")
	}
	if submissionID == "" {
		return appserver.InputLookupResult{}, fmt.Errorf("integration: empty submission id")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return appserver.InputLookupResult{}, fmt.Errorf("integration: no session")
	}
	return decodeInputLookup(c.Session, ctx, branchID, submissionID)
}

// decodeInputLookup asks the App Server for one submission and decodes the
// tagged union. The caller holds the read lock.
func decodeInputLookup(s *session.Session, ctx context.Context, branchID, submissionID string) (appserver.InputLookupResult, error) {
	raw, err := s.LookupInput(ctx, branchID, submissionID)
	if err != nil {
		return appserver.InputLookupResult{}, err
	}
	result, err := appserver.DecodeInputLookupResult(raw)
	if err != nil {
		return appserver.InputLookupResult{}, fmt.Errorf("integration: session.input_lookup: %w", err)
	}
	return result, nil
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

// Rename sets or clears the selected session's manual title through
// session.rename. The write is never retried and never turned into a local
// title: an unsupported method (unknown_method), invalid input, a conflicting
// revision or an unknown result is reported, and a subscription_changed reply
// fails closed as a replay gap.
//
// The title the daemon committed is authoritative: the returned revision orders
// it against the catalog summary and any title notice, and Title/State use that
// order.
func (c *SessionController) Rename(ctx context.Context, title *string) (appserver.RenameResult, error) {
	if c == nil {
		return appserver.RenameResult{}, fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return appserver.RenameResult{}, fmt.Errorf("integration: nil context")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return appserver.RenameResult{}, fmt.Errorf("integration: no session")
	}
	result, err := c.Session.Rename(ctx, title)
	return result, replayGapWrite("session.rename", err)
}

// AttachmentPut stores one text attachment through attachment.put. The write is
// sent once and never retried: the method has no idempotency key and no
// list/get read-back, so a lost or malformed reply is reported as an unknown
// result for the user to decide on. A daemon without the Attachments plugin
// answers unknown_method, surfaced as *appserver.MutationError.Unsupported (a
// visible-disabled WIP gap), and a subscription_changed reply fails closed as a
// replay gap.
func (c *SessionController) AttachmentPut(ctx context.Context, name, text string) (appserver.AttachmentPutResult, error) {
	if c == nil {
		return appserver.AttachmentPutResult{}, fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return appserver.AttachmentPutResult{}, fmt.Errorf("integration: nil context")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return appserver.AttachmentPutResult{}, fmt.Errorf("integration: no session")
	}
	result, err := c.Session.AttachmentPut(ctx, name, text)
	return result, replayGapWrite("attachment.put", err)
}

// Title returns the selected session's newest known title metadata, ordered
// against the catalog summary by revision. It reads only what the daemon
// supplied through a rename reply, a SessionStore title notice, or the summary
// entry; a session that has observed none returns Known=false.
func (c *SessionController) Title(summary appserver.SessionSummary) session.TitleState {
	live := session.TitleState{}
	if c != nil {
		c.mu.RLock()
		if c.Session != nil {
			live = c.Session.Title()
		}
		c.mu.RUnlock()
	}
	return mergeTitle(summary, live)
}

// BranchDirectory returns the session.branches snapshot captured by Open. It is
// empty when hello.features did not advertise the capability or the read failed;
// the daemon's directory is never synthesized locally.
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

// Delete asks the App Server to delete the selected session. The controller
// keeps its view until the daemon confirms; the caller removes the row only
// after a nil error. A subscription_changed reply is a fail-closed replay gap
// like every other mutation: the delete is never resent.
func (c *SessionController) Delete(ctx context.Context) error {
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
	return replayGapWrite("session.delete", c.Session.Delete(ctx))
}

// Keys returns the session.keys snapshot captured by Open. It is empty when the
// daemon did not advertise the capability or the optional read failed; the
// snapshot is never synthesized locally. The read lock keeps the copy from
// racing a Reattach swap.
func (c *SessionController) Keys() appserver.KeysResult {
	if c == nil {
		return appserver.KeysResult{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return appserver.KeysResult{}
	}
	keys, ok := c.Session.Keys()
	if !ok {
		return appserver.KeysResult{}
	}
	return keys
}

// Models reads the model catalog through the selected session's App Server
// connection. provider.management/provider.selection are process-owned
// non-mutating reads, so this bridge adds no cache and no local choice: every
// call returns the daemon's current answer. A missing capability is reported as
// *appserver.CapabilityError so the picker stays visible-disabled with a reason;
// a subscription_changed reply gets the read-only single retry.
func (c *SessionController) Models(ctx context.Context) (session.ModelCatalog, error) {
	if c == nil {
		return session.ModelCatalog{}, fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return session.ModelCatalog{}, fmt.Errorf("integration: nil context")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return session.ModelCatalog{}, fmt.Errorf("integration: no session")
	}
	var catalog session.ModelCatalog
	err := retryReadOnce(ctx, "provider.management", func() error {
		var attemptErr error
		catalog, attemptErr = c.Session.Models(ctx)
		return attemptErr
	})
	if err != nil {
		return session.ModelCatalog{}, err
	}
	return catalog, nil
}

// Model returns the model pair the attach reply reported for the selected
// session. It never falls back to the default selection; a picker shows the
// session pair and the default as separate facts.
func (c *SessionController) Model() (session.ModelSelection, bool) {
	if c == nil {
		return session.ModelSelection{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return session.ModelSelection{}, false
	}
	return c.Session.Model()
}

// CanUpdateModel reports whether this session's App Server connection advertises
// session.model.update. The window keeps the picker WIP while it is false.
func (c *SessionController) CanUpdateModel() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return false
	}
	return c.Session.CanUpdateModel()
}

// UpdateModel switches the selected session's main-branch model. The write is
// never retried: a subscription_changed reply fails closed as a replay gap, and
// an unknown result is reported so the caller can read the pair back through
// RefreshModel instead of resending.
func (c *SessionController) UpdateModel(ctx context.Context, providerID, modelID string) (appserver.ModelUpdateResult, error) {
	if c == nil {
		return appserver.ModelUpdateResult{}, fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return appserver.ModelUpdateResult{}, fmt.Errorf("integration: nil context")
	}
	if providerID == "" || modelID == "" {
		return appserver.ModelUpdateResult{}, fmt.Errorf("integration: empty model target")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return appserver.ModelUpdateResult{}, fmt.Errorf("integration: no session")
	}
	result, err := c.Session.UpdateModel(ctx, providerID, modelID)
	return result, replayGapWrite("session.model.update", err)
}

// RefreshModel reads the selected session's current model pair back from
// session.summaries and stores it. It is the read side of a model switch and of
// an unknown model.update result; the request's pair is never used as a local
// selection. The read is non-mutating, so subscription_changed gets the
// contract's single retry.
func (c *SessionController) RefreshModel(ctx context.Context) (session.ModelSelection, bool, error) {
	if c == nil {
		return session.ModelSelection{}, false, fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return session.ModelSelection{}, false, fmt.Errorf("integration: nil context")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return session.ModelSelection{}, false, fmt.Errorf("integration: no session")
	}
	var (
		selection session.ModelSelection
		ok        bool
	)
	err := retryReadOnce(ctx, "session.summaries", func() error {
		var attemptErr error
		selection, ok, attemptErr = c.Session.RefreshModel(ctx)
		return attemptErr
	})
	if err != nil {
		return session.ModelSelection{}, false, err
	}
	return selection, ok, nil
}

// OpenBranch loads one branch's first-screen history into that branch's own
// projection. Branch switching is not its own RPC (docs/branch-attachment-plan.md
// §1.2): it is the session.history {branchId} read, and the session layer bounds
// it with the daemon-reported branch head.
//
// The known-branch check, idempotency and capability handling live in
// session.OpenBranch. The attach/history read is read-only, so a
// subscription_changed reply gets the contract's single retry before failing
// closed as a replay gap.
func (c *SessionController) OpenBranch(ctx context.Context, branchID string) (session.BranchOpenResult, error) {
	if c == nil {
		return session.BranchOpenResult{}, fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return session.BranchOpenResult{}, fmt.Errorf("integration: nil context")
	}
	if branchID == "" {
		return session.BranchOpenResult{}, fmt.Errorf("integration: empty branch id")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return session.BranchOpenResult{}, fmt.Errorf("integration: no session")
	}
	var result session.BranchOpenResult
	err := retryReadOnce(ctx, "session.open_branch", func() error {
		var attemptErr error
		result, attemptErr = c.Session.OpenBranch(ctx, branchID)
		return attemptErr
	})
	return result, err
}

// RefreshBranchDirectory re-reads session.branches into the read-only snapshot.
// A missing capability or a failed read is reported; no local directory is
// synthesized and previously read entries are not deleted by the caller.
func (c *SessionController) RefreshBranchDirectory(ctx context.Context) error {
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
	return retryReadOnce(ctx, "session.branches", func() error {
		return c.Session.RefreshBranchDirectory(ctx)
	})
}

// EndBranch asks the App Server to end one non-main branch. The reply is not
// translated into local branch state: the directory, projection and head are
// dropped or updated only when the daemon reports the new state. An already
// ended branch returns AlreadyEnded with a nil error (do not retry); an older
// host reports an *appserver.MutationError matching ErrCapabilityUnavailable so
// the entry can stay visible-disabled WIP.
func (c *SessionController) EndBranch(ctx context.Context, branchID string) (appserver.BranchEndOutcome, error) {
	if c == nil {
		return "", fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return "", fmt.Errorf("integration: nil context")
	}
	if branchID == "" {
		return "", fmt.Errorf("integration: empty branch id")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return "", fmt.Errorf("integration: no session")
	}
	outcome, err := c.Session.EndBranch(ctx, branchID)
	return outcome, replayGapWrite("session.branch_end", err)
}

// BranchCreate asks the App Server to create a host branch (side chat) from
// parentBranchID at the parent sequence a row resolved. It never creates a
// local branch: the returned branch is the App Server's, and the caller opens it
// through OpenBranch before showing it.
//
// session.branch_create is a write. A subscription_changed reply is reported as
// a fail-closed replay gap, and an unknown result stays an
// *appserver.UnknownResultError for the caller to resolve through
// session.branches; neither is resent.
func (c *SessionController) BranchCreate(ctx context.Context, parentBranchID string, at int64, prompt string) (appserver.BranchCreateResult, error) {
	if c == nil {
		return appserver.BranchCreateResult{}, fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return appserver.BranchCreateResult{}, fmt.Errorf("integration: nil context")
	}
	if parentBranchID == "" {
		return appserver.BranchCreateResult{}, fmt.Errorf("integration: empty parent branch id")
	}
	if at < 0 {
		return appserver.BranchCreateResult{}, fmt.Errorf("integration: branch cut point %d is negative", at)
	}
	if strings.TrimSpace(prompt) == "" {
		return appserver.BranchCreateResult{}, fmt.Errorf("integration: empty branch prompt")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return appserver.BranchCreateResult{}, fmt.Errorf("integration: no session")
	}
	result, err := c.Session.BranchCreate(ctx, parentBranchID, at, prompt)
	return result, replayGapWrite("session.branch_create", err)
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

// StopBranch asks the App Server to stop one explicit branch, which is the
// branch the window shows after it opened it. A stop names the branch the user
// is reading instead of the session's default, so a side-branch turn is the one
// that ends. Running state still converges through session.state; nothing here
// is replayed or cached.
func (c *SessionController) StopBranch(ctx context.Context, branchID string) error {
	if c == nil {
		return fmt.Errorf("integration: no session")
	}
	if ctx == nil {
		return fmt.Errorf("integration: nil context")
	}
	if branchID == "" {
		return fmt.Errorf("integration: empty branch id")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Session == nil {
		return fmt.Errorf("integration: no session")
	}
	return replayGapWrite("session.stop", c.Session.StopBranch(ctx, branchID))
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
		// The project heading and the row's relative time are catalog facts, not
		// a local list: cwd's base name and updatedAt, as the daemon returned
		// them. An empty cwd stays ungrouped instead of inventing a name.
		group := ""
		if summary.Cwd != "" {
			group = filepath.Base(summary.Cwd)
		}
		rows = append(rows, ui.Session{
			ID:        summary.SessionID,
			Title:     sessionTitle(summary, session.TitleState{}),
			Group:     group,
			UpdatedAt: summary.UpdatedAt,
		})
	}
	return rows, nil
}
