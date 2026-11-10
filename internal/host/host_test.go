package host

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func wiredStartup(order *[]string, failAt Stage, failure error) StartupSteps {
	step := func(stage Stage) func(context.Context) error {
		return func(context.Context) error {
			*order = append(*order, string(stage))
			if stage == failAt {
				return failure
			}
			return nil
		}
	}
	return StartupSteps{
		LocalState:   step(StageLocalState),
		Connect:      step(StageConnect),
		Subscribe:    step(StageSubscribe),
		FirstHistory: step(StageFirstHistory),
		AllowWrites:  step(StageAllowWrites),
	}
}

func TestShellLifecycle(t *testing.T) {
	endpoint := shortEndpoint(t)
	diag, buf := newTestDiag(t)
	connected := false
	var order []string
	var writableBeforeReady error
	var writableWhileExiting error
	var shell *Shell

	var steps StartupSteps
	steps = wiredStartup(&order, "", nil)
	// 连接阶段之前不允许发服务端请求，首轮历史阶段仍在闸门之内。
	steps.Connect = func(context.Context) error {
		order = append(order, string(StageConnect))
		connected = true
		return nil
	}
	steps.FirstHistory = func(context.Context) error {
		order = append(order, string(StageFirstHistory))
		if shell != nil {
			writableBeforeReady = shell.Writable()
		}
		return nil
	}

	cfg := Config{
		Endpoint: endpoint,
		Version:  "0.1.0-dev",
		Diag:     diag,
		Connected: func() bool {
			return connected
		},
		Startup: steps,
		Shutdown: ShutdownSteps{
			StopIntents: func(context.Context) error {
				if shell != nil {
					writableWhileExiting = shell.Writable()
				}
				return nil
			},
			EndInteraction:   func(context.Context) error { time.Sleep(300 * time.Millisecond); return nil },
			CloseConnection:  func(context.Context) error { return nil },
			ReclaimService:   func(context.Context) error { return nil },
			FlushDiagnostics: func(context.Context) error { return diag.Flush() },
		},
		Timeouts: Timeouts{EndInteraction: 20 * time.Millisecond},
	}

	var err error
	shell, err = New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if state := shell.State(); state.Stage != StageIdle || state.Connected {
		t.Fatalf("启动前状态 = %+v", state)
	}
	if err := shell.Writable(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("启动前写闸门 = %v，期望 ErrNotReady", err)
	}

	outcome, err := shell.Start(context.Background())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if outcome != StartPrimary {
		t.Fatalf("启动结局 = %v，期望 primary", outcome)
	}
	wantOrder := []string{"local_state", "connect", "subscribe", "first_history", "allow_writes"}
	if len(order) != len(wantOrder) {
		t.Fatalf("启动顺序 = %v", order)
	}
	for i := range wantOrder {
		if order[i] != wantOrder[i] {
			t.Fatalf("第 %d 阶段 = %s，期望 %s", i+1, order[i], wantOrder[i])
		}
	}
	state := shell.State()
	if state.Stage != StageReady || state.Failed != nil || !state.Connected {
		t.Fatalf("启动后状态 = %+v", state)
	}
	if err := shell.Writable(); err != nil {
		t.Fatalf("就绪后应可写: %v", err)
	}
	if !errors.Is(writableBeforeReady, ErrNotReady) {
		t.Fatalf("首轮历史阶段写闸门 = %v，期望 ErrNotReady", writableBeforeReady)
	}

	// 断线后不当空数据：写闸门拒绝。
	connected = false
	if err := shell.Writable(); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("断线后写闸门 = %v，期望 ErrNotConnected", err)
	}
	shell.ObserveConnection(false, ClassConnectionMissing)
	connected = true

	// 二次启动：同一端点、带参数。本进程应只转交参数并退出。
	secret := "sk-live-cafebabe"
	secondCfg := cfg
	secondCfg.Args = []string{"--open", secret}
	secondCfg.WorkingDir = "/tmp/work"
	second, err := New(secondCfg)
	if err != nil {
		t.Fatalf("New 第二实例: %v", err)
	}
	secondOutcome, err := second.Start(context.Background())
	if err != nil {
		t.Fatalf("二次启动: %v", err)
	}
	if secondOutcome != StartForwarded {
		t.Fatalf("二次启动结局 = %v，期望 forwarded", secondOutcome)
	}
	if state := second.State(); state.Stage != StageIdle {
		t.Fatalf("转交进程不应推进启动阶段，实际 %s", state.Stage)
	}
	activation := waitActivation(t, shell.Activations())
	if len(activation.Args) != 2 || activation.Args[1] != secret {
		t.Fatalf("转交参数 = %v", activation.Args)
	}
	if activation.WorkingDir != "/tmp/work" {
		t.Fatalf("转交工作目录 = %q", activation.WorkingDir)
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("启动参数内容进入诊断: %s", buf.String())
	}

	// 退出：结束交互段超时，其余四段必须继续走完。
	start := time.Now()
	report, err := shell.Shutdown(context.Background())
	elapsed := time.Since(start)
	if err == nil || !errors.Is(err, ErrPhaseTimeout) {
		t.Fatalf("退出错误 = %v，期望 ErrPhaseTimeout", err)
	}
	if timedOut := report.TimedOut(); len(timedOut) != 1 || timedOut[0] != PhaseEndInteraction {
		t.Fatalf("超时阶段 = %v", timedOut)
	}
	if len(report.Results) != 5 {
		t.Fatalf("退出阶段数 = %d，期望 5", len(report.Results))
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("退出耗时 = %v，超时段不应拖住后续", elapsed)
	}
	if !errors.Is(writableWhileExiting, ErrExiting) {
		t.Fatalf("退出中写闸门 = %v，期望 ErrExiting", writableWhileExiting)
	}
	if state := shell.State(); !state.Exiting || !state.Exited {
		t.Fatalf("退出后状态 = %+v", state)
	}
	if _, err := os.Stat(endpoint); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("退出后实例端点仍存在: %v", err)
	}

	records := decodeRecords(t, buf.String())
	if len(recordsOf(records, EventInstancePrimary)) != 1 {
		t.Fatal("缺少主实例记录")
	}
	if len(recordsOf(records, EventInstanceForwarded)) != 1 {
		t.Fatal("缺少接管记录")
	}
	ready := recordsOf(records, EventStartupReady)
	if len(ready) != 1 || ready[0]["version"] != "0.1.0-dev" {
		t.Fatalf("就绪记录 = %v", ready)
	}
	if len(recordsOf(records, EventConnectionChanged)) != 1 {
		t.Fatal("缺少连接状态记录")
	}

	// 诊断顺序：启动事件全部早于退出事件；退出五段按序。
	firstShutdown := -1
	lastStartup := -1
	for i, rec := range records {
		switch rec["event"] {
		case string(EventStartupStage), string(EventStartupReady):
			lastStartup = i
		case string(EventShutdownBegin):
			if firstShutdown == -1 {
				firstShutdown = i
			}
		}
	}
	if firstShutdown == -1 || lastStartup == -1 || lastStartup > firstShutdown {
		t.Fatalf("启动与退出顺序不对: lastStartup=%d firstShutdown=%d", lastStartup, firstShutdown)
	}
	var phases []any
	for _, rec := range recordsOf(records, EventShutdownPhase) {
		phases = append(phases, rec["phase"])
	}
	// 结束交互段超时，只会留下 shutdown.timeout 记录，其余四段的完成记录保持顺序。
	wantPhases := []any{"stop_intents", "close_connection", "reclaim_service", "flush_diagnostics"}
	if len(phases) != len(wantPhases) {
		t.Fatalf("退出阶段记录 = %v", phases)
	}
	for i := range wantPhases {
		if phases[i] != wantPhases[i] {
			t.Fatalf("退出阶段顺序 = %v", phases)
		}
	}
	if len(recordsOf(records, EventShutdownTimeout)) != 1 {
		t.Fatal("缺少退出超时记录")
	}
	if len(recordsOf(records, EventShutdownDone)) != 1 {
		t.Fatal("缺少退出完成记录")
	}
}

