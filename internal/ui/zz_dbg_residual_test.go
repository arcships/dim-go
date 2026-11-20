package ui

import (
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

func TestDbgRowHit(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("1 个文件改动"); err != nil {
		t.Fatalf("click: %v", err)
	}
	tt.Advance(timelineCollapse.Duration + 50*time.Millisecond)
	tt.Frame()
	r, ok := tt.Find("改动文件 internal/ui/composer.go")
	t.Logf("row=%+v ok=%v", r, ok)
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	t.Logf("rightOpen=%v tab=%d texts=%d", app.rightOpen, app.rightTab, len(tt.Texts()))
	tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	t.Logf("menu=%v", tt.Menu())
	tt.Frame()
	t.Logf("menu2=%v", tt.Menu())
	has := false
	for _, x := range tt.Texts() {
		if x == "用默认应用打开" {
			has = true
		}
	}
	t.Logf("hasItem=%v", has)
	for k, v := range app.blockOpen {
		if len(k) > 5 && k[:5] == "fileC" {
			t.Logf("menu state %s = %v", k, *v)
		}
	}
	// 对照：折叠头也挂了同一份菜单。
	if err := tt.RightClick("1 个文件改动"); err != nil {
		t.Logf("header right click: %v", err)
	}
	tt.Frame()
	t.Logf("header menu=%v", tt.Menu())
	a2 := NewApp(Options{State: StateDefault})
	tt2 := gui.NewTester(a2.View, WindowWidth, WindowHeight)
	r2, _ := tt2.Find("1 个文件改动")
	_ = r2
	// 对照：会话行有右键菜单。
	if err := tt.RightClick("修发送路径"); err != nil {
		t.Logf("session right click: %v", err)
	}
	tt.Frame()
	t.Logf("session menu=%v", tt.Menu())
}
