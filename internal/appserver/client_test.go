package appserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

const testPeerTimeout = 2 * time.Second

// testPeer drives the daemon side of an in-memory connection.
type testPeer struct {
	t  *testing.T
	rw net.Conn
	fr *FrameReader
}

// peerRequest is one request as the fake daemon sees it.
type peerRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func newTestClient(t *testing.T) (*Client, *testPeer) {
	t.Helper()
	a, b := net.Pipe()
	t.Cleanup(func() { _ = b.Close() })
	c := NewClient(a)
	t.Cleanup(c.Close)
	return c, &testPeer{t: t, rw: b, fr: NewFrameReader(b)}
}

func (p *testPeer) readRequest() peerRequest {
	p.t.Helper()
	if err := p.rw.SetReadDeadline(time.Now().Add(testPeerTimeout)); err != nil {
		p.t.Fatalf("set read deadline: %v", err)
	}
	defer func() { _ = p.rw.SetReadDeadline(time.Time{}) }()
	raw, err := p.fr.ReadFrame()
	if err != nil {
		p.t.Fatalf("read request: %v", err)
	}
	var req peerRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		p.t.Fatalf("decode request %s: %v", raw, err)
	}
	return req
}

func (p *testPeer) write(line string) {
	p.t.Helper()
	if err := p.rw.SetWriteDeadline(time.Now().Add(testPeerTimeout)); err != nil {
		p.t.Fatalf("set write deadline: %v", err)
	}
	defer func() { _ = p.rw.SetWriteDeadline(time.Time{}) }()
	if _, err := io.WriteString(p.rw, line+"\n"); err != nil {
		p.t.Fatalf("write %s: %v", line, err)
	}
}

func (p *testPeer) replyResult(id json.RawMessage, result string) {
	p.t.Helper()
	p.write(`{"jsonrpc":"2.0","id":` + string(id) + `,"result":` + result + `}`)
}

type callOutcome struct {
	raw json.RawMessage
	err error
}

// callAsync starts Call in a goroutine so the test can drive the peer side.
func callAsync(ctx context.Context, c *Client, method string, params any) <-chan callOutcome {
	out := make(chan callOutcome, 1)
	go func() {
		raw, err := c.Call(ctx, method, params)
		out <- callOutcome{raw: raw, err: err}
	}()
	return out
}

func awaitOutcome(t *testing.T, out <-chan callOutcome) callOutcome {
	t.Helper()
	select {
	case o := <-out:
		return o
	case <-time.After(testPeerTimeout):
		t.Fatal("call did not finish")
		return callOutcome{}
	}
}

func assertAppError(t *testing.T, err error, want string) {
	t.Helper()
	var le *LocalError
	if !errors.As(err, &le) {
		t.Fatalf("error = %v, want LocalError %q", err, want)
	}
	if le.AppError != want {
		t.Fatalf("AppError = %q, want %q (err = %v)", le.AppError, want, err)
	}
}

func TestResponseKey(t *testing.T) {
	cases := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{raw: `1`, want: "1"},
		{raw: ` 1 `, want: "1"},
		{raw: `42`, want: "42"},
		{raw: `"1"`, want: "1"},
		{raw: `"42"`, want: "42"},
		{raw: `"007"`, want: "s:007"},
		{raw: `"abc"`, want: "s:abc"},
		{raw: `""`, want: "s:"},
		{raw: `null`, wantErr: true},
		{raw: ``, wantErr: true},
		{raw: `true`, wantErr: true},
		{raw: `[1]`, wantErr: true},
		{raw: `{"id":1}`, wantErr: true},
	}
	for _, tc := range cases {
		got, err := responseKey(json.RawMessage(tc.raw))
		if tc.wantErr {
			if err == nil {
				t.Errorf("responseKey(%s) = %q, want error", tc.raw, got)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("responseKey(%s) = %q, %v; want %q", tc.raw, got, err, tc.want)
		}
	}
}

type methodOutcome struct {
	method string
	raw    json.RawMessage
	err    error
}

