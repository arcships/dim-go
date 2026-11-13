package ui

import (
	"fmt"
	"math"
	"strconv"
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
		// 与时间线、输入卡之间靠面差与留白分开，不画上下两条线（UI-18）。
		Background(t.Surface).
		Label("队列").Children(func() {
		// 排队只有形状与内容（UI-23）：时钟图标说明"还没轮到"，内容是用户原话，
		// 撤回是一个图标入口；屏幕上不写「排队中」「撤回」。
		iconBox(c, iconClock, t.TextMuted, "")
		typeSecondary.style(t, gui.Text(c, "把日志也开上")).Grow(1).SingleLine()
		iconButton(c, iconButtonSpec{icon: iconUndo, label: "撤回排队", size: rowCompact})
	})
}

// inputCard 是输入卡片：附件块与用量、可编辑正文、附件入口与发送槽。
// 卡片靠"面填充 + 留白"与背景分开，不画描边（UI-18：边界只有一套，能靠留白
// 分开就不加线）；编辑器用 TextAreaBase，不再叠第二道边。模型不在输入区：
// 默认值不上屏，偏离默认时只在标题行末尾留一个淡标记（UI-9）。上下文用量
// 在这一块里（UI-15）。
func (a *App) inputCard(c *gui.Context) {
	t := c.Theme()
	empty := strings.TrimSpace(a.draft) == ""
	running := a.state == StateRunning
	// 卡片内边距取 space-2：与外层 8 相加等于标题行与时间线的 16，四处左缘同一条
	// （UI-22、U2 §4.7 的基线表）。
	gui.Column(c).MinHeight(ComposerHeight).MaxHeight(ComposerMaxHeight).
		Padding(space2).Gap(space1).
		Background(t.Surface).Radius(radius).
		Children(func() {
			gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
				if a.state == StateEmpty {
					return
				}
				chip(c, "发送路径.md")
				gui.Spacer(c)
				a.contextMeter(c)
			})
			a.contextDetail(c)
			// 空输入框给一个两字的最小占位（R6 裁决：占位与"默认聚焦"只选一种，
			// 选占位——它在截图与静态评审里也可见）。首字键入即消失；读屏名仍是
			// 「消息」，键位提示挂在槽位的悬停提示上。
			gui.TextAreaBase(c, &a.draft).
				Placeholder(placeholderInput).
				Grow(1).MinHeight(composerTextMinHeight).Label("消息")
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
				gui.Spacer(c)
				iconButton(c, iconButtonSpec{icon: iconPaperclip, label: "附件", size: rowCompact})
				a.sendSlot(c, running, empty)
			})
		})
}

// contextMeter 是输入卡内的上下文用量（UI-15）：属于输入区的一块，不在标题行、
// 也不另立状态条。数值是常驻形态；点数值（或用键盘 Enter/Space）在数值下方展开
// 一行明细。数值只是样例，不来自任何服务。
//
// 将满（F8）用两条非文字通道：这处数值换 `warning` 色，并在数值前多一个 6px
// 实心圆点。颜色之外必须有形状（§9.0 D4 与"可读可用下限"），色觉障碍下才分得出；
// 不加「快满了」这类词（§9.0 A2/D2）。
func (a *App) contextMeter(c *gui.Context) {
	t := c.Theme()
	percent := a.contextPercent()

	parts := gui.CollapsibleBase(c, &a.contextOpen)
	tr := parts.Trigger.Padding(space1, space2).Radius(radius).
		Cursor(gui.CursorPointer).Label(fmt.Sprintf("上下文 %d%%", percent))
	// 与组头同一套态：默认透明、悬停 surface-hover、按下 surface-pressed、
	// 聚焦环由框架画；用量指示不设禁用态（断线时只是颜色变次要）。
	switch {
	case tr.Pressed():
		tr.Background(t.SurfacePressed)
	case tr.Hovered():
		tr.Background(t.SurfaceHover)
	}
	tr.Children(func() {
		color := t.TextMuted
		if percent >= contextNearPercent {
			color = t.Warning
			nearDot(c, color)
		}
		// 形状 + 数值（UI-23 第 1、3 档）：环形图标说明"占用"，百分比是精确值；
		// 屏幕上不写「上下文」。
		iconBox(c, iconGauge, color, "")
		typeSecondary.style(t, gui.Text(c, fmt.Sprintf("%d%%", percent))).
			TextColor(color)
	})
}

// nearDot 是"将满"的非颜色通道：6px 实心圆，尺寸取图标体系里的最小档（U2 §4.5）。
// 它只在将满时出现，不承载文字；读屏名给出这层含义，数值由用量指示自己报。
func nearDot(c *gui.Context, color gui.Color) {
	box := gui.Box(c).Width(dotContext).Height(dotContext).Shrink(0).
		Label("上下文将满")
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, color, dotContext/2)
	})
}

// contextDetail 是点开用量后的一行明细：只在展开时出现，右对齐贴在数值下方。
func (a *App) contextDetail(c *gui.Context) {
	if !a.contextOpen || a.state == StateEmpty {
		return
	}
	t := c.Theme()
	typeSecondary.style(t, gui.Text(c, a.contextDetailText())).
		TextColor(t.TextMuted).TextAlign(gui.End)
}

