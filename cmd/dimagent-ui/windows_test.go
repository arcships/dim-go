package main

import (
	"testing"

	gui "github.com/egoist/mygo/ui"

	appui "github.com/arcships/dimagent-mygo/internal/ui"
)

// testWindowManager 是一套不碰真实窗口的窗口管理：窗口只记索引与会话名，
// 视图用样例界面（回执行能真的画出来看）。
func testWindowManager(t *testing.T) *windowManager {
	t.Helper()
	m := newWindowManager(windowManagerOptions{
		mode:   modeDemo,
		state:  appui.StateDefault,
		width:  1024,
		height: 768,
	})
	m.create = func(spec windowSpec, index int) *windowSlot {
		slot := &windowSlot{
			index:     index,
			sessionID: spec.sessionID,
			titles:    newWindowTitle(modeDemo, nil),
			view:      appui.NewApp(appui.Options{State: appui.StateDefault}),
		}
		if spec.settings != "" {
			if err := slot.view.ShowSettingsSection(spec.settings); err != nil {
				t.Fatalf("假窗口设置分节：%v", err)
			}
		}
		return slot
	}
	return m
}

// hasReason 报告某个窗口的回执行上是否写着这条原因（画出来，真的看得见）。
// 回执行是一整行文字，按包含关系找。
func hasReason(t *testing.T, slot *windowSlot, reason string) bool {
	t.Helper()
	tt := gui.NewTester(slot.view.View, appui.WindowWidth, appui.WindowHeight)
	tt.Frame()
	return tt.HasText(reason)
}

// TestOpenInNewWindowReusesExistingWindow 核对「已有窗口打开同一会话时把它带到
// 前台，不重复开」：第二次请求不新增窗口，聚焦的是第一个窗口。
func TestOpenInNewWindowReusesExistingWindow(t *testing.T) {
	m := testWindowManager(t)
	main := m.open(windowSpec{})

	first, err := m.OpenSessionFromMenu("参考")
	if err != nil {
		t.Fatalf("在新窗口中打开：%v", err)
	}
	if first != "已在新窗口中打开" {
		t.Fatalf("回执 = %q", first)
	}
	if m.count() != 2 {
		t.Fatalf("打开后窗口数 = %d，想要 2", m.count())
	}
	opened := m.activeSlot()
	if opened == main || opened.pinned() != "参考" {
		t.Fatalf("新窗口应开在「参考」上：%+v", opened)
	}

	second, err := m.OpenSessionFromMenu("参考")
	if err != nil {
		t.Fatalf("再次打开同一会话：%v", err)
	}
	if second != "已切到打开该会话的窗口" {
		t.Fatalf("复用回执 = %q", second)
	}
	if m.count() != 2 {
		t.Fatalf("复用不该重复开窗：窗口数 = %d", m.count())
	}
	if m.activeSlot() != opened {
		t.Fatal("复用时没有把已打开该会话的窗口带到前台")
	}
}

// TestOpenInNewWindowUnknownSession 核对未知会话不给空窗口，原因可读。
func TestOpenInNewWindowUnknownSession(t *testing.T) {
	m := testWindowManager(t)
	main := m.open(windowSpec{})
	if _, err := m.OpenSessionFromMenu("不存在的会话"); err == nil {
		t.Fatal("未知会话应返回原因")
	}
	if m.count() != 1 || m.activeSlot() != main {
		t.Fatalf("未知会话不该开窗：窗口数 = %d", m.count())
	}
}

// TestDeepLinkSessionReusesWindow 核对深链落到已有窗口：已经显示该会话的窗口被
// 带到前台；没显示时在现有窗口里换过去，不另开窗口。
func TestDeepLinkSessionReusesWindow(t *testing.T) {
	m := testWindowManager(t)
	m.markReady()
	main := m.open(windowSpec{})
	second := m.open(windowSpec{sessionID: "参考"})

	m.routeURL("dimagent://session/参考")
	if m.count() != 2 {
		t.Fatalf("深链不该开新窗口：窗口数 = %d", m.count())
	}
	if m.activeSlot() != second {
		t.Fatal("深链没有切到已经显示该会话的窗口")
	}

	m.routeURL("dimagent://session/旧实现对照")
	if m.count() != 2 {
		t.Fatalf("没有窗口显示该会话时应复用现有窗口：窗口数 = %d", m.count())
	}
	if m.activeSlot() != second || second.pinned() != "旧实现对照" {
		t.Fatalf("现有窗口没有换到该会话：active=%+v pinned=%q", m.activeSlot(), second.pinned())
	}
	_ = main
}

// TestDeepLinkSessionOpensWindowWhenNone 核对没有窗口时深链开一个新窗口。
func TestDeepLinkSessionOpensWindowWhenNone(t *testing.T) {
	m := testWindowManager(t)
	m.markReady()
	m.routeURL("dimagent://session/参考")
	if m.count() != 1 {
		t.Fatalf("没有窗口时深链应开一个：窗口数 = %d", m.count())
	}
	if got := m.activeSlot().pinned(); got != "参考" {
		t.Fatalf("新窗口的会话 = %q，想要 参考", got)
	}
}

