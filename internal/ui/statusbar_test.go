package ui

import (
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// statusBarTester renders one status bar in a window of width×height DIPs, with
// the shell's light tokens as its theme.
func statusBarTester(v StatusBarView, width, height int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		renderStatusBar(c, newTokens(c.Theme()), v)
	}, width, height)
}

// fullStatusBarView is every field reported: a branch, a half-used context
// window, a model, an account with plan usage, and a ready connection.
func fullStatusBarView() StatusBarView {
	return StatusBarView{
		Branch:     "feat/statusbar",
		Context:    ContextUsage{Used: 40, Limit: 100, Reported: true},
		Model:      "claude-sonnet-4-5",
		Account:    "eric",
		Plan:       PlanUsage{Percent: 42, Reported: true},
		Connection: ConnReady,
	}
}

// TestStatusBarShowsEveryField checks the reading the bar exists for: the
// branch on the left, the context usage in the middle, and the model, account
// and connection on the right, at the spec's 35-DIP height.
func TestStatusBarShowsEveryField(t *testing.T) {
	v := fullStatusBarView()
	tt := statusBarTester(v, 900, 600)

	for _, want := range []string{
		"Branch: feat/statusbar",
		"Context: 40 / 100 (40%)",
		"Model: claude-sonnet-4-5",
		"Account: eric · 42% used",
		"App Server: Connected",
	} {
		if !tt.HasText(want) {
			t.Errorf("status bar is missing %q, texts %q", want, tt.Texts())
		}
	}

	if _, ok := tt.AccessByRole(ui.RoleStatus, "App Server: Connected"); !ok {
		t.Errorf("the connection is not a live status region, texts %q", tt.Texts())
	}

	bar, ok := tt.AccessByRole(ui.RoleGroup, "Status bar")
	if !ok {
		t.Fatalf("the bar is not a named group, texts %q", tt.Texts())
	}
	if diff := math.Abs(float64(bar.Bounds.H - statusBarH)); diff > 0.5 {
		t.Errorf("bar is %.1f high, want %.1f from docs/ui-layout-spec.md §7", bar.Bounds.H, float64(statusBarH))
	}
	if diff := math.Abs(float64(bar.Bounds.W - 900)); diff > 1 {
		t.Errorf("bar is %.1f wide in a 900-wide window", bar.Bounds.W)
	}

	// The parts keep the spec's order across the bar: branch, meter, then the
	// right-hand readings, each after the one before it.
	branch, ok := tt.Find("Branch: feat/statusbar")
	if !ok {
		t.Fatalf("branch reading is not drawn, texts %q", tt.Texts())
	}
	meter, ok := tt.AccessByRole(ui.RoleProgress, "Context usage")
	if !ok {
		t.Fatalf("context meter is not drawn, texts %q", tt.Texts())
	}
	model, ok := tt.Find("Model: claude-sonnet-4-5")
	if !ok {
		t.Fatalf("model reading is not drawn, texts %q", tt.Texts())
	}
	if !(branch.X+branch.W <= meter.Bounds.X && meter.Bounds.X+meter.Bounds.W <= model.X) {
		t.Errorf("parts are out of order: branch %+v, meter %+v, model %+v", branch, meter.Bounds, model)
	}
}

// TestStatusBarFormatsTokenCounts checks the reading's numbers against
// dim-electron's formatter, so a large window is not drawn as raw digits.
func TestStatusBarFormatsTokenCounts(t *testing.T) {
	cases := []struct {
		tokens int64
		want   string
	}{
		{0, "0"},
		{999, "999"},
		{1_000, "1.0K"},
		{12_000, "12.0K"},
		{24_500, "24.5K"},
		{1_000_000, "1.0M"},
		{1_500_000, "1.5M"},
	}
	for _, tc := range cases {
		if got := formatTokenCount(tc.tokens); got != tc.want {
			t.Errorf("formatTokenCount(%d) = %q, want %q", tc.tokens, got, tc.want)
		}
	}

	v := StatusBarView{
		Width:   900,
		Context: ContextUsage{Used: 24_500, Limit: 128_000, Reported: true},
	}
	tt := statusBarTester(v, 900, 600)
	if !tt.HasText("Context: 24.5K / 128.0K (19%)") {
		t.Errorf("token counts are not formatted as the reading, texts %q", tt.Texts())
	}
}

