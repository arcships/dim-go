package ui

import gui "github.com/egoist/mygo/ui"

// 本文件是首片的两个控件外观：按钮与附件块。两者都用框架的 base 控件
// （ButtonBase、Box）加 U2 的值画，标签取 type-secondary（U2 §1.1），
// 状态按 U2 §4.2 的矩阵：填充、描边、悬停、按下、禁用。

// 按钮内边距：竖向 4 让按钮正好 row-regular 28（行盒 + 2×1 描边 + 2×4，
// 居中余量按 U2 §2.5 不进网格）；横向 12 取 U2 的 space-3。
const (
	buttonPadV = space1
	buttonPadH = space3
)

// actionButton 是一个按钮：主按钮 accent 填充、次按钮 surface + border；
// 悬停与按下换面，禁用转为中性面并交给框架的 0.5 不透明度（U2 §4.1、§4.2）。
//
// text 是显示的文字，label 是读屏名（空则同 text），primary 取主按钮外观，
// disabled 与框架的 Disabled 同义：不响应指针与键盘。
func actionButton(c *gui.Context, text, label string, primary, disabled bool) gui.Element {
	t := c.Theme()
	if label == "" {
		label = text
	}
	b := gui.ButtonBase(c).Label(label).Radius(radius).
		Padding(buttonPadV, buttonPadH).Gap(space1)
	if disabled {
		b.Disabled(true)
	}
	// 次级：surface + 1px border + text；主级：accent + accent-on。
	bg, fg, border := t.Surface, t.Text, t.Border
	if primary {
		bg, fg, border = t.Accent, t.AccentText, gui.Transparent
	}
	if !disabled {
		switch {
		case b.Pressed():
			if primary {
				bg = t.AccentPressed
			} else {
				bg = t.SurfacePressed
			}
		case b.Hovered():
			if primary {
				bg = t.AccentHover
			} else {
				bg = t.SurfaceHover
			}
		}
	} else {
		// 禁用的主按钮不给强调填充：识别由文字标签承担（U2 §4.1）。
		bg, fg, border = t.Surface, t.TextMuted, t.Border
	}
	b.Background(bg)
	// 主按钮的描边是透明色而不是"没有描边"：让主/次与启用/禁用的盒模型一致，
	// 切换发送与停止、启用与禁用时不发生 1px 位移（U2 §4.1）。
	b.Border(strokeHairline, border)
	b.Children(func() {
		typeSecondary.style(t, gui.Text(c, text)).TextColor(fg).SingleLine()
	})
	return b
}

// chip 是附件块：surface 面、1px 描边、控件圆角，文字取次要级（U2 §1.1）。
func chip(c *gui.Context, text string) {
	t := c.Theme()
	gui.Row(c).AlignItems(gui.Center).Shrink(0).
		Padding(space1, space2).Radius(radius).
		Background(t.Surface).Border(strokeHairline, t.Border).Children(func() {
		typeSecondary.style(t, gui.Text(c, text)).SingleLine()
	})
}
