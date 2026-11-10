//go:build !windows

package daemon

import (
	"os/exec"
	"syscall"
)

// configureDetached 让自启实例独立于客户端进程组：客户端崩溃时服务端可以
// 继续存在，由下一个客户端发现并复用；退出回收走显式关闭，不靠信号连带。
func configureDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
