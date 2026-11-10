package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// stubAppError 模拟服务端带 appError 的错误，不依赖连接领域的 wire 类型。
type stubAppError struct{ app string }

func (e stubAppError) Error() string    { return "rpc error: " + e.app }
func (e stubAppError) AppError() string { return e.app }

// fakeWriter 是写登记的替身：按调用序号给出状态，并保留每次登记的意图。
type fakeWriter struct {
	mu          sync.Mutex
	intents     []write.Intent
	viewsByKey  map[string]write.IntentView
	viewsByID   map[string]write.IntentView
	next        int
	registerErr error
	statusFor   func(call int) write.Status
}

func newFakeWriter() *fakeWriter {
	return &fakeWriter{
		viewsByKey: make(map[string]write.IntentView),
		viewsByID:  make(map[string]write.IntentView),
	}
}

func (w *fakeWriter) Register(intent write.Intent) (write.IntentView, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.registerErr != nil {
		return write.IntentView{}, w.registerErr
	}
	w.next++
	if view, exists := w.viewsByKey[intent.Key]; exists {
		return view, nil
	}
	status := write.StatusSucceeded
	if w.statusFor != nil {
		status = w.statusFor(w.next)
	}
	view := write.IntentView{
		ID:        fmt.Sprintf("intent-%d", w.next),
		Key:       intent.Key,
		Kind:      intent.Kind,
		Status:    status,
		Content:   append(json.RawMessage(nil), intent.Params...),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	w.intents = append(w.intents, intent)
	w.viewsByKey[intent.Key] = view
	w.viewsByID[view.ID] = view
	return view, nil
}

func (w *fakeWriter) Wait(ctx context.Context, intentID string) (write.IntentView, error) {
	if err := ctx.Err(); err != nil {
		return write.IntentView{}, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	view, ok := w.viewsByID[intentID]
	if !ok {
		return write.IntentView{}, fmt.Errorf("未知意图 %q", intentID)
	}
	return view, nil
}

func (w *fakeWriter) intentList() []write.Intent {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]write.Intent(nil), w.intents...)
}

func (w *fakeWriter) setStatusFor(fn func(call int) write.Status) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.statusFor = fn
}

type pluginReaderFunc func(context.Context) ([]Plugin, error)

func (f pluginReaderFunc) ReadPlugins(ctx context.Context) ([]Plugin, error) { return f(ctx) }

type skillReaderFunc func(context.Context) ([]Skill, error)

func (f skillReaderFunc) ReadSkills(ctx context.Context) ([]Skill, error) { return f(ctx) }

type hookReaderFunc func(context.Context) ([]Hook, error)

func (f hookReaderFunc) ReadHooks(ctx context.Context) ([]Hook, error) { return f(ctx) }

type schedulerReaderFunc func(context.Context) ([]ScheduledTask, error)

func (f schedulerReaderFunc) ReadScheduledTasks(ctx context.Context) ([]ScheduledTask, error) {
	return f(ctx)
}

type wakeupReaderFunc func(context.Context, WakeupScope) ([]Wakeup, error)

func (f wakeupReaderFunc) ReadWakeups(ctx context.Context, scope WakeupScope) ([]Wakeup, error) {
	return f(ctx, scope)
}

type toolServerReaderStub struct {
	list    func(context.Context) ([]ToolServer, error)
	probe   func(context.Context, string) (ToolServerProbe, error)
	preview func(context.Context, string) (ImportPreview, error)
}

func (s toolServerReaderStub) ReadToolServers(ctx context.Context) ([]ToolServer, error) {
	return s.list(ctx)
}

func (s toolServerReaderStub) ProbeToolServer(ctx context.Context, serverID string) (ToolServerProbe, error) {
	return s.probe(ctx, serverID)
}

func (s toolServerReaderStub) PreviewToolServerImport(ctx context.Context, source string) (ImportPreview, error) {
	return s.preview(ctx, source)
}

type workflowReaderStub struct {
	list func(context.Context) ([]WorkflowDefinition, error)
	get  func(context.Context, string) (WorkflowDefinition, error)
	runs func(context.Context, string) ([]WorkflowRun, error)
}

func (s workflowReaderStub) ReadWorkflows(ctx context.Context) ([]WorkflowDefinition, error) {
	return s.list(ctx)
}

func (s workflowReaderStub) ReadWorkflow(ctx context.Context, workflowID string) (WorkflowDefinition, error) {
	return s.get(ctx, workflowID)
}

func (s workflowReaderStub) ReadWorkflowRuns(ctx context.Context, workflowID string) ([]WorkflowRun, error) {
	return s.runs(ctx, workflowID)
}

