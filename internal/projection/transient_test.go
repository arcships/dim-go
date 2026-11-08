package projection

import (
	"reflect"
	"testing"
)

func testProgress() ModelProgress {
	return ModelProgress{
		SessionID:      "s-1",
		BranchID:       "main",
		TurnID:         "t-1",
		RequestID:      "r-1",
		TextPartID:     "e-text",
		ThinkingPartID: "e-thinking",
		Text:           "partial",
		TextOffset:     3,
		Thinking:       "reasoning",
		ThinkingOffset: 0,
		Tools:          []ModelProgressTool{{CallID: "c1", Name: "read_file", Arguments: `{"path":"x"}`}},
	}
}

func TestModelProgressDoesNotTouchTimeline(t *testing.T) {
	timeline := NewReducer("main", FixedClock(11))
	events := []RawEvent{
		testEvent("e1", 1, "step", "model-start", map[string]any{"turn_id": "t-1", "request_id": "r-1"}),
		testEvent("e2", 2, "step", "thinking", map[string]any{"delta": "reasoning"}),
		testEvent("e3", 3, "step", "text", map[string]any{"delta": "partial"}),
	}
	if err := timeline.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	before := timeline.Snapshot()

	transient := NewTransientReducer("s-1", "main", FixedClock(1700000000999))
	if err := transient.ApplyModelProgress(7, testProgress()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, timeline.Snapshot()) {
		t.Fatal("model progress mutated the durable timeline")
	}

	after := transient.Snapshot()
	if err := timeline.Apply(testEvent("e4", 4, "step", "text", map[string]any{"delta": "!"})); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, transient.Snapshot()) {
		t.Fatal("timeline event mutated transient projection")
	}
	if reflect.DeepEqual(before, timeline.Snapshot()) {
		t.Fatal("timeline event was not applied")
	}
}

func TestModelProgressReplacementAndStaleSequence(t *testing.T) {
	transient := NewTransientReducer("s-1", "main", FixedClock(1700000000999))
	first := testProgress()
	if err := transient.ApplyModelProgress(5, first); err != nil {
		t.Fatal(err)
	}
	second := first
	second.Text = "partial more"
	second.TextOffset = 12
	if err := transient.ApplyModelProgress(6, second); err != nil {
		t.Fatal(err)
	}
	stale := first
	stale.Text = "stale"
	if err := transient.ApplyModelProgress(4, stale); err != nil {
		t.Fatal(err)
	}
	snapshot := transient.Snapshot()
	if snapshot.ModelProgress == nil || snapshot.ModelProgress.Text != "partial more" || snapshot.ModelProgress.TextOffset != 12 {
		t.Fatalf("snapshot=%+v", snapshot.ModelProgress)
	}
	// Snapshots are copies, not aliases of reducer state.
	snapshot.ModelProgress.Tools[0].Name = "mutated"
	if transient.Snapshot().ModelProgress.Tools[0].Name != "read_file" {
		t.Fatal("snapshot aliases transient state")
	}
}

func TestModelProgressEmptyClearsAndMismatchFails(t *testing.T) {
	transient := NewTransientReducer("s-1", "main", FixedClock(1700000000999))
	if err := transient.ApplyModelProgress(1, testProgress()); err != nil {
		t.Fatal(err)
	}
	if err := transient.ApplyModelProgress(2, ModelProgress{SessionID: "s-1", BranchID: "main", Tools: []ModelProgressTool{}}); err != nil {
		t.Fatal(err)
	}
	if progress := transient.Snapshot().ModelProgress; progress != nil {
		t.Fatalf("empty progress did not clear: %+v", progress)
	}
	if err := transient.ApplyModelProgress(3, ModelProgress{SessionID: "other", BranchID: "main"}); err == nil {
		t.Fatal("expected session mismatch error")
	}
	if err := transient.ApplyModelProgress(3, ModelProgress{SessionID: "s-1", BranchID: "other"}); err == nil {
		t.Fatal("expected branch mismatch error")
	}
}

func TestProjectionIsClockDeterministic(t *testing.T) {
	events := []RawEvent{
		testEvent("e1", 1, "step", "model-start", map[string]any{"turn_id": "t", "request_id": "r"}),
		testEvent("e2", 2, "step", "tool-use", map[string]any{
			"call": map[string]any{"id": "c1", "name": "read_file", "input": map[string]any{"path": "x"}},
		}),
		testEvent("e3", 3, "step", "tool-completed", map[string]any{
			"call_id": "c1", "result": map[string]any{"content": "ok"},
		}),
		testEvent("e4", 4, "step", "model-output-end", map[string]any{"status": "completed"}),
	}
	early := NewReducer("main", FixedClock(1))
	late := NewReducer("main", FixedClock(1700000000999))
	if err := early.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	if err := late.ApplyHistory(events); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(early.Snapshot(), late.Snapshot()) {
		t.Fatal("clock leaked into durable snapshot")
	}

	first := NewTransientReducer("s-1", "main", FixedClock(1700000000999))
	second := NewTransientReducer("s-1", "main", FixedClock(1700000000999))
	if err := first.ApplyModelProgress(1, testProgress()); err != nil {
		t.Fatal(err)
	}
	if err := second.ApplyModelProgress(1, testProgress()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Snapshot(), second.Snapshot()) {
		t.Fatal("transient snapshot is not deterministic")
	}
}

func TestTransientNoticesAreIsolatedReplayableAndClockStamped(t *testing.T) {
	transient := NewTransientReducer("s-1", "main", FixedClock(1700000000999))
	if err := transient.ApplyNotice("approval", "n-1", 10, map[string]any{"status": "pending"}); err != nil {
		t.Fatal(err)
	}
	if err := transient.ApplyNotice("question", "n-2", 11, map[string]any{"question": "continue?"}); err != nil {
		t.Fatal(err)
	}
	// Replaying an id replaces in place instead of appending a second notice.
	if err := transient.ApplyNotice("approval", "n-1", 12, map[string]any{"status": "approved"}); err != nil {
		t.Fatal(err)
	}
	// A stale sequence for an existing notice is ignored.
	if err := transient.ApplyNotice("approval", "n-1", 9, map[string]any{"status": "stale"}); err != nil {
		t.Fatal(err)
	}
	notices := transient.Snapshot().Notices
	if len(notices) != 2 {
		t.Fatalf("notices=%+v", notices)
	}
	if notices[0].ID != "n-1" || notices[0].Sequence != 12 {
		t.Fatalf("replayed notice=%+v", notices[0])
	}
	if got := notices[0].Payload["status"]; got != "approved" {
		t.Fatalf("payload=%+v", notices[0].Payload)
	}
	if notices[0].ReceivedAt != 1700000000999 || notices[1].ReceivedAt != 1700000000999 {
		t.Fatalf("clock not injected: %+v", notices)
	}
	// Snapshots are copies, not aliases of reducer state.
	notices[0].Payload["status"] = "mutated"
	if transient.Snapshot().Notices[0].Payload["status"] != "approved" {
		t.Fatal("snapshot aliases transient notice payload")
	}
	if err := transient.ApplyNotice("", "n-3", 1, nil); err == nil {
		t.Fatal("expected missing kind error")
	}
	if err := transient.ApplyNotice("approval", "", 1, nil); err == nil {
		t.Fatal("expected missing id error")
	}
}
