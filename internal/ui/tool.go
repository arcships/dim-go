package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// toolPreviewLines and toolPreviewRunes bound the params and the result a tool
// row draws: the card previews at most three lines of each, and no line runs
// past toolPreviewRunes runes, so a long command or a large result cannot grow
// a row without limit. The shell keeps nothing but the preview; the full text
// belongs to the tool call the projection owns.
const (
	toolPreviewLines = 3
	toolPreviewRunes = 96
)

// The status badge's inset, a compact chip around one short word. The card's
// own radius and border come from tokens.go (k.cardRadius, k.hairline), so the
// badge adds no new shape to the shell.
const (
	toolBadgePadX = 5.25
	toolBadgePadY = 1.75
)

// reasonToolRetry explains why a failed call's retry entry stays disabled: no
// window callback replays a call, so the entry keeps its place instead of
// pretending a click re-ran anything.
const reasonToolRetry = "Unavailable: the shell has no tool retry callback yet, so this entry cannot replay the call."

// Preview fallbacks: a tool call the view model leaves empty still shows the
// line, so the card's shape does not depend on what the App Server sent.
const (
	unnamedTool  = "(unnamed tool)"
	emptyPreview = "(none)"
)

// ToolStatus is the state of one tool call, as the view model reports it. It is
// view state only: the projection owns what the call actually did.
type ToolStatus uint8

const (
	// ToolUnknown is a call the view model did not describe. It never shows as
	// running, so a row cannot claim work the shell has no report of.
	ToolUnknown ToolStatus = iota
	// ToolRunning is a call the App Server has not finished.
	ToolRunning
	// ToolComplete is a call that finished without an error.
	ToolComplete
	// ToolError is a call that failed.
	ToolError
	// ToolCancelled is a call the user stopped.
	ToolCancelled
)

// String is the status label the card shows beside the tool's name.
func (s ToolStatus) String() string {
	switch s {
	case ToolRunning:
		return "Running"
	case ToolComplete:
		return "Complete"
	case ToolError:
		return "Error"
	case ToolCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// Tool is the view model of one tool call: the name, the params and result the
// row previews, and its status. It is view state only, filled by the fixture
// until the TimelineFold projection replaces it in Milestone 3.
type Tool struct {
	// Name is the tool's name, as "bash" or "read_file".
	Name string
	// Params and Result are what the call was asked with and what it came to.
	// The row draws one bounded line of each; the full text is not kept here.
	Params string
	Result string
	// Status decides the card's label, color and whether it is live.
	Status ToolStatus

	// The typed facts of the call's own kind, which the open card's panel draws
	// ahead of the raw params and result (toolrender.go). The projection folds
	// no typed tool-call shape yet — the view model forwards the name, the
	// formatted params and the formatted result only — so an adapter leaves
	// these zero and the panel names the gap instead of guessing. A fact whose
	// Has flag is false is absent, never zero: an unreported exit code must not
	// read as success, nor an unreported count as none.
	//
	// Command and ExitCode describe a command call. ExitCode is only meaningful
	// with HasExit, so a call that has not settled cannot show 0.
	Command  string
	ExitCode int
	HasExit  bool
	// Path and the changed-line counts describe a file call. HasStats is false
	// for a read, or for a change whose source reported no counts, so the panel
	// leaves them out rather than claiming +0/-0.
	Path       string
	Insertions int
	Deletions  int
	HasStats   bool
	// Pattern, Matches and MatchCount describe a search. MatchCount is the hit
	// count the source reported, which may be larger than the paths it listed;
	// HasMatchCount is false when it reported none.
	Pattern       string
	Matches       []string
	MatchCount    int
	HasMatchCount bool
}

// previewLine makes s a single bounded line: every run of whitespace becomes
// one space, and text past toolPreviewRunes runes ends in an ellipsis. The same
// input always gives the same output, which is what a headless fixture checks.
// The open panel's typed facts use it, so a long command or path stays one row.
func previewLine(s string) string {
	return cutRunes(strings.Join(strings.Fields(s), " "), toolPreviewRunes)
}

// cutRunes ends s in an ellipsis past limit runes, or returns it unchanged.
func cutRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "…"
}

// previewText is what the closed card draws for params or result: at most
// toolPreviewLines lines, each one bounded by toolPreviewRunes runes, with the
// last line ending in an ellipsis when the call sent more. Line breaks are kept
// so a multi-line command still reads as the step sequence it was, and each
// line's runs of spaces become one space so a tab cannot shift the layout. A
// call that sent six lines draws three, and the fourth never reaches the row.
func previewText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := range lines {
		lines[i] = previewLine(lines[i])
	}
	if len(lines) <= toolPreviewLines {
		return strings.Join(lines, "\n")
	}
	lines = lines[:toolPreviewLines]
	last := lines[len(lines)-1]
	if !strings.HasSuffix(last, "…") {
		lines[len(lines)-1] = last + "…"
	}
	return strings.Join(lines, "\n")
}

// toolName is the name the card shows, for a row whose model left it empty.
func toolName(tool Tool) string {
	if tool.Name == "" {
		return unnamedTool
	}
	return tool.Name
}