func TestCapabilityMissingIsClassifiedAndRecovers(t *testing.T) {
	calls := 0
	logs := 0
	service, err := New(Config{
		Scheduler: schedulerReaderFunc(func(context.Context) ([]ScheduledTask, error) {
			calls++
			if calls == 1 {
				return nil, stubAppError{app: "unknown_method"}
			}
			return []ScheduledTask{{ID: "task-1", Name: "夜间构建"}}, nil
		}),
		Logf: func(string, ...any) { logs++ },
	})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	snapshot, err := service.ListScheduledTasks(context.Background())
	if !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺能力没有分类: %v", err)
	}
	var capability *CapabilityError
	if !errors.As(err, &capability) || capability.Feature != "schedule.list" {
		t.Fatalf("能力缺失错误没有带能力名: %v", err)
	}
	if !snapshot.CapabilityMissing || snapshot.ReadError == "" {
		t.Fatalf("缺能力读数没有标记: %+v", snapshot)
	}
	if logs == 0 {
		t.Fatal("缺能力没有记录日志")
	}

	snapshot, err = service.ListScheduledTasks(context.Background())
	if err != nil {
		t.Fatalf("能力恢复后读取失败: %v", err)
	}
	if len(snapshot.Items) != 1 || snapshot.Items[0].ID != "task-1" {
		t.Fatalf("能力恢复后没有任务列表: %+v", snapshot)
	}
	if snapshot.CapabilityMissing || snapshot.ReadError != "" {
		t.Fatalf("恢复后仍标记缺能力: %+v", snapshot)
	}

	empty, err := New(Config{})
	if err != nil {
		t.Fatalf("创建空领域实例: %v", err)
	}
	defer empty.Close()
	plugins, err := empty.ListPlugins(context.Background())
	if !errors.Is(err, ErrCapabilityMissing) || !plugins.CapabilityMissing {
		t.Fatalf("没有读取方时没有报能力缺失: %v %+v", err, plugins)
	}
}

func TestPluginListKeepsOptionalUninstalled(t *testing.T) {
	service, err := New(Config{
		Plugins: pluginReaderFunc(func(context.Context) ([]Plugin, error) {
			return []Plugin{
				{ID: "installed", Name: "已装", Installed: true, Enabled: true, Source: SourceLocal},
				{ID: "optional", Name: "可选", Installed: false, Optional: true, Source: SourceBundled},
			}, nil
		}),
	})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	snapshot, err := service.ListPlugins(context.Background())
	if err != nil {
		t.Fatalf("读取插件: %v", err)
	}
	if len(snapshot.Items) != 2 {
		t.Fatalf("可选未安装插件被过滤: %+v", snapshot.Items)
	}
	if snapshot.Items[1].Installed || !snapshot.Items[1].Optional {
		t.Fatalf("可选插件没有显示为未安装: %+v", snapshot.Items[1])
	}
}

func TestHookSourceIsTheOnlyWritableOne(t *testing.T) {
	writer := newFakeWriter()
	service, err := New(Config{Writer: writer})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	for _, source := range []Source{SourceProject, SourcePlugin, SourceUnknown} {
		if _, err := service.WriteHook(context.Background(), Hook{Event: "pre", Source: source}); !errors.Is(err, ErrSourceNotWritable) {
			t.Fatalf("来源 %s 没有被拒绝: %v", source, err)
		}
	}
	if intents := writer.intentList(); len(intents) != 0 {
		t.Fatalf("只读来源发出了写请求: %+v", intents)
	}

	view, err := service.WriteHook(context.Background(), Hook{Event: "pre", Command: "echo ok", Source: SourceUser})
	if err != nil {
		t.Fatalf("用户来源钩子写入失败: %v", err)
	}
	if view.Status != write.StatusSucceeded {
		t.Fatalf("用户来源钩子没有成功: %+v", view)
	}
	intents := writer.intentList()
	if len(intents) != 1 || intents[0].Kind != write.KindManagementWrite {
		t.Fatalf("用户来源钩子没有经写登记: %+v", intents)
	}
	managed, err := DecodeManagementWrite(intents[0].Params)
	if err != nil {
		t.Fatalf("解析管理写入: %v", err)
	}
	if managed.Op != OpHookWrite {
		t.Fatalf("管理写入操作不符: %q", managed.Op)
	}
}

