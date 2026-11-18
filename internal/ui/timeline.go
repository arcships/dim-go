package ui

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
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
		gui.Scroll(c).Grow(1).Fill().TrackScroll(&a.feed).
			Label("对话流").Role(gui.RoleLog).Focusable().TabStop(true).Children(func() {
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

// round 画一轮：用户消息右对齐，下面是助手区（思考、正文、工具分组、轮末块）。
func (a *App) round(c *gui.Context, i int) {
	r := a.rounds[i]
	if r.unknown {
		// 未知提交：只有占位（原文还在本地）与说明行；没有组头、没有操作行，
		// 因为「这一轮是否存在」本身还没有事实。
		gui.Column(c).Shrink(0).Gap(space2).Children(func() {
			a.userBubble(c, r, true)
			a.note(c, r.note)
		})
		return
	}
	gui.Column(c).Shrink(0).Gap(timelineBlockGap).Children(func() {
		if r.divider != "" {
			a.roundDivider(c, r.divider)
		}
		a.userMessage(c, i, r)
		a.assistant(c, i, r)
	})
}

// roundDivider 是轮前的分隔行（压缩 / 窗口截断）：两侧细线 + 中间说明。
func (a *App) roundDivider(c *gui.Context, text string) {
	t := c.Theme()
	gui.Row(c).Grow(1).Gap(space2).AlignItems(gui.Center).Label(text).Children(func() {
		gui.Box(c).Grow(1).Height(strokeHairline).Background(t.Border)
		typeMicro.style(t, gui.Text(c, text)).TextColor(t.TextMuted).SingleLine()
		gui.Box(c).Grow(1).Height(strokeHairline).Background(t.Border)
	})
}

// userMessage 是用户消息行：气泡右对齐；悬停时在气泡下方出现发送时间与动作行。
func (a *App) userMessage(c *gui.Context, i int, r round) {
	col := gui.Column(c).Shrink(0).Gap(space1).Label("用户消息 " + r.user)
	hovered := col.Hovered()
	col.Children(func() {
		a.userBubble(c, r, false)
		if hovered {
			a.userActions(c, i, r)
		}
	})
}

// userBubble 是气泡本体：右对齐、最多 85% 内容宽、浅底圆角。
func (a *App) userBubble(c *gui.Context, r round, pending bool) {
	t := c.Theme()
	gui.Row(c).Grow(1).Children(func() {
		gui.Spacer(c)
		box := gui.Box(c).MaxWidthPercent(userBubbleMaxPercent).
			Padding(space2, timelinePadX).Radius(radiusCard).
			Background(t.SurfaceHover).Label("我的消息")
		if pending {
			box = box.Opacity(0.6)
		}
		box.Children(func() {
			typeBody.style(t, gui.Text(c, r.user)).TextColor(t.Text).Selectable()
		})
	})
}

// userActions 是用户气泡下的发送时间与四个动作（悬停可见）。
func (a *App) userActions(c *gui.Context, i int, r round) {
	t := c.Theme()
	gui.Row(c).Grow(1).Gap(space1).AlignItems(gui.Center).Label("消息动作").Children(func() {
		gui.Spacer(c)
		if r.at != "" {
			typeMicro.style(t, gui.Text(c, r.at)).TextColor(t.TextMuted).
				Label("发送时间 " + r.at).SingleLine()
		}
		iconButton(c, iconButtonSpec{icon: iconCopy, label: "复制整个消息 " + fmt.Sprint(i+1), size: rowCompact})
		iconButton(c, iconButtonSpec{icon: iconPencil, label: "编辑消息 " + fmt.Sprint(i+1), size: rowCompact})
		iconButton(c, iconButtonSpec{icon: iconFork, label: "从此处分叉 " + fmt.Sprint(i+1), size: rowCompact})
		iconButton(c, iconButtonSpec{icon: iconBranch, label: "从此处开始侧边对话 " + fmt.Sprint(i+1), size: rowCompact})
	})
}

// assistant 是助手区：思考、正文、工具分组、轮末块与动作条。
func (a *App) assistant(c *gui.Context, i int, r round) {
	col := gui.Column(c).Shrink(0).Gap(timelineBlockGap).Label("助手消息")
	hovered := col.Hovered()
	col.Children(func() {
		if r.thinking != "" {
			a.thinkingBlock(c, i, r)
		}
		if r.reply != "" {
			markdown.View(c, r.reply, &a.md)
		}
		if len(r.ops) > 0 {
			a.roundHeader(c, i, r)
		}
		if r.status == roundFailed {
			a.failureBlock(c)
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
		if r.status == roundRunning {
			a.processingLine(c, r)
		}
		if hovered {
			a.assistantActions(c, r)
		}
	})
}

// thinkingBlock 是思考块：灯泡 + 「思考了 Xs」+ 箭头；展开为引用块。
// 用 RoleButton 而不是 Disclosure，保证轮次组头仍是第一个可展开分组（读屏顺序）。
func (a *App) thinkingBlock(c *gui.Context, i int, r round) {
	t := c.Theme()
	open := a.blockOf(fmt.Sprintf("think:%d", i), false)
	title := "思考了 " + r.thinkingDuration
	if r.status == roundRunning {
		title = "正在思考 · " + r.thinkingDuration
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
	a.revealPanel(c, *open, progress, func() {
		gui.Row(c).Grow(1).Gap(space2).Children(func() {
			rail := gui.Box(c).Width(strokeHairline).Shrink(0).AlignSelf(gui.Stretch).Label("思考轨道")
			rail.DrawOver(func(p *gui.Painter, rect gui.Rect) {
				p.Fill(rect, t.Border, 0)
			})
			typeSecondary.style(t, gui.Text(c, r.thinking)).TextColor(t.TextMuted).
				Grow(1).Selectable()
		})
	})
}

// toolGroup 是连续工具调用的折叠组：分组头写动作汇总，展开后是缩进的工具行。
func (a *App) roundHeader(c *gui.Context, i int, r round) {
	t := c.Theme()
	open := a.openOf(i)
	summary := groupSummary(r)
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
		statusMark(c, t, r.status)
		iconGlyph(c, iconWorkflow, t.TextMuted, "workflow")
		typeSecondaryStrong.style(t, gui.Text(c, summary)).TextColor(headerColor(t, r.status)).SingleLine()
		chevron(c, t, arrow)
	})
	if btn.Clicked() {
		*open = !*open
	}
	a.revealPanel(c, *open, progress, func() {
		gui.Row(c).Grow(1).Gap(space2).Children(func() {
			rail := gui.Box(c).Width(timelineGroupIndent).Shrink(0).AlignSelf(gui.Stretch).Label("分组轨道")
			rail.DrawOver(func(p *gui.Painter, rect gui.Rect) {
				x := rect.X + timelineGroupIndent/2
				p.Fill(gui.Rect{X: x, Y: rect.Y, W: strokeHairline, H: rect.H}, t.Border, 0)
			})
			gui.Column(c).Grow(1).Gap(space3).Children(func() {
				for n, o := range r.ops {
					a.opRow(c, i, n, o)
					if o.failed && o.cause != "" {
						typeSecondary.style(t, gui.Text(c, o.cause)).TextColor(t.TextMuted).
							PaddingStart(timelineGroupIndent)
					}
				}
			})
		})
	})
}

// toolRow 是一行工具调用：图标 + 动词 + 参数 + 差异计数/失败标记 + 箭头。
func (a *App) opRow(c *gui.Context, roundIndex, opIndex int, o op) {
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
		iconGlyph(c, timelineActionIcon(o.action), t.TextMuted, "tool")
		typeSecondary.style(t, gui.Text(c, o.action)).TextColor(t.TextMuted).SingleLine()
		if fileAction(o.action) {
			iconGlyph(c, iconDocument, t.TextMuted, "file")
		}
		typeSecondary.style(t, gui.Text(c, display)).TextColor(t.Text).Shrink(1).SingleLine()
		if o.added > 0 {
			typeMicro.style(t, gui.Text(c, fmt.Sprintf("+%d", o.added))).TextColor(t.Success)
		}
		if o.removed > 0 {
			typeMicro.style(t, gui.Text(c, fmt.Sprintf("-%d", o.removed))).TextColor(t.Danger)
		}
		if o.failed {
			iconGlyphSize(c, iconXMark, 12, t.Warning, "工具失败")
		}
		if hasDetails {
			chevronLabeled(c, t, arrow, "工具展开指示")
		}
	})
	if btn.Clicked() && hasDetails {
		*open = !*open
	}
	if !hasDetails {
		return
	}
	a.revealPanel(c, *open, progress, func() {
		a.toolDetail(c, o)
	})
}

