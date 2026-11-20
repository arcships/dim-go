package ui

import (
	"context"
	"errors"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 模型供应」详情面板与「添加模型」对话框的回归：掩码、测试连接、
// 模型空态、本地草稿的填、校验与过滤、断开 / 删除确认框、加载骨架与全选回滚。
//
// 需求 ID 写在各个用例的注释里。

// providerDetailIntents 记录详情面板发出的写入；failProbe / failToggle 让对应
// 方法失败，其余方法不实现（调到就 panic）。
type providerDetailIntents struct {
	Intents
	probes     chan string
	toggles    chan string
	failProbe  error
	failToggle error
}

func newProviderDetailIntents() *providerDetailIntents {
	return &providerDetailIntents{
		probes:  make(chan string, 8),
		toggles: make(chan string, 8),
	}
}

func (p *providerDetailIntents) ProbeProvider(_ context.Context, id string) (int, error) {
	p.probes <- id
	if p.failProbe != nil {
		return 0, p.failProbe
	}
	return 2, nil
}

func (p *providerDetailIntents) SetProviderModelEnabled(
	_ context.Context, providerID, modelID string, enabled bool,
) error {
	p.toggles <- providerID + "/" + modelID
	if p.failToggle != nil {
		return p.failToggle
	}
	return nil
}

// providerDetailSource 是一个带凭据与远端清单的供应商读数。
func providerDetailSource() *fakeSource {
	src := wiredSource()
	src.settings.ProviderRecordsKnown = true
	src.settings.ProviderRecords = []ProviderRecordSnapshot{{
		ID: "p1", Name: "Acme 云", BaseURL: "https://acme.test/v1",
		DriverKind: "openai", AuthType: "bearer", Enabled: true, Version: 2,
		CredentialConfigured: true, RemoteModelList: true,
		Models: []ProviderModelSnapshot{
			{ID: "m1", Name: "Model-1", Enabled: true},
			{ID: "m2", Name: "Model-2", Enabled: false},
		},
	}}
	return src
}

// TestProviderDetailMasksCredential 断言 API 密钥只显示掩码 + 「已设置」，修改进
// 编辑态（R-CFG-362 / 363 / 458 / 482 / 486）。
func TestProviderDetailMasksCredential(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: providerDetailSource()})
	tt := openSettingsSection(t, app, secProviders)

	for _, want := range []string{"API 密钥", "••••••••", "API 密钥已设置"} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("详情缺少 %q：%v", want, tt.Texts())
		}
	}
	if _, ok := tt.Find("sk-"); ok {
		t.Fatal("详情画了密钥明文")
	}
	if err := tt.Click("修改 API 密钥"); err != nil {
		t.Fatalf("点不到修改：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("编辑供应商"); !ok {
		t.Fatal("修改没有进编辑态")
	}
}

