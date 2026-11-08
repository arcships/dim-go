package appserver

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

// waitUntil is a bounded, sleep-free wait for an observable condition.
func waitUntil(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(fakeWaitTimeout)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		runtime.Gosched()
	}
}

// blockStreamRouter fills the stream's event channel and primes the router with
// a state notice, so the router is committed to blocking on the next event send
// instead of draining Client.Notifications.
func blockStreamRouter(t *testing.T, server *fakeServer, stream *SessionStream) {
	t.Helper()
	for i := 0; i < cap(stream.events); i++ {
		stream.events <- StreamEvent{Method: "filler"}
	}
	server.emit("session.state", StateNotice{SessionID: stream.SessionID(), BranchID: MainBranchID, Running: true})
	waitUntil(t, "router to block on the full event buffer", func() bool {
		_, ok := stream.BranchState(MainBranchID)
		return ok
	})
}

type closeCounter struct {
	count int
}

func (c *closeCounter) close() { c.count++ }

// TestOpenSessionStreamValidationClosesDedicatedTransport pins the narrowed
// ownership rule: a caller-provided dedicated transport is closed on every
// validation failure, while a shared Client is never closed before a stream
// exists.
func TestOpenSessionStreamValidationClosesDedicatedTransport(t *testing.T) {
	spy := &closeCounter{}
	if _, err := OpenSessionStream(context.Background(), nil, SessionStreamConfig{CloseTransport: spy.close}); err == nil {
		t.Fatal("nil client accepted")
	}
	if spy.count != 1 {
		t.Fatalf("nil-client close count = %d, want 1", spy.count)
	}
	if _, err := OpenSessionStream(context.Background(), nil, SessionStreamConfig{}); err == nil {
		t.Fatal("nil client without a closer accepted")
	}

	_, client := newFakeServer(t, fakeConfig{})
	spy = &closeCounter{}
	if _, err := OpenSessionStream(testContext(t), client, SessionStreamConfig{CloseTransport: spy.close}); !errors.Is(err, ErrProtocol) {
		t.Fatalf("empty sessionId error = %v, want ErrProtocol", err)
	}
	if spy.count != 1 {
		t.Fatalf("empty-session close count = %d, want 1", spy.count)
	}

	doneCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := OpenSessionStream(doneCtx, client, SessionStreamConfig{SessionID: "s1", CloseTransport: spy.close}); err == nil {
		t.Fatal("done context accepted")
	} else {
		assertAppError(t, err, "request_aborted")
	}
	if spy.count != 2 {
		t.Fatalf("done-context close count = %d, want 2", spy.count)
	}

	// With no dedicated closer, validation must not close a shared client.
	if _, err := OpenSessionStream(testContext(t), client, SessionStreamConfig{}); !errors.Is(err, ErrProtocol) {
		t.Fatalf("empty sessionId error = %v, want ErrProtocol", err)
	}
	select {
	case <-client.Done():
		t.Fatal("validation failure closed the shared client")
	default:
	}
}

// stallStreamReader blocks the stream router and then the client reader, so a
// reply to any later request can never be read.
func stallStreamReader(t *testing.T, server *fakeServer, client *Client, stream *SessionStream) {
	t.Helper()
	blockStreamRouter(t, server, stream)
	for i := 0; i < cap(client.notify); i++ {
		client.notify <- Notification{Method: "filler", Params: json.RawMessage(`{}`)}
	}
	server.emit("session.state", StateNotice{SessionID: stream.SessionID(), BranchID: MainBranchID, Running: false})
}

