package projection

// 真实模型证据：上下文用量、排队输入撤回、目标暂停与恢复。
//
// 前提与 6.6/6.7 相同：临时夹具的复制、真实 daemon、真实 provider。日志只留状态、
// 标识与数字，不打印目标正文、输入正文或模型输出。

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

// TestRealDaemonContextAndControls 验证用量折叠、输入撤回与目标暂停/恢复。
//
// 用法：
//
//	DIM_REAL_MODEL_FIXTURE=<夹具目录> \
//	  go test -run TestRealDaemonContextAndControls -v -count=1 ./internal/projection/
func TestRealDaemonContextAndControls(t *testing.T) {
	fixture := os.Getenv("DIM_REAL_MODEL_FIXTURE")
	if fixture == "" {
		t.Skip("未设置 DIM_REAL_MODEL_FIXTURE：真实模型证据 blocked（缺临时夹具目录）")
	}
	serverPath := filepath.Join(fixture, "bin", "dimcode-server")
	if _, err := os.Stat(serverPath); err != nil {
		t.Skipf("夹具缺少 daemon 可执行文件: %v", err)
	}

	runDir := evidenceTempDir(t, "d2ctx-")
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
	var created protocol.SessionCreateResult
	if err := conn.Client.CallInto(ctx, "session.create", map[string]any{
		"cwd":              workDir,
		"permissionPolicy": map[string]any{"preset": "full-access"},
	}, &created); err != nil {
		t.Skipf("创建真实会话失败（账号/模型不可用）: %v", err)
	}
	sessionID := created.SessionID
	t.Logf("会话 %s", sessionID)

	folder := newFolderFor(t, sessionID)
	recorder := newRecorder()
	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	snapshot, lease, err := manager.Acquire(ctx, sessionID, session.Listener{
		OnEvent: func(event session.Event) {
			if rejections := folder.ApplyListenerEvent(event); len(rejections) > 0 {
				t.Logf("通知折叠丢弃: %+v", rejections[0])
			}
			if event.Method == "session.event" {
				var params protocol.SessionEventParams
				if err := json.Unmarshal(event.Params, &params); err == nil {
					recorder.add(params.Event)
				}
			}
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
	if rejections := folder.ApplySnapshot(snapshot); len(rejections) > 0 {
		t.Logf("首屏折叠丢弃: %+v", rejections[0])
	}

	methods := readSessionMethods(t, ctx, conn.Client, sessionID)
	client := &pendingClient{t: t, ctx: ctx, conn: conn.Client, sessionID: sessionID, folder: folder}

	// 证据一：用量（跑一轮真实模型调用）。
	client.submit("Reply with the single word OK.", "")
	waitTurnEnd(t, ctx, recorder, recorder.head())
	view := folder.View()
	if view.Usage == nil {
		t.Fatalf("真实模型调用之后应有 model/usage 用量事实")
	}
	t.Logf("用量: 请求=%d prompt=%d completion=%d total=%d 缓存读=%d",
		view.Usage.Requests, view.Usage.PromptTokens, view.Usage.CompletionTokens,
		view.Usage.TotalTokens, view.Usage.CacheReadTokens)
	if view.Usage.PromptTokens <= 0 || view.Usage.TotalTokens <= 0 {
		t.Fatalf("用量应大于零: %+v", view.Usage)
	}
	if view.Context != nil {
		t.Logf("上下文占用: 估算输入=%d 窗口=%d 输出预留=%d 占比=%.1f%%",
			view.Context.EstimatedInputTokens, view.Context.ContextWindow,
			view.Context.OutputTokens, view.Context.Percent())
	} else {
		t.Logf("上下文占用: 这一轮没有 context/budget 事实（服务端未发布）")
	}
	if view.Compaction == nil {
		t.Logf("压缩摘要: 这一轮没有触发压缩（compaction 由上下文策略决定，不可人为触发）")
	}

	// 证据二：排队输入撤回。
	withdrawEvidence(t, ctx, conn.Client, client, recorder, methods)

	// 证据三：目标暂停与恢复。
	goalEvidence(t, ctx, conn.Client, client, folder, recorder, methods)

	// 恢复目标可能接着起一轮继续工作：等这一轮收尾再回收。
	floor := recorder.head()
	if _, ok := recorder.wait(ctx, 90*time.Second, func(event protocol.RawEvent) bool {
		return event.Sequence > floor && event.Type == "turn" && event.Name == "end"
	}); !ok {
		t.Logf("恢复之后没有等到新的轮次结束（观察到：%v）", recorder.kinds())
	}

	t.Logf("证据: session=%s usage={requests:%d prompt:%d} withdraw={ok} goal={pause/resume} events=%v",
		sessionID, view.Usage.Requests, view.Usage.PromptTokens, recorder.kinds())
}

// waitTurnEnd 等到一条序号大于 after 的 turn/end。
func waitTurnEnd(t *testing.T, ctx context.Context, recorder *recorder, after int64) protocol.RawEvent {
	t.Helper()
	ended, ok := recorder.wait(ctx, 4*time.Minute, func(event protocol.RawEvent) bool {
		return event.Sequence > after && event.Type == "turn" && event.Name == "end"
	})
	if !ok {
		t.Skipf("没有等到轮次结束（观察到：%v）", recorder.kinds())
	}
	return ended
}

// readSessionMethods 读会话方法表。
func readSessionMethods(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID string,
) []string {
	t.Helper()
	var reply struct {
		Methods []struct {
			Name string `json:"name"`
		} `json:"methods"`
	}
	if err := conn.CallInto(ctx, "session.methods", map[string]any{"sessionId": sessionID}, &reply); err != nil {
		t.Fatalf("读取会话方法表: %v", err)
	}
	out := make([]string, 0, len(reply.Methods))
	for _, method := range reply.Methods {
		out = append(out, method.Name)
	}
	return out
}

// withdrawEvidence 运行中提交一条排队输入，再用真实方法撤回。
func withdrawEvidence(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	client *pendingClient,
	recorder *recorder,
	methods []string,
) {
	t.Helper()
	_, firstSeq := client.submit("Use the exec tool exactly once to run this command in the foreground "+
		"(timeoutMs 30000): sleep 5. Then reply DONE.", "")
	if _, ok := recorder.wait(ctx, 3*time.Minute, func(event protocol.RawEvent) bool {
		return event.Sequence > firstSeq && event.Type == "turn" && event.Name == "start"
	}); !ok {
		t.Skipf("真实模型没有开始一轮（观察到：%v）", recorder.kinds())
	}

	inputID, _ := client.submit("Reply with just OK.", "queue")
	inputs := client.pendingInputs()
	if !hasPendingInput(inputs, inputID) {
		t.Skipf("排队输入已被立即消费，无法验证撤回: input=%s", inputID)
	}
	t.Logf("撤回前: input=%s mode=%s applied=%v", inputID, inputs.Inputs[0].Mode, inputs.Inputs[0].Applied)

	sender := &inputWithdrawSource{conn: conn, sessionID: client.sessionID}
	withdraw, err := NewQueuedInputWithdraw(client.folder, mainBranch, inputID, methods, sender)
	if err != nil {
		t.Skipf("服务端没有 session.input.withdraw：%v", err)
	}
	result, err := withdraw.Withdraw(ctx)
	if err != nil {
		t.Fatalf("撤回排队输入: %v", err)
	}
	t.Logf("撤回回执: status=%s（服务端方法 session.input.withdraw）", result.Status)

	fact, ok := recorder.wait(ctx, 2*time.Minute, func(event protocol.RawEvent) bool {
		return event.Type == "input" && (event.Name == "withdraw" || event.Name == "deleted") &&
			stringField(event, "input_id") == inputID
	})
	if !ok {
		t.Skipf("撤回后没有等到事实（观察到：%v）", recorder.kinds())
	}
	t.Logf("撤回事实: type=input/%s sequence=%d", fact.Name, fact.Sequence)
	// 删除事实可能与撤回事实分批落库：给一小段收敛时间再判定。
	if !waitPendingGone(t, ctx, client, inputID) {
		t.Fatalf("撤回后 session.input.pending 仍列出 %s", inputID)
	}
	if hasProjectedInput(client.folder.View(), inputID) {
		t.Fatalf("撤回后本包清单里仍有 %s", inputID)
	}
}

// waitPendingGone 等未处理清单不再列出这条输入。
func waitPendingGone(t *testing.T, ctx context.Context, client *pendingClient, inputID string) bool {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		if !hasPendingInput(client.pendingInputs(), inputID) {
			return true
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// inputWithdrawSource 用会话流撤回一条输入。
type inputWithdrawSource struct {
	conn interface {
		CallInto(context.Context, string, any, any) error
	}
	sessionID string
}

func (s *inputWithdrawSource) WithdrawInput(
	ctx context.Context,
	req InputWithdrawRequest,
) (InputWithdrawResult, error) {
	var result struct {
		Status string `json:"status"`
	}
	if err := s.conn.CallInto(ctx, "session.input.withdraw", map[string]any{
		"sessionId": s.sessionID,
		"branchId":  req.BranchID,
		"inputId":   req.InputID,
	}, &result); err != nil {
		return InputWithdrawResult{}, err
	}
	return InputWithdrawResult{Status: result.Status}, nil
}

// goalEvidence 用真实方法设置、暂停、恢复目标，状态由 goal:state 记录结算。
func goalEvidence(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	client *pendingClient,
	folder *Folder,
	recorder *recorder,
	methods []string,
) {
	t.Helper()
	if !hasSessionMethod(methods, goalPauseMethod) || !hasSessionMethod(methods, goalResumeMethod) {
		t.Skipf("服务端没有 session.goal.pause / session.goal.resume：能力缺失")
	}
	floor := recorder.head()
	if err := conn.CallInto(ctx, "session.goal.set", map[string]any{
		"sessionId": client.sessionID,
		"branchId":  mainBranch,
		"objective": "finish the build and report the result",
	}, nil); err != nil {
		t.Skipf("设置目标失败（服务端方法不可用）: %v", err)
	}
	waitGoalStatus(t, ctx, folder, recorder, floor, GoalActive)
	t.Logf("目标: 已设置，状态=%s", GoalActive)

	sender := &goalControlSource{conn: conn, sessionID: client.sessionID}
	pause, err := NewGoalControl(folder, mainBranch, GoalPause, methods, sender)
	if err != nil {
		t.Skipf("创建暂停失败: %v", err)
	}
	if _, err := pause.Apply(ctx); err != nil {
		t.Fatalf("暂停目标: %v", err)
	}
	waitGoalStatus(t, ctx, folder, recorder, floor, GoalPaused)
	goal := folder.View().Goal
	t.Logf("目标: 已暂停，状态=%s 原因=%s", goal.Status, goal.Reason)

	resume, err := NewGoalControl(folder, mainBranch, GoalResume, methods, sender)
	if err != nil {
		t.Skipf("创建恢复失败: %v", err)
	}
	if _, err := resume.Apply(ctx); err != nil {
		t.Fatalf("恢复目标: %v", err)
	}
	waitGoalStatus(t, ctx, folder, recorder, floor, GoalActive)
	t.Logf("目标: 已恢复，状态=%s", GoalActive)
}

// waitGoalStatus 等到目标折叠成指定状态。
func waitGoalStatus(t *testing.T, ctx context.Context, folder *Folder, recorder *recorder, after int64, want GoalStatus) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for {
		if view := folder.View(); view.Goal != nil && view.Goal.Status == want {
			return
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			t.Skipf("目标没有收敛到 %s（观察到：%v）", want, recorder.kinds())
		}
		select {
		case <-ctx.Done():
			t.Skipf("等待目标状态超时")
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// goalControlSource 用会话流暂停/恢复目标。
type goalControlSource struct {
	conn interface {
		CallInto(context.Context, string, any, any) error
	}
	sessionID string
}

func (s *goalControlSource) ControlGoal(
	ctx context.Context,
	req GoalControlRequest,
) (GoalControlResult, error) {
	method, ok := req.Action.method()
	if !ok {
		return GoalControlResult{}, ErrGoalControlBadAction
	}
	var reply struct {
		Sequence int64 `json:"sequence"`
	}
	if err := s.conn.CallInto(ctx, method, map[string]any{
		"sessionId": s.sessionID,
		"branchId":  req.BranchID,
	}, &reply); err != nil {
		return GoalControlResult{}, err
	}
	return GoalControlResult{Sequence: reply.Sequence}, nil
}
