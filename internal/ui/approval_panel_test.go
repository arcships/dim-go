package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// approvalPanelTester renders only the approval panel, the way the shell does
// once its composer slot has a pending request.
func approvalPanelTester(v ApprovalView, a Actions) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		renderApprovalPanel(c, newTokens(c.Theme()), v, a)
	}, 900, 600)
}

func approvalPanelFixture() ApprovalView {
	return ApprovalView{
		Text: "Run this command?",
		Pending: []Approval{{
			BranchID:  "main",
			RequestID: "req-1",
			Call:      "exec",
			Reason:    "deploy to production",
			Source:    "shell",
		}},
	}
}

// approvalPanelRecorder returns Actions that record every decision.
func approvalPanelRecorder(got *[]approvalCall) Actions {
	return Actions{OnApproval: func(branchID, requestID string, decision ApprovalDecision) {
		*got = append(*got, approvalCall{branchID: branchID, requestID: requestID, decision: decision})
	}}
}

// approvalPanelButton finds one decision button and fails the test when it is
// not exposed with that accessible name.
func approvalPanelButton(t *testing.T, tt *ui.Tester, label string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatalf("%q is not exposed as a button, texts %q", label, tt.Texts())
	}
	return info
}

// TestApprovalPanelRendersPendingRequest checks the panel's visible facts: the
// call name, the reason and the source of the first pending request, and the
// reference's three decisions in order. The one-off decisions are real; the
// always-allow entry stays WIP until session.grants.add exists.
func TestApprovalPanelRendersPendingRequest(t *testing.T) {
	var got []approvalCall
	tt := approvalPanelTester(approvalPanelFixture(), approvalPanelRecorder(&got))

	for _, want := range []string{"Approval required", "Run this command?", "exec", "deploy to production", "Source: shell"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q, texts %q", want, tt.Texts())
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Approval request"); !ok {
		t.Errorf("the panel is not exposed as a named group, texts %q", tt.Texts())
	}

	for _, label := range []string{approvalLabelReject, approvalLabelAllow} {
		info := approvalPanelButton(t, tt, label)
		if info.Disabled {
			t.Errorf("%q is disabled on a wired panel", label)
		}
		if !info.Focusable {
			t.Errorf("%q is not reachable on a wired panel", label)
		}
		if strings.Contains(info.Description, reasonApproval) {
			t.Errorf("%q still describes the WIP gap: %q", label, info.Description)
		}
	}

	grant := approvalPanelButton(t, tt, approvalLabelAllowAll)
	if !grant.Disabled {
		t.Errorf("%q is available, but session.grants.add is not implemented", approvalLabelAllowAll)
	}
	if !strings.Contains(grant.Description, "session.grants.add") {
		t.Errorf("%q description %q does not name session.grants.add", approvalLabelAllowAll, grant.Description)
	}

	if len(got) != 0 {
		t.Fatalf("rendering sent %#v, want no decision before a click", got)
	}
}

// TestApprovalPanelDrawsNothingWithoutPending checks the panel's empty state:
// with no pending request the composer slot stays free, even when the view
// still carries the projection's last text.
func TestApprovalPanelDrawsNothingWithoutPending(t *testing.T) {
	calls := 0
	tt := approvalPanelTester(ApprovalView{Text: "no longer pending"}, Actions{
		OnApproval: func(string, string, ApprovalDecision) { calls++ },
	})

	if texts := tt.Texts(); len(texts) != 0 {
		t.Fatalf("the empty panel drew %q, want nothing", texts)
	}
	for _, role := range []ui.Role{ui.RoleButton, ui.RoleGroup} {
		if info, ok := tt.AccessByRole(role, ""); ok {
			t.Errorf("the empty panel exposed %+v", info)
		}
	}
	if calls != 0 {
		t.Fatalf("the empty panel sent %d decisions", calls)
	}
}

