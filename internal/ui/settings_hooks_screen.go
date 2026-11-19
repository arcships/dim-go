package ui

import (
	"encoding/json"
	"strconv"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › Hooks 的重建实现：元素按
// docs/domains/requirements/inventory-settings-rt.md §15（S-hooks-01 … 32，
// 32 个元素）逐行对应，文案逐字取参考产品中文，数值取
// docs/domains/requirements/design-baseline.md：
//
//	页面标题 17.5/600、分组小标题 12.25/600、名称 13/600、说明与状态 11–12.25、
//	卡片圆角 8、卡内边距 14、来源图标块 32×32、开关 28×16、行内按钮高 28。
//
// 当前可见分支是 App Server 目录（HooksAppServerCatalog）：来源与条目连着
// Source 时每帧从 Extensions().Hooks 分组读（按 source 前缀分成用户 / 项目 /
// 插件三组），没有 Source（-demo）时用样例。屏内只保留**可丢的界面状态**
// （hooksScreen）：编辑框开合、草稿与保存失败文本。写事实本屏没有接到控件上：
// 条目行没有单项开关（参考形态如此），项目总开关的写入（hooks.set_enabled）
// 不在 Intents 里，所以这两处都只改本屏状态。
//
// # 未实现分支
//
//	S-hooks-27…32  非 App Server 分支（本地 config 页、来自配置/工作区/插件三组、
//	               本地钩子编辑对话框）：本客户端恒为 App Server 宿主界面，
//	               该分支不可达（同一判断见清单 §15.2）。
//	S-hooks-12…15  来源卡的状态（正常 / 缺失）、启用徽标、项目总开关与页尾修订号
//	               要有 hooks.catalog 的来源快照（路径 / state / revision）；扩展
//	               读数只给条目，这几行在连着 Source 时不画（不拿条目猜来源状态）。
//	S-hooks-11     插件来源的 Lock 图标：来源里有 plugin 前缀的条目时出现
//	               （形状见 hkIconLock）。
//
// # 与参考的差异
//
//	标题字号取设计基线的 17.5（清单 R-HOK-002 写的 20 是 16px 根字号的换算值；
//	design-baseline §2 的页面标题是 17.5/600，与「账户」「归档会话」同档）。
//	「了解更多」没有外链地址（没有文档 URL 的事实来源），按钮照渲染、点击不动作。

// hooksEntry 是一条钩子：事件、状态、异步标记、命令与三个可选修饰。
type hooksEntry struct {
	event   string
	enabled bool
	async   bool
	command string
	matcher string
	// timeout 是钩子目录给的上限原值（0 表示没有限制）；单位由服务端文档定，
	// 本屏只原样展示这个数，不自行加单位。
	timeout int
	onFail  string
}

// hooksSource 是一个来源：名称、路径、状态、读写权限与条目。
type hooksSource struct {
	title string
	// user 为真表示用户来源（可编辑、无路径）；否则是项目来源。
	user     bool
	path     string
	revision string
	enabled  bool
	readonly bool
	// project 为真表示项目来源：它有一处可选的目录（本端偏好记着选中的那个）。
	project bool
	// plugin 为真时在来源图标右下角叠一个锁（S-hooks-11）。
	plugin bool
	// missing 为真表示来源文件缺失（状态标「缺失」）。
	missing bool
	message string
	entries []hooksEntry
}

// hooksState 是本屏可丢的界面状态：不是事实。
type hooksState struct {
	// saveFailure 是文档保存失败的原因；编辑框草稿与开合也只是视图状态。
	saveFailure string
	editorOpen  bool
	draft       string
}

// hooksScreen 只装视图状态：来源与条目每帧从 Source 读（-demo 时读样例）。
var hooksScreen = hooksState{}

// hkSampleSources 是 -demo 的样例：用户来源正常且启用、项目来源缺失且禁用、
// 两组条目均为空。它只在没有 Source 时上屏。
var hkSampleSources = []hooksSource{
	{
		title: "用户", user: true, revision: "native:user/hooks:1",
		enabled: true, readonly: true,
	},
	{
		title: "项目", path: "~/.dimcode/hooks.json", project: true,
		revision: "native:user/hooks-projects:1",
		enabled:  false, readonly: true, missing: true,
	},
}

// hooksData 取这一帧的来源与条目：连着 Source 时从 Extensions().Hooks 按
// source 前缀分组，否则用样例。第二个返回值是读数本身的读态。
func (a *App) hooksData() ([]hooksSource, ExtListSnapshot) {
	if a.source == nil {
		return hkSampleSources, ExtListSnapshot{}
	}
	list := a.source.Extensions().Hooks
	sources := make([]hooksSource, 0, 3)
	index := map[string]int{}
	for _, item := range list.Items {
		source := hkSourceOf(item)
		i, ok := index[source.title]
		if !ok {
			index[source.title] = len(sources)
			sources = append(sources, source)
			i = len(sources) - 1
		}
		sources[i].entries = append(sources[i].entries, hooksEntry{
			event:   item.Event,
			enabled: item.Enabled,
			async:   item.Async,
			command: item.Command,
			matcher: item.Matcher,
			timeout: item.Timeout,
			onFail:  item.OnFailure,
		})
	}
	return sources, list
}

// hkSourceOf 把一条钩子的来源前缀映射成来源卡：用户 / 项目 / 插件（其余来源
// 用一个只读卡装，名称取原始前缀）。
func hkSourceOf(item ExtItemSnapshot) hooksSource {
	switch item.Source {
	case "user":
		return hooksSource{title: "用户", user: true, enabled: true, readonly: true}
	case "project":
		return hooksSource{title: "项目", enabled: true, readonly: true, project: true}
	case "plugin":
		return hooksSource{title: "插件", enabled: true, readonly: true, plugin: true}
	}
	title := item.Source
	if title == "" {
		title = "其他"
	}
	return hooksSource{title: title, enabled: true, readonly: true}
}

// 本屏几何：参考的 Tailwind 量换算成 design-baseline §1.1 的档位。
const (
	hkSectionGap = 21   // space-y-6：分组之间
	hkLabelGap   = 14   // 小标题与其列表之间
	hkCardGap    = 12   // 来源卡 / 条目组之间
	hkPadV       = 14   // p-4 上下
	hkPadH       = 14   // p-4 左右
	hkRowGap     = 10.5 // gap-3：图标与文字块之间
	hkChipGap    = 7    // gap-2：名称 / 状态 / 徽标之间
	hkTightGap   = 2    // 名称与状态徽标的上下贴合
	hkTextGap    = 3.5  // 名称与路径之间

	hkIconBlock  = 32 // h-8 w-8：来源图标块
	hkIconInline = 16 // h-4 w-4：来源图标
	hkIconMicro  = 14 // h-3.5：警示与按钮图标
	hkIconHit    = 28 // 图标按钮 28×28
	hkIconSize   = 14

	hkBtnH   = 28 // h-7：了解更多 / 编辑
	hkBadgeV = 3.5
	hkBadgeH = 7
	hkBadgeR = 4.75

	hkBannerPadV  = 7   // 提示条上下
	hkBannerPadH  = 14  // 提示条左右
	hkBannerR     = 6   // rounded-md
	hkEmptyPadV   = 9   // 空态行的上下
	hkGroupPadV   = 9.5 // 组头的上下
	hkEntryH      = 44  // 条目行高
	hkEntryPadV   = 7
	hkEditorMinH  = 180
	hkAlphaBadge  = 0.10
	hkAlphaBorder = 0.35
)

// 字阶（design-baseline §2）。
var (
	hkNameStyle    = fontStep{size: 13, line: 19.5, weight: 600}
	hkStatusStyle  = fontStep{size: 12.25, line: 17.5, weight: 400}
	hkNoteStyle    = fontStep{size: 12.25, line: 17.5, weight: 400}
	hkBadgeStyle   = fontStep{size: 10.5, line: 14, weight: 500}
	hkPathStyle    = fontStep{size: 11, line: 17, weight: 400, family: codeFont}
	hkCodeStyle    = fontStep{size: 11, line: 17, weight: 400, family: codeFont}
	hkRevisionSty  = fontStep{size: 11, line: 17, weight: 400, family: codeFont}
	hkEntryName    = fontStep{size: 12.25, line: 17.5, weight: 600}
	hkEntryMetaSty = fontStep{size: 10.5, line: 14, weight: 400}
)

// 本屏图标：Heroicons v2 outline（24 网格、1.5 描边、MIT）的内联路径。
var (
	hkIconLock = mustSVG(`<path d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z"/>`)
)

// 本屏文案：逐字取自参考产品的 zh-CN。
const (
	hkCopyTitle      = "命令钩子"
	hkCopyMore       = "了解更多"
	hkCopyRefresh    = "刷新"
	hkCopyLoadError  = "加载设置失败"
	hkCopyNoCatalog  = "服务端没有声明钩子目录能力，这一屏只有本端能确认的部分。"
	hkCopyWarning    = "钩子会在这些生命周期事件上运行 shell 命令。"
	hkCopySources    = "来源"
	hkCopyEntries    = "条目"
	hkCopyNoEntries  = "没有条目"
	hkCopyNormal     = "正常"
	hkCopyMissing    = "缺失"
	hkCopyEnabled    = "已启用"
	hkCopyDisabled   = "已禁用"
	hkCopyReadonly   = "只读"
	hkCopyEdit       = "编辑"
	hkCopyProjectOn  = "启用项目钩子"
	hkCopyProjectOff = "禁用项目钩子"
	hkCopyRevUser    = "用户修订号 "
	hkCopyRevProject = "项目修订号 "
	hkCopyDocTitle   = "用户钩子文档"
	hkCopyDocDesc    = "编辑用户钩子文档。保存时会按本页读取的修订号替换。"
	hkCopyBadJSON    = "文档必须是 JSON 对象。"
	hkCopyCancel     = "取消"
	hkCopySave       = "保存"
	hkCopyAsync      = "异步"
	hkCopyHooksKey   = "hooks"
	hkCopyChooseDir  = "选择项目目录"
	hkCopyNoPath     = "未选择项目目录"
)

// hkSourcePath 是来源卡要显示的路径：项目目录取本端登记的那一个（选中后以它
// 为准），没有登记时用事实 / 样例带来的路径。
func (a *App) hkSourcePath(s *hooksSource) string {
	if s.user {
		return ""
	}
	if s.project && a.hooksProjectPath != "" {
		return a.hooksProjectPath
	}
	return s.path
}

// renderHooks 是设置 › Hooks 这一屏（设置壳的一个分节）。
func renderHooks(c *gui.Context, a *App) {
	t := c.Theme()
	sources, list := a.hooksData()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(hkSectionGap).Children(func() {
				hkHeader(c, t)
				if list.ReadError != "" {
					hkAlertLine(c, t, hkCopyLoadError+"："+list.ReadError)
				}
				if list.CapabilityMissing {
					hkNoteLine(c, t, hkCopyNoCatalog)
				}
				hkBanner(c, t, hkCopyWarning, true)
				gui.Column(c).Shrink(0).Gap(hkLabelGap).Children(func() {
					screenGroupTitle(c, hkCopySources)
					for i := range sources {
						a.hkSourceCard(c, t, &sources[i])
					}
				})
				gui.Column(c).Shrink(0).Gap(hkLabelGap).Children(func() {
					screenGroupTitle(c, hkCopyEntries)
					for i := range sources {
						hkEntryGroup(c, t, &sources[i])
					}
				})
				if hooksScreen.saveFailure != "" {
					hkAlertLine(c, t, hooksScreen.saveFailure)
				}
				// 修订号要有 hooks.catalog 的来源快照：只在样例里画。
				if a.source == nil {
					hkRevisions(c, t, sources)
				}
			})
		})
	})
	hkEditorDialog(c, t)
	a.dirPickDialog(c, dirPickHooksProject)
}

