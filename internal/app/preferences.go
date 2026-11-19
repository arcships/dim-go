package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Preferences 是客户端自己的本地偏好。它不属于 App Server 的事实，单独落盘在
// 客户端数据目录的 preferences.json；读取失败按默认值启动，写入失败只记诊断。
type Preferences struct {
	// Theme 取「跟随系统」「明」「暗」。
	Theme string `json:"theme"`
	// MotionOff 关闭动效。
	MotionOff bool `json:"motionOff"`
	// TimeZone 取「跟随系统」或 IANA 时区名。
	TimeZone string `json:"timeZone"`
	// SidebarWidth 是左栏宽度（逻辑像素）；0 表示用默认值。
	SidebarWidth int `json:"sidebarWidth"`
	// Workspaces 是本端登记的附加工作区目录（左栏「添加工作区」选中的目录）。
	// 它只是本地登记，不是 App Server 授权的项目根：新建会话的 cwd 仍只从
	// 服务端的项目清单取。
	Workspaces []string `json:"workspaces,omitempty"`
	// HooksProjectPath 是设置 › Hooks 里选中的项目目录（本端登记，用于展示与
	// 后续按目录定位项目钩子）；空表示没有。
	HooksProjectPath string `json:"hooksProjectPath,omitempty"`
	// GlobalShortcutOff 关闭全局唤起快捷键。缺省 false（开启）：旧偏好文件里
	// 没有这一项时按开启处理。
	GlobalShortcutOff bool `json:"globalShortcutOff,omitempty"`
	// GlobalShortcutAccelerator 是全局唤起快捷键的键位（如
	// CmdOrCtrl+Shift+Space）；空表示默认键位。
	GlobalShortcutAccelerator string `json:"globalShortcutAccelerator,omitempty"`
	// NotifyTaskDone 是设置 › 通知的「任务完成或失败时通知」：轮完成或失败时发
	// 系统通知。缺省 true：与参考产品的出厂开关一致。
	NotifyTaskDone bool `json:"notifyTaskDone"`
	// NotifySound 是设置 › 通知的「通知声音」：发通知时带系统提示音。
	NotifySound bool `json:"notifySound"`
	// NotifyNeedsResponse 是设置 › 通知的「需要我回应时通知」：审批或提问待决时
	// 发系统通知。
	NotifyNeedsResponse bool `json:"notifyNeedsResponse"`
	// CloseToTray 是关窗行为：为真时窗口隐藏、进程留在托盘；为假时按退出处理。
	// 缺省 false：与参考产品一致，关窗即退出。
	CloseToTray bool `json:"closeToTray"`
	// Window 是主窗口（索引 0）的尺寸与位置；HasPosition 为假时按居中处理。
	// 主窗口同时写在 Windows[0] 与这里：旧偏好文件只有这一项时仍能读回主窗口位置。
	Window WindowPreferences `json:"window"`
	// Windows 是每个窗口的几何，按窗口索引存（0 是主窗口）。多窗口各记各的位置
	// 与尺寸；关掉的窗口留下的槽位由下一个新窗口沿用。
	Windows []WindowPreferences `json:"windows,omitempty"`
}

// WindowPreferences 是窗口几何。
type WindowPreferences struct {
	X           int  `json:"x"`
	Y           int  `json:"y"`
	Width       int  `json:"width"`
	Height      int  `json:"height"`
	HasPosition bool `json:"hasPosition"`
}

// DefaultPreferences 是缺省偏好。
func DefaultPreferences() Preferences {
	return Preferences{
		Theme:               "跟随系统",
		TimeZone:            "跟随系统",
		NotifyTaskDone:      true,
		NotifySound:         true,
		NotifyNeedsResponse: true,
	}
}

// preferencesPath 是偏好文件位置。
func preferencesPath(dataDir string) string {
	return filepath.Join(dataDir, "preferences.json")
}

// LoadPreferences 读取偏好；文件不存在时返回默认值，损坏时返回错误与默认值。
func LoadPreferences(dataDir string) (Preferences, error) {
	defaults := DefaultPreferences()
	path := preferencesPath(dataDir)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaults, nil
		}
		return defaults, err
	}
	var prefs Preferences
	if err := json.Unmarshal(raw, &prefs); err != nil {
		return defaults, err
	}
	// 三个通知开关出厂为开：旧偏好文件里没有这些键时按开处理，写成显式的 false
	// 才是用户关掉的。
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err == nil {
		if _, ok := keys["notifyTaskDone"]; !ok {
			prefs.NotifyTaskDone = defaults.NotifyTaskDone
		}
		if _, ok := keys["notifySound"]; !ok {
			prefs.NotifySound = defaults.NotifySound
		}
		if _, ok := keys["notifyNeedsResponse"]; !ok {
			prefs.NotifyNeedsResponse = defaults.NotifyNeedsResponse
		}
	}
	return normalizePreferences(prefs), nil
}

