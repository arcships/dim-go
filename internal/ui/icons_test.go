package ui

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

// sidebarGlyphCases names every glyph in the set, so a new icon cannot be
// added without the size and drawing checks seeing it.
func sidebarGlyphCases() map[string]*ui.SVG {
	return map[string]*ui.SVG{
		"plus":    sidebarIcons.plus,
		"search":  sidebarIcons.search,
		"clock":   sidebarIcons.clock,
		"grid":    sidebarIcons.grid,
		"folder":  sidebarIcons.folder,
		"house":   sidebarIcons.house,
		"user":    sidebarIcons.user,
		"gear":    sidebarIcons.gear,
		"chevron": sidebarIcons.chevron,
		"list":    sidebarIcons.list,
	}
}

// TestSidebarGlyphsAreSixteenSquares checks the set's drawing size: every
// glyph parses at the sidebar's 16 DIP step, so no entry draws an icon at a
// size of its own and no glyph needs a separate asset.
func TestSidebarGlyphsAreSixteenSquares(t *testing.T) {
	if sidebarIconSize != 16 {
		t.Fatalf("sidebarIconSize is %v, want 16", sidebarIconSize)
	}
	for name, glyph := range sidebarGlyphCases() {
		if glyph == nil {
			t.Errorf("glyph %q is not parsed", name)
			continue
		}
		if w, h := glyph.Size(); w != sidebarIconSize || h != sidebarIconSize {
			t.Errorf("glyph %q is %v×%v, want %v×%v", name, w, h, sidebarIconSize, sidebarIconSize)
		}
	}
}

// glyphFixtureState is the sidebar fixture with a labelled project group, so
// the test can find the group heading's glyph as well as the entries'.
func glyphFixtureState() *State {
	s := fixture()
	s.Sessions = []Session{
		{ID: "s1", Title: "First session", Group: "alpha"},
		{ID: "s2", Title: "Second session", Group: "alpha"},
	}
	s.Selected = "s1"
	return s
}

// glyphTester renders the sidebar and keeps the frame's tokens, so the glyph
// scan can tell the sidebar's face from the icon's ink.
func glyphTester(t *testing.T, s *State) (*ui.Tester, tokens) {
	t.Helper()
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		theme = c.Theme()
		s.View(c)
	}, 900, 600)
	return tt, newTokens(theme)
}

// TestSidebarGlyphDrawsAtSixteenAndStaysHidden checks the helper every entry
// uses: it lays the icon out at a 16×16 square, in the color it was given, and
// even a named glyph stays out of the accessibility tree, so assistive
// technology reads the entry's label once and not the decoration.
func TestSidebarGlyphDrawsAtSixteenAndStaysHidden(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		sidebarGlyph(c, sidebarIcons.plus, ui.RGB(0, 0, 255)).Label("plus glyph")
	}, 40, 40)

	rect, ok := tt.Find("plus glyph")
	if !ok {
		t.Fatalf("the glyph is not laid out")
	}
	if rect.W != sidebarIconSize || rect.H != sidebarIconSize {
		t.Errorf("the glyph is %v×%v DIPs, want %v×%v", rect.W, rect.H, sidebarIconSize, sidebarIconSize)
	}
	if _, ok := tt.AccessByRole(ui.RoleImage, "plus glyph"); ok {
		t.Error("the glyph is exposed to assistive technology even though its row names the entry")
	}
	if got := tt.Image().RGBAAt(int(rect.X+rect.W/2), int(rect.Y+rect.H/2)); got != (color.RGBA{R: 0, G: 0, B: 255, A: 255}) {
		t.Errorf("the glyph is %v, want the color it was drawn in", got)
	}
}

// glyphCase names one icon's label and whether Find names the whole row (a
// button or a group heading) or only the heading's text. The glyph sits at the
// row's own padding, or one gap and one icon left of a text label.
type glyphCase struct {
	label string
	row   bool
}

// sidebarGlyphCasesInTheFrame is every decoration the sidebar draws, with the
// kind of label Find returns for it.
func sidebarGlyphCasesInTheFrame() []glyphCase {
	return []glyphCase{
		{"New session", true},
		{"Search", true},
		{"Scheduled tasks", true},
		{"Skills and plugins", true},
		{"alpha", true},
		{"Workspace", false},
		{"Sign in to Dim", true},
		{"Settings", true},
	}
}

