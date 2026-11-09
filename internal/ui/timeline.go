package ui

import (
	"fmt"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// timelineTurnGap is the room after a user turn before the assistant turn under
// it, in DIPs. The reference's 800×720 frame puts the bubble's top at y≈74 and
// the assistant's first line at y≈149: the row's own rowPadY is not enough, so
// the user row carries this measured turn gap below the bubble. It is a
// literal, not a spacing step: the reference measures it from its own frame.
const timelineTurnGap = 32

// BlockKind decides how one block of an assistant turn renders.
type BlockKind uint8

const (
	// BlockText is the assistant's reply text, shown as Markdown, exactly as
	// an assistant row's Content is.
	BlockText BlockKind = iota
	// BlockThinking is the model's reasoning. It is folded by default: a
	// header says how long it took, and its text shows while it is open.
	BlockThinking
	// BlockTool is one tool call of the turn, drawn as an expandable card.
	BlockTool
	// BlockFileChanges is the turn's file changes, drawn as a compact line
	// per file.
	BlockFileChanges
)

// Block is one piece of an assistant turn, in the order the projection kept
// it. It is view state only: the projection owns the block boundaries, their
// order and their status, and this struct is what an adapter copies them into.
//
// docs/ui-layout-spec.md §4 records dim-electron's structure: a turn is one
// list row whose blocks render in place, rather than one row per block.
type Block struct {
	// ID identifies the block across frames, so a fold the user opened stays
	// open as the turn grows. It is the projection's block identity (the first
	// event's ID); a fixture that has none is keyed by its index instead.
	ID string
	// Kind decides how the block renders.
	Kind BlockKind
	// Content is the block's text: Markdown for BlockText, the reasoning for
	// BlockThinking, empty for the others.
	Content string
	// Running is the projection's status, true while the block still streams.
	// A running thinking block says "Thinking" instead of a duration.
	Running bool
	// StartedAt and EndedAt are the projection's event timestamps in Unix
	// milliseconds, EndedAt 0 while the block runs or when it never settled.
	// They only decide the thinking header's duration text.
	StartedAt int64
	EndedAt   int64
	// Tool is the call of a BlockTool block.
	Tool Tool
	// Files are the changes of a BlockFileChanges block.
	//
	// Data-source gap: the timeline projection folds no file-change facts
	// (internal/projection handles step/* events but has no file-changes
	// event), so no adapter fills this today. The view renders the block from
	// a fixture only; it invents no daemon fact.
	Files []FileChange
}

// FileStatus is what happened to one changed file, as dim-electron's
// FileRewindPreviewEntry.kind. FileUnknown is a change the model did not
// describe, and the row shows "?" rather than guessing.
type FileStatus uint8

const (
	FileUnknown FileStatus = iota
	FileAdded
	FileModified
	FileDeleted
)

// Letter is the status letter the row shows, as git's.
func (s FileStatus) Letter() string {
	switch s {
	case FileAdded:
		return "A"
	case FileModified:
		return "M"
	case FileDeleted:
		return "D"
	default:
		return "?"
	}
}

// FileChange is one row of a file-changes block: the path, what happened to
// it and, when the source reported counts, how many lines it touched.
type FileChange struct {
	Path   string
	Status FileStatus
	// Insertions and Deletions are the changed line counts. HasStats is false
	// for a binary or unreported change, so the row leaves the counts out
	// instead of claiming 0/0.
	Insertions int
	Deletions  int
	HasStats   bool
}

// blockKey names one block's fold: the row it belongs to and the block's own
// identity, falling back to its place for a fixture block that carries none.
func blockKey(rowID string, block Block, index int) string {
	id := block.ID
	if id == "" {
		id = "#" + strconv.Itoa(index)
	}
	return rowID + "\x00" + id
}

// thinkingLabel is the thinking header's text: "Thinking" while it runs, the
// settled duration as "Thought for 4s", and a plain "Thought" when the
// projection kept no duration. It matches dim-electron's timeline.thinking and
// timeline.thinkingWithDuration strings.
func thinkingLabel(block Block) string {
	if block.Running {
		return "Thinking"
	}
	if duration := block.EndedAt - block.StartedAt; duration > 0 {
		return "Thought for " + formatElapsed(duration)
	}
	return "Thought"
}

// formatElapsed writes a duration the way dim-electron's formatElapsedSeconds
// does: one decimal below a second, whole seconds below a minute, and minutes
// with seconds above that.
func formatElapsed(millis int64) string {
	if millis <= 0 {
		return "0s"
	}
	seconds := millis / 1000
	if seconds < 60 {
		if seconds == 0 {
			return fmt.Sprintf("%.1fs", math.Round(float64(millis)/100)/10)
		}
		return strconv.FormatInt(seconds, 10) + "s"
	}
	minutes := seconds / 60
	rest := seconds % 60
	if rest > 0 {
		return fmt.Sprintf("%dm %ds", minutes, rest)
	}
	return fmt.Sprintf("%dm", minutes)
}

// fileChangesLabel is the file-changes header's text, as dim-electron's
// changeSummary.fileChangesCount: "1 file changed", "3 files changed".
func fileChangesLabel(count int) string {
	if count == 1 {
		return "1 file changed"
	}
	return strconv.Itoa(count) + " files changed"
}

// fileStats is a file row's count text, as dim-electron writes it: "+12/-4".
func fileStats(file FileChange) string {
	return "+" + strconv.Itoa(file.Insertions) + "/-" + strconv.Itoa(file.Deletions)
}

// fileStatusColor is the color of a file row's status letter.
func (k tokens) fileStatusColor(status FileStatus) ui.Color {
	switch status {
	case FileAdded:
		return k.success
	case FileModified:
		return k.warning
	case FileDeleted:
		return k.danger
	default:
		return k.mutedForeground
	}
}

// chevron draws a disclosure arrow in a box, turned by progress: 0 points
// right, 1 points down. mygo's own disclosure draws the same shape; this one is
// used where the header is a row built by hand instead of ui.Collapsible.
func chevron(c *ui.Context, k tokens, color ui.Color, progress float32) {
	ui.Box(c).Size(k.timelineChevron, k.timelineChevron).Shrink(0).AccessHidden(true).
		Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, d := r.X+r.W/2, r.Y+r.H/2, r.W/8
			sin, cos := math.Sincos(float64(90*progress) * math.Pi / 180)
			at := func(x, y float32) (float32, float32) {
				return cx + x*float32(cos) - y*float32(sin), cy + x*float32(sin) + y*float32(cos)
			}
			var path ui.Path
			x, y := at(-d, -2*d)
			path.MoveTo(x, y)
			x, y = at(d, 0)
			path.LineTo(x, y)
			x, y = at(-d, 2*d)
			path.LineTo(x, y)
			p.StrokePath(&path, k.hairline+0.5, color)
		})
}

