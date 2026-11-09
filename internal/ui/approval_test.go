package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// approvalCall is one decision the shell handed to its caller.
type approvalCall struct {
	branchID  string
	requestID string
	decision  ApprovalDecision
}

// approvalFixture is the shell's placeholder state with one pending approval
// that carries the App Server's request identity. A pending approval takes the
// composer's slot, so the row count matters: the panel is not a timeline row.
func approvalFixture() *State {
	s := fixture()
	s.Messages = append(s.Messages, Message{
		ID:       "approval:req-1",
		Author:   AuthorApproval,
		Content:  "exec — deploy to production",
		Approval: &Approval{BranchID: "main", RequestID: "req-1", Call: "exec", Reason: "deploy to production"},
	})
	return s
}

// TestApprovalPanelThroughShellSendsOneDecision checks the wired panel: the
// one-off decisions are real, available buttons, each click sends exactly one
// decision with the request's branch and id, and the request is not resolved
// locally. The always-allow entry stays a visible WIP entry.
func TestApprovalPanelThroughShellSendsOneDecision(t *testing.T) {
	s := approvalFixture()
	var got []approvalCall
	s.OnApproval = func(branchID, requestID string, decision ApprovalDecision) {
		got = append(got, approvalCall{branchID: branchID, requestID: requestID, decision: decision})
	}
	tt := ui.NewTester(s.View, 900, 600)

	for _, label := range []string{approvalLabelReject, approvalLabelAllow} {
		info, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok {
			t.Fatalf("%q is not exposed as a button, texts %q", label, tt.Texts())
		}
		if info.Disabled {
			t.Errorf("%q is disabled on a wired approval panel", label)
		}
		if !info.Focusable {
			t.Errorf("%q is not reachable on a wired approval panel", label)
		}
		if strings.Contains(info.Description, reasonApproval) {
			t.Errorf("%q still describes the WIP gap: %q", label, info.Description)
		}
	}
	grant, ok := tt.AccessByRole(ui.RoleButton, approvalLabelAllowAll)
	if !ok {
		t.Fatalf("%q is not exposed as a button, texts %q", approvalLabelAllowAll, tt.Texts())
	}
	if !grant.Disabled {
		t.Errorf("%q is available without session.grants.add", approvalLabelAllowAll)
	}

	if err := tt.Click(approvalLabelAllow); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].decision != ApprovalAllow {
		t.Fatalf("allow sent %+v, want one allow", got)
	}
	if got[0].branchID != "main" || got[0].requestID != "req-1" {
		t.Fatalf("allow identity = %+v, want main/req-1", got[0])
	}
	if err := tt.Click(approvalLabelReject); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].decision != ApprovalDeny {
		t.Fatalf("reject sent %+v, want a second deny", got)
	}
	if len(s.Messages) != 3 {
		t.Fatalf("%d rows after decisions, want the request kept until a projection frame removes it", len(s.Messages))
	}
}

// TestApprovalPanelWithoutCallbackStaysWIP checks that a pending request with no
// wired callback keeps every decision visible, dimmed and explained, and that a
// click reports nothing.
func TestApprovalPanelWithoutCallbackStaysWIP(t *testing.T) {
	s := approvalFixture()
	tt := ui.NewTester(s.View, 900, 600)

	if !tt.HasText("Approval required") {
		t.Fatalf("approval panel is not drawn, texts %q", tt.Texts())
	}
	wantReasons := map[string]string{
		approvalLabelReject:   reasonApproval,
		approvalLabelAllow:    reasonApproval,
		approvalLabelAllowAll: reasonApprovalGrant,
	}
	for label, reason := range wantReasons {
		info, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok {
			t.Fatalf("%q is not exposed as a button, texts %q", label, tt.Texts())
		}
		if !info.Disabled {
			t.Errorf("%q is available without an approval callback", label)
		}
		if !strings.Contains(info.Description, reason) {
			t.Errorf("%q description %q does not carry %q", label, info.Description, reason)
		}
		if label != approvalLabelAllowAll && !strings.Contains(info.Description, "approval.respond") {
			t.Errorf("%q description %q does not name approval.respond", label, info.Description)
		}
		if !strings.Contains(info.Description, wipBadge) {
			t.Errorf("%q description %q does not include the visible %q badge", label, info.Description, wipBadge)
		}
		if err := tt.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
	}
	if len(s.Messages) != 3 {
		t.Fatalf("%d rows after unavailable decisions, want 3", len(s.Messages))
	}
}

// TestApprovalWithoutRequestIDTakesNoSlot checks that a request without the App
// Server's identity neither takes the composer's slot nor offers a decision
// nobody can address.
func TestApprovalWithoutRequestIDTakesNoSlot(t *testing.T) {
	s := fixture()
	s.Messages = append(s.Messages, Message{ID: "m3", Author: AuthorApproval, Content: "Run it?"})
	calls := 0
	s.OnApproval = func(string, string, ApprovalDecision) { calls++ }
	tt := ui.NewTester(s.View, 900, 600)

	if tt.HasText("Approval required") {
		t.Errorf("an unaddressable request took the composer slot, texts %q", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, approvalLabelReject); ok {
		t.Error("an unaddressable request still offers a reject decision")
	}
	if calls != 0 {
		t.Fatalf("an unaddressable request sent %d decisions", calls)
	}
}