// SavePreferences 原子写入偏好。
func SavePreferences(dataDir string, prefs Preferences) error {
	if strings.TrimSpace(dataDir) == "" {
		return errors.New("缺少客户端数据目录")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(normalizePreferences(prefs), "", "  ")
	if err != nil {
		return err
	}
	path := preferencesPath(dataDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// normalizePreferences 给空字段补默认值；不改用户给出的其他值。
func normalizePreferences(prefs Preferences) Preferences {
	def := DefaultPreferences()
	if strings.TrimSpace(prefs.Theme) == "" {
		prefs.Theme = def.Theme
	}
	if strings.TrimSpace(prefs.TimeZone) == "" {
		prefs.TimeZone = def.TimeZone
	}
	prefs.HooksProjectPath = strings.TrimSpace(prefs.HooksProjectPath)
	prefs.GlobalShortcutAccelerator = strings.TrimSpace(prefs.GlobalShortcutAccelerator)
	prefs.Workspaces = normalizeWorkspaceList(prefs.Workspaces)
	prefs.Window = normalizeWindowBounds(prefs.Window)
	for i := range prefs.Windows {
		prefs.Windows[i] = normalizeWindowBounds(prefs.Windows[i])
	}
	// 主窗口的几何两处都写：旧版本只读 Window，新版本按索引读 Windows。
	if len(prefs.Windows) > 0 && hasBounds(prefs.Windows[0]) {
		prefs.Window = prefs.Windows[0]
	} else if hasBounds(prefs.Window) {
		prefs.Windows = growBoundsList(prefs.Windows, 0)
		prefs.Windows[0] = prefs.Window
	}
	return prefs
}

// normalizeWindowBounds 把负的宽高按 0（用默认值）处理。
func normalizeWindowBounds(bounds WindowPreferences) WindowPreferences {
	if bounds.Width < 0 {
		bounds.Width = 0
	}
	if bounds.Height < 0 {
		bounds.Height = 0
	}
	return bounds
}

// hasBounds 表示这条几何有实际记录。
func hasBounds(bounds WindowPreferences) bool {
	return bounds.HasPosition || bounds.Width > 0 || bounds.Height > 0
}

// growBoundsList 把几何列表补到至少 index+1 项。
func growBoundsList(list []WindowPreferences, index int) []WindowPreferences {
	for len(list) <= index {
		list = append(list, WindowPreferences{})
	}
	return list
}

// WindowFor 返回某个窗口索引的几何；没有记录时返回零值（调用方按居中与默认
// 尺寸处理）。索引 0 在主窗口那一项缺失时回落到旧的 Window 字段。
func (p Preferences) WindowFor(index int) WindowPreferences {
	if index < 0 {
		return WindowPreferences{}
	}
	if index < len(p.Windows) && hasBounds(p.Windows[index]) {
		return p.Windows[index]
	}
	if index == 0 {
		return p.Window
	}
	return WindowPreferences{}
}

// normalizeWorkspaceList 去掉空项与重复项，保留登记顺序。
func normalizeWorkspaceList(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
}

// Preferences 返回当前本地偏好的副本。
func (a *App) Preferences() Preferences {
	a.mu.Lock()
	defer a.mu.Unlock()
	prefs := a.prefs
	prefs.Workspaces = append([]string(nil), a.prefs.Workspaces...)
	prefs.Windows = append([]WindowPreferences(nil), a.prefs.Windows...)
	return prefs
}

// UpdatePreferences 更新并落盘偏好；内存值先应用，写盘失败只记诊断。
func (a *App) UpdatePreferences(prefs Preferences) error {
	prefs = normalizePreferences(prefs)
	a.mu.Lock()
	a.prefs = prefs
	dataDir := a.cfg.DataDir
	a.mu.Unlock()
	if err := SavePreferences(dataDir, prefs); err != nil {
		a.record("本地偏好写入失败: %v", err)
		return err
	}
	return nil
}

// UpdateWindowBounds 记录主窗口（索引 0）的尺寸与位置。
func (a *App) UpdateWindowBounds(x, y, width, height int) error {
	return a.UpdateWindowBoundsFor(0, x, y, width, height)
}

// UpdateWindowBoundsFor 记录某个窗口索引的尺寸与位置；主窗口同时写进 Window
// 与 Windows[0]，保证旧版本回退时仍读得到主窗口位置。
func (a *App) UpdateWindowBoundsFor(index, x, y, width, height int) error {
	if index < 0 {
		index = 0
	}
	prefs := a.Preferences()
	bounds := WindowPreferences{X: x, Y: y, Width: width, Height: height, HasPosition: true}
	prefs.Windows = growBoundsList(prefs.Windows, index)
	prefs.Windows[index] = bounds
	if index == 0 {
		prefs.Window = bounds
	}
	return a.UpdatePreferences(prefs)
}

// loadPreferences 在启动时装载偏好；失败按默认值并记诊断。
func (a *App) loadPreferences() {
	prefs, err := LoadPreferences(a.cfg.DataDir)
	if err != nil {
		a.record("本地偏好读取失败，按默认值启动: %v", err)
	}
	a.mu.Lock()
	a.prefs = prefs
	a.mu.Unlock()
}
