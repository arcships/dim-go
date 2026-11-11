package interaction_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/interaction"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestRealSubmitDuringApproval 验证审批待答期间提交 queue/steer 的真实结果。
//
//	DIM_REAL_MODEL_FIXTURE=<fixture> DIM_REAL_MODEL_SERVER=<server> \
//	  go test -run TestRealSubmitDuringApproval -v -count=1 -timeout 12m ./internal/interaction/
//
// 只记录状态、序号、模式、错误码与队列条数，不打印输入正文。
func TestRealSubmitDuringApproval(t *testing.T) {
	fixture := realFixtureHome(t)
	server := realServerPath(t)
	home := prepareApprovalHome(t, fixture)
	base := startRealApp(t, home, server)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	t.Cleanup(func() { _ = base.Close(context.Background()) })

	t.Run("queue", func(t *testing.T) {
		sessionID := createRealSession(t, ctx, base)
		if err := setWorkspaceWrite(ctx, base, sessionID); err != nil {
			t.Fatalf("设置 workspace-write: %v", err)
		}
		submitRealPrompt(t, ctx, base, sessionID)
		approval := waitApproval(t, ctx, base, sessionID)
		t.Logf("queue: approval=%s pendingApprovals=%d", approval.RequestID, approval.Position+1)
		logPendingInputs(t, ctx, base, sessionID, "before-queue")

		intent := probeSubmit(t, ctx, base, sessionID, "queue-probe", write.DeliveryQueue)
		t.Logf("queue: submit status=%s code=%s", intent.Status, errorCode(intent.Reason))
		logPendingInputs(t, ctx, base, sessionID, "after-queue")

		if _, err := base.Respond(ctx, sessionID, approval.RequestID, interaction.Response{
			Decision: interaction.DecisionApprove,
		}); err != nil {
			t.Fatalf("同意审批: %v", err)
		}
		waitToolResult(t, ctx, base, sessionID, true, 0)
		waitInputsConsumed(t, ctx, base, sessionID, "queue")
		view, _ := base.ReloadLatest(ctx, sessionID)
		if view.PendingCounts.Approvals != 0 || len(view.Turns) < 2 {
			t.Fatalf("queue 消费后应无待决审批且开新轮: approvals=%d turns=%d",
				view.PendingCounts.Approvals, len(view.Turns))
		}
	})

	t.Run("steer", func(t *testing.T) {
		sessionID := createRealSession(t, ctx, base)
		if err := setWorkspaceWrite(ctx, base, sessionID); err != nil {
			t.Fatalf("设置 workspace-write: %v", err)
		}
		submitRealPrompt(t, ctx, base, sessionID)
		approval := waitApproval(t, ctx, base, sessionID)
		t.Logf("steer: approval=%s pendingApprovals=%d", approval.RequestID, approval.Position+1)
		logPendingInputs(t, ctx, base, sessionID, "before-steer")

		intent := probeSubmit(t, ctx, base, sessionID, "steer-probe", write.DeliveryInterject)
		t.Logf("steer: submit status=%s code=%s", intent.Status, errorCode(intent.Reason))
		logPendingInputs(t, ctx, base, sessionID, "after-steer")

		if _, err := base.Respond(ctx, sessionID, approval.RequestID, interaction.Response{
			Decision: interaction.DecisionApprove,
		}); err != nil {
			t.Fatalf("同意审批: %v", err)
		}
		waitToolResult(t, ctx, base, sessionID, true, 0)
		waitInputsConsumed(t, ctx, base, sessionID, "steer")
		view, _ := base.ReloadLatest(ctx, sessionID)
		if view.PendingCounts.Approvals != 0 || len(view.Turns) != 1 {
			t.Fatalf("steer 消费后应无待决审批且不开新轮: approvals=%d turns=%d",
				view.PendingCounts.Approvals, len(view.Turns))
		}
	})
}

func probeSubmit(t *testing.T, ctx context.Context, base *app.App, sessionID, text string, delivery write.Delivery) write.IntentView {
	t.Helper()
	view, err := base.Submit(ctx, app.SubmitRequest{
		SessionID: sessionID,
		Text:      text,
		Delivery:  delivery,
	})
	if err != nil {
		t.Fatalf("提交 %s: %v", delivery, err)
	}
	settled, err := base.Wait(ctx, view.ID)
	if err != nil {
		t.Fatalf("等待提交 %s: %v", delivery, err)
	}
	return settled
}

func logPendingInputs(t *testing.T, ctx context.Context, base *app.App, sessionID, stage string) {
	t.Helper()
	view, _ := base.ReloadLatest(ctx, sessionID)
	t.Logf("%s: approvals=%d queuedInputs=%d turns=%d",
		stage, view.PendingCounts.Approvals, len(view.Pending.Inputs), len(view.Turns))
	for _, input := range view.Pending.Inputs {
		t.Logf("%s: input seq=%d mode=%s applied=%t", stage, input.Sequence, input.Mode, input.Applied)
	}
}

func waitInputsConsumed(t *testing.T, ctx context.Context, base *app.App, sessionID, label string) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		view, _ := base.ReloadLatest(ctx, sessionID)
		if len(view.Pending.Inputs) == 0 {
			t.Logf("%s: consumed approvals=%d queuedInputs=0 turns=%d",
				label, view.PendingCounts.Approvals, len(view.Turns))
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("%s 等待输入消费超时: %v", label, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	view, _ := base.ReloadLatest(ctx, sessionID)
	t.Fatalf("%s 输入未被消费: queuedInputs=%d turns=%d", label, len(view.Pending.Inputs), len(view.Turns))
}

var rpcCodePattern = regexp.MustCompile(`(-?\d{5})`)

func errorCode(reason string) string {
	if match := rpcCodePattern.FindString(reason); match != "" {
		return match
	}
	return "none"
}
