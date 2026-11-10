package session

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// Listener 接收会话流的事实、重同步、重读结果与终态。
//
// 回调按到达顺序串行调用，回调里可以再调用 Manager（例如释放别的持有者）。
type Listener struct {
	// OnEvent 接收实时通知。Method 与 Params 原样保留，未知通知也会送达。
	OnEvent func(Event)
	// OnHistory 接收断线重连后按序号重读的事实。
	OnHistory func(HistoryReplay)
	// OnResync 在断线时调用一次，先于重读。
	OnResync func(Resync)
	// OnFatal 在无法恢复时调用一次；此后订阅进入终态。
	OnFatal func(error)
}

// Event 是一条通知。Method 与 Params 原样保留；已识别的通知另外给出
// 类型化字段，解码失败时只设置 DecodeErr，不丢弃事实。
type Event struct {
	Method string
	Params json.RawMessage

	ApprovalRequest   *protocol.ApprovalRequestNotice
	ApprovalCancelled *protocol.ApprovalCancelledNotice
	ApprovalResolved  *protocol.ApprovalResolvedNotice
	SessionNotice     *protocol.SessionNotice
	// Title 只在 session.notice 是 SessionStore 的标题变更时非空。
	Title *protocol.TitleChanged
	// Wakeup 只在 session.notice 是 Scheduler 的唤醒变化时非空。
	Wakeup *protocol.WakeupNoticeData
	// PermissionChanged 只在 session.event 是 permission/changed 时非空。
	PermissionChanged *protocol.PermissionChanged
	// Progress 是管理方法回执前的进度通知（按 id 与请求关联）。
	Progress *protocol.ManagementProgressNotice
	// ModalityEvent / ModalityEnded 是实时模态通知。
	ModalityEvent *protocol.ModalityRealtimeEventNotice
	ModalityEnded *protocol.ModalityRealtimeEndedNotice
	// DecodeErr 表示已识别方法的字段不合法；上层应继续按 Params 处理。
	DecodeErr error
}

// Resync 表示会话流已断开，即将按 FromSequence 之后的事实重读。
type Resync struct {
	FromSequence int64
	Err          error
}

// HistoryReplay 是一次按序号重读的结果。
type HistoryReplay struct {
	FromSequence int64
	History      protocol.SessionHistoryResult
}

// Snapshot 是会话流打开时读到的初始事实。History 是初始窗口；上层按
// NextBefore / NextAfter 继续翻页。
type Snapshot struct {
	SessionID string
	Attach    protocol.SessionAttachResult
	History   protocol.SessionHistoryResult
	Head      int64
	Features  []string
}

// State 是会话流状态，用于诊断记录。
type State string

const (
	StateOpening      State = "opening"
	StateLive         State = "live"
	StateReconnecting State = "reconnecting"
	StateClosed       State = "closed"
	StateFailed       State = "failed"
)

// Subscription 是一个会话的唯一一份订阅。多个持有者共享它。
type Subscription struct {
	manager   *Manager
	cfg       Config
	sessionID string
	listener  Listener

	ready     chan struct{}
	readyOnce sync.Once

	fatalOnce sync.Once
	closeOnce sync.Once
	closed    chan struct{}

	mu               sync.Mutex
	state            State
	snapshot         *Snapshot
	openErr          error
	conn             *daemon.Connection
	cancelNotify     func()
	pending          []Event
	paused           bool
	deliveredThrough int64
	reconnecting     bool

	// wake 唤醒投递协程；容量 1，重复信号会被合并。
	wake chan struct{}
}

func newSubscription(manager *Manager, cfg Config, sessionID string, listener Listener) *Subscription {
	return &Subscription{
		manager:   manager,
		cfg:       cfg,
		sessionID: sessionID,
		listener:  listener,
		ready:     make(chan struct{}),
		closed:    make(chan struct{}),
		state:     StateOpening,
		wake:      make(chan struct{}, 1),
	}
}

// SessionID 返回会话标识。
func (s *Subscription) SessionID() string { return s.sessionID }

