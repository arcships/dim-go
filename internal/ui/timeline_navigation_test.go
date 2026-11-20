package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件覆盖时间线的两条「按位置重算」行为：回到最新按钮（TL-08）只在滚离
// 底部时出现、点一下恢复跟随；压缩分隔行（TL-67）点开摘要、再点收起。

// TestBackToLatestHiddenAtBottomAndRestoresFollowing：在底部时不常驻按钮；
// 滚离底部才出现；点一下开始滚回底部（TL-064 / R-XR-096 / R-XR-098）。
func TestBackToLatestHiddenAtBottomAndRestoresFollowing(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	// 在底部：没有按钮（不常驻）。
	app.feed.Y, app.feed.MaxY = 0, 0
	tt.Frame()
	if _, ok := tt.Find("回到最新"); ok {
		t.Fatal("在底部时不该有回到最新")
	}

	// 滚离底部超过阈值：按钮出现。
	app.feed.Y, app.feed.MaxY = 0, 400
	tt.Frame()
	if _, ok := tt.Find("回到最新"); !ok {
		t.Fatalf("滚离底部后没有回到最新：%v", tt.Texts())
	}

	// 点一下：开始滚回底部（恢复跟随）。
	if err := tt.Click("回到最新"); err != nil {
		t.Fatalf("点不到回到最新：%v", err)
	}
	if !app.feed.Scrolling() {
		t.Fatal("点击后没有开始滚动")
	}
}

// TestBackToLatestRecalculatesByPosition：显隐判据只看当前滚动位置：底部与
// 阈值内都算「跟随中」，不显示按钮；滚离超过阈值才显示（R-XR-096 / TL-064）。
func TestBackToLatestRecalculatesByPosition(t *testing.T) {
	cases := []struct {
		y, maxY float32
		want    bool
	}{
		{0, 0, false},
		{0, 400, true},
		{390, 400, false},
		{400 - timelineBackThreshold + 1, 400, false},
		{400 - timelineBackThreshold, 400, true},
	}
	for _, tc := range cases {
		feed := gui.ScrollState{Y: tc.y, MaxY: tc.maxY}
		if got := backToLatestVisible(feed); got != tc.want {
			t.Fatalf("位置 %v/%v 的显隐是 %v，想要 %v", tc.y, tc.maxY, got, tc.want)
		}
	}
}

// TestCompressionRowExpandsSummary：压缩分隔行点开摘要区、再点收起；行本身是
// 可展开的分隔行，读屏名报「上下文已压缩」（TL-656 / TL-661）。
func TestCompressionRowExpandsSummary(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	info, ok := tt.Access("上下文已压缩")
	if !ok {
		t.Fatalf("没有压缩分隔行：%v", tt.Texts())
	}
	if info.Role != gui.RoleDisclosure || info.Expanded {
		t.Fatalf("分隔行的读屏口径不对：%+v", info)
	}
	if _, ok := tt.Find("压缩摘要"); ok {
		t.Fatal("收起时不该有摘要区")
	}
	if err := tt.Click("上下文已压缩"); err != nil {
		t.Fatalf("点不到压缩分隔行：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("压缩摘要"); !ok {
		t.Fatalf("展开后没有摘要区：%v", tt.Texts())
	}
	if err := tt.Click("上下文已压缩"); err != nil {
		t.Fatalf("再点不到压缩分隔行：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("压缩摘要"); ok {
		t.Fatal("再点没有收起摘要区")
	}
}
