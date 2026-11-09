package ui

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// motionEvidenceEnv names the directory the motion frames are written into.
// Like the static screenshot evidence, the tests only assert by default and
// write files when the directory is set, so `go test ./internal/ui` stays
// read-only.
const motionEvidenceEnv = "DIMAGENT_UI_MOTION_DIR"

// motionFrameSteps are the frames the evidence records: the start, the two
// intermediate frames the reference's 250 ms layout step shows at 75 and 150,
// and the settled frame at 250.
var motionFrameSteps = []struct {
	name string
	at   time.Duration
}{
	{"000", 0},
	{"075", 75 * time.Millisecond},
	{"150", 150 * time.Millisecond},
	{"250", motionLayout},
}

// cloneFrame copies img, as Tester.Image returns the frame the next frame
// overwrites.
func cloneFrame(img *image.RGBA) *image.RGBA {
	out := image.NewRGBA(img.Bounds())
	copy(out.Pix, img.Pix)
	return out
}

// sameFrame reports whether two frames are pixel-identical.
func sameFrame(a, b *image.RGBA) bool { return bytes.Equal(a.Pix, b.Pix) }

// writeMotionFrame writes img under the evidence directory as name, when the
// directory is set. The test names the file for the scenario and the elapsed
// time, so a reader can compare the frames without reading the test.
func writeMotionFrame(t *testing.T, dir, name string, img *image.RGBA) {
	t.Helper()
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create %s: %v", dir, err)
	}
	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatalf("encode %s: %v", path, err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close %s: %v", path, err)
	}
	t.Logf("wrote %s", path)
}

// TestMotionCurvesMatchTheReference pins the two steps and checks the two
// curves: each starts at 0, ends at 1 and rises without going back, so a
// transition can neither jump nor reverse on the way.
func TestMotionCurvesMatchTheReference(t *testing.T) {
	if motionFast != 150*time.Millisecond {
		t.Errorf("the fast step is %v, want the reference's 150 ms", motionFast)
	}
	if motionLayout != 250*time.Millisecond {
		t.Errorf("the layout step is %v, want the reference's 250 ms", motionLayout)
	}
	for _, tc := range []struct {
		name string
		ease ui.Easing
	}{
		{"fast", motionEaseOut},
		{"layout", motionEaseInOut},
	} {
		if got := tc.ease(0); got != 0 {
			t.Errorf("%s: the curve starts at %v, want 0", tc.name, got)
		}
		if got := tc.ease(1); got != 1 {
			t.Errorf("%s: the curve ends at %v, want 1", tc.name, got)
		}
		previous := float32(-1)
		for i := 0; i <= 100; i++ {
			x := float32(i) / 100
			y := tc.ease(x)
			if y < previous-1e-5 {
				t.Errorf("%s: the curve goes back at %.2f: %v after %v", tc.name, x, y, previous)
				break
			}
			if y < -1e-5 || y > 1+1e-5 {
				t.Errorf("%s: the curve leaves 0..1 at %.2f: %v", tc.name, x, y)
				break
			}
			previous = y
		}
	}
}

// TestMotionTransitionsFollowTheSteps checks the transitions the shell hands
// to mygo: the fast step animates colors alone, the layout step animates the
// panel's position, size and colors over the reference's 250 ms, and the
// reduce-motion path closes both with a negative Duration.
func TestMotionTransitionsFollowTheSteps(t *testing.T) {
	var fast, layout, panel ui.ElementTransition
	tt := ui.NewTester(func(c *ui.Context) {
		fast = motionFastColors(c)
		layout = motionLayoutColors(c)
		panel = motionPanel(c)
	}, 20, 20)
	_ = tt

	if fast.Duration != motionFast || !fast.Colors {
		t.Errorf("the fast transition is %+v, want %v colors", fast, motionFast)
	}
	if fast.Position || fast.Size {
		t.Errorf("the fast transition moves the element: %+v", fast)
	}
	if layout.Duration != motionLayout || !layout.Colors {
		t.Errorf("the layout color transition is %+v, want %v colors", layout, motionLayout)
	}
	if !panel.Position || !panel.Size || !panel.Colors || panel.Duration != motionLayout {
		t.Errorf("the panel transition is %+v, want position/size/colors over %v", panel, motionLayout)
	}
	if panel.Enter == nil || panel.Exit == nil {
		t.Errorf("the panel transition has no Enter/Exit: %+v", panel)
	}

	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	if fast.Duration >= 0 || layout.Duration >= 0 {
		t.Errorf("reduce motion leaves the color fades at %v/%v, want negative Durations", fast.Duration, layout.Duration)
	}
	if panel.Duration >= 0 || panel.ExitDuration >= 0 {
		t.Errorf("reduce motion leaves the panel moving at %v/%v, want negative Durations", panel.Duration, panel.ExitDuration)
	}
}

