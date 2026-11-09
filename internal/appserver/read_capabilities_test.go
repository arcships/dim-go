package appserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// session.keys/session.branches are public protocol-v3 reads: Desktop's
// app-server-protocol wire.ts declares the DTOs and session-reader.ts issues the
// requests on the session stream behind hello.features. These tests pin the
// helper request shapes and the decode/feature-gate behavior.
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

// session.methods is a built-in session-level method (§16), not one of the
// connection-level hello.features. An absent feature entry must not turn it into
// a capability error.
func TestSessionMethodsIsNotFeatureGated(t *testing.T) {
	ctx := testContext(t)
	server, client := newFakeServer(t, fakeConfig{
		Hello:      fakeHello(),
		MethodsRaw: rawJSON(t, map[string]any{"methods": []any{}}),
	})
	result, err := SessionMethods(ctx, client, MethodsParams{SessionID: "s1"}, nil)
	if err != nil {
		t.Fatalf("SessionMethods without the feature: %v", err)
	}
	if result.Methods == nil || len(result.Methods) != 0 {
		t.Fatalf("methods = %#v", result.Methods)
	}
	if calls := server.recordedCalls("session.methods"); len(calls) != 1 {
		t.Fatalf("session.methods calls = %d, want 1", len(calls))
	}
}

func TestSessionBranches(t *testing.T) {
	branchesRaw := func(t *testing.T) json.RawMessage {
		t.Helper()
		return rawJSON(t, map[string]any{
			"branches": []any{
				map[string]any{"branchId": "main", "state": "open", "head": 4},
				map[string]any{
					"branchId":       "child",
					"parentBranchId": "main",
					"branchPoint":    2,
					"createEvent":    3,
					"source":         map[string]any{"kind": "tool", "turn_id": "t1", "request_id": "r1", "call_id": "c1"},
					"state":          "ended",
					"head":           5,
					"endEvent":       5,
				},
			},
		})
	}

	t.Run("directory is decoded and the request is exact", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:       fakeHello("session.branches"),
			BranchesRaw: branchesRaw(t),
		})
		result, err := SessionBranches(ctx, client, BranchesParams{SessionID: "s1"}, []string{"session.branches"})
		if err != nil {
			t.Fatalf("SessionBranches: %v", err)
		}
		if len(result.Branches) != 2 {
			t.Fatalf("branches = %#v", result.Branches)
		}
		main, child := result.Branches[0], result.Branches[1]
		if main.BranchID != "main" || main.State != "open" || main.Head != 4 || main.EndEvent != nil {
			t.Fatalf("main = %#v", main)
		}
		if child.ParentBranchID != "main" || *child.BranchPoint != 2 || *child.CreateEvent != 3 || *child.EndEvent != 5 {
			t.Fatalf("child = %#v", child)
		}
		if child.Source == nil || child.Source.Kind != "tool" || child.Source.CallID != "c1" {
			t.Fatalf("source = %#v", child.Source)
		}
		calls := server.recordedCalls("session.branches")
		if len(calls) != 1 {
			t.Fatalf("session.branches calls = %d, want 1", len(calls))
		}
		params := rawParams(t, calls[0])
		if len(params) != 1 || string(params["sessionId"]) != `"s1"` {
			t.Fatalf("params = %s, want exactly sessionId", calls[0].Params)
		}
	})

	t.Run("missing feature is a capability error and sends nothing", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:       fakeHello("session.keys"),
			BranchesRaw: branchesRaw(t),
		})
		result, err := SessionBranches(ctx, client, BranchesParams{SessionID: "s1"}, []string{"session.keys"})
		if !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("error = %v, want ErrCapabilityUnavailable", err)
		}
		var capability *CapabilityError
		if !errors.As(err, &capability) || capability.Feature != "session.branches" {
			t.Fatalf("capability error = %#v", err)
		}
		if result.Branches != nil {
			t.Fatalf("capability error returned a result: %#v", result)
		}
		if calls := server.recordedCalls("session.branches"); len(calls) != 0 {
			t.Fatalf("session.branches was called %d times without the feature", len(calls))
		}
	})

	t.Run("invalid params never reach the daemon", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:       fakeHello("session.branches"),
			BranchesRaw: branchesRaw(t),
		})
		if _, err := SessionBranches(ctx, client, BranchesParams{}, []string{"session.branches"}); !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if calls := server.recordedCalls("session.branches"); len(calls) != 0 {
			t.Fatalf("session.branches was called %d times with invalid params", len(calls))
		}
	})

	t.Run("malformed result is a protocol error", func(t *testing.T) {
		ctx := testContext(t)
		_, client := newFakeServer(t, fakeConfig{
			Hello:       fakeHello("session.branches"),
			BranchesRaw: rawJSON(t, map[string]any{"branches": []any{map[string]any{"branchId": "main", "state": "lost", "head": 0}}}),
		})
		result, err := SessionBranches(ctx, client, BranchesParams{SessionID: "s1"}, []string{"session.branches"})
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if result.Branches != nil {
			t.Fatalf("protocol error returned a result: %#v", result)
		}
	})

	t.Run("daemon error surfaces once without retry", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:         fakeHello("session.branches"),
			BranchesFault: fakeUnknownMethod(),
		})
		_, err := SessionBranches(ctx, client, BranchesParams{SessionID: "s1"}, []string{"session.branches"})
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32601 {
			t.Fatalf("error = %v, want RPCError -32601", err)
		}
		if calls := server.recordedCalls("session.branches"); len(calls) != 1 {
			t.Fatalf("session.branches calls = %d, want exactly 1", len(calls))
		}
	})
}

