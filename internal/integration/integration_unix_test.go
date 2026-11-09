//go:build unix

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/session"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

const testWait = 3 * time.Second

// fakeDaemon is a minimal protocol-v3 peer for the bridge test. It serves the
// control and stream methods the integration controller uses and nothing else.
// Fault queues and the hello instanceId are test-controlled so replay-gap and
// daemon-restart paths can be driven deterministically.
type fakeDaemon struct {
	t  *testing.T
	ln net.Listener

	mu       sync.Mutex
	instance string
	faults   map[string][]*appserver.RPCError
	calls    map[string]int
	params   map[string]json.RawMessage
	attached map[net.Conn]struct{}
	writeMu  sync.Mutex

	// inputLookup is the verbatim session.input_lookup reply. Tests set it to a
	// typed accepted event, an unknown result or a malformed payload.
	inputLookup json.RawMessage

	// features overrides the hello.features list when non-nil, and branches is
	// the verbatim session.branches reply (nil means the method is absent).
	features []string
	branches json.RawMessage
}

func startFakeDaemon(t *testing.T, address string) *fakeDaemon {
	t.Helper()
	listener, err := net.Listen("unix", address)
	if err != nil {
		t.Fatalf("fake daemon listen %s: %v", address, err)
	}
	daemon := &fakeDaemon{
		t:        t,
		ln:       listener,
		instance: "integration-fake",
		faults:   map[string][]*appserver.RPCError{},
		calls:    map[string]int{},
		params:   map[string]json.RawMessage{},
		attached: map[net.Conn]struct{}{},
	}
	go daemon.accept()
	t.Cleanup(func() { _ = listener.Close() })
	return daemon
}

// subscriptionChangedFault is the daemon's §13 replay-gap error.
func subscriptionChangedFault() *appserver.RPCError {
	return &appserver.RPCError{Code: -32000, Message: "subscription changed", Data: json.RawMessage(`{"appError":"subscription_changed"}`)}
}

// setInstance changes the instanceId reported by the next hello. A different
// value is what the supervisor treats as a daemon restart.
func (d *fakeDaemon) setInstance(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.instance = id
}

// fail queues one reply fault per call for method.
func (d *fakeDaemon) fail(method string, faults ...*appserver.RPCError) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.faults[method] = append(d.faults[method], faults...)
}

// setInputLookup scripts the verbatim session.input_lookup reply.
func (d *fakeDaemon) setInputLookup(raw json.RawMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.inputLookup = append(json.RawMessage(nil), raw...)
}

// setFeatures replaces the hello.features list. It must run before Start.
func (d *fakeDaemon) setFeatures(features ...string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.features = append([]string(nil), features...)
}

// setBranches scripts the verbatim session.branches reply.
func (d *fakeDaemon) setBranches(raw json.RawMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.branches = append(json.RawMessage(nil), raw...)
}

// callCount reports how many requests were received for method.
func (d *fakeDaemon) callCount(method string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls[method]
}

// recordParams keeps the last request params per method so tests can assert
// what the controller actually sent, not just that it sent something.
func (d *fakeDaemon) recordParams(method string, params json.RawMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.params[method] = append(json.RawMessage(nil), params...)
}

// lastParams decodes the last request params for method into a plain map.
func (d *fakeDaemon) lastParams(method string) map[string]any {
	d.mu.Lock()
	raw := append(json.RawMessage(nil), d.params[method]...)
	d.mu.Unlock()
	params := map[string]any{}
	if len(raw) == 0 {
		return params
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		d.t.Fatalf("decode %s params: %v", method, err)
	}
	return params
}

// beginCall records a request and returns its scripted fault, if any.
func (d *fakeDaemon) beginCall(method string) *appserver.RPCError {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls[method]++
	queue := d.faults[method]
	if len(queue) == 0 {
		return nil
	}
	fault := queue[0]
	d.faults[method] = queue[1:]
	return fault
}

// emit writes one server notification to every connection that served a
// session.attach. The test drives session.closed with it instead of sleeping.
func (d *fakeDaemon) emit(method string, params any) {
	d.t.Helper()
	d.mu.Lock()
	conns := make([]net.Conn, 0, len(d.attached))
	for conn := range d.attached {
		conns = append(conns, conn)
	}
	d.mu.Unlock()
	value := map[string]any{"jsonrpc": "2.0", "method": method, "params": params}
	for _, conn := range conns {
		d.write(conn, value)
	}
}

func (d *fakeDaemon) accept() {
	for {
		conn, err := d.ln.Accept()
		if err != nil {
			return
		}
		go d.serve(conn)
	}
}

