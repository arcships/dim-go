package ui

import (
	"strconv"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 插件设置的重建实现：元素按
// docs/domains/requirements/inventory-settings-rt.md §16（S-pluginSettings-01…30，
// 30 个元素）逐行对应，文案逐字取参考产品中文（插件名与字段名来自插件 schema，
// 不翻译），数值取 docs/domains/requirements/design-baseline.md：
//
//	卡标题 13/600、卡片圆角 8、卡内边距 14、
//	字段标签 12.25/500、输入框高 32、开关 28×16、行内按钮高 28、字段间 16。
//
// 参考的这一节没有页首标题与说明（第一张插件卡就是首块内容），本屏也不画。
//
// 只保留可丢的界面状态（pluginSettingsScreen）：3 个插件的表单草稿、保存回执、
// 读写范围。写事实（插件设置的保存与重新加载）还没有对应 Intents 方法：
// 控件照渲染、点击只改本屏状态。
//
// # 与参考的差异
//
//	内容列的内边距由设置壳给（28），参考的这一节是通栏（p-0）：
//	本屏不自行去掉外壳内边距（那会与其余 18 节不一致），差异登记在此。
//	Web 插件的 Search Service 在参考截图里位于一个带边框、缩进 12 的组里
//	（png 实测 512.5–552），按对象字段（S-pluginSettings-30）渲染，组标题为空。
//	「已保存」与「保存失败」没有服务端回执来源，只在保存点击后按本屏校验给出。

// pluginFieldKind 是字段的控件形状，由插件 schema 决定。
type pluginFieldKind int

const (
	plgBool pluginFieldKind = iota
	plgInt
	plgString
	plgArray
	plgGroup
	plgSecret
)

// pluginField 是一个设置字段：标签、说明、形状与当前草稿值。
type pluginField struct {
	name     string
	label    string
	desc     string
	required bool
	kind     pluginFieldKind
	// on / text / items / fields 按 kind 取用。
	on     bool
	text   string
	items  []string
	fields []pluginField
}

// pluginSource 是一个插件的设置卡。
type pluginSource struct {
	title  string
	id     string
	desc   string
	fields []pluginField

	loading   bool
	loadError string
	conflict  bool
	saving    bool
	saved     bool
	err       string
}

// pluginSettingsState 是本屏可丢的界面状态：不是事实。
type pluginSettingsState struct {
	loading     bool
	unavailable bool
	loadError   string

	// scope 是保存范围：0 用户、1 项目（参考的 activeScope）。
	scope          int
	projectSources int
	projectRoots   []string
	rootsLoading   bool
	rootsError     string
	rootName       string
	rootChosen     bool

	sources []pluginSource
}

// pluginSettingsScreen 的首屏取值与参考产品运行时一致：3 个插件、6 个字段实例、
// 用户范围、无冲突、未保存。
var pluginSettingsScreen = pluginSettingsState{
	projectRoots: []string{"未选择"},
	rootName:     "未选择",
	sources: []pluginSource{
		{
			title: "Model log", id: "model-log",
			fields: []pluginField{
				{name: "enabled", label: "Enabled", kind: plgBool},
			},
		},
		{
			title: "Prompt suggestion", id: "prompt-suggestion",
			fields: []pluginField{
				{name: "enabled", label: "Enabled", kind: plgBool},
				{name: "maxChars", label: "Max Chars", kind: plgInt, text: "80"},
			},
		},
		{
			title: "Web", id: "web",
			fields: []pluginField{
				{
					name: "search", kind: plgGroup,
					fields: []pluginField{
						{name: "service", label: "Search Service", kind: plgBool},
					},
				},
				{name: "allowPrivateHosts", label: "Allow Private Hosts", kind: plgArray},
				{name: "proxy", label: "Proxy", kind: plgString},
				{name: "proxyDns", label: "Proxy Dns", kind: plgBool},
			},
		},
	},
}

// 本屏几何：参考的 Tailwind 量换算成 design-baseline §1.1 的档位。
const (
	plgCardGap   = 16 // 卡与卡之间
	plgCardPad   = 14 // p-4：卡内边距
	plgCardInner = 12 // 卡内块间距（卡标题区与字段区之间）
	plgHeadGap   = 2  // 插件名与 source id 之间
	plgLabelGap  = 6  // 字段标签与控件之间
	plgFieldGap  = 16 // 字段与字段之间
	plgBtnGap    = 8  // 保存与重新加载之间
	plgGroupPad  = 12 // 对象组的缩进（组内字段）
	plgGroupGap  = 16

	plgCtlH     = 32 // h-9：输入框与下拉
	plgBtnH     = 28 // h-8：保存 / 重新加载 / 添加
	plgSwitchW  = 28
	plgSwitchH  = 16
	plgIconSize = 14
	plgIconHit  = 24
	plgAddPadV  = 3.5
	plgAlpha05  = 0.5
)

// 字阶（design-baseline §2）。
var (
	plgDescStyle  = fontStep{size: 12.25, line: 17.5, weight: 400}
	plgCardTitle  = fontStep{size: 13, line: 19.5, weight: 600}
	plgIDStyle    = fontStep{size: 11, line: 17, weight: 400, family: codeFont}
	plgLabelStyle = fontStep{size: 12.25, line: 17.5, weight: 500}
	plgSavedStyle = fontStep{size: 11, line: 14, weight: 500}
	plgErrStyle   = fontStep{size: 12.25, line: 17.5, weight: 400}
	plgBtnTextSty = fontStep{size: 12.25, line: 17.5, weight: 400}
)

// 本屏文案：逐字取自参考产品（界面语为 zh-CN，插件名与字段名原样）。
const (
	plgCopyLoading      = "正在加载插件设置..."
	plgCopyTitle        = "插件设置"
	plgCopyUnavailable  = "当前 App Server 不提供插件设置。"
	plgCopyEmpty        = "没有已安装插件提供设置。"
	plgCopyRetry        = "重试"
	plgCopyScopeUser    = "用户"
	plgCopyScopeProj    = "项目"
	plgCopyScopeNote    = "项目范围只影响当前项目根目录下的插件设置。"
	plgCopyRootTitle    = "项目根目录"
	plgCopyNoRootChoice = "未选择"
	plgCopyRootReload   = "重新加载"
	plgCopyRootNote     = "设置将从 %s 读取并写回；该根目录由已打开的项目会话授权。"
	plgCopyRootsError   = "读取项目根目录失败。"
	plgCopyRootsEmpty   = "当前没有已授权的项目根目录。请在项目中打开会话以完成授权。"
	plgCopyNoProjSrc    = "没有已安装插件声明项目级设置。"
	plgCopyNoRoot       = "尚未选择已授权的项目根目录。"
	plgCopySave         = "保存"
	plgCopySaving       = "正在保存..."
	plgCopyReload       = "重新加载"
	plgCopySaved        = "已保存"
	plgCopyConflict     = "设置已在别处更改。请重新加载后再保存。"
	plgCopyWithheld     = "此值包含密钥字段，当前表单无法安全编辑。"
	plgCopyAdd          = "添加"
	plgCopyRemove       = "移除"
	plgCopyRetryCard    = "重试"
	plgCopyErrFallback  = "读取插件设置失败"
	plgCopyNumberOnly   = "只接受数字"
)

// renderPluginSettings 是设置 › 插件设置这一屏（设置壳的一个分节）：
// 参考的这一节没有页首标题与说明，第一张插件卡就是首块内容。
//
// 连着 Source 时没有插件设置的 schema（也就没有字段实例可画）：参考的卡形态照留
// ——一张写着原因的卡，加卡底的「保存 / 重新加载」置灰按钮——但不画样例的三个
// 插件卡。样例只在 -demo（没有 Source）出现。
func renderPluginSettings(c *gui.Context, a *App) {
	t := c.Theme()
	if a != nil && a.wired() {
		gui.Column(c).Grow(1).Children(func() {
			gui.Scroll(c).Grow(1).Children(func() {
				gui.Column(c).Gap(plgCardGap).Children(func() {
					a.pluginManageView(c)
					plgUnavailableCard(c, t)
				})
			})
		})
		return
	}
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Children(func() {
				plgBody(c, t)
			})
		})
	})
}

