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
	gui.Row(c).Height(QueueHeight).Shrink(0).Padding(0, spaceXL).
		Gap(spaceM).AlignItems(gui.Center).
		Background(t.Surface).BorderWidth(hairline, 0, hairline, 0).BorderColor(t.Border).
		Label("队列").Children(func() {
		gui.Text(c, "排队中 1").TextColor(t.TextMuted).
			FontSize(t.FontSize - smallFontSteps)
		gui.Text(c, "把日志也开上").Grow(1).SingleLine().
			FontSize(t.FontSize - smallFontSteps)
		gui.Button(c, "撤回").Label("撤回排队").Focusable()
	})
}

// inputCard 是输入卡片：附件条目、可编辑正文、发送与模型入口。
func (a *App) inputCard(c *gui.Context) {
	t := c.Theme()
	empty := strings.TrimSpace(a.draft) == ""
	gui.Column(c).Height(ComposerHeight).
		Padding(spaceM, spaceL).Gap(spaceS).
		Background(t.Surface).Border(hairline, t.Border).Radius(radiusCard).Children(func() {
		gui.Row(c).Gap(spaceS).AlignItems(gui.Center).Children(func() {
			if a.state == StateEmpty {
				return
			}
			gui.Badge(c, "发送路径.md")
			gui.Text(c, "附件 1").TextColor(t.TextMuted).
				FontSize(t.FontSize - smallFontSteps)
		})
		gui.TextArea(c, &a.draft).Placeholder("写点什么，或把文件拖进来").
			Grow(1).MinHeight(composerTextMinHeight).Label("消息")
		gui.Row(c).Gap(spaceM).AlignItems(gui.Center).Children(func() {
			gui.Text(c, "⌘↵ 发送").TextColor(t.TextMuted).
				FontSize(t.FontSize - smallFontSteps)
			gui.Spacer(c)
			gui.Button(c, "＋ 附件").Label("附件")
			gui.Button(c, "模型：默认 ▾").Label("模型")
			if a.state == StateRunning {
				gui.PrimaryButton(c, "停止").Label("停止")
				return
			}
			gui.PrimaryButton(c, "发送").Label("发送").Disabled(empty)
		})
	})
}

// approvalCard 是审批卡：出现时占据输入位，输入区不存在（UI-1）。
func (a *App) approvalCard(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Height(ApprovalHeight).
		Padding(spaceM, spaceL).Gap(spaceS).
		Background(t.Surface).Border(hairline, t.Accent).Radius(radiusCard).Children(func() {
		gui.Text(c, "需要你同意：跑一条命令").Bold()
		gui.RichText(c,
			gui.Span{Text: "rm -rf ./tmp/*", Color: t.Text},
			gui.Span{Text: "  ｜  影响：工作区内的临时文件", Color: t.TextMuted},
		)
		gui.Spacer(c)
		gui.Row(c).Gap(spaceM).AlignItems(gui.Center).Children(func() {
			gui.PrimaryButton(c, "允许一次").Label("允许一次")
			gui.Button(c, "一直允许这条命令").Label("一直允许这条命令")
			gui.Button(c, "拒绝").Label("拒绝")
			gui.Text(c, "一直允许只对简单命令有效").TextColor(t.TextMuted).
				FontSize(t.FontSize - smallFontSteps)
		})
	})
}
