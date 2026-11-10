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
	"time"
)

const (
	// callbackPath 是环回回调的唯一路径。浏览器被授权服务端重定向回来后只会命中它。
	callbackPath = "/account/callback"
	// DefaultLoginTimeout 是一次登录等待回调的默认上限。
	DefaultLoginTimeout = 5 * time.Minute
	// callbackReadHeaderTimeout 限制回调请求头的读取时间，避免被慢连接占住。
	callbackReadHeaderTimeout = 5 * time.Second
	// callbackShutdownTimeout 是回调应答写出后关闭监听的上限。
	callbackShutdownTimeout = 2 * time.Second
)

// AuthorizationRequest 是服务端给出的一次授权请求。客户端只打开 URL，不解释、
// 不修改其中的查询参数。
type AuthorizationRequest struct {
	// URL 是要交给系统浏览器打开的地址。
	URL string
	// State 是服务端给出的回调校验值；空表示服务端不校验。
	State string
}

// AuthorizationResult 是环回回调收到的结果。Code 是一次性授权码，只在完成交换时
// 使用，本包不保存、不记录、不写入任何本地状态。
type AuthorizationResult struct {
	Code             string
	State            string
	Error            string
	ErrorDescription string
}

// AuthorizationSource 是浏览器登录的服务端侧接口：发起授权、用回调结果完成交换。
// 它由集成层实现，把 wire 形状映射到这两个方法；本包不复制 wire 定义。
type AuthorizationSource interface {
	// BeginAuthorization 用客户端生成的环回回调地址换一次授权请求。
	BeginAuthorization(ctx context.Context, redirectURI string) (AuthorizationRequest, error)
	// CompleteAuthorization 用回调结果完成交换。实现方只发一次，不自行重试。
	CompleteAuthorization(ctx context.Context, result AuthorizationResult) error
}

// Browser 打开系统浏览器。只在用户显式发起登录时调用。
type Browser interface {
	Open(url string) error
}

// Login 发起一次浏览器登录：生成环回回调地址、请服务端给出授权请求、打开系统
// 浏览器、等待回调、把回调结果交给授权来源完成交换，最后重新读取账号状态。
//
// 超时、取消、回调校验失败与服务端拒绝都会明确返回错误；此时本地账号状态不变。
// 授权码不会留在 Account 里，也不写日志。
func (a *Account) Login(ctx context.Context) (Snapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if a.cfg.Auth == nil {
		return a.Snapshot(), ErrLoginUnavailable
	}

	// 只绑环回地址：系统浏览器与授权服务端都只应访问本机回调。
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return a.Snapshot(), fmt.Errorf("%w: %v", ErrLoginUnavailable, err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + callbackPath

	attemptCtx, cancel := context.WithCancel(ctx)
	if !a.beginLogin(cancel) {
		cancel()
		_ = listener.Close()
		return a.Snapshot(), ErrLoginInProgress
	}
	defer func() {
		a.finishLogin()
		cancel()
		_ = listener.Close()
	}()

	callbacks := make(chan AuthorizationResult, 1)
	server := &http.Server{
		Handler:           callbackHandler(callbacks),
		ReadHeaderTimeout: callbackReadHeaderTimeout,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	request, err := a.cfg.Auth.BeginAuthorization(attemptCtx, redirectURI)
	if err != nil {
		a.advanceLogin(loginEventFailed, err.Error())
		return a.Snapshot(), fmt.Errorf("%w: %v", ErrLoginRequest, err)
	}
	if strings.TrimSpace(request.URL) == "" {
		a.advanceLogin(loginEventFailed, "授权请求缺少地址")
		return a.Snapshot(), fmt.Errorf("%w: 授权请求缺少地址", ErrLoginRequest)
	}
	if err := a.cfg.Browser.Open(request.URL); err != nil {
		a.advanceLogin(loginEventFailed, err.Error())
		return a.Snapshot(), fmt.Errorf("%w: %v", ErrBrowserOpen, err)
	}

	callback, err := a.waitCallback(attemptCtx, ctx, callbacks, serveErr)
	if err != nil {
		return a.Snapshot(), err
	}
	// 回执已经交给调用方；给浏览器留一点时间读完应答再关闭监听。
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), callbackShutdownTimeout)
	_ = server.Shutdown(shutdownCtx)
	shutdownCancel()

	if callback.Error != "" {
		reason := callback.Error
		if callback.ErrorDescription != "" {
			reason += ": " + callback.ErrorDescription
		}
		a.advanceLogin(loginEventFailed, reason)
		return a.Snapshot(), fmt.Errorf("%w: %s", ErrLoginDenied, reason)
	}
	if request.State != "" {
		if subtle.ConstantTimeCompare([]byte(request.State), []byte(callback.State)) != 1 {
			a.advanceLogin(loginEventFailed, "回调校验值不一致")
			return a.Snapshot(), ErrLoginStateMismatch
		}
	}
	if strings.TrimSpace(callback.Code) == "" {
		a.advanceLogin(loginEventFailed, "回调缺少授权码")
		return a.Snapshot(), ErrLoginIncomplete
	}

	completion := AuthorizationResult{Code: callback.Code, State: callback.State}
	callback.Code = ""
	err = a.cfg.Auth.CompleteAuthorization(attemptCtx, completion)
	completion.Code = ""
	if err != nil {
		a.advanceLogin(loginEventFailed, err.Error())
		return a.Snapshot(), fmt.Errorf("%w: %v", ErrLoginCompletion, err)
	}
	a.advanceLogin(loginEventCompleted, "")

	snapshot, err := a.Refresh(ctx)
	if err != nil {
		return snapshot, fmt.Errorf("%w: %v", ErrLoginStatusUnread, err)
	}
	return snapshot, nil
}

