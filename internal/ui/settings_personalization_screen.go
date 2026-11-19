package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 个性化的重建实现：结构、文案、控件与间距按
// docs/domains/requirements/inventory-settings-rt.md §3（S-personalization-01…99）
// 与 docs/domains/requirements/requirements-settings.md 的 R-PER-* 条目对齐，
// 视觉常量取 docs/domains/requirements/design-baseline.md。
//
// 分组顺序与参考一致：界面模式、展示 ACP 会话、默认打开目标应用、助手模型、
// 子代理模型、子代理总数限制、命令 Shell、提示词语言、自定义指令；分组之间是
// 1px 分隔线（png 实测 y=216.5 / 303.5 / 437.5 / 603 / 714）。
//
// # 状态
//
// 本屏没有来自事实的读写：设置项在 App Server 模式下的归属方都在服务端，
// protocol v3 目前没有这些写入口。屏内只保留**可丢的界面状态**（psUI）：
// 界面模式、展示 ACP 会话、默认打开目标应用、子代理模型、子代理上限。
// 接线时把这几个字段搬进 App 的界面状态即可，渲染函数不必改。
//
// # 可达分支
//
// 参考实现在 App Server 只读模式下可见 41 个元素（主体 38 / 图标 3），
// 本文件实现这 41 个元素加下拉选项文案；未实现的 58 个都带前置条件：
//
//	S-personalization-11/29/35/36/42…47/50/64  error、无效配置、失败候选项、
//	                 编码未知、运行中 Shell 不一致等错误支路：本客户端没有
//	                 这些事实来源（没有 Shell 探测、没有模型策略快照）。
//	S-personalization-13…17/38…40/58          下拉展开后的选项：见下。
//	S-personalization-65…68                  非 App Server 分支的自定义指令
//	                 （Textarea + 保存）：App Server 模式恒真，不可达。
//	S-personalization-69…99                  模型选择对话框（三步：模式 → 供应商 →
//	                 模型/思考强度）：子代理模型的写入没有 Intents 方法，
//	                 触发按钮只做本屏内的取值切换，不打开对话框。
//
// 下拉选项的落地差异：
//
//	默认打开目标应用（S-13…17）：参考从本机发现编辑器并给图标；本客户端不
//	     做编辑器发现，选项只有「系统默认」一项。
//	命令 Shell（S-38…40）：没有 Shell 探测快照，选项只有「自动（没有受支持的
//	     Shell）」一项，且触发按钮禁用（参考在无快照时同样禁用）。
//	提示词语言（S-58）：语言包取值在服务端，触发按钮禁用，只显示
//	     「App Server 模式下不可用」。
//	子代理模型（S-69…99）：三个模式文案照参考给（继承父会话 / 模型自主选择 /
//	     指定模型），「指定模型」需要模型选择器，标为禁用。
//
// 未接动作（没有对应 Intents 方法，按 D14「未接动作」不提供写入）：
//
//	界面模式切换、展示 ACP 会话、默认打开目标应用、子代理模型与上限、
//	命令 Shell 刷新、打开或创建 AGENTS.md。
//
// 连着 Source 时这些入口一律置灰并把原因写在组下（notePersonalizationWrite）：
// 拨得动却不落盘的开关等于一个假的现状。-demo 里照参考渲染，控件可点，只改
// 屏内状态。
func renderSettingsPersonalizationScreen(c *gui.Context, a *App) {
	t := c.Theme()
	// live 表示连着 Source：这一屏没有任何一项的读写通道，所以能点的控件全部
	// 置灰并把原因写在组下（拨了不生效的开关等于一个假的现状）。
	live := a != nil && a.wired()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(psPageGap).Children(func() {
				typeH2.style(t, gui.Text(c, psCopyTitle)).TextColor(t.Text).
					Role(gui.RoleHeading).Level(2)
				psUIModeSection(c, t, live)
				psRule(c, t)
				psShowACPSection(c, t, live)
				psRule(c, t)
				psDefaultAppSection(c, t, live)
				psRule(c, t)
				psAssistantModelSection(c, t)
				psRule(c, t)
				psSubagentModelSection(c, t, live)
				psRule(c, t)
				psSubagentCountSection(c, t)
				psRule(c, t)
				psExecShellSection(c, t, live)
				psRule(c, t)
				psPromptLocaleSection(c, t)
				psRule(c, t)
				psCustomInstructionsSection(c, t, live)
			})
		})
	})
}

