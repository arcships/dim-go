package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// fakeModalitySession 是会话模态面的测试替身：记录门禁、读取与发送调用。
type fakeModalitySession struct {
	mu         sync.Mutex
	supports   map[string]bool
	supportErr error
	gateCalls  []string

	task      ModalityTask
	taskErr   error
	readCalls []string

	page      ModalityTaskPage
	pageErr   error
	lastQuery ModalityTaskQuery

	result      ModalityTaskResultPage
	resultErr   error
	lastRQuery  ModalityTaskResultQuery
	resultCalls int

	sentFrames []ModalityRealtimeFrame
	sendErr    error
	sendCalls  int
}

func (f *fakeModalitySession) SupportsModalityMethod(_ context.Context, _ string, method string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gateCalls = append(f.gateCalls, method)
	if f.supportErr != nil {
		return false, f.supportErr
	}
	return f.supports[method], nil
}

func (f *fakeModalitySession) ReadModalityTask(_ context.Context, _ string, taskID string) (ModalityTask, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.readCalls = append(f.readCalls, taskID)
	if f.taskErr != nil {
		return ModalityTask{}, f.taskErr
	}
	return f.task, nil
}

func (f *fakeModalitySession) ListModalityTasks(_ context.Context, _ string, query ModalityTaskQuery) (ModalityTaskPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastQuery = query
	if f.pageErr != nil {
		return ModalityTaskPage{}, f.pageErr
	}
	return f.page, nil
}

func (f *fakeModalitySession) ReadModalityTaskResult(_ context.Context, _ string, query ModalityTaskResultQuery) (ModalityTaskResultPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resultCalls++
	f.lastRQuery = query
	if f.resultErr != nil {
		return ModalityTaskResultPage{}, f.resultErr
	}
	return f.result, nil
}

func (f *fakeModalitySession) SendModalityRealtime(_ context.Context, _, _ string, frame ModalityRealtimeFrame) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendCalls++
	f.sentFrames = append(f.sentFrames, frame)
	return f.sendErr
}

func (f *fakeModalitySession) gateCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.gateCalls)
}

func (f *fakeModalitySession) readCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.readCalls)
}

func (f *fakeModalitySession) sentCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sentFrames)
}

func taskFixture(id string, state ModalityTaskState) ModalityTask {
	return ModalityTask{
		ID:        id,
		Category:  "image",
		Operation: "generate",
		State:     state,
		Revision:  1,
		CreatedAt: "2026-11-10T00:00:00Z",
		UpdatedAt: "2026-11-10T00:00:01Z",
		Result: &ModalityTaskResultData{
			Files: []ModalityTaskFile{
				{Name: "out.png", MediaType: "image/png", Bytes: 10, SHA256: "abc"},
			},
		},
	}
}

func modalityOpSender(t *testing.T, replies map[string]func() (write.Reply, error)) *senderFunc {
	t.Helper()
	return &senderFunc{fn: func(_ context.Context, req write.Request) (write.Reply, error) {
		var payload struct {
			Op string `json:"op"`
		}
		if err := json.Unmarshal(req.Params, &payload); err != nil {
			t.Fatalf("解析模态写参数: %v", err)
		}
		handler, ok := replies[payload.Op]
		if !ok {
			t.Fatalf("测试没有准备 %s 的应答", payload.Op)
		}
		return handler()
	}}
}

func newModalityManager(t *testing.T, session ModalitySession, writer *write.Manager) *Manager {
	t.Helper()
	return mustNewManager(t, Config{
		Features:        allFeatures(),
		Modality:        &stubModality{},
		ModalitySession: session,
		Writer:          writer,
	})
}

