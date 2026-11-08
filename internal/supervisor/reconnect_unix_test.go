//go:build unix

package supervisor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

func waitConnectionClosed(t *testing.T, conn *Connection) {
	t.Helper()
	select {
	case <-conn.Done():
	case <-time.After(testWaitTimeout):
		t.Fatal("connection did not close")
	}
}

func TestReconnectSingleFlight(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})

	var dials atomic.Int32
	var armed atomic.Bool
	dialing := make(chan struct{})
	release := make(chan struct{})
	var dialingOnce sync.Once
	dial := func(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
		dials.Add(1)
		if armed.Load() {
			dialingOnce.Do(func() { close(dialing) })
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return DefaultDial(ctx, endpoint)
	}
	sup := newTestSupervisor(t, Config{ProductHome: home, Dial: dial})
	ctx, cancel := testContext(t)
	defer cancel()

	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	armed.Store(true)
	dials.Store(0)

	type result struct {
		conn *Connection
		err  error
	}
	results := make(chan result, 2)
	reconnect := func() {
		conn, err := sup.Reconnect(ctx)
		results <- result{conn: conn, err: err}
	}

	go reconnect()
	select {
	case <-dialing:
	case <-time.After(testWaitTimeout):
		t.Fatal("first reconnect never dialed")
	}
	go reconnect()
	waitFor(t, "second reconnect to join the in-flight call", func() bool {
		sup.mu.Lock()
		defer sup.mu.Unlock()
		return sup.reconnecting != nil && sup.reconnecting.waiters == 1
	})
	close(release)

	first := <-results
	second := <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("reconnect errors: %v, %v", first.err, second.err)
	}
	if first.conn != second.conn {
		t.Fatal("concurrent Reconnect calls did not share one connection")
	}
	if got := dials.Load(); got != 1 {
		t.Fatalf("reconnect dials = %d, want 1", got)
	}
	if got := fake.helloCount(); got != 2 {
		t.Fatalf("hello calls = %d, want 2 (start + one reconnect)", got)
	}
	if _, err := sup.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestReconnectAfterTransportLoss(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
	sup := newTestSupervisor(t, Config{ProductHome: home})
	ctx, cancel := testContext(t)
	defer cancel()

	old, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	fake.dropConnections()
	waitConnectionClosed(t, old)
	if !old.Closed() {
		t.Fatal("Closed() is false after Done fired")
	}
	if err := old.Err(); err == nil {
		t.Fatal("Err() is nil after the transport was lost")
	} else if errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("Err() = %v, want the transport error, not a local close", err)
	} else if !errors.Is(err, io.EOF) {
		t.Fatalf("Err() = %v, want io.EOF from the closed daemon", err)
	}

	conn, err := sup.Reconnect(ctx)
	if err != nil {
		t.Fatalf("Reconnect: %v", err)
	}
	if conn == old {
		t.Fatal("Reconnect reused the lost control connection")
	}
	if !old.Closed() {
		t.Fatal("Reconnect left the lost control connection open")
	}
	if got := sup.ActiveConnections(); got != 1 {
		t.Fatalf("ActiveConnections = %d, want 1 after reconnect", got)
	}
	if conn.InstanceID() != "one" {
		t.Fatalf("instanceId = %q, want one", conn.InstanceID())
	}
	if conn.Restarted() {
		t.Fatal("same instanceId was reported as a restart")
	}
	if conn.Generation() != 1 || sup.Generation() != 1 {
		t.Fatalf("generation = %d (supervisor %d), want 1", conn.Generation(), sup.Generation())
	}
	if old.Stale() {
		t.Fatal("an old connection with the same generation is not stale")
	}
	if got := fake.helloCount(); got != 2 {
		t.Fatalf("hello calls = %d, want 2", got)
	}
	if _, err := sup.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestReconnectRestartGeneration(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
	sup := newTestSupervisor(t, Config{ProductHome: home})
	ctx, cancel := testContext(t)
	defer cancel()

	old, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if old.Generation() != 1 || old.Restarted() || old.PreviousInstanceID() != "" {
		t.Fatalf("first hello metadata = gen %d restarted %v previous %q", old.Generation(), old.Restarted(), old.PreviousInstanceID())
	}
	if sup.Generation() != 1 || sup.InstanceID() != "one" {
		t.Fatalf("supervisor instance = %q generation %d", sup.InstanceID(), sup.Generation())
	}
	stream, err := sup.OpenStream(ctx)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}

	fake.setInstanceID("two")
	result, err := sup.ReconnectResult(ctx)
	if err != nil {
		t.Fatalf("ReconnectResult: %v", err)
	}
	if !stream.Closed() || !errors.Is(stream.Err(), ErrConnectionClosed) {
		t.Fatalf("old stream done=%v err=%v, want a local close", stream.Closed(), stream.Err())
	}
	if got := sup.ActiveConnections(); got != 1 {
		t.Fatalf("ActiveConnections = %d, want only the new control", got)
	}
	if result.Connection == nil || result.NewInstanceID != "two" || result.PreviousInstanceID != "one" {
		t.Fatalf("result = %+v, want one -> two", result)
	}
	if !result.Restarted || result.Generation != 2 {
		t.Fatalf("result restarted=%v generation=%d, want true/2", result.Restarted, result.Generation)
	}
	if result.Connection.Generation() != 2 || !result.Connection.Restarted() {
		t.Fatal("connection metadata does not match ReconnectResult")
	}
	if sup.Generation() != 2 || sup.InstanceID() != "two" {
		t.Fatalf("supervisor instance = %q generation %d, want two/2", sup.InstanceID(), sup.Generation())
	}
	if !old.Stale() {
		t.Fatal("the previous generation connection is not stale")
	}

	same, err := sup.Reconnect(ctx)
	if err != nil {
		t.Fatalf("second Reconnect: %v", err)
	}
	if same.Restarted() {
		t.Fatal("same instanceId after reconnect was reported as a restart")
	}
	if same.Generation() != 2 || sup.Generation() != 2 {
		t.Fatalf("generation = %d (supervisor %d), want 2", same.Generation(), sup.Generation())
	}
	if same.PreviousInstanceID() != "two" {
		t.Fatalf("previous instance = %q, want two", same.PreviousInstanceID())
	}
	if _, err := sup.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestReconnectRejectsClosed(t *testing.T) {
	home := socketHome(t)
	_, _ = startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
	sup := newTestSupervisor(t, Config{ProductHome: home})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := sup.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := sup.Reconnect(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("Reconnect after Close = %v, want ErrClosed", err)
	}
	if _, err := sup.ReconnectResult(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("ReconnectResult after Close = %v, want ErrClosed", err)
	}
}

