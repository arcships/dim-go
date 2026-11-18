package ui

import (
	"fmt"
	"time"

	gui "github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// 本文件是设置 › 关于的重建实现：元素按
// docs/domains/requirements/inventory-settings-rt.md §19（S-about-01…38，
// 38 个元素）逐行对应，文案逐字取参考产品中文，数值取
// docs/domains/requirements/design-baseline.md：
//
//	标签 13/600、版本值 12 等宽、状态行 12.25、公告条 56 高、WIP 行 58 高、
//	卡片圆角 8、卡内边距 11/14、徽标 20 高、按钮 28。
//
// 本页没有页面标题（参考模板首节即「软件更新」），所以第一块内容就是它。
//
// 只保留可丢的界面状态（aboutScreen）：更新状态机、公告展开项、开发态调试
// 计数。更新事实（host 的 Updater）与公告事实（服务端）都没有接进 Source：
// 当前停在参考产品的运行时状态 —— 自动更新已禁用、2 条公告、8 条 WIP 能力、
// 开发态调试条。按钮照渲染、点击只改本屏状态。
//
// # 未实现分支
//
//	S-about-03…10、12…18  更新状态机的其它相位（新版本 / 检查中 / 下载中 /
//	                有新版本可用 / 已下载 / 正在准备 / 出错）：界面按相位都实现
//	                了，但没有任何来源能推动相位变化（Source 没有更新事实、
//	                Intents 没有检查与下载），首屏停在「自动更新已禁用」。
//	S-about-26      公告正文：数据层只有摘要，展开时把摘要按 Markdown 渲染
//	                （不编造正文）。
//	S-about-27      「暂无公告」空态：当前 2 条公告，条件不成立；已实现。

// aboutUpdatePhase 是软件更新的相位（参考 update 状态机的界面分支）。
type aboutUpdatePhase int

const (
	abtDisabled aboutUpdatePhase = iota
	abtIdle
	abtChecking
	abtDownloading
	abtAvailable
	abtDownloaded
	abtPreparing
	abtError
)

// aboutAnnouncement 是一条公告：标题、日期、摘要与正文（正文可缺）。
type aboutAnnouncement struct {
	id      string
	title   string
	date    string
	summary string
	body    string
}

// aboutCapability 是一条 WIP 能力：名称与一行原因。
type aboutCapability struct {
	label  string
	reason string
}

// aboutState 是本屏可丢的界面状态：不是事实。
type aboutState struct {
	phase           aboutUpdatePhase
	currentVersion  string
	updateVersion   string
	updateMessage   string
	downloadPercent int
	updateError     string

	announcements        []aboutAnnouncement
	announcementsLoaded  bool
	announcementsLoading bool
	expanded             string

	capabilities []aboutCapability

	isDev      bool
	debugIndex int
}

// aboutDebugStates 是开发态调试循环的相位名（参考的 debugStateLabel）。
var aboutDebugStates = []string{
	"disabled", "idle", "checking", "downloading",
	"available", "downloaded", "preparing", "error",
}

// aboutScreen 的首屏取值与参考产品运行时一致。
var aboutScreen = aboutState{
	phase:          abtDisabled,
	currentVersion: "v0.9.57",
	announcements: []aboutAnnouncement{
		{
			id: "desktop-0-9-65", title: "DimAgent Desktop 0.9.65 / CLI 0.5.28",
			date: "2026年11月16日", summary: "DimAgent Desktop 0.9.65 / CLI 0.5.28",
		},
		{
			id: "desktop-0-9-26-fix", title: "0.9.26更新失败解决方案",
			date: "2026年9月15日", summary: "0.9.26更新失败解决方案",
		},
	},
	announcementsLoaded: true,
	capabilities: []aboutCapability{
		{"文件历史与回退", "上游不处理文件级历史；会话回退使用 session.fork。"},
		{"删除分支", "App Server 没有删除分支的方法。"},
		{"重命名分支", "只有客户端本地显示名，没有跨客户端的分支元数据。"},
		{"分支独立模型", "session.model.update 仅支持 main；子分支继承切出时的配置。"},
		{"跨客户端归档", "App Server 没有归档事实；只能保留 Desktop 本地隐藏列表。"},
		{"MCP 交互询问", "MCP elicitation 尚未实现；模型提问使用 Questions 的 ask_user。"},
		{"全局用量账本", "目前只有按会话的模型/用量；聚合结果不完整且不是权威数据。"},
		{"远程附件读取", "只有本地客户端可以读取 attachment.put 返回的路径。"},
	},
	isDev: true,
}

// abtMarkdown 是公告正文的渲染配置：一份实例跨帧复用（解析缓存随它）。
var abtMarkdown markdown.Options

// 本屏几何：参考的 Tailwind 量换算成 design-baseline §1.1 的档位。
const (
	abtSectionGap = 28   // space-y-8：分节之间（分隔线两侧同值）
	abtLabelGap   = 21   // 分节标签与其内容之间
	abtStackGap   = 28   // 软件更新里各行之间（实测 31 / 32）
	abtListGap    = 8    // 公告条之间、WIP 行之间
	abtRowPadV    = 10   // 公告条的上下内边距
	abtRowPadH    = 14   // 公告条与 WIP 行的左右内边距
	abtRowH       = 56   // 公告条高
	abtCapH       = 58   // WIP 行高
	abtCardR      = 8.75 // 卡片圆角
	abtTextGap    = 3.5  // 标题与摘要之间
	abtRowGap     = 10.5 // 标题与日期之间
	abtValueGap   = 6    // 「当前版本:」与版本号之间
	abtBtnH       = 28
	abtIconInline = 14
	abtIconBlock  = 16
	abtChevron    = 14
	abtBadgePadV  = 3.5
	abtBadgePadH  = 7
	abtBadgeR     = 4.75
	abtCodePadV   = 3
	abtCodePadH   = 7
	abtLPadH      = 11
	abtGroupPadV  = 14
	abtGroupPadH  = 15
	abtAlpha12    = 0.12
	abtDebugGap   = 10.5
)

// 字阶（design-baseline §2）。
var (
	abtTitleStyle = fontStep{size: 17.5, line: 24.5, weight: 600}
	abtLabelStyle = fontStep{size: 13, line: 19.5, weight: 600}
	abtValueStyle = fontStep{size: 12.25, line: 17.5, weight: 400}
	abtMonoStyle  = fontStep{size: 12, line: 17, weight: 400, family: codeFont}
	abtNoteStyle  = fontStep{size: 12.25, line: 17.5, weight: 400}
	abtSmallStyle = fontStep{size: 11, line: 14, weight: 400}
	abtWipStyle   = fontStep{size: 10.5, line: 14, weight: 500}
)

// 本屏图标：Heroicons v2 outline（24 网格、1.5 描边、MIT）的内联路径。
var (
	abtIconDownload = mustSVG(`<path d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3"/>`)
	abtIconBug      = mustSVG(`<path d="M12 12.75a3 3 0 0 0 3-3v-3a3 3 0 0 0-6 0v3a3 3 0 0 0 3 3Zm0 0v8.25m-7.5-3.75h3m9 0h3m-15-6h3m9 0h3M4.5 7.5l3 1.5m9-1.5-3 1.5M4.5 16.5l3-1.5m9 1.5-3-1.5"/>`)
)

// 本屏文案：逐字取自参考产品中文（英文原文的两条保持英文）。
const (
	abtCopyUpdate         = "软件更新"
	abtCopyCurrent        = "当前版本:"
	abtCopyNew            = "新版本:"
	abtCopyLatest         = "已是最新版本"
	abtCopyChecking       = "检查中..."
	abtCopyDownloading    = "下载中..."
	abtCopyAvailable      = "有新版本可用"
	abtCopyDownloaded     = "更新已下载"
	abtCopyPreparing      = "正在准备…"
	abtCopyFailed         = "检查更新失败"
	abtCopyDisabled       = "自动更新已禁用"
	abtCopyCheck          = "检查更新"
	abtCopyRetry          = "重试"
	abtCopyDownload       = "下载新版本"
	abtCopyInstall        = "重启安装"
	abtCopyAnnouncements  = "公告"
	abtCopyRefresh        = "刷新"
	abtCopyMore           = "查看更多公告"
	abtCopyEmpty          = "暂无公告"
	abtCopyWIP            = "暂不可用"
	abtCopyWIPDesc        = "以下能力仍在 WIP 跟踪中；入口保持可见但禁用，直到 App Server 路径完成。"
	abtCopyWIPBadge       = "WIP"
	abtCopyWIPBadgeLabel  = "WIP，未完成"
	abtCopyDebugTitle     = "Update state debugger"
	abtCopyDebugDesc      = "Development only. Cycles local update UI states without downloading or installing."
	abtCopyDebugNext      = "Next state"
	abtCopyAnnouncePrefix = "公告："
	abtCopyExpand         = "展开"
	abtCopyCollapse       = "收起"
	abtCopyDownloadPct    = "%d%%"
	abtCopyNoTitle        = "无标题"
)

// renderAbout 是设置 › 关于这一屏（设置壳的一个分节）。
func renderAbout(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(abtSectionGap).Children(func() {
				abtUpdateSection(c, t)
				screenRule(c)
				abtAnnouncementSection(c, t)
				abtCapabilitySection(c, t)
				if aboutScreen.isDev {
					abtDebugSection(c, t)
				}
			})
		})
	})
}