// psNoFacts 是"这一组没有事实"的一行原因。
func psNoFacts(c *gui.Context, t *gui.Theme) {
	typeMicro.style(t, gui.Text(c, notePersonalizationWrite)).TextColor(t.TextMuted).
		Selectable()
}

// ---- 分组 ----

// psUIModeSection 是界面模式（S-personalization-02…05）：说明 + 两张等宽卡。
// 卡是单选项（RoleRadio），选中态是主色描边 + 主色 5% 底（参考
// border-primary bg-primary/5），未选中态悬停时描边与底色各让一档。
func psUIModeSection(c *gui.Context, t *gui.Theme, live bool) {
	gui.Column(c).Shrink(0).Gap(psGroupGap).Children(func() {
		psFieldHead(c, t, psCopyUIModeTitle, psCopyUIModeDesc)
		width, _ := c.Size()
		if width < psSplitW {
			// 窄窗：两卡竖排、撑满内容列（R-PER-038）。
			gui.Column(c).Shrink(0).Gap(psGroupGap).
				Role(gui.RoleRadioGroup).Label(psCopyUIModeTitle).Children(func() {
				psModeCard(c, t, psModeOffice, psCopyModeOffice, psCopyModeOfficeDesc, live)
				psModeCard(c, t, psModeCoding, psCopyModeCoding, psCopyModeCodingDesc, live)
			})
			if live {
				psNoFacts(c, t)
			}
			return
		}
		gui.Row(c).Shrink(0).Gap(psGroupGap).
			Role(gui.RoleRadioGroup).Label(psCopyUIModeTitle).Children(func() {
			psModeCard(c, t, psModeOffice, psCopyModeOffice, psCopyModeOfficeDesc, live)
			psModeCard(c, t, psModeCoding, psCopyModeCoding, psCopyModeCodingDesc, live)
		})
		if live {
			psNoFacts(c, t)
		}
	})
}

// psModeCard 是界面模式的一张卡（S-personalization-04/05）：标题 + 一行说明，
// 整卡可点、可聚焦，读屏按单选项播报（名称 + 是否选中）。
func psModeCard(c *gui.Context, t *gui.Theme, mode, label, desc string, live bool) {
	selected := psUI.uiMode == mode
	card := gui.ButtonBase(c).Shrink(0).Grow(1).MinHeight(psModeCardH).
		Padding(psModePad).Radius(radius).AlignItems(gui.Center).
		Role(gui.RoleRadio).AccessSelected(selected).Label(label).
		BorderWidth(strokeHairline).BorderColor(t.Border).Disabled(live)
	if live {
		// 连着 Source：选中态不再是"服务端的口径"，只说明它是本屏的草稿。
		card = card.Opacity(psAlpha50)
	}
	switch {
	case selected:
		card = card.BorderColor(t.Accent).Background(t.Accent.Alpha(psAlpha05))
	case card.Hovered():
		card = card.BorderColor(t.Accent.Alpha(psAlpha50)).
			Background(t.SurfaceHover.Alpha(psAlpha40))
	}
	card.Children(func() {
		gui.Column(c).Grow(1).Gap(psTightGap).Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, label)).TextColor(t.Text).SingleLine()
			typeMicro.style(t, gui.Text(c, desc)).TextColor(t.TextMuted)
		})
	})
	if card.Clicked() && !live {
		psUI.uiMode = mode
	}
}

// psShowACPSection 是「展示 ACP 会话」（S-personalization-06…08）：左标签 +
// 说明，右开关。
func psShowACPSection(c *gui.Context, t *gui.Theme, live bool) {
	gui.Row(c).Shrink(0).Gap(psLooseGap).AlignItems(gui.Center).
		Justify(gui.SpaceBetween).Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Gap(psTightGap).Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, psCopyShowACP)).TextColor(t.Text)
			typeMicro.style(t, gui.Text(c, psCopyShowACPDesc)).TextColor(t.TextMuted)
			if live {
				psNoFacts(c, t)
				return
			}
		})
		if !live {
			gui.Switch(c, &psUI.showACP).Size(psSwitchW, psSwitchH).Label(psCopyShowACP)
		}
	})
}

