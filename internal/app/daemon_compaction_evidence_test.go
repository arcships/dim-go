package app

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// TestRealCompactionEvidence 是上下文占用读数与手动压缩的真实证据。
//
// 只有设置 DIMAGENT_COMPACTION_EVIDENCE=1 时运行；用已验证的触发配方：
// gpt-5.6-terra 小窗口模型 + 连续几轮大输出把上下文推过保留目标，再手动压缩。
// 证据只留数字、状态与摘要长度，不打印摘要正文。
func TestRealCompactionEvidence(t *testing.T) {
	if os.Getenv("DIMAGENT_COMPACTION_EVIDENCE") != "1" {
		t.Skip("未设置 DIMAGENT_COMPACTION_EVIDENCE=1，跳过压缩证据")
	}
	fixtureHome := evidenceFixtureHome(t)
	serverPath := evidenceServerBinary(t)
	runDir, err := os.MkdirTemp("", "dimagent-evidence-")
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
		ClientName:    "dimagent-mygo-compaction-evidence",
		WorkspaceRoot: runDir,
		Version:       testVersion,
		SendTimeout:   60 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	if err := base.Connect(ctx); err != nil {
		t.Fatalf("连接服务端: %v", err)
	}
	defer base.CloseConnection(ctx)

	// 用保留目标最小的模型建会话（gpt-5.6-terra：258k 窗口 / 128k 输出）。
	var created protocol.SessionCreateResult
	if err := base.control().CallInto(ctx, "session.create", map[string]any{
		"cwd":              runDir,
		"permissionPolicy": map[string]any{"preset": "full-access"},
		"model":            map[string]any{"provider": "dimcode-api-oauth", "modelId": "gpt-5.6-terra"},
	}, &created); err != nil {
		t.Fatalf("创建真实会话: %v", err)
	}
	if _, err := base.OpenSession(ctx, created.SessionID); err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	// 首轮历史阶段（无启动会话时只推进阶段）后放行写入。
	if err := base.FirstHistory(ctx); err != nil {
		t.Fatalf("首轮历史阶段: %v", err)
	}
	if err := base.AllowWrites(ctx); err != nil {
		t.Fatalf("放行写入: %v", err)
	}
	t.Logf("session: id=%s", created.SessionID)

	reading, err := base.ReadContextUsage(ctx, created.SessionID)
	if err != nil {
		t.Fatalf("读取占用: %v", err)
	}
	t.Logf("占用读数: 窗口=%d 输出预留=%d 阈值=%d 估算输入=%d 基线=%s",
		reading.ContextWindow, reading.OutputTokens, reading.ThresholdTokens,
		reading.EstimatedInputTokens, reading.BaselineSource)
	if view, ok := base.View(created.SessionID); ok && view.Compactions > 0 {
		t.Fatalf("尚未压缩却已有压缩计数")
	}

	turns := 0
	for turns < 4 && reading.EstimatedInputTokens < 30000 {
		if _, err := base.Submit(ctx, SubmitRequest{
			SessionID: created.SessionID,
			Text:      "Use the exec tool exactly once to run this command: seq 1 20000. Then reply DONE.",
		}); err != nil {
			t.Fatalf("提交第 %d 轮: %v", turns+1, err)
		}
		deadline := time.Now().Add(2 * time.Minute)
		for time.Now().Before(deadline) {
			current, readErr := base.ReadContextUsage(ctx, created.SessionID)
			if readErr == nil {
				reading = current
				if reading.EstimatedInputTokens >= 30000 {
					break
				}
			}
			time.Sleep(3 * time.Second)
		}
		turns++
		t.Logf("第 %d 轮后: 估算输入=%d 阈值=%d", turns, reading.EstimatedInputTokens, reading.ThresholdTokens)
	}

	registered, err := base.CompactContext(ctx, created.SessionID)
	if err != nil {
		t.Fatalf("登记压缩: %v", err)
	}
	settled, err := base.Wait(ctx, registered.ID)
	if err != nil {
		t.Fatalf("等待压缩: %v", err)
	}
	status, sequence, summaryErr := compactReplySummary(settled.Result)
	t.Logf("压缩回执: err=%v status=%s sequence=%d", summaryErr, status, sequence)

	// 等折叠出现压缩事实（context/compaction）。
	deadline := time.Now().Add(2 * time.Minute)
	for {
		view, ok := base.View(created.SessionID)
		if ok && view.Compaction != nil {
			t.Logf("折叠结果: counters=%d status=%s strategy=%s summaryLength=%d",
				view.Compactions, view.Compaction.Status, view.Compaction.Strategy,
				len(view.Compaction.Summary))
			after, readErr := base.ReadContextUsage(ctx, created.SessionID)
			if readErr == nil {
				t.Logf("压缩后占用: 估算输入=%d（压缩前 %d）", after.EstimatedInputTokens, reading.EstimatedInputTokens)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("没有等到压缩事实；压缩回执 status=%s", status)
		}
		time.Sleep(time.Second)
	}
}

// compactReplySummary 只取压缩回执里的状态与序号。
func compactReplySummary(raw []byte) (string, int64, error) {
	var reply struct {
		Status   string `json:"status"`
		Sequence int64  `json:"sequence"`
	}
	if len(raw) == 0 {
		return "", 0, nil
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return "", 0, err
	}
	return reply.Status, reply.Sequence, nil
}
