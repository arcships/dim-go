//go:build unix

package supervisor

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

func TestStartExisting(t *testing.T) {
	home := socketHome(t)
	link, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "existing"})
	spawner := &countingSpawner{}
	sup := newTestSupervisor(t, Config{ProductHome: home, Spawn: spawner.spawn})
	ctx, cancel := testContext(t)
	defer cancel()

	conn, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if conn.InstanceID() != "existing" {
		t.Fatalf("instanceId = %q, want existing", conn.InstanceID())
	}
	if spawner.calls.Load() != 0 {
		t.Fatalf("spawn calls = %d, want 0 for a listening daemon", spawner.calls.Load())
	}
	if sup.OwnsDaemon() {
		t.Fatal("supervisor claims ownership of a daemon it did not spawn")
	}
	if got := sup.ActiveConnections(); got != 1 {
		t.Fatalf("ActiveConnections = %d, want 1", got)
	}
	if got := fake.helloCount(); got != 1 {
		t.Fatalf("hello calls = %d, want 1", got)
	}
	endpoint, ok := sup.Endpoint()
	if !ok || endpoint != link.Endpoint {
		t.Fatalf("Endpoint = %v, %v; want %v", endpoint, ok, link.Endpoint)
	}

	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if report.Owned || report.Skipped != CloseSkippedNotOwned {
		t.Fatalf("Close report = %+v, want a not_owned report", report)
	}
	if fake.pingCount() != 0 || fake.shutdownCount() != 0 {
		t.Fatalf("existing daemon saw ping=%d shutdown=%d, want 0/0", fake.pingCount(), fake.shutdownCount())
	}
	if _, ok := sup.Control(); ok {
		t.Fatal("control connection still tracked after Close")
	}
	waitFor(t, "existing connection closed", func() bool { return fake.activeConnections() == 0 })
}

func TestStartSpawns(t *testing.T) {
	home := socketHome(t)
	link, err := ResolveLink(home)
	if err != nil {
		t.Fatalf("ResolveLink: %v", err)
	}
	var fake *fakeDaemon
	spawner := &countingSpawner{start: func(cfg SpawnConfig) (Process, error) {
		if cfg.Endpoint != link.Endpoint {
			t.Errorf("spawn endpoint = %v, want %v", cfg.Endpoint, link.Endpoint)
		}
		if cfg.NativeHome != link.NativeHome {
			t.Errorf("spawn native home = %q, want %q", cfg.NativeHome, link.NativeHome)
		}
		if cfg.ProductHome != home {
			t.Errorf("spawn product home = %q, want %q", cfg.ProductHome, home)
		}
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{InstanceID: "spawned"})
		return newFakeProcess(), nil
	}}
	sup := newTestSupervisor(t, Config{ProductHome: home, Spawn: spawner.spawn})
	ctx, cancel := testContext(t)
	defer cancel()

	conn, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if fake == nil {
		t.Fatal("spawn did not start the fake daemon")
	}
	if spawner.calls.Load() != 1 {
		t.Fatalf("spawn calls = %d, want 1", spawner.calls.Load())
	}
	if !sup.OwnsDaemon() {
		t.Fatal("supervisor does not own the daemon it spawned")
	}
	if pid, ok := sup.DaemonPID(); !ok || pid != 4242 {
		t.Fatalf("DaemonPID = %d, %v; want 4242, true", pid, ok)
	}
	if conn.InstanceID() != "spawned" {
		t.Fatalf("instanceId = %q, want spawned", conn.InstanceID())
	}
	if got := fake.helloCount(); got != 1 {
		t.Fatalf("hello calls = %d, want 1", got)
	}

	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !report.Owned || !report.Pinged || report.Connections != 1 || !report.Shutdown || report.Skipped != "" {
		t.Fatalf("Close report = %+v, want an owned shutdown at one connection", report)
	}
	if got := fake.shutdownCount(); got != 1 {
		t.Fatalf("shutdown calls = %d, want 1", got)
	}
	if forces := fake.shutdownForces(); len(forces) != 1 || forces[0] {
		t.Fatalf("shutdown forces = %v, want [false]", forces)
	}
	if sup.OwnsDaemon() {
		t.Fatal("supervisor still owns the daemon after shutdown")
	}
	if _, ok := sup.DaemonPID(); ok {
		t.Fatal("supervisor still reports a daemon pid after shutdown")
	}
}

