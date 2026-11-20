package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 外部 Agent 的重建实现：元素按
// docs/domains/requirements/inventory-settings-rt.md §17（S-externalAgents-01…15，
// 15 个元素）逐行对应，文案逐字取参考产品中文，数值取
// docs/domains/requirements/design-baseline.md：
//
//	页面标题 17.5/600、说明 12.25、卡标题 13/600、卡片圆角 8、卡内边距 14/20、
//	行状态 11、行内按钮 28×28、开关 28×16、行高 56。
//
// 当前可见分支是「App Server 模式下不可用」：参考产品在此模式下不探测也不
// 配置外部 agent（App Server 尚未实现外部代理）。agent 行（S-08…14）按参考
// 结构实现，条件 `!unavailable` 成立时才渲染；本屏的默认状态就是不可用，
// 所以首屏只有标题、说明与提示卡三块。
//
// 只保留可丢的界面状态（externalAgentsScreen）：探测结果、行错误与开关。
// 探测与写事实都没有对应的 Intents 方法，控件照渲染、点击只改本屏状态。
//
// # 未实现分支
//
//	无。15 个元素都有对应实现；未出现的 10 个带前置条件（当前 unavailable、
//	探测已结束、无行错误），不是漏做。

// externalAgentStatus 是一行 agent 的三种状态。
type externalAgentStatus int

const (
	eaInstalled externalAgentStatus = iota
	eaUnknownVersion
	eaNotInstalled
)

// externalAgent 是一个外部 agent 的读数。
type externalAgent struct {
	agentType string
	version   string
	status    externalAgentStatus
	execPath  string
	enabled   bool
	err       string
}

// externalAgentsState 是本屏可丢的界面状态：不是事实。
type externalAgentsState struct {
	loading     bool
	loadError   string
	unavailable bool
	agents      []externalAgent
}

// externalAgentsScreen 的首屏取值与参考产品运行时一致：App Server 模式下不可用。
var externalAgentsScreen = externalAgentsState{unavailable: true}

// 本屏几何：参考的 Tailwind 量换算成 design-baseline §1.1 的档位。
const (
	eaPageGap  = 4  // 标题与说明之间
	eaIntroGap = 16 // 说明与内容之间
	eaCardPadV = 24 // 提示卡的上下内边距（png 实测 96→121，卡高 86）
	eaCardPadH = 15 // 提示卡的左右内边距（文字 x=272，卡 x=257）
	eaTextGap  = 2  // 卡内标题与说明之间
	eaRowGap   = 10.5
	eaRowH     = 56 // 行高
	eaIconHit  = 28
	eaIconSize = 14
	eaSwitchW  = 28
	eaSwitchH  = 16
)

// 字阶（design-baseline §2）。
var (
	eaTitleStyle  = typePageTitle
	eaDescStyle   = fontStep{size: 12.25, line: 17.5, weight: 400}
	eaCardTitle   = fontStep{size: 13, line: 19.5, weight: 600}
	eaStatusStyle = fontStep{size: 11, line: 14, weight: 400}
	eaNameStyle   = fontStep{size: 13, line: 19.5, weight: 600}
	eaPathStyle   = fontStep{size: 11, line: 16, weight: 400, family: codeFont}
	eaNoteStyle   = fontStep{size: 11, line: 14, weight: 400}
)

// 本屏文案：逐字取自参考产品中文。
const (
	eaCopyTitle       = "外部 Agent"
	eaCopyIntro       = "将外部 agent CLI 作为子 agent 启用，主 agent 可委派任务给它们。"
	eaCopyLoading     = "正在探测外部 Agent…"
	eaCopyUnavailable = "App Server 模式下不可用"
	eaCopyUnavailDesc = "App Server 尚未实现外部代理。此模式下 Desktop 无法检测或配置它们。"
	eaCopyUnknown     = "未识别版本"
	eaCopyMissing     = "未安装"
	eaCopyUnknownNote = "未能从该文件识别版本号，这个 Agent 可能不可用。"
	eaCopyEdit        = "编辑"
	eaCopyFootNote    = "更改将在新对话中生效。"
	eaCopyRetry       = "重试"
)

// renderExternalAgents 是设置 › 外部 Agent 这一屏（设置壳的一个分节）。
func renderExternalAgents(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Children(func() {
				gui.Column(c).Shrink(0).Gap(eaPageGap).Children(func() {
					eaTitle(c, t)
					// S-externalAgents-02 页面说明句（外部 agent CLI 作为子 agent 启用）
					eaDescStyle.style(t, gui.Text(c, eaCopyIntro)).
						TextColor(t.TextMuted).Selectable()
				})
				gui.Box(c).Shrink(0).Height(eaIntroGap)
				eaBody(c, t)
			})
		})
	})
}

// eaTitle 是页面标题（S-externalAgents-01）。
func eaTitle(c *gui.Context, t *gui.Theme) {
	eaTitleStyle.style(t, gui.Text(c, eaCopyTitle)).TextColor(t.Text).
		Role(gui.RoleHeading).Level(2).SingleLine()
}

