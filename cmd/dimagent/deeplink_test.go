package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/supervisor"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// TestParseConversationLink checks what the app reads out of a URL. Only the
// app's own scheme and thread route name a conversation; the scheme and the
// route are case-insensitive, while the id keeps its case. Every other URL,
// including one with an empty id or an extra slash, stays ignored so it cannot
// move the window's selection.
func TestParseConversationLink(t *testing.T) {
	const id = "8a1e5b6c-7d8f-4a90-b1c2-d3e4f5061728"
	for _, tc := range []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{"lowercase link", "dimagent://thread/" + id, id, true},
		{"uppercase scheme and route", "DIMAGENT://THREAD/" + id, id, true},
		{"mixed case scheme and route", "DimAgent://Thread/" + id, id, true},
		{"id keeps its case", "dimagent://thread/AbC-123", "AbC-123", true},
		{"escaped id", "dimagent://thread/a%2Fb", "a/b", true},
		{"surrounding space", "  dimagent://thread/" + id + "  ", id, true},
		{"empty", "", "", false},
		{"other scheme", "https://thread/" + id, "", false},
		{"other app scheme", "other://thread/" + id, "", false},
		{"other route", "dimagent://session/" + id, "", false},
		{"no route", "dimagent:///" + id, "", false},
		{"no id", "dimagent://thread", "", false},
		{"empty id", "dimagent://thread/", "", false},
		{"extra slash", "dimagent://thread//" + id, "", false},
		{"trailing slash", "dimagent://thread/" + id + "/", "", false},
		{"double slash after scheme", "dimagent:///thread/" + id, "", false},
		{"opaque form", "dimagent:thread/" + id, "", false},
		{"query", "dimagent://thread/" + id + "?focus=1", "", false},
		{"empty query", "dimagent://thread/" + id + "?", "", false},
		{"fragment", "dimagent://thread/" + id + "#top", "", false},
		{"port", "dimagent://thread:80/" + id, "", false},
		{"user info", "dimagent://user@thread/" + id, "", false},
		{"not a URL", "thread/" + id, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseConversationLink(tc.raw)
			if ok != tc.ok || got != tc.want {
				t.Errorf("parseConversationLink(%q) = %q, %v, want %q, %v", tc.raw, got, ok, tc.want, tc.ok)
			}
		})
	}
}

// TestConversationLinkRoundTrips checks the link the menu copies is the link the
// app opens: the two sides share the scheme and route, and the escaped id comes
// back unchanged.
func TestConversationLinkRoundTrips(t *testing.T) {
	for _, id := range []string{"s1", "8a1e5b6c-7d8f-4a90-b1c2-d3e4f5061728", "a/b"} {
		got, ok := parseConversationLink(dimui.ConversationLink(id))
		if !ok || got != id {
			t.Errorf("parseConversationLink(ConversationLink(%q)) = %q, %v", id, got, ok)
		}
	}
}

