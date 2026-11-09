package ui

import (
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// composerParityView builds the composer in isolation at one width state, with
// a live draft and a submission gate, so a test can measure the card the way
// the window draws it.
func composerParityView(t *testing.T, wide bool) (ComposerView, *ui.Tester) {
	t.Helper()
	view := ComposerView{
		Draft:       new(string),
		CanSubmit:   true,
		Placeholder: composerPlaceholder,
		Wide:        wide,
		Shortcut:    KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
		Focus:       new(ui.Handle),
	}
	tt := ui.NewTester(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, Actions{OnSubmit: func(string) {}})
	}, 800, 600)
	return view, tt
}

// composerCardBounds returns the card's box: the composer root is the card's
// group, so it carries the whole surface rather than only the editor inside it.
func composerCardBounds(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleGroup, "")
	if !ok {
		t.Fatalf("the card is not exposed as a group, texts %q", tt.Texts())
	}
	return info.Bounds
}

// TestComposerCardMeasures checks the reference's two card measures: the
// workspace's empty state keeps the narrower 461 DIP card, a conversation the
// wider 520 DIP one, and both stay centred in the column rather than running
// edge to edge.
func TestComposerCardMeasures(t *testing.T) {
	const windowW = 800
	for _, tc := range []struct {
		name string
		wide bool
		want float32
	}{
		{"empty", false, composerEmptyWidth},
		{"conversation", true, composerWideWidth},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, tt := composerParityView(t, tc.wide)
			card := composerCardBounds(t, tt)
			if math.Abs(float64(card.W-tc.want)) > 0.5 {
				t.Errorf("the card is %.1f wide, want the reference's %.0f", card.W, tc.want)
			}
			if card.W >= windowW {
				t.Errorf("the card is %.1f wide in an %d wide window, so it still fills the column", card.W, windowW)
			}
			if center := card.X + card.W/2; math.Abs(float64(center-windowW/2)) > 0.5 {
				t.Errorf("the card's center is %.1f, want the column's %.1f", center, float64(windowW)/2)
			}
			// The editor keeps the card's inset; the card is a frame around it,
			// not another full-width row the editor sits on top of.
			k := newTokens(ui.LightTheme())
			editor, ok := tt.Find("Composer")
			if !ok {
				t.Fatalf("the editor is not drawn, texts %q", tt.Texts())
			}
			if math.Abs(float64(editor.X-(card.X+k.edgePad+k.hairline))) > 0.5 {
				t.Errorf("the editor starts at x=%.1f, want the card's inset at %.1f", editor.X, card.X+k.edgePad+k.hairline)
			}
			if math.Abs(float64(editor.W-(card.W-2*(k.edgePad+k.hairline)))) > 0.5 {
				t.Errorf("the editor is %.1f wide, want the card's inner %.1f", editor.W, card.W-2*(k.edgePad+k.hairline))
			}
		})
	}
}

