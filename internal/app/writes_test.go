package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// newWriteTestContext 返回带超时的上下文，供等待写意图收敛。
func newWriteTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// TestShellRunAndStopMapping 核对会话内命令执行与停止：方法、参数与回执里的运行标识。
func TestShellRunAndStopMapping(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"shell.run", "shell.stop"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "shell.run", Result: map[string]any{"runId": "run-srv-1"}},
			{Method: "shell.stop", Result: map[string]any{"stopped": true}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := newWriteTestContext(t)

	run, err := result.base.RunShell(ctx, ShellRunRequest{
		SessionID: testSession,
		Command:   "go test ./...",
	})
	if err != nil {
		t.Fatalf("登记命令执行: %v", err)
	}
	settled, err := result.base.Wait(ctx, run.ID)
	if err != nil || settled.Status != write.StatusSucceeded {
		t.Fatalf("命令执行应成功: %v %+v", err, settled)
	}
	if settled.InputID != "run-srv-1" {
		t.Fatalf("回执里的运行标识应取服务端值，实际 %q", settled.InputID)
	}
	requests := server.RequestsFor("shell.run")
	if len(requests) != 1 {
		t.Fatalf("shell.run 应只发一次，实际 %d", len(requests))
	}
	params := requestParams(t, requests[0])
	if params["sessionId"] != testSession || params["branchId"] != mainBranchID {
		t.Fatalf("命令执行没有带会话或主线: %v", params)
	}
	if params["command"] != "go test ./..." {
		t.Fatalf("命令执行缺少命令: %v", params)
	}

	stop, err := result.base.StopShell(ctx, ShellStopRequest{SessionID: testSession, RunID: "run-srv-1"})
	if err != nil {
		t.Fatalf("登记命令停止: %v", err)
	}
	if settledStop, err := result.base.Wait(ctx, stop.ID); err != nil || settledStop.Status != write.StatusSucceeded {
		t.Fatalf("命令停止应成功: %v %+v", err, settledStop)
	}
	stopRequests := server.RequestsFor("shell.stop")
	if len(stopRequests) != 1 {
		t.Fatalf("shell.stop 应只发一次，实际 %d", len(stopRequests))
	}
	stopParams := requestParams(t, stopRequests[0])
	if stopParams["sessionId"] != testSession || stopParams["runId"] != "run-srv-1" {
		t.Fatalf("命令停止参数不对: %v", stopParams)
	}
}

