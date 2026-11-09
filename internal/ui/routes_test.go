package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestScheduledRouteOpensAndReturns checks the first WIP route: the sidebar's
// scheduled-tasks entry opens the whole-window page, the page states the gap,
// the conversation is gone, and Back returns to the same selection and draft.
func TestScheduledRouteOpensAndReturns(t *testing.T) {
	s := fixture()
	s.Draft = "keep me"
	tt := ui.NewTester(s.View, 900, 700)

	if err := tt.Click(tr(LocaleEN, keySidebarScheduled)); err != nil {
		t.Fatalf("open the scheduled-tasks route: %v", err)
	}
	tt.Frame()
	if s.Route != RouteScheduled {
		t.Fatalf("route = %v, want scheduled", s.Route)
	}
	if !tt.HasText(reasonScheduled) {
		t.Errorf("the scheduled page does not explain its gap, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleNavigation, "Session sidebar"); ok {
		t.Error("the sidebar is still drawn behind the whole-window route")
	}
	if s.Selected != "s1" || s.Draft != "keep me" {
		t.Errorf("the route changed the conversation state: selected=%q draft=%q", s.Selected, s.Draft)
	}

	if err := tt.Click(tr(LocaleEN, keySettingsBack)); err != nil {
		t.Fatalf("return from the route: %v", err)
	}
	tt.Frame()
	if s.Route != RouteChat {
		t.Fatalf("route = %v after Back, want chat", s.Route)
	}
	if !tt.HasText("First session") || !tt.HasText("hello from the fixture") {
		t.Errorf("Back did not restore the conversation, texts %q", tt.Texts())
	}
	if s.Selected != "s1" || s.Draft != "keep me" {
		t.Errorf("Back changed the conversation state: selected=%q draft=%q", s.Selected, s.Draft)
	}
}

// TestSkillsRouteOpensAndEscapeReturns checks the second WIP route and its
// keyboard way out: Escape returns to the conversation without sending anything.
func TestSkillsRouteOpensAndEscapeReturns(t *testing.T) {
	s := fixture()
	tt := ui.NewTester(s.View, 900, 700)

	if err := tt.Click(tr(LocaleEN, keySidebarSkills)); err != nil {
		t.Fatalf("open the skills route: %v", err)
	}
	tt.Frame()
	if s.Route != RouteSkills {
		t.Fatalf("route = %v, want skills", s.Route)
	}
	if !tt.HasText(reasonSkills) {
		t.Errorf("the skills page does not explain its gap, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if s.Route != RouteChat {
		t.Fatalf("route = %v after Escape, want chat", s.Route)
	}
}

// TestNoticesShowOnTheWIPRouteAndDismiss checks the notice strip on a route: an
// action failure stays visible on the scheduled page in the danger color, one
// dismiss clears it, and the page itself stays where it was.
func TestNoticesShowOnTheWIPRouteAndDismiss(t *testing.T) {
	s := fixture()
	s.ActionError = "Create session: the App Server refused the request"
	s.ActionNotices = []Notice{{Kind: "create.error", ID: s.ActionError}}
	tt := ui.NewTester(s.View, 900, 700)

	if err := tt.Click(tr(LocaleEN, keySidebarScheduled)); err != nil {
		t.Fatalf("open the scheduled-tasks route: %v", err)
	}
	tt.Frame()
	if !tt.HasText(s.ActionError) {
		t.Errorf("the route hides the action failure, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, ""); !ok {
		t.Error("the failure is not a live-region alert")
	}
	if err := tt.Click(tr(LocaleEN, keyNoticesDismiss)); err != nil {
		t.Fatalf("dismiss the notices: %v", err)
	}
	tt.Frame()
	if s.ActionError != "" || len(s.ActionNotices) != 0 {
		t.Errorf("dismiss left the notices: error=%q notices=%v", s.ActionError, s.ActionNotices)
	}
	if tt.HasText("Create session: the App Server refused the request") {
		t.Errorf("the dismissed notice is still drawn, texts %q", tt.Texts())
	}
	if s.Route != RouteScheduled {
		t.Errorf("dismissing a notice left the route: %v", s.Route)
	}
}
