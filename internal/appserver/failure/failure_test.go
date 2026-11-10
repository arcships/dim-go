package failure_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/transport"
)

func TestKindOfKnownFailures(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want failure.Kind
	}{
		{"request timeout", &protocol.TimeoutError{Method: "x", Timeout: time.Second}, failure.KindTimeout},
		{"context deadline", context.DeadlineExceeded, failure.KindTimeout},
		{"connection closed", &protocol.ClosedError{Cause: io.EOF}, failure.KindRetryable},
		{"eof", io.EOF, failure.KindRetryable},
		{"cancel", context.Canceled, failure.KindNonRetryable},
		{"plain", errors.New("boom"), failure.KindNonRetryable},
	}
	for _, tc := range cases {
		if got := failure.KindOf(tc.err); got != tc.want {
			t.Errorf("%s: kind %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestKindOfRPCAppErrors(t *testing.T) {
	cases := []struct {
		appError string
		code     int
		want     failure.Kind
	}{
		{"protocol_version", -32000, failure.KindVersionIncompatible},
		{"session_protocol_version", -32000, failure.KindVersionIncompatible},
		{"request_timeout", -32000, failure.KindTimeout},
		{"connection_closed", -32000, failure.KindRetryable},
		{"service_reloading", -32000, failure.KindRetryable},
		{"connection_busy", -32000, failure.KindRetryable},
		{"subscription_changed", -32000, failure.KindRetryable},
		{"unknown_method", -32601, failure.KindNonRetryable},
		{"service_unavailable", -32000, failure.KindNonRetryable},
		{"", -32602, failure.KindNonRetryable},
	}
	for _, tc := range cases {
		err := &protocol.RPCError{Code: tc.code, Message: "m"}
		if tc.appError != "" {
			err.Data = []byte(`{"appError":"` + tc.appError + `"}`)
		}
		if got := failure.KindOf(err); got != tc.want {
			t.Errorf("appError %q: kind %q, want %q", tc.appError, got, tc.want)
		}
	}
}

func TestMissingEndpointIsClassified(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "d1-failure-")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = transport.Dial(ctx, transport.Endpoint{
		Network: transport.NetworkUnix,
		Address: filepath.Join(dir, "missing.sock"),
	}, time.Second)
	if err == nil {
		t.Fatal("want dial failure")
	}
	if !failure.Is(err, failure.KindConnectionMissing) {
		t.Fatalf("kind %q, want connection_missing (err=%v)", failure.KindOf(err), err)
	}
}

func TestWrapPreservesKindAndRetryability(t *testing.T) {
	classified := failure.New(failure.KindTimeout, "hello", errors.New("slow"))
	wrapped := failure.Wrap(failure.KindNonRetryable, "hello", classified)
	if failure.KindOf(wrapped) != failure.KindTimeout {
		t.Fatalf("wrap must keep the existing kind, got %q", failure.KindOf(wrapped))
	}
	if !failure.Retryable(wrapped) {
		t.Fatal("timeout must be retryable")
	}
	if !errors.Is(wrapped, &failure.Error{Kind: failure.KindTimeout}) {
		t.Fatal("errors.Is by kind failed")
	}
	if failure.Retryable(failure.New(failure.KindConnectionMissing, "dial", nil)) {
		t.Fatal("connection_missing is not a blind retry")
	}
}