// TestComposerToolRowIsIcons checks the tool row the reference draws: the
// attachment, mode and model entries are icon-sized buttons on the left, Send
// is the only entry on the right, and the three keep their accessible names and
// the reason each waits on instead of a visible text button or WIP badge.
func TestComposerToolRowIsIcons(t *testing.T) {
	k := newTokens(ui.LightTheme())
	_, tt := composerParityView(t, true)

	size := k.space(composerToolSpan)
	for _, tc := range []struct {
		label, reason string
		focusable     bool
	}{
		{"Attach file", reasonAttachment, true},
		{"Mode", reasonMode, false},
		{"Model", reasonModel, false},
	} {
		info, ok := tt.AccessByRole(ui.RoleButton, tc.label)
		if !ok {
			t.Fatalf("%s is not exposed as a button, texts %q", tc.label, tt.Texts())
		}
		if !strings.Contains(info.Description, tc.reason) {
			t.Errorf("%s description %q does not explain the gap %q", tc.label, info.Description, tc.reason)
		}
		if info.Focusable != tc.focusable {
			t.Errorf("%s focusable=%v, want %v so the documented Tab order holds", tc.label, info.Focusable, tc.focusable)
		}
		rect, ok := tt.Find(tc.label)
		if !ok {
			t.Fatalf("%s has no drawn box, texts %q", tc.label, tt.Texts())
		}
		if math.Abs(float64(rect.W-size)) > 0.5 || math.Abs(float64(rect.H-size)) > 0.5 {
			t.Errorf("%s is %.1fx%.1f, want the reference's %.0f DIP icon button", tc.label, rect.W, rect.H, size)
		}
	}

	// The toolbar carries no visible WIP text: the context row keeps only its
	// labels, and the reasons live in the entries' descriptions, as the
	// reference's status band shows no badge.
	for _, text := range tt.Texts() {
		if text == wipBadge {
			t.Errorf("the card shows a visible %q badge; the gaps belong in the entries' descriptions", wipBadge)
		}
	}

	attach, _ := tt.Find("Attach file")
	mode, _ := tt.Find("Mode")
	model, _ := tt.Find("Model")
	send, _ := tt.Find("Send")
	if send.X <= model.X+model.W {
		t.Errorf("Send sits at x=%.1f, not right of the model icon ending at %.1f", send.X, model.X+model.W)
	}
	if attach.Y != mode.Y || mode.Y != model.Y {
		t.Errorf("the icons do not share one row: attach y=%.1f, mode y=%.1f, model y=%.1f",
			attach.Y, mode.Y, model.Y)
	}
}

// TestComposerCarriesShortcutHelp checks that the shortcut pair lives on the
// editor rather than in a visible hint row: the reference draws no line under
// the field, and assistive technology still reads both keys.
func TestComposerCarriesShortcutHelp(t *testing.T) {
	_, tt := composerParityView(t, false)

	if tt.HasText(composerHint) {
		t.Errorf("the shortcut hint is still drawn as a row, texts %q", tt.Texts())
	}
	info, ok := tt.Access("Composer")
	if !ok {
		t.Fatalf("the editor is not exposed, texts %q", tt.Texts())
	}
	if info.Description != composerHint {
		t.Errorf("the editor description is %q, want the shortcut help %q", info.Description, composerHint)
	}
	for _, key := range []string{"↩", "⌘↩"} {
		if !strings.Contains(info.KeyShortcuts, key) {
			t.Errorf("the editor key shortcuts %q do not carry %q", info.KeyShortcuts, key)
		}
	}
}

// TestComposerContextRowIsInsideCard checks that the workspace and permission
// entries are the card's bottom band: they sit below the tool row inside the
// card's edges, and the card ends with them rather than with a second surface.
func TestComposerContextRowIsInsideCard(t *testing.T) {
	k := newTokens(ui.LightTheme())
	_, tt := composerParityView(t, false)
	card := composerCardBounds(t, tt)

	tool, ok := tt.Find("Attach file")
	if !ok {
		t.Fatalf("the tool row is not drawn, texts %q", tt.Texts())
	}
	workspace, ok := tt.Find("Workspace")
	if !ok {
		t.Fatalf("the workspace entry is not drawn, texts %q", tt.Texts())
	}
	permission, ok := tt.Find("Default permissions")
	if !ok {
		t.Fatalf("the permission entry is not drawn, texts %q", tt.Texts())
	}
	for _, entry := range []struct {
		label string
		rect  ui.Rect
	}{{"Workspace", workspace}, {"Default permissions", permission}} {
		if entry.rect.Y < tool.Y+tool.H {
			t.Errorf("%s sits at y=%.1f, above the tool row ending at %.1f", entry.label, entry.rect.Y, tool.Y+tool.H)
		}
		if entry.rect.X < card.X+k.edgePad {
			t.Errorf("%s starts at x=%.1f, left of the card's inset at %.1f", entry.label, entry.rect.X, card.X+k.edgePad)
		}
		if right := entry.rect.X + entry.rect.W; right > card.X+card.W-k.edgePad {
			t.Errorf("%s ends at x=%.1f, past the card's inset at %.1f", entry.label, right, card.X+card.W-k.edgePad)
		}
		if bottom := entry.rect.Y + entry.rect.H; bottom >= card.Y+card.H {
			t.Errorf("%s ends at y=%.1f, at or past the card's bottom %.1f", entry.label, bottom, card.Y+card.H)
		}
	}

	// The entries are centred in the card's last band, so what is left under
	// them is that band's own padding and the border, not a second card.
	bottomGap := card.Y + card.H - (workspace.Y + workspace.H)
	if bottomGap < k.gap || bottomGap > k.edgePad+k.hairline {
		t.Errorf("the card leaves %.1f DIP under the context row, want the band's own %.1f–%.1f",
			bottomGap, k.gap, k.edgePad+k.hairline)
	}
	if card.H < 145 || card.H > 165 {
		t.Errorf("the card is %.1f high, want the reference's ≈152 with the context band inside", card.H)
	}
}

