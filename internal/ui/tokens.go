package ui

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// The shell's motion steps, from dim-electron's stylesheet: a layout or panel
// change takes 250 ms on cubic-bezier(0.4, 0, 0.2, 1), and a control's colors
// 150 ms on cubic-bezier(0.23, 1, 0.32, 1). They are the reference's
// --layout-transition-ms and --ease-in-out, and the transition Tailwind
// attaches to its buttons
// (transition-[color,background-color,border-color,box-shadow,transform]).
// The reference's 300 ms image fade is not here: the shell draws no image that
// fades in.
const (
	motionFast   = 150 * time.Millisecond
	motionLayout = 250 * time.Millisecond
)

// The curves are CSS cubic-bezier() through mygo's solver, so a fraction of
// the elapsed time maps to the same progress the reference's CSS asks for.
var (
	motionEaseOut   = ui.CubicBezier(0.23, 1, 0.32, 1)
	motionEaseInOut = ui.CubicBezier(0.4, 0, 0.2, 1)
)

// motionColors is a transition of an element's colors: a control's hover face
// at the fast step, or a surface's theme change at the layout step. While the
// desktop asks for less motion mygo keeps color fades going, so the Duration
// is negative instead: a negative Duration shows the new colors at once, which
// is the shell's motion-reduce path.
func motionColors(c *ui.Context, d time.Duration, ease ui.Easing) ui.ElementTransition {
	if c.Preferences().ReduceMotion {
		d = -1
	}
	return ui.ElementTransition{Duration: d, Ease: ease, Colors: true}
}

// motionFastColors is the 150 ms color step of a sidebar row, a toolbar entry
// or a composer control under the pointer.
func motionFastColors(c *ui.Context) ui.ElementTransition {
	return motionColors(c, motionFast, motionEaseOut)
}

// motionLayoutColors is the 250 ms color step of a surface that changes with
// the theme, as the reference transitions every color on a theme switch.
func motionLayoutColors(c *ui.Context) ui.ElementTransition {
	return motionColors(c, motionLayout, motionEaseInOut)
}

// motionPanel is the appearance and disappearance of a panel that takes the
// main column or sits beside it: the layout timing and curve, position, size
// and colors so the room it takes moves with it, and a small fade-and-lift so
// it does not pop. The caller keeps the panel's identity with a Key, so mygo
// animates the same panel from frame to frame instead of replacing it.
func motionPanel(c *ui.Context) ui.ElementTransition {
	t := ui.ElementTransition{
		Duration: motionLayout,
		Ease:     motionEaseInOut,
		Position: true,
		Size:     true,
		Colors:   true,
		Enter:    &ui.Motion{Opacity: 0, Y: 4},
		Exit:     &ui.Motion{Opacity: 0, Y: 4},
	}
	if c.Preferences().ReduceMotion {
		t.Duration = -1
		t.ExitDuration = -1
	}
	return t
}

// motionSidePanel is motionPanel for a panel beside the conversation, whose
// room the layout gives it in a row: it opens by collapsing its width, so the
// conversation gives the room up smoothly instead of jumping narrow, and goes
// the same way. The enter starts transparent as well, so the content does not
// pop into a zero-width box.
func motionSidePanel(c *ui.Context) ui.ElementTransition {
	t := motionPanel(c)
	t.Enter = &ui.Motion{Opacity: 0, Collapse: true}
	t.Exit = &ui.Motion{Opacity: 0, Collapse: true}
	return t
}

// The shell's type steps, as fractions of the UI font size: the shell's own
// scale on top of the step in force, so that a larger theme text size or a
// larger saved font-size preference grows every heading, label and row
// together.
const (
	typeTitleStep = 1.15
	typeLabelStep = 0.9
	typeSmallStep = 0.8
)

// fontSizeSteps are the UI font sizes the appearance settings offer, in DIPs.
// The shell's body text sits on one of the steps, so a theme size between two
// of them snaps to the nearer one and every step of the type scale moves with
// it.
var fontSizeSteps = [...]float32{12, 13, 14, 16, 18}

// The type sizes the shell falls back to when the theme names none, in DIPs:
// the UI's default step, and the code size one step below it.
const (
	defaultUIFontSize   = 14
	defaultCodeFontSize = 13
)

// densityCompactScale is the share of the regular room compact keeps: three
// quarters. It is mygo's own step between a roomy and a compact spacing unit,
// where ui.Theme.Spacing 4 is the default and 3 is "compact" (theme.go:68-72),
// so the shell tightens by the same ratio its widgets do. The factor is folded
// into the gaps and paddings copied from dim-electron's Tailwind layout, which
// are literals rather than multiples of the spacing unit and so do not follow
// it; heights, widths and type sizes keep their values, so compact leaves more
// of the window to rows without a row's text outgrowing its box.
const densityCompactScale = 3.0 / 4.0

// monospaceFamily is the family the Monospace preference draws its text in.
// mygo resolves "monospace" to the platform's monospaced font; dim-electron
// ships self-hosted Inter and JetBrains Mono stacks instead (global.css:10-18),
// and this shell bundles no font file, so it names the platform families it can
// actually reach rather than a face the desktop would have to ship.
const monospaceFamily = "monospace"

