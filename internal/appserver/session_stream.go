package appserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MainBranchID is the branch the attach head fallback reads (contract §8
// step 4).
const MainBranchID = "main"

// Detach timings from contract §8: the session.delete path waits for the
// session.detach reply, an ordinary holder release does not.
const (
	detachReplyTimeout   = 2 * time.Second
	detachReleaseTimeout = 500 * time.Millisecond
)

// streamEventBuffer bounds the Events channel. The router blocks instead of
// dropping, so a slow consumer applies backpressure to the daemon reader rather
// than losing durable events.
const streamEventBuffer = 64

// streamErrorBuffer bounds the Errors channel; overflow is counted by
// DroppedErrors instead of blocked, mirroring Client.Errors.
const streamErrorBuffer = 8

// StreamReasonTransportClosed is the Reason reported when the transport ends
// without a terminal session.closed.
const StreamReasonTransportClosed = "transport_closed"

// StreamState is the deterministic lifecycle of one session stream.
//
//	attaching -> attached   attach reply and head resolved
//	attaching -> closed     session.closed arrived before Open returned
//	attached  -> closed     session.closed (terminal; never a resync trigger)
//	attached  -> detached   Detach/Release by this client
//	attached  -> lost       transport closed without session.closed (resync)
//
// detached, closed and lost are terminal; once reached they are not replaced.
type StreamState int

const (
	// StreamStateAttaching is the handshake window before Open returns.
	StreamStateAttaching StreamState = iota
	// StreamStateAttached means the attach reply and head are known and
	// notifications are routed in arrival order.
	StreamStateAttached
	// StreamStateDetached means this client released the attachment.
	StreamStateDetached
	// StreamStateClosed means the daemon sent the terminal session.closed.
	StreamStateClosed
	// StreamStateLost means the transport ended without a terminal notice.
	// Contract §14 requires a domain resync: reopen the stream, hello, attach
	// and rebuild from history. This package does not reopen by itself.
	StreamStateLost
)

func (s StreamState) String() string {
	switch s {
	case StreamStateAttaching:
		return "attaching"
	case StreamStateAttached:
		return "attached"
	case StreamStateDetached:
		return "detached"
	case StreamStateClosed:
		return "closed"
	case StreamStateLost:
		return "lost"
	default:
		return fmt.Sprintf("StreamState(%d)", int(s))
	}
}

// Terminal reports whether the state can no longer change. A lost stream still
// needs a resync; closed and detached do not.
func (s StreamState) Terminal() bool {
	return s == StreamStateDetached || s == StreamStateClosed || s == StreamStateLost
}

// ErrSessionClosed marks OpenSessionStream failing because the daemon reported
// the session closed (contract §8: terminal, no resync).
var ErrSessionClosed = errors.New("appserver: session closed")

// SessionClosedError is the terminal session.closed observed before the stream
// finished opening. It unwraps to ErrSessionClosed.
type SessionClosedError struct {
	SessionID string
	Reason    string
}

func (e *SessionClosedError) Error() string {
	if e.Reason == "" {
		return "appserver: session closed: " + e.SessionID
	}
	return fmt.Sprintf("appserver: session closed: %s (%s)", e.SessionID, e.Reason)
}

func (e *SessionClosedError) Unwrap() error { return ErrSessionClosed }

// StreamEvent is one notification routed by a SessionStream, in arrival order.
// Params is verbatim so unknown methods keep every field (contract §4 rule 6);
// Payload is the decoded §11 type when Known.
type StreamEvent struct {
	Method  string
	Params  json.RawMessage
	Payload any
	Known   bool
}

// SessionStreamConfig fixes the session, the connection-level feature snapshot
// and the transport closer for one stream.
type SessionStreamConfig struct {
	// SessionID is the session this dedicated connection streams.
	SessionID string
	// Features must be the hello.features snapshot of the connection the
	// client belongs to. It gates the history head fallback.
	Features []string
	// CloseTransport closes the dedicated connection. The caller should pass its
	// connection owner's Close (for example supervisor.Connection.Close) so that
	// tracking is released as well. Nil closes the Client, which closes the
	// underlying io.Closer.
	CloseTransport func()
}

// SessionStream owns one dedicated session connection: it subscribes to
// notifications before calling session.attach, resolves the main-branch head
// (attach.sequence, or the session.history fallback), and routes this session's
// notifications in arrival order.
//
// It takes ownership of Client.Notifications and Client.Errors for the lifetime
// of the stream. The caller must keep draining Events, otherwise the router
// blocks and request replies stop flowing.
//
// It does not build a projection, read the latest turn window, read
// session.branches/session.keys, or reopen after loss; those stay with the
// caller and are listed as WIP in docs/protocol-contract.md.
type SessionStream struct {
	client         *Client
	sessionID      string
	closeTransport func()

	subscribed chan struct{}
	ready      chan struct{}
	finished   chan struct{}
	stop       chan struct{}

	events  chan StreamEvent
	errs    chan error
	dropped atomic.Uint64

	// attached is set by Open after the attach reply and head are resolved. The
	// router buffers until then, then flushes in arrival order.
	attached atomic.Bool

	mu     sync.Mutex
	state  StreamState
	reason string
	branch map[string]StateNotice

	stopOnce   sync.Once
	connOnce   sync.Once
	finishOnce sync.Once

	// Router-only state, owned by the route goroutine.
	pending  []StreamEvent
	eventSeq map[eventKey]string
	eventIDs map[string]int64

	attach AttachResult
	head   int64
}

type eventKey struct {
	branchID string
	sequence int64
}

// OpenSessionStream starts the notification router, waits until it is
// subscribed, and only then sends session.attach (contract §8 step 2-3). It
// resolves the head as attach.sequence, or, when an older daemon omits it, the
// through of session.history {branchId:"main", after:0, limit:1}.
//
// Every failure closes the dedicated connection: a half-opened stream is
// discarded and the caller opens a new one. A session.closed that arrives
// during the handshake is returned as *SessionClosedError.
func OpenSessionStream(ctx context.Context, client *Client, cfg SessionStreamConfig) (*SessionStream, error) {
	if client == nil {
		return nil, errors.New("appserver: session stream requires a client")
	}
	if err := requireNonEmpty("sessionId", cfg.SessionID); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, localContextError(ctx)
	}
	closeTransport := cfg.CloseTransport
	if closeTransport == nil {
		closeTransport = client.Close
	}

	s := &SessionStream{
		client:         client,
		sessionID:      cfg.SessionID,
		closeTransport: closeTransport,
		subscribed:     make(chan struct{}),
		ready:          make(chan struct{}),
		finished:       make(chan struct{}),
		stop:           make(chan struct{}),
		events:         make(chan StreamEvent, streamEventBuffer),
		errs:           make(chan error, streamErrorBuffer),
		state:          StreamStateAttaching,
		branch:         make(map[string]StateNotice),
		eventSeq:       make(map[eventKey]string),
		eventIDs:       make(map[string]int64),
	}
	go s.route()
	select {
	case <-s.subscribed:
	case <-ctx.Done():
		s.abort()
		return nil, localContextError(ctx)
	case <-client.Done():
		s.abort()
		return nil, &LocalError{AppError: "connection_closed"}
	}

	raw, err := client.Call(ctx, "session.attach", AttachParams{SessionID: cfg.SessionID})
	if err != nil {
		// A terminal notice may have raced the attach reply; report the more
		// specific session state when it is already known.
		s.abort()
		if closedErr := s.closedError(); closedErr != nil {
			return nil, closedErr
		}
		return nil, err
	}
	attach, err := DecodeAttachResult(raw)
	if err != nil {
		s.abort()
		return nil, err
	}
	head, err := s.resolveHead(ctx, attach, cfg.Features)
	if err != nil {
		s.abort()
		if closedErr := s.closedError(); closedErr != nil {
			return nil, closedErr
		}
		return nil, err
	}

	state, reason := s.finalize(attach, head)
	switch state {
	case StreamStateClosed:
		s.abort()
		return nil, &SessionClosedError{SessionID: s.sessionID, Reason: reason}
	case StreamStateLost:
		s.abort()
		return nil, &LocalError{AppError: "connection_closed"}
	case StreamStateDetached:
		// Released while opening; nothing to hand out.
		s.abort()
		return nil, &LocalError{AppError: "request_aborted"}
	}
	return s, nil
}

