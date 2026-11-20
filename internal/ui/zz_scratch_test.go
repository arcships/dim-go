package ui

import (
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

func TestScratchLongOutput(t *testing.T) {
	output := perfCodeLines(50)
	app := perfToolSource(output)
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	t.Logf("open=%v MaxY(before)=%.0f", *app.blockOf("tool:1:0", false), app.feed.MaxY)
	tt.Advance(400 * time.Millisecond)
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
	t.Logf("MaxY=%.0f Y=%.0f", app.feed.MaxY, app.feed.Y)
	if r, ok := tt.Find("工具详情：输出"); ok {
		t.Logf("输出段 rect = %+v", r)
	} else {
		t.Log("没有输出段")
	}

	// 走点击路径对照。
	app2 := perfToolSource(output)
	*app2.blockOf("tool:1:0", false) = false
	tt2 := gui.NewTester(app2.View, WindowWidth, WindowHeight)
	if err := tt2.Click("执行 go test ./... -v"); err != nil {
		t.Fatalf("点不到工具行：%v", err)
	}
	tt2.Advance(300 * time.Millisecond)
	tt2.Frame()
	t.Logf("点击路径：open=%v MaxY=%.0f", *app2.blockOf("tool:1:0", false), app2.feed.MaxY)
	if r, ok := tt2.Find("工具详情：输出"); ok {
		t.Logf("输出段 rect = %+v", r)
	}
}
