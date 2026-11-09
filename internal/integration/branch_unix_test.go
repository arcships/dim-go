//go:build unix

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/session"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
)

// startBranchBridge starts a fake daemon that advertises session.branches with
// the given directory, plus a started supervisor linked to it.
func startBranchBridge(t *testing.T, directory json.RawMessage) (*fakeDaemon, *supervisor.Supervisor, context.Context) {
	t.Helper()
	return startBranchBridgeWithFeatures(t, directory, "session.catalog", "session.history", "session.branches")
}

// startBranchBridgeWithFeatures is startBranchBridge with an explicit feature
// list, so a test can open a session without the branch directory capability.
func startBranchBridgeWithFeatures(t *testing.T, directory json.RawMessage, features ...string) (*fakeDaemon, *supervisor.Supervisor, context.Context) {
	t.Helper()
	home := fakeHome(t)
	link, err := supervisor.ResolveLink(home)
	if err != nil {
		t.Fatalf("resolve link: %v", err)
	}
	daemon := startFakeDaemon(t, link.Endpoint.Address)
	daemon.setFeatures(features...)
	daemon.setBranches(directory)
	sup, err := supervisor.New(supervisor.Config{ProductHome: home})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	ctx, cancel := testContext(t)
	t.Cleanup(func() {
		cancel()
		closeCtx, closeCancel := context.WithTimeout(context.Background(), testWait)
		defer closeCancel()
		_, _ = sup.Close(closeCtx)
	})
	if _, err := sup.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return daemon, sup, ctx
}

// TestControllerBranchSubmitStopAndLookupPassThrough drives the branch-aware
// mutations over the fake daemon: a submit, a stop and a read-back name the
// branch the window shows, the idempotency key survives, and an empty branch is
// refused before any request is sent.
func TestControllerBranchSubmitStopAndLookupPassThrough(t *testing.T) {
	directory := json.RawMessage(`{"branches":[
		{"branchId":"main","state":"open","head":2},
		{"branchId":"side","parentBranchId":"main","state":"open","head":2}
	]}`)
	daemon, sup, ctx := startBranchBridge(t, directory)

	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	// A missing branch id is refused before the bridge touches the daemon: the
	// branch is not something the integration layer may guess.
	if _, err := controller.SubmitBranch(ctx, "", "hello"); err == nil {
		t.Fatal("SubmitBranch without a branch should fail")
	}
	if _, _, err := controller.SubmitBranchWithID(ctx, "", "hello"); err == nil {
		t.Fatal("SubmitBranchWithID without a branch should fail")
	}
	if _, err := controller.SubmitBranch(nil, "side", "hello"); err == nil {
		t.Fatal("SubmitBranch without a context should fail")
	}
	if err := controller.StopBranch(ctx, ""); err == nil {
		t.Fatal("StopBranch without a branch should fail")
	}
	if err := controller.StopBranch(nil, "side"); err == nil {
		t.Fatal("StopBranch without a context should fail")
	}
	if _, err := controller.LookupInputBranch(ctx, "", "sub-1"); err == nil {
		t.Fatal("LookupInputBranch without a branch should fail")
	}
	if _, err := controller.LookupInputBranch(ctx, "side", ""); err == nil {
		t.Fatal("LookupInputBranch without a submission id should fail")
	}
	if got := daemon.callCount("session.submit") + daemon.callCount("session.stop") + daemon.callCount("session.input_lookup"); got != 0 {
		t.Fatalf("rejected calls reached the daemon: %d", got)
	}

	// A branch submit names the branch and carries the submission id.
	result, err := controller.SubmitBranch(ctx, "side", "hello side")
	if err != nil {
		t.Fatalf("SubmitBranch: %v", err)
	}
	if result.Event.ID != "submitted" {
		t.Fatalf("submit result = %#v", result)
	}
	params := daemon.lastParams("session.submit")
	if params["sessionId"] != "s1" || params["branchId"] != "side" {
		t.Fatalf("session.submit params = %#v, want s1 on side", params)
	}
	if id, _ := params["submissionId"].(string); id == "" {
		t.Fatalf("session.submit params lost the idempotency key: %#v", params)
	}

	_, submissionID, err := controller.SubmitBranchWithID(ctx, "side", "hello again")
	if err != nil {
		t.Fatalf("SubmitBranchWithID: %v", err)
	}
	if submissionID == "" {
		t.Fatal("SubmitBranchWithID lost the idempotency key")
	}
	params = daemon.lastParams("session.submit")
	if params["branchId"] != "side" || params["submissionId"] != submissionID {
		t.Fatalf("session.submit params = %#v, want side with %q", params, submissionID)
	}

	// The read-back names the same branch and never resends the submission.
	daemon.setInputLookup(json.RawMessage(`{"status":"accepted","event":{"id":"e3","sequence":3,"create_at":1700000000002,"type":"input","name":"message","data":{"text":"hello again"}}}`))
	lookup, err := controller.LookupInputBranch(ctx, "side", submissionID)
	if err != nil {
		t.Fatalf("LookupInputBranch: %v", err)
	}
	if !lookup.Accepted() {
		t.Fatalf("lookup result = %#v, want accepted", lookup)
	}
	params = daemon.lastParams("session.input_lookup")
	if params["sessionId"] != "s1" || params["branchId"] != "side" || params["submissionId"] != submissionID {
		t.Fatalf("session.input_lookup params = %#v", params)
	}
	if got := daemon.callCount("session.submit"); got != 2 {
		t.Fatalf("session.submit calls = %d, want 2 (the read-back never resends)", got)
	}

	// A stop names the shown branch instead of the session's default one.
	if err := controller.StopBranch(ctx, "side"); err != nil {
		t.Fatalf("StopBranch: %v", err)
	}
	params = daemon.lastParams("session.stop")
	if params["sessionId"] != "s1" || params["branchId"] != "side" {
		t.Fatalf("session.stop params = %#v, want s1 on side", params)
	}
}

