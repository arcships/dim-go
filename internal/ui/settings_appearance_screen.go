package ui

import (
	"fmt"
	"strconv"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 外观」的重建屏：元素按
// docs/domains/requirements/inventory-settings-rt.md §4（S-appearance-01 … 94，
// 94 个元素）逐行对应，文案逐字取参考产品中文，数值取
// docs/domains/requirements/design-baseline.md：
//
//	主题卡 240×69、皮肤卡 240×98、三列卡间距 10、卡片圆角 8、控件高 32、
//	开关 28×16、说明字 10.5、键帽 11、页面标题 17.5、行距 60（快捷键）。
//
// 本文件同时放三屏共用的构件（screenTitle / screenCard / screenRule /
// screenHint / screenInput / screenIcon …），settings_general_screen.go 与
// settings_keybindings_screen.go 直接引用。
//
// 主题之外的可丢界面状态都在 appearanceScreen 里：语言、字体、字号、代码主题、
// 皮肤，皮肤的图片与高级折叠，删除对话框。它们都没有写入通道（也没有落盘），
// 接线时换成 Intents。
//
// 主题不走 appearanceScreen：卡片的选中态读 App.themePref（真正决定配色的那个
// 本地偏好），点卡片就是改主题并落盘（selectTheme），不是只移动一个角标。
//
// 两处没有资产，用等尺寸示意：主题卡的预览是 Sun / Moon / Desktop 图标，
// 皮肤卡的缩略图是按皮肤配色画的窗口示意（不是原始素材）。

const (
	// 间距半档：参考产品在本屏用到 2 / 6 / 10 / 14 / 20，比七档标尺更密
	// （design-baseline §1.1 的 3.5 倍档在 2× 截图上落成这些整数）。
	screenSpace05 = 2
	screenSpace1  = 4
	screenSpace15 = 6
	screenSpace2  = 8
	screenSpace25 = 10
	screenSpace3  = 12
	screenSpace35 = 14
	screenSpace4  = 16
	screenSpace5  = 20

	// 几何：design-baseline §3、§4、§7。圆角取基线 §4 的卡内分区 8.75 与
	// 控件 6.75（tokens.go 的 radiusCard / radius），不再取参考截图拟合时的整数。
	screenStroke     = 1
	screenRadiusCard = radiusCard
	screenRadiusCtl  = radius
	screenCardGap    = 10
	screenControlH   = 32
	screenInputPadV  = 8
	screenSwitchW    = 28
	screenSwitchH    = 16
	screenIconBlock  = 20
	screenIconInline = 16
	screenIconMicro  = 14
	screenDotSize    = 16
	screenDotInset   = 3
	screenHit        = hitMin
	screenDialogW    = 448
	screenDialogMinW = 160
	screenRuleH      = 1

	// 本屏尺寸（参考产品的 Tailwind 确定值，见 design-baseline §1.3、§3）。
	screenThemeCardW = 240
	screenThemeCardH = 69
	screenGridW      = screenThemeCardW*3 + screenCardGap*2
	screenSkinCardW  = 240
	screenSkinCardH  = 98
	screenSkinThumbH = 58
	screenSkinTitleH = 16
	screenLangW      = 168
	screenFontSelW   = 224
	screenSegHSm     = 25
	screenSegHLg     = 41
	screenKeyRowH    = 60
	screenKeyMinW    = 45
	screenSearchW    = 392
	screenCodeLineH  = 18
	screenCodeNumW   = 18
)

// 字阶（design-baseline §2）：页面标题 17.5/24.5（typePageTitle）、标签 14/500、
// 说明 10.5、分组标题与刻度 12.25/10.5。
var (
	screenTitleStyle  = typePageTitle
	screenLabelStyle  = fontStep{size: 14, line: 21, weight: 500}
	screenValueStyle  = fontStep{size: 14, line: 21, weight: 400}
	screenStrongStyle = fontStep{size: 14, line: 21, weight: 600}
	screenHintStyle   = fontStep{size: 12.25, line: 17.5, weight: 400}
	screenGroupStyle  = fontStep{size: 12.25, line: 17.5, weight: 600}
	screenTickStyle   = fontStep{size: 11, line: 18, weight: 400}
	screenCodeStyle   = fontStep{size: 12, line: 17, weight: 400, family: codeFont}
)

// 本屏图标：Heroicons v2 outline（24 网格、1.5 描边、MIT）的内联路径，
// 与 icons.go 同一套约定。
var (
	screenIconSun         = mustSVG(`<path d="M12 3v2.25m6.364.386-1.591 1.591M21 12h-2.25m-.386 6.364-1.591-1.591M12 18.75V21m-4.773-4.227-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0Z"/>`)
	screenIconMoon        = mustSVG(`<path d="M21.752 15.002A9.72 9.72 0 0 1 18 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 0 0 3 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 0 0 9.002-5.998Z"/>`)
	screenIconDesktop     = mustSVG(`<path d="M9 17.25v1.007a3 3 0 0 1-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0 1 15 18.257V17.25m6-12V15a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 15V5.25m18 0A2.25 2.25 0 0 0 18.75 3H5.25A2.25 2.25 0 0 0 3 5.25m18 0V12a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 12V5.25"/>`)
	screenIconChevronDown = mustSVG(`<path d="m19.5 8.25-7.5 7.5-7.5-7.5"/>`)
	screenIconFolderOpen  = mustSVG(`<path d="M3.75 9.776c.112-.017.227-.026.344-.026h15.812c.117 0 .232.009.344.026m-16.5 0a2.25 2.25 0 0 0-1.883 2.542l.857 6a2.25 2.25 0 0 0 2.227 1.932H19.05a2.25 2.25 0 0 0 2.227-1.932l.857-6a2.25 2.25 0 0 0-1.883-2.542m-16.5 0V6A2.25 2.25 0 0 1 6 3.75h3.879a1.5 1.5 0 0 1 1.06.44l2.122 2.12a1.5 1.5 0 0 0 1.06.44H18A2.25 2.25 0 0 1 20.25 9v.776"/>`)
	screenIconTrash       = mustSVG(`<path d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0"/>`)
	screenIconRefresh     = mustSVG(`<path d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99"/>`)
	screenIconBan         = mustSVG(`<path d="M18.364 18.364A9 9 0 0 0 5.636 5.636m12.728 12.728A9 9 0 0 1 5.636 5.636m12.728 12.728L5.636 5.636"/>`)
	screenIconSliders     = mustSVG(`<path d="M10.5 6h9.75M10.5 6a1.5 1.5 0 1 1-3 0m3 0a1.5 1.5 0 1 0-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-9.75 0h9.75"/>`)
)

// ---- 三屏共用构件 ----

// screenTitle 是设置分节的页面标题：与账户屏同级（typePageTitle、二级标题）。
func screenTitle(c *gui.Context, text string) {
	t := c.Theme()
	screenTitleStyle.style(t, gui.Text(c, text)).TextColor(t.Text).
		Role(gui.RoleHeading).Level(2).SingleLine()
}

// screenLabel 是行标签：14、中等字重、通栏。
func screenLabel(c *gui.Context, text string) {
	t := c.Theme()
	screenLabelStyle.style(t, gui.Text(c, text)).TextColor(t.Text).Label(text)
}

// screenValue 是一行取值（14、常规字重）。
func screenValue(c *gui.Context, text string) {
	t := c.Theme()
	screenValueStyle.style(t, gui.Text(c, text)).TextColor(t.Text)
}

// screenHint 是说明文字：10.5、次级前景、可选中复制。
func screenHint(c *gui.Context, text string) {
	t := c.Theme()
	screenHintStyle.style(t, gui.Text(c, text)).TextColor(t.TextMuted).Selectable()
}

// screenGroupTitle 是分组标题（12.25、半粗、次级前景）。
func screenGroupTitle(c *gui.Context, text string) {
	t := c.Theme()
	screenGroupStyle.style(t, gui.Text(c, text)).TextColor(t.TextMuted).Label(text)
}

// screenTick 是刻度文字（10.5、次级前景）。
func screenTick(c *gui.Context, text string) {
	t := c.Theme()
	screenTickStyle.style(t, gui.Text(c, text)).TextColor(t.TextMuted).SingleLine()
}

// screenRule 是分节之间的 1px 细线：用 1px 高的面差表达，不新增自绘描边。
func screenRule(c *gui.Context) {
	t := c.Theme()
	gui.Box(c).Shrink(0).Height(screenRuleH).Background(t.Border).
		Label("分隔线").AccessHidden(true)
}

// screenCard 是设置页的卡：卡面（白 / 暗 0c0c0c）、卡片圆角、1px 主题描边
// （描边走 Draw，着色取 t.Border）。
func screenCard(c *gui.Context, body func()) {
	t := c.Theme()
	card := gui.Column(c).Shrink(0).Background(t.Surface).Radius(screenRadiusCard)
	card.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCard, screenStroke)
	})
	card.Children(body)
}

