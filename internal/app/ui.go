package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
)

// 界面注册的本地状态。
const (
	UIRegistrationOnline   = "online"
	UIRegistrationDetached = "detached"
	UIRegistrationShutdown = "shutdown_requested"
	UIRegistrationClosed   = "closed"
	uiAttachKind           = "gui"
	uiReattachTimeout      = 10 * time.Second
)

// UIRegistration 是一次界面注册的只读状态。
type UIRegistration struct {
	SessionID  string
	UIID       string
	Generation int64
	State      string
}

// uiRegistrationState 是会话内的界面注册状态；只在内存里，不落盘。
type uiRegistrationState struct {
	mu            sync.Mutex
	uiID          string
	generation    int64
	state         string
	hasRegistered bool
}

func (s *uiRegistrationState) store(uiID string, generation int64, state string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uiID = uiID
	s.generation = generation
	s.state = state
	s.hasRegistered = true
}

func (s *uiRegistrationState) snapshot(sessionID string) (UIRegistration, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasRegistered {
		return UIRegistration{}, false
	}
	return UIRegistration{
		SessionID:  sessionID,
		UIID:       s.uiID,
		Generation: s.generation,
		State:      s.state,
	}, true
}

func (s *uiRegistrationState) setState(state string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hasRegistered {
		s.state = state
	}
}

func (s *uiRegistrationState) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hasRegistered {
		s.state = UIRegistrationClosed
	}
	s.uiID = ""
	s.generation = 0
	s.hasRegistered = false
}

// UIShutdownEvent 是 daemon 发来的一条建议性关闭通知。
type UIShutdownEvent struct {
	UIID      string
	SessionID string
	Reason    string
}

// AttachUI 在已 attach 的会话连接上注册一个普通界面（`ui.attach {sessionId, kind:"gui"}`）。
//
// 只对本机会话流开放；门禁 hello-first。成功保存 uiId 与 generation。
func (a *App) AttachUI(ctx context.Context, sessionID string) (UIRegistration, error) {
	runtime, err := a.openRuntime(sessionID, protocol.MethodUIAttach)
	if err != nil {
		return UIRegistration{}, err
	}
	if !a.sessionSupportsMethod(ctx, runtime, protocol.MethodUIAttach) {
		return UIRegistration{}, capabilityMissing(protocol.MethodUIAttach)
	}
	raw, err := runtime.lease.Subscription().Call(ctx, protocol.MethodUIAttach, protocol.UIAttachParams{
		SessionID: sessionID,
		Kind:      uiAttachKind,
	})
	if err != nil {
		return UIRegistration{}, err
	}
	attached, err := protocol.ParseUIAttached(raw)
	if err != nil {
		return UIRegistration{}, fmt.Errorf("界面注册回执无法解析: %w", err)
	}
	runtime.ui.store(attached.UIID, attached.Generation, UIRegistrationOnline)
	return UIRegistration{
		SessionID:  sessionID,
		UIID:       attached.UIID,
		Generation: attached.Generation,
		State:      UIRegistrationOnline,
	}, nil
}

// CloseUI 移除注册（`ui.close {uiId, reason?}`）并清理本地状态。
//
// 没有注册时不做任何请求；会话已关闭时只清本地状态（服务端会随会话移除注册）。
func (a *App) CloseUI(ctx context.Context, sessionID, reason string) error {
	runtime, ok := a.session(sessionID)
	if !ok {
		return nil
	}
	registration, registered := runtime.ui.snapshot(sessionID)
	if !registered {
		return nil
	}
	if !a.sessionSupportsMethod(ctx, runtime, protocol.MethodUIClose) {
		// 能力缺失：仍清掉本地注册，不让界面状态悬空。
		runtime.ui.clear()
		return capabilityMissing(protocol.MethodUIClose)
	}
	params := protocol.UICloseParams{UIID: registration.UIID}
	if reason != "" {
		params.Reason = &reason
	}
	_, err := runtime.lease.Subscription().Call(ctx, protocol.MethodUIClose, params)
	runtime.ui.clear()
	return err
}

// UIRegistration 返回本会话的界面注册状态；没有注册时为 false。
func (a *App) UIRegistration(sessionID string) (UIRegistration, bool) {
	runtime, ok := a.session(sessionID)
	if !ok {
		return UIRegistration{}, false
	}
	return runtime.ui.snapshot(sessionID)
}

