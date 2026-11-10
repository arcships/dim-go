package projection

import (
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// pendingFixture 造四类未完成事项：两条排队输入、一个后台任务、一条协作请求、一批提问，
// 之后逐条给出结算事实。
func pendingFixture() []protocol.RawEvent {
	return []protocol.RawEvent{
		ev("e1", 1, "input", "message", `{"mode":"queue","user":{"content":[{"type":"text","text":"第一条"}]}}`),
		ev("e2", 2, "input", "message", `{"mode":"steer","user":{"content":[{"type":"text","text":"第二条"}]}}`),
		ev("e3", 3, "exec", "started", `{"task_id":"k1","description":"go build"}`),
		ev("e4", 4, "collaboration", "request", `{"call_id":"c9","target":{"kind":"agent","agent_id":"reviewer"},"prompt":"看下"}`),
		ev("e5", 5, "step", "tool-use", `{"turn_id":"t1","request_id":"r1","call":{"id":"q1","name":"ask_user","input":{"questions":[{"question":"继续吗"}]}}}`),
		ev("e6", 6, "input", "consumed", `{"input_id":"e1"}`),
		ev("e7", 7, "collaboration", "delivered", `{"op_event_id":"e4","status":"ok","agent_id":"reviewer","session_id":"child-1"}`),
		ev("e8", 8, "exec", "ended", `{"task_id":"k1","status":"completed"}`),
		ev("e9", 9, "collaboration", "result", `{"call_id":"c9","status":"completed"}`),
		ev("e10", 10, "step", "tool-completed", `{"turn_id":"t1","request_id":"r1","call_id":"q1","result":{"content":[{"type":"text","text":"好"}]}}`),
		ev("e11", 11, "input", "consumed", `{"input_id":"e2"}`),
	}
}

// seedAtCutoff 按服务端在截止点 5 读到的四张清单装载。
func seedAtCutoff(t *testing.T, folder *Folder) {
	t.Helper()
	folder.SeedPendingInputs(mainBranch, 5, []PendingInput{
		{InputID: "e1", Sequence: 1, Mode: "queue", Text: "第一条"},
		{InputID: "e2", Sequence: 2, Mode: "steer", Text: "第二条"},
	})
	folder.SeedPendingTasks(mainBranch, 5, []PendingTask{
		{TaskID: "k1", EventID: "e3", Sequence: 3, Description: "go build", StartedAt: fixtureBaseAt + 3},
	})
	folder.SeedPendingCollaborations(mainBranch, 5, []PendingCollaboration{
		{EventID: "e4", Sequence: 4, CallID: "c9", AgentID: "reviewer", TargetKind: "agent", Prompt: "看下", StartedAt: fixtureBaseAt + 4},
	})
	folder.SeedPendingQuestions(mainBranch, 5, []PendingQuestion{
		{RequestID: "r1", CallID: "q1", EventID: "e5", Sequence: 5, ToolName: "ask_user"},
	})
}

// TestPendingInventoriesUseCutoffAndSettleByFacts 验证四类未完成事项按固定截止点装载，
// 之后按事件结算到零。
func TestPendingInventoriesUseCutoffAndSettleByFacts(t *testing.T) {
	fixture := pendingFixture()

	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(5, fixture[:5]...))
	seedAtCutoff(t, folder)

	view := folder.View()
	if len(view.Pending.Inputs) != 2 {
		t.Fatalf("排队输入应为 2，实际 %+v", view.Pending.Inputs)
	}
	if len(view.Pending.Tasks) != 1 || len(view.Pending.Collaborations) != 1 || len(view.Pending.Questions) != 1 {
		t.Fatalf("四类清单不齐: %+v", view.Pending)
	}
	question := view.Pending.Questions[0]
	if question.RequestID != "r1" || question.CallID != "q1" || question.ToolName != "ask_user" {
		t.Fatalf("提问清单项不对: %+v", question)
	}
	if view.Pending.Total() != 5 {
		t.Fatalf("截止点清单总条数应为 5，实际 %d", view.Pending.Total())
	}

	// 逐条结算：输入、任务、协作请求、提问依次归零。
	mustNotice(t, folder, mainBranch, fixture[5]) // e6 结算 e1
	if got := len(folder.View().Pending.Inputs); got != 1 {
		t.Fatalf("输入结算后应为 1，实际 %d", got)
	}
	mustNotice(t, folder, mainBranch, fixture[7]) // e8 结算 k1
	if got := len(folder.View().Pending.Tasks); got != 0 {
		t.Fatalf("后台任务结算后应为 0，实际 %d", got)
	}
	mustNotice(t, folder, mainBranch, fixture[8]) // e9 结算 c9
	if got := len(folder.View().Pending.Collaborations); got != 0 {
		t.Fatalf("协作请求结算后应为 0，实际 %d", got)
	}
	mustNotice(t, folder, mainBranch, fixture[9]) // e10 结算 q1
	if got := len(folder.View().Pending.Questions); got != 0 {
		t.Fatalf("提问结算后应为 0，实际 %d", got)
	}
	mustNotice(t, folder, mainBranch, fixture[10]) // e11 结算 e2
	if got := folder.View().Pending.Total(); got != 0 {
		t.Fatalf("全部结算后应为 0，实际 %d: %+v", got, folder.View().Pending)
	}
}