// The radius ladder, as dim-electron's --radius-{sm,md,lg,xl}: an inline
// control's corner, a field's or a chip's, a card's, and a floating surface's,
// in DIPs.
//
// The reference derives the ladder from a rem base: --radius is 0.625rem and
// each step is calc(var(--radius) ± Npx) in global.css:45-48, so at the
// reference's default 14px root font size the ladder is 4.75 / 6.75 / 8.75 /
// 12.75, not the 6 / 8 / 10 / 14 the 16px root happens to produce. The shell
// draws at the reference's 14px root, so it keeps those rem-derived values;
// scaling them again with the saved type step would be a second, unrequested
// change.
const (
	radiusSM = 4.75
	radiusMD = 6.75
	radiusLG = 8.75
	radiusXL = 12.75
)

// The sidebar's width in DIPs: the width it opens at, and the narrowest and
// widest a resize may leave it.
const (
	sidebarDefaultW = 256
	sidebarMinW     = 200
	sidebarMaxW     = 360
)

// The main column's frame, from dim-electron's chat-workspace utilities
// (packages/desktop/packages/renderer/src/styles/global.css:19-31): the framed
// workspace is `padding: 4.5px` with `padding-left: 0` while the sidebar is
// open, and its card is `rounded-xl`. docs/l5-reference.md §6 measures the same
// frame on window-workspace.png as a 5 DIP top gap and 4 DIP right and bottom
// gaps at an 800×720 window; 4.5 is the CSS literal those measurements round.
const (
	mainFramePad    = 4.5
	mainFrameRadius = radiusXL
)

// The card and conversation-row insets, converted from the reference's rem
// utilities at its 14px root: a card's px-3 is 10.5 DIPs and a timeline row's
// px-4 is 14, the same conversions the timeline constants already use. They are
// kept as literals rather than Space(n) so a card and a row keep the
// reference's converted inset instead of rounding to the theme's 4 DIP spacing
// unit.
const (
	cardPadXStep = 10.5
	rowPadXStep  = 14
)

// The empty state's hero, from NewThread.vue:294-300. The reference draws the
// official DimAgent mark (a 45:32 filled shape) at h-14, which is 3.5rem = 49
// DIPs high at the 14px root, with a gap-4 (14 DIP) between the mark and the
// heading, and a p-8 (28 DIP) inset. The heading is text-2xl and semibold; the
// reference's stylesheet puts it at 21/28, while the rendered window measures
// about 22 high, so the step keeps the measured 22 at 14 DIPs and the line box
// the rem-derived 28.
const (
	heroMarkH     = 49                      // h-14 at the reference's 14px root
	heroMarkW     = heroMarkH * 45.0 / 32.0 // the official mark's 45:32 viewBox
	heroTitleStep = 22.0 / 14.0             // the heading's 22 DIP at the 14 DIP body
	heroLineStep  = 2.0                     // leading-8: the 2rem line box, 28 at 14
	heroGap       = 14                      // gap-4
	heroPad       = 28                      // p-8
)

// The sidebar's geometry in DIPs, from docs/ui-layout-spec.md §2.2: the
// header is a 48px literal, the table title is h-9 with px-3, a global entry is
// h-8 with px-2, the footer row h-[30px], and the gaps and inner cells are
// px-2 / gap-1 / h-6. Under the 14px root font size those Tailwind steps are
// 31.5, 10.5, 28, 7, 30, 3.5 and 21 DIPs, so they are kept as their converted
// values instead of being copied as steps on the 4 DIP spacing grid.
const (
	sidebarHeaderH    = 48
	sidebarHeaderTop  = 4
	sidebarSectionH   = 31.5
	sidebarRowH       = 34
	sidebarEntryH     = 28
	sidebarFooterH    = 30
	sidebarCell       = 21
	sidebarGap        = 3.5
	sidebarPad        = 7
	sidebarSectionPad = 10.5
)

// The timeline's geometry in DIPs, from docs/ui-layout-spec.md §4.2: the row
// shell is px-4, a block sits mb-3 below the block before it, a disclosure's
// header is a h-7 summary button with px-0, thinking text is pl-3 behind a 2px
// rail, the thinking and tool panels cap at 240px, the user bubble is 85% wide
// with px-4 py-2 and rounded-lg, and a file-change row is px-2 py-1.5 with a
// w-5 status letter. Under the 14px root font size those Tailwind steps are 14,
// 10.5, 24.5, 10.5, 2, 240, 85, 14, 7, 10, 7, 5.25 and 17.5 DIPs; the px
// literals (240, 85) stay literal, as dim-electron's own CSS keeps them.
const (
	timelinePadX       = 14   // px-4: the row shell's left and right
	timelineBlockGap   = 10.5 // mb-3: between two blocks of one turn
	timelineHeaderH    = 24.5 // h-7: a summary or disclosure header's height
	timelinePadStart   = 10.5 // pl-3: thinking text and tool panel inset
	timelineRail       = 2    // border-l-2: thinking text's rail
	timelinePanelMaxH  = 240  // max-h-[240px]: a thinking or tool panel's cap
	timelineChevron    = 14   // h-4: a disclosure's arrow box
	timelineUserWidth  = 85   // max-w-[85%]: the user bubble's widest
	timelineUserPadY   = 7    // py-2: the user bubble's top and bottom
	timelineUserPadX   = 14   // px-4: the user bubble's left and right
	timelineBubbleR    = radiusLG
	timelineFileRowY   = 5.25 // py-1.5: a file-change row's top and bottom
	timelineFileRowX   = 7    // px-2: a file-change row's left and right
	timelineFileGap    = 10.5 // gap-3: between a file row's parts
	timelineFileStatus = 17.5 // w-5: the file row's status letter cell
)

