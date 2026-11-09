package session

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

func stringPtr(value string) *string { return &value }

func renameSessionConfig(t *testing.T, result string) fakeConfig {
	t.Helper()
	return fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
		History: emptyHistory(t, 0),
		Methods: appserver.MethodsResult{Methods: []appserver.MethodDeclaration{{Name: "session.rename", Mutating: true}}},
		Mutations: map[string][]fakeMutation{
			"session.rename": {{Result: json.RawMessage(result)}},
		},
	}
}

func openTitleSession(t *testing.T, cfg fakeConfig, features ...string) (*fakeServer, *Session) {
	t.Helper()
	server, _, s := openTestSession(t, cfg, append([]string{"session.history"}, features...), OpenOptions{})
	return server, s
}

func TestRenameUpdatesTitleState(t *testing.T) {
	server, s := openTitleSession(t, renameSessionConfig(t, `{"title":"Release checklist","titleSource":"manual","revision":3}`), "session.methods")

	if title := s.Title(); title.Known {
		t.Fatalf("title before rename = %#v, want unknown", title)
	}
	result, err := s.Rename(context.Background(), stringPtr("Release checklist"))
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if result.Revision != 3 || result.Title == nil || *result.Title != "Release checklist" {
		t.Fatalf("result = %#v", result)
	}
	title := s.Title()
	if !title.Known || title.Text == nil || *title.Text != "Release checklist" || title.Source == nil || *title.Source != appserver.TitleSourceManual || title.Revision != 3 {
		t.Fatalf("title = %#v", title)
	}
	// The accessor copies: mutating it must not rewrite session state.
	*title.Text = "mutated"
	if again := s.Title(); again.Text == nil || *again.Text != "Release checklist" {
		t.Fatalf("title accessor did not copy: %#v", again)
	}

	calls := server.recorded("session.rename")
	if len(calls) != 1 {
		t.Fatalf("session.rename calls = %d, want 1", len(calls))
	}
	var params appserver.RenameParams
	if err := json.Unmarshal(calls[0].Params, &params); err != nil {
		t.Fatalf("rename params: %v", err)
	}
	if params.SessionID != "s1" || params.Title == nil || *params.Title != "Release checklist" {
		t.Fatalf("rename params = %#v", params)
	}
}

func TestRenameClearKeepsNilTitle(t *testing.T) {
	server, s := openTitleSession(t, renameSessionConfig(t, `{"title":null,"titleSource":null,"revision":4}`), "session.methods")

	if _, err := s.Rename(context.Background(), nil); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	title := s.Title()
	if !title.Known || title.Text != nil || title.Source != nil || title.Revision != 4 {
		t.Fatalf("title = %#v, want a known clear at revision 4", title)
	}
	calls := server.recorded("session.rename")
	var params map[string]json.RawMessage
	if err := json.Unmarshal(calls[0].Params, &params); err != nil {
		t.Fatalf("rename params: %v", err)
	}
	if raw, present := params["title"]; !present || string(raw) != "null" {
		t.Fatalf("title param = %s present=%v, want explicit null", raw, present)
	}
}

func TestRenameNotAdvertisedStillAsksTheDaemon(t *testing.T) {
	// The captured table can be stale (a withdrawn namespace may register
	// again), so it is display state, not a write gate. The daemon decides.
	server, s := openTitleSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
		History: emptyHistory(t, 0),
		Methods: appserver.MethodsResult{Methods: []appserver.MethodDeclaration{{Name: "shell.run", Mutating: true}}},
		Mutations: map[string][]fakeMutation{
			"session.rename": {{Fault: unknownMethod()}},
		},
	}, "session.methods")

	_, err := s.Rename(context.Background(), stringPtr("x"))
	if !errors.Is(err, appserver.ErrCapabilityUnavailable) {
		t.Fatalf("error = %v, want capability unavailable", err)
	}
	var mutation *appserver.MutationError
	if !errors.As(err, &mutation) || !mutation.Unsupported() {
		t.Fatalf("error = %#v, want an unsupported mutation", err)
	}
	if calls := server.recorded("session.rename"); len(calls) != 1 {
		t.Fatalf("session.rename calls = %d, want 1 and no retry", len(calls))
	}
}

func TestRenameWithoutMethodTableAsksTheDaemon(t *testing.T) {
	// The table is unavailable, so the client cannot know whether the method
	// exists. The daemon's unknown_method is the definitive unsupported answer.
	server, s := openTitleSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
		History: emptyHistory(t, 0),
		Mutations: map[string][]fakeMutation{
			"session.rename": {{Fault: unknownMethod()}},
		},
	})

	_, err := s.Rename(context.Background(), stringPtr("x"))
	var mutation *appserver.MutationError
	if !errors.As(err, &mutation) || !mutation.Unsupported() {
		t.Fatalf("error = %v, want unsupported mutation", err)
	}
	if calls := server.recorded("session.rename"); len(calls) != 1 {
		t.Fatalf("session.rename calls = %d, want 1", len(calls))
	}
	if title := s.Title(); title.Known {
		t.Fatalf("title = %#v after a rejected rename, want unknown", title)
	}
}

