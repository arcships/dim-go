//go:build unix

package supervisor

import (
	"context"
	"strings"
	"testing"
)

// TestConnectRejectsPipeNetwork pins Unix endpoint selection: a pipe endpoint
// is a fatal local error, not "daemon absent", because it cannot be dialed on
// this platform.
func TestConnectRejectsPipeNetwork(t *testing.T) {
	_, err := connect(context.Background(), Endpoint{Network: NetworkPipe, Address: `\\.\pipe\dim-rust-x`})
	if err == nil {
		t.Fatal("connect accepted a pipe endpoint on unix")
	}
	if !strings.Contains(err.Error(), "not supported on this platform") {
		t.Fatalf("error = %v, want an unsupported-platform error", err)
	}
	if IsMissingEndpoint(err) {
		t.Fatal("a wrong-network endpoint was classified as a missing daemon")
	}
}
