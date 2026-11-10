package account

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// wireHarness 是「连接 + 写登记」集成形状的测试适配器：账号读取、授权发起与完成、
// 账号动作都走真实 NDJSON；wire 形状只出现在这里，本包不复制它们。
//
// overrideStatus 用于表达「登出之后服务端事实变了」：假服务端的脚本回执固定，不能
// 按请求次数变化，因此由测试适配器在动作之后给出新的状态。
type wireHarness struct {
	client *protocol.Client

	mu          sync.Mutex
	hasOverride bool
	override    Status
	attempts    map[string]string
	cancels     int
	deviceBegin []map[string]any
}

func (h *wireHarness) setStatus(status Status) {
	h.mu.Lock()
	h.hasOverride = true
	h.override = status
	h.mu.Unlock()
}

func (h *wireHarness) currentOverride() (Status, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.override, h.hasOverride
}

// ReadProviders 读取服务端声明的账号提供者。
func (h *wireHarness) ReadProviders(ctx context.Context) ([]Provider, error) {
	var reply []struct {
		ProviderID       string   `json:"providerId"`
		DisplayName      string   `json:"displayName"`
		LoginMethods     []string `json:"loginMethods"`
		DefaultDirectory *string  `json:"defaultDirectory"`
	}
	if err := h.client.CallInto(ctx, "account.providers", nil, &reply); err != nil {
		return nil, err
	}
	providers := make([]Provider, 0, len(reply))
	for _, entry := range reply {
		provider := Provider{
			ID:           entry.ProviderID,
			DisplayName:  entry.DisplayName,
			LoginMethods: make([]LoginMethod, 0, len(entry.LoginMethods)),
		}
		if entry.DefaultDirectory != nil {
			provider.Directory = *entry.DefaultDirectory
		}
		for _, method := range entry.LoginMethods {
			provider.LoginMethods = append(provider.LoginMethods, LoginMethod(method))
		}
		providers = append(providers, provider)
	}
	return providers, nil
}

func (h *wireHarness) rememberAttempt(attempt, providerID string) {
	if attempt == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.attempts == nil {
		h.attempts = make(map[string]string)
	}
	h.attempts[attempt] = providerID
}

func (h *wireHarness) attemptProvider(attempt string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.attempts[attempt]
}

// BeginProviderAuthorization 按 provider 向服务端登记环回回调地址。
func (h *wireHarness) BeginProviderAuthorization(ctx context.Context, begin ProviderAuthorizationBegin) (AuthorizationRequest, error) {
	var result struct {
		URL     string `json:"url"`
		State   string `json:"state"`
		Attempt string `json:"attemptId"`
	}
	params := map[string]any{
		"providerId":  begin.ProviderID,
		"redirectUri": begin.RedirectURI,
	}
	if begin.Directory != "" {
		params["directory"] = begin.Directory
	}
	if err := h.client.CallInto(ctx, "account.login.browser.begin", params, &result); err != nil {
		return AuthorizationRequest{}, err
	}
	h.rememberAttempt(result.Attempt, begin.ProviderID)
	query := url.Values{"code": {"login-code"}}
	if result.State != "" {
		query.Set("state", result.State)
	}
	return AuthorizationRequest{
		URL:     begin.RedirectURI + "?" + query.Encode(),
		State:   result.State,
		Attempt: result.Attempt,
	}, nil
}

// CompleteProviderAuthorization 按 provider 与尝试标识完成交换。
func (h *wireHarness) CompleteProviderAuthorization(
	ctx context.Context,
	providerID string,
	attempt string,
	result AuthorizationResult,
) error {
	_, err := h.client.Call(ctx, "account.login.browser.complete", map[string]any{
		"providerId":  providerID,
		"attemptId":   attempt,
		"callbackUrl": result.CallbackURL,
		"code":        result.Code,
		"state":       result.State,
	})
	return err
}

