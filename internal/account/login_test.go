package account

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// httpBrowser 是打开系统浏览器的替身：立刻按授权地址发起一次回调请求，模拟用户
// 在浏览器里完成授权后被重定向回环回地址。
type httpBrowser struct {
	mu     sync.Mutex
	opened []string
	bodies []string
}

func (b *httpBrowser) Open(rawURL string) error {
	b.mu.Lock()
	b.opened = append(b.opened, rawURL)
	b.mu.Unlock()

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(rawURL)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)

	b.mu.Lock()
	b.bodies = append(b.bodies, string(body))
	b.mu.Unlock()
	return nil
}

func (b *httpBrowser) observed() ([]string, []string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.opened...), append([]string(nil), b.bodies...)
}

// TestLoginOpensBrowserReceivesCallbackAndRefreshes 覆盖成功路径：只绑环回地址、
// 打开系统浏览器、收到回调、把授权码交给完成交换、最后重新读取账号状态。
func TestLoginOpensBrowserReceivesCallbackAndRefreshes(t *testing.T) {
	status := &stubStatusReader{fact: Fact{Status: StatusSignedOut}}
	auth := callbackAuth("code-1", "state-1", "state-1")
	auth.onComplete = func(AuthorizationResult) {
		status.set(signedInFact(), nil)
	}
	browser := &httpBrowser{}
	account := newTestAccount(t, Config{
		Status:       status,
		Providers:    browserProviderSource(),
		Auth:         auth,
		Browser:      browser,
		LoginTimeout: 2 * time.Second,
	})

	snapshot, err := account.Login(context.Background(), LoginRequest{})
	if err != nil {
		t.Fatalf("登录: %v", err)
	}
	if snapshot.Status != StatusSignedIn || snapshot.Identity.Email != "user@example.com" {
		t.Fatalf("登录后的账号状态不符：%+v", snapshot)
	}
	if state := account.LoginState(); state.Phase != LoginCompleted {
		t.Fatalf("登录尝试状态不符：%+v", state)
	}

	redirects := auth.redirectList()
	if len(redirects) != 1 {
		t.Fatalf("授权发起次数不符：%d", len(redirects))
	}
	parsed, err := url.Parse(redirects[0])
	if err != nil {
		t.Fatalf("回调地址无法解析: %v", err)
	}
	if parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" || parsed.Path != callbackPath {
		t.Fatalf("回调地址不是环回地址：%q", redirects[0])
	}

	opened, bodies := browser.observed()
	if len(opened) != 1 || !strings.HasPrefix(opened[0], redirects[0]) {
		t.Fatalf("没有把授权地址原样交给系统浏览器：%v", opened)
	}
	completions := auth.completeList()
	if len(completions) != 1 || completions[0].Code != "code-1" || completions[0].State != "state-1" {
		t.Fatalf("完成交换的结果不符：%+v", completions)
	}
	if strings.TrimSpace(completions[0].CallbackURL) == "" ||
		!strings.HasPrefix(completions[0].CallbackURL, redirects[0]) ||
		!strings.Contains(completions[0].CallbackURL, "code=code-1") {
		t.Fatalf("完成交换没有带上完整回调地址：%q", completions[0].CallbackURL)
	}
	if attempts := auth.attemptList(); len(attempts) != 1 || attempts[0] != "attempt-1" {
		t.Fatalf("完成交换没有交回服务端尝试标识：%v", attempts)
	}
	if cancels := auth.cancelList(); len(cancels) != 0 {
		t.Fatalf("成功的登录不应取消服务端尝试：%v", cancels)
	}
	if len(bodies) != 1 {
		t.Fatalf("回调应答次数不符：%d", len(bodies))
	}
	if strings.Contains(bodies[0], "code-1") {
		t.Fatal("回调应答回显了授权码")
	}
}

