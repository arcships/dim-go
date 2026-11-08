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
	"sync/atomic"
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

	ListenWait       ListenWait
	CloseGrace       CloseGrace
	ReconnectBackoff ReconnectBackoff
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

// ReconnectBackoff bounds the reconnect retry series. Only a missing endpoint
// (ENOENT/ECONNREFUSED) is retried; every other dial error and every hello
// failure is surfaced immediately. Attempts counts connect+hello tries,
// including the first.
type ReconnectBackoff struct {
	Attempts int
	Initial  time.Duration
	Max      time.Duration
}

// DefaultReconnectBackoff retries a restarted daemon for roughly three
// seconds: 100 ms, 200 ms, 400 ms, 800 ms, 1.6 s, then the final attempt.
var DefaultReconnectBackoff = ReconnectBackoff{
	Attempts: 6,
	Initial:  100 * time.Millisecond,
	Max:      2 * time.Second,
}

func (b ReconnectBackoff) normalized() ReconnectBackoff {
	if b.Attempts <= 0 {
		b.Attempts = DefaultReconnectBackoff.Attempts
	}
	if b.Initial <= 0 {
		b.Initial = DefaultReconnectBackoff.Initial
	}
	if b.Max <= 0 {
		b.Max = DefaultReconnectBackoff.Max
	}
	if b.Max < b.Initial {
		b.Max = b.Initial
	}
	return b
}

// ReconnectResult describes one successful control reconnect. The session
// layer compares Generation with older connections to decide which attachments
// must be rebuilt; this package never reopens sessions itself.
type ReconnectResult struct {
	Connection         *Connection
	PreviousInstanceID string
	NewInstanceID      string
	Restarted          bool
	Generation         uint64
}