func TestImportPreviewSelectsOnlyCheckedCandidates(t *testing.T) {
	preview := ImportPreview{
		Candidates: []ImportCandidate{
			{Key: "alpha", Server: ToolServerSpec{Name: "alpha", Transport: ToolTransportStdio, Command: "alpha-bin"}},
			{Key: "beta", Server: ToolServerSpec{Name: "beta", Transport: ToolTransportHTTP, URL: "https://beta.example"}},
			{Key: "gamma", Server: ToolServerSpec{Name: "gamma", Transport: ToolTransportSSE, URL: "https://gamma.example"}},
		},
	}
	reader := toolServerReaderStub{
		list: func(context.Context) ([]ToolServer, error) { return nil, nil },
		probe: func(context.Context, string) (ToolServerProbe, error) {
			return ToolServerProbe{}, nil
		},
		preview: func(context.Context, string) (ImportPreview, error) { return preview, nil },
	}
	writer := newFakeWriter()
	service, err := New(Config{ToolServers: reader, Writer: writer})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	got, err := service.PreviewToolServerImport(context.Background(), "config-a")
	if err != nil {
		t.Fatalf("预览导入: %v", err)
	}
	if len(got.Candidates) != 3 {
		t.Fatalf("预览候选条数不符: %+v", got.Candidates)
	}
	if _, err := service.ApplyToolServerImport(context.Background(), "config-a", nil); !errors.Is(err, ErrEmptySelection) {
		t.Fatalf("空勾选没有拒绝: %v", err)
	}
	if _, err := service.ApplyToolServerImport(context.Background(), "config-b", []string{"alpha"}); !errors.Is(err, ErrPreviewRequired) {
		t.Fatalf("未预览就应用没有被拒绝: %v", err)
	}
	if _, err := service.ApplyToolServerImport(context.Background(), "config-a", []string{"missing"}); !errors.Is(err, ErrUnknownCandidate) {
		t.Fatalf("未知候选没有被拒绝: %v", err)
	}
	if intents := writer.intentList(); len(intents) != 0 {
		t.Fatalf("非法勾选发出了写请求: %+v", intents)
	}

	result, err := service.ApplyToolServerImport(context.Background(), "config-a", []string{"gamma", "beta"})
	if err != nil {
		t.Fatalf("按勾选写入: %v", err)
	}
	if result.Written != 2 || len(result.Views) != 2 {
		t.Fatalf("写入条数不符: %+v", result)
	}
	intents := writer.intentList()
	if len(intents) != 2 {
		t.Fatalf("写入条数不等于勾选条数: %d", len(intents))
	}
	var names []string
	for _, intent := range intents {
		managed, err := DecodeManagementWrite(intent.Params)
		if err != nil {
			t.Fatalf("解析管理写入: %v", err)
		}
		if managed.Op != OpToolServerAdd {
			t.Fatalf("导入写入操作不符: %q", managed.Op)
		}
		var payload ToolServerAddPayload
		if err := json.Unmarshal(managed.Payload, &payload); err != nil {
			t.Fatalf("解析导入载荷: %v", err)
		}
		if payload.ImportSource != "config-a" || payload.CandidateKey == "" {
			t.Fatalf("导入载荷缺少来源或候选键: %+v", payload)
		}
		names = append(names, payload.Server.Name)
	}
	if len(names) != 2 || names[0] != "beta" || names[1] != "gamma" {
		t.Fatalf("写入顺序或候选不符: %v", names)
	}
}

func TestWriteRequiresWriter(t *testing.T) {
	service, err := New(Config{})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	if _, err := service.InstallPlugin(context.Background(), InstallSource{Kind: SourceLocal, Ref: "/tmp/plugin"}); !errors.Is(err, ErrWriterUnavailable) {
		t.Fatalf("没有写登记时没有报不可用: %v", err)
	}
	if _, err := service.SetPluginEnabled(context.Background(), "p", true); !errors.Is(err, ErrWriterUnavailable) {
		t.Fatalf("没有写登记时没有报不可用: %v", err)
	}
	if _, err := service.SetPluginEnabled(context.Background(), "", true); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("空插件标识没有拒绝: %v", err)
	}
}

func TestSnapshotKeepsConfirmedFactsOnReadFailure(t *testing.T) {
	fail := false
	service, err := New(Config{
		Scheduler: schedulerReaderFunc(func(context.Context) ([]ScheduledTask, error) {
			if fail {
				return nil, stubAppError{app: "service_reloading"}
			}
			return []ScheduledTask{{ID: "task-1"}}, nil
		}),
	})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	if _, err := service.ListScheduledTasks(context.Background()); err != nil {
		t.Fatalf("首次读取: %v", err)
	}
	fail = true
	snapshot, err := service.ListScheduledTasks(context.Background())
	if err == nil {
		t.Fatal("读取失败没有报错")
	}
	if len(snapshot.Items) != 1 || snapshot.Items[0].ID != "task-1" {
		t.Fatalf("读取失败清掉了已确认事实: %+v", snapshot)
	}
	if snapshot.ReadError == "" {
		t.Fatal("读取失败没有留下原因")
	}
}

