package ui

import (
	gui "github.com/egoist/mygo/ui"
)

// 时间线的渲染窗口（U4 §4）：行数超过门槛后只画视口里的那几行，视口上下用
// 等高占位顶住滚动长度。
//
// 口径：
//   - 只改「画哪些行」，不改行的内容与顺序；占位无文字、不进读屏。
//   - 行高先按行种类估算，画过的行把实测高度记下来，下一帧起用实测值，
//     所以滚动条刻度与跳转落点会随滚动收敛（第一帧偏、之后准）。
//   - 搜索的当前命中与正在行内编辑的那一行必定进入窗口：跳过去的行要先被
//     画出来，`ScrollIntoView` 才有落点（A5-02 的 375）。
//   - 门槛以下整段渲染：短会话的画法与虚拟化无关，不给几行也套一层换算。

const (
	// timelineWindowMin 是启用渲染窗口的行数门槛。200 轮（1000 行）实测仍在
	// 一帧预算内，这里的门槛留在它之上，让「短会话」保持整段渲染。
	timelineWindowMin = 400
	// timelineWindowBuffer 是视口上下各多画的行数：滚一屏之内不出现空白。
	timelineWindowBuffer = 6
	// timelineWindowMaxRendered 是窗口内最多渲染的行数（测试与自查的上界）：
	// 视口高度 ÷ 单行最小高度，再加两侧缓冲。
	timelineWindowMaxRendered = 96
)

// rowHeights 是实测行高的缓存（可丢的视图状态）：下标是行 / 轮次序号，
// 值是该行排出来的高度（DIP）；0 表示还没画过、只能用估算。
type rowHeights struct {
	heights []float32
	// count 是上一帧的行数：行数变少（换了会话、窗口缩回）就丢掉整段缓存，
	// 免得旧高度落到新内容上；行数变多（追加）时前面的高度仍然有效。
	count int
}

// keep 按本帧的行数收一收缓存。
func (h *rowHeights) keep(n int) {
	if n < h.count {
		h.heights = nil
	}
	h.count = n
	if len(h.heights) > n {
		h.heights = h.heights[:n]
	}
}

// heightOf 取行 i 的实测高度；没有实测时返回 0。
func (h *rowHeights) heightOf(i int) float32 {
	if i < 0 || i >= len(h.heights) {
		return 0
	}
	return h.heights[i]
}

// record 记下行 i 的实测高度。
func (h *rowHeights) record(i int, v float32) {
	if i < 0 || v <= 0 {
		return
	}
	for len(h.heights) <= i {
		h.heights = append(h.heights, 0)
	}
	h.heights[i] = v
}

// rowHeight 取行 i 的高度：实测优先生效，没测过按估算。
func (a *App) rowHeight(i int, estimate func(int) float32) float32 {
	if v := a.rowHeights.heightOf(i); v > 0 {
		return v
	}
	return estimate(i)
}

// measureRow 把一行的实际高度记下来。取的是上一帧布局的结果（新建出来的
// 行第一帧读到 0，下一帧补上），不改变本帧的布局。
func (a *App) measureRow(i int, el gui.Element) {
	if i < 0 {
		return
	}
	a.rowHeights.record(i, el.Bounds().H)
}

// rowSpan 是一帧的渲染范围：画 [start, end)，上下各留 above / below 的占位。
type rowSpan struct {
	start, end   int
	above, below float32
}

// rows 是这一帧真正画出来的行数（自查与测试用）。
func (a *App) renderedRows() int { return a.renderedCount }

// renderedSpan 是这一帧画出来的行区间（自查与测试用）。
func (a *App) renderedSpan() (int, int) { return a.renderedFrom, a.renderedTo }

