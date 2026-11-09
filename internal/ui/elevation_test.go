package ui

import (
	"bytes"
	"image"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

// flatShadows returns the frame's tokens with every shadow step removed, so a
// test can draw the same surface lifted and flat: the two frames must have the
// same layout and the same face, and may differ only outside the surface,
// where the shadow falls. A view that drew a shadow of its own would keep the
// frames apart.
func flatShadows(k tokens) tokens {
	k.shadowCard = shadow{}
	k.shadowPopover = shadow{}
	k.shadowDialog = shadow{}
	return k
}

// shadowProbe is one surface the shadow steps reach: the probe draws it
// over the window, and surface returns the box the frame gave it, so a test can
// check the box did not move and sample the pixel just beyond one of its edges.
type shadowProbe struct {
	name          string
	width, height int
	draw          func(c *ui.Context, k tokens)
	// surface returns the drawn surface's box, from the frame the probe drew.
	surface func(t *testing.T, tt *ui.Tester) ui.Rect
	// side is the edge the shadow is sampled beyond: "bottom" or "start".
	side string
	// label is a text the probe draws; its box must be the same lifted and
	// flat. An empty label skips that check.
	label string
	// settle runs after the first frame, for a surface that enters over a
	// motion step.
	settle func(tt *ui.Tester)
}

// TestShadowStepsReachTheSurfaces draws every surface that lifts by a shadow
// step twice — once with the frame's tokens and once with the steps
// removed — and checks the shadow itself: the surface's box and text stay
// where they were, and the pixel just beyond its edge is darker when the step
// is there. Shadows paint outside their element and take no room, so the two
// frames must have the same layout.
func TestShadowStepsReachTheSurfaces(t *testing.T) {
	for _, theme := range themes() {
		for _, probe := range shadowProbes() {
			t.Run(theme.name+"/"+probe.name, func(t *testing.T) {
				lifted := showProbe(probe, theme.theme.Dark, false)
				flat := showProbe(probe, theme.theme.Dark, true)

				liftedBox := probe.surface(t, lifted)
				flatBox := probe.surface(t, flat)
				if liftedBox != flatBox {
					t.Errorf("the surface moved with its shadow: %+v lifted, %+v flat", liftedBox, flatBox)
				}
				if probe.label != "" {
					liftedText, ok := lifted.Find(probe.label)
					if !ok {
						t.Fatalf("the probe does not draw %q, texts %q", probe.label, lifted.Texts())
					}
					flatText, ok := flat.Find(probe.label)
					if !ok {
						t.Fatalf("the flat probe does not draw %q, texts %q", probe.label, flat.Texts())
					}
					if liftedText != flatText {
						t.Errorf("%q moved with the shadow: %+v lifted, %+v flat", probe.label, liftedText, flatText)
					}
				}

				x, y := shadowSample(liftedBox, probe.side)
				if !imageHasPoint(lifted.Image(), x, y) {
					t.Fatalf("the sample (%d, %d) is outside the %dx%d frame", x, y, probe.width, probe.height)
				}
				liftedInk := rgbSum(lifted.Image(), x, y)
				flatInk := rgbSum(flat.Image(), x, y)
				if liftedInk >= flatInk {
					t.Errorf("the pixel just beyond the surface's %s edge is %d lifted, %d flat: no shadow is drawn",
						probe.side, liftedInk, flatInk)
				}
			})
		}
	}
}

// TestToolCardKeepsItsHairlineWithoutAShadow checks the one surface that must
// not lift: a tool card is a block of the turn, not a surface of its own. Its
// frame is byte-identical with and without the shadow steps, so the card
// keeps only its hairline edge.
func TestToolCardKeepsItsHairlineWithoutAShadow(t *testing.T) {
	msg := Message{
		ID:     "t1",
		Author: AuthorTool,
		Tool:   Tool{Name: "bash", Params: "go test ./...", Result: "ok", Status: ToolComplete},
	}
	draw := func(c *ui.Context, k tokens) {
		ui.Column(c).Fill().Background(k.background).Children(func() {
			toolRow(c, k, &blockFolds{}, msg)
		})
	}
	probe := shadowProbe{name: "tool card", width: 600, height: 300, draw: draw}

	for _, theme := range themes() {
		lifted := showProbe(probe, theme.theme.Dark, false)
		flat := showProbe(probe, theme.theme.Dark, true)
		if !lifted.HasText("bash") {
			t.Fatalf("%s: the tool card is not drawn, texts %q", theme.name, lifted.Texts())
		}
		if !bytes.Equal(lifted.Image().Pix, flat.Image().Pix) {
			t.Errorf("%s: the tool card lifts by a shadow: %s", theme.name, firstDifference(lifted.Image(), flat.Image()))
		}
	}
}

// showProbe renders one probe. flat removes the shadow steps from the
// tokens the probe draws with; both frames settle first so a surface that
// enters over a motion step is measured at rest.
func showProbe(probe shadowProbe, dark, flat bool) *ui.Tester {
	tt := ui.NewTester(func(c *ui.Context) {
		k := newTokens(c.Theme())
		if flat {
			k = flatShadows(k)
		}
		probe.draw(c, k)
	}, probe.width, probe.height)
	if dark {
		tt.SetDark(true)
	}
	tt.Frame()
	if probe.settle != nil {
		probe.settle(tt)
	}
	return tt
}

// shadowProbes lists the surfaces the shadow steps reach, each drawn with the
// shell's own view code rather than a copy of it, so a missing step shows up
// here.
func shadowProbes() []shadowProbe {
	return []shadowProbe{
		{
			name: "main frame", width: 420, height: 240,
			draw: func(c *ui.Context, k tokens) {
				ui.Column(c).Fill().Padding(k.framePad, k.framePad, k.framePad, 0).
					Background(k.background).Children(func() {
					mainFrame(c, k, func() {
						ui.Text(c, "Frame body").FontSize(k.bodySize)
					})
				})
			},
			surface: func(t *testing.T, tt *ui.Tester) ui.Rect {
				info, ok := tt.AccessByRole(ui.RoleGroup, "Main area")
				if !ok {
					t.Fatalf("the main frame is not drawn, texts %q", tt.Texts())
				}
				return info.Bounds
			},
			side:  "bottom",
			label: "Frame body",
		},
		{
			name: "settings route", width: 900, height: 600,
			draw: func(c *ui.Context, k tokens) {
				renderSettings(c, k, SettingsView{Locale: LocaleEN}, wipCapabilities(fixture()), nil, nil, nil)
			},
			surface: func(t *testing.T, tt *ui.Tester) ui.Rect {
				info, ok := tt.AccessByRole(ui.RoleRegion, tr(LocaleEN, keySettingsRegion))
				if !ok {
					t.Fatalf("the settings route is not drawn, texts %q", tt.Texts())
				}
				return info.Bounds
			},
			side:  "bottom",
			label: tr(LocaleEN, keySettingsRegion),
		},
		{
			name: "composer card", width: 800, height: 400,
			draw: func(c *ui.Context, k tokens) {
				view := ComposerView{
					Draft: new(string), CanSubmit: true,
					Placeholder: composerPlaceholder, Focus: new(ui.Handle),
					Locale: LocaleEN,
				}
				ui.Column(c).Fill().Background(k.background).Children(func() {
					composerCard(c, k, view, Actions{})
				})
			},
			surface: func(t *testing.T, tt *ui.Tester) ui.Rect {
				return composerCardBounds(t, tt)
			},
			side:  "bottom",
			label: "Composer",
		},
		{
			name: "approval card", width: 800, height: 400,
			draw: func(c *ui.Context, k tokens) {
				ui.Column(c).Fill().Background(k.background).Children(func() {
					renderApprovalPanel(c, k, ApprovalView{
						Pending: []Approval{{Call: "bash", Reason: "write a file"}},
					}, Actions{})
				})
			},
			surface: func(t *testing.T, tt *ui.Tester) ui.Rect {
				info, ok := tt.AccessByRole(ui.RoleGroup, "Approval request")
				if !ok {
					t.Fatalf("the approval card is not drawn, texts %q", tt.Texts())
				}
				return info.Bounds
			},
			side:  "bottom",
			label: "Approval required",
		},
		{
			name: "search palette", width: 900, height: 700,
			draw: func(c *ui.Context, k tokens) {
				open, query, selected := true, "", 0
				renderSearchPalette(c, k, SearchPaletteView{
					Open: &open, Query: &query, Selected: &selected,
					Results: []SearchResult{{SessionID: "s1", Title: "First session"}},
					Locale:  LocaleEN,
				}, Actions{})
			},
			surface: func(t *testing.T, tt *ui.Tester) ui.Rect {
				info, ok := paletteDialog(tt)
				if !ok {
					t.Fatalf("the palette is not drawn, texts %q", tt.Texts())
				}
				return info.Bounds
			},
			side:   "bottom",
			label:  "First session",
			settle: func(tt *ui.Tester) { tt.Advance(searchMotion) },
		},
		{
			name: "model list dialog", width: 800, height: 600,
			draw: func(c *ui.Context, k tokens) {
				open := true
				composer(c, k, ComposerView{
					Draft: new(string), Placeholder: composerPlaceholder, Focus: new(ui.Handle),
					Locale: LocaleEN, ModelOpen: &open,
					ModelChoices: []ModelChoice{{ProviderID: "anthropic", ModelID: "claude", DisplayName: "Claude", Enabled: true}},
				}, Actions{})
			},
			surface: func(t *testing.T, tt *ui.Tester) ui.Rect {
				info, ok := tt.AccessByRole(ui.RoleDialog, "")
				if !ok {
					t.Fatalf("the model list is not drawn, texts %q", tt.Texts())
				}
				return info.Bounds
			},
			side:  "bottom",
			label: "Composer",
		},
		{
			name: "context panel", width: 520, height: 400,
			draw: func(c *ui.Context, k tokens) {
				s := &State{ContextPanel: ContextPanelView{}}
				ui.Row(c).Fill().Children(func() {
					ui.Box(c).Grow(1).FillHeight().Background(k.background)
					ui.Column(c).Width(contextPanelW + k.panelPad).FillHeight().Children(func() {
						s.contextPanelColumn(c, k, Actions{})
					})
				})
			},
			surface: func(t *testing.T, tt *ui.Tester) ui.Rect {
				info, ok := tt.AccessByRole(ui.RoleRegion, contextPanelRegionLabel)
				if !ok {
					t.Fatalf("the context panel is not drawn, texts %q", tt.Texts())
				}
				return info.Bounds
			},
			side:  "start",
			label: contextPanelTitleText,
		},
	}
}

// shadowSample is the point just beyond one edge of box where a surface's
// shadow falls. The hairline is drawn inside the box, so one DIP out is already
// outside the face and inside the step's blur; the edge's middle is sampled
// rather than a rounded corner, where the box's own edge is antialiased.
func shadowSample(box ui.Rect, side string) (int, int) {
	if side == "start" {
		return int(box.X) - 1, int(box.Y + box.H/2)
	}
	return int(box.X + box.W/2), int(box.Y+box.H) + 1
}

// imageHasPoint reports whether (x, y) is inside img.
func imageHasPoint(img *image.RGBA, x, y int) bool {
	return img != nil && x >= 0 && y >= 0 && x < img.Bounds().Dx() && y < img.Bounds().Dy()
}

// rgbSum is a pixel's brightness as one integer, so two frames can be compared
// for "darker" without matching exact bytes.
func rgbSum(img *image.RGBA, x, y int) int {
	p := img.RGBAAt(x, y)
	return int(p.R) + int(p.G) + int(p.B)
}

// firstDifference names the first pixel two frames of the same size disagree
// on, so a failure says where the unexpected paint is.
func firstDifference(a, b *image.RGBA) string {
	for y := 0; y < a.Bounds().Dy(); y++ {
		for x := 0; x < a.Bounds().Dx(); x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return "first difference at (" + strconv.Itoa(x) + ", " + strconv.Itoa(y) + ")"
			}
		}
	}
	return "no difference"
}
