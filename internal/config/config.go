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
//
// The pinned and archived lists are the desktop's own display state: they name
// session ids the sidebar orders first or hides, and they describe no session
// content. The App Server remains the only source of sessions, titles, runs and
// timelines; an id neither list names is simply in its catalog order, and an id
// the catalog no longer holds draws nothing.
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
	// composer drafts; version 3 adds the desktop-local pinned and archived
	// session id lists. A load accepts any version and ignores fields this
	// version does not know, so a file written by an earlier or later version
	// still yields the state it does know.
	formatVersion = 3
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
	// MaxPinnedSessions bounds the pinned list and MaxArchivedSessions the
	// archived one, so a hand-edited file cannot grow either list without
	// limit. The lists hold ids, not session content, so the bounds are well
	// above any catalog a desktop window shows at once.
	MaxPinnedSessions   = 200
	MaxArchivedSessions = 500
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
// preferences, the window geometry, the per-session drafts and the display-only
// pinned and archived session id lists.
type State struct {
	Settings Settings
	Window   Window
	Drafts   Drafts
	// Pinned and Archived name session ids the sidebar orders first or hides.
	// They are desktop-local display state: they carry no title, message or run
	// fact, and the App Server remains the source of every session fact.
	Pinned   []string
	Archived []string
}

// file is the on-disk shape: the format version, the appearance preferences
// embedded so the JSON stays one flat object, and the optional window, drafts
// and session id lists.
type file struct {
	Version int `json:"version"`
	Settings
	Window   *Window  `json:"window,omitempty"`
	Drafts   Drafts   `json:"drafts,omitempty"`
	Pinned   []string `json:"pinned,omitempty"`
	Archived []string `json:"archived,omitempty"`
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

// Clone returns a deep copy, so a caller can change drafts or the pinned and
// archived lists without touching the state the store still holds.
func (s State) Clone() State {
	out := s
	if s.Drafts != nil {
		out.Drafts = make(Drafts, len(s.Drafts))
		for id, text := range s.Drafts {
			out.Drafts[id] = text
		}
	}
	out.Pinned = append([]string(nil), s.Pinned...)
	out.Archived = append([]string(nil), s.Archived...)
	return out
}

// IsPinned reports whether sessionID is in the pinned list.
func (s State) IsPinned(sessionID string) bool {
	return containsSessionID(s.Pinned, sessionID)
}

// IsArchived reports whether sessionID is in the archived list.
func (s State) IsArchived(sessionID string) bool {
	return containsSessionID(s.Archived, sessionID)
}

// SetPinned adds or removes sessionID from the pinned list and reports whether
// the state changed. Pinning also unarchives the same id: a session cannot be
// both ordered first and hidden. An empty or oversized id is ignored, a
// duplicate changes nothing, and a full list keeps its existing entries. The
// state is normalized before it is returned, so the list stays deduplicated and
// bounded.
func (s *State) SetPinned(sessionID string, pinned bool) bool {
	if !validSessionID(sessionID) {
		return false
	}
	beforePinned := append([]string(nil), s.Pinned...)
	beforeArchived := append([]string(nil), s.Archived...)
	if pinned {
		s.Archived = removeSessionID(s.Archived, sessionID)
		if len(s.Pinned) < MaxPinnedSessions && !s.IsPinned(sessionID) {
			s.Pinned = append(s.Pinned, sessionID)
		}
	} else {
		s.Pinned = removeSessionID(s.Pinned, sessionID)
	}
	s.Pinned, s.Archived = normalizeHidden(s.Pinned, s.Archived)
	return !equalSessionIDs(beforePinned, s.Pinned) || !equalSessionIDs(beforeArchived, s.Archived)
}

// SetArchived adds or removes sessionID from the archived list and reports
// whether the state changed. Archiving also unpins the same id, because an
// archived session never reaches the sidebar where a pin would order it. An
// empty or oversized id is ignored, a duplicate changes nothing, and a full list
// keeps its existing entries.
func (s *State) SetArchived(sessionID string, archived bool) bool {
	if !validSessionID(sessionID) {
		return false
	}
	beforePinned := append([]string(nil), s.Pinned...)
	beforeArchived := append([]string(nil), s.Archived...)
	if archived {
		s.Pinned = removeSessionID(s.Pinned, sessionID)
		if len(s.Archived) < MaxArchivedSessions && !s.IsArchived(sessionID) {
			s.Archived = append(s.Archived, sessionID)
		}
	} else {
		s.Archived = removeSessionID(s.Archived, sessionID)
	}
	s.Pinned, s.Archived = normalizeHidden(s.Pinned, s.Archived)
	return !equalSessionIDs(beforePinned, s.Pinned) || !equalSessionIDs(beforeArchived, s.Archived)
}

// RemoveSession forgets sessionID from both lists, for example because the App
// Server confirmed the session's delete. It removes display state only; the
// delete itself is the App Server's fact.
func (s *State) RemoveSession(sessionID string) {
	s.Pinned = removeSessionID(s.Pinned, sessionID)
	s.Archived = removeSessionID(s.Archived, sessionID)
	s.Pinned, s.Archived = normalizeHidden(s.Pinned, s.Archived)
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

// validSessionID reports whether id may enter a pinned or archived list. It is
// the same bound the drafts use, so a hand-edited file cannot grow a list with
// keys instead of ids; a whitespace-only id names no session.
func validSessionID(id string) bool {
	return id != "" && len(id) <= maxDraftSessionID && strings.TrimSpace(id) != ""
}

// containsSessionID reports whether ids holds id.
func containsSessionID(ids []string, id string) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

// removeSessionID returns ids without id. It returns nil when the list becomes
// empty, which keeps the key out of the saved file.
func removeSessionID(ids []string, id string) []string {
	out := make([]string, 0, len(ids))
	for _, candidate := range ids {
		if candidate != id {
			out = append(out, candidate)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// equalSessionIDs reports whether two id lists are the same sequence.
func equalSessionIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// normalizeSessionIDs drops invalid and duplicate entries, keeps the first
// occurrence order and caps the list. The order is not a pin or archive time:
// the sidebar draws pinned rows in the catalog's own order.
func normalizeSessionIDs(in []string, max int) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, min(len(in), max))
	seen := make(map[string]bool, len(in))
	for _, id := range in {
		if !validSessionID(id) || seen[id] {
			continue
		}
		if len(out) >= max {
			break
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// normalizeHidden normalizes both lists and resolves the overlap: an id that is
// archived is removed from the pinned list, so the same session can never be
// both ordered first in the sidebar and hidden from it.
func normalizeHidden(pinned, archived []string) ([]string, []string) {
	pinned = normalizeSessionIDs(pinned, MaxPinnedSessions)
	archived = normalizeSessionIDs(archived, MaxArchivedSessions)
	if len(pinned) == 0 || len(archived) == 0 {
		return pinned, archived
	}
	kept := make([]string, 0, len(pinned))
	for _, id := range pinned {
		if !containsSessionID(archived, id) {
			kept = append(kept, id)
		}
	}
	if len(kept) == 0 {
		return nil, archived
	}
	return kept, archived
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
// whatever an incomplete file omits. An older file has no window, drafts or
// hidden lists; it loads the fields it has and yields the rest as a first run
// does, and the next save rewrites it in the current shape.
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
	state.Pinned, state.Archived = normalizeHidden(f.Pinned, f.Archived)
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
	doc.Pinned, doc.Archived = normalizeHidden(s.Pinned, s.Archived)
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