func TestSchedulerFollowsNotificationOnceWithoutPolling(t *testing.T) {
	calls := 0
	service, err := New(Config{
		Scheduler: schedulerReaderFunc(func(context.Context) ([]ScheduledTask, error) {
			calls++
			return []ScheduledTask{{ID: "task-1"}}, nil
		}),
	})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	if err := service.Notify(Notice{Topic: TopicScheduler}); err != nil {
		t.Fatalf("登记通知: %v", err)
	}
	if err := service.WaitIdle(context.Background()); err != nil {
		t.Fatalf("等待跟随: %v", err)
	}
	if calls != 1 {
		t.Fatalf("一次通知重读次数不符: %d", calls)
	}
	time.Sleep(50 * time.Millisecond)
	if calls != 1 {
		t.Fatalf("通知之后出现了轮询: %d", calls)
	}
	if snapshot := service.ScheduledTaskSnapshot(); len(snapshot.Items) != 1 {
		t.Fatalf("跟随重读没有更新读数: %+v", snapshot)
	}
	stats := service.FollowStats()
	if stats.Notices[TopicScheduler] != 1 || stats.Reads[TopicScheduler] != 1 {
		t.Fatalf("跟随统计不符: %+v", stats)
	}

	if err := service.Notify(Notice{Topic: TopicScheduler}); err != nil {
		t.Fatalf("登记第二次通知: %v", err)
	}
	if err := service.WaitIdle(context.Background()); err != nil {
		t.Fatalf("等待第二次跟随: %v", err)
	}
	if calls != 2 {
		t.Fatalf("第二次通知没有重读一次: %d", calls)
	}
}

func TestWakeupFollowsCancellationOnce(t *testing.T) {
	cancelled := false
	reads := 0
	service, err := New(Config{
		Writer: newFakeWriter(),
		Wakeups: wakeupReaderFunc(func(_ context.Context, scope WakeupScope) ([]Wakeup, error) {
			reads++
			if cancelled {
				return nil, nil
			}
			return []Wakeup{{ID: "wake-1", SessionID: scope.SessionID, BranchID: scope.BranchID, Status: WakeupScheduled}}, nil
		}),
	})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	scope := WakeupScope{SessionID: "s-1", BranchID: "main"}
	snapshot, err := service.ListWakeups(context.Background(), scope)
	if err != nil || len(snapshot.Items) != 1 {
		t.Fatalf("读取唤醒: %v %+v", err, snapshot)
	}
	if _, err := service.CancelWakeup(context.Background(), WakeupRef{SessionID: "s-1", BranchID: "main", WakeupID: "wake-1"}); err != nil {
		t.Fatalf("取消唤醒: %v", err)
	}
	if reads != 1 {
		t.Fatalf("取消本身不应重读: %d", reads)
	}
	cancelled = true
	if err := service.Notify(Notice{Topic: TopicWakeups, SessionID: "s-1", BranchID: "main"}); err != nil {
		t.Fatalf("登记唤醒通知: %v", err)
	}
	if err := service.WaitIdle(context.Background()); err != nil {
		t.Fatalf("等待唤醒跟随: %v", err)
	}
	if reads != 2 {
		t.Fatalf("通知重读次数不符: %d", reads)
	}
	if snapshot := service.WakeupSnapshot(scope); len(snapshot.Items) != 0 {
		t.Fatalf("取消后记录仍在: %+v", snapshot)
	}
	time.Sleep(50 * time.Millisecond)
	if reads != 2 {
		t.Fatalf("唤醒跟随出现轮询: %d", reads)
	}
	stats := service.FollowStats()
	if stats.WakeupReads[scopeKey(scope)] != 1 {
		t.Fatalf("唤醒重读统计不符: %+v", stats)
	}
}

func TestWriteRetriesWithNewKeyAfterFailure(t *testing.T) {
	writer := newFakeWriter()
	writer.setStatusFor(func(call int) write.Status {
		if call == 1 {
			return write.StatusFailed
		}
		return write.StatusSucceeded
	})
	service, err := New(Config{Writer: writer})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	source := InstallSource{Kind: SourceLocal, Ref: "/tmp/plugin"}
	first, err := service.InstallPlugin(context.Background(), source)
	if err != nil || first.Status != write.StatusFailed {
		t.Fatalf("第一次安装状态不符: %v %+v", err, first)
	}
	second, err := service.InstallPlugin(context.Background(), source)
	if err != nil || second.Status != write.StatusSucceeded {
		t.Fatalf("失败后重试状态不符: %v %+v", err, second)
	}
	intents := writer.intentList()
	if len(intents) != 2 {
		t.Fatalf("失败后没有换键重试: %d", len(intents))
	}
	if intents[0].Key == intents[1].Key {
		t.Fatalf("重试沿用了失败的幂等键: %q", intents[0].Key)
	}
}

