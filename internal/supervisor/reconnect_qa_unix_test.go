//go:build unix

package supervisor

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// TestReconnectWaiterDeadline pins the single-flight waiter contract: a waiter
// that joins an in-flight reconnect gets its own context deadline, while the
// leader still completes and only one dial happens.
func TestReconnectWaiterDeadline(t *testing.T) {
	home := socketHome(t)
	_, _ = startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
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

	leader := make(chan error, 1)
	go func() {
		_, err := sup.Reconnect(ctx)
		leader <- err
	}()
	select {
	case <-dialing:
	case <-time.After(testWaitTimeout):
		t.Fatal("leader reconnect never dialed")
	}

	waiterCtx, waiterCancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer waiterCancel()
	start := time.Now()
	if _, err := sup.Reconnect(waiterCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waiter Reconnect = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("waiter waited %s, want its own deadline", elapsed)
	}

	close(release)
	if err := <-leader; err != nil {
		t.Fatalf("leader Reconnect: %v", err)
	}
	if got := dials.Load(); got != 1 {
		t.Fatalf("reconnect dials = %d, want 1", got)
	}
	if sup.Generation() != 1 {
		t.Fatalf("generation = %d, want 1 for an unchanged instanceId", sup.Generation())
	}
	if _, err := sup.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestReconnectFailNoGeneration pins that generation changes only after a
// successful hello: a rejected reconnect leaves the observed instance and
// generation untouched, and a later valid reconnect stays on the same
// generation.
func TestReconnectFailNoGeneration(t *testing.T) {
	home := socketHome(t)
	_, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "one"})
	spawner := &countingSpawner{}
	sup := newTestSupervisor(t, Config{ProductHome: home, Spawn: spawner.spawn})
	ctx, cancel := testContext(t)
	defer cancel()

	old, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	fake.setHelloRaw(json.RawMessage(`{"protocolVersion":2,"instanceId":"x","serverVersion":"v","daemonProfile":{"kind":"product"}}`))
	if _, err := sup.Reconnect(ctx); !errors.Is(err, appserver.ErrProtocol) {
		t.Fatalf("Reconnect = %v, want a protocol error", err)
	}
	if sup.Generation() != 1 || sup.InstanceID() != "one" {
		t.Fatalf("after failed reconnect: instance %q generation %d, want one/1", sup.InstanceID(), sup.Generation())
	}
	if !old.Closed() {
		t.Fatal("Reconnect left the old control connection open")
	}
	if spawner.calls.Load() != 0 {
		t.Fatalf("spawn calls = %d, want 0 for a hello rejection", spawner.calls.Load())
	}

	fake.setHelloRaw(nil)
	conn, err := sup.Reconnect(ctx)
	if err != nil {
		t.Fatalf("second Reconnect: %v", err)
	}
	if conn.Restarted() || conn.Generation() != 1 || sup.Generation() != 1 {
		t.Fatalf("connection restarted=%v generation=%d (supervisor %d), want false/1/1", conn.Restarted(), conn.Generation(), sup.Generation())
	}
	if _, err := sup.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