// eaBody 在页首下按状态分支：探测中 / 失败 / 不可用卡 / agent 行。
func eaBody(c *gui.Context, t *gui.Theme) {
	switch {
	case externalAgentsScreen.loading:
		// S-externalAgents-03 正在探测外部 Agent…
		typeSecondary.style(t, gui.Text(c, eaCopyLoading)).
			TextColor(t.TextMuted).Label(eaCopyLoading)
		return
	case externalAgentsScreen.loadError != "":
		// S-externalAgents-04 读取失败原因 + 「重试」
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Wrap().Gap(eaRowGap).
			Children(func() {
				eaNoteStyle.style(t, gui.Text(c, externalAgentsScreen.loadError)).
					TextColor(t.Danger).Role(gui.RoleAlert)
				if plgOutlineButton(c, t, eaCopyRetry, rowRegular, 0) {
					// 重新探测需要 Intents：不动作（未接动作）。
				}
			})
		return
	}
	if externalAgentsScreen.unavailable {
		eaUnavailableCard(c, t)
		return
	}
	if len(externalAgentsScreen.agents) == 0 {
		return
	}
	gui.Column(c).Shrink(0).Gap(eaCardGap).Children(func() {
		for i := range externalAgentsScreen.agents {
			eaAgentRow(c, t, &externalAgentsScreen.agents[i])
		}
	})
	// S-externalAgents-15 页尾说明（更改将在新对话中生效）
	eaNoteStyle.style(t, gui.Text(c, eaCopyFootNote)).TextColor(t.TextMuted).
		Selectable()
}

// eaCardGap 是 agent 行之间的间距。
const eaCardGap = 8

// eaUnavailableCard 是不可用提示卡（S-externalAgents-05/06/07）：标题 + 一句说明。
func eaUnavailableCard(c *gui.Context, t *gui.Theme) {
	card := accCardFrame(c, gui.Column(c).Shrink(0).
		Gap(eaTextGap).Padding(eaCardPadV, eaCardPadH))
	card.Label(eaCopyUnavailable)
	card.Children(func() {
		eaCardTitle.style(t, gui.Text(c, eaCopyUnavailable)).TextColor(t.Text).
			Role(gui.RoleHeading).Level(3).SingleLine()
		eaDescStyle.style(t, gui.Text(c, eaCopyUnavailDesc)).
			TextColor(t.TextMuted).Selectable()
	})
}

// eaAgentRow 是一个外部 agent（S-externalAgents-08…14）：名称、状态、路径与
// 可选的可执行文件入口，右端是开关。
func eaAgentRow(c *gui.Context, t *gui.Theme, a *externalAgent) {
	row := accCardFrame(c, gui.Row(c).Shrink(0).MinHeight(eaRowH).
		AlignItems(gui.Center).Gap(eaRowGap).
		Padding(screenSpace15, screenSpace3))
	row.Label(a.agentType)
	row.Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Gap(eaTextGap).Children(func() {
			gui.Row(c).AlignItems(gui.Center).Gap(eaRowGap).Children(func() {
				eaNameStyle.style(t, gui.Text(c, a.agentType)).TextColor(t.Text).
					SingleLine()
				eaStatusText(c, t, a)
			})
			if a.status == eaUnknownVersion && a.execPath != "" {
				gui.Row(c).AlignItems(gui.Center).Gap(eaStatusGap).Children(func() {
					accIcon(c, accIconTriangleAlert, hkIconLockSize, t.Warning,
						"triangle-alert")
					eaNoteStyle.style(t, gui.Text(c, eaCopyUnknownNote)).
						TextColor(t.Warning)
				})
			}
			if a.execPath != "" {
				eaPathStyle.style(t, gui.Text(c, a.execPath)).TextColor(t.TextMuted).
					SingleLine().Selectable().Label("可执行文件：" + a.execPath)
			}
			if a.err != "" {
				eaNoteStyle.style(t, gui.Text(c, a.err)).TextColor(t.Danger).
					Role(gui.RoleAlert)
			}
		})
		if accIconButton(c, accIconSpec{
			icon: iconPencil, label: eaCopyEdit, tip: eaCopyEdit,
			size: eaIconSize, hit: eaIconHit,
		}) {
			// 文件选择器需要宿主能力：不动作（未接动作）。
		}
		gui.Switch(c, &a.enabled).Size(eaSwitchW, eaSwitchH).
			Radius(eaSwitchH / 2).Label(a.agentType)
	})
}

// eaStatusGap 是状态图标与提示文字的间距。
const eaStatusGap = 3.5

// eaStatusText 是行内状态（S-externalAgents-09）：版本号 / 未识别 / 未安装。
func eaStatusText(c *gui.Context, t *gui.Theme, a *externalAgent) {
	text, ink := a.version, t.Success
	switch a.status {
	case eaUnknownVersion:
		text, ink = eaCopyUnknown, t.Warning
	case eaNotInstalled:
		text, ink = eaCopyMissing, t.TextMuted
	}
	if text == "" {
		text = eaCopyMissing
	}
	eaStatusStyle.style(t, gui.Text(c, text)).TextColor(ink).Label(text)
}
