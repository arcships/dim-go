package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

var (
	// ErrConnectionClosed 表示连接已关闭。
	ErrConnectionClosed = errors.New("app server connection closed")
	// ErrRequestTimeout 表示请求在超时时间内没有回执。
	ErrRequestTimeout = errors.New("app server request timed out")
)

// NoTimeout 关闭单请求超时，用于长驻请求。
const NoTimeout = time.Duration(-1)

// DefaultRequestTimeout 是单请求的默认超时。
const DefaultRequestTimeout = 30 * time.Second

// TimeoutError 是一次请求超时。
type TimeoutError struct {
	Method  string
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("request %s timed out after %s", e.Method, e.Timeout)
}

func (e *TimeoutError) Is(target error) bool { return target == ErrRequestTimeout }

// ClosedError 说明连接在请求结束前关闭。
type ClosedError struct {
	Cause error
}

func (e *ClosedError) Error() string {
	if e.Cause == nil {
		return ErrConnectionClosed.Error()
	}
	return fmt.Sprintf("%s: %v", ErrConnectionClosed, e.Cause)
}

func (e *ClosedError) Unwrap() error { return e.Cause }

func (e *ClosedError) Is(target error) bool { return target == ErrConnectionClosed }

// Options 是客户端配置。
type Options struct {
	// MaxFrameBytes 是单帧上限；0 用默认值。
	MaxFrameBytes int
	// RequestTimeout 是单请求默认超时；0 用默认值。
	RequestTimeout time.Duration
	// OnError 接收坏帧、解析失败与监听器 panic。它只用于诊断，不能改变状态。
	OnError func(error)
}

type callResult struct {
	result json.RawMessage
	err    error
}

type pendingCall struct {
	method string
	ch     chan callResult
}

// Client 是一条已完成传输连接的 JSON-RPC 2.0 over NDJSON 客户端。
// 它负责请求关联、超时、通知分发与关闭；不解释任何业务方法。
type Client struct {
	conn           io.ReadWriteCloser
	decoder        *Decoder
	requestTimeout time.Duration
	onError        func(error)

	writeMu sync.Mutex

	mu             sync.Mutex
	pending        map[ID]*pendingCall
	nextID         uint64
	nextListener   int
	notifications  map[int]func(Message)
	closeListeners map[int]func(error)
	closedErr      error

	done      chan struct{}
	closeOnce sync.Once
}

// NewClient 接管 conn 并启动读循环。调用方之后只能用 Close 关闭底层连接。
func NewClient(conn io.ReadWriteCloser, opts Options) *Client {
	if opts.MaxFrameBytes <= 0 {
		opts.MaxFrameBytes = DefaultMaxFrameBytes
	}
	if opts.RequestTimeout <= 0 {
		opts.RequestTimeout = DefaultRequestTimeout
	}
	c := &Client{
		conn:           conn,
		decoder:        NewDecoder(opts.MaxFrameBytes),
		requestTimeout: opts.RequestTimeout,
		onError:        opts.OnError,
		pending:        make(map[ID]*pendingCall),
		notifications:  make(map[int]func(Message)),
		closeListeners: make(map[int]func(error)),
		done:           make(chan struct{}),
	}
	go c.readLoop()
	return c
}

// Call 发送请求并等待回执。超时返回 *TimeoutError。
func (c *Client) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	return c.call(ctx, method, params, c.requestTimeout)
}

// CallWithTimeout 用指定超时发送请求；timeout 为 NoTimeout 时不设超时。
func (c *Client) CallWithTimeout(
	ctx context.Context,
	method string,
	params any,
	timeout time.Duration,
) (json.RawMessage, error) {
	if timeout == 0 {
		timeout = c.requestTimeout
	}
	return c.call(ctx, method, params, timeout)
}

// CallInto 发送请求并把回执解码到 out。
func (c *Client) CallInto(ctx context.Context, method string, params, out any) error {
	raw, err := c.Call(ctx, method, params)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode %s reply: %w", method, err)
	}
	return nil
}

// OnNotification 注册通知监听；返回取消函数。未知方法也会送达。
func (c *Client) OnNotification(fn func(Message)) (cancel func()) {
	if fn == nil {
		return func() {}
	}
	c.mu.Lock()
	id := c.nextListener
	c.nextListener++
	c.notifications[id] = fn
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		delete(c.notifications, id)
		c.mu.Unlock()
	}
}

// OnClose 注册关闭回调；连接已关闭时立即调用。
func (c *Client) OnClose(fn func(error)) (cancel func()) {
	if fn == nil {
		return func() {}
	}
	c.mu.Lock()
	if c.closedErr != nil {
		err := c.closedErr
		c.mu.Unlock()
		fn(err)
		return func() {}
	}
	id := c.nextListener
	c.nextListener++
	c.closeListeners[id] = fn
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		delete(c.closeListeners, id)
		c.mu.Unlock()
	}
}

// Done 在连接关闭后关闭。
func (c *Client) Done() <-chan struct{} { return c.done }

// Err 返回关闭原因；未关闭时为 nil。
func (c *Client) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closedErr
}

