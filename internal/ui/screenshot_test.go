package ui

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The screenshot evidence frame is fixed: both themes render the same fixture
// at the same size, so a diff between two generated files is a rendering
// change and not a change of the input.
const (
	screenshotWidth  = 1440
	screenshotHeight = 900
)

// screenshotEnv names the directory the test writes shell-light.png and
// shell-dark.png into. The test only asserts the frames by default; it writes
// files only when the directory is set, so `go test ./internal/ui` stays
// read-only. The name is the project's own: mygo's internal MYGO_UI_PNG
// variable belongs to the module's own tests and is not read here.
const screenshotEnv = "DIMAGENT_UI_SCREENSHOT_DIR"

// screenshotFixture is a fixed shell state for the evidence images: two
// sessions, one user row, one assistant turn with every timeline block kind —
// thinking, text, two tool calls and file changes — and, when withApproval is
// set, one pending approval that takes the composer's slot. Otherwise the frame
// shows the composer and its draft. It is view state only, hand-filled exactly
// as the shell's Milestone 1 fixture is; no App Server, storage or log is read.
func screenshotFixture(withApproval bool) *State {
	s := &State{
		Sessions: []Session{
			{ID: "s1", Title: "Fix flaky screenshot test"},
			{ID: "s2", Title: "Review App Server reconnect"},
		},
		Selected:   "s1",
		Connection: ConnReady,
		Run:        RunIdle,
		Messages: []Message{
			{
				ID:      "m1",
				Author:  AuthorUser,
				Content: "Show me why the screenshot test flakes on macOS, then rerun the suite.",
			},
			{
				ID:     "m2",
				Author: AuthorAssistant,
				Content: "The frame is settled before capture:\n\n" +
					"- fixed 1440×900 view\n" +
					"- no timers or animation\n" +
					"- one fixture for both themes\n\n" +
					"```go\nimg := tt.Image()\n```",
				Blocks: []Block{
					{
						ID:        "b-thinking",
						Kind:      BlockThinking,
						Content:   "The flake comes from measuring the frame before fonts settle. Capture only after two frames with no pending work.",
						StartedAt: 1_000,
						EndedAt:   6_000,
					},
					{
						ID:   "b-text",
						Kind: BlockText,
						Content: "The frame is settled before capture:\n\n" +
							"- fixed 1440×900 view\n" +
							"- no timers or animation\n" +
							"- one fixture for both themes\n\n" +
							"```go\nimg := tt.Image()\n```",
					},
					{
						ID:   "b-tool-bash",
						Kind: BlockTool,
						Tool: Tool{
							Name:   "bash",
							Params: "go test ./internal/ui -run TestScreenshotFrames",
							Result: "ok  github.com/arcships/dimagent-mygo/internal/ui",
							Status: ToolComplete,
						},
					},
					{
						ID:   "b-tool-read",
						Kind: BlockTool,
						Tool: Tool{
							Name:   "read_file",
							Params: `{"path":"internal/ui/screenshot_test.go"}`,
							Result: "screenshot_test.go, 160 lines",
							Status: ToolComplete,
						},
					},
					{
						ID:   "b-files",
						Kind: BlockFileChanges,
						Files: []FileChange{
							{Path: "internal/ui/screenshot_test.go", Status: FileModified, Insertions: 42, Deletions: 6, HasStats: true},
							{Path: "docs/evidence/ui/shell-light.png", Status: FileModified, Insertions: 0, Deletions: 0},
						},
					},
				},
			},
		},
		Draft: "Rerun the suite and attach the new PNGs.",
	}
	if withApproval {
		s.Messages = append(s.Messages, Message{
			ID:      "m3",
			Author:  AuthorApproval,
			Content: "exec — run the screenshot test and write both PNGs",
			Approval: &Approval{
				BranchID:  "main",
				RequestID: "req-screenshot-1",
				Call:      "exec",
				Reason:    "run the screenshot test and write both PNGs",
			},
		})
	}
	// A wired callback turns the approval decisions into real buttons; the
	// fixture records nothing because it sends nothing.
	s.OnApproval = func(string, string, ApprovalDecision) {}
	s.OnSubmit = func(string) {}
	return s
}

