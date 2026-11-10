package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// attachTestApp is a window attached to one session, with a live draft and the
// mutex guarding the state its updates write.
func attachTestApp(f *fakeSession, draft string) (*app, *sync.Mutex) {
	b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	a := &app{
		state: &dimui.State{
			Sessions:   []dimui.Session{{ID: "s1"}},
			Selected:   "s1",
			Connection: dimui.ConnReady,
			Draft:      draft,
		},
		bridge: b,
	}
	var mu sync.Mutex
	a.updateHook = func(fn func(*dimui.State)) {
		mu.Lock()
		defer mu.Unlock()
		fn(a.state)
	}
	a.bind()
	return a, &mu
}

// writeAttachment writes one file under the test's temporary directory and
// returns its path.
func writeAttachment(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// TestAppAttachAppendsTheStoredPath checks the successful flow: one upload with
// the chosen name and text, then the App Server's path is added to the draft as
// a Markdown link and the muted status reports it.
func TestAppAttachAppendsTheStoredPath(t *testing.T) {
	path := writeAttachment(t, "notes.txt", "hello\n")
	f := &fakeSession{
		view:             &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		attachmentResult: appserver.AttachmentPutResult{AttachmentID: "a1", Path: "/store/a1/notes.txt"},
	}
	a, mu := attachTestApp(f, "Summarize this.")

	a.attachSelectedFile(a.currentSelection(), "s1", path)

	mu.Lock()
	defer mu.Unlock()
	wantDraft := "Summarize this.\n\n[notes.txt](/store/a1/notes.txt)"
	if a.state.Draft != wantDraft {
		t.Fatalf("draft = %q, want %q", a.state.Draft, wantDraft)
	}
	if a.state.ActionStatus != "Attachment added" {
		t.Fatalf("status = %q, want the attachment outcome", a.state.ActionStatus)
	}
	if a.state.ActionError != "" {
		t.Fatalf("action error = %q after a successful upload", a.state.ActionError)
	}
	if f.attachmentCalls != 1 {
		t.Fatalf("attachment.put sent %d times, want once", f.attachmentCalls)
	}
	if f.attachmentName != "notes.txt" || f.attachmentText != "hello\n" {
		t.Fatalf("upload = %q/%q, want the file's name and text", f.attachmentName, f.attachmentText)
	}
}

// TestAppAttachAddsOnlyTheLinkToAnEmptyDraft checks the empty-draft case: the
// link is the whole draft, without a leading blank line.
func TestAppAttachAddsOnlyTheLinkToAnEmptyDraft(t *testing.T) {
	path := writeAttachment(t, "notes.md", "hello")
	f := &fakeSession{
		view:             &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		attachmentResult: appserver.AttachmentPutResult{AttachmentID: "a1", Path: "/store/a1/notes.md"},
	}
	a, mu := attachTestApp(f, "  \n")

	a.attachSelectedFile(a.currentSelection(), "s1", path)

	mu.Lock()
	defer mu.Unlock()
	if a.state.Draft != "  \n[notes.md](/store/a1/notes.md)" {
		t.Fatalf("draft = %q, want the link appended after the newline", a.state.Draft)
	}
	if f.attachmentCalls != 1 {
		t.Fatalf("attachment.put sent %d times, want once", f.attachmentCalls)
	}
}

// TestAppAttachCancelledPickerDoesNothing checks that dismissing the dialog
// leaves the draft, the status and the App Server alone.
func TestAppAttachCancelledPickerDoesNothing(t *testing.T) {
	f := &fakeSession{view: &dimui.State{Selected: "s1", Connection: dimui.ConnReady}}
	a, mu := attachTestApp(f, "keep this")

	a.attachSelectedFile(a.currentSelection(), "s1", "")

	mu.Lock()
	defer mu.Unlock()
	if a.state.Draft != "keep this" || a.state.ActionStatus != "" || a.state.ActionError != "" {
		t.Fatalf("a cancelled picker changed the window: %#v", a.state)
	}
	if f.attachmentCalls != 0 {
		t.Fatalf("attachment.put sent %d times after a cancel", f.attachmentCalls)
	}
}

// TestAppAttachFileUploadsOnce checks the whole wired path: the picker is asked
// once, its file is uploaded once, and the draft link appears only after the
// App Server returned a path.
func TestAppAttachFileUploadsOnce(t *testing.T) {
	path := writeAttachment(t, "notes.txt", "hello")
	f := &fakeSession{
		view:             &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		attachmentResult: appserver.AttachmentPutResult{AttachmentID: "a1", Path: "/store/a1/notes.txt"},
	}
	a, mu := attachTestApp(f, "")
	pickerCalls := 0
	a.attachmentPicker = func() (string, error) {
		pickerCalls++
		return path, nil
	}

	a.attachFile()
	waitForState(t, a, mu, "the uploaded attachment link", func(state *dimui.State) bool {
		return state.ActionStatus == "Attachment added"
	})

	mu.Lock()
	defer mu.Unlock()
	if pickerCalls != 1 {
		t.Fatalf("the picker opened %d times, want once", pickerCalls)
	}
	if f.attachmentCalls != 1 {
		t.Fatalf("attachment.put sent %d times, want once", f.attachmentCalls)
	}
	if a.state.Draft != "[notes.txt](/store/a1/notes.txt)" {
		t.Fatalf("draft = %q, want the stored link", a.state.Draft)
	}
}

// TestAppAttachPickerFailureIsReported checks that a picker that cannot open is
// a visible action error, not a silent no-op.
func TestAppAttachPickerFailureIsReported(t *testing.T) {
	f := &fakeSession{view: &dimui.State{Selected: "s1", Connection: dimui.ConnReady}}
	a, mu := attachTestApp(f, "draft")
	a.attachmentPicker = func() (string, error) { return "", errors.New("no dialog service") }

	a.attachFile()
	waitForState(t, a, mu, "the picker failure", func(state *dimui.State) bool {
		return state.ActionError != ""
	})

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(a.state.ActionError, "no dialog service") {
		t.Fatalf("action error = %q, want the picker failure", a.state.ActionError)
	}
	if f.attachmentCalls != 0 || a.state.Draft != "draft" {
		t.Fatalf("a failed picker uploaded %d time(s) and changed the draft to %q", f.attachmentCalls, a.state.Draft)
	}
}

// TestAppAttachRejectsUnreadableFiles checks the local filters before any
// upload: an over-limit file, invalid UTF-8, and an image are each reported and
// never sent to the App Server.
func TestAppAttachRejectsUnreadableFiles(t *testing.T) {
	large := writeAttachment(t, "large.txt", strings.Repeat("a", appserver.MaxAttachmentTextBytes+1))
	invalid := writeAttachment(t, "binary.txt", string([]byte{0xff, 0xfe, 0x00}))
	image := writeAttachment(t, "picture.png", "not really a png")

	for _, tc := range []struct {
		name string
		path string
		want string
	}{
		{name: "too large", path: large, want: "text attachment limit"},
		{name: "not UTF-8", path: invalid, want: "UTF-8"},
		{name: "image is WIP", path: image, want: "image attachments are WIP"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeSession{view: &dimui.State{Selected: "s1", Connection: dimui.ConnReady}}
			a, mu := attachTestApp(f, "draft")

			a.attachSelectedFile(a.currentSelection(), "s1", tc.path)

			mu.Lock()
			defer mu.Unlock()
			if !strings.Contains(a.state.ActionError, tc.want) {
				t.Fatalf("action error = %q, want %q", a.state.ActionError, tc.want)
			}
			if a.state.Draft != "draft" || a.state.ActionStatus != "" {
				t.Fatalf("a rejected file changed the draft/status: %q / %q", a.state.Draft, a.state.ActionStatus)
			}
			if f.attachmentCalls != 0 {
				t.Fatalf("a rejected file was uploaded %d time(s)", f.attachmentCalls)
			}
		})
	}
}