// hkHeader 是页首（S-hooks-01…03）：标题 + 右端「了解更多」与刷新。
func hkHeader(c *gui.Context, t *gui.Theme) {
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(hkRowGap).Children(func() {
		screenTitle(c, hkCopyTitle)
		gui.Spacer(c)
		if screenAction(c, accIconExternal, "external-link", hkCopyMore, hkBtnH) {
			// 面板外开需要文档地址（没有该事实）：不动作（未接动作）。
		}
		if accIconButton(c, accIconSpec{
			icon: accIconRefresh, label: hkCopyRefresh, tip: hkCopyRefresh,
			size: hkIconSize, hit: hkIconHit,
		}) {
			// 重读目录需要 hooks.catalog 的 Intents：不动作（未接动作）。
		}
	})
}

// hkAlertLine 是一条错误行（S-hooks-04/16/21）：12.25、警示色。
func hkAlertLine(c *gui.Context, t *gui.Theme, text string) {
	hkNoteStyle.style(t, gui.Text(c, text)).TextColor(t.Danger).
		Role(gui.RoleAlert).Label(text)
}

// hkNoteLine 是一条次级说明行（S-hooks-05）：12.25、次级前景。
func hkNoteLine(c *gui.Context, t *gui.Theme, text string) {
	hkNoteStyle.style(t, gui.Text(c, text)).TextColor(t.TextMuted).Label(text)
}

