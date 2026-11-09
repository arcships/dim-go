package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// approvalPanelTester renders only the approval panel, the way the shell will
// once its composer slot calls renderApprovalPanel.
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

// TestApprovalPanelRendersPendingRequest checks the panel's visible facts: the
// call name, the reason and the source of the first pending request, and two
// real decisions. The request is not resolved locally.
func TestApprovalPanelRendersPendingRequest(t *testing.T) {
	var got []approvalCall
	a := Actions{OnApproval: func(branchID, requestID string, decision ApprovalDecision) {
		got = append(got, approvalCall{branchID: branchID, requestID: requestID, decision: decision})
	}}
	tt := approvalPanelTester(approvalPanelFixture(), a)

	for _, want := range []string{"Approval required", "Run this command?", "exec", "deploy to production", "Source: shell"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q, texts %q", want, tt.Texts())
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Approval request"); !ok {
		t.Errorf("the panel is not exposed as a named group, texts %q", tt.Texts())
	}
	for _, label := range []string{"Approve", "Deny"} {
		info, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok {
			t.Fatalf("%q is not exposed as a button, texts %q", label, tt.Texts())
		}
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

	for _, label := range []string{"Approve", "Deny"} {
		info, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok {
			t.Fatalf("%q is not exposed as a button, texts %q", label, tt.Texts())
		}
		if !info.Disabled {
			t.Errorf("%q is available without an approval callback", label)
		}
		if !info.Focusable {
			t.Errorf("%q is unreachable, so its reason is never read", label)
		}
		if !strings.Contains(info.Description, reasonApproval) {
			t.Errorf("%q description %q does not carry the real reason", label, info.Description)
		}
		if !strings.Contains(info.Description, "approval.respond") {
			t.Errorf("%q description %q does not name approval.respond", label, info.Description)
		}
		if !strings.Contains(info.Description, wipBadge) {
			t.Errorf("%q description %q does not include the visible %q badge", label, info.Description, wipBadge)
		}
	}
	if err := tt.Click("Approve"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Deny"); err != nil {
		t.Fatal(err)
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

	for _, label := range []string{"Approve", "Deny"} {
		info, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok {
			t.Fatalf("%q is not exposed as a button, texts %q", label, tt.Texts())
		}
		if !info.Disabled {
			t.Errorf("%q is available without a request identity", label)
		}
		if !strings.Contains(info.Description, "approval.respond") {
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
// decision, with the correct value for each button, and that the panel keeps
// its request and shows no success: only a later projection frame may settle
// it, and a failure belongs to the shell's ActionError, not to this view.
func TestApprovalPanelSendsOneDecision(t *testing.T) {
	var got []approvalCall
	tt := approvalPanelTester(approvalPanelFixture(), Actions{
		OnApproval: func(branchID, requestID string, decision ApprovalDecision) {
			got = append(got, approvalCall{branchID: branchID, requestID: requestID, decision: decision})
		},
	})

	if err := tt.Click("Approve"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalAllow}) {
		t.Fatalf("Approve sent %#v, want one allow for main/req-1", got)
	}

	if err := tt.Click("Deny"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalDeny}) {
		t.Fatalf("Deny sent %#v, want one deny after the allow", got)
	}

	if !tt.HasText("deploy to production") || !tt.HasText("Approve") || !tt.HasText("Deny") {
		t.Fatalf("the panel resolved the request locally, texts %q", tt.Texts())
	}
	for _, success := range []string{"Approved", "Denied", "Resolved", "Success"} {
		if tt.HasText(success) {
			t.Errorf("the panel claims %q without a projection frame, texts %q", success, tt.Texts())
		}
	}
}

// TestApprovalPanelKeyboardReachability checks that Tab reaches both decisions
// and Enter sends the focused one exactly once.
func TestApprovalPanelKeyboardReachability(t *testing.T) {
	var got []approvalCall
	tt := approvalPanelTester(approvalPanelFixture(), Actions{
		OnApproval: func(branchID, requestID string, decision ApprovalDecision) {
			got = append(got, approvalCall{branchID: branchID, requestID: requestID, decision: decision})
		},
	})

	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Approve") {
		t.Fatalf("Tab does not reach Approve, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if len(got) != 1 || got[0] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalAllow}) {
		t.Fatalf("Enter on Approve sent %#v, want one allow", got)
	}

	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Deny") {
		t.Fatalf("Tab does not reach Deny, texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if len(got) != 2 || got[1] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalDeny}) {
		t.Fatalf("Enter on Deny sent %#v, want one deny", got)
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
	tt := approvalPanelTester(v, Actions{
		OnApproval: func(branchID, requestID string, decision ApprovalDecision) {
			got = append(got, approvalCall{branchID: branchID, requestID: requestID, decision: decision})
		},
	})

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
	if err := tt.Click("Approve"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalAllow}) {
		t.Fatalf("the first Approve sent %#v, want one allow for main/req-1", got)
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

// TestApprovalPanelVariantLabels checks the documented variants' wording and
// their decisions: allow_once, allow_all and reject_once change the labels, an
// empty or unknown variant keeps Approve/Deny, and every click still sends one
// decision with the request's identity.
func TestApprovalPanelVariantLabels(t *testing.T) {
	cases := []struct {
		name         string
		variant      ApprovalVariant
		allowLabel   string
		denyLabel    string
		allowIsGrant bool
	}{
		{"unspecified", ApprovalVariantUnspecified, "Approve", "Deny", false},
		{"allow once", ApprovalVariantAllowOnce, "Allow once", "Deny", false},
		{"allow all", ApprovalVariantAllowAll, "Always allow", "Deny", true},
		{"reject once", ApprovalVariantRejectOnce, "Approve", "Deny once", false},
		{"unknown", ApprovalVariant("question"), "Approve", "Deny", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []approvalCall
			v := approvalPanelFixture()
			v.Variant = tc.variant
			tt := approvalPanelTester(v, Actions{
				OnApproval: func(branchID, requestID string, decision ApprovalDecision) {
					got = append(got, approvalCall{branchID: branchID, requestID: requestID, decision: decision})
				},
			})

			allow, ok := tt.AccessByRole(ui.RoleButton, tc.allowLabel)
			if !ok {
				t.Fatalf("%q is not exposed as a button, texts %q", tc.allowLabel, tt.Texts())
			}
			deny, ok := tt.AccessByRole(ui.RoleButton, tc.denyLabel)
			if !ok {
				t.Fatalf("%q is not exposed as a button, texts %q", tc.denyLabel, tt.Texts())
			}
			if deny.Disabled {
				t.Errorf("%q is disabled on a wired panel", tc.denyLabel)
			}
			if tc.allowIsGrant && !allow.Disabled {
				t.Errorf("%q is available, but session.grants.add is not implemented", tc.allowLabel)
			}
			if !tc.allowIsGrant && allow.Disabled {
				t.Errorf("%q is disabled on a wired panel", tc.allowLabel)
			}

			if err := tt.Click(tc.allowLabel); err != nil {
				t.Fatal(err)
			}
			want := len(got)
			if tc.allowIsGrant {
				if want != 0 {
					t.Fatalf("Always allow sent %#v, want nothing until session.grants.add exists", got)
				}
			} else if want != 1 || got[0] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalAllow}) {
				t.Fatalf("%q sent %#v, want one allow for main/req-1", tc.allowLabel, got)
			}

			if err := tt.Click(tc.denyLabel); err != nil {
				t.Fatal(err)
			}
			if len(got) != want+1 || got[want] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalDeny}) {
				t.Fatalf("%q sent %#v, want one deny after the allow", tc.denyLabel, got)
			}
		})
	}
}

// TestApprovalPanelAllowAllExplainsGrants checks that the always-allow entry
// says why it is unavailable: the missing method and the visible WIP badge, as
// every other unwired entry does.
func TestApprovalPanelAllowAllExplainsGrants(t *testing.T) {
	v := approvalPanelFixture()
	v.Variant = ApprovalVariantAllowAll
	tt := approvalPanelTester(v, Actions{
		OnApproval: func(string, string, ApprovalDecision) {},
	})

	info, ok := tt.AccessByRole(ui.RoleButton, "Always allow")
	if !ok {
		t.Fatalf("Always allow is not exposed as a button, texts %q", tt.Texts())
	}
	if !info.Disabled {
		t.Error("Always allow is available without session.grants.add")
	}
	if !info.Focusable {
		t.Error("Always allow is unreachable, so its reason is never read")
	}
	for _, want := range []string{"session.grants.add", wipBadge} {
		if !strings.Contains(info.Description, want) {
			t.Errorf("Always allow description %q does not carry %q", info.Description, want)
		}
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
