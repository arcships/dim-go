// Package session composes appserver streaming, history paging and projection
// into an in-memory session vertical slice.
//
// The Rust App Server remains the only source of session/timeline/state truth.
// This package keeps a rebuildable view only: history pages and live
// notifications feed the same projection, and nothing is written to local
// storage. Reconnect/resync itself stays outside this package; Open reports a
// transport loss through Done, TerminalState and Err so the caller can trigger
// the domain resync described by docs/protocol-contract.md §14.
package session

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/projection"
)

const (
	// defaultPendingLimit bounds main-branch events buffered between attach and
	// the end of the first-screen history read. Overflow fails Open explicitly;
	// it never silently drops an event.
	defaultPendingLimit = 4096
	// closeWaitTimeout bounds the wait for the stream drain goroutine when the
	// caller supplied no deadline of its own.
	closeWaitTimeout = 5 * time.Second
	// openAbortTimeout bounds the cleanup wait when Open fails after the stream
	// was already opened.
	openAbortTimeout = 5 * time.Second
)

// ErrPendingOverflow marks a first-screen history read that could not keep up
// with live main-branch events. Open fails rather than silently losing facts.
var ErrPendingOverflow = errors.New("session: pending event buffer overflow")

// ErrTransportLost marks a stream whose transport ended without a terminal
// session.closed. It is the domain-resync trigger; the session does not reopen
// by itself.
var ErrTransportLost = errors.New("session: transport lost")

// PendingOverflowError reports the bounded pending buffer overflow.
type PendingOverflowError struct {
	SessionID string
	Limit     int
}

func (e *PendingOverflowError) Error() string {
	return fmt.Sprintf("session: %s: pending event buffer exceeded %d events; rebuild from history", e.SessionID, e.Limit)
}

func (e *PendingOverflowError) Is(target error) bool { return target == ErrPendingOverflow }

// TransportLostError is the typed terminal error for a lost session stream.
type TransportLostError struct {
	SessionID string
	Reason    string
}

func (e *TransportLostError) Error() string {
	if e.Reason == "" {
		return "session: transport lost: " + e.SessionID
	}
	return fmt.Sprintf("session: transport lost: %s (%s)", e.SessionID, e.Reason)
}

func (e *TransportLostError) Is(target error) bool { return target == ErrTransportLost }

// HistoryMode names the first-screen history strategy that was attempted.
type HistoryMode string

const (
	// HistoryModeLatestTurn is the session.history.latestTurn path.
	HistoryModeLatestTurn HistoryMode = "latest_turn"
	// HistoryModeCompatibility is the backward after/through scan used when the
	// latestTurn feature is absent or its turnStart drifts.
	HistoryModeCompatibility HistoryMode = "compatibility"
)

// HistoryStatus describes the first-screen history window. Through is the
// attach-time head that bounded every page request. HasOlder means the daemon
// may have older events outside the window. Degraded means the window was
// deliberately reduced because of history_too_large or the page cap; the
// session still opens.
type HistoryStatus struct {
	Mode      HistoryMode
	Through   int64
	TurnStart *int64
	HasOlder  bool
	Degraded  bool
	Reason    string
	Pages     int
	Fallback  bool
}

func cloneHistoryStatus(in HistoryStatus) HistoryStatus {
	out := in
	if in.TurnStart != nil {
		v := *in.TurnStart
		out.TurnStart = &v
	}
	return out
}

// OpenOptions configures an in-memory session view.
type OpenOptions struct {
	// Clock is used only for local transient observation time. Durable
	// timestamps always come from App Server events.
	Clock projection.Clock
	// PendingLimit bounds main-branch events buffered while history loads.
	PendingLimit int
	// CloseTransport is passed to the dedicated stream so a connection owner can
	// release its tracking. Session still closes the Client it owns.
	CloseTransport func()
}

// branchView is one branch's independent projection and transient state.
// Events from another branch never enter it, so sequences from different
// branches cannot collide or fabricate messages in this branch.
type branchView struct {
	reducer   *projection.Reducer
	transient *projection.TransientReducer
	state     *appserver.StateNotice
	approvals map[string]appserver.ApprovalRequest
	head      int64
}

type pendingEvent struct {
	branchID string
	event    projection.RawEvent
}

