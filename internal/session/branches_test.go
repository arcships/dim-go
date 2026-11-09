package session

import (
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// branchTestDirectory is a two-branch session.branches reply: main plus one
// host line cut from it.
func branchTestDirectory(mainHead, sideHead int64) appserver.BranchesResult {
	return appserver.BranchesResult{Branches: []appserver.BranchInfo{
		{BranchID: appserver.MainBranchID, State: "open", Head: mainHead},
		{BranchID: "side", ParentBranchID: appserver.MainBranchID, State: "open", Head: sideHead},
	}}
}

func branchEndedFault() *fakeRPCError {
	return &fakeRPCError{Code: -32000, Message: "branch ended", Data: json.RawMessage(`{"appError":"branch_ended"}`)}
}

func TestOpenBranchLoadsRequestedBranch(t *testing.T) {
	sideHead := int64(5)
	var sideCalls int
	_, _, s := openTestSession(t, fakeConfig{
		Attach:   appserver.AttachResult{Sequence: int64Ptr(2)},
		Branches: branchTestDirectory(2, sideHead),
		History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
			switch params.BranchID {
			case appserver.MainBranchID:
				if params.Through == nil || *params.Through != 2 {
					t.Fatalf("main history through = %#v, want 2", params.Through)
				}
				return historyPage(t, nil, 2, 2, true, nil, nil), nil
			case "side":
				sideCalls++
				if params.Through == nil {
					t.Fatalf("side history through is missing: %#v", params)
				}
				if *params.Through != sideHead {
					// The compatibility scan walks back to the branch start.
					return historyPage(t, nil, *params.Through, *params.Through, true, nil, nil), nil
				}
				return historyPage(t, []appserver.RawEvent{
					userEventWithID("side-e4", 4, "side prompt"),
					assistantEventWithID("side-e5", 5, "t-side", "r-side", "side reply"),
				}, sideHead, sideHead, true, nil, nil), nil
			default:
				t.Fatalf("history for unexpected branch %q", params.BranchID)
				return nil, invalidParams()
			}
		},
	}, []string{"session.history", "session.branches"}, OpenOptions{})

	result, err := s.OpenBranch(testContext(t), "side")
	if err != nil {
		t.Fatalf("OpenBranch: %v", err)
	}
	if result.BranchID != "side" || result.AlreadyLoaded {
		t.Fatalf("open result = %#v", result)
	}
	if result.Head != sideHead || result.History.Through != sideHead {
		t.Fatalf("open head/through = %d/%d, want %d", result.Head, result.History.Through, sideHead)
	}
	if result.History.Mode != HistoryModeCompatibility {
		t.Fatalf("history mode = %q, want compatibility", result.History.Mode)
	}
	if !s.BranchLoaded("side") {
		t.Fatal("branch not marked loaded after OpenBranch")
	}
	if got := s.Through("side"); got != sideHead {
		t.Fatalf("side head = %d, want %d", got, sideHead)
	}
	if view := s.View("side"); view.History.Through != sideHead || view.Head != sideHead {
		t.Fatalf("branch view = %#v", view)
	}
	messages := s.BranchSnapshot("side").Messages
	if len(messages) != 2 || messages[0].Content != "side prompt" || messages[1].Content != "side reply" {
		t.Fatalf("side messages = %#v", messages)
	}
	// Opening a branch must not fold its events into another branch.
	if main := s.Snapshot().Messages; len(main) != 0 {
		t.Fatalf("main messages = %#v, want empty", main)
	}
	if sideCalls == 0 {
		t.Fatal("OpenBranch did not read session.history for the branch")
	}

	// Idempotent: a loaded branch returns its stored status without a request.
	before := sideCalls
	again, err := s.OpenBranch(testContext(t), "side")
	if err != nil {
		t.Fatalf("second OpenBranch: %v", err)
	}
	if !again.AlreadyLoaded || again.Head != sideHead || again.History.Through != sideHead {
		t.Fatalf("second open result = %#v", again)
	}
	if sideCalls != before {
		t.Fatalf("side history calls = %d after a loaded reopen, want %d", sideCalls, before)
	}

	main, err := s.OpenBranch(testContext(t), appserver.MainBranchID)
	if err != nil {
		t.Fatalf("OpenBranch(main): %v", err)
	}
	if !main.AlreadyLoaded {
		t.Fatalf("main open result = %#v, want already loaded", main)
	}
}

