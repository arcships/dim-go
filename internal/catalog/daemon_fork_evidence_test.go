package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
	"github.com/arcships/dimagent-mygo/internal/projection"
)

// realEntryLookup 用 session.summaries 按标识读回一条目录条目。
type realEntryLookup struct {
	client *protocol.Client
}

func (l *realEntryLookup) ReadEntry(ctx context.Context, sessionID string) (Entry, bool, error) {
	raw, err := l.client.Call(ctx, "session.summaries", map[string]any{})
	if err != nil {
		return Entry{}, false, err
	}
	var summaries []struct {
		SessionID     string  `json:"sessionId"`
		Cwd           string  `json:"cwd"`
		CreatedAt     int64   `json:"createdAt"`
		UpdatedAt     int64   `json:"updatedAt"`
		Running       bool    `json:"running"`
		Title         *string `json:"title"`
		TitleSource   *string `json:"titleSource"`
		TitleRevision int64   `json:"titleRevision"`
	}
	if err := json.Unmarshal(raw, &summaries); err != nil {
		return Entry{}, false, fmt.Errorf("会话摘要无法解析: %w", err)
	}
	for _, summary := range summaries {
		if summary.SessionID != sessionID {
			continue
		}
		entry := Entry{
			ID:            summary.SessionID,
			Workspace:     summary.Cwd,
			CreatedAt:     time.UnixMilli(summary.CreatedAt).UTC(),
			UpdatedAt:     time.UnixMilli(summary.UpdatedAt).UTC(),
			Running:       summary.Running,
			TitleRevision: summary.TitleRevision,
		}
		if summary.Title != nil {
			entry.Title = *summary.Title
		}
		if summary.TitleSource != nil {
			entry.TitleSource = TitleSource(*summary.TitleSource)
		}
		return entry, true, nil
	}
	return Entry{}, false, nil
}