// contextPercent 是样例用量的百分比：只做除法与取整，不推算别的口径。
func (a *App) contextPercent() int {
	if a.contextLimit <= 0 {
		return 0
	}
	return int(math.Round(float64(a.contextUsed) * 100 / float64(a.contextLimit)))
}

// contextDetailText 是明细的文字：已用与上限来自同一处样例值。
func (a *App) contextDetailText() string {
	return fmt.Sprintf("已用 %s / %s tokens · 本会话上下文",
		tokenCount(a.contextUsed), tokenCount(a.contextLimit))
}

// tokenCount 把一个 token 数写成紧凑形式（24600 → 24.6k）。
func tokenCount(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	s := strconv.FormatFloat(float64(n)/1000, 'f', 1, 64)
	return strings.TrimSuffix(s, ".0") + "k"
}


// sendSlot 是 R4 的发送槽：同一位置、同一尺寸只放一个动作（UI-1、U3 §4.1）。
//
// 形式按控件选择（UI-23、§9.0「形式的选择」）：发送/停止是高频动作、词形成熟
// （向上箭头 / 方块）、空间在行末——取**纯图标 + tooltip**，tooltip 同时给名称
// 与快捷键，读屏名给「发送」「停止」。结果不确定与需人工确认期间禁用。
func (a *App) sendSlot(c *gui.Context, running, empty bool) {
	icon, label, tip := iconArrowUp, "发送", "发送 · Enter"
	disabled := empty
	switch {
	case running:
		icon, label, tip = iconStop, "停止", "停止 · Enter"
		disabled = false
	case a.state == StateFailed:
		// 结果不确定 / 需人工确认：不提供重发入口，槽位禁用（U3 §6.1）。
		disabled = true
	}
	iconAction(c, iconButtonSpec{icon: icon, label: label, tip: tip, size: rowCompact, disabled: disabled}, true)
}

// approvalCard 是审批卡：出现时占据输入位，输入区不存在（UI-1）。
// 它是唯一需要硬分开的自绘面（一个待你决定的事务），因此允许第三档边界：1px
// 主题 Border。不画第二级边框色、不叠阴影（UI-18）。拿不到命令原文时不提供
// “始终允许”（U3 §5.3）。
func (a *App) approvalCard(c *gui.Context) {
	t := c.Theme()
	cmd := a.approvalCommand
	gui.Column(c).MinHeight(ApprovalHeight).
		Padding(space2).Gap(space3).
		Background(t.Surface).Border(strokeHairline, t.Border).Radius(radius).
		Children(func() {
			// 卡的身份由形状与内容说（UI-23）：时钟图标说明"在等你决定"，命令原文
			// 是内容；不写「需要你同意」这类界面语。拿不到命令原文时只剩图标与三个
			// 决定，读屏名仍给「需要你决定」。
			iconBox(c, iconClock, colorAccentInk.in(t), "需要你决定")
			if cmd != "" {
				// 命令原文与影响分成两块：命令是一段代码（自己的面与内边距），影响是
				// 下面一行说明。两者不再粘在一行，扫一眼就能分清哪是命令、哪是影响。
				gui.Box(c).Shrink(0).Padding(space1, space2).Radius(radius).
					Background(t.SurfacePressed).Children(func() {
					typeCode.style(t, gui.RichText(c,
						gui.Span{Text: cmd, Size: typeCode.sizeIn(t),
							Weight: typeCode.weight, Color: t.Text}))
				})
				typeSecondary.style(t, gui.Text(c, "会读写工作区内的临时文件")).
					TextColor(t.TextMuted).Label("影响：会读写工作区内的临时文件")
			}
			gui.Spacer(c)
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
				// 三个决定是"图标 + 文字"（UI-23 / §9.0 选择表：高后果、不可逆、
				// 同族多选 → 识别与确认同时成立）。
				if textIconButton(c, textIconButtonSpec{
					icon: iconCheck, text: "允许一次", primary: true,
					tip: "只同意这一次",
				}) {
					a.approvalDecision = "allow"
				}
				if cmd != "" {
					// 「一直允许」会对所有会话生效：第一次点击换成确认态，再点才生效。
					text, label, tip := "一直允许", "一直允许", "始终允许只对简单命令有效"
					if a.confirmAlways {
						text, label = "确认一直允许", "确认一直允许（对所有会话生效）"
						tip = "再点一次：对所有会话写入这条命令规则"
					}
					if textIconButton(c, textIconButtonSpec{
						icon: iconCheckLock, text: text, label: label, tip: tip,
					}) {
						if a.confirmAlways {
							a.approvalDecision = "allow-always"
							a.confirmAlways = false
						} else {
							a.confirmAlways = true
						}
					}
				}
				if textIconButton(c, textIconButtonSpec{
					icon: iconXMark, text: "拒绝", tip: "不改规则，这次不执行",
				}) {
					a.approvalDecision = "deny"
				}
			})
		})
}
