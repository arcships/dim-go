package appserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func streamEventFixture(sequence int64) RawEvent {
	return RawEvent{
		ID:       fmt.Sprintf("event-%d", sequence),
		Sequence: sequence,
		CreateAt: fakeCreatedAt,
		Type:     "message",
		Name:     "user_input",
		Data:     json.RawMessage(`{"turn_id":"turn-1"}`),
	}
}

func nextStreamEvent(t *testing.T, stream *SessionStream) StreamEvent {
	t.Helper()
	select {
	case event, ok := <-stream.Events():
		if !ok {
			t.Fatal("stream events closed unexpectedly")
		}
		return event
	case <-time.After(fakeWaitTimeout):
		t.Fatal("timed out waiting for a stream event")
		return StreamEvent{}
	}
}

func nextStreamError(t *testing.T, stream *SessionStream) error {
	t.Helper()
	select {
	case err, ok := <-stream.Errors():
		if !ok {
			t.Fatal("stream errors closed unexpectedly")
		}
		return err
	case <-time.After(fakeWaitTimeout):
		t.Fatal("timed out waiting for a stream error")
		return nil
	}
}

func drainStreamEvents(t *testing.T, stream *SessionStream) []StreamEvent {
	t.Helper()
	var events []StreamEvent
	timeout := time.After(fakeWaitTimeout)
	for {
		select {
		case event, ok := <-stream.Events():
			if !ok {
				return events
			}
			events = append(events, event)
		case <-timeout:
			t.Fatal("stream did not reach a terminal state")
			return events
		}
	}
}

func waitClientDone(t *testing.T, client *Client) {
	t.Helper()
	select {
	case <-client.Done():
	case <-time.After(fakeWaitTimeout):
		t.Fatal("client connection was not closed")
	}
}

func openStream(t *testing.T, cfg fakeConfig, features ...string) (*fakeServer, *Client, *SessionStream) {
	t.Helper()
	server, client := newFakeServer(t, cfg)
	stream, err := OpenSessionStream(testContext(t), client, SessionStreamConfig{
		SessionID: "s1",
		Features:  features,
	})
	if err != nil {
		t.Fatalf("OpenSessionStream: %v", err)
	}
	return server, client, stream
}

// TestOpenSessionStreamSubscribesBeforeAttach checks §8 step 2-3: the
// notifications the daemon emits before the attach reply are received, buffered
// and replayed in arrival order once the head is known.
func TestOpenSessionStreamSubscribesBeforeAttach(t *testing.T) {
	event := streamEventFixture(1)
	cfg := fakeConfig{
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(41), Running: boolPtr(true)}},
		OnAttach: func(server *fakeServer) {
			server.emit("approval.request", ApprovalRequest{
				SessionID: "s1",
				BranchID:  "main",
				RequestID: "request-1",
				Source:    ApprovalSource{Kind: "tool"},
				Call:      ApprovalCall{Name: "shell"},
				Reason:    "replay",
			})
			server.emit("session.event", EventNotice{SessionID: "s1", BranchID: "main", Event: event})
			server.emit("session.state", StateNotice{SessionID: "s1", BranchID: "main", Running: true})
		},
	}
	server, _, stream := openStream(t, cfg, "session.history")

	if stream.Head() != 41 {
		t.Fatalf("head = %d, want 41", stream.Head())
	}
	attach := stream.Attach()
	if attach.Running == nil || !*attach.Running {
		t.Fatalf("attach.running = %v, want true", attach.Running)
	}
	if stream.State() != StreamStateAttached {
		t.Fatalf("state = %s, want attached", stream.State())
	}
	if methods := server.recordedMethods(); len(methods) != 1 || methods[0] != "session.attach" {
		t.Fatalf("methods = %v, want only session.attach", methods)
	}

	first := nextStreamEvent(t, stream)
	if first.Method != "approval.request" || !first.Known {
		t.Fatalf("first event = %#v", first)
	}
	if _, ok := first.Payload.(ApprovalRequest); !ok {
		t.Fatalf("first payload = %T, want ApprovalRequest", first.Payload)
	}
	second := nextStreamEvent(t, stream)
	if second.Method != "session.event" || !second.Known {
		t.Fatalf("second event = %#v", second)
	}
	notice, ok := second.Payload.(EventNotice)
	if !ok || notice.Event.ID != event.ID || notice.Event.Sequence != 1 {
		t.Fatalf("second payload = %#v", second.Payload)
	}
	third := nextStreamEvent(t, stream)
	if third.Method != "session.state" {
		t.Fatalf("third event = %#v", third)
	}
	state, ok := stream.BranchState("main")
	if !ok || !state.Running {
		t.Fatalf("branch state = %#v, %v", state, ok)
	}
}