func TestSubmitTaskWritesOnceAndReadsBack(t *testing.T) {
	session := &fakeModalitySession{
		supports: map[string]bool{
			ModalityMethodTaskSubmit: true,
			ModalityMethodTaskStatus: true,
		},
		task: taskFixture("task-1", ModalityTaskQueued),
	}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"task.submit": func() (write.Reply, error) { return write.Reply{InputID: "task-1"}, nil },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))

	task, err := manager.SubmitTask(context.Background(), ModalityTaskSubmitRequest{
		SessionID:    "s1",
		Category:     "image",
		Operation:    "generate",
		Params:       json.RawMessage(`{"prompt":"猫"}`),
		Files:        []ModalityTaskFileInput{{Name: "in.png", MediaType: "image/png", DataBase64: "aGk="}},
		Model:        &ModalityModelRef{ProviderID: "p1", ModelID: "m1"},
		Node:         "node-a",
		ClientTaskID: "client-1",
	})
	if err != nil {
		t.Fatalf("提交任务: %v", err)
	}
	if task.ID != "task-1" || task.State != ModalityTaskQueued || task.SessionID != "s1" {
		t.Fatalf("提交后任务不符：%+v", task)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatalf("提交应只发一次：%d", len(sender.snapshot()))
	}
	if sender.snapshot()[0].Kind != write.KindModalityWrite {
		t.Fatalf("提交应经模态写入类型：%q", sender.snapshot()[0].Kind)
	}
	var params modalityTaskSubmitParams
	if err := json.Unmarshal(sender.snapshot()[0].Params, &params); err != nil {
		t.Fatalf("解析提交参数: %v", err)
	}
	if params.Op != "task.submit" || params.SessionID != "s1" ||
		params.Category != "image" || params.Operation != "generate" ||
		params.ClientTaskID != "client-1" || params.Node != "node-a" {
		t.Fatalf("提交参数不符：%+v", params)
	}
	if params.Model == nil || params.Model.ProviderID != "p1" || params.Model.ModelID != "m1" {
		t.Fatalf("提交模型引用不符：%+v", params.Model)
	}
	if len(params.Files) != 1 || params.Files[0].Name != "in.png" || params.Files[0].DataBase64 != "aGk=" {
		t.Fatalf("提交输入文件不符：%+v", params.Files)
	}
	if session.readCount() != 1 {
		t.Fatalf("提交后应重读一次任务：%d", session.readCount())
	}
	cached, ok := manager.Task("s1", "task-1")
	if !ok || cached.ID != "task-1" {
		t.Fatalf("任务没有回填缓存：%+v", cached)
	}
}

func TestSubmitTaskDuplicateClientIDIsSentOnce(t *testing.T) {
	session := &fakeModalitySession{
		supports: map[string]bool{
			ModalityMethodTaskSubmit: true,
			ModalityMethodTaskStatus: true,
		},
		task: taskFixture("task-1", ModalityTaskQueued),
	}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"task.submit": func() (write.Reply, error) { return write.Reply{InputID: "task-1"}, nil },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))
	request := ModalityTaskSubmitRequest{
		SessionID: "s1", Category: "image", Operation: "generate", ClientTaskID: "client-1",
	}
	if _, err := manager.SubmitTask(context.Background(), request); err != nil {
		t.Fatalf("第一次提交: %v", err)
	}
	if _, err := manager.SubmitTask(context.Background(), request); err != nil {
		t.Fatalf("第二次提交: %v", err)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatalf("同一客户端任务标识只应发一次：%d", len(sender.snapshot()))
	}
}

func TestSubmitTaskFailureKeepsReason(t *testing.T) {
	session := &fakeModalitySession{
		supports: map[string]bool{
			ModalityMethodTaskSubmit: true,
			ModalityMethodTaskStatus: true,
		},
	}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"task.submit": func() (write.Reply, error) { return write.Reply{}, errors.New("任务被拒绝") },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))

	_, err := manager.SubmitTask(context.Background(), ModalityTaskSubmitRequest{
		SessionID: "s1", Category: "image", Operation: "generate",
	})
	if !errors.Is(err, ErrTaskSubmitFailed) {
		t.Fatalf("提交失败应分类：%v", err)
	}
	if !strings.Contains(err.Error(), "任务被拒绝") {
		t.Fatalf("失败原因应保留：%v", err)
	}
	if session.readCount() != 0 {
		t.Fatal("提交失败不应读取任务")
	}
	if len(sender.snapshot()) != 1 {
		t.Fatal("提交失败不应重发")
	}
}