func (d *fakeDaemon) serve(conn net.Conn) {
	defer func() {
		d.mu.Lock()
		delete(d.attached, conn)
		d.mu.Unlock()
		_ = conn.Close()
	}()
	frames := appserver.NewFrameReader(conn)
	for {
		raw, err := frames.ReadFrame()
		if err != nil {
			return
		}
		if err := appserver.ValidateEnvelope(raw); err != nil {
			return
		}
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(raw, &request); err != nil {
			return
		}
		d.recordParams(request.Method, request.Params)
		d.respond(conn, request.ID, request.Method)
		if request.Method == "session.attach" {
			d.mu.Lock()
			d.attached[conn] = struct{}{}
			d.mu.Unlock()
		}
	}
}

// respond writes one reply for a request.
func (d *fakeDaemon) respond(conn net.Conn, id json.RawMessage, method string) {
	result, fault := d.result(method)
	if fault != nil {
		d.write(conn, map[string]any{"jsonrpc": "2.0", "id": id, "error": fault})
		return
	}
	d.write(conn, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func (d *fakeDaemon) result(method string) (any, *appserver.RPCError) {
	if fault := d.beginCall(method); fault != nil {
		return nil, fault
	}
	switch method {
	case "hello":
		d.mu.Lock()
		instance := d.instance
		features := d.features
		d.mu.Unlock()
		if features == nil {
			features = []string{"session.catalog", "session.history"}
		}
		return appserver.HelloResult{
			InstanceID:      instance,
			ServerVersion:   "0.0.0-fake",
			ProtocolVersion: appserver.WireProtocolVersion,
			DaemonProfile:   appserver.DaemonProfile{Kind: appserver.DaemonProfileProduct},
			Features:        features,
		}, nil
	case "session.attach":
		sequence := int64(2)
		return appserver.AttachResult{Sequence: &sequence}, nil
	case "session.history":
		// Echo the requested bound and window so main-branch and per-branch
		// first-screen reads both satisfy the §9 paging invariants.
		d.mu.Lock()
		rawParams := append(json.RawMessage(nil), d.params[method]...)
		d.mu.Unlock()
		var params appserver.HistoryParams
		_ = json.Unmarshal(rawParams, &params)
		through := int64(2)
		if params.Through != nil {
			through = *params.Through
		}
		after := int64(0)
		if params.After != nil {
			after = *params.After
		}
		all := []appserver.RawEvent{
			{ID: "e1", Sequence: 1, CreateAt: 1700000000000, Type: "input", Name: "message", Data: json.RawMessage(`{"user": {"content": [{"type": "text", "text": "hello from the fake daemon"}]}}`)},
			{ID: "e2", Sequence: 2, CreateAt: 1700000000001, Type: "input", Name: "consumed", Data: json.RawMessage(`{"input_id": "e1"}`)},
		}
		events := make([]appserver.RawEvent, 0, len(all))
		for _, event := range all {
			if event.Sequence > after && event.Sequence <= through {
				events = append(events, event)
			}
		}
		nextAfter := through
		return appserver.HistoryResult{Events: events, Through: &through, NextAfter: &nextAfter, Done: true}, nil
	case "session.branches":
		d.mu.Lock()
		raw := append(json.RawMessage(nil), d.branches...)
		d.mu.Unlock()
		if raw == nil {
			return nil, &appserver.RPCError{Code: -32601, Message: "session.branches not scripted", Data: json.RawMessage(`{"appError":"unknown_method"}`)}
		}
		return raw, nil
	case "session.branch_end":
		return json.RawMessage(`{}`), nil
	case "session.detach":
		return json.RawMessage(`{}`), nil
	case "session.submit":
		return appserver.SubmitResult{Event: appserver.RawEvent{ID: "submitted", Sequence: 3, CreateAt: 1700000000002, Type: "input", Name: "message", Data: json.RawMessage(`{"text":"submitted"}`)}}, nil
	case "session.stop":
		return json.RawMessage(`{}`), nil
	case "session.input_lookup":
		d.mu.Lock()
		raw := append(json.RawMessage(nil), d.inputLookup...)
		d.mu.Unlock()
		if raw == nil {
			return nil, &appserver.RPCError{Code: -32601, Message: "session.input_lookup not scripted", Data: json.RawMessage(`{"appError":"unknown_method"}`)}
		}
		return raw, nil
	case "approval.respond":
		return json.RawMessage(`{}`), nil
	case "session.catalog":
		title := "Fake session"
		return appserver.CatalogResult{Sessions: []appserver.SessionSummary{{
			SessionID:       "s1",
			Cwd:             "/tmp",
			Model:           appserver.SessionModel{ProviderID: "provider", ModelID: "model"},
			CreatedAt:       1700000000000,
			Sequence:        2,
			AttachedClients: 1,
			UpdatedAt:       1700000000001,
			Title:           &title,
		}}}, nil
	default:
		return nil, &appserver.RPCError{Code: -32601, Message: "unknown method", Data: json.RawMessage(`{"appError":"unknown_method"}`)}
	}
}

func (d *fakeDaemon) write(conn net.Conn, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		return
	}
	// Serialize frames: serve goroutines and emit can write to the same stream.
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	_, _ = conn.Write(append(payload, '\n'))
}

// fakeHome returns a short product home so the Unix socket path fits sun_path.
func fakeHome(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "dimint")
	if err != nil {
		t.Fatalf("temp home: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}

func testContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// startBridge starts a fake daemon and a supervisor linked to it, and registers
// cleanup. It is the shared fixture for replay-gap and re-attach tests.
func startBridge(t *testing.T) (*fakeDaemon, *supervisor.Supervisor, context.Context) {
	t.Helper()
	home := fakeHome(t)
	link, err := supervisor.ResolveLink(home)
	if err != nil {
		t.Fatalf("resolve link: %v", err)
	}
	daemon := startFakeDaemon(t, link.Endpoint.Address)
	sup, err := supervisor.New(supervisor.Config{ProductHome: home})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	ctx, cancel := testContext(t)
	t.Cleanup(func() {
		cancel()
		closeCtx, closeCancel := context.WithTimeout(context.Background(), testWait)
		defer closeCancel()
		_, _ = sup.Close(closeCtx)
	})
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return daemon, sup, ctx
}

// waitSessionDone waits for the session drain goroutine to finish. It is a
// channel wait, not a sleep-poll: Done closes exactly once, after the stream
// reached its terminal state and the terminal error was published.
func waitSessionDone(t *testing.T, s *session.Session) {
	t.Helper()
	timer := time.NewTimer(testWait)
	defer timer.Stop()
	select {
	case <-s.Done():
	case <-timer.C:
		t.Fatalf("session did not observe a transport end within %s", testWait)
	}
}

// closeControllerOnCleanup detaches the controller with a bounded context.
func closeControllerOnCleanup(t *testing.T, controller *SessionController) {
	t.Helper()
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), testWait)
		defer closeCancel()
		_ = controller.Close(closeCtx)
	})
}

