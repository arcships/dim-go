package host

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// shutdownHarness 记录退出阶段执行顺序，并可注入挂起与失败。
type shutdownHarness struct {
	mu       sync.Mutex
	order    []string
	hang     map[ShutdownPhase]time.Duration
	fail     map[ShutdownPhase]error
	writable error
	flushed  bool
}

func newShutdownHarness() *shutdownHarness {
	return &shutdownHarness{
		hang: map[ShutdownPhase]time.Duration{},
		fail: map[ShutdownPhase]error{},
	}
}

func (h *shutdownHarness) record(phase ShutdownPhase) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.order = append(h.order, string(phase))
	if phase == PhaseFlushDiagnostics {
		h.flushed = true
	}
}

func (h *shutdownHarness) orderSnapshot() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.order...)
}

func (h *shutdownHarness) wasFlushed() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.flushed
}

func (h *shutdownHarness) writableSnapshot() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.writable
}

func (h *shutdownHarness) steps(store *Store) ShutdownSteps {
	run := func(phase ShutdownPhase) func(context.Context) error {
		return func(ctx context.Context) error {
			if phase == PhaseStopIntents {
				err := store.Writable()
				h.mu.Lock()
				h.writable = err
				h.mu.Unlock()
			}
			h.record(phase)
			if err, ok := h.fail[phase]; ok {
				return err
			}
			if d, ok := h.hang[phase]; ok {
				time.Sleep(d)
			}
			return nil
		}
	}
	return ShutdownSteps{
		StopIntents:      run(PhaseStopIntents),
		EndInteraction:   run(PhaseEndInteraction),
		CloseConnection:  run(PhaseCloseConnection),
		ReclaimService:   run(PhaseReclaimService),
		FlushDiagnostics: run(PhaseFlushDiagnostics),
	}
}

func TestShutdownRunsFivePhasesInOrder(t *testing.T) {
	store, buf := newProbeStore(t, true)
	for _, stage := range StartupStages() {
		if err := store.Advance(stage); err != nil {
			t.Fatalf("推进: %v", err)
		}
	}
	if err := store.Advance(StageReady); err != nil {
		t.Fatalf("ready: %v", err)
	}

	harness := newShutdownHarness()
	shutdown, err := NewShutdown(harness.steps(store), store, NewRecorder(buf), Timeouts{})
	if err != nil {
		t.Fatalf("NewShutdown: %v", err)
	}
	report, err := shutdown.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !report.Complete() {
		t.Fatalf("退出应完整完成: %+v", report)
	}
	if !harness.wasFlushed() {
		t.Fatal("冲刷诊断阶段未执行")
	}
	if !errors.Is(harness.writableSnapshot(), ErrExiting) {
		t.Fatalf("停止新意图阶段应看到退出标记，实际 %v", harness.writableSnapshot())
	}

	want := []string{"stop_intents", "end_interaction", "close_connection", "reclaim_service", "flush_diagnostics"}
	if got := harness.orderSnapshot(); len(got) != len(want) {
		t.Fatalf("退出顺序 = %v", got)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("第 %d 段 = %s，期望 %s", i+1, got[i], want[i])
			}
		}
	}

	state := store.Snapshot()
	if !state.Exiting || !state.Exited {
		t.Fatalf("退出后状态 = %+v", state)
	}
	if err := store.Writable(); !errors.Is(err, ErrExiting) {
		t.Fatalf("退出后仍应拒绝新意图，实际 %v", err)
	}

	records := decodeRecords(t, buf.String())
	if len(recordsOf(records, EventShutdownBegin)) != 1 {
		t.Fatal("缺少退出开始记录")
	}
	if len(recordsOf(records, EventShutdownDone)) != 1 {
		t.Fatal("缺少退出完成记录")
	}
	var phases []any
	for _, rec := range recordsOf(records, EventShutdownPhase) {
		phases = append(phases, rec["phase"])
	}
	for i := range want {
		if phases[i] != want[i] {
			t.Fatalf("阶段记录顺序 = %v", phases)
		}
	}
}

