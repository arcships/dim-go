package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 本文件是「模型供应」详情里按提供商登录（D6 的 provider 账号）：
// OpenAI Codex 的浏览器登录、Grok Build 的设备码登录，以及其他托管帐号的
// 浏览器 / 设备码登录。
//
// 边界与账号登录一致：
//   - 状态、登录方式与账号标识只有服务端一个来源；未读到就是未读到，不推断。
//   - 发起、轮询、完成、取消与登出都只发一次，由调用方驱动；本包不轮询、
//     不重试、不持有任何令牌或授权码副本。
//   - 命名空间由服务端给出（account / account.codex / account.grok）；
//     托管帐号按 providerId 定位，原生 OAuth 只有一个账号。
//
// 不包含：界面、wire 形状、写登记（登录不是配置写入，与账号登录同一口径）。

// ProviderOAuthNamespace 是一个提供商账号的命名空间。它来自
// `provider.management` 的 account.namespace，本包按字符串原样交给集成层。
type ProviderOAuthNamespace string

const (
	// ProviderNamespaceAccount 是托管帐号（Dim 与其它 account.providers 条目）。
	ProviderNamespaceAccount ProviderOAuthNamespace = "account"
	// ProviderNamespaceCodex 是 OpenAI Codex 的账号。
	ProviderNamespaceCodex ProviderOAuthNamespace = "account.codex"
	// ProviderNamespaceGrok 是 Grok Build 的账号。
	ProviderNamespaceGrok ProviderOAuthNamespace = "account.grok"
)

// 命名空间错误。
var (
	// ErrProviderOAuthUnavailable 表示本端没有提供商账号通道。
	ErrProviderOAuthUnavailable = errors.New("提供商账号登录不可用")
	// ErrProviderOAuthRequest 表示请求结构不合法，请求没有发出。
	ErrProviderOAuthRequest = errors.New("提供商账号请求不合法")
)

// ProviderOAuthStatus 是一个提供商账号的状态：只保留界面要显示的非敏感字段。
// 令牌、授权码与刷新凭据从不下发到本层。
type ProviderOAuthStatus struct {
	// Authenticated 为真表示服务端确认已登录。
	Authenticated bool
	// RecoveryRequired 表示服务端要求重新授权（已保存的凭据无法再续期）。
	RecoveryRequired bool
	// AccountLabel 是账号标识（托管帐号给邮箱 / 名称；原生 OAuth 给账号预览）。
	AccountLabel string
	// AccountPreview 是服务端给出的账号预览（通常是脱敏后的标识）。
	AccountPreview string
	// Plan 是服务端给出的套餐名；空表示没有这一项。
	Plan string
	// ExpiresAt 是凭据到期时间；零值表示服务端没有给。
	ExpiresAt time.Time
}

// ProviderOAuthStart 是一次已发起的提供商登录：浏览器方式给授权地址，设备码
// 方式给验证地址与用户码。它不含任何令牌或授权码。
type ProviderOAuthStart struct {
	Attempt string
	// AuthorizationURL 只对浏览器方式有值。
	AuthorizationURL string
	// RedirectURI 是服务端声明的回调地址；空表示服务端没有给。
	RedirectURI string
	// VerificationURI / VerificationURIComplete / UserCode 只对设备码方式有值。
	VerificationURI         string
	VerificationURIComplete string
	UserCode                string
	// Interval 是设备码轮询间隔；零值表示使用 DefaultDevicePollInterval。
	Interval time.Duration
	// ExpiresAt 是这次尝试的过期时间；零值表示由本地 LoginTimeout 决定。
	ExpiresAt time.Time
}

// ProviderOAuthStage 是一次提供商登录尝试的状态。
type ProviderOAuthStage string

const (
	ProviderOAuthPending   ProviderOAuthStage = "pending"
	ProviderOAuthCompleted ProviderOAuthStage = "completed"
	ProviderOAuthFailed    ProviderOAuthStage = "failed"
	ProviderOAuthCancelled ProviderOAuthStage = "cancelled"
	ProviderOAuthExpired   ProviderOAuthStage = "expired"
)

// ProviderOAuthProgress 是一次尝试的服务端状态；Reason 只在失败时非空。
type ProviderOAuthProgress struct {
	Stage  ProviderOAuthStage
	Reason string
}

// ProviderOAuthSource 是提供商账号的服务端侧接口，由集成层实现。
type ProviderOAuthSource interface {
	ReadProviderOAuthStatus(ctx context.Context, namespace, providerID string) (ProviderOAuthStatus, error)
	BeginProviderBrowser(ctx context.Context, namespace, providerID, redirectURI string) (ProviderOAuthStart, error)
	BeginProviderDevice(ctx context.Context, namespace, providerID string) (ProviderOAuthStart, error)
	CompleteProviderBrowser(ctx context.Context, namespace, providerID, attempt, callbackURL string) error
	PollProviderOAuth(ctx context.Context, namespace, providerID, attempt string) (ProviderOAuthProgress, error)
	CancelProviderOAuth(ctx context.Context, namespace, providerID, attempt string) error
	LogoutProviderOAuth(ctx context.Context, namespace, providerID string) error
}

// ProviderOAuth 是按提供商登录的领域入口。它不缓存状态：状态每次由
// provider 读数带回，本类型只做请求校验收口。
type ProviderOAuth struct {
	source ProviderOAuthSource
	now    func() time.Time
}

// NewProviderOAuth 建一个提供商账号入口；没有通道时返回 nil，调用方据此置灰。
func NewProviderOAuth(source ProviderOAuthSource) *ProviderOAuth {
	if source == nil {
		return nil
	}
	return &ProviderOAuth{source: source, now: time.Now}
}