// reconnectCall is the single-flight state shared by concurrent Reconnect
// calls. waiters is count-only bookkeeping exposed to package tests.
type reconnectCall struct {
	done    chan struct{}
	conn    *Connection
	err     error
	waiters int
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
	cfg              Config
	hello            appserver.HelloParams
	dial             DialFunc
	spawn            SpawnFunc
	listenWait       ListenWait
	closeGrace       CloseGrace
	reconnectBackoff ReconnectBackoff

	// startMu serializes Start, OpenStream, Reconnect, and Close so connection
	// establish/teardown cannot race each other.
	startMu sync.Mutex

	mu           sync.Mutex
	link         Link
	linked       bool
	control      *Connection
	conns        map[*Connection]struct{}
	owned        bool
	process      Process
	spawned      bool
	closing      bool
	closed       bool
	reconnecting *reconnectCall
	instanceID   string
	seenInstance bool
	generation   uint64
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
		cfg:              cfg,
		hello:            hello,
		dial:             dial,
		spawn:            spawn,
		listenWait:       cfg.ListenWait.normalized(),
		closeGrace:       cfg.CloseGrace.normalized(),
		reconnectBackoff: cfg.ReconnectBackoff.normalized(),
		conns:            make(map[*Connection]struct{}),
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

	dialed, err := s.dialHello(ctx, link.Endpoint)
	if err == nil {
		return s.newConnection(dialed, true)
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
	dialed, err = s.dialHello(ctx, link.Endpoint)
	if err != nil {
		return nil, err
	}
	return s.newConnection(dialed, true)
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
	dialed, err := s.dialHello(ctx, link.Endpoint)
	if err != nil {
		return nil, err
	}
	return s.newConnection(dialed, false)
}

// Reconnect re-resolves the endpoint and replaces the control connection.
// Concurrent calls are coalesced: one attempt runs and every caller receives
// its connection or error. A closing or closed supervisor refuses reconnect.
//
// A missing endpoint is retried with bounded exponential backoff. When this
// supervisor owns the daemon it may spawn exactly one replacement during the
// attempt; a non-owned daemon is only retried, never spawned. Every non-missing
// dial error and every hello failure is surfaced without retry or spawn.
//
// Reconnect closes all previously tracked connections first; session
// attachments are not reopened. A restart is visible through Connection
// Generation/Restarted or the ReconnectResult metadata.
func (s *Supervisor) Reconnect(ctx context.Context) (*Connection, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	if s.closing || s.closed {
		s.mu.Unlock()
		return nil, ErrClosed
	}
	if !s.linked {
		s.mu.Unlock()
		return nil, ErrNotStarted
	}
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if call := s.reconnecting; call != nil {
		call.waiters++
		s.mu.Unlock()
		select {
		case <-call.done:
			return call.conn, call.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	call := &reconnectCall{done: make(chan struct{})}
	s.reconnecting = call
	s.mu.Unlock()

	conn, err := s.reconnectOnce(ctx)

	s.mu.Lock()
	call.conn, call.err = conn, err
	s.mu.Unlock()
	close(call.done)
	// Clear only after the outcome is published: a Reconnect arriving in this
	// window joins the completed call instead of starting a redundant attempt.
	s.mu.Lock()
	if s.reconnecting == call {
		s.reconnecting = nil
	}
	s.mu.Unlock()
	return conn, err
}

// ReconnectResult reconnects and returns the restart metadata for the new
// control connection. Reconnect is the primary API; this convenience keeps the
// instanceId decision in one value for callers that need it immediately.
func (s *Supervisor) ReconnectResult(ctx context.Context) (ReconnectResult, error) {
	conn, err := s.Reconnect(ctx)
	if err != nil {
		return ReconnectResult{}, err
	}
	return conn.ReconnectResult(), nil
}

func (s *Supervisor) reconnectOnce(ctx context.Context) (*Connection, error) {
	s.startMu.Lock()
	defer s.startMu.Unlock()

	s.mu.Lock()
	if s.closing || s.closed {
		s.mu.Unlock()
		return nil, ErrClosed
	}
	if !s.linked {
		s.mu.Unlock()
		return nil, ErrNotStarted
	}
	s.mu.Unlock()

	// Re-resolve on every reconnect: a moved product home or a replaced
	// symlink must address the same endpoint the daemon will publish.
	link, err := ResolveLink(s.cfg.ProductHome)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.link = link
	s.linked = true
	s.mu.Unlock()

	// Old attachments are invalid once a control reconnect begins.
	s.closeAllConnections()

	backoff := s.reconnectBackoff
	delay := backoff.Initial
	respawned := false
	attempts := 0
	var lastErr error
	for {
		if s.isClosing() {
			return nil, ErrClosed
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		attempts++
		dialed, err := s.dialHello(ctx, link.Endpoint)
		if err == nil {
			return s.newConnection(dialed, true)
		}
		lastErr = err
		if !IsMissingEndpoint(err) {
			return nil, err
		}

		s.mu.Lock()
		owned := s.owned
		s.mu.Unlock()
		if owned && !respawned {
			respawned = true
			if err := s.spawnDaemon(ctx, link); err != nil {
				return nil, err
			}
			if err := waitForListen(ctx, link.Endpoint, s.dial, s.listenWait); err != nil {
				return nil, err
			}
			// The spawn consumed no retry budget; redial immediately.
			continue
		}

		if attempts >= backoff.Attempts {
			break
		}
		if err := sleepContext(ctx, delay); err != nil {
			return nil, err
		}
		delay *= 2
		if delay > backoff.Max {
			delay = backoff.Max
		}
	}
	return nil, fmt.Errorf("supervisor: reconnect to %s failed after %d attempts: %w", link.Endpoint, attempts, lastErr)
}

// closeAllConnections closes every tracked connection, including the control
// connection, so stale attachments cannot outlive a reconnect.
func (s *Supervisor) closeAllConnections() {
	s.mu.Lock()
	conns := make([]*Connection, 0, len(s.conns))
	for conn := range s.conns {
		conns = append(conns, conn)
	}
	s.mu.Unlock()
	for _, conn := range conns {
		conn.Close()
	}
}

// Close closes this supervisor's connections and, when this process spawned
// the daemon, stops it after daemon.ping reports exactly one connection. A
// daemon found already listening is never stopped.
func (s *Supervisor) Close(ctx context.Context) (CloseReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// Refuse new starts/reconnects immediately; an in-flight attempt is
	// serialized below and will fail its post-dial closing check.
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return CloseReport{Skipped: CloseSkippedAlreadyClosed}, nil
	}
	s.closing = true
	s.mu.Unlock()

	s.startMu.Lock()
	defer s.startMu.Unlock()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return CloseReport{Skipped: CloseSkippedAlreadyClosed}, nil
	}
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

func (s *Supervisor) dialHello(ctx context.Context, endpoint Endpoint) (*dialed, error) {
	return dialConn(ctx, endpoint, s.hello, s.dial)
}

// newConnection registers a hello-authenticated connection. It also advances
// the daemon generation when the observed instanceId changes.
func (s *Supervisor) newConnection(dialed *dialed, control bool) (*Connection, error) {
	if dialed == nil || dialed.client == nil {
		return nil, errors.New("supervisor: nil connection")
	}
	s.mu.Lock()
	if s.closing || s.closed {
		s.mu.Unlock()
		dialed.client.Close()
		return nil, ErrClosed
	}
	generation, previous, restarted := s.observeInstanceLocked(dialed.hello.InstanceID)
	conn := &Connection{
		sup:                s,
		client:             dialed.client,
		watch:              dialed.watch,
		hello:              dialed.hello,
		generation:         generation,
		previousInstanceID: previous,
		restarted:          restarted,
	}
	if control {
		s.control = conn
	}
	s.conns[conn] = struct{}{}
	s.mu.Unlock()
	return conn, nil
}

// observeInstanceLocked records a hello instanceId. The first observation is
// generation 1 without a restart; any later different instanceId increments the
// generation and marks every older connection stale.
func (s *Supervisor) observeInstanceLocked(instanceID string) (generation uint64, previous string, restarted bool) {
	if !s.seenInstance {
		s.seenInstance = true
		s.instanceID = instanceID
		s.generation = 1
		return s.generation, "", false
	}
	if instanceID == s.instanceID {
		return s.generation, s.instanceID, false
	}
	previous = s.instanceID
	s.instanceID = instanceID
	s.generation++
	return s.generation, previous, true
}

// Generation is the current daemon generation. It starts at 1 after the first
// successful hello and increments whenever a hello reports a different
// instanceId. A Connection with an older generation is stale and its session
// attachments must be rebuilt by the session layer.
func (s *Supervisor) Generation() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.generation
}

// InstanceID is the most recently observed daemon instanceId.
func (s *Supervisor) InstanceID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.instanceID
}

func (s *Supervisor) isClosing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closing || s.closed
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
	sup                *Supervisor
	client             *appserver.Client
	watch              *watchedConn
	hello              appserver.HelloResult
	generation         uint64
	previousInstanceID string
	restarted          bool
	closedByUs         atomic.Bool
	closeOnce          sync.Once
}