// TestDeepLinkQueuedUntilReady 核对「应用未运行时由系统拉起后处理」：URL 先于
// 窗口就绪到达时先排队，就绪之后再落窗。
func TestDeepLinkQueuedUntilReady(t *testing.T) {
	m := testWindowManager(t)
	m.handleURLNow("dimagent://session/参考")
	if m.count() != 0 {
		t.Fatalf("就绪之前不该开窗：窗口数 = %d", m.count())
	}
	m.open(windowSpec{})
	m.markReady()
	if got := m.activeSlot().pinned(); got != "参考" {
		t.Fatalf("排队中的深链没有在就绪后处理：当前会话 = %q", got)
	}
}

// TestDeepLinkUnknownTargetShowsReason 核对未知目标给可读原因：不开新窗口，
// 原因写在当前窗口的回执行上。
func TestDeepLinkUnknownTargetShowsReason(t *testing.T) {
	m := testWindowManager(t)
	m.markReady()
	main := m.open(windowSpec{})

	m.routeURL("dimagent://nope/thing")
	if m.count() != 1 {
		t.Fatalf("未知目标不该开窗：窗口数 = %d", m.count())
	}
	if !hasReason(t, main, "未知的链接目标") {
		t.Fatal("未知目标的回执行上没有可读原因")
	}

	m.routeURL("dimagent://session/并没有这个会话")
	if m.count() != 1 {
		t.Fatalf("未知会话不该开窗：窗口数 = %d", m.count())
	}
	if !hasReason(t, main, "链接里的会话不在目录里：并没有这个会话") {
		t.Fatal("未知会话的回执行上没有可读原因")
	}

	m.routeURL("dimagent://settings/nope")
	if !hasReason(t, main, "链接里的设置分节不存在：nope") {
		t.Fatal("未知分节的回执行上没有可读原因")
	}
	m.routeURL("https://example.com/x")
	if !hasReason(t, main, "不支持的协议") {
		t.Fatal("错误协议的回执行上没有可读原因")
	}
}

// TestDeepLinkSettingsReusesWindow 核对设置深链复用现有窗口：不新开窗口，切到
// 设置屏。
func TestDeepLinkSettingsReusesWindow(t *testing.T) {
	m := testWindowManager(t)
	m.markReady()
	main := m.open(windowSpec{})

	m.routeURL("dimagent://settings/appearance")
	if m.count() != 1 {
		t.Fatalf("设置深链不该开窗：窗口数 = %d", m.count())
	}
	if m.activeSlot() != main {
		t.Fatal("设置深链没有落在现有窗口上")
	}

	m.routeURL("dimagent://settings")
	if m.count() != 1 {
		t.Fatalf("不带分节的设置深链不该开窗：窗口数 = %d", m.count())
	}
}

// TestCloseOneWindowKeepsOthers 核对关一个窗口不影响其他窗口；最后一个关掉才
// 走退出。
func TestCloseOneWindowKeepsOthers(t *testing.T) {
	m := testWindowManager(t)
	closed := 0
	m.opts.onLastClosed = func() { closed++ }

	main := m.open(windowSpec{})
	second := m.open(windowSpec{sessionID: "参考"})

	m.windowClosed(second)
	if closed != 0 {
		t.Fatal("还有窗口时不该退出")
	}
	if m.count() != 1 || m.activeSlot() != main {
		t.Fatalf("关一个窗口影响了主窗口：窗口数 = %d", m.count())
	}

	m.windowClosed(main)
	if closed != 1 {
		t.Fatalf("最后一个窗口关掉应走退出：%d", closed)
	}
}

// TestWindowIndexReuse 核对窗口索引补空位：关掉第二个窗口之后，新窗口沿用
// 索引 1（窗口偏好按索引记忆，不越开越多）。
func TestWindowIndexReuse(t *testing.T) {
	m := testWindowManager(t)
	m.open(windowSpec{})
	second := m.open(windowSpec{sessionID: "参考"})
	if second.index != 1 {
		t.Fatalf("第二个窗口索引 = %d，想要 1", second.index)
	}
	m.windowClosed(second)
	third := m.open(windowSpec{})
	if third.index != 1 {
		t.Fatalf("关掉的槽位应被下一个窗口沿用：索引 = %d，想要 1", third.index)
	}
}

// TestDeepLinkReasonMentionsUsableTargets 核对链接写错时的原因里写着可用形态：
// 用户能从回执行上直接看到该怎么写。
func TestDeepLinkReasonMentionsUsableTargets(t *testing.T) {
	m := testWindowManager(t)
	m.markReady()
	main := m.open(windowSpec{})
	m.routeURL("dimagent://session/")
	if !hasReason(t, main, "dimagent://") {
		t.Fatal("原因里应写着可用的链接形态")
	}
}
