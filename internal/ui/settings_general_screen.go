package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 通用」的重建屏：元素按
// docs/domains/requirements/inventory-settings-rt.md §5（S-general-01 … 90，
// 90 个元素）逐行对应，文案逐字取参考产品中文，数值取
// docs/domains/requirements/design-baseline.md（卡片圆角 8、控件高 32、
// 开关 28×16、分段项高 41、说明字 10.5）。
//
// 只保留可丢的界面状态（generalScreen）：三个开关、会话数据根目录、代理模式
// 与四个对话框。写事实（开机自启、性能日志、代理、浏览器数据、存储清理）
// 需要设置领域与宿主能力，本屏只改界面状态，接线时换成 Intents。
//
// 本屏共用的构件在 settings_appearance_screen.go 的「三屏共用构件」一节。

// generalState 是本屏可丢的界面状态：开关位置、输入草稿与对话框。不是事实。
type generalState struct {
	keepAwake bool
	perfLog   bool
	launch    bool
	rootDir   string

	proxyMode   int
	proxyURL    string
	proxyDialog bool
	proxyDraft  string

	importOpen bool

	clearOpen   bool
	cleanupOpen bool
}

var generalScreen = generalState{
	keepAwake: true,  // 截图：开
	perfLog:   false, // 截图：关
	launch:    true,  // 截图：开
	proxyMode: 0,     // 系统代理
	proxyURL:  "http://127.0.0.1:7890",
}

// generalProxyModes 是代理的三档，顺序与截图一致。
var generalProxyModes = []struct {
	name string
	icon *gui.SVG
}{
	{"系统代理", screenIconDesktop},
	{"无代理", screenIconBan},
	{"自定义代理", screenIconSliders},
}

// G-H6-02 / G-H7-02 的说明句逐字取需求表，不截断（长句在对话框里折行）。
const (
	generalImportDesc = "此操作会替换内置浏览器中的全部非分区 Cookie，并重新加载已打开的页面。" +
		"每个 Cookie 都会单独尝试导入，因此个别 Cookie 可能失败；部分网站可能需要重新登录。"
	generalClearDesc = "此操作只会清除内置浏览器中的登录状态、Cookie、本地存储、IndexedDB、" +
		"Service Worker 和缓存。浏览器页面会关闭并以空白页重新打开。"
)

// renderGeneral 是设置 › 通用这一屏（设置壳的一个分节）。
//
// 连着 Source 时这一屏没有可读的事实：唤醒、性能日志、开机自启、数据根目录、
// 代理、浏览器数据都由桌面自己保管，而服务端不提供这些键。参考图里的控件形态
// 照留（开关、输入框、代理三档、两行动作），全部置灰、tooltip 写原因、卡片里
// 再写一行说明——看得见有什么，拨不动，也不把样例初值当现状。
// 存储用量是唯一有事实的项，读数取自 Settings().Storage。样例只在 -demo 出现。
func renderGeneral(c *gui.Context, a *App) {
	note := ""
	if a != nil && a.wired() {
		note = noteDesktopSettings
	}
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(screenSpace5).Children(func() {
				// S-general-01 分节标题「通用」
				screenTitle(c, "通用")
				// S-general-02 / S-general-04 保持电脑唤醒
				generalSwitchCard(c, "保持电脑唤醒",
					// S-general-03 保持电脑唤醒的说明句
					"运行期间阻止系统休眠。屏幕也会保持常亮。", &generalScreen.keepAwake, note)
				// S-general-05 / S-general-08 性能日志
				generalSwitchCard(c, "性能日志",
					// S-general-06 性能日志的说明句
					"将各进程 CPU/RSS/IO 采样写入 logs/performance/（每秒一行）。tail 该文件即可实时查看。",
					&generalScreen.perfLog, note)
				if generalPinned() {
					generalCard(c, func() {
						// S-general-07 本次启动由 DIM_PERF 决定采集
						screenHint(c, "本次启动由 DIM_PERF 决定采集，该开关不生效。")
					})
				}
				// S-general-09 / S-general-10 登录时自动启动
				generalSwitchCard(c, "登录时自动启动", "", &generalScreen.launch, note)
				generalRootDirCard(c, note)
				generalProxyCard(c, note)
				generalBrowserCard(c, note)
				generalCleanupCard(c, a, note)
			})
		})
	})
	generalProxyDialog(c)
	generalImportDialog(c)
	generalClearDialog(c)
	generalCleanupDialog(c, a)
}

