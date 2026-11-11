package ui

import gui "github.com/egoist/mygo/ui"

// rightPanel 是 R5 右栏：默认收起；展开后是带标签的容器，只放当前会话
// 的附属内容（文件、改动、后台、目标）。宽度由调用方按 280–（容器−400）
// 算好传入，主区不缩到 400 以下。
func (a *App) rightPanel(c *gui.Context, width float32) {
	t := c.Theme()
	labels := []string{"文件", "改动", "后台", "目标"}
	gui.Column(c).Width(width).Shrink(0).
		Background(t.Surface).
		BorderWidth(0, strokeHairline, 0, 0).BorderColor(t.Border).
		Label("右栏").Role(gui.RoleGroup).Children(func() {
		gui.Row(c).Height(HeaderHeight).Shrink(0).Padding(0, space3).
			AlignItems(gui.Center).Children(func() {
			gui.Tabs(c, &a.rightTab, labels...).Label("右栏标签").Grow(1)
		})
		// 块之间的分隔线取框架 border，不承担控件轮廓（U2 §3.5）。
		gui.Box(c).Height(strokeHairline).Background(t.Border)
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
					typeSecondary.style(t, gui.Text(c, "没有后台任务")).TextColor(t.TextMuted)
				default:
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