// toolDetail 是工具行的展开区：输入 / 输出 / 失败原因，等宽浅底。
func (a *App) toolDetail(c *gui.Context, o op) {
	t := c.Theme()
	gui.Column(c).Gap(space1).PaddingStart(timelineGroupIndent).Children(func() {
		if o.input != "" {
			typeMicro.style(t, gui.Text(c, "输入")).TextColor(t.TextMuted)
			a.codeBlock(c, o.input)
		}
		if o.output != "" {
			typeMicro.style(t, gui.Text(c, "输出")).TextColor(t.TextMuted)
			a.codeBlock(c, o.output)
		}
		if o.cause != "" {
			typeSecondary.style(t, gui.Text(c, o.cause)).TextColor(t.Danger)
		}
	})
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
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Start).Children(func() {
		iconGlyph(c, iconWarn, t.Danger, "failure")
		gui.Column(c).Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, "运行失败")).TextColor(t.Danger)
			typeSecondary.style(t, gui.Text(c, "本次运行失败。")).TextColor(t.Danger)
		})
	})
}

// retryNotice 是自动重试提示：倒计时 + 立即重试（可逆动作）。
func (a *App) retryNotice(c *gui.Context, r round) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
		typeSecondary.style(t, gui.Text(c, fmt.Sprintf("%ds 后重试", r.retrySeconds))).
			TextColor(t.TextMuted)
		gui.Button(c, "立即重试").Label("立即重试")
	})
}

