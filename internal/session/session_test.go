package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/projection"
)

func findError(errs []error, target error) error {
	for _, err := range errs {
		if errors.Is(err, target) {
			return err
		}
	}
	return nil
}

func TestErrorChannelCloseSerializesConcurrentReports(t *testing.T) {
	s := &Session{errs: make(chan error, 256)}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 32; j++ {
				s.report(errors.New("concurrent report"))
			}
		}()
	}
	// Repeated close attempts model a duplicate terminal notification racing the
	// drain goroutine. The channel must close exactly once and reports must never
	// panic by sending after close.
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.closeErrors()
		}()
	}
	wg.Wait()
	select {
	case _, ok := <-s.errs:
		if ok {
			for range s.errs {
			}
		}
	default:
		t.Fatal("error channel was not closed")
	}
}

func TestAdaptEventBounds(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		bad  bool
	}{
		{name: "absent", data: nil},
		{name: "null", data: []byte("null")},
		{name: "object", data: []byte(`{"x":1}`)},
		{name: "malformed", data: []byte("{"), bad: true},
		{name: "non-object", data: []byte(`[1]`), bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := AdaptEvent(appserver.RawEvent{ID: "e-1", Sequence: 1, CreateAt: 1, Type: "input", Name: "message", Data: tc.data})
			if tc.bad && !errors.Is(err, appserver.ErrProtocol) {
				t.Fatalf("error = %v, want ErrProtocol", err)
			}
			if !tc.bad && err != nil {
				t.Fatalf("error = %v", err)
			}
		})
	}
	if _, err := AdaptEvent(appserver.RawEvent{Sequence: 1}); !errors.Is(err, appserver.ErrProtocol) {
		t.Fatalf("empty id error = %v, want ErrProtocol", err)
	}
}

func TestSessionMethodsSnapshotAndCapabilityGate(t *testing.T) {
	cfg := fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
		History: emptyHistory(t, 0),
		Methods: appserver.MethodsResult{Methods: []appserver.MethodDeclaration{{Name: "session.rename", Mutating: true}}},
	}
	server, _, s := openTestSession(t, cfg, []string{"session.history", "session.methods"}, OpenOptions{})
	got := s.Methods()
	if len(got.Methods) != 1 || got.Methods[0].Name != "session.rename" || !got.Methods[0].Mutating {
		t.Fatalf("methods = %#v", got)
	}
	got.Methods[0].Name = "mutated"
	if again := s.Methods(); again.Methods[0].Name != "session.rename" {
		t.Fatalf("methods accessor did not copy: %#v", again)
	}
	found := false
	for _, call := range server.callsSnapshot() {
		if call.Method == "session.methods" {
			found = true
		}
	}
	if !found {
		t.Fatal("session.methods was not requested with capability")
	}
}

func TestSessionMethodsErrorFailsOpen(t *testing.T) {
	_, client := newFakeServer(t, fakeConfig{Attach: appserver.AttachResult{Sequence: int64Ptr(0)}, History: emptyHistory(t, 0), MethodsErr: unknownMethod()})
	_, err := OpenWithOptions(testContext(t), client, []string{"session.history", "session.methods"}, "s1", OpenOptions{})
	if err == nil {
		t.Fatal("Open unexpectedly succeeded after session.methods error")
	}
}

func TestSessionMethodsCapabilityAbsentDoesNotRequest(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{Attach: appserver.AttachResult{Sequence: int64Ptr(0)}, History: emptyHistory(t, 0), Methods: appserver.MethodsResult{Methods: []appserver.MethodDeclaration{{Name: "session.rename"}}}}, []string{"session.history"}, OpenOptions{})
	if got := s.Methods(); len(got.Methods) != 0 {
		t.Fatalf("methods without capability = %#v", got)
	}
	for _, call := range server.callsSnapshot() {
		if call.Method == "session.methods" {
			t.Fatal("session.methods requested without capability")
		}
	}
}

func TestSessionKeysSnapshotCapabilityAndFailure(t *testing.T) {
	t.Run("reads and copies the capability-gated snapshot", func(t *testing.T) {
		through := int64(5)
		server, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(through)},
			History: emptyHistory(t, through),
			Keys: appserver.KeysResult{
				Through: &through,
				Entries: []appserver.SessionKeyEntry{{Key: "k", Sequence: 1, InputID: "i", Text: "v"}},
			},
		}, []string{"session.history", "session.keys"}, OpenOptions{})

		got, ok := s.Keys()
		if !ok || got.Through == nil || *got.Through != through || len(got.Entries) != 1 || got.Entries[0].Text != "v" {
			t.Fatalf("keys = %#v, loaded=%v", got, ok)
		}
		got.Entries[0].Text = "mutated"
		again, ok := s.Keys()
		if !ok || again.Entries[0].Text != "v" {
			t.Fatalf("keys accessor did not copy entries: %#v", again)
		}
		status := s.KeysStatus()
		if !status.Available || status.Reason != "" {
			t.Fatalf("keys status = %#v", status)
		}
		calls := server.callsFor("session.keys")
		if len(calls) != 1 {
			t.Fatalf("session.keys calls = %d, want 1", len(calls))
		}
		var params appserver.KeysParams
		if err := json.Unmarshal(calls[0].Params, &params); err != nil {
			t.Fatalf("keys params: %v", err)
		}
		if params.SessionID != "s1" || params.BranchID != "main" || params.Through == nil || *params.Through != through {
			t.Fatalf("keys params = %#v", params)
		}
	})

	t.Run("absent capability is not requested", func(t *testing.T) {
		server, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
			History: emptyHistory(t, 0),
		}, []string{"session.history"}, OpenOptions{})

		if _, ok := s.Keys(); ok {
			t.Fatal("keys unexpectedly loaded without capability")
		}
		status := s.KeysStatus()
		if status.Available || !strings.Contains(status.Reason, "does not advertise") {
			t.Fatalf("keys status = %#v", status)
		}
		if calls := server.callsFor("session.keys"); len(calls) != 0 {
			t.Fatalf("session.keys calls = %d without capability", len(calls))
		}
	})

	t.Run("read failure degrades without failing open", func(t *testing.T) {
		_, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
			History: emptyHistory(t, 0),
			KeysErr: unknownMethod(),
		}, []string{"session.history", "session.keys"}, OpenOptions{})

		if !s.Ready() {
			t.Fatal("session is not ready after a keys read failure")
		}
		if _, ok := s.Keys(); ok {
			t.Fatal("keys unexpectedly loaded after a read failure")
		}
		status := s.KeysStatus()
		if status.Available || status.Reason == "" {
			t.Fatalf("keys status = %#v", status)
		}
	})

	t.Run("malformed snapshot fails closed", func(t *testing.T) {
		_, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
			History: emptyHistory(t, 0),
			Keys:    map[string]any{"through": 0, "entries": nil},
		}, []string{"session.history", "session.keys"}, OpenOptions{})

		if _, ok := s.Keys(); ok {
			t.Fatal("keys unexpectedly loaded from a malformed snapshot")
		}
		status := s.KeysStatus()
		if status.Available || status.Reason == "" {
			t.Fatalf("keys status = %#v", status)
		}
	})
}

