package ui

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

// lucideRequiredIcons is the set the shell's D3 icon pass asks for by name.
// The test names every one, so dropping or renaming a glyph breaks the build
// even when no view reads it yet.
var lucideRequiredIcons = []string{
	"plus", "search", "clock", "layout-grid", "chevron-right", "folder",
	"user", "settings", "panel-left-close", "list", "pin", "copy", "link",
	"bug", "download", "archive", "square-pen", "trash-2", "chevrons-right",
	"box", "panel-right", "message-square", "terminal", "file-text",
	"git-branch", "shield", "key-round", "bell", "palette", "keyboard", "cpu",
	"plug", "paw-print", "puzzle", "briefcase", "info", "sun", "moon",
	"monitor",
}

// TestLucideIconSetIsComplete checks the parsed set: every required name is
// present, every value parsed, and every glyph is a 16 DIP square. A nil value
// would draw a blank row, so it is a failure here instead of a quiet omission
// in the frame.
func TestLucideIconSetIsComplete(t *testing.T) {
	for _, name := range lucideRequiredIcons {
		icon, ok := lucideIcons[name]
		if !ok {
			t.Errorf("lucide icon %q is missing", name)
			continue
		}
		if icon == nil {
			t.Errorf("lucide icon %q is not parsed", name)
			continue
		}
		if w, h := icon.Size(); w != sidebarIconSize || h != sidebarIconSize {
			t.Errorf("lucide icon %q is %v×%v, want %v×%v", name, w, h, sidebarIconSize, sidebarIconSize)
		}
	}
	if len(lucideIcons) < len(lucideRequiredIcons) {
		t.Fatalf("the set holds %d icons, fewer than the %d required", len(lucideIcons), len(lucideRequiredIcons))
	}
}

// TestLucideIconsDrawInk checks that each glyph renders something: a 16×16
// raster of every icon has at least one pixel with alpha, so a path typo or an
// unsupported shape cannot ship as an invisible decoration.
func TestLucideIconsDrawInk(t *testing.T) {
	for name, icon := range lucideIcons {
		if icon == nil {
			t.Errorf("lucide icon %q is not parsed", name)
			continue
		}
		img := icon.Image(sidebarIconSize, sidebarIconSize, color.Black)
		ink := false
		for i := 3; i < len(img.Pix); i += 4 {
			if img.Pix[i] != 0 {
				ink = true
				break
			}
		}
		if !ink {
			t.Errorf("lucide icon %q draws no ink", name)
		}
	}
}

// TestLucideIconsUseCurrentColor checks that no glyph bakes in a paint: the
// same icon rendered with a red and a blue currentColor must differ, because
// every stroke and fill follows the text color. Two identical rasters mean the
// shapes carry a fixed color and would ignore the theme.
func TestLucideIconsUseCurrentColor(t *testing.T) {
	red := color.RGBA{R: 255, A: 255}
	blue := color.RGBA{B: 255, A: 255}
	for name, icon := range lucideIcons {
		if icon == nil {
			t.Errorf("lucide icon %q is not parsed", name)
			continue
		}
		inRed := icon.Image(sidebarIconSize, sidebarIconSize, red)
		inBlue := icon.Image(sidebarIconSize, sidebarIconSize, blue)
		if len(inRed.Pix) != len(inBlue.Pix) {
			t.Fatalf("lucide icon %q rendered at two sizes", name)
		}
		same := true
		for i := range inRed.Pix {
			if inRed.Pix[i] != inBlue.Pix[i] {
				same = false
				break
			}
		}
		if same {
			t.Errorf("lucide icon %q ignores currentColor", name)
		}
	}
}

// TestLucideIconsAreUnique checks that each key owns its own parsed SVG: two
// keys sharing one value would let a wiring change silently swap two sections'
// marks, and the settings navigation test relies on distinct values.
func TestLucideIconsAreUnique(t *testing.T) {
	seen := map[*ui.SVG]string{}
	for name, icon := range lucideIcons {
		if other, ok := seen[icon]; ok {
			t.Errorf("lucide icons %q and %q share one parsed value", other, name)
		}
		seen[icon] = name
	}
}
