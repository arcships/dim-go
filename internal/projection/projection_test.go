package projection

import (
	"reflect"
	"sort"
	"testing"
)

func testEvent(id string, sequence int64, typ, name string, data map[string]any) RawEvent {
	return RawEvent{ID: id, Sequence: sequence, CreateAt: sequence, Type: typ, Name: name, Data: data}
}

func messageByID(t *testing.T, snapshot Snapshot, id string) Message {
	t.Helper()
	for _, message := range snapshot.Messages {
		if message.ID == id {
			return message
		}
	}
	t.Fatalf("message %q not found in %+v", id, snapshot.Messages)
	return Message{}
}

func TestOrderKey(t *testing.T) {
	if got := OrderKey(7); got != "00000000000000000007" {
		t.Fatalf("got %q", got)
	}
}

func TestInputIsolationAndConsumed(t *testing.T) {
	events := []RawEvent{
		testEvent("e-env", 1, "input", "message", map[string]any{
			"blocks": []any{map[string]any{"type": "raw", "text": "<context>date</context>"}},
		}),
		testEvent("e-user", 2, "input", "message", map[string]any{
			"user": map[string]any{"content": []any{
				map[string]any{"type": "text", "text": "hello"},
				map[string]any{"type": "image", "url": "ignored"},
			}},
		}),
		testEvent("e-consumed", 3, "input", "consumed", map[string]any{"input_id": "e-user"}),
	}
	r := NewReducer("main", FixedClock(1700000000999))
	if err := r.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	snapshot := r.Snapshot()
	if len(snapshot.Messages) != 1 {
		t.Fatalf("environment block leaked into messages: %+v", snapshot.Messages)
	}
	message := snapshot.Messages[0]
	if message.ID != "e-user" || message.Role != "user" || message.Content != "hello" || message.Status != "complete" {
		t.Fatalf("message=%+v", message)
	}
	if !reflect.DeepEqual(snapshot.TimelineUserMessageIds, []string{"e-user"}) {
		t.Fatalf("user ids=%+v", snapshot.TimelineUserMessageIds)
	}
	if message.OrderKey != OrderKey(2) {
		t.Fatalf("orderKey=%q", message.OrderKey)
	}
}

func TestInputJoinsUserContentTextBlocks(t *testing.T) {
	r := NewReducer("main", FixedClock(1))
	if err := r.Apply(testEvent("e-1", 1, "input", "message", map[string]any{
		"user": map[string]any{"content": []any{
			map[string]any{"type": "text", "text": "hello"},
			map[string]any{"type": "image", "source": map[string]any{"data": "ignored"}},
			map[string]any{"type": "text", "text": " world"},
		}},
		// A raw context block never wins over the user's own text.
		"blocks": []any{map[string]any{"type": "raw", "text": "<dim-state>date</dim-state>"}},
	})); err != nil {
		t.Fatal(err)
	}
	snapshot := r.Snapshot()
	if len(snapshot.Messages) != 1 {
		t.Fatalf("messages=%+v", snapshot.Messages)
	}
	if message := snapshot.Messages[0]; message.Content != "hello world" || message.Status != "sending" {
		t.Fatalf("message=%+v", message)
	}
}

func TestInputProjectsRecordedShellInputBlock(t *testing.T) {
	const marker = "<dim-shell-input kind=\"command\" status=\"completed\" exit=\"0\" truncated=\"false\">\n$ printf '&lt;x&gt;'\n&lt;x&gt; &amp; done\n\n</dim-shell-input>"
	r := NewReducer("main", FixedClock(1))
	events := []RawEvent{
		testEvent("e-env", 1, "input", "message", map[string]any{
			"blocks": []any{map[string]any{"type": "raw", "key": "state:env", "text": "<dim-state>date</dim-state>"}},
			"mode":   "append",
		}),
		testEvent("e-shell", 2, "input", "message", map[string]any{
			"blocks": []any{map[string]any{"type": "raw", "text": marker}},
			"mode":   "append",
		}),
		testEvent("e-consumed", 3, "input", "consumed", map[string]any{"input_id": "e-shell"}),
	}
	if err := r.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	snapshot := r.Snapshot()
	if len(snapshot.Messages) != 1 {
		t.Fatalf("environment block leaked into messages: %+v", snapshot.Messages)
	}
	message := snapshot.Messages[0]
	if message.ID != "e-shell" || message.Role != "user" || message.Status != "complete" {
		t.Fatalf("message=%+v", message)
	}
	if want := "$ printf '<x>'\n<x> & done"; message.Content != want {
		t.Fatalf("content=%q, want %q", message.Content, want)
	}
	if !reflect.DeepEqual(snapshot.TimelineUserMessageIds, []string{"e-shell"}) {
		t.Fatalf("user ids=%+v", snapshot.TimelineUserMessageIds)
	}
}

