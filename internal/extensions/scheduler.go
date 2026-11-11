package extensions

import (
	"context"
	"errors"
	"time"
)

// ScheduledTask 是一条计划任务读数。状态与时间都由服务端给出，本端不推算下次
// 运行时间，也不自行调度。
type ScheduledTask struct {
	ID         string
	Name       string
	Schedule   string
	Enabled    bool
	Running    bool
	NextRunAt  time.Time
	LastRunAt  time.Time
	LastStatus string
	// Revision/Prompt/Cwd/Mode 是 alpha.21 job 上的字段，原样保留供界面展示。
	Revision int64
	Prompt   string
	Cwd      string
	Mode     string
}

// SchedulerReader 读取计划任务清单。只读。
type SchedulerReader interface {
	ReadScheduledTasks(ctx context.Context) ([]ScheduledTask, error)
}

// ScheduledRun 是 schedule.history 里的一次运行。
type ScheduledRun struct {
	ID         string
	JobID      string
	Window     string
	State      string
	CreatedAt  string
	EndedAt    string
	SessionID  string
	Submission string
}

// ScheduleHistoryReader 读取某个任务的运行历史。只读。
type ScheduleHistoryReader interface {
	ReadScheduleHistory(ctx context.Context, jobID string, limit int) ([]ScheduledRun, error)
}

// ListScheduledTasks 读取计划任务清单并更新镜像读数。
//
// 缺能力时返回 CapabilityError；能力恢复后再次读取即可得到任务列表，不需要重建
// 领域实例。读取只发生在显式调用与变更通知到达时，没有定时轮询。
func (s *Service) ListScheduledTasks(ctx context.Context) (Snapshot[ScheduledTask], error) {
	if s.cfg.Scheduler == nil {
		return missingReader(s, "schedule.list", &s.scheduled)
	}
	snapshot, err := readInto(s, ctx, "schedule.list", &s.scheduled, s.cfg.Scheduler.ReadScheduledTasks)
	s.mu.Lock()
	s.scheduleReads++
	s.mu.Unlock()
	return snapshot, err
}

// ListScheduleHistory 读取某个任务的运行历史并缓存，供 schedule.watch 的 run
// 通知重读；limit 为 0 表示不声明上限。
func (s *Service) ListScheduleHistory(ctx context.Context, jobID string, limit int) (Snapshot[ScheduledRun], error) {
	if err := requireText("计划任务标识", jobID); err != nil {
		return Snapshot[ScheduledRun]{}, err
	}
	if err := s.requireFeature("schedule.history"); err != nil {
		return Snapshot[ScheduledRun]{}, err
	}
	reader, ok := s.cfg.Scheduler.(ScheduleHistoryReader)
	if !ok {
		return Snapshot[ScheduledRun]{}, s.wrapReadError("schedule.history", ErrCapabilityMissing)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.readMu.Lock()
	defer s.readMu.Unlock()
	if s.isClosed() {
		return Snapshot[ScheduledRun]{}, ErrClosed
	}
	target := s.scheduleHistoryTarget(jobID)
	runs, err := reader.ReadScheduleHistory(ctx, jobID, limit)
	if err != nil {
		wrapped := s.wrapReadError("schedule.history", err)
		s.mu.Lock()
		target.ReadError = wrapped.Error()
		target.CapabilityMissing = errors.Is(wrapped, ErrCapabilityMissing)
		snapshot := cloneSnapshot(*target)
		s.mu.Unlock()
		return snapshot, wrapped
	}
	s.mu.Lock()
	target.Items = append([]ScheduledRun(nil), runs...)
	target.ReadAt = s.cfg.Now()
	target.ReadError = ""
	target.CapabilityMissing = false
	s.scheduleHistoryReads++
	snapshot := cloneSnapshot(*target)
	s.mu.Unlock()
	return snapshot, nil
}

// ScheduleHistorySnapshot 返回某个任务最近一次历史读数，不发起请求。
func (s *Service) ScheduleHistorySnapshot(jobID string) Snapshot[ScheduledRun] {
	s.mu.Lock()
	target, ok := s.scheduleHistory[jobID]
	s.mu.Unlock()
	if !ok {
		return Snapshot[ScheduledRun]{}
	}
	return snapshotOf(s, target)
}

func (s *Service) scheduleHistoryTarget(jobID string) *Snapshot[ScheduledRun] {
	s.mu.Lock()
	defer s.mu.Unlock()
	target, ok := s.scheduleHistory[jobID]
	if !ok {
		target = &Snapshot[ScheduledRun]{}
		s.scheduleHistory[jobID] = target
	}
	return target
}

// ScheduledTaskReads 返回 plan 任务的读取次数（显式与跟随），用于验证不轮询。
func (s *Service) ScheduledTaskReads() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scheduleReads
}

// ScheduleHistoryReads 返回任务历史读取次数。
func (s *Service) ScheduleHistoryReads() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scheduleHistoryReads
}

// ScheduledTaskSnapshot 返回最近一次计划任务读数，不发起请求。
func (s *Service) ScheduledTaskSnapshot() Snapshot[ScheduledTask] {
	return snapshotOf(s, &s.scheduled)
}
