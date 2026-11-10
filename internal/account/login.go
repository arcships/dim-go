package account

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// callbackPath 是环回回调的唯一路径。真实服务端只接受
	// http://127.0.0.1:<port>/auth/callback（或 localhost/::1 同路径）作为
	// 浏览器登录回调，因此这里固定成它。
	callbackPath = "/auth/callback"
	// DefaultLoginTimeout 是一次登录等待完成的默认上限。
	DefaultLoginTimeout = 5 * time.Minute
	// DefaultDevicePollInterval 是设备码轮询的默认间隔。
	DefaultDevicePollInterval = 5 * time.Second
	// callbackReadHeaderTimeout 限制回调请求头的读取时间，避免被慢连接占住。
	callbackReadHeaderTimeout = 5 * time.Second
	// callbackShutdownTimeout 是回调应答写出后关闭监听的上限。
	callbackShutdownTimeout = 2 * time.Second
	// authorizationCancelTimeout 是取消服务端登录尝试的上限；失败不影响本地结果。
	authorizationCancelTimeout = 3 * time.Second
)

// AuthorizationRequest 是服务端给出的一次授权请求。客户端只打开 URL，不解释、
// 不修改其中的查询参数。
type AuthorizationRequest struct {
	// URL 是要交给系统浏览器打开的地址。
	URL string
	// State 是服务端给出的回调校验值；空表示服务端不校验。
	State string
	// Attempt 是服务端给出的一次登录尝试标识；本包原样交回完成或取消。
	// 空表示服务端不需要按尝试区分。
	Attempt string
}

// AuthorizationResult 是环回回调收到的结果。Code 是一次性授权码，只在完成交换时
// 使用，本包不保存、不记录、不写入任何本地状态。
type AuthorizationResult struct {
	Code             string
	State            string
	Error            string
	ErrorDescription string
	// CallbackURL 是环回回调收到的完整地址（含查询串）。服务端按它完成交换；
	// 它包含一次性授权码，本包不保存、不记录。
	CallbackURL string
}

// AuthorizationSource 是浏览器登录的服务端侧接口：发起授权、用回调结果完成交换。
// 它由集成层实现，把 wire 形状映射到这两个方法；本包不复制 wire 定义。
type AuthorizationSource interface {
	// BeginAuthorization 用客户端生成的环回回调地址换一次授权请求。
	BeginAuthorization(ctx context.Context, redirectURI string) (AuthorizationRequest, error)
	// CompleteAuthorization 用回调结果与服务端的尝试标识完成交换。实现方只发
	// 一次，不自行重试。
	CompleteAuthorization(ctx context.Context, attempt string, result AuthorizationResult) error
}

// ProviderAuthorizationBegin 是按 provider 发起浏览器登录所需的参数。真实
// 服务端要求 providerId 与目录，不能只给回调地址。
type ProviderAuthorizationBegin struct {
	ProviderID  string
	Directory   string
	RedirectURI string
}

// ProviderAuthorizationSource 是可选能力：按 provider 发起与完成浏览器登录。
// 实现它时本包使用这些方法；否则退回 AuthorizationSource（单 provider 兼容）。
type ProviderAuthorizationSource interface {
	BeginProviderAuthorization(ctx context.Context, begin ProviderAuthorizationBegin) (AuthorizationRequest, error)
	CompleteProviderAuthorization(ctx context.Context, providerID, attempt string, result AuthorizationResult) error
}

// AuthorizationCanceler 是可选能力：取消服务端尚未结束的一次登录尝试。真实
// 服务端按尝试标识取消；不实现时本包只结束本地等待。
type AuthorizationCanceler interface {
	CancelAuthorization(ctx context.Context, attempt string) error
}

// Browser 打开系统浏览器。只在用户显式发起登录时调用。
type Browser interface {
	Open(url string) error
}

