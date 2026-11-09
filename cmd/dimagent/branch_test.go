package main

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// branchApp builds a window attached to one session, with the branch views the
// test supplies and the mutex guarding the state its background callbacks
// update. The session shows main until a branch open succeeds.
func branchApp(f *fakeSession) (*app, *sessionBridge, *sync.Mutex) {
	b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	a := &app{
		state: &dimui.State{
			Selected:       "s1",
			SelectedBranch: "main",
			Connection:     dimui.ConnReady,
			Messages:       []dimui.Message{{ID: "m-main", Content: "main row"}},
		},
		bridge: b,
	}
	var mu sync.Mutex
	a.updateHook = func(fn func(*dimui.State)) {
		mu.Lock()
		defer mu.Unlock()
		fn(a.state)
	}
	a.bind()
	return a, b, &mu
}

// TestAppSelectBranchOpensAndShowsIt checks the switch: the branch the sidebar
// chose is read through the controller that still belongs to the selected
// session, and the window shows that branch's own projection only afterward.
func TestAppSelectBranchOpensAndShowsIt(t *testing.T) {
	f := &fakeSession{
		view: &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady,
			Messages: []dimui.Message{{ID: "m-main"}}},
		branchViews: map[string]*dimui.State{
			"side": {Selected: "s1", SelectedBranch: "side", Connection: dimui.ConnReady,
				Messages: []dimui.Message{{ID: "m-side", Content: "side row"}}},
		},
	}
	a, _, mu := branchApp(f)

	a.state.OnSelectBranch("side")
	waitForState(t, a, mu, "the opened branch shown", func(state *dimui.State) bool {
		return state.SelectedBranch == "side"
	})
	mu.Lock()
	defer mu.Unlock()
	if len(f.openBranch) != 1 || f.openBranch[0] != "side" {
		t.Fatalf("branches opened = %q, want [side]", f.openBranch)
	}
	if len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m-side" {
		t.Fatalf("messages = %#v, want the side branch's rows", a.state.Messages)
	}
	if a.state.ActionError != "" {
		t.Fatalf("action error = %q after an opened branch", a.state.ActionError)
	}
}

// TestAppSelectBranchFailureKeepsCurrent checks that a refused read changes no
// branch and no row: the window keeps showing main and reports the failure, so
// a branch the App Server did not open never looks selected.
func TestAppSelectBranchFailureKeepsCurrent(t *testing.T) {
	f := &fakeSession{
		view: &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady,
			Messages: []dimui.Message{{ID: "m-main"}}},
		openBranchErr: errors.New("branch is not in the session.branches directory"),
	}
	a, _, mu := branchApp(f)

	a.state.OnSelectBranch("side")
	waitForState(t, a, mu, "the open failure reported", func(state *dimui.State) bool {
		return state.ActionError != ""
	})
	mu.Lock()
	defer mu.Unlock()
	if a.state.SelectedBranch != "main" {
		t.Fatalf("selected branch = %q after a failed open, want main", a.state.SelectedBranch)
	}
	if len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m-main" {
		t.Fatalf("messages = %#v after a failed open, want the main rows", a.state.Messages)
	}
	if len(a.state.ActionNotices) != 1 || a.state.ActionNotices[0].Kind != "branch.error" {
		t.Fatalf("branch failure was not recorded: %#v", a.state.ActionNotices)
	}
}

// TestAppSelectBranchStaleSelectionIsDropped checks the generation guard: a
// branch read that finishes after the user chose another session must not move
// the new session's branch or rows.
func TestAppSelectBranchStaleSelectionIsDropped(t *testing.T) {
	f := &fakeSession{
		view: &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady,
			Messages: []dimui.Message{{ID: "m-main"}}},
		branchViews: map[string]*dimui.State{
			"side": {Selected: "s1", SelectedBranch: "side",
				Messages: []dimui.Message{{ID: "m-side"}}},
		},
		openBranchStart: make(chan struct{}),
		openBranchGate:  make(chan struct{}),
		openBranchDone:  make(chan struct{}),
	}
	a, _, mu := branchApp(f)

	a.state.OnSelectBranch("side")
	select {
	case <-f.openBranchStart:
	case <-time.After(2 * time.Second):
		t.Fatal("the branch read did not start")
	}
	// A newer session selection invalidates the branch read still in flight.
	a.beginSelect()
	close(f.openBranchGate)
	select {
	case <-f.openBranchDone:
	case <-time.After(2 * time.Second):
		t.Fatal("the branch read did not finish")
	}

	mu.Lock()
	defer mu.Unlock()
	if a.state.SelectedBranch != "main" {
		t.Fatalf("selected branch = %q after a stale read, want main", a.state.SelectedBranch)
	}
	if len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m-main" {
		t.Fatalf("messages = %#v after a stale read, want the previous rows", a.state.Messages)
	}
	if a.state.ActionError != "" {
		t.Fatalf("a stale read reported %q", a.state.ActionError)
	}
}