func TestBranchDirectorySnapshotFallbackAndObservedBranches(t *testing.T) {
	sideText := assistantEventWithID("side-2", 2, "t-side", "r-side", "side text")
	directory := appserver.BranchesResult{Branches: []appserver.BranchInfo{
		{BranchID: "main", State: "open", Head: 2},
		{BranchID: "side", ParentBranchID: "main", State: "open", Head: 2},
	}}
	onAttach := func(server *fakeServer) {
		server.emit("session.event", appserver.EventNotice{SessionID: "s1", BranchID: "side", Event: sideText})
	}

	t.Run("reads and copies the capability-gated directory", func(t *testing.T) {
		server, _, s := openTestSession(t, fakeConfig{
			Attach:   appserver.AttachResult{Sequence: int64Ptr(2)},
			OnAttach: onAttach,
			History:  emptyHistory(t, 2),
			Branches: directory,
		}, []string{"session.history", "session.branches"}, OpenOptions{})

		waitFor(t, func() bool { return len(s.BranchSnapshot("side").Messages) == 1 }, "observed side branch")
		got := s.BranchDirectory()
		if len(got.Branches) != 2 || got.Branches[1].BranchID != "side" {
			t.Fatalf("branch directory = %#v", got)
		}
		got.Branches[0].BranchID = "mutated"
		if again := s.BranchDirectory(); again.Branches[0].BranchID != "main" {
			t.Fatalf("branch directory accessor did not copy: %#v", again)
		}
		status := s.BranchDirectoryStatus()
		if !status.Available || status.Reason != "" {
			t.Fatalf("branch directory status = %#v", status)
		}
		calls := server.callsFor("session.branches")
		if len(calls) != 1 {
			t.Fatalf("session.branches calls = %d, want 1", len(calls))
		}
	})

	t.Run("absent capability keeps observed branches only", func(t *testing.T) {
		server, _, s := openTestSession(t, fakeConfig{
			Attach:   appserver.AttachResult{Sequence: int64Ptr(2)},
			OnAttach: onAttach,
			History:  emptyHistory(t, 2),
		}, []string{"session.history"}, OpenOptions{})

		waitFor(t, func() bool { return len(s.BranchSnapshot("side").Messages) == 1 }, "observed side branch")
		if dir := s.BranchDirectory(); len(dir.Branches) != 0 {
			t.Fatalf("branch directory = %#v, want empty", dir)
		}
		status := s.BranchDirectoryStatus()
		if status.Available || !strings.Contains(status.Reason, "does not advertise") {
			t.Fatalf("branch directory status = %#v", status)
		}
		branches := s.Branches()
		if len(branches) != 2 || branches[0] != "main" || branches[1] != "side" {
			t.Fatalf("observed branches = %v, want [main side]", branches)
		}
		if calls := server.callsFor("session.branches"); len(calls) != 0 {
			t.Fatalf("session.branches calls = %d without capability", len(calls))
		}
	})

	t.Run("read failure falls back to observed branches", func(t *testing.T) {
		_, _, s := openTestSession(t, fakeConfig{
			Attach:      appserver.AttachResult{Sequence: int64Ptr(2)},
			OnAttach:    onAttach,
			History:     emptyHistory(t, 2),
			BranchesErr: unknownMethod(),
		}, []string{"session.history", "session.branches"}, OpenOptions{})

		waitFor(t, func() bool { return len(s.BranchSnapshot("side").Messages) == 1 }, "observed side branch")
		if dir := s.BranchDirectory(); len(dir.Branches) != 0 {
			t.Fatalf("branch directory = %#v after failure, want empty", dir)
		}
		status := s.BranchDirectoryStatus()
		if status.Available || status.Reason == "" {
			t.Fatalf("branch directory status = %#v", status)
		}
		if branches := s.Branches(); len(branches) != 2 {
			t.Fatalf("observed branches = %v, want main+side", branches)
		}
	})
}

func TestLatestTurnEmptySessionWithoutTurnStart(t *testing.T) {
	cfg := fakeConfig{
		Attach: appserver.AttachResult{Sequence: int64Ptr(0)},
		History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
			if params.LatestTurn == nil || !*params.LatestTurn {
				return nil, invalidParams()
			}
			return historyPage(t, nil, 0, 0, true, nil, nil), nil
		},
	}
	server, _, s := openTestSession(t, cfg, []string{"session.history", "session.history.latestTurn"}, OpenOptions{})
	status := s.History()
	if status.Mode != HistoryModeLatestTurn || status.Pages != 1 || status.TurnStart != nil || status.Through != 0 {
		t.Fatalf("history status = %#v, want empty latest-turn window", status)
	}
	if snapshot := s.Snapshot(); len(snapshot.Messages) != 0 {
		t.Fatalf("empty session messages = %#v", snapshot.Messages)
	}
	view := s.View(appserver.MainBranchID)
	if !view.Ready || view.Head != 0 || view.History.Through != 0 {
		t.Fatalf("empty session view = %#v, want ready zero-head window", view)
	}
	calls := server.callsSnapshot()
	latest, compatibility := 0, 0
	for _, call := range calls {
		switch call.Method {
		case "session.history":
			var params appserver.HistoryParams
			if err := json.Unmarshal(call.Params, &params); err != nil {
				t.Fatalf("decode history params: %v", err)
			}
			if params.LatestTurn != nil && *params.LatestTurn {
				latest++
			} else {
				compatibility++
			}
		}
	}
	if latest != 1 || compatibility != 0 {
		t.Fatalf("history calls latest=%d compatibility=%d, want one latest-turn only", latest, compatibility)
	}
}

func TestLatestTurnMissingTurnStartNonEmptyFails(t *testing.T) {
	cfg := fakeConfig{
		Attach: appserver.AttachResult{Sequence: int64Ptr(1)},
		History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
			return historyPage(t, []appserver.RawEvent{userEvent(1, "hello")}, 1, 1, true, nil, nil), nil
		},
	}
	_, client := newFakeServer(t, cfg)
	_, err := OpenWithOptions(testContext(t), client, []string{"session.history", "session.history.latestTurn"}, "s1", OpenOptions{})
	if err == nil || !errors.Is(err, appserver.ErrProtocol) {
		t.Fatalf("Open error = %v, want protocol error", err)
	}
}

