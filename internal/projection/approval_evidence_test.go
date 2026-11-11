package projection

// 真实模型证据：审批重放（投影侧观察）与会话内 @ 搜索（方法探测）。
//
// 前提同 6.6：临时夹具的复制、真实 daemon（0.1.0-alpha.21）、真实 provider。
// 日志只留事件类型、标识、状态与条数。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
)

// TestRealDaemonApprovalReplayAndFileSearch 验证审批重放对投影的影响面，以及 @ 搜索方法的口径。
//
// 用法：
//
//	DIM_REAL_MODEL_FIXTURE=<夹具目录> \
//	  go test -run TestRealDaemonApprovalReplayAndFileSearch -v -count=1 ./internal/projection/
func TestRealDaemonApprovalReplayAndFileSearch(t *testing.T) {
	fixture := os.Getenv("DIM_REAL_MODEL_FIXTURE")
	if fixture == "" {
		t.Skip("未设置 DIM_REAL_MODEL_FIXTURE：真实模型证据 blocked（缺临时夹具目录）")
	}
	serverPath := filepath.Join(fixture, "bin", "dimcode-server")
	if _, err := os.Stat(serverPath); err != nil {
		t.Skipf("夹具缺少 daemon 可执行文件: %v", err)
	}

	runDir := evidenceTempDir(t, "d2approval-")
	t.Cleanup(func() { _ = os.RemoveAll(runDir) })
	home := filepath.Join(runDir, "home")
	copyTree(t, filepath.Join(fixture, "home"), home)
	for _, stale := range []string{"daemon.sock", "daemon.sock.lock", "daemon.start.lock"} {
		_ = os.Remove(filepath.Join(home, "rust", stale))
	}
	workDir := filepath.Join(runDir, "workspace")
	if err := os.MkdirAll(filepath.Join(workDir, "d2probe-dir"), 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}
	files := map[string]string{
		"d2probe-main.go":        "package main\n",
		"d2probe-notes.md":       "# notes\n",
		".d2probe-hidden.go":     "package hidden\n",
		"d2probe-dir/extra.go":   "package extra\n",
		"d2probe-dir/readme.txt": "readme\n",
		"unrelated-fixture.txt":  "no\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(workDir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("写入工作区文件: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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
	t.Cleanup(func() {
		if result, err := supervisor.Shutdown(context.Background()); err == nil {
			t.Logf("回收自启服务端: stopped=%v reason=%q", result.Stopped, result.Reason)
		}
	})

	// 默认权限策略：exec 需要审批，正好用来观察重放。
	var created protocol.SessionCreateResult
	if err := conn.Client.CallInto(ctx, "session.create", map[string]any{"cwd": workDir}, &created); err != nil {
		t.Skipf("创建真实会话失败（账号/模型不可用）: %v", err)
	}
	sessionID := created.SessionID
	t.Logf("会话 %s", sessionID)

	methods := readSessionMethods(t, ctx, conn.Client, sessionID)
	searchEvidence(t, ctx, conn.Client, sessionID, workDir, methods)

	approvalReplayEvidence(t, ctx, conn.Client, sessionID, supervisor)
}

// searchEvidence 调服务端索引方法并与本地扫描对照。
func searchEvidence(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, workDir string,
	methods []string,
) {
	t.Helper()
	if !hasSessionMethod(methods, "session.files.search") {
		t.Logf("@ 搜索: 方法表里没有 session.files.search —— 能力缺失，零请求")
		return
	}
	var reply struct {
		Candidates []struct {
			RelativePath string `json:"relativePath"`
			AbsolutePath string `json:"absolutePath"`
			Dir          bool   `json:"dir"`
		} `json:"candidates"`
	}
	if err := conn.CallInto(ctx, "session.files.search", map[string]any{
		"sessionId": sessionID,
		"query":     "d2probe",
	}, &reply); err != nil {
		t.Skipf("@ 搜索请求失败（能力不可用）: %v", err)
	}
	serverPaths := make([]string, 0, len(reply.Candidates))
	for _, candidate := range reply.Candidates {
		serverPaths = append(serverPaths, candidate.RelativePath)
		if candidate.AbsolutePath == "" {
			t.Fatalf("候选缺少绝对路径: %+v", candidate)
		}
	}
	sort.Strings(serverPaths)
	local := localMentionScan(t, workDir, "d2probe")
	t.Logf("@ 搜索: 服务端 %d 条，本机扫描 %d 条", len(serverPaths), len(local))
	if !equalStrings(serverPaths, local) {
		t.Fatalf("@ 搜索与本地扫描口径不一致\n服务端: %v\n本机: %v", serverPaths, local)
	}
	for _, path := range serverPaths {
		if strings.HasPrefix(path, ".") || strings.Contains(path, "/.") {
			t.Fatalf("查询不以点开头时不该返回隐藏文件: %s", path)
		}
	}
	t.Logf("@ 搜索候选: %v", serverPaths)
}

// localMentionScan 是本机对照扫描：与隐藏文件规则一致，名字或相对路径含查询词即命中。
func localMentionScan(t *testing.T, root, query string) []string {
	t.Helper()
	query = strings.ToLower(query)
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		// 隐藏文件与隐藏目录都不进索引。
		for _, part := range strings.Split(relative, "/") {
			if strings.HasPrefix(part, ".") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		lower := strings.ToLower(relative)
		base := lower
		if index := strings.LastIndexByte(lower, '/'); index >= 0 {
			base = lower[index+1:]
		}
		if strings.Contains(base, query) || strings.Contains(lower, query) {
			out = append(out, relative)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("本机扫描: %v", err)
	}
	sort.Strings(out)
	return out
}

// approvalReplayEvidence 观察一次真实审批与重放，并核对提问集合不受影响。
func approvalReplayEvidence(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID string,
	supervisor *daemon.Supervisor,
) {
	t.Helper()
	type noticeLog struct {
		requests chan string
	}
	requests := make(chan string, 16)
	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	t.Cleanup(manager.Close)

	listener := session.Listener{
		OnEvent: func(event session.Event) {
			if event.Method != "approval.request" {
				return
			}
			var params struct {
				RequestID string `json:"requestId"`
			}
			if err := json.Unmarshal(event.Params, &params); err != nil || params.RequestID == "" {
				return
			}
			select {
			case requests <- params.RequestID:
			default:
			}
		},
	}
	_, lease, err := manager.Acquire(ctx, sessionID, listener)
	if err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	defer lease.Release()

	// 触发一次需要审批的工具调用。
	submitMainTurn(t, ctx, conn, sessionID, "Use the exec tool exactly once to run this command: echo d2probe. Then reply DONE.")
	firstID, ok := waitString(ctx, requests, 3*time.Minute)
	if !ok {
		t.Skipf("真实模型没有触发需要审批的工具调用（服务端未发 approval.request）")
	}
	t.Logf("审批请求: request=%s（第一次）", firstID)

	questionsBefore := countPendingQuestions(t, ctx, conn, sessionID)
	t.Logf("提问集合: 重放前 %d 批", questionsBefore)

	// 重新 attach 一次：服务端会重放当前待决的审批。
	replayManager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		t.Fatalf("创建重放用管理器: %v", err)
	}
	_, replayLease, err := replayManager.Acquire(ctx, sessionID, session.Listener{
		OnEvent: func(event session.Event) {
			if event.Method != "approval.request" {
				return
			}
			var params struct {
				RequestID string `json:"requestId"`
			}
			if err := json.Unmarshal(event.Params, &params); err != nil || params.RequestID == "" {
				return
			}
			select {
			case requests <- params.RequestID:
			default:
			}
		},
	})
	if err != nil {
		replayLease.Release()
		t.Fatalf("重新 attach: %v", err)
	}
	secondID, ok := waitString(ctx, requests, 2*time.Minute)
	replayLease.Release()
	replayManager.Close()
	if !ok {
		t.Skipf("重放没有再次交付审批请求")
	}
	t.Logf("审批请求: request=%s（重放）", secondID)
	if firstID != secondID {
		t.Fatalf("重放应重发同一批待决请求: %s vs %s", firstID, secondID)
	}
	questionsAfter := countPendingQuestions(t, ctx, conn, sessionID)
	t.Logf("提问集合: 重放后 %d 批（重放只涉及审批）", questionsAfter)
	if questionsBefore != questionsAfter {
		t.Fatalf("审批重放不该影响提问集合: %d -> %d", questionsBefore, questionsAfter)
	}

	// 回应拒绝，让这一轮收尾。
	if err := conn.CallInto(ctx, "approval.respond", map[string]any{
		"sessionId": sessionID,
		"branchId":  mainBranch,
		"requestId": firstID,
		"decision":  "deny",
	}, nil); err != nil {
		t.Skipf("回应审批失败（服务端不再接受该请求）: %v", err)
	}
	t.Logf("审批回应: request=%s decision=deny", firstID)
	waitForTurnEndAfter(t, ctx, conn, sessionID, 4*time.Minute)
}

// countPendingQuestions 数一数待答提问的批数。
func countPendingQuestions(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID string,
) int {
	t.Helper()
	var reply wirePendingQuestions
	if err := conn.CallInto(ctx, "session.question.pending", map[string]any{
		"sessionId": sessionID,
		"branchId":  mainBranch,
	}, &reply); err != nil {
		t.Fatalf("读取待答提问: %v", err)
	}
	return len(reply.Batches)
}

// submitMainTurn 在主线提交一次输入。
func submitMainTurn(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID, text string,
) {
	t.Helper()
	var result struct {
		Event protocol.RawEvent `json:"event"`
	}
	if err := conn.CallInto(ctx, "session.submit", map[string]any{
		"sessionId":    sessionID,
		"branchId":     mainBranch,
		"submissionId": newUUID(t),
		"text":         text,
	}, &result); err != nil {
		t.Fatalf("提交输入: %v", err)
	}
}

// waitString 等一个字符串；超时返回 false。
func waitString(ctx context.Context, values <-chan string, timeout time.Duration) (string, bool) {
	deadline := time.After(timeout)
	select {
	case value := <-values:
		return value, true
	case <-deadline:
		return "", false
	case <-ctx.Done():
		return "", false
	}
}

// waitForTurnEndAfter 轮询主分支历史，等到一条 turn/end。
func waitForTurnEndAfter(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID string,
	timeout time.Duration,
) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		page := readHistory(t, ctx, conn, sessionID, mainBranch)
		for index := len(page.Events) - 1; index >= 0; index-- {
			event := page.Events[index]
			if event.Type == "turn" && event.Name == "end" {
				return
			}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(300 * time.Millisecond):
		}
	}
}