func TestOpenBranchLatestTurnUsesBranchHead(t *testing.T) {
	sideHead := int64(6)
	_, _, s := openTestSession(t, fakeConfig{
		Attach:   appserver.AttachResult{Sequence: int64Ptr(0)},
		Branches: branchTestDirectory(0, sideHead),
		History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
			switch params.BranchID {
			case appserver.MainBranchID:
				return historyPage(t, nil, 0, 0, true, nil, nil), nil
			case "side":
				if params.LatestTurn == nil || !*params.LatestTurn {
					t.Fatalf("side latestTurn params = %#v", params)
				}
				if params.Through == nil || *params.Through != sideHead {
					t.Fatalf("side through = %#v, want %d", params.Through, sideHead)
				}
				return historyPage(t, []appserver.RawEvent{
					userEventWithID("side-e4", 4, "latest prompt"),
					assistantEventWithID("side-e6", 6, "t-side", "r-side", "latest reply"),
				}, sideHead, sideHead, true, int64Ptr(5), nil), nil
			default:
				t.Fatalf("history for unexpected branch %q", params.BranchID)
				return nil, invalidParams()
			}
		},
	}, []string{"session.history", "session.history.latestTurn", "session.branches"}, OpenOptions{})

	result, err := s.OpenBranch(testContext(t), "side")
	if err != nil {
		t.Fatalf("OpenBranch: %v", err)
	}
	if result.History.Mode != HistoryModeLatestTurn || result.History.TurnStart == nil || *result.History.TurnStart != 5 {
		t.Fatalf("history status = %#v", result.History)
	}
	if result.History.Through != sideHead || !result.History.HasOlder {
		t.Fatalf("history window = %#v", result.History)
	}
	if messages := s.BranchSnapshot("side").Messages; len(messages) != 2 || messages[1].Content != "latest reply" {
		t.Fatalf("side messages = %#v", messages)
	}
}

func TestOpenBranchRejectsUnknownBranch(t *testing.T) {
	_, _, s := openTestSession(t, fakeConfig{
		Attach:   appserver.AttachResult{Sequence: int64Ptr(2)},
		Branches: branchTestDirectory(2, 2),
		History:  emptyHistory(t, 2),
	}, []string{"session.history", "session.branches"}, OpenOptions{})

	_, err := s.OpenBranch(testContext(t), "ghost")
	if !errors.Is(err, ErrUnknownBranch) {
		t.Fatalf("OpenBranch error = %v, want ErrUnknownBranch", err)
	}
	var unknown *UnknownBranchError
	if !errors.As(err, &unknown) || unknown.BranchID != "ghost" || unknown.SessionID != "s1" {
		t.Fatalf("unknown branch error = %#v", err)
	}
	if s.BranchLoaded("ghost") {
		t.Fatal("unknown branch was marked loaded")
	}
	if messages := s.BranchSnapshot("ghost").Messages; len(messages) != 0 {
		t.Fatalf("unknown branch projection = %#v, want empty", messages)
	}
	for _, branch := range s.Branches() {
		if branch == "ghost" {
			t.Fatal("unknown branch was added to the observed branch list")
		}
	}
	if _, err := s.OpenBranch(testContext(t), ""); err == nil {
		t.Fatal("empty branch id should fail")
	}
}

func TestOpenBranchWithoutDirectoryNeedsObservedBranch(t *testing.T) {
	sideEvent := assistantEventWithID("side-e2", 2, "t-side", "r-side", "observed side")
	_, _, s := openTestSession(t, fakeConfig{
		Attach: appserver.AttachResult{Sequence: int64Ptr(2)},
		OnAttach: func(server *fakeServer) {
			server.emit("session.event", appserver.EventNotice{SessionID: "s1", BranchID: "side", Event: sideEvent})
		},
		History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
			switch params.BranchID {
			case appserver.MainBranchID:
				return historyPage(t, nil, 2, 2, true, nil, nil), nil
			case "side":
				if params.Through == nil {
					t.Fatalf("side through is missing: %#v", params)
				}
				if *params.Through == 2 {
					return historyPage(t, []appserver.RawEvent{sideEvent}, 2, 2, true, nil, nil), nil
				}
				return historyPage(t, nil, *params.Through, *params.Through, true, nil, nil), nil
			default:
				t.Fatalf("history for unexpected branch %q", params.BranchID)
				return nil, invalidParams()
			}
		},
	}, []string{"session.history"}, OpenOptions{})
	waitFor(t, func() bool { return len(s.BranchSnapshot("side").Messages) == 1 }, "observed side branch")

	if status := s.BranchDirectoryStatus(); status.Available {
		t.Fatalf("branch directory status = %#v, want unavailable", status)
	}
	result, err := s.OpenBranch(testContext(t), "side")
	if err != nil {
		t.Fatalf("OpenBranch on an observed branch: %v", err)
	}
	if result.AlreadyLoaded || result.Head != 2 {
		t.Fatalf("open result = %#v", result)
	}
	// Replaying the same sequence is a no-op, not a duplicated message.
	if messages := s.BranchSnapshot("side").Messages; len(messages) != 1 {
		t.Fatalf("side messages = %#v", messages)
	}

	_, err = s.OpenBranch(testContext(t), "ghost")
	if !errors.Is(err, ErrUnknownBranch) {
		t.Fatalf("OpenBranch error = %v, want ErrUnknownBranch", err)
	}
	if !strings.Contains(err.Error(), "session.branches is unavailable") {
		t.Fatalf("unknown branch reason = %q", err.Error())
	}
}

