// Package config stores the desktop's own appearance preferences: the color
// theme, the interface and code type sizes, the font family, the sidebar width
// and the density.
//
// These are the choices the appearance surface reports and the shell draws
// with. They are desktop-local: the App Server owns every session, run,
// timeline and approval fact, and nothing of that kind is read or written here.
// This package touches only its own file, never a daemon socket, SQLite,
// JSONL, session file or log.
//
// The file is <user config dir>/dimagent/settings.json, written atomically and
// readable only by the user. A missing file is not an error: a first run opens
// with the shell's own opening values (see Default). A file that cannot be read
// or parsed is returned as an error together with the defaults, so the window
// can report the problem and still open.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// dirName is the directory under the user's config directory that holds
	// this desktop's own preferences.
	dirName = "dimagent"
	// fileName is the preferences file inside dirName.
	fileName = "settings.json"
	// dirPerm and filePerm keep the preferences private to the user.
	dirPerm  os.FileMode = 0o700
	filePerm os.FileMode = 0o600
	// formatVersion identifies the file's shape. It is written on every save;
	// a load ignores fields this version does not know, so a file written by a
	// later version still yields the preferences it does know.
	formatVersion = 1
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

// file is the on-disk shape: the format version plus the preferences
// themselves, which are embedded so the JSON stays one flat object.
type file struct {
	Version int `json:"version"`
	Settings
}

// Default returns the preferences a first run opens with: the zero value.
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

// Path returns the preferences file's path:
// <user config dir>/dimagent/settings.json.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("settings path: %w", err)
	}
	return filepath.Join(dir, dirName, fileName), nil
}

// Load reads the saved preferences. A missing file returns Default and a nil
// error, so a first run is not a failure. A file that cannot be read or parsed
// returns Default and the error; the caller reports it and keeps working with
// the defaults instead of failing to open.
func Load() (Settings, error) {
	path, err := Path()
	if err != nil {
		return Default(), err
	}
	return loadFile(path)
}

// Save writes s to the preferences file. The write goes through a temporary
// file in the same directory and is renamed over the target, so a reader sees
// either the previous file or the new one, and a failure leaves the previous
// file in place.
func Save(s Settings) error {
	path, err := Path()
	if err != nil {
		return err
	}
	return saveFile(path, s, os.Rename)
}

// loadFile reads one preferences file. The defaults are the starting point, so
// a field the file leaves out keeps its default instead of being reset to
// whatever an incomplete file omits.
func loadFile(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("read settings %s: %w", path, err)
	}
	f := file{Version: formatVersion, Settings: Default()}
	if err := json.Unmarshal(data, &f); err != nil {
		return Default(), fmt.Errorf("parse settings %s: %w", path, err)
	}
	return f.Settings, nil
}

// saveFile writes s to path through a temporary file and renames it over the
// target. rename is the last step and a seam for tests: when it fails, path
// still holds the previous file and the temporary file is removed.
func saveFile(path string, s Settings, rename func(oldpath, newpath string) error) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create settings dir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(file{Version: formatVersion, Settings: s}, "", "  ")
	if err != nil {
		// The fields are numbers, so this cannot happen; report it instead of
		// writing a partial file if it ever does.
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