// hkBanner 是一条提示条（S-hooks-06/07）：可选警示图标 + 一句说明。
// 两个提示条都不带动作，点击不响应。
func hkBanner(c *gui.Context, t *gui.Theme, text string, warn bool) {
	row := gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(hkChipGap).
		Padding(hkBannerPadV, hkBannerPadH).Radius(hkBannerR).
		Background(t.SurfaceHover).Label(text)
	row.BorderWidth(strokeHairline).BorderColor(t.Border)
	row.Children(func() {
		if warn {
			accIcon(c, accIconTriangleAlert, hkIconMicro, t.TextMuted, "triangle-alert")
		}
		hkNoteStyle.style(t, gui.Text(c, text)).TextColor(t.TextMuted)
	})
}

// hkSourceCard 是一个来源（S-hooks-09/10/12/14/15）：图标块、名称、状态、
// 徽标、可选路径与消息，右端是「编辑」或项目开关。
//
// 连着 Source 时来源状态（正常 / 缺失）、来源级启用徽标与项目总开关都没有
// 事实（扩展读数只给条目），这几种行不画；插件来源的只读徽标是事实，照画。
func (a *App) hkSourceCard(c *gui.Context, t *gui.Theme, s *hooksSource) {
	wired := a.source != nil
	card := accCardFrame(c, gui.Row(c).Shrink(0).AlignItems(gui.Start).
		Gap(hkRowGap).Padding(hkPadV, hkPadH))
	card.Label(s.title)
	card.Children(func() {
		hkSourceIcon(c, t, s)
		gui.Column(c).Grow(1).MinWidth(0).Gap(hkTightGap).Children(func() {
			gui.Row(c).AlignItems(gui.Center).Gap(hkChipGap).Children(func() {
				hkNameStyle.style(t, gui.Text(c, s.title)).TextColor(t.Text).SingleLine()
				if !wired {
					hkStatusStyle.style(t, gui.Text(c, hkSourceState(s))).TextColor(t.TextMuted)
					hkBadge(c, t, hkEnabledLabel(s.enabled), s.enabled)
				}
				if s.plugin {
					hkBadge(c, t, hkCopyReadonly, false)
				}
			})
			if path := a.hkSourcePath(s); path != "" {
				hkPathStyle.style(t, gui.Text(c, path)).TextColor(t.TextMuted).
					SingleLine().Label("路径：" + path)
			} else if s.project {
				hkNoteLine(c, t, hkCopyNoPath)
			}
			if s.message != "" {
				hkNoteLine(c, t, s.message)
			}
		})
		if s.user {
			if screenAction(c, iconPencil, "pencil", hkCopyEdit, hkBtnH) {
				hooksScreen.draft = hkUserDoc(s.entries)
				hooksScreen.saveFailure = ""
				hooksScreen.editorOpen = true
			}
			return
		}
		// 项目目录：选中后写进本端偏好，路径行与后续按目录定位都以它为准。
		if s.project {
			if screenAction(c, screenIconFolderOpen, "folder-open", hkCopyChooseDir, hkBtnH) {
				a.startDirPick(dirPickHooksProject)
			}
		}
		if !wired {
			hkProjectToggle(c, t, s)
		}
	})
}