func TestRefreshBranchDirectoryReplacesSnapshot(t *testing.T) {
	var calls atomic.Uint64
	_, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(2)},
		History: emptyHistory(t, 2),
		BranchesFn: func(appserver.BranchesParams) (any, *fakeRPCError) {
			if calls.Add(1) == 1 {
				return branchTestDirectory(2, 2), nil
			}
			return appserver.BranchesResult{Branches: []appserver.BranchInfo{
				{BranchID: appserver.MainBranchID, State: "open", Head: 2},
				{BranchID: "side", ParentBranchID: appserver.MainBranchID, State: "ended", Head: 7, EndEvent: int64Ptr(8)},
			}}, nil
		},
	}, []string{"session.history", "session.branches"}, OpenOptions{})

	if dir := s.BranchDirectory(); len(dir.Branches) != 2 || dir.Branches[1].State != "open" {
		t.Fatalf("initial directory = %#v", dir)
	}
	if err := s.RefreshBranchDirectory(testContext(t)); err != nil {
		t.Fatalf("RefreshBranchDirectory: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("session.branches calls = %d, want 2", calls.Load())
	}
	dir := s.BranchDirectory()
	if len(dir.Branches) != 2 || dir.Branches[1].State != "ended" || dir.Branches[1].Head != 7 || dir.Branches[1].EndEvent == nil {
		t.Fatalf("refreshed directory = %#v", dir)
	}
	if status := s.BranchDirectoryStatus(); !status.Available || status.Reason != "" {
		t.Fatalf("branch directory status = %#v", status)
	}
}

func TestRefreshBranchDirectoryFailureKeepsPreviousSnapshot(t *testing.T) {
	var calls atomic.Uint64
	_, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(2)},
		History: emptyHistory(t, 2),
		BranchesFn: func(appserver.BranchesParams) (any, *fakeRPCError) {
			if calls.Add(1) == 1 {
				return branchTestDirectory(2, 2), nil
			}
			return nil, unknownMethod()
		},
	}, []string{"session.history", "session.branches"}, OpenOptions{})

	if err := s.RefreshBranchDirectory(testContext(t)); err == nil {
		t.Fatal("RefreshBranchDirectory unexpectedly succeeded")
	}
	status := s.BranchDirectoryStatus()
	if status.Available || status.Reason == "" {
		t.Fatalf("branch directory status = %#v", status)
	}
	// The last successfully read directory is kept; the status records that the
	// refresh failed rather than pretending it is current.
	if dir := s.BranchDirectory(); len(dir.Branches) != 2 {
		t.Fatalf("directory after a failed refresh = %#v", dir)
	}
}

func TestRefreshBranchDirectoryWithoutCapability(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(2)},
		History: emptyHistory(t, 2),
	}, []string{"session.history"}, OpenOptions{})

	err := s.RefreshBranchDirectory(testContext(t))
	var capability *appserver.CapabilityError
	if !errors.As(err, &capability) || capability.Feature != "session.branches" {
		t.Fatalf("RefreshBranchDirectory error = %v, want a session.branches capability error", err)
	}
	status := s.BranchDirectoryStatus()
	if status.Available || status.Reason == "" {
		t.Fatalf("branch directory status = %#v", status)
	}
	if calls := server.callsFor("session.branches"); len(calls) != 0 {
		t.Fatalf("session.branches calls = %d without the capability", len(calls))
	}
}

func TestOpenBranchWithoutHistoryCapability(t *testing.T) {
	// Open itself requires session.history, so exercise the defensive gate on a
	// directly constructed view instead of inventing a session that cannot open.
	s := &Session{
		sessionID:  "s1",
		branchID:   appserver.MainBranchID,
		features:   []string{"session.branches"},
		branches:   map[string]*branchView{},
		pendingIDs: map[string]struct{}{},
	}

	_, err := s.OpenBranch(testContext(t), "side")
	var capability *appserver.CapabilityError
	if !errors.As(err, &capability) || capability.Feature != "session.history" {
		t.Fatalf("OpenBranch error = %v, want a session.history capability error", err)
	}
}

