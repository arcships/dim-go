package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/session"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

func TestStateFromDaemonWithoutProductHome(t *testing.T) {
	state, bridge, err := stateFromDaemon(context.Background(), "")
	if err != nil {
		t.Fatalf("stateFromDaemon: %v", err)
	}
	if state == nil || state.Connection != dimui.ConnUnknown {
		t.Fatalf("state = %#v, want a not-connected state", state)
	}
	if bridge != nil {
		t.Fatal("bridge must be nil without a daemon connection")
	}
	if len(state.Sessions) != 0 || len(state.Messages) != 0 {
		t.Fatalf("state must not invent fixtures: %#v", state)
	}
}

func TestApplyViewNilIsNoop(t *testing.T) {
	state := &dimui.State{
		Sessions:   []dimui.Session{{ID: "s1"}},
		Selected:   "keep",
		Connection: dimui.ConnReady,
		Draft:      "draft",
	}
	applyView(state, nil)
	if state.Selected != "keep" || state.Connection != dimui.ConnReady || state.Draft != "draft" {
		t.Fatalf("state changed on a nil view: %#v", state)
	}
}

// TestApplyViewKeepsWindowState checks that a session's live view replaces only
// the session facts: the catalog, the draft and the callbacks belong to the
// window and stay as they are.
func TestApplyViewKeepsWindowState(t *testing.T) {
	state := &dimui.State{
		Sessions:      []dimui.Session{{ID: "s1"}},
		Draft:         "draft",
		OnSubmit:      func(string) {},
		ActionNotices: []dimui.Notice{{Kind: "submit.accepted"}},
		ActionError:   "window-owned error",
	}
	view := &dimui.State{
		Selected:            "s1",
		Connection:          dimui.ConnReady,
		ConnectionError:     "",
		SessionClosed:       true,
		SessionClosedReason: "done",
		Run:                 dimui.RunRunning,
		Messages:            []dimui.Message{{ID: "m1"}},
		Notices:             []dimui.Notice{{Kind: "approval"}},
	}
	applyView(state, view)

	if state.Selected != "s1" || state.Connection != dimui.ConnReady || state.Run != dimui.RunRunning {
		t.Fatalf("session facts were not applied: %#v", state)
	}
	if !state.SessionClosed || state.SessionClosedReason != "done" {
		t.Fatalf("terminal session health was not applied: %#v", state)
	}
	if len(state.Messages) != 1 || state.Messages[0].ID != "m1" {
		t.Fatalf("messages were not applied: %#v", state.Messages)
	}
	if len(state.Sessions) != 1 || state.Sessions[0].ID != "s1" || state.Draft != "draft" || state.OnSubmit == nil {
		t.Fatalf("window state was overwritten: %#v", state)
	}
	if len(state.ActionNotices) != 1 || state.ActionNotices[0].Kind != "submit.accepted" || state.ActionError != "window-owned error" {
		t.Fatalf("window-owned action outcomes were overwritten: %#v", state)
	}
	if len(state.Notices) != 1 || state.Notices[0].Kind != "approval" {
		t.Fatalf("projection notices were not applied: %#v", state.Notices)
	}
}

