package projection

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fixtureSchemaVersion is the only golden fixture schema this runner accepts.
const fixtureSchemaVersion = 1

type fixtureClock struct {
	NowMillis int64 `json:"nowMillis"`
}

type fixtureInput struct {
	History   []RawEvent      `json:"history"`
	Live      []RawEvent      `json:"live"`
	Transient []fixtureNotice `json:"transient"`
}

type fixtureNotice struct {
	Kind     string         `json:"kind"`
	ID       string         `json:"id"`
	Sequence int64          `json:"sequence,omitempty"`
	Payload  map[string]any `json:"payload,omitempty"`
}

type fixtureExpect struct {
	Messages               []Message         `json:"messages"`
	TimelineUserMessageIds []string          `json:"timelineUserMessageIds"`
	Transient              []TransientNotice `json:"transient"`
}

type goldenFixture struct {
	SchemaVersion int           `json:"schemaVersion"`
	Name          string        `json:"name"`
	SessionID     string        `json:"sessionId"`
	BranchID      string        `json:"branchId"`
	Clock         *fixtureClock `json:"clock"`
	Input         fixtureInput  `json:"input"`
	Expect        fixtureExpect `json:"expect"`
}

func (fx goldenFixture) validate() error {
	if fx.SchemaVersion != fixtureSchemaVersion {
		return fmt.Errorf("schemaVersion = %d, want %d", fx.SchemaVersion, fixtureSchemaVersion)
	}
	if fx.Name == "" {
		return errors.New("name is required")
	}
	if fx.SessionID == "" {
		return errors.New("sessionId is required")
	}
	if fx.BranchID == "" {
		return errors.New("branchId is required")
	}
	if fx.Clock == nil {
		return errors.New("clock is required")
	}
	if fx.Input.History == nil || fx.Input.Live == nil || fx.Input.Transient == nil {
		return errors.New("input.history, input.live and input.transient must be explicit arrays")
	}
	if fx.Expect.Messages == nil {
		return errors.New("expect.messages must be an explicit array")
	}
	if fx.Expect.TimelineUserMessageIds == nil {
		return errors.New("expect.timelineUserMessageIds must be an explicit array")
	}
	if fx.Expect.Transient == nil {
		return errors.New("expect.transient must be an explicit array")
	}

	// Exact replays of one sequence are the overlap case. A sequence with two
	// different events, or an id reused at another sequence, is an invalid
	// fixture because the reducer dedupes by sequence and keys by id.
	idsBySequence := map[int64]string{}
	sequencesByID := map[string]int64{}
	for _, group := range [][]RawEvent{fx.Input.History, fx.Input.Live} {
		for _, event := range group {
			if err := validateRawEvent(event); err != nil {
				return err
			}
			if id, ok := idsBySequence[event.Sequence]; ok && id != event.ID {
				return fmt.Errorf("sequence %d is used by %q and %q", event.Sequence, id, event.ID)
			}
			idsBySequence[event.Sequence] = event.ID
			if sequence, ok := sequencesByID[event.ID]; ok && sequence != event.Sequence {
				return fmt.Errorf("event %q has sequences %d and %d", event.ID, sequence, event.Sequence)
			}
			sequencesByID[event.ID] = event.Sequence
		}
	}
	for _, notice := range fx.Input.Transient {
		if notice.Kind == "" || notice.ID == "" {
			return errors.New("input.transient entries require kind and id")
		}
	}
	return nil
}

func validateRawEvent(event RawEvent) error {
	switch {
	case event.ID == "":
		return errors.New("raw event id is required")
	case event.Sequence <= 0:
		return fmt.Errorf("raw event %q: sequence must be positive", event.ID)
	case event.CreateAt <= 0:
		return fmt.Errorf("raw event %q: create_at must be positive", event.ID)
	case event.Type == "":
		return fmt.Errorf("raw event %q: type is required", event.ID)
	case event.Name == "":
		return fmt.Errorf("raw event %q: name is required", event.ID)
	}
	return nil
}

// TestGoldenFixtures is the fixture matrix from docs/projection-contract.md
// section 7. Every failure names both the fixture and the file.
func TestGoldenFixtures(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("testdata", "projection"))
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
	names := map[string]string{}
	loaded := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join("testdata", "projection", entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		var fixture goldenFixture
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&fixture); err != nil {
			t.Fatalf("%s: decode: %v", path, err)
		}
		if err := fixture.validate(); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if previous, ok := names[fixture.Name]; ok {
			t.Fatalf("%s: duplicate fixture name %q also used by %s", path, fixture.Name, previous)
		}
		names[fixture.Name] = path
		loaded++
		t.Run(fixture.Name, func(t *testing.T) {
			runGoldenFixture(t, path, fixture)
		})
	}
	if loaded == 0 {
		t.Fatal("no projection fixtures found in testdata/projection")
	}
}

