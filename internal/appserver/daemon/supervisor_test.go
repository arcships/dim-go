package daemon_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/transport"
)

// TestMain 兼作自启测试的假服务端：设置环境变量后，测试二进制本身就是
// 一个最小的 App Server，参数由 client 的自启参数给出。
func TestMain(m *testing.M) {
	if os.Getenv("DIMAGENT_FAKE_DAEMON") == "1" {
		os.Exit(runHelperDaemon(os.Args[1:]))
	}
	os.Exit(m.Run())
}

func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "d1-daemon-")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func nativeHome(dataDir string) string { return filepath.Join(dataDir, "rust") }

func prepareDataDir(t *testing.T) string {
	t.Helper()
	dataDir := shortTempDir(t)
	if err := os.MkdirAll(nativeHome(dataDir), 0o700); err != nil {
		t.Fatalf("mkdir native home: %v", err)
	}
	return dataDir
}

func socketPath(dataDir string) string { return filepath.Join(nativeHome(dataDir), "daemon.sock") }

// fakeDaemon 是外部启动的假 App Server，用于发现与替换测试。
type fakeDaemon struct {
	instanceID      string
	protocolVersion int
	features        []string
	ln              net.Listener

	busy bool

	mu         sync.Mutex
	accepts    int
	shutdowns  int
	lastIfIdle bool
	conns      []net.Conn
	stopOnce   sync.Once
}

func startFakeDaemon(t *testing.T, path string, configure func(*fakeDaemon)) *fakeDaemon {
	t.Helper()
	_ = os.Remove(path)
	d := &fakeDaemon{instanceID: "instance-default", protocolVersion: protocol.ProtocolVersion}
	if configure != nil {
		configure(d)
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("fake daemon listen: %v", err)
	}
	d.ln = ln
	go d.accept()
	t.Cleanup(d.stop)
	return d
}

func (d *fakeDaemon) accept() {
	for {
		conn, err := d.ln.Accept()
		if err != nil {
			return
		}
		d.mu.Lock()
		d.accepts++
		d.conns = append(d.conns, conn)
		d.mu.Unlock()
		go d.serve(conn)
	}
}

func (d *fakeDaemon) serve(conn net.Conn) {
	defer conn.Close()
	decoder := protocol.NewDecoder(0)
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			frames, _ := decoder.Push(buf[:n])
			for _, frame := range frames {
				var request struct {
					ID     json.RawMessage `json:"id"`
					Method string          `json:"method"`
					Params json.RawMessage `json:"params"`
				}
				if json.Unmarshal(frame, &request) != nil || request.Method == "" {
					continue
				}
				d.reply(conn, protocol.ID(request.ID), request.Method, request.Params)
			}
		}
		if err != nil {
			return
		}
	}
}

func (d *fakeDaemon) reply(conn net.Conn, id protocol.ID, method string, params json.RawMessage) {
	if method == "hello" {
		if d.protocolVersion != protocol.ProtocolVersion {
			writeFrame(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"instanceId":      d.instanceID,
					"serverVersion":   "fake",
					"protocolVersion": d.protocolVersion,
					"daemonProfile":   map[string]any{"kind": "mock"},
				},
			})
			return
		}
		writeFrame(conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"result": map[string]any{
				"instanceId":      d.instanceID,
				"serverVersion":   "fake",
				"protocolVersion": protocol.ProtocolVersion,
				"daemonProfile":   map[string]any{"kind": "mock"},
				"features":        d.features,
			},
		})
		return
	}
	if method == "daemon.ping" {
		writeFrame(conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  map[string]any{"sessions": 0, "connections": 1, "serverVersion": "fake"},
		})
		return
	}
	if method == "daemon.shutdown" {
		var shutdown struct {
			Force  bool `json:"force"`
			IfIdle bool `json:"ifIdle"`
		}
		_ = json.Unmarshal(params, &shutdown)
		d.mu.Lock()
		d.shutdowns++
		d.lastIfIdle = shutdown.IfIdle
		busy := d.busy
		d.mu.Unlock()
		if busy {
			writeFrame(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"error": map[string]any{
					"code": -32000, "message": "busy",
					"data": map[string]any{"appError": "daemon_busy", "connections": 2},
				},
			})
			return
		}
		writeFrame(conn, map[string]any{"jsonrpc": "2.0", "id": id, "result": nil})
		d.stop()
		return
	}
	writeFrame(conn, map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": -32601, "message": "unknown", "data": map[string]any{"appError": "unknown_method"}},
	})
}

