package ui

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// turnFixture is one assistant turn with every block kind in order — thinking,
// text, a tool call and file changes — after one user row. It is view state
// only, hand-filled: no App Server, storage or log is read.
//
// The file-changes block has no projection source yet (see Block.Files): the
// fixture is the only place it comes from today, and the view must not pretend
// a daemon reported it.
func turnFixture() *State {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	s.Messages = []Message{
		{ID: "u1", Author: AuthorUser, Content: "why does the screenshot test flake?"},
		{
			ID:     "a1",
			Author: AuthorAssistant,
			Blocks: []Block{
				{
					ID:        "b-thinking",
					Kind:      BlockThinking,
					Content:   "the frame settles before capture",
					StartedAt: 1_000,
					EndedAt:   5_000,
				},
				{ID: "b-text", Kind: BlockText, Content: "The frame is settled before capture."},
				{
					ID:   "b-tool",
					Kind: BlockTool,
					Tool: Tool{
						Name:   "bash",
						Params: "go test ./internal/ui",
						Result: "ok dimagent",
						Status: ToolComplete,
					},
				},
				{
					ID:   "b-files",
					Kind: BlockFileChanges,
					Files: []FileChange{
						{Path: "internal/ui/shell.go", Status: FileModified, Insertions: 12, Deletions: 4, HasStats: true},
						{Path: "docs/ui-layout-spec.md", Status: FileAdded, Insertions: 3, HasStats: true},
						{Path: "internal/ui/legacy.go", Status: FileDeleted, Deletions: 9, HasStats: true},
					},
				},
			},
		},
	}
	return s
}

