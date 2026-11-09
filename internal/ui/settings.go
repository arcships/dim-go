// This file owns the settings surface: the section navigation dim-electron
// draws on the left, and the page of the chosen section on the right. The
// appearance page keeps the desktop's own color theme, type, sidebar-width and
// density preferences, plus the skin and chrome entries the shell cannot offer
// yet. The archived-sessions page lists the rows this desktop hides and offers
// Restore. Every other section keeps its place in the navigation and explains
// the client or page it waits on, so the settings surface reads as WIP instead
// of as an empty page that saved nothing.
//
// It reads SettingsView and the tokens only (see protocols.go); the App Server
// owns every session, run, timeline and approval fact and is not read here.
//
// These are desktop-local preferences, the kind dim-electron keeps in its own
// config (appearance/index.ts): they change how the shell draws, not what a
// session contains. renderSettings writes nothing itself. It reports one
// SettingsChange per user choice and leaves applying and persisting it to the
// window, so the surface cannot claim a preference the desktop did not save.
// The chosen section is display state of this surface alone: a navigation click
// moves the surface's own *Section and reports no SettingsChange.

package ui

import (
	"math"
	"runtime"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// The settings surface's own steps, in DIPs. Padding, gaps, type sizes, radii
// and colors come from the tokens; these are the values only this surface has:
// the width of its reading column (dim-electron's max-w-4xl is 784 DIP at the
// 14px root size), the code size range dim-electron's slider offers
// (AppearanceSettings.vue, 12–20), and the width of a select so the choice
// column does not jump between rows.
const (
	settingsPageMaxW = 784
	settingsCodeMin  = 12
	settingsCodeMax  = 20
	settingsCodeStep = 1
	settingsSelectW  = 200

	// The theme cards, code previews and skin covers keep the reference's
	// proportions: a theme card is a wide, short tile, and the two galleries are
	// rows of small cards.
	settingsThemeCardH   = 64
	settingsThemeDot     = 7
	settingsThemeIconGap = 6
	settingsPreviewH     = 64
	settingsPreviewGap   = 4
	settingsSkinCardH    = 56
)

// The navigation column's geometry, in DIPs, from
// docs/evidence/electron-reference/window-settings.png: the reference's card
// starts 4.5 DIPs inside an 800-wide window, its section column's divider sits
// at x≈228, a row is about 34.5 high, and a selected row's pill is inset 7 DIPs
// from the card edge. The share keeps the column inside a narrow window instead
// of pushing the page off its right edge, and the cap gives the page its
// reading room on a wide one. A section row keeps the sidebar's row height so
// the two navigations read as one shell.
const (
	settingsNavW       = 228
	settingsNavShare   = 30
	settingsNavPadX    = 7
	settingsNavPadY    = 14
	settingsNavIconGap = 6
	settingsNavBackGap = 14
	settingsPagePad    = 28
)

// settingsCompactW is the width below which the route stops drawing the
// section column beside the page. The reference always has the two columns
// because its window is 800 wide; a narrower window cannot give the page its
// reading column beside a 228-wide navigation, so the route shows one level at
// a time instead: the section list, or the chosen page with a return row above
// it. The share above still keeps the column inside a mid-sized window, so the
// two levels only take over when the side-by-side layout stops fitting.
const settingsCompactW = 560

// settingsNavigationLabel names the section column for assistive technology.
// It stays the English fallback for a table that does not carry the key.
// The reference has no message key for it yet, so the label stays English;
// see the missing-key list in the surface's report.
const settingsNavigationLabel = "Settings sections"

// settingsRouteLocalKey is the local key the surface keeps its chosen section
// and level under (ui.Local). It is a type, not a string, so the state cannot
// collide with another element's.
type settingsRouteLocalKey struct{}

// settingsRouteState is the surface's own state between frames: the section
// the reader chose, and whether a narrow window shows that section's page.
// Neither field is a preference: a click moves this state and reports no
// SettingsChange. A wide window draws both levels at once, so it ignores
// detail; a narrow one starts on the list, opens the page on a row click, and
// returns to the list from the page's own return row.
type settingsRouteState struct {
	section SettingsSection
	detail  bool
}

// settingsSectionsBack is the compact layout's return from a section page to
// the section list. i18n.go has no key for it yet, so it stays English; see
// the missing-key list in the surface's report.
const settingsSectionsBack = "All settings"

// keySettingsSectionsBack is the message key the compact return row looks up.
// The English fallback above stands in until i18n.go carries it.
const keySettingsSectionsBack = "settings.backToList"

// settingsSectionsBackLabel is the compact return row's text in the active
// language, falling back to the English name rather than showing a key.
func settingsSectionsBackLabel(l Locale) string {
	if text := tr(l, keySettingsSectionsBack); text != keySettingsSectionsBack {
		return text
	}
	return settingsSectionsBack
}

// settingsLocalNote says where these preferences live. The surface never sends
// them to the App Server: none of them is a session, run or approval fact.
const settingsLocalNote = "These preferences live on this desktop. They change how the shell draws, not any session, run or approval."

// settingsSkinNote says why the skin section has no control to offer. A skin
// or a chrome theme replaces the semantic tokens themselves; system, light and
// dark are color preferences, not stand-ins for one.
const settingsSkinNote = "Skins are not implemented: the shell draws one semantic token set in system, light or dark. No fake skin is offered."

// The reasons the skin and chrome entries wait on. dim-electron resolves skins
// from a skin.json and a window region, and chrome from six historical
// stylesheets (styles/retro-themes.css); this shell has neither, so the entries
// stay visible, unavailable and explained instead of pretending a theme.
const (
	reasonSkin   = "Unavailable: skins need dim-electron's skin.json resolver and window regions; the theme row chooses only system, light or dark."
	reasonChrome = "Unavailable: chrome themes need dim-electron's six historical stylesheets; this shell does not draw a fake chrome."
)

// settingsSectionGaps says why a section without a page of its own has no
// page here: the client or page dim-electron opens for it is missing. The
// reason names what is absent, so a reader sees a WIP page rather than an empty
// one whose settings appear lost or unset. The sections this shell does draw —
// Appearance, Models, Shortcuts, Archived sessions and About — have no entry.
var settingsSectionGaps = map[SettingsSection]string{
	SettingsSectionAccount:         "Unavailable: the shell has no account client or page yet, so it cannot show an account or sign in.",
	SettingsSectionUsage:           "Unavailable: the shell has no usage client or page yet; no App Server method reports usage here.",
	SettingsSectionPersonalization: "Unavailable: the personalization page is not built yet; the shell has no client for the interface-mode, ACP-session, target-app, assistant-model or subagent-model settings it shows.",
	SettingsSectionGeneral:         "Unavailable: the general settings page is not built yet; the shell has no client for its desktop preferences.",
	SettingsSectionMultimodal:      "Unavailable: the multimodal page is not built yet; the shell has no multimodal client or page.",
	SettingsSectionRuntime:         "Unavailable: the runtime page is not built yet; the shell has no runtime client or page.",
	SettingsSectionMCP:             "Unavailable: the MCP page is not built yet; the shell has no MCP server client or page.",
	SettingsSectionNotifications:   "Unavailable: the notifications page is not built yet; the shell has no notification client or page.",
	SettingsSectionPet:             "Unavailable: the pet page is not built yet; the shell has no pet client or page.",
	SettingsSectionPermissions:     "Unavailable: the permissions page is not built yet; the shell shows approvals but has no rule editor, permissions client or page.",
	SettingsSectionGit:             "Unavailable: the Git page is not built yet; the shell has no Git client or page.",
	SettingsSectionHooks:           "Unavailable: the hooks page is not built yet; the shell has no hooks client or page.",
	SettingsSectionPlugins:         "Unavailable: the plugins page is not built yet; the shell has no plugin client or page.",
	SettingsSectionExternalAgent:   "Unavailable: the external agent page is not built yet; the shell has no external-agent client or page.",
	SettingsSectionAbout:           "Unavailable: the about page is not built yet; the shell has no build, version or credits page to show.",
}

// SettingsSection is one page of the settings surface. The zero value is the
// personalization page, which is where the reference opens the route: a view
// that names no section opens there, and a window that has never saved a
// section draws it without extra state. The appearance page stays a normal
// section the navigation reaches.
//
// The numbers are not the navigation's order; settingsSections is, so that the
// reference's 19 entries keep their order while the zero value names the
// opening page.
type SettingsSection uint8

const (
	// SettingsSectionPersonalization is the interface-mode and workspace page.
	// It is the zero value and the surface's default.
	SettingsSectionPersonalization SettingsSection = iota
	// SettingsSectionAccount is the account and sign-in page.
	SettingsSectionAccount
	// SettingsSectionUsage is the usage and spend page.
	SettingsSectionUsage
	// SettingsSectionAppearance is the color, type, layout, skin and language
	// page.
	SettingsSectionAppearance
	// SettingsSectionGeneral is the general desktop preferences page.
	SettingsSectionGeneral
	// SettingsSectionShortcuts is the keyboard-shortcut page.
	SettingsSectionShortcuts
	// SettingsSectionModels is the model-provider page.
	SettingsSectionModels
	// SettingsSectionMultimodal is the multimodal-settings page.
	SettingsSectionMultimodal
	// SettingsSectionRuntime is the runtime page.
	SettingsSectionRuntime
	// SettingsSectionMCP is the MCP-server page.
	SettingsSectionMCP
	// SettingsSectionNotifications is the notification page.
	SettingsSectionNotifications
	// SettingsSectionPet is the pet page.
	SettingsSectionPet
	// SettingsSectionPermissions is the permission-rules page.
	SettingsSectionPermissions
	// SettingsSectionGit is the Git page.
	SettingsSectionGit
	// SettingsSectionHooks is the hooks page.
	SettingsSectionHooks
	// SettingsSectionPlugins is the plugin-settings page.
	SettingsSectionPlugins
	// SettingsSectionExternalAgent is the external-agent page.
	SettingsSectionExternalAgent
	// SettingsSectionArchivedSessions is the archived-session page. The record
	// of an archived session belongs to the App Server; this page waits on a
	// list the shell does not have.
	SettingsSectionArchivedSessions
	// SettingsSectionAbout is the about page.
	SettingsSectionAbout
)

// settingsSectionLast is the last valid SettingsSection; a saved number above
// it names no page and folds onto the opening page.
const settingsSectionLast = SettingsSectionAbout

// settingsSections is the navigation's order: the reference's 17 entries
// (docs/evidence/electron-reference/window-settings.png), from Account to
// External Agent, then this shell's own archived-sessions and about entries.
// The two extra pages are appended rather than placed inside the reference's
// list so the 17 reference rows keep their order. The shell's capability gaps
// are content of the About page, not a navigation row of their own.
var settingsSections = [...]SettingsSection{
	SettingsSectionAccount,
	SettingsSectionUsage,
	SettingsSectionPersonalization,
	SettingsSectionAppearance,
	SettingsSectionGeneral,
	SettingsSectionShortcuts,
	SettingsSectionModels,
	SettingsSectionMultimodal,
	SettingsSectionRuntime,
	SettingsSectionMCP,
	SettingsSectionNotifications,
	SettingsSectionPet,
	SettingsSectionPermissions,
	SettingsSectionGit,
	SettingsSectionHooks,
	SettingsSectionPlugins,
	SettingsSectionExternalAgent,
	SettingsSectionArchivedSessions,
	SettingsSectionAbout,
}

// resolvedSection folds an unknown saved section onto the default, as
// resolveSettings does for the enum preferences. Every value from
// Personalization through About is valid, and a view that names no page opens
// on Personalization, as the reference does.
func resolvedSection(s SettingsSection) SettingsSection {
	if int(s) > int(settingsSectionLast) {
		return SettingsSectionPersonalization
	}
	return s
}

// settingsSectionName is the section's English name, used for the page heading
// and the navigation row. Only Appearance and About have content today; the
// other pages are not built, so their names stay English until i18n.go carries
// the settings.section.* keys.
func settingsSectionName(s SettingsSection) string {
	switch s {
	case SettingsSectionAccount:
		return "Account"
	case SettingsSectionUsage:
		return "Usage"
	case SettingsSectionPersonalization:
		return "Personalization"
	case SettingsSectionGeneral:
		return "General"
	case SettingsSectionShortcuts:
		return "Shortcuts"
	case SettingsSectionModels:
		return "Models"
	case SettingsSectionMultimodal:
		return "Multimodal"
	case SettingsSectionRuntime:
		return "Runtime"
	case SettingsSectionMCP:
		return "MCP"
	case SettingsSectionNotifications:
		return "Notifications"
	case SettingsSectionPet:
		return "Pet"
	case SettingsSectionPermissions:
		return "Permissions"
	case SettingsSectionGit:
		return "Git"
	case SettingsSectionHooks:
		return "Hooks"
	case SettingsSectionPlugins:
		return "Plugins"
	case SettingsSectionExternalAgent:
		return "External Agent"
	case SettingsSectionArchivedSessions:
		return "Archived sessions"
	case SettingsSectionAbout:
		return "About"
	default:
		return tr(DefaultLocale, keySettingsTitle)
	}
}

// settingsSectionLabel is the section's name in the active language. The
// appearance page keeps its existing key, so a Chinese surface names it 外观;
// a section whose settings.section.* key is still missing keeps its English
// name rather than showing an internal key.
func settingsSectionLabel(l Locale, s SettingsSection) string {
	switch s {
	case SettingsSectionAppearance:
		return tr(l, keySettingsTitle)
	}
	slug := settingsSectionSlug(s)
	if slug == "" {
		return settingsSectionName(s)
	}
	key := "settings.section." + slug
	if text := tr(l, key); text != key {
		return text
	}
	return settingsSectionName(s)
}

// settingsSectionSlug is the message-key stem of a section. The appearance page
// keeps its own key (settings.title), so it has no slug.
func settingsSectionSlug(s SettingsSection) string {
	switch s {
	case SettingsSectionAccount:
		return "account"
	case SettingsSectionUsage:
		return "usage"
	case SettingsSectionPersonalization:
		return "personalization"
	case SettingsSectionGeneral:
		return "general"
	case SettingsSectionShortcuts:
		return "shortcuts"
	case SettingsSectionModels:
		return "models"
	case SettingsSectionMultimodal:
		return "multimodal"
	case SettingsSectionRuntime:
		return "runtime"
	case SettingsSectionMCP:
		return "mcp"
	case SettingsSectionNotifications:
		return "notifications"
	case SettingsSectionPet:
		return "pet"
	case SettingsSectionPermissions:
		return "permissions"
	case SettingsSectionGit:
		return "git"
	case SettingsSectionHooks:
		return "hooks"
	case SettingsSectionPlugins:
		return "plugins"
	case SettingsSectionExternalAgent:
		return "externalAgent"
	case SettingsSectionArchivedSessions:
		return "archivedSessions"
	case SettingsSectionAbout:
		return "about"
	default:
		return ""
	}
}

// ThemeChoice is the appearance's color-theme preference: follow the desktop,
// or force light or dark. It is a preference, not the theme in force; the
// window resolves it and applies it through ui.Context.SetTheme.
type ThemeChoice uint8

const (
	// ThemeSystem follows the desktop's light or dark appearance.
	ThemeSystem ThemeChoice = iota
	// ThemeLight forces the light tokens.
	ThemeLight
	// ThemeDark forces the dark tokens.
	ThemeDark
)

// String is the choice's name, as the theme row shows it.
func (t ThemeChoice) String() string {
	switch t {
	case ThemeLight:
		return "Light"
	case ThemeDark:
		return "Dark"
	default:
		return "System"
	}
}

// FontFamily is the family the shell draws its interface and code in: the
// system stack, or the monospace stack. It names a family, not a font file:
// mygo resolves the system one to the platform's interface font and the
// monospace one to the platform's monospaced font (see monospaceFamily).
type FontFamily uint8

const (
	// FontSystem is the platform's interface font.
	FontSystem FontFamily = iota
	// FontMonospace is the platform's monospace font.
	FontMonospace
)

// String is the family's name, as the font row shows it.
func (f FontFamily) String() string {
	if f == FontMonospace {
		return "Monospace"
	}
	return "System font"
}

// Density is the shell's spacing preference: regular, or compact for more rows
// in the same room. It is a desktop preference the window applies to its own
// layout: compact keeps densityCompactScale of every gap and padding (see
// tokens.go), in the shell's views and in mygo's widgets, while type sizes and
// row heights keep their values. It changes no session fact.
type Density uint8

const (
	// DensityRegular is the shell's default spacing.
	DensityRegular Density = iota
	// DensityCompact tightens the rows.
	DensityCompact
)

// String is the density's name, as the layout row shows it.
func (d Density) String() string {
	if d == DensityCompact {
		return "Compact"
	}
	return "Regular"
}

// SettingsField names the one preference a SettingsChange carries.
type SettingsField uint8

const (
	// SettingsFieldTheme is ThemeChoice.
	SettingsFieldTheme SettingsField = iota
	// SettingsFieldFontSize is the UI font-size step, in DIPs.
	SettingsFieldFontSize
	// SettingsFieldFontFamily is the FontFamily.
	SettingsFieldFontFamily
	// SettingsFieldSidebarWidth is the sidebar width, in DIPs.
	SettingsFieldSidebarWidth
	// SettingsFieldDensity is the Density.
	SettingsFieldDensity
	// SettingsFieldCodeFontSize is the code font size, in DIPs.
	SettingsFieldCodeFontSize
	// SettingsFieldLocale is the interface language.
	SettingsFieldLocale
)

// String names the field, for a caller's log or error.
func (f SettingsField) String() string {
	switch f {
	case SettingsFieldFontSize:
		return "font size"
	case SettingsFieldFontFamily:
		return "font family"
	case SettingsFieldSidebarWidth:
		return "sidebar width"
	case SettingsFieldDensity:
		return "density"
	case SettingsFieldCodeFontSize:
		return "code font size"
	case SettingsFieldLocale:
		return "language"
	default:
		return "theme"
	}
}

// ArchivedSession is one row of the archived-sessions page: the session id the
// Restore entry addresses and the catalog's title for it. It is a display copy;
// the App Server owns the session, and the archived list is only the desktop's
// own hidden set.
type ArchivedSession struct {
	SessionID string
	Title     string
}

// settingsArchivedNote explains what the archived-sessions page does: it hides
// rows on this desktop and never moves or deletes a session on the App Server.
const settingsArchivedNote = "Archived sessions are hidden from the sidebar on this desktop; their messages stay on the App Server, and restoring a session brings its row back."

// SettingsView is what the settings surface draws: the desktop's own
// preference values, the interface language among them, plus the display
// copies of the catalog rows the archived-sessions and models pages list. The
// window owns the preferences and persists them; the session and model fields
// are copies of App Server facts, never a second source, and no field is a run
// or approval fact.
//
// A zero field takes the token default, so a window that has never saved a
// preference draws the values the shell opens with and English text.
// resolveSettings fills the defaults and keeps a saved value on the range the
// surface offers, and newTokensFor folds the resolved view into the tokens the
// frame draws with.
type SettingsView struct {
	// Locale is the shell language preference. It is desktop-local and never a
	// session or App Server fact.
	Locale Locale
	// Section is the page the surface opens on. The surface keeps the reader's
	// selection itself once it is open, so this is a starting point, not a
	// preference and not a session fact; a zero value opens the personalization
	// page, as the reference does.
	Section SettingsSection
	// Theme is the color-theme preference: system, light or dark.
	Theme ThemeChoice
	// FontSize is the UI text step, one of fontSizeSteps.
	FontSize float32
	// FontFamily is the interface and code family.
	FontFamily FontFamily
	// SidebarWidth is the sidebar's width in DIPs, held between the sidebar's
	// narrowest and widest.
	SidebarWidth float32
	// Density is the regular or compact spacing.
	Density Density
	// CodeFontSize is the code text size in DIPs, held to its range.
	CodeFontSize float32
	// ArchivedSessions are the catalog rows the archived-sessions page lists,
	// in the catalog's order. The window fills them from its catalog and its
	// desktop-local archived id list; the page reads no session fact of its own
	// and an id the catalog does not hold draws nothing.
	ArchivedSessions []ArchivedSession
	// ModelCatalog is the App Server's model snapshot the models page lists:
	// the same display copy the composer's model list reads, not a second
	// source. The page groups its choices by provider and draws them read-only.
	ModelCatalog ModelCatalog
	// ModelStatus records whether that catalog was read. Available is false
	// while the App Server does not advertise provider.management or the read
	// failed, and Reason then names the gap; the page shows the reason instead
	// of an empty list that would look like "no models".
	ModelStatus ModelStatus
}

// SettingsChange is one preference the user just chose. Field says which one
// changed; the other fields repeat the resolved view, so a caller can take the
// one value or store the whole snapshot. Every value is inside the range the
// surface offers: a width is clamped, a size snapped and an unknown language
// folded to English before it is reported, so a saved value outside the bounds
// cannot leak back through a click.
type SettingsChange struct {
	Field        SettingsField
	Locale       Locale
	Theme        ThemeChoice
	FontSize     float32
	FontFamily   FontFamily
	SidebarWidth float32
	Density      Density
	CodeFontSize float32
}

// defaultSettingsView is the appearance the surface opens with: the
// personalization page, the theme following the desktop, the type sitting on the
// tokens' own default sizes, the sidebar at its default width, and the density
// regular. It is the same value resolveSettings derives from a zero view.
func defaultSettingsView(k tokens) SettingsView {
	return resolveSettings(k, SettingsView{})
}

// resolveSettings turns a saved view into what the surface draws: an unknown
// enum or language falls back to its default, a UI size snaps to the nearest
// offered step, the code size and the sidebar width are held to their ranges,
// and a zero value takes the token default.
func resolveSettings(k tokens, v SettingsView) SettingsView {
	v.Locale = resolveLocale(v.Locale)
	v.Section = resolvedSection(v.Section)
	switch v.Theme {
	case ThemeLight, ThemeDark:
	default:
		v.Theme = ThemeSystem
	}
	v.FontSize = settingsFontSize(k, v.FontSize)
	if v.FontFamily != FontMonospace {
		v.FontFamily = FontSystem
	}
	if v.Density != DensityCompact {
		v.Density = DensityRegular
	}
	v.SidebarWidth = k.resolveSidebarWidth(v.SidebarWidth)
	v.CodeFontSize = settingsCodeFontSize(k, v.CodeFontSize)
	return v
}

// settingsFontSize is the UI step the surface draws for a saved size: the
// token's default step until one is set, and the nearest step for a value
// between two.
func settingsFontSize(k tokens, size float32) float32 {
	if size <= 0 {
		return k.uiFontSize
	}
	return snapFontSize(size)
}

// settingsCodeFontSize is the code size the surface draws: the token's default
// until one is set, and the rounded, clamped value otherwise.
func settingsCodeFontSize(k tokens, size float32) float32 {
	if size <= 0 {
		return k.codeFontSize
	}
	return clampCodeFontSize(size)
}

// clampCodeFontSize holds a code size to the range the slider offers, so a
// step at either end cannot take the code text past it.
func clampCodeFontSize(size float32) float32 {
	size = float32(math.Round(float64(size)))
	if size < settingsCodeMin {
		return settingsCodeMin
	}
	if size > settingsCodeMax {
		return settingsCodeMax
	}
	return size
}

// fontStepIndex is the selected segment of the UI font-size control: the
// nearest offered step's position in fontSizeSteps.
func fontStepIndex(size float32) int {
	step := snapFontSize(size)
	for i, want := range fontSizeSteps {
		if step == want {
			return i
		}
	}
	return 0
}

// fontFamilyAt is the family a select option stands for.
func fontFamilyAt(index int) FontFamily {
	if index == 1 {
		return FontMonospace
	}
	return FontSystem
}

// themeChoiceText is the theme's name in the active language. The choice's own
// String stays the English name, for a caller's log and for the tests that pin
// the default.
func themeChoiceText(l Locale, t ThemeChoice) string {
	switch t {
	case ThemeLight:
		return tr(l, keySettingsLight)
	case ThemeDark:
		return tr(l, keySettingsDark)
	default:
		return tr(l, keySettingsSystem)
	}
}

// fontFamilyText is the family's name in the active language.
func fontFamilyText(l Locale, f FontFamily) string {
	if f == FontMonospace {
		return tr(l, keySettingsFontMonospace)
	}
	return tr(l, keySettingsFontSystem)
}

// publishSettings reports one user choice. The nil callback is a window that
// has not wired persistence yet: the surface stays visible with its current
// values and reports nothing, rather than changing a preference nowhere.
func publishSettings(onChange func(SettingsChange), field SettingsField, v SettingsView) {
	if onChange == nil {
		return
	}
	onChange(SettingsChange{
		Field:        field,
		Locale:       v.Locale,
		Theme:        v.Theme,
		FontSize:     v.FontSize,
		FontFamily:   v.FontFamily,
		SidebarWidth: v.SidebarWidth,
		Density:      v.Density,
		CodeFontSize: v.CodeFontSize,
	})
}

// renderSettings draws the settings route: one card filling the window, the
// section navigation on the left and the chosen section's page on the right.
// The shell draws no sidebar, toolbar, conversation or composer while it shows,
// so a preference change is read on a page of its own rather than beside the
// chat it changes. The card's 4.5 DIP inset and rounded face match the framed
// workspace (mainFramePad/mainFrameRadius), so the route opens as the same
// window with a different card.
//
// A narrow window (settingsCompactW) shows one level at a time instead: the
// section list first, then the chosen page with a return row above it, because
// the reading column does not fit beside the navigation there.
//
// The navigation is one focusable row per section, marked selected for the page
// on screen, and a Back row above it. A click moves the surface's own
// settingsRouteState and reports no SettingsChange, because the page is display
// state, not a desktop preference. Back and Escape call onBack, which the
// window wires to the same state change the settings entry toggles; on a narrow
// window's section page Escape returns to the list first, as the page's return
// row does.
//
// The appearance page draws the theme, language, font, size, code-theme and
// skin groups; the archived-sessions page lists the rows the desktop hides and
// offers Restore; every other page keeps its navigation row and explains the
// client or page it waits on, with a visible WIP badge and no invented control.
// It reads SettingsView and the tokens only. It calls onChange at most once per
// user choice, with the resolved view, and writes no preference itself; a
// Restore click calls onRestore with the row's session id and changes no list
// itself. The surface's own text is drawn in the language the view names, so a
// language choice shows on the next frame.
func renderSettings(c *ui.Context, k tokens, v SettingsView, capabilities []wipCapability, onChange func(SettingsChange), onBack func(), onRestore func(sessionID string)) {
	v = resolveSettings(k, v)
	l := v.Locale
	width, _ := c.Size()
	compact := width < settingsCompactW
	// The chosen section and level are the surface's own state: the view names
	// where the surface opens, and ui.Local keeps the reader's click from frame
	// to frame. A frame that does not build this row drops it, so reopening the
	// route starts on the section the view names and, in a narrow window, on
	// the list.
	root := ui.Row(c).Fill().Padding(k.framePad).Background(k.background)
	state := ui.Local(root, settingsRouteLocalKey{}, func() settingsRouteState {
		return settingsRouteState{section: v.Section}
	})
	if state == nil {
		opened := settingsRouteState{section: v.Section}
		state = &opened
	}
	state.section = resolvedSection(state.section)
	if c.Shortcut(0, ui.KeyEscape) {
		switch {
		case compact && state.detail:
			state.detail = false
		case onBack != nil:
			onBack()
		}
	}

	root.Children(func() {
		ui.Row(c).Key("settings-route").Grow(1).FillWidth().FillHeight().
			Radius(k.frameRadius).BorderWidth(k.hairline).BorderColor(k.border).
			Background(k.card).Clip().Role(ui.RoleRegion).Label(tr(l, keySettingsRegion)).
			Transition(motionPanel(c)).Children(func() {
			if compact {
				if state.detail {
					settingsCompactPage(c, k, v, capabilities, state, onChange, onRestore)
					return
				}
				settingsNavigation(c, k, l, state, onBack, true)
				return
			}
			settingsNavigation(c, k, l, state, onBack, false)
			settingsPage(c, k, v, capabilities, state.section, onChange, onRestore)
		})
	})
}

// settingsNavigation draws the section column: a Back row, then every section
// the reference lists in its order, then the shell's own archived-sessions and
// about pages. Each row is one focusable entry with
// its 16×16 outline glyph, and the selected row carries the sidebar's selection
// pill. The column scrolls when the window is too short for every row, and a
// click only moves the route's section; the Back row calls onBack, which is the
// same callback Escape takes. In a narrow window the column takes the whole card
// and a row click opens that section's page instead of switching the page
// beside it.
func settingsNavigation(c *ui.Context, k tokens, l Locale, state *settingsRouteState, onBack func(), compact bool) {
	nav := ui.Column(c).FillHeight()
	if compact {
		nav.FillWidth()
	} else {
		nav.BasisPercent(settingsNavShare).Shrink(0).MaxWidth(settingsNavW).
			Clip().BorderEndWidth(k.hairline).BorderColor(k.border)
	}
	nav.Role(ui.RoleNavigation).Label(tr(l, keySettingsNavigation))
	nav.Children(func() {
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(settingsNavPadY, settingsNavPadX).Children(func() {
				settingsBackRow(c, k, l, onBack)
				ui.Box(c).FillWidth().Height(settingsNavBackGap).AccessHidden(true)
				for _, s := range settingsSections {
					settingsNavRow(c, k, l, s, s == state.section, state, compact)
				}
			})
		})
	})
}

