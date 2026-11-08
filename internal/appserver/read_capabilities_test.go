package appserver

import (
	"context"
	"errors"
	"testing"
)

func TestSessionBranches(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, fakeConfig{
		Hello:   fakeHello("session.branches"),
		KeysRaw: rawJSON(t, map[string]any{}),
	})
	// The fake server dispatch does not yet script branches; validate the gate
	// and decoder independently below, while keeping this test focused on the
	// no-request capability contract.
	result, err := SessionBranches(ctx, client, BranchesParams{SessionID: "s1"}, []string{"session.history"})
	if !errors.Is(err, ErrCapabilityUnavailable) {
		t.Fatalf("error = %v, want capability error", err)
	}
	if result.Branches != nil {
		t.Fatalf("result = %#v, want zero result", result)
	}
	if calls := server.recordedCalls("session.branches"); len(calls) != 0 {
		t.Fatalf("session.branches calls = %d, want 0", len(calls))
	}

	branches, err := DecodeBranchesResult(rawJSON(t, map[string]any{
		"branches": []any{
			map[string]any{"branchId": "main", "state": "open", "head": 4},
			map[string]any{"branchId": "child", "parentBranchId": "main", "branchPoint": 2, "createEvent": 3, "state": "ended", "head": 5, "endEvent": 5},
		},
	}))
	if err != nil {
		t.Fatalf("DecodeBranchesResult: %v", err)
	}
	if len(branches.Branches) != 2 || branches.Branches[1].ParentBranchID != "main" {
		t.Fatalf("branches = %#v", branches.Branches)
	}
}

func TestSessionMethods(t *testing.T) {
	ctx := context.Background()
	server, client := newFakeServer(t, fakeConfig{Hello: fakeHello("session.methods")})
	result, err := SessionMethods(ctx, client, MethodsParams{SessionID: "s1"}, []string{"session.history"})
	if !errors.Is(err, ErrCapabilityUnavailable) {
		t.Fatalf("error = %v, want capability error", err)
	}
	if result.Methods != nil {
		t.Fatalf("result = %#v, want zero result", result)
	}
	if calls := server.recordedCalls("session.methods"); len(calls) != 0 {
		t.Fatalf("session.methods calls = %d, want 0", len(calls))
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