// generalPinned 是性能日志被 DIM_PERF 钉住的条件：启动参数里没有设，返回假。
// 取值来自宿主环境，不在本屏读进程环境（那是设置领域的事），接线时由事实给。
func generalPinned() bool { return false }

// generalCard 是一个卡：卡面（白 / 暗 0c0c0c）+ 1px 主题描边 + 卡片圆角，
// 内边距 14/20。
func generalCard(c *gui.Context, body func()) {
	screenCard(c, func() {
		gui.Column(c).Padding(screenSpace35, screenSpace5).Gap(screenSpace15).
			Children(body)
	})
}

// generalSwitchCard 是一张开关卡：左侧标题（+ 说明），右侧 28×16 开关。
// note 非空时（连着 Source）开关照画并置灰，原因进 tooltip 与标题下的说明行。
func generalSwitchCard(c *gui.Context, label, hint string, on *bool, note string) {
	generalCard(c, func() {
		gui.Row(c).AlignItems(gui.Center).Gap(screenSpace4).Children(func() {
			gui.Column(c).Grow(1).MinWidth(0).Gap(screenSpace05).Children(func() {
				screenLabel(c, label)
				if hint != "" {
					screenHint(c, hint)
				}
				if note != "" {
					generalNote(c, note)
				}
			})
			sw := gui.Switch(c, on).Size(screenSwitchW, screenSwitchH).
				Radius(screenSwitchH / 2).Label(label)
			if note != "" {
				sw = sw.Disabled(true).Tooltip(note).Opacity(noteDisabledAlpha)
			}
		})
	})
}

// generalNote 是一行"这一项没有事实"的说明：次级前景、可选中复制。
func generalNote(c *gui.Context, text string) {
	typeMicro.style(c.Theme(), gui.Text(c, text)).TextColor(c.Theme().TextMuted).
		Selectable()
}

// generalRootDirCard 是会话数据根目录卡（S-general-11–17）。note 非空时只写
// 原因，不给输入框与「重启 App」（两者都没有通道）。
func generalRootDirCard(c *gui.Context, note string) {
	generalCard(c, func() {
		gui.Column(c).Gap(screenSpace05).Children(func() {
			screenLabel(c, "会话数据根目录")
			screenHint(c, "会话记录、日志与缓存都位于该目录下。留空则使用默认根目录。保存后需要重启 DimAgent 才能生效。")
		})
		if note != "" {
			generalNote(c, note)
		}
		gui.Row(c).AlignItems(gui.Center).Gap(screenSpace2).Children(func() {
			input := screenInput(c, &generalScreen.rootDir, "例如 /Users/you/.dimcode", "会话数据根目录").
				Grow(1)
			if note != "" {
				input = input.Disabled(true).Tooltip(note).Opacity(noteDisabledAlpha)
			}
			screenActionDisabled(c, screenIconRefresh, "refresh", "重启 App", screenControlH,
				note != "", note)
		})
	})
}

