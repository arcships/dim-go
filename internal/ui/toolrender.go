package ui

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// The typed tool panels: one panel per family of call, as dim-electron's
// tool-renderers draw a command, a file change and a search apart from a generic
// call. docs/ui-layout-spec.md §4.5 asks for four panels — command, file, search
// and generic — and leaves every other family WIP, so a call this shell cannot
// classify keeps the raw params and result instead of a panel guessed for it.
//
// Data-source gap: the projection folds no typed tool-call shape and the view
// model (internal/viewmodel) forwards only the name, the formatted params and
// the formatted result, so Tool's typed facts are empty in the app today. The
// panels draw the facts a caller does fill, name the ones it does not, and never
// read a value out of the params preview: that preview is bounded for display
// and may be cut mid-value, so parsing it would invent facts.

// toolKind is the family a tool call belongs to, which decides the panel its
// card draws.
type toolKind uint8

const (
	// toolGeneric is a call the shell cannot classify, or one whose family has
	// no panel of its own. Its panel is the raw params and result.
	toolGeneric toolKind = iota
	// toolCommand is a shell command: bash, shell, exec and their aliases.
	toolCommand
	// toolFile is a read, write or edit of one path.
	toolFile
	// toolSearch is a grep, glob or other file search.
	toolSearch
)

// toolMatchListMax bounds how many matched paths a search panel lists. The
// panel says how many more paths the call returned, so the list is bounded
// without the count being bounded with it.
const toolMatchListMax = 8

// toolKindOf classifies a call by its name and, for a name the shell does not
// know, by the first key of its params preview. The view model sorts the params'
// keys, so a call leading with "command", "path" or "pattern" says what it is
// even under a name that means nothing here. A call neither its name nor its
// params describe is generic, and its card shows the raw params and result
// rather than a guessed panel.
func toolKindOf(tool Tool) toolKind {
	if kind, ok := toolKindByName(tool.Name); ok {
		return kind
	}
	return toolKindByParams(tool.Params)
}

// normalizeToolName reduces a tool name to the tool's own word: a provider's
// namespace is dropped ("mcp__filesystem__read_file" → "read_file",
// "functions.bash" → "bash"), and '-' is read as '_' so "read-file" and
// "read_file" classify alike. It is dim-electron's normalizeToolName, which
// splits a name on "__", '.', '/', ':', whitespace and '-'.
func normalizeToolName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if i := strings.LastIndex(name, "__"); i >= 0 {
		name = name[i+2:]
	}
	if i := strings.LastIndexAny(name, "./: \t"); i >= 0 {
		name = name[i+1:]
	}
	return strings.ReplaceAll(name, "-", "_")
}

// toolKindByName maps the tool names dim-electron's renderers classify, and the
// aliases the App Server's own tools use, to the panel they draw. A name this
// shell does not know is not a guess: it falls through to the params, and then
// to the generic panel.
func toolKindByName(name string) (toolKind, bool) {
	switch normalizeToolName(name) {
	case "read", "read_file", "read_text_file", "open", "cat", "view":
		return toolFile, true
	case "write", "write_file", "edit", "edit_file", "create_file", "replace",
		"update", "delete", "apply_patch", "str_replace_editor", "multi_edit", "multiedit":
		return toolFile, true
	case "grep", "rg", "ripgrep", "glob", "find", "list_files", "ls",
		"grep_search", "file_search", "search_files":
		return toolSearch, true
	case "bash", "sh", "zsh", "shell", "terminal", "powershell", "pwsh",
		"exec", "run", "run_command", "exec_command", "execute_command", "shell_command":
		return toolCommand, true
	}
	// A provider that joins its own prefix to the name keeps the family when
	// the name is still the tool's, as dim-electron's file renderer does for
	// "apply_patch".
	raw := strings.ToLower(name)
	switch {
	case strings.Contains(raw, "search_files"):
		return toolSearch, true
	case strings.Contains(raw, "apply_patch"):
		return toolFile, true
	}
	return toolGeneric, false
}

// toolKindByParams reads the family from the first key of a params preview, for
// a call whose name the shell does not know. It reads the key only, never the
// value: the preview is one bounded line, so a value may be cut mid-way and
// must not be parsed.
func toolKindByParams(params string) toolKind {
	preview := strings.TrimSpace(params)
	for _, lead := range [...]struct {
		key  string
		kind toolKind
	}{
		{`{"command":`, toolCommand},
		{`{"cmd":`, toolCommand},
		{`{"argv":`, toolCommand},
		{`{"file_path":`, toolFile},
		{`{"path":`, toolFile},
		{`{"file":`, toolFile},
		{`{"pattern":`, toolSearch},
	} {
		if strings.HasPrefix(preview, lead.key) {
			return lead.kind
		}
	}
	return toolGeneric
}