func TestDecodeBranchesResult(t *testing.T) {
	valid := []struct {
		name string
		raw  string
	}{
		{name: "empty directory", raw: `{"branches":[]}`},
		{name: "host source", raw: `{"branches":[{"branchId":"main","state":"ending","head":2,"source":{"kind":"host","operation_id":"op-1"}}]}`},
		{name: "tool source", raw: `{"branches":[{"branchId":"child","state":"ended","head":5,"endEvent":5,"source":{"kind":"tool","turn_id":"t","request_id":"r","call_id":"c"}}]}`},
		{name: "unknown members are ignored", raw: `{"branches":[{"branchId":"main","state":"open","head":0,"future":true}],"extra":1}`},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodeBranchesResult(json.RawMessage(tc.raw)); err != nil {
				t.Fatalf("DecodeBranchesResult: %v", err)
			}
		})
	}

	invalid := []struct {
		name string
		raw  string
	}{
		{name: "not an object", raw: `[]`},
		{name: "missing branches", raw: `{}`},
		{name: "branches is null", raw: `{"branches":null}`},
		{name: "branches is not an array", raw: `{"branches":{}}`},
		{name: "entry is not an object", raw: `{"branches":["main"]}`},
		{name: "missing branchId", raw: `{"branches":[{"state":"open","head":0}]}`},
		{name: "unknown state", raw: `{"branches":[{"branchId":"main","state":"lost","head":0}]}`},
		{name: "negative head", raw: `{"branches":[{"branchId":"main","state":"open","head":-1}]}`},
		{name: "negative branchPoint", raw: `{"branches":[{"branchId":"main","state":"open","head":0,"branchPoint":-1}]}`},
		{name: "negative createEvent", raw: `{"branches":[{"branchId":"main","state":"open","head":0,"createEvent":-1}]}`},
		{name: "negative endEvent", raw: `{"branches":[{"branchId":"main","state":"open","head":0,"endEvent":-1}]}`},
		{name: "unknown source kind", raw: `{"branches":[{"branchId":"main","state":"open","head":0,"source":{"kind":"plugin"}}]}`},
		{name: "tool source without call_id", raw: `{"branches":[{"branchId":"main","state":"open","head":0,"source":{"kind":"tool","turn_id":"t","request_id":"r"}}]}`},
		{name: "host source without operation_id", raw: `{"branches":[{"branchId":"main","state":"open","head":0,"source":{"kind":"host"}}]}`},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeBranchesResult(json.RawMessage(tc.raw))
			if !errors.Is(err, ErrProtocol) {
				t.Fatalf("error = %v, want ErrProtocol", err)
			}
			var protocolErr *ProtocolError
			if !errors.As(err, &protocolErr) || protocolErr.Field == "" {
				t.Fatalf("error = %v, want a named ProtocolError field", err)
			}
		})
	}
}
