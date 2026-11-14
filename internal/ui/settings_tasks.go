package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 设置第 5、6 节（U1-ia §4）：
//   5 定时任务（S49–S54）：任务、计划、下次时间、历史
//   6 运行时与模态（T13–T17）：组件、管理进度、模态能力与任务
//
// 两节都沿用设置屏的骨架：分节标题 + 列表行；状态用样式（点），不写状态词。

// cronTask 是一条定时任务：名称与计划是内容；运行态用开关；历史给"上次结果"。
type cronTask struct {
	name    string
	schedule string
	next    string
	on      bool
	result  string
	status  extStatus
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

// settingsRuntime 是「运行时与模态」分节：组件清单 + 模态能力 + 提交入口。
func (a *App) settingsRuntime(c *gui.Context) {
	t := c.Theme()
	settingsSectionTitle(c, "运行时与模态")

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