// finalize moves attaching to attached without overwriting a terminal state the
// router recorded during the handshake.
func (s *SessionStream) finalize(attach AttachResult, head int64) (StreamState, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StreamStateAttaching {
		return s.state, s.reason
	}
	s.state = StreamStateAttached
	s.attach = attach
	s.head = head
	s.attached.Store(true)
	close(s.ready)
	return StreamStateAttached, ""
}

// resolveHead implements contract §8 step 4. A missing attach.sequence on an
// older daemon falls back to the main branch's history head; a missing
// session.history feature is a visible capability error, never a local head.
func (s *SessionStream) resolveHead(ctx context.Context, attach AttachResult, features []string) (int64, error) {
	if attach.Sequence != nil {
		return *attach.Sequence, nil
	}
	if !HasFeature(features, "session.history") {
		return 0, &CapabilityError{Feature: "session.history"}
	}
	after := int64(0)
	limit := 1
	page, err := HistoryPage(ctx, s.client, HistoryParams{
		SessionID: s.sessionID,
		BranchID:  MainBranchID,
		After:     &after,
		Limit:     &limit,
	})
	if err != nil {
		return 0, err
	}
	// HistoryPage already enforces the §9 invariants, so Through is present.
	return *page.Through, nil
}

func (s *SessionStream) closedError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StreamStateClosed {
		return nil
	}
	return &SessionClosedError{SessionID: s.sessionID, Reason: s.reason}
}

