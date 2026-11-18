package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func TestDbgHover(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.rightOpen, app.rightTab = true, tabFiles
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tab, _ := tt.Find("文件")
	t.Logf("tab=%+v", tab)
	before := tt.Image().RGBAAt(int(tab.X+4), int(tab.Y+tab.H/2))
	tt.Move(tab.X+tab.W/2, tab.Y+tab.H/2)
	tt.Frame()
	tt.Frame()
	after := tt.Image().RGBAAt(int(tab.X+4), int(tab.Y+tab.H/2))
	t.Logf("before=%v after=%v", before, after)
	_, ok := tt.Access("关闭标签页：文件")
	t.Logf("close access=%v", ok)
}
