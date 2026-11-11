// Package rutis implements the minimal rutis session a UI leaf needs.
//
// A UI leaf is the far end of a rutis session with the App Server: it binds a
// private address, accepts the daemon's one connection, greets with its
// endpoint id, and then keeps the session open. Business traffic stays on
// protocol v3 (the leaf's ui.attach and ordinary requests); this package only
// speaks the handshake and the newline-framed session channel.
//
// It deliberately implements only what a UI leaf without a plugin catalog
// needs: no plugin definitions, no service offers, no inbound calls beyond a
// refusal. Anything the daemon sends that expects an answer is refused with a
// BindingError; anything else is ignored until the channel closes.
package rutis

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"
)

// ProtocolVersion is the rutis session protocol this package speaks.
const ProtocolVersion = 4

// MaxFrameBytes is the largest newline-framed message either way.
const MaxFrameBytes = 16 * 1024 * 1024

// HandshakeTimeout bounds the daemon's greeting after it connects.
const HandshakeTimeout = 10 * time.Second

var (
	// ErrIncompatible is returned for a wrong protocol version or a malformed
	// greeting.
	ErrIncompatible = errors.New("rutis: incompatible handshake")
	// ErrExpectedPeer is returned when the far end greets as another id.
	ErrExpectedPeer = errors.New("rutis: unexpected far endpoint")
	// ErrClosed is returned when the session ended.
	ErrClosed = errors.New("rutis: session closed")
)

// FrameError is a framing failure (over the limit, invalid UTF-8, or a raw
// newline in a sent message).
type FrameError struct {
	Message string
}

func (e *FrameError) Error() string { return "rutis: " + e.Message }

// Hello is the greeting each end sends first.
type Hello struct {
	Op             string            `json:"op"`
	Version        int               `json:"version"`
	Endpoint       string            `json:"endpoint"`
	Implementation map[string]string `json:"implementation,omitempty"`
	Capabilities   []string          `json:"capabilities,omitempty"`
}

// Greeting is what the far end said of itself.
type Greeting struct {
	Endpoint       string
	Implementation map[string]string
	Capabilities   []string
}

// Has reports whether the far end declared a capability.
func (g Greeting) Has(capability string) bool {
	for _, candidate := range g.Capabilities {
		if candidate == capability {
			return true
		}
	}
	return false
}

// Leaf is a bound UI-leaf address and its one session.
type Leaf struct {
	peer     string
	address  string
	listener net.Listener

	mu        sync.Mutex
	conn      net.Conn
	greeting  Greeting
	sentBytes int
	recvBytes int
	closeErr  error
	done      chan struct{}
	closeOnce sync.Once
}

// Listen binds a private unix socket named ui-<8 hex> under dir and returns a
// leaf whose endpoint is ui-<8 hex>. The directory should be private (0700),
// as the App Server's native home is.
func Listen(dir string) (*Leaf, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("rutis: prepare leaf directory: %w", err)
	}
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		id, err := randomHex(8)
		if err != nil {
			return nil, err
		}
		path := filepath.Join(dir, "ui-"+id)
		listener, err := net.Listen("unix", path)
		if err != nil {
			lastErr = err
			continue
		}
		return &Leaf{
			peer:     "ui-" + id,
			address:  "unix:" + path,
			listener: listener,
			done:     make(chan struct{}),
		}, nil
	}
	return nil, fmt.Errorf("rutis: bind leaf socket: %w", lastErr)
}

// Peer is this leaf's endpoint id, the `peer` argument of ui.open.
func (l *Leaf) Peer() string { return l.peer }

// Address is the `address` argument of ui.open.
func (l *Leaf) Address() string { return l.address }

// Done closes when the accepted session ends or Close is called.
func (l *Leaf) Done() <-chan struct{} { return l.done }

// HandshakeSizes is the byte length of this leaf's greeting and of the far
// end's greeting, without their newlines. It is 0 before the handshake.
func (l *Leaf) HandshakeSizes() (sent, received int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.sentBytes, l.recvBytes
}

