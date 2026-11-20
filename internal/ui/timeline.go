package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 时间线按参考产品的聊天时间线重建：用户气泡右对齐、助手块左对齐；思考块、
// 工具行、工具分组、文件改动、产物与压缩分隔都在同一条内容列里。块与块之间
// 只靠间距区分，不画边框（UI-18）。
//
// 动效取运行时实测值（coverage-matrix §5）：折叠高度 0.3s
// cubic-bezier(0.16, 1, 0.3, 1)，箭头 0.15s cubic-bezier(0.4, 0, 0.2, 1)。
// 减少动效时框架的 AnimateTo 直接给终值，布局与文字不变（UI-5）。

const (
	// timelinePadX 是时间线内容左右内边距（参考 px-4 = 14），与输入卡同一左缘。
	timelinePadX = space4
	// timelineRoundGap 是轮与轮之间的距离；轮内块间距更小（A3）。
	timelineRoundGap = space6
	timelineBlockGap = space4
	// toolRowHeight 是工具行/思考行/分组头的高度（参考 h-[21px]）。
	timelineRowHeight = 21
	// userBubbleMaxPercent 是用户气泡的最大宽度（内容列的百分比）。
	userBubbleMaxPercent = 85
	// timelineBackThreshold 是「回到最新」出现的滚动距离阈值。
	timelineBackThreshold = 120
	// timelineGroupIndent 是分组内容相对分组头的缩进（参考 16）。
	timelineGroupIndent = iconInline
)

var (
	// timelineCollapse 是折叠容器的高度过渡：实测 0.3s / cubic-bezier(0.16,1,0.3,1)。
	timelineCollapse = gui.Animation{
		Duration: 300 * time.Millisecond,
		Ease:     gui.CubicBezier(0.16, 1, 0.3, 1),
	}
	// timelineArrow 是展开箭头的旋转：实测 0.15s / cubic-bezier(0.4,0,0.2,1)。
	timelineArrow = gui.Animation{
		Duration: 150 * time.Millisecond,
		Ease:     gui.CubicBezier(0.4, 0, 0.2, 1),
	}
)

// 工具行/分组头/思考块用到的形状：参考产品用 lucide 的灯泡、工作流、地球、
// 文件与分叉，这里按同一 24 网格、1.5 描边自绘，避免引素材（UI-21）。
var (
	iconBulb     = mustSVG(`<path d="M9 18h6M10.5 21h3M12 3a6 6 0 0 0-3.75 10.688c.471.353.75.906.75 1.487V16h6v-.825c0-.581.279-1.134.75-1.487A6 6 0 0 0 12 3Z"/>`)
	iconWorkflow = mustSVG(`<rect x="3.75" y="3.75" width="7.5" height="7.5" rx="1.5"/><rect x="12.75" y="3.75" width="7.5" height="7.5" rx="1.5"/><rect x="3.75" y="12.75" width="7.5" height="7.5" rx="1.5"/><path d="M16.5 12.75v7.5M12.75 16.5h7.5"/>`)
	iconGlobe    = mustSVG(`<circle cx="12" cy="12" r="8.25"/><path d="M3.75 12h16.5M12 3.75c2.1 2.4 3.15 5.1 3.15 8.25S14.1 18.6 12 20.25c-2.1-1.65-3.15-5.1-3.15-8.25S9.9 6.15 12 3.75Z"/>`)
	iconCopy     = mustSVG(`<rect x="9" y="9" width="10.5" height="10.5" rx="2"/><path d="M6.75 15H5.25A1.5 1.5 0 0 1 3.75 13.5V5.25a1.5 1.5 0 0 1 1.5-1.5h8.25a1.5 1.5 0 0 1 1.5 1.5v1.5"/>`)
	iconFork     = mustSVG(`<circle cx="6.75" cy="6" r="2.25"/><circle cx="6.75" cy="18" r="2.25"/><circle cx="17.25" cy="9.75" r="2.25"/><path d="M6.75 8.25v7.5M17.25 12a5.25 5.25 0 0 1-5.25 5.25H9"/>`)
	iconRetry    = mustSVG(`<path d="M4.5 12a7.5 7.5 0 1 0 2.2-5.303M4.5 4.5V9h4.5"/>`)
	iconFilePlus = mustSVG(`<path d="M14.25 3.75H6.75a1.5 1.5 0 0 0-1.5 1.5v13.5a1.5 1.5 0 0 0 1.5 1.5h10.5a1.5 1.5 0 0 0 1.5-1.5V7.5z"/><path d="M14.25 3.75V7.5h4.5M12 11.25v6M9 14.25h6"/>`)
	iconImage    = mustSVG(`<rect x="3.75" y="4.5" width="16.5" height="15" rx="2.25"/><circle cx="9" cy="9.75" r="1.5"/><path d="m3.75 16.5 4.5-4.5 4.5 4.5 3-3 4.5 4.5"/>`)
)

// timeline 是对话流：连着 Source 时只画它给的快照（不掺样例），否则画样例。
func (a *App) timeline(c *gui.Context) {
	if snap, ok := a.liveSession(); ok {
		a.liveTimeline(c, snap)
		return
	}
	if a.wired() {
		a.emptyTimeline(c)
		return
	}
	if a.state == StateEmpty {
		a.emptyTimeline(c)
		return
	}

	gui.Box(c).Grow(1).Children(func() {
		feed := a.timelineFeed(c)
		feed.Children(func() {
			gui.Column(c).Padding(space6, timelinePadX).Gap(timelineRoundGap).
				Label("对话内容").Children(func() {
				gui.Row(c).Shrink(0).Children(func() {
					iconButton(c, iconButtonSpec{icon: iconChevronUp, label: "读取更早的记录", size: rowCompact})
				})
				for i := range a.rounds {
					a.round(c, i)
				}
			})
		})
		a.backToLatest(c)
	})
}

// timelineFeed 是时间线的滚动容器：与搜索关闭时的焦点归还共用一处。
// 会话内搜索关掉后，焦点回到这条流上（A1-05 的 R-GS-052）。
func (a *App) timelineFeed(c *gui.Context) gui.Element {
	// TL-01 对话流滚动区容器
	feed := gui.Scroll(c).Grow(1).Fill().TrackScroll(&a.feed).
		Label("对话流").Role(gui.RoleLog).Focusable().TabStop(true)
	if a.sessionSearch.returnFocus {
		feed = feed.Focus()
		a.sessionSearch.returnFocus = false
	}
	return feed
}

// round 画一轮：用户消息右对齐，下面是助手区（思考、正文、工具分组、轮末块）。
func (a *App) round(c *gui.Context, i int) {
	r := a.rounds[i]
	if r.unknown {
		// 未知提交：只有占位（原文还在本地）与说明行；没有组头、没有操作行，
		// 因为「这一轮是否存在」本身还没有事实。
		gui.Column(c).Shrink(0).Gap(space2).Children(func() {
			a.userBubble(c, i, r, true)
			a.note(c, r.note)
		})
		return
	}
	// TL-07 / TL-11 一轮的行外壳（轮容器）
	gui.Column(c).Shrink(0).Gap(timelineBlockGap).Children(func() {
		// TL-04 / TL-05 用户消息轨道与标记（每条用户消息一个标记）
		a.userRail(c, strings.TrimSpace(r.user) != "")
		if r.divider != "" {
			a.compressionRow(c, compressionRowKey(i), r.divider, "")
		}
		a.userMessage(c, i, r)
		a.assistant(c, i, r)
	})
}

// roundDivider 是轮前的分隔行（压缩 / 窗口截断）：两侧细线 + 中间说明。
func (a *App) roundDivider(c *gui.Context, text string) {
	// TL-16 / TL-66 窗口截断分隔行
	t := c.Theme()
	gui.Row(c).Grow(1).Gap(space2).AlignItems(gui.Center).Label(text).Children(func() {
		gui.Box(c).Grow(1).Height(strokeHairline).Background(t.Border)
		typeMicro.style(t, gui.Text(c, text)).TextColor(t.TextMuted).SingleLine()
		gui.Box(c).Grow(1).Height(strokeHairline).Background(t.Border)
	})
}

// 消息动作没有写入口时写进读屏名、提示与菜单项的原因。分叉按 fork.go 走
// Intents.CreateBranch（以这一轮结束处为切点）；拿不到轮次时置灰并写原因，
// 不拿别的轮次顶替（DEC-05）。删除与撤回同样没有消息级写入通道。
const (
	messageDeleteUnavailable   = noteMessageDelete
	messageWithdrawUnavailable = noteMessageDelete
	messageCopyNeedsSelection  = "先选中要复制的文本"
)

// userMessage 是用户消息行：气泡右对齐；悬停时在气泡下方出现发送时间、动作行
// 与不可用动作的原因；选中正文时出现选区浮条（TL-44）。编辑中换成行内编辑器。
func (a *App) userMessage(c *gui.Context, i int, r round) {
	if a.inlineEditRow == i {
		a.inlineEditor(c, i, r)
		return
	}
	var edited bool
	if text, ok := a.inlineSaved[i]; ok {
		r.user, edited = text, true
	}
	col := gui.Column(c).Shrink(0).Gap(space1).
		Role(gui.RoleArticle).Label("用户消息 " + r.user)
	// timelineHover 是截图走查态：自动截图没有指针，靠它把悬停动作钉住。
	hovered := col.Hovered() || a.timelineHover == i
	col.Children(func() {
		body := a.userBubble(c, i, r, false)
		if edited {
			gui.Row(c).Grow(1).Gap(space1).AlignItems(gui.Center).Children(func() {
				gui.Spacer(c)
				typeMicro.style(c.Theme(), gui.Text(c, "已编辑")).TextColor(c.Theme().TextMuted).
					Label("已编辑")
			})
		}
		if selection := selectedTextOf(body); selection != "" {
			// ⌘F 打开会话内搜索时预填这段选中文本（A1-03）。
			a.timelineSelected = selection
			a.selectionBar(c, selection)
		}
		// 超过折叠行数时动作行常驻：里面左起第一项就是「展示全部 / 收起」（TL-31）。
		if hovered || bubbleOverflows(r.user) {
			a.userActions(c, i, r, hovered)
		}
	})
}