func TestOpenMultiPageHistoryAttachOverlapAndFreshHead(t *testing.T) {
	const head = int64(520)
	all := make([]appserver.RawEvent, 0, head)
	for sequence := int64(1); sequence <= head; sequence++ {
		all = append(all, userEvent(sequence, fmt.Sprintf("message-%d", sequence)))
	}
	live := userEvent(head+1, "live")
	historyCalls := 0
	cfg := fakeConfig{
		Attach: appserver.AttachResult{Sequence: int64Ptr(head)},
		OnAttach: func(server *fakeServer) {
			// Replay an already-known sequence and the newest live fact before the
			// attach reply. The pending buffer must dedupe the overlap.
			server.emit("session.event", appserver.EventNotice{SessionID: "s1", BranchID: "main", Event: all[head-1]})
			server.emit("session.event", appserver.EventNotice{SessionID: "s1", BranchID: "main", Event: live})
		},
		History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
			historyCalls++
			if params.Through == nil {
				return nil, invalidParams()
			}
			switch historyCalls {
			case 1:
				if params.After == nil || *params.After != 8 || *params.Through != head {
					t.Fatalf("first history params = %#v", params)
				}
				return historyPage(t, all[9:], head, head, false, nil, nil), nil
			case 2:
				if params.After == nil || *params.After != 0 || *params.Through != 9 {
					t.Fatalf("second history params = %#v", params)
				}
				return historyPage(t, all[:9], 9, 9, true, nil, nil), nil
			default:
				t.Fatalf("unexpected history call %#v", params)
				return nil, invalidParams()
			}
		},
	}
	_, _, s := openTestSession(t, cfg, []string{"session.history"}, OpenOptions{})

	waitFor(t, func() bool {
		return s.Head() == head+1 && len(s.Snapshot().Messages) == int(head)+1
	}, "history and overlapping live events")
	if historyCalls != 2 {
		t.Fatalf("history calls = %d, want 2", historyCalls)
	}
	if !s.Ready() {
		t.Fatal("session is not ready after open")
	}
	if s.AttachHead() != head {
		t.Fatalf("attach head = %d, want %d", s.AttachHead(), head)
	}
	if got := s.Head(); got != head+1 {
		t.Fatalf("current head = %d, want %d", got, head+1)
	}
	if got := s.Through("main"); got != head+1 {
		t.Fatalf("through = %d, want %d", got, head+1)
	}
	status := s.History()
	if status.Mode != HistoryModeCompatibility || status.Through != head || status.Degraded || status.HasOlder || status.Pages != 2 {
		t.Fatalf("history status = %#v", status)
	}
	snapshot := s.Snapshot()
	if len(snapshot.Messages) != int(head)+1 {
		t.Fatalf("messages = %d, want %d", len(snapshot.Messages), int(head)+1)
	}
	if snapshot.Messages[len(snapshot.Messages)-1].Content != "live" {
		t.Fatalf("last message = %#v, want the live event", snapshot.Messages[len(snapshot.Messages)-1])
	}
	view := s.View("main")
	if view.Head != head+1 || !view.Ready {
		t.Fatalf("view = %#v", view)
	}
}

func TestLatestTurnPagesAndThrough(t *testing.T) {
	head := int64(100)
	latestCalls := 0
	cfg := fakeConfig{
		Attach: appserver.AttachResult{Sequence: int64Ptr(head)},
		History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
			if params.LatestTurn == nil || !*params.LatestTurn {
				t.Fatalf("latest-turn feature fell back to compatibility: %#v", params)
			}
			if params.Limit == nil || *params.Limit != latestTurnPageLimit {
				t.Fatalf("latest-turn limit = %#v, want %d", params.Limit, latestTurnPageLimit)
			}
			if params.Through == nil || *params.Through != head {
				t.Fatalf("latest-turn through = %#v, want %d", params.Through, head)
			}
			latestCalls++
			switch latestCalls {
			case 1:
				if params.Before != nil {
					t.Fatalf("first latest-turn page carried before = %d", *params.Before)
				}
				events := make([]appserver.RawEvent, 0, 11)
				for sequence := int64(90); sequence <= 100; sequence++ {
					events = append(events, assistantEvent(sequence, "t1", "r1", "x"))
				}
				return historyPage(t, events, head, head, false, int64Ptr(80), int64Ptr(89)), nil
			case 2:
				if params.Before == nil || *params.Before != 89 {
					t.Fatalf("second latest-turn before = %#v, want 89", params.Before)
				}
				events := []appserver.RawEvent{userEvent(79, "prompt")}
				for sequence := int64(80); sequence <= 89; sequence++ {
					events = append(events, assistantEvent(sequence, "t1", "r1", "x"))
				}
				return historyPage(t, events, head, head, true, int64Ptr(80), nil), nil
			default:
				t.Fatalf("unexpected latest-turn call %#v", params)
				return nil, invalidParams()
			}
		},
	}
	_, _, s := openTestSession(t, cfg, []string{"session.history", "session.history.latestTurn"}, OpenOptions{})

	if latestCalls != 2 {
		t.Fatalf("latest-turn calls = %d, want 2", latestCalls)
	}
	status := s.History()
	if status.Mode != HistoryModeLatestTurn || status.Pages != 2 || status.Degraded || status.Fallback {
		t.Fatalf("history status = %#v", status)
	}
	if status.TurnStart == nil || *status.TurnStart != 80 {
		t.Fatalf("turnStart = %#v, want 80", status.TurnStart)
	}
	if !status.HasOlder {
		t.Fatal("expected older turns to be reported")
	}
	snapshot := s.Snapshot()
	if len(snapshot.Messages) != 2 {
		t.Fatalf("messages = %#v, want one user and one assistant", snapshot.Messages)
	}
	if snapshot.Messages[0].ID != "e-79" || snapshot.Messages[0].Content != "prompt" {
		t.Fatalf("first message = %#v", snapshot.Messages[0])
	}
	if snapshot.Messages[1].ID != "a:main:t1:r1" {
		t.Fatalf("assistant id = %q, want a:main:t1:r1", snapshot.Messages[1].ID)
	}
	if !strings.Contains(snapshot.Messages[1].Content, "x") {
		t.Fatalf("assistant content = %q", snapshot.Messages[1].Content)
	}
}

func TestLatestTurnDriftFallsBackToCompatibility(t *testing.T) {
	head := int64(100)
	latestCalls := 0
	compatCalls := 0
	cfg := fakeConfig{
		Attach: appserver.AttachResult{Sequence: int64Ptr(head)},
		History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
			if params.LatestTurn != nil && *params.LatestTurn {
				latestCalls++
				switch latestCalls {
				case 1:
					events := make([]appserver.RawEvent, 0, 21)
					for sequence := int64(80); sequence <= 100; sequence++ {
						events = append(events, assistantEvent(sequence, "t1", "r1", "x"))
					}
					return historyPage(t, events, head, head, false, int64Ptr(80), int64Ptr(79)), nil
				case 2:
					events := make([]appserver.RawEvent, 0, 10)
					for sequence := int64(70); sequence <= 79; sequence++ {
						events = append(events, ignoredEvent(sequence))
					}
					return historyPage(t, events, head, head, true, int64Ptr(70), nil), nil
				}
			}
			compatCalls++
			if params.Through == nil {
				return nil, invalidParams()
			}
			switch compatCalls {
			case 1:
				if params.After == nil || *params.After != 0 || *params.Through != 100 {
					t.Fatalf("first compatibility params = %#v", params)
				}
				events := []appserver.RawEvent{turnStartEvent(80, "t1")}
				for sequence := int64(81); sequence <= 100; sequence++ {
					events = append(events, assistantEvent(sequence, "t1", "r1", "x"))
				}
				return historyPage(t, events, 100, 100, false, nil, nil), nil
			case 2:
				if params.After == nil || *params.After != 0 || *params.Through != 79 {
					t.Fatalf("second compatibility params = %#v", params)
				}
				events := []appserver.RawEvent{userEvent(79, "prompt")}
				for sequence := int64(70); sequence <= 78; sequence++ {
					events = append(events, ignoredEvent(sequence))
				}
				return historyPage(t, events, 79, 79, true, nil, nil), nil
			default:
				t.Fatalf("unexpected compatibility call %#v", params)
				return nil, invalidParams()
			}
		},
	}
	_, _, s := openTestSession(t, cfg, []string{"session.history", "session.history.latestTurn"}, OpenOptions{})

	if latestCalls != 2 {
		t.Fatalf("latest-turn calls = %d, want 2", latestCalls)
	}
	status := s.History()
	if status.Mode != HistoryModeCompatibility || !status.Fallback {
		t.Fatalf("history status = %#v, want compatibility fallback", status)
	}
	if status.TurnStart == nil || *status.TurnStart != 80 {
		t.Fatalf("turnStart = %#v, want 80", status.TurnStart)
	}
	if !status.HasOlder {
		t.Fatal("expected older history after the prompting input")
	}
}

