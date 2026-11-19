package ui

import (
	"fmt"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 多模态的重建实现：元素按
// docs/domains/requirements/inventory-settings-rt.md §8（S-modality-01…70，
// 70 个元素）逐行对应，文案逐字取参考产品中文，数值取
// docs/domains/requirements/design-baseline.md（页签高 72、能力选择器 588×28、
// 表单卡 663、输入框高 32、分节间距 3.5 的倍数）。
//
// 结构：页首 → 不可用提示条 → 五个能力分组页签 → 当前分组面板（分组标题 +
// 计数 + 能力行）→ 任务（提交表单 + 所有者任务）→ 实时会话（两条提示 +
// 表单）。分节之间是 1px 细线（png 实测 y=578 / 1229）。
//
// # 状态
//
// 两档：连着 Source 时（mdLiveScreen）只按事实画——能力横幅读
// ModalitySnapshot（Known / Supported / Reason），任务读 ModalityTasks；服务端
// 没有分组事实，就不画参考里的「0 / N」计数与能力名；实时会话分节没有实时能力
// 事实，整块不画。-demo（没有 Source）保留参考的「未声明」样例态：提示条常显，
// 表单控件按参考渲染但禁用，任务列表为空。
// 屏内只保留可丢的界面状态（modalityUI）：当前分组、状态筛选与表单草稿。
//
// # 可达分支
//
// 参考实现在该状态下可见 40 个元素（主体 37 / 容器 1 / 图标 2），本文件实现
// 这 40 个元素；未实现的 30 个都带前置条件：
//
//	S-modality-02              加载中的转圈：App Server 拒答时没有加载态。
//	S-modality-04              errorText：本屏没有事实来源，不编造错误。
//	S-modality-18/19           已选模型与禁用原因：需要 owner 声明的模型清单。
//	S-modality-20…26           模型选择弹层（搜索、提供商分组、模型行、空态）：
//	                 同上，没有 provider/model 清单就无从列出，故不弹出。
//	S-modality-39/40           提交中的转圈与 submitError：提交方法未声明。
//	S-modality-44/46/47        任务列表错误、任务行与「加载更多」：列表恒空。
//	S-modality-48…54           任务详情（指标、等待超时、等待按钮）：未选中任务。
//	S-modality-58              实时操作清单为空时的提示：当前走「方法未声明」。
//	S-modality-66…70           会话行、结束原因、直连提示与帧发送区：未开会话。
//
// S-modality-11（窄屏分组下拉）已实现：内容列窄于 620 时页签换成通栏下拉，
// 选项文案带计数后缀。
//
// 五个分组的固有上限取自参考的取值（4 / 3 / 4 / 3 / 3）；能力名只有图像组
// 在清单里出现（图像生成 / 图像编辑 / 图像增强 / 图像理解），其余四组的
// 能力名不在任何需求文档里，故只显示标题与计数，不编造行名。
func renderModality(c *gui.Context, a *App) {
	t := c.Theme()
	if a != nil && a.wired() {
		st, _ := a.liveSettings()
		mdLiveScreen(c, t, st)
		return
	}
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Children(func() {
				typePageTitle.style(t, gui.Text(c, mdCopyTitle)).TextColor(t.Text).
					Role(gui.RoleHeading).Level(1).SingleLine()
				mdAlertBanner(c, t, mdCopyUnavailable).
					Margin(mdPageGap, 0, 0, 0)
				mdGroupsBlock(c, t).Margin(mdPageGap, 0, 0, 0)
				mdTaskSection(c, t)
				mdRealtimeSection(c, t)
			})
		})
	})
}

// ---- 连着 Source 时的这一屏 ----
//
// 三件事只按事实画：
//
//   - 能力横幅：只读 ModalitySupported / Known。服务端声明了能力就不画横幅；
//     读不到就说读不到；声明了但不可用就把服务端给的原因写出来。
//   - 任务：读 Settings().ModalityTasks。读到了就列任务，真的是空才写「没有任务」，
//     读失败就把原因写出来。
//   - 分组与能力清单：服务端没有分组事实，所以不画样例的「0 / 4」计数与能力名，
//     只写一句说明。
//
// 实时会话分节整块不画：协议里没有实时方法，也没有实时能力事实；上一版在这一屏
// 恒写「未声明实时方法」，那是一句无法核对的话。
func mdLiveScreen(c *gui.Context, t *gui.Theme, st SettingsSnapshot) {
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Children(func() {
				typePageTitle.style(t, gui.Text(c, mdCopyTitle)).TextColor(t.Text).
					Role(gui.RoleHeading).Level(1).SingleLine()
				if banner := mdCapabilityBanner(st.Modality); banner != "" {
					mdAlertBanner(c, t, banner).Margin(mdPageGap, 0, 0, 0)
				}
				typeSecondary.style(t, gui.Text(c, noteModalityGroups)).
					TextColor(t.TextMuted).Selectable().Margin(mdPageGap, 0, 0, 0)
				mdLiveTaskSection(c, t, st)
			})
		})
	})
}

