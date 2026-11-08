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

// TestOpenSessionStreamIsTakeOncePerClient pins the one-stream-per-client
// precondition and proves a rejected second open leaves the first stream intact.
func TestOpenSessionStreamIsTakeOncePerClient(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, fakeConfig{
		Hello:  fakeHello("session.history"),
		Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(3)}},
	})
	stream, err := OpenSessionStream(ctx, client, SessionStreamConfig{SessionID: "s1"})
	if err != nil {
		t.Fatalf("first OpenSessionStream: %v", err)
	}

	spy := &closeCounter{}
	if _, err := OpenSessionStream(ctx, client, SessionStreamConfig{SessionID: "s1", CloseTransport: spy.close}); !errors.Is(err, ErrStreamAlreadyOpen) {
		t.Fatalf("second OpenSessionStream error = %v, want ErrStreamAlreadyOpen", err)
	}
	if spy.count != 1 {
		t.Fatalf("rejected stream close count = %d, want 1", spy.count)
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
	blockStreamRouter(t, server, stream)

	// The router is blocked, so filling the client handoff makes the reader
	// stop after one more notification; the reply to any request sits unread.
	for i := 0; i < cap(client.notify); i++ {
		client.notify <- Notification{Method: "filler", Params: json.RawMessage(`{}`)}
	}
	server.emit("session.state", StateNotice{SessionID: "s1", BranchID: MainBranchID, Running: false})

	callCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, callErr := client.Call(callCtx, "daemon.ping", nil)
	if elapsed := time.Since(start); elapsed > testPeerTimeout {
		t.Fatalf("Call blocked for %v, want a bounded request_timeout", elapsed)
	}
	assertAppError(t, callErr, "request_timeout")
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
	select {
	case err := <-stream.errs:
		if !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "pre-attach buffer") {
			t.Fatalf("overflow error = %v, want a pre-attach protocol error", err)
		}
	default:
		t.Fatal("pre-attach overflow did not report a protocol error")
	}
}
