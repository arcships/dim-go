package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestRealDaemonExtensionReadsAndWakeup 在真实 App Server（alpha.21 或同版本）上
// 核对管理面读取、工作流能力缺失与唤醒的读取/取消/通知跟随；插件与技能写入在
// 读取证据完成后单独尝试，失败按 blocked 记录，不影响读取结论。
//
// 没有真实服务端时跳过：证据标 blocked，不用替身冒充。
//
// 用法：
//
//	DIM_APP_SERVER_BIN=<app-server-binary> \
//	DIM_APP_SERVER_ARGS=app-server \
//	  go test -run TestRealDaemonExtensionReadsAndWakeup -v ./internal/extensions/
//
// 可选 DIM_APP_SERVER_MODEL_CONFIG 指向离线模型配置；未设置时使用 mock 模型。
func TestRealDaemonExtensionReadsAndWakeup(t *testing.T) {
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("未设置 DIM_APP_SERVER_BIN：真实管理面证据 blocked（缺 App Server 可执行文件）")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dataDir, err := os.MkdirTemp("/tmp", "d11b-ext-")
	if err != nil {
		t.Fatalf("创建临时数据目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	workDir := filepath.Join(dataDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}

	supervisor, err := daemon.New(evidenceDaemonConfig(dataDir, serverPath))
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
			t.Logf("回收自启服务端: stopped=%v reason=%s", result.Stopped, result.Reason)
		}
		supervisor.Close(nil)
	})
	t.Logf("服务端 %s（protocol %d，profile %s，owned=%v）",
		conn.Hello.ServerVersion, conn.Hello.ProtocolVersion, conn.Hello.DaemonProfile.Kind, conn.Owned)

	harness := newAppServerHarness(conn.Client)
	// 真机直接读服务端事实，不叠加测试进程里的写入结果。
	harness.overlayWrites = false
	writes := newWriteManager(t, harness)
	service := newService(t, Config{
		Plugins:     harness,
		Skills:      harness,
		Hooks:       harness,
		ToolServers: harness,
		Scheduler:   harness,
		Wakeups:     harness,
		Writer:      writes,
		Logf: func(format string, args ...any) {
			t.Logf(format, args...)
		},
	})

	// 1. 管理面读取：方法名全部按 alpha.21 实测。
	plugins, err := service.ListPlugins(ctx)
	if err != nil {
		t.Fatalf("读取插件: %v", err)
	}
	sample := make([]string, 0, 5)
	for index, plugin := range plugins.Items {
		if index >= 5 {
			break
		}
		sample = append(sample, plugin.ID)
	}
	t.Logf("插件 %d 个（样例 %v）", len(plugins.Items), sample)

	skills, err := service.ListSkills(ctx)
	if err != nil {
		t.Fatalf("读取技能: %v", err)
	}
	t.Logf("技能 %d 个", len(skills.Items))

	hooks, err := service.ListHooks(ctx)
	if err != nil {
		t.Fatalf("读取钩子: %v", err)
	}
	t.Logf("钩子 %d 条", len(hooks.Items))

	servers, err := service.ListToolServers(ctx)
	if err != nil {
		t.Fatalf("读取 MCP 目录: %v", err)
	}
	t.Logf("MCP 服务器 %d 个", len(servers.Items))

	tasks, err := service.ListScheduledTasks(ctx)
	if err != nil {
		t.Fatalf("读取计划任务: %v", err)
	}
	t.Logf("计划任务 %d 条", len(tasks.Items))

	// 2. workflow.* 在 alpha.21 不注册：保留类型，读取与写都按能力缺失且零请求。
	if _, err := service.ListWorkflows(ctx); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("工作流读取没有按能力缺失: %v", err)
	}
	before := len(writes.Table())
	if _, err := service.StartWorkflow(ctx, "wf-1", nil); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("工作流启动没有按能力缺失: %v", err)
	}
	if after := len(writes.Table()); after != before {
		t.Fatalf("工作流能力缺失时登记了写意图：%d -> %d", before, after)
	}

	// 3. 唤醒：创建（测试夹具直接调真实会话方法）、读取、取消、通知后重读一次。
	var created protocol.SessionCreateResult
	if err := conn.Client.CallInto(ctx, "session.create", map[string]any{"cwd": workDir}, &created); err != nil {
		t.Fatalf("创建会话: %v", err)
	}
	t.Logf("证据会话 %s", created.SessionID)

	streams, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	t.Cleanup(streams.Close)

	arrived := make(chan protocol.WakeupNoticeData, 4)
	_, lease, err := streams.Acquire(ctx, created.SessionID, session.Listener{
		OnEvent: func(event session.Event) {
			if event.Wakeup == nil {
				return
			}
			_ = service.Notify(Notice{
				Topic:     TopicWakeups,
				SessionID: created.SessionID,
				BranchID:  event.Wakeup.BranchID,
			})
			select {
			case arrived <- *event.Wakeup:
			default:
			}
		},
	})
	if err != nil {
		t.Fatalf("打开会话流: %v", err)
	}
	t.Cleanup(lease.Release)
	subscription := lease.Subscription()
	harness.sessionCall = subscription.Call

	at := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	raw, err := subscription.Call(ctx, "session.wakeup.create", map[string]any{
		"sessionId": created.SessionID,
		"branchId":  "main",
		"at":        at,
		"prompt":    "d11b evidence",
	})
	if err != nil {
		t.Fatalf("创建唤醒记录: %v", err)
	}
	var createdWakeup protocol.SessionWakeupCreateResult
	if err := json.Unmarshal(raw, &createdWakeup); err != nil || createdWakeup.WakeupID == "" {
		t.Fatalf("解析唤醒创建回执: %v %s", err, raw)
	}

	scope := WakeupScope{SessionID: created.SessionID, BranchID: "main"}
	snapshot, err := service.ListWakeups(ctx, scope)
	if err != nil {
		t.Fatalf("读取唤醒: %v", err)
	}
	if !hasWakeup(snapshot.Items, createdWakeup.WakeupID) {
		t.Fatalf("读取唤醒没有刚创建的记录: %+v", snapshot.Items)
	}
	t.Logf("唤醒读取：%d 条（%s）", len(snapshot.Items), createdWakeup.WakeupID)

	view, err := service.CancelWakeup(ctx, WakeupRef{
		SessionID: created.SessionID,
		BranchID:  "main",
		WakeupID:  createdWakeup.WakeupID,
	})
	if err != nil || view.Status != write.StatusSucceeded {
		t.Fatalf("取消唤醒: %v %+v", err, view)
	}
	noticeDeadline := time.After(5 * time.Second)
	observedCancel := false
	for !observedCancel {
		select {
		case notice := <-arrived:
			t.Logf("唤醒通知：state=%s", notice.State)
			if notice.State == "cancelled" {
				observedCancel = true
			}
		case <-noticeDeadline:
			t.Log("5 秒内未观察到 cancelled 通知；继续用显式读取核对取消结果")
			observedCancel = true
		}
	}
	if err := service.WaitIdle(ctx); err != nil {
		t.Fatalf("等待唤醒跟随: %v", err)
	}
	snapshot, err = service.ListWakeups(ctx, scope)
	if err != nil {
		t.Fatalf("取消后读取唤醒: %v", err)
	}
	if hasWakeup(snapshot.Items, createdWakeup.WakeupID) {
		t.Fatalf("取消后唤醒记录仍在: %+v", snapshot.Items)
	}
	stats := service.FollowStats()
	t.Logf("唤醒重读计数：%v", stats.WakeupReads)

	// 4. 写入尝试放在最后：插件启停可能触发服务端整体 reload，不污染读取证据。
	pluginAttempted := false
	for index := range plugins.Items {
		plugin := plugins.Items[index]
		if plugin.ID != "Attachments" || plugin.Locked {
			continue
		}
		pluginAttempted = true
		view, err := service.SetPluginEnabled(ctx, plugin.ID, !plugin.Enabled)
		if err != nil || view.Status != write.StatusSucceeded {
			t.Logf("插件启停 blocked：%s -> %v: %v %+v", plugin.ID, !plugin.Enabled, err, view)
			break
		}
		restore, restoreErr := service.SetPluginEnabled(ctx, plugin.ID, plugin.Enabled)
		if restoreErr != nil || restore.Status != write.StatusSucceeded {
			t.Logf("插件恢复 blocked：%s: %v %+v", plugin.ID, restoreErr, restore)
			break
		}
		t.Logf("插件 %s 启停往返成功：enable=%v，恢复=%s", plugin.ID, !plugin.Enabled, restore.Status)
		break
	}
	if !pluginAttempted {
		t.Log("插件启停 blocked：没有找到可安全切换的 Attachments")
	}

	if len(skills.Items) > 0 {
		skill := skills.Items[0]
		view, err := service.SetSkillEnabled(ctx, skill.ID, !skill.Enabled)
		if err != nil || view.Status != write.StatusSucceeded {
			t.Logf("技能启停 blocked：%s: %v %+v", skill.ID, err, view)
		} else {
			restore, restoreErr := service.SetSkillEnabled(ctx, skill.ID, skill.Enabled)
			if restoreErr != nil || restore.Status != write.StatusSucceeded {
				t.Logf("技能恢复 blocked：%s: %v %+v", skill.ID, restoreErr, restore)
			} else {
				t.Logf("技能 %s 启停往返成功：enable=%v，恢复=%s", skill.ID, !skill.Enabled, restore.Status)
			}
		}
	}

	// 让真实服务端知道测试结束：释放订阅后回收自启实例。
	lease.Release()
}

func hasWakeup(wakeups []Wakeup, id string) bool {
	for _, wakeup := range wakeups {
		if wakeup.ID == id {
			return true
		}
	}
	return false
}

// evidenceDaemonConfig 组装证据用配置：模型配置从 /tmp 传入或使用 mock 模型。
func evidenceDaemonConfig(dataDir, serverPath string) daemon.Config {
	cfg := daemon.Config{
		DataDir:    dataDir,
		ServerPath: serverPath,
		ServerArgs: strings.Fields(os.Getenv("DIM_APP_SERVER_ARGS")),
	}
	if modelConfig := os.Getenv("DIM_APP_SERVER_MODEL_CONFIG"); modelConfig != "" {
		cfg.ModelConfig = modelConfig
		return cfg
	}
	cfg.MockModel = true
	return cfg
}
