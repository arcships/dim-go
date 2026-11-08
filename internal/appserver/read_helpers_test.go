package appserver

import (
	"encoding/json"
	"errors"
	"testing"
)

func catalogPages() map[string]fakeCatalogPage {
	return map[string]fakeCatalogPage{
		"":         {Sessions: []SessionSummary{fakeSummary("s1", 0), fakeSummary("s2", 0)}, NextCursor: stringPtr("cursor-2")},
		"cursor-2": {Sessions: []SessionSummary{fakeSummary("s3", 0)}, NextCursor: stringPtr("cursor-3")},
		"cursor-3": {Sessions: []SessionSummary{}, NextCursor: nil},
	}
}

func rawParams(t *testing.T, call fakeRequest) map[string]json.RawMessage {
	t.Helper()
	var params map[string]json.RawMessage
	if err := json.Unmarshal(call.Params, &params); err != nil {
		t.Fatalf("decode request params %s: %v", call.Params, err)
	}
	return params
}

func catalogIDs(sessions []SessionSummary) []string {
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.SessionID)
	}
	return ids
}

func TestCatalogHelper(t *testing.T) {
	t.Run("three pages with origins stripped", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:        fakeHello("session.catalog"),
			CatalogPages: catalogPages(),
		})
		result, err := Catalog(ctx, client, CatalogParams{
			Limit:           intPtr(200),
			WithInternal:    boolPtr(true),
			WithAcpSessions: boolPtr(true),
		}, []string{"session.catalog"})
		if err != nil {
			t.Fatalf("Catalog: %v", err)
		}
		if got := catalogIDs(result.Sessions); len(got) != 3 || got[0] != "s1" || got[1] != "s2" || got[2] != "s3" {
			t.Fatalf("sessions = %v", got)
		}
		if result.NextCursor != nil {
			t.Fatalf("nextCursor = %q, want absent at the end", *result.NextCursor)
		}

		calls := server.recordedCalls("session.catalog")
		if len(calls) != 3 {
			t.Fatalf("catalog calls = %d, want 3", len(calls))
		}
		for i, call := range calls {
			params := rawParams(t, call)
			if _, ok := params["withInternal"]; ok {
				t.Fatalf("call %d carried withInternal", i)
			}
			if _, ok := params["withAcpSessions"]; ok {
				t.Fatalf("call %d carried withAcpSessions", i)
			}
		}
		var first, second, third CatalogParams
		if err := json.Unmarshal(calls[0].Params, &first); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(calls[1].Params, &second); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(calls[2].Params, &third); err != nil {
			t.Fatal(err)
		}
		if first.Cursor != "" || second.Cursor != "cursor-2" || third.Cursor != "cursor-3" {
			t.Fatalf("cursors = %q, %q, %q", first.Cursor, second.Cursor, third.Cursor)
		}
	})

	t.Run("origins -32602 retried once", func(t *testing.T) {
		ctx := testContext(t)
		// The server does not advertise origins and rejects the params; the
		// caller's snapshot is stale and still claims the feature.
		server, client := newFakeServer(t, fakeConfig{
			Hello:        fakeHello("session.catalog"),
			CatalogPages: catalogPages(),
		})
		result, err := Catalog(ctx, client, CatalogParams{
			Limit:           intPtr(200),
			WithInternal:    boolPtr(true),
			WithAcpSessions: boolPtr(true),
		}, []string{"session.catalog", "session.catalog.origins"})
		if err != nil {
			t.Fatalf("Catalog: %v", err)
		}
		if got := catalogIDs(result.Sessions); len(got) != 3 {
			t.Fatalf("sessions = %v", got)
		}
		calls := server.recordedCalls("session.catalog")
		if len(calls) != 4 {
			t.Fatalf("catalog calls = %d, want 4 (1 rejected + 3 pages)", len(calls))
		}
		first := rawParams(t, calls[0])
		if _, ok := first["withInternal"]; !ok {
			t.Fatal("first call did not carry withInternal")
		}
		second := rawParams(t, calls[1])
		if _, ok := second["withInternal"]; ok {
			t.Fatal("retry kept withInternal")
		}
		if _, ok := second["withAcpSessions"]; ok {
			t.Fatal("retry kept withAcpSessions")
		}
		if _, ok := second["cursor"]; ok {
			t.Fatal("retry invented a cursor")
		}
		var secondParams, thirdParams CatalogParams
		if err := json.Unmarshal(calls[2].Params, &secondParams); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(calls[3].Params, &thirdParams); err != nil {
			t.Fatal(err)
		}
		if secondParams.Cursor != "cursor-2" || thirdParams.Cursor != "cursor-3" {
			t.Fatalf("cursors = %q, %q", secondParams.Cursor, thirdParams.Cursor)
		}
	})

	t.Run("missing feature is a capability error", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:        fakeHello("session.history"),
			CatalogPages: catalogPages(),
		})
		result, err := Catalog(ctx, client, CatalogParams{Limit: intPtr(200)}, []string{"session.history"})
		if !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("error = %v, want ErrCapabilityUnavailable", err)
		}
		var capability *CapabilityError
		if !errors.As(err, &capability) || capability.Feature != "session.catalog" {
			t.Fatalf("capability error = %#v", err)
		}
		if result.Sessions != nil {
			t.Fatalf("capability error returned sessions: %#v", result.Sessions)
		}
		if calls := server.recordedCalls("session.catalog"); len(calls) != 0 {
			t.Fatalf("catalog was called %d times without the feature", len(calls))
		}
	})

	t.Run("empty next cursor ends traversal", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello("session.catalog"),
			CatalogPages: map[string]fakeCatalogPage{
				"": {Sessions: []SessionSummary{fakeSummary("s1", 0)}, NextCursor: stringPtr("")},
			},
		})
		result, err := Catalog(ctx, client, CatalogParams{Limit: intPtr(200)}, []string{"session.catalog"})
		if err != nil {
			t.Fatalf("Catalog: %v", err)
		}
		if got := catalogIDs(result.Sessions); len(got) != 1 || got[0] != "s1" {
			t.Fatalf("sessions = %v", got)
		}
		if result.NextCursor != nil {
			t.Fatalf("nextCursor = %q, want absent", *result.NextCursor)
		}
		if calls := server.recordedCalls("session.catalog"); len(calls) != 1 {
			t.Fatalf("catalog calls = %d, want 1", len(calls))
		}
	})

	t.Run("list cwd requires workspaces", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:        fakeHello("session.catalog"),
			CatalogPages: catalogPages(),
		})
		result, err := Catalog(ctx, client, CatalogParams{
			Limit: intPtr(200),
			Cwd:   CwdList("/a", "/b"),
		}, []string{"session.catalog"})
		if !errors.Is(err, ErrCapabilityUnavailable) {
			t.Fatalf("error = %v, want ErrCapabilityUnavailable", err)
		}
		var capability *CapabilityError
		if !errors.As(err, &capability) || capability.Feature != "session.catalog.workspaces" {
			t.Fatalf("capability error = %#v", err)
		}
		if result.Sessions != nil {
			t.Fatalf("capability error returned sessions: %#v", result.Sessions)
		}
		if calls := server.recordedCalls("session.catalog"); len(calls) != 0 {
			t.Fatalf("catalog was called %d times without session.catalog.workspaces", len(calls))
		}
	})

	t.Run("list cwd is sent with workspaces", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:        fakeHello("session.catalog", "session.catalog.workspaces"),
			CatalogPages: catalogPages(),
		})
		if _, err := Catalog(ctx, client, CatalogParams{
			Limit: intPtr(200),
			Cwd:   CwdList("/a", "/b"),
		}, []string{"session.catalog", "session.catalog.workspaces"}); err != nil {
			t.Fatalf("Catalog: %v", err)
		}
		calls := server.recordedCalls("session.catalog")
		if len(calls) == 0 {
			t.Fatal("catalog was never called")
		}
		var cwd []string
		if err := json.Unmarshal(rawParams(t, calls[0])["cwd"], &cwd); err != nil {
			t.Fatalf("cwd params: %v", err)
		}
		if len(cwd) != 2 || cwd[0] != "/a" || cwd[1] != "/b" {
			t.Fatalf("cwd = %v", cwd)
		}
	})

	t.Run("repeated cursor is a protocol error", func(t *testing.T) {
		ctx := testContext(t)
		_, client := newFakeServer(t, fakeConfig{
			Hello: fakeHello("session.catalog"),
			CatalogPages: map[string]fakeCatalogPage{
				"":     {Sessions: []SessionSummary{fakeSummary("s1", 0)}, NextCursor: stringPtr("loop")},
				"loop": {Sessions: []SessionSummary{fakeSummary("s2", 0)}, NextCursor: stringPtr("loop")},
			},
		})
		_, err := Catalog(ctx, client, CatalogParams{Limit: intPtr(200)}, []string{"session.catalog"})
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
	})
}