// previewField names what a preview is before its text, so the card reads as
// "Params: …" to assistive technology as well as to the eye.
func previewField(label, value string) string {
	if value == "" {
		return label + ": " + emptyPreview
	}
	return label + ": " + value
}

// toolRow builds the card of a tool row: the same card a BlockTool uses, keyed
// by the row's own ID so its fold survives a rebuild.
func toolRow(c *ui.Context, k tokens, folds *blockFolds, m Message) {
	ui.Column(c).Padding(k.rowPadY, k.timelinePadX).Children(func() {
		toolCard(c, k, folds, blockKey(m.ID, Block{Kind: BlockTool}, 0), m.Tool)
	})
}

// toolCard draws one tool call as a compact card that folds: its name and
// status badge are always in the header, its params and result preview as at
// most three bounded lines each while closed, and its typed facts, params and
// result fill a panel capped at 240 DIPs while open. A failed call draws its
// border in the danger color and keeps a retry entry, which stays disabled until
// the window has a callback that can replay a call. The card is the card face of
// the surface ladder, not a tint of the border: a call sits on the conversation
// as a raised block with its own text color.
//
// A running call shows its status in a live region that is busy, so it is
// announced once and assistive technology waits for the result rather than
// re-reading the card each frame.
func toolCard(c *ui.Context, k tokens, folds *blockFolds, key string, tool Tool) {
	open := folds.state(key)
	border := k.border
	if tool.Status == ToolError {
		border = k.danger
	}
	ui.Column(c).
		Padding(k.cardPadY, k.cardPadX).
		Gap(k.gap).Radius(k.cardRadius).
		Background(k.card).
		BorderWidth(k.hairline).BorderColor(border).Children(func() {
		var p ui.CollapsibleParts
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			// The disclosure is the arrow and the name; the status sits beside
			// it, outside the button, so a running call keeps its own live
			// region instead of being flattened into the button's name.
			p = ui.CollapsibleBase(c, open)
			p.Trigger.MinHeight(k.timelineHeaderH).Gap(k.gap).AlignItems(ui.Center).
				Children(func() {
					chevron(c, k, k.mutedForeground, p.Progress())
					ui.Text(c, previewLine(toolName(tool))).Bold().FontSize(k.labelSize).
						TextColor(k.cardForeground).SingleLine()
				})
			toolStatusBadge(c, k, tool.Status)
			ui.Spacer(c)
		})
		p.Panel(func() {
			ui.Scroll(c).MaxHeight(k.timelinePanelMaxH).Role(ui.RoleGroup).
				Label("Tool details").Children(func() {
				toolPanel(c, k, tool)
			})
		}).Margin(k.gap, 0, 0, 0).Radius(k.radiusMD).BorderWidth(k.hairline).
			BorderColor(k.border).Clip()
		// The closed card keeps at most three bounded lines of each part, so it
		// still says what the call was without being opened. MaxLines keeps each
		// preview at the card's line budget: the card's height is bounded
		// whatever the App Server sends.
		if !*open && p.Progress() == 0 {
			ui.Text(c, previewField("Params", previewText(tool.Params))).
				FontSize(k.smallSize).TextColor(k.mutedForeground).MaxLines(toolPreviewLines)
			ui.Text(c, previewField("Result", previewText(tool.Result))).
				FontSize(k.smallSize).TextColor(k.mutedForeground).MaxLines(toolPreviewLines)
		}
		// A failed call keeps its retry entry in the card's body, open or
		// closed, so the user does not have to expand the panel to find it. It
		// stays WIP until a callback can replay the call: a disabled entry that
		// explains itself, never a click that reports a success nobody ran.
		if tool.Status == ToolError {
			ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
				wipAction(c, k, "Retry", reasonToolRetry)
			})
		}
	})
}

// panelValue is what a tool card's open panel shows for a part: the value the
// view model kept, or the same "(none)" fallback the closed card uses, so an
// empty part reads the same open or closed.
func panelValue(value string) string {
	if value == "" {
		return emptyPreview
	}
	return value
}

// toolPanelSection draws one titled part of a tool card's open panel, as
// dim-electron's tool panel does: a small muted title, then the call's code.
func toolPanelSection(c *ui.Context, k tokens, title, text string) {
	ui.Text(c, title).FontSize(k.smallSize).Bold().TextColor(k.mutedForeground)
	ui.Text(c, text).FontSize(k.codeFontSize).TextColor(k.cardForeground)
}

// toolStatusBadge draws a call's status as a chip in the color of its state: a
// wash of that color behind the word keeps the status legible beside the tool
// name without reading as a button. Only a running call is a live region: it is
// announced as the card appears and says the row is busy, while a finished call
// stays plain text, so the row does not speak again every frame.
func toolStatusBadge(c *ui.Context, k tokens, status ToolStatus) {
	color := k.toolStatusColor(status)
	chip := func() {
		ui.Row(c).Padding(toolBadgePadY, toolBadgePadX).Radius(k.radiusSM).
			Background(k.wash(color)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, status.String()).FontSize(k.smallSize).TextColor(color)
		})
	}
	if status == ToolRunning {
		ui.Row(c).Role(ui.RoleStatus).Busy(true).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).AccessHidden(true) // the label beside it says what is happening
			chip()
		})
		return
	}
	chip()
}