// screenIcon 画一个装饰图标：尺寸固定，不随字号变化。
func screenIcon(c *gui.Context, icon *gui.SVG, size float32, color gui.Color, mark string) {
	box := gui.Box(c).Width(size).Height(size).Shrink(0)
	if mark != "" {
		box = box.Label("图标：" + mark).AccessHidden(true)
	}
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		paintIcon(p, icon, r, color)
	})
}

// screenIconButton 是一个只有图标的入口：命中区不小于 hit-min。
func screenIconButton(c *gui.Context, icon *gui.SVG, label, tip string, color gui.Color) bool {
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Size(screenHit, screenHit).Radius(screenRadiusCtl).
		Center().Label(label).Tooltip(tip)
	ink := color
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
		ink = t.Text
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
		ink = t.Text
	}
	b.Children(func() {
		screenIcon(c, icon, screenIconMicro, ink, label)
	})
	return b.Clicked()
}

// screenIconRotated 画一个旋转的装饰图标（折叠箭头）。
func screenIconRotated(c *gui.Context, icon *gui.SVG, size float32, color gui.Color, mark string, degrees float32) {
	box := gui.Box(c).Width(size).Height(size).Shrink(0)
	if mark != "" {
		box = box.Label("图标：" + mark).AccessHidden(true)
	}
	if degrees != 0 {
		box = box.Rotate(degrees)
	}
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		paintIcon(p, icon, r, color)
	})
}

// screenCornerDot 是贴在容器右上角的选中角标（绝对定位，不占行高）。
func screenCornerDot(c *gui.Context) {
	t := c.Theme()
	box := gui.Box(c).Width(screenDotSize).Height(screenDotSize).Shrink(0).
		Absolute().Top(0).Right(0).Label("已选中").AccessHidden(true)
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, t.Accent, screenDotSize/2)
		inner := gui.Rect{
			X: r.X + screenDotInset, Y: r.Y + screenDotInset,
			W: r.W - 2*screenDotInset, H: r.H - 2*screenDotInset,
		}
		paintIcon(p, iconCheck, inner, t.AccentText)
	})
}

// screenButton 是一个自绘的文字按钮（描边 / 主色两态），返回是否被按下。
func screenButton(c *gui.Context, text, label, tip string, primary bool, height float32) bool {
	t := c.Theme()
	if label == "" {
		label = text
	}
	if height == 0 {
		height = screenControlH
	}
	b := gui.ButtonBase(c).Shrink(0).Height(height).Padding(controlPadV, controlPadH).
		Radius(screenRadiusCtl).Label(label).Tooltip(tip)
	if primary {
		b = b.Background(t.Accent)
	} else {
		b = b.Background(t.Background)
		b.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
		})
	}
	ink := t.Text
	switch {
	case primary && b.Pressed():
		b = b.Background(t.AccentPressed)
	case primary && b.Hovered():
		b = b.Background(t.AccentHover)
	case !primary && b.Pressed():
		b = b.Background(t.SurfacePressed)
	case !primary && b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	if primary {
		ink = t.AccentText
	}
	b.Children(func() {
		screenValueStyle.style(t, gui.Text(c, text)).TextColor(ink).SingleLine()
	})
	return b.Clicked()
}

// screenTextButtonDisabled 是一个置灰的文字按钮（描边态）：形态与 screenButton
// 一致，点击无效，原因进 tooltip。用于"读不到设置但形态要留"的地方。
func screenTextButtonDisabled(c *gui.Context, text, tip string, height float32) {
	t := c.Theme()
	if height == 0 {
		height = screenControlH
	}
	b := gui.ButtonBase(c).Shrink(0).Height(height).Padding(controlPadV, controlPadH).
		Radius(screenRadiusCtl).Background(t.Background).Label(text).Tooltip(tip).
		Disabled(true).Opacity(noteDisabledAlpha)
	b.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	b.Children(func() {
		screenValueStyle.style(t, gui.Text(c, text)).TextColor(t.Text).SingleLine()
	})
}

// screenActionDisabled 是「图标 + 文字」描边按钮的禁用态（重启 App）：disabled
// 为假时退回可点的 screenAction。
func screenActionDisabled(c *gui.Context, icon *gui.SVG, mark, text string,
	height float32, disabled bool, tip string) {
	if !disabled {
		screenAction(c, icon, mark, text, height)
		return
	}
	t := c.Theme()
	if height == 0 {
		height = screenControlH
	}
	b := gui.ButtonBase(c).Shrink(0).Height(height).Padding(controlPadV, controlPadH).
		Radius(screenRadiusCtl).Background(t.Background).Label(text).Tooltip(tip).
		Disabled(true).Opacity(noteDisabledAlpha)
	b.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	b.Children(func() {
		gui.Row(c).Gap(screenSpace15).AlignItems(gui.Center).Children(func() {
			screenIcon(c, icon, screenIconInline, t.Text, mark)
			screenValueStyle.style(t, gui.Text(c, text)).TextColor(t.Text).SingleLine()
		})
	})
}

// screenAction 是「图标 + 文字」的描边按钮。
func screenAction(c *gui.Context, icon *gui.SVG, mark, text string, height float32) bool {
	t := c.Theme()
	if height == 0 {
		height = screenControlH
	}
	b := gui.ButtonBase(c).Shrink(0).Height(height).Padding(controlPadV, controlPadH).
		Radius(screenRadiusCtl).Background(t.Background).Label(text)
	b.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		gui.Row(c).Gap(screenSpace15).AlignItems(gui.Center).Children(func() {
			screenIcon(c, icon, screenIconInline, t.Text, mark)
			screenValueStyle.style(t, gui.Text(c, text)).TextColor(t.Text).SingleLine()
		})
	})
	return b.Clicked()
}

// screenInput 是一个文本框：输入框底（明 #fafafa / 暗 #121212）、1px 主题描边、高 32。
func screenInput(c *gui.Context, value *string, placeholder, label string) gui.Element {
	t := c.Theme()
	return gui.TextInput(c, value).Label(label).Placeholder(placeholder).
		Height(screenControlH).Padding(screenInputPadV, screenSpace25).
		Background(fieldFill(t))
}

// screenIndexSelect 是取值为下标的自绘下拉（框架 Select 只收字符串）。
func screenIndexSelect(c *gui.Context, selected *int, options []string, label string) {
	t := c.Theme()
	parts := gui.SelectBase[int](c, selected)
	trigger := parts.Trigger.Grow(1).Height(screenControlH).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Background(t.Background).Justify(gui.SpaceBetween).Label(label)
	trigger.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	trigger.Children(func() {
		screenValueStyle.style(t, gui.Text(c, options[*selected])).TextColor(t.Text).SingleLine()
		screenIcon(c, screenIconChevronDown, screenIconMicro, t.TextMuted, "chevron-down")
	})
	parts.Popup(func(panel gui.Element) {
		panel.Padding(screenSpace1).Radius(screenRadiusCard).Background(t.Background)
		panel.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Stroke(r, t.Border, screenRadiusCard, screenStroke)
		})
		for i, option := range options {
			item := parts.Item(i).Padding(screenSpace15, screenSpace25).
				Radius(screenRadiusCtl)
			color := t.Text
			switch {
			case item.Highlighted():
				item = item.Background(t.Accent)
				color = t.AccentText
			case i == *selected:
				item = item.Background(t.Surface)
			}
			item.Children(func() {
				screenValueStyle.style(t, gui.Text(c, option)).TextColor(color).SingleLine()
			})
		}
	})
}

// screenDialogWidth 是对话框面板宽（参考 sm:max-w-md = 448），窗口更窄时收进来。
func screenDialogWidth(c *gui.Context) float32 {
	width, _ := c.Size()
	limit := width - 2*space8
	if limit < screenDialogW {
		if limit < screenDialogMinW {
			return screenDialogMinW
		}
		return limit
	}
	return screenDialogW
}

// screenDialogPanel 是一个模态面板：标题（三级）+ 正文 + 动作。
func screenDialogPanel(c *gui.Context, title string, body func()) {
	t := c.Theme()
	panel := gui.Column(c).Gap(screenSpace3).
		Role(gui.RoleDialog).Label(title)
	panel.Children(func() {
		typeDialogTitle.style(t, gui.Text(c, title)).TextColor(t.Text).
			Role(gui.RoleHeading).Level(3).SingleLine()
		body()
	})
	panel.Width(screenDialogWidth(c))
}

// ---- 外观屏 ----

// appearanceThemeOptions 是主题卡的三支，顺序与截图一致。
var appearanceThemeOptions = []struct {
	name string
	icon *gui.SVG
}{
	{"浅色", screenIconSun},
	{"深色", screenIconMoon},
	{"跟随系统", screenIconDesktop},
}

