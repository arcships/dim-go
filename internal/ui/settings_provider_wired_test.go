package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 设置 › 模型供应接的是 provider 读数与四个写入口：清单与模型来自
// Settings().ProviderRecords，模型开关走 SetProviderModelEnabled，「刷新」走
// ProbeProvider（探测只读，不写配置）。真实服务端的方法在 internal/uiwire 验。

// providerRecordsSource 造一个只有一组 provider 读数的源。
func providerRecordsSource() *fakeSource {
	src := wiredSource()
	src.settings.ProviderRecords = []ProviderRecordSnapshot{{
		ID: "p1", Name: "Acme", BaseURL: "https://acme.test/v1",
		DriverKind: "openai", AuthType: "bearer", Enabled: true, Version: 3,
		Models: []ProviderModelSnapshot{
			{ID: "m1", Name: "Model-1", Enabled: true, Status: "ready"},
			{ID: "m2", Name: "Model-2", Enabled: false},
		},
	}}
	return src
}

// TestProviderSettingsReadsRecords 断言左栏与模型块画的是读数：样例的
// DimAgent OAuth 与 32 个样例模型都不再上屏。
func TestProviderSettingsReadsRecords(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: providerRecordsSource()})
	tt := openSettingsSection(t, app, secProviders)

	for _, want := range []string{"Acme", "Model-1", "Model-2", "https://acme.test/v1"} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("模型供应屏缺少读数里的 %q", want)
		}
	}
	for _, sample := range []string{configDimName, "Claude Opus 5.5", "GPT-6 Sol"} {
		if _, ok := tt.Find(sample); ok {
			t.Fatalf("连着 Source 时仍画了样例 %q", sample)
		}
	}
	if _, ok := tt.Find("模型 · 2 / 2"); !ok {
		t.Fatal("模型计数没有按读数算")
	}
}

// TestProviderSettingsModelToggleWrites 断言模型开关走
// SetProviderModelEnabled（带 provider 与 model 的标识）。
func TestProviderSettingsModelToggleWrites(t *testing.T) {
	resetSettingsScreenState()
	rec := newRecordIntents()
	app := NewApp(Options{State: StateDefault, Source: providerRecordsSource(), Intents: rec})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click("启用模型 Model-2"); err != nil {
		t.Fatalf("点不到模型开关：%v", err)
	}
	tt.Frame()
	rec.expect(t, "SetProviderModelEnabled p1 m2 true")

	if err := tt.Click("启用模型 Model-1"); err != nil {
		t.Fatalf("点不到模型开关：%v", err)
	}
	tt.Frame()
	rec.expect(t, "SetProviderModelEnabled p1 m1 false")
}

// TestProviderSettingsProbeWrites 断言详情底部的「刷新」= 重新探测（provider.probe）。
func TestProviderSettingsProbeWrites(t *testing.T) {
	resetSettingsScreenState()
	rec := newRecordIntents()
	app := NewApp(Options{State: StateDefault, Source: providerRecordsSource(), Intents: rec})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click("刷新"); err != nil {
		t.Fatalf("点不到刷新：%v", err)
	}
	tt.Frame()
	rec.expect(t, "ProbeProvider p1")
}

// TestProviderSettingsReadError 断言读数为空与读取失败各有分支：失败给原因，
// 空清单不画成样例。
func TestProviderSettingsReadError(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.settings.ProviderRecords = nil
	src.settings.ProviderError = "provider.list 超时"
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secProviders)

	if _, ok := tt.Find("读取供应商失败：provider.list 超时"); !ok {
		t.Fatal("读取失败没有给出原因")
	}
	if _, ok := tt.Find("没有读到供应商。"); !ok {
		t.Fatal("空清单没有给空态")
	}
	if _, ok := tt.Find(configDimName); ok {
		t.Fatal("读不到时仍画了样例供应商")
	}
	if _, ok := tt.Find("选择一个提供商查看详情"); !ok {
		t.Fatal("没有选中项时没有空态")
	}
}

// ---- 供应商表单（新增 / 编辑）----

// TestProviderFormOpensFromAdd 断言左栏「添加」打开表单：字段、动作都在，详情让位。
func TestProviderFormOpensFromAdd(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: providerRecordsSource()})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click("添加"); err != nil {
		t.Fatalf("点不到添加：%v", err)
	}
	tt.Frame()
	for _, want := range []string{
		"添加自定义提供商", "名称", "基础 URL", "API 密钥", "模型清单", "保存", "取消",
	} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("表单缺少 %q", want)
		}
	}
	// 表单与详情互斥：详情头部的供应商名不再上屏（列表里的同名行不受影响，
	// 它在左栏且仍是读数的第一项）。
	if _, ok := tt.Find("模型 · 2 / 2"); ok {
		t.Fatal("打开表单后仍画了详情的模型块")
	}
}

