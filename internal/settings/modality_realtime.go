package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// ModalityRealtimeOpenRequest 是一次实时传输打开。Operation 必填；Params 为空
// 表示调用方没有提供参数，本包不补默认对象。
type ModalityRealtimeOpenRequest struct {
	SessionID string
	Operation string
	Model     *ModalityModelRef
	Params    json.RawMessage
}

// ModalityRealtimeConnection 是本进程确认打开的一个实时连接。
type ModalityRealtimeConnection struct {
	SessionID    string
	ConnectionID string
}

// ModalityRealtimeFrame 是一帧实时数据；Kind 取 text / audio / control。
type ModalityRealtimeFrame struct {
	Kind string
	Data json.RawMessage
}

// ModalityRealtimeEvent 是服务端推来的一条实时事件；Event 原样保留。
type ModalityRealtimeEvent struct {
	SessionID    string
	ConnectionID string
	Event        json.RawMessage
}

// ModalityRealtimeEnded 表示一条实时连接结束。
type ModalityRealtimeEnded struct {
	SessionID    string
	ConnectionID string
	Reason       string
}

// ModalityRealtimeListener 是实时事件订阅。两个回调都可以为空，但不能都为空。
// 回调同步执行；不要在里面阻塞，也不要回调本包的管理写入（会等待同一条串行通道）。
type ModalityRealtimeListener struct {
	OnEvent func(ModalityRealtimeEvent)
	OnEnded func(ModalityRealtimeEnded)
}

type modalityRealtimeOpenParams struct {
	Op        string            `json:"op"`
	SessionID string            `json:"sessionId"`
	Operation string            `json:"operation"`
	Model     *modalityModelRef `json:"model,omitempty"`
	Params    json.RawMessage   `json:"params,omitempty"`
}

type modalityRealtimeCloseParams struct {
	Op           string `json:"op"`
	SessionID    string `json:"sessionId"`
	ConnectionID string `json:"connectionId"`
}

// realtimeState 是一条已确认打开的连接及其本地订阅。
type realtimeState struct {
	sessionID    string
	connectionID string
	listeners    map[int]ModalityRealtimeListener
	nextLsn      int
}

// OpenRealtime 打开一条实时连接。打开经写登记登记一次 KindModalityWrite
// （op=realtime.open）；成功后用回执里的连接标识登记本地连接状态，不乐观更新。
func (m *Manager) OpenRealtime(ctx context.Context, request ModalityRealtimeOpenRequest) (ModalityRealtimeConnection, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID, err := normalizeSessionID(request.SessionID)
	if err != nil {
		return ModalityRealtimeConnection{}, err
	}
	operation, ok := trimNonEmpty(request.Operation)
	if !ok {
		return ModalityRealtimeConnection{}, fmt.Errorf("%w: 实时传输缺少 operation", ErrInvalidModalityRequest)
	}
	params, err := normalizeOptionalObject(request.Params)
	if err != nil {
		return ModalityRealtimeConnection{}, err
	}
	if err := validateModelRef(request.Model); err != nil {
		return ModalityRealtimeConnection{}, err
	}

	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	if err := m.requireModalityMethod(ctx, sessionID, ModalityMethodRealtimeOpen); err != nil {
		return ModalityRealtimeConnection{}, err
	}
	key := m.modalityWriteKey("realtime.open", sessionID, "")
	view, err := m.registerModalityWrite(ctx, key, sessionID, "", modalityRealtimeOpenParams{
		Op:        "realtime.open",
		SessionID: sessionID,
		Operation: operation,
		Model:     toWireModelRef(request.Model),
		Params:    params,
	})
	if err != nil {
		return ModalityRealtimeConnection{}, err
	}
	if view.Status != write.StatusSucceeded {
		return ModalityRealtimeConnection{}, fmt.Errorf("%w: %s", ErrRealtimeOpenFailed, failureReason(view))
	}
	connectionID, ok := trimNonEmpty(view.InputID)
	if !ok {
		return ModalityRealtimeConnection{}, fmt.Errorf("%w: 实时打开回执缺少连接标识", ErrMalformedWriteResult)
	}
	m.addRealtimeState(sessionID, connectionID)
	return ModalityRealtimeConnection{SessionID: sessionID, ConnectionID: connectionID}, nil
}

