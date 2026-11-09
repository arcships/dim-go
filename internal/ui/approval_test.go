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
// row that carries the App Server's request identity.
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

// TestApprovalRowSendsOneDecision checks the wired approval row: both decisions
// are real, available buttons, and each click sends exactly one decision with
// the row's branch and request identity. The row itself is not resolved
// locally; only a later session projection frame may remove it.
func TestApprovalRowSendsOneDecision(t *testing.T) {
	s := approvalFixture()
	var got []approvalCall
	s.OnApproval = func(branchID, requestID string, decision ApprovalDecision) {
		got = append(got, approvalCall{branchID: branchID, requestID: requestID, decision: decision})
	}
	tt := ui.NewTester(s.View, 900, 600)

	for _, label := range []string{"Approve", "Deny"} {
		info, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok {
			t.Fatalf("%q is not exposed as a button, texts %q", label, tt.Texts())
		}
		if info.Disabled {
			t.Errorf("%q is disabled on a wired approval row", label)
		}
		if !info.Focusable {
			t.Errorf("%q is not reachable on a wired approval row", label)
		}
		if strings.Contains(info.Description, reasonApproval) {
			t.Errorf("%q still describes the WIP gap: %q", label, info.Description)
		}
	}

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

	if len(s.Messages) != 3 || s.Messages[2].Approval == nil {
		t.Fatalf("the shell resolved the approval row locally: %#v", s.Messages)
	}
	if a, ok := tt.AccessByRole(ui.RoleAlert, ""); ok {
		t.Errorf("an accepted click raised an alert: %#v", a)
	}
}

// TestApprovalRowStaysWIPWithoutCallback checks that an approval row in a
// fixture without OnApproval keeps both decisions visible, dimmed, unreachable
// as actions and explained: a click cannot report a decision nobody sent.
func TestApprovalRowStaysWIPWithoutCallback(t *testing.T) {
	s := approvalFixture()
	tt := ui.NewTester(s.View, 900, 600)

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
	if len(s.Messages) != 3 {
		t.Fatalf("%d rows after clicking unavailable decisions, want 3", len(s.Messages))
	}
}

// TestApprovalRowWithoutRequestIDStaysWIP checks that a row without the App
// Server's request identity keeps the decisions unavailable even when the
// window has a callback: there is no approval.respond request to address.
func TestApprovalRowWithoutRequestIDStaysWIP(t *testing.T) {
	cases := []struct {
		name     string
		approval *Approval
	}{
		{"missing approval", nil},
		{"empty request id", &Approval{BranchID: "main"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := fixture()
			s.Messages = append(s.Messages, Message{ID: "m3", Author: AuthorApproval, Content: "Run it?", Approval: tc.approval})
			calls := 0
			s.OnApproval = func(string, string, ApprovalDecision) { calls++ }
			tt := ui.NewTester(s.View, 900, 600)

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
				t.Fatalf("a row without a request identity sent %d decisions", calls)
			}
		})
	}
}