// artifactsBlock 是本轮产物：计数 + 展开列表（每卡一个打开入口）。
func (a *App) artifactsBlock(c *gui.Context, i int, r round) {
	t := c.Theme()
	open := a.blockOf(fmt.Sprintf("artifacts:%d", i), false)
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
		gui.Row(c).Wrap().Gap(space2).PaddingStart(timelineGroupIndent).Children(func() {
			for _, art := range r.artifacts {
				a.artifactCard(c, art)
			}
		})
	})
}

// artifactCard 是一个产物卡：文件名 + 打开。
func (a *App) artifactCard(c *gui.Context, art artifact) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Padding(space2, space3).Gap(space2).Radius(radius).
		Background(t.Surface).AlignItems(gui.Center).Label("产物 " + art.name).Children(func() {
		iconGlyph(c, iconDocument, t.TextMuted, "artifact")
		typeSecondary.style(t, gui.Text(c, art.name)).TextColor(t.Text).SingleLine()
		gui.Button(c, "打开").Label("打开产物 " + art.name)
	})
}

// fileChangesBlock 是本轮文件改动：计数 + 展开列表（文件、状态、差异）。
func (a *App) fileChangesBlock(c *gui.Context, i int, r round) {
	t := c.Theme()
	open := a.blockOf(fmt.Sprintf("files:%d", i), false)
	title := fmt.Sprintf("%d 个文件改动", len(r.fileChanges))
	btn := gui.ButtonBase(c).Height(timelineRowHeight).Gap(space1).
		Justify(gui.Start).AlignItems(gui.Center).Padding(0, space1).Radius(radius).
		Cursor(gui.CursorPointer).Label(title).Role(gui.RoleDisclosure).Expanded(*open)
	if btn.Hovered() {
		btn = btn.Background(t.SurfaceHover)
	}
	progress := btn.AnimateTo("files", b2f(*open), timelineCollapse)
	arrow := btn.AnimateTo("files-arrow", b2f(*open), timelineArrow)
	btn.Children(func() {
		iconGlyph(c, iconDocument, t.TextMuted, "changes")
		typeSecondary.style(t, gui.Text(c, title)).TextColor(t.Text)
		chevronLabeled(c, t, arrow, "改动展开指示")
	})
	if btn.Clicked() {
		*open = !*open
	}
	a.revealPanel(c, *open, progress, func() {
		gui.Column(c).Gap(space1).PaddingStart(timelineGroupIndent).Children(func() {
			for _, change := range r.fileChanges {
				a.fileChangeRow(c, change)
			}
		})
	})
}

// fileChangeRow 是一个改动文件：图标 + 文件名 + 状态 + 增删数。
func (a *App) fileChangeRow(c *gui.Context, change fileChange) {
	t := c.Theme()
	gui.Row(c).Grow(1).Height(rowRegular).Gap(space2).Padding(0, space2).
		AlignItems(gui.Center).Label("改动文件 " + change.path).Children(func() {
		iconGlyph(c, iconDocument, t.TextMuted, "file")
		typeSecondary.style(t, gui.Text(c, change.path)).TextColor(t.Text).Grow(1).SingleLine()
		typeMicro.style(t, gui.Text(c, change.status)).TextColor(t.TextMuted)
		if change.added > 0 {
			typeMicro.style(t, gui.Text(c, fmt.Sprintf("+%d", change.added))).TextColor(t.Success)
		}
		if change.removed > 0 {
			typeMicro.style(t, gui.Text(c, fmt.Sprintf("-%d", change.removed))).TextColor(t.Danger)
		}
	})
}

