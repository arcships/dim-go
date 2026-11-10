package catalog_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// staticHistory 是 D1 订阅形状的替身：一次返回该分支的全部事实。
type staticHistory struct {
	events []protocol.RawEvent
	calls  int
}

func (h *staticHistory) History(
	ctx context.Context,
	params protocol.SessionHistoryParams,
) (protocol.SessionHistoryResult, error) {
	h.calls++
	through := int64(0)
	for _, event := range h.events {
		if event.Sequence > through {
			through = event.Sequence
		}
	}
	return protocol.SessionHistoryResult{
		Events:    append([]protocol.RawEvent(nil), h.events...),
		Through:   through,
		NextAfter: through,
		Done:      true,
	}, nil
}

// rawEvent 造一条历史事实。
func rawEvent(id string, sequence int64, typ, name, data string) protocol.RawEvent {
	return protocol.RawEvent{
		ID:       id,
		Sequence: sequence,
		CreateAt: 1700000000000 + sequence,
		Type:     typ,
		Name:     name,
		Data:     json.RawMessage(data),
	}
}

// twoTurnHistory 造两轮完整对话：每轮一条用户输入与一段助手文本。
func twoTurnHistory() []protocol.RawEvent {
	return []protocol.RawEvent{
		rawEvent("e1", 1, "input", "message", `{"mode":"queue","user":{"content":[{"type":"text","text":"第一问"}]}}`),
		rawEvent("e2", 2, "input", "consumed", `{"input_id":"e1"}`),
		rawEvent("e3", 3, "turn", "start", `{"turn_id":"t1"}`),
		rawEvent("e4", 4, "step", "model-start", `{"turn_id":"t1","request_id":"r1"}`),
		rawEvent("e5", 5, "step", "text", `{"turn_id":"t1","request_id":"r1","delta":"第一答"}`),
		rawEvent("e6", 6, "step", "model-output-end", `{"turn_id":"t1","request_id":"r1","status":"completed"}`),
		rawEvent("e7", 7, "turn", "end", `{"turn_id":"t1","status":"completed"}`),
		rawEvent("e8", 8, "input", "message", `{"mode":"queue","user":{"content":[{"type":"text","text":"第二问"}]}}`),
		rawEvent("e9", 9, "input", "consumed", `{"input_id":"e8"}`),
		rawEvent("e10", 10, "turn", "start", `{"turn_id":"t2"}`),
		rawEvent("e11", 11, "step", "model-start", `{"turn_id":"t2","request_id":"r2"}`),
		rawEvent("e12", 12, "step", "text", `{"turn_id":"t2","request_id":"r2","delta":"第二答"}`),
		rawEvent("e13", 13, "step", "model-output-end", `{"turn_id":"t2","request_id":"r2","status":"completed"}`),
		rawEvent("e14", 14, "turn", "end", `{"turn_id":"t2","status":"completed"}`),
	}
}

// exportedTurn 是导出文件里一轮的可解析形状（只取本用例关心的字段）。
type exportedTurn struct {
	Index int64 `json:"index"`
	Data  struct {
		Rows []struct {
			Kind string `json:"kind"`
			Text string `json:"text"`
		} `json:"rows"`
	} `json:"data"`
}

// TestExportThroughProjectionTurnHistory 用会话投影的按轮来源跑一次多轮导出：
// 目录领域只按游标接页，不折叠；文件轮次等于历史总轮次，且两轮内容都在。
func TestExportThroughProjectionTurnHistory(t *testing.T) {
	history := &staticHistory{events: twoTurnHistory()}
	source, err := projection.NewTurnHistory("s1", history)
	if err != nil {
		t.Fatalf("创建按轮历史来源: %v", err)
	}

	writes, err := write.New(write.Config{
		Sender:    noopSender{},
		Inspector: noopInspector{},
	})
	if err != nil {
		t.Fatalf("创建写登记: %v", err)
	}
	t.Cleanup(writes.Close)

	reader := &stubReader{entries: []catalog.Entry{{ID: "s1", Title: "导出标题", TitleRevision: 1}}}
	directory, err := catalog.New(catalog.Config{Reader: reader, Writes: writes, History: source})
	if err != nil {
		t.Fatalf("创建目录: %v", err)
	}
	if _, err := directory.Refresh(context.Background(), catalog.Query{}); err != nil {
		t.Fatalf("读取目录: %v", err)
	}

	path := filepath.Join(t.TempDir(), "two-turns.json")
	result, err := directory.Export(context.Background(), catalog.ExportRequest{
		SessionID: "s1",
		Path:      path,
		PageLimit: 1, // 每页一轮，强制走多页
	})
	if err != nil {
		t.Fatalf("导出: %v", err)
	}
	if result.Turns != 2 || result.Pages != 2 {
		t.Fatalf("导出轮次或页数不符：%+v", result)
	}
	if history.calls != 1 {
		t.Fatalf("按轮来源应只装载一次历史：%d", history.calls)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取导出文件: %v", err)
	}
	var document struct {
		Format        string         `json:"format"`
		FormatVersion int            `json:"formatVersion"`
		TurnCount     int            `json:"turnCount"`
		Turns         []exportedTurn `json:"turns"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("导出文件不是合法 JSON: %v", err)
	}
	if document.Format != catalog.ExportFormat || document.FormatVersion != catalog.ExportFormatVersion {
		t.Fatalf("格式或版本不符：%+v", document)
	}
	if document.TurnCount != 2 || len(document.Turns) != 2 {
		t.Fatalf("文件轮次不符：turnCount=%d，turns=%d", document.TurnCount, len(document.Turns))
	}

	want := []struct {
		user      string
		assistant string
	}{
		{"第一问", "第一答"},
		{"第二问", "第二答"},
	}
	for i, turn := range document.Turns {
		if turn.Index != int64(i+1) {
			t.Fatalf("第 %d 轮序号不符：%d", i, turn.Index)
		}
		var gotUser, gotAssistant string
		for _, row := range turn.Data.Rows {
			switch row.Kind {
			case "user_message":
				gotUser = row.Text
			case "assistant_text":
				gotAssistant = row.Text
			}
		}
		if gotUser != want[i].user || gotAssistant != want[i].assistant {
			t.Fatalf("第 %d 轮内容不符：user=%q assistant=%q", i+1, gotUser, gotAssistant)
		}
	}
}

// stubReader 返回固定的一页目录。
type stubReader struct {
	entries []catalog.Entry
}

func (r *stubReader) ReadCatalog(
	ctx context.Context,
	query catalog.Query,
	cursor string,
) (catalog.Page, error) {
	if cursor != "" {
		return catalog.Page{}, nil
	}
	return catalog.Page{Entries: append([]catalog.Entry(nil), r.entries...)}, nil
}

// noopSender 与 noopInspector 只满足写登记的构造要求；本用例不产生写意图。
type noopSender struct{}

func (noopSender) Send(ctx context.Context, req write.Request) (write.Reply, error) {
	return write.Reply{}, nil
}

type noopInspector struct{}

func (noopInspector) LookupInput(
	ctx context.Context,
	query write.InputQuery,
) (write.InputState, error) {
	return write.InputState{}, nil
}
