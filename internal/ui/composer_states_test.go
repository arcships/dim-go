package ui

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// composerStateView builds a composer in isolation with reduced motion, so a
// state test reads the face a control has now rather than a color fade's
// midpoint: a hover face arrives on the frame the pointer moves. The harness
// owns the draft pointer and the focus handle, as the shell does.
func composerStateView(t *testing.T, sending bool, draft string, onSubmit func(string)) (ComposerView, *ui.Tester) {
	t.Helper()
	view := ComposerView{
		Draft:       new(string),
		CanSubmit:   true,
		Sending:     sending,
		Placeholder: composerPlaceholder,
		Shortcut:    KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
		Focus:       new(ui.Handle),
	}
	*view.Draft = draft
	a := Actions{OnSubmit: onSubmit}
	tt := ui.NewTesterWith(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, a)
	}, 720, 480, ui.TesterOptions{Preferences: ui.Preferences{ReduceMotion: true}})
	return view, tt
}

// composerSendFace reads the Send disc's face a few DIPs above its centre: the
// arrow's ink and the rim's antialiasing are out of the way, so the pixel is
// the background the button was given.
func composerSendFace(tt *ui.Tester, send ui.Rect) (uint8, uint8, uint8, uint8) {
	p := tt.Image().RGBAAt(int(send.X+send.W/2), int(send.Y+3))
	return p.R, p.G, p.B, p.A
}

// TestComposerModelChip checks the model status beside Send: it is a muted
// chip, it says the model is not configured rather than naming one, it sits
// left of the send disc on the same row at the reference's distance, and the
// reason the chip waits on is what assistive technology reads.
func TestComposerModelChip(t *testing.T) {
	for _, wide := range []bool{false, true} {
		name := "empty"
		if wide {
			name = "conversation"
		}
		t.Run(name, func(t *testing.T) {
			view := ComposerView{
				Draft:       new(string),
				CanSubmit:   true,
				Placeholder: composerPlaceholder,
				Wide:        wide,
				Shortcut:    KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
				Focus:       new(ui.Handle),
			}
			tt := ui.NewTesterWith(func(c *ui.Context) {
				composer(c, newTokens(c.Theme()), view, Actions{})
			}, 720, 480, ui.TesterOptions{Preferences: ui.Preferences{ReduceMotion: true}})
			k := newTokens(ui.LightTheme())

			label := textBox(t, tt, tr(LocaleEN, keyComposerUnconfigured))
			send, ok := tt.Find(tr(LocaleEN, keyComposerSend))
			if !ok {
				t.Fatalf("Send is not drawn, texts %q", tt.Texts())
			}
			gap := send.X - (label.X + label.W)
			if gap < 0 {
				t.Errorf("the model chip overlaps Send: label ends at %.1f, Send starts at %.1f",
					label.X+label.W, send.X)
			}
			if gap > 8 {
				t.Errorf("the model chip is %.1f DIPs from Send, want the reference's few DIPs", gap)
			}
			if labelCenter, sendCenter := label.Y+label.H/2, send.Y+send.H/2; math.Abs(float64(labelCenter-sendCenter)) > 1 {
				t.Errorf("the model chip's centre is at y=%.1f, not on the Send row at y=%.1f",
					labelCenter, sendCenter)
			}

			chip, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyComposerModelChip))
			if !ok {
				t.Fatalf("the model chip is not a named button, texts %q", tt.Texts())
			}
			if chip.Description != reasonModelPair {
				t.Errorf("the model chip's description is %q, want the gap %q", chip.Description, reasonModelPair)
			}
			if chip.Bounds.X > label.X || chip.Bounds.X+chip.Bounds.W < label.X+label.W {
				t.Errorf("the chip's box %.1fx%.1f does not hold its label at x=%.1f",
					chip.Bounds.W, chip.Bounds.H, label.X)
			}
			if _, face := colorBox(tt, chip.Bounds, k.muted); float32(face) < chip.Bounds.W*chip.Bounds.H/3 {
				t.Errorf("the chip paints %d muted pixels in its %.0fx%.0f box, want the muted pill",
					face, chip.Bounds.W, chip.Bounds.H)
			}
		})
	}
}

