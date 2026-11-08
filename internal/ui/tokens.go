package ui

import "github.com/egoist/mygo/ui"

// The shell's type steps, as fractions of the theme's font size: the shell's
// own scale on top of the theme's text size, so that a larger system text size
// grows every heading, label and row together.
const (
	typeTitleStep = 1.15
	typeLabelStep = 0.9
	typeSmallStep = 0.8
)

// washAlpha is how strongly the shell tints a color it uses as a background.
const washAlpha = 0.12

// tokens are the shell's design tokens: the room, type and color steps its
// views use, so that no view carries a magic number of its own. They come from
// the window's theme every frame, so a theme change, dark mode or a larger
// system text size carries through the whole shell.
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
	sidebarPadY float32
	sidebarW    float32
	composerH   float32
	statusDot   float32

	// Type sizes, in DIPs.
	titleSize float32
	bodySize  float32
	labelSize float32
	smallSize float32

	// Colors: the theme's, named for what the shell uses them for.
	muted, border, surface, accent, success, danger ui.Color
}

// newTokens builds the shell's tokens from the window's theme.
func newTokens(t *ui.Theme) tokens {
	fontSize := t.FontSize
	if fontSize <= 0 {
		fontSize = 14
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
		sidebarPadY: t.Space(2),
		sidebarW:    t.Space(55),
		composerH:   t.Space(18),
		statusDot:   t.Space(2),

		titleSize: fontSize * typeTitleStep,
		bodySize:  fontSize,
		labelSize: fontSize * typeLabelStep,
		smallSize: fontSize * typeSmallStep,

		muted:   t.TextMuted,
		border:  t.Border,
		surface: t.Surface,
		accent:  t.Accent,
		success: t.Success,
		danger:  t.Danger,
	}
}

// space is n units of the theme's Spacing, for a view that needs a step this
// set does not name.
func (k tokens) space(n float32) float32 { return k.theme.Space(n) }

// wash tints a color as the background of a banner or a status area.
func (k tokens) wash(color ui.Color) ui.Color { return color.Alpha(washAlpha) }

// connectionColor is the color of the status row and its dot.
func (k tokens) connectionColor(conn Connection) ui.Color {
	switch conn {
	case ConnReady:
		return k.success
	case ConnError:
		return k.danger
	case ConnConnecting:
		return k.accent
	default:
		return k.muted
	}
}
