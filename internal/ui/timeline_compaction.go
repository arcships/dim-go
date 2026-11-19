package ui

import (
	"strconv"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 压缩分隔行与摘要区（TL-68 / TL-69）。
//
// 状态词取服务端给的压缩状态：running 正在压缩、failed 压缩失败、skipped 已跳过、
// cancelled 已取消，其余按发生过的压缩写「上下文已压缩」。分隔行可点开摘要区，
// 摘要区三态：摘要生成中（进行中）/ 摘要正文 / 无法加载摘要 + 重试；摘要为空写
// 「无摘要」，不留空白区。
//
// 摘要是服务端的事实：本端不生成、不缓存第二份。重试要有取摘要的方法，本版
// 没有（协议只推一次压缩事实），所以连线时「重试」置灰并写原因；样例模式里
// 重试把样例摘要填上，走查能走完交互。

// compactionWord 是分隔行的状态词（TL-665–TL-668）。
func compactionWord(status, text string) string {
	switch status {
	case RowStatusRunning:
		return "正在压缩"
	case RowStatusFailed:
		return "压缩失败"
	case "skipped":
		return "已跳过"
	case RowStatusCancelled:
		return "已取消"
	}
	if status != "" {
		return "上下文已压缩"
	}
	if strings.TrimSpace(text) != "" {
		return text
	}
	return "上下文已压缩"
}

// compactionFacts 取最近一次压缩事实：连着 Source 时读事实，否则读样例。
func (a *App) compactionFacts() *CompactionSnapshot {
	if snap, ok := a.liveSession(); ok {
		return snap.Compaction
	}
	if a.wired() {
		return nil
	}
	return a.demoCompaction
}

// compressionRow 画一条压缩分隔行，并在展开时画出摘要区。
//
// text 是行原文（服务端的摘要或系统提示），status 是这一行的状态；两者都可能
// 为空，此时按事实或默认词画。
func (a *App) compressionRow(c *gui.Context, key, text, status string) {
	t := c.Theme()
	if facts := a.compactionFacts(); facts != nil {
		if facts.Status != "" {
			status = facts.Status
		}
		if strings.TrimSpace(facts.Summary) != "" {
			text = facts.Summary
		}
	}
	word := compactionWord(status, text)
	open := a.compactionOpenOf(key)
	btn := gui.ButtonBase(c).Grow(1).Height(timelineRowHeight).Gap(space2).
		Justify(gui.Start).AlignItems(gui.Center).Padding(0, space1).
		Radius(radius).Label(word).Role(gui.RoleDisclosure).Expanded(*open)
	progress := btn.AnimateTo("compaction-arrow", b2f(*open), timelineArrow)
	btn.Children(func() {
		gui.Box(c).Grow(1).Height(strokeHairline).Background(t.Border)
		if status == RowStatusRunning {
			gui.Spinner(c).Label("正在压缩")
		}
		color := t.TextMuted
		if status == RowStatusFailed {
			color = t.Warning
		}
		typeMicro.style(t, gui.Text(c, word)).TextColor(color).SingleLine()
		chevronLabeled(c, t, progress, "压缩摘要展开指示")
		gui.Box(c).Grow(1).Height(strokeHairline).Background(t.Border)
	})
	if btn.Clicked() {
		*open = !*open
	}
	a.revealPanel(c, *open, progress, func() {
		a.compactionSummary(c, status)
	})
}

// compactionOpenOf 取某条压缩行的展开态（可丢的视图状态）。
func (a *App) compactionOpenOf(key string) *bool {
	if a.compactionOpen == nil {
		a.compactionOpen = map[string]*bool{}
	}
	if p, ok := a.compactionOpen[key]; ok {
		return p
	}
	open := false
	a.compactionOpen[key] = &open
	return &open
}

// compactionSummary 是摘要区（TL-675–TL-681）：三态 + 无摘要兜底。
func (a *App) compactionSummary(c *gui.Context, status string) {
	t := c.Theme()
	facts := a.compactionFacts()
	summary := ""
	if facts != nil {
		summary = strings.TrimSpace(facts.Summary)
	}
	gui.Column(c).Shrink(0).Gap(space1).PaddingStart(timelineGroupIndent).
		Label("压缩摘要").Children(func() {
		switch {
		case summary != "" && status != RowStatusRunning:
			typeBody.style(t, gui.Text(c, summary)).TextColor(t.Text).SingleLine()
		case status == RowStatusRunning:
			gui.Row(c).Gap(space1).AlignItems(gui.Center).Label("摘要生成中").Children(func() {
				gui.Spinner(c).Label("摘要生成中")
				typeMicro.style(t, gui.Text(c, "摘要生成中")).TextColor(t.TextMuted).SingleLine()
			})
		case status == RowStatusFailed:
			a.compactionUnavailable(c)
		case summary == "":
			typeMicro.style(t, gui.Text(c, "无摘要")).TextColor(t.TextMuted).
				Label("无摘要").SingleLine()
		default:
			typeBody.style(t, gui.Text(c, summary)).TextColor(t.Text).SingleLine()
		}
	})
}

// compactionUnavailable 是「无法加载摘要。+ 重试」（TL-677）。
func (a *App) compactionUnavailable(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Label("无法加载摘要").Children(func() {
		typeMicro.style(t, gui.Text(c, "无法加载摘要。")).TextColor(t.TextMuted).SingleLine()
		if a.wired() || a.intents != nil {
			b := gui.ButtonBase(c).Height(rowCompact).Padding(0, space1).Radius(radius).
				Label("重试（" + noteCompactionRetry + "）").Tooltip(noteCompactionRetry).
				Disabled(true).Opacity(0.5)
			b.Children(func() {
				typeMicro.style(t, gui.Text(c, "重试")).TextColor(t.TextMuted).SingleLine()
			})
			return
		}
		if gui.Button(c, "重试").Label("重试").Clicked() {
			a.demoCompaction = &CompactionSnapshot{
				Status:  "completed",
				Summary: "前 12 轮讨论了排队面板与命令面板；摘要由服务端给出，本端只显示。",
			}
		}
	})
}

// compressionRowKey 把行序号做成压缩行的视图状态键。
func compressionRowKey(i int) string { return strconv.Itoa(i) }
