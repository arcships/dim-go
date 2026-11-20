package ui

import (
	"context"
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
// 只保留可丢的界面状态（configScreen）：搜索词、选中项、两个分组的折叠、
// 启用开关、模型的开关。事实（供应商清单、标识、启停、基础 URL 与模型）连着
// Source 时每帧从 Settings().ProviderRecords 读，没有 Source（-demo）时用运行
// 时快照的两组样例。
//
// 已接的写入口（都走既有 Intents）：
//
//	模型开关与「全选」→ SetProviderModelEnabled（provider.models.set_enabled）；
//	详情底部「刷新」→ ProbeProvider（provider.probe，只读，不写配置）；
//	左栏「添加」与详情底部「编辑」→ 供应商表单，保存走 CreateProvider /
//	    UpdateProvider（字段、校验与事实边界见 settings_provider_form.go）。
//
// 未接的写入口（屏上没有对应的控件或事实）：
//
//	provider 级「启用」开关：UpdateProvider 的签名不含 enabled，本屏只改界面状态。
//	「查看套餐」与账户托管提示条的跳转：跨分节路由与地址都不在本屏的事实里。
//
// 未重建的分支：详情面板的另外几种形态（OpenAI Codex OAuth / Grok Build 设备码 /
// 通用托管 OAuth / 未配置表单 / 已配置凭据）与「添加模型」对话框，本屏用与设置壳
// 同一句口径（「尚未按参考产品重建」）占位，不摆看起来能用的控件
// （S-configuration-10 … 80 的未出现分支）。读数的 provider 走 configRecordBody
// （基础 URL + 模型块 + 底部动作），不再落到占位句。
//
// 两处没有资产，用等尺寸示意：供应商品牌图标（参考的 ProviderIcon）统一用
// 中性图标（不按名字造字母占位）；能力徽标只画运行时出现的「视觉」「推理」两种。

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
	// configMarkInk 是图标位里中性图标相对圆底的比例（24 网格的图形留白）。
	configMarkInk  = 0.75
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
)

// 本屏图标：Heroicons v2 outline（24 网格、1.5 描边、MIT）的内联路径，与
// icons.go 同一套约定；chevron 与 refresh 复用外观屏的同名图形。
var (
	configIconLock     = mustSVG(`<path d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z"/>`)
	configIconExternal = mustSVG(`<path d="M13.5 6H5.25A2.25 2.25 0 0 0 3 8.25v10.5A2.25 2.25 0 0 0 5.25 21h10.5A2.25 2.25 0 0 0 18 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25"/>`)
	configIconMinus    = mustSVG(`<path d="M5.25 12h13.5"/>`)
)

// configModel 是一行模型：标识、名称、两项能力与开关初值。连着 Source 时来自
// Settings().ProviderRecords（标识与启停是真的）；只在 -demo 时用运行时快照的
// 32 个模型做样例。
type configModel struct {
	id        string
	name      string
	vision    bool
	reasoning bool
	on        bool
}

// configProvider 是一个供应商：标识、名称、是否已配置、是否由账户托管、
// 基础 URL 与可写版本。account 为真的只有样例里的 DimAgent OAuth
// （运行时唯一已启用且账户托管的一项）；读数的 provider 一律按普通项画。
type configProvider struct {
	id         string
	name       string
	configured bool
	account    bool
	baseURL    string
	version    int
	// authType / driverKind 是协议与认证方式（读数原文，不在界面上改）；
	// credentialConfigured 表示服务端已存凭据（只给「已设置」这一位事实）；
	// remoteModelList 表示协议支持从上游拉模型清单。
	authType             string
	driverKind           string
	credentialConfigured bool
	remoteModelList      bool
}

