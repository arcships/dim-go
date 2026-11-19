package main

import (
	"strings"
	"testing"

	appui "github.com/arcships/dimagent-mygo/internal/ui"
)

// TestWindowTitleFollowsConnectionPhase 核对标题的取法：启动时带着发现说明，
// 连上之后回到产品标题（启动说明作废），断线点明连接失败。
func TestWindowTitleFollowsConnectionPhase(t *testing.T) {
	titles := newWindowTitle(modeConnect, nil)
	titles.setHint("未找到 App Server")

	if got, want := titles.forConnection(appui.Connection{Phase: "connecting"}),
		titleProduct+" — 未找到 App Server"; got != want {
		t.Fatalf("连接中的标题 = %q, want %q", got, want)
	}
	if got := titles.forConnection(appui.Connection{Phase: connectionPhaseConnected}); got != titleProduct {
		t.Fatalf("连上后的标题 = %q, want %q（启动说明不再算数）", got, titleProduct)
	}
	if got := titles.forConnection(appui.Connection{
		Phase: connectionPhaseDisconnected, Reason: "端点连不上",
	}); got != titleConnectFailed {
		t.Fatalf("断线标题 = %q, want %q", got, titleConnectFailed)
	}
}

// TestWindowTitleDemoKeepsSkeletonSuffix 核对样例模式仍带「界面骨架」字样：
// 截图脚本按它认窗口。
func TestWindowTitleDemoKeepsSkeletonSuffix(t *testing.T) {
	titles := newWindowTitle(modeDemo, nil)
	got := titles.base(appui.StateFailed)
	if !strings.HasPrefix(got, titleProduct+" — ") || !strings.Contains(got, "（界面骨架）") {
		t.Fatalf("样例模式标题 = %q, want 「%s — <状态>（界面骨架）」", got, titleProduct)
	}
}