// TestRealDaemonFork 在真实 App Server 上从一轮结束处分叉：新会话进入目录、源会话
// 不变、按标识读回补齐字段；缺方法时按能力缺失跳过。
//
// 用法：
//
//	DIM_APP_SERVER_BIN=$HOME/.dimcode-alpha/rust/dimcode-server \
//	DIM_APP_SERVER_ARGS=app-server \
//	DIM_APP_SERVER_MODEL_CONFIG=/tmp/d4-offline.json \
//	  go test -run TestRealDaemonFork -v ./internal/catalog/
func TestRealDaemonFork(t *testing.T) {
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("未设置 DIM_APP_SERVER_BIN：真实 daemon 证据 blocked（缺 App Server 可执行文件）")
	}
	var serverArgs []string
	if args := os.Getenv("DIM_APP_SERVER_ARGS"); args != "" {
		serverArgs = append(serverArgs, args)
	}

	dataDir, err := os.MkdirTemp("/tmp", "d4fork-")
	if err != nil {
		t.Fatalf("创建临时数据目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	workDir := filepath.Join(dataDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
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
	writes := newWriteManager(t, harness)
	creator := newTestCatalog(t, Config{Reader: harness, Writes: writes})
	created, err := creator.Create(ctx, CreateRequest{Cwd: workDir})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("新建会话: %v", err)
	}

	// 先跑一轮真实对话，让投影给出轮结束处的切点。
	folder, err := projection.New(projection.Config{SessionID: created.ID})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("创建折叠器: %v", err)
	}
	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("创建会话流管理器: %v", err)
	}
	t.Cleanup(manager.Close)
	turnEnded := make(chan struct{}, 1)
	snapshot, lease, err := manager.Acquire(ctx, created.ID, session.Listener{
		OnEvent: func(event session.Event) {
			folder.ApplyListenerEvent(event)
			if isTurnEndNotice(event) {
				select {
				case turnEnded <- struct{}{}:
				default:
				}
			}
		},
	})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("打开会话: %v", err)
	}
	t.Cleanup(lease.Release)
	folder.ApplySnapshot(snapshot)
	submitOneTurn(t, ctx, conn.Client, created.ID, "第一轮：为分叉准备切点", turnEnded)
	point := folder.CutPoint("main", 1)
	if !point.Usable() {
		lease.Release()
		cleanupDaemon(t, supervisor)
		t.Fatalf("投影没有给出第 1 轮的切点：%+v", point)
	}
	t.Logf("投影切点：第 %d 轮结束于序号 %d（turn %s）", point.TurnIndex, point.Sequence, point.TurnID)

	source := &realBranchSource{client: conn.Client, sessionID: created.ID, folder: folder}
	directory := newTestCatalog(t, Config{
		Reader:   harness,
		Writes:   writes,
		Branches: source,
		Lookup:   &realEntryLookup{client: conn.Client},
	})
	if _, err := directory.Refresh(ctx, Query{}); err != nil {
		lease.Release()
		cleanupDaemon(t, supervisor)
		if capabilityMissing(err) {
			t.Skipf("服务端未提供 session.catalog：真实分叉证据 blocked（%v）", err)
		}
		t.Fatalf("读取目录: %v", err)
	}
	before, _ := directory.Entry(created.ID)

	forked, err := directory.Fork(ctx, ForkRequest{SessionID: created.ID, TurnIndex: 1})
	if err != nil {
		lease.Release()
		cleanupDaemon(t, supervisor)
		if capabilityMissing(err) {
			t.Skipf("服务端未提供 session.fork：真实分叉证据 blocked（%v）", err)
		}
		t.Fatalf("分叉: %v", err)
	}
	t.Logf("分叉会话：id=%s title=%q titleSource=%q titleRevision=%d workspace=%q incomplete=%v",
		forked.ID, forked.Title, forked.TitleSource, forked.TitleRevision, forked.Workspace, forked.Incomplete)
	if forked.ID == created.ID {
		lease.Release()
		cleanupDaemon(t, supervisor)
		t.Fatal("分叉返回了源会话标识")
	}
	if forked.Incomplete {
		lease.Release()
		cleanupDaemon(t, supervisor)
		t.Fatalf("单条读回没有补齐分叉条目：%+v", forked)
	}
	if forked.Workspace == "" {
		lease.Release()
		cleanupDaemon(t, supervisor)
		t.Fatalf("分叉条目缺少工作目录：%+v", forked)
	}
	if _, ok := directory.Entry(forked.ID); !ok {
		lease.Release()
		cleanupDaemon(t, supervisor)
		t.Fatal("分叉会话没有并入目录")
	}
	after, _ := directory.Entry(created.ID)
	if after.Title != before.Title || after.TitleRevision != before.TitleRevision {
		lease.Release()
		cleanupDaemon(t, supervisor)
		t.Fatalf("分叉改动了源会话条目：%+v → %+v", before, after)
	}

	// 重读目录能看到源会话与分叉会话，且分叉会话的标识、工作目录一致。
	if _, err := directory.Refresh(ctx, Query{}); err != nil {
		lease.Release()
		cleanupDaemon(t, supervisor)
		t.Fatalf("重读目录: %v", err)
	}
	refreshed := directory.Snapshot()
	var foundSource, foundFork bool
	for _, entry := range refreshed.Entries {
		switch entry.ID {
		case created.ID:
			foundSource = true
		case forked.ID:
			foundFork = true
		}
	}
	if !foundSource || !foundFork {
		lease.Release()
		cleanupDaemon(t, supervisor)
		t.Fatalf("重读后目录缺少会话：%+v", entryIDs(refreshed.Entries))
	}

	// 释放源会话订阅后清理两个会话。
	lease.Release()
	if err := directory.Delete(ctx, DeleteRequest{SessionID: forked.ID}); err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("删除分叉会话: %v", err)
	}
	if err := directory.Delete(ctx, DeleteRequest{SessionID: created.ID}); err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("删除源会话: %v", err)
	}
	cleanupDaemon(t, supervisor)
}
