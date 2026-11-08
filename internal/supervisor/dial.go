package supervisor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// ConnectTimeout bounds one connect attempt (protocol-contract §18).
const ConnectTimeout = 5 * time.Second

// DialFunc establishes the byte stream to one endpoint. It is injectable so
// tests never need a real daemon.
type DialFunc func(ctx context.Context, endpoint Endpoint) (net.Conn, error)

// DialError is a failed connect attempt. Absent is true only for ENOENT and
// ECONNREFUSED, the two errors that mean "no daemon is listening"; every other
// error is fatal and must be surfaced instead of spawning (§18).
type DialError struct {
	Endpoint Endpoint
	Absent   bool
	Err      error
}

func (e *DialError) Error() string {
	state := "fatal"
	if e.Absent {
		state = "absent"
	}
	return fmt.Sprintf("supervisor: dial %s %s: %s: %v", e.Endpoint.Network, e.Endpoint.Address, state, e.Err)
}

func (e *DialError) Unwrap() error { return e.Err }

// IsMissingEndpoint reports whether err is a DialError classified as daemon
// absent. The underlying *net.OpError stays reachable through errors.As.
func IsMissingEndpoint(err error) bool {
	var dialErr *DialError
	if errors.As(err, &dialErr) {
		return dialErr.Absent
	}
	return false
}

func isMissingEndpointError(err error) bool {
	return errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ECONNREFUSED)
}

// DefaultDial connects with a 5-second attempt timeout. On Unix it dials the
// filesystem socket; on Windows it opens the named pipe.
func DefaultDial(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
	return dialOnce(ctx, endpoint, connect)
}

// Dial connects to endpoint, performs hello, and returns a live client. The
// returned client owns the connection; callers must Close it.
func Dial(ctx context.Context, endpoint Endpoint, hello appserver.HelloParams) (*appserver.Client, appserver.HelloResult, error) {
	return DialWith(ctx, endpoint, hello, DefaultDial)
}

// DialWith is Dial with an injectable connect function.
func DialWith(ctx context.Context, endpoint Endpoint, hello appserver.HelloParams, dial DialFunc) (*appserver.Client, appserver.HelloResult, error) {
	var zero appserver.HelloResult
	dialed, err := dialConn(ctx, endpoint, hello, dial)
	if err != nil {
		return nil, zero, err
	}
	return dialed.client, dialed.hello, nil
}

// dialed is one successful connect + hello. watch is the transport wrapper
// that records the terminal reason for a Connection.
type dialed struct {
	client *appserver.Client
	hello  appserver.HelloResult
	watch  *watchedConn
}

func dialConn(ctx context.Context, endpoint Endpoint, hello appserver.HelloParams, dial DialFunc) (*dialed, error) {
	if err := endpoint.Validate(); err != nil {
		return nil, err
	}
	if err := hello.Validate(); err != nil {
		return nil, fmt.Errorf("supervisor: hello params: %w", err)
	}
	if dial == nil {
		dial = DefaultDial
	}
	conn, err := dialOnce(ctx, endpoint, dial)
	if err != nil {
		return nil, err
	}
	watch := &watchedConn{Conn: conn}
	client := appserver.NewClient(watch)
	raw, err := client.Call(ctx, "hello", hello)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("supervisor: hello on %s: %w", endpoint, err)
	}
	result, err := appserver.DecodeHelloResult(raw)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("supervisor: hello on %s: %w", endpoint, err)
	}
	return &dialed{client: client, hello: result, watch: watch}, nil
}

// dialOnce runs one attempt with the 5-second connect timeout (shortened when
// ctx already expires sooner).
func dialOnce(ctx context.Context, endpoint Endpoint, dial DialFunc) (net.Conn, error) {
	if err := endpoint.Validate(); err != nil {
		return nil, &DialError{Endpoint: endpoint, Err: err}
	}
	attemptCtx, cancel := connectAttemptContext(ctx, ConnectTimeout)
	defer cancel()
	return attemptDial(attemptCtx, endpoint, dial)
}

// attemptDial classifies the outcome of one bounded connect.
func attemptDial(ctx context.Context, endpoint Endpoint, dial DialFunc) (net.Conn, error) {
	conn, err := dial(ctx, endpoint)
	if err != nil {
		var dialErr *DialError
		if errors.As(err, &dialErr) {
			return nil, err
		}
		return nil, &DialError{Endpoint: endpoint, Absent: isMissingEndpointError(err), Err: err}
	}
	return conn, nil
}

func connectAttemptContext(ctx context.Context, limit time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= limit {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, limit)
}
