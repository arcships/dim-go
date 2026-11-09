// This file owns the shell's icon vocabulary: the glyphs the sidebar, the
// settings navigation, the header and the composer draw. The artwork is
// Lucide v0.577.0, the icon set the Electron reference uses, parsed from
// lucide_icons.go. Brand marks (the DimAgent mark and the VS Code mark) stay
// their own assets; they are logos, not interface icons.
//
// The glyphs carry no color of their own. Each strokes currentColor and is
// shown with ui.Icon, which paints the shapes in the text color of the row
// beside it, so the light and dark themes, the WIP face's opacity and a
// selection all carry through without a color written down here.

package ui

import "github.com/egoist/mygo/ui"

// sidebarIconSize is the square, in DIPs, every interface glyph is drawn at.
// Lucide's 24×24 artwork is scaled into this box, so the whole shell shares one
// stroke weight and no glyph needs a separate asset.
const sidebarIconSize = 16

// mustLucide returns one glyph of the Lucide set by its package name. The
// lookup is local and the key is a constant beside the icon it names, so a
// missing glyph is a programming error: panic at start-up instead of drawing a
// blank row.
func mustLucide(name string) *ui.SVG {
	icon, ok := lucideIcons[name]
	if !ok {
		panic("ui: missing lucide icon " + name)
	}
	return icon
}

// sidebarGlyphs is the sidebar's inline icon set. The SVGs are parsed once, at
// start-up, and the same value is drawn every frame.
type sidebarGlyphs struct {
	plus     *ui.SVG
	search   *ui.SVG
	clock    *ui.SVG
	grid     *ui.SVG
	folder   *ui.SVG
	pin      *ui.SVG
	house    *ui.SVG
	user     *ui.SVG
	gear     *ui.SVG
	chevron  *ui.SVG
	list     *ui.SVG
	collapse *ui.SVG
}

// sidebarIcons holds the glyphs in the order the sidebar uses them: the four
// global entries, the thread list, the workspace heading, the project groups,
// a pinned session row's tail, the account row and the settings entry. The
// chevron marks the keys entry's disclosure, whose panel opens in the main
// column.
var sidebarIcons = sidebarGlyphs{
	plus:    mustLucide("plus"),
	search:  mustLucide("search"),
	clock:   mustLucide("clock"),
	grid:    mustLucide("layout-grid"),
	folder:  mustLucide("folder"),
	pin:     mustLucide("pin"),
	house:   mustLucide("house"),
	user:    mustLucide("user"),
	gear:    mustLucide("settings"),
	chevron: mustLucide("chevron-right"),
	list:    mustLucide("list"),
	// collapse is the reference's panel-collapse mark at the sidebar's top
	// right: the rounded frame with the arrow in the wide pane pointing back
	// over the narrow one, as Lucide's panel-left-close draws it.
	collapse: mustLucide("panel-left-close"),
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
// 16×16 Lucide glyph per section the navigation lists. They are drawn in the
// same frame as the sidebar's, so the two navigations read as one shell. A
// section with no product concept of its own, such as the shell's capability
// list, gets a generic shape rather than a borrowed meaning.
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
// keyboard, a globe, adjustment sliders, a terminal, a plug, a bell, a paw, a
// shield, a branch, a webhook, a puzzle and a bot. The capability page has no
// reference entry, so it draws the generic layers mark.
var settingsIcons = settingsGlyphs{
	back:            mustLucide("arrow-left"),
	sun:             mustLucide("sun"),
	moon:            mustLucide("moon"),
	system:          mustLucide("monitor"),
	account:         mustLucide("user"),
	usage:           mustLucide("chart-column"),
	personalization: mustLucide("sparkles"),
	appearance:      mustLucide("palette"),
	general:         mustLucide("sliders-horizontal"),
	shortcuts:       mustLucide("keyboard"),
	models:          mustLucide("globe"),
	multimodal:      mustLucide("sliders-vertical"),
	runtime:         mustLucide("terminal"),
	mcp:             mustLucide("plug"),
	notifications:   mustLucide("bell"),
	pet:             mustLucide("paw-print"),
	permissions:     mustLucide("shield-check"),
	git:             mustLucide("git-branch"),
	hooks:           mustLucide("webhook"),
	plugins:         mustLucide("puzzle"),
	externalAgent:   mustLucide("bot"),
	capabilities:    mustLucide("layers"),
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
// it is parsed separately from Lucide's outline frame; the paint comes from
// currentColor, which the entry sets to the brand blue.
var openInEditorGlyph = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path fill="currentColor" d="M23.15 2.587 18.21.21a1.494 1.494 0 0 0-1.705.29l-9.46 8.63-4.12-3.128a.999.999 0 0 0-1.276.057L.327 7.261A1 1 0 0 0 .326 8.74L3.899 12 .326 15.26a1 1 0 0 0 .001 1.479L1.65 17.94a.999.999 0 0 0 1.276.057l4.12-3.128 9.46 8.63a1.492 1.492 0 0 0 1.704.29l4.942-2.377A1.5 1.5 0 0 0 24 20.06V3.939a1.5 1.5 0 0 0-.85-1.352zm-5.146 14.861L10.826 12l7.178-5.448v10.896z"/></svg>`))

// openInEditorBrand is the blue the reference draws the VS Code mark in on its
// editor-open control. It is the VS Code brand fill, not a theme color, so the
// mark keeps its hue in both appearances instead of turning into the muted
// foreground of the text beside it.
var openInEditorBrand = ui.Hex("#2563EB")