func TestClientCorrelatesOutOfOrderResponses(t *testing.T) {
	c, peer := newTestClient(t)
	methods := []string{"one", "two", "three"}
	outcomes := make(chan methodOutcome, len(methods))
	for _, method := range methods {
		method := method
		go func() {
			raw, err := c.Call(context.Background(), method, nil)
			outcomes <- methodOutcome{method: method, raw: raw, err: err}
		}()
	}
	idByMethod := make(map[string]json.RawMessage, len(methods))
	for range methods {
		req := peer.readRequest()
		idByMethod[req.Method] = req.ID
	}
	// Reply in reverse order: correlation must follow the id, not the
	// request order.
	for i := len(methods) - 1; i >= 0; i-- {
		peer.replyResult(idByMethod[methods[i]], fmt.Sprintf("%q", methods[i]))
	}
	seen := make(map[string]bool, len(methods))
	for range methods {
		select {
		case o := <-outcomes:
			if o.err != nil {
				t.Fatalf("Call(%s) failed: %v", o.method, o.err)
			}
			var s string
			if err := json.Unmarshal(o.raw, &s); err != nil {
				t.Fatalf("Call(%s) result %s: %v", o.method, o.raw, err)
			}
			if s != o.method {
				t.Fatalf("Call(%s) received result %q", o.method, s)
			}
			seen[o.method] = true
		case <-time.After(testPeerTimeout):
			t.Fatal("response timeout")
		}
	}
	if len(seen) != len(methods) {
		t.Fatalf("missing results: seen %v", seen)
	}
}

func TestClientCorrelatesDigitStringID(t *testing.T) {
	c, peer := newTestClient(t)
	out := callAsync(context.Background(), c, "session.catalog", map[string]any{"limit": 200})
	req := peer.readRequest()
	if string(req.ID) != "1" {
		t.Fatalf("request id = %s, want 1", req.ID)
	}
	// The daemon may echo the id as a digit string.
	peer.replyResult(json.RawMessage(`"1"`), `{"sessions":[]}`)
	o := awaitOutcome(t, out)
	if o.err != nil {
		t.Fatalf("call failed: %v", o.err)
	}
	if string(o.raw) != `{"sessions":[]}` {
		t.Fatalf("result = %s", o.raw)
	}
}

func TestClientIDsIncreasePerRequest(t *testing.T) {
	c, peer := newTestClient(t)
	for i := 1; i <= 2; i++ {
		out := callAsync(context.Background(), c, "daemon.ping", nil)
		req := peer.readRequest()
		if string(req.ID) != fmt.Sprint(i) {
			t.Fatalf("request %d id = %s", i, req.ID)
		}
		peer.replyResult(req.ID, `{}`)
		if o := awaitOutcome(t, out); o.err != nil {
			t.Fatalf("call %d failed: %v", i, o.err)
		}
	}
}

func TestClientOmitsNilParams(t *testing.T) {
	c, peer := newTestClient(t)
	out := callAsync(context.Background(), c, "daemon.ping", nil)
	req := peer.readRequest()
	if req.Params != nil {
		t.Fatalf("nil params sent as %s", req.Params)
	}
	peer.replyResult(req.ID, `{}`)
	if o := awaitOutcome(t, out); o.err != nil {
		t.Fatalf("call failed: %v", o.err)
	}

	out = callAsync(context.Background(), c, "session.catalog", map[string]any{"limit": 200})
	req = peer.readRequest()
	if string(req.Params) != `{"limit":200}` {
		t.Fatalf("params = %s", req.Params)
	}
	peer.replyResult(req.ID, `{}`)
	if o := awaitOutcome(t, out); o.err != nil {
		t.Fatalf("call failed: %v", o.err)
	}
}

func TestClientPassesUnknownNotificationVerbatim(t *testing.T) {
	c, peer := newTestClient(t)
	params := `{"nested":{"list":[1,2,{"x":null}],"text":"\u2713"},"extra":true}`
	peer.write(`{"jsonrpc":"2.0","method":"future.unknown","params":` + params + `}`)
	select {
	case n := <-c.Notifications():
		if n.Method != "future.unknown" {
			t.Fatalf("method = %q", n.Method)
		}
		if string(n.Params) != params {
			t.Fatalf("params = %s, want %s", n.Params, params)
		}
	case <-time.After(testPeerTimeout):
		t.Fatal("notification timeout")
	}

	peer.write(`{"jsonrpc":"2.0","method":"session.closed","params":{"sessionId":"s1","reason":"done"}}`)
	select {
	case n := <-c.Notifications():
		if n.Method != "session.closed" || string(n.Params) != `{"sessionId":"s1","reason":"done"}` {
			t.Fatalf("notification = %#v", n)
		}
	case <-time.After(testPeerTimeout):
		t.Fatal("notification timeout")
	}

	peer.write(`{"jsonrpc":"2.0","method":"session.notice"}`)
	select {
	case n := <-c.Notifications():
		if n.Method != "session.notice" || len(n.Params) != 0 {
			t.Fatalf("notification = %#v", n)
		}
	case <-time.After(testPeerTimeout):
		t.Fatal("notification timeout")
	}
}

