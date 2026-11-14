package ui

import (
	"context"

	gui "github.com/egoist/mygo/ui"
)

// 设置第 5、6 节（U1-ia §4）：
//   5 定时任务（S49–S54）：任务、计划、下次时间、历史
//   6 运行时与模态（T13–T17）：组件、管理进度、模态能力与任务
//
// 两节都沿用设置屏的骨架：分节标题 + 列表行；状态用样式（点），不写状态词。

// cronTask 是一条定时任务：名称与计划是内容；运行态用开关；历史给"上次结果"。
type cronTask struct {
	name     string
	schedule string
	next     string
	on       bool
	result   string
	status   extStatus
}

// runtimePart 是一个运行时组件：名称、版本、状态。
type runtimePart struct {
	name    string
	version string
	status  extStatus
}

// settingsCron 是「定时任务」分节。
func (a *App) settingsCron(c *gui.Context) {
	t := c.Theme()
	settingsSectionTitle(c, "定时任务")
	if snapshot, ok := a.liveSchedules(); ok {
		a.settingsCronLive(c, snapshot)
		return
	}

	if a.cronState == extLoading {
		a.stateNote(c, "定时任务加载中", iconClock, "clock", "正在读取…", t.TextMuted)
		return
	}
	if a.cronState == extFailed {
		a.failedNote(c, "定时任务读取失败", "读取失败：连接已断开", func() { a.cronState = extLoaded })
		return
	}
	if a.cronState == extEmpty || len(a.cron) == 0 {
		a.emptyNote(c, "定时任务空态", "还没有定时任务")
		return
	}
	for i := range a.cron {
		a.cronRow(c, i)
	}
}

// cronRow 是一行定时任务：名称 + 计划 + 下次 + 状态点 + 开关 + 删除入口。
// 删除是不可逆动作，走行内两步确认（F40）；启停可逆，用框架 Switch。
func (a *App) cronRow(c *gui.Context, i int) {
	// 删除会在同一帧里改 a.cron：索引越界就跳过（下一帧按新长度重建）。
	if i >= len(a.cron) {
		return
	}
	t := c.Theme()
	it := a.cron[i]
	if a.cronDeleting == i {
		gui.Column(c).Shrink(0).Padding(space1, 0).Gap(space1).
			Label("删除定时任务确认").Children(func() {
			gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
				iconGlyph(c, iconXMark, t.Danger, "x-mark")
				typeSecondary.style(t, gui.Text(c, "确认删除「"+it.name+"」？")).
					TextColor(t.Danger).SingleLine()
			})
			gui.Row(c).Gap(space2).Children(func() {
				gui.Spacer(c)
				if actionButton(c, "删除", "确认删除任务", true, false).Clicked() {
					a.cron = append(a.cron[:i], a.cron[i+1:]...)
					a.cronDeleting = -1
				}
				if actionButton(c, "取消", "取消删除任务", false, false).Clicked() {
					a.cronDeleting = -1
				}
			})
		})
		return
	}
	label := "定时任务：" + it.name + " · " + it.schedule + " · " + statusName(it.status)
	width, _ := c.Size()
	inner := width - SettingsNavWidth - 2*space8
	// 窄窗（内宽 < 500）两行：任务名一行，计划 + 下次 + 开关 + 删除一行（F38 的退化）。
	if inner < cronRowMinWidth {
		gui.Column(c).Shrink(0).Gap(space1).Label(label).Children(func() {
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
				statusDotMark(c, statusColor(t, it.status), "cron:"+it.name)
				typeBody.style(t, gui.Text(c, it.name)).TextColor(t.Text).SingleLine()
			})
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
				typeCode.style(t, gui.Text(c, it.schedule)).TextColor(t.TextMuted)
				typeSecondary.style(t, gui.Text(c, it.next)).TextColor(t.TextMuted).SingleLine()
				gui.Spacer(c)
				gui.Switch(c, &it.on).Label("启用任务：" + it.name)
				if iconButton(c, iconButtonSpec{icon: iconXMark, label: "删除任务：" + it.name, size: rowCompact}) {
					a.cronDeleting = i
				}
				a.cron[i] = it
			})
		})
		return
	}
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label(label).Children(func() {
		statusDotMark(c, statusColor(t, it.status), "cron:"+it.name)
		typeBody.style(t, gui.Text(c, it.name)).TextColor(t.Text).Width(extNameWidth).SingleLine()
		typeCode.style(t, gui.Text(c, it.schedule)).TextColor(t.TextMuted).Width(cronScheduleWidth)
		typeSecondary.style(t, gui.Text(c, it.next)).TextColor(t.TextMuted).Grow(1).SingleLine()
		gui.Switch(c, &it.on).Label("启用任务：" + it.name)
		if iconButton(c, iconButtonSpec{icon: iconXMark, label: "删除任务：" + it.name, size: rowCompact}) {
			a.cronDeleting = i
		}
		a.cron[i] = it
	})
}

