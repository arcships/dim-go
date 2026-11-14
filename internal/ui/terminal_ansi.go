package ui

import (
	"strings"
)

// 终端输出的解释器（F30 的变化规则）：只做三类最常用的控制序列，不做全屏光标定位。
//
//	\r        回到行首（后写的内容覆盖同一行——进度条就是这么更新的）
//	\b        退格（删掉前一个字符）
//	\x1b[K     清行（从光标到行尾清空）
//
// 其余 CSI 序列（颜色除外）按不可见处理：颜色在 termLine.color 里，不在这里解析。
func interpretLine(raw string) string {
	var out []rune
	i := 0
	runes := []rune(raw)
	for i < len(runes) {
		r := runes[i]
		switch r {
		case '\r':
			out = out[:0]
			i++
			continue
		case '\b':
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
			i++
			continue
		case 0x1b:
			// \x1b[K：清到行尾；其它 CSI 整段跳过。
			if i+1 < len(runes) && runes[i+1] == '[' {
				j := i + 2
				for j < len(runes) && (runes[j] < '@' || runes[j] > '~') {
					j++
				}
				if j < len(runes) {
					if runes[j] == 'K' {
						out = nil
					}
					i = j + 1
					continue
				}
			}
			i++
			continue
		}
		out = append(out, r)
		i++
	}
	return strings.TrimRight(string(out), "\x00")
}

// termVisible 是输出区当前应该显示的行（把控制序列解释成最终文本）。
func (a *App) termVisible() []termLine {
	return visibleLines(a.termLines)
}

// visibleLines 解释一组输出行里的控制序列。
func visibleLines(lines []termLine) []termLine {
	out := make([]termLine, 0, len(lines))
	for _, ln := range lines {
		out = append(out, termLine{text: interpretLine(ln.text), color: ln.color})
	}
	return out
}
