package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/write"
)

func modelCatalog() ModelCatalog {
	return ModelCatalog{
		Providers: []Provider{
			{
				ID:      "local",
				Name:    "本地",
				Enabled: true,
				Models: []ProviderModel{
					{ID: "m1", Name: "一号"},
					{ID: "m2", Name: "二号"},
				},
				EnabledModelIDs: []string{"m1", "m2"},
			},
			{
				ID:      "cloud",
				Name:    "云端",
				Enabled: true,
				Models: []ProviderModel{
					{ID: "c1", Name: "云端一号"},
					{ID: "c2", Name: "云端二号"},
				},
				EnabledModelIDs: []string{"c1"},
			},
			{
				ID:      "off",
				Name:    "停用",
				Enabled: false,
				Models: []ProviderModel{
					{ID: "x1", Name: "停用一号"},
				},
				EnabledModelIDs: []string{"x1"},
			},
		},
		Selection: &Selection{ProviderID: "cloud", ModelID: "c1", Revision: "sel-1", ReasoningEffort: "high"},
	}
}

func readyAccount(ids ...string) account.ModelList {
	models := make([]account.Model, 0, len(ids))
	for _, id := range ids {
		models = append(models, account.Model{ID: id})
	}
	return account.ModelList{Models: models, Ready: true}
}

func TestRefreshModelsFiltersByAccount(t *testing.T) {
	catalog := &stubCatalog{value: modelCatalog()}
	accountSource := &stubAccount{list: readyAccount("m1", "c1")}
	manager := mustNewManager(t, Config{Models: catalog, Account: accountSource})

	snapshot, err := manager.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型目录: %v", err)
	}
	if !snapshot.AvailabilityReady {
		t.Fatalf("可用性应来自一次成功读取：%+v", snapshot)
	}
	selectable := map[string]bool{}
	for _, provider := range snapshot.Providers {
		for _, model := range provider.Models {
			selectable[provider.ID+"/"+model.ID] = model.Selectable
		}
	}
	want := map[string]bool{
		"local/m1": true,
		"local/m2": false,
		"cloud/c1": true,
		"cloud/c2": false,
		"off/x1":   false,
	}
	for key, expected := range want {
		if selectable[key] != expected {
			t.Fatalf("%s 可选性不符：got %v，want %v（%+v）", key, selectable[key], expected, snapshot.Providers)
		}
	}
	if snapshot.Selection == nil || !snapshot.Selection.Known || !snapshot.Selection.Selectable {
		t.Fatalf("当前选择应可选：%+v", snapshot.Selection)
	}
	if snapshot.Selection.ReasoningEffort != "high" {
		t.Fatalf("当前选择偏好不符：%+v", snapshot.Selection)
	}
	if catalog.calls != 1 || accountSource.calls != 1 {
		t.Fatalf("读取次数不符：catalog=%d account=%d", catalog.calls, accountSource.calls)
	}
}

func TestRefreshModelsAvailabilityNotReady(t *testing.T) {
	manager := mustNewManager(t, Config{
		Models:  &stubCatalog{value: modelCatalog()},
		Account: &stubAccount{list: account.ModelList{Ready: false, Reason: "账号状态未知：不能确定可用模型"}},
	})
	snapshot, err := manager.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型目录: %v", err)
	}
	if snapshot.AvailabilityReady || snapshot.Reason != "账号状态未知：不能确定可用模型" {
		t.Fatalf("可用性不可用时应带原因：%+v", snapshot)
	}
	for _, provider := range snapshot.Providers {
		for _, model := range provider.Models {
			if model.Selectable {
				t.Fatalf("可用性未确认时不能标成可选：%s/%s", provider.ID, model.ID)
			}
			if model.Reason == "" {
				t.Fatalf("不可选应说明原因：%s/%s", provider.ID, model.ID)
			}
		}
	}
	if snapshot.Selection == nil || snapshot.Selection.Selectable || snapshot.Selection.Reason == "" {
		t.Fatalf("不可确认的选择应有原因：%+v", snapshot.Selection)
	}
}

