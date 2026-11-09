// This file owns the window's desktop-local geometry: the position, size and
// maximized state it restores at launch and saves when it changes.
//
// The geometry is desktop state only. It never records the settings route, the
// selected session or any other window view state, so reopening the app restores
// where the window was, not what it showed.

package main

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/config"
)

const (
	// defaultWindowWidth and defaultWindowHeight are the reference window size
	// in points: the sidebar's 256 DIP is the same, so the proportion of chrome
	// matches the desktop app.
	defaultWindowWidth  = 800
	defaultWindowHeight = 720
	// windowSaveDelay debounces resize and move events. A drag produces many
	// events, and the file is written once after the bounds settle.
	windowSaveDelay = 300 * time.Millisecond
)

// windowBookkeeping is the app's window geometry state. normal is the geometry
// the window returns to when it leaves the maximized state, persist tells that
// this launch should save the geometry at all (a capture run does not), and
// timer is the pending debounced save.
type windowBookkeeping struct {
	mu      sync.Mutex
	normal  config.Window
	persist bool
	timer   *time.Timer
}

// workArea is one display's usable area in DIPs. It keeps the clamping pure and
// testable without a native screen.
type workArea struct {
	X      int
	Y      int
	Width  int
	Height int
}

// workAreas converts the connected displays into their work areas, primary
// first, as mygo reports them.
func workAreas(displays []mygo.Display) []workArea {
	out := make([]workArea, 0, len(displays))
	for _, display := range displays {
		area := display.WorkArea
		if area.Width <= 0 || area.Height <= 0 {
			continue
		}
		out = append(out, workArea{X: area.X, Y: area.Y, Width: area.Width, Height: area.Height})
	}
	return out
}

// startupWindow decides the geometry the window opens with. A capture position
// (DIMAGENT_WINDOW_X/Y) wins for that launch and is not saved back, so an
// acceptance screenshot never moves the user's window state. Otherwise a saved
// geometry is restored inside the display it overlaps most; no saved geometry
// opens at the reference size and lets the shell center it.
func startupWindow(saved config.Window, capture bool, captureX, captureY int, areas []workArea) (config.Window, bool) {
	if capture {
		return config.Window{X: captureX, Y: captureY, Width: defaultWindowWidth, Height: defaultWindowHeight}, false
	}
	if !saved.Valid() {
		return config.Window{Width: defaultWindowWidth, Height: defaultWindowHeight}, true
	}
	return clampWindow(saved, areas), true
}

// clampWindow keeps a saved geometry on a connected display. It stays where it
// was when it still fits that display's work area, and is otherwise moved inside
// it; a geometry that overlaps no display is centered on the primary one. The
// size is capped to the work area, and a zero size still means "nothing saved".
func clampWindow(saved config.Window, areas []workArea) config.Window {
	if !saved.Valid() || len(areas) == 0 {
		return saved
	}
	area := areas[0]
	best := -1
	for _, candidate := range areas {
		overlap := overlapArea(saved, candidate)
		if overlap > best {
			best = overlap
			area = candidate
		}
	}
	saved.Width = min(saved.Width, area.Width)
	saved.Height = min(saved.Height, area.Height)
	if best <= 0 {
		// Off every display: center on the primary one instead of restoring a
		// position the user cannot see.
		saved.X = area.X + (area.Width-saved.Width)/2
		saved.Y = area.Y + (area.Height-saved.Height)/2
		return saved
	}
	saved.X = clamp(saved.X, area.X, area.X+area.Width-saved.Width)
	saved.Y = clamp(saved.Y, area.Y, area.Y+area.Height-saved.Height)
	return saved
}

// overlapArea returns how much of window overlaps area, in square DIPs.
func overlapArea(window config.Window, area workArea) int {
	width := min(window.X+window.Width, area.X+area.Width) - max(window.X, area.X)
	height := min(window.Y+window.Height, area.Y+area.Height) - max(window.Y, area.Y)
	if width <= 0 || height <= 0 {
		return 0
	}
	return width * height
}