func TestHistoryTooLargeDegradesInsteadOfFailingOpen(t *testing.T) {
	t.Run("compatibility", func(t *testing.T) {
		_, _, s := openTestSession(t, fakeConfig{
			Attach: appserver.AttachResult{Sequence: int64Ptr(10)},
			History: func(appserver.HistoryParams) (any, *fakeRPCError) {
				return nil, historyTooLarge()
			},
		}, []string{"session.history"}, OpenOptions{})

		status := s.History()
		if !status.Degraded || status.Reason != "history_too_large" || !status.HasOlder {
			t.Fatalf("history status = %#v", status)
		}
		if len(s.Snapshot().Messages) != 0 {
			t.Fatalf("degraded window must be empty, got %#v", s.Snapshot().Messages)
		}
		if got := s.Head(); got != 10 {
			t.Fatalf("head = %d, want attach head 10", got)
		}
	})

	t.Run("latestTurn", func(t *testing.T) {
		calls := 0
		_, _, s := openTestSession(t, fakeConfig{
			Attach: appserver.AttachResult{Sequence: int64Ptr(10)},
			History: func(appserver.HistoryParams) (any, *fakeRPCError) {
				calls++
				return nil, historyTooLarge()
			},
		}, []string{"session.history", "session.history.latestTurn"}, OpenOptions{})

		if calls != 1 {
			t.Fatalf("history calls = %d, want no retry", calls)
		}
		status := s.History()
		if status.Mode != HistoryModeLatestTurn || !status.Degraded || status.Reason != "history_too_large" {
			t.Fatalf("history status = %#v", status)
		}
	})
}

func TestHistoryPageCapDegradesInsteadOfFailingOpen(t *testing.T) {
	head := int64(1000)
	calls := 0
	_, _, s := openTestSession(t, fakeConfig{
		Attach: appserver.AttachResult{Sequence: int64Ptr(head)},
		History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
			if params.Through == nil || params.After == nil {
				return nil, invalidParams()
			}
			calls++
			sequence := *params.Through
			event := ignoredEvent(sequence)
			done := *params.After == 0
			return historyPage(t, []appserver.RawEvent{event}, *params.Through, *params.Through, done, nil, nil), nil
		},
	}, []string{"session.history"}, OpenOptions{})

	if calls != compatibilityPageCap {
		t.Fatalf("history calls = %d, want cap %d", calls, compatibilityPageCap)
	}
	status := s.History()
	if !status.Degraded || status.Reason != "page_cap" || !status.HasOlder {
		t.Fatalf("history status = %#v", status)
	}
	if status.Pages != compatibilityPageCap {
		t.Fatalf("history pages = %d, want %d", status.Pages, compatibilityPageCap)
	}
}

func TestBranchIsolationForEventsAndState(t *testing.T) {
	mainInput := userEventWithID("main-1", 1, "main input")
	mainText := assistantEventWithID("main-2", 2, "t-main", "r-main", "main text")
	sideText := assistantEventWithID("side-2", 2, "t-side", "r-side", "side text")
	cfg := fakeConfig{
		Attach: appserver.AttachResult{Sequence: int64Ptr(3)},
		OnAttach: func(server *fakeServer) {
			server.emit("session.event", appserver.EventNotice{SessionID: "s1", BranchID: "side", Event: sideText})
			server.emit("session.state", appserver.StateNotice{SessionID: "s1", BranchID: "main", Running: false})
			server.emit("session.state", appserver.StateNotice{SessionID: "s1", BranchID: "side", Running: true})
		},
		History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
			if params.After == nil || *params.After != 0 || params.Through == nil || *params.Through != 3 {
				t.Fatalf("history params = %#v", params)
			}
			return historyPage(t, []appserver.RawEvent{mainInput, mainText, ignoredEvent(3)}, 3, 3, true, nil, nil), nil
		},
	}
	_, _, s := openTestSession(t, cfg, []string{"session.history"}, OpenOptions{})

	waitFor(t, func() bool {
		side := s.BranchSnapshot("side")
		sideState, sideOK := s.BranchState("side")
		_, mainOK := s.State()
		return len(side.Messages) == 1 && sideOK && sideState.Running && mainOK
	}, "branch-isolated live event and state")

	mainSnapshot := s.Snapshot()
	if len(mainSnapshot.Messages) != 2 {
		t.Fatalf("main messages = %#v", mainSnapshot.Messages)
	}
	for _, message := range mainSnapshot.Messages {
		if strings.Contains(message.Content, "side") {
			t.Fatalf("side-branch text leaked into main: %#v", message)
		}
	}
	sideSnapshot := s.BranchSnapshot("side")
	if len(sideSnapshot.Messages) != 1 {
		t.Fatalf("side messages = %#v", sideSnapshot.Messages)
	}
	if sideSnapshot.Messages[0].ID != "a:side:t-side:r-side" || sideSnapshot.Messages[0].Content != "side text" {
		t.Fatalf("side message = %#v", sideSnapshot.Messages[0])
	}
	mainState, ok := s.State()
	if !ok || mainState.Running {
		t.Fatalf("main state = %#v, %v; want running=false", mainState, ok)
	}
	sideState, ok := s.BranchState("side")
	if !ok || !sideState.Running {
		t.Fatalf("side state = %#v, %v; want running=true", sideState, ok)
	}
	if s.Head() != 3 {
		t.Fatalf("main head = %d, want 3", s.Head())
	}
	if s.Through("side") != 2 {
		t.Fatalf("side through = %d, want 2", s.Through("side"))
	}
	branches := s.Branches()
	if len(branches) != 2 || branches[0] != "main" || branches[1] != "side" {
		t.Fatalf("branches = %v, want [main side]", branches)
	}
}

func TestSnapshotConcurrentWithOutOfOrderLiveEvents(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(1)},
		History: emptyHistory(t, 1),
	}, []string{"session.history"}, OpenOptions{})

	// Make the fold incremental, then invalidate it with a lower sequence while
	// readers are taking snapshots. Before the locking fix this reproduces the
	// Reducer.foldState data race under -race.
	server.emit("session.event", appserver.EventNotice{SessionID: "s1", BranchID: "main", Event: ignoredEvent(100)})
	waitFor(t, func() bool { return s.Through("main") == 100 }, "first live event")

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 200; j++ {
				_ = s.Snapshot()
				_ = s.View("main")
			}
		}()
	}
	close(start)
	for sequence := int64(99); sequence >= 1; sequence-- {
		server.emit("session.event", appserver.EventNotice{SessionID: "s1", BranchID: "main", Event: ignoredEvent(sequence)})
	}
	server.emit("session.event", appserver.EventNotice{SessionID: "s1", BranchID: "main", Event: ignoredEvent(200)})
	wg.Wait()
	waitFor(t, func() bool { return s.Through("main") == 200 }, "all live events")
}

