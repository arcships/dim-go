package host

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrStageUnwired 表示某个启动阶段没有接线。缺实现的阶段不会被静默跳过，
// 而是在构造时就报出来。
var ErrStageUnwired = errors.New("host: 启动阶段未接线")

// StageStep 是一个启动阶段的执行体。
type StageStep struct {
	// Stage 必须是六个启动阶段之一。
	Stage Stage
	// Run 执行该阶段。返回错误表示停在该阶段。
	Run func(context.Context) error
}

// Startup 按六阶段顺序推进启动。阶段列表必须完整覆盖当前阶段之后的全部阶段，
// 顺序与阶段名都必须与架构一致，否则构造失败。
type Startup struct {
	mu    sync.Mutex
	store *Store
	steps []StageStep
	ran   bool
}

// NewStartup 校验并创建启动编排。
func NewStartup(steps []StageStep, store *Store) (*Startup, error) {
	if store == nil {
		return nil, errors.New("host: 启动缺少状态归属")
	}
	want := stagesAfter(store.Stage())
	if len(want) == 0 {
		return nil, fmt.Errorf("host: 启动阶段已全部完成，当前阶段 %s", store.Stage())
	}
	if len(steps) != len(want) {
		return nil, fmt.Errorf("%w: 需要 %d 个阶段（%v），实际 %d 个",
			ErrStageUnwired, len(want), want, len(steps))
	}
	for i, step := range steps {
		if step.Stage != want[i] {
			return nil, fmt.Errorf("%w: 第 %d 个阶段应为 %s，实际 %s",
				ErrStageUnwired, i+1, want[i], step.Stage)
		}
		if step.Run == nil {
			return nil, fmt.Errorf("%w: %s", ErrStageUnwired, step.Stage)
		}
	}
	return &Startup{store: store, steps: append([]StageStep(nil), steps...)}, nil
}

// Run 依次执行全部阶段。任一阶段失败即停在该阶段，不推进后续阶段。
func (s *Startup) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	if s.ran {
		s.mu.Unlock()
		return errors.New("host: 启动编排只能运行一次")
	}
	s.ran = true
	s.mu.Unlock()

	for _, step := range s.steps {
		if err := ctx.Err(); err != nil {
			s.store.Fail(step.Stage, err)
			return fmt.Errorf("host: 启动在阶段 %s 被取消: %w", step.Stage, err)
		}
		if err := step.Run(ctx); err != nil {
			s.store.Fail(step.Stage, err)
			return fmt.Errorf("host: 启动阶段 %s 失败: %w", step.Stage, err)
		}
		if err := s.store.Advance(step.Stage); err != nil {
			return fmt.Errorf("host: 启动阶段 %s 无法推进: %w", step.Stage, err)
		}
	}
	if err := s.store.Advance(StageReady); err != nil {
		return fmt.Errorf("host: 启动无法进入就绪: %w", err)
	}
	return nil
}