// Close 关闭连接并以 reason 结束所有在途请求。可重复调用，第一次生效。
func (c *Client) Close(reason error) {
	if reason == nil {
		reason = &ClosedError{}
	}
	var normalized *ClosedError
	if !errors.As(reason, &normalized) {
		reason = &ClosedError{Cause: reason}
	}
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closedErr = reason
		pending := c.pending
		c.pending = make(map[ID]*pendingCall)
		listeners := make([]func(error), 0, len(c.closeListeners))
		for _, fn := range c.closeListeners {
			listeners = append(listeners, fn)
		}
		c.closeListeners = make(map[int]func(error))
		c.mu.Unlock()

		close(c.done)
		_ = c.conn.Close()
		for _, call := range pending {
			select {
			case call.ch <- callResult{err: reason}:
			default:
			}
		}
		for _, fn := range listeners {
			c.protect(func() { fn(reason) })
		}
	})
}

func (c *Client) call(
	ctx context.Context,
	method string,
	params any,
	timeout time.Duration,
) (json.RawMessage, error) {
	if err := c.Err(); err != nil {
		return nil, err
	}
	id := c.nextRequestID()
	call := &pendingCall{method: method, ch: make(chan callResult, 1)}
	if err := c.register(id, call); err != nil {
		return nil, err
	}
	frame, err := encodeRequest(id, method, params)
	if err != nil {
		c.unregister(id)
		return nil, err
	}
	if err := c.writeFrame(frame); err != nil {
		c.unregister(id)
		return nil, err
	}

	var timer *time.Timer
	var timeoutC <-chan time.Time
	if timeout != NoTimeout {
		timer = time.NewTimer(timeout)
		timeoutC = timer.C
		defer timer.Stop()
	}
	select {
	case res := <-call.ch:
		return res.result, res.err
	case <-ctx.Done():
		c.unregister(id)
		return nil, ctx.Err()
	case <-timeoutC:
		c.unregister(id)
		return nil, &TimeoutError{Method: method, Timeout: timeout}
	case <-c.done:
		c.unregister(id)
		if err := c.Err(); err != nil {
			return nil, err
		}
		return nil, &ClosedError{}
	}
}

func (c *Client) nextRequestID() ID {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextID++
	return ID(fmt.Sprintf("%d", c.nextID))
}

func (c *Client) register(id ID, call *pendingCall) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closedErr != nil {
		return c.closedErr
	}
	c.pending[id] = call
	return nil
}

func (c *Client) unregister(id ID) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func (c *Client) writeFrame(frame []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := c.Err(); err != nil {
		return err
	}
	if _, err := c.conn.Write(frame); err != nil {
		closed := &ClosedError{Cause: err}
		c.Close(closed)
		return closed
	}
	return nil
}

func (c *Client) readLoop() {
	buf := make([]byte, 64*1024)
	for {
		n, err := c.conn.Read(buf)
		if n > 0 {
			frames, frameErr := c.decoder.Push(buf[:n])
			for _, frame := range frames {
				c.dispatch(frame)
			}
			if frameErr != nil {
				c.report(frameErr)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.Close(&ClosedError{Cause: io.EOF})
			} else {
				c.Close(&ClosedError{Cause: err})
			}
			return
		}
	}
}

func (c *Client) dispatch(frame []byte) {
	message, err := ParseMessage(frame)
	if err != nil {
		c.report(err)
		return
	}
	switch message.Kind {
	case KindNotification:
		c.deliverNotification(message)
	case KindResult, KindError:
		if string(message.ID) == "null" {
			if message.Error != nil {
				c.report(message.Error)
			} else {
				c.report(errors.New("reply has a null id"))
			}
			return
		}
		c.mu.Lock()
		call, ok := c.pending[message.ID]
		if ok {
			delete(c.pending, message.ID)
		}
		c.mu.Unlock()
		if !ok {
			// 已超时或已关闭的请求：回执迟到，没有消费者。与产品基线一致，静默忽略。
			return
		}
		if message.Kind == KindError {
			call.ch <- callResult{err: message.Error}
			return
		}
		call.ch <- callResult{result: message.Result}
	}
}

func (c *Client) deliverNotification(message Message) {
	c.mu.Lock()
	listeners := make([]func(Message), 0, len(c.notifications))
	for _, fn := range c.notifications {
		listeners = append(listeners, fn)
	}
	c.mu.Unlock()
	for _, fn := range listeners {
		listener := fn
		c.protect(func() { listener(message) })
	}
}

func (c *Client) report(err error) {
	if err == nil || c.onError == nil {
		return
	}
	c.protect(func() { c.onError(err) })
}

// protect 隔离监听器 panic：一条坏监听器不能让读循环退出。
func (c *Client) protect(fn func()) {
	defer func() { _ = recover() }()
	fn()
}

func encodeRequest(id ID, method string, params any) ([]byte, error) {
	request := Request{JSONRPC: JSONRPCVersion, ID: id, Method: method}
	if params != nil {
		raw, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("encode params for %s: %w", method, err)
		}
		request.Params = raw
	}
	frame, err := EncodeFrame(request)
	if err != nil {
		return nil, err
	}
	return frame, nil
}
