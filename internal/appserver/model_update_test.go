package appserver

import (
	"encoding/json"
	"errors"
	"testing"
)

const modelUpdateReply = `{"model":{"providerId":"p1","modelId":"m2"},"outcome":"adopted"}`

func modelUpdateConfig(result string) fakeConfig {
	return fakeConfig{
		Hello: fakeHello(ModelUpdateFeature),
		Mutations: map[string][]fakeMutation{
			string(MutationSessionModelUpdate): {{Result: json.RawMessage(result)}},
		},
	}
}

func TestModelUpdateSendsTheMainBranchPair(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, modelUpdateConfig(modelUpdateReply))

	result, err := ModelUpdate(ctx, client, ModelUpdateParams{SessionID: "s1", ProviderID: "p1", ModelID: "m2"}, []string{ModelUpdateFeature})
	if err != nil {
		t.Fatalf("ModelUpdate: %v", err)
	}
	if result.Model.ProviderID != "p1" || result.Model.ModelID != "m2" || result.Outcome != ModelOutcomeAdopted {
		t.Fatalf("result = %#v", result)
	}
	calls := server.recordedCalls(string(MutationSessionModelUpdate))
	if len(calls) != 1 {
		t.Fatalf("session.model.update calls = %d, want 1", len(calls))
	}
	params := rawParams(t, calls[0])
	if string(params["sessionId"]) != `"s1"` || string(params["providerId"]) != `"p1"` || string(params["modelId"]) != `"m2"` {
		t.Fatalf("params = %s", calls[0].Params)
	}
	// The omitted branch is sent as main, matching Desktop's setModel.
	if string(params["branchId"]) != `"main"` {
		t.Fatalf("branchId = %s, want main", params["branchId"])
	}
}

func TestModelUpdateWithoutTheFeatureIsCapabilityWIP(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, modelUpdateConfig(modelUpdateReply))

	_, err := ModelUpdate(ctx, client, ModelUpdateParams{SessionID: "s1", ProviderID: "p1", ModelID: "m2"}, nil)
	if !errors.Is(err, ErrCapabilityUnavailable) {
		t.Fatalf("error = %v, want ErrCapabilityUnavailable", err)
	}
	var capability *CapabilityError
	if !errors.As(err, &capability) || capability.Feature != ModelUpdateFeature {
		t.Fatalf("capability error = %#v", err)
	}
	if calls := server.recordedCalls(string(MutationSessionModelUpdate)); len(calls) != 0 {
		t.Fatalf("session.model.update was sent %d times without the feature", len(calls))
	}
}

func TestModelUpdateRejectsInvalidInputBeforeSending(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, modelUpdateConfig(modelUpdateReply))
	features := []string{ModelUpdateFeature}
	cases := []ModelUpdateParams{
		{BranchID: MainBranchID, ProviderID: "p1", ModelID: "m2"},
		{SessionID: "s1", BranchID: MainBranchID, ModelID: "m2"},
		{SessionID: "s1", BranchID: MainBranchID, ProviderID: "p1"},
		{SessionID: "s1", BranchID: "side", ProviderID: "p1", ModelID: "m2"},
	}
	for _, params := range cases {
		if _, err := ModelUpdate(ctx, client, params, features); !errors.Is(err, ErrProtocol) {
			t.Fatalf("params %#v: error = %v, want ErrProtocol", params, err)
		}
	}
	if calls := server.recordedCalls(string(MutationSessionModelUpdate)); len(calls) != 0 {
		t.Fatalf("invalid requests reached the daemon: %d", len(calls))
	}
}

func TestModelUpdateMapsDaemonErrors(t *testing.T) {
	ctx := testContext(t)
	t.Run("invalid params is a definitive bad request", func(t *testing.T) {
		cfg := modelUpdateConfig(modelUpdateReply)
		cfg.Mutations[string(MutationSessionModelUpdate)] = []fakeMutation{{Fault: fakeBadParams()}}
		_, client := newFakeServer(t, cfg)
		_, err := ModelUpdate(ctx, client, ModelUpdateParams{SessionID: "s1", ProviderID: "p1", ModelID: "m2"}, []string{ModelUpdateFeature})
		var mutation *MutationError
		if !errors.As(err, &mutation) || mutation.AppError != "" || mutation.Mapped.Class != ErrorClassBadRequest {
			t.Fatalf("error = %#v, want a bad request MutationError", err)
		}
		if errors.Is(err, ErrUnknownResult) {
			t.Fatalf("-32602 must not be an unknown result: %v", err)
		}
	})
	t.Run("unknown method stays capability WIP", func(t *testing.T) {
		cfg := modelUpdateConfig(modelUpdateReply)
		cfg.Mutations[string(MutationSessionModelUpdate)] = []fakeMutation{{Fault: fakeUnknownMethod()}}
		_, client := newFakeServer(t, cfg)
		_, err := ModelUpdate(ctx, client, ModelUpdateParams{SessionID: "s1", ProviderID: "p1", ModelID: "m2"}, []string{ModelUpdateFeature})
		if !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("error = %v, want ErrCapabilityUnavailable", err)
		}
	})
}

func TestModelUpdateUnknownResultNamesTheReadBack(t *testing.T) {
	ctx := testContext(t)
	t.Run("lost transport", func(t *testing.T) {
		cfg := modelUpdateConfig(modelUpdateReply)
		cfg.CloseOnMethod = map[string]bool{string(MutationSessionModelUpdate): true}
		_, client := newFakeServer(t, cfg)
		_, err := ModelUpdate(ctx, client, ModelUpdateParams{SessionID: "s1", ProviderID: "p1", ModelID: "m2"}, []string{ModelUpdateFeature})
		var unknown *UnknownResultError
		if !errors.As(err, &unknown) || !errors.Is(err, ErrUnknownResult) {
			t.Fatalf("error = %#v, want an UnknownResultError", err)
		}
		if unknown.ReadBack != "session.summaries" {
			t.Fatalf("read-back = %q, want session.summaries", unknown.ReadBack)
		}
	})
	t.Run("malformed reply", func(t *testing.T) {
		cfg := modelUpdateConfig(`{"model":{"providerId":"p1","modelId":"m2"}}`)
		_, client := newFakeServer(t, cfg)
		_, err := ModelUpdate(ctx, client, ModelUpdateParams{SessionID: "s1", ProviderID: "p1", ModelID: "m2"}, []string{ModelUpdateFeature})
		var unknown *UnknownResultError
		if !errors.As(err, &unknown) || unknown.ReadBack != "session.summaries" {
			t.Fatalf("error = %#v, want a session.summaries read-back", err)
		}
	})
}

func TestDecodeModelUpdateResultRejectsMalformed(t *testing.T) {
	for _, raw := range []string{
		`[]`,
		`{"model":{"providerId":"p1","modelId":"m2"}}`,
		`{"outcome":"adopted"}`,
		`{"model":{"providerId":"","modelId":"m2"},"outcome":"adopted"}`,
		`{"model":{"providerId":"p1","modelId":"m2"},"outcome":"applied"}`,
	} {
		if _, err := DecodeModelUpdateResult(json.RawMessage(raw)); err == nil {
			t.Fatalf("DecodeModelUpdateResult(%s) succeeded, want an error", raw)
		}
	}
}
