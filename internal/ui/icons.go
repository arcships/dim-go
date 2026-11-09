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
	plus     *ui.SVG
	search   *ui.SVG
	clock    *ui.SVG
	grid     *ui.SVG
	folder   *ui.SVG
	house    *ui.SVG
	user     *ui.SVG
	gear     *ui.SVG
	chevron  *ui.SVG
	list     *ui.SVG
	collapse *ui.SVG
}

// sidebarIcon wraps the shapes of one 16×16 glyph in the common SVG frame:
// no fill, a light currentColor stroke and round joins, as the reference's
// outline icons are drawn.
func sidebarIcon(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

// sidebarIcons holds the glyphs in the order the sidebar uses them: the four
// global entries, the thread list, the workspace heading, the project groups,
// the account row and the settings entry. The chevron marks the keys entry's
// disclosure, whose panel opens in the main column.
var sidebarIcons = sidebarGlyphs{
	plus:    sidebarIcon(`<path d="M8 3v10M3 8h10"/>`),
	search:  sidebarIcon(`<circle cx="7" cy="7" r="4.1"/><path d="M10.1 10.1 13.5 13.5"/>`),
	clock:   sidebarIcon(`<circle cx="8" cy="8" r="5.6"/><path d="M8 4.6V8l2.4 1.5"/>`),
	grid:    sidebarIcon(`<rect x="2.2" y="2.2" width="4.7" height="4.7" rx="1"/><rect x="9.1" y="2.2" width="4.7" height="4.7" rx="1"/><rect x="2.2" y="9.1" width="4.7" height="4.7" rx="1"/><rect x="9.1" y="9.1" width="4.7" height="4.7" rx="1"/>`),
	folder:  sidebarIcon(`<path d="M2 4.5A1.5 1.5 0 0 1 3.5 3h2.6l1.5 2h4.9A1.5 1.5 0 0 1 14 6.5v5A1.5 1.5 0 0 1 12.5 13h-9A1.5 1.5 0 0 1 2 11.5Z"/>`),
	house:   sidebarIcon(`<path d="M2.4 7.1 8 2.6l5.6 4.5"/><path d="M3.9 6.2v5.5c0 .7.6 1.3 1.3 1.3h5.6c.7 0 1.3-.6 1.3-1.3V6.2"/><path d="M6.5 13V9.5h3V13"/>`),
	user:    sidebarIcon(`<circle cx="8" cy="5.2" r="2.4"/><path d="M3.5 13c.4-2.6 2.2-4 4.5-4s4.1 1.4 4.5 4"/>`),
	gear:    sidebarIcon(`<path d="M6.8 1.5h2.4l.35 1.9a5 5 0 0 1 1.2.7l1.85-.6 1.2 2.1-1.45 1.3a5 5 0 0 1 0 1.4l1.45 1.3-1.2 2.1-1.85-.6a5 5 0 0 1-1.2.7l-.35 1.9H6.8l-.35-1.9a5 5 0 0 1-1.2-.7l-1.85.6-1.2-2.1 1.45-1.3a5 5 0 0 1 0-1.4L2.2 5.6l1.2-2.1 1.85.6a5 5 0 0 1 1.2-.7Z"/><circle cx="8" cy="8" r="1.9"/>`),
	chevron: sidebarIcon(`<path d="M6 3.5 10.5 8 6 12.5"/>`),
	list:    sidebarIcon(`<path d="M5.5 4h8M5.5 8h8M5.5 12h8"/><circle cx="2.9" cy="4" r=".7" fill="currentColor" stroke="none"/><circle cx="2.9" cy="8" r=".7" fill="currentColor" stroke="none"/><circle cx="2.9" cy="12" r=".7" fill="currentColor" stroke="none"/>`),
	// collapse is the reference's panel-collapse mark at the sidebar's top
	// right: a 10×10 rounded frame split 3.5 DIPs from its left edge, with the
	// arrow in the wide pane pointing back over the narrow one, as lucide's
	// panel-left-close draws it.
	collapse: sidebarIcon(`<rect x="3" y="3" width="10" height="10" rx="1.7"/><path d="M6.4 3v10"/><path d="M10.4 5.7 8 8l2.4 2.3"/>`),
}

// sidebarGlyph draws one icon of the set at the sidebar's 16 DIP step, in
// color. The label or heading beside it already names the entry, so the glyph
// is a decoration and assistive technology skips it instead of reading it
// twice.
func sidebarGlyph(c *ui.Context, icon *ui.SVG, color ui.Color) ui.Element {
	return ui.Icon(c, icon).Size(sidebarIconSize, sidebarIconSize).
		TextColor(color).AccessHidden(true)
}

// settingsGlyphs is the settings route's own icon set: the back arrow and one
// 16×16 outline glyph per section the navigation lists. They are drawn in the
// same frame as the sidebar's (a light currentColor stroke with round joins) so
// the two navigations read as one shell. A section with no product concept of
// its own, such as the shell's capability list, gets a generic shape rather than
// a borrowed meaning.
type settingsGlyphs struct {
	back            *ui.SVG
	sun             *ui.SVG
	moon            *ui.SVG
	system          *ui.SVG
	account         *ui.SVG
	usage           *ui.SVG
	personalization *ui.SVG
	appearance      *ui.SVG
	general         *ui.SVG
	shortcuts       *ui.SVG
	models          *ui.SVG
	multimodal      *ui.SVG
	runtime         *ui.SVG
	mcp             *ui.SVG
	notifications   *ui.SVG
	pet             *ui.SVG
	permissions     *ui.SVG
	git             *ui.SVG
	hooks           *ui.SVG
	plugins         *ui.SVG
	externalAgent   *ui.SVG
	capabilities    *ui.SVG
}

// settingsIcons holds the settings route's glyphs. The section shapes follow
// dim-electron's navigation (docs/evidence/electron-reference/
// window-settings.png): a user, a bar chart, sparkles, a palette, sliders, a
// keyboard, a globe, adjustment sliders, a terminal panel, a chain link, a bell,
// a paw, a shield, a branch, a hook, a plug and a briefcase. The capability page
// has no reference entry, so it draws the generic layers mark.
var settingsIcons = settingsGlyphs{
	back:            sidebarIcon(`<path d="M13 8H3"/><path d="M7 4 3 8l4 4"/>`),
	sun:             sidebarIcon(`<circle cx="8" cy="8" r="3"/><path d="M8 1.5v1.6M8 12.9v1.6M1.5 8h1.6M12.9 8h1.6M3.4 3.4l1.1 1.1M11.5 11.5l1.1 1.1M3.4 12.6l1.1-1.1M11.5 4.5l1.1-1.1"/>`),
	moon:            sidebarIcon(`<path d="M12.8 9.7A5.3 5.3 0 0 1 6.3 3.2 5.7 5.7 0 1 0 12.8 9.7Z"/>`),
	system:          sidebarIcon(`<rect x="2" y="3" width="12" height="8.2" rx="1.4"/><path d="M5.5 13.4h5M8 11.2v2.2"/>`),
	account:         sidebarIcon(`<circle cx="8" cy="5.2" r="2.4"/><path d="M3.5 13c.4-2.6 2.2-4 4.5-4s4.1 1.4 4.5 4"/>`),
	usage:           sidebarIcon(`<path d="M2.5 13.5h11"/><path d="M4.5 13.5V9M8 13.5V3M11.5 13.5V6"/>`),
	personalization: sidebarIcon(`<path d="M8 2.3 9.2 5.6 12.5 6.8 9.2 8 8 11.3 6.8 8 3.5 6.8 6.8 5.6Z"/><path d="M12.9 10.9l.4 1.1 1.1.4-1.1.4-.4 1.1-.4-1.1-1.1-.4 1.1-.4Z"/>`),
	appearance:      sidebarIcon(`<path d="M8 2.3a5.7 5.7 0 1 0 0 11.4c.9 0 1.4-.6 1.4-1.3 0-.5-.2-.8-.2-1.2 0-.7.5-1.2 1.2-1.2h1.1c1.2 0 2.1-.9 2.1-2.1A5.6 5.6 0 0 0 8 2.3Z"/><circle cx="5.4" cy="6.2" r=".6" fill="currentColor" stroke="none"/><circle cx="8" cy="4.8" r=".6" fill="currentColor" stroke="none"/><circle cx="10.6" cy="6.2" r=".6" fill="currentColor" stroke="none"/>`),
	general:         sidebarIcon(`<path d="M2.2 4.5h4.3M9.5 4.5h4.3M2.2 11.5h2.3M7.5 11.5h6.3"/><circle cx="8" cy="4.5" r="1.6"/><circle cx="5.8" cy="11.5" r="1.6"/>`),
	shortcuts:       sidebarIcon(`<rect x="2" y="3.8" width="12" height="8.4" rx="1.6"/><path d="M4.4 6.3h.01M6.7 6.3h.01M9 6.3h.01M11.3 6.3h.01M4.4 9.7h.01M11.3 9.7h.01M6.7 9.7h2.6"/>`),
	models:          sidebarIcon(`<circle cx="8" cy="8" r="5.7"/><path d="M2.5 6.2h11M2.5 9.8h11"/><path d="M8 2.3c1.5 1.6 2.3 3.5 2.3 5.7S9.5 12.1 8 13.7C6.5 12.1 5.7 10.2 5.7 8S6.5 3.9 8 2.3Z"/>`),
	multimodal:      sidebarIcon(`<path d="M4 2.2v3.6M4 9.4v4.4M8 2.2v7.2M8 12.8v1M12 2.2v1.6M12 7.4v6.4"/><circle cx="4" cy="7.6" r="1.6"/><circle cx="8" cy="11" r="1.6"/><circle cx="12" cy="5.4" r="1.6"/>`),
	runtime:         sidebarIcon(`<rect x="2" y="2.6" width="12" height="10.8" rx="1.6"/><path d="M4.8 5.4 6.3 6.9 4.8 8.4M7.9 9h3.3"/>`),
	mcp:             sidebarIcon(`<path d="M6.4 9.6 9.6 6.4"/><path d="M7.4 4.4 8.9 2.9a2.5 2.5 0 0 1 3.6 3.6L10.9 8"/><path d="M8.6 11.6 7.1 13.1a2.5 2.5 0 0 1-3.6-3.6L5.1 8"/>`),
	notifications:   sidebarIcon(`<path d="M4 6.7a4 4 0 0 1 8 0c0 2.9.7 3.9 1.4 4.5H2.6c.7-.6 1.4-1.6 1.4-4.5Z"/><path d="M6.4 13a1.7 1.7 0 0 0 3.2 0"/>`),
	pet:             sidebarIcon(`<circle cx="4.8" cy="5.4" r="1.4"/><circle cx="11.2" cy="5.4" r="1.4"/><circle cx="2.2" cy="8.8" r="1.1"/><circle cx="13.8" cy="8.8" r="1.1"/><path d="M8 8.2c-1.9 0-3.4 1.4-3.4 3 0 1 .8 1.6 1.9 1.6.6 0 1.1-.3 1.5-.3s.9.3 1.5.3c1.1 0 1.9-.6 1.9-1.6 0-1.6-1.5-3-3.4-3Z"/>`),
	permissions:     sidebarIcon(`<path d="M8 2 13 4v3.6c0 3-2 5.6-5 6.4-3-.8-5-3.4-5-6.4V4Z"/><path d="m5.8 8 1.6 1.6L10.4 6.5"/>`),
	git:             sidebarIcon(`<circle cx="4.4" cy="3.5" r="1.5"/><circle cx="4.4" cy="12.5" r="1.5"/><circle cx="11.6" cy="3.5" r="1.5"/><path d="M4.4 5v6M11.6 5v1.4c0 2-1.5 3.1-3.5 3.1H6"/>`),
	hooks:           sidebarIcon(`<path d="M8 2.2v5.6"/><path d="M5.2 5.3 8 8l2.8-2.7"/><circle cx="8" cy="11.3" r="2.5"/>`),
	plugins:         sidebarIcon(`<path d="M5.2 2.2v2.9M10.8 2.2v2.9"/><path d="M3.4 5.1h9.2v2.4a4.6 4.6 0 0 1-9.2 0Z"/><path d="M8 11.7v2.1"/>`),
	externalAgent:   sidebarIcon(`<rect x="2" y="5" width="12" height="8" rx="1.5"/><path d="M5.6 5V3.6A1.6 1.6 0 0 1 7.2 2h1.6a1.6 1.6 0 0 1 1.6 1.6V5M2 8.6h12"/>`),
	capabilities:    sidebarIcon(`<path d="m8 2 6 3-6 3-6-3Z"/><path d="m2 8 6 3 6-3M2 11l6 3 6-3"/>`),
}

// settingsIcon is the glyph a navigation row draws for its section. Every
// section has one; a value outside the enum falls back to the generic layers
// mark rather than drawing nothing.
func settingsIcon(s SettingsSection) *ui.SVG {
	switch s {
	case SettingsSectionAccount:
		return settingsIcons.account
	case SettingsSectionUsage:
		return settingsIcons.usage
	case SettingsSectionPersonalization:
		return settingsIcons.personalization
	case SettingsSectionAppearance:
		return settingsIcons.appearance
	case SettingsSectionGeneral:
		return settingsIcons.general
	case SettingsSectionShortcuts:
		return settingsIcons.shortcuts
	case SettingsSectionModels:
		return settingsIcons.models
	case SettingsSectionMultimodal:
		return settingsIcons.multimodal
	case SettingsSectionRuntime:
		return settingsIcons.runtime
	case SettingsSectionMCP:
		return settingsIcons.mcp
	case SettingsSectionNotifications:
		return settingsIcons.notifications
	case SettingsSectionPet:
		return settingsIcons.pet
	case SettingsSectionPermissions:
		return settingsIcons.permissions
	case SettingsSectionGit:
		return settingsIcons.git
	case SettingsSectionHooks:
		return settingsIcons.hooks
	case SettingsSectionPlugins:
		return settingsIcons.plugins
	case SettingsSectionExternalAgent:
		return settingsIcons.externalAgent
	default:
		return settingsIcons.capabilities
	}
}

// openInEditorGlyph is the Visual Studio Code mark the reference puts on its
// editor-open control. Its 24×24 brand path is filled rather than stroked, so
// it is parsed separately from sidebarIcon's outline frame; the paint comes
// from currentColor, which the entry sets to the brand blue.
var openInEditorGlyph = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="currentColor" d="M23.15 2.587 18.21.21a1.494 1.494 0 0 0-1.705.29l-9.46 8.63-4.12-3.128a.999.999 0 0 0-1.276.057L.327 7.261A1 1 0 0 0 .326 8.74L3.899 12 .326 15.26a1 1 0 0 0 .001 1.479L1.65 17.94a.999.999 0 0 0 1.276.057l4.12-3.128 9.46 8.63a1.492 1.492 0 0 0 1.704.29l4.942-2.377A1.5 1.5 0 0 0 24 20.06V3.939a1.5 1.5 0 0 0-.85-1.352zm-5.146 14.861L10.826 12l7.178-5.448v10.896z"/></svg>`))

// openInEditorBrand is the blue the reference draws the VS Code mark in on its
// editor-open control. It is the VS Code brand fill, not a theme color, so the
// mark keeps its hue in both appearances instead of turning into the muted
// foreground of the text beside it.
var openInEditorBrand = ui.Hex("#2563EB")
