package ui

import (
	"strings"
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

	// R4 输入位：卡片 60–300，上沿可带一行队列（33）。
	ComposerMinHeight = 60
	ComposerMaxHeight = 300
	ComposerHeight    = 124
	ApprovalHeight    = 150
	QueueHeight       = 33

	// R3 的宽度下限：右栏打开时主区不缩到 400 以下。
	MainMinWidth = 400

	// R6 设置（独立屏）：左导航 224 + 内容区上限 1008（U0 §12.7、U1-ia §4）。
	SettingsNavWidth = 224
	// SettingsNavRowHeight 是左导航一节的行高：参考实测 35（相邻两节的 y 步进也是
	// 35，节间不再另加间隙）。
	SettingsNavRowHeight = 35

	SettingsContentMaxWidth  = 1008
	SettingsLabelWidth       = 140
	SettingsRuleCommandWidth = 240
	SettingsRuleSelectWidth  = 120
	// 设置 · 扩展：行内名称与版本的定宽，以及"一行放得下"的最小内宽。
	extNameWidth    = 200
	extVersionWidth = 72
	extRowMinWidth  = extNameWidth + extVersionWidth + space3*3 + 36
	// cronScheduleWidth 是定时任务里「计划」一列的定宽（如 "每天 09:00"）。
	cronScheduleWidth = 120
	// cronRowMinWidth 是一行放得下「名称 + 计划 + 下次 + 开关 + 删除」的最小内宽。
	// keyHintKeysWidth 是快捷键提示表里键位一列的定宽（最长 "Cmd+1…5"）。
	keyHintKeysWidth = 120
	cronRowMinWidth  = extNameWidth + cronScheduleWidth + 72 + space2*5 + rowCompact*2

	// SettingsRuleRowMinWidth 是"命令 + 决定 + 匹配方式"排在一行所需的最小内宽；
	// 窄于它就按行堆叠（一条规则两行），见 U2-forms F23 的退化规则。
	SettingsRuleRowMinWidth = SettingsRuleCommandWidth + space4 + SettingsRuleSelectWidth +
		space4 + SettingsRuleSelectWidth

	// R5 右栏：280–（容器−400）。
	RightPanelMinWidth       = 280
	RightPanelPreferredWidth = 320
	RightPanelMainMinWidth   = 400

	// CompactWidth 是窄窗分档的阈值：窗口窄于它时按 compact 排（U1-ia §3.1
	// 的窄窗行）。窄窗只改七处形态，别处不变：
	//   - 左栏只留边条，展开是覆盖式抽屉（不挤压主区）；
	//   - 右栏整屏替代主区（不与主区并排）；
	//   - 标题行收成「会话名 + 分支 + 面板开关」，会话名单行截断；
	//   - 输入区控制行竖排（工作区一行、权限一行）；
	//   - 设置左导航收成顶部下拉；
	//   - 命令面板与对话框改满宽、左右留 narrowDialogMargin；
	//   - 1px 线在绘制时按 DPR 取整（hairline.go）。
	// 验收档取 720（scripts/capture-windows.py 的第三档）。
	CompactWidth = 800

	// narrowTitleMaxWidth 是窄窗标题行里会话名的宽度上限：超出单行截断，把
	// 余量让给分支标记、打开方式与面板开关；全名仍在读屏名里。
	narrowTitleMaxWidth = 168
	// narrowDialogMargin 是窄窗命令面板与对话框左右留白（space12＝42）。它要
	// 大于框架对话框自身的留白（Modal 的内容区最多「窗口−space10−2×space5」），
	// 否则内容会顶出对话框的圆角面板。
	narrowDialogMargin = space12
)

