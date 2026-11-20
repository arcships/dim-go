package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arcships/dimagent-mygo/internal/account"
)

// 本文件把「按提供商登录」接到 control 连接的账号方法上（见 docs/domains/D6）。
//
// 命名空间由服务端的 provider.management 给出：
//
//	account       托管帐号：目标是一个 providerId，发起时带账号目录与回调地址；
//	account.codex OpenAI Codex：只有一个账号，发起不带参数；
//	account.grok  Grok Build：只有一个账号，发起不带参数。
//
// 方法后缀在各命名空间一致：status / login.browser.begin /
// login.browser.complete / login.begin / login.status / login.cancel / logout。
// 本文件不缓存状态，也不记录授权码、设备码原文或回调地址。

// providerOAuthSuffixes 是各命名空间共用的方法后缀。
const (
	providerOAuthStatusSuffix    = ".status"
	providerOAuthBrowserBegin    = ".login.browser.begin"
	providerOAuthBrowserComplete = ".login.browser.complete"
	providerOAuthDeviceBegin     = ".login.begin"
	providerOAuthDeviceStatus    = ".login.status"
	providerOAuthCancel          = ".login.cancel"
	providerOAuthLogout          = ".logout"
)

// providerOAuthSource 实现 account.ProviderOAuthSource。
type providerOAuthSource struct {
	app *App
}

// providerOAuthParams 是命名空间下的定位参数：只有托管帐号带 providerId。
func providerOAuthParams(namespace, providerID string) map[string]any {
	if account.ProviderOAuthNamespace(namespace) == account.ProviderNamespaceAccount {
		return map[string]any{"providerId": providerID}
	}
	return map[string]any{}
}

// providerOAuthCall 走一次控制连接的账号方法；缺方法时按能力缺失返回，不发请求。
func (s *providerOAuthSource) providerOAuthCall(
	ctx context.Context, op, method string, params map[string]any, into any,
) error {
	if !s.app.controlSupports(method) {
		return capabilityMissing(method)
	}
	client, err := s.app.requireControl(op)
	if err != nil {
		return err
	}
	if params == nil {
		params = map[string]any{}
	}
	return client.CallInto(ctx, method, params, into)
}

// ReadProviderOAuthStatus 读一个提供商账号的非敏感状态。
func (s *providerOAuthSource) ReadProviderOAuthStatus(
	ctx context.Context, namespace, providerID string,
) (account.ProviderOAuthStatus, error) {
	method := namespace + providerOAuthStatusSuffix
	params := providerOAuthParams(namespace, providerID)
	if account.ProviderOAuthNamespace(namespace) == account.ProviderNamespaceAccount {
		var reply struct {
			Account *struct {
				Issuer  string `json:"issuer"`
				Subject string `json:"subject"`
				Email   string `json:"email"`
				Name    string `json:"name"`
			} `json:"account"`
		}
		if err := s.providerOAuthCall(ctx, "读取提供商账号", method, params, &reply); err != nil {
			return account.ProviderOAuthStatus{}, err
		}
		status := account.ProviderOAuthStatus{}
		if reply.Account != nil {
			status.Authenticated = true
			status.AccountLabel = firstNonEmpty(reply.Account.Name, reply.Account.Email, reply.Account.Subject)
			status.AccountPreview = reply.Account.Subject
		}
		return status, nil
	}
	var reply struct {
		Authenticated    bool   `json:"authenticated"`
		RecoveryRequired bool   `json:"recoveryRequired"`
		ExpiresAt        *int64 `json:"expiresAt"`
		AccountIDPreview string `json:"accountIdPreview"`
	}
	if err := s.providerOAuthCall(ctx, "读取提供商账号", method, params, &reply); err != nil {
		return account.ProviderOAuthStatus{}, err
	}
	status := account.ProviderOAuthStatus{
		Authenticated:    reply.Authenticated,
		RecoveryRequired: reply.RecoveryRequired,
		AccountPreview:   reply.AccountIDPreview,
	}
	if reply.Authenticated {
		status.AccountLabel = reply.AccountIDPreview
	}
	if reply.ExpiresAt != nil && *reply.ExpiresAt > 0 {
		status.ExpiresAt = time.Unix(*reply.ExpiresAt, 0)
	}
	return status, nil
}

// providerOAuthDirectory 找托管帐号条目的默认目录：发起浏览器 / 设备码登录时
// 服务端要求它。找不到或没有目录时返回空，由服务端按默认处理。
func (s *providerOAuthSource) providerOAuthDirectory(ctx context.Context, providerID string) string {
	client, err := s.app.requireControl("读取账号提供者")
	if err != nil {
		return ""
	}
	var providers []struct {
		ProviderID       string  `json:"providerId"`
		DefaultDirectory *string `json:"defaultDirectory"`
	}
	if err := client.CallInto(ctx, methodAccountProviders, nil, &providers); err != nil {
		return ""
	}
	for _, provider := range providers {
		if provider.ProviderID != providerID || provider.DefaultDirectory == nil {
			continue
		}
		return strings.TrimSpace(*provider.DefaultDirectory)
	}
	return ""
}

