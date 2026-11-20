package ui

// 本文件是「导出对话」面板：格式（Markdown / JSON / 纯文本）、范围（整条会话 /
// 仅当前分支 / 选中的轮次）、前 20 行预览，以及剪贴板与文件两条导出通路。
//
// 内容只从视图手里已有的快照拼（SessionSnapshot.Rows / 样例 rounds），不另读
// 服务端：导出的是屏上这条对话，不是第二份事实。文件用一个宿主给的保存面板
// （Options.SaveFile）；宿主没给（测试、非桌面运行）时面板退回手输路径，
// 不自己猜平台能力。
//
// 回执必须可核对：剪贴板给字符数，文件给字节数；失败给原样原因（RoleAlert）。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	gui "github.com/egoist/mygo/ui"
)

// 导出格式与范围：界面上的三档各自一个稳定取值（不随文案变）。
const (
	exportFormatMarkdown = "markdown"
	exportFormatJSON     = "json"
	exportFormatText     = "text"

	exportScopeSession = "session"
	exportScopeBranch  = "branch"
	exportScopeTurns   = "turns"
)

// 面板文案（逐字取 requirements-global §7.1 的句子）。
const (
	exportTitle        = "导出对话"
	exportDescFormat   = "格式"
	exportDescScope    = "范围"
	exportDescPreview  = "预览（前 20 行）"
	exportDescPath     = "保存位置"
	exportCopyCancel   = "取消"
	exportCopyClip     = "导出到剪贴板"
	exportCopyFile     = "导出到文件"
	exportCopyWrite    = "写入文件"
	exportCopyReselect = "打开系统保存面板"
	exportEmptyTurns   = "这个会话还没有回合。"
	exportEmptyPreview = "没有可预览的内容。"
	exportManualHint   = "系统保存面板不可用，请直接输入保存路径。"

	// G-G1-03 / G-G1-05 两个分组标题（导出内容 / 消息）。
	exportDescContent  = "导出内容"
	exportDescMessages = "消息"
	// G-G1-04 三项内容勾选。
	exportOptCompress = "压缩记录"
	exportOptTools    = "工具详情"
	exportOptThink    = "思考"

	// G-G1-07 回合搜索的占位与读屏名。
	exportSearchTurns = "搜索回合"
	// G-G1-08 按搜索批量排除；计数为当次匹配且仍勾选的回合数。
	exportExcludeMatchesFmt = "排除 %d 个匹配"
	exportExcludeMatchesTip = "把匹配当前搜索且仍勾选的回合全部取消勾选"
	// G-G1-09 搜索无命中（与「没有回合」分开，前者是过滤结果）。
	exportNoMatchText = "没有回合匹配此搜索。"
	// G-G1-11 回合正文的最大高：超出在块内双向滚动。
	exportTurnBodyMax = 256
	// G-G1-13 勾选「压缩记录」且存在压缩记录时的说明句。
	exportCompressNote = "压缩记录会单独描述被折叠的回合，排除某个回合不会将其从压缩记录中移除。"
	// G-G1-14 导出前快照过期（会话在面板打开后被重新加载）的提示。
	exportStaleNotice = "会话快照已过期并重新加载。请检查消息和排除项后再次导出。"
)

// previewMaxLines 是预览的行数上限。
const previewMaxLines = 20

// exportBodyHeight 是面板正文区的最大高：窗口再矮也把按钮行留在窗内。
func exportBodyHeight(c *gui.Context) float32 {
	_, height := c.Size()
	if height <= 0 {
		return 320
	}
	limit := height - space12*3
	if limit < 160 {
		return 160
	}
	return limit
}

// exportChoice 是一档选择：取值与屏上文案。
type exportChoice struct {
	value string
	label string
}

var (
	exportFormats = []exportChoice{
		{exportFormatMarkdown, "Markdown"},
		{exportFormatJSON, "JSON"},
		{exportFormatText, "纯文本"},
	}
	exportScopes = []exportChoice{
		{exportScopeSession, "整条会话"},
		{exportScopeBranch, "仅当前分支"},
		{exportScopeTurns, "选中的轮次"},
	}
)

// exportState 是面板的可丢视图状态：开合、当前会话、格式与范围、勾选、手输
// 路径与回执。它不是事实：内容每帧从快照重拼。
type exportState struct {
	open      bool
	sessionID string
	title     string
	// at 是打开面板的时间：JSON 文件头与预览都读它，预览不会每帧变一个时间戳。
	at     time.Time
	format string
	scope  string
	// turns 是「选中的轮次」的勾选位；nil 表示全选（打开时的默认）。
	turns map[int]*bool
	// expanded 记本次打开里哪些回合展开了正文（G-G1-11）；正文随快照已在，
	// 不另取，也不留「加载中…」。
	expanded map[int]bool
	// search 是回合清单的过滤词（G-G1-07），只在视图里存活。
	search string
	// includeCompress / includeTools / includeThink 是「导出内容」三项勾选
	// （G-G1-04）：勾上才进导出文本，默认都不勾（与首屏一致）。这三项跨次打开
	// 保留，与格式、范围同一口径（R-GF-033 固定一种行为）。
	includeCompress bool
	includeTools    bool
	includeThink    bool
	// stale 是快照过期提示（G-G1-14）；mark 是打开面板时的快照指纹，用来判
	// 「面板打开后会话被重新加载」。
	stale bool
	mark  string
	// manual 为真表示保存面板不可用，面板里出输入位；path 是手输路径。
	manual       bool
	path         string
	manualReason string
	busy         bool
	failure      string
	receipt      string
}

