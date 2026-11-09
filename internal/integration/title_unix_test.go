//go:build unix

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// waitFor polls a condition bounded by testWait. Session notifications and
// daemon-side call recording have no channel to wait on without reaching into
// their internals, so the poll is bounded by the test deadline.
func waitFor(t *testing.T, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(testWait)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", message)
}

func titleSummary(title, source string, revision int64) appserver.SessionSummary {
	return appserver.SessionSummary{
		SessionID:     "s1",
		Title:         &title,
		TitleSource:   &source,
		TitleRevision: &revision,
	}
}

func unknownMethodFault() *appserver.RPCError {
	return &appserver.RPCError{Code: -32601, Message: "unknown method", Data: json.RawMessage(`{"appError":"unknown_method"}`)}
}

// TestSessionControllerRenameRoundTrip drives session.rename through the bridge:
// the typed reply updates the live title, the catalog summary seeds it first,
// and the sidebar row switches to the committed title without a catalog reread.
func TestSessionControllerRenameRoundTrip(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	daemon.setRename(json.RawMessage(`{"title":"Renamed","titleSource":"manual","revision":2}`))
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	summary := titleSummary("Fake session", appserver.TitleSourceAuto, 1)
	seeded := controller.Title(summary)
	if !seeded.Known || seeded.Text == nil || *seeded.Text != "Fake session" || seeded.Revision != 1 {
		t.Fatalf("seeded title = %#v", seeded)
	}
	if got := controller.State(summary).Sessions[0].Title; got != "Fake session" {
		t.Fatalf("row title = %q, want the catalog title", got)
	}

	next := "Renamed"
	result, err := controller.Rename(ctx, &next)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if result.Title == nil || *result.Title != "Renamed" || result.TitleSource == nil || *result.TitleSource != appserver.TitleSourceManual || result.Revision != 2 {
		t.Fatalf("result = %#v", result)
	}
	params := daemon.lastParams("session.rename")
	if params["sessionId"] != "s1" || params["title"] != "Renamed" {
		t.Fatalf("rename params = %#v", params)
	}
	title := controller.Title(summary)
	if !title.Known || title.Text == nil || *title.Text != "Renamed" || title.Revision != 2 {
		t.Fatalf("title after rename = %#v", title)
	}
	if got := controller.State(summary).Sessions[0].Title; got != "Renamed" {
		t.Fatalf("row title = %q, want the committed title", got)
	}

	// Clearing sends an explicit null and removes the row title; the stale
	// revision-1 summary must not resurrect it.
	daemon.setRename(json.RawMessage(`{"title":null,"titleSource":null,"revision":3}`))
	cleared, err := controller.Rename(ctx, nil)
	if err != nil {
		t.Fatalf("Rename clear: %v", err)
	}
	if cleared.Title != nil || cleared.TitleSource != nil || cleared.Revision != 3 {
		t.Fatalf("clear result = %#v", cleared)
	}
	params = daemon.lastParams("session.rename")
	rawTitle, present := params["title"]
	if !present || rawTitle != nil {
		t.Fatalf("clear params title = %#v present=%v, want explicit null", rawTitle, present)
	}
	title = controller.Title(summary)
	if !title.Known || title.Text != nil || title.Source != nil || title.Revision != 3 {
		t.Fatalf("title after clear = %#v", title)
	}
	if got := controller.State(summary).Sessions[0].Title; got != "s1" {
		t.Fatalf("row title after clear = %q, want the id fallback", got)
	}
}

// TestSessionControllerRenameUnsupported reports a daemon without the method as
// a disabled capability entry: no local title, no retry.
func TestSessionControllerRenameUnsupported(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	daemon.fail("session.rename", unknownMethodFault())
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	next := "x"
	_, err = controller.Rename(ctx, &next)
	if !errors.Is(err, appserver.ErrCapabilityUnavailable) {
		t.Fatalf("error = %v, want capability unavailable", err)
	}
	var mutation *appserver.MutationError
	if !errors.As(err, &mutation) || !mutation.Unsupported() {
		t.Fatalf("error = %#v, want an unsupported mutation", err)
	}
	if got := daemon.callCount("session.rename"); got != 1 {
		t.Fatalf("session.rename calls = %d, want 1 and no retry", got)
	}
	if title := controller.Title(appserver.SessionSummary{SessionID: "s1"}); title.Known {
		t.Fatalf("title = %#v after an unsupported rename, want unknown", title)
	}
}

// TestSessionControllerRenameUnknownResult keeps a lost reply observable: the
// caller gets an unknown result with session.summaries as the read-back, and the
// rename is never resent.
func TestSessionControllerRenameUnknownResult(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	release := daemon.holdRename()
	daemon.setRename(json.RawMessage(`{"title":"later","titleSource":"manual","revision":1}`))
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	callCtx, cancel := context.WithCancel(ctx)
	errs := make(chan error, 1)
	go func() {
		next := "later"
		_, err := controller.Rename(callCtx, &next)
		errs <- err
	}()
	waitFor(t, func() bool { return daemon.callCount("session.rename") == 1 }, "rename request")
	cancel()
	release()

	select {
	case err := <-errs:
		if !errors.Is(err, appserver.ErrUnknownResult) {
			t.Fatalf("error = %v, want unknown result", err)
		}
		var unknown *appserver.UnknownResultError
		if !errors.As(err, &unknown) || unknown.ReadBack != "session.summaries" {
			t.Fatalf("error = %#v, want session.summaries read-back", err)
		}
	case <-time.After(testWait):
		t.Fatal("Rename did not finish after cancellation")
	}
	if got := daemon.callCount("session.rename"); got != 1 {
		t.Fatalf("session.rename calls = %d, want 1 and no retry", got)
	}
	if title := controller.Title(appserver.SessionSummary{SessionID: "s1"}); title.Known {
		t.Fatalf("title = %#v after an unknown result, want unknown", title)
	}
}

// TestSessionControllerCloseWaitsForRename pins the controller read lock: a
// rename in flight owns the session pair, so Close cannot detach under it.
func TestSessionControllerCloseWaitsForRename(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	release := daemon.holdRename()
	daemon.setRename(json.RawMessage(`{"title":"held","titleSource":"manual","revision":1}`))
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}

	renameErr := make(chan error, 1)
	go func() {
		next := "held"
		_, err := controller.Rename(ctx, &next)
		renameErr <- err
	}()
	waitFor(t, func() bool { return daemon.callCount("session.rename") == 1 }, "rename request")

	closeErr := make(chan error, 1)
	go func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), testWait)
		defer cancel()
		closeErr <- controller.Close(closeCtx)
	}()
	select {
	case err := <-closeErr:
		t.Fatalf("Close returned while a rename held the session: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	release()
	select {
	case err := <-renameErr:
		if err != nil {
			t.Fatalf("Rename: %v", err)
		}
	case <-time.After(testWait):
		t.Fatal("Rename did not finish after release")
	}
	select {
	case err := <-closeErr:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(testWait):
		t.Fatal("Close did not finish after the rename completed")
	}
}
