//go:build unix

package supervisor

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

func missingDial(context.Context, Endpoint) (net.Conn, error) { return nil, syscall.ENOENT }

// TestSpawnFailureDoesNotClaimOwnership pins that a failed spawn surfaces the
// spawn error, leaves the supervisor unowned, and does not latch: the next
// Start retries the spawn instead of treating the daemon as started.
func TestSpawnFailureDoesNotClaimOwnership(t *testing.T) {
	home := socketHome(t)
	spawnErr := errors.New("spawn exploded")
	var spawns atomic.Int32
	spawn := func(context.Context, SpawnConfig) (Process, error) {
		spawns.Add(1)
		return nil, spawnErr
	}
	sup := newTestSupervisor(t, Config{ProductHome: home, Dial: missingDial, Spawn: spawn})
	ctx, cancel := testContext(t)
	defer cancel()

	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := sup.Start(ctx); err == nil {
			t.Fatalf("Start attempt %d succeeded", attempt)
		} else if !errors.Is(err, spawnErr) || !strings.Contains(err.Error(), "spawn app server") {
			t.Fatalf("Start attempt %d error = %v, want a wrapped spawn error", attempt, err)
		}
		if sup.OwnsDaemon() {
			t.Fatalf("attempt %d: supervisor owns a daemon whose spawn failed", attempt)
		}
		if _, ok := sup.DaemonPID(); ok {
			t.Fatalf("attempt %d: supervisor reported a pid for a failed spawn", attempt)
		}
	}
	if got := spawns.Load(); got != 2 {
		t.Fatalf("spawn calls = %d, want one per Start attempt", got)
	}

	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if report.Owned || report.Pinged || report.Shutdown || report.Skipped != CloseSkippedNotOwned {
		t.Fatalf("Close report = %+v, want a plain not_owned report", report)
	}
}

// TestSpawnNeverListens pins the owned-daemon behavior when the child never
// opens its endpoint: the wait budget surfaces ErrListenTimeout, ownership is
// already recorded, and Close has no control connection to stop it with.
func TestSpawnNeverListens(t *testing.T) {
	home := socketHome(t)
	spawn := func(context.Context, SpawnConfig) (Process, error) { return newFakeProcess(), nil }
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Dial:        missingDial,
		Spawn:       spawn,
		ListenWait:  ListenWait{Timeout: 30 * time.Millisecond, Interval: 2 * time.Millisecond},
	})
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Now()
	_, err := sup.Start(ctx)
	if !errors.Is(err, ErrListenTimeout) {
		t.Fatalf("Start error = %v, want ErrListenTimeout", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("listen timeout took %s, want a bounded wait", elapsed)
	}
	if !sup.OwnsDaemon() {
		t.Fatal("supervisor lost ownership of the daemon it spawned")
	}
	if pid, ok := sup.DaemonPID(); !ok || pid != 4242 {
		t.Fatalf("DaemonPID = %d, %v; want 4242, true", pid, ok)
	}

	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !report.Owned || report.Pinged || report.Shutdown || report.Skipped != CloseSkippedNotConnected {
		t.Fatalf("Close report = %+v, want owned/not_connected", report)
	}
}

// TestStartDoesNotSpawnAgainWhileOwnedDaemonIsNotListening pins that a second
// Start after a successful spawn never spawns a duplicate daemon.
func TestStartDoesNotSpawnAgainWhileOwnedDaemonIsNotListening(t *testing.T) {
	home := socketHome(t)
	var spawns atomic.Int32
	spawn := func(context.Context, SpawnConfig) (Process, error) {
		spawns.Add(1)
		return newFakeProcess(), nil
	}
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Dial:        missingDial,
		Spawn:       spawn,
		ListenWait:  ListenWait{Timeout: 20 * time.Millisecond, Interval: time.Millisecond},
	})
	ctx, cancel := testContext(t)
	defer cancel()

	if _, err := sup.Start(ctx); !errors.Is(err, ErrListenTimeout) {
		t.Fatalf("first Start error = %v, want ErrListenTimeout", err)
	}
	if _, err := sup.Start(ctx); err == nil {
		t.Fatal("second Start succeeded while the owned daemon was not listening")
	} else {
		if !strings.Contains(err.Error(), "is not listening") {
			t.Fatalf("second Start error = %v, want an owned-daemon-not-listening error", err)
		}
		if !errors.Is(err, syscall.ENOENT) {
			t.Fatalf("second Start error = %v, want the underlying dial error", err)
		}
	}
	if got := spawns.Load(); got != 1 {
		t.Fatalf("spawn calls = %d, want 1 for two Start attempts", got)
	}
	if _, err := sup.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestCloseClosesStreamsBeforePing pins that streams opened through
