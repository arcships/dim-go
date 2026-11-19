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
	// Window 是窗口尺寸与位置；HasPosition 为假时按居中处理。
	Window WindowPreferences `json:"window"`
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
	return Preferences{Theme: "跟随系统", TimeZone: "跟随系统"}
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
	if prefs.Window.Width < 0 {
		prefs.Window.Width = 0
	}
	if prefs.Window.Height < 0 {
		prefs.Window.Height = 0
	}
	return prefs
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

// UpdateWindowBounds 记录窗口尺寸与位置。
func (a *App) UpdateWindowBounds(x, y, width, height int) error {
	prefs := a.Preferences()
	prefs.Window = WindowPreferences{
		X: x, Y: y, Width: width, Height: height, HasPosition: true,
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
