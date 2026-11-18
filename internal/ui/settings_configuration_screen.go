package ui

import (
	"fmt"
	"math"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 模型供应」的重建屏：元素按
// docs/domains/requirements/inventory-settings-rt.md §7（S-configuration-01 … 83，
// 83 行；可见 20 = 主体 16 + 容器 1 + 图标 3，另有数据驱动实例 162 =
// 2 个分组头 + 128 个供应商项 + 32 行模型）逐行对应，文案逐字取参考产品的
// 中文（i18n 的 configurationSettings.*，见文件内注释），数值取
// docs/domains/requirements/design-baseline.md：
//
//	控件高 32（输入框）/ 25（行内搜索、分组头）/ 28（列表行、按钮）、
//	开关 28×16、复选框 14×14、说明字 10.5、列表名 12.25、栏标题 11、
//	卡片圆角 6（rounded-md）、描边 1（--border）、间距 3.5 的倍数。
//
// 版式（运行时实测，视口 1030×725）：左列 w-[220px]（bg-muted/30 + 右侧 1px
// 分隔线），列表行 x 234–443 宽 209；右栏 p-5（17.5），内容 max-w-2xl；
// 明细头部 pb-3 + 底线，账户托管提示条 466,83 542×53，基础 URL 值 466,172
// 542×32，「模型 · 32 / 32」在 466,221，模型行 33 一档，底部动作 840/915,675。
//
// 只保留可丢的界面状态（configScreen）：搜索词、选中项、两个分组的折叠、启用
// 开关、模型的开关。写事实（启用供应商、开关模型、刷新模型列表、添加自定义
// 提供商、账号托管的跳转）都需要模型供应领域与 Intents，本屏只改界面状态，
// 接线时换成 Intents。
//
// 未重建的分支：详情面板的另外六种形态（添加自定义提供商 / OpenAI Codex OAuth /
// Grok Build 设备码 / 通用托管 OAuth / 未配置表单 / 已配置凭据）与「添加模型」
// 对话框，本屏用与设置壳同一句口径（「尚未按参考产品重建」）占位，不摆看起来
// 能用的控件（S-configuration-10 … 80 的未出现分支）。
//
// 两处没有资产，用等尺寸示意：供应商品牌图标（参考的 ProviderIcon）用首字母块
// 代替（与账户屏头像同一方案）；能力徽标只画运行时出现的「视觉」「推理」两种。

// 本屏几何：参考产品的 Tailwind 量 × 0.25rem（根字号 14 时 0.25rem = 3.5px =
// spaceUnit），因此一律写成 spaceUnit 的倍数或既有档位常量。
const (
	// 左列 w-[220px]；列表行 209 = 220 − 两侧 px-1.5。
	configListW = 220

	// 左列内边距：pt-3 / pb-1（标题）、pt-1 / pb-2（搜索）、pt-2 / pb-3（添加）、
	// px-1.5（列表与搜索）、px-3（标题）。
	configPadH     = spaceUnit * 1.5 // 5.25
	configTitlePad = space3          // 10.5
	configTitleGap = space1          // 3.5
	configAddPad   = space2          // 7
	configAddGap   = space3          // 10.5

	// 搜索框：px-2 py-1，高 25（design-baseline §3 行内输入框档）。
	configFieldPadH = space2
	configFieldPadV = space1
	configFieldH    = screenSegHSm // 25

	// 列表行：分组头 h-7（25）、供应商行 28（py-1.5 + 17.5 行盒）、添加行 28。
	configGroupH  = screenSegHSm
	configRowH    = rowRegular
	configRowPadH = space2
	configRowPadV = spaceUnit * 1.5

	// 行内间距：gap-1（3.5）/ gap-1.5（5.25）/ gap-2（7）/ gap-3（10.5）；
	// 组间 space-y-1（3.5）、组内 space-y-0.5（1.75）。
	configGapIcon   = spaceUnit * 1.5
	configGapRow    = space2
	configGapBlock  = space3
	configGapGroups = space1
	configGapItems  = spaceUnit * 0.5

	// 图标与标记：16 / 14 / 12（design-baseline §3），状态点 7，复选框 14×14。
	configIcon     = screenIconMicro
	configIconBig  = screenIconInline
	configIconTiny = 12
	configDot      = spaceUnit * 2
	configCheckbox = spaceUnit * 4
	// 开关 28×16（design-baseline §3）。
	configSwitchW = screenSwitchW
	configSwitchH = screenSwitchH

	// 右栏：p-5（17.5）、块间 gap-5（17.5）、头部 pb-3（10.5）、
	// 提示条 px-3 py-2.5、URL 值 px-3 py-2（高 32）、
	// 模型行 px-3 py-2（33 一档）、内容上限 max-w-2xl（672）。
	configDetailPad  = spaceUnit * 5
	configDetailGap  = spaceUnit * 5
	configHeadPad    = space3
	configBannerPadH = space3
	configBannerPadV = spaceUnit * 2.5
	configCodePadH   = space3
	configCodePadV   = space2
	configCodeH      = spaceUnit * 9
	configModelPadV  = space2
	configModelPadH  = space3
	configDetailMaxW = 672
	configDetailMinW = 185
	configHalfRule   = 1

	// 徽标：rounded bg-primary/10 px-1.5 py-0.5（design-baseline §3 小徽标 10/14）。
	configBadgePadH   = spaceUnit * 1.5
	configBadgePadV   = spaceUnit * 0.5
	configBadgeRadius = radiusSmall

	// 半透明语义色：参考的 primary/30（提示条描边）、primary/5（提示条底）、
	// primary/10（徽标底）。
	configBannerStroke = 0.3
	configBannerFill   = 0.05
	configBadgeFill    = 0.1

	// 大写小字标题的 tracking-wider（0.05em × 11px）。
	configTracking = 0.55

	// 空态的内边距：模型列表没有匹配项时，提示语上下留白（py-6）。
	configEmptyPad = space6
)

// 字阶（design-baseline §2）：栏标题 11/中粗、计数 10、列表名 12.25、
// 说明 10.5、徽标 10、代码 12.25（等宽）。
var (
	configTitleStyle  = fontStep{size: 11, line: 14, weight: 500}
	configCountStyle  = fontStep{size: 10, line: 14, weight: 400}
	configRowStyle    = fontStep{size: 12.25, line: 17.5, weight: 400}
	configStrongStyle = fontStep{size: 12.25, line: 17.5, weight: 500}
	configMicroStyle  = fontStep{size: 10.5, line: 14, weight: 400}
	configBadgeStyle  = fontStep{size: 10, line: 14, weight: 400}
	configCodeStyle   = fontStep{size: 12.25, line: 17.5, weight: 400, family: codeFont}
	configMarkStyle   = fontStep{size: 9, line: 14, weight: 500}
)

// 本屏图标：Heroicons v2 outline（24 网格、1.5 描边、MIT）的内联路径，与
// icons.go 同一套约定；chevron 与 refresh 复用外观屏的同名图形。
var (
	configIconLock     = mustSVG(`<path d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z"/>`)
	configIconExternal = mustSVG(`<path d="M13.5 6H5.25A2.25 2.25 0 0 0 3 8.25v10.5A2.25 2.25 0 0 0 5.25 21h10.5A2.25 2.25 0 0 0 18 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25"/>`)
	configIconMinus    = mustSVG(`<path d="M5.25 12h13.5"/>`)
)

// configModel 是一行模型：名称、两项能力与开关位置。取值来自运行时快照
// （选中 DimAgent OAuth 的 32 个模型），不是事实；接线时换成 Source 的数据。
type configModel struct {
	name      string
	vision    bool
	reasoning bool
	on        bool
}

// configProvider 是一个供应商：名称、是否已配置、是否由账户托管、基础 URL。
// account 为真的只有 DimAgent OAuth（运行时唯一已启用且账户托管的一项）。
type configProvider struct {
	name       string
	configured bool
	account    bool
	baseURL    string
}

// configDimName 是当前选中的账户托管供应商名。
const configDimName = "DimAgent OAuth"

// configDimBaseURL 是它只读展示的基础 URL（运行时实测值）。
const configDimBaseURL = "https://dimagent.cn/v1"

// configDisabledProviderNames 是「未启用」分组的 127 个供应商名，顺序与运行时
// DOM 一致（模型供应.json 的列表趟）。
var configDisabledProviderNames = []string{
	"302.AI", "Adobe Firefly", "Aihubmix", "Alibaba DashScope", "Amazon Speech",
	"Anthropic", "Apiframe", "Volcengine Agent Plan", "Ark Code", "AssemblyAI",
	"Atlas Cloud AI", "AtlasCloud Coding Plan", "AWS Bedrock", "Azure OpenAI",
	"Azure Speech", "Bailian", "Model Studio Coding Plan", "BaiLing", "Beatoven",
	"Black Forest Labs", "ByteDance", "BytePlus Coding Plan", "Cartesia",
	"Command Code", "Custom Provider", "D-ID", "Deepgram", "DeepSeek", "DMXAPI",
	"DouBaoSeed", "each::labs", "ElevenLabs", "fal.ai", "Fireworks", "Fish Audio",
	"FlyMux (Anthropic)", "FlyMux (OpenAI)", "GitHub Models", "Gladia", "Google",
	"Grok Build", "Groq", "HeyGen", "Hugging Face", "Hume AI", "IBM Watson Speech",
	"Ideogram", "JieKou.AI", "KAT-Coder", "Kimi For Coding", "Kling", "LM Studio",
	"Longcat", "Luma AI", "Magpie", "MiniMax (minimax.io)", "MiniMax (minimaxi.com)",
	"MiniMax Coding Plan (minimaxi.com)", "MiniMax Coding Plan (minimax.io)",
	"Mistral AI", "ModelScope", "Moonshot", "Moonshot AI", "Mubert", "MusicAPI.ai",
	"New API", "Nous Research", "Novita AI", "NVIDIA", "o3.fan", "Ollama", "OpenAI",
	"OpenAI Codex", "OpenCode Go", "OpenRouter", "PiAPI", "Pika", "PixVerse",
	"PlayHT", "PPIO", "Baidu Qianfan Coding Plan", "Baidu Qianfan Token Plan",
	"Qiniu", "QwenCloud", "QwenCloud For Coding", "QwenCloud Token Plan", "Recraft",
	"Replicate", "Rev AI", "Reve AI", "RunInfra", "Runware", "Runway", "Sarvam AI",
	"SiliconFlow", "SiliconFlow (com)", "Soniox", "Speechmatics", "Stability AI",
	"Step Plan", "StepFun en", "StepFun", "Suno", "Synthesia", "Tavus",
	"Tencent Hunyuan", "Tencent Token Plan",
	"Tencent Token Plan Enterprise Lite", "Tencent Token Plan Enterprise Lite (Intl)",
	"Tencent Token Plan Enterprise Pro", "Tencent Token Plan Enterprise Pro (Intl)",
	"Tencent Token Plan (Intl)", "Together AI", "TokenFlux", "Upstage",
	"Vercel AI Gateway", "Vidu", "xAI", "Xiaomi MiMo", "Xiaomi Token Plan (Europe)",
	"Xiaomi Token Plan (China)", "Xiaomi Token Plan (Singapore)", "Z.AI",
	"Z.AI Coding Plan", "ZenMux", "ZhipuAI", "ZhipuAI Coding Plan",
}

// configModelTable 是 DimAgent OAuth 的 32 个模型（名称与能力徽标逐行取自
// 运行时 DOM）。开关位置取自截图：首尾两趟可见，中段 14–21 行落在两趟采集之间
// 的空档里，按未启用渲染（见交付说明）。
var configModelTable = []configModel{
	{name: "vokie-dp-v4-flash"},
	{name: "seedream-5.0"},
	{name: "Seedance 2.0", vision: true},
	{name: "Seedance 2.0 Fast", vision: true},
	{name: "seed-2.1-pro", vision: true, reasoning: true, on: true},
	{name: "Seed 2.1 Turbo", vision: true, reasoning: true},
	{name: "GPT-5.6 Terra", vision: true, reasoning: true, on: true},
	{name: "arcship-image", vision: true},
	{name: "DeepSeek V4 Pro", reasoning: true, on: true},
	{name: "Seedance 2.5"},
	{name: "Qwen: Qwen3.8 27B", vision: true, reasoning: true, on: true},
	{name: "MiniMax-M3", vision: true, reasoning: true},
	{name: "GLM-5.3 Flash", vision: true, reasoning: true, on: true},
	{name: "GPT-6 Astra", vision: true, reasoning: true},
	{name: "OpenAI: GPT-Image-2.5-Flare", vision: true},
	{name: "OpenAI: GPT-Image-2.5-Sunburst", vision: true},
	{name: "DeepSeek: DeepSeek V4.1 Flash", vision: true, reasoning: true},
	{name: "GPT-6 Sol", vision: true, reasoning: true},
	{name: "GPT-6 Luna", vision: true, reasoning: true},
	{name: "seedream-flash", vision: true, reasoning: true},
	{name: "Xiaomi: MiMo-V2.6-Flash", vision: true, reasoning: true},
	{name: "Xiaomi: MiMo-V2.6-Pro", vision: true, reasoning: true, on: true},
	{name: "qwen-image-3.0", vision: true},
	{name: "qwen-image-3.0-pro", vision: true},
	{name: "Qwen3.8 Max", vision: true, reasoning: true, on: true},
	{name: "Qwen3.8 Flash", vision: true, reasoning: true, on: true},
	{name: "GPT-6.1 Sol", vision: true, reasoning: true, on: true},
	{name: "Claude Opus 5.5", vision: true, reasoning: true, on: true},
	{name: "Claude Sonnet 5.5", vision: true, reasoning: true, on: true},
	{name: "GLM-5.3", reasoning: true, on: true},
	{name: "seedream-5.5"},
	{name: "minimax-h3-image"},
}

// configState 是本屏可丢的界面状态：搜索词、选中项、折叠、开关。不是事实。
type configState struct {
	providerFilter string
	modelFilter    string

	selected          string
	enabledCollapsed  bool
	disabledCollapsed bool

	// providerToggles 是按供应商名的启用开关（界面态）。初始只有账户托管的
	// DimAgent OAuth 为开（运行时唯一已启用的一项）。
	providerToggles map[string]*bool
	models          []configModel
}

var configScreen = configState{
	selected: configDimName,
	models:   append([]configModel(nil), configModelTable...),
}

// providerToggle 取一个供应商的启用开关（按需建、指针稳定）。
func (s *configState) providerToggle(name string) *bool {
	if s.providerToggles == nil {
		s.providerToggles = map[string]*bool{}
	}
	if on, ok := s.providerToggles[name]; ok {
		return on
	}
	on := name == configDimName
	s.providerToggles[name] = &on
	return &on
}

// renderConfiguration 是设置 › 模型供应这一屏（设置壳的一个分节）。
func renderConfiguration(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Grow(1).AlignItems(gui.Stretch).Background(t.Background).
		Label("模型供应").Children(func() {
		configProviderColumn(c)
		configDetailColumn(c)
	})
}

// ---- 左栏：供应商列表（S-configuration-01 … 09） ----

// configProviderColumn 是左栏：栏标题 + 搜索 + 两段分组 + 底部「添加」。
func configProviderColumn(c *gui.Context) {
	t := c.Theme()
	enabled, disabled := configFilteredProviders()
	column := gui.Column(c).Width(configListW).Shrink(0).
		Background(configListFill(t)).Label("供应商列表")
	// 右缘 1px 分隔线（参考的 border-r）：只有一条边，用填色画，不另立描边。
	column.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Fill(gui.Rect{X: r.X + r.W - configHalfRule, Y: r.Y, W: configHalfRule, H: r.H},
			t.Border, 0)
	})
	column.Children(func() {
		// S-configuration-01：栏标题（11、中粗、大写间距、次级前景）。
		gui.Row(c).Shrink(0).AlignItems(gui.Center).
			Padding(configTitlePad, configTitlePad, configTitleGap, configTitlePad).
			Children(func() {
				configTitleStyle.style(t, gui.Text(c, "模型供应")).TextColor(t.TextMuted).
					LetterSpacing(configTracking).SingleLine().
					Role(gui.RoleHeading).Level(3)
			})
		// S-configuration-02 / 03：搜索框（放大镜在左、输入框在右）。
		gui.Row(c).Shrink(0).Padding(0, configPadH, configAddPad, configPadH).Children(func() {
			configSearchField(c, &configScreen.providerFilter, "搜索供应商", "搜索供应商")
		})
		// S-configuration-05 … 08：两段分组，列表自己在列内滚动。
		gui.Scroll(c).Grow(1).Label("供应商").Children(func() {
			gui.Column(c).Shrink(0).Gap(configGapGroups).
				Padding(0, configPadH, configAddPad, configPadH).Children(func() {
				configProviderGroup(c, "已启用", enabled, &configScreen.enabledCollapsed)
				configProviderGroup(c, "未启用", disabled, &configScreen.disabledCollapsed)
			})
		})
		// S-configuration-09：添加（固定在这一列底部）。
		gui.Row(c).Shrink(0).Padding(configAddPad, configPadH, configAddGap, configPadH).
			Children(func() {
				configAddButton(c)
			})
	})
}

