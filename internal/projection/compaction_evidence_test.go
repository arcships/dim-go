package projection

// 调查证据：上下文占用与压缩的真实触发条件。
//
// 结论先行：
//   - `context/budget` 是服务端内部事实：它按每次模型调用写入时间线，但协议 v3 的
//     事件可见性白名单把它排除（`dim-app-server` 的 ProductEvent::new），因此任何
//     客户端都收不到它。可用的读法是会话方法 `session.context.usage`。
//   - 压缩可以手动触发（会话方法 `session.context.compact`），但 summary 策略要求
//     "切掉之后确实变小"：上下文还没超过它自己的保留目标时，压缩以 failed 收口
//     （`entriesReplaced: 0`，错误 `Compacted context does not reduce input.`）。
//     保留目标 = min(0.5 × 输出预留, 0.1 × (窗口 − 输出预留))，所以要用小窗口模型
//     加足够多的内容才触发成功。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
)

// TestRealDaemonContextAndCompaction 记录占用读数与压缩触发条件。
//
// 用法：
//
//	DIM_REAL_MODEL_FIXTURE=/tmp/dimagent-real-model \
//	  go test -run TestRealDaemonContextAndCompaction -v -count=1 ./internal/projection/
func TestRealDaemonContextAndCompaction(t *testing.T) {
	fixture := os.Getenv("DIM_REAL_MODEL_FIXTURE")
	if fixture == "" {
		t.Skip("未设置 DIM_REAL_MODEL_FIXTURE：真实模型证据 blocked（缺 /tmp 夹具）")
	}
	serverPath := filepath.Join(fixture, "bin", "dimcode-server")
	if _, err := os.Stat(serverPath); err != nil {
		t.Skipf("夹具缺少 daemon 可执行文件: %v", err)
	}

	runDir, err := os.MkdirTemp("/tmp", "d2compact-")
	if err != nil {
		t.Fatalf("创建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runDir) })
	home := filepath.Join(runDir, "home")
	copyTree(t, filepath.Join(fixture, "home"), home)
	for _, stale := range []string{"daemon.sock", "daemon.sock.lock", "daemon.start.lock"} {
		_ = os.Remove(filepath.Join(home, "rust", stale))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
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

	workDir := filepath.Join(runDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}
	// 用保留目标最小的模型（258k 窗口 / 128k 输出 → 保留目标 13k token），
	// 这样普通的短会话也能把上下文推过保留线。
	sessionID := createSession(t, ctx, conn.Client, workDir, "gpt-5.6-terra")
	t.Logf("会话 %s（模型 gpt-5.6-terra）", sessionID)

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
	if rejections := folder.ApplySnapshot(snapshot); len(rejections) > 0 {
		t.Logf("首屏折叠丢弃: %+v", rejections[0])
	}

	client := &pendingClient{t: t, ctx: ctx, conn: conn.Client, sessionID: sessionID, folder: folder}

	// 1. 占用读数：context/budget 从不出现，session.context.usage 给出同样的事实。
	reading := readContextUsage(t, ctx, conn.Client, sessionID)
	t.Logf("占用读数(session.context.usage): 窗口=%d 输出预留=%d 阈值=%d 估算输入=%d 基线=%s 已过阈值=%v",
		reading.ContextWindow, reading.OutputTokens, reading.ThresholdTokens,
		reading.EstimatedInputTokens, reading.BaselineSource, reading.OverThreshold())
	if reading.ContextWindow <= 0 || reading.EstimatedInputTokens <= 0 {
		t.Fatalf("占用读数不完整: %+v", reading)
	}
	if view := folder.View(); view.Context != nil {
		t.Fatalf("协议 v3 不该收到 context/budget 事实，但视图里有: %+v", view.Context)
	} else {
		t.Logf("占用事实(context/budget): 客户端未收到（服务端 v3 可见性白名单排除）")
	}

	// 2. 把上下文推过保留目标：连续几轮大输出。
	turns := 0
	for turns < 4 && reading.EstimatedInputTokens < 30000 {
		client.submit("Use the exec tool exactly once to run this command: seq 1 20000. Then reply DONE.", "")
		waitTurnEnd(t, ctx, recorder, recorder.head())
		turns++
		reading = readContextUsage(t, ctx, conn.Client, sessionID)
		t.Logf("第 %d 轮后: 估算输入=%d 阈值=%d 窗口=%d", turns, reading.EstimatedInputTokens, reading.ThresholdTokens, reading.ContextWindow)
	}

	// 3. 手动压缩：session.context.compact。
	start := time.Now()
	var reply struct {
		Sequence int64  `json:"sequence"`
		Status   string `json:"status"`
	}
	compactErr := conn.Client.CallInto(ctx, "session.context.compact", map[string]any{
		"sessionId": sessionID,
		"branchId":  mainBranch,
	}, &reply)
	t.Logf("压缩回执: err=%v status=%s（耗时 %s）", compactErr, reply.Status, time.Since(start))

	// 4. 压缩事实（context/compaction 对 v3 可见）与折叠结果。
	fact, ok := recorder.wait(ctx, 2*time.Minute, func(event protocol.RawEvent) bool {
		return event.Type == "context" && event.Name == "compaction"
	})
	if !ok {
		t.Skipf("没有等到压缩事实（观察到：%v）", recorder.kinds())
	}
	data := decodeObject(fact.Data)
	diagnostics := object(data["diagnostics"])
	t.Logf("压缩事实: seq=%d status=%s 策略=%s 替换条目=%d 估算输入=%d 估算输出=%d",
		fact.Sequence, firstString(data, "status"), firstString(data, "strategy"),
		toInt64(diagnostics["entriesReplaced"]),
		roundedInt64(diagnostics["estimatedInputTokens"]),
		roundedInt64(diagnostics["estimatedOutputTokens"]))
	if reason := firstString(data, "error"); reason != "" {
		t.Logf("压缩未生效的原因: %s", reason)
	}
	replacement := array(data["replacement"])
	t.Logf("压缩事实形状: keys=%v replacement=%d parts=%v", mapKeys(data), len(replacement), object(data["replacement_parts"]))
	if len(replacement) > 0 {
		// Message.content 是 untagged 的：字符串就是纯文本消息。
		first := object(replacement[0])
		t.Logf("replacement[0]: keys=%v contentIsText=%v", mapKeys(first), stringValue(first["content"]) != "")
	}

	view := folder.View()
	if view.Compactions != 1 {
		t.Fatalf("压缩次数应为 1，实际 %d", view.Compactions)
	}
	if view.Compaction == nil {
		t.Fatalf("折叠里应有最近一次压缩")
	}
	t.Logf("折叠结果: status=%s 策略=%s 摘要长度=%d", view.Compaction.Status, view.Compaction.Strategy, len(view.Compaction.Summary))
	if view.Compaction.Status != firstString(data, "status") {
		t.Fatalf("折叠状态与事实不一致: %+v / %s", view.Compaction, firstString(data, "status"))
	}

	// 成功压缩时：摘要非空、替换了条目、下一次占用读数变小。
	if view.Compaction.Status == "completed" {
		if view.Compaction.Summary == "" {
			t.Fatalf("成功的压缩应带摘要")
		}
		if toInt64(diagnostics["entriesReplaced"]) <= 0 {
			t.Fatalf("成功的压缩应替换条目: %+v", diagnostics)
		}
		after := readContextUsage(t, ctx, conn.Client, sessionID)
		t.Logf("压缩后占用: 估算输入=%d（压缩前 %d）", after.EstimatedInputTokens, reading.EstimatedInputTokens)
		if after.EstimatedInputTokens >= reading.EstimatedInputTokens {
			t.Fatalf("压缩后占用应变小: %d -> %d", reading.EstimatedInputTokens, after.EstimatedInputTokens)
		}
		t.Logf("结论: 手动压缩成功")
	} else {
		t.Logf("结论: 手动压缩未生效（status=%s），保留目标=%d 估算输入=%d",
			view.Compaction.Status, retainTarget(reading), reading.EstimatedInputTokens)
	}

	t.Logf("证据: session=%s turns=%d 窗口=%d 阈值=%d 估算输入=%d 压缩=%s events=%v",
		sessionID, turns, reading.ContextWindow, reading.ThresholdTokens,
		reading.EstimatedInputTokens, view.Compaction.Status, recorder.kinds())
}

// mapKeys 输出对象的键，用于证据里描述形状（不含内容）。
func mapKeys(data map[string]any) []string {
	out := make([]string, 0, len(data))
	for key := range data {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// retainTarget 计算 summary 策略的保留目标：min(0.5 × 输出预留, 0.1 × (窗口 − 输出预留))。
func retainTarget(reading ContextReading) int64 {
	output := float64(reading.OutputTokens)
	window := float64(reading.ContextWindow)
	target := 0.5 * output
	if candidate := 0.1 * (window - output); candidate < target {
		target = candidate
	}
	if target < 0 {
		return 0
	}
	return int64(target)
}

// createSession 建一个指定模型的会话。
func createSession(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	workDir, modelID string,
) string {
	t.Helper()
	var created protocol.SessionCreateResult
	params := map[string]any{
		"cwd":              workDir,
		"permissionPolicy": map[string]any{"preset": "full-access"},
	}
	if modelID != "" {
		params["model"] = map[string]any{"provider": "dimcode-api-oauth", "modelId": modelID}
	}
	if err := conn.CallInto(ctx, "session.create", params, &created); err != nil {
		t.Skipf("创建真实会话失败（账号/模型不可用）: %v", err)
	}
	return created.SessionID
}

// readContextUsage 读一次 `session.context.usage`。
func readContextUsage(
	t *testing.T,
	ctx context.Context,
	conn interface {
		CallInto(context.Context, string, any, any) error
	},
	sessionID string,
) ContextReading {
	t.Helper()
	var reply struct {
		Through              *int64  `json:"through"`
		ContextWindow        float64 `json:"contextWindow"`
		OutputTokens         float64 `json:"outputTokens"`
		ThresholdTokens      float64 `json:"thresholdTokens"`
		EstimatedInputTokens float64 `json:"estimatedInputTokens"`
		BaselineSource       string  `json:"baselineSource"`
	}
	if err := conn.CallInto(ctx, "session.context.usage", map[string]any{
		"sessionId": sessionID,
		"branchId":  mainBranch,
	}, &reply); err != nil {
		t.Skipf("读取上下文占用失败（服务端未提供 session.context.usage）: %v", err)
	}
	reading := ContextReading{
		ContextWindow:        int64(reply.ContextWindow),
		OutputTokens:         int64(reply.OutputTokens),
		ThresholdTokens:      int64(reply.ThresholdTokens),
		EstimatedInputTokens: int64(reply.EstimatedInputTokens),
		BaselineSource:       reply.BaselineSource,
	}
	if reply.Through != nil {
		reading.Through = *reply.Through
	}
	return reading
}
