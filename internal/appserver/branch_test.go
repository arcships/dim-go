package appserver

import (
	"encoding/json"
	"errors"
	"testing"
)

func validBranchCreateParams() BranchCreateParams {
	return BranchCreateParams{
		SessionID:      "s1",
		ParentBranchID: "main",
		At:             int64Ptr(4),
		Prompt:         "side chat",
	}
}

func validBranchEndParams() BranchEndParams {
	return BranchEndParams{SessionID: "s1", BranchID: "b1"}
}

func branchCreateSuccess(t *testing.T) fakeMutation {
	t.Helper()
	return fakeMutation{Result: rawJSON(t, map[string]any{
		"branch": map[string]any{
			"branchId":       "b1",
			"parentBranchId": "main",
			"branchPoint":    4,
			"createEvent":    6,
		},
	})}
}

func TestBranchCreateParamsValidate(t *testing.T) {
	valid := []BranchCreateParams{
		validBranchCreateParams(),
		{SessionID: "s1", ParentBranchID: "main", At: int64Ptr(0), Prompt: "root cut"},
	}
	for _, params := range valid {
		if err := params.Validate(); err != nil {
			t.Errorf("valid params rejected: %+v: %v", params, err)
		}
	}

	invalid := []BranchCreateParams{
		{ParentBranchID: "main", At: int64Ptr(4), Prompt: "p"},
		{SessionID: "s1", At: int64Ptr(4), Prompt: "p"},
		{SessionID: "s1", ParentBranchID: "main", Prompt: "p"},
		{SessionID: "s1", ParentBranchID: "main", At: int64Ptr(-1), Prompt: "p"},
		{SessionID: "s1", ParentBranchID: "main", At: int64Ptr(4)},
	}
	for _, params := range invalid {
		err := params.Validate()
		if !errors.Is(err, ErrProtocol) {
			t.Errorf("invalid params %+v => %v, want ErrProtocol", params, err)
		}
		var protocolErr *ProtocolError
		if !errors.As(err, &protocolErr) || protocolErr.Field == "" {
			t.Errorf("invalid params %+v => %v, want a named ProtocolError field", params, err)
		}
	}
}