// settingsBackRow is the route's return entry: the arrow and label the
// reference puts above the sections. It is the first Tab stop and the same
// action Escape takes.
func settingsBackRow(c *ui.Context, k tokens, l Locale, onBack func()) {
	settingsNavActionRow(c, k, "settings-back", tr(l, keySettingsBack), onBack)
}

// settingsCompactBackRow is the narrow window's return from a section page to
// the section list. It keeps the Back row's shape and focus ring, so the two
// levels read as the same surface.
func settingsCompactBackRow(c *ui.Context, k tokens, l Locale, state *settingsRouteState) {
	settingsNavActionRow(c, k, "settings-sections-back", settingsSectionsBackLabel(l), func() {
		state.detail = false
	})
}

// settingsNavActionRow is one navigation action above the sections: the arrow
// glyph and a label. A nil action draws the row without a click, as a window
// that has not wired the return yet.
func settingsNavActionRow(c *ui.Context, k tokens, key, label string, action func()) {
	row := ui.ButtonBase(c).Key(key).Height(k.sidebarRowH).FillWidth().
		PaddingX(settingsNavPadX).Radius(radiusMD).Gap(settingsNavIconGap).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(label).
		Transition(motionFastColors(c))
	if row.Hovered() {
		row.Background(k.sidebarHover)
	}
	if action != nil {
		row.OnClick(action)
	}
	row.Children(func() {
		sidebarGlyph(c, settingsIcons.back, k.foreground)
		ui.Text(c, label).FontSize(k.bodySize).TextColor(k.foreground).SingleLine()
	})
}