// configListFill 是左列底：参考的 bg-muted/30（内容底与悬停底之间的一档）。
func configListFill(t *gui.Theme) gui.Color {
	return t.SurfaceHover.Mix(t.Surface, 0.3)
}

// configSearchField 是一行搜索：1px 主题描边 + 放大镜 + 无内框的输入框。
func configSearchField(c *gui.Context, value *string, placeholder, label string) {
	t := c.Theme()
	field := gui.Row(c).Grow(1).Height(configFieldH).AlignItems(gui.Center).
		Gap(configGapIcon).Padding(configFieldPadV, configFieldPadH).
		Radius(screenRadiusCtl).Background(t.Background).Label(label)
	field.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	field.Children(func() {
		screenIcon(c, iconSearch, configIcon, t.TextMuted, "search")
		input := gui.TextInputBase(c, value).Grow(1).MinWidth(0).Label(label).
			Placeholder(placeholder)
		configRowStyle.style(t, input).TextColor(t.Text)
		// S-configuration-04 / 22：有搜索词时右端出现清除按钮。
		if *value != "" {
			if screenIconButton(c, iconXMark, "清除搜索", "清除搜索", t.TextMuted) {
				*value = ""
			}
		}
	})
}

// configFilteredProviders 按搜索词把供应商分成两段：名称子串、不区分大小写
// （参考 matchesProviderFilter 的口径：名称或标识命中即算）。
func configFilteredProviders() (enabled, disabled []configProvider) {
	for _, provider := range configProviderList() {
		if !configMatches(configScreen.providerFilter, provider.name) {
			continue
		}
		if provider.configured {
			enabled = append(enabled, provider)
		} else {
			disabled = append(disabled, provider)
		}
	}
	return enabled, disabled
}

