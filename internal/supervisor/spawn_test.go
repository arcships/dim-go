package supervisor

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestBuildDaemonArgs(t *testing.T) {
	unixEndpoint := Endpoint{Network: NetworkUnix, Address: "/home/p/rust/daemon.sock"}
	pipeEndpoint := Endpoint{Network: NetworkPipe, Address: `\\.\pipe\dim-rust-abc`}
	base := SpawnConfig{
		Endpoint:    unixEndpoint,
		NativeHome:  "/home/p/rust",
		ProductHome: "/home/p",
	}
	cases := []struct {
		name    string
		mutate  func(*SpawnConfig)
		want    []string
		wantErr bool
	}{
		{
			name:   "product",
			mutate: func(c *SpawnConfig) {},
			want: []string{
				"--unix", "/home/p/rust/daemon.sock",
				"--no-ws", "--home", "/home/p/rust",
				"--product-home", "/home/p",
			},
		},
		{
			name: "configured",
			mutate: func(c *SpawnConfig) {
				c.ModelConfigPath = "/etc/model.json"
				c.DefaultsPath = "/etc/defaults.json"
			},
			want: []string{
				"--unix", "/home/p/rust/daemon.sock",
				"--no-ws", "--home", "/home/p/rust",
				"--defaults", "/etc/defaults.json",
				"--model-config", "/etc/model.json",
			},
		},
		{
			name: "defaults keeps product home",
			mutate: func(c *SpawnConfig) {
				c.DefaultsPath = "/etc/defaults.json"
			},
			want: []string{
				"--unix", "/home/p/rust/daemon.sock",
				"--no-ws", "--home", "/home/p/rust",
				"--defaults", "/etc/defaults.json",
				"--product-home", "/home/p",
			},
		},
		{
			name: "mock",
			mutate: func(c *SpawnConfig) {
				c.MockModel = true
				c.DefaultsPath = "/etc/defaults.json"
			},
			want: []string{
				"--unix", "/home/p/rust/daemon.sock",
				"--no-ws", "--home", "/home/p/rust",
				"--mock-model",
				"--defaults", "/etc/defaults.json",
			},
		},
		{
			name: "mock ignores product home",
			mutate: func(c *SpawnConfig) {
				c.MockModel = true
			},
			want: []string{
				"--unix", "/home/p/rust/daemon.sock",
				"--no-ws", "--home", "/home/p/rust",
				"--mock-model",
			},
		},
		{
			name: "pipe",
			mutate: func(c *SpawnConfig) {
				c.Endpoint = pipeEndpoint
			},
			want: []string{
				"--pipe", `\\.\pipe\dim-rust-abc`,
				"--no-ws", "--home", "/home/p/rust",
				"--product-home", "/home/p",
			},
		},
		{
			name: "product home required",
			mutate: func(c *SpawnConfig) {
				c.ProductHome = ""
			},
			wantErr: true,
		},
		{
			name: "native home required",
			mutate: func(c *SpawnConfig) {
				c.NativeHome = ""
			},
			wantErr: true,
		},
		{
			name: "invalid endpoint",
			mutate: func(c *SpawnConfig) {
				c.Endpoint.Network = "tcp"
			},
			wantErr: true,
		},
		{
			name: "missing endpoint address",
			mutate: func(c *SpawnConfig) {
				c.Endpoint.Address = ""
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.mutate(&cfg)
			got, err := BuildDaemonArgs(cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("BuildDaemonArgs = %v, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildDaemonArgs: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("args = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("args[%d] = %q, want %q (args %v)", i, got[i], tc.want[i], got)
				}
			}
		})
	}
}

func TestOpenDaemonLog(t *testing.T) {
	home := t.TempDir()
	file, err := openDaemonLog(home)
	if err != nil {
		t.Fatalf("openDaemonLog: %v", err)
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatalf("stat log: %v", err)
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("log mode = %04o, want 0600", perm)
		}
	}
	if _, err := file.Write([]byte("abc")); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close log: %v", err)
	}
	second, err := openDaemonLog(home)
	if err != nil {
		t.Fatalf("reopen log: %v", err)
	}
	if _, err := second.Write([]byte("de")); err != nil {
		t.Fatalf("append log: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("close log: %v", err)
	}
	info, err = os.Stat(filepath.Join(home, daemonLogName))
	if err != nil {
		t.Fatalf("stat reopened log: %v", err)
	}
	if info.Size() != 5 {
		t.Fatalf("log size = %d, want 5 (append mode)", info.Size())
	}
}