// psDefaultAppSection 是「默认打开目标应用」（S-personalization-09…17）：
// 标签 + 说明 + 宽 280 的下拉。
func psDefaultAppSection(c *gui.Context, t *gui.Theme, live bool) {
	gui.Column(c).Shrink(0).Gap(psGroupGap).Children(func() {
		psFieldHead(c, t, psCopyDefaultAppTitle, psCopyDefaultAppDesc)
		gui.Select(c, &psUI.defaultApp, psDefaultAppOptions).
			Width(psSelectXs).Label(psCopyDefaultAppTitle).Disabled(live)
		if live {
			psNoFacts(c, t)
		}
	})
}

// psAssistantModelSection 是「助手模型」（S-personalization-18…21）：说明 +
// 禁用的触发器 + 一段解释。App Server 模式没有该设置的归属方。
func psAssistantModelSection(c *gui.Context, t *gui.Theme) {
	gui.Column(c).Shrink(0).Gap(psRowGap).Children(func() {
		gui.Column(c).Shrink(0).Gap(psTightGap).Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, psCopyAssistantTitle)).TextColor(t.Text)
			typeMicro.style(t, gui.Text(c, psCopyAssistantDesc)).TextColor(t.TextMuted)
			psTrigger(c, t, psTriggerSpec{
				label:    psCopyAssistantTitle,
				value:    psCopyUnavailable,
				icon:     psIconArrowUpFromLine,
				tip:      psCopyAssistantUnavailableDesc,
				disabled: true,
			})
		})
		typeMicro.style(t, gui.Text(c, psCopyAssistantUnavailableDesc)).TextColor(t.TextMuted)
	})
}

// psSubagentModelSection 是「子代理模型」（S-personalization-22…26）：
// 标签 + 触发器。触发器按参考把图标放在取值左边、箭头放在右边；展开是本屏
// 内的三个模式取值（模型选择对话框见文件头）。
func psSubagentModelSection(c *gui.Context, t *gui.Theme, live bool) {
	gui.Column(c).Shrink(0).Gap(psTightGap).Children(func() {
		typeSecondaryStrong.style(t, gui.Text(c, psCopySubagentTitle)).TextColor(t.Text)
		// 触发器与弹层都建在本列的子树里：节点在创建时就挂到当前父节点上，
		// 先建外层再挂子节点，上屏顺序才与参考一致。
		sel := gui.SelectBase(c, &psUI.subagentModel)
		trigger := sel.Trigger.Shrink(0).Height(psBtnH).Padding(controlPadV, psPadBtnX).
			Radius(radius).AlignItems(gui.Center).Label(psCopySubagentTitle).
			BorderWidth(strokeHairline).BorderColor(t.Border).Disabled(live)
		if trigger.Hovered() {
			trigger = trigger.Background(t.SurfaceHover)
		}
		trigger.Children(func() {
			gui.Row(c).Grow(1).Gap(psRowGap).AlignItems(gui.Center).Children(func() {
				psTriggerIcon(c, t, psIconArrowUpFromLine, "arrow-up-from-line")
				typeSecondary.style(t, gui.Text(c, psUI.subagentModel)).TextColor(t.Text).
					SingleLine().Grow(1)
				psTriggerIcon(c, t, psIconChevronDown, "chevron-down")
			})
		})
		if live {
			psNoFacts(c, t)
			return
		}
		sel.Popup(func(panel gui.Element) {
			panel.Padding(space1).Radius(radius).Background(t.Surface).
				BorderWidth(strokeHairline).BorderColor(t.Border)
			for _, option := range psSubagentOptions {
				item := sel.Item(option).Padding(space1, space2).Radius(radius)
				ink := t.Text
				switch {
				case option == psCopySubagentSpecific:
					// 需要模型选择器（S-personalization-69…99）：标为禁用。
					item = item.Disabled(true)
					ink = t.TextMuted
				case item.Highlighted():
					item = item.Background(t.Accent)
					ink = t.AccentText
				case option == psUI.subagentModel:
					item = item.Background(t.SurfaceHover)
				}
				item.Children(func() {
					typeSecondary.style(t, gui.Text(c, option)).TextColor(ink).SingleLine()
				})
			}
		})
	})
}

