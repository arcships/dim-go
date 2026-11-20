package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 通用 › 浏览器数据两个对话框的回归（G-H6 / G-H7）：通道没有接，
// 所以各态不画样例读数，动作置灰并把缺什么写清楚。

// generalBrowserTester 打开设置 › 通用（样例模式：两行动作可点，对话框能开）。
func generalBrowserTester(t *testing.T) (*App, *gui.Tester) {
	t.Helper()
	generalScreen = generalState{keepAwake: true, launch: true}
	app := NewApp(Options{State: StateDefault})
	tt := openSettingsSection(t, app, secGeneral)
	tt.Frame()
	return app, tt
}

// G-H6：标题与说明逐字；扫描行 / 配置行 / 计数没有读数（写原因），导入置灰；
// 取消关掉对话框。
func TestGeneralImportDialogStatesWithoutChannel(t *testing.T) {
	_, tt := generalBrowserTester(t)
	if err := tt.Click("导入…"); err != nil {
		t.Fatalf("点不到导入…：%v", err)
	}
	tt.Frame()
	for _, want := range []string{"导入 Chrome Cookie", generalImportDesc, noteBrowserNoImport} {
		if !tt.HasText(want) {
			t.Fatalf("导入对话框缺 %q：%v", want, tt.Texts())
		}
	}
	// 不画样例配置与假计数。
	for _, banned := range []string{"Default · 可导入", "Profile 1 · 需要授权", "可导入 128 · 会话 96 · 已过期 20 · 分区 12"} {
		if tt.HasText(banned) {
			t.Fatalf("导入对话框画了样例值 %q", banned)
		}
	}
	if info, ok := tt.Access("导入"); !ok || !info.Disabled {
		t.Fatalf("没有通道时「导入」没有置灰：%+v（找到 %v）", info, ok)
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatalf("点不到取消：%v", err)
	}
	tt.Frame()
	if generalScreen.importOpen {
		t.Fatal("取消后导入对话框没有关")
	}
}

// G-H7：标题与说明逐字；成功行 / 错误块没有回执（写原因），清理置灰；取消关掉。
func TestGeneralClearDialogStatesWithoutChannel(t *testing.T) {
	_, tt := generalBrowserTester(t)
	if err := tt.Click("清理数据…"); err != nil {
		t.Fatalf("点不到清理数据…：%v", err)
	}
	tt.Frame()
	for _, want := range []string{"清理浏览器数据", generalClearDesc, noteBrowserNoClear} {
		if !tt.HasText(want) {
			t.Fatalf("清理对话框缺 %q：%v", want, tt.Texts())
		}
	}
	if tt.HasText("内置浏览器数据已清理。") {
		t.Fatal("没有清理通道却画了成功行")
	}
	if info, ok := tt.AccessByRole(gui.RoleButton, "清理浏览器数据"); !ok || !info.Disabled {
		t.Fatalf("没有通道时「清理浏览器数据」没有置灰：%+v（找到 %v）", info, ok)
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatalf("点不到取消：%v", err)
	}
	tt.Frame()
	if generalScreen.clearOpen {
		t.Fatal("取消后清理对话框没有关")
	}
}