// generalProxyCard 是代理卡（S-general-18–30）。note 非空时只写原因，不画
// 三档分段（拨了不生效的分段等于一个假的现状）。
func generalProxyCard(c *gui.Context, note string) {
	generalCard(c, func() {
		gui.Column(c).Gap(screenSpace05).Children(func() {
			screenLabel(c, "代理")
			screenHint(c, "应用于模型请求、登录 API，以及后续新启动的工具进程。")
		})
		if note != "" {
			generalNote(c, note)
		}
		generalProxySegments(c, note != "", note)
		if generalScreen.proxyMode == 2 {
			gui.Column(c).Gap(screenSpace15).Children(func() {
				screenHint(c, "自定义代理")
				gui.Row(c).AlignItems(gui.Center).Gap(screenSpace2).Children(func() {
					screenCodeStyle.style(c.Theme(), gui.Text(c, generalScreen.proxyURL)).
						TextColor(c.Theme().TextMuted).Grow(1).MinWidth(0).SingleLine().
						Label("代理地址")
					if screenButton(c, "编辑", "编辑代理地址", "", false, rowRegular) {
						generalScreen.proxyDraft = generalScreen.proxyURL
						generalScreen.proxyDialog = true
					}
				})
			})
		}
	})
}

// generalProxySegments 是代理模式的三档分段：选中档主色描边、白底。
// disabled 为真时三格照画、整块置灰，点击不换档，原因进 tooltip。
func generalProxySegments(c *gui.Context, disabled bool, note string) {
	t := c.Theme()
	parts := gui.SegmentedBase(c, &generalScreen.proxyMode, len(generalProxyModes))
	track := parts.Track.AlignItems(gui.Stretch).Gap(screenStroke).
		Radius(screenRadiusCtl).Background(t.Border).Padding(screenStroke).
		Label("代理模式").Tooltip(note).Disabled(disabled)
	if disabled {
		track = track.Opacity(noteDisabledAlpha)
	}
	track.Children(func() {
		for i, mode := range generalProxyModes {
			seg := parts.Segment(i).Grow(1).Height(screenSegHLg).AlignItems(gui.Stretch).
				Padding(screenStroke).Radius(screenRadiusCtl).Background(t.Border).
				Label(mode.name)
			if i == generalScreen.proxyMode {
				seg = seg.Background(t.Accent)
			}
			seg.Children(func() {
				gui.Box(c).Grow(1).Radius(screenRadiusCtl).Background(t.Background).
					Center().Children(func() {
					gui.Row(c).Gap(screenSpace15).AlignItems(gui.Center).Children(func() {
						ink := t.TextMuted
						if i == generalScreen.proxyMode {
							ink = t.Text
						}
						screenIcon(c, mode.icon, screenIconInline, ink, mode.name)
						screenStrongStyle.style(t, gui.Text(c, mode.name)).TextColor(t.Text).
							SingleLine()
					})
				})
			})
		}
	})
}

// generalBrowserCard 是浏览器数据卡（S-general-38–45）：标题与说明，下面两行
// 动作，行间 1px 细线。note 非空时两行动作禁用，并把原因写在卡里。
func generalBrowserCard(c *gui.Context, note string) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Children(func() {
		screenCard(c, func() {
			gui.Column(c).Padding(screenSpace35, screenSpace5).Gap(screenSpace05).Children(func() {
				screenStrongStyle.style(t, gui.Text(c, "浏览器数据")).TextColor(t.Text).Label("浏览器数据")
				screenHint(c, "网站登录状态和存储使用独立的浏览器配置。")
				if note != "" {
					generalNote(c, note)
				}
			})
			screenRule(c)
			generalBrowserRow(c, "Chrome 数据", "导入…", "导入 Chrome Cookie", note, func() {
				generalScreen.importOpen = true
			})
			screenRule(c)
			generalBrowserRow(c, "浏览器存储", "清理数据…", "清理浏览器数据", note, func() {
				generalScreen.clearOpen = true
			})
		})
	})
}

// generalBrowserRow 是浏览器数据卡里的一行：行标签 + 右端动作。disabled 非空
// 时动作置灰，原因进 tooltip。
func generalBrowserRow(c *gui.Context, label, action, tip, disabled string, run func()) {
	t := c.Theme()
	gui.Row(c).AlignItems(gui.Center).Padding(screenSpace3, screenSpace5).Children(func() {
		screenStrongStyle.style(t, gui.Text(c, label)).TextColor(t.Text).Label(label)
		gui.Spacer(c)
		if disabled != "" {
			screenTextButtonDisabled(c, action, disabled, rowRegular)
			return
		}
		if screenButton(c, action, tip, "", false, rowRegular) {
			run()
		}
	})
}