func TestInputJoinsMultipleShellInputBlocks(t *testing.T) {
	first := "<dim-shell-input kind=\"command\" status=\"completed\" exit=\"0\" truncated=\"false\">\n$ pwd\n/workspace\n\n</dim-shell-input>"
	second := "<dim-shell-input kind=\"command\" status=\"stopped\" truncated=\"false\">\n$ sleep 9\npartial\n\n</dim-shell-input>"
	r := NewReducer("main", FixedClock(1))
	if err := r.Apply(testEvent("e-1", 1, "input", "message", map[string]any{
		"blocks": []any{
			map[string]any{"type": "raw", "text": first},
			map[string]any{"type": "raw", "text": "<dim-branch-parent branch=\"main\">notice</dim-branch-parent>"},
			map[string]any{"type": "raw", "text": second},
		},
	})); err != nil {
		t.Fatal(err)
	}
	message := messageByID(t, r.Snapshot(), "e-1")
	if want := "$ pwd\n/workspace\n$ sleep 9\npartial"; message.Content != want {
		t.Fatalf("content=%q, want %q", message.Content, want)
	}
}

// TestInputDoesNotGuessContentField pins the upstream wire shape: the field
// does not exist on InputMessageData, so it must not fabricate a user message.
func TestInputDoesNotGuessContentField(t *testing.T) {
	r := NewReducer("main", FixedClock(1))
	if err := r.Apply(testEvent("e-1", 1, "input", "message", map[string]any{"content": "hi"})); err != nil {
		t.Fatal(err)
	}
	snapshot := r.Snapshot()
	if len(snapshot.Messages) != 0 || len(snapshot.TimelineUserMessageIds) != 0 {
		t.Fatalf("undocumented content fabricated a user message: %+v", snapshot)
	}
	if ignored := r.IgnoredEvents(); len(ignored) != 0 {
		t.Fatalf("known input type reported as ignored: %+v", ignored)
	}
}

// TestInputProjectsTextBlocks pins the upstream precedence inside `blocks`.
// The current wire InputBlock enum only has `raw`, so this is defensive parity
// with inputTextFromData rather than a reachable shape.
func TestInputProjectsTextBlocks(t *testing.T) {
	r := NewReducer("main", FixedClock(1))
	if err := r.Apply(testEvent("e-1", 1, "input", "message", map[string]any{
		"blocks": []any{
			map[string]any{"type": "text", "text": "from "},
			map[string]any{"type": "text", "text": "blocks"},
			map[string]any{"type": "raw", "text": "<dim-shell-input kind=\"command\" status=\"completed\" exit=\"0\" truncated=\"false\">\n$ pwd\n/x\n\n</dim-shell-input>"},
		},
	})); err != nil {
		t.Fatal(err)
	}
	if message := messageByID(t, r.Snapshot(), "e-1"); message.Content != "from blocks" {
		t.Fatalf("content=%q", message.Content)
	}
}

func TestTextThinkingBoundariesAndDuplicateSequence(t *testing.T) {
	r := NewReducer("main", FixedClock(1700000000999))
	events := []RawEvent{
		testEvent("e1", 1, "step", "model-start", map[string]any{"turn_id": "t", "request_id": "r"}),
		testEvent("e2", 2, "step", "thinking", map[string]any{"delta": "think"}),
		testEvent("e3", 3, "step", "text", map[string]any{"delta": "hello"}),
		testEvent("e4", 4, "step", "text", map[string]any{"delta": " world"}),
		testEvent("e5", 5, "step", "thinking", map[string]any{"delta": "again"}),
		testEvent("e6", 6, "step", "model-output-end", map[string]any{"status": "completed"}),
	}
	if err := r.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	if err := r.Apply(events[3]); err != nil {
		t.Fatal(err)
	}
	snapshot := r.Snapshot()
	if len(snapshot.Messages) != 1 {
		t.Fatalf("messages=%d", len(snapshot.Messages))
	}
	message := snapshot.Messages[0]
	if message.ID != "a:main:t:r" || message.Content != "hello world" || message.Status != "complete" {
		t.Fatalf("message=%+v", message)
	}
	if len(message.Events) != 3 || message.Events[0].ID != "e2" || message.Events[1].ID != "e3" || message.Events[2].ID != "e5" {
		t.Fatalf("events=%+v", message.Events)
	}
	if message.Events[0].Type != "thinking" || message.Events[1].Type != "text_delta" || message.Events[2].Type != "thinking" {
		t.Fatalf("event types=%+v", message.Events)
	}
	for _, block := range message.Events {
		if block.Status != "complete" || block.EndTime == nil {
			t.Fatalf("open block after output end: %+v", block)
		}
	}
}

