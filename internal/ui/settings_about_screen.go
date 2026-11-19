package ui

import (
	"context"
	"fmt"
	"strings"
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
// 只保留可丢的界面状态（aboutScreen）：更新源草稿、公告展开项、开发态调试
// 计数。更新事实来自宿主的自动更新编排（可选接口 UpdateFacts，适配器在
// internal/uiwire）：未配置更新源时显示「未配置更新源」并禁用检查入口，不假装
// 已是最新；配置后按检查 / 下载 / 已下载 / 出错相位渲染。当前版本与公告标题取
// 本次构建版本（SetBuildVersion），不写样例号。
//
// 更新相位与入口：
//
//	disabled    未配置更新源（或无更新能力）：检查按钮置灰；
//	idle        已配置、未检查：检查更新；
//	checking    检查中：按钮禁用；
//	available   有新版本可用：下载新版本；
//	downloading 下载中：按钮显示百分比并禁用；
//	downloaded  已下载：重启安装（宿主没有安装能力时置灰并写原因）；
//	preparing   正在准备：按钮禁用；
//	error       检查 / 下载失败：重试（下载失败重试下载，否则重试检查）。
//
// # 未实现分支
//
//	S-about-26      公告正文：数据层只有摘要，展开时把摘要按 Markdown 渲染
//	                （不编造正文）。
//	S-about-27      「暂无公告」空态：当前 1 条构建公告，条件不成立；已实现。

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
	downloadPercent int
	updateError     string

	announcements        []aboutAnnouncement
	announcementsLoaded  bool
	announcementsLoading bool
	expanded             string

	capabilities []aboutCapability

	isDev      bool
	debugIndex int

	// 更新源草稿：只在用户改过之后覆盖读数里的地址；保存失败的原因留在
	// feedError 里，不改读数。
	feedDraft string
	feedDirty bool
	feedError string
}

// aboutDebugStates 是开发态调试循环的相位名（参考的 debugStateLabel）。
var aboutDebugStates = []string{
	"disabled", "idle", "checking", "downloading",
	"available", "downloaded", "preparing", "error",
}

// defaultBuildVersion 是没注入构建版本时的兜底：与入口的 defaultVersion 同值。
const defaultBuildVersion = "0.1.0"

// buildVersion 是本次运行的构建版本（形如 x.y.z）。
//
// 入口在打包时用 `-ldflags -X main.version=` 注入，再在创建视图前交给
// SetBuildVersion：关于页不写死参考产品的版本号，写死就成了假事实。
var buildVersion = defaultBuildVersion

// SetBuildVersion 设置本次运行的构建版本。入口在 NewApp 之前调用一次。
//
// 关于页的「当前版本」与公告标题都取它：界面显示的是这次运行的实际构建，
// 不是参考产品的样例值。
func SetBuildVersion(v string) {
	v = strings.TrimSpace(v)
	if v == "" {
		return
	}
	buildVersion = v
	aboutScreen.currentVersion = "v" + v
	aboutScreen.announcements = buildAnnouncements()
}

// buildAnnouncements 是公告的本地条目：服务端公告事实还没有接进 Source，
// 只给「这次运行的是什么构建」这一条真信息，不编造发布记录。
func buildAnnouncements() []aboutAnnouncement {
	return []aboutAnnouncement{{
		id:      "build-" + buildVersion,
		title:   "DimAgent Desktop " + buildVersion,
		summary: "当前运行的客户端构建版本 " + buildVersion + "；公告事实尚未接入服务端。",
	}}
}

// aboutScreen 的首屏取值只用可丢的界面状态与本次构建版本。
var aboutScreen = aboutState{
	phase:               abtDisabled,
	currentVersion:      "v" + buildVersion,
	announcements:       buildAnnouncements(),
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
	abtSectionGap = 21   // space-y-6：分节之间（分隔线两侧同值）
	abtLabelGap   = 21   // 分节标签与其内容之间
	abtStackGap   = 14   // 软件更新里各行之间（实测 31 / 32，行盒含在内）
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
	abtDebugGap   = 10.5
)

