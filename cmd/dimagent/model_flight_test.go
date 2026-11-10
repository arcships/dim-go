package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/session"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

type gatedModelSession struct {
	*fakeSession
	started chan string
	release chan error
}

func (f *gatedModelSession) UpdateModel(ctx context.Context, provider, model string) (appserver.ModelUpdateResult, error) {
	f.started <- provider + "/" + model
	select {
	case err := <-f.release:
		return appserver.ModelUpdateResult{}, err
	case <-ctx.Done():
		return appserver.ModelUpdateResult{}, ctx.Err()
	}
}

func (f *gatedModelSession) RefreshModel(context.Context) (session.ModelSelection, bool, error) {
	return session.ModelSelection{}, false, nil
}

func (f *gatedModelSession) Models(context.Context) (session.ModelCatalog, error) {
	return session.ModelCatalog{}, nil
}

func TestSelectModelSingleFlightPreservesDifferentIntents(t *testing.T) {
	for _, outcome := range []error{nil, errors.New("refused"), &appserver.UnknownResultError{Method: appserver.MutationSessionModelUpdate}} {
		name := "success"
		if outcome != nil {
			name = outcome.Error()
		}
		t.Run(name, func(t *testing.T) {
			f := &gatedModelSession{fakeSession: &fakeSession{}, started: make(chan string, 8), release: make(chan error, 8)}
			a := &app{state: &dimui.State{Selected: "s1", ActionError: "existing"}, bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}}
			updates := make(chan struct{}, 8)
			a.updateHook = func(fn func(*dimui.State)) { fn(a.state); updates <- struct{}{} }
			a.selectModel("p", "m1")
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("update did not start")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			a.selectModel("p", "m1")
			select {
			case <-f.started:
				t.Fatal("duplicate model update reached controller")
			case <-time.After(30 * time.Millisecond):
			}
			if len(updates) != 0 || a.state.ActionError != "existing" {
				t.Fatal("suppressed update changed state")
			}
			a.selectModel("p", "m2")
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("different model intent suppressed")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			f.release <- outcome
			f.release <- outcome
			waitModelFlights(t, a)
			if a.state.ActionError != "existing" || len(a.state.ActionNotices) != 0 || a.state.ModelCatalog.HasCurrent {
				t.Fatal("stale model result published")
			}
			for len(updates) > 0 {
				<-updates
			}
			a.selectModel("p", "m1")
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("guard was not released")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			f.release <- outcome
			waitModelFlights(t, a)
		})
	}
}

func waitModelFlights(t *testing.T, a *app) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		a.modelMu.Lock()
		pending := len(a.modelInFlight)
		a.modelMu.Unlock()
		if pending == 0 {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatal("model guards did not release")
		}
	}
}
