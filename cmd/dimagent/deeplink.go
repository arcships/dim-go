// This file owns the window's conversation deep links: the URL the app
// registers, how one is read back, and what opening it does.
//
// A link names only a session id. It carries no session fact of its own: the id
// is looked up in the catalog the App Server served, and the window selects the
// row through the same path a sidebar click uses. A link whose session the
// catalog does not list reports that instead of inventing a row or reading local
// session storage.

package main

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// parseConversationLink reads the session id out of a conversation deep link.
// Only the app's own scheme and the thread route name a conversation: any other
// URL is ignored (ok is false) so an unrelated link cannot move the window's
// selection. The scheme and the route are case-insensitive, as URL schemes and
// hosts are; the id keeps its case, because it is an opaque App Server fact.
// Exactly one path segment follows the route: an empty id, an extra slash or an
// id that is not one segment is not a conversation link.
func parseConversationLink(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}
	if !strings.EqualFold(u.Scheme, dimui.ConversationLinkScheme) {
		return "", false
	}
	if !strings.EqualFold(u.Host, dimui.ConversationLinkHost) {
		return "", false
	}
	if u.User != nil || u.Port() != "" {
		return "", false
	}
	// A link names one conversation and nothing else, so no query, fragment or
	// trailing "?" is accepted as part of it.
	if u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", false
	}
	path := u.EscapedPath()
	if !strings.HasPrefix(path, "/") {
		return "", false
	}
	escaped := strings.TrimPrefix(path, "/")
	if escaped == "" || strings.Contains(escaped, "/") {
		return "", false
	}
	id, err := url.PathUnescape(escaped)
	if err != nil || id == "" {
		return "", false
	}
	return id, true
}

// openURL handles one URL the system asked the app to open. It runs on the main
// thread, after the window exists and after the startup catalog read, so a link
// resolves against the sessions the App Server already reported. A URL that is
// not a conversation link is ignored: the app opens no other route from it.
func (a *app) openURL(raw string) {
	sessionID, ok := parseConversationLink(raw)
	if !ok {
		return
	}
	a.focusWindow()
	a.openLinkedSession(sessionID)
}

// openLinkedSession selects the session a deep link names, through the same
// selectSession path a sidebar click uses. The id is resolved against the
// catalog only; a link cannot name a session the App Server did not list, so a
// missing row is reported as such and the current selection stays. Nothing is
// read from local session storage.
func (a *app) openLinkedSession(sessionID string) {
	if a.state == nil {
		return
	}
	if a.bridge == nil {
		a.notice("link.error", fmt.Sprintf("Open conversation %q: no App Server is connected, so the session cannot be resolved.", sessionID))
		return
	}
	if _, ok := a.session(sessionID); !ok {
		a.notice("link.error", fmt.Sprintf("Open conversation %q: the App Server's catalog lists no such session.", sessionID))
		return
	}
	log.Printf("dimagent: open conversation %q from a deep link", sessionID)
	a.selectSession(sessionID)
}

// focusWindow brings the window forward when a link reaches a running app. The
// link opens a conversation in it, so the window must be visible and focused
// rather than left behind another one. A window that is gone has nothing to
// focus; the selection below still runs.
func (a *app) focusWindow() {
	if a.win == nil || a.win.IsDestroyed() {
		return
	}
	if !a.win.IsVisible() {
		a.win.Show()
	}
	if a.win.IsMinimized() {
		a.win.Restore()
	}
	a.win.Focus()
}