// glyphInk finds the ink of the glyph drawn before one label: the icon box is
// sidebarIconSize wide and sits on the row's center line, at the row's left
// padding for a row label or one gap and one icon left of a text label. It
// returns the box and whether it holds any pixel that is not the sidebar's
// face.
func glyphInk(tt *ui.Tester, c glyphCase, k tokens) (ui.Rect, bool, bool) {
	rect, ok := tt.Find(c.label)
	if !ok {
		return ui.Rect{}, false, false
	}
	left := rect.X + k.sidebarPadX
	if !c.row {
		left = rect.X - k.sidebarGap - sidebarIconSize
	}
	top := rect.Y + rect.H/2 - sidebarIconSize/2
	box := ui.Rect{X: left, Y: top, W: sidebarIconSize, H: sidebarIconSize}
	image := tt.Image()
	ink := false
	for y := int(top); y < int(top+sidebarIconSize); y++ {
		for x := int(left); x < int(left+sidebarIconSize); x++ {
			if got := image.RGBAAt(x, y); got != (color.RGBA{R: k.sidebar.R, G: k.sidebar.G, B: k.sidebar.B, A: k.sidebar.A}) {
				ink = true
			}
		}
	}
	return box, ok, ink
}

// TestSidebarGlyphsAreDrawnBeforeTheirLabels checks the drawn decoration: each
// global entry, the thread list heading, a project group heading, the account
// row and the settings entry paint ink in the 16 DIP box before their label.
func TestSidebarGlyphsAreDrawnBeforeTheirLabels(t *testing.T) {
	s := glyphFixtureState()
	tt, k := glyphTester(t, s)

	for _, c := range sidebarGlyphCasesInTheFrame() {
		if !tt.HasText(c.label) {
			t.Errorf("%q is not drawn, texts %q", c.label, tt.Texts())
			continue
		}
		box, found, ink := glyphInk(tt, c, k)
		if !found {
			t.Errorf("%q has no findable label", c.label)
			continue
		}
		if !ink {
			t.Errorf("%q draws no glyph in %+v", c.label, box)
		}
	}

	nav, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar")
	if !ok {
		t.Fatalf("the sidebar is not exposed, texts %q", tt.Texts())
	}
	for _, c := range sidebarGlyphCasesInTheFrame() {
		box, _, _ := glyphInk(tt, c, k)
		if box.X < nav.Bounds.X || box.X+box.W > nav.Bounds.X+nav.Bounds.W {
			t.Errorf("the glyph before %q at %+v runs outside the sidebar %+v", c.label, box, nav.Bounds)
		}
	}
}

// TestSidebarHeadingGlyphsSitAtTheRight checks the thread heading's add and
// list icons: the reference draws them at the heading's trailing edge, after
// the text, so the heading keeps its label at the start of the row.
func TestSidebarHeadingGlyphsSitAtTheRight(t *testing.T) {
	s := glyphFixtureState()
	tt, k := glyphTester(t, s)

	nav, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar")
	if !ok {
		t.Fatalf("the sidebar is not exposed, texts %q", tt.Texts())
	}
	heading, ok := tt.Find("Sessions")
	if !ok {
		t.Fatalf("the thread heading is not drawn, texts %q", tt.Texts())
	}
	image := tt.Image()
	face := color.RGBA{R: k.sidebar.R, G: k.sidebar.G, B: k.sidebar.B, A: k.sidebar.A}
	right := nav.Bounds.X + nav.Bounds.W - k.sidebarSectionPadX
	for i := 0; i < 2; i++ {
		box := ui.Rect{
			X: right - float32(i+1)*sidebarIconSize - float32(i)*k.sidebarGap,
			Y: heading.Y + heading.H/2 - sidebarIconSize/2,
			W: sidebarIconSize, H: sidebarIconSize,
		}
		ink := false
		for y := int(box.Y); y < int(box.Y+box.H); y++ {
			for x := int(box.X); x < int(box.X+box.W); x++ {
				if image.RGBAAt(x, y) != face {
					ink = true
				}
			}
		}
		if !ink {
			t.Errorf("heading icon %d paints no ink in %+v", i, box)
		}
		if box.X <= heading.X+heading.W {
			t.Errorf("heading icon %d at x=%.1f sits at or left of the text ending at %.1f",
				i, box.X, heading.X+heading.W)
		}
		if box.X+box.W > nav.Bounds.X+nav.Bounds.W {
			t.Errorf("heading icon %d at %+v runs outside the sidebar %+v", i, box, nav.Bounds)
		}
	}
}

