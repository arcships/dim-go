package projection

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// fakeWithdrawSender 记下撤回请求，按预设回答。
type fakeWithdrawSender struct {
	result InputWithdrawResult
	err    error
	reqs   []InputWithdrawRequest
}

func (f *fakeWithdrawSender) WithdrawInput(
	_ context.Context,
	req InputWithdrawRequest,
) (InputWithdrawResult, error) {
	f.reqs = append(f.reqs, req)
	if f.err != nil {
		return InputWithdrawResult{}, f.err
	}
	return f.result, nil
}

// fakeGoalSender 记下目标控制请求。
type fakeGoalSender struct {
	result GoalControlResult
	err    error
	reqs   []GoalControlRequest
}

func (f *fakeGoalSender) ControlGoal(
	_ context.Context,
	req GoalControlRequest,
) (GoalControlResult, error) {
	f.reqs = append(f.reqs, req)
	if f.err != nil {
		return GoalControlResult{}, f.err
	}
	return f.result, nil
}

var fullMethods = []string{
	"session.input.pending",
	"session.input.withdraw",
	"session.goal.get",
	"session.goal.pause",
	"session.goal.resume",
}

// TestInputWithdrawNeedsMethod 验证方法表里没有撤回方法时能力缺失、零请求。
func TestInputWithdrawNeedsMethod(t *testing.T) {
	folder := taskFolder(t, "task-1")
	sender := &fakeWithdrawSender{result: InputWithdrawResult{Status: "deleted"}}
	if _, err := NewQueuedInputWithdraw(folder, mainBranch, "i1", []string{"session.input.pending"}, sender); !errors.Is(err, ErrInputWithdrawUnavailable) {
		t.Fatalf("缺方法应报能力缺失: %v", err)
	}
	if len(sender.reqs) != 0 {
		t.Fatalf("能力缺失时不该发请求: %+v", sender.reqs)
	}
	if _, err := NewQueuedInputWithdraw(folder, mainBranch, "i1", nil, sender); !errors.Is(err, ErrInputWithdrawUnavailable) {
		t.Fatalf("没探测过方法表也按能力缺失处理: %v", err)
	}
	if _, err := NewQueuedInputWithdraw(folder, mainBranch, "", fullMethods, sender); !errors.Is(err, ErrInputWithdrawNoInput) {
		t.Fatalf("缺输入标识应失败: %v", err)
	}
	if _, err := NewQueuedInputWithdraw(folder, mainBranch, "i1", fullMethods, nil); !errors.Is(err, ErrInputWithdrawNoSender) {
		t.Fatalf("缺发送方应失败: %v", err)
	}
}

// TestInputWithdrawSettlesByFact 验证撤回发一次请求，清单与视图由事实同步移除。
func TestInputWithdrawSettlesByFact(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(1,
		ev("i1", 1, "input", "message", `{"mode":"queue","user":{"content":[{"type":"text","text":"排队中"}]}}`),
	))
	if len(folder.View().Pending.Inputs) != 1 {
		t.Fatalf("应先有未处理输入")
	}

	sender := &fakeWithdrawSender{result: InputWithdrawResult{Status: "deleted"}}
	withdraw, err := NewQueuedInputWithdraw(folder, mainBranch, "i1", fullMethods, sender)
	if err != nil {
		t.Fatalf("创建撤回: %v", err)
	}
	result, err := withdraw.Withdraw(context.Background())
	if err != nil {
		t.Fatalf("撤回: %v", err)
	}
	if !result.Deleted() {
		t.Fatalf("回执应是已删除: %+v", result)
	}
	if len(sender.reqs) != 1 || sender.reqs[0].InputID != "i1" || sender.reqs[0].SessionID != "s-1" {
		t.Fatalf("撤回请求不对: %+v", sender.reqs)
	}
	// 回执本身不改视图：服务端写下的事实才是驱动者。
	if len(folder.View().Pending.Inputs) != 1 {
		t.Fatalf("回执不该本地预删")
	}

	mustNotice(t, folder, mainBranch, ev("w1", 2, "input", "withdraw", `{"input_id":"i1","reason":"换个问法"}`))
	view := folder.View()
	if len(view.Pending.Inputs) != 0 {
		t.Fatalf("撤回后清单应去掉: %+v", view.Pending.Inputs)
	}
	for _, row := range view.Rows {
		if row.Kind == RowUserMessage {
			t.Fatalf("撤回后用户行应消失: %+v", row)
		}
	}
}