// Events delivers routed notifications in arrival order. The channel closes on
// every terminal state; State reports why.
func (s *SessionStream) Events() <-chan StreamEvent { return s.events }

// Errors delivers stream-level protocol reports (decode failures, notifications
// for another session, duplicate-sequence conflicts) and transport errors.
// Overflow increments DroppedErrors instead of blocking.
func (s *SessionStream) Errors() <-chan error { return s.errs }

// DroppedErrors counts stream error reports that could not be buffered.
func (s *SessionStream) DroppedErrors() uint64 { return s.dropped.Load() }

// SessionID is the session this stream is attached to.
func (s *SessionStream) SessionID() string { return s.sessionID }

// State reports the current lifecycle state.
func (s *SessionStream) State() StreamState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// Reason reports the session.closed reason, or the local loss reason for a
// lost stream. It is empty while attached.
func (s *SessionStream) Reason() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reason
}

// Attach returns the decoded session.attach reply. It is meaningful only after
// Open returned successfully.
func (s *SessionStream) Attach() AttachResult { return s.attach }

// Head is the main-branch head resolved during Open.
func (s *SessionStream) Head() int64 { return s.head }

// BranchState returns the last session.state seen for branchID. This is a
// last-arrival record, not a projection: contract §8 forbids assuming order
// between session.event and session.state.
func (s *SessionStream) BranchState(branchID string) (StateNotice, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notice, ok := s.branch[branchID]
	return notice, ok
}

// Detach is the session.delete path: it sends session.detach and waits up to
// the context deadline (default 2s) for the reply, then closes the connection
// whether or not the reply succeeded (contract §8). It is a no-op when the
// stream is already detached, closed or lost.
func (s *SessionStream) Detach(ctx context.Context) error {
	return s.detach(ctx, detachReplyTimeout, true)
}

// Release is the ordinary holder release: it sends session.detach with a 500ms
// bound, ignores the reply, and closes the connection (contract §8).
func (s *SessionStream) Release() {
	_ = s.detach(context.Background(), detachReleaseTimeout, false)
}

func (s *SessionStream) detach(ctx context.Context, fallback time.Duration, wantReply bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !s.markDetached() {
		s.closeConnection()
		return nil
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, fallback)
		defer cancel()
	}
	_, err := s.client.Call(ctx, "session.detach", DetachParams{SessionID: s.sessionID})
	s.closeConnection()
	if wantReply {
		return err
	}
	return nil
}

func (s *SessionStream) markDetached() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StreamStateAttaching && s.state != StreamStateAttached {
		return false
	}
	s.state = StreamStateDetached
	s.reason = ""
	return true
}

// abort closes the connection and waits for the router to finish. Open calls it
// on every failure so no goroutine or attachment outlives a failed open.
func (s *SessionStream) abort() {
	s.closeConnection()
	<-s.finished
}

func (s *SessionStream) closeConnection() {
	s.stopOnce.Do(func() { close(s.stop) })
	s.connOnce.Do(s.closeTransport)
}

// route subscribes first, then routes until a terminal state. Binding the
// subscription happens before Open calls session.attach: Open waits for
// subscribed.
func (s *SessionStream) route() {
	defer s.finish()
	close(s.subscribed)
	ready := s.ready
	for {
		if s.attached.Load() && len(s.pending) > 0 {
			if !s.flushPending() {
				return
			}
			ready = nil
			continue
		}
		select {
		case notification := <-s.client.Notifications():
			if !s.handle(notification) {
				return
			}
		case <-ready:
			// The handshake finished; the next loop iteration flushes the
			// pending buffer in arrival order.
			ready = nil
		case err := <-s.client.Errors():
			s.report(err)
		case <-s.client.Done():
			return
		case <-s.stop:
			return
		}
	}
}

// finish closes the public channels once the router stops. An unexpected stop
// while attaching or attached is a lost transport, which contract §14 turns
// into a domain resync; closed and detached reasons win.
func (s *SessionStream) finish() {
	s.mu.Lock()
	if s.state == StreamStateAttaching || s.state == StreamStateAttached {
		s.state = StreamStateLost
		s.reason = StreamReasonTransportClosed
	}
	s.mu.Unlock()
	s.finishOnce.Do(func() {
		close(s.events)
		close(s.errs)
	})
	close(s.finished)
}