// TestOpenSessionStreamHeadFallback checks §8 step 4 for a daemon that omits
// attach.sequence.
func TestOpenSessionStreamHeadFallback(t *testing.T) {
	cfg := fakeConfig{
		Attach: map[string]AttachResult{"s1": {}},
		HistoryRaw: []json.RawMessage{rawJSON(t, map[string]any{
			"events":    []any{},
			"through":   int64(7),
			"nextAfter": int64(7),
			"done":      true,
		})},
	}
	server, _, stream := openStream(t, cfg, "session.history")

	if stream.Head() != 7 {
		t.Fatalf("head = %d, want fallback through 7", stream.Head())
	}
	methods := server.recordedMethods()
	if len(methods) != 2 || methods[0] != "session.attach" || methods[1] != "session.history" {
		t.Fatalf("methods = %v, want attach then history", methods)
	}
	calls := server.recordedCalls("session.history")
	var params HistoryParams
	if err := json.Unmarshal(calls[0].Params, &params); err != nil {
		t.Fatalf("history params: %v", err)
	}
	if params.SessionID != "s1" || params.BranchID != MainBranchID {
		t.Fatalf("history params = %#v", params)
	}
	if params.After == nil || *params.After != 0 || params.Limit == nil || *params.Limit != 1 {
		t.Fatalf("history window = after %v, limit %v", params.After, params.Limit)
	}
	if params.Through != nil || params.LatestTurn != nil {
		t.Fatalf("history params invented bounds: %#v", params)
	}
}

func TestOpenSessionStreamHeadFallbackRequiresFeature(t *testing.T) {
	server, client := newFakeServer(t, fakeConfig{
		Attach: map[string]AttachResult{"s1": {}},
	})
	_, err := OpenSessionStream(testContext(t), client, SessionStreamConfig{SessionID: "s1"})
	var capability *CapabilityError
	if !errors.As(err, &capability) || capability.Feature != "session.history" {
		t.Fatalf("error = %v, want CapabilityError session.history", err)
	}
	if calls := server.recordedCalls("session.history"); len(calls) != 0 {
		t.Fatalf("history was called without the feature: %d calls", len(calls))
	}
	waitClientDone(t, client)
}

func TestOpenSessionStreamHeadFallbackFailure(t *testing.T) {
	_, client := newFakeServer(t, fakeConfig{
		Attach: map[string]AttachResult{"s1": {}},
	})
	_, err := OpenSessionStream(testContext(t), client, SessionStreamConfig{
		SessionID: "s1",
		Features:  []string{"session.history"},
	})
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) || rpcErr.Code != -32602 {
		t.Fatalf("error = %v, want the history -32602 failure", err)
	}
	waitClientDone(t, client)
}

func TestOpenSessionStreamAttachFailureClosesConnection(t *testing.T) {
	_, client := newFakeServer(t, fakeConfig{})
	_, err := OpenSessionStream(testContext(t), client, SessionStreamConfig{SessionID: "missing"})
	if err == nil {
		t.Fatal("OpenSessionStream succeeded for an unknown session")
	}
	waitClientDone(t, client)
}

func TestOpenSessionStreamRequiresSessionID(t *testing.T) {
	_, client := newFakeServer(t, fakeConfig{})
	_, err := OpenSessionStream(testContext(t), client, SessionStreamConfig{})
	if !errors.Is(err, ErrProtocol) {
		t.Fatalf("error = %v, want ErrProtocol", err)
	}
}

