package host

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ShutdownPhase 是退出阶段。五阶段按序执行，每段各有超时。
type ShutdownPhase string

const (
	// PhaseStopIntents 停止接受新意图。
	PhaseStopIntents ShutdownPhase = "stop_intents"
	// PhaseEndInteraction 结束交互（有超时）。
	PhaseEndInteraction ShutdownPhase = "end_interaction"
	// PhaseCloseConnection 关闭连接。
	PhaseCloseConnection ShutdownPhase = "close_connection"
	// PhaseReclaimService 回收自启服务（有超时）。
	PhaseReclaimService ShutdownPhase = "reclaim_service"
	// PhaseFlushDiagnostics 冲刷诊断。
	PhaseFlushDiagnostics ShutdownPhase = "flush_diagnostics"
)

var shutdownSequence = []ShutdownPhase{
	PhaseStopIntents,
	PhaseEndInteraction,
	PhaseCloseConnection,
	PhaseReclaimService,
	PhaseFlushDiagnostics,
}

// ShutdownPhases 返回退出五阶段的顺序副本。
func ShutdownPhases() []ShutdownPhase {
	return append([]ShutdownPhase(nil), shutdownSequence...)
}

var (
	// ErrPhaseUnwired 表示退出阶段没有接线。
	ErrPhaseUnwired = errors.New("host: 退出阶段未接线")
	// ErrPhaseTimeout 表示退出阶段超时，已记录并继续。
	ErrPhaseTimeout = errors.New("host: 退出阶段超时")
)

// ShutdownSteps 是退出五阶段的执行体。前四段必须接线，缺一段在构造时报错；
// 冲刷诊断不接线时使用诊断记录自身的 Flush。
type ShutdownSteps struct {
	StopIntents      func(context.Context) error
	EndInteraction   func(context.Context) error
	CloseConnection  func(context.Context) error
	ReclaimService   func(context.Context) error
	FlushDiagnostics func(context.Context) error
}

// Timeouts 是退出各段的超时。小于等于 0 时取默认值。
type Timeouts struct {
	StopIntents      time.Duration
	EndInteraction   time.Duration
	CloseConnection  time.Duration
	ReclaimService   time.Duration
	FlushDiagnostics time.Duration
}

// DefaultTimeouts 是默认的退出超时：交互 3 秒（与"停止后 3 秒内非运行"对齐），
// 其余按各自网络或进程操作的上限取值。
func DefaultTimeouts() Timeouts {
	return Timeouts{
		StopIntents:      time.Second,
		EndInteraction:   3 * time.Second,
		CloseConnection:  2 * time.Second,
		ReclaimService:   5 * time.Second,
		FlushDiagnostics: time.Second,
	}
}

// withDefaults 把未设置或非法的超时补成默认值。
func (t Timeouts) withDefaults() Timeouts {
	d := DefaultTimeouts()
	if t.StopIntents <= 0 {
		t.StopIntents = d.StopIntents
	}
	if t.EndInteraction <= 0 {
		t.EndInteraction = d.EndInteraction
	}
	if t.CloseConnection <= 0 {
		t.CloseConnection = d.CloseConnection
	}
	if t.ReclaimService <= 0 {
		t.ReclaimService = d.ReclaimService
	}
	if t.FlushDiagnostics <= 0 {
		t.FlushDiagnostics = d.FlushDiagnostics
	}
	return t
}

func (t Timeouts) forPhase(phase ShutdownPhase) time.Duration {
	switch phase {
	case PhaseStopIntents:
		return t.StopIntents
	case PhaseEndInteraction:
		return t.EndInteraction
	case PhaseCloseConnection:
		return t.CloseConnection
	case PhaseReclaimService:
		return t.ReclaimService
	default:
		return t.FlushDiagnostics
	}
}

// PhaseResult 是一个退出阶段的结果。
type PhaseResult struct {
	Phase    ShutdownPhase
	Err      error
	TimedOut bool
	Elapsed  time.Duration
}

// ShutdownReport 是一次退出的完整结果。
type ShutdownReport struct {
	Results []PhaseResult
	Elapsed time.Duration
}

// TimedOut 返回超时的阶段。
func (r ShutdownReport) TimedOut() []ShutdownPhase {
	var phases []ShutdownPhase
	for _, res := range r.Results {
		if res.TimedOut {
			phases = append(phases, res.Phase)
		}
	}
	return phases
}

// Failed 返回失败的阶段（不含超时）。
func (r ShutdownReport) Failed() []ShutdownPhase {
	var phases []ShutdownPhase
	for _, res := range r.Results {
		if res.Err != nil && !res.TimedOut {
			phases = append(phases, res.Phase)
		}
	}
	return phases
}

// Complete 表示五段全部按序走完且没有超时或失败。
func (r ShutdownReport) Complete() bool {
	if len(r.Results) != len(shutdownSequence) {
		return false
	}
	for _, res := range r.Results {
		if res.Err != nil || res.TimedOut {
			return false
		}
	}
	return true
}

