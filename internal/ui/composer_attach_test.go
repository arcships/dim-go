package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// composerAttachView builds the composer in isolation with a live draft, so a
// test can drive the attachment entry without the rest of the window.
func composerAttachView(t *testing.T, actions Actions, locale Locale) *ui.Tester {
	t.Helper()
	view := ComposerView{
		Draft:       new(string),
		CanSubmit:   true,
		Placeholder: composerPlaceholder,
		Shortcut:    KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
		Focus:       new(ui.Handle),
		Locale:      locale,
	}
	return ui.NewTester(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, actions)
	}, 800, 600)
}

// TestComposerAttachIsAWiredButton checks that a window with OnAttachFile draws
// a real, keyboard-reachable attachment entry rather than the WIP statement:
// it is not disabled, Enter fires the action once, and its description explains
// the text-file flow instead of a missing picker.
func TestComposerAttachIsAWiredButton(t *testing.T) {
	calls := 0
	tt := composerAttachView(t, Actions{
		OnSubmit:     func(string) {},
		OnAttachFile: func() { calls++ },
	}, LocaleEN)

	label := tr(LocaleEN, keyComposerAttach)
	info, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatalf("the composer offers no %q entry, texts %q", label, tt.Texts())
	}
	if info.Disabled {
		t.Fatalf("%q is unavailable even though OnAttachFile is wired", label)
	}
	if !info.Focusable {
		t.Fatalf("%q is unreachable from the keyboard", label)
	}
	if !strings.Contains(info.Description, tr(LocaleEN, keyReasonAttachmentReady)) {
		t.Errorf("%q description %q does not explain the text-file upload", label, info.Description)
	}

	tt.Key(0, ui.KeyTab)
	if !tt.Focused(label) {
		t.Fatalf("Tab does not reach %q, texts %q", label, tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if calls != 1 {
		t.Fatalf("Enter on %q called OnAttachFile %d times, want once", label, calls)
	}
	if err := tt.Click(label); err != nil {
		t.Fatalf("click %q: %v", label, err)
	}
	if calls != 2 {
		t.Fatalf("clicking %q called OnAttachFile %d times, want a second call", label, calls)
	}
}

// TestComposerAttachWithoutACallbackStaysWIP checks the other state: a window
// with no picker callback keeps the entry focusable but unavailable, explains
// the missing callback, and sends nothing when it is activated.
func TestComposerAttachWithoutACallbackStaysWIP(t *testing.T) {
	tt := composerAttachView(t, Actions{OnSubmit: func(string) {}}, LocaleEN)

	label := tr(LocaleEN, keyComposerAttach)
	info, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatalf("the composer offers no %q entry, texts %q", label, tt.Texts())
	}
	if !info.Disabled {
		t.Fatalf("%q is not marked unavailable without OnAttachFile", label)
	}
	if !info.Focusable {
		t.Fatalf("%q is unreachable without OnAttachFile, so its reason is never read", label)
	}
	if !strings.Contains(info.Description, reasonAttachment) {
		t.Errorf("%q description %q does not name the missing callback", label, info.Description)
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyEnter)
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Error("activating the WIP attachment entry raised an alert")
	}
}

// TestComposerAttachTextIsLocalized checks the attachment entry's name and the
// wired description in the Chinese table, not only the English default.
func TestComposerAttachTextIsLocalized(t *testing.T) {
	tt := composerAttachView(t, Actions{
		OnSubmit:     func(string) {},
		OnAttachFile: func() {},
	}, LocaleZHCN)

	label := tr(LocaleZHCN, keyComposerAttach)
	info, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatalf("the composer offers no %q entry, texts %q", label, tt.Texts())
	}
	if info.Description != tr(LocaleZHCN, keyReasonAttachmentReady) {
		t.Errorf("%q description = %q, want the localized text-file description", label, info.Description)
	}
}

// TestWiredAttachmentDropsTheCapabilityCard checks the sidebar status: the
// Unavailable capabilities list carries an attachment card only while the window
// has no picker callback.
func TestWiredAttachmentDropsTheCapabilityCard(t *testing.T) {
	for _, tc := range []struct {
		name string
		s    *State
		want bool
	}{
		{name: "no callback", s: &State{}, want: true},
		{name: "wired", s: &State{OnAttachFile: func() {}}, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, capability := range wipCapabilities(tc.s) {
				if capability.action != stringsEN.text(keyCapabilityAttachments) {
					continue
				}
				if !tc.want {
					t.Fatalf("the attachment card is still drawn with OnAttachFile wired: %#v", capability)
				}
				return
			}
			if tc.want {
				t.Fatal("the attachment card is missing while the window has no picker callback")
			}
		})
	}
}