// BeginDeviceAuthorization 发起设备码登录。
func (h *wireHarness) BeginDeviceAuthorization(
	ctx context.Context,
	providerID string,
	directory string,
) (DeviceAuthorization, error) {
	params := map[string]any{"providerId": providerID}
	if directory != "" {
		params["directory"] = directory
	}
	h.mu.Lock()
	h.deviceBegin = append(h.deviceBegin, params)
	h.mu.Unlock()

	var reply struct {
		AttemptID               string `json:"attemptId"`
		VerificationURI         string `json:"verificationUri"`
		VerificationURIComplete string `json:"verificationUriComplete"`
		UserCode                string `json:"userCode"`
		ExpiresAt               int64  `json:"expiresAt"`
	}
	if err := h.client.CallInto(ctx, "account.login.begin", params, &reply); err != nil {
		return DeviceAuthorization{}, err
	}
	h.rememberAttempt(reply.AttemptID, providerID)
	device := DeviceAuthorization{
		Attempt:                 reply.AttemptID,
		VerificationURI:         reply.VerificationURI,
		VerificationURIComplete: reply.VerificationURIComplete,
		UserCode:                reply.UserCode,
	}
	if reply.ExpiresAt > 0 {
		device.ExpiresAt = time.UnixMilli(reply.ExpiresAt).UTC()
	}
	return device, nil
}

// PollDeviceAuthorization 轮询设备码登录状态。
func (h *wireHarness) PollDeviceAuthorization(ctx context.Context, attempt string) (DeviceLoginState, error) {
	providerID := h.attemptProvider(attempt)
	var reply struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := h.client.CallInto(ctx, "account.login.status", map[string]any{
		"providerId": providerID,
		"attemptId":  attempt,
	}, &reply); err != nil {
		return DeviceLoginState{}, err
	}
	return DeviceLoginState{Status: DeviceLoginStatus(reply.Status), Reason: reply.Error}, nil
}

// CancelAuthorization 取消服务端尝试；provider 由发起时记住。
func (h *wireHarness) CancelAuthorization(ctx context.Context, attempt string) error {
	providerID := h.attemptProvider(attempt)
	h.mu.Lock()
	h.cancels++
	h.mu.Unlock()
	_, err := h.client.Call(ctx, "account.login.cancel", map[string]any{
		"providerId": providerID,
		"attemptId":  attempt,
	})
	return err
}