// handle routes one notification. It returns false when the stream must stop
// (session.closed).
func (s *SessionStream) handle(n Notification) bool {
	payload, known, err := DecodeNotification(n)
	if err != nil {
		s.report(fmt.Errorf("appserver: %s: %w", n.Method, err))
		return true
	}
	if known {
		if sessionID, ok := noticeSessionID(payload); ok && sessionID != s.sessionID {
			s.report(protocolErrorf("sessionId", "%s notification for %q on stream for %q", n.Method, sessionID, s.sessionID))
			return true
		}
		switch notice := payload.(type) {
		case EventNotice:
			if !s.acceptEvent(notice) {
				return true
			}
		case StateNotice:
			s.mu.Lock()
			s.branch[notice.BranchID] = notice
			s.mu.Unlock()
		case ClosedNotice:
			return s.handleClosed(notice, n.Params)
		}
	}
	return s.emit(StreamEvent{Method: n.Method, Params: n.Params, Payload: payload, Known: known})
}

// handleClosed applies the terminal session.closed rule (§8): emit the notice,
// release the attachment by closing the connection, and never resync. A
// concurrent release keeps its detached state.
func (s *SessionStream) handleClosed(n ClosedNotice, params json.RawMessage) bool {
	s.mu.Lock()
	if s.state == StreamStateAttaching || s.state == StreamStateAttached {
		s.state = StreamStateClosed
		s.reason = n.Reason
	}
	s.mu.Unlock()
	s.emit(StreamEvent{
		Method:  "session.closed",
		Params:  append(json.RawMessage(nil), params...),
		Payload: n,
		Known:   true,
	})
	s.closeConnection()
	return false
}

// acceptEvent enforces §8 step 7 idempotency at the routing layer: an event is
// deduplicated by event.id, and a branch sequence must map to exactly one id.
// A same-sequence/different-id conflict is a protocol error: the second event is
// dropped and reported instead of blindly merged (contract §15 still waits for
// upstream semantics). Projection must keep its own (branchId, sequence) guard.
func (s *SessionStream) acceptEvent(n EventNotice) bool {
	key := eventKey{branchID: n.BranchID, sequence: n.Event.Sequence}
	if id, ok := s.eventSeq[key]; ok {
		if id != n.Event.ID {
			s.report(protocolErrorf("session.event", "branch %q sequence %d arrived with id %q and %q", n.BranchID, n.Event.Sequence, id, n.Event.ID))
		}
		return false
	}
	if sequence, ok := s.eventIDs[n.Event.ID]; ok {
		s.report(protocolErrorf("session.event", "event id %q arrived at sequence %d and %d", n.Event.ID, sequence, n.Event.Sequence))
		return false
	}
	s.eventSeq[key] = n.Event.ID
	s.eventIDs[n.Event.ID] = n.Event.Sequence
	return true
}

// emit buffers during the attach handshake and sends in order afterwards. An
// event that was already buffered keeps later events behind it, so the flush
// order is exactly the arrival order.
func (s *SessionStream) emit(event StreamEvent) bool {
	if !s.attached.Load() || len(s.pending) > 0 {
		s.pending = append(s.pending, event)
		return true
	}
	return s.send(event)
}

func (s *SessionStream) flushPending() bool {
	for len(s.pending) > 0 {
		event := s.pending[0]
		s.pending[0] = StreamEvent{}
		s.pending = s.pending[1:]
		if !s.send(event) {
			return false
		}
	}
	s.pending = nil
	return true
}

func (s *SessionStream) send(event StreamEvent) bool {
	select {
	case s.events <- event:
		return true
	case <-s.stop:
		return false
	case <-s.client.Done():
		return false
	}
}

func (s *SessionStream) report(err error) {
	if err == nil {
		return
	}
	select {
	case s.errs <- err:
	default:
		s.dropped.Add(1)
	}
}

// noticeSessionID extracts the sessionId carried by every §11 notification.
func noticeSessionID(payload any) (string, bool) {
	switch notice := payload.(type) {
	case EventNotice:
		return notice.SessionID, true
	case StateNotice:
		return notice.SessionID, true
	case ApprovalRequest:
		return notice.SessionID, true
	case ApprovalResolved:
		return notice.SessionID, true
	case ApprovalCancelled:
		return notice.SessionID, true
	case SessionNotice:
		return notice.SessionID, true
	case ClosedNotice:
		return notice.SessionID, true
	case ModelProgress:
		return notice.SessionID, true
	default:
		return "", false
	}
}
