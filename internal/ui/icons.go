// This file owns the sidebar's icon set: the inline 16×16 SVGs drawn before
// its global entries, its headings and its account row, as the Electron
// reference draws them (docs/evidence/electron-reference/window-workspace.png).
//
// The glyphs carry no color of their own. Each strokes currentColor and is
// shown with ui.Icon, which paints the shapes in the text color of the row
// beside it, so the light and dark themes, the WIP face's opacity and a
// selection all carry through without a color written down here.

package ui

import "github.com/egoist/mygo/ui"

// sidebarIconSize is the square, in DIPs, every sidebar glyph is drawn at. The
// inputs are 16×16 too, so the artwork is displayed at its own scale.
const sidebarIconSize = 16

// sidebarGlyphs is the sidebar's inline icon set. The SVGs are parsed once, at
// start-up, and the same value is drawn every frame.
type sidebarGlyphs struct {
	plus    *ui.SVG
	search  *ui.SVG
	clock   *ui.SVG
	grid    *ui.SVG
	folder  *ui.SVG
	user    *ui.SVG
	gear    *ui.SVG
	chevron *ui.SVG
	list    *ui.SVG
}

// sidebarIcon wraps the shapes of one 16×16 glyph in the common SVG frame:
// no fill, a light currentColor stroke and round joins, as the reference's
// outline icons are drawn.
func sidebarIcon(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

// sidebarIcons holds the glyphs in the order the sidebar uses them: the four
// global entries, the thread list, the project groups, the account row and the
// settings entry. The chevron marks the keys entry's disclosure, whose panel
// opens in the main column.
var sidebarIcons = sidebarGlyphs{
	plus:    sidebarIcon(`<path d="M8 3v10M3 8h10"/>`),
	search:  sidebarIcon(`<circle cx="7" cy="7" r="4.1"/><path d="M10.1 10.1 13.5 13.5"/>`),
	clock:   sidebarIcon(`<circle cx="8" cy="8" r="5.6"/><path d="M8 4.6V8l2.4 1.5"/>`),
	grid:    sidebarIcon(`<rect x="2.2" y="2.2" width="4.7" height="4.7" rx="1"/><rect x="9.1" y="2.2" width="4.7" height="4.7" rx="1"/><rect x="2.2" y="9.1" width="4.7" height="4.7" rx="1"/><rect x="9.1" y="9.1" width="4.7" height="4.7" rx="1"/>`),
	folder:  sidebarIcon(`<path d="M2 4.5A1.5 1.5 0 0 1 3.5 3h2.6l1.5 2h4.9A1.5 1.5 0 0 1 14 6.5v5A1.5 1.5 0 0 1 12.5 13h-9A1.5 1.5 0 0 1 2 11.5Z"/>`),
	user:    sidebarIcon(`<circle cx="8" cy="5.2" r="2.4"/><path d="M3.5 13c.4-2.6 2.2-4 4.5-4s4.1 1.4 4.5 4"/>`),
	gear:    sidebarIcon(`<path d="M6.8 1.5h2.4l.35 1.9a5 5 0 0 1 1.2.7l1.85-.6 1.2 2.1-1.45 1.3a5 5 0 0 1 0 1.4l1.45 1.3-1.2 2.1-1.85-.6a5 5 0 0 1-1.2.7l-.35 1.9H6.8l-.35-1.9a5 5 0 0 1-1.2-.7l-1.85.6-1.2-2.1 1.45-1.3a5 5 0 0 1 0-1.4L2.2 5.6l1.2-2.1 1.85.6a5 5 0 0 1 1.2-.7Z"/><circle cx="8" cy="8" r="1.9"/>`),
	chevron: sidebarIcon(`<path d="M6 3.5 10.5 8 6 12.5"/>`),
	list:    sidebarIcon(`<path d="M5.5 4h8M5.5 8h8M5.5 12h8"/><circle cx="2.9" cy="4" r=".7" fill="currentColor" stroke="none"/><circle cx="2.9" cy="8" r=".7" fill="currentColor" stroke="none"/><circle cx="2.9" cy="12" r=".7" fill="currentColor" stroke="none"/>`),
}

// sidebarGlyph draws one icon of the set at the sidebar's 16 DIP step, in
// color. The label or heading beside it already names the entry, so the glyph
// is a decoration and assistive technology skips it instead of reading it
// twice.
func sidebarGlyph(c *ui.Context, icon *ui.SVG, color ui.Color) ui.Element {
	return ui.Icon(c, icon).Size(sidebarIconSize, sidebarIconSize).
		TextColor(color).AccessHidden(true)
}
