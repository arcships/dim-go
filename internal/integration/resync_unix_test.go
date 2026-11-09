//go:build unix

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// branchDirectory is the fake daemon's session.branches reply for the resync
// tests: main is loaded by OpenSession, side is loaded only by an explicit
// branch read.
func branchDirectory() json.RawMessage {
	return json.RawMessage(`{"branches":[
		{"branchId":"main","state":"open","head":2},
		{"branchId":"side","parentBranchId":"main","state":"open","head":2}
	]}`)
}

// TestOpenBranchRetriesReplayGapReadOnce pins the contract's replay-gap rule on
// the branch-history read: session.history is read-only, so one
// subscription_changed may force one re-read that succeeds.
func TestOpenBranchRetriesReplayGapReadOnce(t *testing.T) {
	daemon, sup, ctx := startBranchBridge(t, branchDirectory())
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	initialHistory := daemon.callCount("session.history")

	daemon.fail("session.history", subscriptionChangedFault())
	result, err := controller.OpenBranch(ctx, "side")
	if err != nil {
		t.Fatalf("OpenBranch after one replay gap: %v", err)
	}
	if result.BranchID != "side" || result.AlreadyLoaded || result.Head != 2 {
		t.Fatalf("open result = %#v", result)
	}
	if got := daemon.callCount("session.history"); got != initialHistory+2 {
		t.Fatalf("session.history calls = %d, want %d (initial + one retry)", got, initialHistory+2)
	}
	if got := daemon.callCount("session.attach"); got != 1 {
		t.Fatalf("session.attach calls = %d, want 1 (the branch read reuses its stream)", got)
	}
	if !controller.Session.BranchLoaded("side") {
		t.Fatal("branch was not marked loaded after the successful retry")
	}
	if messages := controller.Session.BranchSnapshot("side").Messages; len(messages) != 1 ||
		messages[0].Content != "hello from the fake daemon" {
		t.Fatalf("side messages = %#v", messages)
	}
}

// TestOpenBranchFailsClosedOnConsecutiveReplayGaps proves the branch-history
// retry is bounded: a second replay gap is surfaced as a WIP replay gap and the
// branch stays unloaded.
func TestOpenBranchFailsClosedOnConsecutiveReplayGaps(t *testing.T) {
	daemon, sup, ctx := startBranchBridge(t, branchDirectory())
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	initialHistory := daemon.callCount("session.history")

	daemon.fail("session.history", subscriptionChangedFault(), subscriptionChangedFault())
	_, err = controller.OpenBranch(ctx, "side")
	if !errors.Is(err, ErrReplayGap) {
		t.Fatalf("OpenBranch error = %v, want %v", err, ErrReplayGap)
	}
	var gap *ReplayGapError
	if !errors.As(err, &gap) || !gap.ReadOnly || !gap.Retried {
		t.Fatalf("error = %#v, want a retried read-only ReplayGapError", err)
	}
	if !appserver.IsAppError(err, appserver.AppErrorSubscriptionChanged) {
		t.Fatalf("error = %v, want the daemon subscription_changed preserved", err)
	}
	if got := daemon.callCount("session.history"); got != initialHistory+2 {
		t.Fatalf("session.history calls = %d, want exactly %d attempts", got, initialHistory+2)
	}
	if controller.Session.BranchLoaded("side") {
		t.Fatal("a failed branch read marked the branch loaded")
	}
	if messages := controller.Session.BranchSnapshot("side").Messages; len(messages) != 0 {
		t.Fatalf("failed branch read created local rows: %#v", messages)
	}
}