func (c *Connection) Client() *appserver.Client { return c.client }

func (c *Connection) Hello() appserver.HelloResult { return c.hello }

func (c *Connection) InstanceID() string { return c.hello.InstanceID }

func (c *Connection) Features() []string {
	return append([]string(nil), c.hello.Features...)
}

func (c *Connection) HasFeature(name string) bool { return c.hello.HasFeature(name) }

// Done is the wire client's Done channel: it closes when the transport ends or
// the connection is closed. A session layer treats an unexpected close as the
// domain resync trigger; session.resync is not a wire method.
func (c *Connection) Done() <-chan struct{} { return c.client.Done() }

// Closed reports whether Done has fired.
func (c *Connection) Closed() bool {
	select {
	case <-c.client.Done():
		return true
	default:
		return false
	}
}

// Err is the terminal reason once Done has fired, and nil while the
// connection is still live. A remotely lost transport reports the underlying
// read/write error (for example io.EOF); a local close reports
// ErrConnectionClosed.
func (c *Connection) Err() error {
	select {
	case <-c.client.Done():
	default:
		return nil
	}
	if c.watch != nil {
		if err := c.watch.Err(); err != nil {
			return err
		}
	}
	if c.closedByUs.Load() {
		return ErrConnectionClosed
	}
	return ErrTransportLost
}

// Generation is the daemon generation observed by this connection's hello.
// Compare it with Supervisor.Generation to detect a restart.
func (c *Connection) Generation() uint64 { return c.generation }

// PreviousInstanceID is the instanceId observed before this hello. It is empty
// for the first connection and stays unchanged when no restart occurred.
func (c *Connection) PreviousInstanceID() string { return c.previousInstanceID }

// Restarted reports whether this hello observed a different instanceId than the
// previously observed daemon.
func (c *Connection) Restarted() bool { return c.restarted }

// Stale reports whether the supervisor has observed a newer daemon generation
// since this connection was established.
func (c *Connection) Stale() bool {
	if c.sup == nil {
		return false
	}
	return c.generation < c.sup.Generation()
}

// ReconnectResult returns this connection's restart metadata. It is also
// available for connections created by Start and OpenStream.
func (c *Connection) ReconnectResult() ReconnectResult {
	return ReconnectResult{
		Connection:         c,
		PreviousInstanceID: c.previousInstanceID,
		NewInstanceID:      c.hello.InstanceID,
		Restarted:          c.restarted,
		Generation:         c.generation,
	}
}

func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		c.closedByUs.Store(true)
		c.client.Close()
		if c.sup != nil {
			c.sup.release(c)
		}
	})
}
