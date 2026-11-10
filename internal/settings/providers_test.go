package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/write"
)

const testProviderUUID = "01900000-0000-4000-8000-000000000001"

type stubProviderAdmin struct {
	list        ProviderList
	listErr     error
	listCalls   int
	detail      func(providerID string) (ProviderDetail, error)
	detailCalls int
	probe       ProviderProbeResult
	probeErr    error
	probeCalls  int
	lastProbe   ProviderProbeRequest
}

func (s *stubProviderAdmin) ReadProviders(context.Context) (ProviderList, error) {
	s.listCalls++
	if s.listErr != nil {
		return ProviderList{}, s.listErr
	}
	return s.list, nil
}

func (s *stubProviderAdmin) ReadProviderDetail(_ context.Context, providerID string) (ProviderDetail, error) {
	s.detailCalls++
	if s.detail != nil {
		return s.detail(providerID)
	}
	return s.listDetail(providerID)
}

func (s *stubProviderAdmin) listDetail(providerID string) (ProviderDetail, error) {
	return ProviderDetail{}, errors.New("测试没有准备 provider 详情: " + providerID)
}

func (s *stubProviderAdmin) ProbeProvider(_ context.Context, request ProviderProbeRequest) (ProviderProbeResult, error) {
	s.probeCalls++
	s.lastProbe = request
	if s.probeErr != nil {
		return ProviderProbeResult{}, s.probeErr
	}
	return s.probe, nil
}

func providerFixtureDetail(version int) ProviderDetail {
	enabled := true
	return ProviderDetail{
		ID:          "p1",
		DisplayName: "测试供应商",
		Source:      "custom",
		Version:     &version,
		Enabled:     &enabled,
		Actions:     []string{"update", "models.set_enabled"},
		Connection: &ProviderConnectionInfo{
			DriverKind: "openai-compatible",
			AuthType:   "anonymous",
			BaseURL:    "http://127.0.0.1:9/v1",
		},
		Models: []ProviderModelInfo{
			{ID: "m1", DisplayName: "一号", Status: "active", Enabled: true},
			{ID: "m2", DisplayName: "二号", Status: "active", Enabled: true},
			{ID: "m3", DisplayName: "三号", Status: "beta", Enabled: true},
		},
		Preferences: map[string]ModelPreferences{
			"m1": {ReasoningEffort: "high"},
		},
	}
}

func providerFixtureList() ProviderList {
	return ProviderList{Providers: []ProviderSummary{
		{
			ID: "p1", Source: "custom", DisplayName: "测试供应商", DriverKind: "openai-compatible",
			AuthType: "anonymous", BaseURL: "http://127.0.0.1:9/v1", Enabled: true, Version: 3,
			HeaderNames: []string{"X-Test"},
		},
	}}
}

func newProviderManager(t *testing.T, admin ProviderAdmin, sender *senderFunc) (*Manager, *write.Manager) {
	t.Helper()
	var writer *write.Manager
	if sender != nil {
		writer = newTestWriter(t, sender)
	}
	manager := mustNewManager(t, Config{
		Features:  allFeatures(),
		Providers: admin,
		Writer:    writer,
	})
	return manager, writer
}

func TestRefreshProvidersAndReadDetail(t *testing.T) {
	admin := &stubProviderAdmin{
		list: providerFixtureList(),
		detail: func(string) (ProviderDetail, error) {
			return providerFixtureDetail(3), nil
		},
	}
	manager, _ := newProviderManager(t, admin, nil)

	list, err := manager.RefreshProviders(context.Background())
	if err != nil {
		t.Fatalf("读取 provider 清单: %v", err)
	}
	if len(list.Providers) != 1 || list.Providers[0].ID != "p1" || list.Providers[0].Version != 3 {
		t.Fatalf("provider 清单不符：%+v", list)
	}
	list.Providers[0].HeaderNames[0] = "changed"
	cached, ok := manager.Providers()
	if !ok || cached.Providers[0].HeaderNames[0] != "X-Test" {
		t.Fatalf("清单缓存被调用方改动：%+v", cached)
	}

	detail, err := manager.ReadProvider(context.Background(), "p1")
	if err != nil {
		t.Fatalf("读取 provider 详情: %v", err)
	}
	if detail.Version == nil || *detail.Version != 3 || len(detail.Models) != 3 {
		t.Fatalf("provider 详情不符：%+v", detail)
	}
	detail.Models[0].DisplayName = "changed"
	cachedDetail, ok := manager.Provider("p1")
	if !ok || cachedDetail.Models[0].DisplayName != "一号" {
		t.Fatalf("详情缓存被调用方改动：%+v", cachedDetail)
	}
}

