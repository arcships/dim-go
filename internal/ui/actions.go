// This file owns the shell's session actions: the entry that asks the
// window to create a session and the entry that reports a sidebar choice. Both
// take the region's Actions rather than State, so a region can call them without
// reaching back into the window.

package ui

// createSession asks the window to create a session. The shell adds no row and
// selects nothing: the new session appears only when the window publishes what
// session.create returned, so a click the App Server rejected leaves the
// catalog as it was. A missing callback keeps the entry a WIP gap, and an entry
// that already has a creation in flight asks for nothing more.
func createSession(a Actions, creating bool) {
	if a.OnCreateSession == nil || creating {
		return
	}
	a.OnCreateSession()
}

// selectSession tells the caller the sidebar chose another session. The sidebar
// wrote the choice to SidebarView.Selected first; the caller opens that session
// and reports what it finds.
func selectSession(a Actions, sessionID string) {
	if a.OnSelect != nil {
		a.OnSelect(sessionID)
	}
}
