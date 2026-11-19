package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func TestSlashDebug(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Click("消息")
	t.Logf("after click: focused=%v draft=%q", tt.Focused("消息"), app.draft)
	tt.Type("/")
	t.Logf("after type: draft=%q open=%v prefix=%q", app.draft, app.slash.open, app.slash.prefix)
	tt.Frame()
	t.Logf("after frame: draft=%q open=%v prefix=%q", app.draft, app.slash.open, app.slash.prefix)
	if p, ok := slashMenuPrefix(app.draft); true {
		t.Logf("prefix ok=%v p=%q", ok, p)
	}
	t.Logf("texts=%v", tt.Texts())
}
