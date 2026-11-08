package appserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func validSubmitParams() SubmitParams {
	return SubmitParams{
		SessionID:    "s1",
		SubmissionID: "submission-1",
		BranchID:     "main",
		Text:         "hello",
	}
}

func validStopParams() StopParams {
	return StopParams{SessionID: "s1", BranchID: "main"}
}

func validApprovalParams() ApprovalRespondParams {
	return ApprovalRespondParams{SessionID: "s1", BranchID: "main", RequestID: "req-1", Decision: DecisionAllow}
}

func validDeleteParams() DeleteParams {
	return DeleteParams{SessionID: "s1"}
}

type mutationCase struct {
	method   MutationMethod
	readBack string
	call     func(context.Context, *Client) error
}

func mutationCases() []mutationCase {
	return []mutationCase{
		{
			method:   MutationSessionSubmit,
			readBack: "session.input_lookup",
			call: func(ctx context.Context, client *Client) error {
				_, err := Submit(ctx, client, validSubmitParams())
				return err
			},
		},
		{
			method:   MutationSessionStop,
			readBack: "session.state",
			call: func(ctx context.Context, client *Client) error {
				return Stop(ctx, client, validStopParams())
			},
		},
		{
			method:   MutationApprovalRespond,
			readBack: "session.attach",
			call: func(ctx context.Context, client *Client) error {
				_, err := ApprovalRespond(ctx, client, validApprovalParams())
				return err
			},
		},
		{
			method:   MutationSessionDelete,
			readBack: "session.catalog",
			call: func(ctx context.Context, client *Client) error {
				return Delete(ctx, client, validDeleteParams())
			},
		},
	}
}

func mutationSuccess(t *testing.T, method MutationMethod) fakeMutation {
	t.Helper()
	if method == MutationSessionSubmit {
		return fakeMutation{Result: rawJSON(t, map[string]any{"event": fakeEvent(5)})}
	}
	return fakeMutation{}
}

func requireUnknownResult(t *testing.T, err error, method MutationMethod, appError, readBack string) {
	t.Helper()
	if !errors.Is(err, ErrUnknownResult) {
		t.Fatalf("error = %v, want ErrUnknownResult", err)
	}
	var unknown *UnknownResultError
	if !errors.As(err, &unknown) {
		t.Fatalf("error = %T, want *UnknownResultError", err)
	}
	if unknown.Method != method || unknown.AppError != appError || unknown.ReadBack != readBack {
		t.Fatalf("unknown result = %#v, want method %q appError %q readBack %q", unknown, method, appError, readBack)
	}
}

func requireMutationError(t *testing.T, err error, method MutationMethod, appError string) *MutationError {
	t.Helper()
	var mutationErr *MutationError
	if !errors.As(err, &mutationErr) {
		t.Fatalf("error = %v, want *MutationError", err)
	}
	if mutationErr.Method != method || mutationErr.AppError != appError {
		t.Fatalf("mutation error = %#v, want method %q appError %q", mutationErr, method, appError)
	}
	return mutationErr
}

