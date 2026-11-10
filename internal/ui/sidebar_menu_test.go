package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// sidebarMenuTester draws the whole shell with the window's menus drawn in it,
// so a context menu's items expose their disabled state and their reason to
// assistive technology, as the session menu's items do.
func sidebarMenuTester(s *State, width, height int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		c.SetMenuPresentation(ui.PopoverMenus)
		s.View(c)
	}, width, height)
}

// rightClickRow opens a sidebar row's context menu. The row, not its text, is
// the input target, so the click reaches the row's own menu at the text's
// center; a click on the text itself would be rejected.
func rightClickRow(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	rect, ok := tt.Find(label)
	if !ok {
		t.Fatalf("row %q is not drawn, texts %q", label, tt.Texts())
	}
	tt.RightClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
}

// TestSidebarRowContextMenuStructure checks the row menu against the
// reference's session menu: Pin (or Unpin), Rename, Archive (or Unarchive) and,
// after the rule before the destructive entry, Delete session, in that order.
// A window with no callbacks keeps every entry visible, disabled and explained,
// so a missing action reads as WIP rather than as a successful one.
func TestSidebarRowContextMenuStructure(t *testing.T) {
	s := fixture()
	tt := sidebarMenuTester(s, 900, 600)
	rightClickRow(t, tt, "First session")

	want := []string{
		tr(LocaleEN, keySidebarPin),
		tr(LocaleEN, keySidebarRename),
		tr(LocaleEN, keySidebarArchive),
		"-",
		tr(LocaleEN, keySidebarDelete),
	}
	if got := tt.Menu(); !slices.Equal(got, want) {
		t.Fatalf("the row menu draws %q, want %q", got, want)
	}

	for _, tc := range []struct{ key, reason string }{
		{keySidebarPin, tr(LocaleEN, keyReasonPin)},
		{keySidebarRename, tr(LocaleEN, keyReasonRename)},
		{keySidebarArchive, tr(LocaleEN, keyReasonArchive)},
		{keySidebarDelete, tr(LocaleEN, keyReasonDelete)},
	} {
		label := tr(LocaleEN, tc.key)
		info, ok := tt.AccessByRole(ui.RoleMenuItem, label)
		if !ok {
			t.Fatalf("the row menu has no %q item, texts %q", label, tt.Texts())
		}
		if !info.Disabled {
			t.Errorf("%q is available without its callback", label)
		}
		if !strings.HasPrefix(info.Description, wipBadge) || !strings.Contains(info.Description, tc.reason) {
			t.Errorf("%q explains %q, want the badge and %q", label, info.Description, tc.reason)
		}
	}
	if !tt.HasText(wipBadge) {
		t.Errorf("the WIP entries carry no visible %q badge, texts %q", wipBadge, tt.Texts())
	}
	for _, label := range want {
		if label == "-" {
			continue
		}
		if err := tt.ChooseMenuItem(label); err == nil {
			t.Fatalf("chose the disabled %q item", label)
		}
	}
}

