package appserver

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func validAttachmentPutParams() AttachmentPutParams {
	return AttachmentPutParams{SessionID: "s1", Name: "notes.txt", Text: "hello attachment"}
}

func attachmentPutSuccess(t *testing.T) fakeMutation {
	t.Helper()
	return fakeMutation{Result: rawJSON(t, map[string]any{
		"attachmentId": "att-1",
		"path":         "/sessions/s1/files/notes.txt",
		"bytes":        16,
	})}
}

func TestAttachmentPutParamsValidate(t *testing.T) {
	valid := []AttachmentPutParams{
		validAttachmentPutParams(),
		{SessionID: "s1", Text: "hello"},
		{SessionID: "s1", Text: strings.Repeat("a", MaxAttachmentTextBytes)},
	}
	for i, params := range valid {
		if err := params.Validate(); err != nil {
			t.Errorf("valid params[%d] rejected: %v", i, err)
		}
	}

	invalid := []struct {
		name   string
		params AttachmentPutParams
	}{
		{name: "missing sessionId", params: AttachmentPutParams{Text: "hello"}},
		{name: "empty text", params: AttachmentPutParams{SessionID: "s1"}},
		{name: "blank name", params: AttachmentPutParams{SessionID: "s1", Name: "   ", Text: "hello"}},
		{name: "text over 10 MiB", params: AttachmentPutParams{SessionID: "s1", Text: strings.Repeat("a", MaxAttachmentTextBytes+1)}},
		{name: "invalid UTF-8", params: AttachmentPutParams{SessionID: "s1", Text: string([]byte{0xff})}},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
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

func TestDecodeAttachmentPutResult(t *testing.T) {
	valid := []struct {
		name  string
		raw   string
		path  string
		bytes *int64
	}{
		{
			name:  "full result",
			raw:   `{"attachmentId":"att-1","path":"/sessions/s1/files/a.txt","bytes":12}`,
			path:  "/sessions/s1/files/a.txt",
			bytes: int64Ptr(12),
		},
		{
			name: "bytes is optional",
			raw:  `{"attachmentId":"att-1","path":"/sessions/s1/files/a.txt"}`,
			path: "/sessions/s1/files/a.txt",
		},
		{
			name:  "unknown members are ignored",
			raw:   `{"attachmentId":"att-1","path":"/p","bytes":0,"future":true}`,
			path:  "/p",
			bytes: int64Ptr(0),
		},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecodeAttachmentPutResult(json.RawMessage(tc.raw))
			if err != nil {
				t.Fatalf("DecodeAttachmentPutResult: %v", err)
			}
			if result.AttachmentID == "" || result.Path != tc.path {
				t.Fatalf("result = %#v", result)
			}
			if !equalInt64Ptr(result.Bytes, tc.bytes) {
				t.Fatalf("bytes = %v, want %v", result.Bytes, tc.bytes)
			}
		})
	}

	invalid := []struct {
		name string
		raw  string
	}{
		{name: "null result", raw: `null`},
		{name: "not an object", raw: `[]`},
		{name: "missing attachmentId", raw: `{"path":"/p"}`},
		{name: "empty attachmentId", raw: `{"attachmentId":"","path":"/p"}`},
		{name: "missing path", raw: `{"attachmentId":"att-1"}`},
		{name: "empty path", raw: `{"attachmentId":"att-1","path":""}`},
		{name: "path is not a string", raw: `{"attachmentId":"att-1","path":7}`},
		{name: "negative bytes", raw: `{"attachmentId":"att-1","path":"/p","bytes":-1}`},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeAttachmentPutResult(json.RawMessage(tc.raw))
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

func TestAttachmentPutHelper(t *testing.T) {
	t.Run("success sends the exact request and decodes the result", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationAttachmentPut): {attachmentPutSuccess(t)},
			},
		})
		result, err := AttachmentPut(testContext(t), client, validAttachmentPutParams())
		if err != nil {
			t.Fatalf("AttachmentPut: %v", err)
		}
		if result.AttachmentID != "att-1" || result.Path != "/sessions/s1/files/notes.txt" {
			t.Fatalf("result = %#v", result)
		}
		if !equalInt64Ptr(result.Bytes, int64Ptr(16)) {
			t.Fatalf("bytes = %v, want 16", result.Bytes)
		}
		calls := server.recordedCalls(string(MutationAttachmentPut))
		if len(calls) != 1 {
			t.Fatalf("attachment.put calls = %d, want 1", len(calls))
		}
		params := rawParams(t, calls[0])
		if len(params) != 3 {
			t.Fatalf("params = %s, want sessionId, name and text", calls[0].Params)
		}
		for key, want := range map[string]string{
			"sessionId": `"s1"`,
			"name":      `"notes.txt"`,
			"text":      `"hello attachment"`,
		} {
			if got := string(params[key]); got != want {
				t.Fatalf("param %s = %s, want %s", key, got, want)
			}
		}
	})

	t.Run("name is omitted when unset", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationAttachmentPut): {attachmentPutSuccess(t)},
			},
		})
		params := validAttachmentPutParams()
		params.Name = ""
		if _, err := AttachmentPut(testContext(t), client, params); err != nil {
			t.Fatalf("AttachmentPut: %v", err)
		}
		calls := server.recordedCalls(string(MutationAttachmentPut))
		if len(calls) != 1 {
			t.Fatalf("attachment.put calls = %d, want 1", len(calls))
		}
		if got := rawParams(t, calls[0]); len(got) != 2 {
			t.Fatalf("params = %s, want sessionId and text only", calls[0].Params)
		}
	})

	t.Run("invalid params fail before the call", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationAttachmentPut): {attachmentPutSuccess(t)},
			},
		})
		params := validAttachmentPutParams()
		params.Text = ""
		if _, err := AttachmentPut(testContext(t), client, params); !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if calls := server.recordedCalls(string(MutationAttachmentPut)); len(calls) != 0 {
			t.Fatalf("attachment.put was called %d times with invalid params", len(calls))
		}
	})

	t.Run("JSON escaping that overflows the frame limit fails before the call", func(t *testing.T) {
		// The text is within the 10 MiB UTF-8 limit but encodes to roughly twice
		// that size, so sending it would exceed the 16 MiB frame limit.
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationAttachmentPut): {attachmentPutSuccess(t)},
			},
		})
		params := validAttachmentPutParams()
		params.Text = strings.Repeat("\n", MaxAttachmentTextBytes)
		_, err := AttachmentPut(testContext(t), client, params)
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if calls := server.recordedCalls(string(MutationAttachmentPut)); len(calls) != 0 {
			t.Fatalf("attachment.put was called %d times with an oversized encoded frame", len(calls))
		}
	})

	t.Run("malformed result is an unknown result with no read-back", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationAttachmentPut): {{Result: json.RawMessage(`{"attachmentId":"att-1"}`)}},
			},
		})
		_, err := AttachmentPut(testContext(t), client, validAttachmentPutParams())
		if !errors.Is(err, ErrUnknownResult) || !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrUnknownResult + ErrProtocol", err)
		}
		var unknown *UnknownResultError
		if !errors.As(err, &unknown) || unknown.ReadBack != "" || unknown.Refresh {
			t.Fatalf("error = %#v, want no read-back and no refresh", err)
		}
		if calls := server.recordedCalls(string(MutationAttachmentPut)); len(calls) != 1 {
			t.Fatalf("attachment.put calls = %d, want no resend", len(calls))
		}
	})

	t.Run("-32602 is a definitive bad request", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationAttachmentPut): {{Fault: fakeBadParams()}},
			},
		})
		_, err := AttachmentPut(testContext(t), client, validAttachmentPutParams())
		mutationErr := requireMutationError(t, err, MutationAttachmentPut, "")
		if mutationErr.Mapped.Class != ErrorClassBadRequest || mutationErr.Mapped.Retryable {
			t.Fatalf("mapped = %#v, want non-retryable bad request", mutationErr.Mapped)
		}
		if errors.Is(err, ErrUnknownResult) {
			t.Fatalf("error = %v, want a definitive rejection", err)
		}
		if calls := server.recordedCalls(string(MutationAttachmentPut)); len(calls) != 1 {
			t.Fatalf("attachment.put calls = %d, want no retry", len(calls))
		}
	})

	t.Run("unknown_method is visible-disabled WIP and not retried", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello(),
			Mutations: map[string][]fakeMutation{
				string(MutationAttachmentPut): {{Fault: fakeUnknownMethod()}},
			},
		})
		_, err := AttachmentPut(testContext(t), client, validAttachmentPutParams())
		mutationErr := requireMutationError(t, err, MutationAttachmentPut, AppErrorUnknownMethod)
		if !mutationErr.Unsupported() || !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("error = %v, want a capability error", err)
		}
		if mutationErr.Mapped.Class != ErrorClassCapability || mutationErr.Mapped.Retryable {
			t.Fatalf("mapped = %#v, want non-retryable capability", mutationErr.Mapped)
		}
		if calls := server.recordedCalls(string(MutationAttachmentPut)); len(calls) != 1 {
			t.Fatalf("attachment.put calls = %d, want no retry", len(calls))
		}
	})

	t.Run("attachment errors keep the appError and are not retried", func(t *testing.T) {
		for _, appError := range []string{AppErrorAttachmentInvalid, AppErrorAttachmentTooLarge} {
			t.Run(appError, func(t *testing.T) {
				server, client := newFakeServer(t, fakeConfig{
					Hello: fakeHello(),
					Mutations: map[string][]fakeMutation{
						string(MutationAttachmentPut): {
							{Fault: &fakeRPCError{Code: -32000, Message: appError, Data: appErrorData(appError)}},
						},
					},
				})
				_, err := AttachmentPut(testContext(t), client, validAttachmentPutParams())
				mutationErr := requireMutationError(t, err, MutationAttachmentPut, appError)
				if mutationErr.Mapped.Class != ErrorClassBadRequest || mutationErr.Mapped.Retryable {
					t.Fatalf("mapped = %#v, want non-retryable bad request", mutationErr.Mapped)
				}
				if calls := server.recordedCalls(string(MutationAttachmentPut)); len(calls) != 1 {
					t.Fatalf("attachment.put calls = %d, want no retry", len(calls))
				}
			})
		}
	})

	t.Run("disconnect is an unknown result with no read-back", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Hello:         fakeHello(),
			CloseOnMethod: map[string]bool{string(MutationAttachmentPut): true},
		})
		_, err := AttachmentPut(testContext(t), client, validAttachmentPutParams())
		requireUnknownResult(t, err, MutationAttachmentPut, "connection_closed", "")
		if calls := server.recordedCalls(string(MutationAttachmentPut)); len(calls) != 1 {
			t.Fatalf("attachment.put calls = %d, want no resend", len(calls))
		}
	})
}
