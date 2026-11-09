package appserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func acceptedLookup(t *testing.T) json.RawMessage {
	t.Helper()
	return rawJSON(t, map[string]any{
		"status": InputLookupAccepted,
		"event":  fakeEvent(4),
	})
}

func TestInputLookupParamsValidate(t *testing.T) {
	valid := []InputLookupParams{
		{SessionID: "s1", SubmissionID: "sub-1"},
		{SessionID: "s1", BranchID: "child", SubmissionID: "sub-1"},
	}
	for _, params := range valid {
		if err := params.Validate(); err != nil {
			t.Errorf("valid params rejected: %+v: %v", params, err)
		}
	}
	invalid := []InputLookupParams{
		{SubmissionID: "sub-1"},
		{SessionID: "s1"},
		{SessionID: "s1", BranchID: "child"},
	}
	for _, params := range invalid {
		if err := params.Validate(); !errors.Is(err, ErrProtocol) {
			t.Errorf("invalid params %+v => %v, want ErrProtocol", params, err)
		}
	}
}

func TestInputLookupParamsWireShape(t *testing.T) {
	// branchId is optional upstream, so omitting it must omit the member
	// instead of sending "" and inventing a branch.
	data, err := json.Marshal(InputLookupParams{SessionID: "s1", SubmissionID: "sub-1"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"sessionId":"s1","submissionId":"sub-1"}` {
		t.Fatalf("params = %s", data)
	}

	data, err = json.Marshal(InputLookupParams{SessionID: "s1", BranchID: "child", SubmissionID: "sub-1"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"sessionId":"s1","branchId":"child","submissionId":"sub-1"}` {
		t.Fatalf("params = %s", data)
	}
}

func TestDecodeInputLookupResult(t *testing.T) {
	t.Run("accepted carries the durable event", func(t *testing.T) {
		result, err := DecodeInputLookupResult(acceptedLookup(t))
		if err != nil {
			t.Fatalf("DecodeInputLookupResult: %v", err)
		}
		if result.Status != InputLookupAccepted || !result.Accepted() {
			t.Fatalf("status = %q", result.Status)
		}
		if result.Event == nil || result.Event.Sequence != 4 || result.Event.ID != "event-4" {
			t.Fatalf("event = %#v", result.Event)
		}
		if result.Event.Type != "message" || result.Event.Name != "user_input" || result.Event.CreateAt != fakeCreatedAt {
			t.Fatalf("event = %#v", result.Event)
		}
	})

	t.Run("unknown carries no event", func(t *testing.T) {
		result, err := DecodeInputLookupResult(json.RawMessage(`{"status":"unknown"}`))
		if err != nil {
			t.Fatalf("DecodeInputLookupResult: %v", err)
		}
		if result.Accepted() || result.Event != nil {
			t.Fatalf("result = %#v", result)
		}
	})

	t.Run("null event decodes as absent", func(t *testing.T) {
		result, err := DecodeInputLookupResult(json.RawMessage(`{"status":"unknown","event":null}`))
		if err != nil {
			t.Fatalf("DecodeInputLookupResult: %v", err)
		}
		if result.Event != nil {
			t.Fatalf("event = %#v", result.Event)
		}
	})

	t.Run("unknown members are ignored", func(t *testing.T) {
		result, err := DecodeInputLookupResult(json.RawMessage(`{"status":"unknown","future":true}`))
		if err != nil || result.Status != InputLookupUnknown {
			t.Fatalf("result = %#v, %v", result, err)
		}
	})

	invalid := []struct {
		name string
		raw  string
	}{
		{name: "not an object", raw: `[]`},
		{name: "missing status", raw: `{}`},
		{name: "status is not a string", raw: `{"status":5}`},
		{name: "status is unknown casing", raw: `{"status":"Accepted"}`},
		{name: "status is not part of the union", raw: `{"status":"rejected","event":null}`},
		{name: "accepted without event", raw: `{"status":"accepted"}`},
		{name: "accepted with null event", raw: `{"status":"accepted","event":null}`},
		{name: "accepted with invalid event", raw: `{"status":"accepted","event":{"id":"","sequence":4,"create_at":1,"type":"message","name":"user_input"}}`},
		{name: "accepted with zero sequence", raw: `{"status":"accepted","event":{"id":"e1","sequence":0,"create_at":1,"type":"message","name":"user_input"}}`},
		{name: "accepted with missing create_at", raw: `{"status":"accepted","event":{"id":"e1","sequence":4,"type":"message","name":"user_input"}}`},
		{name: "event is not an object", raw: `{"status":"accepted","event":"event-4"}`},
		{name: "unknown with event", raw: `{"status":"unknown","event":{"id":"e1","sequence":4,"create_at":1,"type":"message","name":"user_input"}}`},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeInputLookupResult(json.RawMessage(tc.raw))
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

func TestInputLookupOutcomeHelper(t *testing.T) {
	t.Run("accepted without branchId", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:          fakeHello("session.input_lookup"),
			InputLookupRaw: acceptedLookup(t),
		})
		result, err := InputLookupOutcome(ctx, client, InputLookupParams{
			SessionID:    "s1",
			SubmissionID: "sub-1",
		})
		if err != nil {
			t.Fatalf("InputLookupOutcome: %v", err)
		}
		if !result.Accepted() || result.Event == nil || result.Event.Sequence != 4 {
			t.Fatalf("result = %#v", result)
		}
		calls := server.recordedCalls("session.input_lookup")
		if len(calls) != 1 {
			t.Fatalf("session.input_lookup calls = %d, want 1", len(calls))
		}
		params := rawParams(t, calls[0])
		if len(params) != 2 {
			t.Fatalf("params = %s, want exactly sessionId and submissionId", calls[0].Params)
		}
		if string(params["sessionId"]) != `"s1"` || string(params["submissionId"]) != `"sub-1"` {
			t.Fatalf("params = %s", calls[0].Params)
		}
	})

	t.Run("explicit branchId is forwarded", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:          fakeHello("session.input_lookup"),
			InputLookupRaw: json.RawMessage(`{"status":"unknown"}`),
		})
		result, err := InputLookupOutcome(ctx, client, InputLookupParams{
			SessionID:    "s1",
			BranchID:     "child",
			SubmissionID: "sub-2",
		})
		if err != nil {
			t.Fatalf("InputLookupOutcome: %v", err)
		}
		if result.Accepted() || result.Event != nil {
			t.Fatalf("result = %#v", result)
		}
		calls := server.recordedCalls("session.input_lookup")
		if len(calls) != 1 {
			t.Fatalf("session.input_lookup calls = %d, want 1", len(calls))
		}
		params := rawParams(t, calls[0])
		if string(params["branchId"]) != `"child"` {
			t.Fatalf("params = %s", calls[0].Params)
		}
	})

	t.Run("invalid params never reach the daemon", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:          fakeHello("session.input_lookup"),
			InputLookupRaw: acceptedLookup(t),
		})
		if _, err := InputLookupOutcome(ctx, client, InputLookupParams{SessionID: "s1"}); !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if calls := server.recordedCalls("session.input_lookup"); len(calls) != 0 {
			t.Fatalf("session.input_lookup was called %d times with invalid params", len(calls))
		}
	})

	t.Run("malformed result is a protocol error", func(t *testing.T) {
		ctx := testContext(t)
		_, client := newFakeServer(t, fakeConfig{
			Hello:          fakeHello("session.input_lookup"),
			InputLookupRaw: json.RawMessage(`{"status":"accepted"}`),
		})
		result, err := InputLookupOutcome(ctx, client, InputLookupParams{
			SessionID:    "s1",
			SubmissionID: "sub-1",
		})
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if result.Status != "" || result.Event != nil {
			t.Fatalf("protocol error returned a result: %#v", result)
		}
	})

	t.Run("daemon error surfaces once without retry", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:            fakeHello("session.input_lookup"),
			InputLookupFault: fakeUnknownMethod(),
		})
		_, err := InputLookupOutcome(ctx, client, InputLookupParams{
			SessionID:    "s1",
			SubmissionID: "sub-1",
		})
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32601 {
			t.Fatalf("error = %v, want RPCError -32601", err)
		}
		if calls := server.recordedCalls("session.input_lookup"); len(calls) != 1 {
			t.Fatalf("session.input_lookup calls = %d, want exactly 1", len(calls))
		}
	})

	t.Run("nil client is a protocol error", func(t *testing.T) {
		if _, err := InputLookupOutcome(context.Background(), nil, InputLookupParams{
			SessionID:    "s1",
			SubmissionID: "sub-1",
		}); !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
	})
}

func TestInputLookupRawStillReturnsVerbatim(t *testing.T) {
	ctx := testContext(t)
	// A field this build does not model must survive the compatibility path.
	verbatim := json.RawMessage(`{"status":"accepted","event":{"id":"e1","sequence":4,"create_at":1700000000000,"type":"message","name":"user_input"},"future":{"x":1}}`)
	server, client := newFakeServer(t, fakeConfig{
		Hello:          fakeHello("session.input_lookup"),
		InputLookupRaw: verbatim,
	})
	raw, err := InputLookup(ctx, client, InputLookupParams{SessionID: "s1", SubmissionID: "sub-1"})
	if err != nil {
		t.Fatalf("InputLookup: %v", err)
	}
	if string(raw) != string(verbatim) {
		t.Fatalf("raw = %s, want %s", raw, verbatim)
	}
	if calls := server.recordedCalls("session.input_lookup"); len(calls) != 1 {
		t.Fatalf("session.input_lookup calls = %d, want 1", len(calls))
	}
}