// TestRefreshBranchDirectoryRetriesReplayGapReadOnce covers the directory read:
// session.branches is read-only and gets the same single retry.
func TestRefreshBranchDirectoryRetriesReplayGapReadOnce(t *testing.T) {
	daemon, sup, ctx := startBranchBridge(t, branchDirectory())
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	initialBranches := daemon.callCount("session.branches")

	daemon.fail("session.branches", subscriptionChangedFault())
	if err := controller.RefreshBranchDirectory(ctx); err != nil {
		t.Fatalf("RefreshBranchDirectory after one replay gap: %v", err)
	}
	if got := daemon.callCount("session.branches"); got != initialBranches+2 {
		t.Fatalf("session.branches calls = %d, want %d", got, initialBranches+2)
	}
	if dir := controller.BranchDirectory(); len(dir.Branches) != 2 || dir.Branches[1].BranchID != "side" {
		t.Fatalf("refreshed directory = %#v", dir)
	}
}

// TestRefreshBranchDirectoryFailsClosedOnConsecutiveReplayGaps keeps the
// directory read retry bounded and leaves the previous daemon snapshot intact.
func TestRefreshBranchDirectoryFailsClosedOnConsecutiveReplayGaps(t *testing.T) {
	daemon, sup, ctx := startBranchBridge(t, branchDirectory())
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	initialBranches := daemon.callCount("session.branches")

	daemon.fail("session.branches", subscriptionChangedFault(), subscriptionChangedFault())
	err = controller.RefreshBranchDirectory(ctx)
	if !errors.Is(err, ErrReplayGap) {
		t.Fatalf("RefreshBranchDirectory error = %v, want %v", err, ErrReplayGap)
	}
	var gap *ReplayGapError
	if !errors.As(err, &gap) || !gap.ReadOnly || !gap.Retried {
		t.Fatalf("error = %#v, want a retried read-only ReplayGapError", err)
	}
	if got := daemon.callCount("session.branches"); got != initialBranches+2 {
		t.Fatalf("session.branches calls = %d, want exactly %d attempts", got, initialBranches+2)
	}
	if dir := controller.BranchDirectory(); len(dir.Branches) != 2 {
		t.Fatalf("failed refresh replaced the daemon snapshot: %#v", dir)
	}
}

// TestReattachFailsClosedOnConsecutiveReplayGaps keeps the re-attach retry
// bounded: two replay gaps during attach+history fail closed instead of
// reopening attachments until one happens to succeed.
func TestReattachFailsClosedOnConsecutiveReplayGaps(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	oldConn, oldSession := controller.Conn, controller.Session
	oldConn.Close()
	waitSessionDone(t, oldSession)
	initialAttach := daemon.callCount("session.attach")

	daemon.fail("session.attach", subscriptionChangedFault(), subscriptionChangedFault())
	err = controller.Reattach(ctx)
	if !errors.Is(err, ErrReplayGap) {
		t.Fatalf("Reattach error = %v, want %v", err, ErrReplayGap)
	}
	var gap *ReplayGapError
	if !errors.As(err, &gap) || !gap.ReadOnly || !gap.Retried {
		t.Fatalf("error = %#v, want a retried read-only ReplayGapError", err)
	}
	if got := daemon.callCount("session.attach"); got != initialAttach+2 {
		t.Fatalf("session.attach calls = %d, want exactly %d attempts", got, initialAttach+2)
	}
	if controller.Conn != oldConn || controller.Session != oldSession {
		t.Fatal("fail-closed Reattach mutated the controller")
	}
	if _, ok := controller.LastReattach(); ok {
		t.Fatal("a failed Reattach recorded a successful result")
	}
}