// configKey 是供应商在视图状态里的键：读数的用服务端标识，样例用名称。
func configKey(provider configProvider) string {
	if provider.id != "" {
		return provider.id
	}
	return provider.name
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

	// providerToggles 是按供应商键的启用开关（界面态）。初始只有账户托管的
	// DimAgent OAuth 为开（运行时唯一已启用的一项）。
	providerToggles map[string]*bool
	// modelToggles 是按「供应商键/模型标识」的模型开关（界面态，初值来自读数）。
	modelToggles map[string]*bool
	// models 是 -demo 的样例模型表（读数为真时每帧从 ProviderRecords 取）。
	models []configModel
	// form 是新增 / 编辑供应商表单的草稿（见 settings_provider_form.go）；
	// nil 表示右栏是详情态。
	form *configForm
	// draft 是「添加模型」对话框的本地草稿（见 settings_provider_detail.go）；
	// nil 表示对话框没开。它只校验，不发写请求（模型新建没有通道）。
	draft *configModelDraft
	// probing 是一次 provider 探测（测试连接 / 刷新）进行中：两个入口都禁用。
	probing   bool
	probeErr  string
	probeNote string
	// disconnectFor / deleteFor 是打开着的确认框指向的供应商键；空表示没开。
	disconnectFor string
	deleteFor     string
	// toggleErr 是模型全选写入失败的原因（失败后开关回到改动前的值）。
	toggleErr string
	// selectAfterCreate 是新增成功后要选中的供应商名：新建回执不带标识，
	// 等下一帧的读数里按名字找到它再选中。
	selectAfterCreate string
}

var configScreen = configState{
	selected: configDimName,
	models:   append([]configModel(nil), configModelTable...),
}

// providerToggle 取一个供应商的启用开关（按需建、指针稳定）。
func (s *configState) providerToggle(provider configProvider) *bool {
	if s.providerToggles == nil {
		s.providerToggles = map[string]*bool{}
	}
	key := configKey(provider)
	if on, ok := s.providerToggles[key]; ok {
		return on
	}
	on := provider.configured
	s.providerToggles[key] = &on
	return &on
}

// modelToggle 取一个模型的开关（按需建、指针稳定，初值来自这一帧的读数）。
func (s *configState) modelToggle(provider configProvider, model configModel) *bool {
	if s.modelToggles == nil {
		s.modelToggles = map[string]*bool{}
	}
	key := configKey(provider) + "/" + model.id
	if model.id == "" {
		key = configKey(provider) + "/" + model.name
	}
	if on, ok := s.modelToggles[key]; ok {
		return on
	}
	on := model.on
	s.modelToggles[key] = &on
	return &on
}

// renderConfiguration 是设置 › 模型供应这一屏（设置壳的一个分节）。
func renderConfiguration(c *gui.Context, a *App) {
	t := c.Theme()
	data := a.configView()
	gui.Row(c).Grow(1).AlignItems(gui.Stretch).Background(t.Background).
		Label("模型供应").Children(func() {
		a.configProviderColumn(c, data)
		a.configDetailColumn(c, data)
	})
}

// configView 是一帧的模型供应读数：供应商清单、读数错误、选中项与它的模型。
//
// 连着 Source 时全部来自 Settings().ProviderRecords（标识、启停、基础 URL 与
// 模型启停都是服务端事实）；没有 Source（-demo）时用运行时快照的样例。
type configView struct {
	providers   []configProvider
	readErr     string
	selected    configProvider
	hasSelected bool
	models      []configModel
	// loading 表示清单还没读到过（清单为空且没有错误）：画加载骨架，不把
	// 「还没读到」画成「没有供应商」。
	loading bool
}