var (
	appearanceLanguages = []string{"简体中文", "English", "日本語", "한국어"}
	appearanceUIFonts   = []string{"默认", "PingFang SC", "Helvetica Neue", "Noto Sans SC"}
	appearanceCodeFonts = []string{"默认", "SF Mono", "Menlo", "JetBrains Mono"}
	// appearanceCodeThemes 的代码配色：前四套是「浅色 + 深色」成对的公开配色，
	// 后两套只有深色（参考的下拉分「浅色 + 深色」与「深色」两组）。选了深色
	// 主题时，即使界面是浅色，代码区也用它自己的深色底与字色。
	appearanceCodeThemes = []string{
		"GitHub", "One Dark Pro", "Solarized Light", "Vitesse Dark",
		"Monokai", "Night Owl",
	}
	// appearanceDarkOnlyThemes 是只有深色形态的那一组。
	appearanceDarkOnlyThemes = map[string]bool{
		"One Dark Pro": true, "Vitesse Dark": true, "Monokai": true, "Night Owl": true,
	}
	appearanceSkins      = []string{"Windows 98", "Windows XP", "Windows Vista", "macOS Big Sur", "OS X Tiger"}
	appearanceFits       = []string{"适应", "填充", "拉伸", "平铺"}
	appearancePlacements = []string{"左上", "上", "右上", "左", "中", "右", "左下", "下", "右下"}
	appearanceAttach     = []string{"随内容滚动", "固定"}
	appearanceTextInks   = []string{"自动", "浅色文字", "深色文字"}
)

// appearanceState 是本屏可丢的界面状态：皮肤草稿、对话框与折叠。不是事实。
//
// 主题、界面字体与字号、代码字体与字号、代码配色都不在其中：它们读 App 的
// themePref / appearance（会真正影响渲染、也会落盘的本地偏好）。
type appearanceState struct {
	skin        string
	skinImage   bool
	skinOpacity float64
	skinFit     int
	skinPlace   int
	skinAttach  int
	skinInk     int
	advanced    bool
	reloaded    bool

	// openChoice 是展开中的下拉（字体两行与代码主题，按 Label 区分）；空表示都
	// 收起。fontQuery 是字体面板的搜索词，fontErr 是系统字体清单读不到的原因。
	openChoice string
	fontQuery  string
	fontErr    string

	deleteFor string
	deleting  bool
	deleted   map[string]bool
}

var appearanceScreen = appearanceState{
	skin:        "", // 空 = 内置「默认」外观卡被选中
	skinOpacity: 60,
}

// renderAppearance 是设置 › 外观这一屏（设置壳的一个分节）。
func renderAppearance(c *gui.Context, a *App) {
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(screenSpace5).Children(func() {
				screenTitle(c, "外观")
				appearanceThemeSection(c, a)
				appearanceLanguageSection(c, a.wired())
				appearanceFontSection(c, a)
				appearanceSizeSection(c, a)
				appearanceCodeThemeSection(c, a)
				appearanceSkinSection(c, a.wired())
				appearanceActiveSkin(c)
			})
		})
	})
	appearanceDeleteDialog(c)
}

// appearanceThemeValues 是主题卡的顺序对应到本地偏好的主题词：「明」「暗」
// 「跟随系统」（app.go 的 theme 开关按这三个词选配色）。两个清单顺序必须一致。
var appearanceThemeValues = []string{"明", "暗", "跟随系统"}

// appearanceThemeIndex 把本地偏好的主题词换成卡片序号：认不出来的词按「跟随
// 系统」定位，但主题本身仍取原文（这里不改偏好的值）。
func appearanceThemeIndex(pref string) int {
	switch pref {
	case "明":
		return 0
	case "暗":
		return 1
	}
	return 2
}

// selectTheme 是主题卡的一次点选：改本地偏好里的主题并落盘；落盘失败就回滚到
// 原来那张卡（S-appearance-03 / 05 的「角标回滚」），失败原因进标题下的错误行。
//
// 主题是客户端自己的偏好（不是 App Server 的事实），所以这一项能真的改：换完
// 这一帧就换配色，下一次启动按落盘的值恢复。没有 Intents（-demo）时只改视图。
func (a *App) selectTheme(index int) {
	if index < 0 || index >= len(appearanceThemeValues) {
		return
	}
	next := appearanceThemeValues[index]
	if a.themePref == next {
		return
	}
	previous := a.themePref
	a.themePref = next
	a.persistPreferencesThen(func(saveErr string) {
		// 保存失败：主题退回保存前的值，下一帧角标回到原来那张卡。
		a.themePref = previous
	})
}

// appearanceThemeSection 是 4.1 的主题段（S-appearance-01–08）。
//
// 卡片的选中态读 a.themePref（真正决定配色的那个值）：点一张卡就是改主题，
// 不是只移动一个角度标。
func appearanceThemeSection(c *gui.Context, a *App) {
	t := c.Theme()
	selected := appearanceThemeIndex(a.themePref)
	gui.Column(c).Shrink(0).Gap(screenSpace25).Children(func() {
		screenLabel(c, "主题")
		gui.Row(c).Wrap().MaxWidth(screenGridW).
			GapX(screenCardGap).GapY(screenCardGap).
			Label("主题").Children(func() {
			for i, option := range appearanceThemeOptions {
				if appearanceThemeCard(c, i, option.name, option.icon, i == selected) {
					a.selectTheme(i)
				}
			}
		})
		// S-appearance-06：系统报不出外观时回落浅色，选中仍停在这张卡，并在行下
		// 写清为什么看到的是浅色。
		if sysErr := appearanceSystemAppearanceError(c, a.themePref); sysErr != "" {
			typeMicro.style(t, gui.Text(c, sysErr)).TextColor(t.TextMuted).
				Role(gui.RoleStatus).MaxLines(2)
		}
	})
	screenRule(c)
}

// appearanceSystemAppearanceError 是「跟随系统」卡的一句现状：系统报得出外观时
// 没有这句；报不出（没有系统主题，如无头运行）时返回原因，界面按浅色渲染。
func appearanceSystemAppearanceError(c *gui.Context, pref string) string {
	if pref != "跟随系统" || appearanceSystemKnown {
		return ""
	}
	return noteSystemAppearanceUnknown
}

// appearanceSystemKnown 是宿主这一帧有没有报出系统外观（applyTokens 每帧写一次）。
// 报告通道在现有平台上总是有值，所以这句在正常运行时不会出现；测试置假来验回落
// 分支（S-appearance-06）。
var appearanceSystemKnown = true

// appearanceThemeCard 是一张主题卡（240×69）：预览在上、文字在下、角标在右上。
// 返回这一帧是否被点中。
func appearanceThemeCard(c *gui.Context, index int, name string, icon *gui.SVG,
	selected bool) bool {
	t := c.Theme()
	frame := t.Border
	if selected {
		frame = t.Accent
	}
	b := gui.ButtonBase(c).Shrink(0).Width(screenThemeCardW).Height(screenThemeCardH).
		Radius(screenRadiusCard).Background(t.Surface).Padding(screenStroke).
		AlignItems(gui.Stretch).Label("主题：" + name).Tooltip(name).
		Role(gui.RoleRadio).AccessSelected(selected)
	if !selected {
		switch {
		case b.Pressed():
			frame = t.Border.Mix(t.Text, 0.45)
		case b.Hovered():
			frame = t.Border.Mix(t.Text, 0.3)
		}
	}
	b.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, frame, screenRadiusCard, screenStroke)
	})
	clicked := b.Clicked()
	b.Children(func() {
		gui.Box(c).Grow(1).Padding(screenSpace15, screenSpace15).Children(func() {
			gui.Column(c).Grow(1).Justify(gui.Center).AlignItems(gui.Center).
				Gap(screenSpace1).Children(func() {
				// S-appearance-04：预览图标缺资源时保留同样高度的底色块——文字
				// 不往上跳，卡高也不变。
				appearanceThemeGlyph(c, icon, name, t)
				screenStrongStyle.style(t, gui.Text(c, name)).TextColor(t.Text).SingleLine()
			})
			if selected {
				// S-appearance-07：选中标记缺资源时用纯色圆点兜底。
				appearanceSelectionMark(c, t)
			}
		})
	})
	return clicked
}

// appearanceThemeGlyph 是主题卡的预览图标位：有图标画图标，没有就画一块与图标
// 同尺寸的底色块（S-appearance-04）。
func appearanceThemeGlyph(c *gui.Context, icon *gui.SVG, name string, t *gui.Theme) {
	if icon == nil {
		box := gui.Box(c).Width(screenIconBlock).Height(screenIconBlock).Shrink(0).
			Radius(screenRadiusCtl).Label("图标：" + name).Tooltip(noteThemeIconMissing).
			AccessHidden(true)
		box.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Fill(r, t.SurfaceHover, screenRadiusCtl)
		})
		return
	}
	screenIcon(c, icon, screenIconBlock, t.TextMuted, name)
}

// appearanceSelectionMark 是主题卡的选中角标：图标缺资源时用纯色圆点兜底
// （S-appearance-07），两者命中区与位置一样。
func appearanceSelectionMark(c *gui.Context, t *gui.Theme) {
	if screenCornerDotGlyphMissing {
		box := gui.Box(c).Width(screenDotSize).Height(screenDotSize).Shrink(0).
			Absolute().Top(0).Right(0).Label("已选中").Tooltip(noteThemeMarkMissing).
			AccessHidden(true)
		box.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Fill(r, t.Accent, screenDotSize/2)
		})
		return
	}
	screenCornerDot(c)
}

// screenCornerDotGlyphMissing 表示选中角标里的对勾资源不可用：测试里置真来验
// 兜底的纯色圆点；正常运行走图标。
var screenCornerDotGlyphMissing bool

