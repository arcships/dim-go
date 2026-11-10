package host

import (
	"errors"
	"testing"
)

func newProbeStore(t *testing.T, connected bool) (*Store, *syncBuffer) {
	t.Helper()
	diag, buf := newTestDiag(t)
	probe := connected
	store := NewStore(StoreConfig{
		Diag:      diag,
		Connected: func() bool { return probe },
	})
	return store, buf
}

func TestStoreAdvancesStagesInOrder(t *testing.T) {
	store, buf := newProbeStore(t, true)
	if got := store.Stage(); got != StageIdle {
		t.Fatalf("初始阶段 = %s，期望 idle", got)
	}
	if err := store.Advance(StageConnect); !errors.Is(err, ErrInvalidStage) {
		t.Fatalf("跳跃推进应被拒绝，实际 %v", err)
	}
	if store.Stage() != StageIdle {
		t.Fatalf("被拒绝的推进改变了状态: %s", store.Stage())
	}
	for _, stage := range StartupStages() {
		if err := store.Advance(stage); err != nil {
			t.Fatalf("推进到 %s: %v", stage, err)
		}
	}
	if err := store.Advance(StageReady); err != nil {
		t.Fatalf("推进到 ready: %v", err)
	}
	if got := store.Snapshot().Stage; got != StageReady {
		t.Fatalf("最终阶段 = %s，期望 ready", got)
	}
	if err := store.Writable(); err != nil {
		t.Fatalf("就绪且已连接应可写: %v", err)
	}

	records := decodeRecords(t, buf.String())
	rejected := recordsOf(records, EventStageRejected)
	if len(rejected) != 1 {
		t.Fatalf("阶段拒绝记录条数 = %d，期望 1", len(rejected))
	}
	if rejected[0]["from_stage"] != "idle" || rejected[0]["next_stage"] != "connect" {
		t.Fatalf("拒绝记录字段 = %v", rejected[0])
	}
	if rejected[0]["outcome"] != "skipped" {
		t.Fatalf("拒绝记录结果 = %v，期望 skipped", rejected[0]["outcome"])
	}

	var stages []any
	for _, rec := range recordsOf(records, EventStartupStage) {
		stages = append(stages, rec["stage"])
	}
	want := []any{"single_instance", "local_state", "connect", "subscribe", "first_history", "allow_writes", "ready"}
	if len(stages) != len(want) {
		t.Fatalf("阶段记录 = %v，期望 %v", stages, want)
	}
	for i := range want {
		if stages[i] != want[i] {
			t.Fatalf("第 %d 个阶段 = %v，期望 %v", i+1, stages[i], want[i])
		}
	}
}

func TestStoreWritableStates(t *testing.T) {
	store, _ := newProbeStore(t, true)
	if err := store.Writable(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("未启动时应 ErrNotReady，实际 %v", err)
	}

	for _, stage := range StartupStages() {
		if err := store.Advance(stage); err != nil {
			t.Fatalf("推进 %s: %v", stage, err)
		}
	}
	if err := store.Writable(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("允许写之后仍需 ready，实际 %v", err)
	}
	if err := store.Advance(StageReady); err != nil {
		t.Fatalf("ready: %v", err)
	}

	disconnected, _ := newProbeStore(t, false)
	for _, stage := range StartupStages() {
		if err := disconnected.Advance(stage); err != nil {
			t.Fatalf("推进 %s: %v", stage, err)
		}
	}
	if err := disconnected.Advance(StageReady); err != nil {
		t.Fatalf("ready: %v", err)
	}
	if err := disconnected.Writable(); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("未连接时应 ErrNotConnected，实际 %v", err)
	}

	store.BeginExit()
	if err := store.Writable(); !errors.Is(err, ErrExiting) {
		t.Fatalf("退出中应 ErrExiting，实际 %v", err)
	}
	store.FinishExit()
	if state := store.Snapshot(); !state.Exiting || !state.Exited {
		t.Fatalf("退出快照 = %+v", state)
	}
	if err := store.Writable(); !errors.Is(err, ErrExiting) {
		t.Fatalf("退出完成后仍应拒绝新意图，实际 %v", err)
	}
}

func TestStoreFailStopsProgress(t *testing.T) {
	store, buf := newProbeStore(t, true)
	if err := store.Advance(StageSingleInstance); err != nil {
		t.Fatalf("推进: %v", err)
	}
	store.Fail(StageConnect, classifiedError{msg: "连不上", class: ClassRetryable})

	state := store.Snapshot()
	if state.Stage != StageConnect {
		t.Fatalf("失败后阶段 = %s，期望 connect", state.Stage)
	}
	if state.Failed == nil {
		t.Fatal("失败后应保留错误")
	}
	if err := store.Writable(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("失败后写闸门应拒绝，实际 %v", err)
	}
	if err := store.Advance(StageSubscribe); !errors.Is(err, ErrInvalidStage) {
		t.Fatalf("失败后不允许推进，实际 %v", err)
	}

	failed := recordsOf(decodeRecords(t, buf.String()), EventStartupFailed)
	if len(failed) != 1 {
		t.Fatalf("失败记录条数 = %d", len(failed))
	}
	if failed[0]["stage"] != "connect" || failed[0]["classification"] != "retryable" {
		t.Fatalf("失败记录 = %v", failed[0])
	}
}

func TestStoreObserveConnectionOnlyLogs(t *testing.T) {
	store, buf := newProbeStore(t, false)
	store.ObserveConnection(true, "")
	store.ObserveConnection(false, ClassConnectionMissing)

	records := recordsOf(decodeRecords(t, buf.String()), EventConnectionChanged)
	if len(records) != 2 {
		t.Fatalf("连接记录条数 = %d", len(records))
	}
	if records[0]["connected"] != true {
		t.Fatalf("第一条连接记录 = %v", records[0])
	}
	if records[1]["connected"] != false || records[1]["classification"] != "connection_missing" {
		t.Fatalf("第二条连接记录 = %v", records[1])
	}
	if store.Snapshot().Connected {
		t.Fatal("观察连接不应改变只读探针")
	}
}
