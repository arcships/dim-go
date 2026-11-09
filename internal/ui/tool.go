package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// toolPreviewRunes bounds the params and the result a tool row draws: the card
// is one line each whatever the App Server sends, so a long command or a large
// result cannot grow a row without limit. The shell keeps nothing but the
// preview; the full text belongs to the tool call the projection owns.
const toolPreviewRunes = 96

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
}

// previewLine makes s a single bounded line: every run of whitespace becomes
// one space, so a multi-line command or result stays one row, and text past
// toolPreviewRunes runes ends in an ellipsis. The same input always gives the
// same output, which is what a headless fixture checks.
func previewLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	runes := []rune(s)
	if len(runes) <= toolPreviewRunes {
		return s
	}
	return string(runes[:toolPreviewRunes]) + "…"
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
// status are always in the header, its params and result preview as one bounded
// line each while closed, and its params and result fill a panel capped at 240
// DIPs while open. The card is the card face of the surface ladder, not a tint
// of the border: a call sits on the conversation as a raised block with its own
// text color.
//
// A running call shows its status in a live region that is busy, so it is
// announced once and assistive technology waits for the result rather than
// re-reading the card each frame.
func toolCard(c *ui.Context, k tokens, folds *blockFolds, key string, tool Tool) {
	open := folds.state(key)
	ui.Column(c).
		Padding(k.cardPadY, k.cardPadX).
		Gap(k.gap).Radius(k.cardRadius).
		Background(k.card).
		BorderWidth(k.hairline).BorderColor(k.border).Children(func() {
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
			toolStatusLabel(c, k, tool.Status)
			ui.Spacer(c)
		})
		p.Panel(func() {
			ui.Scroll(c).MaxHeight(k.timelinePanelMaxH).Children(func() {
				ui.Column(c).Padding(k.cardPadY, k.cardPadX).Gap(k.gap).Children(func() {
					toolPanelSection(c, k, "Params", panelValue(tool.Params))
					toolPanelSection(c, k, "Result", panelValue(tool.Result))
				})
			})
		}).Margin(k.gap, 0, 0, 0).Radius(radiusMD).BorderWidth(k.hairline).
			BorderColor(k.border).Clip()
		// The closed card keeps one bounded line of each part, so it still says
		// what the call was without being opened. NoWrap keeps each preview on
		// its line: the card's height is bounded whatever the App Server sends.
		if !*open && p.Progress() == 0 {
			ui.Text(c, previewField("Params", previewLine(tool.Params))).
				FontSize(k.smallSize).TextColor(k.mutedForeground).NoWrap()
			ui.Text(c, previewField("Result", previewLine(tool.Result))).
				FontSize(k.smallSize).TextColor(k.mutedForeground).NoWrap()
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

// toolStatusLabel draws a call's status in the color of its state. Only a
// running call is a live region: it is announced as the card appears and says
// the row is busy, while a finished call stays plain text, so the row does not
// speak again every frame.
func toolStatusLabel(c *ui.Context, k tokens, status ToolStatus) {
	label := status.String()
	if status == ToolRunning {
		ui.Row(c).Role(ui.RoleStatus).Busy(true).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).AccessHidden(true) // the label beside it says what is happening
			ui.Text(c, label).FontSize(k.smallSize).TextColor(k.toolStatusColor(status))
		})
		return
	}
	ui.Text(c, label).FontSize(k.smallSize).TextColor(k.toolStatusColor(status))
}
