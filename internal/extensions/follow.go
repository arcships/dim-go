package extensions

import (
	"context"
	"sort"
	"sync"
)

// Topic 是管理面变更通知的主题。集成层把服务端通知映射成主题，本领域不解释
// 具体通知方法名。
type Topic string

const (
	TopicPlugins   Topic = "plugins"
	TopicSkills    Topic = "skills"
	TopicHooks     Topic = "hooks"
	TopicMCP       Topic = "mcp"
	TopicSchedule  Topic = "schedule"
	TopicWorkflows Topic = "workflows"
	TopicWakeups   Topic = "wakeups"
)

// 以下别名只为兼容尚未跟随改名的集成层与测试；新代码使用上面的名称。
const (
	TopicToolServers = TopicMCP
	TopicScheduler   = TopicSchedule
)

// Notice 是一条管理面变更通知。
//
// 计划任务与插件等全局主题只看 Topic；工作流按 WorkflowID 定位；唤醒按会话与
// 分支定位。scope 相同的通知在读取开始前合并，最多重读一次。
type Notice struct {
	Topic      Topic
	SessionID  string
	BranchID   string
	WorkflowID string
	// Sequence/Kind/JobID 用于 schedule.watch：序号缺口时重读，run 通知重读历史。
	// SequenceSet 为 false 时保持旧行为（每条通知都重读）。
	Sequence    int64
	SequenceSet bool
	Kind        string
	JobID       string
}

// scopeKey 是同主题通知的合并键。
func (n Notice) scopeKey() string {
	switch n.Topic {
	case TopicWakeups:
		return n.SessionID + "\x00" + n.BranchID
	case TopicWorkflows:
		return n.WorkflowID
	default:
		return ""
	}
}

// FollowStats 是事件跟随的读数字段，供诊断与验收观察。
type FollowStats struct {
	// Notices 是收到的通知条数（按主题）。
	Notices map[Topic]uint64 `json:"notices"`
	// Reads 是通知驱动的重读次数（按主题）。
	Reads map[Topic]uint64 `json:"reads"`
	// WakeupReads 是通知驱动的唤醒重读次数（按会话与分支）。
	WakeupReads map[string]uint64 `json:"wakeupReads"`
}

// FollowStats 返回事件跟随计数副本。
func (s *Service) FollowStats() FollowStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	stats := FollowStats{
		Notices:     make(map[Topic]uint64, len(s.followStats.Notices)),
		Reads:       make(map[Topic]uint64, len(s.followStats.Reads)),
		WakeupReads: make(map[string]uint64, len(s.followStats.WakeupReads)),
	}
	for topic, count := range s.followStats.Notices {
		stats.Notices[topic] = count
	}
	for topic, count := range s.followStats.Reads {
		stats.Reads[topic] = count
	}
	for scope, count := range s.followStats.WakeupReads {
		stats.WakeupReads[scope] = count
	}
	return stats
}

// Notify 登记一条管理面变更通知。它不阻塞、不发请求，由唯一的跟随协程按主题
// 重读一次。关闭后返回 ErrClosed。
func (s *Service) Notify(notice Notice) error {
	if s.isClosed() {
		return ErrClosed
	}
	if notice.Topic == "" {
		return invalidRequest("通知缺少主题")
	}
	return s.follower.notify(notice)
}

// WaitIdle 等待已登记的通知全部重读完成。ctx 结束返回 ctx.Err()。
func (s *Service) WaitIdle(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return s.follower.waitIdle(ctx)
}

// handleNotice 是跟随协程唯一的读入口：每个通知对每个作用域只重读一次。
func (s *Service) handleNotice(ctx context.Context, notice Notice) {
	s.recordNotice(notice)
	s.cfg.Logf("extensions: 跟随重读 topic=%s scope=%q", notice.Topic, notice.scopeKey())

	switch notice.Topic {
	case TopicPlugins:
		if s.cfg.Plugins != nil {
			_, _ = s.ListPlugins(ctx)
		}
	case TopicSkills:
		if s.cfg.Skills != nil {
			_, _ = s.ListSkills(ctx)
		}
	case TopicHooks:
		if s.cfg.Hooks != nil {
			_, _ = s.ListHooks(ctx)
		}
	case TopicMCP:
		if s.cfg.ToolServers != nil {
			_, _ = s.ListToolServers(ctx)
		}
	case TopicSchedule:
		s.followSchedule(ctx, notice)
	case TopicWorkflows:
		s.followWorkflows(ctx, notice)
	case TopicWakeups:
		s.followWakeups(ctx, notice)
	default:
		s.cfg.Logf("extensions: 未知跟随主题 topic=%s", notice.Topic)
	}
	// 工作流与唤醒按每个作用域分别计数；其他主题一次通知就是一次重读。
	switch notice.Topic {
	case TopicWorkflows, TopicWakeups:
	default:
		s.recordRead(notice.Topic)
	}
}

func (s *Service) followWorkflows(ctx context.Context, notice Notice) {
	targets := make([]string, 0, 1)
	if notice.WorkflowID != "" {
		s.mu.Lock()
		if _, watched := s.workflowWatch[notice.WorkflowID]; watched {
			targets = append(targets, notice.WorkflowID)
		}
		s.mu.Unlock()
	} else {
		s.mu.Lock()
		for workflowID := range s.workflowWatch {
			targets = append(targets, workflowID)
		}
		s.mu.Unlock()
		sort.Strings(targets)
	}
	for _, workflowID := range targets {
		_, _ = s.readWorkflowRuns(ctx, workflowID)
		s.recordRead(TopicWorkflows)
	}
}

