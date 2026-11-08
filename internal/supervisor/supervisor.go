// Package supervisor resolves the protocol-v3 daemon endpoint, connects and
// handshakes each connection, spawns the daemon when and only when it is
// absent, and stops only a daemon this process started.
//
// The Rust dim-app-server remains the only source of truth. This package keeps
// no session, message, run, or approval state; it only owns transport lifecycle
// so a later session-stream layer can issue session.* calls.
package supervisor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// DefaultClientName is the hello clientName this Go client advertises.
const DefaultClientName = "dimagent-mygo"

// Close report reasons, matching the Desktop supervisor vocabulary.
const (
	CloseSkippedNotOwned      = "not_owned"
	CloseSkippedNotConnected  = "not_connected"
	CloseSkippedOtherClients  = "other_clients"
	CloseSkippedAlreadyClosed = "already_closed"
)

var (
	// ErrClosed is returned once Close has begun; a closing supervisor refuses
	// new connections.
	ErrClosed = errors.New("supervisor: supervisor is closed")
	// ErrNotStarted is returned by OpenStream before Start has resolved a link.
	ErrNotStarted = errors.New("supervisor: supervisor has not started")
)

// Config configures one Supervisor. ProductHome is required; ServerPath and
// the daemon profile fields are only used when spawning.
type Config struct {
	ProductHome string
	// ServerPath is the dim-app-server binary. Empty falls back to
	// `dim-app-server` on PATH when a spawn is needed.
	ServerPath string
	// ClientName is sent in hello; empty uses DefaultClientName.
	ClientName string

	MockModel       bool
	DefaultsPath    string
	ModelConfigPath string

	// Dial and Spawn are injectable; nil uses the real socket/pipe and exec
	// implementations.
	Dial  DialFunc
	Spawn SpawnFunc

	ListenWait ListenWait
	CloseGrace CloseGrace
}

// CloseGrace bounds the daemon.ping retries Close performs while stream
// connections are torn down (Desktop uses 10 attempts at 100 ms).
type CloseGrace struct {
	Attempts int
	Interval time.Duration
}

var DefaultCloseGrace = CloseGrace{Attempts: 10, Interval: 100 * time.Millisecond}

func (g CloseGrace) normalized() CloseGrace {
	if g.Attempts <= 0 {
		g.Attempts = DefaultCloseGrace.Attempts
	}
	if g.Interval <= 0 {
		g.Interval = DefaultCloseGrace.Interval
	}
	return g
}

// CloseReport records what Close decided. Connections is meaningful only when
// Pinged is true.
type CloseReport struct {
	Owned       bool
	Pinged      bool
	Connections int
	Shutdown    bool
	Skipped     string
}

// Supervisor owns daemon discovery, connection handshakes, and exit rules.
type Supervisor struct {
	cfg        Config
	hello      appserver.HelloParams
	dial       DialFunc
	spawn      SpawnFunc
	listenWait ListenWait
	closeGrace CloseGrace

	// startMu serializes Start against Close so a spawn cannot race a close.
	startMu sync.Mutex

	mu      sync.Mutex
	link    Link
	linked  bool
	control *Connection
	conns   map[*Connection]struct{}
	owned   bool
	process Process
	spawned bool
	closing bool
	closed  bool
}

// New validates cfg and builds a supervisor. Endpoint resolution (which may
// create `<productHome>/rust`) happens on Start.
func New(cfg Config) (*Supervisor, error) {
	if strings.TrimSpace(cfg.ProductHome) == "" {
		return nil, errors.New("supervisor: productHome must not be empty")
	}
	name := cfg.ClientName
	if name == "" {
		name = DefaultClientName
	}
	hello := appserver.HelloParams{
		ProtocolVersion: appserver.WireProtocolVersion,
		ClientKind:      appserver.ClientKindOther,
		ClientName:      name,
	}
	if err := hello.Validate(); err != nil {
		return nil, fmt.Errorf("supervisor: hello params: %w", err)
	}
	dial := cfg.Dial
	if dial == nil {
		dial = DefaultDial
	}
	spawn := cfg.Spawn
	if spawn == nil {
		spawn = DefaultSpawn
	}
	return &Supervisor{
		cfg:        cfg,
		hello:      hello,
		dial:       dial,
		spawn:      spawn,
		listenWait: cfg.ListenWait.normalized(),
		closeGrace: cfg.CloseGrace.normalized(),
		conns:      make(map[*Connection]struct{}),
	}, nil
}