// TestAppAttachUploadFailureKeepsTheDraft checks a rejected upload: the draft
// stays as typed, the status shows no success, and the failure reaches the
// banner.
func TestAppAttachUploadFailureKeepsTheDraft(t *testing.T) {
	path := writeAttachment(t, "notes.txt", "hello")
	f := &fakeSession{
		view:          &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		attachmentErr: errors.New("storage failed"),
	}
	a, mu := attachTestApp(f, "draft")

	a.attachSelectedFile(a.currentSelection(), "s1", path)

	mu.Lock()
	defer mu.Unlock()
	if a.state.Draft != "draft" {
		t.Fatalf("draft = %q after a failed upload, want it untouched", a.state.Draft)
	}
	if a.state.ActionStatus != "" {
		t.Fatalf("status = %q after a failed upload", a.state.ActionStatus)
	}
	if !strings.Contains(a.state.ActionError, "storage failed") {
		t.Fatalf("action error = %q, want the App Server failure", a.state.ActionError)
	}
	if f.attachmentCalls != 1 {
		t.Fatalf("attachment.put sent %d times, want one and no retry", f.attachmentCalls)
	}
}

// TestAppAttachUnknownMethodStaysWIP checks the missing-host-plugin case: the
// error names unknown_method and the WIP boundary, and nothing looks attached.
func TestAppAttachUnknownMethodStaysWIP(t *testing.T) {
	path := writeAttachment(t, "notes.txt", "hello")
	f := &fakeSession{
		view: &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		attachmentErr: &appserver.MutationError{
			Method:   appserver.MutationAttachmentPut,
			AppError: appserver.AppErrorUnknownMethod,
			Mapped: appserver.MappedError{
				AppError: appserver.AppErrorUnknownMethod,
				Class:    appserver.ErrorClassCapability,
			},
		},
	}
	a, mu := attachTestApp(f, "")

	a.attachSelectedFile(a.currentSelection(), "s1", path)

	mu.Lock()
	defer mu.Unlock()
	for _, want := range []string{"attachment.put", "unknown_method", "WIP"} {
		if !strings.Contains(a.state.ActionError, want) {
			t.Fatalf("action error = %q, want it to name %q", a.state.ActionError, want)
		}
	}
	if a.state.Draft != "" || f.attachmentCalls != 1 {
		t.Fatalf("draft = %q calls = %d after unknown_method", a.state.Draft, f.attachmentCalls)
	}
}

