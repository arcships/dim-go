package ui

import (
	"fmt"

	gui "github.com/egoist/mygo/ui"
)

// 设置第 7、8 节（U1-ia §4）：
//   7 存储与诊断（T18、T19、T26）：存储读数、服务版本、连接方式、诊断日志
//   8 外观与键位（T20–T23、S35）：主题、动效、时区、快捷键提示表
//
// 读数带单位、只读；诊断导出是可逆动作；键位表按 U3 的 K 表渲染，只读。

// settingsStorage 是「存储与诊断」分节。
func (a *App) settingsStorage(c *gui.Context) {
	t := c.Theme()
	settingsSectionTitle(c, "存储与诊断")
	if st, ok := a.liveSettings(); ok {
		a.settingsStorageLive(c, st)
		return
	}

	// 存储读数：数值 + 单位，单位与数值同字号、次要色。
	readings := []struct{ label, value, unit string }{
		{"占用", "1.2", "GB"},
		{"会话数", "128", "个"},
		{"缓存", "84", "MB"},
	}
	for _, r := range readings {
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).
			Label("存储读数：" + r.label).Children(func() {
			typeControl.style(t, gui.Text(c, r.label)).Width(SettingsLabelWidth).
				TextColor(t.Text)
			typeBody.style(t, gui.Text(c, r.value)).TextColor(t.Text)
			typeSecondary.style(t, gui.Text(c, r.unit)).TextColor(t.TextMuted)
			gui.Spacer(c)
		})
	}

	// 服务版本与连接方式：一行事实。
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label("服务版本").Children(func() {
		typeControl.style(t, gui.Text(c, "服务版本")).Width(SettingsLabelWidth).TextColor(t.Text)
		typeCode.style(t, gui.Text(c, "dim-app-server v3.4.1")).TextColor(t.TextMuted)
		gui.Spacer(c)
	})
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label("连接方式").Children(func() {
		typeControl.style(t, gui.Text(c, "连接方式")).Width(SettingsLabelWidth).TextColor(t.Text)
		typeSecondary.style(t, gui.Text(c, "本机 socket")).TextColor(t.TextMuted)
		gui.Spacer(c)
	})

	// 诊断：导出（可逆、无确认，AC-28）与打开日志（AC-29）。
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Children(func() {
		if actionButton(c, "导出诊断", "导出诊断", false, false).Clicked() {
			a.diagnosticsExported = true
		}
		if iconButton(c, iconButtonSpec{icon: iconChevronR, label: "打开诊断日志", size: rowCompact}) {
			a.diagnosticsOpened = true
		}
		gui.Spacer(c)
	})
	if a.diagnosticsExported {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Shrink(0).Label("导出回执").Children(func() {
			iconGlyph(c, iconCheck, t.Success, "check")
			typeSecondary.style(t, gui.Text(c, "已导出诊断包")).TextColor(t.TextMuted)
		})
	}
}

// settingsStorageLive 是接上事实后的存储分节：只读服务端读数，缺项显示「—」。
func (a *App) settingsStorageLive(c *gui.Context, st SettingsSnapshot) {
	t := c.Theme()
	if !st.StorageKnown {
		reason := "尚未读取"
		if st.StorageError != "" {
			reason = st.StorageError
		}
		a.stateNote(c, "存储读数", iconClock, "clock", reason, t.TextMuted)
	} else {
		readings := []struct {
			label string
			bytes int64
		}{
			{"数据库", st.Storage.DatabaseBytes},
			{"WAL", st.Storage.WALBytes},
			{"可回收", st.Storage.ReclaimableBytes},
		}
		for _, reading := range readings {
			settingsValueRow(c, "存储读数："+reading.label, reading.label, humanBytes(reading.bytes))
		}
	}
	// 服务版本没有在设置领域暴露读数；不拿样例版本顶替。
	settingsValueRow(c, "服务版本", "服务版本", "—")
	settingsValueRow(c, "连接方式", "连接方式", "本机 socket")
	// 诊断导出/打开没有对接的本地能力，本批不提供入口（见 D14 未接清单）。
}

// humanBytes 把字节数换成带单位的读数；0 显示 0 B，不做四舍五入以外的推断。
func humanBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// settingsAppearanceKeys 是「外观与键位」分节：主题 / 动效 / 时区 + 快捷键提示表。
func (a *App) settingsAppearanceKeys(c *gui.Context) {
	t := c.Theme()
	settingsSectionTitle(c, "外观与键位")
	settingsRow(c, "主题", func() {
		gui.Select(c, &a.themePref, []string{"跟随系统", "明", "暗"}).Label("主题选择")
	})
	settingsRow(c, "动效", func() {
		gui.Switch(c, &a.motionOff).Label("关闭动效")
	})
	settingsRow(c, "时区", func() {
		gui.Select(c, &a.timeZone, []string{"跟随系统", "Asia/Shanghai", "UTC"}).Label("时区选择")
	})

	// 快捷键提示表：按 U3 的 K 表渲染，键位等宽、作用是文字；只读。
	typeSecondary.style(t, gui.Text(c, "快捷键")).TextColor(t.TextMuted)
	for _, k := range keyHints {
		gui.Row(c).Gap(space4).AlignItems(gui.Center).Shrink(0).
			Label("快捷键：" + k.keys + " · " + k.what).Children(func() {
			typeCode.style(t, gui.Text(c, k.keys)).TextColor(t.Text).Width(keyHintKeysWidth)
			typeSecondary.style(t, gui.Text(c, k.what)).TextColor(t.TextMuted).SingleLine()
			gui.Spacer(c)
		})
	}
	a.persistPreferences()
}

// keyHint 是一行快捷键提示（键位 + 作用），取自 U3 的 K 表。
type keyHint struct {
	keys string
	what string
}

// keyHints 是首片的键位提示（K 表里已在界面上生效的那些）。
var keyHints = [...]keyHint{
	{"Enter", "发送 / 停止"},
	{"Shift+Enter", "换行"},
	{"Cmd+Enter", "运行中排队发送"},
	{"Y / A / N", "允许一次 / 一直允许 / 拒绝"},
	{"Space", "展开或收起组头、切开关"},
	{"Esc", "关闭菜单、收起搜索、收回右栏"},
	{"Cmd+K", "清空终端"},
	{"Ctrl+C", "中止终端里正在跑的命令"},
	{"Cmd+1…5", "切换走查状态（仅骨架）"},
}
