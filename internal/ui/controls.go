package ui

import gui "github.com/egoist/mygo/ui"

// 本文件是首片的两个控件外观：按钮与附件块。
//
// 按钮直接用框架的 Button / PrimaryButton，不覆盖尺寸、内边距、填充、描边
// 或字体（章程 UI-13）；默认就是 6/14 内边距、surface + border、主按钮
// accent 填充、禁用整棵子树 0.5 不透明度。我们只补读屏名与禁用状态。
//
// 附件块是自绘 Base（Row + 值），不受 UI-13 限制。

// actionButton 是一个按钮：主按钮取框架的 PrimaryButton，次按钮取 Button。
// text 是显示的文字，label 是读屏名（空则同 text），disabled 与框架的
// Disabled 同义：不响应指针与键盘。
func actionButton(c *gui.Context, text, label string, primary, disabled bool) gui.Element {
	var b gui.Element
	if primary {
		b = gui.PrimaryButton(c, text)
	} else {
		b = gui.Button(c, text)
	}
	if label != "" && label != text {
		b = b.Label(label)
	}
	if disabled {
		b = b.Disabled(true)
	}
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