func TestTerminalTransportLossIsObservable(t *testing.T) {
	server, client, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(5)},
		History: emptyHistory(t, 5),
	}, []string{"session.history"}, OpenOptions{})

	server.closeConn()
	waitSessionDone(t, s)
	waitClientDone(t, client)

	if s.TerminalState() != appserver.StreamStateLost || !s.Lost() {
		t.Fatalf("terminal state = %s lost=%v", s.TerminalState(), s.Lost())
	}
	if s.TerminalReason() != appserver.StreamReasonTransportClosed {
		t.Fatalf("terminal reason = %q", s.TerminalReason())
	}
	var lost *TransportLostError
	if !errors.As(s.Err(), &lost) || !errors.Is(s.Err(), ErrTransportLost) {
		t.Fatalf("terminal err = %v, want TransportLostError", s.Err())
	}
	errs := collectErrors(t, s.Errors())
	if findError(errs, ErrTransportLost) == nil {
		t.Fatalf("errors = %v, want the transport loss report", errs)
	}
	if _, ok := s.Closed(); ok {
		t.Fatal("a lost transport must not be reported as a remote close")
	}
}

func TestRemoteClosedIsTerminalWithoutResyncSignal(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(5)},
		History: emptyHistory(t, 5),
	}, []string{"session.history"}, OpenOptions{})

	server.emit("session.closed", appserver.ClosedNotice{SessionID: "s1", Reason: "done"})
	waitSessionDone(t, s)

	closed, ok := s.Closed()
	if !ok || closed.Reason != "done" {
		t.Fatalf("closed = %#v, %v", closed, ok)
	}
	if s.TerminalState() != appserver.StreamStateClosed || s.Lost() {
		t.Fatalf("terminal state = %s lost=%v", s.TerminalState(), s.Lost())
	}
	if s.Err() != nil {
		t.Fatalf("terminal err = %v, want nil for a remote close", s.Err())
	}
}

func TestCloseUsesReleaseAndDetachUsesReply(t *testing.T) {
	t.Run("Close ignores the detach reply and is idempotent", func(t *testing.T) {
		server, client, s := openTestSession(t, fakeConfig{
			Attach:    appserver.AttachResult{Sequence: int64Ptr(1)},
			History:   emptyHistory(t, 1),
			DetachErr: unknownMethod(),
		}, []string{"session.history"}, OpenOptions{})

		if err := s.Close(testContext(t)); err != nil {
			t.Fatalf("Close: %v", err)
		}
		if got := s.TerminalState(); got != appserver.StreamStateDetached {
			t.Fatalf("terminal state = %s, want detached", got)
		}
		waitClientDone(t, client)
		if calls := server.recorded("session.detach"); len(calls) != 1 {
			t.Fatalf("detach calls = %d, want 1", len(calls))
		}
		if err := s.Close(testContext(t)); err != nil {
			t.Fatalf("second Close: %v", err)
		}
		if err := s.Detach(testContext(t)); err != nil {
			t.Fatalf("Detach after Close: %v", err)
		}
		if calls := server.recorded("session.detach"); len(calls) != 1 {
			t.Fatalf("detach calls after second close = %d, want 1", len(calls))
		}
	})

	t.Run("Detach returns the reply error and still closes", func(t *testing.T) {
		server, client, s := openTestSession(t, fakeConfig{
			Attach:    appserver.AttachResult{Sequence: int64Ptr(1)},
			History:   emptyHistory(t, 1),
			DetachErr: unknownMethod(),
		}, []string{"session.history"}, OpenOptions{})

		err := s.Detach(testContext(t))
		var rpcErr *appserver.RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32601 {
			t.Fatalf("Detach error = %v, want the daemon reply", err)
		}
		if got := s.TerminalState(); got != appserver.StreamStateDetached {
			t.Fatalf("terminal state = %s, want detached", got)
		}
		waitClientDone(t, client)
		if calls := server.recorded("session.detach"); len(calls) != 1 {
			t.Fatalf("detach calls = %d, want 1", len(calls))
		}
	})

	t.Run("Release is safe from cleanup", func(t *testing.T) {
		server, client, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(1)},
			History: emptyHistory(t, 1),
		}, []string{"session.history"}, OpenOptions{})
		s.Release()
		waitSessionDone(t, s)
		waitClientDone(t, client)
		if calls := server.recorded("session.detach"); len(calls) != 1 {
			t.Fatalf("detach calls = %d, want 1", len(calls))
		}
	})
}

func TestOpenFailureClosesOwnedClient(t *testing.T) {
	t.Run("attach failure", func(t *testing.T) {
		_, client := newFakeServer(t, fakeConfig{AttachErr: invalidParams()})
		if _, err := Open(testContext(t), client, []string{"session.history"}, "s1"); err == nil {
			t.Fatal("Open succeeded for an attach failure")
		}
		waitClientDone(t, client)
	})

	t.Run("missing history capability", func(t *testing.T) {
		_, client := newFakeServer(t, fakeConfig{Attach: appserver.AttachResult{Sequence: int64Ptr(1)}})
		_, err := Open(testContext(t), client, nil, "s1")
		var capability *appserver.CapabilityError
		if !errors.As(err, &capability) || capability.Feature != "session.history" {
			t.Fatalf("Open error = %v, want CapabilityError session.history", err)
		}
		waitClientDone(t, client)
	})

	t.Run("invalid session id", func(t *testing.T) {
		_, client := newFakeServer(t, fakeConfig{})
		if _, err := Open(testContext(t), client, []string{"session.history"}, ""); err == nil {
			t.Fatal("Open succeeded with an empty session id")
		}
		waitClientDone(t, client)
	})

	t.Run("shared client is not closed by the second open", func(t *testing.T) {
		_, client, first := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(1)},
			History: emptyHistory(t, 1),
		}, []string{"session.history"}, OpenOptions{})
		if _, err := Open(testContext(t), client, []string{"session.history"}, "s1"); !errors.Is(err, appserver.ErrStreamAlreadyOpen) {
			t.Fatalf("second Open error = %v, want ErrStreamAlreadyOpen", err)
		}
		select {
		case <-client.Done():
			t.Fatal("a shared client was closed by the rejected second open")
		default:
		}
		if !first.Ready() {
			t.Fatal("first session stopped working after the rejected second open")
		}
	})
}

func TestOpenFailsWhenTransportLostDuringHistory(t *testing.T) {
	head := int64(10)
	var server *fakeServer
	cfg := fakeConfig{
		Attach: appserver.AttachResult{Sequence: int64Ptr(head)},
		History: func(appserver.HistoryParams) (any, *fakeRPCError) {
			server.closeConn()
			return historyPage(t, nil, head, head, true, nil, nil), nil
		},
	}
	var client *appserver.Client
	server, client = newFakeServer(t, cfg)
	if _, err := Open(testContext(t), client, []string{"session.history"}, "s1"); err == nil {
		t.Fatal("Open succeeded after the transport was lost during history")
	}
	waitClientDone(t, client)
}