func TestDecodeBranchCreateResult(t *testing.T) {
	valid := []struct {
		name string
		raw  string
		want BranchCreateInfo
	}{
		{
			name: "full branch",
			raw:  `{"branch":{"branchId":"b1","parentBranchId":"main","branchPoint":4,"createEvent":6}}`,
			want: BranchCreateInfo{BranchID: "b1", ParentBranchID: "main", BranchPoint: int64Ptr(4), CreateEvent: int64Ptr(6)},
		},
		{
			name: "branchId is the only required member",
			raw:  `{"branch":{"branchId":"b1"}}`,
			want: BranchCreateInfo{BranchID: "b1"},
		},
		{
			name: "unknown members are ignored",
			raw:  `{"branch":{"branchId":"b1","future":true},"extra":1}`,
			want: BranchCreateInfo{BranchID: "b1"},
		},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecodeBranchCreateResult(json.RawMessage(tc.raw))
			if err != nil {
				t.Fatalf("DecodeBranchCreateResult: %v", err)
			}
			if result.Branch == nil {
				t.Fatal("branch is nil")
			}
			got := *result.Branch
			if got.BranchID != tc.want.BranchID || got.ParentBranchID != tc.want.ParentBranchID {
				t.Fatalf("branch = %#v, want %#v", got, tc.want)
			}
			if !equalInt64Ptr(got.BranchPoint, tc.want.BranchPoint) || !equalInt64Ptr(got.CreateEvent, tc.want.CreateEvent) {
				t.Fatalf("branch = %#v, want %#v", got, tc.want)
			}
		})
	}

	invalid := []struct {
		name string
		raw  string
	}{
		{name: "null result", raw: `null`},
		{name: "not an object", raw: `[]`},
		{name: "missing branch", raw: `{}`},
		{name: "null branch", raw: `{"branch":null}`},
		{name: "branch is not an object", raw: `{"branch":"b1"}`},
		{name: "missing branchId", raw: `{"branch":{"parentBranchId":"main"}}`},
		{name: "empty branchId", raw: `{"branch":{"branchId":""}}`},
		{name: "branchId is not a string", raw: `{"branch":{"branchId":7}}`},
		{name: "negative branchPoint", raw: `{"branch":{"branchId":"b1","branchPoint":-1}}`},
		{name: "negative createEvent", raw: `{"branch":{"branchId":"b1","createEvent":-2}}`},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeBranchCreateResult(json.RawMessage(tc.raw))
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

func TestBranchCreateHelper(t *testing.T) {
	t.Run("success sends the exact request and decodes the branch", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionBranchCreate): {branchCreateSuccess(t)},
			},
		})
		result, err := BranchCreate(testContext(t), client, validBranchCreateParams())
		if err != nil {
			t.Fatalf("BranchCreate: %v", err)
		}
		if result.Branch == nil || result.Branch.BranchID != "b1" ||
			result.Branch.ParentBranchID != "main" ||
			!equalInt64Ptr(result.Branch.BranchPoint, int64Ptr(4)) ||
			!equalInt64Ptr(result.Branch.CreateEvent, int64Ptr(6)) {
			t.Fatalf("result = %#v", result)
		}
		calls := server.recordedCalls(string(MutationSessionBranchCreate))
		if len(calls) != 1 {
			t.Fatalf("branch_create calls = %d, want 1", len(calls))
		}
		params := rawParams(t, calls[0])
		if len(params) != 4 {
			t.Fatalf("params = %s, want sessionId, parentBranchId, at and prompt", calls[0].Params)
		}
		for key, want := range map[string]string{
			"sessionId":      `"s1"`,
			"parentBranchId": `"main"`,
			"at":             `4`,
			"prompt":         `"side chat"`,
		} {
			if got := string(params[key]); got != want {
				t.Fatalf("param %s = %s, want %s", key, got, want)
			}
		}
	})

	t.Run("invalid params fail before the call", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionBranchCreate): {branchCreateSuccess(t)},
			},
		})
		params := validBranchCreateParams()
		params.At = nil
		if _, err := BranchCreate(testContext(t), client, params); !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if calls := server.recordedCalls(string(MutationSessionBranchCreate)); len(calls) != 0 {
			t.Fatalf("branch_create was called %d times with invalid params", len(calls))
		}
	})

	t.Run("missing branchId is a protocol error and an unknown result", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionBranchCreate): {{Result: json.RawMessage(`{"branch":{}}`)}},
			},
		})
		_, err := BranchCreate(testContext(t), client, validBranchCreateParams())
		if !errors.Is(err, ErrUnknownResult) || !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrUnknownResult + ErrProtocol", err)
		}
		var unknown *UnknownResultError
		if !errors.As(err, &unknown) || unknown.ReadBack != "session.branches" {
			t.Fatalf("error = %#v, want session.branches read-back", err)
		}
		if calls := server.recordedCalls(string(MutationSessionBranchCreate)); len(calls) != 1 {
			t.Fatalf("branch_create calls = %d, want no resend", len(calls))
		}
	})

	t.Run("-32602 is a definitive bad request", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionBranchCreate): {{Fault: fakeBadParams()}},
			},
		})
		_, err := BranchCreate(testContext(t), client, validBranchCreateParams())
		mutationErr := requireMutationError(t, err, MutationSessionBranchCreate, "")
		if mutationErr.Mapped.Class != ErrorClassBadRequest || mutationErr.Mapped.Retryable {
			t.Fatalf("mapped = %#v, want non-retryable bad request", mutationErr.Mapped)
		}
		if errors.Is(err, ErrUnknownResult) {
			t.Fatalf("error = %v, want a definitive rejection", err)
		}
		if calls := server.recordedCalls(string(MutationSessionBranchCreate)); len(calls) != 1 {
			t.Fatalf("branch_create calls = %d, want no retry", len(calls))
		}
	})

	t.Run("unknown_method is visible-disabled WIP and not retried", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionBranchCreate): {{Fault: fakeUnknownMethod()}},
			},
		})
		_, err := BranchCreate(testContext(t), client, validBranchCreateParams())
		mutationErr := requireMutationError(t, err, MutationSessionBranchCreate, AppErrorUnknownMethod)
		if !mutationErr.Unsupported() || !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("error = %v, want a capability error", err)
		}
		if mutationErr.Mapped.Class != ErrorClassCapability || mutationErr.Mapped.Retryable {
			t.Fatalf("mapped = %#v, want non-retryable capability", mutationErr.Mapped)
		}
		if calls := server.recordedCalls(string(MutationSessionBranchCreate)); len(calls) != 1 {
			t.Fatalf("branch_create calls = %d, want no retry", len(calls))
		}
	})

	t.Run("disconnect is an unknown result", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello:         fakeHello(),
			CloseOnMethod: map[string]bool{string(MutationSessionBranchCreate): true},
		})
		_, err := BranchCreate(testContext(t), client, validBranchCreateParams())
		requireUnknownResult(t, err, MutationSessionBranchCreate, "connection_closed", "session.branches")
		if calls := server.recordedCalls(string(MutationSessionBranchCreate)); len(calls) != 1 {
			t.Fatalf("branch_create calls = %d, want no resend", len(calls))
		}
	})
}

