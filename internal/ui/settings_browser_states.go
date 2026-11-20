package ui

// 本文件是「导入 Chrome Cookie」与「清理浏览器数据」两个对话框的各态渲染
// （G-H6-03 / 06 / 07 / 08 / 09a / 09b / 10、G-H7-03 / 04）。
//
// 这两条链要桌面宿主的浏览器数据通道：扫描 Chrome 配置、按配置读 Cookie 计数、
// 逐个导入并回执、清理内置浏览器数据并回执。本版没有这个通道，所以
//
//   - 默认态是「没有通道」：对话框照画标题、说明与动作，动作置灰并把缺什么写在
//     说明行里（见 settings_general_screen.go 的两个对话框）；
//   - 各态按状态结构逐支画全（扫描中带不确定进度条、无配置、读取中、计数块、
//     部分失败、成功、错误重试、清理成功 / 失败），状态是界面侧可丢的视图状态，
//     测试直接注入后逐态验收；不拿样例配置或假计数冒充通道结果。
//
// 元素锚点写在各自的渲染函数上，供需求对账按元素映射判定。

import (
	"fmt"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// browserImportPhase 是导入对话框的阶段。零值是「没有通道」，因此默认打开
// 对话框时不会出现任何编造的扫描结果。
type browserImportPhase int

const (
	browserImportUnavailable browserImportPhase = iota
	browserImportScanning
	browserImportNoProfiles
	browserImportReading
	browserImportReady
	browserImportImporting
	browserImportPartial
	browserImportDone
	browserImportFailed
)

// browserChromeProfile 是一条 Chrome 配置（G-H6-05）：名称与能力标签。
// capability 是给用户看的短词（如「可读」「需要解锁」），不是原始枚举。
type browserChromeProfile struct {
	name       string
	capability string
}

// browserCookieCounts 是所选配置的 Cookie 计数（G-H6-08）。
type browserCookieCounts struct {
	importable  int
	session     int
	expired     int
	partitioned int
}

// browserImportState 是导入对话框的可丢视图状态。
type browserImportState struct {
	phase browserImportPhase
	// progress 是扫描进度，取值 0–1；负数表示长度未知（画不确定进度条）。
	progress float64
	// profiles / selected 是扫描出的配置与当前选中项。
	profiles []browserChromeProfile
	selected int
	counts   browserCookieCounts
	// imported / failed 是导入回执的计数（G-H6-09a / 09b）。
	imported int
	failed   int
	// message 是错误块的主文案；diagnostic 是等宽诊断行（可为空）。
	message    string
	diagnostic string
	// retrying 表示重试已经在进行：这时「重试」同样禁用，原因换成执行态。
	retrying bool
}

// browserClearPhase 是清理对话框的阶段；零值同样是「没有通道」。
type browserClearPhase int

const (
	browserClearUnavailable browserClearPhase = iota
	browserClearClearing
	browserClearSuccess
	browserClearFailed
)

// browserClearState 是清理对话框的可丢视图状态。
type browserClearState struct {
	phase      browserClearPhase
	message    string
	diagnostic string
}

// 各态的固定文案：逐字取参考产品中文，错误文案在状态没给时兜底。
const (
	browserImportScanningText = "正在扫描 Chrome 配置…"
	browserImportEmptyText    = "未找到 Google Chrome 配置。"
	browserImportReadingText  = "正在读取所选配置…"
	browserImportPartialText  = "部分 Cookie 未能导入，部分网站可能需要重新登录。"
	browserImportFailText     = "无法读取 Chrome Cookie，请关闭 Chrome 后重试。"
	browserClearSuccessText   = "内置浏览器数据已清理。"
	browserClearFailText      = "浏览器数据未能完全清理，请重试。"
)

// iconLoading 是加载行的旋转缺口圆（lucide loader-circle 同形）。
var iconLoading = mustSVG(`<path d="M21 12a9 9 0 1 1-6.219-8.56"/>`)

// generalBrowserImportStates 画导入对话框的当前态。调用方已给出标题与说明；
// 这里只画阶段专属的内容块。
func generalBrowserImportStates(c *gui.Context, st *browserImportState) {
	switch st.phase {
	case browserImportScanning:
		generalBrowserScanning(c, st)
	case browserImportNoProfiles:
		generalBrowserEmpty(c)
	case browserImportReading:
		generalBrowserReading(c)
	case browserImportReady, browserImportImporting:
		generalBrowserProfiles(c, st)
		generalBrowserCounts(c, st.counts)
	case browserImportPartial, browserImportDone:
		generalBrowserProfiles(c, st)
		generalBrowserCounts(c, st.counts)
		generalBrowserResult(c, st)
	case browserImportFailed:
		generalBrowserProfiles(c, st)
		generalBrowserError(c, st)
	default:
		// 没有通道：写清缺什么，不画样例配置与假计数。
		generalNote(c, noteBrowserNoImport)
	}
}

// generalBrowserLoading 是扫描 / 读取共用的加载行：旋转图标 + 14px 灰字，
// 最小高 96（G-H6-03 / G-H6-07）。progress 为负时画不确定进度条。
func generalBrowserLoading(c *gui.Context, text, progressLabel string, progress float64) {
	t := c.Theme()
	gui.Column(c).MinHeight(generalLoadingMinHeight).Center().Gap(space2).
		Role(gui.RoleStatus).Label(text).Children(func() {
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
			generalBrowserSpinner(c, progressLabel)
			typeSecondary.style(t, gui.Text(c, text)).TextColor(t.TextMuted).Label(text)
		})
		value := progress
		if value <= 0 {
			// 没有百分比读数时画不确定进度条：它自己动，不写一个编出来的比例。
			value = -1
		} else if value > 1 {
			value = 1
		}
		gui.Progress(c, value).Width(generalLoadingBarWidth).Label(progressLabel)
	})
}

