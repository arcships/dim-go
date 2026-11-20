package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是审批卡的键盘取证：卡出现即拿焦点（R-IN-842「焦点自动进入卡片」），
// Tab / ⇧Tab 在卡内循环、不逃到背景，Esc 不答复也不把焦点放走。
//
// 焦点口径同 evidence_keyboard_test.go：读 gui.Tester 报的焦点位置，不看内部
// 字段；「在卡内」用卡里的文案判定（焦点落在卡内任何一个元素上时，卡里的文字
// 都在焦点链上）。
//
// 焦点落在卡容器而不是某个决定上：U3 F4 规定卡首次出现只把焦点交给卡容器、
// 不预选任何决定（预选「允许」会让 Enter 变成误批准），Tab 才到第一个动作。

// approvalFocusInside 报告焦点是否还在卡内（卡内一段文案在焦点链上）。
func approvalFocusInside(tt *gui.Tester) bool {
	return tt.Focused("DimAgent 想要：执行") || tt.Focused(approvalInputLabel)
}

// TestApprovalCardTakesFocusAndKeepsItInside：卡出现即聚焦、Tab/⇧Tab 在卡内循环、
// Esc 不答复也不放走焦点。
func TestApprovalCardTakesFocusAndKeepsItInside(t *testing.T) {
	app, tt := newTester(t, StateWaiting)
	if _, ok := app.currentApproval(); !ok {
		t.Skip("走查态没有审批卡")
	}
	// 卡出现即拿焦点：焦点在卡内，而且没有任何一个决定被预选。
	if !approvalFocusInside(tt) {
		t.Fatalf("审批卡没有拿到焦点：%v", tt.Texts())
	}
	for _, label := range []string{approvalCopyReject, approvalCopyAllowOnce, approvalCopyAlwaysCommand} {
		if focusOn(tt, label) {
			t.Fatalf("卡出现就预选了决定 %q", label)
		}
	}

	// Tab 依次走到三个决定，再回到卡容器；⇧Tab 从卡容器走到最后一个决定。
	order := []string{approvalCopyReject, approvalCopyAllowOnce, approvalCopyAlwaysCommand}
	for i, label := range order {
		tt.Key(0, gui.KeyTab)
		if !focusOn(tt, label) {
			t.Fatalf("第 %d 次 Tab 之后焦点不在 %q 上", i+1, label)
		}
	}
	tt.Key(0, gui.KeyTab)
	if !approvalFocusInside(tt) {
		t.Fatal("第三次 Tab 之后的焦点逃到卡外")
	}
	for _, label := range order {
		if focusOn(tt, label) {
			t.Fatalf("Tab 循环回到决定 %q，想要回到卡容器", label)
		}
	}
	tt.Key(gui.Shift, gui.KeyTab)
	if !focusOn(tt, approvalCopyAlwaysCommand) {
		t.Fatal("⇧Tab 没有从卡容器循环到最后一个决定")
	}

	// Esc 不答复，卡不关，焦点留在卡内。
	before := app.approvalDecision
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.approvalDecision != before {
		t.Fatalf("Esc 改动了审批答复：%q", app.approvalDecision)
	}
	if !tt.HasText(approvalCopyReject) {
		t.Fatalf("Esc 之后审批卡消失了：%v", tt.Texts())
	}
	if !focusOn(tt, approvalCopyAlwaysCommand) {
		t.Fatal("Esc 之后焦点离开了卡")
	}
}

// TestApprovalCardFocusDoesNotLeakToBackground：焦点在卡上时 Tab 只走卡内环，
// 背景（左栏、对话流、输入区）一个停靠点都拿不到。
func TestApprovalCardFocusDoesNotLeakToBackground(t *testing.T) {
	_, tt := newTester(t, StateWaiting)
	inside := map[string]bool{
		approvalCopyReject:        true,
		approvalCopyAllowOnce:     true,
		approvalCopyAlwaysCommand: true,
	}
	for i := 0; i < 8; i++ {
		tt.Key(0, gui.KeyTab)
		for _, outside := range []string{"左栏 · 会话列表", "对话流", "新建会话", "收起侧边栏 ⌘B"} {
			if focusOn(tt, outside) {
				t.Fatalf("第 %d 次 Tab 之后焦点逃到卡外的 %q", i+1, outside)
			}
		}
		onDecision := false
		for label := range inside {
			onDecision = onDecision || focusOn(tt, label)
		}
		if !onDecision && !approvalFocusInside(tt) {
			t.Fatalf("第 %d 次 Tab 之后焦点既不在决定上也不在卡内", i+1)
		}
	}
}