// loginAttempt 是一次进行中的登录尝试。尝试标识、授权码与设备码只在内存里，
// 不落盘；结束时交给服务端释放并清空。
type loginAttempt struct {
	provider  Provider
	method    LoginMethod
	attemptID string
	request   AuthorizationRequest
	device    DeviceAuthorization

	ctx      context.Context
	cancel   context.CancelFunc
	deadline time.Time

	listener  net.Listener
	server    *http.Server
	callbacks chan AuthorizationResult
	serveErr  chan error
	canceler  AuthorizationCanceler

	finishOnce  sync.Once
	mu          sync.Mutex
	waiting     bool
	releaseDone bool
}

func (attempt *loginAttempt) setWaiting(waiting bool) {
	attempt.mu.Lock()
	attempt.waiting = waiting
	attempt.mu.Unlock()
}

func (attempt *loginAttempt) isWaiting() bool {
	attempt.mu.Lock()
	defer attempt.mu.Unlock()
	return attempt.waiting
}

func (attempt *loginAttempt) setAttemptID(id string) {
	attempt.mu.Lock()
	attempt.attemptID = id
	attempt.mu.Unlock()
}

func (attempt *loginAttempt) attemptInfo() (string, AuthorizationCanceler) {
	attempt.mu.Lock()
	defer attempt.mu.Unlock()
	return attempt.attemptID, attempt.canceler
}

func (attempt *loginAttempt) setBrowserRequest(request AuthorizationRequest) {
	attempt.mu.Lock()
	attempt.request = request
	attempt.mu.Unlock()
}

func (attempt *loginAttempt) browserRequest() AuthorizationRequest {
	attempt.mu.Lock()
	defer attempt.mu.Unlock()
	return attempt.request
}

func (attempt *loginAttempt) setDevice(device DeviceAuthorization) {
	attempt.mu.Lock()
	attempt.device = device
	attempt.mu.Unlock()
}

func (attempt *loginAttempt) deviceInfo() DeviceAuthorization {
	attempt.mu.Lock()
	defer attempt.mu.Unlock()
	return attempt.device
}

// release 交回服务端释放尚未完成的尝试；没有尝试标识时保持未释放，等发起方
// 拿到标识后再调一次。只真正释放一次。
func (attempt *loginAttempt) release() {
	attempt.mu.Lock()
	if attempt.releaseDone || attempt.attemptID == "" || attempt.canceler == nil {
		attempt.mu.Unlock()
		return
	}
	attempt.releaseDone = true
	attemptID := attempt.attemptID
	canceler := attempt.canceler
	attempt.mu.Unlock()

	releaseCtx, cancel := context.WithTimeout(context.Background(), authorizationCancelTimeout)
	_ = canceler.CancelAuthorization(releaseCtx, attemptID)
	cancel()
}

// Login 按 provider 声明的登录方式发起并等待一次登录：先 `BeginLogin`，再
// `WaitLogin`。完成后重新读取账号状态。
func (a *Account) Login(ctx context.Context, request LoginRequest) (Snapshot, error) {
	if _, err := a.BeginLogin(ctx, request); err != nil {
		return a.Snapshot(), err
	}
	return a.WaitLogin(ctx)
}

