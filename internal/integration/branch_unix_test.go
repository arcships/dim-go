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
	home := fakeHome(t)
	link, err := supervisor.ResolveLink(home)
	if err != nil {
		t.Fatalf("resolve link: %v", err)
	}
	daemon := startFakeDaemon(t, link.Endpoint.Address)
	daemon.setFeatures("session.catalog", "session.history", "session.branches")
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
