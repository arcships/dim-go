package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/config"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// appearanceApp builds a window with the preference hooks the test supplies,
// without a daemon or a real window.
func appearanceApp(state *dimui.State, load func() (config.Settings, error), save func(config.Settings) error) *app {
	return &app{state: state, appearance: appearance{load: load, save: save}}
}

// TestLoadAppearanceAppliesSavedPreferences checks startup: the saved
// preferences become the appearance the shell draws, the sidebar opens at the
// saved width, and no session fact is invented or overwritten.
func TestLoadAppearanceAppliesSavedPreferences(t *testing.T) {
	saved := config.Settings{
		Theme:        2,
		FontSize:     18,
		FontFamily:   1,
		SidebarWidth: 320,
		Density:      1,
		CodeFontSize: 16,
	}
	state := &dimui.State{
		Sessions:   []dimui.Session{{ID: "s1", Title: "Existing"}},
		Selected:   "s1",
		Connection: dimui.ConnReady,
	}
	a := appearanceApp(state, func() (config.Settings, error) { return saved, nil }, nil)
	a.loadAppearance()

	want := dimui.SettingsView{
		Theme:        dimui.ThemeDark,
		FontSize:     18,
		FontFamily:   dimui.FontMonospace,
		SidebarWidth: 320,
		Density:      dimui.DensityCompact,
		CodeFontSize: 16,
	}
	if !reflect.DeepEqual(state.Settings, want) {
		t.Fatalf("settings = %+v, want %+v", state.Settings, want)
	}
	if state.SidebarWidth != 320 {
		t.Fatalf("sidebar width = %.0f, want the saved 320", state.SidebarWidth)
	}
	if a.appearance.last != saved {
		t.Fatalf("last snapshot = %+v, want the loaded %+v", a.appearance.last, saved)
	}
	if len(state.Sessions) != 1 || state.Sessions[0].ID != "s1" || state.Selected != "s1" || state.Connection != dimui.ConnReady {
		t.Fatalf("loading preferences changed session facts: %#v", state)
	}
}

// TestLoadAppearanceDoesNotWriteBackAHandEditedValue checks that a file value
// the shell folds to one of its own choices is the baseline, not a choice: the
// first frame after such a load writes nothing.
func TestLoadAppearanceDoesNotWriteBackAHandEditedValue(t *testing.T) {
	state := &dimui.State{}
	var saved []config.Settings
	a := appearanceApp(state, func() (config.Settings, error) {
		// 300 is no theme choice; the shell folds it to its default when drawing.
		return config.Settings{Theme: 300, SidebarWidth: 999}, nil
	}, func(s config.Settings) error {
		saved = append(saved, s)
		return nil
	})
	a.loadAppearance()
	a.saveAppearance()

	if len(saved) != 0 {
		t.Fatalf("loading an out-of-range file wrote %d snapshot(s): %+v", len(saved), saved)
	}
}

// TestLoadAppearanceKeepsDefaultsOnError checks a damaged or unreadable file:
// the error is reported through the log, the window still opens with the
// defaults the store returned, and nothing else in State changes.
func TestLoadAppearanceKeepsDefaultsOnError(t *testing.T) {
	state := &dimui.State{Connection: dimui.ConnReady}
	a := appearanceApp(state, func() (config.Settings, error) {
		return config.Default(), errors.New("parse settings: unexpected end of JSON input")
	}, nil)
	a.loadAppearance()

	if !reflect.DeepEqual(state.Settings, settingsView(config.Default())) {
		t.Fatalf("settings = %+v, want the defaults %+v", state.Settings, settingsView(config.Default()))
	}
	if state.SidebarWidth != config.Default().SidebarWidth {
		t.Fatalf("sidebar width = %.0f, want the default %.0f", state.SidebarWidth, config.Default().SidebarWidth)
	}
	if state.Connection != dimui.ConnReady {
		t.Fatalf("loading preferences changed the connection: %#v", state)
	}
}

// TestSaveAppearanceWritesOneSnapshotPerChoice checks that a frame which changed
// no preference writes nothing, one choice writes exactly one snapshot, and a
// redraw of the same choice writes nothing more.
func TestSaveAppearanceWritesOneSnapshotPerChoice(t *testing.T) {
	state := &dimui.State{}
	var saved []config.Settings
	a := appearanceApp(state, nil, func(s config.Settings) error {
		saved = append(saved, s)
		return nil
	})
	a.appearance.last = config.Default()

	a.saveAppearance()
	if len(saved) != 0 {
		t.Fatalf("an unchanged frame wrote %d snapshot(s): %+v", len(saved), saved)
	}

	// What State.applySettings stores for a choice the surface reported.
	state.Settings = dimui.SettingsView{
		Theme:        dimui.ThemeDark,
		FontSize:     16,
		FontFamily:   dimui.FontMonospace,
		SidebarWidth: 288,
		Density:      dimui.DensityCompact,
		CodeFontSize: 15,
	}
	state.SidebarWidth = 288
	a.saveAppearance()

	want := config.Settings{Theme: 2, FontSize: 16, FontFamily: 1, SidebarWidth: 288, Density: 1, CodeFontSize: 15}
	if len(saved) != 1 {
		t.Fatalf("one choice wrote %d snapshot(s): %+v", len(saved), saved)
	}
	if saved[0] != want {
		t.Fatalf("saved %+v, want %+v", saved[0], want)
	}
	if state.ActionError != "" {
		t.Fatalf("a successful save reported %q", state.ActionError)
	}

	a.saveAppearance()
	if len(saved) != 1 {
		t.Fatalf("a redraw wrote the unchanged snapshot again: %+v", saved)
	}
}