// plgUnavailableCard 是连接模式下的插件设置卡：卡形（卡面、描边、卡内边距）与
// 卡底的保存 / 重新加载按钮都在，按钮置灰、卡内只写原因——不列样例插件与字段。
func plgUnavailableCard(c *gui.Context, t *gui.Theme) {
	screenCard(c, func() {
		gui.Column(c).Padding(plgCardPad, plgCardPad).Gap(plgCardInner).
			Children(func() {
				// S-pluginSettings-01 「插件设置」
				plgCardTitle.style(t, gui.Text(c, plgCopyTitle)).TextColor(t.Text).
					SingleLine().Role(gui.RoleHeading).Level(2)
				plgNote(c, t, notePluginSettings)
				gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(plgBtnGap).
					Children(func() {
						plgPrimaryButtonDisabled(c, t, plgCopySave, notePluginSettings)
						plgOutlineButtonDisabled(c, t, plgCopyReload, plgBtnH, 0, notePluginSettings)
						gui.Spacer(c)
					})
			})
	})
}

// plgPrimaryButtonDisabled 是同款主按钮的禁用态（保存）：tooltip 写原因。
func plgPrimaryButtonDisabled(c *gui.Context, t *gui.Theme, text, tip string) {
	b := gui.ButtonBase(c).Shrink(0).Height(plgBtnH).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Label(text).Background(t.Accent).Tooltip(tip).Disabled(true)
	b.Children(func() {
		plgBtnTextSty.style(t, gui.Text(c, text)).TextColor(t.AccentText).SingleLine()
	})
	b.Opacity(plgAlpha05)
}