func writeFrame(conn net.Conn, value any) {
	frame, err := protocol.EncodeFrame(value)
	if err != nil {
		return
	}
	_, _ = conn.Write(frame)
}

func (d *fakeDaemon) stop() {
	d.stopOnce.Do(func() {
		_ = d.ln.Close()
		d.mu.Lock()
		conns := append([]net.Conn(nil), d.conns...)
		d.mu.Unlock()
		for _, conn := range conns {
			_ = conn.Close()
		}
	})
}

func (d *fakeDaemon) counts() (accepts, shutdowns int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.accepts, d.shutdowns
}

func (d *fakeDaemon) shutdownUsedIfIdle() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastIfIdle
}

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func dialEndpoint(t *testing.T, path string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := transport.Dial(ctx, transport.Endpoint{Network: transport.NetworkUnix, Address: path}, 500*time.Millisecond)
	if err == nil {
		_ = conn.Close()
	}
	return err
}

func TestConnectToExistingDaemon(t *testing.T) {
	dataDir := prepareDataDir(t)
	path := socketPath(dataDir)
	server := startFakeDaemon(t, path, func(d *fakeDaemon) { d.instanceID = "existing" })

	supervisor, err := daemon.New(daemon.Config{DataDir: dataDir})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	defer supervisor.Close(errors.New("test done"))

	conn, err := supervisor.Connect(context.Background())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if conn.Owned {
		t.Fatal("an existing daemon must not be reported as owned")
	}
	if conn.Hello.InstanceID != "existing" {
		t.Fatalf("instance: %q", conn.Hello.InstanceID)
	}
	if conn.Endpoint.Network != transport.NetworkUnix || conn.Endpoint.Address != supervisor.Endpoint().Address {
		t.Fatalf("endpoint: %+v / %+v", conn.Endpoint, supervisor.Endpoint())
	}
	accepts, _ := server.counts()
	if accepts != 1 {
		t.Fatalf("accepts: %d", accepts)
	}
	if supervisor.Owned() {
		t.Fatal("supervisor must not own an external daemon")
	}
}

func TestVersionIncompatibleStopsRetrying(t *testing.T) {
	dataDir := prepareDataDir(t)
	path := socketPath(dataDir)
	server := startFakeDaemon(t, path, func(d *fakeDaemon) { d.protocolVersion = 4 })

	supervisor, err := daemon.New(daemon.Config{DataDir: dataDir})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	defer supervisor.Close(errors.New("test done"))

	_, err = supervisor.Connect(context.Background())
	if !failure.Is(err, failure.KindVersionIncompatible) {
		t.Fatalf("first connect: %v", err)
	}
	_, err = supervisor.Connect(context.Background())
	if !failure.Is(err, failure.KindVersionIncompatible) {
		t.Fatalf("second connect: %v", err)
	}
	accepts, _ := server.counts()
	if accepts != 1 {
		t.Fatalf("must not retry after an incompatible version, accepts=%d", accepts)
	}
}