// generalLoadingMinHeight 是加载行的最小高（需求给 96）。
const generalLoadingMinHeight = 96

// generalLoadingBarWidth 是加载行里进度条的宽。
const generalLoadingBarWidth = 240

// generalBrowserSpinner 是加载行的旋转缺口圆：减少动效下静止，其余时间按
// 一秒一圈连续转（G-H6-03 / G-H6-07 的动画口径）。
func generalBrowserSpinner(c *gui.Context, mark string) {
	t := c.Theme()
	box := gui.Box(c).Width(screenIconInline).Height(screenIconInline).Shrink(0).
		Label("图标：" + mark).AccessHidden(true)
	if !c.Preferences().ReduceMotion {
		box = box.Rotate(box.Loop("browser-scan-spin", time.Second, gui.Linear) * 360)
	}
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		paintIcon(p, iconLoading, r, t.TextMuted)
	})
}

// generalBrowserScanning 画「正在扫描 Chrome 配置…」（G-H6-03；动画：旋转与
// 进度条在扫描期间连续，减少动效下静态）。
func generalBrowserScanning(c *gui.Context, st *browserImportState) {
	generalBrowserLoading(c, browserImportScanningText, "扫描 Chrome 配置", st.progress)
}

// generalBrowserEmpty 画无配置空态（G-H6-06）；此时不画分组标签（G-H6-04）。
func generalBrowserEmpty(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).MinHeight(generalLoadingMinHeight).Center().Children(func() {
		typeSecondary.style(t, gui.Text(c, browserImportEmptyText)).
			TextColor(t.TextMuted).Center().Label(browserImportEmptyText)
	})
}

// generalBrowserReading 画「正在读取所选配置…」（G-H6-07）；读取中不画计数块，
// 免得上一次的计数闪在屏上。
func generalBrowserReading(c *gui.Context) {
	generalBrowserLoading(c, browserImportReadingText, "读取所选 Chrome 配置", -1)
}

// generalBrowserProfiles 画配置分组与单选行（G-H6-04 / G-H6-05）。
func generalBrowserProfiles(c *gui.Context, st *browserImportState) {
	if len(st.profiles) == 0 {
		return
	}
	t := c.Theme()
	gui.Column(c).Gap(space1).Role(gui.RoleRadioGroup).Label("Chrome 配置").Children(func() {
		// G-H6-04 分组标签「Chrome 配置」
		typeMicro.style(t, gui.Text(c, "Chrome 配置")).TextColor(t.TextMuted).
			Label("Chrome 配置")
		selected := st.selected
		if selected < 0 || selected >= len(st.profiles) {
			selected = 0
		}
		for i, profile := range st.profiles {
			i, profile := i, profile
			disabled := st.phase == browserImportImporting
			row := gui.Row(c).Shrink(0).MinHeight(generalProfileRowHeight).
				Padding(space1, space2).Gap(space2).AlignItems(gui.Center).
				Radius(radius).Role(gui.RoleRadio).
				AccessSelected(i == selected).Disabled(disabled).
				Label(profile.name + " " + profile.capability)
			if i == selected {
				row = row.Background(t.SurfaceHover)
			}
			if disabled {
				row = row.Opacity(noteDisabledAlpha)
			}
			row.Children(func() {
				// 自绘单选圈 + 选中点
				ringColor := t.Border
				if i == selected {
					ringColor = t.Accent
				}
				ring := gui.Box(c).Width(generalRadioSize).Height(generalRadioSize).
					Shrink(0).Radius(generalRadioSize / 2).
					Draw(func(p *gui.Painter, r gui.Rect) {
						p.Stroke(r, ringColor, generalRadioSize/2, strokeHairline)
					})
				ring.Children(func() {
					if i == selected {
						gui.Box(c).Width(generalRadioDot).Height(generalRadioDot).
							Radius(generalRadioDot / 2).Background(t.Accent)
					}
				})
				gui.Box(c).Grow(1).MinWidth(0).Children(func() {
					typeSecondary.style(t, gui.Text(c, profile.name)).TextColor(t.Text).
						Tooltip(profile.name).SingleLine().Label(profile.name)
				})
				typeMicro.style(t, gui.Text(c, profile.capability)).
					TextColor(t.TextMuted).Label(profile.capability)
			})
			if !disabled && row.Clicked() {
				st.selected = i
			}
		}
	})
}

