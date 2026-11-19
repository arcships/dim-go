package ui

import (
	"context"
	"fmt"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 输入区底部栈里的两块面板：排队消息（IN-35–IN-42）与 Shell 运行（IN-45–IN-52）。
//
// 栈的顺序取 requirements-main §4.3：活动条 → 排队消息 → 权限/澄清面板 →
// Shell 运行 → 输入块 → 状态条。本文件只画后两块中排在输入块之上、可由事实
// 支撑的部分：
//
//   - 队列的条目、条数与投递方式来自投影的未消费输入（SessionSnapshot.Queued）；
//   - 命令、模式、状态、输出、截断、退出码与原因来自 shell.run 的回执
//     （SessionSnapshot.ShellRun）。
//
// 缺写通道的动作（队列排序、撤回、编辑保存、转向）保留入口但置灰，原因写在
// tooltip 与读屏名里（DEC 口径，见 unavailable.go）。

const (
	// queuePanelMaxRows 是队列面板一次画出的最多行数；再多就内部滚动。
	queuePanelMaxRows = 5
	// queueRowHeight 是队列行的高度（参考行高 17.5，取行档 24）。
	queueRowHeight = 24
	// shellPanelPad 是 Shell 面板的内边距（与输入卡同一档）。
	shellPanelPad = space3
	// shellOutputMaxHeight 是输出块的最大高：超出后内部滚动。
	shellOutputMaxHeight = 132
)

// 排队消息与命令面板的新增原因文案（集中在 unavailable.go 之外的两条，因为
// 它们只属于底部栈；理由见 decisions.md 的「缺通道就置灰」）。
const (
	noteQueueReorder    = "服务端没有提供排队消息排序（本版不做）"
	noteQueueWithdraw   = "服务端没有提供排队消息撤回（本版不做）"
	noteQueueEdit       = "服务端没有提供排队消息编辑（本版不做）"
	noteQueueSteer      = "服务端没有提供插话写入（本版不做）"
	noteShellOutput     = "服务端没有回传这次运行的输出"
	noteCompactionRetry = "服务端没有提供摘要重取（本版不做）"
)

// 新增的图标形状：三点竖排把手、垃圾桶、列表（与 lucide 同一 24 网格、1.5 描边）。
var (
	iconGripVertical = mustSVG(`<circle cx="9.75" cy="5.25" r="1.125"/><circle cx="14.25" cy="5.25" r="1.125"/><circle cx="9.75" cy="12" r="1.125"/><circle cx="14.25" cy="12" r="1.125"/><circle cx="9.75" cy="18.75" r="1.125"/><circle cx="14.25" cy="18.75" r="1.125"/>`)
	iconTrash        = mustSVG(`<path d="M4.5 6.75h15M9.75 6.75V4.5h4.5v2.25M6.75 6.75l.75 12a1.5 1.5 0 0 0 1.5 1.5h6a1.5 1.5 0 0 0 1.5-1.5l.75-12"/>`)
	iconList         = mustSVG(`<path d="M8.25 6.75h12M8.25 12h12M8.25 17.25h12M3.75 6.75h.008M3.75 12h.008M3.75 17.25h.008"/>`)
	iconCornerDown   = mustSVG(`<path d="M6.75 4.5v9a3 3 0 0 0 3 3h9"/><path d="m15.75 13.5 3 3-3 3"/>`)
)

// resetPanelViewState 重置底部栈与时间线新增块的可丢视图状态。
func (a *App) resetPanelViewState() {
	a.queueCollapsed = false
	a.queueEditing = ""
	a.queueDraft = ""
	a.queueNotice = ""
	a.shellCollapsed = false
	a.shellClosed = ""
	a.inlineEditRow = -1
	a.inlineDraft = ""
	a.inlineSaved = map[int]string{}
	a.inlineNotice = ""
	a.userExpanded = map[int]bool{}
	a.artifactsMore = map[string]bool{}
	a.artifactMenus = map[string]*bool{}
	a.termTabs = nil
	a.termActive = 0
	a.compactionOpen = map[string]*bool{}
	a.demoQueued = nil
	a.demoShellRun = nil
	a.demoCompaction = nil
}

// applyComposerPanelVariant 把底部栈与时间线的走查变体落到视图状态（截图用）。
func (a *App) applyComposerPanelVariant() {
	switch a.variant {
	case VariantQueuePanel:
		a.state = StateRunning
		a.demoQueued = sampleQueued()
	case VariantShellPanel:
		a.demoShellRun = sampleShellRun()
	case VariantInlineEdit:
		a.inlineEditRow = len(a.rounds) - 1
		a.inlineDraft = "把排队的实现讲清楚"
	case VariantCompaction:
		open := true
		a.compactionOpen["1"] = &open
		a.demoCompaction = sampleCompaction()
	case VariantArtifacts:
		if len(a.rounds) > 0 {
			a.rounds[0].artifacts = sampleArtifacts()
			*a.blockOf("artifacts:0", false) = true
		}
		a.artifactsMore["artifacts:0"] = false
	case VariantArtifactsMenu:
		if len(a.rounds) > 0 {
			a.rounds[0].artifacts = sampleArtifacts()
			*a.blockOf("artifacts:0", false) = true
		}
		a.artifactsMore["artifacts:0"] = false
		*a.artifactMenuOpen("artifacts:0/设计稿.pdf") = true
	case VariantReviewMenu:
		a.rightOpen, a.rightTab = true, tabChanges
		*a.rp().rowMenuPtr("change:" + sampleReviewPath()) = true
	case VariantTermLayers:
		a.rightOpen, a.rightTab = true, 4
		a.ensureTermTabs()
		a.syncActiveTerminal()
		a.termTabs = append(a.termTabs, termTab{
			title: "终端 2", state: termExited,
			lines: []termLine{
				{text: "$ git status", color: -1},
				{text: "nothing to commit, working tree clean", color: 2},
			},
		})
		a.termActive = 1
		a.termState, a.termLines = a.termTabs[1].state, a.termTabs[1].lines
	case VariantSubagentFailed:
		a.rightOpen, a.rightTab = true, tabSubagent
		a.panels.subagentFailed = true
	}
}

// sampleQueued 是走查用的三条排队消息（截图与测试；接上 Source 后不读）。
func sampleQueued() []QueuedMessageSnapshot {
	return []QueuedMessageSnapshot{
		{ID: "q1", Mode: "queue", Text: "再补一条错误分支的测试"},
		{ID: "q2", Mode: "steer", Text: "先把编译过的包跑一遍"},
		{ID: "q3", Mode: "queue", Text: "把排队面板的文案与参考对齐，超长的时候单行截断"},
	}
}

// sampleShellRun 是走查用的一次命令运行（截图与测试）。
func sampleShellRun() *ShellRunSnapshot {
	return &ShellRunSnapshot{
		RunID:       "run-sample",
		Command:     "go test ./internal/ui -run TestComposerPanels -count=1",
		Mode:        "record",
		Status:      "failed",
		Output:      "ok   github.com/arcships/dimagent-mygo/internal/ui 0.31s\n--- FAIL: TestComposerPanels\npanic: 断言失败",
		ExitCode:    1,
		HasExitCode: true,
		Reason:      "命令以非零状态退出",
		Recorded:    true,
		Truncated:   true,
	}
}

// sampleCompaction 是走查用的一次上下文压缩（截图与测试）。
func sampleCompaction() *CompactionSnapshot {
	return &CompactionSnapshot{
		Status:   "completed",
		Summary:  "前 12 轮讨论了排队面板与命令面板：队列来自未消费输入，命令输出来自 shell.run 回执。",
		Strategy: "summarize",
	}
}

// sampleReviewPath 是改动右键菜单走查态里那一条文件的路径。
func sampleReviewPath() string { return "internal/ui/composer.go" }

// sampleArtifacts 是走查用的五条产物（超过 3 行才出现「全部 5 个产物」）。
func sampleArtifacts() []artifact {
	return []artifact{
		{name: "设计稿.pdf"},
		{name: "排队面板说明.md"},
		{name: "截图-队列.png"},
		{name: "构建日志.txt"},
		{name: "这次导出的一份名字很长很长的报告-2026-11-19.pdf"},
	}
}

// ---- 排队消息面板 ----

// queuedMessages 取排队消息：连着 Source 时读事实，否则读样例走查数据。
func (a *App) queuedMessages() []QueuedMessageSnapshot {
	if snap, ok := a.liveSession(); ok {
		return snap.Queued
	}
	if a.wired() {
		return nil
	}
	return a.demoQueued
}

// queuePanel 是排队消息面板（IN-35）：标题计数 + 行；位于输入块之上。
func (a *App) queuePanel(c *gui.Context) {
	queued := a.queuedMessages()
	if len(queued) == 0 {
		return
	}
	t := c.Theme()
	gui.Row(c).Justify(gui.Center).Children(func() {
		gui.Column(c).WidthPercent(100).MaxWidth(composerMaxWidth).
			Background(t.SurfaceHover).Radius(radiusWindowCard).Clip().
			Label("排队消息面板").Role(gui.RoleList).Children(func() {
			a.queueHeader(c, len(queued))
			if a.queueCollapsed {
				return
			}
			a.queueRows(c, queued)
		})
	})
}

// queueHeader 是面板标题行：列表图标 + 「{n} 条消息」 + 展开指示（IN-36）。
// 点标题折叠 / 展开；折叠态不占内容高。
func (a *App) queueHeader(c *gui.Context, count int) {
	t := c.Theme()
	title := fmt.Sprintf("%d 条消息", count)
	head := gui.ButtonBase(c).Height(rowRegular).Shrink(0).Gap(space1).
		Padding(space1, space3).Justify(gui.Start).AlignItems(gui.Center).
		Radius(radius).Cursor(gui.CursorPointer).Label(title).
		Role(gui.RoleDisclosure).Expanded(!a.queueCollapsed)
	progress := head.AnimateTo("queue-arrow", b2f(!a.queueCollapsed), timelineArrow)
	head.Children(func() {
		iconGlyph(c, iconList, t.TextMuted, "queue")
		typeMicro.style(t, gui.Text(c, title)).TextColor(t.TextMuted).SingleLine()
		chevronLabeled(c, t, progress, "排队消息展开指示")
	})
	if head.Clicked() {
		a.queueCollapsed = !a.queueCollapsed
	}
}

// queueRows 是面板内容：每条一行；超过上限时内部滚动。
func (a *App) queueRows(c *gui.Context, queued []QueuedMessageSnapshot) {
	rows := queued
	if len(rows) > queuePanelMaxRows {
		rows = rows[:queuePanelMaxRows]
	}
	body := func() {
		gui.Column(c).Gap(0).Padding(0, space1, space2, space1).Children(func() {
			for _, item := range rows {
				a.queueRow(c, item)
			}
		})
	}
	if len(queued) > queuePanelMaxRows {
		gui.Scroll(c).Shrink(0).MaxHeight(queueRowHeight * (queuePanelMaxRows + 1)).
			Label("排队消息列表").Children(body)
		return
	}
	body()
}

// queueRow 是一条排队消息（IN-37 / IN-38）：转向中带旋转与状态词；待发只有
// 正文。悬停出现把手、编辑、移除与转向（IN-39–IN-42）。
func (a *App) queueRow(c *gui.Context, item QueuedMessageSnapshot) {
	t := c.Theme()
	steering := item.Mode == "steer"
	running := a.liveRunning()
	row := gui.Row(c).Height(queueRowHeight).Grow(1).Gap(space1).
		Padding(0, space1).AlignItems(gui.Center)
	if row.Hovered() || a.queueEditing == item.ID {
		row = row.Background(t.SurfaceHover)
	}
	text := item.Text
	if strings.TrimSpace(text) == "" {
		text = "附件"
	}
	label := "排队消息：" + text
	if steering {
		label = "排队消息（正在转向）：" + text
	}
	row = row.Label(label)
	row.Children(func() {
		if steering {
			gui.Spinner(c).Label("正在转向")
			typeMicro.style(t, gui.Text(c, "正在转向")).TextColor(t.TextMuted).SingleLine()
		}
		typeSecondary.style(t, gui.Text(c, text)).TextColor(t.Text).Grow(1).SingleLine()
		if !(row.Hovered() || a.queueEditing == item.ID) {
			return
		}
		iconButton(c, iconButtonSpec{
			icon: iconGripVertical, label: "拖动排序（" + noteQueueReorder + "）",
			tip: noteQueueReorder, size: rowCompact, disabled: true,
		})
		if iconButton(c, iconButtonSpec{
			icon: iconPencil, label: "编辑排队消息（" + text + "）",
			tip: "编辑排队消息", size: rowCompact,
		}) {
			a.beginQueuedEdit(item)
		}
		iconButton(c, iconButtonSpec{
			icon: iconTrash, label: "移除排队消息（" + noteQueueWithdraw + "）",
			tip: noteQueueWithdraw, size: rowCompact, disabled: true,
		})
		if running {
			label := "转向（" + noteQueueSteer + "）"
			b := gui.ButtonBase(c).Height(rowCompact).Padding(0, space1).Radius(radius).
				Label(label).Tooltip(noteQueueSteer).Disabled(true).Opacity(0.5)
			b.Children(func() {
				typeMicro.style(t, gui.Text(c, "转向")).TextColor(t.Accent).SingleLine()
			})
		}
	})
}

// beginQueuedEdit 把排队消息的正文放进输入块（IN-436）：输入卡里出现「正在编辑
// 排队消息」条，发送槽变成保存。
func (a *App) beginQueuedEdit(item QueuedMessageSnapshot) {
	a.queueEditing = item.ID
	a.queueDraft = item.Text
	a.draft = item.Text
	a.queueNotice = ""
}

// cancelQueuedEdit 退出排队消息编辑，草稿回到进入编辑前（空）。
func (a *App) cancelQueuedEdit() {
	a.queueEditing = ""
	a.queueDraft = ""
	a.draft = ""
	a.queueNotice = ""
}

// savingQueuedEdit 报告当前发送槽是不是「保存排队消息」。
func (a *App) savingQueuedEdit() bool { return a.queueEditing != "" }

// saveQueuedEdit 保存排队消息的修改。
//
// 协议没有排队消息的编辑或撤回方法：连着真实服务端时不写任何东西，把原因留在
// 输入卡里（IN-439 的「已开始执行，无法修改」同形处理），草稿保留。样例模式
// 里就地替换样例队列，走查与测试能走完整个交互。
func (a *App) saveQueuedEdit() {
	if a.wired() || a.intents != nil {
		a.queueNotice = noteQueueEdit
		return
	}
	for i := range a.demoQueued {
		if a.demoQueued[i].ID == a.queueEditing {
			a.demoQueued[i].Text = strings.TrimSpace(a.draft)
		}
	}
	a.queueEditing = ""
	a.queueDraft = ""
	a.draft = ""
	a.queueNotice = ""
}

// queuedEditBanner 是输入卡里的编辑条（IN-23）：说明正在编辑哪条排队消息，
// 给「取消」，并在缺通道时说明保存为什么不可用。
func (a *App) queuedEditBanner(c *gui.Context) {
	if a.queueEditing == "" {
		return
	}
	t := c.Theme()
	gui.Row(c).Shrink(0).Gap(space1).AlignItems(gui.Center).
		Label("正在编辑排队消息").Children(func() {
		iconGlyph(c, iconPencil, t.Accent, "editing-queue")
		typeMicro.style(t, gui.Text(c, "正在编辑排队消息")).TextColor(t.Accent).
			Grow(1).SingleLine()
		if iconButton(c, iconButtonSpec{
			icon: iconXMark, label: "取消编辑排队消息", tip: "取消编辑", size: rowCompact,
		}) {
			a.cancelQueuedEdit()
		}
	})
	if a.queueNotice != "" {
		gui.Row(c).Shrink(0).Children(func() {
			typeMicro.style(t, gui.Text(c, a.queueNotice)).TextColor(t.Danger).SingleLine()
		})
	}
}

// ---- Shell 运行面板 ----

// shellRunFacts 取最近一次命令运行：连着 Source 时读事实，否则读样例。
func (a *App) shellRunFacts() *ShellRunSnapshot {
	if snap, ok := a.liveSession(); ok {
		return snap.ShellRun
	}
	if a.wired() {
		return nil
	}
	return a.demoShellRun
}

// shellRunPanel 是 Shell 运行面板（IN-45–IN-52）：标题行（命令、模式、状态、
// 停止 / 关闭）+ 可折叠的输出与脚注。
func (a *App) shellRunPanel(c *gui.Context) {
	run := a.shellRunFacts()
	if run == nil || a.shellClosed == run.RunID {
		return
	}
	t := c.Theme()
	gui.Row(c).Justify(gui.Center).Children(func() {
		gui.Column(c).WidthPercent(100).MaxWidth(composerMaxWidth).
			Background(t.SurfaceHover).Radius(radiusWindowCard).Clip().Padding(shellPanelPad).
			Label("Shell 运行面板").Role(gui.RoleGroup).Children(func() {
			a.shellRunHeader(c, run)
			if a.shellCollapsed {
				return
			}
			a.shellRunBody(c, run)
		})
	})
}

// shellRunHeader 是标题行（IN-46–IN-50）：终端图标、单行命令、模式标签、状态
// 文本，以及停止（运行中）或关闭（结束后）。都在同一行里，命令占剩余宽。
func (a *App) shellRunHeader(c *gui.Context, run *ShellRunSnapshot) {
	t := c.Theme()
	running := run.Status == "running"
	gui.Row(c).Shrink(0).Gap(space2).AlignItems(gui.Center).Children(func() {
		title := gui.ButtonBase(c).Grow(1).Height(rowRegular).Gap(space1).
			Justify(gui.Start).AlignItems(gui.Center).Radius(radius).
			Label("命令：" + run.Command).Role(gui.RoleDisclosure).Expanded(!a.shellCollapsed)
		progress := title.AnimateTo("shell-arrow", b2f(!a.shellCollapsed), timelineArrow)
		title.Children(func() {
			iconGlyph(c, iconTerminal, t.TextMuted, "shell-run")
			typeCode.style(t, gui.Text(c, run.Command)).TextColor(t.Text).Grow(1).SingleLine()
			chevronLabeled(c, t, progress, "命令面板展开指示")
		})
		if title.Clicked() {
			a.shellCollapsed = !a.shellCollapsed
		}
		if mode := shellModeLabel(run.Mode); mode != "" {
			gui.Row(c).Shrink(0).Height(rowCompact).Padding(0, space1).Radius(radiusSmall).
				Background(t.Surface).AlignItems(gui.Center).
				Label("模式：" + mode).Children(func() {
				typeMicro.style(t, gui.Text(c, mode)).TextColor(t.TextMuted).SingleLine()
			})
		}
		gui.Row(c).Shrink(0).Gap(space1).AlignItems(gui.Center).
			Label("状态：" + shellStatusText(run.Status)).Children(func() {
			if running {
				gui.Spinner(c).Label("命令运行中")
			}
			color := t.TextMuted
			if run.Status == "failed" || run.Status == "stopped" {
				color = t.Warning
			}
			typeMicro.style(t, gui.Text(c, shellStatusText(run.Status))).TextColor(color).
				SingleLine()
		})
		if running {
			if iconButton(c, iconButtonSpec{
				icon: iconStop, label: "停止命令", tip: "停止", size: rowCompact,
			}) {
				a.stopShellPanelRun(run)
			}
			return
		}
		if iconButton(c, iconButtonSpec{
			icon: iconXMark, label: "关闭命令面板", tip: "关闭", size: rowCompact,
		}) {
			a.shellClosed = run.RunID
		}
	})
}

// stopShellPanelRun 是面板上的「停止」：有运行标识就发一次停止；样例里把状态
// 收到已停止（不发任何请求）。
func (a *App) stopShellPanelRun(run *ShellRunSnapshot) {
	if a.intents != nil && run.RunID != "" {
		a.dispatch("停止命令", func(ctx context.Context) error {
			return a.intents.StopShell(ctx, run.RunID)
		}, func() { a.lastShellRun = "" })
		return
	}
	if a.demoShellRun != nil && a.demoShellRun.RunID == run.RunID {
		a.demoShellRun.Status = "stopped"
	}
}

// shellRunBody 是面板内容（IN-51 / IN-52）：等宽输出块与逐条脚注。
func (a *App) shellRunBody(c *gui.Context, run *ShellRunSnapshot) {
	t := c.Theme()
	if strings.TrimSpace(run.Output) == "" {
		gui.Row(c).Shrink(0).Children(func() {
			typeMicro.style(t, gui.Text(c, noteShellOutput)).TextColor(t.TextMuted).SingleLine()
		})
	} else {
		lines := strings.Split(strings.TrimRight(run.Output, "\n"), "\n")
		gui.Box(c).Shrink(0).MaxHeight(shellOutputMaxHeight).Padding(space1, space2).
			Radius(radiusSmall).Background(t.Surface).Children(func() {
			gui.Scroll(c).Shrink(0).Children(func() {
				gui.Column(c).Gap(0).Label("命令输出").Children(func() {
					for _, line := range lines {
						typeCode.style(t, gui.Text(c, interpretLine(line))).
							TextColor(t.Text).SingleLine()
					}
				})
			})
		})
	}
	for _, note := range shellFootnotes(run) {
		gui.Row(c).Shrink(0).Label("命令脚注").Children(func() {
			typeMicro.style(t, gui.Text(c, note.text)).TextColor(note.color(t)).SingleLine()
		})
	}
}

// shellFootnote 是一条脚注：文案与配色由条件决定（IN-52 的六条）。
type shellFootnote struct {
	text string
	tone string // muted / warn
}

func (f shellFootnote) color(t *gui.Theme) gui.Color {
	if f.tone == "warn" {
		return t.Warning
	}
	return t.TextMuted
}

// shellFootnotes 按事实逐条给出脚注：跳过数、输出已截断、进程退出码、错误、
// 原因、录制结果。没有事实的那条不出现（不留空行与孤立冒号）。
func shellFootnotes(run *ShellRunSnapshot) []shellFootnote {
	out := make([]shellFootnote, 0, 6)
	if run.HasExitCode {
		tone := "muted"
		if run.ExitCode != 0 {
			tone = "warn"
		}
		out = append(out, shellFootnote{fmt.Sprintf("进程退出码 %d", run.ExitCode), tone})
	}
	if run.Truncated {
		out = append(out, shellFootnote{"输出已截断", "muted"})
	}
	if strings.TrimSpace(run.Error) != "" {
		out = append(out, shellFootnote{"错误：" + run.Error, "warn"})
	}
	if strings.TrimSpace(run.Reason) != "" {
		out = append(out, shellFootnote{"原因：" + run.Reason, "muted"})
	}
	if run.Recorded {
		out = append(out, shellFootnote{"录制结果已写入时间线", "muted"})
	}
	return out
}

// shellModeLabel 把回执里的模式换成界面词；未知模式不画标签（IN-503）。
func shellModeLabel(mode string) string {
	switch mode {
	case "record":
		return "记录"
	case "private":
		return "私有"
	}
	return ""
}

// shellStatusText 是面板的状态文本（IN-48）：运行中 / 已完成 / 失败 / 已停止，
// 认不出的状态写「状态未知」，不假装还在跑。
func shellStatusText(status string) string {
	switch status {
	case "running":
		return "运行中"
	case "completed":
		return "已完成"
	case "failed":
		return "失败"
	case "stopped":
		return "已停止"
	}
	return "状态未知"
}