// settingsNavRow is one section entry: its glyph, its label and, when chosen,
// the neutral selection pill and the Selected state. The glyph is a decoration;
// the row's label names the section for assistive technology. In a narrow
// window the click also opens the page, which is the second level there.
func settingsNavRow(c *ui.Context, k tokens, l Locale, s SettingsSection, selected bool, state *settingsRouteState, compact bool) {
	label := settingsSectionLabel(l, s)
	entry := ui.ButtonBase(c).Key("settings-section:" + settingsSectionSlug(s)).
		Height(k.sidebarRowH).FillWidth().PaddingX(settingsNavPadX).Radius(radiusMD).
		Gap(settingsNavIconGap).AlignItems(ui.Center).Role(ui.RoleListItem).
		Selected(selected).Transition(motionFastColors(c))
	switch {
	case selected:
		entry.Background(k.sidebarAccent)
	case entry.Hovered():
		entry.Background(k.sidebarHover)
	}
	entry.OnClick(func() {
		state.section = s
		if compact {
			state.detail = true
		}
	})
	entry.Children(func() {
		color := k.foreground
		if selected {
			color = k.sidebarAccentForeground
		}
		sidebarGlyph(c, settingsSectionIcon(s), color)
		text := ui.Text(c, label).FontSize(k.bodySize).TextColor(color).
			SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
		if selected {
			text.Bold()
		}
	})
}

