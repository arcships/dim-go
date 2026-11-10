package projection

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// fakeStopSender 记下停止请求，按预设结果回答。
type fakeStopSender struct {
	result ExecStopResult
	err    error
	reqs   []ExecStopRequest
}

func (f *fakeStopSender) StopExecTask(
	_ context.Context,
	req ExecStopRequest,
) (ExecStopResult, error) {
	f.reqs = append(f.reqs, req)
	if f.err != nil {
		return ExecStopResult{}, f.err
	}
	return f.result, nil
}

// TestExecTaskStopMarksStoppingUntilFact 验证停止发出去之后本地标"正在停止"，
// 任务仍由结束事实结算。
func TestExecTaskStopMarksStoppingUntilFact(t *testing.T) {
	folder := taskFolder(t, "task-9")
	sender := &fakeStopSender{result: ExecStopResult{TaskID: "task-9", Status: "cancelled"}}
	stop, err := NewExecTaskStop(folder, mainBranch, "task-9", sender)
	if err != nil {
		t.Fatalf("创建任务停止: %v", err)
	}

	result, err := stop.Stop(context.Background())
	if err != nil {
		t.Fatalf("停止任务: %v", err)
	}
	if !result.Stopped() || result.Status != "cancelled" {
		t.Fatalf("停止结果不对: %+v", result)
	}
	if len(sender.reqs) != 1 {
		t.Fatalf("应发一次停止请求: %+v", sender.reqs)
	}
	if sender.reqs[0].SessionID != "s-1" || sender.reqs[0].BranchID != mainBranch || sender.reqs[0].TaskID != "task-9" {
		t.Fatalf("停止请求不对: %+v", sender.reqs[0])
	}

	// 回执先到：任务还留着，但已标成正在停止。
	view := folder.View()
	if len(view.Pending.Tasks) != 1 || !view.Pending.Tasks[0].Stopping {
		t.Fatalf("停止后应标记正在停止: %+v", view.Pending.Tasks)
	}
	if !folder.TaskRunning(mainBranch, "task-9") {
		t.Fatalf("结束事实到达前任务仍算在跑")
	}

	// 结束事实到达：任务从清单消失，标记随之消失。
	mustNotice(t, folder, mainBranch, ev("end9", 2, "exec", "ended", `{"task_id":"task-9","status":"cancelled"}`))
	view = folder.View()
	if len(view.Pending.Tasks) != 0 {
		t.Fatalf("结束后清单应为空: %+v", view.Pending.Tasks)
	}
	if folder.TaskRunning(mainBranch, "task-9") {
		t.Fatalf("结束后任务不该再算在跑")
	}
}

// TestExecTaskStopIntent 验证登记到写登记的意图形状：op、幂等键与参数。
func TestExecTaskStopIntent(t *testing.T) {
	folder := taskFolder(t, "task-1")
	stop, err := NewExecTaskStop(folder, "side", "task-1", &fakeStopSender{})
	if err != nil {
		t.Fatalf("创建任务停止: %v", err)
	}
	intent := stop.Intent()
	if intent.Op() != "exec.stop" {
		t.Fatalf("op 不对: %q", intent.Op())
	}
	if intent.SessionID != "s-1" || intent.BranchID != "side" || intent.TaskID != "task-1" {
		t.Fatalf("意图字段不对: %+v", intent)
	}
	key := intent.Key()
	if key == "" || key != stop.Intent().Key() {
		t.Fatalf("幂等键应稳定: %q", key)
	}
	other, err := NewExecTaskStop(folder, "side", "task-2", &fakeStopSender{})
	if err != nil {
		t.Fatalf("创建任务停止: %v", err)
	}
	if other.Intent().Key() == key {
		t.Fatalf("不同任务的幂等键不该相同")
	}

	raw, err := intent.Params()
	if err != nil {
		t.Fatalf("编码参数: %v", err)
	}
	var params map[string]any
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatalf("参数不是合法 JSON: %v", err)
	}
	if params["op"] != "exec.stop" || params["branchId"] != "side" || params["taskId"] != "task-1" {
		t.Fatalf("参数不对: %+v", params)
	}
}

// TestExecTaskStopFailures 验证发送失败不标记、构造参数缺失明确失败。
func TestExecTaskStopFailures(t *testing.T) {
	folder := taskFolder(t, "task-1")
	sender := &fakeStopSender{err: errors.New("断开")}
	stop, err := NewExecTaskStop(folder, mainBranch, "task-1", sender)
	if err != nil {
		t.Fatalf("创建任务停止: %v", err)
	}
	if _, err := stop.Stop(context.Background()); err == nil {
		t.Fatalf("发送失败应原样返回")
	}
	if view := folder.View(); len(view.Pending.Tasks) != 1 || view.Pending.Tasks[0].Stopping {
		t.Fatalf("发送失败不该标记正在停止: %+v", view.Pending.Tasks)
	}

	if _, err := NewExecTaskStop(folder, mainBranch, "", &fakeStopSender{}); !errors.Is(err, ErrExecStopNoTask) {
		t.Fatalf("缺任务标识应失败: %v", err)
	}
	if _, err := NewExecTaskStop(folder, mainBranch, "task-1", nil); !errors.Is(err, ErrExecStopNoSender) {
		t.Fatalf("缺发送方应失败: %v", err)
	}
}