func TestRefreshModelsSelectionEdgeCases(t *testing.T) {
	catalog := modelCatalog()
	catalog.Selection = &Selection{ProviderID: "cloud", ModelID: "c2"}
	manager := mustNewManager(t, Config{
		Models:  &stubCatalog{value: catalog},
		Account: &stubAccount{list: readyAccount("c2")},
	})
	snapshot, err := manager.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型目录: %v", err)
	}
	if snapshot.Selection == nil || !snapshot.Selection.Known || snapshot.Selection.Selectable {
		t.Fatalf("模型未启用时选择应不可选：%+v", snapshot.Selection)
	}
	if snapshot.Selection.Reason != "模型未启用" {
		t.Fatalf("选择不可选原因不符：%+v", snapshot.Selection)
	}

	catalog.Selection = &Selection{ProviderID: "missing", ModelID: "m1"}
	manager = mustNewManager(t, Config{
		Models:  &stubCatalog{value: catalog},
		Account: &stubAccount{list: readyAccount("m1")},
	})
	snapshot, err = manager.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型目录: %v", err)
	}
	if snapshot.Selection == nil || snapshot.Selection.Known || snapshot.Selection.Reason == "" {
		t.Fatalf("目录里不存在的选择应标为无法确认：%+v", snapshot.Selection)
	}
}

func TestRefreshModelsCapabilityErrors(t *testing.T) {
	manager := mustNewManager(t, Config{
		Features: stubFeatures{},
		Models:   &stubCatalog{value: modelCatalog()},
		Account:  &stubAccount{list: readyAccount("m1")},
	})
	if _, err := manager.RefreshModels(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺目录能力应分类：%v", err)
	}

	manager = mustNewManager(t, Config{Models: &stubCatalog{value: modelCatalog()}})
	if _, err := manager.RefreshModels(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺账号可用模型视图应分类：%v", err)
	}

	manager = mustNewManager(t, Config{
		Models:  &stubCatalog{err: errors.New("连接中断")},
		Account: &stubAccount{list: readyAccount("m1")},
	})
	if _, err := manager.RefreshModels(context.Background()); !errors.Is(err, ErrModelsRead) {
		t.Fatalf("目录读取失败应分类：%v", err)
	}
}

func TestRefreshModelsRejectsMalformedCatalog(t *testing.T) {
	base := func() ModelCatalog {
		return ModelCatalog{Providers: []Provider{{
			ID:              "p",
			Enabled:         true,
			Models:          []ProviderModel{{ID: "m"}},
			EnabledModelIDs: []string{"m"},
		}}}
	}
	cases := []struct {
		name   string
		mutate func(ModelCatalog) ModelCatalog
	}{
		{"provider 缺标识", func(c ModelCatalog) ModelCatalog { c.Providers[0].ID = " "; return c }},
		{"provider 重复", func(c ModelCatalog) ModelCatalog { c.Providers = append(c.Providers, c.Providers[0]); return c }},
		{"模型缺标识", func(c ModelCatalog) ModelCatalog { c.Providers[0].Models[0].ID = ""; return c }},
		{"模型重复", func(c ModelCatalog) ModelCatalog {
			c.Providers[0].Models = append(c.Providers[0].Models, c.Providers[0].Models[0])
			return c
		}},
		{"启用模型不存在", func(c ModelCatalog) ModelCatalog {
			c.Providers[0].EnabledModelIDs = []string{"other"}
			return c
		}},
		{"选择缺 provider", func(c ModelCatalog) ModelCatalog { c.Selection = &Selection{ModelID: "m"}; return c }},
		{"选择缺模型", func(c ModelCatalog) ModelCatalog { c.Selection = &Selection{ProviderID: "p"}; return c }},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			manager := mustNewManager(t, Config{
				Models:  &stubCatalog{value: testCase.mutate(base())},
				Account: &stubAccount{list: readyAccount("m")},
			})
			if _, err := manager.RefreshModels(context.Background()); !errors.Is(err, ErrMalformedModels) {
				t.Fatalf("应拒绝不合法目录：%v", err)
			}
		})
	}
}

func TestModelsSnapshotIsDetached(t *testing.T) {
	manager := mustNewManager(t, Config{
		Models:  &stubCatalog{value: modelCatalog()},
		Account: &stubAccount{list: readyAccount("m1")},
	})
	snapshot, err := manager.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("读取模型目录: %v", err)
	}
	snapshot.Providers[0].Models[0].Selectable = false
	snapshot.Catalog.Providers[0].Name = "改过"
	cached, ok := manager.Models()
	if !ok {
		t.Fatal("应有模型缓存")
	}
	if !cached.Providers[0].Models[0].Selectable || cached.Catalog.Providers[0].Name == "改过" {
		t.Fatalf("缓存被调用方改动了：%+v", cached)
	}
}

