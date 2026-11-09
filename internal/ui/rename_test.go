package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// openRenameDialog opens the session menu's Rename entry on the fixture.
func openRenameDialog(t *testing.T, tt *ui.Tester) {
	t.Helper()
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatalf("open the session menu: %v", err)
	}
	clickMenuItem(t, tt, menuText(LocaleEN, keySidebarRename))
	tt.Frame()
}

// TestRenameDialogPrefillsAndSendsTheNewTitle checks the wired rename path: the
// menu entry opens a dialog prefilled with the selected title, editing the field
// and confirming calls OnRename once with the text, and the dialog closes
// without changing the title itself.
func TestRenameDialogPrefillsAndSendsTheNewTitle(t *testing.T) {
	s := fixture()
	var got []string
	s.OnRename = func(title string) { got = append(got, title) }
	tt := ui.NewTester(s.View, 900, 700)

	openRenameDialog(t, tt)
	if !s.RenameConfirm {
		t.Fatalf("the menu entry did not open the rename dialog: %+v", s.RenameConfirm)
	}
	field, ok := tt.AccessByRole(ui.RoleTextField, tr(LocaleEN, keyHeaderRenameLabel))
	if !ok {
		t.Fatalf("the rename dialog has no title field, texts %q", tt.Texts())
	}
	if field.Value != "First session" {
		t.Errorf("the field is prefilled with %q, want the selected title", field.Value)
	}
	tt.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Renamed session")
	if err := tt.Click(tr(LocaleEN, keyHeaderRenameConfirm)); err != nil {
		t.Fatalf("confirm the rename: %v", err)
	}
	if len(got) != 1 || got[0] != "Renamed session" {
		t.Fatalf("OnRename got %q, want one request with the typed title", got)
	}
	if s.RenameConfirm {
		t.Error("the dialog stayed open after the confirm")
	}
	if s.Sessions[0].Title != "First session" {
		t.Errorf("the shell changed the title locally to %q", s.Sessions[0].Title)
	}
}

// TestRenameDialogSendsAnEmptyTitleForAClear checks the dialog's clear path: a
// confirm on an empty field calls OnRename with an empty title once, and the
// caller clears the manual title. An empty title is not a cancel.
func TestRenameDialogSendsAnEmptyTitleForAClear(t *testing.T) {
	s := fixture()
	var got []string
	s.OnRename = func(title string) { got = append(got, title) }
	tt := ui.NewTester(s.View, 900, 700)

	openRenameDialog(t, tt)
	field, ok := tt.AccessByRole(ui.RoleTextField, tr(LocaleEN, keyHeaderRenameLabel))
	if !ok {
		t.Fatalf("the rename dialog has no title field, texts %q", tt.Texts())
	}
	tt.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if err := tt.Click(tr(LocaleEN, keyHeaderRenameConfirm)); err != nil {
		t.Fatalf("confirm the clear: %v", err)
	}
	if len(got) != 1 || got[0] != "" {
		t.Fatalf("OnRename got %q, want one empty title", got)
	}
}

// TestRenameDialogCancelAndEscapeSendNothing checks both ways out: Cancel and
// Escape close the dialog without calling OnRename, so a dismissed dialog is
// never a rename.
func TestRenameDialogCancelAndEscapeSendNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		quit func(t *testing.T, tt *ui.Tester)
	}{
		{"cancel", func(t *testing.T, tt *ui.Tester) {
			t.Helper()
			if err := tt.Click(tr(LocaleEN, keyHeaderRenameCancel)); err != nil {
				t.Fatalf("cancel the rename: %v", err)
			}
		}},
		{"escape", func(t *testing.T, tt *ui.Tester) {
			t.Helper()
			tt.Key(0, ui.KeyEscape)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fixture()
			calls := 0
			s.OnRename = func(string) { calls++ }
			tt := ui.NewTester(s.View, 900, 700)

			openRenameDialog(t, tt)
			tc.quit(t, tt)
			tt.Frame()
			if calls != 0 {
				t.Errorf("%s sent %d rename requests, want none", tc.name, calls)
			}
			if s.RenameConfirm {
				t.Errorf("%s left the dialog open", tc.name)
			}
		})
	}
}

// TestRenameDialogTabReachesItsEntries checks the dialog's keyboard order: after
// the field, Tab reaches Cancel and then Rename, so a keyboard user can confirm
// or dismiss the dialog without the pointer.
func TestRenameDialogTabReachesItsEntries(t *testing.T) {
	s := fixture()
	s.OnRename = func(string) {}
	tt := ui.NewTester(s.View, 900, 700)

	openRenameDialog(t, tt)
	field, ok := tt.AccessByRole(ui.RoleTextField, tr(LocaleEN, keyHeaderRenameLabel))
	if !ok {
		t.Fatalf("the rename dialog has no title field, texts %q", tt.Texts())
	}
	tt.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	if !tt.Focused(tr(LocaleEN, keyHeaderRenameLabel)) {
		t.Fatalf("the title field did not take the focus, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused(tr(LocaleEN, keyHeaderRenameCancel)) {
		t.Fatalf("Tab does not reach Cancel, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused(tr(LocaleEN, keyHeaderRenameConfirm)) {
		t.Fatalf("Tab does not reach Rename, texts %q", tt.Texts())
	}
}

// TestRenameMenuStaysWIPWithoutACallback checks the contract's other side: a
// fixture with no OnRename keeps the Rename entry visible, disabled and
// explained with session.rename, instead of opening a dialog that cannot send.
func TestRenameMenuStaysWIPWithoutACallback(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 700)
	if err := tt.Click(tr(LocaleEN, keyHeaderMenu)); err != nil {
		t.Fatal(err)
	}
	info, ok := tt.AccessByRole(ui.RoleMenuItem, menuText(LocaleEN, keySidebarRename))
	if !ok {
		t.Fatalf("the menu has no Rename item, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("Rename is available without an OnRename callback")
	}
	if !strings.Contains(info.Description, tr(LocaleEN, keyReasonRename)) {
		t.Errorf("Rename description %q does not name the gap", info.Description)
	}
	if s.RenameConfirm {
		t.Error("a WIP Rename entry opened the dialog")
	}
}
