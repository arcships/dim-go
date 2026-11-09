// This file owns the window's desktop-local appearance preferences: it loads
// them before the first frame and writes them when the settings surface reports
// a choice.
//
// The preferences are the shell's own: theme, type sizes, font family, sidebar
// width and density. The App Server owns every session, run, timeline and
// approval fact, and none of those is read or written here. The shell reports a
// choice into State and has no storage dependency (internal/ui/settings.go
// leaves persisting to the window), so the window draws a frame through
// content, compares the appearance State now holds with the last snapshot, and
// writes the difference. A write failure is reported in ActionError, which the
// header banner shows; the chosen preference stays applied, so a failed save
// never reverts what the user picked or hides the failure.

package main

import (
	"log"

	"github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/config"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// appearance is the window's appearance preference store. It is desktop-local
// state, not session state: the loaded values and the last snapshot the window
// tried to write.
type appearance struct {
	// load and save are the preference file. Nil means config.Load and
	// config.Save; tests replace them.
	load func() (config.Settings, error)
	save func(config.Settings) error
	// last is the snapshot the window last loaded or wrote. It makes an
	// unchanged frame write nothing, and it is updated before a save is
	// attempted, so a failed write is retried on the next choice rather than on
	// every frame.
	last config.Settings
	// err is the failure the window last reported for a failed save. A later
	// successful save clears it, and only while it is still the header's error:
	// another action's failure is left alone.
	err string
}

// content draws one frame of the window and persists the appearance choice the
// surface applied in it. It is the window's content callback; State.View stays
// the shell's own entry, so the shell needs no knowledge of storage. When a
// save changes the header's error line, one more frame is asked for: this one
// has already drawn, and the user must see the failure (or see it clear).
func (a *app) content(c *ui.Context) {
	if a.state == nil {
		return
	}
	a.state.View(c)
	if a.saveAppearance() {
		c.Invalidate()
	}
}

// loadAppearance fills the window's appearance from the saved preferences
// before the first frame. A missing file gives the shell's opening values and
// no error; a damaged file is reported and the same opening values are used,
// so a bad file opens the window instead of failing it.
func (a *app) loadAppearance() {
	if a.state == nil {
		return
	}
	load := a.appearance.load
	if load == nil {
		load = config.Load
	}
	saved, err := load()
	if err != nil {
		log.Printf("dimagent: load appearance preferences: %v", err)
	}
	a.state.Settings = settingsView(saved)
	a.state.SidebarWidth = saved.SidebarWidth
	// The baseline is the snapshot the window now holds, not the raw file: a
	// value the shell folds to one of its choices is not a user choice, and the
	// first frame must not write it back as one.
	a.appearance.last = appearanceSnapshot(a.state)
}

// saveAppearance writes the appearance snapshot State holds when it differs
// from the last one the window loaded or wrote. Nothing is written for a frame
// that changed no preference, so a redraw cannot keep rewriting the file. It
// reports whether the header's error line changed, so content can ask for the
// frame that shows it.
//
// The snapshot is what the appearance surface reported, which is what
// State.applySettings stores. A bare sidebar drag moves State.SidebarWidth
// without a settings choice; it is not written here, because the store holds
// preferences and a drag has no release to save on (the appearance page's own
// width buttons do report one, and that change carries the width).
func (a *app) saveAppearance() bool {
	if a.state == nil {
		return false
	}
	snapshot := appearanceSnapshot(a.state)
	if snapshot == a.appearance.last {
		return false
	}
	a.appearance.last = snapshot
	save := a.appearance.save
	if save == nil {
		save = config.Save
	}
	if err := save(snapshot); err != nil {
		a.appearance.err = "Save appearance preferences: " + err.Error()
		a.state.ActionError = a.appearance.err
		return true
	}
	if a.appearance.err != "" && a.state.ActionError == a.appearance.err {
		a.state.ActionError = ""
		a.appearance.err = ""
		return true
	}
	a.appearance.err = ""
	return false
}

// appearanceSnapshot copies the appearance State holds into the stored shape.
// Only desktop preferences are read: no session, run, timeline or approval
// field of State reaches the file. The width is the one the appearance page
// reports and applySettings applies, the same value the sidebar draws at until
// a drag moves it.
func appearanceSnapshot(state *dimui.State) config.Settings {
	return config.Settings{
		Theme:        int(state.Settings.Theme),
		FontSize:     state.Settings.FontSize,
		FontFamily:   int(state.Settings.FontFamily),
		SidebarWidth: state.Settings.SidebarWidth,
		Density:      int(state.Settings.Density),
		CodeFontSize: state.Settings.CodeFontSize,
	}
}

// settingsView converts the stored preferences into what the shell draws. The
// enum numbers are the shell's own; the surface still folds an unknown value to
// its default when it resolves the view, so a hand-edited file cannot draw a
// choice the appearance page has no control for.
func settingsView(saved config.Settings) dimui.SettingsView {
	return dimui.SettingsView{
		Theme:        dimui.ThemeChoice(saved.Theme),
		FontSize:     saved.FontSize,
		FontFamily:   dimui.FontFamily(saved.FontFamily),
		SidebarWidth: saved.SidebarWidth,
		Density:      dimui.Density(saved.Density),
		CodeFontSize: saved.CodeFontSize,
	}
}