// BeginLogin 解析 provider 与登录方式，发起一次登录尝试并返回要展示给用户的信息：
// 浏览器方式是授权地址，设备码方式是验证地址与用户码。它不等待用户完成。
//
// 登录方式只来自服务端声明：显式指定的方式必须被声明；未指定时优先 browser，
// 只有 provider 只声明 device 时使用 device。缺声明、缺能力或 provider 不存在
// 都明确报错，不在候选之间猜。
func (a *Account) BeginLogin(ctx context.Context, request LoginRequest) (LoginStart, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	provider, method, err := a.resolveLoginProvider(ctx, request)
	if err != nil {
		return LoginStart{}, err
	}
	switch method {
	case LoginMethodBrowser:
		if !a.hasBrowserAuthorization() {
			return LoginStart{}, fmt.Errorf("%w: provider %q 声明 browser，但本端没有浏览器授权来源",
				ErrLoginMethodUnavailable, provider.ID)
		}
	case LoginMethodDevice:
		if a.cfg.DeviceAuth == nil {
			return LoginStart{}, fmt.Errorf("%w: provider %q 声明 device，但本端没有设备码来源",
				ErrLoginMethodUnavailable, provider.ID)
		}
	default:
		return LoginStart{}, fmt.Errorf("%w: %q", ErrLoginMethodUnavailable, method)
	}

	attemptCtx, cancel := context.WithCancel(ctx)
	attempt := &loginAttempt{
		provider:  provider,
		method:    method,
		ctx:       attemptCtx,
		cancel:    cancel,
		deadline:  a.cfg.Now().Add(a.cfg.LoginTimeout),
		callbacks: make(chan AuthorizationResult, 1),
		serveErr:  make(chan error, 1),
	}
	if method == LoginMethodBrowser {
		if canceler, ok := a.cfg.Auth.(AuthorizationCanceler); ok {
			attempt.canceler = canceler
		}
	} else {
		if canceler, ok := a.cfg.DeviceAuth.(AuthorizationCanceler); ok {
			attempt.canceler = canceler
		}
	}

	// 浏览器方式先占好环回监听与回调服务；失败时不进入「等待回调」。
	if method == LoginMethodBrowser {
		listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
		if listenErr != nil {
			cancel()
			return LoginStart{}, fmt.Errorf("%w: %v", ErrLoginUnavailable, listenErr)
		}
		attempt.listener = listener
		attempt.server = &http.Server{
			Handler:           callbackHandler(attempt.callbacks),
			ReadHeaderTimeout: callbackReadHeaderTimeout,
		}
	}

	if !a.beginLogin(attempt) {
		cancel()
		if attempt.listener != nil {
			_ = attempt.listener.Close()
		}
		return LoginStart{}, ErrLoginInProgress
	}

	start := LoginStart{
		ProviderID: provider.ID,
		Method:     method,
		ExpiresAt:  attempt.deadline,
	}
	switch method {
	case LoginMethodBrowser:
		port := attempt.listener.Addr().(*net.TCPAddr).Port
		redirectURI := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + callbackPath
		go func() { attempt.serveErr <- attempt.server.Serve(attempt.listener) }()

		var authorization AuthorizationRequest
		var beginErr error
		if providerSource, ok := a.cfg.Auth.(ProviderAuthorizationSource); ok {
			authorization, beginErr = providerSource.BeginProviderAuthorization(attemptCtx, ProviderAuthorizationBegin{
				ProviderID:  provider.ID,
				Directory:   provider.Directory,
				RedirectURI: redirectURI,
			})
		} else if legacy, ok := a.cfg.Auth.(AuthorizationSource); ok {
			authorization, beginErr = legacy.BeginAuthorization(attemptCtx, redirectURI)
		} else {
			beginErr = ErrLoginUnavailable
		}
		if beginErr != nil {
			a.finishAttempt(attempt, LoginFailed, beginErr.Error())
			return LoginStart{}, fmt.Errorf("%w: %v", ErrLoginRequest, beginErr)
		}
		attempt.setBrowserRequest(authorization)
		attempt.setAttemptID(authorization.Attempt)
		if attempt.ctx.Err() != nil {
			attempt.release()
			return LoginStart{}, ErrLoginCanceled
		}
		if strings.TrimSpace(authorization.URL) == "" {
			a.finishAttempt(attempt, LoginFailed, "授权请求缺少地址")
			return LoginStart{}, fmt.Errorf("%w: 授权请求缺少地址", ErrLoginRequest)
		}
		if browserErr := a.cfg.Browser.Open(authorization.URL); browserErr != nil {
			a.finishAttempt(attempt, LoginFailed, browserErr.Error())
			return LoginStart{}, fmt.Errorf("%w: %v", ErrBrowserOpen, browserErr)
		}
		start.Attempt = authorization.Attempt
		start.AuthorizationURL = authorization.URL

	case LoginMethodDevice:
		device, beginErr := a.cfg.DeviceAuth.BeginDeviceAuthorization(attemptCtx, provider.ID, provider.Directory)
		if beginErr != nil {
			a.finishAttempt(attempt, LoginFailed, beginErr.Error())
			return LoginStart{}, fmt.Errorf("%w: %v", ErrLoginRequest, beginErr)
		}
		if strings.TrimSpace(device.Attempt) == "" {
			a.finishAttempt(attempt, LoginFailed, "设备码登录缺少尝试标识")
			return LoginStart{}, fmt.Errorf("%w: 设备码登录缺少尝试标识", ErrLoginRequest)
		}
		if strings.TrimSpace(device.VerificationURI) == "" || strings.TrimSpace(device.UserCode) == "" {
			a.finishAttempt(attempt, LoginFailed, "设备码登录缺少验证地址或用户码")
			return LoginStart{}, fmt.Errorf("%w: 设备码登录缺少验证地址或用户码", ErrLoginRequest)
		}
		attempt.setDevice(device)
		attempt.setAttemptID(device.Attempt)
		if attempt.ctx.Err() != nil {
			attempt.release()
			return LoginStart{}, ErrLoginCanceled
		}
		start.Attempt = device.Attempt
		start.VerificationURI = device.VerificationURI
		start.VerificationURIComplete = device.VerificationURIComplete
		start.UserCode = device.UserCode
		start.Interval = deviceInterval(device.Interval)
		if !device.ExpiresAt.IsZero() && device.ExpiresAt.Before(start.ExpiresAt) {
			start.ExpiresAt = device.ExpiresAt
		}
	}
	return start, nil
}