func TestShutdownTimeoutContinues(t *testing.T) {
	store, buf := newProbeStore(t, true)
	harness := newShutdownHarness()
	harness.hang[PhaseEndInteraction] = 400 * time.Millisecond
	timeouts := DefaultTimeouts()
	timeouts.EndInteraction = 20 * time.Millisecond

	shutdown, err := NewShutdown(harness.steps(store), store, NewRecorder(buf), timeouts)
	if err != nil {
		t.Fatalf("NewShutdown: %v", err)
	}
	start := time.Now()
	report, err := shutdown.Run(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("超时应体现在报告错误里")
	}
	if !errors.Is(err, ErrPhaseTimeout) {
		t.Fatalf("错误 = %v，期望包含 ErrPhaseTimeout", err)
	}
	if report.Complete() {
		t.Fatal("超时的退出不应算完整")
	}
	timedOut := report.TimedOut()
	if len(timedOut) != 1 || timedOut[0] != PhaseEndInteraction {
		t.Fatalf("超时阶段 = %v，期望 [end_interaction]", timedOut)
	}
	if got := harness.orderSnapshot(); len(got) != 5 {
		t.Fatalf("后续阶段应在超时后继续执行: %v", got)
	}
	if elapsed > 300*time.Millisecond {
		t.Fatalf("退出等待过久: %v", elapsed)
	}
	if !harness.wasFlushed() {
		t.Fatal("冲刷诊断应在超时后继续执行")
	}

	records := recordsOf(decodeRecords(t, buf.String()), EventShutdownTimeout)
	if len(records) != 1 || records[0]["phase"] != "end_interaction" {
		t.Fatalf("超时记录 = %v", records)
	}
	if records[0]["timeout_ms"] != float64(20) {
		t.Fatalf("超时毫秒 = %v，期望 20", records[0]["timeout_ms"])
	}
}

func TestShutdownFailureContinues(t *testing.T) {
	store, buf := newProbeStore(t, true)
	harness := newShutdownHarness()
	harness.fail[PhaseCloseConnection] = errors.New("连接关闭失败")
	shutdown, err := NewShutdown(harness.steps(store), store, NewRecorder(buf), Timeouts{})
	if err != nil {
		t.Fatalf("NewShutdown: %v", err)
	}
	report, err := shutdown.Run(context.Background())
	if err == nil {
		t.Fatal("失败应体现在报告错误里")
	}
	failed := report.Failed()
	if len(failed) != 1 || failed[0] != PhaseCloseConnection {
		t.Fatalf("失败阶段 = %v", failed)
	}
	if got := harness.orderSnapshot(); len(got) != 5 {
		t.Fatalf("后续阶段应继续执行: %v", got)
	}
	for _, rec := range recordsOf(decodeRecords(t, buf.String()), EventShutdownPhase) {
		if rec["phase"] == "close_connection" && rec["outcome"] != "failed" {
			t.Fatalf("关闭连接记录 = %v", rec)
		}
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	store, _ := newProbeStore(t, true)
	harness := newShutdownHarness()
	shutdown, err := NewShutdown(harness.steps(store), store, nil, Timeouts{})
	if err != nil {
		t.Fatalf("NewShutdown: %v", err)
	}
	first, firstErr := shutdown.Run(context.Background())
	second, secondErr := shutdown.Run(context.Background())
	if firstErr != nil || secondErr != nil {
		t.Fatalf("Run 错误: %v / %v", firstErr, secondErr)
	}
	if got := harness.orderSnapshot(); len(got) != 5 {
		t.Fatalf("重复退出不应重跑阶段: %v", got)
	}
	if len(first.Results) != len(second.Results) {
		t.Fatalf("两次报告不一致: %+v / %+v", first, second)
	}
}

func TestNewShutdownRequiresSteps(t *testing.T) {
	store, _ := newProbeStore(t, true)
	if _, err := NewShutdown(ShutdownSteps{}, nil, nil, Timeouts{}); err == nil {
		t.Fatal("缺少状态归属应报错")
	}
	if _, err := NewShutdown(ShutdownSteps{}, store, nil, Timeouts{}); !errors.Is(err, ErrPhaseUnwired) {
		t.Fatalf("空阶段应报 ErrPhaseUnwired，实际 %v", err)
	}
	harness := newShutdownHarness()
	steps := harness.steps(store)
	steps.ReclaimService = nil
	if _, err := NewShutdown(steps, store, nil, Timeouts{}); !errors.Is(err, ErrPhaseUnwired) {
		t.Fatalf("缺回收阶段应报错，实际 %v", err)
	}
}

func TestShutdownUsesDefaultFlushWhenUnwired(t *testing.T) {
	store, buf := newProbeStore(t, true)
	harness := newShutdownHarness()
	steps := harness.steps(store)
	steps.FlushDiagnostics = nil
	shutdown, err := NewShutdown(steps, store, NewRecorder(buf), Timeouts{})
	if err != nil {
		t.Fatalf("NewShutdown: %v", err)
	}
	if _, err := shutdown.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if harness.wasFlushed() {
		t.Fatal("未接线的冲刷阶段不应调用自定义实现")
	}
	if len(recordsOf(decodeRecords(t, buf.String()), EventShutdownDone)) != 1 {
		t.Fatal("默认冲刷后应留下退出完成记录")
	}
}

func TestShutdownSurvivesCanceledParent(t *testing.T) {
	store, _ := newProbeStore(t, true)
	harness := newShutdownHarness()
	shutdown, err := NewShutdown(harness.steps(store), store, nil, Timeouts{})
	if err != nil {
		t.Fatalf("NewShutdown: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report, _ := shutdown.Run(ctx)
	if len(report.Results) != 5 {
		t.Fatalf("父上下文取消后仍应走完五段，实际 %+v", report)
	}
}