// TestMotionReducedShowsTheHoverAtOnce checks the closed path against the real
// shell: with the desktop asking for less motion, a row's hover face arrives
// on the frame the pointer moves, with no Advance needed. A negative Duration
// is what closes a color fade mygo would otherwise keep under ReduceMotion.
func TestMotionReducedShowsTheHoverAtOnce(t *testing.T) {
	s := fixture()
	var theme *ui.Theme
	tt := ui.NewTesterWith(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 600, ui.TesterOptions{Preferences: ui.Preferences{ReduceMotion: true}})
	k := newTokens(theme)

	second, ok := tt.Find("Second session")
	if !ok {
		t.Fatalf("the second row is not drawn, texts %q", tt.Texts())
	}
	tt.Move(second.X+second.W/2, second.Y+second.H/2)
	tt.Frame()
	face := tt.Image().RGBAAt(int(second.X)+3, int(second.Y+second.H/2))
	if !sameColor(face, k.sidebarHover) {
		t.Errorf("with reduce motion the hover face is %v, want %+v at once", face, k.sidebarHover)
	}
}

// motionFrames advances tt through the reference's layout step and returns
// the frames at 0, 75, 150 and 250 ms, writing each into dir when it is set.
// advance is how far the previous frame is from this one; the first frame is
// captured where it stands.
func motionFrames(t *testing.T, dir, scenario string, tt *ui.Tester) []*image.RGBA {
	t.Helper()
	frames := make([]*image.RGBA, 0, len(motionFrameSteps))
	for i, step := range motionFrameSteps {
		if i > 0 {
			tt.Advance(step.at - motionFrameSteps[i-1].at)
		}
		img := cloneFrame(tt.Image())
		writeMotionFrame(t, dir, "motion-"+scenario+"-"+step.name+".png", img)
		frames = append(frames, img)
	}
	return frames
}

// TestMotionPanelFrames records a panel opening and closing, and checks that
// the frames at 75 and 150 ms are neither the first nor the settled frame: the
// panel is on the way, not already there. The fixture is the real shell's
// context panel beside the conversation, drawn dark so the panel's face and the
// card behind it are different colors.
func TestMotionPanelFrames(t *testing.T) {
	dir := os.Getenv(motionEvidenceEnv)
	s, _ := shellContextPanelState()
	s.Settings = SettingsView{Theme: ThemeDark}
	tt := ui.NewTester(s.View, 900, 600)

	s.ShowContextPanel = true
	tt.Frame()
	open := motionFrames(t, dir, "panel-open", tt)
	for i := 1; i < len(open)-1; i++ {
		if sameFrame(open[i], open[0]) {
			t.Errorf("panel open at %s is identical to the start", motionFrameSteps[i].name)
		}
		if sameFrame(open[i], open[len(open)-1]) {
			t.Errorf("panel open at %s is already the settled frame", motionFrameSteps[i].name)
		}
	}
	if sameFrame(open[0], open[len(open)-1]) {
		t.Error("the panel's start and settled frames are identical")
	}

	s.ShowContextPanel = false
	tt.Frame()
	closed := motionFrames(t, dir, "panel-close", tt)
	for i := 1; i < len(closed)-1; i++ {
		if sameFrame(closed[i], closed[0]) {
			t.Errorf("panel close at %s is identical to the start", motionFrameSteps[i].name)
		}
		if sameFrame(closed[i], closed[len(closed)-1]) {
			t.Errorf("panel close at %s is already the settled frame", motionFrameSteps[i].name)
		}
	}
	if sameFrame(closed[0], closed[len(closed)-1]) {
		t.Error("the panel's close start and settled frames are identical")
	}
}

