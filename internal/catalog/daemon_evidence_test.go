package catalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// TestRealDaemonCatalogLifecycle 在真实 App Server 上核对目录的读写闭环：空目录、
// 新建、重读后名称一致、改名后重读一致、删除后目录里消失。
//
// 没有真实服务端时跳过：证据标 blocked，不用替身冒充。
//
// 用法：
//
//	DIM_APP_SERVER_BIN=/tmp/dim-2-target/debug/dim-app-server \
//	  go test -run TestRealDaemonCatalogLifecycle -v ./internal/catalog/
//
// 老版本服务端需要 `app-server` 子命令，可用 DIM_APP_SERVER_ARGS=app-server 传入。
func TestRealDaemonCatalogLifecycle(t *testing.T) {
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("未设置 DIM_APP_SERVER_BIN：真实 daemon 证据 blocked（缺 App Server 可执行文件）")
	}
	var serverArgs []string
	if args := os.Getenv("DIM_APP_SERVER_ARGS"); args != "" {
		serverArgs = append(serverArgs, args)
	}

	dataDir, err := os.MkdirTemp("/tmp", "d4cat-")
	if err != nil {
		t.Fatalf("创建临时数据目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	workDir := filepath.Join(dataDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}
	// macOS 的 /tmp 是指向 /private/tmp 的符号链接；服务端会返回解析后的路径。
	resolvedWorkDir, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		t.Fatalf("解析工作目录: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	supervisor, err := daemon.New(daemon.Config{
		DataDir:    dataDir,
		ServerPath: serverPath,
		ServerArgs: serverArgs,
		MockModel:  true,
	})
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

	// 改名按服务端事实收敛：支持时本地标题跟服务端回执一致；服务端没有该方法时
	// 明确报缺失并保留本地行，不假装成功。
	renamed, err := catalog.Rename(ctx, RenameRequest{SessionID: created.ID, Title: "目录证据标题"})
	switch {
	case err == nil:
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
	case errors.Is(err, ErrRejected):
		// 本机两个服务端版本都未提供该方法：记为 blocked，并确认本地行未被改动。
		t.Logf("改名 blocked：服务端未提供该方法（%v）", err)
		current, _ := catalog.Entry(created.ID)
		if current.Title != entry.Title {
			cleanupDaemon(t, supervisor)
			t.Fatalf("被拒的改名改动了本地标题：%q → %q", entry.Title, current.Title)
		}
	default:
		cleanupDaemon(t, supervisor)
		t.Fatalf("改名: %v", err)
	}

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