// userBubble 是气泡本体：右对齐、最多 85% 内容宽、浅底圆角；附件、正文同属
// 一个可选区，右键打开消息菜单（TL-41–TL-43）。正文超过折叠行数时底部给
// 「展示全部 / 收起」（TL-31）。
func (a *App) userBubble(c *gui.Context, i int, r round, pending bool) gui.Element {
	// TL-20 / TL-26 用户气泡容器与正文
	t := c.Theme()
	var box gui.Element
	gui.Row(c).Grow(1).Children(func() {
		gui.Spacer(c)
		// 气泡与展开入口同在一个限宽的列里：展开入口排在气泡下方，不参与
		// 可选区，也不会被 85% 的内容宽挤出父容器。
		expanded := a.userExpandedOf(i)
		overflow := bubbleOverflows(r.user)
		gui.Column(c).Shrink(0).MaxWidthPercent(userBubbleMaxPercent).
			AlignItems(gui.End).Gap(space1).Children(func() {
			box = gui.Box(c).WidthPercent(100).
				Padding(space2, timelinePadX).Radius(radiusCard).
				Background(t.SurfaceHover).Label("我的消息").Selectable()
			if pending {
				box = box.Opacity(0.6)
			}
			box.Children(func() {
				a.userAttachmentArea(c, r)
				if r.user != "" {
					body := a.userBody(c, i, r.user, overflow, expanded)
					if !pending {
						// 菜单挂在正文上：可选中文本自带编辑菜单，ContextMenu 在
						// 它上面才会替换掉那条（选区仍由外层可选区持有）。
						svc := c.Services()
						body = body.ContextMenu(func(m *gui.Menu) {
							a.userMessageMenu(m, svc, i, r, selectedTextOf(box))
						})
					}
				}
			})
		})
	})
	a.searchReveal(c, box, i, searchScopeUser)
	return box
}

// userActions 是用户气泡下的发送时间与四个动作（悬停可见）：复制走剪贴板，
// 编辑进入行内编辑器（TL-38），侧边对话打开右栏；分叉没有写入口，置灰并写原因。
func (a *App) userActions(c *gui.Context, i int, r round, hovered bool) {
	// TL-35 发送时间
	t := c.Theme()
	overflow := bubbleOverflows(r.user)
	gui.Column(c).Grow(1).Gap(space1).Label("消息动作").Children(func() {
		gui.Row(c).Grow(1).Gap(space1).AlignItems(gui.Center).Children(func() {
			if overflow {
				a.bubbleExpandButton(c, i, a.userExpandedOf(i))
			}
			if !hovered {
				return
			}
			gui.Spacer(c)
			if r.at != "" {
				typeMicro.style(t, gui.Text(c, r.at)).TextColor(t.TextMuted).
					Label("发送时间 " + r.at).SingleLine()
			}
			if strings.TrimSpace(r.user) != "" {
				if iconButton(c, iconButtonSpec{
					// TL-37 复制整个消息
					icon: iconCopy, label: "复制整个消息 " + fmt.Sprint(i+1),
					tip: "复制整个消息", size: rowCompact,
				}) {
					c.WriteClipboard(r.user)
				}
				if iconButton(c, iconButtonSpec{
					icon: iconPencil, label: "编辑消息 " + fmt.Sprint(i+1),
					tip: "编辑消息：行内编辑", size: rowCompact,
				}) {
					a.beginInlineEdit(i, r.user)
				}
			}
			turn, forkReason := a.forkTurnForRow(i)
			forkReason = a.forkReason(turn, forkReason)
			forkLabel := "从此处分叉 " + fmt.Sprint(i+1)
			forkTip := "从此处分叉：以这一轮结束处为切点"
			if forkReason != "" {
				forkLabel += "（" + forkReason + "）"
				// TL-39 / TL-40 从此处分叉 / 从此处开始侧边对话
				forkTip = forkReason
			}
			if iconButton(c, iconButtonSpec{
				icon:  iconFork,
				label: forkLabel,
				tip:   forkTip, size: rowCompact, disabled: forkReason != "",
			}) {
				a.forkAtTurn(turn, "从此处分叉")
			}
			if iconButton(c, iconButtonSpec{
				icon: iconBranch, label: "从此处开始侧边对话 " + fmt.Sprint(i+1),
				tip: "从此处开始侧边对话", size: rowCompact,
			}) {
				a.openSideChatFromMessage()
			}
		})
		if forkReason := a.forkReasonForRow(i); forkReason != "" {
			gui.Row(c).Grow(1).Children(func() {
				gui.Spacer(c)
				note := "分叉不可用：" + forkReason
				typeMicro.style(t, gui.Text(c, note)).
					TextColor(t.TextMuted).Label(note)
			})
		}
	})
}

// forkReasonForRow 是一行上「从此处分叉」的不可用原因；空表示可点。
func (a *App) forkReasonForRow(i int) string {
	turn, reason := a.forkTurnForRow(i)
	return a.forkReason(turn, reason)
}

// openSideChatFromMessage 打开右栏的侧边对话，并记下它是「从某条消息之前开始」
// （视图状态；侧聊还没接事实，空态文案据此区分用哪一条）。
func (a *App) openSideChatFromMessage() {
	a.rightOpen = true
	a.rpOpen(tabSideChat)
	a.panels.sideChatFromMessage = true
}

// userMessageMenu 是用户消息的右键菜单（TL-41–TL-43）：侧边对话、复制选区、
// 复制整个消息、编辑；删除与撤回没有写入口，置灰并把原因写进菜单项。
func (a *App) userMessageMenu(m *gui.Menu, svc gui.Services, i int, r round, selection string) {
	if m.Item("从此处开始侧边对话").Chosen() {
		a.openSideChatFromMessage()
	}
	// 从此处分叉：切点取这一行的轮次；拿不到时置灰并把原因写进菜单项。
	turn, reason := a.forkTurnForRow(i)
	reason = a.forkReason(turn, reason)
	forkLabel := "从此处分叉"
	if reason != "" {
		forkLabel += "（" + reason + "）"
	}
	if m.Item(forkLabel).Disabled(reason != "").Chosen() {
		a.forkAtTurn(turn, "从此处分叉")
	}
	copyLabel := "复制"
	if selection == "" {
		copyLabel = "复制（" + messageCopyNeedsSelection + "）"
	}
	if m.Item(copyLabel).Disabled(selection == "").Chosen() {
		svc.WriteClipboard(selection)
	}
	if m.Item("复制整个消息").Chosen() {
		svc.WriteClipboard(r.user)
	}
	if strings.TrimSpace(r.user) != "" && m.Item("编辑消息").Chosen() {
		a.beginInlineEdit(i, r.user)
	}
	m.Separator()
	m.Item("删除消息（" + messageDeleteUnavailable + "）").Disabled(true)
	m.Item("撤回（" + messageWithdrawUnavailable + "）").Disabled(true)
}

// assistantMessageMenu 是助手消息的右键菜单（TL-65）：复制选区、复制整个消息；
// 删除与撤回没有写入口，与用户消息一样置灰并写原因。
func (a *App) assistantMessageMenu(m *gui.Menu, svc gui.Services, text, selection string) {
	copyLabel := "复制"
	if selection == "" {
		copyLabel = "复制（" + messageCopyNeedsSelection + "）"
	}
	if m.Item(copyLabel).Disabled(selection == "").Chosen() {
		svc.WriteClipboard(selection)
	}
	if m.Item("复制整个消息").Chosen() {
		svc.WriteClipboard(text)
	}
	m.Separator()
	m.Item("删除消息（" + messageDeleteUnavailable + "）").Disabled(true)
	m.Item("撤回（" + messageWithdrawUnavailable + "）").Disabled(true)
}

// userAttachmentArea 是气泡里的附件与引用区（TL-21–TL-26）：定时来源、图片廊、
// 附件 chip、引用列表与标注，都排在正文之上（R-TL-213）。
//
// 技术口径：RowSnapshot 目前没有附件 / 引用 / 标注 / 定时来源字段（DEC-07）。
// 真实消息没有这些内容，这一区就整块不画——没有内容时不摆占位、不写内部说明；
// 投影补上字段后由 userRound 填进来即可。
func (a *App) userAttachmentArea(c *gui.Context, r round) {
	t := c.Theme()
	if len(r.images) == 0 && len(r.attachments) == 0 && len(r.quotes) == 0 &&
		len(r.marks) == 0 && r.wakeup == "" {
		return
	}
	gui.Column(c).Shrink(0).Gap(space1).Label("消息附件与引用").Children(func() {
		if r.wakeup != "" {
			// TL-25 定时唤醒来源
			typeMicro.style(t, gui.Text(c, r.wakeup)).TextColor(t.TextMuted).Label("定时唤醒来源")
		}
		if len(r.images) > 0 {
			gui.Row(c).Wrap().Gap(space1).Children(func() {
				for _, name := range r.images {
					imageThumb(c, name)
				}
			})
		}
		if len(r.attachments) > 0 {
			gui.Row(c).Wrap().Gap(space1).Children(func() {
				for _, att := range r.attachments {
					attachmentChip(c, att)
				}
			})
		}
		if len(r.quotes) > 0 {
			gui.Column(c).Gap(space1).Children(func() {
				for _, quote := range r.quotes {
					quoteRow(c, quote)
				}
			})
		}
		for _, mark := range r.marks {
			// TL-24 标注卡
			typeMicro.style(t, gui.Text(c, fmt.Sprintf("标注 %d", mark.num))).
				TextColor(t.TextMuted).Label("标注 " + fmt.Sprint(mark.num))
			typeSecondary.style(t, gui.Text(c, mark.text)).TextColor(t.Text).MaxLines(3)
		}
	})
}

// imageThumb 是一张图片附件的缩略位（TL-21）：固定方角、浅底、图片图标；
// 图片内容本身还没有进快照，这里只画占位，不假装加载了图片。
func imageThumb(c *gui.Context, name string) {
	// TL-21 图片附件廊
	t := c.Theme()
	label := "图片附件"
	if name != "" {
		label += " " + name
	}
	box := gui.Box(c).Width(space12).Height(space12).Shrink(0).Radius(radiusSmall).
		Background(t.SurfacePressed).Label(label)
	box.DrawOver(func(p *gui.Painter, rect gui.Rect) {
		size := float32(iconInline * 2)
		paintIcon(p, iconImage, gui.Rect{
			X: rect.X + (rect.W-size)/2,
			Y: rect.Y + (rect.H-size)/2,
			W: size, H: size,
		}, t.TextMuted)
	})
}

