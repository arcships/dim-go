package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// createApp builds a window whose bridge takes the test's create and catalog
// hooks, plus the mutex guarding the state its background callbacks update.
func createApp(b *sessionBridge, state *dimui.State) (*app, *sync.Mutex) {
	a := &app{state: state, bridge: b}
	var mu sync.Mutex
	a.updateHook = func(fn func(*dimui.State)) {
		mu.Lock()
		defer mu.Unlock()
		fn(a.state)
	}
	a.bind()
	return a, &mu
}

// waitForState waits until cond holds for the window state the update hook
// guards.
func waitForState(t *testing.T, a *app, mu *sync.Mutex, what string, cond func(*dimui.State) bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := cond(a.state)
		mu.Unlock()
		if ok {
			return
		}
		time.Sleep(time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	t.Fatalf("window state never reached %s: %#v", what, a.state)
}

// TestAppCreateSessionSelectsNewSession checks the session.create path: one
// request, then the catalog the App Server served is the sidebar's list and the
// created session is selected.
func TestAppCreateSessionSelectsNewSession(t *testing.T) {
	fresh := &fakeSession{view: &dimui.State{
		Selected:   "s3",
		Connection: dimui.ConnReady,
		Messages:   []dimui.Message{{ID: "m3", Content: "new session row"}},
	}}
	var created int
	var gotParams appserver.CreateParams
	b := &sessionBridge{}
	b.createSession = func(_ context.Context, _ *supervisor.Supervisor, params appserver.CreateParams) (sessionActions, appserver.CreateResult, error) {
		created++
		gotParams = params
		return fresh, appserver.CreateResult{SessionID: "s3"}, nil
	}
	b.catalogSessions = func(context.Context, *supervisor.Supervisor) ([]dimui.Session, error) {
		return []dimui.Session{{ID: "s3", Title: "Third"}, {ID: "s1", Title: "One"}, {ID: "s2", Title: "Two"}}, nil
	}
	a, mu := createApp(b, &dimui.State{
		Sessions: []dimui.Session{{ID: "s1", Title: "One"}, {ID: "s2", Title: "Two"}},
		Selected: "s1",
		Messages: []dimui.Message{{ID: "m1", Content: "previous session row"}},
	})
	a.createParams = func() (appserver.CreateParams, error) {
		return appserver.CreateParams{Cwd: "/work"}, nil
	}

	a.state.OnCreateSession()
	waitForState(t, a, mu, "the created session selected", func(state *dimui.State) bool {
		return state.Selected == "s3" && !state.CreatingSession
	})

	if created != 1 {
		t.Fatalf("session.create ran %d times, want 1", created)
	}
	if gotParams.Cwd != "/work" {
		t.Fatalf("session.create params = %#v, want cwd /work", gotParams)
	}
	if _, summary := b.current(); summary.SessionID != "s3" {
		t.Fatalf("bridge summary = %#v, want s3", summary)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(a.state.Sessions) != 3 || a.state.Sessions[0].ID != "s3" || a.state.Sessions[0].Title != "Third" {
		t.Fatalf("catalog = %#v, want the App Server's rows with s3 first", a.state.Sessions)
	}
	// The window folded the new controller's view in: the previous session's
	// conversation must not stay under the new title.
	if len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m3" {
		t.Fatalf("messages = %#v, want the created session's rows", a.state.Messages)
	}
	if a.state.ActionError != "" {
		t.Fatalf("action error = %q after a created session", a.state.ActionError)
	}
	if len(a.state.ActionNotices) != 1 || a.state.ActionNotices[0].Kind != "create.accepted" || a.state.ActionNotices[0].ID != "s3" {
		t.Fatalf("create outcome was not recorded: %#v", a.state.ActionNotices)
	}
}

// TestAppCreateSessionClearsTheClosedTerminalState checks that a creation is a
// normal way out of the closed terminal state: the new session's view replaces
// the previous selection's rows and open health.
func TestAppCreateSessionClearsTheClosedTerminalState(t *testing.T) {
	fresh := &fakeSession{view: &dimui.State{
		Selected:   "s3",
		Connection: dimui.ConnReady,
		Messages:   []dimui.Message{{ID: "m3", Content: "new session row"}},
	}}
	b := &sessionBridge{}
	b.createSession = func(context.Context, *supervisor.Supervisor, appserver.CreateParams) (sessionActions, appserver.CreateResult, error) {
		return fresh, appserver.CreateResult{SessionID: "s3"}, nil
	}
	b.catalogSessions = func(context.Context, *supervisor.Supervisor) ([]dimui.Session, error) {
		return []dimui.Session{{ID: "s3", Title: "Third"}}, nil
	}
	a, mu := createApp(b, &dimui.State{
		Selected:            "s1",
		SessionClosed:       true,
		SessionClosedReason: closedReasonDeleted,
		Messages:            []dimui.Message{{ID: "m1", Content: "closed session row"}},
	})
	a.createParams = func() (appserver.CreateParams, error) {
		return appserver.CreateParams{Cwd: "/work"}, nil
	}

	a.state.OnCreateSession()
	waitForState(t, a, mu, "the closed state cleared by the created session", func(state *dimui.State) bool {
		return state.Selected == "s3" && !state.SessionClosed && !state.CreatingSession
	})

	mu.Lock()
	defer mu.Unlock()
	if a.state.SessionClosedReason != "" {
		t.Fatalf("closed reason = %q after creating a session", a.state.SessionClosedReason)
	}
	if a.state.ClosedSessionTitle != "" || a.state.ClosedSessionProject != "" {
		t.Fatalf("closed labels = %q / %q after creating a session",
			a.state.ClosedSessionTitle, a.state.ClosedSessionProject)
	}
	if len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m3" {
		t.Fatalf("messages = %#v, want the created session's rows", a.state.Messages)
	}
}

