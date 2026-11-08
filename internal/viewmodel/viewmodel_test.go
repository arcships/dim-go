package viewmodel

import (
	"reflect"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// testEvent builds one raw App Server event; CreateAt follows Sequence so the
// fixtures need no clock of their own.
func testEvent(id string, sequence int64, typ, name string, data map[string]any) projection.RawEvent {
	return projection.RawEvent{ID: id, Sequence: sequence, CreateAt: sequence, Type: typ, Name: name, Data: data}
}

func userInput(id string, sequence int64, text string) projection.RawEvent {
	return testEvent(id, sequence, "input", "message", map[string]any{
		"user": map[string]any{"content": []any{
			map[string]any{"type": "text", "text": text},
		}},
	})
}

func inputApplied(id string, sequence int64, inputID string) projection.RawEvent {
	return testEvent(id, sequence, "input", "applied", map[string]any{"input_id": inputID})
}

func modelStart(id string, sequence int64) projection.RawEvent {
	return testEvent(id, sequence, "step", "model-start", map[string]any{"turn_id": "t", "request_id": "r"})
}

func textDelta(id string, sequence int64, delta string) projection.RawEvent {
	return testEvent(id, sequence, "step", "text", map[string]any{
		"turn_id": "t", "request_id": "r", "delta": delta,
	})
}

func thinkingDelta(id string, sequence int64, delta string) projection.RawEvent {
	return testEvent(id, sequence, "step", "thinking", map[string]any{
		"turn_id": "t", "request_id": "r", "delta": delta,
	})
}

func outputEnd(id string, sequence int64, status string) projection.RawEvent {
	return testEvent(id, sequence, "step", "model-output-end", map[string]any{
		"turn_id": "t", "request_id": "r", "status": status,
	})
}

func toolUse(id string, sequence int64, callID, name string, input map[string]any) projection.RawEvent {
	return testEvent(id, sequence, "step", "tool-use", map[string]any{
		"call": map[string]any{"id": callID, "name": name, "input": input},
	})
}

func toolResult(id string, sequence int64, callID, content string, isError bool) projection.RawEvent {
	return testEvent(id, sequence, "step", "tool-completed", map[string]any{
		"call_id": callID,
		"result":  map[string]any{"content": content, "isError": isError},
	})
}

// snapshotOf folds the events through the projection's own reducer, so the
// adapter is tested against a snapshot the projection builds, not a hand-made
// one.
func snapshotOf(t *testing.T, events ...projection.RawEvent) projection.Snapshot {
	t.Helper()
	reducer := projection.NewReducer("main", projection.FixedClock(1000))
	if err := reducer.ApplyHistory(events); err != nil {
		t.Fatalf("projection: %v", err)
	}
	return reducer.Snapshot()
}

// assistantID is the ID the projection gives the assistant message of turn "t",
// request "r" on branch "main".
const assistantID = "a:main:t:r"

func TestBuildUserAndAssistantRows(t *testing.T) {
	snapshot := snapshotOf(t,
		userInput("u1", 1, "hello"),
		inputApplied("e2", 2, "u1"),
		modelStart("e3", 3),
		textDelta("e4", 4, "Hi"),
		textDelta("e5", 5, " there"),
		outputEnd("e6", 6, "completed"),
	)

	rows := Build(snapshot)
	want := []ui.Message{
		{ID: "u1", Author: ui.AuthorUser, Content: "hello"},
		{ID: assistantID, Author: ui.AuthorAssistant, Content: "Hi there"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

func TestBuildKeepsProjectionOrder(t *testing.T) {
	// A second turn, interleaved with a tool call, must keep the snapshot's
	// order rather than the adapter re-sorting anything.
	snapshot := snapshotOf(t,
		userInput("u1", 1, "first"),
		modelStart("e2", 2),
		textDelta("e3", 3, "one"),
		outputEnd("e4", 4, "completed"),
		userInput("u2", 5, "second"),
		toolUse("e6", 6, "c1", "bash", map[string]any{"command": "go test"}),
		toolResult("e7", 7, "c1", "ok", false),
	)
	rows := Build(snapshot)

	var ids []string
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	want := []string{"u1", assistantID, "u2", "c1"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("row IDs = %v, want the snapshot's order %v", ids, want)
	}
	if got := snapshot.Messages[len(snapshot.Messages)-1].ID; got != "c1" {
		t.Fatalf("snapshot order is %q last, so the fixture does not test what it claims", got)
	}
}

// TestBuildThinkingIsNotVisibleContent checks the choice this adapter makes: the
// projection keeps a thinking-only turn as a message, so the shell keeps its row
// with empty content rather than dropping the turn or showing the thinking text.
func TestBuildThinkingIsNotVisibleContent(t *testing.T) {
	snapshot := snapshotOf(t,
		modelStart("e1", 1),
		thinkingDelta("e2", 2, "weighing the options"),
		outputEnd("e3", 3, "completed"),
	)
	rows := Build(snapshot)
	want := []ui.Message{{ID: assistantID, Author: ui.AuthorAssistant, Content: ""}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want one assistant row with empty content", rows)
	}
}

func TestBuildToolRow(t *testing.T) {
	snapshot := snapshotOf(t,
		modelStart("e1", 1),
		toolUse("e2", 2, "c1", "read_file", map[string]any{"path": "README.md"}),
		toolResult("e3", 3, "c1", "ok", false),
	)
	rows := Build(snapshot)

	if len(rows) != 2 {
		t.Fatalf("rows = %#v, want the assistant and the tool", rows)
	}
	want := ui.Message{
		ID:     "c1",
		Author: ui.AuthorTool,
		Tool: ui.Tool{
			Name:   "read_file",
			Params: `{"path":"README.md"}`,
			Result: "ok",
			Status: ui.ToolComplete,
		},
	}
	if !reflect.DeepEqual(rows[1], want) {
		t.Fatalf("tool row = %#v, want %#v", rows[1], want)
	}
}

func TestBuildToolParamsAreSorted(t *testing.T) {
	snapshot := snapshotOf(t, toolUse("e1", 1, "c1", "bash", map[string]any{
		"zeta":   "z",
		"alpha":  1,
		"nested": map[string]any{"gamma": 3, "beta": 2},
	}))
	rows := Build(snapshot)
	if len(rows) != 1 {
		t.Fatalf("rows = %#v, want the one tool row", rows)
	}
	want := `{"alpha":1,"nested":{"beta":2,"gamma":3},"zeta":"z"}`
	if got := rows[0].Tool.Params; got != want {
		t.Fatalf("params = %q, want %q: object keys must sort, at every level", got, want)
	}

	// The same facts built again give the same line, whatever order the map
	// iterates in.
	again := Build(snapshotOf(t, toolUse("e1", 1, "c1", "bash", map[string]any{
		"nested": map[string]any{"beta": 2, "gamma": 3},
		"alpha":  1,
		"zeta":   "z",
	})))
	if !reflect.DeepEqual(rows, again) {
		t.Fatalf("params are not deterministic: %q then %q", rows[0].Tool.Params, again[0].Tool.Params)
	}
}

func TestBuildToolPreviewsAreBounded(t *testing.T) {
	// The bound the adapter promises, pinned here: changing it is a deliberate
	// decision about how much of a call leaves the projection.
	const previewRunes = 160

	long := strings.Repeat("x", 4*previewRunes)
	snapshot := snapshotOf(t,
		toolUse("e1", 1, "c1", "bash", map[string]any{"command": long}),
		toolResult("e2", 2, "c1", "line one\nline two\tline three", false),
	)
	rows := Build(snapshot)
	tool := rows[0].Tool

	params := []rune(tool.Params)
	if len(params) != previewRunes+1 || params[len(params)-1] != '…' {
		t.Fatalf("params are %d runes, want %d and an ellipsis", len(params), previewRunes+1)
	}
	if !strings.HasPrefix(tool.Params, `{"command":"xxx`) {
		t.Fatalf("params %q do not start with the call's own text", tool.Params)
	}
	if strings.ContainsAny(tool.Result, "\n\t") {
		t.Fatalf("result %q keeps its line breaks", tool.Result)
	}
	if tool.Result != "line one line two line three" {
		t.Fatalf("result = %q, want one line", tool.Result)
	}
}

// TestBuildToolResultBeforeCallConverges checks the attach overlap: a result
// replayed before its call still gives one row, with the name the later call
// carries and the result the projection settled.
func TestBuildToolResultBeforeCallConverges(t *testing.T) {
	snapshot := snapshotOf(t,
		toolResult("e1", 1, "c2", "boom", true),
		toolUse("e2", 2, "c2", "write", map[string]any{"path": "a.txt"}),
	)
	rows := Build(snapshot)
	want := []ui.Message{{
		ID:     "c2",
		Author: ui.AuthorTool,
		Tool: ui.Tool{
			Name:   "write",
			Params: `{"path":"a.txt"}`,
			Result: "boom",
			Status: ui.ToolError,
		},
	}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

// TestBuildReplayIsIdempotent checks that a replayed sequence adds no row: the
// projection dedupes it, and the adapter maps the same snapshot to the same rows
// every time it is asked.
func TestBuildReplayIsIdempotent(t *testing.T) {
	events := []projection.RawEvent{
		userInput("u1", 1, "hello"),
		modelStart("e2", 2),
		textDelta("e3", 3, "hi"),
		toolUse("e4", 4, "c1", "bash", map[string]any{"command": "ls"}),
		toolResult("e5", 5, "c1", "a.txt", false),
		outputEnd("e6", 6, "completed"),
	}
	reducer := projection.NewReducer("main", projection.FixedClock(1000))
	if err := reducer.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	once := Build(reducer.Snapshot())

	// The same sequences again, as a history page overlapping live notices.
	if err := reducer.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if err := reducer.Apply(event); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := reducer.Snapshot()
	twice := Build(snapshot)
	if !reflect.DeepEqual(once, twice) {
		t.Fatalf("a replayed sequence changed the rows:\n once  = %#v\n twice = %#v", once, twice)
	}
	if len(twice) != 3 {
		t.Fatalf("%d rows after the replay, want user, assistant and tool", len(twice))
	}

	// Build neither mutates the snapshot it was given nor depends on repetition.
	before := reducer.Snapshot()
	_ = Build(snapshot)
	if !reflect.DeepEqual(before, snapshot) {
		t.Fatal("Build mutated the caller's snapshot")
	}
	if !reflect.DeepEqual(Build(snapshot), twice) {
		t.Fatal("Build returned different rows for the same snapshot")
	}
}

// TestBuildToolStatuses checks the mapping of every status word the projection
// can settle a tool with. Cancelled is built by hand: the reducer cannot reach
// it for a tool call today, since only a model output end carries a cancelled
// status, but the row model offers it and the mapping must not guess.
func TestBuildToolStatuses(t *testing.T) {
	cases := []struct {
		messageStatus string
		want          ui.ToolStatus
	}{
		{"streaming", ui.ToolRunning},
		{"running", ui.ToolRunning},
		{"complete", ui.ToolComplete},
		{"completed", ui.ToolComplete},
		{"error", ui.ToolError},
		{"failed", ui.ToolError},
		{"cancelled", ui.ToolCancelled},
		{"", ui.ToolUnknown},
		{"something new", ui.ToolUnknown},
	}
	for _, tc := range cases {
		snapshot := projection.Snapshot{Messages: []projection.Message{{
			ID:     "c1",
			Role:   "tool",
			Status: tc.messageStatus,
			Events: []projection.Event{{
				Type:     "tool_call",
				Status:   tc.messageStatus,
				ToolName: "bash",
			}},
		}}}
		rows := Build(snapshot)
		if len(rows) != 1 {
			t.Fatalf("status %q: rows = %#v", tc.messageStatus, rows)
		}
		if got := rows[0].Tool.Status; got != tc.want {
			t.Errorf("status %q maps to %v, want %v", tc.messageStatus, got, tc.want)
		}
	}
}

// TestBuildToolNameFallback checks the name the row shows for a call the
// projection has no name for.
func TestBuildToolNameFallback(t *testing.T) {
	snapshot := projection.Snapshot{Messages: []projection.Message{{
		ID:     "c1",
		Role:   "tool",
		Status: "complete",
		Events: []projection.Event{{Type: "tool_call", Status: "complete"}},
	}}}
	rows := Build(snapshot)
	if got := rows[0].Tool.Name; got != "(unnamed tool)" {
		t.Fatalf("name = %q, want the row's fallback", got)
	}
	if rows[0].Tool.Params != "" || rows[0].Tool.Result != "" {
		t.Fatalf("an empty call invented previews: %#v", rows[0].Tool)
	}
}

// TestBuildSkipsUnknownRoles checks that a role the projection does not emit
// draws no row: an approval is a transient notice, and the adapter invents none.
func TestBuildSkipsUnknownRoles(t *testing.T) {
	snapshot := projection.Snapshot{Messages: []projection.Message{
		{ID: "x1", Role: "approval", Content: "Run it?"},
		{ID: "x2", Role: "", Content: "?"},
		{ID: "u1", Role: "user", Content: "kept"},
	}}
	rows := Build(snapshot)
	want := []ui.Message{{ID: "u1", Author: ui.AuthorUser, Content: "kept"}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want only the user row", rows)
	}
}

// TestBuildEmptySnapshot checks that an empty snapshot renders as nothing rather
// than nil, so a caller can range over it without a nil check.
func TestBuildEmptySnapshot(t *testing.T) {
	rows := Build(projection.Snapshot{})
	if rows == nil {
		t.Fatal("Build returned nil for an empty snapshot")
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %#v, want none", rows)
	}
}

func TestTransientMapping(t *testing.T) {
	snapshot := projection.TransientSnapshot{
		ModelProgress: &projection.ModelProgress{Text: "hello", Thinking: "plan", Tools: []projection.ModelProgressTool{{Name: "read_file"}}},
		Notices:       []projection.TransientNotice{{Kind: "session.notice", ID: "n1"}},
	}
	progress, notices := Transient(snapshot)
	if progress == nil || progress.Text != "hello" || progress.Thinking != "plan" || !reflect.DeepEqual(progress.Tools, []string{"read_file"}) {
		t.Fatalf("progress = %#v", progress)
	}
	if !reflect.DeepEqual(notices, []ui.Notice{{Kind: "session.notice", ID: "n1"}}) {
		t.Fatalf("notices = %#v", notices)
	}
	if rows := Build(projection.Snapshot{}); len(rows) != 0 {
		t.Fatalf("durable rows = %#v", rows)
	}
}

func TestRunStateFrom(t *testing.T) {
	cases := []struct {
		name    string
		status  string
		role    string
		want    ui.RunState
		comment string
	}{
		{"empty", "", "", ui.RunIdle, "nothing to run"},
		{"user sending", "sending", "user", ui.RunRunning, "submitted, not consumed"},
		{"assistant streaming", "streaming", "assistant", ui.RunRunning, "a reply still coming"},
		{"tool streaming", "streaming", "tool", ui.RunRunning, "a call still running"},
		{"all settled", "complete", "assistant", ui.RunIdle, "nothing left open"},
	}
	for _, tc := range cases {
		snapshot := projection.Snapshot{}
		if tc.status != "" {
			snapshot.Messages = []projection.Message{{ID: "m1", Role: tc.role, Status: tc.status}}
		}
		got := RunStateFrom(snapshot)
		if got != tc.want {
			t.Errorf("%s: run state %v, want %v (%s)", tc.name, got, tc.want, tc.comment)
		}
		if got == ui.RunStopping {
			t.Errorf("%s: the adapter invented a stopping run", tc.name)
		}
	}

	// A settled turn beside an older streaming row still reads as running: any
	// open message means the session has work on the wire.
	rows := snapshotOf(t,
		userInput("u1", 1, "hello"),
		inputApplied("e2", 2, "u1"),
		modelStart("e3", 3),
	)
	if got := RunStateFrom(rows); got != ui.RunRunning {
		t.Fatalf("run state after an unfinished turn = %v, want running", got)
	}
	if got := RunStateFrom(snapshotOf(t, userInput("u1", 1, "hi"), inputApplied("e2", 2, "u1"))); got != ui.RunIdle {
		t.Fatalf("run state after a consumed input = %v, want idle", got)
	}
}