// appearanceLanguageSection 是程序语言段（S-appearance-09–11）。
// appearanceLanguageSection 是程序语言段（S-appearance-09–11）。
//
// 语言是一个"生效不了"的本地偏好：界面文案只有简体中文一套，没有 i18n 层。所以
// 连着 Source 时下拉照画但置灰，原因写在下面；-demo 里照参考可点（样例不改任何
// 文字，只是形态）。
func appearanceLanguageSection(c *gui.Context, live bool) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(screenSpace25).Children(func() {
		screenLabel(c, "程序语言")
		gui.Row(c).Children(func() {
			sel := gui.Select(c, &appearanceLanguage, appearanceLanguages).
				Width(screenLangW).Height(screenControlH).
				Background(t.Background).Label("程序语言")
			if live {
				sel = sel.Disabled(true).Tooltip(noteAppearanceLanguage).
					Opacity(noteDisabledAlpha)
			}
			gui.Spacer(c)
		})
		if live {
			typeMicro.style(t, gui.Text(c, noteAppearanceLanguage)).
				TextColor(t.TextMuted).Selectable()
		}
	})
	screenRule(c)
}

// appearanceLanguage 是语言下拉的取值：只有简体中文，且它不落盘（没有生效路径）。
var appearanceLanguage = appearanceLanguages[0]

// appearanceFontSection 是字体段（S-appearance-12–28）。
//
// 两项都是能真正生效的本地偏好：UI 字体写进主题的 Font，代码字体写进代码字级的
// 字体栈（见 tokens.go 的 codeFamilyFor），重启按落盘的值恢复。
func appearanceFontSection(c *gui.Context, a *App) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(screenSpace25).Children(func() {
		screenLabel(c, "字体")
		appearanceFontRow(c, t, "UI 字体", "用于应用界面和 Markdown 正文",
			appearanceUIFonts, a.appearance.UIFont, false, func(name string) {
				a.appearance.UIFont = name
				a.persistPreferences()
			})
		appearanceFontRow(c, t, "代码字体", "用于终端、代码块和差异内容",
			appearanceCodeFonts, a.appearance.CodeFont, true, func(name string) {
				a.appearance.CodeFont = name
				a.persistPreferences()
			})
	})
	screenRule(c)
}

// appearanceFontRow 是一行字体选择：左侧标签 + 说明，右侧下拉（224×32），点开
// 在下方展开字体面板（S-appearance-15 / 19）。
//
// 本机没有装这个字体时，面板里该项置灰并写原因（noteFontMissing）——选了也不会
// 生效，所以不让选。默认项恒可用（系统字体栈）。
func appearanceFontRow(c *gui.Context, t *gui.Theme, label, hint string,
	names []string, current string, monospaced bool, apply func(string)) {
	options := appearanceFontChoices(names, monospaced, current)
	index := appearanceChoiceIndex(options, current)
	open := appearanceScreen.openChoice == label
	gui.Column(c).Shrink(0).Gap(screenSpace15).Children(func() {
		gui.Row(c).AlignItems(gui.Center).Gap(screenSpace4).Children(func() {
			gui.Column(c).Grow(1).MinWidth(0).Gap(screenSpace05).Children(func() {
				screenLabel(c, label)
				screenHint(c, hint)
				// 只有"当前选中的这个字体本机没有"才在行下写原因：这一行说的是现状，
				// 不是面板里每一选项的说明。
				if !options[index].usable {
					typeMicro.style(t, gui.Text(c, options[index].reason)).
						TextColor(t.TextMuted).Selectable()
				}
			})
			if appearanceChoiceTrigger(c, t, label, options[index].label, open,
				screenFontSelW) {
				appearanceChoiceToggle(label, true)
			}
		})
		if open {
			appearanceFontPanel(c, t, options, index, apply)
		}
	})
}

// appearanceChoiceTrigger 是一个下拉的触发器：当前取值 + 箭头，点它开合面板。
// 取值本身不可用时整钮置灰并写原因。
func appearanceChoiceTrigger(c *gui.Context, t *gui.Theme, label, value string,
	open bool, width float32) bool {
	b := gui.ButtonBase(c).Shrink(0).Width(width).Height(screenControlH).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Background(t.Background).Justify(gui.SpaceBetween).Label(label).
		Tooltip(value).Cursor(gui.CursorPointer).
		Role(gui.RolePopUpButton).Expanded(open)
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	b.Children(func() {
		screenValueStyle.style(t, gui.Text(c, value)).TextColor(t.Text).SingleLine()
		screenIconRotated(c, screenIconChevronDown, screenIconMicro, t.TextMuted,
			"chevron-down", appearanceChevronDegrees(open))
	})
	return b.Clicked()
}

// appearanceChevronDegrees 是下拉箭头的角度：展开时朝上。
func appearanceChevronDegrees(open bool) float32 {
	if open {
		return 180
	}
	return 0
}

// appearanceChoiceToggle 开合一个下拉：打开字体面板时重新枚举一次系统字体，并清空
// 上一次的搜索词（S-appearance-20）。
func appearanceChoiceToggle(label string, fonts bool) {
	if appearanceScreen.openChoice == label {
		appearanceScreen.openChoice = ""
		appearanceScreen.fontErr = ""
		return
	}
	appearanceScreen.openChoice = label
	appearanceScreen.fontQuery = ""
	if fonts {
		appearanceFontRefresh()
	}
}

// appearanceFontRefresh 重新读一次系统字体清单：读不到（清单为空）就把原因留在
// appearanceScreen.fontErr，面板里换成错误行与「重试」（S-appearance-26）。
func appearanceFontRefresh() {
	if len(systemFontFamilies()) == 0 {
		appearanceScreen.fontErr = noteFontListMissing
		return
	}
	appearanceScreen.fontErr = ""
}

// appearanceFontPanel 是字体面板：搜索框 + 默认项 + 本机字体清单（S-20 / 22 / 23）。
// 搜索词不区分大小写、匹配族名子串；没有匹配时写一句空态。
func appearanceFontPanel(c *gui.Context, t *gui.Theme, options []appearanceFontChoice,
	selected int, apply func(string)) {
	gui.Column(c).Shrink(0).Width(screenFontSelW).Gap(screenSpace1).
		Padding(screenSpace1).Radius(screenRadiusCard).Background(t.Background).
		Label("字体面板").Children(func() {
		panel := gui.Column(c).Grow(1)
		panel.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Stroke(r, t.Border, screenRadiusCard, screenStroke)
		})
		panel.Children(func() {
			// S-appearance-20 搜索字体
			gui.Row(c).Shrink(0).Padding(screenSpace1).Children(func() {
				input := screenInput(c, &appearanceScreen.fontQuery, "搜索字体", "搜索字体")
				input.Grow(1).MinWidth(0)
			})
			if appearanceScreen.fontErr != "" {
				// S-appearance-25 / 26：读不到系统字体清单时写原因并给「重试」。
				gui.Column(c).Shrink(0).Gap(screenSpace1).
					Padding(screenSpace15, screenSpace25).Children(func() {
					typeMicro.style(t, gui.Text(c, appearanceScreen.fontErr)).
						TextColor(t.TextMuted).Role(gui.RoleAlert).MaxLines(2)
					if screenButton(c, "重试", "重试枚举系统字体", "", false, rowCompact) {
						appearanceFontRefresh()
					}
				})
				return
			}
			visible := appearanceFilterChoices(options, appearanceScreen.fontQuery)
			if len(visible) == 0 {
				// S-appearance-21 没有匹配的字体
				typeMicro.style(t, gui.Text(c, "没有匹配的字体")).
					TextColor(t.TextMuted).Padding(screenSpace15, screenSpace25)
				return
			}
			for _, item := range visible {
				appearanceFontItem(c, t, item, item.index == selected, func() {
					if item.choice.usable && apply != nil {
						apply(item.choice.value)
					}
					appearanceChoiceToggle("", false)
				})
			}
		})
	})
}

// appearanceFontItem 是字体面板里的一行（S-appearance-23）：族名 +（不可用时）
// 原因，选中项带对勾；点一行就应用该字体并关闭面板。
func appearanceFontItem(c *gui.Context, t *gui.Theme, item appearanceFilteredChoice,
	selected bool, pick func()) {
	row := gui.ButtonBase(c).Shrink(0).AlignItems(gui.Center).
		Padding(screenSpace15, screenSpace25).Radius(screenRadiusCtl).
		Label(item.choice.label).Cursor(gui.CursorPointer)
	switch {
	case !item.choice.usable:
		row = row.Disabled(true).Opacity(noteDisabledAlpha).Tooltip(item.choice.reason)
	case row.Hovered():
		row = row.Background(t.SurfaceHover)
	}
	if selected {
		row = row.AccessSelected(true)
	}
	if row.Clicked() {
		pick()
	}
	ink := t.Text
	if !item.choice.usable {
		ink = t.TextMuted
	}
	row.Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Children(func() {
			screenValueStyle.style(t, gui.Text(c, item.choice.label)).TextColor(ink).
				SingleLine()
			if !item.choice.usable {
				typeMicro.style(t, gui.Text(c, item.choice.reason)).TextColor(t.TextMuted).
					SingleLine()
			}
		})
		if selected {
			screenIcon(c, iconCheck, screenIconMicro, t.Accent, "对勾")
		}
	})
}