// TestApplyViewRejectsStaleSelection checks the refresh guard: a view built
// from an older controller must not replace the window's newer selection or its
// rows, even if a tick obtained it before the selection changed.
func TestApplyViewRejectsStaleSelection(t *testing.T) {
	state := &dimui.State{
		Selected:   "s2",
		Connection: dimui.ConnReady,
		Messages:   []dimui.Message{{ID: "m2"}},
		Draft:      "keep",
		OnSubmit:   func(string) {},
	}
	view := &dimui.State{
		Selected:        "s1",
		Connection:      dimui.ConnError,
		ConnectionError: "old session lost",
		Messages:        []dimui.Message{{ID: "m1"}},
	}
	applyView(state, view)

	if state.Selected != "s2" || state.Connection != dimui.ConnReady || state.ConnectionError != "" {
		t.Fatalf("stale view changed selection/connection: %#v", state)
	}
	if len(state.Messages) != 1 || state.Messages[0].ID != "m2" {
		t.Fatalf("stale view changed rows: %#v", state.Messages)
	}
	if state.Draft != "keep" || state.OnSubmit == nil {
		t.Fatalf("stale view changed window state: %#v", state)
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

// fakeSession stands in for integration.SessionController.
type fakeSession struct {
	view              *dimui.State
	branchViews       map[string]*dimui.State
	summary           appserver.SessionSummary
	submitErr         error
	submitID          string
	stopErr           error
	reattachErr       error
	openBranchErr     error
	endBranchErr      error
	endBranchOutcome  appserver.BranchEndOutcome
	refreshBranchErr  error
	deleteErr         error
	deleted           []string
	approvalErr       error
	approvalResult    appserver.ApprovalRespondOutcome
	lookupResult      appserver.InputLookupResult
	lookupErr         error
	closed            bool
	submitted         chan string
	stopped           chan struct{}
	approved          chan struct{}
	refreshed         chan struct{}
	liveCalls         chan struct{}
	reconnectStart    chan struct{}
	reconnectGate     chan struct{}
	openBranchStart   chan struct{}
	openBranchGate    chan struct{}
	openBranchDone    chan struct{}
	openBranch        []string
	endedBranches     []string
	refreshCalls      int
	shownBranch       string
	sessionBranch     string
	submitText        []string
	submitBranch      string
	submitBranchText  []string
	submitBranchCalls int
	stopBranch        string
	stopBranchCalls   int
	lookupBranchCalls int
	reattachCalls     int
	stopCalls         int
	approvalCalls     int
	approvalBranch    string
	approvalID        string
	approvalChoice    string
	lookupCalls       int
	lookupBranch      string
	lookupSubmission  string
}

func (f *fakeSession) LiveState(summary appserver.SessionSummary) *dimui.State {
	f.summary = summary
	if f.liveCalls != nil {
		select {
		case f.liveCalls <- struct{}{}:
		default:
		}
	}
	return f.view
}

// StateForBranch returns the branch's view when the test supplied one and the
// session's default view otherwise, as integration.StateForBranch falls back to
// the session's own branch.
func (f *fakeSession) StateForBranch(summary appserver.SessionSummary, branchID string) *dimui.State {
	f.shownBranch = branchID
	if branchID == "" {
		return f.LiveState(summary)
	}
	f.summary = summary
	if view, ok := f.branchViews[branchID]; ok {
		return view
	}
	return f.view
}

// OpenBranch records the read and, when the test gates it, holds it until the
// test releases the gate so a selection can change while it is in flight.
func (f *fakeSession) OpenBranch(_ context.Context, branchID string) (session.BranchOpenResult, error) {
	f.openBranch = append(f.openBranch, branchID)
	if f.openBranchStart != nil {
		close(f.openBranchStart)
	}
	if f.openBranchGate != nil {
		<-f.openBranchGate
	}
	if f.openBranchDone != nil {
		close(f.openBranchDone)
	}
	if f.openBranchErr != nil {
		return session.BranchOpenResult{}, f.openBranchErr
	}
	return session.BranchOpenResult{BranchID: branchID}, nil
}

// EndBranch records the branch it was asked to end and returns the test's
// outcome; an empty outcome means the App Server ended it.
func (f *fakeSession) EndBranch(_ context.Context, branchID string) (appserver.BranchEndOutcome, error) {
	f.endedBranches = append(f.endedBranches, branchID)
	if f.endBranchErr != nil {
		return "", f.endBranchErr
	}
	if f.endBranchOutcome == "" {
		return appserver.BranchEndOutcomeEnded, nil
	}
	return f.endBranchOutcome, nil
}

// RefreshBranchDirectory records the directory read and signals the test when
// it supplied a channel.
func (f *fakeSession) RefreshBranchDirectory(context.Context) error {
	f.refreshCalls++
	if f.refreshed != nil {
		f.refreshed <- struct{}{}
	}
	return f.refreshBranchErr
}

// SessionBranchID is the branch the fake submits on; main unless the test says
// otherwise.
func (f *fakeSession) SessionBranchID() string {
	if f.sessionBranch != "" {
		return f.sessionBranch
	}
	return appserver.MainBranchID
}

func (f *fakeSession) Submit(_ context.Context, text string) (appserver.SubmitResult, error) {
	f.submitText = append(f.submitText, text)
	if f.submitted != nil {
		f.submitted <- text
	}
	return appserver.SubmitResult{}, f.submitErr
}

func (f *fakeSession) SubmitWithID(ctx context.Context, text string) (appserver.SubmitResult, string, error) {
	result, err := f.Submit(ctx, text)
	return result, f.submitID, err
}

// SubmitBranchWithID records the branch it was asked to send on and delegates to
// the same submission path, so a test sees which branch the window named and the
// same submit result the App Server would give.
func (f *fakeSession) SubmitBranchWithID(ctx context.Context, branchID, text string) (appserver.SubmitResult, string, error) {
	f.submitBranchCalls++
	f.submitBranch = branchID
	f.submitBranchText = append(f.submitBranchText, text)
	result, err := f.Submit(ctx, text)
	return result, f.submitID, err
}

func (f *fakeSession) LookupInput(_ context.Context, branchID, submissionID string) (appserver.InputLookupResult, error) {
	f.lookupCalls++
	f.lookupBranch, f.lookupSubmission = branchID, submissionID
	return f.lookupResult, f.lookupErr
}

// LookupInputBranch records that the read-back named an explicit branch.
func (f *fakeSession) LookupInputBranch(ctx context.Context, branchID, submissionID string) (appserver.InputLookupResult, error) {
	f.lookupBranchCalls++
	return f.LookupInput(ctx, branchID, submissionID)
}

func (f *fakeSession) Reattach(context.Context) error {
	f.reattachCalls++
	if f.reconnectStart != nil {
		close(f.reconnectStart)
	}
	if f.reconnectGate != nil {
		<-f.reconnectGate
	}
	return f.reattachErr
}

func (f *fakeSession) Stop(context.Context) error {
	f.stopCalls++
	if f.stopped != nil {
		f.stopped <- struct{}{}
	}
	return f.stopErr
}

// StopBranch records the branch the stop named and takes the same stop path.
func (f *fakeSession) StopBranch(ctx context.Context, branchID string) error {
	f.stopBranchCalls++
	f.stopBranch = branchID
	return f.Stop(ctx)
}

func (f *fakeSession) RespondApproval(_ context.Context, branchID, requestID, decision string) (appserver.ApprovalRespondOutcome, error) {
	f.approvalCalls++
	f.approvalBranch, f.approvalID, f.approvalChoice = branchID, requestID, decision
	if f.approved != nil {
		f.approved <- struct{}{}
	}
	if f.approvalErr != nil {
		return "", f.approvalErr
	}
	if f.approvalResult == "" {
		return appserver.ApprovalOutcomeResponded, nil
	}
	return f.approvalResult, nil
}

func (f *fakeSession) Delete(context.Context) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, f.summary.SessionID)
	return nil
}