// TestSessionControllerBridge drives the bridge end to end over a fake daemon:
// supervisor control -> session stream -> first history window -> viewmodel
// rows, plus catalog rows and detach.
func TestSessionControllerBridge(t *testing.T) {
	home := fakeHome(t)
	link, err := supervisor.ResolveLink(home)
	if err != nil {
		t.Fatalf("resolve link: %v", err)
	}
	_ = startFakeDaemon(t, link.Endpoint.Address)

	sup, err := supervisor.New(supervisor.Config{ProductHome: home})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	// A healthy attachment must not be replaced by an accidental reattach;
	// this also pins the atomicity contract before exercising transport loss.
	oldConn, oldSession := controller.Conn, controller.Session
	if err := controller.Reattach(ctx); err == nil {
		t.Fatal("Reattach on healthy session unexpectedly succeeded")
	}
	if controller.Conn != oldConn || controller.Session != oldSession {
		t.Fatal("failed Reattach mutated the existing controller")
	}
	oldConn.Client().Close()
	waitSessionDone(t, controller.Session)
	if !controller.Session.Lost() {
		t.Fatal("session did not observe transport loss")
	}
	lostState := controller.State(appserver.SessionSummary{SessionID: "s1"})
	if lostState.Connection != ui.ConnError || lostState.ConnectionError == "" {
		t.Fatalf("lost state = %#v, want visible connection error", lostState)
	}
	// The stream can report Lost just before the session drain publishes Err;
	// repeated reads must still never produce a blank banner.
	for i := 0; i < 20; i++ {
		if state := controller.State(appserver.SessionSummary{SessionID: "s1"}); state.ConnectionError == "" {
			t.Fatalf("lost state read %d had blank error: %#v", i, state)
		}
	}
	if err := controller.Reattach(ctx); err != nil {
		t.Fatalf("Reattach after transport loss: %v", err)
	}
	if controller.Conn == oldConn || controller.Session == oldSession {
		t.Fatal("successful Reattach retained stale controller")
	}
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), testWait)
		defer closeCancel()
		_ = controller.Close(closeCtx)
		_, _ = sup.Close(closeCtx)
	})

	summary := appserver.SessionSummary{SessionID: "s1"}
	state := controller.State(summary)
	if state.Selected != "s1" || state.Connection != ui.ConnReady {
		t.Fatalf("state = selected %q connection %v", state.Selected, state.Connection)
	}
	if len(state.Messages) != 1 || state.Messages[0].Content != "hello from the fake daemon" {
		t.Fatalf("messages = %#v", state.Messages)
	}
	if state.Messages[0].Author != ui.AuthorUser {
		t.Fatalf("author = %v, want user", state.Messages[0].Author)
	}
	if state.Run != ui.RunIdle {
		t.Fatalf("run = %v, want idle after the input was consumed", state.Run)
	}

	result, submissionID, err := controller.SubmitWithID(ctx, "hello again")
	if err != nil {
		t.Fatalf("SubmitWithID: %v", err)
	}
	if submissionID == "" || result.Event.ID != "submitted" {
		t.Fatalf("submit result = %#v, id %q", result, submissionID)
	}
	if err := controller.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if _, _, err := controller.SubmitWithID(nil, "nil context"); err == nil {
		t.Fatal("SubmitWithID with nil context should fail")
	}
	cancelled, cancelSubmit := context.WithCancel(ctx)
	cancelSubmit()
	if _, _, err := controller.SubmitWithID(cancelled, "cancelled"); !errors.Is(err, appserver.ErrUnknownResult) {
		t.Fatalf("cancelled SubmitWithID error = %v, want unknown result", err)
	}

	rows, err := Catalog(ctx, sup)
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "s1" || rows[0].Title != "Fake session" {
		t.Fatalf("catalog rows = %#v", rows)
	}

	if err := controller.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if controller.Session.TerminalState() != appserver.StreamStateDetached {
		t.Fatalf("terminal state = %s, want detached", controller.Session.TerminalState())
	}
}

