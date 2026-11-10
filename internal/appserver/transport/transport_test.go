package transport_test

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/transport"
)

func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "d1-transport-")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func TestDialUnixRoundTrip(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "daemon.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		_, _ = io.Copy(conn, conn)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := transport.Dial(ctx, transport.Endpoint{Network: transport.NetworkUnix, Address: path}, time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("ping\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, 5)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf) != "ping\n" {
		t.Fatalf("round trip: %q", buf)
	}
}

func TestDialPipeIsNotImplemented(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := transport.Dial(ctx, transport.Endpoint{Network: transport.NetworkPipe, Address: `\\.\pipe\test`}, time.Second)
	if !errors.Is(err, transport.ErrUnsupportedPlatform) {
		t.Fatalf("want unsupported platform, got %v", err)
	}
}

func TestDialUnknownNetwork(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := transport.Dial(ctx, transport.Endpoint{Network: "tcp", Address: "127.0.0.1:1"}, time.Second); err == nil {
		t.Fatal("want an error for an unknown network")
	}
}