func (f *fakeSession) Close(context.Context) error {
	f.closed = true
	return nil
}

func TestSessionBridgeWithoutSession(t *testing.T) {
	b := &sessionBridge{}
	ctx := context.Background()
	if err := b.Submit(ctx, "hello"); !errors.Is(err, errNoSession) {
		t.Fatalf("Submit error = %v, want errNoSession", err)
	}
	if err := b.Stop(ctx); !errors.Is(err, errNoSession) {
		t.Fatalf("Stop error = %v, want errNoSession", err)
	}
	if view := b.LiveState(); view != nil {
		t.Fatalf("LiveState = %#v without a session", view)
	}
}

func TestSessionBridgeSubmitAndStop(t *testing.T) {
	f := &fakeSession{}
	b := &sessionBridge{controller: f}
	ctx := context.Background()

	if err := b.Submit(ctx, "hello"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if len(f.submitText) != 1 || f.submitText[0] != "hello" {
		t.Fatalf("submitted %q, want [hello]", f.submitText)
	}
	if err := b.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	boom := errors.New("boom")
	f.submitErr, f.stopErr = boom, boom
	if err := b.Submit(ctx, "fail"); !errors.Is(err, boom) {
		t.Fatalf("Submit error = %v, want boom", err)
	}
	if err := b.Stop(ctx); !errors.Is(err, boom) {
		t.Fatalf("Stop error = %v, want boom", err)
	}
}

// TestSessionBridgeActionsRejectStaleSession checks that submit, stop and
// reattach carry the selected session id through the bridge: an action captured
// for an older selection must not reach the newly attached controller.
func TestSessionBridgeActionsRejectStaleSession(t *testing.T) {
	f := &fakeSession{}
	b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	ctx := context.Background()

	if err := b.SubmitFor(ctx, "s2", "hello"); !errors.Is(err, errStaleSelection) {
		t.Fatalf("SubmitFor stale error = %v, want errStaleSelection", err)
	}
	if err := b.StopFor(ctx, "s2"); !errors.Is(err, errStaleSelection) {
		t.Fatalf("StopFor stale error = %v, want errStaleSelection", err)
	}
	if err := b.reattachFor(ctx, "s2"); !errors.Is(err, errStaleSelection) {
		t.Fatalf("reattachFor stale error = %v, want errStaleSelection", err)
	}
	if _, err := b.RespondApprovalFor(ctx, "s2", "main", "req-1", appserver.DecisionAllow); !errors.Is(err, errStaleSelection) {
		t.Fatalf("RespondApprovalFor stale error = %v, want errStaleSelection", err)
	}
	if len(f.submitText) != 0 || f.stopCalls != 0 || f.reattachCalls != 0 || f.approvalCalls != 0 {
		t.Fatalf("stale actions reached the controller: submit=%q stop=%d reattach=%d approvals=%d",
			f.submitText, f.stopCalls, f.reattachCalls, f.approvalCalls)
	}

	if err := b.SubmitFor(ctx, "s1", "hello"); err != nil {
		t.Fatalf("SubmitFor: %v", err)
	}
	if err := b.StopFor(ctx, "s1"); err != nil {
		t.Fatalf("StopFor: %v", err)
	}
	if err := b.reattachFor(ctx, "s1"); err != nil {
		t.Fatalf("reattachFor: %v", err)
	}
	if _, err := b.RespondApprovalFor(ctx, "s1", "main", "req-1", appserver.DecisionAllow); err != nil {
		t.Fatalf("RespondApprovalFor: %v", err)
	}
	if len(f.submitText) != 1 || f.stopCalls != 1 || f.reattachCalls != 1 || f.approvalCalls != 1 {
		t.Fatalf("selected actions did not reach the controller: submit=%q stop=%d reattach=%d approvals=%d",
			f.submitText, f.stopCalls, f.reattachCalls, f.approvalCalls)
	}
}

func TestSessionBridgeOpenSwapsAndClosesPrevious(t *testing.T) {
	old := &fakeSession{}
	next := &fakeSession{}
	b := &sessionBridge{controller: old}
	b.openSession = func(_ context.Context, _ *supervisor.Supervisor, id string) (sessionActions, error) {
		if id != "s2" {
			t.Fatalf("opened %q, want s2", id)
		}
		return next, nil
	}

	if err := b.open(context.Background(), dimui.Session{ID: "s2", Title: "Two"}); err != nil {
		t.Fatalf("open: %v", err)
	}
	if !old.closed {
		t.Error("the previous session was not closed")
	}
	if b.controller != next {
		t.Fatal("the current controller was not swapped")
	}
	if b.summary.SessionID != "s2" || b.summary.Title == nil || *b.summary.Title != "Two" {
		t.Fatalf("summary = %#v, want s2 with its title", b.summary)
	}
}

// TestAppBindsCallbacks checks that the window state exposes every action the
// shell draws, so a click or a shortcut reaches the controller.
func TestAppRefreshesLiveState(t *testing.T) {
	f := &fakeSession{view: &dimui.State{Selected: "s1", Connection: dimui.ConnReady, Messages: []dimui.Message{{ID: "m1"}}}, liveCalls: make(chan struct{}, 4)}
	a := &app{state: &dimui.State{Draft: "keep"}, bridge: &sessionBridge{controller: f}}
	updated := make(chan struct{}, 1)
	a.updateHook = func(fn func(*dimui.State)) {
		fn(a.state)
		select {
		case updated <- struct{}{}:
		default:
		}
	}
	a.startRefresh(5 * time.Millisecond)
	defer a.stopRefresh()
	select {
	case <-updated:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("refresh did not update state")
	}
	if a.state.Selected != "s1" || len(a.state.Messages) != 1 || a.state.Draft != "keep" {
		t.Fatalf("state = %#v, refresh did not apply view safely", a.state)
	}
}

// TestAppRefreshDoesNotClobberNewSelection checks that a tick which read the old
// controller before a new selection was recorded cannot publish that old view
// over the new conversation.
func TestAppRefreshDoesNotClobberNewSelection(t *testing.T) {
	f := &fakeSession{view: &dimui.State{
		Selected:   "s1",
		Connection: dimui.ConnReady,
		Messages:   []dimui.Message{{ID: "m1"}},
	}}
	a := &app{
		state:  &dimui.State{Selected: "s2", Messages: []dimui.Message{{ID: "m2"}}},
		bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}},
	}
	updated := make(chan struct{}, 1)
	a.updateHook = func(fn func(*dimui.State)) {
		fn(a.state)
		select {
		case updated <- struct{}{}:
		default:
		}
	}
	a.startRefresh(5 * time.Millisecond)
	defer a.stopRefresh()
	select {
	case <-updated:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("refresh did not tick")
	}
	if a.state.Selected != "s2" || len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m2" {
		t.Fatalf("old controller overwrote the new selection: %#v", a.state)
	}
}