func TestRenameUnknownResultIsPreservedAndNeverRetried(t *testing.T) {
	gate := newFakeGate()
	server, s := openTitleSession(t, fakeConfig{
		Attach:       appserver.AttachResult{Sequence: int64Ptr(0)},
		History:      emptyHistory(t, 0),
		MutationGate: map[string]*fakeGate{"session.rename": gate},
		Mutations: map[string][]fakeMutation{
			"session.rename": {{Result: json.RawMessage(`{"title":"later","titleSource":"manual","revision":1}`)}},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	errs := make(chan error, 1)
	go func() {
		_, err := s.Rename(ctx, stringPtr("later"))
		errs <- err
	}()
	waitFor(t, func() bool { return len(server.recorded("session.rename")) == 1 }, "rename request")
	cancel()
	gate.open()

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
	if calls := server.recorded("session.rename"); len(calls) != 1 {
		t.Fatalf("session.rename calls = %d, want 1 and no retry", len(calls))
	}
	if title := s.Title(); title.Known {
		t.Fatalf("title = %#v after an unknown result, want unknown", title)
	}
}

func TestTitleNoticeUpdatesStreamState(t *testing.T) {
	server, s := openTitleSession(t, fakeConfig{
		Attach:  appserver.AttachResult{Sequence: int64Ptr(0)},
		History: emptyHistory(t, 0),
	})

	server.emit("session.notice", appserver.SessionNotice{
		SessionID: "s1",
		Source:    appserver.NoticeSourceSessionStore,
		Kind:      appserver.NoticeKindTitle,
		Data:      json.RawMessage(`{"title":"Generated","titleSource":"auto","titleRevision":2}`),
	})
	waitFor(t, func() bool { return s.Title().Known }, "title notice")

	title := s.Title()
	if title.Text == nil || *title.Text != "Generated" || title.Source == nil || *title.Source != appserver.TitleSourceAuto || title.Revision != 2 {
		t.Fatalf("title = %#v", title)
	}

	server.emit("session.notice", appserver.SessionNotice{
		SessionID: "s1",
		Source:    appserver.NoticeSourceSessionStore,
		Kind:      appserver.NoticeKindTitle,
		Data:      json.RawMessage(`{"title":null,"titleSource":null,"titleRevision":3}`),
	})
	waitFor(t, func() bool { return s.Title().Revision == 3 }, "cleared title notice")
	if title := s.Title(); title.Text != nil || title.Source != nil {
		t.Fatalf("title after clear = %#v", title)
	}
}

func TestTitleNoticeRevisionsNeverRegress(t *testing.T) {
	s := &Session{}
	if err := s.applyTitleNotice(json.RawMessage(`{"title":"newer","titleSource":"manual","titleRevision":5}`)); err != nil {
		t.Fatalf("apply newer: %v", err)
	}
	if err := s.applyTitleNotice(json.RawMessage(`{"title":"older","titleSource":"auto","titleRevision":4}`)); err != nil {
		t.Fatalf("apply older: %v", err)
	}
	title := s.Title()
	if title.Revision != 5 || title.Text == nil || *title.Text != "newer" {
		t.Fatalf("title = %#v, want the revision-5 title", title)
	}

	// A malformed notice is reported as a protocol error by the caller and
	// never clears the observed state.
	if err := s.applyTitleNotice(json.RawMessage(`{"title":"x","titleSource":null,"titleRevision":6}`)); !errors.Is(err, appserver.ErrProtocol) {
		t.Fatalf("malformed notice error = %v, want ErrProtocol", err)
	}
	if title := s.Title(); title.Revision != 5 {
		t.Fatalf("title = %#v after a malformed notice, want revision 5", title)
	}
}

func TestTitleReadsAndRenamesShareTheSessionLock(t *testing.T) {
	gate := newFakeGate()
	server, s := openTitleSession(t, fakeConfig{
		Attach:       appserver.AttachResult{Sequence: int64Ptr(0)},
		History:      emptyHistory(t, 0),
		MutationGate: map[string]*fakeGate{"session.rename": gate},
		Mutations: map[string][]fakeMutation{
			"session.rename": {{Result: json.RawMessage(`{"title":"final","titleSource":"manual","revision":1}`)}},
		},
	})

	var readers sync.WaitGroup
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-done:
					return
				default:
					_ = s.Title()
				}
			}
		}()
	}
	errs := make(chan error, 1)
	go func() {
		_, err := s.Rename(context.Background(), stringPtr("final"))
		errs <- err
	}()
	waitFor(t, func() bool { return len(server.recorded("session.rename")) == 1 }, "rename request")
	gate.open()
	if err := <-errs; err != nil {
		t.Fatalf("Rename: %v", err)
	}
	close(done)
	readers.Wait()
	if title := s.Title(); title.Text == nil || *title.Text != "final" {
		t.Fatalf("title = %#v", title)
	}
}