func TestProviderFactsRejectMalformed(t *testing.T) {
	manager, _ := newProviderManager(t, &stubProviderAdmin{
		list: ProviderList{Providers: []ProviderSummary{{ID: "", Version: 1}}},
	}, nil)
	if _, err := manager.RefreshProviders(context.Background()); !errors.Is(err, ErrMalformedProvider) {
		t.Fatalf("清单缺标识应拒绝：%v", err)
	}

	manager, _ = newProviderManager(t, &stubProviderAdmin{
		list: ProviderList{Providers: []ProviderSummary{{ID: "p", Version: 0}}},
	}, nil)
	if _, err := manager.RefreshProviders(context.Background()); !errors.Is(err, ErrMalformedProvider) {
		t.Fatalf("清单版本不合法应拒绝：%v", err)
	}

	bad := providerFixtureDetail(1)
	bad.Models[0].Status = "unknown"
	manager, _ = newProviderManager(t, &stubProviderAdmin{
		detail: func(string) (ProviderDetail, error) { return bad, nil },
	}, nil)
	if _, err := manager.ReadProvider(context.Background(), "p1"); !errors.Is(err, ErrMalformedProvider) {
		t.Fatalf("详情事实不合法应拒绝：%v", err)
	}
}

func TestProbeProviderIsReadOnly(t *testing.T) {
	admin := &stubProviderAdmin{probe: ProviderProbeResult{Models: []ProviderModelInput{
		{ID: "m1", DisplayName: "一号", Status: "active"},
	}}}
	sender := &senderFunc{}
	manager, _ := newProviderManager(t, admin, sender)
	result, err := manager.ProbeProvider(context.Background(), ProviderProbeRequest{
		ProviderID: "p1",
		Connection: &ProviderConnectionInput{
			DriverKind: "openai-compatible",
			Auth:       ProviderAuth{Type: "anonymous"},
			BaseURL:    "http://127.0.0.1:9/v1",
		},
	})
	if err != nil {
		t.Fatalf("探测: %v", err)
	}
	if len(result.Models) != 1 || result.Models[0].ID != "m1" {
		t.Fatalf("探测结果不符：%+v", result)
	}
	if admin.probeCalls != 1 || len(sender.snapshot()) != 0 {
		t.Fatalf("探测不应写配置：probe=%d writes=%d", admin.probeCalls, len(sender.snapshot()))
	}
}

func TestProbeProviderCapabilityAndValidation(t *testing.T) {
	missing := mustNewManager(t, Config{
		Features:  stubFeatures{},
		Providers: &stubProviderAdmin{},
	})
	if _, err := missing.ProbeProvider(context.Background(), ProviderProbeRequest{
		ProviderID: "p1",
		Connection: &ProviderConnectionInput{DriverKind: "openai-compatible", Auth: ProviderAuth{Type: "anonymous"}, BaseURL: "http://x"},
	}); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺探测能力应分类：%v", err)
	}

	admin := &stubProviderAdmin{}
	manager, _ := newProviderManager(t, admin, nil)
	for _, request := range []ProviderProbeRequest{
		{ProviderID: "p1"},
		{ProviderID: "p1", Connection: &ProviderConnectionInput{DriverKind: "openai-compatible", Auth: ProviderAuth{Type: "anonymous"}, BaseURL: "http://x"}, Patch: &ProviderPatch{Enabled: boolPtr(true)}},
		{ProviderID: "p1", Connection: &ProviderConnectionInput{DriverKind: "openai-compatible", Auth: ProviderAuth{Type: "api_key"}, BaseURL: "http://x"}},
	} {
		if _, err := manager.ProbeProvider(context.Background(), request); !errors.Is(err, ErrInvalidProviderRequest) {
			t.Fatalf("不合法探测应拒绝：%+v -> %v", request, err)
		}
	}
	if admin.probeCalls != 0 {
		t.Fatal("不合法探测不应发请求")
	}
}

