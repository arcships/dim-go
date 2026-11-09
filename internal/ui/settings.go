// This file owns the settings surface: the section navigation dim-electron
// draws on the left, and the page of the chosen section on the right. The
// appearance page is the one the shell built first, and it keeps the desktop's
// own color theme, type, sidebar-width and density preferences, plus the skin
// and chrome entries the shell cannot offer yet. Every other section keeps its
// place in the navigation and explains the client or page it waits on, so the
// settings surface reads as WIP instead of as an empty page that saved
// nothing.
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
	"strconv"

	"github.com/egoist/mygo/ui"
)

// The settings surface's own steps, in DIPs. Padding, gaps, type sizes, radii
// and colors come from the tokens; these are the values only this surface has:
// the width of its reading column (dim-electron's max-w-4xl is 784 DIP at the
// 14px root size), the sidebar-width step a keyboard press takes, and the code
// size range dim-electron's slider offers (AppearanceSettings.vue, 12–20).
const (
	settingsPageMaxW  = 784
	settingsWidthStep = 8
	settingsCodeMin   = 12
	settingsCodeMax   = 20
	settingsCodeStep  = 1
)

// The navigation column's geometry, in DIPs, from docs/l5-reference.md §5.5:
// the reference window is 800 wide and its section column's divider sits at
// x≈228.5, so the column takes 30% of the surface and never grows past 200
// DIPs. The share keeps the column inside a narrow window instead of pushing
// the page off its right edge, and the cap gives the page its reading room on
// a wide one. A section row keeps the sidebar's entry height so the two
// navigations read as one shell.
const (
	settingsNavW     = 200
	settingsNavShare = 30
)

// settingsNavigationLabel names the section column for assistive technology.
// It stays the English fallback for a table that does not carry the key.
// The reference has no message key for it yet, so the label stays English;
// see the missing-key list in the surface's report.
const settingsNavigationLabel = "Settings sections"

// settingsSectionLocalKey is the local key the surface keeps its chosen
// section under (ui.Local). It is a type, not a string, so the state cannot
// collide with another element's.
type settingsSectionLocalKey struct{}

// settingsLocalNote says where these preferences live. The surface never sends
// them to the App Server: none of them is a session, run or approval fact.
const settingsLocalNote = "These preferences live on this desktop. They change how the shell draws, not any session, run or approval."

// settingsSkinNote says why the skin section has no control to offer. A skin
// or a chrome theme replaces the semantic tokens themselves; system, light and
// dark are color preferences, not stand-ins for one.
const settingsSkinNote = "Skins and chrome themes are not implemented: the shell draws one semantic token set in system, light or dark. No fake skin is offered."

// The reasons the skin and chrome entries wait on. dim-electron resolves skins
// from a skin.json and a window region, and chrome from six historical
// stylesheets (styles/retro-themes.css); this shell has neither, so the entries
// stay visible, unavailable and explained instead of pretending a theme.
const (
	reasonSkin   = "Unavailable: skins need dim-electron's skin.json resolver and window regions; the theme row chooses only system, light or dark."
	reasonChrome = "Unavailable: chrome themes need dim-electron's six historical stylesheets; this shell does not draw a fake chrome."
)

// settingsSectionGaps says why a section other than Appearance has no page
// here: the client or page dim-electron opens for it is missing. The reason
// names what is absent, so a reader sees a WIP page rather than an empty one
// whose settings appear lost or unset.
var settingsSectionGaps = map[SettingsSection]string{
	SettingsSectionAccount:         "Unavailable: the shell has no account client or page yet, so it cannot show an account or sign in.",
	SettingsSectionUsage:           "Unavailable: the shell has no usage client or page yet; no App Server method reports usage here.",
	SettingsSectionPersonalization: "Unavailable: the personalization page is not built yet; the shell has no client for the interface-mode, ACP-session, target-app, assistant-model or subagent-model settings it shows.",
	SettingsSectionGeneral:         "Unavailable: the general settings page is not built yet; the shell has no client for its desktop preferences.",
	SettingsSectionShortcuts:       "Unavailable: the shortcuts page is not built yet; the shell has no keybinding client or editor.",
	SettingsSectionModels:          "Unavailable: the models page is not built yet; the shell has no model-provider client or page.",
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
}

