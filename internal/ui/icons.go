package ui

import gui "github.com/egoist/mygo/ui"

// 状态图标：Heroicons v2.2.0 outline（24 网格、1.5 描边、MIT）的内联路径。
// 16px 渲染时描边自动落到 1.0px（U2 §6.4）；接入图标资源前不引素材文件。
//
// 图标颜色取所在文字的 TextColor（框架 Icon 的 currentColor），因此这里只放
// 形状，不写颜色。
var (
	iconCheck    = mustSVG(`<path d="m4.5 12.75 6 6 9-13.5"/>`)
	iconXMark    = mustSVG(`<path d="M6 18 18 6M6 6l12 12"/>`)
	iconClock    = mustSVG(`<path d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"/>`)
	iconWarn     = mustSVG(`<path d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"/>`)
	iconDot      = mustSVG(`<circle cx="12" cy="12" r="4" fill="currentColor" stroke="none"/>`)
	iconDotEmpty = mustSVG(`<circle cx="12" cy="12" r="0" fill="none" stroke="none"/>`)
)

// mustSVG 把一条 24 网格路径包成图标 SVG；路径是常量，解析失败属于代码错误。
func mustSVG(inner string) *gui.SVG {
	const head = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" ` +
		`stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">`
	return gui.MustParseSVG([]byte(head + inner + `</svg>`))
}

// statusIcon 把样例里的状态标记换成图标；空标记没有图标。
func statusIcon(marker string) *gui.SVG {
	switch marker {
	case "●":
		return iconDot
	case "◷":
		return iconClock
	case "✕":
		return iconXMark
	}
	return nil
}

// iconBox 画一个固定 16×16 的状态盒：尺寸不随字号变化（U2 §6.4、C5）。
// 图标用 Painter 自绘，不覆盖框架 Icon 组件的字号/尺寸。
func iconBox(c *gui.Context, icon *gui.SVG, color gui.Color, label string) {
	box := gui.Box(c).Width(iconInline).Height(iconInline).Shrink(0)
	if label != "" {
		box = box.Label(label)
	}
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Icon(icon, r, color)
	})
}