// abtUpdateSection 是「软件更新」（S-about-01…18）：标签、版本行、状态行、
// 按相位给的按钮与错误行。
func abtUpdateSection(c *gui.Context, t *gui.Theme) {
	gui.Column(c).Shrink(0).Gap(abtStackGap).Children(func() {
		abtLabelStyle.style(t, gui.Text(c, abtCopyUpdate)).TextColor(t.Text).
			Role(gui.RoleHeading).Level(2).Label(abtCopyUpdate)
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(abtValueGap).Children(func() {
			abtValueStyle.style(t, gui.Text(c, abtCopyCurrent)).TextColor(t.Text)
			abtMonoStyle.style(t, gui.Text(c, dashIfEmpty(aboutScreen.currentVersion))).
				TextColor(t.TextMuted).Selectable().Label("版本号")
		})
		if aboutScreen.updateVersion != "" {
			gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(abtValueGap).Children(func() {
				abtValueStyle.style(t, gui.Text(c, abtCopyNew)).TextColor(t.Text)
				abtMonoStyle.style(t, gui.Text(c, aboutScreen.updateVersion)).
					TextColor(t.TextMuted).Selectable()
			})
		}
		abtUpdateStatus(c, t)
		abtUpdateAction(c, t)
		if aboutScreen.updateError != "" {
			abtNoteStyle.style(t, gui.Text(c, aboutScreen.updateError)).
				TextColor(t.Danger).Role(gui.RoleAlert)
		}
	})
}

