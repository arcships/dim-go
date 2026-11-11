// Package account 是账号领域（D6）：账号状态、浏览器登录、登出、用量与按账号
// 状态过滤的可用模型。
//
// 边界：
//   - 账号状态、身份字段、用量与模型清单都只有服务端一个来源；本包只保留最近一次
//     读取确认的事实。未登录就是未登录，尚未读取就是未知，不推断、不给默认值。
//   - 登录由服务端完成，客户端只生成环回回调地址、打开系统浏览器、把回调结果交给
//     授权来源；不内嵌浏览器、不抓取凭据，也不保存授权码或令牌副本。
//   - 登出与重置登记成写意图，由写登记（D3）发送一次并收敛；本包不自行发送、不自行
//     重试，结果未知时不重复提交。
//   - 用量逐项取自服务端；聚合项由服务端给出，服务端没有就明确不可用，不在本地求和。
//   - 可用模型按当前账号状态过滤；未登录、状态未知或登录过期时返回空清单并说明原因。
//
// 不包含：界面、连接实现、协议 wire 定义、管理面设置写入、模型档位选择。
package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// Status 是账号状态。它只由服务端读取结果填充；StatusUnknown 表示本端尚未得到
// 服务端确认，不等于未登录。
type Status string

const (
	// StatusUnknown 表示尚未读取或已清空，服务端没有确认过当前状态。
	StatusUnknown Status = "unknown"
	// StatusSignedOut 表示服务端确认未登录。
	StatusSignedOut Status = "signed_out"
	// StatusSignedIn 表示服务端确认已登录。
	StatusSignedIn Status = "signed_in"
	// StatusExpired 表示服务端确认登录已过期。
	StatusExpired Status = "expired"
)

// Identity 是账号的身份字段。空值表示服务端没有给出该字段，本包不补默认值。
type Identity struct {
	AccountID   string
	Email       string
	DisplayName string
	// PreferredUsername 是原客户端 AccountUser 里的同一字段；服务端状态回执
	// 可能没有，空表示未知，不用账号名顶替。
	PreferredUsername string
	Plan              string
}

// Fact 是一次账号状态读取的归一化结果。wire 形状由连接领域负责，集成层在交界处
// 把它映射成这个结构。
type Fact struct {
	Status   Status
	Identity Identity
}

// validate 只做结构校验：状态必须是服务端能给出的取值；已登录或已过期必须带账号
// 标识。它不解释身份字段的业务含义，也不把未知状态降级成未登录。
func (f Fact) validate() error {
	switch f.Status {
	case StatusSignedIn, StatusExpired:
		if strings.TrimSpace(f.Identity.AccountID) == "" {
			return fmt.Errorf("%w: 状态 %q 缺少账号标识", ErrMalformedFact, f.Status)
		}
	case StatusSignedOut:
	default:
		return fmt.Errorf("%w: 未知账号状态 %q", ErrMalformedFact, f.Status)
	}
	return nil
}

// Snapshot 是账号与用量的只读视图。
//
// Status/Identity/Usage 是最近一次服务端读取确认的事实；ReadAt 是那次读取的时间。
// ReadError 非空说明最近一次读取失败，但不会把已经确认的事实改成未知。
type Snapshot struct {
	Status   Status
	Identity Identity
	Usage    Usage
	ReadAt   time.Time
	// Reason 说明当前状态的来由（尚未读取、登出已确认等）；空表示由最近一次读取确认。
	Reason string
	// ReadError 是最近一次状态读取失败的原因；空表示最近一次读取成功。
	ReadError string
}

// StatusReader 读取账号状态。只读，不写、不重试。
type StatusReader interface {
	ReadAccount(ctx context.Context) (Fact, error)
}

// UsageReader 逐项读取用量。聚合项不可用时用 Available=false 加原因表达，不得
// 用零值冒充，也不得让调用方在本地求和。
type UsageReader interface {
	ReadUsage(ctx context.Context) (Usage, error)
}

