// Package viewmodel turns the durable timeline projection into the native UI's
// row model.
//
// It is a view adapter, not a fact source. Build and RunStateFrom read only the
// snapshot handed to them, keep no state, and never read a daemon, the SQLite
// store, JSONL or a log: identity, order, status and tool facts stay the
// projection's, and this package only decides how they show as rows. Every row
// it returns is throwaway view state.
package viewmodel

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// previewRunes bounds the params and the result a tool row carries. The row
// draws its own, shorter line (internal/ui), so this is the adapter's budget
// for a value a wider view may show, and it keeps a row model small whatever
// the App Server sent.
const previewRunes = 160

// unnamedTool is what a tool call shows when the projection holds no name for
// it. It matches the row's own fallback, so a caller sees the same name with or
// without the row applying it.
const unnamedTool = "(unnamed tool)"

// Build maps a projection snapshot to conversation rows, in the snapshot's
// order and under its message IDs, so the list keeps a row's place and state as
// the timeline grows.
//
// Roles map as the shell draws them:
//
//   - "user" becomes an AuthorUser row of the input's visible text;
//   - "assistant" becomes an AuthorAssistant row of the reply's Markdown. The
//     projection keeps thinking blocks in the message's events, not in its
//     content, so a thinking-only turn is a row with empty content rather than
//     an invented one: the projection kept the message, so the shell keeps it.
//   - "tool" becomes an AuthorTool row of a compact card: the call's name, a
//     one-line params preview, a one-line result preview and its status.
//
// Any other role draws no row: the projection emits approvals as transient
// notices, not durable messages, and the shell invents none of them.
//
// An empty snapshot gives an empty slice, never nil, so a caller can render it
// without checking.
func Build(snapshot projection.Snapshot) []ui.Message {
	rows := make([]ui.Message, 0, len(snapshot.Messages))
	for _, message := range snapshot.Messages {
		switch message.Role {
		case "user":
			rows = append(rows, ui.Message{
				ID:      message.ID,
				Author:  ui.AuthorUser,
				Content: message.Content,
			})
		case "assistant":
			rows = append(rows, ui.Message{
				ID:      message.ID,
				Author:  ui.AuthorAssistant,
				Content: message.Content,
			})
		case "tool":
			rows = append(rows, ui.Message{
				ID:     message.ID,
				Author: ui.AuthorTool,
				Tool:   buildTool(message),
			})
		}
	}
	return rows
}

// buildTool maps one tool message: its name and params come from the call event
// the projection merges every later fact into, its result from the last result
// event (they are kept in sequence order), and its status from the message the
// projection settled.
func buildTool(message projection.Message) ui.Tool {
	tool := ui.Tool{Name: unnamedTool}
	result := message.Content
	status := message.Status
	for _, event := range message.Events {
		switch event.Type {
		case "tool_call":
			if event.ToolName != "" {
				tool.Name = event.ToolName
			}
			tool.Params = formatParams(event.ToolParams)
			if status == "" {
				status = event.Status
			}
		case "tool_result":
			result = event.Content
			if status == "" {
				status = event.Status
			}
		}
	}
	tool.Result = formatResult(result)
	tool.Status = toolStatus(status)
	return tool
}

// toolStatus maps the projection's status words to the row's.
func toolStatus(status string) ui.ToolStatus {
	switch status {
	case "running", "streaming":
		return ui.ToolRunning
	case "complete", "completed":
		return ui.ToolComplete
	case "error", "failed":
		return ui.ToolError
	case "cancelled":
		return ui.ToolCancelled
	default:
		return ui.ToolUnknown
	}
}

// formatParams renders a call's params as one line with the keys of every object
// sorted, so the same facts always give the same row text whatever order the map
// was built in. A value the wire cannot carry, as a function or a NaN, has no
// compact form: the row shows none of it rather than an order-dependent dump.
//
// A line past previewRunes is cut with an ellipsis, so it is a preview to read,
// not JSON to parse: the full params stay the projection's.
func formatParams(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(params); err != nil {
		return ""
	}
	return boundLine(strings.TrimSpace(buffer.String()))
}

// formatResult makes a result one bounded line: the newlines of a command's
// output are layout, not text a row can show.
func formatResult(result string) string {
	return boundLine(strings.Join(strings.Fields(result), " "))
}

// boundLine cuts s to previewRunes runes, ending an over-long line with an
// ellipsis, so a row model stays small whatever the App Server sent.
func boundLine(s string) string {
	runes := []rune(s)
	if len(runes) <= previewRunes {
		return s
	}
	return string(runes[:previewRunes]) + "…"
}

// RunStateFrom reports what the snapshot's own rows say about the session's run:
// running while the projection still holds a message it has not closed, as an
// input the server has not consumed or a reply still streaming.
//
// It is a hint for the status row, not the session's state: the supervisor's
// session.state remains authoritative, and RunStopping is never returned
// because the durable timeline carries no stopping status.
func RunStateFrom(snapshot projection.Snapshot) ui.RunState {
	for _, message := range snapshot.Messages {
		switch message.Status {
		case "streaming", "sending":
			return ui.RunRunning
		}
	}
	return ui.RunIdle
}