// configMatches 是子串过滤：查询为空时全都算命中。
func configMatches(query, name string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), query)
}

// configProviderList 是供应商的全表：DimAgent OAuth 在前（已启用、账户托管），
// 其后是 127 个未启用项。
func configProviderList() []configProvider {
	out := make([]configProvider, 0, len(configDisabledProviderNames)+1)
	out = append(out, configProvider{
		name: configDimName, configured: true, account: true, baseURL: configDimBaseURL,
	})
	for _, name := range configDisabledProviderNames {
		out = append(out, configProvider{name: name})
	}
	return out
}

// configProviderGroup 是一段分组：折叠头 + （未折叠时）组内各项。
func configProviderGroup(c *gui.Context, label string, providers []configProvider, collapsed *bool) {
	// 搜索生效时两组强制展开（参考行为），折叠状态本身不改。
	expanded := !*collapsed || configScreen.providerFilter != ""
	configGroupHead(c, label, len(providers), expanded, collapsed)
	if !expanded {
		return
	}
	gui.Column(c).Shrink(0).Gap(configGapItems).Children(func() {
		for _, provider := range providers {
			configProviderRow(c, provider)
		}
	})
}

// configGroupHead 是一个分组折叠头：箭头 + 组名 + 计数（S-configuration-05/06）。
func configGroupHead(c *gui.Context, label string, count int, expanded bool, collapsed *bool) {
	t := c.Theme()
	head := gui.ButtonBase(c).Shrink(0).Height(configGroupH).AlignItems(gui.Center).
		Gap(configGapIcon).Padding(0, configRowPadH).Radius(screenRadiusCtl).
		Label(fmt.Sprintf("%s %d", label, count)).Cursor(gui.CursorPointer).
		Role(gui.RoleDisclosure).Expanded(expanded)
	switch {
	case head.Pressed():
		head = head.Background(t.SurfacePressed)
	case head.Hovered():
		head = head.Background(t.SurfaceHover)
	}
	if head.Clicked() {
		// 折叠只是界面状态；搜索生效时点它不改变两组的展开（与参考一致）。
		if configScreen.providerFilter == "" {
			*collapsed = !*collapsed
		}
	}
	degrees := float32(0)
	if expanded {
		degrees = 90
	}
	head.Children(func() {
		screenIconRotated(c, iconChevronR, configIconTiny, t.TextMuted, "chevron", degrees)
		configTitleStyle.style(t, gui.Text(c, label)).TextColor(t.TextMuted).
			LetterSpacing(configTracking).Grow(1).MinWidth(0).SingleLine()
		configCountStyle.style(t, gui.Text(c, fmt.Sprint(count))).
			TextColor(t.TextMuted).SingleLine()
	})
}