func TestSubmitTaskCapabilityGate(t *testing.T) {
	cases := []struct {
		name     string
		supports map[string]bool
		err      error
	}{
		{"缺提交方法", map[string]bool{ModalityMethodTaskStatus: true}, nil},
		{"缺状态方法", map[string]bool{ModalityMethodTaskSubmit: true}, nil},
		{"方法表读取失败", map[string]bool{ModalityMethodTaskSubmit: true, ModalityMethodTaskStatus: true}, errors.New("方法表不可读")},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			session := &fakeModalitySession{supports: testCase.supports, supportErr: testCase.err}
			sender := &senderFunc{}
			manager := newModalityManager(t, session, newTestWriter(t, sender))
			_, err := manager.SubmitTask(context.Background(), ModalityTaskSubmitRequest{
				SessionID: "s1", Category: "image", Operation: "generate",
			})
			if !errors.Is(err, ErrCapabilityMissing) {
				t.Fatalf("缺方法应返回能力缺失：%v", err)
			}
			if len(sender.snapshot()) != 0 {
				t.Fatal("缺方法时不应发出写请求")
			}
			if session.readCount() != 0 {
				t.Fatal("缺方法时不应读取任务")
			}
		})
	}
}

func TestCancelTaskWritesOnceAndReadsBack(t *testing.T) {
	session := &fakeModalitySession{
		supports: map[string]bool{
			ModalityMethodTaskCancel: true,
			ModalityMethodTaskStatus: true,
		},
		task: taskFixture("task-1", ModalityTaskCancelled),
	}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"task.cancel": func() (write.Reply, error) { return write.Reply{InputID: "task-1"}, nil },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))

	task, err := manager.CancelTask(context.Background(), "s1", "task-1")
	if err != nil {
		t.Fatalf("取消任务: %v", err)
	}
	if task.State != ModalityTaskCancelled {
		t.Fatalf("取消后状态不符：%+v", task)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatalf("取消应只发一次：%d", len(sender.snapshot()))
	}
	request := sender.snapshot()[0]
	if request.Kind != write.KindModalityWrite || request.InputID != "task-1" {
		t.Fatalf("取消请求不符：%+v", request)
	}
	var params modalityTaskCancelParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		t.Fatalf("解析取消参数: %v", err)
	}
	if params.Op != "task.cancel" || params.SessionID != "s1" || params.TaskID != "task-1" {
		t.Fatalf("取消参数不符：%+v", params)
	}
	if session.readCount() != 1 {
		t.Fatalf("取消后应重读一次任务：%d", session.readCount())
	}
}

func TestCancelTaskFailureKeepsReason(t *testing.T) {
	session := &fakeModalitySession{
		supports: map[string]bool{
			ModalityMethodTaskCancel: true,
			ModalityMethodTaskStatus: true,
		},
	}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"task.cancel": func() (write.Reply, error) { return write.Reply{}, errors.New("任务不可取消") },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))
	_, err := manager.CancelTask(context.Background(), "s1", "task-1")
	if !errors.Is(err, ErrTaskCancelFailed) {
		t.Fatalf("取消失败应分类：%v", err)
	}
	if !strings.Contains(err.Error(), "任务不可取消") {
		t.Fatalf("取消失败原因应保留：%v", err)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatal("取消失败不应重发")
	}
}

func TestListTasksPaginates(t *testing.T) {
	session := &fakeModalitySession{
		supports: map[string]bool{ModalityMethodTaskList: true},
		page:     ModalityTaskPage{Tasks: []ModalityTask{taskFixture("task-1", ModalityTaskQueued)}, Next: "task-1"},
	}
	manager := newModalityManager(t, session, nil)

	page, err := manager.ListTasks(context.Background(), "s1", ModalityTaskQuery{
		State: ModalityTaskQueued, Category: "image", Limit: 1,
	})
	if err != nil {
		t.Fatalf("读取任务清单: %v", err)
	}
	if len(page.Tasks) != 1 || page.Next != "task-1" || page.Tasks[0].SessionID != "s1" {
		t.Fatalf("任务清单不符：%+v", page)
	}
	if session.lastQuery.State != ModalityTaskQueued || session.lastQuery.Limit != 1 {
		t.Fatalf("任务查询不符：%+v", session.lastQuery)
	}

	session.page = ModalityTaskPage{Tasks: []ModalityTask{taskFixture("task-2", ModalityTaskSucceeded)}}
	page, err = manager.ListTasks(context.Background(), "s1", ModalityTaskQuery{After: "task-1"})
	if err != nil {
		t.Fatalf("按游标读取任务清单: %v", err)
	}
	if page.Next != "" || session.lastQuery.After != "task-1" {
		t.Fatalf("第二页不符：page=%+v query=%+v", page, session.lastQuery)
	}
	if cached, ok := manager.Task("s1", "task-2"); !ok || cached.State != ModalityTaskSucceeded {
		t.Fatalf("清单应回填任务缓存：%+v", cached)
	}
}

