package account

import (
	"context"
	"errors"
	"testing"
)

// TestModelsSignedOutAndUnknownAreEmptyWithReason 确认未登录与状态未知都返回空清单，
// 并且都给出原因，不把空清单当成"没有模型"。
func TestModelsSignedOutAndUnknownAreEmptyWithReason(t *testing.T) {
	reader := &stubModelReader{models: []Model{
		{ID: "m-account", DisplayName: "需要账号", RequiresAccount: true},
		{ID: "m-free", DisplayName: "不需要账号"},
	}}

	unknown := newTestAccount(t, Config{Models: reader})
	if _, err := unknown.RefreshModels(context.Background()); err != nil {
		t.Fatalf("读取模型清单: %v", err)
	}
	list := unknown.Models()
	if len(list.Models) != 0 || list.Reason == "" {
		t.Fatalf("状态未知时应是空清单并给原因：%+v", list)
	}

	signedOut := newTestAccount(t, Config{
		Status: &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Models: reader,
	})
	if _, err := signedOut.Refresh(context.Background()); err != nil {
		t.Fatalf("读取未登录状态: %v", err)
	}
	list, err := signedOut.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型清单: %v", err)
	}
	if len(list.Models) != 0 {
		t.Fatalf("未登录时应返回空清单：%+v", list.Models)
	}
	if list.Reason == "" || !list.Ready {
		t.Fatalf("未登录时应说明原因且标明清单已读取：%+v", list)
	}
}

// TestModelsSignedInFiltersUnavailable 确认已登录时去掉服务端标为不可用的模型，
// 保留其余模型并保持服务端顺序。
func TestModelsSignedInFiltersUnavailable(t *testing.T) {
	reader := &stubModelReader{models: []Model{
		{ID: "m-1", DisplayName: "一号"},
		{ID: "m-2", DisplayName: "二号", Unavailable: true},
		{ID: "m-3", DisplayName: "三号", RequiresAccount: true},
	}}
	account := newTestAccount(t, Config{
		Status: &stubStatusReader{fact: signedInFact()},
		Models: reader,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取已登录状态: %v", err)
	}
	list, err := account.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型清单: %v", err)
	}
	if len(list.Models) != 2 || list.Models[0].ID != "m-1" || list.Models[1].ID != "m-3" {
		t.Fatalf("模型过滤结果不符：%+v", list.Models)
	}
	if list.Reason != "" {
		t.Fatalf("完整清单不应给原因：%q", list.Reason)
	}
}

// TestModelsExpiredHidesAccountModels 确认登录过期时只保留不需要账号的模型，并说明
// 需要账号的模型不可用。
func TestModelsExpiredHidesAccountModels(t *testing.T) {
	reader := &stubModelReader{models: []Model{
		{ID: "m-account", RequiresAccount: true},
		{ID: "m-free"},
	}}
	account := newTestAccount(t, Config{
		Status: &stubStatusReader{fact: Fact{Status: StatusExpired, Identity: Identity{AccountID: "acct-1"}}},
		Models: reader,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取过期状态: %v", err)
	}
	list, err := account.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型清单: %v", err)
	}
	if len(list.Models) != 1 || list.Models[0].ID != "m-free" {
		t.Fatalf("过期时应只保留不需要账号的模型：%+v", list.Models)
	}
	if list.Reason == "" {
		t.Fatal("裁剪清单应说明原因")
	}
}

// TestModelsExpiredWithoutFreeModelsIsEmptyWithReason 确认过期且没有免账号模型时
// 返回空清单并说明原因。
func TestModelsExpiredWithoutFreeModelsIsEmptyWithReason(t *testing.T) {
	reader := &stubModelReader{models: []Model{{ID: "m-account", RequiresAccount: true}}}
	account := newTestAccount(t, Config{
		Status: &stubStatusReader{fact: Fact{Status: StatusExpired, Identity: Identity{AccountID: "acct-1"}}},
		Models: reader,
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取过期状态: %v", err)
	}
	list, err := account.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型清单: %v", err)
	}
	if len(list.Models) != 0 || list.Reason == "" {
		t.Fatalf("空清单应说明原因：%+v", list)
	}
}

// TestModelsWithoutReaderIsUnavailable 确认没有模型清单读取能力时明确不可用。
func TestModelsWithoutReaderIsUnavailable(t *testing.T) {
	account := newTestAccount(t, Config{Status: &stubStatusReader{fact: signedInFact()}})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}
	list, err := account.RefreshModels(context.Background())
	if !errors.Is(err, ErrModelsUnavailable) {
		t.Fatalf("应返回模型清单不可用：%v", err)
	}
	if len(list.Models) != 0 || list.Reason == "" {
		t.Fatalf("不可用时应是空清单并给原因：%+v", list)
	}
}

// TestModelsValidation 确认缺少标识或标识重复的清单整份不采用。
func TestModelsValidation(t *testing.T) {
	cases := []struct {
		name   string
		models []Model
	}{
		{name: "缺标识", models: []Model{{DisplayName: "无名"}}},
		{name: "标识重复", models: []Model{{ID: "m-1"}, {ID: "m-1"}}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			reader := &stubModelReader{models: testCase.models}
			account := newTestAccount(t, Config{
				Status: &stubStatusReader{fact: signedInFact()},
				Models: reader,
			})
			if _, err := account.Refresh(context.Background()); err != nil {
				t.Fatalf("读取账号状态: %v", err)
			}
			if _, err := account.RefreshModels(context.Background()); !errors.Is(err, ErrMalformedModels) {
				t.Fatalf("应返回模型清单不合法：%v", err)
			}
			if list := account.Models(); len(list.Models) != 0 || list.Ready {
				t.Fatalf("不合法清单不应采用：%+v", list)
			}
		})
	}
}

// TestModelsFollowStatusChanges 确认清单随账号状态变化重新过滤：登出清空状态后，
// 原先读到的清单不再作为可用模型返回。
func TestModelsFollowStatusChanges(t *testing.T) {
	status := &stubStatusReader{fact: signedInFact()}
	reader := &stubModelReader{models: []Model{{ID: "m-1"}, {ID: "m-2", RequiresAccount: true}}}
	account := newTestAccount(t, Config{Status: status, Models: reader})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}
	if list, err := account.RefreshModels(context.Background()); err != nil || len(list.Models) != 2 {
		t.Fatalf("已登录时模型清单不符：%+v，%v", list, err)
	}

	status.set(Fact{Status: StatusSignedOut}, nil)
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取未登录状态: %v", err)
	}
	list := account.Models()
	if len(list.Models) != 0 || list.Reason == "" {
		t.Fatalf("未登录后不应再返回可用模型：%+v", list)
	}
}
