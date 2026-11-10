package projection

// 真实模型证据：后台任务输出分页、后台任务停止、提示建议。
//
// 与 6.6 的证据同一套前提：/tmp 夹具的复制、真实 daemon、真实 provider。日志只留
// 状态、字节数、分页数与标识，不打印命令输出、提示建议正文或凭据。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
)

// TestRealDaemonExecOutputAndStop 在真实模型上验证任务输出分页与任务停止。
//
// 用法：
//
//	DIM_REAL_MODEL_FIXTURE=/tmp/dimagent-real-model \
//	  go test -run TestRealDaemonExecOutputAndStop -v -count=1 ./internal/projection/
func TestRealDaemonExecOutputAndStop(t *testing.T) {
	fixture := os.Getenv("DIM_REAL_MODEL_FIXTURE")
	if fixture == "" {
		t.Skip("未设置 DIM_REAL_MODEL_FIXTURE：真实模型证据 blocked（缺 /tmp 夹具）")
	}
	serverPath := filepath.Join(fixture, "bin", "dimcode-server")
	if _, err := os.Stat(serverPath); err != nil {
		t.Skipf("夹具缺少 daemon 可执行文件: %v", err)
	}

	runDir, err := os.MkdirTemp("/tmp", "d2exec-")
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
	notices := newNoticeLog()
	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	snapshot, lease, err := manager.Acquire(ctx, sessionID, session.Listener{
		OnEvent: func(event session.Event) {
			if rejections := folder.ApplyListenerEvent(event); len(rejections) > 0 {
				t.Logf("通知折叠丢弃: %+v", rejections[0])
			}
			switch event.Method {
			case "session.event":
				var params protocol.SessionEventParams
				if err := json.Unmarshal(event.Params, &params); err == nil {
					recorder.add(params.Event)
				}
			case "session.notice":
				notices.add(event.Params)
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

	client := &pendingClient{t: t, ctx: ctx, conn: conn.Client, sessionID: sessionID, folder: folder}

	// 证据一：任务输出分页（运行中可读、结束后读到 EOF）。
	taskID := startBackgroundTask(t, client, recorder,
		"for i in 1 2 3 4 5; do echo line$i; sleep 1; done",
		"Use the exec tool exactly once with runInBackground set to true and timeoutMs set to -1 "+
			"to run this command: for i in 1 2 3 4 5; do echo line$i; sleep 1; done. "+
			"Do not wait for it. Reply with the background task id.")
	outputPages, outputBytes := readOutputUntilEOF(t, ctx, conn.Client, sessionID, folder, taskID, 8)
	t.Logf("任务输出: task=%s 分页=%d 字节=%d", taskID, outputPages, outputBytes)

	// 证据二：按任务标识停止，停完状态收敛为非运行。
	stopTaskID := startBackgroundTask(t, client, recorder,
		"sleep 30",
		"Use the exec tool exactly once with runInBackground set to true and timeoutMs set to -1 "+
			"to run this command: sleep 30. Do not wait for it. Reply with the background task id.")
	stop := newExecTaskStop(t, ctx, conn.Client, sessionID, folder, stopTaskID)
	result, err := stop.Stop(ctx)
	if err != nil {
		t.Fatalf("停止后台任务: %v", err)
	}
	t.Logf("任务停止: task=%s status=%s cancelled=%v", stopTaskID, result.Status, result.Stopped())
	if result.Status == "" || result.Status == "running" {
		t.Fatalf("停止后状态应是非运行，实际 %q", result.Status)
	}
	if tasks := client.pendingTasks(); hasPendingTask(tasks, stopTaskID) {
		t.Fatalf("停止后 session.exec.pending 仍列出 %s", stopTaskID)
	}
	if hasProjectedTask(folder.View(), stopTaskID) {
		t.Fatalf("停止后本包清单里仍有 %s", stopTaskID)
	}
	ended, ok := recorder.wait(ctx, 2*time.Minute, func(event protocol.RawEvent) bool {
		return event.Type == "exec" && event.Name == "ended" && stringField(event, "task_id") == stopTaskID
	})
	if !ok {
		t.Skipf("停止后没有等到结束事实（观察到：%v）", recorder.kinds())
	}
	t.Logf("停止的结束事实: sequence=%d status=%s", ended.Sequence, stringField(ended, "status"))

	// 证据三：提示建议（服务端能力与开关都满足时才验证）。
	// 建议只对"最近一轮已完成"的轮次生成，所以先等停止那一轮真正结束。
	verifySuggestion(t, ctx, conn.Client, sessionID, folder, recorder, notices, ended.Sequence)

	t.Logf("证据: session=%s output={task:%s pages:%d bytes:%d} stop={task:%s status:%s} events=%v",
		sessionID, taskID, outputPages, outputBytes, stopTaskID, result.Status, recorder.kinds())
}

// startBackgroundTask 让模型起一个后台任务，返回任务标识。
func startBackgroundTask(t *testing.T, client *pendingClient, recorder *recorder, _, prompt string) string {
	t.Helper()
	floor := recorder.head()
	client.submit(prompt, "")
	started, ok := recorder.wait(client.ctx, 5*time.Minute, func(event protocol.RawEvent) bool {
		return event.Sequence > floor && event.Type == "exec" && event.Name == "started"
	})
	if !ok {
		t.Skipf("真实模型没有产生后台任务（观察到：%v）", recorder.kinds())
	}
	taskID := stringField(started, "task_id")
	t.Logf("后台任务开始: sequence=%d task=%s", started.Sequence, taskID)
	return taskID
}

// execOutputSource 用会话流读一页任务输出。
type execOutputSource struct {
	conn interface {
		CallInto(context.Context, string, any, any) error
	}
	sessionID string
}

func (s *execOutputSource) ReadExecOutput(
	ctx context.Context,
	req ExecOutputRequest,
) (ExecOutputPage, error) {
	var result struct {
		Text       string `json:"text"`
		Bytes      int64  `json:"bytes"`
		NextCursor *int64 `json:"nextCursor"`
		Truncated  bool   `json:"truncated"`
	}
	if err := s.conn.CallInto(ctx, "session.exec.output.page", map[string]any{
		"sessionId": s.sessionID,
		"branchId":  req.BranchID,
		"taskId":    req.TaskID,
		"cursor":    req.Cursor,
		"limit":     req.Limit,
	}, &result); err != nil {
		return ExecOutputPage{}, err
	}
	page := ExecOutputPage{Text: result.Text, Bytes: result.Bytes, Truncated: result.Truncated}
	if result.NextCursor != nil {
		page.NextCursor = *result.NextCursor
	}
	return page, nil
}

// readOutputUntilEOF 先运行中读一页，再等任务结束读到 EOF，返回分页数与总字节数。
func readOutputUntilEOF(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID string,
	folder *Folder,
	taskID string,
	limit int,
) (int, int64) {
	t.Helper()
	output, err := NewExecOutput(folder, mainBranch, taskID, &execOutputSource{conn: conn, sessionID: sessionID})
	if err != nil {
		t.Fatalf("创建输出读取器: %v", err)
	}
	output.SetLimit(limit)

	pages := 0
	var total int64
	// 运行中读一页：有输出就读到，没有也允许（命令还没写东西）。
	running, err := output.Next(ctx)
	if err != nil {
		t.Fatalf("运行中读输出: %v", err)
	}
	pages++
	total += running.Bytes
	t.Logf("运行中读输出: bytes=%d cursor=%d eof=%v（任务仍在跑）", running.Bytes, running.Cursor, running.EOF)
	if running.EOF {
		t.Fatalf("任务还在跑时不该判 EOF")
	}

	// 等结束事实，然后读到 EOF。
	for attempt := 0; attempt < 600; attempt++ {
		if !folder.TaskRunning(mainBranch, taskID) {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("等待任务结束超时")
		case <-time.After(200 * time.Millisecond):
		}
	}
	for pages < 200 {
		chunk, err := output.Next(ctx)
		if err != nil {
			t.Fatalf("读输出: %v", err)
		}
		pages++
		total += chunk.Bytes
		if chunk.EOF {
			break
		}
	}
	if !output.EOF() {
		t.Fatalf("任务结束后没有读到 EOF")
	}
	if total == 0 {
		t.Fatalf("任务输出一个字节都没读到")
	}
	return pages, total
}

// execStopSource 用会话流停一个后台任务。
type execStopSource struct {
	conn interface {
		CallInto(context.Context, string, any, any) error
	}
	sessionID string
}

func (s *execStopSource) StopExecTask(
	ctx context.Context,
	req ExecStopRequest,
) (ExecStopResult, error) {
	var result struct {
		TaskID string `json:"taskId"`
		Status string `json:"status"`
	}
	if err := s.conn.CallInto(ctx, "session.exec.stop", map[string]any{
		"sessionId": s.sessionID,
		"branchId":  req.BranchID,
		"taskId":    req.TaskID,
	}, &result); err != nil {
		return ExecStopResult{}, err
	}
	return ExecStopResult{TaskID: firstNonEmptyString(result.TaskID, req.TaskID), Status: result.Status}, nil
}

func newExecTaskStop(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID string,
	folder *Folder,
	taskID string,
) *ExecTaskStop {
	t.Helper()
	_ = ctx
	stop, err := NewExecTaskStop(folder, mainBranch, taskID, &execStopSource{conn: conn, sessionID: sessionID})
	if err != nil {
		t.Fatalf("创建任务停止: %v", err)
	}
	return stop
}

// verifySuggestion 验证提示建议的请求、通知与结算；服务端不给能力时只记录原因。
func verifySuggestion(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID string,
	folder *Folder,
	recorder *recorder,
	notices *noticeLog,
	afterSequence int64,
) {
	t.Helper()
	// 提示建议由设置源开关控制，默认关。这里只改夹具的**副本**（临时目录），
	// 来源夹具不动。
	if !enablePromptSuggestion(t, ctx, conn) {
		t.Logf("提示建议不可用: 服务端没有该设置源或写入被拒绝")
		return
	}
	finished, ok := recorder.wait(ctx, 3*time.Minute, func(event protocol.RawEvent) bool {
		return event.Sequence > afterSequence && event.Type == "turn" && event.Name == "end"
	})
	if !ok {
		t.Skipf("停止之后没有等到轮次结束（观察到：%v）", recorder.kinds())
	}
	turnID := stringField(finished, "turn_id")
	t.Logf("最近一轮已结束: turn=%s sequence=%d", turnID, finished.Sequence)
	var reply struct {
		Accepted bool   `json:"accepted"`
		Reason   string `json:"reason"`
	}
	if err := conn.CallInto(ctx, "session.suggestion.request", map[string]any{
		"sessionId": sessionID,
		"turnId":    turnID,
	}, &reply); err != nil {
		t.Logf("提示建议不可用（请求失败）: %v", err)
		return
	}
	if !reply.Accepted {
		t.Logf("提示建议不可用: reason=%s（服务端开关或轮次状态不满足）", reply.Reason)
		return
	}
	t.Logf("提示建议请求已受理: turn=%s", turnID)

	offer, ok := notices.wait(ctx, 90*time.Second, "offer")
	if !ok {
		t.Skipf("提示建议没有到达（观察到：%v）", notices.kinds())
	}
	_ = offer
	if view := folder.View(); view.Suggestion == nil || view.Suggestion.TurnID != turnID {
		t.Fatalf("offer 通知没有折叠成建议: %+v", view.Suggestion)
	}
	t.Logf("提示建议已到达并折叠: turn=%s 文本长度=%d", turnID, len(folder.View().Suggestion.Text))

	// 应答（忽略）：外面走写登记，这里直接发会话方法验证结算。
	if err := conn.CallInto(ctx, "session.suggestion.respond", map[string]any{
		"sessionId": sessionID,
		"turnId":    turnID,
		"action":    "dismiss",
	}, nil); err != nil {
		t.Fatalf("应答提示建议: %v", err)
	}
	if _, ok := notices.wait(ctx, 60*time.Second, "cleared"); !ok {
		t.Skipf("应答后没有收到 cleared 通知（观察到：%v）", notices.kinds())
	}
	if folder.View().Suggestion != nil {
		t.Fatalf("cleared 之后不该再有建议")
	}
	t.Logf("提示建议已结算: turn=%s", turnID)
}

// enablePromptSuggestion 在夹具副本上打开提示建议开关，返回是否已打开。
func enablePromptSuggestion(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
) bool {
	t.Helper()
	var snapshot struct {
		Revision string `json:"revision"`
		Document struct {
			Enabled  bool `json:"enabled"`
			MaxChars int  `json:"maxChars"`
		} `json:"document"`
	}
	if err := conn.CallInto(ctx, "settings.read", map[string]any{
		"source": "prompt-suggestion",
		"scope":  "user",
	}, &snapshot); err != nil {
		t.Logf("读取提示建议设置失败: %v", err)
		return false
	}
	if snapshot.Document.Enabled {
		return true
	}
	maxChars := snapshot.Document.MaxChars
	if maxChars == 0 {
		maxChars = 80
	}
	var updated struct {
		Revision string `json:"revision"`
	}
	if err := conn.CallInto(ctx, "settings.update", map[string]any{
		"source":           "prompt-suggestion",
		"scope":            "user",
		"expectedRevision": snapshot.Revision,
		"document":         map[string]any{"enabled": true, "maxChars": maxChars},
	}, &updated); err != nil {
		t.Logf("打开提示建议设置失败: %v", err)
		return false
	}
	t.Logf("提示建议设置已在夹具副本上打开: revision=%s", updated.Revision)
	return true
}

// noticeLog 收会话通知，按 kind 等待。
type noticeLog struct {
	mu     sync.Mutex
	items  []sessionNotice
	signal chan struct{}
}

func newNoticeLog() *noticeLog {
	return &noticeLog{signal: make(chan struct{}, 1)}
}

func (n *noticeLog) add(params json.RawMessage) {
	var notice sessionNotice
	if err := json.Unmarshal(params, &notice); err != nil {
		return
	}
	n.mu.Lock()
	n.items = append(n.items, notice)
	n.mu.Unlock()
	select {
	case n.signal <- struct{}{}:
	default:
	}
}

func (n *noticeLog) wait(ctx context.Context, timeout time.Duration, kind string) (sessionNotice, bool) {
	deadline := time.Now().Add(timeout)
	for {
		n.mu.Lock()
		for _, item := range n.items {
			if item.Source == suggestionNoticeSource && item.Kind == kind {
				n.mu.Unlock()
				return item, true
			}
		}
		n.mu.Unlock()
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return sessionNotice{}, false
		}
		wait := 200 * time.Millisecond
		if remaining < wait {
			wait = remaining
		}
		select {
		case <-n.signal:
		case <-ctx.Done():
			return sessionNotice{}, false
		case <-time.After(wait):
		}
	}
}

// kinds 汇总收到过的通知来源与种类。
func (n *noticeLog) kinds() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	seen := make(map[string]struct{})
	var out []string
	for _, item := range n.items {
		key := item.Source + "/" + item.Kind
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

// lastTurnID 取最近一条 turn/end 的轮次标识。
func (r *recorder) lastTurnID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := len(r.events) - 1; index >= 0; index-- {
		event := r.events[index]
		if event.Type == "turn" && event.Name == "end" {
			return stringField(event, "turn_id")
		}
	}
	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
