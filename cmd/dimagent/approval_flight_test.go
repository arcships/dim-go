package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

type gatedApprovalSession struct {
	*fakeSession
	started chan string
	release chan error
}

func (f *gatedApprovalSession) RespondApproval(ctx context.Context, branch, request, decision string) (appserver.ApprovalRespondOutcome, error) {
	f.started <- request + ":" + decision
	select {
	case err := <-f.release:
		return appserver.ApprovalOutcomeResponded, err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestAppApprovalSingleFlightAcrossSelectionsAndDecisions(t *testing.T) {
	for _, outcome := range []error{nil, errors.New("refused")} {
		t.Run(map[bool]string{true: "failure", false: "success"}[outcome != nil], func(t *testing.T) {
			f := &gatedApprovalSession{fakeSession: &fakeSession{}, started: make(chan string, 8), release: make(chan error, 8)}
			a := &app{state: &dimui.State{Selected: "s1", ActionError: "existing"}, bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}}
			updates := make(chan struct{}, 8)
			a.updateHook = func(func(*dimui.State)) { updates <- struct{}{} }
			a.respondApproval("main", "req1", dimui.ApprovalAllow)
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("approval did not start")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			a.respondApproval("main", "req1", dimui.ApprovalDeny)
			select {
			case <-f.started:
				t.Fatal("duplicate/conflicting decision reached controller")
			case <-time.After(30 * time.Millisecond):
			}
			if a.state.ActionError != "existing" || len(updates) != 0 {
				t.Fatal("suppression changed state")
			}
			// A different approval request remains independent.
			a.respondApproval("main", "req2", dimui.ApprovalDeny)
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("different approval suppressed")
			}
			f.release <- outcome
			f.release <- outcome
			deadline := time.After(time.Second)
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()
			for {
				a.approvalMu.Lock()
				pending := len(a.approvalInFlight)
				a.approvalMu.Unlock()
				if pending == 0 {
					break
				}
				select {
				case <-ticker.C:
				case <-deadline:
					t.Fatal("guard did not release")
				}
			}
			// Only req2 belongs to the current generation. Successful decisions
			// publish view and status; failures publish one error.
			want := 1
			if outcome == nil {
				want = 2
			}
			if len(updates) != want {
				t.Fatalf("updates=%d, want %d (no stale publication)", len(updates), want)
			}
			a.respondApproval("main", "req1", dimui.ApprovalDeny)
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("completed approval guard not released")
			}
			f.release <- outcome
		})
	}
}
