package main

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/config"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// fakeDesktop builds a store over an in-memory file and records every write, so
// a test sees exactly which desktop-local state was persisted.
func fakeDesktop(initial config.State) (*desktopStore, *[]config.State) {
	writes := []config.State{}
	store := &desktopStore{
		load: func() (config.State, error) { return initial.Clone(), nil },
		save: func(state config.State) error {
			writes = append(writes, state.Clone())
			return nil
		},
	}
	return store, &writes
}

// TestDesktopStoreKeepsSectionsTogether checks that saving a draft or a window
// leaves the other desktop-local sections in the file: one writer replaces the
// whole state, so it must start from the current one.
func TestDesktopStoreKeepsSectionsTogether(t *testing.T) {
	initial := config.DefaultState()
	initial.Settings.Theme = 2
	initial.Window = config.Window{X: 120, Y: 90, Width: 900, Height: 640}
	initial.Drafts = config.Drafts{"s1": "one"}
	store, writes := fakeDesktop(initial)
	if _, err := store.loadOnce(); err != nil {
		t.Fatalf("loadOnce: %v", err)
	}

	if err := store.setDraft("s2", "two"); err != nil {
		t.Fatalf("setDraft: %v", err)
	}
	if len(*writes) != 1 {
		t.Fatalf("setDraft wrote %d time(s), want 1", len(*writes))
	}
	saved := (*writes)[0]
	if saved.Settings.Theme != 2 || saved.Window != initial.Window || saved.DraftFor("s1") != "one" || saved.DraftFor("s2") != "two" {
		t.Fatalf("setDraft dropped another section: %+v", saved)
	}

	if err := store.setWindow(config.Window{X: 10, Y: 20, Width: 700, Height: 500}); err != nil {
		t.Fatalf("setWindow: %v", err)
	}
	saved = (*writes)[1]
	if saved.Settings.Theme != 2 || saved.DraftFor("s2") != "two" || saved.Window.Width != 700 {
		t.Fatalf("setWindow dropped another section: %+v", saved)
	}
}

// TestDesktopStoreDoesNotRewriteAnUnchangedDraft checks that a repeated save of
// the same text writes nothing, so selecting the same session or closing an
// empty composer costs no file write.
func TestDesktopStoreDoesNotRewriteAnUnchangedDraft(t *testing.T) {
	initial := config.DefaultState()
	initial.Drafts = config.Drafts{"s1": "hello"}
	store, writes := fakeDesktop(initial)

	if err := store.setDraft("s1", "hello"); err != nil {
		t.Fatalf("setDraft: %v", err)
	}
	if len(*writes) != 0 {
		t.Fatalf("an unchanged draft wrote %d snapshot(s)", len(*writes))
	}

	if err := store.clearDraft("s1"); err != nil {
		t.Fatalf("clearDraft: %v", err)
	}
	if len(*writes) != 1 || len(store.current().Drafts) != 0 {
		t.Fatalf("clearDraft writes = %d, drafts = %+v", len(*writes), store.current().Drafts)
	}
}

// TestDesktopStoreKeepsTheOldStateOnAFailedSave checks the atomic part: a write
// failure leaves the state the window reads as it was, so a later choice can try
// again without a half-applied draft.
func TestDesktopStoreKeepsTheOldStateOnAFailedSave(t *testing.T) {
	initial := config.DefaultState()
	initial.Drafts = config.Drafts{"s1": "old"}
	boom := errors.New("disk full")
	store := &desktopStore{
		load: func() (config.State, error) { return initial.Clone(), nil },
		save: func(config.State) error { return boom },
	}
	if err := store.setDraft("s1", "new"); !errors.Is(err, boom) {
		t.Fatalf("setDraft error = %v, want %v", err, boom)
	}
	if got := store.current().DraftFor("s1"); got != "old" {
		t.Fatalf("a failed save published %q, want the old draft", got)
	}
}

