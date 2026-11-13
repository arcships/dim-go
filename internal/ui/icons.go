package ui

import gui "github.com/egoist/mygo/ui"

// 图标：Heroicons v2.2.0 outline（24 网格、1.5 描边、MIT）的内联路径。
// 16px 渲染时描边自动落到 1.0px（U2 §6.4）；不引素材文件、不用字符冒充图标
// （UI-21、X15）。
//
// 图标颜色取所在文字的 TextColor（框架 Icon 的 currentColor），因此这里只放
// 形状，不写颜色。
//
// 三类：状态（状态位用）、动作（可点的入口用）、对象（操作行里表示读/改/跑）。
var (
	// 状态。
	iconCheck    = mustSVG(`<path d="m4.5 12.75 6 6 9-13.5"/>`)
	iconXMark    = mustSVG(`<path d="M6 18 18 6M6 6l12 12"/>`)
	iconClock    = mustSVG(`<path d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"/>`)
	iconWarn     = mustSVG(`<path d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"/>`)
	iconDot      = mustSVG(`<circle cx="12" cy="12" r="4" fill="currentColor" stroke="none"/>`)
	iconDotEmpty = mustSVG(`<circle cx="12" cy="12" r="0" fill="none" stroke="none"/>`)

	// 动作：新建、搜索、收起/展开侧栏、右栏、附件、发送、停止。
	iconPlus       = mustSVG(`<path d="M12 4.5v15m7.5-7.5h-15"/>`)
	iconSearch     = mustSVG(`<path d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z"/>`)
	iconChevronL   = mustSVG(`<path d="M15.75 19.5 8.25 12l7.5-7.5"/>`)
	iconChevronUp  = mustSVG(`<path d="m4.5 15.75 7.5-7.5 7.5 7.5"/>`)
	iconPanelRight = mustSVG(`<path d="M3.75 4.5h16.5v15H3.75z"/><path d="M14.25 4.5v15"/>`)
	iconPaperclip  = mustSVG(`<path d="m18.375 12.739-7.693 7.693a4.5 4.5 0 0 1-6.364-6.364l10.94-10.94A3 3 0 1 1 19.5 7.372L8.552 18.32m.009-.01-.01.01m5.699-9.941-7.81 7.81a1.5 1.5 0 0 0 2.112 2.13"/>`)
	iconArrowUp   = mustSVG(`<path d="M12 19.5v-15m0 0-6.75 6.75M12 4.5l6.75 6.75"/>`)
	iconStop      = mustSVG(`<rect x="6.75" y="6.75" width="10.5" height="10.5" rx="1.5"/>`)
	// 审批决定：勾（允许一次）、勾 + 锁（一直允许）、叉（拒绝）。
	iconCheckLock = mustSVG(`<path d="m4.5 12.75 5.25 5.25 9-12"/><rect x="14.25" y="13.5" width="6" height="6" rx="1"/><path d="M15.75 13.5v-1.5a1.5 1.5 0 0 1 3 0v1.5"/>`)


	// 设置入口：齿轮。
	iconGear = mustSVG(`<path d="M10.343 3.94c.09-.542.56-.94 1.11-.94h1.093c.55 0 1.02.398 1.11.94l.149.894c.07.424.384.764.78.93.398.164.855.142 1.205-.108l.737-.527a1.125 1.125 0 0 1 1.45.12l.773.774c.39.389.44 1.002.12 1.45l-.527.737c-.25.35-.272.806-.107 1.204.165.397.505.71.93.78l.893.15c.543.09.94.56.94 1.109v1.094c0 .55-.397 1.02-.94 1.11l-.893.149c-.425.07-.765.383-.93.78-.165.398-.143.854.107 1.204l.527.738c.32.447.269 1.06-.12 1.45l-.774.773a1.125 1.125 0 0 1-1.449.12l-.738-.527c-.35-.25-.806-.272-1.203-.107-.397.165-.71.505-.781.929l-.149.894c-.09.542-.56.94-1.11.94h-1.094c-.55 0-1.019-.398-1.11-.94l-.148-.894c-.071-.424-.384-.764-.781-.93-.398-.164-.854-.142-1.204.108l-.738.527c-.447.32-1.06.269-1.45-.12l-.773-.774a1.125 1.125 0 0 1-.12-1.45l.527-.737c.25-.35.273-.806.108-1.204-.165-.397-.505-.71-.93-.78l-.894-.15c-.542-.09-.94-.56-.94-1.109v-1.094c0-.55.398-1.02.94-1.11l.894-.149c.424-.07.765-.383.93-.78.165-.398.143-.854-.107-1.204l-.527-.738a1.125 1.125 0 0 1 .12-1.45l.773-.773a1.125 1.125 0 0 1 1.45-.12l.737.527c.35.25.807.272 1.204.107.397-.165.71-.505.78-.929l.15-.894Z"/><path d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"/>`)

	// 分支、目标、后台任务：分叉、旗标、播放圈（运行中的任务）。
	iconBranch = mustSVG(`<circle cx="6.75" cy="6" r="2.25"/><circle cx="6.75" cy="18" r="2.25"/><circle cx="17.25" cy="9.75" r="2.25"/><path d="M6.75 8.25v7.5M17.25 12a5.25 5.25 0 0 1-5.25 5.25H9"/>`)
	iconTarget = mustSVG(`<circle cx="12" cy="12" r="8.25"/><circle cx="12" cy="12" r="3.75"/>`)
	iconTask   = mustSVG(`<circle cx="12" cy="12" r="8.25"/><path d="M10.5 9.75 15 12l-4.5 2.25z"/>`)

	// 上下文用量与撤回：环形表（用量）、回转箭头（撤回）、对话气泡（空态）。
	iconGauge = mustSVG(`<path d="M3.75 12a8.25 8.25 0 0 1 16.5 0"/><path d="M12 12l4.5-3"/>`)
	iconUndo  = mustSVG(`<path d="M9 15 4.5 10.5 9 6"/><path d="M4.5 10.5h9a6 6 0 0 1 0 12h-3"/>`)
	iconChat  = mustSVG(`<path d="M20.25 8.511a8.25 8.25 0 0 1-11.6 7.5L3.75 17.25l1.24-4.9A8.25 8.25 0 1 1 20.25 8.5Z"/>`)

	// 对象类型：文件、命令、读取、改动、其它。
	iconDocument = mustSVG(`<path d="M14.25 3.75H6.75a1.5 1.5 0 0 0-1.5 1.5v13.5a1.5 1.5 0 0 0 1.5 1.5h10.5a1.5 1.5 0 0 0 1.5-1.5V7.5z"/><path d="M14.25 3.75V7.5h4.5"/>`)
	iconCommand  = mustSVG(`<path d="m5.25 8.25 3 3.75-3 3.75m5.25 0h6"/>`)
	iconEye      = mustSVG(`<path d="M2.25 12S5.25 5.25 12 5.25 21.75 12 21.75 12 18.75 18.75 12 18.75 2.25 12 2.25 12Z"/><circle cx="12" cy="12" r="2.75"/>`)
	iconPencil   = mustSVG(`<path d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L10.582 16.07a4.5 4.5 0 0 1-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 0 1 1.13-1.897z"/>`)
	iconMore     = mustSVG(`<path d="M6.75 12h.008v.008H6.75V12Zm5.25 0h.008v.008H12V12Zm5.25 0h.008v.008h-.008V12Z"/>`)
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

// actionIcon 是一行操作的图标：按动作取对象类型的形状（UI-21）。
func actionIcon(action string) *gui.SVG {
	switch action {
	case "读取":
		return iconEye
	case "改动":
		return iconPencil
	case "执行":
		return iconCommand
	}
	return iconMore
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

// iconGlyph 画一个"图标 + 文字"里用的图标：16px 盒，带一个给检查器用的标记
// （Label 「图标：<名字>」，对辅助技术隐藏）。检查项 L17 靠它判"文字旁边确实
// 画了图标"，而不是靠像素——文字本身的墨迹会掩盖"图标没了"。
func iconGlyph(c *gui.Context, icon *gui.SVG, color gui.Color, mark string) {
	box := gui.Box(c).Width(iconInline).Height(iconInline).Shrink(0).
		Label("图标：" + mark).AccessHidden(true)
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Icon(icon, r, color)
	})
}