// TestApprovalPanelStaysWIPWithoutCallback checks that pending decisions stay
// visible, dimmed, unreachable as actions and explained when the window has no
// OnApproval callback: a click cannot report a decision nobody sent.
func TestApprovalPanelStaysWIPWithoutCallback(t *testing.T) {
	tt := approvalPanelTester(approvalPanelFixture(), Actions{})

	wantReasons := map[string]string{
		approvalLabelReject:   reasonApproval,
		approvalLabelAllow:    reasonApproval,
		approvalLabelAllowAll: reasonApprovalGrant,
	}
	for label, reason := range wantReasons {
		info := approvalPanelButton(t, tt, label)
		if !info.Disabled {
			t.Errorf("%q is available without an approval callback", label)
		}
		if !info.Focusable {
			t.Errorf("%q is unreachable, so its reason is never read", label)
		}
		if !strings.Contains(info.Description, reason) {
			t.Errorf("%q description %q does not carry the real reason %q", label, info.Description, reason)
		}
		if !strings.Contains(info.Description, wipBadge) {
			t.Errorf("%q description %q does not include the visible %q badge", label, info.Description, wipBadge)
		}
	}
	for _, label := range []string{approvalLabelReject, approvalLabelAllow} {
		info := approvalPanelButton(t, tt, label)
		if !strings.Contains(info.Description, "approval.respond") {
			t.Errorf("%q description %q does not name approval.respond", label, info.Description)
		}
	}
	for _, label := range []string{approvalLabelReject, approvalLabelAllow, approvalLabelAllowAll} {
		if err := tt.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
	}
	if !tt.HasText("deploy to production") {
		t.Fatalf("the pending request left the view after unavailable clicks, texts %q", tt.Texts())
	}
}

// TestApprovalPanelWithoutRequestIDStaysWIP checks that a callback cannot turn
// a request without the App Server's identity into an addressable decision.
func TestApprovalPanelWithoutRequestIDStaysWIP(t *testing.T) {
	calls := 0
	v := approvalPanelFixture()
	v.Pending[0].RequestID = ""
	tt := approvalPanelTester(v, Actions{
		OnApproval: func(string, string, ApprovalDecision) { calls++ },
	})

	for _, label := range []string{approvalLabelReject, approvalLabelAllow, approvalLabelAllowAll} {
		info := approvalPanelButton(t, tt, label)
		if !info.Disabled {
			t.Errorf("%q is available without a request identity", label)
		}
		if label != approvalLabelAllowAll && !strings.Contains(info.Description, "approval.respond") {
			t.Errorf("%q description %q does not name approval.respond", label, info.Description)
		}
		if err := tt.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
	}
	if calls != 0 {
		t.Fatalf("a request without an identity sent %d decisions", calls)
	}
}

// TestApprovalPanelSendsOneDecision checks that one click sends exactly one
// decision, with the correct value for each real decision, and that the panel
// keeps its request and shows no success: only a later projection frame may
// settle it, and a failure belongs to the shell's ActionError, not to this
// view. The always-allow WIP entry sends nothing.
func TestApprovalPanelSendsOneDecision(t *testing.T) {
	var got []approvalCall
	tt := approvalPanelTester(approvalPanelFixture(), approvalPanelRecorder(&got))

	if err := tt.Click(approvalLabelAllow); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalAllow}) {
		t.Fatalf("%q sent %#v, want one allow for main/req-1", approvalLabelAllow, got)
	}

	if err := tt.Click(approvalLabelReject); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalDeny}) {
		t.Fatalf("%q sent %#v, want one deny after the allow", approvalLabelReject, got)
	}

	if err := tt.Click(approvalLabelAllowAll); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%q sent %#v, want nothing until session.grants.add exists", approvalLabelAllowAll, got)
	}

	for _, label := range []string{approvalLabelReject, approvalLabelAllow, approvalLabelAllowAll} {
		if !tt.HasText(label) {
			t.Fatalf("the panel stopped showing %q after a decision, texts %q", label, tt.Texts())
		}
	}
	if !tt.HasText("deploy to production") {
		t.Fatalf("the panel resolved the request locally, texts %q", tt.Texts())
	}
	for _, success := range []string{"Approved", "Denied", "Resolved", "Success"} {
		if tt.HasText(success) {
			t.Errorf("the panel claims %q without a projection frame, texts %q", success, tt.Texts())
		}
	}
}