func TestReconnectBeforeStart(t *testing.T) {
	sup := newTestSupervisor(t, Config{ProductHome: socketHome(t)})
	if _, err := sup.Reconnect(context.Background()); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("Reconnect before Start = %v, want ErrNotStarted", err)
	}
}

func TestReconnectBackoffBounded(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
	var dials atomic.Int32
	spawner := &countingSpawner{}
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Dial: func(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
			dials.Add(1)
			return DefaultDial(ctx, endpoint)
		},
		Spawn:            spawner.spawn,
		ReconnectBackoff: ReconnectBackoff{Attempts: 4, Initial: 2 * time.Millisecond, Max: 5 * time.Millisecond},
	})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	fake.close()
	dials.Store(0)

	start := time.Now()
	_, err := sup.Reconnect(ctx)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Reconnect succeeded after the daemon disappeared")
	}
	if !IsMissingEndpoint(err) {
		t.Fatalf("error = %v, want a missing-endpoint error", err)
	}
	if got := dials.Load(); got != 4 {
		t.Fatalf("reconnect dials = %d, want the 4-attempt bound", got)
	}
	if elapsed > time.Second {
		t.Fatalf("backoff took %s, want a bounded wait", elapsed)
	}
	if spawner.calls.Load() != 0 {
		t.Fatalf("spawn calls = %d, want 0 for a daemon we do not own", spawner.calls.Load())
	}
}

func TestReconnectBackoffCancel(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
	var dials atomic.Int32
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Dial: func(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
			dials.Add(1)
			return DefaultDial(ctx, endpoint)
		},
		ReconnectBackoff: ReconnectBackoff{Attempts: 1000, Initial: 20 * time.Millisecond, Max: 20 * time.Millisecond},
	})
	if _, err := sup.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	fake.close()
	dials.Store(0)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := sup.Reconnect(ctx)
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	if got := dials.Load(); got >= 1000 {
		t.Fatalf("reconnect dials = %d, want cancellation before the attempt bound", got)
	}
	if elapsed > time.Second {
		t.Fatalf("cancel took %s, want a bounded wait", elapsed)
	}
}

func TestReconnectNoSpawnOnFatal(t *testing.T) {
	home := socketHome(t)
	startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
	var armed atomic.Bool
	var dials atomic.Int32
	spawner := &countingSpawner{}
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Dial: func(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
			dials.Add(1)
			if armed.Load() {
				return nil, syscall.EACCES
			}
			return DefaultDial(ctx, endpoint)
		},
		Spawn: spawner.spawn,
	})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	armed.Store(true)
	dials.Store(0)

	_, err := sup.Reconnect(ctx)
	if !errors.Is(err, syscall.EACCES) {
		t.Fatalf("error = %v, want EACCES", err)
	}
	if IsMissingEndpoint(err) {
		t.Fatal("a fatal dial error was classified as missing")
	}
	if got := dials.Load(); got != 1 {
		t.Fatalf("dials = %d, want exactly one non-retried attempt", got)
	}
	if spawner.calls.Load() != 0 {
		t.Fatalf("spawn calls = %d, want 0 on a non-missing error", spawner.calls.Load())
	}
}