// WaitLogin 等待当前尝试完成：浏览器方式等环回回调并完成交换；设备码方式轮询
// 服务端状态。完成后重新读取账号状态；超时、取消与失败都明确返回错误并释放
// 服务端尝试。
func (a *Account) WaitLogin(ctx context.Context) (Snapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	attempt := a.currentAttempt()
	if attempt == nil {
		// 尝试在发起与等待之间被取消或超时时，返回对应原因，而不是笼统的「没有尝试」。
		switch state := a.LoginState(); state.Phase {
		case LoginCanceled:
			return a.Snapshot(), ErrLoginCanceled
		case LoginTimedOut:
			return a.Snapshot(), ErrLoginTimeout
		case LoginFailed:
			return a.Snapshot(), fmt.Errorf("%w: %s", ErrLoginUnavailable, state.Reason)
		}
		return a.Snapshot(), ErrNoLoginAttempt
	}
	attempt.setWaiting(true)
	defer attempt.setWaiting(false)

	var outcome loginOutcome
	switch attempt.method {
	case LoginMethodDevice:
		outcome = a.waitDevice(ctx, attempt)
	default:
		outcome = a.waitBrowser(ctx, attempt)
	}
	a.finishAttempt(attempt, outcome.phase, outcome.reason)
	if outcome.err != nil {
		return a.Snapshot(), outcome.err
	}
	snapshot, err := a.Refresh(ctx)
	if err != nil {
		return snapshot, fmt.Errorf("%w: %v", ErrLoginStatusUnread, err)
	}
	return snapshot, nil
}

type loginOutcome struct {
	phase  LoginPhase
	reason string
	err    error
}