// TestOpenSessionStreamIsTakeOncePerClient pins the one-stream-per-client
// precondition and the transport ownership rule: a rejected second open must
// not close a closer that may be shared with the live stream.
func TestOpenSessionStreamIsTakeOncePerClient(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, fakeConfig{
		Hello:  fakeHello("session.history"),
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(3)}},
	})
	spy := &closeCounter{}
	stream, err := OpenSessionStream(ctx, client, SessionStreamConfig{SessionID: "s1", CloseTransport: spy.close})
	if err != nil {
		t.Fatalf("first OpenSessionStream: %v", err)
	}
	if spy.count != 0 {
		t.Fatalf("successful open closed its transport: %d", spy.count)
	}

	// A rejected second stream sharing the same closer must not tear down the
	// existing stream's connection.
	if _, err := OpenSessionStream(ctx, client, SessionStreamConfig{SessionID: "s1", CloseTransport: spy.close}); !errors.Is(err, ErrStreamAlreadyOpen) {
		t.Fatalf("second OpenSessionStream error = %v, want ErrStreamAlreadyOpen", err)
	}
	if spy.count != 0 {
		t.Fatalf("rejected second stream closed the shared transport: %d", spy.count)
	}

	// Validation failures while a stream is live must leave the shared closer
	// alone too; the transport may belong to the existing stream.
	if _, err := OpenSessionStream(ctx, client, SessionStreamConfig{SessionID: "", CloseTransport: spy.close}); !errors.Is(err, ErrProtocol) {
		t.Fatalf("empty sessionId error = %v, want ErrProtocol", err)
	}
	if spy.count != 0 {
		t.Fatalf("validation failure closed the live stream's transport: %d", spy.count)
	}
	if _, err := OpenSessionStream(ctx, client, SessionStreamConfig{SessionID: "s1"}); !errors.Is(err, ErrStreamAlreadyOpen) {
		t.Fatalf("second OpenSessionStream without a closer error = %v, want ErrStreamAlreadyOpen", err)
	}
	select {
	case <-client.Done():
		t.Fatal("rejected second stream closed the shared client")
	default:
	}

	server.emit("session.state", StateNotice{SessionID: "s1", BranchID: MainBranchID, Running: true})
	if event := nextStreamEvent(t, stream); event.Method != "session.state" {
		t.Fatalf("original stream event = %#v", event)
	}

	// The live stream still owns its transport and closes it on release.
	stream.Release()
	if spy.count != 1 {
		t.Fatalf("Release close count = %d, want 1", spy.count)
	}
}

// TestSessionStreamUndrainedConsumerStallsCalls proves backpressure is bounded:
// with Events undrained, the reader blocks and a request times out instead of
// hanging forever.
func TestSessionStreamUndrainedConsumerStallsCalls(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, fakeConfig{
		Hello:  fakeHello("session.history"),
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(5)}},
	})
	stream, err := OpenSessionStream(ctx, client, SessionStreamConfig{SessionID: "s1"})
	if err != nil {
		t.Fatalf("OpenSessionStream: %v", err)
	}
	stallStreamReader(t, server, client, stream)

	callCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, callErr := client.Call(callCtx, "daemon.ping", nil)
	if elapsed := time.Since(start); elapsed > testPeerTimeout {
		t.Fatalf("Call blocked for %v, want a bounded request_timeout", elapsed)
	}
	assertAppError(t, callErr, "request_timeout")
}

// TestSessionStreamDetachDuringStall proves the delete path still terminates
// when the stream consumer is stalled: the detach request times out, then the
// connection is closed and the stream reaches detached.
func TestSessionStreamDetachDuringStall(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, fakeConfig{
		Hello:  fakeHello("session.history"),
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(5)}},
	})
	stream, err := OpenSessionStream(ctx, client, SessionStreamConfig{SessionID: "s1"})
	if err != nil {
		t.Fatalf("OpenSessionStream: %v", err)
	}
	stallStreamReader(t, server, client, stream)

	detachCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	err = stream.Detach(detachCtx)
	if elapsed := time.Since(start); elapsed > testPeerTimeout {
		t.Fatalf("Detach blocked for %v", elapsed)
	}
	assertAppError(t, err, "request_timeout")
	if state := stream.State(); state != StreamStateDetached {
		t.Fatalf("state = %s, want detached", state)
	}
	waitClientDone(t, client)
}

