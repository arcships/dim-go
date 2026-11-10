package projection

// 真实模型证据：后台任务、排队输入、提问这三类未完成事项，用真实 daemon +
// 真实模型 + /tmp 夹具账号跑一遍，验证服务端的截止点读取与本包的清单结算一致。
//
// 只用夹具的复制：来源目录（含凭据）只读，夹具本身不改；日志只记事件类型、
// 标识、序号与状态，不打印提示词、命令正文、模型输出或凭据。

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

// TestRealDaemonPendingInventories 在真实模型上验证三类未完成事项的读取与结算。
//
// 用法（夹具只存在 /tmp，不入库）：
//
//	DIM_REAL_MODEL_FIXTURE=/tmp/dimagent-real-model \
//	  go test -run TestRealDaemonPendingInventories -v -count=1 ./internal/projection/
func TestRealDaemonPendingInventories(t *testing.T) {
	fixture := os.Getenv("DIM_REAL_MODEL_FIXTURE")
	if fixture == "" {
		t.Skip("未设置 DIM_REAL_MODEL_FIXTURE：真实模型证据 blocked（缺 /tmp 夹具）")
	}
	serverPath := filepath.Join(fixture, "bin", "dimcode-server")
	if _, err := os.Stat(serverPath); err != nil {
		t.Skipf("夹具缺少 daemon 可执行文件: %v", err)
	}

	runDir, err := os.MkdirTemp("/tmp", "d2pending-")
	if err != nil {
		t.Fatalf("创建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runDir) })
	// 夹具只读：复制一份 home 再用，不改来源。
	home := filepath.Join(runDir, "home")
	copyTree(t, filepath.Join(fixture, "home"), home)
	for _, stale := range []string{"daemon.sock", "daemon.sock.lock", "daemon.start.lock"} {
		_ = os.Remove(filepath.Join(home, "rust", stale))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
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
	// 不传 tools：会话声明宿主全部工具（传空数组等于一个工具都不挂）。
	if err := conn.Client.CallInto(ctx, "session.create", map[string]any{
		"cwd": workDir,
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
			if event.Method != "session.event" {
				recorder.note(event.Method)
				return
			}
			var params protocol.SessionEventParams
			if err := json.Unmarshal(event.Params, &params); err != nil {
				return
			}
			if rejections := folder.ApplyListenerEvent(event); len(rejections) > 0 {
				t.Logf("实时折叠丢弃: %+v", rejections[0])
			}
			recorder.add(params.Event)
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

	backgroundSequence := evidenceBackgroundTask(t, client, recorder)
	queueSequence := evidenceQueuedInput(t, client, recorder)
	questionSequence := evidenceQuestion(t, client, recorder)

	t.Logf("证据: session=%s background={startedSeq:%d} queue={submitSeq:%d} question={callSeq:%d} events=%v",
		sessionID, backgroundSequence, queueSequence, questionSequence, recorder.kinds())
}

// pendingClient 是证据里用到的会话方法与截止点读取。
type pendingClient struct {
	t    *testing.T
	ctx  context.Context
	conn interface {
		CallInto(context.Context, string, any, any) error
	}
	sessionID string
	folder    *Folder
}

type wirePendingTasks struct {
	BranchID string            `json:"branchId"`
	Through  int64             `json:"through"`
	Tasks    []wirePendingTask `json:"tasks"`
	Next     json.RawMessage   `json:"next"`
}

type wirePendingTask struct {
	EventID   string `json:"eventId"`
	Sequence  int64  `json:"sequence"`
	StartedAt int64  `json:"startedAt"`
	TaskID    string `json:"taskId"`
	TurnID    string `json:"turnId"`
	RequestID string `json:"requestId"`
	CallID    string `json:"callId"`
}

type wirePendingInputs struct {
	BranchID string             `json:"branchId"`
	Through  int64              `json:"through"`
	Inputs   []wirePendingInput `json:"inputs"`
}

type wirePendingInput struct {
	InputID  string `json:"inputId"`
	Sequence int64  `json:"sequence"`
	Mode     string `json:"mode"`
	Applied  bool   `json:"applied"`
}

type wirePendingQuestions struct {
	BranchID string              `json:"branchId"`
	Through  int64               `json:"through"`
	Batches  []wireQuestionBatch `json:"batches"`
}

type wireQuestionBatch struct {
	EventID   string             `json:"eventId"`
	Sequence  int64              `json:"sequence"`
	RequestID string             `json:"requestId"`
	Calls     []wireQuestionCall `json:"calls"`
}

type wireQuestionCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func (c *pendingClient) pendingTasks() wirePendingTasks {
	var result wirePendingTasks
	if err := c.conn.CallInto(c.ctx, "session.exec.pending", map[string]any{
		"sessionId": c.sessionID,
		"branchId":  mainBranch,
	}, &result); err != nil {
		c.t.Fatalf("读取未结束后台任务: %v", err)
	}
	return result
}

func (c *pendingClient) pendingInputs() wirePendingInputs {
	var result wirePendingInputs
	if err := c.conn.CallInto(c.ctx, "session.input.pending", map[string]any{
		"sessionId": c.sessionID,
		"branchId":  mainBranch,
	}, &result); err != nil {
		c.t.Fatalf("读取未处理输入: %v", err)
	}
	return result
}

func (c *pendingClient) pendingQuestions() wirePendingQuestions {
	var result wirePendingQuestions
	if err := c.conn.CallInto(c.ctx, "session.question.pending", map[string]any{
		"sessionId": c.sessionID,
		"branchId":  mainBranch,
	}, &result); err != nil {
		c.t.Fatalf("读取待答提问: %v", err)
	}
	return result
}

func (c *pendingClient) submit(text, mode string) string {
	var result struct {
		Event protocol.RawEvent `json:"event"`
	}
	params := map[string]any{
		"sessionId":    c.sessionID,
		"branchId":     mainBranch,
		"submissionId": newUUID(c.t),
		"text":         text,
	}
	if mode != "" {
		params["mode"] = mode
	}
	if err := c.conn.CallInto(c.ctx, "session.submit", params, &result); err != nil {
		c.t.Fatalf("提交输入: %v", err)
	}
	return result.Event.ID
}

// ----- 证据一：未结束后台任务 -----

func evidenceBackgroundTask(t *testing.T, client *pendingClient, recorder *recorder) int64 {
	t.Helper()
	floor := recorder.head()
	client.submit("Use the exec tool exactly once with runInBackground set to true and "+
		"timeoutMs set to -1 to run this command: sleep 8. Do not wait for it. "+
		"Reply with the background task id.", "")

	started, ok := recorder.wait(client.ctx, 3*time.Minute, func(event protocol.RawEvent) bool {
		return event.Sequence > floor && event.Type == "exec" && event.Name == "started"
	})
	if !ok {
		t.Skipf("真实模型没有产生后台任务（观察到的事件类型：%v）", recorder.kinds())
	}
	t.Logf("后台任务开始: sequence=%d task=%s", started.Sequence, stringField(started, "task_id"))

	tasks := client.pendingTasks()
	if len(tasks.Tasks) == 0 {
		t.Fatalf("exec/started 之后 session.exec.pending 应列出未结束任务，实际为空")
	}
	for _, task := range tasks.Tasks {
		t.Logf("未结束任务: task=%s sequence=%d through=%d", task.TaskID, task.Sequence, tasks.Through)
	}
	taskID := stringField(started, "task_id")
	if !hasPendingTask(tasks, taskID) {
		t.Fatalf("未结束任务里没有刚启动的 %s", taskID)
	}
	// 用服务端清单装载本包清单：装载点之后的结束事实负责结算。
	client.folder.SeedPendingTasks(mainBranch, tasks.Through, projectionTasks(tasks))
	if got := len(client.folder.View().Pending.Tasks); got != 1 {
		t.Fatalf("按服务端清单装载后应有 1 个未结束任务，实际 %d", got)
	}

	ended, ok := recorder.wait(client.ctx, 3*time.Minute, func(event protocol.RawEvent) bool {
		return event.Sequence > started.Sequence && event.Type == "exec" && event.Name == "ended"
	})
	if !ok {
		t.Skipf("后台任务在等待时间内没有结束（观察到：%v）", recorder.kinds())
	}
	t.Logf("后台任务结束: sequence=%d task=%s", ended.Sequence, stringField(ended, "task_id"))

	if tasks := client.pendingTasks(); hasPendingTask(tasks, taskID) {
		t.Fatalf("任务结束后 session.exec.pending 仍列出 %s", taskID)
	}
	if got := len(client.folder.View().Pending.Tasks); got != 0 {
		t.Fatalf("任务结束后本包清单应结算为 0，实际 %d", got)
	}
	return started.Sequence
}

// ----- 证据二：排队输入 -----

func evidenceQueuedInput(t *testing.T, client *pendingClient, recorder *recorder) int64 {
	t.Helper()
	floor := recorder.head()
	client.submit("Use the exec tool exactly once to run this command in the foreground "+
		"(timeoutMs 30000): sleep 5. Then reply DONE.", "")

	if _, ok := recorder.wait(client.ctx, 3*time.Minute, func(event protocol.RawEvent) bool {
		return event.Sequence > floor && event.Type == "turn" && event.Name == "start"
	}); !ok {
		t.Skipf("真实模型没有开始一轮（观察到：%v）", recorder.kinds())
	}

	inputID := client.submit("Reply with just OK.", "queue")
	t.Logf("运行中提交排队输入: input=%s", inputID)

	inputs := client.pendingInputs()
	if !hasPendingInput(inputs, inputID) {
		t.Skipf("排队输入已被立即消费（未观察到未处理状态）：input=%s", inputID)
	}
	for _, item := range inputs.Inputs {
		t.Logf("未处理输入: input=%s sequence=%d mode=%s applied=%v",
			item.InputID, item.Sequence, item.Mode, item.Applied)
	}
	client.folder.SeedPendingInputs(mainBranch, inputs.Through, projectionInputs(inputs))
	if got := len(client.folder.View().Pending.Inputs); got != 1 {
		t.Fatalf("按服务端清单装载后应有 1 条未处理输入，实际 %d", got)
	}

	consumed, ok := recorder.wait(client.ctx, 3*time.Minute, func(event protocol.RawEvent) bool {
		return event.Type == "input" && event.Name == "consumed" && stringField(event, "input_id") == inputID
	})
	if !ok {
		t.Skipf("排队输入在等待时间内没有被消费（观察到：%v）", recorder.kinds())
	}
	t.Logf("排队输入被消费: sequence=%d input=%s", consumed.Sequence, inputID)

	if inputs := client.pendingInputs(); hasPendingInput(inputs, inputID) {
		t.Fatalf("消费后 session.input.pending 仍列出 %s", inputID)
	}
	if got := len(client.folder.View().Pending.Inputs); got != 0 {
		t.Fatalf("消费后本包清单应结算为 0，实际 %d", got)
	}
	return consumed.Sequence
}

// ----- 证据三：待答提问 -----

func evidenceQuestion(t *testing.T, client *pendingClient, recorder *recorder) int64 {
	t.Helper()
	floor := recorder.head()
	client.submit("Before answering anything, use the ask_user tool exactly once to ask me one "+
		"question: whether to continue. Offer exactly two options labeled yes and no. "+
		"Wait for my answer, then reply with the option I chose.", "")

	call, ok := recorder.wait(client.ctx, 3*time.Minute, func(event protocol.RawEvent) bool {
		return event.Sequence > floor && event.Type == "step" && event.Name == "tool-call" &&
			hasAskUserCall(event)
	})
	if !ok {
		t.Skipf("真实模型没有调用 ask_user（观察到的事件类型：%v）", recorder.kinds())
	}
	t.Logf("提问工具调用: sequence=%d", call.Sequence)

	questions := client.pendingQuestions()
	if len(questions.Batches) == 0 {
		t.Fatalf("ask_user 调用之后 session.question.pending 应列出待答提问，实际为空")
	}
	batch := questions.Batches[0]
	if len(batch.Calls) == 0 {
		t.Fatalf("待答提问批次里没有调用: %+v", batch)
	}
	t.Logf("待答提问: request=%s call=%s sequence=%d through=%d",
		batch.RequestID, batch.Calls[0].ID, batch.Sequence, questions.Through)
	client.folder.SeedPendingQuestions(mainBranch, questions.Through, projectionQuestions(questions))
	if got := len(client.folder.View().Pending.Questions); got != 1 {
		t.Fatalf("按服务端清单装载后应有 1 批待答提问，实际 %d", got)
	}

	answer, ok := firstAnswer(batch.Calls[0].Input)
	if !ok {
		t.Skipf("模型给出的提问没有可作答的选项，跳过应答")
	}
	if err := client.conn.CallInto(client.ctx, "session.question.respond", map[string]any{
		"sessionId": client.sessionID,
		"branchId":  mainBranch,
		"requestId": batch.RequestID,
		"callId":    batch.Calls[0].ID,
		"answers":   []any{map[string]any{"questionId": answer.questionID, "selected": []string{answer.label}}},
	}, nil); err != nil {
		t.Fatalf("应答提问: %v", err)
	}
	t.Logf("已应答提问: request=%s call=%s", batch.RequestID, batch.Calls[0].ID)

	result, ok := recorder.wait(client.ctx, 4*time.Minute, func(event protocol.RawEvent) bool {
		return event.Sequence > batch.Sequence && event.Type == "step" && event.Name == "tool-result" &&
			hasCallID(event, batch.Calls[0].ID)
	})
	if !ok {
		t.Skipf("提问没有被工具结果结算（观察到：%v）", recorder.kinds())
	}
	t.Logf("提问被工具结果结算: sequence=%d call=%s", result.Sequence, batch.Calls[0].ID)

	if questions := client.pendingQuestions(); len(questions.Batches) != 0 {
		t.Fatalf("结算后 session.question.pending 应为空，实际 %d 批", len(questions.Batches))
	}
	if got := len(client.folder.View().Pending.Questions); got != 0 {
		t.Fatalf("结算后本包清单应为 0，实际 %d", got)
	}
	return call.Sequence
}

type answerChoice struct {
	questionID string
	label      string
}

// firstAnswer 从 ask_user 的输入里取第一个问题与第一个选项标签。
func firstAnswer(input json.RawMessage) (answerChoice, bool) {
	var parsed struct {
		Questions []struct {
			ID      string `json:"id"`
			Options []struct {
				Label string `json:"label"`
			} `json:"options"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(input, &parsed); err != nil || len(parsed.Questions) == 0 {
		return answerChoice{}, false
	}
	question := parsed.Questions[0]
	if len(question.Options) == 0 {
		return answerChoice{}, false
	}
	id := question.ID
	if id == "" {
		id = "1"
	}
	return answerChoice{questionID: id, label: question.Options[0].Label}, true
}

// ----- 服务端清单到本包清单的映射 -----

func projectionTasks(tasks wirePendingTasks) []PendingTask {
	out := make([]PendingTask, 0, len(tasks.Tasks))
	for _, task := range tasks.Tasks {
		out = append(out, PendingTask{
			TaskID:    task.TaskID,
			EventID:   task.EventID,
			Sequence:  task.Sequence,
			StartedAt: task.StartedAt,
		})
	}
	return out
}

func projectionInputs(inputs wirePendingInputs) []PendingInput {
	out := make([]PendingInput, 0, len(inputs.Inputs))
	for _, item := range inputs.Inputs {
		out = append(out, PendingInput{
			InputID:  item.InputID,
			Sequence: item.Sequence,
			Mode:     item.Mode,
			Applied:  item.Applied,
		})
	}
	return out
}

func projectionQuestions(questions wirePendingQuestions) []PendingQuestion {
	out := make([]PendingQuestion, 0, len(questions.Batches))
	for _, batch := range questions.Batches {
		for _, call := range batch.Calls {
			out = append(out, PendingQuestion{
				RequestID: batch.RequestID,
				CallID:    call.ID,
				EventID:   batch.EventID,
				Sequence:  batch.Sequence,
				ToolName:  call.Name,
				Input:     call.Input,
			})
		}
	}
	return out
}

func hasPendingTask(tasks wirePendingTasks, taskID string) bool {
	if taskID == "" {
		return false
	}
	for _, task := range tasks.Tasks {
		if task.TaskID == taskID {
			return true
		}
	}
	return false
}

func hasPendingInput(inputs wirePendingInputs, inputID string) bool {
	for _, item := range inputs.Inputs {
		if item.InputID == inputID {
			return true
		}
	}
	return false
}

func hasAskUserCall(event protocol.RawEvent) bool {
	data := decodeObject(event.Data)
	for _, value := range array(data["calls"]) {
		if isQuestionTool(stringValue(object(value)["name"])) {
			return true
		}
	}
	data = decodeObject(event.Data)
	return isQuestionTool(stringValue(object(data["call"])["name"]))
}

func hasCallID(event protocol.RawEvent, callID string) bool {
	data := decodeObject(event.Data)
	for _, value := range array(data["results"]) {
		if firstString(object(value), "call_id", "callId") == callID {
			return true
		}
	}
	return firstString(data, "call_id", "callId") == callID
}

func stringField(event protocol.RawEvent, key string) string {
	return firstString(decodeObject(event.Data), key)
}

// recorder 按到达顺序收事件，供各段证据按条件等待。它只存事件本身，
// 日志里只用类型、标识、序号与状态。
type recorder struct {
	mu      sync.Mutex
	events  []protocol.RawEvent
	notices map[string]int
	signal  chan struct{}
}

func newRecorder() *recorder {
	return &recorder{notices: make(map[string]int), signal: make(chan struct{}, 1)}
}

func (r *recorder) add(event protocol.RawEvent) {
	r.mu.Lock()
	r.events = append(r.events, event)
	r.mu.Unlock()
	select {
	case r.signal <- struct{}{}:
	default:
	}
}

func (r *recorder) note(method string) {
	r.mu.Lock()
	r.notices[method]++
	r.mu.Unlock()
}

// head 返回当前已收事实的最大序号，各段证据用它划清先后。
func (r *recorder) head() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	var head int64
	for _, event := range r.events {
		if event.Sequence > head {
			head = event.Sequence
		}
	}
	return head
}

// wait 等一条满足条件的事实。超时返回 false，由调用方决定是跳过还是失败。
func (r *recorder) wait(
	ctx context.Context,
	timeout time.Duration,
	match func(protocol.RawEvent) bool,
) (protocol.RawEvent, bool) {
	deadline := time.Now().Add(timeout)
	for {
		r.mu.Lock()
		for _, event := range r.events {
			if match(event) {
				r.mu.Unlock()
				return event, true
			}
		}
		r.mu.Unlock()

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return protocol.RawEvent{}, false
		}
		wait := 200 * time.Millisecond
		if remaining < wait {
			wait = remaining
		}
		select {
		case <-r.signal:
		case <-ctx.Done():
			return protocol.RawEvent{}, false
		case <-time.After(wait):
		}
	}
}

// kinds 汇总出现过的事件类型与通知方法，作为"试过什么"的证据。
func (r *recorder) kinds() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := make(map[string]struct{})
	var out []string
	for _, event := range r.events {
		key := event.Type + "/" + event.Name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	for method := range r.notices {
		key := "notice:" + method
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

// copyTree 递归复制夹具目录，保留权限位。夹具只读：只从来源复制。
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, relative)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("复制夹具: %v", err)
	}
}