// exportFileOutcome 是后台保存 / 写入交回渲染线程的一次结果。
type exportFileOutcome struct {
	// cancelled 表示用户在系统保存面板里取消。
	cancelled bool
	// manualReason 非空表示保存面板不可用，退回手输路径。
	manualReason string
	suggested    string
	path         string
	bytes        int64
	err          error
}

// exportTool 是导出里的一次工具调用。
type exportTool struct {
	Action string `json:"action,omitempty"`
	Args   string `json:"args,omitempty"`
	Result string `json:"result,omitempty"`
}

// exportTurn 是一个导出回合：用户消息 + 这一轮的助手内容。
type exportTurn struct {
	Index     int          `json:"index"`
	Branch    string       `json:"branch,omitempty"`
	User      string       `json:"user,omitempty"`
	Assistant []string     `json:"assistant,omitempty"`
	Thinking  []string     `json:"thinking,omitempty"`
	Tools     []exportTool `json:"tools,omitempty"`
	Notes     []string     `json:"notes,omitempty"`
}

// exportDocument 是 JSON 导出的落盘结构。
type exportDocument struct {
	Format        string       `json:"format"`
	FormatVersion int          `json:"formatVersion"`
	ExportedAt    string       `json:"exportedAt"`
	SessionID     string       `json:"sessionId,omitempty"`
	Title         string       `json:"title,omitempty"`
	Branch        string       `json:"branch,omitempty"`
	Scope         string       `json:"scope"`
	TurnCount     int          `json:"turnCount"`
	Turns         []exportTurn `json:"turns"`
}

// exportDocFormat / exportDocVersion 是 JSON 导出的格式名与版本。
const (
	exportDocFormat  = "dimagent.dialogue-export"
	exportDocVersion = 1
)

// openExportPanel 打开面板：会话与标题在打开时取一次（切换会话后不拿新会话的
// 内容顶替旧回执）。
func (a *App) openExportPanel(sessionID, title string) {
	st := &a.export
	st.open = true
	st.sessionID = sessionID
	st.title = title
	st.at = time.Now()
	if st.format == "" {
		st.format = exportFormatMarkdown
	}
	if st.scope == "" {
		st.scope = exportScopeSession
	}
	// 每次打开回到「全部保留」：上一次的排除项不静默沿用（R-GF-033 要求固定
	// 一种口径）；格式、范围与三项内容勾选保留在上次选的那一档。
	st.turns = nil
	st.expanded = map[int]bool{}
	st.search = ""
	st.stale = false
	st.mark = a.exportSnapshotMark()
	st.manual = false
	st.path = ""
	st.manualReason = ""
	st.failure = ""
	st.receipt = ""
	st.busy = false
}

// closeExportPanel 收起面板，保留格式、范围与勾选（记忆口径见 openExportPanel）。
func (a *App) closeExportPanel() {
	a.export.open = false
	a.export.busy = false
}

// exportSession 是左栏行菜单的「导出」：先切到这一行的会话，再打开面板。
//
// 样例模式只改选中的样例行；连着写通道时先发 OpenSession（导出读的是当前快照，
// 不拿别的会话的行顶替）。
func (a *App) exportSession(index int) {
	names := a.sessionNames()
	if index < 0 || index >= len(names) {
		a.headerNote = noteNoSessionID
		return
	}
	sessionID := names[index]
	title := sessionID
	if titles := a.sessionTitles(); index < len(titles) && strings.TrimSpace(titles[index]) != "" {
		title = titles[index]
	}
	if !a.wired() {
		a.selectedSession = "s" + strconv.Itoa(index+1)
		a.openExportPanel(sessionID, title)
		return
	}
	if current, ok := a.liveSession(); ok && current.ID == sessionID {
		// 导出只读快照：已经在看的会话不需要写通道。
		a.openExportPanel(sessionID, title)
		return
	}
	if a.intents == nil {
		a.headerNote = noteNoWriteChannel
		return
	}
	a.headerNote = "正在打开会话…"
	a.dispatch("打开会话", func(ctx context.Context) error {
		return a.intents.OpenSession(ctx, sessionID)
	}, func() {
		a.openExportPanel(sessionID, title)
	})
}

