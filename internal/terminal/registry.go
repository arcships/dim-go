package terminal

import (
	"errors"
	"fmt"
	"sync"
)

// Registry 按会话登记终端实例，保证一个会话同一时刻只有一个运行中的终端。
//
// 约定：重复启动同一会话被明确拒绝（ErrAlreadyRunning），不返回已有实例——
// 归属只有一个调用方，避免两处共享同一个设备句柄。上一实例已退出时，
// 新启动会先回收它的残留进程再替换。
//
// 已停止的实例保留在登记表里，供调用方读取退出状态；不需要时用 Forget 移除。
type Registry struct {
	mu   sync.Mutex
	byID map[string]*Terminal
}

// NewRegistry 创建空的终端登记表。
func NewRegistry() *Registry {
	return &Registry{byID: make(map[string]*Terminal)}
}

// Start 为会话启动终端。同一会话已有存活终端时返回 ErrAlreadyRunning。
func (r *Registry) Start(sessionID, dir string, opts Options) (*Terminal, error) {
	if r == nil {
		return nil, ErrNoTerminal
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if current, ok := r.byID[sessionID]; ok {
		if current.Alive() {
			return nil, fmt.Errorf("%w: %s", ErrAlreadyRunning, sessionID)
		}
		// 旧实例已退出：先清掉它留下的进程，再替换。
		_ = current.Stop()
	}
	t, err := Start(sessionID, dir, opts)
	if err != nil {
		return nil, err
	}
	r.byID[sessionID] = t
	return t, nil
}

// Get 返回该会话的终端实例（可能是已停止的实例）。
func (r *Registry) Get(sessionID string) (*Terminal, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byID[sessionID]
	return t, ok
}

// Stop 停止该会话的终端；没有实例时返回 ErrNoTerminal。
func (r *Registry) Stop(sessionID string) error {
	if r == nil {
		return ErrNoTerminal
	}
	r.mu.Lock()
	t, ok := r.byID[sessionID]
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrNoTerminal, sessionID)
	}
	return t.Stop()
}

// StopAll 停止全部终端，返回合并后的错误；用于进程退出时的收尾。
func (r *Registry) StopAll() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	all := make([]*Terminal, 0, len(r.byID))
	for _, t := range r.byID {
		all = append(all, t)
	}
	r.mu.Unlock()

	var errs []error
	for _, t := range all {
		if err := t.Stop(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Forget 移除某会话的登记项；终端仍在运行时先停止它。
func (r *Registry) Forget(sessionID string) error {
	if r == nil {
		return ErrNoTerminal
	}
	r.mu.Lock()
	t, ok := r.byID[sessionID]
	delete(r.byID, sessionID)
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrNoTerminal, sessionID)
	}
	return t.Stop()
}

// Len 返回登记表中的实例数（含已停止的实例）。
func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byID)
}