func TestRefreshStorageIsReadOnlyReport(t *testing.T) {
	reader := &stubStorage{value: StorageReading{
		DatabaseBytes:    int64Ptr(4096),
		WALBytes:         int64Ptr(512),
		ReclaimableBytes: int64Ptr(128),
	}}
	manager := mustNewManager(t, Config{Storage: reader})
	report, err := manager.RefreshStorage(context.Background())
	if err != nil {
		t.Fatalf("读取存储: %v", err)
	}
	if report.DatabaseBytes != 4096 || report.WALBytes != 512 || report.ReclaimableBytes != 128 {
		t.Fatalf("存储读数不符：%+v", report)
	}
	if reader.calls != 1 {
		t.Fatalf("应只读一次：%d", reader.calls)
	}
	cached, ok := manager.Storage()
	if !ok || cached != report {
		t.Fatalf("存储缓存不符：%+v", cached)
	}
}

func TestRefreshStorageRejectsMissingOrNegative(t *testing.T) {
	cases := []struct {
		name    string
		reading StorageReading
	}{
		{"缺数据库占用", StorageReading{WALBytes: int64Ptr(1), ReclaimableBytes: int64Ptr(1)}},
		{"缺 WAL", StorageReading{DatabaseBytes: int64Ptr(1), ReclaimableBytes: int64Ptr(1)}},
		{"缺可回收", StorageReading{DatabaseBytes: int64Ptr(1), WALBytes: int64Ptr(1)}},
		{"负数", StorageReading{DatabaseBytes: int64Ptr(-1), WALBytes: int64Ptr(1), ReclaimableBytes: int64Ptr(1)}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			manager := mustNewManager(t, Config{Storage: &stubStorage{value: testCase.reading}})
			if _, err := manager.RefreshStorage(context.Background()); !errors.Is(err, ErrMalformedStorage) {
				t.Fatalf("应拒绝不合法读数：%v", err)
			}
			if _, ok := manager.Storage(); ok {
				t.Fatal("不合法读数不应缓存")
			}
		})
	}
}

func TestStorageCapabilityAndReadFailure(t *testing.T) {
	manager := mustNewManager(t, Config{Features: stubFeatures{}})
	if _, err := manager.RefreshStorage(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺存储读数能力应分类：%v", err)
	}

	manager = mustNewManager(t, Config{Storage: &stubStorage{err: rpcError(-32000, "storage_unavailable", "存储不可用")}})
	if _, err := manager.RefreshStorage(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("storage_unavailable 应归为缺口：%v", err)
	}

	manager = mustNewManager(t, Config{Storage: &stubStorage{err: errors.New("连接中断")}})
	if _, err := manager.RefreshStorage(context.Background()); !errors.Is(err, ErrStorageRead) {
		t.Fatalf("读取失败应分类：%v", err)
	}
}

func runtimeFixture() []RuntimeEntry {
	return []RuntimeEntry{
		{Name: "node", Source: RuntimeSourceDownload, Auto: false, Status: RuntimeStatusOff},
		{Name: "uv", Source: RuntimeSourceDownload, Auto: true, Version: "0.4.0", Path: "/rt/uv", Status: RuntimeStatusReady},
	}
}

func TestRefreshRuntimesAndLookup(t *testing.T) {
	reader := &stubRuntimes{value: runtimeFixture()}
	manager := mustNewManager(t, Config{Runtimes: reader})
	entries, err := manager.RefreshRuntimes(context.Background())
	if err != nil {
		t.Fatalf("读取运行组件: %v", err)
	}
	if len(entries) != 2 || entries[1].Name != "uv" || entries[1].Status != RuntimeStatusReady {
		t.Fatalf("运行组件清单不符：%+v", entries)
	}
	entry, ok := manager.Runtime("uv")
	if !ok || entry.Version != "0.4.0" {
		t.Fatalf("单个运行组件不符：%+v", entry)
	}
	if _, ok := manager.Runtime("missing"); ok {
		t.Fatal("不存在的运行组件不应返回")
	}
}

