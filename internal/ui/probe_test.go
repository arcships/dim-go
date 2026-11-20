package ui

import (
	"image/color"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func TestProbeDPI(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.screen, app.settingsSection = ScreenSettings, secAccount
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.SetScale(2)
	tt.Frame()
	r, ok := tt.Find("导航与内容的分隔")
	t.Logf("sep rect %+v ok=%v", r, ok)
	img := tt.Image()
	t.Logf("image bounds %v", img.Bounds())
	// sample column at r.X+0.5 dip * 2
	x := int(r.X*2) + 1
	y := int(r.Y*2) + 5
	t.Logf("px(%d,%d)=%v", x, y, color.RGBAModel.Convert(img.At(x, y)))
	t.Logf("px(%d,%d)=%v", x+1, y, color.RGBAModel.Convert(img.At(x+1, y)))
	t.Logf("px(%d,%d)=%v", x+2, y, color.RGBAModel.Convert(img.At(x+2, y)))
	t.Logf("theme border=%v bg=%v", appThemeBorder, appThemeBG)
}
