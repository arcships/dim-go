// This file owns the desktop's local state store and the two things the window
// keeps in it besides appearance preferences: the composer draft of each
// session and the window geometry.
//
// None of it is session truth. The App Server owns sessions, runs, timelines
// and approvals; a draft is only text the user typed and has not submitted, and
// it is dropped as soon as that submission is accepted. The store only reads and
// writes <user config dir>/dimagent/settings.json through internal/config. It
// never opens a daemon socket, SQLite, JSONL, a session file or a log.

package main

import (
	"log"
	"reflect"
	"strings"
	"sync"

	"github.com/arcships/dimagent-mygo/internal/config"
)

// desktopStore serializes the desktop-local file. Every change loads once,
// edits a deep copy of the current state, writes the whole state atomically and
// only then publishes it, so a failed save leaves both the file and the state
// the window reads unchanged.
type desktopStore struct {
	mu sync.Mutex
	// load and save are the file. Nil load means "open with the defaults", nil
	// save means "read only"; tests use both.
	load func() (config.State, error)
	save func(config.State) error
	// state is the current desktop state and loaded tells that load ran once.
	state  config.State
	loaded bool
}

// newDesktopStore returns the store a normal launch uses.
func newDesktopStore() *desktopStore {
	return &desktopStore{load: config.Load, save: config.Save}
}

// loadOnce loads the file the first time and returns the current state after
// that. The caller reports a load error once; the store keeps the defaults a
// failed load returned, so a damaged file still opens the window.
func (d *desktopStore) loadOnce() (config.State, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.loadLocked()
}

// loadLocked implements loadOnce with mu held.
func (d *desktopStore) loadLocked() (config.State, error) {
	if d.loaded {
		return d.state.Clone(), nil
	}
	state := config.DefaultState()
	var err error
	if d.load != nil {
		state, err = d.load()
	}
	d.state = state.Clone()
	d.loaded = true
	return state, err
}

// current returns a copy of the current state, loading the file if no caller
// did yet.
func (d *desktopStore) current() config.State {
	d.mu.Lock()
	defer d.mu.Unlock()
	state, _ := d.loadLocked()
	return state
}

// update applies fn to a copy of the state and saves the result. Nothing is
// written when fn changed nothing, so a repeated selection or a redraw of the
// same draft stays quiet.
func (d *desktopStore) update(fn func(*config.State)) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, err := d.loadLocked(); err != nil {
		// The defaults a failed load returned are the state this change builds
		// on; the load failure was reported when the state was first read.
		log.Printf("dimagent: desktop state was not loaded cleanly: %v", err)
	}
	next := d.state.Clone()
	fn(&next)
	if reflect.DeepEqual(next, d.state) {
		return nil
	}
	if d.save != nil {
		if err := d.save(next); err != nil {
			return err
		}
	}
	d.state = next
	return nil
}

// setSettings replaces the appearance preferences and keeps the window and
// drafts that the file already holds.
func (d *desktopStore) setSettings(settings config.Settings) error {
	return d.update(func(state *config.State) { state.Settings = settings })
}

// setDraft records sessionID's composer draft.
func (d *desktopStore) setDraft(sessionID, text string) error {
	return d.update(func(state *config.State) { state.SetDraft(sessionID, text) })
}

// clearDraft forgets sessionID's draft.
func (d *desktopStore) clearDraft(sessionID string) error {
	return d.update(func(state *config.State) { state.ClearDraft(sessionID) })
}

// setWindow records the window's normal geometry.
func (d *desktopStore) setWindow(window config.Window) error {
	return d.update(func(state *config.State) { state.Window = window })
}

// loadDesktop loads the desktop-local state before the first frame and restores
// the draft of the session the window already selected. A load failure is
// reported here once; the window keeps the defaults a failed load returned.
func (a *app) loadDesktop() {
	if a.desktop == nil {
		a.desktop = newDesktopStore()
	}
	state, err := a.desktop.loadOnce()
	if err != nil {
		log.Printf("dimagent: load desktop state: %v", err)
	}
	if a.state == nil {
		return
	}
	if draft := state.DraftFor(a.state.Selected); draft != "" {
		a.state.Draft = draft
	}
}

// loadSettings is the appearance loader the window uses when a test did not
// supply one.
func (a *app) loadSettings() (config.Settings, error) {
	if a.desktop == nil {
		return config.Default(), nil
	}
	state, err := a.desktop.loadOnce()
	return state.Settings, err
}

// saveSettings writes an appearance choice and keeps the other desktop-local
// state in the file.
func (a *app) saveSettings(settings config.Settings) error {
	if a.desktop == nil {
		return nil
	}
	return a.desktop.setSettings(settings)
}

// rememberDraft saves the composer text the window currently shows under
// sessionID. It runs before a session switch, before the window closes and when
// the app quits. Empty text removes the entry instead of saving a blank draft.
func (a *app) rememberDraft(sessionID string) {
	if a.desktop == nil || a.state == nil || sessionID == "" {
		return
	}
	if err := a.desktop.setDraft(sessionID, a.state.Draft); err != nil {
		log.Printf("dimagent: save draft for session %q: %v", sessionID, err)
	}
}

// draftFor returns the saved draft of sessionID, or "" when it has none.
func (a *app) draftFor(sessionID string) string {
	if a.desktop == nil {
		return ""
	}
	return a.desktop.current().DraftFor(sessionID)
}

// clearDraft forgets sessionID's saved draft after the App Server accepted its
// submission or after the session was deleted. A failure is logged, not
// reported as a session error: the draft is desktop state.
func (a *app) clearDraft(sessionID string) {
	if a.desktop == nil || sessionID == "" {
		return
	}
	if err := a.desktop.clearDraft(sessionID); err != nil {
		log.Printf("dimagent: clear draft for session %q: %v", sessionID, err)
	}
}

// clearDraftIfMatches forgets sessionID's saved draft when it still holds the
// submitted text. The shell submits trimmed text, so a saved draft that trims to
// it is the same draft. Text typed after the send is a newer draft and stays.
func (a *app) clearDraftIfMatches(sessionID, text string) {
	if a.desktop == nil || sessionID == "" {
		return
	}
	if strings.TrimSpace(a.desktop.current().DraftFor(sessionID)) != text {
		return
	}
	a.clearDraft(sessionID)
}

// saveCurrentDraft persists the draft the window shows now.
func (a *app) saveCurrentDraft() {
	if a.state == nil {
		return
	}
	a.rememberDraft(a.state.Selected)
}