// Greeting is the far end's greeting, once the handshake succeeded.
func (l *Leaf) Greeting() Greeting {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.greeting
}

// Serve accepts exactly one connection, performs the handshake with the
// expected endpoint (dim-daemon by default), then keeps the session open until
// the channel closes, Serve is cancelled, or Close is called.
func (l *Leaf) Serve(ctx context.Context, expectedPeer string) error {
	conn, err := l.accept(ctx)
	if err != nil {
		l.finish(err)
		return err
	}
	l.mu.Lock()
	l.conn = conn
	l.mu.Unlock()
	reader := bufio.NewReaderSize(conn, 64*1024)
	if err := l.handshake(conn, reader, expectedPeer); err != nil {
		l.finish(err)
		_ = conn.Close()
		return err
	}
	err = l.readLoop(ctx, conn, reader)
	l.finish(err)
	_ = conn.Close()
	return err
}

// Close ends the session and releases the socket.
func (l *Leaf) Close() error {
	var err error
	l.closeOnce.Do(func() {
		l.mu.Lock()
		conn := l.conn
		l.closeErr = ErrClosed
		l.mu.Unlock()
		if conn != nil {
			err = conn.Close()
		}
		if l.listener != nil {
			if closeErr := l.listener.Close(); closeErr != nil && err == nil && !errors.Is(closeErr, net.ErrClosed) {
				err = closeErr
			}
		}
		_ = os.Remove(socketPath(l.address))
		close(l.done)
	})
	return err
}

func (l *Leaf) accept(ctx context.Context) (net.Conn, error) {
	type accepted struct {
		conn net.Conn
		err  error
	}
	result := make(chan accepted, 1)
	go func() {
		conn, err := l.listener.Accept()
		result <- accepted{conn: conn, err: err}
	}()
	select {
	case <-ctx.Done():
		_ = l.listener.Close()
		return nil, ctx.Err()
	case outcome := <-result:
		if outcome.err != nil {
			return nil, fmt.Errorf("rutis: accept: %w", outcome.err)
		}
		return outcome.conn, nil
	}
}

func (l *Leaf) handshake(conn net.Conn, reader *bufio.Reader, expectedPeer string) error {
	if expectedPeer == "" {
		expectedPeer = "dim-daemon"
	}
	deadline := time.Now().Add(HandshakeTimeout)
	_ = conn.SetDeadline(deadline)

	hello := Hello{
		Op:             "hello",
		Version:        ProtocolVersion,
		Endpoint:       l.peer,
		Implementation: map[string]string{"name": "mygo-uileaf", "version": "0"},
	}
	helloBytes, err := encodeJSON(hello)
	if err != nil {
		return fmt.Errorf("rutis: encode greeting: %w", err)
	}
	if err := writeFrame(conn, helloBytes); err != nil {
		return fmt.Errorf("rutis: send greeting: %w", err)
	}
	raw, err := readFrame(reader)
	if err != nil {
		return fmt.Errorf("rutis: read greeting: %w", err)
	}
	var far Hello
	if err := json.Unmarshal(raw, &far); err != nil {
		return fmt.Errorf("%w: malformed greeting: %v", ErrIncompatible, err)
	}
	if far.Op != "hello" {
		return fmt.Errorf("%w: expected hello, got %q", ErrIncompatible, far.Op)
	}
	if far.Version != ProtocolVersion {
		return fmt.Errorf("%w: far end speaks protocol %d, this side %d", ErrIncompatible, far.Version, ProtocolVersion)
	}
	if far.Endpoint == "" || far.Endpoint == l.peer {
		return fmt.Errorf("%w: far end named endpoint %q", ErrIncompatible, far.Endpoint)
	}
	if far.Endpoint != expectedPeer {
		return fmt.Errorf("%w: far end greeted as %s, expected %s", ErrExpectedPeer, far.Endpoint, expectedPeer)
	}
	_ = conn.SetDeadline(time.Time{})
	l.mu.Lock()
	l.greeting = Greeting{
		Endpoint:       far.Endpoint,
		Implementation: far.Implementation,
		Capabilities:   far.Capabilities,
	}
	l.sentBytes = len(helloBytes)
	l.recvBytes = len(raw)
	l.mu.Unlock()
	return nil
}