// The surface ladder: how far each face lifts off the background, as the share
// of the way from the background to the foreground. A face is a mix of two
// theme colors, not a wash of the border, so an elevated surface keeps a tint
// of its own in both themes instead of reading as the border at 12%: the
// background is the window, the sidebar and the cards step out of it, and a
// popover sits on top of them.
const (
	sidebarRise = 0.03
	cardRise    = 0.12
	popoverRise = 0.18
)

// The tooltip's own steps: its edge and its chips lift off its face, and its
// secondary text fades toward it. The face itself is the theme's inverse pair,
// the theme turned over so a tip stands out from whatever it covers.
const (
	tooltipBorderRise = 0.18
	tooltipChipRise   = 0.10
	tooltipMutedFade  = 0.30

	// sidebarHoverRise is how far a sidebar row lifts under the pointer, as a
	// mix toward the foreground; the base face is the chrome step, not the row
	// fill.
	sidebarHoverRise = 0.06

	// sidebarSelectionRise tints the chosen row toward the brand color, so the
	// choice reads on the chrome step instead of repeating the hover face.
	sidebarSelectionRise = 0.08
)

// The reference's brand fills, from `--brand-green` in global.css:
// oklch(0.60 0.138 150) in the light appearance and oklch(0.76 0.15 149) in the
// dark one. They are constants because the reference's brand is not the
// platform theme's accent.
var (
	brandGreenLight = ui.Hex("#349651")
	brandGreenDark  = ui.Hex("#64cb7b")
)

// washAlpha is how strongly the shell tints a color it uses as the background
// of a banner or a status area. Faces are mixed (see the surface ladder), not
// washed: a tint over the border has no contrast of its own.
const washAlpha = 0.12

// chromePalette is one of dim-electron's OS-tribute chrome palettes, taken from
// its styles/retro-themes.css. The reference names thirty --chrome-* variables
// per theme; this shell keeps the ten faces its own semantic layer needs, plus
// the two radii and the family the theme names.
//
// Only what the shell can draw is imitated: the reference's gradients,
// 3-D bevels, inset shadows, title-bar stripes and window chrome are not
// reproduced, because this desktop uses the system title bar and draws flat
// faces. The colors, the corner radius and the UI family are.
type chromePalette struct {
	// dark is whether this palette is one of the reference's dark blocks. Only
	// macOS Big Sur has one; the other chrome themes draw their light palette in
	// both appearances, as the reference's color-scheme: light does.
	dark bool
	// The semantic faces, in the reference's names: the window's canvas and
	// ink, the panel its cards and sidebar sit on, the menu a popover uses, the
	// muted secondary text, the edge between faces, the selection fill with the
	// text on it, the hover face, and the tooltip's face.
	canvas       ui.Color
	ink          ui.Color
	panel        ui.Color
	menu         ui.Color
	muted        ui.Color
	edge         ui.Color
	selection    ui.Color
	selectionInk ui.Color
	hover        ui.Color
	tip          ui.Color
	// radius is the reference's --chrome-radius: the corner of a card, a
	// dialog or a popover. controlRadius is --chrome-control-radius, the corner
	// of a button or a field; the reference clamps an input's corner to 6 DIP
	// (min(var(--chrome-control-radius), 6px)).
	radius        float32
	controlRadius float32
	// font is the reference's --chrome-ui-font-family stack. mygo resolves a
	// comma-separated family list the way CSS does: it uses the first family
	// the platform has, so the same stack serves every desktop.
	font string
}

// The reference's interface stacks, from --chrome-ui-font-family. mygo resolves
// a comma-separated list like CSS, so the whole stack is kept: the Windows
// stacks name Tahoma, the mac ones the system face, and every stack ends at a
// family the platform's text engine falls back to.
const (
	chromeTahomaStack = `Tahoma, "Microsoft Sans Serif", "Segoe UI", "Microsoft YaHei UI", "PingFang SC", sans-serif`
	chromeSegoeStack  = `"Segoe UI", Tahoma, "Microsoft YaHei UI", "PingFang SC", sans-serif`
	chromeLucidaStack = `"Lucida Grande", "Lucida Sans Unicode", "Hiragino Sans GB", "PingFang SC", sans-serif`
	chromeAppleStack  = `-apple-system, BlinkMacSystemFont, "SF Pro Text", "Helvetica Neue", "PingFang SC", "Microsoft YaHei UI", sans-serif`
)

