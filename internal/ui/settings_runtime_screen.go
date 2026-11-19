package ui

import (
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 运行环境的重建实现：元素按
// docs/domains/requirements/inventory-settings-rt.md §9（S-runtime-01…29，
// 29 个元素）逐行对应，文案逐字取参考产品中文，数值取
// docs/domains/requirements/design-baseline.md（页首 17.5/12.25、错误卡内边距
// 14、功能组件卡内边距 17.5、按钮高 28）。
//
// 结构：页首（标题 + 说明 + 刷新）→ 运行环境条目区 → 功能组件（OfficeCLI 卡）。
//
// # 可达分支
//
// 条目区两档：读过 Settings().Runtimes 就一条一张卡（名称、版本、状态、原因），
// 没读到就一张原因卡（适配器给的 RuntimesReason）；-demo 保留参考的样例错误卡。
// 功能组件只有 -demo 画「未安装 / 配套版本」，连着 Source 时没有安装状态事实，
// 只写说明并禁用入口。未实现的分支都带前置条件：
//
//	S-runtime-04              加载中的「正在加载...」：读态由适配器合并，界面
//	                 不区分"还没开始读"与"正在读"。
//	S-runtime-07/08           条目内的来源分段、启动时获取开关：服务端不报告。
//	S-runtime-12…14           安装按钮与进度、逐条目错误：宿主安装能力未接。
//	S-runtime-19/20/21        功能组件的加载错误、重试与加载中：服务端没有组件
//	                 状态事实。
//	S-runtime-26…28           下载进度、逐组件错误、已安装说明：同上。
//
// 刷新与「重试」都只表达入口：本版没有手动刷新通道。「下载安装」属宿主能力，
// 同样不写事实。
//
// # 事实来源
//
// 连着 Source 时读 Settings().Runtimes / RuntimesKnown：读过就按条目画（名称、
// 版本、状态与原因）；没读到就把适配器给的 RuntimesReason 写出来，不写死一句
// 「服务端未提供」——那句话在服务端确实提供了清单时就是假的。功能组件没有安装
// 状态事实，只写一句说明并禁用入口。-demo（没有 Source）保留样例错误卡。
func renderRuntime(c *gui.Context, a *App) {
	t := c.Theme()
	st, live := a.liveSettings()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(rtPageGap).Children(func() {
				rtHeader(c, t, live)
				if live {
					rtLiveSection(c, t, st)
				} else {
					rtErrorCard(c, t)
				}
				rtFeatureSection(c, t, live)
			})
		})
	})
}

// rtLiveSection 是连着 Source 时的条目区：读过清单就画条目，没读过就画原因。
func rtLiveSection(c *gui.Context, t *gui.Theme, st SettingsSnapshot) {
	if !st.RuntimesKnown {
		reason := strings.TrimSpace(st.RuntimesReason)
		if reason == "" {
			reason = rtCopyUnread
		}
		rtReasonCard(c, t, reason)
		return
	}
	if len(st.Runtimes) == 0 {
		rtEmptyCard(c, t)
		return
	}
	gui.Column(c).Shrink(0).Gap(rtCardGap).Children(func() {
		for _, entry := range st.Runtimes {
			rtRuntimeCard(c, t, entry)
		}
	})
}

// rtRuntimeCard 是一条运行环境读数：名称、版本、状态与（不可用时的）原因。
func rtRuntimeCard(c *gui.Context, t *gui.Theme, entry RuntimeSnapshot) {
	screenCard(c, func() {
		gui.Column(c).Padding(rtCardPad).Gap(rtCardGap).Children(func() {
			gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(rtRowGap).Children(func() {
				typeH4.style(t, gui.Text(c, entry.Name)).TextColor(t.Text).
					Role(gui.RoleHeading).Level(3).SingleLine()
				if entry.Version != "" {
					typeMicro.style(t, gui.Text(c, entry.Version)).TextColor(t.TextMuted).
						SingleLine()
				}
				gui.Spacer(c)
				if entry.Status != "" {
					typeSecondary.style(t, gui.Text(c, entry.Status)).TextColor(t.TextMuted).
						Role(gui.RoleStatus).Label(entry.Status)
				}
			})
			if entry.Reason != "" {
				typeMicro.style(t, gui.Text(c, entry.Reason)).TextColor(t.Danger).
					Selectable()
			}
		})
	})
}