// State 返回当前状态。
func (s *Subscription) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// start 建立首条流并读取初始历史。订阅在 attach 之前登记，attach 的重放不会丢。
func (s *Subscription) start(ctx context.Context) (*Snapshot, error) {
	conn, attach, history, err := s.open(ctx, 0)
	if err != nil {
		s.finishOpen(err, nil)
		return nil, err
	}
	head := attachHead(attach, history)
	s.mu.Lock()
	s.conn = conn
	s.deliveredThrough = throughhead(history, head)
	s.state = StateLive
	s.mu.Unlock()

	snapshot := &Snapshot{
		SessionID: s.sessionID,
		Attach:    attach,
		History:   history,
		Head:      head,
		Features:  append([]string(nil), conn.Hello.Features...),
	}
	s.finishOpen(nil, snapshot)
	go s.pump()
	go s.watch(conn)
	return snapshot, nil
}

// History 用当前会话流连接读一页历史；会话标识由订阅填好。
func (s *Subscription) History(
	ctx context.Context,
	params protocol.SessionHistoryParams,
) (protocol.SessionHistoryResult, error) {
	var result protocol.SessionHistoryResult
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil || conn.Client.Err() != nil {
		return result, failure.New(failure.KindRetryable, "session.history",
			errors.New("会话流未连接"))
	}
	params.SessionID = s.sessionID
	if err := conn.Client.CallInto(ctx, "session.history", params, &result); err != nil {
		return result, failure.Wrap(failure.KindOf(err), "session.history", err)
	}
	return result, nil
}

// open 打开一条新流：先登记通知，再 attach，最后读历史。
func (s *Subscription) open(
	ctx context.Context,
	after int64,
) (*daemon.Connection, protocol.SessionAttachResult, protocol.SessionHistoryResult, error) {
	var attach protocol.SessionAttachResult
	var history protocol.SessionHistoryResult

	conn, err := s.cfg.Opener.OpenStream(ctx)
	if err != nil {
		return nil, attach, history, err
	}
	cancel := conn.Client.OnNotification(s.onNotification)
	fail := func(openErr error) (*daemon.Connection, protocol.SessionAttachResult, protocol.SessionHistoryResult, error) {
		cancel()
		conn.Close(openErr)
		return nil, attach, history, openErr
	}

	params := protocol.SessionAttachParams{SessionID: s.sessionID}
	if err := conn.Client.CallInto(ctx, "session.attach", params, &attach); err != nil {
		return fail(failure.Wrap(failure.KindOf(err), "session.attach", err))
	}
	head, err := s.resolveHead(ctx, conn, attach)
	if err != nil {
		return fail(err)
	}
	if after > 0 {
		history, err = s.readAfter(ctx, conn, after, head)
	} else {
		history, err = s.readInitial(ctx, conn, head)
	}
	if err != nil {
		return fail(err)
	}
	s.mu.Lock()
	s.cancelNotify = cancel
	s.mu.Unlock()
	return conn, attach, history, nil
}

func (s *Subscription) resolveHead(
	ctx context.Context,
	conn *daemon.Connection,
	attach protocol.SessionAttachResult,
) (int64, error) {
	if attach.Sequence != nil {
		return *attach.Sequence, nil
	}
	// 旧服务端不回 sequence：用一页最小历史取头部序号。
	head := int64(0)
	after := int64(0)
	limit := 1
	params := protocol.SessionHistoryParams{
		SessionID: s.sessionID,
		BranchID:  "main",
		After:     &after,
		Limit:     &limit,
	}
	var page protocol.SessionHistoryResult
	if err := conn.Client.CallInto(ctx, "session.history", params, &page); err != nil {
		return 0, failure.Wrap(failure.KindOf(err), "session.history", err)
	}
	head = page.Through
	return head, nil
}

func (s *Subscription) readInitial(
	ctx context.Context,
	conn *daemon.Connection,
	head int64,
) (protocol.SessionHistoryResult, error) {
	if head <= 0 {
		return protocol.SessionHistoryResult{Through: head, Done: true}, nil
	}
	limit := s.cfg.HistoryLimit
	if conn.HasFeature("session.history.latestTurn") {
		latest := true
		params := protocol.SessionHistoryParams{
			SessionID:  s.sessionID,
			BranchID:   "main",
			LatestTurn: &latest,
			Limit:      &limit,
			Through:    &head,
		}
		return s.callHistory(ctx, conn, params)
	}
	// 旧服务端没有 latestTurn：从起点读一页。窗口与锚点由会话投影继续翻页。
	after := int64(0)
	params := protocol.SessionHistoryParams{
		SessionID: s.sessionID,
		BranchID:  "main",
		After:     &after,
		Limit:     &limit,
		Through:   &head,
	}
	return s.callHistory(ctx, conn, params)
}

