package ui

import (
	"context"
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
// 不解析光标移动。Go 侧没有 pty：连着 Source 时读它的终端快照，否则用样例。

// termState 是终端的四种状态。
type termState int

const (
	termIdle termState = iota
	termRunning
	termExited
	termDisconnected
)

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
	// termEmptyText 是画布还没有输出时的说明行。
	termEmptyText = "还没有输出"
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
func (a *App) terminalView() (termState, []termLine, bool) {
	snapshot, ok := a.liveTerminal()
	if !ok {
		return a.termState, a.termLines, false
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
	return state, lines, true
}

// terminalPane 是终端面板：工具条 + 深色画布。
func (a *App) terminalPane(c *gui.Context) {
	gui.Column(c).Grow(1).Children(func() {
		a.termToolbar(c)
		a.termCanvas(c)
	})
}

// termToolbar 是终端工具条（T-01–T-07）：终端标签（标题 + 关闭进程）、
// 新建终端、关闭终端；右侧是走查用的中止 / 清屏 / 复制 / 断线入口。
func (a *App) termToolbar(c *gui.Context) {
	state, _, _ := a.terminalView()
	gui.Row(c).Height(termToolbarHeight).Shrink(0).
		Padding(space1, panelPad).Gap(space1).AlignItems(gui.Center).
		Label(termLabel(state)).Children(func() {
		a.termTab(c)
		if iconButton(c, iconButtonSpec{
			icon: iconPlus, label: "新建终端", tip: "新建终端",
			size: rowCompact, disabled: a.workspaceTitle() == "",
		}) {
			a.newTerminal()
		}
		if iconButton(c, iconButtonSpec{
			icon: iconXMark, label: "关闭终端", tip: "关闭终端", size: rowCompact,
		}) {
			a.closeTerminal()
		}
		gui.Spacer(c)
		// 中止：终端自己在跑，或输入区刚发出过命令（`!` 的命令不经过终端快照，
		// 只有 lastShellRun 记得它），两者都给入口。
		if state == termRunning || a.lastShellRun != "" {
			if iconButton(c, iconButtonSpec{
				icon: iconStop, label: "中止运行", tip: "中止运行 · Ctrl-C", size: rowCompact,
			}) {
				a.stopShellRun()
			}
		}
		if iconButton(c, iconButtonSpec{
			icon: iconEraser, label: "清屏", tip: "清屏 · Cmd-K", size: rowCompact,
		}) {
			a.termLines = nil
		}
		if iconButton(c, iconButtonSpec{
			icon: iconClipboard, label: "复制输出", tip: "复制输出", size: rowCompact,
		}) {
			_, lines, _ := a.terminalView()
			c.WriteClipboard(joinLines(visibleLines(lines)))
			a.termCopied = true
		}
		if state == termDisconnected {
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
	})
}

// termTab 是工具条里的终端标签（T-01 / T-02 / T-03）：图标 + 标题 + 关闭进程。
// 标题是 shell 上报目录的末段，没有上报时回退「终端」。
func (a *App) termTab(c *gui.Context) {
	t := c.Theme()
	chip := gui.Row(c).Height(rowCompact).Shrink(0).Gap(space1).
		Padding(space1, space2).Radius(radiusSmall).AlignItems(gui.Center).
		Background(t.SurfaceHover)
	chip.Children(func() {
		iconGlyph(c, iconTerminal, t.TextMuted, "terminal")
		typeSecondaryStrong.style(t, gui.Text(c, a.termTitle())).TextColor(t.Text).
			SingleLine()
		if iconButton(c, iconButtonSpec{
			icon: iconXMark, label: "关闭进程", tip: "关闭进程",
			size: controlMini, disabled: a.panels.termClosed,
		}) {
			a.closeTermProcess()
		}
	})
}

// termCanvas 是深色画布（T-08 / T-09）：输出行 + 提示行，右键开菜单。
func (a *App) termCanvas(c *gui.Context) {
	t := c.Theme()
	_, lines, _ := a.terminalView()
	body := gui.Column(c).Grow(1).Background(termCanvasBg).Padding(space2, panelPad).
		Label("终端").Role(gui.RoleGroup).
		ContextMenu(func(m *gui.Menu) { a.termMenu(m, c) })
	body.Children(func() {
		if a.panels.termClosed {
			gui.Column(c).Grow(1).Center().Children(func() {
				typeCode.style(t, gui.Text(c, termClosedNotice)).
					TextColor(termCanvasMuted)
			})
			return
		}
		visible := visibleLines(lines)
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(0).Children(func() {
				if len(visible) == 0 {
					typeCode.style(t, gui.Text(c, termEmptyText)).TextColor(termCanvasMuted)
				}
				for _, ln := range visible {
					typeCode.style(t, gui.Text(c, ln.text)).
						TextColor(termColorOf(t, ln.color))
				}
				a.termPrompt(c)
			})
		})
	})
}

// termMenu 是画布右键菜单（T-15–T-21）：复制选中内容 / 粘贴 / 清屏 /
// 新建终端 / 关闭进程，两组分隔线把危险项单独分开。
func (a *App) termMenu(m *gui.Menu, c *gui.Context) {
	// 画布还没有文本选择能力，选中内容恒为空，项保持禁用（与参考的未选中态一致）。
	m.Item("复制选中内容").Disabled(true)
	if m.Item("粘贴").Chosen() {
		if text := c.ReadClipboard(); text != "" {
			a.termInput += text
		}
	}
	m.Separator()
	if m.Item("清屏").Chosen() {
		a.termLines = nil
	}
	newItem := m.Item("新建终端")
	if a.workspaceTitle() == "" {
		newItem.Disabled(true)
	}
	if newItem.Chosen() {
		a.newTerminal()
	}
	m.Separator()
	closeItem := m.Item("关闭进程").Danger()
	if a.panels.termClosed {
		closeItem.Disabled(true)
	}
	if closeItem.Chosen() {
		a.closeTermProcess()
	}
}

// termPrompt 是提示行：`(base) → 目录` + 输入框。目录名取工作区标题，
// 没有上报时只画 `$`。
func (a *App) termPrompt(c *gui.Context) {
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
		gui.TextInputBase(c, &a.termInput).Label("终端输入").Grow(1).
			Font(codeFont).FontSize(typeCode.sizeIn(t)).TextColor(termCanvasText)
		if a.termInput == "" {
			termCaret(c)
		}
	})
}

