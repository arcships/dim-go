package main

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/session"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// stringPtr is a pointer to one string, for the optional title fields.
func stringPtr(value string) *string { return &value }

// renameApp is a window attached to one session, with the catalog read the test
// supplies and the mutex guarding the state its background callback updates.
func renameApp(f *fakeSession, rows []dimui.Session) (*app, *sessionBridge, *sync.Mutex) {
	b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	b.catalogSessions = func(context.Context, *supervisor.Supervisor) ([]dimui.Session, error) {
		return rows, nil
	}
	a := &app{
		state: &dimui.State{
			Sessions:   []dimui.Session{{ID: "s1", Title: "Old title"}},
			Selected:   "s1",
			Connection: dimui.ConnReady,
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

// TestAppRenameSendsOnceAndRefreshesTheCatalog checks the wired rename: the
// trimmed title is sent through session.rename once, the catalog read back is
// the sidebar's list, and the row shows the committed title. The shell never
// invents a title from the request.
func TestAppRenameSendsOnceAndRefreshesTheCatalog(t *testing.T) {
	f := &fakeSession{
		view:         &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		renameResult: appserver.RenameResult{Title: stringPtr("Renamed"), TitleSource: stringPtr("manual"), Revision: 2},
		titleState:   session.TitleState{Text: stringPtr("Renamed"), Known: true, Revision: 2},
	}
	a, _, mu := renameApp(f, []dimui.Session{{ID: "s1", Title: "Renamed"}})

	a.renameSession("  Renamed  ")
	waitForState(t, a, mu, "the renamed catalog row", func(state *dimui.State) bool {
		return len(state.Sessions) == 1 && state.Sessions[0].Title == "Renamed"
	})
	mu.Lock()
	defer mu.Unlock()
	if len(f.renamed) != 1 {
		t.Fatalf("session.rename sent %d times, want once", len(f.renamed))
	}
	if f.renamed[0] == nil || *f.renamed[0] != "Renamed" {
		t.Fatalf("session.rename title = %v, want the trimmed title", f.renamed[0])
	}
	if a.state.ActionError != "" {
		t.Fatalf("action error = %q after a successful rename", a.state.ActionError)
	}
	if len(a.state.ActionNotices) != 1 || a.state.ActionNotices[0].Kind != "rename.accepted" ||
		a.state.ActionNotices[0].ID != "Renamed" {
		t.Fatalf("rename outcome = %#v, want one accepted notice with the title", a.state.ActionNotices)
	}
}

// TestAppRenameWithAnEmptyTitleClears checks the clear path: a whitespace-only
// title sends a nil title, which session.rename reads as "clear", and the
// outcome notice says the title was cleared.
func TestAppRenameWithAnEmptyTitleClears(t *testing.T) {
	f := &fakeSession{
		view:         &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		renameResult: appserver.RenameResult{Revision: 3},
		titleState:   session.TitleState{Known: true, Revision: 3},
	}
	a, _, mu := renameApp(f, []dimui.Session{{ID: "s1", Title: ""}})

	a.renameSession("   ")
	waitForState(t, a, mu, "the cleared catalog row", func(state *dimui.State) bool {
		return len(state.Sessions) == 1 && state.Sessions[0].Title == ""
	})
	mu.Lock()
	defer mu.Unlock()
	if len(f.renamed) != 1 || f.renamed[0] != nil {
		t.Fatalf("session.rename sent %v, want one nil (clear) request", f.renamed)
	}
	if len(a.state.ActionNotices) != 1 || a.state.ActionNotices[0].ID != "cleared" {
		t.Fatalf("clear outcome = %#v, want the cleared notice", a.state.ActionNotices)
	}
}

// TestAppRenameFailureReportsThroughTheBanner checks a rejected rename: the row
// keeps its title, the failure is the window-owned action error, and nothing
// looks accepted.
func TestAppRenameFailureReportsThroughTheBanner(t *testing.T) {
	f := &fakeSession{
		view:      &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		renameErr: errors.New("invalid params"),
	}
	a, _, mu := renameApp(f, []dimui.Session{{ID: "s1", Title: "Old title"}})

	a.renameSession("New title")
	waitForState(t, a, mu, "the rename failure", func(state *dimui.State) bool {
		return state.ActionError != ""
	})
	mu.Lock()
	defer mu.Unlock()
	if a.state.Sessions[0].Title != "Old title" {
		t.Fatalf("the failed rename changed the title to %q", a.state.Sessions[0].Title)
	}
	if len(a.state.ActionNotices) != 1 || a.state.ActionNotices[0].Kind != "rename.error" {
		t.Fatalf("rename failure was not recorded: %#v", a.state.ActionNotices)
	}
}

// TestAppRenameUnknownMethodShowsTheWIPReason checks unknown_method: the
// failure names the method and the WIP state instead of reading as a bad title,
// and the row keeps its title.
func TestAppRenameUnknownMethodShowsTheWIPReason(t *testing.T) {
	f := &fakeSession{
		view:      &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		renameErr: &appserver.CapabilityError{Feature: "session.rename"},
	}
	a, _, mu := renameApp(f, []dimui.Session{{ID: "s1", Title: "Old title"}})

	a.renameSession("New title")
	waitForState(t, a, mu, "the unknown_method failure", func(state *dimui.State) bool {
		return state.ActionError != ""
	})
	mu.Lock()
	defer mu.Unlock()
	for _, want := range []string{"unknown_method", "unavailable", "session.rename", "WIP"} {
		if !strings.Contains(a.state.ActionError, want) {
			t.Errorf("action error %q does not name %q", a.state.ActionError, want)
		}
	}
	if a.state.Sessions[0].Title != "Old title" {
		t.Fatalf("the refused rename changed the title to %q", a.state.Sessions[0].Title)
	}
}

// TestAppRenameCatalogFailureKeepsTheCommittedTitle checks the partial failure:
// the App Server confirmed the rename but the catalog read failed, so the window
// says so instead of reporting a clean success or dropping the rows.
func TestAppRenameCatalogFailureKeepsTheCommittedTitle(t *testing.T) {
	f := &fakeSession{
		view:         &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		renameResult: appserver.RenameResult{Title: stringPtr("Renamed"), TitleSource: stringPtr("manual"), Revision: 2},
	}
	a, b, mu := renameApp(f, nil)
	b.catalogSessions = func(context.Context, *supervisor.Supervisor) ([]dimui.Session, error) {
		return nil, errors.New("catalog unavailable")
	}

	a.renameSession("Renamed")
	waitForState(t, a, mu, "the catalog failure", func(state *dimui.State) bool {
		return strings.Contains(state.ActionError, "catalog refresh failed")
	})
	mu.Lock()
	defer mu.Unlock()
	if len(a.state.Sessions) != 1 || a.state.Sessions[0].Title != "Old title" {
		t.Fatalf("a failed refresh dropped or rewrote the rows: %#v", a.state.Sessions)
	}
}

// TestReadSessionModelsMapsTheCatalog checks the model snapshot the composer
// displays: the current pair, the catalog's order, each model's availability and
// the current marker, with no selection invented.
func TestReadSessionModelsMapsTheCatalog(t *testing.T) {
	f := &fakeSession{
		modelSelection: session.ModelSelection{ProviderID: "anthropic", ModelID: "claude-3"},
		hasModel:       true,
		modelsCatalog: session.ModelCatalog{
			Models: []session.ModelOption{
				{ProviderID: "anthropic", ProviderName: "Anthropic", ModelID: "claude-3", DisplayName: "Claude 3", Enabled: true},
				{ProviderID: "local", ProviderName: "Local", ModelID: "tiny", Enabled: false, UnavailableReason: "missing limits"},
				{ProviderID: "local", ProviderName: "Local", ModelID: "incomplete", Enabled: true, MissingLimits: []string{"contextWindow"}},
			},
			Current:    session.ModelSelection{ProviderID: "anthropic", ModelID: "claude-3"},
			HasCurrent: true,
		},
	}
	b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}

	catalog, status := readSessionModels(context.Background(), b, "s1")
	if !status.Available || status.Reason != "" {
		t.Fatalf("status = %#v, want an available catalog", status)
	}
	if !catalog.HasCurrent || catalog.Current != "anthropic/claude-3" {
		t.Fatalf("current pair = %q (has=%v), want anthropic/claude-3", catalog.Current, catalog.HasCurrent)
	}
	if len(catalog.Choices) != 3 {
		t.Fatalf("choices = %#v, want the catalog's three models", catalog.Choices)
	}
	if !catalog.Choices[0].Current || catalog.Choices[0].Reason != "" {
		t.Errorf("the current model is marked wrong: %#v", catalog.Choices[0])
	}
	if catalog.Choices[1].Current || catalog.Choices[1].Enabled || catalog.Choices[1].Reason != "missing limits" {
		t.Errorf("the unavailable model is drawn as selectable: %#v", catalog.Choices[1])
	}
	if catalog.Choices[2].Enabled || !strings.Contains(catalog.Choices[2].Reason, "contextWindow") {
		t.Errorf("a model with missing limits is drawn as selectable: %#v", catalog.Choices[2])
	}
}

// TestReadSessionModelsReportsAMissingCapability checks the degraded read: a
// missing provider.management capability leaves the attach reply's pair visible
// and carries the capability reason, never an empty catalog presented as real.
func TestReadSessionModelsReportsAMissingCapability(t *testing.T) {
	f := &fakeSession{
		modelSelection: session.ModelSelection{ProviderID: "anthropic", ModelID: "claude-3"},
		hasModel:       true,
		modelsErr:      &appserver.CapabilityError{Feature: "provider.management"},
	}
	b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}

	catalog, status := readSessionModels(context.Background(), b, "s1")
	if status.Available {
		t.Fatalf("status = %#v, want the missing capability", status)
	}
	if !strings.Contains(status.Reason, "provider.management") {
		t.Errorf("reason = %q, want the missing capability", status.Reason)
	}
	if !catalog.HasCurrent || catalog.Current != "anthropic/claude-3" {
		t.Fatalf("the attach reply's pair was dropped: %#v", catalog)
	}
	if len(catalog.Choices) != 0 {
		t.Fatalf("choices = %#v, want none without a catalog", catalog.Choices)
	}
}

// TestReadSessionModelsWithoutAPair checks a session whose attach reply carried
// no complete pair: the catalog stays available but the chip has no current
// model to name.
func TestReadSessionModelsWithoutAPair(t *testing.T) {
	f := &fakeSession{
		modelsCatalog: session.ModelCatalog{
			Models: []session.ModelOption{{ProviderID: "p", ModelID: "m", Enabled: true}},
		},
	}
	b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}

	catalog, status := readSessionModels(context.Background(), b, "s1")
	if !status.Available {
		t.Fatalf("status = %#v, want the catalog's answer", status)
	}
	if catalog.HasCurrent || catalog.Current != "" {
		t.Fatalf("the window invented a current pair: %#v", catalog)
	}
	if len(catalog.Choices) != 1 || catalog.Choices[0].Current {
		t.Fatalf("choices = %#v, want one model and no current marker", catalog.Choices)
	}
}