func TestSummariesHelper(t *testing.T) {
	t.Run("includeClosed fallback once", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:              fakeHello(),
			Summaries:          []SessionSummary{fakeSummary("s1", 1)},
			SummariesOldDaemon: true,
		})
		summaries, err := Summaries(ctx, client, SummariesParams{IncludeClosed: true})
		if err != nil {
			t.Fatalf("Summaries: %v", err)
		}
		if len(summaries) != 1 || summaries[0].SessionID != "s1" {
			t.Fatalf("summaries = %#v", summaries)
		}
		calls := server.recordedCalls("session.summaries")
		if len(calls) != 2 {
			t.Fatalf("summaries calls = %d, want 2", len(calls))
		}
		if _, ok := rawParams(t, calls[0])["includeClosed"]; !ok {
			t.Fatal("first call omitted includeClosed")
		}
		if _, ok := rawParams(t, calls[1])["includeClosed"]; ok {
			t.Fatal("retry kept includeClosed")
		}
	})

	t.Run("no includeClosed means no retry", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:              fakeHello(),
			Summaries:          []SessionSummary{fakeSummary("s1", 1)},
			SummariesOldDaemon: true,
		})
		if _, err := Summaries(ctx, client, SummariesParams{}); err != nil {
			t.Fatalf("Summaries: %v", err)
		}
		if calls := server.recordedCalls("session.summaries"); len(calls) != 1 {
			t.Fatalf("summaries calls = %d, want 1", len(calls))
		}
	})
}