func TestSubmitMutation(t *testing.T) {
	t.Run("success records the idempotency key and decodes the event", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionSubmit): {mutationSuccess(t, MutationSessionSubmit)},
			},
		})
		result, err := Submit(testContext(t), client, validSubmitParams())
		if err != nil {
			t.Fatalf("Submit: %v", err)
		}
		if result.Event.Sequence != 5 || result.Event.ID != "event-5" {
			t.Fatalf("event = %#v", result.Event)
		}
		calls := server.recordedCalls(string(MutationSessionSubmit))
		if len(calls) != 1 {
			t.Fatalf("submit calls = %d, want 1", len(calls))
		}
		params := rawParams(t, calls[0])
		for key, want := range map[string]string{
			"sessionId":    "s1",
			"submissionId": "submission-1",
			"branchId":     "main",
			"text":         "hello",
		} {
			var got string
			if err := json.Unmarshal(params[key], &got); err != nil {
				t.Fatalf("param %s: %v", key, err)
			}
			if got != want {
				t.Fatalf("param %s = %q, want %q", key, got, want)
			}
		}
		if _, ok := params["mode"]; ok {
			t.Fatal("submit carried an unset mode")
		}
	})

	t.Run("invalid params fail before the call", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionSubmit): {mutationSuccess(t, MutationSessionSubmit)},
			},
		})
		params := validSubmitParams()
		params.Text = ""
		if _, err := Submit(testContext(t), client, params); !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if calls := server.recordedCalls(string(MutationSessionSubmit)); len(calls) != 0 {
			t.Fatalf("submit was called %d times with invalid params", len(calls))
		}
	})

	t.Run("unknown appError stays observable and maps to internal", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionSubmit): {
					{Fault: &fakeRPCError{Code: -32000, Message: "future", Data: appErrorData("future_appearance")}},
				},
			},
		})
		_, err := Submit(testContext(t), client, validSubmitParams())
		mutationErr := requireMutationError(t, err, MutationSessionSubmit, "future_appearance")
		if mutationErr.Mapped.Class != ErrorClassInternal || mutationErr.Mapped.Retryable {
			t.Fatalf("mapped = %#v, want non-retryable internal", mutationErr.Mapped)
		}
		if mutationErr.Unsupported() || errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("unknown appError reported as a capability error: %v", err)
		}
		var mapped MappedError
		if !AsMappedError(err, &mapped) || mapped.Class != ErrorClassInternal {
			t.Fatalf("AsMappedError = %#v, %v", mapped, err)
		}
		if calls := server.recordedCalls(string(MutationSessionSubmit)); len(calls) != 1 {
			t.Fatalf("submit calls = %d, want no retry", len(calls))
		}
	})

	t.Run("submission_unknown is an unknown result with input_lookup read-back", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionSubmit): {
					{Fault: &fakeRPCError{Code: -32000, Message: "unknown", Data: appErrorData(AppErrorSubmissionUnknown)}},
				},
			},
		})
		_, err := Submit(testContext(t), client, validSubmitParams())
		requireUnknownResult(t, err, MutationSessionSubmit, AppErrorSubmissionUnknown, "session.input_lookup")
		if calls := server.recordedCalls(string(MutationSessionSubmit)); len(calls) != 1 {
			t.Fatalf("submit calls = %d, want no resend", len(calls))
		}
	})

	t.Run("submission_ambiguous asks for a refresh and never resends", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionSubmit): {
					{Fault: &fakeRPCError{Code: -32000, Message: "ambiguous", Data: appErrorData(AppErrorSubmissionAmbiguous)}},
				},
			},
		})
		_, err := Submit(testContext(t), client, validSubmitParams())
		requireUnknownResult(t, err, MutationSessionSubmit, AppErrorSubmissionAmbiguous, "")
		var unknown *UnknownResultError
		if !errors.As(err, &unknown) || !unknown.Refresh {
			t.Fatalf("error = %#v, want Refresh", err)
		}
		if calls := server.recordedCalls(string(MutationSessionSubmit)); len(calls) != 1 {
			t.Fatalf("submit calls = %d, want no resend", len(calls))
		}
	})

	t.Run("malformed result is visible as protocol plus unknown result", func(t *testing.T) {
		_, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionSubmit): {
					{Result: json.RawMessage(`{"event":{"id":"event-5"}}`)},
				},
			},
		})
		_, err := Submit(testContext(t), client, validSubmitParams())
		if !errors.Is(err, ErrUnknownResult) || !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrUnknownResult + ErrProtocol", err)
		}
	})
}