// The reference's six palettes (retro-themes.css, the windows-98, windows-xp,
// windows-vista, osx-tiger and macos-bigsur blocks). default is the absence of
// a chrome theme and has no palette of its own.
var (
	chromeWindows98 = chromePalette{
		canvas:        ui.Hex("#ffffff"),
		ink:           ui.Hex("#000000"),
		panel:         ui.Hex("#c0c0c0"),
		menu:          ui.Hex("#c0c0c0"),
		muted:         ui.Hex("#444444"),
		edge:          ui.Hex("#808080"),
		selection:     ui.Hex("#000080"),
		selectionInk:  ui.Hex("#ffffff"),
		hover:         ui.Hex("#d8d8d8"),
		tip:           ui.Hex("#ffffe1"),
		radius:        0,
		controlRadius: 0,
		font:          chromeTahomaStack,
	}
	chromeWindowsXP = chromePalette{
		canvas:        ui.Hex("#ffffff"),
		ink:           ui.Hex("#000000"),
		panel:         ui.Hex("#ece9d8"),
		menu:          ui.Hex("#ffffff"),
		muted:         ui.Hex("#5a594e"),
		edge:          ui.Hex("#aca899"),
		selection:     ui.Hex("#316ac5"),
		selectionInk:  ui.Hex("#ffffff"),
		hover:         ui.Hex("#dce8fc"),
		tip:           ui.Hex("#ffffe1"),
		radius:        7,
		controlRadius: 3,
		font:          chromeTahomaStack,
	}
	chromeWindowsVista = chromePalette{
		canvas:        ui.Hex("#ffffff"),
		ink:           ui.Hex("#17232b"),
		panel:         ui.Hex("#e8eff4"),
		menu:          ui.Hex("#f0f0f0"),
		muted:         ui.Hex("#52616a"),
		edge:          ui.Hex("#8b9ba5"),
		selection:     ui.Hex("#0078d7"),
		selectionInk:  ui.Hex("#ffffff"),
		hover:         ui.Hex("#d9effd"),
		tip:           ui.Hex("#fffef4"),
		radius:        7,
		controlRadius: 3,
		font:          chromeSegoeStack,
	}
	chromeOSXTiger = chromePalette{
		canvas:        ui.Hex("#ffffff"),
		ink:           ui.Hex("#171717"),
		panel:         ui.Hex("#ececec"),
		menu:          ui.Hex("#f8f8f8"),
		muted:         ui.Hex("#595959"),
		edge:          ui.Hex("#999999"),
		selection:     ui.Hex("#3875d7"),
		selectionInk:  ui.Hex("#ffffff"),
		hover:         ui.Hex("#d8e6f8"),
		tip:           ui.Hex("#ffffdf"),
		radius:        7,
		controlRadius: 999,
		font:          chromeLucidaStack,
	}
	chromeMacOSBigSur = chromePalette{
		canvas:        ui.Hex("#ffffff"),
		ink:           ui.Hex("#262426"),
		panel:         ui.Hex("#f0eeef"),
		menu:          ui.Hex("#f6f5f5"),
		muted:         ui.Hex("#656166"),
		edge:          ui.Hex("#c6c3c6"),
		selection:     ui.Hex("#007aff"),
		selectionInk:  ui.Hex("#ffffff"),
		hover:         ui.Hex("#e2e0e3"),
		tip:           ui.Hex("#f5f3f5"),
		radius:        10,
		controlRadius: 6,
		font:          chromeAppleStack,
	}
	chromeMacOSBigSurDark = chromePalette{
		dark:          true,
		canvas:        ui.Hex("#1e1e22"),
		ink:           ui.Hex("#f3f1f5"),
		panel:         ui.Hex("#29282d"),
		menu:          ui.Hex("#353439"),
		muted:         ui.Hex("#b9b6be"),
		edge:          ui.Hex("#545159"),
		selection:     ui.Hex("#0a84ff"),
		selectionInk:  ui.Hex("#ffffff"),
		hover:         ui.Hex("#45434b"),
		tip:           ui.Hex("#353439"),
		radius:        10,
		controlRadius: 6,
		font:          chromeAppleStack,
	}
)

// chromePaletteFor returns the palette of c in the appearance the window is
// in. The reference has a dark block only for macOS Big Sur, so every other
// chrome draws its light palette in the dark appearance too, exactly as its
// color-scheme: light does. default names no palette.
func chromePaletteFor(c Chrome, dark bool) (chromePalette, bool) {
	switch c {
	case ChromeWindows98:
		return chromeWindows98, true
	case ChromeWindowsXP:
		return chromeWindowsXP, true
	case ChromeWindowsVista:
		return chromeWindowsVista, true
	case ChromeOSXTiger:
		return chromeOSXTiger, true
	case ChromeMacOSBigSur:
		if dark {
			return chromeMacOSBigSurDark, true
		}
		return chromeMacOSBigSur, true
	default:
		return chromePalette{}, false
	}
}

// chromeRadiusLadder is the radius ladder a chrome palette asks for, from the
// reference's --radius-sm: calc(var(--radius) - 4px) and its three siblings in
// global.css. A negative step is a square corner and is held at 0.
func chromeRadiusLadder(radius float32) (sm, md, lg, xl float32) {
	clamp := func(r float32) float32 {
		if r < 0 {
			return 0
		}
		return r
	}
	return clamp(radius - 4), clamp(radius - 2), clamp(radius), clamp(radius + 4)
}

// chromeControlRadius is the corner a chrome theme gives mygo's own controls:
// the reference's --chrome-control-radius, clamped to the 6 DIP its inputs ask
// for (min(var(--chrome-control-radius), 6px)), so a pill theme does not turn
// every themed widget into a pill here.
func chromeControlRadius(radius float32) float32 {
	if radius > 6 {
		return 6
	}
	if radius < 0 {
		return 0
	}
	return radius
}

// applyChromeTheme folds a chrome palette into the frame's theme, so mygo's own
// widgets (buttons, fields, lists, scroll bars) draw the same faces as the
// shell's views. The palette overrides the theme's colors, radius, family and
// appearance; danger, warning and success keep the theme's, because the
// reference's chrome blocks do not color them.
func applyChromeTheme(t *ui.Theme, p chromePalette, family FontFamily) {
	t.Dark = p.dark
	t.Background = p.canvas
	t.Text = p.ink
	t.TextMuted = p.muted
	t.Surface = p.panel
	t.SurfaceHover = p.hover
	t.SurfacePressed = p.hover
	t.Border = p.edge
	t.Accent = p.selection
	t.AccentHover = p.selection
	t.AccentPressed = p.selection
	t.AccentText = p.selectionInk
	t.InactiveAccent = p.selection
	t.Focus = p.selection
	t.Inverse = p.tip
	t.InverseText = p.ink
	t.Scrollbar = p.edge
	t.Radius = chromeControlRadius(p.controlRadius)
	// The chrome family is the reference's interface stack; an explicit
	// Monospace preference still wins, because this shell draws the interface
	// and code in one family and the preference names that family.
	if family != FontMonospace {
		t.Font = p.font
	}
}

