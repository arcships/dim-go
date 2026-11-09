package appserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func strPtr(value string) *string { return &value }

func renameFake(result string) fakeConfig {
	return fakeConfig{Mutations: map[string][]fakeMutation{
		"session.rename": {{Result: json.RawMessage(result)}},
	}}
}

func TestRenameSendsTypedRequestAndDecodesReply(t *testing.T) {
	server, client := newFakeServer(t, renameFake(`{"title":"Release checklist","titleSource":"manual","revision":3}`))
	ctx, cancel := context.WithTimeout(context.Background(), fakeWaitTimeout)
	defer cancel()

	result, err := Rename(ctx, client, RenameParams{SessionID: "s1", Title: strPtr("Release checklist")})
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if result.Title == nil || *result.Title != "Release checklist" {
		t.Fatalf("title = %#v", result.Title)
	}
	if result.TitleSource == nil || *result.TitleSource != TitleSourceManual || result.Revision != 3 {
		t.Fatalf("result = %#v", result)
	}
	calls := server.recordedCalls("session.rename")
	if len(calls) != 1 {
		t.Fatalf("session.rename calls = %d, want 1", len(calls))
	}
	var params map[string]json.RawMessage
	if err := json.Unmarshal(calls[0].Params, &params); err != nil {
		t.Fatalf("params: %v", err)
	}
	var sessionID string
	if err := json.Unmarshal(params["sessionId"], &sessionID); err != nil || sessionID != "s1" {
		t.Fatalf("sessionId = %q, err = %v", sessionID, err)
	}
	var title string
	if err := json.Unmarshal(params["title"], &title); err != nil || title != "Release checklist" {
		t.Fatalf("title = %q, err = %v", title, err)
	}
}

func TestRenameClearSendsExplicitNullTitle(t *testing.T) {
	server, client := newFakeServer(t, renameFake(`{"title":null,"titleSource":null,"revision":4}`))
	ctx, cancel := context.WithTimeout(context.Background(), fakeWaitTimeout)
	defer cancel()

	result, err := Rename(ctx, client, RenameParams{SessionID: "s1", Title: nil})
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if result.Title != nil || result.TitleSource != nil || result.Revision != 4 {
		t.Fatalf("result = %#v, want a clear at revision 4", result)
	}
	calls := server.recordedCalls("session.rename")
	if len(calls) != 1 {
		t.Fatalf("session.rename calls = %d, want 1", len(calls))
	}
	var params map[string]json.RawMessage
	if err := json.Unmarshal(calls[0].Params, &params); err != nil {
		t.Fatalf("params: %v", err)
	}
	raw, present := params["title"]
	if !present {
		t.Fatal("title field missing; a clear must send explicit null")
	}
	if string(raw) != "null" {
		t.Fatalf("title = %s, want null", raw)
	}
}

func TestRenameParamsValidation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		params RenameParams
	}{
		{name: "missing session", params: RenameParams{Title: strPtr("x")}},
		{name: "empty title", params: RenameParams{SessionID: "s1", Title: strPtr("")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, client := newFakeServer(t, fakeConfig{})
			if _, err := Rename(ctx, client, tc.params); !errors.Is(err, ErrProtocol) {
				t.Fatalf("error = %v, want ErrProtocol", err)
			}
		})
	}
	t.Run("nil client", func(t *testing.T) {
		if _, err := Rename(ctx, nil, RenameParams{SessionID: "s1"}); !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
	})
}