// 字阶（design-baseline §2）。
var (
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
	abtCopyUpdate            = "软件更新"
	abtCopyCurrent           = "当前版本:"
	abtCopyNew               = "新版本:"
	abtCopyLatest            = "已是最新版本"
	abtCopyChecking          = "检查中..."
	abtCopyDownloading       = "下载中..."
	abtCopyAvailable         = "有新版本可用"
	abtCopyDownloaded        = "更新已下载"
	abtCopyPreparing         = "正在准备…"
	abtCopyFailed            = "检查更新失败"
	abtCopyDisabled          = "自动更新已禁用"
	abtCopyCheck             = "检查更新"
	abtCopyRetry             = "重试"
	abtCopyDownload          = "下载新版本"
	abtCopyInstall           = "重启安装"
	abtCopyAnnouncements     = "公告"
	abtCopyRefresh           = "刷新"
	abtCopyMore              = "查看更多公告"
	abtCopyEmpty             = "暂无公告"
	abtCopyWIP               = "暂不可用"
	abtCopyWIPDesc           = "以下能力仍在 WIP 跟踪中；入口保持可见但禁用，直到 App Server 路径完成。"
	abtCopyWIPBadge          = "WIP"
	abtCopyWIPBadgeLabel     = "WIP，未完成"
	abtCopyDebugTitle        = "Update state debugger"
	abtCopyDebugDesc         = "Development only. Cycles local update UI states without downloading or installing."
	abtCopyDebugNext         = "Next state"
	abtCopyAnnouncePrefix    = "公告："
	abtCopyExpand            = "展开"
	abtCopyDownloadPct       = "%d%%"
	abtCopyNoTitle           = "无标题"
	abtCopyNoSource          = "未配置更新源"
	abtCopyNotChecked        = "尚未检查"
	abtCopySource            = "更新源"
	abtCopySourceLabel       = "更新源地址"
	abtCopySourceSave        = "保存更新源"
	abtCopySourceHint        = "留空则关闭自动更新；填基址时按「基址/latest.json」读取。"
	abtCopySourcePlaceholder = "https://example.com/updates"
	abtCopyNotes             = "发布说明"
)

// aboutWire 是本帧的更新接线：事实与动作都是可选接口，没有实现时本屏按
// 「自动更新已禁用」渲染。它只在渲染本屏时写入，不缓存读数。
var aboutWire struct {
	facts   UpdateFacts
	intents UpdateIntents
}

// aboutUpdateView 是本屏按一帧读数收出来的更新视图：相位、文案与按钮都用它。
type aboutUpdateView struct {
	wired         bool
	configured    bool
	phase         aboutUpdatePhase
	checked       bool
	available     string
	notes         string
	percent       int
	err           string
	retryDownload bool
	installable   bool
	installReason string
	feedURL       string
}