func (a *Account) waitBrowser(ctx context.Context, attempt *loginAttempt) loginOutcome {
	waitCtx, cancel := context.WithDeadline(ctx, attempt.deadline)
	defer cancel()
	stop := context.AfterFunc(attempt.ctx, cancel)
	defer stop()

	select {
	case callback := <-attempt.callbacks:
		a.shutdownBrowser(attempt)
		if callback.Error != "" {
			reason := callback.Error
			if callback.ErrorDescription != "" {
				reason += ": " + callback.ErrorDescription
			}
			return loginOutcome{phase: LoginFailed, reason: reason,
				err: fmt.Errorf("%w: %s", ErrLoginDenied, reason)}
		}
		request := attempt.browserRequest()
		if request.State != "" {
			if subtle.ConstantTimeCompare([]byte(request.State), []byte(callback.State)) != 1 {
				return loginOutcome{phase: LoginFailed, reason: "回调校验值不一致", err: ErrLoginStateMismatch}
			}
		}
		if strings.TrimSpace(callback.Code) == "" {
			return loginOutcome{phase: LoginFailed, reason: "回调缺少授权码", err: ErrLoginIncomplete}
		}
		completion := AuthorizationResult{
			Code:        callback.Code,
			State:       callback.State,
			CallbackURL: callback.CallbackURL,
		}
		callback.Code = ""
		attemptID, _ := attempt.attemptInfo()
		var err error
		if providerSource, ok := a.cfg.Auth.(ProviderAuthorizationSource); ok {
			err = providerSource.CompleteProviderAuthorization(attempt.ctx, attempt.provider.ID, attemptID, completion)
		} else if legacy, ok := a.cfg.Auth.(AuthorizationSource); ok {
			err = legacy.CompleteAuthorization(attempt.ctx, attemptID, completion)
		} else {
			err = ErrLoginUnavailable
		}
		completion.Code = ""
		if err != nil {
			if attempt.ctx.Err() != nil {
				return loginOutcome{phase: LoginCanceled, reason: "登录已取消", err: ErrLoginCanceled}
			}
			return loginOutcome{phase: LoginFailed, reason: err.Error(),
				err: fmt.Errorf("%w: %v", ErrLoginCompletion, err)}
		}
		return loginOutcome{phase: LoginCompleted}

	case err := <-attempt.serveErr:
		// 取消会先关闭监听，serveErr 可能先于取消分支就绪；按取消处理。
		if attempt.ctx.Err() != nil {
			return loginOutcome{phase: LoginCanceled, reason: "登录已取消", err: ErrLoginCanceled}
		}
		if err == nil {
			err = errors.New("回调监听已关闭")
		}
		return loginOutcome{phase: LoginFailed, reason: err.Error(),
			err: fmt.Errorf("%w: 回调监听失败: %v", ErrLoginUnavailable, err)}

	case <-waitCtx.Done():
		return a.waitOutcome(ctx, attempt, false)
	}
}