// attachmentChip 是一个附件 chip（TL-22）：图标 + 文件名；已不可用的附件降
// 对比并写「不可用」（R-TL-212）。
func attachmentChip(c *gui.Context, att messageAttachment) {
	// TL-22 附件 chip 架
	t := c.Theme()
	color := t.TextMuted
	label := "附件 " + att.name
	if att.missing {
		label += "（不可用）"
	}
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(space1).
		Padding(space1, space2).Radius(radiusSmall).
		Background(t.Surface).Label(label).Children(func() {
		iconGlyph(c, iconPaperclip, color, "attachment")
		typeMicro.style(t, gui.Text(c, att.name)).TextColor(color).SingleLine()
	})
}

// quoteRow 是一条已发送的引用（TL-23）：标题行 + 被引用的片段。
func quoteRow(c *gui.Context, quote messageQuote) {
	// TL-23 引用列表
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(space1).Label("引用 " + quote.title).Children(func() {
		typeMicro.style(t, gui.Text(c, quote.title)).TextColor(t.TextMuted).SingleLine()
		typeSecondary.style(t, gui.Text(c, quote.text)).TextColor(t.Text).MaxLines(3)
	})
}

// selectionBar 是选中正文后出现的浮条（TL-44）：复制选区、把选区作为引用填进
// 输入区。它按上一帧的选区成立；点完按钮后选区若还在，浮条也就还在。
func (a *App) selectionBar(c *gui.Context, selection string) {
	gui.Row(c).Grow(1).Gap(space1).AlignItems(gui.Center).Label("选区浮条").Children(func() {
		gui.Spacer(c)
		if actionButton(c, "复制选区", "复制选中的文本", false, false).Clicked() {
			c.WriteClipboard(selection)
		}
		if actionButton(c, "引用到输入区", "把选中的文本作为引用填进输入区", false, false).Clicked() {
			a.draft = quoteIntoDraft(a.draft, selection)
		}
	})
}

// selectedTextOf 把可选区里的选区拼回一段文本；没有选区时返回空串。
func selectedTextOf(e gui.Element) string {
	parts := e.SelectedTexts()
	if len(parts) == 0 {
		return ""
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		runes := []rune(part.Text)
		start, end := part.Start, part.End
		if start < 0 {
			start = 0
		}
		if end > len(runes) {
			end = len(runes)
		}
		if start >= end {
			continue
		}
		out = append(out, string(runes[start:end]))
	}
	return strings.Join(out, "\n")
}

// quoteIntoDraft 把一段选区作为 Markdown 引用接到草稿末尾；草稿为空时不留前导
// 空行，已有内容时隔一空行。
func quoteIntoDraft(draft, selection string) string {
	selection = strings.TrimSpace(selection)
	if selection == "" {
		return draft
	}
	var b strings.Builder
	b.WriteString(draft)
	if strings.TrimSpace(draft) != "" {
		b.WriteString("\n\n")
	}
	for _, line := range strings.Split(selection, "\n") {
		b.WriteString("> ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// assistant 是助手区：思考、正文、工具分组、轮末块与动作条。整块可选（右键
// 菜单按选区复制，TL-65）；Markdown 的选区交给这一层（ShareSelection）。
func (a *App) assistant(c *gui.Context, i int, r round) {
	// TL-12 / TL-50 助手区容器
	col := gui.Column(c).Shrink(0).Gap(timelineBlockGap).
		Role(gui.RoleArticle).Label("助手消息").Selectable()
	hovered := col.Hovered()
	svc := c.Services()
	col = col.ContextMenu(func(m *gui.Menu) {
		a.assistantMessageMenu(m, svc, assistantMessageText(r), selectedTextOf(col))
	})
	col.Children(func() {
		if r.status == roundRunning {
			// 运行中：轮首先写这一轮已经工作了多久、在做什么（TL-46）。
			a.runningSummary(c, i, r)
		}
		if r.thinking != "" {
			a.thinkingBlock(c, i, r)
		}
		if r.reply != "" {
			reply := a.assistantMarkdown(c, searchKeyAssistant(i), r.reply)
			a.searchReveal(c, reply, i, searchScopeAssistant)
		}
		// 完成态的过程摘要；运行中由上面的时长摘要承担，不再重复。
		if r.status != roundRunning && (len(r.ops) > 0 || r.status == roundFailed || r.status == roundStopped) {
			a.roundHeader(c, i, r)
		}
		if r.status == roundFailed {
			a.failureBlock(c)
		}
		if r.status == roundStopped {
			// TL-60 用户中断：停止不是失败，单独给一行通知。
			a.userInterrupted(c)
		}
		if r.retrySeconds > 0 {
			a.retryNotice(c, r)
		}
		if len(r.artifacts) > 0 {
			a.artifactsBlock(c, i, r)
		}
		if len(r.fileChanges) > 0 {
			a.fileChangesBlock(c, i, r)
		}
		if r.note != "" {
			a.note(c, r.note)
		}
		if hovered {
			a.assistantActions(c, i, r)
		}
	})
}

// runningSummary 是运行中一轮的轮首摘要行（TL-46）：两侧细线 +「已经工作了
// {时长} · {动作}」+ 折叠箭头，展开后是这一轮的工具行。
func (a *App) runningSummary(c *gui.Context, i int, r round) {
	t := c.Theme()
	open := a.openOf(i)
	title := runningSummaryText(r)
	var btn gui.Element
	var progress, arrow float32
	gui.Row(c).Grow(1).Gap(space2).AlignItems(gui.Center).Children(func() {
		gui.Box(c).Grow(1).Height(strokeHairline).Background(t.Border)
		btn = gui.ButtonBase(c).Shrink(0).Height(timelineRowHeight).Gap(space1).
			AlignItems(gui.Center).Padding(0, space1).Radius(radius).
			Cursor(gui.CursorPointer).
			// 屏幕上不摆状态位；状态留在可读名里（与完成态组头同一口径）。
			Label("状态：" + statusWord(r.status) + "，" + title).
			Role(gui.RoleDisclosure).Expanded(*open)
		switch {
		case btn.Pressed():
			btn = btn.Background(t.SurfacePressed)
		case btn.Hovered():
			btn = btn.Background(t.SurfaceHover)
		}
		progress = btn.AnimateTo("running", b2f(*open), timelineCollapse)
		arrow = btn.AnimateTo("running-arrow", b2f(*open), timelineArrow)
		btn.Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, title)).TextColor(t.TextMuted).SingleLine()
			chevron(c, t, arrow)
		})
		gui.Box(c).Grow(1).Height(strokeHairline).Background(t.Border)
	})
	if btn.Clicked() {
		*open = !*open
	}
	a.revealPanel(c, *open, progress, func() {
		a.roundOps(c, i, r)
	})
}

// runningSummaryText 是运行中摘要的文案：「已经工作了 {时长}」，
// 有动作时再接「 · {动作}」（参考「已经工作了 48.5s · 执行命令」）。
func runningSummaryText(r round) string {
	duration := r.duration
	if duration == "" {
		duration = "0s"
	}
	text := "已经工作了 " + duration
	if action := runningAction(r); action != "" {
		text += " · " + action
	}
	return text
}

// runningAction 是运行中摘要里的动作：取这一轮最后一个动作的说法；
// 还没有动作时是「等待」（参考的同位文案）。
func runningAction(r round) string {
	for i := len(r.ops) - 1; i >= 0; i-- {
		switch r.ops[i].action {
		case "执行":
			return "执行命令"
		case "读取":
			return "读取文件"
		case "改动", "编辑", "写入":
			return "改动文件"
		case "搜索":
			return "搜索"
		case "抓取":
			return "抓取网页"
		}
	}
	return "等待"
}

// thinkingBlock 是思考块：灯泡 + 「思考了 Xs」+ 箭头；展开为引用块。
// 用 RoleButton 而不是 Disclosure，保证轮次组头仍是第一个可展开分组（读屏顺序）。
func (a *App) thinkingBlock(c *gui.Context, i int, r round) {
	// TL-52 思考块（正在思考 / 思考了 {duration}）
	t := c.Theme()
	open := a.blockOf(fmt.Sprintf("think:%d", i), false)
	title := "思考了 " + r.thinkingDuration
	if r.thinkingDuration == "" {
		title = "思考"
	}
	if r.status == roundRunning {
		title = "正在思考 · " + r.thinkingDuration
		if r.thinkingDuration == "" {
			title = "正在思考"
		}
	}
	btn := gui.ButtonBase(c).Grow(1).Height(timelineRowHeight).Gap(space1).
		Justify(gui.Start).AlignItems(gui.Center).Cursor(gui.CursorPointer).
		Label("思考过程 " + title).Role(gui.RoleButton).Expanded(*open)
	switch {
	case btn.Pressed():
		btn = btn.Background(t.SurfacePressed)
	case btn.Hovered():
		btn = btn.Background(t.SurfaceHover)
	}
	progress := btn.AnimateTo("thinking", b2f(*open), timelineCollapse)
	arrow := btn.AnimateTo("thinking-arrow", b2f(*open), timelineArrow)
	btn.Children(func() {
		iconGlyph(c, iconBulb, t.TextMuted, "thinking")
		typeSecondary.style(t, gui.Text(c, title)).TextColor(t.TextMuted).SingleLine()
		chevronLabeled(c, t, arrow, "思考展开指示")
	})
	if btn.Clicked() {
		*open = !*open
	}
	a.searchReveal(c, btn, i, searchScopeThinking)
	a.revealPanel(c, *open, progress, func() {
		gui.Row(c).Grow(1).Gap(space2).Children(func() {
			rail := gui.Box(c).Width(strokeHairline).Shrink(0).AlignSelf(gui.Stretch).Label("思考轨道")
			rail.DrawOver(func(p *gui.Painter, rect gui.Rect) {
				// 1px 的轨道按 DPR 落到设备像素格：2× 上正好两列实心像素，不发灰。
				p.Fill(hairlineRect(rect, p.Scale(), false), t.Border, 0)
			})
			a.searchText(c, searchKeyThinking(i), r.thinking, typeSecondary, t.TextMuted).
				Grow(1).Selectable()
		})
	})
}

