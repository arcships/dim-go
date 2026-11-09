package ui

import (
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// composerContextView builds a composer in isolation with the context row's
// display copies, so a test can check the row with and without the facts the
// shell has not received yet. The harness owns the draft pointer, the focus
// handle and the submit callback, as the shell does.
func composerContextView(t *testing.T, workspace, permission string) (ComposerView, *ui.Tester) {
	t.Helper()
	view := ComposerView{
		Draft:       new(string),
		CanSubmit:   true,
		Placeholder: composerPlaceholder,
		Workspace:   workspace,
		Permission:  permission,
		Shortcut:    KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
		Focus:       new(ui.Handle),
	}
	tt := ui.NewTester(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, Actions{OnSubmit: func(string) {}})
	}, 720, 480)
	return view, tt
}

// TestComposerContextRowShowsFacts checks the row's data state: the workspace
// on the left and the default permission mode on the right are drawn as the
// window supplied them, with no WIP badge standing in for a fact the view has.
func TestComposerContextRowShowsFacts(t *testing.T) {
	_, tt := composerContextView(t, "/code/dim-electron", "workspace-write")

	for _, want := range []string{
		"Workspace", "/code/dim-electron", "Default permissions", "workspace-write",
	} {
		if !tt.HasText(want) {
			t.Errorf("%q is missing from the context row, texts %q", want, tt.Texts())
		}
	}

	for _, label := range []string{"Workspace", "Default permissions"} {
		info, ok := tt.AccessByRole(ui.RoleGroup, label)
		if !ok {
			t.Fatalf("%s is not a named group, texts %q", label, tt.Texts())
		}
		if info.Disabled || info.Focusable {
			t.Errorf("%s is an entry of the row, not a control: disabled=%v focusable=%v",
				label, info.Disabled, info.Focusable)
		}
		if strings.Contains(info.Description, "Unavailable") {
			t.Errorf("%s has a fact but still explains a gap: %q", label, info.Description)
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Workspace"); ok {
		t.Error("the workspace is exposed as a button without an action behind it")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Default permissions"); ok {
		t.Error("the default permission mode is exposed as a button without an action behind it")
	}

	send, ok := tt.Find("Send")
	if !ok {
		t.Fatalf("the composer offers no Send entry, texts %q", tt.Texts())
	}
	workspace, ok := tt.Find("Workspace")
	if !ok {
		t.Fatalf("the workspace entry has no box, texts %q", tt.Texts())
	}
	if workspace.Y < send.Y+send.H {
		t.Errorf("the context row sits at y=%.1f, not below the tool row ending at %.1f",
			workspace.Y, send.Y+send.H)
	}
}

// TestComposerContextRowShowsWIPWithoutFacts checks the empty state: with no
// workspace or permission fact the row draws the WIP badge and the real reason
// it waits on, and invents no path or mode.
func TestComposerContextRowShowsWIPWithoutFacts(t *testing.T) {
	_, tt := composerContextView(t, "", "")

	for _, tc := range []struct {
		label, reason string
	}{
		{"Workspace", reasonWorkspace},
		{"Default permissions", reasonPermission},
	} {
		info, ok := tt.AccessByRole(ui.RoleGroup, tc.label)
		if !ok {
			t.Fatalf("%s is not a named group, texts %q", tc.label, tt.Texts())
		}
		if !strings.Contains(info.Description, tc.reason) {
			t.Errorf("%s description %q does not explain the gap %q",
				tc.label, info.Description, tc.reason)
		}
		if info.Focusable {
			t.Errorf("%s takes the keyboard focus, so it is a Tab stop", tc.label)
		}
		rect, ok := tt.Find(tc.label)
		if !ok {
			t.Fatalf("%s has no drawn box", tc.label)
		}
		if rect.H <= 0 || rect.W <= 0 {
			t.Errorf("%s has no visible area: %+v", tc.label, rect)
		}
	}
	if !tt.HasText(wipBadge) {
		t.Errorf("the context row shows no %q badge, texts %q", wipBadge, tt.Texts())
	}
}

// TestComposerContextRowLocalizes checks that the row's labels are the
// locale's, not the region's English constants.
func TestComposerContextRowLocalizes(t *testing.T) {
	view, _ := composerContextView(t, "", "")
	view.Locale = LocaleZHCN
	tt := ui.NewTester(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, Actions{})
	}, 720, 480)

	for _, want := range []string{"工作区", "默认权限"} {
		if !tt.HasText(want) {
			t.Errorf("%q is missing from the localized context row, texts %q", want, tt.Texts())
		}
	}
}

// TestComposerSendIsRoundAndCompact checks the Send entry's geometry: the
// spec's h-7 w-7 disc, square and round (one width, height and a radius of half
// of it), drawn inside the tool row.
func TestComposerSendIsRoundAndCompact(t *testing.T) {
	s := fixture()
	s.Connection = ConnReady
	s.Draft = "hello"
	s.OnSubmit = func(string) {}
	tt := ui.NewTester(s.View, 900, 600)

	rect, ok := tt.Find("Send")
	if !ok {
		t.Fatalf("the composer offers no Send entry, texts %q", tt.Texts())
	}
	want := newTokens(ui.LightTheme()).space(composerSendSpan)
	if math.Abs(float64(rect.W-rect.H)) > 0.5 {
		t.Errorf("Send is %.1fx%.1f, not the square box a circle needs", rect.W, rect.H)
	}
	if math.Abs(float64(rect.W-want)) > 0.5 {
		t.Errorf("Send is %.1f wide, want the %.0f DIP disc the spec's h-7 w-7 names",
			rect.W, want)
	}

	// The editor is the card's widest content: the disc sits inside the same
	// row as the attachment entry and to its right.
	attach, ok := tt.Find("Attach file")
	if !ok {
		t.Fatalf("the composer offers no Attach file entry, texts %q", tt.Texts())
	}
	if rect.X <= attach.X {
		t.Errorf("Send sits at x=%.1f, left of Attach file at %.1f", rect.X, attach.X)
	}
	if rect.Y+rect.H > attach.Y+attach.H+attach.H {
		t.Errorf("Send at y=%.1f is not on the tool row", rect.Y)
	}

	info, ok := tt.AccessByRole(ui.RoleButton, "Send")
	if !ok {
		t.Fatalf("Send is not exposed as a button, texts %q", tt.Texts())
	}
	if info.Disabled {
		t.Error("Send is disabled while the draft can be submitted")
	}
}