// windowOptions builds the options for the window this launch opens and records
// the geometry it starts with, including whether this launch persists it. It
// runs on the main thread after the App is ready, so the display list is
// available.
func (a *app) windowOptions() mygo.WindowOptions {
	saved := config.Window{}
	if a.desktop != nil {
		saved = a.desktop.current().Window
	}
	_, envX := os.LookupEnv("DIMAGENT_WINDOW_X")
	_, envY := os.LookupEnv("DIMAGENT_WINDOW_Y")
	geometry, persist := startupWindow(saved, envX || envY,
		envInt("DIMAGENT_WINDOW_X"), envInt("DIMAGENT_WINDOW_Y"),
		workAreas(mygo.Screen.Displays()))

	a.window.mu.Lock()
	a.window.normal = geometry
	a.window.persist = persist
	a.window.mu.Unlock()

	return mygo.WindowOptions{
		Title:     "DimAgent",
		Width:     geometry.Width,
		Height:    geometry.Height,
		X:         geometry.X,
		Y:         geometry.Y,
		Maximized: geometry.Maximized,
		// The reference hides the native title bar: the shell draws from y=0
		// and the traffic lights float over the sidebar. Without this the
		// window keeps a dark system title bar above the light shell.
		TitleBarStyle: mygo.TitleBarHiddenInset,
		// content draws the shell and then writes an appearance choice the
		// surface applied during that frame.
		Content: ui.View(a.content),
	}
}

// bindWindow records the window's geometry when it moves, resizes or changes
// its maximized state, and saves it once more when the window closes. The
// close handler also saves the composer draft, so closing the window keeps both
// pieces of desktop state even before the process exits.
func (a *app) bindWindow() {
	if a.win == nil {
		return
	}
	a.win.OnClose(func(*mygo.CloseEvent) {
		a.saveWindowNow()
		a.saveCurrentDraft()
	})
	if !a.windowPersists() {
		return
	}
	a.win.OnResize(a.scheduleWindowSave)
	a.win.OnMove(a.scheduleWindowSave)
	a.win.OnMaximize(a.scheduleWindowSave)
	a.win.OnUnmaximize(a.scheduleWindowSave)
	a.win.OnRestore(a.scheduleWindowSave)
	a.win.OnEnterFullScreen(a.scheduleWindowSave)
	a.win.OnLeaveFullScreen(a.scheduleWindowSave)
}

// windowPersists reports whether this launch saves the window geometry.
func (a *app) windowPersists() bool {
	a.window.mu.Lock()
	defer a.window.mu.Unlock()
	return a.window.persist
}

// scheduleWindowSave debounces a bounds change into one write.
func (a *app) scheduleWindowSave() {
	if !a.windowPersists() {
		return
	}
	a.window.mu.Lock()
	defer a.window.mu.Unlock()
	if a.window.timer == nil {
		a.window.timer = time.AfterFunc(windowSaveDelay, a.saveWindowNow)
		return
	}
	a.window.timer.Reset(windowSaveDelay)
}

// stopWindowTimer cancels a pending debounced save at shutdown, so no timer
// writes after the final save.
func (a *app) stopWindowTimer() {
	a.window.mu.Lock()
	defer a.window.mu.Unlock()
	if a.window.timer != nil {
		a.window.timer.Stop()
		a.window.timer = nil
	}
}

// saveWindowNow writes the window's current normal geometry. While maximized
// the visible bounds are the screen's, so the normal geometry captured before
// maximizing is kept and only the maximized flag changes.
func (a *app) saveWindowNow() {
	if !a.windowPersists() || a.desktop == nil || a.win == nil {
		return
	}
	window := a.captureWindow()
	if !window.Valid() {
		return
	}
	if err := a.desktop.setWindow(window); err != nil {
		log.Printf("dimagent: save window state: %v", err)
	}
}

// captureWindow updates and returns the window's normal geometry. A destroyed
// or not-yet-shown window has no bounds to save, so the last geometry is kept.
func (a *app) captureWindow() config.Window {
	if a.win == nil || a.win.IsDestroyed() {
		a.window.mu.Lock()
		defer a.window.mu.Unlock()
		return a.window.normal
	}
	maximized := a.win.IsMaximized()
	fullScreen := a.win.IsFullScreen()
	bounds := a.win.Bounds()
	a.window.mu.Lock()
	defer a.window.mu.Unlock()
	if fullScreen {
		// Full screen is not a saved geometry: keep the normal bounds and the
		// maximized flag the window had before it entered full screen.
		return a.window.normal
	}
	if maximized {
		a.window.normal.Maximized = true
		return a.window.normal
	}
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return a.window.normal
	}
	a.window.normal = config.Window{
		X:      bounds.X,
		Y:      bounds.Y,
		Width:  bounds.Width,
		Height: bounds.Height,
	}
	return a.window.normal
}

func clamp(value, low, high int) int {
	if high < low {
		return low
	}
	return min(max(value, low), high)
}
