//go:build unix

package supervisor

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

const testWaitTimeout = 2 * time.Second

// socketHome returns a t.TempDir-backed product home whose Unix socket path
// stays under macOS's 104-byte sun_path limit. t.TempDir embeds the test name,
// so the temp root itself is shortened while cleanup stays with t.TempDir.
func socketHome(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "dimsup")
	if err != nil {
		t.Fatalf("short temp root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	t.Setenv("TMPDIR", root)
	return t.TempDir()
}

type fakeDaemonConfig struct {
	InstanceID string
	// HelloRaw overrides the hello result with arbitrary JSON so tests can
	// exercise protocol rejection.
	HelloRaw json.RawMessage
	Features []string
	// PingConnections overrides the reported connection count.
	PingConnections *int
	// PingConnectionsSeq overrides the reported connection count per ping,
	// starting at the first. Later pings keep the last value. It takes
	// precedence over PingConnections.
	PingConnectionsSeq []int
	Sessions           int
	PingErr            *appserver.RPCError
	ShutdownErr        *appserver.RPCError
}

type fakeRequest struct {
	Method string
	Params json.RawMessage
}

type fakeDaemon struct {
	t   *testing.T
	ln  net.Listener
	cfg fakeDaemonConfig

	mu                sync.Mutex
	requests          []fakeRequest
	hellos            int
	pings             int
	shutdowns         int
	shutdownForceList []bool
	active            int
	conns             map[net.Conn]struct{}
	closed            bool
}

func newFakeDaemon(t *testing.T, endpoint Endpoint, cfg fakeDaemonConfig) *fakeDaemon {
	t.Helper()
	if endpoint.Network != NetworkUnix {
		t.Fatalf("fake daemon needs a unix endpoint, got %s", endpoint)
	}
	listener, err := net.Listen("unix", endpoint.Address)
	if err != nil {
		t.Fatalf("fake daemon listen %s: %v", endpoint.Address, err)
	}
	daemon := &fakeDaemon{t: t, ln: listener, cfg: cfg, conns: make(map[net.Conn]struct{})}
	go daemon.accept()
	t.Cleanup(daemon.close)
	return daemon
}

// startFakeDaemon resolves productHome and starts a fake daemon on its endpoint.
func startFakeDaemon(t *testing.T, productHome string, cfg fakeDaemonConfig) (Link, *fakeDaemon) {
	t.Helper()
	link, err := ResolveLink(productHome)
	if err != nil {
		t.Fatalf("resolve link: %v", err)
	}
	return link, newFakeDaemon(t, link.Endpoint, cfg)
}

func (d *fakeDaemon) accept() {
	for {
		conn, err := d.ln.Accept()
		if err != nil {
			return
		}
		d.mu.Lock()
		closed := d.closed
		if !closed {
			d.conns[conn] = struct{}{}
		}
		d.mu.Unlock()
		if closed {
			_ = conn.Close()
			return
		}
		go d.serve(conn)
	}
}

func (d *fakeDaemon) serve(conn net.Conn) {
	hellod := false
	defer func() {
		_ = conn.Close()
		d.mu.Lock()
		delete(d.conns, conn)
		if hellod && d.active > 0 {
			d.active--
		}
		d.mu.Unlock()
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
		d.record(request.Method, request.Params)

		switch request.Method {
		case "hello":
			hellod = true
			d.mu.Lock()
			d.hellos++
			d.active++
			helloRaw := d.cfg.HelloRaw
			instanceID := d.cfg.InstanceID
			features := d.cfg.Features
			d.mu.Unlock()
			if helloRaw == nil {
				helloRaw = defaultFakeHello(instanceID, features)
			}
			d.writeResult(conn, request.ID, helloRaw)
		case "daemon.ping":
			d.mu.Lock()
			d.pings++
			connections := d.active
			if d.cfg.PingConnections != nil {
				connections = *d.cfg.PingConnections
			}
			if index := d.pings - 1; index < len(d.cfg.PingConnectionsSeq) {
				connections = d.cfg.PingConnectionsSeq[index]
			}
			sessions := d.cfg.Sessions
			pingErr := d.cfg.PingErr
			d.mu.Unlock()
			if pingErr != nil {
				d.writeError(conn, request.ID, pingErr)
				continue
			}
			result, err := json.Marshal(map[string]any{
				"sessions":      sessions,
				"connections":   connections,
				"serverVersion": "0.0.0-fake",
			})
			if err != nil {
				return
			}
			d.writeResult(conn, request.ID, result)
		case "daemon.shutdown":
			var params ShutdownParams
			_ = json.Unmarshal(request.Params, &params)
			d.mu.Lock()
			d.shutdowns++
			d.shutdownForceList = append(d.shutdownForceList, params.Force)
			shutdownErr := d.cfg.ShutdownErr
			d.mu.Unlock()
			if shutdownErr != nil {
				d.writeError(conn, request.ID, shutdownErr)
				continue
			}
			d.writeResult(conn, request.ID, json.RawMessage(`{}`))
		default:
			d.writeError(conn, request.ID, &appserver.RPCError{
				Code:    -32601,
				Message: "unknown method",
				Data:    json.RawMessage(`{"appError":"unknown_method"}`),
			})
		}
	}
}

func (d *fakeDaemon) record(method string, params json.RawMessage) {
	d.mu.Lock()
	d.requests = append(d.requests, fakeRequest{Method: method, Params: append(json.RawMessage(nil), params...)})
	d.mu.Unlock()
}

func (d *fakeDaemon) writeResult(conn net.Conn, id, result json.RawMessage) {
	payload, err := json.Marshal(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  json.RawMessage `json:"result"`
	}{"2.0", id, result})
	if err != nil {
		return
	}
	_, _ = conn.Write(append(payload, '\n'))
}