// hkSourceIcon 是来源图标块（S-hooks-10）：32×32、圆角、次级面；插件来源
// 在右下角叠一个锁（S-hooks-11）。
func hkSourceIcon(c *gui.Context, t *gui.Theme, s *hooksSource) {
	box := gui.Box(c).Width(hkIconBlock).Height(hkIconBlock).Shrink(0).
		Background(t.SurfaceHover).Radius(radius).Label("图标：来源 " + s.title).
		AccessHidden(true)
	box.Children(func() {
		gui.Box(c).Grow(1).Center().Children(func() {
			icon := iconUserCircle
			if !s.user {
				icon = screenIconFolderOpen
			}
			accIcon(c, icon, hkIconInline, t.TextMuted, "")
		})
	})
	if !s.plugin {
		return
	}
	lock := gui.Box(c).Absolute().Right(0).Bottom(0).Width(hkIconLockSize).
		Height(hkIconLockSize).Label("图标：插件来源").AccessHidden(true)
	lock.DrawOver(func(p *gui.Painter, r gui.Rect) {
		paintIcon(p, hkIconLock, r, t.TextMuted)
	})
}

// hkIconLockSize 是插件来源锁图标的尺寸（12）。
const hkIconLockSize = 12

// hkProjectToggle 是项目来源的开关（S-hooks-15）：文案随状态换，开关同步。
func hkProjectToggle(c *gui.Context, t *gui.Theme, s *hooksSource) {
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(hkChipGap).Children(func() {
		label := hkCopyProjectOn
		if s.enabled {
			label = hkCopyProjectOff
		}
		hkStatusStyle.style(t, gui.Text(c, label)).TextColor(t.TextMuted)
		gui.Switch(c, &s.enabled).Size(screenSwitchW, screenSwitchH).
			Radius(screenSwitchH / 2).Label(label)
	})
}

