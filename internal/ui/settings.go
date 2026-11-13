package ui

import (
	"fmt"

	gui "github.com/egoist/mygo/ui"
)

// settings 是 P2 设置屏（独立成屏，U1-ia §4）：左导航 224 + 内容区（上限 1008）。
// 本片只做三个分节——外观、供应商、命令规则；其余分节在后续批次追加节点。
//
// 顶部让位与标题栏一致：窗口控件（红绿灯）落在导航列上方的那条里（UI-19），
// 导航列顶部留出 c.TitleBar().Height，空白区可拖动窗口。
func (a *App) settings(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Fill().AlignItems(gui.Stretch).
		Background(t.Background).Label("设置").Children(func() {
		a.settingsNav(c)
		gui.Column(c).Grow(1).MinWidth(MainMinWidth).Children(func() {
			gui.Column(c).Grow(1).MaxWidth(SettingsContentMaxWidth).
				Padding(space8, space8).Gap(space4).Children(func() {
				switch a.settingsSection {
				case 0:
					a.settingsAppearance(c)
				case 1:
					a.settingsProviders(c)
				default:
					a.settingsRules(c)
				}
			})
		})
	})
}

// settingsNav 是左导航：返回主界面的图标入口 + 分节列表。
func (a *App) settingsNav(c *gui.Context) {
	t := c.Theme()
	sections := []string{"外观", "供应商", "命令规则"}
	gui.Column(c).Width(SettingsNavWidth).Shrink(0).
		Background(t.Surface).Children(func() {
		if inset := titleBarInset(c.TitleBar()); inset > 0 {
			gui.Row(c).Height(inset).Shrink(0).DragWindow().Label("窗口控件让位")
		}
		gui.Row(c).Height(sidebarHeaderHeight).Shrink(0).
			Padding(space1, space2).AlignItems(gui.Center).Gap(space1).Children(func() {
			if iconButton(c, iconButtonSpec{icon: iconChevronL, label: "返回主界面", size: rowCompact}) {
				a.screen = ScreenMain
			}
			gui.Spacer(c)
		})
		for i, name := range sections {
			a.navRow(c, i, name)
		}
		gui.Spacer(c)
	})
}

// navRow 是导航里的一节：自绘行（无外观 ButtonBase），选中态由底色与字重表达。
func (a *App) navRow(c *gui.Context, index int, name string) {
	t := c.Theme()
	row := gui.ButtonBase(c).Padding(space1, space3).Radius(radius).
		Label(name).Cursor(gui.CursorPointer).Justify(gui.Start)
	selected := a.settingsSection == index
	switch {
	case selected:
		row = row.Background(t.SurfacePressed)
	case row.Hovered():
		row = row.Background(t.SurfaceHover)
	}
	if row.Clicked() {
		a.settingsSection = index
	}
	row.Children(func() {
		style := typeBody
		if selected {
			style = typeBodyStrong
		}
		style.style(t, gui.Text(c, name)).TextColor(t.Text)
	})
}

// settingsRow 是一行设置项：左侧标签（界面语）+ 右侧控件。标签列宽固定，
// 让同一分节内的控件左缘对齐（UI-22 的"同类元素同一条左缘"）。
func settingsRow(c *gui.Context, label string, control func()) {
	t := c.Theme()
	gui.Row(c).Gap(space4).AlignItems(gui.Center).Shrink(0).Children(func() {
		typeBody.style(t, gui.Text(c, label)).Width(SettingsLabelWidth).TextColor(t.Text)
		gui.Row(c).Grow(1).AlignItems(gui.Center).Children(control)
	})
}

// settingsSectionTitle 是分节标题：标题层级由字重与字号给（U2 §1）。
func settingsSectionTitle(c *gui.Context, title string) {
	t := c.Theme()
	typeH3.style(t, gui.Text(c, title)).TextColor(t.Text)
}

// settingsAppearance 是「外观」：主题与动效两项。
func (a *App) settingsAppearance(c *gui.Context) {
	settingsSectionTitle(c, "外观")
	settingsRow(c, "主题", func() {
		gui.Select(c, &a.themePref, []string{"跟随系统", "明", "暗"}).Label("主题选择")
	})
	settingsRow(c, "动效", func() {
		gui.Switch(c, &a.motionOff).Label("关闭动效")
	})
}