func TestToolLifecycleMergesCallAndResult(t *testing.T) {
	events := []RawEvent{
		testEvent("e1", 1, "step", "model-start", map[string]any{"turn_id": "t", "request_id": "r"}),
		testEvent("e2", 2, "step", "tool-use-start", map[string]any{
			"call_id": "c1", "name": "read_file", "turn_id": "t", "request_id": "r",
		}),
		testEvent("e3", 3, "step", "tool-use", map[string]any{
			"call": map[string]any{"id": "c1", "name": "read_file", "input": map[string]any{"path": "README.md"}},
		}),
		testEvent("e4", 4, "step", "tool-completed", map[string]any{
			"call_id": "c1", "result": map[string]any{"content": "ok"},
		}),
		testEvent("e5", 5, "step", "model-output-end", map[string]any{"status": "completed"}),
	}
	r := NewReducer("main", FixedClock(1700000000999))
	if err := r.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	snapshot := r.Snapshot()
	if len(snapshot.Messages) != 2 {
		t.Fatalf("messages=%+v", snapshot.Messages)
	}
	tool := messageByID(t, snapshot, "c1")
	if tool.Role != "tool" || tool.Content != "ok" || tool.Status != "complete" {
		t.Fatalf("tool=%+v", tool)
	}
	if tool.OrderKey != OrderKey(2) {
		t.Fatalf("tool orderKey=%q", tool.OrderKey)
	}
	if len(tool.Events) != 2 {
		t.Fatalf("tool events=%+v", tool.Events)
	}
	call := tool.Events[0]
	if call.ID != "tool:c1:call" || call.Type != "tool_call" || call.Status != "complete" {
		t.Fatalf("call=%+v", call)
	}
	if call.ToolName != "read_file" || !reflect.DeepEqual(call.ToolParams, map[string]any{"path": "README.md"}) {
		t.Fatalf("call facts=%+v", call)
	}
	if call.EndTime == nil || *call.EndTime != 4 {
		t.Fatalf("call end=%+v", call.EndTime)
	}
	result := tool.Events[1]
	if result.ID != "tool:c1:result" || result.Type != "tool_result" || result.Status != "complete" || result.Content != "ok" {
		t.Fatalf("result=%+v", result)
	}
	if result.StartTime != 4 || result.EndTime == nil || *result.EndTime != 4 {
		t.Fatalf("result times=%+v", result)
	}
	assistant := messageByID(t, snapshot, "a:main:t:r")
	if assistant.Status != "complete" || len(assistant.Events) != 0 {
		t.Fatalf("assistant=%+v", assistant)
	}
}

func TestToolResultBeforeCallConverges(t *testing.T) {
	events := []RawEvent{
		testEvent("e1", 1, "step", "tool-completed", map[string]any{
			"call_id": "c2",
			"result":  map[string]any{"isError": true, "content": "boom"},
		}),
		testEvent("e2", 2, "step", "tool-use", map[string]any{
			"call": map[string]any{"id": "c2", "name": "write", "input": map[string]any{"path": "a.txt"}},
		}),
	}
	r := NewReducer("main", FixedClock(1))
	if err := r.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	snapshot := r.Snapshot()
	if len(snapshot.Messages) != 1 {
		t.Fatalf("tool duplicated: %+v", snapshot.Messages)
	}
	tool := messageByID(t, snapshot, "c2")
	if tool.Status != "error" || tool.Content != "boom" || tool.OrderKey != OrderKey(2) {
		t.Fatalf("tool=%+v", tool)
	}
	if len(tool.Events) != 2 || tool.Events[0].Type != "tool_call" || tool.Events[1].Type != "tool_result" {
		t.Fatalf("events=%+v", tool.Events)
	}
	if tool.Events[0].ToolName != "write" || tool.Events[0].Status != "error" {
		t.Fatalf("merged call=%+v", tool.Events[0])
	}
	if tool.Events[1].ID != "tool:c2:result" || tool.Events[1].Status != "error" {
		t.Fatalf("pending result=%+v", tool.Events[1])
	}
}