func TestInvalidInstallSourceIsRejectedBeforeWrite(t *testing.T) {
	writer := newFakeWriter()
	service, err := New(Config{Writer: writer})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	for _, source := range []InstallSource{
		{Kind: SourceUser, Ref: "/tmp/plugin"},
		{Kind: SourceLocal, Ref: "  "},
		{Kind: SourceGit, Ref: "x\x00y"},
	} {
		if _, err := service.InstallPlugin(context.Background(), source); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("非法安装来源没有拒绝: %+v %v", source, err)
		}
	}
	if intents := writer.intentList(); len(intents) != 0 {
		t.Fatalf("非法安装来源发出了写请求: %+v", intents)
	}
}

func TestEveryCapabilityReportsMissingWhenReaderIsAbsent(t *testing.T) {
	service, err := New(Config{})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()
	ctx := context.Background()

	checks := []struct {
		name string
		call func() error
	}{
		{"plugin.list", func() error { _, err := service.ListPlugins(ctx); return err }},
		{"skills.catalog", func() error { _, err := service.ListSkills(ctx); return err }},
		{"hooks.catalog", func() error { _, err := service.ListHooks(ctx); return err }},
		{"hooks.catalog", func() error { _, err := service.ReadProjectHooks(ctx, "/work"); return err }},
		{"mcp.catalog", func() error { _, err := service.ListToolServers(ctx); return err }},
		{"mcp.test", func() error { _, err := service.ProbeToolServer(ctx, "srv"); return err }},
		{"mcp.import.preview", func() error {
			_, err := service.PreviewToolServerImport(ctx, "config")
			return err
		}},
		{"schedule.list", func() error { _, err := service.ListScheduledTasks(ctx); return err }},
		{"workflow.list", func() error { _, err := service.ListWorkflows(ctx); return err }},
		{"workflow.get", func() error { _, err := service.ReadWorkflow(ctx, "wf"); return err }},
		{"workflow.runs", func() error { _, err := service.ListWorkflowRuns(ctx, "wf"); return err }},
		{"session.wakeup.list", func() error {
			_, err := service.ListWakeups(ctx, WakeupScope{SessionID: "s"})
			return err
		}},
	}
	for _, check := range checks {
		err := check.call()
		if !errors.Is(err, ErrCapabilityMissing) {
			t.Fatalf("%s 缺能力没有分类: %v", check.name, err)
		}
		var capability *CapabilityError
		if !errors.As(err, &capability) || capability.Feature != check.name {
			t.Fatalf("%s 的能力错误没有带能力名: %v", check.name, err)
		}
	}
}

func TestInstallSourceAcceptsEverySupportedKind(t *testing.T) {
	for _, kind := range []Source{SourceLocal, SourceGit, SourceBundled} {
		source := InstallSource{Kind: kind, Ref: "ref"}
		if err := source.validate(); err != nil {
			t.Fatalf("来源 %s 被拒绝: %v", kind, err)
		}
	}
	for _, kind := range []Source{SourceUser, SourceProject, SourcePlugin, SourceUnknown, ""} {
		source := InstallSource{Kind: kind, Ref: "ref"}
		if err := source.validate(); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("来源 %q 没有被拒绝: %v", kind, err)
		}
	}
}

func TestWakeupReadingsHideCancelledAndExpired(t *testing.T) {
	service, err := New(Config{
		Wakeups: wakeupReaderFunc(func(context.Context, WakeupScope) ([]Wakeup, error) {
			return []Wakeup{
				{ID: "scheduled", Status: WakeupScheduled},
				{ID: "fired", Status: WakeupFired},
				{ID: "cancelled", Status: WakeupCancelled},
				{ID: "expired", Status: WakeupExpired},
			}, nil
		}),
	})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	scope := WakeupScope{SessionID: "s-1"}
	snapshot, err := service.ListWakeups(context.Background(), scope)
	if err != nil {
		t.Fatalf("读取唤醒: %v", err)
	}
	if len(snapshot.Items) != 2 {
		t.Fatalf("取消/过期记录仍出现在读数里: %+v", snapshot.Items)
	}
	for _, wakeup := range snapshot.Items {
		if wakeup.ID == "cancelled" || wakeup.ID == "expired" {
			t.Fatalf("取消/过期记录仍出现在读数里: %+v", snapshot.Items)
		}
	}
}

