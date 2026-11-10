// Package session 维护会话流：每个会话一条连接、一份订阅。
//
// 规则：订阅先于 attach 与历史读取；持有者按会话计数，全部释放才 detach；
// 断线后按序号重读，已交付的序号之前的重放事实被丢弃。
package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
)

// 默认值。
const (
	defaultHistoryLimit     = 200
	defaultReconnectDelay   = 500 * time.Millisecond
	defaultReconnectAttempt = 20
)

// Opener 打开一条已完成握手的独立连接。daemon.Supervisor 实现它。
type Opener interface {
	OpenStream(ctx context.Context) (*daemon.Connection, error)
}

// Config 是会话流管理器配置。
type Config struct {
	Opener Opener
	// HistoryLimit 是初始历史读取的单页上限。
	HistoryLimit int
	// ReconnectAttempts 是断线后的重连尝试次数；用尽即终态。
	ReconnectAttempts int
	// ReconnectDelay 是两次重连之间的等待。
	ReconnectDelay time.Duration
}

func (c Config) withDefaults() Config {
	if c.HistoryLimit <= 0 {
		c.HistoryLimit = defaultHistoryLimit
	}
	if c.ReconnectAttempts <= 0 {
		c.ReconnectAttempts = defaultReconnectAttempt
	}
	if c.ReconnectDelay <= 0 {
		c.ReconnectDelay = defaultReconnectDelay
	}
	return c
}

// Manager 持有每会话唯一的订阅与持有者计数。
type Manager struct {
	cfg Config

	mu      sync.Mutex
	entries map[string]*entry
	closed  bool
}

type entry struct {
	sub     *Subscription
	holders int
}

// NewManager 创建管理器。
func NewManager(cfg Config) (*Manager, error) {
	if cfg.Opener == nil {
		return nil, errors.New("会话流管理器缺少 opener")
	}
	return &Manager{cfg: cfg.withDefaults(), entries: make(map[string]*entry)}, nil
}

// Acquire 增加一个持有者，返回初始快照与释放句柄。首个持有者建立流并订阅；
// 之后同一会话的持有者共享同一份订阅，不再新增连接。
//
// listener 只由首个持有者提供；后续持有者共享同一份事实分发，多窗口由上层
// 自行分发。所有持有者都释放后才 detach 并断开连接。
func (m *Manager) Acquire(
	ctx context.Context,
	sessionID string,
	listener Listener,
) (*Snapshot, *Lease, error) {
	if sessionID == "" {
		return nil, nil, failure.New(failure.KindNonRetryable, "session.acquire", errors.New("会话标识为空"))
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, nil, failure.New(failure.KindNonRetryable, "session.acquire", errors.New("会话流管理器已关闭"))
	}
	if existing, ok := m.entries[sessionID]; ok {
		// 已关闭或进入终态的订阅不能再给新持有者；移除后重建。
		if state := existing.sub.State(); state == StateClosed || state == StateFailed {
			delete(m.entries, sessionID)
		} else {
			existing.holders++
			sub := existing.sub
			m.mu.Unlock()
			snapshot, err := sub.waitReady(ctx)
			if err != nil {
				m.release(sub)
				return nil, nil, err
			}
			return snapshot, &Lease{manager: m, sub: sub}, nil
		}
	}

	sub := newSubscription(m, m.cfg, sessionID, listener)
	m.entries[sessionID] = &entry{sub: sub, holders: 1}
	m.mu.Unlock()

	snapshot, err := sub.start(ctx)
	if err != nil {
		m.forget(sub)
		sub.close()
		return nil, nil, err
	}
	return snapshot, &Lease{manager: m, sub: sub}, nil
}

// Close 关闭全部订阅；进程退出时调用。
func (m *Manager) Close() {
	m.mu.Lock()
	m.closed = true
	subs := make([]*Subscription, 0, len(m.entries))
	for _, entry := range m.entries {
		subs = append(subs, entry.sub)
	}
	m.entries = make(map[string]*entry)
	m.mu.Unlock()
	for _, sub := range subs {
		sub.close()
	}
}

// forget 在订阅进入终态时移除登记，之后的新持有者会重新建立流。
func (m *Manager) forget(sub *Subscription) {
	m.mu.Lock()
	if entry, ok := m.entries[sub.sessionID]; ok && entry.sub == sub {
		delete(m.entries, sub.sessionID)
	}
	m.mu.Unlock()
}

func (m *Manager) release(sub *Subscription) {
	m.mu.Lock()
	entry, ok := m.entries[sub.sessionID]
	if !ok || entry.sub != sub {
		m.mu.Unlock()
		return
	}
	entry.holders--
	last := entry.holders <= 0
	if last {
		delete(m.entries, sub.sessionID)
	}
	m.mu.Unlock()
	if last {
		sub.close()
	}
}

// Lease 是一个持有者的释放句柄。Release 可重复调用，只有第一次生效。
type Lease struct {
	manager *Manager
	sub     *Subscription
	once    sync.Once
}

// Release 释放一个持有者；最后一个持有者释放时才 detach 并断开。
func (l *Lease) Release() {
	if l == nil {
		return
	}
	l.once.Do(func() { l.manager.release(l.sub) })
}

// Subscription 返回共享订阅。
func (l *Lease) Subscription() *Subscription {
	if l == nil {
		return nil
	}
	return l.sub
}