// TestAppCreateSessionFailureDoesNotSelect checks that a rejected session.create
// adds no row, moves no selection and keeps the error visible: the shell never
// shows a session the App Server did not return.
func TestAppCreateSessionFailureDoesNotSelect(t *testing.T) {
	b := &sessionBridge{}
	b.createSession = func(context.Context, *supervisor.Supervisor, appserver.CreateParams) (sessionActions, appserver.CreateResult, error) {
		return nil, appserver.CreateResult{}, errors.New("daemon refused")
	}
	b.catalogSessions = func(context.Context, *supervisor.Supervisor) ([]dimui.Session, error) {
		t.Error("a failed creation read the catalog")
		return nil, nil
	}
	a, mu := createApp(b, &dimui.State{
		Sessions: []dimui.Session{{ID: "s1", Title: "One"}, {ID: "s2", Title: "Two"}},
		Selected: "s1",
	})
	a.createParams = func() (appserver.CreateParams, error) {
		return appserver.CreateParams{Cwd: "/work"}, nil
	}

	a.state.OnCreateSession()
	waitForState(t, a, mu, "the creation failure reported", func(state *dimui.State) bool {
		return len(state.ActionNotices) == 1 && !state.CreatingSession
	})

	mu.Lock()
	defer mu.Unlock()
	if a.state.Selected != "s1" || len(a.state.Sessions) != 2 {
		t.Fatalf("failed creation changed the catalog: %#v", a.state)
	}
	if a.state.ActionError == "" {
		t.Fatal("failed creation was not surfaced")
	}
	if kind := a.state.ActionNotices[0].Kind; kind != "create.error" {
		t.Fatalf("failed creation recorded as %q", kind)
	}
}

