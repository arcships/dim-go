package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestStatusRowLabels checks that the toolbar's status row shows each
// connection state, and that assistive technology reads it as a live status.
func TestStatusRowLabels(t *testing.T) {
	cases := []struct {
		conn  Connection
		label string
	}{
		{ConnUnknown, "Not connected"},
		{ConnConnecting, "Connecting…"},
		{ConnReady, "Connected"},
		{ConnError, "Connection error"},
	}
	for _, tc := range cases {
		s := fixture()
		s.Connection = tc.conn
		tt := ui.NewTester(s.View, 900, 600)
		want := "App Server: " + tc.label
		if !tt.HasText(want) {
			t.Errorf("%v: status row does not show %q, texts %q", tc.conn, want, tt.Texts())
			continue
		}
		if _, ok := tt.AccessByRole(ui.RoleStatus, want); !ok {
			t.Errorf("%v: status %q is not a live status region", tc.conn, want)
		}
	}
}

// TestErrorBanner checks that a failed connection shows its message in an alert
// live region, and that the conversation beside it still builds.
func TestErrorBanner(t *testing.T) {
	s := fixture()
	s.Connection = ConnError
	s.ConnectionError = "daemon did not answer hello"
	tt := ui.NewTester(s.View, 900, 600)

	want := "Connection error: daemon did not answer hello"
	if !tt.HasText(want) {
		t.Fatalf("banner does not show %q, texts %q", want, tt.Texts())
	}
	// A live region's name is its content, which mygo exposes as the region's
	// text: the alert shows nothing itself. Check that the message is drawn
	// inside the region assistive technology reads out.
	alert, ok := tt.AccessByRole(ui.RoleAlert, "")
	if !ok {
		t.Fatalf("the banner is not an alert region, texts %q", tt.Texts())
	}
	text, ok := tt.Find(want)
	if !ok {
		t.Fatalf("the banner's message is not drawn, texts %q", tt.Texts())
	}
	if !containsRect(alert.Bounds, text) {
		t.Errorf("message box %+v is outside the alert region %+v", text, alert.Bounds)
	}
	if !tt.HasText("hello from the fixture") {
		t.Errorf("conversation is missing beside the banner, texts %q", tt.Texts())
	}
}

// containsRect reports whether outer holds inner.
func containsRect(outer, inner ui.Rect) bool {
	return inner.X >= outer.X && inner.Y >= outer.Y &&
		inner.X+inner.W <= outer.X+outer.W && inner.Y+inner.H <= outer.Y+outer.H
}

// TestNoBannerWithoutError checks that a stale error message shows nothing while
// the connection is not in error.
func TestNoBannerWithoutError(t *testing.T) {
	s := fixture()
	s.Connection = ConnReady
	s.ConnectionError = "an error from an attempt that is over"
	tt := ui.NewTester(s.View, 900, 600)

	if tt.HasText("an error from an attempt that is over") {
		t.Errorf("stale error shows while connected, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("an alert is exposed while connected")
	}
}

// TestLoadingConversation checks the loading state: its text and busy region
// show, and the rows wait for history.
func TestLoadingConversation(t *testing.T) {
	s := fixture()
	s.ConversationLoading = true
	tt := ui.NewTester(s.View, 900, 600)

	if !tt.HasText("Loading conversation…") {
		t.Fatalf("loading state is missing, texts %q", tt.Texts())
	}
	if tt.HasText("hello from the fixture") || tt.HasText("No messages yet") {
		t.Errorf("rows or the empty state show while loading, texts %q", tt.Texts())
	}
	status, ok := tt.AccessByRole(ui.RoleStatus, "Loading conversation…")
	if !ok {
		t.Fatalf("the loading state is not a status region, texts %q", tt.Texts())
	}
	if !status.Busy {
		t.Errorf("the loading region is not busy, so assistive technology does not wait for the rows: %+v", status)
	}
}

// TestStatusDefaultsToUnknown checks that a shell built without a supervisor, as
// the fixture is, says so instead of claiming a connection.
func TestStatusDefaultsToUnknown(t *testing.T) {
	tt := ui.NewTester(fixture().View, 900, 600)
	if !tt.HasText("App Server: Not connected") {
		t.Fatalf("a shell without a supervisor does not say it is not connected, texts %q", tt.Texts())
	}
}
