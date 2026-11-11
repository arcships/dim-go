package app

import (
	"context"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
)

func usageFullReply() map[string]any {
	return map[string]any{
		"subscriptionUrl": "https://example.invalid/console/subscription",
		"pricingUrl":      "https://example.invalid/console/pricing",
		"subscriptionId":  4242,
		"plan": map[string]any{
			"name":             "Pro",
			"status":           "active",
			"priceAmount":      9900,
			"priceCurrency":    "CNY",
			"priceInterval":    "month",
			"expiresAt":        "2026-12-31T00:00:00Z",
			"currentTermStart": "2026-12-01T00:00:00Z",
		},
		"credits": map[string]any{
			"usedUnits":      123,
			"totalUnits":     456,
			"remainingUnits": 333,
			"unlimited":      false,
			"resetsAt":       "2027-01-01T00:00:00Z",
		},
		"windows": []any{map[string]any{
			"durationHours": 5,
			"tokenUsed":     7,
			"tokenCap":      100,
			"startedAt":     "2026-11-01T00:00:00Z",
			"expiresAt":     "2026-12-01T00:00:00Z",
		}},
		"features": []any{
			map[string]any{
				"key": "tokens", "used": 120, "allowance": 1000, "remaining": 880,
				"unit": "tokens", "unlimited": false, "periodEnd": "2026-12-01T00:00:00Z",
			},
			map[string]any{"key": "requests", "used": 3, "allowance": 60, "unit": "requests"},
		},
	}
}

// TestAccountUsageMapsNewFields 核对 plan/credits/windows/features/链接的透传。
func TestAccountUsageMapsNewFields(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "account.providers", Result: []any{map[string]any{
				"providerId": "dimcode-api-oauth", "loginMethods": []any{"browser"},
			}}},
			{Method: "account.usage", Result: usageFullReply()},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	reader := &accountUsageReader{wire: result.base.accountWire}
	usage, err := reader.ReadUsage(context.Background())
	if err != nil {
		t.Fatalf("读取用量: %v", err)
	}
	if usage.SubscriptionURL == "" || usage.PricingURL == "" ||
		!usage.HasSubscriptionID || usage.SubscriptionID != 4242 {
		t.Fatalf("链接或订阅标识映射不对: %+v", usage)
	}
	if usage.Plan == nil || usage.Plan.Name != "Pro" || usage.Plan.Status != "active" ||
		usage.Plan.PriceAmount == nil || *usage.Plan.PriceAmount != 9900 ||
		usage.Plan.PriceCurrency != "CNY" || usage.Plan.PriceInterval != "month" ||
		usage.Plan.ExpiresAt.IsZero() || usage.Plan.CurrentTermStart.IsZero() {
		t.Fatalf("套餐映射不对: %+v", usage.Plan)
	}
	if usage.Credits == nil || usage.Credits.UsedUnits == nil || *usage.Credits.UsedUnits != 123 ||
		usage.Credits.TotalUnits == nil || *usage.Credits.TotalUnits != 456 ||
		usage.Credits.RemainingUnits == nil || *usage.Credits.RemainingUnits != 333 ||
		usage.Credits.Unlimited == nil || *usage.Credits.Unlimited ||
		usage.Credits.ResetsAt.IsZero() {
		t.Fatalf("额度映射不对: %+v", usage.Credits)
	}
	if len(usage.Windows) != 1 || usage.Windows[0].DurationHours != 5 ||
		usage.Windows[0].TokenUsed != 7 || usage.Windows[0].TokenCap != 100 ||
		usage.Windows[0].StartedAt.IsZero() || usage.Windows[0].ExpiresAt.IsZero() {
		t.Fatalf("窗口映射不对: %+v", usage.Windows)
	}
	if len(usage.Features) != 2 {
		t.Fatalf("功能额度条数不对: %+v", usage.Features)
	}
	first := usage.Features[0]
	if first.Key != "tokens" || first.Used == nil || *first.Used != 120 ||
		first.Allowance == nil || *first.Allowance != 1000 ||
		first.Remaining == nil || *first.Remaining != 880 ||
		first.Unit != "tokens" || first.Unlimited == nil || *first.Unlimited ||
		first.PeriodEnd.IsZero() {
		t.Fatalf("功能额度映射不对: %+v", first)
	}
	// 第二项缺 remaining/unlimited/periodEnd：保持 nil，不当 0。
	second := usage.Features[1]
	if second.Remaining != nil || second.Unlimited != nil || !second.PeriodEnd.IsZero() {
		t.Fatalf("缺失字段应保持 nil: %+v", second)
	}
	// 通用视图仍然存在，且不改变原始字段语义。
	if len(usage.Lines) != 3 || !usage.Aggregate.Available || usage.Aggregate.Used != 123 {
		t.Fatalf("通用视图不对: lines=%d aggregate=%+v", len(usage.Lines), usage.Aggregate)
	}
}