func (a *Account) waitDevice(ctx context.Context, attempt *loginAttempt) loginOutcome {
	deviceInfo := attempt.deviceInfo()
	deadline := attempt.deadline
	deviceExpiry := false
	if !deviceInfo.ExpiresAt.IsZero() && deviceInfo.ExpiresAt.Before(deadline) {
		deadline = deviceInfo.ExpiresAt
		deviceExpiry = true
	}
	waitCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	stop := context.AfterFunc(attempt.ctx, cancel)
	defer stop()

	interval := deviceInterval(deviceInfo.Interval)
	attemptID, _ := attempt.attemptInfo()
	for {
		state, err := a.cfg.DeviceAuth.PollDeviceAuthorization(waitCtx, attemptID)
		if err != nil {
			if waitCtx.Err() != nil {
				outcome := a.waitOutcome(ctx, attempt, deviceExpiry)
				if deviceExpiry && outcome.err == ErrLoginTimeout {
					outcome.reason = "设备码登录已过期"
					outcome.err = ErrLoginExpired
				}
				return outcome
			}
			return loginOutcome{phase: LoginFailed, reason: err.Error(),
				err: fmt.Errorf("%w: %v", ErrDeviceLogin, err)}
		}
		switch state.Status {
		case DeviceLoginPending:
			timer := time.NewTimer(interval)
			select {
			case <-waitCtx.Done():
				timer.Stop()
				outcome := a.waitOutcome(ctx, attempt, deviceExpiry)
				if deviceExpiry && outcome.err == ErrLoginTimeout {
					outcome.reason = "设备码登录已过期"
					outcome.err = ErrLoginExpired
				}
				return outcome
			case <-timer.C:
			}
		case DeviceLoginCompleted:
			return loginOutcome{phase: LoginCompleted}
		case DeviceLoginFailed:
			reason := strings.TrimSpace(state.Reason)
			if reason == "" {
				reason = "服务端报告登录失败"
			}
			return loginOutcome{phase: LoginFailed, reason: reason,
				err: fmt.Errorf("%w: %s", ErrDeviceLogin, reason)}
		case DeviceLoginExpired:
			return loginOutcome{phase: LoginTimedOut, reason: "设备码登录已过期", err: ErrLoginExpired}
		case DeviceLoginCanceled:
			return loginOutcome{phase: LoginCanceled, reason: "设备码登录已取消", err: ErrLoginCanceled}
		default:
			reason := fmt.Sprintf("未知设备登录状态 %q", state.Status)
			return loginOutcome{phase: LoginFailed, reason: reason,
				err: fmt.Errorf("%w: %s", ErrDeviceLogin, reason)}
		}
	}
}

// waitOutcome 把等待结束归因为取消或超时。
func (a *Account) waitOutcome(parent context.Context, attempt *loginAttempt, deviceExpiry bool) loginOutcome {
	if parent.Err() != nil {
		if errors.Is(parent.Err(), context.DeadlineExceeded) {
			return loginOutcome{phase: LoginTimedOut, reason: "等待登录超时", err: ErrLoginTimeout}
		}
		return loginOutcome{phase: LoginCanceled, reason: parent.Err().Error(), err: ErrLoginCanceled}
	}
	if attempt.ctx.Err() != nil {
		return loginOutcome{phase: LoginCanceled, reason: "登录已取消", err: ErrLoginCanceled}
	}
	if deviceExpiry {
		return loginOutcome{phase: LoginTimedOut, reason: "设备码登录已过期", err: ErrLoginExpired}
	}
	return loginOutcome{phase: LoginTimedOut, reason: "等待登录超时", err: ErrLoginTimeout}
}

// finishAttempt 结束一次尝试：交回服务端释放（成功完成时不释放）、关闭本地
// 监听、推进登录尝试状态并清空引用。同一尝试只结束一次。
func (a *Account) finishAttempt(attempt *loginAttempt, phase LoginPhase, reason string) {
	attempt.finishOnce.Do(func() {
		if phase != LoginCompleted {
			attempt.release()
		}
		attempt.cancel()
		if attempt.server != nil {
			_ = attempt.server.Close()
		}
		if attempt.listener != nil {
			_ = attempt.listener.Close()
		}
		if event, ok := loginEventForPhase(phase); ok {
			a.advanceLogin(event, reason)
		}
		a.mu.Lock()
		if a.login == attempt {
			a.login = nil
		}
		a.mu.Unlock()
	})
}

func loginEventForPhase(phase LoginPhase) (loginEvent, bool) {
	switch phase {
	case LoginCompleted:
		return loginEventCompleted, true
	case LoginFailed:
		return loginEventFailed, true
	case LoginCanceled:
		return loginEventCanceled, true
	case LoginTimedOut:
		return loginEventTimedOut, true
	default:
		return "", false
	}
}

func (a *Account) shutdownBrowser(attempt *loginAttempt) {
	if attempt.server == nil {
		return
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), callbackShutdownTimeout)
	_ = attempt.server.Shutdown(shutdownCtx)
	cancel()
}

