// This file owns the shell's session actions: the entry that asks the
// window to create a session and the entry that reports a sidebar choice.

package ui

// createSession asks the window to create a session. The shell adds no row and
// selects nothing: the new session appears only when the window publishes what
// session.create returned, so a click the App Server rejected leaves the
// catalog as it was.
func (s *State) createSession() {
	if s.OnCreateSession == nil || s.CreatingSession {
		return
	}
	s.OnCreateSession()
}

// selectSession tells the caller the sidebar chose another session. The shell
// changed Selected first; the caller opens that session and reports what it
// finds.
func (s *State) selectSession() {
	if s.OnSelect != nil {
		s.OnSelect(s.Selected)
	}
}
