package ui

import gui "github.com/egoist/mygo/ui"

// header 是 R2 标题行：会话名 + 用量 + 右栏入口。
// 分支、权限、模型取默认值时不出现（UI-9），空态只留会话名。
func (a *App) header(c *gui.Context) {
	t := c.Theme()
	name := "修发送路径"
	if a.state == StateEmpty {
		name = "新会话"
	}
	gui.Row(c).Height(HeaderHeight).Shrink(0).Padding(0, spaceXL).
		Gap(spaceS).AlignItems(gui.Center).
		BorderWidth(0, 0, hairline, 0).BorderColor(t.Border).
		Label("标题行").Role(gui.RoleGroup).Children(func() {
		text := gui.Text(c, name).Bold().SingleLine()
		if a.state == StateEmpty {
			text.TextColor(t.TextMuted)
		}
		if a.state == StateEmpty {
			gui.Spacer(c)
			return
		}
		gui.Spacer(c)
		gui.Text(c, "上下文 12%").TextColor(t.TextMuted).
			FontSize(t.FontSize - smallFontSteps).Label("上下文用量")
		label, hint := "右栏 ›", "打开右栏"
		if a.rightOpen {
			label, hint = "右栏 › 收起", "收起右栏"
		}
		if gui.Button(c, label).Label(hint).Clicked() {
			a.rightOpen = !a.rightOpen
		}
	})
}