// TestSidebarRowContextMenuRunsTheRow checks that every entry addresses the
// row it was opened on, exactly once. Pin and archive carry that row's id to
// the desktop-local lists; rename and delete open the window's dialogs, which
// address the shell's selection, so the menu selects the row first even though
// the window had another session selected. The destructive entry only opens
// the confirmation: it never deletes from the menu itself.
func TestSidebarRowContextMenuRunsTheRow(t *testing.T) {
	s := fixture()
	var (
		pins, archives, selects []string
		renames, deletes        int
	)
	s.OnPinSession = func(sessionID string) { pins = append(pins, sessionID) }
	s.OnArchiveSession = func(sessionID string) { archives = append(archives, sessionID) }
	s.OnSelect = func(sessionID string) { selects = append(selects, sessionID) }
	s.OnRename = func(string) { renames++ }
	s.OnDeleteSession = func(string) { deletes++ }

	tt := sidebarMenuTester(s, 900, 600)

	// The selection starts on s1; the menu opens on s2 and renames that row.
	rightClickRow(t, tt, "Second session")
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarRename))
	if s.Selected != "s2" {
		t.Fatalf("the rename menu left the selection on %q, want s2", s.Selected)
	}
	if !slices.Equal(selects, []string{"s2"}) {
		t.Errorf("the menu opened %q, want s2 once", selects)
	}
	if renames != 0 {
		t.Errorf("Rename sent session.rename %d times before the dialog was confirmed", renames)
	}
	if !s.RenameConfirm || s.renameFor != "s2" {
		t.Errorf("Rename opened %+v for %q, want s2", s.RenameConfirm, s.renameFor)
	}
	if err := tt.Click(tr(LocaleEN, keyHeaderRenameConfirm)); err != nil {
		t.Fatalf("confirming the rename dialog: %v", err)
	}
	if renames != 1 {
		t.Errorf("Rename ran %d times after one confirm, want 1", renames)
	}
	tt.Frame()

	// The pin and archive entries act on the row they were opened on, not on
	// the selection the rename left behind.
	rightClickRow(t, tt, "First session")
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarPin))
	if !slices.Equal(pins, []string{"s1"}) {
		t.Errorf("Pin ran for %q, want s1 once", pins)
	}
	if s.Selected != "s2" {
		t.Errorf("Pin moved the selection to %q, want the rename's s2", s.Selected)
	}
	rightClickRow(t, tt, "First session")
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarArchive))
	if !slices.Equal(archives, []string{"s1"}) {
		t.Errorf("Archive ran for %q, want s1 once", archives)
	}

	// Delete opens the confirmation for the row; the deletion waits for the
	// alert's confirm button and is not sent from the menu.
	rightClickRow(t, tt, "Second session")
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarDelete))
	if deletes != 0 {
		t.Errorf("the menu deleted the session %d times", deletes)
	}
	if !s.DeleteConfirm || s.deleteFor != "s2" {
		t.Errorf("Delete opened %+v for %q, want s2", s.DeleteConfirm, s.deleteFor)
	}
	if err := tt.Click(tr(LocaleEN, keyHeaderDeleteConfirm)); err != nil {
		t.Fatalf("confirming the delete alert: %v", err)
	}
	if deletes != 1 {
		t.Errorf("Delete ran %d times after one confirm, want 1", deletes)
	}
}

// TestSidebarRowContextMenuLabelsFollowTheRow checks the two toggling labels:
// a pinned row offers Unpin, and choosing it asks the caller to unpin that row.
func TestSidebarRowContextMenuLabelsFollowTheRow(t *testing.T) {
	s := fixture()
	s.Pinned = []string{"s1"}
	var pins []string
	s.OnPinSession = func(sessionID string) { pins = append(pins, sessionID) }

	tt := sidebarMenuTester(s, 900, 600)
	rightClickRow(t, tt, "First session")
	if got := tt.Menu(); len(got) == 0 || got[0] != tr(LocaleEN, keySidebarUnpin) {
		t.Fatalf("a pinned row's menu draws %q, want Unpin first", got)
	}
	clickMenuItem(t, tt, tr(LocaleEN, keySidebarUnpin))
	if !slices.Equal(pins, []string{"s1"}) {
		t.Errorf("Unpin ran for %q, want s1 once", pins)
	}
}

// TestSidebarRowContextMenuClosesWithoutAnAction checks the menu's two ways
// out: Escape closes it and gives the focus back to the row, and a press
// outside closes it. Neither runs an entry.
func TestSidebarRowContextMenuClosesWithoutAnAction(t *testing.T) {
	s := fixture()
	pins := 0
	s.OnPinSession = func(string) { pins++ }
	tt := sidebarMenuTester(s, 900, 600)

	rightClickRow(t, tt, "First session")
	if len(tt.Menu()) == 0 {
		t.Fatal("the row menu did not open")
	}
	tt.Key(0, ui.KeyEscape)
	if got := tt.Menu(); len(got) != 0 {
		t.Fatalf("Escape left the row menu open: %q", got)
	}

	rightClickRow(t, tt, "First session")
	tt.ClickAt(20, 560)
	if got := tt.Menu(); len(got) != 0 {
		t.Errorf("an outside click left the row menu open: %q", got)
	}
	if pins != 0 {
		t.Errorf("a menu closed without a choice ran Pin %d times", pins)
	}
}

