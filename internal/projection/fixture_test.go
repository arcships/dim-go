package projection

import (
	"encoding/json"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// 固定时间基：事实的时间只来自 create_at，测试里不带当前时间。
const fixtureBaseAt int64 = 1700000000000

// ev 造一条原始事实。data 是 wire 上的 data 字段原文。
func ev(id string, sequence int64, typ, name, data string) protocol.RawEvent {
	return protocol.RawEvent{
		ID:       id,
		Sequence: sequence,
		CreateAt: fixtureBaseAt + sequence,
		Type:     typ,
		Name:     name,
		Data:     json.RawMessage(data),
	}
}

// conversationFixture 是一段带轮次、思考、文本、工具、目标、系统提示与四类未完成事项的对话。
// 同一份 fixture 既走历史页也走逐条实时通知，用来验证两条路径收敛到同一视图。
func conversationFixture() []protocol.RawEvent {
	return []protocol.RawEvent{
		ev("e1", 1, "input", "message", `{"mode":"queue","user":{"content":[{"type":"text","text":"帮我看下构建"}]}}`),
		ev("e2", 2, "input", "consumed", `{"input_id":"e1"}`),
		ev("e3", 3, "turn", "start", `{"turn_id":"t1"}`),
		ev("e4", 4, "step", "model-start", `{"turn_id":"t1","request_id":"r1"}`),
		ev("e5", 5, "step", "thinking", `{"turn_id":"t1","request_id":"r1","delta":"先想一想"}`),
		ev("e6", 6, "step", "text", `{"turn_id":"t1","request_id":"r1","delta":"我先"}`),
		ev("e7", 7, "step", "text", `{"turn_id":"t1","request_id":"r1","delta":"看文件"}`),
		ev("e8", 8, "step", "tool-use", `{"turn_id":"t1","request_id":"r1","call":{"id":"c1","name":"read_file","input":{"path":"a.go"}}}`),
		ev("e9", 9, "step", "tool-completed", `{"turn_id":"t1","request_id":"r1","call_id":"c1","result":{"content":[{"type":"text","text":"package main"}]}}`),
		ev("e10", 10, "step", "text", `{"turn_id":"t1","request_id":"r1","delta":"看完了"}`),
		ev("e11", 11, "step", "model-output-end", `{"turn_id":"t1","request_id":"r1","status":"completed"}`),
		ev("e12", 12, "turn", "end", `{"turn_id":"t1","status":"completed"}`),
		ev("e13", 13, "input", "message", `{"blocks":[{"key":"goal:state","text":"<dim-goal status=\"active\" judge=\"model\" op=\"set\">\n<objective>\n修好构建\n</objective>\n</dim-goal>"}]}`),
		ev("e14", 14, "exec", "started", `{"task_id":"k1","description":"go test ./..."}`),
		ev("e15", 15, "collaboration", "request", `{"turn_id":"t1","request_id":"r1","call_id":"c9","target":{"kind":"agent","agent_id":"reviewer"},"prompt":"看一下"}`),
		ev("e16", 16, "step", "tool-call", `{"turn_id":"t1","request_id":"r1","calls":[{"id":"q1","name":"ask_user","input":{"questions":[{"question":"继续吗"}]}}]}`),
		ev("e17", 17, "context", "compaction", `{"summary":"压缩了 3 条"}`),
		ev("e18", 18, "input", "message", `{"mode":"queue","user":{"content":[{"type":"text","text":"再跑一次"}]}}`),
	}
}

// noticeJSON 把一条事实包成 `session.event` 通知参数。
func noticeJSON(t *testing.T, branchID string, event protocol.RawEvent) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(protocol.SessionEventParams{
		SessionID: "s-1",
		BranchID:  branchID,
		Event:     event,
	})
	if err != nil {
		t.Fatalf("通知编码失败: %v", err)
	}
	return raw
}

// noticeRaw 把一条通知的参数编码成 JSON。
func noticeRaw(t *testing.T, fields map[string]any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("通知编码失败: %v", err)
	}
	return raw
}

// jsonRaw 把 JSON 原文嵌进参数。
func jsonRaw(data string) json.RawMessage { return json.RawMessage(data) }

// page 把一串事实包成一页历史。
func page(through int64, events ...protocol.RawEvent) protocol.SessionHistoryResult {
	return protocol.SessionHistoryResult{
		Events:    events,
		Through:   through,
		NextAfter: through,
		Done:      true,
	}
}

// newFolder 建一个空的折叠器。
func newFolder(t *testing.T) *Folder {
	t.Helper()
	folder, err := New(Config{SessionID: "s-1"})
	if err != nil {
		t.Fatalf("新建折叠器失败: %v", err)
	}
	return folder
}

// mustHistory 折叠一页历史。
func mustHistory(t *testing.T, folder *Folder, branchID string, result protocol.SessionHistoryResult) {
	t.Helper()
	if rejections := folder.ApplyHistory(branchID, result); len(rejections) > 0 {
		t.Fatalf("历史折叠出现丢弃: %+v", rejections[0])
	}
}

// mustNotice 折叠一条实时通知。
func mustNotice(t *testing.T, folder *Folder, branchID string, event protocol.RawEvent) {
	t.Helper()
	if rejections := folder.ApplyNotice("session.event", noticeJSON(t, branchID, event)); len(rejections) > 0 {
		t.Fatalf("实时折叠出现丢弃: %+v", rejections[0])
	}
}

// progressJSON 造一条 `session.model.progress` 通知。
func progressJSON(t *testing.T, branchID, turnID, requestID, text string, textOffset int, thinking string, thinkingOffset int) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(progressNotice{
		SessionID:      "s-1",
		BranchID:       branchID,
		TurnID:         turnID,
		RequestID:      requestID,
		Text:           text,
		TextOffset:     textOffset,
		Thinking:       thinking,
		ThinkingOffset: thinkingOffset,
	})
	if err != nil {
		t.Fatalf("进度通知编码失败: %v", err)
	}
	return raw
}

// rowFingerprint 是行里与身份无关的部分：身份会因为落库而变，内容不会。
type rowFingerprint struct {
	Kind        RowKind
	TurnIndex   int
	TurnID      string
	Status      RowStatus
	Text        string
	HasTool     bool
	ToolName    string
	ToolResult  string
	ToolIsError bool
	GoalAction  string
	GoalStatus  GoalStatus
}

func fingerprint(view View) []rowFingerprint {
	out := make([]rowFingerprint, 0, len(view.Rows))
	for _, row := range view.Rows {
		item := rowFingerprint{
			Kind:      row.Kind,
			TurnIndex: row.TurnIndex,
			TurnID:    row.TurnID,
			Status:    row.Status,
			Text:      row.Text,
		}
		if row.Tool != nil {
			item.HasTool = true
			item.ToolName = row.Tool.Name
			item.ToolResult = row.Tool.Result
			item.ToolIsError = row.Tool.IsError
		}
		if row.Goal != nil {
			item.GoalAction = row.Goal.Action
			item.GoalStatus = row.Goal.Status
		}
		out = append(out, item)
	}
	return out
}

// rowIDs 取视图里的正式行标识序列。
func rowIDs(view View) []string {
	out := make([]string, 0, len(view.Rows))
	for _, row := range view.Rows {
		out = append(out, row.ID)
	}
	return out
}