// applyChromeTokens replaces the faces buildTokens derived from the theme with
// the chrome palette's, so a card is the reference's panel instead of a step
// toward white and a popover is its menu. The radius ladder follows the
// palette's --chrome-radius, and the tooltip is the reference's tip face.
func applyChromeTokens(k tokens, p chromePalette) tokens {
	k.background = p.canvas
	k.foreground = p.ink
	k.sidebar = p.panel
	k.card = p.panel
	k.cardForeground = p.ink
	k.popover = p.menu
	k.popoverForeground = p.ink

	k.sidebarForeground = p.ink
	k.sidebarPrimary = p.selection
	k.sidebarPrimaryForeground = p.selectionInk
	k.sidebarAccent = p.selection
	k.sidebarAccentForeground = p.selectionInk
	k.sidebarHover = p.hover
	k.sidebarBorder = p.edge
	k.sidebarRing = p.selection

	k.primary = p.selection
	k.onPrimary = p.selectionInk
	k.secondary = p.panel
	k.secondaryForeground = p.ink
	k.muted = p.panel
	k.mutedForeground = p.muted
	k.accent = p.hover
	k.accentForeground = p.ink

	k.border = p.edge
	k.input = p.edge
	k.ring = p.selection

	k.tooltip = p.tip
	k.tooltipForeground = p.ink
	k.tooltipMutedForeground = p.muted
	k.tooltipBorder = p.edge
	k.tooltipChip = p.hover

	sm, md, lg, xl := chromeRadiusLadder(p.radius)
	k.radiusSM, k.radiusMD, k.radiusLG, k.radiusXL = sm, md, lg, xl
	k.cardRadius = lg
	k.frameRadius = xl
	k.timelineBubbleR = lg
	return k
}

// tokens are the shell's design tokens: the room, type, radius and color steps
// its views use, so that no view carries a magic number or a color of its own.
// They are derived every frame from the window's theme and the desktop's saved
// appearance, so a theme change, dark mode, a saved font size or a density
// carries through the whole shell; see newTokensFor.
//
// The colors are the semantic layer dim-electron names in its stylesheet,
// derived here from ui.Theme: a view asks for what a surface is (a card, a
// muted face, the text on the primary fill), never for a theme field, so the
// same view keeps its meaning when the theme's own palette changes.
type tokens struct {
	// theme is the theme the frame draws with, the saved appearance already
	// folded in by appearanceTheme: space asks it for the spacing unit, so a
	// compact density tightens a view's own steps as it tightens the widgets.
	theme *ui.Theme

	// Room, in DIPs, on the theme's Spacing unit. A compact density scales the
	// unit and the literal steps below by densityCompactScale.
	hairline    float32 // a divider's width
	gap         float32 // between the parts of a row
	gapWide     float32 // between the parts of a toolbar row
	rowPadY     float32 // above and below a conversation row
	rowPadX     float32 // left and right of a conversation row
	toolbarPadY float32 // above and below the toolbar
	headerH     float32 // the conversation bar's height, as the reference's 41
	toolbarPadX float32 // left and right of the toolbar
	edgePad     float32 // inside the composer
	panelPad    float32 // inside the empty state and the loading row
	composerH   float32
	statusDot   float32
	cardRadius  float32 // a card's corner, the middle of the radius ladder
	cardPadY    float32 // above and below a card's parts
	cardPadX    float32 // left and right inside a card

	// The radius ladder's four steps, in DIPs: an inline control's corner, a
	// field's or a chip's, a card's and a floating surface's. They are the
	// reference's --radius-{sm,md,lg,xl}, derived from its --radius. A chrome
	// theme re-derives them from its own --chrome-radius (chromeRadiusLadder),
	// so a chosen chrome rounds the shell's own corners as the reference's
	// stylesheet rounds its cards.
	radiusSM float32
	radiusMD float32
	radiusLG float32
	radiusXL float32

	// The main column's frame: the gap between the card and the window edge,
	// and the card's own corner. See mainFramePad and mainFrameRadius.
	framePad    float32
	frameRadius float32

	// The empty state's hero: the brand mark's box, the title's step and line
	// box, and the gap and inset around them. See heroMarkH and heroTitleStep.
	heroMarkW      float32
	heroMarkH      float32
	heroTitleSize  float32
	heroTitleLineH float32
	heroGap        float32
	heroPad        float32

	// The timeline's geometry, in DIPs: the row shell's inset, the gap
	// between two blocks of one turn, a disclosure header and its arrow, the
	// thinking rail and panel cap, the user bubble, and a file-change row.
	// See docs/ui-layout-spec.md §4.2.
	timelinePadX       float32
	timelineBlockGap   float32
	timelineHeaderH    float32
	timelinePadStart   float32
	timelineRail       float32
	timelinePanelMaxH  float32
	timelineChevron    float32
	timelineUserWidth  float32
	timelineUserPadY   float32
	timelineUserPadX   float32
	timelineBubbleR    float32
	timelineFileRowY   float32
	timelineFileRowX   float32
	timelineFileGap    float32
	timelineFileStatus float32

	// The sidebar's geometry, in DIPs: the header's height and top margin,
	// the table title, a thread row, a global entry, the footer row, the
	// status cell and the gaps between them, plus the padding the header,
	// entries and footer keep from the sidebar's edges.
	sidebarHeaderH     float32
	sidebarHeaderTop   float32
	sidebarSectionH    float32
	sidebarRowH        float32
	sidebarEntryH      float32
	sidebarFooterH     float32
	sidebarCell        float32
	sidebarGap         float32
	sidebarPadX        float32
	sidebarSectionPadX float32

	// The sidebar's width in DIPs: the width it opens at, and the narrowest and
	// widest a resize may leave it.
	sidebarW    float32
	sidebarMinW float32
	sidebarMaxW float32

	// Type sizes, in DIPs.
	uiFontSize   float32 // the appearance step the body sits on
	codeFontSize float32 // the size of code
	titleSize    float32
	bodySize     float32
	labelSize    float32
	smallSize    float32

	// Surfaces, from the window out to the faces that lift off it:
	// background → sidebar → card → popover.
	background        ui.Color
	foreground        ui.Color
	sidebar           ui.Color
	card              ui.Color
	cardForeground    ui.Color
	popover           ui.Color
	popoverForeground ui.Color

	// The sidebar's own set: its shape above, its text, the brand mark it
	// carries, and the faces of a hovered or selected row.
	sidebarForeground        ui.Color
	sidebarPrimary           ui.Color
	sidebarPrimaryForeground ui.Color
	sidebarAccent            ui.Color
	sidebarAccentForeground  ui.Color
	sidebarHover             ui.Color
	sidebarBorder            ui.Color
	sidebarRing              ui.Color

	// Interactive faces and the text that sits on them. primary is the brand
	// fill with its paired onPrimary; secondary and muted are neutral faces,
	// with mutedForeground as the secondary text that sits on any face; accent
	// is a neutral hover face, not the brand color.
	primary             ui.Color
	onPrimary           ui.Color
	secondary           ui.Color
	secondaryForeground ui.Color
	muted               ui.Color
	mutedForeground     ui.Color
	accent              ui.Color
	accentForeground    ui.Color
	danger              ui.Color
	onDanger            ui.Color

	// Chrome: the outline between faces, the edge of a field, the keyboard
	// focus ring, and the colors a run, a tool call or a connection is drawn
	// in.
	border  ui.Color
	input   ui.Color
	ring    ui.Color
	success ui.Color
	warning ui.Color

	// The tooltip: a face turned over from the window's, the text on it, its
	// secondary text, its edge and the chips inside it.
	tooltip                ui.Color
	tooltipForeground      ui.Color
	tooltipMutedForeground ui.Color
	tooltipBorder          ui.Color
	tooltipChip            ui.Color
}

