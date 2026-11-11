package catalog

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
)

// TestRealDaemonCatalogLifecycle 在真实 App Server 上核对目录的读写闭环：空目录、
// 新建、重读后名称一致、改名后重读一致、删除后目录里消失。
//
// 没有真实服务端时跳过：证据标 blocked，不用替身冒充。
//
// 用法：
//
//	DIM_APP_SERVER_BIN=<app-server-bin> \
//	  go test -run TestRealDaemonCatalogLifecycle -v ./internal/catalog/
//
// 需要子命令或额外参数的服务端，用 DIM_APP_SERVER_ARGS 传入。
func TestRealDaemonCatalogLifecycle(t *testing.T) {
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("未设置 DIM_APP_SERVER_BIN：真实 daemon 证据 blocked（缺 App Server 可执行文件）")
	}
	var serverArgs []string
	if args := os.Getenv("DIM_APP_SERVER_ARGS"); args != "" {
		serverArgs = append(serverArgs, args)
	}

	dataDir, err := os.MkdirTemp("", "dimagent-catalog-lifecycle-")
	if err != nil {
		t.Fatalf("创建临时数据目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	workDir := filepath.Join(dataDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}
	// 服务端可能返回解析后的路径；比较前先解析符号链接。
	resolvedWorkDir, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		t.Fatalf("解析工作目录: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	supervisor, err := daemon.New(evidenceDaemonConfig(dataDir, serverPath, serverArgs))
	if err != nil {
		t.Fatalf("创建监督者: %v", err)
	}
	conn, err := supervisor.Connect(ctx)
	if err != nil {
		t.Fatalf("连接服务端: %v", err)
	}
	t.Cleanup(func() { supervisor.Close(nil) })
	t.Logf("服务端 %s（protocol %d，profile %s，owned=%v）",
		conn.Hello.ServerVersion, conn.Hello.ProtocolVersion, conn.Hello.DaemonProfile.Kind, conn.Owned)

	harness := &appServerHarness{client: conn.Client}
	catalog := newTestCatalog(t, Config{
		Reader: harness,
		Writes: newWriteManager(t, harness),
	})

	if _, err := catalog.Refresh(ctx, Query{}); err != nil {
		cleanupDaemon(t, supervisor)
		if capabilityMissing(err) {
			t.Skipf("服务端未提供 session.catalog：真实目录证据 blocked（%v）", err)
		}
		t.Fatalf("读取目录: %v", err)
	}
	if entries := catalog.Snapshot().Entries; len(entries) != 0 {
		t.Fatalf("空数据目录不应有条目：%+v", entries)
	}

	created, err := catalog.Create(ctx, CreateRequest{Cwd: workDir})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("新建会话: %v", err)
	}
	t.Logf("新建会话 %s", created.ID)

	if _, err := catalog.Refresh(ctx, Query{}); err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("重读目录: %v", err)
	}
	entry, ok := catalog.Entry(created.ID)
	if !ok {
		cleanupDaemon(t, supervisor)
		t.Fatalf("重读后目录里没有新建的会话 %s", created.ID)
	}
	if entry.Workspace != resolvedWorkDir {
		cleanupDaemon(t, supervisor)
		t.Fatalf("目录里的工作目录不符：got %q，want %q", entry.Workspace, resolvedWorkDir)
	}
	t.Logf("重读条目：title=%q，titleRevision=%d，workspace=%q", entry.Title, entry.TitleRevision, entry.Workspace)

	// 打开会话流，接收改名触发的标题通知；通知按修订号合并进目录。
	titleNotices := make(chan TitleUpdate, 8)
	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("创建会话流管理器: %v", err)
	}
	t.Cleanup(manager.Close)
	_, lease, err := manager.Acquire(ctx, created.ID, session.Listener{
		OnEvent: func(event session.Event) {
			update, ok := decodeTitleNotice(event, created.ID)
			if !ok {
				return
			}
			catalog.ApplyTitle(update)
			select {
			case titleNotices <- update:
			default:
			}
		},
	})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("打开会话: %v", err)
	}
	t.Cleanup(lease.Release)

	// 改名按服务端事实收敛：支持时本地标题跟服务端回执一致，并核对标题通知；
	// 服务端没有该方法时明确报缺失并保留本地行，不假装成功。
	renamed, renameErr := catalog.Rename(ctx, RenameRequest{SessionID: created.ID, Title: "目录证据标题"})
	renameSupported := renameErr == nil
	switch {
	case renameSupported:
		if renamed.Title != "目录证据标题" {
			cleanupDaemon(t, supervisor)
			t.Fatalf("改名后的标题不符：%q", renamed.Title)
		}
		if _, err := catalog.Refresh(ctx, Query{}); err != nil {
			cleanupDaemon(t, supervisor)
			t.Fatalf("改名后重读目录: %v", err)
		}
		reloaded, _ := catalog.Entry(created.ID)
		if reloaded.Title != "目录证据标题" {
			cleanupDaemon(t, supervisor)
			t.Fatalf("重读后的标题不符：%q", reloaded.Title)
		}
		t.Logf("改名后重读：title=%q，titleRevision=%d，titleSource=%q",
			reloaded.Title, reloaded.TitleRevision, reloaded.TitleSource)

		// 服务端的标题通知落到同一修订号；本地状态与通知一致，旧修订号不覆盖。
		select {
		case notice := <-titleNotices:
			if notice.SessionID != created.ID || notice.Title != renamed.Title ||
				notice.Revision != renamed.TitleRevision {
				cleanupDaemon(t, supervisor)
				t.Fatalf("标题通知与改名结果不一致：notice=%+v rename=%+v", notice, renamed)
			}
			current, _ := catalog.Entry(created.ID)
			if current.Title != notice.Title || current.TitleRevision != notice.Revision {
				cleanupDaemon(t, supervisor)
				t.Fatalf("通知合并后本地状态不一致：%+v", current)
			}
			if catalog.ApplyTitle(TitleUpdate{
				SessionID: created.ID,
				Title:     "更旧的标题",
				Source:    TitleSourceAuto,
				Revision:  notice.Revision - 1,
			}) {
				cleanupDaemon(t, supervisor)
				t.Fatal("更低修订号的标题通知覆盖了本地状态")
			}
			t.Logf("标题通知：title=%q，revision=%d，source=%q，本地一致",
				notice.Title, notice.Revision, notice.Source)
		case <-time.After(2 * time.Second):
			cleanupDaemon(t, supervisor)
			t.Fatal("改名后没有收到标题通知")
		}

		// 非法标题：服务端明确拒绝，本地行保持改名后的状态。
		beforeReject, _ := catalog.Entry(created.ID)
		_, rejectErr := catalog.Rename(ctx, RenameRequest{
			SessionID: created.ID,
			Title:     "第一行\n第二行",
		})
		if !errors.Is(rejectErr, ErrRejected) {
			cleanupDaemon(t, supervisor)
			t.Fatalf("非法标题应被拒绝：%v", rejectErr)
		}
		if !strings.Contains(rejectErr.Error(), "Invalid title") {
			cleanupDaemon(t, supervisor)
			t.Fatalf("拒绝原因没有说明非法标题：%v", rejectErr)
		}
		afterReject, _ := catalog.Entry(created.ID)
		if afterReject.Title != beforeReject.Title || afterReject.TitleRevision != beforeReject.TitleRevision {
			cleanupDaemon(t, supervisor)
			t.Fatalf("被拒的改名改动了本地标题：%+v → %+v", beforeReject, afterReject)
		}
		t.Logf("非法标题被拒：%v", rejectErr)
	case errors.Is(renameErr, ErrRejected):
		// 该方法不可用的服务端版本：记为 blocked，并确认本地行未被改动。
		t.Logf("改名 blocked：服务端未提供该方法（%v）", renameErr)
		current, _ := catalog.Entry(created.ID)
		if current.Title != entry.Title {
			cleanupDaemon(t, supervisor)
			t.Fatalf("被拒的改名改动了本地标题：%q → %q", entry.Title, current.Title)
		}
	default:
		cleanupDaemon(t, supervisor)
		t.Fatalf("改名: %v", renameErr)
	}

	// 删除前先释放订阅。
	lease.Release()
	if err := catalog.Delete(ctx, DeleteRequest{SessionID: created.ID}); err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("删除会话: %v", err)
	}
	if _, ok := catalog.Entry(created.ID); ok {
		cleanupDaemon(t, supervisor)
		t.Fatal("删除确认后本地行仍在")
	}
	if _, err := catalog.Refresh(ctx, Query{}); err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("删除后重读目录: %v", err)
	}
	if _, ok := catalog.Entry(created.ID); ok {
		cleanupDaemon(t, supervisor)
		t.Fatal("删除后重读仍能看到该会话")
	}

	// 释放持有者后回收自启实例；失败即报。
	cleanupDaemon(t, supervisor)
}

