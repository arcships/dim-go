package ui

import gui "github.com/egoist/mygo/ui"

// 图标：Heroicons v2.2.0 outline（24 网格、1.5 描边、MIT）的内联路径；上游没有
// 的少数形状（设置导航里的键位、爪印、Webhook、插头、机器人）按同一网格自绘。
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
	iconChevronR   = mustSVG(`<path d="M8.25 4.5 15.75 12l-7.5 7.5"/>`)
	iconChevronL   = mustSVG(`<path d="M15.75 19.5 8.25 12l7.5-7.5"/>`)
	iconChevronUp  = mustSVG(`<path d="m4.5 15.75 7.5-7.5 7.5 7.5"/>`)
	iconPanelRight = mustSVG(`<path d="M3.75 4.5h16.5v15H3.75z"/><path d="M14.25 4.5v15"/>`)
	iconPaperclip  = mustSVG(`<path d="m18.375 12.739-7.693 7.693a4.5 4.5 0 0 1-6.364-6.364l10.94-10.94A3 3 0 1 1 19.5 7.372L8.552 18.32m.009-.01-.01.01m5.699-9.941-7.81 7.81a1.5 1.5 0 0 0 2.112 2.13"/>`)
	iconArrowUp    = mustSVG(`<path d="M12 19.5v-15m0 0-6.75 6.75M12 4.5l6.75 6.75"/>`)
	iconPlay       = mustSVG(`<path d="M8.25 5.25 18.75 12 8.25 18.75Z"/>`)
	iconStop       = mustSVG(`<rect x="6.75" y="6.75" width="10.5" height="10.5" rx="1.5"/>`)
	// 审批决定：勾（允许一次）、勾 + 锁（一直允许）、叉（拒绝）。
	iconCheckLock = mustSVG(`<path d="m4.5 12.75 5.25 5.25 9-12"/><rect x="14.25" y="13.5" width="6" height="6" rx="1"/><path d="M15.75 13.5v-1.5a1.5 1.5 0 0 1 3 0v1.5"/>`)

	// 行末的"更多动作"：三个点。
	iconEllipsis = mustSVG(`<circle cx="6.75" cy="12" r="1.4"/><circle cx="12" cy="12" r="1.4"/><circle cx="17.25" cy="12" r="1.4"/>`)

	// 终端：复制输出（两张叠纸）。
	iconClipboard = mustSVG(`<rect x="8.25" y="8.25" width="10.5" height="10.5" rx="2"/><path d="M15.75 6.75v-.5a2 2 0 0 0-2-2h-7.5a2 2 0 0 0-2 2v7.5a2 2 0 0 0 2 2h.5"/>`)

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

	// 设置左导航：一节一个形状。通用与 Git 复用上面的齿轮与分叉；
	// 其余取 Heroicons v2.2.0 outline 的同名形状。
	iconUserCircle = mustSVG(`<path d="M17.982 18.725A7.488 7.488 0 0 0 12 15.75a7.488 7.488 0 0 0-5.982 2.975m11.963 0a9 9 0 1 0-11.963 0m11.963 0A8.966 8.966 0 0 1 12 21a8.966 8.966 0 0 1-5.982-2.275M15 9.75a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"/>`)
	iconChartBar   = mustSVG(`<path d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75ZM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625ZM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z"/>`)
	iconSparkles   = mustSVG(`<path d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09ZM18.259 8.715 18 9.75l-.259-1.035a3.375 3.375 0 0 0-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 0 0 2.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 0 0 2.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 0 0-2.456 2.456Z"/>`)
	iconSwatch     = mustSVG(`<path d="M4.098 19.902a3.75 3.75 0 0 0 5.304 0l6.401-6.402M6.75 21A3.75 3.75 0 0 1 3 17.25V4.125C3 3.504 3.504 3 4.125 3h5.25c.621 0 1.125.504 1.125 1.125v4.072M6.75 21a3.75 3.75 0 0 0 3.75-3.75V8.197M6.75 21h13.125c.621 0 1.125-.504 1.125-1.125v-5.25c0-.621-.504-1.125-1.125-1.125h-4.072M10.5 8.197l2.88-2.88c.438-.439 1.15-.439 1.59 0l3.712 3.713c.44.44.44 1.152 0 1.59l-2.879 2.88M6.75 17.25h.008v.008H6.75v-.008Z"/>`)
	iconKeyboard   = mustSVG(`<rect x="2.25" y="5.25" width="19.5" height="13.5" rx="2.25"/><path d="M5.25 9h.008v.008H5.25V9Zm4.5 0h.008v.008H9.75V9Zm4.5 0h.008v.008h-.008V9Zm4.5 0h.008v.008h-.008V9ZM5.25 12.75h.008v.008H5.25v-.008Zm4.5 0h.008v.008H9.75v-.008Zm4.5 0h.008v.008h-.008v-.008Zm4.5 0h.008v.008h-.008v-.008ZM8.25 15.75h7.5"/>`)
	iconChip       = mustSVG(`<path d="M8.25 3v1.5M4.5 8.25H3m18 0h-1.5M4.5 12H3m18 0h-1.5m-15 3.75H3m18 0h-1.5M8.25 19.5V21M12 3v1.5m0 15V21m3.75-18v1.5m0 15V21m-9-1.5h10.5a2.25 2.25 0 0 0 2.25-2.25V6.75a2.25 2.25 0 0 0-2.25-2.25H6.75A2.25 2.25 0 0 0 4.5 6.75v10.5a2.25 2.25 0 0 0 2.25 2.25Zm.75-12h9v9h-9v-9Z"/>`)
	iconSliders    = mustSVG(`<path d="M10.5 6h9.75M10.5 6a1.5 1.5 0 1 1-3 0m3 0a1.5 1.5 0 1 0-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-9.75 0h9.75"/>`)
	iconTerminal   = mustSVG(`<path d="m6.75 7.5 3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0 0 21 18V6a2.25 2.25 0 0 0-2.25-2.25H5.25A2.25 2.25 0 0 0 3 6v12a2.25 2.25 0 0 0 2.25 2.25Z"/>`)
	iconWrench     = mustSVG(`<path d="M11.42 15.17 17.25 21A2.652 2.652 0 0 0 21 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 1 1-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 0 0 4.486-6.336l-3.276 3.277a3.004 3.004 0 0 1-2.25-2.25l3.276-3.276a4.5 4.5 0 0 0-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085m-1.745 1.437L5.909 7.5H4.5L2.25 3.75l1.5-1.5L7.5 4.5v1.409l4.26 4.26m-1.745 1.437 1.745-1.437m6.615 8.206L15.75 15.75M4.867 19.125h.008v.008h-.008v-.008Z"/>`)
	iconBell       = mustSVG(`<path d="M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0"/>`)
	iconShield     = mustSVG(`<path d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z"/>`)
	iconArchive    = mustSVG(`<path d="m20.25 7.5-.625 10.632a2.25 2.25 0 0 1-2.247 2.118H6.622a2.25 2.25 0 0 1-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125Z"/>`)
	iconInfo       = mustSVG(`<path d="m11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z"/>`)

	// 上游没有的五个形状：按同一 24 网格、1.5 描边自绘（键位、爪印、Webhook、
	// 插头、机器人），与上游形状的视觉重量一致。
	iconPaw     = mustSVG(`<circle cx="8.25" cy="7.5" r="1.5"/><circle cx="15.75" cy="7.5" r="1.5"/><circle cx="5.25" cy="12" r="1.5"/><circle cx="18.75" cy="12" r="1.5"/><path d="M12 11.25c2.9 0 5.25 2.4 5.25 4.5 0 1.8-1.35 3-3 3-.9 0-1.5-.375-2.25-.375s-1.35.375-2.25.375c-1.65 0-3-1.2-3-3 0-2.1 2.35-4.5 5.25-4.5Z"/>`)
	iconWebhook = mustSVG(`<circle cx="6" cy="7.5" r="2.25"/><circle cx="18" cy="7.5" r="2.25"/><circle cx="12" cy="17.25" r="2.25"/><path d="M8.25 7.5h7.5M7.5 9.5l3.25 6M16.5 9.5l-3.25 6"/>`)
	iconPlug    = mustSVG(`<path d="M9.75 3.75v3M14.25 3.75v3M6.75 6.75h10.5v3.75a5.25 5.25 0 0 1-10.5 0V6.75ZM12 15.75v4.5"/>`)
	iconBot     = mustSVG(`<rect x="4.5" y="7.5" width="15" height="12" rx="2.25"/><path d="M12 4.5v3M9 13.5h.008v.008H9V13.5Zm6 0h.008v.008H15V13.5ZM9.75 16.5h4.5"/><circle cx="12" cy="3.75" r=".75" fill="currentColor" stroke="none"/>`)
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