func (s *Service) followSchedule(ctx context.Context, notice Notice) {
	if s.cfg.Scheduler == nil {
		return
	}
	gap := false
	duplicate := false
	if notice.SequenceSet {
		s.mu.Lock()
		seen := s.scheduleSequenceSeen
		last := s.scheduleSequence
		switch {
		case !seen:
			s.scheduleSequenceSeen = true
			s.scheduleSequence = notice.Sequence
		case notice.Sequence > last:
			gap = notice.Sequence > last+1
			s.scheduleSequence = notice.Sequence
		default:
			duplicate = true
		}
		s.mu.Unlock()
		if duplicate {
			// 同一个序号重复到达：不重复重读，避开服务端的进度重发。
			return
		}
		if gap {
			s.cfg.Logf("extensions: schedule.watch 序号缺口 last=%d next=%d", last, notice.Sequence)
		}
	}
	_, _ = s.ListScheduledTasks(ctx)
	if notice.Kind == "run" && notice.JobID != "" {
		_, _ = s.ListScheduleHistory(ctx, notice.JobID, 0)
	}
}

func (s *Service) followWakeups(ctx context.Context, notice Notice) {
	scopes := make([]WakeupScope, 0, 1)
	if notice.SessionID != "" || notice.BranchID != "" {
		target := WakeupScope{SessionID: notice.SessionID, BranchID: notice.BranchID}
		s.mu.Lock()
		if _, watched := s.wakeupWatch[scopeKey(target)]; watched {
			scopes = append(scopes, target)
		}
		s.mu.Unlock()
	} else {
		s.mu.Lock()
		for _, scope := range s.wakeupWatch {
			scopes = append(scopes, scope)
		}
		s.mu.Unlock()
		sort.Slice(scopes, func(i, j int) bool { return scopeKey(scopes[i]) < scopeKey(scopes[j]) })
	}
	for _, scope := range scopes {
		_, _ = s.readWakeups(ctx, scope)
		s.recordWakeupRead(scope)
		s.recordRead(TopicWakeups)
	}
}

func (s *Service) recordNotice(notice Notice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.followStats.Notices == nil {
		s.followStats.Notices = make(map[Topic]uint64)
		s.followStats.Reads = make(map[Topic]uint64)
		s.followStats.WakeupReads = make(map[string]uint64)
	}
	s.followStats.Notices[notice.Topic]++
}

func (s *Service) recordRead(topic Topic) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.followStats.Reads == nil {
		s.followStats.Reads = make(map[Topic]uint64)
	}
	s.followStats.Reads[topic]++
}

func (s *Service) recordWakeupRead(scope WakeupScope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.followStats.WakeupReads == nil {
		s.followStats.WakeupReads = make(map[string]uint64)
	}
	s.followStats.WakeupReads[scopeKey(scope)]++
}

// follower 是唯一的事件跟随协程：先合并同一作用域的待处理通知，再逐一重读。
// 它没有定时器，也不会周期性拉取全量。
type follower struct {
	mu      sync.Mutex
	handler func(context.Context, Notice)

	pending map[string]Notice
	order   []string
	busy    bool
	closed  bool
	wake    chan struct{}
	idle    chan struct{}
	stop    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func newFollower(handler func(context.Context, Notice)) *follower {
	ctx, cancel := context.WithCancel(context.Background())
	f := &follower{
		handler: handler,
		pending: make(map[string]Notice),
		wake:    make(chan struct{}, 1),
		idle:    make(chan struct{}),
		stop:    make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
	close(f.idle) // 初始空闲
	f.wg.Add(1)
	go f.run()
	return f
}

func (f *follower) notify(notice Notice) error {
	key := string(notice.Topic) + "\x00" + notice.scopeKey()
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return ErrClosed
	}
	if _, duplicate := f.pending[key]; duplicate {
		f.mu.Unlock()
		return nil
	}
	f.pending[key] = notice
	f.order = append(f.order, key)
	if !f.busy {
		f.busy = true
		f.idle = make(chan struct{})
	}
	f.mu.Unlock()
	select {
	case f.wake <- struct{}{}:
	default:
	}
	return nil
}

func (f *follower) run() {
	defer f.wg.Done()
	for {
		select {
		case <-f.wake:
		case <-f.stop:
			return
		}
		for {
			f.mu.Lock()
			if f.closed {
				f.mu.Unlock()
				return
			}
			if len(f.order) == 0 {
				if f.busy {
					f.busy = false
					close(f.idle)
				}
				f.mu.Unlock()
				break
			}
			key := f.order[0]
			f.order = f.order[1:]
			notice := f.pending[key]
			delete(f.pending, key)
			f.mu.Unlock()

			f.runHandler(notice)
		}
	}
}

// runHandler 隔离处理器 panic：一条坏通知不能带走唯一的跟随协程。
func (f *follower) runHandler(notice Notice) {
	defer func() { _ = recover() }()
	f.handler(f.ctx, notice)
}

func (f *follower) waitIdle(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		f.mu.Lock()
		if !f.busy {
			f.mu.Unlock()
			return nil
		}
		idle := f.idle
		f.mu.Unlock()
		select {
		case <-idle:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (f *follower) close() {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return
	}
	f.closed = true
	f.pending = make(map[string]Notice)
	f.order = nil
	if f.busy {
		f.busy = false
		close(f.idle)
	}
	f.mu.Unlock()

	f.cancel()
	close(f.stop)
	f.wg.Wait()
}