// TestAppCreateSessionCreatesOnce checks that one unanswered session.create owns
// the entry: a second click while it is in flight must not ask for another
// session.
func TestAppCreateSessionCreatesOnce(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var created int
	b := &sessionBridge{}
	b.createSession = func(context.Context, *supervisor.Supervisor, appserver.CreateParams) (sessionActions, appserver.CreateResult, error) {
		created++
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		return &fakeSession{view: &dimui.State{Selected: "s3", Connection: dimui.ConnReady}}, appserver.CreateResult{SessionID: "s3"}, nil
	}
	b.catalogSessions = func(context.Context, *supervisor.Supervisor) ([]dimui.Session, error) {
		return []dimui.Session{{ID: "s3"}, {ID: "s1"}}, nil
	}
	a, mu := createApp(b, &dimui.State{Sessions: []dimui.Session{{ID: "s1"}}, Selected: "s1"})
	a.createParams = func() (appserver.CreateParams, error) { return appserver.CreateParams{Cwd: "/work"}, nil }

	a.state.OnCreateSession()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("session.create never started")
	}
	// The first click is still unanswered; a second one must not add a request.
	a.state.OnCreateSession()
	close(release)

	waitForState(t, a, mu, "the created session selected", func(state *dimui.State) bool {
		return state.Selected == "s3"
	})
	if created != 1 {
		t.Fatalf("session.create ran %d times for two clicks on one request, want 1", created)
	}
}

// TestAppCreateSessionDoesNotStealNewerSelection checks the stale-selection
// guard on the creation path: a creation that finishes while the user has
// already chosen another session must not reselect what it created.
func TestAppCreateSessionDoesNotStealNewerSelection(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	b := &sessionBridge{}
	b.createSession = func(context.Context, *supervisor.Supervisor, appserver.CreateParams) (sessionActions, appserver.CreateResult, error) {
		close(started)
		<-release
		return &fakeSession{view: &dimui.State{Selected: "s3", Connection: dimui.ConnReady}}, appserver.CreateResult{SessionID: "s3"}, nil
	}
	b.catalogSessions = func(context.Context, *supervisor.Supervisor) ([]dimui.Session, error) {
		return []dimui.Session{{ID: "s3"}, {ID: "s1"}, {ID: "s2"}}, nil
	}
	b.openSession = func(_ context.Context, _ *supervisor.Supervisor, id string) (sessionActions, error) {
		return &fakeSession{view: &dimui.State{Selected: id, Connection: dimui.ConnReady}}, nil
	}
	a, mu := createApp(b, &dimui.State{
		Sessions: []dimui.Session{{ID: "s1"}, {ID: "s2"}},
		Selected: "s1",
	})
	a.createParams = func() (appserver.CreateParams, error) { return appserver.CreateParams{Cwd: "/work"}, nil }

	a.state.OnCreateSession()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("session.create never started")
	}
	// The user chooses another session while the creation is in flight.
	a.state.Selected = "s2"
	a.state.OnSelect("s2")
	close(release)

	waitForState(t, a, mu, "the newer selection", func(state *dimui.State) bool {
		_, summary := b.current()
		return state.Selected == "s2" && summary.SessionID == "s2"
	})
	mu.Lock()
	defer mu.Unlock()
	if a.state.Selected != "s2" {
		t.Fatalf("the finished creation stole the selection: %#v", a.state)
	}
	if len(a.state.Sessions) != 2 {
		t.Fatalf("a stale creation published its catalog: %#v", a.state.Sessions)
	}
}

// TestAppCreateSessionWithoutSupervisorReportsIt checks that the entry answers
// when there is no App Server to ask: the window records why instead of leaving
// the click without an effect or pretending a session was made.
func TestAppCreateSessionWithoutSupervisorReportsIt(t *testing.T) {
	a, mu := createApp(nil, &dimui.State{Connection: dimui.ConnUnknown})

	a.state.OnCreateSession()
	waitForState(t, a, mu, "the missing supervisor reported", func(state *dimui.State) bool {
		return len(state.ActionNotices) == 1
	})

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(a.state.ActionError, "DIMAGENT_PRODUCT_HOME") {
		t.Fatalf("action error = %q, want it to name the missing configuration", a.state.ActionError)
	}
	if a.state.ActionNotices[0].Kind != "create.error" || a.state.Selected != "" {
		t.Fatalf("outcome = %#v, want a create error and no selection", a.state)
	}
}