// toolGroup 是连续工具调用的折叠组：分组头写动作汇总，展开后是缩进的工具行。
//
// 组头同时承担「工作过程」一行：完成的轮写「工作过程 · 动作汇总」，被停止的轮
// 写「已停止 · 时长」，失败的轮按有没有终端现场分成「{时长} 后失败」与「出错前
// 的工作过程」两种（TL-45 / TL-47 / TL-48 / TL-49）。
func (a *App) roundHeader(c *gui.Context, i int, r round) {
	// TL-51 工具调用分组的组头
	t := c.Theme()
	open := a.openOf(i)
	summary := groupSummary(r)
	title := workProcessTitle(r, summary)
	// 窄窗里左栏只有边条：按窄窗的可用宽度收标题，不把整行挤出内容列。
	title = fitPhrase(c, t, title, a.sidebarCollapsed || narrowWindow(c), a.rightOpen)
	phrase := roundHeaderPhrase(r)
	btn := gui.ButtonBase(c).Grow(1).Height(timelineRowHeight).Gap(space1).
		Justify(gui.Start).AlignItems(gui.Center).Padding(0, space1).Radius(radius).
		Cursor(gui.CursorPointer).
		Label("轮次组头 " + summary + "，状态：" + statusWord(r.status) + "，" + phrase).
		Role(gui.RoleDisclosure).Expanded(*open)
	switch {
	case btn.Pressed():
		btn = btn.Background(t.SurfacePressed)
	case btn.Hovered():
		btn = btn.Background(t.SurfaceHover)
	}
	progress := btn.AnimateTo("group", b2f(*open), timelineCollapse)
	arrow := btn.AnimateTo("group-arrow", b2f(*open), timelineArrow)
	btn.Children(func() {
		// 行首只有分组图标、文案与折叠箭头（参考不摆状态位）。
		iconGlyph(c, iconWorkflow, t.TextMuted, "workflow")
		// TL-72 分组头的动作汇总文案
		typeSecondaryStrong.style(t, gui.Text(c, title)).TextColor(headerColor(t, r.status)).SingleLine()
		// TL-73 分组折叠箭头
		chevron(c, t, arrow)
	})
	if btn.Clicked() {
		*open = !*open
	}
	a.revealPanel(c, *open, progress, func() {
		a.roundOps(c, i, r)
	})
}

// roundOps 是一轮的工具行（分组头与运行中摘要共用）：左侧轨道 + 缩进的操作行。
func (a *App) roundOps(c *gui.Context, i int, r round) {
	// TL-74 分组内容轨道（缩进的操作行）
	t := c.Theme()
	gui.Row(c).Grow(1).Gap(space2).Children(func() {
		rail := gui.Box(c).Width(timelineGroupIndent).Shrink(0).AlignSelf(gui.Stretch).Label("分组轨道")
		rail.DrawOver(func(p *gui.Painter, rect gui.Rect) {
			x := rect.X + timelineGroupIndent/2
			p.Fill(hairlineRect(gui.Rect{X: x, Y: rect.Y, W: strokeHairline, H: rect.H}, p.Scale(), false),
				t.Border, 0)
		})
		gui.Column(c).Grow(1).Gap(space3).Children(func() {
			for n, o := range r.ops {
				// TL-56 子代理通知行：工具名认得出是子代理时画 chip，不画普通工具行。
				if subagentTool(o.action) {
					a.subagentChip(c, o.target, o.output)
					continue
				}
				a.opRow(c, i, n, o)
				// 工具行收起时在行下给一条失败原因；展开后原因在详情的
				// 「失败原因」段里，这里不再重复。
				if o.failed && o.cause != "" &&
					!*a.blockOf(fmt.Sprintf("tool:%d:%d", i, n), false) {
					typeSecondary.style(t, gui.Text(c, o.cause)).TextColor(t.TextMuted).
						PaddingStart(timelineGroupIndent)
				}
			}
		})
	})
}

// toolRow 是一行工具调用：图标 + 动词 + 参数 + 差异计数/失败标记 + 箭头。
func (a *App) opRow(c *gui.Context, roundIndex, opIndex int, o op) {
	// TL-54 工具调用标题行
	t := c.Theme()
	key := fmt.Sprintf("tool:%d:%d", roundIndex, opIndex)
	open := a.blockOf(key, false)
	hasDetails := o.input != "" || o.output != "" || o.cause != ""
	display := primaryParam(o)
	btn := gui.ButtonBase(c).Grow(1).Height(timelineRowHeight).Gap(space1).
		Justify(gui.Start).AlignItems(gui.Center).Padding(0, space1).Radius(radius).
		Cursor(gui.CursorPointer).Label(o.action + " " + display)
	if hasDetails {
		btn = btn.Role(gui.RoleButton).Expanded(*open)
	}
	switch {
	case btn.Pressed():
		btn = btn.Background(t.SurfacePressed)
	case btn.Hovered():
		btn = btn.Background(t.SurfaceHover)
	}
	progress := btn.AnimateTo("tool", b2f(*open), timelineCollapse)
	arrow := btn.AnimateTo("tool-arrow", b2f(*open), timelineArrow)
	btn.Children(func() {
		// TL-76 网页工具行的站点图标批次（有主机名时）
		if o.action == "抓取" {
			a.siteIconBatch(c, webHosts(o.target))
		}
		iconGlyph(c, timelineActionIcon(o.action), t.TextMuted, "tool")
		typeSecondary.style(t, gui.Text(c, o.action)).TextColor(t.TextMuted).SingleLine()
		if fileAction(o.action) {
			iconGlyph(c, iconDocument, t.TextMuted, "file")
		}
		a.searchText(c, searchKeyToolParam(roundIndex, opIndex), display, typeSecondary, t.Text).
			Shrink(1).SingleLine()
		// TL-75 +{新增} / -{删除}
		if o.added > 0 {
			typeMicro.style(t, gui.Text(c, fmt.Sprintf("+%d", o.added))).TextColor(t.Success)
		}
		if o.removed > 0 {
			typeMicro.style(t, gui.Text(c, fmt.Sprintf("-%d", o.removed))).TextColor(t.Danger)
		}
		// TL-77 ✗ 失败状态标记
		if o.failed {
			iconGlyphSize(c, iconXMark, 12, t.Warning, "工具失败")
		}
		// TL-78 工具行展开箭头
		if hasDetails {
			chevronLabeled(c, t, arrow, "工具展开指示")
		}
	})
	if btn.Clicked() && hasDetails {
		*open = !*open
	}
	a.searchReveal(c, btn, roundIndex, searchScopeTool)
	if !hasDetails {
		return
	}
	a.revealPanel(c, *open, progress, func() {
		a.toolDetail(c, roundIndex, opIndex, o)
	})
}

// toolDetail 是工具行的展开区：输入 / 输出 / 失败原因三段，段首写段名，
// 内容用等宽浅底块；失败原因取危险色。技术口径：输入与失败原因分开进快照之前，
// 失败的工具只有输出可用——没有原因原文时「失败原因」段整段不画，不把输出冒充
// 原因，也不在屏上留一句内部说明。
func (a *App) toolDetail(c *gui.Context, row, n int, o op) {
	// TL-55 工具调用详情（展开区）
	t := c.Theme()
	gui.Column(c).Gap(space1).PaddingStart(timelineGroupIndent).Children(func() {
		if o.input != "" {
			a.toolDetailSection(c, "输入", t.TextMuted, func() {
				a.searchCodeBlock(c, searchKeyToolInput(row, n), o.input)
			})
		}
		// TL-59 终端类工具失败：输出与原因收在「终端错误详情」里，标题行带复制。
		if o.terminal && (o.failed || o.cause != "") {
			a.terminalErrorDetail(c, o)
			return
		}
		if o.output != "" {
			a.toolDetailSection(c, "输出", t.TextMuted, func() {
				a.searchCodeBlock(c, searchKeyToolOutput(row, n), o.output)
			})
		}
		if o.cause != "" {
			a.toolDetailSection(c, "失败原因", t.Danger, func() {
				a.codeBlock(c, o.cause)
			})
		}
	})
}

// toolDetailSection 是一段工具详情：段名（可定位）+ 一段内容。
func (a *App) toolDetailSection(c *gui.Context, title string, color gui.Color, body func()) {
	t := c.Theme()
	typeMicro.style(t, gui.Text(c, title)).TextColor(color).Label("工具详情：" + title)
	body()
}

// codeBlock 是等宽浅底块：工具输入/输出与日志共用。
func (a *App) codeBlock(c *gui.Context, text string) {
	t := c.Theme()
	gui.Box(c).Shrink(0).Padding(space2).Radius(radius).Background(t.SurfacePressed).
		Children(func() {
			typeCode.style(t, gui.Text(c, text)).TextColor(t.Text).Selectable()
		})
}

// failureBlock 是块失败的标题与说明（TL-57）。
func (a *App) failureBlock(c *gui.Context) {
	// TL-57 运行失败的标题与说明
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Start).Children(func() {
		iconGlyph(c, iconWarn, t.Danger, "failure")
		gui.Column(c).Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, "运行失败")).TextColor(t.Danger)
			typeSecondary.style(t, gui.Text(c, "本次运行失败。")).TextColor(t.Danger)
		})
	})
}

// retryNotice 是自动重试提示：倒计时 + 立即重试。
//
// 秒数取行事实里的剩余时间（RowSnapshot.RetryIn）；「立即重试」没有通道（协议
// 没有立即重试的方法，RetrySubmit 也不是这件事），所以入口照留、置灰并把原因
// 写在 tooltip 上。剩余时间拿不到时整行不画（见 liveRow 的 RowRetry 分支）。
func (a *App) retryNotice(c *gui.Context, r round) {
	// TL-58 {seconds}s 后重试
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
		typeSecondary.style(t, gui.Text(c, fmt.Sprintf("%ds 后重试", r.retrySeconds))).
			TextColor(t.TextMuted)
		gui.Button(c, "立即重试").Label("立即重试").
			Tooltip(noteRetryNow).Disabled(true).Opacity(noteDisabledAlpha)
	})
}