func TestRefreshRuntimesRejectsMalformed(t *testing.T) {
	cases := []struct {
		name    string
		entries []RuntimeEntry
	}{
		{"缺名称", []RuntimeEntry{{Source: RuntimeSourceDownload, Status: RuntimeStatusOff}}},
		{"重名", []RuntimeEntry{
			{Name: "node", Source: RuntimeSourceDownload, Status: RuntimeStatusOff},
			{Name: "node", Source: RuntimeSourceSystem, Status: RuntimeStatusReady},
		}},
		{"未知来源", []RuntimeEntry{{Name: "node", Source: "bundle", Status: RuntimeStatusOff}}},
		{"未知状态", []RuntimeEntry{{Name: "node", Source: RuntimeSourceDownload, Status: "installing"}}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			manager := mustNewManager(t, Config{Runtimes: &stubRuntimes{value: testCase.entries}})
			if _, err := manager.RefreshRuntimes(context.Background()); !errors.Is(err, ErrMalformedRuntimes) {
				t.Fatalf("应拒绝不合法清单：%v", err)
			}
		})
	}
}

func TestInstallRuntimeRelaysThroughWrite(t *testing.T) {
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{Result: json.RawMessage(`{"name":"uv","source":"download","auto":true,"status":"ready"}`)}, nil
	}}
	writer := newTestWriter(t, sender)
	reader := &stubRuntimes{value: []RuntimeEntry{
		{Name: "uv", Source: RuntimeSourceDownload, Auto: true, Status: RuntimeStatusReady, Version: "0.4.0"},
	}}
	manager := mustNewManager(t, Config{
		Features: allFeatures(),
		Runtimes: reader,
		Writer:   writer,
	})

	entry, err := manager.InstallRuntime(context.Background(), "uv")
	if err != nil {
		t.Fatalf("安装运行组件: %v", err)
	}
	if entry.Status != RuntimeStatusReady || entry.Version != "0.4.0" {
		t.Fatalf("安装后状态不符：%+v", entry)
	}
	if reader.calls != 1 {
		t.Fatalf("安装成功后应重读一次清单：%d", reader.calls)
	}
	requests := sender.snapshot()
	if len(requests) != 1 {
		t.Fatalf("安装应只转达一次：%d", len(requests))
	}
	if requests[0].Kind != write.KindManagementWrite {
		t.Fatalf("安装应经管理写入类型：%q", requests[0].Kind)
	}
	params := decodeParams[runtimeInstallParams](t, requests[0].Params)
	if params.Op != "runtime.install" || params.Name != "uv" {
		t.Fatalf("安装参数不符：%+v", params)
	}
}

func TestInstallRuntimeFailureAndCapabilities(t *testing.T) {
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{}, errors.New("组件包校验失败")
	}}
	writer := newTestWriter(t, sender)
	manager := mustNewManager(t, Config{
		Features: allFeatures(),
		Runtimes: &stubRuntimes{value: runtimeFixture()},
		Writer:   writer,
	})
	if _, err := manager.InstallRuntime(context.Background(), "uv"); !errors.Is(err, ErrInstallFailed) {
		t.Fatalf("安装明确失败应分类：%v", err)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatal("安装失败不应重发")
	}

	if _, err := manager.InstallRuntime(context.Background(), " "); !errors.Is(err, ErrInvalidRuntimeName) {
		t.Fatalf("空名称应拒绝：%v", err)
	}

	noWriter := mustNewManager(t, Config{Features: allFeatures(), Runtimes: &stubRuntimes{value: runtimeFixture()}})
	if _, err := noWriter.InstallRuntime(context.Background(), "uv"); !errors.Is(err, ErrWriteUnavailable) {
		t.Fatalf("缺写登记应返回写不可用：%v", err)
	}

	noFeature := mustNewManager(t, Config{
		Features: stubFeatures{},
		Runtimes: &stubRuntimes{value: runtimeFixture()},
		Writer:   writer,
	})
	if _, err := noFeature.InstallRuntime(context.Background(), "uv"); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺安装能力应分类：%v", err)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatal("缺能力时不应发出写请求")
	}
}

func TestInstallRuntimeMissingEntryAfterSuccess(t *testing.T) {
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{Result: json.RawMessage(`{"name":"uv","status":"ready"}`)}, nil
	}}
	writer := newTestWriter(t, sender)
	manager := mustNewManager(t, Config{
		Features: allFeatures(),
		Runtimes: &stubRuntimes{value: runtimeFixture()},
		Writer:   writer,
	})
	if _, err := manager.InstallRuntime(context.Background(), "python"); !errors.Is(err, ErrRuntimeEntryMissing) {
		t.Fatalf("安装成功但清单缺失应拒绝：%v", err)
	}
}

