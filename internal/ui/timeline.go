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
			typeBody.style(t, gui.Text(c, "还没有消息。写点什么开始，或把文件拖进来。")).
				TextColor(t.TextMuted).TextAlign(gui.Center)
		})
		return
	}

	gui.Scroll(c).TrackScroll(&a.feed).Grow(1).
		Label("对话流").Role(gui.RoleLog).Focusable().TabStop(true).Children(func() {
		gui.Column(c).WidthPercent(100).MaxWidth(ContentMaxWidth).
			AlignSelf(gui.Center).Padding(space6, space4).Gap(space6).
			Label("对话内容").Children(func() {
			for i := range a.rounds {
				a.round(c, i)
			}
		})
	})
}

// round 画一轮：用户消息右对齐，助手正文铺开，下面是过程摘要行。
// 轮次之间 21，轮内块之间 10.5（U2 §2.3）。
func (a *App) round(c *gui.Context, i int) {
	t := c.Theme()
	r := a.rounds[i]
	gui.Column(c).Gap(space3).Children(func() {
		gui.Row(c).Children(func() {
			gui.Spacer(c)
			// 气泡外观画在自绘 Box 上；文本只负责字号与可选。
			gui.Box(c).MaxWidth(userBubbleMaxWidth).
				Padding(space2, space3).Radius(radius).
				Background(t.Surface).Children(func() {
				typeBody.style(t, gui.Text(c, r.user)).Selectable()
			})
		})
		gui.Column(c).Gap(space2).Children(func() {
			a.reply(c, r)
			if r.note != "" {
				a.note(c, r.note)
			}
			a.summary(c, i)
		})
	})
}

// reply 是助手正文。运行中在行尾给 2×14 的 accent-ink 光标（U2 §4.1、§5.3）。
func (a *App) reply(c *gui.Context, r round) {
	t := c.Theme()
	if r.status != roundRunning {
		typeBody.style(t, gui.Text(c, r.reply)).Selectable()
		return
	}
	gui.RichText(c,
		gui.Span{Text: r.reply, Size: typeBody.sizeIn(t), Weight: typeBody.weight, Color: t.Text},
		gui.Span{Text: "\u258d", Size: typeBody.sizeIn(t), Color: colorAccentInk.in(t)},
	).Selectable()
}

// note 是本轮下方的一行说明（结果不确定）：行首 16px 状态位 + 一行正文，
// 说明属于本轮，不是容器（U2 §4.1）。
func (a *App) note(c *gui.Context, note string) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
		iconBox(c, iconWarn, t.Warning, "结果不确定")
		typeBody.style(t, gui.Text(c, note)).Selectable()
	})
}

// summary 是过程摘要行：折叠时只给此刻状态的判断（UI-11），条数、文件数与
// 耗时全部进展开；展开后按发生顺序逐条列出操作。状态落在这一行已有的标记
// 与文字上。
func (a *App) summary(c *gui.Context, i int) {
	t := c.Theme()
	r := a.rounds[i]
	open := &a.summaryOpen[i]

	parts := gui.CollapsibleBase(c, open)
	parts.Trigger.Gap(space1).AlignItems(gui.Center).
		Padding(space1, 0).Radius(radius).Children(func() {
		statusMark(c, t, r.status)
		typeSecondaryStrong.style(t, gui.Text(c, summaryStatus(r.status))).
			TextColor(summaryColor(t, r.status))
		gui.Spacer(c)
		typeSecondary.style(t, gui.Text(c, expandText(*open))).TextColor(t.TextMuted)
	})
	parts.Panel(func() {
		gui.Column(c).Gap(space1).PaddingStart(space4).Children(func() {
			typeSecondary.style(t, gui.Text(c, summaryMeta(r))).TextColor(t.TextMuted)
			for n, o := range r.ops {
				opRow(c, n+1, o)
			}
		})
	})
}

