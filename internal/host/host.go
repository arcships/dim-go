package host

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// StartOutcome 是启动的两种结局。
type StartOutcome int

const (
	// StartPrimary 表示本进程是主实例，启动阶段已按序推进（成功或失败看返回的错误）。
	StartPrimary StartOutcome = iota
	// StartForwarded 表示已有实例在运行，本次启动参数已转交，本进程应立即退出。
	StartForwarded
)

// String 返回可读的结局名。
func (o StartOutcome) String() string {
	if o == StartForwarded {
		return "forwarded"
	}
	return "primary"
}

// ErrForwardFailed 表示二次启动没能把参数转交给已有实例。
var ErrForwardFailed = errors.New("host: 转交启动参数失败")

// StartupSteps 是六阶段中除"单实例"之外的五个阶段的实现。
// 每个阶段都必须接线；缺实现的阶段在构造时即报错，不会被静默跳过。
type StartupSteps struct {
	// LocalState 装载本地状态；失败按空处理由实现方决定。
	LocalState func(context.Context) error
	// Connect 连服务并完成握手。
	Connect func(context.Context) error
	// Subscribe 建立订阅。
	Subscribe func(context.Context) error
	// FirstHistory 读取首轮历史。
	FirstHistory func(context.Context) error
	// AllowWrites 打开写入闸门。
	AllowWrites func(context.Context) error
}

func (s StartupSteps) stageSteps() []StageStep {
	return []StageStep{
		{Stage: StageLocalState, Run: s.LocalState},
		{Stage: StageConnect, Run: s.Connect},
		{Stage: StageSubscribe, Run: s.Subscribe},
		{Stage: StageFirstHistory, Run: s.FirstHistory},
		{Stage: StageAllowWrites, Run: s.AllowWrites},
	}
}

func validateStartupSteps(steps StartupSteps) error {
	for _, step := range steps.stageSteps() {
		if step.Run == nil {
			return fmt.Errorf("%w: %s", ErrStageUnwired, step.Stage)
		}
	}
	return nil
}

// Config 是宿主外壳的构造参数。
type Config struct {
	// DataDir 是应用数据目录，用于推导实例端点。
	DataDir string
	// Endpoint 是实例端点的完整路径；设置后优先于 DataDir。
	Endpoint string
	// Args 是本次启动参数；二次启动时原样转交给已有实例。
	Args []string
	// WorkingDir 是本次启动的工作目录，随转交一起送出。
	WorkingDir string
	// Version 是客户端版本，只写进诊断，必须是 x.y.z 形式。
	Version string
	// Diag 是诊断写入端，可为 nil。
	Diag *Recorder
	// Connected 是连接状态的只读探针，可为 nil（视为未连接）。
	Connected func() bool
	// Startup 是五阶段实现。
	Startup StartupSteps
	// Shutdown 是退出五阶段实现。
	Shutdown ShutdownSteps
	// Timeouts 是退出各段超时；零值取默认值。
	Timeouts Timeouts
	// Instance 覆盖单实例端点的细节配置；Endpoint、DataDir、Diag 为空时取上层配置。
	Instance InstanceConfig
}

// Shell 把单实例、启动阶段、退出顺序、诊断与进程级状态编排到一起。
// 它不是任何一个领域状态的第二驱动者：连接状态只经 Connected 探针读取。
type Shell struct {
	cfg   Config
	diag  *Recorder
	store *Store

	mu       sync.Mutex
	instance *Instance
	startup  *Startup
	shutdown *Shutdown
	started  bool
	outcome  StartOutcome
}

// New 校验并创建宿主外壳。
func New(cfg Config) (*Shell, error) {
	if err := validateStartupSteps(cfg.Startup); err != nil {
		return nil, err
	}
	if cfg.Endpoint == "" && cfg.DataDir == "" {
		return nil, errors.New("host: 未配置实例端点或数据目录")
	}
	instance := cfg.Instance
	if instance.Endpoint == "" {
		instance.Endpoint = cfg.Endpoint
	}
	if instance.DataDir == "" {
		instance.DataDir = cfg.DataDir
	}
	if instance.Diag == nil {
		instance.Diag = cfg.Diag
	}
	cfg.Instance = instance

	store := NewStore(StoreConfig{Diag: cfg.Diag, Connected: cfg.Connected})
	return &Shell{cfg: cfg, diag: cfg.Diag, store: store}, nil
}

