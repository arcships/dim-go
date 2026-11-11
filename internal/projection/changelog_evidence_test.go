package projection

// 调查证据：改动统计能不能拿到行数（+x/−y）。
//
// 做法：让真实模型用 write 与 edit 各改一次文件，然后
//   - 看 step/tool-result 里的 structuredContent 有没有行数字段；
//   - 看 edit 的 display diff 能不能数出增删行；
//   - 在工作区里只读地跑 git，看行数只能从哪里来。
//
// 日志只留字段名与数值，不打印文件内容。

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
)

// TestRealDaemonChangeLineCounts 记录工具结果与本地 git 各自能给出什么。
//
// 用法：
//
//	DIM_REAL_MODEL_FIXTURE=<夹具目录> \
//	  go test -run TestRealDaemonChangeLineCounts -v -count=1 ./internal/projection/
func TestRealDaemonChangeLineCounts(t *testing.T) {
	fixture := os.Getenv("DIM_REAL_MODEL_FIXTURE")
	if fixture == "" {
		t.Skip("未设置 DIM_REAL_MODEL_FIXTURE：真实模型证据 blocked（缺临时夹具目录）")
	}
	serverPath := filepath.Join(fixture, "bin", "dimcode-server")
	if _, err := os.Stat(serverPath); err != nil {
		t.Skipf("夹具缺少 daemon 可执行文件: %v", err)
	}

	runDir := evidenceTempDir(t, "d2lines-")
	t.Cleanup(func() { _ = os.RemoveAll(runDir) })
	home := filepath.Join(runDir, "home")
	copyTree(t, filepath.Join(fixture, "home"), home)
	for _, stale := range []string{"daemon.sock", "daemon.sock.lock", "daemon.start.lock"} {
		_ = os.Remove(filepath.Join(home, "rust", stale))
	}
	workDir := filepath.Join(runDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}
	// 一个已提交的基线文件，用来区分“已跟踪改动”和“未跟踪新文件”。
	if err := os.WriteFile(filepath.Join(workDir, "tracked.txt"), []byte("one\ntwo\nthree\n"), 0o600); err != nil {
		t.Fatalf("写入基线文件: %v", err)
	}
	gitInit(t, workDir)

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

	sessionID := createSession(t, ctx, conn.Client, workDir, "")
	folder := newFolderFor(t, sessionID)
	recorder := newRecorder()
	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	snapshot, lease, err := manager.Acquire(ctx, sessionID, session.Listener{
		OnEvent: func(event session.Event) {
			folder.ApplyListenerEvent(event)
			if event.Method == "session.event" {
				var params protocol.SessionEventParams
				if err := json.Unmarshal(event.Params, &params); err == nil {
					recorder.add(params.Event)
				}
			}
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
	folder.ApplySnapshot(snapshot)
	client := &pendingClient{t: t, ctx: ctx, conn: conn.Client, sessionID: sessionID, folder: folder}

	client.submit("Use the write tool exactly once to create a file named probe.txt whose content is "+
		"exactly three lines: alpha, beta, gamma. Then use the edit tool exactly once to replace the "+
		"line beta with delta in that same file. Use no other tool.", "")
	waitTurnEnd(t, ctx, recorder, recorder.head())

	// 1. 工具结果里的字段：write 给字节数，edit 给替换次数与 display diff。
	page := readHistory(t, ctx, conn.Client, sessionID, mainBranch)
	calls := map[string]string{}
	for _, event := range page.Events {
		data := decodeObject(event.Data)
		if event.Type == "step" && (event.Name == "tool-call" || event.Name == "tool-use") {
			for _, value := range array(data["calls"]) {
				call := object(value)
				calls[stringValue(call["id"])] = stringValue(call["name"])
			}
			call := object(data["call"])
			if id := stringValue(call["id"]); id != "" {
				calls[id] = stringValue(call["name"])
			}
		}
	}

	toolFields := map[string][]string{}
	lineFieldFound := false
	diffCounts := map[string]int{}
	for _, event := range page.Events {
		if event.Type != "step" || event.Name != "tool-result" {
			continue
		}
		for _, value := range array(decodeObject(event.Data)["results"]) {
			result := object(value)
			callID := firstString(result, "call_id", "callId")
			payload := object(result["result"])
			structured := object(payload["structuredContent"])
			if len(structured) == 0 {
				structured = object(payload["structured_content"])
			}
			name := calls[callID]
			if name == "" {
				name = "<unknown>"
			}
			keys := mapKeys(structured)
			toolFields[name] = keys
			for _, key := range keys {
				lower := strings.ToLower(key)
				if strings.Contains(lower, "addition") || strings.Contains(lower, "deletion") ||
					strings.Contains(lower, "numstat") || strings.Contains(lower, "patch") ||
					strings.Contains(lower, "line") {
					lineFieldFound = true
				}
			}
			// edit 的 display diff：数一下 +/- 行（不含 hunk 头与文件头）。
			if diff := stringValue(structured["diff"]); diff != "" {
				added, removed := countDiffLines(diff)
				diffCounts[name] = added*1000 + removed
			}
			if name == "write" {
				t.Logf("write 结果字段: %v（bytesWritten=%d）", keys, roundedInt64(structured["bytesWritten"]))
			}
			if name == "edit" {
				t.Logf("edit 结果字段: %v（replacementCount=%d）", keys, roundedInt64(structured["replacementCount"]))
			}
			if _, known := structured["revision"]; known {
				t.Logf("%s 结果带 revision（内容版本），不带行数", name)
			}
		}
	}
	if lineFieldFound {
		t.Fatalf("工具结果里出现了行数字段，结论需要改写: %v", toolFields)
	}
	t.Logf("工具结果字段汇总: %v", toolFields)
	for name, packed := range diffCounts {
		t.Logf("%s 的 display diff 可数出增删行: +%d -%d（仅该次 edit，含 3 行上下文）", name, packed/1000, packed%1000)
	}

	// 2. 会话事实里有没有别的“文件变更”记录。
	hasChangeFact := false
	for _, event := range page.Events {
		lower := strings.ToLower(string(event.Data))
		if strings.Contains(lower, "additions") || strings.Contains(lower, "deletions") || strings.Contains(lower, "numstat") {
			hasChangeFact = true
		}
	}
	t.Logf("会话事实里的行数字段: %v", hasChangeFact)

	// 3. 本地只读 git 能给什么：已跟踪文件用 numstat，未跟踪文件要单独处理。
	t.Logf("git status: %v", gitPorcelain(t, workDir))
	t.Logf("git diff --numstat（已跟踪）: %v", gitNumstat(t, workDir))
	if _, err := os.Stat(filepath.Join(workDir, "probe.txt")); err == nil {
		t.Logf("未跟踪新文件的 numstat（--no-index）: %v", gitNumstatUntracked(t, workDir, "probe.txt"))
	}

	t.Logf("证据: session=%s 工具字段=%v 会话行数字段=%v", sessionID, toolFields, hasChangeFact)
}

// countDiffLines 数 unified diff 里新增与删除的行数（跳过 hunk 头）。
func countDiffLines(diff string) (int, int) {
	added, removed := 0, 0
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "@@"), strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			continue
		case strings.HasPrefix(line, "+"):
			added++
		case strings.HasPrefix(line, "-"):
			removed++
		}
	}
	return added, removed
}

