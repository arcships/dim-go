package uiwire

import (
	"encoding/json"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/interaction"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// 本文件是审批快照映射的验收：工具名与入参原文带到界面、命令原文只从命令工具的
// 入参里取、成员审批按请求工具名认出来并落到「不可审查」。
//
// 界面（internal/ui）只吃 ApprovalSnapshot：这几个函数是它唯一的事实来源。

func TestApprovalCommandComesFromCommandInput(t *testing.T) {
	exec := json.RawMessage(`{"command":"printf DIMAGENT_APPROVAL_OK","cwd":"/work"}`)
	if got := approvalCommand(exec, "exec"); got != "printf DIMAGENT_APPROVAL_OK" {
		t.Fatalf("命令工具的命令原文 = %q，想要 printf DIMAGENT_APPROVAL_OK", got)
	}
	if got := approvalCommand(json.RawMessage(`{"command":"ls"}`), "Read"); got != "" {
		t.Fatalf("非命令工具不该给出命令：%q", got)
	}
	if got := approvalCommand(json.RawMessage(`not json`), "exec"); got != "" {
		t.Fatalf("入参解不开时不该编命令：%q", got)
	}
	if got := approvalCommand(nil, "exec"); got != "" {
		t.Fatalf("没有入参时不该编命令：%q", got)
	}
}

func TestApprovalInputStringReadsOnlyStrings(t *testing.T) {
	input := json.RawMessage(`{"approval_id":"apr-9","count":3}`)
	if got := approvalInputString(input, "approval_id"); got != "apr-9" {
		t.Fatalf("approval_id = %q，想要 apr-9", got)
	}
	if got := approvalInputString(input, "count"); got != "" {
		t.Fatalf("非字符串字段不该读成文本：%q", got)
	}
	if got := approvalInputString(input, "missing"); got != "" {
		t.Fatalf("缺少的字段不该给值：%q", got)
	}
}

func TestMemberApprovalRecognizedByToolName(t *testing.T) {
	member := memberApproval(interaction.Item{
		Tool:      memberApprovalTool,
		ToolInput: json.RawMessage(`{"approval_id":"apr-9"}`),
	})
	if member == nil {
		t.Fatal("成员审批请求没有认出来")
	}
	if member.State != ui.ApprovalMemberUnavailable || member.ApprovalID != "apr-9" {
		t.Fatalf("成员审批快照 = %+v，想要 unavailable + apr-9", member)
	}
	if member.AgentID != "" || member.BranchID != "" || member.LeaderBranchID != "" {
		t.Fatalf("没有明细通道却写上了成员与分支：%+v", member)
	}

	if other := memberApproval(interaction.Item{Tool: "exec"}); other != nil {
		t.Fatalf("普通工具被当成成员审批：%+v", other)
	}
}
