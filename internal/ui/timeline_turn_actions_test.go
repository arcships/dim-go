package ui

import (
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件覆盖轮末动作条（TL-13 / TL-61 / TL-63）与工具行详情（TL-55）这三处
// 本轮补上的行为：复制整轮的勾反馈、重试追加新轮与在途禁用、停止态只留可用项、
// 详情内容限高滚动、以及消息行悬停动作的键盘入口。

// hoverAssistant 把指针移到助手区上，让轮末动作条显形（悬停可见）。
func hoverAssistant(t *testing.T, tt *gui.Tester) {
	t.Helper()
	rect, ok := tt.Find("助手消息")
	if !ok {
		t.Fatalf("找不到助手区：%v", tt.Texts())
	}
	tt.Move(rect.X+rect.W/2, rect.Y+rect.H/2)
	tt.Frame()
}

// TestTurnCopyWritesClipboardAndShowsCheck：复制整轮助手内容写进剪贴板，图标在
// 反馈窗口内变勾（读屏名带「已复制」），窗口过后按时间算回复制（TL-601 / TL-607）。
func TestTurnCopyWritesClipboardAndShowsCheck(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	hoverAssistant(t, tt)
	if err := tt.Click("复制整个消息"); err != nil {
		t.Fatalf("点不到复制整个消息：%v", err)
	}
	tt.Frame()
	want := assistantMessageText(app.rounds[0])
	if got := tt.Clipboard(); got != want {
		t.Fatalf("剪贴板 %q，想要 %q", got, want)
	}
	if _, ok := tt.Find("复制整个消息（已复制）"); !ok {
		t.Fatalf("复制后没有勾反馈：%v", tt.Texts())
	}
	// 反馈窗口过后回到复制图标（按时间算，不靠下一次点击）。
	tt.Advance(turnCopiedFeedback + time.Second)
	tt.Frame()
	if _, ok := tt.Find("复制整个消息"); !ok {
		t.Fatalf("反馈过后没有回到复制：%v", tt.Texts())
	}
}

// TestTurnRetryAppendsRoundAndDisablesWhileInFlight：重试以同一输入追加一轮，
// 旧轮保留；新轮在跑时按钮禁用，连点只新增一轮（TL-619 / R-XR-123 / R-XR-124）。
func TestTurnRetryAppendsRoundAndDisablesWhileInFlight(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	before := len(app.rounds)
	input := app.rounds[0].user
	hoverAssistant(t, tt)
	if err := tt.Click("重试"); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	tt.Frame()
	if len(app.rounds) != before+1 {
		t.Fatalf("重试后轮数 %d，想要 %d", len(app.rounds), before+1)
	}
	if got := app.rounds[len(app.rounds)-1]; got.user != input {
		t.Fatalf("新轮输入 %q，想要原轮输入 %q", got.user, input)
	}
	if got := app.rounds[len(app.rounds)-1].status; got != roundRunning {
		t.Fatalf("新轮状态 %v，想要运行中", got)
	}
	// 第一次点击之后旧轮仍在原处，可再重试。
	hoverAssistant(t, tt)
	if _, ok := tt.Find("重试（" + noteRetryInFlight + "）"); !ok {
		t.Fatalf("在途时重试没有写原因：%v", tt.Texts())
	}
	if err := tt.Click("重试（" + noteRetryInFlight + "）"); err == nil {
		// 禁用按钮点下去不生效：轮数不变。
		if len(app.rounds) != before+1 {
			t.Fatalf("在途连点又加了一轮：%d", len(app.rounds))
		}
	}
}

// TestTurnActionsHiddenWhileRunningAndTrimmedWhenStopped：运行中的轮脚不出动作条；
// 停止态只留复制与重试，分叉置灰写原因（TL-13 / R-XR-173）。
func TestTurnActionsHiddenWhileRunningAndTrimmedWhenStopped(t *testing.T) {
	app, tt := newTester(t, StateRunning)
	// 把悬停钉在运行中的那一轮上（走查变体用的同一处视图状态），动作条仍不该出现。
	running := -1
	for i, r := range app.rounds {
		if r.status == roundRunning {
			running = i
		}
	}
	if running < 0 {
		t.Fatal("运行态样例里没有运行中的轮")
	}
	app.timelineHover = running
	tt.Frame()
	if _, ok := tt.Find("助手动作"); ok {
		t.Fatal("运行中的轮脚不该出动作条")
	}

	stopped, stoppedTT := newTester(t, StateDefault)
	stopped.rounds[0].status = roundStopped
	stoppedTT.Frame()
	hoverAssistant(t, stoppedTT)
	if _, ok := stoppedTT.Find("复制整个消息"); !ok {
		t.Fatal("停止态仍应能复制已产生的内容")
	}
	if _, ok := stoppedTT.Find("重试"); !ok {
		t.Fatal("停止态仍应能重试")
	}
	if !stoppedTT.HasText("从此处分叉（") {
		t.Fatalf("停止态分叉没有写不可用原因：%v", stoppedTT.Texts())
	}
}

// TestToolDetailScrollsLongContent：详情每一段内容有上限高，超出的部分在段内
// 双向滚动，不把卡片撑高（TL-55 / R-XR-113）。
func TestToolDetailScrollsLongContent(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.rounds[0].ops = []op{{
		action: "执行", target: "go test ./...",
		input:  "command: go test ./...",
		output: strings.Repeat("FAIL internal/ui\n", 80),
		cause:  "权限不足，命令没有执行",
		failed: true,
	}}
	tt.Frame()
	// 工具行在分组里：先展开组头，再点工具行。
	header := "轮次组头 " + groupSummary(app.rounds[0]) + "，状态：" +
		statusWord(app.rounds[0].status) + "，" + roundHeaderPhrase(app.rounds[0])
	if err := tt.Click(header); err != nil {
		t.Fatalf("点不到工具组头：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if err := tt.Click("执行 go test ./..."); err != nil {
		t.Fatalf("点不到工具行：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.Find("工具详情：输出"); !ok {
		t.Fatalf("没有输出段：%v", tt.Texts())
	}
	rect, ok := tt.Find("工具详情内容")
	if !ok {
		t.Fatalf("详情内容没有滚动容器：%v", tt.Texts())
	}
	if rect.H > toolDetailMaxHeight+1 {
		t.Fatalf("详情内容高 %v，超过上限 %v", rect.H, float32(toolDetailMaxHeight))
	}
}

// TestMessageActionsReachableByKeyboard：消息行本身是可聚焦行，键盘 Tab 进到用户
// 消息上时动作条同样显形（悬停之外的第二条入口）。
func TestMessageActionsReachableByKeyboard(t *testing.T) {
	_, tt := newTester(t, StateDefault)
	if _, ok := tt.Find("复制整个消息 1"); ok {
		t.Fatal("未聚焦时不该出现用户消息动作")
	}
	reached := false
	for i := 0; i < 80; i++ {
		tt.Key(0, gui.KeyTab)
		tt.Frame()
		if _, ok := tt.Find("复制整个消息 1"); ok {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatalf("Tab 走不到用户消息动作：%v", tt.Texts())
	}
}
