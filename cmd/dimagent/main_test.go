package main

import (
	"context"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

func TestStateFromDaemonWithoutProductHome(t *testing.T) {
	state, closeResources, err := stateFromDaemon(context.Background(), "")
	if err != nil {
		t.Fatalf("stateFromDaemon: %v", err)
	}
	if state == nil || state.Connection != dimui.ConnUnknown {
		t.Fatalf("state = %#v, want a not-connected state", state)
	}
	if closeResources != nil {
		t.Fatal("closeResources must be nil without a daemon connection")
	}
	if len(state.Sessions) != 0 || len(state.Messages) != 0 {
		t.Fatalf("state must not invent fixtures: %#v", state)
	}
}

func TestRefreshStateWithoutControllerIsNoop(t *testing.T) {
	state := &dimui.State{Selected: "keep", Connection: dimui.ConnReady}
	refreshState(state, nil, appserver.SessionSummary{SessionID: "s1"})
	if state.Selected != "keep" || state.Connection != dimui.ConnReady {
		t.Fatalf("state changed on nil controller: %#v", state)
	}
}

func TestSummaryForRow(t *testing.T) {
	summary := summaryForRow(dimui.Session{ID: "s1", Title: "Title"})
	if summary.SessionID != "s1" {
		t.Fatalf("session id = %q", summary.SessionID)
	}
	if summary.Title == nil || *summary.Title != "Title" {
		t.Fatalf("title = %v", summary.Title)
	}
	if summary.Cwd != "" || summary.Model.ProviderID != "" {
		t.Fatalf("summary must not invent daemon facts: %#v", summary)
	}
}
