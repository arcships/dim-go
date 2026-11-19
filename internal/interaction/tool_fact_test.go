package interaction

import (
	"encoding/json"
	"testing"
)

// 本文件断言审批事实里的工具名、入参原文与来源类别：它们是界面挑输入渲染器的
// 唯一来源，所以要在归一化与只读视图两处都不丢、不串。

func TestApprovalFactKeepsToolAndInput(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	input := json.RawMessage(`{"command":"ls -la","cwd":"/work"}`)
	facts := arbiter.Apply(Fact{
		Kind:       PendingApproval,
		RequestID:  "a-tool",
		SessionID:  testSession,
		Tool:       "exec",
		ToolInput:  input,
		SourceKind: "tool",
	})
	if len(facts) != 0 {
		t.Fatalf("合法事实被丢弃：%v", facts)
	}
	item, ok := arbiter.Current()
	if !ok {
		t.Fatal("审批没有进队列")
	}
	if item.Tool != "exec" || item.SourceKind != "tool" {
		t.Fatalf("工具名/来源 = %q/%q，想要 exec/tool", item.Tool, item.SourceKind)
	}
	if string(item.ToolInput) != string(input) {
		t.Fatalf("入参原文 = %s，想要 %s", item.ToolInput, input)
	}
	// 防御性拷贝：调用方改自己的字节，只读视图不变。
	input[2] = 'X'
	if string(item.ToolInput) == string(input) {
		t.Fatal("只读视图与调用方共用入参字节")
	}
}

func TestApprovalFactRejectsBrokenToolInput(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	rejections := arbiter.Apply(Fact{
		Kind:      PendingApproval,
		RequestID: "a-broken",
		SessionID: testSession,
		Tool:      "exec",
		ToolInput: json.RawMessage(`{"command":`),
	})
	if len(rejections) != 1 || rejections[0].Reason != RejectionInvalidFact {
		t.Fatalf("坏入参没有被丢弃：%v", rejections)
	}
	if _, ok := arbiter.Current(); ok {
		t.Fatal("坏事实进了待决集合")
	}
}