func TestDecodeRealMethodNamesNormalizes(t *testing.T) {
	cases := []struct {
		name   string
		params string
		wantOp WriteOp
		check  func(*testing.T, json.RawMessage)
	}{
		{
			name:   "skills set enabled",
			params: `{"op":"skills.set_enabled","payload":{"skillId":"s1","enabled":true}}`,
			wantOp: OpSkillSetEnabled,
		},
		{
			name:   "hooks save",
			params: `{"op":"hooks.save","payload":{"hook":{"event":"pre"},"expectedRevision":"r1"}}`,
			wantOp: OpHookWrite,
		},
		{
			name:   "mcp save",
			params: `{"op":"mcp.save","payload":{"id":"m1","source":"user","configuration":{}}}`,
			wantOp: OpToolServerAdd,
		},
		{
			name:   "mcp save",
			params: `{"op":"mcp.save","payload":{"id":"m1","source":"user","expectedVersion":"v1","configuration":{"command":"x"}}}`,
			wantOp: OpToolServerAdd,
			check: func(t *testing.T, payload json.RawMessage) {
				var decoded ToolServerAddPayload
				if err := json.Unmarshal(payload, &decoded); err != nil {
					t.Fatalf("解析 mcp.save 载荷: %v", err)
				}
				if decoded.Server.Name != "m1" || decoded.Server.Source != SourceUser ||
					decoded.Server.ExpectedVersion != "v1" || len(decoded.Server.Configuration) == 0 {
					t.Fatalf("mcp.save 归一不符: %+v", decoded)
				}
			},
		},
		{
			name:   "mcp remove",
			params: `{"op":"mcp.remove","payload":{"id":"m1","source":"user"}}`,
			wantOp: OpToolServerRemove,
			check: func(t *testing.T, payload json.RawMessage) {
				var decoded ToolServerRemovePayload
				if err := json.Unmarshal(payload, &decoded); err != nil {
					t.Fatalf("解析 mcp.remove 载荷: %v", err)
				}
				if decoded.ServerID != "m1" {
					t.Fatalf("mcp.remove 归一不符: %+v", decoded)
				}
			},
		},
		{
			name:   "session wakeup cancel",
			params: `{"op":"session.wakeup.cancel","payload":{"sessionId":"s1","wakeupId":"w1"}}`,
			wantOp: OpWakeupCancel,
		},
		{
			name:   "plugin enable",
			params: `{"op":"plugin.enable","payload":{"name":"Attachments"}}`,
			wantOp: OpPluginSetEnabled,
			check: func(t *testing.T, payload json.RawMessage) {
				var decoded PluginEnabledPayload
				if err := json.Unmarshal(payload, &decoded); err != nil {
					t.Fatalf("解析插件启停载荷: %v", err)
				}
				if decoded.PluginID != "Attachments" || !decoded.Enabled {
					t.Fatalf("插件启停归一不符: %+v", decoded)
				}
			},
		},
		{
			name:   "plugin disable",
			params: `{"op":"plugin.disable","payload":{"name":"Attachments"}}`,
			wantOp: OpPluginSetEnabled,
			check: func(t *testing.T, payload json.RawMessage) {
				var decoded PluginEnabledPayload
				if err := json.Unmarshal(payload, &decoded); err != nil {
					t.Fatalf("解析插件启停载荷: %v", err)
				}
				if decoded.PluginID != "Attachments" || decoded.Enabled {
					t.Fatalf("插件启停归一不符: %+v", decoded)
				}
			},
		},
		{
			name:   "mcp import apply",
			params: `{"op":"mcp.import.apply","payload":{"source":"claude-code:user","ids":["c1"]}}`,
			wantOp: OpToolServerAdd,
			check: func(t *testing.T, payload json.RawMessage) {
				var decoded ToolServerAddPayload
				if err := json.Unmarshal(payload, &decoded); err != nil {
					t.Fatalf("解析 MCP 导入载荷: %v", err)
				}
				if decoded.ImportSource != "claude-code:user" || decoded.CandidateKey != "c1" {
					t.Fatalf("MCP 导入归一不符: %+v", decoded)
				}
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			managed, err := DecodeManagementWrite(json.RawMessage(testCase.params))
			if err != nil {
				t.Fatalf("解析真实方法参数: %v", err)
			}
			if managed.Op != testCase.wantOp {
				t.Fatalf("归一操作不符: got %q want %q", managed.Op, testCase.wantOp)
			}
			if testCase.check != nil {
				testCase.check(t, managed.Payload)
			}
		})
	}
}