// SendRealtime 在已打开的连接上发送一帧。发送走会话流，不经过写登记，也不重试。
func (m *Manager) SendRealtime(ctx context.Context, sessionID, connectionID string, frame ModalityRealtimeFrame) error {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID, err := normalizeSessionID(sessionID)
	if err != nil {
		return err
	}
	connectionID, ok := trimNonEmpty(connectionID)
	if !ok {
		return fmt.Errorf("%w: 实时发送缺少连接标识", ErrInvalidModalityRequest)
	}
	kind, err := normalizeFrameKind(frame.Kind)
	if err != nil {
		return err
	}
	data, err := normalizeFrameData(kind, frame.Data)
	if err != nil {
		return err
	}
	if !m.hasRealtimeState(sessionID, connectionID) {
		return fmt.Errorf("%w: %s/%s", ErrRealtimeNotOpen, sessionID, connectionID)
	}
	if err := m.requireModalityMethod(ctx, sessionID, ModalityMethodRealtimeSend); err != nil {
		return err
	}
	if m.cfg.ModalitySession == nil {
		return missing(ModalityMethodRealtimeSend, "未提供会话模态调用面")
	}
	if err := m.cfg.ModalitySession.SendModalityRealtime(ctx, sessionID, connectionID, ModalityRealtimeFrame{
		Kind: kind,
		Data: data,
	}); err != nil {
		return readFailure(ModalityMethodRealtimeSend, ErrRealtimeSendFailed, err)
	}
	return nil
}

// CloseRealtime 关闭一条实时连接。关闭经写登记登记一次 KindModalityWrite
// （op=realtime.close）；成功后移除本地状态并通知订阅者。没有本地状态时不发请求。
func (m *Manager) CloseRealtime(ctx context.Context, sessionID, connectionID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID, err := normalizeSessionID(sessionID)
	if err != nil {
		return err
	}
	connectionID, ok := trimNonEmpty(connectionID)
	if !ok {
		return fmt.Errorf("%w: 实时关闭缺少连接标识", ErrInvalidModalityRequest)
	}

	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	if !m.hasRealtimeState(sessionID, connectionID) {
		return fmt.Errorf("%w: %s/%s", ErrRealtimeNotOpen, sessionID, connectionID)
	}
	if err := m.requireModalityMethod(ctx, sessionID, ModalityMethodRealtimeClose); err != nil {
		return err
	}
	key := m.modalityWriteKey("realtime.close", sessionID, "")
	view, err := m.registerModalityWrite(ctx, key, sessionID, connectionID, modalityRealtimeCloseParams{
		Op:           "realtime.close",
		SessionID:    sessionID,
		ConnectionID: connectionID,
	})
	if err != nil {
		return err
	}
	if view.Status != write.StatusSucceeded {
		return fmt.Errorf("%w: %s", ErrRealtimeCloseFailed, failureReason(view))
	}
	m.endRealtimeState(sessionID, connectionID, "closed")
	return nil
}

// WatchRealtime 订阅一条已打开连接的事件。连接不存在或已结束时返回
// ErrRealtimeNotOpen；取消函数幂等。
func (m *Manager) WatchRealtime(
	sessionID, connectionID string,
	listener ModalityRealtimeListener,
) (func(), error) {
	sessionID, err := normalizeSessionID(sessionID)
	if err != nil {
		return nil, err
	}
	connectionID, ok := trimNonEmpty(connectionID)
	if !ok {
		return nil, fmt.Errorf("%w: 实时订阅缺少连接标识", ErrInvalidModalityRequest)
	}
	if listener.OnEvent == nil && listener.OnEnded == nil {
		return nil, fmt.Errorf("%w: 实时订阅没有回调", ErrInvalidModalityRequest)
	}

	m.mu.Lock()
	state, found := m.realtime[realtimeCacheKey(sessionID, connectionID)]
	if !found {
		m.mu.Unlock()
		return nil, fmt.Errorf("%w: %s/%s", ErrRealtimeNotOpen, sessionID, connectionID)
	}
	lsn := state.nextLsn
	state.nextLsn++
	state.listeners[lsn] = listener
	m.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			m.mu.Lock()
			defer m.mu.Unlock()
			if current, ok := m.realtime[realtimeCacheKey(sessionID, connectionID)]; ok {
				delete(current.listeners, lsn)
			}
		})
	}, nil
}