func TestEndBranchOutcomes(t *testing.T) {
	t.Run("ended", func(t *testing.T) {
		server, _, s := openTestSession(t, fakeConfig{
			Attach:    appserver.AttachResult{Sequence: int64Ptr(2)},
			Branches:  branchTestDirectory(2, 2),
			History:   emptyHistory(t, 2),
			Mutations: map[string][]fakeMutation{"session.branch_end": {{Result: json.RawMessage(`{"ignored":true}`)}}},
		}, []string{"session.history", "session.branches"}, OpenOptions{})

		outcome, err := s.EndBranch(testContext(t), "side")
		if err != nil {
			t.Fatalf("EndBranch: %v", err)
		}
		if outcome != appserver.BranchEndOutcomeEnded {
			t.Fatalf("outcome = %q, want ended", outcome)
		}
		calls := server.recorded("session.branch_end")
		if len(calls) != 1 {
			t.Fatalf("session.branch_end calls = %d, want 1", len(calls))
		}
		var params appserver.BranchEndParams
		if err := json.Unmarshal(calls[0].Params, &params); err != nil {
			t.Fatalf("decode branch_end params: %v", err)
		}
		if params.SessionID != "s1" || params.BranchID != "side" {
			t.Fatalf("branch_end params = %#v", params)
		}
		// The daemon confirms the end through session.branches/session.state; the
		// local snapshot is not deleted on the reply.
		if dir := s.BranchDirectory(); len(dir.Branches) != 2 {
			t.Fatalf("directory after branch_end = %#v", dir)
		}
	})

	t.Run("already ended", func(t *testing.T) {
		server, _, s := openTestSession(t, fakeConfig{
			Attach:    appserver.AttachResult{Sequence: int64Ptr(2)},
			Branches:  branchTestDirectory(2, 2),
			History:   emptyHistory(t, 2),
			Mutations: map[string][]fakeMutation{"session.branch_end": {{Fault: branchEndedFault()}}},
		}, []string{"session.history", "session.branches"}, OpenOptions{})

		outcome, err := s.EndBranch(testContext(t), "side")
		if err != nil {
			t.Fatalf("EndBranch already ended: %v", err)
		}
		if outcome != appserver.BranchEndOutcomeAlreadyEnded {
			t.Fatalf("outcome = %q, want already_ended", outcome)
		}
		if calls := server.recorded("session.branch_end"); len(calls) != 1 {
			t.Fatalf("session.branch_end calls = %d, want exactly one (no retry)", len(calls))
		}
	})

	t.Run("capability missing stays WIP", func(t *testing.T) {
		server, _, s := openTestSession(t, fakeConfig{
			Attach:    appserver.AttachResult{Sequence: int64Ptr(2)},
			Branches:  branchTestDirectory(2, 2),
			History:   emptyHistory(t, 2),
			Mutations: map[string][]fakeMutation{"session.branch_end": {{Fault: unknownMethod()}}},
		}, []string{"session.history", "session.branches"}, OpenOptions{})

		_, err := s.EndBranch(testContext(t), "side")
		if !errors.Is(err, appserver.ErrCapabilityUnavailable) {
			t.Fatalf("EndBranch error = %v, want ErrCapabilityUnavailable", err)
		}
		var mutationErr *appserver.MutationError
		if !errors.As(err, &mutationErr) || !mutationErr.Unsupported() {
			t.Fatalf("EndBranch error = %#v, want an unsupported MutationError", err)
		}
		if calls := server.recorded("session.branch_end"); len(calls) != 1 {
			t.Fatalf("session.branch_end calls = %d, want 1", len(calls))
		}
	})

	t.Run("main and empty branch ids are refused locally", func(t *testing.T) {
		server, _, s := openTestSession(t, fakeConfig{
			Attach:   appserver.AttachResult{Sequence: int64Ptr(2)},
			Branches: branchTestDirectory(2, 2),
			History:  emptyHistory(t, 2),
		}, []string{"session.history", "session.branches"}, OpenOptions{})

		if _, err := s.EndBranch(testContext(t), appserver.MainBranchID); !errors.Is(err, ErrMainBranchEnd) {
			t.Fatalf("EndBranch(main) error = %v, want ErrMainBranchEnd", err)
		}
		if _, err := s.EndBranch(testContext(t), ""); err == nil {
			t.Fatal("EndBranch with an empty branch id should fail")
		}
		if calls := server.recorded("session.branch_end"); len(calls) != 0 {
			t.Fatalf("session.branch_end calls = %d, want 0", len(calls))
		}
	})
}
