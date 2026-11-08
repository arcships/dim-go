package supervisor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

func platformTestHello() appserver.HelloParams {
	return appserver.HelloParams{
		ProtocolVersion: appserver.WireProtocolVersion,
		ClientKind:      appserver.ClientKindOther,
		ClientName:      DefaultClientName,
	}
}

// TestDialHelloTimeout pins the failure semantics for a peer that accepts the
// transport but never answers hello: the caller deadline wins, the error stays
// fatal (never "daemon absent"), no live client escapes, and the transport is
// closed so the peer read loop ends.
func TestDialHelloTimeout(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	served := make(chan struct{})
	go func() {
		defer close(served)
		_, _ = io.Copy(io.Discard, serverConn)
	}()
	t.Cleanup(func() { _ = serverConn.Close() })

	dial := func(context.Context, Endpoint) (net.Conn, error) { return clientConn, nil }
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	client, _, err := DialWith(ctx, Endpoint{Network: NetworkUnix, Address: "/tmp/hello-timeout.sock"}, platformTestHello(), dial)
	elapsed := time.Since(start)
	if err == nil {
		client.Close()
		t.Fatal("DialWith succeeded without a hello reply")
	}
	if client != nil {
		t.Fatal("DialWith returned a client after a hello timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	if IsMissingEndpoint(err) {
		t.Fatal("a hello timeout was classified as a missing daemon")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("hello timeout took %s, want the caller deadline", elapsed)
	}
	select {
	case <-served:
	case <-time.After(2 * time.Second):
		t.Fatal("hello timeout left the transport open")
	}
}

// TestDialPreservesClassifiedDialError covers a DialFunc that already returns a
// *DialError: attemptDial must not re-wrap it, so callers keep the exact error
// and its absent/fatal classification.
func TestDialPreservesClassifiedDialError(t *testing.T) {
	cases := []struct {
		name   string
		dial   *DialError
		absent bool
	}{
		{name: "absent", dial: &DialError{Endpoint: Endpoint{Network: NetworkUnix, Address: "/tmp/x.sock"}, Absent: true, Err: syscall.ENOENT}, absent: true},
		{name: "fatal", dial: &DialError{Endpoint: Endpoint{Network: NetworkUnix, Address: "/tmp/x.sock"}, Absent: false, Err: syscall.EACCES}, absent: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dial := func(context.Context, Endpoint) (net.Conn, error) { return nil, tc.dial }
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			client, _, err := DialWith(ctx, Endpoint{Network: NetworkUnix, Address: "/tmp/x.sock"}, platformTestHello(), dial)
			if err == nil {
				client.Close()
				t.Fatal("DialWith succeeded on a dial error")
			}
			if client != nil {
				t.Fatal("DialWith returned a client on a dial error")
			}
			var dialErr *DialError
			if !errors.As(err, &dialErr) {
				t.Fatalf("error %v is not a *DialError", err)
			}
			if dialErr != tc.dial {
				t.Fatalf("DialError was re-wrapped: got %p, want %p", dialErr, tc.dial)
			}
			if got := IsMissingEndpoint(err); got != tc.absent {
				t.Fatalf("IsMissingEndpoint(%v) = %v, want %v", err, got, tc.absent)
			}
		})
	}
}

// TestDialClassifiesWrappedConnectErrors pins that wrapped ENOENT/ECONNREFUSED
// still mean "daemon absent" while other wrapped errors stay fatal.
func TestDialClassifiesWrappedConnectErrors(t *testing.T) {
	cases := []struct {
		name   string
		dial   error
		absent bool
	}{
		{name: "wrapped ENOENT", dial: fmt.Errorf("connect: %w", syscall.ENOENT), absent: true},
		{name: "wrapped ECONNREFUSED", dial: fmt.Errorf("connect: %w", syscall.ECONNREFUSED), absent: true},
		{name: "wrapped EACCES", dial: fmt.Errorf("connect: %w", syscall.EACCES), absent: false},
		{name: "plain error", dial: errors.New("boom"), absent: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dial := func(context.Context, Endpoint) (net.Conn, error) { return nil, tc.dial }
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, _, err := DialWith(ctx, Endpoint{Network: NetworkUnix, Address: "/tmp/x.sock"}, platformTestHello(), dial)
			if err == nil {
				t.Fatal("DialWith succeeded on a dial error")
			}
			if got := IsMissingEndpoint(err); got != tc.absent {
				t.Fatalf("IsMissingEndpoint(%v) = %v, want %v", err, got, tc.absent)
			}
		})
	}
}

// TestDialRejectsEndpointBeforeDialing covers endpoint selection validation:
// an unsupported network is rejected without any connection attempt.
func TestDialRejectsEndpointBeforeDialing(t *testing.T) {
	var attempts int
	dial := func(context.Context, Endpoint) (net.Conn, error) {
		attempts++
		return nil, syscall.ENOENT
	}
	_, _, err := DialWith(context.Background(), Endpoint{Network: "tcp", Address: "127.0.0.1:1"}, platformTestHello(), dial)
	if err == nil {
		t.Fatal("DialWith accepted an unsupported endpoint network")
	}
	if attempts != 0 {
		t.Fatalf("dial attempts = %d, want 0 for an invalid endpoint", attempts)
	}
	if IsMissingEndpoint(err) {
		t.Fatal("an invalid endpoint was classified as a missing daemon")
	}
}