func TestSessionControllerStateReattachConcurrent(t *testing.T) {
	home := fakeHome(t)
	link, err := supervisor.ResolveLink(home)
	if err != nil {
		t.Fatalf("resolve link: %v", err)
	}
	_ = startFakeDaemon(t, link.Endpoint.Address)
	sup, err := supervisor.New(supervisor.Config{ProductHome: home})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	old := controller.Conn
	old.Client().Close()
	waitSessionDone(t, controller.Session)
	if !controller.Session.Lost() {
		t.Fatal("session did not observe transport loss")
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), testWait)
		defer closeCancel()
		_ = controller.Close(closeCtx)
		_, _ = sup.Close(closeCtx)
	}()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				state := controller.State(appserver.SessionSummary{SessionID: "s1"})
				if state == nil {
					t.Errorf("State returned nil")
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := controller.Reattach(ctx); err != nil {
			t.Errorf("Reattach: %v", err)
		}
	}()
	wg.Wait()
	if controller.Conn == old {
		t.Fatal("Reattach retained old connection")
	}
}

func TestStateWithoutSessionIsSafe(t *testing.T) {
	var controller *SessionController
	if err := controller.Close(context.Background()); err != nil {
		t.Fatalf("nil controller Close: %v", err)
	}
	state := (&SessionController{}).State(appserver.SessionSummary{SessionID: "s1"})
	if state == nil || state.Connection != ui.ConnUnknown || len(state.Messages) != 0 {
		t.Fatalf("state = %#v", state)
	}
	var nilController *SessionController
	if state := nilController.State(appserver.SessionSummary{}); state == nil || state.Connection != ui.ConnUnknown {
		t.Fatalf("nil controller state = %#v", state)
	}
}

