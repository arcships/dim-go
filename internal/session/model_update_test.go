package session

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

func modelUpdateSessionConfig(t *testing.T, result string) fakeConfig {
	t.Helper()
	return fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(0), Model: &appserver.SessionModel{ProviderID: "p1", ModelID: "m1"}},
		History: emptyHistory(t, 0),
		Mutations: map[string][]fakeMutation{
			"session.model.update": {{Result: json.RawMessage(result)}},
		},
	}
}

func sessionSummaries(providerID, modelID string) json.RawMessage {
	return json.RawMessage(`[{"sessionId":"s1","cwd":"/tmp","model":{"providerId":"` + providerID + `","modelId":"` + modelID + `"},"createdAt":1,"running":false,"sequence":0,"attachedClients":1,"updatedAt":1}]`)
}

func TestUpdateModelSendsTheMainPairWithoutGuessing(t *testing.T) {
	server, _, s := openTestSession(t, modelUpdateSessionConfig(t, `{"model":{"providerId":"p2","modelId":"m2"},"outcome":"adopted"}`), modelTestFeatures(appserver.ModelUpdateFeature), OpenOptions{})

	if !s.CanUpdateModel() {
		t.Fatal("CanUpdateModel = false with the feature advertised")
	}
	result, err := s.UpdateModel(testContext(t), "p2", "m2")
	if err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}
	if result.Model.ProviderID != "p2" || result.Model.ModelID != "m2" || result.Outcome != appserver.ModelOutcomeAdopted {
		t.Fatalf("result = %#v", result)
	}
	// The request's pair is not stored as a local selection.
	if selection, ok := s.Model(); !ok || selection.ProviderID != "p1" || selection.ModelID != "m1" {
		t.Fatalf("model after update = %#v (has %v), want the attach pair until the read-back", selection, ok)
	}
	calls := server.recorded("session.model.update")
	if len(calls) != 1 {
		t.Fatalf("session.model.update calls = %d, want 1", len(calls))
	}
	var params appserver.ModelUpdateParams
	if err := json.Unmarshal(calls[0].Params, &params); err != nil {
		t.Fatalf("params: %v", err)
	}
	if params.SessionID != "s1" || params.BranchID != appserver.MainBranchID || params.ProviderID != "p2" || params.ModelID != "m2" {
		t.Fatalf("params = %#v", params)
	}
}

func TestRefreshModelReadsTheAppServerFact(t *testing.T) {
	cfg := modelUpdateSessionConfig(t, `{"model":{"providerId":"p2","modelId":"m2"},"outcome":"adopted"}`)
	// The summary disagrees with the request on purpose: the stored pair must
	// come from the App Server read-back, not from what was asked for.
	cfg.Summaries = sessionSummaries("p9", "m9")
	server, _, s := openTestSession(t, cfg, modelTestFeatures(appserver.ModelUpdateFeature), OpenOptions{})

	if _, err := s.UpdateModel(testContext(t), "p2", "m2"); err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}
	selection, ok, err := s.RefreshModel(testContext(t))
	if err != nil {
		t.Fatalf("RefreshModel: %v", err)
	}
	if !ok || selection.ProviderID != "p9" || selection.ModelID != "m9" {
		t.Fatalf("refreshed model = %#v (ok %v), want p9/m9", selection, ok)
	}
	if current, has := s.Model(); !has || current.ProviderID != "p9" || current.ModelID != "m9" {
		t.Fatalf("Model = %#v (has %v), want the summary pair", current, has)
	}
	if calls := server.recorded("session.summaries"); len(calls) != 1 {
		t.Fatalf("session.summaries calls = %d, want 1", len(calls))
	}
}