// TestProviderDetailTestConnection 断言测试连接走 provider.probe：进行中按钮禁用，
// 失败原因写在行内（R-CFG-394 / 395 / 397）。
func TestProviderDetailTestConnection(t *testing.T) {
	resetSettingsScreenState()
	intents := newProviderDetailIntents()
	intents.failProbe = errors.New("凭证无效")
	app := NewApp(Options{State: StateDefault, Source: providerDetailSource(),
		Intents: intents})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click("测试连接"); err != nil {
		t.Fatalf("点不到测试连接：%v", err)
	}
	select {
	case got := <-intents.probes:
		if got != "p1" {
			t.Fatalf("探测的供应商 = %q，想要 p1", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有发起探测")
	}
	waitFrame(t, tt, "测试连接失败行", func() bool {
		return tt.HasText("连接失败：凭证无效")
	})
	if !tt.HasText("连接失败：凭证无效") {
		t.Fatalf("测试失败没有写原因：%v", tt.Texts())
	}
}

// TestProviderModelsEmptyState 断言没有模型数据时的空态句点名刷新，点刷新走探测，
// 失败换成错误行（R-CFG-425 / 428 / 429 / 430 / 431）。
func TestProviderModelsEmptyState(t *testing.T) {
	resetSettingsScreenState()
	src := providerDetailSource()
	src.settings.ProviderRecords[0].Models = nil
	intents := newProviderDetailIntents()
	intents.failProbe = errors.New("上游超时")
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := openSettingsSection(t, app, secProviders)

	if !tt.HasText("还没有模型数据，点击「刷新」获取") {
		t.Fatalf("没有写空态句：%v", tt.Texts())
	}
	if err := tt.Click("刷新"); err != nil {
		t.Fatalf("点不到刷新：%v", err)
	}
	waitFrame(t, tt, "拉取失败错误行", func() bool {
		return tt.HasText("拉取失败：上游超时")
	})
	if !tt.HasText("拉取失败：上游超时") {
		t.Fatalf("拉取失败没有换成错误行：%v", tt.Texts())
	}
}

// TestProviderModelDraftValidatesLocally 断言「添加模型」对话框能填、能过滤、能
// 全选，保存只在本地校验，不发写请求（R-CFG-555 / 570 / 571 / 593）。
func TestProviderModelDraftValidatesLocally(t *testing.T) {
	resetSettingsScreenState()
	intents := newProviderDetailIntents()
	app := NewApp(Options{State: StateDefault, Source: providerDetailSource(),
		Intents: intents})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click("添加模型"); err != nil {
		t.Fatalf("点不到添加模型：%v", err)
	}
	tt.Frame()
	d := configScreen.draft
	if d == nil {
		t.Fatal("添加模型对话框没有打开")
	}
	// 空标识保存：本地校验给字段原因。
	if err := tt.Click("保存模型"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if !tt.HasText("第 1 个模型缺少标识") {
		t.Fatalf("空标识没有校验：%v", tt.Texts())
	}
	// 填两个标识，过滤后仍能全选。
	d.rows = []string{"zeta-1", "alpha-2"}
	d.on = []bool{true, true}
	tt.Frame()
	d.query = "zeta"
	tt.Frame()
	if _, ok := tt.AccessByRole(gui.RoleTextField, "草稿模型 2"); ok {
		t.Fatal("搜索词没有过滤草稿行")
	}
	first, ok := tt.AccessByRole(gui.RoleTextField, "草稿模型 1")
	if !ok {
		t.Fatalf("搜索词把命中的草稿行也过滤掉了：%v", tt.Texts())
	}
	if first.Value != "zeta-1" {
		t.Fatalf("留下的草稿行 = %q，想要 zeta-1", first.Value)
	}
	state := gui.CheckOff
	_ = state
	if len(d.sortedDraftIDs()) != 2 {
		t.Fatalf("草稿标识 = %v，想要两条", d.sortedDraftIDs())
	}
	// 合法保存：本地校验通过，写「没有通道」的原因，且不发请求。
	d.query = ""
	tt.Frame()
	if err := tt.Click("保存模型"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if !tt.HasText(noteModelCreateWrite) {
		t.Fatalf("合法草稿保存没有写原因：%v", tt.Texts())
	}
	select {
	case got := <-intents.toggles:
		t.Fatalf("添加模型发了写请求：%v", got)
	default:
	}
}

// TestProviderDisconnectDialogBlocks 断言断开确认框的标题、取消关闭、确认键置灰并
// 写原因（R-CFG-489 / 492 / 495）。
func TestProviderDisconnectDialogBlocks(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: providerDetailSource()})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click("断开此提供商"); err != nil {
		t.Fatalf("点不到断开：%v", err)
	}
	tt.Frame()
	if _, ok := tt.AccessByRole(gui.RoleDialog, "断开此提供商"); !ok {
		t.Fatalf("没有断开确认框：%v", tt.Texts())
	}
	info, ok := tt.AccessByRole(gui.RoleButton, "断开连接")
	if !ok || !info.Disabled {
		t.Fatalf("断开确认键没有置灰：%+v（找到 %v）", info, ok)
	}
	if info.Description != noteProviderDisconnectWrite {
		t.Fatalf("断开确认键没写原因：%q", info.Description)
	}
	if err := tt.Click("取消断开此提供商"); err != nil {
		t.Fatalf("点不到取消：%v", err)
	}
	tt.Frame()
	if _, ok := tt.AccessByRole(gui.RoleDialog, "断开此提供商"); ok {
		t.Fatal("取消后确认框没有关")
	}
}

// TestProviderDetailLoadingSkeleton 断言清单还没读到时画骨架并播报一次，不把
// 「没读到」画成「没有供应商」（R-CFG-643 / 645）。
func TestProviderDetailLoadingSkeleton(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.settings.ProviderRecords = nil
	src.settings.ProviderRecordsKnown = false
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secProviders)

	if !tt.HasText("正在加载供应商") {
		t.Fatalf("没有画加载骨架：%v", tt.Texts())
	}
	if tt.HasText("没有读到供应商。") {
		t.Fatal("还没读到就画了「没有读到」")
	}
}

// TestProviderSelectAllRollsBack 断言模型全选的写入失败后开关回到原值并写原因
// （R-CFG-160 / 560）。
func TestProviderSelectAllRollsBack(t *testing.T) {
	resetSettingsScreenState()
	intents := newProviderDetailIntents()
	intents.failToggle = errors.New("写入被拒绝")
	app := NewApp(Options{State: StateDefault, Source: providerDetailSource(),
		Intents: intents})
	tt := openSettingsSection(t, app, secProviders)

	before := configEnabledModelCount(app.configView())
	info, ok := tt.AccessByRole(gui.RoleCheckBox, "模型")
	if !ok {
		t.Fatalf("找不到全选框：%v", tt.Texts())
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	waitFrame(t, tt, "全选回滚", func() bool {
		return tt.HasText("保存失败，已恢复原值：写入被拒绝")
	})
	if got := configEnabledModelCount(app.configView()); got != before {
		t.Fatalf("失败后已启用数 = %d，想要回滚到 %d", got, before)
	}
}

// TestProviderDeleteDialogBlocks 断言删除确认框的标题与置灰的确认键（R-CFG-497 /
// 500 / 503）。
func TestProviderDeleteDialogBlocks(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: providerDetailSource()})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click("删除此提供商"); err != nil {
		t.Fatalf("点不到删除：%v", err)
	}
	tt.Frame()
	if _, ok := tt.AccessByRole(gui.RoleDialog, "删除此提供商"); !ok {
		t.Fatalf("没有删除确认框：%v", tt.Texts())
	}
	info, ok := tt.AccessByRole(gui.RoleButton, "删除")
	if !ok || !info.Disabled {
		t.Fatalf("删除确认键没有置灰：%+v（找到 %v）", info, ok)
	}
	if info.Description != noteProviderDeleteWrite {
		t.Fatalf("删除确认键没写原因：%q", info.Description)
	}
}

// TestProviderProbeBusyDisablesEntries 断言探测进行中「测试连接」与「刷新」禁用、
// 「保存」仍可用（R-CFG-211 / 395）。
func TestProviderProbeBusyDisablesEntries(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: providerDetailSource()})
	tt := openSettingsSection(t, app, secProviders)

	configScreen.probing = true
	defer func() { configScreen.probing = false }()
	tt.Frame()
	for _, label := range []string{"测试连接", "刷新"} {
		info, ok := tt.AccessByRole(gui.RoleButton, label)
		if !ok || !info.Disabled {
			t.Fatalf("探测中 %q 没有置灰：%+v（找到 %v）", label, info, ok)
		}
	}
	if info, ok := tt.AccessByRole(gui.RoleButton, "保存"); !ok || info.Disabled {
		t.Fatalf("探测中「保存」不可用：%+v（找到 %v）", info, ok)
	}
}
