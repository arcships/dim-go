package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 窄窗档（compact）：窗口窄于 CompactWidth（见 tokens.go）时的七处形态差异
// 都从这里取判据，别处不再各写一个阈值。验收档是 720。

// narrowWindow 报告当前窗口是否落在窄窗档。
func narrowWindow(c *gui.Context) bool {
	width, _ := c.Size()
	return width < CompactWidth
}

// dialogWidth 是命令面板与对话框的内容宽：窄窗里占满「窗口 − 2×留白」，
// 其余档取 preferred；两者都不越过窗口（R-GC-039）。
func dialogWidth(c *gui.Context, preferred float32) float32 {
	width, _ := c.Size()
	limit := width - 2*narrowDialogMargin
	if narrowWindow(c) {
		return max(limit, space12*2)
	}
	if limit < preferred {
		if limit < space12*4 {
			return space12 * 2
		}
		return limit
	}
	return preferred
}

// settingsUsableWidth 是设置内容列的可用宽：常驻导航占 224，窄窗里导航收成顶部
// 下拉、内容列占满（与 settings 的排版同一判据）。按内容宽度退化的设置行都用它，
// 别各自再减一次 SettingsNavWidth。
func settingsUsableWidth(c *gui.Context) float32 {
	width, _ := c.Size()
	if narrowWindow(c) {
		return width
	}
	return width - SettingsNavWidth
}