// configView 取这一帧的读数。选中项只是视图状态（configScreen.selected），
// 读数里找不到它时落到第一项，不空着详情。
func (a *App) configView() configView {
	v := configView{}
	if a.source == nil {
		v.providers = configSampleProviders()
		v.models = configScreen.models
	} else {
		settings, _ := a.liveSettings()
		v.readErr = settings.ProviderError
		v.loading = !settings.ProviderRecordsKnown && settings.ProviderError == "" &&
			len(settings.ProviderRecords) == 0
		for _, record := range settings.ProviderRecords {
			v.providers = append(v.providers, configProvider{
				id: record.ID, name: configRecordName(record),
				configured: record.Enabled, baseURL: record.BaseURL,
				version: record.Version, authType: record.AuthType,
				driverKind: record.DriverKind,
				credentialConfigured: record.CredentialConfigured,
				remoteModelList:      record.RemoteModelList,
			})
		}
	}
	for _, provider := range v.providers {
		if configKey(provider) != configScreen.selected {
			continue
		}
		v.selected, v.hasSelected = provider, true
		break
	}
	if !v.hasSelected && len(v.providers) > 0 {
		v.selected, v.hasSelected = v.providers[0], true
		configScreen.selected = configKey(v.selected)
	}
	// 新建成功后按名字选中新供应商（回执不带标识，读数是唯一来源）。
	if name := configScreen.selectAfterCreate; name != "" {
		for _, provider := range v.providers {
			if !strings.EqualFold(strings.TrimSpace(provider.name), name) {
				continue
			}
			v.selected, v.hasSelected = provider, true
			configScreen.selected = configKey(provider)
			configScreen.selectAfterCreate = ""
			break
		}
	}
	if a.source != nil && v.hasSelected {
		v.models = a.configRecordModels(v.selected)
	}
	return v
}

// configRecordName 是 provider 的展示名：没有展示名时用标识（不空着）。
func configRecordName(record ProviderRecordSnapshot) string {
	if strings.TrimSpace(record.Name) != "" {
		return record.Name
	}
	return record.ID
}

// configRecordModels 取一个供应商的模型行（读数的标识与启停；没有能力徽标事实）。
func (a *App) configRecordModels(provider configProvider) []configModel {
	settings, _ := a.liveSettings()
	for _, record := range settings.ProviderRecords {
		if record.ID != provider.id {
			continue
		}
		out := make([]configModel, 0, len(record.Models))
		for _, model := range record.Models {
			name := model.ID
			if strings.TrimSpace(model.Name) != "" {
				name = model.Name
			}
			out = append(out, configModel{id: model.ID, name: name, on: model.Enabled})
		}
		return out
	}
	return nil
}

// ---- 左栏：供应商列表（S-configuration-01 … 09） ----

