package ui

import (
	"context"
	"fmt"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 终端是右栏的第五个标签（U1-ia R5 的职责句与 U1 E6「终端会话」都在右栏；
// S25 的输入区 `!` 前缀是另一件事）。
//
// 形态按参考产品的右栏终端（requirements-panels.md §4，T-01–T-21）：
// 工具条（终端标签 + 关闭进程 + 新建终端 + 关闭终端）、深色画布、提示行
// `(base) → 目录`、画布右键菜单（复制选中内容 / 粘贴 / 清屏 / 新建终端 / 关闭进程）。
//
// 画布是固定的深底，不随主题变白（R-TM-063）；输出里的 ANSI 片段只取颜色，
// 不解析光标移动。
//
// 事实来源：连着 Source 时终端是本机 pty（internal/terminal，经适配器起停与
// 读写），状态与输出行每帧从 TerminalSnapshot 读；输入按回车写回 pty。样例
// 只在 -demo（没有 Source）时出现。

// termState 是终端的四种状态。
type termState int

const (
	termIdle termState = iota
	termRunning
	termExited
	termDisconnected
)

// termExit 是终端进程的结束事实（T-11）：退出码与终止信号都可能没有（进程还
// 在跑，或没有读到退出状态）。界面按这三项决定提示行的写法，不把「不知道」
// 写成「代码 0」。
type termExit struct {
	known    bool
	code     int
	signaled bool
	signal   int
}

// termLine 是输出区的一行：颜色是 ANSI 前 8 色的下标（-1 表示默认色）。
type termLine struct {
	text  string
	color int
}

// 终端面板的尺寸。
const (
	// termToolbarHeight 是工具条高：参考的终端标签 25 + 上下留白（T-01）。
	termToolbarHeight = rowRegular + 2*space1
	// termEnvDefault 是样例里的 shell 环境名前缀；连着 Source 时没有这一项。
	termEnvDefault = "(base)"
	// termClosedNotice 是进程被关掉后内容区的提示行（T-10，全角方括号）。
	termClosedNotice = "［进程已关闭］"
	// termExitedNotice 是没有退出状态可读时的提示行（T-11 / R-TM-093）：不带空的
	// 「代码 」段。
	termExitedNotice = "［进程已退出］"
	// termEmptyText 是画布还没有输出时的说明行。
	termEmptyText = "还没有输出"
	// termNotStartedText 是连着 pty 但终端还没起来时的说明行：点「新建终端」
	// 会在会话的工作目录里启动本机 shell。
	termNotStartedText = "还没有启动终端，点「新建终端」启动本机 shell"
	// termStartingText 是启动请求在途时的说明行（§1.2.3 状态 5）。
	termStartingText = "正在启动终端…"
	// termNoEntryText 是这次运行没有终端写入口时的说明（界面测试 / 单屏截图）。
	termNoEntryText = "当前运行方式没有终端入口"
	// termNoCwdText 是会话没有工作目录时的原因：不拿工作区根顶替。
	termNoCwdText = "这个会话没有工作目录"
	// termReadFailedText 是读不到输出的结束提示（§1.2.3 状态 10）：进程可能还在
	// 跑，但没有新行会进来，所以不再给输入框。
	termReadFailedText = "［读不到终端输出］"
	// 入口不可用时的原因（与 §1.2.2 的文案表逐字一致）。禁用不写原因就成了
	// 没有说明的灰项，所以每个入口都带着自己那一句。
	termRunningNote       = "终端已在运行"
	termStartingNote      = "正在启动终端"
	termStartingCloseNote = "正在启动终端，请稍候"
	termNoProcessNote     = "没有运行中的终端"
	termExitedCloseNote   = "进程已退出"
	termClosedNote        = "进程已关闭"
	termNoOutputNote      = "还没有输出"
	// termCaretWidth / termCaretHeight 是块状光标的尺寸（参考约 9×16）。
	termCaretWidth  = 9
	termCaretHeight = 16
)

// iconEraser 是「清屏」的橡皮擦图形（lucide eraser）。
var iconEraser = mustSVG(`<path d="m7.5 20.25 11.25-11.25a2.25 2.25 0 0 0 0-3.18l-2.32-2.32a2.25 2.25 0 0 0-3.18 0L4.5 12.75a2.25 2.25 0 0 0 0 3.18l4.32 4.32"/><path d="M7.5 20.25h11.25"/>`)

// 终端画布的固定配色：不跟随主题（R-TM-063），取参考产品的深底与终端八色。
var (
	termCanvasBg    = gui.Hex("#1a1a1a")
	termCanvasText  = gui.Hex("#e4e4e4")
	termCanvasMuted = gui.Hex("#8a8a8a")
	termCanvasShell = gui.Hex("#6cc07a")
	termCanvasArrow = gui.Hex("#4ec9b0")
	termAnsiRed     = gui.Hex("#e06c75")
	termAnsiGreen   = gui.Hex("#98c379")
	termAnsiYellow  = gui.Hex("#e5c07b")
	termAnsiBlue    = gui.Hex("#61afef")
)

// termSamples 返回一组样例输出：ANSI 颜色、长行、多行错误。
func termSamples(state termState) []termLine {
	lines := []termLine{
		{text: "$ go test ./internal/ui -run TestSender", color: -1},
		{text: "ok   github.com/arcships/dimagent-mygo/internal/ui", color: 2},
		{text: "$ git status --short", color: -1},
		{text: " M internal/ui/composer.go", color: 1},
		{text: " M internal/ui/timeline.go", color: 1},
		{text: "?? internal/ui/terminal.go", color: 3},
		{text: "$ go vet ./...", color: -1},
		{
			text: "vet: internal/ui/composer.go:213:31: cannot use spec.text (variable of type " +
				"string) as type *string in argument to textIconButton — 这一行故意很长，用来检查长行在窄栏里的折行",
			color: 4,
		},
		{text: "exit status 1", color: 1},
	}
	// ANSI 的三类控制序列放在靠前的位置，截图里能直接看到处理结果：
	// \r 覆盖同一行（进度条）、\b 退格修正、\x1b[K 清行。
	ansi := []termLine{
		{text: "下载中 10%\r下载中 55%\r下载中 100% 完成", color: 3},
		{text: "版本 v1.O\b0", color: -1},
		{text: "临时输出：不要了\x1b[K", color: -1},
	}
	lines = append(append(lines[:2:2], ansi...), lines[2:]...)
	switch state {
	case termIdle:
		return lines[:1]
	case termRunning:
		return append(lines[:4], termLine{text: "$ npm run build", color: -1},
			termLine{text: "build starting…", color: -1})
	case termDisconnected:
		return append(lines[:4], termLine{text: "connection lost", color: 1})
	}
	return lines
}

// terminalView 取终端的状态与输出行：连着 Source 时读快照，否则用样例。
//
// 第三个返回值是结束事实（T-11）：只有连着 pty 且进程真结束了才有；样例里的
// 「已退出」没有退出状态，界面据此只写「进程已退出」。第四个返回值表示输出
// 读出失败（读不到新行，但进程可能还在跑）。
func (a *App) terminalView() (termState, []termLine, termExit, bool, bool) {
	snapshot, ok := a.liveTerminal()
	if !ok {
		return a.termState, a.termLines, a.termExit, false, false
	}
	state := termIdle
	switch {
	case snapshot.Running:
		state = termRunning
	case snapshot.Exited:
		state = termExited
	}
	lines := make([]termLine, 0, len(snapshot.Lines))
	for _, line := range snapshot.Lines {
		lines = append(lines, termLine{text: line, color: -1})
	}
	exit := termExit{}
	if snapshot.Exited {
		exit = termExit{
			known: snapshot.ExitKnown, code: snapshot.ExitCode,
			signaled: snapshot.Signaled, signal: snapshot.Signal,
		}
	}
	return state, lines, exit, snapshot.ReadFailed, true
}

// termLive 表示终端页接的是本机 pty：连着 Source 且有写入口。
//
// 只有 Source 没有写入口（界面测试 / 单屏截图）时不算接线：这时「新建终端」
// 置灰并写原因，不退回到样例——样例只在 -demo（没有 Source）时出现。
func (a *App) termLive() bool { return a.wired() && a.intents != nil }

// termPhase 是终端页的展示状态（§1.2.1）：互斥，按 terminalPhase 里的顺序取第
// 一条成立的。9（关闭失败）不单独成一相：画面留在原状态，原因进全局错误行。
type termPhase int

const (
	termPhaseNoSession termPhase = iota
	termPhaseNoCwd
	termPhaseNoEntry
	termPhaseNotStarted
	termPhaseStarting
	termPhaseRunning
	termPhaseExited
	termPhaseClosed
	termPhaseReadFailed
)

// termBlocker 是「这里根本起不了终端」的前置（状态 1–3）：没有会话、会话没有
// 工作目录、这次运行没有写入口。原因只来自事实，不拿工作区根顶替。
func (a *App) termBlocker() (termPhase, bool) {
	if !a.wired() {
		return 0, false
	}
	snapshot, ok := a.liveSession()
	if !ok {
		return termPhaseNoSession, true
	}
	if strings.TrimSpace(snapshot.Cwd) == "" {
		return termPhaseNoCwd, true
	}
	if a.intents == nil {
		return termPhaseNoEntry, true
	}
	return 0, false
}

// terminalPhase 判当前激活层处在哪个状态。
//
// 顺序（§1.2.1）：该层关闭标记 → 全局前置 → 运行中（读失败先于正常）→ 启动中
// → 已退出 → 未启动。关闭标记先于前置：关掉的页签不因会话切走而变回「没启动」；
// 前置先于进程事实：没有写入口的一帧不画一个"看起来能输入"的终端。
func (a *App) terminalPhase() termPhase {
	a.ensureTermTabs()
	if a.termTabs[a.termActive].closed {
		return termPhaseClosed
	}
	if phase, blocked := a.termBlocker(); blocked {
		return phase
	}
	if !a.wired() {
		// 样例（-demo）：不接事实，按样例的四种状态映射（§1.2.6）。
		switch a.termState {
		case termExited:
			return termPhaseExited
		case termDisconnected:
			return termPhaseReadFailed
		}
		return termPhaseRunning
	}
	snapshot, ok := a.liveTerminal()
	if ok && snapshot.Active && snapshot.Running {
		if snapshot.ReadFailed {
			return termPhaseReadFailed
		}
		return termPhaseRunning
	}
	if a.termStarting {
		return termPhaseStarting
	}
	if ok && snapshot.Active && snapshot.Exited {
		return termPhaseExited
	}
	return termPhaseNotStarted
}

// termPhaseFor 判某一层画布的状态。非激活层不参与会话与入口判定（它可能是
// 样例里的另一层）：只看它自己的关闭标记与样例状态。
func (a *App) termPhaseFor(tab *termTab) termPhase {
	a.ensureTermTabs()
	if tab == nil || tab == &a.termTabs[a.termActive] {
		return a.terminalPhase()
	}
	if tab.closed {
		return termPhaseClosed
	}
	switch tab.state {
	case termExited:
		return termPhaseExited
	case termDisconnected:
		return termPhaseReadFailed
	}
	return termPhaseRunning
}

// termAlive 表示现在有一个在跑的终端（运行中或读不到输出但进程还在）。
func (a *App) termAlive() bool {
	switch a.terminalPhase() {
	case termPhaseRunning, termPhaseReadFailed:
		return true
	}
	return false
}

// termStartNote 返回「新建终端」不能用的原因；空串表示可以启动。
//
// 工作目录只来自会话事实：没有打开的会话、会话没有工作目录都不启动（不猜一个
// 目录，也不拿工作区根顶替）；已经在跑的终端也不重复启动（一个会话一个实例）。
func (a *App) termStartNote() string {
	if !a.wired() {
		return ""
	}
	switch a.terminalPhase() {
	case termPhaseNoSession:
		return noteNoSessionID
	case termPhaseNoCwd:
		return termNoCwdText
	case termPhaseNoEntry:
		return termNoEntryText
	case termPhaseStarting:
		return termStartingNote
	case termPhaseRunning, termPhaseReadFailed:
		return termRunningNote
	}
	return ""
}

// termCloseNote 是页签内「关闭进程」不能用的原因；空串表示现在有进程可关。
func (a *App) termCloseNote() string {
	switch a.terminalPhase() {
	case termPhaseRunning, termPhaseReadFailed:
		return ""
	case termPhaseStarting:
		return termStartingNote
	case termPhaseExited:
		return termExitedCloseNote
	case termPhaseClosed:
		return termClosedNote
	}
	return termNoProcessNote
}

// termCloseTabNote 是「关闭终端」不能用的原因：启动请求在途时先别收页签（pty
// 可能刚起来，收掉会留下一个没有页签的进程）。其余状态都可以收起页签。
func (a *App) termCloseTabNote() string {
	if a.terminalPhase() == termPhaseStarting {
		return termStartingCloseNote
	}
	return ""
}

// termPasteNote 是右键「粘贴」不能用的原因：没有可写的提示行时，粘贴只会改一个
// 看不见的草稿。
func (a *App) termPasteNote() string {
	switch a.terminalPhase() {
	case termPhaseRunning:
		return ""
	case termPhaseStarting:
		return termStartingNote
	}
	return termNoProcessNote
}

// termOutputNote 是清屏 / 复制输出不能用的原因：画布上没有可见输出时，这两个
// 动作都是空转。关闭后画布只剩一句「进程已关闭」，保留的行不上屏，也就不算
// 有可见输出。
func (a *App) termOutputNote() string {
	if a.terminalPhase() == termPhaseClosed {
		return termNoOutputNote
	}
	for _, line := range a.termVisible() {
		if strings.TrimSpace(line.text) != "" {
			return ""
		}
	}
	return termNoOutputNote
}

// termTip 在入口不可用时把原因并进悬停提示（界面只说结果，不写内部术语）。
func termTip(label, note string) string {
	if note == "" {
		return label
	}
	return label + "：" + note
}

// termEmptyNoteFor 是某一层画布没有输出时的说明：每个状态写清自己那一句
// （§1.2.2 的文案表），不画一个看起来在跑或看起来能输入的终端。非激活层按它
// 自己的状态取（样例里可以有几个层，各写各的）。
func (a *App) termEmptyNoteFor(tab *termTab) string {
	return termEmptyNoteOf(a.termPhaseFor(tab))
}

// termEmptyNoteOf 把状态翻成画布说明行的那一句。
func termEmptyNoteOf(phase termPhase) string {
	switch phase {
	case termPhaseNoSession:
		return noteNoSessionID
	case termPhaseNoCwd:
		return termNoCwdText
	case termPhaseNoEntry:
		return termNoEntryText
	case termPhaseNotStarted:
		return termNotStartedText
	case termPhaseStarting:
		return termStartingText
	}
	return termEmptyText
}

// syncLiveTerminal 把 pty 的快照落到激活层的输出缓冲上。
//
// 连着 pty 时只有一个终端实例（TerminalSnapshot 只给一个），所以直接写 App 上
// 的 termState / termLines，再由画布同步进激活层。「清屏」只移动可见基线
// （termCleared），不丢 pty 的输出。连着 Source 但没有写入口时不画样例输出：
// 画布只留那行原因（§1.2.3 状态 3）。
func (a *App) syncLiveTerminal() {
	if !a.wired() {
		return
	}
	if !a.termLive() {
		a.termState, a.termLines, a.termExit, a.termReadFailed = termIdle, nil, termExit{}, false
		return
	}
	state, lines, exit, readFailed, ok := a.terminalView()
	if !ok {
		return
	}
	if a.termCleared > len(lines) {
		// 行缓冲被截断或换了终端：基线失效，从头显示，别把新输出也藏起来。
		a.termCleared = 0
	}
	if a.termCleared < 0 {
		a.termCleared = 0
	}
	a.termState, a.termLines, a.termExit, a.termReadFailed = state, lines[a.termCleared:], exit, readFailed
}

// terminalPane 是终端面板：工具条 + 深色画布。
func (a *App) terminalPane(c *gui.Context) {
	a.syncLiveTerminal()
	a.ensureTermTabs()
	gui.Column(c).Grow(1).Children(func() {
		a.termToolbar(c)
		a.termCanvas(c)
	})
}

// termTab 是一个终端标签对应的一层画布（T-09）：自己的状态、输出与滚动位置。
// 激活层的数据与 App 上的 termState / termLines 同步（其余代码仍读写那一对）。
type termTab struct {
	title  string
	state  termState
	lines  []termLine
	closed bool
	// exit 是该层进程的结束事实（T-11）：没有退出状态时只写「进程已退出」。
	exit termExit
	// readFailed 是该层输出读出失败（状态 10）：提示行收起，画布末尾给提示。
	readFailed bool
	scroll     gui.ScrollState
}

// ensureTermTabs 保证至少有一层画布，并把下标夹在范围内。
func (a *App) ensureTermTabs() {
	if len(a.termTabs) == 0 {
		a.termTabs = []termTab{{
			title:      a.termTitle(),
			state:      a.termState,
			lines:      a.termLines,
			closed:     a.panels.termClosed,
			exit:       a.termExit,
			readFailed: a.termReadFailed,
		}}
		a.termActive = 0
	}
	if a.termActive < 0 || a.termActive >= len(a.termTabs) {
		a.termActive = 0
	}
}

// syncActiveTerminal 把激活层的输出与状态写回层里（其它代码改的是 App 上的
// termState / termLines）。标题跟着事实走：连着 pty 时每帧从会话工作目录重取，
// shell 换了目录后标签标题随之更新（R-TM-024）；样例里标题是「终端 N」，不因
// 工作区标题而改名。
func (a *App) syncActiveTerminal() {
	a.ensureTermTabs()
	tab := &a.termTabs[a.termActive]
	tab.state, tab.lines, tab.closed = a.termState, a.termLines, a.panels.termClosed
	tab.exit, tab.readFailed = a.termExit, a.termReadFailed
	if a.wired() || tab.title == "" {
		tab.title = a.termTitle()
	}
}

// switchTerminal 切到另一层画布：先把当前层存好，再把目标层铺到 App 上的那一对
// 字段；非激活层的输出留在自己的缓冲里，不重建。
func (a *App) switchTerminal(i int) {
	a.ensureTermTabs()
	if i < 0 || i >= len(a.termTabs) || i == a.termActive {
		return
	}
	a.syncActiveTerminal()
	a.termActive = i
	tab := a.termTabs[i]
	a.termState, a.termLines, a.termExit, a.termReadFailed = tab.state, tab.lines, tab.exit, tab.readFailed
	a.panels.termClosed = tab.closed
}

// termLayerCount 是当前画布层数（与终端标签数一致）。
func (a *App) termLayerCount() int {
	a.ensureTermTabs()
	return len(a.termTabs)
}

// termToolbar 是终端工具条（T-01–T-07）：终端标签（标题 + 关闭进程）、
// 新建终端、关闭终端；右侧是中止 / 清屏 / 复制（样例另有断线入口）。
//
// 入口不因状态隐藏，只改可用性并各自带一句原因（§1.2.3）：灰着却没有说明，
// 用户无从知道为什么点不动。
func (a *App) termToolbar(c *gui.Context) {
	startNote := a.termStartNote()
	closeTabNote := a.termCloseTabNote()
	outputNote := a.termOutputNote()
	gui.Row(c).Height(termToolbarHeight).Shrink(0).
		Padding(space1, panelPad).Gap(space1).AlignItems(gui.Center).
		Label(a.termToolbarLabel()).Children(func() {
		a.termTab(c)
		if iconButton(c, iconButtonSpec{
			icon: iconPlus, label: "新建终端", tip: termTip("新建终端", startNote),
			size: rowCompact, disabled: startNote != "",
		}) {
			a.newTerminal()
		}
		if iconButton(c, iconButtonSpec{
			icon: iconXMark, label: "关闭终端", tip: termTip("关闭终端", closeTabNote),
			size: rowCompact, disabled: closeTabNote != "",
		}) {
			a.closeTerminal()
		}
		gui.Spacer(c)
		// 中止：输入区刚发出过 `!` 命令时才有运行标识可停；连着 pty 时终端自己的
		// Ctrl-C 走 pty（由 shell 处理），不另开一条中止通道。
		if a.lastShellRun != "" || (!a.wired() && a.termState == termRunning) {
			if iconButton(c, iconButtonSpec{
				icon: iconStop, label: "中止运行", tip: "中止运行 · Ctrl-C", size: rowCompact,
			}) {
				a.stopShellRun()
			}
		}
		if iconButton(c, iconButtonSpec{
			icon: iconEraser, label: "清屏", tip: termTip("清屏 · Cmd-K", outputNote),
			size: rowCompact, disabled: outputNote != "",
		}) {
			a.clearTerminal()
		}
		if iconButton(c, iconButtonSpec{
			icon: iconClipboard, label: "复制输出", tip: termTip("复制输出", outputNote),
			size: rowCompact, disabled: outputNote != "",
		}) {
			c.WriteClipboard(joinLines(visibleLines(a.termLines)))
			a.termCopied = true
		}
		// 断线与恢复是样例里的退化规则：连着 pty 时没有「模拟断线」这回事。
		if !a.wired() {
			if a.termState == termDisconnected {
				if iconButton(c, iconButtonSpec{
					icon: iconUndo, label: "重新连接", tip: "重新连接", size: rowCompact,
				}) {
					a.reconnectTerminal()
				}
			} else if iconButton(c, iconButtonSpec{
				icon: iconWarn, label: "模拟断线", tip: "模拟断线", size: rowCompact,
			}) {
				a.disconnectTerminal()
			}
		}
	})
}

// termTab 是工具条里的终端标签（T-01 / T-02 / T-03）：图标 + 标题 + 关闭进程。
// 标题是 shell 上报目录的末段，没有上报时回退「终端」。每个标签对应一层画布
// （T-09）：点标签切到那一层，非激活层的输出留在自己的缓冲里。
func (a *App) termTab(c *gui.Context) {
	t := c.Theme()
	a.ensureTermTabs()
	for i := range a.termTabs {
		a.termTabChip(c, t, i)
	}
}

// termTabChip 画一个终端标签；激活的那一个取浅底，其余透明。
func (a *App) termTabChip(c *gui.Context, t *gui.Theme, i int) {
	tab := a.termTabs[i]
	active := i == a.termActive
	chip := gui.Row(c).Height(rowCompact).Shrink(0).Gap(space1).
		Padding(space1, space2).Radius(radiusSmall).AlignItems(gui.Center).
		// 标签里还有「关闭进程」按钮：给 RoleTab 会让整棵子树对读屏不可见，
		// 关进程就只剩右键菜单一条路，所以这里按按钮播报，用选中态报当前层。
		AccessSelected(active).
		Tooltip(a.termTitlePath()).
		Label("终端标签：" + tab.title)
	if active {
		chip = chip.Background(t.SurfaceHover)
	}
	if chip.Clicked() {
		a.switchTerminal(i)
	}
	chip.Children(func() {
		iconGlyph(c, iconTerminal, t.TextMuted, "terminal")
		typeSecondaryStrong.style(t, gui.Text(c, tab.title)).TextColor(t.Text).
			SingleLine()
		// 非激活层不参与当前状态判定：它自己的关闭标记才是事实。
		closeNote := a.termCloseNote()
		if i != a.termActive {
			closeNote = ""
			if tab.closed {
				closeNote = termClosedNote
			}
		}
		if iconButton(c, iconButtonSpec{
			icon: iconXMark, label: "关闭进程", tip: termTip("关闭进程", closeNote),
			size: controlMini, disabled: closeNote != "",
		}) {
			a.closeTermProcessAt(i)
		}
	})
}

// termCanvas 是深色画布（T-08 / T-09）：每个终端标签一层，同尺寸、无偏移、
// 无边框；非激活层不可见但保留自己的滚动缓冲与输出。
func (a *App) termCanvas(c *gui.Context) {
	a.syncActiveTerminal()
	gui.Box(c).Grow(1).Label("终端画布层").Children(func() {
		for i := range a.termTabs {
			active := i == a.termActive
			layer := gui.Column(c).Absolute().Fill().IgnoreInput(!active).
				AccessHidden(!active).Hidden(!active)
			if active {
				layer = layer.Background(termCanvasBg)
			}
			layer.Children(func() {
				a.termLayer(c, &a.termTabs[i])
			})
		}
	})
}

// termLayer 是一层画布的内容：输出行 + 提示行，右键开菜单。非激活层保留自己
// 的行与滚动位置（T-09），切回来时从这里接着画。
func (a *App) termLayer(c *gui.Context, tab *termTab) {
	t := c.Theme()
	body := gui.Column(c).Grow(1).Fill().Padding(space2, panelPad).
		Label("终端").Role(gui.RoleGroup).
		ContextMenu(func(m *gui.Menu) { a.termMenu(m, c) })
	body.Children(func() {
		if tab.closed {
			gui.Column(c).Grow(1).Center().Children(func() {
				// T-10 ［进程已关闭］
				typeCode.style(t, gui.Text(c, termClosedNotice)).
					TextColor(termCanvasMuted)
			})
			return
		}
		visible := visibleLines(tab.lines)
		emptyNote := a.termEmptyNoteFor(tab)
		phase := a.termPhaseFor(tab)
		gui.Scroll(c).Grow(1).TrackScroll(&tab.scroll).Children(func() {
			gui.Column(c).Gap(0).Children(func() {
				if len(visible) == 0 {
					typeCode.style(t, gui.Text(c, emptyNote)).TextColor(termCanvasMuted)
				}
				for _, ln := range visible {
					typeCode.style(t, gui.Text(c, ln.text)).
						TextColor(termColorOf(t, ln.color))
				}
				// T-11 进程自行退出：最后一屏内容保留，末尾给结束提示行（退出码 /
				// 信号按事实写，读不到退出状态时只写「进程已退出」）。
				if phase == termPhaseExited {
					gui.Row(c).Shrink(0).Padding(space1, 0).Grow(1).Justify(gui.Center).
						Label(termExitNotice(tab.exit)).Children(func() {
						typeCode.style(t, gui.Text(c, termExitNotice(tab.exit))).
							TextColor(termCanvasMuted).TextAlign(gui.Center).SingleLine()
					})
				}
				// 状态 10（连着 pty）：读不到新行，但进程可能还在跑。给一行结束
				// 提示，提示行收起——写进去也看不到回声（§1.2.3）。
				if phase == termPhaseReadFailed && a.wired() {
					gui.Row(c).Shrink(0).Padding(space1, 0).Grow(1).Justify(gui.Center).
						Label(termReadFailedText).Children(func() {
						typeCode.style(t, gui.Text(c, termReadFailedText)).
							TextColor(termCanvasMuted).TextAlign(gui.Center).SingleLine()
					})
				}
				if tab == &a.termTabs[a.termActive] {
					a.termPrompt(c)
				}
			})
		})
	})
}

// termMenu 是画布右键菜单（T-15–T-21）：复制选中内容 / 粘贴 / 清屏 /
// 新建终端 / 关闭进程，两组分隔线把危险项单独分开。
//
// 五项与标签条同源判断（§1.2.4）：同一句原因同时写进悬停提示与菜单副标题，
// 不给"灰着又不说为什么"的项。
func (a *App) termMenu(m *gui.Menu, c *gui.Context) {
	// 画布还没有文本选择能力，选中内容恒为空，项保持禁用（与参考的未选中态一致）。
	m.Item("复制选中内容").Disabled(true)
	pasteItem := m.Item("粘贴")
	if note := a.termPasteNote(); note != "" {
		pasteItem.Disabled(true).Subtitle(note)
	}
	if pasteItem.Chosen() {
		if text := c.ReadClipboard(); text != "" {
			a.termInput += text
		}
	}
	m.Separator()
	clearItem := m.Item("清屏")
	if note := a.termOutputNote(); note != "" {
		clearItem.Disabled(true).Subtitle(note)
	}
	if clearItem.Chosen() {
		a.clearTerminal()
	}
	newItem := m.Item("新建终端")
	if note := a.termStartNote(); note != "" {
		newItem.Disabled(true).Subtitle(note)
	}
	if newItem.Chosen() {
		a.newTerminal()
	}
	m.Separator()
	closeItem := m.Item("关闭进程").Danger()
	if note := a.termCloseNote(); note != "" {
		closeItem.Disabled(true).Subtitle(note)
	}
	if closeItem.Chosen() {
		a.closeTermProcess()
	}
}

// termExitNotice 是进程自行退出后的提示行（T-11）。退出码与信号按事实写；
// 退出状态读不到时只写「进程已退出」，不留空的「代码 」段（R-TM-093）。
func termExitNotice(exit termExit) string {
	switch {
	case exit.signaled && exit.signal != 0:
		return fmt.Sprintf("［进程已退出，信号 %d］", exit.signal)
	case exit.known && !exit.signaled:
		return fmt.Sprintf("［进程已退出，代码 %d］", exit.code)
	}
	return termExitedNotice
}

// termCloseNote 见上（页签内「关闭进程」的原因）。

// termPrompt 是提示行：`(base) → 目录` + 输入框。目录名取会话的工作目录。
//
// 只有状态 6（运行中）才画这一行（§1.2.3）：没有 pty 时画出来的 `→ 目录` 会被
// 读成一个正在等输入的 shell，而按什么都没反应；光标同理。
func (a *App) termPrompt(c *gui.Context) {
	if a.terminalPhase() != termPhaseRunning {
		return
	}
	t := c.Theme()
	gui.Row(c).Shrink(0).Gap(space1).AlignItems(gui.Center).Children(func() {
		if env := a.termShellEnv(); env != "" {
			typeCode.style(t, gui.Text(c, env)).TextColor(termCanvasShell)
		}
		typeCode.style(t, gui.Text(c, "→")).TextColor(termCanvasArrow)
		dir := a.termTitle()
		if dir != "终端" {
			typeCode.style(t, gui.Text(c, dir)).TextColor(termCanvasText).
				FontWeight(600)
		}
		input := gui.TextInputBase(c, &a.termInput).Label("终端输入").Grow(1).
			Font(codeFamily()).FontSize(typeCode.sizeIn(t)).TextColor(termCanvasText)
		if a.termLive() {
			input.OnSubmit(func() { a.submitTerminalInput() })
		}
		if a.termInput == "" {
			termCaret(c)
		}
	})
}

// submitTerminalInput 把提示行里的内容写回 pty：行尾补 CR（pty 里的换行是回车），
// 发出去就清空提示行，不等回显。
func (a *App) submitTerminalInput() {
	line := a.termInput
	a.termInput = ""
	if !a.termLive() {
		return
	}
	a.dispatch("终端输入", func(ctx context.Context) error {
		return a.intents.WriteTerminal(ctx, line+"\r")
	}, nil)
}

// clearTerminal 清屏：样例里直接清空输出；连着 pty 时只移动可见基线，pty 的
// 输出不丢（之后新到的行照常显示）。
func (a *App) clearTerminal() {
	if a.termLive() {
		if snapshot, ok := a.liveTerminal(); ok {
			a.termCleared = len(snapshot.Lines)
		}
		a.termLines = nil
		a.syncActiveTerminal()
		return
	}
	a.termLines = nil
}

// termInterrupt 把 Ctrl-C 写进 pty，由 shell 决定怎么处理（终端不另做一套中止）。
func (a *App) termInterrupt() {
	if !a.termLive() {
		return
	}
	a.dispatch("中断终端", func(ctx context.Context) error {
		return a.intents.WriteTerminal(ctx, "\x03")
	}, nil)
}

// termCaret 是提示行尾的空心方块光标（参考产品在输入处画一个块状光标）。
func termCaret(c *gui.Context) {
	caret := gui.Box(c).Width(termCaretWidth).Height(termCaretHeight).Shrink(0).
		Label("终端光标").AccessHidden(true)
	caret.DrawOver(func(p *gui.Painter, r gui.Rect) {
		// 空心方块的四条边各 1 个设备像素：盒缘与厚度都按 DPR 取整，2× 上四条
		// 边一样实心（不取整时 1 DIP 的边在半像素上会被摊成两条灰边）。
		scale := p.Scale()
		box := snapIconRect(r, scale)
		line := deviceLength(scale)
		p.Fill(gui.Rect{X: box.X, Y: box.Y, W: box.W, H: line}, termCanvasMuted, 0)
		p.Fill(gui.Rect{X: box.X, Y: box.Y + box.H - line, W: box.W, H: line}, termCanvasMuted, 0)
		p.Fill(gui.Rect{X: box.X, Y: box.Y, W: line, H: box.H}, termCanvasMuted, 0)
		p.Fill(gui.Rect{X: box.X + box.W - line, Y: box.Y, W: line, H: box.H}, termCanvasMuted, 0)
	})
}

// termTitle 是终端标签与提示行里的目录名：连着 pty 时取会话工作目录的末段，
// 样例里取工作区标题的末段；都没有时回退「终端」（T-003 / R-TM-022）。
func (a *App) termTitle() string {
	title := ""
	if a.wired() {
		if snapshot, ok := a.liveSession(); ok {
			title = strings.TrimSpace(snapshot.Cwd)
		}
	}
	if title == "" {
		title = strings.TrimSpace(a.workspaceTitle())
	}
	if title == "" {
		return "终端"
	}
	if i := strings.LastIndexAny(title, `/\`); i >= 0 && i+1 < len(title) {
		title = title[i+1:]
	}
	if title == "" {
		return "终端"
	}
	return title
}

// termTitlePath 是终端标签悬停时给的完整路径（R-TM-018）：连着 pty 时是会话
// 工作目录原文；没有这条事实时退回标题，不编一个路径。
func (a *App) termTitlePath() string {
	if a.wired() {
		if snapshot, ok := a.liveSession(); ok {
			if p := strings.TrimSpace(snapshot.Cwd); p != "" {
				return p
			}
		}
	}
	return a.termTitle()
}

// termShellEnv 是 shell 的环境名前缀：样例里是 `(base)`，连着 Source 时
// 还没有这一项事实（终端快照不含它）。
func (a *App) termShellEnv() string {
	if a.wired() {
		return ""
	}
	return termEnvDefault
}

// newTerminal 新建一个终端（T-04 / T-19）。
//
// 连着 pty 时就是启动一个本机 shell（一个会话一个实例，按 DEC-08 复用同一个
// 页签）；样例模式按参考产品的形态追加一层画布，走查多层的切换。
func (a *App) newTerminal() {
	if a.wired() {
		a.startLiveTerminal()
		return
	}
	a.syncActiveTerminal()
	title := fmt.Sprintf("终端 %d", len(a.termTabs)+1)
	a.termTabs = append(a.termTabs, termTab{title: title, state: termIdle})
	a.termActive = len(a.termTabs) - 1
	a.termState, a.termLines = termIdle, nil
	a.panels.termClosed = false
	a.termInput = ""
}

// startLiveTerminal 在会话的工作目录里启动本机 shell；没有会话、没有工作目录、
// 没有写入口或已经在跑时不启动（原因由 termStartNote 给出，入口本来就是禁用的）。
//
// 启动请求在途时进状态 5：「新建终端」与「关闭终端」先按住，避免两次启动或收掉
// 一个刚起来的 pty；请求回来（成功或失败）就退出 5。
func (a *App) startLiveTerminal() {
	if a.termStartNote() != "" || !a.termLive() || a.termStarting {
		return
	}
	a.resetTerminalView()
	a.termStarting = true
	a.syncActiveTerminal()
	a.dispatchThen("新建终端", a.intents.StartTerminal, func() {
		a.termStarting = false
		a.resetTerminalView()
		a.syncActiveTerminal()
	}, func(string) {
		a.termStarting = false
	})
}

// resetTerminalView 把终端页的视图状态回到「没启动」。
func (a *App) resetTerminalView() {
	a.panels.termClosed = false
	a.termState = termIdle
	a.termLines = nil
	a.termExit = termExit{}
	a.termReadFailed = false
	a.termCleared = 0
	a.termInput = ""
}

// closeTermProcess 关闭当前终端的进程（T-21）：内容区换成提示行，标签保留。
func (a *App) closeTermProcess() {
	if a.wired() {
		a.stopLiveTerminal(true)
		return
	}
	a.panels.termClosed = true
	a.termState = termExited
	a.termLines = nil
	a.syncActiveTerminal()
}

// stopLiveTerminal 停止本机 pty（回收整棵进程树）；markClosed 为真时进状态 8
// （画布换成「进程已关闭」）；「关闭终端」只收页签，不显示这条提示。停止失败时
// 什么都不改：页签、画面与关闭标记都留在原样，原因走全局错误行（§1.2.5）。
func (a *App) stopLiveTerminal(markClosed bool) {
	if !a.termLive() || !a.termAlive() {
		return
	}
	a.dispatch("关闭终端进程", a.intents.StopTerminal, func() {
		if markClosed {
			a.panels.termClosed = true
			a.termState = termExited
			a.termStarting = false
			a.syncActiveTerminal()
			return
		}
		a.resetTerminalView()
		a.syncActiveTerminal()
	})
}

// closeTermProcessAt 关闭某一层的进程；其它层的输出与滚动位置不变。
func (a *App) closeTermProcessAt(i int) {
	a.ensureTermTabs()
	if i == a.termActive {
		a.closeTermProcess()
		return
	}
	if i < 0 || i >= len(a.termTabs) {
		return
	}
	a.termTabs[i].closed = true
	a.termTabs[i].state = termExited
	a.termTabs[i].lines = nil
}

// closeTerminal 关闭当前终端标签（T-06）：连着 pty 时先停止终端，停止成功后才
// 收标签（停止失败时标签还在，原因走全局错误行，不假装已关，T-06-048）；标签
// 减少 1 个，关掉最后一个时终端页回到「未启动」并切回文件页。
func (a *App) closeTerminal() {
	if a.wired() {
		a.closeLiveTerminal()
		return
	}
	a.closeTermTab()
}

// closeLiveTerminal 收起当前终端页签：有活进程时先停 pty，停止成功后才收标签
// （停止失败时标签还在，原因走全局错误行，不假装已关）；进程已结束或本就没有
// 进程时不发停止请求，直接收页签（§1.2.5）。
func (a *App) closeLiveTerminal() {
	if !a.termLive() || !a.termAlive() || a.terminalPhase() == termPhaseStarting {
		a.closeTermTab()
		return
	}
	a.dispatchThen("关闭终端", a.intents.StopTerminal, func() {
		a.resetTerminalView()
		a.closeTermTab()
	}, nil)
}

// closeTermTab 收掉当前终端标签。
func (a *App) closeTermTab() {
	a.syncActiveTerminal()
	if len(a.termTabs) <= 1 {
		// 最后一层：收回终端页（不留下一个后台还在跑的层），右栏回到文件页。
		a.termTabs = []termTab{{title: a.termTitle(), state: termIdle}}
		a.termActive = 0
		a.resetTerminalView()
		a.rightTab = tabFiles
		return
	}
	i := a.termActive
	a.termTabs = append(a.termTabs[:i:i], a.termTabs[i+1:]...)
	if a.termActive >= len(a.termTabs) {
		a.termActive = len(a.termTabs) - 1
	}
	tab := a.termTabs[a.termActive]
	a.termState, a.termLines, a.termExit, a.termReadFailed = tab.state, tab.lines, tab.exit, tab.readFailed
	a.panels.termClosed = tab.closed
}

// stopShellRun 中止当前的命令运行（可逆：命令可以再跑一次）。
//
// 只有输入区 `!` 命令的运行标识走 StopShell；终端自己的 Ctrl-C 走 pty（见
// termInterrupt），不在这里假装把终端停掉。
func (a *App) stopShellRun() {
	if a.intents != nil && a.lastShellRun != "" {
		runID := a.lastShellRun
		a.dispatch("中止命令", func(ctx context.Context) error {
			return a.intents.StopShell(ctx, runID)
		}, func() { a.lastShellRun = "" })
		return
	}
	if !a.wired() {
		a.termState = termExited
	}
}

// disconnectTerminal / reconnectTerminal 是样例里的断线与恢复（F31 的退化规则）。
func (a *App) disconnectTerminal() {
	a.termState = termDisconnected
	a.termLines = append(a.termLines, termLine{text: "connection lost", color: 1})
}

func (a *App) reconnectTerminal() {
	a.termState = termIdle
	a.termLines = append(a.termLines, termLine{text: "reconnected", color: 2})
}

// termToolbarLabel 是工具条的读屏名（屏幕上只写目录名，不写状态词）。样例按
// 样例状态播报（空闲 / 运行中 / 已退出 / 已断开），连着 pty 时按展示状态播报。
func (a *App) termToolbarLabel() string {
	if !a.wired() {
		switch a.termState {
		case termRunning:
			return "终端 · 运行中"
		case termExited:
			return "终端 · 已退出"
		case termDisconnected:
			return "终端 · 已断开"
		}
		return "终端 · 空闲"
	}
	return termPhaseLabel(a.terminalPhase())
}

// termPhaseLabel 把展示状态翻成读屏名。
func termPhaseLabel(phase termPhase) string {
	switch phase {
	case termPhaseRunning:
		return "终端 · 运行中"
	case termPhaseExited:
		return "终端 · 已退出"
	case termPhaseStarting:
		return "终端 · 启动中"
	case termPhaseReadFailed:
		return "终端 · 读不到输出"
	}
	return "终端 · 空闲"
}

// termColorOf 把 ANSI 前 8 色映射到终端配色；-1 是默认文字色。
func termColorOf(t *gui.Theme, idx int) gui.Color {
	switch idx {
	case 1:
		return termAnsiRed
	case 2:
		return termAnsiGreen
	case 3:
		return termAnsiYellow
	case 4:
		return termAnsiBlue
	}
	return termCanvasText
}

// joinLines 把终端行拼成纯文本，供复制使用。
func joinLines(lines []termLine) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line.text)
		b.WriteByte('\n')
	}
	return b.String()
}

// termLineCount 是输出行数（测试与基线用）。
func (a *App) termLineCount() int { return len(a.termLines) }

// termPlain 是输出区的纯文本（检查器用来判"长行/多行错误确实存在"）。
func (a *App) termPlain() string {
	var b strings.Builder
	for _, ln := range a.termLines {
		b.WriteString(ln.text)
		b.WriteByte('\n')
	}
	return b.String()
}