// Start 尝试成为主实例并按六阶段推进。已有实例时转交参数并返回 StartForwarded；
// 转交失败会再抢一次端点（已有实例可能刚好退出），仍失败则如实返回错误。
func (s *Shell) Start(ctx context.Context) (StartOutcome, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	if s.started {
		outcome := s.outcome
		s.mu.Unlock()
		return outcome, errors.New("host: 启动只能调用一次")
	}
	s.started = true
	s.mu.Unlock()

	startedAt := time.Now()
	instance, err := TryAcquire(s.cfg.Instance)
	if errors.Is(err, ErrInstanceRunning) {
		if forwardErr := s.forward(); forwardErr == nil {
			s.mu.Lock()
			s.outcome = StartForwarded
			s.mu.Unlock()
			return StartForwarded, nil
		} else if instance, err = TryAcquire(s.cfg.Instance); err != nil {
			return StartPrimary, forwardErr
		}
	}
	if err != nil {
		s.store.Fail(StageSingleInstance, err)
		return StartPrimary, err
	}

	s.mu.Lock()
	s.instance = instance
	s.mu.Unlock()
	s.diag.Log(LevelInfo, EventInstancePrimary)
	if err := s.store.Advance(StageSingleInstance); err != nil {
		return StartPrimary, err
	}

	startup, err := NewStartup(s.cfg.Startup.stageSteps(), s.store)
	if err != nil {
		return StartPrimary, err
	}
	s.mu.Lock()
	s.startup = startup
	s.mu.Unlock()
	if err := startup.Run(ctx); err != nil {
		return StartPrimary, err
	}

	fields := []Field{ElapsedField(time.Since(startedAt))}
	if s.cfg.Version != "" {
		fields = append(fields, VersionField(s.cfg.Version))
	}
	s.diag.Log(LevelInfo, EventStartupReady, fields...)
	return StartPrimary, nil
}

// forward 把本次启动参数转交给已有实例。参数内容不写诊断，只记个数。
func (s *Shell) forward() error {
	activation := Activation{
		Args:       append([]string(nil), s.cfg.Args...),
		WorkingDir: s.cfg.WorkingDir,
		At:         time.Now(),
	}
	if err := Forward(s.cfg.Instance, activation); err != nil {
		s.diag.Log(LevelError, EventInstanceForwardFailed,
			OutcomeField(OutcomeFailed), ClassField(classify(err)))
		return fmt.Errorf("%w: %w", ErrForwardFailed, err)
	}
	s.diag.Log(LevelInfo, EventInstanceForwarded, ArgsField(len(activation.Args)))
	return nil
}

// State 返回进程级状态的只读快照。
func (s *Shell) State() State { return s.store.Snapshot() }

// Writable 判断此刻能否发服务端请求；启动未完成、启动失败、未连接、正在退出都会被拒。
func (s *Shell) Writable() error { return s.store.Writable() }

// ObserveConnection 记录一次观察到的连接状态变化，供诊断使用；不改变任何连接状态。
func (s *Shell) ObserveConnection(connected bool, class ErrorClass) {
	s.store.ObserveConnection(connected, class)
}

// Activations 返回二次启动转交过来的参数队列；本进程不是主实例时返回 nil。
func (s *Shell) Activations() <-chan Activation {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.instance == nil {
		return nil
	}
	return s.instance.Activations()
}

// Shutdown 执行退出五阶段，然后释放实例端点。重复调用返回第一次的报告。
func (s *Shell) Shutdown(ctx context.Context) (ShutdownReport, error) {
	s.mu.Lock()
	shutdown := s.shutdown
	if shutdown == nil {
		var err error
		shutdown, err = NewShutdown(s.cfg.Shutdown, s.store, s.diag, s.cfg.Timeouts)
		if err != nil {
			s.mu.Unlock()
			return ShutdownReport{}, err
		}
		s.shutdown = shutdown
	}
	s.mu.Unlock()

	report, err := shutdown.Run(ctx)
	_ = s.Close()
	return report, err
}

// Close 释放实例端点。重复调用安全。
func (s *Shell) Close() error {
	s.mu.Lock()
	instance := s.instance
	s.instance = nil
	s.mu.Unlock()
	if instance == nil {
		return nil
	}
	return instance.Close()
}