func TestListTasksRejectsInvalidQueryWithoutCall(t *testing.T) {
	session := &fakeModalitySession{supports: map[string]bool{ModalityMethodTaskList: true}}
	manager := newModalityManager(t, session, nil)
	for _, query := range []ModalityTaskQuery{
		{State: "done"},
		{Limit: 101},
		{Limit: -1},
	} {
		if _, err := manager.ListTasks(context.Background(), "s1", query); !errors.Is(err, ErrInvalidModalityRequest) {
			t.Fatalf("不合法查询应拒绝：%+v -> %v", query, err)
		}
	}
	if session.gateCount() != 0 {
		t.Fatalf("结构不合法时不应读会话方法表：%d", session.gateCount())
	}
}

func TestReadTaskResultPaginates(t *testing.T) {
	session := &fakeModalitySession{
		supports: map[string]bool{ModalityMethodTaskResult: true},
		result:   ModalityTaskResultPage{DataBase64: "aGk=", EOF: false},
	}
	manager := newModalityManager(t, session, nil)

	page, err := manager.ReadTaskResult(context.Background(), "s1", ModalityTaskResultQuery{
		TaskID: "task-1", Name: "out.png", Offset: 0, Length: 4,
	})
	if err != nil {
		t.Fatalf("读取任务结果: %v", err)
	}
	if page.DataBase64 != "aGk=" || page.EOF {
		t.Fatalf("结果页不符：%+v", page)
	}
	if session.lastRQuery.Offset != 0 || session.lastRQuery.Length != 4 || session.lastRQuery.Name != "out.png" {
		t.Fatalf("结果查询不符：%+v", session.lastRQuery)
	}

	session.result = ModalityTaskResultPage{DataBase64: "", EOF: true}
	page, err = manager.ReadTaskResult(context.Background(), "s1", ModalityTaskResultQuery{
		TaskID: "task-1", Name: "out.png", Offset: 4, Length: 4,
	})
	if err != nil {
		t.Fatalf("读取结果末页: %v", err)
	}
	if !page.EOF || session.resultCalls != 2 {
		t.Fatalf("结果分页不符：%+v calls=%d", page, session.resultCalls)
	}
}

func TestReadTaskResultRejectsInvalidQueryWithoutCall(t *testing.T) {
	session := &fakeModalitySession{supports: map[string]bool{ModalityMethodTaskResult: true}}
	manager := newModalityManager(t, session, nil)
	for _, query := range []ModalityTaskResultQuery{
		{Name: "out.png", Length: 1},
		{TaskID: "t", Length: 1},
		{TaskID: "t", Name: "out.png", Offset: -1, Length: 1},
		{TaskID: "t", Name: "out.png", Length: 0},
		{TaskID: "t", Name: "out.png", Length: ModalityTaskResultMaxBytes + 1},
	} {
		if _, err := manager.ReadTaskResult(context.Background(), "s1", query); !errors.Is(err, ErrInvalidModalityRequest) {
			t.Fatalf("不合法结果查询应拒绝：%+v -> %v", query, err)
		}
	}
	if session.gateCount() != 0 {
		t.Fatalf("结构不合法时不应读会话方法表：%d", session.gateCount())
	}
}