// plgBody 在页面标题下按状态分支：加载 / 不可用 / 失败 / 空 / 表单。
func plgBody(c *gui.Context, t *gui.Theme) {
	switch {
	case pluginSettingsScreen.loading:
		// S-pluginSettings-03 正在加载插件设置...
		plgNote(c, t, plgCopyLoading)
		return
	case pluginSettingsScreen.unavailable:
		// S-pluginSettings-04 当前 App Server 不提供插件设置。
		plgNote(c, t, plgCopyUnavailable)
		return
	case pluginSettingsScreen.loadError != "":
		plgErrorRow(c, t, pluginSettingsScreen.loadError, plgCopyRetry, func() {
			// 重拉插件设置需要 Intents：不动作（未接动作）。
		})
		return
	}
	if len(pluginSettingsScreen.sources) == 0 {
		// S-pluginSettings-06 没有已安装插件提供设置。
		plgNote(c, t, plgCopyEmpty)
		return
	}
	if pluginSettingsScreen.projectSources > 0 {
		plgScopeRow(c, t)
	}
	if pluginSettingsScreen.scope == 1 {
		plgProjectRootSection(c, t)
		if pluginSettingsScreen.projectSources == 0 {
			// S-pluginSettings-11 没有已安装插件声明项目级设置。
			plgNote(c, t, plgCopyNoProjSrc)
			return
		}
		if !pluginSettingsScreen.rootChosen {
			// S-pluginSettings-12 尚未选择已授权的项目根目录。
			plgNote(c, t, plgCopyNoRoot)
			return
		}
	}
	plgNote(c, t, notePluginWrite)
	gui.Column(c).Shrink(0).Gap(plgCardGap).Children(func() {
		for i := range pluginSettingsScreen.sources {
			plgCard(c, t, &pluginSettingsScreen.sources[i])
		}
	})
}

// plgScopeRow 是保存范围（S-pluginSettings-07）：用户 / 项目分段 + 一句说明。
func plgScopeRow(c *gui.Context, t *gui.Theme) {
	gui.Column(c).Shrink(0).Gap(plgLabelGap).Children(func() {
		gui.Segmented(c, &pluginSettingsScreen.scope, plgCopyScopeUser, plgCopyScopeProj).
			Label("保存范围")
		// S-pluginSettings-07 保存范围（用户 / 项目）与说明
		plgNote(c, t, plgCopyScopeNote)
	})
	gui.Box(c).Height(plgCardGap).Shrink(0)
}

