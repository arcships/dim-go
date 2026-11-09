package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestDefaultState checks the first-run state: the zero appearance preference,
// no window geometry and no drafts. The shell resolves a zero appearance field
// to its own opening value, so the store keeps no second copy of those values.
func TestDefaultState(t *testing.T) {
	got := DefaultState()
	if got.Settings != (Settings{}) {
		t.Fatalf("DefaultState().Settings = %+v, want the zero value the shell resolves", got.Settings)
	}
	if got.Window != (Window{}) || got.Drafts != nil {
		t.Fatalf("DefaultState() = %+v, want no window and no drafts", got)
	}
	if len(got.Drafts) != 0 || got.DraftFor("s1") != "" {
		t.Fatalf("a first run invented a draft: %+v", got)
	}
}

// TestLoadMissingFileReturnsDefaults checks the first run: no file is not an
// error, and the window gets the defaults it draws with.
func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-created", "settings.json")
	got, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile of a missing file: %v", err)
	}
	if got.Settings != Default() || got.Window != (Window{}) || got.Drafts != nil {
		t.Fatalf("loadFile of a missing file = %+v, want the defaults %+v", got, DefaultState())
	}
}

// TestSaveLoadRoundTrip checks that every desktop-local field survives a save
// and a load, and that the file and its directory are private to the user.
func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")
	want := State{
		Settings: Settings{
			Theme:        2,
			FontSize:     18,
			FontFamily:   1,
			SidebarWidth: 320,
			Density:      1,
			CodeFontSize: 16,
			Locale:       "zh-CN",
		},
		Window: Window{X: -1200, Y: 40, Width: 1100, Height: 760, Maximized: true},
		Drafts: Drafts{"s1": "unfinished message", "s2": "second draft"},
	}
	if err := saveFile(path, want, os.Rename); err != nil {
		t.Fatalf("saveFile: %v", err)
	}
	got, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	if got.Settings != want.Settings {
		t.Fatalf("round trip settings = %+v, want %+v", got.Settings, want.Settings)
	}
	if got.Window != want.Window {
		t.Fatalf("round trip window = %+v, want %+v", got.Window, want.Window)
	}
	if !reflect.DeepEqual(got.Drafts, want.Drafts) {
		t.Fatalf("round trip drafts = %+v, want %+v", got.Drafts, want.Drafts)
	}

	if info, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("stat settings dir: %v", err)
	} else if perm := info.Mode().Perm(); perm != dirPerm {
		t.Errorf("settings dir mode = %o, want %o", perm, dirPerm)
	}
	if info, err := os.Stat(path); err != nil {
		t.Fatalf("stat settings file: %v", err)
	} else if perm := info.Mode().Perm(); perm != filePerm {
		t.Errorf("settings file mode = %o, want %o", perm, filePerm)
	}
}

// TestLoadVersion1FileMigrates checks a file written before drafts and window
// geometry existed: its appearance preferences load unchanged, the new fields
// start empty, and the next save rewrites the file in the current shape.
func TestLoadVersion1FileMigrates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	old := `{
  "version": 1,
  "theme": 2,
  "fontSize": 18,
  "fontFamily": 1,
  "sidebarWidth": 320,
  "density": 1,
  "codeFontSize": 16,
  "locale": "zh-CN"
}
`
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile of a version 1 file: %v", err)
	}
	wantSettings := Settings{
		Theme:        2,
		FontSize:     18,
		FontFamily:   1,
		SidebarWidth: 320,
		Density:      1,
		CodeFontSize: 16,
		Locale:       "zh-CN",
	}
	if got.Settings != wantSettings {
		t.Fatalf("version 1 settings = %+v, want %+v", got.Settings, wantSettings)
	}
	if got.Window != (Window{}) || got.Drafts != nil {
		t.Fatalf("version 1 file invented window/drafts: %+v", got)
	}

	// A save from the older shape keeps the preferences and writes version 2.
	got.Drafts = Drafts{"s1": "typed but not sent"}
	if err := saveFile(path, got, os.Rename); err != nil {
		t.Fatalf("saveFile after migrating: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	var version int
	if err := json.Unmarshal(doc["version"], &version); err != nil {
		t.Fatalf("version key: %v", err)
	}
	if version != formatVersion {
		t.Errorf("migrated file version = %d, want %d", version, formatVersion)
	}
	migrated, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile after migrating: %v", err)
	}
	if migrated.Settings != wantSettings || migrated.DraftFor("s1") != "typed but not sent" {
		t.Fatalf("migrated file = %+v, want the preferences and the draft", migrated)
	}
}