// settingsCompactPage draws the narrow window's second level: the return row to
// the section list and the chosen section's page. The route cannot draw the
// navigation beside the page at this width, so the page takes the card and the
// return row is the way back to the list.
func settingsCompactPage(c *ui.Context, k tokens, v SettingsView, capabilities []wipCapability, state *settingsRouteState, onChange func(SettingsChange), onRestore func(sessionID string)) {
	ui.Column(c).Grow(1).FillWidth().FillHeight().MinWidth(0).Children(func() {
		ui.Column(c).FillWidth().Padding(settingsNavPadY, settingsNavPadX).Children(func() {
			settingsCompactBackRow(c, k, v.Locale, state)
		})
		settingsPage(c, k, v, capabilities, state.section, onChange, onRestore)
	})
}

// settingsPage draws the right-hand page. The appearance page is the one with
// preferences; the archived-sessions page lists the desktop's hidden rows and
// offers Restore; the models page lists the App Server's model catalog read-only;
// the shortcuts page lists the shortcuts this shell wires; the about page draws
// its own WIP explanation and the shell's unwired capabilities, and every other
// page draws its title, a WIP badge and the client or page it waits on, so the
// section never pretends a preference was read.
func settingsPage(c *ui.Context, k tokens, v SettingsView, capabilities []wipCapability, section SettingsSection, onChange func(SettingsChange), onRestore func(sessionID string)) {
	l := v.Locale
	ui.Scroll(c).Grow(1).FillHeight().MinWidth(0).Clip().Children(func() {
		ui.Column(c).FillWidth().MaxWidth(settingsPageMaxW).Padding(settingsPagePad).Gap(settingsPagePad).
			Children(func() {
				ui.Text(c, settingsSectionLabel(l, section)).Bold().FontSize(k.heroTitleSize).
					TextColor(k.foreground).Role(ui.RoleHeading).Level(1)
				switch section {
				case SettingsSectionAbout:
					// The about page has no build or version fact to show, so it
					// keeps the WIP explanation and, under it, the capability list
					// the reference keeps off its navigation.
					settingsWIPPage(c, k, section, v.Locale)
					settingsDivider(c, k)
					settingsCapabilitiesPage(c, k, v, capabilities)
					return
				case SettingsSectionArchivedSessions:
					settingsArchivedPage(c, k, v, onRestore)
					return
				case SettingsSectionModels:
					settingsModelsPage(c, k, v)
					return
				case SettingsSectionShortcuts:
					settingsShortcutsPage(c, k, v)
					return
				case SettingsSectionAppearance:
					// The appearance page is the one with preferences.
				default:
					settingsWIPPage(c, k, section, v.Locale)
					return
				}
				settingsThemeSection(c, k, v, onChange)
				settingsDivider(c, k)
				settingsLanguageSection(c, k, v, onChange)
				settingsDivider(c, k)
				settingsFontSection(c, k, v, onChange)
				settingsDivider(c, k)
				settingsUISizeSection(c, k, v, onChange)
				settingsDivider(c, k)
				settingsCodeSizeSection(c, k, v, onChange)
				settingsDivider(c, k)
				settingsCodeThemeSection(c, k, v)
				settingsDivider(c, k)
				settingsSkinSection(c, k, v)
				// The desktop-local note sits last so the page opening on the
				// reference's theme, language, font and size groups matches its
				// visible order; the note explains where the preferences live
				// without pushing the controls below the fold.
				ui.Text(c, tr(l, keySettingsLocalNote)).FontSize(k.smallSize).
					TextColor(k.mutedForeground).Wrap()
			})
	})
}