// plgProjectRootSection 是项目根目录（S-pluginSettings-08/09/10）：标签、
// 下拉、「重新加载」与按该根目录的说明。
func plgProjectRootSection(c *gui.Context, t *gui.Theme) {
	gui.Column(c).Shrink(0).Gap(plgLabelGap).Children(func() {
		// S-pluginSettings-08 项目根目录
		plgLabelLine(c, t, plgCopyRootTitle, false)
		gui.Row(c).AlignItems(gui.Center).Gap(plgBtnGap).Children(func() {
			if pluginSettingsScreen.rootsLoading {
				plgNote(c, t, plgCopyLoading)
				return
			}
			options := pluginSettingsScreen.projectRoots
			if len(options) == 0 {
				options = []string{plgCopyNoRootChoice}
			}
			gui.Select(c, &pluginSettingsScreen.rootName, options).Grow(1).
				Height(plgCtlH).Label(plgCopyRootTitle)
			// S-pluginSettings-08 重新加载项目根目录
			plgOutlineButtonDisabled(c, t, plgCopyRootReload, plgBtnH, 0, notePluginWrite)
		})
		if pluginSettingsScreen.rootsError != "" {
			plgNote(c, t, pluginSettingsScreen.rootsError+plgCopyRootsError)
		}
		if pluginSettingsScreen.rootChosen {
			plgNote(c, t, plgCopyRootNote)
		} else {
			// S-pluginSettings-09 / S-pluginSettings-10 项目根目录的说明与空态
			plgNote(c, t, plgCopyRootsEmpty)
		}
	})
	gui.Box(c).Height(plgCardGap).Shrink(0)
}

// plgCard 是一个插件设置卡（S-pluginSettings-13…24）。
func plgCard(c *gui.Context, t *gui.Theme, src *pluginSource) {
	screenCard(c, func() {
		gui.Column(c).Padding(plgCardPad, plgCardPad).Gap(plgCardInner).
			Children(func() {
				gui.Column(c).Shrink(0).Gap(plgHeadGap).Children(func() {
					gui.Row(c).AlignItems(gui.Center).Gap(plgLabelGap).Children(func() {
						plgCardTitle.style(t, gui.Text(c, src.title)).TextColor(t.Text).
							SingleLine().Role(gui.RoleHeading).Level(2)
					})
					if src.desc != "" {
						typeSecondary.style(t, gui.Text(c, src.desc)).
							TextColor(t.TextMuted).Selectable()
					}
					plgIDStyle.style(t, gui.Text(c, src.id)).TextColor(t.TextMuted).
						SingleLine().Selectable().Label("插件标识：" + src.id)
				})
				switch {
				case src.loading:
					plgNote(c, t, plgCopyLoading)
				case src.loadError != "":
					plgErrorRow(c, t, src.loadError, plgCopyRetryCard, func() {
						// 重拉该卡设置需要 Intents：不动作（未接动作）。
					})
				case src.conflict:
					plgErrorRow(c, t, plgCopyConflict, plgCopyReload, func() {
						src.conflict = false
					})
				}
				if !src.loading && src.loadError == "" {
					gui.Column(c).Shrink(0).Gap(plgFieldGap).Children(func() {
						for i := range src.fields {
							plgFieldView(c, t, &src.fields[i])
						}
					})
				}
				plgCardActions(c, t, src)
			})
	})
}

// plgCardActions 是卡内的按钮行（S-pluginSettings-21/22/23/24）。
func plgCardActions(c *gui.Context, t *gui.Theme, src *pluginSource) {
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(plgBtnGap).Children(func() {
		label := plgCopySave
		if src.saving {
			label = plgCopySaving
		}
		save := gui.ButtonBase(c).Shrink(0).Height(plgBtnH).
			Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
			Label(label).Background(t.Accent).Disabled(true).Tooltip(notePluginWrite)
		if src.saving {
			save = save.Opacity(plgAlpha05)
		}
		save.Children(func() {
			plgBtnTextSty.style(t, gui.Text(c, label)).TextColor(t.AccentText).
				SingleLine()
		})
		// 保存与重新加载都要服务端的写入通道，本版没有：入口禁用并写原因
		// （DEC-12），不再走本屏的假回执。
		plgOutlineButtonDisabled(c, t, plgCopyReload, plgBtnH, 0, notePluginWrite)
		gui.Spacer(c)
		if src.saved {
			plgSavedStyle.style(t, gui.Text(c, plgCopySaved)).
				TextColor(t.Success).Label(plgCopySaved)
		}
	})
	if src.err != "" {
		plgErrStyle.style(t, gui.Text(c, src.err)).TextColor(t.Danger).
			Role(gui.RoleAlert).Label(src.err)
	}
}