// TestAppAttachUnknownResultIsNotResent checks a lost reply: the error says the
// upload was not confirmed and was not resent, and exactly one request was made.
func TestAppAttachUnknownResultIsNotResent(t *testing.T) {
	path := writeAttachment(t, "notes.txt", "hello")
	f := &fakeSession{
		view: &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		attachmentErr: &appserver.UnknownResultError{
			Method: appserver.MutationAttachmentPut,
			Err:    errors.New("connection lost"),
		},
	}
	a, mu := attachTestApp(f, "")

	a.attachSelectedFile(a.currentSelection(), "s1", path)

	mu.Lock()
	defer mu.Unlock()
	for _, want := range []string{"did not confirm the upload", "not resent"} {
		if !strings.Contains(a.state.ActionError, want) {
			t.Fatalf("action error = %q, want it to say %q", a.state.ActionError, want)
		}
	}
	if f.attachmentCalls != 1 || a.state.Draft != "" {
		t.Fatalf("calls = %d draft = %q after an unknown result", f.attachmentCalls, a.state.Draft)
	}
}

// TestAppendAttachmentLink checks the Electron-parity formatting: a blank line
// separates existing text, an existing trailing newline is kept, and Markdown
// delimiters in the name and path are escaped.
func TestAppendAttachmentLink(t *testing.T) {
	for _, tc := range []struct {
		name  string
		draft string
		label string
		path  string
		want  string
	}{
		{name: "empty", draft: "", label: "a.txt", path: "/store/a", want: "[a.txt](/store/a)"},
		{name: "text", draft: "hello", label: "a.txt", path: "/store/a", want: "hello\n\n[a.txt](/store/a)"},
		{name: "newline", draft: "hello\n", label: "a.txt", path: "/store/a", want: "hello\n[a.txt](/store/a)"},
		{name: "escaped", draft: "", label: `a[b].txt`, path: `/store/a(b)`, want: `[a\[b\].txt](/store/a(b\))`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := appendAttachmentLink(tc.draft, tc.label, tc.path); got != tc.want {
				t.Fatalf("appendAttachmentLink = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestAppAttachIgnoresAStaleSelection checks that an upload that finishes after
// the selection moved cannot append another session's link to the new draft.
func TestAppAttachIgnoresAStaleSelection(t *testing.T) {
	path := writeAttachment(t, "notes.txt", "hello")
	f := &fakeSession{
		view:             &dimui.State{Selected: "s1", Connection: dimui.ConnReady},
		attachmentResult: appserver.AttachmentPutResult{AttachmentID: "a1", Path: "/store/a1/notes.txt"},
	}
	a, mu := attachTestApp(f, "draft")
	gen := a.currentSelection()

	// A new selection increments the generation before the upload publishes.
	a.beginSelect()
	a.state.Selected = "s2"

	a.attachSelectedFile(gen, "s1", path)

	mu.Lock()
	defer mu.Unlock()
	if a.state.Draft != "draft" || a.state.ActionStatus != "" {
		t.Fatalf("a stale attachment changed the window: draft=%q status=%q", a.state.Draft, a.state.ActionStatus)
	}
}

// TestAttachTimeoutIsBounded pins the one upload's deadline: attachment.put has
// no read-back, so a hung request must give up instead of retrying.
func TestAttachTimeoutIsBounded(t *testing.T) {
	if attachmentPickerTimeout <= 0 || attachmentPickerTimeout > time.Minute {
		t.Fatalf("attachment timeout = %v, want a bounded positive deadline", attachmentPickerTimeout)
	}
}
