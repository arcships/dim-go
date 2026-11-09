// This file owns the appearance settings surface: the desktop's own color
// theme, type, sidebar-width and density preferences, plus the skin and chrome
// entries the shell cannot offer yet. It reads SettingsView and the tokens only
// (see protocols.go); the App Server owns every session, run, timeline and
// approval fact and is not read here.
//
// These are desktop-local preferences, the kind dim-electron keeps in its own
// config (appearance/index.ts): they change how the shell draws, not what a
// session contains. renderSettings writes nothing itself. It reports one
// SettingsChange per user choice and leaves applying and persisting it to the
// window, so the surface cannot claim a preference the desktop did not save.

package ui

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// The appearance page's own steps, in DIPs. Padding, gaps, type sizes, radii
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
	default:
		return "theme"
	}
}

// SettingsView is what the appearance surface draws: the desktop's own
// preference values. It is a display copy. The window owns the preferences and
// persists them; no field is a session, run or approval fact and none comes
// from the App Server.
//
// A zero field takes the token default, so a window that has never saved a
// preference draws the values the shell opens with. resolveSettings fills the
// defaults and keeps a saved value on the range the surface offers, and
// newTokensFor folds the resolved view into the tokens the frame draws with.
type SettingsView struct {
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
// surface offers: a width is clamped and a size snapped before it is reported,
// so a saved value outside the bounds cannot leak back through a click.
type SettingsChange struct {
	Field        SettingsField
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
// enum falls back to its default, a UI size snaps to the nearest offered step,
// the code size and the sidebar width are held to their ranges, and a zero
// value takes the token default.
func resolveSettings(k tokens, v SettingsView) SettingsView {
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

// publishSettings reports one user choice. The nil callback is a window that
// has not wired persistence yet: the surface stays visible with its current
// values and reports nothing, rather than changing a preference nowhere.
func publishSettings(onChange func(SettingsChange), field SettingsField, v SettingsView) {
	if onChange == nil {
		return
	}
	onChange(SettingsChange{
		Field:        field,
		Theme:        v.Theme,
		FontSize:     v.FontSize,
		FontFamily:   v.FontFamily,
		SidebarWidth: v.SidebarWidth,
		Density:      v.Density,
		CodeFontSize: v.CodeFontSize,
	})
}

// renderSettings draws the appearance settings surface: the title, the theme,
// type, layout and skin sections, one row per preference with its current value
// and the control that changes it. It reads SettingsView and the tokens only.
// It calls onChange at most once per user choice, with the resolved view, and
// writes no preference itself. The skin and chrome entries stay visible and
// unavailable.
func renderSettings(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	v = resolveSettings(k, v)
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().MaxWidth(settingsPageMaxW).Padding(k.panelPad).Gap(k.panelPad).
			Role(ui.RoleRegion).Label("Appearance settings").Children(func() {
			ui.Text(c, "Appearance").Bold().FontSize(k.titleSize).TextColor(k.foreground).
				Role(ui.RoleHeading).Level(1)
			ui.Text(c, settingsLocalNote).FontSize(k.smallSize).TextColor(k.mutedForeground)
			settingsDivider(c, k)
			settingsThemeSection(c, k, v, onChange)
			settingsDivider(c, k)
			settingsTypeSection(c, k, v, onChange)
			settingsDivider(c, k)
			settingsLayoutSection(c, k, v, onChange)
			settingsDivider(c, k)
			settingsSkinSection(c, k)
		})
	})
}

// settingsThemeSection is the color-theme row: one choice among System, Light
// and Dark, with the chosen one named beside it.
func settingsThemeSection(c *ui.Context, k tokens, v SettingsView, onChange func(SettingsChange)) {
	settingsSection(c, k, "Color", func() {
		settingsRow(c, k, "Theme", "Current: "+v.Theme.String(), func() {
			selected := themeIndex(v.Theme)
			segmented := ui.Segmented(c, &selected,
				ThemeSystem.String(), ThemeLight.String(), ThemeDark.String()).Label("Theme")
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
	settingsSection(c, k, "Text", func() {
		settingsRow(c, k, "Font size", "Current: "+settingsPx(v.FontSize), func() {
			labels := make([]string, len(fontSizeSteps))
			for i, step := range fontSizeSteps {
				labels[i] = strconv.Itoa(int(step))
			}
			selected := fontStepIndex(v.FontSize)
			segmented := ui.Segmented(c, &selected, labels...).Label("Font size")
			if segmented.Changed() {
				next := v
				next.FontSize = fontSizeSteps[selected]
				publishSettings(onChange, SettingsFieldFontSize, next)
			}
		})
		settingsRow(c, k, "Font family", "Current: "+v.FontFamily.String(), func() {
			selected := fontFamilyIndex(v.FontFamily)
			segmented := ui.Segmented(c, &selected,
				FontSystem.String(), FontMonospace.String()).Label("Font family")
			if segmented.Changed() {
				next := v
				next.FontFamily = fontFamilyAt(selected)
				publishSettings(onChange, SettingsFieldFontFamily, next)
			}
		})
		settingsRow(c, k, "Code font size", "Current: "+settingsPx(v.CodeFontSize), func() {
			ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
				smaller := ui.Button(c, "Smaller code").Disabled(v.CodeFontSize <= settingsCodeMin)
				smaller.OnClick(func() {
					next := v
					next.CodeFontSize = clampCodeFontSize(v.CodeFontSize - settingsCodeStep)
					if next.CodeFontSize != v.CodeFontSize {
						publishSettings(onChange, SettingsFieldCodeFontSize, next)
					}
				})
				larger := ui.Button(c, "Larger code").Disabled(v.CodeFontSize >= settingsCodeMax)
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
	settingsSection(c, k, "Layout", func() {
		settingsRow(c, k, "Sidebar width", "Current: "+settingsPx(v.SidebarWidth), func() {
			ui.Row(c).Gap(k.gap).AlignItems(ui.Center).Children(func() {
				narrower := ui.Button(c, "Narrower sidebar").Disabled(v.SidebarWidth <= k.sidebarMinW)
				narrower.OnClick(func() {
					next := v
					next.SidebarWidth = k.clampSidebarWidth(v.SidebarWidth - settingsWidthStep)
					if next.SidebarWidth != v.SidebarWidth {
						publishSettings(onChange, SettingsFieldSidebarWidth, next)
					}
				})
				wider := ui.Button(c, "Wider sidebar").Disabled(v.SidebarWidth >= k.sidebarMaxW)
				wider.OnClick(func() {
					next := v
					next.SidebarWidth = k.clampSidebarWidth(v.SidebarWidth + settingsWidthStep)
					if next.SidebarWidth != v.SidebarWidth {
						publishSettings(onChange, SettingsFieldSidebarWidth, next)
					}
				})
			})
		})
		settingsRow(c, k, "Density", "Current: "+v.Density.String(), func() {
			selected := densityIndex(v.Density)
			segmented := ui.Segmented(c, &selected,
				DensityCompact.String(), DensityRegular.String()).Label("Density")
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
func settingsSkinSection(c *ui.Context, k tokens) {
	settingsSection(c, k, "Skin and chrome", func() {
		ui.Text(c, settingsSkinNote).FontSize(k.smallSize).TextColor(k.mutedForeground)
		ui.Row(c).Gap(k.gapWide).AlignItems(ui.Center).Children(func() {
			wipAction(c, k, "Skin", reasonSkin)
			wipAction(c, k, "Chrome theme", reasonChrome)
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
