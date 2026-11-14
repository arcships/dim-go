package ui

import gui "github.com/egoist/mygo/ui"

// rightPanel 是 R5 右栏：默认收起；展开后是带标签的容器，只放当前会话
// 的附属内容（文件、改动、后台、目标）。宽度由调用方按 280–（容器−400）
// 算好传入，主区不缩到 400 以下。
func (a *App) rightPanel(c *gui.Context, width float32) {
	t := c.Theme()
	labels := []string{"文件", "改动", "后台", "目标", "终端"}
	// 与主区之间靠面差（surface 对 background）分开，不画竖线（UI-18 第一档）。
	gui.Column(c).Width(width).Shrink(0).
		Background(t.Surface).
		Label("右栏").Role(gui.RoleGroup).Children(func() {
		gui.Row(c).Height(HeaderHeight).Shrink(0).Padding(0, space3).
			AlignItems(gui.Center).Children(func() {
			gui.Tabs(c, &a.rightTab, labels...).Label("右栏标签").Grow(1)
		})
		// 块之间的分隔用框架 Divider：方向随容器，颜色取 border（U2 校准 C8）。
		gui.Divider(c)
		if a.rightTab == 4 {
			// 终端自带滚动（输出区）与提示行，不再套外层滚动容器。
			gui.Column(c).Grow(1).Padding(space3).Gap(space2).Children(func() {
				a.terminalPane(c)
				a.termPrompt(c)
			})
			return
		}
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Padding(space3).Gap(space2).Children(func() {
				switch a.rightTab {
				case 0:
					fileRow(c, "internal/ui/composer.go")
					fileRow(c, "internal/ui/timeline.go")
					fileRow(c, "internal/ui/app.go")
				case 1:
					typeSecondary.style(t, gui.Text(c, "2 个文件 · +41 −7")).TextColor(t.TextMuted)
					fileRow(c, "internal/ui/composer.go  +33 −5")
					fileRow(c, "internal/ui/timeline.go  +8 −2")
				case 2:
					tasks, live := a.sessionTasks()
					if len(tasks) == 0 {
						typeSecondary.style(t, gui.Text(c, "没有后台任务")).TextColor(t.TextMuted)
						break
					}
					for i := range tasks {
						a.taskRow(c, tasks[i], live)
					}
					if live {
						typeSecondary.style(t, gui.Text(c, "取消任务：缺服务端方法")).TextColor(t.TextMuted)
					}
				case 3:
					typeBodyStrong.style(t, gui.Text(c, "把发送路径改成排队"))
					typeSecondary.style(t, gui.Text(c, "运行中的提交先入队，结束后自动发出。")).
						TextColor(t.TextMuted)
				}
			})
		})
	})
}

func fileRow(c *gui.Context, name string) {
	t := c.Theme()
	gui.Row(c).AlignItems(gui.Center).Children(func() {
		typeSecondary.style(t, gui.Text(c, name)).SingleLine()
	})
}

// taskRow 是一行后台任务：状态图标 + 任务名（内容）+ 取消入口。
// 状态用样式（运行中是播放圈 + accent-ink、完成是对勾、失败是叉），名字是内容；
// 取消是纯图标动作（可逆：任务可以再跑一次）。
func (a *App) taskRow(c *gui.Context, t task, live bool) {
	th := c.Theme()
	icon, color := iconTask, colorAccentInk.in(th)
	state := "运行中"
	switch t.state {
	case taskDone:
		icon, color, state = iconCheck, th.TextMuted, "已完成"
	case taskFailed:
		icon, color, state = iconXMark, th.Danger, "失败"
	}
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).
		Label("后台任务：" + t.name + " · " + state).Children(func() {
		iconGlyph(c, icon, color, "task")
		typeSecondary.style(th, gui.Text(c, t.name)).Grow(1).SingleLine()
		if t.state == taskRunning && !live {
			iconButton(c, iconButtonSpec{icon: iconXMark, label: "取消任务", size: rowCompact})
		}
	})
}
