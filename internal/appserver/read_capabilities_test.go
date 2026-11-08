package appserver

import (
	"context"
	"testing"
)

func TestSessionBranches(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, fakeConfig{
		Hello:   fakeHello("session.branches"),
		KeysRaw: rawJSON(t, map[string]any{}),
		BranchesRaw: rawJSON(t, map[string]any{
			"branches": []any{map[string]any{"branchId": "main", "state": "open", "head": 4}},
		}),
	})
	result, err := SessionBranches(ctx, client, BranchesParams{SessionID: "s1"}, []string{"session.branches"})
	if err != nil || len(result.Branches) != 1 {
		t.Fatalf("SessionBranches = %#v, %v", result, err)
	}
	if calls := server.recordedCalls("session.branches"); len(calls) != 1 {
		t.Fatalf("session.branches calls = %d, want 1", len(calls))
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
