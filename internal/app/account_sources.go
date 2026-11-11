package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
)

// 设备码登录的真实方法（D6 §6.3）。
const (
	methodDeviceLoginBegin  = "account.login.begin"
	methodDeviceLoginStatus = "account.login.status"
)

// accountProviderSource 实现 account.ProviderSource：`account.providers` 的只读映射。
type accountProviderSource struct {
	wire *accountWire
}

func (s *accountProviderSource) ReadProviders(ctx context.Context) ([]account.Provider, error) {
	client, err := s.wire.app.requireControl("读取账号提供者")
	if err != nil {
		return nil, err
	}
	var providers []struct {
		ProviderID       string   `json:"providerId"`
		DisplayName      string   `json:"displayName"`
		LoginMethods     []string `json:"loginMethods"`
		DefaultDirectory *string  `json:"defaultDirectory"`
	}
	if err := client.CallInto(ctx, methodAccountProviders, nil, &providers); err != nil {
		return nil, err
	}
	result := make([]account.Provider, 0, len(providers))
	for _, provider := range providers {
		if provider.ProviderID == "" {
			continue
		}
		entry := account.Provider{
			ID:          provider.ProviderID,
			DisplayName: provider.DisplayName,
		}
		for _, method := range provider.LoginMethods {
			entry.LoginMethods = append(entry.LoginMethods, account.LoginMethod(method))
		}
		if provider.DefaultDirectory != nil {
			entry.Directory = *provider.DefaultDirectory
		}
		result = append(result, entry)
	}
	return result, nil
}

// BeginProviderAuthorization 按 provider 发起浏览器登录（`account.login.browser.begin`）。
func (s *accountAuthSource) BeginProviderAuthorization(
	ctx context.Context,
	begin account.ProviderAuthorizationBegin,
) (account.AuthorizationRequest, error) {
	providerID := begin.ProviderID
	directory := begin.Directory
	if providerID == "" || directory == "" {
		resolved, err := s.wire.accountProvider(ctx, "发起账号登录")
		if err != nil {
			return account.AuthorizationRequest{}, err
		}
		if providerID == "" {
			providerID = resolved.ID
		}
		if directory == "" {
			directory = resolved.Directory
		}
	}
	client, err := s.wire.app.requireControl("发起账号登录")
	if err != nil {
		return account.AuthorizationRequest{}, err
	}
	params := map[string]any{
		"providerId":  providerID,
		"redirectUri": begin.RedirectURI,
	}
	if directory != "" {
		params["directory"] = directory
	}
	if email := strings.TrimSpace(begin.Email); email != "" {
		// login_hint：原客户端同样透传，只在发起时带上。
		params["email"] = email
	}
	var reply struct {
		AttemptID        string `json:"attemptId"`
		AuthorizationURL string `json:"authorizationUrl"`
		State            string `json:"state"`
	}
	if err := client.CallInto(ctx, methodLoginBegin, params, &reply); err != nil {
		return account.AuthorizationRequest{}, err
	}
	s.wire.rememberAttempt(reply.AttemptID, providerID)
	return account.AuthorizationRequest{
		URL:     reply.AuthorizationURL,
		State:   reply.State,
		Attempt: reply.AttemptID,
	}, nil
}

// CompleteProviderAuthorization 按 provider 与尝试标识完成浏览器登录交换。
func (s *accountAuthSource) CompleteProviderAuthorization(
	ctx context.Context,
	providerID string,
	attempt string,
	result account.AuthorizationResult,
) error {
	if providerID == "" {
		providerID = s.wire.attemptProvider(attempt)
	}
	if providerID == "" {
		resolved, err := s.wire.accountProvider(ctx, "完成账号登录")
		if err != nil {
			return err
		}
		providerID = resolved.ID
	}
	client, err := s.wire.app.requireControl("完成账号登录")
	if err != nil {
		return err
	}
	_, err = client.Call(ctx, methodLoginComplete, map[string]any{
		"providerId":  providerID,
		"attemptId":   attempt,
		"callbackUrl": result.CallbackURL,
	})
	s.wire.forgetAttempt(attempt)
	return err
}

// accountDeviceSource 实现 account.DeviceAuthorizationSource：设备码发起与轮询。
type accountDeviceSource struct {
	wire *accountWire

	mu       sync.Mutex
	attempts map[string]string
}

// BeginDeviceAuthorization 调 `account.login.begin`，映射设备码信息。
func (s *accountDeviceSource) BeginDeviceAuthorization(
	ctx context.Context,
	providerID string,
	directory string,
) (account.DeviceAuthorization, error) {
	return s.beginDeviceAuthorization(ctx, providerID, directory, "")
}

// BeginDeviceAuthorizationWithEmail 是可选能力：设备码发起时带登录提示。
func (s *accountDeviceSource) BeginDeviceAuthorizationWithEmail(
	ctx context.Context,
	providerID string,
	directory string,
	email string,
) (account.DeviceAuthorization, error) {
	return s.beginDeviceAuthorization(ctx, providerID, directory, email)
}

