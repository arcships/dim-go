package appserver

import (
	"sync"
	"testing"
	"time"
)

type catalogOutcome struct {
	result CatalogResult
	err    error
}

func awaitCatalogOutcome(t *testing.T, ch <-chan catalogOutcome) catalogOutcome {
	t.Helper()
	select {
	case outcome := <-ch:
		return outcome
	case <-time.After(fakeWaitTimeout):
		t.Fatal("catalog refresh did not finish")
		return catalogOutcome{}
	}
}

func sameCatalogIDs(left, right []SessionSummary) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].SessionID != right[i].SessionID {
			return false
		}
	}
	return true
}

func TestCatalogRefresherCoalescesConcurrentRefreshes(t *testing.T) {
	ctx := testContext(t)
	gate := newCatalogGate()
	t.Cleanup(gate.open)
	server, client := newFakeServer(t, fakeConfig{
		Hello:        fakeHello("session.catalog"),
		CatalogPages: catalogPages(),
		CatalogGate:  gate,
	})
	refresher := NewCatalogRefresher(client, []string{"session.catalog"})
	joined := make(chan struct{})
	var once sync.Once
	refresher.joined = func() { once.Do(func() { close(joined) }) }

	params := CatalogParams{Limit: intPtr(200)}
	first := make(chan catalogOutcome, 1)
	go func() {
		result, err := refresher.Refresh(ctx, params, false)
		first <- catalogOutcome{result: result, err: err}
	}()
	select {
	case <-gate.reached:
	case <-time.After(fakeWaitTimeout):
		t.Fatal("first catalog traversal did not start")
	}

	second := make(chan catalogOutcome, 1)
	go func() {
		result, err := refresher.Refresh(ctx, params, false)
		second <- catalogOutcome{result: result, err: err}
	}()
	select {
	case <-joined:
	case <-time.After(fakeWaitTimeout):
		t.Fatal("second refresh did not join the in-flight traversal")
	}

	gate.open()
	firstOutcome := awaitCatalogOutcome(t, first)
	secondOutcome := awaitCatalogOutcome(t, second)
	if firstOutcome.err != nil || secondOutcome.err != nil {
		t.Fatalf("refresh errors = %v, %v", firstOutcome.err, secondOutcome.err)
	}
	if !sameCatalogIDs(firstOutcome.result.Sessions, secondOutcome.result.Sessions) {
		t.Fatalf("waiters received different sessions: %v vs %v",
			catalogIDs(firstOutcome.result.Sessions), catalogIDs(secondOutcome.result.Sessions))
	}
	if got := catalogIDs(firstOutcome.result.Sessions); len(got) != 3 || got[0] != "s1" || got[1] != "s2" || got[2] != "s3" {
		t.Fatalf("sessions = %v", got)
	}
	if calls := server.recordedCalls("session.catalog"); len(calls) != 3 {
		t.Fatalf("catalog calls = %d, want one 3-page traversal", len(calls))
	}
}

func TestCatalogRefresherForceStartsNewTraversal(t *testing.T) {
	ctx := testContext(t)
	gate := newCatalogGate()
	t.Cleanup(gate.open)
	server, client := newFakeServer(t, fakeConfig{
		Hello:        fakeHello("session.catalog"),
		CatalogPages: catalogPages(),
		CatalogGate:  gate,
	})
	refresher := NewCatalogRefresher(client, []string{"session.catalog"})
	params := CatalogParams{Limit: intPtr(200)}

	first := make(chan catalogOutcome, 1)
	go func() {
		result, err := refresher.Refresh(ctx, params, false)
		first <- catalogOutcome{result: result, err: err}
	}()
	select {
	case <-gate.reached:
	case <-time.After(fakeWaitTimeout):
		t.Fatal("first catalog traversal did not start")
	}

	forced := make(chan catalogOutcome, 1)
	go func() {
		result, err := refresher.Refresh(ctx, params, true)
		forced <- catalogOutcome{result: result, err: err}
	}()
	gate.open()

	if outcome := awaitCatalogOutcome(t, first); outcome.err != nil {
		t.Fatalf("first refresh: %v", outcome.err)
	}
	if outcome := awaitCatalogOutcome(t, forced); outcome.err != nil {
		t.Fatalf("forced refresh: %v", outcome.err)
	}
	if calls := server.recordedCalls("session.catalog"); len(calls) != 6 {
		t.Fatalf("catalog calls = %d, want two 3-page traversals", len(calls))
	}
}