// Err 把超时与失败合并成一个错误；全部顺利时为 nil。
func (r ShutdownReport) Err() error {
	var errs []error
	for _, res := range r.Results {
		if res.TimedOut {
			errs = append(errs, fmt.Errorf("%w: %s", ErrPhaseTimeout, res.Phase))
		} else if res.Err != nil {
			errs = append(errs, fmt.Errorf("host: 退出阶段 %s 失败: %w", res.Phase, res.Err))
		}
	}
	return errors.Join(errs...)
}

// Shutdown 执行退出五阶段。任一段超时或失败都记一条诊断后继续，不无限等待。
type Shutdown struct {
	mu       sync.Mutex
	once     sync.Once
	diag     *Recorder
	store    *Store
	steps    ShutdownSteps
	timeouts Timeouts
	report   ShutdownReport
	err      error
}

// NewShutdown 校验并创建退出编排。
func NewShutdown(steps ShutdownSteps, store *Store, diag *Recorder, timeouts Timeouts) (*Shutdown, error) {
	if store == nil {
		return nil, errors.New("host: 退出缺少状态归属")
	}
	required := []struct {
		phase ShutdownPhase
		run   func(context.Context) error
	}{
		{PhaseStopIntents, steps.StopIntents},
		{PhaseEndInteraction, steps.EndInteraction},
		{PhaseCloseConnection, steps.CloseConnection},
		{PhaseReclaimService, steps.ReclaimService},
	}
	for _, item := range required {
		if item.run == nil {
			return nil, fmt.Errorf("%w: %s", ErrPhaseUnwired, item.phase)
		}
	}
	return &Shutdown{
		diag:     diag,
		store:    store,
		steps:    steps,
		timeouts: timeouts.withDefaults(),
	}, nil
}

// Run 按序执行五阶段。重复或并发调用都返回第一次的报告，不会重跑。
func (s *Shutdown) Run(ctx context.Context) (ShutdownReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.once.Do(func() {
		s.execute(ctx)
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.report, s.err
}

func (s *Shutdown) execute(ctx context.Context) {
	start := time.Now()
	// 第一阶段之前先落状态：从这一刻起新意图一律拒绝。
	s.store.BeginExit()
	s.diag.Log(LevelInfo, EventShutdownBegin, ExitingField(true))

	runners := []struct {
		phase ShutdownPhase
		run   func(context.Context) error
	}{
		{PhaseStopIntents, s.steps.StopIntents},
		{PhaseEndInteraction, s.steps.EndInteraction},
		{PhaseCloseConnection, s.steps.CloseConnection},
		{PhaseReclaimService, s.steps.ReclaimService},
		{PhaseFlushDiagnostics, s.flushStep()},
	}

	report := ShutdownReport{}
	for _, runner := range runners {
		report.Results = append(report.Results, s.runPhase(ctx, runner.phase, runner.run))
	}
	report.Elapsed = time.Since(start)

	s.store.FinishExit()
	s.diag.Log(LevelInfo, EventShutdownDone, ElapsedField(report.Elapsed))
	// 冲刷是第五阶段，收尾记录在其后再冲一次，保证 done 也落盘。
	_ = s.diag.Flush()

	err := report.Err()
	s.mu.Lock()
	s.report = report
	s.err = err
	s.mu.Unlock()
}

// runPhase 执行一段退出。即使父上下文已取消也照样收尾，但本段到点即返回。
func (s *Shutdown) runPhase(parent context.Context, phase ShutdownPhase, run func(context.Context) error) PhaseResult {
	timeout := s.timeouts.forPhase(phase)
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), timeout)
	defer cancel()

	start := time.Now()
	done := make(chan error, 1)
	go func() { done <- run(ctx) }()

	var err error
	timedOut := false
	select {
	case err = <-done:
	case <-ctx.Done():
		timedOut = true
		err = ctx.Err()
	}
	elapsed := time.Since(start)

	result := PhaseResult{Phase: phase, Err: err, TimedOut: timedOut, Elapsed: elapsed}
	switch {
	case timedOut:
		s.diag.Log(LevelWarn, EventShutdownTimeout,
			PhaseField(phase), ElapsedField(elapsed), TimeoutField(timeout))
	case err != nil:
		s.diag.Log(LevelError, EventShutdownPhase,
			PhaseField(phase), OutcomeField(OutcomeFailed),
			ClassField(classify(err)), ElapsedField(elapsed))
	default:
		s.diag.Log(LevelInfo, EventShutdownPhase,
			PhaseField(phase), OutcomeField(OutcomeOK), ElapsedField(elapsed))
	}
	return result
}

// flushStep 返回冲刷诊断阶段的执行体；未接线时使用诊断记录自身的 Flush。
func (s *Shutdown) flushStep() func(context.Context) error {
	if s.steps.FlushDiagnostics != nil {
		return s.steps.FlushDiagnostics
	}
	return func(context.Context) error { return s.diag.Flush() }
}