// TestBridgeCreateInstallsController checks that a created session becomes the
// bridge's current controller and that the previous one is released.
func TestBridgeCreateInstallsController(t *testing.T) {
	previous := &fakeSession{}
	fresh := &fakeSession{view: &dimui.State{Selected: "s3"}}
	b := &sessionBridge{controller: previous, summary: appserver.SessionSummary{SessionID: "s1"}}
	b.createSession = func(context.Context, *supervisor.Supervisor, appserver.CreateParams) (sessionActions, appserver.CreateResult, error) {
		return fresh, appserver.CreateResult{SessionID: "s3"}, nil
	}

	row, err := b.create(context.Background(), appserver.CreateParams{Cwd: "/work"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if row.ID != "s3" || row.Title != "" {
		t.Fatalf("row = %#v, want the App Server's id and no title fact", row)
	}
	if b.controller != fresh || b.summary.SessionID != "s3" {
		t.Fatalf("bridge did not install the created session: %#v %#v", b.controller, b.summary)
	}
	if !previous.closed {
		t.Error("the previous controller stayed open after a creation replaced it")
	}
}

// TestBridgeCreateFailureKeepsPrevious checks that a rejected creation installs
// nothing and leaves the current controller attached.
func TestBridgeCreateFailureKeepsPrevious(t *testing.T) {
	previous := &fakeSession{}
	b := &sessionBridge{controller: previous, summary: appserver.SessionSummary{SessionID: "s1"}}
	b.createSession = func(context.Context, *supervisor.Supervisor, appserver.CreateParams) (sessionActions, appserver.CreateResult, error) {
		return nil, appserver.CreateResult{}, errors.New("refused")
	}

	if _, err := b.create(context.Background(), appserver.CreateParams{Cwd: "/work"}); err == nil {
		t.Fatal("a refused creation reported success")
	}
	if b.controller != previous || previous.closed {
		t.Fatalf("a refused creation replaced the controller: %#v", b.controller)
	}
}

// TestBridgeCreateWithoutSession checks that a create result without a session
// id is refused instead of installed as an anonymous session.
func TestBridgeCreateWithoutSession(t *testing.T) {
	b := &sessionBridge{}
	fresh := &fakeSession{}
	b.createSession = func(context.Context, *supervisor.Supervisor, appserver.CreateParams) (sessionActions, appserver.CreateResult, error) {
		return fresh, appserver.CreateResult{}, nil
	}
	if _, err := b.create(context.Background(), appserver.CreateParams{Cwd: "/work"}); err == nil {
		t.Fatal("a create result without a session id was accepted")
	}
	if b.controller != nil {
		t.Fatalf("a session without an id was installed: %#v", b.controller)
	}
	if !fresh.closed {
		t.Error("the unusable controller stayed open")
	}
}

// TestDefaultCreateParams checks that the window sends the directory it runs in
// and no model or permission choice of its own: those belong to the App Server.
func TestDefaultCreateParams(t *testing.T) {
	params, err := defaultCreateParams()
	if err != nil {
		t.Fatalf("defaultCreateParams: %v", err)
	}
	want, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if params.Cwd != want {
		t.Fatalf("cwd = %q, want %q", params.Cwd, want)
	}
	if params.Model != nil || params.ModelPreferences != nil || params.PermissionPolicy != nil {
		t.Fatalf("the window chose daemon policy: %#v", params)
	}
}

// TestWithCreatedRow checks the catalog merge: a session the catalog read has
// not caught up with is added once, and an existing row is left alone.
func TestWithCreatedRow(t *testing.T) {
	rows := []dimui.Session{{ID: "s1", Title: "One"}}
	merged := withCreatedRow(rows, dimui.Session{ID: "s2", Title: "s2"})
	if len(merged) != 2 || merged[0].ID != "s2" || merged[1].ID != "s1" {
		t.Fatalf("merged = %#v, want the created session first", merged)
	}
	again := withCreatedRow(merged, dimui.Session{ID: "s2"})
	if len(again) != 2 || again[0].Title != "s2" || len(rows) != 1 {
		t.Fatalf("an existing row was duplicated: %#v", again)
	}
}

// submitApp builds a window with one selected session on branch main and a
// draft, plus the mutex guarding the state its background callbacks update. The
// read-back retry interval is shrunk so a retried lookup resolves in
// milliseconds instead of seconds.
func submitApp(f *fakeSession) (*app, *sync.Mutex) {
	a, mu, _ := submitAppWithUpdates(f)
	return a, mu
}

// submitAppWithUpdates is submitApp plus a signal per published update, so a
// test can prove a stale read-back published nothing.
func submitAppWithUpdates(f *fakeSession) (*app, *sync.Mutex, <-chan struct{}) {
	bridge := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	a := &app{state: &dimui.State{Selected: "s1", SelectedBranch: "main", Draft: "hello"}, bridge: bridge}
	a.lookupInterval = time.Millisecond
	a.lookupBudget = 2 * time.Second
	a.bind()
	var mu sync.Mutex
	updates := make(chan struct{}, 16)
	a.updateHook = func(fn func(*dimui.State)) {
		mu.Lock()
		fn(a.state)
		mu.Unlock()
		select {
		case updates <- struct{}{}:
		default:
		}
	}
	return a, &mu, updates
}

// TestAppSubmitIsSingleFlight checks that repeated submit triggers while the
// first request is unanswered do not create a second App Server mutation.
func TestAppSubmitIsSingleFlight(t *testing.T) {
	release := make(chan struct{})
	started := make(chan string, 1)
	f := &fakeSession{
		submitGate: release,
		submitted:  started,
		view:       &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
	}
	a, mu := submitApp(f)
	a.state.OnSubmit("hello")
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first submit did not reach the App Server")
	}
	a.state.OnSubmit("hello again")
	close(release)
	waitForNotice(t, a, mu, "submit.accepted")
	if len(f.submitText) != 1 {
		t.Fatalf("submit count = %d, want one mutation while in flight", len(f.submitText))
	}
}

// TestAppSubmitUnknownResultReadsBackAccepted checks that an ambiguous submit is
// resolved through session.input_lookup: the App Server's accepted answer is
// what clears the draft, and the submission is never sent twice.
func TestAppSubmitUnknownResultReadsBackAccepted(t *testing.T) {
	f := &fakeSession{
		submitErr:    &appserver.UnknownResultError{Method: appserver.MutationSessionSubmit},
		submitID:     "sub-1",
		lookupResult: appserver.InputLookupResult{Status: appserver.InputLookupAccepted},
		view:         &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
	}
	a, mu := submitApp(f)

	a.state.OnSubmit("hello")
	waitForNotice(t, a, mu, "submit.accepted")

	if len(f.submitText) != 1 {
		t.Fatalf("submit count = %d after an unknown result, want 1", len(f.submitText))
	}
	if f.lookupCalls != 1 || f.lookupBranch != "main" || f.lookupSubmission != "sub-1" {
		t.Fatalf("read-back = %d (%q, %q), want one lookup of sub-1 on main",
			f.lookupCalls, f.lookupBranch, f.lookupSubmission)
	}
	mu.Lock()
	defer mu.Unlock()
	if a.state.Draft != "" {
		t.Errorf("draft = %q after the App Server confirmed the input, want it cleared", a.state.Draft)
	}
	if a.state.ActionStatus != "Submission accepted" || a.state.ActionError != "" {
		t.Errorf("outcome = %q / %q, want the accepted read-back", a.state.ActionStatus, a.state.ActionError)
	}
}

// TestAppSubmitUnknownResultStaysUnknown checks that read-backs the App Server
// keeps answering as unknown are not reported as success: the draft stays, the
// alert stays, the lookup retries stay inside the read-back budget and nothing
// is resent.
func TestAppSubmitUnknownResultStaysUnknown(t *testing.T) {
	cases := []struct {
		name string
		edit func(*fakeSession)
		// lookups is the expected read-back count: the first lookup plus the
		// retries, or none when there is no submission id to name.
		lookups int
		want    string
	}{
		{"daemon has no record", func(f *fakeSession) {
			f.lookupResult = appserver.InputLookupResult{Status: appserver.InputLookupUnknown}
		}, 1 + submitLookupRetries, "does not hold this submission"},
		{"read-back fails", func(f *fakeSession) {
			f.lookupErr = errors.New("daemon went away")
		}, 1 + submitLookupRetries, "read-back failed"},
		{"no submission id", func(f *fakeSession) {
			f.submitID = ""
		}, 0, "no submission id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeSession{
				submitErr: &appserver.UnknownResultError{Method: appserver.MutationSessionSubmit},
				submitID:  "sub-1",
				view:      &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
			}
			tc.edit(f)
			a, mu := submitApp(f)

			a.state.OnSubmit("hello")
			waitForNotice(t, a, mu, "submit.unknown")

			if len(f.submitText) != 1 {
				t.Fatalf("submit count = %d, want the unknown submission not resent", len(f.submitText))
			}
			mu.Lock()
			defer mu.Unlock()
			if f.lookupCalls != tc.lookups {
				t.Errorf("read-back ran %d times, want %d", f.lookupCalls, tc.lookups)
			}
			if a.state.Draft != "hello" {
				t.Errorf("draft = %q after an unknown result, want it kept", a.state.Draft)
			}
			if a.state.ActionStatus != "Submission result unknown" {
				t.Errorf("status = %q, want it to stay unknown", a.state.ActionStatus)
			}
			if !strings.Contains(a.state.ActionError, tc.want) {
				t.Errorf("action error = %q, want it to mention %q", a.state.ActionError, tc.want)
			}
			for _, notice := range a.state.ActionNotices {
				if notice.Kind == "submit.accepted" {
					t.Fatalf("an unknown result was recorded as accepted: %#v", a.state.ActionNotices)
				}
			}
		})
	}
}