// abtUpdateStatus 是更新状态行（S-about-04…11）。
func abtUpdateStatus(c *gui.Context, t *gui.Theme) {
	text, ink := abtCopyDisabled, t.TextMuted
	icon := (*gui.SVG)(nil)
	spin := false
	switch aboutScreen.phase {
	case abtIdle:
		text, ink, icon = abtCopyLatest, t.Success, iconCheck
	case abtChecking:
		text, ink, icon, spin = abtCopyChecking, t.TextMuted, screenIconRefresh, true
	case abtDownloading:
		text, ink, icon = abtCopyDownloading, t.TextMuted, abtIconDownload
	case abtAvailable:
		text, ink, icon = abtCopyAvailable, colorAccentInk.in(t), abtIconDownload
		if aboutScreen.updateMessage != "" {
			text = aboutScreen.updateMessage
		}
	case abtDownloaded:
		text, ink, icon = abtCopyDownloaded, t.Success, iconCheck
	case abtPreparing:
		text, ink, icon, spin = abtCopyPreparing, t.TextMuted, screenIconRefresh, true
	case abtError:
		text, ink = abtCopyFailed, t.Danger
		if aboutScreen.updateMessage != "" {
			text = aboutScreen.updateMessage
		}
	}
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(abtValueGap).Children(func() {
		if icon != nil {
			box := gui.Box(c).Width(abtIconInline).Height(abtIconInline).Shrink(0).
				Label("图标：更新状态").AccessHidden(true)
			if spin {
				box = box.Rotate(box.Loop("abt-spin", time.Second, gui.Linear) * 360)
			}
			box.DrawOver(func(p *gui.Painter, r gui.Rect) {
				p.Icon(icon, r, ink)
			})
		}
		abtNoteStyle.style(t, gui.Text(c, text)).TextColor(ink).Label(text)
	})
}

