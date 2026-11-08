//go:build unix

package supervisor

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

func testHello() appserver.HelloParams {
	return appserver.HelloParams{
		ProtocolVersion: appserver.WireProtocolVersion,
		ClientKind:      appserver.ClientKindOther,
		ClientName:      DefaultClientName,
	}
}

func TestDialHello(t *testing.T) {
	home := socketHome(t)
	link, fake := startFakeDaemon(t, home, fakeDaemonConfig{InstanceID: "dial-target"})
	ctx, cancel := testContext(t)
	defer cancel()
	client, hello, err := Dial(ctx, link.Endpoint, testHello())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()
	if hello.InstanceID != "dial-target" {
		t.Fatalf("instanceId = %q, want dial-target", hello.InstanceID)
	}
	if !hello.HasFeature("session.catalog") {
		t.Fatalf("features = %v, want session.catalog", hello.Features)
	}
	requests := fake.requestsFor("hello")
	if len(requests) != 1 {
		t.Fatalf("hello requests = %d, want 1", len(requests))
	}
	var params appserver.HelloParams
	if err := json.Unmarshal(requests[0].Params, &params); err != nil {
		t.Fatalf("decode hello params: %v", err)
	}
	if params != testHello() {
		t.Fatalf("hello params = %+v, want %+v", params, testHello())
	}
	client.Close()
	waitFor(t, "fake connection closed", func() bool { return fake.activeConnections() == 0 })
}

func TestDialBadHello(t *testing.T) {
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
			name: "bad",
			raw:  json.RawMessage(`{"protocolVersion":3,"instanceId":"x","serverVersion":"v","daemonProfile":{"kind":"configured"}}`),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := socketHome(t)
			link, fake := startFakeDaemon(t, home, fakeDaemonConfig{HelloRaw: tc.raw})
			ctx, cancel := testContext(t)
			defer cancel()
			client, _, err := Dial(ctx, link.Endpoint, testHello())
			if err == nil {
				client.Close()
				t.Fatal("Dial accepted a bad hello result")
			}
			if client != nil {
				t.Fatal("Dial returned a client on failure")
			}
			if !errors.Is(err, appserver.ErrProtocol) {
				t.Fatalf("error = %v, want a protocol error", err)
			}
			if fake.helloCount() != 1 {
				t.Fatalf("hello calls = %d, want 1", fake.helloCount())
			}
			waitFor(t, "rejected connection closed", func() bool { return fake.activeConnections() == 0 })
		})
	}
}

func TestDialClassifiesErrors(t *testing.T) {
	dir := socketHome(t)

	refusedPath := filepath.Join(dir, "refused.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: refusedPath, Net: "unix"})
	if err != nil {
		t.Fatalf("listen refused socket: %v", err)
	}
	listener.SetUnlinkOnClose(false)
	if err := listener.Close(); err != nil {
		t.Fatalf("close refused listener: %v", err)
	}

	regularPath := filepath.Join(dir, "regular")
	if err := os.WriteFile(regularPath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write regular file: %v", err)
	}

	privateDir := filepath.Join(dir, "private")
	if err := os.Mkdir(privateDir, 0o700); err != nil {
		t.Fatalf("mkdir private: %v", err)
	}
	if err := os.Chmod(privateDir, 0); err != nil {
		t.Fatalf("chmod private: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(privateDir, 0o700) })

	cases := []struct {
		name    string
		address string
		absent  bool
		wantErr error
	}{
		{name: "missing", address: filepath.Join(dir, "missing.sock"), absent: true, wantErr: syscall.ENOENT},
		{name: "refused", address: refusedPath, absent: true, wantErr: syscall.ECONNREFUSED},
		{name: "regular file", address: regularPath, absent: false},
		{name: "permission", address: filepath.Join(privateDir, "daemon.sock"), absent: false, wantErr: syscall.EACCES},
	}
	if os.Geteuid() == 0 {
		cases = cases[:3]
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := testContext(t)
			defer cancel()
			client, _, err := Dial(ctx, Endpoint{Network: NetworkUnix, Address: tc.address}, testHello())
			if err == nil {
				client.Close()
				t.Fatal("Dial succeeded with no daemon")
			}
			if client != nil {
				t.Fatal("Dial returned a client on failure")
			}
			if got := IsMissingEndpoint(err); got != tc.absent {
				t.Fatalf("IsMissingEndpoint(%v) = %v, want %v", err, got, tc.absent)
			}
			var dialErr *DialError
			if !errors.As(err, &dialErr) {
				t.Fatalf("error %v is not a *DialError", err)
			}
			if dialErr.Absent != tc.absent {
				t.Fatalf("DialError.Absent = %v, want %v", dialErr.Absent, tc.absent)
			}
			var opErr *net.OpError
			if !errors.As(err, &opErr) {
				t.Fatalf("error %v does not wrap *net.OpError", err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("error %v does not wrap %v", err, tc.wantErr)
			}
		})
	}
}

func TestDialValidatesHelloBeforeConnecting(t *testing.T) {
	var attempts int
	dial := func(context.Context, Endpoint) (net.Conn, error) {
		attempts++
		return nil, syscall.ENOENT
	}
	_, _, err := DialWith(context.Background(), Endpoint{Network: NetworkUnix, Address: "/tmp/x.sock"}, appserver.HelloParams{
		ProtocolVersion: 2,
		ClientKind:      appserver.ClientKindOther,
		ClientName:      "x",
	}, dial)
	if !errors.Is(err, appserver.ErrProtocol) {
		t.Fatalf("error = %v, want a protocol error", err)
	}
	if attempts != 0 {
		t.Fatalf("dial attempts = %d, want 0", attempts)
	}
}

func TestDialHonorsContextDeadline(t *testing.T) {
	dial := func(ctx context.Context, _ Endpoint) (net.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, _, err := DialWith(ctx, Endpoint{Network: NetworkUnix, Address: "/tmp/slow.sock"}, testHello(), dial)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	if IsMissingEndpoint(err) {
		t.Fatal("a connect timeout was classified as daemon absent")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("dial took %s, want the caller deadline", elapsed)
	}
}
