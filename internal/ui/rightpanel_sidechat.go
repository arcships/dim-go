package ui

import (
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 侧聊面板（右栏「侧聊」标签）：错误横幅 + 结束状态条 + 只读内容区 + 输入区，
// 结束分支走确认框。条目编号见 requirements-panels.md §6（S-01–S-18）。
//
// 侧聊的分支、消息与待处理交互都由 App Server 持有；Go 侧只保留草稿、确认框
// 与本地回声的视图状态。没有事实时呈现空态（与参考产品的空侧聊一致）。

// 侧聊的文案（逐字取 requirements-panels.md §6）。
const (
	sideChatClosedNotice = "此分支已结束，不再接收消息。历史仍可在此查看；完成后可关闭此标签页。"
	sideChatEmptyBefore  = "此侧边对话会从所选消息之前开始。"
	sideChatEmptyThread  = "就当前 thread 提问。侧边对话会继承启动时的上下文。"
	sideChatPlaceholder  = "问一个旁支问题…"
	sideChatEndTitle     = "结束此分支？"
	sideChatEndDetail    = "此操作会永久结束该分支并中断当前回合。分支将不再接收消息，但历史仍可在此标签页中查看。"
	sideChatRetry        = "重试"
	sideChatLoadFailed   = "加载失败"
)

// sideChatPanel 是侧聊面板。
func (a *App) sideChatPanel(c *gui.Context) {
	gui.Column(c).Grow(1).Children(func() {
		if a.panels.sideChatError != "" {
			// 错误原文为空时给通用文案，不留空白横幅（S-02 的异常分支）。
			message := strings.TrimSpace(a.panels.sideChatError)
			if message == "" {
				message = sideChatLoadFailed
			}
			action := ""
			if len(a.panels.sideChatSent) == 0 {
				action = sideChatRetry
			}
			gui.Column(c).Shrink(0).Padding(panelPad, panelPad).Children(func() {
				panelBanner(c, message, action, func() {
					a.panels.sideChatError = ""
				})
			})
		}
		if a.panels.sideChatEnded {
			a.sideChatClosedBar(c)
		}
		a.sideChatContent(c)
		a.sideChatComposer(c)
	})
	if a.panels.sideChatConfirm {
		a.sideChatEndDialog(c)
	}
}

// sideChatClosedBar 是「此分支已结束」状态条（S-04）：灰底、顶部分隔线。
func (a *App) sideChatClosedBar(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Children(func() {
		gui.Divider(c)
		gui.Row(c).Shrink(0).Padding(space2, panelPad).
			Background(t.SurfaceHover).Children(func() {
			typeSecondary.style(t, gui.Text(c, sideChatClosedNotice)).
				TextColor(t.TextMuted).Role(gui.RoleStatus)
		})
	})
}

// sideChatContent 是内容区（S-05–S-08）：有消息给只读时间线，发送中给旋转圈，
// 否则给空态文案（由是否「从某条消息之前发起」决定用哪一条）。
func (a *App) sideChatContent(c *gui.Context) {
	t := c.Theme()
	sent := a.panels.sideChatSent
	switch {
	case len(sent) > 0:
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Padding(panelPad).Gap(space3).Children(func() {
				for _, text := range sent {
					gui.Column(c).Shrink(0).AlignItems(gui.End).Children(func() {
						gui.Column(c).Shrink(0).Padding(space2, space3).
							Radius(radiusCard).Background(t.SurfaceHover).
							Children(func() {
								typeBody.style(t, gui.Text(c, text)).TextColor(t.Text).
									Selectable().MaxLines(8)
							})
					})
				}
			})
		})
	case a.panels.sideChatSending:
		gui.Column(c).Grow(1).Center().Children(func() {
			gui.Spinner(c).Label("正在加载")
		})
	case a.panels.sideChatFromMessage:
		panelEmpty(c, sideChatEmptyBefore)
	default:
		panelEmpty(c, sideChatEmptyThread)
	}
}

// sideChatComposer 是输入区（S-11–S-14）：输入框 + 结束分支 / 停止 / 发送。
func (a *App) sideChatComposer(c *gui.Context) {
	t := c.Theme()
	closed := a.panels.sideChatEnded
	sending := a.panels.sideChatSending
	draft := strings.TrimSpace(a.panels.sideChatDraft)
	gui.Row(c).Shrink(0).Padding(space2, panelPad).Gap(space1).
		AlignItems(gui.Center).Children(func() {
		gui.TextInputBase(c, &a.panels.sideChatDraft).
			Label("侧边对话输入").Placeholder(sideChatPlaceholder).
			Height(panelInputHeight).Radius(radius).Padding(space1, space2).
			Background(t.Surface).Border(strokeHairline, t.Border).Grow(1).
			Disabled(closed || sending)
		if !closed && len(a.panels.sideChatSent) > 0 {
			if iconButton(c, iconButtonSpec{
				icon: iconCircleStop, label: "结束分支", tip: "结束分支", size: rowCompact,
			}) {
				a.panels.sideChatConfirm = true
			}
		}
		if sending {
			if iconButton(c, iconButtonSpec{
				icon: iconStop, label: "停止", tip: "停止", size: rowCompact,
			}) {
				a.panels.sideChatSending = false
			}
		}
		label := "发送"
		if sending {
			label = "引导"
		}
		if iconAction(c, iconButtonSpec{
			icon: iconArrowUp, label: label, tip: label + " · Enter", size: panelSendSize,
			disabled: draft == "" || closed,
		}, true) {
			a.sendSideChat()
		}
	})
}

// sendSideChat 把草稿作为一条消息发出去（本地回声；连着 Source 时由事实接管）。
func (a *App) sendSideChat() {
	text := strings.TrimSpace(a.panels.sideChatDraft)
	if text == "" || a.panels.sideChatEnded {
		return
	}
	a.panels.sideChatSent = append(a.panels.sideChatSent, text)
	a.panels.sideChatDraft = ""
}

// sideChatEndDialog 是结束分支的确认框（S-15–S-18）。
func (a *App) sideChatEndDialog(c *gui.Context) {
	t := c.Theme()
	open := a.panels.sideChatConfirm
	m := gui.Modal(c, &open, func() {
		gui.Column(c).Gap(space3).Children(func() {
			typeH3.style(t, gui.Text(c, sideChatEndTitle)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(3)
			typeBody.style(t, gui.Text(c, sideChatEndDetail)).TextColor(t.TextMuted)
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				if gui.Button(c, "取消").Clicked() {
					a.panels.sideChatConfirm = false
				}
				if panelDangerButton(c, "确认") {
					a.panels.sideChatConfirm = false
					a.panels.sideChatEnded = true
					a.panels.sideChatSending = false
				}
			})
		})
	})
	if !open || m.Dismissed() {
		a.panels.sideChatConfirm = false
	}
}

// panelDangerButton 是面板里的危险主按钮：危险色实底 + 反白文字（S-18）。
func panelDangerButton(c *gui.Context, text string) bool {
	t := c.Theme()
	b := gui.ButtonBase(c).Height(rowRegular).Padding(controlPadV, controlPadH).
		Radius(radius).Background(t.Danger).Label(text)
	switch {
	case b.Pressed():
		b = b.Background(t.Danger.Alpha(0.85))
	case b.Hovered():
		b = b.Background(t.Danger.Alpha(0.9))
	}
	b.Children(func() {
		typeControlStrong.style(t, gui.Text(c, text)).TextColor(t.AccentText).SingleLine()
	})
	return b.Clicked()
}