// TestReattachDiscardsFreshStreamWhenGenerationAdvancesMidFlight pins the
// superseded-instance rule: when another hello advances the daemon generation
// while a re-attach is in flight, the fresh attachment belongs to an older
// instance and is discarded instead of installed.
func TestReattachDiscardsFreshStreamWhenGenerationAdvancesMidFlight(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	oldConn, oldSession := controller.Conn, controller.Session
	oldConn.Close()
	waitSessionDone(t, oldSession)
	// The closed old attachment was released, so only the control connection
	// remains tracked.
	if got := sup.ActiveConnections(); got != 1 {
		t.Fatalf("active connections after old stream close = %d, want 1", got)
	}

	gate := daemon.hold("session.attach")
	defer gate.Release()
	errCh := make(chan error, 1)
	go func() { errCh <- controller.Reattach(ctx) }()

	// The re-attach has sent session.attach but its reply is held. Advancing the
	// daemon instance now makes that attachment a superseded generation.
	<-gate.entered
	daemon.setInstance("integration-fake-2")
	observed, err := sup.OpenStream(ctx)
	if err != nil {
		t.Fatalf("OpenStream for the restarted instance: %v", err)
	}
	if observed.Generation() != oldConn.Generation()+1 {
		t.Fatalf("observed generation = %d, want %d", observed.Generation(), oldConn.Generation()+1)
	}
	observed.Close()
	gate.Release()

	err = <-errCh
	if !errors.Is(err, ErrDaemonRestarted) {
		t.Fatalf("Reattach error = %v, want %v", err, ErrDaemonRestarted)
	}
	if controller.Conn != oldConn || controller.Session != oldSession {
		t.Fatal("superseded Reattach installed the fresh attachment")
	}
	if _, ok := controller.LastReattach(); ok {
		t.Fatal("superseded Reattach recorded a successful result")
	}
	if got := sup.ActiveConnections(); got != 1 {
		t.Fatalf("active connections = %d after discarding the superseded stream, want 1", got)
	}
}

// TestSessionClosedRejectsRetryWithoutRequests pins session.closed as terminal
// for every resync/read path: after the daemon closed the session, nothing
// re-attaches and no read is retried, even when the next reply would be another
// replay gap.
func TestSessionClosedRejectsRetryWithoutRequests(t *testing.T) {
	daemon, sup, ctx := startBranchBridge(t, branchDirectory())
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)
	daemon.emit("session.closed", appserver.ClosedNotice{SessionID: "s1", Reason: "finished"})
	waitSessionDone(t, controller.Session)
	if closed, ok := controller.Session.Closed(); !ok || closed.Reason != "finished" {
		t.Fatalf("closed notice = %#v/%v, want reason finished", closed, ok)
	}
	hellos := daemon.callCount("hello")
	attaches := daemon.callCount("session.attach")
	histories := daemon.callCount("session.history")
	branches := daemon.callCount("session.branches")

	// If anything retried, these faults would be consumed and the call counts
	// would move. None may reach the daemon.
	daemon.fail("session.attach", subscriptionChangedFault(), subscriptionChangedFault())
	daemon.fail("session.history", subscriptionChangedFault(), subscriptionChangedFault())
	daemon.fail("session.branches", subscriptionChangedFault(), subscriptionChangedFault())

	if err := controller.Reattach(ctx); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Reattach after session.closed error = %v, want %v", err, ErrSessionClosed)
	}
	if _, err := controller.OpenBranch(ctx, "side"); !errors.Is(err, appserver.ErrSessionClosed) {
		t.Fatalf("OpenBranch after session.closed error = %v, want the terminal closed error", err)
	}
	if err := controller.RefreshBranchDirectory(ctx); err == nil {
		t.Fatal("RefreshBranchDirectory after session.closed unexpectedly succeeded")
	}

	if got := daemon.callCount("hello"); got != hellos {
		t.Fatalf("hello calls = %d, want no re-dial after session.closed (%d)", got, hellos)
	}
	if got := daemon.callCount("session.attach"); got != attaches {
		t.Fatalf("session.attach calls = %d, want no retry after session.closed (%d)", got, attaches)
	}
	if got := daemon.callCount("session.history"); got != histories {
		t.Fatalf("session.history calls = %d, want no retry after session.closed (%d)", got, histories)
	}
	if got := daemon.callCount("session.branches"); got != branches {
		t.Fatalf("session.branches calls = %d, want no retry after session.closed (%d)", got, branches)
	}
	if state := controller.StateForBranch(appserver.SessionSummary{SessionID: "s1"}, "main"); !state.SessionClosed || state.SessionClosedReason != "finished" {
		t.Fatalf("state = closed %v reason %q, want finished", state.SessionClosed, state.SessionClosedReason)
	}
}

