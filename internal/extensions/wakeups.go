package extensions

import (
	"context"
	"time"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// WakeupStatus 是一条唤醒记录的状态。
type WakeupStatus string

const (
	WakeupScheduled WakeupStatus = "scheduled"
	WakeupFired     WakeupStatus = "fired"
	WakeupCancelled WakeupStatus = "cancelled"
	WakeupUnknown   WakeupStatus = "unknown"
)

// WakeupScope 是唤醒记录的读取范围：会话必填，分支可选。
type WakeupScope struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId,omitempty"`
}

func (s WakeupScope) validate() error {
	return requireText("唤醒记录的会话标识", s.SessionID)
}

func scopeKey(scope WakeupScope) string {
	return scope.SessionID + "\x00" + scope.BranchID
}

// Wakeup 是一条挂在会话或分支上的唤醒记录。取消后服务端不再返回该记录。
type Wakeup struct {
	ID        string
	SessionID string
	BranchID  string
	Title     string
	Prompt    string
	TriggerAt time.Time
	Status    WakeupStatus
}

// WakeupReader 读取某个会话或分支的唤醒记录。只读。
type WakeupReader interface {
	ReadWakeups(ctx context.Context, scope WakeupScope) ([]Wakeup, error)
}

// WakeupCancelPayload 是 wakeup.cancel 意图的载荷。
type WakeupCancelPayload struct {
	WakeupID string `json:"wakeupId"`
}

// ListWakeups 读取唤醒记录并更新该范围的镜像读数。读取过的范围会接收后续
// 变更通知；通知到达时只重读一次。
func (s *Service) ListWakeups(ctx context.Context, scope WakeupScope) (Snapshot[Wakeup], error) {
	if err := scope.validate(); err != nil {
		return Snapshot[Wakeup]{}, err
	}
	s.watchWakeups(scope)
	return s.readWakeups(ctx, scope)
}

// WakeupSnapshot 返回某个范围最近一次唤醒读数，不发起请求。
func (s *Service) WakeupSnapshot(scope WakeupScope) Snapshot[Wakeup] {
	key := scopeKey(scope)
	s.mu.Lock()
	target, ok := s.wakeups[key]
	s.mu.Unlock()
	if !ok {
		return Snapshot[Wakeup]{}
	}
	return snapshotOf(s, target)
}

func (s *Service) readWakeups(ctx context.Context, scope WakeupScope) (Snapshot[Wakeup], error) {
	target := s.wakeupTarget(scope)
	if s.cfg.Wakeups == nil {
		return missingReader(s, "wakeup.list", target)
	}
	return readInto(s, ctx, "wakeup.list", target, func(readCtx context.Context) ([]Wakeup, error) {
		return s.cfg.Wakeups.ReadWakeups(readCtx, scope)
	})
}

func (s *Service) wakeupTarget(scope WakeupScope) *Snapshot[Wakeup] {
	key := scopeKey(scope)
	s.mu.Lock()
	defer s.mu.Unlock()
	target, ok := s.wakeups[key]
	if !ok {
		target = &Snapshot[Wakeup]{}
		s.wakeups[key] = target
	}
	return target
}

func (s *Service) watchWakeups(scope WakeupScope) {
	s.mu.Lock()
	s.wakeupWatch[scopeKey(scope)] = scope
	s.mu.Unlock()
}

// CancelWakeup 取消一条唤醒记录。写入经写登记；本包不因取消成功而自行重读，
// 记录从读数里消失由服务端通知触发的那一次重读确认。
func (s *Service) CancelWakeup(ctx context.Context, wakeupID string) (write.IntentView, error) {
	if err := requireText("唤醒记录标识", wakeupID); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpWakeupCancel, WakeupCancelPayload{WakeupID: wakeupID}, false)
}