// artifactsBlock 是本轮产物：计数 + 展开列表（每卡一个打开入口）。
func (a *App) artifactsBlock(c *gui.Context, i int, r round) {
	// TL-14 本轮产物块
	// A-01 产物块容器（块根：折叠头 + 可展开列表）
	t := c.Theme()
	open := a.blockOf(fmt.Sprintf("artifacts:%d", i), false)
	// A-02 计数文本（{count} 个产物）
	title := fmt.Sprintf("%d 个产物", len(r.artifacts))
	btn := gui.ButtonBase(c).Grow(1).Height(timelineRowHeight).Gap(space1).
		Justify(gui.Start).AlignItems(gui.Center).Padding(0, space1).Radius(radius).
		Cursor(gui.CursorPointer).Label(title).Role(gui.RoleDisclosure).Expanded(*open)
	if btn.Hovered() {
		btn = btn.Background(t.SurfaceHover)
	}
	progress := btn.AnimateTo("artifacts", b2f(*open), timelineCollapse)
	arrow := btn.AnimateTo("artifacts-arrow", b2f(*open), timelineArrow)
	btn.Children(func() {
		iconGlyph(c, iconDocument, t.TextMuted, "artifacts")
		typeSecondary.style(t, gui.Text(c, title)).TextColor(t.Text)
		chevronLabeled(c, t, arrow, "产物展开指示")
	})
	if btn.Clicked() {
		*open = !*open
	}
	a.revealPanel(c, *open, progress, func() {
		a.artifactList(c, fmt.Sprintf("artifacts:%d", i), r.artifacts)
	})
}

// artifactShownRows 是收起时最多列出的产物行数（A-08 / A-18）。
const artifactShownRows = 3

// artifactList 是产物列表（A-08）：收起时最多 3 行，多于 3 行时底部给
// 「全部 {n} 个产物」；展开后列出全部并换成「收起」。
func (a *App) artifactList(c *gui.Context, key string, arts []artifact) {
	expanded := a.artifactsMore[key]
	rows := arts
	if !expanded && len(rows) > artifactShownRows {
		rows = rows[:artifactShownRows]
	}
	// A-07 产物列表容器（每行一条产物）
	gui.Column(c).Shrink(0).Gap(0).PaddingStart(timelineGroupIndent).Children(func() {
		for _, art := range rows {
			a.artifactRow(c, key, art, len(arts))
		}
		if len(arts) > artifactShownRows {
			a.artifactsMoreButton(c, key, len(arts), expanded)
		}
	})
}

// artifactsMoreButton 是列表底部的展开 / 收起入口（A-18 / A-19）。
func (a *App) artifactsMoreButton(c *gui.Context, key string, count int, expanded bool) {
	t := c.Theme()
	icon, text := iconChevronR, fmt.Sprintf("全部 %d 个产物", count)
	if expanded {
		icon, text = iconChevronDown, "收起"
	}
	btn := gui.ButtonBase(c).Shrink(0).Height(rowRegular).Gap(space1).
		Padding(0, space1).Justify(gui.Start).AlignItems(gui.Center).Radius(radius).
		Cursor(gui.CursorPointer).Label(text)
	btn.Children(func() {
		iconGlyph(c, icon, t.TextMuted, "artifacts-more")
		typeMicro.style(t, gui.Text(c, text)).TextColor(t.TextMuted).SingleLine()
	})
	if btn.Clicked() {
		a.artifactsMore[key] = !expanded
	}
}

// artifactRow 是一条产物行（A-08 / A-14）：类型图标 + 文件名（单行截断、
// 悬停给完整名）+「打开」+ 行尾下拉。协议没有产物内容，打开与下拉项都置灰
// 并写原因；下拉本身仍可展开（A-14 的展开、翻转、单开由视图状态承担）。
func (a *App) artifactRow(c *gui.Context, key string, art artifact, count int) {
	t := c.Theme()
	row := gui.Row(c).Shrink(0).Height(rowRegular).Gap(space1).Padding(0, space1).
		AlignItems(gui.Center).Label("产物 " + art.name)
	if row.Hovered() {
		row = row.Background(t.SurfaceHover)
	}
	menuKey := key + "/" + art.name
	row.Children(func() {
		// A-09 文件类型图标
		iconGlyph(c, iconDocument, t.TextMuted, "artifact")
		gui.Box(c).Grow(1).Children(func() {
			// A-10 文件名（单行截断，悬停给完整名）
			typeSecondary.style(t, gui.Text(c, art.name)).TextColor(t.Text).
				Tooltip(art.name).SingleLine()
		})
		// A-13 行内「打开」（协议没有产物内容，置灰并写原因）
		gui.Button(c, "打开").Label("打开产物 " + art.name + "（" + noteArtifact + "）").
			Tooltip(noteArtifact).Disabled(true)
		a.artifactMenuTrigger(c, menuKey, art)
	})
	a.artifactMenu(c, row, menuKey, art)
}

// artifactMenuTrigger 是行尾的下拉触发器（A-14）：ChevronDown 图标按钮，展开后
// 翻转朝上；同时只开一个。能力都不成立时不出现（A-14 的条件）。
func (a *App) artifactMenuTrigger(c *gui.Context, menuKey string, art artifact) {
	if !a.artifactMenuAvailable() {
		return
	}
	open := a.artifactMenuOpen(menuKey)
	icon := iconChevronDown
	label := "更多操作"
	if *open {
		icon = iconChevronUp
		label = "更多操作（已展开）"
	}
	if iconButton(c, iconButtonSpec{
		icon: icon, label: label + "：产物 " + art.name, size: rowCompact,
	}) {
		a.toggleArtifactMenu(menuKey, !*open)
	}
}

// artifactMenuOpen 取某条产物的下拉开合态。
func (a *App) artifactMenuOpen(menuKey string) *bool {
	if a.artifactMenus == nil {
		a.artifactMenus = map[string]*bool{}
	}
	if p, ok := a.artifactMenus[menuKey]; ok {
		return p
	}
	open := false
	a.artifactMenus[menuKey] = &open
	return &open
}

// toggleArtifactMenu 打开或收起一条产物的下拉，并把同组的其它下拉收起（A-121）。
func (a *App) toggleArtifactMenu(menuKey string, open bool) {
	for name := range a.artifactMenus {
		if name != menuKey {
			*a.artifactMenus[name] = false
		}
	}
	*a.artifactMenuOpen(menuKey) = open
}

// artifactMenuAvailable 报告产物行要不要出现行尾下拉：宿主允许本地操作或允许
// 下载时才出现（A-14）。本版的宿主没有这两项能力，连线时不画触发器；样例模式
// 按参考产品的形态画出来（菜单项本身仍置灰，见下）。
func (a *App) artifactMenuAvailable() bool { return !a.wired() }

// artifactMenu 是产物行的下拉菜单（A-15–A-17）：用默认应用打开 / 在文件管理器
// 中显示 / 复制路径 / 复制相对路径。产物没有内容与路径事实，四项都置灰并写原因。
func (a *App) artifactMenu(c *gui.Context, target gui.Element, menuKey string, art artifact) {
	t := c.Theme()
	m := gui.ContextMenuBase(c, target, a.artifactMenuOpen(menuKey))
	m.Popup(func(panel gui.Element) {
		rpMenuPanel(panel, t, panelMenuNoteWidth+panelRowIcon+2*space2)
		note := noteArtifact
		menuItem(c, m, iconExternal, "用默认应用打开", "用默认应用打开", note, true, false)
		menuItem(c, m, iconFolderOpen, "在文件管理器中显示", "在文件管理器中显示", note, true, false)
		rpMenuSeparator(c)
		menuItem(c, m, iconCopy, "复制路径", "复制路径", note, true, false)
		menuItem(c, m, iconCopy, "复制相对路径", "复制相对路径", note, true, false)
	})
}

// fileChangesBlock 是本轮文件改动（D-80–D-97）：折叠头（计数 + 箭头）与展开
// 列表（状态字母、路径、增删数、右键菜单）。
//
// 事实口径：行集合来自该轮的改动声明（路径、状态字母、增删行数能读到的部分）；
// 「用默认应用打开 / 在文件管理器中显示」要宿主打开外部文件的能力，本版没有
// 通道，菜单项保留但置灰并写原因（DEC-09）；复制路径与复制相对路径是本地动作，
// 能算出路径时照常可用。
func (a *App) fileChangesBlock(c *gui.Context, i int, r round) {
	// TL-15 本轮改动块
	t := c.Theme()
	open := a.blockOf(fmt.Sprintf("files:%d", i), false)
	title := fmt.Sprintf("%d 个文件改动", len(r.fileChanges))
	// D-80 块根：通栏、无独立卡片边框，随回合内容滚动。
	gui.Column(c).Shrink(0).Gap(space1).Label("本轮改动").Children(func() {
		// D-81 折叠头（整行可点按钮：箭头 + 计数文本）
		state := "，已收起"
		if *open {
			state = "，已展开"
		}
		btn := gui.ButtonBase(c).Height(timelineRowHeight).Gap(space1).
			Justify(gui.Start).AlignItems(gui.Center).Padding(0, space1).Radius(radius).
			Cursor(gui.CursorPointer).Label(title).Description(title + state).
			Role(gui.RoleDisclosure).Expanded(*open)
		if btn.Hovered() {
			btn = btn.Background(t.SurfaceHover)
		}
		progress := btn.AnimateTo("files", b2f(*open), timelineCollapse)
		arrow := btn.AnimateTo("files-arrow", b2f(*open), timelineArrow)
		btn.Children(func() {
			// D-82 已展开箭头 / D-83 已收起箭头：同一个圆角尖角，按开合转 90°。
			arrowLabeled(c, t, arrow, "改动展开指示")
			// D-84 计数文本：{count} 个文件改动（未撤销口径）。
			typeSecondary.style(t, gui.Text(c, title)).TextColor(t.Text)
		})
		if btn.Clicked() {
			*open = !*open
		}
		a.revealPanel(c, *open, progress, func() {
			// D-88 展开态列表容器：圆角描边，行在容器内。
			list := gui.Column(c).Shrink(0).Gap(0).Padding(space1, space1).
				Radius(radius).Border(strokeHairline, t.Border)
			list.Children(func() {
				for n, change := range r.fileChanges {
					a.fileChangeRow(c, i, n, change)
				}
			})
		})
	})
}