// mdCapabilityBanner 是能力横幅的文案：读到且可用时为空（不画横幅）。
func mdCapabilityBanner(m ModalitySnapshot) string {
	switch {
	case !m.Known:
		return mdCopyCapabilityUnread
	case m.Supported:
		return ""
	case strings.TrimSpace(m.Reason) != "":
		return mdCopyUnsupported + strings.TrimSpace(m.Reason)
	default:
		return mdCopyUnsupported
	}
}

// mdLiveTaskSection 是连上事实时的「任务」分节：说明、提示（读不到 / 读失败）、
// 可用时才出现的提交表单，以及所有者任务清单。
func mdLiveTaskSection(c *gui.Context, t *gui.Theme, st SettingsSnapshot) {
	section := gui.Column(c).Shrink(0).Margin(mdSectionGap, 0, 0, 0)
	section.Children(func() {
		screenRule(c)
		gui.Column(c).Shrink(0).Padding(mdSectionTop, 0, 0, 0).Children(func() {
			typeH4.style(t, gui.Text(c, mdCopyTasks)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(2).SingleLine()
			mdSecondary(c, t, mdCopyTasksDesc).
				Selectable().Margin(mdHeadingGap, 0, 0, 0)
			switch {
			case !st.Modality.TaskKnown:
				mdAlertBanner(c, t, mdCopyTasksUnread).
					Margin(mdSectionInlineGap, 0, 0, 0)
			case st.TasksError != "":
				mdAlertBanner(c, t, mdCopyTasksFailed+st.TasksError).
					Margin(mdSectionInlineGap, 0, 0, 0)
			case st.Modality.TaskSupported:
				// 方法已声明，但本版没有提交通道：表单照渲染并禁用，原因写在下面。
				mdTaskFormCard(c, t).Margin(mdCardGap, 0, 0, 0)
				typeMicro.style(t, gui.Text(c, noteModalitySubmit)).TextColor(t.TextMuted).
					Selectable().Margin(mdSectionInlineGap, 0, 0, 0)
			}
			mdLiveTaskListCard(c, t, st).Margin(mdCardGap, 0, 0, 0)
		})
	})
}

// mdLiveTaskListCard 是所有者任务卡（同 S-modality-41…47）：标题 + 禁用的刷新 +
// 清单。清单只画事实里的任务；读不到 / 读失败时上面已经有提示，这里不再重复写
// 「没有任务」——那是一句会被当成事实的空话。
func mdLiveTaskListCard(c *gui.Context, t *gui.Theme, st SettingsSnapshot) gui.Element {
	card := mdCard(c)
	card.Children(func() {
		gui.Column(c).Gap(mdFieldGap).Children(func() {
			gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(mdRowGap).Children(func() {
				typeSecondaryStrong.style(t, gui.Text(c, mdCopyOwnedTasks)).
					TextColor(t.Text).SingleLine()
				gui.Spacer(c)
				mdIconButton(c, t, accIconRefresh, mdCopyRefresh, mdCopyRefresh, true)
			})
			if !st.Modality.TaskKnown || st.TasksError != "" {
				return
			}
			if len(st.ModalityTasks) == 0 {
				typeSecondary.style(t, gui.Text(c, mdCopyNoTasks)).
					TextColor(t.TextMuted).Selectable()
				return
			}
			gui.Column(c).Shrink(0).Children(func() {
				for i, task := range st.ModalityTasks {
					if i > 0 {
						screenRule(c)
					}
					mdLiveTaskRow(c, t, task)
				}
			})
		})
	})
	return card
}

// mdLiveTaskRow 是一条任务：标识（等宽）、类别 / 操作、状态与进度。
func mdLiveTaskRow(c *gui.Context, t *gui.Theme, task ModalityTaskSnapshot) {
	label := task.ID
	if label == "" {
		label = "任务"
	}
	gui.Row(c).Shrink(0).MinHeight(mdFieldH).AlignItems(gui.Center).Gap(mdRowGap).
		Label("任务：" + label).Children(func() {
		gitCodeStyle.style(t, gui.Text(c, label)).TextColor(t.Text).SingleLine()
		kind := strings.TrimSpace(task.Category + " · " + task.Operation)
		if kind != "·" && kind != "" {
			typeMicro.style(t, gui.Text(c, kind)).TextColor(t.TextMuted).SingleLine()
		}
		gui.Spacer(c)
		if task.Progress != "" {
			typeMicro.style(t, gui.Text(c, task.Progress)).TextColor(t.TextMuted).
				SingleLine()
		}
		if task.State != "" {
			typeSecondary.style(t, gui.Text(c, task.State)).TextColor(t.TextMuted).
				Role(gui.RoleStatus).Label(task.State)
		}
	})
}

// ---- 能力分组 ----

// modalityGroup 是页签条里的一组：名称、组内能力名（只有图像组有据可查）
// 与该组的能力上限（参考的 0 / N 里的 N）。
type modalityGroup struct {
	label        string
	capabilities []string
	total        int
}

var modalityGroups = []modalityGroup{
	{label: mdCopyGroupImage, capabilities: []string{
		mdCopyCapabilityGenerate, mdCopyCapabilityEdit,
		mdCopyCapabilityEnhance, mdCopyCapabilityUnderstand,
	}, total: 4},
	{label: mdCopyGroupVideo, total: 3},
	{label: mdCopyGroupAudio, total: 4},
	{label: mdCopyGroupCreative, total: 3},
	{label: mdCopyGroupRealtime, total: 3},
}

// mdGroupCount 是分组的计数口径「0 / N」：App Server 未声明任何能力，已配置
// 数恒为 0；两处（页签与面板标题行）同源，避免两处对不上。
func mdGroupCount(g modalityGroup) string {
	return fmt.Sprintf("%d / %d", 0, g.total)
}

// mdGroupsBlock 是页签条 + 当前分组面板：两者之间 21。
func mdGroupsBlock(c *gui.Context, t *gui.Theme) gui.Element {
	block := gui.Column(c).Shrink(0).Gap(mdTabsGap)
	block.Children(func() {
		mdGroupTabs(c, t)
		mdCapabilityPanel(c, t)
	})
	return block
}

// mdGroupTabs 是能力分组页签条（S-modality-05…10）或窄屏下的分组下拉
// （S-modality-11）。页签等宽，选中态是底部的 2px 主色条；未选中是 1px 边框色
// 的底线，两者合起来就是参考的标签栏下沿。
func mdGroupTabs(c *gui.Context, t *gui.Theme) {
	if width, _ := c.Size(); width < mdCompactW {
		screenIndexSelect(c, &modalityUI.group, mdGroupOptions(), mdCopyTitle)
		return
	}
	parts := gui.TabsBase(c, &modalityUI.group, len(modalityGroups))
	parts.List.Grow(1).AlignItems(gui.Stretch).Label(mdCopyTitle).Children(func() {
		for i, group := range modalityGroups {
			selected := i == modalityUI.group
			tab := parts.Tab(i).Grow(1).Shrink(1).Height(mdTabH).
				Padding(mdTabPadTop, mdTabPadX, 0, mdTabPadX)
			if tab.Hovered() {
				tab = tab.Background(t.SurfaceHover)
			}
			tab = tab.DrawOver(func(p *gui.Painter, r gui.Rect) {
				height, ink := float32(mdTabRule), t.Border
				if selected {
					height, ink = mdTabRuleSelected, t.Accent
				}
				p.Fill(gui.Rect{X: r.X, Y: r.Y + r.H - height, W: r.W, H: height}, ink, 0)
			})
			tab.Children(func() {
				gui.Column(c).Grow(1).Children(func() {
					typeSecondaryStrong.style(t, gui.Text(c, group.label)).
						TextColor(t.Text).MinHeight(mdTabLabelH).SingleLine()
					typeMicro.style(t, gui.Text(c, mdGroupCount(group))).
						TextColor(t.TextMuted).FontFeatures("tnum").SingleLine().
						Margin(mdTabCountGap, 0, 0, 0)
				})
			})
		}
	})
}

// mdGroupOptions 是窄屏下拉的取值文案：「图像 · 0 / 4」「视频 · 0 / 3」…。
func mdGroupOptions() []string {
	options := make([]string, len(modalityGroups))
	for i, group := range modalityGroups {
		options[i] = group.label + " · " + mdGroupCount(group)
	}
	return options
}

// mdCapabilityPanel 是当前分组的面板（S-modality-12…17）：标题 + 计数 + 能力行。
// 面板本身没有边框，与页签条同底色（参考的 tabpanel 无卡片）。
func mdCapabilityPanel(c *gui.Context, t *gui.Theme) {
	group := modalityGroups[modalityUI.group]
	panel := gui.Column(c).Shrink(0).Gap(mdPanelGap).
		Role(gui.RoleGroup).Label(group.label)
	panel.Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Children(func() {
			typeH4.style(t, gui.Text(c, group.label)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(2).SingleLine()
			gui.Spacer(c)
			typeMicro.style(t, gui.Text(c, mdGroupCount(group))).
				TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
		})
		gui.Column(c).Shrink(0).Gap(mdCapabilityGap).Children(func() {
			for _, capability := range group.capabilities {
				mdCapabilityRow(c, t, capability)
			}
		})
	})
}