// TestComposerSendStates checks the Send disc's three faces: the reference's
// solid primary while a draft cannot be sent, the same disc with a slight
// hover step while one can, and the spinner while a submission is in flight.
// The hover face belongs to the enabled disc alone: a disabled Send must not
// answer the pointer, or it would claim a submission the window would ignore.
func TestComposerSendStates(t *testing.T) {
	k := newTokens(ui.LightTheme())
	hover := composerSendHover(k)

	// A draft that cannot be sent stays the reference's solid disc under the
	// pointer.
	_, tt := composerStateView(t, false, "", func(string) {})
	send, ok := tt.Find(tr(LocaleEN, keyComposerSend))
	if !ok {
		t.Fatalf("Send is not drawn, texts %q", tt.Texts())
	}
	tt.Move(send.X+send.W/2, send.Y+send.H/2)
	tt.Frame()
	if r, g, b, a := composerSendFace(tt, send); r != k.primary.R || g != k.primary.G || b != k.primary.B || a != k.primary.A {
		t.Errorf("a disabled Send under the pointer is %v, want the solid primary", []uint8{r, g, b, a})
	}

	// A draft that can be sent keeps the solid disc until the pointer reaches
	// it, then takes the slight hover step, and goes back when the pointer
	// leaves.
	_, tt = composerStateView(t, false, "hello", func(string) {})
	send, ok = tt.Find(tr(LocaleEN, keyComposerSend))
	if !ok {
		t.Fatalf("Send is not drawn, texts %q", tt.Texts())
	}
	if r, g, b, a := composerSendFace(tt, send); r != k.primary.R || g != k.primary.G || b != k.primary.B || a != k.primary.A {
		t.Errorf("a ready Send away from the pointer is %v, want the solid primary", []uint8{r, g, b, a})
	}
	tt.Move(send.X+send.W/2, send.Y+send.H/2)
	tt.Frame()
	if r, g, b, a := composerSendFace(tt, send); r != hover.R || g != hover.G || b != hover.B || a != hover.A {
		t.Errorf("a ready Send under the pointer is %v, want the hover face %v",
			[]uint8{r, g, b, a}, []uint8{hover.R, hover.G, hover.B, hover.A})
	}
	tt.Move(4, 4)
	tt.Frame()
	if r, g, b, a := composerSendFace(tt, send); r != k.primary.R || g != k.primary.G || b != k.primary.B || a != k.primary.A {
		t.Errorf("Send keeps %v after the pointer left, want the solid primary", []uint8{r, g, b, a})
	}

	// A ready Send with only the arrow drawn repaints nothing: the static disc
	// is the control for the in-flight spinner's tick below.
	before := sendDisc(tt, send)
	tt.Advance(150 * time.Millisecond)
	tt.Frame()
	if after := sendDisc(tt, send); after != before {
		t.Error("a ready Send repaints its disc with no spinner to show")
	}

	if err := tt.Click(tr(LocaleEN, keyComposerSend)); err != nil {
		t.Fatal(err)
	}
}

// sendDisc reads the pixels of the Send disc as one string, so a test can tell
// whether the disc was painted again with a spinner.
func sendDisc(tt *ui.Tester, send ui.Rect) string {
	img := tt.Image()
	var out []byte
	for y := int(send.Y); y < int(send.Y+send.H); y++ {
		for x := int(send.X); x < int(send.X+send.W); x++ {
			p := img.RGBAAt(x, y)
			out = append(out, p.R, p.G, p.B, p.A)
		}
	}
	return string(out)
}

// TestComposerSendSpinnerWhileSending checks the in-flight face: the disc
// swaps its arrow for a decorative spinner, keeps the disabled solid disc,
// says it is busy to assistive technology, and takes no click while the window
// says a submission is in flight.
func TestComposerSendSpinnerWhileSending(t *testing.T) {
	calls := 0
	_, tt := composerStateView(t, true, "hello", func(string) { calls++ })
	k := newTokens(ui.LightTheme())

	send, ok := tt.Find(tr(LocaleEN, keyComposerSend))
	if !ok {
		t.Fatalf("Send is not drawn, texts %q", tt.Texts())
	}
	info, ok := tt.AccessByRole(ui.RoleButton, tr(LocaleEN, keyComposerSend))
	if !ok {
		t.Fatalf("Send is not exposed as a button, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("Send is available while a submission is in flight")
	}
	if !info.Busy {
		t.Error("the in-flight Send is not marked busy for assistive technology")
	}
	if info.Description != tr(LocaleEN, composerSendBusyKey) {
		t.Errorf("the in-flight Send's description is %q, want the in-flight reason %q",
			info.Description, tr(LocaleEN, composerSendBusyKey))
	}
	// The spinner is a decoration, so the frame proves it: the disc ticks
	// between two moments while a submission is in flight, where a ready Send
	// with only the arrow drawn stays still.
	before := sendDisc(tt, send)
	tt.Advance(150 * time.Millisecond)
	tt.Frame()
	if after := sendDisc(tt, send); after == before {
		t.Error("the in-flight Send draws no spinner: its disc did not change between frames")
	}
	if r, g, b, a := composerSendFace(tt, send); r != k.primary.R || g != k.primary.G || b != k.primary.B || a != k.primary.A {
		t.Errorf("the in-flight Send is %v, want the solid primary disc", []uint8{r, g, b, a})
	}
	if err := tt.Click(tr(LocaleEN, keyComposerSend)); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("Send submitted %d times while a submission is in flight", calls)
	}
}
