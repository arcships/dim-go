package session

import (
	"encoding/json"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

func TestApprovalResponsePendingProjection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response fakeMutation
		pending  int
	}{
		{"responded", fakeMutation{Result: json.RawMessage(`{}`)}, 0},
		{"cancelled", fakeMutation{Fault: &fakeRPCError{Code: -32602, Message: "gone"}}, 0},
		{"error", fakeMutation{Fault: &fakeRPCError{Code: -32000, Message: "refused"}}, 1},
		{"unknown", fakeMutation{Fault: &fakeRPCError{Code: -32000, Message: "aborted", Data: json.RawMessage(`{"appError":"request_aborted"}`)}}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, _, s := openTestSession(t, fakeConfig{Attach: appserver.AttachResult{Sequence: int64Ptr(1)}, History: emptyHistory(t, 1), Mutations: map[string][]fakeMutation{"approval.respond": {tc.response}}}, []string{"session.history"}, OpenOptions{})
			server.emit("approval.request", appserver.ApprovalRequest{SessionID: "s1", BranchID: "main", RequestID: "r1", Source: appserver.ApprovalSource{Kind: "tool"}, Call: appserver.ApprovalCall{Name: "exec", Input: json.RawMessage(`{}`)}})
			waitFor(t, func() bool { return len(s.PendingApprovals("main")) == 1 }, "pending approval")
			_, err := s.RespondApproval(testContext(t), appserver.ApprovalRespondParams{BranchID: "main", RequestID: "r1", Decision: appserver.DecisionAllow})
			if (err != nil) != (tc.pending == 1) {
				t.Fatalf("response error=%v", err)
			}
			if got := len(s.PendingApprovals("main")); got != tc.pending {
				t.Fatalf("pending=%d want %d", got, tc.pending)
			}
			// An actual settlement arriving later remains safe; no response path
			// fabricates this event or keeps a second resolved-request store.
			server.emit("approval.resolved", appserver.ApprovalResolved{SessionID: "s1", BranchID: "main", RequestID: "r1", Decision: appserver.DecisionAllow})
			waitFor(t, func() bool { return len(s.PendingApprovals("main")) == 0 }, "settled approval")
			if got := len(server.recorded("approval.respond")); got != 1 {
				t.Fatalf("mutation calls=%d", got)
			}
		})
	}
}