// TestProviderFormValidation 断言必填、URL 形态与重复名称都在框下给原因，
// 且校验不过时不发请求。
func TestProviderFormValidation(t *testing.T) {
	resetSettingsScreenState()
	rec := newRecordIntents()
	app := NewApp(Options{State: StateDefault, Source: providerRecordsSource(), Intents: rec})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click("添加"); err != nil {
		t.Fatalf("点不到添加：%v", err)
	}
	tt.Frame()
	if err := tt.Click("保存"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	for _, want := range []string{"请填写名称", "请填写基础 URL", "请填写 API 密钥"} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("空表单保存后缺少提示 %q", want)
		}
	}

	// URL 形态：不是 http/https 的绝对地址。
	form := configScreen.form
	form.name, form.baseURL, form.apiKey = "New", "api.acme.test/v1", "sk-test"
	tt.Frame()
	if err := tt.Click("保存"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("基础 URL 需以 http:// 或 https:// 开头"); !ok {
		t.Fatal("非法 URL 没有在框下给原因")
	}

	// 重复名称：与读数里的 Acme 同名（不区分大小写）。
	form.baseURL = "https://new.acme.test/v1"
	form.name = "acme"
	tt.Frame()
	if err := tt.Click("保存"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("名称已被其他供应商使用"); !ok {
		t.Fatal("重名没有在框下给原因")
	}

	if len(rec.calls) != 0 {
		t.Fatalf("校验不过仍发了请求：%v", rec.calls)
	}
}

// TestProviderFormCreateWrites 断言新增走 CreateProvider，模型清单随请求下发，
// 成功后回到详情态。
func TestProviderFormCreateWrites(t *testing.T) {
	resetSettingsScreenState()
	rec := newRecordIntents()
	app := NewApp(Options{State: StateDefault, Source: providerRecordsSource(), Intents: rec})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click("添加"); err != nil {
		t.Fatalf("点不到添加：%v", err)
	}
	tt.Frame()
	form := configScreen.form
	form.name, form.baseURL, form.apiKey = "Acme 2", "https://acme2.test/v1", "sk-test"
	form.models = []string{"m9"}
	tt.Frame()

	if err := tt.Click("保存"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	rec.expect(t, "CreateProvider Acme 2 models=m9")
	tt.Frame()
	if _, ok := tt.Find("添加自定义提供商"); ok {
		t.Fatal("保存成功后表单没有关闭")
	}
}

// TestProviderFormEditPrefilled 断言编辑入口带入读数里的值、模型清单只读，
// 保存走 UpdateProvider。
func TestProviderFormEditPrefilled(t *testing.T) {
	resetSettingsScreenState()
	rec := newRecordIntents()
	app := NewApp(Options{State: StateDefault, Source: providerRecordsSource(), Intents: rec})
	tt := openSettingsSection(t, app, secProviders)

	if err := tt.Click("编辑"); err != nil {
		t.Fatalf("点不到编辑：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("编辑供应商"); !ok {
		t.Fatal("编辑入口没有打开表单")
	}
	// 预填在输入框的取值里（文本节点的 Label 就是它的内容，不看值）。
	prefilled := []struct{ label, value string }{
		{"名称", "Acme"},
		{"基础 URL", "https://acme.test/v1"},
		{"模型 1", "m1"},
		{"模型 2", "m2"},
	}
	for _, want := range prefilled {
		info, ok := tt.AccessByRole(gui.RoleTextField, want.label)
		if !ok {
			t.Fatalf("编辑表单缺少字段 %q", want.label)
		}
		if info.Value != want.value {
			t.Fatalf("%s 预填 = %q，想要 %q", want.label, info.Value, want.value)
		}
	}
	// 编辑态的模型清单只读：没有「添加模型」，也没有删除行。
	if _, ok := tt.Find("添加模型"); ok {
		t.Fatal("编辑态仍给了添加模型的入口（模型写通道不存在）")
	}

	form := configScreen.form
	if !form.editing || form.id != "p1" {
		t.Fatalf("编辑草稿没有指向读数里的供应商：%+v", form)
	}
	form.name = "Acme 改名"
	tt.Frame()
	if err := tt.Click("保存"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	rec.expect(t, "UpdateProvider p1")
	tt.Frame()
	if _, ok := tt.Find("编辑供应商"); ok {
		t.Fatal("保存成功后表单没有关闭")
	}
}

// TestProviderFormSelectsCreatedProvider 断言新增成功后焦点落在新供应商上：
// 新建回执不带标识，待选的名称在下一帧的读数里解析成那一项。
func TestProviderFormSelectsCreatedProvider(t *testing.T) {
	resetSettingsScreenState()
	src := providerRecordsSource()
	app := NewApp(Options{State: StateDefault, Source: src})
	// 服务端新建后清单里出现这一项（与 CreateProvider 的回执后重读同构）。
	src.settings.ProviderRecords = append(src.settings.ProviderRecords, ProviderRecordSnapshot{
		ID: "p2", Name: "Acme 2", BaseURL: "https://acme2.test/v1", Enabled: true, Version: 1,
	})
	configScreen.selectAfterCreate = "Acme 2"
	tt := openSettingsSection(t, app, secProviders)

	if got := configScreen.selected; got != "p2" {
		t.Fatalf("新建后选中的是 %q，想要 p2", got)
	}
	if _, ok := tt.Find("https://acme2.test/v1"); !ok {
		t.Fatal("详情没有切到新建的供应商")
	}
}
