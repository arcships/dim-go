package main

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/session"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

func newModelTestApp(f *fakeSession) (*app, *sync.Mutex) {
	a := &app{
		state: &dimui.State{
			Selected:     "s1",
			ModelCatalog: dimui.ModelCatalog{Current: "p1/m1", HasCurrent: true},
			ModelStatus:  dimui.ModelStatus{Available: true},
		},
		bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}},
	}
	mu := &sync.Mutex{}
	a.updateHook = func(fn func(*dimui.State)) {
		mu.Lock()
		defer mu.Unlock()
		fn(a.state)
	}
	return a, mu
}

// TestSelectModelPublishesTheAppServerReadBack checks the success path: one
// session.model.update, one session.summaries read-back, and a chip that names
// the pair the App Server reported rather than the requested one.
func TestSelectModelPublishesTheAppServerReadBack(t *testing.T) {
	f := &fakeSession{
		canUpdateModel: true,
		modelSelection: session.ModelSelection{ProviderID: "p1", ModelID: "m1"},
		hasModel:       true,
		updateModelResult: appserver.ModelUpdateResult{
			Model:   appserver.SessionModel{ProviderID: "p1", ModelID: "m2"},
			Outcome: appserver.ModelOutcomeAdopted,
		},
		// The read-back deliberately disagrees with the request: the window
		// must show the App Server's fact, not the pair it asked for.
		refreshModelSelection: session.ModelSelection{ProviderID: "p2", ModelID: "m3"},
		refreshModelHasModel:  true,
		modelsCatalog: session.ModelCatalog{Models: []session.ModelOption{
			{ProviderID: "p1", ModelID: "m1", DisplayName: "One", Enabled: true},
			{ProviderID: "p2", ModelID: "m3", DisplayName: "Three", Enabled: true},
		}},
	}
	a, mu := newModelTestApp(f)

	a.selectModel("p1", "m2")
	waitForState(t, a, mu, "the read-back model", func(state *dimui.State) bool {
		return state.ModelCatalog.Current == "p2/m3"
	})
	mu.Lock()
	defer mu.Unlock()
	if f.updateModelCalls != 1 || f.updateModelProvider != "p1" || f.updateModelID != "m2" {
		t.Fatalf("update calls = %d (%s/%s), want one p1/m2", f.updateModelCalls, f.updateModelProvider, f.updateModelID)
	}
	if f.refreshModelCalls != 1 {
		t.Fatalf("read-back calls = %d, want 1", f.refreshModelCalls)
	}
	if a.state.ActionError != "" {
		t.Fatalf("action error = %q, want none", a.state.ActionError)
	}
	if len(a.state.ActionNotices) == 0 || a.state.ActionNotices[len(a.state.ActionNotices)-1].Kind != "model.selected" {
		t.Fatalf("notices = %#v, want a model.selected outcome", a.state.ActionNotices)
	}
}

// TestSelectModelFailureKeepsThePreviousChip checks the definitive-failure path:
// the App Server refused or lacks the method, so the old pair stays and the
// error names the WIP method.
func TestSelectModelFailureKeepsThePreviousChip(t *testing.T) {
	f := &fakeSession{
		canUpdateModel: true,
		modelSelection: session.ModelSelection{ProviderID: "p1", ModelID: "m1"},
		hasModel:       true,
		updateModelErr: &appserver.CapabilityError{Feature: appserver.ModelUpdateFeature},
	}
	a, mu := newModelTestApp(f)

	a.selectModel("p1", "m2")
	waitForState(t, a, mu, "the model error", func(state *dimui.State) bool {
		return strings.Contains(state.ActionError, appserver.ModelUpdateFeature)
	})
	mu.Lock()
	defer mu.Unlock()
	if a.state.ModelCatalog.Current != "p1/m1" || !a.state.ModelCatalog.HasCurrent {
		t.Fatalf("chip = %#v, want the previous p1/m1", a.state.ModelCatalog)
	}
	if !strings.Contains(a.state.ActionError, "WIP") {
		t.Fatalf("error = %q, want the WIP reason", a.state.ActionError)
	}
	if f.updateModelCalls != 1 || f.refreshModelCalls != 0 {
		t.Fatalf("calls = update %d, read-back %d; want 1 and 0", f.updateModelCalls, f.refreshModelCalls)
	}
}