// waitCallback 等待环回回调、超时或取消。超时与取消分别返回稳定错误，并且都不会
// 继续调用完成交换。
func (a *Account) waitCallback(
	attemptCtx context.Context,
	parent context.Context,
	callbacks <-chan AuthorizationResult,
	serveErr <-chan error,
) (AuthorizationResult, error) {
	timer := time.NewTimer(a.cfg.LoginTimeout)
	defer timer.Stop()

	select {
	case callback := <-callbacks:
		return callback, nil
	case err := <-serveErr:
		if err == nil {
			err = errors.New("回调监听已关闭")
		}
		a.advanceLogin(loginEventFailed, err.Error())
		return AuthorizationResult{}, fmt.Errorf("%w: 回调监听失败: %v", ErrLoginUnavailable, err)
	case <-attemptCtx.Done():
		if parent.Err() != nil {
			a.advanceLogin(loginEventCanceled, parent.Err().Error())
			return AuthorizationResult{}, ErrLoginCanceled
		}
		a.advanceLogin(loginEventCanceled, "登录已取消")
		return AuthorizationResult{}, ErrLoginCanceled
	case <-timer.C:
		a.advanceLogin(loginEventTimedOut, "等待回调超时")
		return AuthorizationResult{}, ErrLoginTimeout
	}
}

// CancelLogin 取消进行中的登录；没有进行中的登录时什么也不做。它只结束等待，
// 不改变账号状态。
func (a *Account) CancelLogin() {
	a.mu.Lock()
	cancel := a.loginCancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
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
	// ErrLoginRequest 表示授权请求发起失败。
	ErrLoginRequest = errors.New("授权请求失败")
	// ErrBrowserOpen 表示打开系统浏览器失败。
	ErrBrowserOpen = errors.New("打开系统浏览器失败")
	// ErrLoginTimeout 表示等待回调超时。
	ErrLoginTimeout = errors.New("登录回调超时")
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
	// ErrLoginStatusUnread 表示登录已完成，但随后的账号状态读取失败。
	ErrLoginStatusUnread = errors.New("登录已完成，但账号状态读取失败")
)
