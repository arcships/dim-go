package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestStatusRowLeavesTheShell checks that the global status row is gone: a
// healthy or unknown connection draws no "App Server" reading in the frame, as
// the reference shows none. The composer's context row will carry status later;
// this shell no longer paints a bottom bar or a connection line in the toolbar.
func TestStatusRowLeavesTheShell(t *testing.T) {
	for _, conn := range []Connection{ConnUnknown, ConnConnecting, ConnReady, ConnError} {
		s := fixture()
		s.Connection = conn
		tt := ui.NewTester(s.View, 900, 600)
		for _, gone := range []string{"App Server:", "Branch: WIP", "Context: WIP", "Model: WIP", "Account: WIP"} {
			if tt.HasText(gone) {
				t.Errorf("%v: the shell still draws the status text %q, texts %q", conn, gone, tt.Texts())
			}
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

// TestActionErrorBanner checks that a window-owned action error is shown even
// when the connection itself is ready, and survives a live view that has no
// connection error.
func TestActionErrorBanner(t *testing.T) {
	s := fixture()
	s.Connection = ConnReady
	s.ActionError = `Open session "s2": boom`
	s.ActionNotices = []Notice{{Kind: "select.error", ID: "boom"}}
	tt := ui.NewTester(s.View, 900, 600)

	if !tt.HasText(`Open session "s2": boom`) {
		t.Fatalf("action error banner is missing, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); !ok {
		t.Fatalf("action error is not an alert region, texts %q", tt.Texts())
	}
	if !tt.HasText("Notices: 1") {
		t.Errorf("window action outcome count is missing, texts %q", tt.Texts())
	}
}

func TestConnectionErrorBannerTakesPrecedence(t *testing.T) {
	s := fixture()
	s.Connection = ConnError
	s.ConnectionError = "transport lost"
	s.ActionError = "submit failed earlier"
	tt := ui.NewTester(s.View, 900, 600)

	if !tt.HasText("Connection error: transport lost") {
		t.Fatalf("connection error banner is missing, texts %q", tt.Texts())
	}
	if tt.HasText("submit failed earlier") {
		t.Fatalf("stale action error obscures the connection error, texts %q", tt.Texts())
	}
}

// TestClosedSessionGatesActions checks the terminal session.closed health
// state: it is shown, and submit/stop/reconnect are all unavailable even though
// the transport may still say connected.
func TestClosedSessionGatesActions(t *testing.T) {
	s := fixture()
	s.Connection = ConnReady
	s.Selected = "s1"
	s.Draft = "hello"
	s.Run = RunRunning
	s.SessionClosed = true
	s.SessionClosedReason = "done"
	submits, stops, reconnects := 0, 0, 0
	s.OnSubmit = func(string) { submits++ }
	s.OnStop = func() { stops++ }
	s.OnReconnect = func() { reconnects++ }
	tt := ui.NewTester(s.View, 900, 600)

	if !tt.HasText("Session closed: done") {
		t.Fatalf("closed banner is missing, texts %q", tt.Texts())
	}
	if !tt.HasText("Session: Closed") {
		t.Errorf("closed run label is missing, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); !ok {
		t.Fatalf("closed session is not an alert region, texts %q", tt.Texts())
	}
	info, ok := tt.AccessByRole(ui.RoleButton, "Send")
	if !ok {
		t.Fatalf("the composer offers no Send entry, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("Send is available after session.closed")
	}
	for _, label := range []string{"Stop", "Reconnect"} {
		if _, ok := tt.AccessByRole(ui.RoleButton, label); ok {
			t.Errorf("%q is available after session.closed", label)
		}
	}
	if err := tt.Click("Composer"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ComposerShortcutMods, ui.KeyEnter)
	if submits != 0 || stops != 0 || reconnects != 0 {
		t.Fatalf("closed session ran actions: submit=%d stop=%d reconnect=%d", submits, stops, reconnects)
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
	if tt.HasText("hello from the fixture") || tt.HasText(tr(LocaleEN, keyHeroTitle)) || tt.HasText("No messages yet") {
		t.Errorf("rows or an empty state show while loading, texts %q", tt.Texts())
	}
	status, ok := tt.AccessByRole(ui.RoleStatus, "Loading conversation…")
	if !ok {
		t.Fatalf("the loading state is not a status region, texts %q", tt.Texts())
	}
	if !status.Busy {
		t.Errorf("the loading region is not busy, so assistive technology does not wait for the rows: %+v", status)
	}
}

// TestNoConnectionClaimInTheEmptyState checks that a shell built without a
// supervisor makes no connection claim when no session is selected: the empty
// state has no application bar, and the banner stays empty until a real error.
func TestNoConnectionClaimInTheEmptyState(t *testing.T) {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	s.Messages = nil
	tt := ui.NewTester(s.View, 900, 900)

	if tt.HasText("App Server: Not connected") {
		t.Fatalf("the empty state claims a connection state, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("the empty state draws an alert without an error")
	}
}
