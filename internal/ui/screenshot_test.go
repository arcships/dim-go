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
// sessions, one row of each author (user, assistant, tool, approval) and a
// composer draft. It is view state only, hand-filled exactly as the shell's
// Milestone 1 fixture is; no App Server, storage or log is read.
func screenshotFixture() *State {
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
			},
			{
				ID:     "m3",
				Author: AuthorTool,
				Tool: Tool{
					Name:   "bash",
					Params: "go test ./internal/ui -run TestScreenshotFrames",
					Result: "ok  github.com/arcships/dimagent-mygo/internal/ui",
					Status: ToolComplete,
				},
			},
			{
				ID:      "m4",
				Author:  AuthorApproval,
				Content: "exec — run the screenshot test and write both PNGs",
				Approval: &Approval{
					BranchID:  "main",
					RequestID: "req-screenshot-1",
					Call:      "exec",
					Reason:    "run the screenshot test and write both PNGs",
				},
			},
		},
		Draft: "Rerun the suite and attach the new PNGs.",
	}
	// A wired callback turns the approval row's decisions into real buttons;
	// the fixture records nothing because it sends nothing.
	s.OnApproval = func(string, string, ApprovalDecision) {}
	s.OnSubmit = func(string) {}
	return s
}

// screenshotFrames renders the fixture in both appearances. The two testers
// are separate, so a click or a draft edit in one cannot leak into the other.
func screenshotFrames(t *testing.T) map[string]*ui.Tester {
	t.Helper()
	light := ui.NewTester(screenshotFixture().View, screenshotWidth, screenshotHeight)
	dark := ui.NewTester(screenshotFixture().View, screenshotWidth, screenshotHeight)
	dark.SetDark(true)
	return map[string]*ui.Tester{"light": light, "dark": dark}
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

// TestScreenshotFrames checks the evidence frame at rest: it renders the
// fixture's rows in both themes at 1440×900, it is not blank, and light and
// dark differ. When DIMAGENT_UI_SCREENSHOT_DIR is set it also writes the two
// PNGs the docs/evidence/ui README points at; otherwise it touches no file.
func TestScreenshotFrames(t *testing.T) {
	frames := screenshotFrames(t)
	for _, name := range []string{"light", "dark"} {
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

	if bytes.Equal(frames["light"].Image().Pix, frames["dark"].Image().Pix) {
		t.Error("light and dark frames are pixel-identical")
	}

	dir := os.Getenv(screenshotEnv)
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create %s: %v", dir, err)
	}
	for _, name := range []string{"light", "dark"} {
		path := filepath.Join(dir, "shell-"+name+".png")
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
