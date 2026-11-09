package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
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
		Sessions: []dimui.Session{{ID: "s1"}},
		Draft:    "draft",
		OnSubmit: func(string) {},
	}
	view := &dimui.State{
		Selected:        "s1",
		Connection:      dimui.ConnReady,
		ConnectionError: "",
		Run:             dimui.RunRunning,
		Messages:        []dimui.Message{{ID: "m1"}},
	}
	applyView(state, view)

	if state.Selected != "s1" || state.Connection != dimui.ConnReady || state.Run != dimui.RunRunning {
		t.Fatalf("session facts were not applied: %#v", state)
	}
	if len(state.Messages) != 1 || state.Messages[0].ID != "m1" {
		t.Fatalf("messages were not applied: %#v", state.Messages)
	}
	if len(state.Sessions) != 1 || state.Sessions[0].ID != "s1" || state.Draft != "draft" || state.OnSubmit == nil {
		t.Fatalf("window state was overwritten: %#v", state)
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
	view           *dimui.State
	summary        appserver.SessionSummary
	submitErr      error
	stopErr        error
	reattachErr    error
	closed         bool
	submitted      chan string
	stopped        chan struct{}
	liveCalls      chan struct{}
	reconnectStart chan struct{}
	reconnectGate  chan struct{}
	submitText     []string
	reattachCalls  int
	stopCalls      int
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

func (f *fakeSession) Submit(_ context.Context, text string) (appserver.SubmitResult, error) {
	f.submitText = append(f.submitText, text)
	if f.submitted != nil {
		f.submitted <- text
	}
	return appserver.SubmitResult{}, f.submitErr
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
	if len(f.submitText) != 0 || f.stopCalls != 0 || f.reattachCalls != 0 {
		t.Fatalf("stale actions reached the controller: submit=%q stop=%d reattach=%d",
			f.submitText, f.stopCalls, f.reattachCalls)
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
	if len(f.submitText) != 1 || f.stopCalls != 1 || f.reattachCalls != 1 {
		t.Fatalf("selected actions did not reach the controller: submit=%q stop=%d reattach=%d",
			f.submitText, f.stopCalls, f.reattachCalls)
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

func TestAppBindsCallbacks(t *testing.T) {
	a := &app{state: &dimui.State{}}
	a.bind()
	if a.state.OnSelect == nil || a.state.OnSubmit == nil || a.state.OnStop == nil || a.state.OnReconnect == nil {
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
		count := len(a.state.Notices)
		stateMu.Unlock()
		if count >= 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	if len(a.state.Notices) < 2 {
		t.Fatalf("notices = %#v, want submit and stop errors", a.state.Notices)
	}
	kinds := map[string]bool{}
	for _, notice := range a.state.Notices {
		kinds[notice.Kind] = true
	}
	if !kinds["submit.error"] || !kinds["stop.error"] {
		t.Fatalf("notice kinds = %#v", kinds)
	}
	if a.state.Draft != "hello" {
		t.Fatalf("draft = %q, want retained after failed submit", a.state.Draft)
	}
}