// settingsCronLive 是接上事实后的定时任务：启停与删除没有对接写入口，本批只读。
func (a *App) settingsCronLive(c *gui.Context, snapshot SchedulesSnapshot) {
	t := c.Theme()
	switch {
	case snapshot.CapabilityMissing:
		a.stateNote(c, "定时任务", iconClock, "clock", "服务端未声明定时任务能力", t.TextMuted)
		return
	case snapshot.ReadError != "":
		a.stateNote(c, "定时任务读取失败", iconWarn, "exclamation", snapshot.ReadError, t.Danger)
		return
	case len(snapshot.Items) == 0:
		a.emptyNote(c, "定时任务空态", "还没有定时任务")
		return
	}
	for _, item := range snapshot.Items {
		a.cronLiveRow(c, item)
	}
}

// cronLiveRow 是一行定时任务：启用态用开关，运行结果用服务端给的原文。
func (a *App) cronLiveRow(c *gui.Context, item ScheduleSnapshot) {
	t := c.Theme()
	if a.cronDeletingID == item.ID {
		gui.Column(c).Shrink(0).Padding(space1, 0).Gap(space1).
			Label("删除定时任务确认").Children(func() {
			gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
				iconGlyph(c, iconXMark, t.Danger, "x-mark")
				typeSecondary.style(t, gui.Text(c, "确认删除「"+item.Name+"」？")).
					TextColor(t.Danger).SingleLine()
			})
			gui.Row(c).Gap(space2).Children(func() {
				gui.Spacer(c)
				if actionButton(c, "删除", "确认删除任务", true, false).Clicked() && a.intents != nil {
					id := item.ID
					a.dispatch("删除定时任务", func(ctx context.Context) error {
						return a.intents.DeleteSchedule(ctx, id)
					}, func() { a.cronDeletingID = "" })
				}
				if actionButton(c, "取消", "取消删除任务", false, false).Clicked() {
					a.cronDeletingID = ""
				}
			})
		})
		return
	}
	next := ""
	if !item.NextRunAt.IsZero() {
		next = item.NextRunAt.Local().Format("01-02 15:04")
	}
	last := item.LastStatus
	if last == "" {
		last = "—"
	}
	label := "定时任务：" + item.Name + " · " + item.Schedule
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label(label).Children(func() {
		statusDotMark(c, statusColor(t, extOK), "cron:"+item.Name)
		typeBody.style(t, gui.Text(c, item.Name)).TextColor(t.Text).Width(extNameWidth).SingleLine()
		typeCode.style(t, gui.Text(c, item.Schedule)).TextColor(t.TextMuted).Width(cronScheduleWidth)
		typeSecondary.style(t, gui.Text(c, next)).TextColor(t.TextMuted).Grow(1).SingleLine()
		typeSecondary.style(t, gui.Text(c, last)).TextColor(t.TextMuted).SingleLine()
		before := item.Enabled
		on := item.Enabled
		gui.Switch(c, &on).Label("启用任务：" + item.Name)
		if a.intents != nil && on != before {
			id, enabled := item.ID, on
			a.dispatch("启停定时任务", func(ctx context.Context) error {
				return a.intents.SetScheduleEnabled(ctx, id, enabled)
			}, nil)
		}
		if iconButton(c, iconButtonSpec{icon: iconXMark, label: "删除任务：" + item.Name, size: rowCompact}) {
			a.cronDeletingID = item.ID
		}
	})
}

// runtimeStatus 把服务端状态换成列表点的状态。
func runtimeStatus(status string) extStatus {
	switch status {
	case "failed", "down":
		return extError
	case "acquiring", "starting":
		return extUpdate
	}
	return extOK
}

