package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// 账号领域在服务端上的真实方法名（alpha.21 实测，见 D1 §3.7 与 D6 §6.3）。
const (
	methodAccountProviders = "account.providers"
	methodAccountStatus    = "account.status"
	methodAccountUsage     = "account.usage"
	methodProviderCatalog  = "provider.catalog"
	methodLoginBegin       = "account.login.browser.begin"
	methodLoginComplete    = "account.login.browser.complete"
	methodLoginCancel      = "account.login.cancel"
)

// accountProvider 是一次 provider 解析的结果。
type accountProvider struct {
	ID string
	// Directory 是服务端声明的默认目录；为空时不带该参数。
	Directory string
}

// accountWire 是账号域在控制连接上的读取与登录适配。
//
// provider 只解析一次并缓存；登录尝试到 provider 的对应关系只在内存里保存，
// 用于完成/取消时回传 providerId。这里不持有任何凭据。
type accountWire struct {
	app *App

	mu       sync.Mutex
	provider *accountProvider
	attempts map[string]string
}

func (w *accountWire) accountProvider(ctx context.Context, op string) (accountProvider, error) {
	w.mu.Lock()
	cached := w.provider
	w.mu.Unlock()
	if cached != nil {
		return *cached, nil
	}
	client, err := w.app.requireControl(op)
	if err != nil {
		return accountProvider{}, err
	}
	var providers []struct {
		ProviderID       string   `json:"providerId"`
		DisplayName      string   `json:"displayName"`
		LoginMethods     []string `json:"loginMethods"`
		DefaultDirectory *string  `json:"defaultDirectory"`
	}
	if err := client.CallInto(ctx, methodAccountProviders, nil, &providers); err != nil {
		return accountProvider{}, err
	}
	if len(providers) == 0 {
		return accountProvider{}, failure.New(failure.KindNonRetryable, methodAccountProviders,
			fmt.Errorf("服务端没有声明账号提供者"))
	}
	// 优先选支持浏览器登录的提供者；没有就取第一条，不猜能力。
	picked := providers[0]
	for _, provider := range providers {
		for _, method := range provider.LoginMethods {
			if method == "browser" {
				picked = provider
				break
			}
		}
	}
	resolved := accountProvider{ID: picked.ProviderID}
	if picked.DefaultDirectory != nil {
		resolved.Directory = *picked.DefaultDirectory
	}
	if resolved.ID == "" {
		return accountProvider{}, failure.New(failure.KindNonRetryable, methodAccountProviders,
			fmt.Errorf("账号提供者缺少标识"))
	}
	w.mu.Lock()
	w.provider = &resolved
	w.mu.Unlock()
	return resolved, nil
}

func (w *accountWire) rememberAttempt(attempt, providerID string) {
	if attempt == "" {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.attempts == nil {
		w.attempts = make(map[string]string)
	}
	w.attempts[attempt] = providerID
}

func (w *accountWire) attemptProvider(attempt string) string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.attempts[attempt]
}

func (w *accountWire) forgetAttempt(attempt string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.attempts, attempt)
}

// accountStatusReader 读取账号状态。
type accountStatusReader struct {
	wire *accountWire
}

func (r *accountStatusReader) ReadAccount(ctx context.Context) (account.Fact, error) {
	provider, err := r.wire.accountProvider(ctx, "读取账号状态")
	if err != nil {
		return account.Fact{}, err
	}
	client, err := r.wire.app.requireControl("读取账号状态")
	if err != nil {
		return account.Fact{}, err
	}
	var reply struct {
		CredentialEpoch string          `json:"credentialEpoch"`
		Generation      json.RawMessage `json:"generation"`
		Account         *struct {
			Issuer  string `json:"issuer"`
			Subject string `json:"subject"`
			Email   string `json:"email"`
			Name    string `json:"name"`
			Plan    string `json:"plan"`
		} `json:"account"`
	}
	if err := client.CallInto(ctx, methodAccountStatus, map[string]any{"providerId": provider.ID}, &reply); err != nil {
		return account.Fact{}, err
	}
	// 真实回执没有独立的「过期」字段：没有账号对象就是未登录。
	if reply.Account == nil {
		return account.Fact{Status: account.StatusSignedOut}, nil
	}
	return account.Fact{
		Status: account.StatusSignedIn,
		Identity: account.Identity{
			AccountID:   reply.Account.Subject,
			Email:       reply.Account.Email,
			DisplayName: reply.Account.Name,
			Plan:        reply.Account.Plan,
		},
	}, nil
}