// TestSelectModelUnknownResultReadsBackWithoutResending checks §14: the lost
// write is resolved by reading the pair back, and the request is not resent.
func TestSelectModelUnknownResultReadsBackWithoutResending(t *testing.T) {
	f := &fakeSession{
		canUpdateModel: true,
		modelSelection: session.ModelSelection{ProviderID: "p1", ModelID: "m1"},
		hasModel:       true,
		updateModelErr: &appserver.UnknownResultError{
			Method:   appserver.MutationSessionModelUpdate,
			ReadBack: "session.summaries",
			Err:      errors.New("request timeout"),
		},
		refreshModelSelection: session.ModelSelection{ProviderID: "p1", ModelID: "m2"},
		refreshModelHasModel:  true,
		modelsCatalog: session.ModelCatalog{Models: []session.ModelOption{
			{ProviderID: "p1", ModelID: "m2", DisplayName: "Two", Enabled: true},
		}},
	}
	a, mu := newModelTestApp(f)

	a.selectModel("p1", "m2")
	waitForState(t, a, mu, "the read-back model", func(state *dimui.State) bool {
		return state.ModelCatalog.Current == "p1/m2"
	})
	mu.Lock()
	defer mu.Unlock()
	if f.updateModelCalls != 1 {
		t.Fatalf("session.model.update calls = %d, want exactly one", f.updateModelCalls)
	}
	if f.refreshModelCalls != 1 {
		t.Fatalf("read-back calls = %d, want 1", f.refreshModelCalls)
	}
	if !strings.Contains(a.state.ActionError, "not resent") {
		t.Fatalf("error = %q, want the no-resend statement", a.state.ActionError)
	}
}

// TestSelectModelReadBackFailureUsesTheDaemonReply checks the degraded success
// path: the update was adopted but session.summaries failed, so the reply's own
// pair still moves the chip and the failure is reported.
func TestSelectModelReadBackFailureUsesTheDaemonReply(t *testing.T) {
	f := &fakeSession{
		canUpdateModel: true,
		modelSelection: session.ModelSelection{ProviderID: "p1", ModelID: "m1"},
		hasModel:       true,
		updateModelResult: appserver.ModelUpdateResult{
			Model:   appserver.SessionModel{ProviderID: "p1", ModelID: "m2"},
			Outcome: appserver.ModelOutcomeAdopted,
		},
		refreshModelErr: errors.New("session.summaries unavailable"),
		modelsCatalog: session.ModelCatalog{Models: []session.ModelOption{
			{ProviderID: "p1", ModelID: "m1", DisplayName: "One", Enabled: true},
			{ProviderID: "p1", ModelID: "m2", DisplayName: "Two", Enabled: true},
		}},
	}
	a, mu := newModelTestApp(f)

	a.selectModel("p1", "m2")
	waitForState(t, a, mu, "the adopted reply pair", func(state *dimui.State) bool {
		return state.ModelCatalog.Current == "p1/m2"
	})
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(a.state.ActionError, "reading the current model back failed") {
		t.Fatalf("error = %q, want the read-back failure", a.state.ActionError)
	}
	for _, choice := range a.state.ModelCatalog.Choices {
		if choice.ModelID == "m2" && !choice.Current {
			t.Fatalf("choice %#v is not marked current", choice)
		}
	}
}

// TestReadSessionModelsKeepsSelectionWIPWithoutTheFeature checks the catalog
// read still succeeds while the picker is marked unselectable: the App Server
// advertises provider.management but not session.model.update.
func TestReadSessionModelsKeepsSelectionWIPWithoutTheFeature(t *testing.T) {
	f := &fakeSession{
		modelSelection: session.ModelSelection{ProviderID: "p1", ModelID: "m1"},
		hasModel:       true,
		modelsCatalog: session.ModelCatalog{Models: []session.ModelOption{
			{ProviderID: "p1", ModelID: "m1", DisplayName: "One", Enabled: true},
		}},
	}
	b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}

	catalog, status := readSessionModels(context.Background(), b, "s1")
	if !status.Available || status.Selectable {
		t.Fatalf("status = %#v, want a readable but unselectable catalog", status)
	}
	if !strings.Contains(status.Reason, appserver.ModelUpdateFeature) {
		t.Fatalf("reason = %q, want the missing session.model.update", status.Reason)
	}
	if !catalog.HasCurrent || catalog.Current != "p1/m1" {
		t.Fatalf("current pair = %#v, want the App Server's p1/m1", catalog)
	}
}

// TestSelectModelWithoutASessionReportsNothingToWrite covers the guard before
// any request: a window without a selection reports the gap and sends nothing.
func TestSelectModelWithoutASessionReportsNothingToWrite(t *testing.T) {
	f := &fakeSession{}
	a, mu := newModelTestApp(f)
	a.state.Selected = ""

	a.selectModel("p1", "m2")
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(a.state.ActionError, "no session") {
		t.Fatalf("error = %q, want the no-session reason", a.state.ActionError)
	}
	if f.updateModelCalls != 0 || f.refreshModelCalls != 0 {
		t.Fatalf("calls = update %d, read-back %d; want none", f.updateModelCalls, f.refreshModelCalls)
	}
}
