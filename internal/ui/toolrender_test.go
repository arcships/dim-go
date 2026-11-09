package ui

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// panelValueAfter reports whether the frame draws value directly after title, as
// a panel section draws its value under its title. The shell shows "0" and "1"
// elsewhere too — a sidebar count, a shortcut — so a value is checked where it
// stands, not by its text alone.
func panelValueAfter(tt *ui.Tester, title, value string) bool {
	texts := tt.Texts()
	for i, text := range texts {
		if text == title && i+1 < len(texts) && texts[i+1] == value {
			return true
		}
	}
	return false
}

// toolGapPhrase is how a typed panel opens its gap note. The shell draws "WIP"
// elsewhere too — the sidebar's unavailable entries, the status bar — so these
// tests look for the panel's own words rather than the badge on its own.
const toolGapPhrase = "WIP: the tool view model carries no "

// typedToolFixture is one tool row with the given call, and nothing else in the
// conversation: these tests read the card, not a catalog or a turn.
func typedToolFixture(tool Tool) *State {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	s.Messages = []Message{{ID: "t1", Author: AuthorTool, Tool: tool}}
	return s
}

// openToolPanel expands the card named name and returns the tester showing its
// panel. The fold animates open over 200ms, so the frame is advanced until it
// has finished before anything is read.
func openToolPanel(t *testing.T, tool Tool, name string) *ui.Tester {
	t.Helper()
	tt := ui.NewTester(typedToolFixture(tool).View, 900, 700)
	if err := tt.Click(name); err != nil {
		t.Fatalf("clicking the tool card %q: %v", name, err)
	}
	tt.Advance(500 * time.Millisecond)
	tt.Frame()
	return tt
}

// TestToolKindRecognition checks which panel a call gets: its name first, the
// leading key of its params for a name the shell does not know, and the generic
// panel for a call neither describes.
func TestToolKindRecognition(t *testing.T) {
	cases := []struct {
		name   string
		params string
		want   toolKind
	}{
		{"bash", `{"command":"go test ./..."}`, toolCommand},
		{"shell", "", toolCommand},
		{"exec", "", toolCommand},
		{"Bash", "", toolCommand},
		{"run_command", "", toolCommand},
		{"mcp__shell__exec", "", toolCommand},
		{"functions.bash", "", toolCommand},
		{"read", "", toolFile},
		{"read_file", "", toolFile},
		{"write_file", "", toolFile},
		{"edit", "", toolFile},
		{"apply_patch", "", toolFile},
		{"mcp__filesystem__read_file", "", toolFile},
		{"grep", "", toolSearch},
		{"glob", "", toolSearch},
		{"list_files", "", toolSearch},
		{"mystery_tool", `{"command":"ls"}`, toolCommand},
		{"mystery_tool", `{"file_path":"a.go"}`, toolFile},
		{"mystery_tool", `{"path":"a.go"}`, toolFile},
		{"mystery_tool", `{"pattern":"TODO"}`, toolSearch},
		{"mystery_tool", `{"query":"where"}`, toolGeneric},
		{"mystery_tool", `not json at all`, toolGeneric},
		{"", "", toolGeneric},
	}
	for _, tc := range cases {
		got := toolKindOf(Tool{Name: tc.name, Params: tc.params})
		if got != tc.want {
			t.Errorf("toolKindOf(%q, %q) = %v, want %v", tc.name, tc.params, got, tc.want)
		}
	}
}

// TestToolKindDoesNotGuessWebOrWaitingTools checks that a family the spec leaves
// WIP keeps the generic panel: a web search's results are not paths, and a wait
// has neither a command nor a path to show.
func TestToolKindDoesNotGuessWebOrWaitingTools(t *testing.T) {
	for _, name := range []string{"web_search", "webfetch", "wait", "agent", "skill"} {
		if got := toolKindOf(Tool{Name: name}); got != toolGeneric {
			t.Errorf("toolKindOf(%q) = %v, want the generic panel", name, got)
		}
	}
}