// 视觉 token：docs/domains/U2-visual.md §8 的 token 表（v2：默认优先），
// 值只在这里出现一次。
//
// 颜色不写在元素上：主题有字段的取 c.Theme()，框架没有字段的 accent-ink
// （强调色作文字/标记）取本文件的 colorXxx.in(t)。边界只有一套：主题的 Border
// （UI-18），没有第二级边框色。
const (
	// 间距：参考产品的根字号 14px，0.25rem = 3.5px，七档都按 3.5 的倍数
	// （design-baseline §1.1：3.5 / 7 / 10.5 / 14 / 21 / 28 / 42）。
	spaceUnit = 3.5
	space1    = 3.5
	space2    = 7
	space3    = 10.5
	space4    = 14
	space6    = 21
	space8    = 28
	space12   = 42

	// 控件内边距：取参考产品 h-9 / px-4 的 6 / 14（design-baseline §3）。
	// 框架按钮自己按 t.Space 推内边距；自绘的文字按钮不再另定一套档位
	// （U0 §control-baseline）。
	controlPadV = 6
	controlPadH = 14

	// 密度：控件高度（U2 §8.3）。
	rowRegular  = 28
	rowCompact  = 24
	controlMini = 20

	// 图标与状态点（U2 §8.3）。
	iconInline = 16
	iconBlock  = 20
	dotStatus  = 8
	// dotContext 是输入区用量"将满"时的圆点：图标体系里的最小档（U2 §4.5）。
	dotContext = 6

	// 描边与焦点环（U2 §8.3）。
	strokeHairline  = 1
	strokeFocus     = 2
	strokeMarker    = 2
	focusRingOffset = 1

	// 圆角：参考产品由 --radius 0.625rem 派生（design-baseline §4）。
	// 控件与输入框用 md 6.75；卡片用 lg 8.75；主窗卡用 xl 12.75；
	// 小标签与内层块用 sm 4.75。框架 Theme.Radius 只画控件，取控件档。
	radiusSmall      = 4.75
	radius           = 6.75
	radiusCard       = 8.75
	radiusWindowCard = 12.75
	radiusCode       = radius + 2

	// 方向指示的描边：24 网格的 1.5 描边在 16px 盒上约 1px（U2 §4.4、§6.4）。
	strokeChevron = 1.5

	// contextNearPercent 是"上下文将满"的界面口径（U3 T10）：已用 / 上限达到
	// 这个百分比就换颜色与附词。它是可调参数，不是协议事实。
	contextNearPercent = 90

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
	railMarkWidth         = 6
	railMarkHeight        = 2
	// placeholderInput 是空输入框的最小占位：四个字以内，只说明"这里能写"，
	// 不写引导语（UI-23；R6 裁决二选一里选了占位而不是默认聚焦）。
	placeholderInput = "输入"

	// placeholderGoal 是目标编辑框的占位（2 字，与输入框同一策略）。
	placeholderGoal = "目标"
	// placeholderShell 是命令模式的占位（2 字，与普通输入同一策略）。
	placeholderShell = "命令"

	// sidebarTextInset 是侧栏里与列表行文字对齐的左内边距：框架 Sidebar 的内边距 8
	// + 行内边距 8 + 图标槽 16 + 图标与文字间距 8 = 40。它不是魔法数字，是从
	// SidebarItem 的排布推出的同一条文字列（UI-22）。
	sidebarTextInset = space8 + space2

	// headerTargetsMax 是组头里每组最多写几个对象、headerGroupsMax 是最多几组
	// 动作，多出的用省略号收住；完整清单在展开里。
	headerTargetsMax = 2
	headerGroupsMax  = 3
	// headerPhraseMaxRunes 是组头可见文字的字符上限：超过就按对象为单位截断，
	// 动作与对象的第一段一定留住（R7）。
	headerPhraseMaxRunes = 34
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
	// typeControl 是"控件里的文字"：不设行高，行盒由框架按字形度量给——与官方
	// Button / Select / TextInput 同基准（U0 §control-baseline）。字号取参考产品的
	// text-sm：根字号 14 下 0.875rem = 12.25（design-baseline §1.1）。
	typeControl       = fontStep{size: 12.25, weight: 400}
	typeControlStrong = fontStep{size: 12.25, weight: 600}

	// 正文行高取参考 text-sm 的实测行盒 18（design-baseline §2：行高 17.5、
	// 块级行盒 18），不再用 24.5 的 2 倍行距。
	typeBody            = fontStep{size: 12.25, line: 18, weight: 400}
	typeBodyStrong      = fontStep{size: 12.25, line: 18, weight: 600}
	typeSecondary       = fontStep{size: 12.25, weight: 400}
	typeSecondaryStrong = fontStep{size: 12.25, weight: 500}
	// typeTitleRow 是标题行的会话名：跟官方示例的页面标题同一档（15/600，
	// 见 notes.png 的 24 与 permissionflow.png 的 20 之间的最小档；标题行只有 48 高）。
	typeTitleRow = fontStep{size: 15, weight: 600}
	typeMicro    = fontStep{size: 10.5, line: 14, weight: 400}
	typeCode     = fontStep{size: 13, line: 20, weight: 400, family: codeFont}
	typeH1       = fontStep{size: 22.4, line: 33.6, weight: 700}
	typeH2       = fontStep{size: 20, line: 25, weight: 600}
	typeH3       = fontStep{size: 16.1, line: 24.15, weight: 600}
	typeH4       = fontStep{size: 14, line: 21, weight: 600}
	typeH5       = fontStep{size: 13.3, line: 19.95, weight: 600}
	typeH6       = fontStep{size: 12.6, line: 18.9, weight: 600}
)