// TestSaveAppearanceIgnoresABareSidebarDrag checks that a drag which reports no
// settings choice is not written: the store holds preferences, and the
// appearance page's own width buttons are the choice that saves a width. It
// also keeps the first frame quiet: the shell resolves a zero width to its
// default while drawing, and that resolution is not a choice either.
func TestSaveAppearanceIgnoresABareSidebarDrag(t *testing.T) {
	state := &dimui.State{SidebarWidth: 300}
	var saved []config.Settings
	a := appearanceApp(state, nil, func(s config.Settings) error {
		saved = append(saved, s)
		return nil
	})
	a.appearance.last = config.Default()

	a.saveAppearance()
	if len(saved) != 0 {
		t.Fatalf("a bare sidebar drag wrote %d snapshot(s): %+v", len(saved), saved)
	}
}

// TestSaveAppearanceFailureKeepsChoiceAndReportsError checks a failed write:
// the choice stays applied, the failure is left in ActionError for the header
// banner, and an unchanged frame does not retry it. The next choice tries
// again, and a success clears the error this window wrote.
func TestSaveAppearanceFailureKeepsChoiceAndReportsError(t *testing.T) {
	state := &dimui.State{Settings: settingsView(config.Default())}
	boom := errors.New("no space left on device")
	var calls int
	a := appearanceApp(state, nil, func(config.Settings) error {
		calls++
		return boom
	})
	a.appearance.last = config.Default()

	state.Settings.Theme = dimui.ThemeLight
	if ok := a.saveAppearance(); !ok {
		t.Fatal("a failed save did not report a header change")
	}

	if calls != 1 {
		t.Fatalf("the first choice attempted %d save(s), want 1", calls)
	}
	if !strings.Contains(state.ActionError, boom.Error()) {
		t.Fatalf("ActionError = %q, want the save failure", state.ActionError)
	}
	if state.Settings.Theme != dimui.ThemeLight {
		t.Fatalf("a failed save reverted the choice to %v", state.Settings.Theme)
	}

	a.saveAppearance()
	if calls != 1 {
		t.Fatalf("an unchanged frame retried the failed save: %d attempts", calls)
	}

	a.appearance.save = func(config.Settings) error {
		calls++
		return nil
	}
	state.Settings.Theme = dimui.ThemeDark
	if ok := a.saveAppearance(); !ok {
		t.Fatal("a successful save did not report clearing the header error")
	}

	if calls != 2 {
		t.Fatalf("the next choice attempted %d save(s), want 2", calls)
	}
	if state.ActionError != "" {
		t.Fatalf("a successful save left its own error: %q", state.ActionError)
	}
}

// TestSaveAppearanceLeavesAnotherActionsError checks that a settings save,
// successful or not, does not clear a failure another action reported first.
func TestSaveAppearanceLeavesAnotherActionsError(t *testing.T) {
	state := &dimui.State{Settings: settingsView(config.Default())}
	const other = `Open session "s2": boom`
	state.ActionError = other
	a := appearanceApp(state, nil, func(config.Settings) error { return nil })
	a.appearance.last = config.Default()

	state.Settings.Theme = dimui.ThemeDark
	a.saveAppearance()

	if state.ActionError != other {
		t.Fatalf("ActionError = %q, want the other action's error %q", state.ActionError, other)
	}
}

// TestContentSavesTheChoiceAppliedInTheFrame checks the window's content
// callback: it draws through State.View and then writes an appearance choice the
// surface applied during that frame, while a frame with no choice writes
// nothing.
func TestContentSavesTheChoiceAppliedInTheFrame(t *testing.T) {
	state := &dimui.State{Settings: settingsView(config.Default())}
	var saved []config.Settings
	a := appearanceApp(state, nil, func(s config.Settings) error {
		saved = append(saved, s)
		return nil
	})
	a.appearance.last = config.Default()

	tt := ui.NewTester(func(c *ui.Context) { a.content(c) }, 900, 600)
	if len(saved) != 0 {
		t.Fatalf("the first frame wrote %d snapshot(s): %+v", len(saved), saved)
	}

	// What the appearance surface reports when the user picks Dark.
	state.Settings.Theme = dimui.ThemeDark
	tt.Frame()

	if len(saved) != 1 {
		t.Fatalf("the frame with a choice wrote %d snapshot(s): %+v", len(saved), saved)
	}
	if saved[0].Theme != int(dimui.ThemeDark) {
		t.Fatalf("saved theme = %d, want Dark (%d)", saved[0].Theme, dimui.ThemeDark)
	}
}

// TestContentShowsASaveFailureInTheHeader checks that a failed write reaches the
// user: the header banner draws the failure in the frame content asks for after
// the save, while the chosen appearance stays applied.
func TestContentShowsASaveFailureInTheHeader(t *testing.T) {
	state := &dimui.State{Settings: settingsView(config.Default())}
	boom := errors.New("no space left on device")
	a := appearanceApp(state, nil, func(config.Settings) error { return boom })
	a.appearance.last = config.Default()

	tt := ui.NewTester(func(c *ui.Context) { a.content(c) }, 900, 600)

	state.Settings.Theme = dimui.ThemeDark
	tt.Frame()

	want := "Save appearance preferences: " + boom.Error()
	if !tt.HasText(want) {
		t.Fatalf("the header does not show %q, texts %q", want, tt.Texts())
	}
	if state.Settings.Theme != dimui.ThemeDark {
		t.Fatalf("a failed save reverted the choice to %v", state.Settings.Theme)
	}
}