func (s *accountDeviceSource) beginDeviceAuthorization(
	ctx context.Context,
	providerID string,
	directory string,
	email string,
) (account.DeviceAuthorization, error) {
	if providerID == "" || directory == "" {
		resolved, err := s.wire.accountProvider(ctx, "发起设备码登录")
		if err != nil {
			return account.DeviceAuthorization{}, err
		}
		if providerID == "" {
			providerID = resolved.ID
		}
		if directory == "" {
			directory = resolved.Directory
		}
	}
	client, err := s.wire.app.requireControl("发起设备码登录")
	if err != nil {
		return account.DeviceAuthorization{}, err
	}
	params := map[string]any{"providerId": providerID}
	if directory != "" {
		params["directory"] = directory
	}
	if strings.TrimSpace(email) != "" {
		params["email"] = strings.TrimSpace(email)
	}
	var reply struct {
		AttemptID               string  `json:"attemptId"`
		VerificationURI         string  `json:"verificationUri"`
		VerificationURIComplete string  `json:"verificationUriComplete"`
		UserCode                string  `json:"userCode"`
		ExpiresAt               int64   `json:"expiresAt"`
		Interval                float64 `json:"interval"`
	}
	if err := client.CallInto(ctx, methodDeviceLoginBegin, params, &reply); err != nil {
		return account.DeviceAuthorization{}, err
	}
	if reply.AttemptID == "" {
		return account.DeviceAuthorization{}, failure.New(failure.KindNonRetryable, methodDeviceLoginBegin,
			fmt.Errorf("设备码发起回执缺少尝试标识"))
	}
	s.mu.Lock()
	if s.attempts == nil {
		s.attempts = make(map[string]string)
	}
	s.attempts[reply.AttemptID] = providerID
	s.mu.Unlock()
	s.wire.rememberAttempt(reply.AttemptID, providerID)

	interval := 5 * time.Second
	if reply.Interval > 0 {
		interval = time.Duration(reply.Interval * float64(time.Second))
	}
	return account.DeviceAuthorization{
		Attempt:                 reply.AttemptID,
		VerificationURI:         reply.VerificationURI,
		VerificationURIComplete: reply.VerificationURIComplete,
		UserCode:                reply.UserCode,
		ExpiresAt:               wireUnixTime(reply.ExpiresAt),
		Interval:                interval,
	}, nil
}

// PollDeviceAuthorization 调 `account.login.status`，映射设备码尝试状态。
func (s *accountDeviceSource) PollDeviceAuthorization(
	ctx context.Context,
	attempt string,
) (account.DeviceLoginState, error) {
	providerID := s.wire.attemptProvider(attempt)
	if providerID == "" {
		resolved, err := s.wire.accountProvider(ctx, "轮询设备码登录")
		if err != nil {
			return account.DeviceLoginState{}, err
		}
		providerID = resolved.ID
	}
	client, err := s.wire.app.requireControl("轮询设备码登录")
	if err != nil {
		return account.DeviceLoginState{}, err
	}
	var reply struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := client.CallInto(ctx, methodDeviceLoginStatus, map[string]any{
		"providerId": providerID,
		"attemptId":  attempt,
	}, &reply); err != nil {
		return account.DeviceLoginState{}, err
	}
	status, err := deviceLoginStatus(reply.Status)
	if err != nil {
		return account.DeviceLoginState{}, err
	}
	return account.DeviceLoginState{Status: status, Reason: reply.Error}, nil
}

// CancelAuthorization 取消设备码尝试：与浏览器登录共用 account.login.cancel。
func (s *accountDeviceSource) CancelAuthorization(ctx context.Context, attempt string) error {
	providerID := s.wire.attemptProvider(attempt)
	if providerID == "" {
		resolved, err := s.wire.accountProvider(ctx, "取消设备码登录")
		if err != nil {
			return err
		}
		providerID = resolved.ID
	}
	client, err := s.wire.app.requireControl("取消设备码登录")
	if err != nil {
		return err
	}
	_, err = client.Call(ctx, methodLoginCancel, map[string]any{
		"providerId": providerID,
		"attemptId":  attempt,
	})
	s.wire.forgetAttempt(attempt)
	return err
}

// deviceLoginStatus 把服务端状态映射成领域取值；未知状态按事实不合法拒绝。
func deviceLoginStatus(raw string) (account.DeviceLoginStatus, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "pending", "authorization_pending", "slow_down":
		return account.DeviceLoginPending, nil
	case "completed", "complete", "authorized", "success", "succeeded":
		return account.DeviceLoginCompleted, nil
	case "failed", "error", "denied", "access_denied":
		return account.DeviceLoginFailed, nil
	case "expired", "expired_token":
		return account.DeviceLoginExpired, nil
	case "canceled", "cancelled":
		return account.DeviceLoginCanceled, nil
	default:
		return "", failure.New(failure.KindNonRetryable, methodDeviceLoginStatus,
			fmt.Errorf("设备码状态未知: %q", raw))
	}
}

// wireUnixTime 解析秒或毫秒的时间戳；0 表示没有给出。
func wireUnixTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	if value > 1_000_000_000_000 {
		return time.UnixMilli(value).UTC()
	}
	return time.Unix(value, 0).UTC()
}

// 编译期确认：账号领域需要的三个来源都已提供。
var (
	_ account.ProviderSource                 = (*accountProviderSource)(nil)
	_ account.ProviderAuthorizationSource    = (*accountAuthSource)(nil)
	_ account.DeviceAuthorizationSource      = (*accountDeviceSource)(nil)
	_ account.AuthorizationCanceler          = (*accountDeviceSource)(nil)
	_ account.DeviceAuthorizationEmailSource = (*accountDeviceSource)(nil)
)