// TestApprovalPanelKeyboardReachability checks that Tab reaches the decisions
// in the reference's order and Enter sends the focused one exactly once. The
// always-allow WIP entry is reachable so its reason is read, but Enter on it
// sends nothing.
func TestApprovalPanelKeyboardReachability(t *testing.T) {
	var got []approvalCall
	tt := approvalPanelTester(approvalPanelFixture(), approvalPanelRecorder(&got))

	tt.Key(0, ui.KeyTab)
	if !tt.Focused(approvalLabelReject) {
		t.Fatalf("Tab does not reach %q first, texts %q", approvalLabelReject, tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if len(got) != 1 || got[0] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalDeny}) {
		t.Fatalf("Enter on %q sent %#v, want one deny", approvalLabelReject, got)
	}

	tt.Key(0, ui.KeyTab)
	if !tt.Focused(approvalLabelAllow) {
		t.Fatalf("Tab does not reach %q, texts %q", approvalLabelAllow, tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if len(got) != 2 || got[1] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalAllow}) {
		t.Fatalf("Enter on %q sent %#v, want one allow", approvalLabelAllow, got)
	}

	tt.Key(0, ui.KeyTab)
	if !tt.Focused(approvalLabelAllowAll) {
		t.Fatalf("Tab does not reach the WIP entry %q, texts %q", approvalLabelAllowAll, tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if len(got) != 2 {
		t.Fatalf("Enter on %q sent %#v, want nothing", approvalLabelAllowAll, got)
	}
}

// TestApprovalPanelKeepsEveryPendingRequest checks that a projection with more
// than one pending request is not reduced to the first: every request keeps its
// call, reason and source, and the first request's decisions come first.
func TestApprovalPanelKeepsEveryPendingRequest(t *testing.T) {
	var got []approvalCall
	v := ApprovalView{Pending: []Approval{
		{BranchID: "main", RequestID: "req-1", Call: "exec", Reason: "first reason", Source: "shell"},
		{BranchID: "main", RequestID: "req-2", Call: "write", Reason: "second reason", Source: "mcp"},
	}}
	v.InputDisplay = "rm -rf ./build"
	tt := approvalPanelTester(v, approvalPanelRecorder(&got))

	for _, want := range []string{"exec", "first reason", "Source: shell", "write", "second reason", "Source: mcp"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q, texts %q", want, tt.Texts())
		}
	}
	// The view carries one input payload, the primary request's: it is drawn
	// once and is never copied onto the other requests.
	if count := textCount(tt.Texts(), "rm -rf ./build"); count != 1 {
		t.Errorf("the primary input is drawn %d times, want once, texts %q", count, tt.Texts())
	}
	if err := tt.Click(approvalLabelAllow); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalAllow}) {
		t.Fatalf("the first %q sent %#v, want one allow for main/req-1", approvalLabelAllow, got)
	}
}

// TestApprovalPanelRendersReadOnlyInput checks that a request's input is drawn
// as one named, read-only block: its text is visible, the block is exposed as a
// group, and no text field is created, so it cannot be edited or mistaken for
// the composer.
func TestApprovalPanelRendersReadOnlyInput(t *testing.T) {
	v := approvalPanelFixture()
	v.InputDisplay = `{"cmd":"rm -rf ./build","cwd":"/repo"}`
	tt := approvalPanelTester(v, Actions{
		OnApproval: func(string, string, ApprovalDecision) {},
	})

	if !tt.HasText(`{"cmd":"rm -rf ./build","cwd":"/repo"}`) {
		t.Fatalf("the input block is not drawn, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Approval input"); !ok {
		t.Errorf("the input block is not exposed as a named group, texts %q", tt.Texts())
	}
	if info, ok := tt.AccessByRole(ui.RoleTextField, ""); ok {
		t.Errorf("the input block is exposed as an editable field: %+v", info)
	}
	if !tt.HasText("Approval required") || !tt.HasText("deploy to production") {
		t.Errorf("the input block replaced the request's own facts, texts %q", tt.Texts())
	}
}

// TestApprovalPanelWithoutInputDrawsNoBlock checks that a request with no input
// payload draws no block and no placeholder: the card keeps its other parts and
// its height does not reserve a slot for an input nobody sent.
func TestApprovalPanelWithoutInputDrawsNoBlock(t *testing.T) {
	v := approvalPanelFixture()
	tt := approvalPanelTester(v, Actions{
		OnApproval: func(string, string, ApprovalDecision) {},
	})

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Approval input"); ok {
		t.Errorf("a request without input still drew the input block, texts %q", tt.Texts())
	}
	for _, placeholder := range []string{"(none)", "No input", "Input"} {
		if tt.HasText(placeholder) {
			t.Errorf("a request without input drew the placeholder %q, texts %q", placeholder, tt.Texts())
		}
	}
	if !tt.HasText("deploy to production") {
		t.Errorf("the request's own facts are missing, texts %q", tt.Texts())
	}
}