// settingsRuntime 是「运行时与模态」分节：组件清单 + 模态能力 + 提交入口。
func (a *App) settingsRuntime(c *gui.Context) {
	t := c.Theme()
	settingsSectionTitle(c, "运行时与模态")
	if st, ok := a.liveSettings(); ok {
		a.settingsRuntimeLive(c, st)
		return
	}

	if a.runtimeState == extFailed {
		a.failedNote(c, "运行时读取失败", "读取失败：连接已断开", func() { a.runtimeState = extLoaded })
		return
	}
	for _, p := range a.runtime {
		p := p
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).
			Label("运行时组件：" + p.name + " · " + p.version + " · " + statusName(p.status)).
			Children(func() {
				statusDotMark(c, statusColor(t, p.status), "part:"+p.name)
				typeBody.style(t, gui.Text(c, p.name)).TextColor(t.Text).Width(extNameWidth).SingleLine()
				typeSecondary.style(t, gui.Text(c, p.version)).TextColor(t.TextMuted).
					Width(extVersionWidth)
				gui.Spacer(c)
			})
	}

	// 模态能力：一行说明（T16/T17 的样例），下面一行提交入口。
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label("模态能力").Children(func() {
		typeSecondary.style(t, gui.Text(c, "可提交长时间任务（样例：一次最多 1 个）")).
			TextColor(t.TextMuted)
	})
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Children(func() {
		gui.TextInputBase(c, &a.modalTask).Label("模态任务").Grow(1).
			Padding(space1, space2).Radius(radius).Background(t.Surface)
		if actionButton(c, "提交任务", "提交模态任务", true, a.modalTask == "").Clicked() {
			a.modalSubmitted = a.modalTask
			a.modalTask = ""
		}
	})
	if a.modalSubmitted != "" {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Shrink(0).Label("提交回执").Children(func() {
			iconGlyph(c, iconCheck, t.Success, "check")
			typeSecondary.style(t, gui.Text(c, "已提交：「"+a.modalSubmitted+"」")).
				TextColor(t.TextMuted).SingleLine()
		})
	}
}

// settingsRuntimeLive 是接上事实后的运行时分节：组件与模态能力只读。
func (a *App) settingsRuntimeLive(c *gui.Context, st SettingsSnapshot) {
	t := c.Theme()
	if !st.RuntimesKnown {
		reason := "尚未读取"
		if st.RuntimesReason != "" {
			reason = st.RuntimesReason
		}
		a.stateNote(c, "运行时组件", iconClock, "clock", reason, t.TextMuted)
	} else if len(st.Runtimes) == 0 {
		a.emptyNote(c, "运行时空态", "还没有运行时组件")
	} else {
		for _, part := range st.Runtimes {
			part := part
			status := runtimeStatus(part.Status)
			label := "运行时组件：" + part.Name + " · " + part.Version + " · " + statusName(status)
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label(label).Children(func() {
				statusDotMark(c, statusColor(t, status), "part:"+part.Name)
				typeBody.style(t, gui.Text(c, part.Name)).TextColor(t.Text).Width(extNameWidth).SingleLine()
				typeSecondary.style(t, gui.Text(c, part.Version)).TextColor(t.TextMuted).
					Width(extVersionWidth)
				gui.Spacer(c)
			})
		}
	}
	// 模态能力：只显示服务端声明的支持情况，不提供提交入口（写路径未接）。
	modality := st.Modality
	switch {
	case !modality.Supported:
		reason := modality.Reason
		if reason == "" {
			reason = "服务端未声明模态能力"
		}
		a.stateNote(c, "模态能力", iconWarn, "exclamation", reason, t.TextMuted)
	case modality.TaskKnown && modality.TaskSupported:
		a.stateNote(c, "模态能力", iconCheck, "check", "可提交长时间任务", t.TextMuted)
	case modality.TaskKnown:
		a.stateNote(c, "模态能力", iconWarn, "exclamation", "不可提交长时间任务", t.TextMuted)
	}
	// 提交入口保留形状但禁用：服务端未注册 runtime.* 与模态任务提交方法（D14 §5）。
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label("模态任务提交").Children(func() {
		actionButton(c, "提交任务", "提交模态任务", true, true)
		typeSecondary.style(t, gui.Text(c, "blocked：服务端未注册 runtime.* / 模态任务提交方法")).
			TextColor(t.TextMuted)
		gui.Spacer(c)
	})
}

// stateNote / emptyNote / failedNote 是三种非正常态的公共行（加载 / 空 / 失败），
// 与扩展逐字一致，避免三节各写一套。
func (a *App) stateNote(c *gui.Context, label string, icon *gui.SVG, mark, text string, color gui.Color) {
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label(label).Children(func() {
		iconGlyph(c, icon, color, mark)
		typeSecondary.style(c.Theme(), gui.Text(c, text)).TextColor(color)
	})
}

func (a *App) emptyNote(c *gui.Context, label, text string) {
	a.stateNote(c, label, iconChat, "chat", text, c.Theme().TextMuted)
}

func (a *App) failedNote(c *gui.Context, label, text string, retry func()) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label(label).Children(func() {
		iconGlyph(c, iconWarn, t.Danger, "exclamation")
		typeSecondary.style(t, gui.Text(c, text)).TextColor(t.Danger)
		if iconButton(c, iconButtonSpec{icon: iconUndo, label: "重试读取", size: rowCompact}) {
			retry()
		}
		gui.Spacer(c)
	})
}
