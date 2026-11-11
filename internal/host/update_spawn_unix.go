//go:build !windows

package host

import (
	"os/exec"
	"syscall"
)

// configureDetached 让新进程脱离当前会话，父进程退出后继续运行。
func configureDetached(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