// Session is one attached session stream plus its rebuildable views.
//
// Open takes ownership of client: the Client must be dedicated to this Session
// because SessionStream owns its notification/error handoff. On success the
// caller must keep reading Errors (or rely on its bounded buffer) and call
// Close/Detach/Release exactly as the holder semantics require. On failure the
// client is closed, except for ErrStreamAlreadyOpen where the caller violated
// the one-client-per-session ownership rule.
type Session struct {
	client        *appserver.Client
	stream        *appserver.SessionStream
	sessionID     string
	branchID      string
	clock         projection.Clock
	pendingLimit  int
	features      []string
	transientSess *projection.TransientReducer

	mu          sync.Mutex
	branches    map[string]*branchView
	pending     []pendingEvent
	pendingIDs  map[string]struct{}
	pendingErr  error
	ready       bool
	history     HistoryStatus
	closed      *appserver.ClosedNotice
	terminalErr error

	// Optional daemon-provided snapshots read once during Open when the
	// connection advertises the capability. They are only enrichment; a failed
	// optional read never fabricates a local snapshot.
	methods         appserver.MethodsResult
	keySnapshot     appserver.KeysResult
	keysLoaded      bool
	keysStatus      ReadCapabilityStatus
	branchDirectory appserver.BranchesResult
	branchesLoaded  bool
	branchesStatus  ReadCapabilityStatus

	errs       chan error
	errMu      sync.Mutex
	errsClosed bool
	errDropped atomic.Uint64

	done chan struct{}

	closeMu   sync.Mutex
	closeDone chan struct{}
	closeErr  error
}

// Open opens one session stream, loads the first-screen history window, and
// replays notifications that arrived during that read.
//
// client must be a dedicated connection whose hello.features snapshot is
// features; a wrong snapshot silently changes the head/history capability
// gates. Open owns client for the lifetime of the Session.
func Open(ctx context.Context, client *appserver.Client, features []string, sessionID string) (*Session, error) {
	return OpenWithOptions(ctx, client, features, sessionID, OpenOptions{})
}

// OpenWithOptions is Open with an injected clock and pending-buffer limit.
func OpenWithOptions(ctx context.Context, client *appserver.Client, features []string, sessionID string, opts OpenOptions) (*Session, error) {
	if client == nil {
		return nil, fmt.Errorf("session: nil client")
	}
	stream, err := appserver.OpenSessionStream(ctx, client, appserver.SessionStreamConfig{
		SessionID:      sessionID,
		Features:       features,
		CloseTransport: opts.CloseTransport,
	})
	if err != nil {
		// ErrStreamAlreadyOpen means the caller shared a Client with another
		// Session. Closing it here would break that owner, so leave it alone.
		if !errors.Is(err, appserver.ErrStreamAlreadyOpen) {
			client.Close()
		}
		return nil, err
	}
	s := newSession(client, stream, sessionID, opts)
	s.features = append([]string(nil), features...)
	go s.drain()
	if err := s.loadHistory(ctx, features); err != nil {
		s.abortOpen()
		return nil, err
	}
	// These are optional read paths. A daemon that does not advertise a
	// capability is left untouched; advertised optional reads are best-effort
	// enrichment, so a failure degrades to empty state and never fabricates a
	// local snapshot. session.methods remains fail-fast because callers use it as
	// a capability table rather than display enrichment.
	if appserver.HasFeature(features, "session.methods") {
		methods, err := appserver.SessionMethods(ctx, client, appserver.MethodsParams{SessionID: sessionID}, features)
		if err != nil {
			s.abortOpen()
			return nil, err
		}
		s.methods = methods
	}
	if appserver.HasFeature(features, "session.branches") {
		s.readBranchDirectory(ctx)
	} else {
		s.setBranchesStatus(ReadCapabilityStatus{
			Capability: "session.branches",
			Available:  false,
			Reason:     "hello.features does not advertise session.branches",
		})
	}
	if appserver.HasFeature(features, "session.keys") {
		s.readSessionKeys(ctx)
	} else {
		s.setKeysStatus(ReadCapabilityStatus{
			Capability: "session.keys",
			Available:  false,
			Reason:     "hello.features does not advertise session.keys",
		})
	}
	return s, nil
}

