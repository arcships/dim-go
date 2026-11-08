package ui

// Connection is the shell's view of the App Server connection, shown in the
// toolbar's status row. It is throwaway view state: the supervisor (Milestone
// 2) owns the connection, its retries and its errors, and reports them to the
// shell. Setting the field in the fixture proves the row's states; it is not a
// fact about the daemon, and nothing in this package reads a connection from
// anywhere else.
type Connection uint8

const (
	// ConnUnknown is before the shell has heard from the supervisor, which is
	// what a shell built without one shows.
	ConnUnknown Connection = iota
	// ConnConnecting is while the supervisor finds or starts its daemon.
	ConnConnecting
	// ConnReady is while the supervisor reports a healthy connection.
	ConnReady
	// ConnError is after an attempt failed, with State.ConnectionError beside
	// the status in a banner.
	ConnError
)

// String is the status row's label for the state.
func (c Connection) String() string {
	switch c {
	case ConnConnecting:
		return "Connecting…"
	case ConnReady:
		return "Connected"
	case ConnError:
		return "Connection error"
	default:
		return "Not connected"
	}
}