// accountUsageReader 逐项读取用量；聚合项只在服务端给出数值时可用。
type accountUsageReader struct {
	wire *accountWire
}

func (r *accountUsageReader) ReadUsage(ctx context.Context) (account.Usage, error) {
	provider, err := r.wire.accountProvider(ctx, "读取账号用量")
	if err != nil {
		return account.Usage{}, err
	}
	client, err := r.wire.app.requireControl("读取账号用量")
	if err != nil {
		return account.Usage{}, err
	}
	var reply struct {
		Credits *struct {
			UsedUnits  *float64 `json:"usedUnits"`
			TotalUnits *float64 `json:"totalUnits"`
			Unlimited  *bool    `json:"unlimited"`
			ResetsAt   *string  `json:"resetsAt"`
		} `json:"credits"`
		Windows []struct {
			DurationHours float64 `json:"durationHours"`
			TokenUsed     float64 `json:"tokenUsed"`
			TokenCap      float64 `json:"tokenCap"`
			ExpiresAt     *string `json:"expiresAt"`
		} `json:"windows"`
		Features []struct {
			Key       string   `json:"key"`
			Used      *float64 `json:"used"`
			Allowance *float64 `json:"allowance"`
			Unit      *string  `json:"unit"`
			PeriodEnd *string  `json:"periodEnd"`
		} `json:"features"`
		Plan string `json:"plan"`
	}
	if err := client.CallInto(ctx, methodAccountUsage, map[string]any{"providerId": provider.ID}, &reply); err != nil {
		return account.Usage{}, err
	}
	usage := account.Usage{}
	for _, feature := range reply.Features {
		line := account.UsageLine{Key: feature.Key, Label: feature.Key}
		if feature.Used != nil {
			line.Used = *feature.Used
		}
		if feature.Allowance != nil {
			line.Limit = *feature.Allowance
		}
		if feature.Unit != nil {
			line.Unit = *feature.Unit
		}
		line.ResetsAt = wireTime(feature.PeriodEnd)
		usage.Lines = append(usage.Lines, line)
	}
	for _, window := range reply.Windows {
		usage.Lines = append(usage.Lines, account.UsageLine{
			Key:      fmt.Sprintf("window-%gh", window.DurationHours),
			Label:    fmt.Sprintf("最近 %g 小时", window.DurationHours),
			Used:     window.TokenUsed,
			Limit:    window.TokenCap,
			Unit:     "tokens",
			Window:   fmt.Sprintf("%gh", window.DurationHours),
			ResetsAt: wireTime(window.ExpiresAt),
		})
	}
	switch {
	case reply.Credits == nil:
		// 服务端没有给聚合项：保持不可用，由账号领域补原因。
	case reply.Credits.Unlimited != nil && *reply.Credits.Unlimited:
		usage.Aggregate = account.AggregateUsage{Reason: "服务端声明无限额度，没有上限值"}
	case reply.Credits.UsedUnits != nil && reply.Credits.TotalUnits != nil:
		usage.Aggregate = account.AggregateUsage{
			Available: true,
			Used:      *reply.Credits.UsedUnits,
			Limit:     *reply.Credits.TotalUnits,
			Unit:      "units",
		}
	}
	return usage, nil
}

// accountModelReader 读取该账号的模型目录。
type accountModelReader struct {
	wire *accountWire
}