// rtReasonCard 是没读到清单时的原因卡：只写适配器给的原因与「本版没有刷新通道」。
func rtReasonCard(c *gui.Context, t *gui.Theme, reason string) {
	card := gui.Column(c).Shrink(0).Padding(rtCardPad).Gap(rtCardGap).
		Radius(radiusCard).Background(t.Surface).Role(gui.RoleStatus).Label(rtCopyTitle)
	card.BorderWidth(strokeHairline).BorderColor(t.Border)
	card.Children(func() {
		typeSecondary.style(t, gui.Text(c, reason)).TextColor(t.TextMuted).
			LineHeight(mdSmLine).Selectable()
		typeMicro.style(t, gui.Text(c, noteRuntimeRefresh)).TextColor(t.TextMuted).
			Selectable()
	})
}

// rtEmptyCard 是读过清单但服务端没有条目时的空态。
func rtEmptyCard(c *gui.Context, t *gui.Theme) {
	screenCard(c, func() {
		typeSecondary.style(t, gui.Text(c, rtCopyEmptyLive)).TextColor(t.TextMuted).
			Selectable()
	})
}

// rtHeader 是页首（S-runtime-01…03）：标题 + 说明在左，刷新在右、与标题同排。
func rtHeader(c *gui.Context, t *gui.Theme, live bool) {
	gui.Row(c).Shrink(0).Gap(rtHeaderGap).AlignItems(gui.Start).Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Gap(rtTitleGap).Children(func() {
			typePageTitle.style(t, gui.Text(c, rtCopyTitle)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(1).SingleLine()
			mdSecondary(c, t, rtCopyDesc).Selectable()
		})
		rtRefreshButton(c, t, live)
	})
}

// rtRefreshButton 是页首的「刷新」：描边、高 28、图标 + 文字。读数由连接层按需
// 重读，界面没有手动刷新通道：入口置灰并写原因（DEC-13）。
func rtRefreshButton(c *gui.Context, t *gui.Theme, live bool) {
	tip := noteRuntime
	if live {
		tip = noteRuntimeRefresh
	}
	accButton(c, accButtonSpec{
		variant: accButtonOutline, height: rtButtonH, label: rtCopyRefresh,
		tip: tip, paddingX: rtPadBtnX, disabled: true,
		children: func() {
			gui.Row(c).Gap(rtIconGap).AlignItems(gui.Center).Children(func() {
				accIcon(c, accIconRefresh, rtIconInline, t.Text, "refresh")
				typeSecondary.style(t, gui.Text(c, rtCopyRefresh)).TextColor(t.Text).
					SingleLine()
			})
		},
	})
}

// rtErrorCard 是条目区的错误分支（S-runtime-06/07）：危险色描边的卡，
// 内为错误原文与「重试」。参考的 role=status 保留在卡上。
func rtErrorCard(c *gui.Context, t *gui.Theme) {
	card := gui.Column(c).Shrink(0).Padding(rtCardPad).Gap(rtCardGap).
		Radius(radiusCard).Background(t.Surface).
		BorderWidth(strokeHairline).BorderColor(t.Danger.Alpha(rtErrorAlpha)).
		Role(gui.RoleStatus).Label(rtCopyError)
	card.Children(func() {
		typeSecondary.style(t, gui.Text(c, rtCopyError)).TextColor(t.Danger).
			LineHeight(mdSmLine).Selectable()
		// 「重试」左对齐、按内容取宽：卡是列容器，按钮要放进行里才不撑满。
		gui.Row(c).Shrink(0).Children(func() {
			accButton(c, accButtonSpec{
				variant: accButtonOutline, height: rtButtonH, label: rtCopyRetry,
				tip: noteRuntime, paddingX: rtPadBtnX, disabled: true,
				children: func() {
					typeSecondary.style(t, gui.Text(c, rtCopyRetry)).TextColor(t.Text).
						SingleLine()
				},
			})
		})
		// 入口禁用后仍把原因写在卡内，读屏与目视都能看到。
		typeMicro.style(t, gui.Text(c, noteRuntime)).TextColor(t.TextMuted).
			Selectable()
	})
}

// rtFeatureSection 是功能组件（S-runtime-18…29 的可见支）：分组标题 + 卡。
func rtFeatureSection(c *gui.Context, t *gui.Theme, live bool) {
	gui.Column(c).Shrink(0).Gap(rtSectionGap).Children(func() {
		typeH4.style(t, gui.Text(c, rtCopyFeatures)).TextColor(t.Text).
			Role(gui.RoleHeading).Level(2).SingleLine()
		rtFeatureCard(c, t, live)
	})
}

