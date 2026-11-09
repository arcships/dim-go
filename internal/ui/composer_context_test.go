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
func composerContextView(t *testing.T, workspace, permission, branch string) (ComposerView, *ui.Tester) {
	t.Helper()
	view := ComposerView{
		Draft:       new(string),
		CanSubmit:   true,
		Placeholder: composerPlaceholder,
		Workspace:   workspace,
		Permission:  permission,
		Branch:      branch,
		Shortcut:    KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
		Focus:       new(ui.Handle),
	}
	tt := ui.NewTester(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, Actions{OnSubmit: func(string) {}})
	}, 720, 480)
	return view, tt
}

// TestComposerContextRowShowsFacts checks the row's data state: the workspace
// on the left and, on the right, the default permission mode and the session
// branch are drawn as the window supplied them, with no WIP badge standing in
// for a fact the view has. The context meter has no fact here, so it draws its
// dash rather than a number.
func TestComposerContextRowShowsFacts(t *testing.T) {
	_, tt := composerContextView(t, "/code/dim-electron", "workspace-write", "main")

	for _, want := range []string{
		"Workspace", "/code/dim-electron", "Default permissions", "workspace-write",
		"main", composerMeterDash,
	} {
		if !tt.HasText(want) {
			t.Errorf("%q is missing from the context row, texts %q", want, tt.Texts())
		}
	}

	for _, label := range []string{"Workspace", "Default permissions", tr(LocaleEN, composerBranchKey)} {
		info, ok := tt.AccessByRole(ui.RoleGroup, label)
		if !ok {
			t.Fatalf("%s is not a named group, texts %q", label, tt.Texts())
		}
		if info.Disabled || info.Focusable {
			t.Errorf("%s is an entry of the row, not a control: disabled=%v focusable=%v",
				label, info.Disabled, info.Focusable)
		}
	}
	branch, ok := tt.AccessByRole(ui.RoleGroup, tr(LocaleEN, composerBranchKey))
	if !ok {
		t.Fatalf("the session branch is not a named group, texts %q", tt.Texts())
	}
	if !strings.Contains(branch.Description, "not the workspace's git branch") {
		t.Errorf("the branch description %q does not say it is a session branch, not a git branch",
			branch.Description)
	}
	for _, label := range []string{"Workspace", "Default permissions"} {
		info, _ := tt.AccessByRole(ui.RoleGroup, label)
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

// TestComposerContextRowShowsNoWipWithoutFacts checks the empty state: with no
// workspace or permission fact the row keeps the reference's plain labels, shows
// no WIP text, and carries the real reason each entry waits on in its
// description and tooltip instead of inventing a path or mode. The branch entry
// draws nothing at all, because the shell never fills it with a git branch it
// did not read, and the context meter shows its dash rather than a number.
func TestComposerContextRowShowsNoWipWithoutFacts(t *testing.T) {
	_, tt := composerContextView(t, "", "", "")

	for _, tc := range []struct {
		label, reason string
	}{
		{"Workspace", reasonWorkspace},
		{"Default permissions", reasonPermission},
		{tr(LocaleEN, composerContextUsageKey), reasonContext},
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
	if _, ok := tt.AccessByRole(ui.RoleGroup, tr(LocaleEN, composerBranchKey)); ok {
		t.Error("the row draws a branch entry although the view carries no branch")
	}
	if tt.HasText("main") {
		t.Errorf("the row invents the reference's git branch, texts %q", tt.Texts())
	}
	// The dash is a decoration: it paints, and assistive technology does not
	// read it as a value.
	if _, ok := tt.AccessByRole(ui.RoleText, composerMeterDash); ok {
		t.Error("the unknown context meter exposes its dash as accessible text")
	}
	if tt.HasText(wipBadge) {
		t.Errorf("the context row draws the visible %q text, texts %q", wipBadge, tt.Texts())
	}
}

// TestComposerContextRowLocalizes checks that the row's labels are the
// locale's, not the region's English constants.
func TestComposerContextRowLocalizes(t *testing.T) {
	view, _ := composerContextView(t, "", "", "main")
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

// TestComposerContextMeterDrawsKnownUsage checks the meter's data state: a
// known share fills the reference's thin track and drops the dash, while no
// percentage is drawn and the entry stays a read-only display group.
func TestComposerContextMeterDrawsKnownUsage(t *testing.T) {
	view := ComposerView{
		Draft:        new(string),
		CanSubmit:    true,
		Placeholder:  composerPlaceholder,
		ContextKnown: true,
		ContextUsed:  0.5,
		Shortcut:     KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
		Focus:        new(ui.Handle),
	}
	tt := ui.NewTester(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, Actions{})
	}, 720, 480)

	entry, ok := tt.AccessByRole(ui.RoleGroup, tr(LocaleEN, composerContextUsageKey))
	if !ok {
		t.Fatalf("the context meter is not a named group, texts %q", tt.Texts())
	}
	if entry.Disabled || entry.Focusable {
		t.Errorf("the context meter is a control: disabled=%v focusable=%v",
			entry.Disabled, entry.Focusable)
	}
	if entry.Description != "" {
		t.Errorf("a known meter still explains a gap: %q", entry.Description)
	}
	if tt.HasText(composerMeterDash) {
		t.Errorf("a known meter still draws the unknown dash, texts %q", tt.Texts())
	}
	k := newTokens(ui.LightTheme())
	fill, pixels := colorBox(tt, entry.Bounds, k.primary)
	if pixels == 0 {
		t.Fatal("the known meter draws no primary fill")
	}
	if math.Abs(float64(fill.W-composerMeterW/2)) > 3 {
		t.Errorf("the half-full meter fills %.1f DIPs, want ≈%.0f", fill.W, float32(composerMeterW)/2)
	}
	if fill.W > composerMeterW {
		t.Errorf("the fill is %.1f wide, past the meter's %.0f track", fill.W, float32(composerMeterW))
	}
}

// TestComposerContextRowOrder checks the band's order: the workspace stays on
// the left, and the default permission mode, the session branch and the context
// meter follow on the right, as the reference draws its status band.
func TestComposerContextRowOrder(t *testing.T) {
	_, tt := composerContextView(t, "dim-electron", "workspace-write", "main")

	workspace, ok := tt.Find("Workspace")
	if !ok {
		t.Fatalf("the workspace entry is not drawn, texts %q", tt.Texts())
	}
	permission, ok := tt.Find("Default permissions")
	if !ok {
		t.Fatalf("the permission entry is not drawn, texts %q", tt.Texts())
	}
	branch, ok := tt.Find(tr(LocaleEN, composerBranchKey))
	if !ok {
		t.Fatalf("the branch entry is not drawn, texts %q", tt.Texts())
	}
	meter, ok := tt.Find(tr(LocaleEN, composerContextUsageKey))
	if !ok {
		t.Fatalf("the context meter is not drawn, texts %q", tt.Texts())
	}
	if workspace.X+workspace.W > permission.X {
		t.Errorf("the workspace ends at %.1f, past the right-hand entries starting at %.1f",
			workspace.X+workspace.W, permission.X)
	}
	if permission.X+permission.W > branch.X {
		t.Errorf("the permission entry ends at %.1f, past the branch starting at %.1f",
			permission.X+permission.W, branch.X)
	}
	if branch.X+branch.W > meter.X {
		t.Errorf("the branch ends at %.1f, past the meter starting at %.1f",
			branch.X+branch.W, meter.X)
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
	_, tt := composerContextView(t, "", "", "")

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

// TestComposerSendKeepsThePrimaryDiscWhenDisabled checks the empty state's
// Send: the reference draws a solid primary disc, so a disabled Send keeps its
// face instead of fading out. It stays unclickable; the disabled state is what
// assistive technology reads.
func TestComposerSendKeepsThePrimaryDiscWhenDisabled(t *testing.T) {
	_, tt := composerContextView(t, "", "", "")
	send, ok := tt.Find("Send")
	if !ok {
		t.Fatalf("the composer offers no Send entry, texts %q", tt.Texts())
	}
	k := newTokens(ui.LightTheme())
	_, primary := colorBox(tt, send, k.primary)
	if area := send.W * send.H; primary < int(area/2) {
		t.Errorf("the disabled Send shows %d primary pixels in its %.0fx%.0f disc, want the solid face",
			primary, send.W, send.H)
	}
}

// TestComposerContextRowKeepsTabOrder checks that the row adds no Tab stop: the
// documented order, Attach file then the editor, is unchanged with an empty
// draft, and neither context entry is focusable.
func TestComposerContextRowKeepsTabOrder(t *testing.T) {
	_, tt := composerContextView(t, "", "", "main")

	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Attach file") {
		t.Fatalf("Tab does not reach Attach file first, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Composer") {
		t.Fatalf("Tab does not reach the editor next, texts %q", tt.Texts())
	}
	for _, label := range []string{"Workspace", "Default permissions", tr(LocaleEN, composerBranchKey), tr(LocaleEN, composerContextUsageKey)} {
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
	meter, ok := tt.Find(tr(LocaleEN, composerContextUsageKey))
	if !ok {
		t.Fatalf("the context meter is not drawn, texts %q", tt.Texts())
	}
	if workspace.X < 0 || workspace.X+workspace.W > inner {
		t.Errorf("the workspace entry %.1f..%.1f runs outside the card's inner edge at %.1f",
			workspace.X, workspace.X+workspace.W, inner)
	}
	if permission.X+permission.W > inner {
		t.Errorf("the permission entry ends at %.1f, past the card's inner edge at %.1f",
			permission.X+permission.W, inner)
	}
	if meter.X+meter.W > inner {
		t.Errorf("the context meter ends at %.1f, past the card's inner edge at %.1f",
			meter.X+meter.W, inner)
	}
	if workspace.X+workspace.W > permission.X {
		t.Errorf("the context entries overlap: workspace ends at %.1f, permission starts at %.1f",
			workspace.X+workspace.W, permission.X)
	}
	if permission.X+permission.W > meter.X {
		t.Errorf("the permission entry ends at %.1f, past the context meter at %.1f",
			permission.X+permission.W, meter.X)
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
// no workspace or permission fact today, so the entries draw their WIP state
// rather than a local guess. The branch is the selected session's, which the
// shell does have, and the context meter has no usage fact, so it draws its
// dash rather than a number. A conversation moves the workspace out of the
// status band; the workspace's empty state keeps it in the card.
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
		{"Default permissions", reasonPermission},
		{tr(LocaleEN, composerContextUsageKey), reasonContext},
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
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Workspace"); ok {
		t.Error("the conversation's status band still draws the workspace entry")
	}
	if !tt.HasText(composerMeterDash) {
		t.Errorf("the unknown context meter draws no dash, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleGroup, tr(LocaleEN, composerBranchKey)); ok {
		t.Error("the shell draws a branch entry although no branch is selected")
	}

	empty := fixture()
	empty.Sessions = nil
	empty.Selected = ""
	empty.Messages = nil
	ttEmpty := ui.NewTester(empty.View, 900, 600)
	for _, tc := range []struct {
		label, reason string
	}{
		{"Workspace", reasonWorkspace},
		{"Default permissions", reasonPermission},
	} {
		info, ok := ttEmpty.AccessByRole(ui.RoleGroup, tc.label)
		if !ok {
			t.Fatalf("the empty state does not draw %s, texts %q", tc.label, ttEmpty.Texts())
		}
		if !strings.Contains(info.Description, tc.reason) {
			t.Errorf("%s description %q does not explain the gap %q",
				tc.label, info.Description, tc.reason)
		}
	}
}

// TestShellComposerShowsSelectedBranch checks the wiring: the context row's
// branch value is State.SelectedBranch, the App Server's session branch, and
// the shell adds no git branch of its own when there is none.
func TestShellComposerShowsSelectedBranch(t *testing.T) {
	s := fixture()
	s.SelectedBranch = "feat/composer-context"
	tt := ui.NewTester(s.View, 900, 600)

	info, ok := tt.AccessByRole(ui.RoleGroup, tr(LocaleEN, composerBranchKey))
	if !ok {
		t.Fatalf("the branch entry is not drawn by the shell, texts %q", tt.Texts())
	}
	if !strings.Contains(info.Description, "session branch") ||
		!strings.Contains(info.Description, "not the workspace's git branch") {
		t.Errorf("the branch description %q does not name the session branch", info.Description)
	}
	branch, ok := tt.Find("feat/composer-context")
	if !ok {
		t.Fatalf("the selected branch value is not drawn, texts %q", tt.Texts())
	}
	if branch.W <= 0 || branch.H <= 0 {
		t.Errorf("the selected branch value has no visible area: %+v", branch)
	}
}

// TestComposerConversationBandIsOutsideTheCard checks the conversation's
// layout: the card keeps the editor and the tool row only, and the status band
// sits under it with the default permission mode on the left and the session
// branch and context meter on the right. The workspace entry belongs to the
// empty state's card, as the reference's conversation band shows none.
func TestComposerConversationBandIsOutsideTheCard(t *testing.T) {
	view := ComposerView{
		Draft:       new(string),
		CanSubmit:   true,
		Placeholder: composerPlaceholder,
		Wide:        true,
		Permission:  "workspace-write",
		Branch:      "main",
		Shortcut:    KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
		Focus:       new(ui.Handle),
	}
	tt := ui.NewTester(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, Actions{OnSubmit: func(string) {}})
	}, 720, 480)
	card := composerCardBounds(t, tt)

	permission, ok := tt.Find("Default permissions")
	if !ok {
		t.Fatalf("the status band has no permission entry, texts %q", tt.Texts())
	}
	if permission.Y < card.Y+card.H {
		t.Errorf("the permission entry starts at y=%.1f, inside the card ending at %.1f",
			permission.Y, card.Y+card.H)
	}
	if tt.HasText(tr(LocaleEN, keyComposerWorkspace)) {
		t.Errorf("the conversation band still draws the workspace entry, texts %q", tt.Texts())
	}
	for _, label := range []string{tr(LocaleEN, composerBranchKey), tr(LocaleEN, composerContextUsageKey)} {
		rect, ok := tt.Find(label)
		if !ok {
			t.Fatalf("%s is not drawn in the status band, texts %q", label, tt.Texts())
		}
		if rect.Y < card.Y+card.H {
			t.Errorf("%s at y=%.1f sits inside the card ending at %.1f", label, rect.Y, card.Y+card.H)
		}
		if rect.X < permission.X {
			t.Errorf("%s at x=%.1f is not right of the permission entry at %.1f", label, rect.X, permission.X)
		}
	}
	if band, ok := tt.AccessByRole(ui.RoleGroup, tr(LocaleEN, keyComposerStatus)); !ok {
		t.Error("the status band is not a named group")
	} else if band.Bounds.Y < card.Y+card.H {
		t.Errorf("the status band at y=%.1f overlaps the card ending at %.1f", band.Bounds.Y, card.Y+card.H)
	}
}

// TestComposerUnconfiguredLabel checks the muted label beside Send: both states
// draw it, because the shell has no model fact and states that none is
// configured instead of naming one, and it sits left of the send disc in the
// same row.
func TestComposerUnconfiguredLabel(t *testing.T) {
	for _, wide := range []bool{false, true} {
		name := "empty"
		if wide {
			name = "conversation"
		}
		t.Run(name, func(t *testing.T) {
			view := ComposerView{
				Draft:       new(string),
				CanSubmit:   true,
				Placeholder: composerPlaceholder,
				Wide:        wide,
				Shortcut:    KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
				Focus:       new(ui.Handle),
			}
			tt := ui.NewTester(func(c *ui.Context) {
				composer(c, newTokens(c.Theme()), view, Actions{})
			}, 720, 480)
			label := textBox(t, tt, tr(LocaleEN, keyComposerUnconfigured))
			send, ok := tt.Find(tr(LocaleEN, keyComposerSend))
			if !ok {
				t.Fatalf("Send is not drawn, texts %q", tt.Texts())
			}
			if label.X+label.W > send.X {
				t.Errorf("the unconfigured label ends at %.1f, past Send starting at %.1f",
					label.X+label.W, send.X)
			}
			if label.Y < send.Y-1 || label.Y+label.H > send.Y+send.H+1 {
				t.Errorf("the label at y=%.1f is not on the send row at y=%.1f", label.Y, send.Y)
			}
		})
	}
}

// TestComposerNewCopyLocalizes checks that the new composer copy follows the
// interface language like the rest of the region.
func TestComposerNewCopyLocalizes(t *testing.T) {
	view := ComposerView{
		Draft:       new(string),
		CanSubmit:   true,
		Placeholder: composerPlaceholder,
		Wide:        true,
		Branch:      "main",
		Shortcut:    KeyBinding{Mods: ComposerShortcutMods, Key: composerSubmitKey},
		Focus:       new(ui.Handle),
		Locale:      LocaleZHCN,
	}
	tt := ui.NewTester(func(c *ui.Context) {
		composer(c, newTokens(c.Theme()), view, Actions{})
	}, 720, 480)
	for _, want := range []string{"未配置", "默认权限", "输入框状态"} {
		if !tt.HasText(want) {
			t.Errorf("the zh-CN composer is missing %q, texts %q", want, tt.Texts())
		}
	}
	if tt.HasText("Not configured") || tt.HasText("Composer status") {
		t.Errorf("the zh-CN composer still draws English copy, texts %q", tt.Texts())
	}
}