// psSubagentCountSection 是「子代理总数限制」（S-personalization-27…32）：
// 标签 + 右侧取值「不可配置」+ 禁用的滑块 + 一行说明。
func psSubagentCountSection(c *gui.Context, t *gui.Theme) {
	gui.Column(c).Shrink(0).Gap(psGroupGap).Children(func() {
		gui.Row(c).Shrink(0).Gap(psGroupGap).AlignItems(gui.Center).
			Justify(gui.SpaceBetween).Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, psCopySubagentCountTitle)).TextColor(t.Text)
			typeSecondary.style(t, gui.Text(c, psCopySubagentCountUnavailable)).
				TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
		})
		gui.Slider(c, &psUI.subagentLimit, psSubagentMin, psSubagentMax).Disabled(true).
			Label(psCopySubagentCountTitle)
		typeMicro.style(t, gui.Text(c, psCopySubagentCountDesc)).TextColor(t.TextMuted)
	})
}

// psExecShellSection 是「命令 Shell」（S-personalization-33…50 的可见支）：
// 说明 + 禁用的下拉 + 刷新 + 两段说明。
func psExecShellSection(c *gui.Context, t *gui.Theme, live bool) {
	gui.Column(c).Shrink(0).Gap(psGroupGap).Children(func() {
		psFieldHead(c, t, psCopyShellTitle, psCopyShellDesc)
		gui.Select(c, &psUI.shell, psShellOptions).
			Width(psSelectXl).Label(psCopyShellTitle).Disabled(true)
		gui.Row(c).Shrink(0).Gap(psRowGap).AlignItems(gui.Center).Children(func() {
			// 刷新：没有 Shell 探测接口（未接动作），点击不写事实。
			tip := psCopyRefresh
			if live {
				tip = notePersonalizationWrite
			}
			accButton(c, accButtonSpec{
				variant: accButtonOutline, height: psBtnHSm, label: psCopyRefresh,
				tip: tip, disabled: live,
				children: func() {
					typeControl.style(t, gui.Text(c, psCopyRefresh)).TextColor(t.Text).
						SingleLine()
				},
			})
			gui.Spacer(c)
		})
		typeMicro.style(t, gui.Text(c, psCopyShellScope)).TextColor(t.TextMuted)
		typeMicro.style(t, gui.Text(c, psCopyShellCache)).TextColor(t.TextMuted)
	})
}

// psPromptLocaleSection 是「提示词语言」（S-personalization-51…57）：说明 +
// Owner 缺口两个引用 + 禁用的下拉。
func psPromptLocaleSection(c *gui.Context, t *gui.Theme) {
	gui.Column(c).Shrink(0).Gap(psGroupGap).Children(func() {
		gui.Column(c).Shrink(0).Gap(psTightGap).Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, psCopyPromptLocaleTitle)).TextColor(t.Text)
			typeMicro.style(t, gui.Text(c, psCopyPromptLocaleDesc)).TextColor(t.TextMuted)
			psOwnerGapLine(c, t)
		})
		gui.Select(c, &psUI.promptLocale, psPromptLocaleOptions).
			Width(psSelectXl).Label(psCopyPromptLocaleTitle).Disabled(true)
	})
}

// psOwnerGapLine 是「Owner 缺口：<引用> · <引用>」一行：引用取强调色作文字，
// 分隔点对读屏隐藏（参考 aria-hidden）。地址没有登记，故不接外开。
func psOwnerGapLine(c *gui.Context, t *gui.Theme) {
	gui.Row(c).Shrink(0).Wrap().GapX(psTightGap).AlignItems(gui.Center).Children(func() {
		typeMicro.style(t, gui.Text(c, psCopyPromptLocaleGap)).TextColor(t.TextMuted)
		for i, ref := range psPromptLocaleRefs {
			if i > 0 {
				typeMicro.style(t, gui.Text(c, psCopyDot)).TextColor(t.TextMuted).
					AccessHidden(true)
			}
			typeMicro.style(t, gui.Text(c, ref)).TextColor(colorAccentInk.in(t))
		}
	})
}

