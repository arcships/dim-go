package session

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// TestSessionAttachmentPutSendsOneTextUpload drives attachment.put through the
// session: one request carrying the selected session, the name and the text, and
// the App Server's path returned unchanged.
func TestSessionAttachmentPutSendsOneTextUpload(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
		History: emptyHistory(t, 0),
		Mutations: map[string][]fakeMutation{
			string(appserver.MutationAttachmentPut): {{
				Result: json.RawMessage(`{"attachmentId":"a1","path":"/store/a1/notes.txt","bytes":5}`),
			}},
		},
	}, []string{"session.history"}, OpenOptions{})

	result, err := s.AttachmentPut(testContext(t), "notes.txt", "hello")
	if err != nil {
		t.Fatalf("AttachmentPut: %v", err)
	}
	if result.AttachmentID != "a1" || result.Path != "/store/a1/notes.txt" || result.Bytes == nil || *result.Bytes != 5 {
		t.Fatalf("result = %#v, want the App Server's reply", result)
	}
	calls := server.callsFor(string(appserver.MutationAttachmentPut))
	if len(calls) != 1 {
		t.Fatalf("attachment.put calls = %d, want exactly one", len(calls))
	}
	var params appserver.AttachmentPutParams
	if err := json.Unmarshal(calls[0].Params, &params); err != nil {
		t.Fatalf("decode params: %v", err)
	}
	if params.SessionID != "s1" || params.Name != "notes.txt" || params.Text != "hello" {
		t.Fatalf("params = %#v, want the selected session and the chosen text", params)
	}
}

// TestSessionAttachmentPutUnknownMethodIsAWIPGap checks that a host without the
// Attachments plugin reports the missing method as a capability the caller can
// disable, and that the request is not retried.
func TestSessionAttachmentPutUnknownMethodIsAWIPGap(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
		History: emptyHistory(t, 0),
		Mutations: map[string][]fakeMutation{
			string(appserver.MutationAttachmentPut): {{Fault: unknownMethod()}},
		},
	}, []string{"session.history"}, OpenOptions{})

	_, err := s.AttachmentPut(testContext(t), "notes.txt", "hello")
	if !errors.Is(err, appserver.ErrCapabilityUnavailable) {
		t.Fatalf("error = %v, want a capability unavailable error", err)
	}
	var mutation *appserver.MutationError
	if !errors.As(err, &mutation) || !mutation.Unsupported() {
		t.Fatalf("error = %#v, want an unsupported mutation", err)
	}
	if calls := server.callsFor(string(appserver.MutationAttachmentPut)); len(calls) != 1 {
		t.Fatalf("attachment.put calls = %d, want no retry", len(calls))
	}
}

// TestSessionAttachmentPutMalformedResultIsUnknownAndNotResent keeps a lost
// reply honest: a result without the required path is an unknown result, and
// the upload is never sent twice.
func TestSessionAttachmentPutMalformedResultIsUnknownAndNotResent(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
		History: emptyHistory(t, 0),
		Mutations: map[string][]fakeMutation{
			string(appserver.MutationAttachmentPut): {{
				Result: json.RawMessage(`{"attachmentId":"a1"}`),
			}},
		},
	}, []string{"session.history"}, OpenOptions{})

	_, err := s.AttachmentPut(testContext(t), "notes.txt", "hello")
	if !errors.Is(err, appserver.ErrUnknownResult) {
		t.Fatalf("error = %v, want an unknown result", err)
	}
	if calls := server.callsFor(string(appserver.MutationAttachmentPut)); len(calls) != 1 {
		t.Fatalf("attachment.put calls = %d, want no resend", len(calls))
	}
}