func modalityFixture() ModalityCapabilities {
	return ModalityCapabilities{
		Supported:         true,
		CanConfigure:      true,
		TaskSupported:     boolPtr(true),
		RealtimeSupported: boolPtr(false),
		Revision:          stringPtr("rev-1"),
		Categories: []ModalityCategory{
			{
				Category:  "image",
				Available: true,
				Operations: []ModalityOperation{
					{
						CapabilityID: "image.generate",
						Category:     "image",
						Operation:    "generate",
						Available:    false,
						Reasons: []ModalityReason{
							{Dimension: "configuration", Detail: "missing provider"},
						},
						Default: &ModalityOption{
							ProviderID: "p1",
							ModelID:    "m1",
							Available:  false,
							Reason:     &ModalityReason{Dimension: "runtime", Detail: "not installed"},
						},
						Effective: &ModalityModel{ProviderID: "p1", ModelID: "m1", Source: "default"},
						Options: []ModalityOption{
							{ProviderID: "p1", ModelID: "m1", Available: false},
						},
					},
				},
			},
		},
	}
}

func TestRefreshModalityReadsStateAndReasons(t *testing.T) {
	reader := &stubModality{value: modalityFixture()}
	manager := mustNewManager(t, Config{Modality: reader})
	caps, err := manager.RefreshModality(context.Background())
	if err != nil {
		t.Fatalf("读取模态能力: %v", err)
	}
	if !caps.Supported || !caps.CanConfigure || caps.TaskSupported == nil || !*caps.TaskSupported {
		t.Fatalf("模态能力状态不符：%+v", caps)
	}
	if caps.RealtimeSupported == nil || *caps.RealtimeSupported {
		t.Fatalf("实时传输支持位应为显式 false：%+v", caps.RealtimeSupported)
	}
	if caps.Revision == nil || *caps.Revision != "rev-1" {
		t.Fatalf("版本不符：%+v", caps.Revision)
	}
	if len(caps.Categories) != 1 || len(caps.Categories[0].Operations) != 1 {
		t.Fatalf("模态类别不符：%+v", caps.Categories)
	}
	operation := caps.Categories[0].Operations[0]
	if operation.Available || len(operation.Reasons) != 1 || operation.Reasons[0].Dimension != "configuration" {
		t.Fatalf("模态操作原因不符：%+v", operation)
	}
	if operation.Default == nil || operation.Default.Reason == nil || operation.Default.Reason.Detail != "not installed" {
		t.Fatalf("默认选项原因不符：%+v", operation.Default)
	}

	// 缓存是副本。
	caps.Categories[0].Operations[0].Reasons[0].Dimension = "changed"
	cached, ok := manager.Modality()
	if !ok || cached.Categories[0].Operations[0].Reasons[0].Dimension != "configuration" {
		t.Fatalf("缓存被调用方改动了：%+v", cached)
	}
}

func TestRefreshModalityUnsupportedIsNotAnError(t *testing.T) {
	manager := mustNewManager(t, Config{Modality: &stubModality{value: ModalityCapabilities{
		Supported:         false,
		UnavailableReason: "服务端未提供模态能力",
	}}})
	caps, err := manager.RefreshModality(context.Background())
	if err != nil {
		t.Fatalf("服务端声明不支持不是能力缺失错误：%v", err)
	}
	if caps.Supported || caps.UnavailableReason == "" {
		t.Fatalf("不支持状态应带原因：%+v", caps)
	}
}

func TestRefreshModalityCapabilityAndMalformed(t *testing.T) {
	manager := mustNewManager(t, Config{Features: stubFeatures{}})
	if _, err := manager.RefreshModality(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺模态能力应分类：%v", err)
	}

	manager = mustNewManager(t, Config{Modality: &stubModality{err: rpcError(-32601, "unknown_method", "no modality")}})
	if _, err := manager.RefreshModality(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("unknown_method 应归为能力缺失：%v", err)
	}

	cases := []struct {
		name string
		caps ModalityCapabilities
	}{
		{"类别缺名称", ModalityCapabilities{Categories: []ModalityCategory{{}}}},
		{"类别重复", ModalityCapabilities{Categories: []ModalityCategory{{Category: "image"}, {Category: "image"}}}},
		{"能力缺标识", ModalityCapabilities{Categories: []ModalityCategory{{Category: "image", Operations: []ModalityOperation{{Operation: "generate"}}}}}},
		{"能力重复", ModalityCapabilities{Categories: []ModalityCategory{{Category: "image", Operations: []ModalityOperation{
			{CapabilityID: "image.generate", Category: "image", Operation: "generate"},
			{CapabilityID: "image.generate", Category: "image", Operation: "generate"},
		}}}}},
		{"原因缺维度", ModalityCapabilities{Categories: []ModalityCategory{{Category: "image", Reasons: []ModalityReason{{Detail: "x"}}}}}},
		{"有效模型来源未知", ModalityCapabilities{Categories: []ModalityCategory{{Category: "image", Operations: []ModalityOperation{{
			CapabilityID: "image.generate", Category: "image", Operation: "generate",
			Effective: &ModalityModel{ProviderID: "p", ModelID: "m", Source: "guessed"},
		}}}}}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			manager := mustNewManager(t, Config{Modality: &stubModality{value: testCase.caps}})
			if _, err := manager.RefreshModality(context.Background()); !errors.Is(err, ErrMalformedModality) {
				t.Fatalf("应拒绝不合法事实：%v", err)
			}
		})
	}
}