// renderAbout 是设置 › 关于这一屏（设置壳的一个分节）。
func renderAboutWith(c *gui.Context, source Source, intents Intents) {
	var facts UpdateFacts
	if source != nil {
		facts, _ = source.(UpdateFacts)
	}
	var updateIntents UpdateIntents
	if intents != nil {
		updateIntents, _ = intents.(UpdateIntents)
	}
	aboutWire.facts, aboutWire.intents = facts, updateIntents

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

// renderAbout 是设置壳的关于分节：把本帧的 Source 与 Intents 接给可选的
// 更新接口（未实现时本屏仍按「自动更新已禁用」渲染）。
func (a *App) renderAbout(c *gui.Context) {
	if a == nil {
		renderAboutWith(c, nil, nil)
		return
	}
	renderAboutWith(c, a.source, a.intents)
}

// aboutCurrentUpdate 读一帧更新事实；没有实现 UpdateFacts 时用本屏调试相位，
// 与样例运行一致（自动更新已禁用）。
func aboutCurrentUpdate() aboutUpdateView {
	if aboutWire.facts == nil {
		view := aboutUpdateView{
			phase:   aboutScreen.phase,
			percent: aboutScreen.downloadPercent,
			err:     aboutScreen.updateError,
		}
		if aboutScreen.updateVersion != "" {
			view.available = aboutScreen.updateVersion
		}
		return view
	}
	snapshot := aboutWire.facts.Update()
	view := aboutUpdateView{
		wired:         true,
		configured:    snapshot.Configured,
		available:     snapshot.AvailableVersion,
		notes:         snapshot.Notes,
		percent:       snapshot.Percent,
		err:           snapshot.Err,
		installable:   snapshot.Installable,
		installReason: snapshot.InstallReason,
		feedURL:       snapshot.FeedURL,
	}
	if !aboutScreen.feedDirty {
		aboutScreen.feedDraft = snapshot.FeedURL
	}
	if snapshot.ConfigError != "" {
		view.phase, view.err = abtError, snapshot.ConfigError
		return view
	}
	switch snapshot.Phase {
	case "idle":
		if snapshot.Configured {
			view.phase = abtIdle
		} else {
			view.phase = abtDisabled
		}
	case "checking":
		view.phase = abtChecking
	case "up_to_date":
		view.phase, view.checked = abtIdle, true
	case "available":
		view.phase = abtAvailable
	case "downloading":
		view.phase = abtDownloading
	case "downloaded":
		view.phase = abtDownloaded
	case "preparing":
		view.phase = abtPreparing
	case "error":
		view.phase = abtError
		view.retryDownload = snapshot.AvailableVersion != ""
	default:
		if snapshot.Configured {
			view.phase = abtIdle
		} else {
			view.phase = abtDisabled
		}
	}
	return view
}

// abtUpdateSection 是「软件更新」（S-about-01…18）：标签、版本行、状态行、
// 更新源、按相位给的按钮与错误行。
func abtUpdateSection(c *gui.Context, t *gui.Theme) {
	view := aboutCurrentUpdate()
	gui.Column(c).Shrink(0).Gap(abtStackGap).Children(func() {
		abtLabelStyle.style(t, gui.Text(c, abtCopyUpdate)).TextColor(t.Text).
			Role(gui.RoleHeading).Level(2).Label(abtCopyUpdate)
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(abtValueGap).Children(func() {
			abtValueStyle.style(t, gui.Text(c, abtCopyCurrent)).TextColor(t.Text)
			abtMonoStyle.style(t, gui.Text(c, dashIfEmpty(aboutScreen.currentVersion))).
				TextColor(t.TextMuted).Selectable().Label("版本号")
		})
		if view.available != "" {
			gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(abtValueGap).Children(func() {
				abtValueStyle.style(t, gui.Text(c, abtCopyNew)).TextColor(t.Text)
				abtMonoStyle.style(t, gui.Text(c, view.available)).
					TextColor(t.TextMuted).Selectable()
			})
		}
		abtUpdateStatus(c, t, view)
		abtUpdateAction(c, t, view)
		abtUpdateFeedRow(c, t, view)
		if view.phase == abtDownloaded && !view.installable && view.installReason != "" {
			abtSmallStyle.style(t, gui.Text(c, view.installReason)).
				TextColor(t.TextMuted).MaxLines(2)
		}
		if view.notes != "" && (view.phase == abtAvailable || view.phase == abtDownloaded) {
			abtSmallStyle.style(t, gui.Text(c, abtCopyNotes+": "+view.notes)).
				TextColor(t.TextMuted).MaxLines(3)
		}
		if view.phase == abtError && view.err != "" {
			abtNoteStyle.style(t, gui.Text(c, view.err)).
				TextColor(t.Danger).Role(gui.RoleAlert).MaxLines(3)
		}
	})
}

// abtUpdateStatus 是更新状态行（S-about-04…11）。
func abtUpdateStatus(c *gui.Context, t *gui.Theme, view aboutUpdateView) {
	text, ink := abtCopyDisabled, t.TextMuted
	icon := (*gui.SVG)(nil)
	spin := false
	switch view.phase {
	case abtIdle:
		if view.checked {
			text, ink, icon = abtCopyLatest, t.Success, iconCheck
		} else {
			text, ink, icon = abtCopyNotChecked, t.TextMuted, screenIconRefresh
		}
	case abtChecking:
		text, ink, icon, spin = abtCopyChecking, t.TextMuted, screenIconRefresh, true
	case abtDownloading:
		text, ink, icon = abtCopyDownloading, t.TextMuted, abtIconDownload
	case abtAvailable:
		text, ink, icon = abtCopyAvailable, colorAccentInk.in(t), abtIconDownload
	case abtDownloaded:
		text, ink, icon = abtCopyDownloaded, t.Success, iconCheck
	case abtPreparing:
		text, ink, icon, spin = abtCopyPreparing, t.TextMuted, screenIconRefresh, true
	case abtError:
		text, ink = abtCopyFailed, t.Danger
	default:
		if view.wired && !view.configured {
			text = abtCopyNoSource
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
				paintIcon(p, icon, r, ink)
			})
		}
		abtNoteStyle.style(t, gui.Text(c, text)).TextColor(ink).Label(text).MaxLines(2)
	})
}