// mdCapabilityRow 是一条能力（S-modality-15…17）：行标签 + 模型选择器。
func mdCapabilityRow(c *gui.Context, t *gui.Theme, capability string) {
	gui.Column(c).Shrink(0).Gap(mdCapabilityLabelGap).Children(func() {
		typeSecondaryStrong.style(t, gui.Text(c, capability)).
			TextColor(t.Text).LineHeight(mdSmLine).SingleLine()
		mdModelPicker(c, t, capability)
	})
}

// mdModelPicker 是能力行下的模型选择器触发器：宽 588、高 28、1px 描边，左端
// 是占位「选择模型」，右端是下箭头。选模型要 owner 声明模型清单（见文件头），
// 因此点击只表达入口，不打开弹层。
func mdModelPicker(c *gui.Context, t *gui.Theme, capability string) {
	trigger := gui.ButtonBase(c).Shrink(0).MaxWidth(mdPickerW).Height(mdPickerH).
		Padding(0, mdPickerPadX).Radius(radius).AlignItems(gui.Center).
		Role(gui.RolePopUpButton).Label(capability + "，模型选择器，未选择").
		Tooltip(mdCopyChooseModel)
	trigger = trigger.BorderWidth(strokeHairline).BorderColor(t.Border)
	if trigger.Hovered() {
		trigger = trigger.Background(t.SurfaceHover)
	}
	trigger.Children(func() {
		gui.Row(c).Grow(1).Justify(gui.SpaceBetween).AlignItems(gui.Center).
			Children(func() {
				typeSecondary.style(t, gui.Text(c, mdCopyChooseModel)).
					TextColor(t.TextMuted).SingleLine().Grow(1)
				accIcon(c, psIconChevronDown, mdChevronSize, t.TextMuted, "chevron-down")
			})
	})
}

