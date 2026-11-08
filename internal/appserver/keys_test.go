package appserver

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDecodeKeysResult(t *testing.T) {
	valid := []struct {
		name     string
		raw      string
		through  int64
		entryLen int
	}{
		{
			name:    "empty branch",
			raw:     `{"through":0,"entries":[]}`,
			through: 0,
		},
		{
			name: "entries keep wire order",
			raw: `{"through":5,"entries":[` +
				`{"key":"goal:state","sequence":4,"inputId":"i4","text":"<dim-goal kind=\"state\"/>"},` +
				`{"key":"branch:parent","sequence":2,"inputId":"i2","text":""}]}`,
			through:  5,
			entryLen: 2,
		},
		{
			name:     "unknown members are ignored",
			raw:      `{"through":7,"entries":[],"extra":{"unknown":true}}`,
			through:  7,
			entryLen: 0,
		},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecodeKeysResult(json.RawMessage(tc.raw))
			if err != nil {
				t.Fatalf("DecodeKeysResult: %v", err)
			}
			if result.Through == nil || *result.Through != tc.through {
				t.Fatalf("through = %v, want %d", result.Through, tc.through)
			}
			if result.Entries == nil {
				t.Fatal("entries is nil, want a decoded array")
			}
			if len(result.Entries) != tc.entryLen {
				t.Fatalf("entries = %#v, want %d", result.Entries, tc.entryLen)
			}
		})
	}

	invalid := []struct {
		name string
		raw  string
	}{
		{name: "not an object", raw: `[]`},
		{name: "missing through", raw: `{"entries":[]}`},
		{name: "through is not an integer", raw: `{"through":"5","entries":[]}`},
		{name: "negative through", raw: `{"through":-1,"entries":[]}`},
		{name: "missing entries", raw: `{"through":5}`},
		{name: "entries is null", raw: `{"through":5,"entries":null}`},
		{name: "entries is not an array", raw: `{"through":5,"entries":{}}`},
		{name: "entry is not an object", raw: `{"through":5,"entries":["k"]}`},
		{name: "missing key", raw: `{"through":5,"entries":[{"sequence":1,"inputId":"i","text":"t"}]}`},
		{name: "zero sequence", raw: `{"through":5,"entries":[{"key":"k","inputId":"i","text":"t"}]}`},
		{name: "negative sequence", raw: `{"through":5,"entries":[{"key":"k","sequence":-2,"inputId":"i","text":"t"}]}`},
		{name: "missing inputId", raw: `{"through":5,"entries":[{"key":"k","sequence":1,"text":"t"}]}`},
		{name: "text is not a string", raw: `{"through":5,"entries":[{"key":"k","sequence":1,"inputId":"i","text":9}]}`},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeKeysResult(json.RawMessage(tc.raw))
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

func TestKeysParamsValidate(t *testing.T) {
	through := int64(5)
	valid := []KeysParams{
		{SessionID: "s1", Through: &through},
		{SessionID: "s1", BranchID: "main", Through: int64Ptr(0)},
	}
	for _, params := range valid {
		if err := params.Validate(); err != nil {
			t.Errorf("valid params rejected: %+v: %v", params, err)
		}
	}
	invalid := []KeysParams{
		{Through: &through},
		{SessionID: "s1"},
		{SessionID: "s1", Through: int64Ptr(-1)},
	}
	for _, params := range invalid {
		if err := params.Validate(); !errors.Is(err, ErrProtocol) {
			t.Errorf("invalid params %+v => %v, want ErrProtocol", params, err)
		}
	}
}

func TestSessionKeysHelper(t *testing.T) {
	t.Run("snapshot is decoded and the request is exact", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello("session.keys"),
			KeysRaw: rawJSON(t, map[string]any{
				"through": 5,
				"entries": []any{
					map[string]any{"key": "goal:state", "sequence": 4, "inputId": "i4", "text": "<dim-goal kind=\"state\"/>"},
					map[string]any{"key": "branch:parent", "sequence": 2, "inputId": "i2", "text": ""},
				},
			}),
		})
		result, err := SessionKeys(ctx, client, KeysParams{
			SessionID: "s1",
			BranchID:  "main",
			Through:   int64Ptr(5),
		}, []string{"session.keys"})
		if err != nil {
			t.Fatalf("SessionKeys: %v", err)
		}
		if result.Through == nil || *result.Through != 5 {
			t.Fatalf("through = %v", result.Through)
		}
		if len(result.Entries) != 2 {
			t.Fatalf("entries = %#v", result.Entries)
		}
		// Wire order is preserved; ordering policy belongs to the seed layer.
		first, second := result.Entries[0], result.Entries[1]
		if first.Key != "goal:state" || first.Sequence != 4 || first.InputID != "i4" || first.Text != `<dim-goal kind="state"/>` {
			t.Fatalf("first entry = %#v", first)
		}
		if second.Key != "branch:parent" || second.Sequence != 2 || second.InputID != "i2" || second.Text != "" {
			t.Fatalf("second entry = %#v", second)
		}

		calls := server.recordedCalls("session.keys")
		if len(calls) != 1 {
			t.Fatalf("session.keys calls = %d, want 1", len(calls))
		}
		params := rawParams(t, calls[0])
		if len(params) != 3 {
			t.Fatalf("params = %s, want exactly sessionId, branchId and through", calls[0].Params)
		}
		if string(params["sessionId"]) != `"s1"` || string(params["branchId"]) != `"main"` || string(params["through"]) != `5` {
			t.Fatalf("params = %s", calls[0].Params)
		}
	})

	t.Run("missing feature is a capability error and sends nothing", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:   fakeHello("session.history"),
			KeysRaw: rawJSON(t, map[string]any{"through": 5, "entries": []any{}}),
		})
		result, err := SessionKeys(ctx, client, KeysParams{
			SessionID: "s1",
			Through:   int64Ptr(5),
		}, []string{"session.history"})
		if !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("error = %v, want ErrCapabilityUnavailable", err)
		}
		var capability *CapabilityError
		if !errors.As(err, &capability) || capability.Feature != "session.keys" {
			t.Fatalf("capability error = %#v", err)
		}
		if result.Through != nil || result.Entries != nil {
			t.Fatalf("capability error returned a result: %#v", result)
		}
		if calls := server.recordedCalls("session.keys"); len(calls) != 0 {
			t.Fatalf("session.keys was called %d times without the feature", len(calls))
		}
	})

	t.Run("invalid params never reach the daemon", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:   fakeHello("session.keys"),
			KeysRaw: rawJSON(t, map[string]any{"through": 5, "entries": []any{}}),
		})
		if _, err := SessionKeys(ctx, client, KeysParams{}, []string{"session.keys"}); !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if calls := server.recordedCalls("session.keys"); len(calls) != 0 {
			t.Fatalf("session.keys was called %d times with invalid params", len(calls))
		}
	})

	t.Run("malformed result is a protocol error", func(t *testing.T) {
		ctx := testContext(t)
		_, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello("session.keys"),
			KeysRaw: json.RawMessage(`{"through":5,"entries":[` +
				`{"key":"","sequence":1,"inputId":"i","text":"t"}]}`),
		})
		result, err := SessionKeys(ctx, client, KeysParams{
			SessionID: "s1",
			Through:   int64Ptr(5),
		}, []string{"session.keys"})
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if result.Through != nil || result.Entries != nil {
			t.Fatalf("protocol error returned a result: %#v", result)
		}
	})

	t.Run("daemon error surfaces once without retry", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:     fakeHello("session.keys"),
			KeysFault: fakeUnknownMethod(),
		})
		_, err := SessionKeys(ctx, client, KeysParams{
			SessionID: "s1",
			Through:   int64Ptr(5),
		}, []string{"session.keys"})
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32601 {
			t.Fatalf("error = %v, want RPCError -32601", err)
		}
		if calls := server.recordedCalls("session.keys"); len(calls) != 1 {
			t.Fatalf("session.keys calls = %d, want exactly 1", len(calls))
		}
	})
}