// abtUpdateAction 是按相位给的更新按钮（S-about-12…17）。
func abtUpdateAction(c *gui.Context, t *gui.Theme) {
	var label string
	primary, disabled := false, false
	switch aboutScreen.phase {
	case abtIdle, abtError:
		label = abtCopyCheck
		if aboutScreen.phase == abtError {
			label = abtCopyRetry
		}
	case abtChecking:
		label, disabled = abtCopyChecking, true
	case abtDownloading:
		label = fmt.Sprintf(abtCopyDownloadPct, aboutScreen.downloadPercent)
		disabled = true
	case abtAvailable:
		label, primary = abtCopyDownload, true
		if aboutScreen.updateMessage != "" {
			label = aboutScreen.updateMessage
		}
	case abtDownloaded:
		label, primary = abtCopyInstall, true
	case abtPreparing:
		label, disabled = abtCopyPreparing, true
	default:
		// 自动更新已禁用：该态下不渲染检查按钮（R-ABT-084）。
		return
	}
	gui.Row(c).Shrink(0).Children(func() {
		b := gui.ButtonBase(c).Shrink(0).Height(abtBtnH).
			Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
			Label(label).Disabled(disabled)
		if primary {
			b = b.Background(t.Accent)
		} else {
			b = b.Background(t.Background)
			b.BorderWidth(strokeHairline).BorderColor(t.Border)
		}
		ink := t.Text
		switch {
		case disabled:
			b = b.Opacity(abtAlpha50)
		case primary && b.Pressed():
			b = b.Background(t.AccentPressed)
		case primary && b.Hovered():
			b = b.Background(t.AccentHover)
		case b.Pressed():
			b = b.Background(t.SurfacePressed)
		case b.Hovered():
			b = b.Background(t.SurfaceHover)
		}
		if primary {
			ink = t.AccentText
		}
		b.Children(func() {
			abtValueStyle.style(t, gui.Text(c, label)).TextColor(ink).SingleLine()
		})
		if b.Clicked() && !disabled {
			abtAdvanceUpdate()
		}
	})
}

// abtAdvanceUpdate 是本屏内的相位推进：没有事实来源，只走界面（见文件头）。
func abtAdvanceUpdate() {
	switch aboutScreen.phase {
	case abtIdle, abtError:
		aboutScreen.phase = abtChecking
	case abtAvailable:
		aboutScreen.phase = abtDownloading
		aboutScreen.downloadPercent = 0
	case abtDownloaded:
		aboutScreen.phase = abtPreparing
	}
}

// abtAlpha50 是禁用态的整棵子树不透明度。
const abtAlpha50 = 0.5

// abtAnnouncementSection 是「公告」（S-about-19…28）。
func abtAnnouncementSection(c *gui.Context, t *gui.Theme) {
	if aboutScreen.announcementsLoading {
		abtNoteStyle.style(t, gui.Text(c, abtCopyChecking)).TextColor(t.TextMuted)
		return
	}
	gui.Column(c).Shrink(0).Gap(abtLabelGap).Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Children(func() {
			abtLabelStyle.style(t, gui.Text(c, abtCopyAnnouncements)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(2).Label(abtCopyAnnouncements)
			gui.Spacer(c)
			if accIconButton(c, accIconSpec{
				icon: accIconRefresh, label: abtCopyRefresh, tip: abtCopyRefresh,
				size: abtIconInline, hit: rowRegular,
			}) {
				// 重拉公告需要服务端事实：不动作（未接动作）。
			}
		})
		if len(aboutScreen.announcements) == 0 {
			if aboutScreen.announcementsLoaded {
				abtNoteStyle.style(t, gui.Text(c, abtCopyEmpty)).
					TextColor(t.TextMuted).TextAlign(gui.Center)
			}
		} else {
			abtAnnouncementList(c, t)
		}
		gui.Row(c).Shrink(0).Justify(gui.End).Children(func() {
			if screenAction(c, accIconExternal, "external-link", abtCopyMore, abtBtnH) {
				// 公告页地址没有事实来源：不动作（未接动作）。
			}
		})
	})
}