// ModelReader 读取服务端声明的模型清单。清单里的可用性由服务端给出。
type ModelReader interface {
	ReadModels(ctx context.Context) ([]Model, error)
}

// Writer 是写登记的窄接口；*write.Manager 直接满足它。登出与重置都必须经过它。
type Writer interface {
	Register(intent write.Intent) (write.IntentView, error)
	Wait(ctx context.Context, intentID string) (write.IntentView, error)
}

// 编译期确认写登记可以直接交给本领域使用。
var _ Writer = (*write.Manager)(nil)

// Config 是账号领域的配置。Status 必填；其余按能力使用，缺能力时明确报不可用，
// 不构造替代事实。
type Config struct {
	// Status 读取账号状态。
	Status StatusReader
	// Usage 读取用量；nil 表示本端不提供用量读取。
	Usage UsageReader
	// Models 读取模型清单；nil 表示本端不提供模型清单读取。
	Models ModelReader
	// Providers 读取服务端声明的账号 provider 与登录方式；nil 表示本端不提供
	// provider 专属登录。
	Providers ProviderSource
	// Auth 是浏览器登录来源：实现 AuthorizationSource（单 provider 兼容）或
	// ProviderAuthorizationSource（按 provider）。nil 表示本端不支持 browser 登录。
	Auth any
	// DeviceAuth 发起与轮询设备码登录；nil 表示本端不支持 device 登录。
	DeviceAuth DeviceAuthorizationSource
	// Browser 打开系统浏览器；nil 时按平台调用系统默认处理程序。
	Browser Browser
	// Actions 是写登记；登出与重置经它发出。nil 表示本端不提供账号动作。
	Actions Writer
	// LoginTimeout 是一次登录等待回调的上限；0 表示使用 DefaultLoginTimeout。
	LoginTimeout time.Duration
	// Now 可在测试中替换；nil 时使用系统时间。
	Now func() time.Time
}

func (c Config) withDefaults() Config {
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.Browser == nil {
		c.Browser = systemBrowser{}
	}
	if c.LoginTimeout <= 0 {
		c.LoginTimeout = DefaultLoginTimeout
	}
	return c
}

// Account 持有账号与用量的镜像状态。所有导出方法可并发调用。
type Account struct {
	cfg Config

	mu            sync.Mutex
	status        Status
	identity      Identity
	usage         Usage
	readAt        time.Time
	reason        string
	readError     string
	models        []Model
	modelsOK      bool
	modelsErr     string
	loginPhase    LoginPhase
	loginFrom     time.Time
	loginError    string
	loginProvider string
	loginMethod   LoginMethod
	// login 是当前进行中的尝试；尝试标识与授权码只随它存在内存里。
	login *loginAttempt

	// 账号动作只在内存里保留最近一次意图；重启即忘，不补发。
	logoutMu      sync.Mutex
	logoutAttempt int
	logoutView    *write.IntentView
	resetMu       sync.Mutex
	resetAttempt  int
	// resetKeys 按「订阅标识:重置类型」保存在不确定结果下复用的幂等键；只在内存。
	resetKeys map[string]string
}

// New 创建账号领域实例。Status 必填；登录尝试状态机在启动时校验。
func New(cfg Config) (*Account, error) {
	if cfg.Status == nil {
		return nil, ErrMissingStatusReader
	}
	if cfg.Auth != nil {
		if _, legacy := cfg.Auth.(AuthorizationSource); !legacy {
			if _, provider := cfg.Auth.(ProviderAuthorizationSource); !provider {
				return nil, ErrInvalidConfig
			}
		}
	}
	cfg = cfg.withDefaults()
	if err := validateLoginTable(loginTransitions); err != nil {
		return nil, err
	}
	return &Account{
		cfg:        cfg,
		status:     StatusUnknown,
		loginPhase: LoginIdle,
		reason:     "尚未读取账号状态",
	}, nil
}