func newSession(client *appserver.Client, stream *appserver.SessionStream, sessionID string, opts OpenOptions) *Session {
	if opts.Clock == nil {
		opts.Clock = projection.FixedClock(0)
	}
	if opts.PendingLimit <= 0 {
		opts.PendingLimit = defaultPendingLimit
	}
	s := &Session{
		client:        client,
		stream:        stream,
		sessionID:     sessionID,
		branchID:      appserver.MainBranchID,
		clock:         opts.Clock,
		pendingLimit:  opts.PendingLimit,
		transientSess: projection.NewTransientReducer(sessionID, "", opts.Clock),
		branches:      map[string]*branchView{},
		pendingIDs:    map[string]struct{}{},
		errs:          make(chan error, 16),
		done:          make(chan struct{}),
	}
	// The attach head is the initial current head of the main branch. It stays
	// authoritative until live events advance it.
	main := s.branchLocked(appserver.MainBranchID)
	main.head = stream.Head()
	return s
}

func (s *Session) branchLocked(branchID string) *branchView {
	if branch := s.branches[branchID]; branch != nil {
		return branch
	}
	branch := &branchView{
		reducer:   projection.NewReducer(branchID, s.clock),
		transient: projection.NewTransientReducer(s.sessionID, branchID, s.clock),
		approvals: map[string]appserver.ApprovalRequest{},
	}
	s.branches[branchID] = branch
	return branch
}

// loadHistory applies the first-screen window and then flushes the bounded
// pending buffer. It runs before Open returns; the session is not visible to
// callers while it is in flight.
func (s *Session) loadHistory(ctx context.Context, features []string) error {
	if !appserver.HasFeature(features, "session.history") {
		return &appserver.CapabilityError{Feature: "session.history"}
	}
	head := s.stream.Head()
	var window historyWindow
	var err error
	if appserver.HasFeature(features, "session.history.latestTurn") {
		window, err = s.readLatestTurn(ctx, head)
		if err != nil {
			return s.openFailure(err)
		}
		if window.fallback {
			window, err = s.readCompatibility(ctx, head, "latest_turn_turn_start_drift")
			if err != nil {
				return s.openFailure(err)
			}
		}
	} else {
		window, err = s.readCompatibility(ctx, head, "")
		if err != nil {
			return s.openFailure(err)
		}
	}

	s.mu.Lock()
	for _, event := range window.events {
		if err := s.applyEventLocked(appserver.MainBranchID, event); err != nil {
			s.mu.Unlock()
			return s.openFailure(err)
		}
	}
	s.history = window.status
	pending := s.pending
	s.pending = nil
	s.pendingIDs = nil
	pendingErr := s.pendingErr
	s.ready = true
	for _, item := range pending {
		if err := s.applyEventLocked(item.branchID, item.event); err != nil {
			s.report(err)
		}
	}
	s.mu.Unlock()

	if pendingErr != nil {
		return s.openFailure(pendingErr)
	}
	return s.openFailure(nil)
}

// openFailure prefers the terminal stream state over a lower-level request
// error that raced it. This keeps remote close, transport loss and local detach
// distinguishable during Open.
func (s *Session) openFailure(cause error) error {
	if closed, ok := s.Closed(); ok {
		return &appserver.SessionClosedError{SessionID: s.sessionID, Reason: closed.Reason}
	}
	switch state := s.stream.State(); state {
	case appserver.StreamStateClosed:
		return &appserver.SessionClosedError{SessionID: s.sessionID, Reason: s.stream.Reason()}
	case appserver.StreamStateLost:
		return &appserver.LocalError{AppError: "connection_closed"}
	case appserver.StreamStateDetached:
		return &appserver.LocalError{AppError: "request_aborted"}
	}
	return cause
}

// abortOpen releases a half-opened stream and closes the client Open owns.
func (s *Session) abortOpen() {
	s.stream.Release()
	ctx, cancel := context.WithTimeout(context.Background(), openAbortTimeout)
	defer cancel()
	select {
	case <-s.done:
	case <-ctx.Done():
	}
	s.client.Close()
}

func (s *Session) drain() {
	defer close(s.done)
	for event := range s.stream.Events() {
		s.handle(event)
	}
	for err := range s.stream.Errors() {
		s.report(err)
	}
	s.finishDrain()
}