// hkEntryGroup 是一个来源的条目组（S-hooks-18/19/20）：组头 + 条目或空态。
func hkEntryGroup(c *gui.Context, t *gui.Theme, s *hooksSource) {
	screenCard(c, func() {
		gui.Column(c).Shrink(0).Children(func() {
			gui.Row(c).AlignItems(gui.Center).Gap(hkChipGap).
				Padding(hkGroupPadV, hkPadH).Children(func() {
				icon := iconUserCircle
				if !s.user {
					icon = screenIconFolderOpen
				}
				accIcon(c, icon, hkIconMicro, t.TextMuted, "")
				hkNameStyle.style(t, gui.Text(c, s.title)).TextColor(t.Text).SingleLine()
				gui.Spacer(c)
				hkRevisionSty.style(t, gui.Text(c, hkEnabledLabel(s.enabled))).
					TextColor(t.TextMuted)
			})
			screenRule(c)
			if len(s.entries) == 0 {
				gui.Row(c).Shrink(0).Padding(hkEmptyPadV, hkPadH).Children(func() {
					hkNoteStyle.style(t, gui.Text(c, hkCopyNoEntries)).
						TextColor(t.TextMuted)
				})
				return
			}
			for i := range s.entries {
				if i > 0 {
					screenRule(c)
				}
				hkEntryRow(c, t, &s.entries[i])
			}
		})
	})
}

// hkEntryRow 是一条钩子（S-hooks-20）：事件、状态、异步标记、命令与修饰。
func hkEntryRow(c *gui.Context, t *gui.Theme, e *hooksEntry) {
	gui.Row(c).Shrink(0).MinHeight(hkEntryH).AlignItems(gui.Center).
		Role(gui.RoleListItem).
		Label(e.event+"，"+hkEnabledLabel(e.enabled)).
		Gap(hkChipGap).Padding(hkEntryPadV, hkPadH).Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Gap(hkTightGap).Children(func() {
			gui.Row(c).AlignItems(gui.Center).Gap(hkChipGap).Children(func() {
				hkEntryName.style(t, gui.Text(c, e.event)).TextColor(t.Text).SingleLine()
				hkStatusStyle.style(t, gui.Text(c, hkEnabledLabel(e.enabled))).
					TextColor(t.TextMuted)
				if e.async {
					hkBadge(c, t, hkCopyAsync, false)
				}
			})
			hkCodeStyle.style(t, gui.Text(c, e.command)).TextColor(t.TextMuted).
				SingleLine().Label("命令：" + e.command)
			if meta := hkEntryMetaLine(e); meta != "" {
				hkEntryMetaSty.style(t, gui.Text(c, meta)).TextColor(t.TextMuted).SingleLine()
			}
		})
	})
}

// hkEntryMetaLine 拼出条目行末段的匹配器 / 超时 / 失败时。
func hkEntryMetaLine(e *hooksEntry) string {
	parts := make([]string, 0, 3)
	if e.matcher != "" {
		parts = append(parts, "匹配器 "+e.matcher)
	}
	if e.timeout > 0 {
		parts = append(parts, "超时 "+strconv.Itoa(e.timeout))
	}
	if e.onFail != "" {
		parts = append(parts, "失败时 "+e.onFail)
	}
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += " · "
		}
		out += part
	}
	return out
}

