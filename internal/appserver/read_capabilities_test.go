package appserver

import (
	"context"
	"testing"
)

// session.keys/session.branches have no verified upstream RPC. Their Desktop
// wire shapes stay only as unverified decode fixtures; there is intentionally no
// client helper or fake-server method coverage for them.
func TestSessionMethods(t *testing.T) {
	ctx := context.Background()
	server, client := newFakeServer(t, fakeConfig{
		Hello: fakeHello("session.methods"),
		MethodsRaw: rawJSON(t, map[string]any{
			"methods": []any{map[string]any{"name": "shell.run", "mutating": true}},
		}),
	})
	result, err := SessionMethods(ctx, client, MethodsParams{SessionID: "s1"}, []string{"session.methods"})
	if err != nil || len(result.Methods) != 1 {
		t.Fatalf("SessionMethods = %#v, %v", result, err)
	}
	if calls := server.recordedCalls("session.methods"); len(calls) != 1 {
		t.Fatalf("session.methods calls = %d, want 1", len(calls))
	}

	methods, err := DecodeMethodsResult(rawJSON(t, map[string]any{
		"methods": []any{
			map[string]any{"name": "shell.list", "mutating": false},
			map[string]any{"name": "shell.run", "mutating": true},
		},
	}))
	if err != nil {
		t.Fatalf("DecodeMethodsResult: %v", err)
	}
	if len(methods.Methods) != 2 || methods.Methods[1].Name != "shell.run" || !methods.Methods[1].Mutating {
		t.Fatalf("methods = %#v", methods.Methods)
	}
}
