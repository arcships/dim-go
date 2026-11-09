package session

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

func modelCatalogReply() appserver.ProviderManagementResult {
	contextWindow := int64(200000)
	maxOutput := int64(8192)
	providerEnabled := true
	providerDisabled := false
	unavailable := "limits_undeclared"
	return appserver.ProviderManagementResult{
		Providers: []appserver.ManagementProvider{
			{
				ProviderID:          "p1",
				DisplayName:         "Provider One",
				Source:              "builtin",
				CredentialAuthority: "static",
				Enabled:             &providerEnabled,
				Models: []appserver.ManagementModel{
					{
						Model: appserver.ModelDescriptor{
							ModelID:     "m1",
							DisplayName: "Model One",
							Status:      "active",
							Capabilities: appserver.CatalogCapabilities{
								ToolCalling:     true,
								Vision:          false,
								Reasoning:       true,
								ContextWindow:   &contextWindow,
								MaxOutputTokens: &maxOutput,
							},
						},
						Source:  "default",
						Enabled: true,
						Actions: []string{"set_enabled"},
					},
					{
						Model: appserver.ModelDescriptor{
							ModelID:     "m2",
							DisplayName: "Model Two",
							Status:      "beta",
							Capabilities: appserver.CatalogCapabilities{
								Vision: true,
							},
						},
						Source:            "user",
						Enabled:           false,
						MissingLimits:     []string{"contextWindow", "maxOutputTokens"},
						UnavailableReason: &unavailable,
						Actions:           []string{},
					},
				},
				Preferences: map[string]appserver.ModelPreferences{"m1": {ReasoningEffort: "high"}},
			},
			{
				ProviderID:          "p2",
				DisplayName:         "Provider Two",
				Source:              "custom",
				CredentialAuthority: "static",
				Enabled:             &providerDisabled,
				Models: []appserver.ManagementModel{
					{
						Model: appserver.ModelDescriptor{
							ModelID:     "m3",
							DisplayName: "Model Three",
							Status:      "active",
							Capabilities: appserver.CatalogCapabilities{
								ToolCalling: true,
							},
						},
						Source:  "default",
						Enabled: true,
						Actions: []string{},
					},
				},
			},
			{
				ProviderID:          "p3",
				DisplayName:         "No Catalog Record",
				Source:              "managed",
				CredentialAuthority: "account",
				Models:              nil,
			},
		},
		Errors: []appserver.ProviderRecordError{},
	}
}

func modelSelectionReply(providerID, modelID string, version int64) appserver.ProviderSelectionResult {
	if providerID == "" {
		return appserver.ProviderSelectionResult{Selection: nil}
	}
	return appserver.ProviderSelectionResult{Selection: &appserver.ProviderSelection{
		ProviderID: providerID,
		ModelID:    modelID,
		Version:    &version,
	}}
}

func modelTestFeatures(extra ...string) []string {
	features := []string{"session.history"}
	return append(features, extra...)
}

func TestModelsReadsCatalogAndDefault(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:     appserver.AttachResult{Sequence: int64Ptr(0), Model: &appserver.SessionModel{ProviderID: "p1", ModelID: "m1"}},
		History:    emptyHistory(t, 0),
		Management: modelCatalogReply(),
		Selection:  modelSelectionReply("p2", "m3", 7),
	}, modelTestFeatures("provider.management", "provider.selection"), OpenOptions{})

	catalog, err := s.Models(testContext(t))
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if len(catalog.Models) != 3 {
		t.Fatalf("models = %#v, want 3 options", catalog.Models)
	}
	// Provider name, then display name: Provider One's two, then Provider Two's.
	if catalog.Models[0].ModelID != "m1" || catalog.Models[1].ModelID != "m2" || catalog.Models[2].ModelID != "m3" {
		t.Fatalf("model order = %#v", catalog.Models)
	}
	first := catalog.Models[0]
	if first.ProviderID != "p1" || first.ProviderName != "Provider One" || first.ProviderEnabled == nil || !*first.ProviderEnabled {
		t.Fatalf("first option provider = %#v", first)
	}
	if !first.Enabled || first.Source != "default" || !first.ToolCalling || first.Vision || !first.Reasoning {
		t.Fatalf("first option = %#v", first)
	}
	if first.ContextWindow == nil || *first.ContextWindow != 200000 || first.MaxOutputTokens == nil || *first.MaxOutputTokens != 8192 {
		t.Fatalf("first option limits = %#v", first)
	}
	second := catalog.Models[1]
	if second.Enabled || second.UnavailableReason != "limits_undeclared" || len(second.MissingLimits) != 2 || !second.Vision {
		t.Fatalf("second option = %#v", second)
	}
	third := catalog.Models[2]
	if third.ProviderEnabled == nil || *third.ProviderEnabled {
		t.Fatalf("third option provider enabled = %#v, want false", third.ProviderEnabled)
	}
	if third.ContextWindow != nil || third.MaxOutputTokens != nil {
		t.Fatalf("undeclared limits must stay nil, got %#v", third)
	}
	if !catalog.HasCurrent || catalog.Current.ProviderID != "p1" || catalog.Current.ModelID != "m1" {
		t.Fatalf("current = %#v (has %v), want p1/m1", catalog.Current, catalog.HasCurrent)
	}
	if catalog.Default == nil || catalog.Default.ProviderID != "p2" || catalog.Default.ModelID != "m3" {
		t.Fatalf("default = %#v, want p2/m3", catalog.Default)
	}
	if !catalog.DefaultStatus.Available || catalog.DefaultStatus.Capability != "provider.selection" {
		t.Fatalf("default status = %#v", catalog.DefaultStatus)
	}
	if status := s.ModelsStatus(); !status.Available || status.Capability != "provider.management" {
		t.Fatalf("models status = %#v", status)
	}

	// Read-only: the provider records were fetched, never written.
	if calls := server.callsFor("provider.management"); len(calls) != 1 {
		t.Fatalf("provider.management calls = %d, want 1", len(calls))
	}
	if calls := server.callsFor("provider.selection"); len(calls) != 1 {
		t.Fatalf("provider.selection calls = %d, want 1", len(calls))
	}
	if calls := server.callsFor("session.model.update"); len(calls) != 0 {
		t.Fatalf("session.model.update was called %d times", len(calls))
	}
}