// TestAppSelectBranchSkipsTheShownBranch checks that asking for the branch the
// window already shows sends no read, so a click on the current row is quiet.
func TestAppSelectBranchSkipsTheShownBranch(t *testing.T) {
	f := &fakeSession{view: &dimui.State{Selected: "s1", SelectedBranch: "main"}}
	a, _, _ := branchApp(f)

	a.state.OnSelectBranch("main")

	if len(f.openBranch) != 0 {
		t.Fatalf("branches opened = %q, want none for the shown branch", f.openBranch)
	}
}

// TestAppSubmitNamesTheShownBranch checks that a submission on a branch the
// window opened is sent on that branch: the App Server opened it and it is the
// conversation the user reads, so the session's default branch must not receive
// the input.
func TestAppSubmitNamesTheShownBranch(t *testing.T) {
	f := &fakeSession{
		sessionBranch: appserver.MainBranchID,
		submitted:     make(chan string, 1),
		view: &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady,
			Messages: []dimui.Message{{ID: "m-main"}}},
		branchViews: map[string]*dimui.State{
			"side": {Selected: "s1", SelectedBranch: "side", Connection: dimui.ConnReady,
				Messages: []dimui.Message{{ID: "m-side"}}},
		},
	}
	a, _, mu := branchApp(f)
	a.state.OnSelectBranch("side")
	waitForState(t, a, mu, "the opened branch shown", func(state *dimui.State) bool {
		return state.SelectedBranch == "side"
	})

	a.state.OnSubmit("hello")
	select {
	case <-f.submitted:
	case <-time.After(2 * time.Second):
		t.Fatal("submit did not reach the session")
	}
	mu.Lock()
	defer mu.Unlock()
	if f.submitBranchCalls != 1 || f.submitBranch != "side" {
		t.Fatalf("submit named %q %d times, want one submit on side",
			f.submitBranch, f.submitBranchCalls)
	}
}

// TestAppStopNamesTheShownBranch checks that a stop on the shown branch names
// that branch, so the turn the user reads ends instead of the default branch's.
func TestAppStopNamesTheShownBranch(t *testing.T) {
	f := &fakeSession{
		stopped: make(chan struct{}, 1),
		view: &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady,
			Messages: []dimui.Message{{ID: "m-main"}}},
		branchViews: map[string]*dimui.State{
			"side": {Selected: "s1", SelectedBranch: "side", Connection: dimui.ConnReady,
				Messages: []dimui.Message{{ID: "m-side"}}},
		},
	}
	a, _, mu := branchApp(f)
	a.state.OnSelectBranch("side")
	waitForState(t, a, mu, "the opened branch shown", func(state *dimui.State) bool {
		return state.SelectedBranch == "side"
	})

	a.state.OnStop()
	select {
	case <-f.stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("stop did not reach the session")
	}
	mu.Lock()
	defer mu.Unlock()
	if f.stopBranchCalls != 1 || f.stopBranch != "side" {
		t.Fatalf("stop named %q %d times, want one stop on side",
			f.stopBranch, f.stopBranchCalls)
	}
}

