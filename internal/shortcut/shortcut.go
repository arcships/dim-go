// Package shortcut 是全局唤起快捷键的键位写法：默认值、输入规范化与展示。
//
// 为什么单独成包：同一套写法有两处要用——集成层（internal/app 的偏好落盘与
// 系统级注册）与视图层（internal/ui 的设置 › 快捷键）。ui 不 import app（A 领域
// 边界），所以放在这个只依赖标准库的叶包里。本包只做文本换算，不注册、不读
// 偏好、不碰窗口。
package shortcut

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// DefaultAccelerator 是全局唤起的默认键位。Electron 风格写法；macOS 上
// CmdOrCtrl 解析为 ⌘，展示为 ⌘⇧Space。
const DefaultAccelerator = "CmdOrCtrl+Shift+Space"

// modifierOrder 是规范化后修饰键的输出顺序：固定的顺序让同一组键位只有一种
// 写法（偏好里存下来的值不会因为书写顺序来回变）。
var modifierOrder = []string{"CmdOrCtrl", "Cmd", "Ctrl", "Alt", "Shift"}

// modifierAliases 是修饰键的各种写法（大小写不敏感）。
var modifierAliases = map[string]string{
	"cmdorctrl": "CmdOrCtrl", "commandorcontrol": "CmdOrCtrl",
	"cmd": "Cmd", "command": "Cmd", "super": "Cmd", "meta": "Cmd", "win": "Cmd",
	"ctrl": "Ctrl", "control": "Ctrl",
	"alt": "Alt", "option": "Alt", "altgr": "Alt",
	"shift": "Shift",
}

// keyAliases 是具名主键的规范写法。
var keyAliases = map[string]string{
	"plus":       "+",
	"space":      "Space",
	"tab":        "Tab",
	"capslock":   "CapsLock",
	"numlock":    "NumLock",
	"scrolllock": "ScrollLock",
	"backspace":  "Backspace",
	"delete":     "Delete", "del": "Delete",
	"insert": "Insert", "ins": "Insert",
	"return": "Enter", "enter": "Enter",
	"up": "Up", "arrowup": "Up",
	"down": "Down", "arrowdown": "Down",
	"left": "Left", "arrowleft": "Left",
	"right": "Right", "arrowright": "Right",
	"home":     "Home",
	"end":      "End",
	"pageup":   "PageUp",
	"pagedown": "PageDown",
	"escape":   "Escape", "esc": "Escape",
	"printscreen": "PrintScreen",
	"numdec":      "NumDec", "numadd": "NumAdd", "numsub": "NumSub",
	"nummult": "NumMult", "numdiv": "NumDiv",
}

// glyphs 是 macOS 的键位写法到规范写法的换算。录制态与参考产品的中文界面都用
// 这套字形，用户直接在输入框里敲 ⌘⇧Space 也应认得。
var glyphs = strings.NewReplacer(
	"⌘", "Cmd+",
	"⌃", "Ctrl+",
	"⌥", "Alt+",
	"⇧", "Shift+",
	"↵", "Enter+",
	"⌫", "Backspace+",
	"⎋", "Escape+",
	"⇥", "Tab+",
	"␣", "Space+",
	"↑", "Up+",
	"↓", "Down+",
	"←", "Left+",
	"→", "Right+",
)

// Normalize 把用户输入的键位换成规范写法（如 Cmd+Shift+Space）。
//
// 接受 Electron 风格的 "Cmd+Shift+Space"、"cmdorctrl+shift+space" 与 macOS
// 字形 "⌘⇧Space"。返回错误时不猜：空键位、缺主键、缺修饰键、认不出的名字都
// 拒绝——全局快捷键至少要有一个修饰键，否则一个裸键会吃掉整个系统的输入。
func Normalize(input string) (string, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return "", fmt.Errorf("键位不能为空")
	}
	parts := splitParts(glyphs.Replace(text))
	seen := make(map[string]bool, len(parts))
	var modifiers []string
	key := ""
	for i, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			return "", fmt.Errorf("键位写法不完整：%q", input)
		}
		last := i == len(parts)-1
		if modifier, ok := modifierAliases[strings.ToLower(token)]; ok {
			if last {
				return "", fmt.Errorf("键位缺少主键：%q", input)
			}
			if !seen[modifier] {
				seen[modifier] = true
				modifiers = append(modifiers, modifier)
			}
			continue
		}
		if !last {
			return "", fmt.Errorf("无法识别的修饰键 %q", token)
		}
		name, ok := keyName(token)
		if !ok {
			return "", fmt.Errorf("无法识别的按键 %q", token)
		}
		key = name
	}
	if key == "" {
		return "", fmt.Errorf("键位缺少主键：%q", input)
	}
	if len(modifiers) == 0 {
		return "", fmt.Errorf("全局快捷键至少要有一个修饰键")
	}
	ordered := make([]string, 0, len(modifierOrder))
	for _, name := range modifierOrder {
		if seen[name] {
			ordered = append(ordered, name)
		}
	}
	return strings.Join(append(ordered, key), "+"), nil
}

