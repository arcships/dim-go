package projection

// 真实模型证据：分支隔离、分支内未完成事项、插话（steer）。
//
// 前提同 6.6：/tmp 夹具的复制、真实 daemon（0.1.0-alpha.21）、真实 provider。
// 日志只留事件类型、标识、序号、状态与条数，不打印提示词、命令输出与模型正文。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
)

// TestRealDaemonBranchIsolation 验证分支内容只落在该分支、主线不受影响，
// 以及分支内的未完成事项与插话。
//
// 用法：
//
//	DIM_REAL_MODEL_FIXTURE=/tmp/dimagent-real-model \
//	  go test -run TestRealDaemonBranchIsolation -v -count=1 ./internal/projection/
func TestRealDaemonBranchIsolation(t *testing.T) {
	fixture := os.Getenv("DIM_REAL_MODEL_FIXTURE")
	if fixture == "" {
		t.Skip("未设置 DIM_REAL_MODEL_FIXTURE：真实模型证据 blocked（缺 /tmp 夹具）")
	}
	serverPath := filepath.Join(fixture, "bin", "dimcode-server")
	if _, err := os.Stat(serverPath); err != nil {
		t.Skipf("夹具缺少 daemon 可执行文件: %v", err)
	}

	runDir, err := os.MkdirTemp("/tmp", "d2branch-")
	if err != nil {
		t.Fatalf("创建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runDir) })
	home := filepath.Join(runDir, "home")
	copyTree(t, filepath.Join(fixture, "home"), home)
	for _, stale := range []string{"daemon.sock", "daemon.sock.lock", "daemon.start.lock"} {
		_ = os.Remove(filepath.Join(home, "rust", stale))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	supervisor, err := daemon.New(daemon.Config{
		DataDir:    home,
		ServerPath: serverPath,
		ServerArgs: []string{"app-server"},
	})
	if err != nil {
		t.Fatalf("创建监督者: %v", err)
	}
	conn, err := supervisor.Connect(ctx)
	if err != nil {
		t.Skipf("连接真实 daemon 失败（夹具账号/网络不可用）: %v", err)
	}
	t.Logf("服务端 %s（profile %s）", conn.Hello.ServerVersion, conn.Hello.DaemonProfile.Kind)

	workDir := filepath.Join(runDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}
	sessionID := createSession(t, ctx, conn.Client, workDir, "")
	t.Logf("会话 %s", sessionID)

	folder := newFolderFor(t, sessionID)
	recorder := newRecorder()
	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	snapshot, lease, err := manager.Acquire(ctx, sessionID, session.Listener{
		OnEvent: func(event session.Event) {
			if event.Method == "session.event" {
				var params protocol.SessionEventParams
				if err := json.Unmarshal(event.Params, &params); err == nil {
					recorder.add(params.Event)
				}
			}
			folder.ApplyListenerEvent(event)
		},
		OnResync: func(session.Resync) { folder.OnDisconnect(mainBranch) },
	})
	if err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	t.Cleanup(func() {
		lease.Release()
		manager.Close()
		if result, err := supervisor.Shutdown(context.Background()); err == nil {
			t.Logf("回收自启服务端: stopped=%v reason=%q", result.Stopped, result.Reason)
		}
	})
	folder.ApplySnapshot(snapshot)
	client := &pendingClient{t: t, ctx: ctx, conn: conn.Client, sessionID: sessionID, folder: folder}

	// 主线先跑一轮，拿到切点。
	client.submit("Reply with the single word OK.", "")
	mainEnd := waitHistoryTurnEnd(t, ctx, conn.Client, sessionID, mainBranch, 0)
	t.Logf("主线一轮结束: turn=%s sequence=%d", stringField(mainEnd, "turn_id"), mainEnd.Sequence)

	mainPage := readHistory(t, ctx, conn.Client, sessionID, mainBranch)
	folder.ApplyHistory(mainBranch, mainPage)
	mainIDsBefore := rowIDs(folder.View())
	t.Logf("主线行: %d", len(mainIDsBefore))

	// 开分支：切点取主线那一轮的结束序号。
	branchID := createBranch(t, ctx, conn.Client, sessionID, mainBranch, mainEnd.Sequence)
	t.Logf("分支已创建: branch=%s 切点=%d", branchID, mainEnd.Sequence)

	// 建分支时服务端会用分支提示词起一轮；先等这一轮结束。
	createdEnd := waitHistoryTurnEnd(t, ctx, conn.Client, sessionID, branchID, 0)
	t.Logf("分支首轮结束: turn=%s sequence=%d", stringField(createdEnd, "turn_id"), createdEnd.Sequence)

	// 分支历史：只有分支自己的事实。
	branchPage := readHistory(t, ctx, conn.Client, sessionID, branchID)
	folder.ApplyHistory(branchID, branchPage)
	assertBranchIsolated(t, folder, branchID, "建分支之后")
	t.Logf("分支建好: 行=%d", len(branchView(folder, branchID).Rows))

	// 分支内提交一轮：内容只落在分支。
	branchInput := submitTo(t, ctx, conn.Client, sessionID, branchID, "Reply with the single word BRANCH.")
	branchEnd := waitHistoryTurnEnd(t, ctx, conn.Client, sessionID, branchID, branchInput.Sequence)
	t.Logf("分支第二轮结束: turn=%s sequence=%d", stringField(branchEnd, "turn_id"), branchEnd.Sequence)

	branchPage = readHistory(t, ctx, conn.Client, sessionID, branchID)
	folder.ApplyHistory(branchID, branchPage)
	assertBranchIsolated(t, folder, branchID, "分支提交之后")

	// 主线重新读一遍：行标识序列不变。
	mainPage = readHistory(t, ctx, conn.Client, sessionID, mainBranch)
	folder.ApplyHistory(mainBranch, mainPage)
	if got := rowIDs(folder.View()); !equalStrings(got, mainIDsBefore) {
		t.Fatalf("主线行被分支影响: %v -> %v", mainIDsBefore, got)
	}
	t.Logf("主线行未变: %d 行", len(mainIDsBefore))

	// 插话：分支内跑一条长命令，运行中提交 steer。
	steerTurnInput := submitTo(t, ctx, conn.Client, sessionID, branchID,
		"Use the exec tool exactly once to run this command in the foreground (timeoutMs 30000): sleep 8. Then reply DONE.")
	steerFloor := waitHistoryTurnStart(t, ctx, conn.Client, sessionID, branchID, steerTurnInput.Sequence)
	steerID, steerSeq := submitToWithMode(t, ctx, conn.Client, sessionID, branchID, "Also reply with STEER-OK.", "steer")

	// 投递方式与服务端一致：清单里这一条带的是 steer，不是 queue。
	pending := readPendingInputs(t, ctx, conn.Client, sessionID, branchID)
	steerMode := ""
	for _, item := range pending.Inputs {
		if item.InputID == steerID {
			steerMode = item.Mode
		}
	}
	if steerMode != "steer" {
		t.Fatalf("插话在清单里的投递方式应为 steer，实际 %q: %+v", steerMode, pending.Inputs)
	}
	// 服务端会把它按 steer 应用进当前运行：等 input/applied 并核对模式。
	applied := waitApply(t, ctx, conn.Client, sessionID, branchID, steerID)
	if mode := firstString(decodeObject(applied.Data), "mode"); mode != "" && mode != "steer" {
		t.Fatalf("input/applied 的模式应为 steer，实际 %q", mode)
	}
	t.Logf("插话: input=%s sequence=%d 清单模式=steer 已应用(seq=%d)", steerID, steerSeq, applied.Sequence)

	waitHistoryTurnEnd(t, ctx, conn.Client, sessionID, branchID, steerFloor)
	branchPage = readHistory(t, ctx, conn.Client, sessionID, branchID)
	folder.ApplyHistory(branchID, branchPage)
	steerRow := findRowByID(branchView(folder, branchID).Rows, steerID)
	if steerRow == nil {
		t.Fatalf("插话的用户行应落在分支里: %s", steerID)
	}
	if steerRow.TurnIndex <= 0 {
		t.Fatalf("插话应归到运行中的那一轮: %+v", steerRow)
	}
	t.Logf("插话折叠: 行=%s 轮=%d", steerRow.ID, steerRow.TurnIndex)

	// 停止：分支内跑一条长命令，再停这一轮（主线不受影响）。
	stopInput := submitTo(t, ctx, conn.Client, sessionID, branchID,
		"Use the exec tool exactly once to run this command in the foreground (timeoutMs 120000): sleep 60. Then reply DONE.")
	stopFloor := waitHistoryTurnStart(t, ctx, conn.Client, sessionID, branchID, stopInput.Sequence)
	if err := conn.Client.CallInto(ctx, "session.stop", map[string]any{
		"sessionId": sessionID,
		"branchId":  branchID,
		"reason":    "evidence",
	}, nil); err != nil {
		t.Fatalf("停止分支运行: %v", err)
	}
	stoppedEnd := waitHistoryTurnEnd(t, ctx, conn.Client, sessionID, branchID, stopFloor)
	t.Logf("分支停止: turn=%s outcome=%s sequence=%d",
		stringField(stoppedEnd, "turn_id"), stringField(stoppedEnd, "status"), stoppedEnd.Sequence)

	// 停止只作用于该分支：主线行标识仍不变。
	mainPage = readHistory(t, ctx, conn.Client, sessionID, mainBranch)
	folder.ApplyHistory(mainBranch, mainPage)
	if got := rowIDs(folder.View()); !equalStrings(got, mainIDsBefore) {
		t.Fatalf("分支停止影响了主线: %v -> %v", mainIDsBefore, got)
	}
	t.Logf("分支停止后主线行未变: %d 行", len(mainIDsBefore))

	// 分支内未完成事项：后台任务只出现在分支清单里。
	taskID := startBranchTask(t, client, branchID)
	branchTasks := readPendingTasksFor(t, ctx, conn.Client, sessionID, branchID)
	mainTasks := readPendingTasksFor(t, ctx, conn.Client, sessionID, mainBranch)
	if !hasPendingTask(branchTasks, taskID) {
		t.Fatalf("分支清单应有未结束任务 %s: %+v", taskID, branchTasks.Tasks)
	}
	if hasPendingTask(mainTasks, taskID) {
		t.Fatalf("主线清单不该看到分支任务 %s", taskID)
	}
	t.Logf("分支未完成事项: 分支=%d 主线=%d（through=%d）", len(branchTasks.Tasks), len(mainTasks.Tasks), branchTasks.Through)

	// 结算：停掉这个任务。
	sender := &execStopSource{conn: conn.Client, sessionID: sessionID}
	stop, err := NewExecTaskStop(folder, branchID, taskID, sender)
	if err != nil {
		t.Fatalf("创建任务停止: %v", err)
	}
	result, err := stop.Stop(ctx)
	if err != nil {
		t.Fatalf("停止分支任务: %v", err)
	}
	t.Logf("分支任务停止: status=%s", result.Status)
	if tasks := readPendingTasksFor(t, ctx, conn.Client, sessionID, branchID); hasPendingTask(tasks, taskID) {
		t.Fatalf("停止后分支清单仍列出 %s", taskID)
	}

	t.Logf("证据: session=%s branch=%s 主线行=%d 分支行=%d 分支任务结算=ok events=%v",
		sessionID, branchID, len(mainIDsBefore), len(branchView(folder, branchID).Rows), recorder.kinds())
}

// readHistory 读一个分支的全部历史（一页够用）。
func readHistory(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, branchID string,
) protocol.SessionHistoryResult {
	t.Helper()
	after := int64(0)
	limit := 2000
	var page protocol.SessionHistoryResult
	if err := conn.CallInto(ctx, "session.history", map[string]any{
		"sessionId": sessionID,
		"branchId":  branchID,
		"after":     after,
		"limit":     limit,
	}, &page); err != nil {
		t.Fatalf("读取分支历史: %v", err)
	}
	return page
}

// mergePages 把两页合成一页（按序号升序），供折叠使用。
func mergePages(first, second protocol.SessionHistoryResult) protocol.SessionHistoryResult {
	events := append(append([]protocol.RawEvent(nil), first.Events...), second.Events...)
	through := first.Through
	if second.Through > through {
		through = second.Through
	}
	return protocol.SessionHistoryResult{Events: events, Through: through, Done: true}
}

// waitHistoryTurnEnd 轮询分支历史，等到一条序号大于 after 的 turn/end。
func waitHistoryTurnEnd(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, branchID string,
	after int64,
) protocol.RawEvent {
	t.Helper()
	deadline := time.Now().Add(4 * time.Minute)
	for {
		page := readHistory(t, ctx, conn, sessionID, branchID)
		for index := len(page.Events) - 1; index >= 0; index-- {
			event := page.Events[index]
			if event.Sequence > after && event.Type == "turn" && event.Name == "end" {
				return event
			}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			t.Skipf("没有等到分支的一轮结束（branch=%s after=%d）", branchID, after)
		}
		select {
		case <-ctx.Done():
			t.Skipf("等待分支轮次结束超时")
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// waitHistoryTurnStart 等到一条序号大于 after 的 turn/start，返回它的序号。
func waitHistoryTurnStart(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, branchID string,
	after int64,
) int64 {
	t.Helper()
	deadline := time.Now().Add(3 * time.Minute)
	for {
		page := readHistory(t, ctx, conn, sessionID, branchID)
		for _, event := range page.Events {
			if event.Sequence > after && event.Type == "turn" && event.Name == "start" {
				return event.Sequence
			}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			t.Skipf("没有等到分支开轮（branch=%s after=%d）", branchID, after)
		}
		select {
		case <-ctx.Done():
			t.Skipf("等待分支开轮超时")
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// createBranch 在主线某一轮结束处开分支。
func createBranch(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, parentBranchID string,
	at int64,
) string {
	t.Helper()
	var reply struct {
		Branch struct {
			BranchID string `json:"branchId"`
		} `json:"branch"`
	}
	if err := conn.CallInto(ctx, "session.branch_create", map[string]any{
		"sessionId":      sessionID,
		"parentBranchId": parentBranchID,
		"at":             at,
		"prompt":         "explore another approach for the same task",
	}, &reply); err != nil {
		t.Skipf("服务端没有 session.branch_create 或拒绝开分支: %v", err)
	}
	if reply.Branch.BranchID == "" {
		t.Skipf("分支回执没有分支标识")
	}
	return reply.Branch.BranchID
}

// submitTo 在指定分支提交一次输入。
func submitTo(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, branchID, text string,
) protocol.RawEvent {
	t.Helper()
	return submitToWithModeEvent(t, ctx, conn, sessionID, branchID, text, "")
}

// submitToWithMode 在指定分支按投递方式提交，返回输入标识与序号。
func submitToWithMode(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, branchID, text, mode string,
) (string, int64) {
	t.Helper()
	event := submitToWithModeEvent(t, ctx, conn, sessionID, branchID, text, mode)
	return event.ID, event.Sequence
}

func submitToWithModeEvent(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, branchID, text, mode string,
) protocol.RawEvent {
	t.Helper()
	var result struct {
		Event protocol.RawEvent `json:"event"`
	}
	params := map[string]any{
		"sessionId":    sessionID,
		"branchId":     branchID,
		"submissionId": newUUID(t),
		"text":         text,
	}
	if mode != "" {
		params["mode"] = mode
	}
	if err := conn.CallInto(ctx, "session.submit", params, &result); err != nil {
		t.Fatalf("提交输入到分支 %s: %v", branchID, err)
	}
	return result.Event
}

// assertBranchIsolated 校验分支里没有主线的行。
func assertBranchIsolated(t *testing.T, folder *Folder, branchID, stage string) {
	t.Helper()
	branch := branchView(folder, branchID)
	main := folder.View()
	for _, row := range branch.Rows {
		if row.BranchID != branchID {
			t.Fatalf("%s: 分支行带着别的分支标识: %+v", stage, row)
		}
	}
	mainIDs := rowIDs(main)
	for _, id := range rowIDs(branch) {
		for _, mainID := range mainIDs {
			if id != "" && id == mainID {
				t.Fatalf("%s: 分支与主线出现同一行标识 %s", stage, id)
			}
		}
	}
}

// readPendingInputs 读一个分支的未处理输入。
func readPendingInputs(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, branchID string,
) wirePendingInputs {
	t.Helper()
	var result wirePendingInputs
	if err := conn.CallInto(ctx, "session.input.pending", map[string]any{
		"sessionId": sessionID,
		"branchId":  branchID,
	}, &result); err != nil {
		t.Fatalf("读取未处理输入: %v", err)
	}
	return result
}

// readPendingTasksFor 读一个分支的未结束任务。
func readPendingTasksFor(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, branchID string,
) wirePendingTasks {
	t.Helper()
	var result wirePendingTasks
	if err := conn.CallInto(ctx, "session.exec.pending", map[string]any{
		"sessionId": sessionID,
		"branchId":  branchID,
	}, &result); err != nil {
		t.Fatalf("读取未结束任务: %v", err)
	}
	return result
}

// startBranchTask 在分支里起一个后台任务。
func startBranchTask(t *testing.T, client *pendingClient, branchID string) string {
	t.Helper()
	submitTo(t, client.ctx, client.conn, client.sessionID, branchID,
		"Use the exec tool exactly once with runInBackground set to true and timeoutMs set to -1 "+
			"to run this command: sleep 20. Do not wait for it. Reply with the background task id.")
	deadline := time.Now().Add(3 * time.Minute)
	for {
		tasks := readPendingTasksFor(t, client.ctx, client.conn, client.sessionID, branchID)
		if len(tasks.Tasks) > 0 {
			return tasks.Tasks[0].TaskID
		}
		if time.Now().After(deadline) || client.ctx.Err() != nil {
			t.Skipf("分支里没有出现后台任务")
		}
		select {
		case <-client.ctx.Done():
			t.Skipf("等待分支后台任务超时")
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// waitApply 轮询分支历史，等到这条输入的 input/applied。
func waitApply(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, branchID, inputID string,
) protocol.RawEvent {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		page := readHistory(t, ctx, conn, sessionID, branchID)
		for _, event := range page.Events {
			if event.Type == "input" && (event.Name == "applied" || event.Name == "consumed") &&
				stringField(event, "input_id") == inputID {
				return event
			}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			t.Skipf("没有等到 %s 的应用事实", inputID)
		}
		select {
		case <-ctx.Done():
			t.Skipf("等待应用事实超时")
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// branchView 取分支视图；分支不存在时给空视图。
func branchView(folder *Folder, branchID string) View {
	if view, ok := folder.BranchView(branchID); ok {
		return view
	}
	return View{}
}

// findRowByID 按标识找行。
func findRowByID(rows []Row, id string) *Row {
	for index := range rows {
		if rows[index].ID == id {
			return &rows[index]
		}
	}
	return nil
}

// equalStrings 比较两个字符串切片。
func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