// configProviderRow 是一个供应商项：图标 + 名称 + 状态点（S-configuration-07/08）。
func configProviderRow(c *gui.Context, provider configProvider) {
	t := c.Theme()
	selected := configScreen.selected == provider.name
	row := gui.ButtonBase(c).Shrink(0).Height(configRowH).AlignItems(gui.Center).
		Gap(configGapRow).Padding(configRowPadV, configRowPadH).Radius(screenRadiusCtl).
		Label(configProviderLabel(provider)).Tooltip(provider.name).
		Cursor(gui.CursorPointer).AccessSelected(selected)
	switch {
	case selected:
		row = row.Background(t.SurfacePressed)
	case row.Hovered():
		row = row.Background(t.SurfaceHover)
	}
	if row.Clicked() {
		configScreen.selected = provider.name
	}
	style := configRowStyle
	if selected {
		style = configStrongStyle
	}
	row.Children(func() {
		configProviderMark(c, provider)
		style.style(t, gui.Text(c, provider.name)).TextColor(t.Text).Grow(1).MinWidth(0).
			SingleLine()
		configStatusDot(c, provider.configured)
	})
}

// configProviderLabel 是列表项的读屏名：名称 + 启用状态（R-CFG-054）。
func configProviderLabel(provider configProvider) string {
	if provider.configured {
		return provider.name + "，已启用"
	}
	return provider.name + "，未启用"
}