// TestSelectSessionFillsTheModelFacts checks the selection path: opening a
// session reads its model pair/catalog into the window, so the composer shows
// the new session's model rather than the previous one's.
func TestSelectSessionFillsTheModelFacts(t *testing.T) {
	selected := &fakeSession{
		view:           &dimui.State{Selected: "s2", Connection: dimui.ConnReady},
		modelSelection: session.ModelSelection{ProviderID: "p", ModelID: "m"},
		hasModel:       true,
		modelsCatalog:  session.ModelCatalog{Models: []session.ModelOption{{ProviderID: "p", ModelID: "m", Enabled: true}}},
	}
	b := &sessionBridge{summary: appserver.SessionSummary{SessionID: "s1"}}
	b.openSession = func(context.Context, *supervisor.Supervisor, string) (sessionActions, error) {
		return selected, nil
	}
	a := &app{
		state: &dimui.State{
			Sessions:   []dimui.Session{{ID: "s1", Title: "one"}, {ID: "s2", Title: "two"}},
			Selected:   "s1",
			Connection: dimui.ConnReady,
		},
		bridge: b,
	}
	var mu sync.Mutex
	a.updateHook = func(fn func(*dimui.State)) {
		mu.Lock()
		defer mu.Unlock()
		fn(a.state)
	}

	a.selectSession("s2")
	waitForState(t, a, &mu, "the selected session's model", func(state *dimui.State) bool {
		return state.Selected == "s2" && state.ModelCatalog.Current == "p/m"
	})
	mu.Lock()
	defer mu.Unlock()
	if !a.state.ModelStatus.Available {
		t.Errorf("model status = %#v, want the catalog's answer", a.state.ModelStatus)
	}
}
