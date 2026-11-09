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
	if err := tt.Click("Approve"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (approvalCall{branchID: "main", requestID: "req-1", decision: ApprovalAllow}) {
		t.Fatalf("the first Approve sent %#v, want one allow for main/req-1", got)
	}
}
