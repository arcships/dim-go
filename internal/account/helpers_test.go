package account

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// stubStatusReader 是账号状态读取替身，记录调用次数。
type stubStatusReader struct {
	mu    sync.Mutex
	fact  Fact
	err   error
	calls int
}

func (r *stubStatusReader) ReadAccount(ctx context.Context) (Fact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return r.fact, r.err
}

func (r *stubStatusReader) set(fact Fact, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fact = fact
	r.err = err
}

func (r *stubStatusReader) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// stubUsageReader 是用量读取替身。
type stubUsageReader struct {
	mu    sync.Mutex
	usage Usage
	err   error
	calls int
}

func (r *stubUsageReader) ReadUsage(ctx context.Context) (Usage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return r.usage, r.err
}

func (r *stubUsageReader) set(usage Usage, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usage = usage
	r.err = err
}

// stubModelReader 是模型清单读取替身。
type stubModelReader struct {
	mu     sync.Mutex
	models []Model
	err    error
	calls  int
}

func (r *stubModelReader) ReadModels(ctx context.Context) ([]Model, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return append([]Model(nil), r.models...), r.err
}

func (r *stubModelReader) set(models []Model, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models = models
	r.err = err
}

// stubWriter 是写登记替身：同一幂等键不重复登记；Register 返回在途视图，
// Wait 返回预先定好的终态。所有登记按顺序记录，供核对「只发一次」。
type stubWriter struct {
	mu      sync.Mutex
	intents []write.Intent
	views   map[string]write.IntentView
	waits   map[string]error
	nextID  int
	outcome func(intent write.Intent, id string) (write.Status, error)
}

func (w *stubWriter) Register(intent write.Intent) (write.IntentView, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.views == nil {
		w.views = make(map[string]write.IntentView)
		w.waits = make(map[string]error)
	}
	for _, view := range w.views {
		if view.Key == intent.Key {
			return view, nil
		}
	}
	w.nextID++
	id := fmt.Sprintf("intent-%d", w.nextID)
	w.intents = append(w.intents, intent)
	status := write.StatusSucceeded
	var waitErr error
	if w.outcome != nil {
		status, waitErr = w.outcome(intent, id)
	}
	w.views[id] = write.IntentView{
		ID:      id,
		Key:     intent.Key,
		Kind:    intent.Kind,
		Status:  status,
		Content: intent.Params,
	}
	w.waits[id] = waitErr
	return write.IntentView{
		ID:      id,
		Key:     intent.Key,
		Kind:    intent.Kind,
		Status:  write.StatusInFlight,
		Content: intent.Params,
	}, nil
}

func (w *stubWriter) Wait(ctx context.Context, intentID string) (write.IntentView, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.views[intentID], w.waits[intentID]
}

func (w *stubWriter) setOutcome(outcome func(intent write.Intent, id string) (write.Status, error)) {
	w.mu.Lock()
	w.outcome = outcome
	w.mu.Unlock()
}

func (w *stubWriter) registered() []write.Intent {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]write.Intent(nil), w.intents...)
}

func (w *stubWriter) intentCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.intents)
}

// stubAuth 是浏览器登录服务端侧的替身：记录回调地址与交换结果。
type stubAuth struct {
	mu          sync.Mutex
	onBegin     func(redirectURI string) AuthorizationRequest
	beginErr    error
	completeErr error
	cancelErr   error
	onComplete  func(AuthorizationResult)
	redirects   []string
	completes   []AuthorizationResult
	attempts    []string
	cancels     []string
	begins      int
}

func (a *stubAuth) BeginAuthorization(ctx context.Context, redirectURI string) (AuthorizationRequest, error) {
	a.mu.Lock()
	a.begins++
	a.redirects = append(a.redirects, redirectURI)
	beginErr := a.beginErr
	onBegin := a.onBegin
	a.mu.Unlock()
	if beginErr != nil {
		return AuthorizationRequest{}, beginErr
	}
	if onBegin != nil {
		return onBegin(redirectURI), nil
	}
	return AuthorizationRequest{}, nil
}