// appearanceFilteredChoice 是过滤后的一个候选及其在原清单里的序号。
type appearanceFilteredChoice struct {
	choice appearanceFontChoice
	index  int
}

// appearanceFilterChoices 按搜索词过滤字体：不区分大小写，匹配族名子串；查询为空
// 时全部返回（S-appearance-20 / 23）。
func appearanceFilterChoices(options []appearanceFontChoice,
	query string) []appearanceFilteredChoice {
	query = strings.TrimSpace(strings.ToLower(query))
	out := make([]appearanceFilteredChoice, 0, len(options))
	for i, option := range options {
		if query != "" && !strings.Contains(strings.ToLower(option.label), query) {
			continue
		}
		out = append(out, appearanceFilteredChoice{choice: option, index: i})
	}
	return out
}

// appearanceFontChoice 是一个字体选项：显示名、偏好里存的值、能不能选、为什么。
type appearanceFontChoice struct {
	label  string
	value  string
	usable bool
	reason string
	// group 是面板里的分组标题，空表示不分组（字体面板不分组）。
	group string
}

// appearanceFontChoices 把字体名清单换成选项：默认项（系统字体栈）永远可用；其余
// 按本机字体清单判可用性，清单为空（平台不报字体）时按可用处理。清单之后补上
// 本机字体里没列到的族，面板才是「系统字体列表」。
func appearanceFontChoices(names []string, monospaced bool, current string) []appearanceFontChoice {
	out := make([]appearanceFontChoice, 0, len(names))
	seen := make(map[string]bool, len(names))
	for i, name := range names {
		choice := appearanceFontChoice{label: name}
		if i > 0 {
			choice.value = name
		}
		choice.usable, choice.reason = appearanceFontUsable(name, monospaced)
		out = append(out, choice)
		seen[strings.ToLower(name)] = true
	}
	for _, family := range systemFontFamilies() {
		if family.Name == "" || seen[strings.ToLower(family.Name)] {
			continue
		}
		seen[strings.ToLower(family.Name)] = true
		choice := appearanceFontChoice{label: family.Name, value: family.Name, usable: true}
		if monospaced && !family.Monospaced {
			choice.usable, choice.reason = false, noteFontNotMonospaced
		}
		out = append(out, choice)
	}
	// 偏好里的字体不在清单里（旧版本写的、或本机卸掉了）：补一项当前值，
	// 否则下拉会显示成"默认"。
	if current != "" {
		found := false
		for _, choice := range out {
			if strings.EqualFold(choice.value, current) {
				found = true
			}
		}
		if !found {
			usable, reason := appearanceFontUsable(current, monospaced)
			out = append(out, appearanceFontChoice{label: current, value: current,
				usable: usable, reason: reason})
		}
	}
	return out
}

// systemFontFamilies 是本机字体清单；测试里可替换成固定清单。
var systemFontFamilies = gui.FontFamilies

// appearanceFontUsable 判一个字体名在本机能不能用：默认项恒可用；本机清单里没有
// 就不可用；平台不报字体（清单为空）时不判，按可用处理，免得全禁。
func appearanceFontUsable(name string, monospaced bool) (bool, string) {
	if name == "" || name == appearanceFontDefault {
		return true, ""
	}
	families := systemFontFamilies()
	if len(families) == 0 {
		return true, ""
	}
	for _, family := range families {
		if !strings.EqualFold(family.Name, name) {
			continue
		}
		if monospaced && !family.Monospaced {
			return false, noteFontNotMonospaced
		}
		return true, ""
	}
	return false, noteFontMissing
}

// appearanceFontDefault 是字体清单里"系统字体栈"那一项的名字。
const appearanceFontDefault = "默认"

// appearanceChoiceIndex 是当前取值对应的选项序号：按值（不含显示名）匹配，
// 认不出来按第一项（默认）。
func appearanceChoiceIndex(options []appearanceFontChoice, current string) int {
	for i, option := range options {
		if option.value == current {
			return i
		}
	}
	return 0
}

// appearanceCodeThemePanel 是代码主题面板：两组（浅色 + 深色 / 深色）标题 + 各套
// 配色，点一行就选中并收起（S-appearance-46 / 48）。
func appearanceCodeThemePanel(c *gui.Context, t *gui.Theme, options []appearanceFontChoice,
	selected int, apply func(string)) {
	gui.Column(c).Shrink(0).Width(appearanceCodeThemeW).Gap(screenSpace1).
		Padding(screenSpace1).Radius(screenRadiusCard).Background(t.Background).
		Label("代码主题面板").Children(func() {
		panel := gui.Column(c).Grow(1)
		panel.Draw(func(p *gui.Painter, r gui.Rect) {
			p.Stroke(r, t.Border, screenRadiusCard, screenStroke)
		})
		panel.Children(func() {
			group := ""
			for i, option := range options {
				if option.group != group {
					group = option.group
					screenGroupTitle(c, group)
				}
				appearanceCodeThemeItem(c, t, option, i == selected, func() {
					if apply != nil {
						apply(option.value)
					}
					appearanceChoiceToggle("", false)
				})
			}
		})
	})
}

// appearanceCodeThemeItem 是面板里的一行代码主题：名称 + 选中对勾，深色那组右侧
// 另有「深色」标记（选了它，浅色界面下代码区也是深色）。
func appearanceCodeThemeItem(c *gui.Context, t *gui.Theme, option appearanceFontChoice,
	selected bool, pick func()) {
	row := gui.ButtonBase(c).Shrink(0).AlignItems(gui.Center).
		Padding(screenSpace15, screenSpace25).Radius(screenRadiusCtl).
		Label(option.label).Cursor(gui.CursorPointer)
	if row.Hovered() {
		row = row.Background(t.SurfaceHover)
	}
	if selected {
		row = row.AccessSelected(true)
	}
	if row.Clicked() {
		pick()
	}
	row.Children(func() {
		screenValueStyle.style(t, gui.Text(c, option.label)).TextColor(t.Text).
			Grow(1).MinWidth(0).SingleLine()
		if appearanceDarkOnlyThemes[option.value] {
			typeMicro.style(t, gui.Text(c, "深色")).TextColor(t.TextMuted).SingleLine()
		}
		if selected {
			screenIcon(c, iconCheck, screenIconMicro, t.Accent, "对勾")
		}
	})
}

// appearanceSizeSection 是字号段（S-appearance-29–42）：UI 字号五档分段 +
// 代码字号滑块（12–20px，步长 1）。
//
// 两项都真的生效：UI 字号写成主题的字号倍率（所有字级等比放大），代码字号写成
// 代码字级的倍率（见 tokens.go）。改完落盘，重启按上次的档恢复。
func appearanceSizeSection(c *gui.Context, a *App) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(screenSpace35).Children(func() {
		gui.Row(c).AlignItems(gui.Center).Gap(screenSpace25).Children(func() {
			screenLabel(c, "UI 字号")
			gui.Spacer(c)
			screenTick(c, "小")
			if next, changed := appearanceSizeSegments(c,
				appearanceUISizeIndex(a.appearance.UIScale)); changed {
				a.appearance.UIScale = appearanceUISizes[next].scale
				a.persistPreferences()
			}
			screenTick(c, "大")
		})
		gui.Column(c).Shrink(0).Gap(screenSpace15).Children(func() {
			size := float64(appearanceCodeSize(a.appearance.CodeSize))
			gui.Row(c).AlignItems(gui.Center).Children(func() {
				screenLabel(c, "代码字号")
				gui.Spacer(c)
				screenValueStyle.style(t, gui.Text(c, appearanceCodeSizeLabel(size))).
					TextColor(t.Text).Label("代码字号当前值")
			})
			slider := gui.Slider(c, &size, appearanceCodeSizeMin, appearanceCodeSizeMax).Step(1).
				Label("代码字号").Grow(1)
			// S-appearance-39：焦点在滑块上时 Home / End 到两端；左右键每次 ±1 由框架
			// 的 Step 给，点轨道由拖拽落点给。
			if slider.Focused() {
				if c.KeyDown(gui.KeyEnd) {
					size = appearanceCodeSizeMax
				} else if c.KeyDown(gui.KeyHome) {
					size = appearanceCodeSizeMin
				}
			}
			if float32(size) != appearanceCodeSize(a.appearance.CodeSize) {
				a.appearance.CodeSize = float32(size)
				a.persistPreferences()
			}
			gui.Row(c).Children(func() {
				screenTick(c, "12px")
				gui.Spacer(c)
				screenTick(c, "16px")
				gui.Spacer(c)
				screenTick(c, "20px")
			})
		})
	})
	screenRule(c)
}

// appearanceUISizes 是 UI 字号的五档：显示名与字号倍率（参考的五档）。
var appearanceUISizes = []struct {
	name  string
	scale float32
}{
	{"特小", 0.85},
	{"小", 0.93},
	{"中", 1},
	{"大", 1.08},
	{"特大", 1.16},
}

