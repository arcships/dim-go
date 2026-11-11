package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestContextUsageReadMapsReading 核对 session.context.usage 的读取与缓存。
func TestContextUsageReadMapsReading(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.context.usage"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.context.usage", Result: map[string]any{
				"through": 12, "contextWindow": 258000, "outputTokens": 128000,
				"thresholdTokens": 200000, "estimatedInputTokens": 30000,
				"baselineSource": "messages",
			}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	reading, err := result.base.ReadContextUsage(context.Background(), testSession)
	if err != nil {
		t.Fatalf("读取上下文占用: %v", err)
	}
	if reading.Through != 12 || reading.ContextWindow != 258000 || reading.OutputTokens != 128000 ||
		reading.ThresholdTokens != 200000 || reading.EstimatedInputTokens != 30000 ||
		reading.BaselineSource != "messages" {
		t.Fatalf("占用读数映射不对: %+v", reading)
	}
	if reading.Remaining() != 228000 || reading.OverThreshold() {
		t.Fatalf("占用换算不对: remaining=%d over=%v", reading.Remaining(), reading.OverThreshold())
	}
	params := requestParams(t, server.RequestsFor("session.context.usage")[0])
	if params["sessionId"] != testSession || params["branchId"] != mainBranchID {
		t.Fatalf("占用读取参数不对: %v", params)
	}
	cached, ok := result.base.ContextReading(testSession)
	if !ok || cached.EstimatedInputTokens != 30000 {
		t.Fatalf("读数应缓存到会话: %+v ok=%t", cached, ok)
	}

	// 缺能力：hello 与会话方法表都没有时零请求。
	server2 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	if _, err := result2.base.ReadContextUsage(context.Background(), testSession); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺能力应报能力缺失: %v", err)
	}
	if got := len(server2.RequestsFor("session.context.usage")); got != 0 {
		t.Fatalf("缺能力时不应请求（%d 次）", got)
	}
}

// TestCompactContextMapsWrite 核对手动压缩走写登记与会话流，且只发一次。
func TestCompactContextMapsWrite(t *testing.T) {
	server := startFakeServer(t, fakeappserver.Script{
		Features: []string{"session.context.compact"},
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
		Replies: []fakeappserver.Reply{
			{Method: "session.context.compact", Result: map[string]any{"sequence": 9, "status": "completed"}},
		},
	})
	result := startAssembly(t, server, assemblyOptions{session: testSession})
	ctx := context.Background()
	registered, err := result.base.CompactContext(ctx, testSession)
	if err != nil {
		t.Fatalf("登记压缩: %v", err)
	}
	settled, err := result.base.Wait(ctx, registered.ID)
	if err != nil || settled.Status != write.StatusSucceeded {
		t.Fatalf("压缩应成功: %v %+v", err, settled)
	}
	if got := len(server.RequestsFor("session.context.compact")); got != 1 {
		t.Fatalf("压缩应只发一次，实际 %d", got)
	}
	params := requestParams(t, server.RequestsFor("session.context.compact")[0])
	if params["sessionId"] != testSession || params["branchId"] != mainBranchID {
		t.Fatalf("压缩参数不对: %v", params)
	}
	if _, ok := params["op"]; ok {
		t.Fatalf("op 是意图级字段，不应下发: %v", params)
	}
	var reply struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(settled.Result, &reply); err != nil || reply.Status != "completed" {
		t.Fatalf("压缩回执应透传: %v %s", err, settled.Result)
	}

	// 缺能力：零请求，入口直接报能力缺失。
	server2 := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	result2 := startAssembly(t, server2, assemblyOptions{session: testSession})
	if _, err := result2.base.CompactContext(ctx, testSession); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺能力应报能力缺失: %v", err)
	}
	if got := len(server2.RequestsFor("session.context.compact")); got != 0 {
		t.Fatalf("缺能力时不应请求（%d 次）", got)
	}
}