// TestControllerBranchPassThrough drives OpenBranch/RefreshBranchDirectory/
// EndBranch over the fake daemon: the branch read must name the branchId, stay
// idempotent, and never invent branch rows the daemon did not report.
func TestControllerBranchPassThrough(t *testing.T) {
	directory := json.RawMessage(`{"branches":[
		{"branchId":"main","state":"open","head":2},
		{"branchId":"side","parentBranchId":"main","state":"open","head":2}
	]}`)
	daemon, sup, ctx := startBranchBridge(t, directory)

	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	if dir := controller.BranchDirectory(); len(dir.Branches) != 2 || dir.Branches[1].BranchID != "side" {
		t.Fatalf("branch directory = %#v", dir)
	}

	result, err := controller.OpenBranch(ctx, "side")
	if err != nil {
		t.Fatalf("OpenBranch: %v", err)
	}
	if result.BranchID != "side" || result.AlreadyLoaded || result.Head != 2 {
		t.Fatalf("open result = %#v", result)
	}
	params := daemon.lastParams("session.history")
	if params["branchId"] != "side" || params["sessionId"] != "s1" {
		t.Fatalf("session.history params = %#v", params)
	}
	if messages := controller.Session.BranchSnapshot("side").Messages; len(messages) != 1 ||
		messages[0].Content != "hello from the fake daemon" {
		t.Fatalf("side messages = %#v", messages)
	}

	// Idempotent: reopening a loaded branch sends no request.
	historyCalls := daemon.callCount("session.history")
	again, err := controller.OpenBranch(ctx, "side")
	if err != nil {
		t.Fatalf("second OpenBranch: %v", err)
	}
	if !again.AlreadyLoaded {
		t.Fatalf("second open result = %#v", again)
	}
	if got := daemon.callCount("session.history"); got != historyCalls {
		t.Fatalf("session.history calls = %d after a loaded reopen, want %d", got, historyCalls)
	}

	// An unknown branch is reported, never rendered as a local branch row.
	if _, err := controller.OpenBranch(ctx, "ghost"); !errors.Is(err, session.ErrUnknownBranch) {
		t.Fatalf("OpenBranch(ghost) error = %v, want ErrUnknownBranch", err)
	}
	summary := appserver.SessionSummary{SessionID: "s1"}
	if state := controller.StateForBranch(summary, "ghost"); state.SelectedBranch != appserver.MainBranchID {
		t.Fatalf("SelectedBranch = %q, want the main fallback", state.SelectedBranch)
	}

	// Refresh replaces the snapshot with what the daemon now reports.
	daemon.setBranches(json.RawMessage(`{"branches":[
		{"branchId":"main","state":"open","head":2},
		{"branchId":"side","parentBranchId":"main","state":"ended","head":2,"endEvent":3}
	]}`))
	branchesCalls := daemon.callCount("session.branches")
	if err := controller.RefreshBranchDirectory(ctx); err != nil {
		t.Fatalf("RefreshBranchDirectory: %v", err)
	}
	if got := daemon.callCount("session.branches"); got != branchesCalls+1 {
		t.Fatalf("session.branches calls = %d, want %d", got, branchesCalls+1)
	}
	dir := controller.BranchDirectory()
	if len(dir.Branches) != 2 || dir.Branches[1].State != "ended" || dir.Branches[1].EndEvent == nil {
		t.Fatalf("refreshed directory = %#v", dir)
	}
	if state := controller.StateForBranch(summary, "side"); state.SelectedBranch != "side" {
		t.Fatalf("SelectedBranch = %q, want side", state.SelectedBranch)
	}

	// EndBranch only asks the App Server; the local facts stay until the daemon
	// reports the new state.
	outcome, err := controller.EndBranch(ctx, "side")
	if err != nil {
		t.Fatalf("EndBranch: %v", err)
	}
	if outcome != appserver.BranchEndOutcomeEnded {
		t.Fatalf("outcome = %q, want ended", outcome)
	}
	endParams := daemon.lastParams("session.branch_end")
	if endParams["sessionId"] != "s1" || endParams["branchId"] != "side" {
		t.Fatalf("session.branch_end params = %#v", endParams)
	}
	if dir := controller.BranchDirectory(); len(dir.Branches) != 2 {
		t.Fatalf("directory after branch_end = %#v", dir)
	}
	if _, err := controller.EndBranch(ctx, appserver.MainBranchID); !errors.Is(err, session.ErrMainBranchEnd) {
		t.Fatalf("EndBranch(main) error = %v, want ErrMainBranchEnd", err)
	}
}