// settingsProviders 是「供应商」：当前供应商与模型，下面是目录信息（内容）。
func (a *App) settingsProviders(c *gui.Context) {
	t := c.Theme()
	settingsSectionTitle(c, "供应商")
	settingsRow(c, "当前供应商", func() {
		gui.Select(c, &a.provider, a.providers).Label("当前供应商")
	})
	settingsRow(c, "模型", func() {
		gui.Select(c, &a.modelChoice, a.models).Label("模型")
	})
	// 目录单价是内容（G6）：只读一行，不在主界面出现。
	typeSecondary.style(t, gui.Text(c, a.modelPrice)).TextColor(t.TextMuted).
		Label("目录单价")
}

// settingsRules 是「命令规则」：每条规则一行（命令 + 决定 + 匹配方式），下面是
// 高后果动作「清空命令规则」（AC-13，两步确认）。
func (a *App) settingsRules(c *gui.Context) {
	t := c.Theme()
	settingsSectionTitle(c, "命令规则")
	typeSecondary.style(t, gui.Text(c, "规则对所有会话生效；拒绝先于允许。")).
		TextColor(t.TextMuted).Label("规则说明")

	// 窄窗退化（U2-forms F23）：三列排不下一行时，一条规则两行——先命令，
	// 再「决定 + 匹配方式」。表头只在三列并排时出现（堆叠后每列带自己的标签）。
	width, _ := c.Size()
	inner := width - SettingsNavWidth - 2*space8
	stacked := inner < SettingsRuleRowMinWidth

	if !stacked {
		gui.Row(c).Gap(space4).Shrink(0).Children(func() {
			typeSecondary.style(t, gui.Text(c, "命令")).Width(SettingsRuleCommandWidth).
				TextColor(t.TextMuted).Label("命令列")
			typeSecondary.style(t, gui.Text(c, "决定")).Width(SettingsRuleSelectWidth).
				TextColor(t.TextMuted).Label("决定列")
			typeSecondary.style(t, gui.Text(c, "匹配方式")).Width(SettingsRuleSelectWidth).
				TextColor(t.TextMuted).Label("匹配列")
		})
	}
	for i := range a.rules {
		r := &a.rules[i]
		if !stacked {
			gui.Row(c).Gap(space4).AlignItems(gui.Center).Shrink(0).Children(func() {
				typeCode.style(t, gui.RichText(c, gui.Span{
					Text: r.command, Size: typeCode.sizeIn(t), Weight: typeCode.weight, Color: t.Text,
				})).Width(SettingsRuleCommandWidth).SingleLine()
				gui.Select(c, &r.decision, []string{"允许", "拒绝"}).
					Width(SettingsRuleSelectWidth).Label("决定：" + r.command)
				gui.Select(c, &r.match, []string{"精确", "前缀"}).
					Width(SettingsRuleSelectWidth).Label("匹配方式：" + r.command)
			})
			continue
		}
		gui.Column(c).Shrink(0).Gap(space1).Children(func() {
			typeCode.style(t, gui.RichText(c, gui.Span{
				Text: r.command, Size: typeCode.sizeIn(t), Weight: typeCode.weight, Color: t.Text,
			})).SingleLine()
			gui.Row(c).Gap(space3).AlignItems(gui.Center).Shrink(0).Children(func() {
				typeSecondary.style(t, gui.Text(c, "决定")).TextColor(t.TextMuted)
				gui.Select(c, &r.decision, []string{"允许", "拒绝"}).
					Label("决定：" + r.command)
				typeSecondary.style(t, gui.Text(c, "匹配方式")).TextColor(t.TextMuted)
				gui.Select(c, &r.match, []string{"精确", "前缀"}).
					Label("匹配方式：" + r.command)
				gui.Spacer(c)
			})
		})
	}
	gui.Spacer(c)
	// 高后果动作：图标 + 文字，两步确认（U3-actions AC-13）。
	label, tip := "清空命令规则", "删除所有会话的全部命令规则"
	if a.clearRulesConfirm {
		label, tip = "确认清空", "再点一次：清空全部规则，无法撤销"
	}
	gui.Row(c).Shrink(0).Children(func() {
		if textIconButton(c, textIconButtonSpec{
			icon: iconXMark, mark: "x-mark", text: label, tip: tip,
		}) {
			if a.clearRulesConfirm {
				a.rules = nil
				a.clearRulesConfirm = false
			} else {
				a.clearRulesConfirm = true
			}
		}
	})
	if a.clearRulesConfirm {
		typeSecondary.style(t, gui.Text(c, "清空后所有会话立即生效，无法撤销。")).
			TextColor(t.Danger).Label("清空确认说明")
	}
}

// settingsSummary 是入口语（样例）：提供给导航与测试读取的当前值摘要。
func (a *App) settingsSummary() string {
	return fmt.Sprintf("%s · %s", a.themePref, a.provider)
}