// iconButton 是一个只有图标的入口：用框架的无外观 ButtonBase 承担命中、焦点、
// 悬停与按下判定，图标与五态由我们按主题画（UI-13：不改框架 Button 的样式）。
//
// 五态（UI-14、X17）：默认透明底；悬停 `surface-hover`；按下 `surface-pressed`；
// 聚焦由框架画 2px 焦点环；禁用整棵子树 0.5 且不接收指针与键盘。
// 命中区不小于 `hit-min`（24×24），图标本身固定 16×16。
type iconButtonSpec struct {
	// icon 是图标形状，颜色取下面 color（缺省用次要文字色）。
	icon *gui.SVG
	// label 是读屏名；tip 非空时作为悬停提示（名称 + 快捷键），否则用 label。
	label string
	tip   string
	// size 是命中区边长，取 row-compact 24 或 row-regular 28。
	size float32
	// disabled 为真时不接收指针与键盘。
	disabled bool
	// color 是图标颜色；零值表示未设，取 `text-secondary`。
	color gui.Color
}

// iconButton 画一个图标入口，返回它是否被按下（Clicked）。
func iconButton(c *gui.Context, spec iconButtonSpec) bool {
	return iconAction(c, spec, false)
}

// iconAction 是图标入口的本体：primary 为真时用强调色填充（发送/停止这种主
// 动作），否则透明底。五态：默认 / 悬停 / 按下 / 聚焦（框架环）/ 禁用。
func iconAction(c *gui.Context, spec iconButtonSpec, primary bool) bool {
	t := c.Theme()
	size := spec.size
	if size < hitMin {
		size = hitMin
	}
	color := spec.color
	if color == (gui.Color{}) {
		color = t.TextMuted
	}
	tip := spec.label
	if spec.tip != "" {
		tip = spec.tip
	}

	b := gui.ButtonBase(c).Width(size).Height(size).Radius(radius).
		Label(spec.label).Tooltip(tip).Disabled(spec.disabled)
	if primary {
		// 主图标动作：强调填充，图标用 accent-on；禁用时框架式整棵子树 0.5。
		color = t.AccentText
		b = b.Background(t.Accent)
		switch {
		case spec.disabled:
			b = b.Opacity(0.5)
		case b.Pressed():
			b = b.Background(t.AccentPressed)
		case b.Hovered():
			b = b.Background(t.AccentHover)
		}
	} else {
		switch {
		case spec.disabled:
			b = b.Opacity(0.5)
		case b.Pressed():
			b = b.Background(t.SurfacePressed)
		case b.Hovered():
			b = b.Background(t.SurfaceHover)
		}
	}
	b.Children(func() {
		iconBox(c, spec.icon, color, "")
	})
	// ButtonBase 用 Clicked 报告指针、Enter 与 Space（Changed 是值型控件的口径）。
	return b.Clicked()
}

