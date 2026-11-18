package ui

// 临时探针：打印三屏里若干元素的盒子，跑完即删。

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func TestProbeTemp(t *testing.T) {
	cases := []struct {
		name    string
		section settingsSectionID
		render  func(*gui.Context)
		labels  []string
	}{
		{"notifications", secNotifications, renderNotifications, []string{"通知", "任务完成或失败时通知"}},
		{"pets", secPets, renderPets, []string{"宠物", "显示 Pet", "Pet 会保持在其他窗口上方。", "选择 Pet", "自定义 Pets", "尚未安装自定义 Pet。"}},
		{"permissions", secPermissions, renderPermissions, []string{"权限", "进程隔离", "在沙箱中运行命令", "允许网络访问", "保存隔离设置", "规则", "命令", "添加规则", "保存权限"}},
	}
	for _, tc := range cases {
		section, render := tc.section, tc.render
		app := NewApp(Options{State: StateDefault})
		view := func(c *gui.Context) {
			app.screen, app.settingsSection = ScreenSettings, section
			c.SetTheme(theme(false, c.Preferences()))
			tm := c.Theme()
			gui.Row(c).Fill().AlignItems(gui.Stretch).Background(tm.Background).Label("设置").
				Children(func() {
					app.settingsNav(c)
					gui.Column(c).Grow(1).MinWidth(MainMinWidth).Children(func() {
						gui.Column(c).Grow(1).MaxWidth(SettingsContentMaxWidth).
							Padding(space8, space8).Gap(space4).Children(func() {
								render(c)
							})
					})
				})
		}
		tt := gui.NewTester(view, 1030, 725)
		t.Logf("== %s", tc.name)
		for _, label := range tc.labels {
			r, ok := tt.Access(label)
			if ok {
				t.Logf("  %-24s access %+v", label, r.Bounds)
				continue
			}
			r2, ok2 := tt.Find(label)
			t.Logf("  %-24s find  %+v ok=%v", label, r2, ok2)
		}
	}
}