// newTokens builds the shell's tokens from the window's theme alone: the
// appearance a window that has saved no preference draws, with the theme's own
// type size and spacing.
func newTokens(t *ui.Theme) tokens {
	return buildTokens(t, 1)
}

// newTokensFor builds the tokens for one saved appearance: the window's theme
// for its colors, with the desktop's own type step, family and density on top.
// The saved values are resolved against the theme first, so a zero field takes
// the same default the appearance surface draws and a saved value outside the
// offered range is held to it.
//
// The type step moves every step of the scale together through the appearance
// theme, and the density folds into the room both in the theme's spacing unit
// (which mygo's own widgets read) and in the layout's literal steps; see
// appearanceTheme and densityScale.
//
// A chosen chrome theme replaces the theme's own palette before the tokens are
// derived and again on the derived faces, so every region draws the reference's
// canvas, panel, menu, selection and edge rather than the platform theme's.
func newTokensFor(t *ui.Theme, prefs SettingsView) tokens {
	t = themeOrLight(t)
	v := resolveSettings(newTokens(t), prefs)
	k := buildTokens(appearanceTheme(t, v), densityScale(v.Density))
	// The UI steps already follow the appearance theme's size; the code size
	// is not a step of that scale, so it is folded in here.
	k.codeFontSize = v.CodeFontSize
	if p, ok := chromePaletteFor(v.Chrome, t.Dark); ok {
		k = applyChromeTokens(k, p)
	}
	return k
}

// appearanceTheme is the theme the frame draws with: the window's theme with
// the saved type size, family and density folded in. These are the theme's own
// two type fields and the spacing unit, so setting them keeps mygo's widgets
// (buttons, fields, the composer's editor, list rows) in step with the shell's
// views; without it a saved size would grow the shell's text and leave every
// widget on the theme's default.
//
// Regular density leaves the spacing unit as the theme set it, and the system
// family is mygo's empty one, the platform's interface font.
//
// The saved size is the step the shell draws at, not a multiplier: a desktop
// that names one has chosen it, so it replaces the theme's own size instead of
// scaling it again. With no saved size the resolved step is the theme's (which
// already follows a larger system text size).
//
// A chosen chrome palette then replaces the theme's colors, radius and family
// (applyChromeTheme), so mygo's widgets draw the same chrome as the shell's
// own views. The density stays on top: compact tightens a chrome theme too.
func appearanceTheme(t *ui.Theme, v SettingsView) *ui.Theme {
	applied := *t
	applied.FontSize = v.FontSize
	applied.Font = fontFamilyStack(v.FontFamily)
	if v.Density == DensityCompact {
		applied.Spacing = spacingUnit(t) * densityCompactScale
	}
	if p, ok := chromePaletteFor(v.Chrome, t.Dark); ok {
		applyChromeTheme(&applied, p, v.FontFamily)
	}
	return &applied
}

// spacingUnit is the theme's spacing unit in DIPs, mygo's default of 4 when the
// theme names none.
func spacingUnit(t *ui.Theme) float32 {
	if t.Spacing <= 0 {
		return 4
	}
	return t.Spacing
}

