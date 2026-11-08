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

// toolRow builds the compact card of one tool call: its name and status, then
// one bounded line of params and one of result. A running call shows its status
// in a live region that is busy, so it is announced once and assistive
// technology waits for the result rather than re-reading the card each frame.
func (s *State) toolRow(c *ui.Context, k tokens, m Message) {
	tool := m.Tool
	ui.Column(c).
		Padding(k.cardPadY, k.cardPadX).
		Gap(k.gap).Radius(k.cardRadius).
		Background(k.wash(k.border)).
		BorderWidth(k.hairline).BorderColor(k.border).Children(func() {
		ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, previewLine(toolName(tool))).Bold().FontSize(k.labelSize).NoWrap()
			ui.Box(c).Grow(1)
			toolStatusLabel(c, k, tool.Status)
		})
		// NoWrap keeps each preview on its line: the card's height is bounded
		// whatever the App Server sends, and the preview is already cut.
		ui.Text(c, previewField("Params", previewLine(tool.Params))).
			FontSize(k.smallSize).TextColor(k.muted).NoWrap()
		ui.Text(c, previewField("Result", previewLine(tool.Result))).
			FontSize(k.smallSize).TextColor(k.muted).NoWrap()
	})
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