// abtUpdateAction 是按相位给的更新按钮（S-about-12…17）。
func abtUpdateAction(c *gui.Context, t *gui.Theme, view aboutUpdateView) {
	label := abtCopyCheck
	primary, disabled := false, false
	action := abtActionCheck
	switch view.phase {
	case abtIdle:
		label, action = abtCopyCheck, abtActionCheck
	case abtChecking:
		label, disabled = abtCopyChecking, true
	case abtDownloading:
		label = abtCopyDownloading
		if view.percent > 0 {
			label = fmt.Sprintf(abtCopyDownloadPct, view.percent)
		}
		disabled = true
	case abtAvailable:
		label, primary, action = abtCopyDownload, true, abtActionDownload
	case abtDownloaded:
		label, primary, action = abtCopyInstall, true, abtActionInstall
		disabled = !view.installable
	case abtPreparing:
		label, disabled = abtCopyPreparing, true
	case abtError:
		label, action = abtCopyRetry, abtActionCheck
		if view.retryDownload {
			action = abtActionDownload
		}
	default:
		// 未配置更新源 / 自动更新已禁用：入口保留但置灰；状态行写明原因。
		label, disabled = abtCopyCheck, true
	}
	if aboutWire.intents == nil {
		disabled = true
	}
	gui.Row(c).Shrink(0).Children(func() {
		b := gui.ButtonBase(c).Shrink(0).Height(abtBtnH).
			Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
			Label(label).Disabled(disabled)
		if primary && !disabled {
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
		if primary && !disabled {
			ink = t.AccentText
		}
		b.Children(func() {
			abtValueStyle.style(t, gui.Text(c, label)).TextColor(ink).SingleLine()
		})
		if b.Clicked() && !disabled {
			abtRunUpdateAction(action)
		}
	})
}

// abtUpdateActionKind 是更新按钮对应的动作。
type abtUpdateActionKind int

const (
	abtActionCheck abtUpdateActionKind = iota
	abtActionDownload
	abtActionInstall
)

// abtRunUpdateAction 把按钮点击交给更新动作；动作在后台跑，失败原因写回相位。
func abtRunUpdateAction(action abtUpdateActionKind) {
	if aboutWire.intents == nil {
		return
	}
	ctx := context.Background()
	switch action {
	case abtActionCheck:
		_ = aboutWire.intents.CheckUpdate(ctx)
	case abtActionDownload:
		_ = aboutWire.intents.DownloadUpdate(ctx)
	case abtActionInstall:
		_ = aboutWire.intents.InstallUpdate(ctx)
	}
}

// abtUpdateFeedRow 是更新源：地址、保存与一行说明。地址为空时状态行显示
// 「未配置更新源」，检查入口置灰。
func abtUpdateFeedRow(c *gui.Context, t *gui.Theme, view aboutUpdateView) {
	if !view.wired {
		return
	}
	gui.Column(c).Shrink(0).Gap(abtTextGap).Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(abtValueGap).Children(func() {
			abtValueStyle.style(t, gui.Text(c, abtCopySource)).TextColor(t.Text)
			input := screenInput(c, &aboutScreen.feedDraft, abtCopySourcePlaceholder,
				abtCopySourceLabel).Grow(1).MinWidth(0)
			if input.Changed() {
				aboutScreen.feedDirty = true
			}
			abtSourceSaveButton(c, t)
		})
		abtSmallStyle.style(t, gui.Text(c, abtCopySourceHint)).
			TextColor(t.TextMuted).MaxLines(2)
		if aboutScreen.feedError != "" {
			abtSmallStyle.style(t, gui.Text(c, aboutScreen.feedError)).
				TextColor(t.Danger).Role(gui.RoleAlert).MaxLines(2)
		}
	})
}

// abtSourceSaveButton 保存更新源；校验失败不落盘，原因留在本屏。
func abtSourceSaveButton(c *gui.Context, t *gui.Theme) {
	disabled := aboutWire.intents == nil
	b := gui.ButtonBase(c).Shrink(0).Height(abtBtnH).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Label(abtCopySourceSave).Disabled(disabled)
	b = b.Background(t.Background)
	b.BorderWidth(strokeHairline).BorderColor(t.Border)
	switch {
	case disabled:
		b = b.Opacity(abtAlpha50)
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		abtValueStyle.style(t, gui.Text(c, abtCopySourceSave)).TextColor(t.Text).SingleLine()
	})
	if b.Clicked() && !disabled {
		if err := aboutWire.intents.SetUpdateFeed(context.Background(), aboutScreen.feedDraft); err != nil {
			aboutScreen.feedError = err.Error()
		} else {
			aboutScreen.feedDirty, aboutScreen.feedError = false, ""
		}
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

// abtWipBadge 是「WIP」徽标（S-about-33）：中性面、小圆角。要求 R-ABT-258 写
// 的是警示底色，但运行时截图取到的是中性面（#f3f3f3 底、#171717 字），以截图为
// 准；差异登记在文件头。
func abtWipBadge(c *gui.Context, t *gui.Theme) {
	b := gui.Row(c).Shrink(0).AlignItems(gui.Center).
		Padding(abtBadgePadV, abtBadgePadH).Radius(abtBadgeR).
		Background(t.SurfaceHover).Label(abtCopyWIPBadgeLabel)
	b.Children(func() {
		abtWipStyle.style(t, gui.Text(c, abtCopyWIPBadge)).TextColor(t.Text).
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