func (a *stubAuth) CompleteAuthorization(ctx context.Context, attempt string, result AuthorizationResult) error {
	a.mu.Lock()
	a.completes = append(a.completes, result)
	a.attempts = append(a.attempts, attempt)
	completeErr := a.completeErr
	onComplete := a.onComplete
	a.mu.Unlock()
	if onComplete != nil {
		onComplete(result)
	}
	return completeErr
}

// CancelAuthorization 实现可选的服务端尝试取消；记录交回的尝试标识。
func (a *stubAuth) CancelAuthorization(ctx context.Context, attempt string) error {
	a.mu.Lock()
	a.cancels = append(a.cancels, attempt)
	cancelErr := a.cancelErr
	a.mu.Unlock()
	return cancelErr
}

func (a *stubAuth) cancelList() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.cancels...)
}

func (a *stubAuth) attemptList() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.attempts...)
}

func (a *stubAuth) completeList() []AuthorizationResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]AuthorizationResult(nil), a.completes...)
}

func (a *stubAuth) beginCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.begins
}

func (a *stubAuth) redirectList() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.redirects...)
}

// callbackAuth 造一个授权替身：授权地址就是环回回调，带指定的授权码与校验值，
// 模拟授权服务端重定向回客户端。
func callbackAuth(code, state, callbackState string) *stubAuth {
	auth := &stubAuth{}
	auth.onBegin = func(redirectURI string) AuthorizationRequest {
		query := url.Values{}
		if code != "" {
			query.Set("code", code)
		}
		if callbackState != "" {
			query.Set("state", callbackState)
		}
		return AuthorizationRequest{
			URL:     redirectURI + "?" + query.Encode(),
			State:   state,
			Attempt: "attempt-1",
		}
	}
	return auth
}

// stubBrowser 是系统浏览器替身：记录被打开的地址，按 open 处理。
type stubBrowser struct {
	mu     sync.Mutex
	opened []string
	open   func(url string) error
}

func (b *stubBrowser) Open(rawURL string) error {
	b.mu.Lock()
	b.opened = append(b.opened, rawURL)
	open := b.open
	b.mu.Unlock()
	if open != nil {
		return open(rawURL)
	}
	return nil
}

func (b *stubBrowser) openedURLs() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.opened...)
}

// newTestAccount 建一个账号实例；Status 必填，其他按调用方给。
func newTestAccount(t *testing.T, cfg Config) *Account {
	t.Helper()
	if cfg.Status == nil {
		cfg.Status = &stubStatusReader{}
	}
	account, err := New(cfg)
	if err != nil {
		t.Fatalf("创建账号: %v", err)
	}
	return account
}

// signedInFact 造一个已登录事实。
func signedInFact() Fact {
	return Fact{
		Status: StatusSignedIn,
		Identity: Identity{
			AccountID:   "acct-1",
			Email:       "user@example.com",
			DisplayName: "测试用户",
			Plan:        "pro",
		},
	}
}

// mustStatus 断言账号状态。
func mustStatus(t *testing.T, account *Account, want Status) {
	t.Helper()
	if got := account.Status(); got != want {
		t.Fatalf("账号状态不符：got %q，want %q", got, want)
	}
}

// mustIntentsEqual 比较登记顺序里的幂等键。
func mustIntentKeys(t *testing.T, writer *stubWriter, want ...string) {
	t.Helper()
	intents := writer.registered()
	keys := make([]string, 0, len(intents))
	for _, intent := range intents {
		keys = append(keys, intent.Key)
	}
	if len(keys) != len(want) {
		t.Fatalf("登记次数不符：got %v，want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("第 %d 次登记的幂等键不符：got %q，want %q（整体 %v）", i, keys[i], want[i], keys)
		}
	}
}

// waitForLoginPhase 轮询等待登录尝试进入指定状态；用于取消与并发测试。
func waitForLoginPhase(t *testing.T, account *Account, want LoginPhase) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if account.LoginState().Phase == want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("等待登录状态 %q 超时，当前 %q", want, account.LoginState().Phase)
}

// intentParams 解析写意图的透明参数。
func intentParams(t *testing.T, intent write.Intent) map[string]string {
	t.Helper()
	var params map[string]string
	if err := json.Unmarshal(intent.Params, &params); err != nil {
		t.Fatalf("写意图参数无法解析: %v", err)
	}
	return params
}
