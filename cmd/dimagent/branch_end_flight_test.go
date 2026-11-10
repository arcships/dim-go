package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// Each call waits at the wire boundary, letting repeated UI triggers happen
// before the first mutation finishes. No controller state is changed locally.
type gatedBranchEndSession struct {
	*fakeSession
	started chan string
	release chan error
}

func (f *gatedBranchEndSession) EndBranch(ctx context.Context, branch string) (appserver.BranchEndOutcome, error) {
	f.started <- branch
	select {
	case err := <-f.release:
		return appserver.BranchEndOutcome("ended"), err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestAppEndBranchSingleFlight(t *testing.T) {
	for _, outcome := range []error{nil, errors.New("refused")} {
		t.Run(map[bool]string{true: "failure", false: "success"}[outcome != nil], func(t *testing.T) {
			f := &gatedBranchEndSession{fakeSession: &fakeSession{}, started: make(chan string, 8), release: make(chan error, 8)}
			a := &app{state: &dimui.State{Selected: "s1", SelectedBranch: "main"}, bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}}
			updates := make(chan struct{}, 8)
			// These tests exercise requests, not state publication. Avoid a second
			// writer to the UI state while triggering the entry point.
			a.updateHook = func(func(*dimui.State)) { updates <- struct{}{} }
			a.endBranch("side")
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("first end did not start")
			}
			a.endBranch("side")
			select {
			case <-f.started:
				t.Fatal("duplicate end reached controller")
			case <-time.After(30 * time.Millisecond):
			}
			f.release <- outcome
			select {
			case <-updates:
			case <-time.After(time.Second):
				t.Fatal("end did not finish")
			}
			// A completed request (including a failure) must release its guard.
			deadline := time.After(time.Second)
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()
			for {
				a.endBranch("side")
				select {
				case <-f.started:
					f.release <- outcome
					select {
					case <-updates:
					case <-time.After(time.Second):
						t.Fatal("second end did not finish")
					}
					return
				case <-ticker.C:
				case <-deadline:
					t.Fatal("guard was not released")
				}
			}
		})
	}
}

func TestAppEndBranchNewSelectionSuppressesSameTarget(t *testing.T) {
	for _, outcome := range []error{nil, errors.New("refused")} {
		t.Run(map[bool]string{true: "failure", false: "success"}[outcome != nil], func(t *testing.T) {
			testEndBranchNewSelection(t, outcome)
		})
	}
}

func testEndBranchNewSelection(t *testing.T, outcome error) {
	f := &gatedBranchEndSession{fakeSession: &fakeSession{}, started: make(chan string, 8), release: make(chan error, 8)}
	a := &app{state: &dimui.State{Selected: "s1", SelectedBranch: "main"}, bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}}
	updates := make(chan struct{}, 8)
	a.updateHook = func(func(*dimui.State)) { updates <- struct{}{} }
	a.endBranch("side")
	select {
	case <-f.started:
	case <-time.After(time.Second):
		t.Fatal("first end did not start")
	}
	// Selection generations do not change the target mutation identity.
	// Suppress the duplicate and never publish the old success or failure.
	a.mu.Lock()
	a.selectGen++
	a.mu.Unlock()
	a.endBranch("side")
	select {
	case <-f.started:
	case <-time.After(time.Second):
		t.Fatal("new selection was suppressed")
	}
	f.release <- errors.New("refused")
	f.release <- errors.New("refused")
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		a.endBranchMu.Lock()
		pending := len(a.endBranchInFlight)
		a.endBranchMu.Unlock()
		if pending == 0 {
			break
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatal("guards did not release")
		}
	}
	if len(updates) != 1 {
		t.Fatalf("publications = %d, want only the current selection", len(updates))
	}
}

func TestAppEndBranchDifferentBranchesAreIndependent(t *testing.T) {
	f := &gatedBranchEndSession{fakeSession: &fakeSession{}, started: make(chan string, 8), release: make(chan error, 8)}
	a := &app{state: &dimui.State{Selected: "s1", SelectedBranch: "main"}, bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}}
	updates := make(chan struct{}, 8)
	a.updateHook = func(func(*dimui.State)) { updates <- struct{}{} }
	a.endBranch("side-a")
	a.endBranch("side-b")
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case branch := <-f.started:
			seen[branch] = true
		case <-time.After(time.Second):
			t.Fatal("different branch was suppressed")
		}
	}
	if !seen["side-a"] || !seen["side-b"] {
		t.Fatalf("requests = %v", seen)
	}
	f.release <- errors.New("refused")
	f.release <- errors.New("refused")
	for i := 0; i < 2; i++ {
		select {
		case <-updates:
		case <-time.After(time.Second):
			t.Fatal("end did not finish")
		}
	}
}