// TestSidebarGlyphsKeepTheAccessibleNames checks that the decoration changes no
// name assistive technology reads: every entry keeps its label as its name,
// the WIP entries stay unavailable and explained, and the enabled search entry
// stays available.
func TestSidebarGlyphsKeepTheAccessibleNames(t *testing.T) {
	s := glyphFixtureState()
	tt, _ := glyphTester(t, s)

	for _, name := range []string{"New session", "Search", "Scheduled tasks", "Skills and plugins"} {
		info, ok := tt.AccessByRole(ui.RoleButton, name)
		if !ok {
			t.Errorf("%q is not exposed as a button, texts %q", name, tt.Texts())
			continue
		}
		if info.Name != name {
			t.Errorf("%q is exposed as %q", name, info.Name)
		}
	}
	search, _ := tt.AccessByRole(ui.RoleButton, "Search")
	if search.Disabled {
		t.Error("the search entry is unavailable although its filter is local")
	}
	for _, name := range []string{"New session", "Scheduled tasks", "Skills and plugins", "Sign in to Dim"} {
		info, ok := tt.AccessByRole(ui.RoleButton, name)
		if !ok {
			t.Errorf("%q is not exposed as a button, texts %q", name, tt.Texts())
			continue
		}
		if !info.Disabled {
			t.Errorf("%q is available although the shell cannot carry it out", name)
		}
		if !info.Focusable {
			t.Errorf("%q is unreachable, so its reason is never read", name)
		}
		if info.Description == "" {
			t.Errorf("%q lost the reason it waits on", name)
		}
	}
}

// TestSidebarGlyphsAddNoTabStop checks that the icons leave the sidebar's Tab
// order as it was: the glyphs are decorations and take no focus, so Tab still
// walks the entries in their documented order.
func TestSidebarGlyphsAddNoTabStop(t *testing.T) {
	s := glyphFixtureState()
	tt, _ := glyphTester(t, s)

	for _, want := range []string{
		"New session", "Search", "Scheduled tasks",
		"Skills and plugins", "Sessions",
	} {
		tt.Key(0, ui.KeyTab)
		if !tt.Focused(want) {
			t.Fatalf("Tab does not reach %q, texts %q", want, tt.Texts())
		}
	}
}

// TestSidebarGlyphsFitTheNarrowestSidebar checks the narrow width: at the
// sidebar's 200 DIP minimum every labelled entry and its glyph stay inside the
// navigation region instead of running under the conversation.
func TestSidebarGlyphsFitTheNarrowestSidebar(t *testing.T) {
	s := glyphFixtureState()
	s.SidebarWidth = sidebarMinW
	tt, k := glyphTester(t, s)

	nav, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar")
	if !ok {
		t.Fatalf("the sidebar is not exposed, texts %q", tt.Texts())
	}
	if nav.Bounds.W > sidebarMinW+0.5 {
		t.Fatalf("the sidebar is %v wide, want the %v minimum", nav.Bounds.W, sidebarMinW)
	}
	for _, c := range sidebarGlyphCasesInTheFrame() {
		rect, ok := tt.Find(c.label)
		if !ok {
			t.Errorf("%q is not drawn at the narrowest width, texts %q", c.label, tt.Texts())
			continue
		}
		if rect.X < nav.Bounds.X || rect.X+rect.W > nav.Bounds.X+nav.Bounds.W+0.5 {
			t.Errorf("%q at %+v runs outside the sidebar %+v", c.label, rect, nav.Bounds)
		}
		box, _, _ := glyphInk(tt, c, k)
		if box.X < nav.Bounds.X || box.X+box.W > nav.Bounds.X+nav.Bounds.W {
			t.Errorf("the glyph before %q at %+v runs outside the sidebar %+v", c.label, box, nav.Bounds)
		}
	}
}