// opRow 是展开后的一行操作：序号、动作、对象、结果、耗时。失败行只把结果词
// 标成 danger，其余文字保持正文色（U2 §4.1：失败不把正文染红）。
func opRow(c *gui.Context, n int, o op) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
		// 行内编号用极小级（U2 §1.1）。
		typeMicro.style(t, gui.Text(c, fmt.Sprint(n))).Width(opNumberWidth).
			TextColor(t.TextMuted)
		typeSecondary.style(t, gui.Text(c, o.action)).Width(opActionWidth).
			TextColor(t.TextMuted)
		typeSecondary.style(t, gui.Text(c, o.target)).Grow(1).SingleLine().
			TextColor(t.Text)
		typeSecondary.style(t, gui.Text(c, o.result)).Width(opResultWidth).
			TextColor(resultColor(t, o))
		typeSecondary.style(t, gui.Text(c, o.took)).Width(opDurationWidth).
			TextAlign(gui.End).TextColor(t.TextMuted)
	})
	if o.cause != "" {
		typeSecondary.style(t, gui.Text(c, "原因："+o.cause)).TextColor(t.TextMuted).
			PaddingStart(space4)
	}
}

// statusMark 是过程摘要行的行首状态位（U2 §4.1、§6.4、校准 C5）：
// 固定 16×16 盒，尺寸不随字号变化；运行中是居中的 8px 实心圆，其余是
// Heroicons outline 图标（24 网格 1.5 描边 → 16px 时 1.0px）。
func statusMark(c *gui.Context, t *gui.Theme, s roundStatus) {
	if s == roundRunning {
		box := gui.Box(c).Width(iconInline).Height(iconInline).Shrink(0).
			Label("状态：正在运行")
		box.DrawOver(func(p *gui.Painter, r gui.Rect) {
			p.Fill(gui.Rect{
				X: r.X + (iconInline-dotStatus)/2,
				Y: r.Y + (iconInline-dotStatus)/2,
				W: dotStatus, H: dotStatus,
			}, colorAccentInk.in(t), dotStatus/2)
		})
		return
	}
	switch s {
	case roundWaiting:
		iconBox(c, iconClock, colorAccentInk.in(t), "状态：等待你同意")
	case roundFailed:
		iconBox(c, iconXMark, t.Danger, "状态：失败")
	default:
		iconBox(c, iconCheck, t.TextMuted, "状态：完成")
	}
}

// resultColor 只给失败的结果词上 danger，其余结果词取正文色。
func resultColor(t *gui.Theme, o op) gui.Color {
	if o.failed {
		return t.Danger
	}
	return t.Text
}

// summaryStatus 是折叠态唯一保留的判断：此刻是什么状态（UI-11）。
func summaryStatus(s roundStatus) string {
	switch s {
	case roundRunning:
		return "正在运行"
	case roundWaiting:
		return "等待你同意"
	case roundFailed:
		return "失败"
	default:
		return "完成"
	}
}

// summaryColor 让失败行整行取 text-secondary，其余状态取正文色（U2 §4.1）。
func summaryColor(t *gui.Theme, s roundStatus) gui.Color {
	if s == roundFailed {
		return t.TextMuted
	}
	return t.Text
}

// summaryMeta 是展开后的第一行：操作数、文件数与耗时只在这里出现，
// 折叠态不显示（UI-11）。
func summaryMeta(r round) string {
	files := map[string]bool{}
	for _, o := range r.ops {
		if o.action == "改动" {
			files[o.target] = true
		}
	}
	meta := fmt.Sprintf("%d 个操作", len(r.ops))
	if len(files) > 0 {
		meta += fmt.Sprintf(" · 改了 %d 个文件", len(files))
	}
	switch {
	case r.status == roundRunning && r.took != "" && r.took != "—":
		meta += " · 已用 " + r.took
	case r.status != roundWaiting && r.took != "" && r.took != "—":
		meta += " · 耗时 " + r.took
	}
	return meta
}

func expandText(open bool) string {
	if open {
		return "收起"
	}
	return "展开"
}
