package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/arcships/dimagent-mygo/internal/appserver/transport"
)

// SpawnRequest 是自启所需的本机端点信息。
type SpawnRequest struct {
	// NativeHome 是 App Server 的 `--home`。
	NativeHome string
	Endpoint   transport.Endpoint
}

// Child 是自启服务进程的句柄。只有自己拉起的实例才有句柄，回收也只对它生效。
type Child interface {
	PID() int
	Exited() <-chan struct{}
	Kill() error
}

// processChild 是 exec.Cmd 的句柄包装。
type processChild struct {
	cmd  *exec.Cmd
	done chan struct{}

	mu      sync.Mutex
	waitErr error
}

func (p *processChild) PID() int {
	if p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

func (p *processChild) Exited() <-chan struct{} { return p.done }

func (p *processChild) Kill() error {
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

func (p *processChild) WaitError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waitErr
}

// spawnDaemon 拉起自己拥有的服务实例。
//
// 约束：不写 daemon.log，也不读它；子进程的标准输入输出全部丢弃。需要诊断时
// 由上层诊断记录说明「自启了什么、端点是什么」，不复制服务端日志。
func spawnDaemon(cfg Config, request SpawnRequest) (Child, error) {
	if cfg.ServerPath == "" {
		return nil, errors.New("缺少 App Server 可执行文件路径（DIMCODE_APP_SERVER_BIN 或显式配置）")
	}
	info, err := os.Stat(cfg.ServerPath)
	if err != nil {
		return nil, fmt.Errorf("App Server 可执行文件不可用: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("App Server 可执行文件是目录: %s", cfg.ServerPath)
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("打开空设备: %w", err)
	}
	defer devNull.Close()

	cmd := exec.Command(cfg.ServerPath, buildDaemonArgs(cfg, endpoint{
		nativeHome: request.NativeHome,
		network:    request.Endpoint,
	})...)
	cmd.Env = os.Environ()
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	configureDetached(cmd)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 App Server: %w", err)
	}
	child := &processChild{cmd: cmd, done: make(chan struct{})}
	go func() {
		err := cmd.Wait()
		child.mu.Lock()
		child.waitErr = err
		child.mu.Unlock()
		close(child.done)
	}()
	return child, nil
}
