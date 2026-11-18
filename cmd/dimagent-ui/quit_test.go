package main

import (
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
)

// TestQuitGateWaitsForShutdown 核对退出闸门：四阶段没结束前不算结束，settle
// 会等它跑完才返回，之后 settled 才放行 Quit。
func TestQuitGateWaitsForShutdown(t *testing.T) {
	base, err := app.New(app.Config{DataDir: t.TempDir(), ClientName: "quitgate-test"})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	gate := newQuitGate(base, nil)
	if gate.settled() {
		t.Fatal("还没开始收尾时不该算已结束")
	}
	if !gate.settle(5 * time.Second) {
		t.Fatal("退出四阶段应在上限内结束")
	}
	if !gate.settled() {
		t.Fatal("四阶段结束后应算已结束")
	}
	// 再来一次是空操作：四阶段只跑一次。
	if !gate.settle(0) {
		t.Fatal("四阶段已经结束，settle 应立即返回")
	}
}

// TestQuitGateGivesUpAfterLimit 核对上限：四阶段卡住时按上限放行，不让退出
// 永远停在等待里。
func TestQuitGateGivesUpAfterLimit(t *testing.T) {
	base, err := app.New(app.Config{DataDir: t.TempDir(), ClientName: "quitgate-test"})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	gate := newQuitGate(base, nil)
	// 零上限：第一次 settle 立刻到上限；之后 settled 为真，Quit 不再被拦住。
	if gate.settle(0) {
		t.Fatal("零上限时不该声称四阶段已结束")
	}
	if !gate.settled() {
		t.Fatal("到了上限之后应放行退出")
	}
}