// UIStatus 读一次界面清单（`ui.status`，控制连接）；只用于自检与诊断。
func (a *App) UIStatus(ctx context.Context) (protocol.UIStatusResult, error) {
	if !a.controlSupports(protocol.MethodUIStatus) {
		return protocol.UIStatusResult{}, capabilityMissing(protocol.MethodUIStatus)
	}
	client, err := a.requireControl("读取界面清单")
	if err != nil {
		return protocol.UIStatusResult{}, err
	}
	raw, err := client.Call(ctx, protocol.MethodUIStatus, map[string]any{})
	if err != nil {
		return protocol.UIStatusResult{}, err
	}
	return protocol.ParseUIStatusResult(raw)
}

// routeUIShutdown 把 `ui.shutdown` 当建议性关闭处理：记录、标记状态、交宿主决定。
func (a *App) routeUIShutdown(event session.Event) {
	a.routeUIShutdownMessage(event.Method, event.Params)
}

// routeUIShutdownMessage 是 ui.shutdown 的共用处理（会话流与控制连接都可能送）。
func (a *App) routeUIShutdownMessage(method string, params json.RawMessage) {
	notice, err := protocol.ParseUIShutdownNotice(params)
	if err != nil {
		a.record("界面关闭通知无法解析: %v", err)
		return
	}
	sessionID := a.sessionIDForUI(notice.UIID)
	if sessionID != "" {
		if runtime, ok := a.session(sessionID); ok {
			// 控制连接与会话流可能各送一次；已经标记过就不再重复记录。
			if registration, known := runtime.ui.snapshot(sessionID); known && registration.State == UIRegistrationShutdown {
				return
			}
			runtime.ui.setState(UIRegistrationShutdown)
		}
	}
	a.mu.Lock()
	a.uiShutdowns = append(a.uiShutdowns, UIShutdownEvent{
		UIID:      notice.UIID,
		SessionID: sessionID,
		Reason:    notice.Reason,
	})
	a.mu.Unlock()
	a.record("界面关闭建议: uiId=%s 会话=%s 原因=%s", notice.UIID, sessionID, notice.Reason)
}

// sessionIDForUI 按 uiId 找本地注册的会话。
func (a *App) sessionIDForUI(uiID string) string {
	a.mu.Lock()
	runtimes := make([]*sessionRuntime, 0, len(a.sessions))
	for _, runtime := range a.sessions {
		runtimes = append(runtimes, runtime)
	}
	a.mu.Unlock()
	for _, runtime := range runtimes {
		if registration, ok := runtime.ui.snapshot(runtime.id); ok && registration.UIID == uiID {
			return runtime.id
		}
	}
	return ""
}

// DrainUIShutdowns 取走尚未处理的建议性关闭通知（宿主决定如何响应）。
func (a *App) DrainUIShutdowns() []UIShutdownEvent {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.uiShutdowns) == 0 {
		return nil
	}
	events := a.uiShutdowns
	a.uiShutdowns = nil
	return events
}

// UIErrorCode 取界面方法失败的稳定错误码（例如 ui_in_use、session_not_attached）。
func UIErrorCode(err error) string {
	var rpcErr *protocol.RPCError
	if errors.As(err, &rpcErr) {
		return rpcErr.AppError()
	}
	return ""
}

// markUIDetached 在会话流断开时把注册标成 detached（等重连后再注册）。
func (a *App) markUIDetached(runtime *sessionRuntime) {
	runtime.ui.setState(UIRegistrationDetached)
}

// reattachUIAfterResync 在重连重读后重新注册界面；失败只记诊断。
func (a *App) reattachUIAfterResync(runtime *sessionRuntime) {
	registration, ok := runtime.ui.snapshot(runtime.id)
	if !ok || registration.State == UIRegistrationShutdown {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), uiReattachTimeout)
	defer cancel()
	attached, err := a.AttachUI(ctx, runtime.id)
	if err != nil {
		a.record("重连后重新注册界面失败: %v", err)
		return
	}
	a.record("重连后界面注册已恢复: uiId=%s generation=%d", attached.UIID, attached.Generation)
}
