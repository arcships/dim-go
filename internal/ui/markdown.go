package ui

import (
	gui "github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// markdownTheme 是助手正文的块级样式（C4 §5 的路线：用 mygo 内建 ui/markdown）。
//
// 每一项都对着官方示例 markdown-chat.png 量过：示例里 h2 约 18/700、行内代码是
// 带底色的 chip、代码块有自己的面与圆角、引用左侧一条竖线、表格有表头底与边线、
// 链接用强调色。框架的 DefaultTheme 已经给了这些结构；我们把颜色换成自己的 token、
// 把字号按 13 的正文基准写死，避免"零值 options"让样式随框架版本漂。
func markdownTheme(t *gui.Theme) *markdown.Theme {
	md := markdown.DefaultTheme(t)
	// 颜色取我们的 token（U2 §3）：链接用强调色的文字版，代码块与表头用面层，
	// 引用竖线用主题边界色（UI-18）。
	md.Link = colorAccentInk.in(t)
	md.Code = t.Text
	md.CodeBackground = colorSurface.in(t)
	md.CodeBlockBackground = colorSurface.in(t)
	md.CodeBlockBorder = t.Border
	md.CodeBlockRadius = radiusCode
	md.TableBorder = t.Border
	md.TableHeader = colorSurface.in(t)
	md.QuoteBar = t.Border
	md.QuoteText = t.TextMuted
	md.QuoteBarWidth = 3
	md.Rule = t.Border
	// 呼吸（U0 §3.1 的 M2/M6/M7 量出来的）：
	//   段落之间：示例 18px 墨迹间隙 / 21px 行盒 ≈ 0.85em，我们原来 0.75em；
	//   列表项之间：示例 22.5px 行距 = 行盒 + 0.25em，我们已是 0.25em（不动）；
	//   表格行：示例 32px 行距 = 行盒 21 + 上下 0.4em，我们的行盒 19.5 只有 31px，
	//   上下各加 0.05em 补回到 2.2em 的行距。
	md.BlockSpacing = 0.85
	md.ItemSpacing = 0.25
	md.TableCellPadding = [2]float32{0.6, 0.45}

	// 字阶：以正文 1em 为基准，六级的字号与字重与示例一致（示例 h2 ≈ 1.35em/700）。
	sizes := [6]float32{1.6, 1.35, 1.15, 1, 0.95, 0.9}
	weights := [6]int{700, 650, 600, 600, 600, 600}
	for i := range md.Headings {
		md.Headings[i].Size = sizes[i]
		md.Headings[i].Weight = weights[i]
		md.Headings[i].Color = t.Text
		// 标题上下留白：示例 h2 上方墨迹间隙 30px、下方 27px（21px 行盒），
		// 换算成 em 的上/下留白约 1.45 / 0.6。
		md.Headings[i].Top = 1.45
		md.Headings[i].Bottom = 0.6
	}
	md.Headings[5].Color = t.TextMuted
	return md
}