func TestDefaultSpawnMissingBinary(t *testing.T) {
	cfg := SpawnConfig{
		ServerPath:  filepath.Join(t.TempDir(), "missing-dim-app-server"),
		Endpoint:    Endpoint{Network: NetworkUnix, Address: "/tmp/x.sock"},
		NativeHome:  t.TempDir(),
		ProductHome: "/product",
	}
	if _, err := DefaultSpawn(context.Background(), cfg); err == nil {
		t.Fatal("DefaultSpawn succeeded with a missing binary")
	}
}

func TestWaitForListenSucceedsAfterRetry(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { _ = server.Close(); _ = client.Close() })
	tracked := &trackingConn{Conn: client}
	var attempts atomic.Int32
	dial := func(context.Context, Endpoint) (net.Conn, error) {
		if attempts.Add(1) < 3 {
			return nil, syscall.ENOENT
		}
		return tracked, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	endpoint := Endpoint{Network: NetworkUnix, Address: "/tmp/probe.sock"}
	if err := waitForListen(ctx, endpoint, dial, ListenWait{Timeout: time.Second, Interval: 2 * time.Millisecond}); err != nil {
		t.Fatalf("waitForListen: %v", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("attempts = %d, want 3", got)
	}
	if !tracked.closed.Load() {
		t.Fatal("successful probe connection was not closed")
	}
}

func TestWaitForListenTimeout(t *testing.T) {
	var attempts atomic.Int32
	dial := func(context.Context, Endpoint) (net.Conn, error) {
		attempts.Add(1)
		return nil, syscall.ENOENT
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	endpoint := Endpoint{Network: NetworkUnix, Address: "/tmp/never.sock"}
	err := waitForListen(ctx, endpoint, dial, ListenWait{Timeout: 40 * time.Millisecond, Interval: 2 * time.Millisecond})
	elapsed := time.Since(start)
	if !errors.Is(err, ErrListenTimeout) {
		t.Fatalf("error = %v, want ErrListenTimeout", err)
	}
	if attempts.Load() < 2 {
		t.Fatalf("attempts = %d, want at least 2", attempts.Load())
	}
	if elapsed > time.Second {
		t.Fatalf("timeout took %s, want a bounded wait", elapsed)
	}
}

func TestWaitForListenCancelBeforeDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var attempts atomic.Int32
	dial := func(context.Context, Endpoint) (net.Conn, error) {
		attempts.Add(1)
		return nil, syscall.ENOENT
	}
	err := waitForListen(ctx, Endpoint{Network: NetworkUnix, Address: "/tmp/x.sock"}, dial, ListenWait{Timeout: time.Second, Interval: time.Millisecond})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if attempts.Load() != 0 {
		t.Fatalf("attempts = %d, want 0 after prior cancellation", attempts.Load())
	}
}

func TestWaitForListenCancelDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var attempts atomic.Int32
	dial := func(context.Context, Endpoint) (net.Conn, error) {
		if attempts.Add(1) == 1 {
			cancel()
		}
		return nil, syscall.ENOENT
	}
	err := waitForListen(ctx, Endpoint{Network: NetworkUnix, Address: "/tmp/x.sock"}, dial, ListenWait{Timeout: time.Second, Interval: 2 * time.Millisecond})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestWaitForListenFatalError(t *testing.T) {
	var attempts atomic.Int32
	dial := func(context.Context, Endpoint) (net.Conn, error) {
		attempts.Add(1)
		return nil, syscall.EACCES
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := waitForListen(ctx, Endpoint{Network: NetworkUnix, Address: "/tmp/x.sock"}, dial, ListenWait{Timeout: time.Second, Interval: time.Millisecond})
	if err == nil {
		t.Fatal("waitForListen succeeded on a fatal error")
	}
	if errors.Is(err, ErrListenTimeout) || IsMissingEndpoint(err) {
		t.Fatalf("fatal error was misclassified: %v", err)
	}
	if !errors.Is(err, syscall.EACCES) {
		t.Fatalf("error = %v, want EACCES", err)
	}
	if attempts.Load() != 1 {
		t.Fatalf("attempts = %d, want 1", attempts.Load())
	}
}

func TestListenWaitDefaults(t *testing.T) {
	got := (ListenWait{}).normalized()
	if got.Timeout != DefaultListenTimeout || got.Interval != DefaultListenInterval {
		t.Fatalf("normalized = %+v, want the §18 defaults", got)
	}
}

type trackingConn struct {
	net.Conn
	closed atomic.Bool
}

func (c *trackingConn) Close() error {
	c.closed.Store(true)
	return c.Conn.Close()
}