func TestRealtimeOpenSendClose(t *testing.T) {
	session := &fakeModalitySession{supports: map[string]bool{
		ModalityMethodRealtimeOpen:  true,
		ModalityMethodRealtimeSend:  true,
		ModalityMethodRealtimeClose: true,
	}}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"realtime.open":  func() (write.Reply, error) { return write.Reply{InputID: "conn-1"}, nil },
		"realtime.close": func() (write.Reply, error) { return write.Reply{}, nil },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))

	connection, err := manager.OpenRealtime(context.Background(), ModalityRealtimeOpenRequest{
		SessionID: "s1", Operation: "realtime.voice", Params: json.RawMessage(`{"lang":"zh"}`),
	})
	if err != nil {
		t.Fatalf("打开实时连接: %v", err)
	}
	if connection.ConnectionID != "conn-1" {
		t.Fatalf("连接标识不符：%+v", connection)
	}
	if !hasRealtime(manager, "s1", "conn-1") {
		t.Fatal("打开后应有本地连接状态")
	}
	openParams := decodeParams[modalityRealtimeOpenParams](t, sender.snapshot()[0].Params)
	if openParams.Op != "realtime.open" || openParams.Operation != "realtime.voice" {
		t.Fatalf("打开参数不符：%+v", openParams)
	}

	events := make(chan ModalityRealtimeEvent, 1)
	ended := make(chan ModalityRealtimeEnded, 1)
	cancel, err := manager.WatchRealtime("s1", "conn-1", ModalityRealtimeListener{
		OnEvent: func(event ModalityRealtimeEvent) { events <- event },
		OnEnded: func(event ModalityRealtimeEnded) { ended <- event },
	})
	if err != nil {
		t.Fatalf("订阅实时连接: %v", err)
	}
	defer cancel()

	if err := manager.DeliverRealtimeEvent(ModalityRealtimeEvent{
		SessionID: "s1", ConnectionID: "conn-1", Event: json.RawMessage(`{"kind":"text","data":"hi"}`),
	}); err != nil {
		t.Fatalf("投递实时事件: %v", err)
	}
	select {
	case event := <-events:
		if string(event.Event) != `{"kind":"text","data":"hi"}` {
			t.Fatalf("实时事件不符：%s", event.Event)
		}
	default:
		t.Fatal("订阅者没有收到实时事件")
	}

	if err := manager.SendRealtime(context.Background(), "s1", "conn-1", ModalityRealtimeFrame{
		Kind: "text", Data: json.RawMessage(`"你好"`),
	}); err != nil {
		t.Fatalf("发送实时帧: %v", err)
	}
	if session.sentCount() != 1 || session.sentFrames[0].Kind != "text" {
		t.Fatalf("实时帧不符：%+v", session.sentFrames)
	}

	if err := manager.CloseRealtime(context.Background(), "s1", "conn-1"); err != nil {
		t.Fatalf("关闭实时连接: %v", err)
	}
	if hasRealtime(manager, "s1", "conn-1") {
		t.Fatal("关闭后不应保留连接状态")
	}
	select {
	case event := <-ended:
		if event.Reason != "closed" {
			t.Fatalf("结束原因不符：%+v", event)
		}
	default:
		t.Fatal("订阅者没有收到关闭通知")
	}
	closeParams := decodeParams[modalityRealtimeCloseParams](t, sender.snapshot()[1].Params)
	if closeParams.Op != "realtime.close" || closeParams.ConnectionID != "conn-1" {
		t.Fatalf("关闭参数不符：%+v", closeParams)
	}
	if len(sender.snapshot()) != 2 {
		t.Fatalf("打开与关闭应各发一次：%d", len(sender.snapshot()))
	}
}

func hasRealtime(manager *Manager, sessionID, connectionID string) bool {
	_, ok := manager.RealtimeConnection(sessionID, connectionID)
	return ok
}

func TestRealtimeCapabilityGate(t *testing.T) {
	t.Run("缺打开方法", func(t *testing.T) {
		session := &fakeModalitySession{supports: map[string]bool{}}
		sender := &senderFunc{}
		manager := newModalityManager(t, session, newTestWriter(t, sender))
		if _, err := manager.OpenRealtime(context.Background(), ModalityRealtimeOpenRequest{
			SessionID: "s1", Operation: "realtime.voice",
		}); !errors.Is(err, ErrCapabilityMissing) {
			t.Fatalf("缺打开方法应分类：%v", err)
		}
		if len(sender.snapshot()) != 0 {
			t.Fatal("缺方法时不应发出写请求")
		}
	})

	t.Run("缺发送方法", func(t *testing.T) {
		session := &fakeModalitySession{supports: map[string]bool{
			ModalityMethodRealtimeOpen:  true,
			ModalityMethodRealtimeClose: true,
		}}
		sender := modalityOpSender(t, map[string]func() (write.Reply, error){
			"realtime.open": func() (write.Reply, error) { return write.Reply{InputID: "conn-1"}, nil },
		})
		manager := newModalityManager(t, session, newTestWriter(t, sender))
		if _, err := manager.OpenRealtime(context.Background(), ModalityRealtimeOpenRequest{
			SessionID: "s1", Operation: "realtime.voice",
		}); err != nil {
			t.Fatalf("打开实时连接: %v", err)
		}
		if err := manager.SendRealtime(context.Background(), "s1", "conn-1", ModalityRealtimeFrame{
			Kind: "text", Data: json.RawMessage(`"hi"`),
		}); !errors.Is(err, ErrCapabilityMissing) {
			t.Fatalf("缺发送方法应分类：%v", err)
		}
		if session.sentCount() != 0 {
			t.Fatal("缺发送方法时不应发出会话请求")
		}
	})

	t.Run("缺关闭方法", func(t *testing.T) {
		session := &fakeModalitySession{supports: map[string]bool{
			ModalityMethodRealtimeOpen: true,
		}}
		sender := modalityOpSender(t, map[string]func() (write.Reply, error){
			"realtime.open": func() (write.Reply, error) { return write.Reply{InputID: "conn-1"}, nil },
		})
		manager := newModalityManager(t, session, newTestWriter(t, sender))
		if _, err := manager.OpenRealtime(context.Background(), ModalityRealtimeOpenRequest{
			SessionID: "s1", Operation: "realtime.voice",
		}); err != nil {
			t.Fatalf("打开实时连接: %v", err)
		}
		if err := manager.CloseRealtime(context.Background(), "s1", "conn-1"); !errors.Is(err, ErrCapabilityMissing) {
			t.Fatalf("缺关闭方法应分类：%v", err)
		}
		if len(sender.snapshot()) != 1 {
			t.Fatalf("缺关闭方法时不应发出关闭写入：%d", len(sender.snapshot()))
		}
		if !hasRealtime(manager, "s1", "conn-1") {
			t.Fatal("关闭失败应保留连接状态")
		}
	})
}

