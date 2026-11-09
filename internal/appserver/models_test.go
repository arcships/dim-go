package appserver

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

func managementRaw(t *testing.T, extra string) json.RawMessage {
	t.Helper()
	return json.RawMessage(`{
		"drivers":[{"driverKind":"openai","authTypes":["bearer"],"remoteModelList":true}],
		"providers":[
			{
				"providerId":"p1",
				"displayName":"Provider One",
				"source":"builtin",
				"credentialAuthority":"static",
				"version":4,
				"enabled":true,
				"actions":["update","refresh"],
				"connection":{"driverKind":"openai","baseUrl":"https://example.invalid","credentialConfigured":true,"headerNames":[]},
				"models":[
					{
						"model":{"modelId":"m1","displayName":"Model One","status":"active","capabilities":{"modalities":["text"],"toolCalling":true,"streaming":true,"structuredOutput":true,"vision":false,"audio":false,"reasoning":true,"contextWindow":200000,"maxOutputTokens":8192,"providerState":{"unknown":true}},"metadata":{"description":"ignored"}},
						"source":"default","enabled":true,"endpoint":"openai","unavailableReason":null,"actions":["set_enabled"],
						"endpointOverride":{"kind":"ignored"}
					},
					{
						"model":{"modelId":"m2","displayName":"Model Two","status":"beta","capabilities":{"modalities":["text"],"toolCalling":false,"streaming":true,"structuredOutput":false,"vision":true,"audio":false,"reasoning":false}},
						"source":"user","enabled":false,"endpoint":null,"missingLimits":["contextWindow","maxOutputTokens"],"unavailableReason":"limits_undeclared","actions":[]
					}
				],
				"preferences":{"m1":{"reasoningEffort":"high"}}
			}
		],
		"errors":[],` + extra + `"unknownTopLevel":{"ignored":true}
	}`)
}

func TestDecodeProviderManagementResult(t *testing.T) {
	result, err := DecodeProviderManagementResult(managementRaw(t, ""))
	if err != nil {
		t.Fatalf("DecodeProviderManagementResult: %v", err)
	}
	if len(result.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(result.Providers))
	}
	provider := result.Providers[0]
	if provider.ProviderID != "p1" || provider.DisplayName != "Provider One" || provider.Source != "builtin" || provider.CredentialAuthority != "static" {
		t.Fatalf("provider = %#v", provider)
	}
	if provider.Version == nil || *provider.Version != 4 || provider.Enabled == nil || !*provider.Enabled {
		t.Fatalf("provider version/enabled = %#v", provider)
	}
	if len(provider.Models) != 2 {
		t.Fatalf("models = %d, want 2", len(provider.Models))
	}
	first := provider.Models[0]
	if first.Model.ModelID != "m1" || first.Model.DisplayName != "Model One" || first.Model.Status != "active" {
		t.Fatalf("first model = %#v", first.Model)
	}
	if first.Model.Capabilities.ContextWindow == nil || *first.Model.Capabilities.ContextWindow != 200000 {
		t.Fatalf("context window = %v", first.Model.Capabilities.ContextWindow)
	}
	if !first.Enabled || first.Source != "default" {
		t.Fatalf("first model enabled/source = %#v", first)
	}
	second := provider.Models[1]
	if second.Enabled || second.UnavailableReason == nil || *second.UnavailableReason != "limits_undeclared" {
		t.Fatalf("second model = %#v", second)
	}
	if len(second.MissingLimits) != 2 {
		t.Fatalf("missing limits = %#v", second.MissingLimits)
	}
	if provider.Preferences["m1"].ReasoningEffort != "high" {
		t.Fatalf("preferences = %#v", provider.Preferences)
	}
}