// settingsCapabilitiesNote explains the capability list the About page carries.
// There is no message key for it yet, so it stays English in every language.
const settingsCapabilitiesNote = "These actions have no client, page or callback wired in this shell yet. Each card names the entry it waits on; nothing here can be clicked."

// settingsCapabilitiesPage lists the actions the shell has not wired, as part
// of the About page: the reference has no capabilities entry of its own in the
// navigation. It reads the list the shell assembled and offers no control, so a
// reader can learn what is missing without a click that could imply the
// capability exists. The server-owned facts behind each card stay in the App
// Server; only the product status is drawn here.
func settingsCapabilitiesPage(c *ui.Context, k tokens, v SettingsView, capabilities []wipCapability) {
	l := v.Locale
	ui.Text(c, settingsCapabilitiesNote).FontSize(k.smallSize).TextColor(k.mutedForeground)
	ui.Column(c).FillWidth().Gap(k.gap).Role(ui.RoleGroup).Label(tr(l, keyCapabilityGroup)).
		Children(func() {
			for _, capability := range capabilities {
				capabilityCard(c, k, l, capability)
			}
		})
}

// settingsWIPPage explains a section the shell has no page for: the visible
// badge, then the missing client or page. It offers no control, so nothing on
// the page can report a preference the desktop never saved.
func settingsWIPPage(c *ui.Context, k tokens, section SettingsSection, l Locale) {
	ui.Text(c, wipBadge).FontSize(k.smallSize).Bold().TextColor(k.mutedForeground)
	gap := settingsSectionGaps[section]
	if key := "settings.gap." + settingsSectionSlug(section); key != "settings.gap." {
		if translated := tr(l, key); translated != key {
			gap = translated
		}
	}
	ui.Text(c, gap).FontSize(k.bodySize).TextColor(k.mutedForeground)
}

// settingsArchivedPage lists the sessions this desktop hides from the sidebar.
// The rows are catalog copies the window assembled, so the page invents no
// session and no title: an empty list says so, and a row without a catalog title
// reads the locale's untitled placeholder. Restore reports the row's id to the
// window and changes no list itself; while onRestore is nil the entry stays
// visible, disabled and explained rather than looking like it restored
// something.
func settingsArchivedPage(c *ui.Context, k tokens, v SettingsView, onRestore func(sessionID string)) {
	l := v.Locale
	ui.Text(c, tr(l, keySettingsArchivedNote)).FontSize(k.bodySize).
		TextColor(k.mutedForeground).Wrap()
	if len(v.ArchivedSessions) == 0 {
		ui.Text(c, tr(l, keySettingsArchivedEmpty)).FontSize(k.bodySize).
			TextColor(k.mutedForeground).Role(ui.RoleStatus)
		return
	}
	ui.Column(c).FillWidth().Gap(k.gap).Role(ui.RoleGroup).
		Label(tr(l, keySettingsArchivedNote)).Children(func() {
		for _, row := range v.ArchivedSessions {
			title := sessionLabel(row.Title, l)
			ui.Row(c).Key("archived-session:" + row.SessionID).FillWidth().
				MinHeight(k.sidebarRowH).PaddingX(k.gap).Gap(k.gapWide).
				AlignItems(ui.Center).Children(func() {
				ui.Text(c, title).FontSize(k.bodySize).TextColor(k.foreground).
					SingleLine().Grow(1).MinWidth(0)
				restore := tr(l, keySettingsArchivedRestore)
				button := ui.ButtonBase(c).Key("archived-restore:" + row.SessionID).
					Label(restore + " " + title).PaddingX(k.gap).AlignItems(ui.Center).
					Transition(motionFastColors(c))
				if onRestore == nil {
					button.AccessDisabled(true).
						Description(reasonRestoreArchived).Opacity(wipOpacity)
				} else {
					button.OnClick(func() { onRestore(row.SessionID) })
				}
				button.Children(func() {
					ui.Text(c, restore).FontSize(k.bodySize).TextColor(k.foreground).SingleLine()
				})
			})
		}
	})
}

// reasonRestoreArchived explains a Restore entry the window has not wired.
// It is an internal reason, not a message key, like the other reason constants.
const reasonRestoreArchived = "Unavailable: the window has no desktop-local archived-session restore callback yet."

// settingsModelsPage lists the App Server's model catalog, grouped by
// provider, read-only. It draws the catalog the shell copied into the view; the
// page starts no read of its own and offers no switch, so it cannot enable or
// disable a model. When the window could not read the catalog, the page shows
// the window's reason behind the visible WIP badge rather than an empty list,
// which would read as "no models".
func settingsModelsPage(c *ui.Context, k tokens, v SettingsView) {
	l := v.Locale
	ui.Text(c, tr(l, keySettingsModelsNote)).FontSize(k.smallSize).
		TextColor(k.mutedForeground).Wrap()
	if !v.ModelStatus.Available {
		reason := v.ModelStatus.Reason
		if reason == "" {
			reason = tr(l, keySettingsModelsMissing)
		}
		ui.Row(c).FillWidth().Gap(k.gap).AlignItems(ui.Start).Children(func() {
			ui.Text(c, wipBadge).FontSize(k.smallSize).Bold().TextColor(k.mutedForeground)
			ui.Text(c, reason).FontSize(k.bodySize).TextColor(k.mutedForeground).
				Wrap().Grow(1).MinWidth(0)
		})
		return
	}
	if v.ModelCatalog.HasCurrent && v.ModelCatalog.Current != "" {
		ui.Text(c, tr(l, keySettingsModelsCurrent)+": "+v.ModelCatalog.Current).
			FontSize(k.smallSize).TextColor(k.mutedForeground).SingleLine()
	}
	if len(v.ModelCatalog.Choices) == 0 {
		ui.Text(c, tr(l, keySettingsModelsEmpty)).FontSize(k.bodySize).
			TextColor(k.mutedForeground).Role(ui.RoleStatus)
		return
	}
	for _, group := range settingsModelGroups(v.ModelCatalog) {
		settingsModelGroupSection(c, k, l, group)
	}
}