// TestLoadUnknownFieldsAndVersionKeepsKnownState checks that a file from a
// later version still yields the fields this version knows instead of failing.
func TestLoadUnknownFieldsAndVersionKeepsKnownState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	content := `{"version": 99, "theme": 1, "futureField": {"nested": true}, "drafts": {"s1": "keep"}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	if got.Settings.Theme != 1 || got.DraftFor("s1") != "keep" {
		t.Fatalf("known fields were not loaded: %+v", got)
	}
}

// TestSaveTightensAnExistingFile checks that a file someone widened by hand is
// replaced owner-only: the atomic replace takes the temporary file's mode.
func TestSaveTightensAnExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := saveFile(path, DefaultState(), os.Rename); err != nil {
		t.Fatalf("saveFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != filePerm {
		t.Errorf("settings file mode = %o, want %o", perm, filePerm)
	}
}

// TestLoadCorruptFileReportsError checks a damaged file: the error is reported,
// the defaults come back with it, and nothing panics.
func TestLoadCorruptFileReportsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{this is not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := loadFile(path)
	if err == nil {
		t.Fatal("loadFile of a corrupt file returned no error")
	}
	if !strings.Contains(err.Error(), "parse settings") {
		t.Errorf("error = %q, want a parse failure naming the file", err)
	}
	if !reflect.DeepEqual(got, DefaultState()) {
		t.Errorf("loadFile of a corrupt file = %+v, want the defaults %+v", got, DefaultState())
	}
}

// TestLoadWrongFieldTypeReportsError checks a file whose values are the wrong
// type: it is corrupt for this shape, not a preference to guess at.
func TestLoadWrongFieldTypeReportsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadFile(path); err == nil {
		t.Fatal("loadFile of a file with a string theme returned no error")
	}
}

// TestLoadPartialFileKeepsDefaults checks that a field the file omits takes the
// default it opens with, not a zero the store invents.
func TestLoadPartialFileKeepsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	want := DefaultState()
	want.Settings.Theme = 1
	got, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loadFile of a partial file = %+v, want %+v", got, want)
	}
}

// TestSaveReplacesTheFile checks a second save: the target holds the new state
// and no temporary file is left beside it.
func TestSaveReplacesTheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	first := DefaultState()
	if err := saveFile(path, first, os.Rename); err != nil {
		t.Fatalf("first saveFile: %v", err)
	}
	second := first.Clone()
	second.Settings.Theme = 2
	second.Settings.SidebarWidth = 300
	second.Window = Window{X: 10, Y: 20, Width: 900, Height: 600}
	second.Drafts = Drafts{"s1": "next"}
	if err := saveFile(path, second, os.Rename); err != nil {
		t.Fatalf("second saveFile: %v", err)
	}
	got, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	if !reflect.DeepEqual(got, second) {
		t.Fatalf("after replace = %+v, want %+v", got, second)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".settings-") {
			t.Errorf("temporary file %q was left behind", entry.Name())
		}
	}
}

// TestSaveFailureKeepsTheOldFile checks the atomic part: when the replace fails,
// the previous state is still readable and the temporary file is gone, so a
// failed save can never lose or half-write the user's settings.
func TestSaveFailureKeepsTheOldFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	old := DefaultState()
	old.Drafts = Drafts{"s1": "keep this"}
	if err := saveFile(path, old, os.Rename); err != nil {
		t.Fatalf("saveFile: %v", err)
	}
	next := old.Clone()
	next.Settings.FontSize = 18
	next.Drafts["s1"] = "must not land"
	renameErr := errors.New("rename refused")
	err := saveFile(path, next, func(string, string) error { return renameErr })
	if !errors.Is(err, renameErr) {
		t.Fatalf("saveFile with a failing rename = %v, want %v", err, renameErr)
	}
	got, loadErr := loadFile(path)
	if loadErr != nil {
		t.Fatalf("loadFile after a failed save: %v", loadErr)
	}
	if !reflect.DeepEqual(got, old) {
		t.Fatalf("after a failed save = %+v, want the old state %+v", got, old)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".settings-") {
			t.Errorf("temporary file %q was left behind", entry.Name())
		}
	}
}

// TestSavedFileHoldsOnlyDesktopState checks the on-disk shape: the version plus
// the appearance fields when nothing else was saved, and nothing that belongs
// to a session, run, timeline or approval.
func TestSavedFileHoldsOnlyDesktopState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := saveFile(path, DefaultState(), os.Rename); err != nil {
		t.Fatalf("saveFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("the saved file is not a JSON object: %v", err)
	}
	got := make([]string, 0, len(doc))
	for key := range doc {
		got = append(got, key)
	}
	sort.Strings(got)
	want := []string{"codeFontSize", "density", "fontFamily", "fontSize", "sidebarWidth", "theme", "version"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("saved keys = %v, want %v", got, want)
	}
	var version int
	if err := json.Unmarshal(doc["version"], &version); err != nil {
		t.Fatalf("version key: %v", err)
	}
	if version != formatVersion {
		t.Errorf("saved version = %d, want %d", version, formatVersion)
	}
}

// TestSavedFileHoldsWindowAndDrafts checks that the extended shape uses its own
// keys, so the new state stays separable from appearance preferences.
func TestSavedFileHoldsWindowAndDrafts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	state := DefaultState()
	state.Window = Window{X: 4, Y: 5, Width: 640, Height: 480}
	state.Drafts = Drafts{"s1": "hello"}
	if err := saveFile(path, state, os.Rename); err != nil {
		t.Fatalf("saveFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"window", "drafts"} {
		if _, ok := doc[key]; !ok {
			t.Errorf("saved file has no %q key: %s", key, data)
		}
	}
	if _, ok := doc["sessions"]; ok {
		t.Errorf("saved file holds a session field: %s", data)
	}
}

// TestDraftBounds checks the draft limits: whitespace is not a draft, one draft
// is cut at a rune boundary, the map holds at most MaxDrafts entries, and the
// total stays within the file's bound.
func TestDraftBounds(t *testing.T) {
	state := DefaultState()
	state.SetDraft("s1", "   \n\t")
	if state.Drafts != nil {
		t.Fatalf("whitespace-only text became a draft: %+v", state.Drafts)
	}

	state.SetDraft("s1", "keep me")
	if got := state.DraftFor("s1"); got != "keep me" {
		t.Fatalf("DraftFor = %q, want keep me", got)
	}
	state.ClearDraft("s1")
	if got := state.DraftFor("s1"); got != "" || state.Drafts != nil {
		t.Fatalf("ClearDraft left %+v", state.Drafts)
	}

	// A multi-byte rune cut in the middle must not leave invalid UTF-8.
	oversized := strings.Repeat("界", MaxDraftBytes)
	state.SetDraft("s2", oversized)
	got := state.DraftFor("s2")
	if len(got) > MaxDraftBytes {
		t.Fatalf("draft length = %d, want at most %d", len(got), MaxDraftBytes)
	}
	if !utf8.ValidString(got) {
		t.Fatal("the truncated draft is not valid UTF-8")
	}

	many := Drafts{}
	for i := 0; i < MaxDrafts+20; i++ {
		many[draftID(i)] = "text"
	}
	state.Drafts = many
	state = normalizeState(state)
	if len(state.Drafts) != MaxDrafts {
		t.Fatalf("draft count = %d, want at most %d", len(state.Drafts), MaxDrafts)
	}

	huge := Drafts{}
	for i := 0; i < MaxDrafts; i++ {
		huge[draftID(i)] = strings.Repeat("x", MaxDraftBytes)
	}
	state.Drafts = huge
	state = normalizeState(state)
	total := 0
	for _, text := range state.Drafts {
		total += len(text)
	}
	if total > maxDraftTotal {
		t.Fatalf("draft total = %d, want at most %d", total, maxDraftTotal)
	}

	state.Drafts = Drafts{strings.Repeat("x", maxDraftSessionID+1): "text", "s1": "kept"}
	state = normalizeState(state)
	if _, ok := state.Drafts[strings.Repeat("x", maxDraftSessionID+1)]; ok {
		t.Fatal("an oversized session id stayed a draft key")
	}
	if state.DraftFor("s1") != "kept" {
		t.Fatalf("a long session id dropped a normal draft: %+v", state.Drafts)
	}
}

// TestCloneIsDeep checks that changing a clone's drafts leaves the original
// store's map alone.
func TestCloneIsDeep(t *testing.T) {
	state := DefaultState()
	state.Drafts = Drafts{"s1": "one"}
	clone := state.Clone()
	clone.Drafts["s1"] = "changed"
	clone.Drafts["s2"] = "added"
	if state.Drafts["s1"] != "one" || state.Drafts["s2"] != "" {
		t.Fatalf("clone shared the map: %+v", state.Drafts)
	}
}

// TestPathNamesTheDesktopFile checks the file's location and name under the
// user's config directory, without creating anything.
func TestPathNamesTheDesktopFile(t *testing.T) {
	path, err := Path()
	if err != nil {
		t.Skipf("no user config dir on this machine: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("settings path %q is not absolute", path)
	}
	if base := filepath.Base(path); base != fileName {
		t.Errorf("settings file = %q, want %q", base, fileName)
	}
	if dir := filepath.Base(filepath.Dir(path)); dir != dirName {
		t.Errorf("settings directory = %q, want %q", dir, dirName)
	}
}

// normalizeState applies the same draft normalization saveFile applies, without
// a disk round trip.
func normalizeState(state State) State {
	state.Drafts = normalizeDrafts(state.Drafts)
	return state
}

func draftID(i int) string {
	return "session-" + string(rune('a'+i%26)) + "-" + strconv.Itoa(i)
}