func TestCreateProviderWritesOnceAndReadsBack(t *testing.T) {
	admin := &stubProviderAdmin{
		detail: func(string) (ProviderDetail, error) { return providerFixtureDetail(1), nil },
	}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{Result: json.RawMessage(`{"ok":true}`)}, nil
	}}
	manager, _ := newProviderManager(t, admin, sender)

	detail, err := manager.CreateProvider(context.Background(), ProviderCreateRequest{
		ID:          testProviderUUID,
		DisplayName: "测试供应商",
		Connection: ProviderConnectionInput{
			DriverKind: "openai-compatible",
			Auth:       ProviderAuth{Type: "anonymous"},
			BaseURL:    "http://127.0.0.1:9/v1",
			Headers:    map[string]string{"X-Test": "1"},
			Credential: &ProviderCredential{Action: "set_ref", Reference: "ENV_KEY"},
		},
		Models: []ProviderModelInput{{
			ID: "m1", DisplayName: "一号", Status: "active",
			ContextWindow: int64Ptr(1000), MaxOutputTokens: int64Ptr(100),
		}},
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("新建 provider: %v", err)
	}
	if detail.ID != testProviderUUID || detail.Version == nil || *detail.Version != 1 {
		t.Fatalf("新建后详情不符：%+v", detail)
	}
	requests := sender.snapshot()
	if len(requests) != 1 || requests[0].Kind != write.KindManagementWrite {
		t.Fatalf("新建应经管理写入一次：%+v", requests)
	}
	params := decodeParams[providerCreateParams](t, requests[0].Params)
	if params.Op != "provider.create" || params.ProviderID != testProviderUUID || !params.Enabled {
		t.Fatalf("新建参数不符：%+v", params)
	}
	if params.Connection.Credential == nil || params.Connection.Credential.Reference != "ENV_KEY" {
		t.Fatalf("显式凭据引用未下发：%+v", params.Connection.Credential)
	}
	if len(params.Models) != 1 || params.Models[0].ModelID != "m1" {
		t.Fatalf("新建模型参数不符：%+v", params.Models)
	}
	if admin.detailCalls != 1 {
		t.Fatalf("新建后应重读一次详情：%d", admin.detailCalls)
	}
}

func TestCreateProviderValidationAndCapability(t *testing.T) {
	admin := &stubProviderAdmin{}
	sender := &senderFunc{}
	manager, _ := newProviderManager(t, admin, sender)
	if _, err := manager.CreateProvider(context.Background(), ProviderCreateRequest{
		ID: testProviderUUID, DisplayName: "x",
		Connection: ProviderConnectionInput{DriverKind: "openai-compatible", Auth: ProviderAuth{Type: "api_key"}, BaseURL: "http://x"},
	}); !errors.Is(err, ErrInvalidProviderRequest) {
		t.Fatalf("不合法新建应拒绝：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("不合法新建不应发请求")
	}

	missing := mustNewManager(t, Config{
		Features:  stubFeatures{FeatureProviderManagement: true},
		Providers: admin,
		Writer:    newTestWriter(t, sender),
	})
	if _, err := missing.CreateProvider(context.Background(), ProviderCreateRequest{
		ID: testProviderUUID, DisplayName: "x",
		Connection: ProviderConnectionInput{DriverKind: "openai-compatible", Auth: ProviderAuth{Type: "anonymous"}, BaseURL: "http://x"},
	}); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺新建能力应分类：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("缺能力不应发请求")
	}
}

func TestUpdateProviderLocalConflictDoesNotWrite(t *testing.T) {
	admin := &stubProviderAdmin{detail: func(string) (ProviderDetail, error) { return providerFixtureDetail(3), nil }}
	sender := &senderFunc{}
	manager, _ := newProviderManager(t, admin, sender)
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("预读详情: %v", err)
	}
	display := "新名字"
	if _, err := manager.UpdateProvider(context.Background(), ProviderUpdateRequest{
		ProviderID: "p1", ExpectedVersion: 2, Patch: ProviderPatch{DisplayName: &display},
	}); !errors.Is(err, ErrProviderConflict) {
		t.Fatalf("本地版本不符应冲突：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("本地冲突不应发请求")
	}
}

