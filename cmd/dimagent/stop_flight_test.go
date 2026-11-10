package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

type gatedStopSession struct {
	*fakeSession
	started chan string
	release chan error
}

func (f *gatedStopSession) Stop(ctx context.Context) error {
	return f.StopBranch(ctx, appserver.MainBranchID)
}
func (f *gatedStopSession) StopBranch(ctx context.Context, branch string) error {
	f.started <- branch
	select {
	case err := <-f.release:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestAppStopSingleFlightAcrossSelections(t *testing.T) {
	for _, outcome := range []error{nil, errors.New("refused"), &appserver.UnknownResultError{Method: appserver.MutationSessionStop}} {
		name := "success"
		if outcome != nil {
			name = outcome.Error()
		}
		t.Run(name, func(t *testing.T) {
			f := &gatedStopSession{fakeSession: &fakeSession{}, started: make(chan string, 8), release: make(chan error, 8)}
			a := &app{state: &dimui.State{Selected: "s1", SelectedBranch: "main", ActionError: "existing"}, bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}}
			updates := make(chan struct{}, 8)
			a.updateHook = func(fn func(*dimui.State)) { fn(a.state); updates <- struct{}{} }
			a.stop()
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("stop not started")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			a.stop()
			select {
			case <-f.started:
				t.Fatal("duplicate stop reached controller")
			case <-time.After(30 * time.Millisecond):
			}
			if len(updates) != 0 || a.state.ActionError != "existing" {
				t.Fatal("suppressed stop changed state")
			}
			a.mu.Lock()
			a.branch = "side"
			a.mu.Unlock()
			a.stop()
			select {
			case branch := <-f.started:
				if branch != "side" {
					t.Fatal("wrong target")
				}
			case <-time.After(time.Second):
				t.Fatal("different target suppressed")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			f.release <- outcome
			f.release <- outcome
			waitStopFlights(t, a)
			if len(updates) != 0 || a.state.ActionError != "existing" {
				t.Fatal("stale stop published")
			}
			a.mu.Lock()
			a.branch = "main"
			a.mu.Unlock()
			a.stop()
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("guard not released")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			f.release <- outcome
			waitStopFlights(t, a)
		})
	}
}
func waitStopFlights(t *testing.T, a *app) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		a.stopMu.Lock()
		pending := len(a.stopInFlight)
		a.stopMu.Unlock()
		if pending == 0 {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatal("stop guards did not release")
		}
	}
}
