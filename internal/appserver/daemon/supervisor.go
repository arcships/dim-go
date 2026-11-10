package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/transport"
)

// Connection 是一条已完成 hello 的连接。控制连接与每条会话流各有一条。
type Connection struct {
	Client   *protocol.Client
	Hello    protocol.HelloResult
	Endpoint transport.Endpoint
	// Owned 表示这条连接背后的服务实例由本进程拉起；回收只对它生效。
	Owned bool
}

// HasFeature 判断服务端是否声明了某个可选方法。
func (c *Connection) HasFeature(name string) bool {
	return c != nil && c.Hello.HasFeature(name)
}

// Close 关闭这条连接。
func (c *Connection) Close(reason error) {
	if c == nil || c.Client == nil {
		return
	}
	c.Client.Close(reason)
}

// ReconnectResult 是一次重连的结果。
type ReconnectResult struct {
	Connection *Connection
	// Restarted 表示服务实例已经换了（instanceId 变化或重新拉起），
	// 上层必须以新实例的事实重建，而不是接着旧序号。
	Restarted bool
}

// ShutdownResult 是回收自启实例的结果。
type ShutdownResult struct {
	Stopped bool
	// Reason 非空表示没有回收：not_owned / busy / other_clients。
	Reason      string
	Connections int
}

// Supervisor 负责发现、自启、握手与回收。它按连接状态机工作：
// 未连接 → 发现中 → 握手中 → 已连接（或停止），断开后进入重连。
type Supervisor struct {
	cfg Config
	ep  endpoint

	// 以下两个钩子可在测试中替换；nil 时使用默认实现。
	Dial  func(ctx context.Context, endpoint transport.Endpoint, timeout time.Duration) (transport.Transport, error)
	Spawn func(cfg Config, request SpawnRequest) (Child, error)

	// StopTimeout 是等待自启实例退出的上限；超时后只杀自己拉起的进程。
	StopTimeout time.Duration

	connectMu sync.Mutex

	mu       sync.Mutex
	conn     *Connection
	owned    bool
	child    Child
	streams  map[*Connection]struct{}
	terminal error
	closed   bool
}

// New 解析端点并创建监督者；端点目录不可用时直接失败。
func New(cfg Config) (*Supervisor, error) {
	cfg = cfg.withDefaults()
	ep, err := resolveEndpoint(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	return &Supervisor{
		cfg:         cfg,
		ep:          ep,
		Dial:        transport.Dial,
		Spawn:       spawnDaemon,
		StopTimeout: 5 * time.Second,
		streams:     make(map[*Connection]struct{}),
	}, nil
}

// Endpoint 返回解析出的本地端点，供诊断记录使用。
func (s *Supervisor) Endpoint() transport.Endpoint { return s.ep.network }

// Owned 表示当前服务的实例是否由本进程拉起。
func (s *Supervisor) Owned() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.owned
}

// Connect 返回控制连接：先连接已有实例，端点缺失时拉起自己的实例。
// 版本不兼容是终态，之后不会再发起连接。
func (s *Supervisor) Connect(ctx context.Context) (*Connection, error) {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	if err := s.terminalError(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, failure.New(failure.KindNonRetryable, "connect", errors.New("连接监督者已关闭"))
	}
	current := s.conn
	s.mu.Unlock()
	if current != nil && current.Client.Err() == nil {
		return current, nil
	}

	conn, err := s.connectAndHello(ctx)
	if err == nil {
		s.setConnection(conn, s.ownsInstance(current, conn))
		return conn, nil
	}
	s.setTerminal(err)
	if !failure.Is(err, failure.KindConnectionMissing) {
		return nil, err
	}
	if launchErr := s.launch(); launchErr != nil {
		return nil, launchErr
	}
	conn, err = s.waitForDaemon(ctx)
	if err != nil {
		s.setTerminal(err)
		// 自己拉起但没能完成握手的实例不留在机器上，下次重新拉起。
		s.killOwnedChild()
		return nil, err
	}
	s.setConnection(conn, true)
	return conn, nil
}