func TestTwoResultsOnOneCallKeepFixedResultID(t *testing.T) {
	events := []RawEvent{
		testEvent("e1", 1, "step", "tool-use", map[string]any{
			"call": map[string]any{"id": "c1", "name": "read", "input": map[string]any{"path": "x"}},
		}),
		testEvent("e2", 2, "step", "tool-completed", map[string]any{
			"call_id": "c1", "result": map[string]any{"content": "first"},
		}),
		testEvent("e3", 3, "step", "tool-completed", map[string]any{
			"call_id": "c1", "result": map[string]any{"content": "second"},
		}),
	}
	r := NewReducer("main", FixedClock(1))
	if err := r.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	tool := messageByID(t, r.Snapshot(), "c1")
	if tool.Content != "second" || tool.Status != "complete" {
		t.Fatalf("tool=%+v", tool)
	}
	if len(tool.Events) != 3 {
		t.Fatalf("events=%+v", tool.Events)
	}
	// Upstream TimelineFold pushes every result with the same synthetic id and
	// does not dedupe repeated result events for one call.
	if tool.Events[1].ID != "tool:c1:result" || tool.Events[2].ID != "tool:c1:result" {
		t.Fatalf("result ids=%q,%q", tool.Events[1].ID, tool.Events[2].ID)
	}
	if tool.Events[1].Content != "first" || tool.Events[2].Content != "second" {
		t.Fatalf("result contents=%q,%q", tool.Events[1].Content, tool.Events[2].Content)
	}
}

func TestToolCallBatchKeepsStableOrder(t *testing.T) {
	events := []RawEvent{
		testEvent("e1", 1, "step", "tool-call", map[string]any{
			"calls": []any{
				map[string]any{"id": "c1", "name": "first", "input": map[string]any{"n": 1}},
				map[string]any{"id": "c2", "name": "second"},
			},
		}),
	}
	r := NewReducer("main", FixedClock(1))
	if err := r.Apply(events[0]); err != nil {
		t.Fatal(err)
	}
	snapshot := r.Snapshot()
	if len(snapshot.Messages) != 2 {
		t.Fatalf("messages=%+v", snapshot.Messages)
	}
	if snapshot.Messages[0].ID != "c1" || snapshot.Messages[1].ID != "c2" {
		t.Fatalf("batch order=%+v", snapshot.Messages)
	}
	if snapshot.Messages[0].OrderKey != OrderKey(1) || snapshot.Messages[1].OrderKey != OrderKey(1) {
		t.Fatalf("orderKeys=%+v", snapshot.Messages)
	}
}

func TestHistoryAndLivePermutationSnapshotsMatch(t *testing.T) {
	events := []RawEvent{
		testEvent("e1", 1, "step", "model-start", map[string]any{"turn_id": "t", "request_id": "r"}),
		testEvent("e2", 2, "step", "thinking", map[string]any{"delta": "why"}),
		testEvent("e3", 3, "step", "text", map[string]any{"delta": "answer"}),
		testEvent("e4", 4, "step", "tool-use", map[string]any{
			"call": map[string]any{"id": "c1", "name": "read_file", "input": map[string]any{"path": "x"}},
		}),
		testEvent("e5", 5, "step", "tool-completed", map[string]any{
			"call_id": "c1", "result": map[string]any{"content": "ok"},
		}),
		testEvent("e6", 6, "step", "model-output-end", map[string]any{"status": "completed"}),
	}
	history := NewReducer("main", FixedClock(9))
	if err := history.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	live := NewReducer("main", FixedClock(9))
	// The tool result is an allowed out-of-order lifecycle fact: it arrives
	// before its call and must converge into the same tool message. Duplicates
	// and text/thinking order are otherwise preserved.
	liveOrder := []RawEvent{events[0], events[4], events[1], events[2], events[3], events[4], events[3], events[5]}
	for _, event := range liveOrder {
		if err := live.Apply(event); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(history.Snapshot(), live.Snapshot()) {
		t.Fatalf("history=%+v\nlive=%+v", history.Snapshot(), live.Snapshot())
	}
}

func TestBranchIsolation(t *testing.T) {
	events := []RawEvent{
		testEvent("e1", 1, "step", "model-start", map[string]any{"turn_id": "t", "request_id": "r"}),
		testEvent("e2", 2, "step", "text", map[string]any{"delta": "hi"}),
	}
	main := NewReducer("main", FixedClock(1))
	branch := NewReducer("b-2", FixedClock(1))
	for _, reducer := range []*Reducer{main, branch} {
		if err := reducer.ApplyHistory(events); err != nil {
			t.Fatal(err)
		}
	}
	mainMessage := main.Snapshot().Messages[0]
	branchMessage := branch.Snapshot().Messages[0]
	if mainMessage.ID != "a:main:t:r" || branchMessage.ID != "a:b-2:t:r" {
		t.Fatalf("main=%q branch=%q", mainMessage.ID, branchMessage.ID)
	}
}

func TestHistorySortAndLiveIdempotency(t *testing.T) {
	events := []RawEvent{
		testEvent("b", 2, "step", "model-start", map[string]any{"turn_id": "t", "request_id": "r"}),
		testEvent("a", 1, "input", "message", map[string]any{"content": "x"}),
	}
	r := NewReducer("main", FixedClock(1))
	sort.Slice(events, func(i, j int) bool { return events[i].Sequence < events[j].Sequence })
	for _, event := range events {
		if err := r.Apply(event); err != nil {
			t.Fatal(err)
		}
	}
	before := r.Snapshot()
	if err := r.Apply(events[0]); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, r.Snapshot()) {
		t.Fatal("duplicate changed snapshot")
	}
}