// TestSessionStreamReleaseDuringStall proves the ordinary holder release also
// finishes under the 500 ms bound and closes the connection.
func TestSessionStreamReleaseDuringStall(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, fakeConfig{
		Hello:  fakeHello("session.history"),
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(5)}},
	})
	stream, err := OpenSessionStream(ctx, client, SessionStreamConfig{SessionID: "s1"})
	if err != nil {
		t.Fatalf("OpenSessionStream: %v", err)
	}
	stallStreamReader(t, server, client, stream)

	start := time.Now()
	stream.Release()
	if elapsed := time.Since(start); elapsed > testPeerTimeout {
		t.Fatalf("Release blocked for %v", elapsed)
	}
	if state := stream.State(); state != StreamStateDetached {
		t.Fatalf("state = %s, want detached", state)
	}
	waitClientDone(t, client)
}

// TestSessionStreamErrorAccountingIncludesClientDrops overflows both error
// stages and shows why DroppedErrors alone cannot account for total loss.
func TestSessionStreamErrorAccountingIncludesClientDrops(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, fakeConfig{
		Hello:  fakeHello("session.history"),
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(5)}},
	})
	stream, err := OpenSessionStream(ctx, client, SessionStreamConfig{SessionID: "s1"})
	if err != nil {
		t.Fatalf("OpenSessionStream: %v", err)
	}

	// Stage 1: reports reach the router faster than the stream's 8-slot Errors
	// buffer can hold them.
	for i := 0; i < 16; i++ {
		client.errs <- errors.New("router flood")
		waitUntil(t, "router to take a client error", func() bool { return len(client.errs) == 0 })
	}
	waitUntil(t, "stream error buffer to fill", func() bool { return len(stream.Errors()) == streamErrorBuffer })
	waitUntil(t, "stream drops to reach 8", func() bool { return stream.DroppedErrors() == 8 })
	if got := len(stream.Errors()); got != 8 {
		t.Fatalf("stream buffered errors = %d, want 8", got)
	}

	// Stage 2: block the router so Client.Errors overflows; those drops never
	// reach the stream's own counter.
	blockStreamRouter(t, server, stream)
	for i := 0; i < 16; i++ {
		client.report(errors.New("client flood"))
	}
	if got := client.DroppedErrors(); got != 8 {
		t.Fatalf("client dropped errors = %d, want 8", got)
	}
	if got := stream.ClientDroppedErrors(); got != 8 {
		t.Fatalf("ClientDroppedErrors = %d, want 8", got)
	}
	if got := stream.DroppedErrors(); got != 8 {
		t.Fatalf("stream DroppedErrors = %d, want the stage-1 value 8", got)
	}
	// Delivered 8 (stream) + 8 (still buffered in Client.Errors) + 8 stream
	// dropped + 8 client dropped == 32 reports attempted.
	delivered := len(stream.Errors())
	buffered := len(client.Errors())
	if total := delivered + buffered + int(stream.DroppedErrors()) + int(stream.ClientDroppedErrors()); total != 32 {
		t.Fatalf("accounted reports = %d, want 32", total)
	}
}

func TestSessionStreamPreAttachBufferCap(t *testing.T) {
	stream := &SessionStream{
		events: make(chan StreamEvent, 1),
		errs:   make(chan error, streamErrorBuffer),
	}
	for i := 0; i < maxPreAttachPending; i++ {
		if !stream.emit(StreamEvent{Method: "filler"}) {
			t.Fatalf("emit %d stopped the stream", i)
		}
	}
	if got := len(stream.pending); got != maxPreAttachPending {
		t.Fatalf("pending = %d, want %d", got, maxPreAttachPending)
	}
	if !stream.emit(StreamEvent{Method: "session.event"}) {
		t.Fatal("overflow must be reported, not stop the stream")
	}
	if got := len(stream.pending); got != maxPreAttachPending {
		t.Fatalf("pending after overflow = %d, want the cap %d", got, maxPreAttachPending)
	}
	if got := stream.PreAttachDropped(); got != 1 {
		t.Fatalf("PreAttachDropped = %d, want 1", got)
	}
	select {
	case err := <-stream.errs:
		if !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "pre-attach buffer") {
			t.Fatalf("overflow error = %v, want a pre-attach protocol error", err)
		}
	default:
		t.Fatal("pre-attach overflow did not report a protocol error")
	}
	stream.emit(StreamEvent{Method: "session.state"})
	if got := stream.PreAttachDropped(); got != 2 {
		t.Fatalf("PreAttachDropped = %d after second overflow, want 2", got)
	}
}
