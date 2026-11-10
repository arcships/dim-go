package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// TestAppBranchFromCreatesRefreshesAndSelects checks the cut point path: the
// window sends the row's parent sequence and prompt once, re-reads the branch
// directory, opens the branch the App Server returned, and only then shows that
// branch and its rows.
func TestAppBranchFromCreatesRefreshesAndSelects(t *testing.T) {
	f := &fakeSession{
		view: &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady,
			Messages: []dimui.Message{{ID: "m-main"}}},
		branchViews: map[string]*dimui.State{
			"side": {Selected: "s1", SelectedBranch: "side", Connection: dimui.ConnReady,
				Messages: []dimui.Message{{ID: "m-side", Content: "side row"}}},
		},
		createBranchResult: appserver.BranchCreateResult{Branch: &appserver.BranchCreateInfo{
			BranchID: "side", ParentBranchID: appserver.MainBranchID,
		}},
	}
	a, _, mu := branchApp(f)

	a.state.OnBranchFrom("s1", appserver.MainBranchID, 4, "why does this fail?")
	waitForState(t, a, mu, "the created branch shown", func(state *dimui.State) bool {
		return state.SelectedBranch == "side"
	})

	mu.Lock()
	defer mu.Unlock()
	if f.createBranchCalls != 1 {
		t.Fatalf("BranchCreate calls = %d, want one", f.createBranchCalls)
	}
	if f.createBranchParent != appserver.MainBranchID || f.createBranchAt != 4 || f.createBranchPrompt != "why does this fail?" {
		t.Fatalf("BranchCreate got parent=%q at=%d prompt=%q", f.createBranchParent, f.createBranchAt, f.createBranchPrompt)
	}
	if f.refreshCalls != 1 {
		t.Fatalf("branch directory refreshes = %d, want one before the branch is shown", f.refreshCalls)
	}
	if len(f.openBranch) != 1 || f.openBranch[0] != "side" {
		t.Fatalf("branches opened = %q, want [side]", f.openBranch)
	}
	if len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m-side" {
		t.Fatalf("messages = %#v, want the created branch's rows", a.state.Messages)
	}
	if a.state.ActionError != "" {
		t.Fatalf("action error = %q after a created branch", a.state.ActionError)
	}
}

// TestAppBranchFromFailureKeepsTheCurrentBranch checks that a rejected create
// changes nothing: no directory refresh, no branch open and the displayed
// branch and rows stay, while the failure is reported.
func TestAppBranchFromFailureKeepsTheCurrentBranch(t *testing.T) {
	f := &fakeSession{
		view: &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady,
			Messages: []dimui.Message{{ID: "m-main"}}},
		createBranchErr: errors.New("branch is not a rest point"),
	}
	a, _, mu := branchApp(f)

	a.state.OnBranchFrom("s1", appserver.MainBranchID, 4, "prompt")
	waitForState(t, a, mu, "the create failure reported", func(state *dimui.State) bool {
		return state.ActionError != ""
	})

	mu.Lock()
	defer mu.Unlock()
	if a.state.SelectedBranch != appserver.MainBranchID {
		t.Fatalf("selected branch = %q after a failed create, want main", a.state.SelectedBranch)
	}
	if len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m-main" {
		t.Fatalf("messages = %#v after a failed create, want the main rows", a.state.Messages)
	}
	if f.refreshCalls != 0 || len(f.openBranch) != 0 {
		t.Fatalf("a failed create refreshed %d times and opened %q", f.refreshCalls, f.openBranch)
	}
	if !strings.Contains(a.state.ActionError, "branch is not a rest point") {
		t.Fatalf("action error = %q, want the App Server's reason", a.state.ActionError)
	}
	if len(a.state.ActionNotices) != 1 || a.state.ActionNotices[0].Kind != "branch.error" {
		t.Fatalf("branch failure was not recorded: %#v", a.state.ActionNotices)
	}
}

// TestAppBranchFromUnknownResultIsNotResent checks the write's fail-closed
// rule: an unknown result is reported once and the create is never sent again.
func TestAppBranchFromUnknownResultIsNotResent(t *testing.T) {
	f := &fakeSession{
		view: &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady},
		createBranchErr: &appserver.UnknownResultError{
			Method:   appserver.MutationSessionBranchCreate,
			ReadBack: "session.branches",
		},
	}
	a, _, mu := branchApp(f)

	a.state.OnBranchFrom("s1", appserver.MainBranchID, 4, "prompt")
	waitForState(t, a, mu, "the unknown result reported", func(state *dimui.State) bool {
		return state.ActionError != ""
	})

	mu.Lock()
	defer mu.Unlock()
	if f.createBranchCalls != 1 {
		t.Fatalf("BranchCreate calls = %d, want exactly one", f.createBranchCalls)
	}
	if f.refreshCalls != 0 || len(f.openBranch) != 0 {
		t.Fatalf("an unknown result refreshed %d times and opened %q", f.refreshCalls, f.openBranch)
	}
	for _, warning := range []string{"result unknown", "may already have been created", "trying again may create a duplicate", "branch directory", "cannot conclusively identify"} {
		if !strings.Contains(a.state.ActionError, warning) {
			t.Fatalf("action error = %q, missing warning %q", a.state.ActionError, warning)
		}
	}
}

// TestAppBranchFromRefusesAnEmptyPromptBeforeSending checks the window's own
// gate: a blank prompt sends nothing, so the App Server never sees a request
// its validation would reject.
func TestAppBranchFromRefusesAnEmptyPromptBeforeSending(t *testing.T) {
	f := &fakeSession{
		view: &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady},
	}
	a, _, mu := branchApp(f)

	a.state.OnBranchFrom("s1", appserver.MainBranchID, 4, "   ")
	waitForState(t, a, mu, "the empty prompt reported", func(state *dimui.State) bool {
		return state.ActionError != ""
	})

	mu.Lock()
	defer mu.Unlock()
	if f.createBranchCalls != 0 {
		t.Fatalf("BranchCreate calls = %d for an empty prompt, want none", f.createBranchCalls)
	}
}
