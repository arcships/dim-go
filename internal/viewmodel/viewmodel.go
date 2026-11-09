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

	"github.com/arcships/dimagent-mygo/internal/appserver"
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
//   - "assistant" becomes an AuthorAssistant row of the reply's blocks, in the
//     projection's own order: a thinking block for every reasoning block the
//     projection kept, a text block for every text block, with the status and
//     timestamps it folded. The row's Content stays the concatenated reply, so a
//     caller that ignores blocks still has the turn's text, and a turn with no
//     blocks renders it as one Markdown block. A thinking-only turn keeps its
//     row: the projection kept the message, so the shell keeps it, with the
//     reasoning folded in a thinking block rather than dropped or shown as the
//     reply.
//   - "tool" becomes an AuthorTool row of a compact, expandable card: the
//     call's name, its params and result previews and its status.
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
				Blocks:  buildBlocks(message),
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

// buildBlocks maps one assistant message's blocks, in the order the projection
// kept them. Only the block types the projection folds are mapped: text and
// thinking. Tool calls are separate "tool" messages in the projection, so they
// stay rows of their own instead of being guessed into a turn, and file changes
// have no projection fact at all (internal/projection folds no file-changes
// event), so this adapter invents none.
//
// A message with no text or thinking block gives an empty slice, never nil, so
// the row falls back to its Content instead of rendering nothing.
func buildBlocks(message projection.Message) []ui.Block {
	blocks := make([]ui.Block, 0, len(message.Events))
	for _, event := range message.Events {
		block := ui.Block{
			ID:        event.ID,
			Running:   event.Status == "running",
			StartedAt: event.StartTime,
		}
		if event.EndTime != nil {
			block.EndedAt = *event.EndTime
		}
		switch event.Type {
		case "thinking":
			block.Kind = ui.BlockThinking
			block.Content = event.Content
		case "text_delta":
			block.Kind = ui.BlockText
			block.Content = event.Content
		default:
			continue
		}
		blocks = append(blocks, block)
	}
	return blocks
}

// BuildApprovals adapts the session's daemon-owned pending approvals into rows
// after the durable ones. A pending approval is not a durable message: the
// session projection holds it only until the App Server sends approval.resolved
// or approval.cancelled, so callers must rebuild these rows on every frame
// instead of keeping or editing them.
//
// An empty list gives an empty slice, never nil, matching Build.
func BuildApprovals(approvals ...appserver.ApprovalRequest) []ui.Message {
	rows := make([]ui.Message, 0, len(approvals))
	for _, approval := range approvals {
		parts := make([]string, 0, 2)
		if approval.Call.Name != "" {
			parts = append(parts, approval.Call.Name)
		}
		if approval.Reason != "" {
			parts = append(parts, approval.Reason)
		}
		rows = append(rows, ui.Message{
			ID:      approvalRowID(approval.RequestID),
			Author:  ui.AuthorApproval,
			Content: strings.Join(parts, " — "),
			Approval: &ui.Approval{
				BranchID:  approval.BranchID,
				RequestID: approval.RequestID,
				Call:      approval.Call.Name,
				Reason:    approval.Reason,
				Source:    approval.Source.Kind,
			},
		})
	}
	return rows
}

// approvalRowID keys an approval row by its request id. The prefix keeps it
// apart from durable message ids, which the projection owns.
func approvalRowID(requestID string) string {
	return "approval:" + requestID
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

// Transient maps ephemeral session state to throwaway UI status fields.
func Transient(snapshot projection.TransientSnapshot) (*ui.ModelProgress, []ui.Notice) {
	var progress *ui.ModelProgress
	if snapshot.ModelProgress != nil {
		p := snapshot.ModelProgress
		progress = &ui.ModelProgress{Text: p.Text, Thinking: p.Thinking}
		for _, tool := range p.Tools {
			progress.Tools = append(progress.Tools, tool.Name)
		}
	}
	notices := make([]ui.Notice, 0, len(snapshot.Notices))
	for _, notice := range snapshot.Notices {
		notices = append(notices, ui.Notice{Kind: notice.Kind, ID: notice.ID})
	}
	return progress, notices
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
