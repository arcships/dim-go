// Package config stores the desktop's own local state: the color theme, the
// interface and code type sizes, the font family, the sidebar width, the
// density, the window's geometry and the composer drafts.
//
// All of it is desktop-local. The App Server owns every session, run, timeline
// and approval fact, and nothing of that kind is read or written here: a draft
// is text the user typed and has not submitted, keyed by the session id it was
// typed in, not a session fact. This package touches only its own file, never a
// daemon socket, SQLite, JSONL, session file or log.
//
// The file is <user config dir>/dimagent/settings.json, written atomically and
// readable only by the user. A missing file is not an error: a first run opens
// with the shell's own opening values (see Default and DefaultState). A file
// that cannot be read or parsed is returned as an error together with the
// defaults, so the window can report the problem and still open.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	// dirName is the directory under the user's config directory that holds
	// this desktop's own state.
	dirName = "dimagent"
	// fileName is the state file inside dirName.
	fileName = "settings.json"
	// dirPerm and filePerm keep the state private to the user.
	dirPerm  os.FileMode = 0o700
	filePerm os.FileMode = 0o600
	// formatVersion identifies the file's shape. Version 1 held only the
	// appearance preferences; version 2 adds the window geometry and the
	// composer drafts. A load accepts both and ignores fields this version does
	// not know, so a file written by an earlier or later version still yields
	// the state it does know.
	formatVersion = 2
	// MaxDraftBytes bounds one saved draft. Text beyond it is cut at a rune
	// boundary instead of writing an unbounded paste into the file.
	MaxDraftBytes = 16 << 10
	// MaxDrafts bounds how many sessions keep a saved draft.
	MaxDrafts = 100
	// maxDraftTotal bounds the drafts the file holds together, so a user with
	// many sessions cannot grow the preferences file without limit.
	maxDraftTotal = 1 << 20
	// maxDraftSessionID bounds one session id key. App Server ids are far
	// shorter; this only keeps a hand-edited file from making the map grow with
	// keys instead of text.
	maxDraftSessionID = 256
)

// Settings is the appearance preference this desktop keeps: one field per
// ui.SettingsView field, in base types so storage does not depend on the UI
// package. The enum fields use the same numbers the shell's ThemeChoice,
// FontFamily and Density use, 0 being each one's default.
type Settings struct {
	// Theme is the color-theme preference: 0 follows the desktop, 1 light,
	// 2 dark.
	Theme int `json:"theme"`
	// FontSize is the UI text step in DIPs. 0 takes the shell's default.
	FontSize float32 `json:"fontSize"`
	// FontFamily is 0 for the platform's interface font, 1 for monospace.
	FontFamily int `json:"fontFamily"`
	// SidebarWidth is the sidebar's width in DIPs. 0 takes the shell's default.
	SidebarWidth float32 `json:"sidebarWidth"`
	// Density is 0 for regular spacing, 1 for compact.
	Density int `json:"density"`
	// CodeFontSize is the code text size in DIPs. 0 takes the shell's default.
	CodeFontSize float32 `json:"codeFontSize"`
	// Locale is the interface language tag ("en" or "zh-CN"). Empty takes the
	// shell's default language; an unknown tag falls back the same way.
	Locale string `json:"locale,omitempty"`
}

// Window is the desktop window's normal geometry in DIPs, plus whether it was
// maximized. It is window state, not a session fact: nothing here names or
// describes a session.
type Window struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized,omitempty"`
}

// Valid reports whether w is a geometry worth restoring: a window needs a
// positive size, while (0, 0) is a legitimate position. The window layer still
// clamps a valid geometry to the displays that are connected now.
func (w Window) Valid() bool {
	return w.Width > 0 && w.Height > 0
}

// Drafts maps a session id to the composer text the user typed there and has
// not submitted. It is desktop-local state: the App Server never sees this
// file, and a draft is discarded when that session's submission is accepted.
type Drafts map[string]string

// State is everything the desktop keeps in its own file: appearance
// preferences, the window geometry and the per-session drafts.
type State struct {
	Settings Settings
	Window   Window
	Drafts   Drafts
}

// file is the on-disk shape: the format version, the appearance preferences
// embedded so the JSON stays one flat object, and the optional window and
// drafts.
type file struct {
	Version int `json:"version"`
	Settings
	Window *Window `json:"window,omitempty"`
	Drafts Drafts  `json:"drafts,omitempty"`
}

// Default returns the appearance preferences a first run opens with: the zero
// value.
//
// Every field's zero takes the shell's own opening value (SettingsView
// documents it: "A zero field takes the token default, so a window that has
// never saved a preference draws the values the shell opens with"). Keeping the
// defaults here would be a second copy of the shell's numbers, and the UI font
// step is not even a constant: it follows the platform theme's size. The shell
// resolves the zero values to its own tokens, so a first run draws exactly what
// it drew before a preference could be saved.
func Default() Settings {
	return Settings{}
}

// DefaultState returns the whole desktop state a first run opens with. The
// window has no saved geometry yet, so the window layer opens at its reference
// size and centers itself, and no session has a draft.
func DefaultState() State {
	return State{Settings: Default()}
}

// Clone returns a deep copy, so a caller can change drafts without touching the
// state the store still holds.
func (s State) Clone() State {
	out := s
	if s.Drafts == nil {
		return out
	}
	out.Drafts = make(Drafts, len(s.Drafts))
	for id, text := range s.Drafts {
		out.Drafts[id] = text
	}
	return out
}

