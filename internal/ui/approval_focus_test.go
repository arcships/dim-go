package ui

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是审批卡的键盘取证：卡出现即拿焦点（R-IN-842「焦点自动进入卡片」），
// Tab / ⇧Tab 在卡内循环、不逃到背景，Esc 不答复也不把焦点放走。
//
// 焦点口径同 evidence_keyboard_test.go：读 gui.Tester 报的焦点位置，不看内部
// 字段。卡容器自己是否拿着焦点用 Access 的 Focused 判（只认它自己，不认卡内
// 元素）；「焦点还在卡内」用卡里的标题文字判（焦点落在卡内任何一处时，卡里的
// 文字都在焦点链上）。
//
// 焦点落在卡容器而不是某个决定上：U3 F4 规定卡首次出现只把焦点交给卡容器、
// 不预选任何决定（预选「允许」会让 Enter 变成误批准），Tab 才到第一个动作。

// approvalCardTitle 是走查态审批卡的标题（卡里第一段文字）。
const approvalCardTitle = "DimAgent 想要：执行"

// approvalFocusOnCard 报告焦点是否落在卡容器自己身上。
func approvalFocusOnCard(tt *gui.Tester) bool {
	return focusOn(tt, "审批卡")
}

// approvalFocusInside 报告焦点是否还在卡内：找到拿着焦点的那个控件，看它的框在
// 不在卡片的框里（卡容器自己拿着焦点时两者相等）。
func approvalFocusInside(tt *gui.Tester) bool {
	card, ok := tt.Find("审批卡")
	if !ok {
		return false
	}
	for _, name := range tt.Texts() {
		if !focusOn(tt, name) {
			continue
		}
		r, ok := tt.Find(name)
		if !ok {
			continue
		}
		if r.X >= card.X && r.Y >= card.Y &&
			r.X+r.W <= card.X+card.W && r.Y+r.H <= card.Y+card.H {
			return true
		}
	}
	return false
}

// TestApprovalCardTakesFocusOnAppear：卡出现时焦点进卡，不预选任何决定——焦点
// 落在卡容器上，三个决定都不是焦点。
func TestApprovalCardTakesFocusOnAppear(t *testing.T) {
	app, tt := newTester(t, StateWaiting)
	if _, ok := app.currentApproval(); !ok {
		t.Skip("走查态没有审批卡")
	}
	if !approvalFocusOnCard(tt) {
		t.Fatalf("审批卡出现后焦点不在卡容器上：%v", tt.Texts())
	}
	for _, label := range []string{approvalCopyReject, approvalCopyAllowOnce, approvalCopyAlwaysCommand} {
		if focusOn(tt, label) {
			t.Fatalf("卡出现就预选了决定 %q", label)
		}
	}
}

// TestApprovalCardTabCyclesInsideCard：Tab 只在卡内走——每次 Tab 之后焦点都还在
// 卡里，三个决定按「拒绝 → 仅这次 → 长期允许」的顺序各到一次，走完一圈回到卡
// 容器再走下一圈，不逃到左栏、对话流或输入区。
func TestApprovalCardTabCyclesInsideCard(t *testing.T) {
	_, tt := newTester(t, StateWaiting)
	if !approvalFocusOnCard(tt) {
		t.Fatalf("审批卡出现后焦点不在卡容器上：%v", tt.Texts())
	}
	order := []string{approvalCopyReject, approvalCopyAllowOnce, approvalCopyAlwaysCommand}
	seen := make([]bool, len(order))
	last, laps := -1, 0
	for i := 0; i < 12; i++ {
		tt.Key(0, gui.KeyTab)
		if !approvalFocusInside(tt) {
			t.Fatalf("第 %d 次 Tab 之后焦点逃到卡外", i+1)
		}
		for _, outside := range []string{"左栏 · 会话列表", "对话流", "新建会话", "收起侧边栏 ⌘B"} {
			if focusOn(tt, outside) {
				t.Fatalf("第 %d 次 Tab 之后焦点落在卡外的 %q", i+1, outside)
			}
		}
		if approvalFocusOnCard(tt) {
			// 回到卡容器：一圈走完，下一圈从新算起。
			laps++
			last = -1
			if laps > 1 {
				break
			}
			continue
		}
		at := -1
		for j, label := range order {
			if focusOn(tt, label) {
				at = j
			}
		}
		if at < 0 {
			continue // 环上还有卡内的其它控件（复制、展开），它们也在卡里
		}
		if at <= last {
			t.Fatalf("第 %d 次 Tab 之后决定顺序倒退：%q 之前已经到过", i+1, order[at])
		}
		last, seen[at] = at, true
	}
	for j, label := range order {
		if !seen[j] {
			t.Fatalf("Tab 没有走到决定 %q", label)
		}
	}
	if laps < 2 {
		t.Fatal("Tab 走完一圈没有回到卡容器：焦点环不闭合")
	}
}

// TestApprovalCardEscDoesNotAnswerOrMoveFocus：Esc 不答复、不关卡片，焦点留在
// 卡内原处。
func TestApprovalCardEscDoesNotAnswerOrMoveFocus(t *testing.T) {
	app, tt := newTester(t, StateWaiting)
	if !approvalFocusOnCard(tt) {
		t.Fatalf("审批卡出现后焦点不在卡容器上：%v", tt.Texts())
	}
	// 先走到第一个决定，Esc 之后焦点应当还在它上面。
	tt.Key(0, gui.KeyTab)
	for i := 0; i < 4 && !focusOn(tt, approvalCopyReject); i++ {
		tt.Key(0, gui.KeyTab)
	}
	if !focusOn(tt, approvalCopyReject) {
		t.Fatalf("Tab 没有把焦点带到第一个决定：%v", tt.Texts())
	}
	before := app.approvalDecision
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.approvalDecision != before {
		t.Fatalf("Esc 改动了审批答复：%q", app.approvalDecision)
	}
	if !tt.HasText(approvalCopyReject) {
		t.Fatalf("Esc 之后审批卡消失了：%v", tt.Texts())
	}
	if !focusOn(tt, approvalCopyReject) {
		t.Fatal("Esc 之后焦点离开了卡内原来的位置")
	}
}

// TestApprovalCardShiftTabCyclesBackwards：⇧Tab 从卡容器往回走，仍落在卡内。
func TestApprovalCardShiftTabCyclesBackwards(t *testing.T) {
	_, tt := newTester(t, StateWaiting)
	if !approvalFocusOnCard(tt) {
		t.Fatalf("审批卡出现后焦点不在卡容器上：%v", tt.Texts())
	}
	tt.Key(gui.Shift, gui.KeyTab)
	if !approvalFocusInside(tt) {
		t.Fatal("⇧Tab 从卡容器出走时逃到了卡外")
	}
	if approvalFocusOnCard(tt) {
		t.Fatal("⇧Tab 之后焦点还停在卡容器上：环没有往回走")
	}
}