// TestApprovalPanelTruncatesLongInput checks the input block's display bound:
// an oversized input is cut to approvalInputMaxRunes runes and ends in an
// ellipsis, so the block never grows with whatever the App Server sent, and the
// text beyond the cap is not claimed as shown.
func TestApprovalPanelTruncatesLongInput(t *testing.T) {
	v := approvalPanelFixture()
	v.InputDisplay = strings.Repeat("a", approvalInputMaxRunes+500)
	tt := approvalPanelTester(v, Actions{
		OnApproval: func(string, string, ApprovalDecision) {},
	})

	want := strings.Repeat("a", approvalInputMaxRunes) + "…"
	if !tt.HasText(want) {
		t.Fatalf("the input block is not the capped text with an ellipsis, texts %q", tt.Texts())
	}
	for _, text := range tt.Texts() {
		if runes := []rune(text); len(runes) > approvalInputMaxRunes+1 && strings.HasPrefix(text, "aaa") {
			t.Errorf("the input block drew %d runes, want at most %d", len(runes), approvalInputMaxRunes)
		}
	}
}

// TestApprovalPanelBlankInputDrawsNoBlock checks that whitespace-only input is
// treated as no input: the view never draws an empty frame for it.
func TestApprovalPanelBlankInputDrawsNoBlock(t *testing.T) {
	v := approvalPanelFixture()
	v.InputDisplay = "  \n\t  "
	tt := approvalPanelTester(v, Actions{
		OnApproval: func(string, string, ApprovalDecision) {},
	})

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Approval input"); ok {
		t.Errorf("blank input still drew the input block, texts %q", tt.Texts())
	}
}

// TestApprovalPanelDrawsTheReferenceDecisions checks that the panel offers the
// reference's three decisions whatever option the view names: the variant no
// longer selects a pair of labels, because the wire carries no option list and
// the panel keeps the decisions it can send, with the grant entry marked WIP.
func TestApprovalPanelDrawsTheReferenceDecisions(t *testing.T) {
	for _, variant := range []ApprovalVariant{
		ApprovalVariantUnspecified,
		ApprovalVariantAllowOnce,
		ApprovalVariantAllowAll,
		ApprovalVariantRejectOnce,
		ApprovalVariant("question"),
	} {
		t.Run(string(variant), func(t *testing.T) {
			v := approvalPanelFixture()
			v.Variant = variant
			tt := approvalPanelTester(v, Actions{
				OnApproval: func(string, string, ApprovalDecision) {},
			})

			for _, label := range []string{approvalLabelReject, approvalLabelAllow, approvalLabelAllowAll} {
				approvalPanelButton(t, tt, label)
			}
			for _, old := range []string{"Approve", "Deny", "Deny once"} {
				if _, ok := tt.AccessByRole(ui.RoleButton, old); ok {
					t.Errorf("variant %q still draws the old label %q", variant, old)
				}
			}
		})
	}
}