// finishDrain records the terminal reason and publishes a lost-transport error
// before closing Errors. The terminal error is also available through Err even
// when the buffered Errors channel was full.
func (s *Session) finishDrain() {
	state := s.stream.State()
	reason := s.stream.Reason()
	var terminal error
	if state == appserver.StreamStateLost {
		terminal = &TransportLostError{SessionID: s.sessionID, Reason: reason}
	}
	s.mu.Lock()
	s.terminalErr = terminal
	s.mu.Unlock()
	if terminal != nil {
		s.report(terminal)
	}
	// Serialize channel close with report: report holds errMu while sending.
	// Closing after unlocking races with a sender that passed errsClosed and would
	// panic even though both operations are otherwise non-blocking.
	s.closeErrors()
}

// closeErrors atomically marks and closes the error channel. report uses the
// same mutex, so no sender can race a close and panic.
func (s *Session) closeErrors() {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	if s.errsClosed {
		return
	}
	s.errsClosed = true
	close(s.errs)
}

func (s *Session) handle(event appserver.StreamEvent) {
	if !event.Known {
		s.handleUnknown(event)
		return
	}
	switch payload := event.Payload.(type) {
	case appserver.EventNotice:
		s.handleEvent(payload)
	case appserver.StateNotice:
		s.handleState(payload)
	case appserver.ClosedNotice:
		s.handleClosed(payload)
	case appserver.ApprovalRequest:
		s.handleApprovalRequest(payload, event.Params)
	case appserver.ApprovalResolved:
		s.handleApprovalResolved(payload, event.Params)
	case appserver.ApprovalCancelled:
		s.handleApprovalCancelled(payload, event.Params)
	case appserver.ModelProgress:
		s.handleModelProgress(payload, event.Params)
	case appserver.SessionNotice:
		s.handleSessionNotice(payload, event.Params)
	default:
		// A known method without a typed payload is a contract gap; expose the
		// raw notification instead of dropping it.
		s.handleUnknown(event)
	}
}

// handleEvent keeps the durable timeline branch-local. While the first-screen
// window is loading, main-branch events are buffered with an event.id dedupe
// and a hard cap; side-branch events are independent of that history and apply
// immediately.
func (s *Session) handleEvent(notice appserver.EventNotice) {
	raw, err := AdaptEvent(notice.Event)
	if err != nil {
		s.report(err)
		return
	}
	s.mu.Lock()
	if !s.ready && notice.BranchID == appserver.MainBranchID {
		if _, ok := s.pendingIDs[raw.ID]; ok {
			s.mu.Unlock()
			return
		}
		if len(s.pending) >= s.pendingLimit {
			if s.pendingErr == nil {
				s.pendingErr = &PendingOverflowError{SessionID: s.sessionID, Limit: s.pendingLimit}
			}
			s.mu.Unlock()
			return
		}
		s.pending = append(s.pending, pendingEvent{branchID: notice.BranchID, event: raw})
		s.pendingIDs[raw.ID] = struct{}{}
		s.mu.Unlock()
		return
	}
	err = s.applyEventLocked(notice.BranchID, raw)
	s.mu.Unlock()
	if err != nil {
		s.report(err)
	}
}

func (s *Session) applyEventLocked(branchID string, raw projection.RawEvent) error {
	branch := s.branchLocked(branchID)
	if err := branch.reducer.Apply(raw); err != nil {
		return err
	}
	if raw.Sequence > branch.head {
		branch.head = raw.Sequence
	}
	return nil
}

func (s *Session) handleState(notice appserver.StateNotice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	branch := s.branchLocked(notice.BranchID)
	copy := notice
	branch.state = &copy
}

func (s *Session) handleClosed(notice appserver.ClosedNotice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := notice
	s.closed = &copy
}

func (s *Session) handleApprovalRequest(notice appserver.ApprovalRequest, params json.RawMessage) {
	payload := payloadMap(params, notice)
	s.mu.Lock()
	branch := s.branchLocked(notice.BranchID)
	err := branch.transient.ApplyNotice("approval.request", notice.RequestID, 0, payload)
	if err == nil {
		branch.approvals[notice.RequestID] = notice
	}
	s.mu.Unlock()
	if err != nil {
		s.report(err)
	}
}

func (s *Session) handleApprovalResolved(notice appserver.ApprovalResolved, params json.RawMessage) {
	s.handleApprovalSettlement(notice.BranchID, notice.RequestID, "approval.resolved", params, notice)
}