// exportPanel 画导出对话框。内容每帧从快照重拼；没有回合时不给导出按钮可点。
func (a *App) exportPanel(c *gui.Context) {
	st := &a.export
	if !st.open || a.screen != ScreenMain {
		return
	}
	// 面板开着时切了会话：不拿新会话的行顶替旧标题与旧回执，直接收起。
	if snap, ok := a.liveSession(); ok && st.sessionID != "" && snap.ID != st.sessionID {
		a.closeExportPanel()
		return
	}
	t := c.Theme()
	turns, branch := a.exportTurns()
	// G-G1-14 快照过期：面板打开后会话被重新加载（行或分支变了）就不再拿新内容
	// 直接导出，先给一次提示；指纹随后更新，提示恒一条，不叠加。
	if mark := a.exportSnapshotMark(); mark != "" && st.mark != "" && mark != st.mark {
		st.stale = true
		st.mark = mark
	}
	selected := a.exportScopeTurns(turns, branch)
	content := a.exportContent(selected, branch)
	name := a.exportName()

	// 键盘：Esc 等同「取消」。导出进行中不关闭，留给在途写入回执（R-GF-126）。
	if !st.busy && c.Shortcut(0, gui.KeyEscape) {
		a.closeExportPanel()
		return
	}

	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space2).Width(accDialogWidth(c)).
			Role(gui.RoleDialog).Label(exportTitle)
		panel.Children(func() {
			// G-G1-01 「导出对话」标题
			typeDialogTitle.style(t, gui.Text(c, exportTitle)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(3).SingleLine()
			// 正文区可滚动：窄窗或手动路径 + 失败原因同屏时，按钮行仍在窗口内
			// （R-GF-006：内容区可滚动，按钮不被挤到窗外）。
			body := gui.Scroll(c).Shrink(0).MaxHeight(exportBodyHeight(c)).Label("导出面板内容")
			body.Children(func() {
				gui.Column(c).Gap(space2).Children(func() {
					// G-G1-02 导出说明句
					desc := fmt.Sprintf("导出「%s」的会话对话。不包含原始 Provider 请求。", name)
					typeBody.style(t, gui.Text(c, desc)).
						TextColor(t.TextMuted).Selectable().Label(desc)

					// G-G1-03 / G-G1-04 导出内容与格式选项（消息 / 工具详情 / 思考）
					a.exportChoices(c, exportDescFormat, exportFormats, &st.format)
					a.exportChoices(c, exportDescScope, exportScopes, &st.scope)
					// 快照只有已加载的那一段（HasOlder）：导出的范围不能写成全量，
					// 把口径摊在范围下面，不让人以为「整条会话」是全部历史。
					if a.sessionPartial() {
						// G-G1-09 对话未全量加载的口径说明
						screenHint(c, partialHistoryNote)
					}
					// G-G1-03 导出内容分组（三项勾选 / G-G1-13 压缩记录说明）
					a.exportContentOptions(c, turns)
					// G-G1-05 第二分组「消息」：无回合时只给状态句
					screenGroupTitle(c, exportDescMessages)
					if len(turns) == 0 {
						// G-G1-09 没有回合时的状态句
						screenHint(c, exportEmptyTurns)
					} else {
						a.exportTurnPicker(c, turns)
					}

					screenGroupTitle(c, exportDescPreview)
					preview := firstLines(content, previewMaxLines)
					if strings.TrimSpace(preview) == "" {
						screenHint(c, exportEmptyPreview)
					} else {
						a.exportCodeBlock(c, preview)
						if lines := strings.Count(strings.TrimRight(content, "\n"), "\n") + 1; lines > previewMaxLines {
							screenHint(c, fmt.Sprintf("共 %d 行；预览前 %d 行。", lines, previewMaxLines))
						}
					}

					if st.manual {
						screenGroupTitle(c, exportDescPath)
						input := gui.TextInputBase(c, &st.path).Label(exportDescPath).
							Placeholder("/path/to/export.md")
						input.BorderWidth(strokeHairline).BorderColor(t.Border)
						hint := st.manualReason
						if hint == "" {
							hint = exportManualHint
						}
						screenHint(c, hint)
					}

					// G-G1-14 快照过期提示（role=status；不自动导出，等用户再点）
					if st.stale {
						typeSecondary.style(t, gui.Text(c, exportStaleNotice)).
							TextColor(t.Warning).Role(gui.RoleStatus).
							Label(exportStaleNotice).Selectable()
					}
					// G-G1-15 导出失败原因
					if st.failure != "" {
						typeSecondary.style(t, gui.Text(c, st.failure)).TextColor(t.Danger).
							Role(gui.RoleAlert).Label(st.failure).Selectable()
					}
					if st.receipt != "" {
						typeSecondary.style(t, gui.Text(c, st.receipt)).TextColor(t.Success).
							Role(gui.RoleStatus).Label(st.receipt).Selectable()
					}
				})
			})

			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				if st.busy {
					screenTextButtonDisabled(c, exportCopyCancel, "正在导出，不能取消", rowRegular)
					// G-G1-17 导出中…
					screenTextButtonDisabled(c, "导出中…", "正在导出", rowRegular)
					return
				}
				// G-G1-16 取消
				if screenButton(c, exportCopyCancel, exportCopyCancel, exportCopyCancel, false, rowRegular) {
					a.closeExportPanel()
					open = false
				}
				canExport := len(selected) > 0
				if screenButtonDisabledUntil(c, exportCopyClip, "把导出内容复制到剪贴板",
					exportCopyClip, !canExport, rowRegular) && canExport {
					a.exportToClipboard(c, content)
				}
				if st.manual {
					if screenButtonDisabledUntil(c, exportCopyWrite, "把导出内容写到输入的路径",
						exportCopyWrite, !canExport, rowRegular) && canExport {
						a.exportToManualFile(content)
					}
					if a.saveFile != nil && screenButton(c, exportCopyReselect,
						exportCopyReselect, exportCopyReselect, false, rowRegular) {
						st.manual = false
						st.manualReason = ""
						a.startExportFile(content)
					}
					return
				}
				// G-G1-17 导出
				if screenButtonDisabledUntil(c, exportCopyFile,
					"打开保存对话框并写文件", exportCopyFile, !canExport, rowRegular) && canExport {
					a.startExportFile(content)
				}
			})
		})
	}))

	if !open || m.Dismissed() {
		a.closeExportPanel()
	}
}

// exportChoices 画一排单选：选中的一档用主色底，其余描边；点击只改视图状态。
func (a *App) exportChoices(c *gui.Context, label string, options []exportChoice, current *string) {
	t := c.Theme()
	screenGroupTitle(c, label)
	gui.Row(c).Gap(space1).Children(func() {
		for _, option := range options {
			option := option
			selected := *current == option.value
			b := gui.ButtonBase(c).Shrink(0).Height(rowRegular).Padding(0, space3).
				Radius(radius).Label(label + "：" + option.label).Role(gui.RoleRadio).
				AccessSelected(selected).Tooltip(option.label)
			b = b.Background(t.Background)
			if selected {
				b = b.Background(t.SurfaceHover)
			}
			if b.Hovered() && !selected {
				b = b.Background(t.SurfaceHover)
			}
			if b.Clicked() {
				*current = option.value
			}
			ink := t.Text
			if selected {
				ink = t.Accent
			}
			b.Children(func() {
				typeSecondary.style(t, gui.Text(c, option.label)).TextColor(ink).SingleLine()
			})
		}
	})
}