func TestHelloReject(t *testing.T) {
	cases := []struct {
		name string
		raw  json.RawMessage
	}{
		{
			name: "v2",
			raw:  json.RawMessage(`{"protocolVersion":2,"instanceId":"x","serverVersion":"v","daemonProfile":{"kind":"product"}}`),
		},
		{
			name: "empty",
			raw:  json.RawMessage(`{"protocolVersion":3,"instanceId":"","serverVersion":"v","daemonProfile":{"kind":"product"}}`),
		},
		{
			name: "profile",
			raw:  json.RawMessage(`{"protocolVersion":3,"instanceId":"x","serverVersion":"v","daemonProfile":{"kind":"configured"}}`),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := socketHome(t)
			_, fake := startFakeDaemon(t, home, fakeDaemonConfig{HelloRaw: tc.raw})
			spawner := &countingSpawner{}
			sup := newTestSupervisor(t, Config{ProductHome: home, Spawn: spawner.spawn})
			ctx, cancel := testContext(t)
			defer cancel()

			conn, err := sup.Start(ctx)
			if err == nil {
				conn.Close()
				t.Fatal("Start accepted a bad hello")
			}
			if !errors.Is(err, appserver.ErrProtocol) {
				t.Fatalf("error = %v, want a protocol error", err)
			}
			if spawner.calls.Load() != 0 {
				t.Fatalf("spawn calls = %d, want 0 after a protocol rejection", spawner.calls.Load())
			}
			if got := fake.helloCount(); got != 1 {
				t.Fatalf("hello calls = %d, want 1", got)
			}
			if _, err := sup.Close(ctx); err != nil {
				t.Fatalf("Close: %v", err)
			}
			if fake.shutdownCount() != 0 {
				t.Fatal("a rejected daemon was shut down")
			}
		})
	}
}

func TestFatalDial(t *testing.T) {
	home := socketHome(t)
	spawner := &countingSpawner{}
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Dial: func(context.Context, Endpoint) (net.Conn, error) {
			return nil, syscall.EACCES
		},
		Spawn: spawner.spawn,
	})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err == nil {
		t.Fatal("Start succeeded on a fatal dial error")
	} else if !errors.Is(err, syscall.EACCES) || IsMissingEndpoint(err) {
		t.Fatalf("error = %v, want a fatal EACCES", err)
	}
	if spawner.calls.Load() != 0 {
		t.Fatalf("spawn calls = %d, want 0 for a fatal dial error", spawner.calls.Load())
	}
}

func TestOpenStream(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "streams"})
	sup := newTestSupervisor(t, Config{ProductHome: home})
	ctx, cancel := testContext(t)
	defer cancel()

	control, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	stream, err := sup.OpenStream(ctx)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	if stream == control || stream.Client() == control.Client() {
		t.Fatal("OpenStream reused the control connection")
	}
	if got := fake.helloCount(); got != 2 {
		t.Fatalf("hello calls = %d, want one per connection", got)
	}
	if got := sup.ActiveConnections(); got != 2 {
		t.Fatalf("ActiveConnections = %d, want 2", got)
	}
	if stream.InstanceID() != control.InstanceID() {
		t.Fatalf("stream instance = %q, control instance = %q", stream.InstanceID(), control.InstanceID())
	}

	stream.Close()
	waitFor(t, "supervisor tracks one connection", func() bool { return sup.ActiveConnections() == 1 })
	waitFor(t, "fake tracks one connection", func() bool { return fake.activeConnections() == 1 })
	if _, err := sup.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestCloseNotOwned(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "external"})
	sup := newTestSupervisor(t, Config{ProductHome: home})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if report.Owned || report.Skipped != CloseSkippedNotOwned {
		t.Fatalf("report = %+v, want not_owned", report)
	}
	if fake.pingCount() != 0 {
		t.Fatalf("ping calls = %d, want 0 for an unowned daemon", fake.pingCount())
	}
	if fake.shutdownCount() != 0 {
		t.Fatalf("shutdown calls = %d, want 0", fake.shutdownCount())
	}
	waitFor(t, "connection closed", func() bool { return fake.activeConnections() == 0 })
}

