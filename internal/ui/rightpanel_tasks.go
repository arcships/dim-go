package ui

import (
	"fmt"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 任务面板（右栏「后台」标签）：后台进程与唤醒两个子标签共用一套列表，
// 选中一行后下方给详情。
//
// 事实口径：
//   - 进程行有标识、描述、状态、开始时刻与分支；pid、结束时刻、命令行与日志
//     状态在 Go 侧没有读取通道时留空（详情各段按有无拼接，不编造）。
//   - 唤醒行取 Wakeups()（`session.wakeup.list` 的读数）：触发时刻与提示正文
//     是事实。
//   - 「打开日志」要把服务端的日志文件交给系统打开：宿主没有这个通道，入口
//     保留但置灰并写原因（K-27）。
//
// 条目编号见 docs/domains/requirements/requirements-panels.md §5（K-01–K-36）。

// taskCopiedFeedback 是「已复制」反馈的停留时长（K-26：2 秒）。
const taskCopiedFeedback = 2 * time.Second

// taskStateText 是行与详情头共用的状态词：屏幕上的字由数据的取值决定，
// 不写字面量状态（UI-23 / L6）。
func taskStateText(st taskState) string {
	switch st {
	case taskDone:
		return "已完成"
	case taskFailed:
		return "失败"
	case taskStopped:
		return "已停止"
	case taskNoStatus:
		return "状态未知"
	}
	return "运行中"
}

// taskStateIconIn 按主题取状态图标与颜色（K-05–K-09）：运行中强调色、完成
// 次要色、失败危险色、已停止中性方块、状态未知时钟。
func taskStateIconIn(st taskState, th *gui.Theme) (*gui.SVG, gui.Color) {
	muted := gui.Color{}
	accent := gui.Color{}
	danger := gui.Color{}
	if th != nil {
		muted, accent, danger = th.TextMuted, colorAccentInk.in(th), th.Danger
	}
	switch st {
	case taskDone:
		return iconCheck, muted
	case taskFailed:
		return iconXMark, danger
	case taskStopped:
		return iconStop, muted
	case taskNoStatus:
		return iconClock, muted
	}
	return iconSidebarSpinner, accent
}

// taskStateLine 是详情头的状态文本（K-21–K-24）：「{状态} · {时刻}」；没有
// 时刻时只写状态，不留一个空的分隔点。运行中给出已运行时长。
func taskStateLine(item task) string {
	state := taskStateText(item.state)
	if item.kind == taskWakeup {
		return state
	}
	if item.state == taskRunning {
		if elapsed := taskElapsed(item); elapsed != "" {
			return state + " · " + elapsed
		}
		return state
	}
	if item.endedAt == "" {
		return state
	}
	return state + " · " + item.endedAt
}

// taskElapsed 是运行中任务的已运行时长（秒级）；没有开始时刻时为空。
func taskElapsed(item task) string {
	if item.startAt.IsZero() {
		return ""
	}
	elapsed := time.Since(item.startAt)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed < time.Minute {
		return fmt.Sprintf("%ds", int(elapsed.Seconds()))
	}
	return fmt.Sprintf("%dm %ds", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
}

// taskMetaLine 是元信息行（K-28）：{工具} · {开始} · pid {pid} · 结束于 {时刻}，
// 各段按有无拼接；全空时不画这一行。
func taskMetaLine(item task) string {
	parts := make([]string, 0, 4)
	if item.toolName != "" {
		parts = append(parts, item.toolName)
	}
	if item.startedAt != "" {
		parts = append(parts, item.startedAt)
	}
	if item.pid != "" {
		parts = append(parts, "pid "+item.pid)
	}
	if item.endedAt != "" {
		parts = append(parts, "结束于 "+item.endedAt)
	}
	return strings.Join(parts, " · ")
}

// tasksPanel 是任务面板：子标签（后台进程 / 唤醒）+ 列表 + 详情 + 空态。
func (a *App) tasksPanel(c *gui.Context) {
	gui.Column(c).Grow(1).Children(func() {
		gui.Row(c).Height(rightPanelTabHeight).Shrink(0).
			Padding(space1, panelPad).AlignItems(gui.Center).Children(func() {
			// K-02 子标签「后台进程」/「唤醒」
			gui.Tabs(c, &a.panels.taskKind, "后台进程", "唤醒").Label("任务类型")
		})
		if a.panels.taskKind == 1 {
			items, note := a.sessionWakeups()
			if len(items) == 0 {
				// K-19 唤醒页的空态与原因
				panelEmpty(c, "暂无待唤醒事项"+note)
				return
			}
			selected := a.taskSelection(len(items))
			a.taskList(c, items, note == "")
			if selected >= 0 {
				gui.Divider(c)
				a.taskDetail(c, items[selected])
			}
			return
		}
		items, live := a.sessionTasks()
		if len(items) == 0 {
			// K-03 后台进程的空态
			panelEmpty(c, "暂无运行进程")
			return
		}
		selected := a.taskSelection(len(items))
		a.taskList(c, items, live)
		if selected >= 0 {
			gui.Divider(c)
			a.taskDetail(c, items[selected])
		}
	})
}

// taskSelection 是当前选中行在本页的位次；没有选中或越界时为 -1（详情不画）。
func (a *App) taskSelection(count int) int {
	if a.panels.taskSel < 0 || a.panels.taskSel >= count {
		return -1
	}
	return a.panels.taskSel
}

// taskList 是列表容器（K-01 / K-02）：可滚动、逐行渲染。
func (a *App) taskList(c *gui.Context, items []task, live bool) {
	label := "后台进程"
	if a.panels.taskKind == 1 {
		label = "唤醒"
	}
	gui.Scroll(c).Grow(1).Children(func() {
		gui.Column(c).Gap(0).Role(gui.RoleList).Label(label).Children(func() {
			for i := range items {
				a.taskRow(c, items[i], i, live)
				if i < len(items)-1 {
					gui.Divider(c)
				}
			}
		})
	})
}

// taskRow 是一行后台任务：状态图标 + 任务名（内容）+ 分支徽标 + 时间 +
// 行尾停止入口（K-03–K-16）。
func (a *App) taskRow(c *gui.Context, item task, index int, live bool) {
	th := c.Theme()
	state := taskStateText(item.state)
	selected := a.panels.taskSel == index
	row := gui.Row(c).Shrink(0).Height(panelRowHeight).
		Padding(space1, panelPad).Gap(space2).AlignItems(gui.Center)
	if selected {
		row = row.Background(th.SurfaceHover)
	}
	row.Children(func() {
		label := "后台任务：" + item.name + " · " + state
		if item.kind == taskWakeup {
			label = "唤醒：" + item.name
			if item.wakeAt != "" {
				label += " · " + item.wakeAt
			}
		}
		// K-04 行主按钮（状态图标 + 标题）
		main := gui.ButtonBase(c).Grow(1).Height(rowRegular).Padding(space1, space1).
			Radius(radiusSmall).Gap(space2).AlignItems(gui.Center).
			Role(gui.RoleListItem).AccessSelected(selected).
			Label(label).Tooltip(item.name)
		if main.Clicked() {
			a.panels.taskSel = index
			a.panels.taskArmed = -1
		}
		main.Children(func() {
			// K-05–K-09 行首状态图标（运行 / 完成 / 失败 / 已停止 / 时钟）
			a.taskStateGlyph(c, item, th)
			// K-10 任务标题
			typeSecondaryStrong.style(th, gui.Text(c, item.name)).TextColor(th.Text).
				Grow(1).SingleLine()
			// K-11 分支徽标：main 与空分支不画（参考产品同一条件）。
			if item.branch != "" && item.branch != "main" {
				taskBranchBadge(c, th, item.branch)
			}
			// K-12 任务时间（等宽数字）。
			if at := taskRowTime(item); at != "" {
				typeMicro.style(th, gui.Text(c, at)).TextColor(th.TextMuted).SingleLine()
			}
		})
		switch {
		case item.kind == taskWakeup:
			// K-16 唤醒行行尾的旋转圈（没有停止入口）。
			taskTrailingSpinner(c, th, "唤醒进行中")
		case item.pending:
			// K-15 已发停止、等结束：行尾换成禁用的旋转圈。
			taskTrailingSpinner(c, th, "正在停止")
		case item.state == taskRunning:
			a.taskStopButton(c, item, index, live)
		}
	})
}

// taskRowTime 是行上的时间（K-12）：进程取开始时刻，唤醒取触发时刻。
func taskRowTime(item task) string {
	if item.kind == taskWakeup {
		return item.wakeAt
	}
	return item.startedAt
}

// taskStateGlyph 是行首 / 详情头的状态图标（K-05–K-09、K-20）：运行中的圈
// 持续旋转，其余是静态形状。
func (a *App) taskStateGlyph(c *gui.Context, item task, th *gui.Theme) {
	icon, color := taskStateIconIn(item.state, th)
	mark := "状态：" + taskStateText(item.state)
	if item.state == taskRunning {
		// 运行中的圈持续旋转；盒子的读屏名与静态图标同一口径
		// （「图标：task」，见 U2-elements 的后台任务行）。
		spin := gui.Box(c).Width(iconInline).Height(iconInline).Shrink(0).
			Label("图标：task").AccessHidden(true)
		spin.DrawOver(func(p *gui.Painter, r gui.Rect) {
			paintIcon(p, icon, r, color)
		})
		spin.Rotate(spin.Loop("task-running", time.Second, gui.Linear) * 360)
		return
	}
	iconGlyphSize(c, icon, iconInline, color, mark)
}

// taskBranchBadge 是行里的分支徽标（K-11）：10px 等宽、小圆角、面差底。
func taskBranchBadge(c *gui.Context, th *gui.Theme, branch string) {
	gui.Row(c).Shrink(0).Height(sidebarBadgeHeight).Padding(0, space1).
		Radius(radiusSmall).Background(th.SurfacePressed).AlignItems(gui.Center).
		Label("分支：" + branch).Children(func() {
		typeMicro.style(th, gui.Text(c, branch)).TextColor(th.TextMuted).SingleLine()
	})
}

// taskTrailingSpinner 是行尾的旋转圈（K-15 / K-16）：禁用态，只表示进行中。
func taskTrailingSpinner(c *gui.Context, th *gui.Theme, label string) {
	box := gui.Row(c).Width(rowCompact).Height(rowCompact).Shrink(0).
		AlignItems(gui.Center).Justify(gui.Center).Label(label)
	box.Children(func() {
		spin := gui.Icon(c, iconSidebarSpinner).Size(iconInline, iconInline).
			TextColor(th.TextMuted).AccessHidden(true)
		spin.Rotate(spin.Loop("task-trailing", time.Second, gui.Linear) * 360)
	})
}

// taskStopButton 是行尾的停止入口（K-13 / K-14）：第一次点进待确认（勾形 +
// 危险色），再点发停止请求。参考的「3 秒或失焦自动复位」还没有计时器（未实现），
// 现在只在选中行变化时复位。
func (a *App) taskStopButton(c *gui.Context, item task, index int, live bool) {
	th := c.Theme()
	armed := a.panels.taskArmed == index
	label, icon := "取消任务", iconXMark
	color := th.Danger
	if armed {
		label, icon = "确认停止", iconCheck
	}
	if iconButton(c, iconButtonSpec{icon: icon, label: label, size: rowCompact, color: color}) {
		if !armed {
			a.panels.taskArmed = index
			return
		}
		a.panels.taskArmed = -1
		a.intentStopTask(c, item.id, live)
	}
}

// taskDetail 是选中行的详情（K-17）：状态头 + 元信息 + 命令行 + 只读输出框；
// 唤醒行换成唤醒详情（K-35 / K-36）。
func (a *App) taskDetail(c *gui.Context, item task) {
	if item.kind == taskWakeup {
		a.wakeupDetail(c, item)
		return
	}
	th := c.Theme()
	gui.Scroll(c).Grow(1).Children(func() {
		gui.Column(c).Padding(panelPad).Gap(panelGap).Children(func() {
			gui.Row(c).Shrink(0).Gap(space2).AlignItems(gui.Center).Children(func() {
				// K-20 详情头的状态图标
				a.taskStateGlyph(c, item, th)
				// K-21–K-24 详情头的状态文本（状态 · 时刻）
				typeSecondaryStrong.style(th, gui.Text(c, taskStateLine(item))).
					TextColor(th.Text).Grow(1).SingleLine()
				a.taskCopyCommand(c, item)
				// K-27 打开日志：日志在服务端，本端没有打开文件的宿主通道。
				iconButton(c, iconButtonSpec{
					icon: iconExternal, label: "打开日志", tip: "打开日志 · " + noteTaskLog,
					size: rowCompact, disabled: true,
				})
			})
			// K-28 元信息行（各段按有无拼接）
			if meta := taskMetaLine(item); meta != "" {
				typeMicro.style(th, gui.Text(c, meta)).TextColor(th.TextMuted).
					SingleLine()
			}
			// K-29 命令行（$ 前缀，等宽、可换行）
			if command := panelCommandText(item.command); command != "" {
				typeCode.style(th, gui.Text(c, command)).TextColor(th.Text).
					Selectable().MaxLines(4)
			}
			a.taskOutput(c, item)
		})
	})
}

// taskCopyCommand 是「复制命令」（K-25）与「已复制」两秒反馈（K-26）。
// 没有命令原文时不画入口（参考产品同一条件：item.command 未定义）。
func (a *App) taskCopyCommand(c *gui.Context, item task) {
	if item.command == "" {
		return
	}
	th := c.Theme()
	if a.taskCopied(item) {
		// K-26 已复制：图标 + 文本，两秒后回到复制入口；读屏播报一次。
		gui.Row(c).Shrink(0).Gap(space1).AlignItems(gui.Center).
			Label("已复制").Children(func() {
			iconGlyph(c, iconCheck, th.Success, "已复制")
			typeMicro.style(th, gui.Text(c, "已复制")).TextColor(th.Success)
		})
		return
	}
	if iconButton(c, iconButtonSpec{
		icon: iconClipboard, label: "复制命令", tip: "复制命令", size: rowCompact,
	}) {
		a.copyTaskCommand(c, item)
	}
}

// taskCopied 报告「已复制」反馈是否还在两秒窗口内。
func (a *App) taskCopied(item task) bool {
	if a.panels.taskCopiedID == "" || a.panels.taskCopiedID != item.id {
		return false
	}
	return time.Since(a.panels.taskCopiedAt) < taskCopiedFeedback
}

// copyTaskCommand 写剪贴板并起两秒计时；失败时把原因写进面板错误行，不显示
// 「已复制」。
func (a *App) copyTaskCommand(c *gui.Context, item task) {
	c.WriteClipboardThen(item.command, func(err error) {
		if err != nil {
			a.panels.taskCopyError = "复制失败：" + err.Error()
			return
		}
		a.panels.taskCopyError = ""
		a.panels.taskCopiedID = item.id
		a.panels.taskCopiedAt = time.Now()
		c.Announce("已复制")
		time.AfterFunc(taskCopiedFeedback, func() {
			if a.panels.taskCopiedID != item.id {
				return
			}
			a.panels.taskCopiedID = ""
			if a.changed != nil {
				a.changed()
			}
		})
	})
}

// taskOutput 是只读输出框（K-30 / K-31）：深底、最小高 160、可滚动；
// 输出达到服务端上限时在右下角给「预览已截断」（K-32），读取失败时换成
// 失败提示行与「重新连接」（K-33 / K-34）。
func (a *App) taskOutput(c *gui.Context, item task) {
	th := c.Theme()
	if item.logFailed {
		a.taskOutputFailed(c, item)
		return
	}
	box := gui.Column(c).Shrink(0).MinHeight(panelOutputMinHeight).Grow(1).
		Padding(space2, panelPad).Radius(radius).Background(termCanvasBg).
		// K-30 只读输出框
		Label("后台输出")
	box.Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			// K-31 没有输出时的说明行
			typeCode.style(th, gui.Text(c, termEmptyText)).TextColor(termCanvasMuted).
				Selectable().MaxLines(1)
		})
		if item.logTruncated {
			// K-32 预览已截断：右下角的覆盖标签（半透明黑底、白字）。
			gui.Row(c).Shrink(0).Justify(gui.End).Children(func() {
				gui.Row(c).Shrink(0).Padding(0, space1).Radius(radiusSmall).
					Background(gui.RGBA(0, 0, 0, 0.55)).Label("预览已截断").
					Children(func() {
						typeMicro.style(th, gui.Text(c, "预览已截断")).
							TextColor(gui.Hex("#ffffff")).SingleLine()
					})
			})
		}
	})
}