// TestAppSubmitUnknownResultRetriesLateAcceptance checks the late App Server
// answer: an unknown read-back is retried, and the retry's accepted answer
// resolves the submission through the same success path. The submission is
// still sent once and the earlier unknown outcome is cleared.
func TestAppSubmitUnknownResultRetriesLateAcceptance(t *testing.T) {
	f := &fakeSession{
		submitErr: &appserver.UnknownResultError{Method: appserver.MutationSessionSubmit},
		submitID:  "sub-1",
		lookupResults: []appserver.InputLookupResult{
			{Status: appserver.InputLookupUnknown},
			{Status: appserver.InputLookupAccepted},
		},
		view: &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
	}
	a, mu := submitApp(f)
	// An earlier unknown outcome must not survive the accepted retry.
	a.state.ActionError = "an earlier unknown result"
	a.state.ActionStatus = "Submission result unknown"
	a.state.ActionNotices = []dimui.Notice{{Kind: "submit.unknown", ID: "an earlier unknown result"}}

	a.state.OnSubmit("hello")
	waitForNotice(t, a, mu, "submit.accepted")

	if len(f.submitText) != 1 {
		t.Fatalf("submit count = %d after the retried read-back, want 1", len(f.submitText))
	}
	if f.lookupCalls != 2 {
		t.Fatalf("read-back ran %d times, want the unknown answer retried once", f.lookupCalls)
	}
	if f.lookupBranch != "main" || f.lookupSubmission != "sub-1" {
		t.Fatalf("read-back = (%q, %q), want the original submission on main", f.lookupBranch, f.lookupSubmission)
	}
	mu.Lock()
	defer mu.Unlock()
	if a.state.Draft != "" {
		t.Errorf("draft = %q after the accepted retry, want it cleared", a.state.Draft)
	}
	if a.state.ActionStatus != "Submission accepted" || a.state.ActionError != "" {
		t.Errorf("outcome = %q / %q, want the accepted read-back", a.state.ActionStatus, a.state.ActionError)
	}
	for _, notice := range a.state.ActionNotices {
		if notice.Kind == "submit.unknown" {
			t.Fatalf("a superseded unknown outcome stayed visible: %#v", a.state.ActionNotices)
		}
	}
}