// OpenStream 打开一条独立的、已握手的连接给会话流使用。
func (s *Supervisor) OpenStream(ctx context.Context) (*Connection, error) {
	if err := s.terminalError(); err != nil {
		return nil, err
	}
	if _, err := s.Connect(ctx); err != nil {
		return nil, err
	}
	conn, err := s.connectAndHello(ctx)
	if err != nil {
		s.setTerminal(err)
		if !failure.Is(err, failure.KindConnectionMissing) {
			return nil, err
		}
		// 端点消失：重连控制连接（自己拥有的实例会重新拉起），再试一次。
		if _, reconnectErr := s.Reconnect(ctx); reconnectErr != nil {
			return nil, reconnectErr
		}
		conn, err = s.connectAndHello(ctx)
		if err != nil {
			s.setTerminal(err)
			return nil, err
		}
	}
	conn.Owned = s.Owned()
	s.trackStream(conn)
	conn.Client.OnClose(func(error) { s.untrackStream(conn) })
	return conn, nil
}

// Reconnect 丢弃控制连接并重新握手；自己拥有的实例消失时重新拉起。
func (s *Supervisor) Reconnect(ctx context.Context) (ReconnectResult, error) {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	if err := s.terminalError(); err != nil {
		return ReconnectResult{}, err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ReconnectResult{}, failure.New(failure.KindNonRetryable, "reconnect", errors.New("连接监督者已关闭"))
	}
	previous := s.conn
	s.conn = nil
	s.mu.Unlock()
	if previous != nil {
		previous.Close(&protocol.ClosedError{Cause: errors.New("reconnect")})
	}

	conn, err := s.connectAndHello(ctx)
	if err == nil {
		restarted := previous != nil && previous.Hello.InstanceID != conn.Hello.InstanceID
		s.setConnection(conn, s.ownsInstance(previous, conn))
		if restarted {
			s.killOwnedChild()
		}
		return ReconnectResult{Connection: conn, Restarted: restarted}, nil
	}
	s.setTerminal(err)
	if failure.Is(err, failure.KindConnectionMissing) && s.hasChildHandle() {
		s.killOwnedChild()
		if launchErr := s.launch(); launchErr != nil {
			return ReconnectResult{}, launchErr
		}
		conn, err = s.waitForDaemon(ctx)
		if err == nil {
			s.setConnection(conn, true)
			return ReconnectResult{Connection: conn, Restarted: true}, nil
		}
		s.setTerminal(err)
	}
	return ReconnectResult{}, err
}

// Shutdown 只回收自己拉起的实例；外部启动的实例原样留下。
// 有其他客户端、已附加的会话或运行中的轮次时保留实例，并如实返回原因。
func (s *Supervisor) Shutdown(ctx context.Context) (ShutdownResult, error) {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	s.mu.Lock()
	s.closed = true
	conn := s.conn
	owned := s.owned
	s.mu.Unlock()
	if conn == nil {
		if s.killOwnedChild() {
			return ShutdownResult{Stopped: true, Reason: "killed"}, nil
		}
		return ShutdownResult{Reason: "not_connected"}, nil
	}
	if !owned {
		s.closeStreams()
		conn.Close(&protocol.ClosedError{Cause: errors.New("shutdown")})
		s.mu.Lock()
		s.conn = nil
		s.mu.Unlock()
		return ShutdownResult{Reason: "not_owned"}, nil
	}
	s.closeStreams()

	if conn.HasFeature("daemon.shutdown.ifIdle") {
		err := conn.Client.CallInto(ctx, "daemon.shutdown", protocol.ShutdownParams{Force: false, IfIdle: true}, nil)
		switch {
		case err == nil:
			s.reap(conn)
			return ShutdownResult{Stopped: true}, nil
		case appErrorOf(err) == "daemon_busy":
			return ShutdownResult{Reason: "busy", Connections: busyConnections(err)}, nil
		case appErrorOf(err) != "unknown_method":
			return ShutdownResult{}, failure.Wrap(failure.KindOf(err), "daemon.shutdown", err)
		}
	}

	// 旧服务端没有 ifIdle：先用 ping 确认没有别的客户端，再请求关闭。
	var ping protocol.PingResult
	if err := conn.Client.CallInto(ctx, "daemon.ping", struct{}{}, &ping); err != nil {
		if s.killOwnedChild() {
			return ShutdownResult{Stopped: true, Reason: "killed"}, nil
		}
		return ShutdownResult{}, failure.Wrap(failure.KindOf(err), "daemon.ping", err)
	}
	if ping.Connections > 1 {
		return ShutdownResult{Reason: "other_clients", Connections: ping.Connections}, nil
	}
	if err := conn.Client.CallInto(ctx, "daemon.shutdown", protocol.ShutdownParams{Force: false}, nil); err != nil {
		if s.killOwnedChild() {
			return ShutdownResult{Stopped: true, Reason: "killed"}, nil
		}
		return ShutdownResult{}, failure.Wrap(failure.KindOf(err), "daemon.shutdown", err)
	}
	s.reap(conn)
	return ShutdownResult{Stopped: true}, nil
}