func TestHistoryPageHelper(t *testing.T) {
	after := int64(4)
	through := int64(10)
	req := HistoryParams{SessionID: "s1", After: &after, Through: &through}
	validPage := func() json.RawMessage {
		return rawJSON(t, map[string]any{
			"events":    []any{fakeEvent(5)},
			"through":   through,
			"nextAfter": int64(5),
			"done":      false,
		})
	}

	t.Run("valid page", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:      fakeHello("session.history"),
			HistoryRaw: []json.RawMessage{validPage()},
		})
		result, err := HistoryPage(ctx, client, req)
		if err != nil {
			t.Fatalf("HistoryPage: %v", err)
		}
		if result.Through == nil || *result.Through != through {
			t.Fatalf("through = %v", result.Through)
		}
		if len(result.Events) != 1 || result.Events[0].Sequence != 5 {
			t.Fatalf("events = %#v", result.Events)
		}
		if calls := server.recordedCalls("session.history"); len(calls) != 1 {
			t.Fatalf("history calls = %d, want 1", len(calls))
		}
	})

	t.Run("invariant violation", func(t *testing.T) {
		ctx := testContext(t)
		bad := rawJSON(t, map[string]any{
			"events":    []any{fakeEvent(5)},
			"through":   through,
			"nextAfter": int64(3), // before after
			"done":      false,
		})
		_, client := newFakeServer(t, fakeConfig{
			Hello:      fakeHello("session.history"),
			HistoryRaw: []json.RawMessage{bad},
		})
		_, err := HistoryPage(ctx, client, req)
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
	})

	t.Run("request validated before the call", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:      fakeHello("session.history"),
			HistoryRaw: []json.RawMessage{validPage()},
		})
		if _, err := HistoryPage(ctx, client, HistoryParams{}); !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if calls := server.recordedCalls("session.history"); len(calls) != 0 {
			t.Fatalf("history was called %d times with invalid params", len(calls))
		}
	})
}