func TestStopMutation(t *testing.T) {
	t.Run("success ignores the response body", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionStop): {{Result: json.RawMessage(`{"ignored":true}`)}},
			},
		})
		if err := Stop(testContext(t), client, validStopParams()); err != nil {
			t.Fatalf("Stop: %v", err)
		}
		if calls := server.recordedCalls(string(MutationSessionStop)); len(calls) != 1 {
			t.Fatalf("stop calls = %d, want 1", len(calls))
		}
	})

	t.Run("session_closed is a definitive rejection", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionStop): {
					{Fault: &fakeRPCError{Code: -32000, Message: "closed", Data: appErrorData(AppErrorSessionClosed)}},
				},
			},
		})
		err := Stop(testContext(t), client, validStopParams())
		mutationErr := requireMutationError(t, err, MutationSessionStop, AppErrorSessionClosed)
		if mutationErr.Mapped.Class != ErrorClassSessionClosed || mutationErr.Mapped.Retryable {
			t.Fatalf("mapped = %#v", mutationErr.Mapped)
		}
		if calls := server.recordedCalls(string(MutationSessionStop)); len(calls) != 1 {
			t.Fatalf("stop calls = %d, want no retry", len(calls))
		}
	})
}

func TestApprovalRespondMutation(t *testing.T) {
	t.Run("allow is sent as the collapsed decision", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationApprovalRespond): {{Result: json.RawMessage(`{}`)}},
			},
		})
		outcome, err := ApprovalRespond(testContext(t), client, validApprovalParams())
		if err != nil || outcome != ApprovalOutcomeResponded {
			t.Fatalf("ApprovalRespond = %q, %v", outcome, err)
		}
		calls := server.recordedCalls(string(MutationApprovalRespond))
		if len(calls) != 1 {
			t.Fatalf("approval calls = %d, want 1", len(calls))
		}
		var got string
		if err := json.Unmarshal(rawParams(t, calls[0])["decision"], &got); err != nil {
			t.Fatalf("decision: %v", err)
		}
		if got != DecisionAllow {
			t.Fatalf("decision = %q, want allow", got)
		}
	})

	t.Run("invalid decision fails before the call", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationApprovalRespond): {{Result: json.RawMessage(`{}`)}},
			},
		})
		params := validApprovalParams()
		params.Decision = "allow_once"
		if _, err := ApprovalRespond(testContext(t), client, params); !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if calls := server.recordedCalls(string(MutationApprovalRespond)); len(calls) != 0 {
			t.Fatalf("approval was called %d times with invalid params", len(calls))
		}
	})

	t.Run("-32602 means already cancelled and is not an error", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationApprovalRespond): {{Fault: fakeBadParams()}},
			},
		})
		outcome, err := ApprovalRespond(testContext(t), client, validApprovalParams())
		if err != nil || outcome != ApprovalOutcomeCancelled {
			t.Fatalf("ApprovalRespond = %q, %v; want cancelled, nil", outcome, err)
		}
		if calls := server.recordedCalls(string(MutationApprovalRespond)); len(calls) != 1 {
			t.Fatalf("approval calls = %d, want no retry", len(calls))
		}
	})
}

func TestDeleteMutation(t *testing.T) {
	t.Run("success sends only the contract-defined sessionId", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionDelete): {{Result: json.RawMessage(`{"deleted":true}`)}},
			},
		})
		if err := Delete(testContext(t), client, validDeleteParams()); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		calls := server.recordedCalls(string(MutationSessionDelete))
		if len(calls) != 1 {
			t.Fatalf("delete calls = %d, want 1", len(calls))
		}
		params := rawParams(t, calls[0])
		if len(params) != 1 {
			t.Fatalf("delete params = %v, want only sessionId", params)
		}
		if _, ok := params["sessionId"]; !ok {
			t.Fatal("delete params omitted sessionId")
		}
	})

	t.Run("session_active keeps the row and is not retried", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationSessionDelete): {
					{Fault: &fakeRPCError{Code: -32000, Message: "active", Data: appErrorData(AppErrorSessionActive)}},
				},
			},
		})
		err := Delete(testContext(t), client, validDeleteParams())
		mutationErr := requireMutationError(t, err, MutationSessionDelete, AppErrorSessionActive)
		if mutationErr.Mapped.Class != ErrorClassSessionActive || mutationErr.Mapped.Retryable {
			t.Fatalf("mapped = %#v", mutationErr.Mapped)
		}
		if calls := server.recordedCalls(string(MutationSessionDelete)); len(calls) != 1 {
			t.Fatalf("delete calls = %d, want no retry", len(calls))
		}
	})
}