// screenshotSettingsFixture is the appearance surface on the same catalog, so the
// settings frame differs from the conversation frame in the main column only.
func screenshotSettingsFixture() *State {
	s := screenshotFixture(false)
	s.ShowSettings = true
	return s
}

// screenshotFrames renders the evidence states in both appearances: the
// approval state (a pending request takes the composer's slot) and the composer
// state (no pending request, the draft field is visible). Every tester is
// separate, so a click or a draft edit in one cannot leak into another.
func screenshotFrames(t *testing.T) map[string]*ui.Tester {
	t.Helper()
	light := ui.NewTester(screenshotFixture(true).View, screenshotWidth, screenshotHeight)
	dark := ui.NewTester(screenshotFixture(true).View, screenshotWidth, screenshotHeight)
	dark.SetDark(true)
	lightComposer := ui.NewTester(screenshotFixture(false).View, screenshotWidth, screenshotHeight)
	darkComposer := ui.NewTester(screenshotFixture(false).View, screenshotWidth, screenshotHeight)
	darkComposer.SetDark(true)
	lightSettings := ui.NewTester(screenshotSettingsFixture().View, screenshotWidth, screenshotHeight)
	darkSettings := ui.NewTester(screenshotSettingsFixture().View, screenshotWidth, screenshotHeight)
	darkSettings.SetDark(true)
	return map[string]*ui.Tester{
		"light":          light,
		"dark":           dark,
		"light-composer": lightComposer,
		"dark-composer":  darkComposer,
		"light-settings": lightSettings,
		"dark-settings":  darkSettings,
	}
}

// imageHasContent reports whether the frame draws more than one color, which a
// view that rendered nothing or one flat fill would not.
func imageHasContent(img *image.RGBA) bool {
	if len(img.Pix) == 0 {
		return false
	}
	first := img.Pix[:4]
	for i := 4; i < len(img.Pix); i += 4 {
		if !bytes.Equal(img.Pix[i:i+4], first) {
			return true
		}
	}
	return false
}

// TestScreenshotFrames checks the evidence frames at rest: both states render in
// both themes at 1440×900, none is blank, and light and dark differ. When
// DIMAGENT_UI_SCREENSHOT_DIR is set it also writes the PNGs the
// docs/evidence/ui README points at; otherwise it touches no file.
func TestScreenshotFrames(t *testing.T) {
	frames := screenshotFrames(t)
	names := []string{"light", "dark", "light-composer", "dark-composer", "light-settings", "dark-settings"}
	for _, name := range names {
		tt := frames[name]
		img := tt.Image()
		if img == nil {
			t.Fatalf("%s: no image", name)
		}
		bounds := img.Bounds()
		if bounds.Dx() != screenshotWidth || bounds.Dy() != screenshotHeight {
			t.Errorf("%s: image is %dx%d, want %dx%d", name, bounds.Dx(), bounds.Dy(), screenshotWidth, screenshotHeight)
		}
		if !imageHasContent(img) {
			t.Errorf("%s: image is blank", name)
		}
	}

	for _, pair := range [][2]string{{"light", "dark"}, {"light-composer", "dark-composer"}, {"light-settings", "dark-settings"}} {
		if bytes.Equal(frames[pair[0]].Image().Pix, frames[pair[1]].Image().Pix) {
			t.Errorf("%s and %s frames are pixel-identical", pair[0], pair[1])
		}
	}
	if bytes.Equal(frames["light"].Image().Pix, frames["light-composer"].Image().Pix) {
		t.Error("approval and composer frames are pixel-identical")
	}

	dir := os.Getenv(screenshotEnv)
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create %s: %v", dir, err)
	}
	files := map[string]string{
		"light":          "shell-light.png",
		"dark":           "shell-dark.png",
		"light-composer": "shell-light-composer.png",
		"dark-composer":  "shell-dark-composer.png",
		"light-settings": "shell-light-settings.png",
		"dark-settings":  "shell-dark-settings.png",
	}
	for _, name := range names {
		path := filepath.Join(dir, files[name])
		file, err := os.Create(path)
		if err != nil {
			t.Fatalf("create %s: %v", path, err)
		}
		if err := png.Encode(file, frames[name].Image()); err != nil {
			file.Close()
			t.Fatalf("encode %s: %v", path, err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close %s: %v", path, err)
		}
		t.Logf("wrote %s", path)
	}
}