func TestShellStartStopsAtFailingStage(t *testing.T) {
	endpoint := shortEndpoint(t)
	diag, buf := newTestDiag(t)
	var order []string
	failure := classifiedError{msg: "服务端版本不兼容", class: ClassVersionIncompatible}
	cfg := Config{
		Endpoint: endpoint,
		Diag:     diag,
		Startup:  wiredStartup(&order, StageConnect, failure),
		Shutdown: ShutdownSteps{
			StopIntents:     func(context.Context) error { return nil },
			EndInteraction:  func(context.Context) error { return nil },
			CloseConnection: func(context.Context) error { return nil },
			ReclaimService:  func(context.Context) error { return nil },
		},
	}
	shell, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer shell.Close()

	if _, err := shell.Start(context.Background()); !errors.Is(err, failure) {
		t.Fatalf("Start 错误 = %v，期望包含阶段错误", err)
	}
	state := shell.State()
	if state.Stage != StageConnect {
		t.Fatalf("失败后阶段 = %s，期望 connect", state.Stage)
	}
	if state.Failed == nil {
		t.Fatal("失败后应保留错误")
	}
	if err := shell.Writable(); !errors.Is(err, ErrNotReady) {
		t.Fatalf("失败后写闸门 = %v，期望 ErrNotReady", err)
	}
	if len(order) != 2 {
		t.Fatalf("失败后不应继续推进: %v", order)
	}
	failed := recordsOf(decodeRecords(t, buf.String()), EventStartupFailed)
	if len(failed) != 1 || failed[0]["classification"] != "version_incompatible" {
		t.Fatalf("失败记录 = %v", failed)
	}
}

func TestShellNewRequiresConfiguration(t *testing.T) {
	var order []string
	steps := wiredStartup(&order, "", nil)
	if _, err := New(Config{Startup: steps}); err == nil {
		t.Fatal("缺少端点与数据目录应报错")
	}
	if _, err := New(Config{Endpoint: "/tmp/x", Startup: StartupSteps{}}); !errors.Is(err, ErrStageUnwired) {
		t.Fatalf("缺少阶段实现应报 ErrStageUnwired，实际 %v", err)
	}
}

func TestShellStartOnlyOnce(t *testing.T) {
	endpoint := shortEndpoint(t)
	var order []string
	cfg := Config{
		Endpoint: endpoint,
		Startup:  wiredStartup(&order, "", nil),
		Shutdown: ShutdownSteps{
			StopIntents:     func(context.Context) error { return nil },
			EndInteraction:  func(context.Context) error { return nil },
			CloseConnection: func(context.Context) error { return nil },
			ReclaimService:  func(context.Context) error { return nil },
		},
	}
	shell, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer shell.Close()
	if _, err := shell.Start(context.Background()); err != nil {
		t.Fatalf("首次 Start: %v", err)
	}
	if _, err := shell.Start(context.Background()); err == nil {
		t.Fatal("重复 Start 应报错")
	}
}