// rtFeatureCard 是功能组件卡：标题行（名称 + 右侧状态）+ 说明 + 版本 +
// 主按钮。卡用共用卡框（卡面、1px 主题描边、卡片圆角），内边距 17.5。
//
// 连着 Source 时没有组件安装状态事实：不写「未安装」，也不写配套版本与下载大小，
// 只说明读不到并禁用安装入口。
func rtFeatureCard(c *gui.Context, t *gui.Theme, live bool) {
	tip := noteRuntime
	if live {
		tip = noteRuntimeFeature
	}
	screenCard(c, func() {
		gui.Column(c).Padding(rtFeaturePad).Children(func() {
			gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(rtRowGap).Children(func() {
				typeH4.style(t, gui.Text(c, rtCopyOfficeCLI)).TextColor(t.Text).
					Role(gui.RoleHeading).Level(3).SingleLine()
				gui.Spacer(c)
				if !live {
					typeSecondary.style(t, gui.Text(c, rtCopyNotInstalled)).
						TextColor(t.TextMuted).LineHeight(mdSmLine).
						Role(gui.RoleStatus).Label(rtCopyNotInstalled)
				}
			})
			typeSecondary.style(t, gui.Text(c, rtCopyOfficeCLIDesc)).
				TextColor(t.TextMuted).LineHeight(mdSmLine).Selectable().
				Margin(rtTextGap, 0, 0, 0)
			if !live {
				typeMicro.style(t, gui.Text(c, rtCopyOfficeCLIVersion)).
					TextColor(t.TextMuted).SingleLine().Margin(rtTextGap, 0, 0, 0)
			}
			gui.Row(c).Shrink(0).Margin(rtButtonGap, 0, 0, 0).Children(func() {
				gui.PrimaryButton(c, rtCopyInstall).Label(rtCopyInstall).
					Tooltip(tip).Disabled(true)
			})
			typeMicro.style(t, gui.Text(c, tip)).TextColor(t.TextMuted).
				Selectable().Margin(rtTextGap, 0, 0, 0)
		})
	})
}

// 本屏几何：取参考实现的 Tailwind 确定值与 png 实测（÷2）。
const (
	rtPageGap    = 17.5 // space-y-5：页首与两块内容之间
	rtHeaderGap  = 17.5 // 标题块与刷新之间
	rtTitleGap   = 3.5  // 标题与说明之间（space-y-1）
	rtSectionGap = 14   // space-y-4：分组标题与卡之间
	rtCardPad    = 14   // p-4：错误卡内边距
	rtCardGap    = 10.5 // 错误原文与重试之间
	rtFeaturePad = 17.5 // p-5：功能组件卡内边距
	rtTextGap    = 10.5 // 卡内块之间（space-y-3）
	rtButtonGap  = 10.5 // 版本行与安装按钮之间
	rtRowGap     = 7    // 标题与状态之间
	rtIconGap    = 7    // 刷新按钮内图标与文字之间
	rtIconInline = 14   // 按钮内图标
	rtButtonH    = 28   // h-8 / size="sm"：刷新、重试、安装
	rtPadBtnX    = 10.5 // px-3：小按钮的左右内边距（参考 63×28）
	rtErrorAlpha = 0.35 // 错误卡描边的不透明度（png 实测的浅红边）
)

// 本屏文案：逐字取参考产品的 zh-CN（i18n:runtimeSettings.*、
// i18n:featureRuntimes.*）。
//
// 能力缺失的提示保留英文原文，前面加中文说明（「服务端未提供该能力：」），
// 让人先看懂是什么，再能对上服务端的原话。
const (
	rtCopyTitle            = "运行环境"
	rtCopyDesc             = "由 App Server RuntimeManager 管理；Desktop 不下载、不保留第二份运行时。"
	rtCopyRefresh          = "刷新"
	rtCopyError            = "服务端未提供该能力：The App Server does not advertise runtime.list; update the bundled runtime."
	rtCopyRetry            = "重试"
	rtCopyUnread           = "服务端还没有给出运行环境清单（读数未到）"
	rtCopyEmptyLive        = "服务端没有报告任何运行环境条目。"
	rtCopyFeatures         = "功能组件"
	rtCopyOfficeCLI        = "OfficeCLI"
	rtCopyOfficeCLIDesc    = "创建、读取和编辑 Word、Excel、PowerPoint 文件。"
	rtCopyOfficeCLIVersion = "配套版本 1.0.135-desktop.1 · 下载 13.0 MB"
	rtCopyNotInstalled     = "未安装"
	rtCopyInstall          = "下载安装"
)
