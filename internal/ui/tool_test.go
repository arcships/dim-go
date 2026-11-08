package ui

import (
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// toolFixture is one tool row with the given status, params and result, and
// nothing else in the conversation.
func toolFixture(status ToolStatus, params, result string) *State {
	s := fixture()
	s.Messages = []Message{{
		ID:     "t1",
		Author: AuthorTool,
		Tool: Tool{
			Name:   "bash",
			Params: params,
			Result: result,
			Status: status,
		},
	}}
	return s
}

// TestToolRowRendersEachStatus checks the card's parts for every status: the
// author label, the tool's name, its status and one line each of params and
// result.
func TestToolRowRendersEachStatus(t *testing.T) {
	cases := []struct {
		status ToolStatus
		label  string
	}{
		{ToolRunning, "Running"},
		{ToolComplete, "Complete"},
		{ToolError, "Error"},
		{ToolCancelled, "Cancelled"},
		{ToolUnknown, "Unknown"},
	}
	for _, tc := range cases {
		s := toolFixture(tc.status, "go test ./...", "ok dimagent")
		tt := ui.NewTester(s.View, 900, 600)

		for _, want := range []string{"Tool", "bash", tc.label, "Params: go test ./...", "Result: ok dimagent"} {
			if !tt.HasText(want) {
				t.Errorf("%v: card is missing %q, texts %q", tc.status, want, tt.Texts())
			}
		}
	}
}

// TestToolRowBoundsItsPreviews checks that a long or multi-line command and
// result stay one line each: whitespace collapses, the preview stops at the
// card's bound, and the row is no taller than a row with a short preview.
func TestToolRowBoundsItsPreviews(t *testing.T) {
	// The bound the card promises, pinned here: changing it is a deliberate
	// decision about how much of a call a row shows.
	const previewRunes = 96

	long := strings.Repeat("a", 4*previewRunes)
	wantParams := strings.Repeat("a", previewRunes) + "…"

	short := toolFixture(ToolComplete, "ls", "two files")
	longState := toolFixture(ToolComplete, long, "line one\nline two\tline three")
	shortTT := ui.NewTester(short.View, 900, 600)
	longTT := ui.NewTester(longState.View, 900, 600)

	if !longTT.HasText("Params: " + wantParams) {
		t.Fatalf("long params are not cut at %d runes, texts %q", previewRunes, longTT.Texts())
	}
	if longTT.HasText(long) {
		t.Errorf("the whole command shows, texts %q", longTT.Texts())
	}
	if !longTT.HasText("Result: line one line two line three") {
		t.Errorf("multi-line result is not one line, texts %q", longTT.Texts())
	}
	if longTT.HasText("line one\n") || longTT.HasText("\t") {
		t.Errorf("the result keeps its line breaks and tabs, texts %q", longTT.Texts())
	}

	// A long preview and a short one draw the same single line, so a call
	// cannot make its row taller.
	shortRect, ok := shortTT.Find("Params: ls")
	if !ok {
		t.Fatalf("short params are not drawn, texts %q", shortTT.Texts())
	}
	longRect, ok := longTT.Find("Params: " + wantParams)
	if !ok {
		t.Fatalf("long params are not drawn, texts %q", longTT.Texts())
	}
	if math.Abs(float64(shortRect.H-longRect.H)) > 0.5 {
		t.Errorf("long params draw %.2f high, short ones %.2f: the row grew with its content",
			longRect.H, shortRect.H)
	}
	if longRect.H > 40 {
		t.Errorf("params line is %.2f high, so it wrapped", longRect.H)
	}
	resultRect, ok := longTT.Find("Result: line one line two line three")
	if !ok {
		t.Fatalf("the result is not drawn, texts %q", longTT.Texts())
	}
	if longRect.Y >= resultRect.Y {
		t.Errorf("params at y=%.2f are not above the result at y=%.2f: the card lost its order",
			longRect.Y, resultRect.Y)
	}
}

// TestToolRowEmptyParts checks the card's shape when the view model leaves a
// part empty: the lines stay, so rows keep their height and their reading.
func TestToolRowEmptyParts(t *testing.T) {
	s := fixture()
	s.Messages = []Message{{ID: "t1", Author: AuthorTool, Tool: Tool{Status: ToolComplete}}}
	tt := ui.NewTester(s.View, 900, 600)

	for _, want := range []string{"(unnamed tool)", "Params: (none)", "Result: (none)"} {
		if !tt.HasText(want) {
			t.Errorf("card is missing %q, texts %q", want, tt.Texts())
		}
	}
}

// TestToolRowAccessibility checks what assistive technology reads of a tool
// row: the list item's name holds the tool and its status, and only a running
// call is a live, busy status region.
func TestToolRowAccessibility(t *testing.T) {
	running := ui.NewTester(toolFixture(ToolRunning, "pnpm test", "").View, 900, 600)
	item, ok := running.AccessByRole(ui.RoleListItem, "")
	if !ok {
		t.Fatalf("the tool row is not a list item, texts %q", running.Texts())
	}
	for _, want := range []string{"bash", "Running"} {
		if !strings.Contains(item.Name, want) {
			t.Errorf("the tool row's name %q does not hold %q", item.Name, want)
		}
	}
	status, ok := running.AccessByRole(ui.RoleStatus, "Running")
	if !ok {
		t.Fatalf("a running call is not a live status region, texts %q", running.Texts())
	}
	if !status.Busy {
		t.Errorf("the running status is not busy: %+v", status)
	}

	// A finished call says nothing: no live region, so a frame that rebuilds
	// the row cannot speak again.
	done := ui.NewTester(toolFixture(ToolComplete, "pnpm test", "ok").View, 900, 600)
	if _, ok := done.AccessByRole(ui.RoleStatus, "Complete"); ok {
		t.Errorf("a finished call is a live region, texts %q", done.Texts())
	}
	doneItem, ok := done.AccessByRole(ui.RoleListItem, "")
	if !ok {
		t.Fatalf("the finished row is not a list item, texts %q", done.Texts())
	}
	if !strings.Contains(doneItem.Name, "Complete") {
		t.Errorf("the finished row's name %q does not hold its status", doneItem.Name)
	}
}

// TestToolRowNotRunningNeverClaimsWork checks that a row the view model leaves
// undescribed is not announced as running: the zero ToolStatus is unknown.
func TestToolRowNotRunningNeverClaimsWork(t *testing.T) {
	var zero Tool
	if zero.Status == ToolRunning {
		t.Fatal("the zero Tool reports a running call")
	}
	tt := ui.NewTester(fixture().View, 900, 600)
	if _, ok := tt.AccessByRole(ui.RoleStatus, "Running"); ok {
		t.Errorf("an idle shell announces a running tool, texts %q", tt.Texts())
	}
}
