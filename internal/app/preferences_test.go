package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPreferencesRoundTrip 改偏好后重建 App，主题、侧栏宽度与窗口几何都要保留。
func TestPreferencesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	first, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("创建 App：%v", err)
	}
	want := Preferences{
		Theme: "暗", MotionOff: true, TimeZone: "UTC", SidebarWidth: 288,
		Window: WindowPreferences{X: 40, Y: 60, Width: 1280, Height: 820, HasPosition: true},
	}
	if err := first.UpdatePreferences(want); err != nil {
		t.Fatalf("写入偏好：%v", err)
	}

	second, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("重建 App：%v", err)
	}
	got := second.Preferences()
	if got != want {
		t.Fatalf("偏好往返不一致:\n got=%+v\nwant=%+v", got, want)
	}
	if got.Window.X != 40 || got.Window.Y != 60 || got.Window.Width != 1280 || got.Window.Height != 820 {
		t.Fatalf("窗口几何往返不一致：%+v", got.Window)
	}
	if got.SidebarWidth != 288 {
		t.Fatalf("侧栏宽度往返不一致：%d", got.SidebarWidth)
	}
}

// TestPreferencesCorruptFallsBack 文件损坏时按默认值启动并记一条诊断。
func TestPreferencesCorruptFallsBack(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "preferences.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("写坏文件：%v", err)
	}
	app, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("创建 App：%v", err)
	}
	got := app.Preferences()
	if got.Theme != "跟随系统" || got.TimeZone != "跟随系统" || got.SidebarWidth != 0 {
		t.Fatalf("损坏文件没有回到默认值：%+v", got)
	}
	found := false
	for _, line := range app.Diagnostics() {
		if strings.Contains(line, "本地偏好读取失败") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("损坏文件没有记诊断：%v", app.Diagnostics())
	}
}