// readAfter 按序号重读，直到覆盖到 attach 头部或服务端说读完。
func (s *Subscription) readAfter(
	ctx context.Context,
	conn *daemon.Connection,
	after int64,
	head int64,
) (protocol.SessionHistoryResult, error) {
	limit := s.cfg.HistoryLimit
	events := make([]protocol.RawEvent, 0, limit)
	cursor := after
	done := true
	for pages := 0; pages < 100; pages++ {
		params := protocol.SessionHistoryParams{
			SessionID: s.sessionID,
			BranchID:  "main",
			After:     &cursor,
			Limit:     &limit,
		}
		if head > 0 {
			params.Through = &head
		}
		page, err := s.callHistory(ctx, conn, params)
		if err != nil {
			return protocol.SessionHistoryResult{}, err
		}
		for _, event := range page.Events {
			if event.Sequence > after {
				events = append(events, event)
			}
		}
		done = page.Done
		if page.Done || page.NextAfter <= cursor {
			break
		}
		cursor = page.NextAfter
		if head > 0 && cursor >= head {
			done = true
			break
		}
	}
	through := head
	if len(events) > 0 && events[len(events)-1].Sequence > through {
		through = events[len(events)-1].Sequence
	}
	return protocol.SessionHistoryResult{
		Events:    events,
		Through:   through,
		NextAfter: cursor,
		Done:      done,
	}, nil
}

func (s *Subscription) callHistory(
	ctx context.Context,
	conn *daemon.Connection,
	params protocol.SessionHistoryParams,
) (protocol.SessionHistoryResult, error) {
	var page protocol.SessionHistoryResult
	if err := conn.Client.CallInto(ctx, "session.history", params, &page); err != nil {
		return page, failure.Wrap(failure.KindOf(err), "session.history", err)
	}
	return page, nil
}

func (s *Subscription) onNotification(message protocol.Message) {
	s.enqueue(Event{Method: message.Method, Params: message.Params})
}