// fontFamilyStack is the family a font-family preference draws in: the
// platform's interface font for System (mygo's empty family) and the
// platform's monospaced family for Monospace.
func fontFamilyStack(f FontFamily) string {
	if f == FontMonospace {
		return monospaceFamily
	}
	return ""
}

// densityScale is how much of the regular room a density keeps: all of it for
// regular, and densityCompactScale for compact.
func densityScale(d Density) float32 {
	if d == DensityCompact {
		return densityCompactScale
	}
	return 1
}

// themeOrLight is t, or the light theme when a caller passed none.
func themeOrLight(t *ui.Theme) *ui.Theme {
	if t == nil {
		return ui.LightTheme()
	}
	return t
}

// buildTokens derives the tokens from a theme: the type size and the spacing
// unit come from the theme's own fields, and geometryScale tightens the literal
// geometry steps (the Tailwind conversions of the sidebar and timeline
// ladders), which are not multiples of the spacing unit and so do not follow a
// compact unit on their own. 1 keeps the theme's own room.
func buildTokens(t *ui.Theme, geometryScale float32) tokens {
	if t == nil {
		t = ui.LightTheme()
	}
	fontSize := snapFontSize(t.FontSize)

	// Elevation lifts a surface toward white, the way the reference does in both
	// appearances: a card is a step off the window and a popover another step on
	// top. Mixing toward the foreground was the opposite direction in the light
	// theme (it darkened the card below the window), so the ladder now points to
	// white and the light theme keeps its cards on the window's white while the
	// dark theme lightens them.
	background, foreground := t.Background, t.Text
	// The sidebar is the reference's tinted chrome column, not an elevated face:
	// a gray step under the window in the light theme and the theme's own surface
	// step in the dark one.
	// The reference's sidebar is a tinted chrome column: a gray step below the
	// window in the light theme, the theme's surface step in the dark one.
	sidebarFace := t.SurfaceHover
	if t.Dark {
		sidebarFace = t.Surface
	}
	if sidebarFace == ui.Transparent {
		sidebarFace = t.Surface
	}
	if sidebarFace == ui.Transparent {
		sidebarFace = background
	}
	white := ui.Hex("#ffffff")
	sidebar := sidebarFace
	card := background.Mix(white, cardRise)
	popover := background.Mix(white, popoverRise)

	// The reference's primary is its brand green, not the theme's blue accent:
	// white ink on the light green and near-black moss ink on the dark one.
	primary := brandGreenLight
	onPrimary := ui.Hex("#ffffff")
	if t.Dark {
		primary = brandGreenDark
		onPrimary = ui.Hex("#07140a")
	}
	danger := t.Danger
	onDanger := readableOn(danger)

	// accent is a neutral face: SurfaceHover is what a row or a menu item tints
	// to under the pointer. The brand color is primary.
	accent := t.SurfaceHover
	if accent == ui.Transparent {
		accent = t.Surface
	}

	ring := t.Focus
	if ring == ui.Transparent {
		ring = primary
	}

	// The theme's inverse pair is the tooltip's base: dark on a light theme,
	// light on a dark one. A theme that leaves it zero gets the foreground and
	// background pair, the theme turned over, as mygo's own tooltip does.
	tooltip, tooltipForeground := t.Inverse, t.InverseText
	if tooltip == ui.Transparent {
		tooltip = foreground
	}
	if tooltipForeground == ui.Transparent {
		tooltipForeground = background
	}

	return tokens{
		theme:    t,
		hairline: 1,
		gap:      t.Space(0.5),
		gapWide:  t.Space(3),
		rowPadY:  t.Space(1),
		rowPadX:  rowPadXStep * geometryScale,
		// The reference conversation bar is 41 pt high with 30 pt entries; 5.5
		// above and below keeps the same height instead of the theme step's 8.
		// Compact density tightens that padding like every other room step, while
		// the bar's minimum height stays the reference's.
		toolbarPadY: 5.5 * geometryScale,
		headerH:     41,
		toolbarPadX: t.Space(3),
		edgePad:     t.Space(3),
		panelPad:    t.Space(6),
		composerH:   t.Space(18),
		statusDot:   t.Space(2),
		cardRadius:  radiusLG,
		cardPadY:    t.Space(1),
		cardPadX:    cardPadXStep * geometryScale,

		radiusSM: radiusSM,
		radiusMD: radiusMD,
		radiusLG: radiusLG,
		radiusXL: radiusXL,

		// The frame is a CSS literal, not a spacing step, so a compact density
		// leaves it as dim-electron draws it.
		framePad:    mainFramePad,
		frameRadius: mainFrameRadius,

		heroMarkW:      heroMarkW,
		heroMarkH:      heroMarkH,
		heroTitleSize:  fontSize * heroTitleStep,
		heroTitleLineH: fontSize * heroLineStep,
		heroGap:        heroGap * geometryScale,
		heroPad:        heroPad * geometryScale,

		timelinePadX:       timelinePadX * geometryScale,
		timelineBlockGap:   timelineBlockGap * geometryScale,
		timelineHeaderH:    timelineHeaderH,
		timelinePadStart:   timelinePadStart * geometryScale,
		timelineRail:       timelineRail,
		timelinePanelMaxH:  timelinePanelMaxH,
		timelineChevron:    timelineChevron,
		timelineUserWidth:  timelineUserWidth,
		timelineUserPadY:   timelineUserPadY * geometryScale,
		timelineUserPadX:   timelineUserPadX * geometryScale,
		timelineBubbleR:    timelineBubbleR,
		timelineFileRowY:   timelineFileRowY * geometryScale,
		timelineFileRowX:   timelineFileRowX * geometryScale,
		timelineFileGap:    timelineFileGap * geometryScale,
		timelineFileStatus: timelineFileStatus,

		sidebarHeaderH:     sidebarHeaderH,
		sidebarHeaderTop:   sidebarHeaderTop * geometryScale,
		sidebarSectionH:    sidebarSectionH,
		sidebarRowH:        sidebarRowH,
		sidebarEntryH:      sidebarEntryH,
		sidebarFooterH:     sidebarFooterH,
		sidebarCell:        sidebarCell,
		sidebarGap:         sidebarGap * geometryScale,
		sidebarPadX:        sidebarPad * geometryScale,
		sidebarSectionPadX: sidebarSectionPad * geometryScale,

		sidebarW:    sidebarDefaultW,
		sidebarMinW: sidebarMinW,
		sidebarMaxW: sidebarMaxW,

		uiFontSize:   fontSize,
		codeFontSize: defaultCodeFontSize,
		titleSize:    fontSize * typeTitleStep,
		bodySize:     fontSize,
		labelSize:    fontSize * typeLabelStep,
		smallSize:    fontSize * typeSmallStep,

		background:        background,
		foreground:        foreground,
		sidebar:           sidebar,
		card:              card,
		cardForeground:    foreground,
		popover:           popover,
		popoverForeground: foreground,

		sidebarForeground:        foreground,
		sidebarPrimary:           primary,
		sidebarPrimaryForeground: onPrimary,
		sidebarAccent:            sidebar.Mix(foreground, sidebarSelectionRise),
		sidebarAccentForeground:  foreground,
		sidebarHover:             sidebar.Mix(foreground, sidebarHoverRise),
		sidebarBorder:            t.Border.Mix(sidebar, 0.35),
		sidebarRing:              ring,

		primary:             primary,
		onPrimary:           onPrimary,
		secondary:           t.Surface,
		secondaryForeground: foreground,
		muted:               t.Surface,
		mutedForeground:     t.TextMuted,
		accent:              accent,
		accentForeground:    foreground,
		danger:              danger,
		onDanger:            onDanger,

		border:  t.Border,
		input:   t.Border,
		ring:    ring,
		success: t.Success,
		warning: t.Warning,

		tooltip:                tooltip,
		tooltipForeground:      tooltipForeground,
		tooltipMutedForeground: tooltipForeground.Mix(tooltip, tooltipMutedFade),
		tooltipBorder:          tooltip.Mix(tooltipForeground, tooltipBorderRise),
		tooltipChip:            tooltip.Mix(tooltipForeground, tooltipChipRise),
	}
}

