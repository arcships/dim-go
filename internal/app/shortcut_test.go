package app

import (
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/shortcut"
)

// fakeRegistrar 记录注册与注销，用来覆盖成功、被占用两条分支（不碰平台 API）。
type fakeRegistrar struct {
	mu           sync.Mutex
	registered   []string
	unregistered []string
	all          int
	handler      func()
	err          error
}

func (f *fakeRegistrar) Register(accelerator string, fn func()) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.registered = append(f.registered, accelerator)
	f.handler = fn
	return nil
}

func (f *fakeRegistrar) Unregister(accelerator string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unregistered = append(f.unregistered, accelerator)
	kept := f.registered[:0]
	for _, current := range f.registered {
		if current != accelerator {
			kept = append(kept, current)
		}
	}
	f.registered = kept
}

func (f *fakeRegistrar) UnregisterAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.all++
	f.registered = nil
}

func (f *fakeRegistrar) registeredCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.registered)
}

func (f *fakeRegistrar) unregisteredSnapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.unregistered...)
}

// TestGlobalShortcutRegistersByDefault：默认打开，按默认键位注册；按下的回调
// 交给宿主的动作。
func TestGlobalShortcutRegistersByDefault(t *testing.T) {
	base := newShortcutTestApp(t)
	registrar := &fakeRegistrar{}
	fired := 0
	base.SetGlobalShortcutRegistrar(registrar)
	base.SetGlobalShortcutHandler(func() { fired++ })

	if err := base.ApplyGlobalShortcut(); err != nil {
		t.Fatalf("注册默认键位失败：%v", err)
	}
	status := base.GlobalShortcutStatus()
	if !status.Enabled || !status.Registered {
		t.Fatalf("默认应开启并注册成功：%+v", status)
	}
	if status.Accelerator != shortcut.DefaultAccelerator {
		t.Fatalf("键位 = %q，想要 %q", status.Accelerator, shortcut.DefaultAccelerator)
	}
	if want := shortcut.LabelFor(shortcut.DefaultAccelerator, runtime.GOOS); status.Label != want {
		t.Fatalf("展示键位 = %q，想要 %q", status.Label, want)
	}
	if status.Reason != "" {
		t.Fatalf("成功注册不该带原因：%q", status.Reason)
	}
	if got := registrar.registeredCount(); got != 1 {
		t.Fatalf("注册了 %d 次，想要 1 次", got)
	}
	if registrar.handler == nil {
		t.Fatal("没有交下按键回调")
	}
	registrar.handler()
	if fired != 1 {
		t.Fatalf("按键回调触发 %d 次，想要 1 次", fired)
	}
}

// TestGlobalShortcutRegisterFailureKeepsReason：键位被占用时返回错误，状态里
// 留着原因（界面要靠它给说明，不能静默）。
func TestGlobalShortcutRegisterFailureKeepsReason(t *testing.T) {
	base := newShortcutTestApp(t)
	registrar := &fakeRegistrar{err: errors.New("it may be used by another application (-9868)")}
	base.SetGlobalShortcutRegistrar(registrar)

	err := base.ApplyGlobalShortcut()
	if err == nil {
		t.Fatal("注册失败应当返回错误")
	}
	if !strings.Contains(err.Error(), "全局快捷键") {
		t.Fatalf("错误里没说清是哪一项：%v", err)
	}
	status := base.GlobalShortcutStatus()
	if status.Registered {
		t.Fatal("注册失败时状态不该声称已注册")
	}
	if !strings.Contains(status.Reason, "another application") {
		t.Fatalf("状态里没有注册失败原因：%+v", status)
	}
	if got := registrar.registeredCount(); got != 0 {
		t.Fatalf("失败时不应留下注册记录，得到 %d 条", got)
	}
}

// TestGlobalShortcutToggleOffUnregisters：关掉开关先注销，再按开关打开时重新
// 注册；关着不算失败。
func TestGlobalShortcutToggleOffUnregisters(t *testing.T) {
	base := newShortcutTestApp(t)
	registrar := &fakeRegistrar{}
	base.SetGlobalShortcutRegistrar(registrar)
	if err := base.ApplyGlobalShortcut(); err != nil {
		t.Fatalf("首次注册失败：%v", err)
	}

	prefs := base.Preferences()
	prefs.GlobalShortcutOff = true
	if err := base.UpdatePreferences(prefs); err != nil {
		t.Fatalf("写入偏好：%v", err)
	}
	if err := base.ApplyGlobalShortcut(); err != nil {
		t.Fatalf("关闭开关不该报错：%v", err)
	}
	status := base.GlobalShortcutStatus()
	if status.Enabled || status.Registered {
		t.Fatalf("关掉后状态应为未启用、未注册：%+v", status)
	}
	if got := registrar.unregisteredSnapshot(); len(got) != 1 || got[0] != shortcut.DefaultAccelerator {
		t.Fatalf("关闭时没有注销默认键位：%v", got)
	}

	prefs.GlobalShortcutOff = false
	if err := base.UpdatePreferences(prefs); err != nil {
		t.Fatalf("再次写入偏好：%v", err)
	}
	if err := base.ApplyGlobalShortcut(); err != nil {
		t.Fatalf("重新打开失败：%v", err)
	}
	if !base.GlobalShortcutStatus().Registered {
		t.Fatal("重新打开后应重新注册")
	}
	if got := registrar.registeredCount(); got != 1 {
		t.Fatalf("当前注册条目 = %d，想要 1 条", got)
	}
}