// codeFontStack 是代码的字体栈底：系统等宽，不内嵌字体（U2 §1.3）。
const codeFontStack = "SF Mono, Menlo, monospace"

// codeFont 是代码字级的族标记：字级里写它表示"这段是代码"，渲染时换成当前偏好
// 的代码字体栈与字号（见 currentCodeLook）。它不是最终的字体名。
const codeFont = codeFontStack

// defaultCodeSize 是代码字号的默认档（typeCode 的 13）：偏好里的代码字号按它换算
// 成倍率。
const defaultCodeSize = 13

// codeLook 是本帧代码文字用的字体栈与字号倍率。
type codeLook struct {
	family string
	scale  float32
}

// currentCodeLook 是本帧代码文字用的字体栈与字号倍率：applyTokens 每帧按本地偏好
// 写一次（本地偏好全局一份，各窗口同值），所有代码字级渲染时读它——所以改设置
// 下一帧就生效，不需要重启。
var currentCodeLook = codeLook{family: codeFontStack, scale: 1}

// codeFamily 返回本帧代码文字用的字体栈。
func codeFamily() string { return currentCodeLook.family }

// codeFamilyFor 把偏好里的代码字体名换成字体栈：空名 = 系统等宽；具体字体在前、
// 系统等宽兜底（选中的字体不在本机时仍有等宽可读）。
func codeFamilyFor(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == codeFontStack {
		return codeFontStack
	}
	return name + ", " + codeFontStack
}

// codeScaleFor 把偏好里的代码字号换成相对默认档的倍率；越界按默认。
func codeScaleFor(size float32) float32 {
	if size < 10 || size > 24 {
		return 1
	}
	return size / defaultCodeSize
}

// typeBodySize 是参考产品的根字号（14），也是主题 FontSize 的基准：本帧的平台字号
// 与它相比得到文本缩放。正文本身取 text-sm = 0.875 × 根字号（12.25，见 typeBody）。
const typeBodySize = 14

// textScale 是本帧文字相对正文的倍数：主题字号由框架按系统文本缩放放大，
// 这里用它与 14 的比把同一缩放带到每一级字号上。
func textScale(t *gui.Theme) float32 {
	if t == nil || t.FontSize <= 0 {
		return 1
	}
	return t.FontSize / typeBodySize
}

// sizeIn 返回该字级在 t 下的字号，供 span 与需要数字的地方使用。代码字级再乘
// 本帧的代码字号倍率（设置 › 外观 › 代码字号）。
func (f fontStep) sizeIn(t *gui.Theme) float32 {
	size := f.size * textScale(t)
	if f.family == codeFont {
		size *= currentCodeLook.scale
	}
	return size
}