// TestAccountUsageMissingFieldsStayNil 核对服务端没给的字段保持 nil。
func TestAccountUsageMissingFieldsStayNil(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "account.providers", Result: []any{map[string]any{
				"providerId": "dimcode-api-oauth", "loginMethods": []any{"browser"},
			}}},
			{Method: "account.usage", Result: map[string]any{}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	usage, err := (&accountUsageReader{wire: result.base.accountWire}).ReadUsage(context.Background())
	if err != nil {
		t.Fatalf("读取用量: %v", err)
	}
	if usage.Plan != nil || usage.Credits != nil || usage.Windows != nil || usage.Features != nil {
		t.Fatalf("缺失字段应为 nil: %+v", usage)
	}
	if usage.HasSubscriptionID || usage.SubscriptionID != 0 ||
		usage.SubscriptionURL != "" || usage.PricingURL != "" {
		t.Fatalf("缺失订阅字段应为零值: %+v", usage)
	}
	if usage.Aggregate.Available || usage.Aggregate.Used != 0 {
		t.Fatalf("没有 credits 时聚合应不可用: %+v", usage.Aggregate)
	}
}

// TestAccountUsageDeepCopiesValues 核对两次读取互不共享内存，修改一次不影响另一次。
func TestAccountUsageDeepCopiesValues(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "account.providers", Result: []any{map[string]any{
				"providerId": "dimcode-api-oauth", "loginMethods": []any{"browser"},
			}}},
			{Method: "account.usage", Result: usageFullReply()},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	reader := &accountUsageReader{wire: result.base.accountWire}
	first, err := reader.ReadUsage(context.Background())
	if err != nil {
		t.Fatalf("第一次读取: %v", err)
	}
	// 改第一次返回值里的指针与切片内容。
	if first.Plan != nil && first.Plan.PriceAmount != nil {
		*first.Plan.PriceAmount = 1
	}
	if first.Credits != nil && first.Credits.UsedUnits != nil {
		*first.Credits.UsedUnits = 0
	}
	if len(first.Features) > 0 {
		first.Features[0].Key = "changed"
		if first.Features[0].Used != nil {
			*first.Features[0].Used = -1
		}
	}
	if len(first.Windows) > 0 {
		first.Windows[0].TokenUsed = -1
	}
	first.SubscriptionURL = "changed"

	second, err := reader.ReadUsage(context.Background())
	if err != nil {
		t.Fatalf("第二次读取: %v", err)
	}
	if second.Plan == nil || second.Plan.PriceAmount == nil || *second.Plan.PriceAmount != 9900 {
		t.Fatalf("第二次读取被第一次修改影响: %+v", second.Plan)
	}
	if second.Credits == nil || second.Credits.UsedUnits == nil || *second.Credits.UsedUnits != 123 {
		t.Fatalf("第二次额度被影响: %+v", second.Credits)
	}
	if len(second.Features) == 0 || second.Features[0].Key != "tokens" ||
		second.Features[0].Used == nil || *second.Features[0].Used != 120 {
		t.Fatalf("第二次功能额度被影响: %+v", second.Features)
	}
	if len(second.Windows) == 0 || second.Windows[0].TokenUsed != 7 {
		t.Fatalf("第二次窗口被影响: %+v", second.Windows)
	}
	if second.SubscriptionURL == "changed" {
		t.Fatalf("第二次链接被影响: %q", second.SubscriptionURL)
	}
	_ = account.Usage{}
}