func TestUpdateProviderServerConflictReadsOnce(t *testing.T) {
	version := 1
	admin := &stubProviderAdmin{detail: func(string) (ProviderDetail, error) { return providerFixtureDetail(version), nil }}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		version = 2
		return write.Reply{}, errors.New("provider 版本已变")
	}}
	manager, _ := newProviderManager(t, admin, sender)
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("预读详情: %v", err)
	}
	display := "新名字"
	if _, err := manager.UpdateProvider(context.Background(), ProviderUpdateRequest{
		ProviderID: "p1", ExpectedVersion: 1, Patch: ProviderPatch{DisplayName: &display},
	}); !errors.Is(err, ErrProviderConflict) {
		t.Fatalf("服务端版本冲突应分类：%v", err)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatalf("冲突不应重发：%d", len(sender.snapshot()))
	}
	if admin.detailCalls != 2 {
		t.Fatalf("冲突判定应只读一次当前详情：%d", admin.detailCalls)
	}
}

func TestUpdateProviderFailureKeepsReason(t *testing.T) {
	admin := &stubProviderAdmin{detail: func(string) (ProviderDetail, error) { return providerFixtureDetail(1), nil }}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{}, errors.New("参数错误")
	}}
	manager, _ := newProviderManager(t, admin, sender)
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("预读详情: %v", err)
	}
	display := "新名字"
	_, err := manager.UpdateProvider(context.Background(), ProviderUpdateRequest{
		ProviderID: "p1", ExpectedVersion: 1, Patch: ProviderPatch{DisplayName: &display},
	})
	if !errors.Is(err, ErrProviderWriteFailed) {
		t.Fatalf("写入失败应分类：%v", err)
	}
	if !strings.Contains(err.Error(), "参数错误") {
		t.Fatalf("失败原因应保留：%v", err)
	}
}

func TestSetProviderEnabledAndModels(t *testing.T) {
	version := 1
	admin := &stubProviderAdmin{detail: func(string) (ProviderDetail, error) { return providerFixtureDetail(version), nil }}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		version++
		return write.Reply{Result: json.RawMessage(`{"ok":true}`)}, nil
	}}
	manager, _ := newProviderManager(t, admin, sender)
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("预读详情: %v", err)
	}
	if _, err := manager.SetProviderEnabled(context.Background(), "p1", 1, false); err != nil {
		t.Fatalf("设置启用状态: %v", err)
	}
	params := decodeParams[providerUpdateParams](t, sender.snapshot()[0].Params)
	if params.Op != "provider.update" || params.Patch.Enabled == nil || *params.Patch.Enabled {
		t.Fatalf("启用状态参数不符：%+v", params)
	}

	version = 2
	admin.detailCalls = 0
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("重读详情: %v", err)
	}
	if _, err := manager.SetModelsEnabled(context.Background(), "p1", 2, []string{"m1", "m3"}); err != nil {
		t.Fatalf("设置启用模型: %v", err)
	}
	enabledParams := decodeParams[providerModelsEnabledParams](t, sender.snapshot()[1].Params)
	if enabledParams.Op != "provider.models.set_enabled" || len(enabledParams.ModelIDs) != 2 || enabledParams.ModelIDs[1] != "m3" {
		t.Fatalf("启用模型参数不符：%+v", enabledParams)
	}
}

func TestSetModelEnabledToggles(t *testing.T) {
	version := 1
	admin := &stubProviderAdmin{detail: func(string) (ProviderDetail, error) { return providerFixtureDetail(version), nil }}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		version++
		return write.Reply{Result: json.RawMessage(`{"ok":true}`)}, nil
	}}
	manager, _ := newProviderManager(t, admin, sender)
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("预读详情: %v", err)
	}
	// 停用已启用的 m2；启用集合应为 m1、m3。
	if _, err := manager.SetModelEnabled(context.Background(), "p1", "m2", 1, false); err != nil {
		t.Fatalf("停用模型: %v", err)
	}
	params := decodeParams[providerModelsEnabledParams](t, sender.snapshot()[0].Params)
	if len(params.ModelIDs) != 2 || params.ModelIDs[0] != "m1" || params.ModelIDs[1] != "m3" {
		t.Fatalf("停用后的启用集合不符：%+v", params.ModelIDs)
	}

	// 已处于目标状态：不再写。
	version = 2
	admin.detailCalls = 0
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("重读详情: %v", err)
	}
	if _, err := manager.SetModelEnabled(context.Background(), "p1", "m1", 2, true); err != nil {
		t.Fatalf("重复启用: %v", err)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatalf("目标状态相同不应写入：%d", len(sender.snapshot()))
	}
}