func TestAppBindsCallbacks(t *testing.T) {
	a := &app{state: &dimui.State{}}
	a.bind()
	if a.state.OnSelect == nil || a.state.OnSelectBranch == nil || a.state.OnSubmit == nil || a.state.OnStop == nil ||
		a.state.OnReconnect == nil || a.state.OnApproval == nil {
		t.Fatalf("callbacks not bound: %#v", a.state)
	}
}

func TestAppReconnectStaleFailureDoesNotOverwriteNewSelection(t *testing.T) {
	boom := errors.New("stale reconnect")
	started := make(chan struct{})
	gate := make(chan struct{})
	f := &fakeSession{reattachErr: boom, reconnectStart: started, reconnectGate: gate}
	bridge := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	a := &app{state: &dimui.State{Connection: dimui.ConnError, Selected: "s1"}, bridge: bridge}
	a.bind()
	updates := make(chan struct{}, 2)
	a.updateHook = func(fn func(*dimui.State)) { fn(a.state); updates <- struct{}{} }
	a.state.OnReconnect()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("reattach did not start")
	}
	a.beginSelect() // invalidate the reconnect generation as if the user chose another session
	a.state.Connection = dimui.ConnReady
	close(gate)
	select {
	case <-updates:
	case <-time.After(2 * time.Second):
		t.Fatal("initial reconnect update missing")
	}
	select {
	case <-updates:
		t.Fatal("stale reconnect failure updated the new selection")
	case <-time.After(50 * time.Millisecond):
	}
	if a.state.Connection != dimui.ConnReady || a.state.ConnectionError != "" {
		t.Fatalf("stale failure changed state: %#v", a.state)
	}
}

// TestAppReconnectFailureDoesNotOverwriteReady checks that a stale reconnect
// failure cannot turn a session that another attempt already healed back into
// an error.
func TestAppReconnectFailureDoesNotOverwriteReady(t *testing.T) {
	boom := errors.New("stale reconnect")
	started := make(chan struct{})
	gate := make(chan struct{})
	f := &fakeSession{reattachErr: boom, reconnectStart: started, reconnectGate: gate}
	bridge := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	a := &app{state: &dimui.State{Connection: dimui.ConnError, Selected: "s1"}, bridge: bridge}
	a.bind()
	updates := make(chan struct{}, 2)
	a.updateHook = func(fn func(*dimui.State)) { fn(a.state); updates <- struct{}{} }
	a.state.OnReconnect()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("reattach did not start")
	}
	a.state.Connection = dimui.ConnReady
	close(gate)
	for i := 0; i < 2; i++ {
		select {
		case <-updates:
		case <-time.After(2 * time.Second):
			t.Fatal("reconnect updates missing")
		}
	}
	if a.state.Connection != dimui.ConnReady || a.state.ConnectionError != "" || a.state.ConversationLoading {
		t.Fatalf("stale reconnect failure changed a ready session: %#v", a.state)
	}
}

func TestAppReconnectReattachesAndRefreshes(t *testing.T) {
	f := &fakeSession{view: &dimui.State{Connection: dimui.ConnReady, Selected: "s1"}}
	bridge := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	a := &app{state: &dimui.State{Connection: dimui.ConnError, Selected: "s1"}, bridge: bridge}
	a.bind()
	updates := make(chan struct{}, 2)
	a.updateHook = func(fn func(*dimui.State)) {
		fn(a.state)
		updates <- struct{}{}
	}
	a.state.OnReconnect()
	deadline := time.After(2 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-updates:
		case <-deadline:
			t.Fatal("reconnect did not refresh")
		}
	}
	if a.state.Connection != dimui.ConnReady {
		t.Fatalf("connection = %v, want ready", a.state.Connection)
	}
}