// TestSidebarRowContextMenuIsKeyboardReachable checks the menu's keys: with the
// menu open, Down moves the highlight onto an item and Enter chooses it once.
func TestSidebarRowContextMenuIsKeyboardReachable(t *testing.T) {
	s := fixture()
	var (
		pins, renames, archives, deletes int
	)
	s.OnPinSession = func(string) { pins++ }
	s.OnRename = func(string) { renames++ }
	s.OnArchiveSession = func(string) { archives++ }
	s.OnDeleteSession = func(string) { deletes++ }
	tt := sidebarMenuTester(s, 900, 600)

	rightClickRow(t, tt, "First session")
	labels := []string{
		tr(LocaleEN, keySidebarPin),
		tr(LocaleEN, keySidebarRename),
		tr(LocaleEN, keySidebarArchive),
		tr(LocaleEN, keySidebarDelete),
	}
	reached := ""
	for range len(labels) {
		tt.Key(0, ui.KeyDown)
		for _, label := range labels {
			if info, ok := tt.AccessByRole(ui.RoleMenuItem, label); ok && info.Focused {
				reached = label
			}
		}
		if reached != "" {
			break
		}
	}
	if reached == "" {
		t.Fatalf("Down reached no menu item, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	ran := pins + renames + archives + deletes
	if ran != 1 {
		t.Errorf("Enter on %q ran %d entries, want 1", reached, ran)
	}
	if got := tt.Menu(); len(got) != 0 {
		t.Errorf("choosing %q left the row menu open: %q", reached, got)
	}
}

// TestSidebarGroupRowsHaveNoContextMenu checks that only session rows carry the
// actions: a project heading names a group, not a session, so it offers no menu
// and no entry can address a session the heading does not hold.
func TestSidebarGroupRowsHaveNoContextMenu(t *testing.T) {
	s := fixture()
	s.Sessions = []Session{
		{ID: "s1", Title: "First session", Group: "alpha"},
		{ID: "s2", Title: "Second session", Group: "alpha"},
	}
	s.Selected = "s1"
	tt := sidebarMenuTester(s, 900, 600)
	rightClickRow(t, tt, "alpha")
	if got := tt.Menu(); len(got) != 0 {
		t.Errorf("a project heading opened the row menu %q", got)
	}
}

// TestSidebarRowContextMenuItemReasonsStayLocalized checks that a translated
// window explains the gaps in its language: the item labels and the reasons
// come from the same locale table as the rest of the shell.
func TestSidebarRowContextMenuItemReasonsStayLocalized(t *testing.T) {
	s := fixture()
	s.Settings = SettingsView{Locale: LocaleZHCN}
	tt := sidebarMenuTester(s, 900, 600)
	rightClickRow(t, tt, "First session")

	want := []string{
		tr(LocaleZHCN, keySidebarPin),
		tr(LocaleZHCN, keySidebarRename),
		tr(LocaleZHCN, keySidebarArchive),
		"-",
		tr(LocaleZHCN, keySidebarDelete),
	}
	if got := tt.Menu(); !slices.Equal(got, want) {
		t.Fatalf("the zh-CN row menu draws %q, want %q", got, want)
	}
	info, ok := tt.AccessByRole(ui.RoleMenuItem, tr(LocaleZHCN, keySidebarRename))
	if !ok {
		t.Fatalf("the zh-CN row menu has no Rename item, texts %q", tt.Texts())
	}
	if info.Description != wipBadge+"\n"+tr(LocaleZHCN, keyReasonRename) {
		t.Errorf("the zh-CN Rename reason is %q, want the badge and %q", info.Description, tr(LocaleZHCN, keyReasonRename))
	}
	if !strings.Contains(info.Description, "不可用：") {
		t.Errorf("the zh-CN reason is not translated: %q", info.Description)
	}
}