func runGoldenFixture(t *testing.T, path string, fixture goldenFixture) {
	t.Helper()
	clock := FixedClock(fixture.Clock.NowMillis)
	fail := func(format string, args ...any) {
		t.Helper()
		t.Fatalf("fixture %q (%s): %s", fixture.Name, path, fmt.Sprintf(format, args...))
	}

	snapshot := foldHistoryThenLive(t, fixture, clock)

	// Durable output must be the declared snapshot: no transient leakage, no
	// extra messages, and only the contract's minimal fields.
	assertJSONEqual(t, "expect.messages", fixture.Expect.Messages, snapshot.Messages, fail)
	assertJSONEqual(t, "expect.timelineUserMessageIds", fixture.Expect.TimelineUserMessageIds, snapshot.TimelineUserMessageIds, fail)
	for _, message := range snapshot.Messages {
		if message.Role == "assistant" && !strings.HasPrefix(message.ID, "a:"+fixture.BranchID+":") {
			fail("assistant %q is not scoped to branch %q", message.ID, fixture.BranchID)
		}
	}

	// Same fixed clock, fresh reducer: byte-identical snapshot.
	assertJSONEqual(t, "deterministic rerun", snapshot, foldHistoryThenLive(t, fixture, clock), fail)
	// Different injected clock: durable snapshot must not change, proving the
	// reducer used create_at facts rather than local time.
	assertJSONEqual(t, "clock independence", snapshot, foldHistoryThenLive(t, fixture, FixedClock(fixture.Clock.NowMillis+1000)), fail)

	// Sequence order (the history paging order) must converge to the same
	// durable snapshot as history-arrival + live-arrival.
	union := append(append([]RawEvent{}, fixture.Input.History...), fixture.Input.Live...)
	canonical := NewReducer(fixture.BranchID, clock)
	if err := canonical.ApplyHistory(union); err != nil {
		fail("canonical history: %v", err)
	}
	assertJSONEqual(t, "history/live equivalence", snapshot, canonical.Snapshot(), fail)

	// Attach overlap: live notices are replayed before history pages. The same
	// facts in the opposite application order must converge on the same durable
	// snapshot, including blocks/facts that arrived before the live suffix.
	liveFirst := NewReducer(fixture.BranchID, clock)
	for i, event := range fixture.Input.Live {
		if err := liveFirst.Apply(event); err != nil {
			fail("live[%d] %q: %v", i, event.ID, err)
		}
	}
	if err := liveFirst.ApplyHistory(fixture.Input.History); err != nil {
		fail("live/history equivalence: %v", err)
	}
	assertJSONEqual(t, "live/history equivalence", snapshot, liveFirst.Snapshot(), fail)

	// Transient projections are independent and clock-stamped.
	transient := foldTransient(t, fixture, clock)
	assertJSONEqual(t, "expect.transient", fixture.Expect.Transient, transient, fail)
	assertJSONEqual(t, "transient deterministic rerun", transient, foldTransient(t, fixture, clock), fail)
	shifted := foldTransient(t, fixture, FixedClock(fixture.Clock.NowMillis+1000))
	for i := range shifted {
		if shifted[i].ReceivedAt != transient[i].ReceivedAt+1000 {
			fail("transient notice %q receivedAt = %d, want injected clock %d", shifted[i].ID, shifted[i].ReceivedAt, transient[i].ReceivedAt+1000)
		}
		shifted[i].ReceivedAt = transient[i].ReceivedAt
	}
	assertJSONEqual(t, "transient clock independence", transient, shifted, fail)
}

func foldHistoryThenLive(t *testing.T, fixture goldenFixture, clock Clock) Snapshot {
	t.Helper()
	reducer := NewReducer(fixture.BranchID, clock)
	if err := reducer.ApplyHistory(fixture.Input.History); err != nil {
		t.Fatalf("fixture %q: history: %v", fixture.Name, err)
	}
	for i, event := range fixture.Input.Live {
		if err := reducer.Apply(event); err != nil {
			t.Fatalf("fixture %q: live[%d] %q: %v", fixture.Name, i, event.ID, err)
		}
	}
	return reducer.Snapshot()
}

func foldTransient(t *testing.T, fixture goldenFixture, clock Clock) []TransientNotice {
	t.Helper()
	reducer := NewTransientReducer(fixture.SessionID, fixture.BranchID, clock)
	for i, notice := range fixture.Input.Transient {
		if err := reducer.ApplyNotice(notice.Kind, notice.ID, notice.Sequence, notice.Payload); err != nil {
			t.Fatalf("fixture %q: transient[%d] %q: %v", fixture.Name, i, notice.ID, err)
		}
	}
	return reducer.Snapshot().Notices
}

func assertJSONEqual(t *testing.T, label string, want, got any, fail func(string, ...any)) {
	t.Helper()
	if reflect.DeepEqual(want, got) {
		return
	}
	fail("%s mismatch\nwant:\n%s\ngot:\n%s", label, indentJSON(want), indentJSON(got))
}

func indentJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%#v", value)
	}
	return string(data)
}
