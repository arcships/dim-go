//go:build unix

package integration

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// TestSessionControllerAttachmentPutRoundTrip drives attachment.put through the
// bridge: one request with the selected session, name and text, and the App
// Server's path returned unchanged.
func TestSessionControllerAttachmentPutRoundTrip(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	daemon.setAttachment(json.RawMessage(`{"attachmentId":"a1","path":"/store/a1/notes.txt","bytes":5}`))
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	result, err := controller.AttachmentPut(ctx, "notes.txt", "hello")
	if err != nil {
		t.Fatalf("AttachmentPut: %v", err)
	}
	if result.AttachmentID != "a1" || result.Path != "/store/a1/notes.txt" || result.Bytes == nil || *result.Bytes != 5 {
		t.Fatalf("result = %#v, want the App Server's reply", result)
	}
	params := daemon.lastParams("attachment.put")
	if params["sessionId"] != "s1" || params["name"] != "notes.txt" || params["text"] != "hello" {
		t.Fatalf("params = %#v, want the selected session, name and text", params)
	}
	if calls := daemon.callCount("attachment.put"); calls != 1 {
		t.Fatalf("attachment.put calls = %d, want exactly one", calls)
	}
}

// TestSessionControllerAttachmentPutUnknownMethodIsNotRetried checks the WIP
// boundary: a host without the Attachments plugin answers unknown_method, which
// the caller sees as a capability gap and never resends.
func TestSessionControllerAttachmentPutUnknownMethodIsNotRetried(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	daemon.fail("attachment.put", unknownMethodFault())
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	_, err = controller.AttachmentPut(ctx, "notes.txt", "hello")
	if !errors.Is(err, appserver.ErrCapabilityUnavailable) {
		t.Fatalf("error = %v, want capability unavailable", err)
	}
	var mutation *appserver.MutationError
	if !errors.As(err, &mutation) || !mutation.Unsupported() {
		t.Fatalf("error = %#v, want an unsupported mutation", err)
	}
	if calls := daemon.callCount("attachment.put"); calls != 1 {
		t.Fatalf("attachment.put calls = %d, want no retry", calls)
	}
}

// TestSessionControllerAttachmentPutUnknownResultIsNotResent keeps a malformed
// reply honest: attachment.put has no read-back method, so the caller gets an
// unknown result and the upload is never sent twice.
func TestSessionControllerAttachmentPutUnknownResultIsNotResent(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	daemon.setAttachment(json.RawMessage(`{"attachmentId":"a1"}`))
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	_, err = controller.AttachmentPut(ctx, "notes.txt", "hello")
	if !errors.Is(err, appserver.ErrUnknownResult) {
		t.Fatalf("error = %v, want an unknown result", err)
	}
	if calls := daemon.callCount("attachment.put"); calls != 1 {
		t.Fatalf("attachment.put calls = %d, want no resend", calls)
	}
}
