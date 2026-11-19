package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestPreferencesRoundTrip 改偏好后重建 App，主题、侧栏宽度、登记的目录与窗口
// 几何都要保留。
func TestPreferencesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	first, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("创建 App：%v", err)
	}
	want := Preferences{
		Theme: "暗", MotionOff: true, TimeZone: "UTC", SidebarWidth: 288,
		Workspaces:       []string{"/tmp/dimagent-test/ws-alpha", "/tmp/dimagent-test/ws-beta"},
		HooksProjectPath: "/tmp/dimagent-test/hooks-project",
		UpdateFeedURL:    "https://example.com/updates/latest.json",
		Window:           WindowPreferences{X: 40, Y: 60, Width: 1280, Height: 820, HasPosition: true},
	}
	if err := first.UpdatePreferences(want); err != nil {
		t.Fatalf("写入偏好：%v", err)
	}

	second, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("重建 App：%v", err)
	}
	got := second.Preferences()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("偏好往返不一致:\n got=%+v\nwant=%+v", got, want)
	}
	if got.Window.X != 40 || got.Window.Y != 60 || got.Window.Width != 1280 || got.Window.Height != 820 {
		t.Fatalf("窗口几何往返不一致：%+v", got.Window)
	}
	if got.SidebarWidth != 288 {
		t.Fatalf("侧栏宽度往返不一致：%d", got.SidebarWidth)
	}
}

// TestAppearancePreferencesRoundTrip 外观偏好（界面字体与字号、代码字体与字号、
// 代码主题）落在 preferences.json 的 appearance.* 下，重建后原样回来，并且越界值
// 被夹到可用区间。
func TestAppearancePreferencesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	first, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("创建 App：%v", err)
	}
	want := AppearancePreferences{
		UIFont: "PingFang SC", CodeFont: "Menlo", UIScale: 1.08, CodeSize: 18,
		CodeTheme: "One Dark Pro",
	}
	if err := first.UpdatePreferences(Preferences{Appearance: want}); err != nil {
		t.Fatalf("写入偏好：%v", err)
	}
	second, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("重建 App：%v", err)
	}
	if got := second.Preferences().Appearance; got != want {
		t.Fatalf("外观偏好往返不一致：\n got=%+v\nwant=%+v", got, want)
	}

	// appearance.* 是 preferences.json 里的一节（不是顶层键）。
	raw, err := os.ReadFile(filepath.Join(dir, "preferences.json"))
	if err != nil {
		t.Fatalf("读偏好文件：%v", err)
	}
	if !strings.Contains(string(raw), `"appearance"`) ||
		!strings.Contains(string(raw), `"codeTheme": "One Dark Pro"`) {
		t.Fatalf("偏好文件里没有 appearance 这一节：\n%s", raw)
	}

	// 越界值夹回区间：倍率 0.8–1.3、代码字号 10–24。
	clamped := normalizeAppearancePreferences(AppearancePreferences{
		UIScale: 4, CodeSize: 99, UIFont: "  Menlo  ",
	})
	if clamped.UIScale != 1.3 || clamped.CodeSize != 24 || clamped.UIFont != "Menlo" {
		t.Fatalf("越界值没有夹回区间：%+v", clamped)
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

// TestNotificationPreferencesPersist 核对设置 › 通知的三个开关真的落盘：出厂全开，
// 关掉后重建 App 仍是关，且与关窗偏好互不影响。
func TestNotificationPreferencesPersist(t *testing.T) {
	dir := t.TempDir()
	first, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("创建 App：%v", err)
	}
	defaults := first.Preferences()
	if !defaults.NotifyTaskDone || !defaults.NotifySound || !defaults.NotifyNeedsResponse {
		t.Fatalf("通知开关出厂应为开：%+v", defaults)
	}
	if defaults.CloseToTray {
		t.Fatalf("关窗偏好出厂应为关：%+v", defaults)
	}

	want := defaults
	want.NotifyTaskDone = false
	want.NotifySound = false
	want.NotifyNeedsResponse = true
	want.CloseToTray = true
	if err := first.UpdatePreferences(want); err != nil {
		t.Fatalf("写入偏好：%v", err)
	}

	second, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("重建 App：%v", err)
	}
	got := second.Preferences()
	if got.NotifyTaskDone || got.NotifySound || !got.NotifyNeedsResponse || !got.CloseToTray {
		t.Fatalf("通知开关与关窗偏好没有持久化：%+v", got)
	}
}