func TestEditResetAndPreferencesWrites(t *testing.T) {
	version := 1
	admin := &stubProviderAdmin{detail: func(string) (ProviderDetail, error) { return providerFixtureDetail(version), nil }}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		version++
		return write.Reply{Result: json.RawMessage(`{"ok":true}`)}, nil
	}}
	manager, _ := newProviderManager(t, admin, sender)
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("预读详情: %v", err)
	}

	display := "新一号"
	if _, err := manager.EditModel(context.Background(), ProviderModelEditRequest{
		ProviderID: "p1", ModelID: "m1", ExpectedVersion: 1,
		Edit: ProviderModelEdit{DisplayName: &display},
	}); err != nil {
		t.Fatalf("编辑模型: %v", err)
	}
	editParams := decodeParams[providerModelEditParams](t, sender.snapshot()[0].Params)
	if editParams.Op != "provider.models.edit" || editParams.ModelID != "m1" || editParams.Edit.DisplayName == nil || *editParams.Edit.DisplayName != "新一号" {
		t.Fatalf("编辑模型参数不符：%+v", editParams)
	}

	version = 2
	admin.detailCalls = 0
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("重读详情: %v", err)
	}
	if _, err := manager.ResetModel(context.Background(), "p1", "m1", 2); err != nil {
		t.Fatalf("重置模型: %v", err)
	}
	resetParams := decodeParams[providerModelResetParams](t, sender.snapshot()[1].Params)
	if resetParams.Op != "provider.models.reset" || resetParams.ModelID != "m1" {
		t.Fatalf("重置模型参数不符：%+v", resetParams)
	}

	version = 3
	admin.detailCalls = 0
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("重读详情: %v", err)
	}
	if _, err := manager.UpdateModelPreferences(context.Background(), ModelPreferenceRequest{
		ProviderID: "p1", ModelID: "m1", ExpectedVersion: 3,
		Patch: ModelPreferencePatch{ReasoningEffort: &PreferenceUpdate{Action: "set", Value: "low"}},
	}); err != nil {
		t.Fatalf("写入偏好: %v", err)
	}
	prefParams := decodeParams[providerPreferencesParams](t, sender.snapshot()[2].Params)
	if prefParams.Op != "provider.preferences.update" || prefParams.Patch.ReasoningEffort == nil || prefParams.Patch.ReasoningEffort.Value != "low" {
		t.Fatalf("偏好参数不符：%+v", prefParams)
	}
}

func TestProviderWritesRequireCapabilities(t *testing.T) {
	admin := &stubProviderAdmin{detail: func(string) (ProviderDetail, error) { return providerFixtureDetail(1), nil }}
	sender := &senderFunc{}
	manager := mustNewManager(t, Config{
		Features:  stubFeatures{FeatureProviderManagement: true},
		Providers: admin,
		Writer:    newTestWriter(t, sender),
	})
	display := "x"
	operations := []func() error{
		func() error {
			_, err := manager.CreateProvider(context.Background(), ProviderCreateRequest{
				ID: testProviderUUID, DisplayName: "x",
				Connection: ProviderConnectionInput{DriverKind: "openai-compatible", Auth: ProviderAuth{Type: "anonymous"}, BaseURL: "http://x"},
			})
			return err
		},
		func() error {
			_, err := manager.UpdateProvider(context.Background(), ProviderUpdateRequest{ProviderID: "p1", ExpectedVersion: 1, Patch: ProviderPatch{DisplayName: &display}})
			return err
		},
		func() error {
			_, err := manager.SetModelsEnabled(context.Background(), "p1", 1, []string{"m1"})
			return err
		},
		func() error {
			_, err := manager.EditModel(context.Background(), ProviderModelEditRequest{ProviderID: "p1", ModelID: "m1", ExpectedVersion: 1, Edit: ProviderModelEdit{DisplayName: &display}})
			return err
		},
		func() error {
			_, err := manager.ResetModel(context.Background(), "p1", "m1", 1)
			return err
		},
		func() error {
			_, err := manager.UpdateModelPreferences(context.Background(), ModelPreferenceRequest{ProviderID: "p1", ModelID: "m1", ExpectedVersion: 1, Patch: ModelPreferencePatch{ServiceTier: &PreferenceUpdate{Action: "clear"}}})
			return err
		},
	}
	for index, operation := range operations {
		if err := operation(); !errors.Is(err, ErrCapabilityMissing) {
			t.Fatalf("第 %d 个写操作缺能力应分类：%v", index, err)
		}
	}
	if len(sender.snapshot()) != 0 {
		t.Fatalf("缺能力不应发请求：%d", len(sender.snapshot()))
	}
}

