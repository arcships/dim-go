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

// statusDot 是连接状态：8px 实心圆，连上是 success，断线是 danger；屏幕上一个
// 字都不写（UI-23 第 1 档），读屏名给「连接状态：已连接 / 未连接」。
func statusDot(c *gui.Context, color gui.Color, ok bool) {
	dot := gui.Box(c).Width(dotStatus).Height(dotStatus).Shrink(0)
	if ok {
		dot = dot.Label("连接状态：已连接")
	} else {
		dot = dot.Label("连接状态：未连接")
	}
	dot.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, color, dotStatus/2)
	})
}

// avatar 是账号身份：24px 圆，里面是账号名的首字母（内容，不是界面语）。
// 读屏名给「账号：<首字母>」；没有首字母时退化为一个中性点。
func avatar(c *gui.Context, t *gui.Theme, initial string) {
	box := gui.Box(c).Width(rowCompact).Height(rowCompact).Shrink(0).
		Background(t.SurfacePressed).Radius(rowCompact/2).
		Label("账号：" + initial)
	box.Children(func() {
		if initial == "" {
			return
		}
		typeSecondaryStrong.style(t, gui.Text(c, initial)).TextColor(t.Text).
			TextAlign(gui.Center)
	})
}

// extStatusDot 是扩展列表行里的状态点：同样是 8px 圆，但带一个给检查器用的
// 标记（「图标：dot:<名称>」），因为列表里有多行、同一页会出现多个点。
func extStatusDot(c *gui.Context, color gui.Color, name string) {
	dot := gui.Box(c).Width(dotStatus).Height(dotStatus).Shrink(0).
		Label("图标：dot:" + name).AccessHidden(true)
	dot.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, color, dotStatus/2)
	})
}