// appearanceUISizeIndex 把偏好里的倍率换成档位序号；认不出来按「中」。
func appearanceUISizeIndex(scale float32) int {
	if scale <= 0 {
		return 2
	}
	best := 2
	bestDiff := float32(1)
	for i, size := range appearanceUISizes {
		diff := size.scale - scale
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			best, bestDiff = i, diff
		}
	}
	return best
}

// 代码字号的可用区间与默认档：与参考的滑块（12–20）一致，默认 13。
const (
	appearanceCodeSizeMin = 12
	appearanceCodeSizeMax = 20
)

// appearanceCodeSize 是偏好里的代码字号；0 表示默认档（13）。
func appearanceCodeSize(size float32) float32 {
	if size < appearanceCodeSizeMin || size > appearanceCodeSizeMax {
		return defaultCodeSize
	}
	return size
}

// appearanceCodeSizeLabel 是代码字号的读数（如「12px」，带单位）。
func appearanceCodeSizeLabel(size float64) string {
	return strconv.Itoa(int(size+0.5)) + "px"
}

// appearanceSizeSegments 是 UI 字号的五档分段：选中段主色底、反白字。
// 返回选中的档位与是否变了（变了由调用处写进本地偏好）。
func appearanceSizeSegments(c *gui.Context, selected int) (int, bool) {
	t := c.Theme()
	if selected < 0 || selected >= len(appearanceUISizes) {
		selected = 2
	}
	value := selected
	parts := gui.SegmentedBase(c, &value, len(appearanceUISizes))
	track := parts.Track.AlignItems(gui.Stretch).Gap(screenStroke).
		Radius(screenRadiusCtl).Background(t.Border).Padding(screenStroke).
		Label("UI 字号")
	// S-appearance-31：左右方向键在五档之间移动（框架的分组焦点给），Home / End
	// 直接到最小 / 最大档。
	if track.Focused() || track.FocusWithin() {
		if c.KeyDown(gui.KeyEnd) {
			value = len(appearanceUISizes) - 1
		} else if c.KeyDown(gui.KeyHome) {
			value = 0
		}
	}
	track.Children(func() {
		for i, size := range appearanceUISizes {
			seg := parts.Segment(i).Center().Height(screenSegHSm).
				Padding(0, screenSpace25).Radius(screenRadiusCtl).
				Background(t.Background).Label(size.name)
			if i == selected {
				seg = seg.Background(t.Accent)
			}
			seg.Children(func() {
				color, weight := t.Text, 500
				if i == selected {
					color, weight = t.AccentText, 600
				}
				gui.Text(c, size.name).FontSize(screenValueStyle.sizeIn(t)).
					FontWeight(weight).TextColor(color).SingleLine()
			})
		}
	})
	return value, value != selected
}

// ---- 代码主题 ----

// appearanceCodeToken 是预览里的一段代码着色。
type appearanceCodeToken struct {
	text string
	kind int
}

const (
	appearanceCodePlain = iota
	appearanceCodeKeyword
	appearanceCodeString
	appearanceCodeFunc
)

var appearanceCodeSample = [][]appearanceCodeToken{
	{{"function", appearanceCodeKeyword}, {" ", appearanceCodePlain}, {"greet", appearanceCodeFunc}, {"() {", appearanceCodePlain}},
	{{"  ", appearanceCodePlain}, {"const", appearanceCodeKeyword}, {" name = ", appearanceCodePlain}, {`"world"`, appearanceCodeString}},
	{{"  console.", appearanceCodePlain}, {"log", appearanceCodeFunc}, {"(name)", appearanceCodePlain}},
	{{"  ", appearanceCodePlain}, {"return", appearanceCodeKeyword}, {" true", appearanceCodePlain}},
	{{"}", appearanceCodePlain}},
}

// appearanceCodeThemeSection 是代码主题段（S-appearance-43–49）：下拉 + 预览。
//
// 代码主题真的生效：它决定 markdown 里行内代码与代码块的文字色、底色与边线
// （见 markdown.go 的 codeThemePalette），也落盘、重启恢复。默认那一套直接用主题
// token，另外三套是编辑器公开的配色值。
func appearanceCodeThemeSection(c *gui.Context, a *App) {
	t := c.Theme()
	options := make([]appearanceFontChoice, 0, len(appearanceCodeThemes))
	for _, name := range appearanceCodeThemes {
		group := "浅色 + 深色"
		if appearanceDarkOnlyThemes[name] {
			group = "深色"
		}
		options = append(options, appearanceFontChoice{label: name, value: name,
			usable: true, group: group})
	}
	index := appearanceChoiceIndex(options, a.appearance.CodeTheme)
	open := appearanceScreen.openChoice == "代码主题"
	gui.Column(c).Shrink(0).Gap(screenSpace25).Children(func() {
		screenLabel(c, "代码主题")
		gui.Column(c).Shrink(0).Gap(screenSpace1).Children(func() {
			if appearanceChoiceTrigger(c, t, "代码主题", options[index].label, open,
				appearanceCodeThemeW) {
				appearanceChoiceToggle("代码主题", false)
			}
			if open {
				appearanceCodeThemePanel(c, t, options, index, func(name string) {
					a.appearance.CodeTheme = name
					a.persistPreferences()
				})
			}
		})
		appearanceCodePreview(c, a.appearance.CodeTheme)
	})
	screenRule(c)
}

// appearanceCodeThemeW 是代码主题下拉的宽（与预览同宽）。
const appearanceCodeThemeW = 280

// appearanceCodeInk 是一套代码配色在一屏上的取值：底、正文、关键字、字符串、
// 函数名与边线。浅色 / 深色两种形态各一份；只有深色的主题两份相同，所以在浅色
// 界面里选它，预览与代码区也是深色（S-appearance-48）。
type appearanceCodeInk struct {
	bg, fg, border, keyword, str, fn gui.Color
}

// appearanceCodeInks 是六套公开配色的两种形态（前四套成对，后两套只有深色）。
var appearanceCodeInks = map[string][2]appearanceCodeInk{
	"GitHub": {
		{colorBG.light, colorText.light, gui.Hex("#d0d7de"), gui.Hex("#cf222e"),
			gui.Hex("#0a3069"), gui.Hex("#8250df")},
		{colorBG.dark, colorText.dark, gui.Hex("#30363d"), gui.Hex("#ff7b72"),
			gui.Hex("#a5d6ff"), gui.Hex("#d2a8ff")},
	},
	"One Dark Pro": {
		{gui.Hex("#282c34"), gui.Hex("#abb2bf"), gui.Hex("#3a404b"), gui.Hex("#c678dd"),
			gui.Hex("#98c379"), gui.Hex("#61afef")},
		{gui.Hex("#282c34"), gui.Hex("#abb2bf"), gui.Hex("#3a404b"), gui.Hex("#c678dd"),
			gui.Hex("#98c379"), gui.Hex("#61afef")},
	},
	"Solarized Light": {
		{gui.Hex("#fdf6e3"), gui.Hex("#657b83"), gui.Hex("#e3dcc6"), gui.Hex("#859900"),
			gui.Hex("#2aa198"), gui.Hex("#268bd2")},
		{gui.Hex("#002b36"), gui.Hex("#839496"), gui.Hex("#0b3b45"), gui.Hex("#859900"),
			gui.Hex("#2aa198"), gui.Hex("#268bd2")},
	},
	"Vitesse Dark": {
		{gui.Hex("#121212"), gui.Hex("#dbd7ca"), gui.Hex("#2a2a2a"), gui.Hex("#cb7676"),
			gui.Hex("#c98a7d"), gui.Hex("#4d9375")},
		{gui.Hex("#121212"), gui.Hex("#dbd7ca"), gui.Hex("#2a2a2a"), gui.Hex("#cb7676"),
			gui.Hex("#c98a7d"), gui.Hex("#4d9375")},
	},
	"Monokai": {
		{gui.Hex("#272822"), gui.Hex("#f8f8f2"), gui.Hex("#3d3e37"), gui.Hex("#f92672"),
			gui.Hex("#e6db74"), gui.Hex("#a6e22e")},
		{gui.Hex("#272822"), gui.Hex("#f8f8f2"), gui.Hex("#3d3e37"), gui.Hex("#f92672"),
			gui.Hex("#e6db74"), gui.Hex("#a6e22e")},
	},
	"Night Owl": {
		{gui.Hex("#011627"), gui.Hex("#d6deeb"), gui.Hex("#1d3b53"), gui.Hex("#c792ea"),
			gui.Hex("#ecc48d"), gui.Hex("#82aaff")},
		{gui.Hex("#011627"), gui.Hex("#d6deeb"), gui.Hex("#1d3b53"), gui.Hex("#c792ea"),
			gui.Hex("#ecc48d"), gui.Hex("#82aaff")},
	},
}

// appearanceCodeInkFor 取一套配色的形态：只有深色的主题两种形态相同，其余按界面
// 深浅取浅色 / 深色那一份。名字认不出来时 ok 为假（调用方用主题 token）。
func appearanceCodeInkFor(name string, dark bool) (appearanceCodeInk, bool) {
	set, ok := appearanceCodeInks[name]
	if !ok {
		return appearanceCodeInk{}, false
	}
	if dark || appearanceDarkOnlyThemes[name] {
		return set[1], true
	}
	return set[0], true
}