// SettingsSection is one page of the settings surface. The zero value is the
// appearance page, which is the one the shell can draw today: a saved view that
// names no section opens there, and a window that has never saved a section
// draws it without extra state.
//
// The numbers are not the navigation's order; settingsSections is, so that the
// reference's 17 entries keep their order while the zero value keeps the built
// page.
type SettingsSection uint8

const (
	// SettingsSectionAppearance is the color, type, layout, skin and language
	// page. It is the zero value and the surface's default.
	SettingsSectionAppearance SettingsSection = iota
	// SettingsSectionAccount is the account and sign-in page.
	SettingsSectionAccount
	// SettingsSectionUsage is the usage and spend page.
	SettingsSectionUsage
	// SettingsSectionPersonalization is the interface-mode and workspace page.
	SettingsSectionPersonalization
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
)

// settingsSections is the navigation's order, as the reference's left column
// lists it (docs/l5-reference.md §5.5): Account, Usage, Personalization,
// Appearance, General, Shortcuts, Models, Multimodal, Runtime, MCP,
// Notifications, Pet, Permissions, Git, Hooks, Plugins, External Agent.
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
}

// resolvedSection folds an unknown saved section onto the default, as
// resolveSettings does for the enum preferences. Every value from Appearance
// through ExternalAgent is valid.
func resolvedSection(s SettingsSection) SettingsSection {
	if int(s) > int(SettingsSectionExternalAgent) {
		return SettingsSectionAppearance
	}
	return s
}

// settingsSectionName is the section's English name, used for the page heading
// and the navigation row. Only Appearance has a message key today; the other
// pages are not built, so their names stay English until i18n.go carries the
// settings.section.* keys.
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
	default:
		return tr(DefaultLocale, keySettingsTitle)
	}
}