// configProviderColumn 是左栏：栏标题 + 搜索 + 两段分组 + 底部「添加」。
func (a *App) configProviderColumn(c *gui.Context, data configView) {
	t := c.Theme()
	enabled, disabled := configFilteredProviders(data.providers)
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
				// S-configuration-01 左栏标题「模型供应」
				configTitleStyle.style(t, gui.Text(c, "模型供应")).TextColor(t.TextMuted).
					LetterSpacing(configTracking).SingleLine().
					Role(gui.RoleHeading).Level(3)
			})
		// S-configuration-02 / 03：搜索框（放大镜在左、输入框在右）。
		gui.Row(c).Shrink(0).Padding(0, configPadH, configAddPad, configPadH).Children(func() {
			// S-configuration-02 / S-configuration-03 搜索供应商（放大镜 + 输入框）
			configSearchField(c, &configScreen.providerFilter, "搜索供应商", "搜索供应商")
		})
		// 读数失败与读空：两行都来自读数，不拿空清单当「没有供应商」。
		if data.readErr != "" {
			gui.Row(c).Shrink(0).Padding(0, configPadH, configAddPad, configPadH).
				Children(func() {
					// S-configuration-04 读取供应商失败
					configMicroStyle.style(t, gui.Text(c, "读取供应商失败："+data.readErr)).
						TextColor(t.Danger).Role(gui.RoleAlert).MaxLines(2)
				})
		}
		// S-configuration-05 … 08：两段分组，列表自己在列内滚动。
		gui.Scroll(c).Grow(1).Label("供应商").Children(func() {
			gui.Column(c).Shrink(0).Gap(configGapGroups).
				Padding(0, configPadH, configAddPad, configPadH).Children(func() {
				if len(data.providers) == 0 {
					// S-configuration-04 没有读到供应商
					configMicroStyle.style(t, gui.Text(c, "没有读到供应商。")).
						TextColor(t.TextMuted).Role(gui.RoleStatus).MaxLines(2)
					return
				}
				// S-configuration-05 / S-configuration-06 已启用分组
				a.configProviderGroup(c, data, "已启用", enabled,
					&configScreen.enabledCollapsed)
				// S-configuration-07 / S-configuration-08 未启用分组
				a.configProviderGroup(c, data, "未启用", disabled,
					&configScreen.disabledCollapsed)
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

// configSearchField 是一行搜索：输入框底 + 1px 主题描边 + 放大镜 + 无内框的输入框。
func configSearchField(c *gui.Context, value *string, placeholder, label string) {
	t := c.Theme()
	field := gui.Row(c).Grow(1).Height(configFieldH).AlignItems(gui.Center).
		Gap(configGapIcon).Padding(configFieldPadV, configFieldPadH).
		Radius(screenRadiusCtl).Background(fieldFill(t)).Label(label)
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
func configFilteredProviders(providers []configProvider) (enabled, disabled []configProvider) {
	for _, provider := range providers {
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

// configSampleProviders 是 -demo 的供应商全表：DimAgent OAuth 在前（已启用、
// 账户托管），其后是 127 个未启用项。连着 Source 时整表来自读数。
func configSampleProviders() []configProvider {
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
func (a *App) configProviderGroup(c *gui.Context, data configView, label string,
	providers []configProvider, collapsed *bool) {
	// 搜索生效时两组强制展开（参考行为），折叠状态本身不改。
	expanded := !*collapsed || configScreen.providerFilter != ""
	configGroupHead(c, label, len(providers), expanded, collapsed)
	if !expanded {
		return
	}
	gui.Column(c).Shrink(0).Gap(configGapItems).Children(func() {
		for _, provider := range providers {
			a.configProviderRow(c, data, provider)
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
func (a *App) configProviderRow(c *gui.Context, data configView, provider configProvider) {
	t := c.Theme()
	selected := configScreen.selected == configKey(provider)
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
		// 换供应商时收起表单：表单的草稿属于上一项，留着会让人以为在改新的这一项。
		configScreen.selected = configKey(provider)
		configFormClose()
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

// configProviderMark 是供应商图标位：没有品牌资产，统一用中性图标（不按名字
// 造字母占位，127 行才不会重复同一个字母块）；已启用的用主色底。
func configProviderMark(c *gui.Context, provider configProvider) {
	configProviderMarkIcon(c, provider.name, configIcon, provider.configured)
}

// configProviderMarkIcon 画一个供应商图标位：圆底 + 中性图标。
func configProviderMarkIcon(c *gui.Context, name string, size float32, configured bool) {
	t := c.Theme()
	box := gui.Box(c).Size(size, size).Shrink(0).Center().
		Label("图标：" + name).AccessHidden(true)
	fill, ink := t.SurfacePressed, t.TextMuted
	if configured {
		fill, ink = t.Accent.Alpha(configBadgeFill), colorAccentInk.in(t)
	}
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, fill, size/2)
	})
	box.Children(func() {
		icon := gui.Box(c).Size(size*configMarkInk, size*configMarkInk).Shrink(0)
		icon.DrawOver(func(p *gui.Painter, r gui.Rect) {
			paintIcon(p, iconProvider, r, ink)
		})
	})
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
// S-configuration-09 左栏底部的「添加」
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
		// S-configuration-64：打开「添加自定义提供商」表单（见
		// settings_provider_form.go）。
		configScreen.form = configFormNew()
	}
}

// ---- 右栏：详情（S-configuration-11 … 28） ----

// configDetailColumn 是右栏：新增 / 编辑表单，或头部 + 账户托管（或读数详情）。
func (a *App) configDetailColumn(c *gui.Context, data configView) {
	gui.Column(c).Grow(1).MinWidth(0).Padding(configDetailPad).Children(func() {
		// 表单占满右栏：它与详情互斥（参考的 panelMode）。
		if configScreen.form != nil {
			a.configFormPanel(c, data)
			return
		}
		gui.Column(c).Width(configDetailContentWidth(c)).Grow(1).Gap(configDetailGap).
			Children(func() {
				if !data.hasSelected {
					// S-configuration-82：没有选中项时的空态。
					configMicroStyle.style(c.Theme(), gui.Text(c, "选择一个提供商查看详情")).
						TextColor(c.Theme().TextMuted)
					return
				}
				a.configDetailHead(c, data.selected)
				switch {
				case data.selected.account:
					a.configManagedBody(c, data)
				case a.source != nil:
					// 读数的 provider：基础 URL 与模型启停都是事实，照画同一套形态。
					a.configRecordBody(c, data)
				default:
					configUnbuiltBody(c)
				}
			})
	})
}

// configDetailContentWidth 是详情内容的宽：参考的 max-w-2xl（672）封顶，
// 窄窗时收进可用宽度（见 design-baseline §7 的设置内容最大宽 1008）。
func configDetailContentWidth(c *gui.Context) float32 {
	usable := float32(math.Min(float64(settingsUsableWidth(c)), SettingsContentMaxWidth))
	avail := usable - 2*space8 - configListW - 2*configDetailPad
	if avail < configDetailMinW {
		return configDetailMinW
	}
	if avail > configDetailMaxW {
		return configDetailMaxW
	}
	return avail
}

// configDetailHead 是详情头部：图标 + 名称 + 启用状态 + 启用开关，下方一条底线
// （S-configuration-11 … 13）。
func (a *App) configDetailHead(c *gui.Context, provider configProvider) {
	t := c.Theme()
	toggle := configScreen.providerToggle(provider)
	gui.Column(c).Shrink(0).Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Justify(gui.SpaceBetween).
			Gap(configGapRow).Padding(0, 0, configHeadPad, 0).Children(func() {
			gui.Row(c).Grow(1).MinWidth(0).AlignItems(gui.Center).
				Gap(configGapRow).Children(func() {
				configProviderMarkBig(c, provider)
				gui.Column(c).Grow(1).MinWidth(0).Children(func() {
					configStrongStyle.style(t, gui.Text(c, provider.name)).TextColor(t.Text).
						SingleLine()
					configMicroStyle.style(t, gui.Text(c, configStatusLabel(*toggle))).
						TextColor(t.TextMuted).SingleLine()
				})
			})
			gui.Switch(c, toggle).
				Size(configSwitchW, configSwitchH).Radius(configSwitchH / 2).
				Label("启用 " + provider.name)
		})
		screenRule(c)
	})
}

// configStatusLabel 是详情头部的状态词（参考 selectedProviderStatusLabel）：
// 与开关同源，开着就是「已启用」。
func configStatusLabel(on bool) string {
	if on {
		return "已启用"
	}
	return "未启用"
}

// configProviderMarkBig 是详情头部的供应商图标位（16）：与列表行同一个中性图标。
func configProviderMarkBig(c *gui.Context, provider configProvider) {
	configProviderMarkIcon(c, provider.name, configIconBig, provider.configured)
}

// configManagedBody 是账户托管供应商的只读详情（S-configuration-14 … 28）：
// 提示条 + 基础 URL + 模型块 + 底部动作。
func (a *App) configManagedBody(c *gui.Context, data configView) {
	gui.Column(c).Grow(1).MinHeight(0).Gap(configDetailGap).Children(func() {
		gui.Column(c).Shrink(0).Gap(configDetailGap).Children(func() {
			configManagedBanner(c)
			configBaseURL(c, data.selected.baseURL)
		})
		a.configModelsPanel(c, data)
		a.configDetailFooter(c, data)
	})
}

// configRecordBody 是读数里的 provider：基础 URL（有就读）+ 模型块 + 底部动作。
// 账户托管提示条只在样例的 DimAgent OAuth 上出现（读数没有「托管」这项事实）。
func (a *App) configRecordBody(c *gui.Context, data configView) {
	gui.Column(c).Grow(1).MinHeight(0).Gap(configDetailGap).Children(func() {
		if data.selected.baseURL != "" {
			gui.Column(c).Shrink(0).Gap(configDetailGap).Children(func() {
				configBaseURL(c, data.selected.baseURL)
			})
		}
		a.configModelsPanel(c, data)
		a.configDetailFooter(c, data)
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
		// S-configuration-83 「由 Dim 账户管理」提示条（跳账户设置）
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
func (a *App) configModelsPanel(c *gui.Context, data configView) {
	gui.Column(c).Grow(1).MinHeight(0).Gap(configGapRow).Children(func() {
		a.configModelsToolbar(c, data)
		gui.Row(c).Shrink(0).Children(func() {
			// S-configuration-72 搜索模型名称或 ID
			configSearchField(c, &configScreen.modelFilter, "搜索模型名称或 ID",
				"搜索模型名称或 ID")
		})
		a.configModelList(c, data)
	})
}

// configModelsToolbar 是「模型 · 可见 / 总数」一行：三态复选框 + 标签
// （S-configuration-20）。
func (a *App) configModelsToolbar(c *gui.Context, data configView) {
	t := c.Theme()
	// S-configuration-70 模型 · {visible}/{total}
	visible := configVisibleModelIndexes(data.models)
	label := fmt.Sprintf("模型 · %d / %d", len(visible), len(data.models))
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(configGapRow).Children(func() {
		a.configModelsCheckbox(c, data)
		configMicroStyle.style(t, gui.Text(c, label)).TextColor(t.TextMuted).
			Label(label).SingleLine()
	})
}

// configModelsCheckbox 是全选框：全部启用为勾、部分为横杠、全未启用为空。
func (a *App) configModelsCheckbox(c *gui.Context, data configView) {
	t := c.Theme()
	state := gui.CheckOff
	on := configEnabledModelCount(data)
	switch {
	case on == len(data.models) && on > 0:
		state = gui.CheckOn
	case on > 0:
		state = gui.CheckMixed
	}
	box := gui.CheckboxStateBase(c, &state).Size(configCheckbox, configCheckbox).
		Shrink(0).Radius(configBadgeRadius).Label("模型")
	if box.Changed() {
		// 全选 / 全不选：每个模型各发一次启停（读数里的模型才有标识）。
		next := state == gui.CheckOn
		for _, model := range data.models {
			*configScreen.modelToggle(data.selected, model) = next
			a.configSetModelEnabled(data.selected, model, next)
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
				paintIcon(p, icon, r, t.AccentText)
			})
		default:
			box.DrawOver(func(p *gui.Painter, r gui.Rect) {
				p.Stroke(r, t.Border, configBadgeRadius, screenStroke)
			})
		}
	})
}

// configEnabledModelCount 是已启用的模型数（开关位是界面状态，初值来自读数）。
func configEnabledModelCount(data configView) int {
	n := 0
	for _, model := range data.models {
		if *configScreen.modelToggle(data.selected, model) {
			n++
		}
	}
	return n
}

// configVisibleModelIndexes 是过滤后的模型下标（名称子串、不区分大小写）。
func configVisibleModelIndexes(models []configModel) []int {
	out := make([]int, 0, len(models))
	for i, model := range models {
		if configMatches(configScreen.modelFilter, model.name) {
			out = append(out, i)
		}
	}
	return out
}

// configModelList 是模型列表（S-configuration-23/24）：1px 外框，行间一条细线，
// 列表自己滚。
func (a *App) configModelList(c *gui.Context, data configView) {
	// S-configuration-73 模型列表（名称 + 能力徽标 + 开关）
	t := c.Theme()
	frame := gui.Column(c).Grow(1).MinHeight(0).Radius(screenRadiusCtl).
		Background(t.Background).Label("模型列表")
	frame.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	frame.Children(func() {
		indexes := configVisibleModelIndexes(data.models)
		if len(indexes) == 0 {
			// S-configuration-24：没有匹配的模型。
			gui.Row(c).Grow(1).Center().Padding(configEmptyPad, 0).Children(func() {
				// S-configuration-74 / S-configuration-80 没有匹配的模型
				configRowStyle.style(t, gui.Text(c, "没有匹配的模型")).
					TextColor(t.TextMuted).SingleLine()
			})
			return
		}
		gui.Scroll(c).Grow(1).Label("模型").Children(func() {
			gui.Column(c).Shrink(0).Children(func() {
				for k, i := range indexes {
					a.configModelRow(c, data, i)
					if k < len(indexes)-1 {
						screenRule(c)
					}
				}
			})
		})
	})
}

// configModelRow 是一行模型：名称 + 能力徽标 + 开关（S-configuration-23）。
// S-configuration-73 一行模型（名称 + 视觉 / 推理徽标 + 启停开关）
func (a *App) configModelRow(c *gui.Context, data configView, index int) {
	t := c.Theme()
	model := data.models[index]
	ink := t.Text
	if !*configScreen.providerToggle(data.selected) {
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
		toggle := configScreen.modelToggle(data.selected, model)
		sw := gui.Switch(c, toggle).Size(configSwitchW, configSwitchH).
			Radius(configSwitchH / 2).Label("启用模型 " + model.name)
		if sw.Changed() {
			a.configSetModelEnabled(data.selected, model, *toggle)
		}
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

// configDetailFooter 是详情底部动作（S-configuration-27/28）：编辑、刷新、查看套餐。
// S-configuration-27 / S-configuration-28 详情底部动作（编辑 / 刷新 / 查看套餐）
func (a *App) configDetailFooter(c *gui.Context, data configView) {
	gui.Row(c).Shrink(0).Justify(gui.End).Gap(configGapRow).Children(func() {
		// 账户托管的供应商不能在这里改（提示条写明原因），其余可进编辑表单。
		if !data.selected.account {
			if screenAction(c, iconPencil, "edit", "编辑", rowRegular) {
				configScreen.form = configFormEdit(data.selected, data.models)
			}
		}
		if screenAction(c, screenIconRefresh, "refresh", "刷新", rowRegular) {
			// 「刷新」= 重新探测这个 provider 的模型清单（provider.probe，只读）。
			a.configProbeProvider(data.selected)
		}
		if screenAction(c, configIconExternal, "external", "查看套餐", rowRegular) {
			// 外部页面（套餐页）的地址由账户领域给，本屏不拼 URL。
		}
	})
}

// configUnbuiltBody 是其余供应商的详情：配置表单（API 密钥 / 测试连接 /
// 断开 / 删除，S-configuration-44 … 63）尚未重建，用设置壳的同一句口径占位。
// S-configuration-44 … 63 配置表单尚未重建的占位
func configUnbuiltBody(c *gui.Context) {
	t := c.Theme()
	configRowStyle.style(t, gui.Text(c, "尚未按参考产品重建")).TextColor(t.TextMuted)
}

// S-configuration-81 加载骨架（读数未到的占位）
// configSetModelEnabled 把模型开关落到服务端（provider.models.set_enabled）。
// 样例行（没有标识）与没有 Intents 时只改界面状态。
func (a *App) configSetModelEnabled(provider configProvider, model configModel, enabled bool) {
	if a.intents == nil || provider.id == "" || model.id == "" {
		return
	}
	providerID, modelID := provider.id, model.id
	a.dispatch("启停模型", func(ctx context.Context) error {
		return a.intents.SetProviderModelEnabled(ctx, providerID, modelID, enabled)
	}, nil)
}

// configProbeProvider 重新探测一个 provider 的模型清单（provider.probe，只读，
// 不写配置）。结果由下一帧的 ProviderRecords / Models 读数带回来。
func (a *App) configProbeProvider(provider configProvider) {
	if a.intents == nil || provider.id == "" {
		return
	}
	id := provider.id
	a.dispatch("探测供应商", func(ctx context.Context) error {
		_, err := a.intents.ProbeProvider(ctx, id)
		return err
	}, nil)
}
