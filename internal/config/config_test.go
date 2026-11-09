package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// TestDefault checks the first-run preferences: the zero value, whose fields
// the shell resolves to its own opening values. The store keeps no second copy
// of those values, so a platform whose UI font step differs still opens with
// the shell's own step.
func TestDefault(t *testing.T) {
	if got := Default(); got != (Settings{}) {
		t.Fatalf("Default() = %+v, want the zero value the shell resolves", got)
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
	if got != Default() {
		t.Fatalf("loadFile of a missing file = %+v, want the defaults %+v", got, Default())
	}
}

// TestSaveLoadRoundTrip checks that every preference survives a save and a
// load, and that the file and its directory are private to the user.
func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")
	want := Settings{
		Theme:        2,
		FontSize:     18,
		FontFamily:   1,
		SidebarWidth: 320,
		Density:      1,
		CodeFontSize: 16,
		Locale:       "zh-CN",
	}
	if err := saveFile(path, want, os.Rename); err != nil {
		t.Fatalf("saveFile: %v", err)
	}
	got, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	if got != want {
		t.Fatalf("round trip = %+v, want %+v", got, want)
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

// TestSaveTightensAnExistingFile checks that a file someone widened by hand is
// replaced owner-only: the atomic replace takes the temporary file's mode.
func TestSaveTightensAnExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := saveFile(path, Default(), os.Rename); err != nil {
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
	if got != Default() {
		t.Errorf("loadFile of a corrupt file = %+v, want the defaults %+v", got, Default())
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
	want := Default()
	want.Theme = 1
	got, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	if got != want {
		t.Fatalf("loadFile of a partial file = %+v, want %+v", got, want)
	}
}

// TestSaveReplacesTheFile checks a second save: the target holds the new
// preferences and no temporary file is left beside it.
func TestSaveReplacesTheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	first := Default()
	if err := saveFile(path, first, os.Rename); err != nil {
		t.Fatalf("first saveFile: %v", err)
	}
	second := first
	second.Theme = 2
	second.SidebarWidth = 300
	if err := saveFile(path, second, os.Rename); err != nil {
		t.Fatalf("second saveFile: %v", err)
	}
	got, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	if got != second {
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
// the previous preferences are still readable and the temporary file is gone,
// so a failed save can never lose or half-write the user's settings.
func TestSaveFailureKeepsTheOldFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	old := Default()
	if err := saveFile(path, old, os.Rename); err != nil {
		t.Fatalf("saveFile: %v", err)
	}
	next := old
	next.FontSize = 18
	renameErr := errors.New("rename refused")
	err := saveFile(path, next, func(string, string) error { return renameErr })
	if !errors.Is(err, renameErr) {
		t.Fatalf("saveFile with a failing rename = %v, want %v", err, renameErr)
	}
	got, loadErr := loadFile(path)
	if loadErr != nil {
		t.Fatalf("loadFile after a failed save: %v", loadErr)
	}
	if got != old {
		t.Fatalf("after a failed save = %+v, want the old preferences %+v", got, old)
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

// TestSavedFileHoldsOnlyAppearancePreferences checks the on-disk shape: the
// version plus the six appearance fields, and nothing that belongs to a
// session, run, timeline or approval.
func TestSavedFileHoldsOnlyAppearancePreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := saveFile(path, Default(), os.Rename); err != nil {
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