// DraftFor returns the saved draft for sessionID, or "" when there is none.
func (s State) DraftFor(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	return s.Drafts[sessionID]
}

// SetDraft records text as sessionID's composer draft. Empty or whitespace-only
// text clears the entry, an oversized draft is cut at a rune boundary, and the
// map keeps at most MaxDrafts entries within maxDraftTotal bytes. An empty
// session id is not draft state and is ignored.
func (s *State) SetDraft(sessionID, text string) {
	if sessionID == "" {
		return
	}
	text = normalizeDraft(text)
	if text == "" {
		s.ClearDraft(sessionID)
		return
	}
	if s.Drafts == nil {
		s.Drafts = make(Drafts, 1)
	}
	s.Drafts[sessionID] = text
	s.Drafts = normalizeDrafts(s.Drafts)
}

// ClearDraft forgets sessionID's draft, for example because the App Server
// accepted the submission it held or because the session was deleted.
func (s *State) ClearDraft(sessionID string) {
	if len(s.Drafts) == 0 {
		return
	}
	delete(s.Drafts, sessionID)
	if len(s.Drafts) == 0 {
		s.Drafts = nil
	}
}

// normalizeDraft drops whitespace-only text and bounds one draft's size.
func normalizeDraft(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	if len(text) <= MaxDraftBytes {
		return text
	}
	cut := MaxDraftBytes
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut]
}

// normalizeDrafts applies the per-draft and total bounds deterministically:
// session ids are visited in sorted order, so the same input always keeps the
// same entries. It returns nil for no drafts, which keeps the key out of the
// saved file.
func normalizeDrafts(in Drafts) Drafts {
	if len(in) == 0 {
		return nil
	}
	ids := make([]string, 0, len(in))
	for id := range in {
		if id == "" || len(id) > maxDraftSessionID {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make(Drafts, len(ids))
	total := 0
	for _, id := range ids {
		text := normalizeDraft(in[id])
		if text == "" {
			continue
		}
		if len(out) >= MaxDrafts || total+len(text) > maxDraftTotal {
			break
		}
		out[id] = text
		total += len(text)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Path returns the state file's path:
// <user config dir>/dimagent/settings.json.
func Path() (string, error) {
	// Acceptance runs point the desktop preferences at their own directory so a
	// screenshot never inherits the reviewer's theme or width. A normal launch
	// leaves the variable unset and uses the user's config directory.
	if home := os.Getenv("DIMAGENT_CONFIG_HOME"); home != "" {
		return filepath.Join(home, dirName, fileName), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("settings path: %w", err)
	}
	return filepath.Join(dir, dirName, fileName), nil
}

// Load reads the saved desktop state. A missing file returns DefaultState and a
// nil error, so a first run is not a failure. A file that cannot be read or
// parsed returns DefaultState and the error; the caller reports it and keeps
// working with the defaults instead of failing to open.
func Load() (State, error) {
	path, err := Path()
	if err != nil {
		return DefaultState(), err
	}
	return loadFile(path)
}

// Save writes s to the state file. The write goes through a temporary file in
// the same directory and is renamed over the target, so a reader sees either
// the previous file or the new one, and a failure leaves the previous file in
// place.
func Save(s State) error {
	path, err := Path()
	if err != nil {
		return err
	}
	return saveFile(path, s, os.Rename)
}

// loadFile reads one state file. The defaults are the starting point, so a
// field the file leaves out keeps its default instead of being reset to
// whatever an incomplete file omits. A version 1 file has no window or drafts;
// it loads its appearance preferences and yields the rest as a first run does,
// and the next save rewrites it in the current shape.
func loadFile(path string) (State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultState(), nil
	}
	if err != nil {
		return DefaultState(), fmt.Errorf("read settings %s: %w", path, err)
	}
	f := file{Version: formatVersion, Settings: Default()}
	if err := json.Unmarshal(data, &f); err != nil {
		return DefaultState(), fmt.Errorf("parse settings %s: %w", path, err)
	}
	state := State{Settings: f.Settings, Drafts: normalizeDrafts(f.Drafts)}
	if f.Window != nil {
		state.Window = *f.Window
	}
	return state, nil
}

// saveFile writes s to path through a temporary file and renames it over the
// target. rename is the last step and a seam for tests: when it fails, path
// still holds the previous file and the temporary file is removed.
func saveFile(path string, s State, rename func(oldpath, newpath string) error) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create settings dir %s: %w", dir, err)
	}
	doc := file{Version: formatVersion, Settings: s.Settings, Drafts: normalizeDrafts(s.Drafts)}
	if s.Window.Valid() {
		window := s.Window
		doc.Window = &window
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		// The fields are numbers, strings and a map of strings, so this cannot
		// happen; report it instead of writing a partial file if it ever does.
		return fmt.Errorf("encode settings: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, ".settings-*.json")
	if err != nil {
		return fmt.Errorf("create settings temp file: %w", err)
	}
	tmpName := tmp.Name()
	// Chmod is explicit: CreateTemp already opens the file owner-only, and the
	// preference file must not be wider than that even if that changes.
	if err := tmp.Chmod(filePerm); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("set settings file mode: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write settings temp file: %w", err)
	}
	// Sync before the rename: the replace is atomic, and this also makes the
	// new contents durable before the old file is no longer reachable. A
	// preference is saved once per choice, so the cost is not on a drag path.
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("sync settings temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close settings temp file: %w", err)
	}
	if err := rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("replace settings %s: %w", path, err)
	}
	return nil
}