// Close 关闭控制与流连接，但不回收服务实例（用于非自己拉起的实例）。
func (s *Supervisor) Close(reason error) {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	s.mu.Lock()
	s.closed = true
	conn := s.conn
	s.conn = nil
	s.owned = false
	s.mu.Unlock()
	if conn != nil {
		conn.Close(reason)
	}
	s.closeStreams()
}

// childAlive 报告自己拉起的子进程是否仍在运行。
func (s *Supervisor) childAlive() bool {
	s.mu.Lock()
	child := s.child
	s.mu.Unlock()
	if child == nil {
		return false
	}
	select {
	case <-child.Exited():
		return false
	default:
		return true
	}
}

// hasChildHandle 报告是否存在自启实例的句柄（进程可能已经退出）。
func (s *Supervisor) hasChildHandle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.child != nil
}

// ownsInstance 判断一条新连接背后是否还是自己拉起的那个实例。
func (s *Supervisor) ownsInstance(previous, conn *Connection) bool {
	if !s.childAlive() {
		return false
	}
	if previous != nil && previous.Hello.InstanceID != conn.Hello.InstanceID {
		s.killOwnedChild()
		return false
	}
	return true
}

func (s *Supervisor) terminalError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.terminal
}

// setTerminal 只把版本不兼容记为终态；其他错误允许下次重连再看。
func (s *Supervisor) setTerminal(err error) {
	if err == nil || !failure.Is(err, failure.KindVersionIncompatible) {
		return
	}
	s.mu.Lock()
	if s.terminal == nil {
		s.terminal = err
	}
	s.mu.Unlock()
}

func (s *Supervisor) connectAndHello(ctx context.Context) (*Connection, error) {
	raw, err := s.Dial(ctx, s.ep.network, s.cfg.ConnectTimeout)
	if err != nil {
		return nil, failure.Wrap(failure.KindOf(err), "dial", err)
	}
	client := protocol.NewClient(raw, protocol.Options{RequestTimeout: s.cfg.RequestTimeout})
	hello, err := handshake(ctx, client, s.cfg.ClientName)
	if err != nil {
		client.Close(err)
		return nil, err
	}
	return &Connection{Client: client, Hello: hello, Endpoint: s.ep.network}, nil
}

func (s *Supervisor) launch() error {
	s.mu.Lock()
	existing := s.child
	s.mu.Unlock()
	if existing != nil {
		select {
		case <-existing.Exited():
			// 自启实例已退出：清掉句柄后重新拉起，own 的是一条新的实例。
			s.mu.Lock()
			if s.child == existing {
				s.child = nil
			}
			s.mu.Unlock()
		default:
			return nil
		}
	}
	child, err := s.Spawn(s.cfg, SpawnRequest{NativeHome: s.ep.nativeHome, Endpoint: s.ep.network})
	if err != nil {
		return failure.Wrap(failure.KindNonRetryable, "spawn daemon", err)
	}
	s.mu.Lock()
	s.child = child
	s.mu.Unlock()
	return nil
}

