package ui

import (
	"testing"
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