func (d *fakeDaemon) writeError(conn net.Conn, id json.RawMessage, rpcErr *appserver.RPCError) {
	payload, err := json.Marshal(struct {
		JSONRPC string              `json:"jsonrpc"`
		ID      json.RawMessage     `json:"id"`
		Error   *appserver.RPCError `json:"error"`
	}{"2.0", id, rpcErr})
	if err != nil {
		return
	}
	_, _ = conn.Write(append(payload, '\n'))
}

func defaultFakeHello(instanceID string, features []string) json.RawMessage {
	if instanceID == "" {
		instanceID = "fake-instance"
	}
	if features == nil {
		features = []string{"session.catalog"}
	}
	raw, err := json.Marshal(appserver.HelloResult{
		InstanceID:      instanceID,
		ServerVersion:   "0.0.0-fake",
		ProtocolVersion: appserver.WireProtocolVersion,
		DaemonProfile:   appserver.DaemonProfile{Kind: appserver.DaemonProfileProduct},
		Features:        features,
	})
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

func (d *fakeDaemon) close() {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	conns := d.conns
	d.conns = nil
	d.mu.Unlock()
	_ = d.ln.Close()
	for conn := range conns {
		_ = conn.Close()
	}
}

func (d *fakeDaemon) requestsFor(method string) []fakeRequest {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []fakeRequest
	for _, request := range d.requests {
		if request.Method == method {
			out = append(out, request)
		}
	}
	return out
}

func (d *fakeDaemon) helloCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.hellos
}

func (d *fakeDaemon) pingCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pings
}

func (d *fakeDaemon) shutdownCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.shutdowns
}

func (d *fakeDaemon) shutdownForces() []bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]bool(nil), d.shutdownForceList...)
}

func (d *fakeDaemon) activeConnections() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.active
}

// setInstanceID changes the instanceId returned by subsequent hellos, modelling
// a daemon restart without replacing the listener.
func (d *fakeDaemon) setInstanceID(instanceID string) {
	d.mu.Lock()
	d.cfg.InstanceID = instanceID
	d.mu.Unlock()
}

// setHelloRaw overrides subsequent hello results, modelling a daemon whose
// protocol is no longer compatible.
func (d *fakeDaemon) setHelloRaw(raw json.RawMessage) {
	d.mu.Lock()
	d.cfg.HelloRaw = append(json.RawMessage(nil), raw...)
	d.mu.Unlock()
}

// dropConnections closes every live accepted connection but keeps listening,
// so tests can simulate a lost transport and then reconnect.
func (d *fakeDaemon) dropConnections() {
	d.mu.Lock()
	conns := make([]net.Conn, 0, len(d.conns))
	for conn := range d.conns {
		conns = append(conns, conn)
	}
	d.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Close()
	}
}

func (d *fakeDaemon) listenAddress() string { return d.ln.Addr().String() }

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(testWaitTimeout)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

type fakeProcess struct {
	pid  int
	done chan error
}

func newFakeProcess() *fakeProcess {
	return &fakeProcess{pid: 4242, done: make(chan error, 1)}
}

func (p *fakeProcess) Pid() int { return p.pid }

func (p *fakeProcess) Done() <-chan error { return p.done }

type countingSpawner struct {
	calls atomic.Int32
	start func(cfg SpawnConfig) (Process, error)
}

func (s *countingSpawner) spawn(ctx context.Context, cfg SpawnConfig) (Process, error) {
	s.calls.Add(1)
	if s.start == nil {
		return nil, errors.New("unexpected spawn")
	}
	return s.start(cfg)
}

func testContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 5*time.Second)
}
