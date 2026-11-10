package host

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRecorderOnlyWritesWhitelistFields(t *testing.T) {
	diag, buf := newTestDiag(t)
	diag.Log(LevelInfo, EventStartupStage,
		StageField(StageConnect),
		FromStageField(StageIdle),
		NextStageField(StageSingleInstance),
		PhaseField(PhaseEndInteraction),
		OutcomeField(OutcomeOK),
		ClassField(ClassTimeout),
		ConnectedField(true),
		ExitingField(false),
		VersionField("0.1.0-dev"),
		ElapsedField(12*time.Millisecond),
		TimeoutField(3*time.Second),
		ArgsField(2),
		DroppedField(0),
		UpdateStateField(UpdateAvailable),
		UpdateVersionField("2.1.0"),
	)
	if err := diag.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	records := decodeRecords(t, buf.String())
	if len(records) != 1 {
		t.Fatalf("记录条数 = %d，期望 1", len(records))
	}
	rec := records[0]
	for key := range rec {
		if !recordAllowedKey(key) {
			t.Fatalf("出现白名单之外的字段 %q: %v", key, rec)
		}
	}
	for _, forbidden := range []string{"message", "body", "error", "credential", "content", "text"} {
		if _, ok := rec[forbidden]; ok {
			t.Fatalf("记录里出现不允许的字段 %q", forbidden)
		}
	}
	checks := map[string]any{
		"time":           nil,
		"level":          "info",
		"event":          "startup.stage",
		"stage":          "connect",
		"from_stage":     "idle",
		"next_stage":     "single_instance",
		"phase":          "end_interaction",
		"outcome":        "ok",
		"classification": "timeout",
		"connected":      true,
		"exiting":        false,
		"version":        "0.1.0-dev",
		"elapsed_ms":     float64(12),
		"timeout_ms":     float64(3000),
		"args":           float64(2),
		"dropped":        float64(0),
		"update_state":   "available",
		"update_version": "2.1.0",
	}
	for key, want := range checks {
		if want == nil {
			if _, ok := rec[key]; !ok {
				t.Fatalf("缺少字段 %q", key)
			}
			continue
		}
		if got := rec[key]; got != want {
			t.Fatalf("字段 %q = %v，期望 %v", key, got, want)
		}
	}
}

func TestRecorderRejectsCredentialsAndBodies(t *testing.T) {
	diag, buf := newTestDiag(t)
	secret := "sk-live-9f2c8b7a6d5e4f3a2b1c0d9e"
	body := "用户正文：我的密码是 hunter2"

	// 版本号只接受 x.y.z，凭据串会变成固定占位符。
	diag.Log(LevelError, EventStartupFailed,
		VersionField(secret),
		// 分类只接受枚举，凭据串收敛为 other。
		ClassField(ErrorClass(secret)),
		// 阶段只接受已知阶段名，任意串直接丢弃。
		StageField(Stage(secret)),
		// 业务正文没有可写入的字段：事件名形状不合规，只会记成固定事件名。
	)
	diag.Log(LevelInfo, Event(body))

	out := buf.String()
	if strings.Contains(out, secret) {
		t.Fatalf("凭据进入日志: %s", out)
	}
	if strings.Contains(out, body) {
		t.Fatalf("正文进入日志: %s", out)
	}
	if !strings.Contains(out, redacted) {
		t.Fatalf("版本号未按占位符遮蔽: %s", out)
	}
	if !strings.Contains(out, "event.invalid") {
		t.Fatalf("非法事件名未收敛为 event.invalid: %s", out)
	}
	if got := diag.Dropped(); got != 1 {
		t.Fatalf("丢弃字段数 = %d，期望 1", got)
	}

	records := decodeRecords(t, out)
	if len(records) != 2 {
		t.Fatalf("记录条数 = %d，期望 2", len(records))
	}
	if records[0]["classification"] != "other" {
		t.Fatalf("未知分类 = %v，期望 other", records[0]["classification"])
	}
	if records[0]["stage"] != nil {
		t.Fatalf("非法阶段应被丢弃，实际 %v", records[0]["stage"])
	}
	if dropped, ok := records[0]["dropped"]; !ok || dropped != float64(1) {
		t.Fatalf("丢弃计数 = %v，期望 1", dropped)
	}
}

