package account

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// UsageLine 是服务端给出的一项用量。逐项读取、逐项呈现，本包不改写数值，也不
// 把多项相加。
type UsageLine struct {
	// Key 是服务端给出的稳定标识，必填。
	Key string
	// Label 是展示名；空表示服务端没有给出。
	Label string
	// Used 与 Limit 是这一项的已用与上限；Unit 是单位。含义由服务端定义。
	Used  float64
	Limit float64
	Unit  string
	// Window 是统计窗口（例如 "day"、"month"）；空表示服务端没有给出。
	Window string
	// ResetsAt 是这一项的重置时间；零值表示服务端没有给出。
	ResetsAt time.Time
}

// AggregateUsage 是聚合用量。它只能由服务端给出：Available 为假时 Used/Limit/Unit
// 一律没有意义，调用方不得当成零值展示，也不得用逐项数据自行求和。
type AggregateUsage struct {
	Available bool
	Used      float64
	Limit     float64
	Unit      string
	// Reason 说明聚合项为什么不可用。
	Reason string
}

// UsagePlan 是账号套餐信息。数值字段用指针表示「服务端没有给」，缺失不等于 0。
// PriceAmount 是货币最小单位（例如分），本包不换算货币。
type UsagePlan struct {
	Name             string
	Status           string
	PriceAmount      *float64
	PriceCurrency    string
	PriceInterval    string
	ExpiresAt        time.Time
	CurrentTermStart time.Time
}

// UsageCredits 是套餐额度单位。它不是货币；Unlimited 为 true 时余额 0 不代表耗尽。
type UsageCredits struct {
	UsedUnits      *float64
	TotalUnits     *float64
	RemainingUnits *float64
	Unlimited      *bool
	ResetsAt       time.Time
}

// UsageWindow 是一个滚动窗口的 token 配额与用量。
type UsageWindow struct {
	DurationHours float64
	TokenUsed     float64
	TokenCap      float64
	StartedAt     time.Time
	ExpiresAt     time.Time
}

// UsageFeature 是一项功能额度。数值字段用指针表示「服务端没有给」，Unit 由服务端
// 给出（如 call），本包不推导单位或换算。
type UsageFeature struct {
	Key       string
	Used      *float64
	Allowance *float64
	Remaining *float64
	Unit      string
	Unlimited *bool
	PeriodEnd time.Time
}

// Usage 是一次用量读取的结果。Ready 表示数据来自一次成功的服务端读取；
// ReadError 非空表示最近一次读取失败，此时 Ready 与数据仍是上一次确认的事实。
//
// Lines/Aggregate 是逐项与聚合的通用视图；Plan/Credits/Windows/Features 保留
// 服务端原始字段与可选语义。SubscriptionURL/PricingURL 是链接，不是金额。
type Usage struct {
	Ready     bool
	Lines     []UsageLine
	Aggregate AggregateUsage
	// Plan/Credits 为 nil 表示服务端没有给出对应对象。
	Plan    *UsagePlan
	Credits *UsageCredits
	// Windows/Features 为 nil 表示服务端没有给出该数组；非 nil 空切片表示明确给了空数组。
	Windows  []UsageWindow
	Features []UsageFeature
	// SubscriptionURL / PricingURL 是管理页与价格页链接，不是金额。
	SubscriptionURL string
	PricingURL      string
	// SubscriptionID 是服务端给出的订阅标识；重置额度必须带它。
	SubscriptionID    int64
	HasSubscriptionID bool
	ObservedAt        time.Time
	ReadError         string
}

// ResetType 是重置额度消耗的类型，与原客户端一致。
type ResetType string

const (
	// ResetTypeWindow 重置当前滚动窗口。
	ResetTypeWindow ResetType = "window"
	// ResetTypeMonthlyFull 重置整月额度。
	ResetTypeMonthlyFull ResetType = "monthly_full"
)

func (t ResetType) valid() bool {
	return t == ResetTypeWindow || t == ResetTypeMonthlyFull
}

// RefreshUsage 逐项读取用量。聚合项由服务端给出：没有给出时保持 Available=false
// 并带原因，不用逐项数据求和。
//
// 读取失败时不改写已确认的数据，只记录 ReadError 并返回错误。
func (a *Account) RefreshUsage(ctx context.Context) (Usage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if a.cfg.Usage == nil {
		a.mu.Lock()
		a.usage.ReadError = ErrUsageUnavailable.Error()
		usage := cloneUsage(a.usage)
		a.mu.Unlock()
		return usage, ErrUsageUnavailable
	}

	usage, err := a.cfg.Usage.ReadUsage(ctx)
	if err != nil {
		a.mu.Lock()
		a.usage.ReadError = err.Error()
		current := cloneUsage(a.usage)
		a.mu.Unlock()
		return current, fmt.Errorf("%w: %v", ErrUsageRead, err)
	}
	if err := validateUsageLines(usage.Lines); err != nil {
		a.mu.Lock()
		a.usage.ReadError = err.Error()
		current := cloneUsage(a.usage)
		a.mu.Unlock()
		return current, err
	}

	now := a.cfg.Now()
	usage.Lines = append([]UsageLine(nil), usage.Lines...)
	usage.ObservedAt = now
	usage.Ready = true
	usage.ReadError = ""
	if !usage.Aggregate.Available && strings.TrimSpace(usage.Aggregate.Reason) == "" {
		usage.Aggregate.Reason = "服务端未提供聚合用量"
	}
	if usage.Aggregate.Available {
		usage.Aggregate.Reason = ""
	}

	a.mu.Lock()
	a.usage = usage
	current := cloneUsage(a.usage)
	a.mu.Unlock()
	return current, nil
}

// validateUsageLines 只校验结构：缺少标识的项无法逐项呈现，按服务端事实不合法处理。
func validateUsageLines(lines []UsageLine) error {
	for _, line := range lines {
		if strings.TrimSpace(line.Key) == "" {
			return fmt.Errorf("%w: 用量项缺少标识", ErrMalformedUsage)
		}
	}
	return nil
}