// appearanceCodePreview 是代码主题预览：左浅右深两块，各五行代码。配色跟着选中的
// 主题换（S-appearance-46 / 48）。
func appearanceCodePreview(c *gui.Context, theme string) {
	t := c.Theme()
	box := gui.Row(c).Shrink(0).Clip().Radius(screenRadiusCard)
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCard, screenStroke)
	})
	box.Children(func() {
		for _, dark := range []bool{false, true} {
			ink, ok := appearanceCodeInkFor(theme, dark)
			if !ok {
				ink = appearanceCodeInk{bg: colorBG.light, fg: colorText.light}
				if dark {
					ink = appearanceCodeInk{bg: colorBG.dark, fg: colorText.dark}
				}
			}
			appearanceCodePanel(c, ink)
		}
	})
}

// appearanceCodePanel 是预览的一半：行号 + 着色代码。
func appearanceCodePanel(c *gui.Context, ink appearanceCodeInk) {
	t := c.Theme()
	gui.Column(c).Grow(1).Background(ink.bg).Padding(screenSpace3, screenSpace3).
		Gap(screenSpace05).Children(func() {
		for i, line := range appearanceCodeSample {
			line := line
			gui.Row(c).Height(screenCodeLineH).AlignItems(gui.Center).
				Gap(screenSpace3).Children(func() {
				gui.Text(c, strconv.Itoa(i+1)).
					Font(screenCodeStyle.family).FontSize(screenCodeStyle.sizeIn(t)).
					TextColor(ink.fg.Alpha(0.45)).Width(screenCodeNumW).
					TextAlign(gui.End).SingleLine()
				spans := make([]gui.Span, 0, len(line))
				for _, token := range line {
					spans = append(spans, gui.Span{
						Text:   token.text,
						Font:   screenCodeStyle.family,
						Size:   screenCodeStyle.sizeIn(t),
						Weight: 400,
						Color:  appearanceCodeColor(ink, token.kind),
					})
				}
				gui.RichText(c, spans...).SingleLine()
			})
		}
	})
}

// appearanceCodeColor 是预览的着色：关键字、字符串、函数名三种 + 正文。
func appearanceCodeColor(ink appearanceCodeInk, kind int) gui.Color {
	switch kind {
	case appearanceCodeKeyword:
		return ink.keyword
	case appearanceCodeString:
		return ink.str
	case appearanceCodeFunc:
		return ink.fn
	}
	return ink.fg
}

// ---- 我的皮肤 ----

// appearanceSkinSection 是 4.3 我的皮肤（S-appearance-50–74）。
func appearanceSkinSection(c *gui.Context, live bool) {
	gui.Column(c).Shrink(0).Gap(screenSpace25).Children(func() {
		screenLabel(c, "我的皮肤")
		if live {
			typeMicro.style(c.Theme(), gui.Text(c, noteSkinMissing)).
				TextColor(c.Theme().TextMuted).Selectable()
		}
		gui.Row(c).Wrap().MaxWidth(screenGridW).
			GapX(screenCardGap).GapY(screenCardGap).
			Label("我的皮肤").Children(func() {
			for _, name := range appearanceSkins {
				if appearanceScreen.deleted[name] {
					continue
				}
				appearanceSkinCard(c, name, live)
			}
			appearanceDefaultSkinCard(c, live)
		})
		screenHint(c, "暂无自定义皮肤。可用图片新建或导入已有皮肤。")
		gui.Row(c).Gap(screenSpace2).Children(func() {
			// 皮肤渲染、图片选择与打开目录都没有通道：形态照留，连着 Source 时置灰。
			screenActionDisabled(c, iconPlus, "plus", "用图片新建", screenControlH, live,
				noteSkinMissing)
			screenActionDisabled(c, screenIconFolderOpen, "folder-open", "打开目录",
				screenControlH, live, noteSkinMissing)
		})
	})
}

// appearanceSkinCard 是一张皮肤卡（240×98）：缩略图在上、名称在下、角标在右上。
func appearanceSkinCard(c *gui.Context, name string, live bool) {
	t := c.Theme()
	selected := appearanceScreen.skin == name
	frame := t.Border
	if selected {
		frame = t.Accent
	}
	card := gui.ButtonBase(c).Shrink(0).Width(screenSkinCardW).Height(screenSkinCardH).
		Radius(screenRadiusCard).Background(t.Surface).Padding(screenStroke).
		AlignItems(gui.Stretch).Label("皮肤：" + name).Tooltip(name).
		Role(gui.RoleRadio).AccessSelected(selected).Disabled(live).Opacity(skinAlpha(live))
	if !selected {
		switch {
		case card.Pressed():
			frame = t.Border.Mix(t.Text, 0.45)
		case card.Hovered():
			frame = t.Border.Mix(t.Text, 0.3)
		}
	}
	card.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, frame, screenRadiusCard, screenStroke)
	})
	if card.Clicked() && !live {
		appearanceScreen.skin = name
	}
	card.Children(func() {
		gui.Box(c).Grow(1).Padding(screenSpace15, screenSpace15).Children(func() {
			gui.Column(c).Gap(screenSpace1).Children(func() {
				appearanceSkinThumb(c, name)
				gui.Row(c).AlignItems(gui.Center).Children(func() {
					gui.Spacer(c)
					screenStrongStyle.style(t, gui.Text(c, name)).TextColor(t.Text).SingleLine()
					gui.Spacer(c)
					if card.Hovered() && !selected {
						if screenIconButton(c, screenIconTrash, "删除皮肤："+name, "删除皮肤", t.TextMuted) {
							appearanceScreen.deleteFor = name
						}
					}
				})
			})
			if selected {
				screenCornerDot(c)
			}
		})
	})
}

// skinAlpha 是皮肤卡的透明度：连着 Source 时置灰（没有皮肤渲染器，选了不生效）。
func skinAlpha(live bool) float32 {
	if live {
		return noteDisabledAlpha
	}
	return 1
}

// appearanceSkinThumb 是按皮肤配色画的窗口示意（没有缩略图资产）。
func appearanceSkinThumb(c *gui.Context, name string) {
	t := c.Theme()
	title, body, ink := appearanceSkinColors(name, t)
	box := gui.Box(c).Shrink(0).Height(screenSkinThumbH).Clip()
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, body, screenSpace1)
		p.Fill(gui.Rect{X: r.X, Y: r.Y, W: r.W, H: screenSkinTitleH}, title, screenSpace1)
		for i := 0; i < 3; i++ {
			dot := gui.Rect{
				X: r.X + screenSpace15 + float32(i)*screenSpace2,
				Y: r.Y + screenSpace15,
				W: screenSpace05, H: screenSpace05,
			}
			p.Fill(dot, ink, screenSpace05/2)
		}
		side := gui.Rect{
			X: r.X + screenSpace1, Y: r.Y + screenSkinTitleH + screenSpace1,
			W: r.W * 0.3, H: r.H - screenSkinTitleH - 2*screenSpace1,
		}
		p.Fill(side, t.Border.Alpha(0.35), screenSpace05)
		for i := 0; i < 3; i++ {
			line := gui.Rect{
				X: r.X + r.W*0.4, Y: r.Y + screenSkinTitleH + screenSpace15 + float32(i)*screenSpace2,
				W: r.W * 0.5, H: screenSpace05,
			}
			p.Fill(line, t.TextMuted.Alpha(0.35), screenSpace05/2)
		}
	})
}

// appearanceSkinColors 返回 (标题栏, 窗体底, 标题栏标记) 三色。
func appearanceSkinColors(name string, t *gui.Theme) (title, body, ink gui.Color) {
	switch name {
	case "Windows 98":
		return gui.Hex("#000080"), gui.Hex("#c0c0c0"), gui.Hex("#ffffff")
	case "Windows XP":
		return gui.Hex("#0a55c4"), gui.Hex("#d4d0c8"), gui.Hex("#ffffff")
	case "Windows Vista":
		return gui.Hex("#6f9fd8"), gui.Hex("#e6eef7"), gui.Hex("#ffffff")
	case "macOS Big Sur":
		return gui.Hex("#e8e8e8"), gui.Hex("#f7f7f7"), gui.Hex("#ff5f57")
	case "OS X Tiger":
		return gui.Hex("#d8d8d8"), gui.Hex("#efefef"), gui.Hex("#ff5f57")
	}
	return t.Surface, t.Background, t.TextMuted
}