// TestAppSubmitReadsBackOnTheShownBranch checks that an ambiguous submission on a
// side branch is read back on that same branch. The read-back key is the branch
// the App Server received; naming the default branch would ask about an input
// that was never sent there.
func TestAppSubmitReadsBackOnTheShownBranch(t *testing.T) {
	f := &fakeSession{
		sessionBranch: appserver.MainBranchID,
		submitErr:     &appserver.UnknownResultError{Method: appserver.MutationSessionSubmit},
		submitID:      "sub-1",
		lookupResult:  appserver.InputLookupResult{Status: appserver.InputLookupAccepted},
		view: &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady,
			Messages: []dimui.Message{{ID: "m-main"}}},
		branchViews: map[string]*dimui.State{
			"side": {Selected: "s1", SelectedBranch: "side", Connection: dimui.ConnReady,
				Messages: []dimui.Message{{ID: "m-side"}}},
		},
	}
	a, _, mu := branchApp(f)
	a.state.OnSelectBranch("side")
	waitForState(t, a, mu, "the opened branch shown", func(state *dimui.State) bool {
		return state.SelectedBranch == "side"
	})
	a.state.Draft = "hello"

	a.state.OnSubmit("hello")
	waitForNotice(t, a, mu, "submit.accepted")

	mu.Lock()
	defer mu.Unlock()
	if f.submitBranchCalls != 1 || f.submitBranch != "side" {
		t.Fatalf("submit named %q %d times, want one submit on side",
			f.submitBranch, f.submitBranchCalls)
	}
	if f.lookupBranchCalls != 1 || f.lookupBranch != "side" || f.lookupSubmission != "sub-1" {
		t.Fatalf("read-back = %d on %q for %q, want one lookup on the shown branch side for sub-1",
			f.lookupBranchCalls, f.lookupBranch, f.lookupSubmission)
	}
}

// TestAppSubmitWithoutShownBranchUsesTheSessionBranch checks the default path: a
// window that opened no branch has none of its own to name, so the submit and
// the stop keep the session's own branch and the read-back names that branch.
func TestAppSubmitWithoutShownBranchUsesTheSessionBranch(t *testing.T) {
	f := &fakeSession{
		sessionBranch: appserver.MainBranchID,
		submitErr:     &appserver.UnknownResultError{Method: appserver.MutationSessionSubmit},
		submitID:      "sub-1",
		lookupResult:  appserver.InputLookupResult{Status: appserver.InputLookupAccepted},
		stopped:       make(chan struct{}, 1),
		view:          &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady},
	}
	a, _, mu := branchApp(f)
	a.state.Draft = "hello"

	a.state.OnSubmit("hello")
	waitForNotice(t, a, mu, "submit.accepted")
	a.state.OnStop()
	select {
	case <-f.stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("stop did not reach the session")
	}

	mu.Lock()
	defer mu.Unlock()
	if f.submitBranchCalls != 0 || f.stopBranchCalls != 0 {
		t.Fatalf("the default path named a branch: submits %d, stops %d",
			f.submitBranchCalls, f.stopBranchCalls)
	}
	if f.lookupBranchCalls != 1 || f.lookupBranch != appserver.MainBranchID {
		t.Fatalf("read-back = %d on %q, want one lookup on the session's branch %q",
			f.lookupBranchCalls, f.lookupBranch, appserver.MainBranchID)
	}
}

// TestAppSubmitAfterSessionSwitchForgetsTheOldBranch checks that a branch the
// previous session showed cannot name an action on the new one: a selection
// clears the displayed branch, so the submit falls back to the session's own and
// the old side branch receives nothing.
func TestAppSubmitAfterSessionSwitchForgetsTheOldBranch(t *testing.T) {
	f := &fakeSession{
		sessionBranch: appserver.MainBranchID,
		submitted:     make(chan string, 1),
		view:          &dimui.State{Selected: "s1", SelectedBranch: "main", Connection: dimui.ConnReady},
	}
	a, _, mu := branchApp(f)
	if !a.commitBranch("side", a.beginBranchSelect()) {
		t.Fatal("the test could not show the side branch")
	}
	// A newer session selection forgets the branch the previous one showed.
	a.beginSelect()

	a.state.OnSubmit("hello")
	select {
	case <-f.submitted:
	case <-time.After(2 * time.Second):
		t.Fatal("submit did not reach the session")
	}
	mu.Lock()
	defer mu.Unlock()
	if branch, _ := a.displayBranch(); branch != "" {
		t.Fatalf("displayed branch = %q after a session switch, want none", branch)
	}
	if f.submitBranchCalls != 0 {
		t.Fatalf("submit named %q, want the session's own branch", f.submitBranch)
	}
}