// termCaret 是提示行尾的空心方块光标（参考产品在输入处画一个块状光标）。
func termCaret(c *gui.Context) {
	caret := gui.Box(c).Width(termCaretWidth).Height(termCaretHeight).Shrink(0).
		Label("终端光标").AccessHidden(true)
	caret.DrawOver(func(p *gui.Painter, r gui.Rect) {
		line := float32(1)
		p.Fill(gui.Rect{X: r.X, Y: r.Y, W: r.W, H: line}, termCanvasMuted, 0)
		p.Fill(gui.Rect{X: r.X, Y: r.Y + r.H - line, W: r.W, H: line}, termCanvasMuted, 0)
		p.Fill(gui.Rect{X: r.X, Y: r.Y, W: line, H: r.H}, termCanvasMuted, 0)
		p.Fill(gui.Rect{X: r.X + r.W - line, Y: r.Y, W: line, H: r.H}, termCanvasMuted, 0)
	})
}

// termTitle 是终端标签与提示行里的目录名：取工作区标题的末段；没有上报时
// 回退「终端」（T-003 / R-TM-022）。
func (a *App) termTitle() string {
	title := strings.TrimSpace(a.workspaceTitle())
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

// termShellEnv 是 shell 的环境名前缀：样例里是 `(base)`，连着 Source 时
// 还没有这一项事实（终端快照不含它）。
func (a *App) termShellEnv() string {
	if a.wired() {
		return ""
	}
	return termEnvDefault
}

// newTerminal 新建一个终端（T-04 / T-19）：清掉提示并把状态回到空闲。
//
// 宿主框架没有多实例终端标签，本产品也只显示一个终端页签（DEC-08）：
// 「新建终端」复用同一个页签，不追加第二个标签。
func (a *App) newTerminal() {
	a.panels.termClosed = false
	a.termState = termIdle
	a.termLines = nil
	a.termInput = ""
}

// closeTermProcess 关闭当前终端的进程（T-21）：内容区换成提示行，标签保留。
func (a *App) closeTermProcess() {
	a.panels.termClosed = true
	a.termState = termExited
	a.termLines = nil
}

// closeTerminal 关闭终端标签（T-06）：切回文件面板（参考里是切回上一页签）。
func (a *App) closeTerminal() {
	a.panels.termClosed = false
	a.rightTab = tabFiles
}

// stopShellRun 中止当前的命令运行（可逆：命令可以再跑一次）。
func (a *App) stopShellRun() {
	if a.intents != nil && a.lastShellRun != "" {
		runID := a.lastShellRun
		a.dispatch("中止命令", func(ctx context.Context) error {
			return a.intents.StopShell(ctx, runID)
		}, func() { a.lastShellRun = "" })
		return
	}
	a.termState = termExited
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

// termLabel 是工具条的读屏名（屏幕上只写目录名，不写状态词）。
func termLabel(st termState) string {
	switch st {
	case termRunning:
		return "终端 · 运行中"
	case termExited:
		return "终端 · 已退出"
	case termDisconnected:
		return "终端 · 已断开"
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