// configProviderMark 是供应商图标位：没有品牌资产，用首字母块示意（与账户屏
// 头像同一方案）；已启用的用主色底。
func configProviderMark(c *gui.Context, provider configProvider) {
	t := c.Theme()
	box := gui.Box(c).Size(configIcon, configIcon).Shrink(0).Center().
		Label("图标：" + provider.name).AccessHidden(true)
	fill, ink := t.SurfacePressed, t.TextMuted
	if provider.configured {
		fill, ink = t.Accent.Alpha(configBadgeFill), colorAccentInk.in(t)
	}
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, fill, configIcon/2)
	})
	box.Children(func() {
		configMarkStyle.style(t, gui.Text(c, providerInitial(provider.name))).
			TextColor(ink).SingleLine()
	})
}

// providerInitial 是首字母块里的字符：取名称的第一个字符并大写（中文名原样）。
func providerInitial(name string) string {
	for _, r := range name {
		return strings.ToUpper(string(r))
	}
	return "?"
}

// configStatusDot 是状态点：已配置为主色实心，未配置为描边空心（● / ○）。
func configStatusDot(c *gui.Context, on bool) {
	t := c.Theme()
	label := "未启用"
	if on {
		label = "已启用"
	}
	box := gui.Box(c).Size(configDot, configDot).Shrink(0).
		Label("状态：" + label).AccessHidden(true)
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		if on {
			p.Fill(r, t.Accent, configDot/2)
			return
		}
		p.Stroke(r, t.Border, configDot/2, screenStroke)
	})
}