// DeliverRealtimeEvent 把服务端推来的实时事件交给对应连接。集成层按通知路由
// 调用它；本包不改连接状态。
func (m *Manager) DeliverRealtimeEvent(event ModalityRealtimeEvent) error {
	sessionID, err := normalizeSessionID(event.SessionID)
	if err != nil {
		return err
	}
	connectionID, ok := trimNonEmpty(event.ConnectionID)
	if !ok {
		return fmt.Errorf("%w: 实时事件缺少连接标识", ErrInvalidModalityRequest)
	}
	event.SessionID = sessionID
	event.ConnectionID = connectionID
	if len(event.Event) == 0 || !json.Valid(event.Event) {
		return fmt.Errorf("%w: 实时事件不是合法 JSON", ErrInvalidModalityRequest)
	}

	m.mu.Lock()
	state, found := m.realtime[realtimeCacheKey(event.SessionID, connectionID)]
	if !found {
		m.mu.Unlock()
		return fmt.Errorf("%w: %s/%s", ErrRealtimeNotOpen, event.SessionID, connectionID)
	}
	listeners := snapshotRealtimeListeners(state)
	m.mu.Unlock()

	for _, listener := range listeners {
		notifyRealtimeEvent(listener.OnEvent, event)
	}
	return nil
}

// DeliverRealtimeEnded 把服务端推来的结束事件交给对应连接，并移除本地状态。
// 未知连接按幂等处理：重复的结束事件不报错。
func (m *Manager) DeliverRealtimeEnded(ended ModalityRealtimeEnded) error {
	sessionID, err := normalizeSessionID(ended.SessionID)
	if err != nil {
		return err
	}
	connectionID, ok := trimNonEmpty(ended.ConnectionID)
	if !ok {
		return fmt.Errorf("%w: 实时结束事件缺少连接标识", ErrInvalidModalityRequest)
	}
	m.endRealtimeState(sessionID, connectionID, ended.Reason)
	return nil
}

// RealtimeConnection 返回本进程确认仍打开的连接。
func (m *Manager) RealtimeConnection(sessionID, connectionID string) (ModalityRealtimeConnection, bool) {
	sessionID, ok := trimNonEmpty(sessionID)
	if !ok {
		return ModalityRealtimeConnection{}, false
	}
	connectionID, ok = trimNonEmpty(connectionID)
	if !ok {
		return ModalityRealtimeConnection{}, false
	}
	if !m.hasRealtimeState(sessionID, connectionID) {
		return ModalityRealtimeConnection{}, false
	}
	return ModalityRealtimeConnection{SessionID: sessionID, ConnectionID: connectionID}, true
}

func (m *Manager) addRealtimeState(sessionID, connectionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := realtimeCacheKey(sessionID, connectionID)
	if _, exists := m.realtime[key]; exists {
		return
	}
	m.realtime[key] = &realtimeState{
		sessionID:    sessionID,
		connectionID: connectionID,
		listeners:    make(map[int]ModalityRealtimeListener),
	}
}

func (m *Manager) hasRealtimeState(sessionID, connectionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, found := m.realtime[realtimeCacheKey(sessionID, connectionID)]
	return found
}

// endRealtimeState 移除连接并按结束原因通知订阅者；未知连接按幂等处理。
func (m *Manager) endRealtimeState(sessionID, connectionID, reason string) {
	m.mu.Lock()
	key := realtimeCacheKey(sessionID, connectionID)
	state, found := m.realtime[key]
	if !found {
		m.mu.Unlock()
		return
	}
	delete(m.realtime, key)
	listeners := snapshotRealtimeListeners(state)
	m.mu.Unlock()

	ended := ModalityRealtimeEnded{
		SessionID:    sessionID,
		ConnectionID: connectionID,
		Reason:       reason,
	}
	for _, listener := range listeners {
		notifyRealtimeEnded(listener.OnEnded, ended)
	}
}

func realtimeCacheKey(sessionID, connectionID string) string {
	return sessionID + "\x00" + connectionID
}

func snapshotRealtimeListeners(state *realtimeState) []ModalityRealtimeListener {
	listeners := make([]ModalityRealtimeListener, 0, len(state.listeners))
	for _, listener := range state.listeners {
		listeners = append(listeners, listener)
	}
	return listeners
}

// notifyRealtimeEvent 调用一个可选回调；回调 panic 被吞掉，不带走连接状态。
func notifyRealtimeEvent(fn func(ModalityRealtimeEvent), event ModalityRealtimeEvent) {
	if fn == nil {
		return
	}
	defer func() { _ = recover() }()
	fn(event)
}

func notifyRealtimeEnded(fn func(ModalityRealtimeEnded), ended ModalityRealtimeEnded) {
	if fn == nil {
		return
	}
	defer func() { _ = recover() }()
	fn(ended)
}
