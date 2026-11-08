package supervisor

import (
	"errors"
	"io"
	"net"
	"testing"
)

func TestWatchedConnRecordsReadError(t *testing.T) {
	local, remote := net.Pipe()
	defer local.Close()
	conn := &watchedConn{Conn: local}
	if err := remote.Close(); err != nil {
		t.Fatalf("close remote: %v", err)
	}
	if _, err := conn.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("Read error = %v, want io.EOF", err)
	}
	if err := conn.Err(); !errors.Is(err, io.EOF) {
		t.Fatalf("Err = %v, want io.EOF", err)
	}
}

func TestWatchedConnRecordsLocalClose(t *testing.T) {
	local, remote := net.Pipe()
	defer remote.Close()
	conn := &watchedConn{Conn: local}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !errors.Is(conn.Err(), ErrConnectionClosed) {
		t.Fatalf("Err = %v, want ErrConnectionClosed", conn.Err())
	}
}

func TestWatchedConnKeepsFirstError(t *testing.T) {
	local, remote := net.Pipe()
	defer local.Close()
	conn := &watchedConn{Conn: local}
	if err := remote.Close(); err != nil {
		t.Fatalf("close remote: %v", err)
	}
	if _, err := conn.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("Read error = %v, want io.EOF", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close watched conn: %v", err)
	}
	if err := conn.Err(); !errors.Is(err, io.EOF) {
		t.Fatalf("Err = %v, want the first io.EOF", err)
	}
}
