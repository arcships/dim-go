package ui

// RunState is the shell's view of what the selected session is doing, shown
// beside the connection. It is throwaway view state: the session's run comes
// from the App Server's own session state, which the supervisor reports, and
// nothing in this package infers it from rows, timers or logs.
type RunState uint8

const (
	// RunIdle is a session with nothing running, which is what a shell built
	// without a session shows.
	RunIdle RunState = iota
	// RunRunning is a turn the App Server is still working on.
	RunRunning
	// RunStopping is a turn whose stop the App Server has taken and is
	// settling; the shell offers no second stop while it settles.
	RunStopping
)

// String is the status row's label for the state.
func (s RunState) String() string {
	switch s {
	case RunRunning:
		return "Running"
	case RunStopping:
		return "Stopping…"
	default:
		return "Idle"
	}
}
