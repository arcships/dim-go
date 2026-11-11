package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// LoginMethod 是服务端为某个 provider 声明的登录方式。本包只认浏览器与设备码
// 两种；其他取值原样保留但不参与分发。
type LoginMethod string

const (
	// LoginMethodBrowser 是授权页 + 环回回调。
	LoginMethodBrowser LoginMethod = "browser"
	// LoginMethodDevice 是设备码：展示验证地址与用户码，轮询到完成。
	LoginMethodDevice LoginMethod = "device"
)

// Provider 是服务端声明的一个账号提供者。登录方式必须由服务端给出，本包不推断。
type Provider struct {
	// ID 是 provider 标识，必填。
	ID string
	// DisplayName 是展示名；空表示服务端没有给出。
	DisplayName string
	// LoginMethods 是服务端声明的登录方式。
	LoginMethods []LoginMethod
	// Directory 是账号目录（发现地址）；空表示使用服务端默认。
	Directory string
}

// ProviderSource 读取服务端声明的账号提供者。只读，不写、不重试。
type ProviderSource interface {
	ReadProviders(ctx context.Context) ([]Provider, error)
}

// LoginRequest 是一次登录请求。
type LoginRequest struct {
	// ProviderID 指定 provider；空表示只有唯一 provider 时自动选择，多个时必须指定。
	ProviderID string
	// Method 指定登录方式；空表示按 provider 声明选择：有 browser 用 browser，
	// 否则只有 device 时用 device。显式指定的方式必须被 provider 声明。
	Method LoginMethod
	// Email 是可选的登录提示（login_hint），原客户端同样透传。
	Email string
}

// LoginStart 是一次已发起的登录尝试。浏览器方式给授权地址；设备码方式给验证
// 地址与用户码。它不含任何令牌或授权码。
type LoginStart struct {
	ProviderID string
	Method     LoginMethod
	// Attempt 是服务端给出的尝试标识；取消与轮询按它进行。
	Attempt string
	// AuthorizationURL 只对 browser 有值，交给系统浏览器打开。
	AuthorizationURL string
	// VerificationURI / VerificationURIComplete / UserCode 只对 device 有值。
	VerificationURI         string
	VerificationURIComplete string
	UserCode                string
	// ExpiresAt 是尝试的过期时间；零值表示由本地 LoginTimeout 决定。
	ExpiresAt time.Time
	// Interval 是设备码轮询间隔；零值表示使用 DefaultDevicePollInterval。
	Interval time.Duration
}

// DeviceAuthorization 是服务端给出的设备码登录信息。
type DeviceAuthorization struct {
	Attempt                 string
	VerificationURI         string
	VerificationURIComplete string
	UserCode                string
	ExpiresAt               time.Time
	Interval                time.Duration
}

// DeviceLoginStatus 是设备码尝试的状态。
type DeviceLoginStatus string

const (
	// DeviceLoginPending 表示用户还没完成授权。
	DeviceLoginPending DeviceLoginStatus = "pending"
	// DeviceLoginCompleted 表示服务端确认完成。
	DeviceLoginCompleted DeviceLoginStatus = "completed"
	// DeviceLoginFailed 表示服务端报告失败。
	DeviceLoginFailed DeviceLoginStatus = "failed"
	// DeviceLoginExpired 表示尝试过期。
	DeviceLoginExpired DeviceLoginStatus = "expired"
	// DeviceLoginCanceled 表示尝试被取消。
	DeviceLoginCanceled DeviceLoginStatus = "canceled"
)

// DeviceLoginState 是一次设备码轮询的结果。
type DeviceLoginState struct {
	Status DeviceLoginStatus
	// Reason 在失败或取消时说明原因；其他状态可为空。
	Reason string
}

// DeviceAuthorizationSource 是设备码登录的服务端侧接口：发起尝试、轮询状态。
// 它由集成层实现，把 wire 形状映射到这两个方法；本包不复制 wire 定义。
type DeviceAuthorizationSource interface {
	BeginDeviceAuthorization(ctx context.Context, providerID, directory string) (DeviceAuthorization, error)
	PollDeviceAuthorization(ctx context.Context, attempt string) (DeviceLoginState, error)
}

// DeviceAuthorizationEmailSource 是可选能力：设备码发起时带登录提示（login_hint）。
// 不实现时忽略 LoginRequest.Email，其余流程不变。
type DeviceAuthorizationEmailSource interface {
	BeginDeviceAuthorizationWithEmail(
		ctx context.Context, providerID, directory, email string,
	) (DeviceAuthorization, error)
}

// Providers 读取服务端声明的账号提供者。没有 provider 读取能力时明确不可用，
// 不构造替代提供者。
func (a *Account) Providers(ctx context.Context) ([]Provider, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if a.cfg.Providers == nil {
		return nil, ErrProvidersUnavailable
	}
	providers, err := a.cfg.Providers.ReadProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProvidersRead, err)
	}
	return validateProviders(providers)
}

