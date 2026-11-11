package ui

import gui "github.com/egoist/mygo/ui"

// header 是 R2 标题行：会话名 + 用量 + 右栏入口。
// 分支、权限、模型取默认值时不出现（UI-9）；偏离默认的模型只在行末留一个
// 淡标记，不占输入区。空态只留会话名。
func (a *App) header(c *gui.Context) {
	t := c.Theme()
	name := "修发送路径"
	if a.state == StateEmpty {
		name = "新会话"
	}
	gui.Row(c).Height(HeaderHeight).Shrink(0).Padding(0, space4).
		Gap(space2).AlignItems(gui.Center).
		BorderWidth(0, 0, strokeHairline, 0).BorderColor(t.Border).
		Label("标题行").Role(gui.RoleGroup).Children(func() {
		title := typeTitleRow.style(t, gui.Text(c, name)).SingleLine()
		if a.state == StateEmpty {
			title.TextColor(t.TextMuted)
		}
		if a.state == StateEmpty {
			gui.Spacer(c)
			return
		}
		gui.Spacer(c)
		typeSecondary.style(t, gui.Text(c, "上下文 12%")).TextColor(t.TextMuted).
			Label("上下文用量")
		if a.model != "" {
			// 偏离默认的模型：最小形式，不写“模型：”前缀（UI-9）。
			typeSecondary.style(t, gui.Text(c, a.model)).TextColor(t.TextMuted).
				Label("模型 " + a.model)
		}
		label, hint := "右栏 ›", "打开右栏"
		if a.rightOpen {
			label, hint = "右栏 › 收起", "收起右栏"
		}
		if actionButton(c, label, hint, false, false).Clicked() {
			a.rightOpen = !a.rightOpen
		}
	})
}