func (s *Supervisor) waitForDaemon(ctx context.Context) (*Connection, error) {
	deadline := time.Now().Add(s.cfg.SpawnTimeout)
	var last error
	for {
		conn, err := s.connectAndHello(ctx)
		if err == nil {
			return conn, nil
		}
		last = err
		if !failure.Is(err, failure.KindConnectionMissing) {
			return nil, err
		}
		s.mu.Lock()
		child := s.child
		s.mu.Unlock()
		if child != nil {
			select {
			case <-child.Exited():
				return nil, failure.New(
					failure.KindNonRetryable,
					"等待自启 App Server",
					fmt.Errorf("服务进程在端点就绪前退出: %w", last),
				)
			default:
			}
		}
		if time.Now().After(deadline) {
			return nil, failure.New(
				failure.KindTimeout,
				"等待自启 App Server",
				fmt.Errorf("在 %s 内端点未就绪: %w", s.cfg.SpawnTimeout, last),
			)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (s *Supervisor) setConnection(conn *Connection, owned bool) {
	s.mu.Lock()
	previous := s.conn
	s.conn = conn
	s.owned = owned
	if conn != nil {
		conn.Owned = owned
	}
	s.mu.Unlock()
	if previous != nil && previous != conn {
		previous.Close(&protocol.ClosedError{Cause: errors.New("connection replaced")})
	}
}

func (s *Supervisor) trackStream(conn *Connection) {
	s.mu.Lock()
	s.streams[conn] = struct{}{}
	s.mu.Unlock()
}

func (s *Supervisor) untrackStream(conn *Connection) {
	s.mu.Lock()
	delete(s.streams, conn)
	s.mu.Unlock()
}

func (s *Supervisor) closeStreams() {
	s.mu.Lock()
	streams := make([]*Connection, 0, len(s.streams))
	for conn := range s.streams {
		streams = append(streams, conn)
	}
	s.streams = make(map[*Connection]struct{})
	s.mu.Unlock()
	for _, conn := range streams {
		conn.Close(&protocol.ClosedError{Cause: errors.New("supervisor closing")})
	}
}

// killOwnedChild 只杀自己拉起的进程，返回是否真的有一个自启实例。
func (s *Supervisor) killOwnedChild() bool {
	s.mu.Lock()
	child := s.child
	s.child = nil
	s.mu.Unlock()
	if child == nil {
		return false
	}
	_ = child.Kill()
	select {
	case <-child.Exited():
	case <-time.After(s.StopTimeout):
	}
	return true
}

// reap 关闭连接并等待自启实例退出；超时后只杀自己拉起的进程。
func (s *Supervisor) reap(conn *Connection) {
	conn.Close(&protocol.ClosedError{Cause: errors.New("shutdown")})
	s.mu.Lock()
	s.conn = nil
	s.owned = false
	child := s.child
	s.mu.Unlock()
	if child == nil {
		return
	}
	select {
	case <-child.Exited():
	case <-time.After(s.StopTimeout):
		_ = child.Kill()
		select {
		case <-child.Exited():
		case <-time.After(s.StopTimeout):
		}
	}
	s.mu.Lock()
	if s.child == child {
		s.child = nil
	}
	s.mu.Unlock()
}

func appErrorOf(err error) string {
	var rpcErr *protocol.RPCError
	if errors.As(err, &rpcErr) {
		return rpcErr.AppError()
	}
	return ""
}

func busyConnections(err error) int {
	var rpcErr *protocol.RPCError
	if !errors.As(err, &rpcErr) || len(rpcErr.Data) == 0 {
		return 0
	}
	var data struct {
		Connections int `json:"connections"`
	}
	if jsonErr := json.Unmarshal(rpcErr.Data, &data); jsonErr != nil {
		return 0
	}
	return data.Connections
}