// ---- 任务 ----

// mdTaskSection 是「任务」分节（S-modality-27…54 的可见支）：细线 → 标题 →
// 说明 → 未声明提示 → 提交表单卡 → 所有者任务卡。
func mdTaskSection(c *gui.Context, t *gui.Theme) {
	section := gui.Column(c).Shrink(0).Margin(mdSectionGap, 0, 0, 0)
	section.Children(func() {
		screenRule(c)
		gui.Column(c).Shrink(0).Padding(mdSectionTop, 0, 0, 0).Children(func() {
			typeH4.style(t, gui.Text(c, mdCopyTasks)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(2).SingleLine()
			mdSecondary(c, t, mdCopyTasksDesc).
				Selectable().Margin(mdHeadingGap, 0, 0, 0)
			mdAlertBanner(c, t, mdCopyTaskUnavailable).Margin(mdSectionInlineGap, 0, 0, 0)
			mdTaskFormCard(c, t).Margin(mdCardGap, 0, 0, 0)
			mdTaskListCard(c, t).Margin(mdCardGap, 0, 0, 0)
		})
	})
}

// mdTaskFormCard 是提交任务表单（S-modality-30…40）：小标题 + 两列表单 +
// 通栏参数与文件 + 三列可选字段 + 提交按钮。App Server 未声明任务方法，
// 所有控件按参考渲染并禁用（可点次数 0）。
func mdTaskFormCard(c *gui.Context, t *gui.Theme) gui.Element {
	card := mdCard(c)
	card.Children(func() {
		column := gui.Column(c).Gap(mdFieldGap)
		column.Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, mdCopySubmitTask)).
				TextColor(t.Text).SingleLine()
			mdFieldRow(c, t,
				mdFieldSpec{label: mdCopyCategory, value: &modalityUI.category},
				mdFieldSpec{label: mdCopyOperation, value: &modalityUI.operation})
			mdFieldColumn(c, t, mdCopyParams, func() {
				mdTextArea(c, &modalityUI.params, mdCopyParams, mdTextareaH)
			})
			mdFieldColumn(c, t, mdCopyFiles, func() {
				mdTextArea(c, &modalityUI.files, mdCopyFiles, mdTextareaFilesH)
			})
			mdFieldTriple(c, t,
				mdFieldSpec{label: mdCopyProvider, value: &modalityUI.provider},
				mdFieldSpec{label: mdCopyModel, value: &modalityUI.model},
				mdFieldSpec{label: mdCopyClientTaskID, value: &modalityUI.clientTaskID})
			gui.Row(c).Shrink(0).Children(func() {
				gui.PrimaryButton(c, mdCopySubmitTask).Label(mdCopySubmitTask).
					Disabled(true)
			})
		})
	})
	return card
}