func TestRealtimeSendFailureKeepsReason(t *testing.T) {
	session := &fakeModalitySession{
		supports: map[string]bool{
			ModalityMethodRealtimeOpen: true,
			ModalityMethodRealtimeSend: true,
		},
		sendErr: errors.New("发送通道关闭"),
	}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"realtime.open": func() (write.Reply, error) { return write.Reply{InputID: "conn-1"}, nil },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))
	if _, err := manager.OpenRealtime(context.Background(), ModalityRealtimeOpenRequest{
		SessionID: "s1", Operation: "realtime.voice",
	}); err != nil {
		t.Fatalf("打开实时连接: %v", err)
	}
	err := manager.SendRealtime(context.Background(), "s1", "conn-1", ModalityRealtimeFrame{
		Kind: "text", Data: json.RawMessage(`"hi"`),
	})
	if !errors.Is(err, ErrRealtimeSendFailed) {
		t.Fatalf("发送失败应分类：%v", err)
	}
	if !strings.Contains(err.Error(), "发送通道关闭") {
		t.Fatalf("发送失败原因应保留：%v", err)
	}
	if !hasRealtime(manager, "s1", "conn-1") {
		t.Fatal("发送失败不应移除连接")
	}
}

func TestRealtimeCloseFailureKeepsState(t *testing.T) {
	session := &fakeModalitySession{supports: map[string]bool{
		ModalityMethodRealtimeOpen:  true,
		ModalityMethodRealtimeClose: true,
	}}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"realtime.open":  func() (write.Reply, error) { return write.Reply{InputID: "conn-1"}, nil },
		"realtime.close": func() (write.Reply, error) { return write.Reply{}, errors.New("关闭被拒绝") },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))
	if _, err := manager.OpenRealtime(context.Background(), ModalityRealtimeOpenRequest{
		SessionID: "s1", Operation: "realtime.voice",
	}); err != nil {
		t.Fatalf("打开实时连接: %v", err)
	}
	err := manager.CloseRealtime(context.Background(), "s1", "conn-1")
	if !errors.Is(err, ErrRealtimeCloseFailed) {
		t.Fatalf("关闭失败应分类：%v", err)
	}
	if !strings.Contains(err.Error(), "关闭被拒绝") {
		t.Fatalf("关闭失败原因应保留：%v", err)
	}
	if !hasRealtime(manager, "s1", "conn-1") {
		t.Fatal("关闭失败应保留连接状态")
	}
}