// TestMotionHoverFrames records a session row under the pointer, and checks
// that the row's face at 75 and 150 ms is neither its resting face nor its
// hover face. It samples the row's own padding, as TestSidebarRowFaces does.
func TestMotionHoverFrames(t *testing.T) {
	dir := os.Getenv(motionEvidenceEnv)
	s := screenshotFixture(false)
	tt := ui.NewTester(s.View, 900, 600)

	row, ok := tt.Find("Review App Server reconnect")
	if !ok {
		t.Fatalf("the second row is not drawn, texts %q", tt.Texts())
	}
	at := func(img *image.RGBA) uint32 {
		p := img.RGBAAt(int(row.X)+3, int(row.Y+row.H/2))
		return uint32(p.R)<<16 | uint32(p.G)<<8 | uint32(p.B)
	}
	rest := at(tt.Image())
	tt.Move(row.X+row.W/2, row.Y+row.H/2)
	tt.Frame()
	frames := motionFrames(t, dir, "hover", tt)

	start, settled := at(frames[0]), at(frames[len(frames)-1])
	if start == settled {
		t.Fatalf("the hover does not change the row's face: %06x", start)
	}
	if rest == settled {
		t.Fatalf("the resting face %06x is the hover face", settled)
	}
	// The fast step is 150 ms: 75 is on the way, and 150 has arrived.
	if mid := at(frames[1]); mid == start || mid == settled {
		t.Errorf("hover at %s is %06x, one of the ends %06x/%06x", motionFrameSteps[1].name, mid, start, settled)
	}
	for i := 2; i < len(frames); i++ {
		if got := at(frames[i]); got != settled {
			t.Errorf("hover at %s is %06x, want the settled %06x after the fast 150 ms step",
				motionFrameSteps[i].name, got, settled)
		}
	}
}

// TestMotionThemeFrames records a theme switch, and checks that a surface's
// color at 75 and 150 ms is neither the old theme's nor the new theme's. It
// samples the main card's face, which the reference transitions over the
// layout step; text colors are not part of the evidence because mygo's
// ElementTransition moves background and border colors alone.
func TestMotionThemeFrames(t *testing.T) {
	dir := os.Getenv(motionEvidenceEnv)
	s := screenshotFixture(false)
	tt := ui.NewTester(s.View, 900, 600)

	card, ok := tt.Find("Main area")
	if !ok {
		t.Fatalf("the main area is not drawn, texts %q", tt.Texts())
	}
	// The card's face shows through below the timeline and the sidebar's
	// face through its own padding: both are the surfaces the theme moves.
	points := [][2]int{
		{int(card.X + card.W/2), int(card.Y + card.H - 10)},
		{3, 300},
	}
	at := func(img *image.RGBA, point [2]int) uint32 {
		p := img.RGBAAt(point[0], point[1])
		return uint32(p.R)<<16 | uint32(p.G)<<8 | uint32(p.B)
	}
	light := make([]uint32, len(points))
	for i, point := range points {
		light[i] = at(tt.Image(), point)
	}

	tt.SetDark(true)
	frames := motionFrames(t, dir, "theme", tt)
	for i, point := range points {
		start, settled := at(frames[0], point), at(frames[len(frames)-1], point)
		if light[i] != start {
			t.Errorf("point %v: the first frame is %06x, want the old theme's %06x", point, start, light[i])
		}
		if start == settled {
			t.Fatalf("point %v: the theme switch does not change the face: %06x", point, start)
		}
		for frame := 1; frame < len(frames)-1; frame++ {
			mid := at(frames[frame], point)
			if mid == start || mid == settled {
				t.Errorf("point %v: theme at %s is %06x, one of the ends %06x/%06x",
					point, motionFrameSteps[frame].name, mid, start, settled)
			}
		}
	}
}
