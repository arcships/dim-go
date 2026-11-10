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
	WakeupExpired   WakeupStatus = "expired"
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

// WakeupRef 定位一条待取消的唤醒记录。真实方法 session.wakeup.cancel 需要会话
// 标识；分支标识只用于本地读数定位。
type WakeupRef struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId,omitempty"`
	WakeupID  string `json:"wakeupId"`
}

func (r WakeupRef) validate() error {
	if err := requireText("唤醒记录的会话标识", r.SessionID); err != nil {
		return err
	}
	return requireText("唤醒记录标识", r.WakeupID)
}

// WakeupCancelPayload 是 session.wakeup.cancel 意图的载荷。
type WakeupCancelPayload struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId,omitempty"`
	WakeupID  string `json:"wakeupId"`
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
		return missingReader(s, "session.wakeup.list", target)
	}
	return readInto(s, ctx, "session.wakeup.list", target, func(readCtx context.Context) ([]Wakeup, error) {
		wakeups, err := s.cfg.Wakeups.ReadWakeups(readCtx, scope)
		if err != nil {
			return nil, err
		}
		// A.md：取消后记录不再出现。服务端会保留 cancelled/expired 状态，本端
		// 只把仍然有效的记录放进读数；取消结果由写回执与通知确认。
		active := make([]Wakeup, 0, len(wakeups))
		for _, wakeup := range wakeups {
			if wakeup.Status == WakeupCancelled || wakeup.Status == WakeupExpired {
				continue
			}
			active = append(active, wakeup)
		}
		return active, nil
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
func (s *Service) CancelWakeup(ctx context.Context, ref WakeupRef) (write.IntentView, error) {
	if err := ref.validate(); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpWakeupCancel, WakeupCancelPayload{
		SessionID: ref.SessionID,
		BranchID:  ref.BranchID,
		WakeupID:  ref.WakeupID,
	}, false)
}
