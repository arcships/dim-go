package ui

import (
	"strings"

	gui "github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/shortcut"
)

// 本文件把「设置 › 快捷键」里的键位写法换成框架的修饰键与主键。
//
// 键位写法与出厂值在 internal/shortcut（叶包，集成层与视图层共用同一套写法），
// 这里只做「Electron 写法 → gui.Modifiers + gui.Key」这一步换算。换算不出来的
// 键位一律当作按不到：偏好里混进坏值时界面照旧用出厂值（shortcut.Binding 已经
// 把坏值挡在外面），这里再挡一次，不猜一个别的键去占位。

// keySymbols 是规范写法的主键名 → 框架按键。只有这张表里的键能改成界面内快捷键；
// 表外的键（例如 NumDec）认不出，改键时按「无法识别」拒绝。
var keySymbols = map[string]gui.Key{
	"Enter": gui.KeyEnter, "Escape": gui.KeyEscape, "Tab": gui.KeyTab, "Space": gui.KeySpace,
	"Backspace": gui.KeyBackspace, "Delete": gui.KeyDelete, "Insert": gui.KeyInsert,
	"Home": gui.KeyHome, "End": gui.KeyEnd, "PageUp": gui.KeyPageUp, "PageDown": gui.KeyPageDown,
	"Left": gui.KeyLeft, "Right": gui.KeyRight, "Up": gui.KeyUp, "Down": gui.KeyDown,
	"F1": gui.KeyF1, "F2": gui.KeyF2, "F3": gui.KeyF3, "F4": gui.KeyF4, "F5": gui.KeyF5,
	"F6": gui.KeyF6, "F7": gui.KeyF7, "F8": gui.KeyF8, "F9": gui.KeyF9, "F10": gui.KeyF10,
	"F11": gui.KeyF11, "F12": gui.KeyF12,
	"0": gui.Key0, "1": gui.Key1, "2": gui.Key2, "3": gui.Key3, "4": gui.Key4,
	"5": gui.Key5, "6": gui.Key6, "7": gui.Key7, "8": gui.Key8, "9": gui.Key9,
	"A": gui.KeyA, "B": gui.KeyB, "C": gui.KeyC, "D": gui.KeyD, "E": gui.KeyE,
	"F": gui.KeyF, "G": gui.KeyG, "H": gui.KeyH, "I": gui.KeyI, "J": gui.KeyJ,
	"K": gui.KeyK, "L": gui.KeyL, "M": gui.KeyM, "N": gui.KeyN, "O": gui.KeyO,
	"P": gui.KeyP, "Q": gui.KeyQ, "R": gui.KeyR, "S": gui.KeyS, "T": gui.KeyT,
	"U": gui.KeyU, "V": gui.KeyV, "W": gui.KeyW, "X": gui.KeyX, "Y": gui.KeyY,
	"Z": gui.KeyZ,
	"-": gui.KeyMinus, "=": gui.KeyEqual, ",": gui.KeyComma, ".": gui.KeyPeriod,
	"/": gui.KeySlash, ";": gui.KeySemicolon, "'": gui.KeyQuote,
	"[": gui.KeyBracketLeft, "]": gui.KeyBracketRight, "\\": gui.KeyBackslash,
	"`": gui.KeyBackquote,
}

// keyModifierNames 是规范写法里的修饰键。CmdOrCtrl 与 Cmd 在 macOS 上是 ⌘，
// 在别的平台上是 Ctrl（与框架的 gui.Cmd 同一口径）。
var keyModifierNames = map[string]gui.Modifiers{
	"CmdOrCtrl": gui.Cmd,
	"Cmd":       gui.Super,
	"Ctrl":      gui.Ctrl,
	"Alt":       gui.Alt,
	"Shift":     gui.Shift,
}

// parseAccelerator 把规范写法（如 `CmdOrCtrl+Shift+B`）换成框架的修饰键与主键。
// 第三个返回值为假表示这个键位本版按不到（表外的主键、没有主键、没有修饰键）。
func parseAccelerator(accelerator string) (gui.Modifiers, gui.Key, bool) {
	text := strings.TrimSpace(accelerator)
	if text == "" {
		return 0, gui.KeyUnknown, false
	}
	var mods gui.Modifiers
	key := gui.KeyUnknown
	for _, part := range strings.Split(text, "+") {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		if mod, ok := keyModifierNames[token]; ok {
			mods |= mod
			continue
		}
		sym, ok := keySymbols[token]
		if !ok {
			return 0, gui.KeyUnknown, false
		}
		key = sym
	}
	if key == gui.KeyUnknown || mods == 0 {
		return 0, gui.KeyUnknown, false
	}
	return mods, key, true
}

// keybindingAccel 是某个界面内动作当前的键位：改过的取本地镜像，没改过的取出厂值。
func (a *App) keybindingAccel(id string) string {
	return shortcut.Binding(a.keybindings, id)
}

// keybindingHit 判断某个界面内动作的键位这一帧是否被按下。键位认不出来时返回假：
// 按不到的键不冒充能按。
func (a *App) keybindingHit(c *gui.Context, id string) bool {
	mods, key, ok := parseAccelerator(a.keybindingAccel(id))
	if !ok {
		return false
	}
	return c.Shortcut(mods, key)
}