// settingsModelGroup is one provider's models, in the catalog's order. The
// grouping is display only: it keeps ProviderName from the last row that names
// one and appends every choice unchanged.
type settingsModelGroup struct {
	ProviderID   string
	ProviderName string
	Choices      []ModelChoice
}

// settingsModelGroups groups the catalog's choices by provider without
// reordering them: the first provider keeps the first place, and every choice
// stays in the order the App Server served it.
func settingsModelGroups(catalog ModelCatalog) []settingsModelGroup {
	groups := make([]settingsModelGroup, 0, len(catalog.Choices))
	index := make(map[string]int, len(catalog.Choices))
	for _, choice := range catalog.Choices {
		at, ok := index[choice.ProviderID]
		if !ok {
			at = len(groups)
			index[choice.ProviderID] = at
			groups = append(groups, settingsModelGroup{
				ProviderID:   choice.ProviderID,
				ProviderName: choice.ProviderName,
			})
		}
		if groups[at].ProviderName == "" {
			groups[at].ProviderName = choice.ProviderName
		}
		groups[at].Choices = append(groups[at].Choices, choice)
	}
	return groups
}

// settingsModelGroupSection draws one provider heading and its models.
func settingsModelGroupSection(c *ui.Context, k tokens, l Locale, group settingsModelGroup) {
	title := group.ProviderName
	if title == "" {
		title = group.ProviderID
	}
	if group.ProviderID != "" && group.ProviderID != title {
		title += " (" + group.ProviderID + ")"
	}
	settingsSection(c, k, title, func() {
		ui.Column(c).FillWidth().Gap(k.gap).Role(ui.RoleGroup).Label(title).
			Children(func() {
				for _, choice := range group.Choices {
					settingsModelRow(c, k, l, choice)
				}
			})
	})
}

// settingsModelRow is one model's card: its name and id, the current marker,
// whether the provider enabled it, its declared context and output limits, its
// capabilities and, when it is unavailable, the catalog's reason. Every value
// comes from the copied ModelChoice; a missing limit reads n/a and a model that
// declared no capability says so instead of leaving the line blank.
func settingsModelRow(c *ui.Context, k tokens, l Locale, choice ModelChoice) {
	name := choice.DisplayName
	if name == "" {
		name = choice.ModelID
	}
	if name == "" {
		name = choice.ProviderID
	}
	ui.Column(c).FillWidth().Padding(k.gap, k.edgePad).Radius(radiusSM).
		Background(k.muted).Gap(settingsPreviewGap).Children(func() {
		ui.Row(c).FillWidth().Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, name).FontSize(k.bodySize).TextColor(k.foreground).
				SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
			if choice.Current {
				ui.Text(c, tr(l, keySettingsModelsCurrent)).FontSize(k.smallSize).
					TextColor(k.mutedForeground).SingleLine()
			}
			state, color := tr(l, keySettingsModelsDisabled), k.mutedForeground
			if choice.Enabled {
				state = tr(l, keySettingsModelsEnabled)
			} else {
				color = k.danger
			}
			ui.Text(c, state).FontSize(k.smallSize).TextColor(color).SingleLine()
		})
		ui.Text(c, choice.ProviderID+"/"+choice.ModelID).FontSize(k.smallSize).
			TextColor(k.mutedForeground).SingleLine()
		ui.Text(c, settingsModelLimits(l, choice)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).SingleLine().Ellipsis("…")
		ui.Text(c, tr(l, keySettingsModelsCaps)+": "+settingsModelCapabilities(l, choice)).
			FontSize(k.smallSize).TextColor(k.mutedForeground).SingleLine().Ellipsis("…")
		if !choice.Enabled || choice.Reason != "" {
			reason := tr(l, keySettingsModelsUnavailable)
			if choice.Reason != "" {
				reason += ": " + choice.Reason
			}
			ui.Text(c, reason).FontSize(k.smallSize).TextColor(k.danger).Wrap()
		}
	})
}

// settingsModelLimits prints the model's declared context and output limits,
// with n/a where the App Server declared none.
func settingsModelLimits(l Locale, choice ModelChoice) string {
	return tr(l, keySettingsModelsContext) + " " + settingsModelCount(choice.ContextWindow) +
		" · " + tr(l, keySettingsModelsOutput) + " " + settingsModelCount(choice.MaxOutputTokens)
}

// settingsModelCount is one declared limit: the token count formatted like the
// status bar's usage readout, or n/a for a missing value.
func settingsModelCount(value *int64) string {
	if value == nil || *value <= 0 {
		return debugInfoMissing
	}
	return formatTokenCount(*value)
}

// settingsModelCapabilities lists the capabilities the catalog declared, or the
// locale's "none declared" when the model declared none.
func settingsModelCapabilities(l Locale, choice ModelChoice) string {
	caps := make([]string, 0, 3)
	if choice.ToolCalling {
		caps = append(caps, tr(l, keySettingsModelsCapTools))
	}
	if choice.Vision {
		caps = append(caps, tr(l, keySettingsModelsCapVision))
	}
	if choice.Reasoning {
		caps = append(caps, tr(l, keySettingsModelsCapReason))
	}
	if len(caps) == 0 {
		return tr(l, keySettingsModelsNoCaps)
	}
	return strings.Join(caps, " · ")
}

// settingsShortcutRow is one shortcut the shortcuts page lists: the message key
// of its action and the keys as they are drawn. Both come from the shell's own
// wiring, so the page cannot list a shortcut this window does not listen for.
type settingsShortcutRow struct {
	actionKey string
	keys      string
}

// settingsShortcutRows is the shortcuts this shell wires, read from the same
// constants the regions listen with: Cmd+G opens the search palette, Cmd+F
// opens the find bar, Enter submits the composer and Shift+Enter inserts a
// line. Settings has no keyboard shortcut in this shell, so no row claims one.
func settingsShortcutRows() []settingsShortcutRow {
	return []settingsShortcutRow{
		{actionKey: keySettingsShortcutsSearch, keys: settingsShortcutKeys(searchShortcutMods, searchShortcutKey)},
		{actionKey: keySettingsShortcutsFind, keys: settingsShortcutKeys(ui.Cmd, ui.KeyF)},
		{actionKey: keySettingsShortcutsSend, keys: settingsShortcutKeys(0, composerSubmitKey)},
		{actionKey: keySettingsShortcutsNewline, keys: settingsShortcutKeys(ui.Shift, ui.KeyEnter)},
	}
}

// settingsShortcutKeys draws one wired shortcut: the platform's own modifier
// glyph on macOS, the words elsewhere, as the palette's hints do. It is a
// platform label rather than shell text, so the i18n table carries no entry.
func settingsShortcutKeys(mods ui.Modifiers, key ui.Key) string {
	name := settingsShortcutKeyName(key)
	if name == "" {
		return ""
	}
	switch mods {
	case 0:
		return name
	case ui.Cmd:
		if runtime.GOOS == "darwin" {
			return "⌘" + name
		}
		return "Ctrl+" + name
	case ui.Shift:
		return "Shift+" + name
	default:
		return name
	}
}

// settingsShortcutKeyName is the key's name in a shortcut label. The page only
// lists the shell's wired shortcuts, so the letters and keys below are the
// whole set it can draw; an unknown key returns no label rather than a guess.
func settingsShortcutKeyName(key ui.Key) string {
	switch key {
	case ui.KeyG:
		return "G"
	case ui.KeyF:
		return "F"
	case ui.KeyEnter:
		return "Enter"
	default:
		return ""
	}
}

