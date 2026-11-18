package ui

import (
	"fmt"
	"path"

	gui "github.com/egoist/mygo/ui"
)

// 文件预览页（tabFilePreview，清单 RP-A-12 / P-01）：文件页点一行就打开这一页。
//
// 事实来源：连着 Source 且适配器实现 PanelFacts 时读 Preview 的真实读数
// （D7 的 Preview：编码识别、读取上限、二进制与截断说明都在快照里）；只在
// -demo（没有 Source）时用样例，样例见 rightpanel_sample.go。
//
// 本页只留可丢的视图状态：正在看哪个文件（rightPanelState.previewPath）。
// 内容每帧按同一请求读快照，视图不缓存正文。

// rpOpenPreview 在预览页打开一个文件：标签按需出现，路径落到视图状态。
func (a *App) rpOpenPreview(p string) {
	s := a.rp()
	s.previewPath = p
	a.rpOpen(tabFilePreview)
}

// rpPreviewSnapshot 取当前文件的预览读数：连着 Source 时读 PanelFacts，
// 否则给样例。refresh 来自「重新加载 / 重试」（一次点击只重读一次）。
func (a *App) rpPreviewSnapshot(p string) PreviewSnapshot {
	facts, ok := a.rpFacts()
	if !ok {
		return rpSamplePreview(p)
	}
	return facts.Preview(PreviewRequest{Cwd: a.rpCwd(), Path: p})
}

// rpPreviewPage 是预览页本体。
func (a *App) rpPreviewPage(c *gui.Context) {
	t := c.Theme()
	s := a.rp()
	// 「重新加载」与文件页共用一个计数：一次点击只让下一次读跳过缓存。
	refresh := a.rpTakeRefresh()
	gui.Column(c).Grow(1).Children(func() {
		a.rpOfflineBar(c)
		if s.previewPath == "" {
			a.rpEmptyBlockIcon(c, iconDocument, "没有打开的文件",
				"在文件页点一个文件就能在这里看它的内容。")
			return
		}
		snapshot := a.rpPreviewSnapshotWithRefresh(s.previewPath, refresh)
		a.rpPreviewHeader(c, t, snapshot)
		rpRule(c)
		gui.Scroll(c).Grow(1).Label("文件内容").Children(func() {
			gui.Column(c).Padding(space2, panelRowPadH).Gap(space1).Children(func() {
				a.rpPreviewBody(c, t, snapshot)
			})
		})
	})
}

// rpPreviewSnapshotWithRefresh 取一次预览读数；refresh 为真时要求重读。
func (a *App) rpPreviewSnapshotWithRefresh(p string, refresh bool) PreviewSnapshot {
	facts, ok := a.rpFacts()
	if !ok {
		return rpSamplePreview(p)
	}
	return facts.Preview(PreviewRequest{Cwd: a.rpCwd(), Path: p, Refresh: refresh})
}

// rpPreviewHeader 是页头：文件图标 + 文件名 + 路径，右端是大小与编码。
//
// 大小、编码与截断都来自读数；读不到时右端不画（不发明数字）。
func (a *App) rpPreviewHeader(c *gui.Context, t *gui.Theme, snapshot PreviewSnapshot) {
	name := path.Base(snapshot.Path)
	if name == "" {
		name = snapshot.Path
	}
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(panelRowGap).
		Padding(space1, space3, space1, panelRowPadH).Children(func() {
		rpGlyph(c, rpFileIcon(name), panelRowIcon, t.TextMuted, "")
		gui.Column(c).Grow(1).MinWidth(0).Children(func() {
			typeControl.style(t, gui.Text(c, name)).TextColor(t.Text).SingleLine()
			dir := path.Dir(snapshot.Path)
			if dir == "." {
				dir = ""
			}
			if dir != "" {
				typeMicro.style(t, gui.Text(c, dir)).TextColor(t.TextMuted).SingleLine()
			}
		})
		if snapshot.Phase == PanelReadReady {
			typeSecondary.style(t, gui.Text(c, rpPreviewSizeLabel(snapshot))).
				TextColor(t.TextMuted).SingleLine()
		}
	})
}

// rpPreviewSizeLabel 是页头右端的「大小 · 编码」。
func rpPreviewSizeLabel(snapshot PreviewSnapshot) string {
	label := rpByteLabel(snapshot.Size)
	if snapshot.Encoding != "" {
		label += " · " + snapshot.Encoding
	}
	return label
}

// rpByteLabel 把字节数写成 B / KiB / MiB（读数来自 Size，不四舍五入成整数）。
func rpByteLabel(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.1f KiB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MiB", float64(size)/(1024*1024))
}

// rpPreviewBody 按读态与内容类别画正文：读态、失败 + 重试、空文件、二进制、
// 文本，以及截断 / 编码说明。
func (a *App) rpPreviewBody(c *gui.Context, t *gui.Theme, snapshot PreviewSnapshot) {
	switch snapshot.Phase {
	case PanelReadIdle:
		typeSecondary.style(t, gui.Text(c, "还没有可以读取的文件。")).
			TextColor(t.TextMuted).Role(gui.RoleStatus).SingleLine()
		return
	case PanelReadLoading:
		typeSecondary.style(t, gui.Text(c, "正在读取文件…")).
			TextColor(t.TextMuted).Role(gui.RoleStatus).SingleLine()
		return
	case PanelReadFailed:
		reason := snapshot.Error
		if reason == "" {
			reason = "文件读取没有给出原因。"
		}
		gui.Row(c).Shrink(0).Gap(space1).AlignItems(gui.Center).Children(func() {
			typeSecondary.style(t, gui.Text(c, reason)).TextColor(t.Danger).
				Role(gui.RoleAlert).Grow(1).MaxLines(2)
			if rpTextAction(c, iconRetry, "重试", "重试读取文件", false) {
				a.rp().reload++
			}
		})
		return
	}
	switch snapshot.Kind {
	case PreviewEmpty:
		typeSecondary.style(t, gui.Text(c, "空文件，没有内容。")).
			TextColor(t.TextMuted).SingleLine()
	case PreviewBinary:
		typeSecondary.style(t, gui.Text(c, "二进制或未识别编码，不显示内容。")).
			TextColor(t.TextMuted).SingleLine()
	case PreviewText:
		// 正文是事实原文：不设读屏名（设了会把可见文字换成读屏名）。
		typeCode.style(t, gui.Text(c, snapshot.Content)).TextColor(t.Text).
			Selectable()
	default:
		typeSecondary.style(t, gui.Text(c, "没有可预览的内容。")).
			TextColor(t.TextMuted).SingleLine()
	}
	if snapshot.Note != "" {
		typeSecondary.style(t, gui.Text(c, snapshot.Note)).TextColor(t.TextMuted).
			Selectable()
	}
}