// userRow draws what the user submitted as a right-aligned bubble: at most 85%
// of the row, on the muted face, as docs/ui-layout-spec.md §4.2 records it.
func userRow(c *ui.Context, k tokens, m Message) {
	ui.Row(c).Padding(k.rowPadY, k.timelinePadX, k.rowPadY+timelineTurnGap, k.timelinePadX).
		Justify(ui.End).Children(func() {
		ui.Column(c).MaxWidthPercent(k.timelineUserWidth).
			Padding(k.timelineUserPadY, k.timelineUserPadX).
			Radius(k.timelineBubbleR).Background(k.muted).Children(func() {
			ui.Text(c, m.Content).FontSize(k.bodySize).TextColor(k.foreground)
		})
	})
}

// assistantRow draws one assistant turn: its blocks in the projection's order.
// A row with no blocks falls back to Content as one Markdown block, which is
// what a fixture and an adapter that keeps only text hand it. The reference
// draws no author line on an assistant turn, so neither does this row; the
// conversation list exposes the rendered text as the row's name.
func assistantRow(c *ui.Context, k tokens, v TimelineView, m Message) {
	ui.Column(c).Padding(k.rowPadY, k.timelinePadX).Gap(k.gap).Children(func() {
		if len(m.Blocks) == 0 {
			if m.Content != "" {
				renderMarkdownContent(c, k, m.Content, MarkdownOptions{})
			}
			return
		}
		ui.Column(c).Gap(k.timelineBlockGap).Children(func() {
			for i, block := range m.Blocks {
				blockView(c, k, v, m.ID, i, block)
			}
		})
	})
}

// blockView draws one block of a turn. The fold key joins the row and the
// block, so two turns with the same block identity keep separate folds.
func blockView(c *ui.Context, k tokens, v TimelineView, rowID string, index int, block Block) {
	key := blockKey(rowID, block, index)
	switch block.Kind {
	case BlockThinking:
		thinkingBlock(c, k, v, key, block)
	case BlockTool:
		toolCard(c, k, v.folds, key, block.Tool)
	case BlockFileChanges:
		fileChangesBlock(c, k, block.Files)
	default:
		if block.Content != "" {
			renderMarkdownContent(c, k, block.Content, MarkdownOptions{})
		}
	}
}