// exportContentOptions 是「导出内容」分组：三项勾选（压缩记录 / 工具详情 /
// 思考），勾选「压缩记录」且本轮快照里确有压缩记录时补一句说明。
// G-G1-03 分组标签 / G-G1-04 三项勾选 / G-G1-13 压缩记录说明
func (a *App) exportContentOptions(c *gui.Context, turns []exportTurn) {
	st := &a.export
	t := c.Theme()
	screenGroupTitle(c, exportDescContent)
	gui.Column(c).Gap(space1).Children(func() {
		gui.Checkbox(c, &st.includeCompress, exportOptCompress).Height(rowRegular)
		gui.Checkbox(c, &st.includeTools, exportOptTools).Height(rowRegular)
		gui.Checkbox(c, &st.includeThink, exportOptThink).Height(rowRegular)
	})
	// G-G1-13 压缩记录说明：只在勾上且这次导出的回合里存在压缩记录时出现。
	if st.includeCompress && exportHasCompression(turns) {
		typeMicro.style(t, gui.Text(c, exportCompressNote)).TextColor(t.TextMuted).
			Label(exportCompressNote)
	}
}

// exportHasCompression 判断回合里是否含压缩记录（导出 Tuns 的 Notes 前缀）。
func exportHasCompression(turns []exportTurn) bool {
	const prefix = "压缩记录："
	for _, turn := range turns {
		for _, note := range turn.Notes {
			if strings.HasPrefix(note, prefix) {
				return true
			}
		}
	}
	return false
}

// exportSnapshotMark 是当前会话快照的指纹：标识、分支、行数、轮次数与最后一行。
// 只用来发现「面板打开后会话被重新加载」，不参与导出内容。
func (a *App) exportSnapshotMark() string {
	snap, ok := a.liveSession()
	if !ok {
		return ""
	}
	last := ""
	if n := len(snap.Rows); n > 0 {
		row := snap.Rows[n-1]
		last = row.Kind + ":" + row.Text
	}
	return fmt.Sprintf("%s|%s|%d|%d|%t|%s",
		snap.ID, snap.Branch, len(snap.Rows), snap.TurnCount, snap.HasOlder, last)
}

// exportTurnMatches 是回合搜索的匹配口径：序号、用户消息、助手回复、思考、
// 工具（动作 / 参数 / 结果）与备注里任意一处含查询词即匹配（不区分大小写）。
func exportTurnMatches(turn exportTurn, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	fields := []string{turn.User, strings.Join(turn.Assistant, "\n"),
		strings.Join(turn.Thinking, "\n"), strings.Join(turn.Notes, "\n")}
	for _, tool := range turn.Tools {
		fields = append(fields, tool.Action, tool.Args, tool.Result)
	}
	fields = append(fields, fmt.Sprintf("第 %d 轮", turn.Index))
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), q) {
			return true
		}
	}
	return false
}

// exportTurnPicker 是「消息」下的回合清单：计数、搜索、按搜索批量排除、全部
// 保留 / 全部排除与逐轮勾选 + 展开正文。
func (a *App) exportTurnPicker(c *gui.Context, turns []exportTurn) {
	t := c.Theme()
	st := &a.export
	kept := 0
	for _, turn := range turns {
		if *a.exportTurnPtr(turn.Index) {
			kept++
		}
	}
	// G-G1-06 保留 {kept} / {total} 个回合
	count := fmt.Sprintf("保留 %d / %d 个回合", kept, len(turns))
	typeSecondary.style(t, gui.Text(c, count)).
		TextColor(t.TextMuted).Label(count)

	// G-G1-07 回合搜索（左置放大镜、占位「搜索回合」，无边框）
	gui.Row(c).Shrink(0).Height(screenControlH).Padding(0, space2).
		Gap(space2).AlignItems(gui.Center).Children(func() {
		iconGlyph(c, iconSearch, t.TextMuted, "search")
		gui.TextInputBase(c, &st.search).Placeholder(exportSearchTurns).
			Label(exportSearchTurns).Grow(1)
	})

	filtered := make([]exportTurn, 0, len(turns))
	for _, turn := range turns {
		if exportTurnMatches(turn, st.search) {
			filtered = append(filtered, turn)
		}
	}
	// G-G1-06 无过滤时计数即全量；过滤时把命中数写在排除按钮上，不改 total。
	matchedKept := 0
	for _, turn := range filtered {
		if *a.exportTurnPtr(turn.Index) {
			matchedKept++
		}
	}
	total := len(turns)
	gui.Row(c).Gap(space2).Children(func() {
		// G-G1-08 排除 {count} 个匹配（与「全部排除」是两个入口，不合并）
		if strings.TrimSpace(st.search) != "" {
			label := fmt.Sprintf(exportExcludeMatchesFmt, matchedKept)
			if exportTextButton(c, label, exportExcludeMatchesTip, matchedKept == 0) {
				for _, turn := range filtered {
					*a.exportTurnPtr(turn.Index) = false
				}
			}
		}
		// G-G1-12 全部保留 / 全部排除
		if exportTextButton(c, "全部保留", "第 1 轮起全部勾选", kept == total) {
			for _, turn := range turns {
				*a.exportTurnPtr(turn.Index) = true
			}
		}
		if exportTextButton(c, "全部排除", "全部回合取消勾选", kept == 0) {
			for _, turn := range turns {
				*a.exportTurnPtr(turn.Index) = false
			}
		}
	})
	if len(filtered) == 0 {
		// G-G1-09 搜索无匹配（「没有回合」与「没有匹配」是两个条件）
		screenHint(c, exportNoMatchText)
		return
	}
	// G-G1-10 回合清单
	list := gui.Column(c).Shrink(0).Label("轮次清单")
	list.Children(func() {
		gui.Column(c).Gap(space1).Children(func() {
			for _, turn := range filtered {
				a.exportTurnRow(c, turn)
			}
		})
	})
}