// generalCleanupCard 是数据清理卡（S-general-67–69）。存储用量是这一屏里唯一
// 有事实的项：连着 Source 时也保留入口，读数在对话框里取 Settings().Storage。
func generalCleanupCard(c *gui.Context, a *App, note string) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Children(func() {
		screenCard(c, func() {
			gui.Row(c).AlignItems(gui.Center).Gap(screenSpace4).
				Padding(screenSpace35, screenSpace5).Children(func() {
				gui.Column(c).Grow(1).MinWidth(0).Gap(screenSpace05).Children(func() {
					screenStrongStyle.style(t, gui.Text(c, "数据清理")).TextColor(t.Text).Label("数据清理")
					screenHint(c, "会话存储由 App Server 管理，可在此查看其存储用量报告。")
					if note != "" {
						generalNote(c, note)
					}
				})
				if screenButton(c, "查看…", "查看存储用量报告", "", false, rowRegular) {
					generalScreen.cleanupOpen = true
				}
			})
		})
	})
}

// ---- 对话框 ----

// generalProxyDialog 是自定义代理对话框（S-general-31–37）。
func generalProxyDialog(c *gui.Context) {
	if !generalScreen.proxyDialog {
		return
	}
	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		screenDialogPanel(c, "自定义代理", func() {
			screenHint(c, "输入 http 或 https 代理 URL。")
			gui.Column(c).Gap(screenSpace15).Children(func() {
				screenLabel(c, "代理 URL")
				screenInput(c, &generalScreen.proxyDraft, "http://127.0.0.1:7890", "代理 URL")
			})
			gui.Row(c).Justify(gui.End).Gap(screenSpace2).Children(func() {
				if screenButton(c, "取消", "取消代理设置", "", false, rowRegular) {
					generalScreen.proxyDialog = false
					open = false
				}
				if screenButton(c, "保存", "保存代理设置", "", true, rowRegular) {
					generalScreen.proxyURL = generalScreen.proxyDraft
					generalScreen.proxyMode = 2
					generalScreen.proxyDialog = false
					open = false
				}
			})
		})
	}))

	if !open || m.Dismissed() {
		generalScreen.proxyDialog = false
	}
}

// generalImportDialog 是「导入 Chrome Cookie」对话框（S-general-46–60）。
//
// 导入整条链（扫描 Chrome 配置、按配置读计数、逐个导入并给部分失败回执）都要
// 桌面宿主的浏览器数据通道；本版没有这个通道，所以扫描中 / 无配置 / 读取中 /
// 部分失败 / 错误重试 / 成功回执这些态都到不了。这里不画样例配置与假计数，只
// 保留标题、说明与两个按钮，并把缺什么写在对话框里。
func generalImportDialog(c *gui.Context) {
	if !generalScreen.importOpen {
		return
	}
	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		// G-H6-01 「导入 Chrome Cookie」
		screenDialogPanel(c, "导入 Chrome Cookie", func() {
			// G-H6-02 导入说明句（逐字，不截断）
			screenHint(c, generalImportDesc)
			// 扫描行、配置行与计数没有读数：写原因（不画样例配置与假计数）
			generalNote(c, noteBrowserNoImport)
			gui.Row(c).Justify(gui.End).Gap(screenSpace2).Children(func() {
				// G-H6-11 取消
				if screenButton(c, "取消", "取消导入 Chrome Cookie", "", false, rowRegular) {
					generalScreen.importOpen = false
					open = false
				}
				// G-H6-12 导入（没有通道：置灰并写原因）
				screenTextButtonDisabled(c, "导入", noteBrowserNoImport, rowRegular)
			})
		})
	}))

	if !open || m.Dismissed() {
		generalScreen.importOpen = false
	}
}

