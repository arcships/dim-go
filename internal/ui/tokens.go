package ui

import (
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 区域尺寸：取自 docs/domains/U1-ia.md §1、§7（U0 §12 的数值）。
// 这组是布局契约，不随视觉规范变化。
const (
	// WindowWidth 与 WindowHeight 是首片骨架的默认窗口尺寸。
	WindowWidth  = 1440
	WindowHeight = 900

	// R1 左栏。
	SidebarWidth     = 256
	SidebarRailWidth = 12
	SidebarMinWidth  = 200
	SidebarMaxWidth  = 360

	// R2 标题行。
	HeaderHeight = 48

	// R3 主区内容上限。
	ContentMaxWidth = 784

	// R4 输入位：卡片 60–300，上沿可带一行队列（33）。
	ComposerMinHeight = 60
	ComposerMaxHeight = 300
	ComposerHeight    = 124
	ApprovalHeight    = 150
	QueueHeight       = 33

	// R3 的宽度下限：右栏打开时主区不缩到 400 以下。
	MainMinWidth = 400

	// R5 右栏：280–（容器−400）。
	RightPanelMinWidth       = 280
	RightPanelPreferredWidth = 320
	RightPanelMainMinWidth   = 400
)

// 视觉 token：docs/domains/U2-visual.md §8 的 token 表（v2：默认优先），
// 值只在这里出现一次。
//
// 颜色不写在元素上：主题有字段的取 c.Theme()，框架没有字段的两个 token
// （border-strong、accent-ink）取本文件的 colorXxx.in(t)。
const (
	// 间距：沿用框架基础单位 4，只用七档（U2 §2.1、§2.2）。
	spaceUnit = 4
	space1    = 4
	space2    = 8
	space3    = 12
	space4    = 16
	space6    = 24
	space8    = 32
	space12   = 48

	// 密度：控件高度（U2 §8.3）。
	rowRegular  = 28
	rowCompact  = 24
	controlMini = 20

	// 图标与状态点（U2 §8.3）。
	iconInline = 16
	iconBlock  = 20
	dotStatus  = 8

	// 描边与焦点环（U2 §8.3）。
	strokeHairline  = 1
	strokeFocus     = 2
	strokeMarker    = 2
	focusRingOffset = 1

	// 圆角：控件与卡片同值，沿用框架 Theme.Radius（U2 §8.3）。
	radius     = 6
	radiusCode = radius + 2

	// 命中区下限（U2 §6.5、§8.3）。
	hitMin = 24

	// 沿用框架的度量（U2 §8.5）。
	scrollbarWidth = 6
)

// 动效：沿用框架默认时长与节奏（U2 §5.1、§5.3、§8.4）。
const (
	motionTransition = 200 * time.Millisecond
	motionSpring     = 500 * time.Millisecond
	motionContent    = 350 * time.Millisecond

	caretCycle   = 1060 * time.Millisecond
	caretWidth   = 2
	caretHeight  = 14
	tooltipDelay = 600 * time.Millisecond
	tooltipWarm  = 400 * time.Millisecond
)

// 首片局部尺寸：宽度与最小高度，不是间距，不进 §2.2 的标尺。
const (
	composerPadding       = space2 // 队列条与输入卡之间 8（U2 §2.3）
	sidebarHeaderHeight   = 32
	userBubbleMaxWidth    = 520
	composerTextMinHeight = 44
	opNumberWidth         = 16
	opActionWidth         = 36
	opResultWidth         = 56
	opDurationWidth       = 44
	railMarkWidth         = 6
	railMarkHeight        = 2
)

// 字阶：11 个语义级（U2 §1.1、§8.2）。
//
// line 是默认字号下的绝对行高，由框架的倍率推出；0 表示"不设行高"，交给框架
// 按字体度量排版。style 应用时把非零行高换算成字号的倍数，字号再乘以本帧的
// 文本缩放，行高随字号一起放大（U2 §7.4）。
type fontStep struct {
	size   float32
	line   float32
	weight int
	// family 为空是系统 UI 字体栈；代码级用系统等宽（U2 §1.3）。
	family string
}

var (
	typeBody            = fontStep{size: 14, line: 21, weight: 400}
	typeBodyStrong      = fontStep{size: 14, line: 21, weight: 600}
	typeSecondary       = fontStep{size: 12.25, weight: 400}
	typeSecondaryStrong = fontStep{size: 12.25, weight: 600}
	typeMicro           = fontStep{size: 10.5, weight: 400}
	typeCode            = fontStep{size: 12.25, line: 17.15, weight: 400, family: codeFont}
	typeTitleRow        = fontStep{size: 14, weight: 600}
	typeH1              = fontStep{size: 22.4, line: 33.6, weight: 700}
	typeH2              = fontStep{size: 18.9, line: 28.35, weight: 650}
	typeH3              = fontStep{size: 16.1, line: 24.15, weight: 600}
	typeH4              = fontStep{size: 14, line: 21, weight: 600}
	typeH5              = fontStep{size: 13.3, line: 19.95, weight: 600}
	typeH6              = fontStep{size: 12.6, line: 18.9, weight: 600}
)

// codeFont 是代码的字体栈：系统等宽，不内嵌字体（U2 §1.3）。
const codeFont = "SF Mono, Menlo, monospace"

// typeBodySize 是正文在文本缩放 1 时的字号，也是主题 FontSize 的基准。
const typeBodySize = 14

// textScale 是本帧文字相对正文的倍数：主题字号由框架按系统文本缩放放大，
// 这里用它与 14 的比把同一缩放带到每一级字号上。
func textScale(t *gui.Theme) float32 {
	if t == nil || t.FontSize <= 0 {
		return 1
	}
	return t.FontSize / typeBodySize
}

// sizeIn 返回该字级在 t 下的字号，供 span 与需要数字的地方使用。
func (f fontStep) sizeIn(t *gui.Theme) float32 { return f.size * textScale(t) }

// style 把字级写到 e：字号（含文本缩放）、行高与字重。行高按倍数给，
// 因此文本放大后行盒等比变高，不会截断（U2 §7.4）；line 为 0 时不设行高，
// 由框架按字体度量排版（U2 §1.1）。
func (f fontStep) style(t *gui.Theme, e gui.Element) gui.Element {
	if f.family != "" {
		e = e.Font(f.family)
	}
	e = e.FontSize(f.sizeIn(t)).FontWeight(f.weight)
	if f.line > 0 {
		e = e.LineHeight(f.line / f.size)
	}
	return e
}

// colorSet 是一个语义色 token 的明、暗两值（U2 §3.2、§3.3）。
type colorSet struct {
	light, dark gui.Color
}

// in 返回该 token 在主题外观下的值。
func (c colorSet) in(t *gui.Theme) gui.Color {
	if t != nil && t.Dark {
		return c.dark
	}
	return c.light
}

// 颜色 token：每主题 18 个设计 token（其中 border-strong、accent-ink 框架没有
// 字段），另 3 个沿用框架、不参与设计（U2 §8.1）。
var (
	colorBG             = colorSet{gui.Hex("#ffffff"), gui.Hex("#18181b")}
	colorSurface        = colorSet{gui.Hex("#f4f4f5"), gui.Hex("#27272a")}
	colorSurfaceHover   = colorSet{gui.Hex("#e9e9ec"), gui.Hex("#323236")}
	colorSurfacePressed = colorSet{gui.Hex("#dddde1"), gui.Hex("#3c3c41")}
	colorBorder         = colorSet{gui.Hex("#d9d9de"), gui.Hex("#3f3f46")}
	colorBorderStrong   = colorSet{gui.Hex("#85858b"), gui.Hex("#8b8b94")}
	colorText           = colorSet{gui.Hex("#18181b"), gui.Hex("#f4f4f5")}
	colorTextSecondary  = colorSet{gui.Hex("#5f5f68"), gui.Hex("#a1a1aa")}
	colorAccent         = colorSet{gui.Hex("#2563eb"), gui.Hex("#2563eb")}
	colorAccentHover    = colorSet{gui.Hex("#1d4ed8"), gui.Hex("#1d4ed8")}
	colorAccentPressed  = colorSet{gui.Hex("#1e40af"), gui.Hex("#1e40af")}
	colorAccentOn       = colorSet{gui.Hex("#ffffff"), gui.Hex("#ffffff")}
	colorAccentInk      = colorSet{gui.Hex("#1d4ed8"), gui.Hex("#60a5fa")}
	colorSuccess        = colorSet{gui.Hex("#15803d"), gui.Hex("#22c55e")}
	colorWarning        = colorSet{gui.Hex("#b45309"), gui.Hex("#f59e0b")}
	colorDanger         = colorSet{gui.Hex("#b91c1c"), gui.Hex("#f87171")}
	colorSelection      = colorSet{gui.RGBA(37, 99, 235, 0.20), gui.RGBA(59, 130, 246, 0.30)}
	colorFocus          = colorSet{gui.Hex("#2563eb"), gui.Hex("#60a5fa")}

	// 沿用框架默认、不参与设计的三个字段（U2 §8.1）。
	colorInverse     = colorSet{gui.Hex("#18181b"), gui.Hex("#f4f4f5")}
	colorInverseText = colorSet{gui.Hex("#ffffff"), gui.Hex("#18181b")}
	colorScrollbar   = colorSet{gui.RGBA(0, 0, 0, 0.32), gui.RGBA(255, 255, 255, 0.35)}
)

// theme 把 token 映射到框架主题（U2 §3.4、§8.5）：外观决定用哪一套值；
// 文本缩放与高对比按框架的同一套做法跟随系统设置。
func theme(dark bool, prefs gui.Preferences) *gui.Theme {
	t := gui.LightTheme()
	if dark {
		t = gui.DarkTheme()
	}
	t.Background = colorBG.in(t)
	t.Surface = colorSurface.in(t)
	t.SurfaceHover = colorSurfaceHover.in(t)
	t.SurfacePressed = colorSurfacePressed.in(t)
	t.Border = colorBorder.in(t)
	t.Text = colorText.in(t)
	t.TextMuted = colorTextSecondary.in(t)
	t.Accent = colorAccent.in(t)
	t.AccentHover = colorAccentHover.in(t)
	t.AccentPressed = colorAccentPressed.in(t)
	t.AccentText = colorAccentOn.in(t)
	t.Danger = colorDanger.in(t)
	t.Warning = colorWarning.in(t)
	t.Success = colorSuccess.in(t)
	t.Selection = colorSelection.in(t)
	t.Focus = colorFocus.in(t)
	t.Inverse = colorInverse.in(t)
	t.InverseText = colorInverseText.in(t)
	t.Scrollbar = colorScrollbar.in(t)

	t.Font = "" // 系统 UI 字体栈（U2 §1.3）
	t.FontSize = typeBodySize
	t.Spacing = spaceUnit
	t.Radius = radius
	t.FocusRingWidth = strokeFocus
	t.FocusRingOffset = focusRingOffset
	t.ScrollbarWidth = scrollbarWidth
	t.TooltipDelay = tooltipDelay

	// SetTheme 的主题不经过框架的 Theme.follow，这里按它的同一套规则跟随
	// 系统设置（ui/preferences.go 的 follow）：文本缩放乘字号；高对比把
	// Border/TextMuted 往 Text 混、Focus 转不透明、Scrollbar 加深。
	if s := prefs.TextScale; s > 0 && s != 1 {
		t.FontSize *= s
	}
	if prefs.HighContrast {
		t.Border = t.Border.Mix(t.Text, 0.45)
		t.TextMuted = t.TextMuted.Mix(t.Text, 0.4)
		t.Focus.A = 255
		t.Scrollbar = t.Scrollbar.Alpha(1.6)
	}
	return t
}