// TestApprovalPanelAllowAllExplainsGrants checks that the always-allow entry
// says why it is unavailable: the missing method and the visible WIP badge, as
// every other unwired entry does.
func TestApprovalPanelAllowAllExplainsGrants(t *testing.T) {
	tt := approvalPanelTester(approvalPanelFixture(), Actions{
		OnApproval: func(string, string, ApprovalDecision) {},
	})

	info := approvalPanelButton(t, tt, approvalLabelAllowAll)
	if !info.Disabled {
		t.Error("Allow all is available without session.grants.add")
	}
	if !info.Focusable {
		t.Error("Allow all is unreachable, so its reason is never read")
	}
	for _, want := range []string{"session.grants.add", wipBadge} {
		if !strings.Contains(info.Description, want) {
			t.Errorf("Allow all description %q does not carry %q", info.Description, want)
		}
	}
}

// TestApprovalPanelKeepsTheComposerSlotMeasure checks the panel's width in the
// composer's slot: at the reference's 520 DIP conversation measure the card
// keeps that cap and centres itself, and every decision stays inside the card
// on one line. The row's measures, not the text's, are checked, so a long
// reason cannot hide an overflowing control.
func TestApprovalPanelKeepsTheComposerSlotMeasure(t *testing.T) {
	tt := approvalPanelTester(approvalPanelFixture(), Actions{
		OnApproval: func(string, string, ApprovalDecision) {},
	})
	tt.SetSize(520, 600)

	card, ok := tt.AccessByRole(ui.RoleGroup, "Approval request")
	if !ok {
		t.Fatalf("the panel is not exposed as a named group, texts %q", tt.Texts())
	}
	if card.Bounds.W > approvalPanelWidth {
		t.Errorf("the card is %.1f DIPs wide, want at most %d", card.Bounds.W, approvalPanelWidth)
	}
	if card.Bounds.X < 0 || card.Bounds.X+card.Bounds.W > 520 {
		t.Errorf("the card runs %.1f..%.1f in a 520 DIP window", card.Bounds.X, card.Bounds.X+card.Bounds.W)
	}

	var lines = map[float32]bool{}
	for _, label := range []string{approvalLabelReject, approvalLabelAllow, approvalLabelAllowAll} {
		button := approvalPanelButton(t, tt, label)
		if button.Bounds.X < card.Bounds.X || button.Bounds.X+button.Bounds.W > card.Bounds.X+card.Bounds.W {
			t.Errorf("%q runs %.1f..%.1f outside the card's %.1f..%.1f",
				label, button.Bounds.X, button.Bounds.X+button.Bounds.W, card.Bounds.X, card.Bounds.X+card.Bounds.W)
		}
		lines[button.Bounds.Y] = true
	}
	if len(lines) != 1 {
		t.Errorf("the decisions take %d lines at the 520 DIP measure, want one", len(lines))
	}
}

// TestApprovalPanelWrapsInANarrowWindow checks the narrow-window path: the
// decisions keep their order but wrap onto more than one line instead of
// running past the card or the window, and the card itself stays inside the
// window.
func TestApprovalPanelWrapsInANarrowWindow(t *testing.T) {
	tt := approvalPanelTester(approvalPanelFixture(), Actions{
		OnApproval: func(string, string, ApprovalDecision) {},
	})
	tt.SetSize(320, 600)

	card, ok := tt.AccessByRole(ui.RoleGroup, "Approval request")
	if !ok {
		t.Fatalf("the panel is not exposed as a named group, texts %q", tt.Texts())
	}
	if card.Bounds.X < 0 || card.Bounds.X+card.Bounds.W > 320 {
		t.Errorf("the card runs %.1f..%.1f in a 320 DIP window", card.Bounds.X, card.Bounds.X+card.Bounds.W)
	}

	lines := map[float32]bool{}
	var previous float32 = -1
	for _, label := range []string{approvalLabelReject, approvalLabelAllow, approvalLabelAllowAll} {
		button := approvalPanelButton(t, tt, label)
		if button.Bounds.X < 0 || button.Bounds.X+button.Bounds.W > 320 {
			t.Errorf("%q runs %.1f..%.1f outside a 320 DIP window",
				label, button.Bounds.X, button.Bounds.X+button.Bounds.W)
		}
		if button.Bounds.X < card.Bounds.X || button.Bounds.X+button.Bounds.W > card.Bounds.X+card.Bounds.W {
			t.Errorf("%q runs %.1f..%.1f outside the card's %.1f..%.1f",
				label, button.Bounds.X, button.Bounds.X+button.Bounds.W, card.Bounds.X, card.Bounds.X+card.Bounds.W)
		}
		if previous >= 0 && button.Bounds.Y < previous {
			t.Errorf("%q moved back up to y=%.1f after %.1f", label, button.Bounds.Y, previous)
		}
		previous = button.Bounds.Y
		lines[button.Bounds.Y] = true
	}
	if len(lines) < 2 {
		t.Errorf("the decisions take %d line in a 320 DIP window, want them wrapped", len(lines))
	}
}

