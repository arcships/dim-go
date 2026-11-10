package host

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Stage 是启动阶段。六阶段按序推进，完成"允许写"后进入 ready 终态。
type Stage string

const (
	// StageIdle 是进程刚启动、尚未判定单实例的状态。
	StageIdle Stage = "idle"
	// StageSingleInstance 是单实例判定。
	StageSingleInstance Stage = "single_instance"
	// StageLocalState 是装载本地状态（失败按空处理）。
	StageLocalState Stage = "local_state"
	// StageConnect 是连服务。
	StageConnect Stage = "connect"
	// StageSubscribe 是订阅。
	StageSubscribe Stage = "subscribe"
	// StageFirstHistory 是首轮历史。
	StageFirstHistory Stage = "first_history"
	// StageAllowWrites 是允许写。
	StageAllowWrites Stage = "allow_writes"
	// StageReady 是六阶段全部完成的终态。
	StageReady Stage = "ready"
)

// startupSequence 是六个启动阶段的固定顺序。
var startupSequence = []Stage{
	StageSingleInstance,
	StageLocalState,
	StageConnect,
	StageSubscribe,
	StageFirstHistory,
	StageAllowWrites,
}

// StartupStages 返回六个启动阶段的顺序副本。
func StartupStages() []Stage {
	return append([]Stage(nil), startupSequence...)
}

// stagePosition 返回阶段序号；idle 为 -1，未知阶段为 -2。
func stagePosition(stage Stage) int {
	if stage == StageIdle {
		return -1
	}
	if stage == StageReady {
		return len(startupSequence)
	}
	for i, s := range startupSequence {
		if s == stage {
			return i
		}
	}
	return -2
}

// nextStage 返回阶段推进的下一站。
func nextStage(stage Stage) (Stage, bool) {
	pos := stagePosition(stage)
	switch {
	case pos == -1:
		return StageSingleInstance, true
	case pos >= 0 && pos < len(startupSequence):
		if pos+1 == len(startupSequence) {
			return StageReady, true
		}
		return startupSequence[pos+1], true
	default:
		return "", false
	}
}

// stagesAfter 返回当前阶段之后的全部阶段。
func stagesAfter(stage Stage) []Stage {
	pos := stagePosition(stage)
	if pos == -2 || pos == len(startupSequence) {
		return nil
	}
	rest := startupSequence[pos+1:]
	return append([]Stage(nil), rest...)
}

var (
	// ErrNotReady 表示启动阶段尚未完成。
	ErrNotReady = errors.New("host: 启动尚未完成")
	// ErrNotConnected 表示当前没有连接，不能发服务端请求。
	ErrNotConnected = errors.New("host: 尚未连接服务端")
	// ErrExiting 表示进程正在退出，不再接受新意图。
	ErrExiting = errors.New("host: 正在退出")
	// ErrInvalidStage 表示一次不合法的启动阶段推进，已被丢弃。
	ErrInvalidStage = errors.New("host: 启动阶段推进不合法")
)

// State 是进程级状态的只读快照。
type State struct {
	// Stage 是当前阶段：已完成的最后一个阶段，或停在的失败阶段。
	Stage Stage
	// Failed 是导致启动停在当前阶段的错误；正常推进时为 nil。
	Failed error
	// Connected 是连接领域此刻的只读观察值，host 不写它。
	Connected bool
	// Exiting 表示退出已经开始，新意图一律拒绝。
	Exiting bool
	// Exited 表示退出五阶段已经走完。
	Exited bool
}

// StoreConfig 是进程级状态的构造参数。
type StoreConfig struct {
	// Diag 是诊断写入端，可为 nil。
	Diag *Recorder
	// Connected 是连接状态的只读探针，可为 nil（视为未连接）。
	Connected func() bool
}

// Store 持有并驱动进程级状态：启动阶段、失败、退出标记。
// 连接状态只读不写，唯一驱动者仍是连接领域。
type Store struct {
	mu        sync.Mutex
	diag      *Recorder
	connected func() bool
	stage     Stage
	failed    error
	exiting   bool
	exited    bool
	changedAt time.Time
}

