package integration

import (
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

type healthFixture struct {
	closed *appserver.ClosedNotice
	state  appserver.StreamState
	reason string
}

func (f healthFixture) Closed() (appserver.ClosedNotice, bool) {
	if f.closed == nil {
		return appserver.ClosedNotice{}, false
	}
	return *f.closed, true
}

func (f healthFixture) TerminalState() appserver.StreamState { return f.state }
func (f healthFixture) TerminalReason() string               { return f.reason }

// TestTerminalClosed covers the session.closed health state mapping: a closed
// notice or a closed stream is terminal and carries its reason, while lost and
// detached states are not reported as closed (lost still needs a resync).
func TestTerminalClosed(t *testing.T) {
	notice := appserver.ClosedNotice{SessionID: "s1", Reason: "done"}
	cases := []struct {
		name       string
		fixture    healthFixture
		wantClosed bool
		wantReason string
	}{
		{"closed notice", healthFixture{closed: &notice}, true, "done"},
		{"closed notice without reason", healthFixture{closed: &appserver.ClosedNotice{SessionID: "s1"}, reason: "fallback"}, true, "fallback"},
		{"closed stream", healthFixture{state: appserver.StreamStateClosed, reason: "stream done"}, true, "stream done"},
		{"attached", healthFixture{state: appserver.StreamStateAttached}, false, ""},
		{"lost", healthFixture{state: appserver.StreamStateLost, reason: "connection_closed"}, false, ""},
		{"detached", healthFixture{state: appserver.StreamStateDetached}, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closed, reason := terminalClosed(tc.fixture)
			if closed != tc.wantClosed || reason != tc.wantReason {
				t.Fatalf("terminalClosed = %v, %q; want %v, %q", closed, reason, tc.wantClosed, tc.wantReason)
			}
		})
	}
}