func TestMutationUnknownMethodIsVisibleWIP(t *testing.T) {
	for _, tc := range mutationCases() {
		t.Run(string(tc.method), func(t *testing.T) {
			server, client := newFakeServer(t, fakeConfig{
				Hello: fakeHello(),
				Mutations: map[string][]fakeMutation{
					string(tc.method): {{Fault: fakeUnknownMethod()}},
				},
			})
			err := tc.call(testContext(t), client)
			mutationErr := requireMutationError(t, err, tc.method, AppErrorUnknownMethod)
			if !mutationErr.Unsupported() {
				t.Fatalf("Unsupported = false for %v", err)
			}
			if !errors.Is(err, ErrCapabilityUnavailable) {
				t.Fatalf("errors.Is(%v, ErrCapabilityUnavailable) = false", err)
			}
			if mutationErr.Mapped.Class != ErrorClassCapability || mutationErr.Mapped.Retryable {
				t.Fatalf("mapped = %#v, want non-retryable capability", mutationErr.Mapped)
			}
			if calls := server.recordedCalls(string(tc.method)); len(calls) != 1 {
				t.Fatalf("%s calls = %d, want no retry", tc.method, len(calls))
			}
		})
	}
}

func TestMutationTimeoutIsUnknownResult(t *testing.T) {
	for _, tc := range mutationCases() {
		t.Run(string(tc.method), func(t *testing.T) {
			gate := newCatalogGate()
			t.Cleanup(gate.open)
			server, client := newFakeServer(t, fakeConfig{
				Hello:        fakeHello(),
				MutationGate: map[string]*catalogGate{string(tc.method): gate},
				Mutations: map[string][]fakeMutation{
					string(tc.method): {mutationSuccess(t, tc.method)},
				},
			})
			ctx, cancel := context.WithCancel(testContext(t))
			defer cancel()
			out := make(chan error, 1)
			go func() { out <- tc.call(ctx, client) }()
			select {
			case <-gate.reached:
			case <-time.After(fakeWaitTimeout):
				t.Fatal("mutation did not reach the fake server")
			}
			cancel()
			var err error
			select {
			case err = <-out:
			case <-time.After(fakeWaitTimeout):
				t.Fatal("mutation did not finish")
			}
			requireUnknownResult(t, err, tc.method, "request_aborted", tc.readBack)
			if calls := server.recordedCalls(string(tc.method)); len(calls) != 1 {
				t.Fatalf("%s calls = %d, want no resend", tc.method, len(calls))
			}
		})
	}
}

func TestMutationDisconnectIsUnknownResult(t *testing.T) {
	for _, tc := range mutationCases() {
		t.Run(string(tc.method), func(t *testing.T) {
			server, client := newFakeServer(t, fakeConfig{
				Hello:         fakeHello(),
				CloseOnMethod: map[string]bool{string(tc.method): true},
			})
			err := tc.call(testContext(t), client)
			requireUnknownResult(t, err, tc.method, "connection_closed", tc.readBack)
			if calls := server.recordedCalls(string(tc.method)); len(calls) != 1 {
				t.Fatalf("%s calls = %d, want no resend", tc.method, len(calls))
			}
		})
	}
}

func TestMapMutationErrorTimeoutIsUnknownResult(t *testing.T) {
	for _, tc := range mutationCases() {
		t.Run(string(tc.method), func(t *testing.T) {
			err := mapMutationError(tc.method, &LocalError{AppError: "request_timeout"})
			requireUnknownResult(t, err, tc.method, "request_timeout", tc.readBack)
			if !errors.Is(err, ErrUnknownResult) {
				t.Fatalf("error = %v, want ErrUnknownResult", err)
			}
		})
	}
}
