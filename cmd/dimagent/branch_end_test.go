package main

import (
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// branchEndFake builds a session whose directory holds main and side, with the
// side branch's own history and a main view that no longer lists it. The main
// view is what a successful end and refresh publishes.
func branchEndFake() *fakeSession {
	return &fakeSession{
		view: &dimui.State{
			Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady,
			Branches: []dimui.Branch{
				{ID: "main", Label: "main", State: "open"},
				{ID: "side", Label: "side", State: "open"},
			},
			Messages: []dimui.Message{{ID: "m-main", Content: "main row"}},
		},
		branchViews: map[string]*dimui.State{
			"side": {
				Selected: "s1", SelectedBranch: "side", Connection: dimui.ConnReady,
				Branches: []dimui.Branch{
					{ID: "main", Label: "main", State: "open"},
					{ID: "side", Label: "side", State: "open"},
				},
				Messages: []dimui.Message{{ID: "m-side", Content: "side row"}},
			},
			"main": {
				Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady,
				Branches: []dimui.Branch{{ID: "main", Label: "main", State: "open"}},
				Messages: []dimui.Message{{ID: "m-main", Content: "main row"}},
			},
		},
		refreshed: make(chan struct{}, 1),
	}
}

// TestAppEndBranchRefreshesAndFallsBackToMain checks the end path: the shown
// branch is ended through the controller, the directory is re-read, and a
// branch that was displayed falls back to main only after the App Server
// answered.
func TestAppEndBranchRefreshesAndFallsBackToMain(t *testing.T) {
	f := branchEndFake()
	a, _, mu := branchApp(f)
	a.state.OnSelectBranch("side")
	waitForState(t, a, mu, "the side branch shown", func(state *dimui.State) bool {
		return state.SelectedBranch == "side"
	})

	a.state.OnEndBranch("side")
	waitForState(t, a, mu, "the ended branch replaced by main", func(state *dimui.State) bool {
		return state.SelectedBranch == "main" && len(state.Branches) == 1 && state.Branches[0].ID == "main"
	})

	mu.Lock()
	defer mu.Unlock()
	if len(f.endedBranches) != 1 || f.endedBranches[0] != "side" {
		t.Fatalf("branches ended = %q, want [side]", f.endedBranches)
	}
	if f.refreshCalls != 1 {
		t.Fatalf("branch directory refreshes = %d, want 1 after an end", f.refreshCalls)
	}
	if len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m-main" {
		t.Fatalf("messages = %#v, want the main branch's rows", a.state.Messages)
	}
	if a.state.ActionError != "" {
		t.Fatalf("action error = %q after an ended branch", a.state.ActionError)
	}
}

// TestAppEndBranchFailureKeepsTheRow checks that a refused end changes nothing:
// the directory keeps the row, the shown branch stays, no refresh is sent, and
// the failure is reported instead of looking like an ended branch.
func TestAppEndBranchFailureKeepsTheRow(t *testing.T) {
	f := branchEndFake()
	f.endBranchErr = errors.New("session.branch_end is not supported")
	a, _, mu := branchApp(f)
	a.state.OnSelectBranch("side")
	waitForState(t, a, mu, "the side branch shown", func(state *dimui.State) bool {
		return state.SelectedBranch == "side"
	})

	a.state.OnEndBranch("side")
	waitForState(t, a, mu, "the end failure reported", func(state *dimui.State) bool {
		return state.ActionError != ""
	})

	mu.Lock()
	defer mu.Unlock()
	if a.state.SelectedBranch != "side" {
		t.Fatalf("selected branch = %q after a failed end, want the shown side", a.state.SelectedBranch)
	}
	if len(a.state.Branches) != 2 {
		t.Fatalf("branches = %#v after a failed end, want both rows kept", a.state.Branches)
	}
	if f.refreshCalls != 0 {
		t.Fatalf("branch directory refreshes = %d after a failed end, want 0", f.refreshCalls)
	}
	if f.shownBranch == appserver.MainBranchID {
		t.Fatalf("a failed end read the main branch instead of keeping side")
	}
}

// TestAppRefreshBranchesReadsTheDirectory checks the refresh action: one
// session.branches read through the controller, and the view it returns is the
// one the window publishes.
func TestAppRefreshBranchesReadsTheDirectory(t *testing.T) {
	f := &fakeSession{
		view: &dimui.State{
			Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady,
			Branches: []dimui.Branch{
				{ID: "main", Label: "main", State: "open"},
				{ID: "side", Label: "side", State: "open"},
			},
		},
		refreshed: make(chan struct{}, 1),
	}
	a, _, mu := branchApp(f)

	a.state.OnRefreshBranches()
	select {
	case <-f.refreshed:
	case <-time.After(2 * time.Second):
		t.Fatal("the branch directory refresh did not reach the session")
	}
	waitForState(t, a, mu, "the refreshed directory", func(state *dimui.State) bool {
		return len(state.Branches) == 2 && state.Branches[1].ID == "side"
	})

	mu.Lock()
	defer mu.Unlock()
	if f.refreshCalls != 1 {
		t.Fatalf("branch directory refreshes = %d, want 1", f.refreshCalls)
	}
	if a.state.ActionError != "" {
		t.Fatalf("action error = %q after a directory refresh", a.state.ActionError)
	}
}

// TestAppEndBranchRejectsMain checks the App Server's own rule before a send:
// main is not endable, so a direct callback cannot ask the daemon to end it.
func TestAppEndBranchRejectsMain(t *testing.T) {
	f := branchEndFake()
	a, _, mu := branchApp(f)

	a.state.OnEndBranch(appserver.MainBranchID)
	waitForState(t, a, mu, "the main refusal reported", func(state *dimui.State) bool {
		return state.ActionError != ""
	})

	mu.Lock()
	defer mu.Unlock()
	if len(f.endedBranches) != 0 {
		t.Fatalf("main was sent to session.branch_end: %q", f.endedBranches)
	}
	if f.refreshCalls != 0 {
		t.Fatalf("a main refusal refreshed the directory %d times", f.refreshCalls)
	}
}