func (s *Session) handleApprovalCancelled(notice appserver.ApprovalCancelled, params json.RawMessage) {
	s.handleApprovalSettlement(notice.BranchID, notice.RequestID, "approval.cancelled", params, notice)
}

func (s *Session) handleApprovalSettlement(branchID, requestID, kind string, params json.RawMessage, value any) {
	payload := payloadMap(params, value)
	s.mu.Lock()
	branch := s.branchLocked(branchID)
	err := branch.transient.ApplyNotice(kind, requestID, 0, payload)
	if err == nil {
		delete(branch.approvals, requestID)
	}
	s.mu.Unlock()
	if err != nil {
		s.report(err)
	}
}

func (s *Session) handleModelProgress(notice appserver.ModelProgress, params json.RawMessage) {
	var progress projection.ModelProgress
	if err := json.Unmarshal(params, &progress); err != nil {
		s.report(fmt.Errorf("session: session.model.progress: %w: %v", appserver.ErrProtocol, err))
		return
	}
	s.mu.Lock()
	branch := s.branchLocked(notice.BranchID)
	err := branch.transient.ApplyModelProgress(0, progress)
	s.mu.Unlock()
	if err != nil {
		s.report(err)
	}
}

func (s *Session) handleSessionNotice(notice appserver.SessionNotice, params json.RawMessage) {
	payload := payloadMap(params, notice)
	id := notice.Source + ":" + notice.Kind
	if id == ":" {
		id = "session.notice"
	}
	s.mu.Lock()
	err := s.transientSess.ApplyNotice("session.notice", id, 0, payload)
	s.mu.Unlock()
	if err != nil {
		s.report(err)
	}
}

// handleUnknown explicitly exposes notification methods outside the frozen
// contract. Unknown params are retained verbatim; a notification that names a
// different session is rejected instead of being attributed to this view.
func (s *Session) handleUnknown(event appserver.StreamEvent) {
	payload := payloadMap(event.Params, nil)
	if payload != nil {
		if sessionID, ok := payload["sessionId"].(string); ok && sessionID != "" && sessionID != s.sessionID {
			s.report(fmt.Errorf("session: %s notification for session %q on stream for %q: %w", event.Method, sessionID, s.sessionID, appserver.ErrProtocol))
			return
		}
	}
	if event.Method == "" {
		s.report(fmt.Errorf("session: notification without method: %w", appserver.ErrProtocol))
		return
	}
	s.mu.Lock()
	err := s.transientSess.ApplyNotice(event.Method, event.Method, 0, payload)
	s.mu.Unlock()
	if err != nil {
		s.report(err)
	}
}

func payloadMap(raw json.RawMessage, fallback any) map[string]any {
	if len(raw) > 0 && !strings.EqualFold(strings.TrimSpace(string(raw)), "null") {
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err == nil && payload != nil {
			return payload
		}
	}
	if fallback == nil {
		return nil
	}
	data, err := json.Marshal(fallback)
	if err != nil {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil
	}
	return payload
}

func (s *Session) report(err error) {
	if err == nil {
		return
	}
	s.errMu.Lock()
	defer s.errMu.Unlock()
	if s.errsClosed {
		return
	}
	select {
	case s.errs <- err:
	default:
		s.errDropped.Add(1)
	}
}

// Errors delivers session-level protocol reports and the terminal
// TransportLostError. It is bounded; overflow increments DroppedErrors instead
// of blocking. Use Err and TerminalState for the authoritative terminal answer.
func (s *Session) Errors() <-chan error { return s.errs }

// Done is closed once the stream drain goroutine has finished. It is safe to
// wait on after Close/Detach/Release and on a transport loss.
func (s *Session) Done() <-chan struct{} { return s.done }

// DroppedErrors counts every report lost at any stage: this Session's Errors
// buffer, the SessionStream Errors buffer, and the Client.Errors buffer.
func (s *Session) DroppedErrors() uint64 {
	return s.errDropped.Load() + s.stream.DroppedErrors() + s.stream.ClientDroppedErrors()
}

// SessionID is the session this view is attached to.
func (s *Session) SessionID() string { return s.sessionID }