// TestThinkingBlockIsCollapsedByDefaultAndExpands checks the thinking fold: the
// header is a disclosure reading "Thought for Ns", the reasoning text is not
// built while it is closed, and a click opens and closes it.
func TestThinkingBlockIsCollapsedByDefaultAndExpands(t *testing.T) {
	s := turnFixture()
	tt := ui.NewTester(s.View, 900, 700)

	if !tt.HasText("Thought for 4s") {
		t.Fatalf("the thinking header is missing, texts %q", tt.Texts())
	}
	if tt.HasText("the frame settles before capture") {
		t.Fatalf("the thinking text shows while the block is closed, texts %q", tt.Texts())
	}
	trigger, ok := tt.AccessByRole(ui.RoleDisclosure, "Thought for 4s")
	if !ok {
		t.Fatalf("the thinking header is not a disclosure, texts %q", tt.Texts())
	}
	if trigger.Expanded {
		t.Fatalf("the thinking block starts open: %+v", trigger)
	}
	if !trigger.Focusable {
		t.Errorf("the thinking header is not reachable from the keyboard")
	}

	if err := tt.Click("Thought for 4s"); err != nil {
		t.Fatal(err)
	}
	// Let the fold finish opening: the panel grows with the disclosure's
	// animation, so a frame at once would measure it at height 0.
	tt.Advance(500 * time.Millisecond)
	tt.Frame()
	if !tt.HasText("the frame settles before capture") {
		t.Fatalf("opening the thinking block does not show its text, texts %q", tt.Texts())
	}
	if open, ok := tt.AccessByRole(ui.RoleDisclosure, "Thought for 4s"); !ok || !open.Expanded {
		t.Fatalf("the open thinking header is not reported as expanded: %+v, %v", open, ok)
	}

	if err := tt.Click("Thought for 4s"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(500 * time.Millisecond)
	tt.Frame()
	if tt.HasText("the frame settles before capture") {
		t.Fatalf("closing the thinking block leaves its text built, texts %q", tt.Texts())
	}
}

// TestThinkingBlockSaysThinkingWhileItRuns checks the header of a block the
// projection has not closed: it says "Thinking", not a duration the facts do
// not have yet.
func TestThinkingBlockSaysThinkingWhileItRuns(t *testing.T) {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	s.Messages = []Message{{
		ID:     "a1",
		Author: AuthorAssistant,
		Blocks: []Block{{ID: "b1", Kind: BlockThinking, Content: "still going", Running: true, StartedAt: 1_000}},
	}}
	tt := ui.NewTester(s.View, 900, 700)
	if !tt.HasText("Thinking") {
		t.Fatalf("a running thinking block has no header, texts %q", tt.Texts())
	}
	if tt.HasText("Thought for") {
		t.Fatalf("a running thinking block claims a settled duration, texts %q", tt.Texts())
	}
}

// TestToolBlockExpands checks the tool card's fold: the header names the call
// and its status, the closed card keeps one-line previews, and opening it shows
// the bounded params and result panel instead.
func TestToolBlockExpands(t *testing.T) {
	s := turnFixture()
	tt := ui.NewTester(s.View, 900, 700)

	for _, want := range []string{"bash", "Complete", "Params: go test ./internal/ui", "Result: ok dimagent"} {
		if !tt.HasText(want) {
			t.Fatalf("the closed tool card is missing %q, texts %q", want, tt.Texts())
		}
	}
	trigger, ok := tt.AccessByRole(ui.RoleDisclosure, "bash")
	if !ok {
		t.Fatalf("the tool header is not a disclosure, texts %q", tt.Texts())
	}
	if trigger.Expanded {
		t.Fatalf("the tool card starts open: %+v", trigger)
	}
	if !trigger.Focusable {
		t.Errorf("the tool header is not reachable from the keyboard")
	}

	if err := tt.Click("bash"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(500 * time.Millisecond)
	tt.Frame()
	if open, ok := tt.AccessByRole(ui.RoleDisclosure, "bash"); !ok || !open.Expanded {
		t.Fatalf("the open tool header is not reported as expanded: %+v, %v", open, ok)
	}
	for _, want := range []string{"Params", "Result", "go test ./internal/ui", "ok dimagent"} {
		if !tt.HasText(want) {
			t.Fatalf("the open tool panel is missing %q, texts %q", want, tt.Texts())
		}
	}
	// The panel is capped at 240 DIPs, as the spec's max-h-[240px] is: its
	// content stays inside that box instead of growing the row without bound.
	panel, ok := tt.Find("Result")
	if !ok {
		t.Fatalf("the open panel has no result section, texts %q", tt.Texts())
	}
	if panel.H <= 0 {
		t.Errorf("the tool panel is not laid out: %.1f high", panel.H)
	}
	if panel.H > 240 {
		t.Errorf("the tool panel is %.1f high, want at most 240", panel.H)
	}
}

// TestToolRowKeepsItsStatusLive checks that the standalone tool row still
// exposes a running call as a busy live region beside the disclosure, so
// assistive technology hears the work starting and the button's name stays the
// call's.
func TestToolRowKeepsItsStatusLive(t *testing.T) {
	s := toolFixture(ToolRunning, "pnpm test", "")
	tt := ui.NewTester(s.View, 900, 600)

	status, ok := tt.AccessByRole(ui.RoleStatus, "Running")
	if !ok || !status.Busy {
		t.Fatalf("a running call is not a busy live region: %+v, %v", status, ok)
	}
	if _, ok := tt.AccessByRole(ui.RoleDisclosure, "bash"); !ok {
		t.Fatalf("the running card's header is not a disclosure, texts %q", tt.Texts())
	}
}

// TestFileChangesRenderAsCompactRows checks the file-changes block: the header
// counts the files, each file is one line with its status letter, path and
// counts, and the block offers no fold of its own.
func TestFileChangesRenderAsCompactRows(t *testing.T) {
	s := turnFixture()
	tt := ui.NewTester(s.View, 900, 700)

	for _, want := range []string{
		"3 files changed",
		"internal/ui/shell.go", "M", "+12/-4",
		"docs/ui-layout-spec.md", "A", "+3/-0",
		"internal/ui/legacy.go", "D", "+0/-9",
	} {
		if !tt.HasText(want) {
			t.Fatalf("the file-changes block is missing %q, texts %q", want, tt.Texts())
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleDisclosure, "3 files changed"); ok {
		t.Errorf("the file-changes header is a fold, but a file list is text")
	}
	// A file row is a compact line: its text stays on one line, well under a
	// block header's room plus a block's own height.
	row, ok := tt.Find("internal/ui/shell.go")
	if !ok {
		t.Fatalf("the changed path is not drawn, texts %q", tt.Texts())
	}
	if row.H > 24 {
		t.Errorf("a file row is %.1f high, want a compact single line", row.H)
	}
	letter, ok := tt.Find("M")
	if !ok {
		t.Fatalf("the status letter is not drawn, texts %q", tt.Texts())
	}
	if letter.Y < row.Y-1 || letter.Y > row.Y+row.H+1 {
		t.Errorf("the status letter at y=%.1f is not on its file row at y=%.1f", letter.Y, row.Y)
	}
}

// TestTurnBlocksKeepTheirOrder checks that a turn's blocks render in the
// projection's order, from the thinking header down to the file changes.
func TestTurnBlocksKeepTheirOrder(t *testing.T) {
	s := turnFixture()
	tt := ui.NewTester(s.View, 900, 700)

	want := []string{
		"Thought for 4s",
		"The frame is settled before capture.",
		"bash",
		"3 files changed",
	}
	var prev float32
	for i, text := range want {
		rect, ok := tt.Find(text)
		if !ok {
			t.Fatalf("block %d (%q) is not drawn, texts %q", i, text, tt.Texts())
		}
		if i > 0 && rect.Y <= prev {
			t.Fatalf("block %q at y=%.1f, not after the block before it at %.1f", text, rect.Y, prev)
		}
		prev = rect.Y
	}
}

// TestTimelineKeyboardReachability checks that both folds are Tab stops and
// that Space and Enter toggle them while they have the focus, so a keyboard or
// screen-reader user can read the reasoning and the call's details.
func TestTimelineKeyboardReachability(t *testing.T) {
	s := turnFixture()
	tt := ui.NewTester(s.View, 900, 700)

	// Tab from the start of the window until the thinking header has the focus,
	// so the test does not depend on how many entries precede it.
	focused := false
	for range 30 {
		tt.Key(0, ui.KeyTab)
		if tt.Focused("Thought for 4s") {
			focused = true
			break
		}
	}
	if !focused {
		t.Fatalf("Tab never reaches the thinking header, texts %q", tt.Texts())
	}

	tt.Key(0, ui.KeySpace)
	tt.Advance(500 * time.Millisecond)
	tt.Frame()
	if open, ok := tt.AccessByRole(ui.RoleDisclosure, "Thought for 4s"); !ok || !open.Expanded {
		t.Fatalf("Space does not open the thinking block: %+v, %v", open, ok)
	}

	tt.Key(0, ui.KeyTab)
	if !tt.Focused("bash") {
		t.Fatalf("Tab does not reach the tool header after the thinking header, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	tt.Advance(500 * time.Millisecond)
	tt.Frame()
	if open, ok := tt.AccessByRole(ui.RoleDisclosure, "bash"); !ok || !open.Expanded {
		t.Fatalf("Enter does not open the tool card: %+v, %v", open, ok)
	}
	if !tt.HasText("ok dimagent") {
		t.Fatalf("the opened tool panel is not built, texts %q", tt.Texts())
	}
}

// TestErrorAndRetryAreTheirOwnRows checks the two end-of-turn notices: a failed
// turn is an alert row in the danger color, and a retry notice is a warning
// row, each with its own author label rather than folded into the reply.
func TestErrorAndRetryAreTheirOwnRows(t *testing.T) {
	s := fixture()
	s.Sessions = nil
	s.Selected = ""
	s.Messages = []Message{
		{ID: "e1", Author: AuthorError, Content: "the model call failed"},
		{ID: "r1", Author: AuthorRetry, Content: "retrying the turn"},
	}
	tt := ui.NewTester(s.View, 900, 600)

	for _, want := range []string{"Error", "the model call failed", "Retry", "retrying the turn"} {
		if !tt.HasText(want) {
			t.Fatalf("the notice rows are missing %q, texts %q", want, tt.Texts())
		}
	}
	alert, ok := tt.AccessByRole(ui.RoleAlert, "the model call failed")
	if !ok {
		t.Fatalf("the failed turn is not an alert region, texts %q", tt.Texts())
	}
	if strings.Contains(alert.Name, "retrying the turn") {
		t.Errorf("the alert region holds the retry notice too: %q", alert.Name)
	}
}

// TestUserRowIsARightAlignedBubble checks the user row's shape from the spec: at
// most 85% of the row, on the muted face, ending at the row's right edge.
func TestUserRowIsARightAlignedBubble(t *testing.T) {
	s := turnFixture()
	tt := ui.NewTester(s.View, 900, 700)

	text, ok := tt.Find("why does the screenshot test flake?")
	if !ok {
		t.Fatalf("the user row is not drawn, texts %q", tt.Texts())
	}
	if text.W > 900*0.85 {
		t.Errorf("the user bubble text is %.1f wide, beyond 85%% of a 900-wide window", text.W)
	}
	// The bubble's text ends near the conversation's right edge: the row pads
	// it by 14 DIPs and the sidebar is 256 wide.
	if right := text.X + text.W; right < 900-256-30 {
		t.Errorf("the user bubble ends at x=%.1f, not against the conversation's right edge", right)
	}
}

// TestTimelineKeepsTheReferenceContentInset checks the message column's
// horizontal room against the reference's 800×720 frame: the assistant text
// starts 25 DIPs inside the card and the user bubble ends 25 DIPs inside it,
// while the bubble keeps its 85% cap.
func TestTimelineKeepsTheReferenceContentInset(t *testing.T) {
	const (
		width  = 800
		height = 720
	)
	s := &State{
		Sessions:       []Session{{ID: "s1", Title: "Polish the screenshot shell", Group: "dimagent-mygo"}},
		Selected:       "s1",
		Connection:     ConnReady,
		SelectedBranch: mainBranchID,
		Messages: []Message{
			{ID: "m1", Author: AuthorUser, Content: "Explain what this repository does in one paragraph."},
			{ID: "m2", Author: AuthorAssistant, Content: "Echo: Explain what this repository does in one paragraph."},
		},
	}
	tt := ui.NewTester(s.View, width, height)
	k := newTokens(ui.LightTheme())

	card, ok := tt.AccessByRole(ui.RoleGroup, "Main area")
	if !ok {
		t.Fatalf("the conversation has no card, texts %q", tt.Texts())
	}
	// The card's outer edge is what the reference measures from; the hairline
	// and the inner edge make up the rest of the 24.5 DIP inset.
	inset := conversationInsetX + k.hairline
	reply := textBox(t, tt, "Echo: Explain what this repository does in one paragraph.")
	if left := reply.X - card.Bounds.X; math.Abs(float64(left-inset)) > 1 {
		t.Errorf("the assistant text starts %.1f inside the card, want the reference's ≈%v",
			left, inset)
	}
	bubble := textBox(t, tt, "Explain what this repository does in one paragraph.")
	right := card.Bounds.X + card.Bounds.W - (bubble.X + bubble.W + k.timelineUserPadX)
	if math.Abs(float64(right-inset)) > 1 {
		t.Errorf("the user bubble ends %.1f inside the card, want the reference's ≈%v",
			right, inset)
	}
	if bubble.W > width*k.timelineUserWidth/100 {
		t.Errorf("the bubble text is %.1f wide, beyond the 85%% cap", bubble.W)
	}
}