// settingsSectionLabel is the section's name in the active language. The
// appearance page keeps its existing key, so a Chinese surface names it 外观;
// the pages without keys stay English rather than showing an internal key.
func settingsSectionLabel(l Locale, s SettingsSection) string {
	if s == SettingsSectionAppearance {
		return tr(l, keySettingsTitle)
	}
	return tr(l, "settings.section."+settingsSectionSlug(s))
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

// SettingsView is what the settings surface draws: the desktop's own
// preference values, the interface language among them. It is a display copy.
// The window owns the preferences and persists them; no field is a session, run
// or approval fact and none comes from the App Server.
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
	// preference and not a session fact; a zero value opens the appearance
	// page.
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

// defaultSettingsView is the appearance the surface opens with: the theme
// follows the desktop, the type sits on the tokens' own default sizes, the
// sidebar is its default width, and the density is regular. It is the same
// value resolveSettings derives from a zero view.
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

// themeIndex is the selected segment of the theme control: System, Light, Dark.
func themeIndex(theme ThemeChoice) int {
	switch theme {
	case ThemeLight:
		return 1
	case ThemeDark:
		return 2
	default:
		return 0
	}
}

// themeAt is the choice a theme segment stands for.
func themeAt(index int) ThemeChoice {
	switch index {
	case 1:
		return ThemeLight
	case 2:
		return ThemeDark
	default:
		return ThemeSystem
	}
}

// fontFamilyIndex is the selected segment of the family control: System font,
// Monospace.
func fontFamilyIndex(family FontFamily) int {
	if family == FontMonospace {
		return 1
	}
	return 0
}

// fontFamilyAt is the family a segment stands for.
func fontFamilyAt(index int) FontFamily {
	if index == 1 {
		return FontMonospace
	}
	return FontSystem
}

// densityIndex is the selected segment of the density control: Compact,
// Regular.
func densityIndex(density Density) int {
	if density == DensityCompact {
		return 0
	}
	return 1
}

// densityAt is the density a segment stands for.
func densityAt(index int) Density {
	if index == 0 {
		return DensityCompact
	}
	return DensityRegular
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

// densityText is the density's name in the active language.
func densityText(l Locale, d Density) string {
	if d == DensityCompact {
		return tr(l, keySettingsDensityCompact)
	}
	return tr(l, keySettingsDensityRegular)
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

// renderSettings draws the settings surface: the section navigation on the
// left and the chosen section's page on the right. The navigation is one
// focusable row per section, marked selected for the page on screen; a click
// moves the surface's own *Section and reports no SettingsChange, because the
// page is display state, not a desktop preference.
//
// The appearance page draws the theme, type, layout, skin and language
// sections, one row per preference with its current value and the control that
// changes it. Every other page keeps its navigation row and explains the client
// or page it waits on, with a visible WIP badge and no invented control. It
// reads SettingsView and the tokens only. It calls onChange at most once per
// user choice, with the resolved view, and writes no preference itself. The
// surface's own text is drawn in the language the view names, so a language
// choice shows on the next frame.
func renderSettings(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	v = resolveSettings(k, v)
	l := v.Locale
	row := ui.Row(c).Fill().Role(ui.RoleRegion).Label(tr(l, keyHeaderSettings))
	// The chosen section is the surface's own state: the view names where the
	// surface opens, and ui.Local keeps the reader's click from frame to frame.
	// A frame that does not build this row drops it, so reopening the surface
	// starts on the section the view names.
	section := ui.Local(row, settingsSectionLocalKey{}, func() SettingsSection { return v.Section })
	if section == nil {
		opened := v.Section
		section = &opened
	}
	*section = resolvedSection(*section)
	row.Children(func() {
		settingsNavigation(c, k, l, section)
		settingsPage(c, k, v, *section, onChange)
	})
}

// settingsNavigation draws the section column: every section the reference
// lists, in its order, as one focusable row. The selected row carries the
// accent face and the Selected state; a click only moves *section.
func settingsNavigation(c *ui.Context, k tokens, l Locale, section *SettingsSection) {
	nav := ui.Column(c).BasisPercent(settingsNavShare).Shrink(0).MaxWidth(settingsNavW).
		FillHeight().Padding(k.gapWide, k.gapWide).Gap(0).Clip().
		BorderEndWidth(k.hairline).BorderColor(k.border).
		Role(ui.RoleNavigation).Label(tr(l, keySettingsNavigation))
	nav.Children(func() {
		for _, s := range settingsSections {
			selected := s == *section
			entry := ui.ButtonBase(c).Height(k.sidebarEntryH).FillWidth().
				PaddingX(k.sidebarPadX).Radius(radiusMD).Gap(k.gap).
				AlignItems(ui.Center).Role(ui.RoleListItem).
				Selected(selected)
			switch {
			case selected:
				entry.Background(k.accent)
			case entry.Hovered():
				entry.Background(k.sidebarHover)
			}
			entry.OnClick(func() {
				*section = s
			})
			entry.Children(func() {
				text := ui.Text(c, settingsSectionLabel(l, s)).FontSize(k.bodySize).
					TextColor(k.foreground).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
				if selected {
					text.Bold()
				}
			})
		}
	})
}

// settingsPage draws the right-hand page. The appearance page is the only one
// with controls; every other page draws its title, a WIP badge and the client
// or page it waits on, so the section never pretends a preference was read.
func settingsPage(c *ui.Context, k tokens, v SettingsView, section SettingsSection, onChange func(SettingsChange)) {
	l := v.Locale
	ui.Scroll(c).Grow(1).FillHeight().MinWidth(0).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(settingsPageMaxW).Padding(k.panelPad).Gap(k.panelPad).
			Children(func() {
				ui.Text(c, settingsSectionLabel(l, section)).Bold().FontSize(k.titleSize).
					TextColor(k.foreground).Role(ui.RoleHeading).Level(1)
				if section != SettingsSectionAppearance {
					settingsWIPPage(c, k, section, v.Locale)
					return
				}
				ui.Text(c, tr(l, keySettingsLocalNote)).FontSize(k.smallSize).TextColor(k.mutedForeground)
				settingsDivider(c, k)
				settingsThemeSection(c, k, v, onChange)
				settingsDivider(c, k)
				settingsTypeSection(c, k, v, onChange)
				settingsDivider(c, k)
				settingsLayoutSection(c, k, v, onChange)
				settingsDivider(c, k)
				settingsSkinSection(c, k, v)
				// The language row sits last so a reader who changes it has already
				// passed the preferences whose labels it switches; the rest of the page
				// is redrawn in the new language on the next frame.
				settingsDivider(c, k)
				settingsLanguageSection(c, k, v, onChange)
			})
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

// settingsThemeSection is the color-theme row: one choice among System, Light
// and Dark, with the chosen one named beside it.
func settingsThemeSection(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	l := v.Locale
	settingsSection(c, k, tr(l, keySettingsColor), func() {
		settingsRow(c, k, tr(l, keySettingsTheme), tr(l, keySettingsCurrentPrefix)+themeChoiceText(l, v.Theme), func() {
			selected := themeIndex(v.Theme)
			segmented := ui.Segmented(c, &selected,
				tr(l, keySettingsSystem), tr(l, keySettingsLight), tr(l, keySettingsDark)).Label(tr(l, keySettingsTheme))
			if segmented.Changed() {
				next := v
				next.Theme = themeAt(selected)
				publishSettings(onChange, SettingsFieldTheme, next)
			}
		})
	})
}

// settingsTypeSection is the type set: the UI font-size steps, the family, and
// the code size. The UI steps are a radio group, so the arrows step through
// them and wrap at either end; the code size steps by one between its bounds.
func settingsTypeSection(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	l := v.Locale
	settingsSection(c, k, tr(l, keySettingsText), func() {
		settingsRow(c, k, tr(l, keySettingsFontSize), tr(l, keySettingsCurrentPrefix)+settingsPx(v.FontSize), func() {
			labels := make([]string, len(fontSizeSteps))
			for i, step := range fontSizeSteps {
				labels[i] = strconv.Itoa(int(step))
			}
			selected := fontStepIndex(v.FontSize)
			segmented := ui.Segmented(c, &selected, labels...).Label(tr(l, keySettingsFontSize))
			if segmented.Changed() {
				next := v
				next.FontSize = fontSizeSteps[selected]
				publishSettings(onChange, SettingsFieldFontSize, next)
			}
		})
		settingsRow(c, k, tr(l, keySettingsFontFamily), tr(l, keySettingsCurrentPrefix)+fontFamilyText(l, v.FontFamily), func() {
			selected := fontFamilyIndex(v.FontFamily)
			segmented := ui.Segmented(c, &selected,
				tr(l, keySettingsFontSystem), tr(l, keySettingsFontMonospace)).Label(tr(l, keySettingsFontFamily))
			if segmented.Changed() {
				next := v
				next.FontFamily = fontFamilyAt(selected)
				publishSettings(onChange, SettingsFieldFontFamily, next)
			}
		})
		settingsRow(c, k, tr(l, keySettingsCodeFontSize), tr(l, keySettingsCurrentPrefix)+settingsPx(v.CodeFontSize), func() {
			ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
				smaller := ui.Button(c, tr(l, keySettingsSmallerCode)).Disabled(v.CodeFontSize <= settingsCodeMin)
				smaller.OnClick(func() {
					next := v
					next.CodeFontSize = clampCodeFontSize(v.CodeFontSize - settingsCodeStep)
					if next.CodeFontSize != v.CodeFontSize {
						publishSettings(onChange, SettingsFieldCodeFontSize, next)
					}
				})
				larger := ui.Button(c, tr(l, keySettingsLargerCode)).Disabled(v.CodeFontSize >= settingsCodeMax)
				larger.OnClick(func() {
					next := v
					next.CodeFontSize = clampCodeFontSize(v.CodeFontSize + settingsCodeStep)
					if next.CodeFontSize != v.CodeFontSize {
						publishSettings(onChange, SettingsFieldCodeFontSize, next)
					}
				})
			})
		})
	})
}

// settingsLayoutSection is the room the shell keeps: the sidebar's width, one
// step per press and held to its bounds, and the compact or regular density.
func settingsLayoutSection(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	l := v.Locale
	settingsSection(c, k, tr(l, keySettingsLayout), func() {
		settingsRow(c, k, tr(l, keySettingsSidebarWidth), tr(l, keySettingsCurrentPrefix)+settingsPx(v.SidebarWidth), func() {
			ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
				narrower := ui.Button(c, tr(l, keySettingsNarrowerSidebar)).Disabled(v.SidebarWidth <= k.sidebarMinW)
				narrower.OnClick(func() {
					next := v
					next.SidebarWidth = k.clampSidebarWidth(v.SidebarWidth - settingsWidthStep)
					if next.SidebarWidth != v.SidebarWidth {
						publishSettings(onChange, SettingsFieldSidebarWidth, next)
					}
				})
				wider := ui.Button(c, tr(l, keySettingsWiderSidebar)).Disabled(v.SidebarWidth >= k.sidebarMaxW)
				wider.OnClick(func() {
					next := v
					next.SidebarWidth = k.clampSidebarWidth(v.SidebarWidth + settingsWidthStep)
					if next.SidebarWidth != v.SidebarWidth {
						publishSettings(onChange, SettingsFieldSidebarWidth, next)
					}
				})
			})
		})
		settingsRow(c, k, tr(l, keySettingsDensity), tr(l, keySettingsCurrentPrefix)+densityText(l, v.Density), func() {
			selected := densityIndex(v.Density)
			segmented := ui.Segmented(c, &selected,
				tr(l, keySettingsDensityCompact), tr(l, keySettingsDensityRegular)).Label(tr(l, keySettingsDensity))
			if segmented.Changed() {
				next := v
				next.Density = densityAt(selected)
				publishSettings(onChange, SettingsFieldDensity, next)
			}
		})
	})
}