func TestConsumedBeforeInputConverges(t *testing.T) {
	r := NewReducer("main", FixedClock(1))
	// A live consumed notice can arrive before history replays its input.
	if err := r.Apply(testEvent("e-2", 2, "input", "consumed", map[string]any{"input_id": "e-1"})); err != nil {
		t.Fatal(err)
	}
	if err := r.Apply(testEvent("e-1", 1, "input", "message", map[string]any{
		"user": map[string]any{"content": []any{map[string]any{"type": "text", "text": "hi"}}},
	})); err != nil {
		t.Fatal(err)
	}
	snapshot := r.Snapshot()
	if len(snapshot.Messages) != 1 {
		t.Fatalf("messages=%+v", snapshot.Messages)
	}
	message := snapshot.Messages[0]
	if message.ID != "e-1" || message.Status != "complete" || message.OrderKey != OrderKey(1) {
		t.Fatalf("message=%+v", message)
	}
	if message.UpdatedAt != 2 {
		t.Fatalf("updatedAt=%d", message.UpdatedAt)
	}
}

func TestModelOutputEndStatusClosesBlocks(t *testing.T) {
	cases := []struct {
		name   string
		status string
		want   string
	}{
		{name: "completed", status: "completed", want: "complete"},
		{name: "failed", status: "failed", want: "error"},
		{name: "cancelled", status: "cancelled", want: "cancelled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewReducer("main", FixedClock(1))
			events := []RawEvent{
				testEvent("e-1", 1, "step", "model-start", map[string]any{"turn_id": "t", "request_id": "r"}),
				testEvent("e-2", 2, "step", "thinking", map[string]any{"delta": "why"}),
				testEvent("e-3", 3, "step", "text", map[string]any{"delta": "hi"}),
				testEvent("e-4", 4, "step", "model-output-end", map[string]any{"status": tc.status}),
			}
			if err := r.ApplyHistory(events); err != nil {
				t.Fatal(err)
			}
			message := r.Snapshot().Messages[0]
			if message.Status != tc.want {
				t.Fatalf("message status=%q", message.Status)
			}
			// The thinking block closed at the text boundary; only the running
			// text block receives the model-output-end terminal status.
			thinking := message.Events[0]
			if thinking.Status != "complete" || thinking.EndTime == nil || *thinking.EndTime != 3 {
				t.Fatalf("thinking block=%+v", thinking)
			}
			text := message.Events[1]
			if text.Status != tc.want || text.EndTime == nil || *text.EndTime != 4 {
				t.Fatalf("text block=%+v", text)
			}
		})
	}
}

func TestUnknownEventDoesNotFabricateMessages(t *testing.T) {
	r := NewReducer("main", FixedClock(1))
	if err := r.Apply(testEvent("e-1", 1, "session", "subscription-changed", map[string]any{"kind": "approval"})); err != nil {
		t.Fatal(err)
	}
	if len(r.Snapshot().Messages) != 0 {
		t.Fatalf("unknown event fabricated messages: %+v", r.Snapshot().Messages)
	}
	ignored := r.IgnoredEvents()
	if len(ignored) != 1 || ignored[0].ID != "e-1" {
		t.Fatalf("ignored=%+v", ignored)
	}
}