// configAddButton 是左栏底部的「添加」（S-configuration-09）。
func configAddButton(c *gui.Context) {
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Height(configRowH).AlignItems(gui.Center).
		Gap(configGapRow).Padding(configRowPadV, configRowPadH).Radius(screenRadiusCtl).
		Label("添加").Tooltip("添加自定义提供商").Cursor(gui.CursorPointer)
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		screenIcon(c, iconPlus, configIcon, t.TextMuted, "plus")
		configRowStyle.style(t, gui.Text(c, "添加")).TextColor(t.TextMuted).SingleLine()
	})
	if b.Clicked() {
		// 「添加自定义提供商」面板（S-configuration-64 … 77）尚未重建：点击只留在
		// 视图里，不摆看似的表单。
	}
}

// ---- 右栏：详情（S-configuration-11 … 28） ----

// configDetailColumn 是右栏：头部 + 账户托管（或未重建分支）。
func configDetailColumn(c *gui.Context) {
	provider, ok := configSelectedProvider()
	gui.Column(c).Grow(1).MinWidth(0).Padding(configDetailPad).Children(func() {
		gui.Column(c).Width(configDetailContentWidth(c)).Grow(1).Gap(configDetailGap).
			Children(func() {
				if !ok {
					// S-configuration-82：没有选中项时的空态。
					configMicroStyle.style(c.Theme(), gui.Text(c, "选择一个提供商查看详情")).
						TextColor(c.Theme().TextMuted)
					return
				}
				configDetailHead(c, provider)
				if provider.account {
					configManagedBody(c, provider)
					return
				}
				configUnbuiltBody(c)
			})
	})
}

// configDetailContentWidth 是详情内容的宽：参考的 max-w-2xl（672）封顶，
// 窄窗时收进可用宽度（见 design-baseline §7 的设置内容最大宽 1008）。
func configDetailContentWidth(c *gui.Context) float32 {
	width, _ := c.Size()
	usable := float32(math.Min(float64(width-SettingsNavWidth), SettingsContentMaxWidth))
	avail := usable - 2*space8 - configListW - 2*configDetailPad
	if avail < configDetailMinW {
		return configDetailMinW
	}
	if avail > configDetailMaxW {
		return configDetailMaxW
	}
	return avail
}

// configSelectedProvider 取当前选中的供应商。
func configSelectedProvider() (configProvider, bool) {
	for _, provider := range configProviderList() {
		if provider.name == configScreen.selected {
			return provider, true
		}
	}
	return configProvider{}, false
}

// configDetailHead 是详情头部：图标 + 名称 + 启用状态 + 启用开关，下方一条底线
// （S-configuration-11 … 13）。
func configDetailHead(c *gui.Context, provider configProvider) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Justify(gui.SpaceBetween).
			Gap(configGapRow).Padding(0, 0, configHeadPad, 0).Children(func() {
			gui.Row(c).Grow(1).MinWidth(0).AlignItems(gui.Center).
				Gap(configGapRow).Children(func() {
				configProviderMarkBig(c, provider)
				gui.Column(c).Grow(1).MinWidth(0).Children(func() {
					configStrongStyle.style(t, gui.Text(c, provider.name)).TextColor(t.Text).
						SingleLine()
					configMicroStyle.style(t, gui.Text(c, configStatusLabel(provider))).
						TextColor(t.TextMuted).SingleLine()
				})
			})
			gui.Switch(c, configScreen.providerToggle(provider.name)).
				Size(configSwitchW, configSwitchH).Radius(configSwitchH / 2).
				Label("启用 " + provider.name)
		})
		screenRule(c)
	})
}

// configStatusLabel 是详情头部的状态词（参考 selectedProviderStatusLabel）。
func configStatusLabel(provider configProvider) string {
	if provider.configured {
		return "已启用"
	}
	return "未启用"
}

// configProviderMarkBig 是详情头部的供应商图标位（16）。
func configProviderMarkBig(c *gui.Context, provider configProvider) {
	t := c.Theme()
	box := gui.Box(c).Size(configIconBig, configIconBig).Shrink(0).Center().
		Label("图标：" + provider.name).AccessHidden(true)
	fill, ink := t.SurfacePressed, t.TextMuted
	if provider.configured {
		fill, ink = t.Accent.Alpha(configBadgeFill), colorAccentInk.in(t)
	}
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, fill, configIconBig/2)
	})
	box.Children(func() {
		configRowStyle.style(t, gui.Text(c, providerInitial(provider.name))).
			TextColor(ink).SingleLine()
	})
}