// NewStore 创建进程级状态，初始阶段为 idle。
func NewStore(cfg StoreConfig) *Store {
	return &Store{
		diag:      cfg.Diag,
		connected: cfg.Connected,
		stage:     StageIdle,
		changedAt: time.Now(),
	}
}

// Stage 返回当前阶段。
func (s *Store) Stage() Stage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stage
}

// Snapshot 返回进程级状态的只读快照；连接探针在锁外调用，避免与连接领域互锁。
func (s *Store) Snapshot() State {
	s.mu.Lock()
	state := State{
		Stage:   s.stage,
		Failed:  s.failed,
		Exiting: s.exiting,
		Exited:  s.exited,
	}
	probe := s.connected
	s.mu.Unlock()
	if probe != nil {
		state.Connected = probe()
	}
	return state
}

// Advance 记录一个阶段的完成。只接受严格按序推进：失败中、退出中或跳跃推进一律丢弃，
// 写一条诊断并返回 ErrInvalidStage，状态不变。
func (s *Store) Advance(stage Stage) error {
	s.mu.Lock()
	from := s.stage
	if s.failed != nil || s.exiting || s.exited {
		s.mu.Unlock()
		s.reject(from, stage)
		return ErrInvalidStage
	}
	want, ok := nextStage(from)
	if !ok || want != stage {
		s.mu.Unlock()
		s.reject(from, stage)
		return ErrInvalidStage
	}
	elapsed := time.Since(s.changedAt)
	s.stage = stage
	s.changedAt = time.Now()
	s.mu.Unlock()

	s.diag.Log(LevelInfo, EventStartupStage,
		StageField(stage), OutcomeField(OutcomeOK), ElapsedField(elapsed))
	return nil
}

// Fail 记录启动停在某个阶段。失败后不再接受任何阶段推进，直到进程重启。
func (s *Store) Fail(stage Stage, err error) {
	s.mu.Lock()
	elapsed := time.Since(s.changedAt)
	s.stage = stage
	s.failed = err
	s.changedAt = time.Now()
	s.mu.Unlock()

	s.diag.Log(LevelError, EventStartupFailed,
		StageField(stage), OutcomeField(OutcomeFailed),
		ClassField(classify(err)), ElapsedField(elapsed))
}

// BeginExit 标记退出开始；从此刻起新意图一律拒绝。
func (s *Store) BeginExit() {
	s.mu.Lock()
	s.exiting = true
	s.changedAt = time.Now()
	s.mu.Unlock()
}

// FinishExit 标记退出五阶段已经走完。
func (s *Store) FinishExit() {
	s.mu.Lock()
	s.exited = true
	s.mu.Unlock()
}

// Writable 判断此刻能否发服务端请求。启动未完成、启动失败、没有连接、正在退出
// 四种情况分别返回明确错误，不做"未连接当空数据"的降级。
func (s *Store) Writable() error {
	state := s.Snapshot()
	switch {
	case state.Exiting:
		return ErrExiting
	case state.Failed != nil:
		return fmt.Errorf("%w: %w", ErrNotReady, state.Failed)
	case state.Stage != StageReady:
		return fmt.Errorf("%w: 当前阶段 %s", ErrNotReady, state.Stage)
	case !state.Connected:
		return ErrNotConnected
	default:
		return nil
	}
}

// ObserveConnection 记录一次观察到的连接状态变化。连接状态本身由连接领域持有，
// 这里只写诊断，不改状态。
func (s *Store) ObserveConnection(connected bool, class ErrorClass) {
	fields := []Field{ConnectedField(connected)}
	if !connected || class != "" {
		fields = append(fields, ClassField(class))
	}
	s.diag.Log(LevelInfo, EventConnectionChanged, fields...)
}

func (s *Store) reject(from, to Stage) {
	s.diag.Log(LevelWarn, EventStageRejected,
		FromStageField(from), NextStageField(to), OutcomeField(OutcomeSkipped))
}