func TestThinkingDeltasShareBlock(t *testing.T) {
	events := []RawEvent{
		testEvent("e-1", 1, "step", "model-start", map[string]any{"turn_id": "t", "request_id": "r"}),
		testEvent("e-2", 2, "step", "thinking", map[string]any{"delta": "a"}),
		testEvent("e-3", 3, "step", "thinking", map[string]any{"delta": "b"}),
		testEvent("e-4", 4, "step", "text", map[string]any{"delta": "x"}),
		testEvent("e-5", 5, "step", "thinking", map[string]any{"delta": "c"}),
		testEvent("e-6", 6, "step", "model-output-end", map[string]any{"status": "completed"}),
	}
	r := NewReducer("main", FixedClock(1))
	if err := r.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	message := messageByID(t, r.Snapshot(), "a:main:t:r")
	if message.Content != "x" {
		t.Fatalf("content=%q", message.Content)
	}
	want := []struct {
		id      string
		typ     string
		content string
		start   int64
		end     int64
	}{
		{"e-2", "thinking", "ab", 2, 4},
		{"e-4", "text_delta", "x", 4, 5},
		{"e-5", "thinking", "c", 5, 6},
	}
	if len(message.Events) != len(want) {
		t.Fatalf("events=%+v", message.Events)
	}
	for i, expected := range want {
		block := message.Events[i]
		if block.ID != expected.id || block.Type != expected.typ || block.Content != expected.content || block.Status != "complete" {
			t.Fatalf("block[%d]=%+v", i, block)
		}
		if block.StartTime != expected.start || block.EndTime == nil || *block.EndTime != expected.end {
			t.Fatalf("block[%d] times=%+v", i, block)
		}
	}
}

func TestDeltaWithoutModelStartCreatesAssistant(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		r := NewReducer("main", FixedClock(1))
		if err := r.Apply(testEvent("e-1", 1, "step", "text", map[string]any{
			"turn_id": "t", "request_id": "r", "delta": "hi",
		})); err != nil {
			t.Fatal(err)
		}
		message := messageByID(t, r.Snapshot(), "a:main:t:r")
		if message.Content != "hi" || message.Status != "streaming" {
			t.Fatalf("message=%+v", message)
		}
		if len(message.Events) != 1 || message.Events[0].ID != "e-1" || message.Events[0].Type != "text_delta" {
			t.Fatalf("events=%+v", message.Events)
		}
	})
	t.Run("thinking", func(t *testing.T) {
		r := NewReducer("main", FixedClock(1))
		if err := r.Apply(testEvent("e-1", 1, "step", "thinking", map[string]any{
			"turn_id": "t", "request_id": "r", "delta": "why",
		})); err != nil {
			t.Fatal(err)
		}
		message := messageByID(t, r.Snapshot(), "a:main:t:r")
		if message.Content != "" || len(message.Events) != 1 || message.Events[0].Type != "thinking" {
			t.Fatalf("message=%+v", message)
		}
	})
}

func twoAssistantEvents() []RawEvent {
	return []RawEvent{
		testEvent("e-1", 1, "step", "model-start", map[string]any{"turn_id": "t-1", "request_id": "r-1"}),
		testEvent("e-2", 2, "step", "model-start", map[string]any{"turn_id": "t-1", "request_id": "r-2"}),
		testEvent("e-3", 3, "step", "thinking", map[string]any{"turn_id": "t-1", "request_id": "r-1", "delta": "thinkA"}),
		testEvent("e-4", 4, "step", "text", map[string]any{"turn_id": "t-1", "request_id": "r-2", "delta": "B"}),
		testEvent("e-5", 5, "step", "text", map[string]any{"turn_id": "t-1", "request_id": "r-1", "delta": "A"}),
		testEvent("e-6", 6, "step", "thinking", map[string]any{"turn_id": "t-1", "request_id": "r-2", "delta": "thinkB"}),
		testEvent("e-7", 7, "step", "text", map[string]any{"turn_id": "t-1", "request_id": "r-2", "delta": "B2"}),
		testEvent("e-8", 8, "step", "model-output-end", map[string]any{"turn_id": "t-1", "request_id": "r-1", "status": "completed"}),
		testEvent("e-9", 9, "step", "model-output-end", map[string]any{"turn_id": "t-1", "request_id": "r-2", "status": "completed"}),
	}
}