// resolveLoginProvider 选择 provider 与登录方式。多个 provider 且未指定时拒绝，
// 不在候选之间猜。
func (a *Account) resolveLoginProvider(ctx context.Context, request LoginRequest) (Provider, LoginMethod, error) {
	providers, err := a.Providers(ctx)
	if err != nil {
		return Provider{}, "", err
	}
	provider, err := selectProvider(providers, request.ProviderID)
	if err != nil {
		return Provider{}, "", err
	}
	method, err := selectLoginMethod(provider, request.Method)
	if err != nil {
		return Provider{}, "", err
	}
	return provider, method, nil
}

func validateProviders(providers []Provider) ([]Provider, error) {
	seen := make(map[string]bool, len(providers))
	validated := make([]Provider, 0, len(providers))
	for _, provider := range providers {
		id := strings.TrimSpace(provider.ID)
		if id == "" {
			return nil, fmt.Errorf("%w: provider 缺少标识", ErrMalformedProviders)
		}
		if seen[id] {
			return nil, fmt.Errorf("%w: provider 标识重复 %q", ErrMalformedProviders, id)
		}
		seen[id] = true
		provider.ID = id
		provider.LoginMethods = append([]LoginMethod(nil), provider.LoginMethods...)
		validated = append(validated, provider)
	}
	if len(validated) == 0 {
		return nil, fmt.Errorf("%w: 服务端没有声明账号 provider", ErrProvidersUnavailable)
	}
	return validated, nil
}

func selectProvider(providers []Provider, requested string) (Provider, error) {
	if requested != "" {
		for _, provider := range providers {
			if provider.ID == requested {
				return provider, nil
			}
		}
		return Provider{}, fmt.Errorf("%w: %q", ErrProviderUnknown, requested)
	}
	if len(providers) == 1 {
		return providers[0], nil
	}
	return Provider{}, fmt.Errorf("%w: 服务端声明了 %d 个 provider", ErrProviderSelectionRequired, len(providers))
}

// selectLoginMethod 按声明选择登录方式。显式指定必须被声明；未指定时优先
// browser，其次只在 provider 只声明 device 时使用 device。
func selectLoginMethod(provider Provider, requested LoginMethod) (LoginMethod, error) {
	if requested != "" {
		for _, declared := range provider.LoginMethods {
			if declared == requested {
				if !requested.supported() {
					return "", fmt.Errorf("%w: %q", ErrLoginMethodUnavailable, requested)
				}
				return requested, nil
			}
		}
		return "", fmt.Errorf("%w: provider %q 未声明 %q（已声明 %s）",
			ErrLoginMethodUnavailable, provider.ID, requested, declaredMethods(provider))
	}
	for _, declared := range provider.LoginMethods {
		if declared == LoginMethodBrowser {
			return LoginMethodBrowser, nil
		}
	}
	for _, declared := range provider.LoginMethods {
		if declared == LoginMethodDevice {
			return LoginMethodDevice, nil
		}
	}
	return "", fmt.Errorf("%w: provider %q 没有可用的登录方式（已声明 %s）",
		ErrLoginMethodUnavailable, provider.ID, declaredMethods(provider))
}

func (m LoginMethod) supported() bool {
	return m == LoginMethodBrowser || m == LoginMethodDevice
}

func declaredMethods(provider Provider) string {
	if len(provider.LoginMethods) == 0 {
		return "无"
	}
	names := make([]string, 0, len(provider.LoginMethods))
	for _, method := range provider.LoginMethods {
		names = append(names, string(method))
	}
	return strings.Join(names, "/")
}

// provider 与登录方式相关错误。
var (
	// ErrProvidersUnavailable 表示本端没有 provider 读取能力，或服务端没有声明 provider。
	ErrProvidersUnavailable = errors.New("账号登录不可用：缺少 provider 声明")
	// ErrProvidersRead 表示 provider 读取失败。
	ErrProvidersRead = errors.New("账号 provider 读取失败")
	// ErrMalformedProviders 表示 provider 声明结构不合法。
	ErrMalformedProviders = errors.New("账号 provider 声明不合法")
	// ErrProviderUnknown 表示指定的 provider 不存在。
	ErrProviderUnknown = errors.New("账号 provider 不存在")
	// ErrProviderSelectionRequired 表示存在多个 provider 但没有指定。
	ErrProviderSelectionRequired = errors.New("必须指定账号 provider")
	// ErrLoginMethodUnavailable 表示 provider 没有声明该登录方式，或本端不支持它。
	ErrLoginMethodUnavailable = errors.New("该账号 provider 不支持所选登录方式")
)