// settingsSkinSection names the appearance layer the shell does not have. Its
// two entries stay visible, unavailable and explained; the surface has no
// third color theme and no skin to preview.
func settingsSkinSection(c *ui.Context, k tokens, v SettingsView) {
	l := v.Locale
	settingsSection(c, k, tr(l, keySettingsSkinSection), func() {
		ui.Text(c, tr(l, keySettingsSkinNote)).FontSize(k.smallSize).TextColor(k.mutedForeground)
		ui.Row(c).Gap(k.gapWide).AlignItems(ui.Center).Children(func() {
			wipAction(c, k, tr(l, keySettingsSkin), tr(l, keyReasonSkin))
			wipAction(c, k, tr(l, keySettingsChrome), tr(l, keyReasonChrome))
		})
	})
}

// settingsLanguageSection is the interface-language row. The control's labels
// are each language's own name, so a reader finds their language written the
// way they write it; choosing one reports SettingsFieldLocale and the shell
// draws the next frame in it.
func settingsLanguageSection(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	l := v.Locale
	settingsSection(c, k, tr(l, keySettingsLanguage), func() {
		settingsRow(c, k, tr(l, keySettingsLanguageRow), tr(l, keySettingsCurrentPrefix)+l.Name(), func() {
			selected := localeIndex(l)
			segmented := ui.Segmented(c, &selected,
				LocaleEN.Name(), LocaleZHCN.Name()).Label(tr(l, keySettingsLanguageRow))
			if segmented.Changed() {
				next := v
				next.Locale = localeAt(selected)
				publishSettings(onChange, SettingsFieldLocale, next)
			}
		})
	})
}