// TestAppOpenURLSelectsTheLinkedSession checks the deep-link path end to end
// inside the window: the link's id is looked up in the catalog and opened
// through the same selection a sidebar click uses, so the window shows that
// session and no request names another one.
func TestAppOpenURLSelectsTheLinkedSession(t *testing.T) {
	next := &fakeSession{view: &dimui.State{
		Selected:   "s2",
		Connection: dimui.ConnReady,
		Messages:   []dimui.Message{{ID: "m2", Content: "two's row"}},
	}}
	var opened []string
	b := &sessionBridge{}
	b.openSession = func(_ context.Context, _ *supervisor.Supervisor, id string) (sessionActions, error) {
		opened = append(opened, id)
		return next, nil
	}
	a, mu := createApp(b, &dimui.State{
		Sessions: []dimui.Session{{ID: "s1", Title: "One"}, {ID: "s2", Title: "Two"}},
		Selected: "s1",
		Messages: []dimui.Message{{ID: "m1", Content: "one's row"}},
	})

	a.openURL("dimagent://thread/s2")
	waitForState(t, a, mu, "the linked session selected", func(state *dimui.State) bool {
		return state.Selected == "s2" && !state.ConversationLoading
	})

	if len(opened) != 1 || opened[0] != "s2" {
		t.Fatalf("opened sessions = %q, want exactly [s2]", opened)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(a.state.Messages) != 1 || a.state.Messages[0].ID != "m2" {
		t.Fatalf("messages = %#v, want the linked session's rows", a.state.Messages)
	}
	if a.state.ActionError != "" || len(a.state.ActionNotices) != 0 {
		t.Fatalf("a linked session reported an outcome: %q %#v", a.state.ActionError, a.state.ActionNotices)
	}
}

// TestAppOpenURLReportsAMissingSession checks the link's other outcome: an id
// the catalog does not list is reported as such, the current selection stays,
// and no session is opened.
func TestAppOpenURLReportsAMissingSession(t *testing.T) {
	b := &sessionBridge{}
	var opened []string
	b.openSession = func(_ context.Context, _ *supervisor.Supervisor, id string) (sessionActions, error) {
		opened = append(opened, id)
		return &fakeSession{}, nil
	}
	a, mu := createApp(b, &dimui.State{
		Sessions: []dimui.Session{{ID: "s1", Title: "One"}},
		Selected: "s1",
	})

	a.openURL("dimagent://thread/missing")

	if len(opened) != 0 {
		t.Fatalf("a missing session was opened: %q", opened)
	}
	mu.Lock()
	defer mu.Unlock()
	if a.state.Selected != "s1" {
		t.Fatalf("selected = %q, want the previous session kept", a.state.Selected)
	}
	if !strings.Contains(a.state.ActionError, "missing") || !strings.Contains(a.state.ActionError, "no such session") {
		t.Errorf("action error = %q, want it to name the missing session", a.state.ActionError)
	}
	if len(a.state.ActionNotices) != 1 || a.state.ActionNotices[0].Kind != "link.error" {
		t.Errorf("missing session notice = %#v, want one link.error", a.state.ActionNotices)
	}
}

// TestAppOpenURLIgnoresOtherURLs checks that a URL which is not a conversation
// link changes nothing: no selection, no notice, no open, and no clipboard or
// session fact behind it.
func TestAppOpenURLIgnoresOtherURLs(t *testing.T) {
	b := &sessionBridge{}
	var opened []string
	b.openSession = func(_ context.Context, _ *supervisor.Supervisor, id string) (sessionActions, error) {
		opened = append(opened, id)
		return &fakeSession{}, nil
	}
	a, mu := createApp(b, &dimui.State{
		Sessions: []dimui.Session{{ID: "s1"}, {ID: "s2"}},
		Selected: "s1",
	})

	for _, raw := range []string{
		"https://example.com/thread/s2",
		"dimagent://session/s2",
		"dimagent://thread//s2",
		"",
	} {
		a.openURL(raw)
	}

	time.Sleep(20 * time.Millisecond)
	if len(opened) != 0 {
		t.Fatalf("an ignored URL opened a session: %q", opened)
	}
	mu.Lock()
	defer mu.Unlock()
	if a.state.Selected != "s1" || a.state.ActionError != "" || len(a.state.ActionNotices) != 0 {
		t.Fatalf("an ignored URL changed the window: %#v", a.state)
	}
}

// TestAppOpenURLWithoutABridgeReportsTheGap checks a link that arrives with no
// App Server: the session cannot be resolved, so the window reports that
// instead of selecting a row it does not have or reading local session storage.
func TestAppOpenURLWithoutABridgeReportsTheGap(t *testing.T) {
	a, mu := createApp(nil, &dimui.State{Sessions: []dimui.Session{{ID: "s1"}}})

	a.openURL("dimagent://thread/s1")

	mu.Lock()
	defer mu.Unlock()
	if a.state.Selected != "" {
		t.Fatalf("selected = %q without an App Server", a.state.Selected)
	}
	if !strings.Contains(a.state.ActionError, "no App Server") {
		t.Errorf("action error = %q, want it to name the missing connection", a.state.ActionError)
	}
	if len(a.state.ActionNotices) != 1 || a.state.ActionNotices[0].Kind != "link.error" {
		t.Errorf("missing connection notice = %#v, want one link.error", a.state.ActionNotices)
	}
}
