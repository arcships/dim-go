//go:build unix

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

const testWait = 3 * time.Second

// fakeDaemon is a minimal protocol-v3 peer for the bridge test. It serves the
// control and stream methods the integration controller uses and nothing else.
type fakeDaemon struct {
	t  *testing.T
	ln net.Listener
}

func startFakeDaemon(t *testing.T, address string) *fakeDaemon {
	t.Helper()
	listener, err := net.Listen("unix", address)
	if err != nil {
		t.Fatalf("fake daemon listen %s: %v", address, err)
	}
	daemon := &fakeDaemon{t: t, ln: listener}
	go daemon.accept()
	t.Cleanup(func() { _ = listener.Close() })
	return daemon
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
	defer func() { _ = conn.Close() }()
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
		}
		if err := json.Unmarshal(raw, &request); err != nil {
			return
		}
		d.respond(conn, request.ID, request.Method)
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
	switch method {
	case "hello":
		return appserver.HelloResult{
			InstanceID:      "integration-fake",
			ServerVersion:   "0.0.0-fake",
			ProtocolVersion: appserver.WireProtocolVersion,
			DaemonProfile:   appserver.DaemonProfile{Kind: appserver.DaemonProfileProduct},
			Features:        []string{"session.catalog", "session.history"},
		}, nil
	case "session.attach":
		sequence := int64(2)
		return appserver.AttachResult{Sequence: &sequence}, nil
	case "session.history":
		return json.RawMessage(`{
			"events": [
				{"id": "e1", "sequence": 1, "create_at": 1700000000000, "type": "input", "name": "message", "data": {"user": {"content": [{"type": "text", "text": "hello from the fake daemon"}]}}},
				{"id": "e2", "sequence": 2, "create_at": 1700000000001, "type": "input", "name": "consumed", "data": {"input_id": "e1"}}
			],
			"through": 2,
			"nextAfter": 2,
			"done": true
		}`), nil
	case "session.detach":
		return json.RawMessage(`{}`), nil
	case "session.submit":
		return appserver.SubmitResult{Event: appserver.RawEvent{ID: "submitted", Sequence: 3, CreateAt: 1700000000002, Type: "input", Name: "message", Data: json.RawMessage(`{"text":"submitted"}`)}}, nil
	case "session.stop":
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