// TestStatusBarZeroUsageIsAValue checks that a reported zero is drawn as 0% and
// a zero-length fill, not as WIP: no tokens used is a reading, while no report
// is a gap.
func TestStatusBarZeroUsageIsAValue(t *testing.T) {
	v := StatusBarView{
		Width:   900,
		Context: ContextUsage{Used: 0, Limit: 100, Reported: true},
	}
	tt := statusBarTester(v, 900, 600)

	if !tt.HasText("Context: 0 / 100 (0%)") {
		t.Errorf("reported zero usage is missing, texts %q", tt.Texts())
	}
	meter, ok := tt.AccessByRole(ui.RoleProgress, "Context usage")
	if !ok {
		t.Fatalf("reported zero usage draws no meter, texts %q", tt.Texts())
	}
	if meter.Now != 0 {
		t.Errorf("meter reads %.2f for zero usage", meter.Now)
	}
}

// TestStatusBarShowsVisibleWIPWithoutData checks the bar's empty state: every
// item whose data the window has not passed shows a visible WIP badge and
// explains itself, and nothing stands in a branch, percentage or model.
func TestStatusBarShowsVisibleWIPWithoutData(t *testing.T) {
	tt := statusBarTester(StatusBarView{Width: 900}, 900, 600)

	for _, want := range []string{
		"Branch: WIP",
		"Context: WIP",
		"Model: WIP",
		"Account: WIP",
		"App Server: Not connected",
	} {
		if !tt.HasText(want) {
			t.Errorf("status bar is missing %q, texts %q", want, tt.Texts())
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleProgress, "Context usage"); ok {
		t.Error("a context meter is drawn with no usage reading")
	}
	for _, fabricated := range []string{"0%", "0 / 0", "Plan:"} {
		if tt.HasText(fabricated) {
			t.Errorf("status bar fabricates %q with no data, texts %q", fabricated, tt.Texts())
		}
	}

	// The WIP badge carries the missing fact as its description, so the gap is
	// read out with the item and not only seen.
	info, ok := tt.AccessByRole(ui.RoleText, "Context: WIP")
	if !ok {
		t.Fatalf("the context item is not a text reading, texts %q", tt.Texts())
	}
	if info.Description != reasonStatusContext {
		t.Errorf("context WIP describes %q, want %q", info.Description, reasonStatusContext)
	}
}

// TestStatusBarKnownUsageWithoutLimitHasNoBar checks partial data: a reported
// used count with no known window shows the count, marks only the limit WIP,
// and draws no meter, because a bar needs both numbers.
func TestStatusBarKnownUsageWithoutLimitHasNoBar(t *testing.T) {
	v := StatusBarView{
		Width:   900,
		Context: ContextUsage{Used: 12_000, Reported: true},
	}
	tt := statusBarTester(v, 900, 600)

	if !tt.HasText("Context: 12.0K tokens · limit WIP") {
		t.Errorf("used tokens without a limit are not shown, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleProgress, "Context usage"); ok {
		t.Error("a meter is drawn without a known context window")
	}
}

// TestStatusBarMeterDrawsTheKnownShare checks the meter's geometry: the fill is
// the reported share of the 35-DIP track, in the success color below half, and
// the rest of the track keeps the muted face.
func TestStatusBarMeterDrawsTheKnownShare(t *testing.T) {
	const used, limit = 40, 100
	v := StatusBarView{
		Width:   900,
		Context: ContextUsage{Used: used, Limit: limit, Reported: true},
	}
	tt := statusBarTester(v, 900, 600)

	meter, ok := tt.AccessByRole(ui.RoleProgress, "Context usage")
	if !ok {
		t.Fatalf("context meter is not drawn, texts %q", tt.Texts())
	}
	if diff := math.Abs(float64(meter.Bounds.W - statusBarMeterW)); diff > 0.5 {
		t.Errorf("meter is %.1f wide, want %.1f", meter.Bounds.W, float64(statusBarMeterW))
	}
	want := float64(used) / float64(limit)
	if diff := math.Abs(meter.Now - want); diff > 0.001 {
		t.Errorf("meter reads %.4f, want %.4f", meter.Now, want)
	}

	k := newTokens(ui.LightTheme())
	img := tt.Image()
	x, y := int(meter.Bounds.X), int(meter.Bounds.Y+1)
	fillEnd := x + int(float32(want)*statusBarMeterW)
	if got := img.RGBAAt(x+2, y); got != (color.RGBA{R: k.success.R, G: k.success.G, B: k.success.B, A: k.success.A}) {
		t.Errorf("meter fill starts with %+v, want the success color %+v", got, k.success)
	}
	if got := img.RGBAAt(fillEnd+3, y); got != (color.RGBA{R: k.muted.R, G: k.muted.G, B: k.muted.B, A: k.muted.A}) {
		t.Errorf("meter track after the fill is %+v, want the muted face %+v", got, k.muted)
	}
}

// TestContextUsageColorFollowsTheThresholds checks the meter's fill color at
// dim-electron's three bands, so a full context window reads as danger rather
// than as the same success tint as an empty one.
func TestContextUsageColorFollowsTheThresholds(t *testing.T) {
	k := newTokens(ui.LightTheme())
	cases := []struct {
		fraction float64
		want     ui.Color
	}{
		{0, k.success},
		{0.49, k.success},
		{0.5, k.warning},
		{0.79, k.warning},
		{0.8, k.danger},
		{1, k.danger},
	}
	for _, tc := range cases {
		if got := contextUsageColor(k, tc.fraction); got != tc.want {
			t.Errorf("contextUsageColor(%.2f) = %+v, want %+v", tc.fraction, got, tc.want)
		}
	}
}

// TestStatusBarEstimatedUsageIsMarked checks that an estimate is drawn with
// dim-electron's "~" marker instead of reading as an exact measurement.
func TestStatusBarEstimatedUsageIsMarked(t *testing.T) {
	v := StatusBarView{
		Width:   900,
		Context: ContextUsage{Used: 40, Limit: 100, Reported: true, Estimated: true},
	}
	tt := statusBarTester(v, 900, 600)

	if !tt.HasText("Context: ~40 / 100 (~40%)") {
		t.Errorf("estimated usage is not marked, texts %q", tt.Texts())
	}
}

// TestStatusBarCompactOmitsTheMeter checks the first narrow step: below the
// compact width the middle item leaves, while the branch and the right-hand
// readings keep their places.
func TestStatusBarCompactOmitsTheMeter(t *testing.T) {
	v := fullStatusBarView()
	v.Branch, v.Model, v.Account = "main", "sonnet", "eric"
	v.Width = statusBarCompactW - 1
	tt := statusBarTester(v, int(v.Width), 600)

	if tt.HasText("Context") {
		t.Errorf("context meter survives in a compact bar, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleProgress, "Context usage"); ok {
		t.Error("a context meter is drawn below the compact width")
	}
	for _, want := range []string{"Branch: main", "Model: sonnet", "Account: eric · 42% used", "App Server: Connected"} {
		if !tt.HasText(want) {
			t.Errorf("compact bar is missing %q, texts %q", want, tt.Texts())
		}
	}
}

// TestStatusBarNarrowKeepsTheEndsApart checks the narrowest step: only the
// branch and the connection remain, both inside the window and not overlapping,
// so a small window loses the middle readings rather than drawing them over
// each other.
func TestStatusBarNarrowKeepsTheEndsApart(t *testing.T) {
	v := fullStatusBarView()
	v.Width = statusBarNarrowW - 1
	tt := statusBarTester(v, int(v.Width), 600)

	for _, omitted := range []string{"Context", "Model", "Account"} {
		if tt.HasText(omitted) {
			t.Errorf("%q survives in a narrow bar, texts %q", omitted, tt.Texts())
		}
	}
	branch, ok := tt.Find("Branch: feat/statusbar")
	if !ok {
		t.Fatalf("narrow bar drops the branch, texts %q", tt.Texts())
	}
	conn, ok := tt.Find("App Server: Connected")
	if !ok {
		t.Fatalf("narrow bar drops the connection, texts %q", tt.Texts())
	}
	if branch.X+branch.W > conn.X {
		t.Errorf("branch %+v and connection %+v overlap or are out of order", branch, conn)
	}
	for _, r := range []ui.Rect{branch, conn} {
		if r.X < 0 || r.X+r.W > v.Width {
			t.Errorf("%+v is outside a %.0f-wide bar", r, v.Width)
		}
	}
}

// TestShellDrawsNoGlobalStatusBar checks the reference's bottom edge: the shell
// no longer paints the 35-DIP bar, and its branch, context, model, account and
// connection readings are absent from the frame. The statusbar renderer remains
// available for the composer's context row work, which is owned elsewhere.
func TestShellDrawsNoGlobalStatusBar(t *testing.T) {
	s := fixture()
	s.SelectedBranch = "feat/chrome-parity"
	tt := ui.NewTester(s.View, 900, 600)

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Status bar"); ok {
		t.Errorf("the shell still draws the global status bar, texts %q", tt.Texts())
	}
	for _, gone := range []string{
		"Branch: feat/chrome-parity", "Branch: WIP", "Context: WIP",
		"Model: WIP", "Account: WIP", "App Server: Not connected",
	} {
		if tt.HasText(gone) {
			t.Errorf("the shell still draws the status reading %q, texts %q", gone, tt.Texts())
		}
	}
}