// BeginProviderBrowser 发起浏览器登录，返回授权地址与尝试标识。
func (s *providerOAuthSource) BeginProviderBrowser(
	ctx context.Context, namespace, providerID, redirectURI string,
) (account.ProviderOAuthStart, error) {
	method := namespace + providerOAuthBrowserBegin
	params := providerOAuthParams(namespace, providerID)
	if account.ProviderOAuthNamespace(namespace) == account.ProviderNamespaceAccount {
		params["redirectUri"] = redirectURI
		if directory := s.providerOAuthDirectory(ctx, providerID); directory != "" {
			params["directory"] = directory
		}
	}
	var reply struct {
		AttemptID        string `json:"attemptId"`
		AuthorizationURL string `json:"authorizationUrl"`
		RedirectURI      string `json:"redirectUri"`
	}
	if err := s.providerOAuthCall(ctx, "发起提供商登录", method, params, &reply); err != nil {
		return account.ProviderOAuthStart{}, err
	}
	return account.ProviderOAuthStart{
		Attempt:          reply.AttemptID,
		AuthorizationURL: reply.AuthorizationURL,
		RedirectURI:      reply.RedirectURI,
	}, nil
}

// BeginProviderDevice 发起设备码登录，返回验证地址与用户码。
func (s *providerOAuthSource) BeginProviderDevice(
	ctx context.Context, namespace, providerID string,
) (account.ProviderOAuthStart, error) {
	method := namespace + providerOAuthDeviceBegin
	params := providerOAuthParams(namespace, providerID)
	if account.ProviderOAuthNamespace(namespace) == account.ProviderNamespaceAccount {
		if directory := s.providerOAuthDirectory(ctx, providerID); directory != "" {
			params["directory"] = directory
		}
	}
	var reply struct {
		AttemptID               string `json:"attemptId"`
		VerificationURI         string `json:"verificationUri"`
		VerificationURIComplete string `json:"verificationUriComplete"`
		UserCode                string `json:"userCode"`
		IntervalSeconds         int64  `json:"intervalSeconds"`
		ExpiresAt               int64  `json:"expiresAt"`
	}
	if err := s.providerOAuthCall(ctx, "发起提供商登录", method, params, &reply); err != nil {
		return account.ProviderOAuthStart{}, err
	}
	start := account.ProviderOAuthStart{
		Attempt:                 reply.AttemptID,
		VerificationURI:         reply.VerificationURI,
		VerificationURIComplete: reply.VerificationURIComplete,
		UserCode:                reply.UserCode,
	}
	if reply.IntervalSeconds > 0 {
		start.Interval = time.Duration(reply.IntervalSeconds) * time.Second
	}
	if reply.ExpiresAt > 0 {
		start.ExpiresAt = time.Unix(reply.ExpiresAt, 0)
	}
	return start, nil
}

// CompleteProviderBrowser 用回调地址完成浏览器登录。
func (s *providerOAuthSource) CompleteProviderBrowser(
	ctx context.Context, namespace, providerID, attempt, callbackURL string,
) error {
	method := namespace + providerOAuthBrowserComplete
	params := providerOAuthParams(namespace, providerID)
	params["attemptId"] = attempt
	params["callbackUrl"] = callbackURL
	return s.providerOAuthCall(ctx, "完成提供商登录", method, params, nil)
}

// PollProviderOAuth 查一次尝试状态。
func (s *providerOAuthSource) PollProviderOAuth(
	ctx context.Context, namespace, providerID, attempt string,
) (account.ProviderOAuthProgress, error) {
	method := namespace + providerOAuthDeviceStatus
	params := providerOAuthParams(namespace, providerID)
	params["attemptId"] = attempt
	var reply struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := s.providerOAuthCall(ctx, "查询提供商登录", method, params, &reply); err != nil {
		return account.ProviderOAuthProgress{}, err
	}
	stage := account.ProviderOAuthStage(strings.ToLower(strings.TrimSpace(reply.Status)))
	if stage == "" {
		return account.ProviderOAuthProgress{}, fmt.Errorf("登录状态回执缺少 status")
	}
	return account.ProviderOAuthProgress{Stage: stage, Reason: providerOAuthReason(reply.Error)}, nil
}

// providerOAuthReason 把服务端的错误码换成一句可读原因；认不出的保留原文。
func providerOAuthReason(code string) string {
	switch strings.TrimSpace(code) {
	case "":
		return ""
	case "oauth_failed":
		return "授权服务端拒绝了这次登录"
	case "account_changed":
		return "账号已变化，请重新登录"
	case "login_attempt_closed":
		return "这次登录已经结束，请重新发起"
	case "login_attempt_limit":
		return "登录尝试过多，请稍后再试"
	}
	return code
}

// CancelProviderOAuth 取消一次进行中的尝试。
func (s *providerOAuthSource) CancelProviderOAuth(
	ctx context.Context, namespace, providerID, attempt string,
) error {
	method := namespace + providerOAuthCancel
	params := providerOAuthParams(namespace, providerID)
	params["attemptId"] = attempt
	return s.providerOAuthCall(ctx, "取消提供商登录", method, params, nil)
}

// LogoutProviderOAuth 登出一个提供商账号。
func (s *providerOAuthSource) LogoutProviderOAuth(
	ctx context.Context, namespace, providerID string,
) error {
	method := namespace + providerOAuthLogout
	params := providerOAuthParams(namespace, providerID)
	return s.providerOAuthCall(ctx, "退出提供商账号", method, params, nil)
}

// 编译期确认本文件实现了账号领域的来源接口。
var _ account.ProviderOAuthSource = (*providerOAuthSource)(nil)