// arrowLabeled 画折叠箭头并把它标成「不单独播报」：箭头方向只作视觉提示，读屏
// 由折叠头按钮的状态（已展开 / 已收起）表达（D-82 / D-83）。
func arrowLabeled(c *gui.Context, t *gui.Theme, progress float32, label string) gui.Element {
	box := gui.Box(c).Width(iconInline).Height(iconInline).Shrink(0).
		Label(label).AccessHidden(true)
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.StrokePolyline(chevronPoints(r, chevronAngle(progress)), strokeChevron, t.TextMuted)
	})
	return box
}

// timelineFilePath 是改动行的绝对路径（D-96）：路径本身是绝对路径时原样用，
// 相对路径按会话工作目录拼；没有工作目录时 ok 为假（不拼半条路径）。
func (a *App) timelineFilePath(p string) (string, bool) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", false
	}
	if filepath.IsAbs(p) {
		return p, true
	}
	cwd := a.rpCwd()
	if cwd == "" {
		return "", false
	}
	return filepath.Join(cwd, filepath.FromSlash(p)), true
}

// workspaceRelativePath 是「复制相对路径」的唯一判定（D-97）：行属于工作区时给出
// 相对路径，文件在工作区外、或者没有工作目录算不出来时 ok 为假。时间线改动行与
// 右栏改动菜单共用这一处，条件与文案不会一处一个样。
//
// 行上的路径已经是相对路径时也要求有工作目录：相对路径是相对工作区说的，没有
// 工作目录就无从判断它属于哪个工作区（不拿裸路径冒充相对路径）。
func workspaceRelativePath(cwd, p string) (string, bool) {
	p = strings.TrimSpace(p)
	cwd = strings.TrimSpace(cwd)
	if p == "" || cwd == "" {
		return "", false
	}
	if !filepath.IsAbs(p) {
		rel := filepath.Clean(filepath.FromSlash(p))
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", false
		}
		return filepath.ToSlash(rel), true
	}
	rel, err := filepath.Rel(cwd, p)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// fileChangeInk 是状态字母的颜色（D-90）：新增绿、删除红、改动警示色、
// 重命名与未跟踪用强调色；认不出的状态取次要色。
func fileChangeInk(status string, t *gui.Theme) gui.Color {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "A":
		return t.Success
	case "D":
		return t.Danger
	case "M", "U":
		return t.Warning
	case "R", "?":
		return colorAccentInk.in(t)
	}
	return t.TextMuted
}

// fileChangeStats 是行内统计（D-92）：「+{insertions}/-{deletions}」；两段都
// 没有时返回空串（不把「不知道」画成 0）。
func fileChangeStats(change fileChange) string {
	if change.added == 0 && change.removed == 0 {
		return ""
	}
	return fmt.Sprintf("+%d/-%d", change.added, change.removed)
}

// fileChangeRow 是一个改动文件（D-89–D-97）：图标 + 文件名 + 状态字母 + 统计；
// 行右键给打开类与复制类菜单。
func (a *App) fileChangeRow(c *gui.Context, turn, index int, change fileChange) {
	t := c.Theme()
	// D-89 改动文件行（行按钮，带右键菜单）
	row := gui.ButtonBase(c).Grow(1).Height(rowRegular).Gap(space2).Padding(0, space2).
		Radius(radiusSmall).AlignItems(gui.Center).Role(gui.RoleListItem).
		Label("改动文件 " + change.path).Tooltip(change.path).
		Cursor(gui.CursorPointer)
	if row.Clicked() {
		// 行点击在右栏预览该文件（与参考一致：行本身不是纯展示）。
		a.rpOpenPreview(change.path)
	}
	if row.Hovered() {
		row = row.Background(t.SurfaceHover)
	}
	row.Children(func() {
		iconGlyph(c, iconDocument, t.TextMuted, "file")
		// D-91 文件路径（title = 完整路径）
		typeSecondary.style(t, gui.Text(c, change.path)).TextColor(t.Text).Grow(1).SingleLine()
		// D-90 状态字母（等宽，按类型着色）
		if change.status != "" {
			typeMicro.style(t, gui.Text(c, change.status)).
				TextColor(fileChangeInk(change.status, t)).SingleLine()
		}
		// D-92 行内统计（等宽；无增删行数时不画）
		if stats := fileChangeStats(change); stats != "" {
			typeMicro.style(t, gui.Text(c, stats)).TextColor(t.TextMuted).SingleLine()
		}
	})
	a.fileChangeMenu(c, row, fmt.Sprintf("fileChange:%d:%d", turn, index), change)
}

// fileChangeMenu 是改动文件行的右键菜单（D-93–D-97）：用默认应用打开 / 在文件
// 管理器中显示（宿主能力，置灰写原因）/ 分隔线 / 复制路径 / 复制相对路径。
func (a *App) fileChangeMenu(c *gui.Context, target gui.Element, key string, change fileChange) {
	m := gui.ContextMenuBase(c, target, a.blockOf("fileChangeMenu:"+key, false))
	m.Popup(func(panel gui.Element) {
		rpMenuPanel(panel, c.Theme(), panelMenuNoteWidth+panelRowIcon+2*space2)
		// D-93 / D-94：打开外部应用与文件管理器都要宿主能力，本版没有通道。
		menuItem(c, m, iconExternal, "用默认应用打开", "用默认应用打开", notePreviewIO, true, false)
		menuItem(c, m, iconFolderOpen, "在文件管理器中显示", "在文件管理器中显示", notePreviewIO, true, false)
		// D-95 打开类与复制类之间的分隔线
		rpMenuSeparator(c)
		// D-96 复制路径（绝对路径；算不出工作目录时置灰并写原因）
		abs, absOK := a.timelineFilePath(change.path)
		absNote := ""
		if !absOK {
			absNote = noteNoWorkspacePath
		}
		if menuItem(c, m, iconCopy, "复制路径", "复制路径", absNote, !absOK, false) && absOK {
			c.WriteClipboard(abs)
		}
		// D-97 复制相对路径（行属于工作区才出现；与右栏同一判定）
		if rel, ok := workspaceRelativePath(a.rpCwd(), change.path); ok {
			if menuItem(c, m, iconCopy, "复制相对路径", "复制相对路径", "", false, false) {
				c.WriteClipboard(rel)
			}
		}
	})
}

// assistantActions 是轮末动作条：复制、分叉、重试与完成时间（悬停可见）。
func (a *App) assistantActions(c *gui.Context, i int, r round) {
	// TL-13 / TL-61 / TL-62 / TL-63 / TL-64 轮末动作条
	t := c.Theme()
	gui.Row(c).Gap(space1).AlignItems(gui.Center).Label("助手动作").Children(func() {
		iconButton(c, iconButtonSpec{icon: iconCopy, label: "复制整个消息", size: rowCompact})
		turn, reason := a.forkTurnForRow(i)
		reason = a.forkReason(turn, reason)
		forkLabel, forkTip := "从此处分叉", "从此处分叉：以这一轮结束处为切点"
		if reason != "" {
			forkLabel += "（" + reason + "）"
			forkTip = reason
		}
		if iconButton(c, iconButtonSpec{
			icon: iconFork, label: forkLabel, tip: forkTip,
			size: rowCompact, disabled: reason != "",
		}) {
			a.forkAtTurn(turn, "从此处分叉")
		}
		iconButton(c, iconButtonSpec{icon: iconRetry, label: "重试", size: rowCompact})
		if r.at != "" {
			typeMicro.style(t, gui.Text(c, r.at)).TextColor(t.TextMuted).SingleLine()
		}
	})
}

// note 是本轮下方的一行说明（结果不确定）：行首状态位 + 一行正文。
func (a *App) note(c *gui.Context, note string) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Label("结果不确定行").Children(func() {
		iconGlyph(c, iconWarn, t.Warning, "exclamation")
		typeBody.style(t, gui.Text(c, note)).Selectable()
	})
}

// backToLatest 是浮在时间线右下角的「回到最新」按钮，只在滚离底部后出现。
func (a *App) backToLatest(c *gui.Context) {
	// TL-08 回到最新
	if a.feed.MaxY <= 0 || a.feed.MaxY-a.feed.Y < timelineBackThreshold {
		return
	}
	gui.Row(c).Absolute().Bottom(space3).Right(timelinePadX).Children(func() {
		if iconButton(c, iconButtonSpec{icon: iconChevronDown, label: "回到最新", size: rowCompact}) {
			a.feed.ScrollTo(0, math.MaxFloat32, gui.Smooth)
		}
	})
}

// revealPanel 是折叠容器的展开区：高度按 0.3s 曲线增长，内容随进度淡入。
func (a *App) revealPanel(c *gui.Context, open bool, progress float32, body func()) {
	if !open && progress <= 0 {
		return
	}
	clip := gui.Column(c).Shrink(0).ClipY()
	var inner gui.Element
	clip.Children(func() {
		inner = gui.Column(c).Shrink(0)
		inner = inner.Opacity(0.2 + 0.8*progress)
		inner.Children(body)
	})
	if progress < 1 {
		clip.Height(progress * inner.Bounds().H)
	}
}

// groupSummary 是分组头的动作汇总（TL-72）：
// 「思考1轮 · 改2次文件、搜1次、抓取2次、其他1次」。
func groupSummary(r round) string {
	// TL-72 分组头的动作汇总
	parts := make([]string, 0, 6)
	if r.thinking != "" {
		parts = append(parts, "思考1轮")
	}
	counts := map[string]int{}
	for _, o := range r.ops {
		counts[o.action]++
	}
	var pieces []string
	for _, item := range []struct {
		action string
		word   string
	}{
		{"读取", "读"}, {"改动", "改"}, {"执行", "执行"}, {"写入", "写"}, {"编辑", "改"},
		{"搜索", "搜"}, {"抓取", "抓取"},
	} {
		if n := counts[item.action]; n > 0 {
			pieces = append(pieces, fmt.Sprintf("%s%d次%s", item.word, n, summaryUnit(item.action)))
			delete(counts, item.action)
		}
	}
	other := 0
	for _, n := range counts {
		other += n
	}
	if other > 0 {
		pieces = append(pieces, fmt.Sprintf("其他%d次", other))
	}
	if len(pieces) > 0 {
		parts = append(parts, strings.Join(pieces, "、"))
	}
	return strings.Join(parts, " · ")
}

