package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/shortcut"
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
	// UpdateFeedURL 是设置 › 关于里填写的更新源（latest.json 的地址或基址）。
	// 空表示未配置：自动更新关闭，界面显示「未配置更新源」，不假装已是最新。
	UpdateFeedURL string `json:"updateFeedURL,omitempty"`
	// PluginFeedURL 是插件市场的上游源地址（返回插件清单的 JSON）。空表示未
	// 配置：插件市场整块禁用并写「未配置插件源」，不画样例条目。
	PluginFeedURL string `json:"pluginFeedURL,omitempty"`
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
	// View 是跨重启记住的视图状态（左栏收起、视图口径、分组折叠、右栏开合与
	// 标签页、文件页筛选、改动页范围）。它不是服务端事实，也不含任何内容。
	View ViewPreferences `json:"view"`
	// Appearance 是设置 › 外观里会真正影响渲染的几项本地偏好：界面字体与字号、
	// 代码字体与字号、代码配色。它不是服务端事实。
	Appearance AppearancePreferences `json:"appearance"`
	// Keybindings 是设置 › 快捷键里「通用」「对话」两组改过的键位（动作 ID →
	// Electron 写法）。只存与出厂值不同的项：删掉一项就是重置。它不是服务端事实，
	// 也不影响全局唤起那一条（那条单独存）。
	Keybindings map[string]string `json:"keybindings,omitempty"`
}

// AppearancePreferences 是「设置 › 外观」里会真正影响渲染的本地偏好。
//
// 空串 / 0 表示用默认：字体空 = 系统 UI 字体栈或系统等宽，字号 0 = 参考的默认
// 档（界面按平台默认字号，代码 13）。
type AppearancePreferences struct {
	// UIFont 是界面字体族；空表示系统 UI 字体栈。
	UIFont string `json:"uiFont,omitempty"`
	// CodeFont 是代码字体族；空表示系统等宽。
	CodeFont string `json:"codeFont,omitempty"`
	// UIScale 是界面字号的倍率（参考五档：0.85 / 0.93 / 1 / 1.08 / 1.16）。
	UIScale float32 `json:"uiScale,omitempty"`
	// CodeSize 是代码字号（DIP）；0 表示默认 13。
	CodeSize float32 `json:"codeSize,omitempty"`
	// CodeTheme 是代码配色名；空表示 GitHub（本版内置的一套）。
	CodeTheme string `json:"codeTheme,omitempty"`
}

// ViewPreferences 是视图状态在偏好文件里的形状。
type ViewPreferences struct {
	SidebarCollapsed    bool              `json:"sidebarCollapsed,omitempty"`
	SidebarFlat         bool              `json:"sidebarFlat,omitempty"`
	SidebarByCreated    bool              `json:"sidebarByCreated,omitempty"`
	CollapsedGroups     []string          `json:"collapsedGroups,omitempty"`
	PinnedGroups        []string          `json:"pinnedGroups,omitempty"`
	WorkspaceNames      map[string]string `json:"workspaceNames,omitempty"`
	RightOpen           bool              `json:"rightOpen,omitempty"`
	RightTab            int               `json:"rightTab,omitempty"`
	RightTabs           []int             `json:"rightTabs,omitempty"`
	RightExpanded       bool              `json:"rightExpanded,omitempty"`
	RightWidth          int               `json:"rightWidth,omitempty"`
	FilesFilter         string            `json:"filesFilter,omitempty"`
	ChangesScope        int               `json:"changesScope,omitempty"`
	ChangesGroupsClosed []string          `json:"changesGroupsClosed,omitempty"`
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
	prefs.UpdateFeedURL = strings.TrimSpace(prefs.UpdateFeedURL)
	prefs.GlobalShortcutAccelerator = strings.TrimSpace(prefs.GlobalShortcutAccelerator)
	prefs.Workspaces = normalizeWorkspaceList(prefs.Workspaces)
	prefs.Window = normalizeWindowBounds(prefs.Window)
	for i := range prefs.Windows {
		prefs.Windows[i] = normalizeWindowBounds(prefs.Windows[i])
	}
	prefs.View = normalizeViewPreferences(prefs.View)
	prefs.Appearance = normalizeAppearancePreferences(prefs.Appearance)
	prefs.Keybindings = shortcut.NormalizeBindings(prefs.Keybindings)
	return prefs
}

// normalizeAppearancePreferences 把外观偏好夹到可用区间：字体名去空白，字号倍率
// 落在 0.8–1.3，代码字号落在 10–24，代码配色只认已知的几套。
func normalizeAppearancePreferences(look AppearancePreferences) AppearancePreferences {
	look.UIFont = strings.TrimSpace(look.UIFont)
	look.CodeFont = strings.TrimSpace(look.CodeFont)
	look.CodeTheme = strings.TrimSpace(look.CodeTheme)
	if look.UIScale != 0 {
		if look.UIScale < 0.8 {
			look.UIScale = 0.8
		}
		if look.UIScale > 1.3 {
			look.UIScale = 1.3
		}
	}
	if look.CodeSize != 0 {
		if look.CodeSize < 10 {
			look.CodeSize = 10
		}
		if look.CodeSize > 24 {
			look.CodeSize = 24
		}
	}
	return look
}

// normalizeViewPreferences 去掉视图状态里没有意义的取值：负的标签与宽度按 0，
// 重复的分组标识只留一份，保持原顺序。
func normalizeViewPreferences(view ViewPreferences) ViewPreferences {
	if view.RightTab < 0 {
		view.RightTab = 0
	}
	if view.RightWidth < 0 {
		view.RightWidth = 0
	}
	view.CollapsedGroups = normalizeWorkspaceList(view.CollapsedGroups)
	view.PinnedGroups = normalizeWorkspaceList(view.PinnedGroups)
	view.ChangesGroupsClosed = normalizeWorkspaceList(view.ChangesGroupsClosed)
	if len(view.WorkspaceNames) == 0 {
		view.WorkspaceNames = nil
	}
	seen := make(map[int]bool, len(view.RightTabs))
	tabs := make([]int, 0, len(view.RightTabs))
	for _, tab := range view.RightTabs {
		if tab < 0 || seen[tab] {
			continue
		}
		seen[tab] = true
		tabs = append(tabs, tab)
	}
	if len(tabs) == 0 {
		view.RightTabs = nil
	} else {
		view.RightTabs = tabs
	}
	return view
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
	prefs.View.CollapsedGroups = append([]string(nil), a.prefs.View.CollapsedGroups...)
	prefs.View.PinnedGroups = append([]string(nil), a.prefs.View.PinnedGroups...)
	if a.prefs.View.WorkspaceNames != nil {
		prefs.View.WorkspaceNames = make(map[string]string, len(a.prefs.View.WorkspaceNames))
		for id, name := range a.prefs.View.WorkspaceNames {
			prefs.View.WorkspaceNames[id] = name
		}
	}
	prefs.View.RightTabs = append([]int(nil), a.prefs.View.RightTabs...)
	prefs.View.ChangesGroupsClosed = append([]string(nil), a.prefs.View.ChangesGroupsClosed...)
	if a.prefs.Keybindings != nil {
		prefs.Keybindings = make(map[string]string, len(a.prefs.Keybindings))
		for id, binding := range a.prefs.Keybindings {
			prefs.Keybindings[id] = binding
		}
	}
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
