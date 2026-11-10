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

// realBranchSource 是证据用的分支来源：目录读服务端的 session.branches，切点由
// 会话投影的 CutPoint 给出（不另建折叠）。
type realBranchSource struct {
	client    *protocol.Client
	sessionID string
	folder    *projection.Folder
}

func (s *realBranchSource) ReadBranches(
	ctx context.Context,
	sessionID string,
) ([]Branch, bool, error) {
	raw, err := s.client.Call(ctx, "session.branches", map[string]any{"sessionId": sessionID})
	if err != nil {
		return nil, false, err
	}
	var result struct {
		Branches []struct {
			BranchID       string `json:"branchId"`
			ParentBranchID string `json:"parentBranchId"`
			BranchPoint    *int64 `json:"branchPoint"`
			State          string `json:"state"`
			Head           int64  `json:"head"`
		} `json:"branches"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, false, fmt.Errorf("分支目录回执无法解析: %w", err)
	}
	branches := make([]Branch, 0, len(result.Branches))
	for _, item := range result.Branches {
		branch := Branch{
			ID:       item.BranchID,
			ParentID: item.ParentBranchID,
			Status:   BranchStatus(item.State),
			Head:     item.Head,
		}
		if item.BranchPoint != nil {
			branch.CutSequence = *item.BranchPoint
		}
		branches = append(branches, branch)
	}
	return branches, true, nil
}

func (s *realBranchSource) TurnCut(sessionID, branchID string, turnIndex int) CutPoint {
	if sessionID != "" && sessionID != s.sessionID {
		return CutPoint{Status: CutNoBranch, TurnIndex: turnIndex}
	}
	return s.folder.CutPoint(branchID, turnIndex)
}

// TestRealDaemonBranchLifecycle 在真实 App Server 上核对分支创建与结束：创建后
// 目录出现新分支、主线不变；结束后目录收敛为 ended 且头部固定。
//
// 用法：
//
//	DIM_APP_SERVER_BIN=/tmp/dim-2-target/debug/dim-app-server \
//	  go test -run TestRealDaemonBranchLifecycle -v ./internal/catalog/
func TestRealDaemonBranchLifecycle(t *testing.T) {
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("未设置 DIM_APP_SERVER_BIN：真实 daemon 证据 blocked（缺 App Server 可执行文件）")
	}
	var serverArgs []string
	if args := os.Getenv("DIM_APP_SERVER_ARGS"); args != "" {
		serverArgs = append(serverArgs, args)
	}

	dataDir, err := os.MkdirTemp("/tmp", "d4branch-")
	if err != nil {
		t.Fatalf("创建临时数据目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	workDir := filepath.Join(dataDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
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
	writes := newWriteManager(t, harness)
	creator := newTestCatalog(t, Config{Reader: harness, Writes: writes})
	created, err := creator.Create(ctx, CreateRequest{Cwd: workDir})
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("新建会话: %v", err)
	}

	// 先跑一轮真实对话，让投影有可用的轮结束切点。
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
	submitOneTurn(t, ctx, conn.Client, created.ID, "第一轮：为分支准备切点", turnEnded)
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
	})
	before, err := directory.RefreshBranches(ctx, created.ID)
	if err != nil {
		cleanupDaemon(t, supervisor)
		if capabilityMissing(err) {
			t.Skipf("服务端未提供 session.branches：真实分支证据 blocked（%v）", err)
		}
		t.Fatalf("读取分支目录: %v", err)
	}
	t.Logf("初始目录: %+v", before.Branches)

	branch, err := directory.CreateBranch(ctx, BranchCreateRequest{
		SessionID:      created.ID,
		ParentBranchID: "main",
		TurnIndex:      1,
		Prompt:         "真实分支证据",
	})
	if err != nil {
		cleanupDaemon(t, supervisor)
		if capabilityMissing(err) {
			t.Skipf("服务端未提供 session.branch_create：真实分支证据 blocked（%v）", err)
		}
		t.Fatalf("创建分支: %v", err)
	}
	t.Logf("新分支: %+v", branch)

	merged := directory.Branches(created.ID)
	if len(merged.Branches) != len(before.Branches)+1 {
		cleanupDaemon(t, supervisor)
		t.Fatalf("新分支没有并入目录：%+v", merged.Branches)
	}
	for index, old := range before.Branches {
		if merged.Branches[index] != old {
			cleanupDaemon(t, supervisor)
			t.Fatalf("已有分支被改动：%+v → %+v", old, merged.Branches[index])
		}
	}

	// 再读一次服务端目录：新分支存在、状态 open。
	refreshed, err := directory.RefreshBranches(ctx, created.ID)
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("重读分支目录: %v", err)
	}
	if len(refreshed.Branches) != len(before.Branches)+1 {
		cleanupDaemon(t, supervisor)
		t.Fatalf("重读后分支数不符：%+v", refreshed.Branches)
	}

	ended, err := directory.EndBranch(ctx, BranchEndRequest{SessionID: created.ID, BranchID: branch.ID})
	if err != nil {
		cleanupDaemon(t, supervisor)
		if capabilityMissing(err) {
			t.Skipf("服务端未提供 session.branch_end：真实分支证据 blocked（%v）", err)
		}
		t.Fatalf("结束分支: %v", err)
	}
	if ended.Status != BranchEnding {
		cleanupDaemon(t, supervisor)
		t.Fatalf("结束受理后的状态不符：%+v", ended)
	}

	converged, err := directory.RefreshBranches(ctx, created.ID)
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("结束后重读分支目录: %v", err)
	}
	endedBranch, ok := findBranch(converged.Branches, branch.ID)
	if !ok || endedBranch.Status != BranchEnded || endedBranch.Head <= 0 {
		cleanupDaemon(t, supervisor)
		t.Fatalf("分支没有收敛为已结束：%+v", converged.Branches)
	}
	// 已结束分支的头部固定：再读一次不变。
	again, err := directory.RefreshBranches(ctx, created.ID)
	if err != nil {
		cleanupDaemon(t, supervisor)
		t.Fatalf("再次重读分支目录: %v", err)
	}
	againBranch, ok := findBranch(again.Branches, branch.ID)
	if !ok || againBranch.Head != endedBranch.Head || againBranch.Status != BranchEnded {
		cleanupDaemon(t, supervisor)
		t.Fatalf("已结束分支头部不固定：%+v → %+v", endedBranch, againBranch)
	}
	t.Logf("分支收敛：state=%s head=%d", againBranch.Status, againBranch.Head)

	lease.Release()
	cleanupDaemon(t, supervisor)
}

// findBranch 在目录里查找一个分支。
func findBranch(branches []Branch, branchID string) (Branch, bool) {
	for _, branch := range branches {
		if branch.ID == branchID {
			return branch, true
		}
	}
	return Branch{}, false
}