// exportTurnRow 是一行回合：展开箭头 + 方框 + 序号 + 预览（单行）；展开后
// 正文在行下方以等宽块显示（最大高 256，双向滚动）。
// G-G1-10 回合条目 / G-G1-11 回合正文
func (a *App) exportTurnRow(c *gui.Context, turn exportTurn) {
	st := &a.export
	if st.expanded == nil {
		st.expanded = map[int]bool{}
	}
	expanded := st.expanded[turn.Index]
	// G-G1-10 展开箭头
	arrow := accIconSpec{
		icon: iconChevronR, label: "展开回合正文", tip: "展开这一轮的正文",
		size: screenIconMicro, hit: rowRegular,
	}
	if expanded {
		arrow.icon = iconChevronDown
		arrow.label, arrow.tip = "收起回合正文", "收起这一轮的正文"
	}
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(space1).Children(func() {
		if accIconButton(c, arrow) {
			st.expanded[turn.Index] = !expanded
		}
		label := fmt.Sprintf("第 %d 轮：%s", turn.Index, turnPreview(turn))
		box := gui.Checkbox(c, a.exportTurnPtr(turn.Index), label)
		box.Grow(1).Height(rowRegular)
	})
	if !expanded {
		return
	}
	a.exportTurnBodyBlock(c, turn)
}

// exportTurnBodyBlock 是展开后的回合正文块：等宽 12px、浅底、圆角 6、内边距 8、
// 最大高 256、双向滚动、可选中复制。正文随快照已在，没有「加载中…」阶段；正文
// 为空时块内留空（G-G1-11）。
func (a *App) exportTurnBodyBlock(c *gui.Context, turn exportTurn) {
	t := c.Theme()
	body := exportTurnBodyText(turn)
	gui.ScrollBoth(c).Shrink(0).MaxHeight(exportTurnBodyMax).Children(func() {
		gui.Column(c).Padding(space2).Radius(radius).Background(t.SurfaceHover).
			Label("回合正文").Selectable().Children(func() {
			if strings.TrimSpace(body) == "" {
				return
			}
			typeCode.style(t, gui.Text(c, body)).TextColor(t.Text).Selectable()
		})
	})
}