func (r *accountModelReader) ReadModels(ctx context.Context) ([]account.Model, error) {
	provider, err := r.wire.accountProvider(ctx, "读取模型清单")
	if err != nil {
		return nil, err
	}
	client, err := r.wire.app.requireControl("读取模型清单")
	if err != nil {
		return nil, err
	}
	var reply struct {
		Catalog *struct {
			Models []struct {
				ID                string  `json:"id"`
				Name              string  `json:"name"`
				UnavailableReason *string `json:"unavailableReason"`
			} `json:"models"`
		} `json:"catalog"`
	}
	if err := client.CallInto(ctx, methodProviderCatalog, map[string]any{"providerId": provider.ID}, &reply); err != nil {
		return nil, err
	}
	if reply.Catalog == nil {
		return nil, failure.New(failure.KindNonRetryable, methodProviderCatalog,
			fmt.Errorf("服务端没有该账号的模型目录"))
	}
	models := make([]account.Model, 0, len(reply.Catalog.Models))
	for _, entry := range reply.Catalog.Models {
		models = append(models, account.Model{
			ID:          entry.ID,
			DisplayName: entry.Name,
			// 账号目录里的模型都挂在账号下；是否可用以服务端原因为准。
			RequiresAccount: true,
			Unavailable:     entry.UnavailableReason != nil,
		})
	}
	return models, nil
}

// accountAuthSource 是浏览器登录的服务端侧：发起授权、完成交换与取消尝试。
type accountAuthSource struct {
	wire *accountWire
}

func (s *accountAuthSource) BeginAuthorization(
	ctx context.Context,
	redirectURI string,
) (account.AuthorizationRequest, error) {
	provider, err := s.wire.accountProvider(ctx, "发起账号登录")
	if err != nil {
		return account.AuthorizationRequest{}, err
	}
	client, err := s.wire.app.requireControl("发起账号登录")
	if err != nil {
		return account.AuthorizationRequest{}, err
	}
	params := map[string]any{
		"providerId":  provider.ID,
		"redirectUri": redirectURI,
	}
	if provider.Directory != "" {
		params["directory"] = provider.Directory
	}
	var reply struct {
		AttemptID        string          `json:"attemptId"`
		AuthorizationURL string          `json:"authorizationUrl"`
		ExpiresAt        json.RawMessage `json:"expiresAt"`
		State            string          `json:"state"`
	}
	if err := client.CallInto(ctx, methodLoginBegin, params, &reply); err != nil {
		return account.AuthorizationRequest{}, err
	}
	s.wire.rememberAttempt(reply.AttemptID, provider.ID)
	return account.AuthorizationRequest{
		URL:     reply.AuthorizationURL,
		State:   reply.State,
		Attempt: reply.AttemptID,
	}, nil
}

func (s *accountAuthSource) CompleteAuthorization(
	ctx context.Context,
	attempt string,
	result account.AuthorizationResult,
) error {
	providerID := s.wire.attemptProvider(attempt)
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

// CancelAuthorization 取消服务端尚未结束的登录尝试；尽力而为，失败不改变本地状态。
func (s *accountAuthSource) CancelAuthorization(ctx context.Context, attempt string) error {
	providerID := s.wire.attemptProvider(attempt)
	if providerID == "" {
		resolved, err := s.wire.accountProvider(ctx, "取消账号登录")
		if err != nil {
			return err
		}
		providerID = resolved.ID
	}
	client, err := s.wire.app.requireControl("取消账号登录")
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

// wireTime 把服务端的 RFC3339 时间映射成 UTC；解析失败按未给出处理。
func wireTime(raw *string) time.Time {
	if raw == nil {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

// requireControl 返回控制连接；未连接时给出连接缺失分类，调用方据此失败关闭。
func (a *App) requireControl(op string) (*protocol.Client, error) {
	client := a.control()
	if client == nil {
		return nil, failure.New(failure.KindConnectionMissing, op, ErrNotConnected)
	}
	return client, nil
}