// decodeTitleNotice 解出 SessionStore 的标题通知，并校验属于目标会话。
func decodeTitleNotice(event session.Event, sessionID string) (TitleUpdate, bool) {
	if event.Method != "session.notice" {
		return TitleUpdate{}, false
	}
	var notice struct {
		SessionID string `json:"sessionId"`
		Source    string `json:"source"`
		Kind      string `json:"kind"`
		Data      struct {
			Title         *string `json:"title"`
			TitleSource   string  `json:"titleSource"`
			TitleRevision int64   `json:"titleRevision"`
		} `json:"data"`
	}
	if err := json.Unmarshal(event.Params, &notice); err != nil {
		return TitleUpdate{}, false
	}
	if notice.SessionID != sessionID || notice.Source != "SessionStore" || notice.Kind != "title" {
		return TitleUpdate{}, false
	}
	title := ""
	if notice.Data.Title != nil {
		title = *notice.Data.Title
	}
	return TitleUpdate{
		SessionID: notice.SessionID,
		Title:     title,
		Source:    TitleSource(notice.Data.TitleSource),
		Revision:  notice.Data.TitleRevision,
	}, true
}

// evidenceDaemonConfig 组装证据用的服务端配置：给了离线模型配置就用它，
// 否则退回 mock 模型（夹具与配置只放临时目录，不改来源目录）。
func evidenceDaemonConfig(dataDir, serverPath string, serverArgs []string) daemon.Config {
	cfg := daemon.Config{DataDir: dataDir, ServerPath: serverPath, ServerArgs: serverArgs}
	if modelConfig := os.Getenv("DIM_APP_SERVER_MODEL_CONFIG"); modelConfig != "" {
		cfg.ModelConfig = modelConfig
		return cfg
	}
	cfg.MockModel = true
	return cfg
}

// evidenceMockDaemonConfig 用于需要真实模型回应的证据（导出、分支）：固定 mock 模型。
func evidenceMockDaemonConfig(dataDir, serverPath string, serverArgs []string) daemon.Config {
	return daemon.Config{DataDir: dataDir, ServerPath: serverPath, ServerArgs: serverArgs, MockModel: true}
}

// cleanupDaemon 关闭控制连接并回收自启实例。
func cleanupDaemon(t *testing.T, supervisor *daemon.Supervisor) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := supervisor.Shutdown(ctx)
	if err != nil {
		t.Fatalf("回收自启服务端: %v", err)
	}
	t.Logf("回收自启服务端: stopped=%v", result.Stopped)
}

// capabilityMissing 判断错误是否是服务端明确的能力缺失（unknown_method）。
func capabilityMissing(err error) bool {
	var rpcErr *protocol.RPCError
	return errors.As(err, &rpcErr) && rpcErr.Code == -32601
}