func TestDecodeProviderManagementResultRejectsMalformed(t *testing.T) {
	base := `{"providers":[{"providerId":"p1","displayName":"P","source":"builtin","credentialAuthority":"static","models":[]}],"errors":[]}`
	invalid := []struct {
		name string
		raw  string
	}{
		{name: "not an object", raw: `[]`},
		{name: "missing providers", raw: `{"errors":[]}`},
		{name: "providers is null", raw: `{"providers":null,"errors":[]}`},
		{name: "duplicate provider", raw: `{"providers":[{"providerId":"p1","displayName":"P","source":"builtin","credentialAuthority":"static"},{"providerId":"p1","displayName":"Q","source":"builtin","credentialAuthority":"static"}],"errors":[]}`},
		{name: "bad provider source", raw: `{"providers":[{"providerId":"p1","displayName":"P","source":"peer","credentialAuthority":"static"}],"errors":[]}`},
		{name: "bad credential authority", raw: `{"providers":[{"providerId":"p1","displayName":"P","source":"builtin","credentialAuthority":"session"}],"errors":[]}`},
		{name: "negative version", raw: `{"providers":[{"providerId":"p1","displayName":"P","source":"builtin","credentialAuthority":"static","version":-1}],"errors":[]}`},
		{name: "model missing id", raw: `{"providers":[{"providerId":"p1","displayName":"P","source":"builtin","credentialAuthority":"static","models":[{"model":{"displayName":"M","status":"active"},"source":"default","enabled":true}]}],"errors":[]}`},
		{name: "model missing status", raw: `{"providers":[{"providerId":"p1","displayName":"P","source":"builtin","credentialAuthority":"static","models":[{"model":{"modelId":"m","displayName":"M"},"source":"default","enabled":true}]}],"errors":[]}`},
		{name: "model bad source", raw: `{"providers":[{"providerId":"p1","displayName":"P","source":"builtin","credentialAuthority":"static","models":[{"model":{"modelId":"m","displayName":"M","status":"active"},"source":"local","enabled":true}]}],"errors":[]}`},
		{name: "model zero limit", raw: `{"providers":[{"providerId":"p1","displayName":"P","source":"builtin","credentialAuthority":"static","models":[{"model":{"modelId":"m","displayName":"M","status":"active","capabilities":{"contextWindow":0}},"source":"default","enabled":true}]}],"errors":[]}`},
		{name: "unknown missing limit name", raw: `{"providers":[{"providerId":"p1","displayName":"P","source":"builtin","credentialAuthority":"static","models":[{"model":{"modelId":"m","displayName":"M","status":"active"},"source":"default","enabled":true,"missingLimits":["temperature"]}]}],"errors":[]}`},
		{name: "error missing message", raw: `{"providers":[],"errors":[{"providerId":"p1"}]}`},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecodeProviderManagementResult(json.RawMessage(tc.raw))
			if !errors.Is(err, ErrProtocol) {
				t.Fatalf("error = %v, want ErrProtocol", err)
			}
			if result.Providers != nil {
				t.Fatalf("protocol error returned providers: %#v", result.Providers)
			}
		})
	}
	if _, err := DecodeProviderManagementResult(json.RawMessage(base)); err != nil {
		t.Fatalf("valid baseline rejected: %v", err)
	}
}