// glyphInkBox returns the tight box of the pixels inside area that are not the
// background, and how many there are: the drawn strokes of a glyph, without
// asking the view how it drew them.
func glyphInkBox(tt *ui.Tester, area ui.Rect, background ui.Color) (ui.Rect, int) {
	img := tt.Image()
	bg := color.RGBA{R: background.R, G: background.G, B: background.B, A: background.A}
	x0, y0 := max(int(area.X), 0), max(int(area.Y), 0)
	x1, y1 := min(int(area.X+area.W), img.Bounds().Dx()), min(int(area.Y+area.H), img.Bounds().Dy())
	minX, minY, maxX, maxY := math.MaxInt, math.MaxInt, -1, -1
	count := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if img.RGBAAt(x, y) == bg {
				continue
			}
			count++
			minX, minY = min(minX, x), min(minY, y)
			maxX, maxY = max(maxX, x), max(maxY, y)
		}
	}
	if count == 0 {
		return ui.Rect{}, 0
	}
	return ui.Rect{
		X: float32(minX), Y: float32(minY),
		W: float32(maxX - minX + 1), H: float32(maxY - minY + 1),
	}, count
}

// TestComposerModeGlyphIsADoubleChevron checks the Mode entry's mark: the
// reference draws the agent-mode picker as » — a pair of chevrons — rather than
// the single forward chevron the sidebar uses. The pair paints a wider glyph at
// the same 16 DIP step, so the drawn ink tells the two apart without the test
// reading the SVG.
func TestComposerModeGlyphIsADoubleChevron(t *testing.T) {
	k := newTokens(ui.LightTheme())
	_, tt := composerParityView(t, true)
	mode, ok := tt.Find("Mode")
	if !ok {
		t.Fatalf("the mode entry is not drawn, texts %q", tt.Texts())
	}
	double, doubleInk := glyphInkBox(tt, mode, k.background)

	singleTT := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Size(40, 40).Background(k.background).Center().Children(func() {
			composerGlyph(c, k, sidebarIcons.chevron)
		})
	}, 40, 40)
	single, singleInk := glyphInkBox(singleTT, ui.Rect{X: 0, Y: 0, W: 40, H: 40}, k.background)
	if doubleInk == 0 || singleInk == 0 {
		t.Fatalf("the marks paint no ink: mode %d pixels, single chevron %d pixels", doubleInk, singleInk)
	}
	if double.W <= single.W*1.4 {
		t.Errorf("the mode mark's ink is %.1f wide, want the » pair wider than the single chevron's %.1f",
			double.W, single.W)
	}
	if math.Abs(float64(double.H-single.H)) > 2 {
		t.Errorf("the mode mark is %.1f high, want the single chevron's %.1f", double.H, single.H)
	}
}
