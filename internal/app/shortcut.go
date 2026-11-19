package app

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/arcships/dimagent-mygo/internal/shortcut"
)

// GlobalShortcutRegistrar 是系统级快捷键的注册入口。
//
// mygo 的 GlobalShortcut 满足它；实现方会把注册与注销切到主线程执行，所以调用
// 方必须假定它会等主线程。测试用假实现覆盖注册成功与被占用两条分支。
type GlobalShortcutRegistrar interface {
	// Register 注册一个系统级快捷键，按下时调用 fn，返回失败原因。
	Register(accelerator string, fn func()) error
	// Unregister 注销一个快捷键；没注册过是空操作。
	Unregister(accelerator string)
	// UnregisterAll 注销本进程注册的全部快捷键。
	UnregisterAll()
}

// GlobalShortcutStatus 是全局唤起快捷键的本端状态：偏好里的开关与键位，加上
// 这次进程内注册的结果。它不是 App Server 的事实。
type GlobalShortcutStatus struct {
	// Enabled 是设置里的开关。
	Enabled bool
	// Accelerator 是规范化后的键位（如 CmdOrCtrl+Shift+Space）。
	Accelerator string
	// Label 是给人看的键位（macOS 上是 ⌘⇧Space）。
	Label string
	// Registered 表示系统已经接受了这次注册。
	Registered bool
	// Reason 是注册失败的原因；空表示没有失败。
	Reason string
}

// globalShortcut 是注册侧的运行时状态。
//
// 锁的边界：applyMu 串行化注册与注销，只在注册侧使用；status 用原子指针发布，
// 渲染线程读它时不取任何锁。注册会阻塞等主线程，若渲染线程也要取同一把锁，
// 就会和正在渲染的帧互相等死（帧里要读状态）。
type globalShortcut struct {
	applyMu    sync.Mutex
	registrar  GlobalShortcutRegistrar
	handler    func()
	registered string
	status     atomic.Pointer[GlobalShortcutStatus]
}

// SetGlobalShortcutRegistrar 装上系统级快捷键的注册入口。宿主在窗口就绪后给；
// nil 表示本进程没有注册能力（例如只有集成层的测试）。
func (a *App) SetGlobalShortcutRegistrar(registrar GlobalShortcutRegistrar) {
	a.shortcut.applyMu.Lock()
	defer a.shortcut.applyMu.Unlock()
	a.shortcut.registrar = registrar
}

// SetGlobalShortcutHandler 装下快捷键按下时要执行的宿主动作（把窗口带到前台并
// 聚焦输入区）。它由宿主给出：集成层不碰窗口。
func (a *App) SetGlobalShortcutHandler(fn func()) {
	a.shortcut.applyMu.Lock()
	defer a.shortcut.applyMu.Unlock()
	a.shortcut.handler = fn
}

// ApplyGlobalShortcut 按当前偏好重新注册全局唤起快捷键：先注销上一次的，再按
// 开关决定是否注册。开关关着不算失败；注册不上（键位被占用、键位不合法）返回
// 错误，并把原因留在 GlobalShortcutStatus 里给界面显示。
func (a *App) ApplyGlobalShortcut() error {
	a.shortcut.applyMu.Lock()
	defer a.shortcut.applyMu.Unlock()
	return a.applyGlobalShortcutLocked()
}

func (a *App) applyGlobalShortcutLocked() error {
	prefs := a.Preferences()
	enabled := !prefs.GlobalShortcutOff
	status := GlobalShortcutStatus{Enabled: enabled}

	accelerator := shortcut.DefaultAccelerator
	if raw := strings.TrimSpace(prefs.GlobalShortcutAccelerator); raw != "" {
		normalized, err := shortcut.Normalize(raw)
		if err != nil {
			// 键位不合法：先撤掉旧的注册，免得屏上写着开关打开、实际按的是旧键。
			a.unregisterGlobalShortcutLocked()
			status.Accelerator = raw
			status.Label = raw
			status.Reason = "键位无法识别：" + err.Error()
			a.shortcut.status.Store(&status)
			a.record("全局快捷键键位不合法（%s）：%v", raw, err)
			return fmt.Errorf("全局快捷键键位无法识别：%w", err)
		}
		accelerator = normalized
	}
	status.Accelerator = accelerator
	status.Label = shortcut.Label(accelerator)

	a.unregisterGlobalShortcutLocked()
	if !enabled {
		a.shortcut.status.Store(&status)
		return nil
	}
	if a.shortcut.registrar == nil {
		// 还没有注册入口（宿主未就绪）：只发布状态，不算失败。
		a.shortcut.status.Store(&status)
		return nil
	}
	handler := a.shortcut.handler
	if handler == nil {
		handler = func() {}
	}
	if err := a.shortcut.registrar.Register(accelerator, handler); err != nil {
		status.Reason = err.Error()
		a.shortcut.status.Store(&status)
		a.record("全局快捷键注册失败（%s）：%v", accelerator, err)
		return fmt.Errorf("无法注册全局快捷键 %s：%w", status.Label, err)
	}
	a.shortcut.registered = accelerator
	status.Registered = true
	a.shortcut.status.Store(&status)
	return nil
}

// unregisterGlobalShortcutLocked 注销上一次注册的键位；调用方持 applyMu。
func (a *App) unregisterGlobalShortcutLocked() {
	if a.shortcut.registered == "" {
		return
	}
	if a.shortcut.registrar != nil {
		a.shortcut.registrar.Unregister(a.shortcut.registered)
	}
	a.shortcut.registered = ""
}

// ReleaseGlobalShortcuts 注销本进程注册的全部系统级快捷键（退出前调用）。
//
// 必须在主线程上调用：注销会切到主线程执行，而从后台协程调用会等主线程，退出
// 路径又在主线程上等收尾，两边会互相等。
func (a *App) ReleaseGlobalShortcuts() error {
	a.shortcut.applyMu.Lock()
	defer a.shortcut.applyMu.Unlock()
	if a.shortcut.registrar == nil {
		return nil
	}
	a.shortcut.registrar.UnregisterAll()
	a.shortcut.registered = ""
	status := a.currentGlobalShortcutStatus()
	status.Registered = false
	status.Reason = ""
	a.shortcut.status.Store(&status)
	return nil
}

// GlobalShortcutStatus 返回全局唤起快捷键的当前状态；还没应用过时按偏好给出
// 开关与键位（启动第一帧也要能画出设置项）。
func (a *App) GlobalShortcutStatus() GlobalShortcutStatus {
	return a.currentGlobalShortcutStatus()
}

func (a *App) currentGlobalShortcutStatus() GlobalShortcutStatus {
	if status := a.shortcut.status.Load(); status != nil {
		return *status
	}
	prefs := a.Preferences()
	accelerator := shortcut.DefaultAccelerator
	if raw := strings.TrimSpace(prefs.GlobalShortcutAccelerator); raw != "" {
		accelerator = raw
	}
	return GlobalShortcutStatus{
		Enabled:     !prefs.GlobalShortcutOff,
		Accelerator: accelerator,
		Label:       shortcut.Label(accelerator),
	}
}