// keyName 把主键换成规范写法：具名键查别名表，单个字符保持原样（字母转大写）。
func keyName(token string) (string, bool) {
	lower := strings.ToLower(token)
	if name, ok := keyAliases[lower]; ok {
		return name, true
	}
	if len([]rune(token)) == 1 {
		// 单字符键（a、1、,）：字母统一大写，符号原样保留。
		return strings.ToUpper(token), true
	}
	if len(lower) >= 2 && lower[0] == 'f' {
		if n, err := strconv.Atoi(lower[1:]); err == nil && n >= 1 && n <= 24 {
			return "F" + strconv.Itoa(n), true
		}
	}
	return "", false
}

// splitParts 按 "+" 拆键位，允许 "+" 自己当主键（如 "Cmd++"）。
func splitParts(text string) []string {
	if text == "+" {
		return []string{"+"}
	}
	if strings.HasSuffix(text, "++") {
		parts := splitNonEmpty(strings.TrimSuffix(text, "++"))
		return append(parts, "+")
	}
	return splitNonEmpty(text)
}

func splitNonEmpty(text string) []string {
	raw := strings.Split(text, "+")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if token := strings.TrimSpace(part); token != "" {
			parts = append(parts, token)
		}
	}
	return parts
}

// Label 返回给人看的键位。macOS 用字形（⌘⇧Space），其他平台用加号写法。
func Label(accelerator string) string {
	return LabelFor(accelerator, runtime.GOOS)
}

// LabelFor 是 Label 的指定平台版本，供测试固定平台口径。
func LabelFor(accelerator, goos string) string {
	text := strings.TrimSpace(accelerator)
	if text == "" {
		text = DefaultAccelerator
	}
	parts := splitParts(glyphs.Replace(text))
	darwin := goos == "darwin"
	var modifiers []string
	var keys []string
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		if modifier, ok := modifierAliases[strings.ToLower(token)]; ok {
			if darwin {
				modifiers = append(modifiers, modifierGlyph(modifier))
			} else {
				modifiers = append(modifiers, modifierName(modifier, goos))
			}
			continue
		}
		if name, ok := keyName(token); ok {
			if darwin {
				keys = append(keys, keyGlyph(name))
			} else {
				keys = append(keys, name)
			}
			continue
		}
		keys = append(keys, token)
	}
	if darwin {
		return strings.Join(modifiers, "") + strings.Join(keys, "")
	}
	return strings.Join(append(modifiers, keys...), "+")
}

// modifierName 是非 macOS 平台上的修饰键文字：CmdOrCtrl 在那边就是 Ctrl。
func modifierName(modifier, goos string) string {
	if modifier == "CmdOrCtrl" {
		if goos == "darwin" {
			return "⌘"
		}
		return "Ctrl"
	}
	return modifier
}

// modifierGlyph 是 macOS 的修饰键字形。CmdOrCtrl 与 Cmd 都画成 ⌘。
func modifierGlyph(modifier string) string {
	switch modifier {
	case "CmdOrCtrl", "Cmd":
		return "⌘"
	case "Ctrl":
		return "⌃"
	case "Alt":
		return "⌥"
	case "Shift":
		return "⇧"
	}
	return modifier
}

// keyGlyph 是 macOS 上主键的字形；没有字形的键保留名字。
func keyGlyph(name string) string {
	switch name {
	case "Enter":
		return "↵"
	case "Tab":
		return "⇥"
	case "Backspace":
		return "⌫"
	case "Delete":
		return "⌦"
	case "Escape":
		return "⎋"
	case "Up":
		return "↑"
	case "Down":
		return "↓"
	case "Left":
		return "←"
	case "Right":
		return "→"
	}
	return name
}
