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
	// sideChatEndedNote / sideChatSendingNote 是输入区不可用的两种界面原因
	// （S-11）：与「本版没有写入通道」分开写，灰着不说明就成了没有原因的灰。
	sideChatEndedNote   = "此分支已结束，不再接收消息"
	sideChatSendingNote = "正在发送，稍后可继续输入"
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
			// 「重试」只在 -demo 有：连着 Source 时没有重读侧聊的通道，留一个
			// 点了只清错误行的按钮就是假重试（U3 §13.6）。
			action := ""
			if len(a.panels.sideChatSent) == 0 && !a.wired() {
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
//
// 没打开会话与"有会话但读不到侧聊"分开写：前者缺整个会话，后者缺通道
// （U3 §13.6）。
func (a *App) sideChatContent(c *gui.Context) {
	t := c.Theme()
	if a.source != nil && !a.hasOpenSession() {
		panelEmptyNote(c, "未打开会话", "打开一个会话后，这里显示它的侧边对话。")
		return
	}
	sent := a.panels.sideChatSent
	switch {
	case len(sent) > 0:
		gui.Scroll(c).Grow(1).Label("侧边对话消息").Role(gui.RoleLog).Children(func() {
			gui.Column(c).Padding(panelPad).Gap(space3).Children(func() {
				for _, text := range sent {
					gui.Column(c).Shrink(0).AlignItems(gui.End).Children(func() {
						gui.Column(c).Shrink(0).Padding(space2, space3).
							Radius(radiusCard).Background(t.SurfaceHover).
							Role(gui.RoleListItem).Children(func() {
							typeBody.style(t, gui.Text(c, text)).TextColor(t.Text).
								Selectable().MaxLines(8)
						})
					})
				}
			})
		})
	case a.panels.sideChatSending:
		gui.Column(c).Grow(1).Center().Children(func() {
			gui.Spinner(c).Label("正在等待回复")
		})
	case a.panels.sideChatFromMessage:
		panelEmptyNote(c, sideChatEmptyBefore, a.sideChatFactsNote())
	default:
		panelEmptyNote(c, sideChatEmptyThread, a.sideChatFactsNote())
	}
}

// sideChatFactsNote 是没有消息可画时的原因：连着 Source 时侧聊的分支与消息都在
// 服务端，本端没有读取通道；-demo 的样例本来就有内容，不需要这句。
func (a *App) sideChatFactsNote() string {
	if a.wired() {
		return noteSideChatFacts
	}
	return ""
}

// sideChatComposer 是输入区（S-11–S-14）：输入框 + 结束分支 / 停止 / 发送。
//
// 侧聊的写入没有通道（`Intents` 里没有侧聊方法）：连着 Source 时输入框与发送
// 一起置灰，原因写在输入区上方（也在 tooltip 里），绝不把草稿 append 进本地
// 切片冒充「已发送」。「停止」与「结束分支」同样置灰写原因，不把确认框当成真
// 结束（那样状态条会写"此分支已结束"，服务端却照旧）。
func (a *App) sideChatComposer(c *gui.Context) {
	t := c.Theme()
	closed := a.panels.sideChatEnded
	sending := a.panels.sideChatSending
	draft := strings.TrimSpace(a.panels.sideChatDraft)
	live := a.wired()
	gui.Column(c).Shrink(0).Padding(space2, panelPad).Gap(space1).Children(func() {
		// 原因写在输入区上方：禁用态旁边看得见为什么不能发，不只在 tooltip 里。
		// 分支已结束时上方的状态条已经说明了，不再重复一行。
		if live {
			typeMicro.style(t, gui.Text(c, noteSideChatWrite)).TextColor(t.TextMuted).
				Role(gui.RoleStatus)
		}
		gui.Row(c).Shrink(0).Gap(space1).AlignItems(gui.Center).Children(func() {
			input := gui.TextInputBase(c, &a.panels.sideChatDraft).
				Label("侧边对话输入").Placeholder(sideChatPlaceholder).
				Height(panelInputHeight).Radius(radius).Padding(space1, space2).
				Background(t.Surface).Border(strokeHairline, t.Border).Grow(1).
				Disabled(closed || sending || live)
			if note := a.sideChatDisabledNote(); note != "" {
				input = input.Tooltip(note)
			}
			if !closed && len(a.panels.sideChatSent) > 0 {
				end := iconButtonSpec{
					icon: iconCircleStop, label: "结束分支", tip: "结束分支", size: rowCompact,
				}
				if live {
					end.disabled, end.tip = true, noteSideChatWrite
				}
				if iconButton(c, end) {
					a.panels.sideChatConfirm = true
				}
			}
			if sending {
				stop := iconButtonSpec{icon: iconStop, label: "停止", tip: "停止", size: rowCompact}
				if live {
					stop.disabled, stop.tip = true, noteSideChatWrite
				}
				if iconButton(c, stop) {
					a.panels.sideChatSending = false
				}
			}
			label := "发送"
			if sending {
				label = "引导"
			}
			tip := label + " · Enter"
			if note := a.sideChatDisabledNote(); note != "" {
				tip = note
			}
			if iconAction(c, iconButtonSpec{
				icon: iconArrowUp, label: label, tip: tip, size: panelSendSize,
				disabled: draft == "" || closed || live,
			}, true) {
				a.sendSideChat()
			}
		})
	})
}

// sideChatDisabledNote 是输入区与发送入口不可用的原因（S-11/S-14）：分支已结束、
// 正在发送、本版没有写入通道三种分开写。空串表示可以输入。
func (a *App) sideChatDisabledNote() string {
	switch {
	case a.panels.sideChatEnded:
		return sideChatEndedNote
	case a.panels.sideChatSending:
		return sideChatSendingNote
	case a.wired():
		return noteSideChatWrite
	}
	return ""
}

// sendSideChat 把草稿发出去。-demo（没有 Source）时这是样例走查的本地回声；
// 连着 Source 时侧聊没有写入通道，这里直接返回：不 append 本地切片、不清草稿，
// 也不在只读时间线上画出那条"看起来已发送"的气泡（输入框与发送已置灰写原因）。
func (a *App) sendSideChat() {
	text := strings.TrimSpace(a.panels.sideChatDraft)
	if text == "" || a.panels.sideChatEnded || a.wired() {
		return
	}
	a.panels.sideChatSent = append(a.panels.sideChatSent, text)
	a.panels.sideChatDraft = ""
}

// sideChatEndDialog 是结束分支的确认框（S-15–S-18）。
//
// 连着 Source 时它不出现：结束分支没有通道，确认后只能改本地布尔值，状态条
// 会写「此分支已结束」而服务端照旧——那是一条假回执。入口已置灰并写原因。
func (a *App) sideChatEndDialog(c *gui.Context) {
	if a.wired() {
		a.panels.sideChatConfirm = false
		return
	}
	t := c.Theme()
	open := a.panels.sideChatConfirm
	m := modalShell(gui.Modal(c, &open, func() {
		gui.Column(c).Gap(space3).Role(gui.RoleAlertDialog).Label(sideChatEndTitle).
			Children(func() {
				typeDialogTitle.style(t, gui.Text(c, sideChatEndTitle)).TextColor(t.Text).
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
	}))

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