func TestProviderManagementHelper(t *testing.T) {
	t.Run("reads one non-mutating request", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:         fakeHello("provider.management"),
			ManagementRaw: managementRaw(t, ""),
		})
		result, err := ProviderManagement(ctx, client, ProviderManagementParams{}, []string{"provider.management"})
		if err != nil {
			t.Fatalf("ProviderManagement: %v", err)
		}
		if len(result.Providers) != 1 {
			t.Fatalf("providers = %d, want 1", len(result.Providers))
		}
		calls := server.recordedCalls("provider.management")
		if len(calls) != 1 {
			t.Fatalf("provider.management calls = %d, want 1", len(calls))
		}
		if params := rawParams(t, calls[0]); len(params) != 0 {
			t.Fatalf("params = %s, want an empty scope", calls[0].Params)
		}
	})

	t.Run("provider id scope is sent verbatim", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:         fakeHello("provider.management"),
			ManagementRaw: managementRaw(t, ""),
		})
		if _, err := ProviderManagement(ctx, client, ProviderManagementParams{ProviderID: "p1"}, []string{"provider.management"}); err != nil {
			t.Fatalf("ProviderManagement: %v", err)
		}
		calls := server.recordedCalls("provider.management")
		if len(calls) != 1 || string(rawParams(t, calls[0])["providerId"]) != `"p1"` {
			t.Fatalf("params = %s, want providerId p1", calls[0].Params)
		}
	})

	t.Run("missing feature is a capability error and sends nothing", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:         fakeHello("session.catalog"),
			ManagementRaw: managementRaw(t, ""),
		})
		result, err := ProviderManagement(ctx, client, ProviderManagementParams{}, []string{"session.catalog"})
		if !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("error = %v, want ErrCapabilityUnavailable", err)
		}
		var capability *CapabilityError
		if !errors.As(err, &capability) || capability.Feature != "provider.management" {
			t.Fatalf("capability error = %#v", err)
		}
		if result.Providers != nil {
			t.Fatalf("capability error returned a result: %#v", result)
		}
		if calls := server.recordedCalls("provider.management"); len(calls) != 0 {
			t.Fatalf("provider.management was called %d times without the feature", len(calls))
		}
	})

	t.Run("daemon error surfaces once without retry", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:           fakeHello("provider.management"),
			ManagementFault: fakeUnknownMethod(),
		})
		result, err := ProviderManagement(ctx, client, ProviderManagementParams{}, []string{"provider.management"})
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32601 {
			t.Fatalf("error = %v, want RPCError -32601", err)
		}
		if result.Providers != nil {
			t.Fatalf("error returned a partial result: %#v", result)
		}
		if calls := server.recordedCalls("provider.management"); len(calls) != 1 {
			t.Fatalf("provider.management calls = %d, want exactly 1", len(calls))
		}
	})

	t.Run("nil client is a protocol error", func(t *testing.T) {
		_, err := ProviderManagement(testContext(t), nil, ProviderManagementParams{}, []string{"provider.management"})
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
	})
}

func TestDecodeProviderSelectionResult(t *testing.T) {
	valid := []struct {
		name  string
		raw   string
		empty bool
	}{
		{name: "selection", raw: `{"selection":{"providerId":"p1","modelId":"m1","version":3}}`},
		{name: "null selection", raw: `{"selection":null}`, empty: true},
		{name: "unknown fields are ignored", raw: `{"selection":{"providerId":"p1","modelId":"m1","version":0,"scope":"global"}}`},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecodeProviderSelectionResult(json.RawMessage(tc.raw))
			if err != nil {
				t.Fatalf("DecodeProviderSelectionResult: %v", err)
			}
			if tc.empty {
				if result.Selection != nil {
					t.Fatalf("selection = %#v, want nil", result.Selection)
				}
				return
			}
			if result.Selection == nil || result.Selection.ProviderID != "p1" || result.Selection.ModelID != "m1" {
				t.Fatalf("selection = %#v", result.Selection)
			}
		})
	}

	invalid := []struct {
		name string
		raw  string
	}{
		{name: "not an object", raw: `[]`},
		{name: "missing provider", raw: `{"selection":{"modelId":"m1","version":1}}`},
		{name: "missing model", raw: `{"selection":{"providerId":"p1","version":1}}`},
		{name: "missing version", raw: `{"selection":{"providerId":"p1","modelId":"m1"}}`},
		{name: "negative version", raw: `{"selection":{"providerId":"p1","modelId":"m1","version":-1}}`},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodeProviderSelectionResult(json.RawMessage(tc.raw)); !errors.Is(err, ErrProtocol) {
				t.Fatalf("error = %v, want ErrProtocol", err)
			}
		})
	}
}