// hkRevisions 是页尾的两行修订号（S-hooks-22）：等宽、次级。
func hkRevisions(c *gui.Context, t *gui.Theme, sources []hooksSource) {
	user, project := "", ""
	if len(sources) > 1 {
		user, project = sources[0].revision, sources[1].revision
	}
	gui.Column(c).Shrink(0).Gap(hkTightGap).Children(func() {
		hkRevisionSty.style(t, gui.Text(c, hkCopyRevUser+user)).
			TextColor(t.TextMuted).Selectable()
		hkRevisionSty.style(t, gui.Text(c, hkCopyRevProject+project)).
			TextColor(t.TextMuted).Selectable()
	})
}

// hkBadge 是一个状态徽标：启用取强调色（浅底 + 强调边），禁用取中性面。
func hkBadge(c *gui.Context, t *gui.Theme, text string, on bool) {
	ink := t.TextMuted
	b := gui.Row(c).Shrink(0).AlignItems(gui.Center).Padding(hkBadgeV, hkBadgeH).
		Radius(hkBadgeR).Label(text).Background(t.SurfaceHover)
	b.BorderWidth(strokeHairline).BorderColor(t.Border)
	if on {
		ink = colorAccentInk.in(t)
		b = b.Background(t.Accent.Alpha(hkAlphaBadge)).
			BorderColor(t.Accent.Alpha(hkAlphaBorder))
	}
	b.Children(func() {
		hkBadgeStyle.style(t, gui.Text(c, text)).TextColor(ink).SingleLine()
	})
}

// hkSourceState 是来源状态（正常 / 缺失）。
func hkSourceState(s *hooksSource) string {
	if s.missing {
		return hkCopyMissing
	}
	return hkCopyNormal
}

// hkEnabledLabel 是启用状态的界面语。
func hkEnabledLabel(on bool) string {
	if on {
		return hkCopyEnabled
	}
	return hkCopyDisabled
}

// hkUserDoc 把用户条目拼成编辑框的初始文档（只有事实给了条目才有内容）。
func hkUserDoc(entries []hooksEntry) string {
	doc := map[string][]map[string]any{hkCopyHooksKey: {}}
	for _, e := range entries {
		item := map[string]any{"event": e.event, "command": e.command}
		if e.matcher != "" {
			item["matcher"] = e.matcher
		}
		if e.timeout > 0 {
			item["timeout"] = e.timeout
		}
		if e.async {
			item["async"] = true
		}
		doc[hkCopyHooksKey] = append(doc[hkCopyHooksKey], item)
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

// hkEditorDialog 是用户钩子文档编辑框（S-hooks-23…26）。
func hkEditorDialog(c *gui.Context, t *gui.Theme) {
	if !hooksScreen.editorOpen {
		return
	}
	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(hkRowGap).
			Role(gui.RoleDialog).Label(hkCopyDocTitle)
		panel.Children(func() {
			typeH3.style(t, gui.Text(c, hkCopyDocTitle)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(3).SingleLine()
			screenHint(c, hkCopyDocDesc)
			editor := gui.TextArea(c, &hooksScreen.draft).MinHeight(hkEditorMinH).
				Label(hkCopyDocTitle)
			editor.BorderWidth(strokeHairline).BorderColor(t.Border)
			if hooksScreen.saveFailure != "" {
				hkAlertLine(c, t, hooksScreen.saveFailure)
			}
			gui.Row(c).Justify(gui.End).Gap(hkChipGap).Children(func() {
				if screenButton(c, hkCopyCancel, hkCopyCancel, "", false, rowRegular) {
					hooksScreen.editorOpen = false
					hooksScreen.saveFailure = ""
					open = false
				}
				if screenButton(c, hkCopySave, hkCopySave, "", true, rowRegular) {
					if err := hkCheckDocObject(hooksScreen.draft); err != nil {
						hooksScreen.saveFailure = hkCopyBadJSON
					} else {
						// 写回需要 hooks.save 的 Intents：保存只在界面里收尾（未接动作）。
						hooksScreen.saveFailure = ""
						hooksScreen.editorOpen = false
						open = false
					}
				}
			})
		})
		panel.Width(accDialogWidth(c))
	}))

	if !open || m.Dismissed() {
		hooksScreen.editorOpen = false
	}
}

// hkCheckDocObject 判文档是不是一个 JSON 对象。
func hkCheckDocObject(doc string) error {
	var obj map[string]json.RawMessage
	return json.Unmarshal([]byte(doc), &obj)
}