// TestSessionStreamRoutesAndFilters checks §11 routing: known notices for this
// session are decoded, unknown methods pass through verbatim, and notices for
// another session or malformed params are reported instead of routed.
func TestSessionStreamRoutesAndFilters(t *testing.T) {
	server, _, stream := openStream(t, fakeConfig{
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(3)}},
	})

	other := streamEventFixture(2)
	server.emit("session.event", EventNotice{SessionID: "s2", BranchID: "main", Event: other})
	server.emit("session.state", map[string]any{"branchId": "main", "running": true})
	server.emit("session.future", map[string]any{"sessionId": "s1", "payload": map[string]any{"x": 1}})

	firstErr := nextStreamError(t, stream)
	secondErr := nextStreamError(t, stream)
	if !errors.Is(firstErr, ErrProtocol) || !errors.Is(secondErr, ErrProtocol) {
		t.Fatalf("errors = %v, %v; want protocol errors", firstErr, secondErr)
	}

	event := nextStreamEvent(t, stream)
	if event.Method != "session.future" || event.Known {
		t.Fatalf("event = %#v, want the unknown method passed through", event)
	}
	var params map[string]any
	if err := json.Unmarshal(event.Params, &params); err != nil {
		t.Fatalf("unknown params: %v", err)
	}
	if params["sessionId"] != "s1" {
		t.Fatalf("unknown params = %#v", params)
	}
	if _, ok := params["payload"].(map[string]any); !ok {
		t.Fatalf("unknown params lost payload: %#v", params)
	}
}

// TestSessionStreamEventIdempotencyAndSequenceConflicts covers §8 step 7: the
// pending buffer dedupes by event.id, and one branch sequence may not map to two
// ids. Conflicts are reported and not routed; the stream stays usable.
func TestSessionStreamEventIdempotencyAndSequenceConflicts(t *testing.T) {
	first := streamEventFixture(1)
	cfg := fakeConfig{
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(1)}},
		OnAttach: func(server *fakeServer) {
			server.emit("session.event", EventNotice{SessionID: "s1", BranchID: "main", Event: first})
			server.emit("session.event", EventNotice{SessionID: "s1", BranchID: "main", Event: first})
		},
	}
	server, _, stream := openStream(t, cfg)

	event := nextStreamEvent(t, stream)
	notice, ok := event.Payload.(EventNotice)
	if !ok || notice.Event.ID != first.ID {
		t.Fatalf("event = %#v, want the first event once", event)
	}

	conflict := streamEventFixture(1)
	conflict.ID = "event-conflict"
	server.emit("session.event", EventNotice{SessionID: "s1", BranchID: "main", Event: conflict})
	// Same id at a different sequence is also a protocol conflict.
	moved := first
	moved.Sequence = 9
	server.emit("session.event", EventNotice{SessionID: "s1", BranchID: "main", Event: moved})
	// A duplicate of the already accepted event is dropped silently.
	server.emit("session.event", EventNotice{SessionID: "s1", BranchID: "main", Event: first})
	next := streamEventFixture(2)
	server.emit("session.event", EventNotice{SessionID: "s1", BranchID: "main", Event: next})

	if err := nextStreamError(t, stream); !errors.Is(err, ErrProtocol) {
		t.Fatalf("sequence conflict error = %v, want ErrProtocol", err)
	}
	if err := nextStreamError(t, stream); !errors.Is(err, ErrProtocol) {
		t.Fatalf("id conflict error = %v, want ErrProtocol", err)
	}
	event = nextStreamEvent(t, stream)
	notice, ok = event.Payload.(EventNotice)
	if !ok || notice.Event.ID != next.ID || notice.Event.Sequence != next.Sequence {
		t.Fatalf("event after conflicts = %#v, want %s", event, next.ID)
	}
}

func TestSessionStreamClosedIsTerminal(t *testing.T) {
	server, client, stream := openStream(t, fakeConfig{
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(5)}},
	})
	server.emit("session.closed", ClosedNotice{SessionID: "s1", Reason: "done"})

	events := drainStreamEvents(t, stream)
	if len(events) != 1 {
		t.Fatalf("events = %#v, want only session.closed", events)
	}
	if events[0].Method != "session.closed" || !events[0].Known {
		t.Fatalf("closed event = %#v", events[0])
	}
	if stream.State() != StreamStateClosed {
		t.Fatalf("state = %s, want closed", stream.State())
	}
	if stream.Reason() != "done" {
		t.Fatalf("reason = %q, want done", stream.Reason())
	}
	waitClientDone(t, client)
}

func TestSessionStreamTransportLossNeedsResync(t *testing.T) {
	server, client, stream := openStream(t, fakeConfig{
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(5)}},
	})
	server.closeConn()

	if events := drainStreamEvents(t, stream); len(events) != 0 {
		t.Fatalf("events = %#v, want none before the loss", events)
	}
	if stream.State() != StreamStateLost {
		t.Fatalf("state = %s, want lost", stream.State())
	}
	if stream.Reason() != StreamReasonTransportClosed {
		t.Fatalf("reason = %q, want %q", stream.Reason(), StreamReasonTransportClosed)
	}
	waitClientDone(t, client)
}