// TestComposerSendDisabledWithoutDraft checks that the round button follows the
// same gate as before: an empty draft leaves it disabled and unpressable.
func TestComposerSendDisabledWithoutDraft(t *testing.T) {
	_, tt := composerContextView(t, "", "")

	info, ok := tt.AccessByRole(ui.RoleButton, "Send")
	if !ok {
		t.Fatalf("the composer offers no Send entry, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Fatal("Send is available with an empty draft")
	}
	if tt.Focused("Send") {
		t.Fatal("a disabled Send took the focus")
	}
}

// TestComposerContextRowKeepsTabOrder checks that the row adds no Tab stop: the
// documented order, Attach file then the editor, is unchanged with an empty
// draft, and neither context entry is focusable.
func TestComposerContextRowKeepsTabOrder(t *testing.T) {
	_, tt := composerContextView(t, "", "")

	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Attach file") {
		t.Fatalf("Tab does not reach Attach file first, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Composer") {
		t.Fatalf("Tab does not reach the editor next, texts %q", tt.Texts())
	}
	for _, label := range []string{"Workspace", "Default permissions"} {
		info, ok := tt.AccessByRole(ui.RoleGroup, label)
		if !ok {
			t.Fatalf("%s is not drawn, texts %q", label, tt.Texts())
		}
		if info.Focusable {
			t.Errorf("%s takes the keyboard focus", label)
		}
	}
}

// TestComposerContextRowFitsNarrowWindow checks that the row keeps both entries
// inside a narrow card: the workspace and the permission mode do not overlap,
// neither runs past the card's inner edge, and the round Send entry stays
// visible on the same width.
func TestComposerContextRowFitsNarrowWindow(t *testing.T) {
	const width = 360
	view := ComposerView{
		Draft:       new(string),
		CanSubmit:   true,
		Placeholder: composerPlaceholder,
		Workspace:   "dim-electron",
		Permission:  "workspace-write",
		Shortcut:    KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
		Focus:       new(ui.Handle),
	}
	tt := ui.NewTester(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, Actions{OnSubmit: func(string) {}})
	}, width, 480)

	// The card's inner edge: the window less the card's inset and its hairline
	// border. A right edge past it would draw over the border.
	k := newTokens(ui.LightTheme())
	inner := float32(width) - k.edgePad - k.hairline

	workspace, ok := tt.Find("Workspace")
	if !ok {
		t.Fatalf("the workspace entry is not drawn, texts %q", tt.Texts())
	}
	permission, ok := tt.Find("Default permissions")
	if !ok {
		t.Fatalf("the permission entry is not drawn, texts %q", tt.Texts())
	}
	if workspace.X < 0 || workspace.X+workspace.W > inner {
		t.Errorf("the workspace entry %.1f..%.1f runs outside the card's inner edge at %.1f",
			workspace.X, workspace.X+workspace.W, inner)
	}
	if permission.X+permission.W > inner {
		t.Errorf("the permission entry ends at %.1f, past the card's inner edge at %.1f",
			permission.X+permission.W, inner)
	}
	if workspace.X+workspace.W > permission.X {
		t.Errorf("the context entries overlap: workspace ends at %.1f, permission starts at %.1f",
			workspace.X+workspace.W, permission.X)
	}

	send, ok := tt.Find("Send")
	if !ok {
		t.Fatalf("the composer offers no Send entry, texts %q", tt.Texts())
	}
	if send.W <= 0 || send.H <= 0 {
		t.Errorf("Send has no visible area in the narrow window: %+v", send)
	}
	if send.X+send.W > inner {
		t.Errorf("Send ends at %.1f, past the card's inner edge at %.1f", send.X+send.W, inner)
	}
}

// TestShellComposerContextRowIsWIP checks the shell's assembly: the window has
// no workspace or permission fact today, so the context row draws its WIP state
// rather than a local guess.
func TestShellComposerContextRowIsWIP(t *testing.T) {
	s := fixture()
	if s.Workspace != "" || s.Permission != "" {
		t.Fatalf("the fixture invented a context fact: workspace=%q permission=%q",
			s.Workspace, s.Permission)
	}
	tt := ui.NewTester(s.View, 900, 600)

	for _, tc := range []struct {
		label, reason string
	}{
		{"Workspace", reasonWorkspace},
		{"Default permissions", reasonPermission},
	} {
		info, ok := tt.AccessByRole(ui.RoleGroup, tc.label)
		if !ok {
			t.Fatalf("%s is not drawn by the shell, texts %q", tc.label, tt.Texts())
		}
		if !strings.Contains(info.Description, tc.reason) {
			t.Errorf("%s description %q does not explain the gap %q",
				tc.label, info.Description, tc.reason)
		}
	}
}