// TestNotificationPreferencesDefaultForOldFile 核对旧偏好文件里没有这三个键时按
// 出厂值（开）处理，写成显式的 false 才是用户关掉的。
func TestNotificationPreferencesDefaultForOldFile(t *testing.T) {
	dir := t.TempDir()
	raw := []byte("{\n  \"theme\": \"暗\",\n  \"notifyTaskDone\": false\n}\n")
	if err := os.WriteFile(filepath.Join(dir, "preferences.json"), raw, 0o600); err != nil {
		t.Fatalf("写偏好文件：%v", err)
	}
	app, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("创建 App：%v", err)
	}
	got := app.Preferences()
	if got.NotifyTaskDone {
		t.Fatalf("文件里显式的 false 应保留：%+v", got)
	}
	if !got.NotifySound || !got.NotifyNeedsResponse {
		t.Fatalf("缺省的键应按出厂值（开）处理：%+v", got)
	}
}

// TestWindowBoundsPerIndex 核对多窗口的几何各记各的：第二个窗口不影响主窗口，
// 主窗口同时写在旧的 Window 字段里；重建 App 后两项都还在。
func TestWindowBoundsPerIndex(t *testing.T) {
	dir := t.TempDir()
	first, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("创建 App：%v", err)
	}
	if err := first.UpdateWindowBoundsFor(0, 10, 20, 1200, 800); err != nil {
		t.Fatalf("写主窗口几何：%v", err)
	}
	if err := first.UpdateWindowBoundsFor(1, 300, 180, 900, 700); err != nil {
		t.Fatalf("写第二个窗口几何：%v", err)
	}

	second, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("重建 App：%v", err)
	}
	prefs := second.Preferences()
	main := prefs.WindowFor(0)
	if main.X != 10 || main.Y != 20 || main.Width != 1200 || main.Height != 800 || !main.HasPosition {
		t.Fatalf("主窗口几何 = %+v，想要 10,20 1200x800", main)
	}
	extra := prefs.WindowFor(1)
	if extra.X != 300 || extra.Y != 180 || extra.Width != 900 || extra.Height != 700 || !extra.HasPosition {
		t.Fatalf("第二个窗口几何 = %+v，想要 300,180 900x700", extra)
	}
	if prefs.Window.Width != 1200 || prefs.Window.X != 10 {
		t.Fatalf("主窗口几何也要写在 Window 里（旧版本回退可读）：%+v", prefs.Window)
	}
	if third := prefs.WindowFor(2); third != (WindowPreferences{}) {
		t.Fatalf("没有记录的窗口索引应返回零值：%+v", third)
	}
}

// TestWindowBoundsLegacyFile 核对旧偏好文件（只有 Window）仍能读回主窗口几何。
func TestWindowBoundsLegacyFile(t *testing.T) {
	dir := t.TempDir()
	raw := []byte("{\n  \"window\": {\"x\": 12, \"y\": 34, \"width\": 1024, \"height\": 768, \"hasPosition\": true}\n}\n")
	if err := os.WriteFile(filepath.Join(dir, "preferences.json"), raw, 0o600); err != nil {
		t.Fatalf("写偏好文件：%v", err)
	}
	app, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("创建 App：%v", err)
	}
	got := app.Preferences().WindowFor(0)
	if got.X != 12 || got.Y != 34 || got.Width != 1024 || got.Height != 768 || !got.HasPosition {
		t.Fatalf("旧文件的窗口几何 = %+v", got)
	}
	if extra := app.Preferences().WindowFor(1); extra != (WindowPreferences{}) {
		t.Fatalf("旧文件没有第二个窗口的几何：%+v", extra)
	}
}
