package ui

import (
	"math"

	"github.com/egoist/mygo/ui"
)

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
func newTokensFor(t *ui.Theme, prefs SettingsView) tokens {
	t = themeOrLight(t)
	v := resolveSettings(newTokens(t), prefs)
	k := buildTokens(appearanceTheme(t, v), densityScale(v.Density))
	// The UI steps already follow the appearance theme's size; the code size
	// is not a step of that scale, so it is folded in here.
	k.codeFontSize = v.CodeFontSize
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
func appearanceTheme(t *ui.Theme, v SettingsView) *ui.Theme {
	applied := *t
	applied.FontSize = v.FontSize
	applied.Font = fontFamilyStack(v.FontFamily)
	if v.Density == DensityCompact {
		applied.Spacing = spacingUnit(t) * densityCompactScale
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
