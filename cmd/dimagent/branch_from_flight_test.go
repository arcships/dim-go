package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

type gatedBranchCreateSession struct {
	*fakeSession
	started chan string
	release chan error
}

func (f *gatedBranchCreateSession) BranchCreate(ctx context.Context, parent string, at int64, prompt string) (appserver.BranchCreateResult, error) {
	f.started <- prompt
	select {
	case err := <-f.release:
		return appserver.BranchCreateResult{Branch: &appserver.BranchCreateInfo{BranchID: "side"}}, err
	case <-ctx.Done():
		return appserver.BranchCreateResult{}, ctx.Err()
	}
}

func TestAppBranchFromSingleFlightPreservesDifferentIntents(t *testing.T) {
	for _, outcome := range []error{nil, errors.New("refused"), &appserver.UnknownResultError{Method: appserver.MutationSessionBranchCreate}} {
		t.Run(map[bool]string{true: "failure", false: "success"}[outcome != nil], func(t *testing.T) {
			f := &gatedBranchCreateSession{fakeSession: &fakeSession{}, started: make(chan string, 8), release: make(chan error, 8)}
			a := &app{state: &dimui.State{Selected: "s1", SelectedBranch: "main", ActionError: "existing"}, bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}}
			updates := make(chan struct{}, 8)
			a.updateHook = func(fn func(*dimui.State)) { fn(a.state); updates <- struct{}{} }
			a.branchFrom("s1", "main", 4, "question")
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("create did not start")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			_, before := a.displayBranch()
			a.branchFrom("s1", "main", 4, "question")
			select {
			case <-f.started:
				t.Fatal("duplicate branch creation reached controller")
			case <-time.After(30 * time.Millisecond):
			}
			_, after := a.displayBranch()
			if before != after || len(updates) != 1 || a.state.ActionError != "existing" || a.state.ActionStatus != "Branch creation is already in progress; the duplicate request was not sent." {
				t.Fatal("suppressed create changed generation/error or omitted visible pending status")
			}
			<-updates
			// A new prompt is explicitly different intent, not a duplicate.
			a.branchFrom("s1", "main", 4, " question ")
			select {
			case prompt := <-f.started:
				if prompt != " question " {
					t.Fatal("exact prompt changed")
				}
			case <-time.After(time.Second):
				t.Fatal("different intent suppressed")
			}
			// The same prompt at a different cut is independently meaningful.
			a.branchFrom("s1", "main", 5, "question")
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("different cut suppressed")
			}
			// All results are now stale; none can replace the new selection.
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			f.release <- outcome
			f.release <- outcome
			f.release <- outcome
			// Completion is observed via the guard only after the implementation.
			waitBranchCreates(t, a)
			if len(updates) != 0 {
				t.Fatal("stale branch creation published")
			}
			a.branchFrom("s1", "main", 4, "question")
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("completed guard not released")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			f.release <- outcome
			waitBranchCreates(t, a)
		})
	}
}

func TestBranchCreateGuardDoesNotRetainPrompt(t *testing.T) {
	typ := reflect.TypeOf(branchCreateFlight{})
	if _, ok := typ.FieldByName("prompt"); ok {
		t.Fatal("guard retains full prompt")
	}
	field, ok := typ.FieldByName("digest")
	if !ok || field.Type != reflect.TypeOf([32]byte{}) {
		t.Fatal("guard must use complete SHA256 digest")
	}
}

func waitBranchCreates(t *testing.T, a *app) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		a.branchCreateMu.Lock()
		pending := len(a.branchCreateInFlight)
		a.branchCreateMu.Unlock()
		if pending == 0 {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatal("branch-create guards did not release")
		}
	}
}