// textIconButton 是"图标 + 文字"的入口：后果大、需要确认的动作用它（审批的三
// 个决定）。同样用无外观 ButtonBase 承担命中与焦点，外观由这里按主题画。
type textIconButtonSpec struct {
	icon *gui.SVG
	// mark 是给检查器用的图标标记（图标：<mark>）；同时写进元素表清单。
	mark     string
	text     string
	label    string
	tip      string
	primary  bool
	disabled bool
}

func textIconButton(c *gui.Context, spec textIconButtonSpec) bool {
	t := c.Theme()
	color := t.Text
	bg := gui.Color{}
	if spec.primary {
		color = t.AccentText
		bg = t.Accent
	}
	label := spec.label
	if label == "" {
		label = spec.text
	}
	b := gui.ButtonBase(c).Padding(controlPadV, controlPadH).Radius(radius).
		Label(label).Tooltip(spec.tip).Disabled(spec.disabled)
	if bg.A != 0 {
		b = b.Background(bg)
	} else {
		b = b.Background(t.Surface)
	}
	// 只有次按钮画轮廓（落在 surface 上需要一条界）；主按钮的强调填充本身就是界。
	if !spec.primary {
		b = b.Border(strokeHairline, t.Border)
	}
	switch {
	case spec.disabled:
		b = b.Opacity(0.5)
	case spec.primary && b.Pressed():
		b = b.Background(t.AccentPressed)
	case spec.primary && b.Hovered():
		b = b.Background(t.AccentHover)
	case !spec.primary && b.Pressed():
		b = b.Background(t.SurfacePressed)
	case !spec.primary && b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
			iconGlyph(c, spec.icon, color, spec.mark)
			typeControl.style(t, gui.Text(c, spec.text)).TextColor(color).SingleLine()
		})
	})
	return b.Clicked()
}

// iconMenuButton 是同类入口，用来切换面板开合：图标本身表达状态，
// 读屏名按状态换（打开右栏 / 收起右栏）。
func iconMenuButton(c *gui.Context, icon *gui.SVG, label string, open bool) bool {
	return iconButton(c, iconButtonSpec{icon: icon, label: label, size: rowCompact})
}