// mdTaskListCard 是所有者任务卡（S-modality-41…47 的可见支）：小标题 + 禁用的
// 刷新 + 空态一行。参考这一行没有状态筛选下拉（只有标题、刷新与空态）。
func mdTaskListCard(c *gui.Context, t *gui.Theme) gui.Element {
	card := mdCard(c)
	card.Children(func() {
		column := gui.Column(c).Gap(mdFieldGap)
		column.Children(func() {
			gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(mdRowGap).Children(func() {
				typeSecondaryStrong.style(t, gui.Text(c, mdCopyOwnedTasks)).
					TextColor(t.Text).SingleLine()
				gui.Spacer(c)
				mdIconButton(c, t, accIconRefresh, mdCopyRefresh, mdCopyRefresh, true)
			})
			typeSecondary.style(t, gui.Text(c, mdCopyNoTasks)).
				TextColor(t.TextMuted).Selectable()
		})
	})
	return card
}

// ---- 实时会话 ----

// mdRealtimeSection 是「实时会话」分节（S-modality-55…65）：细线 → 标题 →
// 说明 → 两条提示 → 表单卡。
func mdRealtimeSection(c *gui.Context, t *gui.Theme) {
	section := gui.Column(c).Shrink(0).Margin(mdSectionGap, 0, 0, 0)
	section.Children(func() {
		screenRule(c)
		gui.Column(c).Shrink(0).Padding(mdRealtimeTop, 0, 0, 0).Children(func() {
			typeH4.style(t, gui.Text(c, mdCopyRealtime)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(2).SingleLine()
			mdSecondary(c, t, mdCopyRealtimeDesc).
				Selectable().Margin(mdHeadingGap, 0, 0, 0)
			mdAlertBanner(c, t, mdCopyRealtimeUnavail).
				Margin(mdSectionInlineGap, 0, 0, 0)
			mdAlertBanner(c, t, mdCopyAudioUnsupported).
				Margin(mdSectionInlineGap, 0, 0, 0)
			mdRealtimeFormCard(c, t).Margin(mdRealtimeCardGap, 0, 0, 0)
		})
	})
}

// mdRealtimeFormCard 是实时会话表单（S-modality-60…65）：两列字段 + 打开 /
// 关闭两个按钮。方法未声明，控件全部禁用。
func mdRealtimeFormCard(c *gui.Context, t *gui.Theme) gui.Element {
	card := mdCardWidth(c, mdRealtimeCardW, mdRealtimeCardPad)
	card.Children(func() {
		gui.Column(c).Gap(mdFieldGap).Children(func() {
			mdFieldRowCustom(c, t,
				mdFieldSpec{label: mdCopyOperation, empty: true},
				mdFieldSpec{label: mdCopyParams, value: &modalityUI.rtParams})
			gui.Row(c).Shrink(0).Gap(mdFieldGap).Children(func() {
				mdFieldColumn(c, t, mdCopyProvider, func() {
					mdDisabledInput(c, &modalityUI.rtProvider, mdCopyProvider)
				})
				mdFieldColumn(c, t, mdCopyModel, func() {
					mdDisabledInput(c, &modalityUI.rtModel, mdCopyModel)
				})
			})
			gui.Row(c).Shrink(0).Gap(mdButtonGap).Children(func() {
				gui.PrimaryButton(c, mdCopyOpenSession).Label(mdCopyOpenSession).
					Disabled(true)
				mdIconlessButton(c, t, mdCopyCloseSession, true)
			})
		})
	})
	return card
}

// ---- 构件 ----

// mdCard 是任务表单卡：卡面（白 / 暗 0c0c0c）、1px 主题描边、卡片圆角、
// 内边距 10.5，宽度按参考的 663（窄窗时收到可用宽度）。
func mdCard(c *gui.Context) gui.Element {
	return mdCardWidth(c, mdFormCardW, mdFormCardPad)
}

// mdCardWidth 是带宽度上限与内边距的卡框。
func mdCardWidth(c *gui.Context, maxWidth, padding float32) gui.Element {
	t := c.Theme()
	return gui.Column(c).Shrink(0).MaxWidth(maxWidth).Padding(padding).
		Radius(radiusCard).Background(t.Surface).
		BorderWidth(strokeHairline).BorderColor(t.Border)
}

// mdFieldSpec 是一条表单字段的取值与标签；empty 为真时画一个空的下拉框
// （实时会话的「操作」：方法未声明，没有任何可选项）。
type mdFieldSpec struct {
	label string
	value *string
	empty bool
}

// mdFieldRow 是两列表单行（类别 / 操作）。
func mdFieldRow(c *gui.Context, t *gui.Theme, left, right mdFieldSpec) {
	mdFieldRowCustom(c, t, left, right)
}

// mdFieldRowCustom 是两列表单行，右列可以是空下拉。
func mdFieldRowCustom(c *gui.Context, t *gui.Theme, left, right mdFieldSpec) {
	gui.Row(c).Shrink(0).Gap(mdFieldGap).AlignItems(gui.Stretch).Children(func() {
		mdFieldSpecColumn(c, t, left)
		mdFieldSpecColumn(c, t, right)
	})
}

// mdFieldTriple 是三列表单行（提供商 / 模型 / 客户端任务 ID）。
func mdFieldTriple(c *gui.Context, t *gui.Theme, first, second, third mdFieldSpec) {
	gui.Row(c).Shrink(0).Gap(mdFieldGap).AlignItems(gui.Stretch).Children(func() {
		mdFieldSpecColumn(c, t, first)
		mdFieldSpecColumn(c, t, second)
		mdFieldSpecColumn(c, t, third)
	})
}

// mdFieldSpecColumn 是带标签的一列：标签在上、控件在下。
func mdFieldSpecColumn(c *gui.Context, t *gui.Theme, spec mdFieldSpec) {
	gui.Column(c).Shrink(0).Grow(1).MinWidth(0).Gap(mdLabelGap).Children(func() {
		typeSecondary.style(t, gui.Text(c, spec.label)).TextColor(t.TextMuted).
			SingleLine()
		if spec.empty {
			mdEmptySelect(c, t, spec.label)
			return
		}
		mdDisabledInput(c, spec.value, spec.label)
	})
}

// mdFieldColumn 是一个通栏字段：标签 + 调用方给的控件。Grow 让它在行里与
// 相邻字段等分宽度，在列里撑满可用宽度。
func mdFieldColumn(c *gui.Context, t *gui.Theme, label string, control func()) {
	gui.Column(c).Shrink(0).Grow(1).MinWidth(0).Gap(mdLabelGap).Children(func() {
		typeSecondary.style(t, gui.Text(c, label)).TextColor(t.TextMuted).
			SingleLine()
		control()
	})
}

// mdDisabledInput 是一个禁用的单行输入框：高 32、1px 主题描边、输入框底。
func mdDisabledInput(c *gui.Context, value *string, label string) gui.Element {
	t := c.Theme()
	return gui.TextInput(c, value).Label(label).Height(mdFieldH).
		Padding(controlPadV, mdPickerPadX).Background(fieldFill(t)).Disabled(true)
}

// mdTextArea 是一个禁用的多行输入框：宽通栏、给定高度。
func mdTextArea(c *gui.Context, value *string, label string, height float32) gui.Element {
	t := c.Theme()
	return gui.TextArea(c, value).Label(label).Height(height).
		Padding(mdTextareaPadV, mdTextareaPadX).Background(fieldFill(t)).
		Disabled(true)
}

// mdEmptySelect 是一个空的下拉触发器：与输入框同高、右端下箭头、禁用。
func mdEmptySelect(c *gui.Context, t *gui.Theme, label string) {
	trigger := gui.ButtonBase(c).Shrink(0).Grow(1).Height(mdFieldH).
		Padding(controlPadV, mdPickerPadX).Radius(radius).AlignItems(gui.Center).
		Role(gui.RolePopUpButton).Label(label).Disabled(true)
	trigger = trigger.BorderWidth(strokeHairline).BorderColor(t.Border)
	trigger.Children(func() {
		gui.Row(c).Grow(1).Justify(gui.SpaceBetween).AlignItems(gui.Center).
			Children(func() {
				gui.Spacer(c)
				accIcon(c, psIconChevronDown, mdChevronSize, t.TextMuted, "chevron-down")
			})
	})
}

// mdIconButton 是「图标 + 文字」的描边按钮（刷新）。
func mdIconButton(c *gui.Context, t *gui.Theme, icon *gui.SVG, text, label string,
	disabled bool) bool {
	return accButton(c, accButtonSpec{
		variant: accButtonOutline, height: mdFieldH, label: label, tip: label,
		paddingX: mdPickerPadX, disabled: disabled,
		children: func() {
			gui.Row(c).Gap(mdRowGap).AlignItems(gui.Center).Children(func() {
				accIcon(c, icon, mdIconSmall, t.Text, "refresh")
				typeSecondary.style(t, gui.Text(c, text)).TextColor(t.Text).
					SingleLine()
			})
		},
	})
}

// mdIconlessButton 是次按钮（关闭会话）：描边、无图标、禁用。
func mdIconlessButton(c *gui.Context, t *gui.Theme, text string, disabled bool) bool {
	return accButton(c, accButtonSpec{
		variant: accButtonOutline, height: mdFieldH, label: text, tip: text,
		paddingX: mdPickerPadX, disabled: disabled,
		children: func() {
			typeSecondary.style(t, gui.Text(c, text)).TextColor(t.Text).SingleLine()
		},
	})
}

// mdSecondary 是次级前景的说明文字：12.25 字号，行高按参考的 text-sm（17.5）。
func mdSecondary(c *gui.Context, t *gui.Theme, text string) gui.Element {
	return typeSecondary.style(t, gui.Text(c, text)).TextColor(t.TextMuted).
		LineHeight(mdSmLine)
}

// mdAlertBanner 是提示条（S-modality-03、29、57、59）：次级面、圆角、内边距
// 7/10.5，左端警示圆图标，文字次级前景、可选中复制。
func mdAlertBanner(c *gui.Context, t *gui.Theme, text string) gui.Element {
	banner := gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(mdAlertGap).
		Padding(mdAlertPadY, mdAlertPadX).Radius(radius).
		Background(t.SurfaceHover).Role(gui.RoleStatus).Label(text)
	banner.Children(func() {
		accIcon(c, mdIconAlertCircle, mdIconSmall, t.TextMuted, "alert-circle")
		mdSecondary(c, t, text).Selectable()
	})
	return banner
}

// ---- 状态 ----

// modalityUIState 是本屏可丢的界面状态：当前分组与表单草稿。
// 批量字段一起接线时搬进 App 即可。
type modalityUIState struct {
	group int

	category     string
	operation    string
	params       string
	files        string
	provider     string
	model        string
	clientTaskID string

	rtParams   string
	rtProvider string
	rtModel    string
}

var modalityUI = modalityUIState{
	params:   "{}",
	rtParams: "{}",
}

// ---- 几何 / 字阶 / 图标 ----

// 本屏几何：取参考实现的 Tailwind 确定值与 png 实测（÷2），换算成
// design-baseline §1.1 的 3.5 网格档位。
const (
	mdPageGap            = 17.5 // space-y-5：页首、提示条、页签块之间
	mdTabsGap            = 21   // 页签条与面板之间
	mdTabH               = 72   // 页签高（DOM 实测）
	mdTabPadTop          = 10.5 // 页签上内边距
	mdTabPadX            = 10.5 // 页签左右内边距
	mdTabLabelH          = 35   // min-h-10：页签名的最小高度
	mdTabCountGap        = 2    // 页签名与计数之间
	mdTabRule            = 1    // 未选中页签的底线
	mdTabRuleSelected    = 2    // 选中页签的主色条
	mdCompactW           = 800  // 窗口窄于此换成分组下拉（参考的 isCompact）
	mdPanelGap           = 21   // 面板标题行与能力行之间
	mdCapabilityGap      = 24.5 // space-y-7：能力行之间
	mdCapabilityLabelGap = 7    // 能力行标签与选择器之间
	// mdSmLine 是 12.25 字级的行高倍数：参考的 text-sm 行高 17.5 / 12.25。
	mdSmLine           = 17.5 / 12.25
	mdPickerW          = 588  // 模型选择器宽（DOM 实测）
	mdPickerH          = 28   // 选择器高（h-7）
	mdPickerPadX       = 10.5 // px-3
	mdChevronSize      = 12   // 选择器右端箭头
	mdSectionGap       = 28   // 分节之前
	mdSectionTop       = 21   // 分节细线之后的顶距
	mdSectionInlineGap = 14   // space-y-4：说明与提示条之间
	mdHeadingGap       = 4    // 标题与说明之间（实测 3.5）
	mdRealtimeTop      = 28   // 实时会话细线之后的顶距
	mdRealtimeCardGap  = 17.5 // 最后一条提示与表单卡之间
	mdFormCardW        = 663  // 任务卡宽（png 实测 663）
	mdFormCardPad      = 10.5 // 任务卡内边距
	mdRealtimeCardW    = 672  // max-w-3xl：实时表单卡宽
	mdRealtimeCardPad  = 14   // 实时表单卡内边距
	mdCardGap          = 24.5 // 两张任务卡之间
	mdFieldGap         = 10.5 // gap-3：字段行之间
	mdLabelGap         = 3.5  // 标签与控件之间
	mdFieldH           = 32   // 输入框与按钮高（h-9）
	mdTextareaH        = 72   // 参数多行输入
	mdTextareaFilesH   = 58   // 文件多行输入
	mdTextareaPadV     = 6
	mdTextareaPadX     = 10.5
	mdRowGap           = 7  // 行内图标与文字之间
	mdButtonGap        = 7  // 两个按钮之间
	mdIconSmall        = 14 // 行内图标
	mdAlertPadY        = 7
	mdAlertPadX        = 10.5
	mdAlertGap         = 7
)

// 本屏图标：Heroicons v2 outline（24 网格、1.5 描边）的内联路径，与 icons.go
// 同一套约定。下箭头与刷新复用同包已有的路径。
var mdIconAlertCircle = mustSVG(`<path d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"/>`)

// 本屏文案：逐字取参考产品的 zh-CN（i18n:modalitySettings.*、
// i18n:modalityTask.*、i18n:modalityRealtime.*）。
//
// 能力缺失的提示保留英文原文，前面加中文说明（「服务端未提供该能力：」），
// 让人先看懂是什么，再能对上服务端的原话。
const (
	mdCopyTitle       = "多模态"
	mdCopyUnavailable = "服务端未提供该能力：The App Server does not expose modality.capabilities."

	// 连着 Source 时的三段文案（按事实分支，不写死"未提供"）。
	mdCopyCapabilityUnread = "还没有读到模态能力（读数未到）"
	mdCopyUnsupported      = "服务端声明模态能力不可用："
	mdCopyTasksUnread      = "还没有读到模态任务清单（读数未到）"
	mdCopyTasksFailed      = "读取模态任务失败："

	mdCopyGroupImage    = "图像"
	mdCopyGroupVideo    = "视频"
	mdCopyGroupAudio    = "语音与音频"
	mdCopyGroupCreative = "创作与工具"
	mdCopyGroupRealtime = "实时"

	mdCopyCapabilityGenerate   = "图像生成"
	mdCopyCapabilityEdit       = "图像编辑"
	mdCopyCapabilityEnhance    = "图像增强"
	mdCopyCapabilityUnderstand = "图像理解"
	mdCopyChooseModel          = "选择模型"

	mdCopyTasks            = "任务"
	mdCopyTasksDesc        = "提交并跟踪由 App Server 拥有的 Modality 任务。"
	mdCopyTaskUnavailable  = "App Server 未声明 Modality 任务方法。"
	mdCopySubmitTask       = "提交任务"
	mdCopyCategory         = "类别"
	mdCopyOperation        = "操作"
	mdCopyParams           = "参数（JSON 对象）"
	mdCopyFiles            = "文件（JSON 数组）"
	mdCopyProvider         = "提供商（可选）"
	mdCopyModel            = "模型（可选）"
	mdCopyClientTaskID     = "客户端任务 ID（可选）"
	mdCopyOwnedTasks       = "所有者任务"
	mdCopyRefresh          = "刷新"
	mdCopyNoTasks          = "没有任务。"
	mdCopyRealtime         = "实时会话"
	mdCopyRealtimeDesc     = "通过 App Server 打开并驱动 Modality 实时会话。"
	mdCopyRealtimeUnavail  = "App Server 未声明 Modality 实时方法。"
	mdCopyAudioUnsupported = "此构建不支持音频帧：Desktop 没有安全的音频采集或编码器。文本帧仍可使用。"
	mdCopyOpenSession      = "打开会话"
	mdCopyCloseSession     = "关闭会话"
)