func TestRecorderDropsZeroValueField(t *testing.T) {
	diag, buf := newTestDiag(t)
	diag.Log(LevelInfo, EventStartupStage, Field{}, StageField(StageConnect))
	if diag.Dropped() != 1 {
		t.Fatalf("丢弃字段数 = %d，期望 1", diag.Dropped())
	}
	records := decodeRecords(t, buf.String())
	if len(records) != 1 || records[0]["stage"] != "connect" {
		t.Fatalf("记录 = %v", records)
	}
	for key := range records[0] {
		if !recordAllowedKey(key) {
			t.Fatalf("出现白名单之外的字段 %q", key)
		}
	}
	if _, ok := records[0][""]; ok {
		t.Fatal("出现空字段名")
	}
}

func TestRecorderEveryDeclaredEventMatchesShape(t *testing.T) {
	events := []Event{
		EventInstancePrimary, EventInstanceForwarded, EventInstanceForwardFailed,
		EventInstanceStaleRemoved, EventActivationDropped, EventStartupStage,
		EventStartupFailed, EventStartupReady, EventStageRejected, EventShutdownBegin,
		EventShutdownPhase, EventShutdownTimeout, EventShutdownDone, EventConnectionChanged,
		EventUpdateCheck, EventUpdateDownload, EventUpdateSwitch, EventUpdateSwitchDeferred,
		EventUpdateTransitionRejected,
	}
	for _, event := range events {
		if !eventPattern.MatchString(string(event)) {
			t.Fatalf("事件名 %q 不符合形状约束", event)
		}
	}
}

func TestRecorderIsConcurrentAndLineAtomic(t *testing.T) {
	diag, buf := newTestDiag(t)
	const goroutines, perGoroutine = 8, 25
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				diag.Log(LevelInfo, EventStartupStage, StageField(StageConnect))
			}
		}()
	}
	wg.Wait()
	if err := diag.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	records := decodeRecords(t, buf.String())
	if len(records) != goroutines*perGoroutine {
		t.Fatalf("记录条数 = %d，期望 %d", len(records), goroutines*perGoroutine)
	}
}

func TestRecorderFlushAndWriteFailures(t *testing.T) {
	flusher := &flushCountingWriter{}
	diag := NewRecorder(flusher)
	diag.Log(LevelInfo, EventInstancePrimary)
	if err := diag.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if flusher.flushes != 1 || flusher.synced != 1 {
		t.Fatalf("Flush/Sync 调用次数 = %d/%d，期望 1/1", flusher.flushes, flusher.synced)
	}

	failing := NewRecorder(errWriter{})
	failing.Log(LevelInfo, EventInstancePrimary)
	if failing.WriteFailures() != 1 {
		t.Fatalf("写入失败计数 = %d，期望 1", failing.WriteFailures())
	}

	var nilRecorder *Recorder
	nilRecorder.Log(LevelInfo, EventInstancePrimary)
	if err := nilRecorder.Flush(); err != nil {
		t.Fatalf("nil Recorder Flush: %v", err)
	}
	if nilRecorder.Dropped() != 0 || nilRecorder.WriteFailures() != 0 {
		t.Fatal("nil Recorder 的计数应为 0")
	}
}

func TestVersionFieldValidatesSemver(t *testing.T) {
	diag, buf := newTestDiag(t)
	diag.Log(LevelInfo, EventStartupReady,
		VersionField("1.2.3-beta.1"),
		VersionField("v1.2.3"),
	)
	records := decodeRecords(t, buf.String())
	if records[0]["version"] != redacted {
		t.Fatalf("重复字段应以后写为准，实际 %v", records[0]["version"])
	}
}

type flushCountingWriter struct {
	flushes int
	synced  int
}

func (w *flushCountingWriter) Write(p []byte) (int, error) { return len(p), nil }

func (w *flushCountingWriter) Flush() error {
	w.flushes++
	return nil
}

func (w *flushCountingWriter) Sync() error {
	w.synced++
	return nil
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("写入失败") }