// abtAnnouncementList 是公告列表：一张卡、条间 1px 细线。
func abtAnnouncementList(c *gui.Context, t *gui.Theme) {
	screenCard(c, func() {
		gui.Column(c).Shrink(0).Children(func() {
			for i := range aboutScreen.announcements {
				if i > 0 {
					screenRule(c)
				}
				abtAnnouncementRow(c, t, &aboutScreen.announcements[i])
			}
		})
	})
}

// abtAnnouncementRow 是一条公告（S-about-21…26）：标题、日期、摘要与展开正文。
func abtAnnouncementRow(c *gui.Context, t *gui.Theme, a *aboutAnnouncement) {
	expanded := aboutScreen.expanded == a.id
	title := a.title
	if title == "" {
		title = abtCopyNoTitle
	}
	head := gui.ButtonBase(c).Shrink(0).MinHeight(abtRowH).
		Padding(abtRowPadV, abtRowPadH).Justify(gui.Start).Label(abtCopyAnnouncePrefix + title)
	switch {
	case head.Pressed():
		head = head.Background(t.SurfacePressed)
	case head.Hovered():
		head = head.Background(t.SurfaceHover)
	}
	head.Children(func() {
		gui.Row(c).Grow(1).MinWidth(0).AlignItems(gui.Center).Gap(abtRowGap).Children(func() {
			gui.Column(c).Grow(1).MinWidth(0).Gap(abtTextGap).Children(func() {
				gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(abtRowGap).Children(func() {
					abtLabelStyle.style(t, gui.Text(c, title)).TextColor(t.Text).
						SingleLine()
					gui.Spacer(c)
					if a.date != "" {
						abtSmallStyle.style(t, gui.Text(c, a.date)).
							TextColor(t.TextMuted).Label("发布 " + a.date)
					}
				})
				if a.summary != "" {
					abtNoteStyle.style(t, gui.Text(c, a.summary)).
						TextColor(t.TextMuted).SingleLine().Selectable()
				}
			})
			screenIconRotated(c, screenIconChevronDown, abtChevron, t.TextMuted,
				abtCopyExpand, abtChevronOpen(expanded))
		})
	})
	if head.Clicked() {
		if expanded {
			aboutScreen.expanded = ""
		} else {
			aboutScreen.expanded = a.id
		}
	}
	if !expanded || a.body == "" {
		return
	}
	gui.Column(c).Shrink(0).Padding(0, abtRowPadH).Children(func() {
		abtMarkdown.Theme = markdownTheme(t)
		markdown.View(c, a.body, &abtMarkdown)
	})
}

// abtChevronOpen 是折叠箭头的旋转角：展开时朝上。
func abtChevronOpen(expanded bool) float32 {
	if expanded {
		return abtChevronFlip
	}
	return 0
}

// abtChevronFlip 是展开态箭头旋转 180°。
const abtChevronFlip = 180