// exportTurnBodyText 把一轮的内容摊成正文：用户 / 助手 / 思考 / 工具 / 备注。
func exportTurnBodyText(turn exportTurn) string {
	var b strings.Builder
	if strings.TrimSpace(turn.User) != "" {
		b.WriteString("用户：" + turn.User + "\n")
	}
	if len(turn.Assistant) > 0 {
		b.WriteString("助手：" + strings.Join(turn.Assistant, "\n\n") + "\n")
	}
	for _, thinking := range turn.Thinking {
		b.WriteString("思考：" + thinking + "\n")
	}
	for _, tool := range turn.Tools {
		b.WriteString("工具：" + toolLine(tool) + "\n")
	}
	for _, note := range turn.Notes {
		b.WriteString(note + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// exportTurnPtr 取一轮勾选位的稳定指针：只在视图里存，不写事实。
func (a *App) exportTurnPtr(index int) *bool {
	if a.export.turns == nil {
		a.export.turns = map[int]*bool{}
	}
	if p, ok := a.export.turns[index]; ok {
		return p
	}
	v := true
	p := &v
	a.export.turns[index] = p
	return p
}

// exportCodeBlock 是预览块：等宽、浅底、圆角、内边距，可选中复制。
func (a *App) exportCodeBlock(c *gui.Context, text string) {
	t := c.Theme()
	block := gui.Column(c).Shrink(0).Grow(1).Padding(space2).Radius(radius).
		Background(t.SurfaceHover).Label("对话预览").Selectable()
	block.Children(func() {
		typeCode.style(t, gui.Text(c, text)).TextColor(t.Text).Selectable()
	})
}

// exportTurns 从当前快照拼出回合；第二个返回值是当前分支。
//
// 只认快照里已有的行：用户行开一轮，助手 / 思考 / 工具 / 失败 / 压缩归到当前
// 轮；没有用户行之前的行也归入首轮，不丢内容。样例模式按 rounds 一一成轮。
func (a *App) exportTurns() ([]exportTurn, string) {
	if snap, ok := a.liveSession(); ok {
		turns := make([]exportTurn, 0, len(snap.Rows))
		current := -1
		ensure := func(index int, branch string) int {
			if current >= 0 {
				return current
			}
			if index <= 0 {
				index = 1
			}
			turns = append(turns, exportTurn{Index: index, Branch: branch})
			current = len(turns) - 1
			return current
		}
		for _, row := range snap.Rows {
			switch row.Kind {
			case RowUser:
				index := row.TurnIndex
				if index <= 0 {
					index = len(turns) + 1
				}
				branch := row.BranchID
				if branch == "" {
					branch = snap.Branch
				}
				turns = append(turns, exportTurn{Index: index, Branch: branch, User: row.Text})
				current = len(turns) - 1
			case RowAssistant:
				if strings.TrimSpace(row.Text) == "" {
					continue
				}
				i := ensure(row.TurnIndex, branchOf(row, snap.Branch))
				turns[i].Assistant = append(turns[i].Assistant, row.Text)
			case RowThinking:
				if strings.TrimSpace(row.Text) == "" {
					continue
				}
				i := ensure(row.TurnIndex, branchOf(row, snap.Branch))
				turns[i].Thinking = append(turns[i].Thinking, row.Text)
			case RowTool:
				i := ensure(row.TurnIndex, branchOf(row, snap.Branch))
				turns[i].Tools = append(turns[i].Tools, exportTool{
					Action: row.Title, Args: row.Args, Result: row.Text,
				})
			case RowToolGroup:
				i := ensure(row.TurnIndex, branchOf(row, snap.Branch))
				for _, child := range row.Children {
					turns[i].Tools = append(turns[i].Tools, exportTool{
						Action: child.Title, Args: child.Args, Result: child.Text,
					})
				}
			case RowError:
				i := ensure(row.TurnIndex, branchOf(row, snap.Branch))
				note := "本轮失败"
				if strings.TrimSpace(row.Text) != "" {
					note = "失败：" + row.Text
				}
				turns[i].Notes = append(turns[i].Notes, note)
			case RowCompression:
				i := ensure(row.TurnIndex, branchOf(row, snap.Branch))
				if strings.TrimSpace(row.Text) != "" {
					turns[i].Notes = append(turns[i].Notes, "压缩记录："+row.Text)
				}
			case RowFileChanges, RowArtifact:
				i := ensure(row.TurnIndex, branchOf(row, snap.Branch))
				for _, child := range row.Children {
					if strings.TrimSpace(child.Text) != "" {
						turns[i].Notes = append(turns[i].Notes, "改动："+child.Text)
					}
				}
			default:
				if strings.TrimSpace(row.Text) == "" {
					continue
				}
				i := ensure(row.TurnIndex, branchOf(row, snap.Branch))
				turns[i].Assistant = append(turns[i].Assistant, row.Text)
			}
		}
		return turns, snap.Branch
	}

	turns := make([]exportTurn, 0, len(a.rounds))
	for i, r := range a.rounds {
		turn := exportTurn{Index: i + 1, Branch: a.branch, User: r.user}
		if strings.TrimSpace(r.reply) != "" {
			turn.Assistant = append(turn.Assistant, r.reply)
		}
		if strings.TrimSpace(r.thinking) != "" {
			turn.Thinking = append(turn.Thinking, r.thinking)
		}
		for _, item := range r.ops {
			turn.Tools = append(turn.Tools, exportTool{
				Action: item.action, Args: item.target, Result: item.result,
			})
		}
		if r.note != "" {
			turn.Notes = append(turn.Notes, r.note)
		}
		if r.divider != "" {
			turn.Notes = append(turn.Notes, "压缩记录："+r.divider)
		}
		turns = append(turns, turn)
	}
	return turns, a.branch
}

// branchOf 取一行的分支；空时退回当前分支。
func branchOf(row RowSnapshot, current string) string {
	if row.BranchID != "" {
		return row.BranchID
	}
	return current
}

// exportScopeTurns 按当前范围筛回合：整条会话 / 仅当前分支 / 选中的轮次。
func (a *App) exportScopeTurns(turns []exportTurn, branch string) []exportTurn {
	if len(turns) == 0 {
		return nil
	}
	switch a.export.scope {
	case exportScopeBranch:
		out := make([]exportTurn, 0, len(turns))
		for _, turn := range turns {
			if turn.Branch == "" || branch == "" || turn.Branch == branch {
				out = append(out, turn)
			}
		}
		return out
	case exportScopeTurns:
		out := make([]exportTurn, 0, len(turns))
		for _, turn := range turns {
			if *a.exportTurnPtr(turn.Index) {
				out = append(out, turn)
			}
		}
		return out
	default:
		return turns
	}
}

// exportContent 按格式生成整份导出文本。空回合也给出文件头：导出的是一份文件，
// 不是空字符串。
//
// G-G1-03 三项内容勾选在这里生效：没勾的段整段不进导出（工具 / 思考 /
// 压缩记录），消息正文始终导出。
func (a *App) exportContent(turns []exportTurn, branch string) string {
	turns = a.exportApplyContentOptions(turns)
	meta := exportDocument{
		Format:        exportDocFormat,
		FormatVersion: exportDocVersion,
		ExportedAt:    a.export.at.Format(time.RFC3339),
		SessionID:     a.export.sessionID,
		Title:         a.exportName(),
		Branch:        branch,
		Scope:         exportScopeLabel(a.export.scope),
		TurnCount:     len(turns),
		Turns:         turns,
	}
	if meta.Turns == nil {
		meta.Turns = []exportTurn{}
	}
	switch a.export.format {
	case exportFormatJSON:
		encoded, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			// 结构体全是可编码字段；这里保底给一条可读的错误文本，不返回空串。
			return "导出失败：" + err.Error()
		}
		return string(encoded) + "\n"
	case exportFormatText:
		return exportText(meta)
	default:
		return exportMarkdown(meta)
	}
}

// exportApplyContentOptions 按三项勾选过滤回合内容：未勾的段置空，不改消息
// 正文与轮次序号（G-G1-03「三项全部关闭时导出为纯消息」）。
func (a *App) exportApplyContentOptions(turns []exportTurn) []exportTurn {
	st := &a.export
	if st.includeCompress && st.includeTools && st.includeThink {
		return turns
	}
	out := make([]exportTurn, len(turns))
	for i, turn := range turns {
		copied := turn
		if !st.includeTools {
			copied.Tools = nil
		}
		if !st.includeThink {
			copied.Thinking = nil
		}
		if !st.includeCompress {
			notes := make([]string, 0, len(turn.Notes))
			for _, note := range turn.Notes {
				if strings.HasPrefix(note, "压缩记录：") {
					continue
				}
				notes = append(notes, note)
			}
			copied.Notes = notes
		}
		out[i] = copied
	}
	return out
}

// exportMarkdown 是 Markdown 导出：文件头 + 每轮的小节。
func exportMarkdown(doc exportDocument) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", nonEmpty(doc.Title, "未命名会话"))
	fmt.Fprintf(&b, "- 会话：%s\n", nonEmpty(doc.SessionID, "（未知）"))
	fmt.Fprintf(&b, "- 分支：%s\n", nonEmpty(doc.Branch, "（未知）"))
	fmt.Fprintf(&b, "- 范围：%s\n", doc.Scope)
	fmt.Fprintf(&b, "- 回合：%d\n", doc.TurnCount)
	for _, turn := range doc.Turns {
		fmt.Fprintf(&b, "\n## 第 %d 轮\n", turn.Index)
		if strings.TrimSpace(turn.User) != "" {
			fmt.Fprintf(&b, "\n### 你\n\n%s\n", turn.User)
		}
		if len(turn.Assistant) > 0 {
			fmt.Fprintf(&b, "\n### 助手\n\n%s\n", strings.Join(turn.Assistant, "\n\n"))
		}
		for _, thinking := range turn.Thinking {
			fmt.Fprintf(&b, "\n> 思考：%s\n", indentAfterFirst(thinking, "> "))
		}
		for _, tool := range turn.Tools {
			fmt.Fprintf(&b, "\n> 工具：%s\n", toolLine(tool))
		}
		for _, note := range turn.Notes {
			fmt.Fprintf(&b, "\n> %s\n", note)
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// exportText 是纯文本导出：与 Markdown 同一份内容，去掉标记。
func exportText(doc exportDocument) string {
	var b strings.Builder
	b.WriteString(nonEmpty(doc.Title, "未命名会话"))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "会话：%s\n", nonEmpty(doc.SessionID, "（未知）"))
	fmt.Fprintf(&b, "分支：%s\n", nonEmpty(doc.Branch, "（未知）"))
	fmt.Fprintf(&b, "范围：%s\n", doc.Scope)
	fmt.Fprintf(&b, "回合：%d\n", doc.TurnCount)
	for _, turn := range doc.Turns {
		fmt.Fprintf(&b, "\n--- 第 %d 轮 ---\n", turn.Index)
		if strings.TrimSpace(turn.User) != "" {
			fmt.Fprintf(&b, "\n用户：\n%s\n", turn.User)
		}
		if len(turn.Assistant) > 0 {
			fmt.Fprintf(&b, "\n助手：\n%s\n", strings.Join(turn.Assistant, "\n\n"))
		}
		for _, thinking := range turn.Thinking {
			fmt.Fprintf(&b, "\n思考：\n%s\n", thinking)
		}
		for _, tool := range turn.Tools {
			fmt.Fprintf(&b, "\n工具：%s\n", toolLine(tool))
		}
		for _, note := range turn.Notes {
			fmt.Fprintf(&b, "\n%s\n", note)
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// toolLine 把一次工具调用写成一行。
func toolLine(tool exportTool) string {
	action := nonEmpty(tool.Action, "工具")
	if strings.TrimSpace(tool.Args) != "" {
		action += " " + tool.Args
	}
	result := strings.TrimSpace(tool.Result)
	if result == "" {
		return action
	}
	return action + " — " + result
}

// indentAfterFirst 把第二行起缩进，保证引用块里仍是引用。
func indentAfterFirst(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

// exportScopeLabel 是范围在文件头里的写法。
func exportScopeLabel(scope string) string {
	for _, choice := range exportScopes {
		if choice.value == scope {
			return choice.label
		}
	}
	return exportScopes[0].label
}

// exportExtension 是格式对应的文件扩展名。
func exportExtension(format string) string {
	switch format {
	case exportFormatJSON:
		return ".json"
	case exportFormatText:
		return ".txt"
	default:
		return ".md"
	}
}

// exportName 是面板里的会话名：优先当前快照的标题，其次打开时记下的名字。
func (a *App) exportName() string {
	if snap, ok := a.liveSession(); ok {
		if title := strings.TrimSpace(snap.Title); title != "" {
			return title
		}
	}
	if title := strings.TrimSpace(a.export.title); title != "" {
		return title
	}
	return "未命名会话"
}

// exportSuggestedPath 是保存面板的建议路径：会话工作目录 + 带时间戳的文件名；
// 没有工作目录时只给文件名（用户自己选位置）。
func (a *App) exportSuggestedPath() string {
	name := "dimagent-export-" + time.Now().Format("20060102-150405") + exportExtension(a.export.format)
	if snap, ok := a.liveSession(); ok && strings.TrimSpace(snap.Cwd) != "" {
		return filepath.Join(snap.Cwd, name)
	}
	return name
}

// exportToClipboard 把内容写进剪贴板，并在面板里给字符数回执或失败原因。
func (a *App) exportToClipboard(c *gui.Context, content string) {
	st := &a.export
	st.busy = true
	st.failure = ""
	st.receipt = ""
	c.WriteClipboardThen(content, func(err error) {
		st.busy = false
		if err != nil {
			st.failure = "复制失败：" + err.Error()
			return
		}
		st.receipt = fmt.Sprintf("已复制到剪贴板 · %d 个字符", utf8.RuneCountInString(content))
	})
}

// startExportFile 走系统保存面板；宿主没给面板时退回手输路径。
func (a *App) startExportFile(content string) {
	st := &a.export
	suggested := a.exportSuggestedPath()
	// 先把上一次的路径清掉：格式换了以后旧扩展名不该被沿用。
	st.path = suggested
	st.failure = ""
	st.receipt = ""
	save := a.saveFile
	if save == nil {
		st.manual = true
		st.manualReason = exportManualHint
		return
	}
	st.busy = true
	go func() {
		path, err := save(suggested)
		outcome := exportFileOutcome{suggested: suggested}
		switch {
		case err != nil:
			outcome.manualReason = "系统保存面板不可用（" + err.Error() + "），请直接输入保存路径。"
		case path == "":
			outcome.cancelled = true
		default:
			outcome.path = path
			outcome.bytes, outcome.err = writeExportFile(path, content)
		}
		a.deliverExportOutcome(outcome)
	}()
}

// exportToManualFile 写手输路径的文件。
func (a *App) exportToManualFile(content string) {
	path := strings.TrimSpace(a.export.path)
	if path == "" {
		a.export.failure = "没有填写保存路径"
		return
	}
	st := &a.export
	st.busy = true
	st.failure = ""
	st.receipt = ""
	go func() {
		bytes, err := writeExportFile(path, content)
		a.deliverExportOutcome(exportFileOutcome{path: path, bytes: bytes, err: err})
	}()
}

// deliverExportOutcome 把后台结果放进槽位；下一帧在渲染线程上应用。
func (a *App) deliverExportOutcome(outcome exportFileOutcome) {
	a.exportMu.Lock()
	a.exportResults = append(a.exportResults, outcome)
	a.exportMu.Unlock()
	if a.changed != nil {
		a.changed()
	}
}

// applyExportResults 在渲染线程上落后台结果：取消不改状态，面板打不开退回手输
// 路径，写失败留在面板里给原因，成功收起面板并在标题行给字节数回执。
func (a *App) applyExportResults() {
	a.exportMu.Lock()
	results := a.exportResults
	a.exportResults = nil
	a.exportMu.Unlock()
	for _, outcome := range results {
		st := &a.export
		st.busy = false
		switch {
		case outcome.cancelled:
			// 取消不是失败：回到面板，按钮恢复。
		case outcome.manualReason != "":
			st.manual = true
			st.manualReason = outcome.manualReason
			if st.path == "" {
				st.path = outcome.suggested
			}
		case outcome.err != nil:
			st.failure = "导出失败：" + outcome.err.Error()
		default:
			st.receipt = fmt.Sprintf("已导出到 %s · %d 字节", outcome.path, outcome.bytes)
			a.headerNote = st.receipt
			a.exported = outcome.path
			a.closeExportPanel()
		}
	}
}

// writeExportFile 写文件并返回字节数；"~" 展开成用户主目录，其余路径原样用。
func writeExportFile(rawPath, content string) (int64, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return 0, fmt.Errorf("没有填写保存路径")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return 0, fmt.Errorf("读不到用户主目录：%v", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if info, err := os.Stat(dir); err != nil {
			return 0, fmt.Errorf("目录不存在：%s", dir)
		} else if !info.IsDir() {
			return 0, fmt.Errorf("不是目录：%s", dir)
		}
	}
	data := []byte(content)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}

// firstLines 取文本的前 n 行（行尾的换行不计），用于预览。
func firstLines(text string, n int) string {
	if n <= 0 {
		return ""
	}
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// turnPreview 是一轮的单行预览：用户消息首行，没有时取助手首行。
func turnPreview(turn exportTurn) string {
	text := turn.User
	if strings.TrimSpace(text) == "" && len(turn.Assistant) > 0 {
		text = turn.Assistant[0]
	}
	text = strings.TrimSpace(strings.SplitN(text, "\n", 2)[0])
	runes := []rune(text)
	if len(runes) > 48 {
		text = string(runes[:48]) + "…"
	}
	if text == "" {
		return "（空回合）"
	}
	return text
}

// nonEmpty 取第一个非空串。
func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// exportTextButton 是一枚文字按钮；reached 为真时置灰（已达状态）。
func exportTextButton(c *gui.Context, text, tip string, reached bool) bool {
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Height(rowRegular).Padding(0, space1).
		Radius(radius).Label(text).Tooltip(tip).Disabled(reached)
	ink := t.TextMuted
	if reached {
		b = b.Opacity(noteDisabledAlpha)
	} else if b.Hovered() {
		ink = t.Text
	}
	b.Children(func() {
		typeSecondary.style(t, gui.Text(c, text)).TextColor(ink).SingleLine()
	})
	return b.Clicked() && !reached
}

// screenButtonDisabledUntil 是 screenButton 的禁用版：disabled 为真时画成
// 可见但不可点（透明度与其它禁用入口一致），点击一律无效。
func screenButtonDisabledUntil(c *gui.Context, text, tip, label string, disabled bool, height float32) bool {
	if disabled {
		screenTextButtonDisabled(c, text, tip, height)
		return false
	}
	return screenButton(c, text, label, tip, false, height)
}