// Refresh 读取账号状态，并用服务端事实替换本地镜像状态。
//
// 读取失败或事实结构不合法时不改状态，只记录 ReadError 并返回错误；调用方可以
// 继续显示上一次确认的事实（ReadAt 给出它是什么时候确认的）。
func (a *Account) Refresh(ctx context.Context) (Snapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	fact, err := a.cfg.Status.ReadAccount(ctx)
	if err != nil {
		return a.failRead(fmt.Errorf("%w: %v", ErrStatusRead, err))
	}
	if err := fact.validate(); err != nil {
		return a.failRead(err)
	}

	now := a.cfg.Now()
	a.mu.Lock()
	previous := a.status
	a.status = fact.Status
	a.readAt = now
	a.reason = ""
	a.readError = ""
	switch fact.Status {
	case StatusSignedOut:
		// 未登录时不保留上一账号的身份与用量。
		a.identity = Identity{}
		a.usage = Usage{}
		a.resetKeys = nil
	case StatusSignedIn:
		a.identity = fact.Identity
		if previous != StatusSignedIn {
			// 新的账号会话：上一次登出已经是历史，允许再次登出；重置键重新开始。
			a.logoutView = nil
			a.resetKeys = nil
		}
	default:
		a.identity = fact.Identity
	}
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	return snapshot, nil
}

// failRead 记录一次读取失败并返回当前快照与原因；状态保持不变。
func (a *Account) failRead(err error) (Snapshot, error) {
	a.mu.Lock()
	a.readError = err.Error()
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	return snapshot, err
}

// Snapshot 返回账号与用量的只读副本。
func (a *Account) Snapshot() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.snapshotLocked()
}

// Status 返回当前账号状态；未读取时为 StatusUnknown。
func (a *Account) Status() Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status
}

func (a *Account) snapshotLocked() Snapshot {
	return Snapshot{
		Status:    a.status,
		Identity:  a.identity,
		Usage:     cloneUsage(a.usage),
		ReadAt:    a.readAt,
		Reason:    a.reason,
		ReadError: a.readError,
	}
}

// Usage 返回最近一次确认的用量；Ready 为假时其余字段无意义。
func (a *Account) Usage() Usage {
	a.mu.Lock()
	defer a.mu.Unlock()
	return cloneUsage(a.usage)
}

// clearConfirmedLogout 处理已确认的登出：本地账号状态清空。它不是「服务端确认
// 未登录」，因此状态回到未知；重新读取后仍以服务端为准。
func (a *Account) clearConfirmedLogout() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = StatusUnknown
	a.identity = Identity{}
	a.usage = Usage{}
	a.resetKeys = nil
	a.readAt = time.Time{}
	a.readError = ""
	a.reason = "登出已确认，本地账号状态已清空；重新读取后以服务端为准"
}

func cloneUsage(usage Usage) Usage {
	usage.Lines = append([]UsageLine(nil), usage.Lines...)
	usage.Windows = append([]UsageWindow(nil), usage.Windows...)
	if usage.Features != nil {
		features := make([]UsageFeature, len(usage.Features))
		for index, feature := range usage.Features {
			feature.Used = cloneFloat(feature.Used)
			feature.Allowance = cloneFloat(feature.Allowance)
			feature.Remaining = cloneFloat(feature.Remaining)
			feature.Unlimited = cloneBool(feature.Unlimited)
			features[index] = feature
		}
		usage.Features = features
	}
	if usage.Plan != nil {
		plan := *usage.Plan
		plan.PriceAmount = cloneFloat(plan.PriceAmount)
		usage.Plan = &plan
	}
	if usage.Credits != nil {
		credits := *usage.Credits
		credits.UsedUnits = cloneFloat(credits.UsedUnits)
		credits.TotalUnits = cloneFloat(credits.TotalUnits)
		credits.RemainingUnits = cloneFloat(credits.RemainingUnits)
		credits.Unlimited = cloneBool(credits.Unlimited)
		usage.Credits = &credits
	}
	return usage
}

func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

