package ui

import (
	"image/png"
	"os"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func TestZZNavProbe(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.screen, app.settingsSection = ScreenSettings, secAccount
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	f, err := os.Create("/tmp/settings-nav.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
	f.Close()
	for _, s := range settingsSectionDefs {
		info, ok := tt.Find(s.label)
		t.Logf("%-8s ok=%v box=%+v icon=%v", s.label, ok, info, tt.HasLabel("图标："+s.label))
	}
}
