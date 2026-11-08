//go:build windows

package supervisor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"time"
)

func connect(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
	if endpoint.Network != NetworkPipe {
		return nil, fmt.Errorf("supervisor: endpoint network %q is not supported on this platform", endpoint.Network)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	name, err := syscall.UTF16PtrFromString(endpoint.Address)
	if err != nil {
		return nil, &net.OpError{Op: "dial", Net: NetworkPipe, Addr: pipeAddr(endpoint.Address), Err: err}
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		return nil, &net.OpError{Op: "dial", Net: NetworkPipe, Addr: pipeAddr(endpoint.Address), Err: err}
	}
	return &pipeConn{handle: handle, address: endpoint.Address}, nil
}

// pipeConn adapts a Windows pipe handle to net.Conn. The handle is opened in
// blocking mode, so deadlines are not implemented; the JSON-RPC client bounds
// waits through request contexts instead.
type pipeConn struct {
	handle  syscall.Handle
	address string
}

func (c *pipeConn) Read(p []byte) (int, error) {
	var done uint32
	err := syscall.ReadFile(c.handle, p, &done, nil)
	if err != nil {
		switch {
		case errors.Is(err, syscall.ERROR_BROKEN_PIPE), errors.Is(err, syscall.ERROR_HANDLE_EOF):
			return int(done), io.EOF
		case errors.Is(err, syscall.ERROR_MORE_DATA):
			// Message-mode pipe with a partial message: deliver the bytes and
			// let the next Read continue the stream.
			return int(done), nil
		default:
			return int(done), err
		}
	}
	return int(done), nil
}

func (c *pipeConn) Write(p []byte) (int, error) {
	var done uint32
	err := syscall.WriteFile(c.handle, p, &done, nil)
	return int(done), err
}

func (c *pipeConn) Close() error { return syscall.CloseHandle(c.handle) }

func (c *pipeConn) LocalAddr() net.Addr  { return pipeAddr(c.address) }
func (c *pipeConn) RemoteAddr() net.Addr { return pipeAddr(c.address) }

func (c *pipeConn) SetDeadline(time.Time) error      { return nil }
func (c *pipeConn) SetReadDeadline(time.Time) error  { return nil }
func (c *pipeConn) SetWriteDeadline(time.Time) error { return nil }

type pipeAddr string

func (a pipeAddr) Network() string { return NetworkPipe }
func (a pipeAddr) String() string  { return string(a) }
