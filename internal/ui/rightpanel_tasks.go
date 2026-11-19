package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 任务面板（右栏「后台」标签）：后台进程与唤醒两个子标签共用一套列表，
// 选中一行后下方给详情。
//
// 事实只有 task（标识、描述、状态）；命令行、pid、日志与唤醒时刻都由
// App Server 提供，Go 侧拿不到时不画那一段（不编造）。条目编号见
// docs/domains/requirements/requirements-panels.md §5（K-01–K-36）。

// taskStateText 是行与详情头共用的状态文本：屏幕上的字由数据的取值决定，
// 不写字面量状态（UI-23 / L6）。已停止态还缺事实（快照只有 stopping），
// 未实现，见交回清单。
func taskStateText(st taskState) string {
	switch st {
	case taskDone:
		return "已完成"
	case taskFailed:
		return "失败"
	}
	return "运行中"
}

// taskStateIconIn 按主题取状态图标：运行中用强调色、完成用次要色、失败用危险色。
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
	}
	return iconTask, accent
}

// tasksPanel 是任务面板：子标签（后台进程 / 唤醒）+ 列表 + 详情 + 空态。
func (a *App) tasksPanel(c *gui.Context) {
	gui.Column(c).Grow(1).Children(func() {
		gui.Row(c).Height(rightPanelTabHeight).Shrink(0).
			Padding(space1, panelPad).AlignItems(gui.Center).Children(func() {
			gui.Tabs(c, &a.panels.taskKind, "后台进程", "唤醒").Label("任务类型")
		})
		items, live := a.sessionTasks()
		if a.panels.taskKind == 1 {
			// 唤醒：Go 侧还没有唤醒事实（Source 的会话快照不含它），
			// 因此只呈现空态与原因（K-19、DEC-16）。
			panelEmpty(c, "暂无待唤醒事项（"+noteSchedule+"）")
			return
		}
		if len(items) == 0 {
			panelEmpty(c, "暂无运行进程")
			return
		}
		a.taskList(c, items, live)
		if a.panels.taskSel >= 0 && a.panels.taskSel < len(items) {
			gui.Divider(c)
			a.taskDetail(c, items[a.panels.taskSel])
		}
	})
}

// taskList 是列表容器（K-01）：可滚动、逐行渲染。
func (a *App) taskList(c *gui.Context, items []task, live bool) {
	gui.Scroll(c).Grow(1).Children(func() {
		gui.Column(c).Gap(0).Role(gui.RoleList).Label("后台进程").Children(func() {
			for i := range items {
				a.taskRow(c, items[i], i, live)
				if i < len(items)-1 {
					gui.Divider(c)
				}
			}
		})
	})
}

// taskRow 是一行后台任务：状态图标 + 任务名（内容）+ 行尾停止入口。
// 状态用样式（运行中是播放圈 + accent-ink、完成是对勾、失败是叉），名字是内容；
// 停止是纯图标动作（可逆：任务可以再跑一次），先点一次进待确认，再点才发。
func (a *App) taskRow(c *gui.Context, item task, index int, live bool) {
	th := c.Theme()
	icon, color := taskStateIconIn(item.state, th)
	state := taskStateText(item.state)
	selected := a.panels.taskSel == index
	row := gui.Row(c).Shrink(0).Height(panelRowHeight).
		Padding(space1, panelPad).Gap(space2).AlignItems(gui.Center)
	if selected {
		row = row.Background(th.SurfaceHover)
	}
	row.Children(func() {
		main := gui.ButtonBase(c).Grow(1).Height(rowRegular).Padding(space1, space1).
			Radius(radiusSmall).Gap(space2).AlignItems(gui.Center).
			Label("后台任务：" + item.name + " · " + state).Tooltip(item.name)
		if main.Clicked() {
			a.panels.taskSel = index
			a.panels.taskArmed = -1
		}
		main.Children(func() {
			iconGlyph(c, icon, color, "task")
			typeSecondaryStrong.style(th, gui.Text(c, item.name)).TextColor(th.Text).
				Grow(1).SingleLine()
		})
		if item.state == taskRunning {
			a.taskStopButton(c, item, index, live)
		}
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

// taskDetail 是选中行的详情（K-17）：状态头 + 命令行 + 只读输出框。
// 元信息（pid、结束时刻）与日志入口在 Go 侧没有事实，故不渲染。
func (a *App) taskDetail(c *gui.Context, item task) {
	th := c.Theme()
	icon, color := taskStateIconIn(item.state, th)
	state := taskStateText(item.state)
	gui.Scroll(c).Grow(1).Children(func() {
		gui.Column(c).Padding(panelPad).Gap(panelGap).Children(func() {
			gui.Row(c).Shrink(0).Gap(space2).AlignItems(gui.Center).Children(func() {
				iconGlyph(c, icon, color, "task")
				typeSecondaryStrong.style(th, gui.Text(c, state)).TextColor(th.Text).
					Grow(1).SingleLine()
				if item.name != "" {
					if iconButton(c, iconButtonSpec{
						icon: iconClipboard, label: "复制命令", tip: "复制命令",
						size: rowCompact,
					}) {
						c.WriteClipboard(item.name)
					}
				}
			})
			if command := panelCommandText(item.name); command != "" {
				typeCode.style(th, gui.Text(c, command)).TextColor(th.Text).
					Selectable().MaxLines(4)
			}
			a.taskOutput(c)
		})
	})
}

// taskOutput 是只读输出框（K-30 / K-31）：深底、最小高 160、可滚动。
// 输出内容来自后台进程的日志流；没有事实时只画框与空态说明。
func (a *App) taskOutput(c *gui.Context) {
	th := c.Theme()
	gui.Column(c).Shrink(0).MinHeight(panelOutputMinHeight).Grow(1).
		Padding(space2, panelPad).Radius(radius).Background(termCanvasBg).
		Label("后台输出").Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			typeCode.style(th, gui.Text(c, termEmptyText)).TextColor(termCanvasMuted).
				Selectable().MaxLines(1)
		})
	})
}