func TestCloseOwned(t *testing.T) {
	home := socketHome(t)
	var fake *fakeDaemon
	spawner := &countingSpawner{start: func(cfg SpawnConfig) (Process, error) {
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{InstanceID: "owned"})
		return newFakeProcess(), nil
	}}
	sup := newTestSupervisor(t, Config{ProductHome: home, Spawn: spawner.spawn})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !report.Owned || !report.Pinged || report.Connections != 1 || !report.Shutdown {
		t.Fatalf("report = %+v, want owned/shutdown with connections=1", report)
	}
	if fake.shutdownCount() != 1 {
		t.Fatalf("shutdown calls = %d, want 1", fake.shutdownCount())
	}
}

func TestCloseOtherClients(t *testing.T) {
	home := socketHome(t)
	connections := 2
	var fake *fakeDaemon
	spawner := &countingSpawner{start: func(cfg SpawnConfig) (Process, error) {
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{
			InstanceID:      "busy",
			PingConnections: &connections,
		})
		return newFakeProcess(), nil
	}}
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Spawn:       spawner.spawn,
		CloseGrace:  CloseGrace{Attempts: 2, Interval: 5 * time.Millisecond},
	})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !report.Pinged || report.Connections != 2 || report.Shutdown {
		t.Fatalf("report = %+v, want pinged with connections=2 and no shutdown", report)
	}
	if report.Skipped != CloseSkippedOtherClients {
		t.Fatalf("Skipped = %q, want %q", report.Skipped, CloseSkippedOtherClients)
	}
	if fake.pingCount() != 2 {
		t.Fatalf("ping calls = %d, want 2 grace attempts", fake.pingCount())
	}
	if fake.shutdownCount() != 0 {
		t.Fatalf("shutdown calls = %d, want 0 while other clients exist", fake.shutdownCount())
	}
	if !sup.OwnsDaemon() {
		t.Fatal("supervisor should still own the daemon it left running")
	}
}

func TestCloseNoControl(t *testing.T) {
	home := socketHome(t)
	badHello := json.RawMessage(`{"protocolVersion":2,"instanceId":"x","serverVersion":"v","daemonProfile":{"kind":"product"}}`)
	var fake *fakeDaemon
	spawner := &countingSpawner{start: func(cfg SpawnConfig) (Process, error) {
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{HelloRaw: badHello})
		return newFakeProcess(), nil
	}}
	sup := newTestSupervisor(t, Config{ProductHome: home, Spawn: spawner.spawn})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err == nil {
		t.Fatal("Start accepted a bad hello after spawning")
	}
	if !sup.OwnsDaemon() {
		t.Fatal("supervisor lost ownership after a failed handshake")
	}
	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !report.Owned || report.Pinged || report.Shutdown || report.Skipped != CloseSkippedNotConnected {
		t.Fatalf("report = %+v, want owned/not_connected", report)
	}
	if fake.pingCount() != 0 || fake.shutdownCount() != 0 {
		t.Fatalf("daemon saw ping=%d shutdown=%d, want 0/0", fake.pingCount(), fake.shutdownCount())
	}
}

