package supervisor

import (
	"errors"
	"net"
	"sync"
)

var (
	// ErrConnectionClosed is the terminal reason for a connection this
	// supervisor or its caller closed locally.
	ErrConnectionClosed = errors.New("supervisor: connection closed")
	// ErrTransportLost is the fallback terminal reason when the transport
	// ended without exposing a more specific read/write error.
	ErrTransportLost = errors.New("supervisor: connection transport lost")
)

// watchedConn records the first terminal read/write/close error so a
// Connection can report why it ended. It adds no goroutines; appserver.Client
// already exposes Done, and this wrapper supplies the reason.
type watchedConn struct {
	net.Conn

	mu  sync.Mutex
	err error
}

func (c *watchedConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if err != nil {
		c.record(err)
	}
	return n, err
}

func (c *watchedConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if err != nil {
		c.record(err)
	}
	return n, err
}

func (c *watchedConn) Close() error {
	c.record(ErrConnectionClosed)
	return c.Conn.Close()
}

func (c *watchedConn) record(err error) {
	if err == nil {
		return
	}
	c.mu.Lock()
	if c.err == nil {
		c.err = err
	}
	c.mu.Unlock()
}

// Err returns the first terminal transport error, or nil while the connection
// has not failed.
func (c *watchedConn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}
