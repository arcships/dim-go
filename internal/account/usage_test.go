package account

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestUsageItemsAndUnavailableAggregate 确认用量逐项一致；服务端没有聚合项时明确
// 不可用，不用逐项数据求和冒充。
func TestUsageItemsAndUnavailableAggregate(t *testing.T) {
	resetsAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	reader := &stubUsageReader{usage: Usage{
		Lines: []UsageLine{
			{Key: "tokens", Label: "令牌", Used: 1234, Limit: 5000, Unit: "tokens", Window: "day", ResetsAt: resetsAt},
			{Key: "requests", Label: "请求", Used: 7, Limit: 100, Unit: "requests", Window: "minute"},
		},
		Aggregate: AggregateUsage{Available: false},
	}}
	account := newTestAccount(t, Config{Usage: reader})

	usage, err := account.RefreshUsage(context.Background())
	if err != nil {
		t.Fatalf("读取用量: %v", err)
	}
	if !usage.Ready {
		t.Fatal("读取成功后用量应是可用状态")
	}
	if len(usage.Lines) != 2 {
		t.Fatalf("用量项条数不符：%+v", usage.Lines)
	}
	first := usage.Lines[0]
	if first.Key != "tokens" || first.Used != 1234 || first.Limit != 5000 ||
		first.Unit != "tokens" || first.Window != "day" || !first.ResetsAt.Equal(resetsAt) {
		t.Fatalf("用量项没有逐项保留：%+v", first)
	}
	if usage.Aggregate.Available {
		t.Fatalf("服务端没有聚合项时不应可用：%+v", usage.Aggregate)
	}
	if usage.Aggregate.Used != 0 || usage.Aggregate.Limit != 0 {
		t.Fatalf("聚合不可用时不应有数值：%+v", usage.Aggregate)
	}
	if usage.Aggregate.Reason == "" {
		t.Fatal("聚合不可用时应给原因")
	}
	if usage.ObservedAt.IsZero() {
		t.Fatal("读取成功后应有观察时间")
	}
	if usage.ReadError != "" {
		t.Fatalf("读取成功不应有错误：%q", usage.ReadError)
	}
}

// TestUsageAggregateUsesServerValues 确认聚合项只采用服务端给的值。
func TestUsageAggregateUsesServerValues(t *testing.T) {
	reader := &stubUsageReader{usage: Usage{
		Lines: []UsageLine{
			{Key: "tokens", Used: 1, Limit: 10},
			{Key: "requests", Used: 2, Limit: 20},
		},
		Aggregate: AggregateUsage{Available: true, Used: 3, Limit: 30, Unit: "units"},
	}}
	account := newTestAccount(t, Config{Usage: reader})

	usage, err := account.RefreshUsage(context.Background())
	if err != nil {
		t.Fatalf("读取用量: %v", err)
	}
	if !usage.Aggregate.Available || usage.Aggregate.Used != 3 || usage.Aggregate.Limit != 30 ||
		usage.Aggregate.Unit != "units" || usage.Aggregate.Reason != "" {
		t.Fatalf("聚合项应采用服务端值：%+v", usage.Aggregate)
	}
}

// TestUsageReadFailureKeepsConfirmedData 确认读取失败只记录错误，不把上一次确认的
// 逐项数据改成零值。
func TestUsageReadFailureKeepsConfirmedData(t *testing.T) {
	reader := &stubUsageReader{usage: Usage{
		Lines: []UsageLine{{Key: "tokens", Used: 42, Limit: 100}},
	}}
	account := newTestAccount(t, Config{Usage: reader})
	first, err := account.RefreshUsage(context.Background())
	if err != nil {
		t.Fatalf("读取用量: %v", err)
	}

	reader.set(Usage{}, errors.New("连接中断"))
	usage, err := account.RefreshUsage(context.Background())
	if !errors.Is(err, ErrUsageRead) {
		t.Fatalf("应返回用量读取失败：%v", err)
	}
	if !usage.Ready || len(usage.Lines) != 1 || usage.Lines[0].Used != 42 {
		t.Fatalf("读取失败不应清空已确认用量：%+v", usage)
	}
	if !usage.ObservedAt.Equal(first.ObservedAt) {
		t.Fatalf("读取失败不应推进观察时间：got %v，want %v", usage.ObservedAt, first.ObservedAt)
	}
	if usage.ReadError == "" {
		t.Fatal("读取失败应记录原因")
	}
}

// TestUsageWithoutReaderIsUnavailable 确认没有用量读取能力时明确不可用，而不是空用量。
func TestUsageWithoutReaderIsUnavailable(t *testing.T) {
	account := newTestAccount(t, Config{})
	usage, err := account.RefreshUsage(context.Background())
	if !errors.Is(err, ErrUsageUnavailable) {
		t.Fatalf("应返回用量不可用：%v", err)
	}
	if usage.Ready || len(usage.Lines) != 0 {
		t.Fatalf("不可用时不应有数据：%+v", usage)
	}
	if usage.ReadError == "" {
		t.Fatal("不可用时应说明原因")
	}
}