// psCustomInstructionsSection 是「自定义指令」（S-personalization-59…64）：
// 标题 + 说明 + 文件行（路径 + 打开按钮）+ 一行说明。
func psCustomInstructionsSection(c *gui.Context, t *gui.Theme, live bool) {
	gui.Column(c).Shrink(0).Gap(psLooseGap).Children(func() {
		gui.Column(c).Shrink(0).Gap(psTightGap).Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, psCopyInstructionsTitle)).
				TextColor(t.Text).Role(gui.RoleHeading).Level(3)
			typeSecondary.style(t, gui.Text(c, psCopyInstructionsDesc)).
				TextColor(t.TextMuted).MaxWidth(psProseMax)
		})
		gui.Column(c).Shrink(0).Gap(psGroupGap).Children(func() {
			row := gui.Row(c).Shrink(0).Wrap().GapX(space3).GapY(space2).
				AlignItems(gui.Center).Padding(psFilePadY, psFilePadX).
				Radius(radius).Background(t.Surface).
				Role(gui.RoleGroup).Label(psCopyAgentsPath).
				BorderWidth(strokeHairline).BorderColor(t.Border)
			row.Children(func() {
				gui.Text(c, psCopyAgentsPath).Font(codeFont).
					FontSize(typeCode.sizeIn(t)).TextColor(t.Text).
					SingleLine().Grow(1).MinWidth(0)
				// 打开或创建文件：没有打开本地文件的接口（未接动作）。
				b := gui.PrimaryButton(c, psCopyOpenAgentsFile).Label(psCopyOpenAgentsFile)
				if live {
					b = b.Disabled(true).Tooltip(notePersonalizationWrite)
				}
			})
			typeMicro.style(t, gui.Text(c, psCopyAgentsFileNote)).
				TextColor(t.TextMuted).MaxWidth(psProseMax)
		})
	})
}

// ---- 组件 ----

// psFieldHead 是一组的标签 + 一行说明（参考 space-y-1）。
func psFieldHead(c *gui.Context, t *gui.Theme, label, desc string) {
	gui.Column(c).Shrink(0).Gap(psTightGap).Children(func() {
		typeSecondaryStrong.style(t, gui.Text(c, label)).TextColor(t.Text)
		if desc != "" {
			typeMicro.style(t, gui.Text(c, desc)).TextColor(t.TextMuted)
		}
	})
}

// psRule 是分组之间的 1px 分隔线：取主题边框色，不占高度以外的空间。
func psRule(c *gui.Context, t *gui.Theme) {
	gui.Box(c).Shrink(0).Height(strokeHairline).Background(t.Border)
}

// psTriggerSpec 是整行宽的设置触发器（参考的 w-full h-9 outline 按钮）。
type psTriggerSpec struct {
	label    string
	value    string
	tip      string
	icon     *gui.SVG
	disabled bool
}

// psTrigger 画一个整行宽的触发器（非下拉：本屏里它只表达取值）。
// 五态与 accButton 一致：默认 / 悬停 / 按下 / 聚焦（框架焦点环）/ 禁用。
func psTrigger(c *gui.Context, t *gui.Theme, spec psTriggerSpec) bool {
	b := gui.ButtonBase(c).Shrink(0).Height(psBtnH).Padding(controlPadV, psPadBtnX).
		Radius(radius).AlignItems(gui.Center).Label(spec.label).Disabled(spec.disabled).
		BorderWidth(strokeHairline).BorderColor(t.Border)
	if spec.tip != "" {
		b = b.Tooltip(spec.tip)
	}
	ink := t.Text
	switch {
	case spec.disabled:
		b = b.Opacity(psAlpha50)
		ink = t.TextMuted
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		gui.Row(c).Grow(1).Gap(psRowGap).AlignItems(gui.Center).Children(func() {
			if spec.icon != nil {
				psTriggerIcon(c, t, spec.icon, "trigger-icon")
			}
			typeSecondary.style(t, gui.Text(c, spec.value)).TextColor(ink).
				SingleLine().Grow(1)
			psTriggerIcon(c, t, psIconChevronDown, "chevron-down")
		})
	})
	return b.Clicked()
}