func TestClientRejectsNotificationCarryingID(t *testing.T) {
	c, peer := newTestClient(t)
	peer.write(`{"jsonrpc":"2.0","id":9,"method":"session.event","params":{"sessionId":"s1"}}`)
	select {
	case err := <-c.Errors():
		if err == nil {
			t.Fatal("nil error")
		}
	case <-time.After(testPeerTimeout):
		t.Fatal("error timeout")
	}
	select {
	case n := <-c.Notifications():
		t.Fatalf("peer request delivered as notification: %#v", n)
	default:
	}

	// The connection must stay usable after the protocol violation.
	out := callAsync(context.Background(), c, "daemon.ping", nil)
	req := peer.readRequest()
	peer.replyResult(req.ID, `{"sessions":1}`)
	if o := awaitOutcome(t, out); o.err != nil {
		t.Fatalf("call after violation failed: %v", o.err)
	}
}

func TestClientReportsMalformedFramesAndStaysUsable(t *testing.T) {
	c, peer := newTestClient(t)
	out := callAsync(context.Background(), c, "session.catalog", nil)
	req := peer.readRequest()
	peer.write(`this is not json`)
	peer.write(`{"jsonrpc":"1.0","method":"x"}`)
	peer.write(`{"jsonrpc":"2.0","id":1,"result":null,"error":{"code":1,"message":"x"}}`)
	peer.replyResult(req.ID, `{"sessions":[]}`)

	for i := 0; i < 3; i++ {
		select {
		case err := <-c.Errors():
			if err == nil {
				t.Fatal("nil error")
			}
		case <-time.After(testPeerTimeout):
			t.Fatalf("missing error %d", i+1)
		}
	}
	o := awaitOutcome(t, out)
	if o.err != nil {
		t.Fatalf("in-flight call failed: %v", o.err)
	}
	if string(o.raw) != `{"sessions":[]}` {
		t.Fatalf("result = %s", o.raw)
	}
}

func TestClientReportsUncorrelatedNullIDError(t *testing.T) {
	c, peer := newTestClient(t)
	peer.write(`{"jsonrpc":"2.0","id":null,"error":{"code":-32602,"message":"bad params","data":{"appError":"unknown_method"}}}`)
	var rpcErr *RPCError
	select {
	case err := <-c.Errors():
		if !errors.As(err, &rpcErr) {
			t.Fatalf("error = %v, want wrapped *RPCError", err)
		}
	case <-time.After(testPeerTimeout):
		t.Fatal("error timeout")
	}
	if rpcErr.Code != -32602 || rpcErr.Message != "bad params" {
		t.Fatalf("RPCError = %#v", rpcErr)
	}
	if string(rpcErr.Data) != `{"appError":"unknown_method"}` {
		t.Fatalf("error data = %s", rpcErr.Data)
	}

	out := callAsync(context.Background(), c, "daemon.ping", nil)
	req := peer.readRequest()
	peer.replyResult(req.ID, `{}`)
	if o := awaitOutcome(t, out); o.err != nil {
		t.Fatalf("call after null-id error failed: %v", o.err)
	}
}

func TestClientReturnsServerErrorForItsRequest(t *testing.T) {
	c, peer := newTestClient(t)
	failing := callAsync(context.Background(), c, "sessions.delete", nil)
	healthy := callAsync(context.Background(), c, "daemon.ping", nil)
	idByMethod := make(map[string]json.RawMessage, 2)
	for range 2 {
		req := peer.readRequest()
		idByMethod[req.Method] = req.ID
	}
	// Answer the healthy request first, out of order.
	peer.replyResult(idByMethod["daemon.ping"], `{"ok":true}`)
	peer.write(`{"jsonrpc":"2.0","id":` + string(idByMethod["sessions.delete"]) + `,"error":{"code":-32602,"message":"invalid params","data":{"appError":"unknown_method"}}}`)

	if o := awaitOutcome(t, healthy); o.err != nil {
		t.Fatalf("healthy call failed: %v", o.err)
	}
	o := awaitOutcome(t, failing)
	var rpcErr *RPCError
	if !errors.As(o.err, &rpcErr) {
		t.Fatalf("error = %v, want *RPCError", o.err)
	}
	if rpcErr.Code != -32602 || rpcErr.Message != "invalid params" || string(rpcErr.Data) != `{"appError":"unknown_method"}` {
		t.Fatalf("RPCError = %#v", rpcErr)
	}
}

func TestClientTimesOutWhenPeerDoesNotRead(t *testing.T) {
	// The peer never reads, so the request cannot even be written. The
	// request deadline must still bound the write.
	c, _ := newTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.Call(ctx, "slow", nil)
	if elapsed := time.Since(start); elapsed > testPeerTimeout {
		t.Fatalf("Call blocked for %v", elapsed)
	}
	assertAppError(t, err, "request_timeout")
}

