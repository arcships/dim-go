package ui

import (
	"encoding/json"

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
// 当前可见分支是 App Server 目录（HooksAppServerCatalog）：2 个来源（用户 /
// 项目），条目 2 组均为空。屏内只保留**可丢的界面状态**（hooksScreen）：来源
// 与条目的读数、编辑框草稿、加载与失败文本。写事实（hooks.save、
// hooks.set_enabled）还没有对应的 Intents 方法，控件照渲染、点击只改本屏状态。
//
// # 未实现分支
//
//	S-hooks-27…32  非 App Server 分支（本地 config 页、来自配置/工作区/插件三组、
//	               本地钩子编辑对话框）：本客户端恒为 App Server 宿主界面，
//	               该分支不可达（同一判断见清单 §15.2）。
//	S-hooks-11     插件来源的 Lock 图标：当前没有 plugin: 来源，条件不成立；
//	               形状已备（hkIconLock），接线后按 source 前缀显示。
//
// # 与参考的差异
//
//	标题字号取设计基线的 17.5（清单 R-HOK-002 写的 20 是 16px 根字号的换算值；
//	design-baseline §2 的页面标题是 17.5/600，与「账户」「归档会话」同档）。
//	「了解更多」没有外链地址（没有文档 URL 的事实来源），按钮照渲染、点击不动作。
//	项目来源的路径用 ~ 占位（参考截图里是采集机器的绝对路径，不写进仓库）。
//	条目行（S-hooks-20）按参考结构实现，当前两组皆空，故只走空态。

// hooksEntry 是一条钩子：事件、状态、异步标记、命令与三个可选修饰。
type hooksEntry struct {
	event   string
	enabled bool
	async   bool
	command string
	matcher string
	timeout string
	onFail  string
}

// hooksSource 是一个来源：名称、路径、状态、读写权限与条目。
type hooksSource struct {
	title    string
	// user 为真表示用户来源（可编辑、无路径）；否则是项目来源。
	user     bool
	path     string
	revision string
	enabled  bool
	readonly bool
	// plugin 为真时在来源图标右下角叠一个锁（S-hooks-11）。
	plugin bool
	// missing 为真表示来源文件缺失（状态标「缺失」）。
	missing bool
	message string
	entries []hooksEntry
}

// hooksState 是本屏可丢的界面状态：不是事实。
type hooksState struct {
	loading   bool
	loadError string
	// writeFailure 是启用开关失败的原因；saveFailure 是文档保存失败的原因。
	writeFailure string
	saveFailure  string
	// disableAll 表示用户文档关闭了所有钩子。
	disableAll bool
	sources    []hooksSource

	editorOpen bool
	draft      string
}

// hooksScreen 的首屏取值与参考产品运行时一致：用户来源正常且启用、项目来源
// 缺失且禁用、两组条目均为空、修订号各 1。
var hooksScreen = hooksState{
	sources: []hooksSource{
		{
			title: "用户", user: true, revision: "native:user/hooks:1",
			enabled: true, readonly: true,
		},
		{
			title: "项目", path: "~/.dimcode/hooks.json",
			revision: "native:user/hooks-projects:1",
			enabled:  false, readonly: true, missing: true,
			message: "项目钩子文档尚未创建，启用项目钩子后会写入该文件。",
		},
	},
}

// 本屏几何：参考的 Tailwind 量换算成 design-baseline §1.1 的档位。
const (
	hkPageGap   = 28   // space-y-8：标题区与内容之间
	hkSectionGap = 21  // space-y-6：分组之间
	hkLabelGap  = 14   // 小标题与其列表之间
	hkCardGap   = 12   // 来源卡 / 条目组之间
	hkPadV      = 14   // p-4 上下
	hkPadH      = 14   // p-4 左右
	hkRowGap    = 10.5 // gap-3：图标与文字块之间
	hkChipGap   = 7    // gap-2：名称 / 状态 / 徽标之间
	hkTightGap  = 2    // 名称与状态徽标的上下贴合
	hkTextGap   = 3.5  // 名称与路径之间

	hkIconBlock  = 32 // h-8 w-8：来源图标块
	hkIconInline = 16 // h-4 w-4：来源图标
	hkIconMicro  = 14 // h-3.5：警示与按钮图标
	hkIconHit    = 28 // 图标按钮 28×28
	hkIconSize   = 14

	hkBtnH    = 28 // h-7：了解更多 / 编辑
	hkBadgeV  = 3.5
	hkBadgeH  = 7
	hkBadgeR  = 4.75
	hkRuleRow = 1

	hkBannerPadV  = 7   // 提示条上下
	hkBannerPadH  = 14  // 提示条左右
	hkBannerR     = 6   // rounded-md
	hkEmptyPadV   = 9   // 空态行的上下
	hkGroupPadV   = 9.5 // 组头的上下
	hkEntryH      = 44  // 条目行高
	hkEntryPadV   = 7
	hkDialogW     = 448
	hkEditorMinH  = 180
	hkDraftLines  = 8
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
	hkCopyLoading    = "正在加载..."
	hkCopyAllOff     = "用户文档已关闭所有钩子。"
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
)

// renderHooks 是设置 › Hooks 这一屏（设置壳的一个分节）。
func renderHooks(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(hkSectionGap).Children(func() {
				hkHeader(c, t)
				if hooksScreen.loadError != "" {
					hkAlertLine(c, t, hooksScreen.loadError)
				} else if hooksScreen.loading {
					hkNoteLine(c, t, hkCopyLoading)
				}
				if hooksScreen.disableAll {
					hkBanner(c, t, hkCopyAllOff, false)
				}
				hkBanner(c, t, hkCopyWarning, true)
				gui.Column(c).Shrink(0).Gap(hkLabelGap).Children(func() {
					screenGroupTitle(c, hkCopySources)
					for i := range hooksScreen.sources {
						hkSourceCard(c, t, &hooksScreen.sources[i])
					}
				})
				if hooksScreen.writeFailure != "" {
					hkAlertLine(c, t, hooksScreen.writeFailure)
				}
				gui.Column(c).Shrink(0).Gap(hkLabelGap).Children(func() {
					screenGroupTitle(c, hkCopyEntries)
					for i := range hooksScreen.sources {
						hkEntryGroup(c, t, &hooksScreen.sources[i])
					}
				})
				if hooksScreen.saveFailure != "" {
					hkAlertLine(c, t, hooksScreen.saveFailure)
				}
				hkRevisions(c, t)
			})
		})
	})
	hkEditorDialog(c, t)
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
func hkSourceCard(c *gui.Context, t *gui.Theme, s *hooksSource) {
	card := accCardFrame(c, gui.Row(c).Shrink(0).AlignItems(gui.Start).
		Gap(hkRowGap).Padding(hkPadV, hkPadH))
	card.Label(s.title)
	card.Children(func() {
		hkSourceIcon(c, t, s)
		gui.Column(c).Grow(1).MinWidth(0).Gap(hkTightGap).Children(func() {
			gui.Row(c).AlignItems(gui.Center).Gap(hkChipGap).Children(func() {
				hkNameStyle.style(t, gui.Text(c, s.title)).TextColor(t.Text).SingleLine()
				hkStatusStyle.style(t, gui.Text(c, hkSourceState(s))).TextColor(t.TextMuted)
				hkBadge(c, t, hkEnabledLabel(s.enabled), s.enabled)
				if s.readonly {
					hkBadge(c, t, hkCopyReadonly, false)
				}
			})
			if s.path != "" {
				hkPathStyle.style(t, gui.Text(c, s.path)).TextColor(t.TextMuted).
					SingleLine().Label("路径：" + s.path)
			}
			if s.message != "" {
				hkNoteLine(c, t, s.message)
			}
		})
		if s.user {
			if screenAction(c, iconPencil, "pencil", hkCopyEdit, hkBtnH) {
				hooksScreen.draft = hkUserDoc(hooksScreen.sources[0].entries)
				hooksScreen.saveFailure = ""
				hooksScreen.editorOpen = true
			}
			return
		}
		hkProjectToggle(c, t, s)
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
		p.Icon(hkIconLock, r, t.TextMuted)
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
	if e.timeout != "" {
		parts = append(parts, "超时 "+e.timeout)
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
func hkRevisions(c *gui.Context, t *gui.Theme) {
	user, project := "", ""
	if len(hooksScreen.sources) > 1 {
		user, project = hooksScreen.sources[0].revision, hooksScreen.sources[1].revision
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
		if e.timeout != "" {
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
	m := gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(hkRowGap)
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
	})
	if !open || m.Dismissed() {
		hooksScreen.editorOpen = false
	}
}

// hkCheckDocObject 判文档是不是一个 JSON 对象。
func hkCheckDocObject(doc string) error {
	var obj map[string]json.RawMessage
	return json.Unmarshal([]byte(doc), &obj)
}