// TestModalityWriteMapping 核对四种模态写入：方法名、参数透传与回执标识。
func TestModalityWriteMapping(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{
			"modality.task.submit", "modality.task.cancel",
			"modality.realtime.open", "modality.realtime.close",
		},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "modality.task.submit", Result: map[string]any{"taskId": "task-1"}},
			{Method: "modality.task.cancel", Result: map[string]any{"ok": true}},
			{Method: "modality.realtime.open", Result: map[string]any{"connectionId": "conn-1"}},
			{Method: "modality.realtime.close", Result: map[string]any{"ok": true}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := newWriteTestContext(t)

	submit, err := result.base.ModalityWrite(ctx, ModalityWriteRequest{
		SessionID: testSession,
		Op:        ModalityTaskSubmit,
		Params:    map[string]any{"model": "m1"},
	})
	if err != nil {
		t.Fatalf("登记模态任务: %v", err)
	}
	settled, err := result.base.Wait(ctx, submit.ID)
	if err != nil || settled.Status != write.StatusSucceeded {
		t.Fatalf("模态任务应成功: %v %+v", err, settled)
	}
	if settled.InputID != "task-1" {
		t.Fatalf("模态任务标识应取服务端值，实际 %q", settled.InputID)
	}
	submitParams := requestParams(t, server.RequestsFor("modality.task.submit")[0])
	if submitParams["sessionId"] != testSession || submitParams["branchId"] != mainBranchID {
		t.Fatalf("模态任务没有带会话或主线: %v", submitParams)
	}
	if submitParams["model"] != "m1" {
		t.Fatalf("模态载荷没有透传: %v", submitParams)
	}
	if _, ok := submitParams["op"]; ok {
		t.Fatalf("op 是意图级字段，不应下发: %v", submitParams)
	}

	cancel, err := result.base.ModalityWrite(ctx, ModalityWriteRequest{
		SessionID: testSession, Op: ModalityTaskCancel, Params: map[string]any{"taskId": "task-1"},
	})
	if err != nil {
		t.Fatalf("登记模态取消: %v", err)
	}
	if settledCancel, err := result.base.Wait(ctx, cancel.ID); err != nil || settledCancel.Status != write.StatusSucceeded {
		t.Fatalf("模态取消应成功: %v %+v", err, settledCancel)
	}
	open, err := result.base.ModalityWrite(ctx, ModalityWriteRequest{SessionID: testSession, Op: ModalityRealtimeOpen})
	if err != nil {
		t.Fatalf("登记实时传输打开: %v", err)
	}
	settledOpen, err := result.base.Wait(ctx, open.ID)
	if err != nil || settledOpen.Status != write.StatusSucceeded {
		t.Fatalf("实时传输打开应成功: %v %+v", err, settledOpen)
	}
	if settledOpen.InputID != "conn-1" {
		t.Fatalf("实时连接标识应取服务端值，实际 %q", settledOpen.InputID)
	}
	closeIntent, err := result.base.ModalityWrite(ctx, ModalityWriteRequest{
		SessionID: testSession, Op: ModalityRealtimeClose, Params: map[string]any{"connectionId": "conn-1"},
	})
	if err != nil {
		t.Fatalf("登记实时传输关闭: %v", err)
	}
	if settledClose, err := result.base.Wait(ctx, closeIntent.ID); err != nil || settledClose.Status != write.StatusSucceeded {
		t.Fatalf("实时传输关闭应成功: %v %+v", err, settledClose)
	}

	for _, method := range []string{
		"modality.task.submit", "modality.task.cancel",
		"modality.realtime.open", "modality.realtime.close",
	} {
		if got := len(server.RequestsFor(method)); got != 1 {
			t.Fatalf("%s 应只发一次，实际 %d", method, got)
		}
	}
	closeParams := requestParams(t, server.RequestsFor("modality.realtime.close")[0])
	if closeParams["connectionId"] != "conn-1" {
		t.Fatalf("关闭实时传输没有带连接标识: %v", closeParams)
	}
}

// TestBranchLifecycleMapping 核对开分支与结束分支：方法、参数与能力门禁。
func TestBranchLifecycleMapping(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.branch_create", "session.branch_end"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.branch_create", Result: map[string]any{"branch": map[string]any{
				"branchId": "b-1", "parentBranchId": "main", "branchPoint": 3,
			}}},
			{Method: "session.branch_end", Result: nil},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := newWriteTestContext(t)
	writes := result.base.Writes()

	created, err := writes.Register(write.Intent{
		Key:       "branch-create-1",
		Kind:      write.KindSessionLifecycle,
		SessionID: testSession,
		Params:    json.RawMessage(`{"op":"branch.create","parentBranchId":"main","at":3,"prompt":"换一条路"}`),
	})
	if err != nil {
		t.Fatalf("登记开分支: %v", err)
	}
	if settled, err := writes.Wait(ctx, created.ID); err != nil || settled.Status != write.StatusSucceeded {
		t.Fatalf("开分支应成功: %v %+v", err, settled)
	}
	createRequests := server.RequestsFor("session.branch_create")
	if len(createRequests) != 1 {
		t.Fatalf("session.branch_create 应只发一次，实际 %d", len(createRequests))
	}
	createParams := requestParams(t, createRequests[0])
	if createParams["sessionId"] != testSession || createParams["parentBranchId"] != "main" {
		t.Fatalf("开分支参数不对: %v", createParams)
	}
	if createParams["at"] != float64(3) || createParams["prompt"] != "换一条路" {
		t.Fatalf("开分支缺少切点或首条输入: %v", createParams)
	}

	ended, err := writes.Register(write.Intent{
		Key:       "branch-end-1",
		Kind:      write.KindSessionLifecycle,
		SessionID: testSession,
		BranchID:  "b-1",
		Params:    json.RawMessage(`{"op":"branch.end"}`),
	})
	if err != nil {
		t.Fatalf("登记结束分支: %v", err)
	}
	if settled, err := writes.Wait(ctx, ended.ID); err != nil || settled.Status != write.StatusSucceeded {
		t.Fatalf("结束分支应成功: %v %+v", err, settled)
	}
	endRequests := server.RequestsFor("session.branch_end")
	if len(endRequests) != 1 {
		t.Fatalf("session.branch_end 应只发一次，实际 %d", len(endRequests))
	}
	endParams := requestParams(t, endRequests[0])
	if endParams["sessionId"] != testSession || endParams["branchId"] != "b-1" {
		t.Fatalf("结束分支参数不对: %v", endParams)
	}
}

