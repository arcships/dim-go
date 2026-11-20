package ui

import (
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件覆盖时间线里三处「没有内容时怎么收」的行为：思考块没有正文时不展开
// 空区（TL-52）、分组内容的轨道与缩进（TL-74）、恢复遮罩挡住内容（TL-71）。

// TestGroupOpsRailIndentsRows：分组内容有左侧轨道，工具行缩进在轨道右侧
// （TL-74 / R-TL-730）。
func TestGroupOpsRailIndentsRows(t *testing.T) {
	app, tt := newTester(t, StateRunning)
	// 运行中的最新一轮默认展开：轨道与工具行都在屏上。
	rail, ok := tt.Find("分组轨道")
	if !ok {
		t.Fatalf("没有分组轨道：%v", tt.Texts())
	}
	row, ok := tt.Find("读取 composer.go")
	if !ok {
		t.Fatalf("没有工具行：%v", tt.Texts())
	}
	if row.X <= rail.X {
		t.Fatalf("工具行没有缩进到轨道右侧：行 x=%v，轨道 x=%v", row.X, rail.X)
	}
	_ = app
}

// TestRestoreMaskCoversTimeline：装载第一轮时只画恢复遮罩，不铺半截内容
// （TL-71 / R-TL-700）。
func TestRestoreMaskCoversTimeline(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.source = &fakeSource{
		conn: Connection{Phase: "connected"}, names: []string{"恢复中"},
		hasSess: true,
		session: SessionSnapshot{
			ID: "s1", Title: "恢复中", LoadPhase: "loading_first_turn",
			Rows: liveFactSession().Rows,
		},
	}
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("恢复遮罩"); !ok {
		t.Fatalf("装载第一轮时没有恢复遮罩：%v", tt.Texts())
	}
	if tt.HasText("我把工具行接上了。") {
		t.Fatal("恢复期间不该铺出时间线内容")
	}
}

// TestThinkingBlockPresenceKeepsChevron：思考块给展开箭头，点开后有引用块。
func TestThinkingBlockPresenceKeepsChevron(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	if !tt.HasText("思考展开指示") {
		t.Fatalf("有正文的思考块没有箭头：%v", tt.Texts())
	}
	if err := tt.Click("思考过程 思考了 0.3s"); err != nil {
		t.Fatalf("点不到思考块：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.Find("思考轨道"); !ok {
		t.Fatalf("展开后没有引用块：%v", tt.Texts())
	}
}