func TestTransientApprovalModelProgressAndUnknownNotices(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(1)},
		History: emptyHistory(t, 1),
	}, []string{"session.history"}, OpenOptions{})

	server.emit("approval.request", appserver.ApprovalRequest{
		SessionID: "s1",
		BranchID:  "main",
		RequestID: "r1",
		Source:    appserver.ApprovalSource{Kind: "tool"},
		Call:      appserver.ApprovalCall{Name: "shell"},
	})
	server.emit("approval.request", appserver.ApprovalRequest{
		SessionID: "s1",
		BranchID:  "side",
		RequestID: "r2",
		Source:    appserver.ApprovalSource{Kind: "tool"},
		Call:      appserver.ApprovalCall{Name: "shell"},
	})
	server.emit("session.model.progress", appserver.ModelProgress{
		SessionID:  "s1",
		BranchID:   "main",
		Text:       "hello",
		TextOffset: 5,
	})
	server.emit("session.notice", appserver.SessionNotice{
		SessionID: "s1",
		Source:    "title",
		Kind:      "updated",
		Data:      json.RawMessage(`{"title":"new"}`),
	})
	server.emit("session.future", map[string]any{"sessionId": "s1", "payload": map[string]any{"x": 1}})

	waitFor(t, func() bool {
		main := s.PendingApprovals("main")
		side := s.PendingApprovals("side")
		transient := s.Transient()
		notices := s.SessionNotices()
		return len(main) == 1 && len(side) == 1 &&
			transient.ModelProgress != nil && transient.ModelProgress.Text == "hello" &&
			hasTransientKind(notices, "session.notice", "title:updated") &&
			hasTransientKind(notices, "session.future", "session.future")
	}, "pending approvals and transient notices")

	mainTransient := s.Transient()
	if mainTransient.ModelProgress == nil || mainTransient.ModelProgress.Text != "hello" {
		t.Fatalf("main model progress = %#v", mainTransient.ModelProgress)
	}
	if !hasTransientKind(mainTransient, "approval.request", "r1") {
		t.Fatalf("main transient notices = %#v", mainTransient.Notices)
	}
	if hasTransientKind(mainTransient, "approval.request", "r2") {
		t.Fatal("side approval leaked into the main transient view")
	}
	sideTransient := s.BranchTransient("side")
	if !hasTransientKind(sideTransient, "approval.request", "r2") {
		t.Fatalf("side transient notices = %#v", sideTransient.Notices)
	}
	notices := s.SessionNotices()
	if !hasTransientKind(notices, "session.notice", "title:updated") {
		t.Fatalf("session notices = %#v", notices.Notices)
	}
	if !hasTransientKind(notices, "session.future", "session.future") {
		t.Fatalf("unknown method not exposed: %#v", notices.Notices)
	}
	if len(s.Snapshot().Messages) != 0 {
		t.Fatalf("transient notices entered the durable timeline: %#v", s.Snapshot().Messages)
	}

	server.emit("approval.resolved", appserver.ApprovalResolved{
		SessionID: "s1",
		BranchID:  "main",
		RequestID: "r1",
		Decision:  "allow",
	})
	waitFor(t, func() bool { return len(s.PendingApprovals("main")) == 0 }, "main approval resolution")
	if len(s.PendingApprovals("side")) != 1 {
		t.Fatal("resolving a main approval removed the side approval")
	}
	server.emit("approval.cancelled", appserver.ApprovalCancelled{
		SessionID: "s1",
		BranchID:  "side",
		RequestID: "r2",
		Reason:    "cancelled",
	})
	waitFor(t, func() bool { return len(s.PendingApprovals("side")) == 0 }, "side approval cancellation")

	server.emit("session.model.progress", appserver.ModelProgress{SessionID: "s1", BranchID: "main"})
	waitFor(t, func() bool { return s.Transient().ModelProgress == nil }, "empty model progress clearing")
}

func hasTransientKind(snapshot projection.TransientSnapshot, kind, id string) bool {
	for _, notice := range snapshot.Notices {
		if notice.Kind == kind && notice.ID == id {
			return true
		}
	}
	return false
}

func TestSubmitUsesUUIDAndDoesNotResendUnknownResults(t *testing.T) {
	t.Run("successful submit sends a v4 submission id", func(t *testing.T) {
		result := map[string]any{"event": userEvent(1, "hello")}
		resultJSON, _ := json.Marshal(result)
		server, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(1)},
			History: emptyHistory(t, 1),
			Mutations: map[string][]fakeMutation{
				"session.submit": {{Result: resultJSON}},
			},
		}, []string{"session.history"}, OpenOptions{})

		if _, err := s.Submit(testContext(t), "hello"); err != nil {
			t.Fatalf("Submit: %v", err)
		}
		calls := server.recorded("session.submit")
		if len(calls) != 1 {
			t.Fatalf("submit calls = %d, want 1", len(calls))
		}
		var params appserver.SubmitParams
		if err := json.Unmarshal(calls[0].Params, &params); err != nil {
			t.Fatalf("submit params: %v", err)
		}
		if params.SessionID != "s1" || params.BranchID != "main" || params.Text != "hello" {
			t.Fatalf("submit params = %#v", params)
		}
		assertUUIDv4(t, params.SubmissionID)
	})

	t.Run("timeout is unknown and is not resent", func(t *testing.T) {
		gate := newFakeGate()
		t.Cleanup(gate.open)
		server, _, s := openTestSession(t, fakeConfig{
			Attach:       appserver.AttachResult{Sequence: int64Ptr(1)},
			History:      emptyHistory(t, 1),
			MutationGate: map[string]*fakeGate{"session.submit": gate},
			Mutations:    map[string][]fakeMutation{"session.submit": {{Result: json.RawMessage(`{}`)}}},
		}, []string{"session.history"}, OpenOptions{})

		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
		defer cancel()
		_, err := s.Submit(ctx, "hello")
		if !errors.Is(err, appserver.ErrUnknownResult) {
			t.Fatalf("Submit error = %v, want ErrUnknownResult", err)
		}
		time.Sleep(20 * time.Millisecond)
		if calls := server.recorded("session.submit"); len(calls) != 1 {
			t.Fatalf("submit calls = %d, want exactly one after a timeout", len(calls))
		}
	})

	t.Run("submission_unknown is unknown and is not resent", func(t *testing.T) {
		server, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(1)},
			History: emptyHistory(t, 1),
			Mutations: map[string][]fakeMutation{
				"session.submit": {{Fault: submissionUnknown()}},
			},
		}, []string{"session.history"}, OpenOptions{})

		_, err := s.Submit(testContext(t), "hello")
		if !errors.Is(err, appserver.ErrUnknownResult) {
			t.Fatalf("Submit error = %v, want ErrUnknownResult", err)
		}
		if calls := server.recorded("session.submit"); len(calls) != 1 {
			t.Fatalf("submit calls = %d, want exactly one", len(calls))
		}
	})

	t.Run("SubmitWithID returns the UUID for unknown-result read-back", func(t *testing.T) {
		gate := newFakeGate()
		t.Cleanup(gate.open)
		server, _, s := openTestSession(t, fakeConfig{
			Attach:       appserver.AttachResult{Sequence: int64Ptr(1)},
			History:      emptyHistory(t, 1),
			MutationGate: map[string]*fakeGate{"session.submit": gate},
			Mutations:    map[string][]fakeMutation{"session.submit": {{Result: json.RawMessage(`{}`)}}},
		}, []string{"session.history"}, OpenOptions{})

		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
		defer cancel()
		_, submissionID, err := s.SubmitWithID(ctx, "hello")
		if !errors.Is(err, appserver.ErrUnknownResult) {
			t.Fatalf("SubmitWithID error = %v, want ErrUnknownResult", err)
		}
		assertUUIDv4(t, submissionID)
		calls := server.recorded("session.submit")
		if len(calls) != 1 {
			t.Fatalf("submit calls = %d, want exactly one", len(calls))
		}
		var params appserver.SubmitParams
		if err := json.Unmarshal(calls[0].Params, &params); err != nil {
			t.Fatalf("submit params: %v", err)
		}
		if params.SubmissionID != submissionID {
			t.Fatalf("returned submission id %q != sent %q", submissionID, params.SubmissionID)
		}
	})
}