func TestHookDocumentMatchesAlpha21Shape(t *testing.T) {
	document, err := HookDocument(Hook{
		Event: "SessionStart", Command: "printf d11b", Source: SourceUser, Type: "command",
	})
	if err != nil {
		t.Fatalf("生成钩子文档: %v", err)
	}
	var parsed struct {
		Hooks map[string][]struct {
			Matcher any `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(document, &parsed); err != nil {
		t.Fatalf("解析钩子文档: %v", err)
	}
	entries := parsed.Hooks["SessionStart"]
	if len(entries) != 1 || entries[0].Matcher != nil || len(entries[0].Hooks) != 1 ||
		entries[0].Hooks[0].Type != "command" || entries[0].Hooks[0].Command != "printf d11b" {
		t.Fatalf("钩子文档形状不符: %s", document)
	}
}

func TestWriteHookWithRevisionCarriesVersionAndDocument(t *testing.T) {
	writer := newFakeWriter()
	service, err := New(Config{Writer: writer})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	view, err := service.WriteHookWithRevision(context.Background(), Hook{
		Event: "SessionStart", Command: "printf d11b", Source: SourceUser, Type: "command",
	}, "native:user/hooks:1")
	if err != nil || view.Status != write.StatusSucceeded {
		t.Fatalf("带版本写钩子: %v %+v", err, view)
	}
	intents := writer.intentList()
	if len(intents) != 1 {
		t.Fatalf("写意图条数不符: %d", len(intents))
	}
	var payload HookWritePayload
	managed, err := DecodeManagementWrite(intents[0].Params)
	if err != nil {
		t.Fatalf("解析管理写入: %v", err)
	}
	if err := json.Unmarshal(managed.Payload, &payload); err != nil {
		t.Fatalf("解析钩子载荷: %v", err)
	}
	if payload.ExpectedRevision != "native:user/hooks:1" || len(payload.Document) == 0 {
		t.Fatalf("钩子载荷缺少版本或文档: %+v", payload)
	}
}

func TestToolServerConfigurationOmitsIncompatibleFields(t *testing.T) {
	stdio, err := ToolServerConfiguration(ToolServerSpec{
		Name: "cat", Transport: ToolTransportStdio, Command: "cat",
	})
	if err != nil {
		t.Fatalf("生成 stdio 配置: %v", err)
	}
	var stdioDoc map[string]any
	if err := json.Unmarshal(stdio, &stdioDoc); err != nil {
		t.Fatalf("解析 stdio 配置: %v", err)
	}
	if stdioDoc["command"] != "cat" {
		t.Fatalf("stdio 配置缺少 command: %s", stdio)
	}
	if _, exists := stdioDoc["url"]; exists {
		t.Fatalf("stdio 配置不应带 url: %s", stdio)
	}

	httpConfig, err := ToolServerConfiguration(ToolServerSpec{
		Name: "remote", Transport: ToolTransportHTTP, URL: "https://example.com",
	})
	if err != nil {
		t.Fatalf("生成 http 配置: %v", err)
	}
	var httpDoc map[string]any
	if err := json.Unmarshal(httpConfig, &httpDoc); err != nil {
		t.Fatalf("解析 http 配置: %v", err)
	}
	if httpDoc["url"] != "https://example.com" {
		t.Fatalf("http 配置缺少 url: %s", httpConfig)
	}
	if _, exists := httpDoc["command"]; exists {
		t.Fatalf("http 配置不应带 command: %s", httpConfig)
	}
}

type stubFeatures map[string]bool

func (f stubFeatures) HasFeature(name string) bool { return f[name] }

type projectHookReaderStub struct {
	catalog ProjectHookCatalog
}

func (r projectHookReaderStub) ReadProjectHooks(context.Context, string) (ProjectHookCatalog, error) {
	return r.catalog, nil
}

type skillCatalogReaderStub struct {
	catalog SkillCatalog
}

func (r skillCatalogReaderStub) ReadSkills(ctx context.Context) ([]Skill, error) {
	return r.catalog.Skills, nil
}

func (r skillCatalogReaderStub) ReadSkillCatalog(context.Context) (SkillCatalog, error) {
	return r.catalog, nil
}

func TestNewManagementWritesCarryPayloads(t *testing.T) {
	writer := newFakeWriter()
	service, err := New(Config{Writer: writer})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()
	ctx := context.Background()

	if _, err := service.SetProjectHooksEnabled(ctx, "/work", true, "rev-1"); err != nil {
		t.Fatalf("项目钩子启停: %v", err)
	}
	if _, err := service.RefreshSkills(ctx); err != nil {
		t.Fatalf("技能刷新: %v", err)
	}
	if _, err := service.StartMCPAuth(ctx, MCPTarget{ID: "srv"}, "secret"); err != nil {
		t.Fatalf("MCP 认证开始: %v", err)
	}
	if _, err := service.CompleteMCPAuth(ctx, "attempt-1", "http://localhost/cb"); err != nil {
		t.Fatalf("MCP 认证完成: %v", err)
	}
	if _, err := service.CancelMCPAuth(ctx, "attempt-1"); err != nil {
		t.Fatalf("MCP 认证取消: %v", err)
	}
	if _, err := service.LogoutMCPAuth(ctx, MCPTarget{ID: "srv"}); err != nil {
		t.Fatalf("MCP 登出: %v", err)
	}

	intents := writer.intentList()
	wantOps := []WriteOp{
		OpHookSetEnabled, OpSkillsRefresh,
		OpMCPAuthStart, OpMCPAuthComplete, OpMCPAuthCancel, OpMCPAuthLogout,
	}
	if len(intents) != len(wantOps) {
		t.Fatalf("写意图条数不符: %d", len(intents))
	}
	for index, intent := range intents {
		managed, err := DecodeManagementWrite(intent.Params)
		if err != nil {
			t.Fatalf("解析第 %d 条管理写入: %v", index, err)
		}
		if managed.Op != wantOps[index] {
			t.Fatalf("第 %d 条操作不符: got %q want %q", index, managed.Op, wantOps[index])
		}
	}
	var project HooksProjectEnabledPayload
	managed, _ := DecodeManagementWrite(intents[0].Params)
	if err := json.Unmarshal(managed.Payload, &project); err != nil {
		t.Fatalf("解析项目钩子载荷: %v", err)
	}
	if project.Cwd != "/work" || !project.Enabled || project.ExpectedRevision != "rev-1" {
		t.Fatalf("项目钩子载荷不符: %+v", project)
	}
	var complete MCPAuthCompletePayload
	managed, _ = DecodeManagementWrite(intents[3].Params)
	if err := json.Unmarshal(managed.Payload, &complete); err != nil {
		t.Fatalf("解析认证完成载荷: %v", err)
	}
	if complete.AttemptID != "attempt-1" || complete.CallbackURL != "http://localhost/cb" {
		t.Fatalf("认证完成载荷不符: %+v", complete)
	}
}

func TestNewCapabilitiesGateOnHello(t *testing.T) {
	writer := newFakeWriter()
	service, err := New(Config{
		Writer:   writer,
		Features: stubFeatures{"hooks.set_enabled": true},
	})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()
	ctx := context.Background()

	checks := []struct {
		name string
		call func() error
	}{
		{"skills.refresh", func() error { _, err := service.RefreshSkills(ctx); return err }},
		{"skills.reload_catalog", func() error { _, err := service.ReloadSkillCatalog(ctx); return err }},
		{"mcp.auth.start", func() error {
			_, err := service.StartMCPAuth(ctx, MCPTarget{ID: "srv"}, "")
			return err
		}},
		{"mcp.auth.complete", func() error {
			_, err := service.CompleteMCPAuth(ctx, "attempt", "http://localhost/cb")
			return err
		}},
		{"mcp.auth.cancel", func() error { _, err := service.CancelMCPAuth(ctx, "attempt"); return err }},
		{"mcp.auth.logout", func() error {
			_, err := service.LogoutMCPAuth(ctx, MCPTarget{ID: "srv"})
			return err
		}},
	}
	for _, check := range checks {
		err := check.call()
		if !errors.Is(err, ErrCapabilityMissing) {
			t.Fatalf("%s 没有按 hello 门禁返回能力缺失: %v", check.name, err)
		}
		var capability *CapabilityError
		if !errors.As(err, &capability) || capability.Feature != check.name {
			t.Fatalf("%s 的能力错误没有带能力名: %v", check.name, err)
		}
	}
	if intents := writer.intentList(); len(intents) != 0 {
		t.Fatalf("hello 缺能力时仍登记了写意图: %+v", intents)
	}

	if _, err := service.SetProjectHooksEnabled(ctx, "/work", true, "rev"); err != nil {
		t.Fatalf("已声明能力被误挡: %v", err)
	}
	if intents := writer.intentList(); len(intents) != 1 {
		t.Fatalf("已声明能力没有登记写意图: %d", len(intents))
	}
}

func TestSkillCatalogVersionAndProjectHooksRead(t *testing.T) {
	service, err := New(Config{
		Skills: skillCatalogReaderStub{catalog: SkillCatalog{
			Version: 3,
			Skills:  []Skill{{ID: "s1"}},
			Issues:  []SkillIssue{{Path: "/p", Message: "m"}},
		}},
		ProjectHooks: projectHookReaderStub{catalog: ProjectHookCatalog{
			Hooks:            []Hook{{Event: "SessionStart", Source: SourceUser}},
			Sources:          []HookSourceState{{Source: SourceProject, Enabled: true, State: "ok"}},
			ProjectsRevision: "projects-2",
		}},
	})
	if err != nil {
		t.Fatalf("创建领域实例: %v", err)
	}
	defer service.Close()

	if _, err := service.ListSkills(context.Background()); err != nil {
		t.Fatalf("读取技能目录: %v", err)
	}
	if version := service.SkillCatalogVersion(); version != 3 {
		t.Fatalf("技能目录版本不符: %d", version)
	}
	if issues := service.SkillIssues(); len(issues) != 1 {
		t.Fatalf("技能问题清单不符: %+v", issues)
	}

	project, err := service.ReadProjectHooks(context.Background(), "/work")
	if err != nil {
		t.Fatalf("读取项目钩子: %v", err)
	}
	if project.ProjectsRevision != "projects-2" || len(project.Sources) != 1 ||
		!project.Sources[0].Enabled || project.Sources[0].Source != SourceProject {
		t.Fatalf("项目钩子读数不符: %+v", project)
	}
	if project.ReadAt.IsZero() {
		t.Fatal("项目钩子读数缺少读取时间")
	}
}