// ErrorCode 把本领域的类型化错误映射成原客户端使用的稳定错误码，供集成层
// 组成 `AccountErrorPayload{code,message}`。没有匹配时返回空串，由调用方决定。
func ErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrProviderUnknown),
		errors.Is(err, ErrProviderSelectionRequired),
		errors.Is(err, ErrLoginMethodUnavailable),
		errors.Is(err, ErrMalformedProviders),
		errors.Is(err, ErrMalformedFact),
		errors.Is(err, ErrMalformedUsage),
		errors.Is(err, ErrMalformedModels),
		errors.Is(err, ErrResetInvalidType),
		errors.Is(err, ErrResetMissingSubscription):
		return "account_invalid_input"
	case errors.Is(err, ErrProvidersUnavailable),
		errors.Is(err, ErrActionsUnavailable),
		errors.Is(err, ErrMissingStatusReader),
		errors.Is(err, ErrInvalidConfig),
		errors.Is(err, ErrLoginUnavailable):
		return "account_provider_unavailable"
	case errors.Is(err, ErrLoginCanceled):
		return "login_cancelled"
	case errors.Is(err, ErrLoginTimeout), errors.Is(err, ErrLoginExpired):
		return "login_timeout"
	case errors.Is(err, ErrLoginInProgress):
		return "login_in_progress"
	case errors.Is(err, ErrLoginRequest),
		errors.Is(err, ErrBrowserOpen),
		errors.Is(err, ErrLoginDenied),
		errors.Is(err, ErrLoginStateMismatch),
		errors.Is(err, ErrLoginIncomplete),
		errors.Is(err, ErrLoginCompletion),
		errors.Is(err, ErrDeviceLogin):
		return "login_failed"
	case errors.Is(err, ErrResetRequiresAccount):
		return "account_required"
	case errors.Is(err, ErrStatusRead):
		return "account_status_failed"
	case errors.Is(err, ErrUsageRead), errors.Is(err, ErrUsageUnavailable):
		return "account_usage_failed"
	case errors.Is(err, ErrModelsRead), errors.Is(err, ErrModelsUnavailable):
		return "model_setup_failed"
	default:
		return ""
	}
}

// 领域错误。所有错误都带稳定语义，供上层区分「缺能力」「读取失败」与「动作未发出」。
var (
	// ErrMissingStatusReader 表示没有账号状态读取来源。
	ErrMissingStatusReader = errors.New("账号缺少状态读取来源")
	// ErrInvalidConfig 表示配置里的能力实现不被支持。
	ErrInvalidConfig = errors.New("账号配置不合法")
	// ErrMalformedFact 表示服务端给出的账号事实结构不合法。
	ErrMalformedFact = errors.New("账号事实不合法")
	// ErrStatusRead 表示账号状态读取失败。
	ErrStatusRead = errors.New("账号状态读取失败")
	// ErrUsageRead 表示用量读取失败。
	ErrUsageRead = errors.New("用量读取失败")
	// ErrUsageUnavailable 表示本端不提供用量读取。
	ErrUsageUnavailable = errors.New("本端不提供用量读取")
	// ErrMalformedUsage 表示服务端给出的用量结构不合法。
	ErrMalformedUsage = errors.New("用量事实不合法")
	// ErrModelsRead 表示模型清单读取失败。
	ErrModelsRead = errors.New("模型清单读取失败")
	// ErrModelsUnavailable 表示本端不提供模型清单读取。
	ErrModelsUnavailable = errors.New("本端不提供模型清单读取")
	// ErrMalformedModels 表示服务端给出的模型清单结构不合法。
	ErrMalformedModels = errors.New("模型清单事实不合法")
	// ErrActionsUnavailable 表示没有写登记，账号动作不可用。
	ErrActionsUnavailable = errors.New("账号动作不可用：缺少写登记")
	// ErrResetRequiresAccount 表示未登录时调用重置。
	ErrResetRequiresAccount = errors.New("重置需要已登录账号")
	// ErrResetInvalidType 表示重置类型不合法。
	ErrResetInvalidType = errors.New("重置类型不合法")
	// ErrResetMissingSubscription 表示缺少订阅标识，无法发起重置。
	ErrResetMissingSubscription = errors.New("重置需要已读取的订阅标识")
)