func TestSpawnOwnedDaemonAndReclaimIt(t *testing.T) {
	dataDir := prepareDataDir(t)
	path := socketPath(dataDir)

	t.Setenv("DIMAGENT_FAKE_DAEMON", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	supervisor, err := daemon.New(daemon.Config{
		DataDir:      dataDir,
		ServerPath:   executable,
		ServerArgs:   []string{"--"},
		MockModel:    true,
		SpawnTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	defer supervisor.Close(errors.New("test done"))

	conn, err := supervisor.Connect(context.Background())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if !conn.Owned {
		t.Fatal("a spawned daemon must be reported as owned")
	}
	if conn.Hello.InstanceID != "helper" {
		t.Fatalf("instance: %q", conn.Hello.InstanceID)
	}
	if err := dialEndpoint(t, path); err != nil {
		t.Fatalf("spawned daemon is not reachable: %v", err)
	}

	result, err := supervisor.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if !result.Stopped {
		t.Fatalf("owned daemon was not reclaimed: %+v", result)
	}
	waitFor(t, "owned daemon to exit", func() bool { return dialEndpoint(t, path) != nil })
}

func TestReconnectRelaunchesOwnedInstance(t *testing.T) {
	dataDir := prepareDataDir(t)
	spawns := 0
	var child *dummyChild
	var server *fakeDaemon
	supervisor, err := daemon.New(daemon.Config{DataDir: dataDir, ServerPath: "/unused"})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	supervisor.StopTimeout = 100 * time.Millisecond
	supervisor.Spawn = func(_ daemon.Config, request daemon.SpawnRequest) (daemon.Child, error) {
		spawns++
		index := spawns
		server = startFakeDaemon(t, request.Endpoint.Address, func(d *fakeDaemon) {
			d.instanceID = fmt.Sprintf("owned-%d", index)
		})
		child = &dummyChild{done: make(chan struct{})}
		return child, nil
	}

	conn, err := supervisor.Connect(context.Background())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if !conn.Owned || conn.Hello.InstanceID != "owned-1" {
		t.Fatalf("first connection: owned=%v instance=%q", conn.Owned, conn.Hello.InstanceID)
	}
	// 模拟自启实例退出：端点消失、子进程结束。
	if server == nil {
		t.Fatal("spawned server missing")
	}
	server.stop()
	_ = child.Kill()

	result, err := supervisor.Reconnect(context.Background())
	if err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	if spawns != 2 {
		t.Fatalf("owned instance must be relaunched, spawns=%d", spawns)
	}
	if !result.Restarted || result.Connection.Hello.InstanceID != "owned-2" {
		t.Fatalf("unexpected reconnect result: %+v", result)
	}
	if !result.Connection.Owned {
		t.Fatal("relaunched instance must stay owned")
	}
}

func TestShutdownLeavesExternalDaemon(t *testing.T) {
	dataDir := prepareDataDir(t)
	path := socketPath(dataDir)
	server := startFakeDaemon(t, path, nil)

	supervisor, err := daemon.New(daemon.Config{DataDir: dataDir})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	if _, err := supervisor.Connect(context.Background()); err != nil {
		t.Fatalf("connect: %v", err)
	}
	result, err := supervisor.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if result.Stopped || result.Reason != "not_owned" {
		t.Fatalf("unexpected shutdown result: %+v", result)
	}
	_, shutdowns := server.counts()
	if shutdowns != 0 {
		t.Fatalf("must not ask an external daemon to stop, shutdowns=%d", shutdowns)
	}
	if err := dialEndpoint(t, path); err != nil {
		t.Fatalf("external daemon must keep running: %v", err)
	}
}

func TestReconnectToReplacedDaemon(t *testing.T) {
	dataDir := prepareDataDir(t)
	path := socketPath(dataDir)
	first := startFakeDaemon(t, path, func(d *fakeDaemon) { d.instanceID = "first" })

	supervisor, err := daemon.New(daemon.Config{DataDir: dataDir})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	defer supervisor.Close(errors.New("test done"))

	conn, err := supervisor.Connect(context.Background())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if conn.Hello.InstanceID != "first" {
		t.Fatalf("instance: %q", conn.Hello.InstanceID)
	}

	first.stop()
	startFakeDaemon(t, path, func(d *fakeDaemon) { d.instanceID = "second" })

	result, err := supervisor.Reconnect(context.Background())
	if err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	if !result.Restarted {
		t.Fatal("replacing the daemon must be reported as restarted")
	}
	if result.Connection.Hello.InstanceID != "second" {
		t.Fatalf("instance after reconnect: %q", result.Connection.Hello.InstanceID)
	}
}

type dummyChild struct {
	done chan struct{}
	once sync.Once
}

func (d *dummyChild) PID() int                { return 1 }
func (d *dummyChild) Exited() <-chan struct{} { return d.done }
func (d *dummyChild) Kill() error {
	d.once.Do(func() { close(d.done) })
	return nil
}

func TestSpawnedDaemonIsReclaimedWhenHandshakeNeverCompletes(t *testing.T) {
	dataDir := prepareDataDir(t)
	child := &dummyChild{done: make(chan struct{})}
	supervisor, err := daemon.New(daemon.Config{
		DataDir:      dataDir,
		ServerPath:   "/unused",
		SpawnTimeout: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	supervisor.StopTimeout = 100 * time.Millisecond
	supervisor.Spawn = func(_ daemon.Config, _ daemon.SpawnRequest) (daemon.Child, error) {
		return child, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := supervisor.Connect(ctx); !failure.Is(err, failure.KindTimeout) {
		t.Fatalf("connect: %v", err)
	}
	select {
	case <-child.done:
	case <-time.After(time.Second):
		t.Fatal("spawned child must be reclaimed when it never becomes usable")
	}
}

func TestShutdownUsesIdleDecisionWhenAdvertised(t *testing.T) {
	cases := []struct {
		name            string
		busy            bool
		wantStopped     bool
		wantReason      string
		wantConnections int
	}{
		{"idle", false, true, "", 0},
		{"busy", true, false, "busy", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dataDir := prepareDataDir(t)
			var server *fakeDaemon
			supervisor, err := daemon.New(daemon.Config{DataDir: dataDir, ServerPath: "/unused"})
			if err != nil {
				t.Fatalf("supervisor: %v", err)
			}
			supervisor.StopTimeout = 100 * time.Millisecond
			supervisor.Spawn = func(_ daemon.Config, request daemon.SpawnRequest) (daemon.Child, error) {
				server = startFakeDaemon(t, request.Endpoint.Address, func(d *fakeDaemon) {
					d.features = []string{"daemon.shutdown.ifIdle"}
					d.busy = tc.busy
				})
				return &dummyChild{done: make(chan struct{})}, nil
			}

			conn, err := supervisor.Connect(context.Background())
			if err != nil {
				t.Fatalf("connect: %v", err)
			}
			if !conn.Owned {
				t.Fatal("spawned daemon must be owned")
			}
			result, err := supervisor.Shutdown(context.Background())
			if err != nil {
				t.Fatalf("shutdown: %v", err)
			}
			if result.Stopped != tc.wantStopped || result.Reason != tc.wantReason {
				t.Fatalf("result: %+v", result)
			}
			if tc.busy {
				if result.Connections != tc.wantConnections {
					t.Fatalf("connections: %+v", result)
				}
				if err := dialEndpoint(t, server.ln.Addr().String()); err != nil {
					t.Fatalf("busy daemon must stay up: %v", err)
				}
			}
			if !server.shutdownUsedIfIdle() {
				t.Fatal("shutdown must carry ifIdle when the daemon advertises it")
			}
		})
	}
}

func TestSendToUnknownMethodIsNonRetryable(t *testing.T) {
	// 直接验证错误分类的输入来源，避免把服务端业务错误当连接错误。
	err := &protocol.RPCError{Code: -32601, Message: "unknown", Data: []byte(`{"appError":"unknown_method"}`)}
	if !failure.Is(err, failure.KindNonRetryable) {
		t.Fatalf("kind: %q", failure.KindOf(err))
	}
}

// runHelperDaemon 是自启测试的子进程入口：解析自启参数并服务一个最小协议。
func runHelperDaemon(args []string) int {
	path := flagValue(args, "--unix")
	if path == "" {
		fmt.Fprintln(os.Stderr, "missing --unix")
		return 2
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "listen:", err)
		return 3
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return 0
		}
		go serveHelperConn(conn, ln)
	}
}

func flagValue(args []string, name string) string {
	for index, arg := range args {
		if arg == name && index+1 < len(args) {
			return args[index+1]
		}
	}
	return ""
}

func serveHelperConn(conn net.Conn, ln net.Listener) {
	defer conn.Close()
	decoder := protocol.NewDecoder(0)
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			frames, _ := decoder.Push(buf[:n])
			for _, frame := range frames {
				var request struct {
					ID     json.RawMessage `json:"id"`
					Method string          `json:"method"`
				}
				if json.Unmarshal(frame, &request) != nil || request.Method == "" {
					continue
				}
				switch request.Method {
				case "hello":
					writeFrame(conn, map[string]any{
						"jsonrpc": "2.0",
						"id":      protocol.ID(request.ID),
						"result": map[string]any{
							"instanceId":      "helper",
							"serverVersion":   "helper",
							"protocolVersion": protocol.ProtocolVersion,
							"daemonProfile":   map[string]any{"kind": "mock"},
							"features":        []string{},
						},
					})
				case "daemon.ping":
					writeFrame(conn, map[string]any{
						"jsonrpc": "2.0",
						"id":      protocol.ID(request.ID),
						"result":  map[string]any{"sessions": 0, "connections": 1, "serverVersion": "helper"},
					})
				case "daemon.shutdown":
					writeFrame(conn, map[string]any{
						"jsonrpc": "2.0",
						"id":      protocol.ID(request.ID),
						"result":  nil,
					})
					_ = conn.Close()
					_ = ln.Close()
					return
				default:
					writeFrame(conn, map[string]any{
						"jsonrpc": "2.0",
						"id":      protocol.ID(request.ID),
						"error": map[string]any{
							"code": -32601, "message": "unknown",
							"data": map[string]any{"appError": "unknown_method"},
						},
					})
				}
			}
		}
		if err != nil {
			return
		}
	}
}