func TestBranchEndParamsValidate(t *testing.T) {
	if err := validBranchEndParams().Validate(); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}
	invalid := []BranchEndParams{
		{BranchID: "b1"},
		{SessionID: "s1"},
	}
	for _, params := range invalid {
		if err := params.Validate(); !errors.Is(err, ErrProtocol) {
			t.Errorf("invalid params %+v => %v, want ErrProtocol", params, err)
		}
	}
}

func TestBranchEndHelper(t *testing.T) {
	t.Run("success sends only sessionId and branchId", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionBranchEnd): {{Result: json.RawMessage(`{"ignored":true}`)}},
			},
		})
		outcome, err := BranchEnd(testContext(t), client, validBranchEndParams())
		if err != nil || outcome != BranchEndOutcomeEnded {
			t.Fatalf("BranchEnd = %q, %v; want ended, nil", outcome, err)
		}
		calls := server.recordedCalls(string(MutationSessionBranchEnd))
		if len(calls) != 1 {
			t.Fatalf("branch_end calls = %d, want 1", len(calls))
		}
		params := rawParams(t, calls[0])
		if len(params) != 2 || string(params["sessionId"]) != `"s1"` || string(params["branchId"]) != `"b1"` {
			t.Fatalf("params = %s, want only sessionId and branchId", calls[0].Params)
		}
	})

	t.Run("invalid params fail before the call", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionBranchEnd): {{Result: json.RawMessage(`{}`)}},
			},
		})
		if _, err := BranchEnd(testContext(t), client, BranchEndParams{SessionID: "s1"}); !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if calls := server.recordedCalls(string(MutationSessionBranchEnd)); len(calls) != 0 {
			t.Fatalf("branch_end was called %d times with invalid params", len(calls))
		}
	})

	t.Run("branch_ended is a typed outcome and is not retried", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionBranchEnd): {
					{Fault: &fakeRPCError{Code: -32000, Message: "ended", Data: appErrorData(AppErrorBranchEnded)}},
				},
			},
		})
		outcome, err := BranchEnd(testContext(t), client, validBranchEndParams())
		if err != nil || outcome != BranchEndOutcomeAlreadyEnded {
			t.Fatalf("BranchEnd = %q, %v; want already_ended, nil", outcome, err)
		}
		if calls := server.recordedCalls(string(MutationSessionBranchEnd)); len(calls) != 1 {
			t.Fatalf("branch_end calls = %d, want no retry", len(calls))
		}
	})

	t.Run("unknown_method is visible-disabled WIP and not retried", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionBranchEnd): {{Fault: fakeUnknownMethod()}},
			},
		})
		_, err := BranchEnd(testContext(t), client, validBranchEndParams())
		mutationErr := requireMutationError(t, err, MutationSessionBranchEnd, AppErrorUnknownMethod)
		if !mutationErr.Unsupported() || !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("error = %v, want a capability error", err)
		}
		if calls := server.recordedCalls(string(MutationSessionBranchEnd)); len(calls) != 1 {
			t.Fatalf("branch_end calls = %d, want no retry", len(calls))
		}
	})

	t.Run("-32602 wins over branch_ended and stays a bad request", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionBranchEnd): {
					{Fault: &fakeRPCError{Code: -32602, Message: "invalid params", Data: appErrorData(AppErrorBranchEnded)}},
				},
			},
		})
		_, err := BranchEnd(testContext(t), client, validBranchEndParams())
		mutationErr := requireMutationError(t, err, MutationSessionBranchEnd, AppErrorBranchEnded)
		if mutationErr.Mapped.Class != ErrorClassBadRequest || mutationErr.Mapped.Retryable {
			t.Fatalf("mapped = %#v, want non-retryable bad request", mutationErr.Mapped)
		}
		if calls := server.recordedCalls(string(MutationSessionBranchEnd)); len(calls) != 1 {
			t.Fatalf("branch_end calls = %d, want no retry", len(calls))
		}
	})

	t.Run("disconnect is an unknown result", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello:         fakeHello(),
			CloseOnMethod: map[string]bool{string(MutationSessionBranchEnd): true},
		})
		_, err := BranchEnd(testContext(t), client, validBranchEndParams())
		requireUnknownResult(t, err, MutationSessionBranchEnd, "connection_closed", "session.branches")
		if calls := server.recordedCalls(string(MutationSessionBranchEnd)); len(calls) != 1 {
			t.Fatalf("branch_end calls = %d, want no resend", len(calls))
		}
	})
}

func equalInt64Ptr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