// toolPanel draws what a tool card's open panel holds: the typed facts of the
// call's own family, then the raw params and result the view model keeps, so a
// card never shows less than the generic one did. Every value is one bounded
// line, as the closed card's previews are.
func toolPanel(c *ui.Context, k tokens, tool Tool) {
	ui.Column(c).Padding(k.cardPadY, k.cardPadX).Gap(k.gap).Children(func() {
		switch toolKindOf(tool) {
		case toolCommand:
			toolCommandFacts(c, k, tool)
		case toolFile:
			toolFileFacts(c, k, tool)
		case toolSearch:
			toolSearchFacts(c, k, tool)
		}
		toolPanelSection(c, k, "Params", panelValue(tool.Params))
		toolPanelSection(c, k, "Result", panelValue(tool.Result))
	})
}

// toolCommandFacts draws a command's panel: the command it ran and its exit
// status. Neither is guessed: a command the view model did not carry is named as
// a gap, and so is an exit code it did not report, so a failed call can never be
// drawn as "Exit code: 0". A call that is still running has no exit code yet, so
// its panel does not claim the fact is missing.
func toolCommandFacts(c *ui.Context, k tokens, tool Tool) {
	var missing []string
	if tool.Command != "" {
		toolPanelSection(c, k, "Command", previewLine(tool.Command))
	} else {
		missing = append(missing, "command")
	}
	if tool.HasExit {
		toolPanelSection(c, k, "Exit code", strconv.Itoa(tool.ExitCode))
	} else if tool.Status != ToolRunning {
		missing = append(missing, "exit code")
	}
	toolGapNote(c, k, missing)
}

// toolFileFacts draws a file call's panel: the path it touched and, when the
// source reported them, the lines it added and removed. A read has no line
// counts, and a change whose source reported none says nothing instead of
// claiming +0/-0, which would read as "nothing changed".
func toolFileFacts(c *ui.Context, k tokens, tool Tool) {
	var missing []string
	if tool.Path != "" {
		toolPanelSection(c, k, "Path", previewLine(tool.Path))
	} else {
		missing = append(missing, "path")
	}
	if tool.HasStats {
		toolPanelSection(c, k, "Lines", fileStats(FileChange{
			Insertions: tool.Insertions,
			Deletions:  tool.Deletions,
		}))
	}
	toolGapNote(c, k, missing)
}

// toolSearchFacts draws a search's panel: the pattern it looked for, how many
// hits it found and the paths it returned. A count the source did not report is
// a gap rather than a count taken from the listed paths, which may be a window
// onto more hits; and a listed path is one bounded line, so a long one cannot
// grow the card.
func toolSearchFacts(c *ui.Context, k tokens, tool Tool) {
	var missing []string
	if tool.Pattern != "" {
		toolPanelSection(c, k, "Pattern", previewLine(tool.Pattern))
	} else {
		missing = append(missing, "pattern")
	}
	switch {
	case tool.HasMatchCount:
		toolPanelSection(c, k, "Matches", matchCountLabel(tool.MatchCount))
	case len(tool.Matches) > 0:
		// The paths are the call's own; the count is not, so the list is titled
		// and left uncounted instead of counted from the window.
		ui.Text(c, "Matches").FontSize(k.smallSize).Bold().TextColor(k.mutedForeground)
	default:
		missing = append(missing, "match count")
	}
	for i, match := range tool.Matches {
		if i == toolMatchListMax {
			ui.Text(c, "… and "+strconv.Itoa(len(tool.Matches)-toolMatchListMax)+" more matches").
				FontSize(k.smallSize).TextColor(k.mutedForeground)
			break
		}
		ui.Text(c, previewLine(match)).FontSize(k.codeFontSize).
			TextColor(k.cardForeground).SingleLine()
	}
	toolGapNote(c, k, missing)
}

// matchCountLabel is a search's hit count, as "1 match" or "3 matches".
func matchCountLabel(count int) string {
	if count == 1 {
		return "1 match"
	}
	return strconv.Itoa(count) + " matches"
}

// toolGapNote names a typed panel's missing facts once, in the shell's WIP
// voice, so the gap is visible and never filled with a guess: a card says what
// it cannot draw instead of drawing a zero or a blank.
func toolGapNote(c *ui.Context, k tokens, missing []string) {
	if len(missing) == 0 {
		return
	}
	ui.Text(c, wipBadge+": the tool view model carries no "+strings.Join(missing, " or ")+".").
		FontSize(k.smallSize).TextColor(k.mutedForeground)
}