func TestProviderSelectionHelper(t *testing.T) {
	t.Run("reads the stored default", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:        fakeHello("provider.selection"),
			SelectionRaw: json.RawMessage(`{"selection":{"providerId":"p1","modelId":"m1","version":2}}`),
		})
		result, err := ReadProviderSelection(ctx, client, ProviderSelectionParams{}, []string{"provider.selection"})
		if err != nil {
			t.Fatalf("ProviderSelection: %v", err)
		}
		if result.Selection == nil || result.Selection.ModelID != "m1" || result.Selection.Version == nil || *result.Selection.Version != 2 {
			t.Fatalf("selection = %#v", result.Selection)
		}
		if calls := server.recordedCalls("provider.selection"); len(calls) != 1 {
			t.Fatalf("provider.selection calls = %d, want 1", len(calls))
		}
	})

	t.Run("cwd scope is sent verbatim", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:        fakeHello("provider.selection"),
			SelectionRaw: json.RawMessage(`{"selection":null}`),
		})
		if _, err := ReadProviderSelection(ctx, client, ProviderSelectionParams{Cwd: "/work"}, []string{"provider.selection"}); err != nil {
			t.Fatalf("ProviderSelection: %v", err)
		}
		calls := server.recordedCalls("provider.selection")
		if len(calls) != 1 || string(rawParams(t, calls[0])["cwd"]) != `"/work"` {
			t.Fatalf("params = %s, want cwd /work", calls[0].Params)
		}
	})

	t.Run("missing feature is a capability error and sends nothing", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:        fakeHello("provider.management"),
			SelectionRaw: json.RawMessage(`{"selection":null}`),
		})
		_, err := ReadProviderSelection(ctx, client, ProviderSelectionParams{}, []string{"provider.management"})
		if !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("error = %v, want ErrCapabilityUnavailable", err)
		}
		if calls := server.recordedCalls("provider.selection"); len(calls) != 0 {
			t.Fatalf("provider.selection was called %d times without the feature", len(calls))
		}
	})

	t.Run("malformed result is a protocol error", func(t *testing.T) {
		ctx := testContext(t)
		_, client := newFakeServer(t, fakeConfig{
			Hello:        fakeHello("provider.selection"),
			SelectionRaw: json.RawMessage(`{"selection":{"providerId":"p1"}}`),
		})
		result, err := ReadProviderSelection(ctx, client, ProviderSelectionParams{}, []string{"provider.selection"})
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if result.Selection != nil {
			t.Fatalf("protocol error returned a selection: %#v", result.Selection)
		}
	})

	t.Run("daemon error surfaces once without retry", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:          fakeHello("provider.selection"),
			SelectionFault: fakeBadParams(),
		})
		_, err := ReadProviderSelection(ctx, client, ProviderSelectionParams{}, []string{"provider.selection"})
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32602 {
			t.Fatalf("error = %v, want RPCError -32602", err)
		}
		if calls := server.recordedCalls("provider.selection"); len(calls) != 1 {
			t.Fatalf("provider.selection calls = %d, want exactly 1", len(calls))
		}
	})
}

// TestProviderManagementConcurrentReads pins that the helper shares one Client
// safely: every caller gets its own decoded snapshot.
func TestProviderManagementConcurrentReads(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, fakeConfig{
		Hello:         fakeHello("provider.management"),
		ManagementRaw: managementRaw(t, ""),
	})
	const readers = 8
	var wg sync.WaitGroup
	errs := make(chan error, readers)
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := ProviderManagement(ctx, client, ProviderManagementParams{}, []string{"provider.management"})
			if err != nil {
				errs <- err
				return
			}
			if len(result.Providers) != 1 || len(result.Providers[0].Models) != 2 {
				errs <- errors.New("concurrent read returned a different catalog")
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent read: %v", err)
	}
	if calls := server.recordedCalls("provider.management"); len(calls) != readers {
		t.Fatalf("provider.management calls = %d, want %d", len(calls), readers)
	}
}