// TestNewWriteKindsRespectGates 核对三类新意图的门禁：缺方法零请求、退出中零请求。
func TestNewWriteKindsRespectGates(t *testing.T) {
	// 服务端没有这些会话方法：sender 先查能力，缺方法即失败且不发请求。
	server := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := newWriteTestContext(t)

	noCapability, err := result.base.RunShell(ctx, ShellRunRequest{SessionID: testSession, Command: "pwd"})
	if err != nil {
		t.Fatalf("登记命令执行: %v", err)
	}
	settledRun, err := result.base.Wait(ctx, noCapability.ID)
	if err != nil {
		t.Fatalf("等待命令执行收敛: %v", err)
	}
	if settledRun.Status != write.StatusFailed || !strings.Contains(settledRun.Reason, "能力") {
		t.Fatalf("缺方法应明确失败并说明能力缺失: %+v", settledRun)
	}
	if got := len(server.RequestsFor("shell.run")); got != 0 {
		t.Fatalf("缺方法时不应发 shell.run，实际 %d", got)
	}

	// 退出中：登记入口直接拒绝；直接登记到写登记的意图也在发送前被挡住。
	if err := result.base.StopIntents(ctx); err != nil {
		t.Fatalf("停止新意图: %v", err)
	}
	if _, err := result.base.RunShell(ctx, ShellRunRequest{SessionID: testSession, Command: "pwd"}); !errors.Is(err, ErrExiting) {
		t.Fatalf("退出中命令执行应被拒: %v", err)
	}
	if _, err := result.base.ModalityWrite(ctx, ModalityWriteRequest{SessionID: testSession, Op: ModalityRealtimeOpen}); !errors.Is(err, ErrExiting) {
		t.Fatalf("退出中模态写入应被拒: %v", err)
	}
	registered, err := result.base.Writes().Register(write.Intent{
		Key:       "modality-after-exit",
		Kind:      write.KindModalityWrite,
		SessionID: testSession,
		Params:    json.RawMessage(`{"op":"realtime.open"}`),
	})
	if err != nil {
		t.Fatalf("登记模态写入: %v", err)
	}
	settled, err := result.base.Writes().Wait(ctx, registered.ID)
	if err != nil {
		t.Fatalf("等待收敛: %v", err)
	}
	if settled.Status != write.StatusFailed || !strings.Contains(settled.Reason, "退出") {
		t.Fatalf("退出中的写入应在发送前被拒: %+v", settled)
	}
	if got := len(server.RequestsFor("modality.realtime.open")); got != 0 {
		t.Fatalf("退出中不应发模态请求，实际 %d", got)
	}
}
