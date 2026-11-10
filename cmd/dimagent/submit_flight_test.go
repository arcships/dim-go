package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

type gatedSubmitSession struct {
	*fakeSession
	started chan string
	release chan error
}

func (f *gatedSubmitSession) SubmitWithID(ctx context.Context, text string) (appserver.SubmitResult, string, error) {
	return f.SubmitBranchWithID(ctx, "main", text)
}
func (f *gatedSubmitSession) SubmitBranchWithID(ctx context.Context, branch, text string) (appserver.SubmitResult, string, error) {
	f.started <- branch + ":" + text
	select {
	case err := <-f.release:
		return appserver.SubmitResult{}, "submission", err
	case <-ctx.Done():
		return appserver.SubmitResult{}, "submission", ctx.Err()
	}
}
func TestSubmitSingleFlightPreservesDifferentIntents(t *testing.T) {
	for _, outcome := range []error{nil, errors.New("refused"), &appserver.UnknownResultError{Method: appserver.MutationSessionSubmit}} {
		name := "success"
		if outcome != nil {
			name = outcome.Error()
		}
		t.Run(name, func(t *testing.T) {
			f := &gatedSubmitSession{fakeSession: &fakeSession{}, started: make(chan string, 8), release: make(chan error, 8)}
			a := &app{state: &dimui.State{Selected: "s1", Draft: "new draft", ActionError: "existing"}, bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}}
			a.updateHook = func(fn func(*dimui.State)) { fn(a.state) }
			next := func(want string) {
				t.Helper()
				select {
				case got := <-f.started:
					if got != want {
						t.Fatalf("target=%q want %q", got, want)
					}
				case <-time.After(time.Second):
					t.Fatalf("intent %q suppressed", want)
				}
			}
			a.submit("one")
			next("main:one")
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			a.submit("one")
			select {
			case <-f.started:
				t.Fatal("duplicate submission")
			case <-time.After(30 * time.Millisecond):
			}
			a.submit("two")
			next("main:two")
			a.mu.Lock()
			a.branch = "side"
			a.mu.Unlock()
			a.submit("one")
			next("side:one")
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			for i := 0; i < 3; i++ {
				f.release <- outcome
			}
			waitSubmitFlights(t, a)
			if a.state.Draft != "new draft" || a.state.ActionError != "existing" {
				t.Fatal("stale submit published")
			}
			a.mu.Lock()
			a.branch = "main"
			a.mu.Unlock()
			a.submit("one")
			next("main:one")
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			f.release <- outcome
			waitSubmitFlights(t, a)
		})
	}
}
func waitSubmitFlights(t *testing.T, a *app) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		a.submitMu.Lock()
		pending := len(a.submitInFlight)
		a.submitMu.Unlock()
		if pending == 0 {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatal("submit guards not released")
		}
	}
}

func TestSubmitAcceptedPreservesNewerDraft(t *testing.T) {
	f := &gatedSubmitSession{fakeSession: &fakeSession{}, started: make(chan string, 1), release: make(chan error, 1)}
	a := &app{state: &dimui.State{Selected: "s1", Draft: "newer"}, bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}}
	a.updateHook = func(fn func(*dimui.State)) { fn(a.state) }
	a.submit("older")
	select {
	case <-f.started:
	case <-time.After(time.Second):
		t.Fatal("not started")
	}
	f.release <- nil
	waitSubmitFlights(t, a)
	if a.state.Draft != "newer" {
		t.Fatal("accepted older submission erased newer draft")
	}
}