// plgFieldView 按字段形状渲染（S-pluginSettings-20/25/26/27/28/29/30）。
func plgFieldView(c *gui.Context, t *gui.Theme, f *pluginField) {
	switch f.kind {
	case plgGroup:
		plgGroupView(c, t, f)
	case plgBool:
		plgBoolRow(c, t, f)
	case plgArray:
		gui.Column(c).Shrink(0).Gap(plgLabelGap).Children(func() {
			plgLabelLine(c, t, f.label, f.required)
			if f.desc != "" {
				plgNote(c, t, f.desc)
			}
			gui.Column(c).Shrink(0).Gap(plgLabelGap).Children(func() {
				for i := range f.items {
					plgArrayRow(c, t, f, i)
				}
			})
			gui.Row(c).Shrink(0).Children(func() {
				if plgOutlineButton(c, t, plgCopyAdd, plgBtnH, plgAddWidth) {
					f.items = append(f.items, "")
				}
			})
		})
	case plgSecret:
		gui.Column(c).Shrink(0).Gap(plgLabelGap).Children(func() {
			plgLabelLine(c, t, f.label, f.required)
			plgNote(c, t, plgCopyWithheld)
		})
	default:
		gui.Column(c).Shrink(0).Gap(plgLabelGap).Children(func() {
			plgLabelLine(c, t, f.label, f.required)
			if f.desc != "" {
				plgNote(c, t, f.desc)
			}
			input := gui.TextInput(c, &f.text).Height(plgCtlH).
				Padding(screenInputPadV, screenSpace25).Background(fieldFill(t)).
				Label(f.label)
			input.BorderWidth(strokeHairline).BorderColor(t.Border)
		})
	}
}

// plgGroupView 是一个对象字段（S-pluginSettings-30）：带边框、缩进 12 的组。
func plgGroupView(c *gui.Context, t *gui.Theme, f *pluginField) {
	group := gui.Column(c).Shrink(0).Padding(plgGroupPad, plgGroupPad).
		Gap(plgGroupGap).Radius(radiusCard).Background(t.Surface)
	group.BorderWidth(strokeHairline).BorderColor(t.Border)
	group.Children(func() {
		if f.label != "" {
			plgLabelLine(c, t, f.label, f.required)
		}
		for i := range f.fields {
			plgFieldView(c, t, &f.fields[i])
		}
	})
}

// plgBoolRow 是布尔字段（S-pluginSettings-27）：标签与开关同一行。
func plgBoolRow(c *gui.Context, t *gui.Theme, f *pluginField) {
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(plgLabelGap).Children(func() {
		plgLabelLine(c, t, f.label, f.required)
		if f.desc != "" {
			plgNote(c, t, f.desc)
		}
		gui.Spacer(c)
		gui.Switch(c, &f.on).Size(plgSwitchW, plgSwitchH).
			Radius(plgSwitchH / 2).Label(f.label)
	})
}

// plgArrayRow 是数组字段的一项（S-pluginSettings-29）：输入 + 移除。
func plgArrayRow(c *gui.Context, t *gui.Theme, f *pluginField, i int) {
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(plgLabelGap).Children(func() {
		input := gui.TextInput(c, &f.items[i]).Height(plgCtlH).Grow(1).
			Padding(screenInputPadV, screenSpace25).Background(fieldFill(t)).
			Label(f.label)
		input.BorderWidth(strokeHairline).BorderColor(t.Border)
		if accIconButton(c, accIconSpec{
			icon: iconXMark, label: plgCopyRemove, tip: plgCopyRemove,
			size: plgIconSize, hit: plgIconHit,
		}) {
			f.items = append(f.items[:i], f.items[i+1:]...)
		}
	})
}