// TestAppDeleteRemovesTheConfirmedRow checks that a delete removes the catalog
// row only after the controller reports success, and leaves the selection alone
// when another session is displayed.
func TestAppDeleteRemovesTheConfirmedRow(t *testing.T) {
	f := &fakeSession{summary: appserver.SessionSummary{SessionID: "s1"}}
	bridge := &sessionBridge{controller: f, summary: f.summary}
	a := &app{
		state: &dimui.State{
			Sessions: []dimui.Session{{ID: "s1", Title: "one"}, {ID: "s2", Title: "two"}},
			Selected: "s1",
		},
		bridge: bridge,
	}
	states := make(chan dimui.State, 8)
	a.updateHook = func(fn func(*dimui.State)) {
		fn(a.state)
		select {
		case states <- *a.state:
		default:
		}
	}
	a.bind()
	// The thread action always names the displayed session; deleting another
	// row would target a controller the window does not hold.
	a.state.OnDeleteSession("s1")

	deadline := time.After(2 * time.Second)
	for {
		select {
		case state := <-states:
			if len(state.Sessions) == 1 && state.Sessions[0].ID == "s2" {
				if state.Selected == "s1" {
					t.Fatalf("selection = %q after deleting s1", state.Selected)
				}
				return
			}
		case <-deadline:
			t.Fatal("delete did not remove the row")
		}
	}
}

// TestAppDeleteFailureKeepsTheRow checks that a rejected delete keeps the row
// and reports the failure through the banner.
func TestAppDeleteFailureKeepsTheRow(t *testing.T) {
	f := &fakeSession{summary: appserver.SessionSummary{SessionID: "s1"}, deleteErr: errors.New("session active")}
	a := &app{
		state:  &dimui.State{Sessions: []dimui.Session{{ID: "s1", Title: "one"}}, Selected: "s1"},
		bridge: &sessionBridge{controller: f, summary: f.summary},
	}
	states := make(chan dimui.State, 8)
	a.updateHook = func(fn func(*dimui.State)) {
		fn(a.state)
		select {
		case states <- *a.state:
		default:
		}
	}
	a.bind()
	a.state.OnDeleteSession("s1")

	deadline := time.After(2 * time.Second)
	for {
		select {
		case state := <-states:
			if state.ActionError != "" {
				if len(state.Sessions) != 1 || state.Selected != "s1" {
					t.Fatalf("failed delete changed the window: sessions=%d selected=%q", len(state.Sessions), state.Selected)
				}
				return
			}
		case <-deadline:
			t.Fatal("failed delete reported nothing")
		}
	}
}

// TestAppSubmitDoesNotTargetStaleController checks that the window captures the
// selected id before the background submit and refuses a controller that no
// longer belongs to it.
func TestAppSubmitDoesNotTargetStaleController(t *testing.T) {
	f := &fakeSession{}
	bridge := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	a := &app{state: &dimui.State{Selected: "s2"}, bridge: bridge}
	updated := make(chan struct{}, 1)
	a.updateHook = func(fn func(*dimui.State)) {
		fn(a.state)
		select {
		case updated <- struct{}{}:
		default:
		}
	}
	a.bind()
	a.state.OnSubmit("hello")

	select {
	case <-updated:
	case <-time.After(2 * time.Second):
		t.Fatal("stale submit did not publish an error")
	}
	if len(f.submitText) != 0 {
		t.Fatalf("stale submit reached the wrong controller: %q", f.submitText)
	}
}