// 重建主窗口（标题栏 / 输入区）所需的形状：参考产品的 lucide 同名图标按同一
// 24 网格、1.5 描边绘制。与上面的分组分开，避免改动既有的图标清单。
var (
	iconChevronDown = mustSVG(`<path d="m19.5 8.25-7.5 7.5-7.5-7.5"/>`)
	iconPin         = mustSVG(`<path d="M15 3.75 20.25 9l-3 3 .75 3.75-2.25 2.25-4.5-4.5L6 18.75 3.75 21l1.5-3.75 5.25-5.25-4.5-4.5L8.25 5.25 12 6l3-3Z"/>`)
	iconPinOff      = mustSVG(`<path d="M15 3.75 20.25 9l-3 3 .75 3.75-2.25 2.25-2.25-2.25M8.25 5.25 12 6l-4.5 4.5M6 18.75 3.75 21l1.5-3.75m.75-6L9 9m-3 8.25 3-3M3 3l18 18"/>`)
	iconExternal    = mustSVG(`<path d="M13.5 4.5h6v6M19.5 4.5 11.25 12.75"/><path d="M18.75 13.5v4.125a1.875 1.875 0 0 1-1.875 1.875H5.625A1.875 1.875 0 0 1 3.75 17.625V6.375A1.875 1.875 0 0 1 5.625 4.5H9.75"/>`)
	iconDownload    = mustSVG(`<path d="M12 3.75v11.25m0 0 4.5-4.5m-4.5 4.5-4.5-4.5M4.5 19.5h15"/>`)
	iconLink        = mustSVG(`<path d="M9.75 14.25 14.25 9.75"/><path d="M7.5 16.5 5.25 18.75a3.182 3.182 0 0 1-4.5-4.5L3 12M16.5 7.5l2.25-2.25a3.182 3.182 0 0 1 4.5 4.5L21 12"/>`)
	iconFileCode    = mustSVG(`<path d="M14.25 3.75H6.75a1.5 1.5 0 0 0-1.5 1.5v13.5a1.5 1.5 0 0 0 1.5 1.5h10.5a1.5 1.5 0 0 0 1.5-1.5V7.5z"/><path d="M14.25 3.75V7.5h4.5M9.75 11.25 7.5 13.5l2.25 2.25m4.5-4.5 2.25 2.25-2.25 2.25"/>`)
	iconFolderOpen  = mustSVG(`<path d="M3.75 9V6.75A1.5 1.5 0 0 1 5.25 5.25h3l2.25 2.25h6A1.5 1.5 0 0 1 18 9v1.5"/><path d="M3.75 9h15.375a1.5 1.5 0 0 1 1.44 1.92l-1.5 6A1.5 1.5 0 0 1 17.625 18H4.875a1.5 1.5 0 0 1-1.44-1.92l1.5-6A1.5 1.5 0 0 1 6.375 9"/>`)
	iconPanelClose  = mustSVG(`<path d="M3.75 4.5h16.5v15H3.75z"/><path d="M14.25 4.5v15"/><path d="M10.5 9.75 8.25 12l2.25 2.25"/>`)
	iconPanelOpen   = mustSVG(`<path d="M3.75 4.5h16.5v15H3.75z"/><path d="M9.75 4.5v15"/><path d="M13.5 9.75 15.75 12l-2.25 2.25"/>`)
	iconHome        = mustSVG(`<path d="m3.75 10.5 8.25-6.75 8.25 6.75v8.25a1.5 1.5 0 0 1-1.5 1.5h-3.75V15h-6v5.25H5.25a1.5 1.5 0 0 1-1.5-1.5z"/>`)
	iconCube        = mustSVG(`<path d="m12 2.25 8.25 4.5v10.5L12 21.75l-8.25-4.5V6.75z"/><path d="M3.75 6.75 12 11.25l8.25-4.5M12 11.25v10.5"/>`)
	iconChevronsR   = mustSVG(`<path d="m6 6.75 5.25 5.25L6 17.25M12.75 6.75 18 12l-5.25 5.25"/>`)
	iconEffort      = mustSVG(`<path d="M4.5 7.5h15M4.5 12h15M4.5 16.5h9"/>`)
	iconProvider    = mustSVG(`<path d="M19.5 12a7.5 7.5 0 1 1-2.25-5.303"/><path d="M19.5 4.5V9h-4.5"/>`)
	// iconBrand 是品牌标记（空态与设置 › 账户的套餐行共用）：参考产品的 Dim 标志
	// ——一圈缺口环（缺口在左下）加一根穿过缺口的竖杆，观感介于「d」与「@」之间。
	// 它不是 24 网格的线性图标（环比图标粗、标记本身是宽的），故单独给 viewBox
	// 与两条各自的描边宽度；比例按 empty.png 与 账户-top.png 的标记实测（墨迹
	// 宽高比 34:24.5，环宽约为标记宽的 0.18，竖杆约为 0.13）。
	iconBrand = mustBrandSVG(
		`<g transform="rotate(5 17 13.4)">` +
			`<path stroke-width="6.2" d="M11.72 21.89A13.6 10.6 0 1 1 23.64 20.45"/>` +
			`</g>` +
			`<path stroke-width="4.4" d="M19.4 13.2V22"/>`)
)