// TestLoginTimeout 确认等待回调超时后明确返回超时，不再调用完成交换。
func TestLoginTimeout(t *testing.T) {
	auth := callbackAuth("code-1", "state-1", "state-1")
	account := newTestAccount(t, Config{
		Status:       &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:    browserProviderSource(),
		Auth:         auth,
		Browser:      &stubBrowser{},
		LoginTimeout: 50 * time.Millisecond,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取未登录状态: %v", err)
	}

	_, err := account.Login(context.Background(), LoginRequest{})
	if !errors.Is(err, ErrLoginTimeout) {
		t.Fatalf("应返回登录超时：%v", err)
	}
	if len(auth.completeList()) != 0 {
		t.Fatal("超时后不应调用完成交换")
	}
	if state := account.LoginState(); state.Phase != LoginTimedOut || state.Reason == "" {
		t.Fatalf("超时后的登录尝试状态不符：%+v", state)
	}
	if account.Status() != StatusSignedOut {
		t.Fatalf("超时不应改变账号状态：%q", account.Status())
	}
	if cancels := auth.cancelList(); len(cancels) != 1 || cancels[0] != "attempt-1" {
		t.Fatalf("超时后应交回服务端释放尝试：%v", cancels)
	}
}

// TestLoginDeniedByAuthorizationServer 确认授权服务端拒绝时登录失败且不完成交换。
func TestLoginDeniedByAuthorizationServer(t *testing.T) {
	auth := &stubAuth{}
	auth.onBegin = func(redirectURI string) AuthorizationRequest {
		query := url.Values{
			"error":             {"access_denied"},
			"error_description": {"用户拒绝了授权"},
			"state":             {"state-1"},
		}
		return AuthorizationRequest{
			URL:   redirectURI + "?" + query.Encode(),
			State: "state-1",
		}
	}
	account := newTestAccount(t, Config{
		Status:       &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:    browserProviderSource(),
		Auth:         auth,
		Browser:      &httpBrowser{},
		LoginTimeout: 2 * time.Second,
	})

	_, err := account.Login(context.Background(), LoginRequest{})
	if !errors.Is(err, ErrLoginDenied) {
		t.Fatalf("应返回登录被拒绝：%v", err)
	}
	if len(auth.completeList()) != 0 {
		t.Fatal("被拒绝后不应调用完成交换")
	}
	if state := account.LoginState(); state.Phase != LoginFailed {
		t.Fatalf("登录尝试状态不符：%+v", state)
	}
}

// TestLoginStateMismatch 确认回调校验值不一致时拒绝完成交换。
func TestLoginStateMismatch(t *testing.T) {
	auth := callbackAuth("code-1", "state-server", "state-other")
	account := newTestAccount(t, Config{
		Status:       &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:    browserProviderSource(),
		Auth:         auth,
		Browser:      &httpBrowser{},
		LoginTimeout: 2 * time.Second,
	})

	_, err := account.Login(context.Background(), LoginRequest{})
	if !errors.Is(err, ErrLoginStateMismatch) {
		t.Fatalf("应返回校验值不一致：%v", err)
	}
	if len(auth.completeList()) != 0 {
		t.Fatal("校验失败后不应调用完成交换")
	}
}

// TestLoginIncompleteCallback 确认回调缺少授权码时明确失败。
func TestLoginIncompleteCallback(t *testing.T) {
	auth := callbackAuth("", "state-1", "state-1")
	account := newTestAccount(t, Config{
		Status:       &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:    browserProviderSource(),
		Auth:         auth,
		Browser:      &httpBrowser{},
		LoginTimeout: 2 * time.Second,
	})

	_, err := account.Login(context.Background(), LoginRequest{})
	if !errors.Is(err, ErrLoginIncomplete) {
		t.Fatalf("应返回回调缺少授权码：%v", err)
	}
	if len(auth.completeList()) != 0 {
		t.Fatal("缺少授权码时不应调用完成交换")
	}
}

// TestLoginCompletionFailure 确认完成交换失败时账号状态不变。
func TestLoginCompletionFailure(t *testing.T) {
	auth := callbackAuth("code-1", "state-1", "state-1")
	auth.completeErr = errors.New("交换被服务端拒绝")
	account := newTestAccount(t, Config{
		Status:       &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:    browserProviderSource(),
		Auth:         auth,
		Browser:      &httpBrowser{},
		LoginTimeout: 2 * time.Second,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取未登录状态: %v", err)
	}

	_, err := account.Login(context.Background(), LoginRequest{})
	if !errors.Is(err, ErrLoginCompletion) {
		t.Fatalf("应返回完成交换失败：%v", err)
	}
	if account.Status() != StatusSignedOut {
		t.Fatalf("交换失败不应改变账号状态：%q", account.Status())
	}
	if len(auth.completeList()) != 1 {
		t.Fatalf("完成交换应只调用一次：%d", len(auth.completeList()))
	}
}

// TestLoginCancel 确认取消登录会结束等待，并让登录尝试停在已取消。
func TestLoginCancel(t *testing.T) {
	auth := callbackAuth("code-1", "state-1", "state-1")
	account := newTestAccount(t, Config{
		Status:       &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:    browserProviderSource(),
		Auth:         auth,
		Browser:      &stubBrowser{},
		LoginTimeout: 5 * time.Second,
	})

	done := make(chan error, 1)
	go func() {
		_, err := account.Login(context.Background(), LoginRequest{})
		done <- err
	}()
	waitForLoginPhase(t, account, LoginWaiting)
	account.CancelLogin()

	select {
	case err := <-done:
		if !errors.Is(err, ErrLoginCanceled) {
			t.Fatalf("应返回登录已取消：%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("取消后登录没有结束")
	}
	if state := account.LoginState(); state.Phase != LoginCanceled {
		t.Fatalf("登录尝试状态不符：%+v", state)
	}
	if cancels := auth.cancelList(); len(cancels) != 1 || cancels[0] != "attempt-1" {
		t.Fatalf("取消后应交回服务端释放尝试：%v", cancels)
	}
}

// TestLoginBrowserOpenFailure 确认打不开系统浏览器时失败，并关闭环回监听。
func TestLoginBrowserOpenFailure(t *testing.T) {
	auth := callbackAuth("code-1", "state-1", "state-1")
	account := newTestAccount(t, Config{
		Status:       &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:    browserProviderSource(),
		Auth:         auth,
		Browser:      &stubBrowser{open: func(string) error { return errors.New("没有可用的浏览器") }},
		LoginTimeout: 2 * time.Second,
	})

	_, err := account.Login(context.Background(), LoginRequest{})
	if !errors.Is(err, ErrBrowserOpen) {
		t.Fatalf("应返回打开浏览器失败：%v", err)
	}
	if len(auth.completeList()) != 0 {
		t.Fatal("浏览器没打开时不应调用完成交换")
	}
	redirects := auth.redirectList()
	if len(redirects) != 1 {
		t.Fatalf("授权发起次数不符：%d", len(redirects))
	}
	parsed, parseErr := url.Parse(redirects[0])
	if parseErr != nil {
		t.Fatalf("回调地址无法解析: %v", parseErr)
	}
	connection, dialErr := net.DialTimeout("tcp", parsed.Host, 200*time.Millisecond)
	if dialErr == nil {
		_ = connection.Close()
		t.Fatal("登录失败后环回监听仍在")
	}
}

// TestLoginRequiresProviderDeclaration 确认没有 provider 声明时明确报不可用，不猜流程。
func TestLoginRequiresProviderDeclaration(t *testing.T) {
	account := newTestAccount(t, Config{Auth: callbackAuth("code-1", "state-1", "state-1")})
	if _, err := account.Login(context.Background(), LoginRequest{}); !errors.Is(err, ErrProvidersUnavailable) {
		t.Fatalf("应返回缺少 provider 声明：%v", err)
	}
}

// TestLoginRequiresSourceForDeclaredMethod 确认 provider 声明了方式但本端没有对应
// 能力时明确报不可用，不静默换一种方式。
func TestLoginRequiresSourceForDeclaredMethod(t *testing.T) {
	browserOnly := newTestAccount(t, Config{Providers: browserProviderSource()})
	if _, err := browserOnly.Login(context.Background(), LoginRequest{}); !errors.Is(err, ErrLoginMethodUnavailable) {
		t.Fatalf("browser 缺授权来源时应报不可用：%v", err)
	}

	deviceOnly := newTestAccount(t, Config{Providers: deviceProviderSource()})
	if _, err := deviceOnly.Login(context.Background(), LoginRequest{}); !errors.Is(err, ErrLoginMethodUnavailable) {
		t.Fatalf("device 缺设备码来源时应报不可用：%v", err)
	}
}

// TestLoginRejectsConcurrentAttempt 确认同一时刻只允许一次登录尝试。
func TestLoginRejectsConcurrentAttempt(t *testing.T) {
	began := make(chan struct{})
	release := make(chan struct{})
	auth := &stubAuth{}
	var once sync.Once
	auth.onBegin = func(redirectURI string) AuthorizationRequest {
		once.Do(func() { close(began) })
		<-release
		return AuthorizationRequest{URL: "http://127.0.0.1/unused", State: "state-1"}
	}
	account := newTestAccount(t, Config{
		Status:       &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:    browserProviderSource(),
		Auth:         auth,
		Browser:      &stubBrowser{},
		LoginTimeout: 5 * time.Second,
	})

	done := make(chan error, 1)
	go func() {
		_, err := account.Login(context.Background(), LoginRequest{})
		done <- err
	}()
	<-began

	if _, err := account.Login(context.Background(), LoginRequest{}); !errors.Is(err, ErrLoginInProgress) {
		t.Fatalf("并发的第二次登录应被拒绝：%v", err)
	}

	close(release)
	waitForLoginPhase(t, account, LoginWaiting)
	account.CancelLogin()
	select {
	case err := <-done:
		if !errors.Is(err, ErrLoginCanceled) {
			t.Fatalf("第一次登录应被取消：%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("取消后登录没有结束")
	}
	if count := auth.beginCount(); count != 1 {
		t.Fatalf("授权发起次数不符：%d", count)
	}
}

// TestLoginParsesStateFromAuthorizationURL 确认服务端只在授权地址里带 state 时，
// 本包能从地址读取并校验回调（与原客户端一致）。
func TestLoginParsesStateFromAuthorizationURL(t *testing.T) {
	status := &stubStatusReader{fact: signedInFact()}
	auth := &stubAuth{}
	auth.onBegin = func(redirectURI string) AuthorizationRequest {
		query := url.Values{"code": {"code-1"}, "state": {"state-1"}}
		return AuthorizationRequest{
			URL:     redirectURI + "?" + query.Encode(),
			Attempt: "attempt-1",
		}
	}
	account := newTestAccount(t, Config{
		Status:       status,
		Providers:    browserProviderSource(),
		Auth:         auth,
		Browser:      &httpBrowser{},
		LoginTimeout: 2 * time.Second,
	})
	snapshot, err := account.Login(context.Background(), LoginRequest{})
	if err != nil {
		t.Fatalf("登录: %v", err)
	}
	if snapshot.Status != StatusSignedIn {
		t.Fatalf("登录后的状态不符：%+v", snapshot)
	}
	if completions := auth.completeList(); len(completions) != 1 || completions[0].State != "state-1" {
		t.Fatalf("完成交换没有使用地址里的校验值：%+v", completions)
	}
}

// TestLoginRequiresAuthorizationState 确认浏览器登录缺少服务端校验值时拒绝，
// 与原客户端一致：没有 state 就无法校验回调。
func TestLoginRequiresAuthorizationState(t *testing.T) {
	auth := &stubAuth{}
	auth.onBegin = func(redirectURI string) AuthorizationRequest {
		return AuthorizationRequest{URL: redirectURI, Attempt: "attempt-1"}
	}
	browser := &stubBrowser{}
	account := newTestAccount(t, Config{
		Status:       &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:    browserProviderSource(),
		Auth:         auth,
		Browser:      browser,
		LoginTimeout: 2 * time.Second,
	})
	if _, err := account.Login(context.Background(), LoginRequest{}); !errors.Is(err, ErrLoginRequest) {
		t.Fatalf("缺少校验值应拒绝：%v", err)
	}
	if opened := browser.openedURLs(); len(opened) != 0 {
		t.Fatalf("缺少校验值时不应打开浏览器：%v", opened)
	}
	if cancels := auth.cancelList(); len(cancels) != 1 || cancels[0] != "attempt-1" {
		t.Fatalf("失败后应交回服务端释放尝试：%v", cancels)
	}
}

// TestLoginStatusUnreadAfterCompletion 确认授权完成但随后读取账号状态失败时明确分开
// 两类结果：授权成功、账号状态未知。
func TestLoginStatusUnreadAfterCompletion(t *testing.T) {
	status := &stubStatusReader{err: errors.New("连接中断")}
	account := newTestAccount(t, Config{
		Status:       status,
		Providers:    browserProviderSource(),
		Auth:         callbackAuth("code-1", "state-1", "state-1"),
		Browser:      &httpBrowser{},
		LoginTimeout: 2 * time.Second,
	})

	snapshot, err := account.Login(context.Background(), LoginRequest{})
	if !errors.Is(err, ErrLoginStatusUnread) {
		t.Fatalf("应返回登录后状态读取失败：%v", err)
	}
	if snapshot.Status != StatusUnknown {
		t.Fatalf("账号状态未确认时应是未知：%q", snapshot.Status)
	}
	if state := account.LoginState(); state.Phase != LoginCompleted {
		t.Fatalf("授权本身已完成，登录尝试状态不符：%+v", state)
	}
}