func TestLookupInputReadsBackSubmission(t *testing.T) {
	t.Run("defaults branch and returns the raw App Server result", func(t *testing.T) {
		raw := json.RawMessage(`{"inputId":"i-1","status":"queued","submissionId":"sub-1"}`)
		calls := 0
		_, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(1)},
			History: emptyHistory(t, 1),
			InputLookup: func(params InputLookupParams) (any, *fakeRPCError) {
				calls++
				if params.SessionID != "s1" || params.BranchID != "main" || params.SubmissionID != "sub-1" {
					t.Fatalf("input lookup params = %#v", params)
				}
				return raw, nil
			},
		}, []string{"session.history"}, OpenOptions{})

		got, err := s.LookupInput(testContext(t), "", "sub-1")
		if err != nil {
			t.Fatalf("LookupInput: %v", err)
		}
		if string(got) != string(raw) {
			t.Fatalf("lookup result = %s, want %s", got, raw)
		}
		if calls != 1 {
			t.Fatalf("input lookup calls = %d, want 1", calls)
		}
	})

	t.Run("preserves an explicit branch", func(t *testing.T) {
		_, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(1)},
			History: emptyHistory(t, 1),
			InputLookup: func(params InputLookupParams) (any, *fakeRPCError) {
				if params.BranchID != "side" || params.SubmissionID != "sub-2" {
					t.Fatalf("input lookup params = %#v", params)
				}
				return json.RawMessage(`{"status":"unknown"}`), nil
			},
		}, []string{"session.history"}, OpenOptions{})

		if _, err := s.LookupInput(testContext(t), "side", "sub-2"); err != nil {
			t.Fatalf("LookupInput: %v", err)
		}
	})

	t.Run("empty submission id fails before a request", func(t *testing.T) {
		called := false
		_, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(1)},
			History: emptyHistory(t, 1),
			InputLookup: func(InputLookupParams) (any, *fakeRPCError) {
				called = true
				return nil, invalidParams()
			},
		}, []string{"session.history"}, OpenOptions{})

		if _, err := s.LookupInput(testContext(t), "", ""); !errors.Is(err, appserver.ErrProtocol) {
			t.Fatalf("LookupInput error = %v, want ErrProtocol", err)
		}
		if called {
			t.Fatal("input lookup was sent for an empty submission id")
		}
	})

	t.Run("unknown_method is surfaced", func(t *testing.T) {
		_, _, s := openTestSession(t, fakeConfig{
			Attach:  appserver.AttachResult{Sequence: int64Ptr(1)},
			History: emptyHistory(t, 1),
		}, []string{"session.history"}, OpenOptions{})

		_, err := s.LookupInput(testContext(t), "", "sub-3")
		var rpcErr *appserver.RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32601 {
			t.Fatalf("LookupInput error = %v, want a daemon unknown_method", err)
		}
	})

	t.Run("reads back the submission id from an unknown submit", func(t *testing.T) {
		gate := newFakeGate()
		t.Cleanup(gate.open)
		raw := json.RawMessage(`{"inputId":"i-9","status":"queued"}`)
		var mu sync.Mutex
		submittedID := ""
		_, _, s := openTestSession(t, fakeConfig{
			Attach:       appserver.AttachResult{Sequence: int64Ptr(1)},
			History:      emptyHistory(t, 1),
			MutationGate: map[string]*fakeGate{"session.submit": gate},
			Mutations:    map[string][]fakeMutation{"session.submit": {{Result: json.RawMessage(`{}`)}}},
			InputLookup: func(params InputLookupParams) (any, *fakeRPCError) {
				mu.Lock()
				want := submittedID
				mu.Unlock()
				if params.SubmissionID != want {
					t.Fatalf("lookup submission id = %q, want %q", params.SubmissionID, want)
				}
				return raw, nil
			},
		}, []string{"session.history"}, OpenOptions{})

		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
		_, submissionID, err := s.SubmitWithID(ctx, "hello")
		cancel()
		if !errors.Is(err, appserver.ErrUnknownResult) {
			t.Fatalf("SubmitWithID error = %v, want ErrUnknownResult", err)
		}
		mu.Lock()
		submittedID = submissionID
		mu.Unlock()
		gate.open()

		got, err := s.LookupInput(testContext(t), "", submissionID)
		if err != nil {
			t.Fatalf("LookupInput: %v", err)
		}
		if string(got) != string(raw) {
			t.Fatalf("lookup result = %s, want %s", got, raw)
		}
	})
}

func TestStopAndApprovalDefaultBranch(t *testing.T) {
	server, _, s := openTestSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(1)},
		History: emptyHistory(t, 1),
		Mutations: map[string][]fakeMutation{
			"session.stop":     {{Result: json.RawMessage(`{}`)}, {Result: json.RawMessage(`{}`)}},
			"approval.respond": {{Result: json.RawMessage(`{}`)}, {Result: json.RawMessage(`{}`)}},
		},
	}, []string{"session.history"}, OpenOptions{})

	if err := s.Stop(testContext(t)); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := s.StopBranch(testContext(t), "side"); err != nil {
		t.Fatalf("StopBranch: %v", err)
	}
	if _, err := s.RespondApproval(testContext(t), appserver.ApprovalRespondParams{RequestID: "r1", Decision: appserver.DecisionAllow}); err != nil {
		t.Fatalf("RespondApproval: %v", err)
	}
	if _, err := s.RespondApproval(testContext(t), appserver.ApprovalRespondParams{BranchID: "side", RequestID: "r2", Decision: appserver.DecisionDeny}); err != nil {
		t.Fatalf("RespondApproval side: %v", err)
	}

	stops := server.recorded("session.stop")
	if len(stops) != 2 {
		t.Fatalf("stop calls = %d, want 2", len(stops))
	}
	var firstStop, secondStop appserver.StopParams
	_ = json.Unmarshal(stops[0].Params, &firstStop)
	_ = json.Unmarshal(stops[1].Params, &secondStop)
	if firstStop.BranchID != "main" || secondStop.BranchID != "side" {
		t.Fatalf("stop branches = %q, %q", firstStop.BranchID, secondStop.BranchID)
	}

	approvals := server.recorded("approval.respond")
	if len(approvals) != 2 {
		t.Fatalf("approval calls = %d, want 2", len(approvals))
	}
	var firstApproval, secondApproval appserver.ApprovalRespondParams
	_ = json.Unmarshal(approvals[0].Params, &firstApproval)
	_ = json.Unmarshal(approvals[1].Params, &secondApproval)
	if firstApproval.BranchID != "main" || secondApproval.BranchID != "side" {
		t.Fatalf("approval branches = %q, %q", firstApproval.BranchID, secondApproval.BranchID)
	}
}