// Start returns the control connection, connecting to an existing daemon or
// spawning one only when the endpoint is absent. A hello or non-missing dial
// failure is surfaced without spawning.
func (s *Supervisor) Start(ctx context.Context) (*Connection, error) {
	s.mu.Lock()
	if s.closing || s.closed {
		s.mu.Unlock()
		return nil, ErrClosed
	}
	if s.control != nil {
		control := s.control
		s.mu.Unlock()
		return control, nil
	}
	s.mu.Unlock()

	s.startMu.Lock()
	defer s.startMu.Unlock()

	s.mu.Lock()
	if s.closing || s.closed {
		s.mu.Unlock()
		return nil, ErrClosed
	}
	if s.control != nil {
		control := s.control
		s.mu.Unlock()
		return control, nil
	}
	link, linked := s.link, s.linked
	s.mu.Unlock()

	if !linked {
		resolved, err := ResolveLink(s.cfg.ProductHome)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.link = resolved
		s.linked = true
		s.mu.Unlock()
		link = resolved
	}

	client, hello, err := s.dialHello(ctx, link.Endpoint)
	if err == nil {
		return s.registerControl(client, hello)
	}
	if !IsMissingEndpoint(err) {
		return nil, err
	}

	s.mu.Lock()
	alreadySpawned := s.spawned
	s.mu.Unlock()
	if alreadySpawned {
		return nil, fmt.Errorf("supervisor: owned daemon at %s is not listening: %w", link.Endpoint, err)
	}
	if err := s.spawnDaemon(ctx, link); err != nil {
		return nil, err
	}
	if err := waitForListen(ctx, link.Endpoint, s.dial, s.listenWait); err != nil {
		return nil, err
	}
	client, hello, err = s.dialHello(ctx, link.Endpoint)
	if err != nil {
		return nil, err
	}
	return s.registerControl(client, hello)
}

// OpenStream opens one additional hello-authenticated connection. Session
// attach is deliberately not implemented here.
func (s *Supervisor) OpenStream(ctx context.Context) (*Connection, error) {
	// Share the start/close serialization so Close cannot ping while a stream
	// is still being established and counted.
	s.startMu.Lock()
	defer s.startMu.Unlock()

	s.mu.Lock()
	if s.closing || s.closed {
		s.mu.Unlock()
		return nil, ErrClosed
	}
	link, linked := s.link, s.linked
	s.mu.Unlock()
	if !linked {
		return nil, ErrNotStarted
	}
	client, hello, err := s.dialHello(ctx, link.Endpoint)
	if err != nil {
		return nil, err
	}
	conn := &Connection{sup: s, client: client, hello: hello}
	s.mu.Lock()
	if s.closing || s.closed {
		s.mu.Unlock()
		client.Close()
		return nil, ErrClosed
	}
	s.conns[conn] = struct{}{}
	s.mu.Unlock()
	return conn, nil
}

// Close closes this supervisor's connections and, when this process spawned
// the daemon, stops it after daemon.ping reports exactly one connection. A
// daemon found already listening is never stopped.
func (s *Supervisor) Close(ctx context.Context) (CloseReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.startMu.Lock()
	defer s.startMu.Unlock()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return CloseReport{Skipped: CloseSkippedAlreadyClosed}, nil
	}
	s.closing = true
	owned := s.owned
	control := s.control
	others := make([]*Connection, 0, len(s.conns))
	for conn := range s.conns {
		if conn != control {
			others = append(others, conn)
		}
	}
	s.mu.Unlock()

	// Streams this supervisor opened are not other clients; close them before
	// asking the daemon how many remain.
	for _, conn := range others {
		conn.Close()
	}

	report := CloseReport{Owned: owned}
	if !owned {
		if control != nil {
			control.Close()
		}
		s.finishClose()
		report.Skipped = CloseSkippedNotOwned
		return report, nil
	}
	if control == nil {
		s.finishClose()
		report.Skipped = CloseSkippedNotConnected
		return report, nil
	}
	defer control.Close()

	var ping PingResult
	var err error
	for attempt := 0; attempt < s.closeGrace.Attempts; attempt++ {
		if attempt > 0 {
			if err := sleepContext(ctx, s.closeGrace.Interval); err != nil {
				s.finishClose()
				return report, err
			}
		}
		ping, err = s.ping(ctx, control)
		if err != nil {
			s.finishClose()
			return report, fmt.Errorf("supervisor: daemon.ping: %w", err)
		}
		report.Pinged = true
		report.Connections = *ping.Connections
		if report.Connections == 1 {
			break
		}
	}
	if report.Connections != 1 {
		s.finishClose()
		report.Skipped = CloseSkippedOtherClients
		return report, nil
	}
	if _, err := control.client.Call(ctx, "daemon.shutdown", ShutdownParams{Force: false}); err != nil {
		s.finishClose()
		return report, fmt.Errorf("supervisor: daemon.shutdown: %w", err)
	}
	report.Shutdown = true
	s.mu.Lock()
	s.owned = false
	s.process = nil
	s.mu.Unlock()
	s.finishClose()
	return report, nil
}

