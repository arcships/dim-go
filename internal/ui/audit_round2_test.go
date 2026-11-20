package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是 docs/audit-fakes-round2.md §4 里这一批的回归断言（P0-9 侧聊回声、
// P0-10 子代理恒加载、P0-1 有工具名无入参的审批卡、P1-12 重试行）。
//
// 每条断言的都是同一组三件事：**禁用态、原因可见、不产生假状态**。用例都连着
// 注入的 Source（连接模式）：这几条只在"有事实可读、界面却自己编"时才算假，
// -demo 的样例走查不在本文件的口径里。

// P0-9：侧聊没有写入通道（Intents 里没有侧聊方法）。连接模式下输入框与「发送」
// 一起置灰、原因写在输入区上方与 tooltip 上；点发送既不 append 本地切片，也不
// 在只读时间线上画出那条"看起来已发送"的气泡。
func TestSideChatSendDisabledWithoutWriteChannel(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	app.rightOpen, app.rightTab = true, tabSideChat
	app.panels.sideChatDraft = "这条消息会发给谁"
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	// 原因可见（写在输入区上方，不是只在 tooltip 里）。
	if !tt.HasText(noteSideChatWrite) {
		t.Fatalf("没有写清侧聊没有写入通道：%v", tt.Texts())
	}
	input, ok := tt.Access("侧边对话输入")
	if !ok {
		t.Fatalf("侧聊没有输入框：%v", tt.Texts())
	}
	if !input.Disabled {
		t.Fatalf("连接模式下侧聊输入框没有置灰：%+v", input)
	}
	if input.Description != noteSideChatWrite {
		t.Fatalf("输入框没写原因：%+v", input)
	}

	// 发送按钮：按位置点（主输入区也有一个「发送」）。
	app.panels.sideChatDraft = ""
	tt.Frame()
	in := input.Bounds
	app.panels.sideChatDraft = "这条消息会发给谁"
	tt.Frame()
	sendX, sendY := in.X+in.W+space1+panelSendSize/2, in.Y+in.H/2
	tt.ClickAt(sendX, sendY)
	tt.Frame()

	if len(app.panels.sideChatSent) != 0 {
		t.Fatalf("点了发送仍 append 了一条本地回声：%v", app.panels.sideChatSent)
	}
	if app.panels.sideChatDraft != "这条消息会发给谁" {
		t.Fatalf("草稿被清掉了（装作已发出）：%q", app.panels.sideChatDraft)
	}

	// 直接调用也不产生假消息（防止别处绕开按钮）。
	app.sendSideChat()
	if len(app.panels.sideChatSent) != 0 {
		t.Fatalf("sendSideChat 在连接模式下仍写出本地回声：%v", app.panels.sideChatSent)
	}
	if app.panels.sideChatDraft != "这条消息会发给谁" {
		t.Fatalf("sendSideChat 在连接模式下清掉了草稿：%q", app.panels.sideChatDraft)
	}
}

// P0-10：子代理没有事实可说（协议没有子代理会话流，subagentReady / subagentLines
// 都没有写入点）。连接模式下给空态「未打开子代理会话」+ 原因，绝不恒转圈，也不
// 拿样例顶上。
func TestSubagentPanelEmptyStateWithoutFacts(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	app.rightOpen, app.rightTab = true, tabSubagent
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	// 连跑几帧：状态不随帧变化，屏上始终是空态与原因。
	for i := 0; i < 3; i++ {
		tt.Frame()
		if !tt.HasText(subagentNoSession) {
			t.Fatalf("第 %d 帧没有空态标题：%v", i, tt.Texts())
		}
		if !tt.HasText(noteSubagentFacts) {
			t.Fatalf("第 %d 帧没写原因：%v", i, tt.Texts())
		}
		if tt.HasText("正在加载...") {
			t.Fatalf("第 %d 帧仍在转圈：%v", i, tt.Texts())
		}
	}
	if len(app.panels.subagentLines) != 0 {
		t.Fatalf("空态下自己造了消息：%v", app.panels.subagentLines)
	}
	if tt.HasText(subagentSamples()[0]) {
		t.Fatalf("连接模式画了样例消息：%v", tt.Texts())
	}

	// 出错时「重试」没有重读通道：置灰写原因，点了不清错误（清了也只是空转）。
	app.panels.subagentError = "读取子代理会话失败"
	tt.Frame()
	retry, ok := tt.Access(subagentRetry)
	if !ok {
		t.Fatalf("错误态没有重试入口：%v", tt.Texts())
	}
	if !retry.Disabled {
		t.Fatalf("没有读取通道的重试仍可点：%+v", retry)
	}
	if retry.Description != noteSubagentFacts {
		t.Fatalf("重试没写原因：%+v", retry)
	}
	_ = tt.Click(subagentRetry)
	tt.Frame()
	if app.panels.subagentError == "" {
		t.Fatal("点没有通道的重试却把错误清掉了（假动作）")
	}
}

// P0-1 残留：有工具名、读不到入参时，卡片不画内容区，但要写清读不到什么，并把
// 「拒绝 / 仅这次」两个决定一起置灰——看不到要执行什么就不放行；这张快照没有
// 长期规则，长期允许整块不画（R-AP-214）。
func TestApprovalCardMissingInputDisablesDecisions(t *testing.T) {
	src := wiredSource()
	src.hasAppr = true
	src.approval = ApprovalSnapshot{ID: "a9", Tool: "exec", Input: ""}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if !tt.HasText(noteApprovalInput) {
		t.Fatalf("没有写清读不到入参：%v", tt.Texts())
	}
	if !tt.HasText(noteApprovalBlocked) {
		t.Fatalf("没有写清为什么不能决定：%v", tt.Texts())
	}
	p0ButtonsDisabled(t, tt, "拒绝", "仅这次")
	if info, ok := tt.Access("始终允许此命令"); ok {
		t.Fatalf("没有长期规则时仍画了长期允许：%+v", info)
	}

	// 置灰的入口不能真的答复（两个决定都点一遍）。
	for _, label := range []string{"拒绝", "仅这次"} {
		_ = tt.Click(label)
		tt.Frame()
		if app.approvalDecision != "" {
			t.Fatalf("点 %q 答出了一个看不到内容的决定：%q", label, app.approvalDecision)
		}
	}
}