// mustBrandSVG 解析品牌标记：viewBox 取标记自身的比例（34×24.5），描边宽度写在
// 路径上——不套图标那套 1.5 描边。
func mustBrandSVG(inner string) *gui.SVG {
	const head = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 34 24.5" fill="none" ` +
		`stroke="currentColor" stroke-linecap="round" stroke-linejoin="round">`
	return gui.MustParseSVG([]byte(head + inner + `</svg>`))
}

// 打开方式的应用图标：参考产品用各应用的品牌图标；这里按同一 24 网格画可辨认的
// 形状（VS Code 的折角、Zed 的字母 Z、终端的提示符、Ghostty 的幽灵轮廓）。
var (
	iconVSCode = mustSVG(`<path d="M17.25 3.75 6.75 12l10.5 8.25V3.75Z"/><path d="M6.75 12 3.75 9.75v4.5L6.75 12Z"/>`)
	iconZed    = mustSVG(`<path d="M5.25 4.5h13.5l-13.5 15h13.5"/>`)
	iconGhost  = mustSVG(`<path d="M6 18.75V8.25a6 6 0 0 1 12 0v10.5l-2.25-1.5-2.25 1.5-2.25-1.5-2.25 1.5-3-1.5Z"/><path d="M9.75 10.5h.008v.008H9.75V10.5Zm4.5 0h.008v.008h-.008V10.5Z"/>`)
)