// snapFontSize returns the font-size step nearest size, so the shell's body
// text always sits on one of the steps the appearance settings offer.
func snapFontSize(size float32) float32 {
	if size <= 0 {
		return defaultUIFontSize
	}
	best := fontSizeSteps[0]
	for _, step := range fontSizeSteps {
		if math.Abs(float64(step-size)) < math.Abs(float64(best-size)) {
			best = step
		}
	}
	return best
}

// readableOn is the text color legible on fill: near-white on a dark fill and
// near-black on a light one. It fills in an *-on token whose theme color is
// missing, so text on a filled control never disappears into it.
func readableOn(fill ui.Color) ui.Color {
	luminance := (0.2126*float32(fill.R) + 0.7152*float32(fill.G) + 0.0722*float32(fill.B)) / 255
	if luminance < 0.6 {
		return ui.RGB(0xfa, 0xfa, 0xfa)
	}
	return ui.RGB(0x18, 0x18, 0x1b)
}

// clampSidebarWidth keeps a width a resize asks for between the sidebar's
// narrowest and widest, so a drag can never hide the session list nor take the
// room the conversation needs.
func (k tokens) clampSidebarWidth(w float32) float32 {
	if w < k.sidebarMinW {
		return k.sidebarMinW
	}
	if w > k.sidebarMaxW {
		return k.sidebarMaxW
	}
	return w
}

// resolveSidebarWidth is the width the sidebar draws at: the default until a
// resize sets one, and always inside the sidebar's bounds.
func (k tokens) resolveSidebarWidth(w float32) float32 {
	if w <= 0 {
		return k.sidebarW
	}
	return k.clampSidebarWidth(w)
}

// space is n units of the theme's Spacing, for a view that needs a step this
// set does not name.
func (k tokens) space(n float32) float32 { return k.theme.Space(n) }

// wash tints a color as the background of a banner or a status area. Card and
// popover faces are not washes: they are steps of the surface ladder, so the
// text on them keeps its contrast.
func (k tokens) wash(color ui.Color) ui.Color { return color.Alpha(washAlpha) }

// runColor is the color of the run status row and its dot.
func (k tokens) runColor(state RunState) ui.Color {
	switch state {
	case RunRunning:
		return k.primary
	case RunStopping:
		return k.warning
	default:
		return k.mutedForeground
	}
}

// toolStatusColor is the color of a tool call's status label.
func (k tokens) toolStatusColor(status ToolStatus) ui.Color {
	switch status {
	case ToolRunning:
		return k.primary
	case ToolComplete:
		return k.success
	case ToolError:
		return k.danger
	default:
		return k.mutedForeground
	}
}

// connectionColor is the color of the status row and its dot.
func (k tokens) connectionColor(conn Connection) ui.Color {
	switch conn {
	case ConnReady:
		return k.success
	case ConnError:
		return k.danger
	case ConnConnecting:
		return k.primary
	default:
		return k.mutedForeground
	}
}
