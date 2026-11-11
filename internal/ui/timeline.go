package ui

import (
	"fmt"

	gui "github.com/egoist/mygo/ui"
)

// timeline 是 R3 对话流：内容上限 784 居中，纵向滚动。
// 空态只给一行说明，复用同一区域，不拆页面（U1-ia §4）。
func (a *App) timeline(c *gui.Context) {
	t := c.Theme()
	if a.state == StateEmpty {
		gui.Box(c).Grow(1).Center().Label("对话流").Role(gui.RoleLog).Children(func() {
			gui.Text(c, "还没有消息。写点什么开始，或把文件拖进来。").
				TextColor(t.TextMuted).TextAlign(gui.Center)
		})
		return
	}

	gui.Scroll(c).TrackScroll(&a.feed).Grow(1).
		Label("对话流").Role(gui.RoleLog).Focusable().TabStop(true).Children(func() {
		gui.Column(c).WidthPercent(100).MaxWidth(ContentMaxWidth).
			AlignSelf(gui.Center).Padding(spaceXXL, spaceXL).Gap(spaceL).
			Label("对话内容").Children(func() {
			for i := range a.rounds {
				a.round(c, i)
			}
		})
	})
}

// round 画一轮：用户消息右对齐，助手正文铺开，下面是过程摘要行。
func (a *App) round(c *gui.Context, i int) {
	t := c.Theme()
	r := a.rounds[i]
	gui.Column(c).Gap(spaceS).Children(func() {
		gui.Row(c).Children(func() {
			gui.Spacer(c)
			gui.Text(c, r.user).Selectable().
				Padding(spaceM, spaceL).Radius(radiusRow).
				Background(t.Surface).MaxWidth(userBubbleMaxWidth)
		})
		gui.Column(c).Gap(spaceS).Children(func() {
			reply := r.reply
			if r.status == roundRunning {
				reply += " ▍"
			}
			gui.Text(c, reply).Selectable()
			if r.note != "" {
				gui.Text(c, r.note).TextColor(t.TextMuted).
					FontSize(t.FontSize - smallFontSteps)
			}
			a.summary(c, i)
		})
	})
}

// summary 是过程摘要行：折叠时只给状态与一句摘要，展开后按发生顺序
// 逐条列出操作（UI-11）。状态落在这一行已有的标记与文字上。
func (a *App) summary(c *gui.Context, i int) {
	t := c.Theme()
	r := a.rounds[i]
	open := &a.summaryOpen[i]

	parts := gui.CollapsibleBase(c, open)
	parts.Trigger.Gap(spaceS).AlignItems(gui.Center).
		Padding(spaceXS, 0).Radius(radiusChip).Children(func() {
		gui.RichText(c,
			gui.Span{Text: statusMark(r.status), Color: statusColor(t, r.status)},
			gui.Span{Text: "  " + summaryText(r), Color: summaryColor(t, r.status)},
		)
		gui.Spacer(c)
		gui.Text(c, expandText(*open)).TextColor(t.TextMuted).
			FontSize(t.FontSize - smallFontSteps)
	})
	parts.Panel(func() {
		gui.Column(c).Gap(spaceXXS).PaddingStart(spaceL).Children(func() {
			for n, o := range r.ops {
				opRow(c, n+1, o)
			}
		})
	})
}

// opRow 是展开后的一行操作：序号、动作、对象、结果、耗时。
func opRow(c *gui.Context, n int, o op) {
	t := c.Theme()
	color := t.Text
	if o.failed {
		color = t.TextMuted
	}
	gui.Row(c).Gap(spaceM).AlignItems(gui.Center).Children(func() {
		gui.Text(c, fmt.Sprint(n)).Width(opNumberWidth).TextColor(t.TextMuted).
			FontSize(t.FontSize - smallFontSteps)
		gui.Text(c, o.action).Width(opActionWidth).TextColor(t.TextMuted).
			FontSize(t.FontSize - smallFontSteps)
		gui.Text(c, o.target).Grow(1).SingleLine().TextColor(color).
			FontSize(t.FontSize - smallFontSteps)
		gui.Text(c, o.result).Width(opResultWidth).TextColor(color).
			FontSize(t.FontSize - smallFontSteps)
		gui.Text(c, o.took).Width(opDurationWidth).TextAlign(gui.End).TextColor(t.TextMuted).
			FontSize(t.FontSize - smallFontSteps)
	})
	if o.cause != "" {
		gui.Text(c, "原因："+o.cause).TextColor(t.TextMuted).
			FontSize(t.FontSize - smallFontSteps).PaddingStart(spaceXL)
	}
}

// statusMark 是过程摘要行的行首标记：一个位，按状态取一个。
func statusMark(s roundStatus) string {
	switch s {
	case roundRunning:
		return "●"
	case roundWaiting:
		return "◷"
	case roundFailed:
		return "✕"
	default:
		return "✓"
	}
}

func statusColor(t *gui.Theme, s roundStatus) gui.Color {
	switch s {
	case roundRunning:
		return t.Accent
	case roundWaiting:
		return t.Accent
	case roundFailed:
		return t.Danger
	default:
		return t.TextMuted
	}
}

func summaryColor(t *gui.Theme, s roundStatus) gui.Color {
	if s == roundFailed {
		return t.TextMuted
	}
	return t.Text
}

// summaryText 折叠时只给判断所需：状态、操作数与耗时；文件数给到一轮。
func summaryText(r round) string {
	files := map[string]bool{}
	for _, o := range r.ops {
		if o.action == "改动" {
			files[o.target] = true
		}
	}
	switch r.status {
	case roundRunning:
		return "正在运行 · " + r.took
	case roundWaiting:
		return "等待你同意"
	case roundFailed:
		return "失败 · " + r.took
	default:
		return fmt.Sprintf("完成 · %d 个操作 · 改了 %d 个文件 · %s",
			len(r.ops), len(files), r.took)
	}
}

func expandText(open bool) string {
	if open {
		return "收起"
	}
	return "展开"
}
