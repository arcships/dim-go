package host

import (
	"context"
	"errors"
	"testing"
)

func testSteps(order *[]string, failAt Stage, failure error) []StageStep {
	steps := make([]StageStep, 0, len(StartupStages()))
	for _, stage := range StartupStages() {
		stage := stage
		steps = append(steps, StageStep{
			Stage: stage,
			Run: func(context.Context) error {
				*order = append(*order, string(stage))
				if stage == failAt {
					return failure
				}
				return nil
			},
		})
	}
	return steps
}

func TestStartupRunsSixStagesInOrder(t *testing.T) {
	store, buf := newProbeStore(t, true)
	var order []string
	startup, err := NewStartup(testSteps(&order, "", nil), store)
	if err != nil {
		t.Fatalf("NewStartup: %v", err)
	}
	if err := startup.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []string{"single_instance", "local_state", "connect", "subscribe", "first_history", "allow_writes"}
	if len(order) != len(want) {
		t.Fatalf("执行顺序 = %v，期望 %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("第 %d 阶段 = %s，期望 %s", i+1, order[i], want[i])
		}
	}
	if store.Stage() != StageReady {
		t.Fatalf("最终阶段 = %s，期望 ready", store.Stage())
	}
	records := recordsOf(decodeRecords(t, buf.String()), EventStartupStage)
	if len(records) != 7 {
		t.Fatalf("阶段记录条数 = %d，期望 7（六阶段加 ready）", len(records))
	}
}

func TestStartupStopsAtFailingStage(t *testing.T) {
	store, buf := newProbeStore(t, true)
	var order []string
	failure := classifiedError{msg: "握手版本不兼容", class: ClassVersionIncompatible}
	startup, err := NewStartup(testSteps(&order, StageConnect, failure), store)
	if err != nil {
		t.Fatalf("NewStartup: %v", err)
	}
	err = startup.Run(context.Background())
	if !errors.Is(err, failure) {
		t.Fatalf("Run 错误 = %v，期望包含阶段错误", err)
	}
	if store.Stage() != StageConnect {
		t.Fatalf("失败后阶段 = %s，期望 connect", store.Stage())
	}
	if store.Writable(); !errors.Is(store.Writable(), ErrNotReady) {
		t.Fatalf("失败后写闸门应拒绝，实际 %v", store.Writable())
	}
	if len(order) != 3 {
		t.Fatalf("执行顺序 = %v，失败后不应继续", order)
	}
	failed := recordsOf(decodeRecords(t, buf.String()), EventStartupFailed)
	if len(failed) != 1 || failed[0]["stage"] != "connect" {
		t.Fatalf("失败记录 = %v", failed)
	}
	if failed[0]["classification"] != "version_incompatible" {
		t.Fatalf("失败分类 = %v", failed[0]["classification"])
	}
}

func TestNewStartupRequiresFullSuffix(t *testing.T) {
	store, _ := newProbeStore(t, true)
	var order []string
	steps := testSteps(&order, "", nil)

	if _, err := NewStartup(nil, nil); err == nil {
		t.Fatal("缺少状态归属应报错")
	}
	if _, err := NewStartup(nil, store); !errors.Is(err, ErrStageUnwired) {
		t.Fatalf("空阶段列表应报 ErrStageUnwired，实际 %v", err)
	}
	if _, err := NewStartup(steps[:len(steps)-1], store); !errors.Is(err, ErrStageUnwired) {
		t.Fatalf("缺最后一个阶段应报错，实际 %v", err)
	}
	swapped := append([]StageStep(nil), steps...)
	swapped[1], swapped[2] = swapped[2], swapped[1]
	if _, err := NewStartup(swapped, store); !errors.Is(err, ErrStageUnwired) {
		t.Fatalf("阶段顺序错误应报错，实际 %v", err)
	}
	unwired := append([]StageStep(nil), steps...)
	unwired[2].Run = nil
	if _, err := NewStartup(unwired, store); !errors.Is(err, ErrStageUnwired) {
		t.Fatalf("阶段未接线应报错，实际 %v", err)
	}

	ready, _ := newProbeStore(t, true)
	for _, stage := range StartupStages() {
		if err := ready.Advance(stage); err != nil {
			t.Fatalf("推进: %v", err)
		}
	}
	if err := ready.Advance(StageReady); err != nil {
		t.Fatalf("ready: %v", err)
	}
	if _, err := NewStartup(testSteps(&order, "", nil), ready); err == nil {
		t.Fatal("已就绪时不应再创建启动编排")
	}
}

func TestStartupRunsOnce(t *testing.T) {
	store, _ := newProbeStore(t, true)
	var order []string
	startup, err := NewStartup(testSteps(&order, "", nil), store)
	if err != nil {
		t.Fatalf("NewStartup: %v", err)
	}
	if err := startup.Run(context.Background()); err != nil {
		t.Fatalf("首次 Run: %v", err)
	}
	if err := startup.Run(context.Background()); err == nil {
		t.Fatal("重复 Run 应报错")
	}
	if len(order) != len(StartupStages()) {
		t.Fatalf("重复 Run 不应再次执行阶段: %v", order)
	}
}

func TestStartupStopsOnCanceledContext(t *testing.T) {
	store, _ := newProbeStore(t, true)
	var order []string
	startup, err := NewStartup(testSteps(&order, "", nil), store)
	if err != nil {
		t.Fatalf("NewStartup: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := startup.Run(ctx); err == nil {
		t.Fatal("已取消的上下文应让启动失败")
	}
	if len(order) != 0 {
		t.Fatalf("取消后不应执行任何阶段: %v", order)
	}
	if store.Stage() != StageSingleInstance {
		t.Fatalf("失败阶段 = %s，期望 single_instance", store.Stage())
	}
}