// TestSubmitUnknownResultIsNotResent drives a real in-flight write whose reply
// arrives after the caller's context ended: the result is unknown, the
// idempotency key is returned, and the mutation is never resent.
func TestSubmitUnknownResultIsNotResent(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	gate := daemon.hold("session.submit")
	defer gate.Release()
	submitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	type submitOutcome struct {
		result       appserver.SubmitResult
		submissionID string
		err          error
	}
	out := make(chan submitOutcome, 1)
	go func() {
		result, submissionID, err := controller.SubmitWithID(submitCtx, "hello again")
		out <- submitOutcome{result: result, submissionID: submissionID, err: err}
	}()

	// The request has been received and its reply is held; ending the context
	// now makes the outcome unknown without a resend.
	<-gate.entered
	cancel()
	got := <-out
	gate.Release()

	if !errors.Is(got.err, appserver.ErrUnknownResult) {
		t.Fatalf("SubmitWithID error = %v, want %v", got.err, appserver.ErrUnknownResult)
	}
	if errors.Is(got.err, ErrReplayGap) {
		t.Fatalf("unknown result was misclassified as a replay gap: %v", got.err)
	}
	if got.submissionID == "" {
		t.Fatal("SubmitWithID lost the idempotency key on an unknown result")
	}
	if got.result.Event.ID != "" {
		t.Fatalf("unknown result carried a fabricated event: %#v", got.result)
	}
	if calls := daemon.callCount("session.submit"); calls != 1 {
		t.Fatalf("session.submit calls = %d, want 1 (never resent)", calls)
	}

	// The idempotency key is the read-back handle, not a resend token.
	if _, err := controller.LookupInput(ctx, "main", got.submissionID); err == nil {
		t.Fatal("LookupInput unexpectedly succeeded without a scripted reply")
	}
	if calls := daemon.callCount("session.submit"); calls != 1 {
		t.Fatalf("session.submit calls = %d after read-back, want 1", calls)
	}
}

// TestDeleteSubscriptionChangedFailsClosed fills the last mutation gap in the
// bridge: session.delete is a write, so a replay gap is reported as fail-closed
// and the delete is never sent twice.
func TestDeleteSubscriptionChangedFailsClosed(t *testing.T) {
	daemon, sup, ctx := startBridge(t)
	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	daemon.fail("session.delete", subscriptionChangedFault(), subscriptionChangedFault())
	err = controller.Delete(ctx)
	if !errors.Is(err, ErrReplayGap) {
		t.Fatalf("Delete error = %v, want %v", err, ErrReplayGap)
	}
	var gap *ReplayGapError
	if !errors.As(err, &gap) || gap.ReadOnly {
		t.Fatalf("error = %#v, want a fail-closed write ReplayGapError", err)
	}
	if !appserver.IsAppError(err, appserver.AppErrorSubscriptionChanged) {
		t.Fatalf("error = %v, want the daemon subscription_changed preserved", err)
	}
	if got := daemon.callCount("session.delete"); got != 1 {
		t.Fatalf("session.delete calls = %d, want 1 (no replay)", got)
	}
}

// TestRetryReadOnceHonoursCancelledContext keeps the contract's recovery budget
// tied to a live request: a cancelled context is not a license for a second
// attempt.
func TestRetryReadOnceHonoursCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	attempts := 0
	err := retryReadOnce(ctx, "session.history", func() error {
		attempts++
		return subscriptionChangedFault()
	})
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1 after a cancelled context", attempts)
	}
	if !appserver.IsAppError(err, appserver.AppErrorSubscriptionChanged) {
		t.Fatalf("error = %v, want the raw replay gap for the abandoned attempt", err)
	}
}
