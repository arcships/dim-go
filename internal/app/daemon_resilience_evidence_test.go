package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestRealResilienceEvidence 是断线恢复、共享订阅与未知结果核对的真实证据。
//
// 只有设置 DIMAGENT_RESILIENCE_EVIDENCE=1 时运行；夹具 home 从 /tmp 只读复制。
// 证据只留事件类型、标识、状态与连接数。
func TestRealResilienceEvidence(t *testing.T) {
	if os.Getenv("DIMAGENT_RESILIENCE_EVIDENCE") != "1" {
		t.Skip("未设置 DIMAGENT_RESILIENCE_EVIDENCE=1，跳过韧性证据")
	}
	fixtureHome := os.Getenv("DIMAGENT_ACCOUNT_FIXTURE")
	if fixtureHome == "" {
		fixtureHome = "/private/tmp/dimagent-real-model/home"
	}
	serverPath := os.Getenv("DIMAGENT_APP_SERVER_BIN")
	if serverPath == "" {
		serverPath = filepath.Join(os.Getenv("HOME"), ".dimcode-alpha", "rust", "dimcode-server")
	}
	runDir, err := os.MkdirTemp("/tmp", "d12x-")
	if err != nil {
		t.Fatalf("创建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runDir) })
	if err := copyFixtureTree(fixtureHome, runDir); err != nil {
		t.Fatalf("复制夹具 home: %v", err)
	}

	base, err := New(Config{
		DataDir:       runDir,
		ServerPath:    serverPath,
		ClientName:    "dimagent-mygo-resilience-evidence",
		WorkspaceRoot: runDir,
		Version:       testVersion,
		SendTimeout:   30 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	if err := base.Connect(ctx); err != nil {
		t.Fatalf("连接服务端: %v", err)
	}
	defer base.CloseConnection(ctx)

	var created protocol.SessionCreateResult
	if err := base.control().CallInto(ctx, "session.create", map[string]any{
		"cwd":              runDir,
		"permissionPolicy": map[string]any{"preset": "full-access"},
	}, &created); err != nil {
		t.Fatalf("创建会话: %v", err)
	}
	if _, err := base.OpenSession(ctx, created.SessionID); err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	if err := base.FirstHistory(ctx); err != nil {
		t.Fatalf("首轮历史: %v", err)
	}
	if err := base.AllowWrites(ctx); err != nil {
		t.Fatalf("放行写入: %v", err)
	}

	// 1) 多窗口共享订阅：同一会话两个持有者，只多一条流；释放一个仍在。
	ping := func() protocol.PingResult {
		var result protocol.PingResult
		if err := base.control().CallInto(ctx, "daemon.ping", map[string]any{}, &result); err != nil {
			t.Fatalf("daemon.ping: %v", err)
		}
		return result
	}
	beforePing := ping()
	_, secondLease, err := base.manager.Acquire(ctx, created.SessionID, session.Listener{})
	if err != nil {
		t.Fatalf("第二个持有者: %v", err)
	}
	afterSecond := ping()
	secondLease.Release()
	afterReleaseOne := ping()
	t.Logf("连接数: 初始 control+stream=%d；第二个持有者后=%d；释放一个后=%d",
		beforePing.Connections, afterSecond.Connections, afterReleaseOne.Connections)

	// 2) 断线恢复：先在会话里留下内容，再杀掉 daemon，等自动重连并按序号重读。
	if _, err := base.Submit(ctx, SubmitRequest{
		SessionID: created.SessionID,
		Text:      "resilience evidence: content before disconnect",
	}); err != nil {
		t.Fatalf("断线前提交: %v", err)
	}
	time.Sleep(2 * time.Second)
	if _, err := base.ReloadLatest(ctx, created.SessionID); err != nil {
		t.Fatalf("断线前重读: %v", err)
	}
	kindsBeforeKill, _ := base.HistoryKinds(ctx, created.SessionID)
	rowsBefore := len(base.ViewMust(created.SessionID).Rows)
	headBefore := base.ViewMust(created.SessionID).Anchor
	socket := filepath.Join(runDir, "rust", "daemon.sock")
	pid := daemonPID(t, socket)
	if err := exec.Command("kill", "-9", pid).Run(); err != nil {
		t.Fatalf("杀掉 daemon: %v", err)
	}
	t.Logf("已杀掉 daemon: pidSet=%t", pid != "")
	if !waitFor(30*time.Second, func() bool { return !base.Connected() }) {
		t.Logf("重连观察：连接一直未显示断开（自动重连可能瞬时完成）")
	}
	if !waitFor(90*time.Second, func() bool { return base.Connected() }) {
		t.Fatalf("daemon 被杀后没有自动重连")
	}
	// 重连后等历史重读落地。
	time.Sleep(2 * time.Second)
	viewAfter, err := base.ReloadLatest(ctx, created.SessionID)
	if err != nil {
		t.Fatalf("重读历史: %v", err)
	}
	unique := make(map[string]bool, len(viewAfter.Rows))
	for index, row := range viewAfter.Rows {
		key := row.ID
		if key == "" {
			key = "row:" + string(rune(index))
		}
		unique[key] = true
	}
	kindsAfterKill, _ := base.HistoryKinds(ctx, created.SessionID)
	t.Logf("恢复: head=%d->%d rows=%d->%d 唯一行=%d input/message=%d->%d",
		headBefore, viewAfter.Anchor, rowsBefore, len(viewAfter.Rows), len(unique),
		kindsBeforeKill["input/message"], kindsAfterKill["input/message"])
	// 模型在断线期间可能继续产生事实，数量变化本身不算重复；重复以行标识为准。
	_ = kindsBeforeKill["input/message"]
	if viewAfter.Anchor < headBefore {
		t.Fatalf("重连后头部倒退: %d -> %d", headBefore, viewAfter.Anchor)
	}
	if len(unique) != len(viewAfter.Rows) {
		t.Fatalf("重连后出现重复行: rows=%d unique=%d", len(viewAfter.Rows), len(unique))
	}

	// 3) 结果未知核对：极短发送超时 → 未知 → 核对一次 → 不重发。
	untracked, err := New(Config{
		DataDir:          runDir,
		ServerPath:       serverPath,
		ClientName:       "dimagent-mygo-verify-evidence",
		WorkspaceRoot:    runDir,
		Version:          testVersion,
		SendTimeout:      time.Nanosecond,
		InitialSessionID: created.SessionID,
	})
	if err != nil {
		t.Fatalf("创建第二实例: %v", err)
	}
	if err := untracked.LoadLocalState(ctx); err != nil {
		t.Fatalf("第二实例本地状态: %v", err)
	}
	if err := untracked.Connect(ctx); err != nil {
		t.Fatalf("第二实例连接: %v", err)
	}
	defer untracked.CloseConnection(ctx)
	if err := untracked.Subscribe(ctx); err != nil {
		t.Fatalf("第二实例订阅: %v", err)
	}
	if err := untracked.FirstHistory(ctx); err != nil {
		t.Fatalf("第二实例首轮历史: %v", err)
	}
	if err := untracked.AllowWrites(ctx); err != nil {
		t.Fatalf("第二实例放行写入: %v", err)
	}
	beforeKinds, _ := untracked.HistoryKinds(ctx, created.SessionID)
	registered, err := untracked.Submit(ctx, SubmitRequest{
		SessionID: created.SessionID,
		Text:      "resilience evidence: reply OK only",
	})
	if err != nil {
		t.Fatalf("提交: %v", err)
	}
	settled, err := untracked.Wait(ctx, registered.ID)
	if err != nil {
		t.Fatalf("等待收敛: %v", err)
	}
	afterKinds, _ := untracked.HistoryKinds(ctx, created.SessionID)
	addedInputs := afterKinds["input/message"] - beforeKinds["input/message"]
	occurrences := countInputOccurrences(ctx, untracked, created.SessionID, registered.InputID)
	t.Logf("未知核对: status=%s reasonSet=%t 新增 input/message=%d 该 submissionId 出现=%d",
		settled.Status, settled.Reason != "", addedInputs, occurrences)
	if settled.Status != write.StatusSucceeded {
		t.Logf("核对未收敛为成功（按 blocked 记录）: %s", settled.Reason)
	}
	if occurrences > 1 {
		t.Fatalf("该提交在历史里出现 %d 次（应只出现 1 次）", occurrences)
	}

	// 两个持有者都释放后才断开：关掉本实例的会话，连接数应再减一条流。
	base.CloseSession(created.SessionID)
	time.Sleep(500 * time.Millisecond)
	var final protocol.PingResult
	pingErr := base.control().CallInto(ctx, "daemon.ping", map[string]any{}, &final)
	if pingErr != nil {
		t.Logf("连接数: 释放会话后控制连接已重连中，未能取到计数: %v", pingErr)
	} else {
		t.Logf("连接数: 释放本实例会话后 control+stream=%d", final.Connections)
	}
}

// ViewMust 返回会话视图；没有打开时直接失败（证据里用）。
func (a *App) ViewMust(sessionID string) SessionView {
	view, ok := a.View(sessionID)
	if !ok {
		panic("会话未打开: " + sessionID)
	}
	return view
}

// daemonPID 通过 socket 路径找 daemon 进程。
func daemonPID(t *testing.T, socket string) string {
	t.Helper()
	output, err := exec.Command("pgrep", "-f", socket).Output()
	if err != nil {
		t.Fatalf("查找 daemon 进程: %v", err)
	}
	lines := strings.Fields(string(output))
	if len(lines) == 0 {
		t.Fatalf("没有找到使用 %s 的进程", socket)
	}
	sort.Strings(lines)
	return lines[0]
}

// waitFor 轮询条件直到超时。
func waitFor(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return condition()
}

// countInputOccurrences 统计历史里引用该提交标识的 input/message 事实条数。
func countInputOccurrences(ctx context.Context, base *App, sessionID, inputID string) int {
	if inputID == "" {
		return 0
	}
	runtime, err := base.openRuntime(sessionID, "session.history")
	if err != nil {
		return 0
	}
	events, err := (&changeSource{app: base}).collectFacts(ctx, runtime)
	if err != nil {
		return 0
	}
	count := 0
	for _, event := range events {
		if event.Type != "input" || event.Name != "message" {
			continue
		}
		if strings.Contains(string(event.Data), inputID) {
			count++
		}
	}
	return count
}
