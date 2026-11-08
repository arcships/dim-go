package supervisor

import (
	"errors"
	"io"
	"net"
	"testing"
)

// TestWatchedConnRecordsWriteError pins the lost-transport reason for a
// write-side failure: the first write error is kept so Connection.Err can
// report why the stream ended instead of falling back to ErrTransportLost.
func TestWatchedConnRecordsWriteError(t *testing.T) {
	local, remote := net.Pipe()
	defer local.Close()
	conn := &watchedConn{Conn: local}
	if err := remote.Close(); err != nil {
		t.Fatalf("close remote: %v", err)
	}
	if _, err := conn.Write([]byte("x")); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Write error = %v, want io.ErrClosedPipe", err)
	}
	if err := conn.Err(); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Err = %v, want io.ErrClosedPipe", err)
	}
}