// wireIdentity 是账号状态回执里的身份字段。
type wireIdentity struct {
	AccountID   string `json:"accountId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Plan        string `json:"plan"`
}

// ReadAccount 读取账号状态并映射成本领域的不可变事实。
func (h *wireHarness) ReadAccount(ctx context.Context) (Fact, error) {
	if status, ok := h.currentOverride(); ok {
		return Fact{Status: status, Identity: wireIdentityFact()}, nil
	}
	var result struct {
		Status   string       `json:"status"`
		Identity wireIdentity `json:"identity"`
	}
	if err := h.client.CallInto(ctx, "account.status", nil, &result); err != nil {
		return Fact{}, err
	}
	status, err := parseWireStatus(result.Status)
	if err != nil {
		return Fact{}, err
	}
	return Fact{
		Status: status,
		Identity: Identity{
			AccountID:   result.Identity.AccountID,
			Email:       result.Identity.Email,
			DisplayName: result.Identity.DisplayName,
			Plan:        result.Identity.Plan,
		},
	}, nil
}

func wireIdentityFact() Identity {
	return Identity{AccountID: "acct-wire", Email: "wire@example.com", DisplayName: "接口用户", Plan: "pro"}
}

func parseWireStatus(raw string) (Status, error) {
	switch Status(raw) {
	case StatusSignedIn, StatusSignedOut, StatusExpired:
		return Status(raw), nil
	default:
		return "", errors.New("服务端账号状态无法识别: " + raw)
	}
}

// ReadUsage 读取逐项用量；没有聚合项时保持不可用。
func (h *wireHarness) ReadUsage(ctx context.Context) (Usage, error) {
	var result struct {
		Lines []struct {
			Key      string  `json:"key"`
			Label    string  `json:"label"`
			Used     float64 `json:"used"`
			Limit    float64 `json:"limit"`
			Unit     string  `json:"unit"`
			Window   string  `json:"window"`
			ResetsAt int64   `json:"resetsAt"`
		} `json:"lines"`
		Aggregate *struct {
			Used  float64 `json:"used"`
			Limit float64 `json:"limit"`
			Unit  string  `json:"unit"`
		} `json:"aggregate"`
	}
	if err := h.client.CallInto(ctx, "account.usage", nil, &result); err != nil {
		return Usage{}, err
	}
	usage := Usage{Lines: make([]UsageLine, 0, len(result.Lines))}
	for _, line := range result.Lines {
		mapped := UsageLine{
			Key:    line.Key,
			Label:  line.Label,
			Used:   line.Used,
			Limit:  line.Limit,
			Unit:   line.Unit,
			Window: line.Window,
		}
		if line.ResetsAt > 0 {
			mapped.ResetsAt = time.UnixMilli(line.ResetsAt).UTC()
		}
		usage.Lines = append(usage.Lines, mapped)
	}
	if result.Aggregate != nil {
		usage.Aggregate = AggregateUsage{
			Available: true,
			Used:      result.Aggregate.Used,
			Limit:     result.Aggregate.Limit,
			Unit:      result.Aggregate.Unit,
		}
	}
	return usage, nil
}

// ReadModels 读取服务端模型清单。
func (h *wireHarness) ReadModels(ctx context.Context) ([]Model, error) {
	var result struct {
		Models []struct {
			ID              string `json:"id"`
			DisplayName     string `json:"displayName"`
			RequiresAccount bool   `json:"requiresAccount"`
			Unavailable     bool   `json:"unavailable"`
		} `json:"models"`
	}
	if err := h.client.CallInto(ctx, "model.list", nil, &result); err != nil {
		return nil, err
	}
	models := make([]Model, 0, len(result.Models))
	for _, model := range result.Models {
		models = append(models, Model{
			ID:              model.ID,
			DisplayName:     model.DisplayName,
			RequiresAccount: model.RequiresAccount,
			Unavailable:     model.Unavailable,
		})
	}
	return models, nil
}

// Send 是写登记的发送方：账号动作按透明参数里的操作名映射到服务端方法，只发一次。
func (h *wireHarness) Send(ctx context.Context, req write.Request) (write.Reply, error) {
	if req.Kind != write.KindAccountAction {
		return write.Reply{}, errors.New("未支持的写意图: " + string(req.Kind))
	}
	var payload struct {
		Op string `json:"op"`
	}
	if err := json.Unmarshal(req.Params, &payload); err != nil {
		return write.Reply{}, err
	}
	raw, err := h.client.Call(ctx, "account."+payload.Op, nil)
	if err != nil {
		return write.Reply{}, err
	}
	return write.Reply{Result: raw}, nil
}

// unconfirmedInspector 表示本测试不做未知核对：未知动作最终进入需人工确认。
type unconfirmedInspector struct{}

func (unconfirmedInspector) LookupInput(context.Context, write.InputQuery) (write.InputState, error) {
	return write.InputState{}, nil
}

// startAccountServer 启动假服务端，并建好连接与写登记。
func startAccountServer(t *testing.T, script fakeappserver.Script) (*fakeappserver.Server, *protocol.Client, *write.Manager) {
	t.Helper()
	server, err := fakeappserver.Start(script)
	if err != nil {
		t.Fatalf("启动假服务端: %v", err)
	}
	supervisor, err := daemon.New(daemon.Config{
		DataDir:        server.DataDir(),
		RequestTimeout: 2 * time.Second,
	})
	if err != nil {
		_ = server.Close()
		t.Fatalf("创建连接监督者: %v", err)
	}
	connection, err := supervisor.Connect(context.Background())
	if err != nil {
		supervisor.Close(nil)
		_ = server.Close()
		t.Fatalf("连接假服务端: %v", err)
	}
	manager, err := write.New(write.Config{
		Sender:      &wireHarness{client: connection.Client},
		Inspector:   unconfirmedInspector{},
		SendTimeout: 300 * time.Millisecond,
		Verify: write.VerifyPolicy{
			MaxAttempts: 1,
			Interval:    time.Millisecond,
			Timeout:     500 * time.Millisecond,
		},
	})
	if err != nil {
		supervisor.Close(nil)
		_ = server.Close()
		t.Fatalf("创建写登记: %v", err)
	}
	t.Cleanup(func() {
		manager.Close()
		supervisor.Close(nil)
		_ = server.Close()
	})
	return server, connection.Client, manager
}

// TestAppServerReadsStatusUsageAndModels 用假服务端覆盖范围 1、4、5：账号状态、
// 逐项用量（聚合不可用）与按账号过滤的模型清单。
func TestAppServerReadsStatusUsageAndModels(t *testing.T) {
	server, client, _ := startAccountServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "account.status", Result: map[string]any{
				"status": "signed_in",
				"identity": map[string]any{
					"accountId":   "acct-wire",
					"email":       "wire@example.com",
					"displayName": "接口用户",
					"plan":        "pro",
				},
			}},
			{Method: "account.usage", Result: map[string]any{
				"lines": []any{
					map[string]any{
						"key":      "tokens",
						"label":    "令牌",
						"used":     120,
						"limit":    1000,
						"unit":     "tokens",
						"window":   "day",
						"resetsAt": int64(1764547200000),
					},
					map[string]any{
						"key":    "requests",
						"label":  "请求",
						"used":   3,
						"limit":  60,
						"unit":   "requests",
						"window": "minute",
					},
				},
			}},
			{Method: "model.list", Result: map[string]any{
				"models": []any{
					map[string]any{"id": "m-1", "displayName": "一号"},
					map[string]any{"id": "m-2", "displayName": "二号", "unavailable": true},
					map[string]any{"id": "m-3", "displayName": "三号", "requiresAccount": true},
				},
			}},
		},
	})

	harness := &wireHarness{client: client}
	account := newTestAccount(t, Config{Status: harness, Usage: harness, Models: harness})

	snapshot, err := account.Refresh(context.Background())
	if err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}
	if snapshot.Status != StatusSignedIn || snapshot.Identity.Email != "wire@example.com" || snapshot.Identity.Plan != "pro" {
		t.Fatalf("账号状态不符：%+v", snapshot)
	}

	usage, err := account.RefreshUsage(context.Background())
	if err != nil {
		t.Fatalf("读取用量: %v", err)
	}
	if len(usage.Lines) != 2 || usage.Lines[0].Key != "tokens" || usage.Lines[0].Used != 120 ||
		usage.Lines[1].Key != "requests" || usage.Lines[1].Limit != 60 {
		t.Fatalf("逐项用量不符：%+v", usage.Lines)
	}
	if !usage.Lines[0].ResetsAt.Equal(time.UnixMilli(1764547200000).UTC()) {
		t.Fatalf("重置时间不符：%v", usage.Lines[0].ResetsAt)
	}
	if usage.Aggregate.Available || usage.Aggregate.Reason == "" {
		t.Fatalf("聚合用量不可用时应说明原因：%+v", usage.Aggregate)
	}

	list, err := account.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型清单: %v", err)
	}
	if len(list.Models) != 2 || list.Models[0].ID != "m-1" || list.Models[1].ID != "m-3" {
		t.Fatalf("已登录时的模型清单不符：%+v", list.Models)
	}

	for _, method := range []string{"account.status", "account.usage", "model.list"} {
		if count := len(server.RequestsFor(method)); count != 1 {
			t.Fatalf("%s 请求次数不符：%d", method, count)
		}
	}
}

// TestAppServerLoginThroughLoopback 用假服务端走一次完整登录：索要授权、打开系统
// 浏览器、环回回调、完成交换、重新读取账号状态。
func TestAppServerLoginThroughLoopback(t *testing.T) {
	server, client, _ := startAccountServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "account.providers", Result: []any{
				map[string]any{
					"providerId":       "dimcode-api-oauth",
					"displayName":      "Dim OAuth",
					"loginMethods":     []any{"browser", "device"},
					"defaultDirectory": "https://dimagent.cn",
				},
			}},
			{Method: "account.login.browser.begin", Result: map[string]any{
				"url":       "https://provider.example/authorize",
				"state":     "wire-state",
				"attemptId": "attempt-wire",
			}},
			{Method: "account.login.browser.complete", Result: map[string]any{"ok": true}},
			{Method: "account.status", Result: map[string]any{
				"status": "signed_in",
				"identity": map[string]any{
					"accountId": "acct-wire",
					"email":     "wire@example.com",
				},
			}},
		},
	})

	harness := &wireHarness{client: client}
	browser := &httpBrowser{}
	account := newTestAccount(t, Config{
		Status:       harness,
		Providers:    harness,
		Auth:         harness,
		Browser:      browser,
		LoginTimeout: 2 * time.Second,
	})

	snapshot, err := account.Login(context.Background(), LoginRequest{})
	if err != nil {
		t.Fatalf("登录: %v", err)
	}
	if snapshot.Status != StatusSignedIn || snapshot.Identity.AccountID != "acct-wire" {
		t.Fatalf("登录后的账号状态不符：%+v", snapshot)
	}
	if state := account.LoginState(); state.Phase != LoginCompleted {
		t.Fatalf("登录尝试状态不符：%+v", state)
	}

	beginRequests := server.RequestsFor("account.login.browser.begin")
	if len(beginRequests) != 1 {
		t.Fatalf("授权发起次数不符：%d", len(beginRequests))
	}
	var beginParams map[string]any
	if err := json.Unmarshal(beginRequests[0].Params, &beginParams); err != nil {
		t.Fatalf("授权发起参数无法解析: %v", err)
	}
	redirectURI, _ := beginParams["redirectUri"].(string)
	parsed, err := url.Parse(redirectURI)
	if err != nil || parsed.Hostname() != "127.0.0.1" || parsed.Path != callbackPath {
		t.Fatalf("回调地址不符：%q", redirectURI)
	}
	if beginParams["providerId"] != "dimcode-api-oauth" || beginParams["directory"] != "https://dimagent.cn" {
		t.Fatalf("授权发起没有带上 provider 与目录：%+v", beginParams)
	}

	completeRequests := server.RequestsFor("account.login.browser.complete")
	if len(completeRequests) != 1 {
		t.Fatalf("完成交换次数不符：%d", len(completeRequests))
	}
	var completeParams map[string]any
	if err := json.Unmarshal(completeRequests[0].Params, &completeParams); err != nil {
		t.Fatalf("完成交换参数无法解析: %v", err)
	}
	if completeParams["code"] != "login-code" || completeParams["state"] != "wire-state" {
		t.Fatalf("完成交换参数不符：%+v", completeParams)
	}
	if completeParams["providerId"] != "dimcode-api-oauth" || completeParams["attemptId"] != "attempt-wire" {
		t.Fatalf("完成交换没有带上服务端尝试标识：%+v", completeParams)
	}
	callbackURL, _ := completeParams["callbackUrl"].(string)
	if !strings.HasPrefix(callbackURL, redirectURI) || !strings.Contains(callbackURL, "code=login-code") {
		t.Fatalf("完成交换没有带上完整回调地址：%q", callbackURL)
	}

	if count := len(server.RequestsFor("account.status")); count != 1 {
		t.Fatalf("登录后应重新读取一次账号状态：%d", count)
	}
	opened, bodies := browser.observed()
	if len(opened) != 1 || !strings.HasPrefix(opened[0], redirectURI) {
		t.Fatalf("没有打开授权地址：%v", opened)
	}
	if len(bodies) != 1 || strings.Contains(bodies[0], "login-code") {
		t.Fatalf("回调应答不应回显授权码：%v", bodies)
	}
}

// TestAppServerDeviceLoginThroughProvider 用假服务端核对：provider 只声明 device 时
// 走设备码流程，发起用 providerId/directory，轮询到完成后重新读取账号状态。
func TestAppServerDeviceLoginThroughProvider(t *testing.T) {
	server, client, _ := startAccountServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "account.providers", Result: []any{
				map[string]any{
					"providerId":   "provider-device",
					"displayName":  "设备码账号",
					"loginMethods": []any{"device"},
				},
			}},
			{Method: "account.login.begin", Result: map[string]any{
				"attemptId":               "device-attempt",
				"verificationUri":         "https://issuer.example/device",
				"verificationUriComplete": "https://issuer.example/device?code=ABCD",
				"userCode":                "ABCD-EFGH",
			}},
			{Method: "account.login.status", Result: map[string]any{"status": "completed"}},
			{Method: "account.status", Result: map[string]any{
				"status": "signed_in",
				"identity": map[string]any{
					"accountId": "acct-wire",
					"email":     "wire@example.com",
				},
			}},
		},
	})

	harness := &wireHarness{client: client}
	account := newTestAccount(t, Config{
		Status:       harness,
		Providers:    harness,
		Auth:         harness,
		DeviceAuth:   harness,
		Browser:      &stubBrowser{},
		LoginTimeout: 2 * time.Second,
	})

	start, err := account.BeginLogin(context.Background(), LoginRequest{Method: LoginMethodDevice})
	if err != nil {
		t.Fatalf("发起设备码登录: %v", err)
	}
	if start.ProviderID != "provider-device" || start.UserCode != "ABCD-EFGH" || start.VerificationURI == "" {
		t.Fatalf("设备码提示不符：%+v", start)
	}
	snapshot, err := account.WaitLogin(context.Background())
	if err != nil {
		t.Fatalf("等待设备码登录: %v", err)
	}
	if snapshot.Status != StatusSignedIn || snapshot.Identity.AccountID != "acct-wire" {
		t.Fatalf("设备码登录后的账号状态不符：%+v", snapshot)
	}
	if state := account.LoginState(); state.Method != LoginMethodDevice || state.Phase != LoginCompleted {
		t.Fatalf("登录尝试状态不符：%+v", state)
	}

	beginRequests := server.RequestsFor("account.login.begin")
	if len(beginRequests) != 1 {
		t.Fatalf("设备码发起次数不符：%d", len(beginRequests))
	}
	var beginParams map[string]any
	if err := json.Unmarshal(beginRequests[0].Params, &beginParams); err != nil {
		t.Fatalf("设备码发起参数无法解析: %v", err)
	}
	if beginParams["providerId"] != "provider-device" {
		t.Fatalf("设备码发起没有带上 provider：%+v", beginParams)
	}
	statusRequests := server.RequestsFor("account.login.status")
	if len(statusRequests) != 1 {
		t.Fatalf("设备码轮询次数不符：%d", len(statusRequests))
	}
	var statusParams map[string]any
	if err := json.Unmarshal(statusRequests[0].Params, &statusParams); err != nil {
		t.Fatalf("设备码轮询参数无法解析: %v", err)
	}
	if statusParams["providerId"] != "provider-device" || statusParams["attemptId"] != "device-attempt" {
		t.Fatalf("设备码轮询参数不符：%+v", statusParams)
	}
	if count := len(server.RequestsFor("account.login.browser.begin")); count != 0 {
		t.Fatalf("设备码流程不应发起浏览器登录：%d", count)
	}
}

// TestAppServerLogoutThroughWriteManager 用假服务端核对：登出经写登记只发一次，
// 成功后本地账号状态清空，再次登出不重发。
func TestAppServerLogoutThroughWriteManager(t *testing.T) {
	server, client, manager := startAccountServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "account.status", Result: map[string]any{
				"status": "signed_in",
				"identity": map[string]any{
					"accountId": "acct-wire",
					"email":     "wire@example.com",
				},
			}},
			{Method: "account.usage", Result: map[string]any{
				"lines": []any{
					map[string]any{"key": "tokens", "used": 10, "limit": 100},
				},
			}},
			{Method: "model.list", Result: map[string]any{
				"models": []any{
					map[string]any{"id": "m-1"},
					map[string]any{"id": "m-2", "requiresAccount": true},
				},
			}},
			{Method: "account.logout", Result: map[string]any{"ok": true}},
		},
	})

	harness := &wireHarness{client: client}
	account := newTestAccount(t, Config{
		Status:  harness,
		Usage:   harness,
		Models:  harness,
		Actions: manager,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}
	if _, err := account.RefreshUsage(context.Background()); err != nil {
		t.Fatalf("读取用量: %v", err)
	}
	if list, err := account.RefreshModels(context.Background()); err != nil || len(list.Models) != 2 {
		t.Fatalf("读取模型清单: %+v，%v", list, err)
	}

	view, err := account.Logout(context.Background())
	if err != nil {
		t.Fatalf("登出: %v", err)
	}
	if view.Status != write.StatusSucceeded {
		t.Fatalf("登出结果不符：%+v", view)
	}
	if count := len(server.RequestsFor("account.logout")); count != 1 {
		t.Fatalf("登出请求次数不符：%d", count)
	}
	table := manager.Table()
	if len(table) != 1 || table[0].Kind != write.KindAccountAction || table[0].Status != write.StatusSucceeded {
		t.Fatalf("登出没有经过写登记：%+v", table)
	}

	snapshot := account.Snapshot()
	if snapshot.Status != StatusUnknown || snapshot.Identity != (Identity{}) {
		t.Fatalf("登出成功后本地状态应清空：%+v", snapshot)
	}
	if snapshot.Usage.Ready || len(snapshot.Usage.Lines) != 0 {
		t.Fatalf("登出成功后用量应清空：%+v", snapshot.Usage)
	}

	// 再次登出：不重复发请求。
	if _, err := account.Logout(context.Background()); err != nil {
		t.Fatalf("再次登出: %v", err)
	}
	if count := len(server.RequestsFor("account.logout")); count != 1 {
		t.Fatalf("已成功的登出被重复发送：%d", count)
	}

	// 重新读取后得到服务端确认的未登录状态。
	harness.setStatus(StatusSignedOut)
	snapshot, err = account.Refresh(context.Background())
	if err != nil {
		t.Fatalf("重新读取账号状态: %v", err)
	}
	if snapshot.Status != StatusSignedOut || snapshot.Identity != (Identity{}) {
		t.Fatalf("重新读取后的未登录状态不符：%+v", snapshot)
	}
	if models := account.Models(); len(models.Models) != 0 || models.Reason == "" {
		t.Fatalf("未登录时模型清单应为空并说明原因：%+v", models)
	}
}

// TestAppServerResetUnknownIsNotResent 用假服务端核对：重置没有回执时进入需人工
// 确认，不自动重发；再次显式重置与「重启」都不产生新请求。
func TestAppServerResetUnknownIsNotResent(t *testing.T) {
	server, client, manager := startAccountServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "account.reset", Result: map[string]any{"ok": true}},
			{Method: "account.status", Result: map[string]any{
				"status": "signed_in",
				"identity": map[string]any{
					"accountId": "acct-wire",
				},
			}},
		},
		Faults: []fakeappserver.Fault{
			{Method: "account.reset", NoReply: true},
		},
	})

	harness := &wireHarness{client: client}
	account := newTestAccount(t, Config{Status: harness, Usage: harness, Actions: manager})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}

	view, err := account.Reset(context.Background())
	if err != nil {
		t.Fatalf("重置: %v", err)
	}
	if view.Status != write.StatusNeedsManualConfirmation {
		t.Fatalf("无回执的重置应进入需人工确认：%+v", view)
	}
	if count := len(server.RequestsFor("account.reset")); count != 1 {
		t.Fatalf("重置请求次数不符：%d", count)
	}

	second, err := account.Reset(context.Background())
	if err != nil {
		t.Fatalf("再次重置: %v", err)
	}
	if second.ID != view.ID {
		t.Fatalf("结果未定时应等待同一条意图：got %q，want %q", second.ID, view.ID)
	}
	if count := len(server.RequestsFor("account.reset")); count != 1 {
		t.Fatalf("结果未定的重置被重复提交：%d", count)
	}

	// 模拟重启：新的账号实例与写登记不持有旧意图，也不补发。
	restarted := newTestAccount(t, Config{Status: harness, Usage: harness, Actions: manager})
	if restarted.Status() != StatusUnknown {
		t.Fatalf("新实例应从未知开始：%q", restarted.Status())
	}
	if count := len(server.RequestsFor("account.reset")); count != 1 {
		t.Fatalf("重启后不应补发重置：%d", count)
	}
}