func TestProviderPatchFieldsAndCredentialsAreExplicit(t *testing.T) {
	version := 1
	admin := &stubProviderAdmin{detail: func(string) (ProviderDetail, error) { return providerFixtureDetail(version), nil }}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		version++
		return write.Reply{Result: json.RawMessage(`{"ok":true}`)}, nil
	}}
	manager, _ := newProviderManager(t, admin, sender)
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("预读详情: %v", err)
	}
	display := "改名"
	base := "http://127.0.0.1:10/v1"
	enabled := false
	if _, err := manager.UpdateProvider(context.Background(), ProviderUpdateRequest{
		ProviderID: "p1", ExpectedVersion: 1,
		Patch: ProviderPatch{
			DisplayName: &display,
			BaseURL:     &base,
			Enabled:     &enabled,
			Credential:  &ProviderCredentialUpdate{Action: "set", Value: "explicit-secret"},
			Headers:     []ProviderHeaderUpdate{{Action: "set", Name: "X-Test", Value: "1"}, {Action: "remove", Name: "X-Old"}},
		},
	}); err != nil {
		t.Fatalf("更新 provider: %v", err)
	}
	params := decodeParams[providerUpdateParams](t, sender.snapshot()[0].Params)
	if params.Patch.DisplayName == nil || *params.Patch.DisplayName != "改名" ||
		params.Patch.BaseURL == nil || *params.Patch.BaseURL != base ||
		params.Patch.Enabled == nil || *params.Patch.Enabled ||
		params.Patch.Credential == nil || params.Patch.Credential.Action != "set" || params.Patch.Credential.Value != "explicit-secret" {
		t.Fatalf("显式补丁字段不符：%+v", params.Patch)
	}
	if len(params.Patch.Headers) != 2 || params.Patch.Headers[0].Name != "X-Test" || params.Patch.Headers[1].Action != "remove" {
		t.Fatalf("显式请求头不符：%+v", params.Patch.Headers)
	}
	// 详情类型没有凭据字段；只保留是否已配置。
	detail, ok := manager.Provider("p1")
	if !ok || detail.Connection == nil || detail.Connection.CredentialConfigured {
		t.Fatalf("详情不应带凭据：%+v", detail.Connection)
	}
}

func TestProviderPatchValidation(t *testing.T) {
	admin := &stubProviderAdmin{detail: func(string) (ProviderDetail, error) { return providerFixtureDetail(1), nil }}
	sender := &senderFunc{}
	manager, _ := newProviderManager(t, admin, sender)
	empty := ""
	badPatch := []ProviderPatch{
		{},
		{DisplayName: &empty},
		{Credential: &ProviderCredentialUpdate{Action: "set"}},
		{Credential: &ProviderCredentialUpdate{Action: "unknown"}},
		{Headers: []ProviderHeaderUpdate{{Action: "set", Name: "X", Value: ""}}},
		{Headers: []ProviderHeaderUpdate{{Action: "remove", Name: "X", Value: "v"}}},
		{Headers: []ProviderHeaderUpdate{{Action: "set", Name: "X", Value: "1"}, {Action: "remove", Name: "X"}}},
	}
	for _, patch := range badPatch {
		if _, err := manager.UpdateProvider(context.Background(), ProviderUpdateRequest{ProviderID: "p1", ExpectedVersion: 1, Patch: patch}); !errors.Is(err, ErrInvalidProviderRequest) {
			t.Fatalf("不合法补丁应拒绝：%+v -> %v", patch, err)
		}
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("不合法补丁不应发请求")
	}

	good := ProviderPatch{Credential: &ProviderCredentialUpdate{Action: "clear"}}
	if _, err := manager.UpdateProvider(context.Background(), ProviderUpdateRequest{ProviderID: "p1", ExpectedVersion: 1, Patch: good}); !errors.Is(err, ErrProviderWriteFailed) && !errors.Is(err, ErrProviderConflict) {
		// 未 stub detail 的写后重读会失败；这里只要求不是参数错误。
		if errors.Is(err, ErrInvalidProviderRequest) {
			t.Fatalf("clear 凭据不应被参数校验拒绝：%v", err)
		}
	}
}

