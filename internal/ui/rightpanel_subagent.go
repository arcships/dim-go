package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 子代理面板（右栏「子代理」标签）：加载态、行内错误横幅、首屏错误，以及
// 只读时间线。条目编号见 requirements-panels.md §7（G-01–G-08）。
//
// 子代理的会话流由 App Server 提供；Go 侧没有这条事实，因此加载 / 就绪 / 错误
// 三态由视图状态驱动，时间线按收到的只读消息渲染。

// 子代理的文案（逐字取 requirements-panels.md §7）。
const (
	subagentLoadingText = "正在加载..."
	subagentRetry       = "重试"
	subagentLoadFailed  = "加载失败"
	subagentEmptyText   = "还没有输出"
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
	ready := a.panels.subagentReady
	failure := a.panels.subagentError
	gui.Column(c).Grow(1).Children(func() {
		switch {
		case failure != "" && ready:
			// 已就绪但流出错：行内错误横幅（G-01–G-03）。
			gui.Column(c).Shrink(0).Padding(panelPad, panelPad).Children(func() {
				panelBanner(c, failure, subagentRetry, func() {
					a.panels.subagentError = ""
				})
			})
			a.subagentTimeline(c)
		case failure != "" && !ready:
			// 还没就绪就出错：空态错误文本 + 次级重试按钮（G-04 / G-05）。
			gui.Column(c).Grow(1).Center().Padding(space4, panelCenterPad).
				Gap(space3).AlignItems(gui.Center).Children(func() {
				typeBody.style(t, gui.Text(c, failure)).TextColor(t.Warning).
					TextAlign(gui.Center).MaxLines(6)
				if gui.Button(c, subagentRetry).Label(subagentRetry).Clicked() {
					a.panels.subagentError = ""
				}
			})
		case !ready:
			// 加载态：旋转圈 + 文案（G-06 / G-07）。
			gui.Column(c).Grow(1).Center().Gap(space2).AlignItems(gui.Center).
				Children(func() {
				gui.Spinner(c).Label("正在加载")
				typeBody.style(t, gui.Text(c, subagentLoadingText)).TextColor(t.TextMuted)
			})
		default:
			a.subagentTimeline(c)
		}
	})
}

// subagentTimeline 是只读时间线（G-08）：按消息顺序铺开，不可编辑。
func (a *App) subagentTimeline(c *gui.Context) {
	lines := a.panels.subagentLines
	if len(lines) == 0 && !a.wired() {
		lines = subagentSamples()
	}
	gui.Scroll(c).Grow(1).Locked().Children(func() {
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