func TestConcurrentCloseAndSubmit(t *testing.T) {
	gate := newFakeGate()
	server, _, s := openTestSession(t, fakeConfig{
		Attach:       appserver.AttachResult{Sequence: int64Ptr(1)},
		History:      emptyHistory(t, 1),
		MutationGate: map[string]*fakeGate{"session.submit": gate},
		Mutations:    map[string][]fakeMutation{"session.submit": {{Result: json.RawMessage(`{}`)}}},
	}, []string{"session.history"}, OpenOptions{})

	errs := make(chan error, 1)
	go func() {
		_, err := s.Submit(context.Background(), "hello")
		errs <- err
	}()
	waitFor(t, func() bool { return len(server.recorded("session.submit")) == 1 }, "submit request")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	gate.open()
	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("submit unexpectedly succeeded after close")
		}
	case <-time.After(testWait):
		t.Fatal("submit did not finish after close")
	}
	if calls := server.recorded("session.submit"); len(calls) != 1 {
		t.Fatalf("submit calls = %d, want 1", len(calls))
	}
}

func TestPendingBufferIsBounded(t *testing.T) {
	dedupe := &Session{
		sessionID:    "s1",
		pendingLimit: 10,
		pendingIDs:   map[string]struct{}{},
	}
	event := userEvent(1, "one")
	dedupe.handleEvent(appserver.EventNotice{SessionID: "s1", BranchID: "main", Event: event})
	dedupe.handleEvent(appserver.EventNotice{SessionID: "s1", BranchID: "main", Event: event})
	dedupe.mu.Lock()
	if len(dedupe.pending) != 1 {
		dedupe.mu.Unlock()
		t.Fatalf("pending = %d after a duplicate event.id, want 1", len(dedupe.pending))
	}
	dedupe.mu.Unlock()

	s := &Session{
		sessionID:    "s1",
		pendingLimit: 1,
		pendingIDs:   map[string]struct{}{},
	}
	s.handleEvent(appserver.EventNotice{SessionID: "s1", BranchID: "main", Event: userEvent(1, "one")})
	s.handleEvent(appserver.EventNotice{SessionID: "s1", BranchID: "main", Event: userEvent(2, "two")})
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.pending) != 1 {
		t.Fatalf("pending = %d, want 1", len(s.pending))
	}
	if !errors.Is(s.pendingErr, ErrPendingOverflow) {
		t.Fatalf("pending error = %v, want ErrPendingOverflow", s.pendingErr)
	}
}

func TestLatestTurnCapAndProtocolErrors(t *testing.T) {
	t.Run("page cap degrades", func(t *testing.T) {
		head := int64(1000)
		nextBefore := head - 1
		calls := 0
		_, _, s := openTestSession(t, fakeConfig{
			Attach: appserver.AttachResult{Sequence: int64Ptr(head)},
			History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
				if params.LatestTurn == nil || !*params.LatestTurn {
					t.Fatalf("expected latestTurn params, got %#v", params)
				}
				if params.Through == nil || *params.Through != head {
					t.Fatalf("through = %#v, want %d", params.Through, head)
				}
				calls++
				before := nextBefore
				nextBefore--
				return historyPage(t, []appserver.RawEvent{ignoredEvent(head)}, head, head, false, int64Ptr(1), &before), nil
			},
		}, []string{"session.history", "session.history.latestTurn"}, OpenOptions{})

		if calls != latestTurnPageCap {
			t.Fatalf("latest-turn calls = %d, want cap %d", calls, latestTurnPageCap)
		}
		status := s.History()
		if status.Mode != HistoryModeLatestTurn || !status.Degraded || status.Reason != "page_cap" || !status.HasOlder {
			t.Fatalf("history status = %#v", status)
		}
	})

	t.Run("empty session without turnStart is valid", func(t *testing.T) {
		server, client := newFakeServer(t, fakeConfig{
			Attach: appserver.AttachResult{Sequence: int64Ptr(0)},
			History: func(params appserver.HistoryParams) (any, *fakeRPCError) {
				if params.Through == nil || *params.Through != 0 {
					t.Fatalf("through = %#v, want 0", params.Through)
				}
				return historyPage(t, nil, 0, 0, true, nil, nil), nil
			},
		})
		s, err := Open(testContext(t), client, []string{"session.history", "session.history.latestTurn"}, "s1")
		if err != nil {
			t.Fatalf("Open empty session: %v", err)
		}
		t.Cleanup(func() { _ = s.Close(context.Background()) })
		if !s.Ready() || len(s.Snapshot().Messages) != 0 {
			t.Fatalf("empty session state: ready=%v snapshot=%#v", s.Ready(), s.Snapshot())
		}
		status := s.History()
		if status.Mode != HistoryModeLatestTurn || status.Through != 0 || status.TurnStart != nil || status.Pages != 1 {
			t.Fatalf("history status = %#v", status)
		}
		if calls := len(server.callsFor("session.history")); calls != 1 {
			t.Fatalf("history calls = %d, want 1", calls)
		}
	})

	t.Run("missing turnStart fails fast", func(t *testing.T) {
		head := int64(10)
		_, client := newFakeServer(t, fakeConfig{
			Attach: appserver.AttachResult{Sequence: int64Ptr(head)},
			History: func(appserver.HistoryParams) (any, *fakeRPCError) {
				return historyPage(t, nil, head, head, true, nil, nil), nil
			},
		})
		_, err := Open(testContext(t), client, []string{"session.history", "session.history.latestTurn"}, "s1")
		if !errors.Is(err, appserver.ErrProtocol) {
			t.Fatalf("Open error = %v, want ErrProtocol", err)
		}
		waitClientDone(t, client)
	})

	t.Run("missing nextBefore fails fast", func(t *testing.T) {
		head := int64(10)
		_, client := newFakeServer(t, fakeConfig{
			Attach: appserver.AttachResult{Sequence: int64Ptr(head)},
			History: func(appserver.HistoryParams) (any, *fakeRPCError) {
				return historyPage(t, nil, head, head, false, int64Ptr(1), nil), nil
			},
		})
		_, err := Open(testContext(t), client, []string{"session.history", "session.history.latestTurn"}, "s1")
		if !errors.Is(err, appserver.ErrProtocol) {
			t.Fatalf("Open error = %v, want ErrProtocol", err)
		}
		waitClientDone(t, client)
	})
}

func TestUUIDSubmissionIDsAreUniqueV4(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 1000; i++ {
		id, err := newUUID()
		if err != nil {
			t.Fatalf("newUUID: %v", err)
		}
		assertUUIDv4(t, id)
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate UUID %q", id)
		}
		seen[id] = struct{}{}
	}
}

func assertUUIDv4(t *testing.T, id string) {
	t.Helper()
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Fatalf("UUID %q has the wrong shape", id)
	}
	if id[14] != '4' {
		t.Fatalf("UUID %q is not version 4", id)
	}
	if !strings.ContainsRune("89ab", rune(id[19])) {
		t.Fatalf("UUID %q has the wrong variant", id)
	}
}