// BranchID is the default branch used by Submit/Stop/RespondApproval when the
// caller does not name one.
func (s *Session) BranchID() string { return s.branchID }

// AttachHead is the attach-time main-branch head. It is the upper bound used by
// the first-screen history read.
func (s *Session) AttachHead() int64 { return s.stream.Head() }

// Head is the current main-branch head: the attach head advanced by every
// durable event observed since. Use Through for a non-main branch.
func (s *Session) Head() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.headLocked(appserver.MainBranchID)
}

// Through is the highest durable sequence observed for branchID, including the
// attach head for the main branch.
func (s *Session) Through(branchID string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.headLocked(branchID)
}

func (s *Session) headLocked(branchID string) int64 {
	branch := s.branches[branchID]
	if branch == nil {
		return 0
	}
	if branchID == appserver.MainBranchID {
		if attachHead := s.stream.Head(); attachHead > branch.head {
			return attachHead
		}
	}
	return branch.head
}

// Attach returns the decoded session.attach reply.
func (s *Session) Attach() appserver.AttachResult { return s.stream.Attach() }

// Ready reports whether the first-screen history read finished.
func (s *Session) Ready() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ready
}

// History returns the first-screen history window status.
func (s *Session) History() HistoryStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneHistoryStatus(s.history)
}

// Snapshot returns the main-branch durable timeline. The returned value is a
// deep copy; concurrent readers cannot mutate reducer state.
func (s *Session) Snapshot() projection.Snapshot { return s.BranchSnapshot(appserver.MainBranchID) }

// BranchSnapshot returns one branch's durable timeline. Branch sequences are
// never merged, so a side-branch event cannot fabricate or evict a main-branch
// message.
func (s *Session) BranchSnapshot(branchID string) projection.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked(branchID)
}

func (s *Session) snapshotLocked(branchID string) projection.Snapshot {
	if branch := s.branches[branchID]; branch != nil {
		return branch.reducer.Snapshot()
	}
	return projection.NewReducer(branchID, s.clock).Snapshot()
}

// Transient returns the main-branch transient projection (approval notices and
// model progress). It never contributes durable messages.
func (s *Session) Transient() projection.TransientSnapshot {
	return s.BranchTransient(appserver.MainBranchID)
}

// BranchTransient returns one branch's transient projection.
func (s *Session) BranchTransient(branchID string) projection.TransientSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	if branch := s.branches[branchID]; branch != nil {
		return branch.transient.Snapshot()
	}
	return projection.NewTransientReducer(s.sessionID, branchID, s.clock).Snapshot()
}

// SessionNotices returns the session-scoped transient notices: session.notice
// payloads and unknown notification methods kept for diagnosis.
func (s *Session) SessionNotices() projection.TransientSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transientSess.Snapshot()
}

// PendingApprovals returns the branch's unresolved approvals in request-id
// order. Resolved/cancelled notices remove an entry; there is no local
// approval truth beyond copying the daemon's replay and settlement notices.
func (s *Session) PendingApprovals(branchID string) []appserver.ApprovalRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	branch := s.branches[branchID]
	if branch == nil || len(branch.approvals) == 0 {
		return nil
	}
	out := make([]appserver.ApprovalRequest, 0, len(branch.approvals))
	for _, approval := range branch.approvals {
		out = append(out, approval)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].RequestID < out[j].RequestID })
	return out
}

// Methods returns the daemon-advertised session method table captured during
// Open. An absent capability yields a zero result.
func (s *Session) Methods() appserver.MethodsResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return appserver.MethodsResult{Methods: append([]appserver.MethodDeclaration(nil), s.methods.Methods...)}
}

// ReadCapabilityStatus describes an optional capability read. Available is true
// only after a successful validated reply. Reason is empty on success and
// otherwise names the missing hello.features gate or the read/decoder failure.
type ReadCapabilityStatus struct {
	Capability string
	Available  bool
	Reason     string
}

// Keys returns the session.keys snapshot captured during Open. Entries are
// copied; ok is false when the feature was not advertised or the optional read
// failed. No local snapshot is synthesized.
func (s *Session) Keys() (appserver.KeysResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.keysLoaded {
		return appserver.KeysResult{}, false
	}
	out := appserver.KeysResult{
		Through: cloneInt64(s.keySnapshot.Through),
		Entries: append([]appserver.SessionKeyEntry(nil), s.keySnapshot.Entries...),
	}
	return out, true
}