// rowSpanFor 算这一帧要画的行区间。[viewport] 是滚动区的可见高度（DIP，
// 0 表示还没有布局结果），[estimate] 按行序号给高度估算值，用它折算视口落在
// 累加高度里的哪几行。
func (a *App) rowSpanFor(c *gui.Context, viewport float32, n int, estimate func(int) float32) rowSpan {
	a.renderedCount = n
	a.renderedFrom, a.renderedTo = 0, n
	a.rowHeights.keep(n)
	if n <= timelineWindowMin {
		return rowSpan{start: 0, end: n}
	}
	if viewport <= 0 {
		// 第一帧还没有布局结果：按窗口高估一个视口，宁可多画几行也不留白。
		if _, h := c.Size(); h > 0 {
			viewport = float32(h)
		}
	}
	if viewport <= 0 {
		return rowSpan{start: 0, end: n}
	}
	gap := float32(timelineRoundGap)
	top := a.feed.Y
	if top < 0 {
		top = 0
	}

	// 一遍累加：看视口的上缘落在哪一行的中间，下缘越过哪一行。
	start, end := -1, n
	y := float32(0)
	for i := 0; i < n; i++ {
		h := a.rowHeight(i, estimate) + gap
		if start < 0 && y+h > top {
			start = i
		}
		if end == n && y+h > top+viewport {
			end = i
		}
		y += h
	}
	if start < 0 {
		start = n - 1
	}
	if end <= start {
		end = start + 1
	}

	// 两侧各留缓冲；搜索的当前命中与正在行内编辑的行必定在场。
	start -= timelineWindowBuffer
	end += timelineWindowBuffer
	if row, ok := a.windowAnchorRow(); ok && row >= 0 && row < n {
		if row < start || row >= end {
			start, end = row-timelineWindowBuffer, row+timelineWindowBuffer+1
		}
	}
	if start < 0 {
		start = 0
	}
	if end > n {
		end = n
	}
	if end-start > timelineWindowMaxRendered {
		end = start + timelineWindowMaxRendered
	}

	// 占位高度按被跳过的行累加：一块占 k 行时，它的高度 = 各行高 + 行间
	// 间距（块内的 k-1 条），块与相邻行之间的那条间距由容器的 Gap 给出。
	above, below := float32(0), float32(0)
	if start > 0 {
		above = -gap
		for i := 0; i < start; i++ {
			above += a.rowHeight(i, estimate) + gap
		}
	}
	if end < n {
		below = -gap
		for i := end; i < n; i++ {
			below += a.rowHeight(i, estimate) + gap
		}
	}
	a.renderedCount = end - start
	a.renderedFrom, a.renderedTo = start, end
	return rowSpan{start: start, end: end, above: above, below: below}
}

// windowAnchorRow 是本帧必定要画的那一行：正在行内编辑的那一行优先，其次
// 是会话内搜索的当前命中（跳过去要先有落点）。
func (a *App) windowAnchorRow() (int, bool) {
	if a.inlineEditRow >= 0 {
		return a.inlineEditRow, true
	}
	s := &a.sessionSearch
	if !s.open {
		return 0, false
	}
	h, ok := s.currentHit()
	if !ok || h.unit < 0 || h.unit >= len(s.units) {
		return 0, false
	}
	return s.units[h.unit].row, true
}

// windowSpacer 是视口外的等高占位：无文字、无边框、不进读屏。
func windowSpacer(c *gui.Context, h float32) {
	if h <= 0 {
		return
	}
	gui.Box(c).Shrink(0).Height(h).WidthPercent(100).
		Label("时间线占位").AccessHidden(true)
}

// estimateLiveRow 是连着事实时按行种类给的高度估算（DIP）：一行文字 21、
// 用户气泡 60 上下、正文一段 96 上下。估算只影响跳过去的落点与滚动条刻度，
// 画过的行马上换成实测值。
func estimateLiveRow(row RowSnapshot) float32 {
	switch row.Kind {
	case RowUser:
		return 60
	case RowThinking, RowTool, RowCompression, RowRetry:
		return timelineRowHeight
	case RowToolGroup:
		return timelineRowHeight * 2
	case RowError:
		return timelineRowHeight * 2
	case RowFileChanges, RowArtifact:
		return timelineRowHeight * 3
	default:
		return 96
	}
}

// estimateRound 是样例模式按一轮的内容给的高度估算：与 liveRow 同一口径。
func estimateRound(r round) float32 {
	h := float32(60)
	if r.thinking != "" {
		h += timelineRowHeight
	}
	if r.reply != "" {
		h += 96
	}
	h += float32(len(r.ops)) * (timelineRowHeight + float32(space3))
	return h
}