// summaryUnit 是动作汇总里的量词。
func summaryUnit(action string) string {
	switch action {
	case "读取", "改动", "写入", "编辑":
		return "文件"
	case "执行":
		return "命令"
	default:
		return ""
	}
}

// primaryParam 是工具行的主参数：命令取第一段，其余取末段文件名或原文。
func primaryParam(o op) string {
	target := strings.TrimSpace(o.target)
	if target == "" {
		return ""
	}
	return shortTarget(target)
}

// fileAction 表示动作对象是文件，行内要带文件图标。
func fileAction(action string) bool {
	switch action {
	case "读取", "改动", "写入", "编辑":
		return true
	}
	return false
}

// timelineActionIcon 给动作取一个对象类型图标（UI-21）。
func timelineActionIcon(action string) *gui.SVG {
	switch action {
	case "读取":
		return iconEye
	case "改动", "编辑":
		return iconPencil
	case "写入":
		return iconFilePlus
	case "执行":
		return iconTerminal
	case "搜索":
		return iconSearch
	case "抓取":
		return iconGlobe
	default:
		return iconWrench
	}
}

// iconGlyphSize 是任意边长的图标盒（失败标记用 12，其它用 16）。
func iconGlyphSize(c *gui.Context, icon *gui.SVG, size float32, color gui.Color, mark string) {
	box := gui.Box(c).Width(size).Height(size).Shrink(0).
		Label("图标：" + mark).AccessHidden(true)
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		paintIcon(p, icon, r, color)
	})
}

// blockOf 取一个块的开合指针（可丢的视图状态，按需创建）。
func (a *App) blockOf(key string, initial bool) *bool {
	if a.blockOpen == nil {
		a.blockOpen = make(map[string]*bool)
	}
	if p, ok := a.blockOpen[key]; ok {
		return p
	}
	v := initial
	p := &v
	a.blockOpen[key] = p
	return p
}

// b2f 把开合布尔换成动画目标。
func b2f(open bool) float32 {
	if open {
		return 1
	}
	return 0
}

// chevron 是行末的方向指示：16×16 盒、单笔尖角，折叠时指向右、展开时指向下，
// 随开合旋转（U2 §4.4）。它不是独立按钮、不单独承担命中，也不进读屏（U3 §1.1
// L5）；标签留给测试与检查器定位，不参与读屏与行的可读名称。
func chevron(c *gui.Context, t *gui.Theme, progress float32) {
	chevronLabeled(c, t, progress, "方向指示")
}

// chevronLabeled 是带可定位标签的方向指示（不同块用不同标签，便于检查器定位）。
func chevronLabeled(c *gui.Context, t *gui.Theme, progress float32, label string) {
	box := gui.Box(c).Width(iconInline).Height(iconInline).Shrink(0).
		Label(label).AccessHidden(true)
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.StrokePolyline(chevronPoints(r, chevronAngle(progress)), strokeChevron, t.TextMuted)
	})
}

// chevronAngle 把开合进度换成方向指示的旋转角：折叠 0°（指向右），展开 90°
// （指向下）。动效关闭时框架的 Progress 直接给 0 或 1，角度也就瞬时到位（UI-5）。
func chevronAngle(progress float32) float32 { return 90 * progress }

// chevronPoints 是盒内的一条尖角折线，绕盒中心顺时针转 deg 度。折叠时尖角朝右，
// 转 90° 后朝下；纯函数，便于逐点核对方向（U2 §4.4）。
func chevronPoints(r gui.Rect, deg float32) [][2]float32 {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	d := r.W / 4
	local := [3][2]float32{{-d * 0.5, -d}, {d * 0.8, 0}, {-d * 0.5, d}}
	sin, cos := float32(math.Sin(float64(deg)*math.Pi/180)), float32(math.Cos(float64(deg)*math.Pi/180))
	pts := make([][2]float32, len(local))
	for i, p := range local {
		pts[i] = [2]float32{cx + p[0]*cos - p[1]*sin, cy + p[0]*sin + p[1]*cos}
	}
	return pts
}

// statusWord 只用于读屏名称：屏幕上不显示状态词（§9.0 A2「能画出来的不写」），
// 组头的可读名称由「内容短语 + 状态」合成（§9.0 可读下限）。
func statusWord(s roundStatus) string {
	switch s {
	case roundRunning:
		return "正在运行"
	case roundWaiting:
		return "等待你同意"
	case roundFailed:
		return "失败"
	case roundStopped:
		return "已停止"
	default:
		return "完成"
	}
}

// headerColor 让失败的一轮组头文字取警告色、被停止的一轮取弱化色，其余取正文
// 色；状态本身由行首图标与颜色表达（U2 §4.1、TL-48）。
func headerColor(t *gui.Theme, s roundStatus) gui.Color {
	switch s {
	case roundFailed:
		return t.Warning
	case roundStopped:
		return t.TextMuted
	}
	return t.Text
}

// roundHeaderPhrase 是本组的内容总览：做了什么——动作与代表对象，按首次出现
// 的顺序；不写状态词、计数与时长（§9.0 A3、D5）。对象多时取前面几个并收成
// 省略号，完整清单在展开里。没有动作的一组返回空串，调用方据此不出组头。
func roundHeaderPhrase(r round) string {
	if len(r.ops) == 0 {
		return ""
	}
	type group struct {
		action  string
		targets []string
	}
	var groups []group
	index := map[string]int{}
	for _, o := range r.ops {
		i, ok := index[o.action]
		if !ok {
			i = len(groups)
			index[o.action] = i
			groups = append(groups, group{action: o.action})
		}
		if t := shortTarget(o.target); !slices.Contains(groups[i].targets, t) {
			groups[i].targets = append(groups[i].targets, t)
		}
	}
	if len(groups) > headerGroupsMax {
		groups = groups[:headerGroupsMax]
		groups[headerGroupsMax-1].targets = append(groups[headerGroupsMax-1].targets, "…")
	}
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		targets := g.targets
		joined := strings.Join(targets, "、")
		if len(targets) > headerTargetsMax {
			joined += " …"
		}
		parts = append(parts, g.action+" "+joined)
	}
	return strings.Join(parts, " · ")
}

// headerAvailableWidth 是上面那条式子本身（纯函数，便于核对）。
func headerAvailableWidth(size func() (float32, float32), sidebarCollapsed, rightOpen bool) float32 {
	w, _ := size()
	sidebar := float32(SidebarWidth)
	if sidebarCollapsed {
		sidebar = SidebarRailWidth
	}
	panel := panelWidth(w, sidebar, rightOpen)
	return w - sidebar - panel - 2*space4 - iconInline - space1 - space1 - iconInline - space2 - space2
}

// fitPhrase 把组头收到可用的行宽内：先量（MeasureText），超宽才截；截断以对象
// 为单位（R7）——动作与对象的第一段一定留住，其余收成「…」。宽窗下不截。
func fitPhrase(c *gui.Context, t *gui.Theme, phrase string, sidebarCollapsed, rightOpen bool) string {
	width := headerAvailableWidth(c.Size, sidebarCollapsed, rightOpen)
	if phraseWidth(c, t, phrase) <= width {
		return phrase
	}
	shown := phrase
	for len([]rune(shown)) > 6 {
		shown = shortenPhrase(shown)
		if phraseWidth(c, t, shown) <= width {
			return shown
		}
	}
	return shown
}

// phraseWidth 量一句组头在组头字级下占的宽度。
func phraseWidth(c *gui.Context, t *gui.Theme, s string) float32 {
	w, _ := c.MeasureText(0, gui.Span{
		Text: s, Size: typeSecondaryStrong.sizeIn(t), Weight: typeSecondaryStrong.weight,
	})
	return w
}

// shortenPhrase 把一句组头按对象收一档：先去掉最后一个动作组，再把对象截到
// 第一段，最后才按字符收。
func shortenPhrase(phrase string) string {
	if i := strings.LastIndex(phrase, " · "); i > 0 {
		return phrase[:i] + " …"
	}
	if i := strings.Index(phrase, " "); i > 0 {
		head, obj := phrase[:i], phrase[i+1:]
		if first := firstSegment(strings.TrimSuffix(obj, " …")); first != "" &&
			first != strings.TrimSuffix(obj, " …") {
			return head + " " + first + " …"
		}
	}
	runes := []rune(strings.TrimSuffix(phrase, "…"))
	if len(runes) > 4 {
		runes = runes[:len(runes)-3]
	}
	return string(runes) + "…"
}

// truncatePhrase 以**对象**为单位截断组头（R7）：动作与对象的第一段一定留住，
// 多出来的在最后一个对象上收成「…」。整句超长时，逐个动作去掉尾部对象；仍超长
// 就把最后一个对象截到第一段（命令取第一个词、路径取末段），最后才收句。
func truncatePhrase(phrase string) string {
	if len([]rune(phrase)) <= headerPhraseMaxRunes {
		return phrase
	}
	// 先在最后一个「 · 」处收句，保留至少一个完整动作组。
	if i := strings.LastIndex(phrase, " · "); i > 0 {
		return truncatePhrase(phrase[:i] + " …")
	}
	// 只剩一个动作组：把对象截到第一段。
	if i := strings.Index(phrase, " "); i > 0 {
		head, obj := phrase[:i], phrase[i+1:]
		if first := firstSegment(obj); first != "" && first != obj {
			return head + " " + first + " …"
		}
	}
	// 连第一段都放不下：按字符硬收。
	runes := []rune(phrase)
	if len(runes) > headerPhraseMaxRunes-1 {
		runes = runes[:headerPhraseMaxRunes-1]
	}
	return string(runes) + "…"
}