func (s *Subscription) enqueue(event Event) {
	s.mu.Lock()
	s.pending = append(s.pending, event)
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// pump 串行投递事实：暂停时事实留在缓冲里，恢复后按到达顺序补投。
func (s *Subscription) pump() {
	for {
		select {
		case <-s.closed:
			return
		case <-s.wake:
		}
		for {
			s.mu.Lock()
			if s.paused || len(s.pending) == 0 {
				s.mu.Unlock()
				break
			}
			event := s.pending[0]
			s.pending = s.pending[1:]
			s.mu.Unlock()
			s.deliver(event)
		}
	}
}

func (s *Subscription) deliver(event Event) {
	if event.Method == "session.event" {
		var params protocol.SessionEventParams
		if err := json.Unmarshal(event.Params, &params); err == nil && params.Event.Sequence > 0 {
			s.mu.Lock()
			delivered := s.deliveredThrough
			if params.Event.Sequence > delivered {
				s.deliveredThrough = params.Event.Sequence
			}
			s.mu.Unlock()
			if params.Event.Sequence <= delivered {
				// attach 重放或重读已覆盖的事实：丢弃，不重复交给上层。
				return
			}
		}
	}
	decodeEvent(&event)
	s.notify(func() {
		if s.listener.OnEvent != nil {
			s.listener.OnEvent(event)
		}
	})
}

// watch 等连接关闭；只有当前连接才触发重连。
func (s *Subscription) watch(conn *daemon.Connection) {
	select {
	case <-s.closed:
		return
	case <-conn.Client.Done():
	}
	s.mu.Lock()
	current := s.conn == conn
	s.mu.Unlock()
	if !current {
		return
	}
	s.onLost(conn.Client.Err())
}

// onLost 发布重同步并按序号重读；版本不兼容或不可重试错误进入终态。
func (s *Subscription) onLost(cause error) {
	if cause == nil {
		cause = errors.New("会话流断开")
	}
	s.mu.Lock()
	if s.reconnecting || s.state == StateClosed || s.state == StateFailed {
		s.mu.Unlock()
		return
	}
	s.reconnecting = true
	s.paused = true
	s.state = StateReconnecting
	from := s.deliveredThrough
	s.mu.Unlock()

	s.notify(func() {
		if s.listener.OnResync != nil {
			s.listener.OnResync(Resync{FromSequence: from, Err: cause})
		}
	})

	var lastErr error
	for attempt := 0; attempt < s.cfg.ReconnectAttempts; attempt++ {
		select {
		case <-s.closed:
			return
		case <-time.After(s.cfg.ReconnectDelay):
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		conn, attach, history, err := s.open(ctx, from)
		cancel()
		if err != nil {
			lastErr = err
			switch failure.KindOf(err) {
			case failure.KindVersionIncompatible, failure.KindNonRetryable:
				s.fatal(err)
				return
			}
			continue
		}
		head := attachHead(attach, history)
		s.mu.Lock()
		previous := s.conn
		s.conn = conn
		s.mu.Unlock()
		if previous != nil {
			previous.Close(&protocol.ClosedError{Cause: errors.New("stream replaced")})
		}
		go s.watch(conn)
		if s.listener.OnHistory != nil && len(history.Events) > 0 {
			replay := HistoryReplay{FromSequence: from, History: history}
			s.notify(func() { s.listener.OnHistory(replay) })
		}
		s.mu.Lock()
		if through := throughhead(history, head); through > s.deliveredThrough {
			s.deliveredThrough = through
		}
		s.paused = false
		s.reconnecting = false
		s.state = StateLive
		s.mu.Unlock()
		select {
		case s.wake <- struct{}{}:
		default:
		}
		return
	}
	if lastErr == nil {
		lastErr = cause
	}
	s.fatal(lastErr)
}

// fatal 进入终态：从管理器移除，通知一次 OnFatal，然后关闭。
func (s *Subscription) fatal(err error) {
	if err == nil {
		err = errors.New("会话流不可恢复")
	}
	s.fatalOnce.Do(func() {
		s.manager.forget(s)
		s.mu.Lock()
		s.state = StateFailed
		s.mu.Unlock()
		s.notify(func() {
			if s.listener.OnFatal != nil {
				s.listener.OnFatal(err)
			}
		})
		s.close()
	})
}

// close 释放订阅：detach 后断开连接。可重复调用。
func (s *Subscription) close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.state = StateClosed
		conn := s.conn
		cancel := s.cancelNotify
		s.conn = nil
		s.cancelNotify = nil
		s.pending = nil
		s.mu.Unlock()

		close(s.closed)
		if cancel != nil {
			cancel()
		}
		if conn != nil {
			ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
			_ = conn.Client.CallInto(ctx, "session.detach", protocol.SessionDetachParams{SessionID: s.sessionID}, nil)
			stop()
			conn.Close(&protocol.ClosedError{Cause: errors.New("subscription released")})
		}
	})
}

func (s *Subscription) finishOpen(err error, snapshot *Snapshot) {
	s.readyOnce.Do(func() {
		s.mu.Lock()
		s.openErr = err
		s.snapshot = snapshot
		s.mu.Unlock()
		close(s.ready)
	})
}

func (s *Subscription) waitReady(ctx context.Context) (*Snapshot, error) {
	select {
	case <-s.ready:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.closed:
		select {
		case <-s.ready:
		default:
			return nil, failure.New(failure.KindNonRetryable, "session.open",
				errors.New("会话流已关闭"))
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.openErr != nil {
		return nil, s.openErr
	}
	return s.snapshot, nil
}

// notify 隔离监听器 panic，避免一条坏回调打断投递。
func (s *Subscription) notify(fn func()) {
	defer func() { _ = recover() }()
	fn()
}

func attachHead(attach protocol.SessionAttachResult, history protocol.SessionHistoryResult) int64 {
	if attach.Sequence != nil {
		return *attach.Sequence
	}
	return history.Through
}

// throughhead 是已由快照或重读覆盖的最高序号：它之前的实时事实不再交付。
func throughhead(history protocol.SessionHistoryResult, head int64) int64 {
	through := history.Through
	if head > through {
		through = head
	}
	if len(history.Events) > 0 {
		if last := history.Events[len(history.Events)-1].Sequence; last > through {
			through = last
		}
	}
	return through
}
