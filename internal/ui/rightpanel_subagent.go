package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 子代理面板（右栏「子代理」标签）：行内错误横幅、首屏错误，以及只读时间线。
// 条目编号见 requirements-panels.md §7（G-01–G-08）。
//
// 子代理的会话流由 App Server 提供，而协议 v3 没有这条事实（`Intents` 里也没有
// 读取子代理会话的方法）。因此连上 Source 又拿不到消息时，面板给空态
// 「未打开子代理会话」并写清原因，不摆一个永远转下去的加载圈——那不是"加载中"，
// 是一个没有写入点的假状态。

// 子代理的文案（逐字取 requirements-panels.md §7）。
const (
	subagentRetry      = "重试"
	subagentLoadFailed = "加载失败"
	subagentEmptyText  = "还没有输出"
	// subagentNoSession 是没有子代理消息时的空态标题。
	subagentNoSession = "未打开子代理会话"
)

// iconCircleStop 是「结束分支」的圆形停止图标（CircleStop）。
var iconCircleStop = mustSVG(`<circle cx="12" cy="12" r="8.25"/><rect x="8.75" y="8.75" width="6.5" height="6.5" rx="1"/>`)

// subagentSamples 是子代理只读时间线的样例（没有 Source 时的走查内容）。
func subagentSamples() []string {
	return []string{
		"把 internal/ui 里所有调用旧发送路径的地方找出来。",
		"找到 3 处：composer.go:213、timeline.go:88、app.go:441。三处都在改完后重新跑过测试。",
	}
}

// subagentPanel 是子代理面板。
func (a *App) subagentPanel(c *gui.Context) {
	t := c.Theme()
	live := a.wired()
	failure := a.panels.subagentError
	// G-02：读取失败但错误原文为空时仍画错误行，文案换成通用的「加载失败」，
	// 不留一条空横幅、也不退回空态。
	failed := a.panels.subagentFailed || failure != ""
	if failed && failure == "" {
		failure = subagentLoadFailed
	}
	lines := a.panels.subagentLines
	gui.Column(c).Grow(1).Children(func() {
		switch {
		case failed && len(lines) > 0:
			// 已有消息但流出错：行内错误横幅（G-01–G-03）。
			gui.Column(c).Shrink(0).Padding(panelPad, panelPad).Children(func() {
				if live {
					// 没有重读通道：横幅只写原文，重试入口在下面单独置灰写原因。
					panelBanner(c, failure, "", nil)
					typeMicro.style(t, gui.Text(c, noteSubagentFacts)).
						TextColor(t.TextMuted)
					a.subagentRetryEntry(c)
					return
				}
				panelBanner(c, failure, subagentRetry, func() {
					a.panels.subagentError = ""
					a.panels.subagentFailed = false
				})
			})
			a.subagentTimeline(c)
		case failed:
			// 还没有消息就出错：空态错误文本 + 重试（G-04 / G-05）。
			gui.Column(c).Grow(1).Center().Padding(space4, panelCenterPad).
				Gap(space3).AlignItems(gui.Center).Children(func() {
				typeBody.style(t, gui.Text(c, failure)).TextColor(t.Warning).
					TextAlign(gui.Center).MaxLines(6)
				if live {
					typeMicro.style(t, gui.Text(c, noteSubagentFacts)).
						TextColor(t.TextMuted).TextAlign(gui.Center).MaxLines(4)
				}
				a.subagentRetryEntry(c)
			})
		case live && len(lines) == 0:
			// 没有事实：空态 + 原因，不转圈。
			gui.Column(c).Grow(1).Center().Padding(space4, panelCenterPad).
				Gap(space2).AlignItems(gui.Center).Children(func() {
				typeBody.style(t, gui.Text(c, subagentNoSession)).TextColor(t.TextMuted).
					TextAlign(gui.Center)
				typeMicro.style(t, gui.Text(c, noteSubagentFacts)).TextColor(t.TextMuted).
					TextAlign(gui.Center).MaxLines(4).Role(gui.RoleStatus)
			})
		default:
			a.subagentTimeline(c)
		}
	})
}

// subagentRetryEntry 画「重试」。连着 Source 时没有重读通道（协议没有子代理
// 会话的读取方法）：入口照留、置灰、原因写在 tooltip 上；-demo 里它只清掉样例
// 错误，不发任何请求。
func (a *App) subagentRetryEntry(c *gui.Context) {
	btn := gui.Button(c, subagentRetry).Label(subagentRetry)
	if a.wired() {
		btn.Tooltip(noteSubagentFacts).Disabled(true).Opacity(noteDisabledAlpha)
		return
	}
	if btn.Clicked() {
		a.panels.subagentError = ""
		a.panels.subagentFailed = false
	}
}

// subagentTimeline 是只读时间线（G-08）：按消息顺序铺开，不可编辑。
func (a *App) subagentTimeline(c *gui.Context) {
	lines := a.panels.subagentLines
	if len(lines) == 0 && !a.wired() {
		lines = subagentSamples()
	}
	gui.Scroll(c).Grow(1).Children(func() {
		gui.Column(c).Padding(panelPad).Gap(space3).Children(func() {
			if len(lines) == 0 {
				typeCode.style(c.Theme(), gui.Text(c, subagentEmptyText)).
					TextColor(c.Theme().TextMuted).MaxLines(1)
				return
			}
			for i, text := range lines {
				panelMessage(c, text, i%2 == 0)
			}
		})
	})
}