// thinkingBlock draws a folded thinking block: a disclosure header reading
// "Thinking" or "Thought for Ns", and, while open, the reasoning behind a 2px
// rail in a panel capped at 240 DIPs, as the spec's max-h-[240px] does.
func thinkingBlock(c *ui.Context, k tokens, v TimelineView, key string, block Block) {
	p := ui.CollapsibleBase(c, v.folds.state(key))
	p.Trigger.MinHeight(k.timelineHeaderH).Gap(k.gap).AlignItems(ui.Center).
		Children(func() {
			chevron(c, k, k.mutedForeground, p.Progress())
			ui.Text(c, thinkingLabel(block)).FontSize(k.bodySize).
				TextColor(k.mutedForeground).SingleLine()
		})
	p.Panel(func() {
		ui.Scroll(c).MaxHeight(k.timelinePanelMaxH).Children(func() {
			ui.Text(c, block.Content).FontSize(k.bodySize).TextColor(k.mutedForeground)
		})
	}).Margin(k.gap, 0, 0, 0).PaddingStart(k.timelinePadStart).
		BorderStartWidth(k.timelineRail).BorderColor(k.border)
}

// fileChangesBlock draws the turn's file changes as the compact block the spec
// describes: one header line, then one line per file inside a bordered box.
// There is no per-file fold: a file row is a line of text, not an action.
func fileChangesBlock(c *ui.Context, k tokens, files []FileChange) {
	if len(files) == 0 {
		return
	}
	ui.Column(c).Gap(k.gap).Children(func() {
		ui.Text(c, fileChangesLabel(len(files))).FontSize(k.labelSize).Bold().
			TextColor(k.mutedForeground)
		ui.Column(c).Radius(radiusMD).BorderWidth(k.hairline).BorderColor(k.border).
			Clip().Children(func() {
			for i, file := range files {
				fileChangeRow(c, k, file, i == len(files)-1)
			}
		})
	})
}

// fileChangeRow draws one file of a file-changes block: its status letter, its
// path and, when the source reported them, its line counts.
func fileChangeRow(c *ui.Context, k tokens, file FileChange, last bool) {
	row := ui.Row(c).Padding(k.timelineFileRowY, k.timelineFileRowX).
		Gap(k.timelineFileGap).AlignItems(ui.Center)
	if !last {
		row.BorderWidth(0, 0, k.hairline, 0).BorderColor(k.border)
	}
	row.Children(func() {
		ui.Text(c, file.Status.Letter()).FontSize(k.codeFontSize).Bold().
			Width(k.timelineFileStatus).TextAlign(ui.Center).
			TextColor(k.fileStatusColor(file.Status))
		ui.Text(c, file.Path).FontSize(k.bodySize).TextColor(k.mutedForeground).
			SingleLine().Grow(1)
		if file.HasStats {
			ui.Text(c, fileStats(file)).FontSize(k.codeFontSize).
				TextColor(k.mutedForeground).SingleLine()
		}
	})
}

// errorRow draws a turn that ended in failure as its own row: dim-electron
// keeps the terminal notice in the turn's tail, and this shell gives it a row
// and an alert region instead. The text comes from the caller; the shell adds
// no reason of its own.
func errorRow(c *ui.Context, k tokens, m Message) {
	ui.Column(c).Padding(k.rowPadY, k.timelinePadX).Gap(k.gap).Children(func() {
		ui.Text(c, m.Author.String()).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
		ui.Row(c).Role(ui.RoleAlert).Label(m.Content).Children(func() {
			ui.Text(c, m.Content).FontSize(k.labelSize).TextColor(k.danger).Bold()
		})
	})
}

// retryRow draws a retry notice as its own row: a retried turn's line, in the
// warning color, so it reads apart from the rows that follow it.
func retryRow(c *ui.Context, k tokens, m Message) {
	ui.Column(c).Padding(k.rowPadY, k.timelinePadX).Gap(k.gap).Children(func() {
		ui.Text(c, m.Author.String()).FontSize(k.smallSize).TextColor(k.mutedForeground).Unselectable()
		ui.Text(c, m.Content).FontSize(k.labelSize).TextColor(k.warning)
	})
}
