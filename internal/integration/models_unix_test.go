//go:build unix

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
)

// modelManagementReply is one healthy builtin Provider with one selectable and
// one unavailable model, exactly as provider.management reports them.
const modelManagementReply = `{
	"drivers":[{"driverKind":"openai","authTypes":["bearer"],"remoteModelList":true}],
	"providers":[{
		"providerId":"p1","displayName":"Provider One","source":"builtin","credentialAuthority":"static",
		"version":3,"enabled":true,"actions":["update","refresh"],
		"connection":{"driverKind":"openai","baseUrl":"https://example.invalid","credentialConfigured":false,"headerNames":[]},
		"models":[
			{"model":{"modelId":"m1","displayName":"Model One","status":"active","capabilities":{"modalities":["text"],"toolCalling":true,"streaming":true,"structuredOutput":true,"vision":false,"audio":false,"reasoning":false,"contextWindow":200000,"maxOutputTokens":8192}},"source":"default","enabled":true,"endpoint":"openai","unavailableReason":null,"actions":["set_enabled"]},
			{"model":{"modelId":"m2","displayName":"Model Two","status":"active","capabilities":{"modalities":["text"],"toolCalling":true,"streaming":true,"structuredOutput":true,"vision":false,"audio":false,"reasoning":false}},"source":"default","enabled":true,"endpoint":"openai","missingLimits":["contextWindow","maxOutputTokens"],"unavailableReason":"limits_undeclared","actions":[]}
		],
		"preferences":{}
	}],
	"errors":[]
}`

const modelSelectionReply = `{"selection":{"providerId":"p1","modelId":"m1","version":3}}`

// startModelBridge starts a fake daemon that advertises the Provider management
// reads, then links a supervisor to it. The daemon must be configured before
// Start, so this cannot reuse startBridge.
func startModelBridge(t *testing.T, features ...string) (*fakeDaemon, *supervisor.Supervisor, context.Context) {
	t.Helper()
	home := fakeHome(t)
	link, err := supervisor.ResolveLink(home)
	if err != nil {
		t.Fatalf("resolve link: %v", err)
	}
	daemon := startFakeDaemon(t, link.Endpoint.Address)
	daemon.setFeatures(features...)
	sup, err := supervisor.New(supervisor.Config{ProductHome: home})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	ctx, cancel := testContext(t)
	t.Cleanup(func() {
		cancel()
		closeCtx, closeCancel := context.WithTimeout(context.Background(), testWait)
		defer closeCancel()
		_, _ = sup.Close(closeCtx)
	})
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return daemon, sup, ctx
}

func TestModelsPassThroughReadsCatalogAndSessionModel(t *testing.T) {
	daemon, sup, ctx := startModelBridge(t, "session.history", "provider.management", "provider.selection")
	daemon.setManagement(json.RawMessage(modelManagementReply))
	daemon.setSelection(json.RawMessage(modelSelectionReply))
	daemon.setAttachModel("p1", "m1")

	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	catalog, err := controller.Models(ctx)
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if len(catalog.Models) != 2 {
		t.Fatalf("models = %#v, want 2", catalog.Models)
	}
	if catalog.Models[0].ModelID != "m1" || catalog.Models[1].ModelID != "m2" {
		t.Fatalf("model order = %#v", catalog.Models)
	}
	if catalog.Models[1].UnavailableReason != "limits_undeclared" || len(catalog.Models[1].MissingLimits) != 2 {
		t.Fatalf("unavailable model = %#v", catalog.Models[1])
	}
	if !catalog.HasCurrent || catalog.Current.ProviderID != "p1" || catalog.Current.ModelID != "m1" {
		t.Fatalf("current = %#v (has %v)", catalog.Current, catalog.HasCurrent)
	}
	if catalog.Default == nil || catalog.Default.ModelID != "m1" {
		t.Fatalf("default = %#v", catalog.Default)
	}
	if selection, ok := controller.Model(); !ok || selection.ModelID != "m1" {
		t.Fatalf("Model = %#v (ok %v)", selection, ok)
	}
	if calls := daemon.callCount("provider.management"); calls != 1 {
		t.Fatalf("provider.management calls = %d, want 1", calls)
	}
	if calls := daemon.callCount("provider.selection"); calls != 1 {
		t.Fatalf("provider.selection calls = %d, want 1", calls)
	}
}

func TestModelsWithoutCapabilityIsWIP(t *testing.T) {
	daemon, sup, ctx := startModelBridge(t, "session.history")

	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	catalog, err := controller.Models(ctx)
	if !errors.Is(err, appserver.ErrCapabilityUnavailable) {
		t.Fatalf("error = %v, want ErrCapabilityUnavailable", err)
	}
	var capability *appserver.CapabilityError
	if !errors.As(err, &capability) || capability.Feature != "provider.management" {
		t.Fatalf("capability error = %#v", err)
	}
	if len(catalog.Models) != 0 {
		t.Fatalf("capability error returned a catalog: %#v", catalog)
	}
	if calls := daemon.callCount("provider.management"); calls != 0 {
		t.Fatalf("provider.management calls = %d, want 0", calls)
	}
}

func TestModelsSubscriptionChangedGetsOneRetry(t *testing.T) {
	daemon, sup, ctx := startModelBridge(t, "session.history", "provider.management", "provider.selection")
	daemon.setManagement(json.RawMessage(modelManagementReply))
	daemon.setSelection(json.RawMessage(modelSelectionReply))
	daemon.fail("provider.management", subscriptionChangedFault())

	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	catalog, err := controller.Models(ctx)
	if err != nil {
		t.Fatalf("Models after one replay-gap retry: %v", err)
	}
	if len(catalog.Models) != 2 {
		t.Fatalf("models = %#v, want 2", catalog.Models)
	}
	if calls := daemon.callCount("provider.management"); calls != 2 {
		t.Fatalf("provider.management calls = %d, want exactly 2 (one retry)", calls)
	}
}

func TestModelsRequireSessionAndContext(t *testing.T) {
	controller := &SessionController{}
	if _, err := controller.Models(nil); err == nil {
		t.Fatal("Models without a context should fail")
	}
	if _, err := controller.Models(context.Background()); err == nil {
		t.Fatal("Models without a session should fail")
	}
	if _, ok := controller.Model(); ok {
		t.Fatal("Model without a session should report no selection")
	}
	var nilController *SessionController
	if _, err := nilController.Models(context.Background()); err == nil {
		t.Fatal("Models on a nil controller should fail")
	}
	if _, ok := nilController.Model(); ok {
		t.Fatal("Model on a nil controller should report no selection")
	}
}