// psTriggerIcon 是触发器里的 14px 图标：次文字色，不单独播报。
func psTriggerIcon(c *gui.Context, t *gui.Theme, icon *gui.SVG, mark string) {
	accIcon(c, icon, psIcoInline, t.TextMuted, mark)
}

// ---- 状态 ----

// psUIState 是本屏可丢的界面状态：只影响外观，不落盘、不写事实。
// 接线时搬进 App 的界面状态字段即可。
type psUIState struct {
	uiMode        string
	showACP       bool
	defaultApp    string
	subagentModel string
	subagentLimit float64
	shell         string
	promptLocale  string
}

var psUI = psUIState{
	uiMode:        psModeCoding,
	defaultApp:    psCopyAppSystem,
	subagentModel: psCopySubagentInherit,
	subagentLimit: psSubagentMax,
	shell:         psCopyShellNone,
	promptLocale:  psCopyUnavailable,
}

// 界面模式的取值（参考 ui-mode store 的 simple / pro）。
const (
	psModeOffice = "office"
	psModeCoding = "coding"
)

// 子代理上限的取值域（参考 MIN_SUBAGENT_COUNT / MAX_SUBAGENT_COUNT）。
const (
	psSubagentMin = 1
	psSubagentMax = 100
)

// psSubagentOptions / psDefaultAppOptions / psShellOptions / psPromptLocaleOptions
// 是各下拉的取值与文案。选项文案逐字取参考的 zh-CN。
var (
	psSubagentOptions     = []string{psCopySubagentInherit, psCopySubagentAgentDecides, psCopySubagentSpecific}
	psDefaultAppOptions   = []string{psCopyAppSystem}
	psShellOptions        = []string{psCopyShellNone}
	psPromptLocaleOptions = []string{psCopyUnavailable}
	psPromptLocaleRefs    = []string{"arcships/dim#1018", "arcships/dim#548"}
)

// 本屏图标：Heroicons v2 outline（24 网格、1.5 描边）的内联路径，与 icons.go
// 同一套约定（mygo 不带图标集）。警告三角复用设置 › 账户的内联路径。
var (
	psIconArrowUpFromLine = mustSVG(`<path d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5m-13.5-9L12 3m0 0 4.5 4.5M12 3v13.5"/>`)
	psIconChevronDown     = mustSVG(`<path d="m19.5 8.25-7.5 7.5-7.5-7.5"/>`)
)

// 本屏几何：取参考实现的 Tailwind 确定值，换算成 design-baseline §1.1 的
// 3.5 网格档位。圆角统一取本仓库的 Theme.Radius（6；参考 rounded-lg 8.75，
// 与设置 › 账户同一条 D-02 待裁决），颜色取主题 token。
const (
	psPageGap   = 28   // space-y-8：分组（含分隔线）之间
	psGroupGap  = 10.5 // space-y-3 / gap-3：组内块之间
	psLooseGap  = 14   // space-y-4：自定义指令组的块之间
	psRowGap    = 7    // space-y-2 / gap-2：说明与控件之间、图标与取值之间
	psTightGap  = 3.5  // space-y-1：标签与说明之间
	psModePad   = 10.5 // p-3：界面模式卡内边距
	psModeCardH = 58   // 参考实测 365×58
	psBtnH      = 32   // h-9：下拉与触发器
	psBtnHSm    = 28   // h-8 / size="sm"：刷新
	psPadBtnX   = 10.5 // px-3：触发器左右内边距
	psFilePadX  = 14   // px-4：文件行左右内边距
	psFilePadY  = 10.5 // py-3：文件行上下内边距
	psSelectXs  = 280  // max-w-xs：默认打开目标应用
	psSelectXl  = 504  // max-w-xl：命令 Shell、提示词语言
	psSwitchW   = 28   // 开关（design-baseline §3）
	psSwitchH   = 16
	psIcoInline = 14   // h-3.5 w-3.5：触发器内图标
	psProseMax  = 672  // max-w-3xl：说明段与注释的排版上限
	psSplitW    = 900  // 两卡并排的窗口宽下限（窄于此竖排）
	psAlpha05   = 0.05 // 选中卡底色
	psAlpha40   = 0.40 // 悬停卡底色
	psAlpha50   = 0.50 // 禁用整棵子树、悬停描边
)