// appearanceDefaultSkinCard 是网格末尾的内置外观卡「默认」：X 图标 + 文字。
func appearanceDefaultSkinCard(c *gui.Context, live bool) {
	t := c.Theme()
	selected := appearanceScreen.skin == ""
	frame := t.Border
	if selected {
		frame = t.Accent
	}
	card := gui.ButtonBase(c).Shrink(0).Width(screenSkinCardW).Height(screenSkinCardH).
		Radius(screenRadiusCard).Background(t.Surface).Padding(screenStroke).
		AlignItems(gui.Stretch).Label("皮肤：默认").Tooltip("默认").
		Role(gui.RoleRadio).AccessSelected(selected).Disabled(live).Opacity(skinAlpha(live))
	if !selected {
		switch {
		case card.Pressed():
			frame = t.Border.Mix(t.Text, 0.45)
		case card.Hovered():
			frame = t.Border.Mix(t.Text, 0.3)
		}
	}
	card.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, frame, screenRadiusCard, screenStroke)
	})
	if card.Clicked() && !live {
		appearanceScreen.skin = ""
	}
	card.Children(func() {
		gui.Box(c).Grow(1).Padding(screenSpace15, screenSpace15).Children(func() {
			gui.Column(c).Grow(1).Justify(gui.Center).AlignItems(gui.Center).
				Gap(screenSpace15).Children(func() {
				mark := gui.Box(c).Size(screenSegHLg, screenSegHLg).Shrink(0).
					Radius(screenRadiusCard).Background(t.Surface)
				mark.Children(func() {
					gui.Box(c).Grow(1).Center().Children(func() {
						screenIcon(c, iconXMark, screenIconBlock, t.TextMuted, "x-mark")
					})
				})
				screenValueStyle.style(t, gui.Text(c, "默认")).TextColor(t.Text).SingleLine()
			})
			if selected {
				screenCornerDot(c)
			}
		})
	})
}

// ---- 当前皮肤设置（条件：有激活皮肤，S-appearance-76–93） ----

// appearanceActiveSkin 是「当前皮肤设置」段：窗口背景、透明度、大小、位置、
// 附着方式、文字颜色与高级折叠。
func appearanceActiveSkin(c *gui.Context) {
	name := appearanceScreen.skin
	if name == "" {
		return
	}
	t := c.Theme()
	screenRule(c)
	gui.Column(c).Shrink(0).Gap(screenSpace25).Children(func() {
		// S-appearance-76 当前皮肤设置
		screenLabel(c, "当前皮肤设置")
		// S-appearance-77 窗口背景
		appearanceSkinBackgroundRow(c)
		// S-appearance-82 透明度（滑块）
		appearanceSkinOpacityRow(c)
		// S-appearance-83 大小（Select）
		appearanceSkinSelectRow(c, "大小", &appearanceScreen.skinFit, appearanceFits)
		// S-appearance-84 位置（Select）
		appearanceSkinSelectRow(c, "位置", &appearanceScreen.skinPlace, appearancePlacements)
		// S-appearance-85 附着方式（Select）
		appearanceSkinSelectRow(c, "附着方式", &appearanceScreen.skinAttach, appearanceAttach)
		if appearanceScreen.skinAttach == 1 {
			gui.Row(c).Gap(screenSpace15).AlignItems(gui.Center).Children(func() {
				screenIcon(c, iconWarn, screenIconMicro, t.Warning, "exclamation")
				// S-appearance-86 ⚠ 可能影响滚动性能
				screenHint(c, "可能影响滚动性能")
			})
		}
		gui.Column(c).Shrink(0).Gap(screenSpace05).Children(func() {
			// S-appearance-87 文字颜色 · 根据背景调整文字颜色，保证可读性。
			screenLabel(c, "文字颜色")
			screenHint(c, "根据背景调整文字颜色，保证可读性。")
		})
		gui.Row(c).Children(func() {
			// S-appearance-88 文字颜色（Select）
			screenIndexSelect(c, &appearanceScreen.skinInk, appearanceTextInks, "文字颜色")
		})
		appearanceSkinAdvanced(c)
	})
}

// appearanceSkinBackgroundRow 是窗口背景一行：路径 + 更换 + 移除。
func appearanceSkinBackgroundRow(c *gui.Context) {
	t := c.Theme()
	value := "未设置图片"
	if appearanceScreen.skinImage {
		value = "~/Pictures/background.png"
	}
	gui.Column(c).Shrink(0).Gap(screenSpace15).Children(func() {
		// S-appearance-77 窗口背景（标签）
		screenLabel(c, "窗口背景")
		gui.Row(c).AlignItems(gui.Center).Gap(screenSpace2).Children(func() {
			// S-appearance-78 背景图路径（未设置图片）
			screenCodeStyle.style(t, gui.Text(c, value)).TextColor(t.TextMuted).
				Grow(1).MinWidth(0).SingleLine().Label("背景图路径")
			// S-appearance-79 更换
			if screenButton(c, "更换", "更换背景图", "", false, rowRegular) {
				appearanceScreen.skinImage = true
			}
			if appearanceScreen.skinImage {
				// S-appearance-80 移除背景图
				if screenIconButton(c, iconXMark, "移除背景图", "移除背景图", t.TextMuted) {
					appearanceScreen.skinImage = false
				}
			}
		})
	})
}

// appearanceSkinOpacityRow 是透明度一行：标签 + 值 + 滑块（0–100，步长 5）。
func appearanceSkinOpacityRow(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(screenSpace15).Children(func() {
		gui.Row(c).AlignItems(gui.Center).Children(func() {
			screenLabel(c, "透明度")
			gui.Spacer(c)
			// S-appearance-81 透明度当前值
			screenValueStyle.style(t, gui.Text(c, strconv.Itoa(int(appearanceScreen.skinOpacity+0.5))+"%")).
				TextColor(t.Text).Label("透明度当前值")
		})
		// S-appearance-82 透明度滑块
		gui.Slider(c, &appearanceScreen.skinOpacity, 0, 100).Step(5).
			Label("透明度").Grow(1)
	})
}

// appearanceSkinSelectRow 是一行下拉：标签在上、选择在下。
func appearanceSkinSelectRow(c *gui.Context, label string, value *int, options []string) {
	gui.Column(c).Shrink(0).Gap(screenSpace15).Children(func() {
		screenLabel(c, label)
		gui.Row(c).Children(func() {
			screenIndexSelect(c, value, options, label)
		})
	})
}

// appearanceSkinAdvanced 是高级折叠：展开后给外部打开、打开 skin.json、重新加载。
func appearanceSkinAdvanced(c *gui.Context) {
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Height(screenControlH).Padding(controlPadV, controlPadH).
		// S-appearance-89 高级
		Radius(screenRadiusCtl).Justify(gui.Start).Label("高级")
	if b.Hovered() {
		b = b.Background(t.SurfaceHover)
	}
	if b.Clicked() {
		appearanceScreen.advanced = !appearanceScreen.advanced
	}
	b.Children(func() {
		gui.Row(c).Gap(screenSpace15).AlignItems(gui.Center).Children(func() {
			screenValueStyle.style(t, gui.Text(c, "高级")).TextColor(t.Text).SingleLine()
			rotate := float32(0)
			if appearanceScreen.advanced {
				rotate = 90
			}
			screenIconRotated(c, iconChevronR, screenIconMicro, t.TextMuted, "chevron", rotate)
		})
	})
	if !appearanceScreen.advanced {
		return
	}
	gui.Row(c).Gap(screenSpace2).Children(func() {
		// S-appearance-90 外部打开
		screenButton(c, "外部打开", "外部打开", "", false, rowRegular)
		// S-appearance-92 打开 skin.json
		screenButton(c, "打开 skin.json", "打开 skin.json", "", false, rowRegular)
		// S-appearance-93 重新加载
		if screenButton(c, "重新加载", "重新加载皮肤", "", false, rowRegular) {
			appearanceScreen.reloaded = true
		}
	})
	if appearanceScreen.reloaded {
		// S-appearance-94 皮肤已重新加载
		screenHint(c, "皮肤已重新加载")
	}
}

// ---- 删除对话框（S-appearance-60–65） ----

// appearanceDeleteDialog 是「删除皮肤？」确认框。
func appearanceDeleteDialog(c *gui.Context) {
	if appearanceScreen.deleteFor == "" {
		return
	}
	t := c.Theme()
	open := true
	name := appearanceScreen.deleteFor
	m := modalShell(gui.Modal(c, &open, func() {
		screenDialogPanel(c, "删除皮肤？", func() {
			typeBody.style(t, gui.Text(c, fmt.Sprintf("“%s”及其全部资源将被永久删除。", name))).
				TextColor(t.TextMuted).Selectable()
			gui.Row(c).Justify(gui.End).Gap(screenSpace2).Children(func() {
				if screenButton(c, "取消", "取消删除", "", false, rowRegular) {
					appearanceScreen.deleteFor = ""
					open = false
				}
				label := "删除"
				if appearanceScreen.deleting {
					label = "正在删除…"
				}
				if screenButton(c, label, label, "", true, rowRegular) {
					if appearanceScreen.deleted == nil {
						appearanceScreen.deleted = map[string]bool{}
					}
					appearanceScreen.deleted[name] = true
					if appearanceScreen.skin == name {
						appearanceScreen.skin = ""
					}
					appearanceScreen.deleteFor = ""
					appearanceScreen.deleting = false
					open = false
				}
			})
		})
	}))

	if !open || m.Dismissed() {
		appearanceScreen.deleteFor = ""
		appearanceScreen.deleting = false
	}
}
