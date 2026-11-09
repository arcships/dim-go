//go:build unix

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/supervisor"
)

// startKeysBridge starts a fake daemon that advertises session.keys with the
// given snapshot, plus a started supervisor linked to it.
func startKeysBridge(t *testing.T, keys json.RawMessage) (*fakeDaemon, *supervisor.Supervisor, context.Context) {
	t.Helper()
	home := fakeHome(t)
	link, err := supervisor.ResolveLink(home)
	if err != nil {
		t.Fatalf("resolve link: %v", err)
	}
	daemon := startFakeDaemon(t, link.Endpoint.Address)
	daemon.setFeatures("session.catalog", "session.history", "session.keys")
	daemon.setKeys(keys)
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

// TestControllerKeysPassThrough checks that the controller hands the caller the
// session.keys snapshot Open captured: the through and the entries survive the
// bridge unchanged, and the caller gets a copy rather than the session's own
// slice.
func TestControllerKeysPassThrough(t *testing.T) {
	daemon, sup, ctx := startKeysBridge(t, json.RawMessage(`{"through":7,"entries":[
		{"key":"task","sequence":7,"inputId":"input-1","text":"wire the keys panel"}
	]}`))

	controller, err := OpenSession(ctx, sup, "s1")
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	closeControllerOnCleanup(t, controller)

	keys := controller.Keys()
	if keys.Through == nil || *keys.Through != 7 {
		t.Fatalf("keys through = %v, want 7", keys.Through)
	}
	if len(keys.Entries) != 1 || keys.Entries[0].Key != "task" ||
		keys.Entries[0].Sequence != 7 || keys.Entries[0].InputID != "input-1" ||
		keys.Entries[0].Text != "wire the keys panel" {
		t.Fatalf("keys entries = %#v", keys.Entries)
	}
	params := daemon.lastParams("session.keys")
	if params["sessionId"] != "s1" || params["branchId"] != "main" {
		t.Fatalf("session.keys params = %#v, want s1 on main", params)
	}

	// The caller's copy is independent: rewriting it must not change the
	// snapshot the session keeps.
	keys.Entries[0].Key = "rewritten"
	again := controller.Keys()
	if len(again.Entries) != 1 || again.Entries[0].Key != "task" {
		t.Fatalf("the snapshot changed through the caller's copy: %#v", again.Entries)
	}
	if got := daemon.callCount("session.keys"); got != 1 {
		t.Fatalf("session.keys calls = %d, want the one read Open made", got)
	}
}

// TestControllerKeysWithoutSessionIsEmpty checks that a controller with no
// attachment reports no snapshot instead of synthesizing one.
func TestControllerKeysWithoutSessionIsEmpty(t *testing.T) {
	controller := &SessionController{}
	if keys := controller.Keys(); keys.Through != nil || len(keys.Entries) != 0 {
		t.Fatalf("keys without a session = %#v, want empty", keys)
	}
	var nilController *SessionController
	if keys := nilController.Keys(); keys.Through != nil || len(keys.Entries) != 0 {
		t.Fatalf("keys on a nil controller = %#v, want empty", keys)
	}
}