func TestReconnectRespawnsOwned(t *testing.T) {
	home := socketHome(t)
	var fake *fakeDaemon
	spawner := &countingSpawner{}
	spawner.start = func(cfg SpawnConfig) (Process, error) {
		fake = newFakeDaemon(t, cfg.Endpoint, fakeDaemonConfig{
			InstanceID: fmt.Sprintf("spawn-%d", spawner.calls.Load()),
		})
		return newFakeProcess(), nil
	}
	sup := newTestSupervisor(t, Config{ProductHome: home, Spawn: spawner.spawn})
	ctx, cancel := testContext(t)
	defer cancel()

	first, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if first.InstanceID() != "spawn-1" {
		t.Fatalf("first instance = %q, want spawn-1", first.InstanceID())
	}
	fake.close()
	waitConnectionClosed(t, first)

	reconnected, err := sup.Reconnect(ctx)
	if err != nil {
		t.Fatalf("Reconnect: %v", err)
	}
	if got := spawner.calls.Load(); got != 2 {
		t.Fatalf("spawn calls = %d, want 2 (initial + owned respawn)", got)
	}
	if reconnected.InstanceID() != "spawn-2" {
		t.Fatalf("instance = %q, want spawn-2", reconnected.InstanceID())
	}
	if !sup.OwnsDaemon() {
		t.Fatal("supervisor lost ownership after respawn")
	}
	if _, err := sup.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if fake.shutdownCount() != 1 {
		t.Fatalf("shutdown calls = %d, want 1", fake.shutdownCount())
	}
}

func TestReconnectHelloReject(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
	spawner := &countingSpawner{}
	sup := newTestSupervisor(t, Config{ProductHome: home, Spawn: spawner.spawn})
	ctx, cancel := testContext(t)
	defer cancel()
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	fake.setHelloRaw(json.RawMessage(`{"protocolVersion":2,"instanceId":"x","serverVersion":"v","daemonProfile":{"kind":"product"}}`))
	if _, err := sup.Reconnect(ctx); !errors.Is(err, appserver.ErrProtocol) {
		t.Fatalf("Reconnect = %v, want a protocol error", err)
	}
	if spawner.calls.Load() != 0 {
		t.Fatalf("spawn calls = %d, want 0 for a hello rejection", spawner.calls.Load())
	}
}

func TestOpenStreamRestartGeneration(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
	sup := newTestSupervisor(t, Config{ProductHome: home})
	ctx, cancel := testContext(t)
	defer cancel()
	control, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	fake.setInstanceID("two")
	stream, err := sup.OpenStream(ctx)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	if !stream.Restarted() || stream.PreviousInstanceID() != "one" || stream.Generation() != 2 {
		t.Fatalf("stream metadata = restarted %v previous %q generation %d", stream.Restarted(), stream.PreviousInstanceID(), stream.Generation())
	}
	if sup.Generation() != 2 || sup.InstanceID() != "two" {
		t.Fatalf("supervisor instance = %q generation %d", sup.InstanceID(), sup.Generation())
	}
	if !control.Stale() {
		t.Fatal("the control connection from the old generation is not stale")
	}
}

func TestReconnectCloseInFlight(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
	var dials atomic.Int32
	sup := newTestSupervisor(t, Config{
		ProductHome: home,
		Dial: func(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
			dials.Add(1)
			return DefaultDial(ctx, endpoint)
		},
		ReconnectBackoff: ReconnectBackoff{Attempts: 1000, Initial: 100 * time.Millisecond, Max: 100 * time.Millisecond},
	})
	if _, err := sup.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	fake.close()

	reconnectErr := make(chan error, 1)
	go func() {
		_, err := sup.Reconnect(context.Background())
		reconnectErr <- err
	}()
	waitFor(t, "reconnect to start dialing", func() bool { return dials.Load() > 1 })

	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	closeReport := make(chan CloseReport, 1)
	go func() {
		report, _ := sup.Close(closeCtx)
		closeReport <- report
	}()
	select {
	case err := <-reconnectErr:
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("in-flight Reconnect = %v, want ErrClosed after Close", err)
		}
	case <-time.After(testWaitTimeout):
		t.Fatal("in-flight Reconnect did not abort after Close")
	}
	select {
	case <-closeReport:
	case <-time.After(testWaitTimeout):
		t.Fatal("Close did not finish after the reconnect aborted")
	}
	if _, err := sup.Reconnect(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatalf("Reconnect after Close = %v, want ErrClosed", err)
	}
}

func TestConnectionLocalCloseReason(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
	sup := newTestSupervisor(t, Config{ProductHome: home})
	ctx, cancel := testContext(t)
	defer cancel()
	conn, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if conn.Closed() {
		t.Fatal("fresh connection reports Closed")
	}
	if err := conn.Err(); err != nil {
		t.Fatalf("fresh connection Err = %v, want nil", err)
	}
	conn.Close()
	waitConnectionClosed(t, conn)
	if !errors.Is(conn.Err(), ErrConnectionClosed) {
		t.Fatalf("Err = %v, want ErrConnectionClosed", conn.Err())
	}
	waitFor(t, "fake connection closed", func() bool { return fake.activeConnections() == 0 })
}