// taskOutputFailed 是输出读取失败（K-33）与「重新连接」（K-34）：危险色的
// 提示行，行尾是重连入口；重连只重读一次输出，不清掉任务本身。
func (a *App) taskOutputFailed(c *gui.Context, item task) {
	th := c.Theme()
	gui.Row(c).Shrink(0).MinHeight(panelOutputMinHeight).
		Padding(space2, panelPad).Gap(space2).Radius(radius).
		Background(th.Danger.Alpha(0.10)).
		Role(gui.RoleAlert).AlignItems(gui.Center).Children(func() {
		iconGlyph(c, iconWarn, th.Danger, "终端操作失败")
		typeBody.style(th, gui.Text(c, "终端操作失败")).TextColor(th.Danger).
			Grow(1).MaxLines(2)
		if actionButton(c, "重新连接", "重新连接输出", false, false).Clicked() {
			a.reconnectTaskOutput(item)
		}
	})
}

// reconnectTaskOutput 重读任务的输出（K-34）：本版没有输出读取通道，重连只
// 清掉失败标记并让视图再画一帧；连着 Source 时同样从空输出重画，不装作已经
// 读到内容。
func (a *App) reconnectTaskOutput(item task) {
	for i := range a.tasks {
		if a.tasks[i].id == item.id {
			a.tasks[i].logFailed = false
		}
	}
	a.panels.taskReload++
}

// wakeupDetail 是唤醒详情（K-35 / K-36）：触发时刻 + 提示正文。
func (a *App) wakeupDetail(c *gui.Context, item task) {
	th := c.Theme()
	gui.Scroll(c).Grow(1).Children(func() {
		gui.Column(c).Padding(panelPad).Gap(panelGap).Children(func() {
			at := item.wakeAt
			if at == "" {
				at = "时刻未知"
			}
			// K-35 {time} 触发
			typeSecondaryStrong.style(th, gui.Text(c, at+" 触发")).TextColor(th.Text).
				SingleLine().Label(at + " 触发")
			// K-36 唤醒提示内容（可换行）
			if item.wakeBody != "" {
				typeBody.style(th, gui.Text(c, item.wakeBody)).TextColor(th.Text).
					Selectable().MaxLines(12)
			}
		})
	})
}