// settingsShortcutsPage lists the shortcuts this shell wires. Each row names an
// action and its keys; the page is read-only, so it offers no editor and says
// the editing gap behind the visible WIP badge. The rows are derived from the
// shell's constants, so no row names a shortcut this window does not listen
// for.
func settingsShortcutsPage(c *ui.Context, k tokens, v SettingsView) {
	l := v.Locale
	ui.Text(c, tr(l, keySettingsShortcutsNote)).FontSize(k.bodySize).
		TextColor(k.mutedForeground).Wrap()
	rows := settingsShortcutRows()
	ui.Column(c).FillWidth().Gap(k.gap).Role(ui.RoleGroup).
		Label(tr(l, keySettingsShortcutsNote)).Children(func() {
		ui.Row(c).FillWidth().PaddingX(k.gap).Gap(k.gapWide).Children(func() {
			ui.Text(c, tr(l, keySettingsShortcutsAction)).Bold().FontSize(k.smallSize).
				TextColor(k.mutedForeground).Grow(1)
			ui.Text(c, tr(l, keySettingsShortcutsKeys)).Bold().FontSize(k.smallSize).
				TextColor(k.mutedForeground)
		})
		for _, row := range rows {
			ui.Row(c).FillWidth().MinHeight(k.sidebarRowH).PaddingX(k.gap).
				Gap(k.gapWide).AlignItems(ui.Center).Children(func() {
				ui.Text(c, tr(l, row.actionKey)).FontSize(k.bodySize).
					TextColor(k.foreground).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
				ui.Text(c, row.keys).FontSize(k.bodySize).
					TextColor(k.mutedForeground).SingleLine()
			})
		}
	})
	ui.Row(c).FillWidth().Gap(k.gap).AlignItems(ui.Start).Children(func() {
		ui.Text(c, wipBadge).FontSize(k.smallSize).Bold().TextColor(k.mutedForeground)
		ui.Text(c, tr(l, keySettingsShortcutsEdit)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).Wrap().Grow(1).MinWidth(0)
	})
}

// settingsThemeSection is the color-theme choice as three cards: Light, Dark
// and System, in the reference's order, with the chosen card outlined in the
// brand color and marked by a dot. They form a radio group, so the arrows move
// between the cards and a click selects one; a choice reports
// SettingsFieldTheme with its own ThemeChoice, never a card position.
func settingsThemeSection(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	l := v.Locale
	settingsSection(c, k, tr(l, keySettingsThemeGroup), func() {
		selected := v.Theme
		ui.Row(c).FillWidth().Gap(k.gapWide).Role(ui.RoleRadioGroup).
			Label(tr(l, keySettingsThemeGroup)).Children(func() {
			for _, choice := range []ThemeChoice{ThemeLight, ThemeDark, ThemeSystem} {
				choice := choice
				c.Key("settings-theme:" + themeChoiceSlug(choice))
				card := ui.RadioBase(c, &selected, choice).
					Grow(1).Shrink(1).MinWidth(0).Height(settingsThemeCardH).
					Padding(k.gapWide).Radius(radiusLG).BorderWidth(k.hairline).
					BorderColor(k.border).Background(k.card).
					Label(themeChoiceText(l, choice)).Transition(motionFastColors(c))
				if choice == v.Theme {
					card.BorderWidth(1.5).BorderColor(k.primary)
				}
				if card.Hovered() && choice != v.Theme {
					card.Background(k.secondary)
				}
				if card.Changed() {
					next := v
					next.Theme = choice
					publishSettings(onChange, SettingsFieldTheme, next)
				}
				card.Children(func() {
					ui.Column(c).FillWidth().Gap(settingsThemeIconGap).Children(func() {
						ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
							dot := ui.Box(c).Size(settingsThemeDot, settingsThemeDot).Radius(settingsThemeDot / 2)
							if choice == v.Theme {
								dot.Background(k.primary)
							}
							dot.AccessHidden(true)
						})
						ui.Column(c).Grow(1).FillWidth().Justify(ui.Center).AlignItems(ui.Center).
							Gap(settingsThemeIconGap).Children(func() {
							sidebarGlyph(c, settingsThemeIcon(choice), k.foreground)
							ui.Text(c, themeChoiceText(l, choice)).FontSize(k.bodySize).
								Bold().TextColor(k.foreground).SingleLine().Ellipsis("…")
						})
					})
				})
			}
		})
	})
}

// settingsLanguageSection is the interface-language choice. The control's
// options are each language's own name, so a reader finds their language
// written the way they write it; choosing one reports SettingsFieldLocale and
// the shell draws the next frame in it.
func settingsLanguageSection(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	l := v.Locale
	options := []string{LocaleEN.Name(), LocaleZHCN.Name()}
	selected := l.Name()
	settingsSection(c, k, tr(l, keySettingsLanguage), func() {
		settingsControlRow(c, k, tr(l, keySettingsLanguageRow), tr(l, keySettingsLanguageNote), func() {
			sel := ui.Select(c, &selected, options).Label(tr(l, keySettingsLanguageRow)).
				Width(settingsSelectW).Shrink(1)
			if sel.Changed() {
				if index := optionIndex(selected, options); index >= 0 {
					next := v
					next.Locale = localeAt(index)
					publishSettings(onChange, SettingsFieldLocale, next)
				}
			}
		})
	})
}

// settingsFontSection is the family pair: a UI-font select the shell can
// change, and a code-font select that is read-only because this shell draws the
// interface and code in one family. The second select names that limitation in
// its description instead of offering a separate family the tokens cannot
// apply.
func settingsFontSection(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	l := v.Locale
	options := []string{tr(l, keySettingsFontSystem), tr(l, keySettingsFontMonospace)}
	selected := fontFamilyText(l, v.FontFamily)
	settingsSection(c, k, tr(l, keySettingsText), func() {
		settingsControlRow(c, k, tr(l, keySettingsFontFamily), tr(l, keySettingsFontNote), func() {
			sel := ui.Select(c, &selected, options).Label(tr(l, keySettingsFontFamily)).
				Width(settingsSelectW).Shrink(1)
			if sel.Changed() {
				if index := optionIndex(selected, options); index >= 0 {
					next := v
					next.FontFamily = fontFamilyAt(index)
					publishSettings(onChange, SettingsFieldFontFamily, next)
				}
			}
		})
		settingsControlRow(c, k, tr(l, keySettingsCodeFont), tr(l, keySettingsCodeFontGap), func() {
			same := tr(l, keySettingsCodeFontSame)
			ui.Select(c, &same, []string{same}).Label(tr(l, keySettingsCodeFont)).
				Width(settingsSelectW).Shrink(1).Disabled(true).AccessDisabled(true).
				Opacity(wipOpacity)
		})
	})
}

// settingsUISizeSection is the UI text size as five segments, from the
// smallest to the largest step the theme offers. A choice reports
// SettingsFieldFontSize; the next frame folds it into every type step.
func settingsUISizeSection(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	l := v.Locale
	labels := []string{
		tr(l, keySettingsSizeXS),
		tr(l, keySettingsSizeSmall),
		tr(l, keySettingsSizeMedium),
		tr(l, keySettingsSizeLarge),
		tr(l, keySettingsSizeXL),
	}
	selected := fontStepIndex(v.FontSize)
	settingsSection(c, k, tr(l, keySettingsFontSize), func() {
		ui.Row(c).FillWidth().Children(func() {
			ui.Box(c).Grow(1)
			segmented := ui.Segmented(c, &selected, labels...).Label(tr(l, keySettingsFontSize))
			if segmented.Changed() {
				next := v
				next.FontSize = fontSizeSteps[selected]
				publishSettings(onChange, SettingsFieldFontSize, next)
			}
		})
	})
}

// settingsCodeSizeSection is the code text size as a 12–20 px slider. The
// slider snaps to whole DIPs, so a drag reports an integer; the current value
// and the range are written around it as text, not as another control.
func settingsCodeSizeSection(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	l := v.Locale
	code := float64(clampCodeFontSize(v.CodeFontSize))
	settingsSection(c, k, tr(l, keySettingsCodeFontSize), func() {
		ui.Row(c).FillWidth().Children(func() {
			ui.Box(c).Grow(1)
			ui.Text(c, settingsPx(float32(code))).FontSize(k.smallSize).TextColor(k.mutedForeground)
		})
		slider := ui.StepSlider(c, &code, settingsCodeMin, settingsCodeMax, settingsCodeStep).
			Label(tr(l, keySettingsCodeFontSize)).FillWidth()
		if slider.Changed() {
			next := v
			next.CodeFontSize = clampCodeFontSize(float32(code))
			publishSettings(onChange, SettingsFieldCodeFontSize, next)
		}
		ui.Row(c).FillWidth().Justify(ui.SpaceBetween).Children(func() {
			for _, value := range []float32{settingsCodeMin, (settingsCodeMin + settingsCodeMax) / 2, settingsCodeMax} {
				ui.Text(c, settingsPx(value)).FontSize(k.smallSize).TextColor(k.mutedForeground).
					SingleLine()
			}
		})
	})
}