func assertTwoAssistants(t *testing.T, snapshot Snapshot) {
	t.Helper()
	if len(snapshot.Messages) != 2 {
		t.Fatalf("messages=%+v", snapshot.Messages)
	}
	first := messageByID(t, snapshot, "a:main:t-1:r-1")
	if first.Content != "A" || first.Status != "complete" || first.CreatedAt != 1 || first.UpdatedAt != 8 || first.OrderKey != OrderKey(1) {
		t.Fatalf("assistant r-1=%+v", first)
	}
	firstWant := []struct {
		id      string
		typ     string
		content string
		start   int64
		end     int64
	}{
		{"e-3", "thinking", "thinkA", 3, 5},
		{"e-5", "text_delta", "A", 5, 8},
	}
	if len(first.Events) != len(firstWant) {
		t.Fatalf("assistant r-1 events=%+v", first.Events)
	}
	for i, expected := range firstWant {
		block := first.Events[i]
		if block.ID != expected.id || block.Type != expected.typ || block.Content != expected.content || block.Status != "complete" {
			t.Fatalf("assistant r-1 block[%d]=%+v", i, block)
		}
		if block.StartTime != expected.start || block.EndTime == nil || *block.EndTime != expected.end {
			t.Fatalf("assistant r-1 block[%d] times=%+v", i, block)
		}
	}
	second := messageByID(t, snapshot, "a:main:t-1:r-2")
	if second.Content != "BB2" || second.Status != "complete" || second.CreatedAt != 2 || second.UpdatedAt != 9 || second.OrderKey != OrderKey(2) {
		t.Fatalf("assistant r-2=%+v", second)
	}
	secondWant := []struct {
		id      string
		typ     string
		content string
		start   int64
		end     int64
	}{
		{"e-4", "text_delta", "B", 4, 6},
		{"e-6", "thinking", "thinkB", 6, 7},
		{"e-7", "text_delta", "B2", 7, 9},
	}
	if len(second.Events) != len(secondWant) {
		t.Fatalf("assistant r-2 events=%+v", second.Events)
	}
	for i, expected := range secondWant {
		block := second.Events[i]
		if block.ID != expected.id || block.Type != expected.typ || block.Content != expected.content || block.Status != "complete" {
			t.Fatalf("assistant r-2 block[%d]=%+v", i, block)
		}
		if block.StartTime != expected.start || block.EndTime == nil || *block.EndTime != expected.end {
			t.Fatalf("assistant r-2 block[%d] times=%+v", i, block)
		}
	}
}

func TestTwoAssistantsInterleaved(t *testing.T) {
	r := NewReducer("main", FixedClock(1))
	if err := r.ApplyHistory(twoAssistantEvents()); err != nil {
		t.Fatal(err)
	}
	assertTwoAssistants(t, r.Snapshot())
}