// generalProfileRowHeight 是配置单选行的高（需求给 40）。
const generalProfileRowHeight = 40

// generalRadioSize / generalRadioDot 是配置单选圈与选中点的直径：圈 16 取
// icon-inline 档，选中点 8 取 space-2 档（需求：圈 16、点 8）。
const (
	generalRadioSize = iconInline
	generalRadioDot  = space2
)

// generalBrowserCounts 画计数块（G-H6-08）：可导入 · 会话 · 已过期 · 分区，
// 每一项为 0 也照写，不省略。
func generalBrowserCounts(c *gui.Context, counts browserCookieCounts) {
	t := c.Theme()
	text := fmt.Sprintf("可导入 %d · 会话 %d · 已过期 %d · 分区 %d",
		counts.importable, counts.session, counts.expired, counts.partitioned)
	gui.Box(c).Shrink(0).Padding(space2).Radius(screenRadiusCtl).
		Background(t.SurfaceHover).Children(func() {
		typeMicro.style(t, gui.Text(c, text)).TextColor(t.TextMuted).
			Label(text).Selectable()
	})
}

// generalBrowserResult 画导入回执：失败数为 0 是成功行（G-H6-09a），大于 0 是
// 告警行 + 12px 警示说明（G-H6-09b）。
func generalBrowserResult(c *gui.Context, st *browserImportState) {
	t := c.Theme()
	text := fmt.Sprintf("已导入 %d · 失败 %d · 已跳过 %d 个过期 Cookie 和 %d 个分区 Cookie。",
		st.imported, st.failed, st.counts.expired, st.counts.partitioned)
	gui.Column(c).Gap(screenSpace05).Role(gui.RoleStatus).Label(text).Children(func() {
		icon, ink := iconCheck, t.Success
		if st.failed > 0 {
			icon, ink = iconWarn, t.Warning
		}
		gui.Row(c).Gap(space1).AlignItems(gui.Start).Children(func() {
			screenIcon(c, icon, screenIconInline, ink, "导入结果")
			typeSecondary.style(t, gui.Text(c, text)).TextColor(t.Text).Selectable()
		})
		if st.failed > 0 {
			// G-H6-09b 12px 警示说明
			typeMicro.style(t, gui.Text(c, browserImportPartialText)).
				TextColor(t.Warning).Label(browserImportPartialText).Selectable()
		}
	})
}

// generalBrowserError 画错误块（G-H6-10）：危险色消息 + 等宽诊断行 + 「重试」。
// 没有通道时「重试」置灰并写原因；重试进行中同样禁用。
func generalBrowserError(c *gui.Context, st *browserImportState) {
	t := c.Theme()
	message := st.message
	if message == "" {
		message = browserImportFailText
	}
	gui.Column(c).Gap(space1).Role(gui.RoleAlert).Label(message).Children(func() {
		typeSecondary.style(t, gui.Text(c, message)).TextColor(t.Danger).
			Label(message).Selectable()
		if st.diagnostic != "" {
			// 诊断行：等宽 12px、可折行、可复制。
			typeCode.style(t, gui.Text(c, st.diagnostic)).TextColor(t.TextMuted).
				Label(st.diagnostic).Selectable()
		}
		gui.Row(c).Justify(gui.End).Children(func() {
			tip := noteBrowserNoImport
			if st.retrying {
				tip = "正在重试"
			}
			screenTextButtonDisabled(c, "重试", tip, rowRegular)
		})
	})
}

// generalClearStates 画清理对话框的当前态（G-H7-03 / G-H7-04）。「没有通道」
// 与「清理中」都只写原因 / 状态，不画回执。
func generalClearStates(c *gui.Context, st *browserClearState) {
	switch st.phase {
	case browserClearSuccess:
		generalClearSuccess(c)
	case browserClearFailed:
		generalClearError(c, st)
	default:
		generalNote(c, noteBrowserNoClear)
	}
}

// generalClearSuccess 画清理成功行（G-H7-03）：对勾 + 逐字文案，无底色。
func generalClearSuccess(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Gap(space1).AlignItems(gui.Center).Role(gui.RoleStatus).
		Label(browserClearSuccessText).Children(func() {
		screenIcon(c, iconCheck, screenIconInline, t.Success, "已清理")
		typeSecondary.style(t, gui.Text(c, browserClearSuccessText)).
			TextColor(t.Text).Label(browserClearSuccessText).Selectable()
	})
}

// generalClearError 画清理失败块（G-H7-04）：危险色消息 + 等宽诊断行。
func generalClearError(c *gui.Context, st *browserClearState) {
	t := c.Theme()
	message := st.message
	if message == "" {
		message = browserClearFailText
	}
	gui.Column(c).Gap(space1).Role(gui.RoleAlert).Label(message).Children(func() {
		typeSecondary.style(t, gui.Text(c, message)).TextColor(t.Danger).
			Label(message).Selectable()
		if st.diagnostic != "" {
			typeCode.style(t, gui.Text(c, st.diagnostic)).TextColor(t.TextMuted).
				Label(st.diagnostic).Selectable()
		}
	})
}