// TestInputWithdrawIntent 验证写登记意图的形状。
func TestInputWithdrawIntent(t *testing.T) {
	folder := newFolder(t)
	sender := &fakeWithdrawSender{}
	withdraw, err := NewQueuedInputWithdraw(folder, "side", "i9", fullMethods, sender)
	if err != nil {
		t.Fatalf("创建撤回: %v", err)
	}
	intent := withdraw.Intent()
	if intent.Op() != "input.withdraw" || intent.Key() == "" {
		t.Fatalf("意图不对: %+v", intent)
	}
	if intent.Key() != withdraw.Intent().Key() {
		t.Fatalf("幂等键应稳定")
	}
	raw, err := intent.Params()
	if err != nil {
		t.Fatalf("编码参数: %v", err)
	}
	var params map[string]any
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatalf("参数不是合法 JSON: %v", err)
	}
	want := map[string]any{"op": "input.withdraw", "branchId": "side", "inputId": "i9"}
	if !reflect.DeepEqual(params, want) {
		t.Fatalf("参数不对: %+v", params)
	}
}

// TestGoalControlNeedsMethod 验证目标控制先探测方法名。
func TestGoalControlNeedsMethod(t *testing.T) {
	folder := newFolder(t)
	sender := &fakeGoalSender{}
	if _, err := NewGoalControl(folder, mainBranch, GoalPause, []string{"session.goal.get"}, sender); !errors.Is(err, ErrGoalControlUnavailable) {
		t.Fatalf("缺方法应报能力缺失: %v", err)
	}
	if len(sender.reqs) != 0 {
		t.Fatalf("能力缺失时不该发请求")
	}
	if _, err := NewGoalControl(folder, mainBranch, "edit", fullMethods, sender); !errors.Is(err, ErrGoalControlBadAction) {
		t.Fatalf("非法动作应被拒绝: %v", err)
	}
	if _, err := NewGoalControl(folder, mainBranch, GoalPause, fullMethods, nil); !errors.Is(err, ErrGoalControlNoSender) {
		t.Fatalf("缺发送方应失败: %v", err)
	}
}

// TestGoalPauseAndResume 验证暂停/恢复各发一次，状态由服务端写下的记录结算。
func TestGoalPauseAndResume(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(1,
		goalRecord(t, "g1", 1, "<dim-goal kind=\"state\" status=\"active\" branch=\"main\">\n<objective>\n修好构建\n</objective>\n目标在进行中。\n</dim-goal>"),
	))

	sender := &fakeGoalSender{result: GoalControlResult{Sequence: 2}}
	pause, err := NewGoalControl(folder, mainBranch, GoalPause, fullMethods, sender)
	if err != nil {
		t.Fatalf("创建暂停: %v", err)
	}
	if pause.Intent().Op() != "goal.pause" {
		t.Fatalf("暂停 op 不对: %q", pause.Intent().Op())
	}
	if _, err := pause.Apply(context.Background()); err != nil {
		t.Fatalf("暂停: %v", err)
	}
	if len(sender.reqs) != 1 || sender.reqs[0].Action != GoalPause {
		t.Fatalf("暂停请求不对: %+v", sender.reqs)
	}
	// 回执不改状态：服务端写下的记录才是驱动者。
	if folder.View().Goal == nil || folder.View().Goal.Status != GoalActive {
		t.Fatalf("回执不该本地预改目标: %+v", folder.View().Goal)
	}

	mustNotice(t, folder, mainBranch, goalRecord(t, "g2", 3,
		"<dim-goal kind=\"state\" status=\"paused\" reason=\"user\" branch=\"main\">\n<objective>\n修好构建\n</objective>\n目标已暂停。\n</dim-goal>"))
	goal := folder.View().Goal
	if goal == nil || goal.Status != GoalPaused || goal.Reason != "user" {
		t.Fatalf("暂停后目标状态不对: %+v", goal)
	}

	resume, err := NewGoalControl(folder, mainBranch, GoalResume, fullMethods, sender)
	if err != nil {
		t.Fatalf("创建恢复: %v", err)
	}
	if resume.Intent().Op() != "goal.resume" {
		t.Fatalf("恢复 op 不对: %q", resume.Intent().Op())
	}
	if _, err := resume.Apply(context.Background()); err != nil {
		t.Fatalf("恢复: %v", err)
	}
	mustNotice(t, folder, mainBranch, goalRecord(t, "g3", 4,
		"<dim-goal kind=\"state\" status=\"active\" branch=\"main\">\n<objective>\n修好构建\n</objective>\n目标在进行中。\n</dim-goal>"))
	goal = folder.View().Goal
	if goal == nil || goal.Status != GoalActive {
		t.Fatalf("恢复后目标状态不对: %+v", goal)
	}
}