// TestControllerBranchCreatePassThrough drives session.branch_create over the
// fake daemon: the cut point, parent branch and prompt are what the window
// resolved, the reply's branch is returned, and an unknown result is reported
// once instead of being resent.
func TestControllerBranchCreatePassThrough(t *testing.T) {
	directory := json.RawMessage(`{"branches":[
		{"branchId":"main","state":"open","head":2}
	]}`)
	daemon, sup, ctx := startBranchBridge(t, directory)
	daemon.setBranchCreate(json.RawMessage(`{"branch":{"branchId":"side","parentBranchId":"main","branchPoint":4,"createEvent":5}}`))

	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	result, err := controller.BranchCreate(ctx, appserver.MainBranchID, 4, "why?")
	if err != nil {
		t.Fatalf("BranchCreate: %v", err)
	}
	if result.Branch == nil || result.Branch.BranchID != "side" {
		t.Fatalf("created branch = %#v", result.Branch)
	}
	params := daemon.lastParams("session.branch_create")
	if params["sessionId"] != "s1" || params["parentBranchId"] != appserver.MainBranchID {
		t.Fatalf("session.branch_create params = %#v", params)
	}
	if at, _ := params["at"].(float64); at != 4 {
		t.Fatalf("session.branch_create at = %#v, want 4", params["at"])
	}
	if params["prompt"] != "why?" {
		t.Fatalf("session.branch_create prompt = %#v", params["prompt"])
	}
	if calls := daemon.callCount("session.branch_create"); calls != 1 {
		t.Fatalf("session.branch_create calls = %d, want one", calls)
	}

	// A reply without a branch id is an unknown result, and the write is not
	// resent.
	daemon.setBranchCreate(json.RawMessage(`{}`))
	if _, err := controller.BranchCreate(ctx, appserver.MainBranchID, 4, "again"); err == nil {
		t.Fatal("BranchCreate accepted a reply without a branch id")
	}
	if calls := daemon.callCount("session.branch_create"); calls != 2 {
		t.Fatalf("session.branch_create calls = %d, want exactly two (never resent)", calls)
	}
}

// TestControllerBranchCreateRequiresTheDirectoryCapability checks the gate: a
// session without session.branches cannot present a created line, so the write
// is refused before the daemon sees it.
func TestControllerBranchCreateRequiresTheDirectoryCapability(t *testing.T) {
	daemon, sup, ctx := startBranchBridgeWithFeatures(t, json.RawMessage(`{"branches":[]}`), "session.catalog", "session.history")
	daemon.setBranchCreate(json.RawMessage(`{"branch":{"branchId":"side"}}`))

	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	_, err = controller.BranchCreate(ctx, appserver.MainBranchID, 4, "prompt")
	var capability *appserver.CapabilityError
	if !errors.As(err, &capability) || capability.Feature != "session.branches" {
		t.Fatalf("BranchCreate error = %v, want a session.branches capability error", err)
	}
	if calls := daemon.callCount("session.branch_create"); calls != 0 {
		t.Fatalf("session.branch_create calls = %d without the capability", calls)
	}
}