// TestLoadDesktopRestoresTheSelectedSessionsDraft checks startup: the draft saved
// for the session the window already shows is put back into the composer, and a
// session without a draft stays empty.
func TestLoadDesktopRestoresTheSelectedSessionsDraft(t *testing.T) {
	initial := config.DefaultState()
	initial.Drafts = config.Drafts{"s1": "picked up", "s2": "other"}
	store, _ := fakeDesktop(initial)
	a := &app{state: &dimui.State{Selected: "s1"}, desktop: store}
	a.loadDesktop()
	if a.state.Draft != "picked up" {
		t.Fatalf("draft = %q, want the saved draft of s1", a.state.Draft)
	}

	other := &app{state: &dimui.State{Selected: "s3"}, desktop: store}
	other.loadDesktop()
	if other.state.Draft != "" {
		t.Fatalf("draft = %q for a session without one, want empty", other.state.Draft)
	}
}

// TestSelectSessionSavesTheDraftItLeavesAndRestoresTheOneItOpens checks a
// session switch: the outgoing composer text is saved under the outgoing
// session, and the incoming session's saved draft appears in the composer.
func TestSelectSessionSavesTheDraftItLeavesAndRestoresTheOneItOpens(t *testing.T) {
	first := &fakeSession{view: &dimui.State{Selected: "s1", Connection: dimui.ConnReady}}
	second := &fakeSession{view: &dimui.State{Selected: "s2", Connection: dimui.ConnReady}}
	bridge := &sessionBridge{controller: first, summary: appserver.SessionSummary{SessionID: "s1"}}
	bridge.openSession = func(_ context.Context, _ *supervisor.Supervisor, id string) (sessionActions, error) {
		return second, nil
	}
	initial := config.DefaultState()
	initial.Drafts = config.Drafts{"s1": "stale saved text", "s2": "draft for two"}
	store, _ := fakeDesktop(initial)
	state := &dimui.State{
		Sessions:   []dimui.Session{{ID: "s1"}, {ID: "s2"}},
		Selected:   "s1",
		Draft:      "typed but not sent",
		Connection: dimui.ConnReady,
	}
	a := &app{state: state, bridge: bridge, desktop: store}
	var mu sync.Mutex
	a.updateHook = func(fn func(*dimui.State)) {
		mu.Lock()
		defer mu.Unlock()
		fn(a.state)
	}
	a.bind()

	a.selectSession("s2")
	waitForState(t, a, &mu, "s2's saved draft", func(state *dimui.State) bool {
		return state.Selected == "s2" && state.Draft == "draft for two"
	})
	mu.Lock()
	defer mu.Unlock()
	if got := store.current().DraftFor("s1"); got != "typed but not sent" {
		t.Fatalf("saved draft for s1 = %q, want the outgoing composer text", got)
	}
}

// TestSubmitClearsTheSavedDraft checks that an accepted submission drops the
// session's saved draft.
func TestSubmitClearsTheSavedDraft(t *testing.T) {
	f := &fakeSession{view: &dimui.State{Selected: "s1", Connection: dimui.ConnReady}}
	a, mu := submitApp(f)
	initial := config.DefaultState()
	initial.Drafts = config.Drafts{"s1": "hello"}
	store, _ := fakeDesktop(initial)
	a.desktop = store

	a.state.OnSubmit("hello")
	waitForState(t, a, mu, "the accepted draft cleared", func(state *dimui.State) bool {
		return state.Draft == ""
	})
	if got := store.current().DraftFor("s1"); got != "" {
		t.Fatalf("saved draft = %q after acceptance, want it cleared", got)
	}
}

// TestSubmitKeepsADraftTypedAfterTheSend checks the boundary: acceptance clears
// the text that was sent, not a newer saved draft.
func TestSubmitKeepsADraftTypedAfterTheSend(t *testing.T) {
	f := &fakeSession{view: &dimui.State{Selected: "s1", Connection: dimui.ConnReady}}
	a, mu := submitApp(f)
	initial := config.DefaultState()
	initial.Drafts = config.Drafts{"s1": "newer text"}
	store, _ := fakeDesktop(initial)
	a.desktop = store

	a.state.OnSubmit("hello")
	waitForNotice(t, a, mu, "submit.accepted")
	if got := store.current().DraftFor("s1"); got != "newer text" {
		t.Fatalf("saved draft = %q, want the newer text kept", got)
	}
}