// TestAppSubmitUnknownResultRetriesLookupError checks that a transient
// read-back error is retried read-only: the next lookup's accepted answer
// resolves the submission without resending it.
func TestAppSubmitUnknownResultRetriesLookupError(t *testing.T) {
	f := &fakeSession{
		submitErr: &appserver.UnknownResultError{Method: appserver.MutationSessionSubmit},
		submitID:  "sub-1",
		lookupResults: []appserver.InputLookupResult{
			{Status: appserver.InputLookupUnknown},
			{Status: appserver.InputLookupAccepted},
		},
		lookupErrs: []error{errors.New("daemon went away")},
		view:       &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
	}
	a, mu := submitApp(f)

	a.state.OnSubmit("hello")
	waitForNotice(t, a, mu, "submit.accepted")

	if len(f.submitText) != 1 {
		t.Fatalf("submit count = %d after a failed read-back, want the submission not resent", len(f.submitText))
	}
	if f.lookupCalls != 2 {
		t.Fatalf("read-back ran %d times, want one retry after the error", f.lookupCalls)
	}
	mu.Lock()
	defer mu.Unlock()
	if a.state.Draft != "" || a.state.ActionStatus != "Submission accepted" || a.state.ActionError != "" {
		t.Errorf("outcome = draft %q / %q / %q, want the accepted read-back",
			a.state.Draft, a.state.ActionStatus, a.state.ActionError)
	}
}