func TestModelsWithoutCapabilityReportsWIP(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:     appserver.AttachResult{Sequence: int64Ptr(0)},
		History:    emptyHistory(t, 0),
		Management: modelCatalogReply(),
		Selection:  modelSelectionReply("p1", "m1", 1),
	}, modelTestFeatures(), OpenOptions{})

	catalog, err := s.Models(testContext(t))
	if !errors.Is(err, appserver.ErrCapabilityUnavailable) {
		t.Fatalf("error = %v, want ErrCapabilityUnavailable", err)
	}
	var capability *appserver.CapabilityError
	if !errors.As(err, &capability) || capability.Feature != "provider.management" {
		t.Fatalf("capability error = %#v", err)
	}
	if len(catalog.Models) != 0 || catalog.Default != nil || catalog.HasCurrent {
		t.Fatalf("capability error returned a catalog: %#v", catalog)
	}
	status := s.ModelsStatus()
	if status.Available || status.Capability != "provider.management" || !strings.Contains(status.Reason, "provider.management") {
		t.Fatalf("models status = %#v, want an unavailable reason naming the feature", status)
	}
	if calls := server.callsFor("provider.management"); len(calls) != 0 {
		t.Fatalf("provider.management was called %d times without the feature", len(calls))
	}
}

func TestModelsSelectionCapabilityMissingDegrades(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:     appserver.AttachResult{Sequence: int64Ptr(0)},
		History:    emptyHistory(t, 0),
		Management: modelCatalogReply(),
		Selection:  modelSelectionReply("p1", "m1", 1),
	}, modelTestFeatures("provider.management"), OpenOptions{})

	catalog, err := s.Models(testContext(t))
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if len(catalog.Models) != 3 {
		t.Fatalf("models = %d, want the catalog to survive a missing selection capability", len(catalog.Models))
	}
	if catalog.Default != nil {
		t.Fatalf("default = %#v, want nil; no local default may be invented", catalog.Default)
	}
	if catalog.DefaultStatus.Available || !strings.Contains(catalog.DefaultStatus.Reason, "provider.selection") {
		t.Fatalf("default status = %#v", catalog.DefaultStatus)
	}
	if calls := server.callsFor("provider.selection"); len(calls) != 0 {
		t.Fatalf("provider.selection was called %d times without the feature", len(calls))
	}
}

func TestModelsSelectionReadFailureDegrades(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:       appserver.AttachResult{Sequence: int64Ptr(0)},
		History:      emptyHistory(t, 0),
		Management:   modelCatalogReply(),
		SelectionErr: unknownMethod(),
	}, modelTestFeatures("provider.management", "provider.selection"), OpenOptions{})

	catalog, err := s.Models(testContext(t))
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if len(catalog.Models) != 3 || catalog.Default != nil {
		t.Fatalf("catalog = %#v, want models and no default", catalog)
	}
	if catalog.DefaultStatus.Available || catalog.DefaultStatus.Reason == "" {
		t.Fatalf("default status = %#v, want a visible failure reason", catalog.DefaultStatus)
	}
	if calls := server.callsFor("provider.selection"); len(calls) != 1 {
		t.Fatalf("provider.selection calls = %d, want 1", len(calls))
	}
}

func TestModelsMalformedCatalogFailsClosed(t *testing.T) {
	_, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
		History: emptyHistory(t, 0),
		Management: json.RawMessage(`{"providers":[{"providerId":"p1","displayName":"P","source":"builtin","credentialAuthority":"static","models":[
			{"model":{"modelId":"m1","displayName":"M","status":"active"},"source":"default","enabled":true},
			{"model":{"displayName":"No Id","status":"active"},"source":"default","enabled":true}]}],"errors":[]}`),
	}, modelTestFeatures("provider.management"), OpenOptions{})

	catalog, err := s.Models(testContext(t))
	if !errors.Is(err, appserver.ErrProtocol) {
		t.Fatalf("error = %v, want ErrProtocol", err)
	}
	if len(catalog.Models) != 0 {
		t.Fatalf("protocol error returned models: %#v", catalog.Models)
	}
	if status := s.ModelsStatus(); status.Available {
		t.Fatalf("models status = %#v, want unavailable", status)
	}
}

