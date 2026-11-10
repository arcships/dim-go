package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

type gatedDeleteSession struct {
	*fakeSession
	started chan struct{}
	release chan error
}

func (f *gatedDeleteSession) Delete(ctx context.Context) error {
	f.started <- struct{}{}
	select {
	case err := <-f.release:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func TestAppDeleteSingleFlightAcrossSelections(t *testing.T) {
	for _, outcome := range []error{nil, errors.New("refused"), &appserver.UnknownResultError{Method: appserver.MutationSessionDelete}} {
		name := "success"
		if outcome != nil {
			name = outcome.Error()
		}
		t.Run(name, func(t *testing.T) {
			f := &gatedDeleteSession{fakeSession: &fakeSession{}, started: make(chan struct{}, 8), release: make(chan error, 8)}
			b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
			b.catalogSessions = func(context.Context, *supervisor.Supervisor) ([]dimui.Session, error) {
				return []dimui.Session{{ID: "s1"}}, nil
			}
			a := &app{state: &dimui.State{Selected: "s1", ActionError: "existing"}, bridge: b}
			a.updateHook = func(fn func(*dimui.State)) { fn(a.state) }
			a.deleteSession("s1")
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("delete not started")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			a.deleteSession("s1")
			// The old selectMu only queues the duplicate; release the first request
			// before checking whether a second mutation reaches the controller.
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			f.release <- outcome
			select {
			case <-f.started:
				t.Fatal("queued duplicate delete reached controller")
			case <-time.After(30 * time.Millisecond):
			}
			waitDeleteFlights(t, a)
			if a.state.ActionError != "existing" || len(a.state.ActionNotices) != 0 {
				t.Fatal("stale delete published")
			}
			a.deleteSession("s1")
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("guard not released")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			f.release <- outcome
			waitDeleteFlights(t, a)
		})
	}
}
func waitDeleteFlights(t *testing.T, a *app) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		a.deleteMu.Lock()
		pending := len(a.deleteInFlight)
		a.deleteMu.Unlock()
		if pending == 0 {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatal("delete guards did not release")
		}
	}
}
