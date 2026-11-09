package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// settingsGlyphCases names every glyph the settings route draws, so a new icon
// cannot be added without the size check seeing it.
func settingsGlyphCases() map[string]*ui.SVG {
	return map[string]*ui.SVG{
		"back":            settingsIcons.back,
		"sun":             settingsIcons.sun,
		"moon":            settingsIcons.moon,
		"system":          settingsIcons.system,
		"account":         settingsIcons.account,
		"usage":           settingsIcons.usage,
		"personalization": settingsIcons.personalization,
		"appearance":      settingsIcons.appearance,
		"general":         settingsIcons.general,
		"shortcuts":       settingsIcons.shortcuts,
		"models":          settingsIcons.models,
		"multimodal":      settingsIcons.multimodal,
		"runtime":         settingsIcons.runtime,
		"mcp":             settingsIcons.mcp,
		"notifications":   settingsIcons.notifications,
		"pet":             settingsIcons.pet,
		"permissions":     settingsIcons.permissions,
		"git":             settingsIcons.git,
		"hooks":           settingsIcons.hooks,
		"plugins":         settingsIcons.plugins,
		"externalAgent":   settingsIcons.externalAgent,
		"capabilities":    settingsIcons.capabilities,
	}
}

// TestSettingsGlyphsAreSixteenSquares checks the settings icon set: every glyph
// parses at the 16 DIP step, and every navigation section maps to one of them,
// so no row falls back to text alone.
func TestSettingsGlyphsAreSixteenSquares(t *testing.T) {
	seen := map[*ui.SVG]string{}
	for name, glyph := range settingsGlyphCases() {
		if glyph == nil {
			t.Errorf("glyph %q is not parsed", name)
			continue
		}
		if w, h := glyph.Size(); w != sidebarIconSize || h != sidebarIconSize {
			t.Errorf("glyph %q is %v×%v, want %v×%v", name, w, h, sidebarIconSize, sidebarIconSize)
		}
		if other, ok := seen[glyph]; ok {
			t.Errorf("glyphs %q and %q share one shape, so a section has no icon of its own", other, name)
		}
		seen[glyph] = name
	}
	if len(settingsSections) != 18 {
		t.Errorf("the navigation lists %d sections, want the reference's 17 plus the capability page", len(settingsSections))
	}
	for _, section := range settingsSections {
		glyph := settingsIcon(section)
		if glyph == nil {
			t.Errorf("section %q has no glyph", settingsSectionName(section))
			continue
		}
		if w, h := glyph.Size(); w != sidebarIconSize || h != sidebarIconSize {
			t.Errorf("section %q draws its glyph at %v×%v, want %v×%v", settingsSectionName(section), w, h, sidebarIconSize, sidebarIconSize)
		}
	}
}

// TestSettingsThemeCardsUseTheirOwnMarks checks the three theme cards draw the
// sun, moon and display marks, each parsed at the icon step.
func TestSettingsThemeCardsUseTheirOwnMarks(t *testing.T) {
	for _, choice := range []ThemeChoice{ThemeSystem, ThemeLight, ThemeDark} {
		glyph := settingsThemeIcon(choice)
		if glyph == nil {
			t.Errorf("theme %v has no glyph", choice)
			continue
		}
		if w, h := glyph.Size(); w != sidebarIconSize || h != sidebarIconSize {
			t.Errorf("theme %v draws its glyph at %v×%v, want %v×%v", choice, w, h, sidebarIconSize, sidebarIconSize)
		}
	}
	if settingsIcons.sun == settingsIcons.moon || settingsIcons.moon == settingsIcons.system ||
		settingsIcons.sun == settingsIcons.system {
		t.Error("two theme cards share one glyph")
	}
}