// OpenStream are torn down before daemon.ping decides ownership, so they never
// count as other clients and the owned daemon is still shut down.
func TestCloseClosesStreamsBeforePing(t *testing.T) {
	home := socketHome(t)
	var fake *fakeDaemon
	spawner := &countingSpawner{start: func(cfg SpawnConfig) (Process, error) {
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{InstanceID: "streamed"})
		return newFakeProcess(), nil
	}}
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Spawn:       spawner.spawn,
		CloseGrace:  CloseGrace{Attempts: 100, Interval: 5 * time.Millisecond},
	})
	ctx, cancel := testContext(t)
	defer cancel()

	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	stream, err := sup.OpenStream(ctx)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	// Leave the stream open: Close owns tearing it down before the ping.
	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !report.Owned || !report.Pinged || report.Connections != 1 || !report.Shutdown {
		t.Fatalf("Close report = %+v, want an owned shutdown at one connection", report)
	}
	if got := fake.shutdownCount(); got != 1 {
		t.Fatalf("shutdown calls = %d, want 1", got)
	}
	if got := sup.ActiveConnections(); got != 0 {
		t.Fatalf("ActiveConnections = %d, want 0 after Close", got)
	}
	stream.Close() // idempotent after Close tore the stream down
}

// TestCloseRetriesPingUntilSingleConnection pins the grace loop: Close keeps
// pinging while other clients disconnect and shuts down once the count drops to
// one.
func TestCloseRetriesPingUntilSingleConnection(t *testing.T) {
	home := socketHome(t)
	var fake *fakeDaemon
	spawner := &countingSpawner{start: func(cfg SpawnConfig) (Process, error) {
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{
			InstanceID:         "draining",
			PingConnectionsSeq: []int{2, 2, 1},
		})
		return newFakeProcess(), nil
	}}
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Spawn:       spawner.spawn,
		CloseGrace:  CloseGrace{Attempts: 5, Interval: time.Millisecond},
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
	if !report.Owned || !report.Pinged || report.Connections != 1 || !report.Shutdown || report.Skipped != "" {
		t.Fatalf("Close report = %+v, want an owned shutdown after draining", report)
	}
	if got := fake.pingCount(); got != 3 {
		t.Fatalf("ping calls = %d, want 3", got)
	}
	if got := fake.shutdownCount(); got != 1 {
		t.Fatalf("shutdown calls = %d, want 1", got)
	}
}

// TestCloseShutdownError pins that a rejected daemon.shutdown is surfaced
// while ownership stays with the still-running daemon and Close stays
// idempotent.
func TestCloseShutdownError(t *testing.T) {
	home := socketHome(t)
	shutdownErr := &appserver.RPCError{
		Code:    -32603,
		Message: "shutdown failed",
		Data:    json.RawMessage(`{"appError":"service_unavailable"}`),
	}
	var fake *fakeDaemon
	spawner := &countingSpawner{start: func(cfg SpawnConfig) (Process, error) {
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{InstanceID: "stuck", ShutdownErr: shutdownErr})
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
		t.Fatal("Close succeeded with a rejected daemon.shutdown")
	}
	var rpcErr *appserver.RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("error = %v, want the daemon RPC error", err)
	}
	if rpcErr.Code != shutdownErr.Code || rpcErr.Message != shutdownErr.Message {
		t.Fatalf("RPC error = %+v, want %+v", rpcErr, shutdownErr)
	}
	if !report.Owned || !report.Pinged || report.Connections != 1 || report.Shutdown || report.Skipped != "" {
		t.Fatalf("Close report = %+v, want owned/pinged without a shutdown", report)
	}
	if got := fake.shutdownCount(); got != 1 {
		t.Fatalf("shutdown calls = %d, want 1", got)
	}
	// Ownership is only cleared after a confirmed shutdown.
	if !sup.OwnsDaemon() {
		t.Fatal("ownership was cleared although the daemon was not stopped")
	}
	if _, ok := sup.DaemonPID(); !ok {
		t.Fatal("daemon pid was cleared although the daemon was not stopped")
	}
	second, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if second.Skipped != CloseSkippedAlreadyClosed {
		t.Fatalf("second Close = %+v, want already_closed", second)
	}
	if got := fake.shutdownCount(); got != 1 {
		t.Fatalf("shutdown calls = %d after second Close, want 1", got)
	}
}

// TestCloseAfterControlClosed pins that closing the control connection detaches
// the only channel Close could ping and shut down through: the owned daemon is
// left running and no daemon calls are attempted.
func TestCloseAfterControlClosed(t *testing.T) {
	home := socketHome(t)
	var fake *fakeDaemon
	spawner := &countingSpawner{start: func(cfg SpawnConfig) (Process, error) {
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{InstanceID: "detached"})
		return newFakeProcess(), nil
	}}
	sup := newTestSupervisor(t, Config{ProductHome: home, Spawn: spawner.spawn})
	ctx, cancel := testContext(t)
	defer cancel()

	control, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	control.Close()
	waitFor(t, "daemon saw the control connection close", func() bool { return fake.activeConnections() == 0 })

	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !report.Owned || report.Pinged || report.Shutdown || report.Skipped != CloseSkippedNotConnected {
		t.Fatalf("Close report = %+v, want owned/not_connected", report)
	}
	if fake.pingCount() != 0 || fake.shutdownCount() != 0 {
		t.Fatalf("daemon saw ping=%d shutdown=%d, want 0/0", fake.pingCount(), fake.shutdownCount())
	}
	if !sup.OwnsDaemon() {
		t.Fatal("ownership should remain until a confirmed shutdown")
	}
}

