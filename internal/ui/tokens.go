package ui

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// The shell's type steps, as fractions of the theme's font size: the shell's
// own scale on top of the theme's text size, so that a larger system text size
// grows every heading, label and row together.
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

// The radius ladder, as dim-electron's --radius-{sm,md,lg,xl}: an inline
// control's corner, a field's or a chip's, a card's, and a floating surface's,
// in DIPs.
const (
	radiusSM = 6
	radiusMD = 8
	radiusLG = 10
	radiusXL = 14
)

// The sidebar's width in DIPs: the width it opens at, and the narrowest and
// widest a resize may leave it.
const (
	sidebarDefaultW = 256
	sidebarMinW     = 200
	sidebarMaxW     = 360
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

// The surface ladder: how far each face lifts off the background, as the share
// of the way from the background to the foreground. A face is a mix of two
// theme colors, not a wash of the border, so an elevated surface keeps a tint
// of its own in both themes instead of reading as the border at 12%: the
// background is the window, the sidebar and the cards step out of it, and a
// popover sits on top of them.
const (
	sidebarRise = 0.03
	cardRise    = 0.06
	popoverRise = 0.10
)

// The tooltip's own steps: its edge and its chips lift off its face, and its
// secondary text fades toward it. The face itself is the theme's inverse pair,
// the theme turned over so a tip stands out from whatever it covers.
const (
	tooltipBorderRise = 0.18
	tooltipChipRise   = 0.10
	tooltipMutedFade  = 0.30
)

// washAlpha is how strongly the shell tints a color it uses as the background
// of a banner or a status area. Faces are mixed (see the surface ladder), not
// washed: a tint over the border has no contrast of its own.
const washAlpha = 0.12

// tokens are the shell's design tokens: the room, type, radius and color steps
// its views use, so that no view carries a magic number or a color of its own.
// They come from the window's theme every frame, so a theme change, dark mode
// or a larger system text size carries through the whole shell.
//
// The colors are the semantic layer dim-electron names in its stylesheet,
// derived here from ui.Theme: a view asks for what a surface is (a card, a
// muted face, the text on the primary fill), never for a theme field, so the
// same view keeps its meaning when the theme's own palette changes.
type tokens struct {
	theme *ui.Theme

	// Room, in DIPs, on the theme's Spacing unit.
	hairline    float32 // a divider's width
	gap         float32 // between the parts of a row
	gapWide     float32 // between the parts of a toolbar row
	rowPadY     float32 // above and below a conversation row
	rowPadX     float32 // left and right of a conversation row
	toolbarPadY float32 // above and below the toolbar
	toolbarPadX float32 // left and right of the toolbar
	edgePad     float32 // inside the composer
	panelPad    float32 // inside the empty state and the loading row
	composerH   float32
	statusDot   float32
	cardRadius  float32 // a card's corner, the middle of the radius ladder
	cardPadY    float32 // above and below a card's parts
	cardPadX    float32 // left and right inside a card

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

// newTokens builds the shell's tokens from the window's theme.
func newTokens(t *ui.Theme) tokens {
	if t == nil {
		t = ui.LightTheme()
	}
	fontSize := snapFontSize(t.FontSize)

	// The elevation steps mix the background toward the foreground, so the
	// ladder points the same way in both themes: a card is a step away from
	// the window, a popover another one on top of it.
	background, foreground := t.Background, t.Text
	sidebar := background.Mix(foreground, sidebarRise)
	card := background.Mix(foreground, cardRise)
	popover := background.Mix(foreground, popoverRise)

	primary := t.Accent
	onPrimary := t.AccentText
	if onPrimary == ui.Transparent {
		onPrimary = readableOn(primary)
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
		theme:       t,
		hairline:    1,
		gap:         t.Space(0.5),
		gapWide:     t.Space(3),
		rowPadY:     t.Space(1),
		rowPadX:     t.Space(4),
		toolbarPadY: t.Space(2),
		toolbarPadX: t.Space(3),
		edgePad:     t.Space(3),
		panelPad:    t.Space(6),
		composerH:   t.Space(18),
		statusDot:   t.Space(2),
		cardRadius:  radiusLG,
		cardPadY:    t.Space(1),
		cardPadX:    t.Space(2),

		sidebarHeaderH:     sidebarHeaderH,
		sidebarHeaderTop:   sidebarHeaderTop,
		sidebarSectionH:    sidebarSectionH,
		sidebarRowH:        sidebarRowH,
		sidebarEntryH:      sidebarEntryH,
		sidebarFooterH:     sidebarFooterH,
		sidebarCell:        sidebarCell,
		sidebarGap:         sidebarGap,
		sidebarPadX:        sidebarPad,
		sidebarSectionPadX: sidebarSectionPad,

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
		sidebarAccent:            accent,
		sidebarAccentForeground:  foreground,
		sidebarHover:             accent,
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

// runColorFor a shell state: a terminal closed session has no active run to
// color, even if the last state notice said running.
func (k tokens) runColorFor(s *State) ui.Color {
	if s.SessionClosed {
		return k.mutedForeground
	}
	return k.runColor(s.Run)
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