func TestCapabilityErrorsShareOneClassification(t *testing.T) {
	manager := mustNewManager(t, Config{})
	checks := []struct {
		name string
		err  error
	}{
		{"sources", errOf(manager.RefreshSources(context.Background()))},
		{"projects", errOf(manager.RefreshProjects(context.Background()))},
		{"models", errOf(manager.RefreshModels(context.Background()))},
		{"storage", errOf(manager.RefreshStorage(context.Background()))},
		{"runtimes", errOf(manager.RefreshRuntimes(context.Background()))},
		{"modality", errOf(manager.RefreshModality(context.Background()))},
	}
	for _, check := range checks {
		var capability *CapabilityError
		if !errors.As(check.err, &capability) || capability.Capability == "" {
			t.Fatalf("%s 的缺失应带能力名：%v", check.name, check.err)
		}
		if !errors.Is(check.err, ErrCapabilityMissing) {
			t.Fatalf("%s 的缺失应归到共同分类：%v", check.name, check.err)
		}
	}
}

func errOf(_ interface{}, err error) error { return err }

// mutableCatalog 是一个可变的模型目录替身：写成功后由测试更新选择，模拟服务端
// 在两次读取之间改变事实。
type mutableCatalog struct {
	mu      sync.Mutex
	catalog ModelCatalog
	calls   int
	err     error
}

func (m *mutableCatalog) ReadModelCatalog(context.Context) (ModelCatalog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.err != nil {
		return ModelCatalog{}, m.err
	}
	return cloneCatalog(m.catalog), nil
}

func (m *mutableCatalog) selectModel(providerID, modelID, revision string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.catalog.Selection = &Selection{
		ProviderID: providerID,
		ModelID:    modelID,
		Revision:   revision,
	}
}

func (m *mutableCatalog) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func newSelectionManager(t *testing.T, catalog *mutableCatalog, accountSource AccountAvailability, sender *senderFunc) (*Manager, *write.Manager) {
	t.Helper()
	var writer *write.Manager
	if sender != nil {
		writer = newTestWriter(t, sender)
	}
	manager := mustNewManager(t, Config{
		Features: allFeatures(),
		Models:   catalog,
		Account:  accountSource,
		Writer:   writer,
	})
	return manager, writer
}

func TestSelectModelWritesAndRefreshes(t *testing.T) {
	catalog := &mutableCatalog{catalog: modelCatalog()}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		catalog.selectModel("cloud", "c1", "sel-2")
		return write.Reply{Result: json.RawMessage(`{"ok":true}`)}, nil
	}}
	manager, _ := newSelectionManager(t, catalog, &stubAccount{list: readyAccount("m1", "c1")}, sender)
	if _, err := manager.RefreshModels(context.Background()); err != nil {
		t.Fatalf("预读模型目录: %v", err)
	}

	snapshot, err := manager.SelectModel(context.Background(), SelectModelRequest{
		ProviderID:       "cloud",
		ModelID:          "c1",
		ExpectedRevision: "sel-1",
	})
	if err != nil {
		t.Fatalf("写入模型选择: %v", err)
	}
	if snapshot.Selection == nil || snapshot.Selection.ProviderID != "cloud" ||
		snapshot.Selection.ModelID != "c1" || snapshot.Selection.Revision != "sel-2" {
		t.Fatalf("选择写入后应回填服务端事实：%+v", snapshot.Selection)
	}
	if !snapshot.Selection.Selectable {
		t.Fatalf("回填的选择应可选：%+v", snapshot.Selection)
	}
	requests := sender.snapshot()
	if len(requests) != 1 {
		t.Fatalf("选择写入应只发一次：%d", len(requests))
	}
	if requests[0].Kind != write.KindManagementWrite {
		t.Fatalf("选择写入应经管理写入类型：%q", requests[0].Kind)
	}
	params := decodeParams[selectionWriteParams](t, requests[0].Params)
	if params.Op != "provider.select" || params.ProviderID != "cloud" ||
		params.ModelID != "c1" || params.ExpectedRevision != "sel-1" {
		t.Fatalf("选择写入参数不符：%+v", params)
	}
	if catalog.callCount() != 2 {
		t.Fatalf("成功写入后应重读一次目录：%d", catalog.callCount())
	}
	cached, ok := manager.Models()
	if !ok || cached.Selection == nil || cached.Selection.Revision != "sel-2" {
		t.Fatalf("成功写入应回填缓存：%+v", cached.Selection)
	}
}

