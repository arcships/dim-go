package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

func TestZZProbeTabs(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.rightOpen = true
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	for _, label := range []string{"文件", "改动", "后台", "目标", "终端", "侧聊", "子代理"} {
		r, ok := tt.Find(label)
		t.Logf("%s -> %+v ok=%v", label, r, ok)
	}
}