// TestLoadDesktopRestoresHiddenLists checks startup: the pinned and archived ids
// load into the window's display state, and they stay copies the store does not
// share.
func TestLoadDesktopRestoresHiddenLists(t *testing.T) {
	initial := config.DefaultState()
	initial.Pinned = []string{"s2"}
	initial.Archived = []string{"s3"}
	store, _ := fakeDesktop(initial)
	a := &app{state: &dimui.State{Selected: "s1"}, desktop: store}
	a.loadDesktop()

	if !slices.Equal(a.state.Pinned, []string{"s2"}) || !slices.Equal(a.state.Archived, []string{"s3"}) {
		t.Fatalf("loaded hidden lists = %v / %v, want s2 pinned and s3 archived", a.state.Pinned, a.state.Archived)
	}
	a.state.Pinned[0] = "changed"
	if store.current().Pinned[0] != "s2" {
		t.Fatal("the window's hidden list shares storage with the store")
	}
}

// TestAppTogglesPinnedAndArchived checks the menu callbacks: each writes the
// desktop-local list once, publishes it to the window, and a second click
// toggles it back. The App Server is not involved.
func TestAppTogglesPinnedAndArchived(t *testing.T) {
	store, writes := fakeDesktop(config.DefaultState())
	a := &app{state: &dimui.State{Selected: "s1"}, desktop: store}
	a.updateHook = func(fn func(*dimui.State)) { fn(a.state) }
	a.loadDesktop()

	a.togglePin("s1")
	if !slices.Equal(a.state.Pinned, []string{"s1"}) || len(*writes) != 1 {
		t.Fatalf("after pin: state=%v writes=%d, want s1 pinned once", a.state.Pinned, len(*writes))
	}
	a.togglePin("s1")
	if len(a.state.Pinned) != 0 || len(*writes) != 2 {
		t.Fatalf("after unpin: state=%v writes=%d, want empty and two writes", a.state.Pinned, len(*writes))
	}

	a.toggleArchive("s2")
	if !slices.Equal(a.state.Archived, []string{"s2"}) || len(*writes) != 3 {
		t.Fatalf("after archive: state=%v writes=%d, want s2 archived once", a.state.Archived, len(*writes))
	}
	a.restoreArchived("s2")
	if len(a.state.Archived) != 0 || len(*writes) != 4 {
		t.Fatalf("after restore: state=%v writes=%d, want empty and four writes", a.state.Archived, len(*writes))
	}
}

// TestAppDeleteForgetsHiddenDisplayState checks the delete path: after the App
// Server confirms, the deleted id leaves both desktop-local lists, so neither
// store keeps naming a row that no longer exists.
func TestAppDeleteForgetsHiddenDisplayState(t *testing.T) {
	f := &fakeSession{summary: appserver.SessionSummary{SessionID: "s1"}}
	initial := config.DefaultState()
	initial.Pinned = []string{"s1"}
	initial.Archived = []string{"s2"}
	store, _ := fakeDesktop(initial)
	state := &dimui.State{
		Sessions: []dimui.Session{{ID: "s1", Title: "one"}, {ID: "s2", Title: "two"}},
		Selected: "s1",
	}
	a := &app{state: state, bridge: &sessionBridge{controller: f, summary: f.summary}, desktop: store}
	var mu sync.Mutex
	a.updateHook = func(fn func(*dimui.State)) {
		mu.Lock()
		defer mu.Unlock()
		fn(a.state)
	}
	a.loadDesktop()
	a.bind()
	a.state.OnDeleteSession("s1")

	waitForState(t, a, &mu, "the deleted id removed from the hidden lists", func(state *dimui.State) bool {
		return len(state.Sessions) == 1 && !slices.Contains(state.Pinned, "s1") && slices.Contains(state.Archived, "s2")
	})
	mu.Lock()
	if store.current().IsPinned("s1") {
		mu.Unlock()
		t.Fatal("the store still pins the deleted session")
	}
	if !store.current().IsArchived("s2") {
		mu.Unlock()
		t.Fatal("the delete dropped another session's archived mark")
	}
	mu.Unlock()
}
