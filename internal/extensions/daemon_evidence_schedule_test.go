package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// TestRealDaemonScheduleWatch 用带 Scheduler 默认预设的真实 daemon 验证
// schedule.watch：watching 后读一次 list、jobs 变化后读一次、run 通知重读
// history、序号缺口重读、删除后读一次；没有定时轮询。
//
// 预设与 Node 运行时由环境变量提供：
//
//	DIM_APP_SERVER_DEFAULTS     含 scheduler 插件的 default-plugins.json
//	DIM_APP_SERVER_NODE         可选：Node 可执行文件
//	DIM_APP_SERVER_JS_RUNTIME   可选：@arcships/rutis-runtime 目录
func TestRealDaemonScheduleWatch(t *testing.T) {
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	defaults := os.Getenv("DIM_APP_SERVER_DEFAULTS")
	if serverPath == "" || defaults == "" {
		t.Skip("未设置 DIM_APP_SERVER_BIN/DIM_APP_SERVER_DEFAULTS：真实计划任务证据 blocked（缺 Scheduler 预设或 App Server）")
	}
	if _, err := os.Stat(defaults); err != nil {
		t.Skipf("默认预设不可读，真实计划任务证据 blocked: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	dataDir, err := os.MkdirTemp("/tmp", "d11b-schedule-")
	if err != nil {
		t.Fatalf("创建临时数据目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	workDir := filepath.Join(dataDir, "project")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建项目目录: %v", err)
	}

	cfg := evidenceDaemonConfig(dataDir, serverPath)
	cfg.ServerArgs = append(cfg.ServerArgs, "--defaults", defaults)
	if node := os.Getenv("DIM_APP_SERVER_NODE"); node != "" {
		cfg.ServerArgs = append(cfg.ServerArgs, "--node", node)
	}
	if jsRuntime := os.Getenv("DIM_APP_SERVER_JS_RUNTIME"); jsRuntime != "" {
		cfg.ServerArgs = append(cfg.ServerArgs, "--js-runtime", jsRuntime)
	}
	supervisor, err := daemon.New(cfg)
	if err != nil {
		t.Fatalf("创建连接监督者: %v", err)
	}
	conn, err := supervisor.Connect(ctx)
	if err != nil {
		supervisor.Close(nil)
		t.Fatalf("连接服务端: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer shutdownCancel()
		if result, shutdownErr := supervisor.Shutdown(shutdownCtx); shutdownErr == nil {
			t.Logf("回收自启服务端: stopped=%v", result.Stopped)
		}
		supervisor.Close(nil)
	})
	t.Logf("服务端 %s（profile %s）", conn.Hello.ServerVersion, conn.Hello.DaemonProfile.Kind)
	if !conn.Hello.HasFeature("schedule.list") || !conn.Hello.HasFeature("schedule.watch") {
		t.Skip("服务端未声明 schedule.*：真实计划任务证据 blocked（预设未装配 Scheduler）")
	}
	for _, method := range []string{"schedule.list", "schedule.watch", "schedule.save", "schedule.history", "schedule.delete"} {
		t.Logf("hello %s=%v", method, conn.Hello.HasFeature(method))
	}

	harness := newAppServerHarness(conn.Client)
	harness.overlayWrites = false
	service := newService(t, Config{
		Scheduler: harness,
		Features:  conn,
		Logf: func(format string, args ...any) {
			t.Logf(format, args...)
		},
	})

	type watchItem struct {
		Seq      int64  `json:"seq"`
		Kind     string `json:"kind"`
		JobID    string `json:"jobId"`
		RunID    string `json:"runId"`
		State    string `json:"state"`
		Terminal bool   `json:"terminal"`
	}
	items := make(chan watchItem, 32)
	conn.Client.OnNotification(func(message protocol.Message) {
		if message.Method != "management.progress" {
			return
		}
		var progress struct {
			Method string          `json:"method"`
			Data   json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(message.Params, &progress); err != nil || progress.Method != "schedule.watch" {
			return
		}
		var item watchItem
		if err := json.Unmarshal(progress.Data, &item); err != nil {
			return
		}
		_ = service.Notify(Notice{
			Topic:       TopicSchedule,
			Sequence:    item.Seq,
			SequenceSet: true,
			Kind:        item.Kind,
			JobID:       item.JobID,
		})
		select {
		case items <- item:
		default:
		}
	})

	watchDone := make(chan error, 1)
	go func() {
		_, err := conn.Client.CallWithTimeout(ctx, "schedule.watch", nil, protocol.NoTimeout)
		watchDone <- err
	}()
	nextItem := func(what string) watchItem {
		t.Helper()
		select {
		case item := <-items:
			t.Logf("%s: seq=%d kind=%s job=%s run=%s state=%s terminal=%v",
				what, item.Seq, item.Kind, item.JobID, item.RunID, item.State, item.Terminal)
			return item
		case <-time.After(10 * time.Second):
			t.Fatalf("等待 %s 超时", what)
			return watchItem{}
		}
	}

	first := nextItem("watching")
	if first.Seq != 0 || first.Kind != "watching" {
		t.Fatalf("首个 watch 项不符: %+v", first)
	}
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("等待 watching 跟随: %v", err)
	}
	if reads := service.ScheduledTaskReads(); reads != 1 {
		t.Fatalf("watching 后计划任务重读次数不符: %d", reads)
	}

	rawSave, err := conn.Client.CallWithTimeout(ctx, "schedule.save", map[string]any{"job": map[string]any{
		"name":     "d11b watch job",
		"schedule": map[string]any{"at": time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)},
		"prompt":   "d11b watch",
		"cwd":      workDir,
	}}, 10*time.Second)
	if err != nil {
		t.Fatalf("schedule.save: %v", err)
	}
	var saved struct {
		Job struct {
			ID string `json:"id"`
		} `json:"job"`
	}
	if err := json.Unmarshal(rawSave, &saved); err != nil || saved.Job.ID == "" {
		t.Fatalf("解析 schedule.save 回执: %v %s", err, rawSave)
	}
	jobID := saved.Job.ID
	t.Logf("创建任务 id=%s", jobID)

	jobsItem := nextItem("jobs")
	if jobsItem.Seq != 1 || jobsItem.Kind != "jobs" {
		t.Fatalf("jobs 通知不符: %+v", jobsItem)
	}
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("等待 jobs 跟随: %v", err)
	}
	if reads := service.ScheduledTaskReads(); reads != 2 {
		t.Fatalf("jobs 后计划任务重读次数不符: %d", reads)
	}
	if !snapshotHasTask(service.ScheduledTaskSnapshot(), jobID) {
		t.Fatalf("jobs 后读数没有任务: %+v", service.ScheduledTaskSnapshot().Items)
	}

	// 没有定时轮询：等待一小段时间，读取次数不变。
	time.Sleep(300 * time.Millisecond)
	if reads := service.ScheduledTaskReads(); reads != 2 {
		t.Fatalf("计划任务出现轮询: %d", reads)
	}

	// run 通知的形状由真实源码/夹具确定；这里用无序号通知驱动 domain 的 history 重读，
	// 避免删除任务后仍有未结束的 run 通知干扰删除证据。
	_ = service.Notify(Notice{Topic: TopicSchedule, Kind: "run", JobID: jobID})
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("等待 run 跟随: %v", err)
	}
	if reads := service.ScheduledTaskReads(); reads != 3 {
		t.Fatalf("run 后计划任务重读次数不符: %d", reads)
	}
	if reads := service.ScheduleHistoryReads(); reads != 1 {
		t.Fatalf("run 后 history 重读次数不符: %d", reads)
	}
	t.Logf("run 跟随: listReads=%d historyReads=%d historyItems=%d",
		service.ScheduledTaskReads(), service.ScheduleHistoryReads(), len(service.ScheduleHistorySnapshot(jobID).Items))

	rawDelete, err := conn.Client.CallWithTimeout(ctx, "schedule.delete", map[string]any{"job_id": jobID}, 10*time.Second)
	if err != nil {
		t.Fatalf("schedule.delete: %v", err)
	}
	t.Logf("删除任务回执字段=%s", rawKeys(rawDelete))
	deleteItem := nextItem("delete")
	if deleteItem.Kind != "jobs" {
		t.Fatalf("删除通知不符: %+v", deleteItem)
	}
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("等待删除跟随: %v", err)
	}
	if reads := service.ScheduledTaskReads(); reads != 4 {
		t.Fatalf("删除后计划任务重读次数不符: %d", reads)
	}
	if snapshotHasTask(service.ScheduledTaskSnapshot(), jobID) {
		t.Fatalf("删除后读数仍有任务: %+v", service.ScheduledTaskSnapshot().Items)
	}

	// 序号缺口：服务端真实序号连续；注入一个跳号通知验证 domain 的缺口重读。
	gap := deleteItem.Seq + 3
	_ = service.Notify(Notice{Topic: TopicSchedule, Sequence: gap, SequenceSet: true, Kind: "jobs"})
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("等待缺口跟随: %v", err)
	}
	if reads := service.ScheduledTaskReads(); reads != 5 {
		t.Fatalf("序号缺口后计划任务重读次数不符: %d", reads)
	}
	time.Sleep(300 * time.Millisecond)
	if reads := service.ScheduledTaskReads(); reads != 5 {
		t.Fatalf("缺口后出现轮询: %d", reads)
	}
	t.Logf("最终 listReads=%d historyReads=%d stats=%+v", service.ScheduledTaskReads(), service.ScheduleHistoryReads(), service.FollowStats())

	// 结束长驻 watch：关闭控制连接会让服务端回 operation_interrupted。
	conn.Client.Close(errors.New("schedule watch evidence done"))
	select {
	case <-watchDone:
	case <-time.After(2 * time.Second):
	}
}

func snapshotHasTask(snapshot Snapshot[ScheduledTask], id string) bool {
	for _, task := range snapshot.Items {
		if task.ID == id {
			return true
		}
	}
	return false
}