// settingsCodeThemeSection names the code theme the shell does not have. The
// select is read-only and says the code follows the interface theme, which is
// what the shell really does; the two preview cards stay WIP and draw no fake
// syntax colors.
func settingsCodeThemeSection(c *ui.Context, k tokens, v SettingsView) {
	l := v.Locale
	settingsSection(c, k, tr(l, keySettingsCodeTheme), func() {
		ui.Text(c, tr(l, keySettingsCodeThemeNote)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).Wrap()
		same := tr(l, keySettingsCodeThemeSame)
		ui.Select(c, &same, []string{same}).Label(tr(l, keySettingsCodeTheme)).
			Width(settingsSelectW).Shrink(1).Disabled(true).AccessDisabled(true).
			Opacity(wipOpacity)
		ui.Row(c).FillWidth().Gap(k.gapWide).Children(func() {
			settingsWIPPreview(c, k, l, tr(l, keySettingsCodePreviewLight))
			settingsWIPPreview(c, k, l, tr(l, keySettingsCodePreviewDark))
		})
	})
}

// settingsWIPPreview is one preview card the shell cannot fill: a bordered
// face with its name and the WIP badge, and a note that no preview exists
// rather than invented syntax colors.
func settingsWIPPreview(c *ui.Context, k tokens, l Locale, label string) {
	ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Height(settingsPreviewH).
		Padding(k.gapWide).Gap(settingsPreviewGap).Radius(radiusMD).
		BorderWidth(k.hairline).BorderColor(k.border).Background(k.card).Children(func() {
		ui.Row(c).FillWidth().Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(k.bodySize).TextColor(k.foreground).SingleLine().
				Ellipsis("…").Grow(1).MinWidth(0)
			ui.Text(c, wipBadge).FontSize(k.smallSize).Bold().TextColor(k.mutedForeground)
		})
		ui.Text(c, tr(l, keySettingsPreviewGap)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).Wrap()
	})
}

// settingsSkinSection is the skin gallery the reference draws. The shell has no
// skin assets and no skin resolver, so every cover is a WIP card with only its
// place and the visible badge, and the two actions stay unavailable with the
// reason they wait on.
func settingsSkinSection(c *ui.Context, k tokens, v SettingsView) {
	l := v.Locale
	settingsSection(c, k, tr(l, keySettingsSkinSection), func() {
		ui.Text(c, tr(l, keySettingsSkinNote)).FontSize(k.smallSize).
			TextColor(k.mutedForeground).Wrap()
		for row := 0; row < 2; row++ {
			ui.Row(c).FillWidth().Gap(k.gapWide).Children(func() {
				for column := 0; column < 3; column++ {
					settingsSkinCard(c, k, l, row*3+column+1)
				}
			})
		}
		ui.Row(c).FillWidth().Gap(k.gapWide).AlignItems(ui.Center).Children(func() {
			settingsWIPButton(c, tr(l, keySettingsSkinNew), tr(l, keyReasonSkin))
			settingsWIPButton(c, tr(l, keySettingsSkinOpen), tr(l, keyReasonSkin))
		})
	})
}

// settingsWIPButton is an unavailable action that shares its row: it is disabled
// for assistive technology, carries the reason as its description, and shrinks
// with the row so a narrow window truncates the label instead of overflowing.
// The group's WIP cards already show the visible badge.
func settingsWIPButton(c *ui.Context, label, reason string) {
	ui.Button(c, label).AccessDisabled(true).Description(reason).Opacity(wipOpacity).
		Grow(1).Shrink(1).MinWidth(0)
}

// settingsSkinCard is one empty cover of the skin gallery: its number, the WIP
// badge and no image, so the gallery keeps the reference's shape without
// pretending a skin asset exists.
func settingsSkinCard(c *ui.Context, k tokens, l Locale, index int) {
	ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Height(settingsSkinCardH).
		Padding(k.gapWide).Gap(settingsPreviewGap).Radius(radiusMD).
		BorderWidth(k.hairline).BorderColor(k.border).Background(k.card).Children(func() {
		ui.Row(c).FillWidth().Gap(k.gap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, tr(l, keySettingsSkinCard)+" "+strconv.Itoa(index)).FontSize(k.bodySize).
				TextColor(k.foreground).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
			ui.Text(c, wipBadge).FontSize(k.smallSize).Bold().TextColor(k.mutedForeground)
		})
	})
}

// settingsSection draws one titled group of setting rows.
func settingsSection(c *ui.Context, k tokens, title string, body func()) {
	ui.Column(c).FillWidth().Gap(k.gapWide).Children(func() {
		ui.Text(c, title).Bold().FontSize(k.bodySize).TextColor(k.foreground).
			Role(ui.RoleHeading).Level(2)
		body()
	})
}

// settingsControlRow draws one setting: its label and explanation on the left,
// the control that changes it on the right. The control carries the setting's
// name for assistive technology, so the row's text is not a second, unrelated
// label.
func settingsControlRow(c *ui.Context, k tokens, label, note string, control func()) {
	ui.Row(c).FillWidth().Gap(k.gapWide).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(settingsPreviewGap).Children(func() {
			ui.Text(c, label).FontSize(k.bodySize).TextColor(k.foreground).SingleLine()
			if note != "" {
				ui.Text(c, note).FontSize(k.smallSize).TextColor(k.mutedForeground).Wrap()
			}
		})
		control()
	})
}

// settingsDivider is the hairline between two sections.
func settingsDivider(c *ui.Context, k tokens) {
	ui.Box(c).FillWidth().Height(k.hairline).Background(k.border).AccessHidden(true)
}

// settingsPx writes a size for the code-size readout.
func settingsPx(size float32) string {
	return strconv.Itoa(int(size)) + " px"
}

// optionIndex is the position of a chosen option in its select's list, or -1
// when the select carries a value the list does not offer.
func optionIndex(value string, options []string) int {
	for i, option := range options {
		if option == value {
			return i
		}
	}
	return -1
}

// themeChoiceSlug is the theme card's stable key stem.
func themeChoiceSlug(choice ThemeChoice) string {
	switch choice {
	case ThemeLight:
		return "light"
	case ThemeDark:
		return "dark"
	default:
		return "system"
	}
}

// settingsThemeIcon is the mark each theme card carries: a sun for light, a
// moon for dark, and a display for following the desktop.
func settingsThemeIcon(choice ThemeChoice) *ui.SVG {
	switch choice {
	case ThemeLight:
		return settingsIcons.sun
	case ThemeDark:
		return settingsIcons.moon
	default:
		return settingsIcons.system
	}
}

// settingsExtraIcons holds the two glyphs this shell adds to the reference's
// navigation: the archived-sessions and about entries. icons.go owns the
// reference's marks and the About page's list, so the shell's own
// pages carry their own shapes here instead of changing that set.
var settingsExtraIcons = struct {
	archive *ui.SVG
	about   *ui.SVG
}{
	archive: sidebarIcon(`<rect x="2.2" y="2.8" width="11.6" height="2.8" rx="0.9"/><path d="M3.6 5.6v6.2c0 1 .8 1.8 1.8 1.8h5.2c1 0 1.8-.8 1.8-1.8V5.6"/><path d="M6.4 8.4h3.2"/>`),
	about:   sidebarIcon(`<circle cx="8" cy="8" r="5.7"/><path d="M8 7.4v4.2"/><circle cx="8" cy="5.1" r=".7" fill="currentColor" stroke="none"/>`),
}

// settingsSectionIcon is the glyph a navigation row draws for its section: the
// shell's own two marks first, then the reference's set from icons.go. Every
// section still draws a 16×16 outline glyph, so no row falls back to text
// alone.
func settingsSectionIcon(s SettingsSection) *ui.SVG {
	switch s {
	case SettingsSectionArchivedSessions:
		return settingsExtraIcons.archive
	case SettingsSectionAbout:
		return settingsExtraIcons.about
	default:
		return settingsIcon(s)
	}
}