func TestSessionTitlePrecedence(t *testing.T) {
	title := "Title"
	preview := "Preview"
	cases := []struct {
		name    string
		summary appserver.SessionSummary
		want    string
	}{
		{"title", appserver.SessionSummary{SessionID: "s1", Title: &title, Preview: &preview}, title},
		{"preview", appserver.SessionSummary{SessionID: "s1", Preview: &preview}, preview},
		{"id", appserver.SessionSummary{SessionID: "s1"}, "s1"},
		{"empty title falls back to preview", appserver.SessionSummary{SessionID: "s1", Title: new(string), Preview: &preview}, preview},
	}
	for _, tc := range cases {
		if got := sessionTitle(tc.summary); got != tc.want {
			t.Errorf("%s: sessionTitle = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestOpenSessionRetriesSubscriptionChangedReadOnce pins the contract's only
// allowed replay-gap recovery: attach + first history is a read, so it may
// re-attach and retry exactly once.
func TestOpenSessionRetriesSubscriptionChangedReadOnce(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	daemon.fail("session.history", subscriptionChangedFault())
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession after one replay gap: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	if got := daemon.callCount("session.attach"); got != 2 {
		t.Fatalf("session.attach calls = %d, want 2 (initial + one retry)", got)
	}
	if got := daemon.callCount("session.history"); got != 2 {
		t.Fatalf("session.history calls = %d, want 2", got)
	}
	state := controller.State(appserver.SessionSummary{SessionID: "s1"})
	if state.Connection != ui.ConnReady || len(state.Messages) != 1 {
		t.Fatalf("state after retry = %#v", state)
	}
}

// TestOpenSessionSubscriptionChangedFailsClosedAfterSingleRetry proves the retry
// is bounded: a second replay gap is surfaced as a WIP error, not a loop.
func TestOpenSessionSubscriptionChangedFailsClosedAfterSingleRetry(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	daemon.fail("session.history", subscriptionChangedFault(), subscriptionChangedFault(), subscriptionChangedFault())
	controller, err := OpenSession(ctx, sup, "s1")
	if controller != nil {
		t.Fatal("OpenSession returned a controller for a persistent replay gap")
	}
	if !errors.Is(err, ErrReplayGap) {
		t.Fatalf("error = %v, want %v", err, ErrReplayGap)
	}
	var gap *ReplayGapError
	if !errors.As(err, &gap) || !gap.ReadOnly || !gap.Retried {
		t.Fatalf("error = %#v, want a retried read-only ReplayGapError", err)
	}
	if !appserver.IsAppError(err, appserver.AppErrorSubscriptionChanged) {
		t.Fatalf("error = %v, want the daemon subscription_changed preserved", err)
	}
	if got := daemon.callCount("session.history"); got != 2 {
		t.Fatalf("session.history calls = %d, want exactly 2 attempts", got)
	}
	if got := daemon.callCount("session.attach"); got != 2 {
		t.Fatalf("session.attach calls = %d, want exactly 2 attempts", got)
	}
}

// TestCatalogRetriesSubscriptionChangedReadOnce covers the control-connection
// read path: session.catalog is read-only and gets the same single retry.
func TestCatalogRetriesSubscriptionChangedReadOnce(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	daemon.fail("session.catalog", subscriptionChangedFault())
	rows, err := Catalog(ctx, sup)
	if err != nil {
		t.Fatalf("Catalog after one replay gap: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "s1" {
		t.Fatalf("rows = %#v", rows)
	}
	if got := daemon.callCount("session.catalog"); got != 2 {
		t.Fatalf("session.catalog calls = %d, want 2", got)
	}
}

// TestCatalogSubscriptionChangedFailsClosedAfterSingleRetry checks the bounded
// read retry does not become an unbounded catalog loop.
func TestCatalogSubscriptionChangedFailsClosedAfterSingleRetry(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	daemon.fail("session.catalog", subscriptionChangedFault(), subscriptionChangedFault(), subscriptionChangedFault())
	if _, err := Catalog(ctx, sup); !errors.Is(err, ErrReplayGap) {
		t.Fatalf("Catalog error = %v, want %v", err, ErrReplayGap)
	}
	if got := daemon.callCount("session.catalog"); got != 2 {
		t.Fatalf("session.catalog calls = %d, want exactly 2 attempts", got)
	}
}

// TestSubmitSubscriptionChangedIsNotReplayed pins the write rule: a mutation
// that hits a replay gap is reported once and never resent.
func TestSubmitSubscriptionChangedIsNotReplayed(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	daemon.fail("session.submit", subscriptionChangedFault(), subscriptionChangedFault())
	_, submissionID, err := controller.SubmitWithID(ctx, "hello again")
	if submissionID == "" {
		t.Fatal("SubmitWithID lost the idempotency key on a replay gap")
	}
	if !errors.Is(err, ErrReplayGap) {
		t.Fatalf("SubmitWithID error = %v, want %v", err, ErrReplayGap)
	}
	var gap *ReplayGapError
	if !errors.As(err, &gap) || gap.ReadOnly {
		t.Fatalf("error = %#v, want a fail-closed write ReplayGapError", err)
	}
	if got := daemon.callCount("session.submit"); got != 1 {
		t.Fatalf("session.submit calls = %d, want 1 (no replay)", got)
	}
}

// TestLookupInputReturnsTypedAccepted drives §10 read-back over the fake daemon:
// the accepted tagged union is decoded into the typed result, the submission is
// never resent, and the reply is not cached across calls.
func TestLookupInputReturnsTypedAccepted(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	daemon.setInputLookup(json.RawMessage(`{"status":"accepted","event":{"id":"e3","sequence":3,"create_at":1700000000002,"type":"input","name":"message","data":{"text":"hello again"}}}`))

	result, err := controller.LookupInput(ctx, "main", "sub-1")
	if err != nil {
		t.Fatalf("LookupInput: %v", err)
	}
	if !result.Accepted() || result.Status != appserver.InputLookupAccepted {
		t.Fatalf("result = %#v, want accepted", result)
	}
	if result.Event == nil || result.Event.ID != "e3" || result.Event.Sequence != 3 {
		t.Fatalf("accepted event = %#v", result.Event)
	}
	if got := daemon.callCount("session.submit"); got != 0 {
		t.Fatalf("session.submit calls = %d, want 0 (lookup never resends)", got)
	}
	if got := daemon.callCount("session.input_lookup"); got != 1 {
		t.Fatalf("session.input_lookup calls = %d, want 1", got)
	}

	// A second lookup must ask the daemon again instead of replaying a cached
	// accepted answer.
	daemon.setInputLookup(json.RawMessage(`{"status":"unknown"}`))
	again, err := controller.LookupInput(ctx, "main", "sub-1")
	if err != nil {
		t.Fatalf("second LookupInput: %v", err)
	}
	if again.Accepted() || again.Status != appserver.InputLookupUnknown {
		t.Fatalf("second result = %#v, want unknown (not cached)", again)
	}
	if got := daemon.callCount("session.input_lookup"); got != 2 {
		t.Fatalf("session.input_lookup calls = %d, want 2", got)
	}
}

// TestLookupInputReturnsTypedUnknown pins that an unknown read-back is a final
// typed answer, not an error and never an accepted result.
func TestLookupInputReturnsTypedUnknown(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	daemon.setInputLookup(json.RawMessage(`{"status":"unknown"}`))

	result, err := controller.LookupInput(ctx, "main", "sub-1")
	if err != nil {
		t.Fatalf("LookupInput: %v", err)
	}
	if result.Status != appserver.InputLookupUnknown || result.Accepted() || result.Event != nil {
		t.Fatalf("result = %#v, want typed unknown with no event", result)
	}
}

// TestLookupInputMalformedRejectsReply covers the fail-closed decode: a status
// outside the tagged union, an accepted status without its event and invalid
// JSON are errors whose zero result is never accepted.
func TestLookupInputMalformedRejectsReply(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	for _, raw := range []json.RawMessage{
		json.RawMessage(`{"status":"accepted"}`),
		json.RawMessage(`{"status":"pending","event":{"id":"e3","sequence":3,"create_at":1700000000002,"type":"input","name":"message"}}`),
		json.RawMessage(`{"status":`),
	} {
		daemon.setInputLookup(raw)
		result, err := controller.LookupInput(ctx, "main", "sub-1")
		if err == nil {
			t.Fatalf("malformed reply %s: expected an error", raw)
		}
		if result.Accepted() || result.Status == appserver.InputLookupAccepted {
			t.Fatalf("malformed reply %s: result = %#v, want a non-accepted zero result", raw, result)
		}
	}
}

// TestLookupInputFailsFastWithoutCall pins the parameter checks: nil context,
// empty submission id and no session all fail before any request. An empty
// branch id is valid and defaults to the session's current branch.
func TestLookupInputFailsFastWithoutCall(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	cases := []struct {
		name         string
		controller   *SessionController
		ctx          context.Context
		branchID     string
		submissionID string
	}{
		{"nil controller", nil, ctx, "main", "sub-1"},
		{"nil context", controller, nil, "main", "sub-1"},
		{"empty submission", controller, ctx, "main", ""},
		{"no session", &SessionController{}, ctx, "main", "sub-1"},
	}
	for _, tc := range cases {
		if _, err := tc.controller.LookupInput(tc.ctx, tc.branchID, tc.submissionID); err == nil {
			t.Errorf("%s: expected an error", tc.name)
		}
	}
	if got := daemon.callCount("session.input_lookup"); got != 0 {
		t.Fatalf("session.input_lookup calls = %d, want 0", got)
	}
}

// TestLookupInputDefaultsBranch keeps the empty-branch case usable from the
// window: it must ask the daemon with the session's current branch instead of
// failing locally or inventing a branch id.
func TestLookupInputDefaultsBranch(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	daemon.setInputLookup(json.RawMessage(`{"status":"unknown"}`))

	if _, err := controller.LookupInput(ctx, "", "sub-1"); err != nil {
		t.Fatalf("LookupInput with default branch: %v", err)
	}
	if got := daemon.callCount("session.input_lookup"); got != 1 {
		t.Fatalf("session.input_lookup calls = %d, want 1", got)
	}
	params := daemon.lastParams("session.input_lookup")
	if params["branchId"] != "main" {
		t.Fatalf("branchId = %v, want the session's current branch main", params["branchId"])
	}
}

// TestReattachFailsClosedWhenDaemonGenerationAdvanced covers two facts at once:
// a newer hello instanceId advances the supervisor generation, and an old but
// still-live attachment must not be replaced (that would double-attach).
func TestReattachFailsClosedWhenDaemonGenerationAdvanced(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	oldConn, oldSession := controller.Conn, controller.Session
	daemon.setInstance("integration-fake-2")
	observed, err := sup.OpenStream(ctx)
	if err != nil {
		t.Fatalf("OpenStream for the restarted instance: %v", err)
	}
	observed.Close()
	if observed.Generation() != oldConn.Generation()+1 || !oldConn.Stale() {
		t.Fatalf("generation = %d old = %d stale = %v, want generation advanced and old stale", observed.Generation(), oldConn.Generation(), oldConn.Stale())
	}
	err = controller.Reattach(ctx)
	if !errors.Is(err, ErrSessionNotLost) {
		t.Fatalf("Reattach error = %v, want %v", err, ErrSessionNotLost)
	}
	if controller.Conn != oldConn || controller.Session != oldSession {
		t.Fatal("fail-closed Reattach mutated the controller")
	}
}

// TestReattachAfterDaemonRestartRebuildsFromNewInstance pins the verified
// instanceId rule: a restart is not an error, the attachment is rebuilt from the
// new instance's history and the identity change is reported.
func TestReattachAfterDaemonRestartRebuildsFromNewInstance(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	oldSession := controller.Session
	daemon.setInstance("integration-fake-2")
	controller.Conn.Client().Close()
	waitSessionDone(t, oldSession)
	if err := controller.Reattach(ctx); err != nil {
		t.Fatalf("Reattach after daemon restart: %v", err)
	}
	info, ok := controller.LastReattach()
	if !ok {
		t.Fatal("LastReattach has no record after a successful Reattach")
	}
	want := ReattachResult{
		SessionID:          "s1",
		PreviousGeneration: 1,
		Generation:         2,
		PreviousInstanceID: "integration-fake",
		InstanceID:         "integration-fake-2",
		Restarted:          true,
	}
	if info != want {
		t.Fatalf("LastReattach = %#v, want %#v", info, want)
	}
	if controller.Session == oldSession || controller.Session.Lost() {
		t.Fatal("Reattach kept the stale or lost session")
	}
	state := controller.State(appserver.SessionSummary{SessionID: "s1"})
	if state.Connection != ui.ConnReady || len(state.Messages) != 1 {
		t.Fatalf("state after restart re-attach = %#v", state)
	}
}

// TestReattachRetriesSubscriptionChangedOnce covers the replay-gap retry on the
// re-attach path and pins that a plain re-attach is not reported as a restart.
func TestReattachRetriesSubscriptionChangedOnce(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	controller.Conn.Client().Close()
	waitSessionDone(t, controller.Session)
	daemon.fail("session.attach", subscriptionChangedFault())
	if err := controller.Reattach(ctx); err != nil {
		t.Fatalf("Reattach after one replay gap: %v", err)
	}
	if got := daemon.callCount("session.attach"); got != 3 {
		t.Fatalf("session.attach calls = %d, want 3 (initial + failed retry + retry)", got)
	}
	info, ok := controller.LastReattach()
	if !ok || info.Restarted || info.InstanceID != info.PreviousInstanceID {
		t.Fatalf("LastReattach = %#v, want a same-instance re-attach", info)
	}
}

// TestReattachRejectsClosedSession pins session.closed as terminal: no re-attach
// may turn a closed session back into a live one.
func TestReattachRejectsClosedSession(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	oldConn, oldSession := controller.Conn, controller.Session
	daemon.emit("session.closed", appserver.ClosedNotice{SessionID: "s1", Reason: "finished"})
	waitSessionDone(t, oldSession)
	closed, ok := oldSession.Closed()
	if !ok || closed.Reason != "finished" {
		t.Fatalf("closed notice = %#v/%v, want reason finished", closed, ok)
	}
	if oldSession.TerminalState() != appserver.StreamStateClosed {
		t.Fatalf("terminal state = %s, want closed", oldSession.TerminalState())
	}
	if err := controller.Reattach(ctx); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Reattach error = %v, want %v", err, ErrSessionClosed)
	}
	if controller.Conn != oldConn || controller.Session != oldSession {
		t.Fatal("closed-session Reattach mutated the controller")
	}
}

// TestReattachAfterCloseFailsClosed pins that a closed controller cannot be
// resurrected by a racing re-attach.
func TestReattachAfterCloseFailsClosed(t *testing.T) {
	_, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	if err := controller.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if controller.Session.TerminalState() != appserver.StreamStateDetached {
		t.Fatalf("terminal state = %s, want detached", controller.Session.TerminalState())
	}
	if err := controller.Reattach(ctx); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Reattach after Close error = %v, want %v", err, ErrSessionClosed)
	}
}

// countApprovalRows counts the pending-approval rows in a bridge frame.
func countApprovalRows(rows []ui.Message) int {
	count := 0
	for _, row := range rows {
		if row.Author == ui.AuthorApproval {
			count++
		}
	}
	return count
}

// waitForApprovalRows polls the bridge frames until the approval-row count
// matches. The session applies notifications on its drain goroutine, so there is
// no channel to wait on without reaching into its projection; the poll is
// bounded by the test context, not an open-ended sleep.
func waitForApprovalRows(t *testing.T, controller *SessionController, summary appserver.SessionSummary, want int) []ui.Message {
	t.Helper()
	deadline := time.Now().Add(testWait)
	var rows []ui.Message
	for time.Now().Before(deadline) {
		rows = controller.StateForBranch(summary, "main").Messages
		if countApprovalRows(rows) == want {
			return rows
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("approval rows = %d, want %d (messages %#v)", countApprovalRows(rows), want, rows)
	return nil
}

// TestApprovalDecisionRoundTripThroughBridge drives the approval vertical slice
// over the fake daemon: approval.request becomes a row with the daemon's
// identity, RespondApproval sends the folded decision, the row stays until the
// daemon settles it, and approval.resolved removes it. Nothing is resolved
// locally before the App Server answers.
func TestApprovalDecisionRoundTripThroughBridge(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	summary := appserver.SessionSummary{SessionID: "s1"}

	daemon.emit("approval.request", appserver.ApprovalRequest{
		SessionID: "s1",
		BranchID:  "main",
		RequestID: "req-1",
		Source:    appserver.ApprovalSource{Kind: "tool"},
		Call:      appserver.ApprovalCall{Name: "exec", Input: json.RawMessage(`{"cmd":"deploy"}`)},
		Reason:    "deploy to production",
	})
	rows := waitForApprovalRows(t, controller, summary, 1)
	row := rows[len(rows)-1]
	if row.Approval == nil || row.Approval.RequestID != "req-1" || row.Approval.BranchID != "main" {
		t.Fatalf("approval row identity = %#v", row.Approval)
	}
	if row.Content != "exec — deploy to production" {
		t.Fatalf("approval row content = %q", row.Content)
	}

	outcome, err := controller.RespondApproval(ctx, "main", "req-1", appserver.DecisionAllow)
	if err != nil || outcome != appserver.ApprovalOutcomeResponded {
		t.Fatalf("RespondApproval = %q, %v; want responded", outcome, err)
	}
	if got := daemon.callCount("approval.respond"); got != 1 {
		t.Fatalf("approval.respond calls = %d, want 1", got)
	}
	if rows := controller.StateForBranch(summary, "main").Messages; countApprovalRows(rows) != 1 {
		t.Fatalf("approval row was resolved locally before the daemon settled it: %#v", rows)
	}

	daemon.emit("approval.resolved", appserver.ApprovalResolved{
		SessionID: "s1",
		BranchID:  "main",
		RequestID: "req-1",
		Decision:  appserver.DecisionAllow,
	})
	waitForApprovalRows(t, controller, summary, 0)
}

// TestApprovalAlreadyGoneIsCancelledNotResent covers §12's -32602 rule through
// the bridge: the outcome is cancelled, the daemon's answer is not resent, and
// the stale pending row is dropped because the daemon said it no longer exists.
func TestApprovalAlreadyGoneIsCancelledNotResent(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	summary := appserver.SessionSummary{SessionID: "s1"}

	daemon.emit("approval.request", appserver.ApprovalRequest{
		SessionID: "s1",
		BranchID:  "main",
		RequestID: "req-1",
		Source:    appserver.ApprovalSource{Kind: "tool"},
		Call:      appserver.ApprovalCall{Name: "exec"},
	})
	waitForApprovalRows(t, controller, summary, 1)

	daemon.fail("approval.respond", &appserver.RPCError{Code: -32602, Message: "invalid params"})
	outcome, err := controller.RespondApproval(ctx, "main", "req-1", appserver.DecisionDeny)
	if err != nil {
		t.Fatalf("RespondApproval error = %v, want cancelled without an error", err)
	}
	if outcome != appserver.ApprovalOutcomeCancelled {
		t.Fatalf("outcome = %q, want cancelled", outcome)
	}
	if got := daemon.callCount("approval.respond"); got != 1 {
		t.Fatalf("approval.respond calls = %d, want 1 (never resent)", got)
	}
	waitForApprovalRows(t, controller, summary, 0)
}