func TestClientTimesOutWhenPeerDoesNotReplyAndIgnoresLateReply(t *testing.T) {
	c, peer := newTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	out := callAsync(ctx, c, "session.attach", map[string]string{"sessionId": "s1"})
	stale := peer.readRequest()
	o := awaitOutcome(t, out)
	assertAppError(t, o.err, "request_timeout")

	// A late reply for the timed-out request must not disturb later calls.
	peer.replyResult(stale.ID, `{"stale":true}`)
	next := callAsync(context.Background(), c, "session.history", nil)
	req := peer.readRequest()
	if req.Method != "session.history" || string(req.ID) == string(stale.ID) {
		t.Fatalf("next request = %#v", req)
	}
	peer.replyResult(req.ID, `{"events":[]}`)
	o = awaitOutcome(t, next)
	if o.err != nil {
		t.Fatalf("call after late reply failed: %v", o.err)
	}
	if string(o.raw) != `{"events":[]}` {
		t.Fatalf("result = %s", o.raw)
	}
}

func TestClientIgnoresRepliesForUnknownIDs(t *testing.T) {
	c, peer := newTestClient(t)
	out := callAsync(context.Background(), c, "session.attach", nil)
	req := peer.readRequest()
	// Neither a never-issued number nor a digit string that is not the issued
	// id may settle the in-flight request.
	peer.replyResult(json.RawMessage(`999`), `{"wrong":true}`)
	peer.replyResult(json.RawMessage(`"007"`), `{"wrong":true}`)
	// A duplicate reply for a request that is already settled is also ignored.
	peer.replyResult(req.ID, `{"ok":true}`)
	peer.replyResult(req.ID, `{"duplicate":true}`)
	o := awaitOutcome(t, out)
	if o.err != nil {
		t.Fatalf("call failed: %v", o.err)
	}
	if string(o.raw) != `{"ok":true}` {
		t.Fatalf("result = %s", o.raw)
	}
	select {
	case err := <-c.Errors():
		t.Fatalf("unknown-id replies reported as errors: %v", err)
	default:
	}
}

func TestClientAbortsOnContextCancel(t *testing.T) {
	c, peer := newTestClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	out := callAsync(ctx, c, "session.history", nil)
	_ = peer.readRequest()
	cancel()
	o := awaitOutcome(t, out)
	assertAppError(t, o.err, "request_aborted")
}

func TestClientCloseRejectsInFlightRequests(t *testing.T) {
	c, peer := newTestClient(t)
	out := callAsync(context.Background(), c, "session.attach", nil)
	_ = peer.readRequest()
	c.Close()
	o := awaitOutcome(t, out)
	assertAppError(t, o.err, "connection_closed")

	_, err := c.Call(context.Background(), "daemon.ping", nil)
	assertAppError(t, err, "connection_closed")
}

func TestClientRejectsEmptyMethod(t *testing.T) {
	c, _ := newTestClient(t)
	if _, err := c.Call(context.Background(), "", nil); err == nil {
		t.Fatal("expected error for empty method")
	}
}

// signalingConn reports when a write reaches the transport, so a test can prove
// that cancellation works while that write is blocked.
type signalingConn struct {
	net.Conn
	writeStarted chan struct{}
	once         sync.Once
}

func (s *signalingConn) Write(p []byte) (int, error) {
	s.once.Do(func() { close(s.writeStarted) })
	return s.Conn.Write(p)
}

func TestClientAbortsCallBlockedInWrite(t *testing.T) {
	a, b := net.Pipe()
	t.Cleanup(func() { _ = b.Close() })
	conn := &signalingConn{Conn: a, writeStarted: make(chan struct{})}
	c := NewClient(conn)
	t.Cleanup(c.Close)

	ctx, cancel := context.WithCancel(context.Background())
	out := callAsync(ctx, c, "session.history", nil)
	select {
	case <-conn.writeStarted:
	case <-time.After(testPeerTimeout):
		t.Fatal("write did not reach the transport")
	}
	// The peer never reads, so the write is still blocked here.
	cancel()
	o := awaitOutcome(t, out)
	assertAppError(t, o.err, "request_aborted")

	// A later request must honor its own deadline even while the abandoned
	// write is still stuck in the transport.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()
	start := time.Now()
	_, err := c.Call(ctx2, "session.catalog", nil)
	if elapsed := time.Since(start); elapsed > testPeerTimeout {
		t.Fatalf("second call blocked for %v", elapsed)
	}
	assertAppError(t, err, "request_timeout")
}