// TestUsageLineValidation 确认缺少标识的用量项按不合法处理，不静默丢弃或补标识。
func TestUsageLineValidation(t *testing.T) {
	reader := &stubUsageReader{usage: Usage{Lines: []UsageLine{{Used: 1}}}}
	account := newTestAccount(t, Config{Usage: reader})

	_, err := account.RefreshUsage(context.Background())
	if !errors.Is(err, ErrMalformedUsage) {
		t.Fatalf("应返回用量不合法：%v", err)
	}
	if usage := account.Usage(); usage.Ready || len(usage.Lines) != 0 {
		t.Fatalf("不合法用量不应采用：%+v", usage)
	}
}

// TestUsageOptionalFieldsPreserveMissing 确认可选数值/布尔字段用 nil 表示缺失，
// 0 与 false 是服务端明确给出的值。
func TestUsageOptionalFieldsPreserveMissing(t *testing.T) {
	zero := 0.0
	unlimited := false
	reader := &stubUsageReader{usage: Usage{
		Plan: &UsagePlan{PriceAmount: &zero},
		Features: []UsageFeature{
			{Key: "tokens", Used: &zero, Allowance: nil, Remaining: nil, Unlimited: &unlimited},
		},
		Credits: &UsageCredits{UsedUnits: &zero, Unlimited: nil},
	}}
	account := newTestAccount(t, Config{Usage: reader})
	usage, err := account.RefreshUsage(context.Background())
	if err != nil {
		t.Fatalf("读取用量: %v", err)
	}
	if usage.Plan == nil || usage.Plan.PriceAmount == nil || *usage.Plan.PriceAmount != 0 {
		t.Fatalf("0 价格应保留为明确值：%+v", usage.Plan)
	}
	if usage.Features[0].Allowance != nil || usage.Features[0].Remaining != nil {
		t.Fatalf("缺失字段应保持 nil：%+v", usage.Features[0])
	}
	if usage.Features[0].Unlimited == nil || *usage.Features[0].Unlimited {
		t.Fatalf("false 应保留为明确值：%+v", usage.Features[0].Unlimited)
	}
	if usage.Credits == nil || usage.Credits.Unlimited != nil {
		t.Fatalf("缺失布尔应保持 nil：%+v", usage.Credits)
	}
}

// TestUsageCloneDeepCopiesOptionalFields 确认 Snapshot/Usage 副本不共享指针与切片。
func TestUsageCloneDeepCopiesOptionalFields(t *testing.T) {
	price := 9900.0
	used := 1.0
	remaining := 2.0
	unlimited := false
	reader := &stubUsageReader{usage: Usage{
		Plan:    &UsagePlan{Name: "Pro", PriceAmount: &price},
		Credits: &UsageCredits{UsedUnits: &used, RemainingUnits: &remaining, Unlimited: &unlimited},
		Windows: []UsageWindow{{DurationHours: 5, TokenUsed: 1}},
		Features: []UsageFeature{{
			Key: "tokens", Used: &used, Remaining: &remaining, Unlimited: &unlimited,
		}},
	}}
	account := newTestAccount(t, Config{Usage: reader})
	if _, err := account.RefreshUsage(context.Background()); err != nil {
		t.Fatalf("读取用量: %v", err)
	}

	snapshot := account.Usage()
	*snapshot.Plan.PriceAmount = 1
	*snapshot.Credits.UsedUnits = 9
	*snapshot.Credits.RemainingUnits = 8
	*snapshot.Credits.Unlimited = true
	*snapshot.Features[0].Used = 7
	*snapshot.Features[0].Remaining = 6
	*snapshot.Features[0].Unlimited = true
	snapshot.Windows[0].TokenUsed = 5

	current := account.Usage()
	if *current.Plan.PriceAmount != 9900 || *current.Credits.UsedUnits != 1 ||
		*current.Credits.RemainingUnits != 2 || *current.Credits.Unlimited ||
		*current.Features[0].Used != 1 || *current.Features[0].Remaining != 2 ||
		*current.Features[0].Unlimited || current.Windows[0].TokenUsed != 1 {
		t.Fatalf("用量副本共享了底层数据：%+v", current)
	}
}

// TestUsageIsDetachedFromReader 确认读取替身返回的切片被复制，不共享底层数组。
func TestUsageIsDetachedFromReader(t *testing.T) {
	lines := []UsageLine{{Key: "tokens", Used: 1}}
	reader := &stubUsageReader{usage: Usage{Lines: lines}}
	account := newTestAccount(t, Config{Usage: reader})
	if _, err := account.RefreshUsage(context.Background()); err != nil {
		t.Fatalf("读取用量: %v", err)
	}
	lines[0].Used = 99
	if usage := account.Usage(); usage.Lines[0].Used != 1 {
		t.Fatalf("用量与读取来源共享了切片：%+v", usage)
	}
}