// TestCloseBeforeStart pins that Close on a fresh supervisor reports not_owned
// without dialing, spawning, or preventing the later (now refused) Start.
func TestCloseBeforeStart(t *testing.T) {
	home := socketHome(t)
	var dials, spawns atomic.Int32
	dial := func(context.Context, Endpoint) (net.Conn, error) {
		dials.Add(1)
		return nil, syscall.ENOENT
	}
	spawn := func(context.Context, SpawnConfig) (Process, error) {
		spawns.Add(1)
		return newFakeProcess(), nil
	}
	sup := newTestSupervisor(t, Config{ProductHome: home, Dial: dial, Spawn: spawn})
	ctx, cancel := testContext(t)
	defer cancel()

	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if report.Owned || report.Pinged || report.Shutdown || report.Skipped != CloseSkippedNotOwned {
		t.Fatalf("Close report = %+v, want a plain not_owned report", report)
	}
	if dials.Load() != 0 || spawns.Load() != 0 {
		t.Fatalf("Close dialed %d and spawned %d, want 0/0", dials.Load(), spawns.Load())
	}
	if _, err := sup.Start(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("Start after Close = %v, want ErrClosed", err)
	}
}

// TestCloseGraceHonorsContext pins that a canceled Close deadline stops the
// ping grace loop, still marks the supervisor closed, and never shuts down a
// daemon while other clients are reported.
func TestCloseGraceHonorsContext(t *testing.T) {
	home := socketHome(t)
	connections := 2
	var fake *fakeDaemon
	spawner := &countingSpawner{start: func(cfg SpawnConfig) (Process, error) {
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{InstanceID: "busy", PingConnections: &connections})
		return newFakeProcess(), nil
	}}
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Spawn:       spawner.spawn,
		CloseGrace:  CloseGrace{Attempts: 20, Interval: time.Hour},
	})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer closeCancel()
	start := time.Now()
	report, err := sup.Close(closeCtx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close error = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("Close took %s, want the caller deadline", elapsed)
	}
	if !report.Owned || report.Shutdown {
		t.Fatalf("Close report = %+v, want owned without a shutdown", report)
	}
	if got := fake.shutdownCount(); got != 0 {
		t.Fatalf("shutdown calls = %d, want 0 while other clients exist", got)
	}
	// finishClose already ran, so a retry is a no-op.
	second, err := sup.Close(context.Background())
	if err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if second.Skipped != CloseSkippedAlreadyClosed {
		t.Fatalf("second Close = %+v, want already_closed", second)
	}
}

// TestListenFatalErrorAfterSpawn pins that a non-missing connect failure while
// waiting for a freshly spawned daemon is surfaced immediately instead of being
// retried or reported as a listen timeout.
func TestListenFatalErrorAfterSpawn(t *testing.T) {
	home := socketHome(t)
	var attempts atomic.Int32
	dial := func(context.Context, Endpoint) (net.Conn, error) {
		if attempts.Add(1) == 1 {
			return nil, syscall.ENOENT
		}
		return nil, syscall.EACCES
	}
	spawner := &countingSpawner{start: func(SpawnConfig) (Process, error) { return newFakeProcess(), nil }}
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Dial:        dial,
		Spawn:       spawner.spawn,
		ListenWait:  ListenWait{Timeout: time.Second, Interval: time.Millisecond},
	})
	ctx, cancel := testContext(t)
	defer cancel()

	_, err := sup.Start(ctx)
	if err == nil {
		t.Fatal("Start succeeded on a fatal listen error")
	}
	if errors.Is(err, ErrListenTimeout) {
		t.Fatalf("error = %v, want a fatal error, not a listen timeout", err)
	}
	if !errors.Is(err, syscall.EACCES) {
		t.Fatalf("error = %v, want EACCES", err)
	}
	if !strings.Contains(err.Error(), "waiting for") {
		t.Fatalf("error = %v, want the wait-for-listen context", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("dial attempts = %d, want no retry after a fatal error", got)
	}
	if got := spawner.calls.Load(); got != 1 {
		t.Fatalf("spawn calls = %d, want 1", got)
	}
	if !sup.OwnsDaemon() {
		t.Fatal("supervisor lost ownership of the daemon it spawned")
	}

	report, err := sup.Close(ctx)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !report.Owned || report.Pinged || report.Skipped != CloseSkippedNotConnected {
		t.Fatalf("Close report = %+v, want owned/not_connected", report)
	}
}