func TestModelsToleratesUnknownFields(t *testing.T) {
	_, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
		History: emptyHistory(t, 0),
		Management: json.RawMessage(`{
			"drivers":[{"driverKind":"openai","authTypes":["bearer"],"remoteModelList":true,"futureDriverField":1}],
			"providers":[{
				"providerId":"p1","displayName":"Provider One","source":"builtin","credentialAuthority":"static",
				"version":2,"enabled":true,"actions":["update"],
				"connection":{"driverKind":"openai","baseUrl":"https://example.invalid","credentialConfigured":false,"headerNames":[],"futureConnectionField":true},
				"models":[{
					"model":{"modelId":"m1","displayName":"Model One","status":"active",
						"capabilities":{"modalities":["text"],"toolCalling":true,"streaming":true,"structuredOutput":true,"vision":false,"audio":false,"reasoning":true,"futureCapability":7},
						"metadata":{"description":"ignored"},"modelGuidePrompt":"ignored"},
					"source":"default","enabled":true,"endpoint":"openai","unavailableReason":null,
					"actions":["set_enabled"],"endpointOverride":{"kind":"new-api"},"futureModelField":"ok"
				}],
				"preferences":{"m1":{"reasoningEffort":"low","futurePreference":"ok"}},
				"futureProviderField":{"ignored":true}
			}],
			"errors":[],
			"futureTopLevel":[1,2,3]
		}`),
		Selection: json.RawMessage(`{"selection":{"providerId":"p1","modelId":"m1","version":2,"scope":"global"}}`),
	}, modelTestFeatures("provider.management", "provider.selection"), OpenOptions{})

	catalog, err := s.Models(testContext(t))
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if len(catalog.Models) != 1 || catalog.Models[0].ModelID != "m1" {
		t.Fatalf("models = %#v", catalog.Models)
	}
	if catalog.Default == nil || catalog.Default.ModelID != "m1" {
		t.Fatalf("default = %#v", catalog.Default)
	}
}

func TestModelReadsAttachSelection(t *testing.T) {
	t.Run("complete pair", func(t *testing.T) {
		_, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(0), Model: &appserver.SessionModel{ProviderID: "p1", ModelID: "m1"}},
			History: emptyHistory(t, 0),
		}, modelTestFeatures(), OpenOptions{})
		selection, ok := s.Model()
		if !ok || selection.ProviderID != "p1" || selection.ModelID != "m1" {
			t.Fatalf("Model = %#v (ok %v)", selection, ok)
		}
	})

	t.Run("partial pair is not a selection", func(t *testing.T) {
		_, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(0), Model: &appserver.SessionModel{ProviderID: "p1"}},
			History: emptyHistory(t, 0),
		}, modelTestFeatures(), OpenOptions{})
		if selection, ok := s.Model(); ok {
			t.Fatalf("Model = %#v, want no selection from a partial pair", selection)
		}
	})

	t.Run("absent model", func(t *testing.T) {
		_, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
			History: emptyHistory(t, 0),
		}, modelTestFeatures(), OpenOptions{})
		if selection, ok := s.Model(); ok {
			t.Fatalf("Model = %#v, want no selection", selection)
		}
	})
}

func TestModelsNilSession(t *testing.T) {
	var s *Session
	if _, err := s.Models(testContext(t)); err == nil {
		t.Fatal("Models on a nil session should fail")
	}
	if _, ok := s.Model(); ok {
		t.Fatal("Model on a nil session should report no selection")
	}
	if status := s.ModelsStatus(); status.Available {
		t.Fatalf("ModelsStatus = %#v, want unavailable", status)
	}
}

func TestModelsConcurrentReads(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:     appserver.AttachResult{Sequence: int64Ptr(0), Model: &appserver.SessionModel{ProviderID: "p1", ModelID: "m1"}},
		History:    emptyHistory(t, 0),
		Management: modelCatalogReply(),
		Selection:  modelSelectionReply("p1", "m1", 1),
	}, modelTestFeatures("provider.management", "provider.selection"), OpenOptions{})

	const readers = 8
	ctx := testContext(t)
	var wg sync.WaitGroup
	failures := make(chan string, readers)
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			catalog, err := s.Models(ctx)
			if err != nil {
				failures <- err.Error()
				return
			}
			if len(catalog.Models) != 3 || catalog.Default == nil || catalog.Default.ModelID != "m1" || !catalog.HasCurrent {
				failures <- "concurrent read returned a different catalog"
			}
		}()
	}
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Fatalf("concurrent read: %s", failure)
	}
	if calls := server.callsFor("provider.management"); len(calls) != readers {
		t.Fatalf("provider.management calls = %d, want %d", len(calls), readers)
	}
	if calls := server.callsFor("provider.selection"); len(calls) != readers {
		t.Fatalf("provider.selection calls = %d, want %d", len(calls), readers)
	}
}