// generalClearDialog 是「清理浏览器数据」对话框（S-general-61–66）。
//
// 清理成功 / 失败回执要宿主真的清一遍浏览器数据；本版没有这个通道，因此不假装
// 清理成功（成功行与错误块都画不出来），只在对话框里写清缺什么，动作置灰。
func generalClearDialog(c *gui.Context) {
	if !generalScreen.clearOpen {
		return
	}
	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		// G-H7-01 「清理浏览器数据」
		screenDialogPanel(c, "清理浏览器数据", func() {
			// G-H7-02 清理说明句（逐字，不截断）
			screenHint(c, generalClearDesc)
			// 成功行与错误块没有回执：写原因（不假装清理成功）
			generalNote(c, noteBrowserNoClear)
			gui.Row(c).Justify(gui.End).Gap(screenSpace2).Children(func() {
				// G-H7-05 取消
				if screenButton(c, "取消", "取消清理浏览器数据", "", false, rowRegular) {
					generalScreen.clearOpen = false
					open = false
				}
				// G-H7-06 清理浏览器数据（没有通道：置灰并写原因）
				screenTextButtonDisabled(c, "清理浏览器数据", noteBrowserNoClear, rowRegular)
			})
		})
	}))

	if !open || m.Dismissed() {
		generalScreen.clearOpen = false
	}
}

// generalCleanupDialog 是数据清理对话框的 App Server 支（S-general-70–74、86）：
// 扫描就绪后给出数据库 / WAL / 可回收三行读数，并说明跨归属方不可清理。
func generalCleanupDialog(c *gui.Context, a *App) {
	if !generalScreen.cleanupOpen {
		return
	}
	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		screenDialogPanel(c, "存储报告", func() {
			screenHint(c, "会话存储由 App Server 管理，可在此查看其存储用量报告。")
			if a != nil && a.wired() {
				// 读数取 Settings().Storage：没读到就写没读到，不写 0 B（0 B 是一个
				// 会被当成事实的数字）。
				st, _ := a.liveSettings()
				if !st.StorageKnown {
					// 读失败与「还没读到」分开写：有原因就给原因，不在读数位置留空。
					if st.StorageError != "" {
						generalNote(c, "读取存储用量失败："+st.StorageError)
						return
					}
					generalNote(c, noteStorageUnread)
					return
				}
				gui.Column(c).Gap(screenSpace15).Children(func() {
					generalMetricRow(c, "数据库", rpByteLabel(st.Storage.DatabaseBytes))
					generalMetricRow(c, "WAL", rpByteLabel(st.Storage.WALBytes))
					generalMetricRow(c, "可回收", rpByteLabel(st.Storage.ReclaimableBytes))
				})
				return
			}
			gui.Column(c).Gap(screenSpace15).Children(func() {
				generalMetricRow(c, "数据库", "0 B")
				generalMetricRow(c, "WAL", "0 B")
				generalMetricRow(c, "可回收", "0 B")
			})
			screenHint(c, "会话存储由 App Server 管理，此应用不支持跨归属方的扫描和清理。")
			screenHint(c, "如需回收可回收空间，请停止守护进程并离线运行 `dimcode daemon vacuum`。")
			gui.Row(c).Justify(gui.End).Gap(screenSpace2).Children(func() {
				// S-general-89 关闭数据清理
				if screenButton(c, "关闭", "关闭数据清理", "", false, rowRegular) {
					generalScreen.cleanupOpen = false
					open = false
				}
			})
		})
	}))

	if !open || m.Dismissed() {
		generalScreen.cleanupOpen = false
	}
}

// generalMetricRow 是一条读数：左标签、右值。
func generalMetricRow(c *gui.Context, label, value string) {
	t := c.Theme()
	gui.Row(c).AlignItems(gui.Center).Children(func() {
		screenValueStyle.style(t, gui.Text(c, label)).TextColor(t.TextMuted).Label(label)
		gui.Spacer(c)
		screenValueStyle.style(t, gui.Text(c, value)).TextColor(t.Text).SingleLine()
	})
}