// TestToolCommandPanelShowsCommandAndExitCode checks a settled command's panel:
// the command and the exit status the view model carries, then the raw params
// and result, with no gap note standing in for a fact it has.
func TestToolCommandPanelShowsCommandAndExitCode(t *testing.T) {
	tool := Tool{
		Name:     "bash",
		Params:   `{"command":"go test ./..."}`,
		Result:   "FAIL",
		Status:   ToolComplete,
		Command:  "go test ./...",
		ExitCode: 1,
		HasExit:  true,
	}
	tt := openToolPanel(t, tool, "bash")
	for _, want := range []string{"Command", "Exit code", "Params", "Result", "FAIL"} {
		if !tt.HasText(want) {
			t.Errorf("the command panel is missing %q, texts %q", want, tt.Texts())
		}
	}
	if !panelValueAfter(tt, "Exit code", "1") {
		t.Errorf("the exit code 1 is not drawn under its title, texts %q", tt.Texts())
	}
	if tt.HasText(toolGapPhrase) {
		t.Errorf("the command panel names a gap although the call carries its facts, texts %q", tt.Texts())
	}
}

// TestToolCommandPanelShowsAZeroExitCode checks that a reported 0 is a fact and
// not a missing one: a successful command reads "Exit code 0" and no gap note.
func TestToolCommandPanelShowsAZeroExitCode(t *testing.T) {
	tool := Tool{
		Name:    "bash",
		Command: "true",
		Status:  ToolComplete,
		HasExit: true,
	}
	tt := openToolPanel(t, tool, "bash")
	if !tt.HasText("Exit code") {
		t.Fatalf("the exit code line is missing, texts %q", tt.Texts())
	}
	if !panelValueAfter(tt, "Exit code", "0") {
		t.Errorf("the exit code 0 is not drawn under its title, texts %q", tt.Texts())
	}
	if tt.HasText(toolGapPhrase) {
		t.Errorf("a reported exit code is drawn as a gap, texts %q", tt.Texts())
	}
}

// TestToolCommandPanelNamesMissingFacts checks a command the view model does not
// describe: the panel says WIP and names what it cannot draw, and draws no
// command or exit code of its own.
func TestToolCommandPanelNamesMissingFacts(t *testing.T) {
	tt := openToolPanel(t, Tool{Name: "bash", Status: ToolComplete}, "bash")
	for _, absent := range []string{"Command", "Exit code"} {
		if tt.HasText(absent) {
			t.Errorf("the panel claims a %q it was not given, texts %q", absent, tt.Texts())
		}
	}
	if !tt.HasText(toolGapPhrase + "command or exit code.") {
		t.Errorf("the panel does not name both missing facts, texts %q", tt.Texts())
	}
}

// TestToolCommandPanelDoesNotCallARunningCallIncomplete checks that a call still
// running is not told its exit code is missing: it has none yet, and the panel
// says only what it lacks.
func TestToolCommandPanelDoesNotCallARunningCallIncomplete(t *testing.T) {
	tt := openToolPanel(t, Tool{Name: "bash", Command: "sleep 5", Status: ToolRunning}, "bash")
	if !tt.HasText("Command") {
		t.Errorf("the running command is not drawn, texts %q", tt.Texts())
	}
	if tt.HasText("exit code") {
		t.Errorf("a running call is reported as missing its exit code, texts %q", tt.Texts())
	}
	if tt.HasText("Exit code") {
		t.Errorf("a running call claims an exit code, texts %q", tt.Texts())
	}
}

// TestToolFilePanelShowsPathAndLineCounts checks a file change's panel: the path
// it touched and the lines it added and removed, in the same "+N/-M" the
// file-changes block writes.
func TestToolFilePanelShowsPathAndLineCounts(t *testing.T) {
	tool := Tool{
		Name:       "edit_file",
		Params:     `{"file_path":"internal/ui/tool.go","old_string":"a","new_string":"b"}`,
		Status:     ToolComplete,
		Path:       "internal/ui/tool.go",
		Insertions: 12,
		Deletions:  4,
		HasStats:   true,
	}
	tt := openToolPanel(t, tool, "edit_file")
	for _, want := range []string{"Path", "internal/ui/tool.go", "Lines", "+12/-4"} {
		if !tt.HasText(want) {
			t.Errorf("the file panel is missing %q, texts %q", want, tt.Texts())
		}
	}
	if tt.HasText(toolGapPhrase) {
		t.Errorf("the file panel names a gap although the call carries its facts, texts %q", tt.Texts())
	}
}