func TestOpenSessionStreamClosedDuringAttach(t *testing.T) {
	cfg := fakeConfig{
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(9)}},
		OnAttach: func(server *fakeServer) {
			server.emit("session.closed", ClosedNotice{SessionID: "s1", Reason: "done"})
		},
	}
	server, client := newFakeServer(t, cfg)
	_, err := OpenSessionStream(testContext(t), client, SessionStreamConfig{
		SessionID: "s1",
		Features:  []string{"session.history"},
	})
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("error = %v, want ErrSessionClosed", err)
	}
	var closed *SessionClosedError
	if !errors.As(err, &closed) || closed.Reason != "done" {
		t.Fatalf("error = %#v, want reason done", err)
	}
	if calls := server.recordedCalls("session.history"); len(calls) != 0 {
		t.Fatalf("history was called after the terminal notice: %d calls", len(calls))
	}
	waitClientDone(t, client)
}

func TestSessionStreamDetachLifecycle(t *testing.T) {
	attachConfig := func() fakeConfig {
		return fakeConfig{Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(5)}}}
	}

	t.Run("detach waits for the reply and closes", func(t *testing.T) {
		server, client, stream := openStream(t, attachConfig())
		if err := stream.Detach(testContext(t)); err != nil {
			t.Fatalf("Detach: %v", err)
		}
		if stream.State() != StreamStateDetached {
			t.Fatalf("state = %s, want detached", stream.State())
		}
		calls := server.recordedCalls("session.detach")
		if len(calls) != 1 {
			t.Fatalf("detach calls = %d, want 1", len(calls))
		}
		var params DetachParams
		if err := json.Unmarshal(calls[0].Params, &params); err != nil || params.SessionID != "s1" {
			t.Fatalf("detach params = %s (%v)", calls[0].Params, err)
		}
		waitClientDone(t, client)
	})

	t.Run("release sends detach and closes", func(t *testing.T) {
		server, client, stream := openStream(t, attachConfig())
		stream.Release()
		if stream.State() != StreamStateDetached {
			t.Fatalf("state = %s, want detached", stream.State())
		}
		if calls := server.recordedCalls("session.detach"); len(calls) != 1 {
			t.Fatalf("detach calls = %d, want 1", len(calls))
		}
		waitClientDone(t, client)
	})

	t.Run("detach after closed is a no-op", func(t *testing.T) {
		server, _, stream := openStream(t, attachConfig())
		server.emit("session.closed", ClosedNotice{SessionID: "s1", Reason: "done"})
		drainStreamEvents(t, stream)
		if err := stream.Detach(testContext(t)); err != nil {
			t.Fatalf("Detach after closed: %v", err)
		}
		if stream.State() != StreamStateClosed {
			t.Fatalf("state = %s, want closed", stream.State())
		}
		if calls := server.recordedCalls("session.detach"); len(calls) != 0 {
			t.Fatalf("detach calls after closed = %d, want 0", len(calls))
		}
	})

	t.Run("detach reply failure still closes", func(t *testing.T) {
		cfg := attachConfig()
		cfg.DetachErr = fakeUnknownMethod()
		_, client, stream := openStream(t, cfg)
		err := stream.Detach(testContext(t))
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32601 {
			t.Fatalf("Detach error = %v, want the daemon failure", err)
		}
		if stream.State() != StreamStateDetached {
			t.Fatalf("state = %s, want detached", stream.State())
		}
		waitClientDone(t, client)
	})
}

func TestDetachParamsValidate(t *testing.T) {
	if err := (DetachParams{SessionID: "s1"}).Validate(); err != nil {
		t.Fatalf("valid params: %v", err)
	}
	if err := (DetachParams{}).Validate(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("empty sessionId = %v, want ErrProtocol", err)
	}
}

func TestStreamStateTerminal(t *testing.T) {
	for _, tc := range []struct {
		state StreamState
		want  bool
	}{
		{StreamStateAttaching, false},
		{StreamStateAttached, false},
		{StreamStateDetached, true},
		{StreamStateClosed, true},
		{StreamStateLost, true},
	} {
		if got := tc.state.Terminal(); got != tc.want {
			t.Errorf("%s.Terminal() = %v, want %v", tc.state, got, tc.want)
		}
	}
}
