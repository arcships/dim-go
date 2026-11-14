package ui

import (
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 终端是右栏的第五个标签（U1-ia R5 的职责句与 U1 E6「终端会话」都在右栏；
// S25 的输入区 `!` 前缀是另一件事，属后续批次，本批不做）。
//
// 表面 = 等宽输出区 + 提示行；状态用样式（行首点 + 颜色），不在标题里写「运行中」。
// 输出里的 ANSI 片段只取颜色（前 8 色），不解析光标移动——本批是界面形态，不接 pty。

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

// terminalPane 是终端表面：一行工具条（状态点 + 清屏 / 中止）+ 输出区 + 提示行。
func (a *App) terminalPane(c *gui.Context) {
	t := c.Theme()

	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).
		Label(termLabel(a.termState)).Children(func() {
		// 状态：行首一个 8px 点（空闲次要色、运行中 accent、已退出成功色、断开
		// danger）；屏幕上不写状态词，只给读屏名（UI-23/A2）。
		statusDotMark(c, termColor(t, a.termState), "term")
		gui.Spacer(c)
		// 中止只在运行中出现（可逆，AC-18）；清屏一直在（局部不可逆，AC-19）。
		if a.termState == termRunning {
			if iconButton(c, iconButtonSpec{icon: iconStop, label: "中止运行", size: rowCompact}) {
				a.termState = termExited
			}
		}
		// 断开与恢复：真实入口（样例里模拟连接状态，F31 的退化规则）。
		if a.termState == termDisconnected {
			if iconButton(c, iconButtonSpec{icon: iconUndo, label: "重新连接", size: rowCompact}) {
				a.termState = termIdle
				a.termLines = append(a.termLines, termLine{text: "reconnected", color: 2})
			}
		} else if iconButton(c, iconButtonSpec{icon: iconWarn, label: "模拟断线", size: rowCompact}) {
			a.termState = termDisconnected
			a.termLines = append(a.termLines, termLine{text: "connection lost", color: 1})
		}
		if iconButton(c, iconButtonSpec{icon: iconXMark, label: "清屏", size: rowCompact}) {
			a.termLines = nil
		}
		if iconButton(c, iconButtonSpec{icon: iconClipboard, label: "复制输出", size: rowCompact}) {
			a.termCopied = true
		}
	})

	if len(a.termLines) == 0 {
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label("终端空态").Children(func() {
			iconGlyph(c, iconChat, t.TextMuted, "chat")
			typeSecondary.style(t, gui.Text(c, "还没有输出")).TextColor(t.TextMuted)
		})
		return
	}
	// 输出区：等宽、可滚动、按宽度折行（终端里长行折行比横向滚动更常用）。
	gui.Scroll(c).Grow(1).Children(func() {
		gui.Column(c).Gap(0).Children(func() {
			for _, ln := range a.termVisible() {
				line := typeCode.style(t, gui.Text(c, ln.text)).TextColor(termColorOf(t, ln.color))
				_ = line
			}
		})
	})
}

// termPrompt 是提示行：等宽输入 + 前缀 `$`。
func (a *App) termPrompt(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Padding(0, space1).Children(func() {
		typeCode.style(t, gui.Text(c, "$")).TextColor(t.TextMuted)
		gui.TextInputBase(c, &a.termInput).Label("终端输入").Grow(1).
			Font(codeFont).FontSize(typeCode.sizeIn(t))
	})
}

// termColor 是状态点的颜色：空闲次要色、运行中 accent-ink、已退出成功色、断开 danger。
func termColor(t *gui.Theme, st termState) gui.Color {
	switch st {
	case termRunning:
		return colorAccentInk.in(t)
	case termExited:
		return t.Success
	case termDisconnected:
		return t.Danger
	}
	return t.TextMuted
}

// termLabel 是状态的读屏名（屏幕上不写状态词）。
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

// termColorOf 把 ANSI 前 8 色映射到主题色；-1 是默认文字色。
func termColorOf(t *gui.Theme, idx int) gui.Color {
	switch idx {
	case 1:
		return t.Danger
	case 2:
		return t.Success
	case 3:
		return t.Warning
	case 4:
		return colorAccentInk.in(t)
	}
	return t.Text
}

// statusDotMark 是带检查器标记的状态点（一行里可能有多个点，标记带名称）。
func statusDotMark(c *gui.Context, color gui.Color, mark string) {
	dot := gui.Box(c).Width(dotStatus).Height(dotStatus).Shrink(0).
		Label("图标：dot:" + mark).AccessHidden(true)
	dot.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, color, dotStatus/2)
	})
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