// TestToolFilePanelOmitsUnreportedCounts checks a read, or a change whose source
// gave no counts: the counts are left out rather than drawn as +0/-0, which
// would claim nothing changed.
func TestToolFilePanelOmitsUnreportedCounts(t *testing.T) {
	tt := openToolPanel(t, Tool{Name: "read_file", Path: "internal/ui/tool.go", Status: ToolComplete}, "read_file")
	if !tt.HasText("Path") {
		t.Fatalf("the path is not drawn, texts %q", tt.Texts())
	}
	for _, absent := range []string{"Lines", "+0/-0"} {
		if tt.HasText(absent) {
			t.Errorf("the panel draws %q it was not given, texts %q", absent, tt.Texts())
		}
	}
}

// TestToolFilePanelNamesAMissingPath checks a file call the view model does not
// describe: the gap is named, and no path or count is invented.
func TestToolFilePanelNamesAMissingPath(t *testing.T) {
	tt := openToolPanel(t, Tool{Name: "write_file", Status: ToolComplete}, "write_file")
	if tt.HasText("Path") || tt.HasText("Lines") {
		t.Errorf("the panel draws a path it was not given, texts %q", tt.Texts())
	}
	if !tt.HasText(toolGapPhrase + "path.") {
		t.Errorf("the panel does not name the missing path, texts %q", tt.Texts())
	}
}

// TestToolSearchPanelShowsPatternAndMatches checks a search's panel: the pattern,
// the hit count and the paths the call returned.
func TestToolSearchPanelShowsPatternAndMatches(t *testing.T) {
	tool := Tool{
		Name:          "grep",
		Pattern:       "TODO",
		Status:        ToolComplete,
		Matches:       []string{"internal/ui/tool.go", "internal/ui/timeline.go"},
		MatchCount:    2,
		HasMatchCount: true,
	}
	tt := openToolPanel(t, tool, "grep")
	for _, want := range []string{
		"Pattern", "TODO", "Matches", "2 matches",
		"internal/ui/tool.go", "internal/ui/timeline.go",
	} {
		if !tt.HasText(want) {
			t.Errorf("the search panel is missing %q, texts %q", want, tt.Texts())
		}
	}
	if tt.HasText(toolGapPhrase) {
		t.Errorf("the search panel names a gap although the call carries its facts, texts %q", tt.Texts())
	}
}

// TestToolSearchPanelBoundsItsPathList checks that a search with many hits lists
// a window of them and says how many more it returned, so a large result cannot
// grow the card without bound.
func TestToolSearchPanelBoundsItsPathList(t *testing.T) {
	matches := make([]string, 12)
	for i := range matches {
		matches[i] = "pkg/file" + strconv.Itoa(i) + ".go"
	}
	tool := Tool{Name: "glob", Pattern: "**/*.go", Status: ToolComplete, Matches: matches}
	tt := openToolPanel(t, tool, "glob")

	if _, ok := tt.Find("pkg/file7.go"); !ok {
		t.Errorf("the eighth match is not listed, texts %q", tt.Texts())
	}
	if _, ok := tt.Find("pkg/file8.go"); ok {
		t.Errorf("the list is not bounded: it shows the ninth match, texts %q", tt.Texts())
	}
	if !tt.HasText("… and 4 more matches") {
		t.Errorf("the panel does not say how many matches it left out, texts %q", tt.Texts())
	}
	// No count was reported, so none is drawn: the window is not the total.
	if _, ok := tt.Find("12 matches"); ok {
		t.Errorf("the panel counts the listed paths as the total, texts %q", tt.Texts())
	}
}

// TestToolSearchPanelNamesMissingFacts checks a search the view model does not
// describe: the pattern and the count are named as gaps, and neither is invented.
func TestToolSearchPanelNamesMissingFacts(t *testing.T) {
	tt := openToolPanel(t, Tool{Name: "grep", Status: ToolComplete}, "grep")
	for _, absent := range []string{"Pattern", "Matches", "0 matches"} {
		if tt.HasText(absent) {
			t.Errorf("the panel draws %q it was not given, texts %q", absent, tt.Texts())
		}
	}
	if !tt.HasText(toolGapPhrase + "pattern or match count.") {
		t.Errorf("the panel does not name both missing facts, texts %q", tt.Texts())
	}
}