func TestDecodeRenameResultValidation(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		good bool
	}{
		{name: "manual title", raw: `{"title":"x","titleSource":"manual","revision":1}`, good: true},
		{name: "auto title", raw: `{"title":"x","titleSource":"auto","revision":2}`, good: true},
		{name: "cleared", raw: `{"title":null,"titleSource":null,"revision":0}`, good: true},
		{name: "missing title", raw: `{"titleSource":"manual","revision":1}`},
		{name: "missing titleSource", raw: `{"title":"x","revision":1}`},
		{name: "missing revision", raw: `{"title":"x","titleSource":"manual"}`},
		{name: "negative revision", raw: `{"title":"x","titleSource":"manual","revision":-1}`},
		{name: "unknown source", raw: `{"title":"x","titleSource":"generated","revision":1}`},
		{name: "title without source", raw: `{"title":"x","titleSource":null,"revision":1}`},
		{name: "source without title", raw: `{"title":null,"titleSource":"manual","revision":1}`},
		{name: "not an object", raw: `[1]`},
		{name: "null result", raw: `null`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecodeRenameResult(json.RawMessage(tc.raw))
			if tc.good {
				if err != nil {
					t.Fatalf("DecodeRenameResult(%s) = %v", tc.raw, err)
				}
				return
			}
			if !errors.Is(err, ErrProtocol) {
				t.Fatalf("DecodeRenameResult(%s) error = %v, want ErrProtocol (result %#v)", tc.raw, err, result)
			}
		})
	}
}

func TestRenameErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		fault    *fakeRPCError
		wantErr  func(*testing.T, error)
		wantCall int
	}{
		{
			name:  "unknown method is a capability error",
			fault: fakeUnknownMethod(),
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				var mutation *MutationError
				if !errors.As(err, &mutation) {
					t.Fatalf("error = %T %v, want *MutationError", err, err)
				}
				if !mutation.Unsupported() || !errors.Is(err, ErrCapabilityUnavailable) {
					t.Fatalf("mutation = %#v, want unsupported capability", mutation)
				}
			},
			wantCall: 1,
		},
		{
			name:  "-32602 wins over a misleading appError",
			fault: &fakeRPCError{Code: -32602, Message: "invalid params", Data: json.RawMessage(`{"appError":"unknown_method"}`)},
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				var mutation *MutationError
				if !errors.As(err, &mutation) || mutation.Mapped.Class != ErrorClassBadRequest {
					t.Fatalf("error = %v, want a definitive bad request", err)
				}
			},
			wantCall: 1,
		},
		{
			name:  "invalid params",
			fault: &fakeRPCError{Code: -32000, Message: "invalid params", Data: json.RawMessage(`{"appError":"invalid_params"}`)},
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				var mutation *MutationError
				if !errors.As(err, &mutation) || mutation.Mapped.Class != ErrorClassBadRequest || mutation.AppError != AppErrorInvalidParams {
					t.Fatalf("error = %v, want invalid_params bad request", err)
				}
			},
			wantCall: 1,
		},
		{
			name:  "session not found",
			fault: &fakeRPCError{Code: -32000, Message: "missing", Data: json.RawMessage(`{"appError":"session_not_found"}`)},
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				var mutation *MutationError
				if !errors.As(err, &mutation) || mutation.Mapped.Class != ErrorClassNotFound {
					t.Fatalf("error = %v, want not_found", err)
				}
			},
			wantCall: 1,
		},
		{
			name:  "storage failed stays a named internal error",
			fault: &fakeRPCError{Code: -32000, Message: "storage", Data: json.RawMessage(`{"appError":"storage_failed"}`)},
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				var mutation *MutationError
				if !errors.As(err, &mutation) || mutation.Mapped.Class != ErrorClassInternal || mutation.AppError != AppErrorStorageFailed {
					t.Fatalf("error = %v, want storage_failed internal", err)
				}
			},
			wantCall: 1,
		},
		{
			name:  "service reloading is retryable transient",
			fault: &fakeRPCError{Code: -32000, Message: "reloading", Data: json.RawMessage(`{"appError":"service_reloading"}`)},
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				var mutation *MutationError
				if !errors.As(err, &mutation) || !mutation.Mapped.Retryable {
					t.Fatalf("error = %v, want retryable transient", err)
				}
			},
			wantCall: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, client := newFakeServer(t, fakeConfig{Mutations: map[string][]fakeMutation{
				"session.rename": {{Fault: tc.fault}},
			}})
			ctx, cancel := context.WithTimeout(context.Background(), fakeWaitTimeout)
			defer cancel()
			_, err := Rename(ctx, client, RenameParams{SessionID: "s1", Title: strPtr("x")})
			if err == nil {
				t.Fatal("Rename unexpectedly succeeded")
			}
			tc.wantErr(t, err)
			if calls := server.recordedCalls("session.rename"); len(calls) != tc.wantCall {
				t.Fatalf("session.rename calls = %d, want %d", len(calls), tc.wantCall)
			}
		})
	}
}