func TestSelectModelLocalConflictDoesNotWrite(t *testing.T) {
	catalog := &mutableCatalog{catalog: modelCatalog()}
	catalog.selectModel("cloud", "c1", "sel-2")
	sender := &senderFunc{}
	manager, _ := newSelectionManager(t, catalog, &stubAccount{list: readyAccount("c1")}, sender)
	if _, err := manager.RefreshModels(context.Background()); err != nil {
		t.Fatalf("预读模型目录: %v", err)
	}

	if _, err := manager.SelectModel(context.Background(), SelectModelRequest{
		ProviderID:       "cloud",
		ModelID:          "c1",
		ExpectedRevision: "sel-1",
	}); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("本地版本已变应返回冲突：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("本地冲突时不应发出写请求")
	}
	if catalog.callCount() != 1 {
		t.Fatalf("本地冲突不应再读目录：%d", catalog.callCount())
	}
}

func TestSelectModelServerConflictReadsOnce(t *testing.T) {
	catalog := &mutableCatalog{catalog: modelCatalog()}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		// 模拟另一个客户端在写入前改了选择。
		catalog.selectModel("cloud", "c1", "sel-2")
		return write.Reply{}, rpcError(-32000, "revision_conflict", "选择版本已变")
	}}
	manager, _ := newSelectionManager(t, catalog, &stubAccount{list: readyAccount("c1")}, sender)
	if _, err := manager.RefreshModels(context.Background()); err != nil {
		t.Fatalf("预读模型目录: %v", err)
	}

	if _, err := manager.SelectModel(context.Background(), SelectModelRequest{
		ProviderID:       "cloud",
		ModelID:          "c1",
		ExpectedRevision: "sel-1",
	}); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("服务端版本冲突应返回冲突：%v", err)
	}
	requests := sender.snapshot()
	if len(requests) != 1 {
		t.Fatalf("冲突不应重发：%d", len(requests))
	}
	if catalog.callCount() != 2 {
		t.Fatalf("冲突判定只应读一次当前目录：%d", catalog.callCount())
	}
	cached, _ := manager.Models()
	if cached.Selection == nil || cached.Selection.Revision != "sel-2" {
		t.Fatalf("冲突后应采用服务端当前选择：%+v", cached.Selection)
	}
}

func TestSelectModelRejectsUnavailable(t *testing.T) {
	cases := []struct {
		name       string
		account    AccountAvailability
		providerID string
		modelID    string
		contains   string
	}{
		{
			name:       "未登录",
			account:    &stubAccount{list: account.ModelList{Ready: true, Reason: "未登录：可用模型需要账号"}},
			providerID: "cloud",
			modelID:    "c1",
			contains:   "未登录",
		},
		{
			name:       "目标模型不在账号清单",
			account:    &stubAccount{list: readyAccount("m1")},
			providerID: "cloud",
			modelID:    "c1",
			contains:   "当前账号状态不包含该模型",
		},
		{
			name:       "目录里没有该模型",
			account:    &stubAccount{list: readyAccount("m1", "c1")},
			providerID: "cloud",
			modelID:    "missing",
			contains:   "模型目录里没有",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			catalog := &mutableCatalog{catalog: modelCatalog()}
			sender := &senderFunc{}
			manager, _ := newSelectionManager(t, catalog, testCase.account, sender)
			if _, err := manager.RefreshModels(context.Background()); err != nil {
				t.Fatalf("预读模型目录: %v", err)
			}
			_, err := manager.SelectModel(context.Background(), SelectModelRequest{
				ProviderID:       testCase.providerID,
				ModelID:          testCase.modelID,
				ExpectedRevision: "sel-1",
			})
			if !errors.Is(err, ErrSelectionUnavailable) {
				t.Fatalf("不可写应返回选择不可用：%v", err)
			}
			var unavailable *SelectionUnavailableError
			if !errors.As(err, &unavailable) || !strings.Contains(unavailable.Reason, testCase.contains) {
				t.Fatalf("不可写原因不符：%v", err)
			}
			if len(sender.snapshot()) != 0 {
				t.Fatal("不可写时不应发出请求")
			}
		})
	}
}