// TestToolGenericPanelKeepsRawParamsAndResult checks the fallback: a call the
// shell cannot classify keeps the generic panel, with no typed section and no gap
// note claiming a fact the call never had.
func TestToolGenericPanelKeepsRawParamsAndResult(t *testing.T) {
	tool := Tool{Name: "mystery_tool", Params: `{"query":"where"}`, Result: "done", Status: ToolComplete}
	tt := openToolPanel(t, tool, "mystery_tool")
	for _, want := range []string{"Params", "Result", "done"} {
		if !tt.HasText(want) {
			t.Errorf("the generic panel is missing %q, texts %q", want, tt.Texts())
		}
	}
	for _, absent := range []string{"Command", "Exit code", "Path", "Lines", "Pattern", "Matches", toolGapPhrase} {
		if tt.HasText(absent) {
			t.Errorf("the generic panel draws %q, texts %q", absent, tt.Texts())
		}
	}
}

// TestToolTypedValuesAreBounded checks that a long typed value stays one bounded
// line: the detail a command or a path carries cannot make the panel wider than
// the card's previews do.
func TestToolTypedValuesAreBounded(t *testing.T) {
	long := strings.Repeat("c", 4*toolPreviewRunes)
	want := strings.Repeat("c", toolPreviewRunes) + "…"
	tool := Tool{Name: "bash", Command: long, Status: ToolComplete, HasExit: true}
	tt := openToolPanel(t, tool, "bash")

	if !tt.HasText("Command") {
		t.Fatalf("the command line is not drawn, texts %q", tt.Texts())
	}
	if !tt.HasText(want) {
		t.Fatalf("the command is not cut at %d runes, texts %q", toolPreviewRunes, tt.Texts())
	}
	if tt.HasText(long) {
		t.Errorf("the whole command is drawn, texts %q", tt.Texts())
	}
}

// TestToolCardTypedPanelFoldsAndTakesTheKeyboard checks that the typed panel rides
// the card's existing disclosure: closed by default, reachable with Tab, opened
// and closed with Enter, and its facts built only while it is open.
func TestToolCardTypedPanelFoldsAndTakesTheKeyboard(t *testing.T) {
	s := typedToolFixture(Tool{
		Name:    "bash",
		Command: "go test ./...",
		Status:  ToolComplete,
		HasExit: true,
	})
	tt := ui.NewTester(s.View, 900, 700)

	if tt.HasText("Exit code") {
		t.Fatalf("the closed card draws its typed panel, texts %q", tt.Texts())
	}
	trigger, ok := tt.AccessByRole(ui.RoleDisclosure, "bash")
	if !ok {
		t.Fatalf("the tool card is not a disclosure, texts %q", tt.Texts())
	}
	if trigger.Expanded {
		t.Fatalf("the tool card starts open: %+v", trigger)
	}
	if !trigger.Focusable {
		t.Error("the tool card is not reachable from the keyboard")
	}

	reached := false
	for i := 0; i < 40 && !reached; i++ {
		tt.Key(0, ui.KeyTab)
		reached = tt.Focused("bash")
	}
	if !reached {
		t.Fatalf("Tab does not reach the tool card, texts %q", tt.Texts())
	}

	tt.Key(0, ui.KeyEnter)
	tt.Advance(500 * time.Millisecond)
	tt.Frame()
	if open, ok := tt.AccessByRole(ui.RoleDisclosure, "bash"); !ok || !open.Expanded {
		t.Fatalf("Enter does not open the typed panel: %+v, %v", open, ok)
	}
	for _, want := range []string{"Command", "go test ./...", "Exit code"} {
		if !tt.HasText(want) {
			t.Fatalf("the open panel is missing %q, texts %q", want, tt.Texts())
		}
	}

	tt.Key(0, ui.KeyEnter)
	tt.Advance(500 * time.Millisecond)
	tt.Frame()
	if tt.HasText("Exit code") {
		t.Errorf("closing the card leaves its typed panel built, texts %q", tt.Texts())
	}
}
