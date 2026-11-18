package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// TestAboutShowsBuildVersion 核对关于页的版本与公告取本次构建版本：
// 界面不摆参考产品的样例号（v0.9.57 / 0.9.65）。
func TestAboutShowsBuildVersion(t *testing.T) {
	oldVersion, oldAnnouncements, oldBuild := aboutScreen.currentVersion, aboutScreen.announcements, buildVersion
	t.Cleanup(func() {
		aboutScreen.currentVersion = oldVersion
		aboutScreen.announcements = oldAnnouncements
		buildVersion = oldBuild
	})

	SetBuildVersion("1.2.3")
	app := NewApp(Options{State: StateDefault})
	app.screen, app.settingsSection = ScreenSettings, secAbout
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if !tt.HasText("DimAgent Desktop 1.2.3") {
		t.Fatal("公告标题不是本次构建版本")
	}
	// 版本号那一行的无障碍标签是「版本号」，值本身不在标签树里：直接从界面状态核。
	if got := aboutScreen.currentVersion; got != "v1.2.3" {
		t.Fatalf("当前版本 = %q，想要 v1.2.3", got)
	}
	for _, sample := range []string{"v0.9.57", "0.9.65"} {
		if tt.HasText(sample) {
			t.Fatalf("关于页仍留着参考产品的样例版本号 %q", sample)
		}
	}
}