// validateTarget 校验命名空间与定位参数：托管帐号要 providerId，原生 OAuth 不认
// providerId（服务端只有一个账号），认不出的命名空间直接拒绝。
func validateTarget(namespace, providerID string) error {
	switch ProviderOAuthNamespace(strings.TrimSpace(namespace)) {
	case ProviderNamespaceAccount:
		if strings.TrimSpace(providerID) == "" {
			return fmt.Errorf("%w: 托管帐号缺少 provider 标识", ErrProviderOAuthRequest)
		}
		return nil
	case ProviderNamespaceCodex, ProviderNamespaceGrok:
		return nil
	default:
		return fmt.Errorf("%w: 未知的账号命名空间 %q", ErrProviderOAuthRequest, namespace)
	}
}

// ReadStatus 读一个提供商账号的非敏感状态。
func (o *ProviderOAuth) ReadStatus(ctx context.Context, namespace, providerID string) (ProviderOAuthStatus, error) {
	if err := validateTarget(namespace, providerID); err != nil {
		return ProviderOAuthStatus{}, err
	}
	return o.source.ReadProviderOAuthStatus(ctx, namespace, providerID)
}

// BeginBrowser 发起浏览器登录；redirectURI 为空时交给服务端默认值。
func (o *ProviderOAuth) BeginBrowser(ctx context.Context, namespace, providerID, redirectURI string) (ProviderOAuthStart, error) {
	if err := validateTarget(namespace, providerID); err != nil {
		return ProviderOAuthStart{}, err
	}
	start, err := o.source.BeginProviderBrowser(ctx, namespace, providerID, strings.TrimSpace(redirectURI))
	if err != nil {
		return ProviderOAuthStart{}, err
	}
	if strings.TrimSpace(start.Attempt) == "" {
		return ProviderOAuthStart{}, fmt.Errorf("%w: 授权请求缺少尝试标识", ErrProviderOAuthRequest)
	}
	if strings.TrimSpace(start.AuthorizationURL) == "" {
		return ProviderOAuthStart{}, fmt.Errorf("%w: 授权请求缺少地址", ErrProviderOAuthRequest)
	}
	return start, nil
}

// BeginDevice 发起设备码登录。
func (o *ProviderOAuth) BeginDevice(ctx context.Context, namespace, providerID string) (ProviderOAuthStart, error) {
	if err := validateTarget(namespace, providerID); err != nil {
		return ProviderOAuthStart{}, err
	}
	start, err := o.source.BeginProviderDevice(ctx, namespace, providerID)
	if err != nil {
		return ProviderOAuthStart{}, err
	}
	if strings.TrimSpace(start.Attempt) == "" {
		return ProviderOAuthStart{}, fmt.Errorf("%w: 设备码登录缺少尝试标识", ErrProviderOAuthRequest)
	}
	if strings.TrimSpace(start.VerificationURI) == "" || strings.TrimSpace(start.UserCode) == "" {
		return ProviderOAuthStart{}, fmt.Errorf("%w: 设备码登录缺少验证地址或用户码", ErrProviderOAuthRequest)
	}
	if start.Interval <= 0 {
		start.Interval = DefaultDevicePollInterval
	}
	return start, nil
}

// CompleteBrowser 用回调地址完成浏览器登录的一次交换。
func (o *ProviderOAuth) CompleteBrowser(ctx context.Context, namespace, providerID, attempt, callbackURL string) error {
	if err := validateTarget(namespace, providerID); err != nil {
		return err
	}
	if strings.TrimSpace(attempt) == "" {
		return fmt.Errorf("%w: 缺少尝试标识", ErrProviderOAuthRequest)
	}
	callback := strings.TrimSpace(callbackURL)
	if callback == "" {
		return fmt.Errorf("%w: 回调地址不能为空", ErrProviderOAuthRequest)
	}
	return o.source.CompleteProviderBrowser(ctx, namespace, providerID, attempt, callback)
}

// Poll 查一次尝试状态；不做等待，轮询节奏由调用方决定。
func (o *ProviderOAuth) Poll(ctx context.Context, namespace, providerID, attempt string) (ProviderOAuthProgress, error) {
	if err := validateTarget(namespace, providerID); err != nil {
		return ProviderOAuthProgress{}, err
	}
	if strings.TrimSpace(attempt) == "" {
		return ProviderOAuthProgress{}, fmt.Errorf("%w: 缺少尝试标识", ErrProviderOAuthRequest)
	}
	progress, err := o.source.PollProviderOAuth(ctx, namespace, providerID, attempt)
	if err != nil {
		return ProviderOAuthProgress{}, err
	}
	switch progress.Stage {
	case ProviderOAuthPending, ProviderOAuthCompleted, ProviderOAuthFailed,
		ProviderOAuthCancelled, ProviderOAuthExpired:
		return progress, nil
	}
	return ProviderOAuthProgress{}, fmt.Errorf("%w: 未知的登录状态 %q", ErrProviderOAuthRequest, progress.Stage)
}

// Cancel 取消一次进行中的尝试；没有尝试标识时什么也不做。
func (o *ProviderOAuth) Cancel(ctx context.Context, namespace, providerID, attempt string) error {
	if err := validateTarget(namespace, providerID); err != nil {
		return err
	}
	if strings.TrimSpace(attempt) == "" {
		return nil
	}
	return o.source.CancelProviderOAuth(ctx, namespace, providerID, attempt)
}

// Logout 登出一个提供商账号（清除服务端保存的凭据）。
func (o *ProviderOAuth) Logout(ctx context.Context, namespace, providerID string) error {
	if err := validateTarget(namespace, providerID); err != nil {
		return err
	}
	return o.source.LogoutProviderOAuth(ctx, namespace, providerID)
}