// KeysStatus reports whether session.keys is available and, if not, why.
func (s *Session) KeysStatus() ReadCapabilityStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.keysStatus
	if status.Capability == "" {
		status.Capability = "session.keys"
	}
	return status
}

// BranchDirectory returns the session.branches snapshot captured during Open.
// The value is copied; an unavailable or failed read yields an empty directory.
func (s *Session) BranchDirectory() appserver.BranchesResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.branchesLoaded {
		return appserver.BranchesResult{}
	}
	return appserver.BranchesResult{
		Branches: append([]appserver.BranchInfo(nil), s.branchDirectory.Branches...),
	}
}

// BranchDirectoryStatus reports whether session.branches is available and, if
// not, why. Branches() remains the event-derived fallback.
func (s *Session) BranchDirectoryStatus() ReadCapabilityStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.branchesStatus
	if status.Capability == "" {
		status.Capability = "session.branches"
	}
	return status
}

// Branches returns the branches observed so far from attach notifications, sorted.
// This is projection-local observation, not a daemon branch directory. Main is
// always present after Open.
func (s *Session) Branches() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.branches))
	for branchID := range s.branches {
		out = append(out, branchID)
	}
	sort.Strings(out)
	return out
}

// View is one consistent read of a branch's durable and transient state.
type View struct {
	BranchID  string
	Snapshot  projection.Snapshot
	Transient projection.TransientSnapshot
	State     *appserver.StateNotice
	Head      int64
	Ready     bool
	History   HistoryStatus
	Closed    *appserver.ClosedNotice
}

// View returns an atomic read, including the current head, so a timer render
// and an event-driven render cannot observe a half-updated projection.
func (s *Session) View(branchID string) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	view := View{
		BranchID:  branchID,
		Snapshot:  s.snapshotLocked(branchID),
		Transient: projection.NewTransientReducer(s.sessionID, branchID, s.clock).Snapshot(),
		Head:      s.headLocked(branchID),
		Ready:     s.ready,
		History:   cloneHistoryStatus(s.history),
	}
	if branch := s.branches[branchID]; branch != nil {
		view.Transient = branch.transient.Snapshot()
		if branch.state != nil {
			copy := *branch.state
			view.State = &copy
		}
	}
	if s.closed != nil {
		copy := *s.closed
		view.Closed = &copy
	}
	return view
}

// State returns the main branch's last-arrival run state. State and events are
// unordered per docs/protocol-contract.md §8; this is a last-arrival record.
func (s *Session) State() (appserver.StateNotice, bool) {
	return s.BranchState(appserver.MainBranchID)
}

// BranchState returns one branch's last-arrival run state. A side branch never
// overwrites the main branch's running indicator.
func (s *Session) BranchState(branchID string) (appserver.StateNotice, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	branch := s.branches[branchID]
	if branch == nil || branch.state == nil {
		return appserver.StateNotice{}, false
	}
	return *branch.state, true
}

// Closed reports the terminal session.closed notice when the daemon sent one.
func (s *Session) Closed() (appserver.ClosedNotice, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed == nil {
		return appserver.ClosedNotice{}, false
	}
	return *s.closed, true
}

// TerminalState reports the stream lifecycle: attached, closed, detached or
// lost.
func (s *Session) TerminalState() appserver.StreamState { return s.stream.State() }

// TerminalReason reports the session.closed reason or the transport_closed loss
// reason.
func (s *Session) TerminalReason() string { return s.stream.Reason() }

// Lost reports whether the transport ended without a terminal session.closed.
func (s *Session) Lost() bool { return s.stream.State() == appserver.StreamStateLost }

// Err returns the terminal TransportLostError for a lost stream and nil for a
// remote close or a deliberate local release.
func (s *Session) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.terminalErr
}

// Close is the ordinary holder release: session.detach is fire-and-forget with
// a 500ms bound and the reply is ignored, then the owned client is closed. It
// never waits for the 2s delete-path reply.
func (s *Session) Close(ctx context.Context) error { return s.close(ctx, false) }

// Detach is the delete/explicit path: it sends session.detach and waits for the
// reply (2s default) before closing. The first close semantics win; later
// Close/Detach calls return the same result.
func (s *Session) Detach(ctx context.Context) error { return s.close(ctx, true) }