// settingsSection draws one titled group of setting rows.
func settingsSection(c *ui.Context, k tokens, title string, body func()) {
	ui.Column(c).FillWidth().Gap(k.gapWide).Children(func() {
		ui.Text(c, title).Bold().FontSize(k.labelSize).TextColor(k.foreground).
			Role(ui.RoleHeading).Level(2)
		body()
	})
}

// settingsRow draws one setting: its label on the left, its current value, and
// the control that changes it on the right. The current value is text, not a
// control, so assistive technology reads the setting and its value before the
// choice.
func settingsRow(c *ui.Context, k tokens, label, current string, control func()) {
	ui.Row(c).FillWidth().Gap(k.gapWide).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(k.bodySize).TextColor(k.foreground).SingleLine()
		ui.Box(c).Grow(1)
		ui.Text(c, current).FontSize(k.smallSize).TextColor(k.mutedForeground).SingleLine()
		control()
	})
}

// settingsDivider is the hairline between two sections.
func settingsDivider(c *ui.Context, k tokens) {
	ui.Box(c).FillWidth().Height(k.hairline).Background(k.border).AccessHidden(true)
}

// settingsPx writes a size for a row's current-value text.
func settingsPx(size float32) string {
	return strconv.Itoa(int(size)) + " px"
}
