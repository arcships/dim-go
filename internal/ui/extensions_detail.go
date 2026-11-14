package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 扩展的二级页：详情（F47）、批量选择（F48）、MCP 的认证 / 导入 / 连接测试（F49–F51）。
//
// 三块都在设置第 4 节的同一块内容区里完成，不新开页面也不借右栏：
//   - 详情：屏内二级页（列表整块换成详情，顶部一个返回入口）；
//   - 选择：列表进入选择态（每行复选框 + 工具条）；
//   - MCP：行内加认证状态与两个入口，连接测试三态（进行中 / 成功 / 失败）。

// extDetailView 是扩展详情（屏内二级页）：描述、版本、来源、能力、启停、卸载。
func (a *App) extDetailView(c *gui.Context) {
	t := c.Theme()
	item := a.currentExt()
	if item == nil {
		a.extDetail = -1
		return
	}
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Children(func() {
		if iconButton(c, iconButtonSpec{icon: iconChevronL, label: "返回扩展列表", size: rowCompact}) {
			a.extDetail = -1
		}
		typeH3.style(t, gui.Text(c, item.name)).TextColor(t.Text)
		typeSecondary.style(t, gui.Text(c, item.version)).TextColor(t.TextMuted)
	})
	typeSecondary.style(t, gui.Text(c, extDescription(item.name))).TextColor(t.TextMuted).
		Label("扩展描述")

	for _, row := range [][2]string{{"来源", "本地安装"}, {"能力", "读写工作区文件、发起命令"}} {
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).
			Label("扩展详情：" + row[0]).Children(func() {
			typeControl.style(t, gui.Text(c, row[0])).Width(SettingsLabelWidth).
				TextColor(t.TextMuted)
			typeSecondary.style(t, gui.Text(c, row[1])).TextColor(t.Text)
			gui.Spacer(c)
		})
	}

	// 启停（可逆，Switch）与卸载（不可逆，两步确认）。
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Children(func() {
		typeControl.style(t, gui.Text(c, "启用")).Width(SettingsLabelWidth).TextColor(t.TextMuted)
		gui.Switch(c, &item.on).Label("启用：" + item.name)
		gui.Spacer(c)
	})
	if a.extUninstallConfirm {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Shrink(0).
			Label("卸载扩展确认").Children(func() {
			iconGlyph(c, iconXMark, t.Danger, "x-mark")
			typeSecondary.style(t, gui.Text(c, "确认卸载「"+item.name+"」？")).TextColor(t.Danger)
			gui.Spacer(c)
		})
		gui.Row(c).Gap(space2).Children(func() {
			if actionButton(c, "卸载", "确认卸载", true, false).Clicked() {
				a.removeExt(a.extDetail)
				a.extDetail = -1
				a.extUninstallConfirm = false
			}
			if actionButton(c, "取消", "取消卸载", false, false).Clicked() {
				a.extUninstallConfirm = false
			}
			gui.Spacer(c)
		})
		return
	}
	gui.Row(c).Gap(space2).Children(func() {
		if actionButton(c, "卸载", "卸载扩展", false, false).Clicked() {
			a.extUninstallConfirm = true
		}
		gui.Spacer(c)
	})
}

// selectToolbar 是选择态的工具条：已选 N + 启用 / 停用 / 取消。
func (a *App) selectToolbar(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Padding(0, 0, space1, 0).
		Label("批量工具条").Children(func() {
		typeSecondary.style(t, gui.Text(c, "已选 "+itoa(len(a.extSelected)))).
			TextColor(t.TextMuted)
		gui.Spacer(c)
		if actionButton(c, "启用", "批量启用", false, len(a.extSelected) == 0).Clicked() {
			a.setSelected(true)
		}
		if actionButton(c, "停用", "批量停用", false, len(a.extSelected) == 0).Clicked() {
			a.setSelected(false)
		}
		if actionButton(c, "取消", "退出选择态", false, false).Clicked() {
			a.selectMode = false
			a.extSelected = map[string]bool{}
		}
	})
}

// mcpRow 是 MCP 页签下的一行：名称 + 版本 + 认证状态 + 导入 / 连接测试入口 + 开关。
func (a *App) mcpRow(c *gui.Context, it *extItem) {
	t := c.Theme()
	auth := "未认证"
	authColor := t.TextMuted
	if it.on {
		auth, authColor = "已认证", t.Success
	}
	status := a.mcpTest[it.name] // mcpIdle / mcpTesting / mcpOK / mcpFail
	label := "MCP：" + it.name + " · " + it.version + " · " + auth
	gui.Column(c).Shrink(0).Gap(space1).Label(label).Children(func() {
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
			statusDotMark(c, authColor, "auth:"+it.name)
			typeBody.style(t, gui.Text(c, it.name)).TextColor(t.Text).Width(extNameWidth).SingleLine()
			typeSecondary.style(t, gui.Text(c, it.version)).TextColor(t.TextMuted).
				Width(extVersionWidth)
			gui.Spacer(c)
			if actionButton(c, "导入", "导入配置："+it.name, false, false).Clicked() {
				a.importMCP = it.name
			}
			if actionButton(c, "测试", "连接测试："+it.name, false, status == mcpTesting).Clicked() {
				a.mcpTest[it.name] = mcpTesting
				a.mcpTestDone[it.name] = true
			}
			gui.Switch(c, &it.on).Label("启用：" + it.name)
		})
		// 连接测试的三态：进行中 / 成功 / 失败，都在这行下面一行里。
		switch status {
		case mcpTesting:
			a.stateNote(c, "连接测试进行中", iconClock, "clock", "正在连接…", t.TextMuted)
		case mcpOK:
			a.stateNote(c, "连接测试成功", iconCheck, "check", "连接正常", t.Success)
		case mcpFail:
			a.stateNote(c, "连接测试失败", iconWarn, "exclamation", "连接失败：超时", t.Danger)
		}
		if a.importMCP == it.name {
			a.stateNote(c, "导入回执", iconCheck, "check", "已导入配置："+it.name, t.TextMuted)
		}
	})
}

// extDescription 是样例描述（内容）。
func extDescription(name string) string {
	return name + "：在本会话里提供一组工具，可按需启用或停用。"
}

// itoa 是小整数转字符串（避免为几处引入 strconv 的重复写法）。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// mcpState 是 MCP 连接测试的四态。
type mcpState int

const (
	mcpIdle mcpState = iota
	mcpTesting
	mcpOK
	mcpFail
)