func TestRenameLostTransportIsUnknownResult(t *testing.T) {
	server, client := newFakeServer(t, fakeConfig{CloseOnMethod: map[string]bool{"session.rename": true}})
	ctx, cancel := context.WithTimeout(context.Background(), fakeWaitTimeout)
	defer cancel()

	_, err := Rename(ctx, client, RenameParams{SessionID: "s1", Title: strPtr("x")})
	if !errors.Is(err, ErrUnknownResult) {
		t.Fatalf("error = %v, want unknown result", err)
	}
	var unknown *UnknownResultError
	if !errors.As(err, &unknown) {
		t.Fatalf("error = %T, want *UnknownResultError", err)
	}
	if unknown.ReadBack != "session.summaries" {
		t.Fatalf("read-back = %q, want session.summaries", unknown.ReadBack)
	}
	if calls := server.recordedCalls("session.rename"); len(calls) != 1 {
		t.Fatalf("session.rename calls = %d, want 1 and no retry", len(calls))
	}
}

func TestRenameMalformedResultIsUnknownAndNotRetried(t *testing.T) {
	server, client := newFakeServer(t, renameFake(`{}`))
	ctx, cancel := context.WithTimeout(context.Background(), fakeWaitTimeout)
	defer cancel()

	_, err := Rename(ctx, client, RenameParams{SessionID: "s1", Title: strPtr("x")})
	if !errors.Is(err, ErrUnknownResult) {
		t.Fatalf("error = %v, want unknown result", err)
	}
	if calls := server.recordedCalls("session.rename"); len(calls) != 1 {
		t.Fatalf("session.rename calls = %d, want 1 and no retry", len(calls))
	}
}

func TestDecodeTitleNotice(t *testing.T) {
	notice, err := DecodeTitleNotice(json.RawMessage(`{"title":"Generated","titleSource":"auto","titleRevision":2}`))
	if err != nil {
		t.Fatalf("DecodeTitleNotice: %v", err)
	}
	if notice.Title == nil || *notice.Title != "Generated" || notice.TitleSource == nil || *notice.TitleSource != TitleSourceAuto || notice.TitleRevision != 2 {
		t.Fatalf("notice = %#v", notice)
	}
	cleared, err := DecodeTitleNotice(json.RawMessage(`{"title":null,"titleSource":null,"titleRevision":3}`))
	if err != nil {
		t.Fatalf("DecodeTitleNotice clear: %v", err)
	}
	if cleared.Title != nil || cleared.TitleSource != nil || cleared.TitleRevision != 3 {
		t.Fatalf("cleared = %#v", cleared)
	}

	for _, raw := range []string{
		`{"titleSource":null,"titleRevision":1}`,
		`{"title":null,"titleRevision":1}`,
		`{"title":null,"titleSource":null}`,
		`{"title":null,"titleSource":"manual","titleRevision":1}`,
		`{"title":"x","titleSource":null,"titleRevision":1}`,
		`{"title":"x","titleSource":"auto","titleRevision":-1}`,
		`{"title":"x","titleSource":"future","titleRevision":1}`,
		`null`,
	} {
		if _, err := DecodeTitleNotice(json.RawMessage(raw)); !errors.Is(err, ErrProtocol) {
			t.Fatalf("DecodeTitleNotice(%s) error = %v, want ErrProtocol", raw, err)
		}
	}
}

func TestIsTitleNotice(t *testing.T) {
	if !IsTitleNotice(SessionNotice{Source: NoticeSourceSessionStore, Kind: NoticeKindTitle}) {
		t.Fatal("SessionStore title notice not recognized")
	}
	if IsTitleNotice(SessionNotice{Source: "other", Kind: NoticeKindTitle}) {
		t.Fatal("foreign source recognized as a title notice")
	}
	if IsTitleNotice(SessionNotice{Source: NoticeSourceSessionStore, Kind: "other"}) {
		t.Fatal("foreign kind recognized as a title notice")
	}
}