// Release is Close without a caller context. It is safe to call from cleanup
// paths that cannot wait.
func (s *Session) Release() { _ = s.close(context.Background(), false) }

func (s *Session) close(ctx context.Context, detach bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.closeMu.Lock()
	if s.closeDone == nil {
		s.closeDone = make(chan struct{})
		done := s.closeDone
		s.closeMu.Unlock()
		err := s.performClose(ctx, detach)
		s.closeMu.Lock()
		s.closeErr = err
		close(done)
		s.closeMu.Unlock()
		return err
	}
	done := s.closeDone
	s.closeMu.Unlock()
	select {
	case <-done:
		s.closeMu.Lock()
		err := s.closeErr
		s.closeMu.Unlock()
		return err
	case <-ctx.Done():
		return contextError(ctx)
	}
}

func (s *Session) performClose(ctx context.Context, detach bool) error {
	var detachErr error
	if detach {
		detachErr = s.stream.Detach(ctx)
	} else {
		s.stream.Release()
	}
	waitErr := s.waitDone(ctx)
	// Open owns the client; closing it after the stream is done releases the
	// transport and the one-stream slot without leaving request goroutines.
	s.client.Close()
	if detachErr != nil {
		return detachErr
	}
	return waitErr
}

func (s *Session) waitDone(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, closeWaitTimeout)
		defer cancel()
	}
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return contextError(ctx)
	}
}

func contextError(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &appserver.LocalError{AppError: "request_timeout", Err: ctx.Err()}
	}
	return &appserver.LocalError{AppError: "request_aborted", Err: ctx.Err()}
}

// Submit sends a text submission on the default branch with a fresh UUID
// submission id. It never retries; unknown results stay observable.
func (s *Session) Submit(ctx context.Context, text string) (appserver.SubmitResult, error) {
	result, _, err := s.SubmitWithID(ctx, text)
	return result, err
}

// SubmitWithID sends a submission and returns its UUID for unknown-result read-back.
func (s *Session) SubmitWithID(ctx context.Context, text string) (appserver.SubmitResult, string, error) {
	return s.SubmitBranchWithID(ctx, s.branchID, text)
}

// SubmitBranch sends a text submission on branchID with a fresh UUID
// submission id. submissionId is the idempotency key; a caller must read the
// input back instead of resending when the result is unknown.
func (s *Session) SubmitBranch(ctx context.Context, branchID, text string) (appserver.SubmitResult, error) {
	result, _, err := s.SubmitBranchWithID(ctx, branchID, text)
	return result, err
}

// SubmitBranchWithID sends a text submission on branchID and returns the UUID
// that was sent, so an unknown result can be read back with session.input_lookup
// without resending.
func (s *Session) SubmitBranchWithID(ctx context.Context, branchID, text string) (appserver.SubmitResult, string, error) {
	if branchID == "" {
		branchID = s.branchID
	}
	submissionID, err := newUUID()
	if err != nil {
		return appserver.SubmitResult{}, "", err
	}
	result, err := appserver.Submit(ctx, s.client, appserver.SubmitParams{
		SessionID:    s.sessionID,
		SubmissionID: submissionID,
		BranchID:     branchID,
		Text:         text,
	})
	return result, submissionID, err
}

// Stop stops the default branch.
func (s *Session) Stop(ctx context.Context) error { return s.StopBranch(ctx, s.branchID) }

// StopBranch stops one branch. Stop state converges through session.state.
func (s *Session) StopBranch(ctx context.Context, branchID string) error {
	if branchID == "" {
		branchID = s.branchID
	}
	return appserver.Stop(ctx, s.client, appserver.StopParams{SessionID: s.sessionID, BranchID: branchID})
}

// RespondApproval sends an approval decision. An empty BranchID defaults to the
// session's branch. A -32602 reply is reported as cancelled by the appserver
// helper, not as a local success.
func (s *Session) RespondApproval(ctx context.Context, params appserver.ApprovalRespondParams) (appserver.ApprovalRespondOutcome, error) {
	params.SessionID = s.sessionID
	if params.BranchID == "" {
		params.BranchID = s.branchID
	}
	return appserver.ApprovalRespond(ctx, s.client, params)
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