// Endpoint reports the resolved endpoint after Start has run.
func (s *Supervisor) Endpoint() (Endpoint, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.linked {
		return Endpoint{}, false
	}
	return s.link.Endpoint, true
}

// OwnsDaemon reports whether this supervisor spawned the current daemon.
func (s *Supervisor) OwnsDaemon() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.owned
}

// ActiveConnections is the number of hello-authenticated connections this
// supervisor still tracks.
func (s *Supervisor) ActiveConnections() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.conns)
}

// DaemonPID reports the pid of the daemon this supervisor spawned, when it
// still owns one. It is diagnostic only; lifecycle decisions use daemon.ping.
func (s *Supervisor) DaemonPID() (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.process == nil {
		return 0, false
	}
	return s.process.Pid(), true
}

// Control returns the control connection created by Start.
func (s *Supervisor) Control() (*Connection, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.control, s.control != nil
}

func (s *Supervisor) dialHello(ctx context.Context, endpoint Endpoint) (*appserver.Client, appserver.HelloResult, error) {
	return DialWith(ctx, endpoint, s.hello, s.dial)
}

func (s *Supervisor) registerControl(client *appserver.Client, hello appserver.HelloResult) (*Connection, error) {
	conn := &Connection{sup: s, client: client, hello: hello}
	s.mu.Lock()
	if s.closing || s.closed {
		s.mu.Unlock()
		client.Close()
		return nil, ErrClosed
	}
	s.control = conn
	s.conns[conn] = struct{}{}
	s.mu.Unlock()
	return conn, nil
}

func (s *Supervisor) spawnDaemon(ctx context.Context, link Link) error {
	cfg := SpawnConfig{
		ServerPath:      s.cfg.ServerPath,
		Endpoint:        link.Endpoint,
		NativeHome:      link.NativeHome,
		ProductHome:     link.ProductHome,
		MockModel:       s.cfg.MockModel,
		DefaultsPath:    s.cfg.DefaultsPath,
		ModelConfigPath: s.cfg.ModelConfigPath,
	}
	process, err := s.spawn(ctx, cfg)
	if err != nil {
		return fmt.Errorf("supervisor: spawn app server: %w", err)
	}
	s.mu.Lock()
	s.spawned = true
	s.owned = true
	s.process = process
	s.mu.Unlock()
	return nil
}

func (s *Supervisor) ping(ctx context.Context, conn *Connection) (PingResult, error) {
	raw, err := conn.client.Call(ctx, "daemon.ping", struct{}{})
	if err != nil {
		return PingResult{}, err
	}
	return DecodePingResult(raw)
}

func (s *Supervisor) finishClose() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
}

func (s *Supervisor) release(conn *Connection) {
	s.mu.Lock()
	delete(s.conns, conn)
	if s.control == conn {
		s.control = nil
	}
	s.mu.Unlock()
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Connection is one hello-authenticated connection. Closing it releases the
// supervisor's tracking; closing the control connection also means Close can
// no longer shut down an owned daemon on that connection.
type Connection struct {
	sup       *Supervisor
	client    *appserver.Client
	hello     appserver.HelloResult
	closeOnce sync.Once
}

func (c *Connection) Client() *appserver.Client { return c.client }

func (c *Connection) Hello() appserver.HelloResult { return c.hello }

func (c *Connection) InstanceID() string { return c.hello.InstanceID }

func (c *Connection) Features() []string {
	return append([]string(nil), c.hello.Features...)
}

func (c *Connection) HasFeature(name string) bool { return c.hello.HasFeature(name) }

func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		c.client.Close()
		if c.sup != nil {
			c.sup.release(c)
		}
	})
}
