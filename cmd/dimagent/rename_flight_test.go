package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

type gatedRenameSession struct {
	*fakeSession
	started chan *string
	release chan error
}

func (f *gatedRenameSession) Rename(ctx context.Context, title *string) (appserver.RenameResult, error) {
	f.started <- title
	select {
	case err := <-f.release:
		return appserver.RenameResult{}, err
	case <-ctx.Done():
		return appserver.RenameResult{}, ctx.Err()
	}
}

func TestRenameSingleFlightUsesWireIntent(t *testing.T) {
	for _, outcome := range []error{nil, errors.New("refused"), &appserver.UnknownResultError{Method: appserver.MutationSessionRename}} {
		name := "success"
		if outcome != nil {
			name = outcome.Error()
		}
		for _, clear := range []bool{false, true} {
			name2 := "title"
			if clear {
				name2 = "clear"
			}
			t.Run(name+"/"+name2, func(t *testing.T) {
				f := &gatedRenameSession{fakeSession: &fakeSession{}, started: make(chan *string, 8), release: make(chan error, 8)}
				b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
				b.catalogSessions = func(context.Context, *supervisor.Supervisor) ([]dimui.Session, error) { return nil, nil }
				a := &app{bridge: b, state: &dimui.State{Selected: "s1", ActionError: "existing"}}
				a.updateHook = func(fn func(*dimui.State)) { fn(a.state) }
				first, duplicate := "  Title  ", "Title"
				if clear {
					first, duplicate = " \t ", ""
				}
				a.renameSession(first)
				select {
				case title := <-f.started:
					if clear && title != nil {
						t.Fatal("clear must send nil")
					}
					if !clear && (title == nil || *title != "Title") {
						t.Fatal("wire title not normalized")
					}
				case <-time.After(time.Second):
					t.Fatal("rename not started")
				}
				a.mu.Lock()
				a.selectGen++
				a.mu.Unlock()
				a.renameSession(duplicate)
				select {
				case <-f.started:
					t.Fatal("duplicate normalized rename reached controller")
				case <-time.After(30 * time.Millisecond):
				}
				if a.state.ActionError != "existing" || len(a.state.ActionNotices) != 0 || a.state.ActionStatus != "Session rename is already in progress; the duplicate request was not sent." {
					t.Fatal("duplicate changed error or omitted visible pending status")
				}
				a.renameSession("Different")
				select {
				case <-f.started:
				case <-time.After(time.Second):
					t.Fatal("different title suppressed")
				}
				a.mu.Lock()
				a.selectGen++
				a.mu.Unlock()
				f.release <- outcome
				f.release <- outcome
				waitRenameFlights(t, a)
				if a.state.ActionError != "existing" || len(a.state.ActionNotices) != 0 || a.state.ActionStatus != "Session rename is already in progress; the duplicate request was not sent." {
					t.Fatal("stale rename published")
				}
				a.renameSession(duplicate)
				select {
				case <-f.started:
				case <-time.After(time.Second):
					t.Fatal("guard not released")
				}
				a.mu.Lock()
				a.selectGen++
				a.mu.Unlock()
				f.release <- outcome
				waitRenameFlights(t, a)
			})
		}
	}
}
func TestRenameGuardDoesNotRetainTitle(t *testing.T) {
	typ := reflect.TypeOf(renameFlight{})
	if _, ok := typ.FieldByName("title"); ok {
		t.Fatal("guard retains full title")
	}
	field, ok := typ.FieldByName("digest")
	if !ok || field.Type != reflect.TypeOf([32]byte{}) {
		t.Fatal("guard must use complete SHA256 digest")
	}
}

func waitRenameFlights(t *testing.T, a *app) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		a.renameMu.Lock()
		pending := len(a.renameInFlight)
		a.renameMu.Unlock()
		if pending == 0 {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatal("rename guards did not release")
		}
	}
}
