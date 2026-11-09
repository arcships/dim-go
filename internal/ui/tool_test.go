package ui

import (
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// toolFixture is one tool row with the given status, params and result, and
// nothing else in the conversation. The catalog is left out because its rows
// are list items too, and these tests read the conversation list's items.
func toolFixture(status ToolStatus, params, result string) *State {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
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
// author label, the tool's name, its status and a preview line each of params
// and result.
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

		for _, want := range []string{"bash", tc.label, "Params: go test ./...", "Result: ok dimagent"} {
			if !tt.HasText(want) {
				t.Errorf("%v: card is missing %q, texts %q", tc.status, want, tt.Texts())
			}
		}
	}
}

// TestToolPreviewTextCapsItsLines checks the preview's pure shape: at most
// toolPreviewLines lines, each line cut at toolPreviewRunes runes, with an
// ellipsis saying what was left out and the line breaks that were sent kept.
func TestToolPreviewTextCapsItsLines(t *testing.T) {
	longLine := strings.Repeat("a", 4*toolPreviewRunes)
	cases := []struct {
		name, in, want string
	}{
		{"empty", "", ""},
		{"one line", "go test ./...", "go test ./..."},
		{"three lines", "a\nb\nc", "a\nb\nc"},
		{"past the bound", "a\nb\nc\nd\ne", "a\nb\nc…"},
		{"line breaks kept", "line one\nline two\tline three", "line one\nline two line three"},
		{"windows breaks", "a\r\nb\r\nc", "a\nb\nc"},
		{"outer blank lines", "\n\na\nb\n\n", "a\nb"},
		{"long line", longLine, strings.Repeat("a", toolPreviewRunes) + "…"},
		{"long third line", "a\nb\n" + longLine, "a\nb\n" + strings.Repeat("a", toolPreviewRunes) + "…"},
	}
	for _, tc := range cases {
		if got := previewText(tc.in); got != tc.want {
			t.Errorf("%s: previewText(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

// TestToolRowBoundsItsPreviews checks that the closed card shows at most three
// lines of params and of result, cuts a long line at the card's bound, and
// leaves the lines past the third out of the row entirely.
func TestToolRowBoundsItsPreviews(t *testing.T) {
	long := strings.Repeat("a", 4*toolPreviewRunes)
	wantParams := "Params: " + strings.Repeat("a", toolPreviewRunes) + "…"
	sixLines := "line one\nline two\tline three\nline four\nline five\nline six"
	wantResult := "Result: line one\nline two line three\nline four…"

	tt := ui.NewTester(toolFixture(ToolComplete, long, sixLines).View, 900, 600)

	if !tt.HasText(wantParams) {
		t.Fatalf("long params are not cut at %d runes, texts %q", toolPreviewRunes, tt.Texts())
	}
	if tt.HasText(long) {
		t.Errorf("the whole command shows, texts %q", tt.Texts())
	}
	if !tt.HasText(wantResult) {
		t.Errorf("the result is not drawn as its first %d lines, texts %q", toolPreviewLines, tt.Texts())
	}
	for _, absent := range []string{"line five", "line six"} {
		if tt.HasText(absent) {
			t.Errorf("the preview shows %q past its %d-line bound, texts %q", absent, toolPreviewLines, tt.Texts())
		}
	}
	if tt.HasText("\r") {
		t.Errorf("the preview keeps a carriage return, texts %q", tt.Texts())
	}
}

// TestToolRowPreviewHeightIsBounded checks that the preview's height follows
// the line budget, not the call: three lines and six lines draw the same
// height, a one-line call draws less, and no preview is taller than its three
// lines.
func TestToolRowPreviewHeightIsBounded(t *testing.T) {
	height := func(t *testing.T, result string) (paramRect, resultRect ui.Rect) {
		t.Helper()
		tt := ui.NewTester(toolFixture(ToolComplete, "ls", result).View, 900, 600)
		paramRect, ok := tt.Find("Params: ls")
		if !ok {
			t.Fatalf("params are not drawn, texts %q", tt.Texts())
		}
		resultRect, ok = tt.Find("Result: " + previewText(result))
		if !ok {
			t.Fatalf("the result is not drawn, texts %q", tt.Texts())
		}
		return paramRect, resultRect
	}

	one := "line one"
	three := "line one\nline two\nline three"
	six := "line one\nline two\nline three\nline four\nline five\nline six"
	_, oneRect := height(t, one)
	paramRect, threeRect := height(t, three)
	_, sixRect := height(t, six)

	if threeRect.H <= oneRect.H {
		t.Errorf("three lines draw %.2f high, one line %.2f: the bound is not in use", threeRect.H, oneRect.H)
	}
	if math.Abs(float64(threeRect.H-sixRect.H)) > 0.5 {
		t.Errorf("six lines draw %.2f high, three lines %.2f: the preview grew past its bound", sixRect.H, threeRect.H)
	}
	if sixRect.H > 3*oneRect.H+0.5 {
		t.Errorf("the result draws %.2f high, more than its %d lines of %.2f", sixRect.H, toolPreviewLines, oneRect.H)
	}
	if paramRect.Y >= threeRect.Y {
		t.Errorf("params at y=%.2f are not above the result at y=%.2f: the card lost its order",
			paramRect.Y, threeRect.Y)
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
	if _, ok := tt.AccessByRole(ui.RoleButton, "Retry"); ok {
		t.Errorf("a finished call keeps a retry entry, texts %q", tt.Texts())
	}
}

// TestToolRowDrawsItsFace checks that the card is painted with the semantic
// card face rather than the border tint it used to be: an opaque face covers
// the card's padding and the gaps between its lines, while a 12% tint of the
// border leaves every pixel the color of whatever is under it.
func TestToolRowDrawsItsFace(t *testing.T) {
	for _, dark := range []bool{false, true} {
		s := toolFixture(ToolComplete, "ls", "ok")
		var theme *ui.Theme
		tt := ui.NewTester(func(c *ui.Context) {
			theme = c.Theme()
			s.View(c)
		}, 900, 600)
		tt.SetDark(dark)

		item, ok := tt.AccessByRole(ui.RoleListItem, "")
		if !ok {
			t.Fatalf("dark=%v: the tool row is not a list item, texts %q", dark, tt.Texts())
		}
		face := newTokens(theme).card
		if share := faceShare(tt, item.Bounds, face); share < 0.2 {
			t.Errorf("dark=%v: only %.0f%% of the row is the card face %+v: the card is a tint of what is under it",
				dark, share*100, face)
		}
	}
}

// cardRect is the card's box inside a tool row: the row pads the card by
// timelinePadX and rowPadY, so the list item's own box surrounds it.
func cardRect(item ui.Rect, k tokens) ui.Rect {
	return ui.Rect{
		X: item.X + k.timelinePadX,
		Y: item.Y + k.rowPadY,
		W: item.W - 2*k.timelinePadX,
		H: item.H - 2*k.rowPadY,
	}
}

// edgePixels counts the pixels of face in the outermost band of r, where a
// card's border is drawn and its inset text is not.
func edgePixels(tt *ui.Tester, r ui.Rect, face ui.Color, band int) int {
	img := tt.Image()
	hits := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			edge := x-int(r.X) < band || int(r.X+r.W)-x <= band ||
				y-int(r.Y) < band || int(r.Y+r.H)-y <= band
			if !edge || x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
				continue
			}
			if p := img.RGBAAt(x, y); p.R == face.R && p.G == face.G && p.B == face.B && p.A == face.A {
				hits++
			}
		}
	}
	return hits
}

// TestToolRowErrorDrawsDangerBorder checks the failed call's card: its border
// is painted in the danger color rather than the ordinary hairline, while a
// finished call keeps the neutral border.
func TestToolRowErrorDrawsDangerBorder(t *testing.T) {
	for _, dark := range []bool{false, true} {
		cases := []struct {
			status ToolStatus
			want   bool
		}{
			{ToolError, true},
			{ToolComplete, false},
			{ToolCancelled, false},
		}
		for _, tc := range cases {
			s := toolFixture(tc.status, "go test ./...", "FAIL")
			var theme *ui.Theme
			tt := ui.NewTester(func(c *ui.Context) {
				theme = c.Theme()
				s.View(c)
			}, 900, 600)
			tt.SetDark(dark)
			k := newTokens(theme)

			item, ok := tt.AccessByRole(ui.RoleListItem, "")
			if !ok {
				t.Fatalf("dark=%v status=%v: the row is not a list item, texts %q", dark, tc.status, tt.Texts())
			}
			got := edgePixels(tt, cardRect(item.Bounds, k), k.danger, 3) > 0
			if got != tc.want {
				t.Errorf("dark=%v: a %v card draws a danger border=%v, want %v", dark, tc.status, got, tc.want)
			}
		}
	}
}

// TestToolRowErrorKeepsARetryEntry checks the retry entry of a failed call: it
// is visible, keeps the keyboard focus so its reason is read, is disabled
// because no callback can replay the call, and says why it cannot act. A
// running or finished call keeps no retry entry.
func TestToolRowErrorKeepsARetryEntry(t *testing.T) {
	cases := []struct {
		status ToolStatus
		want   bool
	}{
		{ToolError, true},
		{ToolRunning, false},
		{ToolComplete, false},
		{ToolCancelled, false},
	}
	for _, tc := range cases {
		tt := ui.NewTester(toolFixture(tc.status, "go test ./...", "FAIL").View, 900, 600)
		info, ok := tt.AccessByRole(ui.RoleButton, "Retry")
		if ok != tc.want {
			t.Errorf("%v: the card has a Retry button=%v, want %v, texts %q",
				tc.status, ok, tc.want, tt.Texts())
			continue
		}
		if !tc.want {
			continue
		}
		if !info.Disabled {
			t.Error("the retry entry takes a click although no callback can replay the call")
		}
		if !info.Focusable {
			t.Error("the retry entry is unreachable, so its reason is never read")
		}
		for _, want := range []string{reasonToolRetry, wipBadge} {
			if !strings.Contains(info.Description, want) {
				t.Errorf("the retry entry's description %q does not carry %q", info.Description, want)
			}
		}
		if err := tt.Click("Retry"); err != nil {
			t.Fatalf("clicking the retry entry: %v", err)
		}
		if !tt.HasText("FAIL") {
			t.Errorf("the card changed after a click on an unavailable retry, texts %q", tt.Texts())
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