func TestTwoAssistantsLiveHistoryPermutation(t *testing.T) {
	events := twoAssistantEvents()
	history := NewReducer("main", FixedClock(1))
	if err := history.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	live := NewReducer("main", FixedClock(1))
	// Per-assistant order is preserved while the two requests interleave
	// differently from sequence order.
	liveOrder := []RawEvent{events[0], events[1], events[3], events[2], events[5], events[4], events[6], events[7], events[8]}
	for _, event := range liveOrder {
		if err := live.Apply(event); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(history.Snapshot(), live.Snapshot()) {
		t.Fatalf("history=%+v\nlive=%+v", history.Snapshot(), live.Snapshot())
	}
	// Attach overlap: replaying the history page after live must be idempotent.
	if err := live.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(history.Snapshot(), live.Snapshot()) {
		t.Fatalf("history=%+v\nlive after history replay=%+v", history.Snapshot(), live.Snapshot())
	}
}

func TestTwoResultsOutOfOrderHistoryLiveEqual(t *testing.T) {
	events := []RawEvent{
		testEvent("e-1", 1, "step", "tool-use", map[string]any{
			"call": map[string]any{"id": "c1", "name": "read", "input": map[string]any{"path": "x"}},
		}),
		testEvent("e-2", 2, "step", "tool-completed", map[string]any{
			"call_id": "c1", "result": map[string]any{"content": "first"},
		}),
		testEvent("e-3", 3, "step", "tool-completed", map[string]any{
			"call_id": "c1", "result": map[string]any{"content": "second"},
		}),
	}
	history := NewReducer("main", FixedClock(1))
	if err := history.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	live := NewReducer("main", FixedClock(1))
	for _, event := range []RawEvent{events[0], events[2], events[1]} {
		if err := live.Apply(event); err != nil {
			t.Fatal(err)
		}
	}
	historySnapshot := history.Snapshot()
	if !reflect.DeepEqual(historySnapshot, live.Snapshot()) {
		t.Fatalf("history=%+v\nlive=%+v", historySnapshot, live.Snapshot())
	}
	tool := messageByID(t, historySnapshot, "c1")
	if tool.Content != "second" || tool.Status != "complete" {
		t.Fatalf("tool=%+v", tool)
	}
	call := tool.Events[0]
	if call.EndTime == nil || *call.EndTime != 3 {
		t.Fatalf("call end=%+v", call.EndTime)
	}
	if len(tool.Events) != 3 || tool.Events[1].Content != "first" || tool.Events[2].Content != "second" {
		t.Fatalf("events=%+v", tool.Events)
	}
}

func TestNullUserIsEnvironmentContext(t *testing.T) {
	r := NewReducer("main", FixedClock(1))
	if err := r.Apply(testEvent("e-1", 1, "input", "message", map[string]any{
		"user":   nil,
		"blocks": []any{map[string]any{"type": "raw", "text": "<context>date</context>"}},
	})); err != nil {
		t.Fatal(err)
	}
	if messages := r.Snapshot().Messages; len(messages) != 0 {
		t.Fatalf("null user leaked: %+v", messages)
	}
	if err := r.Apply(testEvent("e-2", 2, "input", "message", map[string]any{
		"user": map[string]any{"content": []any{map[string]any{"type": "text", "text": "hi"}}},
	})); err != nil {
		t.Fatal(err)
	}
	snapshot := r.Snapshot()
	message := messageByID(t, snapshot, "e-2")
	if message.Role != "user" || message.Content != "hi" {
		t.Fatalf("message=%+v", message)
	}
	if !reflect.DeepEqual(snapshot.TimelineUserMessageIds, []string{"e-2"}) {
		t.Fatalf("user ids=%+v", snapshot.TimelineUserMessageIds)
	}
}

func TestCompleteInputRejectsUndocumentedAliases(t *testing.T) {
	r := NewReducer("main", FixedClock(1))
	events := []RawEvent{
		testEvent("e-1", 1, "input", "message", map[string]any{
			"user": map[string]any{"content": []any{map[string]any{"type": "text", "text": "hi"}}},
		}),
		testEvent("e-2", 2, "input", "consumed", map[string]any{"message_id": "e-1"}),
	}
	for _, event := range events {
		if err := r.Apply(event); err != nil {
			t.Fatal(err)
		}
	}
	if message := messageByID(t, r.Snapshot(), "e-1"); message.Status != "sending" {
		t.Fatalf("alias completed input: %+v", message)
	}
	if err := r.Apply(testEvent("e-3", 3, "input", "consumed", map[string]any{"input_id": "e-1"})); err != nil {
		t.Fatal(err)
	}
	if message := messageByID(t, r.Snapshot(), "e-1"); message.Status != "complete" {
		t.Fatalf("documented key did not complete input: %+v", message)
	}
}

func TestInputDeletedRemovesMessage(t *testing.T) {
	r := NewReducer("main", FixedClock(1))
	events := []RawEvent{
		testEvent("e-1", 1, "input", "message", map[string]any{
			"user": map[string]any{"content": []any{map[string]any{"type": "text", "text": "hi"}}},
		}),
		testEvent("e-2", 2, "input", "consumed", map[string]any{"input_id": "e-1"}),
		testEvent("e-3", 3, "input", "deleted", map[string]any{"input_id": "e-1"}),
	}
	for _, event := range events {
		if err := r.Apply(event); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := r.Snapshot()
	if len(snapshot.Messages) != 0 {
		t.Fatalf("deleted input remained: %+v", snapshot.Messages)
	}
	if len(snapshot.TimelineUserMessageIds) != 0 {
		t.Fatalf("deleted input id remained: %+v", snapshot.TimelineUserMessageIds)
	}
	// A later consumed notice must not resurrect the deleted message.
	if err := r.Apply(testEvent("e-4", 4, "input", "consumed", map[string]any{"input_id": "e-1"})); err != nil {
		t.Fatal(err)
	}
	if len(r.Snapshot().Messages) != 0 {
		t.Fatalf("deleted input resurrected: %+v", r.Snapshot().Messages)
	}
}
