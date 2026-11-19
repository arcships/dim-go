package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func TestProbeMagnifier(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	tt.Key(0, gui.KeyTab)
	tt.Frame()
	box, _ := tt.Find("会话内搜索")
	t.Logf("panel %+v active=%v", box, textInputActive(tt))
	if icon, ok := tt.Find("搜索图标"); ok {
		t.Logf("icon %+v", icon)
	} else {
		t.Log("no icon label")
	}
	icon, _ := tt.Find("搜索图标")
	tt.ClickAt(icon.X+icon.W/2, icon.Y+icon.H/2)
	t.Logf("after icon click focus=%v open=%v row=%v", app.sessionSearch.focus, app.sessionSearch.open, app.sessionSearch.rowClicked)
	tt.Frame()
	t.Logf("after frame focus=%v active=%v", app.sessionSearch.focus, textInputActive(tt))
}

func textInputActive(tt *gui.Tester) bool {
	active, _ := tt.TextInputActive()
	return active
}