// TestDeliveredBeforeSettleKeepsRequestPending 验证成员接入只标记已接入，
// 请求仍留在清单里，直到结算事实到达。
func TestDeliveredBeforeSettleKeepsRequestPending(t *testing.T) {
	fixture := pendingFixture()
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(5, fixture[:5]...))
	seedAtCutoff(t, folder)

	mustNotice(t, folder, mainBranch, fixture[6]) // e7 delivered
	view := folder.View()
	if len(view.Pending.Collaborations) != 1 {
		t.Fatalf("接入不等于结算: %+v", view.Pending.Collaborations)
	}
	if !view.Pending.Collaborations[0].Delivered {
		t.Fatalf("应标记已接入: %+v", view.Pending.Collaborations[0])
	}
	if view.Pending.Collaborations[0].SessionID != "child-1" {
		t.Fatalf("成员会话标识没记下来: %+v", view.Pending.Collaborations[0])
	}
}

// TestLateSeedDoesNotResurrectSettledItems 验证读取晚于结算事件时不会把已结算的
// 事项重新装回清单。
func TestLateSeedDoesNotResurrectSettledItems(t *testing.T) {
	fixture := pendingFixture()
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(11, fixture...))

	settled := folder.View().Pending
	if settled.Total() != 0 {
		t.Fatalf("不带清单时应全部结算: %+v", settled)
	}

	// 服务端在截止点 5 读到的清单在结算事实之后才装进本地。
	seedAtCutoff(t, folder)
	view := folder.View()
	if view.Pending.Total() != 0 {
		t.Fatalf("晚到的清单把已结算的事项装回来了: %+v", view.Pending)
	}
}

// TestSeedAboveCutoffSettlementIsIgnored 验证截止点以下的结算事实不改清单：
// 清单是那一时刻的权威结果。
func TestSeedAboveCutoffSettlementIsIgnored(t *testing.T) {
	fixture := pendingFixture()
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(11, fixture...))

	// 服务端在截止点 9 仍报告 e1 未结算，则序号 6 的消费事实不能改这条清单项。
	folder.SeedPendingInputs(mainBranch, 9, []PendingInput{
		{InputID: "e1", Sequence: 1, Mode: "queue", Text: "第一条"},
	})
	view := folder.View()
	if len(view.Pending.Inputs) != 1 || view.Pending.Inputs[0].InputID != "e1" {
		t.Fatalf("截止点清单应压过更早的结算事实: %+v", view.Pending.Inputs)
	}
}

// TestTurnStartClosesPreviousQuestions 验证新的一轮开始会清掉上一轮遗留的提问。
func TestTurnStartClosesPreviousQuestions(t *testing.T) {
	folder := newFolder(t)
	fixture := pendingFixture()
	mustHistory(t, folder, mainBranch, page(5, fixture[:5]...))
	seedAtCutoff(t, folder)
	if len(folder.View().Pending.Questions) != 1 {
		t.Fatalf("提问应处于待答状态")
	}

	mustNotice(t, folder, mainBranch, ev("e12", 12, "turn", "start", `{"turn_id":"t2"}`))
	if len(folder.View().Pending.Questions) != 0 {
		t.Fatalf("新一轮开始后上一轮的提问应清空: %+v", folder.View().Pending.Questions)
	}
}

// TestAppliedInputStaysPendingButNotWithdrawable 验证"已应用未消费"的输入仍在清单里。
func TestAppliedInputStaysPendingButNotWithdrawable(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(1,
		ev("i1", 1, "input", "message", `{"mode":"queue","user":{"content":[{"type":"text","text":"排队"}]}}`),
	))
	mustNotice(t, folder, mainBranch, ev("i2", 2, "input", "applied", `{"input_id":"i1"}`))

	view := folder.View()
	if len(view.Pending.Inputs) != 1 {
		t.Fatalf("已应用的输入仍在清单里: %+v", view.Pending.Inputs)
	}
	if !view.Pending.Inputs[0].Applied {
		t.Fatalf("应标记已应用: %+v", view.Pending.Inputs[0])
	}

	mustNotice(t, folder, mainBranch, ev("i3", 3, "input", "consumed", `{"input_id":"i1"}`))
	if len(folder.View().Pending.Inputs) != 0 {
		t.Fatalf("消费后应移出清单")
	}
}

// TestWithdrawnInputLeavesViewAndInventory 验证撤回的输入从清单与视图里一起消失。
func TestWithdrawnInputLeavesViewAndInventory(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(1,
		ev("i1", 1, "input", "message", `{"mode":"queue","user":{"content":[{"type":"text","text":"排队中"}]}}`),
	))
	if len(folder.View().Pending.Inputs) != 1 {
		t.Fatalf("输入应处于未结算状态")
	}

	mustNotice(t, folder, mainBranch, ev("i2", 2, "input", "withdraw", `{"input_id":"i1","reason":"换个问法"}`))
	view := folder.View()
	if len(view.Pending.Inputs) != 0 {
		t.Fatalf("撤回后应移出清单: %+v", view.Pending.Inputs)
	}
	for _, row := range view.Rows {
		if row.Kind == RowUserMessage {
			t.Fatalf("撤回后用户行应消失: %+v", row)
		}
	}
}