func (l *Leaf) readLoop(ctx context.Context, conn net.Conn, reader *bufio.Reader) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-l.done:
			return ErrClosed
		default:
		}
		raw, err := readFrame(reader)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return ErrClosed
			}
			return err
		}
		if err := l.handleFrame(conn, raw); err != nil {
			return err
		}
	}
}

type incoming struct {
	Op     string          `json:"op"`
	ID     json.RawMessage `json:"id"`
	Target *string         `json:"target"`
	Method string          `json:"method"`
}

// handleFrame answers a call this leaf does not serve with a BindingError.
// Notifications and offers need no reply and are ignored: this leaf has no
// catalog and no services to announce.
func (l *Leaf) handleFrame(conn net.Conn, raw []byte) error {
	var frame incoming
	if err := json.Unmarshal(raw, &frame); err != nil {
		return fmt.Errorf("%w: malformed frame: %v", ErrIncompatible, err)
	}
	switch frame.Op {
	case "invoke", "call", "get":
		var id string
		if err := json.Unmarshal(frame.ID, &id); err != nil || id == "" {
			return fmt.Errorf("%w: call without id", ErrIncompatible)
		}
		return writeJSON(conn, map[string]any{
			"op": "throw",
			"id": id,
			"error": map[string]any{
				"name":    "BindingError",
				"message": "the UI leaf serves no methods",
			},
		})
	default:
		return nil
	}
}

func (l *Leaf) finish(err error) {
	l.mu.Lock()
	if l.closeErr == nil {
		l.closeErr = err
	}
	l.mu.Unlock()
	l.closeOnce.Do(func() {
		if l.listener != nil {
			_ = l.listener.Close()
		}
		_ = os.Remove(socketPath(l.address))
		close(l.done)
	})
}

func socketPath(address string) string {
	const prefix = "unix:"
	if len(address) > len(prefix) && address[:len(prefix)] == prefix {
		return address[len(prefix):]
	}
	return ""
}

// writeJSON writes one JSON message plus newline.
func writeJSON(w io.Writer, value any) error {
	data, err := encodeJSON(value)
	if err != nil {
		return err
	}
	return writeFrame(w, data)
}

// encodeJSON marshals one message and enforces the frame limit. Encoding
// JSON never leaves a raw newline inside a string.
func encodeJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxFrameBytes {
		return nil, &FrameError{Message: fmt.Sprintf("message of %d bytes exceeds the limit of %d", len(data), MaxFrameBytes)}
	}
	return data, nil
}

func writeFrame(w io.Writer, data []byte) error {
	data = append(data, '\n')
	_, err := w.Write(data)
	return err
}

// readFrame reads one newline-terminated UTF-8 message of at most
// MaxFrameBytes, without its newline.
func readFrame(reader *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		chunk, err := reader.ReadSlice('\n')
		line = append(line, chunk...)
		if len(line) > MaxFrameBytes+1 {
			return nil, &FrameError{Message: fmt.Sprintf("a message exceeds the limit of %d bytes", MaxFrameBytes)}
		}
		if err == nil {
			line = line[:len(line)-1]
			if !utf8.Valid(line) {
				return nil, &FrameError{Message: "a message is not UTF-8"}
			}
			return line, nil
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) {
			if len(line) > 0 {
				return nil, &FrameError{Message: "stream ended inside a message"}
			}
			return nil, io.EOF
		}
		return nil, err
	}
}

func randomHex(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("rutis: random id: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}