func TestClosePingError(t *testing.T) {
	home := socketHome(t)
	var fake *fakeDaemon
	spawner := &countingSpawner{start: func(cfg SpawnConfig) (Process, error) {
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{
			InstanceID: "unresponsive",
			PingErr: &appserver.RPCError{
				Code:    -32603,
				Message: "ping failed",
				Data:    json.RawMessage(`{"appError":"service_unavailable"}`),
			},
		})
		return newFakeProcess(), nil
	}}
	sup := newTestSupervisor(t, Config{ProductHome: home, Spawn: spawner.spawn})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	report, err := sup.Close(ctx)
	if err == nil {
		t.Fatal("Close succeeded without a usable daemon.ping")
	}
	var rpcErr *appserver.RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("error = %v, want the daemon RPC error", err)
	}
	if report.Shutdown || report.Pinged || !report.Owned {
		t.Fatalf("report = %+v, want owned without a ping or shutdown", report)
	}
	if fake.shutdownCount() != 0 {
		t.Fatal("daemon.shutdown was sent without a successful ping")
	}
}

func TestCloseIdempotent(t *testing.T) {
	home := socketHome(t)
	var fake *fakeDaemon
	spawner := &countingSpawner{start: func(cfg SpawnConfig) (Process, error) {
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{InstanceID: "owned"})
		return newFakeProcess(), nil
	}}
	sup := newTestSupervisor(t, Config{ProductHome: home, Spawn: spawner.spawn})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	first, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if !first.Shutdown {
		t.Fatalf("first Close = %+v, want shutdown", first)
	}
	second, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if second.Skipped != CloseSkippedAlreadyClosed || second.Shutdown {
		t.Fatalf("second Close = %+v, want already_closed", second)
	}
	if fake.shutdownCount() != 1 {
		t.Fatalf("shutdown calls = %d, want exactly 1", fake.shutdownCount())
	}
}

func TestStartAfterClose(t *testing.T) {
	home := socketHome(t)
	_, _ = startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "existing"})
	sup := newTestSupervisor(t, Config{ProductHome: home})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := sup.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := sup.Start(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("Start after Close = %v, want ErrClosed", err)
	}
	if _, err := sup.OpenStream(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("OpenStream after Close = %v, want ErrClosed", err)
	}
}

func TestOpenStreamBeforeStart(t *testing.T) {
	sup := newTestSupervisor(t, Config{ProductHome: socketHome(t)})
	if _, err := sup.OpenStream(context.Background()); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("OpenStream before Start = %v, want ErrNotStarted", err)
	}
}

func TestStartSerialized(t *testing.T) {
	home := socketHome(t)
	fakeLink, fake := startFakeDaemon(t, socketHome(t), fakeDaemonConfig{InstanceID: "serialized"})
	var ready atomic.Bool
	dial := func(ctx context.Context, _ Endpoint) (net.Conn, error) {
		if !ready.Load() {
			return nil, syscall.ENOENT
		}
		var dialer net.Dialer
		return dialer.DialContext(ctx, NetworkUnix, fakeLink.Endpoint.Address)
	}
	spawner := &countingSpawner{start: func(SpawnConfig) (Process, error) {
		ready.Store(true)
		return newFakeProcess(), nil
	}}
	sup := newTestSupervisor(t, Config{ProductHome: home, Dial: dial, Spawn: spawner.spawn})
	ctx, cancel := testContext(t)
	defer cancel()

	type result struct {
		conn *Connection
		err  error
	}
	results := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() {
			conn, err := sup.Start(ctx)
			results <- result{conn: conn, err: err}
		}()
	}
	first := <-results
	second := <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("concurrent Start errors: %v, %v", first.err, second.err)
	}
	if first.conn != second.conn {
		t.Fatal("concurrent Start calls returned different control connections")
	}
	if got := spawner.calls.Load(); got != 1 {
		t.Fatalf("spawn calls = %d, want 1", got)
	}
	if got := fake.helloCount(); got != 1 {
		t.Fatalf("hello calls = %d, want 1", got)
	}
	if _, err := sup.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func newTestSupervisor(t *testing.T, cfg Config) *Supervisor {
	t.Helper()
	sup, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return sup
}
