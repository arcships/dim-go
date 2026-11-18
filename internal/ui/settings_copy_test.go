package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 设置屏的文案与元素口径（visual-diff 已修条目）：
//
//	S2 命令 Shell 的取值是「已保存的 Shell 不可用：」；
//	S3 快捷键搜索框只有占位「搜索快捷键」（无放大镜、无省略号）；
//	S5 插件设置没有页首标题与说明；
//	S6 公告日期与账户节同一口径（数字与年月日之间不插空格）；
//	S8 多模态的「所有者任务」行只有标题 + 刷新（没有状态筛选下拉）。
func TestSettingsCopyMatchesReference(t *testing.T) {
	resetSettingsScreenState()
	app := NewApp(Options{State: StateDefault})
	app.screen = ScreenSettings
	render := func(id settingsSectionID) *gui.Tester {
		app.settingsSection = id
		return gui.NewTester(app.View, WindowWidth, WindowHeight)
	}

	// S2：选择器取值。
	tt := render(secPersonalization)
	if !tt.HasText("已保存的 Shell 不可用：") {
		t.Fatal("命令 Shell 的取值不是参考文案")
	}
	if tt.HasText("自动（没有受支持的 Shell）") {
		t.Fatal("命令 Shell 仍在使用旧取值")
	}

	// S3：占位逐字，且没有前置放大镜。
	tt = render(secKeybindings)
	info, ok := tt.AccessByRole(gui.RoleTextField, "搜索快捷键")
	if !ok {
		t.Fatal("快捷键屏缺少搜索框")
	}
	if info.Placeholder != "搜索快捷键" {
		t.Fatalf("搜索框占位 = %q，想要「搜索快捷键」", info.Placeholder)
	}
	if tt.HasText("图标：search") {
		t.Fatal("搜索框里不应再有放大镜图标")
	}

	// S5：没有页首标题下的说明行（第一张插件卡就是首块内容）。
	tt = render(secPluginSettings)
	if tt.HasText("配置 App Server 插件声明的设置。") {
		t.Fatal("插件设置不应再有页首说明")
	}

	// S6：公告日期不插空格，与账户节同一个格式。
	tt = render(secAbout)
	if !tt.HasText("2026年11月16日") {
		t.Fatal("公告日期不是「2026年11月16日」")
	}
	if tt.HasText("2026 年 11 月 16 日") {
		t.Fatal("公告日期里多了空格")
	}

	// S8：所有者任务没有状态筛选。
	tt = render(secModality)
	for _, gone := range []string{"全部状态", "状态筛选"} {
		if tt.HasText(gone) {
			t.Fatalf("所有者任务行不应再有 %q", gone)
		}
	}
}