// firstSegment 是对象的第一段：命令取第一个词，路径取末段。
func firstSegment(obj string) string {
	if i := strings.IndexAny(obj, " \t"); i > 0 {
		return obj[:i]
	}
	if i := strings.LastIndex(obj, "/"); i >= 0 && i+1 < len(obj) {
		return obj[i+1:]
	}
	return obj
}

// shortTarget 把操作对象写成组头里够用的短形式：文件路径取末段；命令与其它
// 带空格的原文保持原样（命令的末段不是它的名字）。
func shortTarget(s string) string {
	if strings.ContainsAny(s, " \t") {
		return s
	}
	if i := strings.LastIndex(s, "/"); i >= 0 && i+1 < len(s) {
		return s[i+1:]
	}
	return s
}

// roundMeta 是展开后的第一行：计数只在这里出现，组头与主界面不写计数与时长
// （§9.0 取向 2、UI-11）。
func roundMeta(r round) string {
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
	return meta
}

// liveTimeline 只画 Source 给的快照：按行的种类落到既有的块上（用户气泡、助手
// 正文、思考块、工具行与工具组、失败行、产物、改动、压缩分隔），不掺样例。
//
// 块的开合态按行序号存（可丢的视图状态），行的顺序与内容都只来自 Source。
func (a *App) liveTimeline(c *gui.Context, snap SessionSnapshot) {
	// 正在装载第一轮：画恢复遮罩，不把「还没有消息」当结论（TL-71）。
	if snap.LoadPhase == "loading_first_turn" {
		a.restoreMask(c)
		return
	}
	// 正在编辑的那一行被事实移走（消息被删或历史重载）：关掉编辑器，不把改写
	// 落到别的行上（TL-323）。
	if a.inlineEditRow >= len(snap.Rows) {
		a.cancelInlineEdit()
	}
	if len(snap.Rows) == 0 {
		a.emptyTimeline(c)
		return
	}
	gui.Box(c).Grow(1).Children(func() {
		feed := a.timelineFeed(c)
		feed.Children(func() {
			gui.Column(c).Padding(space6, timelinePadX).Gap(timelineRoundGap).
				Label("对话内容").Children(func() {
				// TL-06 视口外还有未装载的行时给等高占位。
				a.virtualTopPlaceholder(c)
				for i := range snap.Rows {
					a.liveRow(c, i, snap.Rows[i])
				}
			})
		})
		a.backToLatest(c)
	})
}

// liveRow 画一行事实：外层只负责用户消息轨道，行内容交给 liveRowBody。
// TL-04 / TL-05 轨道与标记按行种类出现。
func (a *App) liveRow(c *gui.Context, i int, row RowSnapshot) {
	gui.Column(c).Shrink(0).Children(func() {
		a.userRail(c, row.Kind == RowUser)
		a.liveRowBody(c, i, row)
	})
}

// liveRowBody 画一行事实：种类决定用哪个块，字段决定块里的文字；认不出的种类
// 仍按正文画（纯文本回退，不丢内容）。
func (a *App) liveRowBody(c *gui.Context, i int, row RowSnapshot) {
	switch row.Kind {
	case RowUser:
		a.userMessage(c, i, userRound(row))
	case RowThinking:
		a.thinkingBlock(c, i, thinkingRound(row))
	case RowToolGroup:
		if len(row.Children) == 0 {
			return
		}
		a.roundHeader(c, i, toolGroupRound(row))
	case RowTool:
		// TL-56 子代理通知行
		if subagentTool(row.Title) {
			a.subagentChip(c, row.Args, row.Text)
			return
		}
		a.opRow(c, i, 0, toolOp(row))
	case RowFileChanges:
		if len(row.Children) == 0 {
			return
		}
		a.fileChangesBlock(c, i, round{fileChanges: fileChangeRows(row)})
	case RowArtifact:
		if len(row.Children) == 0 {
			return
		}
		a.artifactsBlock(c, i, round{artifacts: artifactRows(row)})
	case RowError:
		a.failureBlock(c)
	case RowRetry:
		// 剩余时间是独立事实：拿不到（RetryIn <= 0）就不画这一行，不拿已耗时
		// 折算成"还剩"。
		if row.RetryIn <= 0 {
			return
		}
		a.retryNotice(c, round{retrySeconds: int(math.Ceil(row.RetryIn.Seconds()))})
	case RowCompression:
		a.compressionRow(c, compressionRowKey(i), row.Text, row.Status)
	default:
		a.liveText(c, i, row)
	}
}

// liveText 是纯文本回退：正文非空就按助手正文画；空行没有可画的内容。
//
// TL-18 系统通知文本 / TL-19 其它角色的消息正文：服务端的 system 行与认不出
// 种类的行都落到这里，正文原样画成一个文本块（不分系统/其它角色另起样式）。
func (a *App) liveText(c *gui.Context, i int, row RowSnapshot) {
	if strings.TrimSpace(row.Text) == "" {
		return
	}
	a.assistantText(c, i, row.Text)
}

// assistantText 是一段助手正文：整段可选，右键菜单按选区复制（TL-65）。
func (a *App) assistantText(c *gui.Context, i int, text string) {
	// TL-53 markdown 正文
	col := gui.Column(c).Shrink(0).Label("助手正文").Selectable()
	svc := c.Services()
	col = col.ContextMenu(func(m *gui.Menu) {
		a.assistantMessageMenu(m, svc, text, selectedTextOf(col))
	})
	col.Children(func() {
		body := a.assistantMarkdown(c, searchKeyAssistant(i), text)
		a.searchReveal(c, body, i, searchScopeAssistant)
	})
	if selection := selectedTextOf(col); selection != "" {
		// ⌘F 打开会话内搜索时预填这段选中文本（A1-03）。
		a.timelineSelected = selection
	}
}

// userRound 把用户行投成一条用户消息：正文与状态来自事实。技术口径：附件、引用、
// 标注与定时来源还没有进 RowSnapshot（DEC-07），所以这里不填；气泡里没有这些
// 内容时整块不画，不摆占位也不写内部说明。
func userRound(row RowSnapshot) round {
	return round{user: row.Text, status: liveRoundStatus(row.Status)}
}

// assistantMessageText 是「复制整个消息」要复制的正文：回复原文；没有回复时
// 退回思考原文（这一轮只留下了思考）。
func assistantMessageText(r round) string {
	if strings.TrimSpace(r.reply) != "" {
		return r.reply
	}
	return r.thinking
}

// thinkingRound 把思考行投成思考块要的一轮（标题里的时长由事实给出）。
func thinkingRound(row RowSnapshot) round {
	return round{
		thinking:         row.Text,
		thinkingDuration: rowDurationText(row.Duration),
		status:           liveRoundStatus(row.Status),
	}
}

// toolGroupRound 把工具组投成组头的形状：动作、对象、结果都来自子行。
func toolGroupRound(row RowSnapshot) round {
	ops := make([]op, 0, len(row.Children))
	for _, child := range row.Children {
		ops = append(ops, toolOp(child))
	}
	return round{ops: ops, status: liveRoundStatus(row.Status)}
}

// toolOp 把工具行投成操作行：Title 是动作、Args 是参数摘要、Text 是结果原文。
// 输入原文没有进快照，所以行里只展开输出。
func toolOp(row RowSnapshot) op {
	return op{
		action:   row.Title,
		target:   row.Args,
		output:   row.Text,
		added:    row.DiffAdd,
		removed:  row.DiffRemove,
		failed:   row.Status == RowStatusFailed || row.Status == RowStatusCancelled,
		terminal: row.Title == "执行",
	}
}

// fileChangeRows 把改动块的子行投成改动文件行。
func fileChangeRows(row RowSnapshot) []fileChange {
	out := make([]fileChange, 0, len(row.Children))
	for _, child := range row.Children {
		out = append(out, fileChange{
			path:    child.Text,
			status:  child.Args,
			added:   child.DiffAdd,
			removed: child.DiffRemove,
		})
	}
	return out
}

// artifactRows 把产物块的子行投成产物卡。
func artifactRows(row RowSnapshot) []artifact {
	out := make([]artifact, 0, len(row.Children))
	for _, child := range row.Children {
		name := child.Title
		if name == "" {
			name = child.Text
		}
		out = append(out, artifact{name: name})
	}
	return out
}

// liveRoundStatus 把行状态换成块的四种状态；取消是「被用户停止」，与失败分开画。
func liveRoundStatus(status string) roundStatus {
	switch status {
	case RowStatusRunning, RowStatusPending:
		return roundRunning
	case RowStatusCancelled:
		return roundStopped
	case RowStatusFailed:
		return roundFailed
	default:
		return roundDone
	}
}

// rowDurationText 把时长写成块标题里的一档（0.3s / 12s / 1m30s）；0 表示没有
// 时长事实，返回空串，块标题就不写时长。
func rowDurationText(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	return fmt.Sprintf("%dm%ds", seconds/60, seconds%60)
}

// emptyTimeline 是"会话里还没有消息"的空态（与无会话的空态区分）。
// 还没登录且没有供应商时，空态下面给两条首次运行入口（见 firstRunNeeded）。
func (a *App) emptyTimeline(c *gui.Context) {
	// TL-02 当前会话还没有消息
	t := c.Theme()
	gui.Box(c).Grow(1).Center().Label("对话流").Role(gui.RoleLog).Children(func() {
		gui.Column(c).AlignItems(gui.Center).Gap(space4).Children(func() {
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Label("空态").Children(func() {
				iconGlyph(c, iconChat, t.TextMuted, "chat")
				typeSecondary.style(t, gui.Text(c, "当前会话还没有消息")).TextColor(t.TextMuted)
			})
			if a.firstRunNeeded() {
				a.firstRunGuide(c)
			}
		})
	})
}

// timelineSkeleton 是会话切换/恢复中的骨架屏（TL-09）：三条错落的灰条。
// 当前没有「加载遮罩可见」的事实来源，先留形状；接上加载态后再挂载。
func (a *App) timelineSkeleton(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Grow(1).Padding(space6, timelinePadX).Gap(space3).Label("加载中").Children(func() {
		for _, w := range []float32{70, 92, 54} {
			gui.Box(c).Height(space2).WidthPercent(w).Radius(radiusSmall).
				Background(t.SurfacePressed)
		}
	})
}