func (a *Account) hasBrowserAuthorization() bool {
	if a.cfg.Auth == nil {
		return false
	}
	if _, ok := a.cfg.Auth.(AuthorizationSource); ok {
		return true
	}
	_, ok := a.cfg.Auth.(ProviderAuthorizationSource)
	return ok
}

func (a *Account) currentAttempt() *loginAttempt {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.login
}

func deviceInterval(interval time.Duration) time.Duration {
	if interval <= 0 {
		return DefaultDevicePollInterval
	}
	return interval
}

// CancelLogin 取消进行中的登录；没有进行中的登录时什么也不做。等待中的
// `WaitLogin` 会返回取消；没有等待者时这里直接结束尝试并交回服务端释放。
func (a *Account) CancelLogin() {
	attempt := a.currentAttempt()
	if attempt == nil {
		return
	}
	attempt.cancel()
	if !attempt.isWaiting() {
		a.finishAttempt(attempt, LoginCanceled, "登录已取消")
	}
}

// callbackHandler 接收环回回调。只接受 GET；应答页面固定文案，不回显授权码或
// 任何请求参数。
func callbackHandler(callbacks chan<- AuthorizationResult) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		query := r.URL.Query()
		result := AuthorizationResult{
			Code:             query.Get("code"),
			State:            query.Get("state"),
			Error:            query.Get("error"),
			ErrorDescription: query.Get("error_description"),
			CallbackURL:      "http://" + r.Host + r.URL.RequestURI(),
		}
		message := "登录完成，可以回到客户端。"
		if result.Error != "" {
			message = "登录未完成：授权服务端返回了拒绝或错误。"
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, "<!doctype html><html><head><meta charset=\"utf-8\"><title>DimAgent</title></head><body><p>"+message+"</p></body></html>")
		select {
		case callbacks <- result:
		default:
		}
	})
	return mux
}

// 登录相关错误。
var (
	// ErrLoginUnavailable 表示本端不提供登录（缺授权来源或无法监听环回地址）。
	ErrLoginUnavailable = errors.New("账号登录不可用")
	// ErrLoginInProgress 表示已有一次登录在进行中。
	ErrLoginInProgress = errors.New("已有登录在进行")
	// ErrNoLoginAttempt 表示没有已发起的登录尝试。
	ErrNoLoginAttempt = errors.New("没有进行中的登录尝试")
	// ErrLoginRequest 表示授权或设备码请求发起失败。
	ErrLoginRequest = errors.New("登录请求失败")
	// ErrBrowserOpen 表示打开系统浏览器失败。
	ErrBrowserOpen = errors.New("打开系统浏览器失败")
	// ErrLoginTimeout 表示等待登录超时。
	ErrLoginTimeout = errors.New("登录等待超时")
	// ErrLoginCanceled 表示登录被取消。
	ErrLoginCanceled = errors.New("登录已取消")
	// ErrLoginDenied 表示授权服务端拒绝或回调带回错误。
	ErrLoginDenied = errors.New("登录被拒绝")
	// ErrLoginStateMismatch 表示回调校验值不一致。
	ErrLoginStateMismatch = errors.New("登录回调校验值不一致")
	// ErrLoginIncomplete 表示回调缺少授权码。
	ErrLoginIncomplete = errors.New("登录回调缺少授权码")
	// ErrLoginCompletion 表示用回调结果完成交换失败。
	ErrLoginCompletion = errors.New("登录完成交换失败")
	// ErrLoginExpired 表示服务端报告设备码尝试已过期。
	ErrLoginExpired = errors.New("登录尝试已过期")
	// ErrDeviceLogin 表示设备码登录失败。
	ErrDeviceLogin = errors.New("设备码登录失败")
	// ErrLoginStatusUnread 表示登录已完成，但随后的账号状态读取失败。
	ErrLoginStatusUnread = errors.New("登录已完成，但账号状态读取失败")
)