// configManagedBody 是账户托管供应商的只读详情（S-configuration-14 … 28）：
// 提示条 + 基础 URL + 模型块 + 底部动作。
func configManagedBody(c *gui.Context, provider configProvider) {
	gui.Column(c).Grow(1).MinHeight(0).Gap(configDetailGap).Children(func() {
		gui.Column(c).Shrink(0).Gap(configDetailGap).Children(func() {
			configManagedBanner(c)
			configBaseURL(c, provider.baseURL)
		})
		configModelsPanel(c)
		configDetailFooter(c)
	})
}

// configManagedBanner 是「由 Dim 账户管理」提示条（S-configuration-14/15）：
// 整条是一个入口（参考点它跳账户设置）。
func configManagedBanner(c *gui.Context) {
	t := c.Theme()
	ink := colorAccentInk.in(t)
	b := gui.ButtonBase(c).Shrink(0).AlignItems(gui.Start).Gap(configGapBlock).
		Padding(configBannerPadV, configBannerPadH).Radius(screenRadiusCtl).
		Background(t.Accent.Alpha(configBannerFill)).
		Label("由 Dim 账户管理").Tooltip("打开账户设置").
		Cursor(gui.CursorPointer)
	b.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Accent.Alpha(configBannerStroke), screenRadiusCtl, screenStroke)
	})
	switch {
	case b.Pressed():
		b = b.Background(t.Accent.Alpha(configBannerFill * 2))
	case b.Hovered():
		b = b.Background(t.Accent.Alpha(configBannerFill * 2))
	}
	if b.Clicked() {
		// 跳转账户设置要设置壳的路由（跨分节），接线时由 Intents 给。
	}
	b.Children(func() {
		// 图标 mt-0.5：与标题的首行基线上沿对齐。
		gui.Box(c).Shrink(0).Margin(configGapItems, 0).Children(func() {
			screenIcon(c, configIconLock, configIconBig, ink, "lock")
		})
		gui.Column(c).Grow(1).MinWidth(0).Gap(configGapItems).Children(func() {
			configStrongStyle.style(t, gui.Text(c, "由 Dim 账户管理")).TextColor(ink).
				SingleLine()
			configMicroStyle.style(t, gui.Text(c, "此提供商由你的 Dim 账户管理，无法在此编辑。")).
				TextColor(t.TextMuted).Selectable()
		})
	})
}

// configBaseURL 是只读的基础 URL（S-configuration-17/18）：标签 + 代码块。
func configBaseURL(c *gui.Context, url string) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(configGapIcon).Children(func() {
		configMicroStyle.style(t, gui.Text(c, "基础 URL")).TextColor(t.TextMuted).
			Label("基础 URL").SingleLine()
		gui.Row(c).Shrink(0).Height(configCodeH).AlignItems(gui.Center).
			Padding(configCodePadV, configCodePadH).Radius(screenRadiusCtl).
			Background(t.SurfaceHover).Label("基础 URL 取值").Children(func() {
			configCodeStyle.style(t, gui.Text(c, url)).TextColor(t.TextMuted).
				SingleLine().Selectable()
		})
	})
}

// configModelsPanel 是模型块（S-configuration-20 … 26）：全选复选框 + 计数、
// 搜索、列表。列表自己滚，占满剩下的高度。
func configModelsPanel(c *gui.Context) {
	gui.Column(c).Grow(1).MinHeight(0).Gap(configGapRow).Children(func() {
		configModelsToolbar(c)
		gui.Row(c).Shrink(0).Children(func() {
			configSearchField(c, &configScreen.modelFilter, "搜索模型名称或 ID",
				"搜索模型名称或 ID")
		})
		configModelList(c)
	})
}

// configModelsToolbar 是「模型 · 可见 / 总数」一行：三态复选框 + 标签
// （S-configuration-20）。
func configModelsToolbar(c *gui.Context) {
	t := c.Theme()
	visible := configVisibleModelIndexes()
	label := fmt.Sprintf("模型 · %d / %d", len(visible), len(configScreen.models))
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(configGapRow).Children(func() {
		configModelsCheckbox(c)
		configMicroStyle.style(t, gui.Text(c, label)).TextColor(t.TextMuted).
			Label(label).SingleLine()
	})
}

// configModelsCheckbox 是全选框：全部启用为勾、部分为横杠、全未启用为空。
func configModelsCheckbox(c *gui.Context) {
	t := c.Theme()
	state := gui.CheckOff
	on := configEnabledModelCount()
	switch {
	case on == len(configScreen.models) && on > 0:
		state = gui.CheckOn
	case on > 0:
		state = gui.CheckMixed
	}
	box := gui.CheckboxStateBase(c, &state).Size(configCheckbox, configCheckbox).
		Shrink(0).Radius(configBadgeRadius).Label("模型")
	if box.Changed() {
		// 逐个开关是模型供应领域的事实：这里只改界面状态。
		next := state == gui.CheckOn
		for i := range configScreen.models {
			configScreen.models[i].on = next
		}
	}
	box.Children(func() {
		switch state {
		case gui.CheckOn, gui.CheckMixed:
			box.DrawOver(func(p *gui.Painter, r gui.Rect) {
				p.Fill(r, t.Accent, configBadgeRadius)
				icon := iconCheck
				if state == gui.CheckMixed {
					icon = configIconMinus
				}
				p.Icon(icon, r, t.AccentText)
			})
		default:
			box.DrawOver(func(p *gui.Painter, r gui.Rect) {
				p.Stroke(r, t.Border, configBadgeRadius, screenStroke)
			})
		}
	})
}