// gitInit 把工作区变成一个已提交的仓库，用来区分已跟踪与未跟踪改动。
func gitInit(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		command := exec.Command("git", append([]string{"-C", dir}, args...)...)
		command.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=evidence", "GIT_AUTHOR_EMAIL=evidence@example.invalid",
			"GIT_COMMITTER_NAME=evidence", "GIT_COMMITTER_EMAIL=evidence@example.invalid",
		)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v（%s）", args, err, strings.TrimSpace(string(output)))
		}
	}
	run("init", "-q")
	run("add", "tracked.txt")
	run("commit", "-q", "-m", "base")
}

func gitPorcelain(t *testing.T, dir string) []string {
	return strings.Split(strings.TrimSpace(runGit(t, dir, "status", "--porcelain")), "\n")
}

func gitNumstat(t *testing.T, dir string) []string {
	out := strings.TrimSpace(runGit(t, dir, "diff", "--numstat"))
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

func gitNumstatUntracked(t *testing.T, dir, name string) []string {
	out := strings.TrimSpace(runGit(t, dir, "diff", "--no-index", "--numstat", os.DevNull, name))
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	sort.Strings(lines)
	return lines
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := command.Output()
	if err != nil {
		// git diff --no-index 在有差异时退出码为 1。
		if exit, ok := err.(*exec.ExitError); ok && len(exit.Stderr) == 0 && len(output) > 0 {
			return string(output)
		}
		t.Fatalf("git %v: %v", args, err)
	}
	return string(output)
}
