package ui

import (
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// composer 是 R4 输入位：同一位置只放一件事（写 / 排队 / 回答）。
// 队列与不可用原因贴在输入区上沿，不放到标题行或时间线顶部（UI-10）。
func (a *App) composer(c *gui.Context) {
	gui.Column(c).Shrink(0).Children(func() {
		if a.state == StateRunning {
			a.queueBar(c)
		}
		gui.Box(c).Padding(composerPadding).Children(func() {
			if a.state == StateWaiting {
				a.approvalCard(c)
				return
			}
			a.inputCard(c)
		})
	})
}

// queueBar 是上沿的当前值行：运行中提交先入队，紧贴输入区。
func (a *App) queueBar(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Height(QueueHeight).Shrink(0).Padding(0, space4).
		Gap(space2).AlignItems(gui.Center).
		Background(t.Surface).
		BorderWidth(strokeHairline, 0, strokeHairline, 0).
		BorderColor(t.Border).
		Label("队列").Children(func() {
		typeSecondary.style(t, gui.Text(c, "排队中 1")).TextColor(t.TextMuted)
		typeSecondary.style(t, gui.Text(c, "把日志也开上")).Grow(1).SingleLine()
		actionButton(c, "撤回", "撤回排队", false, false)
	})
}

// inputCard 是输入卡片：附件块、可编辑正文、附件入口与发送槽。
// 模型不在这里：取默认值时不上屏，偏离默认时只在标题行末尾留一个淡标记（UI-9）。
func (a *App) inputCard(c *gui.Context) {
	t := c.Theme()
	empty := strings.TrimSpace(a.draft) == ""
	running := a.state == StateRunning
	gui.Column(c).MinHeight(ComposerHeight).MaxHeight(ComposerMaxHeight).
		Padding(space3).Gap(space1).
		Background(t.Surface).Border(strokeHairline, colorBorderStrong.in(t)).Radius(radius).
		Children(func() {
			gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
				if a.state == StateEmpty {
					return
				}
				chip(c, "发送路径.md")
			})
			typeBody.style(t, gui.TextArea(c, &a.draft).
				Placeholder("写点什么，或把文件拖进来")).
				Grow(1).MinHeight(composerTextMinHeight).Label("消息")
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
				// 运行中 Enter 是排队，空闲时 Enter 发送（U3 K1）。
				hint := "↵ 发送"
				if running {
					hint = "↵ 排队"
				}
				typeSecondary.style(t, gui.Text(c, hint)).TextColor(t.TextMuted)
				gui.Spacer(c)
				actionButton(c, "＋ 附件", "附件", false, false)
				a.sendSlot(c, running, empty)
			})
		})
}

// sendSlot 是 R4 的发送槽：同一位置、同一尺寸只放一个主按钮。
// 空闲是"发送"，运行中是"停止"，两者不并存；审批卡占位时整个输入卡不存在，
// 槽也随之消失（UI-1）。
func (a *App) sendSlot(c *gui.Context, running, empty bool) {
	text, label := "发送", "发送"
	if running {
		text, label = "停止", "停止"
	}
	actionButton(c, text, label, true, !running && empty).MinWidth(sendSlotWidth)
}

// approvalCard 是审批卡：出现时占据输入位，输入区不存在（UI-1）。
// 卡面 surface、1px accent 描边、圆角 6、内边距 16（U2 §4.1）。
// 拿不到命令原文时不提供“始终允许”（U3 §5.3：不给看不清作用域的按钮）。
func (a *App) approvalCard(c *gui.Context) {
	t := c.Theme()
	cmd := a.approvalCommand
	gui.Column(c).MinHeight(ApprovalHeight).
		Padding(space4).Gap(space3).
		Background(t.Surface).Border(strokeHairline, t.Accent).Radius(radius).
		Children(func() {
			typeBodyStrong.style(t, gui.Text(c, "需要你同意：跑一条命令"))
			if cmd != "" {
				gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
					typeCode.style(t, gui.RichText(c,
						gui.Span{Text: cmd, Size: typeCode.sizeIn(t),
							Weight: typeCode.weight, Color: t.Text}))
					typeSecondary.style(t, gui.Text(c, "｜  影响：工作区内的临时文件")).
						TextColor(t.TextMuted)
				})
			}
			gui.Spacer(c)
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
				actionButton(c, "允许一次", "允许一次", true, false)
				if cmd != "" {
					actionButton(c, "始终允许", "始终允许", false, false)
				}
				actionButton(c, "拒绝", "拒绝", false, false)
				if cmd != "" {
					typeSecondary.style(t, gui.Text(c, "始终允许只对简单命令有效")).
						TextColor(t.TextMuted)
				}
			})
		})
}