// TestAppSubmitUnknownResultStopsOnSessionSwitch checks that a read-back
// waiting to retry stops when the window moves to another session: its late
// accepted answer must not clear the new session's draft or publish an outcome
// there, and the submission is never resent.
func TestAppSubmitUnknownResultStopsOnSessionSwitch(t *testing.T) {
	started := make(chan struct{}, 1)
	f := &fakeSession{
		submitErr: &appserver.UnknownResultError{Method: appserver.MutationSessionSubmit},
		submitID:  "sub-1",
		lookupResults: []appserver.InputLookupResult{
			{Status: appserver.InputLookupUnknown},
			{Status: appserver.InputLookupAccepted},
		},
		lookupStarted: started,
		view:          &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
	}
	a, mu, updates := submitAppWithUpdates(f)
	// Long enough that only the selection change can end the wait.
	a.lookupInterval = 10 * time.Second
	a.lookupBudget = 20 * time.Second

	a.state.OnSubmit("hello")
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("read-back did not start")
	}
	a.beginSelect() // the user moves to another session while the retry waits

	select {
	case <-updates:
		t.Fatal("a stale read-back published an outcome into the new selection")
	case <-time.After(50 * time.Millisecond):
	}
	mu.Lock()
	defer mu.Unlock()
	if f.lookupCalls != 1 {
		t.Errorf("read-back ran %d times after the session switch, want the retry stopped", f.lookupCalls)
	}
	if len(f.submitText) != 1 {
		t.Errorf("submit count = %d, want the submission not resent", len(f.submitText))
	}
	if a.state.Draft != "hello" {
		t.Errorf("draft = %q after the session switch, want it untouched", a.state.Draft)
	}
	if a.state.ActionStatus != "" || a.state.ActionError != "" || len(a.state.ActionNotices) != 0 {
		t.Errorf("a stale read-back changed the new selection: %q / %q / %#v",
			a.state.ActionStatus, a.state.ActionError, a.state.ActionNotices)
	}
}