// TestAppSubmitReachesSession checks the submit callback the composer invokes.
func TestAppSubmitReachesSession(t *testing.T) {
	sent := make(chan string, 1)
	f := &fakeSession{submitted: sent}
	a := &app{state: &dimui.State{}, bridge: &sessionBridge{controller: f}}
	a.bind()

	a.state.OnSubmit("hello")
	select {
	case got := <-sent:
		if got != "hello" {
			t.Fatalf("submitted %q, want hello", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("submit did not reach the session")
	}
}

// TestAppStopReachesSession checks the stop callback the toolbar's entry
// invokes.
func TestAppStopReachesSession(t *testing.T) {
	stopped := make(chan struct{}, 1)
	f := &fakeSession{stopped: stopped}
	a := &app{state: &dimui.State{}, bridge: &sessionBridge{controller: f}}
	a.bind()

	a.state.OnStop()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("stop did not reach the session")
	}
}

// TestAppSubmitUnknownResultKeepsDraftAndDoesNotRetry verifies that an
// ambiguous App Server mutation is surfaced as-is: the app neither clears the
// composer nor retries the request or invents a timeline row.
func TestAppSubmitUnknownResultKeepsDraftAndDoesNotRetry(t *testing.T) {
	unknown := &appserver.UnknownResultError{Method: appserver.MutationSessionSubmit}
	sent := make(chan string, 1)
	f := &fakeSession{submitted: sent, submitErr: unknown}
	a := &app{state: &dimui.State{Draft: "hello"}, bridge: &sessionBridge{controller: f}}
	a.bind()

	a.state.OnSubmit("hello")
	select {
	case got := <-sent:
		if got != "hello" {
			t.Fatalf("submitted %q, want hello", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("submit did not reach the session")
	}
	if len(f.submitText) != 1 {
		t.Fatalf("submit count = %d, want exactly one", len(f.submitText))
	}
	if a.state.Draft != "hello" {
		t.Fatalf("draft = %q after unknown result, want it retained", a.state.Draft)
	}
}

// TestAppSelectOpensSession checks the selection callback: the sidebar's chosen
// id reaches the opener and becomes the current session.
// TestAppStaleSelectionCannotOverwriteNewerSelection verifies that a slower
// attachment cannot publish its view after a newer sidebar choice. Both
// attachments still go through the controller opener; the guard only protects
// window state from stale results.
func TestAppStaleSelectionCannotOverwriteNewerSelection(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	opened := make(chan string, 2)
	first := &fakeSession{view: &dimui.State{Selected: "s1", Connection: dimui.ConnReady}}
	second := &fakeSession{view: &dimui.State{Selected: "s2", Connection: dimui.ConnReady}}
	b := &sessionBridge{}
	b.openSession = func(_ context.Context, _ *supervisor.Supervisor, id string) (sessionActions, error) {
		opened <- id
		if id == "s1" {
			close(firstStarted)
			<-releaseFirst
			return first, nil
		}
		return second, nil
	}
	a := &app{
		state:  &dimui.State{Sessions: []dimui.Session{{ID: "s1"}, {ID: "s2"}}},
		bridge: b,
	}
	updates := make(chan struct{}, 2)
	a.updateHook = func(fn func(*dimui.State)) {
		fn(a.state)
		updates <- struct{}{}
	}
	a.bind()
	a.state.Selected = "s1"
	a.state.OnSelect("s1")
	select {
	case got := <-opened:
		if got != "s1" {
			t.Fatalf("first open = %q, want s1", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first selection did not start")
	}
	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first attachment did not reach opener")
	}
	// The second selection supersedes the first while its attachment is in flight.
	a.state.Selected = "s2"
	a.state.OnSelect("s2")

	// The opener serializes attachments; release the first, then wait for the
	// second controller's update callback, which synchronizes all state reads.
	close(releaseFirst)
	select {
	case got := <-opened:
		if got != "s2" {
			t.Fatalf("second open = %q, want s2", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second selection did not reach opener")
	}
	select {
	case <-updates:
	case <-time.After(2 * time.Second):
		t.Fatal("new selection did not publish a view")
	}
	if b.summary.SessionID != "s2" || a.state.Selected != "s2" || a.state.Connection != dimui.ConnReady {
		t.Fatalf("new selection did not win: state=%#v summary=%#v", a.state, b.summary)
	}
}

// TestAppSelectFailureRestoresPreviousSelection checks that a failed session
// open does not leave the new title over the previous conversation: the
// controller still holds the old session, so its view is restored and the error
// is surfaced separately.
// TestAppExpiredOpenFailureRestoresCurrentController covers an older open that
// completes while a newer selection is in flight: when the newer open fails, the
// window must restore the controller that is actually current, not a view
// captured before that older open swapped it.
func TestAppExpiredOpenFailureRestoresCurrentController(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	first := &fakeSession{view: &dimui.State{
		Selected:   "s1",
		Connection: dimui.ConnReady,
		Messages:   []dimui.Message{{ID: "m1"}},
	}}
	openErr := errors.New("s2 open failed")
	b := &sessionBridge{}
	b.openSession = func(_ context.Context, _ *supervisor.Supervisor, id string) (sessionActions, error) {
		if id == "s1" {
			close(started)
			<-release
			return first, nil
		}
		return nil, openErr
	}
	a := &app{
		state: &dimui.State{
			Sessions: []dimui.Session{{ID: "s1"}, {ID: "s2"}},
		},
		bridge: b,
	}
	updated := make(chan struct{}, 1)
	a.updateHook = func(fn func(*dimui.State)) {
		fn(a.state)
		select {
		case updated <- struct{}{}:
		default:
		}
	}
	a.bind()

	a.state.Selected = "s1"
	a.state.OnSelect("s1")
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first open did not start")
	}
	a.state.Selected = "s2"
	a.state.OnSelect("s2")
	close(release)

	select {
	case <-updated:
	case <-time.After(2 * time.Second):
		t.Fatal("failed newer open did not publish")
	}
	if a.state.Selected != "s1" || len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m1" {
		t.Fatalf("expired open failure left a mismatched view: %#v", a.state)
	}
	if a.state.ActionError == "" {
		t.Fatal("failed newer open was not surfaced")
	}
	if len(a.state.ActionNotices) != 1 || a.state.ActionNotices[0].Kind != "select.error" {
		t.Fatalf("selection failure was not recorded: %#v", a.state.ActionNotices)
	}
}

func TestAppSelectFailureRestoresPreviousSelection(t *testing.T) {
	previous := &dimui.State{
		Selected:   "s1",
		Connection: dimui.ConnReady,
		Messages:   []dimui.Message{{ID: "m1"}},
	}
	f := &fakeSession{view: previous}
	b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	b.openSession = func(context.Context, *supervisor.Supervisor, string) (sessionActions, error) {
		return nil, errors.New("open failed")
	}
	a := &app{
		state: &dimui.State{
			Sessions:   []dimui.Session{{ID: "s1", Title: "One"}, {ID: "s2", Title: "Two"}},
			Selected:   "s2",
			Connection: dimui.ConnReady,
			Messages:   []dimui.Message{{ID: "m2"}},
			Draft:      "keep",
		},
		bridge: b,
	}
	updated := make(chan struct{}, 1)
	a.updateHook = func(fn func(*dimui.State)) {
		fn(a.state)
		select {
		case updated <- struct{}{}:
		default:
		}
	}
	a.bind()
	a.state.OnSelect("s2")

	select {
	case <-updated:
	case <-time.After(2 * time.Second):
		t.Fatal("failed selection did not publish its error")
	}
	if a.state.Selected != "s1" || len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m1" {
		t.Fatalf("failed selection left a mismatched view: %#v", a.state)
	}
	if a.state.ConversationLoading {
		t.Error("loading is still true after a failed selection")
	}
	if a.state.Connection != dimui.ConnReady || a.state.ConnectionError != "" {
		t.Fatalf("failed selection changed the previous connection: %#v", a.state)
	}
	if a.state.ActionError == "" {
		t.Fatalf("selection error was not surfaced: %#v", a.state)
	}
	if len(a.state.ActionNotices) != 1 || a.state.ActionNotices[0].Kind != "select.error" {
		t.Fatalf("selection outcome was not recorded: %#v", a.state.ActionNotices)
	}
	if a.state.Draft != "keep" || a.state.OnSubmit == nil || len(a.state.Sessions) != 2 {
		t.Fatalf("window state was lost: %#v", a.state)
	}
}

func TestAppSelectOpensSession(t *testing.T) {
	opened := make(chan string, 1)
	live := make(chan struct{}, 1)
	next := &fakeSession{
		view:      &dimui.State{Selected: "s2", Connection: dimui.ConnReady},
		liveCalls: live,
	}
	b := &sessionBridge{}
	b.openSession = func(_ context.Context, _ *supervisor.Supervisor, id string) (sessionActions, error) {
		opened <- id
		return next, nil
	}
	a := &app{
		state:  &dimui.State{Sessions: []dimui.Session{{ID: "s1"}, {ID: "s2"}}},
		bridge: b,
	}
	a.bind()

	a.state.OnSelect("s2")
	select {
	case id := <-opened:
		if id != "s2" {
			t.Fatalf("opened %q, want s2", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("selection did not open a session")
	}
	select {
	case <-live:
	case <-time.After(2 * time.Second):
		t.Fatal("selection did not read the session's live view")
	}
	if b.summary.SessionID != "s2" {
		t.Fatalf("current summary = %#v, want s2", b.summary)
	}
}

func TestAppSubmitAndStopPublishOutcomes(t *testing.T) {
	boom := errors.New("mutation failed")
	f := &fakeSession{submitErr: boom, stopErr: boom}
	a := &app{state: &dimui.State{Draft: "hello"}, bridge: &sessionBridge{controller: f}}
	a.bind()
	var stateMu sync.Mutex
	a.updateHook = func(fn func(*dimui.State)) { stateMu.Lock(); defer stateMu.Unlock(); fn(a.state) }
	a.state.OnSubmit("hello")
	a.state.OnStop()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stateMu.Lock()
		count := len(a.state.ActionNotices)
		stateMu.Unlock()
		if count >= 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	if len(a.state.ActionNotices) < 2 {
		t.Fatalf("action notices = %#v, want submit and stop errors", a.state.ActionNotices)
	}
	kinds := map[string]bool{}
	for _, notice := range a.state.ActionNotices {
		kinds[notice.Kind] = true
	}
	if !kinds["submit.error"] || !kinds["stop.error"] {
		t.Fatalf("notice kinds = %#v", kinds)
	}
	if a.state.ActionError == "" {
		t.Fatal("action error was not surfaced")
	}
	if a.state.Draft != "hello" {
		t.Fatalf("draft = %q, want retained after failed submit", a.state.Draft)
	}
}

// approvalApp builds a window whose selected session holds one pending
// approval, plus the mutex that guards the state its background callbacks
// update.
func approvalApp(f *fakeSession) (*app, *sync.Mutex) {
	bridge := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	a := &app{
		state: &dimui.State{
			Selected: "s1",
			Messages: []dimui.Message{{
				ID:       "approval:req-1",
				Author:   dimui.AuthorApproval,
				Content:  "exec — deploy",
				Approval: &dimui.Approval{BranchID: "main", RequestID: "req-1"},
			}},
		},
		bridge: bridge,
	}
	a.bind()
	var mu sync.Mutex
	a.updateHook = func(fn func(*dimui.State)) {
		mu.Lock()
		defer mu.Unlock()
		fn(a.state)
	}
	return a, &mu
}

// waitForNotice waits until the window publishes a window-owned outcome of the
// given kind.
func waitForNotice(t *testing.T, a *app, mu *sync.Mutex, kind string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		found := false
		for _, notice := range a.state.ActionNotices {
			if notice.Kind == kind {
				found = true
			}
		}
		mu.Unlock()
		if found {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("window did not publish %q", kind)
}

// TestApprovalDecisionMapping pins the shell's decision vocabulary to the wire
// values approval.respond accepts, and rejects anything else before a send.
func TestApprovalDecisionMapping(t *testing.T) {
	cases := []struct {
		decision dimui.ApprovalDecision
		want     string
		ok       bool
	}{
		{dimui.ApprovalAllow, appserver.DecisionAllow, true},
		{dimui.ApprovalDeny, appserver.DecisionDeny, true},
		{dimui.ApprovalDecision("allow_all"), "", false},
		{dimui.ApprovalDecision(""), "", false},
	}
	for _, tc := range cases {
		got, ok := approvalDecision(tc.decision)
		if got != tc.want || ok != tc.ok {
			t.Errorf("approvalDecision(%q) = %q, %v; want %q, %v", tc.decision, got, ok, tc.want, tc.ok)
		}
	}
}

// TestAppRespondApprovalReachesSessionOnce checks the callback the approval
// row invokes: one click sends exactly one folded decision through the
// controller of the selected session, and the row survives because only the
// session projection may resolve it.
func TestAppRespondApprovalReachesSessionOnce(t *testing.T) {
	f := &fakeSession{approved: make(chan struct{}, 1)}
	a, mu := approvalApp(f)

	a.state.OnApproval("main", "req-1", dimui.ApprovalAllow)
	select {
	case <-f.approved:
	case <-time.After(2 * time.Second):
		t.Fatal("approval decision did not reach the session")
	}
	waitForNotice(t, a, mu, "approval.responded")

	if f.approvalCalls != 1 {
		t.Fatalf("approval.respond ran %d times for one click, want once", f.approvalCalls)
	}
	if f.approvalBranch != "main" || f.approvalID != "req-1" || f.approvalChoice != appserver.DecisionAllow {
		t.Fatalf("sent (%q, %q, %q), want (main, req-1, allow)",
			f.approvalBranch, f.approvalID, f.approvalChoice)
	}
	mu.Lock()
	rows, actionError := len(a.state.Messages), a.state.ActionError
	mu.Unlock()
	if rows != 1 {
		t.Fatalf("%d rows after an accepted decision, want the approval kept until the projection resolves it", rows)
	}
	if actionError != "" {
		t.Fatalf("action error %q after an accepted decision", actionError)
	}
}

// TestAppRespondApprovalDenyMapsToWire checks the Deny entry's decision too.
func TestAppRespondApprovalDenyMapsToWire(t *testing.T) {
	f := &fakeSession{approved: make(chan struct{}, 1)}
	a, _ := approvalApp(f)

	a.state.OnApproval("main", "req-1", dimui.ApprovalDeny)
	select {
	case <-f.approved:
	case <-time.After(2 * time.Second):
		t.Fatal("deny decision did not reach the session")
	}
	if f.approvalChoice != appserver.DecisionDeny {
		t.Fatalf("sent decision %q, want %q", f.approvalChoice, appserver.DecisionDeny)
	}
}

// TestAppRespondApprovalFailureStaysVisible checks that a rejected decision is
// published as an error and the approval stays on screen: nothing claims the
// decision was accepted.
func TestAppRespondApprovalFailureStaysVisible(t *testing.T) {
	boom := errors.New("approval.respond failed")
	f := &fakeSession{approvalErr: boom, approved: make(chan struct{}, 1)}
	a, mu := approvalApp(f)

	a.state.OnApproval("main", "req-1", dimui.ApprovalAllow)
	select {
	case <-f.approved:
	case <-time.After(2 * time.Second):
		t.Fatal("approval decision did not reach the session")
	}
	waitForNotice(t, a, mu, "approval.error")

	mu.Lock()
	defer mu.Unlock()
	if a.state.ActionError == "" {
		t.Fatal("failed decision was not surfaced")
	}
	if len(a.state.Messages) != 1 || a.state.Messages[0].Approval == nil {
		t.Fatalf("approval row disappeared without a projection update: %#v", a.state.Messages)
	}
	for _, notice := range a.state.ActionNotices {
		if notice.Kind == "approval.responded" {
			t.Fatalf("a rejected decision was reported as accepted: %#v", a.state.ActionNotices)
		}
	}
}

// TestAppRespondApprovalCancelledIsNotSuccess checks the §12 already-resolved
// outcome: it is recorded as cancelled, never as a decision the App Server
// applied.
func TestAppRespondApprovalCancelledIsNotSuccess(t *testing.T) {
	f := &fakeSession{
		approvalResult: appserver.ApprovalOutcomeCancelled,
		approved:       make(chan struct{}, 1),
	}
	a, mu := approvalApp(f)

	a.state.OnApproval("main", "req-1", dimui.ApprovalAllow)
	select {
	case <-f.approved:
	case <-time.After(2 * time.Second):
		t.Fatal("approval decision did not reach the session")
	}
	waitForNotice(t, a, mu, "approval.cancelled")

	mu.Lock()
	defer mu.Unlock()
	if a.state.ActionError != "" {
		t.Fatalf("cancelled outcome surfaced as an error: %q", a.state.ActionError)
	}
	for _, notice := range a.state.ActionNotices {
		if notice.Kind == "approval.responded" {
			t.Fatalf("a cancelled approval was reported as responded: %#v", a.state.ActionNotices)
		}
	}
}

// TestAppRespondApprovalRejectsStaleSelection checks that a decision clicked on
// an older conversation cannot reach the newly attached session.
func TestAppRespondApprovalRejectsStaleSelection(t *testing.T) {
	f := &fakeSession{approved: make(chan struct{}, 1)}
	a, mu := approvalApp(f)
	a.state.Selected = "s2"

	a.state.OnApproval("main", "req-1", dimui.ApprovalAllow)
	waitForNotice(t, a, mu, "approval.error")

	if f.approvalCalls != 0 {
		t.Fatalf("stale approval reached the controller %d times", f.approvalCalls)
	}
}

// TestAppRespondApprovalRejectsIncompleteIdentity checks that a decision with
// no request identity or an unknown vocabulary is refused locally instead of
// being sent as a guess.
func TestAppRespondApprovalRejectsIncompleteIdentity(t *testing.T) {
	f := &fakeSession{approved: make(chan struct{}, 1)}
	a, mu := approvalApp(f)

	a.state.OnApproval("", "req-1", dimui.ApprovalAllow)
	a.state.OnApproval("main", "", dimui.ApprovalAllow)
	a.state.OnApproval("main", "req-1", dimui.ApprovalDecision("maybe"))

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := len(a.state.ActionNotices)
		mu.Unlock()
		if count >= 3 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(a.state.ActionNotices) != 3 {
		t.Fatalf("notices = %#v, want one error per refused decision", a.state.ActionNotices)
	}
	for _, notice := range a.state.ActionNotices {
		if notice.Kind != "approval.error" {
			t.Fatalf("refused decision recorded as %q", notice.Kind)
		}
	}
	if f.approvalCalls != 0 {
		t.Fatalf("an invalid decision reached the controller %d times", f.approvalCalls)
	}
}