// plgLabelLine 是字段标签（S-pluginSettings-25）：必填时在标签后加星号。
func plgLabelLine(c *gui.Context, t *gui.Theme, label string, required bool) {
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(plgHeadGap).Children(func() {
		plgLabelStyle.style(t, gui.Text(c, label)).TextColor(t.Text).Label(label)
		if required {
			plgLabelStyle.style(t, gui.Text(c, "*")).TextColor(t.Danger).
				Label("必填").AccessHidden(true)
		}
	})
}

// plgNote 是一句说明（11–12.25、次级前景、可复制）。
func plgNote(c *gui.Context, t *gui.Theme, text string) {
	typeSecondary.style(t, gui.Text(c, text)).TextColor(t.TextMuted).Selectable()
}

// plgErrorRow 是一条错误 + 行内重试（S-pluginSettings-05/18/19）。重新拉取要
// 服务端通道，本版不做：按钮置灰并写原因（DEC-12）。
func plgErrorRow(c *gui.Context, t *gui.Theme, text, action string, run func()) {
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Wrap().Gap(plgBtnGap).Children(func() {
		plgErrStyle.style(t, gui.Text(c, text)).TextColor(t.Danger).
			Role(gui.RoleAlert).Label(text)
		plgOutlineButtonDisabled(c, t, action, plgBtnH, 0, notePluginWrite)
	})
}

// plgOutlineButton 是一个描边小按钮（高度与最小宽由调用处给）。
func plgOutlineButton(c *gui.Context, t *gui.Theme, text string, height, minWidth float32) bool {
	b := gui.ButtonBase(c).Shrink(0).Height(height).MinWidth(minWidth).
		Padding(plgAddPadV, controlPadH).Radius(radius).Label(text).
		Background(t.Background)
	b.BorderWidth(strokeHairline).BorderColor(t.Border)
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		plgBtnTextSty.style(t, gui.Text(c, text)).TextColor(t.Text).SingleLine()
	})
	return b.Clicked()
}

// plgOutlineButtonDisabled 是同款描边按钮的禁用态：tooltip 写原因，点击不生效。
func plgOutlineButtonDisabled(c *gui.Context, t *gui.Theme, text string, height, minWidth float32, tip string) {
	b := gui.ButtonBase(c).Shrink(0).Height(height).MinWidth(minWidth).
		Padding(plgAddPadV, controlPadH).Radius(radius).Label(text).
		Background(t.Background).Tooltip(tip).Disabled(true)
	b.BorderWidth(strokeHairline).BorderColor(t.Border)
	b.Children(func() {
		plgBtnTextSty.style(t, gui.Text(c, text)).TextColor(t.Text).SingleLine()
	})
	b.Opacity(0.5)
}

// plgValidate 是本屏的保存校验：数字字段只收整数，其余字段原样。保存已按
// D-12 禁用，本函数只保留给将来的写入路径接线时复用。
func plgValidate(src *pluginSource) {
	src.err = ""
	src.saved = false
	if msg := plgCheckFields(src.fields); msg != "" {
		src.err = msg
		return
	}
	src.saving = true
	src.saving = false
	src.saved = true
}

// plgCheckFields 递归校验字段；返回空串表示通过。
func plgCheckFields(fields []pluginField) string {
	for i := range fields {
		f := &fields[i]
		switch f.kind {
		case plgInt:
			if f.text == "" {
				f.text = "0"
				continue
			}
			if _, err := strconv.Atoi(f.text); err != nil {
				return "「" + f.label + "」" + plgCopyNumberOnly
			}
		case plgGroup:
			if msg := plgCheckFields(f.fields); msg != "" {
				return msg
			}
		}
	}
	return ""
}

// plgReload 丢掉未保存的改动，回到最近一次读数（S-pluginSettings-22）。
func plgReload(src *pluginSource) {
	src.err = ""
	src.saved = false
	src.conflict = false
}

// plgAddWidth 是「添加」按钮的最小宽（参考 48）。
const plgAddWidth = 48