func TestRealtimeDeliverUnknownAndEndedAfterClose(t *testing.T) {
	session := &fakeModalitySession{supports: map[string]bool{
		ModalityMethodRealtimeOpen:  true,
		ModalityMethodRealtimeClose: true,
	}}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"realtime.open":  func() (write.Reply, error) { return write.Reply{InputID: "conn-1"}, nil },
		"realtime.close": func() (write.Reply, error) { return write.Reply{}, nil },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))

	if err := manager.DeliverRealtimeEvent(ModalityRealtimeEvent{
		SessionID: "s1", ConnectionID: "missing", Event: json.RawMessage(`{}`),
	}); !errors.Is(err, ErrRealtimeNotOpen) {
		t.Fatalf("未知连接的事件应拒绝：%v", err)
	}
	if err := manager.DeliverRealtimeEnded(ModalityRealtimeEnded{
		SessionID: "s1", ConnectionID: "missing",
	}); err != nil {
		t.Fatalf("未知连接的结束事件应幂等：%v", err)
	}

	if _, err := manager.OpenRealtime(context.Background(), ModalityRealtimeOpenRequest{
		SessionID: "s1", Operation: "realtime.voice",
	}); err != nil {
		t.Fatalf("打开实时连接: %v", err)
	}
	if err := manager.CloseRealtime(context.Background(), "s1", "conn-1"); err != nil {
		t.Fatalf("关闭实时连接: %v", err)
	}
	if err := manager.DeliverRealtimeEnded(ModalityRealtimeEnded{
		SessionID: "s1", ConnectionID: "conn-1", Reason: "provider",
	}); err != nil {
		t.Fatalf("关闭后的结束事件应幂等：%v", err)
	}
}

func TestRealtimeRejectsInvalidFramesWithoutCall(t *testing.T) {
	session := &fakeModalitySession{supports: map[string]bool{
		ModalityMethodRealtimeOpen: true,
		ModalityMethodRealtimeSend: true,
	}}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"realtime.open": func() (write.Reply, error) { return write.Reply{InputID: "conn-1"}, nil },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))
	if _, err := manager.OpenRealtime(context.Background(), ModalityRealtimeOpenRequest{
		SessionID: "s1", Operation: "realtime.voice",
	}); err != nil {
		t.Fatalf("打开实时连接: %v", err)
	}
	for _, frame := range []ModalityRealtimeFrame{
		{Kind: "video", Data: json.RawMessage(`"x"`)},
		{Kind: "text"},
		{Kind: "audio", Data: json.RawMessage(`"not-base64?"`)},
	} {
		if err := manager.SendRealtime(context.Background(), "s1", "conn-1", frame); !errors.Is(err, ErrInvalidModalityRequest) {
			t.Fatalf("不合法帧应拒绝：%+v -> %v", frame, err)
		}
	}
	if session.sentCount() != 0 {
		t.Fatalf("不合法帧不应发出会话请求：%d", session.sentCount())
	}
}

func TestModalityTaskValidationRejectsMalformedFacts(t *testing.T) {
	valid := taskFixture("task-1", ModalityTaskRunning)
	cases := []struct {
		name string
		task ModalityTask
	}{
		{"缺标识", func() ModalityTask { t := valid; t.ID = ""; return t }()},
		{"缺 category", func() ModalityTask { t := valid; t.Category = ""; return t }()},
		{"缺 operation", func() ModalityTask { t := valid; t.Operation = ""; return t }()},
		{"未知状态", func() ModalityTask { t := valid; t.State = "done"; return t }()},
		{"缺创建时间", func() ModalityTask { t := valid; t.CreatedAt = ""; return t }()},
		{"结果文件缺摘要", func() ModalityTask {
			t := valid
			t.Result = &ModalityTaskResultData{Files: []ModalityTaskFile{{Name: "a", MediaType: "text/plain"}}}
			return t
		}()},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := validateTask(testCase.task); !errors.Is(err, ErrMalformedModality) {
				t.Fatalf("不合法任务应拒绝：%v", err)
			}
		})
	}
}

func TestReadTaskAndMalformedFacts(t *testing.T) {
	session := &fakeModalitySession{
		supports: map[string]bool{ModalityMethodTaskStatus: true},
		task:     taskFixture("task-1", ModalityTaskRunning),
	}
	manager := newModalityManager(t, session, nil)
	task, err := manager.ReadTask(context.Background(), "s1", "task-1")
	if err != nil {
		t.Fatalf("读取任务: %v", err)
	}
	if task.ID != "task-1" || task.State != ModalityTaskRunning || task.SessionID != "s1" {
		t.Fatalf("任务不符：%+v", task)
	}
	if session.readCount() != 1 {
		t.Fatalf("任务读取次数不符：%d", session.readCount())
	}

	session.task = taskFixture("task-1", ModalityTaskRunning)
	session.task.Result.Files[0].SHA256 = ""
	if _, err := manager.ReadTask(context.Background(), "s1", "task-1"); !errors.Is(err, ErrMalformedModality) {
		t.Fatalf("事实不合法应拒绝：%v", err)
	}

	session.supports = map[string]bool{}
	if _, err := manager.ReadTask(context.Background(), "s1", "task-1"); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺状态方法应分类：%v", err)
	}
}