// abtCapabilitySection 是「暂不可用」（S-about-29…33）：说明 + 8 条 WIP 行。
func abtCapabilitySection(c *gui.Context, t *gui.Theme) {
	gui.Column(c).Shrink(0).Gap(abtCapabilityGap).Children(func() {
		gui.Column(c).Shrink(0).Gap(abtTextGap).Children(func() {
			abtLabelStyle.style(t, gui.Text(c, abtCopyWIP)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(2).Label(abtCopyWIP)
			abtNoteStyle.style(t, gui.Text(c, abtCopyWIPDesc)).
				TextColor(t.TextMuted).Selectable()
		})
		gui.Column(c).Shrink(0).Gap(abtListGap).Children(func() {
			for _, cap := range aboutScreen.capabilities {
				abtCapabilityRow(c, t, cap)
			}
		})
	})
}

// abtCapabilityGap 是「暂不可用」标签区与列表之间（实测 10.5）。
const abtCapabilityGap = 10.5

// abtCapabilityRow 是一条 WIP 能力（S-about-31/32/33）：名称、原因与 WIP 徽标。
func abtCapabilityRow(c *gui.Context, t *gui.Theme, cap aboutCapability) {
	card := accCardFrame(c, gui.Row(c).Shrink(0).MinHeight(abtCapH).
		AlignItems(gui.Center).Gap(abtRowGap).
		Padding(abtRowPadV, abtLPadH))
	card.Label(cap.label)
	card.Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Gap(abtTextGap).Children(func() {
			abtLabelStyle.style(t, gui.Text(c, cap.label)).TextColor(t.Text).
				SingleLine()
			if cap.reason != "" {
				abtSmallStyle.style(t, gui.Text(c, cap.reason)).
					TextColor(t.TextMuted).SingleLine().Selectable()
			}
		})
		abtWipBadge(c, t)
	})
}

// abtWipBadge 是「WIP」徽标（S-about-33）：警示底色、小圆角。
func abtWipBadge(c *gui.Context, t *gui.Theme) {
	b := gui.Row(c).Shrink(0).AlignItems(gui.Center).
		Padding(abtBadgePadV, abtBadgePadH).Radius(abtBadgeR).
		Background(t.Warning.Alpha(abtAlpha12)).Label(abtCopyWIPBadgeLabel)
	b.Children(func() {
		abtWipStyle.style(t, gui.Text(c, abtCopyWIPBadge)).TextColor(t.Warning).
			SingleLine()
	})
}

// abtDebugSection 是开发态调试条（S-about-34…38）：虚线框、虫子图标、
// 当前相位与「Next state」。
func abtDebugSection(c *gui.Context, t *gui.Theme) {
	box := gui.Column(c).Shrink(0).Gap(abtDebugGap).
		Padding(abtGroupPadV, abtGroupPadH).Radius(abtCardR).
		Background(t.Background).Label(abtCopyDebugTitle)
	box.BorderWidth(strokeHairline).BorderColor(t.Border)
	box.BorderStyle(gui.BorderDashed)
	box.Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Start).Gap(abtRowGap).Children(func() {
			accIcon(c, abtIconBug, abtIconBlock, t.TextMuted, "bug")
			gui.Column(c).Grow(1).MinWidth(0).Gap(abtTextGap).Children(func() {
				abtLabelStyle.style(t, gui.Text(c, abtCopyDebugTitle)).
					TextColor(t.Text).Label(abtCopyDebugTitle)
				abtSmallStyle.style(t, gui.Text(c, abtCopyDebugDesc)).
					TextColor(t.TextMuted).Selectable()
			})
		})
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Children(func() {
			chip := gui.Row(c).Shrink(0).AlignItems(gui.Center).
				Padding(abtCodePadV, abtCodePadH).Radius(abtBadgeR).
				Background(t.SurfaceHover).Label("调试相位")
			chip.Children(func() {
				abtSmallStyle.style(t, gui.Text(c, aboutDebugLabel())).
					TextColor(t.TextMuted).SingleLine().Selectable()
			})
			gui.Spacer(c)
			if screenButton(c, abtCopyDebugNext, abtCopyDebugNext, "", false, abtBtnH) {
				aboutScreen.debugIndex =
					(aboutScreen.debugIndex + 1) % len(aboutDebugStates)
				aboutScreen.phase = aboutUpdatePhase(
					(aboutScreen.debugIndex + len(aboutDebugStates) - 1) %
						len(aboutDebugStates))
			}
		})
	})
}

// aboutDebugLabel 是调试条上的当前相位名。
func aboutDebugLabel() string {
	if aboutScreen.debugIndex < 0 || aboutScreen.debugIndex >= len(aboutDebugStates) {
		return aboutDebugStates[0]
	}
	return aboutDebugStates[aboutScreen.debugIndex]
}