func TestSelectModelFailureKeepsReason(t *testing.T) {
	catalog := &mutableCatalog{catalog: modelCatalog()}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{}, errors.New("provider 不可用")
	}}
	manager, _ := newSelectionManager(t, catalog, &stubAccount{list: readyAccount("c1")}, sender)
	if _, err := manager.RefreshModels(context.Background()); err != nil {
		t.Fatalf("预读模型目录: %v", err)
	}

	_, err := manager.SelectModel(context.Background(), SelectModelRequest{
		ProviderID:       "cloud",
		ModelID:          "c1",
		ExpectedRevision: "sel-1",
	})
	if !errors.Is(err, ErrSelectionFailed) {
		t.Fatalf("写登记失败应返回选择写入失败：%v", err)
	}
	if !strings.Contains(err.Error(), "provider 不可用") {
		t.Fatalf("失败原因应保留：%v", err)
	}
	if errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("版本未变不应报冲突：%v", err)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatal("失败不应重发")
	}
	cached, _ := manager.Models()
	if cached.Selection == nil || cached.Selection.Revision != "sel-1" {
		t.Fatalf("失败不应乐观更新缓存：%+v", cached.Selection)
	}
}

func TestSelectModelRequiresRevisionWhenKnown(t *testing.T) {
	catalog := &mutableCatalog{catalog: modelCatalog()}
	sender := &senderFunc{}
	manager, _ := newSelectionManager(t, catalog, &stubAccount{list: readyAccount("c1")}, sender)
	if _, err := manager.RefreshModels(context.Background()); err != nil {
		t.Fatalf("预读模型目录: %v", err)
	}
	if _, err := manager.SelectModel(context.Background(), SelectModelRequest{
		ProviderID: "cloud",
		ModelID:    "c1",
	}); !errors.Is(err, ErrRevisionMissing) {
		t.Fatalf("已有版本时应要求前置条件：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("缺前置条件不应发出请求")
	}
}

func TestSelectModelCapabilities(t *testing.T) {
	catalog := &mutableCatalog{catalog: modelCatalog()}
	sender := &senderFunc{}
	writer := newTestWriter(t, sender)
	manager := mustNewManager(t, Config{
		Features: stubFeatures{FeatureModelProviders: true},
		Models:   catalog,
		Account:  &stubAccount{list: readyAccount("c1")},
		Writer:   writer,
	})
	if _, err := manager.RefreshModels(context.Background()); err != nil {
		t.Fatalf("预读模型目录: %v", err)
	}
	if _, err := manager.SelectModel(context.Background(), SelectModelRequest{
		ProviderID: "cloud", ModelID: "c1", ExpectedRevision: "sel-1",
	}); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺选择写入能力应分类：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("缺能力时不应发出请求")
	}

	noAccount := mustNewManager(t, Config{
		Features: allFeatures(),
		Models:   catalog,
		Writer:   writer,
	})
	if _, err := noAccount.SelectModel(context.Background(), SelectModelRequest{
		ProviderID: "cloud", ModelID: "c1", ExpectedRevision: "sel-1",
	}); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺账号视图应分类：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("缺账号视图时不应发出请求")
	}

	noWriter := mustNewManager(t, Config{
		Features: allFeatures(),
		Models:   catalog,
		Account:  &stubAccount{list: readyAccount("c1")},
	})
	if _, err := noWriter.SelectModel(context.Background(), SelectModelRequest{
		ProviderID: "cloud", ModelID: "c1", ExpectedRevision: "sel-1",
	}); !errors.Is(err, ErrWriteUnavailable) {
		t.Fatalf("缺写登记应返回写不可用：%v", err)
	}
}
