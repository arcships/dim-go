package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func TestProbePaletteAccess(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Key(gui.Cmd, gui.KeyK)
	if !app.palette.open {
		t.Fatal("no palette")
	}
	for _, txt := range tt.Texts() {
		if info, ok := tt.Access(txt); ok {
			t.Logf("text=%q -> access role=%v name=%q desc=%q", txt, info.Role, info.Name, info.Description)
		}
	}
	// 组标题
	for _, name := range []string{"分组：动作", "分组：会话", "动作", "停止这次运行：中止当前回复"} {
		if info, ok := tt.Access(name); ok {
			t.Logf("access %q -> role=%v name=%q disabled=%v", name, info.Role, info.Name, info.Disabled)
		} else {
			t.Logf("access %q -> NOT FOUND", name)
		}
	}
}