// processingLine 是运行中的轮末提示：正在处理 · {时长}（TL-70）。
func (a *App) processingLine(c *gui.Context, r round) {
	t := c.Theme()
	duration := r.duration
	if duration == "" {
		duration = "0s"
	}
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
		iconGlyph(c, iconDot, colorAccentInk.in(t), "activity")
		typeSecondary.style(t, gui.Text(c, "正在处理 · "+duration)).TextColor(t.TextMuted)
	})
}

// assistantActions 是轮末动作条：复制、分叉、重试与完成时间（悬停可见）。
func (a *App) assistantActions(c *gui.Context, r round) {
	t := c.Theme()
	gui.Row(c).Gap(space1).AlignItems(gui.Center).Label("助手动作").Children(func() {
		iconButton(c, iconButtonSpec{icon: iconCopy, label: "复制整个消息", size: rowCompact})
		iconButton(c, iconButtonSpec{icon: iconFork, label: "从此处分叉", size: rowCompact})
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
	box := gui.Box(c).Width(size).Height(size).Shrink(0).Label("图标：" + mark)
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Icon(icon, r, color)
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

// statusMark 是过程摘要行的行首状态位（U2 §4.1、§6.4、校准 C5）：
// 固定 16×16 盒，尺寸不随字号变化；运行中是居中的 8px 实心圆，其余是
// Heroicons outline 图标（24 网格 1.5 描边 → 16px 时 1.0px）。
func statusMark(c *gui.Context, t *gui.Theme, s roundStatus) {
	if s == roundRunning {
		box := gui.Box(c).Width(iconInline).Height(iconInline).Shrink(0).
			Label("状态：" + statusWord(s))
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
		iconBox(c, iconClock, colorAccentInk.in(t), "状态："+statusWord(s))
	case roundFailed:
		iconBox(c, iconXMark, t.Danger, "状态："+statusWord(s))
	default:
		iconBox(c, iconCheck, t.TextMuted, "状态："+statusWord(s))
	}
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
	default:
		return "完成"
	}
}

// headerColor 让失败的一轮组头文字取 text-secondary（弱化），其余取正文色；
// 状态本身由行首图标与颜色表达（U2 §4.1）。
func headerColor(t *gui.Theme, s roundStatus) gui.Color {
	if s == roundFailed {
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

// liveTimeline 只画 Source 给的快照：用户消息右对齐，其余按行类型落块。
// 完整的工具/思考事实要等 Source 补字段；当前先保证「事实进来、样例不掺」。
func (a *App) liveTimeline(c *gui.Context, snap SessionSnapshot) {
	if len(snap.Rows) == 0 {
		a.emptyTimeline(c)
		return
	}
	t := c.Theme()
	gui.Box(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Fill().TrackScroll(&a.feed).
			Label("对话流").Role(gui.RoleLog).Focusable().TabStop(true).Children(func() {
			gui.Column(c).Padding(space6, timelinePadX).Gap(timelineRoundGap).
				Label("对话内容").Children(func() {
				for _, row := range snap.Rows {
					row := row
					switch row.Kind {
					case "user":
						gui.Row(c).Grow(1).Children(func() {
							gui.Spacer(c)
							gui.Box(c).MaxWidthPercent(userBubbleMaxPercent).
								Padding(space2, timelinePadX).Radius(radiusCard).
								Background(t.SurfaceHover).Children(func() {
								typeBody.style(t, gui.Text(c, row.Text)).TextColor(t.Text).Selectable()
							})
						})
					case "group":
						gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
							iconGlyph(c, iconWorkflow, t.TextMuted, "workflow")
							typeSecondaryStrong.style(t, gui.Text(c, row.Text)).TextColor(t.Text)
						})
					default:
						markdown.View(c, row.Text, &a.md)
					}
				}
			})
		})
		a.backToLatest(c)
	})
}

// emptyTimeline 是"会话里还没有消息"的空态（与无会话的空态区分）。
func (a *App) emptyTimeline(c *gui.Context) {
	t := c.Theme()
	gui.Box(c).Grow(1).Center().Label("对话流").Role(gui.RoleLog).Children(func() {
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Label("空态").Children(func() {
			iconGlyph(c, iconChat, t.TextMuted, "chat")
			typeSecondary.style(t, gui.Text(c, "当前会话还没有消息")).TextColor(t.TextMuted)
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
