package main

import (
	"testing"

	"github.com/arcships/dimagent-mygo/internal/config"
)

// TestStartupWindowRestoresTheSavedGeometry checks that a normal launch opens
// where the window was, clamped to the display that is connected now.
func TestStartupWindowRestoresTheSavedGeometry(t *testing.T) {
	saved := config.Window{X: 120, Y: 90, Width: 900, Height: 640, Maximized: true}
	areas := []workArea{{X: 0, Y: 25, Width: 1440, Height: 875}}
	got, persist := startupWindow(saved, false, 0, 0, areas)
	if !persist {
		t.Fatal("a normal launch must persist the window geometry")
	}
	if got != saved {
		t.Fatalf("startup window = %+v, want the saved %+v", got, saved)
	}
}

// TestStartupWindowClampsAnOffscreenGeometry checks the visible-screen rule: a
// window saved on a display that is no longer connected opens centered on the
// primary one with the saved size, not at coordinates the user cannot reach.
func TestStartupWindowClampsAnOffscreenGeometry(t *testing.T) {
	saved := config.Window{X: 5000, Y: 4000, Width: 800, Height: 600}
	areas := []workArea{{X: 0, Y: 25, Width: 1440, Height: 875}}
	got, persist := startupWindow(saved, false, 0, 0, areas)
	if !persist {
		t.Fatal("a normal launch must persist the window geometry")
	}
	want := config.Window{X: 320, Y: 162, Width: 800, Height: 600}
	if got != want {
		t.Fatalf("clamped window = %+v, want %+v", got, want)
	}
}

// TestStartupWindowUsesTheReferenceSizeWithoutSavedGeometry checks a first run:
// the window opens at its reference size and lets the shell center it.
func TestStartupWindowUsesTheReferenceSizeWithoutSavedGeometry(t *testing.T) {
	got, persist := startupWindow(config.Window{}, false, 0, 0,
		[]workArea{{X: 0, Y: 0, Width: 1440, Height: 900}})
	if !persist {
		t.Fatal("a normal launch must persist the window geometry")
	}
	if got != (config.Window{Width: defaultWindowWidth, Height: defaultWindowHeight}) {
		t.Fatalf("first-run window = %+v, want %dx%d centered", got, defaultWindowWidth, defaultWindowHeight)
	}
}

// TestStartupWindowCapturePositionWins checks the accepted screenshot exception:
// DIMAGENT_WINDOW_X/Y place the window for that launch, ignore the saved
// geometry and are not saved back, so a capture never moves the user's window.
func TestStartupWindowCapturePositionWins(t *testing.T) {
	saved := config.Window{X: 120, Y: 90, Width: 900, Height: 640, Maximized: true}
	got, persist := startupWindow(saved, true, 40, 60,
		[]workArea{{X: 0, Y: 0, Width: 1440, Height: 900}})
	if persist {
		t.Fatal("a capture launch must not save its window geometry")
	}
	want := config.Window{X: 40, Y: 60, Width: defaultWindowWidth, Height: defaultWindowHeight}
	if got != want {
		t.Fatalf("capture window = %+v, want %+v", got, want)
	}
}

// TestClampWindowKeepsGeometryOnItsOwnDisplay checks a window on a display left
// of the primary one: negative coordinates are valid and stay untouched while
// the window fits.
func TestClampWindowKeepsGeometryOnItsOwnDisplay(t *testing.T) {
	saved := config.Window{X: -1200, Y: 100, Width: 1100, Height: 700}
	areas := []workArea{
		{X: 0, Y: 0, Width: 1440, Height: 900},
		{X: -1440, Y: 0, Width: 1440, Height: 900},
	}
	if got := clampWindow(saved, areas); got != saved {
		t.Fatalf("clampWindow = %+v, want the saved %+v", got, saved)
	}
}

// TestClampWindowMovesASpanningWindowToItsMainDisplay checks that a geometry
// overlapping two displays is kept inside the one it overlaps most.
func TestClampWindowMovesASpanningWindowToItsMainDisplay(t *testing.T) {
	saved := config.Window{X: 1300, Y: 100, Width: 800, Height: 600}
	areas := []workArea{
		{X: 0, Y: 0, Width: 1440, Height: 900},
		{X: 1440, Y: 0, Width: 1280, Height: 1024},
	}
	got := clampWindow(saved, areas)
	want := config.Window{X: 1440, Y: 100, Width: 800, Height: 600}
	if got != want {
		t.Fatalf("clampWindow = %+v, want %+v", got, want)
	}
}

// TestClampWindowCapsOversizedGeometry checks a window larger than its display:
// it shrinks to the work area and moves inside it.
func TestClampWindowCapsOversizedGeometry(t *testing.T) {
	saved := config.Window{X: 0, Y: 0, Width: 2000, Height: 1200}
	areas := []workArea{{X: 0, Y: 25, Width: 1440, Height: 875}}
	got := clampWindow(saved, areas)
	want := config.Window{X: 0, Y: 25, Width: 1440, Height: 875}
	if got != want {
		t.Fatalf("clampWindow = %+v, want %+v", got, want)
	}
}

// TestClampWindowWithoutDisplayInformationKeepsTheSavedGeometry checks the
// fallback: when the screen reports no work area, nothing is invented and the
// window layer decides.
func TestClampWindowWithoutDisplayInformationKeepsTheSavedGeometry(t *testing.T) {
	saved := config.Window{X: 20, Y: 30, Width: 800, Height: 600}
	if got := clampWindow(saved, nil); got != saved {
		t.Fatalf("clampWindow without displays = %+v, want %+v", got, saved)
	}
	if got := clampWindow(config.Window{}, []workArea{{Width: 1440, Height: 900}}); got != (config.Window{}) {
		t.Fatalf("clampWindow of an empty geometry = %+v, want the zero value", got)
	}
}
