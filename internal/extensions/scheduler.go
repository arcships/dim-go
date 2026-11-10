package extensions

import (
	"context"
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

// ListScheduledTasks 读取计划任务清单并更新镜像读数。
//
// 缺能力时返回 CapabilityError；能力恢复后再次读取即可得到任务列表，不需要重建
// 领域实例。读取只发生在显式调用与变更通知到达时，没有定时轮询。
func (s *Service) ListScheduledTasks(ctx context.Context) (Snapshot[ScheduledTask], error) {
	if s.cfg.Scheduler == nil {
		return missingReader(s, "schedule.list", &s.scheduled)
	}
	return readInto(s, ctx, "schedule.list", &s.scheduled, s.cfg.Scheduler.ReadScheduledTasks)
}

// ScheduledTaskSnapshot 返回最近一次计划任务读数，不发起请求。
func (s *Service) ScheduledTaskSnapshot() Snapshot[ScheduledTask] {
	return snapshotOf(s, &s.scheduled)
}
