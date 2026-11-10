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

// Usage 是一次用量读取的结果。Ready 表示 Lines/Aggregate 来自一次成功的服务端读取。
// ReadError 非空表示最近一次读取失败；此时 Ready 与数据仍是上一次确认的事实。
type Usage struct {
	Ready      bool
	Lines      []UsageLine
	Aggregate  AggregateUsage
	ObservedAt time.Time
	ReadError  string
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
