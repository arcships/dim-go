package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 运行环境的重建实现：元素按
// docs/domains/requirements/inventory-settings-rt.md §9（S-runtime-01…29，
// 29 个元素）逐行对应，文案逐字取参考产品中文，数值取
// docs/domains/requirements/design-baseline.md（页首 17.5/12.25、错误卡内边距
// 14、功能组件卡内边距 17.5、按钮高 28）。
//
// 结构：页首（标题 + 说明 + 刷新）→ 运行环境条目区（当前是错误分支）→
// 功能组件（OfficeCLI 卡）。
//
// # 可达分支
//
// App Server 当前不声明 runtime.list，条目区显示错误分支；功能组件快照已到、
// OfficeCLI 未安装。参考实现在该状态下可见 11 个元素，本文件实现这 11 个；
// 未实现的 18 个都带前置条件：
//
//	S-runtime-04              加载中的「正在加载...」：错误分支取代了它。
//	S-runtime-05…17           entries 非 null 时的条目卡（Node.js / Python / uv
//	                 三条、状态图标、来源分段、启动时获取开关、安装按钮、
//	                 版本与路径、原因、缺失条目兜底、条目内错误）：App Server
//	                 没有 runtime.list，就没有可画的条目。
//	S-runtime-19/20/21        功能组件的加载错误、重试与加载中：快照已到。
//	S-runtime-26…28           下载进度、逐组件错误、已安装说明：未在下载、
//	                 无错误、组件未安装。
//
// 刷新与「重试」都只表达入口：protocol v3 没有 runtime 读取方法，点击不写
// 事实（未接动作）。「下载安装」属宿主能力，同样不写事实。
func renderRuntime(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(rtPageGap).Children(func() {
				rtHeader(c, t)
				rtErrorCard(c, t)
				rtFeatureSection(c, t)
			})
		})
	})
}

// rtHeader 是页首（S-runtime-01…03）：标题 + 说明在左，刷新在右、与标题同排。
func rtHeader(c *gui.Context, t *gui.Theme) {
	gui.Row(c).Shrink(0).Gap(rtHeaderGap).AlignItems(gui.Start).Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Gap(rtTitleGap).Children(func() {
			typeH2.style(t, gui.Text(c, rtCopyTitle)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(1).SingleLine()
			mdSecondary(c, t, rtCopyDesc).Selectable()
		})
		rtRefreshButton(c, t)
	})
}

// rtRefreshButton 是页首的「刷新」：描边、高 28、图标 + 文字。
func rtRefreshButton(c *gui.Context, t *gui.Theme) {
	accButton(c, accButtonSpec{
		variant: accButtonOutline, height: rtButtonH, label: rtCopyRefresh,
		tip: rtCopyRefresh, paddingX: rtPadBtnX,
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
				tip: rtCopyRetry, paddingX: rtPadBtnX,
				children: func() {
					typeSecondary.style(t, gui.Text(c, rtCopyRetry)).TextColor(t.Text).
						SingleLine()
				},
			})
		})
	})
}

// rtFeatureSection 是功能组件（S-runtime-18…29 的可见支）：分组标题 + 卡。
func rtFeatureSection(c *gui.Context, t *gui.Theme) {
	gui.Column(c).Shrink(0).Gap(rtSectionGap).Children(func() {
		typeH4.style(t, gui.Text(c, rtCopyFeatures)).TextColor(t.Text).
			Role(gui.RoleHeading).Level(2).SingleLine()
		rtFeatureCard(c, t)
	})
}

// rtFeatureCard 是功能组件卡：标题行（名称 + 右侧状态）+ 说明 + 版本 +
// 主按钮。卡用共用卡框（卡面、1px 主题描边、卡片圆角），内边距 17.5。
func rtFeatureCard(c *gui.Context, t *gui.Theme) {
	screenCard(c, func() {
		gui.Column(c).Padding(rtFeaturePad).Children(func() {
			gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(rtRowGap).Children(func() {
				typeH4.style(t, gui.Text(c, rtCopyOfficeCLI)).TextColor(t.Text).
					Role(gui.RoleHeading).Level(3).SingleLine()
				gui.Spacer(c)
				typeSecondary.style(t, gui.Text(c, rtCopyNotInstalled)).
					TextColor(t.TextMuted).LineHeight(mdSmLine).
					Role(gui.RoleStatus).Label(rtCopyNotInstalled)
			})
			typeSecondary.style(t, gui.Text(c, rtCopyOfficeCLIDesc)).
				TextColor(t.TextMuted).LineHeight(mdSmLine).Selectable().
				Margin(rtTextGap, 0, 0, 0)
			typeMicro.style(t, gui.Text(c, rtCopyOfficeCLIVersion)).
				TextColor(t.TextMuted).SingleLine().Margin(rtTextGap, 0, 0, 0)
			gui.Row(c).Shrink(0).Margin(rtButtonGap, 0, 0, 0).Children(func() {
				screenButton(c, rtCopyInstall, rtCopyInstall, "", true, rtButtonH)
			})
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
const (
	rtCopyTitle            = "运行环境"
	rtCopyDesc             = "由 App Server RuntimeManager 管理；Desktop 不下载、不保留第二份运行时。"
	rtCopyRefresh          = "刷新"
	rtCopyError            = "The App Server does not advertise runtime.list; update the bundled runtime."
	rtCopyRetry            = "重试"
	rtCopyFeatures         = "功能组件"
	rtCopyOfficeCLI        = "OfficeCLI"
	rtCopyOfficeCLIDesc    = "创建、读取和编辑 Word、Excel、PowerPoint 文件。"
	rtCopyOfficeCLIVersion = "配套版本 1.0.135-desktop.1 · 下载 13.0 MB"
	rtCopyNotInstalled     = "未安装"
	rtCopyInstall          = "下载安装"
)
