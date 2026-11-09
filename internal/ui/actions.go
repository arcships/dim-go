// This file owns the shell's session actions: the entry that asks the
// window to create a session, the entry that reports a sidebar choice and the
// entry that asks for another branch. All take the region's Actions rather than
// State, so a region can call them without reaching back into the window.

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

// selectBranch tells the caller the sidebar chose another branch. Unlike a
// session choice, the sidebar writes nothing first: SidebarView.SelectedBranch
// moves only when the window publishes a branch the App Server opened, and a
// failed read keeps the current one. The branch already shown asks for no read,
// so the callback fires once for a new branch.
func selectBranch(a Actions, branchID string, current bool) {
	if a.OnSelectBranch == nil || branchID == "" || current {
		return
	}
	a.OnSelectBranch(branchID)
}