// TestGlobalShortcutChangeAccelerator：改键先注销旧键位，再注册新键位，键位落成
// 规范写法。
func TestGlobalShortcutChangeAccelerator(t *testing.T) {
	base := newShortcutTestApp(t)
	registrar := &fakeRegistrar{}
	base.SetGlobalShortcutRegistrar(registrar)
	if err := base.ApplyGlobalShortcut(); err != nil {
		t.Fatalf("首次注册失败：%v", err)
	}

	prefs := base.Preferences()
	prefs.GlobalShortcutAccelerator = "⌘⇧K"
	if err := base.UpdatePreferences(prefs); err != nil {
		t.Fatalf("写入偏好：%v", err)
	}
	if err := base.ApplyGlobalShortcut(); err != nil {
		t.Fatalf("改键失败：%v", err)
	}
	status := base.GlobalShortcutStatus()
	if status.Accelerator != "Cmd+Shift+K" || !status.Registered {
		t.Fatalf("改键后的状态 = %+v，想要 Cmd+Shift+K 且已注册", status)
	}
	if got := registrar.unregisteredSnapshot(); len(got) != 1 || got[0] != shortcut.DefaultAccelerator {
		t.Fatalf("改键时没有注销旧键位：%v", got)
	}
}

// TestGlobalShortcutInvalidAcceleratorReportsReason：偏好里手写了一个认不出的
// 键位时，撤掉旧注册并把原因交给界面。
func TestGlobalShortcutInvalidAcceleratorReportsReason(t *testing.T) {
	base := newShortcutTestApp(t)
	registrar := &fakeRegistrar{}
	base.SetGlobalShortcutRegistrar(registrar)
	if err := base.ApplyGlobalShortcut(); err != nil {
		t.Fatalf("首次注册失败：%v", err)
	}

	prefs := base.Preferences()
	prefs.GlobalShortcutAccelerator = "K"
	if err := base.UpdatePreferences(prefs); err != nil {
		t.Fatalf("写入偏好：%v", err)
	}
	if err := base.ApplyGlobalShortcut(); err == nil {
		t.Fatal("不合法的键位应当报错")
	}
	status := base.GlobalShortcutStatus()
	if status.Registered {
		t.Fatal("键位不合法时不该还在注册状态")
	}
	if !strings.Contains(status.Reason, "键位无法识别") {
		t.Fatalf("状态里没有键位不合法的原因：%+v", status)
	}
	if got := registrar.unregisteredSnapshot(); len(got) != 1 {
		t.Fatalf("键位不合法时要撤掉旧注册，得到 %v", got)
	}
}

// TestGlobalShortcutPreferencesPersist：开关与键位落盘，重建 App 后恢复，且
// 状态读出来就是恢复后的开关与键位。
func TestGlobalShortcutPreferencesPersist(t *testing.T) {
	dir := t.TempDir()
	first, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("创建 App：%v", err)
	}
	prefs := first.Preferences()
	prefs.GlobalShortcutOff = true
	prefs.GlobalShortcutAccelerator = "Cmd+Shift+K"
	if err := first.UpdatePreferences(prefs); err != nil {
		t.Fatalf("写入偏好：%v", err)
	}

	second, err := New(Config{DataDir: dir})
	if err != nil {
		t.Fatalf("重建 App：%v", err)
	}
	got := second.Preferences()
	if !got.GlobalShortcutOff || got.GlobalShortcutAccelerator != "Cmd+Shift+K" {
		t.Fatalf("偏好没有恢复：%+v", got)
	}
	status := second.GlobalShortcutStatus()
	if status.Enabled || status.Accelerator != "Cmd+Shift+K" || status.Label == "" {
		t.Fatalf("恢复后的状态 = %+v，想要关闭且键位 Cmd+Shift+K", status)
	}
}

// TestReleaseGlobalShortcutsUnregistersAll：退出前注销全部系统级快捷键。
func TestReleaseGlobalShortcutsUnregistersAll(t *testing.T) {
	base := newShortcutTestApp(t)
	registrar := &fakeRegistrar{}
	base.SetGlobalShortcutRegistrar(registrar)
	if err := base.ApplyGlobalShortcut(); err != nil {
		t.Fatalf("注册失败：%v", err)
	}
	if err := base.ReleaseGlobalShortcuts(); err != nil {
		t.Fatalf("注销失败：%v", err)
	}
	if registrar.all != 1 {
		t.Fatalf("UnregisterAll 调用 %d 次，想要 1 次", registrar.all)
	}
	if base.GlobalShortcutStatus().Registered {
		t.Fatal("注销后状态不该还是已注册")
	}
}

// newShortcutTestApp 建一个只装了本地状态的集成层。
func newShortcutTestApp(t *testing.T) *App {
	t.Helper()
	base, err := New(Config{DataDir: t.TempDir(), ClientName: "shortcut-test"})
	if err != nil {
		t.Fatalf("创建 App：%v", err)
	}
	return base
}