// 本屏文案：逐字取参考产品的 zh-CN（i18n:personalizationSettings.*、
// i18n:modeSelection.*、i18n:customInstructionsSettings.*）。
const (
	psCopyTitle = "个性化"

	psCopyUIModeTitle    = "界面模式"
	psCopyUIModeDesc     = "办公模式隐藏开发细节、以结果为先；编程模式展示完整开发界面。"
	psCopyModeOffice     = "办公模式"
	psCopyModeOfficeDesc = "更专注于文档、表格、设计等日常办公，把事做好就行。"
	psCopyModeCoding     = "编程模式"
	psCopyModeCodingDesc = "面向写代码和工程开发，diff、终端、分支等尽在掌控。"

	psCopyShowACP     = "展示 ACP 会话"
	psCopyShowACPDesc = "显示 ACP 创建的会话及其工作区。"

	psCopyDefaultAppTitle = "默认打开目标应用"
	psCopyDefaultAppDesc  = "选择在外部编辑器中打开文件时使用的应用"
	psCopyAppSystem       = "系统默认"

	psCopyAssistantTitle           = "助手模型"
	psCopyAssistantDesc            = "用于工具自动安全审计等轻量工作。"
	psCopyUnavailable              = "App Server 模式下不可用"
	psCopyAssistantUnavailableDesc = "助手模型是用于轻量工作（例如代为审批审核）的辅助模型。" +
		"App Server 模式没有该设置的归属方，因此此控件不可用。新对话的默认模型请在模型选择器中选择。"

	psCopySubagentTitle        = "子代理模型"
	psCopySubagentInherit      = "继承父会话"
	psCopySubagentAgentDecides = "模型自主选择"
	psCopySubagentSpecific     = "指定模型"

	psCopySubagentCountTitle       = "子代理总数限制"
	psCopySubagentCountUnavailable = "不可配置"
	psCopySubagentCountDesc        = "App Server 模式没有成员并发设置，因此 Desktop 不会应用子代理上限。"

	psCopyShellTitle = "命令 Shell"
	psCopyShellDesc  = "选择 Agent 内置 exec 工具使用的 Shell。"
	// psCopyShellNone 是选择器取值：没有可用的已保存 Shell 时，参考显示这句。
	psCopyShellNone  = "已保存的 Shell 不可用："
	psCopyRefresh    = "刷新"
	psCopyShellScope = "TUI 和 Desktop 共享此设置。新建或重新加载的任务使用它；当前任务保持原 Shell。"
	psCopyShellCache = "已有任务以其他 Shell 重新加载时，可能降低提示词缓存复用率并消耗额外 token。"

	psCopyPromptLocaleTitle = "提示词语言"
	psCopyPromptLocaleDesc  = "提示词语言已禁用：App Server 会话还没有语言载体，Agent 无法接收此设置。" +
		"Desktop 不会写入本地提示词语言文件。"
	psCopyPromptLocaleGap = "Owner 缺口："

	psCopyInstructionsTitle = "自定义指令"
	psCopyInstructionsDesc  = "App Server 在分支加载时从下面的文件读取你的全局自定义指令，并定期重新检查。" +
		"Desktop 不保存其副本。"
	psCopyAgentsPath     = "~/AGENTS.md"
	psCopyOpenAgentsFile = "打开或创建文件"
	psCopyAgentsFileNote = "PromptSources 会读取此 AGENTS.md 文件。请在编辑器中修改；更改将在新的运行中生效。"
	psCopyDot            = "·"
)