func TestRefreshModelKeepsTheOldPairWhenTheReadFails(t *testing.T) {
	cfg := modelUpdateSessionConfig(t, `{"model":{"providerId":"p2","modelId":"m2"},"outcome":"adopted"}`)
	cfg.SummariesErr = &fakeRPCError{Code: -32000, Message: "storage failed", Data: json.RawMessage(`{"appError":"storage_failed"}`)}
	_, _, s := openTestSession(t, cfg, modelTestFeatures(appserver.ModelUpdateFeature), OpenOptions{})

	if _, _, err := s.RefreshModel(testContext(t)); err == nil {
		t.Fatal("RefreshModel succeeded with a failed summary read")
	}
	if selection, ok := s.Model(); !ok || selection.ProviderID != "p1" || selection.ModelID != "m1" {
		t.Fatalf("model = %#v (has %v), want the attach pair", selection, ok)
	}
}

func TestRefreshModelUnknownSessionKeepsTheOldPair(t *testing.T) {
	cfg := modelUpdateSessionConfig(t, `{"model":{"providerId":"p2","modelId":"m2"},"outcome":"adopted"}`)
	cfg.Summaries = sessionSummaries("p9", "m9")
	// Rewrite the list without this session.
	cfg.Summaries = json.RawMessage(`[{"sessionId":"other","cwd":"/tmp","model":{"providerId":"p9","modelId":"m9"},"createdAt":1,"running":false,"sequence":0,"attachedClients":1,"updatedAt":1}]`)
	_, _, s := openTestSession(t, cfg, modelTestFeatures(appserver.ModelUpdateFeature), OpenOptions{})

	if _, ok, err := s.RefreshModel(testContext(t)); err == nil || ok {
		t.Fatalf("RefreshModel = (%v, %v), want an error for an absent session", ok, err)
	}
	if selection, ok := s.Model(); !ok || selection.ProviderID != "p1" || selection.ModelID != "m1" {
		t.Fatalf("model = %#v (has %v), want the attach pair", selection, ok)
	}
}

func TestUpdateModelWithoutTheFeatureIsCapabilityWIP(t *testing.T) {
	server, _, s := openTestSession(t, modelUpdateSessionConfig(t, `{"model":{"providerId":"p2","modelId":"m2"},"outcome":"adopted"}`), modelTestFeatures(), OpenOptions{})

	if s.CanUpdateModel() {
		t.Fatal("CanUpdateModel = true without the feature")
	}
	_, err := s.UpdateModel(testContext(t), "p2", "m2")
	if !errors.Is(err, appserver.ErrCapabilityUnavailable) {
		t.Fatalf("error = %v, want ErrCapabilityUnavailable", err)
	}
	if calls := server.recorded("session.model.update"); len(calls) != 0 {
		t.Fatalf("session.model.update was sent %d times without the feature", len(calls))
	}
}

func TestUpdateModelUnknownResultIsNotResentAndCanBeReadBack(t *testing.T) {
	gate := newFakeGate()
	cfg := modelUpdateSessionConfig(t, `{"model":{"providerId":"p2","modelId":"m2"},"outcome":"adopted"}`)
	cfg.Summaries = sessionSummaries("p2", "m2")
	cfg.MutationGate = map[string]*fakeGate{"session.model.update": gate}
	server, _, s := openTestSession(t, cfg, modelTestFeatures(appserver.ModelUpdateFeature), OpenOptions{})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := s.UpdateModel(ctx, "p2", "m2")
	var unknown *appserver.UnknownResultError
	if !errors.As(err, &unknown) || !errors.Is(err, appserver.ErrUnknownResult) {
		t.Fatalf("error = %#v, want an UnknownResultError", err)
	}
	if unknown.ReadBack != "session.summaries" {
		t.Fatalf("read-back = %q, want session.summaries", unknown.ReadBack)
	}
	if calls := server.recorded("session.model.update"); len(calls) != 1 {
		t.Fatalf("session.model.update calls = %d, want exactly one (no resend)", len(calls))
	}
	// The read-back resolves the doubt through the App Server's summary.
	gate.open()
	selection, ok, err := s.RefreshModel(testContext(t))
	if err != nil || !ok || selection.ProviderID != "p2" || selection.ModelID != "m2" {
		t.Fatalf("RefreshModel = (%#v, %v, %v), want p2/m2", selection, ok, err)
	}
	if calls := server.recorded("session.model.update"); len(calls) != 1 {
		t.Fatalf("session.model.update calls = %d after read-back, want one", len(calls))
	}
}
