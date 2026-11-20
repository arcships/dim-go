package ui

import (
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置壳（S-shell）本版可做条目的回归：图标缺资源时导航文字不左移、
// 读数失败仍可切节、窄窗的返回与节列表、返回按钮的命中区。
//
// 需求 ID 写在每个用例的注释里。

// TestSettingsNavIconMissingKeepsTextColumn 断言分节图标缺资源时只留空位，名称的
// 左缘与行高不变（S-shell-04 … 40 的「异常」行）。
func TestSettingsNavIconMissingKeepsTextColumn(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := openSettingsSection(t, app, secAccount)

	before, ok := tt.Access("通用")
	if !ok {
		t.Fatalf("导航里没有「通用」：%v", tt.Texts())
	}
	saved := settingsSectionDefs[4].icon
	settingsSectionDefs[4].icon = nil
	defer func() { settingsSectionDefs[4].icon = saved }()
	tt.Frame()

	after, ok := tt.Access("通用")
	if !ok {
		t.Fatal("图标缺资源后「通用」不见了")
	}
	if after.Bounds.X != before.Bounds.X {
		t.Fatalf("图标缺资源后名称左缘 = %v，想要 %v（不左移）", after.Bounds.X, before.Bounds.X)
	}
	if after.Bounds.Y != before.Bounds.Y || after.Bounds.H != before.Bounds.H {
		t.Fatalf("图标缺资源后行位置/行高变了：%+v → %+v", before.Bounds, after.Bounds)
	}
}

// TestSettingsNavKeepsSectionClickableOnReadFailure 断言某节读数失败时导航项仍可点、
// 切过去以后节内自己写错误（S-shell-03 / 07 / 11 / 15 / 19 / 21 / 25 / 29 / 33 / 35）。
func TestSettingsNavKeepsSectionClickableOnReadFailure(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.settings.ProviderRecords = nil
	src.settings.ProviderError = "provider.list 超时"
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secAccount)

	if err := tt.Click("模型供应"); err != nil {
		t.Fatalf("读数失败时导航项点不了：%v", err)
	}
	tt.Frame()
	if app.settingsSection != secProviders {
		t.Fatalf("切到的分节 = %v，想要模型供应", app.settingsSection)
	}
	if !tt.HasText("读取供应商失败：provider.list 超时") {
		t.Fatalf("节内没有写错误：%v", tt.Texts())
	}
}

// TestSettingsReadFailureMarksNavRow 断言读数失败的节在导航行尾挂警示点，原因写进
// 读屏说明。
func TestSettingsReadFailureMarksNavRow(t *testing.T) {
	resetSettingsScreenState()
	src := wiredSource()
	src.extensions.MCP = ExtListSnapshot{ReadError: "mcp.catalog 不可用"}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := openSettingsSection(t, app, secAccount)

	row, ok := tt.AccessByRole(gui.RoleListItem, "MCP")
	if !ok {
		t.Fatalf("找不到 MCP 导航行：%v", tt.Texts())
	}
	if row.Description != "mcp.catalog 不可用" {
		t.Fatalf("导航行没有写读数失败的原因：%+v", row)
	}
}

// TestSettingsNarrowBackToSectionList 断言窄窗的返回先去节列表、再回主界面
// （S-shell-41 / S-shell-42）。
func TestSettingsNarrowBackToSectionList(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	app.screen = ScreenSettings
	tt := gui.NewTester(app.View, narrowWidth, WindowHeight)

	if _, ok := tt.Find("返回节列表"); !ok {
		t.Fatalf("窄窗设置没有返回节列表的按钮：%v", tt.Texts())
	}
	if err := tt.Click("返回节列表"); err != nil {
		t.Fatalf("点不到返回：%v", err)
	}
	tt.Frame()
	if !app.settingsNarrowList {
		t.Fatal("返回后没有停在节列表")
	}
	// 节列表里 19 节都在，点一行进该节。
	for _, label := range []string{"账户", "外观", "关于"} {
		if _, ok := tt.AccessByRole(gui.RoleListItem, label); !ok {
			t.Fatalf("节列表缺少 %q：%v", label, tt.Texts())
		}
	}
	if err := tt.Click("外观"); err != nil {
		t.Fatalf("点不到节列表里的外观：%v", err)
	}
	tt.Frame()
	if app.settingsSection != secAppearance || app.settingsNarrowList {
		t.Fatalf("节列表点外观后：分节 %v、列表态 %v", app.settingsSection, app.settingsNarrowList)
	}
	if !strings.Contains(joinedText(tt), "主题") {
		t.Fatal("进节后没有画外观内容")
	}
	// 再回一次：这次回主界面。
	if _, ok := tt.Find("返回节列表"); !ok {
		t.Fatalf("进节后返回按钮不在：%v", tt.Texts())
	}
	if err := tt.Click("返回节列表"); err != nil {
		t.Fatalf("点不到返回：%v", err)
	}
	tt.Frame()
	if err := tt.Click("返回主界面"); err != nil {
		t.Fatalf("节列表态没有「返回主界面」：%v", tt.Texts())
	}
	if app.screen != ScreenMain {
		t.Fatalf("节列表态返回后屏幕 = %v，想要主界面", app.screen)
	}
}

// TestSettingsNarrowBackIconMissingKeepsHitArea 断言返回按钮的图标资源缺失时命中区
// 仍是 24×24，点得动（S-shell-42）。
func TestSettingsNarrowBackIconMissingKeepsHitArea(t *testing.T) {
	resetSettingsScreenState()
	saved := iconChevronL
	iconChevronL = nil
	defer func() { iconChevronL = saved }()

	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	app.screen = ScreenSettings
	tt := gui.NewTester(app.View, narrowWidth, WindowHeight)

	info, ok := tt.Access("返回节列表")
	if !ok {
		t.Fatalf("图标缺资源时返回按钮不见了：%v", tt.Texts())
	}
	if info.Bounds.W < hitMin || info.Bounds.H < hitMin {
		t.Fatalf("返回按钮命中区 = %v×%v，想要不小于 %v", info.Bounds.W, info.Bounds.H, hitMin)
	}
	if err := tt.Click("返回节列表"); err != nil {
		t.Fatalf("图标缺资源时点不到返回：%v", err)
	}
	tt.Frame()
	if !app.settingsNarrowList {
		t.Fatal("点返回没有回到节列表")
	}
}