// style 把字级写到 e：字号（含文本缩放）、行高与字重。行高按倍数给，
// 因此文本放大后行盒等比变高，不会截断（U2 §7.4）；line 为 0 时不设行高，
// 由框架按字体度量排版（U2 §1.1）。
func (f fontStep) style(t *gui.Theme, e gui.Element) gui.Element {
	family := f.family
	if family == codeFont {
		family = codeFamily()
	}
	if family != "" {
		e = e.Font(family)
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

// 颜色 token：每主题 18 个设计 token（其中 accent-ink、输入框底 field 框架没有
// 字段），另 3 个沿用框架、不参与设计（U2 §8.1）。明暗层级固定为
// 底 → 卡 → 浮层 → 交互面（design-baseline §5.3）：
// 明 #f6fdff → #ffffff → #ffffff → #f5f5f5；暗 #090c0d → #0c0c0c → #222222 → #262626。
// 边界只取主题的 Border，不再有第二级边框色（UI-18）；输入框底是卡面之上的
// 独立一档（明 #fafafa / 暗 #121212），不参与这四级层级。
var (
	colorBG             = colorSet{gui.Hex("#f6fdff"), gui.Hex("#090c0d")}
	colorSurface        = colorSet{gui.Hex("#ffffff"), gui.Hex("#0c0c0c")}
	colorSurfaceHover   = colorSet{gui.Hex("#f5f5f5"), gui.Hex("#262626")}
	colorSurfacePressed = colorSet{gui.Hex("#ebebeb"), gui.Hex("#2b2b2b")}
	colorBorder         = colorSet{gui.Hex("#e5e5e5"), gui.Hex("#2e2e2e")}
	colorText           = colorSet{gui.Hex("#0a0a0a"), gui.Hex("#ececec")}
	colorTextSecondary  = colorSet{gui.Hex("#737373"), gui.Hex("#8a8a8a")}
	// 主色取 --primary 的 oklch → sRGB（明 oklch(0.6 0.138 150) = #349651，
	// 暗 oklch(0.76 0.15 149) = #63cb7b）。悬停 / 按下按亮度各降升一档派生。
	colorAccent        = colorSet{gui.Hex("#349651"), gui.Hex("#63cb7b")}
	colorAccentHover   = colorSet{gui.Hex("#40a35c"), gui.Hex("#71d887")}
	colorAccentPressed = colorSet{gui.Hex("#218a45"), gui.Hex("#5ebc72")}
	colorAccentOn      = colorSet{gui.Hex("#ffffff"), gui.Hex("#010602")}
	// accent-ink 是强调色作文字 / 标记：作填充的主色在明色下对白底只有 3.7，
	// 不能兼作正文，取同色相更深一档（明 5.98 对底、6.15 对卡）。
	colorAccentInk = colorSet{gui.Hex("#197037"), gui.Hex("#63cb7b")}
	colorSuccess   = colorSet{gui.Hex("#009966"), gui.Hex("#00d492")}
	colorWarning   = colorSet{gui.Hex("#e17100"), gui.Hex("#ffb900")}
	colorDanger    = colorSet{gui.Hex("#e7000b"), gui.Hex("#ff6467")}
	colorSelection = colorSet{gui.RGBA(52, 150, 81, 0.20), gui.RGBA(99, 203, 123, 0.30)}
	// 焦点环：明 --ring #a1a1a1，暗 --brand-green #63cb7b（design-baseline §5）。
	colorFocus = colorSet{gui.Hex("#a1a1a1"), gui.Hex("#63cb7b")}

	// 沿用框架默认、不参与设计的三个字段（U2 §8.1）。
	colorInverse     = colorSet{gui.Hex("#0a0a0a"), gui.Hex("#ececec")}
	colorInverseText = colorSet{gui.Hex("#ffffff"), gui.Hex("#0c0c0c")}
	colorScrollbar   = colorSet{gui.RGBA(0, 0, 0, 0.32), gui.RGBA(255, 255, 255, 0.35)}
)

// 输入框底：参考实测 #fafafa（比卡面白一档、比窗口底灰一档）。暗色按基线的同一
// 层级取近值 #121212（design-baseline §5.2 的代码块底，卡片 #0c0c0c 之上）。
var colorField = colorSet{gui.Hex("#fafafa"), gui.Hex("#121212")}

// 品牌标记的两个色（参考实测，明暗同一值——它们是品牌色，不随主题变）：
// 空态标记 #74bc2f、账户标记 #42894c。
var (
	colorBrandEmpty   = gui.Hex("#74bc2f")
	colorBrandAccount = gui.Hex("#42894c")
	// colorBrandOn 是品牌标记上的前景（账户头像的首字母）。
	colorBrandOn = gui.Hex("#ffffff")
)

// fieldFill 是输入框的底色。
func fieldFill(t *gui.Theme) gui.Color { return colorField.in(t) }

// appearanceLook 是「设置 › 外观」里会真正影响渲染的几项，由本地偏好算出。
type appearanceLook struct {
	// font 是界面字体族；空 = 系统 UI 字体栈。
	font string
	// scale 是界面字号倍率；0 或 1 = 平台默认。
	scale float32
}

// themeFor 在 theme 之上套「设置 › 外观」的本地偏好：界面字体与界面字号。
// 字号改在主题的 FontSize 上——所有字级都按它等比缩放（textScale）。
func themeFor(dark bool, prefs gui.Preferences, look appearanceLook) *gui.Theme {
	t := theme(dark, prefs)
	if name := strings.TrimSpace(look.font); name != "" {
		t.Font = name
	}
	if look.scale > 0 && look.scale != 1 {
		t.FontSize *= look.scale
	}
	return t
}

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
	// 字号不覆盖：用框架的平台默认（ui/theme.go 的 defaultFontSize：macOS 13、
	// 其它 14）。我们的字阶按它等比缩放（textScale），所以每条都与官方控件同基准
	// （UI-24：并排对照；U0 §control-baseline §3 的对照表第 1、2 行）。
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