func TestModelEditAndPreferenceFields(t *testing.T) {
	version := 1
	admin := &stubProviderAdmin{detail: func(string) (ProviderDetail, error) { return providerFixtureDetail(version), nil }}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		version++
		return write.Reply{Result: json.RawMessage(`{"ok":true}`)}, nil
	}}
	manager, _ := newProviderManager(t, admin, sender)
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("预读详情: %v", err)
	}
	display := "编辑后"
	contextWindow := int64(4096)
	maxOutput := int64(512)
	tool := true
	vision := true
	modalities := []string{"text", "image"}
	reasoning := &ProviderModelReasoning{Supported: true, EffortOptions: []string{"low", "high"}}
	if _, err := manager.EditModel(context.Background(), ProviderModelEditRequest{
		ProviderID: "p1", ModelID: "m1", ExpectedVersion: 1,
		Edit: ProviderModelEdit{
			DisplayName: &display, ContextWindow: &contextWindow, MaxOutputTokens: &maxOutput,
			ToolCalling: &tool, Vision: &vision, ModalityCapabilities: &modalities, Reasoning: reasoning,
		},
	}); err != nil {
		t.Fatalf("编辑模型: %v", err)
	}
	params := decodeParams[providerModelEditParams](t, sender.snapshot()[0].Params)
	if params.Edit.DisplayName == nil || *params.Edit.DisplayName != display ||
		params.Edit.ContextWindow == nil || *params.Edit.ContextWindow != contextWindow ||
		params.Edit.MaxOutputTokens == nil || *params.Edit.MaxOutputTokens != maxOutput ||
		params.Edit.ToolCalling == nil || !*params.Edit.ToolCalling ||
		params.Edit.Vision == nil || !*params.Edit.Vision ||
		params.Edit.ModalityCapabilities == nil || len(*params.Edit.ModalityCapabilities) != 2 ||
		params.Edit.Reasoning == nil || !params.Edit.Reasoning.Supported {
		t.Fatalf("模型编辑字段不符：%+v", params.Edit)
	}

	version = 2
	admin.detailCalls = 0
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("重读详情: %v", err)
	}
	if _, err := manager.UpdateModelPreferences(context.Background(), ModelPreferenceRequest{
		ProviderID: "p1", ModelID: "m1", ExpectedVersion: 2,
		Patch: ModelPreferencePatch{
			ReasoningEffort: &PreferenceUpdate{Action: "clear"},
			ServiceTier:     &PreferenceUpdate{Action: "set", Value: "priority"},
		},
	}); err != nil {
		t.Fatalf("偏好写入: %v", err)
	}
	pref := decodeParams[providerPreferencesParams](t, sender.snapshot()[1].Params)
	if pref.Patch.ReasoningEffort == nil || pref.Patch.ReasoningEffort.Action != "clear" ||
		pref.Patch.ServiceTier == nil || pref.Patch.ServiceTier.Value != "priority" {
		t.Fatalf("偏好字段不符：%+v", pref.Patch)
	}
}

func TestProviderCreateRequiresUUIDAndManagedNotEditable(t *testing.T) {
	admin := &stubProviderAdmin{}
	sender := &senderFunc{}
	manager, _ := newProviderManager(t, admin, sender)
	if _, err := manager.CreateProvider(context.Background(), ProviderCreateRequest{
		ID: "not-a-uuid", DisplayName: "x",
		Connection: ProviderConnectionInput{DriverKind: "openai-compatible", Auth: ProviderAuth{Type: "anonymous"}, BaseURL: "http://x"},
	}); !errors.Is(err, ErrInvalidProviderRequest) {
		t.Fatalf("非 UUID 新建应拒绝：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("非 UUID 新建不应发请求")
	}

	managed := providerFixtureDetail(5)
	managed.Version = nil
	admin.detail = func(string) (ProviderDetail, error) { return managed, nil }
	if _, err := manager.ReadProvider(context.Background(), "p1"); err != nil {
		t.Fatalf("读取托管 provider: %v", err)
	}
	if _, err := manager.SetModelEnabled(context.Background(), "p1", "m1", 5, false); !errors.Is(err, ErrProviderNotEditable) {
		t.Fatalf("无版本 provider 应拒绝单模型写入：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("无版本 provider 不应发请求")
	}
}

func TestProviderReadCapabilities(t *testing.T) {
	manager := mustNewManager(t, Config{Features: stubFeatures{}})
	if _, err := manager.RefreshProviders(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺 provider.list 应分类：%v", err)
	}
	if _, err := manager.ReadProvider(context.Background(), "p1"); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺 provider.management 应分类：%v", err)
	}
}