// TestApprovalPanelLongInputStaysInsideTheCard checks that the read-only input
// block wraps its text in a narrow window: a long command cannot widen the
// block past the card, and the text it draws stays inside the block's box.
func TestApprovalPanelLongInputStaysInsideTheCard(t *testing.T) {
	v := approvalPanelFixture()
	v.InputDisplay = strings.Repeat("x", 400) + " --flag"
	tt := approvalPanelTester(v, Actions{
		OnApproval: func(string, string, ApprovalDecision) {},
	})
	tt.SetSize(320, 600)

	card, ok := tt.AccessByRole(ui.RoleGroup, "Approval request")
	if !ok {
		t.Fatalf("the panel is not exposed as a named group, texts %q", tt.Texts())
	}
	input, ok := tt.AccessByRole(ui.RoleGroup, "Approval input")
	if !ok {
		t.Fatalf("the input block is not exposed as a named group, texts %q", tt.Texts())
	}
	if input.Bounds.X < card.Bounds.X || input.Bounds.X+input.Bounds.W > card.Bounds.X+card.Bounds.W {
		t.Errorf("the input block runs %.1f..%.1f outside the card's %.1f..%.1f",
			input.Bounds.X, input.Bounds.X+input.Bounds.W, card.Bounds.X, card.Bounds.X+card.Bounds.W)
	}
	text, ok := tt.Find(v.InputDisplay)
	if !ok {
		t.Fatalf("the input text is not drawn, texts %q", tt.Texts())
	}
	if text.X < input.Bounds.X || text.X+text.W > input.Bounds.X+input.Bounds.W {
		t.Errorf("the input text runs %.1f..%.1f outside the input block's %.1f..%.1f",
			text.X, text.X+text.W, input.Bounds.X, input.Bounds.X+input.Bounds.W)
	}
}

// TestApprovalPanelRejectUsesDanger checks the reject decision's color: its
// outline and label are painted in the shell's danger token, while the allow
// decision keeps the ordinary face, so the refusal reads as the destructive
// choice without a new token.
func TestApprovalPanelRejectUsesDanger(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(map[bool]string{false: "light", true: "dark"}[dark], func(t *testing.T) {
			theme := ui.LightTheme()
			if dark {
				theme = ui.DarkTheme()
			}
			k := newTokens(theme)
			tt := ui.NewTester(func(c *ui.Context) {
				renderApprovalPanel(c, newTokens(c.Theme()), approvalPanelFixture(), Actions{
					OnApproval: func(string, string, ApprovalDecision) {},
				})
			}, 520, 600)
			tt.SetDark(dark)

			reject := approvalPanelButton(t, tt, approvalLabelReject)
			if _, pixels := colorBox(tt, reject.Bounds, k.danger); pixels == 0 {
				t.Errorf("%q paints no %v pixels in its %.0fx%.0f box",
					approvalLabelReject, k.danger, reject.Bounds.W, reject.Bounds.H)
			}
			allow := approvalPanelButton(t, tt, approvalLabelAllow)
			if _, pixels := colorBox(tt, allow.Bounds, k.danger); pixels != 0 {
				t.Errorf("%q paints %d danger pixels; only reject should", approvalLabelAllow, pixels)
			}
		})
	}
}

// textCount counts the drawn text nodes equal to s.
func textCount(texts []string, s string) int {
	n := 0
	for _, text := range texts {
		if text == s {
			n++
		}
	}
	return n
}