// configEnabledModelCount 是已启用的模型数（界面状态）。
func configEnabledModelCount() int {
	n := 0
	for _, model := range configScreen.models {
		if model.on {
			n++
		}
	}
	return n
}

// configVisibleModelIndexes 是过滤后的模型下标（名称子串、不区分大小写）。
func configVisibleModelIndexes() []int {
	out := make([]int, 0, len(configScreen.models))
	for i, model := range configScreen.models {
		if configMatches(configScreen.modelFilter, model.name) {
			out = append(out, i)
		}
	}
	return out
}

// configModelList 是模型列表（S-configuration-23/24）：1px 外框，行间一条细线，
// 列表自己滚。
func configModelList(c *gui.Context) {
	t := c.Theme()
	frame := gui.Column(c).Grow(1).MinHeight(0).Radius(screenRadiusCtl).
		Background(t.Background).Label("模型列表")
	frame.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	frame.Children(func() {
		indexes := configVisibleModelIndexes()
		if len(indexes) == 0 {
			// S-configuration-24：没有匹配的模型。
			gui.Row(c).Grow(1).Center().Padding(configEmptyPad, 0).Children(func() {
				configRowStyle.style(t, gui.Text(c, "没有匹配的模型")).
					TextColor(t.TextMuted).SingleLine()
			})
			return
		}
		gui.Scroll(c).Grow(1).Label("模型").Children(func() {
			gui.Column(c).Shrink(0).Children(func() {
				for k, i := range indexes {
					configModelRow(c, i)
					if k < len(indexes)-1 {
						screenRule(c)
					}
				}
			})
		})
	})
}

// configModelRow 是一行模型：名称 + 能力徽标 + 开关（S-configuration-23）。
func configModelRow(c *gui.Context, index int) {
	t := c.Theme()
	model := &configScreen.models[index]
	ink := t.Text
	if !*configScreen.providerToggle(configDimName) {
		// 供应商关掉后模型名退到次级前景（参考的 !isSelectedProviderConfigured）。
		ink = t.TextMuted
	}
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(configGapBlock).
		Padding(configModelPadV, configModelPadH).Children(func() {
		gui.Row(c).Grow(1).MinWidth(0).AlignItems(gui.Center).Gap(configGapIcon).
			Children(func() {
				configRowStyle.style(t, gui.Text(c, model.name)).TextColor(ink).
					SingleLine()
				if model.vision {
					configBadge(c, "视觉", "能力：视觉")
				}
				if model.reasoning {
					configBadge(c, "推理", "能力：推理")
				}
			})
		gui.Switch(c, &model.on).Size(configSwitchW, configSwitchH).
			Radius(configSwitchH / 2).Label("启用模型 " + model.name)
	})
}

// configBadge 是一个能力徽标（10、主色、primary/10 底、圆角 4）。
func configBadge(c *gui.Context, text, label string) {
	t := c.Theme()
	gui.Row(c).Shrink(0).AlignItems(gui.Center).
		Padding(configBadgePadV, configBadgePadH).Radius(configBadgeRadius).
		Background(t.Accent.Alpha(configBadgeFill)).Label(label).Children(func() {
		configBadgeStyle.style(t, gui.Text(c, text)).TextColor(colorAccentInk.in(t)).
			SingleLine()
	})
}

// configDetailFooter 是详情底部动作（S-configuration-27/28）：刷新、查看套餐。
func configDetailFooter(c *gui.Context) {
	gui.Row(c).Shrink(0).Justify(gui.End).Gap(configGapRow).Children(func() {
		if screenAction(c, screenIconRefresh, "refresh", "刷新", rowRegular) {
			// 拉取模型列表是模型供应领域的事实：控件照渲染，点击不写事实。
		}
		if screenAction(c, configIconExternal, "external", "查看套餐", rowRegular) {
			// 外部页面（套餐页）的地址由账户领域给，本屏不拼 URL。
		}
	})
}

// configUnbuiltBody 是其余供应商的详情：配置表单（API 密钥 / 测试连接 /
// 断开 / 删除，S-configuration-44 … 63）尚未重建，用设置壳的同一句口径占位。
func configUnbuiltBody(c *gui.Context) {
	t := c.Theme()
	configRowStyle.style(t, gui.Text(c, "尚未按参考产品重建")).TextColor(t.TextMuted)
}