func TestModalityWriteResultRequiresResourceID(t *testing.T) {
	session := &fakeModalitySession{
		supports: map[string]bool{
			ModalityMethodTaskSubmit:   true,
			ModalityMethodTaskStatus:   true,
			ModalityMethodRealtimeOpen: true,
		},
		task: taskFixture("task-1", ModalityTaskQueued),
	}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"task.submit":   func() (write.Reply, error) { return write.Reply{}, nil },
		"realtime.open": func() (write.Reply, error) { return write.Reply{}, nil },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))

	if _, err := manager.SubmitTask(context.Background(), ModalityTaskSubmitRequest{
		SessionID: "s1", Category: "image", Operation: "generate",
	}); !errors.Is(err, ErrMalformedWriteResult) {
		t.Fatalf("提交回执缺任务标识应拒绝：%v", err)
	}
	if _, err := manager.OpenRealtime(context.Background(), ModalityRealtimeOpenRequest{
		SessionID: "s1", Operation: "realtime.voice",
	}); !errors.Is(err, ErrMalformedWriteResult) {
		t.Fatalf("打开回执缺连接标识应拒绝：%v", err)
	}
}

func TestModalitySessionReadFailuresAndBadBase64(t *testing.T) {
	session := &fakeModalitySession{
		supports: map[string]bool{
			ModalityMethodTaskList:   true,
			ModalityMethodTaskResult: true,
		},
		pageErr:   errors.New("清单读取失败"),
		resultErr: errors.New("结果读取失败"),
	}
	manager := newModalityManager(t, session, nil)
	if _, err := manager.ListTasks(context.Background(), "s1", ModalityTaskQuery{}); !errors.Is(err, ErrTaskListFailed) {
		t.Fatalf("清单读取失败应分类：%v", err)
	}
	if _, err := manager.ReadTaskResult(context.Background(), "s1", ModalityTaskResultQuery{
		TaskID: "task-1", Name: "out.png", Length: 4,
	}); !errors.Is(err, ErrTaskResultFailed) {
		t.Fatalf("结果读取失败应分类：%v", err)
	}

	session.resultErr = nil
	session.result = ModalityTaskResultPage{DataBase64: "%%%", EOF: true}
	if _, err := manager.ReadTaskResult(context.Background(), "s1", ModalityTaskResultQuery{
		TaskID: "task-1", Name: "out.png", Length: 4,
	}); !errors.Is(err, ErrMalformedModality) {
		t.Fatalf("结果不是合法 base64 应拒绝：%v", err)
	}
}

func TestRealtimeWatchAndCloseEdgeCases(t *testing.T) {
	session := &fakeModalitySession{supports: map[string]bool{
		ModalityMethodRealtimeOpen:  true,
		ModalityMethodRealtimeClose: true,
	}}
	sender := modalityOpSender(t, map[string]func() (write.Reply, error){
		"realtime.open":  func() (write.Reply, error) { return write.Reply{InputID: "conn-1"}, nil },
		"realtime.close": func() (write.Reply, error) { return write.Reply{}, nil },
	})
	manager := newModalityManager(t, session, newTestWriter(t, sender))

	if _, err := manager.WatchRealtime("s1", "missing", ModalityRealtimeListener{OnEvent: func(ModalityRealtimeEvent) {}}); !errors.Is(err, ErrRealtimeNotOpen) {
		t.Fatalf("订阅未知连接应拒绝：%v", err)
	}
	if err := manager.CloseRealtime(context.Background(), "s1", "missing"); !errors.Is(err, ErrRealtimeNotOpen) {
		t.Fatalf("关闭未知连接应拒绝：%v", err)
	}
	if _, err := manager.OpenRealtime(context.Background(), ModalityRealtimeOpenRequest{
		SessionID: "s1", Operation: "realtime.voice",
	}); err != nil {
		t.Fatalf("打开实时连接: %v", err)
	}
	if _, err := manager.WatchRealtime("s1", "conn-1", ModalityRealtimeListener{}); !errors.Is(err, ErrInvalidModalityRequest) {
		t.Fatalf("空订阅应拒绝：%v", err)
	}
}
